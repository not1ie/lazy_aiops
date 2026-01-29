package firewall

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

// Firewall 防火墙设备
type Firewall struct {
	BaseModel
	Name          string     `gorm:"size:128" json:"name"`
	Vendor        string     `gorm:"size:64" json:"vendor"`  // huawei, cisco, fortinet, paloalto
	Model         string     `gorm:"size:64" json:"model"`
	IP            string     `gorm:"size:64;index" json:"ip"`
	ManagePort    int        `gorm:"default:443" json:"manage_port"`
	SNMPVersion   string     `gorm:"size:16;default:v2c" json:"snmp_version"` // v1, v2c, v3
	SNMPCommunity string     `gorm:"size:128" json:"snmp_community"`
	SNMPPort      int        `gorm:"default:161" json:"snmp_port"`
	// SNMPv3
	SNMPUser       string `gorm:"size:64" json:"snmp_user"`
	SNMPAuthProto  string `gorm:"size:16" json:"snmp_auth_proto"`  // MD5, SHA
	SNMPAuthPass   string `gorm:"size:128" json:"-"`
	SNMPPrivProto  string `gorm:"size:16" json:"snmp_priv_proto"`  // DES, AES
	SNMPPrivPass   string `gorm:"size:128" json:"-"`
	Status         int    `gorm:"default:1" json:"status"` // 1:在线 0:离线 2:告警
	LastCheckAt    *time.Time `json:"last_check_at"`
	CPUUsage       float64 `json:"cpu_usage"`
	MemoryUsage    float64 `json:"memory_usage"`
	SessionCount   int64   `json:"session_count"`
	Throughput     int64   `json:"throughput"` // bps
	Description    string  `gorm:"size:512" json:"description"`
}

// FirewallRule 防火墙规则
type FirewallRule struct {
	BaseModel
	FirewallID  string `gorm:"size:36;index" json:"firewall_id"`
	Name        string `gorm:"size:128" json:"name"`
	Priority    int    `json:"priority"`
	Action      string `gorm:"size:16" json:"action"` // allow, deny
	SrcZone     string `gorm:"size:64" json:"src_zone"`
	DstZone     string `gorm:"size:64" json:"dst_zone"`
	SrcAddr     string `gorm:"size:256" json:"src_addr"`
	DstAddr     string `gorm:"size:256" json:"dst_addr"`
	Service     string `gorm:"size:128" json:"service"`
	Protocol    string `gorm:"size:16" json:"protocol"`
	SrcPort     string `gorm:"size:64" json:"src_port"`
	DstPort     string `gorm:"size:64" json:"dst_port"`
	Enabled     bool   `gorm:"default:true" json:"enabled"`
	HitCount    int64  `json:"hit_count"`
	Description string `gorm:"size:512" json:"description"`
}

// SNMPMetric SNMP采集指标
type SNMPMetric struct {
	BaseModel
	FirewallID   string    `gorm:"size:36;index" json:"firewall_id"`
	MetricType   string    `gorm:"size:64;index" json:"metric_type"` // cpu, memory, session, throughput, interface
	MetricName   string    `gorm:"size:128" json:"metric_name"`
	Value        float64   `json:"value"`
	Unit         string    `gorm:"size:32" json:"unit"`
	CollectedAt  time.Time `gorm:"index" json:"collected_at"`
}
