package terminal

import (
	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/pkg/plugin"
)

func init() {
	plugin.Register("terminal", func() plugin.Plugin {
		return &TerminalPlugin{}
	})
}

type TerminalPlugin struct {
	core *core.Core
	cfg  map[string]interface{}
}

func (p *TerminalPlugin) Name() string        { return "terminal" }
func (p *TerminalPlugin) Version() string     { return "1.0.0" }
func (p *TerminalPlugin) Description() string { return "WebTerminal - 基于WebSocket的SSH终端" }

func (p *TerminalPlugin) Init(c *core.Core, cfg map[string]interface{}) error {
	p.core = c
	p.cfg = cfg
	return nil
}

func (p *TerminalPlugin) Start() error { return nil }
func (p *TerminalPlugin) Stop() error  { return nil }

func (p *TerminalPlugin) Migrate() error {
	if err := p.core.DB.AutoMigrate(&TerminalSession{}, &TerminalRecord{}, &jumpCommandAudit{}, &jumpCommandRule{}); err != nil {
		return err
	}
	var count int64
	p.core.DB.Model(&jumpCommandRule{}).Count(&count)
	if count == 0 {
		rules := []jumpCommandRule{
			{
				ID:        "rule-block-rm",
				Name:      "高危删除命令拦截 (rm -rf)",
				Pattern:   `rm\s+-[a-zA-Z]*r[a-zA-Z]*f\s+.*|rm\s+-[a-zA-Z]*f[a-zA-Z]*r\s+.*`,
				MatchType: "regex",
				RuleKind:  "risk",
				Protocol:  "ssh",
				Severity:  "critical",
				Action:    "block",
				Priority:  100,
				Enabled:   true,
			},
			{
				ID:        "rule-block-db-drop",
				Name:      "高危删库删表指令拦截 (DROP/TRUNCATE)",
				Pattern:   `(?i)drop\s+(database|table)|truncate\s+table`,
				MatchType: "regex",
				RuleKind:  "risk",
				Protocol:  "ssh",
				Severity:  "critical",
				Action:    "block",
				Priority:  90,
				Enabled:   true,
			},
			{
				ID:        "rule-block-mkfs",
				Name:      "高危磁盘格式化拦截 (mkfs)",
				Pattern:   `mkfs.*|dd\s+if=.*of=/dev/.*`,
				MatchType: "regex",
				RuleKind:  "risk",
				Protocol:  "ssh",
				Severity:  "critical",
				Action:    "block",
				Priority:  80,
				Enabled:   true,
			},
		}
		for _, r := range rules {
			p.core.DB.Create(&r)
		}
	}
	return nil
}

func (p *TerminalPlugin) RegisterRoutes(g *gin.RouterGroup) {
	secretKey := ""
	if p.core != nil && p.core.Config != nil {
		secretKey = p.core.Config.JWT.Secret
	}
	h := NewTerminalHandler(p.core.DB, p.core.Auth, secretKey)

	// 会话管理
	g.GET("/sessions", h.ListSessions)
	g.GET("/sessions/:id", h.GetSession)
	g.POST("/sessions", h.CreateSession)
	g.POST("/quick-connect-host/:host_id", h.QuickConnectHost)
	g.POST("/sessions/precheck", h.PrecheckConnection)
	g.PUT("/sessions/:id", h.UpdateSession)
	g.POST("/sessions/:id/share", h.ShareSession)
	g.DELETE("/sessions/:id", h.CloseSession)
	g.DELETE("/sessions/:id/purge", h.DeleteSession)

	// WebSocket连接
	g.GET("/ws/:id", h.HandleWebSocket)

	// 录像回放
	g.GET("/records", h.ListRecords)
	g.GET("/audits", h.ListCommandAudits)
	g.GET("/records/:id", h.GetRecord)
	g.GET("/records/:id/download", h.DownloadRecord)
	g.GET("/records/:id/asciinema", h.DownloadRecordAsciinema)
	g.POST("/records/export", h.ExportRecords)
	g.DELETE("/records/:id", h.DeleteRecord)
	g.POST("/records/cleanup", h.CleanupRecords)
}
