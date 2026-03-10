<template>
  <el-card class="page-card">
    <template #header>
      <div class="header">
        <div>
          <div class="title">会话审计</div>
          <div class="desc">发起会话、关闭会话并查看命令审计记录。</div>
        </div>
        <div class="actions">
          <el-button type="primary" icon="VideoPlay" @click="openStartDialog">发起会话</el-button>
          <el-button icon="Refresh" @click="loadSessions">刷新</el-button>
        </div>
      </div>
    </template>

    <div class="toolbar">
      <el-input v-model="filters.username" clearable placeholder="按用户过滤" class="filter-item" @change="loadSessions" />
      <el-select v-model="filters.status" clearable placeholder="状态" class="filter-item" @change="loadSessions">
        <el-option label="pending_approval" value="pending_approval" />
        <el-option label="active" value="active" />
        <el-option label="closed" value="closed" />
        <el-option label="blocked" value="blocked" />
        <el-option label="rejected" value="rejected" />
      </el-select>
      <el-select v-model="filters.asset_id" clearable filterable placeholder="资产" class="filter-item" @change="loadSessions">
        <el-option v-for="item in assets" :key="item.id" :label="item.name" :value="item.id" />
      </el-select>
    </div>

    <el-table :fit="true" :data="sessions" v-loading="loading" stripe>
      <el-table-column prop="session_no" label="会话号" min-width="180" />
      <el-table-column prop="username" label="用户" width="120" />
      <el-table-column prop="asset_name" label="资产" min-width="150" />
      <el-table-column prop="account_name" label="账号" min-width="120" />
      <el-table-column prop="protocol" label="协议" width="90" />
      <el-table-column label="状态" width="90">
        <template #default="{ row }">
          <el-tag :type="statusType(row.status)">{{ row.status }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="command_count" label="命令数" width="90" />
      <el-table-column label="开始时间" min-width="170">
        <template #default="{ row }">{{ formatTime(row.started_at) }}</template>
      </el-table-column>
      <el-table-column label="审批人" min-width="120">
        <template #default="{ row }">{{ row.approved_by || '-' }}</template>
      </el-table-column>
      <el-table-column label="结束时间" min-width="170">
        <template #default="{ row }">{{ formatTime(row.ended_at) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="560" fixed="right">
        <template #default="{ row }">
          <el-button
            v-if="isAdmin && row.status === 'pending_approval'"
            size="small"
            type="success"
            plain
            @click="approveSession(row)"
          >
            通过
          </el-button>
          <el-button
            v-if="isAdmin && row.status === 'pending_approval'"
            size="small"
            type="danger"
            plain
            @click="rejectSession(row)"
          >
            拒绝
          </el-button>
          <el-button size="small" type="success" plain :disabled="row.status !== 'active'" @click="connectSession(row)">连接</el-button>
          <el-button size="small" type="primary" plain @click="openCommands(row)">命令审计</el-button>
          <el-button size="small" type="warning" plain @click="openRiskEvents(row)">风险事件</el-button>
          <el-button size="small" plain :disabled="row.status !== 'active'" @click="openRecordDialog(row)">录入命令</el-button>
          <el-button size="small" type="danger" plain :disabled="row.status !== 'active'" @click="closeSession(row)">关闭</el-button>
          <el-button
            v-if="isAdmin"
            size="small"
            type="danger"
            plain
            :disabled="!['active','pending_approval'].includes(row.status)"
            @click="disconnectSession(row)"
          >
            强制断开
          </el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-card>

  <el-dialog append-to-body v-model="startDialogVisible" title="发起会话" width="560px">
    <el-form :model="startForm" label-width="92px">
      <el-form-item label="资产" required>
        <el-select v-model="startForm.asset_id" filterable style="width: 100%">
          <el-option v-for="item in assets" :key="item.id" :label="`${item.name}(${item.protocol})`" :value="item.id" />
        </el-select>
      </el-form-item>
      <el-form-item label="账号" required>
        <el-select v-model="startForm.account_id" filterable style="width: 100%">
          <el-option v-for="item in accounts" :key="item.id" :label="`${item.name}/${item.username}`" :value="item.id" />
        </el-select>
      </el-form-item>
      <el-form-item label="协议">
        <el-select v-model="startForm.protocol" clearable style="width: 100%">
          <el-option label="ssh" value="ssh" />
          <el-option label="docker" value="docker" />
          <el-option label="k8s" value="k8s" />
          <el-option label="mysql" value="mysql" />
          <el-option label="postgres" value="postgres" />
          <el-option label="redis" value="redis" />
          <el-option label="mongodb" value="mongodb" />
        </el-select>
      </el-form-item>
      <el-form-item label="来源IP">
        <el-input v-model="startForm.source_ip" placeholder="仅展示，后端将自动记录客户端真实IP" disabled />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="startDialogVisible = false">取消</el-button>
      <el-button type="primary" :loading="starting" @click="startSession">开始</el-button>
    </template>
  </el-dialog>

  <el-dialog append-to-body v-model="recordDialogVisible" title="录入命令审计" width="620px">
    <el-form :model="commandForm" label-width="90px">
      <el-form-item label="命令" required>
        <el-input v-model="commandForm.command" type="textarea" :rows="3" />
      </el-form-item>
      <el-form-item label="返回码">
        <el-input-number v-model="commandForm.result_code" :min="-1" :max="9999" />
      </el-form-item>
      <el-form-item label="输出摘要">
        <el-input v-model="commandForm.output_snippet" type="textarea" :rows="4" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="recordDialogVisible = false">取消</el-button>
      <el-button type="primary" :loading="recording" @click="recordCommand">提交</el-button>
    </template>
  </el-dialog>

  <el-dialog append-to-body v-model="sqlConsoleVisible" title="SQL 控制台" width="760px">
    <el-form :model="sqlForm" label-width="90px">
      <el-form-item label="会话">
        <el-input :model-value="`${currentSession?.session_no || '-'} / ${currentSession?.asset_name || '-'}`" disabled />
      </el-form-item>
      <el-form-item label="数据库">
        <el-input v-model="sqlForm.database" placeholder="可选，留空按资产默认数据库执行" />
      </el-form-item>
      <el-form-item label="SQL" required>
        <el-input v-model="sqlForm.sql" type="textarea" :rows="6" placeholder="一次只执行一条 SQL" />
      </el-form-item>
    </el-form>
    <div v-if="sqlResultText" class="sql-result">
      <pre>{{ sqlResultText }}</pre>
    </div>
    <template #footer>
      <el-button @click="sqlConsoleVisible = false">关闭</el-button>
      <el-button type="primary" :loading="sqlExecuting" @click="executeSQL">执行并审计</el-button>
    </template>
  </el-dialog>

  <el-drawer v-model="commandsVisible" title="命令审计记录" size="52%">
    <template #default>
      <div class="drawer-subtitle">会话：{{ currentSession?.session_no || '-' }} / {{ currentSession?.asset_name || '-' }}</div>
      <el-table :fit="true" :data="commands" v-loading="commandsLoading" stripe>
        <el-table-column label="执行时间" min-width="170">
          <template #default="{ row }">{{ formatTime(row.executed_at) }}</template>
        </el-table-column>
        <el-table-column prop="username" label="用户" width="110" />
        <el-table-column prop="command_type" label="类型" width="90" />
        <el-table-column prop="result_code" label="返回码" width="90" />
        <el-table-column label="风险级别" width="110">
          <template #default="{ row }">
            <el-tag v-if="row.risk_level" :type="riskType(row.risk_level)">{{ row.risk_level }}</el-tag>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="动作" width="110">
          <template #default="{ row }">
            <el-tag v-if="row.risk_action" :type="riskActionType(row)">{{ row.risk_action }}</el-tag>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="命中规则" min-width="220" show-overflow-tooltip>
          <template #default="{ row }">{{ formatRuleNames(row) }}</template>
        </el-table-column>
        <el-table-column prop="risk_reason" label="判定说明" min-width="220" show-overflow-tooltip />
        <el-table-column prop="command" label="命令" min-width="220" show-overflow-tooltip />
        <el-table-column prop="output_snippet" label="输出摘要" min-width="260" show-overflow-tooltip />
      </el-table>
    </template>
  </el-drawer>

  <el-drawer v-model="riskEventsVisible" title="风险事件" size="48%">
    <template #default>
      <div class="drawer-subtitle">会话：{{ currentSession?.session_no || '-' }} / {{ currentSession?.asset_name || '-' }}</div>
      <el-table :fit="true" :data="riskEvents" v-loading="riskEventsLoading" stripe>
        <el-table-column label="触发时间" min-width="170">
          <template #default="{ row }">{{ formatTime(row.fired_at) }}</template>
        </el-table-column>
        <el-table-column prop="username" label="用户" width="110" />
        <el-table-column prop="event_type" label="事件类型" width="110" />
        <el-table-column label="级别" width="90">
          <template #default="{ row }">
            <el-tag :type="riskType(row.severity)">{{ row.severity }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="rule_name" label="命中规则" min-width="180" show-overflow-tooltip />
        <el-table-column prop="description" label="说明" min-width="180" show-overflow-tooltip />
        <el-table-column prop="command" label="命令" min-width="220" show-overflow-tooltip />
      </el-table>
    </template>
  </el-drawer>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getErrorMessage, isCancelError } from '@/utils/error'

const loading = ref(false)
const starting = ref(false)
const recording = ref(false)
const commandsLoading = ref(false)
const riskEventsLoading = ref(false)
const sqlExecuting = ref(false)

const sessions = ref([])
const assets = ref([])
const accounts = ref([])
const commands = ref([])
const riskEvents = ref([])

const currentSession = ref(null)

const startDialogVisible = ref(false)
const recordDialogVisible = ref(false)
const sqlConsoleVisible = ref(false)
const commandsVisible = ref(false)
const riskEventsVisible = ref(false)

const recordSessionID = ref('')
const sqlSessionID = ref('')

const filters = reactive({
  username: '',
  status: '',
  asset_id: ''
})

const startForm = reactive({
  asset_id: '',
  account_id: '',
  protocol: '',
  source_ip: ''
})

const commandForm = reactive({
  command: '',
  result_code: 0,
  output_snippet: ''
})

const sqlForm = reactive({
  database: '',
  sql: ''
})
const sqlResultText = ref('')

const isAdmin = localStorage.getItem('role_code') === 'admin'
const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const formatTime = (value) => {
  if (!value) return '-'
  return String(value).replace('T', ' ').replace('Z', '')
}

const statusType = (status) => {
  if (status === 'pending_approval') return 'warning'
  if (status === 'active') return 'success'
  if (status === 'rejected') return 'danger'
  if (status === 'blocked') return 'danger'
  return 'info'
}

const riskType = (level) => {
  if (level === 'critical') return 'danger'
  if (level === 'info') return 'info'
  return 'warning'
}

const riskActionType = (row) => {
  if (row.whitelist_hit || row.risk_action === 'allow') return 'success'
  if (row.blocked || row.risk_action === 'block') return 'danger'
  return 'warning'
}

const formatRuleNames = (row) => {
  if (!row) return '-'
  if (row.rule_name) return row.rule_name
  if (row.whitelist_hit) return '白名单'
  return '-'
}

const loadAssets = async () => {
  const res = await axios.get('/api/v1/jump/assets', { headers: authHeaders() })
  if (res.data.code === 0) assets.value = res.data.data || []
}

const loadAccounts = async () => {
  const res = await axios.get('/api/v1/jump/accounts', { headers: authHeaders() })
  if (res.data.code === 0) accounts.value = res.data.data || []
}

const loadSessions = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/v1/jump/sessions', {
      headers: authHeaders(),
      params: {
        username: filters.username || undefined,
        status: filters.status || undefined,
        asset_id: filters.asset_id || undefined
      }
    })
    if (res.data.code === 0) sessions.value = res.data.data || []
  } catch (error) {
    ElMessage.error(getErrorMessage(error, '加载会话失败'))
  } finally {
    loading.value = false
  }
}

