package ai

import "time"

// AIOpsIncident AI 运维闭环记录
type AIOpsIncident struct {
	BaseModel
	IncidentID         string     `gorm:"size:64;uniqueIndex" json:"incident_id"`
	Title              string     `gorm:"size:256" json:"title"`
	Query              string     `gorm:"type:text" json:"query"`
	Context            string     `gorm:"type:text" json:"context"`
	Status             string     `gorm:"size:32;index" json:"status"`
	RootCauseSummary   string     `gorm:"type:text" json:"root_cause_summary"`
	RiskLevel          string     `gorm:"size:32" json:"risk_level"`
	WorkOrderID        string     `gorm:"size:64;index" json:"workorder_id"`
	PlanJSON           string     `gorm:"type:text" json:"plan_json"`
	EvidenceJSON       string     `gorm:"type:text" json:"evidence_json"`
	RootCauseAt        *time.Time `json:"root_cause_at"`
	FirstFixActionAt   *time.Time `json:"first_fix_action_at"`
	ResolvedAt         *time.Time `json:"resolved_at"`
	MTTDSeconds        int64      `json:"mttd_seconds"`
	MTTRSeconds        int64      `json:"mttr_seconds"`
	LastPreflightScore int        `json:"last_preflight_score"`
}

// AIOpsTimelineEvent 故障时间轴事件
type AIOpsTimelineEvent struct {
	BaseModel
	IncidentID string `gorm:"size:64;index" json:"incident_id"`
	Stage      string `gorm:"size:64;index" json:"stage"`  // precheck/tool_call/llm_response/apply/verify/rollback
	Status     string `gorm:"size:32;index" json:"status"` // success/fail/pending
	Detail     string `gorm:"type:text" json:"detail"`     // 说明
	MetaJSON   string `gorm:"type:text" json:"meta_json"`  // 附加数据
	DurationMS int64  `json:"duration_ms"`                 // 步骤耗时
	Actor      string `gorm:"size:128" json:"actor"`       // 操作人/系统
}

type AIOpsDiagnoseRequest struct {
	Query       string         `json:"query" binding:"required"`
	Context     string         `json:"context"`
	ContextHint *AIContextHint `json:"context_hint"`
	IncidentID  string         `json:"incident_id"`
	Title       string         `json:"title"`
}

type AIOpsDiagnoseResponse struct {
	IncidentID      string                   `json:"incident_id"`
	Status          string                   `json:"status"`
	Reply           string                   `json:"reply"`
	ContextPack     *AIContextPack           `json:"context_pack,omitempty"`
	ToolCalls       []AIToolTrace            `json:"tool_calls,omitempty"`
	ExecutionPlan   *AIExecutionPlan         `json:"execution_plan,omitempty"`
	RelatedRunbooks []AIOpsRunbookSuggestion `json:"related_runbooks,omitempty"`
	RootCauseAt     *time.Time               `json:"root_cause_at,omitempty"`
	FirstFixAction  *time.Time               `json:"first_fix_action_at,omitempty"`
	MTTDSeconds     int64                    `json:"mttd_seconds"`
	MTTRSeconds     int64                    `json:"mttr_seconds"`
}

type AIOpsApproveRequest struct {
	IncidentID string `json:"incident_id" binding:"required"`
	Approved   bool   `json:"approved"`
	Comment    string `json:"comment"`
}

type AIOpsExecuteRequest struct {
	IncidentID string `json:"incident_id" binding:"required"`
	Stage      string `json:"stage" binding:"required"` // apply/verify/rollback
	Success    bool   `json:"success"`
	Result     string `json:"result"`
}

type AIOpsPreflightRequest struct {
	Command  string `json:"command"`
	PlanJSON string `json:"plan_json"`
	Context  string `json:"context"` // prod/staging/dev
}

type AIOpsRiskFactor struct {
	Factor string  `json:"factor"`
	Weight float64 `json:"weight"`
	Detail string  `json:"detail"`
}

type AIOpsPreflightResult struct {
	RiskScore         int               `json:"risk_score"`
	RiskFactors       []AIOpsRiskFactor `json:"risk_factors"`
	BlastRadius       string            `json:"blast_radius"`
	RecommendedTime   string            `json:"recommended_time"`
	SaferAlternative  string            `json:"safer_alternative"`
	MaintenanceWindow string            `json:"maintenance_window"`
	EscalateApproval  bool              `json:"escalate_approval"`
}

type AIOpsTimelineQuery struct {
	EvidenceFile string   `json:"evidence_file"`
	IncidentID   string   `json:"incident_id"`
	Format       string   `json:"format"` // rich/mermaid/json
	CompareFiles []string `json:"compare_files"`
}

type AIOpsIncidentDetail struct {
	Incident *AIOpsIncident       `json:"incident"`
	Events   []AIOpsTimelineEvent `json:"events"`
}

type AIOpsRunbookGenerateRequest struct {
	IncidentID string `json:"incident_id" binding:"required"`
	Title      string `json:"title"`
	Tags       string `json:"tags"`
	Category   string `json:"category"`
}

type AIOpsRunbookSuggestion struct {
	ID           uint      `json:"id"`
	Title        string    `json:"title"`
	Tags         string    `json:"tags"`
	Category     string    `json:"category"`
	Score        int       `json:"score"`
	MatchedTerms []string  `json:"matched_terms,omitempty"`
	Summary      string    `json:"summary"`
	UpdatedAt    time.Time `json:"updated_at"`
}
