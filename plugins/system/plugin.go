package system

import (
	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/pkg/plugin"
)

func init() {
	plugin.Register("system", func() plugin.Plugin {
		return &SystemPlugin{}
	})
}

type SystemPlugin struct {
	core *core.Core
}

func (p *SystemPlugin) Name() string        { return "system" }
func (p *SystemPlugin) Version() string     { return "1.0.0" }
func (p *SystemPlugin) Description() string { return "系统管理 - 部门、菜单、角色权限" }

func (p *SystemPlugin) Init(c *core.Core, cfg map[string]interface{}) error {
	p.core = c
	return nil
}

func (p *SystemPlugin) Start() error { return nil }
func (p *SystemPlugin) Stop() error  { return nil }

func (p *SystemPlugin) Migrate() error {
	return p.core.DB.AutoMigrate(&Department{}, &Menu{})
}

func (p *SystemPlugin) RegisterRoutes(g *gin.RouterGroup) {
	h := NewSystemHandler(p.core.DB)
	
	// Dept
	g.GET("/depts", h.ListDepartments)
	g.POST("/depts", h.CreateDepartment)
	g.PUT("/depts/:id", h.UpdateDepartment)
	g.DELETE("/depts/:id", h.DeleteDepartment)

	// Menu
	g.GET("/menus", h.ListMenus)
	g.POST("/menus", h.CreateMenu)
}
