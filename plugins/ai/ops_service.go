package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/lazyautoops/lazy-auto-ops/plugins/workorder"
	"gorm.io/gorm"
)

const lazySREEnginePrompt = `# Role: LazySRE Autonomous Engine
You are a Senior Principal Site Reliability Engineer (L7) with expertise in Kubernetes, Linux Internals, and Distributed Systems.

# Core Philosophy:
1. Evidence First: Never guess. Use provided tools first.
2. First Principles: Network -> Infrastructure -> Resource -> Application -> External dependency.
3. Safety First: Destructive action MUST include risk assessment.
4. No Fluff: Keep concise and technical.

# Response Format:
- Status: [Investigating / Diagnosing / Proposing Fix / Monitoring]
- Reasoning: one-line
- Commands: executable commands
- Risk Level: [Low/Medium/High/Critical] + short reason`

func (s *AIService) DiagnoseIncident(req *AIOpsDiagnoseRequest, userID string) (*AIOpsDiagnoseResponse, error) {
	if req == nil || strings.TrimSpace(req.Query) == "" {
		return nil, fmt.Errorf("query 不能为空")
	}
	now := time.Now()
	incidentID := strings.TrimSpace(req.IncidentID)
	if incidentID == "" {
		incidentID = "CHG-" + now.Format("20060102-150405")
	}

	incident, err := s.getOrCreateIncident(incidentID, req, now)
	if err != nil {
		return nil, err
	}

	precheckStart := time.Now()
	snapshot := s.RuntimeSnapshot()
	precheckDetail := fmt.Sprintf("provider=%s model=%s auth=%s has_api_key=%t", snapshot.Provider, snapshot.Model, snapshot.AuthType, snapshot.HasAPIKey)
	s.logTimeline(incidentID, "precheck", "success", precheckDetail, sinceMS(precheckStart), map[string]interface{}{
		"runtime": snapshot,
	}, "system")

	pack, err := s.BuildContextPack(req.ContextHint)
	if err != nil {
		return nil, err
	}

	chatReq := &ChatRequest{
		Message:     req.Query,
		Context:     req.Context,
		AutoContext: true,
		ContextHint: req.ContextHint,
	}
	traces, toolContext := s.maybeUseTools(chatReq, pack, nil)
	for _, trace := range traces {
		status := normalizeEventStatus(trace.Status)
		s.logTimeline(incidentID, "tool_call", status, fmt.Sprintf("%s: %s", trace.Name, trace.Summary), 0, trace, "system")
	}

	replyStart := time.Now()
	reply, _, err := s.core.AI.CallLLM(lazySREEnginePrompt, []map[string]string{
		{
			"role": "user",
			"content": strings.Join([]string{
				"用户问题:",
				req.Query,
				"",
				"上下文:",
				nonEmptyContext(req.Context, "无"),
				"",
				"环境摘要:",
				func() string {
					if pack == nil {
						return "无"
					}
					return pack.Summary
				}(),
				"",
				"工具证据:",
				nonEmptyContext(toolContext, "无"),
			}, "\n"),
		},
	})
	if err != nil {
		s.logTimeline(incidentID, "llm_response", "fail", err.Error(), sinceMS(replyStart), nil, "llm")
		return nil, err
	}
	s.logTimeline(incidentID, "llm_response", "success", truncateText(reply, 400), sinceMS(replyStart), nil, "llm")

	plan, err := s.buildExecutionPlan(chatReq, pack, traces, reply)
	if err != nil {
		return nil, err
	}

	rootCauseAt := time.Now()
	incident.RootCauseAt = &rootCauseAt
	incident.Status = "diagnosed"
	incident.RootCauseSummary = truncateText(reply, 1000)
	if plan != nil {
		incident.RiskLevel = strings.TrimSpace(strings.ToLower(plan.RiskLevel))
	}
	incident.MTTDSeconds = int64(rootCauseAt.Sub(incident.CreatedAt).Seconds())
	if incident.MTTDSeconds < 0 {
		incident.MTTDSeconds = 0
	}
	incident.PlanJSON = mustMarshalString(plan)
	incident.EvidenceJSON = mustMarshalString(map[string]interface{}{
		"context_pack": pack,
		"tool_calls":   traces,
		"reply":        reply,
		"query":        req.Query,
		"context":      req.Context,
	})
	if err := s.db.Save(incident).Error; err != nil {
		return nil, err
	}

	return &AIOpsDiagnoseResponse{
		IncidentID:     incidentID,
		Status:         incident.Status,
		Reply:          reply,
		ContextPack:    pack,
		ToolCalls:      traces,
		ExecutionPlan:  plan,
		RootCauseAt:    incident.RootCauseAt,
		FirstFixAction: incident.FirstFixActionAt,
		MTTDSeconds:    incident.MTTDSeconds,
		MTTRSeconds:    incident.MTTRSeconds,
	}, nil
}

