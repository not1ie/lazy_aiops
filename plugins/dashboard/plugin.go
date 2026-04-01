package dashboard

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/pkg/plugin"
)

func init() {
	plugin.Register("dashboard", func() plugin.Plugin {
		return &DashboardPlugin{}
	})
}

type DashboardPlugin struct {
	core         *core.Core
	cfg          map[string]interface{}
	handler      *DashboardHandler
	agentTimeout time.Duration
}

func (p *DashboardPlugin) Name() string    { return "dashboard" }
func (p *DashboardPlugin) Version() string { return "1.0.0" }
func (p *DashboardPlugin) Description() string {
	return "全局仪表盘聚合 - 统一状态快照与健康评分输入"
}

func (p *DashboardPlugin) Init(c *core.Core, cfg map[string]interface{}) error {
	p.core = c
	p.cfg = cfg
	p.agentTimeout = parseDashboardAgentTimeout(cfg)
	return nil
}

func (p *DashboardPlugin) Start() error { return nil }
func (p *DashboardPlugin) Stop() error  { return nil }

func (p *DashboardPlugin) Migrate() error { return nil }

func (p *DashboardPlugin) RegisterRoutes(g *gin.RouterGroup) {
	p.handler = NewDashboardHandler(p.core.DB, p.agentTimeout)
	g.GET("/overview", p.handler.GetOverview)
}

func parseDashboardAgentTimeout(cfg map[string]interface{}) time.Duration {
	const fallback = 90 * time.Second
	if cfg == nil {
		return fallback
	}
	raw, ok := cfg["agent_timeout_seconds"]
	if !ok {
		raw = cfg["agent_timeout"]
	}
	if !ok && raw == nil {
		return fallback
	}

	parse := func(s string) time.Duration {
		n, err := strconv.Atoi(s)
		if err != nil || n <= 0 {
			return fallback
		}
		if n < 30 {
			n = 30
		}
		if n > 3600 {
			n = 3600
		}
		return time.Duration(n) * time.Second
	}

	switch v := raw.(type) {
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
