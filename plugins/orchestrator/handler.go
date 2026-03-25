package orchestrator

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/plugins/workflow"
	"gorm.io/gorm"
)

type OrchestratorHandler struct {
	db     *gorm.DB
	engine *workflow.Engine
}

func NewOrchestratorHandler(db *gorm.DB, engine *workflow.Engine) *OrchestratorHandler {
	return &OrchestratorHandler{db: db, engine: engine}
}

type ruleRequest struct {
	Name             string `json:"name" binding:"required"`
	Source           string `json:"source"`
	EventType        string `json:"event_type"`
	WorkflowID       string `json:"workflow_id" binding:"required"`
	MatchContains    string `json:"match_contains"`
	DefaultVariables string `json:"default_variables"`
	Enabled          *bool  `json:"enabled"`
}

type ingestRequest struct {
	Source    string                 `json:"source"`
	EventType string                 `json:"event_type"`
	External  string                 `json:"external_id"`
	Summary   string                 `json:"summary"`
	Payload   map[string]interface{} `json:"payload"`
	Variables map[string]interface{} `json:"variables"`
}

type runbookExecRequest struct {
	WorkflowID string                 `json:"workflow_id" binding:"required"`
	Source     string                 `json:"source"`
	EventType  string                 `json:"event_type"`
	Summary    string                 `json:"summary"`
	Variables  map[string]interface{} `json:"variables"`
	Payload    map[string]interface{} `json:"payload"`
}

func (h *OrchestratorHandler) GetOverview(c *gin.Context) {
	var rulesTotal, rulesEnabled int64
	var events24h, dispatched24h, failed24h int64

	_ = h.db.Model(&OrchestrationRule{}).Count(&rulesTotal).Error
	_ = h.db.Model(&OrchestrationRule{}).Where("enabled = ?", true).Count(&rulesEnabled).Error

	since := time.Now().Add(-24 * time.Hour)
	_ = h.db.Model(&OrchestrationEvent{}).Where("received_at >= ?", since).Count(&events24h).Error
	_ = h.db.Model(&OrchestrationEvent{}).Where("received_at >= ? AND status IN ?", since, []string{"dispatched", "partial"}).Count(&dispatched24h).Error
	_ = h.db.Model(&OrchestrationEvent{}).Where("received_at >= ? AND status = ?", since, "failed").Count(&failed24h).Error

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"rules_total":      rulesTotal,
			"rules_enabled":    rulesEnabled,
			"events_24h":       events24h,
			"dispatched_24h":   dispatched24h,
			"failed_24h":       failed24h,
			"generated_at":     time.Now(),
			"orchestrator_ver": "v1",
		},
	})
}

func (h *OrchestratorHandler) ListRules(c *gin.Context) {
	var rows []OrchestrationRule
	query := h.db.Order("updated_at DESC")
	if source := strings.TrimSpace(c.Query("source")); source != "" {
		query = query.Where("source = ?", source)
	}
	if eventType := strings.TrimSpace(c.Query("event_type")); eventType != "" {
		query = query.Where("event_type = ?", eventType)
	}
	if enabled := strings.TrimSpace(c.Query("enabled")); enabled != "" {
		query = query.Where("enabled = ?", enabled == "true" || enabled == "1")
	}
	if err := query.Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": rows})
}

func (h *OrchestratorHandler) CreateRule(c *gin.Context) {
	var req ruleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	wf, err := h.loadWorkflow(req.WorkflowID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if err := validateVariablesJSON(req.DefaultVariables); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "默认变量JSON格式错误"})
		return
	}

	r := OrchestrationRule{
		Name:             strings.TrimSpace(req.Name),
		Source:           normalizeWildcardField(req.Source),
		EventType:        normalizeWildcardField(req.EventType),
		WorkflowID:       wf.ID,
		WorkflowName:     wf.Name,
		MatchContains:    strings.TrimSpace(req.MatchContains),
		DefaultVariables: strings.TrimSpace(req.DefaultVariables),
		CreatedBy:        c.GetString("username"),
		Enabled:          true,
	}
	if req.Enabled != nil {
		r.Enabled = *req.Enabled
	}
	if err := h.db.Create(&r).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": r})
}

func (h *OrchestratorHandler) UpdateRule(c *gin.Context) {
	id := c.Param("id")
	var row OrchestrationRule
	if err := h.db.First(&row, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "规则不存在"})
		return
	}

	var req ruleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	wf, err := h.loadWorkflow(req.WorkflowID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if err := validateVariablesJSON(req.DefaultVariables); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "默认变量JSON格式错误"})
		return
	}

	updates := map[string]interface{}{
		"name":              strings.TrimSpace(req.Name),
		"source":            normalizeWildcardField(req.Source),
		"event_type":        normalizeWildcardField(req.EventType),
		"workflow_id":       wf.ID,
		"workflow_name":     wf.Name,
		"match_contains":    strings.TrimSpace(req.MatchContains),
		"default_variables": strings.TrimSpace(req.DefaultVariables),
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if err := h.db.Model(&row).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	_ = h.db.First(&row, "id = ?", id).Error
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": row})
}

