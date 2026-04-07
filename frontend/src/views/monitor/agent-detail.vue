<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>Agent 详情</h2>
        <p class="page-desc">查看采集器基础信息与标签。</p>
      </div>
      <div class="page-actions">
        <el-button icon="Refresh" @click="fetchAgent">刷新</el-button>
      </div>
    </div>

    <el-descriptions :column="2" border>
      <el-descriptions-item label="Agent ID">{{ agent.agent_id }}</el-descriptions-item>
      <el-descriptions-item label="主机名">{{ agent.hostname }}</el-descriptions-item>
      <el-descriptions-item label="IP">{{ agent.ip }}</el-descriptions-item>
      <el-descriptions-item label="版本">{{ agent.version }}</el-descriptions-item>
      <el-descriptions-item label="OS">{{ agent.os }}</el-descriptions-item>
      <el-descriptions-item label="状态">
        <StatusBadge
          :text="agentStatusMeta.text"
          :type="agentStatusMeta.type"
          :reason="agentStatusReason"
          :updated-at="agent.last_seen"
          size="small"
        />
      </el-descriptions-item>
      <el-descriptions-item label="最后心跳">{{ formatTime(agent.last_seen) }}</el-descriptions-item>
      <el-descriptions-item label="状态说明">{{ agentStatusReason }}</el-descriptions-item>
    </el-descriptions>

    <el-divider />

    <h3 class="section-title">标签</h3>
    <div>
      <el-tag v-for="(v, k) in labels" :key="k" class="mr-2" size="small">{{ k }}={{ v }}</el-tag>
      <div v-if="Object.keys(labels).length === 0" class="muted">无标签</div>
    </div>

    <el-divider />

    <h3 class="section-title">元信息</h3>
    <el-input v-model="metaText" type="textarea" :rows="10" readonly />

    <el-divider />

    <h3 class="section-title">心跳趋势</h3>
    <div class="history-controls">
      <el-select v-model="hours" class="w-40" @change="fetchHistory">
        <el-option label="最近1小时" :value="1" />
        <el-option label="最近6小时" :value="6" />
        <el-option label="最近24小时" :value="24" />
        <el-option label="最近72小时" :value="72" />
      </el-select>
      <el-button icon="Refresh" @click="fetchHistory">刷新</el-button>
    </div>
    <div ref="historyRef" class="chart-box"></div>
  </el-card>
</template>

<script setup>
import { computed, ref, onMounted, onBeforeUnmount } from 'vue'
import { useRoute } from 'vue-router'
import axios from 'axios'
import * as echarts from 'echarts'
import { ElMessage } from 'element-plus'
import { getErrorMessage } from '@/utils/error'
import StatusBadge from '@/components/common/StatusBadge.vue'
import { monitorAgentStatusMeta } from '@/utils/status'

const route = useRoute()
const agent = ref({})
const labels = ref({})
const metaText = ref('')
const historyRef = ref(null)
let historyChart = null
const hours = ref(24)
const loading = ref(false)
const historyLoading = ref(false)
const nowTick = ref(Date.now())
let statusTicker = null
let autoRefreshTicker = null

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })
const agentStatusMeta = computed(() => monitorAgentStatusMeta(agent.value, { staleMinutes: 3, nowMs: nowTick.value }))
const agentStatusReason = computed(() => {
  if (agentStatusMeta.value.key === 'online') return '心跳正常'
  if (agentStatusMeta.value.key === 'stale') return '超过 3 分钟未上报心跳'
  if (agentStatusMeta.value.key === 'offline') return '采集器离线或不可达'
  if (agentStatusMeta.value.key === 'maintenance') return '采集器处于维护状态'
  return '状态未知，请检查采集器进程和网络'
})

const parseJSON = (text) => {
  try {
    return text ? JSON.parse(text) : {}
  } catch {
    return {}
  }
}

