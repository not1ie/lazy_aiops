<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>编排中心</h2>
        <p class="page-desc">统一事件接入、规则路由与 Runbook 执行，贴近企业值班场景。</p>
      </div>
      <div class="page-actions">
        <el-button type="primary" icon="Plus" @click="openRuleDialog()">新建规则</el-button>
        <el-button icon="Promotion" @click="openIngestDialog">注入事件</el-button>
        <el-button icon="Refresh" @click="refreshActiveTab">刷新</el-button>
      </div>
    </div>

    <el-row :gutter="12" class="mb-12">
      <el-col :md="4" :sm="12" :xs="24"><el-card><div class="k">规则总数</div><div class="v">{{ overview.rules_total || 0 }}</div></el-card></el-col>
      <el-col :md="4" :sm="12" :xs="24"><el-card><div class="k">启用规则</div><div class="v info">{{ overview.rules_enabled || 0 }}</div></el-card></el-col>
      <el-col :md="5" :sm="12" :xs="24"><el-card><div class="k">24h 事件</div><div class="v">{{ overview.events_24h || 0 }}</div></el-card></el-col>
      <el-col :md="5" :sm="12" :xs="24"><el-card><div class="k">24h 分发成功</div><div class="v success">{{ overview.dispatched_24h || 0 }}</div></el-card></el-col>
      <el-col :md="6" :sm="12" :xs="24"><el-card><div class="k">24h 失败/部分失败</div><div class="v danger">{{ overview.failed_24h || 0 }}</div></el-card></el-col>
    </el-row>

    <el-tabs v-model="activeTab">
      <el-tab-pane label="路由规则" name="rules">
        <div class="toolbar">
          <el-input v-model.trim="ruleFilter.source" placeholder="来源（source）" clearable style="width: 180px" />
          <el-input v-model.trim="ruleFilter.event_type" placeholder="事件类型（event_type）" clearable style="width: 220px" />
          <el-select v-model="ruleFilter.enabled" clearable placeholder="启用状态" style="width: 130px">
            <el-option label="启用" value="true" />
            <el-option label="停用" value="false" />
          </el-select>
          <el-button type="primary" icon="Search" @click="fetchRules">查询</el-button>
          <el-button icon="Refresh" @click="resetRuleFilter">重置</el-button>
        </div>

        <el-table :fit="true" :data="rules" stripe v-loading="rulesLoading">
          <el-table-column prop="name" label="规则名" min-width="160" />
          <el-table-column prop="source" label="来源" width="120" />
          <el-table-column prop="event_type" label="事件类型" min-width="160" />
          <el-table-column prop="workflow_name" label="目标 Runbook" min-width="180" show-overflow-tooltip />
          <el-table-column label="状态" width="90">
            <template #default="{ row }">
              <StatusBadge v-bind="ruleEnabledBadge(row)" />
            </template>
          </el-table-column>
          <el-table-column label="触发/失败" width="110">
            <template #default="{ row }">{{ row.trigger_count || 0 }}/{{ row.failure_count || 0 }}</template>
          </el-table-column>
          <el-table-column label="最近触发" min-width="170">
            <template #default="{ row }">{{ formatTime(row.last_triggered_at) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="260" fixed="right">
            <template #default="{ row }">
              <el-button size="small" @click="openRuleDialog(row)">编辑</el-button>
              <el-button size="small" type="primary" plain @click="prefillRunbookFromRule(row)">快速执行</el-button>
              <el-button size="small" type="danger" plain @click="removeRule(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="事件接入" name="events">
        <div class="toolbar">
          <el-input v-model.trim="eventFilter.source" placeholder="来源（source）" clearable style="width: 180px" />
          <el-input v-model.trim="eventFilter.event_type" placeholder="事件类型（event_type）" clearable style="width: 220px" />
          <el-select v-model="eventFilter.status" clearable placeholder="状态" style="width: 160px">
            <el-option label="received" value="received" />
            <el-option label="dispatched" value="dispatched" />
            <el-option label="partial" value="partial" />
            <el-option label="failed" value="failed" />
            <el-option label="ignored" value="ignored" />
          </el-select>
          <el-button type="primary" icon="Search" @click="fetchEvents">查询</el-button>
          <el-button icon="Promotion" @click="openIngestDialog">注入事件</el-button>
          <el-button icon="Refresh" @click="resetEventFilter">重置</el-button>
        </div>

        <el-table :fit="true" :data="events" stripe v-loading="eventsLoading">
          <el-table-column label="接入时间" width="180">
            <template #default="{ row }">{{ formatTime(row.received_at) }}</template>
          </el-table-column>
          <el-table-column prop="source" label="来源" width="120" />
          <el-table-column prop="event_type" label="事件类型" min-width="160" />
          <el-table-column prop="summary" label="摘要" min-width="260" show-overflow-tooltip />
          <el-table-column label="状态" width="120">
            <template #default="{ row }">
              <StatusBadge v-bind="eventStatusBadge(row)" />
            </template>
          </el-table-column>
          <el-table-column label="规则命中" width="100">
            <template #default="{ row }">{{ row.matched_rule || 0 }}</template>
          </el-table-column>
          <el-table-column label="成功/失败" width="110">
            <template #default="{ row }">{{ row.success_runs || 0 }}/{{ row.failed_runs || 0 }}</template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="分发记录" name="dispatches">
        <div class="toolbar">
          <el-input v-model.trim="dispatchFilter.event_id" placeholder="事件ID（可选）" clearable style="width: 300px" />
          <el-select v-model="dispatchFilter.status" clearable placeholder="执行状态" style="width: 160px">
            <el-option label="success" value="success" />
            <el-option label="failed" value="failed" />
            <el-option label="skipped" value="skipped" />
          </el-select>
          <el-button type="primary" icon="Search" @click="fetchDispatches">查询</el-button>
          <el-button icon="Refresh" @click="resetDispatchFilter">重置</el-button>
        </div>

        <el-table :fit="true" :data="dispatches" stripe v-loading="dispatchesLoading">
          <el-table-column label="开始时间" width="180">
            <template #default="{ row }">{{ formatTime(row.started_at) }}</template>
          </el-table-column>
          <el-table-column prop="workflow_name" label="Runbook" min-width="180" show-overflow-tooltip />
          <el-table-column prop="status" label="状态" width="100">
            <template #default="{ row }">
              <StatusBadge v-bind="dispatchStatusBadge(row)" />
            </template>
          </el-table-column>
          <el-table-column prop="trigger_by" label="触发来源" width="140" />
          <el-table-column prop="execution_id" label="执行ID" min-width="220" show-overflow-tooltip />
          <el-table-column prop="error" label="错误信息" min-width="260" show-overflow-tooltip />
          <el-table-column label="结束时间" width="180">
            <template #default="{ row }">{{ formatTime(row.finished_at) }}</template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="Runbook 执行" name="runbook">
        <el-form label-width="120px" class="runbook-form">
          <el-row :gutter="12">
            <el-col :md="12" :sm="24">
              <el-form-item label="目标 Runbook" required>
                <el-select v-model="runbookForm.workflow_id" filterable clearable style="width: 100%" placeholder="选择工作流">
                  <el-option v-for="wf in workflows" :key="wf.id" :label="wf.name" :value="wf.id" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :md="12" :sm="24">
              <el-form-item label="事件来源">
                <el-input v-model.trim="runbookForm.source" placeholder="manual-runbook" />
              </el-form-item>
            </el-col>
            <el-col :md="12" :sm="24">
              <el-form-item label="事件类型">
                <el-input v-model.trim="runbookForm.event_type" placeholder="runbook.execute" />
              </el-form-item>
            </el-col>
            <el-col :md="12" :sm="24">
              <el-form-item label="摘要">
                <el-input v-model.trim="runbookForm.summary" placeholder="手动触发编排任务" />
              </el-form-item>
            </el-col>
            <el-col :span="24">
              <el-form-item label="运行变量 JSON">
                <el-input v-model="runbookForm.variables" type="textarea" :rows="6" placeholder='{"service":"api-gateway","env":"prod"}' />
              </el-form-item>
            </el-col>
            <el-col :span="24">
              <el-form-item label="事件载荷 JSON">
                <el-input v-model="runbookForm.payload" type="textarea" :rows="6" placeholder='{"alarm":"cpu_high","host":"10.0.0.8"}' />
              </el-form-item>
            </el-col>
          </el-row>
        </el-form>
        <div class="runbook-actions">
          <el-button type="primary" :loading="runbookExecuting" @click="executeRunbook">执行 Runbook</el-button>
        </div>
      </el-tab-pane>
    </el-tabs>

    <el-dialog append-to-body v-model="ruleDialogVisible" :title="ruleForm.id ? '编辑规则' : '新建规则'" width="760px" @closed="resetRuleForm">
      <el-form label-width="110px">
        <el-row :gutter="12">
          <el-col :md="12" :sm="24">
            <el-form-item label="规则名称" required>
              <el-input v-model.trim="ruleForm.name" />
            </el-form-item>
          </el-col>
          <el-col :md="12" :sm="24">
            <el-form-item label="目标 Runbook" required>
              <el-select v-model="ruleForm.workflow_id" filterable clearable style="width: 100%">
                <el-option v-for="wf in workflows" :key="wf.id" :label="wf.name" :value="wf.id" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :md="12" :sm="24">
            <el-form-item label="来源匹配">
              <el-input v-model.trim="ruleForm.source" placeholder="* 表示匹配全部" />
            </el-form-item>
          </el-col>
          <el-col :md="12" :sm="24">
            <el-form-item label="事件匹配">
              <el-input v-model.trim="ruleForm.event_type" placeholder="* 表示匹配全部" />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="Payload 包含">
              <el-input v-model.trim="ruleForm.match_contains" placeholder="命中关键字（可选）" />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="默认变量 JSON">
              <el-input v-model="ruleForm.default_variables" type="textarea" :rows="5" placeholder='{"service":"payment"}' />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="是否启用">
              <el-switch v-model="ruleForm.enabled" />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
      <template #footer>
        <el-button @click="ruleDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="ruleSaving" @click="saveRule">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog append-to-body v-model="ingestDialogVisible" title="注入事件（调试）" width="760px" @closed="resetIngestForm">
      <el-form label-width="110px">
        <el-row :gutter="12">
          <el-col :md="12" :sm="24">
            <el-form-item label="来源 source">
              <el-input v-model.trim="ingestForm.source" placeholder="manual" />
            </el-form-item>
          </el-col>
          <el-col :md="12" :sm="24">
            <el-form-item label="类型 event_type">
              <el-input v-model.trim="ingestForm.event_type" placeholder="domain.cert_expiring" />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="摘要">
              <el-input v-model.trim="ingestForm.summary" placeholder="证书即将过期，需触发通知与工单" />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="变量 JSON">
              <el-input v-model="ingestForm.variables" type="textarea" :rows="4" placeholder='{"domain":"example.com"}' />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="Payload JSON">
              <el-input v-model="ingestForm.payload" type="textarea" :rows="6" placeholder='{"domain":"example.com","days_to_expire":7}' />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
      <template #footer>
        <el-button @click="ingestDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="ingestSubmitting" @click="submitIngest">提交注入</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { onMounted, reactive, ref, watch } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getErrorMessage, isCancelError } from '@/utils/error'
