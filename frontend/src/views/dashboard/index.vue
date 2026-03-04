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

    <el-row :gutter="16" class="panel-row motion-up delay-4">
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
import { ref, reactive, onMounted, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'
import * as echarts from 'echarts'
import { ElMessage } from 'element-plus'

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

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const go = (path) => router.push(path)

const toArray = (v) => (Array.isArray(v) ? v : [])
const toNumber = (v, fallback = 0) => {
  const n = Number(v)
  return Number.isFinite(n) ? n : fallback
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
    axios.get('/api/v1/monitor/metrics/history', { headers: authHeaders(), params: { hours: 24 } })
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
    failures.push('CMDB')
  }

  const dockerPayload = extractData(calls[1])
  const dockerHosts = toArray(dockerPayload)
  if (dockerPayload !== null) {
    stats.dockerTotal = dockerHosts.length
    stats.dockerOnline = dockerHosts.filter((h) => String(h.status).toLowerCase() === 'online').length
    moduleStatus.docker = 'ok'
  } else {
    moduleStatus.docker = 'error'
    failures.push('Docker')
  }

  const clustersPayload = extractData(calls[2])
  const clusters = toArray(clustersPayload)
  if (clustersPayload !== null) {
    stats.k8sTotal = clusters.length
    stats.k8sHealthy = clusters.filter((c) => toNumber(c.status, -1) === 1 || String(c.status).toLowerCase() === 'online').length
    moduleStatus.k8s = 'ok'
  } else {
    moduleStatus.k8s = 'error'
    failures.push('K8s')
  }

  const alertsPayload = extractData(calls[3])
  const alerts = toArray(alertsPayload)
  if (alertsPayload !== null) {
    stats.alertTotal = alerts.length
    stats.alertOpen = alerts.filter((a) => toNumber(a.status, -1) === 0).length
    recentAlerts.value = alerts.slice(0, 8)
  } else {
    failures.push('告警')
  }

  const tasksPayload = extractData(calls[4])
  const tasks = toArray(tasksPayload)
  if (tasksPayload !== null) {
    stats.taskTotal = tasks.length
    stats.taskEnabled = tasks.filter((t) => Boolean(t.enabled)).length
    moduleStatus.task = 'ok'
  } else {
    moduleStatus.task = 'error'
    failures.push('任务')
  }

  const agentsPayload = extractData(calls[5])
  const agents = toArray(agentsPayload)
  if (agentsPayload !== null) {
    stats.agentTotal = agents.length
    stats.agentOnline = agents.filter((a) => String(a.status).toLowerCase() === 'online').length
  } else {
    failures.push('Agent')
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
    failures.push('监控')
  }

  const historyPayload = extractData(calls[7])
  const history = toArray(historyPayload)
  if (historyPayload === null) {
    failures.push('趋势')
  }
  trendRecords.value = history
    .map((item) => ({
      timestamp: item.timestamp,
      cpu_usage: toNumber(item.cpu_usage),
      memory_usage: toNumber(item.memory_usage),
      disk_usage: toNumber(item.disk_usage)
    }))
    .sort((a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime())

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
</style>
