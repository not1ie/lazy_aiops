package ai

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/lazyautoops/lazy-auto-ops/plugins/alert"
	"github.com/lazyautoops/lazy-auto-ops/plugins/cmdb"
	"github.com/lazyautoops/lazy-auto-ops/plugins/docker"
	"github.com/lazyautoops/lazy-auto-ops/plugins/k8s"
	"github.com/lazyautoops/lazy-auto-ops/plugins/monitor"
	"github.com/lazyautoops/lazy-auto-ops/plugins/workflow"
	"github.com/lazyautoops/lazy-auto-ops/plugins/workorder"
)

func normalizeContextHint(hint *AIContextHint) *AIContextHint {
	if hint == nil {
		return nil
	}
	query := map[string]string{}
	for key, value := range hint.Query {
		k := strings.TrimSpace(key)
		v := strings.TrimSpace(value)
		if k == "" || v == "" {
			continue
		}
		query[k] = v
	}
	return &AIContextHint{
		Path:     strings.TrimSpace(hint.Path),
		FullPath: strings.TrimSpace(hint.FullPath),
		Title:    strings.TrimSpace(hint.Title),
		Query:    query,
	}
}

func detectContextScope(path string) string {
	switch {
	case strings.HasPrefix(path, "/host"),
		strings.HasPrefix(path, "/cmdb"),
		strings.HasPrefix(path, "/asset"),
		strings.HasPrefix(path, "/firewall"),
		strings.HasPrefix(path, "/jump"),
		strings.HasPrefix(path, "/terminal"):
		return "asset"
	case strings.HasPrefix(path, "/k8s"),
		strings.HasPrefix(path, "/docker"):
		return "k8s"
	case strings.HasPrefix(path, "/monitor"),
		strings.HasPrefix(path, "/alert"),
		strings.HasPrefix(path, "/notify"),
		strings.HasPrefix(path, "/domain"),
		strings.HasPrefix(path, "/topology"),
		strings.HasPrefix(path, "/cost"):
		return "monitor"
	case strings.HasPrefix(path, "/delivery"),
		strings.HasPrefix(path, "/cicd"),
		strings.HasPrefix(path, "/workorder"),
		strings.HasPrefix(path, "/workflow"),
		strings.HasPrefix(path, "/executor"),
		strings.HasPrefix(path, "/task"),
		strings.HasPrefix(path, "/oncall"),
		strings.HasPrefix(path, "/application"),
		strings.HasPrefix(path, "/sqlaudit"),
		strings.HasPrefix(path, "/gitops"),
		strings.HasPrefix(path, "/ansible"),
		strings.HasPrefix(path, "/nacos"):
		return "delivery"
	case strings.HasPrefix(path, "/system"):
		return "system"
	default:
		return "general"
	}
}

func (s *AIService) BuildContextPack(hint *AIContextHint) (*AIContextPack, error) {
	normalized := normalizeContextHint(hint)
	scope := "general"
	title := "平台上下文"
	if normalized != nil {
		scope = detectContextScope(normalized.Path)
		if normalized.Title != "" {
			title = normalized.Title
		}
	}

	pack := &AIContextPack{
		Scope:       scope,
		Title:       title,
		Highlights:  []string{},
		Facts:       map[string]string{},
		Route:       normalized,
		GeneratedAt: time.Now(),
	}

	var err error
	switch scope {
	case "asset":
		err = s.fillAssetContext(pack, normalized)
	case "k8s":
		err = s.fillK8sContext(pack, normalized)
	case "monitor":
		err = s.fillMonitorContext(pack, normalized)
	case "delivery":
		err = s.fillDeliveryContext(pack, normalized)
	case "system":
		pack.Summary = "当前位于系统管理场景，建议结合账号、权限、审计和登录日志来判断问题范围。"
		pack.Highlights = append(pack.Highlights, "适合排查账号权限、验证码、审计日志与登录异常。")
	default:
		pack.Summary = "当前没有识别到明确的业务场景，将按平台通用运维问答模式处理。"
		pack.Highlights = append(pack.Highlights, "如果你希望获得更准的建议，建议先打开主机、K8s、监控或交付页面再进入 AI 助手。")
	}
	if err != nil {
		return nil, err
	}
	return pack, nil
}

