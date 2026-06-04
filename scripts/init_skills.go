//go:build ignore

package main

import (
	"fmt"
	"log"

	"github.com/lazyautoops/lazy-auto-ops/plugins/ai"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	db, err := gorm.Open(sqlite.Open("/Users/hayakawaaki/workspace/lazy_aiops/data/lazy-auto-ops.db"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	skills := []ai.AISkill{
		{
			BaseModel:        ai.BaseModel{},
			Name:             "K8s 集群诊断专家 (k8s_doctor)",
			Description:      "K8s 集群健康诊断专家。自动排查 NotReady 节点、Pending Pods 以及异常事件。",
			SystemPrompt:     "你是一个资深的 Kubernetes 运维专家。你的任务是分析当前集群的健康状况。你需要调用提供的工具，获取节点、Pod 的运行状态和集群事件。用专业的语言指出问题所在，并提供可行的修复建议。",
			ToolBindings:     `["get_k8s_pods", "get_k8s_events", "search_hosts"]`,
			ParametersSchema: `{"cluster_id": "string (可选, 默认自动获取)"}`,
			IsSystem:         true,
		},
		{
			BaseModel:        ai.BaseModel{},
			Name:             "主机深度巡检 (host_inspector)",
			Description:      "服务器深度巡检助手。结合资产和告警分析主机健康度。",
			SystemPrompt:     "你是一个 Linux 系统专家。用户要求你对某台或某些主机进行深度巡检。调用主机查询工具获取其基本信息，结合相关的监控告警数据。请输出一份包含资源健康度以及可能隐患的巡检报告，重点用粗体标识异常项。",
			ToolBindings:     `["search_hosts", "get_open_alerts"]`,
			ParametersSchema: `{"ip_or_hostname": "string (必填, 主机IP或名称)"}`,
			IsSystem:         true,
		},
		{
			BaseModel:        ai.BaseModel{},
			Name:             "全网告警根因分发引擎 (alert_triage)",
			Description:      "监控告警分析与分发引擎。聚合当前全局活动告警，提炼链路根因。",
			SystemPrompt:     "你是一个 SRE 值班工程师，正面临大量的监控告警。调用工具获取当前的全局活动告警列表。请将它们按业务或底层资源分类，合并同类项，指出可能的连锁故障根因，并给出紧急止血方案的优先级建议。",
			ToolBindings:     `["get_open_alerts"]`,
			ParametersSchema: `{"severity": "string (可选, 如 critical)"}`,
			IsSystem:         true,
		},
	}

	for _, s := range skills {
		var count int64
		db.Model(&ai.AISkill{}).Where("name = ?", s.Name).Count(&count)
		if count == 0 {
			db.Create(&s)
		}
	}
	fmt.Println("Preset AI skills inserted.")
}
