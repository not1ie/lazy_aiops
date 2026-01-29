package monitor

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

// DomainMonitor 域名监控
type DomainMonitor struct {
	BaseModel
	Domain      string     `gorm:"size:256;index" json:"domain"`
	Type        string     `gorm:"size:32" json:"type"` // http, https, tcp
	Port        int        `gorm:"default:443" json:"port"`
	Interval    int        `gorm:"default:60" json:"interval"` // 检测间隔(秒)
	Timeout     int        `gorm:"default:10" json:"timeout"`
	Status      int        `gorm:"default:1" json:"status"`       // 1:正常 0:异常
	Enabled     bool       `gorm:"default:true" json:"enabled"`
	SSLExpire   *time.Time `json:"ssl_expire"`
	LastCheck   *time.Time `json:"last_check"`
	ResponseTime int       `json:"response_time"` // 响应时间(ms)
}

// AlertRule 告警规则
type AlertRule struct {
	BaseModel
	Name        string `gorm:"size:128" json:"name"`
	Type        string `gorm:"size:64" json:"type"`    // domain, host, custom
	Target      string `gorm:"size:256" json:"target"` // 监控目标ID
	Condition   string `gorm:"size:256" json:"condition"`
	Threshold   string `gorm:"size:64" json:"threshold"`
	Severity    string `gorm:"size:32" json:"severity"` // critical, warning, info
	NotifyType  string `gorm:"size:64" json:"notify_type"` // webhook, email, sms
	NotifyURL   string `gorm:"size:512" json:"notify_url"`
	Enabled     bool   `gorm:"default:true" json:"enabled"`
}

// AlertRecord 告警记录
type AlertRecord struct {
	BaseModel
	RuleID      string     `gorm:"size:36;index" json:"rule_id"`
	RuleName    string     `gorm:"size:128" json:"rule_name"`
	Target      string     `gorm:"size:256" json:"target"`
	Content     string     `gorm:"type:text" json:"content"`
	Severity    string     `gorm:"size:32" json:"severity"`
	Status      int        `gorm:"default:0" json:"status"` // 0:未处理 1:已处理 2:已忽略
	ResolvedAt  *time.Time `json:"resolved_at"`
	ResolvedBy  string     `gorm:"size:64" json:"resolved_by"`
}

// MetricRecord 指标记录
type MetricRecord struct {
	BaseModel
	Timestamp   time.Time `gorm:"index" json:"timestamp"`
	CPUUsage    float64   `json:"cpu_usage"`
	MemoryUsage float64   `json:"memory_usage"`
	DiskUsage   float64   `json:"disk_usage"`
	NetworkIn   uint64    `json:"network_in"`
	NetworkOut  uint64    `json:"network_out"`
}
