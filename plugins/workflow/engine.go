package workflow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"text/template"
	"time"

	"gorm.io/gorm"
)

// Engine 工作流引擎
type Engine struct {
	db          *gorm.DB
	executions  sync.Map // executionID -> *ExecutionContext
	notifier    func(channelID, title, content string) error
	aiAnalyzer  func(prompt string) string
}

type ExecutionContext struct {
	Execution *WorkflowExecution
	Workflow  *Workflow
	Variables map[string]interface{}
	Cancel    chan struct{}
}

type Node struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Name       string                 `json:"name"`
	Config     map[string]interface{} `json:"config"`
	Next       []string               `json:"next"`
	Conditions []Condition            `json:"conditions,omitempty"`
}

type Condition struct {
	Expression string `json:"expression"`
	Target     string `json:"target"`
}

func NewEngine(db *gorm.DB) *Engine {
	return &Engine{db: db}
}

func (e *Engine) SetNotifier(notifier func(string, string, string) error) {
	e.notifier = notifier
}

func (e *Engine) SetAIAnalyzer(analyzer func(string) string) {
	e.aiAnalyzer = analyzer
}

// Execute 执行工作流
func (e *Engine) Execute(workflow *Workflow, variables map[string]interface{}, triggerBy string) (*WorkflowExecution, error) {
	// 创建执行实例
	execution := &WorkflowExecution{
		WorkflowID:   workflow.ID,
		WorkflowName: workflow.Name,
		Status:       0,
		StartedAt:    time.Now(),
		Trigger:      workflow.Trigger,
		TriggerBy:    triggerBy,
	}

	varsJSON, _ := json.Marshal(variables)
	execution.Variables = string(varsJSON)

	if err := e.db.Create(execution).Error; err != nil {
		return nil, err
	}

	// 创建执行上下文
	ctx := &ExecutionContext{
		Execution: execution,
		Workflow:  workflow,
		Variables: variables,
		Cancel:    make(chan struct{}),
	}
	e.executions.Store(execution.ID, ctx)

	// 异步执行
	go e.run(ctx)

	return execution, nil
}

func (e *Engine) run(ctx *ExecutionContext) {
	defer func() {
		if r := recover(); r != nil {
			e.finishExecution(ctx, 2, fmt.Sprintf("panic: %v", r))
		}
		e.executions.Delete(ctx.Execution.ID)
	}()

	// 解析流程定义
	var definition struct {
		Nodes []Node `json:"nodes"`
	}
	if err := json.Unmarshal([]byte(ctx.Workflow.Definition), &definition); err != nil {
		e.finishExecution(ctx, 2, "解析流程定义失败: "+err.Error())
		return
	}

	// 构建节点映射
	nodeMap := make(map[string]*Node)
	var startNode *Node
	for i := range definition.Nodes {
		node := &definition.Nodes[i]
		nodeMap[node.ID] = node
		if node.Type == "start" {
			startNode = node
		}
	}

	if startNode == nil {
		e.finishExecution(ctx, 2, "未找到开始节点")
		return
	}

	// 从开始节点执行
	if err := e.executeNode(ctx, startNode, nodeMap); err != nil {
		e.finishExecution(ctx, 2, err.Error())
		return
	}

	e.finishExecution(ctx, 1, "")
}

func (e *Engine) executeNode(ctx *ExecutionContext, node *Node, nodeMap map[string]*Node) error {
	select {
	case <-ctx.Cancel:
		return fmt.Errorf("执行已取消")
	default:
	}

	// 更新当前节点
	e.db.Model(ctx.Execution).Update("current_node", node.ID)

	// 创建节点执行记录
	nodeExec := &WorkflowNodeExecution{
		ExecutionID: ctx.Execution.ID,
		NodeID:      node.ID,
		NodeName:    node.Name,
		NodeType:    node.Type,
		Status:      0,
		StartedAt:   time.Now(),
	}
	inputJSON, _ := json.Marshal(node.Config)
	nodeExec.Input = string(inputJSON)
	e.db.Create(nodeExec)

	// 执行节点
	var output interface{}
	var err error

	switch node.Type {
	case "start", "end":
		// 无操作
	case "shell":
		output, err = e.executeShell(ctx, node)
	case "http":
		output, err = e.executeHTTP(ctx, node)
	case "condition":
		// 条件节点在后面处理
	case "parallel":
		err = e.executeParallel(ctx, node, nodeMap)
	case "wait":
		err = e.executeWait(ctx, node)
	case "notify":
		err = e.executeNotify(ctx, node)
	case "ai":
		output, err = e.executeAI(ctx, node)
	case "approval":
		err = e.executeApproval(ctx, node)
	default:
		err = fmt.Errorf("未知节点类型: %s", node.Type)
	}

	// 更新节点执行记录
	now := time.Now()
	nodeExec.FinishedAt = &now
	nodeExec.Duration = int(now.Sub(nodeExec.StartedAt).Seconds())

	if err != nil {
		nodeExec.Status = 2
		nodeExec.Error = err.Error()
		e.db.Save(nodeExec)
		return err
	}

	nodeExec.Status = 1
	if output != nil {
		outputJSON, _ := json.Marshal(output)
		nodeExec.Output = string(outputJSON)
		// 保存输出到变量
		ctx.Variables[node.ID+"_output"] = output
	}
	e.db.Save(nodeExec)

	// 执行下一个节点
	if node.Type == "end" {
		return nil
	}

	nextNodeID := ""
	if node.Type == "condition" && len(node.Conditions) > 0 {
		// 条件判断
		for _, cond := range node.Conditions {
			if e.evaluateCondition(ctx, cond.Expression) {
				nextNodeID = cond.Target
				break
			}
		}
	} else if len(node.Next) > 0 {
		nextNodeID = node.Next[0]
	}

	if nextNodeID != "" {
		nextNode, ok := nodeMap[nextNodeID]
		if !ok {
			return fmt.Errorf("未找到节点: %s", nextNodeID)
		}
		return e.executeNode(ctx, nextNode, nodeMap)
	}

	return nil
}

