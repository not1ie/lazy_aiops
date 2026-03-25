package orchestrator

import (
	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/pkg/plugin"
	"github.com/lazyautoops/lazy-auto-ops/plugins/workflow"
)

func init() {
	plugin.Register("orchestrator", func() plugin.Plugin {
		return &OrchestratorPlugin{}
	})
}

type OrchestratorPlugin struct {
	core   *core.Core
	cfg    map[string]interface{}
	engine *workflow.Engine
}

func (p *OrchestratorPlugin) Name() string    { return "orchestrator" }
func (p *OrchestratorPlugin) Version() string { return "1.0.0" }
func (p *OrchestratorPlugin) Description() string {
	return "编排中心 - 统一事件接入、规则路由、Runbook执行"
}

func (p *OrchestratorPlugin) Init(c *core.Core, cfg map[string]interface{}) error {
	p.core = c
	p.cfg = cfg
	p.engine = workflow.NewEngine(c.DB)
	return nil
}

func (p *OrchestratorPlugin) Start() error { return nil }
func (p *OrchestratorPlugin) Stop() error  { return nil }

func (p *OrchestratorPlugin) Migrate() error {
	return p.core.DB.AutoMigrate(
		&OrchestrationRule{},
		&OrchestrationEvent{},
		&OrchestrationDispatch{},
	)
}

func (p *OrchestratorPlugin) RegisterRoutes(g *gin.RouterGroup) {
	h := NewOrchestratorHandler(p.core.DB, p.engine)

	g.GET("/overview", h.GetOverview)

	rules := g.Group("/rules")
	{
		rules.GET("", h.ListRules)
		rules.POST("", h.CreateRule)
		rules.PUT("/:id", h.UpdateRule)
		rules.DELETE("/:id", h.DeleteRule)
	}

	events := g.Group("/events")
	{
		events.GET("", h.ListEvents)
		events.POST("/ingest", h.IngestEvent)
		events.POST("/webhook/:source", h.WebhookIngest)
	}

	g.GET("/dispatches", h.ListDispatches)
	g.POST("/runbooks/execute", h.ExecuteRunbook)
}
