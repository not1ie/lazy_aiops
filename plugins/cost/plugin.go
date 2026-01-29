package cost

import (
	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/internal/plugin"
)

func init() {
	plugin.Register("cost", func() plugin.Plugin {
		return &CostPlugin{}
	})
}

type CostPlugin struct {
	core    *core.Core
	cfg     map[string]interface{}
	handler *CostHandler
}

func (p *CostPlugin) Name() string        { return "cost" }
func (p *CostPlugin) Version() string     { return "1.0.0" }
func (p *CostPlugin) Description() string { return "成本分析 - 云费用统计、预算管理、优化建议" }

func (p *CostPlugin) Init(c *core.Core, cfg map[string]interface{}) error {
	p.core = c
	p.cfg = cfg
	return nil
}

func (p *CostPlugin) Start() error { return nil }
func (p *CostPlugin) Stop() error  { return nil }

func (p *CostPlugin) Migrate() error {
	return p.core.DB.AutoMigrate(&CloudAccount{}, &CostRecord{}, &CostBudget{}, &CostAlert{}, &CostOptimization{})
}

func (p *CostPlugin) RegisterRoutes(r *gin.RouterGroup) {
	p.handler = NewCostHandler(p.core.DB)

	// 账号管理
	r.GET("/accounts", p.handler.ListAccounts)
	r.POST("/accounts", p.handler.CreateAccount)
	r.PUT("/accounts/:id", p.handler.UpdateAccount)
	r.DELETE("/accounts/:id", p.handler.DeleteAccount)
	r.POST("/accounts/:id/sync", p.handler.SyncCost)

	// 费用查询
	r.GET("/summary", p.handler.GetCostSummary)
	r.GET("/records", p.handler.ListCostRecords)
	r.GET("/trend", p.handler.GetCostTrend)
	r.GET("/top-resources", p.handler.GetTopResources)

	// 预算管理
	r.GET("/budgets", p.handler.ListBudgets)
	r.POST("/budgets", p.handler.CreateBudget)
	r.PUT("/budgets/:id", p.handler.UpdateBudget)
	r.DELETE("/budgets/:id", p.handler.DeleteBudget)
	r.GET("/budgets/status", p.handler.GetBudgetStatus)

	// 告警
	r.GET("/alerts", p.handler.ListAlerts)
	r.POST("/alerts/:id/ack", p.handler.AckAlert)

	// 优化建议
	r.GET("/optimizations", p.handler.ListOptimizations)
	r.PUT("/optimizations/:id", p.handler.UpdateOptimization)
	r.POST("/optimizations/analyze", p.handler.AnalyzeOptimization)
}
