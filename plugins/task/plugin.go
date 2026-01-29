package task

import (
	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/pkg/plugin"
)

func init() {
	plugin.Register("task", func() plugin.Plugin {
		return &TaskPlugin{}
	})
}

type TaskPlugin struct {
	core      *core.Core
	cfg       map[string]interface{}
	scheduler *Scheduler
}

func (p *TaskPlugin) Name() string        { return "task" }
func (p *TaskPlugin) Version() string     { return "1.0.0" }
func (p *TaskPlugin) Description() string { return "任务调度模块 - 定时任务、脚本执行、Ansible" }

func (p *TaskPlugin) Init(c *core.Core, cfg map[string]interface{}) error {
	p.core = c
	p.cfg = cfg
	p.scheduler = NewScheduler(c.DB)
	return nil
}

func (p *TaskPlugin) Start() error {
	if p.scheduler != nil {
		return p.scheduler.Start()
	}
	return nil
}

func (p *TaskPlugin) Stop() error {
	if p.scheduler != nil {
		return p.scheduler.Stop()
	}
	return nil
}

func (p *TaskPlugin) Migrate() error {
	return p.core.DB.AutoMigrate(&Task{}, &TaskExecution{})
}

func (p *TaskPlugin) RegisterRoutes(g *gin.RouterGroup) {
	h := NewTaskHandler(p.core.DB, p.scheduler)

	// 任务管理
	tasks := g.Group("/tasks")
	{
		tasks.GET("", h.List)
		tasks.POST("", h.Create)
		tasks.GET("/:id", h.Get)
		tasks.PUT("/:id", h.Update)
		tasks.DELETE("/:id", h.Delete)
		tasks.POST("/:id/run", h.Run)
		tasks.POST("/:id/enable", h.Enable)
		tasks.POST("/:id/disable", h.Disable)
	}

	// 执行记录
	executions := g.Group("/executions")
	{
		executions.GET("", h.ListExecutions)
		executions.GET("/:id", h.GetExecution)
	}
}
