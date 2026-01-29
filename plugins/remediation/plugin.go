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

func (p *RemediationPlugin) Name() string        { return "remediation" }
func (p *RemediationPlugin) Version() string     { return "1.0.0" }
func (p *RemediationPlugin) Description() string { return "故障自愈 - 监听告警并自动执行修复脚本" }

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
	h := &RemediationHandler{db: p.db}
	g.GET("/logs", h.ListLogs)
	g.GET("/logs/:id", h.GetLog)
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
	// 1. 查找未处理且开启自愈的告警
	// 我们需要关联 Alert 和 AlertRule
	var alerts []alert.Alert
	
	// 查找 status = 0 (触发中) 的告警
	// 并且没有对应的成功或正在运行的 RemediationLog
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
	// 假设 Target 是主机名或 IP
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
		// 执行远程脚本
		client := &core.SSHClient{
			Host:     host.IP,
			Port:     host.Port,
			Username: host.Credential.Username,
			Password: host.Credential.Password,
			Key:      host.Credential.PrivateKey,
			Timeout:  30 * time.Second,
		}
		stdout, stderr, execErr = client.Execute(rule.RecoverScript)
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
		// 自动解决告警
		p.db.Model(&a).Updates(map[string]interface{}{
			"status":      2, // 已恢复
			"resolved_at": now,
		})
	}
	
	p.db.Save(remLog)
	log.Printf("[Remediation] Finished recovery for alert %s with status %s", a.ID, remLog.Status)
}

type RemediationHandler struct {
	db *gorm.DB
}

func (h *RemediationHandler) ListLogs(c *gin.Context) {
	var logs []RemediationLog
	h.db.Order("created_at DESC").Limit(100).Find(&logs)
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
