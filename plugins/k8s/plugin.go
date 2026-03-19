package k8s

import (
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
	core *core.Core
	cfg  map[string]interface{}
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

func (p *K8sPlugin) Start() error { return nil }
func (p *K8sPlugin) Stop() error  { return nil }

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
