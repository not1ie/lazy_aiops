package workorder

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/plugins/workflow"
	"gorm.io/gorm"
)

type WorkOrderHandler struct {
	db         *gorm.DB
	aiAnalyzer func(order *WorkOrder) (suggestion, risk string)
}

func NewWorkOrderHandler(db *gorm.DB) *WorkOrderHandler {
	return &WorkOrderHandler{db: db}
}

func (h *WorkOrderHandler) SetAIAnalyzer(analyzer func(order *WorkOrder) (string, string)) {
	h.aiAnalyzer = analyzer
}

// ListTypes 工单类型列表
func (h *WorkOrderHandler) ListTypes(c *gin.Context) {
	var types []WorkOrderType
	if err := h.db.Find(&types).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": types})
}

// CreateType 创建工单类型
func (h *WorkOrderHandler) CreateType(c *gin.Context) {
	var t WorkOrderType
	if err := c.ShouldBindJSON(&t); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := h.db.Create(&t).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": t})
}

// UpdateType 更新工单类型
func (h *WorkOrderHandler) UpdateType(c *gin.Context) {
	id := c.Param("id")
	var t WorkOrderType
	if err := h.db.First(&t, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "工单类型不存在"})
		return
	}
	var req WorkOrderType
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	updates := map[string]interface{}{
		"name":        req.Name,
		"code":        req.Code,
		"icon":        req.Icon,
		"flow_id":     req.FlowID,
		"template":    req.Template,
		"enabled":     req.Enabled,
		"description": req.Description,
	}
	if err := h.db.Model(&t).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	_ = h.db.First(&t, "id = ?", id)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": t})
}

// DeleteType 删除工单类型
func (h *WorkOrderHandler) DeleteType(c *gin.Context) {
	id := c.Param("id")
	var count int64
	h.db.Model(&WorkOrder{}).Where("type_id = ?", id).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "该类型下存在工单，不能删除"})
		return
	}
	if err := h.db.Delete(&WorkOrderType{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// ListOrders 工单列表
func (h *WorkOrderHandler) ListOrders(c *gin.Context) {
	var orders []WorkOrder
	query := h.db.Order("created_at DESC")

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if typeID := c.Query("type_id"); typeID != "" {
		query = query.Where("type_id = ?", typeID)
	}
	if submitter := c.Query("submitter"); submitter != "" {
		query = query.Where("submitter_id = ?", submitter)
	}
	if assignee := c.Query("assignee"); assignee != "" {
		query = query.Where("assignee_id = ?", assignee)
	}

	// 我的待办
	if c.Query("my_pending") == "true" {
		userID := c.GetString("user_id")
		query = query.Where("assignee_id = ? AND status IN (0, 1, 4)", userID)
	}

	if err := query.Limit(200).Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": orders})
}

// CreateOrder 创建工单
func (h *WorkOrderHandler) CreateOrder(c *gin.Context) {
	var order WorkOrder
	if err := c.ShouldBindJSON(&order); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	order.Submitter = c.GetString("username")
	order.SubmitterID = c.GetString("user_id")
	order.Status = 0

	// 获取工单类型
	var orderType WorkOrderType
	if err := h.db.First(&orderType, "id = ?", order.TypeID).Error; err == nil {
		order.TypeName = orderType.Name
	}

	// AI分析
	if h.aiAnalyzer != nil {
		suggestion, risk := h.aiAnalyzer(&order)
		order.AISuggestion = suggestion
		order.AIRisk = risk
	}
	createdOrder, err := CreateOrderWithDefaults(h.db, CreateOrderInput{
		TypeCode:     orderType.Code,
		Title:        order.Title,
		Content:      order.Content,
		FormData:     order.FormData,
		Priority:     order.Priority,
		Submitter:    order.Submitter,
		SubmitterID:  order.SubmitterID,
		Assignee:     order.Assignee,
		AssigneeID:   order.AssigneeID,
		AISuggestion: order.AISuggestion,
		AIRisk:       order.AIRisk,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": createdOrder})
}

// GetOrder 获取工单详情
func (h *WorkOrderHandler) GetOrder(c *gin.Context) {
	id := c.Param("id")
	var order WorkOrder
	if err := h.db.First(&order, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "工单不存在"})
		return
	}

	// 获取审批步骤
	var steps []WorkOrderStep
	h.db.Where("order_id = ?", id).Order("step").Find(&steps)

	// 获取评论
	var comments []WorkOrderComment
	h.db.Where("order_id = ?", id).Order("created_at").Find(&comments)
	workflowRuntime := loadLatestWorkflowRuntime(h.db, &order)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"order":            order,
			"steps":            steps,
			"comments":         comments,
			"workflow_runtime": workflowRuntime,
		},
	})
}

