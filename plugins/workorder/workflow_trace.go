package workorder

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/lazyautoops/lazy-auto-ops/plugins/workflow"
	"gorm.io/gorm"
)

var workflowTraceRegistry sync.Map

func startWorkflowExecutionTrace(db *gorm.DB, orderID, workflowID, executionID, operator string) {
	if db == nil || strings.TrimSpace(orderID) == "" || strings.TrimSpace(executionID) == "" {
		return
	}
	if _, loaded := workflowTraceRegistry.LoadOrStore(executionID, struct{}{}); loaded {
		return
	}

	go func() {
		defer workflowTraceRegistry.Delete(executionID)

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		deadline := time.Now().Add(6 * time.Hour)
		lastStatus := -1
		for {
			var exec workflow.WorkflowExecution
			if err := db.First(&exec, "id = ?", executionID).Error; err != nil {
				addComment(db, orderID, operator, "system", fmt.Sprintf("Workflow 追踪中断：执行ID %s 查询失败。", executionID))
				return
			}

			if exec.Status != lastStatus {
				if exec.Status == 4 {
					syncWorkOrderPendingApproval(db, orderID, &exec, workflowID, operator)
				}
				lastStatus = exec.Status
			}

			if isWorkflowTerminal(exec.Status) {
				syncWorkOrderStatusByWorkflow(db, orderID, &exec, operator)
				addComment(db, orderID, operator, "system", buildWorkflowExecutionSummary(db, &exec, workflowID))
				return
			}

			if time.Now().After(deadline) {
				addComment(db, orderID, operator, "system", fmt.Sprintf("Workflow 追踪超时：%s (%s)，请在执行记录中继续查看。", nonEmpty(exec.WorkflowName, workflowID), executionID))
				return
			}

			<-ticker.C
		}
	}()
}

func syncWorkOrderPendingApproval(db *gorm.DB, orderID string, exec *workflow.WorkflowExecution, workflowID, operator string) bool {
	if db == nil || exec == nil || strings.TrimSpace(orderID) == "" {
		return false
	}

	var order WorkOrder
	if err := db.First(&order, "id = ?", orderID).Error; err != nil {
		return false
	}
	if order.Status == 6 {
		return false
	}
	if order.Status == 1 {
		return false
	}

	update := map[string]interface{}{
		"completed_at": nil,
	}
	update["status"] = 1
	if err := db.Model(&order).Updates(update).Error; err != nil {
		return false
	}

	addComment(db, orderID, operator, "system",
		fmt.Sprintf("工单已切换为“审批中”：Workflow %s (%s) 正在等待审批。请在工作流执行记录处理审批节点，处理后系统会继续追踪并回写结果。",
			nonEmpty(exec.WorkflowName, workflowID), exec.ID))
	return true
}

func isWorkflowTerminal(status int) bool {
	switch status {
	case 1, 2, 3:
		return true
	default:
		return false
	}
}

func workflowStatusText(status int) string {
	switch status {
	case 0:
		return "运行中"
	case 1:
		return "成功"
	case 2:
		return "失败"
	case 3:
		return "取消"
	case 4:
		return "等待审批"
	default:
		return "未知"
	}
}

func buildWorkflowExecutionSummary(db *gorm.DB, exec *workflow.WorkflowExecution, fallbackWorkflowID string) string {
	lines := []string{
		fmt.Sprintf("Workflow 执行结束：%s (%s)", nonEmpty(exec.WorkflowName, fallbackWorkflowID), exec.ID),
		"状态: " + workflowStatusText(exec.Status),
	}

	var total, success, failed, skipped int64
	db.Model(&workflow.WorkflowNodeExecution{}).Where("execution_id = ?", exec.ID).Count(&total)
	db.Model(&workflow.WorkflowNodeExecution{}).Where("execution_id = ? AND status = 1", exec.ID).Count(&success)
	db.Model(&workflow.WorkflowNodeExecution{}).Where("execution_id = ? AND status = 2", exec.ID).Count(&failed)
	db.Model(&workflow.WorkflowNodeExecution{}).Where("execution_id = ? AND status = 3", exec.ID).Count(&skipped)
	if total > 0 {
		lines = append(lines, fmt.Sprintf("节点结果: total=%d, success=%d, failed=%d, skipped=%d", total, success, failed, skipped))
	}

	if strings.TrimSpace(exec.Error) != "" {
		lines = append(lines, "执行错误: "+truncateForComment(exec.Error, 240))
	}

	if exec.Status == 2 {
		var failedNode workflow.WorkflowNodeExecution
		if err := db.Where("execution_id = ? AND status = 2", exec.ID).Order("started_at DESC").First(&failedNode).Error; err == nil {
			nodeName := strings.TrimSpace(failedNode.NodeName)
			if nodeName == "" {
				nodeName = failedNode.NodeID
			}
			lines = append(lines, "失败节点: "+nodeName)
			if strings.TrimSpace(failedNode.Error) != "" {
				lines = append(lines, "节点错误: "+truncateForComment(failedNode.Error, 200))
			}
		}
	}

	return strings.Join(lines, "\n")
}

