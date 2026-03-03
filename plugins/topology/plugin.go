package topology

import (
	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/pkg/plugin"
)

func init() {
	plugin.Register("topology", func() plugin.Plugin {
		return &TopologyPlugin{}
	})
}

type TopologyPlugin struct {
	core    *core.Core
	cfg     map[string]interface{}
	handler *TopologyHandler
}

func (p *TopologyPlugin) Name() string    { return "topology" }
func (p *TopologyPlugin) Version() string { return "1.0.0" }
func (p *TopologyPlugin) Description() string {
	return "服务拓扑 - 可视化服务依赖、调用链分析"
}

func (p *TopologyPlugin) Init(c *core.Core, cfg map[string]interface{}) error {
	p.core = c
	p.cfg = cfg
	return nil
}

func (p *TopologyPlugin) Start() error { return nil }
func (p *TopologyPlugin) Stop() error  { return nil }

func (p *TopologyPlugin) Migrate() error {
	db := p.core.DB
	if err := db.AutoMigrate(&ServiceNode{}, &ServiceEdge{}, &TopologyView{}, &ServiceDependency{}); err != nil {
		return err
	}

	// 历史版本将 name 设为全局唯一，导致自动发现重复同步时触发 UNIQUE constraint。
	// 迁移为 (name, namespace, cluster) 组合唯一后，这里清理旧索引。
	migrator := db.Migrator()
	if migrator.HasIndex(&ServiceNode{}, "idx_service_nodes_name") {
		_ = migrator.DropIndex(&ServiceNode{}, "idx_service_nodes_name")
	}
	if !migrator.HasIndex(&ServiceNode{}, "idx_service_nodes_lookup") {
		if err := migrator.CreateIndex(&ServiceNode{}, "idx_service_nodes_lookup"); err != nil {
			return err
		}
	}

	return nil
}

func (p *TopologyPlugin) RegisterRoutes(r *gin.RouterGroup) {
	p.handler = NewTopologyHandler(p.core.DB)

	r.GET("/data", p.handler.GetTopology)
	r.GET("/nodes", p.handler.ListNodes)
	r.POST("/nodes", p.handler.CreateNode)
	r.PUT("/nodes/:id", p.handler.UpdateNode)
	r.DELETE("/nodes/:id", p.handler.DeleteNode)
	r.PUT("/nodes/:id/position", p.handler.UpdateNodePosition)
	r.GET("/nodes/:id/detail", p.handler.GetNodeDetail)
	r.GET("/edges", p.handler.ListEdges)
	r.POST("/edges", p.handler.CreateEdge)
	r.DELETE("/edges/:id", p.handler.DeleteEdge)
	r.GET("/analyze", p.handler.AnalyzeDependencies)
	r.POST("/sync-k8s", p.handler.SyncFromK8s)
	r.POST("/discover", p.handler.Discover)
	r.GET("/views", p.handler.ListViews)
	r.POST("/views", p.handler.CreateView)
	r.POST("/layout/save", p.handler.SaveLayout)
	r.POST("/layout/auto", p.handler.AutoLayout)
	r.GET("/export", p.handler.ExportTopology)
	r.POST("/import", p.handler.ImportTopology)
}
