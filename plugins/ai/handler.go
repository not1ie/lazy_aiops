package ai

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"gorm.io/gorm"
)

type AIHandler struct {
	db      *gorm.DB
	service *AIService
}

type providerConfigPayload struct {
	Name          string          `json:"name"`
	Provider      string          `json:"provider"`
	BaseURL       string          `json:"base_url"`
	Model         string          `json:"model"`
	AuthType      string          `json:"auth_type"`
	APIKey        string          `json:"api_key"`
	TimeoutSecond int             `json:"timeout_second"`
	ExtraHeaders  json.RawMessage `json:"extra_headers"`
	Description   string          `json:"description"`
}

func NewAIHandler(db *gorm.DB, service *AIService) *AIHandler {
	return &AIHandler{db: db, service: service}
}

func normalizeProviderPayload(req *providerConfigPayload) {
	req.Name = strings.TrimSpace(req.Name)
	req.Provider = strings.TrimSpace(strings.ToLower(req.Provider))
	req.BaseURL = strings.TrimSpace(req.BaseURL)
	req.Model = strings.TrimSpace(req.Model)
	req.AuthType = strings.TrimSpace(strings.ToLower(req.AuthType))
	req.Description = strings.TrimSpace(req.Description)

	if req.Name == "" {
		req.Name = "custom-api"
	}
	if req.Provider == "" {
		req.Provider = "openai"
	}
	if req.Model == "" {
		req.Model = "gpt-3.5-turbo"
	}
	if req.AuthType == "" {
		req.AuthType = "bearer"
	}
	if req.TimeoutSecond <= 0 {
		req.TimeoutSecond = 60
	}
}

func parseHeaderMap(raw json.RawMessage) (map[string]string, string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return map[string]string{}, "{}", nil
	}
	headers := map[string]string{}

	var asMap map[string]interface{}
	if err := json.Unmarshal(raw, &asMap); err == nil {
		for key, value := range asMap {
			k := strings.TrimSpace(key)
			v := strings.TrimSpace(fmt.Sprint(value))
			if k == "" || v == "" || v == "<nil>" {
				continue
			}
			headers[k] = v
		}
		serialized, _ := json.Marshal(headers)
		return headers, string(serialized), nil
	}

	var rawString string
	if err := json.Unmarshal(raw, &rawString); err != nil {
		return nil, "", err
	}
	rawString = strings.TrimSpace(rawString)
	if rawString == "" {
		return map[string]string{}, "{}", nil
	}
	if err := json.Unmarshal([]byte(rawString), &asMap); err != nil {
		return nil, "", err
	}
	for key, value := range asMap {
		k := strings.TrimSpace(key)
		v := strings.TrimSpace(fmt.Sprint(value))
		if k == "" || v == "" || v == "<nil>" {
			continue
		}
		headers[k] = v
	}
	serialized, _ := json.Marshal(headers)
	return headers, string(serialized), nil
}

func (h *AIHandler) currentUserID(c *gin.Context) string {
	userID := strings.TrimSpace(c.GetString("user_id"))
	if userID != "" {
		return userID
	}
	return strings.TrimSpace(c.GetString("username"))
}

// Chat 对话
func (h *AIHandler) Chat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	userID := h.currentUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未提供认证信息"})
		return
	}
	resp, err := h.service.Chat(&req, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": resp})
}

// BuildContextPack 预览自动上下文包
func (h *AIHandler) BuildContextPack(c *gin.Context) {
	var req struct {
		ContextHint *AIContextHint `json:"context_hint"`
	}
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	pack, err := h.service.BuildContextPack(req.ContextHint)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": pack})
}

// CreateApprovalWorkOrderFromMessage 从 AI 助手消息创建审批工单
func (h *AIHandler) CreateApprovalWorkOrderFromMessage(c *gin.Context) {
	messageID := strings.TrimSpace(c.Param("id"))
	if messageID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "消息ID不能为空"})
		return
	}
	userID := h.currentUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未提供认证信息"})
		return
	}
	username := strings.TrimSpace(c.GetString("username"))
	order, plan, err := h.service.CreateWorkOrderFromMessage(messageID, userID, username)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"order": order,
			"plan":  plan,
		},
	})
}

// CreateSession 显式创建会话
func (h *AIHandler) CreateSession(c *gin.Context) {
	var req struct {
		Title   string `json:"title"`
		Context string `json:"context"`
	}
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	userID := h.currentUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未提供认证信息"})
		return
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "新会话"
	}
	session := ChatSession{
		UserID:  userID,
		Title:   title,
		Type:    "chat",
		Context: strings.TrimSpace(req.Context),
	}
	if err := h.db.Create(&session).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": session})
}

// ListSessions 会话列表
func (h *AIHandler) ListSessions(c *gin.Context) {
	userID := h.currentUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未提供认证信息"})
		return
	}
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
	userID := h.currentUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未提供认证信息"})
		return
	}
	var session ChatSession
	if err := h.db.First(&session, "id = ?", sessionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "会话不存在"})
		return
	}
	if session.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权访问该会话"})
		return
	}
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
	userID := h.currentUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未提供认证信息"})
		return
	}
	var session ChatSession
	if err := h.db.First(&session, "id = ?", sessionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "会话不存在"})
		return
	}
	if session.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权删除该会话"})
		return
	}
	h.db.Delete(&ChatMessage{}, "session_id = ?", sessionID)
	h.db.Delete(&ChatSession{}, "id = ?", sessionID)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// ListProviderConfigs 模型接入配置列表
