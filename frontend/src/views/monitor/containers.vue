<template>
  <el-card class="page-card">
    <div class="page-header motion-up delay-1">
      <div>
        <h2>容器监控</h2>
        <p class="page-desc">基于 Prometheus/cAdvisor 的容器指标（CPU/内存）。</p>
      </div>
      <div class="page-actions">
        <el-button icon="Refresh" @click="fetchMetrics">刷新</el-button>
      </div>
    </div>

    <div class="meta-row motion-up delay-2" v-if="lastRefresh">
      <span class="meta-text">刷新时间：{{ lastRefresh }}</span>
    </div>

    <el-alert class="motion-up delay-2" type="info" :closable="false" show-icon>
      如果没有数据，请确认 Prometheus 已采集 cAdvisor 或 kubelet/cAdvisor 指标。
    </el-alert>

    <div class="filter-bar motion-up delay-3">
      <el-select v-model="companyFilter" placeholder="公司" class="w-40" clearable>
        <el-option v-for="item in companies" :key="item" :label="item" :value="item" />
      </el-select>
      <el-select v-model="envFilter" placeholder="环境" class="w-40" clearable>
        <el-option v-for="item in envs" :key="item" :label="item" :value="item" />
      </el-select>
      <el-select v-model="stackFilter" placeholder="Stack/命名空间" class="w-52" clearable>
        <el-option v-for="item in stacks" :key="item" :label="item" :value="item" />
      </el-select>
      <el-select v-model="instanceFilter" placeholder="主机" class="w-52" clearable>
        <el-option v-for="inst in filteredInstanceOptions" :key="inst" :label="inst" :value="inst" />
      </el-select>
      <el-input v-model="keyword" placeholder="搜索容器/镜像/节点" class="w-52" clearable />
      <el-select v-model="topN" class="w-40">
        <el-option label="Top 20" :value="20" />
        <el-option label="Top 50" :value="50" />
        <el-option label="Top 100" :value="100" />
      </el-select>
    </div>

    <div class="layout-bar motion-up delay-4">
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

    <el-row :gutter="16" class="summary-row motion-up delay-5">
      <el-col :span="6"><el-card class="summary-card"><div class="card-title">容器数</div><div class="card-value">{{ stats.total }}</div></el-card></el-col>
      <el-col :span="6"><el-card class="summary-card"><div class="card-title">CPU Top</div><div class="card-value">{{ stats.maxCpu }}</div></el-card></el-col>
      <el-col :span="6"><el-card class="summary-card"><div class="card-title">内存 Top(MiB)</div><div class="card-value">{{ stats.maxMem }}</div></el-card></el-col>
    </el-row>

    <template v-for="panel in panels" :key="panel.id">
      <el-card v-if="panel.id === 'top' && panel.visible" class="panel-card motion-up delay-6">
        <div class="panel-header">
          <div>
            <h3>Top 排行</h3>
            <p class="panel-desc">当前 Top 容器 CPU / 内存排行。</p>
          </div>
          <div class="panel-actions">
            <el-button size="small" @click="renderTopCharts">刷新排行</el-button>
          </div>
        </div>
        <el-row :gutter="16">
          <el-col :span="12">
            <div ref="topCpuChartRef" class="chart-box"></div>
          </el-col>
          <el-col :span="12">
            <div ref="topMemChartRef" class="chart-box"></div>
          </el-col>
        </el-row>
      </el-card>

      <el-card v-if="panel.id === 'trend' && panel.visible" class="panel-card motion-up delay-7" v-loading="chartLoading">
        <div class="panel-header">
          <div>
            <h3>趋势看板</h3>
            <p class="panel-desc">Top 容器 CPU/内存趋势（近 {{ rangeLabel }}）。</p>
          </div>
          <div class="panel-actions">
            <el-radio-group v-model="rangeHours" size="small">
              <el-radio-button v-for="opt in rangeOptions" :key="opt.value" :label="opt.value">
                {{ opt.label }}
              </el-radio-button>
            </el-radio-group>
            <el-button size="small" @click="fetchCharts">刷新趋势</el-button>
          </div>
        </div>
        <el-row :gutter="16">
          <el-col :span="12">
            <div ref="cpuChartRef" class="chart-box"></div>
          </el-col>
          <el-col :span="12">
            <div ref="memChartRef" class="chart-box"></div>
          </el-col>
        </el-row>
      </el-card>
    </template>

    <el-table class="motion-up delay-8" :data="filteredRows" v-loading="loading" style="width: 100%; margin-top: 12px">
      <el-table-column prop="container" label="容器" min-width="220" />
      <el-table-column prop="image" label="镜像" min-width="200" />
      <el-table-column prop="instance" label="节点" min-width="160" />
      <el-table-column prop="cpu" label="CPU(核)" width="120" sortable />
      <el-table-column prop="memory" label="内存(MiB)" width="140" sortable />
    </el-table>
  </el-card>
