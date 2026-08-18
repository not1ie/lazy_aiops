<template>
  <el-drawer
    v-model="visible"
    :size="isFullscreen ? '100%' : '80%'"
    :with-header="false"
    :destroy-on-close="false"
    :before-close="handleBeforeClose"
    class="web-terminal-drawer"
    direction="rtl"
  >
    <div class="terminal-container" :class="{ 'is-fullscreen': isFullscreen }">
      <!-- 终端头部工具栏 -->
      <div class="terminal-header">
        <div class="header-left">
          <div class="host-badge">
            <el-icon class="host-icon"><Monitor /></el-icon>
            <span class="host-name">{{ hostInfo?.name || hostInfo?.hostname || hostInfo?.ip || 'SSH Terminal' }}</span>
            <span class="host-ip">({{ hostInfo?.username || 'root' }}@{{ hostInfo?.ip || 'localhost' }}:{{ hostInfo?.port || 22 }})</span>
          </div>
          <el-tag :type="statusTagType" size="small" effect="dark" class="status-tag">
            <span class="status-dot" :class="statusClass"></span>
            {{ statusText }}
          </el-tag>
          <transition name="fade">
            <span v-if="copyTipVisible" class="copy-tip">
              <el-icon><Check /></el-icon> 已复制选中内容
            </span>
          </transition>
        </div>

        <div class="header-right">
          <!-- 快捷指令栏切换 -->
          <el-tooltip content="常用运维快捷指令" placement="bottom">
            <el-button 
              size="small" 
              :type="showSnippets ? 'primary' : 'default'" 
              plain 
              @click="showSnippets = !showSnippets"
            >
              <el-icon><Collection /></el-icon> 快捷指令
            </el-button>
          </el-tooltip>

          <!-- 清屏 -->
          <el-tooltip content="清空终端 (Ctrl+L)" placement="bottom">
            <el-button size="small" plain @click="clearTerminal">
              <el-icon><Delete /></el-icon> 清屏
            </el-button>
          </el-tooltip>

          <!-- 重新连接 -->
          <el-tooltip content="重新连接当前主机" placement="bottom">
            <el-button size="small" plain :loading="status === 'connecting'" @click="reconnect">
              <el-icon><RefreshRight /></el-icon> 重连
            </el-button>
          </el-tooltip>

          <!-- 全屏切换 -->
          <el-tooltip :content="isFullscreen ? '退出全屏' : '全屏模式'" placement="bottom">
            <el-button size="small" plain @click="toggleFullscreen">
              <el-icon v-if="!isFullscreen"><FullScreen /></el-icon>
              <el-icon v-else><Crop /></el-icon>
            </el-button>
          </el-tooltip>

          <!-- 关闭抽屉 -->
          <el-button size="small" type="danger" plain @click="closeDrawer">
            <el-icon><Close /></el-icon> 关闭
          </el-button>
        </div>
      </div>

      <!-- 快捷命令指令条 -->
      <transition name="el-zoom-in-top">
        <div v-if="showSnippets" class="snippets-bar">
          <span class="snippets-label">快捷执行:</span>
          <el-button 
            v-for="item in snippetList" 
            :key="item.cmd" 
            size="small" 
            class="snippet-btn"
            @click="sendSnippet(item.cmd)"
          >
            {{ item.label }}
          </el-button>
        </div>
      </transition>

      <!-- 终端渲染主体 -->
      <div 
        ref="xtermRef" 
        class="xterm-wrapper" 
        @contextmenu.prevent="handleContextMenu"
      ></div>

      <!-- 终端底部状态与提示信息 -->
      <div class="terminal-footer">
        <div class="footer-tips">
          <span>💡 提示：支持鼠标<strong>划选自动复制</strong>，<strong>Ctrl+C</strong>（无选中时中断，有选中时复制），<strong>Ctrl+V</strong> 直接粘贴</span>
        </div>
        <div class="footer-meta">
          <span>编码: UTF-8</span>
          <span class="meta-sep">|</span>
          <span>终端类型: xterm-256color</span>
        </div>
      </div>

      <!-- 右键上下文菜单 -->
      <div 
        v-if="contextMenuVisible" 
        class="custom-context-menu" 
        :style="{ left: contextMenuPos.x + 'px', top: contextMenuPos.y + 'px' }"
        @click.stop
      >
        <div class="menu-item" @click="handlePasteFromClipboard">
          <el-icon><DocumentCopy /></el-icon> 粘贴剪贴板内容 (Ctrl+V)
        </div>
        <div class="menu-item" @click="handleCopySelection">
          <el-icon><CopyDocument /></el-icon> 复制已选文字 (Ctrl+C)
        </div>
        <div class="menu-divider"></div>
        <div class="menu-item" @click="clearTerminal">
          <el-icon><Delete /></el-icon> 清空当前屏幕 (Ctrl+L)
        </div>
        <div class="menu-item" @click="toggleFullscreen">
          <el-icon><FullScreen /></el-icon> {{ isFullscreen ? '退出全屏模式' : '进入全屏模式' }}
        </div>
        <div class="menu-divider"></div>
        <div class="menu-item" @click="reconnect">
          <el-icon><RefreshRight /></el-icon> 重新建立连接
        </div>
      </div>
    </div>

    <!-- 凭据补充弹窗（若主机未绑定凭据） -->
    <el-dialog
      v-model="credentialDialogVisible"
      title="补充登录凭据"
      width="450px"
      append-to-body
      :close-on-click-modal="false"
    >
      <el-form :model="credentialForm" label-width="80px" size="default">
        <el-form-item label="登录用户">
          <el-input v-model="credentialForm.username" placeholder="默认 root" clearable />
        </el-form-item>
        <el-form-item label="认证方式">
          <el-radio-group v-model="authType">
            <el-radio-button label="password">密码认证</el-radio-button>
            <el-radio-button label="key">私钥认证</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="authType === 'password'" label="登录密码">
          <el-input 
            v-model="credentialForm.password" 
            type="password" 
            placeholder="请输入服务器 SSH 密码" 
            show-password 
            @keyup.enter="submitCredentialAndConnect"
          />
        </el-form-item>
        <el-form-item v-else label="SSH私钥">
          <el-input 
            v-model="credentialForm.key_auth" 
            type="textarea" 
            :rows="4" 
            placeholder="-----BEGIN RSA PRIVATE KEY-----..." 
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="credentialDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="connecting" @click="submitCredentialAndConnect">
          连接服务器
        </el-button>
      </template>
    </el-dialog>
  </el-drawer>
