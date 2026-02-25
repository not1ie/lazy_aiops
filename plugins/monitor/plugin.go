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
	return p.core.DB.AutoMigrate(&DomainMonitor{}, &AlertRule{}, &AlertRecord{}, &MetricRecord{}, &MonitorSetting{}, &AgentHeartbeat{}, &AgentHeartbeatRecord{}, &PromQueryHistory{}, &DashboardTemplate{})
}

func (p *MonitorPlugin) RegisterRoutes(g *gin.RouterGroup) {
	promURL, _ := p.cfg["prometheus_url"].(string)
	pushURL, _ := p.cfg["pushgateway_url"].(string)
	agentSecret, _ := p.cfg["agent_secret"].(string)
	agentTimeout := 90 * time.Second
	if v, ok := p.cfg["agent_timeout"].(int); ok && v > 0 {
		agentTimeout = time.Duration(v) * time.Second
	}
	h := NewMonitorHandler(p.core.DB, p.collector, promURL, pushURL, agentSecret, agentTimeout)

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

	// Prometheus / Pushgateway
	g.GET("/prometheus/query", h.ProxyPromQuery)
	g.GET("/prometheus/query_range", h.ProxyPromQueryRange)
	g.GET("/prometheus/buildinfo", h.ProxyPromBuildInfo)
	g.GET("/prometheus/runtimeinfo", h.ProxyPromRuntimeInfo)
	g.GET("/dashboards", h.ListDashboards)
	g.POST("/dashboards", h.SaveDashboard)
	g.PUT("/dashboards/:id", h.UpdateDashboard)
	g.DELETE("/dashboards/:id", h.DeleteDashboard)
	g.GET("/pushgateway/metrics", h.ProxyPushgatewayMetrics)
	g.GET("/prometheus/history", h.ListPromHistory)
	g.POST("/prometheus/history", h.CreatePromHistory)
	g.PUT("/prometheus/history/:id", h.UpdatePromHistory)
	g.GET("/settings", h.ListSettings)
	g.POST("/settings", h.CreateSetting)
	g.PUT("/settings/:id", h.UpdateSetting)
	g.DELETE("/settings/:id", h.DeleteSetting)
	g.POST("/settings/:id/activate", h.ActivateSetting)
	g.POST("/settings/:id/test", h.TestSetting)

	// Agent heartbeat
	g.POST("/agents/heartbeat", h.AgentHeartbeat)
	g.GET("/agents", h.ListAgents)
	g.GET("/agents/:id", h.GetAgent)
	g.GET("/agents/:id/history", h.GetAgentHistory)
}
