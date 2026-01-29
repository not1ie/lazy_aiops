package ai

import (
	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/internal/plugin"
)

func init() {
	plugin.Register("ai", func() plugin.Plugin {
		return &AIPlugin{}
	})
}

type AIPlugin struct {
	core    *core.Core
	cfg     map[string]interface{}
	service *AIService
}

func (p *AIPlugin) Name() string        { return "ai" }
func (p *AIPlugin) Version() string     { return "1.0.0" }
func (p *AIPlugin) Description() string { return "AI运维助手 - 日志分析、故障诊断、智能问答" }

func (p *AIPlugin) Init(c *core.Core, cfg map[string]interface{}) error {
	p.core = c
	p.cfg = cfg

	// 从配置获取LLM设置
	provider := "openai"
	apiKey := ""
	baseURL := ""
	model := "gpt-3.5-turbo"

	if v, ok := cfg["provider"].(string); ok {
		provider = v
	}
	if v, ok := cfg["api_key"].(string); ok {
		apiKey = v
	}
	if v, ok := cfg["base_url"].(string); ok {
		baseURL = v
	}
	if v, ok := cfg["model"].(string); ok {
		model = v
	}

	p.service = NewAIService(c.DB, provider, apiKey, baseURL, model)
	return nil
}

func (p *AIPlugin) Start() error { return nil }
func (p *AIPlugin) Stop() error  { return nil }

func (p *AIPlugin) Migrate() error {
	return p.core.DB.AutoMigrate(&ChatSession{}, &ChatMessage{}, &LogAnalysis{})
}

func (p *AIPlugin) RegisterRoutes(g *gin.RouterGroup) {
	h := NewAIHandler(p.core.DB, p.service)

	// 对话
	g.POST("/chat", h.Chat)
	g.GET("/sessions", h.ListSessions)
	g.GET("/sessions/:id/messages", h.GetSessionMessages)
	g.DELETE("/sessions/:id", h.DeleteSession)

	// 分析
	g.POST("/analyze/logs", h.AnalyzeLogs)
	g.POST("/analyze/logs-detailed", h.AnalyzeLogsDetailed)
	g.GET("/analyze/history", h.ListAnalysisHistory)
	g.GET("/analyze/:id", h.GetAnalysisDetail)
	g.POST("/analyze/error", h.AnalyzeError)
	g.POST("/analyze/performance", h.AnalyzePerformance)

	// 建议
	g.POST("/suggest/fix", h.SuggestFix)
	g.POST("/suggest/optimize", h.SuggestOptimize)
}
