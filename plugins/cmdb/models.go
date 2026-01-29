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
	Password   string `gorm:"size:256" json:"-"`
	PrivateKey string `gorm:"type:text" json:"-"`
	Passphrase string `gorm:"size:256" json:"-"`
}
