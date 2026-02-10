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
	return p.core.DB.AutoMigrate(&DockerHost{})
}

func (p *DockerPlugin) RegisterRoutes(g *gin.RouterGroup) {
	h := NewDockerHandler(p.core.DB)
	g.GET("/hosts", h.ListHosts)
	g.POST("/hosts", h.AddHost)
	g.POST("/hosts/sync", h.SyncHosts) // 全局同步
	g.DELETE("/hosts/:id", h.DeleteHost)
	g.GET("/hosts/:id/info", h.GetHostInfo)
	g.POST("/hosts/:id/test", h.TestConnection) // 新增测试接口
	
	// Containers
	g.GET("/hosts/:id/containers", h.ListContainers)
	g.POST("/hosts/:id/containers", h.CreateContainer) // 创建容器
	g.GET("/hosts/:id/containers/:container_id", h.InspectContainer)
	g.GET("/hosts/:id/containers/:container_id/logs", h.ContainerLogs) // 日志
	g.POST("/hosts/:id/containers/:container_id/exec", h.ExecContainer) // 执行命令
	g.POST("/hosts/:id/containers/:container_id/:action", h.ContainerAction) // action: start, stop, restart, remove
	
	// Images
	g.GET("/hosts/:id/images", h.ListImages)
	g.POST("/hosts/:id/images/pull", h.PullImage) // 拉取镜像
	g.DELETE("/hosts/:id/images/:image_id", h.RemoveImage)
	
	// Networks
	g.GET("/hosts/:id/networks", h.ListNetworks)

	// Swarm
	g.GET("/hosts/:id/services", h.ListServices)
	g.GET("/hosts/:id/services/:service_id", h.InspectService)
	g.GET("/hosts/:id/services/:service_id/tasks", h.ListServiceTasks)
	g.GET("/hosts/:id/stacks", h.ListStacks)
	g.GET("/hosts/:id/stacks/:stack/services", h.ListStackServices)
}
