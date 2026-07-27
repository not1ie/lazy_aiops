package core

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/lazyautoops/lazy-auto-ops/internal/config"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Core 核心模块，提供基础服务给插件使用
type Core struct {
	Config *config.Config
	DB     *gorm.DB
	Auth   *AuthService
	AI     *AIService
}

// New 创建核心模块
func New(cfg *config.Config) (*Core, error) {
	c := &Core{Config: cfg}

	// 初始化数据库
	if err := c.initDB(); err != nil {
		return nil, err
	}

	// 初始化认证服务
	c.Auth = NewAuthService(c.DB, cfg.JWT)

	// 初始化AI服务
	c.initAI()

	// 自动迁移核心表
	if err := c.migrate(); err != nil {
		return nil, err
	}

	// 初始化默认管理员
	if err := c.Auth.InitDefaultAdmin(); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Core) initAI() {
	// 尝试从 ai 插件配置中获取 AI 设置
	provider := "openai"
	apiKey := ""
	baseURL := ""
	model := "gpt-3.5-turbo"

	if aiCfg, ok := c.Config.Plugins["ai"]; ok {
		if v, ok := aiCfg.Config["provider"].(string); ok {
			provider = v
		}
		if v, ok := aiCfg.Config["api_key"].(string); ok {
			apiKey = v
		}
		if v, ok := aiCfg.Config["base_url"].(string); ok {
			baseURL = v
		}
		if v, ok := aiCfg.Config["model"].(string); ok {
			model = v
		}
	}

	c.AI = NewAIService(provider, apiKey, baseURL, model)
}

func (c *Core) initDB() error {
	var dialector gorm.Dialector

	switch c.Config.Database.Driver {
	case "sqlite":
		// 确保数据目录存在
		dir := filepath.Dir(c.Config.Database.DSN)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		dialector = sqlite.Open(c.Config.Database.DSN)
	case "mysql":
		dialector = mysql.Open(c.Config.Database.DSN)
	case "postgres", "postgresql":
		dialector = postgres.Open(c.Config.Database.DSN)
	default:
		dir := filepath.Dir(c.Config.Database.DSN)
		if err := os.MkdirAll(dir, 0755); err != nil {
			dialector = sqlite.Open("data/lazy-auto-ops.db")
		} else {
			dialector = sqlite.Open(c.Config.Database.DSN)
		}
	}

	logLevel := logger.Silent
	if c.Config.Server.Mode == "debug" {
		logLevel = logger.Info
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})

	var pingErr error
	if err == nil {
		if sqlDB, errDB := db.DB(); errDB == nil {
			pingErr = sqlDB.Ping()
		}
	}

	if err != nil || pingErr != nil {
		log.Printf("[HIGH-AVAILABILITY WARNING] Primary DB (%s) connection/ping failed (err: %v, ping: %v). Falling back to local SQLite DB!", c.Config.Database.Driver, err, pingErr)
		sqliteDir := "data"
		_ = os.MkdirAll(sqliteDir, 0755)
		sqlitePath := filepath.Join(sqliteDir, "lazy_aiops.db")
		fallbackDB, fallbackErr := gorm.Open(sqlite.Open(sqlitePath), &gorm.Config{
			Logger: logger.Default.LogMode(logLevel),
		})
		if fallbackErr != nil {
			return fmt.Errorf("both primary DB and fallback SQLite DB failed: primary err: %v, ping: %v, fallback err: %v", err, pingErr, fallbackErr)
		}
		c.DB = fallbackDB
		c.Config.Database.Driver = "sqlite"
		return nil
	}

	// Configure connection pooling for production-grade databases
	if c.Config.Database.Driver != "sqlite" {
		sqlDB, err := db.DB()
		if err == nil {
			sqlDB.SetMaxIdleConns(10)
			sqlDB.SetMaxOpenConns(100)
			sqlDB.SetConnMaxLifetime(time.Hour)
		}
	}

	c.DB = db
	return nil
}

func (c *Core) migrate() error {
	return c.DB.AutoMigrate(&User{}, &Role{}, &Permission{}, &OperationLog{})
}
