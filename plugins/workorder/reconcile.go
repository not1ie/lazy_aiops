package workorder

import (
	"strings"

	"github.com/lazyautoops/lazy-auto-ops/plugins/workflow"
	"gorm.io/gorm"
)

const defaultReconcileBatchLimit = 200

func reconcileWorkflowLinkedOrders(db *gorm.DB, operator string, limit int) (int, int, error) {
	if db == nil {
		return 0, 0, nil
	}
	if strings.TrimSpace(operator) == "" {
		operator = "system/reconcile"
	}
	if limit <= 0 {
		limit = defaultReconcileBatchLimit
	}

	var orders []WorkOrder
	if err := db.
		Where("status IN ?", []int{1, 2, 4}).
		Order("updated_at DESC").
		Limit(limit).
		Find(&orders).Error; err != nil {
		return 0, 0, err
	}

	checked := 0
	updated := 0
	for i := range orders {
		order := &orders[i]
		payload := parseAIPlanFormData(order.FormData)
		if payload == nil || strings.TrimSpace(payload.GeneratedWorkflowID) == "" {
			continue
		}
		checked++

		var exec workflow.WorkflowExecution
		likePattern := "%"
		if strings.TrimSpace(order.ID) != "" {
			likePattern = "%\"workorder_id\":\"" + strings.TrimSpace(order.ID) + "\"%"
		}

		query := db.Where("workflow_id = ?", strings.TrimSpace(payload.GeneratedWorkflowID))
		if likePattern != "%" {
			query = query.Where("variables LIKE ?", likePattern)
		}
		if err := query.Order("started_at DESC").First(&exec).Error; err != nil {
			continue
		}

		if exec.Status == 4 {
			if syncWorkOrderPendingApproval(db, order.ID, &exec, strings.TrimSpace(payload.GeneratedWorkflowID), operator) {
				updated++
			}
			continue
		}
		if isWorkflowTerminal(exec.Status) {
			if syncWorkOrderStatusByWorkflow(db, order.ID, &exec, operator) {
				updated++
			}
		}
	}
	return checked, updated, nil
}
