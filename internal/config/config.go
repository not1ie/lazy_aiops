package config

import (
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig            `mapstructure:"server"`
	Database DatabaseConfig          `mapstructure:"database"`
	JWT      JWTConfig               `mapstructure:"jwt"`
	Plugins  map[string]PluginConfig `mapstructure:"plugins"`
}

type ServerConfig struct {
	Port string `mapstructure:"port"`
	Mode string `mapstructure:"mode"` // debug, release
}

type DatabaseConfig struct {
	Driver string `mapstructure:"driver"` // sqlite, mysql, postgres
	DSN    string `mapstructure:"dsn"`
}

type JWTConfig struct {
	Secret string `mapstructure:"secret"`
	Expire int    `mapstructure:"expire"` // hours
}

type PluginConfig struct {
	Enabled bool                   `mapstructure:"enabled"`
	Config  map[string]interface{} `mapstructure:"config"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath("/etc/lazy-auto-ops")

	// 环境变量支持
	viper.SetEnvPrefix("LAO")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// 默认值
	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// 环境变量覆盖
	if port := os.Getenv("LAO_SERVER_PORT"); port != "" {
		cfg.Server.Port = port
	}

	return &cfg, nil
}

func setDefaults() {
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.mode", "debug")
	viper.SetDefault("database.driver", "sqlite")
	viper.SetDefault("database.dsn", "data/lazy-auto-ops.db")
	viper.SetDefault("jwt.secret", "lazy-auto-ops-secret-key")
	viper.SetDefault("jwt.expire", 24)

	// 默认启用的插件
	viper.SetDefault("plugins.cmdb.enabled", true)
	viper.SetDefault("plugins.monitor.enabled", true)
	viper.SetDefault("plugins.task.enabled", true)
}
