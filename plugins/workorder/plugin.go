package workorder

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
	plugin.Register("workorder", func() plugin.Plugin {
		return &WorkOrderPlugin{}
	})
}

type WorkOrderPlugin struct {
	core         *core.Core
	cfg          map[string]interface{}
	statusTicker *time.Ticker
	stopCh       chan struct{}
	wg           sync.WaitGroup
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
	interval := p.reconcileInterval()
	p.statusTicker = time.NewTicker(interval)
	p.stopCh = make(chan struct{})
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		if checked, updated, err := reconcileWorkflowLinkedOrders(p.core.DB, "system/reconcile", defaultReconcileBatchLimit); err != nil {
			log.Printf("[WorkOrder] reconcile bootstrap failed: %v", err)
		} else if checked > 0 || updated > 0 {
			log.Printf("[WorkOrder] reconcile bootstrap checked=%d updated=%d", checked, updated)
		}
		for {
			select {
			case <-p.stopCh:
				return
			case <-p.statusTicker.C:
				if checked, updated, err := reconcileWorkflowLinkedOrders(p.core.DB, "system/reconcile", defaultReconcileBatchLimit); err != nil {
					log.Printf("[WorkOrder] reconcile tick failed: %v", err)
				} else if updated > 0 {
					log.Printf("[WorkOrder] reconcile tick checked=%d updated=%d", checked, updated)
				}
			}
		}
	}()
	return nil
}

func (p *WorkOrderPlugin) Stop() error {
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

func (p *WorkOrderPlugin) reconcileInterval() time.Duration {
	const fallback = 30 * time.Second
	if p.cfg == nil {
		return fallback
	}
	value, ok := p.cfg["workflow_reconcile_interval_seconds"]
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
