package cmdb

import (
	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/pkg/plugin"
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

func (p *CMDBPlugin) Name() string    { return "cmdb" }
func (p *CMDBPlugin) Version() string { return "1.0.0" }
func (p *CMDBPlugin) Description() string {
	return "资产管理模块 - 主机、数据库、云资源管理"
}

func (p *CMDBPlugin) Init(c *core.Core, cfg map[string]interface{}) error {
	p.core = c
	p.cfg = cfg
	return nil
}

func (p *CMDBPlugin) Start() error { return nil }
func (p *CMDBPlugin) Stop() error  { return nil }

func (p *CMDBPlugin) Migrate() error {
	return p.core.DB.AutoMigrate(
		&Host{},
		&HostGroup{},
		&Credential{},
		&DatabaseAsset{},
		&CloudAccount{},
		&CloudResource{},
	)
}

func (p *CMDBPlugin) RegisterRoutes(g *gin.RouterGroup) {
	h := NewHostHandler(p.core.DB, p.core.Config.JWT.Secret)

	// 主机管理
	hosts := g.Group("/hosts")
	{
		hosts.GET("", h.List)
		hosts.POST("", h.Create)
		hosts.GET("/:id", h.Get)
		hosts.POST("/:id/test", h.TestHost)
		hosts.PUT("/:id", h.Update)
		hosts.DELETE("/:id", h.Delete)
	}

	// 主机分组
	groups := g.Group("/groups")
	{
		groups.GET("", h.ListGroups)
		groups.POST("", h.CreateGroup)
		groups.PUT("/:id", h.UpdateGroup)
		groups.DELETE("/:id", h.DeleteGroup)
	}

	// 凭据管理
	creds := g.Group("/credentials")
	{
		creds.GET("", h.ListCredentials)
		creds.POST("", h.CreateCredential)
		creds.POST("/:id/test", h.TestCredential)
		creds.PUT("/:id", h.UpdateCredential)
		creds.DELETE("/:id", h.DeleteCredential)
	}

	// 数据库资产
	databases := g.Group("/databases")
	{
		databases.GET("", h.ListDatabases)
		databases.POST("", h.CreateDatabase)
		databases.GET("/:id", h.GetDatabase)
		databases.POST("/:id/test", h.TestDatabase)
		databases.PUT("/:id", h.UpdateDatabase)
		databases.DELETE("/:id", h.DeleteDatabase)
	}

	// 云资源
	cloudAccounts := g.Group("/cloud/accounts")
	{
		cloudAccounts.GET("", h.ListCloudAccounts)
		cloudAccounts.POST("", h.CreateCloudAccount)
		cloudAccounts.GET("/:id", h.GetCloudAccount)
		cloudAccounts.POST("/:id/test", h.TestCloudAccount)
		cloudAccounts.PUT("/:id", h.UpdateCloudAccount)
		cloudAccounts.DELETE("/:id", h.DeleteCloudAccount)
	}
	cloudResources := g.Group("/cloud/resources")
	{
		cloudResources.GET("", h.ListCloudResources)
		cloudResources.POST("", h.CreateCloudResource)
		cloudResources.GET("/:id", h.GetCloudResource)
		cloudResources.PUT("/:id", h.UpdateCloudResource)
		cloudResources.DELETE("/:id", h.DeleteCloudResource)
	}
}
