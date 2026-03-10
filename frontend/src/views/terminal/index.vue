<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>WebTerminal</h2>
        <p class="page-desc">SSH 会话管理、在线终端与操作录像回放。</p>
      </div>
      <div class="page-actions">
        <el-button type="primary" icon="Plus" @click="openCreateDialog">新建会话</el-button>
        <el-button icon="Refresh" @click="reloadAll">刷新</el-button>
      </div>
    </div>

    <el-card shadow="never" class="section-card">
      <template #header>
        <div class="card-title">在线会话</div>
      </template>
      <div class="section-filters">
        <el-input v-model="sessionFilters.keyword" clearable placeholder="搜索主机/操作人/登录用户" style="width: 260px" @keyup.enter="fetchSessions" />
        <el-select v-model="sessionFilters.status" clearable placeholder="状态" style="width: 130px">
          <el-option label="待连" :value="0" />
          <el-option label="在线" :value="1" />
          <el-option label="已关闭" :value="2" />
          <el-option label="失败" :value="3" />
        </el-select>
        <el-button @click="fetchSessions">筛选</el-button>
        <el-button @click="resetSessionFilters">重置</el-button>
      </div>
      <el-table :fit="true" :data="sessions" v-loading="sessionLoading" stripe>
        <el-table-column prop="operator" label="操作人" width="120" />
        <el-table-column prop="host" label="主机" min-width="180" />
        <el-table-column prop="port" label="端口" width="90" />
        <el-table-column prop="username" label="登录用户" width="120" />
        <el-table-column prop="started_at" label="开始时间" width="170">
          <template #default="{ row }">{{ formatTime(row.started_at || row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)">{{ statusText(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="last_error" label="失败原因" min-width="220" show-overflow-tooltip>
          <template #default="{ row }">{{ row.last_error || '-' }}</template>
        </el-table-column>
        <el-table-column label="操作" width="340" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="primary" @click="openTerminal(row)">连接</el-button>
            <el-button v-if="row.status !== 1" size="small" @click="openEditDialog(row)">编辑</el-button>
            <el-button size="small" @click="openRecordBySession(row)">录像</el-button>
            <el-button size="small" type="warning" @click="closeSession(row)">关闭</el-button>
            <el-button size="small" type="danger" plain @click="deleteSession(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card shadow="never" class="section-card">
      <template #header>
        <div class="section-header">
          <div class="card-title">会话录像</div>
          <div class="record-actions">
            <el-button size="small" :disabled="!selectedRecordIds.length" @click="exportSelectedRecords('json')">批量导出 JSON</el-button>
            <el-button size="small" :disabled="!selectedRecordIds.length" @click="exportSelectedRecords('cast')">批量导出 Cast</el-button>
            <el-button size="small" @click="cleanupRecords(7)">清理 7 天前</el-button>
            <el-button size="small" @click="cleanupRecords(30)">清理 30 天前</el-button>
          </div>
        </div>
      </template>
      <div class="section-filters">
        <el-input v-model="recordFilters.keyword" clearable placeholder="搜索主机/操作人/会话ID" style="width: 260px" @keyup.enter="fetchRecords" />
        <el-button @click="fetchRecords">筛选</el-button>
        <el-button @click="resetRecordFilters">重置</el-button>
      </div>
      <el-table :fit="true" :data="records" v-loading="recordLoading" stripe @selection-change="onRecordSelectionChange">
        <el-table-column type="selection" width="48" />
        <el-table-column prop="operator" label="操作人" width="120" />
        <el-table-column prop="host" label="主机" min-width="180" />
        <el-table-column prop="duration" label="时长(s)" width="90" />
        <el-table-column prop="created_at" label="创建时间" width="170">
          <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="330" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="viewRecord(row)">回放</el-button>
            <el-button size="small" @click="downloadRecord(row)">下载</el-button>
            <el-button size="small" @click="downloadRecordCast(row)">导出 Cast</el-button>
            <el-button size="small" type="danger" plain @click="deleteRecord(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card shadow="never" class="section-card">
      <template #header>
        <div class="card-title">命令审计</div>
      </template>
      <div class="section-filters">
        <el-input v-model="auditFilters.keyword" clearable placeholder="搜索命令/主机/操作人/登录用户/会话号" style="width: 320px" @keyup.enter="fetchAudits" />
        <el-select v-model="auditFilters.risk_level" clearable placeholder="风险等级" style="width: 130px">
          <el-option label="info" value="info" />
          <el-option label="warning" value="warning" />
          <el-option label="critical" value="critical" />
        </el-select>
        <el-select v-model="auditFilters.blocked" clearable placeholder="执行结果" style="width: 130px">
          <el-option label="已阻断" value="true" />
          <el-option label="已放行" value="false" />
        </el-select>
        <el-button @click="fetchAudits">筛选</el-button>
        <el-button @click="resetAuditFilters">重置</el-button>
      </div>
      <el-table :fit="true" :data="audits" v-loading="auditLoading" stripe>
        <el-table-column prop="executed_at" label="时间" width="170">
          <template #default="{ row }">{{ formatTime(row.executed_at) }}</template>
        </el-table-column>
        <el-table-column prop="session_no" label="会话号" min-width="150" show-overflow-tooltip />
        <el-table-column prop="host" label="主机" min-width="160" show-overflow-tooltip />
        <el-table-column prop="operator" label="操作人" width="110" />
        <el-table-column prop="login_user" label="登录用户" width="110" />
        <el-table-column prop="protocol" label="协议" width="100" />
        <el-table-column prop="command" label="命令" min-width="280" show-overflow-tooltip />
        <el-table-column label="风险" width="110">
          <template #default="{ row }">
            <el-tag v-if="row.risk_level" :type="auditRiskTagType(row.risk_level)">{{ row.risk_level }}</el-tag>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="结果" width="100">
          <template #default="{ row }">
            <el-tag :type="row.blocked ? 'danger' : 'success'">{{ row.blocked ? '阻断' : '放行' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="risk_reason" label="说明" min-width="220" show-overflow-tooltip>
          <template #default="{ row }">{{ row.risk_reason || row.rule_name || '-' }}</template>
        </el-table-column>
        <el-table-column label="操作" width="90" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="openAuditDetail(row)">详情</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog append-to-body v-model="createVisible" :title="editingSessionId ? '编辑终端会话' : '新建终端会话'" width="680px" @closed="onCreateDialogClosed">
      <el-form :model="createForm" label-width="100px">
        <el-form-item label="主机地址">
          <el-input v-model="createForm.host" placeholder="例如 10.0.0.10" />
        </el-form-item>
        <el-form-item label="端口">
          <el-input-number v-model="createForm.port" :min="1" :max="65535" />
        </el-form-item>
        <el-form-item label="用户名">
          <el-input v-model="createForm.username" placeholder="root" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="createForm.password" type="password" show-password placeholder="可选，与私钥二选一" />
        </el-form-item>
        <el-form-item label="私钥">
          <el-input v-model="createForm.key_auth" type="textarea" :rows="4" placeholder="可选，与密码二选一" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button :loading="precheckLoading" @click="precheckConnection">测试连接</el-button>
        <el-button type="primary" @click="submitSession">{{ editingSessionId ? '保存' : '创建并连接' }}</el-button>
      </template>
    </el-dialog>

    <el-dialog append-to-body v-model="terminalVisible" :title="`终端: ${currentSession?.host || ''}`" width="980px" top="4vh" @closed="onTerminalDialogClosed">
      <div class="terminal-toolbar">
        <el-button @click="sendCtrlC">Ctrl+C</el-button>
        <el-button @click="sendResize">同步窗口</el-button>
        <el-button @click="clearTerminal">清屏</el-button>
        <span class="terminal-status">{{ terminalStatus }}</span>
      </div>
      <div ref="terminalRef" class="terminal-output"></div>
      <template #footer>
        <el-button @click="terminalVisible = false">关闭终端窗口</el-button>
      </template>
    </el-dialog>

    <el-dialog append-to-body v-model="recordVisible" title="录像回放" width="980px" top="5vh" @closed="onRecordDialogClosed">
      <div class="record-meta">
        <span>主机: {{ currentRecord?.host || '-' }}</span>
        <span>操作人: {{ currentRecord?.operator || '-' }}</span>
        <span>时长: {{ currentRecord?.duration || 0 }}s</span>
      </div>
      <div class="record-toolbar">
        <el-button size="small" @click="toggleRecordPlayback">
          {{ recordPlaying ? '暂停' : '播放' }}
        </el-button>
        <el-button size="small" @click="restartRecordPlayback">重播</el-button>
        <el-select v-model="recordPlaybackSpeed" size="small" style="width: 120px">
          <el-option label="0.5x" :value="0.5" />
          <el-option label="1x" :value="1" />
          <el-option label="2x" :value="2" />
          <el-option label="4x" :value="4" />
        </el-select>
        <span class="record-progress">
          {{ recordPlaybackIndex }}/{{ recordEvents.length }}
        </span>
      </div>
      <div ref="recordTerminalRef" class="record-output"></div>
      <template #footer>
        <el-button @click="recordVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <el-drawer v-model="auditDetailVisible" title="命令审计详情" size="720px" @closed="onAuditDetailClosed">
      <div v-if="currentAudit" class="audit-detail">
        <div class="audit-grid">
          <div><span class="audit-label">时间</span><span>{{ formatTime(currentAudit.executed_at) }}</span></div>
          <div><span class="audit-label">会话号</span><span>{{ currentAudit.session_no || '-' }}</span></div>
          <div><span class="audit-label">主机</span><span>{{ currentAudit.host || '-' }}</span></div>
          <div><span class="audit-label">资产</span><span>{{ currentAudit.asset_name || '-' }}</span></div>
          <div><span class="audit-label">操作人</span><span>{{ currentAudit.operator || '-' }}</span></div>
          <div><span class="audit-label">登录用户</span><span>{{ currentAudit.login_user || '-' }}</span></div>
          <div><span class="audit-label">协议</span><span>{{ currentAudit.protocol || '-' }}</span></div>
          <div><span class="audit-label">结果</span><span>{{ currentAudit.blocked ? '阻断' : '放行' }}</span></div>
          <div><span class="audit-label">风险等级</span><span>{{ currentAudit.risk_level || '-' }}</span></div>
          <div><span class="audit-label">动作</span><span>{{ currentAudit.risk_action || '-' }}</span></div>
        </div>
        <div class="audit-block">
          <div class="audit-block-title">命令</div>
          <pre>{{ currentAudit.command || '-' }}</pre>
        </div>
        <div class="audit-block">
          <div class="audit-block-title">命中规则</div>
          <pre>{{ currentAudit.rule_name || '-' }}</pre>
        </div>
        <div class="audit-block">
          <div class="audit-block-title">风险说明</div>
          <pre>{{ currentAudit.risk_reason || '-' }}</pre>
        </div>
      </div>
    </el-drawer>
  </el-card>
</template>

<script setup>
import { nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Terminal } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import 'xterm/css/xterm.css'
import { getErrorMessage, isCancelError } from '@/utils/error'

const sessions = ref([])
const records = ref([])
const sessionLoading = ref(false)
const recordLoading = ref(false)
const audits = ref([])
const auditLoading = ref(false)
const sessionFilters = ref({
  keyword: '',
  status: ''
})
const recordFilters = ref({
  keyword: ''
})
const auditFilters = ref({
  keyword: '',
  risk_level: '',
  blocked: ''
})
const selectedRecordIds = ref([])
const auditDetailVisible = ref(false)
const currentAudit = ref(null)

const createVisible = ref(false)
const editingSessionId = ref('')
const precheckLoading = ref(false)
const createForm = ref({
  host_id: '',
  host: '',
  port: 22,
  username: '',
  password: '',
  key_auth: ''
})

const terminalVisible = ref(false)
const terminalRef = ref(null)
const currentSession = ref(null)
const terminalStatus = ref('未连接')
let ws = null
let term = null
let fitAddon = null
let terminalDataListener = null
let keepAliveTimer = null
let reconnectTimer = null
let reconnectAttempts = 0
let manualTerminalClose = false
let wsGeneration = 0

const recordVisible = ref(false)
const currentRecord = ref(null)
const recordTerminalRef = ref(null)
const recordEvents = ref([])
const recordPlaybackIndex = ref(0)
const recordPlaybackSpeed = ref(1)
const recordPlaying = ref(false)
let recordTerm = null
let recordFitAddon = null
let recordPlaybackTimer = null
const route = useRoute()

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const formatTime = (v) => {
  if (!v) return '-'
  return String(v).slice(0, 19).replace('T', ' ')
}

const statusText = (status) => {
  if (status === 1) return '在线'
  if (status === 2) return '已关闭'
  if (status === 3) return '失败'
  return '待连'
}

const statusType = (status) => {
  if (status === 1) return 'success'
  if (status === 3) return 'danger'
  if (status === 2) return 'warning'
  return 'info'
}

const fetchSessions = async () => {
  sessionLoading.value = true
  try {
    const params = {}
    if (sessionFilters.value.keyword.trim()) params.keyword = sessionFilters.value.keyword.trim()
    if (sessionFilters.value.status !== '' && sessionFilters.value.status !== null && sessionFilters.value.status !== undefined) {
      params.status = sessionFilters.value.status
    }
    const res = await axios.get('/api/v1/terminal/sessions', { headers: authHeaders(), params })
    sessions.value = res.data?.data || []
    if (currentSession.value?.id) {
      currentSession.value = sessions.value.find(item => item.id === currentSession.value.id) || null
      if (!currentSession.value && terminalVisible.value) {
        terminalVisible.value = false
      }
    }
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '获取会话失败'))
  } finally {
    sessionLoading.value = false
  }
}

