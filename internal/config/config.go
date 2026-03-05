package config

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"os"
	"path/filepath"
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
	Port        string   `mapstructure:"port"`
	Mode        string   `mapstructure:"mode"` // debug, release
	CORSOrigins []string `mapstructure:"cors_origins"`
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
	cfg.Server.CORSOrigins = resolveCORSOrigins(cfg.Server.CORSOrigins)
	cfg.JWT.Secret = resolveJWTSecret(cfg.JWT.Secret, cfg.Database.DSN)

	return &cfg, nil
}

func setDefaults() {
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.mode", "debug")
	viper.SetDefault("server.cors_origins", []string{"*"})
	viper.SetDefault("database.driver", "sqlite")
	viper.SetDefault("database.dsn", "data/lazy-auto-ops.db")
	viper.SetDefault("jwt.expire", 24)

	// 默认启用的插件
	viper.SetDefault("plugins.cmdb.enabled", true)
	viper.SetDefault("plugins.monitor.enabled", true)
	viper.SetDefault("plugins.task.enabled", true)
	viper.SetDefault("plugins.jump.enabled", true)
	viper.SetDefault("plugins.rbac.enabled", true)
	viper.SetDefault("plugins.system.enabled", true)
}

func resolveJWTSecret(rawSecret, dsn string) string {
	if envSecret := strings.TrimSpace(os.Getenv("LAO_JWT_SECRET")); envSecret != "" {
		return envSecret
	}

	secret := strings.TrimSpace(rawSecret)
	if !isInsecureJWTSecret(secret) {
		return secret
	}

	if secretFile := strings.TrimSpace(os.Getenv("LAO_JWT_SECRET_FILE")); secretFile != "" {
		if loaded, err := loadOrCreateSecretFile(secretFile); err == nil {
			return loaded
		}
	}

	if secretFile := defaultJWTSecretFile(dsn); secretFile != "" {
		if loaded, err := loadOrCreateSecretFile(secretFile); err == nil {
			return loaded
		}
	}

	sum := sha256.Sum256([]byte("lazy-auto-ops-jwt:" + strings.TrimSpace(dsn)))
	return hex.EncodeToString(sum[:])
}

func isInsecureJWTSecret(secret string) bool {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return true
	}
	switch secret {
	case "lazy-auto-ops-secret-key", "lazy-auto-ops-secret-change-me-in-production", "${JWT_SECRET}":
		return true
	default:
		return false
	}
}

func defaultJWTSecretFile(dsn string) string {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" || strings.Contains(dsn, "@tcp(") {
		return ""
	}
	clean := filepath.Clean(dsn)
	if filepath.IsAbs(clean) {
		return filepath.Join(filepath.Dir(clean), ".jwt_secret")
	}
	return filepath.Join(filepath.Dir(clean), ".jwt_secret")
}

func loadOrCreateSecretFile(secretFile string) (string, error) {
	if data, err := os.ReadFile(secretFile); err == nil {
		if secret := strings.TrimSpace(string(data)); secret != "" {
			return secret, nil
		}
	}

	random := make([]byte, 48)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	secret := base64.RawURLEncoding.EncodeToString(random)
	if err := os.MkdirAll(filepath.Dir(secretFile), 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(secretFile, []byte(secret), 0600); err != nil {
		return "", err
	}
	return secret, nil
}

func resolveCORSOrigins(origins []string) []string {
	if raw := strings.TrimSpace(os.Getenv("LAO_SERVER_CORS_ORIGINS")); raw != "" {
		parts := strings.Split(raw, ",")
		resolved := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				resolved = append(resolved, p)
			}
		}
		if len(resolved) > 0 {
			return resolved
		}
	}

	resolved := make([]string, 0, len(origins))
	for _, o := range origins {
		o = strings.TrimSpace(o)
		if o != "" {
			resolved = append(resolved, o)
		}
	}
	if len(resolved) == 0 {
		return []string{"*"}
	}
	return resolved
}
