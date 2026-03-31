<template>
  <el-card class="dashboard-card" v-loading="loading">
    <div class="page-header motion-up delay-1">
      <div>
        <h2>全局仪表盘</h2>
        <p class="page-desc">主机、容器、K8s、告警与任务的统一视图。</p>
      </div>
      <div class="page-actions">
        <span class="updated-at" v-if="lastUpdated">刷新时间：{{ lastUpdated }}</span>
        <el-button icon="Refresh" :loading="refreshing" @click="refreshDashboard">刷新</el-button>
      </div>
    </div>

    <div class="scope-bar motion-up delay-2">
      <span class="scope-label">范围</span>
      <el-select v-model="scope.environment" class="scope-item" @change="handleScopeEnvironmentChange">
        <el-option v-for="item in environmentOptions" :key="item.value" :label="item.label" :value="item.value" />
      </el-select>
      <el-select v-model="scope.clusterId" class="scope-item" clearable placeholder="全部集群" @change="handleScopeClusterChange">
        <el-option v-for="item in scopeClusters" :key="item.id" :label="item.display_name || item.name" :value="item.id" />
      </el-select>
      <el-select v-model="scope.namespace" class="scope-item" clearable placeholder="全部命名空间" @change="refreshDashboard">
        <el-option v-for="item in scopeNamespaces" :key="item" :label="item" :value="item" />
      </el-select>
      <el-select v-model="scope.timeWindowHours" class="scope-item narrow" @change="refreshDashboard">
        <el-option v-for="item in timeWindowOptions" :key="item.value" :label="item.label" :value="item.value" />
      </el-select>
    </div>

    <el-row :gutter="16" class="motion-up delay-3">
      <el-col :span="4">
        <el-card class="kpi-card" shadow="never">
          <div class="kpi-title">主机总数</div>
          <div class="kpi-value">{{ stats.hostTotal }}</div>
          <div class="kpi-sub">在线 {{ stats.hostOnline }} / 离线 {{ stats.hostOffline }}</div>
        </el-card>
      </el-col>
      <el-col :span="4">
        <el-card class="kpi-card" shadow="never">
          <div class="kpi-title">Docker 环境</div>
          <div class="kpi-value">{{ stats.dockerTotal }}</div>
          <div class="kpi-sub">在线 {{ stats.dockerOnline }} / 离线 {{ stats.dockerOffline }}</div>
        </el-card>
      </el-col>
      <el-col :span="4">
        <el-card class="kpi-card" shadow="never">
          <div class="kpi-title">K8s 集群</div>
          <div class="kpi-value">{{ stats.k8sTotal }}</div>
          <div class="kpi-sub">正常 {{ stats.k8sHealthy }}</div>
        </el-card>
      </el-col>
      <el-col :span="4">
        <el-card class="kpi-card" shadow="never">
          <div class="kpi-title">活跃告警</div>
          <div class="kpi-value danger">{{ stats.alertOpen }}</div>
          <div class="kpi-sub">总告警 {{ stats.alertTotal }}</div>
        </el-card>
      </el-col>
      <el-col :span="4">
        <el-card class="kpi-card" shadow="never">
          <div class="kpi-title">启用任务</div>
          <div class="kpi-value">{{ stats.taskEnabled }}</div>
          <div class="kpi-sub">总任务 {{ stats.taskTotal }}</div>
        </el-card>
      </el-col>
      <el-col :span="4">
        <el-card class="kpi-card" shadow="never">
          <div class="kpi-title">在线 Agent</div>
          <div class="kpi-value">{{ stats.agentOnline }}</div>
          <div class="kpi-sub">总 Agent {{ stats.agentTotal }}</div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="panel-row motion-up delay-4">
      <el-col :span="24">
        <el-card shadow="never" class="backlog-overview-card">
          <div class="panel-header backlog-header">
            <div>
              <h3>待处置积压总览</h3>
              <p class="panel-desc">按中心聚合超时与未闭环事项，可直接跳转处置。</p>
            </div>
            <div class="backlog-totals">
              <el-tag type="warning" effect="light">总积压 {{ backlog.total }}</el-tag>
              <el-tag type="danger" effect="light">超时 {{ backlog.overdue }}</el-tag>
            </div>
          </div>
          <div class="backlog-grid">
            <div v-for="item in backlogCards" :key="item.key" class="backlog-item">
              <div class="backlog-item-top">
                <span class="backlog-label">{{ item.label }}</span>
                <el-tag size="small" :type="item.overdue > 0 ? 'warning' : 'success'">
                  {{ item.overdue > 0 ? `超时 ${item.overdue}` : '正常' }}
                </el-tag>
              </div>
              <div class="backlog-value">{{ item.value }}</div>
              <div class="backlog-desc">{{ item.desc }}</div>
              <div class="backlog-actions">
                <el-button
                  link
                  type="warning"
                  :disabled="item.value <= 0"
                  :loading="backlogActionLoading[item.key]"
                  @click="runBacklogAction(item.key)"
                >
                  一键处置
                </el-button>
                <el-button link type="primary" @click="go(item.path)">进入处置</el-button>
              </div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="panel-row motion-up delay-5">
      <el-col :span="16">
        <el-card shadow="never">
          <div class="panel-header">
            <div>
              <h3>系统趋势（24h）</h3>
              <p class="panel-desc">CPU / 内存 / 磁盘历史变化。</p>
            </div>
          </div>
          <div ref="trendRef" class="chart-box"></div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="never" class="stack-card">
          <div class="panel-header">
            <div>
              <h3>实时资源</h3>
              <p class="panel-desc">当前主机资源占用。</p>
            </div>
          </div>
          <div class="resource-row">
            <span>CPU</span>
            <el-progress :percentage="safePercent(realtime.cpu)" :show-text="false" />
            <b>{{ formatPercent(realtime.cpu) }}</b>
          </div>
          <div class="resource-row">
            <span>内存</span>
            <el-progress :percentage="safePercent(realtime.memory)" :show-text="false" />
            <b>{{ formatPercent(realtime.memory) }}</b>
          </div>
          <div class="resource-row">
            <span>磁盘</span>
            <el-progress :percentage="safePercent(realtime.disk)" :show-text="false" />
            <b>{{ formatPercent(realtime.disk) }}</b>
          </div>
          <div class="resource-row">
            <span>网络</span>
            <div class="network-value">{{ formatNumber(realtime.network) }} MB/s</div>
          </div>
        </el-card>

        <el-card shadow="never" class="stack-card">
          <div class="panel-header">
            <div>
              <h3>模块状态</h3>
              <p class="panel-desc">核心模块健康与快捷入口。</p>
            </div>
          </div>
          <div class="module-row">
            <span>CMDB</span>
            <el-tag :type="moduleTagType(moduleStatus.cmdb)">{{ moduleTagText(moduleStatus.cmdb) }}</el-tag>
            <el-button link @click="go('/host')">进入</el-button>
          </div>
          <div class="module-row">
            <span>Docker</span>
            <el-tag :type="moduleTagType(moduleStatus.docker)">{{ moduleTagText(moduleStatus.docker) }}</el-tag>
            <el-button link @click="go('/docker')">进入</el-button>
          </div>
          <div class="module-row">
            <span>K8s</span>
            <el-tag :type="moduleTagType(moduleStatus.k8s)">{{ moduleTagText(moduleStatus.k8s) }}</el-tag>
            <el-button link @click="go('/k8s/clusters')">进入</el-button>
          </div>
          <div class="module-row">
            <span>监控</span>
            <el-tag :type="moduleTagType(moduleStatus.monitor)">{{ moduleTagText(moduleStatus.monitor) }}</el-tag>
            <el-button link @click="go('/monitor/overview')">进入</el-button>
          </div>
          <div class="module-row">
            <span>任务调度</span>
            <el-tag :type="moduleTagType(moduleStatus.task)">{{ moduleTagText(moduleStatus.task) }}</el-tag>
            <el-button link @click="go('/task/schedules')">进入</el-button>
          </div>
          <div class="module-row">
            <span>主机状态时效</span>
            <el-tag :type="stats.hostStale > 0 ? 'warning' : 'success'">
              {{ stats.hostStale > 0 ? `过期 ${stats.hostStale}` : '实时' }}
            </el-tag>
            <el-button link @click="go('/host')">巡检</el-button>
          </div>
          <div class="module-row">
            <span>Docker状态时效</span>
            <el-tag :type="stats.dockerStale > 0 ? 'warning' : 'success'">
              {{ stats.dockerStale > 0 ? `过期 ${stats.dockerStale}` : '实时' }}
            </el-tag>
            <el-button link @click="go('/docker')">巡检</el-button>
          </div>
        </el-card>

        <el-card shadow="never" class="stack-card risk-card">
          <div class="panel-header">
            <div>
              <h3>Deployment 风险</h3>
              <p class="panel-desc">{{ deploymentRisk.clusterName }} / {{ deploymentRisk.namespaceLabel }}</p>
            </div>
            <el-button link type="primary" @click="goDeploymentCenter">进入</el-button>
          </div>
          <div class="risk-kpi-grid">
            <div class="risk-kpi-item">
              <div class="risk-kpi-label">总量</div>
              <div class="risk-kpi-value">{{ deploymentRisk.total }}</div>
            </div>
            <div class="risk-kpi-item">
              <div class="risk-kpi-label">健康</div>
              <div class="risk-kpi-value success">{{ deploymentRisk.healthy }}</div>
            </div>
            <div class="risk-kpi-item">
              <div class="risk-kpi-label">发布中</div>
              <div class="risk-kpi-value warning">{{ deploymentRisk.progressing }}</div>
            </div>
            <div class="risk-kpi-item">
              <div class="risk-kpi-label">异常</div>
              <div class="risk-kpi-value danger">{{ deploymentRisk.degraded }}</div>
            </div>
          </div>
          <div class="risk-extra">
            <span>副本缺口 {{ deploymentRisk.gapReplicas }}</span>
            <span>异常 Pod {{ deploymentRisk.podAbnormal }}</span>
          </div>
          <div class="risk-list" v-if="deploymentRisk.topRisk.length">
            <div class="risk-list-item" v-for="item in deploymentRisk.topRisk" :key="item.key">
              <span class="risk-name">{{ item.name }}</span>
              <el-tag size="small" :type="item.type">{{ item.label }}</el-tag>
            </div>
          </div>
          <div class="panel-desc" v-else>当前范围内无明显风险。</div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="panel-row motion-up delay-6">
      <el-col :span="16">
        <el-card shadow="never">
          <div class="panel-header">
            <div>
              <h3>最近告警</h3>
              <p class="panel-desc">最新 8 条告警事件。</p>
            </div>
            <el-button size="small" @click="go('/alert/events')">查看全部</el-button>
          </div>
          <el-table :fit="true" :data="recentAlerts" size="small" style="width: 100%">
            <el-table-column prop="rule_name" label="规则" min-width="140" />
            <el-table-column prop="target" label="目标" min-width="160" />
            <el-table-column prop="severity" label="级别" width="110" />
            <el-table-column prop="status" label="状态" width="110">
              <template #default="{ row }">
                <el-tag :type="row.status === 0 ? 'danger' : row.status === 1 ? 'success' : 'info'">
                  {{ row.status === 0 ? '未处理' : row.status === 1 ? '已处理' : '已忽略' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="created_at" label="时间" width="180">
              <template #default="{ row }">
                {{ formatTime(row.created_at) }}
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="never">
          <div class="panel-header">
            <div>
              <h3>主机 CPU Top</h3>
              <p class="panel-desc">按当前 CPU 使用率排序。</p>
            </div>
            <el-button size="small" @click="refreshTopHosts">刷新</el-button>
          </div>
          <el-table :fit="true" :data="topHosts" size="small" style="width: 100%">
            <el-table-column prop="instance" label="主机" min-width="160" />
            <el-table-column prop="cpu" label="CPU%" width="90">
              <template #default="{ row }">
                {{ formatNumber(row.cpu, 1) }}
              </template>
            </el-table-column>
            <el-table-column prop="memory" label="内存%" width="90">
              <template #default="{ row }">
                {{ formatNumber(row.memory, 1) }}
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>
  </el-card>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'
import * as echarts from 'echarts'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getErrorMessage, isCancelError } from '@/utils/error'

const router = useRouter()
const loading = ref(false)
const refreshing = ref(false)
const lastUpdated = ref('')
const realtimeRefreshing = ref(false)

const trendRef = ref(null)
let trendChart = null
let realtimeTimer = null
let overviewTimer = null

const realtime = reactive({
  cpu: 0,
  memory: 0,
  disk: 0,
  network: 0
})

const stats = reactive({
  hostTotal: 0,
  hostOnline: 0,
  hostOffline: 0,
  hostStale: 0,
  dockerTotal: 0,
  dockerOnline: 0,
  dockerOffline: 0,
  dockerStale: 0,
  k8sTotal: 0,
  k8sHealthy: 0,
  alertTotal: 0,
  alertOpen: 0,
  taskTotal: 0,
  taskEnabled: 0,
  agentTotal: 0,
  agentOnline: 0
})

const backlog = reactive({
  total: 0,
  overdue: 0,
  asset: 0,
  monitor: 0,
  k8s: 0,
  delivery: 0,
  assetOverdue: 0,
  monitorOverdue: 0,
  k8sOverdue: 0,
  deliveryOverdue: 0
})
const backlogActionLoading = reactive({
  asset: false,
  monitor: false,
  k8s: false,
  delivery: false
})
const backlogSource = reactive({
  offlineHostIds: [],
  offlineNetworkDeviceIds: [],
  staleDomainNames: [],
  staleCertIds: [],
  degradedClusterIds: [],
  staleClusterIds: [],
  longRunningExecutionIds: [],
  longRunningWorkflowIds: [],
  failedTerminalSessionIds: []
})

const moduleStatus = reactive({
  cmdb: 'unknown',
  docker: 'unknown',
  k8s: 'unknown',
  monitor: 'unknown',
  task: 'unknown'
})

const environmentOptions = [
  { label: '全部环境', value: 'all' },
  { label: '生产', value: 'prod' },
  { label: '预发/测试', value: 'staging' },
  { label: '开发', value: 'dev' }
]

const timeWindowOptions = [
  { label: '最近 6 小时', value: 6 },
  { label: '最近 12 小时', value: 12 },
  { label: '最近 24 小时', value: 24 },
  { label: '最近 72 小时', value: 72 }
]

const scope = reactive({
  environment: 'all',
  clusterId: '',
  namespace: '',
  timeWindowHours: 24
})

const scopeClusters = ref([])
const scopeNamespaces = ref([])
const scopeNamespaceLoadedFor = ref('')
const partialFailureNotice = reactive({ key: '', at: 0 })

const deploymentRisk = reactive({
  clusterId: '',
  clusterName: '未选择集群',
  namespaceLabel: '全部命名空间',
  total: 0,
  healthy: 0,
  progressing: 0,
  degraded: 0,
  gapReplicas: 0,
  podAbnormal: 0,
  topRisk: []
})

const recentAlerts = ref([])
const topHosts = ref([])
const trendRecords = ref([])

const backlogCards = computed(() => [
  {
    key: 'asset',
    label: '资产管理中心',
    value: backlog.asset,
    overdue: backlog.assetOverdue,
    desc: '主机/网络/防火墙/堡垒机',
    path: '/host'
  },
  {
    key: 'monitor',
    label: '监控告警中心',
    value: backlog.monitor,
    overdue: backlog.monitorOverdue,
    desc: '告警超时、证书与域名风险',
    path: '/monitor/center'
  },
  {
    key: 'k8s',
    label: '容器平台总览',
    value: backlog.k8s,
    overdue: backlog.k8sOverdue,
    desc: '集群异常与状态时效',
    path: '/k8s/overview'
  },
  {
    key: 'delivery',
    label: '服务管理',
    value: backlog.delivery,
    overdue: backlog.deliveryOverdue,
    desc: '交付、自动化、终端会话',
    path: '/delivery/center'
  }
])

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const go = (path) => router.push(path)

const toArray = (v) => (Array.isArray(v) ? v : [])
const toNumber = (v, fallback = 0) => {
  const n = Number(v)
  return Number.isFinite(n) ? n : fallback
}
const normalizeText = (v) => String(v ?? '').trim().toLowerCase()
const nowMs = () => Date.now()

const inferClusterEnvironment = (cluster) => {
  const raw = `${cluster?.display_name || ''} ${cluster?.name || ''} ${cluster?.description || ''}`
  const text = normalizeText(raw)
  if (text.includes('prod') || text.includes('生产')) return 'prod'
  if (text.includes('staging') || text.includes('stage') || text.includes('测试') || text.includes('预发')) return 'staging'
  if (text.includes('dev') || text.includes('开发')) return 'dev'
  return 'other'
}

const clusterInEnvironment = (cluster) => {
  if (!cluster) return false
  if (scope.environment === 'all') return true
  return inferClusterEnvironment(cluster) === scope.environment
}

const clusterInScope = (cluster) => {
  if (!clusterInEnvironment(cluster)) return false
  if (!scope.clusterId) return true
  return cluster.id === scope.clusterId
}

const parseTimestamp = (value) => {
  if (!value) return null
  const ts = new Date(value).getTime()
  return Number.isNaN(ts) ? null : ts
}

const elapsedMinutes = (value) => {
  const ts = parseTimestamp(value)
  if (!ts) return 0
  const diff = Math.floor((nowMs() - ts) / 60000)
  return diff > 0 ? diff : 0
}

const statusFreshnessTs = (row) =>
  row?.last_check_at ||
  row?.last_seen_at ||
  row?.last_heartbeat_at ||
  row?.updated_at

const isStatusStale = (row, minutes = 3) => {
  const ts = parseTimestamp(statusFreshnessTs(row))
  if (!ts) return true
  return nowMs() - ts > minutes * 60 * 1000
}

const isOnlineStatus = (status) => {
  const normalized = normalizeText(status)
  return normalized === 'online' || normalized === 'ready' || normalized === 'running' || Number(status) === 1
}

const isMaintenanceStatus = (status) => {
  const normalized = normalizeText(status)
  return normalized === 'maintenance' || normalized === 'maintain' || Number(status) === 2
}

const isAlertOpen = (status) => {
  const v = Number(status)
  return v === 0 || v === 1
}

const isWorkorderApprovalPending = (status) => {
  const v = Number(status)
  return v === 0 || v === 1
}

const isScheduleEnabled = (row) => row?.enabled === true || Number(row?.enabled) === 1

const clusterFreshnessTs = (row) =>
  row?.last_sync_at ||
  row?.last_seen_at ||
  row?.last_heartbeat_at ||
  row?.last_checked_at ||
  row?.updated_at

const inTimeWindow = (value) => {
  const ts = parseTimestamp(value)
  if (!ts) return true
  const hours = toNumber(scope.timeWindowHours, 24)
  return ts >= nowMs() - hours * 60 * 60 * 1000
}

const summarizeSettled = (results) => {
  const ok = results.filter((item) => item.status === 'fulfilled').length
  return { ok, fail: results.length - ok }
}

const throttledPartialFailureMessage = (failures) => {
  if (!failures.length) return
  const key = failures.join('|')
  const now = nowMs()
  if (partialFailureNotice.key === key && now - partialFailureNotice.at < 15000) return
  partialFailureNotice.key = key
  partialFailureNotice.at = now
  ElMessage.warning(`部分数据加载失败：${failures.join('、')}`)
}

const ensureScopeNamespaces = async (force = false) => {
  if (!scope.clusterId) {
    scopeNamespaces.value = []
    scope.namespace = ''
    scopeNamespaceLoadedFor.value = ''
    return
  }
  if (!force && scopeNamespaceLoadedFor.value === scope.clusterId && scopeNamespaces.value.length) return
  try {
    const res = await axios.get(`/api/v1/k8s/clusters/${scope.clusterId}/namespaces`, { headers: authHeaders() })
    const list = toArray(res.data?.data).map((item) => item.name).filter(Boolean)
    scopeNamespaces.value = list
    scopeNamespaceLoadedFor.value = scope.clusterId
    if (scope.namespace && !scopeNamespaces.value.includes(scope.namespace)) {
      scope.namespace = ''
    }
  } catch (err) {
    scopeNamespaces.value = []
    scope.namespace = ''
  }
}

const handleScopeEnvironmentChange = async () => {
  if (scope.clusterId) {
    scope.clusterId = ''
    scope.namespace = ''
    scopeNamespaceLoadedFor.value = ''
  }
  await refreshDashboard()
}

const handleScopeClusterChange = async () => {
  scope.namespace = ''
  await ensureScopeNamespaces(true)
  await refreshDashboard()
}

const goDeploymentCenter = () => {
  router.push({
    path: '/k8s/deployments',
    query: {
      clusterId: scope.clusterId || undefined,
      namespace: scope.namespace || undefined
    }
  })
}

const refreshDeploymentRisk = async (scopedClusters, failures) => {
  const targetCluster = scope.clusterId
    ? scopedClusters.find((item) => item.id === scope.clusterId)
    : scopedClusters[0]

  if (!targetCluster) {
    deploymentRisk.clusterId = ''
    deploymentRisk.clusterName = '未选择集群'
    deploymentRisk.namespaceLabel = '全部命名空间'
    deploymentRisk.total = 0
    deploymentRisk.healthy = 0
    deploymentRisk.progressing = 0
    deploymentRisk.degraded = 0
    deploymentRisk.gapReplicas = 0
    deploymentRisk.podAbnormal = 0
    deploymentRisk.topRisk = []
    return
  }

  const namespace = scope.namespace || ''
  deploymentRisk.clusterId = targetCluster.id
  deploymentRisk.clusterName = targetCluster.display_name || targetCluster.name
  deploymentRisk.namespaceLabel = namespace || '全部命名空间'

  try {
    const workloadsRes = await axios.get(`/api/v1/k8s/clusters/${targetCluster.id}/workloads`, {
      headers: authHeaders(),
      params: { namespace }
    })
    const all = toArray(workloadsRes.data?.data)
    const deployments = all.filter((item) => item.kind === 'Deployment')
    deploymentRisk.total = deployments.length
    deploymentRisk.healthy = 0
    deploymentRisk.progressing = 0
    deploymentRisk.degraded = 0
    deploymentRisk.gapReplicas = 0
    const topRisk = []

    deployments.forEach((item) => {
      const replicas = toNumber(item.replicas, 0)
      const ready = toNumber(item.ready, 0)
      const available = toNumber(item.available, 0)
      const gap = Math.max(0, replicas - ready)
      deploymentRisk.gapReplicas += gap
      if (replicas <= 0) {
        topRisk.push({ key: `${item.namespace}/${item.name}`, name: `${item.namespace}/${item.name}`, label: 'Scaled 0', type: 'info' })
        return
      }
      if (ready >= replicas && available >= replicas) {
        deploymentRisk.healthy += 1
        return
      }
      if (ready > 0 || available > 0) {
        deploymentRisk.progressing += 1
        topRisk.push({ key: `${item.namespace}/${item.name}`, name: `${item.namespace}/${item.name}`, label: `${ready}/${replicas}`, type: 'warning' })
        return
      }
      deploymentRisk.degraded += 1
      topRisk.push({ key: `${item.namespace}/${item.name}`, name: `${item.namespace}/${item.name}`, label: '0 Ready', type: 'danger' })
    })

    const candidateNamespaces = namespace
      ? [namespace]
      : Array.from(new Set(deployments.map((item) => item.namespace).filter(Boolean))).slice(0, 6)
    const podCalls = await Promise.allSettled(
      candidateNamespaces.map((ns) => axios.get(`/api/v1/k8s/clusters/${targetCluster.id}/namespaces/${ns}/pods`, { headers: authHeaders() }))
    )
    const abnormal = []
    podCalls.forEach((item) => {
      if (item.status !== 'fulfilled') return
      const rows = toArray(item.value.data?.data)
      rows.forEach((pod) => {
        const status = normalizeText(pod.status)
        if (pod.owner_kind !== 'Deployment') return
        if (status === 'running' || status === 'succeeded' || status === 'completed') return
        abnormal.push(pod)
      })
    })
    deploymentRisk.podAbnormal = abnormal.length
    deploymentRisk.topRisk = topRisk.slice(0, 6)
  } catch (err) {
    deploymentRisk.total = 0
    deploymentRisk.healthy = 0
    deploymentRisk.progressing = 0
    deploymentRisk.degraded = 0
    deploymentRisk.gapReplicas = 0
    deploymentRisk.podAbnormal = 0
    deploymentRisk.topRisk = []
    failures.push(getErrorMessage(err, 'Deployment 风险'))
  }
}

const runBacklogAction = async (key) => {
  if (!Object.prototype.hasOwnProperty.call(backlogActionLoading, key)) return
  if (backlogActionLoading[key]) return
  backlogActionLoading[key] = true
  try {
    if (key === 'asset') {
      const hostIds = backlogSource.offlineHostIds.slice(0, 3)
      const networkIds = backlogSource.offlineNetworkDeviceIds.slice(0, 3)
      await ElMessageBox.confirm(
        `将同步防火墙资产，并巡检离线主机(${hostIds.length})/网络设备(${networkIds.length})，确认执行吗？`,
        '资产一键处置',
        { type: 'warning' }
      )
      const jobs = [
        axios.post('/api/v1/cmdb/network-devices/sync/firewalls', {}, { headers: authHeaders() }),
        ...hostIds.map((id) => axios.post(`/api/v1/cmdb/hosts/${id}/test`, {}, { headers: authHeaders() })),
        ...networkIds.map((id) => axios.post(`/api/v1/cmdb/network-devices/${id}/test`, {}, { headers: authHeaders() }))
      ]
      const summary = summarizeSettled(await Promise.allSettled(jobs))
      ElMessage.success(`资产处置完成：成功 ${summary.ok}，失败 ${summary.fail}`)
    } else if (key === 'monitor') {
      const domains = backlogSource.staleDomainNames.slice(0, 5)
      const certs = backlogSource.staleCertIds.slice(0, 5)
      if (!domains.length && !certs.length) {
        ElMessage.info('当前没有待复检的域名/证书项')
        return
      }
      await ElMessageBox.confirm(
        `将复检域名(${domains.length})和证书(${certs.length})，确认执行吗？`,
        '监控中心一键处置',
        { type: 'warning' }
      )
      const jobs = [
        ...domains.map((domain) => axios.post('/api/v1/domain/domains/check', { domain }, { headers: authHeaders() })),
        ...certs.map((id) => axios.post(`/api/v1/domain/certs/${id}/check`, {}, { headers: authHeaders() }))
      ]
      const summary = summarizeSettled(await Promise.allSettled(jobs))
      ElMessage.success(`监控复检完成：成功 ${summary.ok}，失败 ${summary.fail}`)
    } else if (key === 'k8s') {
      const clusterIds = [...new Set([...backlogSource.degradedClusterIds, ...backlogSource.staleClusterIds])].slice(0, 5)
      if (!clusterIds.length) {
        ElMessage.info('当前没有需要巡检的集群')
        return
      }
      await ElMessageBox.confirm(
        `将对 ${clusterIds.length} 个异常/超时集群执行连接测试，确认执行吗？`,
        'K8s一键处置',
        { type: 'warning' }
      )
      const jobs = clusterIds.map((id) => axios.post(`/api/v1/k8s/clusters/${id}/test`, {}, { headers: authHeaders() }))
      const summary = summarizeSettled(await Promise.allSettled(jobs))
      ElMessage.success(`K8s巡检完成：成功 ${summary.ok}，失败 ${summary.fail}`)
    } else if (key === 'delivery') {
      const executionIds = backlogSource.longRunningExecutionIds.slice(0, 3)
      const workflowIds = backlogSource.longRunningWorkflowIds.slice(0, 3)
      const failedSessions = backlogSource.failedTerminalSessionIds.slice(0, 10)
      if (!executionIds.length && !workflowIds.length && !failedSessions.length) {
        ElMessage.info('当前没有可自动处置的服务管理积压')
        return
      }
      await ElMessageBox.confirm(
        `将取消超时执行(${executionIds.length})、超时流程(${workflowIds.length})，并清理失败会话(${failedSessions.length})，确认执行吗？`,
        '服务管理一键处置',
        { type: 'warning' }
      )
      const jobs = [
        ...executionIds.map((id) => axios.post(`/api/v1/cicd/executions/${id}/cancel`, {}, { headers: authHeaders() })),
        ...workflowIds.map((id) => axios.post(`/api/v1/workflow/executions/${id}/cancel`, {}, { headers: authHeaders() })),
        ...failedSessions.map((id) => axios.delete(`/api/v1/terminal/sessions/${id}/purge`, { headers: authHeaders() }))
      ]
      const summary = summarizeSettled(await Promise.allSettled(jobs))
      ElMessage.success(`服务管理处置完成：成功 ${summary.ok}，失败 ${summary.fail}`)
    }
    await refreshDashboard()
  } catch (err) {
    if (isCancelError(err)) return
    ElMessage.error(getErrorMessage(err, '执行一键处置失败'))
  } finally {
    backlogActionLoading[key] = false
  }
}

const safePercent = (v) => Math.max(0, Math.min(100, toNumber(v)))
const formatPercent = (v) => `${toNumber(v).toFixed(1)}%`
const formatNumber = (v, digits = 2) => toNumber(v).toFixed(digits)
const formatTime = (val) => {
  if (!val) return '-'
  const t = new Date(val)
  if (Number.isNaN(t.getTime())) return '-'
  return t.toLocaleString()
}

const moduleTagType = (status) => {
  if (status === 'ok') return 'success'
  if (status === 'warning') return 'warning'
  if (status === 'error') return 'danger'
  return 'info'
}

const moduleTagText = (status) => {
  if (status === 'ok') return '正常'
  if (status === 'warning') return '预警'
  if (status === 'error') return '异常'
  return '未知'
}

const extractData = (result) => {
  if (result.status !== 'fulfilled') return null
  const body = result.value?.data
  if (!body || typeof body !== 'object') return null
  if (Object.prototype.hasOwnProperty.call(body, 'code')) {
    return body.code === 0 ? body.data : null
  }
  return body.data ?? null
}

const extractFailure = (result, label) => {
  if (result.status !== 'rejected') return ''
  return `${label}(${getErrorMessage(result.reason, '请求失败')})`
}

const renderTrend = () => {
  const dom = trendRef.value
  if (!(dom instanceof HTMLDivElement)) return
  if (!trendChart) {
    trendChart = echarts.init(dom)
  }

  const records = toArray(trendRecords.value)
  if (!records.length) {
    trendChart.setOption({
      title: {
        text: '暂无历史数据',
        left: 'center',
        top: 'middle',
        textStyle: { color: '#9ca3af', fontSize: 14, fontWeight: 500 }
      },
      xAxis: { show: false, type: 'category', data: [] },
      yAxis: { show: false, type: 'value' },
      series: []
    })
    return
  }

  trendChart.setOption({
    color: ['#3b82f6', '#10b981', '#f59e0b'],
    tooltip: { trigger: 'axis' },
    legend: { top: 0, data: ['CPU', '内存', '磁盘'] },
    grid: { left: 40, right: 20, top: 36, bottom: 24 },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: records.map((item) => formatTime(item.timestamp))
    },
    yAxis: {
      type: 'value',
      axisLabel: { formatter: '{value}%' }
    },
    series: [
      {
        name: 'CPU',
        type: 'line',
        smooth: true,
        showSymbol: false,
        areaStyle: { opacity: 0.12 },
        data: records.map((item) => toNumber(item.cpu_usage))
      },
      {
        name: '内存',
        type: 'line',
        smooth: true,
        showSymbol: false,
        areaStyle: { opacity: 0.1 },
        data: records.map((item) => toNumber(item.memory_usage))
      },
      {
        name: '磁盘',
        type: 'line',
        smooth: true,
        showSymbol: false,
        areaStyle: { opacity: 0.08 },
        data: records.map((item) => toNumber(item.disk_usage))
      }
    ]
  })
}

