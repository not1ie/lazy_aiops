<template>
  <el-card class="page-card">
    <div class="page-header motion-up delay-1">
      <div>
        <h2>主机监控</h2>
        <p class="page-desc">基于 Prometheus / Node Exporter 的主机指标与趋势。</p>
      </div>
      <div class="page-actions">
        <el-button icon="Refresh" @click="refreshAll">刷新</el-button>
      </div>
    </div>

    <div class="filter-bar motion-up delay-2">
      <el-select v-model="companyFilter" placeholder="公司" class="w-40" clearable>
        <el-option v-for="item in companies" :key="item" :label="item" :value="item" />
      </el-select>
      <el-select v-model="envFilter" placeholder="环境" class="w-40" clearable>
        <el-option v-for="item in envs" :key="item" :label="item" :value="item" />
      </el-select>
      <el-select v-model="instanceFilter" placeholder="选择主机" class="w-52" clearable>
        <el-option v-for="inst in filteredInstances" :key="inst" :label="inst" :value="inst" />
      </el-select>
      <el-input v-model="keyword" placeholder="搜索主机/IP" class="w-52" clearable />
      <el-radio-group v-model="rangeHours" size="small">
        <el-radio-button v-for="opt in rangeOptions" :key="opt.value" :label="opt.value">
          {{ opt.label }}
        </el-radio-button>
      </el-radio-group>
    </div>

    <div class="layout-bar motion-up delay-3">
      <div class="layout-title">面板布局</div>
      <div class="layout-items">
        <div
          v-for="(panel, idx) in panels"
          :key="panel.id"
          class="layout-item"
          draggable="true"
          @dragstart="onDragStart(idx)"
          @dragover.prevent
          @drop="onDrop(idx)"
        >
          <span class="drag-handle">≡</span>
          <el-checkbox v-model="panel.visible">{{ panel.title }}</el-checkbox>
        </div>
      </div>
      <div class="layout-actions">
        <el-input v-model="layoutName" placeholder="模板名称" class="w-40" />
        <el-button size="small" @click="saveLayout">保存模板</el-button>
        <el-select v-model="selectedLayout" placeholder="加载模板" class="w-40" clearable @change="applyLayout">
          <el-option v-for="item in layouts" :key="item.name" :label="item.name" :value="item.name" />
        </el-select>
        <el-button size="small" type="danger" plain @click="deleteLayout" :disabled="!selectedLayout">删除模板</el-button>
      </div>
    </div>

    <el-row :gutter="16" class="summary-row motion-up delay-4">
      <el-col :span="6"><el-card class="summary-card"><div class="card-title">主机数</div><div class="card-value">{{ summary.total }}</div></el-card></el-col>
      <el-col :span="6">
        <el-card class="summary-card">
          <div class="card-title">CPU 平均</div>
          <el-progress type="dashboard" :percentage="Number(summary.avgCpu)" :color="progressColor(Number(summary.avgCpu))" />
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card class="summary-card">
          <div class="card-title">内存 平均</div>
          <el-progress type="dashboard" :percentage="Number(summary.avgMem)" :color="progressColor(Number(summary.avgMem))" />
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card class="summary-card">
          <div class="card-title">磁盘 平均</div>
          <el-progress type="dashboard" :percentage="Number(summary.avgDisk)" :color="progressColor(Number(summary.avgDisk))" />
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="health-row motion-up delay-5">
      <el-col :span="6"><el-card class="summary-card"><div class="card-title">健康</div><div class="card-value health-ok">{{ healthSummary.ok }}</div></el-card></el-col>
      <el-col :span="6"><el-card class="summary-card"><div class="card-title">预警</div><div class="card-value health-warn">{{ healthSummary.warn }}</div></el-card></el-col>
      <el-col :span="6"><el-card class="summary-card"><div class="card-title">严重</div><div class="card-value health-critical">{{ healthSummary.critical }}</div></el-card></el-col>
      <el-col :span="6">
        <el-card class="summary-card">
          <div class="card-title">当前主机</div>
          <div class="card-sub">{{ selectedHost.instance || '-' }}</div>
          <div class="card-mini">
            CPU {{ selectedHost.cpu || '0.00' }}% / 内存 {{ selectedHost.memory || '0.00' }}%
          </div>
        </el-card>
      </el-col>
    </el-row>

    <template v-for="panel in panels" :key="panel.id">
      <el-card v-if="panel.id === 'top' && panel.visible" class="panel-card motion-up delay-6">
        <div class="panel-header">
          <div>
            <h3>主机排行</h3>
            <p class="panel-desc">当前 CPU / 内存 Top 主机。</p>
          </div>
          <div class="panel-actions">
            <el-button size="small" @click="renderTopCharts">刷新排行</el-button>
          </div>
        </div>
        <el-row :gutter="16">
          <el-col :span="12"><div ref="topCpuChartRef" class="chart-box"></div></el-col>
          <el-col :span="12"><div ref="topMemChartRef" class="chart-box"></div></el-col>
        </el-row>
      </el-card>

      <el-card v-if="panel.id === 'trend' && panel.visible" class="panel-card motion-up delay-7" v-loading="chartLoading">
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
    </template>

    <el-table class="motion-up delay-8" :data="filteredRows" v-loading="loading" style="width: 100%; margin-top: 12px" @row-click="selectInstance">
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
import { ref, computed, onMounted, watch, onBeforeUnmount, nextTick } from 'vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import * as echarts from 'echarts'

