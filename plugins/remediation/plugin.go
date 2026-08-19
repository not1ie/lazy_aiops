package remediation

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/pkg/plugin"
	"github.com/lazyautoops/lazy-auto-ops/plugins/alert"
	"github.com/lazyautoops/lazy-auto-ops/plugins/cmdb"
	"gorm.io/gorm"
)

func init() {
	plugin.Register("remediation", func() plugin.Plugin {
		return &RemediationPlugin{}
	})
}

type RemediationPlugin struct {
	core    *core.Core
	db      *gorm.DB
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	running bool
}

func (p *RemediationPlugin) Name() string    { return "remediation" }
func (p *RemediationPlugin) Version() string { return "1.0.0" }
func (p *RemediationPlugin) Description() string {
	return "故障自愈 - 监听告警并自动执行修复脚本"
}

func (p *RemediationPlugin) Init(c *core.Core, cfg map[string]interface{}) error {
	p.core = c
	p.db = c.DB
	p.ctx, p.cancel = context.WithCancel(context.Background())
	return nil
}

func (p *RemediationPlugin) Start() error {
	p.running = true
	p.wg.Add(1)
	go p.worker()
	return nil
}

func (p *RemediationPlugin) Stop() error {
	p.cancel()
	p.wg.Wait()
	p.running = false
	return nil
}

func (p *RemediationPlugin) Migrate() error {
	return p.db.AutoMigrate(&RemediationLog{})
}

func (p *RemediationPlugin) RegisterRoutes(g *gin.RouterGroup) {
	h := &RemediationHandler{plugin: p, db: p.db}
	g.GET("/logs", h.ListLogs)
	g.GET("/logs/:id", h.GetLog)
	g.POST("/trigger/:alert_id", h.TriggerRemediation)
}

func (p *RemediationPlugin) worker() {
	defer p.wg.Done()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	log.Println("[Remediation] Worker started")

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.checkAndRemediate()
		}
	}
}

func (p *RemediationPlugin) checkAndRemediate() {
	var alerts []alert.Alert

	// 查找 status = 0 (触发中) 的告警，并且对应规则开启了 auto_recover
	// 并且没有在 running 或最近成功的自愈日志
	err := p.db.Table("alerts").
		Select("alerts.*").
		Joins("JOIN alert_rules ON alerts.rule_id = alert_rules.id").
		Where("alerts.status = ? AND alert_rules.auto_recover = ?", 0, true).
		Where("alerts.id NOT IN (SELECT alert_id FROM remediation_logs WHERE status IN ('success', 'running'))").
		Find(&alerts).Error

	if err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Printf("[Remediation] Failed to query alerts: %v", err)
		}
		return
	}

	for _, a := range alerts {
		p.wg.Add(1)
		go func(alt alert.Alert) {
			defer p.wg.Done()
			p.executeRemediation(alt)
		}(a)
	}
}

func (p *RemediationPlugin) executeRemediation(a alert.Alert) {
	// 获取规则
	var rule alert.AlertRule
	if err := p.db.First(&rule, "id = ?", a.RuleID).Error; err != nil {
		return
	}

	if rule.RecoverScript == "" {
		return
	}

	log.Printf("[Remediation] Starting recovery for alert %s (%s) on %s", a.ID, rule.Name, a.Target)

	// 创建日志
	remLog := &RemediationLog{
		AlertID:   a.ID,
		RuleID:    rule.ID,
		Target:    a.Target,
		Action:    rule.RecoverScript,
		Status:    "running",
		StartedAt: time.Now(),
	}
	p.db.Create(remLog)

	// 查找主机及其凭据
	var host cmdb.Host
	err := p.db.Preload("Credential").Where("ip = ? OR name = ?", a.Target, a.Target).First(&host).Error

	var stdout, stderr string
	var execErr error

	if err != nil {
		stdout = ""
		stderr = fmt.Sprintf("未在 CMDB 中找到主机 %s: %v", a.Target, err)
		execErr = err
	} else if host.Credential == nil {
		stdout = ""
		stderr = "主机未关联凭据"
		execErr = fmt.Errorf("no credential")
	} else {
		if err := cmdb.DecryptCredentialFields(p.core.Config.JWT.Secret, host.Credential); err != nil {
			stdout = ""
			stderr = "主机凭据解密失败"
			execErr = err
		}
	}

	if execErr == nil && host.Credential != nil {
		client := &core.SSHClient{
			Host:     host.IP,
			Port:     host.Port,
			Username: host.Credential.Username,
			Password: host.Credential.Password,
			Key:      host.Credential.PrivateKey,
			Timeout:  45 * time.Second,
		}
		stdout, stderr, execErr = client.ExecuteWithPool(rule.RecoverScript)
	}

	now := time.Now()
	remLog.FinishedAt = &now
	remLog.Duration = int(now.Sub(remLog.StartedAt).Seconds())
	remLog.Stdout = stdout
	remLog.Stderr = stderr

	if execErr != nil {
		remLog.Status = "failed"
		remLog.Error = execErr.Error()
	} else {
		remLog.Status = "success"
		// 自动恢复告警状态
		p.db.Model(&alert.Alert{}).Where("id = ?", a.ID).Updates(map[string]interface{}{
			"status":        2, // 已恢复
			"resolved_at":   now,
			"status_reason": "故障自愈脚本执行成功，告警已自动恢复",
		})
	}

	p.db.Save(remLog)
	log.Printf("[Remediation] Finished recovery for alert %s with status %s", a.ID, remLog.Status)
}

type RemediationHandler struct {
	plugin *RemediationPlugin
	db     *gorm.DB
}

func (h *RemediationHandler) ListLogs(c *gin.Context) {
	query := h.db.Order("created_at DESC")
	if alertID := c.Query("alert_id"); alertID != "" {
		query = query.Where("alert_id = ?", alertID)
	}
	if target := c.Query("target"); target != "" {
		query = query.Where("target LIKE ?", "%"+target+"%")
	}
	var logs []RemediationLog
	query.Limit(100).Find(&logs)
	c.JSON(200, gin.H{"code": 0, "data": logs})
}

func (h *RemediationHandler) GetLog(c *gin.Context) {
	var log RemediationLog
	if err := h.db.First(&log, c.Param("id")).Error; err != nil {
		c.JSON(404, gin.H{"code": 404, "message": "日志不存在"})
		return
	}
	c.JSON(200, gin.H{"code": 0, "data": log})
}

func (h *RemediationHandler) TriggerRemediation(c *gin.Context) {
	alertID := c.Param("alert_id")
	var a alert.Alert
	if err := h.db.First(&a, "id = ?", alertID).Error; err != nil {
		c.JSON(404, gin.H{"code": 404, "message": "告警事件不存在"})
		return
	}
	go h.plugin.executeRemediation(a)
	c.JSON(200, gin.H{"code": 0, "message": "已触发故障自愈执行"})
}