const refreshTopHosts = async () => {
  try {
    const res = await axios.get('/api/v1/monitor/servers', { headers: authHeaders() })
    const list = toArray(res.data?.data)
    topHosts.value = list
      .map((item) => ({
        instance: item.instance || item.ip || item.hostname || '-',
        cpu: toNumber(item.cpu || item.cpu_usage),
        memory: toNumber(item.memory || item.memory_usage)
      }))
      .sort((a, b) => b.cpu - a.cpu)
      .slice(0, 8)
  } catch (e) {
    topHosts.value = []
    ElMessage.error(getErrorMessage(e, '加载主机排行失败'))
  }
}

const refreshRealtimeMetrics = async () => {
  if (realtimeRefreshing.value) return
  realtimeRefreshing.value = true
  try {
    const res = await axios.get('/api/v1/monitor/metrics', { headers: authHeaders() })
    const payload = res.data?.code === 0 ? res.data.data : null
    if (payload && typeof payload === 'object') {
      realtime.cpu = toNumber(payload.cpu)
      realtime.memory = toNumber(payload.memory)
      realtime.disk = toNumber(payload.disk)
      realtime.network = toNumber(payload.network)
      moduleStatus.monitor = 'ok'
    }
  } catch (e) {
    moduleStatus.monitor = 'error'
  } finally {
    realtimeRefreshing.value = false
  }
}

