package api

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type loginLogRecord struct {
	ID        string `gorm:"primaryKey;size:36"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
	Username  string         `gorm:"size:64;index"`
	IP        string         `gorm:"size:64;index"`
	UserAgent string         `gorm:"size:256"`
	Status    int            `gorm:"default:1;index"`
	Message   string         `gorm:"size:256"`
	LoginAt   time.Time      `gorm:"index"`
}

func (l *loginLogRecord) BeforeCreate(tx *gorm.DB) error {
	if l.ID == "" {
		l.ID = uuid.New().String()
	}
	return nil
}

func (loginLogRecord) TableName() string {
	return "login_logs"
}
