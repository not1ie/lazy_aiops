package ai

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// DiagnoseOps 故障诊断闭环入口
func (h *AIHandler) DiagnoseOps(c *gin.Context) {
	var req AIOpsDiagnoseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	userID := h.currentUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未提供认证信息"})
		return
	}
	resp, err := h.service.DiagnoseIncident(&req, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": resp})
}

// PreflightOps 变更前风险评分
func (h *AIHandler) PreflightOps(c *gin.Context) {
	var req AIOpsPreflightRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	result, err := h.service.PreflightRisk(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// ApproveOps 审批诊断结果（通过时创建工单）
func (h *AIHandler) ApproveOps(c *gin.Context) {
	var req AIOpsApproveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	userID := h.currentUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未提供认证信息"})
		return
	}
	username := strings.TrimSpace(c.GetString("username"))
	order, err := h.service.ApproveIncident(&req, userID, username)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "审批结果已记录",
		"data": gin.H{
			"approved": req.Approved,
			"order":    order,
		},
	})
}

// ExecuteOps 回写 apply/verify/rollback 结果
func (h *AIHandler) ExecuteOps(c *gin.Context) {
	var req AIOpsExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	actor := strings.TrimSpace(c.GetString("username"))
	incident, err := h.service.MarkIncidentStage(&req, actor)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": incident})
}

// TimelineOps 导出时间轴
func (h *AIHandler) TimelineOps(c *gin.Context) {
	var req AIOpsTimelineQuery
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	result, err := h.service.BuildTimeline(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// ListIncidentsOps 查看 incident 历史
func (h *AIHandler) ListIncidentsOps(c *gin.Context) {
	status := strings.TrimSpace(c.Query("status"))
	limit := 50
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		if num, err := strconv.Atoi(raw); err == nil {
			limit = num
		}
	}
	rows, err := h.service.ListIncidents(status, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": rows})
}

// GetIncidentOps 查看 incident 详情
func (h *AIHandler) GetIncidentOps(c *gin.Context) {
	incidentID := strings.TrimSpace(c.Param("id"))
	if incidentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "incident id 不能为空"})
		return
	}
	detail, err := h.service.GetIncidentDetail(incidentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": detail})
}

// GenerateRunbookOps 从 incident 生成 runbook 文档
func (h *AIHandler) GenerateRunbookOps(c *gin.Context) {
	var req AIOpsRunbookGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	operator := strings.TrimSpace(c.GetString("username"))
	doc, err := h.service.GenerateRunbookFromIncident(&req, operator)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": doc})
}
