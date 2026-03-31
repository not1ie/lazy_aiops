package workorder

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lazyautoops/lazy-auto-ops/plugins/workflow"
	"gorm.io/gorm"
)

type aiExecutionPlanPayload struct {
	NeedApproval       bool                   `json:"need_approval"`
	Title              string                 `json:"title"`
	Objective          string                 `json:"objective"`
	Summary            string                 `json:"summary"`
	RiskLevel          string                 `json:"risk_level"`
	WorkOrderTypeCode  string                 `json:"workorder_type_code"`
	ApprovalReason     string                 `json:"approval_reason"`
	Prechecks          []string               `json:"prechecks"`
	Steps              []aiExecutionStepDraft `json:"steps"`
	RollbackSteps      []string               `json:"rollback_steps"`
	ValidationSteps    []string               `json:"validation_steps"`
	CreatedWorkOrderID string                 `json:"created_workorder_id"`
}

type aiExecutionStepDraft struct {
	Title                string `json:"title"`
	Action               string `json:"action"`
	Risk                 string `json:"risk"`
	NodeType             string `json:"node_type,omitempty"`
	CommandHint          string `json:"command_hint,omitempty"`
	Method               string `json:"method,omitempty"`
	URL                  string `json:"url,omitempty"`
	Body                 string `json:"body,omitempty"`
	RequiresConfirmation bool   `json:"requires_confirmation,omitempty"`
}

type aiPlanFormData struct {
	Source              string                  `json:"source"`
	SessionID           string                  `json:"session_id"`
	MessageID           string                  `json:"message_id"`
	ContextScope        string                  `json:"context_scope"`
	ContextSummary      string                  `json:"context_summary"`
	GeneratedWorkflowID string                  `json:"generated_workflow_id,omitempty"`
	Plan                *aiExecutionPlanPayload `json:"plan"`
}

type runbookNode struct {
	ID     string                 `json:"id"`
	Type   string                 `json:"type"`
	Name   string                 `json:"name"`
	Config map[string]interface{} `json:"config,omitempty"`
	Next   []string               `json:"next,omitempty"`
}

func parseAIPlanFormData(formData string) *aiPlanFormData {
	if strings.TrimSpace(formData) == "" {
		return nil
	}
	var payload aiPlanFormData
	if err := json.Unmarshal([]byte(formData), &payload); err != nil {
		return nil
	}
	if payload.Source != "ai_execution_plan" || payload.Plan == nil {
		return nil
	}
	return &payload
}

func manualConfirmationItems(payload *aiPlanFormData) []string {
	if payload == nil || payload.Plan == nil {
		return nil
	}
	items := make([]string, 0)
	for _, step := range payload.Plan.Steps {
		nodeType := strings.TrimSpace(strings.ToLower(step.NodeType))
		if step.RequiresConfirmation || nodeType == "manual" {
			title := strings.TrimSpace(step.Title)
			if title == "" {
				title = strings.TrimSpace(step.Action)
			}
			if title != "" {
				items = append(items, title)
			}
		}
	}
	return items
}

func buildRunbookExecutionVariables(order *WorkOrder, payload *aiPlanFormData) map[string]interface{} {
	vars := map[string]interface{}{
		"workorder_id":    order.ID,
		"workorder_title": order.Title,
		"workorder_type":  order.TypeName,
		"submitter":       order.Submitter,
	}
	if payload == nil || payload.Plan == nil {
		return vars
	}
	vars["plan_title"] = payload.Plan.Title
	vars["plan_summary"] = payload.Plan.Summary
	vars["plan_objective"] = payload.Plan.Objective
	vars["risk_level"] = payload.Plan.RiskLevel
	vars["approval_reason"] = payload.Plan.ApprovalReason
	vars["context_scope"] = payload.ContextScope
	vars["context_summary"] = payload.ContextSummary
	vars["manual_items"] = manualConfirmationItems(payload)
	vars["prechecks"] = payload.Plan.Prechecks
	vars["validation_steps"] = payload.Plan.ValidationSteps
	vars["rollback_steps"] = payload.Plan.RollbackSteps
	return vars
}