const loading = ref(false)
const chartLoading = ref(false)
const keyword = ref('')
const instanceFilter = ref('')
const instances = ref([])
const instanceMeta = ref([])
const companies = ref([])
const envs = ref([])
const companyFilter = ref('')
const envFilter = ref('')
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
const topCpuChartRef = ref(null)
const topMemChartRef = ref(null)
let cpuChart = null
let memChart = null
let diskChart = null
let netChart = null
let topCpuChart = null
let topMemChart = null
const panels = ref([
  { id: 'top', title: '主机排行', visible: true },
  { id: 'trend', title: '主机趋势', visible: true }
])
const dragIndex = ref(null)
const layoutName = ref('')
const selectedLayout = ref('')
const layouts = ref([])

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const rangeLabel = computed(() => {
  const found = rangeOptions.find(opt => opt.value === rangeHours.value)
  return found ? found.label : `${rangeHours.value}h`
})

const getErrorMessage = (err, fallback) => {
  return err?.response?.data?.error || err?.response?.data?.message || err?.message || fallback
}

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

const queryCpu = (instance) => `100 - (avg by(instance) (irate(node_cpu_seconds_total{${buildSelector('mode="idle"', instance)}}[5m])) * 100)`

const queryMem = (instance) => `100 * (1 - (node_memory_MemAvailable_bytes{${buildSelector('', instance)}} / node_memory_MemTotal_bytes{${buildSelector('', instance)}}))`

const queryDisk = (instance) => `max by(instance) (100 - (node_filesystem_free_bytes{${buildSelector('fstype!="tmpfs",mountpoint="/"', instance)}} / node_filesystem_size_bytes{${buildSelector('fstype!="tmpfs",mountpoint="/"', instance)}}) * 100)`

const queryNet = (instance) => `sum by(instance) (rate(node_network_receive_bytes_total{${buildSelector('', instance)}}[5m]) + rate(node_network_transmit_bytes_total{${buildSelector('', instance)}}[5m])) / 1024 / 1024`

const queryLoad = (instance) => `node_load1{${buildSelector('', instance)}}`

const queryUptime = (instance) => `time() - node_boot_time_seconds{${buildSelector('', instance)}}`

const filteredInstances = computed(() => {
  if (!companyFilter.value && !envFilter.value) return instances.value
  const metaMap = new Map(instanceMeta.value.map(item => [item.instance, item]))
  return instances.value.filter(inst => {
    const meta = metaMap.get(inst)
    if (!meta) return false
    if (companyFilter.value && meta.com !== companyFilter.value) return false
    if (envFilter.value && meta.env !== envFilter.value) return false
    return true
  })
})

const resolveChartEl = (value) => {
  if (!value) return null
  if (Array.isArray(value)) {
    for (const item of value) {
      const resolved = resolveChartEl(item)
      if (resolved instanceof HTMLElement) return resolved
    }
    return null
  }
  if (value?.$el) return resolveChartEl(value.$el)
  if (value instanceof HTMLElement) return value
  return null
}

