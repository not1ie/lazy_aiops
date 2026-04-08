<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>监控概览</h2>
        <p class="page-desc">资源趋势、告警汇总与在线状态。</p>
      </div>
      <div class="page-actions">
        <el-button icon="Refresh" @click="refreshAll">刷新</el-button>
      </div>
    </div>

    <el-row :gutter="16" class="metric-cards">
      <el-col :span="6"><el-card><div class="card-title">CPU</div><div class="card-value">{{ realtime.cpu }}%</div></el-card></el-col>
      <el-col :span="6"><el-card><div class="card-title">内存</div><div class="card-value">{{ realtime.memory }}%</div></el-card></el-col>
      <el-col :span="6"><el-card><div class="card-title">磁盘</div><div class="card-value">{{ realtime.disk }}%</div></el-card></el-col>
      <el-col :span="6"><el-card><div class="card-title">网络(MB/s)</div><div class="card-value">{{ realtime.network }}</div></el-card></el-col>
    </el-row>

    <el-divider />

    <el-row :gutter="16">
      <el-col :span="16">
        <el-card>
          <div class="section-title">资源趋势 (最近1小时)</div>
          <div ref="trendRef" class="chart-box"></div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card>
          <div class="section-title">告警汇总</div>
          <el-statistic title="未处理" :value="alertStats.open" />
          <el-statistic title="已处理" :value="alertStats.closed" />
          <el-statistic title="已忽略" :value="alertStats.ignored" />
        </el-card>
        <el-card class="mt-12">
          <div class="section-title">Agent 在线</div>
          <el-statistic title="在线" :value="agentStats.online" />
          <el-statistic title="状态过期" :value="agentStats.stale" />
          <el-statistic title="离线" :value="agentStats.offline" />
          <el-statistic title="未知" :value="agentStats.unknown" />
        </el-card>
        <el-card class="mt-12">
          <div class="section-title">最近告警</div>
          <el-table :fit="true" :data="recentAlerts" size="small" style="width: 100%">
            <el-table-column prop="rule_name" label="规则" min-width="120" />
            <el-table-column prop="severity" label="级别" width="90" />
            <el-table-column prop="status" label="状态" width="90">
              <template #default="scope">
                <StatusBadge v-bind="alertStatusBadge(scope.row)" />
              </template>
            </el-table-column>
          </el-table>
        </el-card>
        <el-card class="mt-12">
          <div class="section-title">Top 节点</div>
          <el-table :fit="true" :data="topNodes" size="small" style="width: 100%">
            <el-table-column prop="instance" label="节点" min-width="140" />
            <el-table-column prop="cpu" label="CPU%" width="90" />
            <el-table-column prop="memory" label="内存%" width="90" />
          </el-table>
        </el-card>
      </el-col>
    </el-row>
  </el-card>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount } from 'vue'
import axios from 'axios'
import * as echarts from 'echarts'
import { ElMessage } from 'element-plus'
import { getErrorMessage } from '@/utils/error'
import { monitorAgentStatusMeta, monitorAlertStatusMeta } from '@/utils/status'
import StatusBadge from '@/components/common/StatusBadge.vue'

const realtime = ref({ cpu: 0, memory: 0, disk: 0, network: 0 })
const alertStats = ref({ open: 0, closed: 0, ignored: 0 })
const agentStats = ref({ online: 0, stale: 0, offline: 0, unknown: 0 })
const recentAlerts = ref([])
const topNodes = ref([])

const trendRef = ref(null)
let trendChart = null
let refreshTimer = null

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const alertStatusBadge = (row) => {
  const meta = monitorAlertStatusMeta(row)
  const parts = []
  if (row?.severity) parts.push(`级别: ${row.severity}`)
  if (row?.target) parts.push(`目标: ${row.target}`)
  if (row?.message) parts.push(row.message)
  return {
    text: meta.text,
    type: meta.type,
    source: meta.source,
    checkAt: meta.checkAt,
    reason: parts.join(' | ') || meta.reason,
    isStale: meta.isStale,
    staleText: meta.staleText,
    updatedAt: row?.updated_at || row?.created_at
  }
}

const fetchRealtime = async () => {
  try {
    const res = await axios.get('/api/v1/monitor/metrics', { headers: authHeaders() })
    realtime.value = res.data.data || realtime.value
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '加载实时指标失败'))
  }
}