</template>

<script setup>
import { ref, onMounted, computed, watch, onBeforeUnmount } from 'vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import * as echarts from 'echarts'

const rows = ref([])
const loading = ref(false)
const keyword = ref('')
const topN = ref(50)
const lastRefresh = ref('')
const instanceFilter = ref('')
const instances = ref([])
const filterMeta = ref([])
const companyFilter = ref('')
const envFilter = ref('')
const stackFilter = ref('')
const companies = ref([])
const envs = ref([])
const stacks = ref([])
const rangeHours = ref(1)
const rangeOptions = [
  { label: '1h', value: 1 },
  { label: '6h', value: 6 },
  { label: '24h', value: 24 },
  { label: '7d', value: 168 }
]
const chartLoading = ref(false)
const cpuChartRef = ref(null)
const memChartRef = ref(null)
const topCpuChartRef = ref(null)
const topMemChartRef = ref(null)
let cpuChart = null
let memChart = null
let topCpuChart = null
let topMemChart = null
const panels = ref([
  { id: 'top', title: 'Top 排行', visible: true },
  { id: 'trend', title: '趋势看板', visible: true }
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

const selectorParts = () => {
  const parts = ['id!="/"', 'image!=""', 'name!=""']
  if (companyFilter.value) parts.push(`com="${companyFilter.value}"`)
  if (envFilter.value) parts.push(`env="${envFilter.value}"`)
  if (stackFilter.value) parts.push(`container_label_com_docker_stack_namespace="${stackFilter.value}"`)
  if (instanceFilter.value) parts.push(`instance="${instanceFilter.value}"`)
  return parts.join(',')
}

const filteredInstanceOptions = computed(() => {
  if (!companyFilter.value && !envFilter.value && !stackFilter.value) return instances.value
  const metaMap = new Map(filterMeta.value.map(item => [item.instance, item]))
  return instances.value.filter(inst => {
    const meta = metaMap.get(inst)
    if (!meta) return false
    if (companyFilter.value && meta.com !== companyFilter.value) return false
    if (envFilter.value && meta.env !== envFilter.value) return false
    if (stackFilter.value && meta.stack !== stackFilter.value) return false
    return true
  })
})

const cpuQuery = (n) => `topk(${n},
  sum by (name, instance, image) (
    rate(container_cpu_usage_seconds_total{${selectorParts()}}[5m])
  )
)`

const memQuery = (n) => `topk(${n},
  sum by (name, instance, image) (
    container_memory_working_set_bytes{${selectorParts()}}
  )
)`

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

const pickContainer = (m) => (
  m.container ||
  m.name ||
  m.container_label_com_docker_swarm_service_name ||
  m.container_label_com_docker_swarm_task_name ||
  m.container_label_com_docker_container_name ||
  '-'
)

const pickImage = (m) => (
  m.image || '-'
)

const pickInstance = (m) => (
  m.instance || '-'
)

const fetchFilterOptions = async () => {
  try {
    const res = await fetchProm('count by(com, env, instance, container_label_com_docker_stack_namespace) (container_cpu_usage_seconds_total{image!=""})')
    const instSet = new Set()
    const comSet = new Set()
    const envSet = new Set()
    const stackSet = new Set()
    const metaMap = new Map()
    res.forEach(item => {
      const m = item.metric || {}
      if (m.instance) instSet.add(m.instance)
      if (m.com) comSet.add(m.com)
      if (m.env) envSet.add(m.env)
      if (m.container_label_com_docker_stack_namespace) stackSet.add(m.container_label_com_docker_stack_namespace)
      if (m.instance) {
        metaMap.set(m.instance, {
          instance: m.instance,
          com: m.com || '',
          env: m.env || '',
          stack: m.container_label_com_docker_stack_namespace || ''
        })
      }
    })
    instances.value = Array.from(instSet)
    companies.value = Array.from(comSet)
    envs.value = Array.from(envSet)
    stacks.value = Array.from(stackSet)
    filterMeta.value = Array.from(metaMap.values())
    if (!instanceFilter.value && instances.value.length) {
      instanceFilter.value = instances.value[0]
    }
  } catch (err) {
    ElMessage.error('获取过滤项失败')
  }
}

const fetchMetrics = async () => {
  loading.value = true
  try {
    const settled = await Promise.allSettled([fetchProm(cpuQuery(topN.value)), fetchProm(memQuery(topN.value))])
    const cpuRes = settled[0].status === 'fulfilled' ? settled[0].value : []
    const memRes = settled[1].status === 'fulfilled' ? settled[1].value : []
    const failedParts = []
    if (settled[0].status === 'rejected') failedParts.push(`CPU(${getErrorMessage(settled[0].reason, '查询失败')})`)
    if (settled[1].status === 'rejected') failedParts.push(`内存(${getErrorMessage(settled[1].reason, '查询失败')})`)
    if (failedParts.length && failedParts.length < 2) {
      ElMessage.warning(`部分指标查询失败：${failedParts.join('；')}`)
    }
    if (failedParts.length === 2) {
      throw new Error(failedParts.join('；'))
    }
    const map = {}

    cpuRes.forEach((item) => {
      const m = item.metric || {}
      const container = pickContainer(m)
      const image = pickImage(m)
      const instance = pickInstance(m)
      const key = `${container}|${instance}|${image}`
      map[key] = map[key] || {
        container,
        image,
        instance,
        cpu: 0,
        memory: 0
      }
      map[key].cpu = Number(item.value?.[1] || 0).toFixed(3)
    })
    memRes.forEach((item) => {
      const m = item.metric || {}
      const container = pickContainer(m)
      const image = pickImage(m)
      const instance = pickInstance(m)
      const key = `${container}|${instance}|${image}`
      map[key] = map[key] || {
        container,
        image,
        instance,
        cpu: 0,
        memory: 0
      }
      map[key].memory = (Number(item.value?.[1] || 0) / 1024 / 1024).toFixed(1)
    })
    rows.value = Object.values(map)
    lastRefresh.value = new Date().toLocaleString()
    if (!rows.value.length) {
      ElMessage.warning('未获取到容器指标，请确认 Prometheus 已采集 cAdvisor 指标')
    }
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '拉取容器指标失败'))
  } finally {
    loading.value = false
  }
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

const layoutScope = 'containers'
const layoutStorageKey = 'lazy_aiops_monitor_containers_layouts'

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
      stack: stackFilter.value,
      instance: instanceFilter.value,
      topN: topN.value,
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
  stackFilter.value = filters.stack || ''
  instanceFilter.value = filters.instance || ''
  topN.value = Number(filters.topN) || 50
  rangeHours.value = Number(filters.rangeHours) || 1
  fetchMetrics()
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

const buildSeries = (result, transform) => {
  return result.map((item) => {
    const m = item.metric || {}
    const name = pickContainer(m)
    const values = (item.values || []).map(v => transform(Number(v[1])))
    return { name, type: 'line', showSymbol: false, data: values }
  })
}

const resolveChartEl = (value) => {
  if (!value) return null
  if (Array.isArray(value)) return resolveChartEl(value[0])
  if (value?.$el) return value.$el
  return value
}

const ensureChartInstance = (instance, holderRef) => {
  const el = resolveChartEl(holderRef.value)
  if (!el) return instance
  if (instance && instance.getDom && instance.getDom() !== el) {
    instance.dispose()
    instance = null
  }
  if (!instance) instance = echarts.init(el)
  return instance
}

const ensureCharts = () => {
  cpuChart = ensureChartInstance(cpuChart, cpuChartRef)
  memChart = ensureChartInstance(memChart, memChartRef)
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
    xAxis: { type: 'category', data: labels, axisLabel: { showMaxLabel: true } },
    yAxis: { type: 'value', axisLabel: { formatter: (v) => `${v}` } },
    series
  })
}