func (s *AIService) ApproveIncident(req *AIOpsApproveRequest, userID, username string) (*workorder.WorkOrder, error) {
	if req == nil || strings.TrimSpace(req.IncidentID) == "" {
		return nil, fmt.Errorf("incident_id 不能为空")
	}
	incident, err := s.getIncident(req.IncidentID)
	if err != nil {
		return nil, err
	}
	if !req.Approved {
		incident.Status = "rejected"
		if err := s.db.Save(incident).Error; err != nil {
			return nil, err
		}
		s.logTimeline(incident.IncidentID, "apply", "fail", nonEmptyContext(req.Comment, "审批拒绝"), 0, nil, username)
		return nil, nil
	}

	var plan AIExecutionPlan
	if strings.TrimSpace(incident.PlanJSON) == "" {
		return nil, fmt.Errorf("incident 未生成可审批执行计划")
	}
	if err := json.Unmarshal([]byte(incident.PlanJSON), &plan); err != nil {
		return nil, fmt.Errorf("解析执行计划失败: %w", err)
	}
	if !plan.NeedApproval {
		return nil, fmt.Errorf("当前计划不需要审批")
	}

	formData := mustMarshalString(map[string]interface{}{
		"source":     "ai_ops_incident",
		"incident":   incident.IncidentID,
		"plan":       plan,
		"created_by": username,
	})
	order, err := workorder.CreateOrderWithDefaults(s.db, workorder.CreateOrderInput{
		TypeCode:     nonEmptyContext(plan.WorkOrderTypeCode, "change_apply"),
		Title:        nonEmptyContext(plan.Title, incident.Title),
		Content:      nonEmptyContext(plan.Summary, incident.Query),
		FormData:     formData,
		Priority:     priorityFromRisk(plan.RiskLevel),
		Submitter:    username,
		SubmitterID:  userID,
		AISuggestion: incident.RootCauseSummary,
		AIRisk:       nonEmptyContext(plan.RiskLevel, "medium"),
	})
	if err != nil {
		return nil, err
	}

	incident.WorkOrderID = order.ID
	incident.Status = "approved"
	if err := s.db.Save(incident).Error; err != nil {
		return nil, err
	}
	s.logTimeline(incident.IncidentID, "apply", "pending", "审批通过，工单已创建: "+order.ID, 0, nil, username)
	return order, nil
}