const fetchAlerts = async () => {
  try {
    const res = await axios.get('/api/v1/monitor/alerts', { headers: authHeaders() })
    const data = res.data.data || []
    alertStats.value.open = data.filter(a => a.status === 0).length
    alertStats.value.closed = data.filter(a => a.status === 1).length
    alertStats.value.ignored = data.filter(a => a.status === 2).length
    recentAlerts.value = data.slice(0, 5)
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '加载告警汇总失败'))
  }
}

const fetchAgents = async () => {
  try {
    const res = await axios.get('/api/v1/monitor/agents', { headers: authHeaders() })
    const data = res.data.data || []
    const now = Date.now()
    const counts = { online: 0, stale: 0, offline: 0, unknown: 0 }
    data.forEach((item) => {
      const key = monitorAgentStatusMeta(item, { staleMinutes: 3, nowMs: now }).key
      if (Object.prototype.hasOwnProperty.call(counts, key)) {
        counts[key] += 1
      } else if (key === 'stale') {
        counts.stale += 1
      } else {
        counts.unknown += 1
      }
    })
    agentStats.value = counts
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '加载 Agent 状态失败'))
  }
}

const fetchHistory = async () => {
  try {
    const res = await axios.get('/api/v1/monitor/metrics/history', {
      headers: authHeaders(),
      params: { hours: 1 }
    })
    const data = res.data.data || []
    renderTrend(data)
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '加载趋势数据失败'))
  }
}

const fetchProm = async (query) => {
  const res = await axios.get('/api/v1/monitor/prometheus/query', {
    headers: authHeaders(),
    params: { query }
  })
  return res.data?.data?.result || []
}

const fetchTopNodes = async () => {
  try {
    const cpuQuery = 'topk(5, 100 - (avg by(instance) (irate(node_cpu_seconds_total{mode="idle"}[5m])) * 100))'
    const memQuery = 'topk(5, (1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100)'
    const [cpuRes, memRes] = await Promise.all([fetchProm(cpuQuery), fetchProm(memQuery)])
    const map = {}
    cpuRes.forEach((item) => {
      const inst = item.metric?.instance || '-'
      map[inst] = map[inst] || { instance: inst, cpu: '-', memory: '-' }
      map[inst].cpu = Number(item.value?.[1] || 0).toFixed(1)
    })
    memRes.forEach((item) => {
      const inst = item.metric?.instance || '-'
      map[inst] = map[inst] || { instance: inst, cpu: '-', memory: '-' }
      map[inst].memory = Number(item.value?.[1] || 0).toFixed(1)
    })
    topNodes.value = Object.values(map)
  } catch (e) {
    topNodes.value = []
    ElMessage.error(getErrorMessage(e, '加载 Top 节点失败'))
  }
}

const renderTrend = (records) => {
  if (!trendRef.value) return
  if (!trendChart) trendChart = echarts.init(trendRef.value)
  const labels = records.map(r => new Date(r.timestamp).toLocaleTimeString())
  trendChart.setOption({
    tooltip: { trigger: 'axis' },
    legend: { top: 0 },
    xAxis: { type: 'category', data: labels },
    yAxis: { type: 'value' },
    series: [
      { name: 'CPU', type: 'line', showSymbol: false, data: records.map(r => r.cpu_usage) },
      { name: '内存', type: 'line', showSymbol: false, data: records.map(r => r.memory_usage) },
      { name: '磁盘', type: 'line', showSymbol: false, data: records.map(r => r.disk_usage) }
    ]
  })
}

const refreshAll = async () => {
  await Promise.all([fetchRealtime(), fetchHistory(), fetchAlerts(), fetchAgents(), fetchTopNodes()])
}

onMounted(() => {
  refreshAll()
  refreshTimer = window.setInterval(() => {
    if (document.hidden) return
    refreshAll()
  }, 60 * 1000)
})

onBeforeUnmount(() => {
  if (trendChart) {
    trendChart.dispose()
    trendChart = null
  }
  if (refreshTimer) {
    window.clearInterval(refreshTimer)
    refreshTimer = null
  }
})
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 16px; }
.page-desc { color: #606266; margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; }
.metric-cards { margin-bottom: 8px; }
.card-title { color: #909399; font-size: 12px; }
.card-value { font-size: 22px; font-weight: 600; margin-top: 6px; }
.section-title { margin-bottom: 8px; font-weight: 600; }
.chart-box { width: 100%; height: 320px; }
.mt-12 { margin-top: 12px; }
</style>
