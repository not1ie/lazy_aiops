package docker

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

// DockerHost Docker主机
type DockerHost struct {
	BaseModel
	Name           string     `gorm:"size:128" json:"name"`
	HostID         string     `gorm:"size:36" json:"host_id"` // 关联CMDB Host
	Status         string     `gorm:"size:32" json:"status"`  // online, offline, error, unknown
	Version        string     `gorm:"size:128" json:"version"`
	ContainerCount int        `json:"container_count"`
	ImageCount     int        `json:"image_count"`
	LastCheckAt    *time.Time `json:"last_check_at"`
	LastOnlineAt   *time.Time `json:"last_online_at"`
	LastError      string     `gorm:"size:512" json:"last_error"`
	LatencyMs      int64      `json:"latency_ms"`
}

// DockerRegistry 镜像仓库
type DockerRegistry struct {
	BaseModel
	Name     string `gorm:"size:128" json:"name"`
	URL      string `gorm:"size:256" json:"url"`
	Username string `gorm:"size:128" json:"username"`
	Password string `gorm:"size:256" json:"password"`
	Insecure bool   `json:"insecure"`
}

// DockerRegistryLogin 仓库登录记录（按主机）
type DockerRegistryLogin struct {
	BaseModel
	RegistryID   string    `gorm:"size:36;index" json:"registry_id"`
	DockerHostID string    `gorm:"size:36;index" json:"docker_host_id"`
	Status       string    `gorm:"size:32" json:"status"` // success, failed
	Message      string    `gorm:"size:512" json:"message"`
	LastLoginAt  time.Time `json:"last_login_at"`
}

// DockerContainer 容器信息
type DockerContainer struct {
	ID      string   `json:"id"`
	Names   []string `json:"names"`
	Image   string   `json:"image"`
	ImageID string   `json:"image_id"`
	State   string   `json:"state"`
	Status  string   `json:"status"`
	Created string   `json:"created"` // Changed from int64 to string
	Ports   string   `json:"ports"`
}