func maybeCreateRunbookWorkflow(db *gorm.DB, order *WorkOrder, operator string) (*workflow.Workflow, error) {
	if order == nil || strings.TrimSpace(order.FormData) == "" {
		return nil, nil
	}
	payload := parseAIPlanFormData(order.FormData)
	if payload == nil {
		return nil, nil
	}
	if !payload.Plan.NeedApproval {
		return nil, nil
	}
	if strings.TrimSpace(payload.GeneratedWorkflowID) != "" {
		var existing workflow.Workflow
		if err := db.First(&existing, "id = ?", payload.GeneratedWorkflowID).Error; err == nil {
			return &existing, nil
		}
	}

	definition := buildRunbookWorkflowDefinition(order, payload)
	variables := buildRunbookWorkflowVariables(order, payload)
	name := strings.TrimSpace(payload.Plan.Title)
	if name == "" {
		name = strings.TrimSpace(order.Title)
	}
	if name == "" {
		name = "AI Runbook"
	}
	wf := &workflow.Workflow{
		Name:        "[AI Runbook] " + truncateText(name, 80),
		Description: truncateText(payload.Plan.Summary, 512),
		Category:    "custom",
		Definition:  definition,
		Variables:   variables,
		Trigger:     "manual",
		Enabled:     false,
		CreatedBy:   strings.TrimSpace(operator),
	}
	if err := db.Create(wf).Error; err != nil {
		return nil, err
	}

	payload.GeneratedWorkflowID = wf.ID
	nextFormData, _ := json.Marshal(payload)
	if err := db.Model(order).Update("form_data", string(nextFormData)).Error; err != nil {
		return nil, err
	}
	addComment(db, order.ID, operator, "system", fmt.Sprintf("审批通过后已自动生成 Workflow Runbook 草案：%s (%s)，请在工作流中心补充或确认后执行。", wf.Name, wf.ID))
	return wf, nil
}

func buildRunbookWorkflowDefinition(order *WorkOrder, payload *aiPlanFormData) string {
	nodes := []runbookNode{
		{ID: "start", Type: "start", Name: "开始", Next: []string{"summary"}},
		{
			ID:   "summary",
			Type: "notify",
			Name: "执行摘要",
			Config: map[string]interface{}{
				"title":   fmt.Sprintf("AI Runbook - %s", payload.Plan.Title),
				"content": buildSummaryContent(order, payload),
			},
		},
	}

	prevID := "summary"
	stepCount := 0
	for index, step := range payload.Plan.Steps {
		stepCount++
		nodeID := fmt.Sprintf("step_%d", stepCount)
		nodes = append(nodes, buildPlanStepNode(nodeID, index+1, step))
		for i := range nodes {
			if nodes[i].ID == prevID {
				nodes[i].Next = []string{nodeID}
				break
			}
		}
		prevID = nodeID
	}

	if len(payload.Plan.ValidationSteps) > 0 {
		nodes = append(nodes, runbookNode{
			ID:   "validate",
			Type: "notify",
			Name: "验证步骤",
			Config: map[string]interface{}{
				"title":   "验证步骤",
				"content": strings.Join(payload.Plan.ValidationSteps, "\n"),
			},
		})
		for i := range nodes {
			if nodes[i].ID == prevID {
				nodes[i].Next = []string{"validate"}
				break
			}
		}
		prevID = "validate"
	}

	if len(payload.Plan.RollbackSteps) > 0 {
		nodes = append(nodes, runbookNode{
			ID:   "rollback",
			Type: "notify",
			Name: "回滚预案",
			Config: map[string]interface{}{
				"title":   "回滚预案",
				"content": strings.Join(payload.Plan.RollbackSteps, "\n"),
			},
		})
		for i := range nodes {
			if nodes[i].ID == prevID {
				nodes[i].Next = []string{"rollback"}
				break
			}
		}
		prevID = "rollback"
	}

	nodes = append(nodes, runbookNode{ID: "end", Type: "end", Name: "结束"})
	for i := range nodes {
		if nodes[i].ID == prevID {
			nodes[i].Next = []string{"end"}
			break
		}
	}

	return marshalPrettyJSON(map[string]interface{}{"nodes": nodes})
}

func buildRunbookWorkflowVariables(order *WorkOrder, payload *aiPlanFormData) string {
	return marshalPrettyJSON(map[string]interface{}{
		"workorder_id":    order.ID,
		"workorder_title": order.Title,
		"plan_title":      payload.Plan.Title,
		"plan_objective":  payload.Plan.Objective,
		"plan_summary":    payload.Plan.Summary,
		"risk_level":      payload.Plan.RiskLevel,
		"context_scope":   payload.ContextScope,
		"context_summary": payload.ContextSummary,
		"approval_reason": payload.Plan.ApprovalReason,
	})
}

