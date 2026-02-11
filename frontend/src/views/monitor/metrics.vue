<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>指标采集</h2>
        <p class="page-desc">Prometheus 查询与实时系统指标。</p>
      </div>
      <div class="page-actions">
        <el-button icon="Refresh" @click="fetchRealtime">刷新实时指标</el-button>
      </div>
    </div>

    <el-row :gutter="16" class="metric-cards">
      <el-col :span="6"><el-card><div class="card-title">CPU</div><div class="card-value">{{ realtime.cpu }}%</div></el-card></el-col>
      <el-col :span="6"><el-card><div class="card-title">内存</div><div class="card-value">{{ realtime.memory }}%</div></el-card></el-col>
      <el-col :span="6"><el-card><div class="card-title">磁盘</div><div class="card-value">{{ realtime.disk }}%</div></el-card></el-col>
      <el-col :span="6"><el-card><div class="card-title">网络(MB/s)</div><div class="card-value">{{ realtime.network }}</div></el-card></el-col>
    </el-row>
    <div class="meta-row">
      <el-tag v-if="promInfo.version" type="info">Prometheus v{{ promInfo.version }}</el-tag>
      <span class="meta-text" v-if="promInfo.uptime">运行时长：{{ promInfo.uptime }}</span>
      <span class="meta-text" v-if="currentProm">当前：{{ currentProm }}</span>
    </div>

    <el-divider />

    <h3 class="section-title">Prometheus 接入</h3>
    <div class="query-bar">
      <el-input v-model="form.name" placeholder="名称" class="w-40" />
      <el-input v-model="form.prometheus_url" placeholder="Prometheus 地址，例如 http://10.0.0.1:9090" class="w-52" />
      <el-select v-model="form.auth_type" class="w-40">
        <el-option label="无认证" value="none" />
        <el-option label="Basic" value="basic" />
        <el-option label="Bearer" value="bearer" />
      </el-select>
      <el-input v-if="form.auth_type === 'basic'" v-model="form.username" placeholder="用户名" class="w-40" />
      <el-input v-if="form.auth_type === 'basic'" v-model="form.password" placeholder="密码" class="w-40" show-password />
      <el-input v-if="form.auth_type === 'bearer'" v-model="form.token" placeholder="Token" class="w-52" />
      <el-button type="primary" @click="saveSetting">{{ form.id ? '更新' : '新增' }}</el-button>
      <el-button @click="resetForm">清空</el-button>
    </div>

    <el-table :data="settings" size="small" style="width: 100%; margin-top: 12px" :row-class-name="settingsRowClass">
      <el-table-column prop="name" label="名称" width="160" />
      <el-table-column prop="prometheus_url" label="地址" min-width="240" />
      <el-table-column prop="auth_type" label="认证" width="120" />
      <el-table-column prop="active" label="当前" width="90">
        <template #default="scope">
          <el-tag v-if="scope.row.active" type="success">当前</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="260">
        <template #default="scope">
          <el-space size="8">
            <el-button size="small" @click="editSetting(scope.row)">编辑</el-button>
            <el-button size="small" type="success" plain @click="activateSetting(scope.row)">设为当前</el-button>
            <el-button size="small" type="info" plain @click="testSetting(scope.row)">测试</el-button>
            <el-button size="small" type="danger" plain @click="deleteSetting(scope.row)">删除</el-button>
          </el-space>
        </template>
      </el-table-column>
    </el-table>

    <el-divider />

    <h3 class="section-title">Prometheus 查询</h3>
    <div class="query-bar">
      <el-input v-model="query" placeholder="例如: up" class="w-52" />
      <el-select v-model="mode" class="w-40">
        <el-option label="即时" value="instant" />
        <el-option label="范围" value="range" />
      </el-select>
      <el-input v-model="rangeStart" placeholder="start (unix sec)" class="w-52" />
      <el-input v-model="rangeEnd" placeholder="end (unix sec)" class="w-52" />
      <el-input v-model="rangeStep" placeholder="step (sec)" class="w-40" />
      <el-button type="primary" @click="runQuery">查询</el-button>
    </div>
    <div class="template-bar">
      <el-tag
        v-for="tpl in templates"
        :key="tpl.label"
        class="template-tag"
        @click="applyTemplate(tpl)"
      >
        {{ tpl.label }}
      </el-tag>
    </div>

    <el-tabs v-model="activeTab">
      <el-tab-pane label="图表" name="chart">
        <div ref="chartRef" class="chart-box"></div>
      </el-tab-pane>
      <el-tab-pane label="历史" name="history">
        <div class="history-bar">
          <el-checkbox v-model="onlyFavorite" @change="fetchHistory">只看收藏</el-checkbox>
        </div>
        <el-table :data="history" size="small" style="width: 100%">
          <el-table-column prop="name" label="名称" width="140">
            <template #default="scope">
              <el-input v-model="scope.row.name" size="small" @change="saveName(scope.row)" />
            </template>
          </el-table-column>
          <el-table-column prop="time" label="时间" width="160" />
          <el-table-column prop="mode" label="类型" width="100" />
          <el-table-column prop="query" label="查询" min-width="260" />
          <el-table-column prop="range" label="范围" min-width="200" />
          <el-table-column prop="favorite" label="收藏" width="90">
            <template #default="scope">
              <el-switch v-model="scope.row.favorite" @change="toggleFavorite(scope.row)" />
            </template>
          </el-table-column>
          <el-table-column label="操作" width="120">
            <template #default="scope">
              <el-button size="small" @click="reRun(scope.row)">重跑</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
      <el-tab-pane label="原始结果" name="raw">
        <el-input v-model="result" type="textarea" :rows="18" readonly />
      </el-tab-pane>
    </el-tabs>
  </el-card>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount } from 'vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import * as echarts from 'echarts'