// ApproveOrder 审批工单
func (h *WorkOrderHandler) ApproveOrder(c *gin.Context) {
	id := c.Param("id")
	var order WorkOrder
	if err := h.db.First(&order, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "工单不存在"})
		return
	}

	var req struct {
		Approved bool   `json:"approved"`
		Comment  string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	userID := c.GetString("user_id")
	username := c.GetString("username")
	now := time.Now()

	// 更新当前步骤
	var step WorkOrderStep
	if err := h.db.Where("order_id = ? AND step = ? AND status = 0", id, order.CurrentStep).First(&step).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "当前步骤不存在或已处理"})
		return
	}

	status := 1
	if !req.Approved {
		status = 2
	}

	h.db.Model(&step).Updates(map[string]interface{}{
		"status":      status,
		"approver":    username,
		"approver_id": userID,
		"comment":     req.Comment,
		"approved_at": now,
	})

	// 更新工单状态
	if req.Approved {
		var generatedWorkflow *workflow.Workflow
		if order.CurrentStep >= order.TotalSteps {
			// 审批完成
			h.db.Model(&order).Updates(map[string]interface{}{
				"status": 2,
			})
			addComment(h.db, id, username, "system", "审批通过")
			_ = h.db.First(&order, "id = ?", id)
			workflowDraft, err := maybeCreateRunbookWorkflow(h.db, &order, username)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
				return
			}
			generatedWorkflow = workflowDraft
			c.JSON(http.StatusOK, gin.H{
				"code":    0,
				"message": "审批完成",
				"data": gin.H{
					"workflow": generatedWorkflow,
				},
			})
			return
		} else {
			// 进入下一步
			h.db.Model(&order).Updates(map[string]interface{}{
				"current_step": order.CurrentStep + 1,
				"status":       1,
			})
			addComment(h.db, id, username, "system", "审批通过，进入下一步")
		}
	} else {
		h.db.Model(&order).Update("status", 3)
		addComment(h.db, id, username, "system", "审批拒绝: "+req.Comment)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "审批完成"})
}

// ExecuteOrder 执行工单
func (h *WorkOrderHandler) ExecuteOrder(c *gin.Context) {
	id := c.Param("id")
	var order WorkOrder
	if err := h.db.First(&order, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "工单不存在"})
		return
	}

	if order.Status != 2 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "工单未审批通过"})
		return
	}

	username := c.GetString("username")
	payload := parseAIPlanFormData(order.FormData)
	manualItems := manualConfirmationItems(payload)

	executionID := ""
	workflowID := ""
	comment := "开始执行"

	if payload != nil && strings.TrimSpace(payload.GeneratedWorkflowID) != "" {
		workflowID = strings.TrimSpace(payload.GeneratedWorkflowID)
		if activeExec := findActiveWorkflowExecutionForOrder(h.db, &order, payload); activeExec != nil {
			c.JSON(http.StatusConflict, gin.H{
				"code":    409,
				"message": fmt.Sprintf("已存在进行中的 Workflow 执行（%s），请勿重复触发。", activeExec.ID),
				"data": gin.H{
					"workflow_id":  activeExec.WorkflowID,
					"execution_id": activeExec.ID,
					"status":       activeExec.Status,
					"status_text":  workflowStatusText(activeExec.Status),
				},
			})
			return
		}
		execution, err := workflow.ExecuteWorkflowByID(h.db, workflowID, buildRunbookExecutionVariables(&order, payload), username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "触发关联 Workflow 失败: " + err.Error()})
			return
		}
		executionID = execution.ID
		comment = "开始执行，已触发 Workflow Runbook: " + workflowID
		if strings.TrimSpace(executionID) != "" {
			comment += "，执行ID: " + executionID
		}
	}

	h.db.Model(&order).Updates(map[string]interface{}{
		"status":   4,
		"assignee": username,
	})
	addComment(h.db, id, username, "system", comment)
	if strings.TrimSpace(executionID) != "" {
		startWorkflowExecutionTrace(h.db, id, workflowID, executionID, username)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "开始执行",
		"data": gin.H{
			"workflow_id":  workflowID,
			"execution_id": executionID,
			"manual_items": manualItems,
		},
	})
}

