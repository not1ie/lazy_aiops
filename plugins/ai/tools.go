package ai

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/lazyautoops/lazy-auto-ops/plugins/alert"
	"github.com/lazyautoops/lazy-auto-ops/plugins/cmdb"
	"github.com/lazyautoops/lazy-auto-ops/plugins/docker"
	"github.com/lazyautoops/lazy-auto-ops/plugins/k8s"
	"github.com/lazyautoops/lazy-auto-ops/plugins/monitor"
	"github.com/lazyautoops/lazy-auto-ops/plugins/workflow"
	"github.com/lazyautoops/lazy-auto-ops/plugins/workorder"
)

type aiToolDefinition struct {
	Name        string
	Description string
	Scopes      []string
	Run         func(args map[string]string) (string, error)
}

func (s *AIService) availableToolsForScope(scope string) []aiToolDefinition {
	defs := []aiToolDefinition{
		{
			Name:        "get_asset_overview",
			Description: "查询主机、分组、Docker 主机的资产总览统计。适合主机、资产、终端排障。",
			Scopes:      []string{"asset", "general"},
			Run:         s.toolGetAssetOverview,
		},
		{
			Name:        "search_hosts",
			Description: "按主机名、IP、标签搜索主机。参数: query。",
			Scopes:      []string{"asset", "general"},
			Run:         s.toolSearchHosts,
		},
		{
			Name:        "get_k8s_overview",
			Description: "查询 K8s 集群和 Docker 环境总览。适合集群、Pod、容器异常定位。",
			Scopes:      []string{"k8s", "general"},
			Run:         s.toolGetK8sOverview,
		},
		{
			Name:        "get_open_alerts",
			Description: "查询当前未恢复或未确认告警。参数可选: severity, limit。",
			Scopes:      []string{"monitor", "k8s", "delivery", "general"},
			Run:         s.toolGetOpenAlerts,
		},
		{
			Name:        "get_agent_status",
			Description: "查询 Agent 在线率和最近离线/异常节点。",
			Scopes:      []string{"monitor", "asset", "general"},
			Run:         s.toolGetAgentStatus,
		},
		{
			Name:        "get_delivery_overview",
			Description: "查询工单、流程、审批总览。",
			Scopes:      []string{"delivery", "general"},
			Run:         s.toolGetDeliveryOverview,
		},
		{
			Name:        "get_recent_workorders",
			Description: "查询最近工单。参数可选: status, limit。",
			Scopes:      []string{"delivery", "general"},
			Run:         s.toolGetRecentWorkorders,
		},
		{
			Name:        "get_recent_workflow_runs",
			Description: "查询最近流程执行。参数可选: status, limit。",
			Scopes:      []string{"delivery", "general"},
			Run:         s.toolGetRecentWorkflowRuns,
		},
	}

	allowed := make([]aiToolDefinition, 0)
	for _, item := range defs {
		if containsString(item.Scopes, scope) || containsString(item.Scopes, "general") {
			allowed = append(allowed, item)
		}
	}
	return allowed
}

func (s *AIService) maybeUseTools(req *ChatRequest, pack *AIContextPack, history []ChatMessage) ([]AIToolTrace, string) {
	scope := "general"
	if pack != nil && strings.TrimSpace(pack.Scope) != "" {
		scope = strings.TrimSpace(pack.Scope)
	}
	tools := s.availableToolsForScope(scope)
	if len(tools) == 0 {
		return nil, ""
	}

	plan, err := s.planToolCalls(req, pack, history, tools)
	if err != nil || plan == nil || !plan.UseTools || len(plan.ToolCalls) == 0 {
		return nil, ""
	}

	toolMap := map[string]aiToolDefinition{}
	for _, item := range tools {
		toolMap[item.Name] = item
	}

	traces := make([]AIToolTrace, 0, len(plan.ToolCalls))
	resultLines := make([]string, 0, len(plan.ToolCalls))
	for idx, call := range plan.ToolCalls {
		if idx >= 3 {
			break
		}
		def, ok := toolMap[strings.TrimSpace(call.Name)]
		if !ok {
			traces = append(traces, AIToolTrace{
				Name:      call.Name,
				Reason:    call.Reason,
				Arguments: call.Arguments,
				Status:    "skipped",
				Summary:   "工具未注册或当前场景不可用",
			})
			continue
		}
		summary, runErr := def.Run(call.Arguments)
		trace := AIToolTrace{
			Name:      def.Name,
			Reason:    call.Reason,
			Arguments: call.Arguments,
			Status:    "success",
			Summary:   summary,
		}
		if runErr != nil {
			trace.Status = "error"
			trace.Summary = runErr.Error()
		}
		traces = append(traces, trace)
		resultLines = append(resultLines, fmt.Sprintf("[%s] %s", trace.Name, trace.Summary))
	}

	return traces, strings.Join(resultLines, "\n")
}