</template>

<script setup>
import { ref, computed, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { Terminal } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import 'xterm/css/xterm.css'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import {
  Monitor,
  Check,
  Collection,
  Delete,
  RefreshRight,
  FullScreen,
  Crop,
  Close,
  DocumentCopy,
  CopyDocument
} from '@element-plus/icons-vue'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  },
  hostInfo: {
    type: Object,
    default: () => ({})
  },
  sessionId: {
    type: String,
    default: ''
  }
})

const emit = defineEmits(['update:modelValue', 'close'])

const visible = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})

const xtermRef = ref(null)
const isFullscreen = ref(false)
const showSnippets = ref(false)
const status = ref('idle') // idle, connecting, online, disconnected, error
const copyTipVisible = ref(false)
let copyTipTimer = null

// 凭据补充表单
const credentialDialogVisible = ref(false)
const connecting = ref(false)
const authType = ref('password')
const credentialForm = ref({
  username: 'root',
  password: '',
  key_auth: ''
})

// 右键菜单
const contextMenuVisible = ref(false)
const contextMenuPos = ref({ x: 0, y: 0 })

// 终端实例与 WebSocket
let term = null
let fitAddon = null
let ws = null
let wsGeneration = 0
let resizeObserver = null
let pingTimer = null
let activeSessionId = ref('')

// 快捷运维指令列表
const snippetList = [
  { label: '系统资源 (top)', cmd: 'top\n' },
  { label: '磁盘用量 (df -h)', cmd: 'df -h\n' },
  { label: '内存状态 (free -m)', cmd: 'free -m\n' },
  { label: '网络端口 (netstat)', cmd: 'netstat -tulnp || ss -tulnp\n' },
  { label: '容器状态 (docker ps)', cmd: 'docker ps\n' },
  { label: 'IP 地址 (ip a)', cmd: 'ip a\n' },
  { label: '系统日志 (dmesg)', cmd: 'dmesg -T | tail -n 30\n' },
  { label: '运行时间 (uptime)', cmd: 'uptime\n' }
]

// 状态文字与样式映射
const statusText = computed(() => {
  switch (status.value) {
    case 'connecting': return '连接中...'
    case 'online': return '已连接'
    case 'reconnected': return '已重连'
    case 'disconnected': return '已断开'
    case 'error': return '连接异常'
    default: return '未连接'
  }
})

const statusTagType = computed(() => {
  switch (status.value) {
    case 'online':
    case 'reconnected': return 'success'
    case 'connecting': return 'warning'
    case 'disconnected': return 'info'
    case 'error': return 'danger'
    default: return 'info'
  }
})

