package ai

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

// ChatSession 对话会话
type ChatSession struct {
	BaseModel
	UserID  string `gorm:"size:36;index" json:"user_id"`
	Title   string `gorm:"size:256" json:"title"`
	Type    string `gorm:"size:32" json:"type"`      // chat, analyze, suggest
	Context string `gorm:"type:text" json:"context"` // 上下文信息
}

// ChatMessage 对话消息
type ChatMessage struct {
	BaseModel
	SessionID string `gorm:"size:36;index" json:"session_id"`
	Role      string `gorm:"size:32" json:"role"` // user, assistant, system
	Content   string `gorm:"type:text" json:"content"`
	Meta      string `gorm:"type:text" json:"meta"`
	TokenUsed int    `json:"token_used"`
}

// AnalyzeRequest 分析请求
type AnalyzeRequest struct {
	Type    string `json:"type"`    // logs, error, performance
	Content string `json:"content"` // 日志内容或错误信息
	Context string `json:"context"` // 额外上下文
}

// AnalyzeResponse 分析响应
type AnalyzeResponse struct {
	Summary     string   `json:"summary"`     // 摘要
	Issues      []Issue  `json:"issues"`      // 发现的问题
	Suggestions []string `json:"suggestions"` // 建议
	Severity    string   `json:"severity"`    // 严重程度
}

// Issue 问题
type Issue struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Location    string `json:"location"`
	Severity    string `json:"severity"`
}

// ChatRequest 对话请求
type ChatRequest struct {
	SessionID   string         `json:"session_id"`
	Message     string         `json:"message" binding:"required"`
	Context     string         `json:"context"` // 可选的上下文
	AutoContext bool           `json:"auto_context"`
	ContextHint *AIContextHint `json:"context_hint"`
}

// ChatResponse 对话响应
type ChatResponse struct {
	SessionID      string           `json:"session_id"`
	Reply          string           `json:"reply"`
	TokenUsed      int              `json:"token_used"`
	ContextSummary string           `json:"context_summary,omitempty"`
	ContextPack    *AIContextPack   `json:"context_pack,omitempty"`
	ToolCalls      []AIToolTrace    `json:"tool_calls,omitempty"`
	ExecutionPlan  *AIExecutionPlan `json:"execution_plan,omitempty"`
}

// AIContextHint 聊天上下文线索
type AIContextHint struct {
	Path     string            `json:"path"`
	FullPath string            `json:"full_path"`
	Title    string            `json:"title"`
	Query    map[string]string `json:"query"`
}

// AIContextPack 自动构建的运维上下文包
type AIContextPack struct {
	Scope       string            `json:"scope"`
	Title       string            `json:"title"`
	Summary     string            `json:"summary"`
	Highlights  []string          `json:"highlights"`
	Facts       map[string]string `json:"facts"`
	Route       *AIContextHint    `json:"route,omitempty"`
	GeneratedAt time.Time         `json:"generated_at"`
}

// AIToolPlan 模型规划出的工具调用请求
type AIToolPlan struct {
	UseTools  bool                `json:"use_tools"`
	Focus     string              `json:"focus"`
	ToolCalls []AIToolCallRequest `json:"tool_calls"`
}

// AIToolCallRequest 单次工具调用请求
type AIToolCallRequest struct {
	Name      string            `json:"name"`
	Reason    string            `json:"reason"`
	Arguments map[string]string `json:"arguments"`
}

// AIToolTrace 工具调用轨迹
type AIToolTrace struct {
	Name      string            `json:"name"`
	Reason    string            `json:"reason,omitempty"`
	Arguments map[string]string `json:"arguments,omitempty"`
	Summary   string            `json:"summary,omitempty"`
	Status    string            `json:"status"`
}

// ChatMessageMeta 消息扩展元信息
type ChatMessageMeta struct {
	ContextSummary string           `json:"context_summary,omitempty"`
	ContextScope   string           `json:"context_scope,omitempty"`
	ToolCalls      []AIToolTrace    `json:"tool_calls,omitempty"`
	ExecutionPlan  *AIExecutionPlan `json:"execution_plan,omitempty"`
}

// AIExecutionPlan 审批前执行计划
type AIExecutionPlan struct {
	NeedApproval       bool              `json:"need_approval"`
	Title              string            `json:"title"`
	Objective          string            `json:"objective"`
	Summary            string            `json:"summary"`
	RiskLevel          string            `json:"risk_level"`
	WorkOrderTypeCode  string            `json:"workorder_type_code"`
	ApprovalReason     string            `json:"approval_reason"`
	Prechecks          []string          `json:"prechecks"`
	Steps              []AIExecutionStep `json:"steps"`
	RollbackSteps      []string          `json:"rollback_steps"`
	ValidationSteps    []string          `json:"validation_steps"`
	CreatedWorkOrderID string            `json:"created_workorder_id,omitempty"`
}

// AIExecutionStep 执行步骤
type AIExecutionStep struct {
	Title                string `json:"title"`
	Action               string `json:"action"`
	Risk                 string `json:"risk"`
	NodeType             string `json:"node_type,omitempty"`
	CommandHint          string `json:"command_hint,omitempty"`
	Method               string `json:"method,omitempty"`
	URL                  string `json:"url,omitempty"`
	Body                 string `json:"body,omitempty"`
	RequiresConfirmation bool   `json:"requires_confirmation,omitempty"`
}

// LogAnalysis 日志分析记录
type LogAnalysis struct {
	BaseModel
	Service      string  `gorm:"size:128;index" json:"service"`
	NeedAlert    bool    `json:"need_alert"`
	AlertLevel   string  `gorm:"size:32" json:"alert_level"` // critical/warning/info
	RootCause    string  `gorm:"type:text" json:"root_cause"`
	Impact       string  `gorm:"type:text" json:"impact"`     // 存储为分号分隔的字符串
	Solutions    string  `gorm:"type:text" json:"solutions"`  // 存储为分号分隔的字符串
	Prevention   string  `gorm:"type:text" json:"prevention"` // 存储为分号分隔的字符串
	Confidence   float64 `json:"confidence"`
	LogCount     int     `json:"log_count"`
	ErrorCount   int     `json:"error_count"`
	WarningCount int     `json:"warning_count"`
}

// AIProviderConfig AI模型接入配置
type AIProviderConfig struct {
	BaseModel
	Name          string `gorm:"size:128;index" json:"name"`
	Provider      string `gorm:"size:64;index" json:"provider"`
	BaseURL       string `gorm:"size:512" json:"base_url"`
	Model         string `gorm:"size:128" json:"model"`
	AuthType      string `gorm:"size:32" json:"auth_type"` // bearer/x-api-key/api-key/none
	APIKey        string `gorm:"size:1024" json:"api_key"`
	ExtraHeaders  string `gorm:"type:text" json:"extra_headers"` // JSON object string
	TimeoutSecond int    `gorm:"default:60" json:"timeout_second"`
	Active        bool   `gorm:"default:false;index" json:"active"`
	Description   string `gorm:"size:512" json:"description"`
}
