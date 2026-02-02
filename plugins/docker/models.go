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
	Name      string `gorm:"size:128" json:"name"`
	HostID    string `gorm:"size:36" json:"host_id"` // 关联CMDB Host
	Status    string `gorm:"size:32" json:"status"` // online, offline
	Version   string `gorm:"size:64" json:"version"`
	ContainerCount int `json:"container_count"`
	ImageCount     int `json:"image_count"`
}

// DockerContainer 容器信息
type DockerContainer struct {
	ID      string   `json:"id"`
	Names   []string `json:"names"`
	Image   string   `json:"image"`
	State   string   `json:"state"`
	Status  string   `json:"status"`
	Created string   `json:"created"` // Changed from int64 to string
	Ports   string   `json:"ports"`
}
