package monitor

import (
	"time"
	
	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/pkg/plugin"
)

func init() {
	plugin.Register("monitor", func() plugin.Plugin {
		return &MonitorPlugin{}
	})
}

type MonitorPlugin struct {
	core      *core.Core
	cfg       map[string]interface{}
	collector *Collector
}

func (p *MonitorPlugin) Name() string        { return "monitor" }
func (p *MonitorPlugin) Version() string     { return "1.0.0" }
func (p *MonitorPlugin) Description() string { return "监控告警模块 - 域名监控、主机监控、告警管理" }

func (p *MonitorPlugin) Init(c *core.Core, cfg map[string]interface{}) error {
	p.core = c
	p.cfg = cfg
	
	// 获取采集间隔配置，默认30秒
	interval := 30 * time.Second
	if v, ok := cfg["interval"].(int); ok {
		interval = time.Duration(v) * time.Second
	}
	
	p.collector = NewCollector(c.DB, interval)
	return nil
}

func (p *MonitorPlugin) Start() error {
	if p.collector != nil {
		return p.collector.Start()
	}
	return nil
}

func (p *MonitorPlugin) Stop() error {
	if p.collector != nil {
		return p.collector.Stop()
	}
	return nil
}

func (p *MonitorPlugin) Migrate() error {
	return p.core.DB.AutoMigrate(&DomainMonitor{}, &AlertRule{}, &AlertRecord{}, &MetricRecord{})
}

func (p *MonitorPlugin) RegisterRoutes(g *gin.RouterGroup) {
	h := NewMonitorHandler(p.core.DB, p.collector)

	// 域名监控
	domains := g.Group("/domains")
	{
		domains.GET("", h.ListDomains)
		domains.POST("", h.CreateDomain)
		domains.DELETE("/:id", h.DeleteDomain)
	}

	// 告警规则
	rules := g.Group("/rules")
	{
		rules.GET("", h.ListRules)
		rules.POST("", h.CreateRule)
	}

	// 告警记录
	alerts := g.Group("/alerts")
	{
		alerts.GET("", h.ListAlerts)
	}
	
	// 监控指标
	g.GET("/metrics", h.GetMetrics)
	g.GET("/metrics/realtime", h.GetRealtimeMetrics)
	g.GET("/metrics/history", h.GetMetricsHistory)
	g.GET("/servers", h.GetServerMetrics)
	g.GET("/chart", h.GetChartData)
}