const statusClass = computed(() => {
  switch (status.value) {
    case 'online':
    case 'reconnected': return 'is-online'
    case 'connecting': return 'is-connecting'
    case 'disconnected': return 'is-disconnected'
    case 'error': return 'is-error'
    default: return ''
  }
})

const authHeaders = () => {
  const token = localStorage.getItem('token') || ''
  return { Authorization: `Bearer ${token}` }
}

const triggerCopyTip = () => {
  copyTipVisible.value = true
  if (copyTipTimer) clearTimeout(copyTipTimer)
  copyTipTimer = setTimeout(() => {
    copyTipVisible.value = false
  }, 2000)
}

// 初始化 xterm 终端实例（针对 256 色高亮、UTF-8 中文、智能复制粘贴深度配置）
const initTerminal = async () => {
  await nextTick()
  if (!xtermRef.value) return

  if (term) {
    term.dispose()
    term = null
  }

  term = new Terminal({
    cursorBlink: true,
    cursorStyle: 'block',
    fontSize: 13.5,
    fontFamily: 'Consolas, "Fira Code", Monaco, Menlo, "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", "WenQuanYi Micro Hei", monospace',
    lineHeight: 1.22,
    letterSpacing: 0,
    allowProposedApi: true,
    scrollback: 10000,
    theme: {
      background: '#0d1117',
      foreground: '#e6edf3',
      cursor: '#58a6ff',
      cursorAccent: '#0d1117',
      selectionBackground: 'rgba(88, 166, 255, 0.35)',
      black: '#484f58',
      red: '#ff7b72',
      green: '#3fb950',
      yellow: '#d29922',
      blue: '#58a6ff',
      magenta: '#bc8cff',
      cyan: '#39c5cf',
      white: '#b1bac4',
      brightBlack: '#6e7681',
      brightRed: '#ffa198',
      brightGreen: '#56d364',
      brightYellow: '#e3b341',
      brightBlue: '#79c0ff',
      brightMagenta: '#d2a8ff',
      brightCyan: '#56d4dd',
      brightWhite: '#f0f6fc'
    }
  })

  fitAddon = new FitAddon()
  term.loadAddon(fitAddon)
  term.open(xtermRef.value)
  fitAddon.fit()

  // 1. 选中即复制 (Copy on Select)
  term.onSelectionChange(() => {
    const selection = term.getSelection()?.trim()
    if (selection) {
      navigator.clipboard?.writeText(selection).then(() => {
        triggerCopyTip()
      }).catch(() => {})
    }
  })

  // 2. 快捷键拦截适配 (Ctrl+C / Ctrl+V / Cmd+C / Cmd+V)
  term.attachCustomKeyEventHandler((e) => {
    // 处理 Ctrl+C / Cmd+C
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'c') {
      if (term.hasSelection()) {
        const text = term.getSelection()
        navigator.clipboard?.writeText(text).catch(() => {})
        triggerCopyTip()
        return false // 阻止向终端发送 SIGINT，完成复制
      }
      return true // 无选区，向终端发送 SIGINT (\x03)
    }

    // 处理 Ctrl+V / Cmd+V / Ctrl+Shift+V
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'v') {
      if (e.type === 'keydown') {
        navigator.clipboard?.readText().then(text => {
          if (text && ws && ws.readyState === WebSocket.OPEN) {
            ws.send(text)
          }
        }).catch(() => {})
      }
      return false
    }

    return true
  })

  // 3. 用户输入发送
  term.onData((data) => {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(data)
    }
  })

  // 4. 自适应尺寸监听
  if (!resizeObserver && xtermRef.value) {
    resizeObserver = new ResizeObserver(() => {
      if (fitAddon && term) {
        fitAddon.fit()
        sendResize()
      }
    })
    resizeObserver.observe(xtermRef.value)
  }
}

const sendResize = () => {
  if (term && ws && ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({
      type: 'resize',
      cols: term.cols,
      rows: term.rows
    }))
  }
}

const buildWsUrl = (sessionId) => {
  const token = localStorage.getItem('token') || ''
  const proto = window.location.protocol === 'https:' ? 'wss' : 'ws'
  return `${proto}://${window.location.host}/api/v1/terminal/ws/${sessionId}?token=${encodeURIComponent(token)}`
}

