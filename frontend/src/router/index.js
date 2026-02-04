import { createRouter, createWebHistory } from 'vue-router'
import Layout from '@/layout/index.vue'

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/login/index.vue')
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
        meta: { title: '仪表盘', icon: 'Odometer' }
      },
      {
        path: 'ai',
        name: 'AI',
        component: () => import('@/views/ai/index.vue'),
        meta: { title: 'AI运维助手', icon: 'MagicStick' }
      },

      // CMDB / Asset
      {
        path: 'host',
        name: 'Host',
        component: () => import('@/views/cmdb/host.vue'),
        meta: { title: '主机管理', icon: 'Monitor' }
      },
      {
        path: 'cmdb/group',
        name: 'CMDBGroup',
        component: () => import('@/views/cmdb/group.vue'),
        meta: { title: '主机分组', icon: 'FolderOpened' }
      },
      {
        path: 'cmdb/credential',
        name: 'CMDBCredential',
        component: () => import('@/views/cmdb/credential.vue'),
        meta: { title: '凭据管理', icon: 'Key' }
      },
      {
        path: 'cmdb/database',
        name: 'CMDBDatabase',
        component: () => import('@/views/cmdb/database.vue'),
        meta: { title: '数据库资产', icon: 'Coin' }
      },
      {
        path: 'cmdb/cloud',
        name: 'CMDBCloud',
        component: () => import('@/views/cmdb/cloud.vue'),
        meta: { title: '云资源', icon: 'Cloudy' }
      },
      {
        path: 'firewall',
        name: 'Firewall',
        component: () => import('@/views/firewall/index.vue'),
        meta: { title: '防火墙管理', icon: 'Lock' }
      },

      // Container / K8s
      {
        path: 'docker',
        name: 'Docker',
        component: () => import('@/views/docker/index.vue'),
        meta: { title: 'Docker管理', icon: 'Platform' }
      },
      {
        path: 'k8s/clusters',
        name: 'K8sClusters',
        component: () => import('@/views/k8s/clusters.vue'),
        meta: { title: 'K8s集群', icon: 'Connection' }
      },
      {
        path: 'k8s/namespaces',
        name: 'K8sNamespaces',
        component: () => import('@/views/k8s/namespaces.vue'),
        meta: { title: '命名空间', icon: 'Collection' }
      },
      {
        path: 'k8s/workloads',
        name: 'K8sWorkloads',
        component: () => import('@/views/k8s/workloads.vue'),
        meta: { title: '工作负载', icon: 'Cpu' }
      },
      {
        path: 'k8s/workloads/detail',
        name: 'K8sWorkloadDetail',
        component: () => import('@/views/k8s/workload-detail.vue'),
        meta: { title: '工作负载详情', icon: 'Cpu' }
      },
      {
        path: 'k8s/pods',
        name: 'K8sPods',
        component: () => import('@/views/k8s/pods.vue'),
        meta: { title: 'Pods', icon: 'Box' }
      },
      {
        path: 'k8s/pods/detail',
        name: 'K8sPodDetail',
        component: () => import('@/views/k8s/pod-detail.vue'),
        meta: { title: 'Pod详情', icon: 'Box' }
      },
      {
        path: 'k8s/services',
        name: 'K8sServices',
        component: () => import('@/views/k8s/services.vue'),
        meta: { title: '服务与Ingress', icon: 'Share' }
      },
      {
        path: 'k8s/configs',
        name: 'K8sConfigs',
        component: () => import('@/views/k8s/configs.vue'),
        meta: { title: 'Config/Secret', icon: 'Tickets' }
      },
      {
        path: 'k8s/storage',
        name: 'K8sStorage',
        component: () => import('@/views/k8s/storage.vue'),
        meta: { title: '存储管理', icon: 'Coin' }
      },
      {
        path: 'k8s/nodes',
        name: 'K8sNodes',
        component: () => import('@/views/k8s/nodes.vue'),
        meta: { title: '节点管理', icon: 'Grid' }
      },
      {
        path: 'k8s/events',
        name: 'K8sEvents',
        component: () => import('@/views/k8s/events.vue'),
        meta: { title: '事件与诊断', icon: 'Warning' }
      },
      {
        path: 'k8s/terminal',
        name: 'K8sTerminal',
        component: () => import('@/views/k8s/terminal.vue'),
        meta: { title: 'K8s WebShell', icon: 'Monitor' }
      },

      // Monitoring / Alert
      {
        path: 'monitor/overview',
        name: 'MonitorOverview',
        component: () => import('@/views/monitor/overview.vue'),
        meta: { title: '监控概览', icon: 'Histogram' }
      },
      {
        path: 'monitor/metrics',
        name: 'MonitorMetrics',
        component: () => import('@/views/monitor/metrics.vue'),
        meta: { title: '指标采集', icon: 'DataBoard' }
      },
      {
        path: 'monitor/agents',
        name: 'MonitorAgents',
        component: () => import('@/views/monitor/agents.vue'),
        meta: { title: 'Agent心跳', icon: 'AlarmClock' }
      },
      {
        path: 'monitor/agents/detail',
        name: 'MonitorAgentDetail',
        component: () => import('@/views/monitor/agent-detail.vue'),
        meta: { title: 'Agent详情', icon: 'AlarmClock' }
      },
      {
        path: 'alert/rules',
        name: 'AlertRules',
        component: () => import('@/views/alert/rules.vue'),
        meta: { title: '告警规则', icon: 'Bell' }
      },
      {
        path: 'alert/events',
        name: 'AlertEvents',
        component: () => import('@/views/alert/events.vue'),
        meta: { title: '告警事件', icon: 'Notification' }
      },
      {
        path: 'alert/silences',
        name: 'AlertSilences',
        component: () => import('@/views/alert/silences.vue'),
        meta: { title: '告警静默', icon: 'Notification' }
      },
      {
        path: 'alert/aggregation',
        name: 'AlertAggregation',
        component: () => import('@/views/alert/aggregation.vue'),
        meta: { title: '告警聚合', icon: 'Notification' }
      },
      {
        path: 'alert/history',
        name: 'AlertHistory',
        component: () => import('@/views/alert/history.vue'),
        meta: { title: '告警复盘', icon: 'Notification' }
      },
      {
        path: 'alert/history/detail',
        name: 'AlertHistoryDetail',
        component: () => import('@/views/alert/history-detail.vue'),
        meta: { title: '复盘详情', icon: 'Notification' }
      },
      {
        path: 'alert/events/detail',
        name: 'AlertDetail',
        component: () => import('@/views/alert/detail.vue'),
        meta: { title: '告警详情', icon: 'Notification' }
      },
      {
        path: 'notify/channels',
        name: 'NotifyChannels',
        component: () => import('@/views/notify/channels.vue'),
        meta: { title: '通知渠道', icon: 'Message' }
      },
      {
        path: 'notify/groups',
        name: 'NotifyGroups',
        component: () => import('@/views/notify/groups.vue'),
        meta: { title: '通知组', icon: 'Message' }
      },
      {
        path: 'notify/templates',
        name: 'NotifyTemplates',
        component: () => import('@/views/notify/templates.vue'),
        meta: { title: '通知模板', icon: 'Document' }
      },
      {
        path: 'domain/ssl',
        name: 'DomainSSL',
        component: () => import('@/views/domain/ssl.vue'),
        meta: { title: '域名与证书', icon: 'Link' }
      },

      // Automation
      {
        path: 'workflow/designer',
        name: 'WorkflowDesigner',
        component: () => import('@/views/workflow/designer.vue'),
        meta: { title: '工作流编排', icon: 'Operation' }
      },
      {
        path: 'executor',
        name: 'Executor',
        component: () => import('@/views/executor/index.vue'),
        meta: { title: '批量执行', icon: 'Tools' }
      },
      {
        path: 'task/schedules',
        name: 'TaskSchedules',
        component: () => import('@/views/task/schedules.vue'),
        meta: { title: '任务调度', icon: 'Clock' }
      },
      {
        path: 'ansible/playbooks',
        name: 'AnsiblePlaybooks',
        component: () => import('@/views/ansible/playbooks.vue'),
        meta: { title: 'Playbook管理', icon: 'Document' }
      },
      {
        path: 'ansible/inventories',
        name: 'AnsibleInventories',
        component: () => import('@/views/ansible/inventories.vue'),
        meta: { title: 'Inventory管理', icon: 'List' }
      },

      // CI/CD
      {
        path: 'cicd/pipelines',
        name: 'CICDPipelines',
        component: () => import('@/views/cicd/pipelines.vue'),
        meta: { title: '流水线管理', icon: 'Connection' }
      },
      {
        path: 'cicd/executions',
        name: 'CICDExecutions',
        component: () => import('@/views/cicd/executions.vue'),
        meta: { title: '执行记录', icon: 'List' }
      },
      {
        path: 'cicd/schedules',
        name: 'CICDSchedules',
        component: () => import('@/views/cicd/schedules.vue'),
        meta: { title: '定时发布', icon: 'AlarmClock' }
      },
      {
        path: 'cicd/releases',
        name: 'CICDReleases',
        component: () => import('@/views/cicd/releases.vue'),
        meta: { title: '发布管理', icon: 'Tickets' }
      },

      // Config Center
      {
        path: 'nacos/servers',
        name: 'NacosServers',
        component: () => import('@/views/nacos/servers.vue'),
        meta: { title: 'Nacos服务器', icon: 'Connection' }
      },
      {
        path: 'nacos/configs',
        name: 'NacosConfigs',
        component: () => import('@/views/nacos/configs.vue'),
        meta: { title: '配置管理', icon: 'Edit' }
      },

      // Change Management
      {
        path: 'workorder/tickets',
        name: 'WorkorderTickets',
        component: () => import('@/views/workorder/tickets.vue'),
        meta: { title: '工单管理', icon: 'Tickets' }
      },
      {
        path: 'workorder/types',
        name: 'WorkorderTypes',
        component: () => import('@/views/workorder/types.vue'),
        meta: { title: '工单类型', icon: 'CollectionTag' }
      },
      {
        path: 'sqlaudit/requests',
        name: 'SQLAuditRequests',
        component: () => import('@/views/sqlaudit/requests.vue'),
        meta: { title: 'SQL工单', icon: 'Document' }
      },
      {
        path: 'sqlaudit/rules',
        name: 'SQLAuditRules',
        component: () => import('@/views/sqlaudit/rules.vue'),
        meta: { title: 'SQL审核规则', icon: 'Warning' }
      },
      {
        path: 'gitops/repos',
        name: 'GitOpsRepos',
        component: () => import('@/views/gitops/repos.vue'),
        meta: { title: 'GitOps仓库', icon: 'Share' }
      },
      {
        path: 'gitops/sync',
        name: 'GitOpsSync',
        component: () => import('@/views/gitops/sync.vue'),
        meta: { title: '同步记录', icon: 'List' }
      },

      // Collaboration
      {
        path: 'oncall/schedule',
        name: 'OncallSchedule',
        component: () => import('@/views/oncall/schedule.vue'),
        meta: { title: '值班排班', icon: 'Calendar' }
      },
      {
        path: 'oncall/escalation',
        name: 'OncallEscalation',
        component: () => import('@/views/oncall/escalation.vue'),
        meta: { title: '升级策略', icon: 'Bell' }
      },
      {
        path: 'terminal',
        name: 'Terminal',
        component: () => import('@/views/terminal/index.vue'),
        meta: { title: 'WebTerminal', icon: 'Monitor' }
      },

      // Visualization / Cost
      {
        path: 'topology',
        name: 'Topology',
        component: () => import('@/views/topology/graph.vue'),
        meta: { title: '服务拓扑', icon: 'Share' }
      },
      {
        path: 'cost/overview',
        name: 'CostOverview',
        component: () => import('@/views/cost/overview.vue'),
        meta: { title: '成本概览', icon: 'Coin' }
      },
      {
        path: 'cost/budget',
        name: 'CostBudget',
        component: () => import('@/views/cost/budget.vue'),
        meta: { title: '预算与告警', icon: 'Warning' }
      },

      // App Center
      {
        path: 'application',
        name: 'Application',
        component: () => import('@/views/application/index.vue'),
        meta: { title: '应用中心', icon: 'Box' }
      },

      // System
      {
        path: 'system/users',
        name: 'SystemUsers',
        component: () => import('@/views/system/users.vue'),
        meta: { title: '用户管理', icon: 'User' }
      },
      {
        path: 'system/roles',
        name: 'SystemRoles',
        component: () => import('@/views/system/roles.vue'),
        meta: { title: '角色管理', icon: 'UserFilled' }
      },
      {
        path: 'system/menus',
        name: 'SystemMenus',
        component: () => import('@/views/system/menus.vue'),
        meta: { title: '菜单管理', icon: 'Menu' }
      },
      {
        path: 'system/dept',
        name: 'Department',
        component: () => import('@/views/system/dept.vue'),
        meta: { title: '部门管理', icon: 'OfficeBuilding' }
      },
      {
        path: 'system/posts',
        name: 'SystemPosts',
        component: () => import('@/views/system/posts.vue'),
        meta: { title: '岗位管理', icon: 'Briefcase' }
      },
      {
        path: 'system/login-logs',
        name: 'SystemLoginLogs',
        component: () => import('@/views/system/login-logs.vue'),
        meta: { title: '登录日志', icon: 'Notebook' }
      },
      {
        path: 'system/audit-logs',
        name: 'SystemAuditLogs',
        component: () => import('@/views/system/audit-logs.vue'),
        meta: { title: '操作日志', icon: 'Notebook' }
      },
      {
        path: 'system/captcha',
        name: 'SystemCaptcha',
        component: () => import('@/views/system/captcha.vue'),
        meta: { title: '验证码配置', icon: 'Lock' }
      }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to, from, next) => {
  const token = localStorage.getItem('token')
  if (to.name !== 'Login' && !token) next({ name: 'Login' })
  else next()
})

export default router