func (s *AIService) fillAssetContext(pack *AIContextPack, hint *AIContextHint) error {
	var totalHosts, onlineHosts, maintenanceHosts, hostGroups, dockerHosts int64
	if err := s.db.Model(&cmdb.Host{}).Count(&totalHosts).Error; err != nil {
		return err
	}
	if err := s.db.Model(&cmdb.Host{}).Where("status = ?", 1).Count(&onlineHosts).Error; err != nil {
		return err
	}
	if err := s.db.Model(&cmdb.Host{}).Where("status = ?", 2).Count(&maintenanceHosts).Error; err != nil {
		return err
	}
	if err := s.db.Model(&cmdb.HostGroup{}).Count(&hostGroups).Error; err != nil {
		return err
	}
	if err := s.db.Model(&docker.DockerHost{}).Count(&dockerHosts).Error; err != nil {
		return err
	}

	var recentHosts []cmdb.Host
	if err := s.db.Order("updated_at desc").Limit(3).Find(&recentHosts).Error; err != nil {
		return err
	}

	pack.Facts["hosts_total"] = fmt.Sprintf("%d", totalHosts)
	pack.Facts["hosts_online"] = fmt.Sprintf("%d", onlineHosts)
	pack.Facts["hosts_maintenance"] = fmt.Sprintf("%d", maintenanceHosts)
	pack.Facts["host_groups"] = fmt.Sprintf("%d", hostGroups)
	pack.Facts["docker_hosts"] = fmt.Sprintf("%d", dockerHosts)
	pack.Summary = fmt.Sprintf("资产域当前共有 %d 台主机，其中在线 %d 台、维护中 %d 台，分组 %d 个，已接入 Docker 主机 %d 台。", totalHosts, onlineHosts, maintenanceHosts, hostGroups, dockerHosts)
	if len(recentHosts) > 0 {
		names := make([]string, 0, len(recentHosts))
		for _, item := range recentHosts {
			names = append(names, strings.TrimSpace(fmt.Sprintf("%s(%s)", item.Name, item.IP)))
		}
		pack.Highlights = append(pack.Highlights, "最近更新主机: "+strings.Join(names, "、"))
	}
	if hint != nil && hint.FullPath != "" {
		pack.Highlights = append(pack.Highlights, "当前关注页面: "+hint.FullPath)
	}
	return nil
}

func (s *AIService) fillK8sContext(pack *AIContextPack, hint *AIContextHint) error {
	var totalClusters, healthyClusters, abnormalClusters, dockerHosts int64
	if err := s.db.Model(&k8s.Cluster{}).Count(&totalClusters).Error; err != nil {
		return err
	}
	if err := s.db.Model(&k8s.Cluster{}).Where("status = ?", 1).Count(&healthyClusters).Error; err != nil {
		return err
	}
	if err := s.db.Model(&k8s.Cluster{}).Where("status <> ?", 1).Count(&abnormalClusters).Error; err != nil {
		return err
	}
	if err := s.db.Model(&docker.DockerHost{}).Where("status = ?", "online").Count(&dockerHosts).Error; err != nil {
		return err
	}

	var clusters []k8s.Cluster
	if err := s.db.Order("updated_at desc").Limit(3).Find(&clusters).Error; err != nil {
		return err
	}

	pack.Facts["clusters_total"] = fmt.Sprintf("%d", totalClusters)
	pack.Facts["clusters_healthy"] = fmt.Sprintf("%d", healthyClusters)
	pack.Facts["clusters_abnormal"] = fmt.Sprintf("%d", abnormalClusters)
	pack.Facts["docker_hosts_online"] = fmt.Sprintf("%d", dockerHosts)
	pack.Summary = fmt.Sprintf("容器域当前共有 %d 个 K8s 集群，其中健康 %d 个、异常或维护 %d 个，在线 Docker 环境 %d 个。", totalClusters, healthyClusters, abnormalClusters, dockerHosts)
	if len(clusters) > 0 {
		names := make([]string, 0, len(clusters))
		for _, item := range clusters {
			label := strings.TrimSpace(item.DisplayName)
			if label == "" {
				label = item.Name
			}
			names = append(names, label)
		}
		pack.Highlights = append(pack.Highlights, "最近活跃集群: "+strings.Join(names, "、"))
	}
	if hint != nil && len(hint.Query) > 0 {
		pack.Highlights = append(pack.Highlights, "页面参数: "+formatQueryPairs(hint.Query))
	}
	return nil
}

