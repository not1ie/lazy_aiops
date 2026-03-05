package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/config"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/pkg/plugin"
)

type Server struct {
	config *config.Config
	core   *core.Core
	pm     *plugin.Manager
	engine *gin.Engine
}

func NewServer(cfg *config.Config, c *core.Core, pm *plugin.Manager) *Server {
	gin.SetMode(cfg.Server.Mode)
	engine := gin.New()
	engine.Use(gin.Recovery(), gin.Logger(), CORSMiddleware(cfg.Server.CORSOrigins))

	return &Server{
		config: cfg,
		core:   c,
		pm:     pm,
		engine: engine,
	}
}

func (s *Server) Run() error {
	s.setupRoutes()
	return s.engine.Run(":" + s.config.Server.Port)
}

func (s *Server) setupRoutes() {
	// SPA 前端静态文件服务
	// 1. 静态资源 (assets, favicon等)
	s.engine.Static("/assets", "./web/static/assets")
	s.engine.StaticFile("/favicon.ico", "./web/static/favicon.ico")

	// 2. 所有非 API 路径都返回 index.html (SPA History Mode)
	s.engine.NoRoute(func(c *gin.Context) {
		if !strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.File("./web/static/index.html")
		} else {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "API not found"})
		}
	})

	// 健康检查
	s.engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// API v1
	v1 := s.engine.Group("/api/v1")

	// 公开接口
	s.setupPublicRoutes(v1)

	// 需要认证的接口
	auth := v1.Group("")
	auth.Use(
		AuthMiddleware(s.core.Auth),
		OperationLogMiddleware(s.core.DB),
		ForcePasswordChangeMiddleware(s.core.Auth),
		RBACMiddleware(s.core.DB),
	)
	s.setupAuthRoutes(auth)

	// 注册插件路由
	for _, p := range s.pm.GetLoadedPlugins() {
		pluginGroup := auth.Group("/" + p.Name())
		p.RegisterRoutes(pluginGroup)
	}
}

func (s *Server) setupPublicRoutes(g *gin.RouterGroup) {
	// 登录
	g.POST("/login", func(c *gin.Context) {
		var req core.LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			s.recordLoginLog(c, req.Username, 0, "参数错误")
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
			return
		}

		resp, err := s.core.Auth.Login(&req)
		if err != nil {
			s.recordLoginLog(c, req.Username, 0, err.Error())
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": err.Error()})
			return
		}
		s.recordLoginLog(c, req.Username, 1, "登录成功")

		c.JSON(http.StatusOK, gin.H{"code": 0, "data": resp})
	})

	// 系统信息
	g.GET("/system/info", func(c *gin.Context) {
		plugins := make([]gin.H, 0)
		for _, p := range s.pm.GetLoadedPlugins() {
			plugins = append(plugins, gin.H{
				"name":        p.Name(),
				"version":     p.Version(),
				"description": p.Description(),
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": gin.H{
				"name":    "Lazy Auto Ops",
				"version": "1.0.0",
				"plugins": plugins,
			},
		})
	})
}

func (s *Server) setupAuthRoutes(g *gin.RouterGroup) {
	// 获取当前用户信息
	g.GET("/user/info", func(c *gin.Context) {
		userID := c.GetString("user_id")
		user, err := s.core.Auth.GetUserByID(userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取用户信息失败"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": user})
	})

	// 插件列表
	g.GET("/plugins", func(c *gin.Context) {
		available := s.pm.ListAvailable()
		loaded := make([]gin.H, 0)
		for _, p := range s.pm.GetLoadedPlugins() {
			loaded = append(loaded, gin.H{
				"name":        p.Name(),
				"version":     p.Version(),
				"description": p.Description(),
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": gin.H{
				"available": available,
				"loaded":    loaded,
			},
		})
	})
}

func (s *Server) recordLoginLog(c *gin.Context, username string, status int, message string) {
	if s.core == nil || s.core.DB == nil {
		return
	}
	_ = s.core.DB.Create(&loginLogRecord{
		Username:  username,
		IP:        c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
		Status:    status,
		Message:   message,
		LoginAt:   time.Now(),
	}).Error
}