func (s *AIService) MarkIncidentStage(req *AIOpsExecuteRequest, actor string) (*AIOpsIncident, error) {
	if req == nil || strings.TrimSpace(req.IncidentID) == "" {
		return nil, fmt.Errorf("incident_id 不能为空")
	}
	stage := strings.TrimSpace(strings.ToLower(req.Stage))
	if stage != "apply" && stage != "verify" && stage != "rollback" {
		return nil, fmt.Errorf("stage 仅支持 apply/verify/rollback")
	}
	incident, err := s.getIncident(req.IncidentID)
	if err != nil {
		return nil, err
	}

	status := "fail"
	if req.Success {
		status = "success"
	}
	s.logTimeline(incident.IncidentID, stage, status, nonEmptyContext(req.Result, "-"), 0, nil, actor)

	now := time.Now()
	if stage == "apply" && req.Success && incident.FirstFixActionAt == nil {
		incident.FirstFixActionAt = &now
		incident.Status = "executing"
	}
	if stage == "verify" && req.Success {
		incident.Status = "resolved"
		incident.ResolvedAt = &now
		if incident.RootCauseAt != nil {
			incident.MTTRSeconds = int64(now.Sub(*incident.RootCauseAt).Seconds())
		} else if incident.FirstFixActionAt != nil {
			incident.MTTRSeconds = int64(now.Sub(*incident.FirstFixActionAt).Seconds())
		}
		if incident.MTTRSeconds < 0 {
			incident.MTTRSeconds = 0
		}
	}
	if stage == "rollback" && req.Success {
		incident.Status = "rolled_back"
	}
	if err := s.db.Save(incident).Error; err != nil {
		return nil, err
	}
	return incident, nil
}

func (s *AIService) PreflightRisk(req *AIOpsPreflightRequest) (*AIOpsPreflightResult, error) {
	if req == nil {
		return nil, fmt.Errorf("empty preflight request")
	}
	command := strings.TrimSpace(req.Command)
	plan := strings.TrimSpace(req.PlanJSON)
	context := strings.TrimSpace(strings.ToLower(req.Context))
	if context == "" {
		context = "prod"
	}
	if command == "" && plan == "" {
		return nil, fmt.Errorf("command 或 plan_json 至少提供一个")
	}

	var successCnt, totalCnt int64
	s.db.Model(&AIOpsTimelineEvent{}).Where("stage = ?", "verify").Count(&totalCnt)
	s.db.Model(&AIOpsTimelineEvent{}).Where("stage = ? AND status = ?", "verify", "success").Count(&successCnt)
	successRate := 0.0
	if totalCnt > 0 {
		successRate = float64(successCnt) / float64(totalCnt)
	}

	baseScore := 25
	riskFactors := make([]AIOpsRiskFactor, 0, 5)
	if context == "prod" {
		baseScore += 20
		riskFactors = append(riskFactors, AIOpsRiskFactor{Factor: "生产环境", Weight: 0.22, Detail: "目标上下文为 prod"})
	}
	rawText := strings.ToLower(command + "\n" + plan)
	if containsAny(rawText, []string{"delete", "drop", "truncate", "scale", "restart", "rollback", "patch", "kill"}) {
		baseScore += 30
		riskFactors = append(riskFactors, AIOpsRiskFactor{Factor: "包含高风险动作", Weight: 0.35, Detail: "命令涉及重启/删除/回滚/扩缩容"})
	}
	if successRate > 0 {
		if successRate < 0.6 {
			baseScore += 25
			riskFactors = append(riskFactors, AIOpsRiskFactor{Factor: "历史成功率偏低", Weight: 0.25, Detail: fmt.Sprintf("最近验证成功率 %.0f%%", successRate*100)})
		} else {
			baseScore -= 10
			riskFactors = append(riskFactors, AIOpsRiskFactor{Factor: "历史成功率较高", Weight: -0.10, Detail: fmt.Sprintf("最近验证成功率 %.0f%%", successRate*100)})
		}
	}
	if len(strings.Fields(command)) > 10 {
		baseScore += 8
		riskFactors = append(riskFactors, AIOpsRiskFactor{Factor: "命令复杂度偏高", Weight: 0.08, Detail: "命令参数较多，建议分步执行"})
	}
	if baseScore < 0 {
		baseScore = 0
	}
	if baseScore > 100 {
		baseScore = 100
	}

	assistant, _, err := s.core.AI.CallLLM(
		"你是变更风险评估器，请只返回 JSON。",
		[]map[string]string{
			{
				"role": "user",
				"content": fmt.Sprintf(`请根据以下信息做风险评估并输出 JSON:
context=%s
command=%s
plan=%s
history_success_rate=%.2f
输出格式:
{"blast_radius":"...","recommended_time":"...","safer_alternative":"..."}`,
					context, nonEmptyContext(command, "-"), nonEmptyContext(plan, "-"), successRate),
			},
		},
	)
	extra := map[string]string{
		"blast_radius":      "可能影响目标服务实例与上游调用链",
		"recommended_time":  "建议在维护窗口低峰执行",
		"safer_alternative": "先做只读探测，再灰度执行最小变更",
	}
	if err == nil {
		if obj := extractJSONObject(assistant); obj != "" {
			_ = json.Unmarshal([]byte(obj), &extra)
		}
	}

	return &AIOpsPreflightResult{
		RiskScore:         baseScore,
		RiskFactors:       riskFactors,
		BlastRadius:       strings.TrimSpace(extra["blast_radius"]),
		RecommendedTime:   strings.TrimSpace(extra["recommended_time"]),
		SaferAlternative:  strings.TrimSpace(extra["safer_alternative"]),
		MaintenanceWindow: "建议配置 maintenance_window，默认按低峰策略",
		EscalateApproval:  baseScore >= 70,
	}, nil
}