const realtime = ref({ cpu: 0, memory: 0, disk: 0, network: 0 })
const query = ref('up')
const mode = ref('instant')
const rangeStart = ref('')
const rangeEnd = ref('')
const rangeStep = ref('30')
const result = ref('')
const settings = ref([])
const form = ref({
  id: '',
  name: '',
  prometheus_url: '',
  auth_type: 'none',
  username: '',
  password: '',
  token: ''
})
const promInfo = ref({ version: '', uptime: '' })
const currentProm = ref('')
const activeTab = ref('chart')
const chartRef = ref(null)
let chartInstance = null
const templates = ref([
  { label: 'CPU使用率', mode: 'instant', query: '100 - (avg by(instance) (irate(node_cpu_seconds_total{mode=\"idle\"}[5m])) * 100)' },
  { label: '内存使用率', mode: 'instant', query: '(1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100' },
  { label: '磁盘使用率', mode: 'instant', query: '(1 - (node_filesystem_free_bytes{fstype!=\"tmpfs\"} / node_filesystem_size_bytes{fstype!=\"tmpfs\"})) * 100' },
  { label: '节点在线', mode: 'instant', query: 'up' },
  { label: 'CPU趋势', mode: 'range', query: '100 - (avg by(instance) (irate(node_cpu_seconds_total{mode=\"idle\"}[5m])) * 100)' },
  { label: '内存趋势', mode: 'range', query: '(1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100' },
  { label: '网络入流量', mode: 'range', query: 'sum by(instance) (rate(node_network_receive_bytes_total[5m]))' }
])
const history = ref([])
const onlyFavorite = ref(false)

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const fetchRealtime = async () => {
  const res = await axios.get('/api/v1/monitor/metrics', { headers: authHeaders() })
  realtime.value = res.data.data || realtime.value
}

const fetchSettings = async () => {
  const res = await axios.get('/api/v1/monitor/settings', { headers: authHeaders() })
  if (res.data.code === 0) {
    settings.value = res.data.data || []
    const active = settings.value.find(s => s.active)
    currentProm.value = active ? (active.name || active.prometheus_url) : ''
  }
}

const resetForm = () => {
  form.value = {
    id: '',
    name: '',
    prometheus_url: '',
    auth_type: 'none',
    username: '',
    password: '',
    token: ''
  }
}

const editSetting = (row) => {
  form.value = {
    id: row.id,
    name: row.name || '',
    prometheus_url: row.prometheus_url || '',
    auth_type: row.auth_type || 'none',
    username: row.username || '',
    password: row.password || '',
    token: row.token || ''
  }
}

