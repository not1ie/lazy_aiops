package log

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/pkg/plugin"
)

func init() {
	plugin.Register("log", func() plugin.Plugin {
		return &LogPlugin{}
	})
}

type LogPlugin struct {
	core *core.Core
	cfg  map[string]interface{}
	stop chan struct{}
}

func (p *LogPlugin) Name() string        { return "log" }
func (p *LogPlugin) Version() string     { return "1.0.0" }
func (p *LogPlugin) Description() string { return "日志中心 - 日志接入、查询与日志告警" }

func (p *LogPlugin) Init(c *core.Core, cfg map[string]interface{}) error {
	p.core = c
	p.cfg = cfg
	p.stop = make(chan struct{})
	return nil
}

func (p *LogPlugin) Start() error {
	go p.startConnectionChecker()
	return nil
}

func (p *LogPlugin) Stop() error {
	close(p.stop)
	return nil
}

func (p *LogPlugin) startConnectionChecker() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Initial check
	p.checkConnections()

	for {
		select {
		case <-ticker.C:
			p.checkConnections()
		case <-p.stop:
			return
		}
	}
}

func (p *LogPlugin) checkConnections() {
	var libs []LogLibrary
	if err := p.core.DB.Find(&libs).Error; err != nil {
		return
	}

	for _, lib := range libs {
		url := strings.ToLower(lib.Source)
		if url == "" {
			continue
		}

		timeout := 2 * time.Second
		u := strings.TrimPrefix(url, "http://")
		u = strings.TrimPrefix(u, "https://")
		parts := strings.Split(u, "/")
		host := parts[0]
		if !strings.Contains(host, ":") {
			if lib.Type == "es" {
				host = host + ":9200"
			} else {
				host = host + ":3100"
			}
		}

		status := "active"
		reason := ""
		conn, err := net.DialTimeout("tcp", host, timeout)
		if err != nil {
			status = "error"
			reason = fmt.Sprintf("无法连接至该数据源: %v", err)
		} else {
			conn.Close()
		}

		if lib.Status != status || lib.StatusReason != reason {
			p.core.DB.Model(&lib).Updates(map[string]interface{}{
				"status":        status,
				"status_reason": reason,
				"updated_at":    time.Now(),
			})
		}
	}
}

func (p *LogPlugin) Migrate() error {
	return p.core.DB.AutoMigrate(&LogLibrary{}, &LogAlertRule{}, &LogPermission{})
}

func (p *LogPlugin) RegisterRoutes(g *gin.RouterGroup) {
	h := NewLogHandler(p.core.DB)

	// 日志库
	g.GET("/libraries", h.ListLibraries)
	g.POST("/libraries", h.CreateLibrary)
	g.PUT("/libraries/:id", h.UpdateLibrary)
	g.DELETE("/libraries/:id", h.DeleteLibrary)
	g.POST("/libraries/test", h.TestLibraryConnection)

	// 查询
	g.GET("/query", h.QueryLogs)

	// 告警规则
	g.GET("/alerts", h.ListAlertRules)
	g.POST("/alerts", h.CreateAlertRule)
	g.PUT("/alerts/:id", h.UpdateAlertRule)
	g.DELETE("/alerts/:id", h.DeleteAlertRule)
	g.POST("/alerts/:id/toggle", h.ToggleAlertRule)

	// 权限
	g.GET("/permissions", h.ListPermissions)
	g.POST("/permissions", h.CreatePermission)
	g.DELETE("/permissions/:id", h.DeletePermission)
}
