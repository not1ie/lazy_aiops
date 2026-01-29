package notify

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

// NotifyChannel 通知渠道
type NotifyChannel struct {
	BaseModel
	Name        string `gorm:"size:128" json:"name"`
	Type        string `gorm:"size:32" json:"type"` // webhook, feishu, dingtalk, wecom, email, sms
	Webhook     string `gorm:"size:512" json:"webhook"`
	Secret      string `gorm:"size:256" json:"-"` // 签名密钥
	AppID       string `gorm:"size:128" json:"app_id"`
	AppSecret   string `gorm:"size:256" json:"-"`
	// 邮件配置
	SMTPHost    string `gorm:"size:128" json:"smtp_host"`
	SMTPPort    int    `json:"smtp_port"`
	SMTPUser    string `gorm:"size:128" json:"smtp_user"`
	SMTPPass    string `gorm:"size:256" json:"-"`
	// 短信配置
	SMSProvider string `gorm:"size:32" json:"sms_provider"` // aliyun, tencent
	SMSSign     string `gorm:"size:64" json:"sms_sign"`
	SMSTemplate string `gorm:"size:128" json:"sms_template"`
	Enabled     bool   `gorm:"default:true" json:"enabled"`
	Description string `gorm:"size:512" json:"description"`
}

// NotifyTemplate 通知模板
type NotifyTemplate struct {
	BaseModel
	Name        string `gorm:"size:128" json:"name"`
	Type        string `gorm:"size:32" json:"type"` // alert, workorder, task, custom
	Title       string `gorm:"size:256" json:"title"`
	Content     string `gorm:"type:text" json:"content"` // 支持变量 {{.xxx}}
	ChannelType string `gorm:"size:32" json:"channel_type"`
	Enabled     bool   `gorm:"default:true" json:"enabled"`
}

// NotifyRecord 通知记录
type NotifyRecord struct {
	BaseModel
	ChannelID   string     `gorm:"size:36;index" json:"channel_id"`
	ChannelName string     `gorm:"size:128" json:"channel_name"`
	ChannelType string     `gorm:"size:32" json:"channel_type"`
	Title       string     `gorm:"size:256" json:"title"`
	Content     string     `gorm:"type:text" json:"content"`
	Receiver    string     `gorm:"size:256" json:"receiver"` // 接收人
	Status      int        `gorm:"default:0" json:"status"`  // 0:待发送 1:成功 2:失败
	ErrorMsg    string     `gorm:"size:512" json:"error_msg"`
	RetryCount  int        `json:"retry_count"`
	SentAt      *time.Time `json:"sent_at"`
	Source      string     `gorm:"size:64" json:"source"`    // alert, workorder, manual
	SourceID    string     `gorm:"size:36" json:"source_id"` // 关联ID
}

// NotifyGroup 通知组
type NotifyGroup struct {
	BaseModel
	Name        string `gorm:"size:128" json:"name"`
	Description string `gorm:"size:512" json:"description"`
	Channels    string `gorm:"type:text" json:"channels"` // 渠道ID列表,JSON数组
	Users       string `gorm:"type:text" json:"users"`    // 用户ID列表,JSON数组
}