func buildPlanStepNode(nodeID string, index int, step aiExecutionStepDraft) runbookNode {
	title := strings.TrimSpace(step.Title)
	if title == "" {
		title = fmt.Sprintf("步骤 %d", index)
	}
	switch inferRunbookNodeType(step) {
	case "http":
		return runbookNode{
			ID:   nodeID,
			Type: "http",
			Name: title,
			Config: map[string]interface{}{
				"method": normalizeHTTPMethod(step.Method),
				"url":    strings.TrimSpace(step.URL),
				"body":   strings.TrimSpace(step.Body),
			},
		}
	case "shell":
		return runbookNode{
			ID:   nodeID,
			Type: "shell",
			Name: title,
			Config: map[string]interface{}{
				"script":  buildShellScript(step),
				"timeout": 600,
			},
		}
	case "notify", "manual":
		return buildManualStepNode(nodeID, index, title, step)
	}
	return buildManualStepNode(nodeID, index, title, step)
}

func buildManualStepNode(nodeID string, index int, title string, step aiExecutionStepDraft) runbookNode {
	return runbookNode{
		ID:   nodeID,
		Type: "notify",
		Name: title,
		Config: map[string]interface{}{
			"title": fmt.Sprintf("步骤 %d: %s", index, title),
			"content": strings.TrimSpace(strings.Join([]string{
				"动作: " + strings.TrimSpace(step.Action),
				formatOptionalLine("风险", step.Risk),
				formatOptionalLine("HTTP", joinHTTPHint(step)),
				formatOptionalLine("命令提示", step.CommandHint),
				func() string {
					if step.RequiresConfirmation {
						return "说明: 该步骤需要人工确认后继续，请在工作流设计器中补充审批或手工处理节点。"
					}
					return "说明: 该步骤暂无可执行命令，请在工作流设计器中补充为 shell/http/approval 节点。"
				}(),
			}, "\n")),
		},
	}
}

func buildShellScript(step aiExecutionStepDraft) string {
	lines := []string{
		"set -e",
		"# AI Runbook step generated from approval plan.",
		"# Please review and adjust before running in production.",
	}
	if action := strings.TrimSpace(step.Action); action != "" {
		lines = append(lines, "# Action: "+action)
	}
	if risk := strings.TrimSpace(step.Risk); risk != "" {
		lines = append(lines, "# Risk: "+risk)
	}
	lines = append(lines, strings.TrimSpace(step.CommandHint))
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func inferRunbookNodeType(step aiExecutionStepDraft) string {
	nodeType := strings.TrimSpace(strings.ToLower(step.NodeType))
	switch nodeType {
	case "shell", "http", "notify", "manual":
		return nodeType
	}
	if strings.TrimSpace(step.URL) != "" || strings.TrimSpace(step.Method) != "" {
		return "http"
	}
	if strings.TrimSpace(step.CommandHint) != "" {
		return "shell"
	}
	if step.RequiresConfirmation {
		return "manual"
	}
	action := strings.ToLower(strings.TrimSpace(step.Action))
	if strings.Contains(action, "接口") || strings.Contains(action, "api") || strings.Contains(action, "webhook") || strings.Contains(action, "请求") {
		return "http"
	}
	return "manual"
}

func normalizeHTTPMethod(method string) string {
	raw := strings.TrimSpace(strings.ToUpper(method))
	switch raw {
	case "GET", "POST", "PUT", "DELETE", "PATCH":
		return raw
	default:
		return "POST"
	}
}

func joinHTTPHint(step aiExecutionStepDraft) string {
	url := strings.TrimSpace(step.URL)
	method := normalizeHTTPMethod(step.Method)
	if url == "" {
		return ""
	}
	if body := strings.TrimSpace(step.Body); body != "" {
		return fmt.Sprintf("%s %s\nBody: %s", method, url, body)
	}
	return fmt.Sprintf("%s %s", method, url)
}

func marshalPrettyJSON(value interface{}) string {
	data, _ := json.MarshalIndent(value, "", "  ")
	return string(data)
}

func buildSummaryContent(order *WorkOrder, payload *aiPlanFormData) string {
	lines := []string{
		"工单: " + order.Title,
		formatOptionalLine("目标", payload.Plan.Objective),
		formatOptionalLine("摘要", payload.Plan.Summary),
		formatOptionalLine("风险等级", payload.Plan.RiskLevel),
		formatOptionalLine("审批原因", payload.Plan.ApprovalReason),
		formatOptionalLine("上下文", payload.ContextSummary),
	}
	if len(payload.Plan.Prechecks) > 0 {
		lines = append(lines, "前置检查:\n"+strings.Join(payload.Plan.Prechecks, "\n"))
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func formatOptionalLine(label, value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return label + ": " + strings.TrimSpace(value)
}

func truncateText(text string, max int) string {
	raw := strings.TrimSpace(text)
	if len(raw) <= max {
		return raw
	}
	return raw[:max] + "..."
}
