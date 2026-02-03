package application

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

// Application 应用/服务
type Application struct {
	BaseModel
	Name        string `gorm:"size:64;not null" json:"name"`
	Code        string `gorm:"size:64;uniqueIndex" json:"code"` // 唯一标识
	Description string `gorm:"size:256" json:"description"`
	Owner       string `gorm:"size:64" json:"owner"`
	Language    string `gorm:"size:32" json:"language"` // java, go, nodejs
	GitRepo     string `gorm:"size:256" json:"git_repo"`
	BuildTool   string `gorm:"size:32" json:"build_tool"` // maven, npm, go build
}

// AppEnvironment 应用环境配置
type AppEnvironment struct {
	BaseModel
	AppID       string `gorm:"size:36;index" json:"app_id"`
	EnvName     string `gorm:"size:32" json:"env_name"` // dev, test, prod
	ClusterID   string `gorm:"size:36" json:"cluster_id"` // K8s Cluster ID
	Namespace   string `gorm:"size:64" json:"namespace"`
	Replicas    int    `gorm:"default:1" json:"replicas"`
	CPUQuota    string `gorm:"size:32" json:"cpu_quota"`
	MemQuota    string `gorm:"size:32" json:"mem_quota"`
	HealthCheck string `gorm:"size:256" json:"health_check"`
}