func (s *AIService) BuildTimeline(query *AIOpsTimelineQuery) (map[string]interface{}, error) {
	q := &AIOpsTimelineQuery{}
	if query != nil {
		*q = *query
	}
	format := strings.TrimSpace(strings.ToLower(q.Format))
	if format == "" {
		format = "rich"
	}

	events := make([]AIOpsTimelineEvent, 0)
	source := "db"
	incidentID := strings.TrimSpace(q.IncidentID)
	var incident AIOpsIncident
	if q.EvidenceFile != "" {
		source = q.EvidenceFile
		loaded, loadIncidentID, err := loadTimelineFromFile(q.EvidenceFile)
		if err != nil {
			return nil, err
		}
		events = loaded
		if incidentID == "" {
			incidentID = loadIncidentID
		}
	} else {
		if incidentID == "" {
			return nil, fmt.Errorf("incident_id 不能为空（未指定 evidence_file 时）")
		}
		if err := s.db.Where("incident_id = ?", incidentID).First(&incident).Error; err != nil {
			return nil, err
		}
		if err := s.db.Where("incident_id = ?", incidentID).Order("created_at asc").Find(&events).Error; err != nil {
			return nil, err
		}
	}

	markers := buildTimelineMarkers(events, &incident)
	result := map[string]interface{}{
		"incident_id": incidentID,
		"source":      source,
		"events":      events,
		"markers":     markers,
	}
	switch format {
	case "json":
		// keep raw events only
	case "mermaid":
		result["timeline"] = buildMermaidTimeline(incidentID, events, markers)
	default:
		result["timeline"] = buildRichTimeline(incidentID, events, markers)
	}

	if len(q.CompareFiles) > 0 {
		result["compare"] = buildTimelineCompareSummary(q.CompareFiles)
	}
	return result, nil
}

func (s *AIService) getOrCreateIncident(incidentID string, req *AIOpsDiagnoseRequest, now time.Time) (*AIOpsIncident, error) {
	var incident AIOpsIncident
	err := s.db.Where("incident_id = ?", incidentID).First(&incident).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = truncateText(strings.TrimSpace(req.Query), 80)
	}
	if incident.ID == "" {
		incident = AIOpsIncident{
			IncidentID: incidentID,
			Title:      title,
			Query:      strings.TrimSpace(req.Query),
			Context:    strings.TrimSpace(req.Context),
			Status:     "diagnosing",
		}
		if err := s.db.Create(&incident).Error; err != nil {
			return nil, err
		}
	} else {
		incident.Query = strings.TrimSpace(req.Query)
		incident.Context = strings.TrimSpace(req.Context)
		incident.Title = title
		incident.Status = "diagnosing"
		if err := s.db.Save(&incident).Error; err != nil {
			return nil, err
		}
	}
	return &incident, nil
}