func (s *AIService) planToolCalls(req *ChatRequest, pack *AIContextPack, history []ChatMessage, tools []aiToolDefinition) (*AIToolPlan, error) {
	toolDescriptions := make([]string, 0, len(tools))
	for _, item := range tools {
		toolDescriptions = append(toolDescriptions, fmt.Sprintf("- %s: %s", item.Name, item.Description))
	}

	historyLines := make([]string, 0, minInt(len(history), 6))
	for _, item := range tailMessages(history, 6) {
		historyLines = append(historyLines, fmt.Sprintf("%s: %s", item.Role, truncate(strings.TrimSpace(item.Content), 120)))
	}

	prompt := strings.Join([]string{
		"你是运维 AI 的只读工具规划器。",
		"请根据用户问题和当前场景，决定是否需要调用只读工具来补充证据。",
		"不要规划任何写操作、变更操作或危险动作。",
		"最多调用 3 个工具；如果现有上下文已足够，请返回 use_tools=false。",
		"必须严格返回 JSON，不要输出 Markdown 代码块。",
		fmt.Sprintf("当前用户问题: %s", strings.TrimSpace(req.Message)),
		fmt.Sprintf("当前场景摘要: %s", func() string {
			if pack == nil {
				return "无"
			}
			return pack.Summary
		}()),
		"最近对话:",
		strings.Join(historyLines, "\n"),
		"可用工具:",
		strings.Join(toolDescriptions, "\n"),
		`返回格式:
{"use_tools":true,"focus":"一句话说明关注点","tool_calls":[{"name":"工具名","reason":"调用原因","arguments":{"query":"可选参数"}}]}`,
	}, "\n")

	reply, _, err := s.core.AI.CallLLM("你是一个严格输出 JSON 的工具规划器。", []map[string]string{
		{"role": "user", "content": prompt},
	})
	if err != nil {
		return nil, err
	}

	reply = extractJSONObject(reply)
	if reply == "" {
		return nil, nil
	}

	var plan AIToolPlan
	if err := json.Unmarshal([]byte(reply), &plan); err != nil {
		return nil, nil
	}
	if len(plan.ToolCalls) > 3 {
		plan.ToolCalls = plan.ToolCalls[:3]
	}
	return &plan, nil
}

func (s *AIService) toolGetAssetOverview(args map[string]string) (string, error) {
	var totalHosts, onlineHosts, maintenanceHosts, dockerHosts int64
	if err := s.db.Model(&cmdb.Host{}).Count(&totalHosts).Error; err != nil {
		return "", err
	}
	if err := s.db.Model(&cmdb.Host{}).Where("status = ?", 1).Count(&onlineHosts).Error; err != nil {
		return "", err
	}
	if err := s.db.Model(&cmdb.Host{}).Where("status = ?", 2).Count(&maintenanceHosts).Error; err != nil {
		return "", err
	}
	if err := s.db.Model(&docker.DockerHost{}).Where("status = ?", "online").Count(&dockerHosts).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("资产总览: 主机 %d 台，在线 %d 台，维护中 %d 台，在线 Docker 主机 %d 台。", totalHosts, onlineHosts, maintenanceHosts, dockerHosts), nil
}

