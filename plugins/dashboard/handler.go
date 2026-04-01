package dashboard

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/plugins/alert"
	"github.com/lazyautoops/lazy-auto-ops/plugins/cicd"
	"github.com/lazyautoops/lazy-auto-ops/plugins/cmdb"
	"github.com/lazyautoops/lazy-auto-ops/plugins/docker"
	"github.com/lazyautoops/lazy-auto-ops/plugins/domain"
	"github.com/lazyautoops/lazy-auto-ops/plugins/firewall"
	"github.com/lazyautoops/lazy-auto-ops/plugins/jump"
	"github.com/lazyautoops/lazy-auto-ops/plugins/k8s"
	"github.com/lazyautoops/lazy-auto-ops/plugins/monitor"
	"github.com/lazyautoops/lazy-auto-ops/plugins/task"
	"github.com/lazyautoops/lazy-auto-ops/plugins/terminal"
	"github.com/lazyautoops/lazy-auto-ops/plugins/workflow"
	"github.com/lazyautoops/lazy-auto-ops/plugins/workorder"
	"gorm.io/gorm"
)

const overviewContractVersion = "2026-04-01.v1"

type DashboardHandler struct {
	db           *gorm.DB
	agentTimeout time.Duration
}

type overviewResponse struct {
	ContractVersion string            `json:"contract_version"`
	GeneratedAt     time.Time         `json:"generated_at"`
	Hours           int               `json:"hours"`
	StatusContract  statusContract    `json:"status_contract"`
	Summary         overviewSummary   `json:"summary"`
	Quality         overviewQuality   `json:"quality"`
	Snapshots       overviewSnapshots `json:"snapshots"`
	SourceErrors    map[string]string `json:"source_errors"`
}

type statusContract struct {
	HostStaleMinutes    int `json:"host_stale_minutes"`
	DockerStaleMinutes  int `json:"docker_stale_minutes"`
	K8sStaleMinutes     int `json:"k8s_stale_minutes"`
	NetworkStaleMinutes int `json:"network_stale_minutes"`
	FirewallStaleMin    int `json:"firewall_stale_minutes"`
	DomainStaleHours    int `json:"domain_stale_hours"`
	AgentOfflineSeconds int `json:"agent_offline_seconds"`
}

type overviewSummary struct {
	HostTotal       int               `json:"host_total"`
	HostOnline      int               `json:"host_online"`
	HostOffline     int               `json:"host_offline"`
	HostStale       int               `json:"host_stale"`
	DockerTotal     int               `json:"docker_total"`
	DockerOnline    int               `json:"docker_online"`
	DockerOffline   int               `json:"docker_offline"`
	DockerStale     int               `json:"docker_stale"`
	K8sTotal        int               `json:"k8s_total"`
	K8sHealthy      int               `json:"k8s_healthy"`
	K8sUnhealthy    int               `json:"k8s_unhealthy"`
	K8sMaintenance  int               `json:"k8s_maintenance"`
	K8sStale        int               `json:"k8s_stale"`
	FirewallTotal   int               `json:"firewall_total"`
	FirewallOnline  int               `json:"firewall_online"`
	FirewallOffline int               `json:"firewall_offline"`
	FirewallAlert   int               `json:"firewall_alert"`
	FirewallStale   int               `json:"firewall_stale"`
	DomainTotal     int               `json:"domain_total"`
	DomainHealthy   int               `json:"domain_healthy"`
	DomainWarning   int               `json:"domain_warning"`
	DomainCritical  int               `json:"domain_critical"`
	DomainStale     int               `json:"domain_stale"`
	AlertTotal      int               `json:"alert_total"`
	AlertOpen       int               `json:"alert_open"`
	TaskTotal       int               `json:"task_total"`
	TaskEnabled     int               `json:"task_enabled"`
	AgentTotal      int               `json:"agent_total"`
	AgentOnline     int               `json:"agent_online"`
	ModuleStatus    map[string]string `json:"module_status"`
}

type overviewQuality struct {
	TrustScore                 int                `json:"trust_score"`
	TrustGrade                 string             `json:"trust_grade"`
	Summary                    string             `json:"summary"`
	Dimensions                 []qualityDimension `json:"dimensions"`
	ActionHints                []actionHint       `json:"action_hints"`
	CompletenessProblemModules int                `json:"completeness_problem_modules"`
	ConsistencyIssues          int                `json:"consistency_issues"`
	SourceErrorCount           int                `json:"source_error_count"`
}

type qualityDimension struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Score  int    `json:"score"`
	Detail string `json:"detail"`
}

type actionHint struct {
	Key           string `json:"key"`
	Priority      int    `json:"priority"`
	PriorityLabel string `json:"priority_label"`
	Module        string `json:"module"`
	Title         string `json:"title"`
	Reason        string `json:"reason"`
	Path          string `json:"path"`
	Action        string `json:"action"`
}

type overviewSnapshots struct {
	Hosts              []hostSnapshot              `json:"hosts"`
	DockerHosts        []dockerHostSnapshot        `json:"docker_hosts"`
	K8sClusters        []k8sClusterSnapshot        `json:"k8s_clusters"`
	Alerts             []alertSnapshot             `json:"alerts"`
	Tasks              []taskSnapshot              `json:"tasks"`
	Agents             []agentSnapshot             `json:"agents"`
	Metrics            metricSnapshot              `json:"metrics"`
	MetricHistory      []metricHistorySnapshot     `json:"metric_history"`
	NetworkDevices     []networkDeviceSnapshot     `json:"network_devices"`
	Firewalls          []firewallSnapshot          `json:"firewalls"`
	JumpSessions       []jumpSessionSnapshot       `json:"jump_sessions"`
	JumpRiskEvents     []jumpRiskEventSnapshot     `json:"jump_risk_events"`
	Domains            []domainSnapshot            `json:"domains"`
	Certs              []certSnapshot              `json:"certs"`
	CICDExecutions     []cicdExecutionSnapshot     `json:"cicd_executions"`
	CICDSchedules      []cicdScheduleSnapshot      `json:"cicd_schedules"`
	Workorders         []workorderSnapshot         `json:"workorders"`
	WorkflowExecutions []workflowExecutionSnapshot `json:"workflow_executions"`
	TerminalSessions   []terminalSessionSnapshot   `json:"terminal_sessions"`
	JumpAssets         []jumpAssetSnapshot         `json:"jump_assets"`
	JumpIntegration    jumpIntegrationSnapshot     `json:"jump_integration"`
}

