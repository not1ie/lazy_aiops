package sre

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/pkg/plugin"
)

func init() {
	plugin.Register("sre", func() plugin.Plugin {
		return &SrePlugin{}
	})
}

// SrePlugin 将 lazysre sidecar 能力集成进 lazy_aiops 插件体系
type SrePlugin struct {
	core    *core.Core
	cfg     map[string]interface{}
	client  *SreClient
	handler *SreHandler
}

func (p *SrePlugin) Name() string    { return "sre" }
func (p *SrePlugin) Version() string { return "1.0.0" }
func (p *SrePlugin) Description() string {
	return "AI SRE 运维技能引擎 - 对接 lazysre sidecar，提供 Skill 执行、Preflight 风险评分、Runbook 管理、SLO 监控"
}

func (p *SrePlugin) Init(c *core.Core, cfg map[string]interface{}) error {
	p.core = c
	p.cfg = cfg

	// 从插件配置里读取 sidecar 地址，默认 localhost:19090
	sidecarURL := "http://127.0.0.1:19090"
	if v, ok := cfg["sidecar_url"]; ok {
		if s, ok := v.(string); ok && s != "" {
			sidecarURL = s
		}
	}

	p.client = NewSreClient(sidecarURL)
	p.handler = NewSreHandler(p.client)

	log.Printf("[SRE] 插件初始化完成, sidecar 地址: %s", sidecarURL)
	return nil
}

func (p *SrePlugin) Start() error {
	// 启动时检查 sidecar 是否在线（不强依赖，离线时路由仍注册，调用时返回 503）
	log.Printf("[SRE] 插件已启动，等待 lazysre sidecar 就绪...")
	return nil
}

func (p *SrePlugin) Stop() error {
	log.Println("[SRE] 插件已停止")
	return nil
}

// Migrate 无需数据库迁移（数据存储由 lazysre 自身管理）
func (p *SrePlugin) Migrate() error { return nil }

// RegisterRoutes 注册所有 /sre/* 路由
func (p *SrePlugin) RegisterRoutes(g *gin.RouterGroup) {
	p.handler.RegisterRoutes(g)
}
