package executor

import (
	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/pkg/plugin"
)

func init() {
	plugin.Register("executor", func() plugin.Plugin {
		return &ExecutorPlugin{}
	})
}

type ExecutorPlugin struct {
	core *core.Core
	cfg  map[string]interface{}
}

func (p *ExecutorPlugin) Name() string        { return "executor" }
func (p *ExecutorPlugin) Version() string     { return "1.0.0" }
func (p *ExecutorPlugin) Description() string { return "批量执行 - 多主机命令执行、实时输出" }

func (p *ExecutorPlugin) Init(c *core.Core, cfg map[string]interface{}) error {
	p.core = c
	p.cfg = cfg
	return nil
}

func (p *ExecutorPlugin) Start() error {
	p.initBuiltinTemplates()
	return nil
}

func (p *ExecutorPlugin) Stop() error { return nil }

func (p *ExecutorPlugin) Migrate() error {
	return p.core.DB.AutoMigrate(&BatchExecution{}, &BatchExecutionResult{}, &CommandTemplate{})
}

func (p *ExecutorPlugin) RegisterRoutes(g *gin.RouterGroup) {
	h := NewExecutorHandler(p.core.DB)

	// 批量执行
	g.POST("/execute", h.Execute)
	g.GET("/executions", h.ListExecutions)
	g.GET("/executions/:id", h.GetExecution)
	g.POST("/executions/:id/cancel", h.CancelExecution)
	g.GET("/executions/:id/results", h.GetResults)
	g.GET("/executions/:id/stream", h.StreamResults)

	// 命令模板
	templates := g.Group("/templates")
	{
		templates.GET("", h.ListTemplates)
		templates.POST("", h.CreateTemplate)
		templates.DELETE("/:id", h.DeleteTemplate)
	}
}

func (p *ExecutorPlugin) initBuiltinTemplates() {
	templates := []CommandTemplate{
		{Name: "查看磁盘使用", Category: "system", Content: "df -h", IsBuiltin: true},
		{Name: "查看内存使用", Category: "system", Content: "free -m", IsBuiltin: true},
		{Name: "查看CPU信息", Category: "system", Content: "top -bn1 | head -20", IsBuiltin: true},
		{Name: "查看进程列表", Category: "system", Content: "ps aux | head -20", IsBuiltin: true},
		{Name: "查看网络连接", Category: "network", Content: "netstat -tunlp", IsBuiltin: true},
		{Name: "查看系统日志", Category: "log", Content: "tail -100 /var/log/messages", IsBuiltin: true},
		{Name: "重启服务", Category: "service", Content: "systemctl restart {{.service}}", Variables: `{"service": "nginx"}`, IsBuiltin: true},
		{Name: "查看服务状态", Category: "service", Content: "systemctl status {{.service}}", Variables: `{"service": "nginx"}`, IsBuiltin: true},
	}

	for _, t := range templates {
		var count int64
		p.core.DB.Model(&CommandTemplate{}).Where("name = ? AND is_builtin = ?", t.Name, true).Count(&count)
		if count == 0 {
			p.core.DB.Create(&t)
		}
	}
}