const ensureChartInstance = (instance, holderRef) => {
  const el = resolveChartEl(holderRef.value)
  if (!el) return instance
  if (el.clientWidth <= 0 || el.clientHeight <= 0) return instance
  if (instance && instance.getDom && instance.getDom() !== el) {
    instance.dispose()
    instance = null
  }
  try {
    if (!instance) {
      instance = echarts.getInstanceByDom(el) || echarts.init(el)
    }
  } catch {
    if (instance) {
      instance.dispose()
    }
    return null
  }
  return instance
}

const ensureCharts = () => {
  cpuChart = ensureChartInstance(cpuChart, cpuChartRef)
  memChart = ensureChartInstance(memChart, memChartRef)
  diskChart = ensureChartInstance(diskChart, diskChartRef)
  netChart = ensureChartInstance(netChart, netChartRef)
  topCpuChart = ensureChartInstance(topCpuChart, topCpuChartRef)
  topMemChart = ensureChartInstance(topMemChart, topMemChartRef)
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

const renderBarChart = (chart, title, items, valueKey, unit) => {
  if (!chart) return
  const labels = items.map(item => item.instance)
  const data = items.map(item => Number(item[valueKey] || 0))
  chart.setOption({
    title: { text: title, left: 'left', textStyle: { fontSize: 12 } },
    tooltip: { trigger: 'axis', valueFormatter: (val) => `${Number(val).toFixed(2)} ${unit}` },
    grid: { left: 120, right: 20, top: 40, bottom: 20 },
    xAxis: { type: 'value' },
    yAxis: { type: 'category', data: labels, axisLabel: { width: 180, overflow: 'truncate' } },
    series: [{ type: 'bar', data }]
  })
}

const renderTopCharts = () => {
  ensureCharts()
  if (!topCpuChart || !topMemChart) return
  const topCpu = [...filteredRows.value].sort((a, b) => Number(b.cpu || 0) - Number(a.cpu || 0)).slice(0, 10)
  const topMem = [...filteredRows.value].sort((a, b) => Number(b.memory || 0) - Number(a.memory || 0)).slice(0, 10)
  renderBarChart(topCpuChart, 'CPU Top 10 (%)', topCpu, 'cpu', '%')
  renderBarChart(topMemChart, '内存 Top 10 (%)', topMem, 'memory', '%')
}

const onDragStart = (idx) => {
  dragIndex.value = idx
}

const onDrop = (idx) => {
  if (dragIndex.value === null || dragIndex.value === idx) return
  const next = [...panels.value]
  const [moved] = next.splice(dragIndex.value, 1)
  next.splice(idx, 0, moved)
  panels.value = next
  dragIndex.value = null
  persistLayouts()
}

const layoutScope = 'hosts'
const layoutStorageKey = 'lazy_aiops_monitor_hosts_layouts'

const persistLayouts = () => {
  localStorage.setItem(layoutStorageKey, JSON.stringify(layouts.value))
}

const loadLocalLayouts = () => {
  try {
    const raw = localStorage.getItem(layoutStorageKey)
    layouts.value = raw ? JSON.parse(raw) : []
  } catch (e) {
    layouts.value = []
  }
}

const fetchLayouts = async () => {
  try {
    const res = await axios.get('/api/v1/monitor/dashboards', {
      headers: authHeaders(),
      params: { scope: layoutScope }
    })
    if (res.data?.code === 0) {
      layouts.value = (res.data.data || []).map(item => ({
        id: item.id,
        name: item.name,
        payload: item.payload,
        created_by: item.created_by,
        updated_at: item.updated_at
      }))
      persistLayouts()
      return
    }
  } catch (err) {
    // fallback to local
  }
  loadLocalLayouts()
}

const saveLayout = async () => {
  const name = layoutName.value.trim()
  if (!name) {
    ElMessage.warning('请输入模板名称')
    return
  }
  const payload = {
    name,
    panels: panels.value,
    filters: {
      company: companyFilter.value,
      env: envFilter.value,
      instance: instanceFilter.value,
      rangeHours: rangeHours.value
    }
  }
  const existing = layouts.value.find(item => item.name === name)
  const data = {
    name,
    scope: layoutScope,
    payload: JSON.stringify(payload)
  }
  try {
    if (existing?.id) {
      await axios.put(`/api/v1/monitor/dashboards/${existing.id}`, data, { headers: authHeaders() })
    } else {
      await axios.post('/api/v1/monitor/dashboards', data, { headers: authHeaders() })
    }
    await fetchLayouts()
    selectedLayout.value = name
    ElMessage.success('模板已保存')
  } catch (err) {
    const next = layouts.value.filter(item => item.name !== name)
    next.push({ name, payload })
    layouts.value = next
    persistLayouts()
    selectedLayout.value = name
    ElMessage.success('模板已保存(本地)')
  }
}

const applyLayout = () => {
  const selected = layouts.value.find(item => item.name === selectedLayout.value)
  if (!selected) return
  let rawPayload = selected.payload || selected
  if (typeof rawPayload === 'string') {
    try {
      rawPayload = JSON.parse(rawPayload)
    } catch (e) {
      rawPayload = selected
    }
  }
  panels.value = rawPayload.panels || panels.value
  const filters = rawPayload.filters || {}
  companyFilter.value = filters.company || ''
  envFilter.value = filters.env || ''
  instanceFilter.value = filters.instance || ''
  rangeHours.value = Number(filters.rangeHours) || 1
  fetchTable()
  fetchCharts()
}

const deleteLayout = async () => {
  if (!selectedLayout.value) return
  const selected = layouts.value.find(item => item.name === selectedLayout.value)
  try {
    if (selected?.id) {
      await axios.delete(`/api/v1/monitor/dashboards/${selected.id}`, { headers: authHeaders() })
    }
    await fetchLayouts()
    selectedLayout.value = ''
    ElMessage.success('模板已删除')
  } catch (err) {
    layouts.value = layouts.value.filter(item => item.name !== selectedLayout.value)
    persistLayouts()
    selectedLayout.value = ''
    ElMessage.success('模板已删除(本地)')
  }
}

const calcStep = (hours) => {
  if (hours <= 1) return 30
  if (hours <= 6) return 60
  if (hours <= 24) return 120
  return 300
}

const fetchInstances = async () => {
  try {
    let res = await fetchProm('up{job=~"node.*"}')
    if (!res.length) {
      res = await fetchProm('up{job=~"node.*"}')
    }
    const instSet = new Set()
    const comSet = new Set()
    const envSet = new Set()
    const metaMap = new Map()
    res.forEach(item => {
      const m = item.metric || {}
      if (m.instance) {
        instSet.add(m.instance)
        metaMap.set(m.instance, { instance: m.instance, com: m.com || '', env: m.env || '' })
      }
      if (m.com) comSet.add(m.com)
      if (m.env) envSet.add(m.env)
    })
    instances.value = Array.from(instSet)
    companies.value = Array.from(comSet)
    envs.value = Array.from(envSet)
    instanceMeta.value = Array.from(metaMap.values())
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
    const settled = await Promise.allSettled([
      fetchProm(queryCpu('')),
      fetchProm(queryMem('')),
      fetchProm(queryDisk('')),
      fetchProm(queryNet('')),
      fetchProm(queryLoad('')),
      fetchProm(queryUptime(''))
    ])
    const cpuRes = settled[0].status === 'fulfilled' ? settled[0].value : []
    const memRes = settled[1].status === 'fulfilled' ? settled[1].value : []
    const diskRes = settled[2].status === 'fulfilled' ? settled[2].value : []
    const netRes = settled[3].status === 'fulfilled' ? settled[3].value : []
    const loadRes = settled[4].status === 'fulfilled' ? settled[4].value : []
    const upRes = settled[5].status === 'fulfilled' ? settled[5].value : []
    const metricNames = ['CPU', '内存', '磁盘', '网络', '负载', '运行时长']
    const failedParts = settled
      .map((item, idx) => (item.status === 'rejected' ? `${metricNames[idx]}(${getErrorMessage(item.reason, '查询失败')})` : ''))
      .filter(Boolean)
    if (failedParts.length && failedParts.length < settled.length) {
      ElMessage.warning(`部分主机指标查询失败：${failedParts.join('；')}`)
    }
    if (failedParts.length === settled.length) {
      throw new Error(failedParts.join('；'))
    }
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
    renderTopCharts()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '主机指标获取失败'))
  } finally {
    loading.value = false
  }
}

