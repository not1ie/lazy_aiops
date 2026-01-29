package sqlaudit

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

// DBInstance 数据库实例
type DBInstance struct {
	BaseModel
	Name        string `gorm:"size:128" json:"name"`
	Type        string `gorm:"size:32" json:"type"` // mysql, postgresql, oracle, sqlserver
	Host        string `gorm:"size:256" json:"host"`
	Port        int    `gorm:"default:3306" json:"port"`
	Username    string `gorm:"size:64" json:"username"`
	Password    string `gorm:"size:256" json:"-"`
	Database    string `gorm:"size:128" json:"database"`
	Charset     string `gorm:"size:32;default:utf8mb4" json:"charset"`
	Status      int    `gorm:"default:1" json:"status"` // 1:正常 0:异常
	Environment string `gorm:"size:32" json:"environment"` // dev, test, staging, prod
	Description string `gorm:"size:512" json:"description"`
}

// SQLWorkOrder SQL工单
type SQLWorkOrder struct {
	BaseModel
	Title        string     `gorm:"size:256" json:"title"`
	InstanceID   string     `gorm:"size:36;index" json:"instance_id"`
	Instance     *DBInstance `gorm:"foreignKey:InstanceID" json:"instance,omitempty"`
	Database     string     `gorm:"size:128" json:"database"`
	SQLType      string     `gorm:"size:32" json:"sql_type"` // DDL, DML, DQL
	SQLContent   string     `gorm:"type:longtext" json:"sql_content"`
	Status       int        `gorm:"default:0" json:"status"` // 0:待审核 1:审核通过 2:审核拒绝 3:执行中 4:执行成功 5:执行失败 6:已回滚
	AuditResult  string     `gorm:"type:text" json:"audit_result"` // 审核结果
	AuditLevel   int        `json:"audit_level"` // 0:通过 1:警告 2:错误
	AffectedRows int64      `json:"affected_rows"`
	ExecuteTime  int        `json:"execute_time"` // 执行时间(ms)
	RollbackSQL  string     `gorm:"type:longtext" json:"rollback_sql"`
	Submitter    string     `gorm:"size:64" json:"submitter"`
	Reviewer     string     `gorm:"size:64" json:"reviewer"`
	Executor     string     `gorm:"size:64" json:"executor"`
	ReviewedAt   *time.Time `json:"reviewed_at"`
	ExecutedAt   *time.Time `json:"executed_at"`
	ReviewRemark string     `gorm:"size:512" json:"review_remark"`
	ScheduledAt  *time.Time `json:"scheduled_at"` // 定时执行时间
}

// SQLAuditLog SQL审计日志
type SQLAuditLog struct {
	BaseModel
	InstanceID   string    `gorm:"size:36;index" json:"instance_id"`
	InstanceName string    `gorm:"size:128" json:"instance_name"`
	Database     string    `gorm:"size:128" json:"database"`
	Username     string    `gorm:"size:64" json:"username"`
	ClientIP     string    `gorm:"size:64" json:"client_ip"`
	SQLType      string    `gorm:"size:32;index" json:"sql_type"`
	SQLContent   string    `gorm:"type:text" json:"sql_content"`
	AffectedRows int64     `json:"affected_rows"`
	ExecuteTime  int       `json:"execute_time"` // ms
	Status       int       `gorm:"default:1" json:"status"` // 1:成功 0:失败
	ErrorMsg     string    `gorm:"size:512" json:"error_msg"`
	ExecutedAt   time.Time `gorm:"index" json:"executed_at"`
}

// SQLAuditRule SQL审核规则
type SQLAuditRule struct {
	BaseModel
	Name        string `gorm:"size:128" json:"name"`
	Type        string `gorm:"size:32" json:"type"` // syntax, security, performance
	Level       int    `json:"level"` // 0:info 1:warning 2:error
	Pattern     string `gorm:"size:512" json:"pattern"` // 正则表达式
	Message     string `gorm:"size:512" json:"message"`
	Suggestion  string `gorm:"size:512" json:"suggestion"`
	Enabled     bool   `gorm:"default:true" json:"enabled"`
	Description string `gorm:"size:512" json:"description"`
}
