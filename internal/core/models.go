package core

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BaseModel 基础模型
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

// User 用户模型
type User struct {
	BaseModel
	Username string `gorm:"uniqueIndex;size:64" json:"username"`
	Password string `gorm:"size:128" json:"-"`
	Nickname string `gorm:"size:64" json:"nickname"`
	Email    string `gorm:"size:128" json:"email"`
	Phone    string `gorm:"size:20" json:"phone"`
	Avatar   string `gorm:"size:256" json:"avatar"`
	Status   int    `gorm:"default:1" json:"status"` // 1:启用 0:禁用
	RoleID   string `gorm:"size:36" json:"role_id"`
	Role     *Role  `gorm:"foreignKey:RoleID" json:"role,omitempty"`
}

// Role 角色模型
type Role struct {
	BaseModel
	Name        string        `gorm:"uniqueIndex;size:64" json:"name"`
	Code        string        `gorm:"uniqueIndex;size:64" json:"code"`
	Description string        `gorm:"size:256" json:"description"`
	Permissions []*Permission `gorm:"many2many:role_permissions" json:"permissions,omitempty"`
}

// Permission 权限模型
type Permission struct {
	BaseModel
	Name     string `gorm:"size:64" json:"name"`
	Code     string `gorm:"uniqueIndex;size:128" json:"code"` // 如: cmdb:host:read
	Type     string `gorm:"size:32" json:"type"`              // menu, button, api
	ParentID string `gorm:"size:36" json:"parent_id"`
}

// OperationLog 操作日志
type OperationLog struct {
	BaseModel
	UserID    string `gorm:"size:36;index" json:"user_id"`
	Username  string `gorm:"size:64" json:"username"`
	Module    string `gorm:"size:64;index" json:"module"`
	Action    string `gorm:"size:64" json:"action"`
	Target    string `gorm:"size:256" json:"target"`
	Detail    string `gorm:"type:text" json:"detail"`
	IP        string `gorm:"size:64" json:"ip"`
	UserAgent string `gorm:"size:256" json:"user_agent"`
	Status    int    `gorm:"default:1" json:"status"` // 1:成功 0:失败
}
