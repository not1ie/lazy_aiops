<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>K8s 容器 WebShell</h2>
        <p class="page-desc">一键直连 Kubernetes Pod 容器内部环境，执行命令与实时排障。</p>
      </div>
      <div class="page-actions">
        <el-button :type="connected ? 'danger' : 'primary'" :disabled="!canConnect" @click="toggleConnection">
          {{ connected ? '断开连接' : '建立连接' }}
        </el-button>
        <el-button icon="Refresh" @click="refreshPods">刷新集群</el-button>
      </div>
    </div>

    <!-- 顶部快速选择栏 -->
    <div class="target-bar">
      <div class="target-item">
        <span class="label">集群:</span>
        <el-select v-model="clusterId" placeholder="选择集群" class="w-44" size="small" @change="handleClusterChange">
          <el-option v-for="c in clusters" :key="c.id" :label="c.display_name || c.name" :value="c.id" />
        </el-select>
      </div>
      <div class="target-item">
        <span class="label">命名空间:</span>
        <el-select v-model="namespace" placeholder="命名空间" class="w-44" size="small" @change="fetchPods">
          <el-option v-for="ns in namespaces" :key="ns.name" :label="ns.name" :value="ns.name" />
        </el-select>
      </div>
      <div class="target-item">
        <span class="label">Pod:</span>
        <el-select v-model="podName" placeholder="Pod" class="w-52" size="small" filterable @change="handlePodChange">
          <el-option v-for="p in pods" :key="p.name" :label="p.name" :value="p.name" />
        </el-select>
      </div>
      <div class="target-item">
        <span class="label">容器:</span>
        <el-select v-model="container" placeholder="容器" class="w-40" size="small">
          <el-option v-for="c in containers" :key="c" :label="c" :value="c" />
        </el-select>
      </div>
      <div class="target-item">
        <span class="label">Shell:</span>
        <el-select v-model="shell" placeholder="Shell" class="w-32" size="small">
          <el-option label="/bin/bash" value="bash" />
          <el-option label="/bin/sh" value="sh" />
          <el-option label="/bin/ash" value="ash" />
        </el-select>
      </div>
      <div class="target-item">
        <el-tag :type="connected ? 'success' : 'info'" size="small" effect="dark">
          {{ connected ? '已连接' : '未连接' }}
        </el-tag>
      </div>
    </div>

    <!-- 常用容器排障快捷指令 -->
    <div class="snippets-bar">
      <span class="snippets-label">容器常用指令:</span>
      <el-button 
        v-for="item in snippetList" 
        :key="item.cmd" 
        size="small" 
        class="snippet-btn"
        @click="sendSnippet(item.cmd)"
      >
        {{ item.label }}
      </el-button>
      <el-button size="small" plain @click="clearTerminal">清屏</el-button>
    </div>

    <!-- WebShell 终端渲染容器 -->
    <div class="terminal-shell">
      <div ref="terminalRef" class="terminal-container"></div>
    </div>
  </el-card>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount, computed } from 'vue'
import { useRoute } from 'vue-router'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import { Terminal } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import 'xterm/css/xterm.css'
import { getErrorMessage } from '@/utils/error'

const clusters = ref([])
const namespaces = ref([])
const pods = ref([])
const containers = ref([])

const clusterId = ref('')
const namespace = ref('')
const podName = ref('')
const container = ref('')
const shell = ref('bash')

const connected = ref(false)
const terminalRef = ref(null)
let term = null
let fitAddon = null
let dataListener = null
let ws = null
let pingTimer = null
const route = useRoute()

const snippetList = [
  { label: '环境变量 (env)', cmd: 'env\n' },
  { label: '进程列表 (ps aux)', cmd: 'ps aux || ps -ef\n' },
  { label: '磁盘空间 (df -h)', cmd: 'df -h\n' },
  { label: 'DNS 配置 (resolv.conf)', cmd: 'cat /etc/resolv.conf\n' },
  { label: '网络连接 (netstat)', cmd: 'netstat -tulnp || ss -tulnp || cat /proc/net/tcp\n' },
  { label: '目录清单 (ls -la)', cmd: 'ls -la\n' }
]

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const canConnect = computed(() => {
  return clusterId.value && namespace.value && podName.value && container.value && shell.value
})