const saveSetting = async () => {
  if (!form.value.prometheus_url) {
    ElMessage.warning('请填写地址')
    return
  }
  if (form.value.id) {
    await axios.put(`/api/v1/monitor/settings/${form.value.id}`, form.value, { headers: authHeaders() })
    ElMessage.success('更新成功')
  } else {
    await axios.post('/api/v1/monitor/settings', form.value, { headers: authHeaders() })
    ElMessage.success('新增成功')
  }
  await fetchSettings()
  resetForm()
}

const deleteSetting = async (row) => {
  await axios.delete(`/api/v1/monitor/settings/${row.id}`, { headers: authHeaders() })
  ElMessage.success('删除成功')
  fetchSettings()
}

const activateSetting = async (row) => {
  await axios.post(`/api/v1/monitor/settings/${row.id}/activate`, {}, { headers: authHeaders() })
  ElMessage.success('已设为当前')
  fetchSettings()
  fetchPromInfo()
}

const testSetting = async (row) => {
  try {
    const res = await axios.post(`/api/v1/monitor/settings/${row.id}/test`, {}, { headers: authHeaders() })
    if (res.data?.status === 'success') {
      ElMessage.success('连接成功')
    } else {
      ElMessage.warning('Prometheus 返回异常')
    }
  } catch (err) {
    ElMessage.error('连接失败')
  }
}

const settingsRowClass = ({ row }) => {
  return row.active ? 'row-active' : ''
}

const fetchPromInfo = async () => {
  try {
    const [buildRes, runtimeRes] = await Promise.all([
      axios.get('/api/v1/monitor/prometheus/buildinfo', { headers: authHeaders() }),
      axios.get('/api/v1/monitor/prometheus/runtimeinfo', { headers: authHeaders() })
    ])
    const version = buildRes.data?.data?.version || ''
    const uptimeSec = runtimeRes.data?.data?.uptime || 0
    let uptime = ''
    if (uptimeSec > 0) {
      const sec = Number(uptimeSec)
      const h = Math.floor(sec / 3600)
      const m = Math.floor((sec % 3600) / 60)
      uptime = `${h}h ${m}m`
    }
    promInfo.value = { version, uptime }
  } catch (e) {}
}

const runQuery = async () => {
  try {
    if (mode.value === 'instant') {
      const res = await axios.get('/api/v1/monitor/prometheus/query', {
        headers: authHeaders(),
        params: { query: query.value }
      })
      result.value = JSON.stringify(res.data, null, 2)
      renderInstant(res.data)
      await saveHistory('instant')
    } else {
      if (!rangeStart.value || !rangeEnd.value || !rangeStep.value) {
        ElMessage.warning('请填写 start/end/step')
        return
      }
      const res = await axios.get('/api/v1/monitor/prometheus/query_range', {
        headers: authHeaders(),
        params: {
          query: query.value,
          start: rangeStart.value,
          end: rangeEnd.value,
          step: rangeStep.value
        }
      })
      result.value = JSON.stringify(res.data, null, 2)
      renderRange(res.data)
      await saveHistory('range')
    }
  } catch (err) {
    result.value = err?.response?.data ? JSON.stringify(err.response.data, null, 2) : String(err)
  }
}

const applyTemplate = (tpl) => {
  mode.value = tpl.mode
  query.value = tpl.query
  if (tpl.mode === 'range') {
    initRange()
  }
  runQuery()
}

const fetchHistory = async () => {
  const res = await axios.get('/api/v1/monitor/prometheus/history', {
    headers: authHeaders(),
    params: onlyFavorite.value ? { favorite: true } : {}
  })
  const items = res.data.data || []
  history.value = items.map((it) => ({
    id: it.id,
    time: new Date(it.created_at).toLocaleString(),
    mode: it.mode === 'range' ? '范围' : '即时',
    query: it.query,
    range: it.mode === 'range' ? `${it.start} ~ ${it.end} / ${it.step}s` : '-'
    ,
    name: it.name || '',
    favorite: !!it.favorite
  }))
}

