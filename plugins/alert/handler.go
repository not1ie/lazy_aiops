package alert

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
	"github.com/lazyautoops/lazy-auto-ops/plugins/workorder"
	"gorm.io/gorm"
)

type AlertHandler struct {
	db         *gorm.DB
	aggregator *Aggregator
	notifier   func(channelID, title, content string) error
}

type createWorkOrderFromAlertRequest struct {
	TypeCode    string `json:"type_code"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	Priority    int    `json:"priority"`
	Assignee    string `json:"assignee"`
	AssigneeID  string `json:"assignee_id"`
	AutoApprove bool   `json:"auto_approve"`
	AutoExecute bool   `json:"auto_execute"`
	WorkflowID  string `json:"workflow_id"`
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
		"status":        1,
		"acked_at":      now,
		"acked_by":      c.GetString("username"),
		"status_reason": "已人工确认",
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
	_ = h.db.Model(&alert).Update("status_reason", "告警已恢复").Error
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已解决"})
}

// CreateWorkOrderFromAlert 告警转工单（可选自动触发 Runbook）
func (h *AlertHandler) CreateWorkOrderFromAlert(c *gin.Context) {
	id := c.Param("id")
	var alert Alert
	if err := h.db.First(&alert, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "告警不存在"})
		return
	}

	if strings.TrimSpace(alert.WorkOrderID) != "" {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "该告警已关联工单",
			"data": gin.H{
				"alert_id":              alert.ID,
				"work_order_id":         alert.WorkOrderID,
				"workflow_id":           alert.WorkflowID,
				"workflow_execution_id": alert.WorkflowExecutionID,
				"linked_at":             alert.LinkedAt,
				"status_reason":         alert.StatusReason,
			},
		})
		return
	}

	if !h.db.Migrator().HasTable(&workorder.WorkOrder{}) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "工单模块未启用，无法执行告警转工单"})
		return
	}

	var req createWorkOrderFromAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if req.AutoExecute && strings.TrimSpace(req.WorkflowID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "启用自动执行时，workflow_id 不能为空"})
		return
	}

	username := strings.TrimSpace(c.GetString("username"))
	if username == "" {
		username = "system"
	}
	userID := strings.TrimSpace(c.GetString("user_id"))

	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = fmt.Sprintf("【告警】%s - %s", nonEmptyText(alert.RuleName, alert.Metric, alert.ID), nonEmptyText(alert.Target, "-"))
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		content = buildAlertWorkOrderContent(&alert)
	}

	formDataRaw, _ := json.Marshal(gin.H{
		"source":      "alert_link",
		"alert_id":    alert.ID,
		"rule_id":     alert.RuleID,
		"rule_name":   alert.RuleName,
		"target":      alert.Target,
		"metric":      alert.Metric,
		"severity":    alert.Severity,
		"value":       alert.Value,
		"threshold":   alert.Threshold,
		"fingerprint": alert.Fingerprint,
	})

	order, err := workorder.CreateOrderWithDefaults(h.db, workorder.CreateOrderInput{
		TypeCode:     nonEmptyText(strings.TrimSpace(req.TypeCode), "incident"),
		Title:        title,
		Content:      content,
		FormData:     string(formDataRaw),
		Priority:     req.Priority,
		Submitter:    username,
		SubmitterID:  userID,
		Assignee:     strings.TrimSpace(req.Assignee),
		AssigneeID:   strings.TrimSpace(req.AssigneeID),
		AIRisk:       strings.TrimSpace(alert.Severity),
		AISuggestion: strings.TrimSpace(alert.AISuggestion),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建工单失败: " + err.Error()})
		return
	}

	now := time.Now()
	alertUpdates := map[string]interface{}{
		"work_order_id": order.ID,
		"linked_at":     now,
		"status_reason": fmt.Sprintf("已关联工单 %s", order.ID),
	}
	if alert.Status == 0 {
		alertUpdates["status"] = 1
		alertUpdates["acked_at"] = now
		alertUpdates["acked_by"] = username
	}
	if err := h.db.Model(&alert).Updates(alertUpdates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新告警状态失败: " + err.Error()})
		return
	}
	addWorkOrderSystemComment(h.db, order.ID, username, fmt.Sprintf("由告警联动创建工单：%s (%s)", nonEmptyText(alert.RuleName, alert.Metric), alert.ID))

	if req.AutoApprove {
		_ = h.db.Model(&workorder.WorkOrderStep{}).
			Where("order_id = ? AND status = 0", order.ID).
			Updates(map[string]interface{}{
				"status":      1,
				"approver":    username,
				"approver_id": userID,
				"comment":     "告警联动自动审批",
				"approved_at": now,
			}).Error
		_ = h.db.Model(&workorder.WorkOrder{}).Where("id = ?", order.ID).Updates(map[string]interface{}{
			"status":       2,
			"current_step": order.TotalSteps,
		}).Error
		addWorkOrderSystemComment(h.db, order.ID, username, "工单已自动审批通过")
	}

	executionID := ""
	workflowID := strings.TrimSpace(req.WorkflowID)
	if req.AutoExecute && workflowID != "" {
		var latest workorder.WorkOrder
		if err := h.db.First(&latest, "id = ?", order.ID).Error; err == nil {
			if latest.Status != 2 {
				c.JSON(http.StatusBadRequest, gin.H{
					"code":    400,
					"message": "自动执行前工单必须处于“已通过”状态，请启用 auto_approve 或先完成审批",
					"data": gin.H{
						"work_order_id": order.ID,
						"status":        latest.Status,
					},
				})
				return
			}
		}
		exec, execErr := workflow.ExecuteWorkflowByID(h.db, workflowID, map[string]interface{}{
			"alert_id":       alert.ID,
			"alert_target":   alert.Target,
			"alert_metric":   alert.Metric,
			"alert_value":    alert.Value,
			"alert_severity": alert.Severity,
			"workorder_id":   order.ID,
			"rule_name":      alert.RuleName,
		}, username)
		if execErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "触发 Runbook 失败: " + execErr.Error(),
				"data": gin.H{
					"work_order_id": order.ID,
				},
			})
			return
		}
		executionID = exec.ID
		_ = h.db.Model(&workorder.WorkOrder{}).Where("id = ?", order.ID).Updates(map[string]interface{}{
			"status":      4,
			"assignee":    username,
			"assignee_id": userID,
		}).Error
		addWorkOrderSystemComment(h.db, order.ID, username, fmt.Sprintf("已联动触发 Runbook：%s（执行ID: %s）", workflowID, executionID))
		_ = h.db.Model(&alert).Updates(map[string]interface{}{
			"workflow_id":           workflowID,
			"workflow_execution_id": executionID,
			"status_reason":         fmt.Sprintf("已关联工单 %s，并触发 Runbook 执行 %s", order.ID, executionID),
		}).Error
	}

	_ = h.db.First(&alert, "id = ?", id).Error
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "告警联动工单成功",
		"data": gin.H{
			"alert_id":              alert.ID,
			"work_order_id":         alert.WorkOrderID,
			"workflow_id":           alert.WorkflowID,
			"workflow_execution_id": alert.WorkflowExecutionID,
			"linked_at":             alert.LinkedAt,
			"status_reason":         alert.StatusReason,
		},
	})
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
		RuleID:       req.RuleID,
		RuleName:     req.RuleName,
		Target:       req.Target,
		Metric:       req.Metric,
		Value:        req.Value,
		Threshold:    req.Threshold,
		Severity:     req.Severity,
		Status:       0,
		StatusReason: "告警触发，待处理",
		FiredAt:      time.Now(),
		Labels:       string(labelsJSON),
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

func addWorkOrderSystemComment(db *gorm.DB, orderID, username, content string) {
	if db == nil || strings.TrimSpace(orderID) == "" || strings.TrimSpace(content) == "" {
		return
	}
	item := workorder.WorkOrderComment{
		OrderID:  strings.TrimSpace(orderID),
		Username: strings.TrimSpace(username),
		Type:     "system",
		Content:  strings.TrimSpace(content),
	}
	_ = db.Create(&item).Error
}

func buildAlertWorkOrderContent(alert *Alert) string {
	if alert == nil {
		return "告警联动工单"
	}
	lines := []string{
		fmt.Sprintf("告警规则: %s", nonEmptyText(alert.RuleName, "-")),
		fmt.Sprintf("监控目标: %s", nonEmptyText(alert.Target, "-")),
		fmt.Sprintf("监控指标: %s", nonEmptyText(alert.Metric, "-")),
		fmt.Sprintf("当前值/阈值: %s / %s", nonEmptyText(alert.Value, "-"), nonEmptyText(alert.Threshold, "-")),
		fmt.Sprintf("严重级别: %s", nonEmptyText(alert.Severity, "warning")),
		fmt.Sprintf("触发时间: %s", alert.FiredAt.Format(time.RFC3339)),
		"",
		"处理建议:",
		"1. 先确认告警真实性与影响范围",
		"2. 按既定 SOP 执行排障与回滚预案",
		"3. 处理完成后补充根因与复盘结论",
	}
	if strings.TrimSpace(alert.AISuggestion) != "" {
		lines = append(lines, "", "AI建议:", strings.TrimSpace(alert.AISuggestion))
	}
	return strings.Join(lines, "\n")
}

func nonEmptyText(values ...string) string {
	for i := range values {
		v := strings.TrimSpace(values[i])
		if v != "" {
			return v
		}
	}
	return ""
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
