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

    <el-row :gutter="16" class="motion-up delay-2">
      <el-col :span="4">
        <el-card class="kpi-card" shadow="never">
          <div class="kpi-title">主机总数</div>
          <div class="kpi-value">{{ stats.hostTotal }}</div>
          <div class="kpi-sub">在线 {{ stats.hostOnline }}</div>
        </el-card>
      </el-col>
      <el-col :span="4">
        <el-card class="kpi-card" shadow="never">
          <div class="kpi-title">Docker 环境</div>
          <div class="kpi-value">{{ stats.dockerTotal }}</div>
          <div class="kpi-sub">在线 {{ stats.dockerOnline }}</div>
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

    <el-row :gutter="16" class="panel-row motion-up delay-3">
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
              <el-button link type="primary" @click="go(item.path)">进入处置</el-button>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="panel-row motion-up delay-4">
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
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="panel-row motion-up delay-5">
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
import { ElMessage } from 'element-plus'
import { getErrorMessage } from '@/utils/error'

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
  dockerTotal: 0,
  dockerOnline: 0,
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
  collab: 0,
  assetOverdue: 0,
  monitorOverdue: 0,
  k8sOverdue: 0,
  deliveryOverdue: 0,
  collabOverdue: 0
})

const moduleStatus = reactive({
  cmdb: 'unknown',
  docker: 'unknown',
  k8s: 'unknown',
  monitor: 'unknown',
  task: 'unknown'
})

const recentAlerts = ref([])
const topHosts = ref([])
const trendRecords = ref([])

