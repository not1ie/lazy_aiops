package firewall

import (
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/pkg/plugin"
)

func init() {
	plugin.Register("firewall", func() plugin.Plugin {
		return &FirewallPlugin{}
	})
}

type FirewallPlugin struct {
	core         *core.Core
	cfg          map[string]interface{}
	statusTicker *time.Ticker
	stopCh       chan struct{}
	wg           sync.WaitGroup
}

func (p *FirewallPlugin) Name() string        { return "firewall" }
func (p *FirewallPlugin) Version() string     { return "1.0.0" }
func (p *FirewallPlugin) Description() string { return "防火墙管理 - SNMP监控、配置管理" }

func (p *FirewallPlugin) Init(c *core.Core, cfg map[string]interface{}) error {
	p.core = c
	p.cfg = cfg
	return nil
}

func (p *FirewallPlugin) Start() error {
	handler := NewFirewallHandler(p.core.DB)
	interval := p.statusSyncInterval()
	p.statusTicker = time.NewTicker(interval)
	p.stopCh = make(chan struct{})
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		if _, err := handler.syncAllDeviceStatus(); err != nil {
			log.Printf("[Firewall] device status bootstrap sync failed: %v", err)
		}
		for {
			select {
			case <-p.stopCh:
				return
			case <-p.statusTicker.C:
				if _, err := handler.syncAllDeviceStatus(); err != nil {
					log.Printf("[Firewall] device status auto-sync failed: %v", err)
				}
			}
		}
	}()
	return nil
}

func (p *FirewallPlugin) Stop() error {
	if p.statusTicker != nil {
		p.statusTicker.Stop()
		p.statusTicker = nil
	}
	if p.stopCh != nil {
		close(p.stopCh)
		p.stopCh = nil
	}
	p.wg.Wait()
	return nil
}

func (p *FirewallPlugin) statusSyncInterval() time.Duration {
	const fallback = 75 * time.Second
	if p.cfg == nil {
		return fallback
	}
	value, ok := p.cfg["status_sync_interval_seconds"]
	if !ok {
		return fallback
	}
	parse := func(raw string) time.Duration {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			return fallback
		}
		if n < 10 {
			n = 10
		}
		if n > 300 {
			n = 300
		}
		return time.Duration(n) * time.Second
	}
	switch v := value.(type) {
	case int:
		return parse(strconv.Itoa(v))
	case int64:
		return parse(strconv.FormatInt(v, 10))
	case float64:
		return parse(strconv.Itoa(int(v)))
	case string:
		return parse(v)
	default:
		return fallback
	}
}

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
