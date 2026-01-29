package plugin

import (
	"fmt"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/config"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
)

// Plugin 插件接口 - 所有插件必须实现
type Plugin interface {
	// 基本信息
	Name() string
	Version() string
	Description() string

	// 生命周期
	Init(core *core.Core, cfg map[string]interface{}) error
	Start() error
	Stop() error

	// 路由注册
	RegisterRoutes(group *gin.RouterGroup)

	// 数据库迁移
	Migrate() error
}

// Manager 插件管理器
type Manager struct {
	mu       sync.RWMutex
	plugins  map[string]Plugin
	registry map[string]func() Plugin // 插件工厂注册表
	core     *core.Core
	loaded   []string
}

var (
	manager *Manager
	once    sync.Once
)

// GetManager 获取插件管理器单例
func GetManager() *Manager {
	once.Do(func() {
		manager = &Manager{
			plugins:  make(map[string]Plugin),
			registry: make(map[string]func() Plugin),
			loaded:   make([]string, 0),
		}
	})
	return manager
}

// Register 注册插件工厂（插件包init时调用）
func Register(name string, factory func() Plugin) {
	m := GetManager()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.registry[name] = factory
}

// SetCore 设置核心模块引用
func (m *Manager) SetCore(c *core.Core) {
	m.core = c
}

// LoadEnabledPlugins 加载配置中启用的插件
func (m *Manager) LoadEnabledPlugins(pluginConfigs map[string]config.PluginConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, factory := range m.registry {
		cfg, exists := pluginConfigs[name]
		if !exists || !cfg.Enabled {
			continue
		}

		p := factory()
		if err := p.Init(m.core, cfg.Config); err != nil {
			return fmt.Errorf("初始化插件 %s 失败: %w", name, err)
		}

		if err := p.Migrate(); err != nil {
			return fmt.Errorf("插件 %s 数据库迁移失败: %w", name, err)
		}

		if err := p.Start(); err != nil {
			return fmt.Errorf("启动插件 %s 失败: %w", name, err)
		}

		m.plugins[name] = p
		m.loaded = append(m.loaded, name)
		fmt.Printf("✅ 插件已加载: %s v%s\n", p.Name(), p.Version())
	}

	return nil
}

// GetPlugin 获取已加载的插件
func (m *Manager) GetPlugin(name string) (Plugin, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.plugins[name]
	return p, ok
}

// GetLoadedPlugins 获取所有已加载的插件
func (m *Manager) GetLoadedPlugins() []Plugin {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]Plugin, 0, len(m.plugins))
	for _, p := range m.plugins {
		result = append(result, p)
	}
	return result
}

// ListAvailable 列出所有可用插件
func (m *Manager) ListAvailable() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	names := make([]string, 0, len(m.registry))
	for name := range m.registry {
		names = append(names, name)
	}
	return names
}

// StopAll 停止所有插件
func (m *Manager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, p := range m.plugins {
		_ = p.Stop()
	}
}
