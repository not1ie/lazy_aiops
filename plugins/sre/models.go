package sre

import "time"

// SreSkill 技能描述
type SreSkill struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

// SrePingResponse ping 响应
type SrePingResponse struct {
	Status  string `json:"status"`
	Version string `json:"version,omitempty"`
}

// SreBridgeStatus bridge 状态
type SreBridgeStatus struct {
	Connected bool   `json:"connected"`
	BaseURL   string `json:"base_url"`
	Message   string `json:"message,omitempty"`
}

// SreSkillRunRequest 执行 skill 请求
type SreSkillRunRequest struct {
	SkillName   string                 `json:"skill_name" binding:"required"`
	Apply       bool                   `json:"apply"`       // false=dry-run，true=实际执行
	Execute     bool                   `json:"execute"`     // 需要 apply 同时为 true 才真正跑
	AutoRollback bool                  `json:"auto_rollback"`
	Params      map[string]interface{} `json:"params"`
}

// SreSkillRunResponse 执行 skill 响应
type SreSkillRunResponse struct {
	IncidentID  string                 `json:"incident_id,omitempty"`
	Status      string                 `json:"status"`      // dry-run / running / done / failed
	Actionables []string               `json:"actionables,omitempty"`
	Evidence    map[string]interface{} `json:"evidence,omitempty"`
	Timeline    []SreTimelineEvent     `json:"timeline,omitempty"`
	Message     string                 `json:"message,omitempty"`
}

// SreTimelineEvent 时间轴事件
type SreTimelineEvent struct {
	At      time.Time `json:"at"`
	Stage   string    `json:"stage"`
	Message string    `json:"message"`
	Success bool      `json:"success"`
}

// SreDiagnoseRequest 诊断请求
type SreDiagnoseRequest struct {
	Query       string                 `json:"query" binding:"required"`
	Context     string                 `json:"context,omitempty"`
	ContextHint map[string]interface{} `json:"context_hint,omitempty"`
}

// SreDiagnoseResponse 诊断响应
type SreDiagnoseResponse struct {
	IncidentID          string        `json:"incident_id"`
	Actionables         []string      `json:"actionables"`
	ExecutionTemplates  []string      `json:"execution_templates"`
	ApprovalTicket      string        `json:"approval_ticket,omitempty"`
	Timeline            []SreTimelineEvent `json:"timeline"`
}

// SreApproveRequest 审批请求
type SreApproveRequest struct {
	IncidentID string `json:"incident_id" binding:"required"`
	Approved   bool   `json:"approved"`
	Comment    string `json:"comment"`
}

// SrePreflightRequest 风险评分请求
type SrePreflightRequest struct {
	Command string `json:"command" binding:"required"`
	Context string `json:"context,omitempty"`
}

// SrePreflightResponse 风险评分响应
type SrePreflightResponse struct {
	RiskScore   int      `json:"risk_score"`   // 0-100
	RiskLevel   string   `json:"risk_level"`   // low/medium/high/critical
	Reasons     []string `json:"reasons"`
	Blocked     bool     `json:"blocked"`      // risk_score >= 70 时 true
	Suggestion  string   `json:"suggestion,omitempty"`
}

// SreRunbookItem runbook 条目
type SreRunbookItem struct {
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Description string    `json:"description,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SreSloStatus SLO 状态
type SreSloStatus struct {
	Name       string  `json:"name"`
	Target     float64 `json:"target"`
	Current    float64 `json:"current"`
	BurnRate1h float64 `json:"burn_rate_1h"`
	Status     string  `json:"status"` // ok / warning / critical
}