const fetchRecords = async () => {
  recordLoading.value = true
  try {
    const params = {}
    if (recordFilters.value.keyword.trim()) params.keyword = recordFilters.value.keyword.trim()
    const res = await axios.get('/api/v1/terminal/records', { headers: authHeaders(), params })
    records.value = res.data?.data || []
    selectedRecordIds.value = selectedRecordIds.value.filter(id => records.value.some(item => item.id === id))
    if (currentRecord.value?.id) {
      currentRecord.value = records.value.find(item => item.id === currentRecord.value.id) || null
      if (!currentRecord.value && recordVisible.value) {
        recordVisible.value = false
      }
    }
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '获取录像失败'))
  } finally {
    recordLoading.value = false
  }
}

const fetchAudits = async () => {
  auditLoading.value = true
  try {
    const params = {}
    if (auditFilters.value.keyword.trim()) params.keyword = auditFilters.value.keyword.trim()
    if (auditFilters.value.risk_level) params.risk_level = auditFilters.value.risk_level
    if (auditFilters.value.blocked) params.blocked = auditFilters.value.blocked
    const res = await axios.get('/api/v1/terminal/audits', { headers: authHeaders(), params })
    audits.value = res.data?.data || []
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '获取命令审计失败'))
  } finally {
    auditLoading.value = false
  }
}