type hostSnapshot struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	IP           string     `json:"ip"`
	Status       int        `json:"status"`
	LastCheckAt  *time.Time `json:"last_check_at"`
	LastOnlineAt *time.Time `json:"last_online_at"`
	StatusReason string     `json:"status_reason"`
	GroupID      string     `json:"group_id"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type dockerHostSnapshot struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	HostID       string     `json:"host_id"`
	Status       string     `json:"status"`
	LastCheckAt  *time.Time `json:"last_check_at"`
	LastOnlineAt *time.Time `json:"last_online_at"`
	LastError    string     `json:"last_error"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type k8sClusterSnapshot struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	DisplayName  string     `json:"display_name"`
	Status       int        `json:"status"`
	LastCheckAt  *time.Time `json:"last_check_at"`
	LastOnlineAt *time.Time `json:"last_online_at"`
	StatusReason string     `json:"status_reason"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type alertSnapshot struct {
	ID        string    `json:"id"`
	RuleName  string    `json:"rule_name"`
	Target    string    `json:"target"`
	Severity  string    `json:"severity"`
	Status    int       `json:"status"`
	FiredAt   time.Time `json:"fired_at"`
	CreatedAt time.Time `json:"created_at"`
}

type taskSnapshot struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Enabled   bool      `json:"enabled"`
	UpdatedAt time.Time `json:"updated_at"`
}

type agentSnapshot struct {
	ID       string    `json:"id"`
	AgentID  string    `json:"agent_id"`
	Hostname string    `json:"hostname"`
	IP       string    `json:"ip"`
	Status   string    `json:"status"`
	LastSeen time.Time `json:"last_seen"`
	CPU      float64   `json:"cpu"`
	Memory   float64   `json:"memory"`
	Disk     float64   `json:"disk"`
	NetIn    float64   `json:"net_in"`
	NetOut   float64   `json:"net_out"`
}

type metricSnapshot struct {
	CPU     float64 `json:"cpu"`
	Memory  float64 `json:"memory"`
	Disk    float64 `json:"disk"`
	Network float64 `json:"network"`
}

type metricHistorySnapshot struct {
	Timestamp   time.Time `json:"timestamp"`
	CPUUsage    float64   `json:"cpu_usage"`
	MemoryUsage float64   `json:"memory_usage"`
	DiskUsage   float64   `json:"disk_usage"`
}

type networkDeviceSnapshot struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	IP           string     `json:"ip"`
	DeviceType   string     `json:"device_type"`
	Vendor       string     `json:"vendor"`
	Status       int        `json:"status"`
	LastCheckAt  *time.Time `json:"last_check_at"`
	StatusReason string     `json:"status_reason"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type firewallSnapshot struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	IP           string     `json:"ip"`
	Vendor       string     `json:"vendor"`
	Status       int        `json:"status"`
	LastCheckAt  *time.Time `json:"last_check_at"`
	StatusReason string     `json:"status_reason"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type jumpSessionSnapshot struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	AssetName string    `json:"asset_name"`
	Username  string    `json:"username"`
	StartedAt time.Time `json:"started_at"`
	CreatedAt time.Time `json:"created_at"`
}

type jumpRiskEventSnapshot struct {
	ID        string    `json:"id"`
	Severity  string    `json:"severity"`
	EventType string    `json:"event_type"`
	AssetName string    `json:"asset_name"`
	Username  string    `json:"username"`
	FiredAt   time.Time `json:"fired_at"`
}

type domainSnapshot struct {
	ID             string     `json:"id"`
	Domain         string     `json:"domain"`
	HealthStatus   string     `json:"health_status"`
	HTTPStatusCode int        `json:"http_status_code"`
	ResponseTimeMS int        `json:"response_time_ms"`
	LastCheckAt    *time.Time `json:"last_check_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type certSnapshot struct {
	ID           string     `json:"id"`
	Domain       string     `json:"domain"`
	Status       int        `json:"status"`
	DaysToExpire int        `json:"days_to_expire"`
	LastCheckAt  *time.Time `json:"last_check_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type cicdExecutionSnapshot struct {
	ID         string     `json:"id"`
	Status     int        `json:"status"`
	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
}

type cicdScheduleSnapshot struct {
	ID        string     `json:"id"`
	Enabled   bool       `json:"enabled"`
	LastRunAt *time.Time `json:"last_run_at"`
	NextRunAt *time.Time `json:"next_run_at"`
}

type workorderSnapshot struct {
	ID        string    `json:"id"`
	Status    int       `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type workflowExecutionSnapshot struct {
	ID        string    `json:"id"`
	Status    int       `json:"status"`
	StartedAt time.Time `json:"started_at"`
}

type terminalSessionSnapshot struct {
	ID        string     `json:"id"`
	Status    int        `json:"status"`
	StartedAt *time.Time `json:"started_at"`
	CreatedAt time.Time  `json:"created_at"`
}