func (s *AIService) getIncident(incidentID string) (*AIOpsIncident, error) {
	var incident AIOpsIncident
	if err := s.db.Where("incident_id = ?", incidentID).First(&incident).Error; err != nil {
		return nil, err
	}
	return &incident, nil
}

func (s *AIService) logTimeline(incidentID, stage, status, detail string, durationMS int64, meta interface{}, actor string) {
	event := AIOpsTimelineEvent{
		IncidentID: strings.TrimSpace(incidentID),
		Stage:      strings.TrimSpace(stage),
		Status:     normalizeEventStatus(status),
		Detail:     strings.TrimSpace(detail),
		DurationMS: durationMS,
		Actor:      strings.TrimSpace(actor),
	}
	if meta != nil {
		event.MetaJSON = mustMarshalString(meta)
	}
	_ = s.db.Create(&event).Error
}

func buildTimelineMarkers(events []AIOpsTimelineEvent, incident *AIOpsIncident) map[string]interface{} {
	out := map[string]interface{}{}
	var rootCauseAt *time.Time
	var firstFixAt *time.Time
	for i := range events {
		ts := events[i].CreatedAt
		if rootCauseAt == nil && events[i].Stage == "llm_response" && events[i].Status == "success" {
			rootCauseAt = &ts
		}
		if firstFixAt == nil && events[i].Stage == "apply" && events[i].Status == "success" {
			firstFixAt = &ts
		}
	}
	if incident != nil {
		if incident.RootCauseAt != nil {
			rootCauseAt = incident.RootCauseAt
		}
		if incident.FirstFixActionAt != nil {
			firstFixAt = incident.FirstFixActionAt
		}
	}
	if rootCauseAt != nil {
		out["root_cause_at"] = *rootCauseAt
	}
	if firstFixAt != nil {
		out["first_fix_action_at"] = *firstFixAt
	}
	if rootCauseAt != nil && firstFixAt != nil {
		out["mttd_seconds"] = int64(firstFixAt.Sub(*rootCauseAt).Seconds())
		if out["mttd_seconds"].(int64) < 0 {
			out["mttd_seconds"] = int64(0)
		}
	}
	if incident != nil {
		out["incident_status"] = incident.Status
		out["mttd_persisted_seconds"] = incident.MTTDSeconds
		out["mttr_persisted_seconds"] = incident.MTTRSeconds
	}
	return out
}

func buildRichTimeline(incidentID string, events []AIOpsTimelineEvent, markers map[string]interface{}) string {
	lines := []string{
		fmt.Sprintf("Incident: %s", incidentID),
		"========================================",
	}
	for _, event := range events {
		lines = append(lines, fmt.Sprintf(
			"%s  %-11s  %-7s  %5dms  %s",
			event.CreatedAt.Format("2006-01-02 15:04:05"),
			event.Stage,
			strings.ToUpper(event.Status),
			event.DurationMS,
			event.Detail,
		))
	}
	lines = append(lines, "----------------------------------------")
	if v, ok := markers["root_cause_at"]; ok {
		lines = append(lines, fmt.Sprintf("Root cause inferred at: %v", v))
	}
	if v, ok := markers["first_fix_action_at"]; ok {
		lines = append(lines, fmt.Sprintf("First fix action at: %v", v))
	}
	if v, ok := markers["mttd_persisted_seconds"]; ok {
		lines = append(lines, fmt.Sprintf("MTTD: %vs", v))
	}
	if v, ok := markers["mttr_persisted_seconds"]; ok {
		lines = append(lines, fmt.Sprintf("MTTR: %vs", v))
	}
	return strings.Join(lines, "\n")
}

