package firewall

import (
	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/internal/plugin"
)

func init() {
	plugin.Register("firewall", func() plugin.Plugin {
		return &FirewallPlugin{}
	})
}

type FirewallPlugin struct {
	core *core.Core
	cfg  map[string]interface{}
}

func (p *FirewallPlugin) Name() string        { return "firewall" }
func (p *FirewallPlugin) Version() string     { return "1.0.0" }
func (p *FirewallPlugin) Description() string { return "防火墙管理 - SNMP监控、配置管理" }

func (p *FirewallPlugin) Init(c *core.Core, cfg map[string]interface{}) error {
	p.core = c
	p.cfg = cfg
	return nil
}

func (p *FirewallPlugin) Start() error { return nil }
func (p *FirewallPlugin) Stop() error  { return nil }

func (p *FirewallPlugin) Migrate() error {
	return p.core.DB.AutoMigrate(&Firewall{}, &FirewallRule{}, &SNMPMetric{})
}

func (p *FirewallPlugin) RegisterRoutes(g *gin.RouterGroup) {
	h := NewFirewallHandler(p.core.DB)

	// 防火墙设备
	devices := g.Group("/devices")
	{
		devices.GET("", h.ListDevices)
		devices.POST("", h.CreateDevice)
		devices.GET("/:id", h.GetDevice)
		devices.PUT("/:id", h.UpdateDevice)
		devices.DELETE("/:id", h.DeleteDevice)
	}

	// SNMP
	g.POST("/devices/:id/snmp/test", h.TestSNMP)
	g.GET("/devices/:id/snmp/metrics", h.GetSNMPMetrics)
	g.POST("/devices/:id/snmp/collect", h.CollectSNMP)

	// 规则管理
	rules := g.Group("/devices/:id/rules")
	{
		rules.GET("", h.ListRules)
		rules.POST("", h.CreateRule)
		rules.DELETE("/:rule_id", h.DeleteRule)
	}
}