import StatusBadge from '@/components/common/StatusBadge.vue'
import { booleanEnabledStatusMeta, workflowDispatchStatusMeta, workflowEventStatusMeta } from '@/utils/status'

const activeTab = ref('rules')

const overview = ref({
  rules_total: 0,
  rules_enabled: 0,
  events_24h: 0,
  dispatched_24h: 0,
  failed_24h: 0
})

const workflows = ref([])

const rules = ref([])
const rulesLoading = ref(false)
const ruleSaving = ref(false)
const ruleDialogVisible = ref(false)
const ruleForm = reactive({
  id: '',
  name: '',
  source: '*',
  event_type: '*',
  workflow_id: '',
  match_contains: '',
  default_variables: '{}',
  enabled: true
})
const ruleFilter = reactive({
  source: '',
  event_type: '',
  enabled: ''
})

const events = ref([])
const eventsLoading = ref(false)
const eventFilter = reactive({
  source: '',
  event_type: '',
  status: ''
})

const dispatches = ref([])
const dispatchesLoading = ref(false)
const dispatchFilter = reactive({
  event_id: '',
  status: ''
})

const ingestDialogVisible = ref(false)
const ingestSubmitting = ref(false)
const ingestForm = reactive({
  source: 'manual',
  event_type: '',
  summary: '',
  variables: '{}',
  payload: '{}'
})

