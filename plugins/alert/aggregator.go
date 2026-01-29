package alert

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

// Aggregator 告警聚合器 - AI智能降噪
type Aggregator struct {
	db          *gorm.DB
	mu          sync.RWMutex
	groups      map[string]*AlertGroup // 聚合组
	interval    time.Duration          // 聚合窗口
	aiEnabled   bool
	aiAnalyzer  func(alerts []*Alert) string
}

type AlertGroup struct {
	Key       string
	Alerts    []*Alert
	FirstSeen time.Time
	LastSeen  time.Time
	Count     int
	Notified  bool
}

func NewAggregator(db *gorm.DB, intervalSec int) *Aggregator {
	if intervalSec <= 0 {
		intervalSec = 60 // 默认60秒
	}
	return &Aggregator{
		db:       db,
		groups:   make(map[string]*AlertGroup),
		interval: time.Duration(intervalSec) * time.Second,
	}
}

// SetAIAnalyzer 设置AI分析器
func (a *Aggregator) SetAIAnalyzer(analyzer func(alerts []*Alert) string) {
	a.aiAnalyzer = analyzer
	a.aiEnabled = true
}

// Process 处理告警
func (a *Aggregator) Process(alert *Alert) (*Alert, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// 生成指纹
	alert.Fingerprint = a.generateFingerprint(alert)

	// 检查是否被静默
	if a.isSilenced(alert) {
		alert.Status = 3 // 已抑制
		a.db.Create(alert)
		return alert, false
	}

	// 生成聚合键
	alert.GroupKey = a.generateGroupKey(alert)

	// 查找现有聚合组
	group, exists := a.groups[alert.GroupKey]
	if !exists {
		// 新建聚合组
		group = &AlertGroup{
			Key:       alert.GroupKey,
			Alerts:    make([]*Alert, 0),
			FirstSeen: time.Now(),
			Count:     0,
		}
		a.groups[alert.GroupKey] = group
	}

	// 检查是否是重复告警
	for _, existing := range group.Alerts {
		if existing.Fingerprint == alert.Fingerprint && existing.Status == 0 {
			// 更新计数
			existing.Count++
			a.db.Model(existing).Update("count", existing.Count)
			return existing, false // 不需要发送通知
		}
	}

	// 添加到聚合组
	group.Alerts = append(group.Alerts, alert)
	group.LastSeen = time.Now()
	group.Count++
	alert.Count = 1

	// 保存告警
	a.db.Create(alert)

	// 判断是否需要发送通知
	shouldNotify := false
	if !group.Notified {
		// 首次告警，立即通知
		shouldNotify = true
		group.Notified = true
	} else if time.Since(group.FirstSeen) > a.interval {
		// 超过聚合窗口，发送聚合通知
		shouldNotify = true
		group.FirstSeen = time.Now()
	}

	return alert, shouldNotify
}

// GetAggregatedAlerts 获取聚合后的告警
func (a *Aggregator) GetAggregatedAlerts(groupKey string) []*Alert {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if group, exists := a.groups[groupKey]; exists {
		return group.Alerts
	}
	return nil
}

// GenerateSummary 生成聚合摘要
func (a *Aggregator) GenerateSummary(groupKey string) string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	group, exists := a.groups[groupKey]
	if !exists || len(group.Alerts) == 0 {
		return ""
	}

	// 按严重程度统计
	severityCount := make(map[string]int)
	targetCount := make(map[string]int)
	for _, alert := range group.Alerts {
		severityCount[alert.Severity]++
		targetCount[alert.Target]++
	}

	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("📊 告警聚合摘要 (共%d条)\n\n", group.Count))

	// 严重程度分布
	summary.WriteString("**严重程度分布:**\n")
	for severity, count := range severityCount {
		icon := "ℹ️"
		switch severity {
		case "critical":
			icon = "🔴"
		case "warning":
			icon = "🟡"
		}
		summary.WriteString(fmt.Sprintf("- %s %s: %d条\n", icon, severity, count))
	}

	// 目标分布
	summary.WriteString("\n**告警目标:**\n")
	for target, count := range targetCount {
		summary.WriteString(fmt.Sprintf("- %s: %d条\n", target, count))
	}

	// AI分析
	if a.aiEnabled && a.aiAnalyzer != nil {
		analysis := a.aiAnalyzer(group.Alerts)
		if analysis != "" {
			summary.WriteString("\n**🤖 AI分析:**\n")
			summary.WriteString(analysis)
		}
	}

	return summary.String()
}

