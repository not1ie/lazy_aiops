package cmdb

import (
	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/internal/plugin"
)

func init() {
	plugin.Register("cmdb", func() plugin.Plugin {
		return &CMDBPlugin{}
	})
}

type CMDBPlugin struct {
	core *core.Core
	cfg  map[string]interface{}
}

func (p *CMDBPlugin) Name() string        { return "cmdb" }
func (p *CMDBPlugin) Version() string     { return "1.0.0" }
func (p *CMDBPlugin) Description() string { return "资产管理模块 - 主机、数据库、云资源管理" }

func (p *CMDBPlugin) Init(c *core.Core, cfg map[string]interface{}) error {
	p.core = c
	p.cfg = cfg
	return nil
}

func (p *CMDBPlugin) Start() error { return nil }
func (p *CMDBPlugin) Stop() error  { return nil }

func (p *CMDBPlugin) Migrate() error {
	return p.core.DB.AutoMigrate(&Host{}, &HostGroup{}, &Credential{})
}

func (p *CMDBPlugin) RegisterRoutes(g *gin.RouterGroup) {
	h := NewHostHandler(p.core.DB)

	// 主机管理
	hosts := g.Group("/hosts")
	{
		hosts.GET("", h.List)
		hosts.POST("", h.Create)
		hosts.GET("/:id", h.Get)
		hosts.PUT("/:id", h.Update)
		hosts.DELETE("/:id", h.Delete)
	}

	// 主机分组
	groups := g.Group("/groups")
	{
		groups.GET("", h.ListGroups)
		groups.POST("", h.CreateGroup)
	}

	// 凭据管理
	creds := g.Group("/credentials")
	{
		creds.GET("", h.ListCredentials)
		creds.POST("", h.CreateCredential)
	}
}
