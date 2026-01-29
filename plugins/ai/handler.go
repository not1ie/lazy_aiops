package ai

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AIHandler struct {
	db      *gorm.DB
	service *AIService
}

func NewAIHandler(db *gorm.DB, service *AIService) *AIHandler {
	return &AIHandler{db: db, service: service}
}

// Chat 对话
func (h *AIHandler) Chat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	userID := c.GetString("user_id")
	resp, err := h.service.Chat(req.SessionID, userID, req.Message, req.Context)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": resp})
}

// ListSessions 会话列表
func (h *AIHandler) ListSessions(c *gin.Context) {
	userID := c.GetString("user_id")
	var sessions []ChatSession
	if err := h.db.Where("user_id = ?", userID).Order("updated_at DESC").Find(&sessions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": sessions})
}

// GetSessionMessages 获取会话消息
func (h *AIHandler) GetSessionMessages(c *gin.Context) {
	sessionID := c.Param("id")
	var messages []ChatMessage
	if err := h.db.Where("session_id = ?", sessionID).Order("created_at").Find(&messages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": messages})
}

// DeleteSession 删除会话
func (h *AIHandler) DeleteSession(c *gin.Context) {
	sessionID := c.Param("id")
	h.db.Delete(&ChatMessage{}, "session_id = ?", sessionID)
	h.db.Delete(&ChatSession{}, "id = ?", sessionID)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// AnalyzeLogs 分析日志
func (h *AIHandler) AnalyzeLogs(c *gin.Context) {
	var req struct {
		Logs    string `json:"logs" binding:"required"`
		Context string `json:"context"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	resp, err := h.service.AnalyzeLogs(req.Logs, req.Context)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": resp})
}

// AnalyzeLogsDetailed 详细日志分析（新增）
func (h *AIHandler) AnalyzeLogsDetailed(c *gin.Context) {
	var req LogAnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	// 模拟分析过程（实际应该调用AI服务）
	result, err := h.service.AnalyzeLogsDetailed(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	// 保存分析结果到数据库
	analysisRecord := LogAnalysis{
		Service:      result.Service,
		NeedAlert:    result.NeedAlert,
		AlertLevel:   result.AlertLevel,
		RootCause:    result.RootCause,
		Impact:       strings.Join(result.Impact, "; "),
		Solutions:    strings.Join(result.Solutions, "; "),
		Prevention:   strings.Join(result.Prevention, "; "),
		Confidence:   result.Confidence,
		LogCount:     result.LogCount,
		ErrorCount:   result.ErrorCount,
		WarningCount: result.WarningCount,
	}
	h.db.Create(&analysisRecord)

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// ListAnalysisHistory 分析历史记录
func (h *AIHandler) ListAnalysisHistory(c *gin.Context) {
	var records []LogAnalysis
	query := h.db.Order("created_at DESC")

	if service := c.Query("service"); service != "" {
		query = query.Where("service = ?", service)
	}

	if err := query.Limit(100).Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": records})
}

// GetAnalysisDetail 获取分析详情
func (h *AIHandler) GetAnalysisDetail(c *gin.Context) {
	id := c.Param("id")
	var record LogAnalysis
	if err := h.db.First(&record, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "记录不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": record})
}

// AnalyzeError 分析错误
func (h *AIHandler) AnalyzeError(c *gin.Context) {
	var req struct {
		Error      string `json:"error" binding:"required"`
		StackTrace string `json:"stack_trace"`
		Context    string `json:"context"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	resp, err := h.service.AnalyzeError(req.Error, req.StackTrace, req.Context)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": resp})
}

// AnalyzePerformance 分析性能
func (h *AIHandler) AnalyzePerformance(c *gin.Context) {
	var req struct {
		Target  string `json:"target" binding:"required"`
		Metrics string `json:"metrics"`
		Context string `json:"context"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	resp, err := h.service.SuggestOptimize(req.Target, req.Metrics, req.Context)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"suggestion": resp}})
}

// SuggestFix 建议修复
func (h *AIHandler) SuggestFix(c *gin.Context) {
	var req struct {
		Issue   string `json:"issue" binding:"required"`
		Context string `json:"context"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	resp, err := h.service.SuggestFix(req.Issue, req.Context)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"suggestion": resp}})
}

// SuggestOptimize 建议优化
func (h *AIHandler) SuggestOptimize(c *gin.Context) {
	var req struct {
		Target  string `json:"target" binding:"required"`
		Metrics string `json:"metrics"`
		Context string `json:"context"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	resp, err := h.service.SuggestOptimize(req.Target, req.Metrics, req.Context)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"suggestion": resp}})
}
