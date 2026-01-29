package main

import (
	"log"

	"github.com/lazyautoops/lazy-auto-ops/internal/api"
	"github.com/lazyautoops/lazy-auto-ops/internal/config"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/pkg/plugin"

	// 核心插件
	_ "github.com/lazyautoops/lazy-auto-ops/plugins/ai"
	_ "github.com/lazyautoops/lazy-auto-ops/plugins/alert"
	_ "github.com/lazyautoops/lazy-auto-ops/plugins/ansible"
	_ "github.com/lazyautoops/lazy-auto-ops/plugins/cicd"
	_ "github.com/lazyautoops/lazy-auto-ops/plugins/cmdb"
	_ "github.com/lazyautoops/lazy-auto-ops/plugins/cost"
	_ "github.com/lazyautoops/lazy-auto-ops/plugins/domain"
	_ "github.com/lazyautoops/lazy-auto-ops/plugins/executor"
	_ "github.com/lazyautoops/lazy-auto-ops/plugins/firewall"
	_ "github.com/lazyautoops/lazy-auto-ops/plugins/gitops"
	_ "github.com/lazyautoops/lazy-auto-ops/plugins/k8s"
	_ "github.com/lazyautoops/lazy-auto-ops/plugins/knowledge"
	_ "github.com/lazyautoops/lazy-auto-ops/plugins/monitor"
	_ "github.com/lazyautoops/lazy-auto-ops/plugins/nacos"
	_ "github.com/lazyautoops/lazy-auto-ops/plugins/notify"
	_ "github.com/lazyautoops/lazy-auto-ops/plugins/oncall"
	_ "github.com/lazyautoops/lazy-auto-ops/plugins/remediation"
	_ "github.com/lazyautoops/lazy-auto-ops/plugins/sqlaudit"
	_ "github.com/lazyautoops/lazy-auto-ops/plugins/task"
	_ "github.com/lazyautoops/lazy-auto-ops/plugins/terminal"
	_ "github.com/lazyautoops/lazy-auto-ops/plugins/topology"
	_ "github.com/lazyautoops/lazy-auto-ops/plugins/workflow"
	_ "github.com/lazyautoops/lazy-auto-ops/plugins/workorder"
)

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化核心模块
	coreModule, err := core.New(cfg)
	if err != nil {
		log.Fatalf("初始化核心模块失败: %v", err)
	}

	// 初始化插件管理器
	pm := plugin.GetManager()
	pm.SetCore(coreModule)

	// 加载启用的插件
	if err := pm.LoadEnabledPlugins(cfg.Plugins); err != nil {
		log.Fatalf("加载插件失败: %v", err)
	}

	// 启动API服务
	server := api.NewServer(cfg, coreModule, pm)
	log.Printf("🚀 Lazy Auto Ops 启动中... 端口: %s", cfg.Server.Port)
	if err := server.Run(); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