const openStartDialog = async () => {
  Object.assign(startForm, {
    asset_id: '',
    account_id: '',
    protocol: '',
    source_ip: ''
  })
  await Promise.all([loadAssets(), loadAccounts()])
  startDialogVisible.value = true
}

const startSession = async () => {
  if (!startForm.asset_id || !startForm.account_id) {
    ElMessage.warning('请选择资产与账号')
    return
  }
  starting.value = true
  try {
    const res = await axios.post('/api/v1/jump/sessions/start', { ...startForm }, { headers: authHeaders() })
    if (res.data.code === 0) {
      const needApprove = !!res.data?.data?.need_approve
      ElMessage.success(needApprove ? '会话已提交审批' : '会话已创建')
      startDialogVisible.value = false
      loadSessions()
    }
  } catch (error) {
    ElMessage.error(getErrorMessage(error, '发起会话失败'))
  } finally {
    starting.value = false
  }
}

const closeSession = async (row) => {
  try {
    await ElMessageBox.confirm(`确认关闭会话 ${row.session_no} 吗？`, '提示', { type: 'warning' })
    await axios.post(`/api/v1/jump/sessions/${row.id}/close`, { close_reason: 'manual' }, { headers: authHeaders() })
    ElMessage.success('会话已关闭')
    await loadSessions()
  } catch (error) {
    if (!isCancelError(error)) ElMessage.error(getErrorMessage(error, '关闭会话失败'))
  }
}

