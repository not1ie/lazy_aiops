package domain

import (
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/pkg/plugin"
	"gorm.io/gorm"
)

func init() {
	plugin.Register("domain", func() plugin.Plugin {
		return &DomainPlugin{}
	})
}

type DomainPlugin struct {
	core         *core.Core
	cfg          map[string]interface{}
	statusTicker *time.Ticker
	stopCh       chan struct{}
	wg           sync.WaitGroup
}

func (p *DomainPlugin) Name() string    { return "domain" }
func (p *DomainPlugin) Version() string { return "1.0.0" }
func (p *DomainPlugin) Description() string {
	return "域名管理 - 云域名到期监控、SSL证书监控"
}

func (p *DomainPlugin) Init(c *core.Core, cfg map[string]interface{}) error {
	p.core = c
	p.cfg = cfg
	return nil
}

func (p *DomainPlugin) Start() error {
	handler := NewDomainHandler(p.core.DB)
	interval := p.statusSyncInterval()
	p.statusTicker = time.NewTicker(interval)
	p.stopCh = make(chan struct{})
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		if _, err := handler.syncAllRuntime(); err != nil {
			log.Printf("[Domain] runtime bootstrap sync failed: %v", err)
		}
		for {
			select {
			case <-p.stopCh:
				return
			case <-p.statusTicker.C:
				if _, err := handler.syncAllRuntime(); err != nil {
					log.Printf("[Domain] runtime auto-sync failed: %v", err)
				}
			}
		}
	}()
	return nil
}

func (p *DomainPlugin) Stop() error {
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

func (p *DomainPlugin) statusSyncInterval() time.Duration {
	const fallback = 10 * time.Minute
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
		if n < 60 {
			n = 60
		}
		if n > 3600 {
			n = 3600
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

func (p *DomainPlugin) Migrate() error {
	if err := p.core.DB.AutoMigrate(&CloudDomain{}, &CloudAccount{}, &SSLCertificate{}); err != nil {
		return err
	}
	return ensureCertSansCompatibility(p.core.DB)
}

func ensureCertSansCompatibility(db *gorm.DB) error {
	migrator := db.Migrator()
	columns := loadTableColumns(db, "ssl_certificates")
	hasSans := false
	hasLegacySANs := false
	for _, col := range columns {
		switch strings.ToLower(strings.TrimSpace(col)) {
		case "sans":
			hasSans = true
		case "s_a_ns":
			hasLegacySANs = true
		}
	}
	if !hasSans {
		_ = migrator.AddColumn(&SSLCertificate{}, "SANs")
		for _, col := range loadTableColumns(db, "ssl_certificates") {
			if strings.ToLower(strings.TrimSpace(col)) == "sans" {
				hasSans = true
				break
			}
		}
	}
	if !hasSans || !hasLegacySANs {
		return nil
	}
	if err := db.Exec(`
		UPDATE ssl_certificates
		SET sans = COALESCE(NULLIF(sans, ''), s_a_ns)
		WHERE (sans IS NULL OR sans = '')
		  AND s_a_ns IS NOT NULL
		  AND s_a_ns <> ''
	`).Error; err != nil {
		// Keep startup resilient for mixed historical schemas.
		if isMissingSansColumnError(err) {
			return nil
		}
		return err
	}
	return nil
}

func (p *DomainPlugin) RegisterRoutes(g *gin.RouterGroup) {
	h := NewDomainHandler(p.core.DB)

	// 云账号
	accounts := g.Group("/accounts")
	{
		accounts.GET("", h.ListAccounts)
		accounts.POST("", h.CreateAccount)
		accounts.DELETE("/:id", h.DeleteAccount)
		accounts.POST("/:id/sync", h.SyncDomains)
	}

	// 域名
	domains := g.Group("/domains")
	{
		domains.GET("", h.ListDomains)
		domains.GET("/expiring", h.ListExpiringDomains)
		domains.POST("/check", h.CheckDomain)
		domains.POST("/check_all", h.CheckAllDomains)
	}

	// SSL证书
	certs := g.Group("/certs")
	{
		certs.GET("", h.ListCerts)
		certs.POST("", h.CreateCert)
		certs.DELETE("/:id", h.DeleteCert)
		certs.POST("/:id/check", h.CheckCert)
		certs.POST("/check_all", h.CheckAllCerts)
		certs.GET("/expiring", h.ListExpiringCerts)
	}
}
