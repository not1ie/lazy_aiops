package executor

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

// BatchExecution 批量执行任务
type BatchExecution struct {
	BaseModel
	Name        string     `gorm:"size:128" json:"name"`
	Type        string     `gorm:"size:32" json:"type"` // shell, script, ansible
	Content     string     `gorm:"type:text" json:"content"`
	Targets     string     `gorm:"type:text" json:"targets"` // 目标主机ID列表JSON
	TargetCount int        `json:"target_count"`
	Timeout     int        `gorm:"default:300" json:"timeout"`
	Concurrency int        `gorm:"default:10" json:"concurrency"` // 并发数
	Status      int        `gorm:"default:0" json:"status"` // 0:运行中 1:成功 2:部分失败 3:全部失败 4:已取消
	Progress    int        `json:"progress"` // 进度百分比
	SuccessCount int       `json:"success_count"`
	FailedCount  int       `json:"failed_count"`
	StartedAt   time.Time  `json:"started_at"`
	FinishedAt  *time.Time `json:"finished_at"`
	Duration    int        `json:"duration"`
	Executor    string     `gorm:"size:64" json:"executor"`
}

// BatchExecutionResult 执行结果
type BatchExecutionResult struct {
	BaseModel
	ExecutionID string     `gorm:"size:36;index" json:"execution_id"`
	HostID      string     `gorm:"size:36" json:"host_id"`
	HostIP      string     `gorm:"size:64" json:"host_ip"`
	HostName    string     `gorm:"size:128" json:"host_name"`
	Status      int        `gorm:"default:0" json:"status"` // 0:等待 1:运行中 2:成功 3:失败 4:超时
	ExitCode    int        `json:"exit_code"`
	Stdout      string     `gorm:"type:longtext" json:"stdout"`
	Stderr      string     `gorm:"type:longtext" json:"stderr"`
	StartedAt   *time.Time `json:"started_at"`
	FinishedAt  *time.Time `json:"finished_at"`
	Duration    int        `json:"duration"`
}

// CommandTemplate 命令模板
type CommandTemplate struct {
	BaseModel
	Name        string `gorm:"size:128" json:"name"`
	Category    string `gorm:"size:64" json:"category"`
	Content     string `gorm:"type:text" json:"content"`
	Variables   string `gorm:"type:text" json:"variables"` // 变量定义JSON
	Description string `gorm:"size:512" json:"description"`
	IsBuiltin   bool   `gorm:"default:false" json:"is_builtin"`
}
