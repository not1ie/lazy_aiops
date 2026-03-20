package domain

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

// CloudAccount 云账号
type CloudAccount struct {
	BaseModel
	Name        string     `gorm:"size:128" json:"name"`
	Provider    string     `gorm:"size:32" json:"provider"` // aliyun, tencent, huawei, aws, cloudflare
	AccessKey   string     `gorm:"size:128" json:"access_key"`
	SecretKey   string     `gorm:"size:256" json:"-"`
	Region      string     `gorm:"size:64" json:"region"`
	Status      int        `gorm:"default:1" json:"status"`
	DomainCount int        `json:"domain_count"`
	LastSyncAt  *time.Time `json:"last_sync_at"`
}

// CloudDomain 云域名
type CloudDomain struct {
	BaseModel
	AccountID       string        `gorm:"size:36;index" json:"account_id"`
	Account         *CloudAccount `gorm:"foreignKey:AccountID" json:"account,omitempty"`
	Domain          string        `gorm:"size:256;index" json:"domain"`
	Provider        string        `gorm:"size:32" json:"provider"`
	RegistrationAt  *time.Time    `json:"registration_at"`
	ExpirationAt    *time.Time    `json:"expiration_at"`
	DaysToExpire    int           `json:"days_to_expire"`
	AutoRenew       bool          `json:"auto_renew"`
	Status          string        `gorm:"size:32" json:"status"` // normal, expired, transferring
	DNSStatus       string        `gorm:"size:32" json:"dns_status"`
	RecordCount     int           `json:"record_count"`
	HealthStatus    string        `gorm:"size:32" json:"health_status"` // healthy, warning, critical, unknown
	DNSResolved     bool          `json:"dns_resolved"`
	HTTPStatusCode  int           `json:"http_status_code"`
	ResponseTimeMS  int           `json:"response_time_ms"`
	SSLDaysToExpire int           `json:"ssl_days_to_expire"`
	LastCheckAt     *time.Time    `json:"last_check_at"`
}

// SSLCertificate SSL证书
type SSLCertificate struct {
	BaseModel
	Domain       string     `gorm:"size:256;index" json:"domain"`
	Issuer       string     `gorm:"size:256" json:"issuer"`
	Subject      string     `gorm:"size:512" json:"subject"`
	SANs         string     `gorm:"column:sans;type:text" json:"sans"` // Subject Alternative Names
	NotBefore    *time.Time `json:"not_before"`
	NotAfter     *time.Time `json:"not_after"`
	DaysToExpire int        `json:"days_to_expire"`
	SerialNumber string     `gorm:"size:128" json:"serial_number"`
	Fingerprint  string     `gorm:"size:128" json:"fingerprint"`
	Status       int        `gorm:"default:1" json:"status"` // 1:正常 0:过期 2:即将过期
	LastCheckAt  *time.Time `json:"last_check_at"`
}
