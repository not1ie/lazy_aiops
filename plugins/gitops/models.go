package gitops

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

// GitRepo Git仓库
type GitRepo struct {
	BaseModel
	Name        string     `gorm:"size:128;uniqueIndex" json:"name"`
	URL         string     `gorm:"size:512" json:"url"`
	Branch      string     `gorm:"size:64;default:main" json:"branch"`
	SSHKey      string     `gorm:"type:text" json:"-"` // SSH私钥
	LocalPath   string     `gorm:"size:512" json:"local_path"`
	Status      int        `gorm:"default:0" json:"status"` // 0:正常 1:同步中 2:错误
	LastSyncAt  *time.Time `json:"last_sync_at"`
	Description string     `gorm:"size:512" json:"description"`
}

// GitConfig 配置文件
type GitConfig struct {
	BaseModel
	RepoID      string   `gorm:"size:36;index" json:"repo_id"`
	Repo        *GitRepo `gorm:"foreignKey:RepoID" json:"repo,omitempty"`
	Name        string   `gorm:"size:128" json:"name"`
	FilePath    string   `gorm:"size:512" json:"file_path"` // 相对于仓库根目录的路径
	Type        string   `gorm:"size:32" json:"type"`       // yaml, json, toml, ini
	Environment string   `gorm:"size:64" json:"environment"` // dev, staging, prod
	Description string   `gorm:"size:512" json:"description"`
}

// ConfigChange 配置变更记录
type ConfigChange struct {
	BaseModel
	ConfigID      string `gorm:"size:36;index" json:"config_id"`
	ConfigName    string `gorm:"size:128" json:"config_name"`
	ChangeType    string `gorm:"size:32" json:"change_type"` // create, update, delete
	Content       string `gorm:"type:text" json:"content"`
	CommitHash    string `gorm:"size:64" json:"commit_hash"`
	CommitMessage string `gorm:"size:512" json:"commit_message"`
	CommitBy      string `gorm:"size:64" json:"commit_by"`
}
