<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>主机监控</h2>
        <p class="page-desc">基于 Prometheus / Node Exporter 的主机指标与趋势。</p>
      </div>
      <div class="page-actions">
        <el-button icon="Refresh" @click="refreshAll">刷新</el-button>
      </div>
    </div>

    <div class="filter-bar">
      <el-select v-model="instanceFilter" placeholder="选择主机" class="w-52" clearable>
        <el-option v-for="inst in instances" :key="inst" :label="inst" :value="inst" />
      </el-select>
      <el-input v-model="keyword" placeholder="搜索主机/IP" class="w-52" clearable />
      <el-radio-group v-model="rangeHours" size="small">
        <el-radio-button v-for="opt in rangeOptions" :key="opt.value" :label="opt.value">
          {{ opt.label }}
        </el-radio-button>
      </el-radio-group>
    </div>

    <el-row :gutter="16" class="summary-row">
      <el-col :span="6"><el-card><div class="card-title">主机数</div><div class="card-value">{{ summary.total }}</div></el-card></el-col>
      <el-col :span="6"><el-card><div class="card-title">CPU 平均</div><div class="card-value">{{ summary.avgCpu }}%</div></el-card></el-col>
      <el-col :span="6"><el-card><div class="card-title">内存 平均</div><div class="card-value">{{ summary.avgMem }}%</div></el-card></el-col>
      <el-col :span="6"><el-card><div class="card-title">磁盘 平均</div><div class="card-value">{{ summary.avgDisk }}%</div></el-card></el-col>
    </el-row>

    <el-card class="panel-card" v-loading="chartLoading">
      <div class="panel-header">
        <div>
          <h3>主机趋势</h3>
          <p class="panel-desc">CPU / 内存 / 磁盘 / 网络（近 {{ rangeLabel }}）。</p>
        </div>
        <div class="panel-actions">
          <el-button size="small" @click="fetchCharts">刷新趋势</el-button>
        </div>
      </div>
      <el-row :gutter="16">
        <el-col :span="12"><div ref="cpuChartRef" class="chart-box"></div></el-col>
        <el-col :span="12"><div ref="memChartRef" class="chart-box"></div></el-col>
      </el-row>
      <el-row :gutter="16" class="chart-row">
        <el-col :span="12"><div ref="diskChartRef" class="chart-box"></div></el-col>
        <el-col :span="12"><div ref="netChartRef" class="chart-box"></div></el-col>
      </el-row>
    </el-card>

    <el-table :data="filteredRows" v-loading="loading" style="width: 100%; margin-top: 12px">
      <el-table-column prop="instance" label="主机" min-width="200" />
      <el-table-column prop="cpu" label="CPU(%)" width="120" sortable />
      <el-table-column prop="memory" label="内存(%)" width="120" sortable />
      <el-table-column prop="disk" label="磁盘(%)" width="120" sortable />
      <el-table-column prop="load1" label="Load1" width="120" sortable />
      <el-table-column prop="network" label="网络(MB/s)" width="140" sortable />
      <el-table-column prop="uptime" label="运行时长" width="160" />
    </el-table>
  </el-card>
</template>

<script setup>
import { ref, computed, onMounted, watch, onBeforeUnmount } from 'vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import * as echarts from 'echarts'

const loading = ref(false)
const chartLoading = ref(false)
const keyword = ref('')
const instanceFilter = ref('')
const instances = ref([])
const rows = ref([])

const rangeHours = ref(1)
const rangeOptions = [
  { label: '1h', value: 1 },
  { label: '6h', value: 6 },
  { label: '24h', value: 24 },
  { label: '7d', value: 168 }
]

const cpuChartRef = ref(null)
const memChartRef = ref(null)
const diskChartRef = ref(null)
const netChartRef = ref(null)
let cpuChart = null
let memChart = null
let diskChart = null
let netChart = null

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const rangeLabel = computed(() => {
  const found = rangeOptions.find(opt => opt.value === rangeHours.value)
  return found ? found.label : `${rangeHours.value}h`
})

const buildSelector = (base, instance) => {
  if (!instance) return base
  if (!base) return `instance="${instance}"`
  return `${base},instance="${instance}"`
}

const fetchProm = async (query) => {
  const res = await axios.get('/api/v1/monitor/prometheus/query', {
    headers: authHeaders(),
    params: { query: query.replace(/\s+/g, ' ').trim() }
  })
  if (res.data?.status && res.data.status !== 'success') {
    throw new Error(res.data?.error || 'Prometheus 查询失败')
  }
  return res.data?.data?.result || []
}

