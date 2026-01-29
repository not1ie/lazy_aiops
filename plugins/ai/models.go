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
	UserID   string `gorm:"size:36;index" json:"user_id"`
	Title    string `gorm:"size:256" json:"title"`
	Type     string `gorm:"size:32" json:"type"` // chat, analyze, suggest
	Context  string `gorm:"type:text" json:"context"` // 上下文信息
}

// ChatMessage 对话消息
type ChatMessage struct {
	BaseModel
	SessionID string `gorm:"size:36;index" json:"session_id"`
	Role      string `gorm:"size:32" json:"role"` // user, assistant, system
	Content   string `gorm:"type:text" json:"content"`
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
	SessionID string `json:"session_id"`
	Message   string `json:"message" binding:"required"`
	Context   string `json:"context"` // 可选的上下文
}

// ChatResponse 对话响应
type ChatResponse struct {
	SessionID string `json:"session_id"`
	Reply     string `json:"reply"`
	TokenUsed int    `json:"token_used"`
}

// LogAnalysis 日志分析记录
type LogAnalysis struct {
	BaseModel
	Service      string  `gorm:"size:128;index" json:"service"`
	NeedAlert    bool    `json:"need_alert"`
	AlertLevel   string  `gorm:"size:32" json:"alert_level"` // critical/warning/info
	RootCause    string  `gorm:"type:text" json:"root_cause"`
	Impact       string  `gorm:"type:text" json:"impact"`       // 存储为分号分隔的字符串
	Solutions    string  `gorm:"type:text" json:"solutions"`    // 存储为分号分隔的字符串
	Prevention   string  `gorm:"type:text" json:"prevention"`   // 存储为分号分隔的字符串
	Confidence   float64 `json:"confidence"`
	LogCount     int     `json:"log_count"`
	ErrorCount   int     `json:"error_count"`
	WarningCount int     `json:"warning_count"`
}
