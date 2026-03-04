package rbac

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/pkg/plugin"
	"gorm.io/gorm"
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
	if err := p.initDefaultPermissions(); err != nil {
		return err
	}
	return p.ensureAdminPermissions()
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

type permSeed struct {
	Name   string
	Code   string
	Type   string
	Parent string
}

// initDefaultPermissions 初始化默认权限
func (p *RBACPlugin) initDefaultPermissions() error {
	seeds := []permSeed{
		{Name: "仪表盘", Code: "dashboard", Type: "menu"},
		{Name: "AI运维助手", Code: "ai", Type: "menu"},

		{Name: "资产管理", Code: "cmdb", Type: "menu"},
		{Name: "防火墙管理", Code: "firewall", Type: "menu", Parent: "cmdb"},

		{Name: "Docker管理", Code: "docker", Type: "menu"},
		{Name: "K8s管理", Code: "k8s", Type: "menu"},

		{Name: "监控中心", Code: "monitor", Type: "menu"},
		{Name: "告警管理", Code: "alert", Type: "menu"},
		{Name: "通知管理", Code: "notify", Type: "menu"},
		{Name: "域名与证书", Code: "domain", Type: "menu"},

		{Name: "工作流编排", Code: "workflow", Type: "menu"},
		{Name: "批量执行", Code: "executor", Type: "menu"},
		{Name: "任务调度", Code: "task", Type: "menu"},
		{Name: "Ansible", Code: "ansible", Type: "menu"},

		{Name: "CI/CD", Code: "cicd", Type: "menu"},
		{Name: "应用中心", Code: "application", Type: "menu", Parent: "cicd"},

		{Name: "配置中心", Code: "nacos", Type: "menu"},

		{Name: "工单管理", Code: "workorder", Type: "menu"},
		{Name: "SQL审核", Code: "sqlaudit", Type: "menu"},
		{Name: "GitOps", Code: "gitops", Type: "menu"},

		{Name: "值班管理", Code: "oncall", Type: "menu"},
		{Name: "Web终端", Code: "terminal", Type: "menu"},
		{Name: "堡垒机", Code: "jump", Type: "menu"},
		{Name: "资产接入", Code: "jump:asset", Type: "menu", Parent: "jump"},
		{Name: "授权策略", Code: "jump:policy", Type: "menu", Parent: "jump"},
		{Name: "命令风控", Code: "jump:rule", Type: "menu", Parent: "jump"},
		{Name: "会话审计", Code: "jump:session", Type: "menu", Parent: "jump"},

		{Name: "服务拓扑", Code: "topology", Type: "menu"},
		{Name: "成本管理", Code: "cost", Type: "menu"},

		{Name: "系统管理", Code: "system", Type: "menu"},
		{Name: "用户管理", Code: "system:user", Type: "menu", Parent: "system"},
		{Name: "角色管理", Code: "system:role", Type: "menu", Parent: "system"},
		{Name: "权限管理", Code: "system:permission", Type: "menu", Parent: "system"},
		{Name: "部门管理", Code: "system:dept", Type: "menu", Parent: "system"},
		{Name: "岗位管理", Code: "system:post", Type: "menu", Parent: "system"},
		{Name: "登录日志", Code: "system:loginlog", Type: "menu", Parent: "system"},
		{Name: "验证码配置", Code: "system:captcha", Type: "menu", Parent: "system"},
		{Name: "操作日志", Code: "system:log", Type: "menu", Parent: "system"},

		{Name: "知识库", Code: "knowledge", Type: "menu"},
		{Name: "自动修复", Code: "remediation", Type: "menu"},
	}

	byCode := make(map[string]core.Permission, len(seeds))
	for _, seed := range seeds {
		var perm core.Permission
		err := p.core.DB.Where("code = ?", seed.Code).First(&perm).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				perm = core.Permission{
					Name: seed.Name,
					Code: seed.Code,
					Type: seed.Type,
				}
				if err := p.core.DB.Create(&perm).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		}
		byCode[seed.Code] = perm
	}

	for _, seed := range seeds {
		if seed.Parent == "" {
			continue
		}
		child, ok := byCode[seed.Code]
		if !ok {
			continue
		}
		parent, ok := byCode[seed.Parent]
		if !ok || parent.ID == "" {
			continue
		}
		if child.ParentID != parent.ID {
			p.core.DB.Model(&core.Permission{}).Where("id = ?", child.ID).Update("parent_id", parent.ID)
		}
	}

	return nil
}

func (p *RBACPlugin) ensureAdminPermissions() error {
	var admin core.Role
	if err := p.core.DB.Where("code = ?", "admin").First(&admin).Error; err != nil {
		return nil
	}

	var perms []core.Permission
	if err := p.core.DB.Find(&perms).Error; err != nil {
		return err
	}

	permPtrs := make([]*core.Permission, 0, len(perms))
	for i := range perms {
		permPtrs = append(permPtrs, &perms[i])
	}
	return p.core.DB.Model(&admin).Association("Permissions").Replace(permPtrs)
}