func findActiveWorkflowExecutionForOrder(db *gorm.DB, order *WorkOrder, payload *aiPlanFormData) *workflow.WorkflowExecution {
	if db == nil || order == nil || payload == nil {
		return nil
	}
	workflowID := strings.TrimSpace(payload.GeneratedWorkflowID)
	if workflowID == "" {
		return nil
	}

	var exec workflow.WorkflowExecution
	likePattern := fmt.Sprintf("%%\"workorder_id\":\"%s\"%%", order.ID)
	err := db.Where("workflow_id = ? AND status IN (0, 4) AND variables LIKE ?", workflowID, likePattern).
		Order("started_at DESC").
		First(&exec).Error
	if err == nil {
		return &exec
	}
	if err != gorm.ErrRecordNotFound {
		return nil
	}

	// 兼容历史数据：若变量未注入工单ID，回退按 workflow_id 查活动执行。
	if err = db.Where("workflow_id = ? AND status IN (0, 4)", workflowID).
		Order("started_at DESC").
		First(&exec).Error; err != nil {
		return nil
	}
	return &exec
}

func loadLatestWorkflowRuntime(db *gorm.DB, order *WorkOrder) *WorkOrderWorkflowRuntime {
	if db == nil || order == nil {
		return nil
	}
	payload := parseAIPlanFormData(order.FormData)
	if payload == nil || strings.TrimSpace(payload.GeneratedWorkflowID) == "" {
		return nil
	}
	workflowID := strings.TrimSpace(payload.GeneratedWorkflowID)

	var exec workflow.WorkflowExecution
	likePattern := fmt.Sprintf("%%\"workorder_id\":\"%s\"%%", order.ID)
	err := db.Where("workflow_id = ? AND variables LIKE ?", workflowID, likePattern).
		Order("started_at DESC").
		First(&exec).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil
		}
		if err = db.Where("workflow_id = ?", workflowID).Order("started_at DESC").First(&exec).Error; err != nil {
			return nil
		}
	}

	result := &WorkOrderWorkflowRuntime{
		WorkflowID:  workflowID,
		ExecutionID: exec.ID,
		Status:      exec.Status,
		StatusText:  workflowStatusText(exec.Status),
		CurrentNode: strings.TrimSpace(exec.CurrentNode),
		TriggerBy:   strings.TrimSpace(exec.TriggerBy),
		Duration:    exec.Duration,
		Error:       strings.TrimSpace(exec.Error),
	}
	result.StartedAt = &exec.StartedAt
	result.FinishedAt = exec.FinishedAt

	db.Model(&workflow.WorkflowNodeExecution{}).Where("execution_id = ?", exec.ID).Count(&result.TotalNodes)
	db.Model(&workflow.WorkflowNodeExecution{}).Where("execution_id = ? AND status = 1", exec.ID).Count(&result.SuccessNodes)
	db.Model(&workflow.WorkflowNodeExecution{}).Where("execution_id = ? AND status = 2", exec.ID).Count(&result.FailedNodes)
	db.Model(&workflow.WorkflowNodeExecution{}).Where("execution_id = ? AND status = 3", exec.ID).Count(&result.SkippedNodes)

	if exec.Status == 2 {
		var failedNode workflow.WorkflowNodeExecution
		if err := db.Where("execution_id = ? AND status = 2", exec.ID).Order("started_at DESC").First(&failedNode).Error; err == nil {
			result.FailedNodeID = failedNode.NodeID
			result.FailedNodeName = strings.TrimSpace(failedNode.NodeName)
			result.FailedNodeError = strings.TrimSpace(failedNode.Error)
		}
	}

	return result
}

func syncWorkOrderStatusByWorkflow(db *gorm.DB, orderID string, exec *workflow.WorkflowExecution, operator string) bool {
	if db == nil || exec == nil || strings.TrimSpace(orderID) == "" {
		return false
	}
	var order WorkOrder
	if err := db.First(&order, "id = ?", orderID).Error; err != nil {
		return false
	}

	// 用户主动取消后，不覆盖工单状态。
	if order.Status == 6 {
		return false
	}

	switch exec.Status {
	case 1:
		if order.Status == 5 {
			return false
		}
		now := time.Now()
		if err := db.Model(&order).Updates(map[string]interface{}{
			"status":       5,
			"completed_at": now,
		}).Error; err == nil {
			addComment(db, orderID, operator, "system", fmt.Sprintf("Workflow 执行成功，工单已自动完成：%s (%s)。", nonEmpty(exec.WorkflowName, ""), exec.ID))
			return true
		}
	case 2:
		if order.Status == 2 && order.CompletedAt == nil {
			return false
		}
		if err := db.Model(&order).Updates(map[string]interface{}{
			"status":       2,
			"completed_at": nil,
		}).Error; err == nil {
			addComment(db, orderID, operator, "system", "Workflow 执行失败，工单已回退到“已通过”，请人工处理后可重试执行。")
			return true
		}
	case 3:
		if order.Status == 2 && order.CompletedAt == nil {
			return false
		}
		if err := db.Model(&order).Updates(map[string]interface{}{
			"status":       2,
			"completed_at": nil,
		}).Error; err == nil {
			addComment(db, orderID, operator, "system", "Workflow 执行已取消，工单已回退到“已通过”，可人工处理后重新执行。")
			return true
		}
	}
	return false
}

func truncateForComment(text string, max int) string {
	raw := strings.TrimSpace(text)
	if len(raw) <= max {
		return raw
	}
	return raw[:max] + "..."
}

func nonEmpty(primary, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return strings.TrimSpace(primary)
	}
	return strings.TrimSpace(fallback)
}