func (s *AIService) toolSearchHosts(args map[string]string) (string, error) {
	query := strings.TrimSpace(args["query"])
	rows := make([]cmdb.Host, 0)
	db := s.db.Model(&cmdb.Host{}).Order("updated_at desc").Limit(5)
	if query != "" {
		like := "%" + query + "%"
		db = db.Where("name LIKE ? OR ip LIKE ? OR tags LIKE ?", like, like, like)
	}
	if err := db.Find(&rows).Error; err != nil {
		return "", err
	}
	if len(rows) == 0 {
		if query == "" {
			return "没有查到最近主机记录。", nil
		}
		return fmt.Sprintf("没有找到匹配“%s”的主机。", query), nil
	}
	items := make([]string, 0, len(rows))
	for _, item := range rows {
		items = append(items, fmt.Sprintf("%s(%s, status=%d, os=%s)", item.Name, item.IP, item.Status, item.OS))
	}
	return "主机结果: " + strings.Join(items, "；"), nil
}

func (s *AIService) toolGetK8sOverview(args map[string]string) (string, error) {
	var totalClusters, healthyClusters, abnormalClusters, dockerHosts int64
	if err := s.db.Model(&k8s.Cluster{}).Count(&totalClusters).Error; err != nil {
		return "", err
	}
	if err := s.db.Model(&k8s.Cluster{}).Where("status = ?", 1).Count(&healthyClusters).Error; err != nil {
		return "", err
	}
	if err := s.db.Model(&k8s.Cluster{}).Where("status <> ?", 1).Count(&abnormalClusters).Error; err != nil {
		return "", err
	}
	if err := s.db.Model(&docker.DockerHost{}).Where("status = ?", "online").Count(&dockerHosts).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("容器总览: K8s 集群 %d 个，健康 %d 个，异常/维护 %d 个，在线 Docker 环境 %d 个。", totalClusters, healthyClusters, abnormalClusters, dockerHosts), nil
}

func (s *AIService) toolGetOpenAlerts(args map[string]string) (string, error) {
	limit := parseLimitArg(args["limit"], 5)
	query := s.db.Model(&alert.Alert{}).Where("status in ?", []int{0, 1})
	severity := strings.TrimSpace(args["severity"])
	if severity != "" {
		query = query.Where("severity = ?", severity)
	}
	var rows []alert.Alert
	if err := query.Order("fired_at desc").Limit(limit).Find(&rows).Error; err != nil {
		return "", err
	}
	if len(rows) == 0 {
		if severity != "" {
			return fmt.Sprintf("当前没有 severity=%s 的未恢复告警。", severity), nil
		}
		return "当前没有未恢复或未确认的告警。", nil
	}
	items := make([]string, 0, len(rows))
	for _, item := range rows {
		items = append(items, fmt.Sprintf("%s/%s(%s=%s, count=%d)", item.RuleName, item.Target, item.Metric, item.Value, item.Count))
	}
	return "当前告警: " + strings.Join(items, "；"), nil
}

func (s *AIService) toolGetAgentStatus(args map[string]string) (string, error) {
	var total, online int64
	if err := s.db.Model(&monitor.AgentHeartbeat{}).Count(&total).Error; err != nil {
		return "", err
	}
	if err := s.db.Model(&monitor.AgentHeartbeat{}).Where("status = ?", "online").Count(&online).Error; err != nil {
		return "", err
	}
	var rows []monitor.AgentHeartbeat
	if err := s.db.Where("status <> ?", "online").Order("last_seen asc").Limit(3).Find(&rows).Error; err != nil {
		return "", err
	}
	items := make([]string, 0, len(rows))
	for _, item := range rows {
		items = append(items, fmt.Sprintf("%s/%s(%s)", item.Hostname, item.IP, item.Status))
	}
	summary := fmt.Sprintf("Agent 状态: 在线 %d/%d。", online, total)
	if len(items) > 0 {
		summary += " 最近异常 Agent: " + strings.Join(items, "；")
	}
	return summary, nil
}

