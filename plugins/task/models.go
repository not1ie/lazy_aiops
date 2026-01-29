package task

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

// Task 任务
type Task struct {
	BaseModel
	Name        string     `gorm:"size:128" json:"name"`
	Type        string     `gorm:"size:32" json:"type"` // shell, python, ansible
	Content     string     `gorm:"type:text" json:"content"`
	Targets     string     `gorm:"type:text" json:"targets"` // 目标主机ID列表,逗号分隔
	Cron        string     `gorm:"size:64" json:"cron"`      // cron表达式,为空则手动执行
	Timeout     int        `gorm:"default:300" json:"timeout"`
	Enabled     bool       `gorm:"default:true" json:"enabled"`
	LastRunAt   *time.Time `json:"last_run_at"`
	NextRunAt   *time.Time `json:"next_run_at"`
	CreatedBy   string     `gorm:"size:64" json:"created_by"`
}

// TaskExecution 任务执行记录
type TaskExecution struct {
	BaseModel
	TaskID    string     `gorm:"size:36;index" json:"task_id"`
	TaskName  string     `gorm:"size:128" json:"task_name"`
	Status    int        `gorm:"default:0" json:"status"` // 0:运行中 1:成功 2:失败 3:超时
	Output    string     `gorm:"type:text" json:"output"`
	Error     string     `gorm:"type:text" json:"error"`
	StartAt   time.Time  `json:"start_at"`
	EndAt     *time.Time `json:"end_at"`
	Duration  int        `json:"duration"` // 执行时长(秒)
	Executor  string     `gorm:"size:64" json:"executor"`
}
