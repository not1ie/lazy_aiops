package workflow

import (
	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/internal/plugin"
)

func init() {
	plugin.Register("workflow", func() plugin.Plugin {
		return &WorkflowPlugin{}
	})
}

type WorkflowPlugin struct {
	core   *core.Core
	cfg    map[string]interface{}
	engine *Engine
}

func (p *WorkflowPlugin) Name() string        { return "workflow" }
func (p *WorkflowPlugin) Version() string     { return "1.0.0" }
func (p *WorkflowPlugin) Description() string { return "运维编排 - 可视化工作流、自动化任务" }

func (p *WorkflowPlugin) Init(c *core.Core, cfg map[string]interface{}) error {
	p.core = c
	p.cfg = cfg
	p.engine = NewEngine(c.DB)
	return nil
}

func (p *WorkflowPlugin) Start() error {
	p.initBuiltinTemplates()
	return nil
}

func (p *WorkflowPlugin) Stop() error { return nil }

func (p *WorkflowPlugin) Migrate() error {
	return p.core.DB.AutoMigrate(&Workflow{}, &WorkflowExecution{}, &WorkflowNodeExecution{}, &WorkflowTemplate{})
}

func (p *WorkflowPlugin) RegisterRoutes(g *gin.RouterGroup) {
	h := NewWorkflowHandler(p.core.DB, p.engine)

	// 工作流
	workflows := g.Group("/workflows")
	{
		workflows.GET("", h.ListWorkflows)
		workflows.POST("", h.CreateWorkflow)
		workflows.GET("/:id", h.GetWorkflow)
		workflows.PUT("/:id", h.UpdateWorkflow)
		workflows.DELETE("/:id", h.DeleteWorkflow)
		workflows.POST("/:id/execute", h.ExecuteWorkflow)
	}

	// 执行记录
	executions := g.Group("/executions")
	{
		executions.GET("", h.ListExecutions)
		executions.GET("/:id", h.GetExecution)
		executions.POST("/:id/cancel", h.CancelExecution)
	}

	// 模板
	templates := g.Group("/templates")
	{
		templates.GET("", h.ListTemplates)
		templates.POST("/:id/create", h.CreateFromTemplate)
	}

	// Webhook触发
	g.POST("/webhook/:id", h.WebhookTrigger)

	// 验证
	g.POST("/validate", h.ValidateDefinition)

	// 统计
	g.GET("/stats", h.GetStats)
}

func (p *WorkflowPlugin) initBuiltinTemplates() {
	templates := []WorkflowTemplate{
		{
			Name:        "服务重启",
			Category:    "deploy",
			Description: "重启指定服务",
			Icon:        "refresh",
			IsBuiltin:   true,
			Definition: `{
				"nodes": [
					{"id": "start", "type": "start", "name": "开始", "next": ["stop"]},
					{"id": "stop", "type": "shell", "name": "停止服务", "config": {"script": "systemctl stop {{.service}}"}, "next": ["wait"]},
					{"id": "wait", "type": "wait", "name": "等待5秒", "config": {"seconds": 5}, "next": ["start_svc"]},
					{"id": "start_svc", "type": "shell", "name": "启动服务", "config": {"script": "systemctl start {{.service}}"}, "next": ["check"]},
					{"id": "check", "type": "shell", "name": "检查状态", "config": {"script": "systemctl status {{.service}}"}, "next": ["notify"]},
					{"id": "notify", "type": "notify", "name": "发送通知", "config": {"title": "服务重启完成", "content": "{{.service}} 已重启"}, "next": ["end"]},
					{"id": "end", "type": "end", "name": "结束"}
				]
			}`,
			Variables: `{"service": "nginx"}`,
		},
		{
			Name:        "健康检查",
			Category:    "monitor",
			Description: "检查服务健康状态",
			Icon:        "heart",
			IsBuiltin:   true,
			Definition: `{
				"nodes": [
					{"id": "start", "type": "start", "name": "开始", "next": ["http_check"]},
					{"id": "http_check", "type": "http", "name": "HTTP检查", "config": {"method": "GET", "url": "{{.url}}"}, "next": ["condition"]},
					{"id": "condition", "type": "condition", "name": "判断结果", "conditions": [{"expression": "success", "target": "end"}, {"expression": "default", "target": "alert"}]},
					{"id": "alert", "type": "notify", "name": "发送告警", "config": {"title": "健康检查失败", "content": "{{.url}} 不可访问"}, "next": ["end"]},
					{"id": "end", "type": "end", "name": "结束"}
				]
			}`,
			Variables: `{"url": "http://localhost:8080/health"}`,
		},
	}

	for _, t := range templates {
		var count int64
		p.core.DB.Model(&WorkflowTemplate{}).Where("name = ? AND is_builtin = ?", t.Name, true).Count(&count)
		if count == 0 {
			p.core.DB.Create(&t)
		}
	}
}
