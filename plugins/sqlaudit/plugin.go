package sqlaudit

import (
	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/internal/plugin"
)

func init() {
	plugin.Register("sqlaudit", func() plugin.Plugin {
		return &SQLAuditPlugin{}
	})
}

type SQLAuditPlugin struct {
	core *core.Core
	cfg  map[string]interface{}
}

func (p *SQLAuditPlugin) Name() string        { return "sqlaudit" }
func (p *SQLAuditPlugin) Version() string     { return "1.0.0" }
func (p *SQLAuditPlugin) Description() string { return "SQL审计 - SQL工单、审核规则、执行审计" }

func (p *SQLAuditPlugin) Init(c *core.Core, cfg map[string]interface{}) error {
	p.core = c
	p.cfg = cfg
	return nil
}

func (p *SQLAuditPlugin) Start() error {
	// 初始化默认审核规则
	p.initDefaultRules()
	return nil
}

func (p *SQLAuditPlugin) Stop() error { return nil }

func (p *SQLAuditPlugin) Migrate() error {
	return p.core.DB.AutoMigrate(&DBInstance{}, &SQLWorkOrder{}, &SQLAuditLog{}, &SQLAuditRule{})
}

func (p *SQLAuditPlugin) RegisterRoutes(g *gin.RouterGroup) {
	h := NewSQLAuditHandler(p.core.DB)

	// 数据库实例
	instances := g.Group("/instances")
	{
		instances.GET("", h.ListInstances)
		instances.POST("", h.CreateInstance)
		instances.GET("/:id", h.GetInstance)
		instances.PUT("/:id", h.UpdateInstance)
		instances.DELETE("/:id", h.DeleteInstance)
		instances.POST("/:id/test", h.TestConnection)
	}

	// SQL工单
	orders := g.Group("/orders")
	{
		orders.GET("", h.ListWorkOrders)
		orders.POST("", h.CreateWorkOrder)
		orders.GET("/:id", h.GetWorkOrder)
		orders.POST("/:id/review", h.ReviewWorkOrder)
		orders.POST("/:id/execute", h.ExecuteWorkOrder)
	}

	// 审计日志
	g.GET("/logs", h.ListAuditLogs)

	// 审核规则
	rules := g.Group("/rules")
	{
		rules.GET("", h.ListRules)
		rules.POST("", h.CreateRule)
		rules.PUT("/:id", h.UpdateRule)
		rules.DELETE("/:id", h.DeleteRule)
	}

	// SQL分析
	g.POST("/analyze", h.AnalyzeSQL)
	g.GET("/statistics", h.GetStatistics)
}

func (p *SQLAuditPlugin) initDefaultRules() {
	defaultRules := []SQLAuditRule{
		{
			Name:        "禁止SELECT *",
			Type:        "performance",
			Level:       1,
			Pattern:     `(?i)select\s+\*`,
			Message:     "不建议使用 SELECT *，请明确指定需要的字段",
			Suggestion:  "将 * 替换为具体的字段名",
			Enabled:     true,
			Description: "SELECT * 会查询所有字段，可能导致不必要的数据传输",
		},
		{
			Name:        "DELETE无WHERE条件",
			Type:        "security",
			Level:       2,
			Pattern:     `(?i)delete\s+from\s+\w+\s*;?\s*$`,
			Message:     "DELETE 语句缺少 WHERE 条件",
			Suggestion:  "添加 WHERE 条件限制删除范围",
			Enabled:     true,
			Description: "无条件的 DELETE 会删除表中所有数据",
		},
		{
			Name:        "UPDATE无WHERE条件",
			Type:        "security",
			Level:       2,
			Pattern:     `(?i)update\s+\w+\s+set\s+[^;]+;?\s*$`,
			Message:     "UPDATE 语句缺少 WHERE 条件",
			Suggestion:  "添加 WHERE 条件限制更新范围",
			Enabled:     true,
			Description: "无条件的 UPDATE 会更新表中所有数据",
		},
	}

	for _, rule := range defaultRules {
		var count int64
		p.core.DB.Model(&SQLAuditRule{}).Where("name = ?", rule.Name).Count(&count)
		if count == 0 {
			p.core.DB.Create(&rule)
		}
	}
}
