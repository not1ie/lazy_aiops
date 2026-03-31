package cmdb

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

// Host 主机
type Host struct {
	BaseModel
	Name         string      `gorm:"size:128" json:"name"`
	IP           string      `gorm:"size:64;index" json:"ip"`
	Port         int         `gorm:"default:22" json:"port"`
	OS           string      `gorm:"size:64" json:"os"`
	Status       int         `gorm:"default:1" json:"status"` // 1:在线 0:离线 2:维护
	LastCheckAt  *time.Time  `json:"last_check_at"`
	LastOnlineAt *time.Time  `json:"last_online_at"`
	StatusReason string      `gorm:"size:256" json:"status_reason"`
	GroupID      string      `gorm:"size:36" json:"group_id"`
	Group        *HostGroup  `gorm:"foreignKey:GroupID" json:"group,omitempty"`
	CredentialID string      `gorm:"size:36" json:"credential_id"`
	Credential   *Credential `gorm:"foreignKey:CredentialID" json:"credential,omitempty"`
	CPU          string      `gorm:"size:64" json:"cpu"`
	Memory       string      `gorm:"size:64" json:"memory"`
	Disk         string      `gorm:"size:128" json:"disk"`
	Tags         string      `gorm:"size:256" json:"tags"`
	Description  string      `gorm:"size:512" json:"description"`
}

// NetworkDevice 网络设备资产（交换机/防火墙）
type NetworkDevice struct {
	BaseModel
	Name            string      `gorm:"size:128" json:"name"`
	DeviceType      string      `gorm:"size:32;index:idx_network_type_ip" json:"device_type"` // switch, firewall
	Vendor          string      `gorm:"size:64" json:"vendor"`
	Model           string      `gorm:"size:64" json:"model"`
	IP              string      `gorm:"size:64;index:idx_network_type_ip" json:"ip"`
	ManagePort      int         `gorm:"default:22" json:"manage_port"`
	SNMPVersion     string      `gorm:"size:16;default:v2c" json:"snmp_version"` // v1, v2c, v3
	SNMPCommunity   string      `gorm:"size:128" json:"snmp_community"`
	SNMPPort        int         `gorm:"default:161" json:"snmp_port"`
	SNMPUser        string      `gorm:"size:64" json:"snmp_user"`
	SNMPAuthProto   string      `gorm:"size:16" json:"snmp_auth_proto"` // MD5, SHA
	SNMPAuthPass    string      `gorm:"size:128" json:"snmp_auth_pass,omitempty"`
	SNMPPrivProto   string      `gorm:"size:16" json:"snmp_priv_proto"` // DES, AES
	SNMPPrivPass    string      `gorm:"size:128" json:"snmp_priv_pass,omitempty"`
	CredentialID    string      `gorm:"size:36" json:"credential_id"`
	Credential      *Credential `gorm:"foreignKey:CredentialID" json:"credential,omitempty"`
	Location        string      `gorm:"size:128" json:"location"`
	Rack            string      `gorm:"size:64" json:"rack"`
	SerialNumber    string      `gorm:"size:128" json:"serial_number"`
	FirmwareVersion string      `gorm:"size:128" json:"firmware_version"`
	Status          int         `gorm:"default:1" json:"status"` // 1:在线 0:离线 2:告警
	LastCheckAt     *time.Time  `json:"last_check_at"`
	Tags            string      `gorm:"size:256" json:"tags"`
	Description     string      `gorm:"size:512" json:"description"`
}

// HostGroup 主机分组
type HostGroup struct {
	BaseModel
	Name        string `gorm:"size:64;uniqueIndex" json:"name"`
	Description string `gorm:"size:256" json:"description"`
	ParentID    string `gorm:"size:36" json:"parent_id"`
}

// Credential 凭据
type Credential struct {
	BaseModel
	Name       string `gorm:"size:64" json:"name"`
	Type       string `gorm:"size:32" json:"type"` // password, key
	Username   string `gorm:"size:64" json:"username"`
	Password   string `gorm:"size:256" json:"password,omitempty"`
	PrivateKey string `gorm:"type:text" json:"private_key,omitempty"`
	Passphrase string `gorm:"size:256" json:"passphrase,omitempty"`
	AccessKey  string `gorm:"size:256" json:"access_key,omitempty"`
	SecretKey  string `gorm:"size:256" json:"secret_key,omitempty"`
	Notes      string `gorm:"size:512" json:"notes,omitempty"`
}

// DatabaseAsset 数据库资产
type DatabaseAsset struct {
	BaseModel
	Name        string `gorm:"size:128" json:"name"`
	Type        string `gorm:"size:32" json:"type"` // mysql, postgres, redis, mongodb, oracle
	Host        string `gorm:"size:128" json:"host"`
	Port        int    `gorm:"default:3306" json:"port"`
	Username    string `gorm:"size:64" json:"username"`
	Password    string `gorm:"size:256" json:"password,omitempty"`
	Database    string `gorm:"size:128" json:"database"`
	Environment string `gorm:"size:32" json:"environment"` // dev, test, prod
	Owner       string `gorm:"size:64" json:"owner"`
	Tags        string `gorm:"size:256" json:"tags"`
	Status      int    `gorm:"default:1" json:"status"` // 1:正常 0:禁用
	Description string `gorm:"size:512" json:"description"`
}

// CloudAccount 云账号
type CloudAccount struct {
	BaseModel
	Name        string `gorm:"size:128" json:"name"`
	Provider    string `gorm:"size:32" json:"provider"` // tencent, baidu, aliyun, huawei, aws
	AccessKey   string `gorm:"size:256" json:"access_key,omitempty"`
	SecretKey   string `gorm:"size:256" json:"secret_key,omitempty"`
	Region      string `gorm:"size:64" json:"region"`
	Status      int    `gorm:"default:1" json:"status"` // 1:启用 0:禁用
	Description string `gorm:"size:512" json:"description"`
}

// CloudResource 云资源
type CloudResource struct {
	BaseModel
	AccountID  string        `gorm:"size:36;index" json:"account_id"`
	Account    *CloudAccount `gorm:"foreignKey:AccountID" json:"account,omitempty"`
	ResourceID string        `gorm:"size:128" json:"resource_id"`
	Name       string        `gorm:"size:128" json:"name"`
	Type       string        `gorm:"size:32" json:"type"` // ecs, rds, slb, vpc
	Region     string        `gorm:"size:64" json:"region"`
	Zone       string        `gorm:"size:64" json:"zone"`
	IP         string        `gorm:"size:64" json:"ip"`
	Status     string        `gorm:"size:32" json:"status"`
	Spec       string        `gorm:"size:128" json:"spec"`
	Tags       string        `gorm:"size:256" json:"tags"`
}