func (h *OrchestratorHandler) DeleteRule(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&OrchestrationRule{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

func (h *OrchestratorHandler) ListEvents(c *gin.Context) {
	var rows []OrchestrationEvent
	query := h.db.Order("received_at DESC")
	if source := strings.TrimSpace(c.Query("source")); source != "" {
		query = query.Where("source = ?", source)
	}
	if eventType := strings.TrimSpace(c.Query("event_type")); eventType != "" {
		query = query.Where("event_type = ?", eventType)
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	}
	limit := 100
	if err := query.Limit(limit).Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": rows})
}

func (h *OrchestratorHandler) ListDispatches(c *gin.Context) {
	var rows []OrchestrationDispatch
	query := h.db.Order("started_at DESC")
	if eventID := strings.TrimSpace(c.Query("event_id")); eventID != "" {
		query = query.Where("event_id = ?", eventID)
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Limit(200).Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": rows})
}

func (h *OrchestratorHandler) IngestEvent(c *gin.Context) {
	var req ingestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	source := strings.TrimSpace(req.Source)
	if source == "" {
		source = "manual"
	}
	result, err := h.ingest(source, req, c.GetString("username"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

func (h *OrchestratorHandler) WebhookIngest(c *gin.Context) {
	var req ingestRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	source := strings.TrimSpace(c.Param("source"))
	if source == "" {
		source = "webhook"
	}
	result, err := h.ingest(source, req, "webhook:"+source)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

func (h *OrchestratorHandler) ExecuteRunbook(c *gin.Context) {
	var req runbookExecRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	source := strings.TrimSpace(req.Source)
	if source == "" {
		source = "manual-runbook"
	}
	eventType := strings.TrimSpace(req.EventType)
	if eventType == "" {
		eventType = "runbook.execute"
	}
	payload := req.Payload
	if payload == nil {
		payload = map[string]interface{}{}
	}
	variables := req.Variables
	if variables == nil {
		variables = map[string]interface{}{}
	}
	request := ingestRequest{
		Source:    source,
		EventType: eventType,
		Summary:   req.Summary,
		Payload:   payload,
		Variables: variables,
	}
	result, err := h.ingest(source, request, c.GetString("username"), req.WorkflowID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

func (h *OrchestratorHandler) ingest(source string, req ingestRequest, triggerBy string, forceWorkflowID ...string) (gin.H, error) {
	payload := req.Payload
	if payload == nil {
		payload = map[string]interface{}{}
	}
	var payloadJSON string
	if raw, err := json.Marshal(payload); err == nil {
		payloadJSON = string(raw)
	}

	event := OrchestrationEvent{
		Source:     strings.TrimSpace(source),
		EventType:  strings.TrimSpace(req.EventType),
		ExternalID: strings.TrimSpace(req.External),
		Summary:    strings.TrimSpace(req.Summary),
		Payload:    payloadJSON,
		Status:     "received",
		ReceivedAt: time.Now(),
	}
	if err := h.db.Create(&event).Error; err != nil {
		return nil, err
	}

	rules := make([]OrchestrationRule, 0)
	if len(forceWorkflowID) > 0 && strings.TrimSpace(forceWorkflowID[0]) != "" {
		wf, err := h.loadWorkflow(strings.TrimSpace(forceWorkflowID[0]))
		if err != nil {
			return nil, err
		}
		rules = append(rules, OrchestrationRule{
			BaseModel:        BaseModel{ID: "adhoc-" + wf.ID},
			Name:             "即时执行",
			Source:           event.Source,
			EventType:        event.EventType,
			WorkflowID:       wf.ID,
			WorkflowName:     wf.Name,
			DefaultVariables: "",
			Enabled:          true,
		})
	} else {
		if err := h.db.Where("enabled = ?", true).Order("updated_at DESC").Find(&rules).Error; err != nil {
			return nil, err
		}
	}

	matched := make([]OrchestrationRule, 0, len(rules))
	for i := range rules {
		if h.ruleMatched(&rules[i], event.Source, event.EventType, payloadJSON) {
			matched = append(matched, rules[i])
		}
	}
	event.MatchedRule = len(matched)

	successRuns := 0
	failedRuns := 0
	dispatches := make([]OrchestrationDispatch, 0, len(matched))
	for i := range matched {
		dispatch, err := h.dispatchRule(&event, &matched[i], payload, req.Variables, triggerBy)
		if err != nil || dispatch.Status == "failed" || dispatch.Status == "skipped" {
			failedRuns++
		} else {
			successRuns++
		}
		dispatches = append(dispatches, dispatch)
	}

	event.SuccessRuns = successRuns
	event.FailedRuns = failedRuns
	switch {
	case event.MatchedRule == 0:
		event.Status = "ignored"
	case successRuns > 0 && failedRuns == 0:
		event.Status = "dispatched"
	case successRuns > 0:
		event.Status = "partial"
	default:
		event.Status = "failed"
	}
	_ = h.db.Model(&event).Updates(map[string]interface{}{
		"status":       event.Status,
		"matched_rule": event.MatchedRule,
		"success_runs": event.SuccessRuns,
		"failed_runs":  event.FailedRuns,
	}).Error

	return gin.H{
		"event":      event,
		"dispatches": dispatches,
	}, nil
}

func (h *OrchestratorHandler) dispatchRule(event *OrchestrationEvent, rule *OrchestrationRule, payload map[string]interface{}, runtimeVars map[string]interface{}, triggerBy string) (OrchestrationDispatch, error) {
	start := time.Now()
	dispatch := OrchestrationDispatch{
		EventID:      event.ID,
		RuleID:       rule.ID,
		WorkflowID:   rule.WorkflowID,
		WorkflowName: rule.WorkflowName,
		Status:       "failed",
		TriggerBy:    triggerBy,
		StartedAt:    start,
	}

	wf, err := h.loadWorkflow(rule.WorkflowID)
	if err != nil {
		dispatch.Error = err.Error()
		now := time.Now()
		dispatch.FinishedAt = &now
		_ = h.db.Create(&dispatch).Error
		h.bumpRuleCounter(rule.ID, false, err.Error(), start)
		return dispatch, err
	}
	if !wf.Enabled {
		err = errors.New("关联工作流已禁用")
		dispatch.Error = err.Error()
		dispatch.Status = "skipped"
		now := time.Now()
		dispatch.FinishedAt = &now
		_ = h.db.Create(&dispatch).Error
		h.bumpRuleCounter(rule.ID, false, err.Error(), start)
		return dispatch, err
	}

	vars := map[string]interface{}{
		"event_id":      event.ID,
		"event_source":  event.Source,
		"event_type":    event.EventType,
		"event_summary": event.Summary,
		"event_payload": payload,
	}
	mergeVariables(vars, parseVariablesJSON(rule.DefaultVariables))
	mergeVariables(vars, runtimeVars)
	exec, err := h.engine.Execute(&wf, vars, triggerBy)
	now := time.Now()
	dispatch.FinishedAt = &now
	if err != nil {
		dispatch.Error = err.Error()
		_ = h.db.Create(&dispatch).Error
		h.bumpRuleCounter(rule.ID, false, err.Error(), start)
		return dispatch, err
	}

	dispatch.ExecutionID = exec.ID
	dispatch.Status = "success"
	dispatch.Error = ""
	_ = h.db.Create(&dispatch).Error
	h.bumpRuleCounter(rule.ID, true, "", start)
	return dispatch, nil
}

func (h *OrchestratorHandler) bumpRuleCounter(ruleID string, success bool, lastError string, at time.Time) {
	if strings.TrimSpace(ruleID) == "" || strings.HasPrefix(ruleID, "adhoc-") {
		return
	}
	updates := map[string]interface{}{
		"trigger_count":     gorm.Expr("trigger_count + ?", 1),
		"last_triggered_at": at,
		"last_error":        lastError,
	}
	if !success {
		updates["failure_count"] = gorm.Expr("failure_count + ?", 1)
	}
	_ = h.db.Model(&OrchestrationRule{}).Where("id = ?", ruleID).Updates(updates).Error
}

func (h *OrchestratorHandler) ruleMatched(rule *OrchestrationRule, source, eventType, payloadJSON string) bool {
	if rule == nil || !rule.Enabled {
		return false
	}
	if !wildcardMatched(rule.Source, source) {
		return false
	}
	if !wildcardMatched(rule.EventType, eventType) {
		return false
	}
	if needle := strings.TrimSpace(rule.MatchContains); needle != "" {
		return strings.Contains(strings.ToLower(payloadJSON), strings.ToLower(needle))
	}
	return true
}

func wildcardMatched(expected, actual string) bool {
	expected = normalizeWildcardField(expected)
	actual = strings.TrimSpace(actual)
	if expected == "*" || expected == "" {
		return true
	}
	return strings.EqualFold(expected, actual)
}

func normalizeWildcardField(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "*"
	}
	return strings.ToLower(v)
}

func mergeVariables(dst map[string]interface{}, src map[string]interface{}) {
	if dst == nil || src == nil {
		return
	}
	for k, v := range src {
		dst[k] = v
	}
}

func parseVariablesJSON(raw string) map[string]interface{} {
	result := map[string]interface{}{}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return result
	}
	_ = json.Unmarshal([]byte(raw), &result)
	return result
}

func validateVariablesJSON(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var obj map[string]interface{}
	return json.Unmarshal([]byte(raw), &obj)
}

func (h *OrchestratorHandler) loadWorkflow(id string) (workflow.Workflow, error) {
	var wf workflow.Workflow
	if strings.TrimSpace(id) == "" {
		return wf, errors.New("workflow_id 不能为空")
	}
	if err := h.db.First(&wf, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return wf, errors.New("工作流不存在")
		}
		return wf, err
	}
	return wf, nil
}
