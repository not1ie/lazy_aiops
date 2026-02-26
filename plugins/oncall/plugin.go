package oncall

import (
	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/pkg/plugin"
)

func init() {
	plugin.Register("oncall", func() plugin.Plugin {
		return &OnCallPlugin{}
	})
}

type OnCallPlugin struct {
	core *core.Core
	cfg  map[string]interface{}
}

func (p *OnCallPlugin) Name() string    { return "oncall" }
func (p *OnCallPlugin) Version() string { return "1.0.0" }
func (p *OnCallPlugin) Description() string {
	return "值班排班 - 轮换排班、换班、升级策略"
}

func (p *OnCallPlugin) Init(c *core.Core, cfg map[string]interface{}) error {
	p.core = c
	p.cfg = cfg
	return nil
}

func (p *OnCallPlugin) Start() error { return nil }
func (p *OnCallPlugin) Stop() error  { return nil }

func (p *OnCallPlugin) Migrate() error {
	return p.core.DB.AutoMigrate(&OnCallSchedule{}, &OnCallShift{}, &OnCallTeam{}, &OnCallOverride{}, &OnCallEscalation{})
}

func (p *OnCallPlugin) RegisterRoutes(g *gin.RouterGroup) {
	h := NewOnCallHandler(p.core.DB)

	// 排班
	schedules := g.Group("/schedules")
	{
		schedules.GET("", h.ListSchedules)
		schedules.POST("", h.CreateSchedule)
		schedules.GET("/:id", h.GetSchedule)
		schedules.PUT("/:id", h.UpdateSchedule)
		schedules.DELETE("/:id", h.DeleteSchedule)
		schedules.POST("/:id/generate", h.GenerateShifts)
		schedules.GET("/:id/shifts", h.ListShifts)
		schedules.GET("/:id/current", h.GetCurrentOnCall)
	}

	// 换班
	g.POST("/shifts/:shift_id/swap", h.SwapShift)

	// 团队
	teams := g.Group("/teams")
	{
		teams.GET("", h.ListTeams)
		teams.POST("", h.CreateTeam)
	}

	// 升级策略
	escalations := g.Group("/escalations")
	{
		escalations.GET("", h.ListEscalations)
		escalations.POST("", h.CreateEscalation)
		escalations.PUT("/:id", h.UpdateEscalation)
		escalations.DELETE("/:id", h.DeleteEscalation)
	}

	// 查询当前值班
	g.GET("/whoisoncall", h.WhoIsOnCall)
}