const fetchAgent = async () => {
  const id = route.query.id
  if (!id) return
  if (loading.value) return
  loading.value = true
  try {
    const res = await axios.get(`/api/v1/monitor/agents/${id}`, { headers: authHeaders() })
    agent.value = res.data.data || {}
    labels.value = parseJSON(agent.value.labels)
    const meta = parseJSON(agent.value.meta)
    metaText.value = JSON.stringify(meta, null, 2)
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '加载 Agent 详情失败'))
  } finally {
    loading.value = false
  }
}

const fetchHistory = async () => {
  const id = route.query.id
  if (!id) return
  if (historyLoading.value) return
  historyLoading.value = true
  try {
    const res = await axios.get(`/api/v1/monitor/agents/${id}/history`, {
      headers: authHeaders(),
      params: { hours: hours.value }
    })
    const records = res.data.data || []
    renderHistory(records)
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '加载 Agent 历史失败'))
  } finally {
    historyLoading.value = false
  }
}

const renderHistory = (records) => {
  if (!historyRef.value) return
  if (!historyChart) historyChart = echarts.init(historyRef.value)
  const labels = records.map(r => new Date(r.timestamp).toLocaleTimeString())
  const series = []

  const cpu = records.map(r => {
    const meta = parseJSON(r.meta)
    return typeof meta.cpu === 'number' ? meta.cpu : null
  })
  const mem = records.map(r => {
    const meta = parseJSON(r.meta)
    return typeof meta.memory === 'number' ? meta.memory : null
  })
  const disk = records.map(r => {
    const meta = parseJSON(r.meta)
    return typeof meta.disk === 'number' ? meta.disk : null
  })
  const netIn = records.map(r => {
    const meta = parseJSON(r.meta)
    return typeof meta.net_in === 'number' ? meta.net_in : null
  })
  const netOut = records.map(r => {
    const meta = parseJSON(r.meta)
    return typeof meta.net_out === 'number' ? meta.net_out : null
  })

  if (cpu.some(v => v !== null)) {
    series.push({ name: 'CPU', type: 'line', showSymbol: false, data: cpu })
  }
  if (mem.some(v => v !== null)) {
    series.push({ name: '内存', type: 'line', showSymbol: false, data: mem })
  }
  if (disk.some(v => v !== null)) {
    series.push({ name: '磁盘', type: 'line', showSymbol: false, data: disk })
  }
  if (netIn.some(v => v !== null)) {
    series.push({ name: '入流量', type: 'line', showSymbol: false, data: netIn })
  }
  if (netOut.some(v => v !== null)) {
    series.push({ name: '出流量', type: 'line', showSymbol: false, data: netOut })
  }
  if (series.length === 0) {
    series.push({ name: '心跳', type: 'line', showSymbol: false, data: records.map(() => 1) })
  }

  historyChart.setOption({
    tooltip: { trigger: 'axis' },
    legend: { top: 0 },
    xAxis: { type: 'category', data: labels },
    yAxis: { type: 'value' },
    series
  })
}

onMounted(async () => {
  await fetchAgent()
  await fetchHistory()
  statusTicker = window.setInterval(() => {
    nowTick.value = Date.now()
  }, 60 * 1000)
  autoRefreshTicker = window.setInterval(() => {
    if (document.hidden) return
    fetchAgent()
  }, 60 * 1000)
})

onBeforeUnmount(() => {
  if (statusTicker) {
    window.clearInterval(statusTicker)
    statusTicker = null
  }
  if (autoRefreshTicker) {
    window.clearInterval(autoRefreshTicker)
    autoRefreshTicker = null
  }
  if (historyChart) {
    historyChart.dispose()
    historyChart = null
  }
})

const formatTime = (value) => {
  if (!value) return '-'
  const ts = new Date(value).getTime()
  if (Number.isNaN(ts)) return '-'
  return new Date(ts).toLocaleString()
}
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 16px; }
.page-desc { color: #606266; margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; }
.section-title { margin: 12px 0; }
.mr-2 { margin-right: 6px; margin-bottom: 6px; }
.muted { color: #909399; font-size: 12px; margin-top: 6px; }
.history-controls { display: flex; gap: 8px; align-items: center; margin-bottom: 8px; }
.chart-box { width: 100%; height: 300px; }
.w-40 { width: 160px; }
</style>