const refreshDashboard = async () => {
  if (refreshing.value) return
  refreshing.value = true
  try {
    const calls = await Promise.allSettled([
      axios.get('/api/v1/cmdb/hosts', { headers: authHeaders(), params: { live: 1 } }),
      axios.get('/api/v1/docker/hosts', { headers: authHeaders(), params: { sync: 1 } }),
      axios.get('/api/v1/k8s/clusters', { headers: authHeaders() }),
      axios.get('/api/v1/monitor/alerts', { headers: authHeaders() }),
      axios.get('/api/v1/task/tasks', { headers: authHeaders() }),
      axios.get('/api/v1/monitor/agents', { headers: authHeaders() }),
      axios.get('/api/v1/monitor/metrics', { headers: authHeaders() }),
      axios.get('/api/v1/monitor/metrics/history', { headers: authHeaders(), params: { hours: Number(scope.timeWindowHours || 24) } }),
      axios.get('/api/v1/cmdb/network-devices', { headers: authHeaders() }),
      axios.get('/api/v1/firewall/devices', { headers: authHeaders() }),
      axios.get('/api/v1/jump/sessions', { headers: authHeaders() }),
      axios.get('/api/v1/jump/risk-events', { headers: authHeaders() }),
      axios.get('/api/v1/domain/domains', { headers: authHeaders() }),
      axios.get('/api/v1/domain/certs', { headers: authHeaders() }),
      axios.get('/api/v1/cicd/executions', { headers: authHeaders() }),
      axios.get('/api/v1/cicd/schedules', { headers: authHeaders() }),
      axios.get('/api/v1/workorder/orders', { headers: authHeaders() }),
      axios.get('/api/v1/workflow/executions', { headers: authHeaders() }),
      axios.get('/api/v1/terminal/sessions', { headers: authHeaders() })
    ])

    const failures = []

    const hostsPayload = extractData(calls[0])
    const hosts = toArray(hostsPayload)
    if (hostsPayload !== null) {
      stats.hostTotal = hosts.length
      stats.hostOnline = hosts.filter((h) => toNumber(h.status, -1) === 1 || normalizeText(h.status) === 'online').length
      stats.hostOffline = hosts.filter((h) => !isOnlineStatus(h.status)).length
      stats.hostStale = hosts.filter((h) => isStatusStale(h, 3)).length
      moduleStatus.cmdb = stats.hostStale > 0 ? 'warning' : 'ok'
    } else {
      stats.hostOffline = 0
      stats.hostStale = 0
      moduleStatus.cmdb = 'error'
      failures.push(extractFailure(calls[0], 'CMDB') || 'CMDB')
    }

    const dockerPayload = extractData(calls[1])
    const dockerHosts = toArray(dockerPayload)
    if (dockerPayload !== null) {
      stats.dockerTotal = dockerHosts.length
      stats.dockerOnline = dockerHosts.filter((h) => normalizeText(h.status) === 'online').length
      stats.dockerOffline = dockerHosts.filter((h) => normalizeText(h.status) !== 'online').length
      stats.dockerStale = dockerHosts.filter((h) => isStatusStale(h, 3)).length
      moduleStatus.docker = stats.dockerStale > 0 ? 'warning' : 'ok'
    } else {
      stats.dockerOffline = 0
      stats.dockerStale = 0
      moduleStatus.docker = 'error'
      failures.push(extractFailure(calls[1], 'Docker') || 'Docker')
    }

    const clustersPayload = extractData(calls[2])
    const allClusters = toArray(clustersPayload)
    const envClusters = allClusters.filter((item) => clusterInEnvironment(item))
    scopeClusters.value = envClusters
    if (scope.clusterId && !envClusters.some((item) => item.id === scope.clusterId)) {
      scope.clusterId = ''
      scope.namespace = ''
      scopeNamespaceLoadedFor.value = ''
    }
    if (!scope.clusterId && envClusters.length === 1) {
      scope.clusterId = envClusters[0].id
    }
    await ensureScopeNamespaces()

    const scopedClusters = allClusters.filter((item) => clusterInScope(item))
    if (clustersPayload !== null) {
      stats.k8sTotal = scopedClusters.length
      stats.k8sHealthy = scopedClusters.filter((c) => toNumber(c.status, -1) === 1 || normalizeText(c.status) === 'online').length
      moduleStatus.k8s = 'ok'
    } else {
      moduleStatus.k8s = 'error'
      failures.push(extractFailure(calls[2], 'K8s') || 'K8s')
    }

    const alertsPayload = extractData(calls[3])
    const alerts = toArray(alertsPayload)
    const scopedAlerts = alerts.filter((item) => inTimeWindow(item.fired_at || item.created_at))
    if (alertsPayload !== null) {
      stats.alertTotal = scopedAlerts.length
      stats.alertOpen = scopedAlerts.filter((a) => toNumber(a.status, -1) === 0).length
      recentAlerts.value = scopedAlerts.slice(0, 8)
    } else {
      failures.push(extractFailure(calls[3], '告警') || '告警')
    }

    const tasksPayload = extractData(calls[4])
    const tasks = toArray(tasksPayload)
    if (tasksPayload !== null) {
      stats.taskTotal = tasks.length
      stats.taskEnabled = tasks.filter((t) => Boolean(t.enabled)).length
      moduleStatus.task = 'ok'
    } else {
      moduleStatus.task = 'error'
      failures.push(extractFailure(calls[4], '任务') || '任务')
    }

    const agentsPayload = extractData(calls[5])
    const agents = toArray(agentsPayload)
    if (agentsPayload !== null) {
      stats.agentTotal = agents.length
      stats.agentOnline = agents.filter((a) => normalizeText(a.status) === 'online').length
    } else {
      failures.push(extractFailure(calls[5], 'Agent') || 'Agent')
    }

    const metricData = extractData(calls[6])
    if (metricData !== null) {
      realtime.cpu = toNumber(metricData.cpu)
      realtime.memory = toNumber(metricData.memory)
      realtime.disk = toNumber(metricData.disk)
      realtime.network = toNumber(metricData.network)
      moduleStatus.monitor = 'ok'
    } else {
      moduleStatus.monitor = 'error'
      failures.push(extractFailure(calls[6], '监控') || '监控')
    }

    const historyPayload = extractData(calls[7])
    const history = toArray(historyPayload).filter((item) => inTimeWindow(item.timestamp))
    if (historyPayload === null) {
      failures.push(extractFailure(calls[7], '趋势') || '趋势')
    }
    trendRecords.value = history
      .map((item) => ({
        timestamp: item.timestamp,
        cpu_usage: toNumber(item.cpu_usage),
        memory_usage: toNumber(item.memory_usage),
        disk_usage: toNumber(item.disk_usage)
      }))
      .sort((a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime())

    const networkDevices = toArray(extractData(calls[8]))
    const firewalls = toArray(extractData(calls[9]))
    const jumpSessions = toArray(extractData(calls[10]))
    const jumpRiskEvents = toArray(extractData(calls[11]))
    const domains = toArray(extractData(calls[12]))
    const certs = toArray(extractData(calls[13]))
    const deliveryExecutions = toArray(extractData(calls[14]))
    const deliverySchedules = toArray(extractData(calls[15]))
    const workorders = toArray(extractData(calls[16]))
    const workflowExecutions = toArray(extractData(calls[17]))
    const terminalSessions = toArray(extractData(calls[18]))

    const offlineHosts = hosts.filter((item) => !isOnlineStatus(item.status))
    const offlineNetworks = networkDevices.filter((item) => !isOnlineStatus(item.status))
    const hostOffline = offlineHosts.length
    const networkOffline = offlineNetworks.length
    const firewallAlert = firewalls.filter((item) => Number(item.status) === 2).length
    const jumpPendingTimeout = jumpSessions.filter(
      (item) => normalizeText(item.status) === 'pending_approval' && elapsedMinutes(item.started_at) >= 30
    ).length
    const riskCritical = jumpRiskEvents.filter((item) => normalizeText(item.severity) === 'critical').length
    backlog.asset = hostOffline + networkOffline + firewallAlert + jumpPendingTimeout + riskCritical
    backlog.assetOverdue = jumpPendingTimeout

    const monitorAlertTimeout = scopedAlerts.filter(
      (item) => isAlertOpen(item.status) && elapsedMinutes(item.fired_at || item.created_at) >= 60
    ).length
    const monitorCriticalOpen = scopedAlerts.filter(
      (item) => isAlertOpen(item.status) && normalizeText(item.severity) === 'critical'
    ).length
    const domainRiskRows = []
    const staleDomainNames = []
    const staleCertIds = []
    domains.forEach((item) => {
      const health = normalizeText(item.health_status)
      if (health === 'warning' || health === 'critical') {
        const checkedAt = item.last_checked_at || item.updated_at
        domainRiskRows.push({ checkedAt })
        const ts = parseTimestamp(checkedAt)
        if (!ts || nowMs() - ts > 24 * 60 * 60 * 1000) {
          if (item.domain) staleDomainNames.push(item.domain)
        }
      }
    })
    certs.forEach((item) => {
      if (toNumber(item.days_to_expire, 0) <= 30) {
        const checkedAt = item.last_check_at || item.updated_at
        domainRiskRows.push({ checkedAt })
        const ts = parseTimestamp(checkedAt)
        if (!ts || nowMs() - ts > 24 * 60 * 60 * 1000) {
          if (item.id) staleCertIds.push(item.id)
        }
      }
    })
    const monitorRiskStale = domainRiskRows.filter((item) => {
      const ts = parseTimestamp(item.checkedAt)
      return !ts || nowMs() - ts > 24 * 60 * 60 * 1000
    }).length
    backlog.monitor = monitorAlertTimeout + monitorCriticalOpen + monitorRiskStale
    backlog.monitorOverdue = monitorAlertTimeout + monitorRiskStale

    const degradedClusters = scopedClusters.filter(
      (item) => !isOnlineStatus(item.status) && !isMaintenanceStatus(item.status)
    )
    const staleClusters = scopedClusters.filter(
      (item) => isOnlineStatus(item.status) && elapsedMinutes(clusterFreshnessTs(item)) >= 15
    )
    const k8sClusterDegraded = degradedClusters.length
    const k8sClusterStale = staleClusters.length
    backlog.k8s = k8sClusterDegraded + k8sClusterStale
    backlog.k8sOverdue = k8sClusterStale

    const deliveryOrderTimeout = workorders.filter(
      (item) => isWorkorderApprovalPending(item.status) && elapsedMinutes(item.created_at) >= 120
    ).length
    const deliveryLongExecutions = deliveryExecutions.filter((item) => Number(item.status) === 0 && elapsedMinutes(item.started_at) >= 30)
    const deliveryExecutionLong = deliveryLongExecutions.length
    const deliveryScheduleStale = deliverySchedules.filter((item) => {
      if (!isScheduleEnabled(item)) return false
      const nextTs = parseTimestamp(item.next_run_at)
      if (!nextTs) return true
      return nextTs < nowMs() - 5 * 60 * 1000
    }).length
    backlog.delivery = deliveryOrderTimeout + deliveryExecutionLong + deliveryScheduleStale

    const collabOrderTimeout = workorders.filter((item) => {
      const status = Number(item.status)
      return (status === 0 || status === 1 || status === 4) && elapsedMinutes(item.created_at) >= 120
    }).length
    const longWorkflowExecutions = workflowExecutions.filter(
      (item) => Number(item.status) === 0 && elapsedMinutes(item.started_at) >= 15
    )
    const collabWorkflowLong = longWorkflowExecutions.length
    const collabTerminalPending = terminalSessions.filter((item) => {
      if (Number(item.status) !== 0) return false
      return elapsedMinutes(item.started_at || item.created_at) >= 10
    }).length
    const failedTerminalSessions = terminalSessions.filter((item) => Number(item.status) === 3)
    const collabTerminalFailed = failedTerminalSessions.length
    backlog.delivery += collabOrderTimeout + collabWorkflowLong + collabTerminalPending + collabTerminalFailed
    backlog.deliveryOverdue = deliveryOrderTimeout + deliveryExecutionLong + deliveryScheduleStale + collabOrderTimeout + collabWorkflowLong + collabTerminalPending
    backlog.total = backlog.asset + backlog.monitor + backlog.k8s + backlog.delivery
    backlog.overdue =
      backlog.assetOverdue +
      backlog.monitorOverdue +
      backlog.k8sOverdue +
      backlog.deliveryOverdue

    backlogSource.offlineHostIds = offlineHosts.map((item) => item.id).filter(Boolean)
    backlogSource.offlineNetworkDeviceIds = offlineNetworks.map((item) => item.id).filter(Boolean)
    backlogSource.staleDomainNames = staleDomainNames
    backlogSource.staleCertIds = staleCertIds
    backlogSource.degradedClusterIds = degradedClusters.map((item) => item.id).filter(Boolean)
    backlogSource.staleClusterIds = staleClusters.map((item) => item.id).filter(Boolean)
    backlogSource.longRunningExecutionIds = deliveryLongExecutions
      .filter((item) => elapsedMinutes(item.started_at) >= 120)
      .map((item) => item.id)
      .filter(Boolean)
    backlogSource.longRunningWorkflowIds = longWorkflowExecutions
      .filter((item) => elapsedMinutes(item.started_at) >= 120)
      .map((item) => item.id)
      .filter(Boolean)
    backlogSource.failedTerminalSessionIds = failedTerminalSessions.map((item) => item.id).filter(Boolean)

    await refreshDeploymentRisk(scopedClusters, failures)

    lastUpdated.value = new Date().toLocaleString()
    renderTrend()
    await refreshTopHosts()
    throttledPartialFailureMessage(failures)
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '仪表盘刷新失败'))
  } finally {
    refreshing.value = false
  }
}

