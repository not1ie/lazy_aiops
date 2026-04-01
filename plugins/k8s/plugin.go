package k8s

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
	plugin.Register("k8s", func() plugin.Plugin {
		return &K8sPlugin{}
	})
}

type K8sPlugin struct {
	core         *core.Core
	cfg          map[string]interface{}
	statusTicker *time.Ticker
	stopCh       chan struct{}
	wg           sync.WaitGroup
}

func (p *K8sPlugin) Name() string    { return "k8s" }
func (p *K8sPlugin) Version() string { return "1.0.0" }
func (p *K8sPlugin) Description() string {
	return "Kubernetes管理 - 多集群、节点、工作负载、Pod管理"
}

func (p *K8sPlugin) Init(c *core.Core, cfg map[string]interface{}) error {
	p.core = c
	p.cfg = cfg
	return nil
}

func (p *K8sPlugin) Start() error {
	handler := NewK8sHandler(p.core.DB, p.core.Auth)
	interval := p.statusSyncInterval()
	p.statusTicker = time.NewTicker(interval)
	p.stopCh = make(chan struct{})
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		if _, err := handler.syncAllClusterStatuses(); err != nil {
			log.Printf("[K8s] cluster status bootstrap sync failed: %v", err)
		}
		for {
			select {
			case <-p.stopCh:
				return
			case <-p.statusTicker.C:
				if _, err := handler.syncAllClusterStatuses(); err != nil {
					log.Printf("[K8s] cluster status auto-sync failed: %v", err)
				}
			}
		}
	}()
	return nil
}

func (p *K8sPlugin) Stop() error {
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

func (p *K8sPlugin) statusSyncInterval() time.Duration {
	const fallback = 90 * time.Second
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

func (p *K8sPlugin) Migrate() error {
	return p.core.DB.AutoMigrate(&Cluster{})
}

func (p *K8sPlugin) RegisterRoutes(g *gin.RouterGroup) {
	h := NewK8sHandler(p.core.DB, p.core.Auth)

	// 集群管理
	clusters := g.Group("/clusters")
	{
		clusters.GET("", h.ListClusters)
		clusters.POST("", h.CreateCluster)
		clusters.GET("/:id", h.GetCluster)
		clusters.PUT("/:id", h.UpdateCluster)
		clusters.DELETE("/:id", h.DeleteCluster)
		clusters.POST("/:id/test", h.TestConnection)
	}

	// 节点管理
	g.GET("/clusters/:id/nodes", h.ListNodes)

	// 命名空间
	g.GET("/clusters/:id/namespaces", h.ListNamespaces)

	// 工作负载
	g.GET("/clusters/:id/workloads", h.ListWorkloads)
	g.GET("/clusters/:id/namespaces/:ns/workloads/:kind/:name", h.GetWorkload)
	g.GET("/clusters/:id/namespaces/:ns/workloads/:kind/:name/manifest", h.GetWorkloadManifest)
	g.POST("/clusters/:id/namespaces/:ns/workloads/:kind/:name/manifest/apply", h.ApplyWorkloadManifest)
	g.PUT("/clusters/:id/namespaces/:ns/workloads/:kind/:name/scale", h.ScaleWorkload)
	g.POST("/clusters/:id/namespaces/:ns/workloads/:kind/:name/restart", h.RestartWorkloadByRef)
	g.GET("/clusters/:id/namespaces/:ns/deployments", h.ListDeployments)
	g.POST("/clusters/:id/namespaces/:ns/deployments", h.CreateDeployment)
	g.DELETE("/clusters/:id/namespaces/:ns/deployments/:name", h.DeleteDeployment)
	g.PUT("/clusters/:id/namespaces/:ns/deployments/:name/scale", h.ScaleDeployment)
	g.POST("/clusters/:id/namespaces/:ns/deployments/:name/restart", h.RestartDeployment)
	g.GET("/clusters/:id/namespaces/:ns/deployments/:name/runtime", h.GetDeploymentRuntime)
	g.PUT("/clusters/:id/namespaces/:ns/deployments/:name/env", h.UpdateDeploymentEnv)
	g.PUT("/clusters/:id/namespaces/:ns/deployments/:name/image", h.UpdateDeploymentImage)
	g.PUT("/clusters/:id/namespaces/:ns/deployments/:name/domains", h.UpdateDeploymentDomains)

	// Pod管理
	g.GET("/clusters/:id/namespaces/:ns/pods", h.ListPods)
	g.GET("/clusters/:id/namespaces/:ns/pods/:name", h.GetPod)
	g.GET("/clusters/:id/namespaces/:ns/pods/:name/logs", h.GetPodLogs)
	g.GET("/clusters/:id/namespaces/:ns/pods/:name/logs/stream", h.StreamPodLogs)
	g.DELETE("/clusters/:id/namespaces/:ns/pods/:name", h.DeletePod)
	g.POST("/clusters/:id/namespaces/:ns/pods/:name/restart", h.RestartPod)
	g.POST("/clusters/:id/namespaces/:ns/pods/:name/restart-workload", h.RestartWorkload)
	g.GET("/clusters/:id/namespaces/:ns/pods/:name/exec", h.ExecPod)

	// Service & Ingress
	g.GET("/clusters/:id/services", h.ListServices)
	g.GET("/clusters/:id/ingresses", h.ListIngresses)

	// ConfigMap & Secret
	g.GET("/clusters/:id/configmaps", h.ListConfigMaps)
	g.GET("/clusters/:id/secrets", h.ListSecrets)

	// Storage
	g.GET("/clusters/:id/storageclasses", h.ListStorageClasses)
	g.GET("/clusters/:id/persistentvolumes", h.ListPersistentVolumes)
	g.GET("/clusters/:id/namespaces/:ns/persistentvolumeclaims", h.ListPersistentVolumeClaims)

	// Events
	g.GET("/clusters/:id/events", h.ListEvents)
}