const fetchCharts = async () => {
  chartLoading.value = true
  try {
    await nextTick()
    ensureCharts()
    const end = Math.floor(Date.now() / 1000)
    const start = end - (rangeHours.value * 3600)
    const step = calcStep(rangeHours.value)
    const inst = instanceFilter.value
    const settled = await Promise.allSettled([
      fetchPromRange(queryCpu(inst), start, end, step),
      fetchPromRange(queryMem(inst), start, end, step),
      fetchPromRange(queryDisk(inst), start, end, step),
      fetchPromRange(queryNet(inst), start, end, step)
    ])
    const cpuRes = settled[0].status === 'fulfilled' ? settled[0].value : []
    const memRes = settled[1].status === 'fulfilled' ? settled[1].value : []
    const diskRes = settled[2].status === 'fulfilled' ? settled[2].value : []
    const netRes = settled[3].status === 'fulfilled' ? settled[3].value : []
    const metricNames = ['CPU趋势', '内存趋势', '磁盘趋势', '网络趋势']
    const failedParts = settled
      .map((item, idx) => (item.status === 'rejected' ? `${metricNames[idx]}(${getErrorMessage(item.reason, '查询失败')})` : ''))
      .filter(Boolean)
    if (failedParts.length && failedParts.length < settled.length) {
      ElMessage.warning(`部分趋势查询失败：${failedParts.join('；')}`)
    }
    if (failedParts.length === settled.length) {
      throw new Error(failedParts.join('；'))
    }
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
    ElMessage.error(getErrorMessage(err, '趋势图加载失败'))
  } finally {
    chartLoading.value = false
  }
}

