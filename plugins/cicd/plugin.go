package cicd

import (
	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/pkg/plugin"
)

func init() {
	plugin.Register("cicd", func() plugin.Plugin {
		return &CICDPlugin{}
	})
}

type CICDPlugin struct {
	core    *core.Core
	cfg     map[string]interface{}
	handler *CICDHandler
}

func (p *CICDPlugin) Name() string    { return "cicd" }
func (p *CICDPlugin) Version() string { return "1.0.0" }
func (p *CICDPlugin) Description() string {
	return "CI/CD集成 - Jenkins/GitLab/ArgoCD对接、定时发布"
}

func (p *CICDPlugin) Init(c *core.Core, cfg map[string]interface{}) error {
	p.core = c
	p.cfg = cfg
	return nil
}

func (p *CICDPlugin) Start() error { return nil }
func (p *CICDPlugin) Stop() error  { return nil }

func (p *CICDPlugin) Migrate() error {
	return p.core.DB.AutoMigrate(&CICDPipeline{}, &CICDJob{}, &CICDExecution{}, &CICDSchedule{}, &CICDRelease{})
}

func (p *CICDPlugin) RegisterRoutes(r *gin.RouterGroup) {
	p.handler = NewCICDHandler(p.core.DB, p.core.Config.JWT.Secret)

	// Pipeline管理
	r.GET("/credentials", p.handler.ListCredentials)
	r.GET("/pipelines", p.handler.ListPipelines)
	r.POST("/pipelines", p.handler.CreatePipeline)
	r.GET("/pipelines/:id", p.handler.GetPipeline)
	r.PUT("/pipelines/:id", p.handler.UpdatePipeline)
	r.DELETE("/pipelines/:id", p.handler.DeletePipeline)
	r.POST("/pipelines/:id/trigger", p.handler.TriggerPipeline)
	r.POST("/orders/:orderID/execute", p.handler.ExecuteByWorkOrder)
	r.POST("/pipelines/:id/sync", p.handler.SyncFromRemote)

	// 执行记录
	r.GET("/executions", p.handler.ListExecutions)
	r.GET("/executions/:id", p.handler.GetExecution)
	r.GET("/executions/:id/logs", p.handler.GetExecutionLogs)
	r.POST("/executions/:id/cancel", p.handler.CancelExecution)

	// 定时发布
	r.GET("/schedules", p.handler.ListSchedules)
	r.POST("/schedules", p.handler.CreateSchedule)
	r.PUT("/schedules/:id", p.handler.UpdateSchedule)
	r.DELETE("/schedules/:id", p.handler.DeleteSchedule)
	r.POST("/schedules/:id/toggle", p.handler.ToggleSchedule)

	// 发布管理
	r.GET("/releases", p.handler.ListReleases)
	r.POST("/releases", p.handler.CreateRelease)
	r.PUT("/releases/:id", p.handler.UpdateRelease)
	r.DELETE("/releases/:id", p.handler.DeleteRelease)

	// Webhook
	r.POST("/webhook/:provider", p.handler.HandleWebhook)
}