const reloadAll = async () => {
  await Promise.all([fetchSessions(), fetchRecords(), fetchAudits()])
}

const resetSessionFilters = async () => {
  sessionFilters.value = { keyword: '', status: '' }
  await fetchSessions()
}

const resetRecordFilters = async () => {
  recordFilters.value = { keyword: '' }
  await fetchRecords()
}

const resetAuditFilters = async () => {
  auditFilters.value = { keyword: '', risk_level: '', blocked: '' }
  await fetchAudits()
}

const onRecordSelectionChange = (rows) => {
  selectedRecordIds.value = Array.isArray(rows) ? rows.map(item => item.id) : []
}

const resetCreateForm = () => {
  editingSessionId.value = ''
  createForm.value = { host_id: '', host: '', port: 22, username: '', password: '', key_auth: '' }
}

const onCreateDialogClosed = () => {
  resetCreateForm()
}

const openCreateDialog = () => {
  resetCreateForm()
  createVisible.value = true
}

const openEditDialog = (row) => {
  editingSessionId.value = row.id
  createForm.value = {
    host_id: row.host_id || row.host,
    host: row.host || '',
    port: row.port || 22,
    username: row.username || '',
    password: '',
    key_auth: ''
  }
  createVisible.value = true
}

const submitSession = async () => {
  if (!createForm.value.host.trim() || !createForm.value.username.trim()) {
    ElMessage.warning('请填写主机地址和用户名')
    return
  }
  if (!editingSessionId.value && !createForm.value.password && !createForm.value.key_auth) {
    ElMessage.warning('密码和私钥至少填写一个')
    return
  }

  const payload = {
    ...createForm.value,
    host_id: createForm.value.host_id || createForm.value.host
  }

  try {
    if (editingSessionId.value) {
      await axios.put(`/api/v1/terminal/sessions/${editingSessionId.value}`, payload, { headers: authHeaders() })
      ElMessage.success('会话已更新')
      createVisible.value = false
      editingSessionId.value = ''
      await fetchSessions()
      return
    }

    const res = await axios.post('/api/v1/terminal/sessions', payload, { headers: authHeaders() })
    const sess = res.data?.data
    ElMessage.success('会话创建成功')
    createVisible.value = false
    await fetchSessions()
    if (sess?.id) openTerminal(sess)
  } catch (err) {
    ElMessage.error(getErrorMessage(err, editingSessionId.value ? '编辑会话失败' : '创建会话失败'))
  }
}