const runbookExecuting = ref(false)
const runbookForm = reactive({
  workflow_id: '',
  source: 'manual-runbook',
  event_type: 'runbook.execute',
  summary: '',
  variables: '{}',
  payload: '{}'
})

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const formatTime = (value) => {
  if (!value) return '-'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return '-'
  return d.toLocaleString()
}

const ruleEnabledBadge = (row) => booleanEnabledStatusMeta(row, {
  source: '编排规则',
  enabledReason: '规则已启用',
  disabledReason: '规则已停用',
  checkAt: row?.updated_at || row?.last_triggered_at
})

const eventStatusBadge = (row) => workflowEventStatusMeta(row, {
  source: '编排事件',
  checkAt: row?.updated_at || row?.received_at
})

const dispatchStatusBadge = (row) => workflowDispatchStatusMeta(row, {
  source: '编排分发',
  checkAt: row?.finished_at || row?.started_at
})

const parseJSONObject = (raw, fallback = {}) => {
  const content = String(raw || '').trim()
  if (!content) return fallback
  try {
    const obj = JSON.parse(content)
    if (obj && typeof obj === 'object' && !Array.isArray(obj)) return obj
    throw new Error('JSON 必须是对象')
  } catch (err) {
    throw new Error(err?.message || 'JSON 格式错误')
  }
}