const disconnectSession = async (row) => {
  try {
    const { value } = await ElMessageBox.prompt(`请输入断开原因（会话 ${row.session_no}）`, '强制断开', {
      confirmButtonText: '确认',
      cancelButtonText: '取消',
      inputPlaceholder: '例如：检测到高风险操作'
    })
    await axios.post(`/api/v1/jump/sessions/${row.id}/disconnect`, {
      reason: value || '管理员强制断开'
    }, { headers: authHeaders() })
    ElMessage.success('会话已强制断开')
    await loadSessions()
  } catch (error) {
    if (!isCancelError(error)) ElMessage.error(getErrorMessage(error, '强制断开失败'))
  }
}

const approveSession = async (row) => {
  try {
    await ElMessageBox.confirm(`确认通过会话 ${row.session_no} 吗？`, '审批确认', { type: 'warning' })
    await axios.post(`/api/v1/jump/sessions/${row.id}/approve`, {}, { headers: authHeaders() })
    ElMessage.success('已通过')
    await loadSessions()
  } catch (error) {
    if (!isCancelError(error)) ElMessage.error(getErrorMessage(error, '审批失败'))
  }
}

const rejectSession = async (row) => {
  try {
    const { value } = await ElMessageBox.prompt(`请输入拒绝原因（会话 ${row.session_no}）`, '审批拒绝', {
      confirmButtonText: '确认',
      cancelButtonText: '取消',
      inputPlaceholder: '例如：不在授权时间窗口内',
      inputValue: ''
    })
    await axios.post(`/api/v1/jump/sessions/${row.id}/reject`, { reason: value || '' }, { headers: authHeaders() })
    ElMessage.success('已拒绝')
    await loadSessions()
  } catch (error) {
    if (!isCancelError(error)) ElMessage.error(getErrorMessage(error, '拒绝失败'))
  }
}

