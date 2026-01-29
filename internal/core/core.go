package core

import (
	"os"
	"path/filepath"

	"github.com/lazyautoops/lazy-auto-ops/internal/config"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Core 核心模块，提供基础服务给插件使用
type Core struct {
	Config *config.Config
	DB     *gorm.DB
	Auth   *AuthService
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
	default:
		dialector = sqlite.Open(c.Config.Database.DSN)
	}

	logLevel := logger.Silent
	if c.Config.Server.Mode == "debug" {
		logLevel = logger.Info
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return err
	}

	c.DB = db
	return nil
}

func (c *Core) migrate() error {
	return c.DB.AutoMigrate(&User{}, &Role{}, &Permission{}, &OperationLog{})
}