const saveHistory = async (m) => {
  await axios.post('/api/v1/monitor/prometheus/history', {
    mode: m,
    query: query.value,
    start: m === 'range' ? rangeStart.value : '',
    end: m === 'range' ? rangeEnd.value : '',
    step: m === 'range' ? rangeStep.value : ''
  }, { headers: authHeaders() })
  await fetchHistory()
}

const toggleFavorite = async (row) => {
  await axios.put(`/api/v1/monitor/prometheus/history/${row.id}`, {
    favorite: row.favorite
  }, { headers: authHeaders() })
  if (onlyFavorite.value) fetchHistory()
}

const saveName = async (row) => {
  await axios.put(`/api/v1/monitor/prometheus/history/${row.id}`, {
    name: row.name
  }, { headers: authHeaders() })
}

const reRun = (row) => {
  mode.value = row.mode === '范围' ? 'range' : 'instant'
  query.value = row.query
  if (mode.value === 'range') {
    const parts = (row.range || '').split('/')
    if (parts.length === 2) {
      const times = parts[0].split('~')
      if (times.length === 2) {
        rangeStart.value = times[0].trim()
        rangeEnd.value = times[1].trim()
      }
      rangeStep.value = parts[1].replace('s', '').trim()
    }
  }
  runQuery()
}

const ensureChart = () => {
  if (!chartRef.value) return
  if (!chartInstance) {
    chartInstance = echarts.init(chartRef.value)
  }
}

const renderInstant = (data) => {
  ensureChart()
  if (!chartInstance) return
  const series = []
  const result = data?.data?.result || []
  result.forEach((item) => {
    const name = item.metric?.__name__ || item.metric?.job || 'series'
    const value = Array.isArray(item.value) ? Number(item.value[1]) : 0
    series.push({ name, type: 'bar', data: [value] })
  })
  chartInstance.setOption({
    tooltip: { trigger: 'axis' },
    legend: { top: 0 },
    xAxis: { type: 'category', data: ['value'] },
    yAxis: { type: 'value' },
    series
  })
}

const renderRange = (data) => {
  ensureChart()
  if (!chartInstance) return
  const result = data?.data?.result || []
  const series = result.map((item, idx) => {
    const name = item.metric?.__name__ || item.metric?.job || `series_${idx + 1}`
    const values = (item.values || []).map(v => Number(v[1]))
    return { name, type: 'line', showSymbol: false, data: values }
  })
  const labels = (result[0]?.values || []).map(v => new Date(Number(v[0]) * 1000).toLocaleTimeString())
  chartInstance.setOption({
    tooltip: { trigger: 'axis' },
    legend: { top: 0 },
    xAxis: { type: 'category', data: labels },
    yAxis: { type: 'value' },
    series
  })
}

const initRange = () => {
  const end = Math.floor(Date.now() / 1000)
  const start = end - 3600
  rangeStart.value = String(start)
  rangeEnd.value = String(end)
}

onMounted(() => {
  initRange()
  fetchSettings()
  fetchPromInfo()
  fetchRealtime()
  ensureChart()
  fetchHistory()
})

onBeforeUnmount(() => {
  if (chartInstance) {
    chartInstance.dispose()
    chartInstance = null
  }
})
</script>

<style scoped>
.page-card { max-width: 1180px; margin: 0 auto; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 16px; }
.page-desc { color: #606266; margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; }
.metric-cards { margin-bottom: 8px; }
.card-title { color: #909399; font-size: 12px; }
.card-value { font-size: 22px; font-weight: 600; margin-top: 6px; }
.section-title { margin: 12px 0; }
.meta-row { display: flex; gap: 12px; align-items: center; margin: 6px 0 12px; }
.meta-text { color: #606266; font-size: 12px; }
.row-active td { background: #f0f9eb !important; }
.query-bar { display: flex; gap: 8px; flex-wrap: wrap; margin-bottom: 12px; }
.template-bar { display: flex; flex-wrap: wrap; gap: 8px; margin-bottom: 12px; }
.template-tag { cursor: pointer; }
.history-bar { display: flex; align-items: center; margin-bottom: 8px; }
.w-52 { width: 220px; }
.w-40 { width: 160px; }
.chart-box { width: 100%; height: 360px; }
</style>
