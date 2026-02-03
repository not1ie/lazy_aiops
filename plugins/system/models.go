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
