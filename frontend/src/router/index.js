import { createRouter, createWebHistory } from 'vue-router'
import Layout from '@/layout/index.vue'

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/login/index.vue')
  },
  {
    path: '/403',
    name: 'Forbidden',
    component: () => import('@/views/error/403.vue')
  },
  {
    path: '/',
    component: Layout,
    redirect: '/dashboard',
    children: [
      // ============================================================
      // 主中心（侧边栏可见）
      // ============================================================
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('@/views/dashboard/index.vue'),
        meta: { title: '仪表盘', icon: 'Odometer', perm: 'dashboard' }
      },
      {
        path: 'ai',
        name: 'AI',
        component: () => import('@/views/ai/index.vue'),
        meta: { title: 'AI运维助手', icon: 'MagicStick', perm: 'ai' }
      },
      {
        path: 'asset',
        name: 'AssetCenter',
        component: () => import('@/views/hub/asset.vue'),
        meta: { title: '资产与安全中心', icon: 'Monitor', perm: 'cmdb' }
      },
      {
        path: 'k8s',
        name: 'ContainerPlatform',
        component: () => import('@/views/hub/k8s.vue'),
        meta: { title: '容器编排平台', icon: 'Platform', perm: 'k8s' }
      },
      {
        path: 'monitor',
        name: 'ObservabilityHub',
        component: () => import('@/views/hub/monitor.vue'),
        meta: { title: '统一观测中心', icon: 'Histogram', perm: 'monitor' }
      },
      {
        path: 'delivery',
        name: 'DeliveryLifecycle',
        component: () => import('@/views/hub/delivery.vue'),
        meta: { title: '交付与生命周期', icon: 'Connection', perm: 'cicd' }
      },
      {
        path: 'system',
        name: 'SystemGovernance',
        component: () => import('@/views/hub/system.vue'),
        meta: { title: '系统治理中心', icon: 'Setting', perm: 'system' }
      },

      // ============================================================
      // 功能详情页（hidden，通过 hub 或直接链接访问）
      // ============================================================

      // --- 资产管理 ---
      { path: 'host', redirect: '/asset' },
      { path: 'cmdb/group', redirect: '/asset' },
      { path: 'cmdb/credential', name: 'CMDBCredential', component: () => import('@/views/cmdb/credential.vue'), meta: { title: '凭据管理', hidden: true, perm: 'cmdb' } },
      { path: 'cmdb/database', redirect: '/asset' },
      { path: 'cmdb/cloud', redirect: '/asset' },
      { path: 'cmdb/network-devices', redirect: '/asset' },
      { path: 'firewall', redirect: '/asset' },

      // --- AI 技能 & AIOps ---
      { path: 'ai-skills', name: 'AISkills', component: () => import('@/views/hub/ai-skills.vue'), meta: { title: 'AI技能管理', hidden: true, perm: 'ai' } },
      { path: 'ai/ops', name: 'AIOps', component: () => import('@/views/ai/ops.vue'), meta: { title: 'AIOps 故障诊断', hidden: true, perm: 'ai' } },
      { path: 'registries', name: 'Registries', component: () => import('@/views/hub/registries.vue'), meta: { title: '镜像仓库', hidden: true, perm: 'cicd' } },

      // --- 容器/K8s ---
      { path: 'docker', name: 'Docker', component: () => import('@/views/docker/index.vue'), meta: { title: 'Docker管理', hidden: true, perm: 'docker' } },
      { path: 'k8s/clusters', name: 'K8sClusters', component: () => import('@/views/k8s/clusters.vue'), meta: { title: 'K8s集群', hidden: true, perm: 'k8s' } },
      { path: 'k8s/namespaces', name: 'K8sNamespaces', component: () => import('@/views/k8s/namespaces.vue'), meta: { title: '命名空间', hidden: true, perm: 'k8s' } },
      { path: 'k8s/workloads', name: 'K8sWorkloads', component: () => import('@/views/k8s/workloads.vue'), meta: { title: '工作负载', hidden: true, perm: 'k8s' } },
      { path: 'k8s/deployments', name: 'K8sDeployments', component: () => import('@/views/k8s/deployments.vue'), meta: { title: 'Deployments', hidden: true, perm: 'k8s' } },
      { path: 'k8s/workloads/detail', name: 'K8sWorkloadDetail', component: () => import('@/views/k8s/workload-detail.vue'), meta: { title: '工作负载详情', hidden: true, perm: 'k8s' } },
      { path: 'k8s/pods', name: 'K8sPods', component: () => import('@/views/k8s/pods.vue'), meta: { title: 'Pods', hidden: true, perm: 'k8s' } },
      { path: 'k8s/pods/detail', name: 'K8sPodDetail', component: () => import('@/views/k8s/pod-detail.vue'), meta: { title: 'Pod详情', hidden: true, perm: 'k8s' } },
      { path: 'k8s/services', name: 'K8sServices', component: () => import('@/views/k8s/services.vue'), meta: { title: '服务与Ingress', hidden: true, perm: 'k8s' } },
      { path: 'k8s/configs', name: 'K8sConfigs', component: () => import('@/views/k8s/configs.vue'), meta: { title: 'Config/Secret', hidden: true, perm: 'k8s' } },
      { path: 'k8s/storage', name: 'K8sStorage', component: () => import('@/views/k8s/storage.vue'), meta: { title: '存储管理', hidden: true, perm: 'k8s' } },
      { path: 'k8s/nodes', name: 'K8sNodes', component: () => import('@/views/k8s/nodes.vue'), meta: { title: '节点管理', hidden: true, perm: 'k8s' } },
      { path: 'k8s/events', name: 'K8sEvents', component: () => import('@/views/k8s/events.vue'), meta: { title: '事件与诊断', hidden: true, perm: 'k8s' } },
      { path: 'k8s/terminal', name: 'K8sTerminal', component: () => import('@/views/k8s/terminal.vue'), meta: { title: 'K8s WebShell', hidden: true, perm: 'k8s' } },

      // --- 监控 & 告警 ---
      { path: 'monitor/hosts', name: 'MonitorHosts', component: () => import('@/views/monitor/hosts.vue'), meta: { title: '主机监控', hidden: true, perm: 'monitor' } },
      { path: 'monitor/overview', name: 'MonitorOverview', component: () => import('@/views/monitor/overview.vue'), meta: { title: '监控概览', hidden: true, perm: 'monitor' } },
      { path: 'monitor/metrics', name: 'MonitorMetrics', component: () => import('@/views/monitor/metrics.vue'), meta: { title: '指标采集', hidden: true, perm: 'monitor' } },
      { path: 'monitor/containers', name: 'MonitorContainers', component: () => import('@/views/monitor/containers.vue'), meta: { title: '容器监控', hidden: true, perm: 'monitor' } },
      { path: 'monitor/pods', name: 'MonitorPods', component: () => import('@/views/monitor/pods.vue'), meta: { title: 'Pod监控', hidden: true, perm: 'monitor' } },
      { path: 'monitor/agents', name: 'MonitorAgents', component: () => import('@/views/monitor/agents.vue'), meta: { title: 'Agent心跳', hidden: true, perm: 'monitor' } },
      { path: 'monitor/agents/detail', name: 'MonitorAgentDetail', component: () => import('@/views/monitor/agent-detail.vue'), meta: { title: 'Agent详情', hidden: true, perm: 'monitor' } },
      { path: 'alert/rules', name: 'AlertRules', component: () => import('@/views/alert/rules.vue'), meta: { title: '告警规则', hidden: true, perm: 'alert' } },
      { path: 'alert/events', name: 'AlertEvents', component: () => import('@/views/alert/events.vue'), meta: { title: '告警事件', hidden: true, perm: 'alert' } },
      { path: 'alert/events/detail', name: 'AlertDetail', component: () => import('@/views/alert/detail.vue'), meta: { title: '告警详情', hidden: true, perm: 'alert' } },
      { path: 'alert/silences', name: 'AlertSilences', component: () => import('@/views/alert/silences.vue'), meta: { title: '告警静默', hidden: true, perm: 'alert' } },
      { path: 'alert/aggregation', name: 'AlertAggregation', component: () => import('@/views/alert/aggregation.vue'), meta: { title: '告警聚合', hidden: true, perm: 'alert' } },
      { path: 'alert/history', name: 'AlertHistory', component: () => import('@/views/alert/history.vue'), meta: { title: '告警复盘', hidden: true, perm: 'alert' } },
      { path: 'alert/history/detail', name: 'AlertHistoryDetail', component: () => import('@/views/alert/history-detail.vue'), meta: { title: '复盘详情', hidden: true, perm: 'alert' } },
      { path: 'notify/channels', name: 'NotifyChannels', component: () => import('@/views/notify/channels.vue'), meta: { title: '通知渠道', hidden: true, perm: 'notify' } },
      { path: 'notify/groups', name: 'NotifyGroups', component: () => import('@/views/notify/groups.vue'), meta: { title: '通知组', hidden: true, perm: 'notify' } },
      { path: 'notify/templates', name: 'NotifyTemplates', component: () => import('@/views/notify/templates.vue'), meta: { title: '通知模板', hidden: true, perm: 'notify' } },
      { path: 'domain/ssl', name: 'DomainSSL', component: () => import('@/views/domain/ssl.vue'), meta: { title: '域名与证书', hidden: true, perm: 'domain' } },

      // --- 运维自动化 ---
      { path: 'workflow/orchestrator', name: 'WorkflowOrchestrator', component: () => import('@/views/workflow/orchestrator.vue'), meta: { title: '编排中心', hidden: true, perm: 'workflow' } },
      { path: 'workflow/designer', name: 'WorkflowDesigner', component: () => import('@/views/workflow/designer.vue'), meta: { title: '工作流编排', hidden: true, perm: 'workflow' } },
      { path: 'executor', name: 'Executor', component: () => import('@/views/executor/index.vue'), meta: { title: '批量执行', hidden: true, perm: 'executor' } },
      { path: 'task/schedules', name: 'TaskSchedules', component: () => import('@/views/task/schedules.vue'), meta: { title: '任务调度', hidden: true, perm: 'task' } },
      { path: 'ansible/playbooks', name: 'AnsiblePlaybooks', component: () => import('@/views/ansible/playbooks.vue'), meta: { title: 'Playbook管理', hidden: true, perm: 'ansible' } },
      { path: 'ansible/inventories', name: 'AnsibleInventories', component: () => import('@/views/ansible/inventories.vue'), meta: { title: 'Inventory管理', hidden: true, perm: 'ansible' } },

      // --- CI/CD & 交付 ---
      { path: 'cicd/pipelines', name: 'CICDPipelines', component: () => import('@/views/cicd/pipelines.vue'), meta: { title: '流水线管理', hidden: true, perm: 'cicd' } },
      { path: 'cicd/executions', name: 'CICDExecutions', component: () => import('@/views/cicd/executions.vue'), meta: { title: '执行记录', hidden: true, perm: 'cicd' } },
      { path: 'cicd/schedules', name: 'CICDSchedules', component: () => import('@/views/cicd/schedules.vue'), meta: { title: '定时发布', hidden: true, perm: 'cicd' } },
      { path: 'cicd/releases', name: 'CICDReleases', component: () => import('@/views/cicd/releases.vue'), meta: { title: '发布管理', hidden: true, perm: 'cicd' } },

      // --- 配置中心 ---
      { path: 'nacos/servers', name: 'NacosServers', component: () => import('@/views/nacos/servers.vue'), meta: { title: 'Nacos服务器', hidden: true, perm: 'nacos' } },
      { path: 'nacos/configs', name: 'NacosConfigs', component: () => import('@/views/nacos/configs.vue'), meta: { title: '配置管理', hidden: true, perm: 'nacos' } },

      // --- 变更管理 ---
      { path: 'workorder/tickets', name: 'WorkorderTickets', component: () => import('@/views/workorder/tickets.vue'), meta: { title: '工单管理', hidden: true, perm: 'workorder' } },
      { path: 'workorder/types', name: 'WorkorderTypes', component: () => import('@/views/workorder/types.vue'), meta: { title: '工单类型', hidden: true, perm: 'workorder' } },
      { path: 'sqlaudit/requests', name: 'SQLAuditRequests', component: () => import('@/views/sqlaudit/requests.vue'), meta: { title: 'SQL工单', hidden: true, perm: 'sqlaudit' } },
      { path: 'sqlaudit/rules', name: 'SQLAuditRules', component: () => import('@/views/sqlaudit/rules.vue'), meta: { title: 'SQL审核规则', hidden: true, perm: 'sqlaudit' } },
      { path: 'gitops/repos', name: 'GitOpsRepos', component: () => import('@/views/gitops/repos.vue'), meta: { title: 'GitOps仓库', hidden: true, perm: 'gitops' } },
      { path: 'gitops/sync', name: 'GitOpsSync', component: () => import('@/views/gitops/sync.vue'), meta: { title: '同步记录', hidden: true, perm: 'gitops' } },

      // --- 值班管理 ---
      { path: 'oncall/schedule', name: 'OncallSchedule', component: () => import('@/views/oncall/schedule.vue'), meta: { title: '值班排班', hidden: true, perm: 'oncall' } },
      { path: 'oncall/escalation', name: 'OncallEscalation', component: () => import('@/views/oncall/escalation.vue'), meta: { title: '升级策略', hidden: true, perm: 'oncall' } },

      // --- 堡垒机 ---
      { path: 'jump/assets', name: 'JumpAssets', component: () => import('@/views/jump/assets.vue'), meta: { title: '堡垒机资产', hidden: true, perm: 'jump:asset' } },
      { path: 'jump/policies', name: 'JumpPolicies', component: () => import('@/views/jump/policies.vue'), meta: { title: '授权策略', hidden: true, perm: 'jump:policy' } },
      { path: 'jump/command-rules', name: 'JumpCommandRules', component: () => import('@/views/jump/command-rules.vue'), meta: { title: '命令风控', hidden: true, perm: 'jump:rule' } },
      { path: 'jump/sessions', name: 'JumpSessions', component: () => import('@/views/jump/sessions.vue'), meta: { title: '会话审计', hidden: true, perm: 'jump:session' } },
      { path: 'terminal', name: 'Terminal', component: () => import('@/views/terminal/index.vue'), meta: { title: 'WebTerminal', hidden: true, perm: 'terminal' } },

      // --- 知识库 & 自愈 ---
      { path: 'knowledge', name: 'Knowledge', component: () => import('@/views/knowledge/index.vue'), meta: { title: '运维知识库', hidden: true, perm: 'knowledge' } },
      { path: 'remediation', name: 'Remediation', component: () => import('@/views/remediation/index.vue'), meta: { title: '故障自愈日志', hidden: true, perm: 'remediation' } },

      // --- 可视化 & 成本 ---
      { path: 'topology', name: 'Topology', component: () => import('@/views/topology/graph.vue'), meta: { title: '服务拓扑', hidden: true, perm: 'topology' } },
      { path: 'cost/overview', name: 'CostOverview', component: () => import('@/views/cost/overview.vue'), meta: { title: '成本概览', hidden: true, perm: 'cost' } },
      { path: 'cost/budget', name: 'CostBudget', component: () => import('@/views/cost/budget.vue'), meta: { title: '预算与告警', hidden: true, perm: 'cost' } },

      // --- 应用中心 ---
      { path: 'application', name: 'Application', component: () => import('@/views/application/index.vue'), meta: { title: '应用中心', hidden: true, perm: 'application' } },

      // --- 系统管理 ---
      { path: 'system/users', name: 'SystemUsers', component: () => import('@/views/system/users.vue'), meta: { title: '用户管理', hidden: true, perm: 'system:user' } },
      { path: 'system/roles', name: 'SystemRoles', component: () => import('@/views/system/roles.vue'), meta: { title: '角色管理', hidden: true, perm: 'system:role' } },
      { path: 'system/menus', name: 'SystemMenus', component: () => import('@/views/system/menus.vue'), meta: { title: '权限管理', hidden: true, perm: 'system:permission' } },
      { path: 'system/dept', name: 'Department', component: () => import('@/views/system/dept.vue'), meta: { title: '部门管理', hidden: true, perm: 'system:dept' } },
      { path: 'system/posts', name: 'SystemPosts', component: () => import('@/views/system/posts.vue'), meta: { title: '岗位管理', hidden: true, perm: 'system:post' } },
      { path: 'system/login-logs', name: 'SystemLoginLogs', component: () => import('@/views/system/login-logs.vue'), meta: { title: '登录日志', hidden: true, perm: 'system:loginlog' } },
      { path: 'system/audit-logs', name: 'SystemAuditLogs', component: () => import('@/views/system/audit-logs.vue'), meta: { title: '操作日志', hidden: true, perm: 'system:log' } },
      { path: 'system/captcha', name: 'SystemCaptcha', component: () => import('@/views/system/captcha.vue'), meta: { title: '验证码配置', hidden: true, perm: 'system:captcha' } }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

const hasPerm = (code) => {
  if (!code) return true
  const roleCode = localStorage.getItem('role_code')
  if (roleCode === 'admin') return true
  const perms = JSON.parse(localStorage.getItem('permissions') || '[]')
  if (perms.includes(code)) return true
  const parts = code.split(':')
  while (parts.length > 1) {
    parts.pop()
    if (perms.includes(parts.join(':'))) return true
  }
  return false
}

router.beforeEach((to, from, next) => {
  const token = localStorage.getItem('token')
  if (to.name !== 'Login' && !token) {
    next({ name: 'Login' })
    return
  }
  if (to.name !== 'Login' && to.name !== 'Forbidden' && to.meta?.perm && !hasPerm(to.meta.perm)) {
    next({ name: 'Forbidden' })
    return
  }
  next()
})

export default router
