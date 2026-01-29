package docker

import (
	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/internal/plugin"
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
	g.DELETE("/hosts/:id", h.DeleteHost)
	g.GET("/hosts/:id/containers", h.ListContainers)
	g.POST("/hosts/:id/containers/:container_id/:action", h.Action)
}
