<template>
  <el-card class="page-card">
    <template #header>
      <div class="header">
        <div>
          <div class="title">命令风控规则</div>
          <div class="desc">支持风险规则与白名单规则，多规则合并判定，支持批量启停。</div>
        </div>
        <div class="actions">
          <el-button type="primary" icon="Plus" @click="openCreate">新增规则</el-button>
          <el-button icon="Refresh" @click="loadAll">刷新</el-button>
        </div>
      </div>
    </template>

    <el-row :gutter="10" class="stats-row">
      <el-col :md="4" :sm="8" :xs="12"><el-card shadow="hover" class="stat-card"><div class="stat-k">规则总数</div><div class="stat-v">{{ stats.rules_total }}</div></el-card></el-col>
      <el-col :md="4" :sm="8" :xs="12"><el-card shadow="hover" class="stat-card"><div class="stat-k">启用规则</div><div class="stat-v">{{ stats.rules_enabled }}</div></el-card></el-col>
      <el-col :md="4" :sm="8" :xs="12"><el-card shadow="hover" class="stat-card"><div class="stat-k">白名单</div><div class="stat-v">{{ stats.rules_allow }}</div></el-card></el-col>
      <el-col :md="4" :sm="8" :xs="12"><el-card shadow="hover" class="stat-card"><div class="stat-k">近{{ stats.window_days }}天命令</div><div class="stat-v">{{ stats.commands_window }}</div></el-card></el-col>
      <el-col :md="4" :sm="8" :xs="12"><el-card shadow="hover" class="stat-card"><div class="stat-k">风险命令</div><div class="stat-v danger">{{ stats.commands_risky }}</div></el-card></el-col>
      <el-col :md="4" :sm="8" :xs="12"><el-card shadow="hover" class="stat-card"><div class="stat-k">阻断命令</div><div class="stat-v danger">{{ stats.commands_blocked }}</div></el-card></el-col>
    </el-row>

    <el-row :gutter="12" class="chart-row">
      <el-col :md="12" :sm="24">
        <el-card shadow="never">
          <div class="sub-title">按天趋势</div>
          <div ref="trendRef" class="chart-box"></div>
        </el-card>
      </el-col>
      <el-col :md="12" :sm="24">
        <el-card shadow="never">
          <div class="sub-title">高风险资产 Top10</div>
          <div ref="assetRef" class="chart-box"></div>
        </el-card>
      </el-col>
    </el-row>

    <el-card shadow="never" class="chart-single-card">
      <div class="sub-title">高风险用户 Top10</div>
      <div ref="userRef" class="chart-box"></div>
    </el-card>

    <div class="toolbar">
      <el-input v-model="filters.keyword" clearable placeholder="规则名/表达式" class="filter-item" @change="loadRules" />
      <el-select v-model="filters.rule_kind" clearable placeholder="规则类型" class="filter-item" @change="loadRules">
        <el-option label="风险规则" value="risk" />
        <el-option label="白名单" value="allow" />
      </el-select>
      <el-select v-model="filters.protocol" clearable placeholder="协议" class="filter-item" @change="loadRules">
        <el-option label="ssh" value="ssh" />
        <el-option label="docker" value="docker" />
        <el-option label="k8s" value="k8s" />
      </el-select>
      <el-select v-model="filters.enabled" clearable placeholder="状态" class="filter-item" @change="loadRules">
        <el-option label="启用" :value="true" />
        <el-option label="禁用" :value="false" />
      </el-select>
    </div>

    <div class="batch-bar">
      <el-button size="small" :disabled="selectedIDs.length === 0" @click="batchOperate('enable')">批量启用</el-button>
      <el-button size="small" :disabled="selectedIDs.length === 0" @click="batchOperate('disable')">批量禁用</el-button>
      <el-button size="small" type="danger" :disabled="selectedIDs.length === 0" @click="batchOperate('delete')">批量删除</el-button>
      <span class="batch-tip">已选 {{ selectedIDs.length }} 条</span>
    </div>

    <el-table :fit="true" :data="rules" v-loading="loading" stripe @selection-change="onSelectionChange">
      <el-table-column type="selection" width="44" fixed="left" />
      <el-table-column prop="name" label="规则" min-width="160" />
      <el-table-column label="类型" width="90">
        <template #default="{ row }">
          <el-tag :type="row.rule_kind === 'allow' ? 'success' : 'warning'">{{ row.rule_kind || 'risk' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="pattern" label="匹配表达式" min-width="260" show-overflow-tooltip />
      <el-table-column prop="match_type" label="匹配方式" width="110" />
      <el-table-column prop="protocol" label="协议" width="100">
        <template #default="{ row }">{{ row.protocol || '全部' }}</template>
      </el-table-column>
      <el-table-column prop="severity" label="风险级别" width="110">
        <template #default="{ row }">
          <el-tag v-if="row.rule_kind !== 'allow'" :type="severityTag(row.severity)">{{ row.severity }}</el-tag>
          <el-tag v-else type="success">allow</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="action" label="动作" width="100">
        <template #default="{ row }">
          <el-tag :type="row.action === 'block' ? 'danger' : (row.rule_kind === 'allow' ? 'success' : 'warning')">
            {{ row.rule_kind === 'allow' ? 'allow' : row.action }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="priority" label="优先级" width="90" />
      <el-table-column label="状态" width="110">
        <template #default="{ row }">
          <el-switch v-model="row.enabled" @change="toggleEnabled(row)" />
        </template>
      </el-table-column>
      <el-table-column label="操作" width="170" fixed="right">
        <template #default="{ row }">
          <el-button size="small" type="primary" plain @click="openEdit(row)">编辑</el-button>
          <el-button size="small" type="danger" plain @click="removeRule(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-card shadow="never" class="top-card">
      <template #header>
        <div class="sub-title">规则命中 Top10</div>
      </template>
      <el-table :fit="true" :data="stats.top_rules || []" stripe>
        <el-table-column prop="rule" label="规则" min-width="260" />
        <el-table-column prop="count" label="命中次数" width="120" />
      </el-table>
    </el-card>
  </el-card>

  <el-dialog append-to-body v-model="dialogVisible" :title="editing ? '编辑风控规则' : '新增风控规则'" width="680px">
    <el-form :model="form" label-width="92px">
      <el-form-item label="规则名称" required>
        <el-input v-model="form.name" />
      </el-form-item>
      <el-form-item label="规则类型">
        <el-select v-model="form.rule_kind" style="width: 100%">
          <el-option label="风险规则" value="risk" />
          <el-option label="白名单" value="allow" />
        </el-select>
      </el-form-item>
      <el-form-item label="匹配表达式" required>
        <el-input v-model="form.pattern" type="textarea" :rows="2" placeholder="如 rm -rf / 或 ^\s*DROP\s+DATABASE" />
      </el-form-item>
      <el-form-item label="匹配方式">
        <el-select v-model="form.match_type" style="width: 100%">
          <el-option label="contains" value="contains" />
          <el-option label="prefix" value="prefix" />
          <el-option label="exact" value="exact" />
          <el-option label="regex" value="regex" />
        </el-select>
      </el-form-item>
      <el-form-item label="协议范围">
        <el-select v-model="form.protocol" clearable style="width: 100%" placeholder="为空表示全部协议">
          <el-option label="ssh" value="ssh" />
          <el-option label="docker" value="docker" />
          <el-option label="k8s" value="k8s" />
        </el-select>
      </el-form-item>
      <el-form-item label="风险与动作">
        <div class="inline-fields">
          <el-select v-model="form.severity" :disabled="form.rule_kind === 'allow'">
            <el-option label="critical" value="critical" />
            <el-option label="warning" value="warning" />
            <el-option label="info" value="info" />
          </el-select>
          <el-select v-model="form.action" :disabled="form.rule_kind === 'allow'">
            <el-option label="alert" value="alert" />
            <el-option label="block" value="block" />
          </el-select>
          <el-input-number v-model="form.priority" :min="1" :max="9999" />
        </div>
      </el-form-item>
      <el-form-item label="说明">
        <el-input v-model="form.description" type="textarea" :rows="2" />
      </el-form-item>
      <el-form-item label="启用">
        <el-switch v-model="form.enabled" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="dialogVisible = false">取消</el-button>
      <el-button type="primary" :loading="saving" @click="saveRule">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import axios from 'axios'
import * as echarts from 'echarts'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getErrorMessage, isCancelError } from '@/utils/error'

const loading = ref(false)
const saving = ref(false)
const dialogVisible = ref(false)
const editing = ref(false)
const currentID = ref('')
const rules = ref([])
const selectedIDs = ref([])
const trendRef = ref(null)
const assetRef = ref(null)
const userRef = ref(null)
let trendChart = null
let assetChart = null
let userChart = null

const stats = reactive({
  rules_total: 0,
  rules_enabled: 0,
  rules_allow: 0,
  commands_total: 0,
  commands_window: 0,
  window_days: 7,
  commands_risky: 0,
  commands_blocked: 0,
  top_rules: [],
  top_assets: [],
  top_users: [],
  trend_by_day: []
})

const filters = reactive({
  keyword: '',
  rule_kind: '',
  protocol: '',
  enabled: null
})

const form = reactive({
  name: '',
  rule_kind: 'risk',
  pattern: '',
  match_type: 'contains',
  protocol: '',
  severity: 'warning',
  action: 'alert',
  priority: 100,
  description: '',
  enabled: true
})

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const severityTag = (severity) => {
  if (severity === 'critical') return 'danger'
  if (severity === 'info') return 'info'
  return 'warning'
}

watch(() => form.rule_kind, (kind) => {
  if (kind === 'allow') {
    form.severity = 'info'
    form.action = 'alert'
  }
})

const onSelectionChange = (rows) => {
  selectedIDs.value = (rows || []).map(item => item.id).filter(Boolean)
}

const loadRules = async () => {
  loading.value = true
  try {
    const params = {
      keyword: filters.keyword || undefined,
      rule_kind: filters.rule_kind || undefined,
      protocol: filters.protocol || undefined,
      enabled: filters.enabled === null ? undefined : filters.enabled
    }
    const res = await axios.get('/api/v1/jump/command-rules', { headers: authHeaders(), params })
    if (res.data.code === 0) rules.value = res.data.data || []
  } catch (error) {
    ElMessage.error(getErrorMessage(error, '加载规则失败'))
  } finally {
    loading.value = false
  }
}

const renderCharts = async () => {
  await nextTick()
  if (!trendRef.value || !assetRef.value || !userRef.value) return
  if (!trendChart) trendChart = echarts.init(trendRef.value)
  if (!assetChart) assetChart = echarts.init(assetRef.value)
  if (!userChart) userChart = echarts.init(userRef.value)

  const days = (stats.trend_by_day || []).map(i => i.day)
  const total = (stats.trend_by_day || []).map(i => i.total || 0)
  const risky = (stats.trend_by_day || []).map(i => i.risky || 0)
  const blocked = (stats.trend_by_day || []).map(i => i.blocked || 0)
  const allowPass = (stats.trend_by_day || []).map(i => i.allow_pass || 0)

  trendChart.setOption({
    tooltip: { trigger: 'axis' },
    legend: { top: 0, data: ['总命令', '风险', '阻断', '白名单放行'] },
    grid: { left: 38, right: 14, top: 34, bottom: 24 },
    xAxis: { type: 'category', data: days },
    yAxis: { type: 'value' },
    series: [
      { name: '总命令', type: 'line', data: total, smooth: true },
      { name: '风险', type: 'line', data: risky, smooth: true },
      { name: '阻断', type: 'line', data: blocked, smooth: true },
      { name: '白名单放行', type: 'line', data: allowPass, smooth: true }
    ]
  })

  const assetNames = (stats.top_assets || []).map(i => i.name)
  const assetValues = (stats.top_assets || []).map(i => i.count)
  assetChart.setOption({
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
    grid: { left: 120, right: 18, top: 10, bottom: 20 },
    xAxis: { type: 'value' },
    yAxis: { type: 'category', data: assetNames, inverse: true },
    series: [{ type: 'bar', data: assetValues }]
  })

  const userNames = (stats.top_users || []).map(i => i.name)
  const userValues = (stats.top_users || []).map(i => i.count)
  userChart.setOption({
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
    grid: { left: 40, right: 18, top: 16, bottom: 60 },
    xAxis: { type: 'category', data: userNames, axisLabel: { rotate: 25 } },
    yAxis: { type: 'value' },
    series: [{ type: 'bar', data: userValues }]
  })
}

const loadStats = async () => {
  try {
    const res = await axios.get('/api/v1/jump/command-rules/stats', { headers: authHeaders(), params: { days: 7 } })
    if (res.data.code === 0) {
      Object.assign(stats, {
        rules_total: res.data.data?.rules_total || 0,
        rules_enabled: res.data.data?.rules_enabled || 0,
        rules_allow: res.data.data?.rules_allow || 0,
        commands_total: res.data.data?.commands_total || 0,
        commands_window: res.data.data?.commands_window || 0,
        window_days: res.data.data?.window_days || 7,
        commands_risky: res.data.data?.commands_risky || 0,
        commands_blocked: res.data.data?.commands_blocked || 0,
        top_rules: res.data.data?.top_rules || [],
        top_assets: res.data.data?.top_assets || [],
        top_users: res.data.data?.top_users || [],
        trend_by_day: res.data.data?.trend_by_day || []
      })
      renderCharts()
    }
  } catch {
    // ignore
  }
}

const loadAll = async () => {
  await Promise.all([loadRules(), loadStats()])
}

const resetForm = () => {
  Object.assign(form, {
    name: '',
    rule_kind: 'risk',
    pattern: '',
    match_type: 'contains',
    protocol: '',
    severity: 'warning',
    action: 'alert',
    priority: 100,
    description: '',
    enabled: true
  })
}

const openCreate = () => {
  editing.value = false
  currentID.value = ''
  resetForm()
  dialogVisible.value = true
}

const openEdit = (row) => {
  editing.value = true
  currentID.value = row.id
  Object.assign(form, {
    name: row.name || '',
    rule_kind: row.rule_kind || 'risk',
    pattern: row.pattern || '',
    match_type: row.match_type || 'contains',
    protocol: row.protocol || '',
    severity: row.severity || 'warning',
    action: row.action || 'alert',
    priority: row.priority || 100,
    description: row.description || '',
    enabled: !!row.enabled
  })
  dialogVisible.value = true
}

const saveRule = async () => {
  if (!form.name.trim() || !form.pattern.trim()) {
    ElMessage.warning('规则名称和匹配表达式必填')
    return
  }
  saving.value = true
  try {
    const payload = {
      name: form.name.trim(),
      rule_kind: form.rule_kind,
      pattern: form.pattern.trim(),
      match_type: form.match_type,
      protocol: form.protocol,
      severity: form.rule_kind === 'allow' ? 'info' : form.severity,
      action: form.rule_kind === 'allow' ? 'alert' : form.action,
      priority: form.priority,
      description: form.description,
      enabled: form.enabled
    }
    if (editing.value) {
      await axios.put(`/api/v1/jump/command-rules/${currentID.value}`, payload, { headers: authHeaders() })
      ElMessage.success('规则已更新')
    } else {
      await axios.post('/api/v1/jump/command-rules', payload, { headers: authHeaders() })
      ElMessage.success('规则已创建')
    }
    dialogVisible.value = false
    await loadAll()
  } catch (error) {
    ElMessage.error(getErrorMessage(error, '保存失败'))
  } finally {
    saving.value = false
  }
}

const toggleEnabled = async (row) => {
  const next = !!row.enabled
  try {
    await axios.put(`/api/v1/jump/command-rules/${row.id}`, { enabled: next }, { headers: authHeaders() })
  } catch (error) {
    row.enabled = !next
    ElMessage.error(getErrorMessage(error, '更新状态失败'))
  }
}

const removeRule = async (row) => {
  try {
    await ElMessageBox.confirm(`确认删除规则 ${row.name} 吗？`, '提示', { type: 'warning' })
    await axios.delete(`/api/v1/jump/command-rules/${row.id}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    await loadAll()
  } catch (error) {
    if (!isCancelError(error)) ElMessage.error(getErrorMessage(error, '删除失败'))
  }
}

const batchOperate = async (action) => {
  const actionText = action === 'enable' ? '批量启用' : (action === 'disable' ? '批量禁用' : '批量删除')
  const run = async () => {
    await axios.post('/api/v1/jump/command-rules/batch', { action, ids: selectedIDs.value }, { headers: authHeaders() })
    ElMessage.success(actionText + '成功')
    selectedIDs.value = []
    await loadAll()
  }
  try {
    if (action === 'delete') {
      await ElMessageBox.confirm(`确认${actionText}已选 ${selectedIDs.value.length} 条规则吗？`, '提示', { type: 'warning' })
    }
    await run()
  } catch (error) {
    if (!isCancelError(error)) {
      ElMessage.error(getErrorMessage(error, actionText + '失败'))
    }
  }
}

onMounted(loadAll)
onBeforeUnmount(() => {
  if (trendChart) trendChart.dispose()
  if (assetChart) assetChart.dispose()
  if (userChart) userChart.dispose()
})
</script>

<style scoped>
.page-card { max-width: 100%; }
.header { display: flex; justify-content: space-between; align-items: center; gap: 12px; }
.title { font-size: 18px; font-weight: 600; }
.desc { color: #909399; margin-top: 4px; }
.actions { display: flex; gap: 8px; }
.stats-row { margin-bottom: 12px; }
.stat-card { padding: 4px; }
.stat-k { color: #909399; font-size: 12px; }
.stat-v { margin-top: 6px; font-size: 22px; font-weight: 600; color: #303133; }
.stat-v.danger { color: #f56c6c; }
.chart-row { margin-bottom: 12px; }
.chart-single-card { margin-bottom: 12px; }
.sub-title { margin-bottom: 8px; font-weight: 600; }
.chart-box { height: 260px; }
.toolbar { margin-bottom: 10px; display: flex; gap: 8px; flex-wrap: wrap; }
.filter-item { width: 220px; }
.batch-bar { margin-bottom: 10px; display: flex; gap: 8px; align-items: center; }
.batch-tip { color: #909399; font-size: 12px; }
.inline-fields { width: 100%; display: grid; gap: 8px; grid-template-columns: repeat(3, minmax(0, 1fr)); }
.top-card { margin-top: 12px; }
@media (max-width: 768px) {
  .header { flex-direction: column; align-items: flex-start; }
  .filter-item { width: 100%; }
  .inline-fields { grid-template-columns: 1fr; }
}
</style>
