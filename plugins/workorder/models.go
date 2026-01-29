package workorder

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

// WorkOrderType 工单类型
type WorkOrderType struct {
	BaseModel
	Name        string `gorm:"size:64" json:"name"`
	Code        string `gorm:"size:32;uniqueIndex" json:"code"`
	Icon        string `gorm:"size:64" json:"icon"`
	FlowID      string `gorm:"size:36" json:"flow_id"` // 关联审批流程
	Template    string `gorm:"type:text" json:"template"` // 表单模板JSON
	Enabled     bool   `gorm:"default:true" json:"enabled"`
	Description string `gorm:"size:512" json:"description"`
}

// WorkOrder 工单
type WorkOrder struct {
	BaseModel
	Title       string     `gorm:"size:256" json:"title"`
	TypeID      string     `gorm:"size:36;index" json:"type_id"`
	TypeName    string     `gorm:"size:64" json:"type_name"`
	Content     string     `gorm:"type:text" json:"content"`
	FormData    string     `gorm:"type:text" json:"form_data"` // 表单数据JSON
	Priority    int        `gorm:"default:2" json:"priority"`  // 1:紧急 2:高 3:中 4:低
	Status      int        `gorm:"default:0;index" json:"status"` // 0:待审批 1:审批中 2:已通过 3:已拒绝 4:执行中 5:已完成 6:已取消
	Submitter   string     `gorm:"size:64" json:"submitter"`
	SubmitterID string     `gorm:"size:36" json:"submitter_id"`
	Assignee    string     `gorm:"size:64" json:"assignee"`
	AssigneeID  string     `gorm:"size:36" json:"assignee_id"`
	CurrentStep int        `json:"current_step"`
	TotalSteps  int        `json:"total_steps"`
	// AI辅助
	AISuggestion string     `gorm:"type:text" json:"ai_suggestion"` // AI处理建议
	AIRisk       string     `gorm:"size:32" json:"ai_risk"`         // AI风险评估
	// 时间
	ExpectedAt   *time.Time `json:"expected_at"`
	CompletedAt  *time.Time `json:"completed_at"`
}

// WorkOrderFlow 审批流程
type WorkOrderFlow struct {
	BaseModel
	Name        string `gorm:"size:128" json:"name"`
	Steps       string `gorm:"type:text" json:"steps"` // 步骤JSON
	Enabled     bool   `gorm:"default:true" json:"enabled"`
	Description string `gorm:"size:512" json:"description"`
}

// WorkOrderStep 审批步骤
type WorkOrderStep struct {
	BaseModel
	OrderID     string     `gorm:"size:36;index" json:"order_id"`
	Step        int        `json:"step"`
	Name        string     `gorm:"size:64" json:"name"`
	Approver    string     `gorm:"size:64" json:"approver"`
	ApproverID  string     `gorm:"size:36" json:"approver_id"`
	Status      int        `gorm:"default:0" json:"status"` // 0:待审批 1:通过 2:拒绝
	Comment     string     `gorm:"size:512" json:"comment"`
	ApprovedAt  *time.Time `json:"approved_at"`
}

// WorkOrderComment 工单评论
type WorkOrderComment struct {
	BaseModel
	OrderID  string `gorm:"size:36;index" json:"order_id"`
	UserID   string `gorm:"size:36" json:"user_id"`
	Username string `gorm:"size:64" json:"username"`
	Content  string `gorm:"type:text" json:"content"`
	Type     string `gorm:"size:32" json:"type"` // comment, system, ai
}

// WorkOrderAttachment 工单附件
type WorkOrderAttachment struct {
	BaseModel
	OrderID  string `gorm:"size:36;index" json:"order_id"`
	Name     string `gorm:"size:256" json:"name"`
	Path     string `gorm:"size:512" json:"path"`
	Size     int64  `json:"size"`
	MimeType string `gorm:"size:128" json:"mime_type"`
	Uploader string `gorm:"size:64" json:"uploader"`
}