const fetchPromRange = async (query, start, end, step) => {
  const res = await axios.get('/api/v1/monitor/prometheus/query_range', {
    headers: authHeaders(),
    params: {
      query: query.replace(/\s+/g, ' ').trim(),
      start,
      end,
      step
    }
  })
  if (res.data?.status && res.data.status !== 'success') {
    throw new Error(res.data?.error || 'Prometheus 查询失败')
  }
  return res.data?.data?.result || []
}

const queryCpu = (instance) =>
  `100 - (avg by(instance) (irate(node_cpu_seconds_total{${buildSelector('mode="idle"', instance)}}[5m])) * 100)`

const queryMem = (instance) =>
  `100 * (1 - (node_memory_MemAvailable_bytes{${buildSelector('', instance)}} / node_memory_MemTotal_bytes{${buildSelector('', instance)}}))`

const queryDisk = (instance) =>
  `max by(instance) (100 - (node_filesystem_free_bytes{${buildSelector('fstype!="tmpfs"', instance)}} / node_filesystem_size_bytes{${buildSelector('fstype!="tmpfs"', instance)}}) * 100)`

const queryNet = (instance) =>
  `sum by(instance) (rate(node_network_receive_bytes_total{${buildSelector('', instance)}}[5m]) + rate(node_network_transmit_bytes_total{${buildSelector('', instance)}}[5m])) / 1024 / 1024`

const queryLoad = (instance) =>
  `node_load1{${buildSelector('', instance)}}`

const queryUptime = (instance) =>
  `node_time_seconds{${buildSelector('', instance)}} - node_boot_time_seconds{${buildSelector('', instance)}}`

const ensureCharts = () => {
  if (cpuChartRef.value && !cpuChart) cpuChart = echarts.init(cpuChartRef.value)
  if (memChartRef.value && !memChart) memChart = echarts.init(memChartRef.value)
  if (diskChartRef.value && !diskChart) diskChart = echarts.init(diskChartRef.value)
  if (netChartRef.value && !netChart) netChart = echarts.init(netChartRef.value)
}

const renderChart = (chart, title, labels, series, unit) => {
  if (!chart) return
  chart.setOption({
    title: { text: title, left: 'left', textStyle: { fontSize: 12 } },
    tooltip: { trigger: 'axis', valueFormatter: (val) => `${Number(val).toFixed(2)} ${unit}` },
    legend: { top: 24, type: 'scroll' },
    grid: { left: 50, right: 20, top: 60, bottom: 30 },
    xAxis: { type: 'category', data: labels },
    yAxis: { type: 'value' },
    series
  })
}

const calcStep = (hours) => {
  if (hours <= 1) return 30
  if (hours <= 6) return 60
  if (hours <= 24) return 120
  return 300
}

const fetchInstances = async () => {
  try {
    let res = await fetchProm('node_uname_info')
    if (!res.length) {
      res = await fetchProm('up{job=~"node.*"}')
    }
    instances.value = Array.from(new Set(res.map(item => item.metric?.instance).filter(Boolean)))
    if (!instanceFilter.value && instances.value.length) {
      instanceFilter.value = instances.value[0]
    }
  } catch (err) {
    ElMessage.error('获取主机列表失败')
  }
}

const fetchTable = async () => {
  loading.value = true
  try {
    const [cpuRes, memRes, diskRes, netRes, loadRes, upRes] = await Promise.all([
      fetchProm(queryCpu('')),
      fetchProm(queryMem('')),
      fetchProm(queryDisk('')),
      fetchProm(queryNet('')),
      fetchProm(queryLoad('')),
      fetchProm(queryUptime(''))
    ])
    const map = {}
    const apply = (res, key, format) => {
      res.forEach(item => {
        const inst = item.metric?.instance || '-'
        map[inst] = map[inst] || { instance: inst, cpu: 0, memory: 0, disk: 0, network: 0, load1: 0, uptime: '-' }
        map[inst][key] = format(Number(item.value?.[1] || 0))
      })
    }
    apply(cpuRes, 'cpu', v => v.toFixed(2))
    apply(memRes, 'memory', v => v.toFixed(2))
    apply(diskRes, 'disk', v => v.toFixed(2))
    apply(netRes, 'network', v => v.toFixed(2))
    apply(loadRes, 'load1', v => v.toFixed(2))
    apply(upRes, 'uptime', v => {
      const hours = v / 3600
      if (!Number.isFinite(hours)) return '-'
      return hours > 24 ? `${(hours / 24).toFixed(1)}d` : `${hours.toFixed(1)}h`
    })
    rows.value = Object.values(map)
  } catch (err) {
    ElMessage.error('主机指标获取失败')
  } finally {
    loading.value = false
  }
}

