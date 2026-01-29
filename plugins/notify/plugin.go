package notify

import (
	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/internal/plugin"
)

func init() {
	plugin.Register("notify", func() plugin.Plugin {
		return &NotifyPlugin{}
	})
}

type NotifyPlugin struct {
	core *core.Core
	cfg  map[string]interface{}
}

func (p *NotifyPlugin) Name() string        { return "notify" }
func (p *NotifyPlugin) Version() string     { return "1.0.0" }
func (p *NotifyPlugin) Description() string { return "通知中心 - 飞书/钉钉/企微/邮件/Webhook" }

func (p *NotifyPlugin) Init(c *core.Core, cfg map[string]interface{}) error {
	p.core = c
	p.cfg = cfg
	return nil
}

func (p *NotifyPlugin) Start() error { return nil }
func (p *NotifyPlugin) Stop() error  { return nil }

func (p *NotifyPlugin) Migrate() error {
	return p.core.DB.AutoMigrate(&NotifyChannel{}, &NotifyTemplate{}, &NotifyRecord{}, &NotifyGroup{})
}

func (p *NotifyPlugin) RegisterRoutes(g *gin.RouterGroup) {
	h := NewNotifyHandler(p.core.DB)

	// 渠道管理
	channels := g.Group("/channels")
	{
		channels.GET("", h.ListChannels)
		channels.POST("", h.CreateChannel)
		channels.PUT("/:id", h.UpdateChannel)
		channels.DELETE("/:id", h.DeleteChannel)
		channels.POST("/:id/test", h.TestChannel)
	}

	// 发送通知
	g.POST("/send", h.SendNotify)

	// 通知记录
	g.GET("/records", h.ListRecords)

	// 通知组
	groups := g.Group("/groups")
	{
		groups.GET("", h.ListGroups)
		groups.POST("", h.CreateGroup)
		groups.PUT("/:id", h.UpdateGroup)
		groups.DELETE("/:id", h.DeleteGroup)
	}

	// 模板
	templates := g.Group("/templates")
	{
		templates.GET("", h.ListTemplates)
		templates.POST("", h.CreateTemplate)
	}
}
