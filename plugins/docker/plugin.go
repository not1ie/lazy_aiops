package docker

import (
	"log"
	"strconv"
	"sync"
	"time"

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
	core         *core.Core
	cfg          map[string]interface{}
	statusTicker *time.Ticker
	stopCh       chan struct{}
	wg           sync.WaitGroup
}

func (p *DockerPlugin) Name() string        { return "docker" }
func (p *DockerPlugin) Version() string     { return "1.0.0" }
func (p *DockerPlugin) Description() string { return "Docker管理 - 远程容器管理" }

func (p *DockerPlugin) Init(c *core.Core, cfg map[string]interface{}) error {
	p.core = c
	p.cfg = cfg
	return nil
}

func (p *DockerPlugin) Start() error {
	handler := NewDockerHandler(p.core.DB, p.core.Auth, p.core.Config.JWT.Secret)
	interval := p.statusSyncInterval()
	p.statusTicker = time.NewTicker(interval)
	p.stopCh = make(chan struct{})
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		for {
			select {
			case <-p.stopCh:
				return
			case <-p.statusTicker.C:
				if _, err := handler.syncAllHostsStatus(); err != nil {
					log.Printf("[Docker] host status auto-sync failed: %v", err)
				}
			}
		}
	}()
	return nil
}

func (p *DockerPlugin) Stop() error {
	if p.statusTicker != nil {
		p.statusTicker.Stop()
		p.statusTicker = nil
	}
	if p.stopCh != nil {
		close(p.stopCh)
		p.stopCh = nil
	}
	p.wg.Wait()
	return nil
}

func (p *DockerPlugin) statusSyncInterval() time.Duration {
	const fallback = 60 * time.Second
	if p.cfg == nil {
		return fallback
	}
	value, ok := p.cfg["status_sync_interval_seconds"]
	if !ok {
		return fallback
	}
	parse := func(raw string) time.Duration {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			return fallback
		}
		if n < 10 {
			n = 10
		}
		if n > 300 {
			n = 300
		}
		return time.Duration(n) * time.Second
	}
	switch v := value.(type) {
	case int:
		return parse(strconv.Itoa(v))
	case int64:
		return parse(strconv.FormatInt(v, 10))
	case float64:
		return parse(strconv.Itoa(int(v)))
	case string:
		return parse(v)
	default:
		return fallback
	}
}

func (p *DockerPlugin) Migrate() error {
	return p.core.DB.AutoMigrate(&DockerHost{}, &DockerRegistry{}, &DockerRegistryLogin{})
}

func (p *DockerPlugin) RegisterRoutes(g *gin.RouterGroup) {
	h := NewDockerHandler(p.core.DB, p.core.Auth, p.core.Config.JWT.Secret)
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
