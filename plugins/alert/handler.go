package alert

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AlertHandler struct {
	db         *gorm.DB
	aggregator *Aggregator
	notifier   func(channelID, title, content string) error
}

func NewAlertHandler(db *gorm.DB, aggregator *Aggregator) *AlertHandler {
	return &AlertHandler{db: db, aggregator: aggregator}
}

func (h *AlertHandler) SetNotifier(notifier func(channelID, title, content string) error) {
	h.notifier = notifier
}

// ListRules 规则列表
func (h *AlertHandler) ListRules(c *gin.Context) {
	var rules []AlertRule
	if err := h.db.Find(&rules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": rules})
}

// CreateRule 创建规则
func (h *AlertHandler) CreateRule(c *gin.Context) {
	var rule AlertRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := h.db.Create(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": rule})
}

// UpdateRule 更新规则
func (h *AlertHandler) UpdateRule(c *gin.Context) {
	id := c.Param("id")
	var rule AlertRule
	if err := h.db.First(&rule, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "规则不存在"})
		return
	}
	var req AlertRule
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	updates := map[string]interface{}{
		"name":            req.Name,
		"type":            req.Type,
		"target":          req.Target,
		"metric":          req.Metric,
		"operator":        req.Operator,
		"threshold":       req.Threshold,
		"duration":        req.Duration,
		"severity":        req.Severity,
		"notify_group_id": req.NotifyGroupID,
		"enabled":         req.Enabled,
		"ai_analysis":     req.AIAnalysis,
		"auto_recover":    req.AutoRecover,
		"recover_script":  req.RecoverScript,
		"description":     req.Description,
	}
	if err := h.db.Model(&rule).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	_ = h.db.First(&rule, "id = ?", id)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": rule})
}

// DeleteRule 删除规则
func (h *AlertHandler) DeleteRule(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&AlertRule{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// ListAlerts 告警列表
func (h *AlertHandler) ListAlerts(c *gin.Context) {
	var alerts []Alert
	query := h.db.Order("fired_at DESC")

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if severity := c.Query("severity"); severity != "" {
		query = query.Where("severity = ?", severity)
	}
	if target := c.Query("target"); target != "" {
		query = query.Where("target LIKE ?", "%"+target+"%")
	}

	if err := query.Limit(200).Find(&alerts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": alerts})
}

// GetAlert 获取告警详情
func (h *AlertHandler) GetAlert(c *gin.Context) {
	id := c.Param("id")
	var alert Alert
	if err := h.db.First(&alert, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "告警不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": alert})
}

// AckAlert 确认告警
func (h *AlertHandler) AckAlert(c *gin.Context) {
	id := c.Param("id")
	var alert Alert
	if err := h.db.First(&alert, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "告警不存在"})
		return
	}

	now := time.Now()
	h.db.Model(&alert).Updates(map[string]interface{}{
		"status":   1,
		"acked_at": now,
		"acked_by": c.GetString("username"),
	})

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "确认成功"})
}

// ResolveAlert 解决告警
func (h *AlertHandler) ResolveAlert(c *gin.Context) {
	id := c.Param("id")
	var alert Alert
	if err := h.db.First(&alert, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "告警不存在"})
		return
	}

	h.aggregator.Resolve(alert.Fingerprint)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已解决"})
}