// Resolve 解决告警
func (a *Aggregator) Resolve(fingerprint string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	now := time.Now()
	for _, group := range a.groups {
		for i, alert := range group.Alerts {
			if alert.Fingerprint == fingerprint {
				alert.Status = 2
				alert.ResolvedAt = &now
				a.db.Model(alert).Updates(map[string]interface{}{
					"status":      2,
					"resolved_at": now,
				})
				// 从聚合组移除
				group.Alerts = append(group.Alerts[:i], group.Alerts[i+1:]...)
				return
			}
		}
	}
}

// Cleanup 清理过期的聚合组
func (a *Aggregator) Cleanup() {
	a.mu.Lock()
	defer a.mu.Unlock()

	threshold := time.Now().Add(-a.interval * 5)
	for key, group := range a.groups {
		if group.LastSeen.Before(threshold) && len(group.Alerts) == 0 {
			delete(a.groups, key)
		}
	}
}

func (a *Aggregator) generateFingerprint(alert *Alert) string {
	data := fmt.Sprintf("%s|%s|%s|%s", alert.RuleID, alert.Target, alert.Metric, alert.Severity)
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (a *Aggregator) generateGroupKey(alert *Alert) string {
	// 默认按规则+目标聚合
	return fmt.Sprintf("%s:%s", alert.RuleID, alert.Target)
}

func (a *Aggregator) isSilenced(alert *Alert) bool {
	var silences []AlertSilence
	now := time.Now()
	a.db.Where("status = 1 AND starts_at <= ? AND ends_at >= ?", now, now).Find(&silences)

	for _, silence := range silences {
		if a.matchSilence(alert, &silence) {
			return true
		}
	}
	return false
}

func (a *Aggregator) matchSilence(alert *Alert, silence *AlertSilence) bool {
	var matchers []map[string]string
	if err := json.Unmarshal([]byte(silence.Matchers), &matchers); err != nil {
		return false
	}

	alertLabels := make(map[string]string)
	if alert.Labels != "" {
		json.Unmarshal([]byte(alert.Labels), &alertLabels)
	}
	alertLabels["rule_id"] = alert.RuleID
	alertLabels["target"] = alert.Target
	alertLabels["severity"] = alert.Severity

	for _, matcher := range matchers {
		name := matcher["name"]
		value := matcher["value"]
		op := matcher["op"]
		if op == "" {
			op = "="
		}

		alertValue, exists := alertLabels[name]
		if !exists {
			return false
		}

		switch op {
		case "=":
			if alertValue != value {
				return false
			}
		case "!=":
			if alertValue == value {
				return false
			}
		case "=~":
			// 正则匹配，简化处理
			if !strings.Contains(alertValue, value) {
				return false
			}
		}
	}

	return true
}

// GetStats 获取聚合统计
func (a *Aggregator) GetStats() map[string]interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()

	totalGroups := len(a.groups)
	totalAlerts := 0
	severityStats := make(map[string]int)

	for _, group := range a.groups {
		totalAlerts += len(group.Alerts)
		for _, alert := range group.Alerts {
			severityStats[alert.Severity]++
		}
	}

	// 按severity排序
	severities := []string{"critical", "warning", "info"}
	sortedStats := make([]map[string]interface{}, 0)
	for _, s := range severities {
		if count, ok := severityStats[s]; ok {
			sortedStats = append(sortedStats, map[string]interface{}{
				"severity": s,
				"count":    count,
			})
		}
	}

	return map[string]interface{}{
		"total_groups":   totalGroups,
		"total_alerts":   totalAlerts,
		"severity_stats": sortedStats,
	}
}

// SortAlertsBySeverity 按严重程度排序
func SortAlertsBySeverity(alerts []*Alert) {
	severityOrder := map[string]int{
		"critical": 0,
		"warning":  1,
		"info":     2,
	}
	sort.Slice(alerts, func(i, j int) bool {
		return severityOrder[alerts[i].Severity] < severityOrder[alerts[j].Severity]
	})
}
