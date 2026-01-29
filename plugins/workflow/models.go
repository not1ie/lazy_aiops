package workflow

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

// Workflow 工作流定义
type Workflow struct {
	BaseModel
	Name        string `gorm:"size:128" json:"name"`
	Description string `gorm:"size:512" json:"description"`
	Category    string `gorm:"size:64" json:"category"` // deploy, monitor, backup, custom
	Definition  string `gorm:"type:longtext" json:"definition"` // 流程定义JSON
	Variables   string `gorm:"type:text" json:"variables"` // 变量定义JSON
	Trigger     string `gorm:"size:64" json:"trigger"` // manual, schedule, webhook, alert
	CronExpr    string `gorm:"size:64" json:"cron_expr"` // 定时触发表达式
	Enabled     bool   `gorm:"default:true" json:"enabled"`
	Version     int    `gorm:"default:1" json:"version"`
	CreatedBy   string `gorm:"size:64" json:"created_by"`
}

// WorkflowNode 节点类型
// - start: 开始节点
// - end: 结束节点
// - shell: 执行Shell命令
// - http: HTTP请求
// - condition: 条件判断
// - parallel: 并行执行
// - wait: 等待
// - notify: 发送通知
// - ai: AI分析
// - approval: 人工审批

// WorkflowExecution 工作流执行实例
type WorkflowExecution struct {
	BaseModel
	WorkflowID   string     `gorm:"size:36;index" json:"workflow_id"`
	WorkflowName string     `gorm:"size:128" json:"workflow_name"`
	Status       int        `gorm:"default:0" json:"status"` // 0:运行中 1:成功 2:失败 3:取消 4:等待审批
	Variables    string     `gorm:"type:text" json:"variables"` // 运行时变量
	CurrentNode  string     `gorm:"size:64" json:"current_node"`
	StartedAt    time.Time  `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at"`
	Duration     int        `json:"duration"` // 执行时长(秒)
	Trigger      string     `gorm:"size:64" json:"trigger"`
	TriggerBy    string     `gorm:"size:64" json:"trigger_by"`
	Error        string     `gorm:"type:text" json:"error"`
}

// WorkflowNodeExecution 节点执行记录
type WorkflowNodeExecution struct {
	BaseModel
	ExecutionID string     `gorm:"size:36;index" json:"execution_id"`
	NodeID      string     `gorm:"size:64" json:"node_id"`
	NodeName    string     `gorm:"size:128" json:"node_name"`
	NodeType    string     `gorm:"size:32" json:"node_type"`
	Status      int        `gorm:"default:0" json:"status"` // 0:运行中 1:成功 2:失败 3:跳过
	Input       string     `gorm:"type:text" json:"input"`
	Output      string     `gorm:"type:text" json:"output"`
	Error       string     `gorm:"type:text" json:"error"`
	StartedAt   time.Time  `json:"started_at"`
	FinishedAt  *time.Time `json:"finished_at"`
	Duration    int        `json:"duration"`
}

// WorkflowTemplate 工作流模板
type WorkflowTemplate struct {
	BaseModel
	Name        string `gorm:"size:128" json:"name"`
	Category    string `gorm:"size:64" json:"category"`
	Description string `gorm:"size:512" json:"description"`
	Definition  string `gorm:"type:longtext" json:"definition"`
	Variables   string `gorm:"type:text" json:"variables"`
	Icon        string `gorm:"size:64" json:"icon"`
	IsBuiltin   bool   `gorm:"default:false" json:"is_builtin"`
}
