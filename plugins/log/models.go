package log

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

type LogLibrary struct {
	BaseModel
	Name         string `gorm:"size:128;uniqueIndex" json:"name"`
	Type         string `gorm:"size:32" json:"type"` // "es", "loki"
	Source       string `gorm:"size:256" json:"source"`
	Retention    string `gorm:"size:32" json:"retention"`
	Status       string `gorm:"size:32;default:'active'" json:"status"` // "active", "error"
	StatusReason string `gorm:"size:256" json:"status_reason"`
}

type LogAlertRule struct {
	BaseModel
	Name        string  `gorm:"size:128;uniqueIndex" json:"name"`
	LibraryID   string  `gorm:"size:36" json:"library_id"`
	LibraryName string  `gorm:"size:128" json:"library_name"`
	Level       string  `gorm:"size:16" json:"level"` // "P0", "P1", "P2"
	Query       string  `gorm:"type:text" json:"query"`
	Operator    string  `gorm:"size:8" json:"operator"` // ">", "<", ">=", "<=", "=="
	Threshold   float64 `json:"threshold"`
	Duration    string  `gorm:"size:32" json:"duration"`
	Channels    string  `gorm:"size:256" json:"channels"` // Comma-separated (e.g. "钉钉,邮件")
	Enabled     bool    `gorm:"default:true" json:"enabled"`
}

type LogPermission struct {
	BaseModel
	Type       string `gorm:"size:32" json:"type"` // "user", "role"
	Subject    string `gorm:"size:128" json:"subject"`
	LibraryIDs string `gorm:"type:text" json:"library_ids"` // Comma-separated or JSON list
	Actions    string `gorm:"size:256" json:"actions"`     // Comma-separated
	Creator    string `gorm:"size:64;default:'admin'" json:"creator"`
}
