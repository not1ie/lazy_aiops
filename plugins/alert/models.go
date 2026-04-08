package alert

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

// AlertRule 告警规则
type AlertRule struct {
	BaseModel
	Name          string `gorm:"size:128" json:"name"`
	Type          string `gorm:"size:64" json:"type"`     // host, k8s, domain, ssl, custom
	Target        string `gorm:"size:256" json:"target"`  // 监控目标
	Metric        string `gorm:"size:128" json:"metric"`  // cpu, memory, disk, http_status
	Operator      string `gorm:"size:16" json:"operator"` // >, <, >=, <=, ==, !=
	Threshold     string `gorm:"size:64" json:"threshold"`
	Duration      int    `json:"duration"`                // 持续时间(秒)
	Severity      string `gorm:"size:32" json:"severity"` // critical, warning, info
	NotifyGroupID string `gorm:"size:36" json:"notify_group_id"`
	Enabled       bool   `gorm:"default:true" json:"enabled"`
	// AI增强
	AIAnalysis    bool   `gorm:"default:false" json:"ai_analysis"`  // 是否启用AI分析
	AutoRecover   bool   `gorm:"default:false" json:"auto_recover"` // 是否自动恢复
	RecoverScript string `gorm:"type:text" json:"recover_script"`   // 自动恢复脚本
	Description   string `gorm:"size:512" json:"description"`
}

// Alert 告警事件
type Alert struct {
	BaseModel
	RuleID      string     `gorm:"size:36;index" json:"rule_id"`
	RuleName    string     `gorm:"size:128" json:"rule_name"`
	Fingerprint string     `gorm:"size:64;index" json:"fingerprint"` // 告警指纹，用于去重
	Target      string     `gorm:"size:256" json:"target"`
	Metric      string     `gorm:"size:128" json:"metric"`
	Value       string     `gorm:"size:64" json:"value"`
	Threshold   string     `gorm:"size:64" json:"threshold"`
	Severity    string     `gorm:"size:32;index" json:"severity"`
	Status      int        `gorm:"default:0;index" json:"status"` // 0:触发 1:已确认 2:已恢复 3:已抑制
	FiredAt     time.Time  `gorm:"index" json:"fired_at"`
	ResolvedAt  *time.Time `json:"resolved_at"`
	AckedAt     *time.Time `json:"acked_at"`
	AckedBy     string     `gorm:"size:64" json:"acked_by"`
	// AI分析结果
	AIAnalysis   string `gorm:"type:text" json:"ai_analysis"`
	AISuggestion string `gorm:"type:text" json:"ai_suggestion"`
	// 聚合信息
	GroupKey    string `gorm:"size:128;index" json:"group_key"` // 聚合键
	Count       int    `gorm:"default:1" json:"count"`          // 聚合计数
	Labels      string `gorm:"type:text" json:"labels"`         // 标签JSON
	Annotations string `gorm:"type:text" json:"annotations"`    // 注解JSON
	// 闭环联动
	WorkOrderID         string     `gorm:"size:36;index" json:"work_order_id"`
	WorkflowID          string     `gorm:"size:36;index" json:"workflow_id"`
	WorkflowExecutionID string     `gorm:"size:36;index" json:"workflow_execution_id"`
	LinkedAt            *time.Time `json:"linked_at"`
	StatusReason        string     `gorm:"size:512" json:"status_reason"`
}

// AlertSilence 告警静默
type AlertSilence struct {
	BaseModel
	Name      string    `gorm:"size:128" json:"name"`
	Matchers  string    `gorm:"type:text" json:"matchers"` // 匹配规则JSON
	StartsAt  time.Time `json:"starts_at"`
	EndsAt    time.Time `json:"ends_at"`
	CreatedBy string    `gorm:"size:64" json:"created_by"`
	Comment   string    `gorm:"size:512" json:"comment"`
	Status    int       `gorm:"default:1" json:"status"` // 1:活跃 0:过期
}

// AlertAggregation 告警聚合配置
type AlertAggregation struct {
	BaseModel
	Name        string `gorm:"size:128" json:"name"`
	GroupBy     string `gorm:"size:256" json:"group_by"` // 聚合字段,逗号分隔
	Interval    int    `json:"interval"`                 // 聚合时间窗口(秒)
	Enabled     bool   `gorm:"default:true" json:"enabled"`
	Description string `gorm:"size:512" json:"description"`
}

// AlertHistory 告警历史(用于AI分析)
type AlertHistory struct {
	BaseModel
	AlertID    string     `gorm:"size:36;index" json:"alert_id"`
	RuleID     string     `gorm:"size:36;index" json:"rule_id"`
	Target     string     `gorm:"size:256" json:"target"`
	Severity   string     `gorm:"size:32" json:"severity"`
	FiredAt    time.Time  `json:"fired_at"`
	ResolvedAt *time.Time `json:"resolved_at"`
	Duration   int        `json:"duration"`                    // 持续时间(秒)
	Resolution string     `gorm:"size:256" json:"resolution"`  // 解决方式
	RootCause  string     `gorm:"type:text" json:"root_cause"` // 根因
}