func buildMermaidTimeline(incidentID string, events []AIOpsTimelineEvent, markers map[string]interface{}) string {
	lines := []string{
		"sequenceDiagram",
		fmt.Sprintf("    participant U as \"%s\"", incidentID),
		"    participant S as \"LazyAIOps\"",
	}
	for _, event := range events {
		label := strings.ReplaceAll(truncateText(event.Detail, 60), "\"", "'")
		lines = append(lines, fmt.Sprintf("    U->>S: %s [%s]", event.Stage, strings.ToUpper(event.Status)))
		lines = append(lines, fmt.Sprintf("    Note right of S: %s", label))
	}
	if v, ok := markers["root_cause_at"]; ok {
		lines = append(lines, fmt.Sprintf("    Note over S: root_cause_at=%v", v))
	}
	if v, ok := markers["first_fix_action_at"]; ok {
		lines = append(lines, fmt.Sprintf("    Note over S: first_fix_action_at=%v", v))
	}
	if v, ok := markers["mttd_persisted_seconds"]; ok {
		lines = append(lines, fmt.Sprintf("    Note over S: MTTD=%vs", v))
	}
	if v, ok := markers["mttr_persisted_seconds"]; ok {
		lines = append(lines, fmt.Sprintf("    Note over S: MTTR=%vs", v))
	}
	return strings.Join(lines, "\n")
}

func buildTimelineCompareSummary(files []string) map[string]interface{} {
	type fileStats struct {
		File        string         `json:"file"`
		IncidentID  string         `json:"incident_id"`
		TotalEvents int            `json:"total_events"`
		ByStage     map[string]int `json:"by_stage"`
	}
	items := make([]fileStats, 0, len(files))
	for _, file := range files {
		rows, incidentID, err := loadTimelineFromFile(file)
		if err != nil {
			items = append(items, fileStats{
				File: file,
				ByStage: map[string]int{
					"error": 1,
				},
			})
			continue
		}
		stageCount := map[string]int{}
		for _, row := range rows {
			stageCount[row.Stage]++
		}
		items = append(items, fileStats{
			File:        file,
			IncidentID:  incidentID,
			TotalEvents: len(rows),
			ByStage:     stageCount,
		})
	}
	return map[string]interface{}{"files": items}
}

func loadTimelineFromFile(path string) ([]AIOpsTimelineEvent, string, error) {
	clean := filepath.Clean(strings.TrimSpace(path))
	body, err := os.ReadFile(clean)
	if err != nil {
		return nil, "", err
	}
	type wrapper struct {
		IncidentID string               `json:"incident_id"`
		Events     []AIOpsTimelineEvent `json:"events"`
	}
	obj := wrapper{}
	if err := json.Unmarshal(body, &obj); err == nil && len(obj.Events) > 0 {
		sort.Slice(obj.Events, func(i, j int) bool { return obj.Events[i].CreatedAt.Before(obj.Events[j].CreatedAt) })
		return obj.Events, obj.IncidentID, nil
	}
	rows := make([]AIOpsTimelineEvent, 0)
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, "", err
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].CreatedAt.Before(rows[j].CreatedAt) })
	incidentID := ""
	if len(rows) > 0 {
		incidentID = rows[0].IncidentID
	}
	return rows, incidentID, nil
}

func sinceMS(start time.Time) int64 {
	if start.IsZero() {
		return 0
	}
	return int64(time.Since(start) / time.Millisecond)
}

func mustMarshalString(v interface{}) string {
	if v == nil {
		return ""
	}
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(data)
}

func normalizeEventStatus(status string) string {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "success":
		return "success"
	case "fail", "failed", "error":
		return "fail"
	case "pending":
		return "pending"
	default:
		return "pending"
	}
}

func containsAny(raw string, keys []string) bool {
	for _, key := range keys {
		if strings.Contains(raw, strings.ToLower(strings.TrimSpace(key))) {
			return true
		}
	}
	return false
}

func nonEmptyContext(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func truncateText(value string, maxLen int) string {
	value = strings.TrimSpace(value)
	if maxLen <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= maxLen {
		return value
	}
	return string(runes[:maxLen]) + "..."
}
