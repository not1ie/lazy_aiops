package ansible

import (
	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/internal/plugin"
)

func init() {
	plugin.Register("ansible", func() plugin.Plugin {
		return &AnsiblePlugin{}
	})
}

type AnsiblePlugin struct {
	core    *core.Core
	cfg     map[string]interface{}
	handler *AnsibleHandler
}

func (p *AnsiblePlugin) Name() string        { return "ansible" }
func (p *AnsiblePlugin) Version() string     { return "1.0.0" }
func (p *AnsiblePlugin) Description() string { return "Ansible管理 - Playbook/Inventory/Role管理与执行" }

func (p *AnsiblePlugin) Init(c *core.Core, cfg map[string]interface{}) error {
	p.core = c
	p.cfg = cfg
	return nil
}

func (p *AnsiblePlugin) Start() error { return nil }
func (p *AnsiblePlugin) Stop() error  { return nil }

func (p *AnsiblePlugin) Migrate() error {
	return p.core.DB.AutoMigrate(&AnsiblePlaybook{}, &AnsibleInventory{}, &AnsibleRole{}, &AnsibleExecution{})
}

func (p *AnsiblePlugin) RegisterRoutes(r *gin.RouterGroup) {
	p.handler = NewAnsibleHandler(p.core.DB, "")

	// Playbook
	r.GET("/playbooks", p.handler.ListPlaybooks)
	r.POST("/playbooks", p.handler.CreatePlaybook)
	r.GET("/playbooks/:id", p.handler.GetPlaybook)
	r.PUT("/playbooks/:id", p.handler.UpdatePlaybook)
	r.DELETE("/playbooks/:id", p.handler.DeletePlaybook)
	r.POST("/playbooks/:id/execute", p.handler.ExecutePlaybook)
	r.POST("/playbooks/:id/validate", p.handler.ValidatePlaybook)
	r.GET("/playbooks/:id/variables", p.handler.ParsePlaybook)

	// Inventory
	r.GET("/inventories", p.handler.ListInventories)
	r.POST("/inventories", p.handler.CreateInventory)
	r.GET("/inventories/:id", p.handler.GetInventory)
	r.PUT("/inventories/:id", p.handler.UpdateInventory)
	r.DELETE("/inventories/:id", p.handler.DeleteInventory)
	r.POST("/inventories/sync-cmdb", p.handler.SyncFromCMDB)

	// Role
	r.GET("/roles", p.handler.ListRoles)
	r.POST("/roles/install", p.handler.InstallRole)

	// 执行记录
	r.GET("/executions", p.handler.ListExecutions)
	r.GET("/executions/:id", p.handler.GetExecution)
	r.POST("/executions/:id/cancel", p.handler.CancelExecution)
	r.GET("/executions/:id/stream", p.handler.StreamOutput)
}
