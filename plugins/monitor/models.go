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

// MonitorSetting 监控配置
type MonitorSetting struct {
	BaseModel
	PrometheusURL  string `gorm:"size:256" json:"prometheus_url"`
	PushgatewayURL string `gorm:"size:256" json:"pushgateway_url"`
	AuthType       string `gorm:"size:16" json:"auth_type"` // none/basic/bearer
	Username       string `gorm:"size:128" json:"username"`
	Password       string `gorm:"size:256" json:"password"`
	Token          string `gorm:"size:512" json:"token"`
}

// PromQueryHistory Prometheus查询历史
type PromQueryHistory struct {
	BaseModel
	Mode      string `gorm:"size:16" json:"mode"` // instant/range
	Query     string `gorm:"type:text" json:"query"`
	Start     string `gorm:"size:32" json:"start"`
	End       string `gorm:"size:32" json:"end"`
	Step      string `gorm:"size:32" json:"step"`
	CreatedBy string `gorm:"size:64" json:"created_by"`
	Name      string `gorm:"size:128" json:"name"`
	Favorite  bool   `gorm:"default:false" json:"favorite"`
}

// AgentHeartbeat 采集器心跳
type AgentHeartbeat struct {
	BaseModel
	AgentID   string `gorm:"size:64;uniqueIndex" json:"agent_id"`
	Hostname  string `gorm:"size:128" json:"hostname"`
	IP        string `gorm:"size:64" json:"ip"`
	Version   string `gorm:"size:64" json:"version"`
	OS        string `gorm:"size:128" json:"os"`
	Labels    string `gorm:"type:text" json:"labels"`
	Meta      string `gorm:"type:text" json:"meta"`
	CPU       float64 `json:"cpu"`
	Memory    float64 `json:"memory"`
	Disk      float64 `json:"disk"`
	NetIn     float64 `json:"net_in"`
	NetOut    float64 `json:"net_out"`
	Status    string `gorm:"size:16" json:"status"`
	LastSeen  time.Time `gorm:"index" json:"last_seen"`
}

// AgentHeartbeatRecord 心跳历史
type AgentHeartbeatRecord struct {
	BaseModel
	AgentID  string    `gorm:"size:64;index" json:"agent_id"`
	Timestamp time.Time `gorm:"index" json:"timestamp"`
	Labels   string    `gorm:"type:text" json:"labels"`
	Meta     string    `gorm:"type:text" json:"meta"`
}
