package cmdb

import (
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/pkg/plugin"
)

func init() {
	plugin.Register("cmdb", func() plugin.Plugin {
		return &CMDBPlugin{}
	})
}

type CMDBPlugin struct {
	core         *core.Core
	cfg          map[string]interface{}
	statusTicker *time.Ticker
	stopCh       chan struct{}
	wg           sync.WaitGroup
}

func (p *CMDBPlugin) Name() string    { return "cmdb" }
func (p *CMDBPlugin) Version() string { return "1.0.0" }
func (p *CMDBPlugin) Description() string {
	return "资产管理模块 - 主机、数据库、云资源管理"
}

func (p *CMDBPlugin) Init(c *core.Core, cfg map[string]interface{}) error {
	p.core = c
	p.cfg = cfg
	return nil
}

func (p *CMDBPlugin) Start() error {
	handler := NewHostHandler(p.core.DB, p.core.Config.JWT.Secret)
	interval := p.statusSyncInterval()
	p.statusTicker = time.NewTicker(interval)
	p.stopCh = make(chan struct{})
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		if _, err := handler.syncHostStatuses(nil, 2*time.Second); err != nil {
			log.Printf("[CMDB] host status bootstrap sync failed: %v", err)
		}
		for {
			select {
			case <-p.stopCh:
				return
			case <-p.statusTicker.C:
				if _, err := handler.syncHostStatuses(nil, 2*time.Second); err != nil {
					log.Printf("[CMDB] host status auto-sync failed: %v", err)
				}
			}
		}
	}()
	return nil
}

func (p *CMDBPlugin) Stop() error {
	if p.statusTicker != nil {
		p.statusTicker.Stop()
		p.statusTicker = nil
	}
	if p.stopCh != nil {
		close(p.stopCh)
		p.stopCh = nil
	}
	p.wg.Wait()
	return nil
}

func (p *CMDBPlugin) statusSyncInterval() time.Duration {
	const fallback = 45 * time.Second
	if p.cfg == nil {
		return fallback
	}
	value, ok := p.cfg["status_sync_interval_seconds"]
	if !ok {
		return fallback
	}
	parse := func(raw string) time.Duration {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			return fallback
		}
		if n < 10 {
			n = 10
		}
		if n > 300 {
			n = 300
		}
		return time.Duration(n) * time.Second
	}
	switch v := value.(type) {
	case int:
		return parse(strconv.Itoa(v))
	case int64:
		return parse(strconv.FormatInt(v, 10))
	case float64:
		return parse(strconv.Itoa(int(v)))
	case string:
		return parse(v)
	default:
		return fallback
	}
}

func (p *CMDBPlugin) Migrate() error {
	return p.core.DB.AutoMigrate(
		&Host{},
		&NetworkDevice{},
		&HostGroup{},
		&Credential{},
		&DatabaseAsset{},
		&CloudAccount{},
		&CloudResource{},
	)
}

func (p *CMDBPlugin) RegisterRoutes(g *gin.RouterGroup) {
	h := NewHostHandler(p.core.DB, p.core.Config.JWT.Secret)

	// 主机管理
	hosts := g.Group("/hosts")
	{
		hosts.GET("", h.List)
		hosts.POST("", h.Create)
		hosts.POST("/sync-status", h.SyncHostStatuses)
		hosts.GET("/:id", h.Get)
		hosts.POST("/:id/test", h.TestHost)
		hosts.PUT("/:id", h.Update)
		hosts.DELETE("/:id", h.Delete)
	}

	// 网络设备（交换机/防火墙）
	networkDevices := g.Group("/network-devices")
	{
		networkDevices.GET("", h.ListNetworkDevices)
		networkDevices.POST("", h.CreateNetworkDevice)
		networkDevices.GET("/:id", h.GetNetworkDevice)
		networkDevices.POST("/:id/test", h.TestNetworkDevice)
		networkDevices.PUT("/:id", h.UpdateNetworkDevice)
		networkDevices.DELETE("/:id", h.DeleteNetworkDevice)
		networkDevices.POST("/sync/firewalls", h.SyncNetworkDevicesFromFirewalls)
	}

	// 主机分组
	groups := g.Group("/groups")
	{
		groups.GET("", h.ListGroups)
		groups.POST("", h.CreateGroup)
		groups.PUT("/:id", h.UpdateGroup)
		groups.DELETE("/:id", h.DeleteGroup)
	}

	// 凭据管理
	creds := g.Group("/credentials")
	{
		creds.GET("", h.ListCredentials)
		creds.GET("/:id", h.GetCredential)
		creds.POST("", h.CreateCredential)
		creds.POST("/:id/test", h.TestCredential)
		creds.PUT("/:id", h.UpdateCredential)
		creds.DELETE("/:id", h.DeleteCredential)
	}

	// 数据库资产
	databases := g.Group("/databases")
	{
		databases.GET("", h.ListDatabases)
		databases.POST("", h.CreateDatabase)
		databases.GET("/:id", h.GetDatabase)
		databases.POST("/:id/test", h.TestDatabase)
		databases.PUT("/:id", h.UpdateDatabase)
		databases.DELETE("/:id", h.DeleteDatabase)
	}

	// 云资源
	cloudAccounts := g.Group("/cloud/accounts")
	{
		cloudAccounts.GET("", h.ListCloudAccounts)
		cloudAccounts.POST("", h.CreateCloudAccount)
		cloudAccounts.GET("/:id", h.GetCloudAccount)
		cloudAccounts.POST("/:id/test", h.TestCloudAccount)
		cloudAccounts.PUT("/:id", h.UpdateCloudAccount)
		cloudAccounts.DELETE("/:id", h.DeleteCloudAccount)
	}
	cloudResources := g.Group("/cloud/resources")
	{
		cloudResources.GET("", h.ListCloudResources)
		cloudResources.POST("", h.CreateCloudResource)
		cloudResources.GET("/:id", h.GetCloudResource)
		cloudResources.PUT("/:id", h.UpdateCloudResource)
		cloudResources.DELETE("/:id", h.DeleteCloudResource)
	}
}
