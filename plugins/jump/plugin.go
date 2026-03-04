package jump

import (
	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/pkg/plugin"
)

func init() {
	plugin.Register("jump", func() plugin.Plugin {
		return &JumpPlugin{}
	})
}

type JumpPlugin struct {
	core    *core.Core
	cfg     map[string]interface{}
	handler *JumpHandler
}

func (p *JumpPlugin) Name() string        { return "jump" }
func (p *JumpPlugin) Version() string     { return "1.0.0" }
func (p *JumpPlugin) Description() string { return "堡垒机融合 - 资产、授权、会话审计" }

func (p *JumpPlugin) Init(c *core.Core, cfg map[string]interface{}) error {
	p.core = c
	p.cfg = cfg
	return nil
}

func (p *JumpPlugin) Start() error { return nil }
func (p *JumpPlugin) Stop() error  { return nil }

func (p *JumpPlugin) Migrate() error {
	return p.core.DB.AutoMigrate(
		&JumpAsset{},
		&JumpAccount{},
		&JumpPermissionPolicy{},
		&JumpSession{},
		&JumpCommandRule{},
		&JumpCommand{},
	)
}

func (p *JumpPlugin) RegisterRoutes(r *gin.RouterGroup) {
	secretKey := ""
	if p.core != nil && p.core.Config != nil {
		secretKey = p.core.Config.JWT.Secret
	}
	p.handler = NewJumpHandler(p.core.DB, secretKey)

	// 资产
	r.GET("/assets", p.handler.ListAssets)
	r.POST("/assets", p.handler.CreateAsset)
	r.GET("/assets/:id", p.handler.GetAsset)
	r.PUT("/assets/:id", p.handler.UpdateAsset)
	r.DELETE("/assets/:id", p.handler.DeleteAsset)

	// 账号
	r.GET("/accounts", p.handler.ListAccounts)
	r.POST("/accounts", p.handler.CreateAccount)
	r.GET("/accounts/:id", p.handler.GetAccount)
	r.PUT("/accounts/:id", p.handler.UpdateAccount)
	r.DELETE("/accounts/:id", p.handler.DeleteAccount)

	// 授权策略
	r.GET("/policies", p.handler.ListPolicies)
	r.POST("/policies", p.handler.CreatePolicy)
	r.GET("/policies/:id", p.handler.GetPolicy)
	r.PUT("/policies/:id", p.handler.UpdatePolicy)
	r.DELETE("/policies/:id", p.handler.DeletePolicy)

	// 命令风控规则
	r.GET("/command-rules", p.handler.ListCommandRules)
	r.GET("/command-rules/stats", p.handler.GetCommandRuleStats)
	r.POST("/command-rules/batch", p.handler.BatchCommandRules)
	r.POST("/command-rules", p.handler.CreateCommandRule)
	r.GET("/command-rules/:id", p.handler.GetCommandRule)
	r.PUT("/command-rules/:id", p.handler.UpdateCommandRule)
	r.DELETE("/command-rules/:id", p.handler.DeleteCommandRule)

	// 会话与审计
	r.GET("/sessions", p.handler.ListSessions)
	r.POST("/sessions/start", p.handler.StartSession)
	r.POST("/sessions/:id/approve", p.handler.ApproveSession)
	r.POST("/sessions/:id/reject", p.handler.RejectSession)
	r.POST("/sessions/:id/connect", p.handler.ConnectSession)
	r.GET("/sessions/:id", p.handler.GetSession)
	r.POST("/sessions/:id/commands", p.handler.RecordCommand)
	r.GET("/sessions/:id/commands", p.handler.ListSessionCommands)
	r.POST("/sessions/:id/close", p.handler.CloseSession)

	// 资产同步
	r.POST("/sync/cmdb-hosts", p.handler.SyncFromCMDBHosts)
	r.POST("/sync/k8s-clusters", p.handler.SyncFromK8sClusters)
	r.POST("/sync/docker-hosts", p.handler.SyncFromDockerHosts)
	r.POST("/sync/all", p.handler.SyncAllAssets)
}
