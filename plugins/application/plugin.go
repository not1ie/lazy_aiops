package application

import (
	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/pkg/plugin"
)

func init() {
	plugin.Register("application", func() plugin.Plugin {
		return &AppPlugin{}
	})
}

type AppPlugin struct {
	core *core.Core
}

func (p *AppPlugin) Name() string        { return "application" }
func (p *AppPlugin) Version() string     { return "1.0.0" }
func (p *AppPlugin) Description() string { return "应用中心 - DevOps 核心服务管理" }

func (p *AppPlugin) Init(c *core.Core, cfg map[string]interface{}) error {
	p.core = c
	return nil
}

func (p *AppPlugin) Start() error { return nil }
func (p *AppPlugin) Stop() error  { return nil }

func (p *AppPlugin) Migrate() error {
	return p.core.DB.AutoMigrate(&Application{}, &AppEnvironment{})
}

func (p *AppPlugin) RegisterRoutes(g *gin.RouterGroup) {
	h := NewAppHandler(p.core.DB)
	g.GET("/apps", h.ListApps)
	g.POST("/apps", h.CreateApp)
	g.GET("/apps/:id/configs", h.GetAppConfigs)
	g.POST("/configs", h.CreateAppConfig)
}
