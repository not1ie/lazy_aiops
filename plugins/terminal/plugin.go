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
	return p.core.DB.AutoMigrate(&TerminalSession{}, &TerminalRecord{})
}

func (p *TerminalPlugin) RegisterRoutes(g *gin.RouterGroup) {
	h := NewTerminalHandler(p.core.DB, p.core.Auth)

	// 会话管理
	g.GET("/sessions", h.ListSessions)
	g.GET("/sessions/:id", h.GetSession)
	g.POST("/sessions", h.CreateSession)
	g.PUT("/sessions/:id", h.UpdateSession)
	g.DELETE("/sessions/:id", h.CloseSession)
	g.DELETE("/sessions/:id/purge", h.DeleteSession)

	// WebSocket连接
	g.GET("/ws/:id", h.HandleWebSocket)

	// 录像回放
	g.GET("/records", h.ListRecords)
	g.GET("/records/:id", h.GetRecord)
}