// ReceiveAlert 接收告警(Webhook)
func (h *AlertHandler) ReceiveAlert(c *gin.Context) {
	var req struct {
		RuleID    string            `json:"rule_id"`
		RuleName  string            `json:"rule_name"`
		Target    string            `json:"target"`
		Metric    string            `json:"metric"`
		Value     string            `json:"value"`
		Threshold string            `json:"threshold"`
		Severity  string            `json:"severity"`
		Labels    map[string]string `json:"labels"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	labelsJSON, _ := json.Marshal(req.Labels)
	alert := &Alert{
		RuleID:    req.RuleID,
		RuleName:  req.RuleName,
		Target:    req.Target,
		Metric:    req.Metric,
		Value:     req.Value,
		Threshold: req.Threshold,
		Severity:  req.Severity,
		Status:    0,
		FiredAt:   time.Now(),
		Labels:    string(labelsJSON),
	}

	// 通过聚合器处理
	processedAlert, shouldNotify := h.aggregator.Process(alert)

	// 发送通知
	if shouldNotify && h.notifier != nil {
		summary := h.aggregator.GenerateSummary(processedAlert.GroupKey)
		title := "🚨 告警通知: " + processedAlert.RuleName
		notifyGroupID := ""
		if ruleID := strings.TrimSpace(processedAlert.RuleID); ruleID != "" {
			var rule AlertRule
			if err := h.db.Select("notify_group_id").First(&rule, "id = ?", ruleID).Error; err == nil {
				notifyGroupID = strings.TrimSpace(rule.NotifyGroupID)
			}
		}
		go h.notifier(notifyGroupID, title, summary)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": processedAlert})
}

// GetStats 获取统计
func (h *AlertHandler) GetStats(c *gin.Context) {
	// 聚合器统计
	aggStats := h.aggregator.GetStats()

	// 数据库统计
	var totalAlerts, activeAlerts, resolvedAlerts int64
	h.db.Model(&Alert{}).Count(&totalAlerts)
	h.db.Model(&Alert{}).Where("status = 0").Count(&activeAlerts)
	h.db.Model(&Alert{}).Where("status = 2").Count(&resolvedAlerts)

	// 今日统计
	today := time.Now().Truncate(24 * time.Hour)
	var todayAlerts int64
	h.db.Model(&Alert{}).Where("fired_at >= ?", today).Count(&todayAlerts)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"total":       totalAlerts,
			"active":      activeAlerts,
			"resolved":    resolvedAlerts,
			"today":       todayAlerts,
			"aggregation": aggStats,
		},
	})
}

// ListSilences 静默列表
func (h *AlertHandler) ListSilences(c *gin.Context) {
	var silences []AlertSilence
	if err := h.db.Find(&silences).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": silences})
}

// CreateSilence 创建静默
func (h *AlertHandler) CreateSilence(c *gin.Context) {
	var silence AlertSilence
	if err := c.ShouldBindJSON(&silence); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	silence.CreatedBy = c.GetString("username")
	if err := h.db.Create(&silence).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": silence})
}

// UpdateSilence 更新静默
func (h *AlertHandler) UpdateSilence(c *gin.Context) {
	id := c.Param("id")
	var silence AlertSilence
	if err := h.db.First(&silence, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "静默不存在"})
		return
	}
	var req struct {
		Name     *string    `json:"name"`
		Matchers *string    `json:"matchers"`
		StartsAt *time.Time `json:"starts_at"`
		EndsAt   *time.Time `json:"ends_at"`
		Comment  *string    `json:"comment"`
		Status   *int       `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Matchers != nil {
		updates["matchers"] = *req.Matchers
	}
	if req.StartsAt != nil {
		updates["starts_at"] = *req.StartsAt
	}
	if req.EndsAt != nil {
		updates["ends_at"] = *req.EndsAt
	}
	if req.Comment != nil {
		updates["comment"] = *req.Comment
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if len(updates) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": silence})
		return
	}
	if err := h.db.Model(&silence).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	if err := h.db.First(&silence, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": silence})
}

// ListAggregations 聚合配置列表
func (h *AlertHandler) ListAggregations(c *gin.Context) {
	var aggs []AlertAggregation
	if err := h.db.Find(&aggs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": aggs})
}

// CreateAggregation 创建聚合配置
func (h *AlertHandler) CreateAggregation(c *gin.Context) {
	var agg AlertAggregation
	if err := c.ShouldBindJSON(&agg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := h.db.Create(&agg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": agg})
}

// UpdateAggregation 更新聚合配置
func (h *AlertHandler) UpdateAggregation(c *gin.Context) {
	id := c.Param("id")
	var agg AlertAggregation
	if err := h.db.First(&agg, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "聚合配置不存在"})
		return
	}
	var req struct {
		Name        *string `json:"name"`
		GroupBy     *string `json:"group_by"`
		Interval    *int    `json:"interval"`
		Enabled     *bool   `json:"enabled"`
		Description *string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.GroupBy != nil {
		updates["group_by"] = *req.GroupBy
	}
	if req.Interval != nil {
		updates["interval"] = *req.Interval
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if len(updates) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": agg})
		return
	}
	if err := h.db.Model(&agg).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	if err := h.db.First(&agg, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": agg})
}

// DeleteAggregation 删除聚合配置
func (h *AlertHandler) DeleteAggregation(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&AlertAggregation{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// ListHistory 告警复盘历史
func (h *AlertHandler) ListHistory(c *gin.Context) {
	var items []AlertHistory
	query := h.db.Order("fired_at DESC")
	if sev := c.Query("severity"); sev != "" {
		query = query.Where("severity = ?", sev)
	}
	if target := c.Query("target"); target != "" {
		query = query.Where("target LIKE ?", "%"+target+"%")
	}
	if ruleID := c.Query("rule_id"); ruleID != "" {
		query = query.Where("rule_id LIKE ?", "%"+ruleID+"%")
	}
	if start := c.Query("start"); start != "" {
		query = query.Where("fired_at >= ?", start)
	}
	if end := c.Query("end"); end != "" {
		query = query.Where("fired_at <= ?", end)
	}
	if err := query.Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": items})
}

// GetHistory 获取复盘详情
func (h *AlertHandler) GetHistory(c *gin.Context) {
	id := c.Param("id")
	var item AlertHistory
	if err := h.db.First(&item, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "记录不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": item})
}

// UpdateHistory 更新复盘信息
func (h *AlertHandler) UpdateHistory(c *gin.Context) {
	id := c.Param("id")
	var item AlertHistory
	if err := h.db.First(&item, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "记录不存在"})
		return
	}
	var req struct {
		Resolution string `json:"resolution"`
		RootCause  string `json:"root_cause"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := h.db.Model(&item).Updates(map[string]interface{}{
		"resolution": req.Resolution,
		"root_cause": req.RootCause,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// DeleteSilence 删除静默
func (h *AlertHandler) DeleteSilence(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&AlertSilence{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}