const renderBarChart = (chart, title, items, valueKey, unit) => {
  if (!chart) return
  const labels = items.map(item => item.container)
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
  const topCpu = [...filteredRows.value]
    .sort((a, b) => Number(b.cpu || 0) - Number(a.cpu || 0))
    .slice(0, 10)
  const topMem = [...filteredRows.value]
    .sort((a, b) => Number(b.memory || 0) - Number(a.memory || 0))
    .slice(0, 10)
  renderBarChart(topCpuChart, 'CPU Top 10 (核)', topCpu, 'cpu', '核')
  renderBarChart(topMemChart, '内存 Top 10 (MiB)', topMem, 'memory', 'MiB')
}

const calcStep = (hours) => {
  if (hours <= 1) return 30
  if (hours <= 6) return 60
  if (hours <= 24) return 120
  return 300
}

const fetchCharts = async () => {
  chartLoading.value = true
  try {
    ensureCharts()
    const end = Math.floor(Date.now() / 1000)
    const start = end - (rangeHours.value * 3600)
    const step = calcStep(rangeHours.value)
    const settled = await Promise.allSettled([
      fetchPromRange(cpuQuery(5), start, end, step),
      fetchPromRange(memQuery(5), start, end, step)
    ])
    const cpuSeries = settled[0].status === 'fulfilled' ? settled[0].value : []
    const memSeries = settled[1].status === 'fulfilled' ? settled[1].value : []
    const failedParts = []
    if (settled[0].status === 'rejected') failedParts.push(`CPU趋势(${getErrorMessage(settled[0].reason, '查询失败')})`)
    if (settled[1].status === 'rejected') failedParts.push(`内存趋势(${getErrorMessage(settled[1].reason, '查询失败')})`)
    if (failedParts.length && failedParts.length < 2) {
      ElMessage.warning(`部分趋势查询失败：${failedParts.join('；')}`)
    }
    if (failedParts.length === 2) {
      throw new Error(failedParts.join('；'))
    }
    const labels = (cpuSeries[0]?.values || memSeries[0]?.values || []).map(v =>
      new Date(Number(v[0]) * 1000).toLocaleTimeString()
    )
    const cpuLines = buildSeries(cpuSeries, (v) => v)
    const memLines = buildSeries(memSeries, (v) => v / 1024 / 1024)
    renderChart(cpuChart, 'CPU Top 5 (核)', labels, cpuLines, '核')
    renderChart(memChart, '内存 Top 5 (MiB)', labels, memLines, 'MiB')
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '趋势图加载失败'))
  } finally {
    chartLoading.value = false
  }
}