const onResize = () => {
  if (trendChart) trendChart.resize()
}

const startAutoRefresh = () => {
  if (!realtimeTimer) {
    realtimeTimer = setInterval(() => {
      if (document.hidden) return
      refreshRealtimeMetrics()
    }, 10000)
  }
  if (!overviewTimer) {
    overviewTimer = setInterval(() => {
      if (document.hidden) return
      refreshDashboard()
    }, 60000)
  }
}

const stopAutoRefresh = () => {
  if (realtimeTimer) {
    clearInterval(realtimeTimer)
    realtimeTimer = null
  }
  if (overviewTimer) {
    clearInterval(overviewTimer)
    overviewTimer = null
  }
}

onMounted(async () => {
  loading.value = true
  await refreshDashboard()
  loading.value = false
  startAutoRefresh()
  window.addEventListener('resize', onResize)
})

onBeforeUnmount(() => {
  stopAutoRefresh()
  window.removeEventListener('resize', onResize)
  if (trendChart) {
    trendChart.dispose()
    trendChart = null
  }
})
</script>

<style scoped>
.dashboard-card {
  max-width: 1280px;
  margin: 0 auto;
  border-radius: 18px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 14px;
}

.page-header h2 {
  margin: 0;
  font-size: 26px;
  font-weight: 600;
  letter-spacing: -0.2px;
}

