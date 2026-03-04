<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>告警复盘</h2>
        <p class="page-desc">查看告警历史与复盘信息。</p>
      </div>
      <div class="page-actions">
        <el-button @click="fieldDialogVisible = true">字段选择</el-button>
        <el-button @click="exportCSV">导出CSV</el-button>
        <el-button @click="exportJSON">导出JSON</el-button>
        <el-button icon="Refresh" @click="fetchHistory">刷新</el-button>
      </div>
    </div>

    <el-row :gutter="16" class="summary-row">
      <el-col :span="12">
        <el-card>
          <div class="section-title">按级别统计</div>
          <div ref="severityRef" class="chart-box"></div>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card>
          <div class="section-title">按时间统计</div>
          <div class="chart-toolbar">
            <el-date-picker
              v-model="statsRange"
              type="daterange"
              range-separator="-"
              start-placeholder="开始日期"
              end-placeholder="结束日期"
              @change="renderCharts(items)"
            />
          </div>
          <div ref="timelineRef" class="chart-box"></div>
        </el-card>
      </el-col>
    </el-row>

    <el-divider />

    <div class="filter-bar">
      <el-select v-model="filterSeverity" placeholder="级别" class="w-40" clearable @change="fetchHistory">
        <el-option label="critical" value="critical" />
        <el-option label="warning" value="warning" />
        <el-option label="info" value="info" />
      </el-select>
      <el-input v-model="filterTarget" placeholder="目标包含" class="w-52" clearable @change="fetchHistory" />
      <el-input v-model="filterRule" placeholder="规则ID包含" class="w-52" clearable @change="fetchHistory" />
      <el-date-picker
        v-model="filterRange"
        type="daterange"
        range-separator="-"
        start-placeholder="开始日期"
        end-placeholder="结束日期"
        @change="fetchHistory"
      />
      <el-button type="primary" @click="fetchHistory">筛选</el-button>
    </div>

    <el-table :fit="true" :data="items" stripe style="width: 100%">
      <el-table-column v-if="fields.rule_id" prop="rule_id" label="规则ID" min-width="160" />
      <el-table-column v-if="fields.target" prop="target" label="目标" min-width="200" />
      <el-table-column v-if="fields.severity" prop="severity" label="级别" width="120" />
      <el-table-column v-if="fields.fired_at" prop="fired_at" label="触发时间" min-width="180" />
      <el-table-column v-if="fields.resolved_at" prop="resolved_at" label="恢复时间" min-width="180" />
      <el-table-column v-if="fields.duration" prop="duration" label="持续(s)" width="100" />
      <el-table-column v-if="fields.resolution" prop="resolution" label="处理方式" min-width="160" />
      <el-table-column v-if="fields.root_cause" prop="root_cause" label="根因" min-width="200" />
      <el-table-column label="操作" width="120">
        <template #default="scope">
          <el-button size="small" @click="openDetail(scope.row)">详情</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog append-to-body v-model="fieldDialogVisible" title="导出字段选择" width="520px">
      <el-checkbox-group v-model="selectedFieldKeys">
        <el-checkbox label="rule_id">规则ID</el-checkbox>
        <el-checkbox label="target">目标</el-checkbox>
        <el-checkbox label="severity">级别</el-checkbox>
        <el-checkbox label="fired_at">触发时间</el-checkbox>
        <el-checkbox label="resolved_at">恢复时间</el-checkbox>
        <el-checkbox label="duration">持续(s)</el-checkbox>
        <el-checkbox label="resolution">处理方式</el-checkbox>
        <el-checkbox label="root_cause">根因</el-checkbox>
      </el-checkbox-group>
      <template #footer>
        <el-button @click="fieldDialogVisible = false">关闭</el-button>
        <el-button type="primary" @click="applyFields">应用</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'
import * as echarts from 'echarts'

const items = ref([])
const router = useRouter()
const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })
const severityRef = ref(null)
const timelineRef = ref(null)
let severityChart = null
let timelineChart = null
const filterSeverity = ref('')
const filterRange = ref([])
const filterTarget = ref('')
const filterRule = ref('')
const statsRange = ref([])
const fieldDialogVisible = ref(false)
const storageKey = 'alert_history_fields'
const selectedFieldKeys = ref(['rule_id','target','severity','fired_at','resolved_at','duration','resolution','root_cause'])
const fields = ref({
  rule_id: true,
  target: true,
  severity: true,
  fired_at: true,
  resolved_at: true,
  duration: true,
  resolution: true,
  root_cause: true
})

const buildParams = () => {
  const params = {}
  if (filterSeverity.value) params.severity = filterSeverity.value
  if (filterTarget.value) params.target = filterTarget.value
  if (filterRule.value) params.rule_id = filterRule.value
  if (filterRange.value && filterRange.value.length === 2) {
    params.start = filterRange.value[0]
    params.end = filterRange.value[1]
  }
  return params
}

