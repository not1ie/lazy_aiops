package system

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

// Department 部门
type Department struct {
	BaseModel
	Name     string       `gorm:"size:64;not null" json:"name"`
	ParentID string       `gorm:"size:36" json:"parent_id"`
	Sort     int          `gorm:"default:0" json:"sort"`
	Leader   string       `gorm:"size:64" json:"leader"`
	Phone    string       `gorm:"size:20" json:"phone"`
	Email    string       `gorm:"size:64" json:"email"`
	Status   int          `gorm:"default:1" json:"status"` // 1:启用 0:禁用
	Children []Department `gorm:"-" json:"children"`       // 树形结构辅助字段
}

// Menu 菜单
type Menu struct {
	BaseModel
	Name      string `gorm:"size:64;not null" json:"name"`
	Title     string `gorm:"size:64" json:"title"`
	Path      string `gorm:"size:128" json:"path"`
	Component string `gorm:"size:128" json:"component"`
	Icon      string `gorm:"size:64" json:"icon"`
	Sort      int    `gorm:"default:0" json:"sort"`
	ParentID  string `gorm:"size:36" json:"parent_id"`
	Type      int    `gorm:"default:1" json:"type"` // 1:目录 2:菜单 3:按钮
	Perm      string `gorm:"size:64" json:"perm"`   // 权限标识
	Visible   bool   `gorm:"default:true" json:"visible"`
	Children  []Menu `gorm:"-" json:"children"`
}

// Post 岗位
type Post struct {
	BaseModel
	Name        string `gorm:"size:64;not null" json:"name"`
	Code        string `gorm:"size:64;uniqueIndex" json:"code"`
	Sort        int    `gorm:"default:0" json:"sort"`
	Status      int    `gorm:"default:1" json:"status"` // 1:启用 0:禁用
	Description string `gorm:"size:256" json:"description"`
}

// CaptchaConfig 验证码配置
type CaptchaConfig struct {
	BaseModel
	Enabled       bool   `gorm:"default:true" json:"enabled"`
	Type          string `gorm:"size:32;default:math" json:"type"` // math, string
	Length        int    `gorm:"default:4" json:"length"`
	ExpireSeconds int    `gorm:"default:120" json:"expire_seconds"`
	NoiseLevel    int    `gorm:"default:1" json:"noise_level"`
	Background    string `gorm:"size:32;default:white" json:"background"`
	CaseSensitive bool   `gorm:"default:false" json:"case_sensitive"`
}

// LoginLog 登录日志
type LoginLog struct {
	BaseModel
	Username  string    `gorm:"size:64;index" json:"username"`
	IP        string    `gorm:"size:64;index" json:"ip"`
	UserAgent string    `gorm:"size:256" json:"user_agent"`
	Status    int       `gorm:"default:1;index" json:"status"` // 1:成功 0:失败
	Message   string    `gorm:"size:256" json:"message"`
	LoginAt   time.Time `gorm:"index" json:"login_at"`
}