func (s *AIService) toolGetDeliveryOverview(args map[string]string) (string, error) {
	var pendingOrders, runningOrders, waitingApproval, runningExecutions int64
	if err := s.db.Model(&workorder.WorkOrder{}).Where("status in ?", []int{0, 1}).Count(&pendingOrders).Error; err != nil {
		return "", err
	}
	if err := s.db.Model(&workorder.WorkOrder{}).Where("status = ?", 4).Count(&runningOrders).Error; err != nil {
		return "", err
	}
	if err := s.db.Model(&workflow.WorkflowExecution{}).Where("status = ?", 4).Count(&waitingApproval).Error; err != nil {
		return "", err
	}
	if err := s.db.Model(&workflow.WorkflowExecution{}).Where("status = ?", 0).Count(&runningExecutions).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("交付总览: 待处理工单 %d 个，执行中工单 %d 个，运行中流程 %d 个，等待审批流程 %d 个。", pendingOrders, runningOrders, runningExecutions, waitingApproval), nil
}

func (s *AIService) toolGetRecentWorkorders(args map[string]string) (string, error) {
	limit := parseLimitArg(args["limit"], 5)
	query := s.db.Model(&workorder.WorkOrder{}).Order("updated_at desc").Limit(limit)
	if status := strings.TrimSpace(args["status"]); status != "" {
		if statusNum, err := strconv.Atoi(status); err == nil {
			query = query.Where("status = ?", statusNum)
		}
	}
	var rows []workorder.WorkOrder
	if err := query.Find(&rows).Error; err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return "最近没有匹配条件的工单。", nil
	}
	items := make([]string, 0, len(rows))
	for _, item := range rows {
		items = append(items, fmt.Sprintf("%s(status=%d, priority=%d)", item.Title, item.Status, item.Priority))
	}
	return "最近工单: " + strings.Join(items, "；"), nil
}

func (s *AIService) toolGetRecentWorkflowRuns(args map[string]string) (string, error) {
	limit := parseLimitArg(args["limit"], 5)
	query := s.db.Model(&workflow.WorkflowExecution{}).Order("started_at desc").Limit(limit)
	if status := strings.TrimSpace(args["status"]); status != "" {
		if statusNum, err := strconv.Atoi(status); err == nil {
			query = query.Where("status = ?", statusNum)
		}
	}
	var rows []workflow.WorkflowExecution
	if err := query.Find(&rows).Error; err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return "最近没有匹配条件的流程执行。", nil
	}
	items := make([]string, 0, len(rows))
	for _, item := range rows {
		items = append(items, fmt.Sprintf("%s(status=%d, trigger=%s, by=%s)", item.WorkflowName, item.Status, item.Trigger, item.TriggerBy))
	}
	return "最近流程执行: " + strings.Join(items, "；"), nil
}

func extractJSONObject(input string) string {
	text := strings.TrimSpace(input)
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start >= 0 && end > start {
		return text[start : end+1]
	}
	return ""
}

func parseLimitArg(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return fallback
	}
	if value > 10 {
		return 10
	}
	return value
}

func tailMessages(history []ChatMessage, n int) []ChatMessage {
	if len(history) <= n {
		return history
	}
	return history[len(history)-n:]
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if strings.TrimSpace(item) == strings.TrimSpace(target) {
			return true
		}
	}
	return false
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func formatToolTraces(traces []AIToolTrace) string {
	if len(traces) == 0 {
		return ""
	}
	lines := make([]string, 0, len(traces)+1)
	lines = append(lines, "系统已经调用的只读工具结果:")
	for _, item := range traces {
		if item.Status != "success" {
			lines = append(lines, fmt.Sprintf("- %s(%s): %s", item.Name, item.Status, item.Summary))
			continue
		}
		lines = append(lines, fmt.Sprintf("- %s: %s", item.Name, item.Summary))
	}
	return strings.Join(lines, "\n")
}

func marshalMessageMeta(meta ChatMessageMeta) string {
	data, _ := json.Marshal(meta)
	return string(data)
}