// CompleteOrder 完成工单
func (h *WorkOrderHandler) CompleteOrder(c *gin.Context) {
	id := c.Param("id")
	var order WorkOrder
	if err := h.db.First(&order, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "工单不存在"})
		return
	}

	var req struct {
		Result string `json:"result"`
	}
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	payload := parseAIPlanFormData(order.FormData)
	if activeExec := findActiveWorkflowExecutionForOrder(h.db, &order, payload); activeExec != nil {
		c.JSON(http.StatusConflict, gin.H{
			"code":    409,
			"message": fmt.Sprintf("存在未结束的 Workflow 执行（%s），请先等待完成或取消后再手动完成工单。", activeExec.ID),
			"data": gin.H{
				"workflow_id":  activeExec.WorkflowID,
				"execution_id": activeExec.ID,
				"status":       activeExec.Status,
				"status_text":  workflowStatusText(activeExec.Status),
			},
		})
		return
	}

	now := time.Now()
	username := c.GetString("username")
	h.db.Model(&order).Updates(map[string]interface{}{
		"status":       5,
		"completed_at": now,
	})
	addComment(h.db, id, username, "system", "工单已完成: "+req.Result)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "工单已完成"})
}

// CancelOrder 取消工单
func (h *WorkOrderHandler) CancelOrder(c *gin.Context) {
	id := c.Param("id")
	username := c.GetString("username")
	var order WorkOrder
	if err := h.db.First(&order, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "工单不存在"})
		return
	}

	h.db.Model(&order).Updates(map[string]interface{}{
		"status":       6,
		"completed_at": nil,
	})
	addComment(h.db, id, username, "system", "工单已取消")

	executionID := ""
	cancelErrText := ""
	payload := parseAIPlanFormData(order.FormData)
	if activeExec := findActiveWorkflowExecutionForOrder(h.db, &order, payload); activeExec != nil {
		executionID = activeExec.ID
		if err := workflow.CancelWorkflowExecutionByID(h.db, activeExec.ID); err != nil {
			cancelErrText = err.Error()
			addComment(h.db, id, username, "system", "工单已取消，但关联 Workflow 取消失败："+cancelErrText)
		} else {
			addComment(h.db, id, username, "system", "已联动取消关联 Workflow 执行："+activeExec.ID)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "工单已取消",
		"data": gin.H{
			"execution_id": executionID,
			"cancel_error": cancelErrText,
		},
	})
}

// AddComment 添加评论
func (h *WorkOrderHandler) AddComment(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	addComment(h.db, id, c.GetString("username"), "comment", req.Content)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "评论成功"})
}

// GetStats 统计
func (h *WorkOrderHandler) GetStats(c *gin.Context) {
	var total, pending, processing, completed int64
	h.db.Model(&WorkOrder{}).Count(&total)
	h.db.Model(&WorkOrder{}).Where("status IN (0, 1)").Count(&pending)
	h.db.Model(&WorkOrder{}).Where("status = 4").Count(&processing)
	h.db.Model(&WorkOrder{}).Where("status = 5").Count(&completed)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"total":      total,
			"pending":    pending,
			"processing": processing,
			"completed":  completed,
		},
	})
}

// 辅助方法
func createApprovalSteps(db *gorm.DB, order *WorkOrder, orderType *WorkOrderType) {
	if orderType == nil || orderType.FlowID == "" {
		order.TotalSteps = 1
		order.CurrentStep = 1
		db.Save(order)
		db.Create(&WorkOrderStep{
			OrderID: order.ID,
			Step:    1,
			Name:    "默认审批",
			Status:  0,
		})
		return
	}

	var flow WorkOrderFlow
	if err := db.First(&flow, "id = ?", orderType.FlowID).Error; err != nil {
		order.TotalSteps = 1
		order.CurrentStep = 1
		db.Save(order)
		db.Create(&WorkOrderStep{
			OrderID: order.ID,
			Step:    1,
			Name:    "默认审批",
			Status:  0,
		})
		return
	}

	var steps []map[string]interface{}
	if err := json.Unmarshal([]byte(flow.Steps), &steps); err != nil {
		order.TotalSteps = 1
		order.CurrentStep = 1
		db.Save(order)
		db.Create(&WorkOrderStep{
			OrderID: order.ID,
			Step:    1,
			Name:    "默认审批",
			Status:  0,
		})
		return
	}
	if len(steps) == 0 {
		order.TotalSteps = 1
		order.CurrentStep = 1
		db.Save(order)
		db.Create(&WorkOrderStep{
			OrderID: order.ID,
			Step:    1,
			Name:    "默认审批",
			Status:  0,
		})
		return
	}

	order.TotalSteps = len(steps)
	order.CurrentStep = 1
	db.Save(order)

	for i, s := range steps {
		step := WorkOrderStep{
			OrderID: order.ID,
			Step:    i + 1,
			Name:    s["name"].(string),
			Status:  0,
		}
		if approver, ok := s["approver"].(string); ok {
			step.Approver = approver
		}
		db.Create(&step)
	}
}

func addComment(db *gorm.DB, orderID, username, commentType, content string) {
	comment := WorkOrderComment{
		OrderID:  orderID,
		Username: username,
		Type:     commentType,
		Content:  content,
	}
	db.Create(&comment)
}
