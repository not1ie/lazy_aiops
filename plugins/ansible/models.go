package ansible

import (
	"time"

	"github.com/lazyautoops/lazy-auto-ops/internal/core"
)

// AnsiblePlaybook Playbook
type AnsiblePlaybook struct {
	core.BaseModel
	Name        string `json:"name" gorm:"size:100"`
	Description string `json:"description" gorm:"size:500"`
	FilePath    string `json:"file_path" gorm:"size:500"`
	Content     string `json:"content" gorm:"type:longtext"`
	Tags        string `json:"tags" gorm:"size:500"`
	Category    string `json:"category" gorm:"size:100"`
}

// AnsibleInventory Inventory
type AnsibleInventory struct {
	core.BaseModel
	Name        string `json:"name" gorm:"size:100"`
	Description string `json:"description" gorm:"size:500"`
	FilePath    string `json:"file_path" gorm:"size:500"`
	Content     string `json:"content" gorm:"type:longtext"`
	Source      string `json:"source" gorm:"size:50"` // manual, cmdb
}

// AnsibleRole Role
type AnsibleRole struct {
	core.BaseModel
	Name        string `json:"name" gorm:"size:100"`
	Version     string `json:"version" gorm:"size:50"`
	Description string `json:"description" gorm:"size:500"`
	Source      string `json:"source" gorm:"size:50"` // galaxy, git, local
	Path        string `json:"path" gorm:"size:500"`
}

// AnsibleExecution 执行记录
type AnsibleExecution struct {
	core.BaseModel
	PlaybookID   string     `json:"playbook_id" gorm:"size:36;index"`
	PlaybookName string     `json:"playbook_name" gorm:"size:100"`
	InventoryID  string     `json:"inventory_id" gorm:"size:36"`
	ExtraVars    string     `json:"extra_vars" gorm:"type:text"`
	Tags         string     `json:"tags" gorm:"size:500"`
	Limit        string     `json:"limit" gorm:"size:500"`
	Check        bool       `json:"check"`
	Status       int        `json:"status" gorm:"default:0"` // 0运行中 1成功 2失败 3取消
	Output       string     `json:"output" gorm:"type:longtext"`
	StartedAt    time.Time  `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at"`
	Duration     int        `json:"duration"`
	Executor     string     `json:"executor" gorm:"size:100"`
}