type jumpAssetSnapshot struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Source    string    `json:"source"`
	SourceRef string    `json:"source_ref"`
	Enabled   bool      `json:"enabled"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedAt time.Time `json:"created_at"`
}

type jumpIntegrationSnapshot struct {
	Enabled        bool       `json:"enabled"`
	BaseURL        string     `json:"base_url"`
	VerifyTLS      bool       `json:"verify_tls"`
	AutoSync       bool       `json:"auto_sync"`
	LastSyncStatus string     `json:"last_sync_status"`
	LastSyncMsg    string     `json:"last_sync_msg"`
	LastSyncAt     *time.Time `json:"last_sync_at"`
}

func NewDashboardHandler(db *gorm.DB, agentTimeout time.Duration) *DashboardHandler {
	if agentTimeout <= 0 {
		agentTimeout = 90 * time.Second
	}
	return &DashboardHandler{db: db, agentTimeout: agentTimeout}
}

func (h *DashboardHandler) GetOverview(c *gin.Context) {
	hours := clampInt(parseInt(c.Query("hours"), 24), 1, 168)
	now := time.Now()
	since := now.Add(-time.Duration(hours) * time.Hour)
	contract := statusContract{
		HostStaleMinutes:    3,
		DockerStaleMinutes:  3,
		K8sStaleMinutes:     15,
		NetworkStaleMinutes: 5,
		FirewallStaleMin:    5,
		DomainStaleHours:    24,
		AgentOfflineSeconds: int(h.agentTimeout / time.Second),
	}

	snapshots, sourceErrors := h.collectSnapshots(now, since, hours)
	summary := buildSummary(now, contract, snapshots, sourceErrors)
	quality := buildQuality(now, contract, summary, snapshots, sourceErrors)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": overviewResponse{
			ContractVersion: overviewContractVersion,
			GeneratedAt:     now,
			Hours:           hours,
			StatusContract:  contract,
			Summary:         summary,
			Quality:         quality,
			Snapshots:       snapshots,
			SourceErrors:    sourceErrors,
		},
	})
}

func (h *DashboardHandler) collectSnapshots(now, since time.Time, hours int) (overviewSnapshots, map[string]string) {
	out := overviewSnapshots{}
	sourceErrors := map[string]string{}

	if rows, err := h.listHosts(); err != nil {
		sourceErrors["hosts"] = err.Error()
	} else {
		out.Hosts = rows
	}
	if rows, err := h.listDockerHosts(); err != nil {
		sourceErrors["docker_hosts"] = err.Error()
	} else {
		out.DockerHosts = rows
	}
	if rows, err := h.listK8sClusters(); err != nil {
		sourceErrors["k8s_clusters"] = err.Error()
	} else {
		out.K8sClusters = rows
	}
	if rows, err := h.listAlerts(since); err != nil {
		sourceErrors["alerts"] = err.Error()
	} else {
		out.Alerts = rows
	}
	if rows, err := h.listTasks(); err != nil {
		sourceErrors["tasks"] = err.Error()
	} else {
		out.Tasks = rows
	}
	if rows, err := h.listAgents(now); err != nil {
		sourceErrors["agents"] = err.Error()
	} else {
		out.Agents = rows
	}
	if metrics, err := h.getRealtimeMetrics(out.Agents); err != nil {
		sourceErrors["metrics"] = err.Error()
	} else {
		out.Metrics = metrics
	}
	if rows, err := h.listMetricHistory(since, hours); err != nil {
		sourceErrors["metric_history"] = err.Error()
	} else {
		out.MetricHistory = rows
	}
	if rows, err := h.listNetworkDevices(); err != nil {
		sourceErrors["network_devices"] = err.Error()
	} else {
		out.NetworkDevices = rows
	}
	if rows, err := h.listFirewalls(); err != nil {
		sourceErrors["firewalls"] = err.Error()
	} else {
		out.Firewalls = rows
	}
	if rows, err := h.listJumpSessions(since); err != nil {
		sourceErrors["jump_sessions"] = err.Error()
	} else {
		out.JumpSessions = rows
	}
	if rows, err := h.listJumpRiskEvents(since); err != nil {
		sourceErrors["jump_risk_events"] = err.Error()
	} else {
		out.JumpRiskEvents = rows
	}
	if rows, err := h.listDomains(); err != nil {
		sourceErrors["domains"] = err.Error()
	} else {
		out.Domains = rows
	}
	if rows, err := h.listCerts(); err != nil {
		sourceErrors["certs"] = err.Error()
	} else {
		out.Certs = rows
	}
	if rows, err := h.listCICDExecutions(since); err != nil {
		sourceErrors["cicd_executions"] = err.Error()
	} else {
		out.CICDExecutions = rows
	}
	if rows, err := h.listCICDSchedules(); err != nil {
		sourceErrors["cicd_schedules"] = err.Error()
	} else {
		out.CICDSchedules = rows
	}
	if rows, err := h.listWorkorders(since); err != nil {
		sourceErrors["workorders"] = err.Error()
	} else {
		out.Workorders = rows
	}
	if rows, err := h.listWorkflowExecutions(since); err != nil {
		sourceErrors["workflow_executions"] = err.Error()
	} else {
		out.WorkflowExecutions = rows
	}
	if rows, err := h.listTerminalSessions(since); err != nil {
		sourceErrors["terminal_sessions"] = err.Error()
	} else {
		out.TerminalSessions = rows
	}
	if rows, err := h.listJumpAssets(); err != nil {
		sourceErrors["jump_assets"] = err.Error()
	} else {
		out.JumpAssets = rows
	}
	if config, err := h.getJumpIntegration(); err != nil {
		sourceErrors["jump_integration"] = err.Error()
	} else {
		out.JumpIntegration = config
	}

	return out, sourceErrors
}

func buildSummary(now time.Time, contract statusContract, snapshots overviewSnapshots, sourceErrors map[string]string) overviewSummary {
	summary := overviewSummary{
		HostTotal:     len(snapshots.Hosts),
		DockerTotal:   len(snapshots.DockerHosts),
		K8sTotal:      len(snapshots.K8sClusters),
		FirewallTotal: len(snapshots.Firewalls),
		DomainTotal:   len(snapshots.Domains),
		AlertTotal:    len(snapshots.Alerts),
		TaskTotal:     len(snapshots.Tasks),
		AgentTotal:    len(snapshots.Agents),
		ModuleStatus: map[string]string{
			"cmdb":     "ok",
			"docker":   "ok",
			"k8s":      "ok",
			"firewall": "ok",
			"domain":   "ok",
			"jump":     "ok",
			"monitor":  "ok",
			"task":     "ok",
		},
	}

	for _, item := range snapshots.Hosts {
		if item.Status == 1 {
			summary.HostOnline++
		} else {
			summary.HostOffline++
		}
		if staleWithFallback(now, time.Duration(contract.HostStaleMinutes)*time.Minute, item.UpdatedAt, item.LastCheckAt) {
			summary.HostStale++
		}
	}
	for _, item := range snapshots.DockerHosts {
		if normalizeText(item.Status) == "online" {
			summary.DockerOnline++
		} else {
			summary.DockerOffline++
		}
		if staleWithFallback(now, time.Duration(contract.DockerStaleMinutes)*time.Minute, item.UpdatedAt, item.LastCheckAt) {
			summary.DockerStale++
		}
	}
	for _, item := range snapshots.K8sClusters {
		switch item.Status {
		case 1:
			summary.K8sHealthy++
		case 2:
			summary.K8sMaintenance++
		default:
			summary.K8sUnhealthy++
		}
		if item.Status == 1 && staleWithFallback(now, time.Duration(contract.K8sStaleMinutes)*time.Minute, item.UpdatedAt, item.LastCheckAt) {
			summary.K8sStale++
		}
	}
	for _, item := range snapshots.Firewalls {
		switch item.Status {
		case 1:
			summary.FirewallOnline++
		case 2:
			summary.FirewallAlert++
		default:
			summary.FirewallOffline++
		}
		if item.Status == 1 && staleWithFallback(now, time.Duration(contract.FirewallStaleMin)*time.Minute, item.UpdatedAt, item.LastCheckAt) {
			summary.FirewallStale++
		}
	}
	for _, item := range snapshots.Domains {
		switch normalizeText(item.HealthStatus) {
		case "healthy":
			summary.DomainHealthy++
		case "warning":
			summary.DomainWarning++
		case "critical":
			summary.DomainCritical++
		}
		if staleWithFallback(now, time.Duration(contract.DomainStaleHours)*time.Hour, item.UpdatedAt, item.LastCheckAt) {
			summary.DomainStale++
		}
	}
	for _, item := range snapshots.Alerts {
		if item.Status == 0 || item.Status == 1 {
			summary.AlertOpen++
		}
	}
	for _, item := range snapshots.Tasks {
		if item.Enabled {
			summary.TaskEnabled++
		}
	}
	for _, item := range snapshots.Agents {
		if normalizeText(item.Status) == "online" {
			summary.AgentOnline++
		}
	}

	networkOffline := 0
	networkStale := 0
	for _, item := range snapshots.NetworkDevices {
		if item.Status != 1 {
			networkOffline++
		} else if staleWithFallback(now, time.Duration(contract.NetworkStaleMinutes)*time.Minute, item.UpdatedAt, item.LastCheckAt) {
			networkStale++
		}
	}

	if summary.HostOffline > 0 {
		summary.ModuleStatus["cmdb"] = "error"
	} else if summary.HostStale > 0 || networkOffline > 0 || networkStale > 0 {
		summary.ModuleStatus["cmdb"] = "warning"
	}
	if summary.DockerOffline > 0 {
		summary.ModuleStatus["docker"] = "error"
	} else if summary.DockerStale > 0 {
		summary.ModuleStatus["docker"] = "warning"
	}
	if summary.K8sUnhealthy > 0 {
		summary.ModuleStatus["k8s"] = "error"
	} else if summary.K8sStale > 0 || summary.K8sMaintenance > 0 {
		summary.ModuleStatus["k8s"] = "warning"
	}
	if summary.FirewallAlert > 0 {
		summary.ModuleStatus["firewall"] = "error"
	} else if summary.FirewallOffline > 0 || summary.FirewallStale > 0 {
		summary.ModuleStatus["firewall"] = "warning"
	}
	if summary.DomainCritical > 0 {
		summary.ModuleStatus["domain"] = "error"
	} else if summary.DomainWarning > 0 || summary.DomainStale > 0 {
		summary.ModuleStatus["domain"] = "warning"
	}

	jumpPendingTimeout := 0
	riskCritical := 0
	for _, item := range snapshots.JumpSessions {
		if normalizeText(item.Status) == "pending_approval" && now.Sub(item.StartedAt) >= 30*time.Minute {
			jumpPendingTimeout++
		}
	}
	for _, item := range snapshots.JumpRiskEvents {
		if normalizeText(item.Severity) == "critical" {
			riskCritical++
		}
	}
	jumpSyncFailed := normalizeText(snapshots.JumpIntegration.LastSyncStatus) == "failed"
	if riskCritical > 0 {
		summary.ModuleStatus["jump"] = "error"
	} else if jumpPendingTimeout > 0 || jumpSyncFailed {
		summary.ModuleStatus["jump"] = "warning"
	}

	if len(snapshots.MetricHistory) == 0 && summary.AgentOnline == 0 {
		summary.ModuleStatus["monitor"] = "warning"
	}
	if summary.TaskTotal == 0 {
		summary.ModuleStatus["task"] = "warning"
	}

	for key := range sourceErrors {
		switch key {
		case "hosts", "network_devices":
			summary.ModuleStatus["cmdb"] = "error"
		case "docker_hosts":
			summary.ModuleStatus["docker"] = "error"
		case "k8s_clusters":
			summary.ModuleStatus["k8s"] = "error"
		case "firewalls":
			summary.ModuleStatus["firewall"] = "error"
		case "domains", "certs":
			summary.ModuleStatus["domain"] = "error"
		case "jump_sessions", "jump_risk_events", "jump_assets", "jump_integration":
			summary.ModuleStatus["jump"] = "error"
		case "alerts", "metrics", "metric_history", "agents":
			summary.ModuleStatus["monitor"] = "error"
		case "tasks":
			summary.ModuleStatus["task"] = "error"
		}
	}
	return summary
}

func buildQuality(now time.Time, contract statusContract, summary overviewSummary, snapshots overviewSnapshots, sourceErrors map[string]string) overviewQuality {
	moduleTotal := len(summary.ModuleStatus)
	moduleError := 0
	moduleWarning := 0
	for _, status := range summary.ModuleStatus {
		switch status {
		case "error":
			moduleError++
		case "warning":
			moduleWarning++
		}
	}
	if moduleTotal == 0 {
		moduleTotal = 1
	}
	availability := clampInt(int(((float64(moduleTotal)-float64(moduleError)-float64(moduleWarning)*0.5)/float64(moduleTotal))*100), 0, 100)

	freshnessTotal := summary.HostTotal + summary.DockerTotal + summary.K8sTotal + summary.FirewallTotal + summary.DomainTotal
	freshnessIssues := summary.HostStale + summary.DockerStale + summary.K8sStale + summary.FirewallStale + summary.DomainStale
	freshness := 100
	if freshnessTotal > 0 {
		freshness = clampInt(int((1-float64(freshnessIssues)/float64(freshnessTotal))*100), 0, 100)
	}

	completeness, completenessProblems := calcCompletenessScore(snapshots)
	consistencyIssues := countConsistencyIssues(snapshots)
	consistency := clampInt(100-consistencyIssues*8, 0, 100)

	weighted := int(float64(availability)*0.32 + float64(freshness)*0.28 + float64(completeness)*0.22 + float64(consistency)*0.18)
	riskPenalty := minInt(25, summary.HostOffline*2+summary.K8sUnhealthy*3+summary.FirewallAlert*2+summary.DomainCritical*2+len(sourceErrors)*3)
	trustScore := clampInt(weighted-riskPenalty, 0, 100)

	trustGrade := "A"
	qualitySummary := "状态链路整体稳定，可重点关注增长性优化。"
	switch {
	case trustScore < 60:
		trustGrade = "D"
		qualitySummary = "状态可信度偏低，应优先恢复核心链路可用性并补齐状态字段。"
	case trustScore < 75:
		trustGrade = "C"
		qualitySummary = "状态可信度一般，建议先修复失败链路与跨模块冲突，再做体验优化。"
	case trustScore < 90:
		trustGrade = "B"
		qualitySummary = "核心链路可用，但存在局部时效或字段缺口，建议按建议清单优先处理。"
	}

	dimensions := []qualityDimension{
		{
			Key:    "availability",
			Label:  "链路可用性",
			Score:  availability,
			Detail: fmt.Sprintf("失败 %d 项，降级 %d 项", moduleError, moduleWarning),
		},
		{
			Key:    "freshness",
			Label:  "状态时效",
			Score:  freshness,
			Detail: fmt.Sprintf("过期状态 %d 个", freshnessIssues),
		},
		{
			Key:    "completeness",
			Label:  "字段完整性",
			Score:  completeness,
			Detail: fmt.Sprintf("缺口模块 %d", completenessProblems),
		},
		{
			Key:    "consistency",
			Label:  "跨模块一致性",
			Score:  consistency,
			Detail: fmt.Sprintf("冲突项 %d", consistencyIssues),
		},
	}

	actionHints := buildActionHints(now, contract, summary, snapshots, sourceErrors, completenessProblems, consistencyIssues)

	return overviewQuality{
		TrustScore:                 trustScore,
		TrustGrade:                 trustGrade,
		Summary:                    qualitySummary,
		Dimensions:                 dimensions,
		ActionHints:                actionHints,
		CompletenessProblemModules: completenessProblems,
		ConsistencyIssues:          consistencyIssues,
		SourceErrorCount:           len(sourceErrors),
	}
}

func buildActionHints(now time.Time, contract statusContract, summary overviewSummary, snapshots overviewSnapshots, sourceErrors map[string]string, completenessProblems, consistencyIssues int) []actionHint {
	items := make([]actionHint, 0, 10)
	add := func(item actionHint) {
		item.Priority = clampInt(item.Priority, 0, 100)
		item.PriorityLabel = priorityLabel(item.Priority)
		if strings.TrimSpace(item.Path) == "" {
			item.Path = "/dashboard"
		}
		if strings.TrimSpace(item.Action) == "" {
			item.Action = "进入处置"
		}
		items = append(items, item)
	}

	if len(sourceErrors) > 0 {
		add(actionHint{
			Key:      "pipeline-errors",
			Priority: 95,
			Module:   "全局链路",
			Title:    "先恢复失败数据链路",
			Reason:   fmt.Sprintf("当前有 %d 条链路失败，建议先恢复采集与鉴权。", len(sourceErrors)),
			Path:     "/dashboard",
			Action:   "查看自检",
		})
	}
	if summary.HostOffline > 0 || summary.HostStale > 0 {
		priority := 74
		if summary.HostOffline > 0 {
			priority = 90
		}
		add(actionHint{
			Key:      "cmdb-health",
			Priority: priority,
			Module:   "资产管理",
			Title:    "修复主机状态可信度",
			Reason:   fmt.Sprintf("主机离线 %d 台，状态过期 %d 台。", summary.HostOffline, summary.HostStale),
			Path:     "/host",
			Action:   "进入主机管理",
		})
	}
	if summary.DockerOffline > 0 || summary.DockerStale > 0 {
		priority := 72
		if summary.DockerOffline > 0 {
			priority = 86
		}
		add(actionHint{
			Key:      "docker-health",
			Priority: priority,
			Module:   "容器管理",
			Title:    "修复 Docker 环境状态",
			Reason:   fmt.Sprintf("Docker 离线 %d 台，状态过期 %d 台。", summary.DockerOffline, summary.DockerStale),
			Path:     "/docker",
			Action:   "进入 Docker 管理",
		})
	}
	if summary.K8sUnhealthy > 0 || summary.K8sStale > 0 {
		priority := 76
		if summary.K8sUnhealthy > 0 {
			priority = 88
		}
		add(actionHint{
			Key:      "k8s-health",
			Priority: priority,
			Module:   "K8s",
			Title:    "处理异常或过期集群",
			Reason:   fmt.Sprintf("异常集群 %d 个，状态过期 %d 个。", summary.K8sUnhealthy, summary.K8sStale),
			Path:     "/k8s/clusters",
			Action:   "进入集群列表",
		})
	}

	jumpPendingTimeout := 0
	for _, item := range snapshots.JumpSessions {
		if normalizeText(item.Status) == "pending_approval" && now.Sub(item.StartedAt) >= 30*time.Minute {
			jumpPendingTimeout++
		}
	}
	if summary.ModuleStatus["jump"] != "ok" || normalizeText(snapshots.JumpIntegration.LastSyncStatus) == "failed" {
		syncMsg := truncateText(snapshots.JumpIntegration.LastSyncMsg, 72)
		if syncMsg == "" {
			syncMsg = "Jump 同步链路异常"
		}
		add(actionHint{
			Key:      "jump-sync",
			Priority: 85,
			Module:   "堡垒机",
			Title:    "修复 Jump 资产同步",
			Reason:   fmt.Sprintf("最近同步状态异常：%s", syncMsg),
			Path:     "/jump/assets",
			Action:   "进入堡垒机资产",
		})
	}
	if consistencyIssues > 0 {
		add(actionHint{
			Key:      "cross-consistency",
			Priority: 80,
			Module:   "跨模块",
			Title:    "消除资产映射冲突",
			Reason:   fmt.Sprintf("发现 %d 条跨模块一致性冲突，可能导致状态误判。", consistencyIssues),
			Path:     "/dashboard",
			Action:   "进入完整性总览",
		})
	}
	if completenessProblems > 0 {
		add(actionHint{
			Key:      "field-completeness",
			Priority: 68,
			Module:   "数据质量",
			Title:    "补齐状态字段",
			Reason:   fmt.Sprintf("共有 %d 个模块存在字段缺口，建议补齐状态/时间戳/原因。", completenessProblems),
			Path:     "/dashboard",
			Action:   "进入字段完整性",
		})
	}

	alertTimeout := 0
	for _, item := range snapshots.Alerts {
		if (item.Status == 0 || item.Status == 1) && now.Sub(item.FiredAt) >= 60*time.Minute {
			alertTimeout++
		}
	}
	domainRiskStale := summary.DomainStale
	overdueTotal := jumpPendingTimeout + alertTimeout + domainRiskStale
	if overdueTotal > 0 {
		add(actionHint{
			Key:      "overdue-backlog",
			Priority: 82,
			Module:   "待处置",
			Title:    "优先清理超时积压",
			Reason:   fmt.Sprintf("当前超时积压 %d 项，优先处理可显著降低故障放大风险。", overdueTotal),
			Path:     "/dashboard",
			Action:   "进入积压总览",
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Priority > items[j].Priority
	})
	if len(items) > 10 {
		items = items[:10]
	}
	return items
}

func priorityLabel(priority int) string {
	switch {
	case priority >= 90:
		return "P0"
	case priority >= 70:
		return "P1"
	default:
		return "P2"
	}
}

func calcCompletenessScore(snapshots overviewSnapshots) (score int, problemModules int) {
	rates := []int{
		completenessRateHosts(snapshots.Hosts),
		completenessRateDockerHosts(snapshots.DockerHosts),
		completenessRateK8sClusters(snapshots.K8sClusters),
		completenessRateNetworkDevices(snapshots.NetworkDevices),
		completenessRateFirewalls(snapshots.Firewalls),
		completenessRateJumpAssets(snapshots.JumpAssets),
	}

	total := 0
	for _, rate := range rates {
		total += rate
		if rate < 95 {
			problemModules++
		}
	}
	if len(rates) == 0 {
		return 100, 0
	}
	score = clampInt(total/len(rates), 0, 100)
	return score, problemModules
}

func completenessRateHosts(rows []hostSnapshot) int {
	acc := newCompletenessAccumulator()
	for _, row := range rows {
		acc.add(true)
		acc.add(hasAnyString(row.ID, row.IP, row.Name))
		acc.add(validTimestampPtr(row.LastCheckAt))
		if row.Status != 1 {
			acc.add(strings.TrimSpace(row.StatusReason) != "")
		}
	}
	return acc.rate()
}

func completenessRateDockerHosts(rows []dockerHostSnapshot) int {
	acc := newCompletenessAccumulator()
	for _, row := range rows {
		acc.add(strings.TrimSpace(row.Status) != "")
		acc.add(hasAnyString(row.ID, row.HostID, row.Name))
		acc.add(validTimestampPtr(row.LastCheckAt))
		if normalizeText(row.Status) != "online" {
			acc.add(strings.TrimSpace(row.LastError) != "")
		}
	}
	return acc.rate()
}

func completenessRateK8sClusters(rows []k8sClusterSnapshot) int {
	acc := newCompletenessAccumulator()
	for _, row := range rows {
		acc.add(row.Status == 0 || row.Status == 1 || row.Status == 2)
		acc.add(hasAnyString(row.ID, row.Name, row.DisplayName))
		acc.add(validTimestampPtr(row.LastCheckAt))
	}
	return acc.rate()
}

func completenessRateNetworkDevices(rows []networkDeviceSnapshot) int {
	acc := newCompletenessAccumulator()
	for _, row := range rows {
		acc.add(row.Status == 0 || row.Status == 1 || row.Status == 2)
		acc.add(hasAnyString(row.ID, row.IP, row.Name))
		acc.add(validTimestampPtr(row.LastCheckAt))
		if row.Status != 1 {
			acc.add(strings.TrimSpace(row.StatusReason) != "")
		}
	}
	return acc.rate()
}

func completenessRateFirewalls(rows []firewallSnapshot) int {
	acc := newCompletenessAccumulator()
	for _, row := range rows {
		acc.add(row.Status == 0 || row.Status == 1 || row.Status == 2)
		acc.add(hasAnyString(row.ID, row.IP, row.Name))
		acc.add(validTimestampPtr(row.LastCheckAt))
		if row.Status != 1 {
			acc.add(strings.TrimSpace(row.StatusReason) != "")
		}
	}
	return acc.rate()
}

func completenessRateJumpAssets(rows []jumpAssetSnapshot) int {
	acc := newCompletenessAccumulator()
	for _, row := range rows {
		acc.add(hasAnyString(row.ID, row.Name))
		acc.add(!row.UpdatedAt.IsZero())
		if src := normalizeText(row.Source); src != "" && src != "manual" && src != "local" {
			acc.add(strings.TrimSpace(row.SourceRef) != "")
		}
	}
	return acc.rate()
}

func countConsistencyIssues(snapshots overviewSnapshots) int {
	issues := 0
	cmdbStatusByID := make(map[string]int, len(snapshots.Hosts))
	dockerHostIDSet := make(map[string]struct{}, len(snapshots.DockerHosts))
	k8sIDSet := make(map[string]struct{}, len(snapshots.K8sClusters))

	for _, row := range snapshots.Hosts {
		if strings.TrimSpace(row.ID) == "" {
			continue
		}
		cmdbStatusByID[row.ID] = row.Status
	}
	for _, row := range snapshots.DockerHosts {
		if strings.TrimSpace(row.HostID) != "" {
			dockerHostIDSet[row.HostID] = struct{}{}
		}
	}
	for _, row := range snapshots.K8sClusters {
		if strings.TrimSpace(row.ID) != "" {
			k8sIDSet[row.ID] = struct{}{}
		}
	}

	for _, row := range snapshots.DockerHosts {
		hostID := strings.TrimSpace(row.HostID)
		if hostID == "" {
			issues++
			continue
		}
		hostStatus, ok := cmdbStatusByID[hostID]
		if !ok {
			issues++
			continue
		}
		dockerOnline := normalizeText(row.Status) == "online"
		cmdbOnline := hostStatus == 1
		if dockerOnline != cmdbOnline {
			issues++
		}
	}

	for _, row := range snapshots.JumpAssets {
		source := normalizeText(row.Source)
		sourceRef := strings.TrimSpace(row.SourceRef)
		if source == "manual" || source == "local" || source == "" {
			continue
		}
		if sourceRef == "" {
			issues++
			continue
		}
		switch source {
		case "cmdb_host":
			if _, ok := cmdbStatusByID[sourceRef]; !ok {
				issues++
			}
		case "docker_host":
			if _, ok := dockerHostIDSet[sourceRef]; !ok {
				issues++
			}
		case "k8s_cluster":
			if _, ok := k8sIDSet[sourceRef]; !ok {
				issues++
			}
		}
	}
	return issues
}

type completenessAccumulator struct {
	expected int
	missing  int
}

func newCompletenessAccumulator() *completenessAccumulator {
	return &completenessAccumulator{}
}

func (a *completenessAccumulator) add(ok bool) {
	a.expected++
	if !ok {
		a.missing++
	}
}

func (a *completenessAccumulator) rate() int {
	if a.expected == 0 {
		return 100
	}
	return clampInt(int((float64(a.expected-a.missing)/float64(a.expected))*100), 0, 100)
}

func validTimestampPtr(v *time.Time) bool {
	return v != nil && !v.IsZero()
}

func hasAnyString(values ...string) bool {
	for _, item := range values {
		if strings.TrimSpace(item) != "" {
			return true
		}
	}
	return false
}

func (h *DashboardHandler) listHosts() ([]hostSnapshot, error) {
	if !h.db.Migrator().HasTable(&cmdb.Host{}) {
		return []hostSnapshot{}, nil
	}
	var rows []hostSnapshot
	err := h.db.Model(&cmdb.Host{}).
		Select("id, name, ip, status, last_check_at, last_online_at, status_reason, group_id, updated_at").
		Find(&rows).Error
	return rows, wrapErr("查询主机失败", err)
}

func (h *DashboardHandler) listDockerHosts() ([]dockerHostSnapshot, error) {
	if !h.db.Migrator().HasTable(&docker.DockerHost{}) {
		return []dockerHostSnapshot{}, nil
	}
	var rows []dockerHostSnapshot
	err := h.db.Model(&docker.DockerHost{}).
		Select("id, name, host_id, status, last_check_at, last_online_at, last_error, updated_at").
		Find(&rows).Error
	return rows, wrapErr("查询 Docker 主机失败", err)
}

func (h *DashboardHandler) listK8sClusters() ([]k8sClusterSnapshot, error) {
	if !h.db.Migrator().HasTable(&k8s.Cluster{}) {
		return []k8sClusterSnapshot{}, nil
	}
	var rows []k8sClusterSnapshot
	err := h.db.Model(&k8s.Cluster{}).
		Select("id, name, display_name, status, last_check_at, last_online_at, status_reason, updated_at").
		Find(&rows).Error
	return rows, wrapErr("查询 K8s 集群失败", err)
}

func (h *DashboardHandler) listAlerts(since time.Time) ([]alertSnapshot, error) {
	if !h.db.Migrator().HasTable(&alert.Alert{}) {
		return []alertSnapshot{}, nil
	}
	var rows []alertSnapshot
	err := h.db.Model(&alert.Alert{}).
		Select("id, rule_name, target, severity, status, fired_at, created_at").
		Where("(status IN ? OR fired_at >= ? OR created_at >= ?)", []int{0, 1}, since, since).
		Order("created_at DESC").
		Limit(1200).
		Find(&rows).Error
	return rows, wrapErr("查询告警失败", err)
}

func (h *DashboardHandler) listTasks() ([]taskSnapshot, error) {
	if !h.db.Migrator().HasTable(&task.Task{}) {
		return []taskSnapshot{}, nil
	}
	var rows []taskSnapshot
	err := h.db.Model(&task.Task{}).
		Select("id, name, enabled, updated_at").
		Order("updated_at DESC").
		Find(&rows).Error
	return rows, wrapErr("查询任务失败", err)
}

func (h *DashboardHandler) listAgents(now time.Time) ([]agentSnapshot, error) {
	if !h.db.Migrator().HasTable(&monitor.AgentHeartbeat{}) {
		return []agentSnapshot{}, nil
	}
	var raw []monitor.AgentHeartbeat
	if err := h.db.Order("last_seen DESC").Find(&raw).Error; err != nil {
		return nil, wrapErr("查询 Agent 失败", err)
	}
	rows := make([]agentSnapshot, 0, len(raw))
	for _, item := range raw {
		status := "online"
		if now.Sub(item.LastSeen) > h.agentTimeout {
			status = "offline"
		}
		rows = append(rows, agentSnapshot{
			ID:       item.ID,
			AgentID:  item.AgentID,
			Hostname: item.Hostname,
			IP:       item.IP,
			Status:   status,
			LastSeen: item.LastSeen,
			CPU:      item.CPU,
			Memory:   item.Memory,
			Disk:     item.Disk,
			NetIn:    item.NetIn,
			NetOut:   item.NetOut,
		})
	}
	return rows, nil
}

func (h *DashboardHandler) getRealtimeMetrics(agents []agentSnapshot) (metricSnapshot, error) {
	out := metricSnapshot{}
	if h.db.Migrator().HasTable(&monitor.MetricRecord{}) {
		var row monitor.MetricRecord
		if err := h.db.Order("timestamp DESC").First(&row).Error; err == nil {
			out.CPU = round2(row.CPUUsage)
			out.Memory = round2(row.MemoryUsage)
			out.Disk = round2(row.DiskUsage)
			out.Network = round2(float64(row.NetworkIn+row.NetworkOut) / (1024 * 1024))
			return out, nil
		} else if err != nil && err != gorm.ErrRecordNotFound {
			return out, wrapErr("查询实时指标失败", err)
		}
	}

	online := 0
	var cpu, memory, disk, network float64
	for _, item := range agents {
		if normalizeText(item.Status) != "online" {
			continue
		}
		online++
		cpu += item.CPU
		memory += item.Memory
		disk += item.Disk
		network += (item.NetIn + item.NetOut) / (1024 * 1024)
	}
	if online > 0 {
		out.CPU = round2(cpu / float64(online))
		out.Memory = round2(memory / float64(online))
		out.Disk = round2(disk / float64(online))
		out.Network = round2(network / float64(online))
	}
	return out, nil
}

func (h *DashboardHandler) listMetricHistory(since time.Time, hours int) ([]metricHistorySnapshot, error) {
	if !h.db.Migrator().HasTable(&monitor.MetricRecord{}) {
		return []metricHistorySnapshot{}, nil
	}
	var rows []metricHistorySnapshot
	limit := clampInt(hours*120, 120, 2400)
	err := h.db.Model(&monitor.MetricRecord{}).
		Select("timestamp, cpu_usage, memory_usage, disk_usage").
		Where("timestamp >= ?", since).
		Order("timestamp ASC").
		Limit(limit).
		Find(&rows).Error
	return rows, wrapErr("查询历史指标失败", err)
}

func (h *DashboardHandler) listNetworkDevices() ([]networkDeviceSnapshot, error) {
	if !h.db.Migrator().HasTable(&cmdb.NetworkDevice{}) {
		return []networkDeviceSnapshot{}, nil
	}
	var rows []networkDeviceSnapshot
	err := h.db.Model(&cmdb.NetworkDevice{}).
		Select("id, name, ip, device_type, vendor, status, last_check_at, status_reason, updated_at").
		Find(&rows).Error
	return rows, wrapErr("查询网络设备失败", err)
}

func (h *DashboardHandler) listFirewalls() ([]firewallSnapshot, error) {
	if !h.db.Migrator().HasTable(&firewall.Firewall{}) {
		return []firewallSnapshot{}, nil
	}
	var rows []firewallSnapshot
	err := h.db.Model(&firewall.Firewall{}).
		Select("id, name, ip, vendor, status, last_check_at, status_reason, updated_at").
		Find(&rows).Error
	return rows, wrapErr("查询防火墙失败", err)
}

func (h *DashboardHandler) listJumpSessions(since time.Time) ([]jumpSessionSnapshot, error) {
	if !h.db.Migrator().HasTable(&jump.JumpSession{}) {
		return []jumpSessionSnapshot{}, nil
	}
	var rows []jumpSessionSnapshot
	err := h.db.Model(&jump.JumpSession{}).
		Select("id, status, asset_name, username, started_at, created_at").
		Where("(status IN ? OR started_at >= ?)", []string{"pending_approval", "active", "blocked"}, since).
		Order("started_at DESC").
		Limit(1200).
		Find(&rows).Error
	return rows, wrapErr("查询堡垒机会话失败", err)
}

func (h *DashboardHandler) listJumpRiskEvents(since time.Time) ([]jumpRiskEventSnapshot, error) {
	if !h.db.Migrator().HasTable(&jump.JumpRiskEvent{}) {
		return []jumpRiskEventSnapshot{}, nil
	}
	var rows []jumpRiskEventSnapshot
	err := h.db.Model(&jump.JumpRiskEvent{}).
		Select("id, severity, event_type, asset_name, username, fired_at").
		Where("fired_at >= ?", since).
		Order("fired_at DESC").
		Limit(1200).
		Find(&rows).Error
	return rows, wrapErr("查询堡垒机风控事件失败", err)
}

func (h *DashboardHandler) listDomains() ([]domainSnapshot, error) {
	if !h.db.Migrator().HasTable(&domain.CloudDomain{}) {
		return []domainSnapshot{}, nil
	}
	var rows []domainSnapshot
	err := h.db.Model(&domain.CloudDomain{}).
		Select("id, domain, health_status, http_status_code, response_time_ms, last_check_at, updated_at").
		Find(&rows).Error
	return rows, wrapErr("查询域名失败", err)
}

func (h *DashboardHandler) listCerts() ([]certSnapshot, error) {
	if !h.db.Migrator().HasTable(&domain.SSLCertificate{}) {
		return []certSnapshot{}, nil
	}
	var rows []certSnapshot
	err := h.db.Model(&domain.SSLCertificate{}).
		Select("id, domain, status, days_to_expire, last_check_at, updated_at").
		Find(&rows).Error
	return rows, wrapErr("查询证书失败", err)
}

func (h *DashboardHandler) listCICDExecutions(since time.Time) ([]cicdExecutionSnapshot, error) {
	if !h.db.Migrator().HasTable(&cicd.CICDExecution{}) {
		return []cicdExecutionSnapshot{}, nil
	}
	var rows []cicdExecutionSnapshot
	err := h.db.Model(&cicd.CICDExecution{}).
		Select("id, status, started_at, finished_at").
		Where("(status = ? OR started_at >= ?)", 0, since).
		Order("started_at DESC").
		Limit(1200).
		Find(&rows).Error
	return rows, wrapErr("查询流水线执行失败", err)
}

func (h *DashboardHandler) listCICDSchedules() ([]cicdScheduleSnapshot, error) {
	if !h.db.Migrator().HasTable(&cicd.CICDSchedule{}) {
		return []cicdScheduleSnapshot{}, nil
	}
	var rows []cicdScheduleSnapshot
	err := h.db.Model(&cicd.CICDSchedule{}).
		Select("id, enabled, last_run_at, next_run_at").
		Order("updated_at DESC").
		Find(&rows).Error
	return rows, wrapErr("查询流水线计划失败", err)
}

func (h *DashboardHandler) listWorkorders(since time.Time) ([]workorderSnapshot, error) {
	if !h.db.Migrator().HasTable(&workorder.WorkOrder{}) {
		return []workorderSnapshot{}, nil
	}
	var rows []workorderSnapshot
	err := h.db.Model(&workorder.WorkOrder{}).
		Select("id, status, created_at").
		Where("(status IN ? OR created_at >= ?)", []int{0, 1, 4}, since).
		Order("created_at DESC").
		Limit(1200).
		Find(&rows).Error
	return rows, wrapErr("查询工单失败", err)
}

func (h *DashboardHandler) listWorkflowExecutions(since time.Time) ([]workflowExecutionSnapshot, error) {
	if !h.db.Migrator().HasTable(&workflow.WorkflowExecution{}) {
		return []workflowExecutionSnapshot{}, nil
	}
	var rows []workflowExecutionSnapshot
	err := h.db.Model(&workflow.WorkflowExecution{}).
		Select("id, status, started_at").
		Where("(status = ? OR started_at >= ?)", 0, since).
		Order("started_at DESC").
		Limit(1200).
		Find(&rows).Error
	return rows, wrapErr("查询工作流执行失败", err)
}

func (h *DashboardHandler) listTerminalSessions(since time.Time) ([]terminalSessionSnapshot, error) {
	if !h.db.Migrator().HasTable(&terminal.TerminalSession{}) {
		return []terminalSessionSnapshot{}, nil
	}
	var rows []terminalSessionSnapshot
	err := h.db.Model(&terminal.TerminalSession{}).
		Select("id, status, started_at, created_at").
		Where("(status IN ? OR created_at >= ?)", []int{0, 1, 3}, since).
		Order("created_at DESC").
		Limit(1200).
		Find(&rows).Error
	return rows, wrapErr("查询终端会话失败", err)
}

func (h *DashboardHandler) listJumpAssets() ([]jumpAssetSnapshot, error) {
	if !h.db.Migrator().HasTable(&jump.JumpAsset{}) {
		return []jumpAssetSnapshot{}, nil
	}
	var rows []jumpAssetSnapshot
	err := h.db.Model(&jump.JumpAsset{}).
		Select("id, name, source, source_ref, enabled, updated_at, created_at").
		Order("updated_at DESC").
		Find(&rows).Error
	return rows, wrapErr("查询堡垒机资产失败", err)
}

func (h *DashboardHandler) getJumpIntegration() (jumpIntegrationSnapshot, error) {
	out := jumpIntegrationSnapshot{}
	if !h.db.Migrator().HasTable(&jump.JumpIntegrationConfig{}) {
		return out, nil
	}
	var cfg jump.JumpIntegrationConfig
	err := h.db.Order("updated_at DESC").First(&cfg).Error
	if err == gorm.ErrRecordNotFound {
		return out, nil
	}
	if err != nil {
		return out, wrapErr("查询 Jump 集成配置失败", err)
	}
	out.Enabled = cfg.Enabled
	out.BaseURL = cfg.BaseURL
	out.VerifyTLS = cfg.VerifyTLS
	out.AutoSync = cfg.AutoSync
	out.LastSyncStatus = cfg.LastSyncStatus
	out.LastSyncMsg = cfg.LastSyncMsg
	out.LastSyncAt = cfg.LastSyncAt
	return out, nil
}

func wrapErr(message string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

func parseInt(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return fallback
	}
	return value
}

func clampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

func normalizeText(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func truncateText(v string, maxLen int) string {
	v = strings.TrimSpace(v)
	if maxLen <= 0 || len(v) <= maxLen {
		return v
	}
	if maxLen <= 1 {
		return v[:maxLen]
	}
	return v[:maxLen-1] + "…"
}

func staleWithFallback(now time.Time, ttl time.Duration, updatedAt time.Time, primary *time.Time) bool {
	ts := updatedAt
	if primary != nil && !primary.IsZero() {
		ts = *primary
	}
	if ts.IsZero() {
		return true
	}
	return now.Sub(ts) > ttl
}
