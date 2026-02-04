package nacos

import (
	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/pkg/plugin"
)

func init() {
	plugin.Register("nacos", func() plugin.Plugin {
		return &NacosPlugin{}
	})
}

type NacosPlugin struct {
	core    *core.Core
	cfg     map[string]interface{}
	handler *NacosHandler
	scheduler *Scheduler
}

func (p *NacosPlugin) Name() string        { return "nacos" }
func (p *NacosPlugin) Version() string     { return "1.0.0" }
func (p *NacosPlugin) Description() string { return "Nacos配置中心 - 配置管理、服务发现、命名空间" }

func (p *NacosPlugin) Init(c *core.Core, cfg map[string]interface{}) error {
	p.core = c
	p.cfg = cfg
	p.handler = NewNacosHandler(c.DB, nil)
	p.scheduler = NewScheduler(c.DB, p.handler)
	p.handler.scheduler = p.scheduler
	return nil
}

func (p *NacosPlugin) Start() error {
	if p.scheduler != nil {
		return p.scheduler.Start()
	}
	return nil
}
func (p *NacosPlugin) Stop() error {
	if p.scheduler != nil {
		return p.scheduler.Stop()
	}
	return nil
}

func (p *NacosPlugin) Migrate() error {
	return p.core.DB.AutoMigrate(&NacosServer{}, &NacosConfig{}, &NacosConfigHistory{}, &NacosService{}, &NacosInstance{}, &NacosNamespace{}, &NacosSyncSchedule{})
}

func (p *NacosPlugin) RegisterRoutes(r *gin.RouterGroup) {
	// 服务器管理
	r.GET("/servers", p.handler.ListServers)
	r.POST("/servers", p.handler.CreateServer)
	r.GET("/servers/:id", p.handler.GetServer)
	r.PUT("/servers/:id", p.handler.UpdateServer)
	r.DELETE("/servers/:id", p.handler.DeleteServer)
	r.POST("/servers/:id/test", p.handler.TestConnection)
	r.POST("/servers/:id/sync-configs", p.handler.SyncConfigs)
	r.POST("/servers/:id/sync-services", p.handler.SyncServices)
	r.GET("/servers/:id/namespaces", p.handler.ListNamespaces)

	// 配置管理
	r.GET("/configs", p.handler.ListConfigs)
	r.GET("/configs/:id", p.handler.GetConfig)
	r.PUT("/configs/:id", p.handler.UpdateConfig)
	r.GET("/configs/:id/history", p.handler.GetConfigHistory)
	r.GET("/configs/:id/compare", p.handler.CompareConfig)
	r.POST("/configs/history/:history_id/rollback", p.handler.RollbackConfig)

	// 服务发现
	r.GET("/services", p.handler.ListServices)
	r.GET("/services/instances", p.handler.GetServiceInstances)

	// 同步计划
	r.GET("/schedules", p.handler.ListSyncSchedules)
	r.POST("/schedules", p.handler.CreateSyncSchedule)
	r.PUT("/schedules/:id", p.handler.UpdateSyncSchedule)
	r.DELETE("/schedules/:id", p.handler.DeleteSyncSchedule)
	r.POST("/schedules/:id/toggle", p.handler.ToggleSyncSchedule)
}
