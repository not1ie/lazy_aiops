package api

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
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

	workspacePresetInitMu   sync.Mutex
	workspacePresetInitDone bool
	loginLimiter            *ipLimiter
}

func NewServer(cfg *config.Config, c *core.Core, pm *plugin.Manager) *Server {
	gin.SetMode(cfg.Server.Mode)
	engine := gin.New()
	engine.Use(gin.Recovery(), gin.Logger(), CORSMiddleware(cfg.Server.CORSOrigins), PaginationLimiterMiddleware())

	return &Server{
		config:       cfg,
		core:         c,
		pm:           pm,
		engine:       engine,
		loginLimiter: newIPLimiter(1*time.Minute, 10),
	}
}

func (s *Server) Run() error {
	s.setupRoutes()

	srv := &http.Server{
		Addr:    ":" + s.config.Server.Port,
		Handler: s.engine,
	}

	go func() {
		log.Printf("🚀 Server is running on port %s", s.config.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s.pm.StopAll()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
	return nil
}

func (s *Server) setupRoutes() {
	// SPA 前端静态文件服务
	// 1. 静态资源 (assets, favicon等)
	s.engine.Static("/assets", "./web/static/assets")
	s.engine.StaticFile("/favicon.ico", "./web/static/favicon.ico")

	// 2. 所有非 API 路径且非 assets 路径才返回 index.html (SPA History Mode)
	s.engine.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "API not found"})
		} else if strings.HasPrefix(path, "/assets") {
			c.String(http.StatusNotFound, "Asset not found")
		} else {
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
			c.Header("Pragma", "no-cache")
			c.Header("Expires", "0")
			c.File("./web/static/index.html")
		}
	})

	// 健康检查
	s.engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 浏览器错误日志上报
	s.engine.POST("/api/v1/debug/error", func(c *gin.Context) {
		var req struct {
			Message string `json:"message"`
			Stack   string `json:"stack"`
			URL     string `json:"url"`
		}
		if err := c.ShouldBindJSON(&req); err == nil {
			log.Printf("[BROWSER_ERROR] Msg: %s, Stack: %s, URL: %s", req.Message, req.Stack, req.URL)
		}
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
	g.POST("/login", s.loginLimiter.limitMiddleware(), func(c *gin.Context) {
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
	if s.core != nil && s.core.DB != nil {
		if err := s.core.DB.AutoMigrate(&workspacePresetRecord{}); err != nil {
			log.Printf("[WorkspacePreset] auto migrate failed: %v", err)
		}
		if err := s.ensureWorkspacePresetUniqueConstraint(); err != nil {
			log.Printf("[WorkspacePreset] ensure unique constraint failed: %v", err)
		}
		if err := s.ensureDefaultTeamWorkspacePresets(); err != nil {
			log.Printf("[WorkspacePreset] bootstrap defaults failed: %v", err)
		}
	}

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

	// 工作台模板
	s.setupWorkspacePresetRoutes(g)
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