const fetchOverview = async () => {
  try {
    const res = await axios.get('/api/v1/orchestrator/overview', { headers: authHeaders() })
    if (res.data?.code === 0) overview.value = res.data.data || overview.value
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '加载编排概览失败'))
  }
}

const fetchWorkflows = async () => {
  try {
    const res = await axios.get('/api/v1/workflow/workflows', { headers: authHeaders() })
    if (res.data?.code === 0) workflows.value = res.data.data || []
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '加载工作流列表失败'))
  }
}

const fetchRules = async () => {
  rulesLoading.value = true
  try {
    const params = {}
    if (ruleFilter.source) params.source = ruleFilter.source
    if (ruleFilter.event_type) params.event_type = ruleFilter.event_type
    if (ruleFilter.enabled !== '') params.enabled = ruleFilter.enabled
    const res = await axios.get('/api/v1/orchestrator/rules', { params, headers: authHeaders() })
    if (res.data?.code === 0) rules.value = res.data.data || []
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '加载规则失败'))
  } finally {
    rulesLoading.value = false
  }
}

const fetchEvents = async () => {
  eventsLoading.value = true
  try {
    const params = {}
    if (eventFilter.source) params.source = eventFilter.source
    if (eventFilter.event_type) params.event_type = eventFilter.event_type
    if (eventFilter.status) params.status = eventFilter.status
    const res = await axios.get('/api/v1/orchestrator/events', { params, headers: authHeaders() })
    if (res.data?.code === 0) events.value = res.data.data || []
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '加载事件失败'))
  } finally {
    eventsLoading.value = false
  }
}

const fetchDispatches = async () => {
  dispatchesLoading.value = true
  try {
    const params = {}
    if (dispatchFilter.event_id) params.event_id = dispatchFilter.event_id
    if (dispatchFilter.status) params.status = dispatchFilter.status
    const res = await axios.get('/api/v1/orchestrator/dispatches', { params, headers: authHeaders() })
    if (res.data?.code === 0) dispatches.value = res.data.data || []
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '加载分发记录失败'))
  } finally {
    dispatchesLoading.value = false
  }
}