func (e *Engine) executeShell(ctx *ExecutionContext, node *Node) (interface{}, error) {
	script, _ := node.Config["script"].(string)
	script = e.renderTemplate(script, ctx.Variables)

	timeout := 300
	if t, ok := node.Config["timeout"].(float64); ok {
		timeout = int(t)
	}

	cmd := exec.Command("sh", "-c", script)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	done := make(chan error)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		if err != nil {
			return nil, fmt.Errorf("执行失败: %s, stderr: %s", err.Error(), stderr.String())
		}
		return map[string]string{
			"stdout": stdout.String(),
			"stderr": stderr.String(),
		}, nil
	case <-time.After(time.Duration(timeout) * time.Second):
		cmd.Process.Kill()
		return nil, fmt.Errorf("执行超时")
	}
}

func (e *Engine) executeHTTP(ctx *ExecutionContext, node *Node) (interface{}, error) {
	method, _ := node.Config["method"].(string)
	url, _ := node.Config["url"].(string)
	body, _ := node.Config["body"].(string)

	url = e.renderTemplate(url, ctx.Variables)
	body = e.renderTemplate(body, ctx.Variables)

	var req *http.Request
	var err error

	if body != "" {
		req, err = http.NewRequest(method, url, strings.NewReader(body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		return nil, err
	}

	if headers, ok := node.Config["headers"].(map[string]interface{}); ok {
		for k, v := range headers {
			req.Header.Set(k, fmt.Sprintf("%v", v))
		}
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	return map[string]interface{}{
		"status_code": resp.StatusCode,
		"body":        string(respBody),
	}, nil
}

func (e *Engine) executeParallel(ctx *ExecutionContext, node *Node, nodeMap map[string]*Node) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(node.Next))

	for _, nextID := range node.Next {
		nextNode, ok := nodeMap[nextID]
		if !ok {
			continue
		}
		wg.Add(1)
		go func(n *Node) {
			defer wg.Done()
			if err := e.executeNode(ctx, n, nodeMap); err != nil {
				errChan <- err
			}
		}(nextNode)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) executeWait(ctx *ExecutionContext, node *Node) error {
	seconds := 10
	if s, ok := node.Config["seconds"].(float64); ok {
		seconds = int(s)
	}
	time.Sleep(time.Duration(seconds) * time.Second)
	return nil
}

func (e *Engine) executeNotify(ctx *ExecutionContext, node *Node) error {
	if e.notifier == nil {
		return nil
	}
	channelID, _ := node.Config["channel_id"].(string)
	title, _ := node.Config["title"].(string)
	content, _ := node.Config["content"].(string)

	title = e.renderTemplate(title, ctx.Variables)
	content = e.renderTemplate(content, ctx.Variables)

	return e.notifier(channelID, title, content)
}

func (e *Engine) executeAI(ctx *ExecutionContext, node *Node) (interface{}, error) {
	if e.aiAnalyzer == nil {
		return "AI未配置", nil
	}
	prompt, _ := node.Config["prompt"].(string)
	prompt = e.renderTemplate(prompt, ctx.Variables)
	result := e.aiAnalyzer(prompt)
	return result, nil
}

func (e *Engine) executeApproval(ctx *ExecutionContext, node *Node) error {
	// 更新执行状态为等待审批
	e.db.Model(ctx.Execution).Update("status", 4)
	// TODO: 发送审批通知，等待审批结果
	return nil
}

func (e *Engine) evaluateCondition(ctx *ExecutionContext, expression string) bool {
	// 简单实现：检查变量是否存在且为true
	expression = strings.TrimSpace(expression)
	if val, ok := ctx.Variables[expression]; ok {
		switch v := val.(type) {
		case bool:
			return v
		case string:
			return v != "" && v != "false" && v != "0"
		case float64:
			return v != 0
		}
	}
	return expression == "true" || expression == "default"
}

func (e *Engine) renderTemplate(text string, vars map[string]interface{}) string {
	tmpl, err := template.New("").Parse(text)
	if err != nil {
		return text
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return text
	}
	return buf.String()
}

func (e *Engine) finishExecution(ctx *ExecutionContext, status int, errMsg string) {
	now := time.Now()
	updates := map[string]interface{}{
		"status":      status,
		"finished_at": now,
		"duration":    int(now.Sub(ctx.Execution.StartedAt).Seconds()),
	}
	if errMsg != "" {
		updates["error"] = errMsg
	}
	e.db.Model(ctx.Execution).Updates(updates)
}

// Cancel 取消执行
func (e *Engine) Cancel(executionID string) error {
	if ctx, ok := e.executions.Load(executionID); ok {
		close(ctx.(*ExecutionContext).Cancel)
		e.db.Model(&WorkflowExecution{}).Where("id = ?", executionID).Update("status", 3)
		return nil
	}
	return fmt.Errorf("执行不存在或已结束")
}