const openRecordDialog = (row) => {
  recordSessionID.value = row.id
  Object.assign(commandForm, {
    command: '',
    result_code: 0,
    output_snippet: ''
  })
  recordDialogVisible.value = true
}

const openSQLConsole = (row) => {
  currentSession.value = row
  sqlSessionID.value = row.id
  sqlResultText.value = ''
  Object.assign(sqlForm, {
    database: '',
    sql: ''
  })
  sqlConsoleVisible.value = true
}

const recordCommand = async () => {
  if (!commandForm.command) {
    ElMessage.warning('请填写命令内容')
    return
  }
  recording.value = true
  try {
    const res = await axios.post(`/api/v1/jump/sessions/${recordSessionID.value}/commands`, { ...commandForm }, { headers: authHeaders() })
    if (res.data.code === 0) {
      ElMessage.success('命令审计已记录')
      recordDialogVisible.value = false
      loadSessions()
      if (currentSession.value?.id === recordSessionID.value) {
        openCommands(currentSession.value)
      }
    }
  } catch (error) {
    ElMessage.error(getErrorMessage(error, '记录失败'))
  } finally {
    recording.value = false
  }
}

const openCommands = async (row) => {
  currentSession.value = row
  commandsVisible.value = true
  commandsLoading.value = true
  try {
    const res = await axios.get(`/api/v1/jump/sessions/${row.id}/commands`, { headers: authHeaders() })
    if (res.data.code === 0) {
      commands.value = res.data.data || []
    }
  } catch (error) {
    ElMessage.error(getErrorMessage(error, '加载命令失败'))
  } finally {
    commandsLoading.value = false
  }
}

