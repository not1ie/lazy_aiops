package terminal

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

// TerminalSession 终端会话
type TerminalSession struct {
	BaseModel
	HostID        string     `gorm:"size:36;index" json:"host_id"`
	Host          string     `gorm:"size:128" json:"host"`
	Port          int        `gorm:"default:22" json:"port"`
	Username      string     `gorm:"size:64" json:"username"`
	Password      string     `gorm:"size:256" json:"-"`
	PrivateKey    string     `gorm:"type:text" json:"-"`
	UserID        string     `gorm:"size:36;index" json:"user_id"`
	Operator      string     `gorm:"size:64" json:"operator"`
	Status        int        `gorm:"default:0" json:"status"` // 0:待连接 1:已连接 2:已关闭 3:连接失败
	LastError     string     `gorm:"size:512" json:"last_error"`
	StartedAt     *time.Time `json:"started_at"`
	EndedAt       *time.Time `json:"ended_at"`
	HasPassword   bool       `gorm:"-" json:"has_password"`
	HasPrivateKey bool       `gorm:"-" json:"has_private_key"`
}

// TerminalRecord 终端录像
type TerminalRecord struct {
	BaseModel
	SessionID string `gorm:"size:36;index" json:"session_id"`
	Host      string `gorm:"size:128" json:"host"`
	Operator  string `gorm:"size:64" json:"operator"`
	Duration  int    `json:"duration"`                  // 时长(秒)
	Data      string `gorm:"type:longtext" json:"data"` // 录像数据JSON
}