func (h *AIHandler) ListProviderConfigs(c *gin.Context) {
	var list []AIProviderConfig
	if err := h.db.Order("updated_at desc").Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	for i := range list {
		if strings.TrimSpace(list[i].APIKey) != "" {
			list[i].APIKey = "******"
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"configs": list,
			"runtime": h.service.RuntimeSnapshot(),
		},
	})
}

// GetProviderConfig 获取模型配置详情（用于编辑）
func (h *AIHandler) GetProviderConfig(c *gin.Context) {
	id := c.Param("id")
	var cfg AIProviderConfig
	if err := h.db.First(&cfg, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "配置不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": cfg})
}

// CreateProviderConfig 新增模型接入配置
func (h *AIHandler) CreateProviderConfig(c *gin.Context) {
	var req providerConfigPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	normalizeProviderPayload(&req)
	if req.AuthType != "bearer" && req.AuthType != "x-api-key" && req.AuthType != "api-key" && req.AuthType != "none" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "auth_type 仅支持 bearer/x-api-key/api-key/none"})
		return
	}
	_, serializedHeaders, err := parseHeaderMap(req.ExtraHeaders)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "extra_headers 必须是对象或 JSON 字符串"})
		return
	}
	cfg := AIProviderConfig{
		Name:          req.Name,
		Provider:      req.Provider,
		BaseURL:       req.BaseURL,
		Model:         req.Model,
		AuthType:      req.AuthType,
		APIKey:        strings.TrimSpace(req.APIKey),
		ExtraHeaders:  serializedHeaders,
		TimeoutSecond: req.TimeoutSecond,
		Description:   req.Description,
	}

	var activeCount int64
	h.db.Model(&AIProviderConfig{}).Where("active = ?", true).Count(&activeCount)
	if activeCount == 0 {
		cfg.Active = true
	}
	if err := h.db.Create(&cfg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	if cfg.Active {
		h.service.ApplyProviderConfig(&cfg)
	}
	cfg.APIKey = ""
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": cfg})
}

// UpdateProviderConfig 更新模型接入配置
func (h *AIHandler) UpdateProviderConfig(c *gin.Context) {
	id := c.Param("id")
	var existing AIProviderConfig
	if err := h.db.First(&existing, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "配置不存在"})
		return
	}

	var req providerConfigPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	normalizeProviderPayload(&req)
	if req.AuthType != "bearer" && req.AuthType != "x-api-key" && req.AuthType != "api-key" && req.AuthType != "none" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "auth_type 仅支持 bearer/x-api-key/api-key/none"})
		return
	}
	_, serializedHeaders, err := parseHeaderMap(req.ExtraHeaders)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "extra_headers 必须是对象或 JSON 字符串"})
		return
	}
	apiKey := existing.APIKey
	if trimmed := strings.TrimSpace(req.APIKey); trimmed != "" {
		apiKey = trimmed
	}
	updates := map[string]interface{}{
		"name":           req.Name,
		"provider":       req.Provider,
		"base_url":       req.BaseURL,
		"model":          req.Model,
		"auth_type":      req.AuthType,
		"api_key":        apiKey,
		"extra_headers":  serializedHeaders,
		"timeout_second": req.TimeoutSecond,
		"description":    req.Description,
	}
	if err := h.db.Model(&AIProviderConfig{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	if existing.Active {
		h.service.ApplyProviderConfig(&AIProviderConfig{
			Provider:      req.Provider,
			APIKey:        apiKey,
			BaseURL:       req.BaseURL,
			Model:         req.Model,
			AuthType:      req.AuthType,
			ExtraHeaders:  serializedHeaders,
			TimeoutSecond: req.TimeoutSecond,
		})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// DeleteProviderConfig 删除模型接入配置
func (h *AIHandler) DeleteProviderConfig(c *gin.Context) {
	id := c.Param("id")
	var current AIProviderConfig
	if err := h.db.First(&current, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "配置不存在"})
		return
	}
	if err := h.db.Delete(&AIProviderConfig{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	if current.Active {
		var fallback AIProviderConfig
		if err := h.db.Order("updated_at desc").First(&fallback).Error; err == nil {
			h.db.Model(&AIProviderConfig{}).Where("id = ?", fallback.ID).Update("active", true)
			h.service.ApplyProviderConfig(&fallback)
		}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// ActivateProviderConfig 激活模型接入配置
func (h *AIHandler) ActivateProviderConfig(c *gin.Context) {
	id := c.Param("id")
	var cfg AIProviderConfig
	if err := h.db.First(&cfg, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "配置不存在"})
		return
	}
	if err := h.db.Model(&AIProviderConfig{}).Where("active = ?", true).Update("active", false).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	if err := h.db.Model(&AIProviderConfig{}).Where("id = ?", id).Update("active", true).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	h.service.ApplyProviderConfig(&cfg)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已激活"})
}

// TestProviderConfig 测试模型接入配置
func (h *AIHandler) TestProviderConfig(c *gin.Context) {
	id := c.Param("id")
	var cfg AIProviderConfig
	if err := h.db.First(&cfg, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "配置不存在"})
		return
	}

	headers := map[string]string{}
	if strings.TrimSpace(cfg.ExtraHeaders) != "" {
		if err := json.Unmarshal([]byte(cfg.ExtraHeaders), &headers); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "extra_headers JSON 无效"})
			return
		}
	}
	tester := core.NewAIService(cfg.Provider, cfg.APIKey, cfg.BaseURL, cfg.Model)
	tester.UpdateConfig(cfg.Provider, cfg.APIKey, cfg.BaseURL, cfg.Model, cfg.AuthType, headers, cfg.TimeoutSecond)
	reply, tokens, err := tester.CallSimple("请仅返回: pong")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "测试失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "连接成功", "data": gin.H{"reply": strings.TrimSpace(reply), "token_used": tokens}})
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
