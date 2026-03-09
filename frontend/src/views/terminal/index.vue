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
            <el-button size="small" @click="cleanupRecords(7)">清理 7 天前</el-button>
            <el-button size="small" @click="cleanupRecords(30)">清理 30 天前</el-button>
          </div>
        </div>
      </template>
      <el-table :fit="true" :data="records" v-loading="recordLoading" stripe>
        <el-table-column prop="operator" label="操作人" width="120" />
        <el-table-column prop="host" label="主机" min-width="180" />
        <el-table-column prop="duration" label="时长(s)" width="90" />
        <el-table-column prop="created_at" label="创建时间" width="170">
          <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="viewRecord(row)">回放</el-button>
            <el-button size="small" type="danger" plain @click="deleteRecord(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog append-to-body v-model="createVisible" :title="editingSessionId ? '编辑终端会话' : '新建终端会话'" width="680px">
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

const sessions = ref([])
const records = ref([])
const sessionLoading = ref(false)
const recordLoading = ref(false)

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
let ws = null
let term = null
let fitAddon = null
let terminalDataListener = null

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
    const res = await axios.get('/api/v1/terminal/sessions', { headers: authHeaders() })
    sessions.value = res.data?.data || []
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '获取会话失败')
  } finally {
    sessionLoading.value = false
  }
}

const fetchRecords = async () => {
  recordLoading.value = true
  try {
    const res = await axios.get('/api/v1/terminal/records', { headers: authHeaders() })
    records.value = res.data?.data || []
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '获取录像失败')
  } finally {
    recordLoading.value = false
  }
}

const reloadAll = async () => {
  await Promise.all([fetchSessions(), fetchRecords()])
}

const openCreateDialog = () => {
  editingSessionId.value = ''
  createForm.value = { host_id: '', host: '', port: 22, username: '', password: '', key_auth: '' }
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
    ElMessage.error(err.response?.data?.message || (editingSessionId.value ? '编辑会话失败' : '创建会话失败'))
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
    ElMessage.error(err.response?.data?.message || '连接测试失败')
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
    if (err !== 'cancel') ElMessage.error(err.response?.data?.message || '关闭失败')
  }
}

const deleteSession = async (row) => {
  try {
    await ElMessageBox.confirm(`确认删除会话 ${row.host} 吗？删除后不可恢复。`, '提示', { type: 'warning' })
    await axios.delete(`/api/v1/terminal/sessions/${row.id}/purge`, { headers: authHeaders() })
    ElMessage.success('会话已删除')
    await fetchSessions()
  } catch (err) {
    if (err !== 'cancel') ElMessage.error(err.response?.data?.message || '删除失败')
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

const openTerminal = async (row) => {
  closeWebSocket()
  currentSession.value = row
  terminalVisible.value = true
  await initTerminal()

  const url = buildWsUrl(row.id)
  ws = new WebSocket(url)
  ws.binaryType = 'arraybuffer'

  ws.onopen = () => {
    writelnTerminal(`[系统] 已连接 ${row.host}:${row.port}`)
    sendResize()
  }

  ws.onmessage = (event) => {
    if (typeof event.data === 'string') {
      writeTerminal(event.data)
      return
    }
    const decoder = new TextDecoder('utf-8')
    writeTerminal(decoder.decode(event.data))
  }

  ws.onerror = () => {
    writelnTerminal('\r\n[系统] 连接异常：网络抖动或服务端中断')
  }

  ws.onclose = async (event) => {
    await fetchSessions()
    const latest = sessions.value.find(item => item.id === row.id)
    const backendReason = latest?.last_error ? String(latest.last_error).trim() : ''
    const wsReason = parseWsCloseReason(event)
    const reason = backendReason || wsReason
    writelnTerminal(reason ? `\r\n[系统] 连接关闭：${reason}` : '\r\n[系统] 连接关闭')
  }
}

const openTerminalByID = async (id) => {
  if (!id) return
  try {
    const res = await axios.get(`/api/v1/terminal/sessions/${id}`, { headers: authHeaders() })
    if (res.data.code === 0 && res.data.data?.id) {
      openTerminal(res.data.data)
    }
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '读取会话失败')
  }
}

const onTerminalDialogClosed = () => {
  closeWebSocket()
  destroyTerminal()
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
    ElMessage.error(err.response?.data?.message || '读取录像失败')
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

const deleteRecord = async (row) => {
  try {
    await ElMessageBox.confirm(`确认删除 ${row.host} 的这条录像吗？`, '提示', { type: 'warning' })
    await axios.delete(`/api/v1/terminal/records/${row.id}`, { headers: authHeaders() })
    ElMessage.success('录像已删除')
    await fetchRecords()
  } catch (err) {
    if (err !== 'cancel') ElMessage.error(err.response?.data?.message || '删除录像失败')
  }
}

const cleanupRecords = async (keepDays) => {
  try {
    await ElMessageBox.confirm(`确认清理 ${keepDays} 天前的历史录像吗？`, '提示', { type: 'warning' })
    const res = await axios.post('/api/v1/terminal/records/cleanup', { keep_days: keepDays }, { headers: authHeaders() })
    ElMessage.success(`清理完成，删除 ${res.data?.data?.deleted || 0} 条录像`)
    await fetchRecords()
  } catch (err) {
    if (err !== 'cancel') ElMessage.error(err.response?.data?.message || '清理录像失败')
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
.record-actions { display: flex; gap: 8px; }
.terminal-toolbar { display: flex; gap: 8px; margin-bottom: 10px; }
.terminal-output { height: 480px; overflow: auto; background: #0f172a; color: #dbeafe; padding: 10px; border-radius: 6px; font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace; white-space: pre-wrap; line-height: 1.4; }
.record-meta { display: flex; gap: 16px; margin-bottom: 10px; color: #606266; }
.record-toolbar { display: flex; align-items: center; gap: 8px; margin-bottom: 10px; }
.record-progress { margin-left: auto; color: #606266; font-size: 13px; }
.record-output { height: 460px; overflow: auto; background: #111827; border-radius: 6px; padding: 10px; }
</style>
