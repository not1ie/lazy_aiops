package rbac

import (
	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/internal/plugin"
)

func init() {
	plugin.Register("rbac", func() plugin.Plugin {
		return &RBACPlugin{}
	})
}

type RBACPlugin struct {
	core *core.Core
	cfg  map[string]interface{}
}

func (p *RBACPlugin) Name() string        { return "rbac" }
func (p *RBACPlugin) Version() string     { return "1.0.0" }
func (p *RBACPlugin) Description() string { return "RBAC权限管理 - 用户、角色、权限管理" }

func (p *RBACPlugin) Init(c *core.Core, cfg map[string]interface{}) error {
	p.core = c
	p.cfg = cfg
	return nil
}

func (p *RBACPlugin) Start() error {
	// 初始化默认权限
	return p.initDefaultPermissions()
}

func (p *RBACPlugin) Stop() error { return nil }

func (p *RBACPlugin) Migrate() error {
	// 核心模型已在core包中定义，这里不需要额外迁移
	return nil
}

func (p *RBACPlugin) RegisterRoutes(g *gin.RouterGroup) {
	h := NewRBACHandler(p.core.DB)

	// 用户管理
	users := g.Group("/users")
	{
		users.GET("", h.ListUsers)
		users.POST("", h.CreateUser)
		users.GET("/:id", h.GetUser)
		users.PUT("/:id", h.UpdateUser)
		users.DELETE("/:id", h.DeleteUser)
		users.PUT("/:id/password", h.ChangePassword)
		users.PUT("/:id/status", h.ChangeStatus)
	}

	// 角色管理
	roles := g.Group("/roles")
	{
		roles.GET("", h.ListRoles)
		roles.POST("", h.CreateRole)
		roles.GET("/:id", h.GetRole)
		roles.PUT("/:id", h.UpdateRole)
		roles.DELETE("/:id", h.DeleteRole)
		roles.PUT("/:id/permissions", h.UpdateRolePermissions)
	}

	// 权限管理
	permissions := g.Group("/permissions")
	{
		permissions.GET("", h.ListPermissions)
		permissions.POST("", h.CreatePermission)
		permissions.GET("/:id", h.GetPermission)
		permissions.PUT("/:id", h.UpdatePermission)
		permissions.DELETE("/:id", h.DeletePermission)
		permissions.GET("/tree", h.GetPermissionTree)
	}

	// 操作日志
	logs := g.Group("/logs")
	{
		logs.GET("", h.ListOperationLogs)
		logs.GET("/:id", h.GetOperationLog)
	}
}

// initDefaultPermissions 初始化默认权限
func (p *RBACPlugin) initDefaultPermissions() error {
	permissions := []core.Permission{
		// 仪表板
		{Name: "仪表板", Code: "dashboard", Type: "menu"},
		{Name: "查看仪表板", Code: "dashboard:view", Type: "api"},
		
		// CMDB
		{Name: "资产管理", Code: "cmdb", Type: "menu"},
		{Name: "查看主机", Code: "cmdb:host:read", Type: "api"},
		{Name: "创建主机", Code: "cmdb:host:create", Type: "api"},
		{Name: "更新主机", Code: "cmdb:host:update", Type: "api"},
		{Name: "删除主机", Code: "cmdb:host:delete", Type: "api"},
		
		// 监控
		{Name: "监控中心", Code: "monitor", Type: "menu"},
		{Name: "查看监控", Code: "monitor:view", Type: "api"},
		{Name: "配置监控", Code: "monitor:config", Type: "api"},
		
		// 告警
		{Name: "告警管理", Code: "alert", Type: "menu"},
		{Name: "查看告警", Code: "alert:read", Type: "api"},
		{Name: "处理告警", Code: "alert:handle", Type: "api"},
		{Name: "配置规则", Code: "alert:rule:config", Type: "api"},
		
		// 任务
		{Name: "任务调度", Code: "task", Type: "menu"},
		{Name: "查看任务", Code: "task:read", Type: "api"},
		{Name: "创建任务", Code: "task:create", Type: "api"},
		{Name: "执行任务", Code: "task:execute", Type: "api"},
		{Name: "删除任务", Code: "task:delete", Type: "api"},
		
		// AI
		{Name: "AI分析", Code: "ai", Type: "menu"},
		{Name: "日志分析", Code: "ai:log:analyze", Type: "api"},
		{Name: "查看历史", Code: "ai:history:read", Type: "api"},
		
		// 系统管理
		{Name: "系统管理", Code: "system", Type: "menu"},
		{Name: "用户管理", Code: "system:user", Type: "menu"},
		{Name: "角色管理", Code: "system:role", Type: "menu"},
		{Name: "权限管理", Code: "system:permission", Type: "menu"},
		{Name: "操作日志", Code: "system:log", Type: "menu"},
	}
	
	for _, perm := range permissions {
		var existing core.Permission
		if err := p.core.DB.Where("code = ?", perm.Code).First(&existing).Error; err != nil {
			// 不存在则创建
			p.core.DB.Create(&perm)
		}
	}
	
	return nil
}
