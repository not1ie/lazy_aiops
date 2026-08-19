package monitor

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type InspectionIssue struct {
	Level       string `json:"level"`       // critical, warning, info
	Category    string `json:"category"`    // host, disk, memory, cpu, ssl, k8s, alert
	Target      string `json:"target"`      // 目标对象名/IP
	Title       string `json:"title"`       // 简要标题
	Description string `json:"description"` // 详细描述
	Suggestion  string `json:"suggestion"`  // 优化/修复建议
}

type InspectionReport struct {
	Score           int               `json:"score"`            // 0 - 100 综合健康评分
	Grade           string            `json:"grade"`            // 优(A), 良(B), 中(C), 差(D)
	CheckTime       string            `json:"check_time"`       // 巡检时间
	TotalChecks     int               `json:"total_checks"`     // 检查项总数
	PassedChecks    int               `json:"passed_checks"`    // 通过项数
	WarningCount    int               `json:"warning_count"`    // 警告数
	CriticalCount   int               `json:"critical_count"`   // 严重隐患数
	HostStats       map[string]int    `json:"host_stats"`       // total, online, offline
	DiskRisks       []InspectionIssue `json:"disk_risks"`       // 磁盘告急列表
	MemRisks        []InspectionIssue `json:"mem_risks"`        // 内存高载列表
	CpuRisks        []InspectionIssue `json:"cpu_risks"`        // CPU高载列表
	OfflineHosts    []InspectionIssue `json:"offline_hosts"`    // 离线失联主机
	SslRisks        []InspectionIssue `json:"ssl_risks"`        // SSL证书隐患
	K8sRisks        []InspectionIssue `json:"k8s_risks"`        // 容器异常
	AlertRisks      []InspectionIssue `json:"alert_risks"`      // 未恢复严重告警
	AllIssues       []InspectionIssue `json:"all_issues"`       // 所有问题清单
	Recommendations []string          `json:"recommendations"`  // SRE 运维优化总体建议
}