const fetchClusters = async () => {
  try {
    const res = await axios.get('/api/v1/k8s/clusters', { headers: authHeaders() })
    clusters.value = res.data.data || []
    if (!clusterId.value && clusters.value.length > 0) {
      clusterId.value = clusters.value[0].id
    }
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '获取集群失败'))
  }
}

const fetchNamespaces = async () => {
  if (!clusterId.value) return
  try {
    const res = await axios.get(`/api/v1/k8s/clusters/${clusterId.value}/namespaces`, { headers: authHeaders() })
    namespaces.value = res.data.data || []
    if (!namespace.value && namespaces.value.length > 0) {
      namespace.value = namespaces.value[0].name
    }
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '获取命名空间失败'))
  }
}

const fetchPods = async () => {
  if (!clusterId.value || !namespace.value) return
  try {
    const res = await axios.get(`/api/v1/k8s/clusters/${clusterId.value}/namespaces/${namespace.value}/pods`, {
      headers: authHeaders()
    })
    pods.value = res.data.data || []
    if (!podName.value && pods.value.length > 0) {
      podName.value = pods.value[0].name
    }
    handlePodChange()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '获取 Pod 列表失败'))
  }
}

const handlePodChange = () => {
  const pod = pods.value.find(p => p.name === podName.value)
  containers.value = pod?.containers?.map(c => c.name) || []
  if (!container.value && containers.value.length > 0) {
    container.value = containers.value[0]
  }
}

const refreshPods = async () => {
  await fetchPods()
}

const handleClusterChange = async () => {
  await fetchNamespaces()
  await fetchPods()
}

const connect = async () => {
  if (!canConnect.value) return
  const token = localStorage.getItem('token') || ''
  if (!token) {
    ElMessage.error('登录状态失效，请重新登录')
    return
  }
  const cmd = shell.value ? `/bin/${shell.value}` : '/bin/sh'
  const basePath = `/api/v1/k8s/clusters/${encodeURIComponent(clusterId.value)}/namespaces/${encodeURIComponent(namespace.value)}/pods/${encodeURIComponent(podName.value)}/exec`
  try {
    await axios.get(basePath, {
      headers: authHeaders(),
      params: {
        container: container.value,
        command: cmd,
        dry_run: 1
      }
    })
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '终端预检查失败'))
    return
  }

  const wsProto = window.location.protocol === 'https:' ? 'wss' : 'ws'
  const wsUrl = `${wsProto}://${window.location.host}${basePath}?container=${encodeURIComponent(container.value)}&token=${encodeURIComponent(token)}&command=${encodeURIComponent(cmd)}`

  ws = new WebSocket(wsUrl)
  ws.binaryType = 'arraybuffer'
  ws.onopen = () => {
    connected.value = true
    term?.writeln('\x1b[32m[系统] K8s 容器 WebShell 连接成功，已就绪。\x1b[0m\r\n')
    sendResize()

    if (pingTimer) clearInterval(pingTimer)
    pingTimer = setInterval(() => {
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'ping' }))
      }
    }, 15000)
  }
  ws.onmessage = (evt) => {
    if (evt.data instanceof ArrayBuffer) {
      term?.write(new Uint8Array(evt.data))
      return
    }
    term?.write(evt.data)
  }
  ws.onclose = (evt) => {
    connected.value = false
    if (pingTimer) clearInterval(pingTimer)
    const reason = evt?.reason ? ` (${evt.reason})` : ''
    term?.writeln(`\r\n\x1b[33m[系统] 容器连接已断开 [${evt?.code ?? '-'}]${reason}。\x1b[0m\r\n`)
  }
  ws.onerror = (evt) => {
    console.error('[K8s WebShell] websocket error', evt)
    ElMessage.error('WebSocket连接失败，请检查集群权限/网络后重试')
  }
}