const precheckConnection = async () => {
  if (!createForm.value.host.trim() || !createForm.value.username.trim()) {
    ElMessage.warning('请填写主机地址和用户名')
    return
  }
  if (!createForm.value.password && !createForm.value.key_auth) {
    ElMessage.warning('密码和私钥至少填写一个')
    return
  }

  precheckLoading.value = true
  try {
    const res = await axios.post('/api/v1/terminal/sessions/precheck', createForm.value, { headers: authHeaders() })
    ElMessage.success(res.data?.data?.message || res.data?.message || '连接测试通过')
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '连接测试失败'))
  } finally {
    precheckLoading.value = false
  }
}

const closeSession = async (row) => {
  try {
    await ElMessageBox.confirm(`确认关闭会话 ${row.host} 吗？`, '提示', { type: 'warning' })
    await axios.delete(`/api/v1/terminal/sessions/${row.id}`, { headers: authHeaders() })
    ElMessage.success('会话已关闭')
    await fetchSessions()
  } catch (err) {
    if (!isCancelError(err)) ElMessage.error(getErrorMessage(err, '关闭失败'))
  }
}

const deleteSession = async (row) => {
  try {
    await ElMessageBox.confirm(`确认删除会话 ${row.host} 吗？删除后不可恢复。`, '提示', { type: 'warning' })
    await axios.delete(`/api/v1/terminal/sessions/${row.id}/purge`, { headers: authHeaders() })
    ElMessage.success('会话已删除')
    if (currentSession.value?.id === row.id) {
      terminalVisible.value = false
    }
    await fetchSessions()
  } catch (err) {
    if (!isCancelError(err)) ElMessage.error(getErrorMessage(err, '删除失败'))
  }
}

