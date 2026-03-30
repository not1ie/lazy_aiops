package ai

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lazyautoops/lazy-auto-ops/plugins/workorder"
)

func (s *AIService) buildExecutionPlan(req *ChatRequest, pack *AIContextPack, toolTraces []AIToolTrace, reply string) (*AIExecutionPlan, error) {
	prompt := strings.Join([]string{
		"你是运维变更计划规划器。",
		"请根据用户问题、当前场景、只读证据和最终回答，判断是否需要生成审批前执行计划。",
		"如果只是解释性问答、纯知识咨询、无需变更，则 need_approval=false。",
		"如果需要任何重启、变更配置、扩缩容、回滚、发布、执行脚本、数据库操作或潜在影响业务的动作，则 need_approval=true。",
		"风险等级只能是 low/medium/high。",
		"workorder_type_code 优先使用 incident、change_apply、routine 之一。",
		"steps 中每一步尽量给出 node_type，候选只能是 shell/http/manual/notify。",
		"如果是 shell，尽量提供 command_hint；如果是 http，尽量提供 method、url、body；如果必须人工确认或补充信息，使用 manual 并把 requires_confirmation=true。",
		"必须严格输出 JSON，不要输出 Markdown 代码块。",
		fmt.Sprintf("用户问题: %s", strings.TrimSpace(req.Message)),
		fmt.Sprintf("场景摘要: %s", func() string {
			if pack == nil {
				return "无"
			}
			return pack.Summary
		}()),
		fmt.Sprintf("工具证据: %s", formatToolTraces(toolTraces)),
		fmt.Sprintf("最终回答: %s", strings.TrimSpace(reply)),
		`返回格式:
{"need_approval":true,"title":"审批标题","objective":"目标","summary":"计划摘要","risk_level":"medium","workorder_type_code":"change_apply","approval_reason":"为什么要审批","prechecks":["检查项"],"steps":[{"title":"步骤标题","action":"要做什么","risk":"风险说明","node_type":"shell","command_hint":"命令提示","method":"","url":"","body":"","requires_confirmation":false}],"rollback_steps":["回滚步骤"],"validation_steps":["验证步骤"]}`,
	}, "\n")

	result, _, err := s.core.AI.CallLLM("你是一个严格输出 JSON 的审批计划规划器。", []map[string]string{
		{"role": "user", "content": prompt},
	})
	if err != nil {
		return nil, err
	}
	result = extractJSONObject(result)
	if result == "" {
		return nil, nil
	}

	var plan AIExecutionPlan
	if err := json.Unmarshal([]byte(result), &plan); err != nil {
		return nil, nil
	}
	if !plan.NeedApproval {
		return nil, nil
	}
	plan.Title = strings.TrimSpace(plan.Title)
	if plan.Title == "" {
		plan.Title = truncate(req.Message, 60)
	}
	if strings.TrimSpace(plan.WorkOrderTypeCode) == "" {
		plan.WorkOrderTypeCode = inferWorkOrderTypeCode(pack)
	}
	if strings.TrimSpace(plan.RiskLevel) == "" {
		plan.RiskLevel = "medium"
	}
	for index := range plan.Steps {
		plan.Steps[index].NodeType = normalizePlanStepNodeType(plan.Steps[index])
	}
	return &plan, nil
}

func inferWorkOrderTypeCode(pack *AIContextPack) string {
	if pack == nil {
		return "change_apply"
	}
	switch pack.Scope {
	case "monitor", "asset", "k8s":
		return "incident"
	case "delivery":
		return "change_apply"
	default:
		return "routine"
	}
}

func (s *AIService) CreateWorkOrderFromMessage(messageID, userID, username string) (*workorder.WorkOrder, *AIExecutionPlan, error) {
	var message ChatMessage
	if err := s.db.First(&message, "id = ?", messageID).Error; err != nil {
		return nil, nil, err
	}
	if message.Role != "assistant" {
		return nil, nil, fmt.Errorf("仅支持从助手消息创建审批工单")
	}

	var session ChatSession
	if err := s.db.First(&session, "id = ?", message.SessionID).Error; err != nil {
		return nil, nil, err
	}
	if session.UserID != userID {
		return nil, nil, fmt.Errorf("无权访问该消息")
	}

	var meta ChatMessageMeta
	if strings.TrimSpace(message.Meta) != "" {
		_ = json.Unmarshal([]byte(message.Meta), &meta)
	}
	if meta.ExecutionPlan == nil || !meta.ExecutionPlan.NeedApproval {
		return nil, nil, fmt.Errorf("当前消息没有可审批的执行计划")
	}
	if meta.ExecutionPlan.CreatedWorkOrderID != "" {
		var existing workorder.WorkOrder
		if err := s.db.First(&existing, "id = ?", meta.ExecutionPlan.CreatedWorkOrderID).Error; err == nil {
			return &existing, meta.ExecutionPlan, nil
		}
	}

	formData, _ := json.Marshal(map[string]interface{}{
		"source":          "ai_execution_plan",
		"session_id":      session.ID,
		"message_id":      message.ID,
		"context_scope":   meta.ContextScope,
		"context_summary": meta.ContextSummary,
		"plan":            meta.ExecutionPlan,
	})

	order, err := workorder.CreateOrderWithDefaults(s.db, workorder.CreateOrderInput{
		TypeCode:     meta.ExecutionPlan.WorkOrderTypeCode,
		Title:        meta.ExecutionPlan.Title,
		Content:      meta.ExecutionPlan.Summary + "\n\n审批原因: " + meta.ExecutionPlan.ApprovalReason,
		FormData:     string(formData),
		Priority:     priorityFromRisk(meta.ExecutionPlan.RiskLevel),
		Submitter:    username,
		SubmitterID:  userID,
		AISuggestion: message.Content,
		AIRisk:       meta.ExecutionPlan.RiskLevel,
	})
	if err != nil {
		return nil, nil, err
	}

	meta.ExecutionPlan.CreatedWorkOrderID = order.ID
	message.Meta = marshalMessageMeta(meta)
	if err := s.db.Save(&message).Error; err != nil {
		return nil, nil, err
	}
	return order, meta.ExecutionPlan, nil
}

func priorityFromRisk(risk string) int {
	switch strings.TrimSpace(strings.ToLower(risk)) {
	case "high":
		return 1
	case "medium":
		return 2
	case "low":
		return 3
	default:
		return 2
	}
}

func normalizePlanStepNodeType(step AIExecutionStep) string {
	nodeType := strings.TrimSpace(strings.ToLower(step.NodeType))
	switch nodeType {
	case "shell", "http", "manual", "notify":
		return nodeType
	}
	if strings.TrimSpace(step.URL) != "" || strings.TrimSpace(step.Method) != "" {
		return "http"
	}
	if strings.TrimSpace(step.CommandHint) != "" {
		return "shell"
	}
	action := strings.ToLower(strings.TrimSpace(step.Action))
	if strings.Contains(action, "接口") || strings.Contains(action, "webhook") || strings.Contains(action, "api") || strings.Contains(action, "请求") {
		return "http"
	}
	if step.RequiresConfirmation {
		return "manual"
	}
	return "manual"
}
