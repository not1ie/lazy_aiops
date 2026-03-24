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
        path: 'cmdb',
        redirect: '/asset/overview'
      },
      {
        path: 'k8s',
        redirect: '/k8s/overview'
      },
      {
        path: 'monitor',
        redirect: '/monitor/center'
      },
      {
        path: 'domain',
        redirect: '/domain/center'
      },
      {
        path: 'cicd',
        redirect: '/delivery/center'
      },
      {
        path: 'delivery',
        redirect: '/delivery/center'
      },

      // CMDB / Asset
      {
        path: 'asset/overview',
        name: 'AssetOverview',
        component: () => import('@/views/hub/asset.vue'),
        meta: { title: '资产总览', icon: 'DataBoard', perm: 'cmdb' }
      },
      {
        path: 'asset/ops',
        name: 'AssetOps',
        component: () => import('@/views/hub/asset-ops.vue'),
        meta: { title: '资产作战台', icon: 'Monitor' }
      },
      {
        path: 'host',
        name: 'Host',
        component: () => import('@/views/cmdb/host.vue'),
        meta: { title: '主机管理', icon: 'Monitor', perm: 'cmdb' }
      },
      {
        path: 'cmdb/group',
        name: 'CMDBGroup',
        component: () => import('@/views/cmdb/group.vue'),
        meta: { title: '主机分组', icon: 'FolderOpened', perm: 'cmdb' }
      },
      {
        path: 'cmdb/credential',
        name: 'CMDBCredential',
        component: () => import('@/views/cmdb/credential.vue'),
        meta: { title: '凭据管理', icon: 'Key', perm: 'cmdb' }
      },
      {
        path: 'cmdb/database',
        name: 'CMDBDatabase',
        component: () => import('@/views/cmdb/database.vue'),
        meta: { title: '数据库资产', icon: 'Coin', perm: 'cmdb' }
      },
      {
        path: 'cmdb/cloud',
        name: 'CMDBCloud',
        component: () => import('@/views/cmdb/cloud.vue'),
        meta: { title: '云资源', icon: 'Cloudy', perm: 'cmdb' }
      },
      {
        path: 'cmdb/network-devices',
        name: 'CMDBNetworkDevices',
        component: () => import('@/views/cmdb/network-device.vue'),
        meta: { title: '网络设备', icon: 'Switch', perm: 'cmdb' }
      },
      {
        path: 'firewall',
        name: 'Firewall',
        component: () => import('@/views/firewall/index.vue'),
        meta: { title: '防火墙管理', icon: 'Lock', perm: 'firewall' }
      },

      // Container / K8s
      {
        path: 'k8s/overview',
        name: 'K8sOverview',
        component: () => import('@/views/hub/k8s.vue'),
        meta: { title: '容器平台总览', icon: 'Connection', perm: 'k8s' }
      },
      {
        path: 'docker',
        name: 'Docker',
        component: () => import('@/views/docker/index.vue'),
        meta: { title: 'Docker管理', icon: 'Platform', perm: 'docker' }
      },
      {
        path: 'k8s/clusters',
        name: 'K8sClusters',
        component: () => import('@/views/k8s/clusters.vue'),
        meta: { title: 'K8s集群', icon: 'Connection', perm: 'k8s' }
      },
      {
        path: 'k8s/namespaces',
        name: 'K8sNamespaces',
        component: () => import('@/views/k8s/namespaces.vue'),
        meta: { title: '命名空间', icon: 'Collection', perm: 'k8s' }
      },
      {
        path: 'k8s/workloads',
        name: 'K8sWorkloads',
        component: () => import('@/views/k8s/workloads.vue'),
        meta: { title: '工作负载', icon: 'Cpu', perm: 'k8s' }
      },
      {
        path: 'k8s/deployments',
        name: 'K8sDeployments',
        component: () => import('@/views/k8s/deployments.vue'),
        meta: { title: 'Deployments', icon: 'Cpu', perm: 'k8s' }
      },
      {
        path: 'k8s/workloads/detail',
        name: 'K8sWorkloadDetail',
        component: () => import('@/views/k8s/workload-detail.vue'),
        meta: { title: '工作负载详情', icon: 'Cpu', perm: 'k8s' }
      },
      {
        path: 'k8s/pods',
        name: 'K8sPods',
        component: () => import('@/views/k8s/pods.vue'),
        meta: { title: 'Pods', icon: 'Box', perm: 'k8s' }
      },
      {
        path: 'k8s/pods/detail',
        name: 'K8sPodDetail',
        component: () => import('@/views/k8s/pod-detail.vue'),
        meta: { title: 'Pod详情', icon: 'Box', perm: 'k8s' }
      },
      {
        path: 'k8s/services',
        name: 'K8sServices',
        component: () => import('@/views/k8s/services.vue'),
        meta: { title: '服务与Ingress', icon: 'Share', perm: 'k8s' }
      },
      {
        path: 'k8s/configs',
        name: 'K8sConfigs',
        component: () => import('@/views/k8s/configs.vue'),
        meta: { title: 'Config/Secret', icon: 'Tickets', perm: 'k8s' }
      },
      {
        path: 'k8s/storage',
        name: 'K8sStorage',
        component: () => import('@/views/k8s/storage.vue'),
        meta: { title: '存储管理', icon: 'Coin', perm: 'k8s' }
      },
      {
        path: 'k8s/nodes',
        name: 'K8sNodes',
        component: () => import('@/views/k8s/nodes.vue'),
        meta: { title: '节点管理', icon: 'Grid', perm: 'k8s' }
      },
      {
        path: 'k8s/events',
        name: 'K8sEvents',
        component: () => import('@/views/k8s/events.vue'),
        meta: { title: '事件与诊断', icon: 'Warning', perm: 'k8s' }
      },
      {
        path: 'k8s/terminal',
        name: 'K8sTerminal',
        component: () => import('@/views/k8s/terminal.vue'),
        meta: { title: 'K8s WebShell', icon: 'Monitor', perm: 'k8s' }
      },

      // Monitoring / Alert
      {
        path: 'monitor/center',
        name: 'MonitorCenter',
        component: () => import('@/views/hub/monitor.vue'),
        meta: { title: '监控告警中心', icon: 'Histogram' }
      },
      {
        path: 'monitor/hosts',
        name: 'MonitorHosts',
        component: () => import('@/views/monitor/hosts.vue'),
        meta: { title: '主机监控', icon: 'Monitor', perm: 'monitor' }
      },
      {
        path: 'monitor/overview',
        name: 'MonitorOverview',
        component: () => import('@/views/monitor/overview.vue'),
        meta: { title: '监控概览', icon: 'Histogram', perm: 'monitor' }
      },
      {
        path: 'monitor/metrics',
        name: 'MonitorMetrics',
        component: () => import('@/views/monitor/metrics.vue'),
        meta: { title: '指标采集', icon: 'DataBoard', perm: 'monitor' }
      },
      {
        path: 'monitor/containers',
        name: 'MonitorContainers',
        component: () => import('@/views/monitor/containers.vue'),
        meta: { title: '容器监控', icon: 'Box', perm: 'monitor' }
      },
      {
        path: 'monitor/pods',
        name: 'MonitorPods',
        component: () => import('@/views/monitor/pods.vue'),
        meta: { title: 'Pod监控', icon: 'Histogram', perm: 'monitor' }
      },
      {
        path: 'monitor/agents',
        name: 'MonitorAgents',
        component: () => import('@/views/monitor/agents.vue'),
        meta: { title: 'Agent心跳', icon: 'AlarmClock', perm: 'monitor' }
      },
      {
        path: 'monitor/agents/detail',
        name: 'MonitorAgentDetail',
        component: () => import('@/views/monitor/agent-detail.vue'),
        meta: { title: 'Agent详情', icon: 'AlarmClock', perm: 'monitor' }
      },
      {
        path: 'alert/rules',
        name: 'AlertRules',
        component: () => import('@/views/alert/rules.vue'),
        meta: { title: '告警规则', icon: 'Bell', perm: 'alert' }
      },
      {
        path: 'alert/events',
        name: 'AlertEvents',
        component: () => import('@/views/alert/events.vue'),
        meta: { title: '告警事件', icon: 'Notification', perm: 'alert' }
      },
      {
        path: 'alert/silences',
        name: 'AlertSilences',
        component: () => import('@/views/alert/silences.vue'),
        meta: { title: '告警静默', icon: 'Notification', perm: 'alert' }
      },
      {
        path: 'alert/aggregation',
        name: 'AlertAggregation',
        component: () => import('@/views/alert/aggregation.vue'),
        meta: { title: '告警聚合', icon: 'Notification', perm: 'alert' }
      },
      {
        path: 'alert/history',
        name: 'AlertHistory',
        component: () => import('@/views/alert/history.vue'),
        meta: { title: '告警复盘', icon: 'Notification', perm: 'alert' }
      },
      {
        path: 'alert/history/detail',
        name: 'AlertHistoryDetail',
        component: () => import('@/views/alert/history-detail.vue'),
        meta: { title: '复盘详情', icon: 'Notification', perm: 'alert' }
      },
      {
        path: 'alert/events/detail',
        name: 'AlertDetail',
        component: () => import('@/views/alert/detail.vue'),
        meta: { title: '告警详情', icon: 'Notification', perm: 'alert' }
      },
      {
        path: 'notify/channels',
        name: 'NotifyChannels',
        component: () => import('@/views/notify/channels.vue'),
        meta: { title: '通知渠道', icon: 'Message', perm: 'notify' }
      },
      {
        path: 'notify/groups',
        name: 'NotifyGroups',
        component: () => import('@/views/notify/groups.vue'),
        meta: { title: '通知组', icon: 'Message', perm: 'notify' }
      },
      {
        path: 'notify/templates',
        name: 'NotifyTemplates',
        component: () => import('@/views/notify/templates.vue'),
        meta: { title: '通知模板', icon: 'Document', perm: 'notify' }
      },
      {
        path: 'domain/ssl',
        name: 'DomainSSL',
        component: () => import('@/views/domain/ssl.vue'),
        meta: { title: '域名与证书', icon: 'Link', perm: 'domain' }
      },
      {
        path: 'domain/center',
        name: 'DomainCenter',
        component: () => import('@/views/hub/domain-center.vue'),
        meta: { title: '域名监控中心', icon: 'Link' }
      },

      // Automation
      {
        path: 'workflow/designer',
        name: 'WorkflowDesigner',
        component: () => import('@/views/workflow/designer.vue'),
        meta: { title: '工作流编排', icon: 'Operation', perm: 'workflow' }
      },
      {
        path: 'executor',
        name: 'Executor',
        component: () => import('@/views/executor/index.vue'),
        meta: { title: '批量执行', icon: 'Tools', perm: 'executor' }
      },
      {
        path: 'task/schedules',
        name: 'TaskSchedules',
        component: () => import('@/views/task/schedules.vue'),
        meta: { title: '任务调度', icon: 'Clock', perm: 'task' }
      },
      {
        path: 'ansible/playbooks',
        name: 'AnsiblePlaybooks',
        component: () => import('@/views/ansible/playbooks.vue'),
        meta: { title: 'Playbook管理', icon: 'Document', perm: 'ansible' }
      },
      {
        path: 'ansible/inventories',
        name: 'AnsibleInventories',
        component: () => import('@/views/ansible/inventories.vue'),
        meta: { title: 'Inventory管理', icon: 'List', perm: 'ansible' }
      },

      // CI/CD
      {
        path: 'delivery/center',
        name: 'DeliveryCenter',
        component: () => import('@/views/hub/delivery.vue'),
        meta: { title: '交付中心', icon: 'Connection' }
      },
      {
        path: 'cicd/pipelines',
        name: 'CICDPipelines',
        component: () => import('@/views/cicd/pipelines.vue'),
        meta: { title: '流水线管理', icon: 'Connection', perm: 'cicd' }
      },
      {
        path: 'cicd/executions',
        name: 'CICDExecutions',
        component: () => import('@/views/cicd/executions.vue'),
        meta: { title: '执行记录', icon: 'List', perm: 'cicd' }
      },
      {
        path: 'cicd/schedules',
        name: 'CICDSchedules',
        component: () => import('@/views/cicd/schedules.vue'),
        meta: { title: '定时发布', icon: 'AlarmClock', perm: 'cicd' }
      },
      {
        path: 'cicd/releases',
        name: 'CICDReleases',
        component: () => import('@/views/cicd/releases.vue'),
        meta: { title: '发布管理', icon: 'Tickets', perm: 'cicd' }
      },

      // Config Center
      {
        path: 'nacos/servers',
        name: 'NacosServers',
        component: () => import('@/views/nacos/servers.vue'),
        meta: { title: 'Nacos服务器', icon: 'Connection', perm: 'nacos' }
      },
      {
        path: 'nacos/configs',
        name: 'NacosConfigs',
        component: () => import('@/views/nacos/configs.vue'),
        meta: { title: '配置管理', icon: 'Edit', perm: 'nacos' }
      },

      // Change Management
      {
        path: 'workorder/tickets',
        name: 'WorkorderTickets',
        component: () => import('@/views/workorder/tickets.vue'),
        meta: { title: '工单管理', icon: 'Tickets', perm: 'workorder' }
      },
      {
        path: 'workorder/types',
        name: 'WorkorderTypes',
        component: () => import('@/views/workorder/types.vue'),
        meta: { title: '工单类型', icon: 'CollectionTag', perm: 'workorder' }
      },
      {
        path: 'sqlaudit/requests',
        name: 'SQLAuditRequests',
        component: () => import('@/views/sqlaudit/requests.vue'),
        meta: { title: 'SQL工单', icon: 'Document', perm: 'sqlaudit' }
      },
      {
        path: 'sqlaudit/rules',
        name: 'SQLAuditRules',
        component: () => import('@/views/sqlaudit/rules.vue'),
        meta: { title: 'SQL审核规则', icon: 'Warning', perm: 'sqlaudit' }
      },
      {
        path: 'gitops/repos',
        name: 'GitOpsRepos',
        component: () => import('@/views/gitops/repos.vue'),
        meta: { title: 'GitOps仓库', icon: 'Share', perm: 'gitops' }
      },
      {
        path: 'gitops/sync',
        name: 'GitOpsSync',
        component: () => import('@/views/gitops/sync.vue'),
        meta: { title: '同步记录', icon: 'List', perm: 'gitops' }
      },

      {
        path: 'oncall/schedule',
        name: 'OncallSchedule',
        component: () => import('@/views/oncall/schedule.vue'),
        meta: { title: '值班排班', icon: 'Calendar', perm: 'oncall' }
      },
      {
        path: 'oncall/escalation',
        name: 'OncallEscalation',
        component: () => import('@/views/oncall/escalation.vue'),
        meta: { title: '升级策略', icon: 'Bell', perm: 'oncall' }
      },
      {
        path: 'jump/assets',
        name: 'JumpAssets',
        component: () => import('@/views/jump/assets.vue'),
        meta: { title: '堡垒机资产', icon: 'Monitor', perm: 'jump:asset' }
      },
      {
        path: 'jump/policies',
        name: 'JumpPolicies',
        component: () => import('@/views/jump/policies.vue'),
        meta: { title: '授权策略', icon: 'Lock', perm: 'jump:policy' }
      },
      {
        path: 'jump/command-rules',
        name: 'JumpCommandRules',
        component: () => import('@/views/jump/command-rules.vue'),
        meta: { title: '命令风控', icon: 'Warning', perm: 'jump:rule' }
      },
      {
        path: 'jump/sessions',
        name: 'JumpSessions',
        component: () => import('@/views/jump/sessions.vue'),
        meta: { title: '会话审计', icon: 'Document', perm: 'jump:session' }
      },
      {
        path: 'terminal',
        name: 'Terminal',
        component: () => import('@/views/terminal/index.vue'),
        meta: { title: 'WebTerminal', icon: 'Monitor', perm: 'terminal' }
      },

      // Visualization / Cost
      {
        path: 'topology',
        name: 'Topology',
        component: () => import('@/views/topology/graph.vue'),
        meta: { title: '服务拓扑', icon: 'Share', perm: 'topology' }
      },
      {
        path: 'cost/overview',
        name: 'CostOverview',
        component: () => import('@/views/cost/overview.vue'),
        meta: { title: '成本概览', icon: 'Coin', perm: 'cost' }
      },
      {
        path: 'cost/budget',
        name: 'CostBudget',
        component: () => import('@/views/cost/budget.vue'),
        meta: { title: '预算与告警', icon: 'Warning', perm: 'cost' }
      },

      // App Center
      {
        path: 'application',
        name: 'Application',
        component: () => import('@/views/application/index.vue'),
        meta: { title: '应用中心', icon: 'Box', perm: 'application' }
      },

      // System
      {
        path: 'system/users',
        name: 'SystemUsers',
        component: () => import('@/views/system/users.vue'),
        meta: { title: '用户管理', icon: 'User', perm: 'system:user' }
      },
      {
        path: 'system/roles',
        name: 'SystemRoles',
        component: () => import('@/views/system/roles.vue'),
        meta: { title: '角色管理', icon: 'UserFilled', perm: 'system:role' }
      },
      {
        path: 'system/menus',
        name: 'SystemMenus',
        component: () => import('@/views/system/menus.vue'),
        meta: { title: '权限管理', icon: 'Menu', perm: 'system:permission' }
      },
      {
        path: 'system/dept',
        name: 'Department',
        component: () => import('@/views/system/dept.vue'),
        meta: { title: '部门管理', icon: 'OfficeBuilding', perm: 'system:dept' }
      },
      {
        path: 'system/posts',
        name: 'SystemPosts',
        component: () => import('@/views/system/posts.vue'),
        meta: { title: '岗位管理', icon: 'Briefcase', perm: 'system:post' }
      },
      {
        path: 'system/login-logs',
        name: 'SystemLoginLogs',
        component: () => import('@/views/system/login-logs.vue'),
        meta: { title: '登录日志', icon: 'Notebook', perm: 'system:loginlog' }
      },
      {
        path: 'system/audit-logs',
        name: 'SystemAuditLogs',
        component: () => import('@/views/system/audit-logs.vue'),
        meta: { title: '操作日志', icon: 'Notebook', perm: 'system:log' }
      },
      {
        path: 'system/captcha',
        name: 'SystemCaptcha',
        component: () => import('@/views/system/captcha.vue'),
        meta: { title: '验证码配置', icon: 'Lock', perm: 'system:captcha' }
      }
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