const writeTerminal = (text) => {
  if (!term) return
  term.write(text)
}

const writelnTerminal = (text) => {
  if (!term) return
  term.writeln(text)
}

const initTerminal = async () => {
  await nextTick()
  if (!terminalRef.value) return

  if (terminalDataListener) {
    terminalDataListener.dispose()
    terminalDataListener = null
  }
  if (term) {
    term.dispose()
    term = null
  }

  term = new Terminal({
    cursorBlink: true,
    fontSize: 13,
    fontFamily: 'SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
    theme: {
      background: '#08152f',
      foreground: '#dbeafe',
      cursor: '#60a5fa',
      selectionBackground: 'rgba(96, 165, 250, 0.25)'
    },
    scrollback: 3000
  })
  fitAddon = new FitAddon()
  term.loadAddon(fitAddon)
  term.open(terminalRef.value)
  fitAddon.fit()
  terminalDataListener = term.onData((data) => {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(data)
    }
  })
}

const destroyTerminal = () => {
  if (terminalDataListener) {
    terminalDataListener.dispose()
    terminalDataListener = null
  }
  if (term) {
    term.dispose()
    term = null
  }
  fitAddon = null
}

const stopKeepAlive = () => {
  if (keepAliveTimer) {
    window.clearInterval(keepAliveTimer)
    keepAliveTimer = null
  }
}

const startKeepAlive = () => {
  stopKeepAlive()
  keepAliveTimer = window.setInterval(() => {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: 'ping', ts: Date.now() }))
    }
  }, 20000)
}

const clearReconnectTimer = () => {
  if (reconnectTimer) {
    window.clearTimeout(reconnectTimer)
    reconnectTimer = null
  }
}

const initRecordTerminal = async () => {
  await nextTick()
  if (!recordTerminalRef.value) return
  if (recordTerm) {
    recordTerm.dispose()
    recordTerm = null
  }
  recordTerm = new Terminal({
    cursorBlink: false,
    disableStdin: true,
    fontSize: 13,
    fontFamily: 'SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
    theme: {
      background: '#0b1220',
      foreground: '#e5e7eb',
      selectionBackground: 'rgba(148, 163, 184, 0.25)'
    },
    scrollback: 5000
  })
  recordFitAddon = new FitAddon()
  recordTerm.loadAddon(recordFitAddon)
  recordTerm.open(recordTerminalRef.value)
  recordFitAddon.fit()
}

const destroyRecordTerminal = () => {
  stopRecordPlayback()
  if (recordTerm) {
    recordTerm.dispose()
    recordTerm = null
  }
  recordFitAddon = null
}

const closeWebSocket = () => {
  stopKeepAlive()
  clearReconnectTimer()
  if (ws) {
    try {
      ws.close()
    } catch {
      // noop
    }
    ws = null
  }
}