.page-desc {
  margin: 6px 0 0;
  color: #6b7280;
}

.page-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.updated-at {
  color: #9ca3af;
  font-size: 12px;
}

.scope-bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  background: linear-gradient(180deg, #ffffff 0%, #f8fafc 100%);
}

.scope-label {
  color: #4b5563;
  font-size: 12px;
  font-weight: 600;
}

.scope-item {
  width: 180px;
}

.scope-item.narrow {
  width: 150px;
}

.kpi-card {
  border-radius: 14px;
  background: linear-gradient(180deg, #ffffff 0%, #f8fafc 100%);
  border: 1px solid #edf2f7;
}

.kpi-title {
  color: #6b7280;
  font-size: 12px;
}

.kpi-value {
  margin-top: 8px;
  font-size: 30px;
  font-weight: 600;
  line-height: 1;
}

.kpi-value.danger {
  color: #ef4444;
}

.kpi-sub {
  margin-top: 8px;
  font-size: 12px;
  color: #9ca3af;
}

.panel-row {
  margin-top: 16px;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}

.panel-header h3 {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  letter-spacing: -0.2px;
}

.panel-desc {
  margin: 4px 0 0;
  color: #9ca3af;
  font-size: 12px;
}

.chart-box {
  width: 100%;
  height: 320px;
}

.stack-card {
  margin-bottom: 12px;
}

.backlog-overview-card {
  border-radius: 14px;
  border: 1px solid #e5e7eb;
  background: linear-gradient(180deg, #ffffff 0%, #f8fafc 100%);
}

.backlog-header {
  margin-bottom: 14px;
}

.backlog-totals {
  display: flex;
  align-items: center;
  gap: 8px;
}

.backlog-grid {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 10px;
}

.backlog-item {
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  padding: 12px;
  background: #fff;
}

.backlog-item-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.backlog-label {
  font-size: 13px;
  color: #4b5563;
}

.backlog-value {
  margin-top: 10px;
  font-size: 26px;
  font-weight: 600;
  line-height: 1;
  color: #111827;
}

.backlog-desc {
  margin-top: 6px;
  font-size: 12px;
  color: #9ca3af;
}

.backlog-actions {
  margin-top: 8px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.resource-row {
  display: grid;
  grid-template-columns: 48px 1fr 58px;
  gap: 8px;
  align-items: center;
  margin-bottom: 10px;
  color: #4b5563;
}

.resource-row b {
  text-align: right;
  color: #111827;
}

.network-value {
  grid-column: 2 / 4;
  color: #111827;
  font-weight: 600;
}

.module-row {
  display: grid;
  grid-template-columns: 1fr auto auto;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.risk-card {
  border: 1px solid #e5e7eb;
}

.risk-kpi-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 8px;
  margin-top: 4px;
}

.risk-kpi-item {
  padding: 8px 10px;
  border: 1px solid #e5e7eb;
  border-radius: 10px;
  background: #fff;
}

.risk-kpi-label {
  font-size: 12px;
  color: #6b7280;
}

.risk-kpi-value {
  margin-top: 4px;
  font-size: 18px;
  font-weight: 600;
  color: #111827;
}

.risk-kpi-value.success { color: #16a34a; }
.risk-kpi-value.warning { color: #d97706; }
.risk-kpi-value.danger { color: #dc2626; }

.risk-extra {
  margin-top: 8px;
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 12px;
  color: #6b7280;
}

.risk-list {
  margin-top: 10px;
  border-top: 1px dashed #e5e7eb;
  padding-top: 10px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.risk-list-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.risk-name {
  font-size: 12px;
  color: #374151;
}

.motion-up {
  animation: motion-up 0.38s ease both;
}

.delay-1 { animation-delay: 0.03s; }
.delay-2 { animation-delay: 0.06s; }
.delay-3 { animation-delay: 0.09s; }
.delay-4 { animation-delay: 0.12s; }
.delay-5 { animation-delay: 0.15s; }
.delay-6 { animation-delay: 0.18s; }

@keyframes motion-up {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@media (max-width: 1400px) {
  .backlog-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (max-width: 980px) {
  .backlog-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .scope-item {
    width: 100%;
  }

  .risk-kpi-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 760px) {
  .backlog-grid {
    grid-template-columns: 1fr;
  }
}
</style>
