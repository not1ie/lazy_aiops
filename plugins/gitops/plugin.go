package gitops

import (
	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/internal/plugin"
)

func init() {
	plugin.Register("gitops", func() plugin.Plugin {
		return &GitOpsPlugin{}
	})
}

type GitOpsPlugin struct {
	core *core.Core
	cfg  map[string]interface{}
}

func (p *GitOpsPlugin) Name() string        { return "gitops" }
func (p *GitOpsPlugin) Version() string     { return "1.0.0" }
func (p *GitOpsPlugin) Description() string { return "GitOps配置管理 - Git仓库同步、配置变更追踪" }

func (p *GitOpsPlugin) Init(c *core.Core, cfg map[string]interface{}) error {
	p.core = c
	p.cfg = cfg
	return nil
}

func (p *GitOpsPlugin) Start() error { return nil }
func (p *GitOpsPlugin) Stop() error  { return nil }

func (p *GitOpsPlugin) Migrate() error {
	return p.core.DB.AutoMigrate(&GitRepo{}, &GitConfig{}, &ConfigChange{})
}

func (p *GitOpsPlugin) RegisterRoutes(g *gin.RouterGroup) {
	workDir := ""
	if v, ok := p.cfg["work_dir"].(string); ok {
		workDir = v
	}
	h := NewGitOpsHandler(p.core.DB, workDir)

	// 仓库管理
	repos := g.Group("/repos")
	{
		repos.GET("", h.ListRepos)
		repos.POST("", h.CreateRepo)
		repos.GET("/:id", h.GetRepo)
		repos.DELETE("/:id", h.DeleteRepo)
		repos.POST("/:id/sync", h.SyncRepo)
	}

	// 配置管理
	configs := g.Group("/configs")
	{
		configs.GET("", h.ListConfigs)
		configs.POST("", h.CreateConfig)
		configs.GET("/:id", h.GetConfig)
		configs.PUT("/:id", h.UpdateConfig)
	}

	// 变更历史
	g.GET("/changes", h.ListChanges)
}
