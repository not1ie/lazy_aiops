package oncall

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

// OnCallSchedule 值班排班
type OnCallSchedule struct {
	BaseModel
	Name        string `gorm:"size:128" json:"name"`
	TeamID      string `gorm:"size:36" json:"team_id"`
	TeamName    string `gorm:"size:64" json:"team_name"`
	Type        string `gorm:"size:32" json:"type"` // daily, weekly, custom
	Timezone    string `gorm:"size:64;default:Asia/Shanghai" json:"timezone"`
	StartTime   string `gorm:"size:8" json:"start_time"`  // 09:00
	EndTime     string `gorm:"size:8" json:"end_time"`    // 18:00
	Rotation    string `gorm:"type:text" json:"rotation"` // 轮换规则JSON
	Enabled     bool   `gorm:"default:true" json:"enabled"`
	Description string `gorm:"size:512" json:"description"`
}

// OnCallShift 值班班次
type OnCallShift struct {
	BaseModel
	ScheduleID string    `gorm:"size:36;index" json:"schedule_id"`
	UserID     string    `gorm:"size:36" json:"user_id"`
	Username   string    `gorm:"size:64" json:"username"`
	Phone      string    `gorm:"size:20" json:"phone"`
	Email      string    `gorm:"size:128" json:"email"`
	StartAt    time.Time `gorm:"index" json:"start_at"`
	EndAt      time.Time `gorm:"index" json:"end_at"`
	Type       string    `gorm:"size:32" json:"type"` // primary, backup
	Status     int       `gorm:"default:1" json:"status"` // 1:正常 0:已换班
	SwappedBy  string    `gorm:"size:36" json:"swapped_by"`
}

// OnCallTeam 值班团队
type OnCallTeam struct {
	BaseModel
	Name        string `gorm:"size:64" json:"name"`
	Members     string `gorm:"type:text" json:"members"` // 成员ID列表JSON
	NotifyGroup string `gorm:"size:36" json:"notify_group"` // 通知组ID
	Description string `gorm:"size:512" json:"description"`
}

// OnCallOverride 临时换班
type OnCallOverride struct {
	BaseModel
	ScheduleID   string    `gorm:"size:36;index" json:"schedule_id"`
	OriginalUser string    `gorm:"size:36" json:"original_user"`
	OverrideUser string    `gorm:"size:36" json:"override_user"`
	StartAt      time.Time `json:"start_at"`
	EndAt        time.Time `json:"end_at"`
	Reason       string    `gorm:"size:256" json:"reason"`
	CreatedBy    string    `gorm:"size:64" json:"created_by"`
}

// OnCallEscalation 升级策略
type OnCallEscalation struct {
	BaseModel
	Name        string `gorm:"size:128" json:"name"`
	ScheduleID  string `gorm:"size:36" json:"schedule_id"`
	Rules       string `gorm:"type:text" json:"rules"` // 升级规则JSON
	Enabled     bool   `gorm:"default:true" json:"enabled"`
	Description string `gorm:"size:512" json:"description"`
}