const resetRuleForm = () => {
  ruleForm.id = ''
  ruleForm.name = ''
  ruleForm.source = '*'
  ruleForm.event_type = '*'
  ruleForm.workflow_id = ''
  ruleForm.match_contains = ''
  ruleForm.default_variables = '{}'
  ruleForm.enabled = true
}

const openRuleDialog = (row) => {
  if (!row) {
    resetRuleForm()
  } else {
    ruleForm.id = row.id
    ruleForm.name = row.name || ''
    ruleForm.source = row.source || '*'
    ruleForm.event_type = row.event_type || '*'
    ruleForm.workflow_id = row.workflow_id || ''
    ruleForm.match_contains = row.match_contains || ''
    ruleForm.default_variables = row.default_variables || '{}'
    ruleForm.enabled = row.enabled !== false
  }
  ruleDialogVisible.value = true
}

const saveRule = async () => {
  const name = ruleForm.name.trim()
  if (!name) {
    ElMessage.warning('请输入规则名称')
    return
  }
  if (!ruleForm.workflow_id) {
    ElMessage.warning('请选择目标 Runbook')
    return
  }
  let defaultVariables = '{}'
  try {
    defaultVariables = JSON.stringify(parseJSONObject(ruleForm.default_variables, {}))
  } catch (err) {
    ElMessage.warning(`默认变量 JSON 错误：${err.message}`)
    return
  }

  const payload = {
    name,
    source: ruleForm.source || '*',
    event_type: ruleForm.event_type || '*',
    workflow_id: ruleForm.workflow_id,
    match_contains: ruleForm.match_contains || '',
    default_variables: defaultVariables,
    enabled: Boolean(ruleForm.enabled)
  }

  ruleSaving.value = true
  try {
    const url = ruleForm.id ? `/api/v1/orchestrator/rules/${ruleForm.id}` : '/api/v1/orchestrator/rules'
    const method = ruleForm.id ? 'put' : 'post'
    const res = await axios[method](url, payload, { headers: authHeaders() })
    if (res.data?.code !== 0) throw new Error(res.data?.message || '保存规则失败')
    ElMessage.success('规则保存成功')
    ruleDialogVisible.value = false
    await Promise.all([fetchRules(), fetchOverview()])
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '保存规则失败'))
  } finally {
    ruleSaving.value = false
  }
}

const removeRule = async (row) => {
  try {
    await ElMessageBox.confirm(`确认删除规则「${row.name || '-'}」吗？`, '提示', { type: 'warning' })
    const res = await axios.delete(`/api/v1/orchestrator/rules/${row.id}`, { headers: authHeaders() })
    if (res.data?.code !== 0) throw new Error(res.data?.message || '删除失败')
    ElMessage.success('规则已删除')
    await Promise.all([fetchRules(), fetchOverview()])
  } catch (err) {
    if (isCancelError(err)) return
    ElMessage.error(getErrorMessage(err, '删除规则失败'))
  }
}

const resetRuleFilter = () => {
  ruleFilter.source = ''
  ruleFilter.event_type = ''
  ruleFilter.enabled = ''
  fetchRules()
}

const resetEventFilter = () => {
  eventFilter.source = ''
  eventFilter.event_type = ''
  eventFilter.status = ''
  fetchEvents()
}

const resetDispatchFilter = () => {
  dispatchFilter.event_id = ''
  dispatchFilter.status = ''
  fetchDispatches()
}

const resetIngestForm = () => {
  ingestForm.source = 'manual'
  ingestForm.event_type = ''
  ingestForm.summary = ''
  ingestForm.variables = '{}'
  ingestForm.payload = '{}'
}

const openIngestDialog = () => {
  ingestDialogVisible.value = true
}

