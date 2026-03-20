package domain

import (
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
	core *core.Core
	cfg  map[string]interface{}
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

func (p *DomainPlugin) Start() error { return nil }
func (p *DomainPlugin) Stop() error  { return nil }

func (p *DomainPlugin) Migrate() error {
	if err := p.core.DB.AutoMigrate(&CloudDomain{}, &CloudAccount{}, &SSLCertificate{}); err != nil {
		return err
	}
	return ensureCertSansCompatibility(p.core.DB)
}

func ensureCertSansCompatibility(db *gorm.DB) error {
	migrator := db.Migrator()
	hasSans := migrator.HasColumn(&SSLCertificate{}, "sans")
	hasLegacySANs := migrator.HasColumn(&SSLCertificate{}, "s_a_ns")
	if !hasSans {
		if err := migrator.AddColumn(&SSLCertificate{}, "SANs"); err == nil {
			hasSans = migrator.HasColumn(&SSLCertificate{}, "sans")
		}
	}
	if !hasSans || !hasLegacySANs {
		return nil
	}
	return db.Exec(`
		UPDATE ssl_certificates
		SET sans = COALESCE(NULLIF(sans, ''), s_a_ns)
		WHERE (sans IS NULL OR sans = '')
		  AND s_a_ns IS NOT NULL
		  AND s_a_ns <> ''
	`).Error
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