const buildWsUrl = (sessionId) => {
  const token = localStorage.getItem('token') || ''
  const proto = window.location.protocol === 'https:' ? 'wss' : 'ws'
  return `${proto}://${window.location.host}/api/v1/terminal/ws/${sessionId}?token=${encodeURIComponent(token)}`
}

const scheduleReconnect = (row) => {
  if (!terminalVisible.value || manualTerminalClose || !row?.id) return
  if (reconnectAttempts >= 3) {
    terminalStatus.value = '已断开'
    writelnTerminal('\r\n[系统] 自动重连失败，已停止重试')
    return
  }
  reconnectAttempts += 1
  const delay = Math.min(5000, reconnectAttempts * 1500)
  terminalStatus.value = `重连中(${reconnectAttempts}/3)...`
  writelnTerminal(`\r\n[系统] ${delay / 1000}s 后尝试自动重连(${reconnectAttempts}/3)`)
  clearReconnectTimer()
  reconnectTimer = window.setTimeout(() => {
    connectTerminalSocket(row, true)
  }, delay)
}

const connectTerminalSocket = (row, isReconnect = false) => {
  clearReconnectTimer()
  const url = buildWsUrl(row.id)
  const currentGeneration = ++wsGeneration
  manualTerminalClose = false
  ws = new WebSocket(url)
  ws.binaryType = 'arraybuffer'

  ws.onopen = () => {
    if (currentGeneration !== wsGeneration) return
    terminalStatus.value = isReconnect ? '已重连' : '在线'
    reconnectAttempts = 0
    startKeepAlive()
    writelnTerminal(isReconnect ? `\r\n[系统] 已重新连接 ${row.host}:${row.port}` : `[系统] 已连接 ${row.host}:${row.port}`)
    sendResize()
  }

  ws.onmessage = (event) => {
    if (currentGeneration !== wsGeneration) return
    if (typeof event.data === 'string') {
      try {
        const control = JSON.parse(event.data)
        if (control?.type === 'pong') return
      } catch {
        // noop
      }
      writeTerminal(event.data)
      return
    }
    const decoder = new TextDecoder('utf-8')
    writeTerminal(decoder.decode(event.data))
  }

  ws.onerror = () => {
    if (currentGeneration !== wsGeneration) return
    terminalStatus.value = '连接异常'
    writelnTerminal('\r\n[系统] 连接异常：网络抖动或服务端中断')
  }

  ws.onclose = async (event) => {
    if (currentGeneration !== wsGeneration) return
    stopKeepAlive()
    await fetchSessions()
    const latest = sessions.value.find(item => item.id === row.id)
    const backendReason = latest?.last_error ? String(latest.last_error).trim() : ''
    const wsReason = parseWsCloseReason(event)
    const reason = backendReason || wsReason
    terminalStatus.value = reason ? '连接关闭' : '已断开'
    writelnTerminal(reason ? `\r\n[系统] 连接关闭：${reason}` : '\r\n[系统] 连接关闭')
    if (!manualTerminalClose && event.code !== 1000 && event.code !== 1001) {
      scheduleReconnect(row)
    }
  }
}

const openTerminal = async (row) => {
  closeWebSocket()
  currentSession.value = row
  terminalVisible.value = true
  terminalStatus.value = '连接中'
  reconnectAttempts = 0
  await initTerminal()
  connectTerminalSocket(row, false)
}

const openTerminalByID = async (id) => {
  if (!id) return
  try {
    const res = await axios.get(`/api/v1/terminal/sessions/${id}`, { headers: authHeaders() })
    if (res.data.code === 0 && res.data.data?.id) {
      openTerminal(res.data.data)
    }
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '读取会话失败'))
  }
}

const onTerminalDialogClosed = () => {
  manualTerminalClose = true
  terminalStatus.value = '未连接'
  closeWebSocket()
  destroyTerminal()
  currentSession.value = null
}

const sendMessage = (payload) => {
  if (!ws || ws.readyState !== WebSocket.OPEN) {
    ElMessage.warning('终端未连接')
    return false
  }
  ws.send(payload)
  return true
}

const sendCtrlC = () => {
  sendMessage('\u0003')
}

const sendResize = () => {
  if (!term || !fitAddon) return
  fitAddon.fit()
  sendMessage(JSON.stringify({ type: 'resize', cols: term.cols, rows: term.rows }))
}