func (s *AIService) fillMonitorContext(pack *AIContextPack, hint *AIContextHint) error {
	var openAlerts, criticalAlerts, warningAlerts, enabledRules, onlineAgents, totalAgents int64
	if err := s.db.Model(&alert.Alert{}).Where("status in ?", []int{0, 1}).Count(&openAlerts).Error; err != nil {
		return err
	}
	if err := s.db.Model(&alert.Alert{}).Where("status in ? AND severity = ?", []int{0, 1}, "critical").Count(&criticalAlerts).Error; err != nil {
		return err
	}
	if err := s.db.Model(&alert.Alert{}).Where("status in ? AND severity = ?", []int{0, 1}, "warning").Count(&warningAlerts).Error; err != nil {
		return err
	}
	if err := s.db.Model(&alert.AlertRule{}).Where("enabled = ?", true).Count(&enabledRules).Error; err != nil {
		return err
	}
	if err := s.db.Model(&monitor.AgentHeartbeat{}).Count(&totalAgents).Error; err != nil {
		return err
	}
	if err := s.db.Model(&monitor.AgentHeartbeat{}).Where("status = ?", "online").Count(&onlineAgents).Error; err != nil {
		return err
	}

	var latestAlerts []alert.Alert
	if err := s.db.Order("fired_at desc").Limit(3).Find(&latestAlerts).Error; err != nil {
		return err
	}

	pack.Facts["alerts_open"] = fmt.Sprintf("%d", openAlerts)
	pack.Facts["alerts_critical"] = fmt.Sprintf("%d", criticalAlerts)
	pack.Facts["alerts_warning"] = fmt.Sprintf("%d", warningAlerts)
	pack.Facts["rules_enabled"] = fmt.Sprintf("%d", enabledRules)
	pack.Facts["agents_online"] = fmt.Sprintf("%d", onlineAgents)
	pack.Facts["agents_total"] = fmt.Sprintf("%d", totalAgents)
	pack.Summary = fmt.Sprintf("监控域当前存在未恢复或未确认告警 %d 条，其中 critical %d 条、warning %d 条；启用规则 %d 条，在线 Agent %d/%d。", openAlerts, criticalAlerts, warningAlerts, enabledRules, onlineAgents, totalAgents)
	if len(latestAlerts) > 0 {
		items := make([]string, 0, len(latestAlerts))
		for _, item := range latestAlerts {
			items = append(items, strings.TrimSpace(fmt.Sprintf("%s/%s", item.RuleName, item.Target)))
		}
		pack.Highlights = append(pack.Highlights, "最近告警: "+strings.Join(items, "、"))
	}
	if hint != nil && hint.Title != "" {
		pack.Highlights = append(pack.Highlights, "当前监控视角: "+hint.Title)
	}
	return nil
}

