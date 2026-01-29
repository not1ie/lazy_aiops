package knowledge

import (
	"time"

	"gorm.io/gorm"
)

// Document 知识库文档
type Document struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Title     string         `gorm:"size:255;not null" json:"title"`
	Content   string         `gorm:"type:text;not null" json:"content"` // Markdown内容
	Tags      string         `gorm:"size:255" json:"tags"`              // 标签，逗号分隔
	Category  string         `gorm:"size:50" json:"category"`           // 分类：runbook, post-mortem, guide
	CreatedBy string         `gorm:"size:50" json:"created_by"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// QARequest 问答请求
type QARequest struct {
	Question string `json:"question" binding:"required"`
}

// QAResponse 问答响应
type QAResponse struct {
	Answer     string     `json:"answer"`
	References []Document `json:"references"`
}
