package sre

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// SreHandler 处理所有 /api/v1/sre/* 路由
type SreHandler struct {
	client *SreClient
}

func NewSreHandler(client *SreClient) *SreHandler {
	return &SreHandler{client: client}
}

// Ping GET /api/v1/sre/ping — lazysre 健康检查
func (h *SreHandler) Ping(c *gin.Context) {
	resp, err := h.client.Ping(context.Background())
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"code":    503,
			"message": "lazysre 不可用: " + err.Error(),
			"data":    SreBridgeStatus{Connected: false, BaseURL: h.client.baseURL, Message: err.Error()},
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": SreBridgeStatus{Connected: true, BaseURL: h.client.baseURL, Message: resp.Status},
	})
}

// Status GET /api/v1/sre/status — 插件状态（不强依赖 sidecar）
func (h *SreHandler) Status(c *gin.Context) {
	available := h.client.IsAvailable(context.Background())
	statusStr := "online"
	if !available {
		statusStr = "offline"
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{"sidecar_status": statusStr, "sidecar_url": h.client.baseURL},
	})
}

// Overview GET /api/v1/sre/overview — 平台概览
func (h *SreHandler) Overview(c *gin.Context) {
	result, err := h.client.GetOverview(context.Background())
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"code": 502, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// Briefing GET /api/v1/sre/briefing — 环境快速总览
func (h *SreHandler) Briefing(c *gin.Context) {
	result, err := h.client.GetBriefing(context.Background())
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"code": 502, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// ListTools GET /api/v1/sre/tools — 可用工具列表
func (h *SreHandler) ListTools(c *gin.Context) {
	result, err := h.client.ListTools(context.Background())
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"code": 502, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// ListWorkflows GET /api/v1/sre/workflows — Skill/Workflow 列表
func (h *SreHandler) ListWorkflows(c *gin.Context) {
	result, err := h.client.ListWorkflows(context.Background())
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"code": 502, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// ListTemplates GET /api/v1/sre/templates — 模板列表
func (h *SreHandler) ListTemplates(c *gin.Context) {
	result, err := h.client.ListTemplates(context.Background())
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"code": 502, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// QuickStart POST /api/v1/sre/quickstart — 自然语言触发诊断/任务
func (h *SreHandler) QuickStart(c *gin.Context) {
	var req SreDiagnoseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误: " + err.Error()})
		return
	}
	result, err := h.client.QuickStart(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"code": 502, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// ListRuns GET /api/v1/sre/runs — 执行记录列表
func (h *SreHandler) ListRuns(c *gin.Context) {
	limit := 20
	result, err := h.client.ListRuns(context.Background(), limit)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"code": 502, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// GetRun GET /api/v1/sre/runs/:run_id — 单次执行详情
func (h *SreHandler) GetRun(c *gin.Context) {
	runID := c.Param("run_id")
	if runID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "run_id 不能为空"})
		return
	}
	result, err := h.client.GetRun(context.Background(), runID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"code": 502, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// ApproveRun POST /api/v1/sre/runs/:run_id/approve — 审批执行
func (h *SreHandler) ApproveRun(c *gin.Context) {
	runID := c.Param("run_id")
	if runID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "run_id 不能为空"})
		return
	}
	var req SreApproveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误: " + err.Error()})
		return
	}
	result, err := h.client.ApproveRun(context.Background(), runID, &req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"code": 502, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// ListTasks GET /api/v1/sre/tasks — 任务列表
func (h *SreHandler) ListTasks(c *gin.Context) {
	result, err := h.client.ListTasks(context.Background())
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"code": 502, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// GetArtifacts GET /api/v1/sre/artifacts — 产物/报告列表
func (h *SreHandler) GetArtifacts(c *gin.Context) {
	result, err := h.client.GetArtifacts(context.Background())
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"code": 502, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// RegisterRoutes 注册路由（由 plugin.go 调用）
func (h *SreHandler) RegisterRoutes(g *gin.RouterGroup) {
	// 健康与状态
	g.GET("/ping", h.Ping)
	g.GET("/status", h.Status)

	// 概览
	g.GET("/overview", h.Overview)
	g.GET("/briefing", h.Briefing)

	// 工具 & Workflow
	g.GET("/tools", h.ListTools)
	g.GET("/workflows", h.ListWorkflows)
	g.GET("/templates", h.ListTemplates)

	// 核心：自然语言触发
	g.POST("/quickstart", h.QuickStart)

	// 执行记录
	g.GET("/runs", h.ListRuns)
	g.GET("/runs/:run_id", h.GetRun)
	g.POST("/runs/:run_id/approve", h.ApproveRun)

	// 任务 & 产物
	g.GET("/tasks", h.ListTasks)
	g.GET("/artifacts", h.GetArtifacts)

	fmt.Println("[SRE] 路由注册完成: /ping /status /overview /briefing /tools /workflows /templates /quickstart /runs /tasks /artifacts")
}