// GetInspectionReport 执行全平台资产与运维健康巡检（加权评分模型）
func (h *MonitorHandler) GetInspectionReport(c *gin.Context) {
	report := InspectionReport{
		CheckTime:       time.Now().Format("2006-01-02 15:04:05"),
		HostStats:       make(map[string]int),
		DiskRisks:       make([]InspectionIssue, 0),
		MemRisks:        make([]InspectionIssue, 0),
		CpuRisks:        make([]InspectionIssue, 0),
		OfflineHosts:    make([]InspectionIssue, 0),
		SslRisks:        make([]InspectionIssue, 0),
		K8sRisks:        make([]InspectionIssue, 0),
		AlertRisks:      make([]InspectionIssue, 0),
		AllIssues:       make([]InspectionIssue, 0),
		Recommendations: make([]string, 0),
	}

	// 1. 巡检主机心跳与资源使用率
	var heartbeats []AgentHeartbeat
	_ = h.db.Find(&heartbeats).Error

	totalHosts := len(heartbeats)
	onlineCount := 0
	offlineCount := 0

	now := time.Now()
	for _, hb := range heartbeats {
		isOffline := hb.Status != "online" || now.Sub(hb.LastSeen) > 3*time.Minute
		if isOffline {
			offlineCount++
			issue := InspectionIssue{
				Level:       "warning",
				Category:    "host",
				Target:      hb.IP,
				Title:       fmt.Sprintf("主机失联: %s (%s)", hb.Hostname, hb.IP),
				Description: fmt.Sprintf("主机自 %s 后未上报心跳指标，疑似宕机或网络中断", hb.LastSeen.Format("2006-01-02 15:04:05")),
				Suggestion:  "请检查该主机的物理/云状态、网络连通性及 SSH/Agent 服务状态",
			}
			report.OfflineHosts = append(report.OfflineHosts, issue)
			report.AllIssues = append(report.AllIssues, issue)
		} else {
			onlineCount++

			// 检查磁盘
			if hb.Disk >= 92 {
				issue := InspectionIssue{
					Level:       "critical",
					Category:    "disk",
					Target:      hb.IP,
					Title:       fmt.Sprintf("磁盘极度严重告急 (%d%%): %s", int(hb.Disk), hb.Hostname),
					Description: fmt.Sprintf("主机 %s (%s) 根磁盘已用 %d%%，随时可能引发只读故障或进程崩溃", hb.Hostname, hb.IP, int(hb.Disk)),
					Suggestion:  "建议立即使用 LazyOps Web SFTP 文件管理器或故障自愈脚本清理 /var/log 和 /tmp 大文件",
				}
				report.DiskRisks = append(report.DiskRisks, issue)
				report.AllIssues = append(report.AllIssues, issue)
			} else if hb.Disk >= 85 {
				issue := InspectionIssue{
					Level:       "warning",
					Category:    "disk",
					Target:      hb.IP,
					Title:       fmt.Sprintf("磁盘空间预警 (%d%%): %s", int(hb.Disk), hb.Hostname),
					Description: fmt.Sprintf("主机 %s (%s) 根磁盘已用 %d%%，建议排查", hb.Hostname, hb.IP, int(hb.Disk)),
					Suggestion:  "排查历史日志归档与 Docker 缓存镜像 (`docker system prune`)",
				}
				report.DiskRisks = append(report.DiskRisks, issue)
				report.AllIssues = append(report.AllIssues, issue)
			}

			// 检查内存
			if hb.Memory >= 90 {
				issue := InspectionIssue{
					Level:       "critical",
					Category:    "memory",
					Target:      hb.IP,
					Title:       fmt.Sprintf("内存高负荷 (%d%%): %s", int(hb.Memory), hb.Hostname),
					Description: fmt.Sprintf("主机 %s (%s) 内存使用率高达 %d%%，存在 OOM 杀进程风险", hb.Hostname, hb.IP, int(hb.Memory)),
					Suggestion:  "检查异常内存泄漏进程，或配置自动释放缓存与扩容内存",
				}
				report.MemRisks = append(report.MemRisks, issue)
				report.AllIssues = append(report.AllIssues, issue)
			} else if hb.Memory >= 80 {
				issue := InspectionIssue{
					Level:       "warning",
					Category:    "memory",
					Target:      hb.IP,
					Title:       fmt.Sprintf("内存偏高 (%d%%): %s", int(hb.Memory), hb.Hostname),
					Description: fmt.Sprintf("主机 %s (%s) 内存使用率为 %d%%", hb.Hostname, hb.IP, int(hb.Memory)),
					Suggestion:  "监控内存增长趋势，确认是否为正常业务峰值",
				}
				report.MemRisks = append(report.MemRisks, issue)
				report.AllIssues = append(report.AllIssues, issue)
			}

			// 检查 CPU
			if hb.CPU >= 85 {
				issue := InspectionIssue{
					Level:       "warning",
					Category:    "cpu",
					Target:      hb.IP,
					Title:       fmt.Sprintf("CPU 高负荷 (%d%%): %s", int(hb.CPU), hb.Hostname),
					Description: fmt.Sprintf("主机 %s (%s) CPU 使用率高达 %d%%", hb.Hostname, hb.IP, int(hb.CPU)),
					Suggestion:  "通过终端快捷指令 `top` 查看消耗 CPU 最多的前 5 个进程",
				}
				report.CpuRisks = append(report.CpuRisks, issue)
				report.AllIssues = append(report.AllIssues, issue)
			}
		}
	}

	report.HostStats["total"] = totalHosts
	report.HostStats["online"] = onlineCount
	report.HostStats["offline"] = offlineCount

	// 2. 检查未恢复的严重告警
	type SimpleAlert struct {
		ID       string
		RuleName string
		Target   string
		Metric   string
		Severity string
		Status   int
	}
	var unrecoveredAlerts []SimpleAlert
	_ = h.db.Table("alerts").Where("status = ?", 0).Find(&unrecoveredAlerts).Error

	for _, alt := range unrecoveredAlerts {
		if alt.Severity == "critical" {
			issue := InspectionIssue{
				Level:       "critical",
				Category:    "alert",
				Target:      alt.Target,
				Title:       fmt.Sprintf("未恢复高危告警: %s", alt.RuleName),
				Description: fmt.Sprintf("目标 %s 存在持续触发中的严重告警 (指标: %s)", alt.Target, alt.Metric),
				Suggestion:  "建议在告警事件中立即使用「自愈」或「转工单」处理",
			}
			report.AlertRisks = append(report.AlertRisks, issue)
			report.AllIssues = append(report.AllIssues, issue)
		}
	}

	// 3. 加权综合评分模型
	// A. 主机可用度得分 (满分 40分)
	hostScore := 40.0
	if totalHosts > 0 {
		hostScore = (float64(onlineCount) / float64(totalHosts)) * 40.0
	}

	// B. 磁盘健康度得分 (满分 25分)
	diskScore := 25.0 - float64(len(report.DiskRisks))*3.0
	if diskScore < 0 {
		diskScore = 0
	}

	// C. 内存与 CPU 健康度得分 (满分 20分)
	resourceScore := 20.0 - float64(len(report.MemRisks))*2.0 - float64(len(report.CpuRisks))*1.0
	if resourceScore < 0 {
		resourceScore = 0
	}

	// D. 告警治理得分 (满分 15分)
	alertScore := 15.0 - float64(len(report.AlertRisks))*3.0
	if alertScore < 0 {
		alertScore = 0
	}

	score := int(hostScore + diskScore + resourceScore + alertScore)
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}
	report.Score = score

	if score >= 90 {
		report.Grade = "A (优秀)"
		report.Recommendations = append(report.Recommendations, "🎉 系统整体运行状况优良，核心指标均在健康区间。")
	} else if score >= 75 {
		report.Grade = "B (良好)"
		report.Recommendations = append(report.Recommendations, "⚠️ 发现个别失联主机或轻度资源压力，建议定期执行磁盘与内存清理。")
	} else if score >= 60 {
		report.Grade = "C (及格)"
		report.Recommendations = append(report.Recommendations, "🚨 存在一定数量的高危告警与资源瓶颈，建议启用自动自愈规则或转工单跟进。")
	} else {
		report.Grade = "D (高危)"
		report.Recommendations = append(report.Recommendations, "❌ 多台主机失联或处于极高负载，需运维人员立即排查处置！")
	}

	report.TotalChecks = totalHosts*4 + len(unrecoveredAlerts) + 5
	report.CriticalCount = 0
	report.WarningCount = 0
	for _, iss := range report.AllIssues {
		if iss.Level == "critical" {
			report.CriticalCount++
		} else if iss.Level == "warning" {
			report.WarningCount++
		}
	}
	report.PassedChecks = report.TotalChecks - len(report.AllIssues)
	if report.PassedChecks < 0 {
		report.PassedChecks = 0
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": report,
	})
}