const disconnect = () => {
  if (pingTimer) clearInterval(pingTimer)
  if (ws) {
    ws.close()
    ws = null
  }
  connected.value = false
}

const toggleConnection = () => {
  if (connected.value) disconnect()
  else connect()
}

const sendResize = () => {
  if (!ws || ws.readyState !== WebSocket.OPEN || !term) return
  ws.send(JSON.stringify({ type: 'resize', cols: term.cols, rows: term.rows }))
}

const handleResize = () => {
  if (!fitAddon || !term) return
  fitAddon.fit()
  sendResize()
}

const sendSnippet = (cmd) => {
  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(cmd)
    term?.focus()
  } else {
    ElMessage.warning('容器 WebShell 尚未连接')
  }
}

const clearTerminal = () => {
  term?.clear()
}

onMounted(async () => {
  clusterId.value = route.query.clusterId || ''
  namespace.value = route.query.namespace || ''
  podName.value = route.query.pod || route.query.name || ''
  container.value = route.query.container || ''
  shell.value = route.query.shell || 'bash'

  term = new Terminal({
    cursorBlink: true,
    fontSize: 13.5,
    fontFamily: 'Consolas, "Fira Code", Monaco, Menlo, monospace',
    lineHeight: 1.25,
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
      white: '#b1bac4'
    }
  })
  fitAddon = new FitAddon()
  term.loadAddon(fitAddon)
  if (terminalRef.value) {
    term.open(terminalRef.value)
    fitAddon.fit()
  }

  // 选中即复制
  term.onSelectionChange(() => {
    const sel = term.getSelection()?.trim()
    if (sel) {
      navigator.clipboard?.writeText(sel).catch(() => {})
    }
  })

  // 快捷键拦截
  term.attachCustomKeyEventHandler((e) => {
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'c') {
      if (term.hasSelection()) {
        navigator.clipboard?.writeText(term.getSelection()).catch(() => {})
        return false
      }
      return true
    }
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

  dataListener = term.onData((data) => {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(data)
    }
  })
  window.addEventListener('resize', handleResize)

  await fetchClusters()
  await fetchNamespaces()
  await fetchPods()
  if (podName.value) {
    handlePodChange()
    // 自动连接
    if (canConnect.value) {
      connect()
    }
  }
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', handleResize)
  if (dataListener) dataListener.dispose()
  if (term) term.dispose()
  disconnect()
})
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 14px; }
.page-desc { color: #606266; margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; }

.target-bar {
  display: flex;
  align-items: center;
  gap: 14px;
  background: #f8fafc;
  padding: 10px 14px;
  border-radius: 6px;
  border: 1px solid #e2e8f0;
  flex-wrap: wrap;
  margin-bottom: 10px;
}
.target-item {
  display: flex;
  align-items: center;
  gap: 6px;
}
.target-item .label {
  font-size: 12.5px;
  font-weight: 600;
  color: #475569;
}
.w-32 { width: 120px; }
.w-40 { width: 150px; }
.w-44 { width: 170px; }
.w-52 { width: 220px; }

.snippets-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: #1e293b;
  border-radius: 6px 6px 0 0;
  border-bottom: 1px solid #334155;
  flex-wrap: wrap;
}
.snippets-label {
  font-size: 12px;
  color: #94a3b8;
}
.snippet-btn {
  background: #334155;
  border-color: #475569;
  color: #f1f5f9;
}
.snippet-btn:hover {
  background: #475569;
  color: #38bdf8;
}

.terminal-shell {
  margin-top: 0;
}
.terminal-container {
  height: 560px;
  background: #0d1117;
  border-radius: 0 0 8px 8px;
  overflow: hidden;
  padding: 8px 12px;
}
</style>