const parseWsCloseReason = (event) => {
  if (!event) return ''
  const reason = String(event.reason || '').trim()
  if (reason) return reason
  switch (event.code) {
    case 1000:
    case 1001:
      return ''
    case 1006:
      return '连接中断：网络异常或服务端断开'
    case 1008:
      return '连接被策略拒绝'
    case 1011:
      return '服务端处理异常'
    default:
      return event.code ? `连接关闭(code=${event.code})` : ''
  }
}

const clearTerminal = () => {
  if (term) term.clear()
}

const handleWindowResize = () => {
  if (terminalVisible.value) sendResize()
  if (recordVisible.value && recordFitAddon) recordFitAddon.fit()
}

const viewRecord = async (row) => {
  try {
    const res = await axios.get(`/api/v1/terminal/records/${row.id}`, { headers: authHeaders() })
    const detail = res.data?.data
    currentRecord.value = detail
    let list = []
    try {
      list = JSON.parse(detail?.data || '[]')
    } catch {
      list = []
    }
    recordEvents.value = Array.isArray(list) ? list : []
    recordPlaybackIndex.value = 0
    recordVisible.value = true
    await initRecordTerminal()
    restartRecordPlayback()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '读取录像失败'))
  }
}

const renderReplayContent = (item) => {
  if (!recordTerm || !item) return
  const content = String(item.content || '')
  if (item.type === 'input') {
    const escaped = content
      .replace(/\u0003/g, '^C')
      .replace(/\r/g, '')
      .replace(/\n/g, '\r\n')
    recordTerm.write(`\x1b[38;5;81m${escaped}\x1b[0m`)
    return
  }
  recordTerm.write(content)
}

const stopRecordPlayback = () => {
  if (recordPlaybackTimer) {
    window.clearTimeout(recordPlaybackTimer)
    recordPlaybackTimer = null
  }
  recordPlaying.value = false
}

const scheduleNextRecordEvent = () => {
  if (!recordPlaying.value || !recordEvents.value.length) return
  if (recordPlaybackIndex.value >= recordEvents.value.length) {
    stopRecordPlayback()
    return
  }

  const current = recordEvents.value[recordPlaybackIndex.value]
  renderReplayContent(current)
  recordPlaybackIndex.value += 1

  if (recordPlaybackIndex.value >= recordEvents.value.length) {
    stopRecordPlayback()
    return
  }

  const next = recordEvents.value[recordPlaybackIndex.value]
  const currentTime = Number(current?.time || 0)
  const nextTime = Number(next?.time || 0)
  const delta = Math.max(10, nextTime - currentTime)
  recordPlaybackTimer = window.setTimeout(scheduleNextRecordEvent, Math.max(10, delta / recordPlaybackSpeed.value))
}

const toggleRecordPlayback = () => {
  if (!recordEvents.value.length) return
  if (recordPlaying.value) {
    stopRecordPlayback()
    return
  }
  recordPlaying.value = true
  scheduleNextRecordEvent()
}

const restartRecordPlayback = () => {
  stopRecordPlayback()
  recordPlaybackIndex.value = 0
  if (recordTerm) {
    recordTerm.reset()
  }
  if (!recordEvents.value.length) return
  recordPlaying.value = true
  scheduleNextRecordEvent()
}

const onRecordDialogClosed = () => {
  currentRecord.value = null
  recordEvents.value = []
  destroyRecordTerminal()
}

const onAuditDetailClosed = () => {
  currentAudit.value = null
}

const deleteRecord = async (row) => {
  try {
    await ElMessageBox.confirm(`确认删除 ${row.host} 的这条录像吗？`, '提示', { type: 'warning' })
    await axios.delete(`/api/v1/terminal/records/${row.id}`, { headers: authHeaders() })
    ElMessage.success('录像已删除')
    if (currentRecord.value?.id === row.id) {
      recordVisible.value = false
    }
    await fetchRecords()
  } catch (err) {
    if (!isCancelError(err)) ElMessage.error(getErrorMessage(err, '删除录像失败'))
  }
}

const downloadRecord = async (row) => {
  try {
    const res = await axios.get(`/api/v1/terminal/records/${row.id}/download`, {
      headers: authHeaders(),
      responseType: 'blob'
    })
    const blob = new Blob([res.data], { type: 'application/json;charset=utf-8' })
    const url = window.URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `terminal-record-${row.host || 'unknown'}-${row.id}.json`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    window.URL.revokeObjectURL(url)
    ElMessage.success('录像下载已开始')
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '下载录像失败'))
  }
}

