package docker

import (
	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/pkg/plugin"
)

func init() {
	plugin.Register("docker", func() plugin.Plugin {
		return &DockerPlugin{}
	})
}

type DockerPlugin struct {
	core *core.Core
}

func (p *DockerPlugin) Name() string        { return "docker" }
func (p *DockerPlugin) Version() string     { return "1.0.0" }
func (p *DockerPlugin) Description() string { return "Docker管理 - 远程容器管理" }

func (p *DockerPlugin) Init(c *core.Core, cfg map[string]interface{}) error {
	p.core = c
	return nil
}

func (p *DockerPlugin) Start() error { return nil }
func (p *DockerPlugin) Stop() error  { return nil }

func (p *DockerPlugin) Migrate() error {
	return p.core.DB.AutoMigrate(&DockerHost{}, &DockerRegistry{}, &DockerRegistryLogin{})
}

func (p *DockerPlugin) RegisterRoutes(g *gin.RouterGroup) {
	h := NewDockerHandler(p.core.DB, p.core.Auth)
	g.GET("/hosts", h.ListHosts)
	g.POST("/hosts", h.AddHost)
	g.POST("/hosts/sync", h.SyncHosts) // 全局同步
	g.DELETE("/hosts/:id", h.DeleteHost)
	g.GET("/hosts/:id/info", h.GetHostInfo)
	g.POST("/hosts/:id/test", h.TestConnection) // 新增测试接口

	// Containers
	g.GET("/hosts/:id/containers", h.ListContainers)
	g.GET("/hosts/:id/containers/stats", h.ListContainerStats)
	g.POST("/hosts/:id/containers", h.CreateContainer) // 创建容器
	g.GET("/hosts/:id/containers/:container_id", h.InspectContainer)
	g.GET("/hosts/:id/containers/:container_id/logs", h.ContainerLogs)       // 日志
	g.POST("/hosts/:id/containers/:container_id/exec", h.ExecContainer)      // 执行命令
	g.GET("/hosts/:id/containers/:container_id/exec/ws", h.ExecContainerWS)  // WebSocket终端
	g.POST("/hosts/:id/containers/:container_id/:action", h.ContainerAction) // action: start, stop, restart, remove

	// Images
	g.GET("/hosts/:id/images", h.ListImages)
	g.POST("/hosts/:id/images/pull", h.PullImage) // 拉取镜像
	g.DELETE("/hosts/:id/images/:image_id", h.RemoveImage)
	g.POST("/hosts/:id/images/prune", h.PruneImages)
	g.POST("/hosts/:id/images/build", h.BuildImage)
	g.POST("/hosts/:id/images/load", h.LoadImage)

	// Networks
	g.GET("/hosts/:id/networks", h.ListNetworks)
	g.GET("/hosts/:id/networks/:network_id", h.InspectNetwork)
	g.GET("/hosts/:id/networks/usage", h.ListNetworkUsage)
	g.DELETE("/hosts/:id/networks/:network_id", h.RemoveNetwork)

	// Swarm
	g.GET("/hosts/:id/services", h.ListServices)
	g.POST("/hosts/:id/services", h.CreateService)
	g.GET("/hosts/:id/services/:service_id", h.InspectService)
	g.GET("/hosts/:id/services/:service_id/tasks", h.ListServiceTasks)
	g.GET("/hosts/:id/services/:service_id/logs", h.ServiceLogs)
	g.POST("/hosts/:id/services/:service_id/scale", h.ScaleService)
	g.POST("/hosts/:id/services/:service_id/update", h.UpdateService)
	g.POST("/hosts/:id/services/:service_id/update_image", h.UpdateServiceImage)
	g.POST("/hosts/:id/services/:service_id/restart", h.RestartService)
	g.POST("/hosts/:id/services/:service_id/rollback", h.RollbackService)
	g.DELETE("/hosts/:id/services/:service_id", h.RemoveService)
	g.GET("/hosts/:id/stacks", h.ListStacks)
	g.GET("/hosts/:id/stacks/:stack/services", h.ListStackServices)
	g.DELETE("/hosts/:id/stacks/:stack", h.RemoveStack)
	g.POST("/hosts/:id/stacks/deploy", h.DeployStack)
	g.POST("/hosts/:id/stacks/deploy/git", h.DeployStackFromGit)
	g.GET("/hosts/:id/nodes", h.ListNodes)
	g.GET("/hosts/:id/events", h.ListEvents)

	// Volumes
	g.GET("/hosts/:id/volumes", h.ListVolumes)
	g.GET("/hosts/:id/volumes/:volume", h.InspectVolume)
	g.GET("/hosts/:id/volumes/usage", h.ListVolumeUsage)
	g.POST("/hosts/:id/volumes", h.CreateVolume)
	g.DELETE("/hosts/:id/volumes/:volume", h.RemoveVolume)

	// Secrets
	g.GET("/hosts/:id/secrets", h.ListSecrets)
	g.GET("/hosts/:id/secrets/:secret", h.InspectSecret)
	g.POST("/hosts/:id/secrets", h.CreateSecret)
	g.DELETE("/hosts/:id/secrets/:secret", h.RemoveSecret)

	// Configs
	g.GET("/hosts/:id/configs", h.ListConfigs)
	g.GET("/hosts/:id/configs/:config", h.InspectConfig)
	g.POST("/hosts/:id/configs", h.CreateConfig)
	g.DELETE("/hosts/:id/configs/:config", h.RemoveConfig)

	// Registries
	g.GET("/registries", h.ListRegistries)
	g.POST("/registries", h.CreateRegistry)
	g.DELETE("/registries/:registry_id", h.DeleteRegistry)
	g.POST("/hosts/:id/registries/:registry_id/login", h.LoginRegistry)
	g.GET("/hosts/:id/registries", h.ListRegistriesForHost)
}
