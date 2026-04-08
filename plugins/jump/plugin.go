package jump

import (
	"errors"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/pkg/plugin"
	"gorm.io/gorm"
)

func init() {
	plugin.Register("jump", func() plugin.Plugin {
		return &JumpPlugin{}
	})
}

type JumpPlugin struct {
	core           *core.Core
	cfg            map[string]interface{}
	handler        *JumpHandler
	autoSyncTicker *time.Ticker
	stopCh         chan struct{}
	wg             sync.WaitGroup
}

func (p *JumpPlugin) Name() string        { return "jump" }
func (p *JumpPlugin) Version() string     { return "1.0.0" }
func (p *JumpPlugin) Description() string { return "堡垒机融合 - 资产、授权、会话审计" }

func (p *JumpPlugin) Init(c *core.Core, cfg map[string]interface{}) error {
	p.core = c
	p.cfg = cfg
	return nil
}

func (p *JumpPlugin) Start() error {
	secretKey := ""
	if p.core != nil && p.core.Config != nil {
		secretKey = p.core.Config.JWT.Secret
	}
	if p.handler == nil {
		p.handler = NewJumpHandler(p.core.DB, secretKey)
	}

	interval := p.autoSyncInterval()
	p.autoSyncTicker = time.NewTicker(interval)
	p.stopCh = make(chan struct{})
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		if err := p.handler.runAutoSyncCycle("bootstrap"); err != nil {
			log.Printf("[Jump] auto-sync bootstrap failed: %v", err)
		}
		for {
			select {
			case <-p.stopCh:
				return
			case <-p.autoSyncTicker.C:
				if err := p.handler.runAutoSyncCycle("ticker"); err != nil {
					log.Printf("[Jump] auto-sync tick failed: %v", err)
				}
			}
		}
	}()
	return nil
}

func (p *JumpPlugin) Stop() error {
	if p.autoSyncTicker != nil {
		p.autoSyncTicker.Stop()
		p.autoSyncTicker = nil
	}
	if p.stopCh != nil {
		close(p.stopCh)
		p.stopCh = nil
	}
	p.wg.Wait()
	return nil
}

func (p *JumpPlugin) Migrate() error {
	if err := p.core.DB.AutoMigrate(
		&JumpAsset{},
		&JumpAccount{},
		&JumpPermissionPolicy{},
		&JumpSession{},
		&JumpCommandRule{},
		&JumpCommand{},
		&JumpRiskEvent{},
		&JumpIntegrationConfig{},
	); err != nil {
		return err
	}
	return ensureDefaultJumpCommandRules(p.core)
}

func (p *JumpPlugin) RegisterRoutes(r *gin.RouterGroup) {
	secretKey := ""
	if p.core != nil && p.core.Config != nil {
		secretKey = p.core.Config.JWT.Secret
	}
	if p.handler == nil {
		p.handler = NewJumpHandler(p.core.DB, secretKey)
	}

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
	r.POST("/sessions/:id/sql/execute", p.handler.ExecuteSQL)
	r.POST("/sessions/:id/close", p.handler.CloseSession)
	r.POST("/sessions/:id/disconnect", p.handler.DisconnectSession)
	r.GET("/risk-events", p.handler.ListRiskEvents)

	// 资产同步
	r.POST("/sync/cmdb-hosts", p.handler.SyncFromCMDBHosts)
	r.POST("/sync/k8s-clusters", p.handler.SyncFromK8sClusters)
	r.POST("/sync/docker-hosts", p.handler.SyncFromDockerHosts)
	r.POST("/sync/jumpserver", p.handler.SyncFromJumpServer)
	r.POST("/sync/jumpserver-sessions", p.handler.SyncJumpServerSessions)
	r.POST("/sync/all", p.handler.SyncAllAssets)

	// 外部集成配置（JumpServer）
	r.GET("/integration/config", p.handler.GetIntegrationConfig)
	r.PUT("/integration/config", p.handler.UpdateIntegrationConfig)
	r.POST("/integration/test", p.handler.TestIntegrationConnection)
}

func ensureDefaultJumpCommandRules(c *core.Core) error {
	if c == nil || c.DB == nil {
		return nil
	}
	defaults := []JumpCommandRule{
		{
			Name:      "阻断 rm -rf 根目录",
			Pattern:   "rm -rf /",
			MatchType: "contains",
			RuleKind:  "risk",
			Protocol:  "ssh",
			Severity:  "critical",
			Action:    "block",
			Priority:  1000,
			Enabled:   true,
		},
		{
			Name:      "阻断 mkfs 格式化命令",
			Pattern:   "mkfs.",
			MatchType: "contains",
			RuleKind:  "risk",
			Protocol:  "",
			Severity:  "critical",
			Action:    "block",
			Priority:  900,
			Enabled:   true,
		},
	}

	for i := range defaults {
		rule := defaults[i]
		rule.Name = strings.TrimSpace(rule.Name)
		if rule.Name == "" {
			continue
		}
		var existing JumpCommandRule
		err := c.DB.Where("name = ?", rule.Name).First(&existing).Error
		if err == nil {
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if createErr := c.DB.Create(&rule).Error; createErr != nil {
			return createErr
		}
	}
	return nil
}

func (p *JumpPlugin) autoSyncInterval() time.Duration {
	const fallback = 120 * time.Second
	if p.cfg == nil {
		return fallback
	}
	value, ok := p.cfg["integration_auto_sync_interval_seconds"]
	if !ok {
		return fallback
	}
	parse := func(raw string) time.Duration {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			return fallback
		}
		if n < 30 {
			n = 30
		}
		if n > 900 {
			n = 900
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