const submitIngest = async () => {
  let variables = {}
  let payload = {}
  try {
    variables = parseJSONObject(ingestForm.variables, {})
    payload = parseJSONObject(ingestForm.payload, {})
  } catch (err) {
    ElMessage.warning(`注入参数错误：${err.message}`)
    return
  }

  ingestSubmitting.value = true
  try {
    const req = {
      source: ingestForm.source || 'manual',
      event_type: ingestForm.event_type || '',
      summary: ingestForm.summary || '',
      variables,
      payload
    }
    const res = await axios.post('/api/v1/orchestrator/events/ingest', req, { headers: authHeaders() })
    if (res.data?.code !== 0) throw new Error(res.data?.message || '注入失败')
    ElMessage.success('事件注入成功')
    ingestDialogVisible.value = false
    await Promise.all([fetchOverview(), fetchEvents(), fetchDispatches()])
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '事件注入失败'))
  } finally {
    ingestSubmitting.value = false
  }
}

const prefillRunbookFromRule = (row) => {
  runbookForm.workflow_id = row.workflow_id || ''
  runbookForm.source = row.source && row.source !== '*' ? row.source : 'manual-runbook'
  runbookForm.event_type = row.event_type && row.event_type !== '*' ? row.event_type : 'runbook.execute'
  runbookForm.summary = `手动触发规则：${row.name || '-'}`
  runbookForm.variables = row.default_variables || '{}'
  runbookForm.payload = '{}'
  activeTab.value = 'runbook'
}

const executeRunbook = async () => {
  if (!runbookForm.workflow_id) {
    ElMessage.warning('请选择目标 Runbook')
    return
  }
  let variables = {}
  let payload = {}
  try {
    variables = parseJSONObject(runbookForm.variables, {})
    payload = parseJSONObject(runbookForm.payload, {})
  } catch (err) {
    ElMessage.warning(`JSON 参数错误：${err.message}`)
    return
  }
  runbookExecuting.value = true
  try {
    const req = {
      workflow_id: runbookForm.workflow_id,
      source: runbookForm.source || 'manual-runbook',
      event_type: runbookForm.event_type || 'runbook.execute',
      summary: runbookForm.summary || '',
      variables,
      payload
    }
    const res = await axios.post('/api/v1/orchestrator/runbooks/execute', req, { headers: authHeaders() })
    if (res.data?.code !== 0) throw new Error(res.data?.message || '执行失败')
    ElMessage.success('Runbook 已提交执行')
    await Promise.all([fetchOverview(), fetchEvents(), fetchDispatches()])
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '执行 Runbook 失败'))
  } finally {
    runbookExecuting.value = false
  }
}

const refreshActiveTab = async () => {
  await fetchOverview()
  if (activeTab.value === 'rules') {
    await fetchRules()
  } else if (activeTab.value === 'events') {
    await fetchEvents()
  } else if (activeTab.value === 'dispatches') {
    await fetchDispatches()
  }
}

watch(activeTab, (tab) => {
  if (tab === 'rules') fetchRules()
  if (tab === 'events') fetchEvents()
  if (tab === 'dispatches') fetchDispatches()
})

onMounted(async () => {
  await Promise.all([fetchOverview(), fetchWorkflows(), fetchRules(), fetchEvents(), fetchDispatches()])
})
</script>

<style scoped>
.page-card {
  border-radius: 18px;
}

.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 12px;
}

.page-header h2 {
  margin: 0;
  font-size: 34px;
  font-weight: 700;
}

.page-desc {
  margin: 8px 0 0;
  color: var(--el-text-color-secondary);
}

.page-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.mb-12 {
  margin-bottom: 12px;
}

.k {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.v {
  margin-top: 6px;
  font-size: 34px;
  font-weight: 700;
  color: var(--el-text-color-primary);
}

.v.success {
  color: #22c55e;
}

.v.info {
  color: #3b82f6;
}

.v.danger {
  color: #ef4444;
}

.toolbar {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 12px;
}

.runbook-form {
  max-width: 1100px;
}

.runbook-actions {
  display: flex;
  justify-content: flex-end;
}
</style>
