package alert

import (
	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/internal/plugin"
)

func init() {
	plugin.Register("alert", func() plugin.Plugin {
		return &AlertPlugin{}
	})
}

type AlertPlugin struct {
	core       *core.Core
	cfg        map[string]interface{}
	aggregator *Aggregator
}

func (p *AlertPlugin) Name() string        { return "alert" }
func (p *AlertPlugin) Version() string     { return "1.0.0" }
func (p *AlertPlugin) Description() string { return "告警中心 - AI智能降噪、聚合、静默" }

func (p *AlertPlugin) Init(c *core.Core, cfg map[string]interface{}) error {
	p.core = c
	p.cfg = cfg

	interval := 60
	if v, ok := cfg["aggregate_interval"].(int); ok {
		interval = v
	}
	p.aggregator = NewAggregator(c.DB, interval)
	return nil
}

func (p *AlertPlugin) Start() error { return nil }
func (p *AlertPlugin) Stop() error  { return nil }

func (p *AlertPlugin) Migrate() error {
	return p.core.DB.AutoMigrate(&AlertRule{}, &Alert{}, &AlertSilence{}, &AlertHistory{})
}

func (p *AlertPlugin) RegisterRoutes(g *gin.RouterGroup) {
	h := NewAlertHandler(p.core.DB, p.aggregator)

	// 规则
	rules := g.Group("/rules")
	{
		rules.GET("", h.ListRules)
		rules.POST("", h.CreateRule)
		rules.PUT("/:id", h.UpdateRule)
		rules.DELETE("/:id", h.DeleteRule)
	}

	// 告警
	alerts := g.Group("/alerts")
	{
		alerts.GET("", h.ListAlerts)
		alerts.GET("/:id", h.GetAlert)
		alerts.POST("/:id/ack", h.AckAlert)
		alerts.POST("/:id/resolve", h.ResolveAlert)
	}

	// Webhook接收
	g.POST("/webhook", h.ReceiveAlert)

	// 统计
	g.GET("/stats", h.GetStats)

	// 静默
	silences := g.Group("/silences")
	{
		silences.GET("", h.ListSilences)
		silences.POST("", h.CreateSilence)
		silences.DELETE("/:id", h.DeleteSilence)
	}
}
