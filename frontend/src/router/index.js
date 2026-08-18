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
      { path: 'dashboard', name: 'Dashboard', component: () => import('@/views/dashboard/index.vue'), meta: { title: '仪表盘', perm: 'dashboard' } },
      
      // 资产管理
      { path: 'host', name: 'CMDBHost', component: () => import('@/views/cmdb/host-main.vue'), meta: { title: '主机管理', perm: 'cmdb' } },
      { path: 'cmdb/credential', name: 'CMDBCredential', component: () => import('@/views/cmdb/credential.vue'), meta: { title: '凭据管理', perm: 'cmdb' } },
      { path: 'cmdb/database', name: 'CMDBDatabase', component: () => import('@/views/cmdb/database.vue'), meta: { title: '数据库资产', perm: 'cmdb' } },
      { path: 'cmdb/cloud', name: 'CMDBCloud', component: () => import('@/views/cmdb/cloud.vue'), meta: { title: '云资源', perm: 'cmdb' } },
      { path: 'cmdb/network-devices', name: 'CMDBNetwork', component: () => import('@/views/cmdb/network-device.vue'), meta: { title: '网络与防火墙', perm: 'cmdb' } },
      { path: 'jump/assets', name: 'JumpAssets', component: () => import('@/views/jump/assets.vue'), meta: { title: '堡垒机接入', perm: 'jump:asset' } },
      
      // Kubernetes
      { path: 'k8s/clusters', name: 'K8sClusters', component: () => import('@/views/k8s/clusters.vue'), meta: { title: 'K8s集群', perm: 'k8s' } },
      { path: 'k8s/namespaces', name: 'K8sNamespaces', component: () => import('@/views/k8s/namespaces.vue'), meta: { title: '命名空间', perm: 'k8s' } },
      { path: 'k8s/workloads', name: 'K8sWorkloads', component: () => import('@/views/k8s/workloads.vue'), meta: { title: '工作负载', perm: 'k8s' } },
      { path: 'k8s/pods', name: 'K8sPods', component: () => import('@/views/k8s/pods.vue'), meta: { title: 'Pods', perm: 'k8s' } },
      { path: 'k8s/services', name: 'K8sServices', component: () => import('@/views/k8s/services.vue'), meta: { title: '服务发现', perm: 'k8s' } },
      { path: 'k8s/configs', name: 'K8sConfigs', component: () => import('@/views/k8s/configs.vue'), meta: { title: 'Config/Secret', perm: 'k8s' } },
      { path: 'k8s/storage', name: 'K8sStorage', component: () => import('@/views/k8s/storage.vue'), meta: { title: '存储管理', perm: 'k8s' } },
      { path: 'k8s/nodes', name: 'K8sNodes', component: () => import('@/views/k8s/nodes.vue'), meta: { title: '节点管理', perm: 'k8s' } },
      { path: 'k8s/events', name: 'K8sEvents', component: () => import('@/views/k8s/events.vue'), meta: { title: '事件与诊断', perm: 'k8s' } },
      { path: 'k8s/terminal', name: 'K8sTerminal', component: () => import('@/views/k8s/terminal.vue'), meta: { title: 'WebShell', perm: 'k8s' } },
      
      // Docker & Portainer
      { path: 'docker', name: 'Docker', component: () => import('@/views/docker/index.vue'), meta: { title: '容器控制台', perm: 'docker' } },
      
      // 应用发布
      { path: 'cicd/pipelines', name: 'CICDPipelines', component: () => import('@/views/cicd/pipelines.vue'), meta: { title: '流水线', perm: 'cicd' } },
      { path: 'cicd/releases', name: 'CICDReleases', component: () => import('@/views/cicd/releases.vue'), meta: { title: '发布管理', perm: 'cicd' } },
      
      // 监控与告警
      { path: 'monitor/hosts', name: 'MonitorHosts', component: () => import('@/views/monitor/hosts.vue'), meta: { title: '主机监控', perm: 'monitor' } },
      { path: 'monitor/metrics', name: 'MonitorMetrics', component: () => import('@/views/monitor/metrics.vue'), meta: { title: '数据源配置', perm: 'monitor' } },
      { path: 'alert/events', name: 'AlertEvents', component: () => import('@/views/alert/events.vue'), meta: { title: '告警事件', perm: 'alert' } },
      { path: 'alert/history', name: 'AlertHistory', component: () => import('@/views/alert/history.vue'), meta: { title: '告警复盘', perm: 'alert' } },
      { path: 'alert/rules', name: 'AlertRules', component: () => import('@/views/alert/rules.vue'), meta: { title: '告警规则', perm: 'alert' } },
      { path: 'alert/silences', name: 'AlertSilences', component: () => import('@/views/alert/silences.vue'), meta: { title: '告警静默', perm: 'alert' } },
      { path: 'alert/aggregation', name: 'AlertAggregation', component: () => import('@/views/alert/aggregation.vue'), meta: { title: '告警聚合', perm: 'alert' } },
      
      // 日志中心
      { path: 'log/query', name: 'LogQuery', component: () => import('@/views/log/query.vue'), meta: { title: '日志查询', perm: 'log' } },
      { path: 'log/library', name: 'LogLibrary', component: () => import('@/views/log/library.vue'), meta: { title: '日志库', perm: 'log' } },
      { path: 'log/alerts', name: 'LogAlerts', component: () => import('@/views/log/alert.vue'), meta: { title: '日志告警', perm: 'log' } },
      { path: 'log/permissions', name: 'LogPermissions', component: () => import('@/views/log/permission.vue'), meta: { title: '日志权限', perm: 'log' } },
      
      // 作业任务
      { path: 'executor', name: 'Executor', component: () => import('@/views/executor/index.vue'), meta: { title: '批量执行', perm: 'executor' } },
      { path: 'task/schedules', name: 'TaskSchedules', component: () => import('@/views/task/schedules.vue'), meta: { title: '任务调度', perm: 'task' } },
      
      // AIOps智能
      { path: 'ai/assistant', name: 'AIAssistant', component: () => import('@/views/ai/index.vue'), meta: { title: 'AI智能助手', perm: 'ai' } },
      { path: 'ai/config', name: 'AIConfig', component: () => import('@/views/ai/index.vue'), meta: { title: '模型接入配置', perm: 'ai' } },
      { path: 'ai/ops', name: 'AIOps', component: () => import('@/views/ai/ops.vue'), meta: { title: '故障诊断', perm: 'ai' } },
      { path: 'ai-skills', name: 'AISkills', component: () => import('@/views/hub/ai-skills.vue'), meta: { title: 'AI技能管理', perm: 'ai' } },
      
      // 系统管理
      { path: 'system/users', name: 'SystemUsers', component: () => import('@/views/system/users.vue'), meta: { title: '用户管理', perm: 'system:user' } },
      { path: 'system/roles', name: 'SystemRoles', component: () => import('@/views/system/roles.vue'), meta: { title: '角色管理', perm: 'system:role' } },
      { path: 'system/menus', name: 'SystemMenus', component: () => import('@/views/system/menus.vue'), meta: { title: '权限管理', perm: 'system:permission' } },
      { path: 'system/dept', name: 'Department', component: () => import('@/views/system/dept.vue'), meta: { title: '部门管理', perm: 'system:dept' } },
      { path: 'system/posts', name: 'SystemPosts', component: () => import('@/views/system/posts.vue'), meta: { title: '岗位管理', perm: 'system:post' } },
      { path: 'system/login-logs', name: 'SystemLoginLogs', component: () => import('@/views/system/login-logs.vue'), meta: { title: '登录日志', perm: 'system:loginlog' } },
      { path: 'system/audit-logs', name: 'SystemAuditLogs', component: () => import('@/views/system/audit-logs.vue'), meta: { title: '操作日志', perm: 'system:log' } },
      
      // 堡垒机安全
      { path: 'jump/policies', name: 'JumpPolicies', component: () => import('@/views/jump/policies.vue'), meta: { title: '授权策略', perm: 'jump:policy' } },
      { path: 'jump/command-rules', name: 'JumpCommandRules', component: () => import('@/views/jump/command-rules.vue'), meta: { title: '命令风控', perm: 'jump:rule' } },
      { path: 'jump/sessions', name: 'JumpSessions', component: () => import('@/views/jump/sessions.vue'), meta: { title: '会话审计', perm: 'jump:session' } },
      { path: 'terminal', name: 'Terminal', component: () => import('@/views/terminal/index.vue'), meta: { title: 'WebTerminal', perm: 'terminal' } },

      // 服务拓扑
      { path: 'cmdb/topology', name: 'CMDBTopology', component: () => import('@/views/topology/graph.vue'), meta: { title: '服务拓扑', perm: 'cmdb' } },

      // 运维工单
      { path: 'workorder/orders', name: 'WorkOrders', component: () => import('@/views/workorder/tickets.vue'), meta: { title: '运维工单', perm: 'workorder' } },
      { path: 'workorder/types', name: 'WorkOrderTypes', component: () => import('@/views/workorder/types.vue'), meta: { title: '工单流程', perm: 'workorder' } },

      // 故障自愈
      { path: 'remediation/logs', name: 'RemediationLogs', component: () => import('@/views/remediation/index.vue'), meta: { title: '故障自愈', perm: 'remediation' } },

      // 值班排班
      { path: 'oncall/schedules', name: 'OncallSchedules', component: () => import('@/views/oncall/schedule.vue'), meta: { title: '值班排班', perm: 'oncall' } },
      { path: 'oncall/escalations', name: 'OncallEscalations', component: () => import('@/views/oncall/escalation.vue'), meta: { title: '升级策略', perm: 'oncall' } },

      // 通知管理
      { path: 'notify/channels', name: 'NotifyChannels', component: () => import('@/views/notify/channels.vue'), meta: { title: '通知渠道', perm: 'notify' } },
      { path: 'notify/groups', name: 'NotifyGroups', component: () => import('@/views/notify/groups.vue'), meta: { title: '通知组', perm: 'notify' } },
      { path: 'notify/templates', name: 'NotifyTemplates', component: () => import('@/views/notify/templates.vue'), meta: { title: '通知模板', perm: 'notify' } },

      // 运维流程
      { path: 'workflow/workflows', name: 'Workflows', component: () => import('@/views/workflow/designer.vue'), meta: { title: '运维流程', perm: 'workflow' } },
      { path: 'workflow/orchestrators', name: 'Orchestrators', component: () => import('@/views/workflow/orchestrator.vue'), meta: { title: '编排中心', perm: 'workflow' } },

      // Ansible管理
      { path: 'ansible/playbooks', name: 'AnsiblePlaybooks', component: () => import('@/views/ansible/playbooks.vue'), meta: { title: 'Playbooks', perm: 'ansible' } },
      { path: 'ansible/inventories', name: 'AnsibleInventories', component: () => import('@/views/ansible/inventories.vue'), meta: { title: '主机清单', perm: 'ansible' } },

      // 域名管理
      { path: 'domain/ssl', name: 'DomainSSL', component: () => import('@/views/domain/ssl.vue'), meta: { title: '域名证书', perm: 'domain' } },

      // SQL审计
      { path: 'sqlaudit/requests', name: 'SQLAuditRequests', component: () => import('@/views/sqlaudit/requests.vue'), meta: { title: 'SQL审计', perm: 'sqlaudit' } },
      { path: 'sqlaudit/rules', name: 'SQLAuditRules', component: () => import('@/views/sqlaudit/rules.vue'), meta: { title: '审计规则', perm: 'sqlaudit' } },

      // 成本分析
      { path: 'cost/overview', name: 'CostOverview', component: () => import('@/views/cost/overview.vue'), meta: { title: '成本分析', perm: 'cost' } },
      { path: 'cost/budget', name: 'CostBudget', component: () => import('@/views/cost/budget.vue'), meta: { title: '预算管理', perm: 'cost' } },

      // AI知识库
      { path: 'knowledge/docs', name: 'KnowledgeDocs', component: () => import('@/views/knowledge/index.vue'), meta: { title: 'AI知识库', perm: 'knowledge' } }
    ]
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'NotFound',
    redirect: '/403'
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
  try {
    const userStr = localStorage.getItem('user_info')
    if (userStr) {
      const user = JSON.parse(userStr)
      if (user?.role === 'admin' || user?.username === 'admin') return true
    }
    const perms = JSON.parse(localStorage.getItem('permissions') || '[]')
    if (Array.isArray(perms) && (perms.includes('*') || perms.includes('all') || perms.includes(code))) return true
    const parts = code.split(':')
    while (parts.length > 1) {
      parts.pop()
      if (Array.isArray(perms) && perms.includes(parts.join(':'))) return true
    }
  } catch (e) {
    return true
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

router.onError((error, to) => {
  const isChunkLoadFailed =
    error?.message?.includes('Failed to fetch dynamically imported module') ||
    error?.message?.includes('Importing a module script failed') ||
    error?.message?.includes('Loading chunk') ||
    error?.message?.includes('dynamically imported module')

  if (isChunkLoadFailed) {
    const key = 'chunk_reload_' + (to?.path || 'app')
    const lastReload = sessionStorage.getItem(key)
    if (!lastReload || Date.now() - Number(lastReload) > 10000) {
      sessionStorage.setItem(key, String(Date.now()))
      window.location.reload()
    }
  }
})

export default router