const backlogCards = computed(() => [
  {
    key: 'asset',
    label: '资产作战台',
    value: backlog.asset,
    overdue: backlog.assetOverdue,
    desc: '主机/网络/防火墙/堡垒机',
    path: '/asset/ops'
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
    label: '交付中心',
    value: backlog.delivery,
    overdue: backlog.deliveryOverdue,
    desc: '审批、执行、调度时效',
    path: '/delivery/center'
  },
  {
    key: 'collab',
    label: '协同中心',
    value: backlog.collab,
    overdue: backlog.collabOverdue,
    desc: '工单、流程、终端会话',
    path: '/collab/center'
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

const parseTimestamp = (value) => {
  if (!value) return null
  const ts = new Date(value).getTime()
  return Number.isNaN(ts) ? null : ts
}

const elapsedMinutes = (value) => {
  const ts = parseTimestamp(value)
  if (!ts) return 0
  const diff = Math.floor((Date.now() - ts) / 60000)
  return diff > 0 ? diff : 0
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
  if (status === 'error') return 'danger'
  return 'info'
}

const moduleTagText = (status) => {
  if (status === 'ok') return '正常'
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

  const calls = await Promise.allSettled([
    axios.get('/api/v1/cmdb/hosts', { headers: authHeaders() }),
    axios.get('/api/v1/docker/hosts', { headers: authHeaders() }),
    axios.get('/api/v1/k8s/clusters', { headers: authHeaders() }),
    axios.get('/api/v1/monitor/alerts', { headers: authHeaders() }),
    axios.get('/api/v1/task/tasks', { headers: authHeaders() }),
    axios.get('/api/v1/monitor/agents', { headers: authHeaders() }),
    axios.get('/api/v1/monitor/metrics', { headers: authHeaders() }),
    axios.get('/api/v1/monitor/metrics/history', { headers: authHeaders(), params: { hours: 24 } }),
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
    stats.hostOnline = hosts.filter((h) => toNumber(h.status, -1) === 1 || String(h.status).toLowerCase() === 'online').length
    moduleStatus.cmdb = 'ok'
  } else {
    moduleStatus.cmdb = 'error'
    failures.push(extractFailure(calls[0], 'CMDB') || 'CMDB')
  }

  const dockerPayload = extractData(calls[1])
  const dockerHosts = toArray(dockerPayload)
  if (dockerPayload !== null) {
    stats.dockerTotal = dockerHosts.length
    stats.dockerOnline = dockerHosts.filter((h) => String(h.status).toLowerCase() === 'online').length
    moduleStatus.docker = 'ok'
  } else {
    moduleStatus.docker = 'error'
    failures.push(extractFailure(calls[1], 'Docker') || 'Docker')
  }

  const clustersPayload = extractData(calls[2])
  const clusters = toArray(clustersPayload)
  if (clustersPayload !== null) {
    stats.k8sTotal = clusters.length
    stats.k8sHealthy = clusters.filter((c) => toNumber(c.status, -1) === 1 || String(c.status).toLowerCase() === 'online').length
    moduleStatus.k8s = 'ok'
  } else {
    moduleStatus.k8s = 'error'
    failures.push(extractFailure(calls[2], 'K8s') || 'K8s')
  }

  const alertsPayload = extractData(calls[3])
  const alerts = toArray(alertsPayload)
  if (alertsPayload !== null) {
    stats.alertTotal = alerts.length
    stats.alertOpen = alerts.filter((a) => toNumber(a.status, -1) === 0).length
    recentAlerts.value = alerts.slice(0, 8)
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
    stats.agentOnline = agents.filter((a) => String(a.status).toLowerCase() === 'online').length
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
  const history = toArray(historyPayload)
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

  const hostOffline = hosts.filter((item) => !isOnlineStatus(item.status)).length
  const networkOffline = networkDevices.filter((item) => !isOnlineStatus(item.status)).length
  const firewallAlert = firewalls.filter((item) => Number(item.status) === 2).length
  const jumpPendingTimeout = jumpSessions.filter(
    (item) => normalizeText(item.status) === 'pending_approval' && elapsedMinutes(item.started_at) >= 30
  ).length
  const riskCritical = jumpRiskEvents.filter((item) => normalizeText(item.severity) === 'critical').length
  backlog.asset = hostOffline + networkOffline + firewallAlert + jumpPendingTimeout + riskCritical
  backlog.assetOverdue = jumpPendingTimeout

  const monitorAlertTimeout = alerts.filter(
    (item) => isAlertOpen(item.status) && elapsedMinutes(item.fired_at || item.created_at) >= 60
  ).length
  const monitorCriticalOpen = alerts.filter(
    (item) => isAlertOpen(item.status) && normalizeText(item.severity) === 'critical'
  ).length
  const domainRiskRows = []
  domains.forEach((item) => {
    const health = normalizeText(item.health_status)
    if (health === 'warning' || health === 'critical') {
      domainRiskRows.push({ checkedAt: item.last_checked_at || item.updated_at })
    }
  })
  certs.forEach((item) => {
    if (toNumber(item.days_to_expire, 0) <= 30) {
      domainRiskRows.push({ checkedAt: item.last_check_at || item.updated_at })
    }
  })
  const monitorRiskStale = domainRiskRows.filter((item) => {
    const ts = parseTimestamp(item.checkedAt)
    return !ts || Date.now() - ts > 24 * 60 * 60 * 1000
  }).length
  backlog.monitor = monitorAlertTimeout + monitorCriticalOpen + monitorRiskStale
  backlog.monitorOverdue = monitorAlertTimeout + monitorRiskStale

  const k8sClusterDegraded = clusters.filter(
    (item) => !isOnlineStatus(item.status) && !isMaintenanceStatus(item.status)
  ).length
  const k8sClusterStale = clusters.filter(
    (item) => isOnlineStatus(item.status) && elapsedMinutes(clusterFreshnessTs(item)) >= 15
  ).length
  backlog.k8s = k8sClusterDegraded + k8sClusterStale
  backlog.k8sOverdue = k8sClusterStale

  const deliveryOrderTimeout = workorders.filter(
    (item) => isWorkorderApprovalPending(item.status) && elapsedMinutes(item.created_at) >= 120
  ).length
  const deliveryExecutionLong = deliveryExecutions.filter(
    (item) => Number(item.status) === 0 && elapsedMinutes(item.started_at) >= 30
  ).length
  const deliveryScheduleStale = deliverySchedules.filter((item) => {
    if (!isScheduleEnabled(item)) return false
    const nextTs = parseTimestamp(item.next_run_at)
    if (!nextTs) return true
    return nextTs < Date.now() - 5 * 60 * 1000
  }).length
  backlog.delivery = deliveryOrderTimeout + deliveryExecutionLong + deliveryScheduleStale
  backlog.deliveryOverdue = backlog.delivery

  const collabOrderTimeout = workorders.filter((item) => {
    const status = Number(item.status)
    return (status === 0 || status === 1 || status === 4) && elapsedMinutes(item.created_at) >= 120
  }).length
  const collabWorkflowLong = workflowExecutions.filter(
    (item) => Number(item.status) === 0 && elapsedMinutes(item.started_at) >= 15
  ).length
  const collabTerminalPending = terminalSessions.filter((item) => {
    if (Number(item.status) !== 0) return false
    return elapsedMinutes(item.started_at || item.created_at) >= 10
  }).length
  const collabTerminalFailed = terminalSessions.filter((item) => Number(item.status) === 3).length
  backlog.collab = collabOrderTimeout + collabWorkflowLong + collabTerminalPending + collabTerminalFailed
  backlog.collabOverdue = collabOrderTimeout + collabWorkflowLong + collabTerminalPending

  backlog.total = backlog.asset + backlog.monitor + backlog.k8s + backlog.delivery + backlog.collab
  backlog.overdue =
    backlog.assetOverdue +
    backlog.monitorOverdue +
    backlog.k8sOverdue +
    backlog.deliveryOverdue +
    backlog.collabOverdue

  lastUpdated.value = new Date().toLocaleString()
  renderTrend()
  await refreshTopHosts()

  if (failures.length) {
    ElMessage.warning(`部分数据加载失败：${failures.join('、')}`)
  }

  refreshing.value = false
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

.motion-up {
  animation: motion-up 0.38s ease both;
}

.delay-1 { animation-delay: 0.03s; }
.delay-2 { animation-delay: 0.06s; }
.delay-3 { animation-delay: 0.09s; }
.delay-4 { animation-delay: 0.12s; }
.delay-5 { animation-delay: 0.15s; }

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
}

@media (max-width: 760px) {
  .backlog-grid {
    grid-template-columns: 1fr;
  }
}
</style>