const refreshAll = async () => {
  await nextTick()
  await fetchInstances()
  await fetchTable()
  await fetchCharts()
}

const selectInstance = (row) => {
  if (row?.instance) instanceFilter.value = row.instance
}

const progressColor = (val) => {
  if (val >= 80) return '#F56C6C'
  if (val >= 60) return '#E6A23C'
  return '#67C23A'
}

const filteredRows = computed(() => {
  const key = keyword.value.trim().toLowerCase()
  const metaMap = new Map(instanceMeta.value.map(item => [item.instance, item]))
  return rows.value.filter(r => {
    if (instanceFilter.value && r.instance !== instanceFilter.value) return false
    if (companyFilter.value || envFilter.value) {
      const meta = metaMap.get(r.instance)
      if (companyFilter.value && meta?.com !== companyFilter.value) return false
      if (envFilter.value && meta?.env !== envFilter.value) return false
    }
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

const healthSummary = computed(() => {
  let ok = 0
  let warn = 0
  let critical = 0
  filteredRows.value.forEach((r) => {
    const cpu = Number(r.cpu || 0)
    const mem = Number(r.memory || 0)
    const disk = Number(r.disk || 0)
    if (cpu >= 85 || mem >= 90 || disk >= 90) {
      critical++
    } else if (cpu >= 70 || mem >= 80 || disk >= 80) {
      warn++
    } else {
      ok++
    }
  })
  return { ok, warn, critical }
})

const selectedHost = computed(() => {
  if (instanceFilter.value) {
    const found = rows.value.find(item => item.instance === instanceFilter.value)
    if (found) return found
  }
  return filteredRows.value[0] || {}
})

watch(rangeHours, fetchCharts)
watch(instanceFilter, fetchCharts)
watch(filteredRows, renderTopCharts)
watch([companyFilter, envFilter], () => {
  if (instanceFilter.value && !filteredInstances.value.includes(instanceFilter.value)) {
    instanceFilter.value = filteredInstances.value[0] || ''
  }
  fetchTable()
  fetchCharts()
})

onMounted(() => {
  fetchLayouts()
  refreshAll()
})

onBeforeUnmount(() => {
  if (cpuChart) cpuChart.dispose()
  if (memChart) memChart.dispose()
  if (diskChart) diskChart.dispose()
  if (netChart) netChart.dispose()
  if (topCpuChart) topCpuChart.dispose()
  if (topMemChart) topMemChart.dispose()
  cpuChart = null
  memChart = null
  diskChart = null
  netChart = null
  topCpuChart = null
  topMemChart = null
})
</script>

<style scoped>
.page-card {
  max-width: 1180px;
  margin: 0 auto;
  border-radius: 20px;
  border: 1px solid rgba(15, 23, 42, 0.08);
  background: linear-gradient(180deg, #f8fbff 0%, #f6f8fc 100%);
  font-family: -apple-system, BlinkMacSystemFont, "SF Pro Text", "PingFang SC", "Helvetica Neue", Arial, sans-serif;
}
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 16px; }
.page-header h2 { margin: 0; font-weight: 600; letter-spacing: -0.01em; }
.page-desc { color: #606266; margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; }
.motion-up { opacity: 0; transform: translateY(10px); animation: fade-up 0.42s cubic-bezier(0.21, 1, 0.35, 1) forwards; }
.delay-1 { animation-delay: 30ms; }
.delay-2 { animation-delay: 70ms; }
.delay-3 { animation-delay: 110ms; }
.delay-4 { animation-delay: 150ms; }
.delay-5 { animation-delay: 190ms; }
.delay-6 { animation-delay: 230ms; }
.delay-7 { animation-delay: 270ms; }
.delay-8 { animation-delay: 310ms; }
.filter-bar {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 12px;
  padding: 10px;
  border: 1px solid rgba(15, 23, 42, 0.08);
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.72);
  backdrop-filter: blur(10px);
}
.layout-bar {
  margin-bottom: 12px;
  padding: 10px 12px;
  background: rgba(255, 255, 255, 0.72);
  border: 1px solid rgba(15, 23, 42, 0.08);
  border-radius: 14px;
  backdrop-filter: blur(10px);
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
}
.layout-title { font-weight: 600; color: #606266; }
.layout-items { display: flex; gap: 8px; flex-wrap: wrap; }
.layout-item { display: flex; align-items: center; gap: 6px; padding: 5px 8px; border: 1px dashed #dcdfe6; border-radius: 10px; background: #fff; cursor: grab; transition: box-shadow 0.2s ease, transform 0.2s ease; }
.layout-item:hover { transform: translateY(-1px); box-shadow: 0 6px 14px rgba(15, 23, 42, 0.08); }
.drag-handle { font-weight: 700; color: #909399; }
.layout-actions { display: flex; gap: 8px; flex-wrap: wrap; align-items: center; }
.summary-row { margin-top: 8px; }
.health-row { margin-top: 8px; }
.summary-card { transition: transform 0.22s ease, box-shadow 0.22s ease; }
.summary-card:hover { transform: translateY(-2px); box-shadow: 0 10px 18px rgba(15, 23, 42, 0.08); }
.panel-card {
  margin-top: 16px;
  border-radius: 16px;
  border: 1px solid rgba(15, 23, 42, 0.08);
  background: rgba(255, 255, 255, 0.82);
  backdrop-filter: blur(12px);
  transition: transform 0.24s ease, box-shadow 0.24s ease, border-color 0.2s ease;
}
.panel-card:hover { transform: translateY(-2px); box-shadow: 0 12px 24px rgba(15, 23, 42, 0.08); }
.panel-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 8px; }
.panel-desc { color: #909399; font-size: 12px; margin: 4px 0 0; }
.panel-actions { display: flex; align-items: center; gap: 8px; }
.chart-box { width: 100%; height: 260px; transition: filter 0.28s ease, transform 0.28s ease; }
.panel-card:hover .chart-box { filter: saturate(1.04); }
.chart-row { margin-top: 8px; }
.card-title { color: #909399; font-size: 12px; }
.card-value { font-size: 20px; font-weight: 600; margin-top: 6px; }
.card-sub { margin-top: 6px; font-weight: 600; color: #303133; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.card-mini { margin-top: 4px; font-size: 12px; color: #606266; }
.health-ok { color: #67C23A; }
.health-warn { color: #E6A23C; }
.health-critical { color: #F56C6C; }
.w-52 { width: 220px; }
.w-40 { width: 140px; }
:deep(.el-card) {
  border-radius: 16px;
  border: 1px solid rgba(15, 23, 42, 0.08);
  transition: box-shadow 0.24s ease;
}
:deep(.el-card:hover) { box-shadow: 0 12px 24px rgba(15, 23, 42, 0.06); }
:deep(.el-button) { border-radius: 10px; transition: all 0.2s ease; }
:deep(.el-button:hover) { transform: translateY(-1px); }
:deep(.el-input__wrapper),
:deep(.el-select__wrapper) {
  border-radius: 10px;
  box-shadow: 0 0 0 1px rgba(148, 163, 184, 0.18) inset;
}
:deep(.el-table__body tr > td) { transition: background-color 0.2s ease; }
:deep(.el-table__body tr:hover > td) { background: #f7fbff !important; }
@keyframes fade-up {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}
@media (prefers-reduced-motion: reduce) {
  .motion-up { opacity: 1; transform: none; animation: none; }
  .summary-card,
  .panel-card,
  .chart-box { transition: none; }
}
</style>