const downloadRecordCast = async (row) => {
  try {
    const res = await axios.get(`/api/v1/terminal/records/${row.id}/asciinema`, {
      headers: authHeaders(),
      responseType: 'blob'
    })
    const blob = new Blob([res.data], { type: 'application/x-asciicast;charset=utf-8' })
    const url = window.URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `terminal-record-${row.host || 'unknown'}-${row.id}.cast`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    window.URL.revokeObjectURL(url)
    ElMessage.success('Cast 导出已开始')
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '导出 Cast 失败'))
  }
}

const exportSelectedRecords = async (format) => {
  if (!selectedRecordIds.value.length) {
    ElMessage.warning('请先选择录像')
    return
  }
  try {
    const res = await axios.post('/api/v1/terminal/records/export', {
      ids: selectedRecordIds.value,
      format
    }, {
      headers: authHeaders(),
      responseType: 'blob'
    })
    const blob = new Blob([res.data], { type: 'application/zip' })
    const url = window.URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `terminal-records-${format}-${Date.now()}.zip`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    window.URL.revokeObjectURL(url)
    ElMessage.success(`批量导出 ${selectedRecordIds.value.length} 条录像成功`)
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '批量导出失败'))
  }
}

const openAuditDetail = (row) => {
  currentAudit.value = row
  auditDetailVisible.value = true
}

const auditRiskTagType = (level) => {
  if (level === 'critical') return 'danger'
  if (level === 'warning') return 'warning'
  if (level === 'info') return 'info'
  return 'info'
}

const cleanupRecords = async (keepDays) => {
  try {
    await ElMessageBox.confirm(`确认清理 ${keepDays} 天前的历史录像吗？`, '提示', { type: 'warning' })
    const res = await axios.post('/api/v1/terminal/records/cleanup', { keep_days: keepDays }, { headers: authHeaders() })
    ElMessage.success(`清理完成，删除 ${res.data?.data?.deleted || 0} 条录像`)
    await fetchRecords()
  } catch (err) {
    if (!isCancelError(err)) ElMessage.error(getErrorMessage(err, '清理录像失败'))
  }
}

const openRecordBySession = (row) => {
  const found = records.value.find(item => item.session_id === row.id)
  if (!found) {
    ElMessage.info('该会话暂无录像')
    return
  }
  viewRecord(found)
}

onMounted(async () => {
  window.addEventListener('resize', handleWindowResize)
  await reloadAll()
  const sid = route.query.session_id
  if (typeof sid === 'string' && sid) {
    openTerminalByID(sid)
  }
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', handleWindowResize)
  closeWebSocket()
  destroyTerminal()
  destroyRecordTerminal()
})
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; gap: 12px; margin-bottom: 16px; }
.page-desc { margin: 4px 0 0; color: #606266; }
.page-actions { display: flex; gap: 8px; align-items: center; }
.section-card { margin-bottom: 12px; }
.card-title { font-weight: 600; }
.section-header { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.section-filters { display: flex; align-items: center; gap: 8px; margin-bottom: 12px; flex-wrap: wrap; }
.record-actions { display: flex; gap: 8px; }
.terminal-toolbar { display: flex; gap: 8px; margin-bottom: 10px; }
.terminal-status { margin-left: auto; color: #606266; font-size: 13px; align-self: center; }
.terminal-output { height: 480px; overflow: auto; background: #0f172a; color: #dbeafe; padding: 10px; border-radius: 6px; font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace; white-space: pre-wrap; line-height: 1.4; }
.record-meta { display: flex; gap: 16px; margin-bottom: 10px; color: #606266; }
.record-toolbar { display: flex; align-items: center; gap: 8px; margin-bottom: 10px; }
.record-progress { margin-left: auto; color: #606266; font-size: 13px; }
.record-output { height: 460px; overflow: auto; background: #111827; border-radius: 6px; padding: 10px; }
.audit-detail { display: flex; flex-direction: column; gap: 16px; }
.audit-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; }
.audit-grid > div { display: flex; flex-direction: column; gap: 4px; padding: 12px; border-radius: 10px; background: #f8fafc; }
.audit-label { color: #909399; font-size: 12px; }
.audit-block { border-radius: 10px; background: #0f172a; color: #e5e7eb; padding: 14px; }
.audit-block-title { margin-bottom: 8px; color: #93c5fd; font-weight: 600; }
.audit-block pre { margin: 0; white-space: pre-wrap; word-break: break-word; font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace; }
</style>
