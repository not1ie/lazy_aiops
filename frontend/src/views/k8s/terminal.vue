<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>K8s WebShell</h2>
        <p class="page-desc">连接到 Pod/容器并执行命令。</p>
      </div>
      <div class="page-actions">
        <el-button type="primary" :disabled="!canConnect" @click="toggleConnection">
          {{ connected ? '断开' : '连接' }}
        </el-button>
        <el-button icon="Refresh" @click="refreshPods">刷新</el-button>
      </div>
    </div>

    <el-form label-width="110px" class="form-block">
      <el-form-item label="集群">
        <el-select v-model="clusterId" placeholder="选择集群" class="w-52" @change="handleClusterChange">
          <el-option v-for="c in clusters" :key="c.id" :label="c.display_name || c.name" :value="c.id" />
        </el-select>
      </el-form-item>
      <el-form-item label="命名空间">
        <el-select v-model="namespace" placeholder="命名空间" class="w-52" @change="fetchPods">
          <el-option v-for="ns in namespaces" :key="ns.name" :label="ns.name" :value="ns.name" />
        </el-select>
      </el-form-item>
      <el-form-item label="Pod">
        <el-select v-model="podName" placeholder="Pod" class="w-52" @change="handlePodChange">
          <el-option v-for="p in pods" :key="p.name" :label="p.name" :value="p.name" />
        </el-select>
      </el-form-item>
      <el-form-item label="容器">
        <el-select v-model="container" placeholder="容器" class="w-52">
          <el-option v-for="c in containers" :key="c" :label="c" :value="c" />
        </el-select>
      </el-form-item>
      <el-form-item label="Shell">
        <el-select v-model="shell" placeholder="Shell" class="w-52">
          <el-option label="/bin/sh" value="sh" />
          <el-option label="/bin/bash" value="bash" />
          <el-option label="/bin/ash" value="ash" />
        </el-select>
      </el-form-item>
    </el-form>

    <el-divider />

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

const clusters = ref([])
const namespaces = ref([])
const pods = ref([])
const containers = ref([])

const clusterId = ref('')
const namespace = ref('')
const podName = ref('')
const container = ref('')
const shell = ref('sh')

const connected = ref(false)
const terminalRef = ref(null)
let term = null
let fitAddon = null
let dataListener = null
let ws = null
const route = useRoute()

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const canConnect = computed(() => {
  return clusterId.value && namespace.value && podName.value && container.value && shell.value
})

const fetchClusters = async () => {
  const res = await axios.get('/api/v1/k8s/clusters', { headers: authHeaders() })
  clusters.value = res.data.data || []
  if (!clusterId.value && clusters.value.length > 0) {
    clusterId.value = clusters.value[0].id
  }
}

const fetchNamespaces = async () => {
  if (!clusterId.value) return
  const res = await axios.get(`/api/v1/k8s/clusters/${clusterId.value}/namespaces`, { headers: authHeaders() })
  namespaces.value = res.data.data || []
  if (!namespace.value && namespaces.value.length > 0) {
    namespace.value = namespaces.value[0].name
  }
}

const fetchPods = async () => {
  if (!clusterId.value || !namespace.value) return
  const res = await axios.get(`/api/v1/k8s/clusters/${clusterId.value}/namespaces/${namespace.value}/pods`, {
    headers: authHeaders()
  })
  pods.value = res.data.data || []
  if (!podName.value && pods.value.length > 0) {
    podName.value = pods.value[0].name
  }
  handlePodChange()
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

const connect = () => {
  if (!canConnect.value) return
  const token = localStorage.getItem('token') || ''
  if (!token) {
    ElMessage.error('登录状态失效，请重新登录')
    return
  }
  const wsProto = window.location.protocol === 'https:' ? 'wss' : 'ws'
  const cmd = shell.value ? `/bin/${shell.value}` : ''
  const wsUrl = `${wsProto}://${window.location.host}/api/v1/k8s/clusters/${encodeURIComponent(clusterId.value)}/namespaces/${encodeURIComponent(namespace.value)}/pods/${encodeURIComponent(podName.value)}/exec?container=${encodeURIComponent(container.value)}&token=${encodeURIComponent(token)}&command=${encodeURIComponent(cmd)}`

  ws = new WebSocket(wsUrl)
  ws.binaryType = 'arraybuffer'
  ws.onopen = () => {
    connected.value = true
    term?.writeln('连接成功。')
    sendResize()
  }
  ws.onmessage = (evt) => {
    if (evt.data instanceof ArrayBuffer) {
      const text = new TextDecoder().decode(new Uint8Array(evt.data))
      term?.write(text)
      return
    }
    term?.write(evt.data)
  }
  ws.onclose = (evt) => {
    connected.value = false
    const reason = evt?.reason ? ` (${evt.reason})` : ''
    term?.writeln(`\r\n连接已关闭 [${evt?.code ?? '-'}]${reason}。`)
  }
  ws.onerror = (evt) => {
    console.error('[K8s WebShell] websocket error', evt)
    ElMessage.error('连接失败')
  }
}

const disconnect = () => {
  if (ws) {
    ws.close()
    ws = null
  }
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

onMounted(async () => {
  clusterId.value = route.query.clusterId || ''
  namespace.value = route.query.namespace || ''
  podName.value = route.query.pod || ''
  container.value = route.query.container || ''
  shell.value = route.query.shell || 'sh'

  term = new Terminal({
    cursorBlink: true,
    fontSize: 13,
    theme: {
      background: '#0f172a',
      foreground: '#e2e8f0'
    }
  })
  fitAddon = new FitAddon()
  term.loadAddon(fitAddon)
  if (terminalRef.value) {
    term.open(terminalRef.value)
    fitAddon.fit()
  }
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
.page-card { max-width: 1180px; margin: 0 auto; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 16px; }
.page-desc { color: #606266; margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; }
.form-block { max-width: 520px; }
.w-52 { width: 220px; }
.terminal-shell { margin-top: 16px; }
.terminal-container {
  height: 360px;
  background: #0f172a;
  border-radius: 8px;
  overflow: hidden;
  padding: 8px;
}
</style>