const connectWebSocket = (sessionId) => {
  closeWebSocket()
  status.value = 'connecting'
  const url = buildWsUrl(sessionId)
  const currentGen = ++wsGeneration

  ws = new WebSocket(url)
  ws.binaryType = 'arraybuffer'

  ws.onopen = () => {
    if (currentGen !== wsGeneration) return
    status.value = 'online'
    term?.writeln('\x1b[32m[系统] SSH 终端连接成功，已建立交互式会话。\x1b[0m\r\n')
    sendResize()

    // 保活心跳
    if (pingTimer) clearInterval(pingTimer)
    pingTimer = setInterval(() => {
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'ping', ts: Date.now() }))
      }
    }, 15000)
  }

  // 接收数据（原生二进制 UTF-8 字节流解码）
  ws.onmessage = (event) => {
    if (currentGen !== wsGeneration) return
    if (typeof event.data === 'string') {
      try {
        const msg = JSON.parse(event.data)
        if (msg?.type === 'pong') return
      } catch {}
      term?.write(event.data)
      return
    }
    // 关键：直接将 Uint8Array 传给 xterm 保证多字节 UTF-8 中文字符流式拼接不乱码
    term?.write(new Uint8Array(event.data))
  }

  ws.onerror = () => {
    if (currentGen !== wsGeneration) return
    status.value = 'error'
    term?.writeln('\r\n\x1b[31m[系统] WebSocket 连接发生异常，请检查网络或认证状态。\x1b[0m')
  }

  ws.onclose = (event) => {
    if (currentGen !== wsGeneration) return
    status.value = 'disconnected'
    if (pingTimer) clearInterval(pingTimer)
    term?.writeln(`\r\n\x1b[33m[系统] 会话连接已断开 (代码: ${event.code})。\x1b[0m`)
  }
}

const closeWebSocket = () => {
  if (pingTimer) clearInterval(pingTimer)
  if (ws) {
    try {
      ws.close()
    } catch {}
    ws = null
  }
}

// 快速创建会话并连接
const startSessionConnect = async (customCreds = null) => {
  if (!props.hostInfo?.id) {
    if (props.sessionId) {
      activeSessionId.value = props.sessionId
      await initTerminal()
      connectWebSocket(props.sessionId)
      return
    }
    return
  }

  status.value = 'connecting'
  await initTerminal()
  term?.writeln(`\x1b[36m[系统] 正在向服务器 ${props.hostInfo.name || props.hostInfo.ip} 发起 SSH 连接...\x1b[0m`)

  try {
    const payload = customCreds || {}
    const res = await axios.post(`/api/v1/terminal/quick-connect-host/${props.hostInfo.id}`, payload, {
      headers: authHeaders()
    })

    if (res.data?.code === 0 && res.data?.data?.id) {
      activeSessionId.value = res.data.data.id
      connectWebSocket(res.data.data.id)
    } else if (res.data?.code === 4001) {
      // 需要补充凭据
      credentialForm.value.username = res.data.data?.username || 'root'
      credentialDialogVisible.value = true
      status.value = 'idle'
      term?.writeln('\x1b[33m[系统] 该主机未绑定已保存凭据，请在弹出的对话框中输入密码或私钥。\x1b[0m')
    } else {
      status.value = 'error'
      term?.writeln(`\r\n\x1b[31m[系统] 连接失败: ${res.data?.message || '未知错误'}\x1b[0m`)
    }
  } catch (err) {
    status.value = 'error'
    const msg = err.response?.data?.message || err.message || '连接失败'
    term?.writeln(`\r\n\x1b[31m[系统] 初始化终端会话异常: ${msg}\x1b[0m`)
  }
}

const submitCredentialAndConnect = async () => {
  if (authType.value === 'password' && !credentialForm.value.password.trim()) {
    ElMessage.warning('请输入登录密码')
    return
  }
  if (authType.value === 'key' && !credentialForm.value.key_auth.trim()) {
    ElMessage.warning('请输入 SSH 私钥')
    return
  }

  connecting.value = true
  try {
    await startSessionConnect({
      username: credentialForm.value.username || 'root',
      password: authType.value === 'password' ? credentialForm.value.password : '',
      key_auth: authType.value === 'key' ? credentialForm.value.key_auth : ''
    })
    credentialDialogVisible.value = false
  } finally {
    connecting.value = false
  }
}

const clearTerminal = () => {
  term?.clear()
}

const reconnect = () => {
  if (activeSessionId.value) {
    connectWebSocket(activeSessionId.value)
  } else {
    startSessionConnect()
  }
}

const sendSnippet = (cmd) => {
  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(cmd)
    term?.focus()
  } else {
    ElMessage.warning('终端尚未连接')
  }
}

const toggleFullscreen = () => {
  isFullscreen.value = !isFullscreen.value
  nextTick(() => {
    if (fitAddon) {
      fitAddon.fit()
      sendResize()
    }
  })
}