const openRiskEvents = async (row) => {
  currentSession.value = row
  riskEventsVisible.value = true
  riskEventsLoading.value = true
  try {
    const res = await axios.get('/api/v1/jump/risk-events', {
      headers: authHeaders(),
      params: { session_id: row.id }
    })
    if (res.data.code === 0) {
      riskEvents.value = res.data.data || []
    }
  } catch (error) {
    ElMessage.error(getErrorMessage(error, '加载风险事件失败'))
  } finally {
    riskEventsLoading.value = false
  }
}

const connectSession = async (row) => {
  try {
    const res = await axios.post(`/api/v1/jump/sessions/${row.id}/connect`, {}, { headers: authHeaders() })
    if (res.data.code !== 0) return
    const data = res.data.data || {}
    if (data.mode === 'sql' || ['mysql', 'postgres'].includes(String(row.protocol || '').toLowerCase())) {
      openSQLConsole(row)
      return
    }
    const openUrl = data.open_url
    if (!openUrl) {
      ElMessage.warning('未返回可连接地址')
      return
    }
    window.open(openUrl, '_blank')
  } catch (error) {
    ElMessage.error(getErrorMessage(error, '连接失败'))
  }
}

const executeSQL = async () => {
  if (!sqlSessionID.value) {
    ElMessage.warning('会话不存在')
    return
  }
  if (!sqlForm.sql.trim()) {
    ElMessage.warning('请填写 SQL')
    return
  }
  sqlExecuting.value = true
  try {
    const res = await axios.post(
      `/api/v1/jump/sessions/${sqlSessionID.value}/sql/execute`,
      { sql: sqlForm.sql, database: sqlForm.database || '' },
      { headers: authHeaders() }
    )
    if (res.data.code === 0) {
      const data = res.data.data || {}
      sqlResultText.value = JSON.stringify(data, null, 2)
      ElMessage.success('SQL 执行并审计成功')
      await Promise.all([loadSessions(), openCommands(currentSession.value)])
    } else {
      sqlResultText.value = JSON.stringify(res.data, null, 2)
      ElMessage.warning(res.data.message || 'SQL 执行失败')
    }
  } catch (error) {
    const payload = error?.response?.data || { message: 'SQL 执行失败' }
    sqlResultText.value = JSON.stringify(payload, null, 2)
    ElMessage.error(payload.message || 'SQL 执行失败')
  } finally {
    sqlExecuting.value = false
  }
}

onMounted(async () => {
  await Promise.all([loadAssets(), loadAccounts(), loadSessions()])
})
</script>

<style scoped>
.page-card { max-width: 100%; }
.header { display: flex; justify-content: space-between; align-items: center; gap: 12px; }
.title { font-size: 18px; font-weight: 600; }
.desc { color: #909399; margin-top: 4px; }
.actions { display: flex; gap: 8px; }
.toolbar { margin-bottom: 12px; display: flex; gap: 8px; flex-wrap: wrap; }
.filter-item { width: 220px; }
.drawer-subtitle { margin-bottom: 12px; color: #606266; }
.sql-result { margin-top: 8px; max-height: 220px; overflow: auto; background: #0b1220; color: #cfd8e3; border-radius: 8px; padding: 10px; }
.sql-result pre { margin: 0; white-space: pre-wrap; word-break: break-word; font-size: 12px; line-height: 1.4; }
@media (max-width: 768px) {
  .header { flex-direction: column; align-items: flex-start; }
  .filter-item { width: 100%; }
}
</style>
