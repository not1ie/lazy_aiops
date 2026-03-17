package workflow

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type WorkflowHandler struct {
	db     *gorm.DB
	engine *Engine
}

func NewWorkflowHandler(db *gorm.DB, engine *Engine) *WorkflowHandler {
	return &WorkflowHandler{db: db, engine: engine}
}

// ListWorkflows 工作流列表
func (h *WorkflowHandler) ListWorkflows(c *gin.Context) {
	var workflows []Workflow
	query := h.db.Order("created_at DESC")
	if category := c.Query("category"); category != "" {
		query = query.Where("category = ?", category)
	}
	if err := query.Find(&workflows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": workflows})
}

// CreateWorkflow 创建工作流
func (h *WorkflowHandler) CreateWorkflow(c *gin.Context) {
	var workflow Workflow
	if err := c.ShouldBindJSON(&workflow); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	workflow.CreatedBy = c.GetString("username")
	if err := h.db.Create(&workflow).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": workflow})
}

// GetWorkflow 获取工作流详情
func (h *WorkflowHandler) GetWorkflow(c *gin.Context) {
	id := c.Param("id")
	var workflow Workflow
	if err := h.db.First(&workflow, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "工作流不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": workflow})
}

// UpdateWorkflow 更新工作流
func (h *WorkflowHandler) UpdateWorkflow(c *gin.Context) {
	id := c.Param("id")
	var workflow Workflow
	if err := h.db.First(&workflow, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "工作流不存在"})
		return
	}
	var req Workflow
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	updates := map[string]interface{}{
		"name":        req.Name,
		"description": req.Description,
		"category":    req.Category,
		"definition":  req.Definition,
		"variables":   req.Variables,
		"trigger":     req.Trigger,
		"cron_expr":   req.CronExpr,
		"enabled":     req.Enabled,
		"version":     workflow.Version + 1,
	}
	if err := h.db.Model(&workflow).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	_ = h.db.First(&workflow, "id = ?", id)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": workflow})
}

// DeleteWorkflow 删除工作流
func (h *WorkflowHandler) DeleteWorkflow(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&Workflow{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// ExecuteWorkflow 执行工作流
func (h *WorkflowHandler) ExecuteWorkflow(c *gin.Context) {
	id := c.Param("id")
	var workflow Workflow
	if err := h.db.First(&workflow, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "工作流不存在"})
		return
	}

	var req struct {
		Variables map[string]interface{} `json:"variables"`
	}
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if req.Variables == nil {
		req.Variables = make(map[string]interface{})
	}

	execution, err := h.engine.Execute(&workflow, req.Variables, c.GetString("username"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": execution})
}

// ListExecutions 执行记录列表
func (h *WorkflowHandler) ListExecutions(c *gin.Context) {
	var executions []WorkflowExecution
	query := h.db.Order("started_at DESC")

	if workflowID := c.Query("workflow_id"); workflowID != "" {
		query = query.Where("workflow_id = ?", workflowID)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Limit(100).Find(&executions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": executions})
}

// GetExecution 获取执行详情
func (h *WorkflowHandler) GetExecution(c *gin.Context) {
	id := c.Param("id")
	var execution WorkflowExecution
	if err := h.db.First(&execution, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "执行记录不存在"})
		return
	}

	// 获取节点执行记录
	var nodeExecutions []WorkflowNodeExecution
	h.db.Where("execution_id = ?", id).Order("started_at").Find(&nodeExecutions)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"execution": execution,
			"nodes":     nodeExecutions,
		},
	})
}

// CancelExecution 取消执行
func (h *WorkflowHandler) CancelExecution(c *gin.Context) {
	id := c.Param("id")
	if err := h.engine.Cancel(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已取消"})
}

// ListTemplates 模板列表
func (h *WorkflowHandler) ListTemplates(c *gin.Context) {
	var templates []WorkflowTemplate
	if err := h.db.Find(&templates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": templates})
}

// CreateFromTemplate 从模板创建
func (h *WorkflowHandler) CreateFromTemplate(c *gin.Context) {
	id := c.Param("id")
	var template WorkflowTemplate
	if err := h.db.First(&template, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "模板不存在"})
		return
	}

	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	workflow := Workflow{
		Name:        req.Name,
		Description: template.Description,
		Category:    template.Category,
		Definition:  template.Definition,
		Variables:   template.Variables,
		Trigger:     "manual",
		CreatedBy:   c.GetString("username"),
	}

	if err := h.db.Create(&workflow).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": workflow})
}

// WebhookTrigger Webhook触发
func (h *WorkflowHandler) WebhookTrigger(c *gin.Context) {
	id := c.Param("id")
	var workflow Workflow
	if err := h.db.First(&workflow, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "工作流不存在"})
		return
	}

	if workflow.Trigger != "webhook" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "该工作流不支持Webhook触发"})
		return
	}

	var variables map[string]interface{}
	if err := c.ShouldBindJSON(&variables); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if variables == nil {
		variables = make(map[string]interface{})
	}

	execution, err := h.engine.Execute(&workflow, variables, "webhook")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": execution})
}

// GetStats 统计
func (h *WorkflowHandler) GetStats(c *gin.Context) {
	var total, running, success, failed int64
	h.db.Model(&WorkflowExecution{}).Count(&total)
	h.db.Model(&WorkflowExecution{}).Where("status = 0").Count(&running)
	h.db.Model(&WorkflowExecution{}).Where("status = 1").Count(&success)
	h.db.Model(&WorkflowExecution{}).Where("status = 2").Count(&failed)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"total":   total,
			"running": running,
			"success": success,
			"failed":  failed,
		},
	})
}

// ValidateDefinition 验证流程定义
func (h *WorkflowHandler) ValidateDefinition(c *gin.Context) {
	var req struct {
		Definition string `json:"definition" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	var definition struct {
		Nodes []Node `json:"nodes"`
	}
	if err := json.Unmarshal([]byte(req.Definition), &definition); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "JSON格式错误: " + err.Error()})
		return
	}

	// 验证节点
	hasStart, hasEnd := false, false
	for _, node := range definition.Nodes {
		if node.Type == "start" {
			hasStart = true
		}
		if node.Type == "end" {
			hasEnd = true
		}
	}

	if !hasStart {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "缺少开始节点"})
		return
	}
	if !hasEnd {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "缺少结束节点"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "验证通过", "data": gin.H{"node_count": len(definition.Nodes)}})
}
