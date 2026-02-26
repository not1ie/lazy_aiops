package workorder

import (
	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/pkg/plugin"
)

func init() {
	plugin.Register("workorder", func() plugin.Plugin {
		return &WorkOrderPlugin{}
	})
}

type WorkOrderPlugin struct {
	core *core.Core
	cfg  map[string]interface{}
}

func (p *WorkOrderPlugin) Name() string        { return "workorder" }
func (p *WorkOrderPlugin) Version() string     { return "1.0.0" }
func (p *WorkOrderPlugin) Description() string { return "运维工单 - 审批流程、AI辅助处理" }

func (p *WorkOrderPlugin) Init(c *core.Core, cfg map[string]interface{}) error {
	p.core = c
	p.cfg = cfg
	return nil
}

func (p *WorkOrderPlugin) Start() error {
	p.initDefaultTypes()
	return nil
}

func (p *WorkOrderPlugin) Stop() error { return nil }

func (p *WorkOrderPlugin) Migrate() error {
	return p.core.DB.AutoMigrate(
		&WorkOrderType{}, &WorkOrder{}, &WorkOrderFlow{},
		&WorkOrderStep{}, &WorkOrderComment{}, &WorkOrderAttachment{},
	)
}

func (p *WorkOrderPlugin) RegisterRoutes(g *gin.RouterGroup) {
	h := NewWorkOrderHandler(p.core.DB)

	// 工单类型
	types := g.Group("/types")
	{
		types.GET("", h.ListTypes)
		types.POST("", h.CreateType)
		types.PUT("/:id", h.UpdateType)
		types.DELETE("/:id", h.DeleteType)
	}

	// 工单
	orders := g.Group("/orders")
	{
		orders.GET("", h.ListOrders)
		orders.POST("", h.CreateOrder)
		orders.GET("/:id", h.GetOrder)
		orders.POST("/:id/approve", h.ApproveOrder)
		orders.POST("/:id/execute", h.ExecuteOrder)
		orders.POST("/:id/complete", h.CompleteOrder)
		orders.POST("/:id/cancel", h.CancelOrder)
		orders.POST("/:id/comment", h.AddComment)
	}

	// 统计
	g.GET("/stats", h.GetStats)
}

func (p *WorkOrderPlugin) initDefaultTypes() {
	defaultTypes := []WorkOrderType{
		{Name: "服务器申请", Code: "server_apply", Icon: "server", Description: "申请新服务器资源"},
		{Name: "权限申请", Code: "permission_apply", Icon: "key", Description: "申请系统权限"},
		{Name: "变更申请", Code: "change_apply", Icon: "edit", Description: "系统变更申请"},
		{Name: "故障处理", Code: "incident", Icon: "warning", Description: "故障处理工单"},
		{Name: "日常运维", Code: "routine", Icon: "tool", Description: "日常运维任务"},
	}

	for _, t := range defaultTypes {
		var count int64
		p.core.DB.Model(&WorkOrderType{}).Where("code = ?", t.Code).Count(&count)
		if count == 0 {
			p.core.DB.Create(&t)
		}
	}
}
