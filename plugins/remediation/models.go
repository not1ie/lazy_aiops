package remediation

import (
	"time"

	"gorm.io/gorm"
)

// RemediationLog 故障自愈日志
type RemediationLog struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	AlertID     string         `gorm:"size:36;index" json:"alert_id"`
	RuleID      string         `gorm:"size:36;index" json:"rule_id"`
	Target      string         `gorm:"size:256" json:"target"`
	Action      string         `gorm:"type:text" json:"action"`      // 执行的脚本
	Status      string         `gorm:"size:32" json:"status"`       // success, failed, running
	Stdout      string         `gorm:"type:text" json:"stdout"`
	Stderr      string         `gorm:"type:text" json:"stderr"`
	Error       string         `gorm:"type:text" json:"error"`
	StartedAt   time.Time      `json:"started_at"`
	FinishedAt  *time.Time     `json:"finished_at"`
	Duration    int            `json:"duration"` // 秒
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}
