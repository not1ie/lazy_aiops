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
      <el-table :data="sessions" v-loading="sessionLoading" stripe>
        <el-table-column prop="operator" label="操作人" width="120" />
        <el-table-column prop="host" label="主机" min-width="180" />
        <el-table-column prop="port" label="端口" width="90" />
        <el-table-column prop="username" label="登录用户" width="120" />
        <el-table-column prop="started_at" label="开始时间" width="170">
          <template #default="{ row }">{{ formatTime(row.started_at || row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'info'">{{ row.status === 1 ? '在线' : '待连' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="230" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="primary" @click="openTerminal(row)">连接</el-button>
            <el-button size="small" @click="openRecordBySession(row)">录像</el-button>
            <el-button size="small" type="danger" @click="closeSession(row)">关闭</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card shadow="never" class="section-card">
      <template #header>
        <div class="card-title">会话录像</div>
      </template>
      <el-table :data="records" v-loading="recordLoading" stripe>
        <el-table-column prop="operator" label="操作人" width="120" />
        <el-table-column prop="host" label="主机" min-width="180" />
        <el-table-column prop="duration" label="时长(s)" width="90" />
        <el-table-column prop="created_at" label="创建时间" width="170">
          <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="120" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="viewRecord(row)">回放</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="createVisible" title="新建终端会话" width="680px">
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
        <el-button type="primary" @click="submitCreate">创建并连接</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="terminalVisible" :title="`终端: ${currentSession?.host || ''}`" width="980px" top="4vh" @closed="onTerminalDialogClosed">
      <div class="terminal-toolbar">
        <el-input v-model="commandInput" placeholder="输入命令后回车发送" @keyup.enter="sendCommand" clearable />
        <el-button type="primary" @click="sendCommand">发送</el-button>
        <el-button @click="sendCtrlC">Ctrl+C</el-button>
        <el-button @click="sendResize">同步窗口</el-button>
      </div>
      <div ref="terminalOutputRef" class="terminal-output">{{ terminalOutput }}</div>
      <template #footer>
        <el-button @click="terminalVisible = false">关闭终端窗口</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="recordVisible" title="录像回放" width="980px" top="5vh">
      <div class="record-meta">
        <span>主机: {{ currentRecord?.host || '-' }}</span>
        <span>操作人: {{ currentRecord?.operator || '-' }}</span>
        <span>时长: {{ currentRecord?.duration || 0 }}s</span>
      </div>
      <div class="record-output">{{ recordOutput }}</div>
      <template #footer>
        <el-button @click="recordVisible = false">关闭</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const sessions = ref([])
const records = ref([])
const sessionLoading = ref(false)
const recordLoading = ref(false)

const createVisible = ref(false)
const createForm = ref({
  host_id: '',
  host: '',
  port: 22,
  username: '',
  password: '',
  key_auth: ''
})

const terminalVisible = ref(false)
const terminalOutputRef = ref(null)
const currentSession = ref(null)
const commandInput = ref('')
const terminalOutput = ref('')
let ws = null

const recordVisible = ref(false)
const currentRecord = ref(null)
const recordOutput = ref('')

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const formatTime = (v) => {
  if (!v) return '-'
  return String(v).slice(0, 19).replace('T', ' ')
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
  createForm.value = { host_id: '', host: '', port: 22, username: '', password: '', key_auth: '' }
  createVisible.value = true
}

const submitCreate = async () => {
  if (!createForm.value.host.trim() || !createForm.value.username.trim()) {
    ElMessage.warning('请填写主机地址和用户名')
    return
  }
  if (!createForm.value.password && !createForm.value.key_auth) {
    ElMessage.warning('密码和私钥至少填写一个')
    return
  }

  const payload = {
    ...createForm.value,
    host_id: createForm.value.host_id || createForm.value.host
  }

  try {
    const res = await axios.post('/api/v1/terminal/sessions', payload, { headers: authHeaders() })
    const sess = res.data?.data
    ElMessage.success('会话创建成功')
    createVisible.value = false
    await fetchSessions()
    if (sess?.id) openTerminal(sess)
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '创建会话失败')
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

const scrollTerminalToBottom = () => {
  nextTick(() => {
    if (terminalOutputRef.value) {
      terminalOutputRef.value.scrollTop = terminalOutputRef.value.scrollHeight
    }
  })
}

const appendOutput = (text) => {
  terminalOutput.value += text
  if (terminalOutput.value.length > 200000) {
    terminalOutput.value = terminalOutput.value.slice(-120000)
  }
  scrollTerminalToBottom()
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

const openTerminal = (row) => {
  closeWebSocket()
  currentSession.value = row
  commandInput.value = ''
  terminalOutput.value = ''
  terminalVisible.value = true

  const url = buildWsUrl(row.id)
  ws = new WebSocket(url)
  ws.binaryType = 'arraybuffer'

  ws.onopen = () => {
    appendOutput(`\n[系统] 已连接 ${row.host}:${row.port}\n`)
    sendResize()
  }

  ws.onmessage = (event) => {
    if (typeof event.data === 'string') {
      appendOutput(event.data)
      return
    }
    const decoder = new TextDecoder('utf-8')
    appendOutput(decoder.decode(event.data))
  }

  ws.onerror = () => {
    appendOutput('\n[系统] 连接异常\n')
  }

  ws.onclose = () => {
    appendOutput('\n[系统] 连接关闭\n')
  }
}

const onTerminalDialogClosed = () => {
  closeWebSocket()
}

const sendMessage = (payload) => {
  if (!ws || ws.readyState !== WebSocket.OPEN) {
    ElMessage.warning('终端未连接')
    return false
  }
  ws.send(payload)
  return true
}

const sendCommand = () => {
  const cmd = commandInput.value.trim()
  if (!cmd) return
  if (sendMessage(`${cmd}\n`)) {
    commandInput.value = ''
  }
}

const sendCtrlC = () => {
  sendMessage('\u0003')
}

const sendResize = () => {
  const cols = Math.max(80, Math.floor((window.innerWidth - 240) / 8))
  const rows = 40
  sendMessage(JSON.stringify({ type: 'resize', cols, rows }))
}

const viewRecord = async (row) => {
  try {
    const res = await axios.get(`/api/v1/terminal/records/${row.id}`, { headers: authHeaders() })
    const detail = res.data?.data
    currentRecord.value = detail
    recordOutput.value = ''
    let list = []
    try {
      list = JSON.parse(detail?.data || '[]')
    } catch {
      list = []
    }
    recordOutput.value = list.map(item => `${item.type === 'input' ? '> ' : ''}${item.content || ''}`).join('')
    recordVisible.value = true
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '读取录像失败')
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

onMounted(reloadAll)

onBeforeUnmount(() => {
  closeWebSocket()
})
</script>

<style scoped>
.page-card { max-width: 1280px; margin: 0 auto; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; gap: 12px; margin-bottom: 16px; }
.page-desc { margin: 4px 0 0; color: #606266; }
.page-actions { display: flex; gap: 8px; align-items: center; }
.section-card { margin-bottom: 12px; }
.card-title { font-weight: 600; }
.terminal-toolbar { display: flex; gap: 8px; margin-bottom: 10px; }
.terminal-output { height: 480px; overflow: auto; background: #0f172a; color: #dbeafe; padding: 10px; border-radius: 6px; font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace; white-space: pre-wrap; line-height: 1.4; }
.record-meta { display: flex; gap: 16px; margin-bottom: 10px; color: #606266; }
.record-output { height: 460px; overflow: auto; background: #111827; color: #e5e7eb; padding: 10px; border-radius: 6px; white-space: pre-wrap; font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace; }
</style>