const fetchCharts = async () => {
  chartLoading.value = true
  try {
    ensureCharts()
    const end = Math.floor(Date.now() / 1000)
    const start = end - (rangeHours.value * 3600)
    const step = calcStep(rangeHours.value)
    const inst = instanceFilter.value
    const [cpuRes, memRes, diskRes, netRes] = await Promise.all([
      fetchPromRange(queryCpu(inst), start, end, step),
      fetchPromRange(queryMem(inst), start, end, step),
      fetchPromRange(queryDisk(inst), start, end, step),
      fetchPromRange(queryNet(inst), start, end, step)
    ])
    const labels = (cpuRes[0]?.values || memRes[0]?.values || []).map(v =>
      new Date(Number(v[0]) * 1000).toLocaleTimeString()
    )
    const seriesCpu = cpuRes.map(item => ({ name: item.metric?.instance || 'cpu', type: 'line', showSymbol: false, data: item.values.map(v => Number(v[1])) }))
    const seriesMem = memRes.map(item => ({ name: item.metric?.instance || 'mem', type: 'line', showSymbol: false, data: item.values.map(v => Number(v[1])) }))
    const seriesDisk = diskRes.map(item => ({ name: item.metric?.instance || 'disk', type: 'line', showSymbol: false, data: item.values.map(v => Number(v[1])) }))
    const seriesNet = netRes.map(item => ({ name: item.metric?.instance || 'net', type: 'line', showSymbol: false, data: item.values.map(v => Number(v[1])) }))

    renderChart(cpuChart, 'CPU 使用率(%)', labels, seriesCpu, '%')
    renderChart(memChart, '内存 使用率(%)', labels, seriesMem, '%')
    renderChart(diskChart, '磁盘 使用率(%)', labels, seriesDisk, '%')
    renderChart(netChart, '网络 吞吐(MB/s)', labels, seriesNet, 'MB/s')
  } catch (err) {
    ElMessage.error('趋势图加载失败')
  } finally {
    chartLoading.value = false
  }
}

const refreshAll = async () => {
  await fetchInstances()
  await fetchTable()
  await fetchCharts()
}

const filteredRows = computed(() => {
  const key = keyword.value.trim().toLowerCase()
  return rows.value.filter(r => {
    if (instanceFilter.value && r.instance !== instanceFilter.value) return false
    if (!key) return true
    return (r.instance || '').toLowerCase().includes(key)
  })
})

const summary = computed(() => {
  const total = filteredRows.value.length
  const avg = (key) => {
    if (!filteredRows.value.length) return '0.00'
    const sum = filteredRows.value.reduce((acc, r) => acc + Number(r[key] || 0), 0)
    return (sum / filteredRows.value.length).toFixed(2)
  }
  return {
    total,
    avgCpu: avg('cpu'),
    avgMem: avg('memory'),
    avgDisk: avg('disk')
  }
})

watch(rangeHours, fetchCharts)
watch(instanceFilter, fetchCharts)

onMounted(refreshAll)

onBeforeUnmount(() => {
  if (cpuChart) cpuChart.dispose()
  if (memChart) memChart.dispose()
  if (diskChart) diskChart.dispose()
  if (netChart) netChart.dispose()
  cpuChart = null
  memChart = null
  diskChart = null
  netChart = null
})
</script>

<style scoped>
.page-card { max-width: 1180px; margin: 0 auto; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 16px; }
.page-desc { color: #606266; margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; }
.filter-bar { display: flex; flex-wrap: wrap; gap: 8px; margin-bottom: 12px; }
.summary-row { margin-top: 8px; }
.panel-card { margin-top: 16px; }
.panel-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 8px; }
.panel-desc { color: #909399; font-size: 12px; margin: 4px 0 0; }
.panel-actions { display: flex; align-items: center; gap: 8px; }
.chart-box { width: 100%; height: 260px; }
.chart-row { margin-top: 8px; }
.card-title { color: #909399; font-size: 12px; }
.card-value { font-size: 20px; font-weight: 600; margin-top: 6px; }
.w-52 { width: 220px; }
</style>
