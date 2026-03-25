package orchestrator

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BaseModel struct {
	ID        string         `gorm:"primaryKey;size:36" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (b *BaseModel) BeforeCreate(tx *gorm.DB) error {
	if b.ID == "" {
		b.ID = uuid.New().String()
	}
	return nil
}

// OrchestrationRule 事件路由规则（source/event_type -> workflow）
type OrchestrationRule struct {
	BaseModel
	Name             string     `gorm:"size:128;index" json:"name"`
	Source           string     `gorm:"size:64;index" json:"source"`      // e.g. domain, k8s, cicd
	EventType        string     `gorm:"size:64;index" json:"event_type"`  // e.g. cert_expiring
	WorkflowID       string     `gorm:"size:36;index" json:"workflow_id"` // plugins/workflow.Workflow.ID
	WorkflowName     string     `gorm:"size:128" json:"workflow_name"`
	MatchContains    string     `gorm:"size:256" json:"match_contains"` // payload JSON contains
	DefaultVariables string     `gorm:"type:text" json:"default_variables"`
	Enabled          bool       `gorm:"default:true;index" json:"enabled"`
	TriggerCount     int64      `gorm:"default:0" json:"trigger_count"`
	FailureCount     int64      `gorm:"default:0" json:"failure_count"`
	LastTriggeredAt  *time.Time `json:"last_triggered_at"`
	LastError        string     `gorm:"type:text" json:"last_error"`
	CreatedBy        string     `gorm:"size:64" json:"created_by"`
}

// OrchestrationEvent 外部事件接入记录
type OrchestrationEvent struct {
	BaseModel
	Source      string    `gorm:"size:64;index" json:"source"`
	EventType   string    `gorm:"size:64;index" json:"event_type"`
	ExternalID  string    `gorm:"size:128;index" json:"external_id"`
	Summary     string    `gorm:"size:256" json:"summary"`
	Payload     string    `gorm:"type:longtext" json:"payload"`
	Status      string    `gorm:"size:32;index" json:"status"` // received/dispatched/partial/failed/ignored
	MatchedRule int       `gorm:"default:0" json:"matched_rule"`
	SuccessRuns int       `gorm:"default:0" json:"success_runs"`
	FailedRuns  int       `gorm:"default:0" json:"failed_runs"`
	ReceivedAt  time.Time `gorm:"index" json:"received_at"`
}

// OrchestrationDispatch 事件分发与执行记录
type OrchestrationDispatch struct {
	BaseModel
	EventID      string     `gorm:"size:36;index" json:"event_id"`
	RuleID       string     `gorm:"size:36;index" json:"rule_id"`
	WorkflowID   string     `gorm:"size:36;index" json:"workflow_id"`
	WorkflowName string     `gorm:"size:128" json:"workflow_name"`
	ExecutionID  string     `gorm:"size:36;index" json:"execution_id"`
	Status       string     `gorm:"size:32;index" json:"status"` // success/failed/skipped
	Error        string     `gorm:"type:text" json:"error"`
	TriggerBy    string     `gorm:"size:64" json:"trigger_by"`
	StartedAt    time.Time  `gorm:"index" json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at"`
}