const filteredRows = computed(() => {
  const key = keyword.value.trim().toLowerCase()
  const inst = instanceFilter.value
  if (!key) {
    return inst ? rows.value.filter(r => r.instance === inst) : rows.value
  }
  return rows.value.filter(r =>
    (!inst || r.instance === inst) &&
    (
      (r.container || '').toLowerCase().includes(key) ||
      (r.image || '').toLowerCase().includes(key) ||
      (r.instance || '').toLowerCase().includes(key)
    )
  )
})

const stats = computed(() => {
  const total = filteredRows.value.length
  const maxCpu = filteredRows.value.reduce((max, r) => Math.max(max, Number(r.cpu || 0)), 0).toFixed(3)
  const maxMem = filteredRows.value.reduce((max, r) => Math.max(max, Number(r.memory || 0)), 0).toFixed(1)
  return { total, maxCpu, maxMem }
})

watch(rangeHours, fetchCharts)
watch(instanceFilter, () => {
  fetchMetrics()
  fetchCharts()
})
watch([companyFilter, envFilter, stackFilter], () => {
  if (instanceFilter.value && !filteredInstanceOptions.value.includes(instanceFilter.value)) {
    instanceFilter.value = filteredInstanceOptions.value[0] || ''
  }
  fetchMetrics()
  fetchCharts()
})
watch(filteredRows, () => {
  renderTopCharts()
})

onMounted(() => {
  fetchLayouts()
  fetchFilterOptions()
  fetchMetrics()
  fetchCharts()
  renderTopCharts()
})

onBeforeUnmount(() => {
  if (cpuChart) cpuChart.dispose()
  if (memChart) memChart.dispose()
  if (topCpuChart) topCpuChart.dispose()
  if (topMemChart) topMemChart.dispose()
  cpuChart = null
  memChart = null
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
  margin-top: 12px;
  padding: 10px;
  border: 1px solid rgba(15, 23, 42, 0.08);
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.72);
  backdrop-filter: blur(10px);
}
.layout-bar {
  margin-top: 12px;
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
.summary-row { margin-top: 12px; }
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
.meta-row { display: flex; align-items: center; margin-top: 8px; color: #606266; font-size: 12px; }
.card-title { color: #909399; font-size: 12px; }
.card-value { font-size: 20px; font-weight: 600; margin-top: 6px; }
.w-52 { width: 220px; }
.w-40 { width: 140px; }
:deep(.el-alert) {
  margin-top: 12px;
  border-radius: 12px;
  border: 1px solid rgba(15, 23, 42, 0.06);
  background: rgba(255, 255, 255, 0.78);
}
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
