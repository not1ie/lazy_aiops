package cost

import (
	"time"

	"github.com/lazyautoops/lazy-auto-ops/internal/core"
)

// CloudAccount 云账号
type CloudAccount struct {
	core.BaseModel
	Name        string `json:"name" gorm:"size:100"`
	Provider    string `json:"provider" gorm:"size:50"` // aliyun, tencent, aws, huawei
	AccessKey   string `json:"access_key" gorm:"size:200"`
	SecretKey   string `json:"secret_key" gorm:"size:500"`
	Region      string `json:"region" gorm:"size:100"`
	Description string `json:"description" gorm:"size:500"`
	Status      int    `json:"status" gorm:"default:1"`
}

// CostRecord 费用记录
type CostRecord struct {
	core.BaseModel
	AccountID    string    `json:"account_id" gorm:"size:36;index"`
	AccountName  string    `json:"account_name" gorm:"size:100"`
	Provider     string    `json:"provider" gorm:"size:50"`
	ProductCode  string    `json:"product_code" gorm:"size:100;index"` // ecs, rds, oss
	ProductName  string    `json:"product_name" gorm:"size:200"`
	ResourceID   string    `json:"resource_id" gorm:"size:200"`
	ResourceName string    `json:"resource_name" gorm:"size:200"`
	Region       string    `json:"region" gorm:"size:100"`
	BillingDate  time.Time `json:"billing_date" gorm:"index"`
	Amount       float64   `json:"amount"`
	Currency     string    `json:"currency" gorm:"size:10"`
	PaymentType  string    `json:"payment_type" gorm:"size:50"` // prepay, postpay
	Tags         string    `json:"tags" gorm:"type:text"`
}

// CostBudget 预算
type CostBudget struct {
	core.BaseModel
	Name        string    `json:"name" gorm:"size:100"`
	AccountID   string    `json:"account_id" gorm:"size:36"`
	ProductCode string    `json:"product_code" gorm:"size:100"`
	BudgetType  string    `json:"budget_type" gorm:"size:50"` // monthly, quarterly, yearly
	Amount      float64   `json:"amount"`
	AlertAt     float64   `json:"alert_at"` // 告警阈值百分比
	StartDate   time.Time `json:"start_date"`
	EndDate     time.Time `json:"end_date"`
	Status      int       `json:"status" gorm:"default:1"`
}

// CostAlert 费用告警
type CostAlert struct {
	core.BaseModel
	BudgetID    string    `json:"budget_id" gorm:"size:36"`
	BudgetName  string    `json:"budget_name" gorm:"size:100"`
	AlertType   string    `json:"alert_type" gorm:"size:50"` // threshold, anomaly
	CurrentCost float64   `json:"current_cost"`
	BudgetCost  float64   `json:"budget_cost"`
	Percentage  float64   `json:"percentage"`
	Message     string    `json:"message" gorm:"size:500"`
	AlertAt     time.Time `json:"alert_at"`
	Status      int       `json:"status" gorm:"default:0"` // 0未处理 1已处理
}

// CostOptimization 优化建议
type CostOptimization struct {
	core.BaseModel
	AccountID    string  `json:"account_id" gorm:"size:36"`
	ResourceID   string  `json:"resource_id" gorm:"size:200"`
	ResourceName string  `json:"resource_name" gorm:"size:200"`
	ResourceType string  `json:"resource_type" gorm:"size:100"`
	OptType      string  `json:"opt_type" gorm:"size:50"` // downgrade, release, reserved
	CurrentSpec  string  `json:"current_spec" gorm:"size:200"`
	SuggestSpec  string  `json:"suggest_spec" gorm:"size:200"`
	CurrentCost  float64 `json:"current_cost"`
	SuggestCost  float64 `json:"suggest_cost"`
	SaveAmount   float64 `json:"save_amount"`
	Reason       string  `json:"reason" gorm:"size:500"`
	Status       int     `json:"status" gorm:"default:0"` // 0待处理 1已采纳 2已忽略
}