func (s *AIService) fillDeliveryContext(pack *AIContextPack, hint *AIContextHint) error {
	var pendingOrders, runningOrders, completedOrders, waitingApproval, runningExecutions, enabledWorkflows int64
	if err := s.db.Model(&workorder.WorkOrder{}).Where("status in ?", []int{0, 1}).Count(&pendingOrders).Error; err != nil {
		return err
	}
	if err := s.db.Model(&workorder.WorkOrder{}).Where("status = ?", 4).Count(&runningOrders).Error; err != nil {
		return err
	}
	if err := s.db.Model(&workorder.WorkOrder{}).Where("status = ?", 5).Count(&completedOrders).Error; err != nil {
		return err
	}
	if err := s.db.Model(&workflow.WorkflowExecution{}).Where("status = ?", 4).Count(&waitingApproval).Error; err != nil {
		return err
	}
	if err := s.db.Model(&workflow.WorkflowExecution{}).Where("status = ?", 0).Count(&runningExecutions).Error; err != nil {
		return err
	}
	if err := s.db.Model(&workflow.Workflow{}).Where("enabled = ?", true).Count(&enabledWorkflows).Error; err != nil {
		return err
	}

	var latestOrders []workorder.WorkOrder
	if err := s.db.Order("updated_at desc").Limit(3).Find(&latestOrders).Error; err != nil {
		return err
	}

	pack.Facts["orders_pending"] = fmt.Sprintf("%d", pendingOrders)
	pack.Facts["orders_running"] = fmt.Sprintf("%d", runningOrders)
	pack.Facts["orders_completed"] = fmt.Sprintf("%d", completedOrders)
	pack.Facts["workflow_waiting_approval"] = fmt.Sprintf("%d", waitingApproval)
	pack.Facts["workflow_running"] = fmt.Sprintf("%d", runningExecutions)
	pack.Facts["workflow_enabled"] = fmt.Sprintf("%d", enabledWorkflows)
	pack.Summary = fmt.Sprintf("交付域当前待处理工单 %d 个、执行中工单 %d 个、已完成工单 %d 个；运行中流程 %d 个，等待审批流程 %d 个，已启用工作流 %d 个。", pendingOrders, runningOrders, completedOrders, runningExecutions, waitingApproval, enabledWorkflows)
	if len(latestOrders) > 0 {
		items := make([]string, 0, len(latestOrders))
		for _, item := range latestOrders {
			items = append(items, strings.TrimSpace(item.Title))
		}
		pack.Highlights = append(pack.Highlights, "最近工单: "+strings.Join(items, "、"))
	}
	if hint != nil && hint.FullPath != "" {
		pack.Highlights = append(pack.Highlights, "当前交付页面: "+hint.FullPath)
	}
	return nil
}

func formatQueryPairs(query map[string]string) string {
	if len(query) == 0 {
		return "-"
	}
	keys := make([]string, 0, len(query))
	for key := range query {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	pairs := make([]string, 0, len(keys))
	for _, key := range keys {
		pairs = append(pairs, fmt.Sprintf("%s=%s", key, query[key]))
	}
	return strings.Join(pairs, ", ")
}

func formatContextPack(pack *AIContextPack) string {
	if pack == nil {
		return ""
	}
	lines := []string{
		fmt.Sprintf("场景: %s", pack.Scope),
		fmt.Sprintf("页面: %s", strings.TrimSpace(pack.Title)),
		fmt.Sprintf("摘要: %s", strings.TrimSpace(pack.Summary)),
	}
	if pack.Route != nil && strings.TrimSpace(pack.Route.FullPath) != "" {
		lines = append(lines, fmt.Sprintf("路由: %s", pack.Route.FullPath))
	}
	if len(pack.Highlights) > 0 {
		lines = append(lines, "要点:")
		for _, item := range pack.Highlights {
			lines = append(lines, "- "+strings.TrimSpace(item))
		}
	}
	if len(pack.Facts) > 0 {
		keys := make([]string, 0, len(pack.Facts))
		for key := range pack.Facts {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		lines = append(lines, "事实:")
		for _, key := range keys {
			lines = append(lines, fmt.Sprintf("- %s: %s", key, pack.Facts[key]))
		}
	}
	return strings.Join(lines, "\n")
}
