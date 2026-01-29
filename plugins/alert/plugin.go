package alert

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/pkg/plugin"
)

func init() {
	plugin.Register("alert", func() plugin.Plugin {
		return &AlertPlugin{}
	})
}

type AlertPlugin struct {
	core       *core.Core
	cfg        map[string]interface{}
	aggregator *Aggregator
}

func (p *AlertPlugin) Name() string        { return "alert" }
func (p *AlertPlugin) Version() string     { return "1.0.0" }
func (p *AlertPlugin) Description() string { return "告警中心 - AI智能降噪、聚合、静默" }

func (p *AlertPlugin) Init(c *core.Core, cfg map[string]interface{}) error {
	p.core = c
	p.cfg = cfg

	interval := 60
	if v, ok := cfg["aggregate_interval"].(int); ok {
		interval = v
	}
	p.aggregator = NewAggregator(c.DB, interval)

	// 设置 AI 分析器
	p.aggregator.SetAIAnalyzer(func(alerts []*Alert) string {
		if len(alerts) == 0 {
			return ""
		}
		var prompt strings.Builder
		prompt.WriteString("你是一个专业的运维专家。请分析以下聚合在一起的告警事件，找出它们的共同点、可能的根本原因，并给出排查建议。\n\n告警列表：\n")
		for i, a := range alerts {
			prompt.WriteString(fmt.Sprintf("%d. [%s] %s - 目标: %s, 指标: %s, 当前值: %s\n", 
				i+1, a.Severity, a.RuleName, a.Target, a.Metric, a.Value))
		}
		prompt.WriteString("\n请以简练的中文回答。")

		analysis, _, err := p.core.AI.CallSimple(prompt.String())
		if err != nil {
			return "AI 分析失败: " + err.Error()
		}
		return analysis
	})

	return nil
}

func (p *AlertPlugin) Start() error { return nil }
func (p *AlertPlugin) Stop() error  { return nil }

func (p *AlertPlugin) Migrate() error {
	return p.core.DB.AutoMigrate(&AlertRule{}, &Alert{}, &AlertSilence{}, &AlertHistory{})
}

func (p *AlertPlugin) RegisterRoutes(g *gin.RouterGroup) {
	h := NewAlertHandler(p.core.DB, p.aggregator)

	// 规则
	rules := g.Group("/rules")
	{
		rules.GET("", h.ListRules)
		rules.POST("", h.CreateRule)
		rules.PUT("/:id", h.UpdateRule)
		rules.DELETE("/:id", h.DeleteRule)
	}

	// 告警
	alerts := g.Group("/alerts")
	{
		alerts.GET("", h.ListAlerts)
		alerts.GET("/:id", h.GetAlert)
		alerts.POST("/:id/ack", h.AckAlert)
		alerts.POST("/:id/resolve", h.ResolveAlert)
	}

	// Webhook接收
	g.POST("/webhook", h.ReceiveAlert)

	// 统计
	g.GET("/stats", h.GetStats)

	// 静默
	silences := g.Group("/silences")
	{
		silences.GET("", h.ListSilences)
		silences.POST("", h.CreateSilence)
		silences.DELETE("/:id", h.DeleteSilence)
	}
}