const handleContextMenu = (e) => {
  contextMenuPos.value = { x: e.clientX, y: e.clientY }
  contextMenuVisible.value = true
}

const handlePasteFromClipboard = () => {
  contextMenuVisible.value = false
  navigator.clipboard?.readText().then(text => {
    if (text && ws && ws.readyState === WebSocket.OPEN) {
      ws.send(text)
    }
  }).catch(() => {
    ElMessage.warning('未能读取剪贴板，请使用快捷键 Ctrl+V / Cmd+V 粘贴')
  })
}

const handleCopySelection = () => {
  contextMenuVisible.value = false
  const text = term?.getSelection()
  if (text) {
    navigator.clipboard?.writeText(text).then(() => {
      triggerCopyTip()
    }).catch(() => {})
  } else {
    ElMessage.info('未选中文本')
  }
}

const closeContextMenu = () => {
  contextMenuVisible.value = false
}

const handleBeforeClose = (done) => {
  closeWebSocket()
  done()
}

const closeDrawer = () => {
  closeWebSocket()
  visible.value = false
  emit('close')
}

watch(() => props.modelValue, (val) => {
  if (val) {
    isFullscreen.value = false
    status.value = 'idle'
    nextTick(() => {
      startSessionConnect()
    })
  } else {
    closeWebSocket()
  }
})

onMounted(() => {
  window.addEventListener('click', closeContextMenu)
})

onBeforeUnmount(() => {
  window.removeEventListener('click', closeContextMenu)
  if (resizeObserver) resizeObserver.disconnect()
  closeWebSocket()
  if (term) term.dispose()
})
</script>

<style scoped>
.web-terminal-drawer :deep(.el-drawer__body) {
  padding: 0;
  background-color: #0d1117;
}

.terminal-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  background-color: #0d1117;
  color: #e6edf3;
  overflow: hidden;
  user-select: none;
}

.terminal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 16px;
  background: #161b22;
  border-bottom: 1px solid #30363d;
  flex-shrink: 0;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.host-badge {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 600;
  color: #f0f6fc;
}

.host-icon {
  font-size: 17px;
  color: #58a6ff;
}

.host-ip {
  font-size: 12px;
  color: #8b949e;
  font-weight: normal;
}

.status-tag {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
}

.status-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background-color: #8b949e;
}

.status-dot.is-online {
  background-color: #3fb950;
  box-shadow: 0 0 6px rgba(63, 185, 80, 0.8);
}

.status-dot.is-connecting {
  background-color: #d29922;
  animation: blink 1.2s infinite;
}

.status-dot.is-error {
  background-color: #f85149;
}

@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.3; }
}

.copy-tip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: #3fb950;
  background: rgba(63, 185, 80, 0.15);
  padding: 2px 8px;
  border-radius: 4px;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.snippets-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  background: #1c2128;
  border-bottom: 1px solid #30363d;
  flex-wrap: wrap;
}

.snippets-label {
  font-size: 12px;
  color: #8b949e;
  margin-right: 4px;
}

.snippet-btn {
  background: #21262d;
  border-color: #30363d;
  color: #c9d1d9;
}

.snippet-btn:hover {
  background: #30363d;
  border-color: #58a6ff;
  color: #58a6ff;
}

.xterm-wrapper {
  flex: 1;
  padding: 8px 12px;
  background-color: #0d1117;
  overflow: hidden;
  user-select: text;
}

.xterm-wrapper :deep(.xterm) {
  height: 100%;
}

.terminal-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 16px;
  background: #161b22;
  border-top: 1px solid #30363d;
  font-size: 11.5px;
  color: #8b949e;
  flex-shrink: 0;
}

.footer-meta {
  display: flex;
  align-items: center;
  gap: 8px;
}

.meta-sep {
  color: #30363d;
}

.custom-context-menu {
  position: fixed;
  z-index: 3000;
  background: #1c2128;
  border: 1px solid #30363d;
  border-radius: 6px;
  padding: 4px 0;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.45);
  min-width: 180px;
}

.menu-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 7px 14px;
  font-size: 12.5px;
  color: #c9d1d9;
  cursor: pointer;
  transition: all 0.15s;
}

.menu-item:hover {
  background: #30363d;
  color: #58a6ff;
}

.menu-divider {
  height: 1px;
  background: #30363d;
  margin: 4px 0;
}

.fade-enter-active, .fade-leave-active {
  transition: opacity 0.3s;
}
.fade-enter-from, .fade-leave-to {
  opacity: 0;
}
</style>