const fetchHistory = async () => {
  const params = buildParams()
  const res = await axios.get('/api/v1/alert/history', { headers: authHeaders(), params })
  items.value = res.data.data || []
  renderCharts(items.value)
}

onMounted(fetchHistory)
onMounted(() => {
  const saved = localStorage.getItem(storageKey)
  if (saved) {
    try {
      selectedFieldKeys.value = JSON.parse(saved)
      applyFields()
    } catch {}
  }
})

const renderCharts = (list) => {
  if (!severityRef.value || !timelineRef.value) return
  if (!severityChart) severityChart = echarts.init(severityRef.value)
  if (!timelineChart) timelineChart = echarts.init(timelineRef.value)

  const sevCount = { critical: 0, warning: 0, info: 0 }
  list.forEach(i => {
    if (sevCount[i.severity] !== undefined) sevCount[i.severity] += 1
  })
  severityChart.setOption({
    tooltip: { trigger: 'item' },
    legend: { top: 0 },
    series: [
      {
        name: 'Severity',
        type: 'pie',
        radius: '60%',
        data: [
          { value: sevCount.critical, name: 'critical' },
          { value: sevCount.warning, name: 'warning' },
          { value: sevCount.info, name: 'info' }
        ]
      }
    ]
  })

  const days = []
  const counts = []
  const map = new Map()
  let startDate = new Date()
  let endDate = new Date()
  if (statsRange.value && statsRange.value.length === 2) {
    startDate = new Date(statsRange.value[0])
    endDate = new Date(statsRange.value[1])
  } else {
    endDate = new Date()
    startDate = new Date(endDate.getTime() - 6 * 24 * 3600 * 1000)
  }
  startDate.setHours(0, 0, 0, 0)
  endDate.setHours(0, 0, 0, 0)
  for (let d = new Date(startDate); d <= endDate; d.setDate(d.getDate() + 1)) {
    const key = d.toISOString().slice(0, 10)
    days.push(key)
    map.set(key, 0)
  }
  list.forEach(i => {
    const key = String(i.fired_at).slice(0, 10)
    if (map.has(key)) map.set(key, map.get(key) + 1)
  })
  days.forEach(d => counts.push(map.get(d) || 0))
  timelineChart.setOption({
    tooltip: { trigger: 'axis' },
    xAxis: { type: 'category', data: days },
    yAxis: { type: 'value' },
    series: [{ name: '告警数', type: 'bar', data: counts }]
  })
}

onBeforeUnmount(() => {
  if (severityChart) severityChart.dispose()
  if (timelineChart) timelineChart.dispose()
})

const exportCSV = () => {
  const params = buildParams()
  let filename = 'alert-history.csv'
  if (params.severity) filename = `alert-history-${params.severity}.csv`
  const header = selectedFieldKeys.value
  const rows = items.value.map(i => header.map(k => i[k] ?? ''))
  const meta = [
    `# export_time=${new Date().toISOString()}`,
    `# severity=${params.severity || ''}`,
    `# start=${params.start || ''}`,
    `# end=${params.end || ''}`
  ]
  const csv = [...meta, header, ...rows]
    .map(r => r.map(v => `"${String(v).replace(/"/g, '""')}"`).join(','))
    .join('\n')
  const blob = new Blob([csv], { type: 'text/csv' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(url)
}

const exportJSON = () => {
  const params = buildParams()
  let filename = 'alert-history.json'
  if (params.severity) filename = `alert-history-${params.severity}.json`
  const payload = {
    exported_at: new Date().toISOString(),
    filter: params,
    fields: selectedFieldKeys.value,
    items: items.value.map(i => {
      const obj = {}
      selectedFieldKeys.value.forEach(k => { obj[k] = i[k] })
      return obj
    })
  }
  const blob = new Blob([JSON.stringify(payload, null, 2)], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(url)
}

const openDetail = (row) => {
  router.push({ path: '/alert/history/detail', query: { id: row.id } })
}

const applyFields = () => {
  const next = {}
  selectedFieldKeys.value.forEach(k => { next[k] = true })
  fields.value = next
  fieldDialogVisible.value = false
  localStorage.setItem(storageKey, JSON.stringify(selectedFieldKeys.value))
}
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 16px; }
.page-desc { color: #606266; margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; }
.summary-row { margin-bottom: 16px; }
.chart-box { width: 100%; height: 260px; }
.section-title { margin-bottom: 8px; font-weight: 600; }
.chart-toolbar { margin-bottom: 8px; display: flex; justify-content: flex-end; }
.filter-bar { display: flex; gap: 8px; flex-wrap: wrap; margin-bottom: 12px; }
.w-40 { width: 160px; }
.w-52 { width: 220px; }
</style>
