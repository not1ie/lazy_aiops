<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>Pod 详情</h2>
        <p class="page-desc">查看 Pod 状态、容器信息、日志与事件。</p>
      </div>
      <div class="page-actions">
        <el-button icon="Refresh" @click="reloadAll">刷新</el-button>
        <el-button @click="restartPod">重启 Pod</el-button>
        <el-button type="primary" @click="restartWorkload">滚动重启工作负载</el-button>
        <el-button type="danger" @click="deletePod">删除 Pod</el-button>
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
        <el-select v-model="podName" placeholder="Pod" class="w-52" @change="fetchPodDetail">
          <el-option v-for="p in pods" :key="p.name" :label="p.name" :value="p.name" />
        </el-select>
      </el-form-item>
    </el-form>

    <el-tabs v-model="activeTab">
      <el-tab-pane label="概览" name="overview">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="名称">{{ pod.name }}</el-descriptions-item>
          <el-descriptions-item label="命名空间">{{ pod.namespace }}</el-descriptions-item>
          <el-descriptions-item label="状态">{{ pod.status }}</el-descriptions-item>
          <el-descriptions-item label="节点">{{ pod.node }}</el-descriptions-item>
          <el-descriptions-item label="IP">{{ pod.ip }}</el-descriptions-item>
          <el-descriptions-item label="重启次数">{{ pod.restarts }}</el-descriptions-item>
          <el-descriptions-item label="控制器">{{ pod.owner_kind ? `${pod.owner_kind}/${pod.owner_name}` : '-' }}</el-descriptions-item>
          <el-descriptions-item label="创建时间">{{ pod.created_at }}</el-descriptions-item>
        </el-descriptions>

        <el-divider />

        <h3 class="section-title">容器</h3>
        <el-table :fit="false" :data="pod.containers || []" stripe style="width: 100%">
          <el-table-column prop="name" label="容器" min-width="180" />
          <el-table-column prop="image" label="镜像" min-width="220" />
          <el-table-column prop="state" label="状态" width="140" />
          <el-table-column prop="ready" label="就绪" width="100">
            <template #default="scope">
              <el-tag :type="scope.row.ready ? 'success' : 'warning'">
                {{ scope.row.ready ? 'Ready' : 'NotReady' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="160">
            <template #default="scope">
              <el-button size="small" type="primary" @click="openTerminal(scope.row)">WebShell</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="日志" name="logs">
        <div class="log-controls">
          <el-select v-model="logContainer" placeholder="容器" class="w-52">
            <el-option v-for="c in logContainers" :key="c" :label="c" :value="c" />
          </el-select>
          <el-input-number v-model="logTail" :min="10" :max="2000" />
          <el-input v-model="logFilter" placeholder="过滤关键词(逗号/正则:/re/，排除:-error)" class="w-52" />
          <el-button type="primary" @click="fetchLogs">获取日志</el-button>
          <el-switch v-model="logStreaming" active-text="实时" @change="toggleStream" />
          <el-button @click="togglePause" :disabled="!logStreaming">{{ logPaused ? '继续' : '暂停' }}</el-button>
          <el-button @click="logText = ''">清空</el-button>
        </div>
        <el-alert v-if="logPaused" type="warning" show-icon title="日志已暂停" description="实时连接仍保持，但不追加日志。" />
        <el-input ref="logInputRef" v-model="logText" type="textarea" :rows="18" readonly />
        <div class="log-highlight" v-html="highlightedLog" @click="jumpToMatch"></div>
      </el-tab-pane>

      <el-tab-pane label="事件" name="events">
        <div class="event-controls">
          <el-select v-model="eventType" placeholder="类型" class="w-52" clearable>
            <el-option label="Normal" value="Normal" />
            <el-option label="Warning" value="Warning" />
          </el-select>
          <el-input v-model="eventKeyword" placeholder="关键词 (reason/message)" class="w-52" clearable />
          <el-button icon="Refresh" @click="fetchEvents">刷新</el-button>
        </div>
        <el-table :fit="false" :data="events" stripe style="width: 100%">
          <el-table-column prop="type" label="类型" width="100" />
          <el-table-column prop="reason" label="原因" width="160" />
          <el-table-column prop="message" label="信息" min-width="260" />
          <el-table-column prop="involved_object" label="对象" min-width="200" />
          <el-table-column prop="count" label="次数" width="80" />
          <el-table-column prop="last_seen" label="最近时间" min-width="180" />
        </el-table>
      </el-tab-pane>
    </el-tabs>
  </el-card>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount, watch, computed, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const clusters = ref([])
const namespaces = ref([])
const pods = ref([])

const clusterId = ref('')
const namespace = ref('')
const podName = ref('')
const pod = ref({})

const activeTab = ref('overview')

const logText = ref('')
const logTail = ref(200)
const logContainer = ref('')
const logContainers = ref([])
const logStreaming = ref(false)
const logFilter = ref('')
let logSource = null
let retryTimer = null
const retryCount = ref(0)
const logInputRef = ref(null)
const logPaused = ref(false)
const highlightPalette = ['#fde047', '#93c5fd', '#a7f3d0', '#fca5a5', '#fcd34d']
const highlightedLog = computed(() => {
  if (!logFilter.value) return ''
  const escaped = logText.value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')

  const rawPatterns = logFilter.value
    .split(',')
    .map(v => v.trim())
    .filter(Boolean)
  const patterns = rawPatterns.filter(p => !p.startsWith('-')).map(p => p.trim())

  if (patterns.length === 0) return `<pre>${escaped}</pre>`

  const regexParts = []
  const colors = new Map()
  let colorIndex = 0
  patterns.forEach((p) => {
    const reMatch = p.match(new RegExp("^/(.*)/(i|g|ig|gi)?$"))
    if (reMatch) {
      regexParts.push(`(${reMatch[1]})`)
      colors.set(reMatch[1], highlightPalette[colorIndex % highlightPalette.length])
      colorIndex += 1
    } else {
      const escapedLiteral = p.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
      regexParts.push(`(${escapedLiteral})`)
      colors.set(escapedLiteral, highlightPalette[colorIndex % highlightPalette.length])
      colorIndex += 1
    }
  })
  const regex = new RegExp(regexParts.join('|'), 'gi')
  const html = escaped.replace(regex, (m) => {
    const key = m.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
    const color = colors.get(key) || highlightPalette[0]
    return `<mark style=\"background:${color}\">${m}</mark>`
  })
  return `<pre>${html}</pre>`
})

const jumpToMatch = async () => {
  if (!logFilter.value) return
  const rawPatterns = logFilter.value
    .split(',')
    .map(v => v.trim())
    .filter(Boolean)
  const patterns = rawPatterns.filter(p => !p.startsWith('-')).map(p => p.trim())
  const excludes = rawPatterns.filter(p => p.startsWith('-')).map(p => p.slice(1).trim()).filter(Boolean)
  if (patterns.length === 0) return

  const hay = logText.value
  let idx = -1
  patterns.some((p) => {
    const reMatch = p.match(new RegExp("^/(.*)/(i|g|ig|gi)?$"))
    if (reMatch) {
      try {
        const re = new RegExp(reMatch[1], reMatch[2] || 'i')
        const m = hay.match(re)
        if (m && m.index !== undefined) {
          idx = m.index
          return true
        }
      } catch {
        return false
      }
    } else {
      const needle = p.toLowerCase()
      const pos = hay.toLowerCase().indexOf(needle)
      if (pos >= 0) {
        idx = pos
        return true
      }
    }
    return false
  })
  if (idx < 0) return
  if (excludes.length > 0) {
    const lineText = hay.split('\n').find((_, i, arr) => {
      const pos = arr.slice(0, i).join('\n').length
      return idx <= pos + arr[i].length
    }) || ''
    const blocked = excludes.some((p) => {
      const reMatch = p.match(new RegExp("^/(.*)/(i|g|ig|gi)?$"))
      if (reMatch) {
        try {
          const re = new RegExp(reMatch[1], reMatch[2] || 'i')
          return re.test(lineText)
        } catch {
          return false
        }
      }
      return lineText.toLowerCase().includes(p.toLowerCase())
    })
    if (blocked) return
  }
  await nextTick()
  const el = logInputRef.value?.$el?.querySelector('textarea')
  if (!el) return
  el.focus()
  el.setSelectionRange(idx, Math.min(idx + 10, hay.length))
  const line = hay.slice(0, idx).split('\n').length
  const lineHeight = 18
  el.scrollTop = Math.max(0, (line - 3) * lineHeight)
}

const events = ref([])
const eventType = ref('')
const eventKeyword = ref('')

const route = useRoute()
const router = useRouter()

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

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
  await fetchPodDetail()
}

const fetchPodDetail = async () => {
  if (!clusterId.value || !namespace.value || !podName.value) return
  const res = await axios.get(`/api/v1/k8s/clusters/${clusterId.value}/namespaces/${namespace.value}/pods/${podName.value}`, {
    headers: authHeaders()
  })
  pod.value = res.data.data || {}
  logContainers.value = pod.value.containers?.map(c => c.name) || []
  if (!logContainer.value && logContainers.value.length > 0) {
    logContainer.value = logContainers.value[0]
  }
  await fetchEvents()
}

const fetchLogs = async () => {
  if (!clusterId.value || !namespace.value || !podName.value) return
  const res = await axios.get(`/api/v1/k8s/clusters/${clusterId.value}/namespaces/${namespace.value}/pods/${podName.value}/logs`, {
    headers: authHeaders(),
    params: { container: logContainer.value, tail: logTail.value }
  })
  logText.value = res.data.data || ''
}

const toggleStream = () => {
  if (logStreaming.value) {
    startStream()
  } else {
    stopStream()
  }
}

const togglePause = () => {
  logPaused.value = !logPaused.value
}

watch(logContainer, () => {
  if (logStreaming.value) {
    startStream()
  }
})

watch(logTail, () => {
  if (logStreaming.value) {
    startStream()
  }
})

const startStream = () => {
  stopStream()
  if (!clusterId.value || !namespace.value || !podName.value) return
  const token = localStorage.getItem('token')
  const url = `/api/v1/k8s/clusters/${clusterId.value}/namespaces/${namespace.value}/pods/${podName.value}/logs/stream` +
    `?container=${encodeURIComponent(logContainer.value || '')}` +
    `&tail=${encodeURIComponent(logTail.value)}` +
    `&token=${encodeURIComponent(token)}`
  logSource = new EventSource(url)
  retryCount.value = 0
  logSource.onmessage = (evt) => {
    if (logPaused.value) return
    if (logFilter.value) {
      const rawPatterns = logFilter.value
        .split(',')
        .map(v => v.trim())
        .filter(Boolean)
      const patterns = rawPatterns.filter(p => !p.startsWith('-')).map(p => p.trim())
      const excludes = rawPatterns.filter(p => p.startsWith('-')).map(p => p.slice(1).trim()).filter(Boolean)
      if (patterns.length > 0 || excludes.length > 0) {
        const text = String(evt.data || '')
        const hit = patterns.length === 0 ? true : patterns.some((p) => {
          const reMatch = p.match(new RegExp('^/(.*)/(i|g|ig|gi)?$'))
          if (reMatch) {
            try {
              const re = new RegExp(reMatch[1], reMatch[2] || 'i')
              return re.test(text)
            } catch {
              return false
            }
          }
          return text.toLowerCase().includes(p.toLowerCase())
        })
        const blocked = excludes.some((p) => {
          const reMatch = p.match(new RegExp('^/(.*)/(i|g|ig|gi)?$'))
          if (reMatch) {
            try {
              const re = new RegExp(reMatch[1], reMatch[2] || 'i')
              return re.test(text)
            } catch {
              return false
            }
          }
          return text.toLowerCase().includes(p.toLowerCase())
        })
        if (blocked) return
        if (!hit) return
      }
    }
    logText.value += evt.data + '\\n'
  }
  logSource.onerror = () => {
    if (!logStreaming.value) return
    scheduleRetry()
  }
}

const scheduleRetry = () => {
  stopStream()
  if (retryTimer) clearTimeout(retryTimer)
  const delay = Math.min(10000, 1000 * (retryCount.value + 1))
  retryTimer = setTimeout(() => {
    retryCount.value += 1
    startStream()
  }, delay)
}

const stopStream = () => {
  if (logSource) {
    logSource.close()
    logSource = null
  }
  if (retryTimer) {
    clearTimeout(retryTimer)
    retryTimer = null
  }
  logPaused.value = false
}

const fetchEvents = async () => {
  if (!clusterId.value || !namespace.value || !podName.value) return
  const res = await axios.get(`/api/v1/k8s/clusters/${clusterId.value}/events`, {
    headers: authHeaders(),
    params: { namespace: namespace.value }
  })
  const raw = res.data.data || []
  events.value = raw.filter((e) => {
    if (!(e.involved_object || '').includes(podName.value)) return false
    if (eventType.value && e.type !== eventType.value) return false
    if (eventKeyword.value) {
      const keyword = eventKeyword.value.toLowerCase()
      const reason = (e.reason || '').toLowerCase()
      const message = (e.message || '').toLowerCase()
      if (!reason.includes(keyword) && !message.includes(keyword)) return false
    }
    return true
  })
}

const handleClusterChange = async () => {
  await fetchNamespaces()
  await fetchPods()
}

const reloadAll = async () => {
  await fetchPodDetail()
  await fetchLogs()
  await fetchEvents()
}

const deletePod = async () => {
  if (!clusterId.value || !namespace.value || !podName.value) return
  await ElMessageBox.confirm(`确认删除 Pod ${podName.value} 吗？`, '提示', { type: 'warning' })
  await axios.delete(`/api/v1/k8s/clusters/${clusterId.value}/namespaces/${namespace.value}/pods/${podName.value}`, {
    headers: authHeaders()
  })
  ElMessage.success('删除成功')
  router.push({ path: '/k8s/pods', query: { clusterId: clusterId.value, namespace: namespace.value } })
}

const restartPod = async () => {
  if (!clusterId.value || !namespace.value || !podName.value) return
  await ElMessageBox.confirm(`确认重启 Pod ${podName.value} 吗？（将删除 Pod）`, '提示', { type: 'warning' })
  const res = await axios.post(`/api/v1/k8s/clusters/${clusterId.value}/namespaces/${namespace.value}/pods/${podName.value}/restart`, {}, {
    headers: authHeaders()
  })
  ElMessage.success(res.data.message || '已重启')
}

const restartWorkload = async () => {
  if (!clusterId.value || !namespace.value || !podName.value) return
  await ElMessageBox.confirm('确认对所属工作负载执行滚动重启吗？', '提示', { type: 'warning' })
  const res = await axios.post(`/api/v1/k8s/clusters/${clusterId.value}/namespaces/${namespace.value}/pods/${podName.value}/restart-workload`, {}, {
    headers: authHeaders()
  })
  ElMessage.success(res.data.message || '已触发滚动重启')
}

const openTerminal = (containerRow) => {
  router.push({
    path: '/k8s/terminal',
    query: {
      clusterId: clusterId.value,
      namespace: namespace.value,
      pod: podName.value,
      container: containerRow.name
    }
  })
}

onMounted(async () => {
  clusterId.value = route.query.clusterId || ''
  namespace.value = route.query.namespace || ''
  podName.value = route.query.name || ''
  await fetchClusters()
  await fetchNamespaces()
  await fetchPods()
})

onBeforeUnmount(() => {
  stopStream()
})
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 16px; }
.page-desc { color: #606266; margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; }
.form-block { max-width: 520px; }
.w-52 { width: 220px; }
.section-title { margin: 12px 0; }
.log-controls { display: flex; gap: 12px; margin-bottom: 12px; align-items: center; }
.event-controls { display: flex; gap: 12px; margin-bottom: 12px; align-items: center; }
.log-highlight {
  background: #0f172a;
  color: #e2e8f0;
  border-radius: 6px;
  padding: 12px;
  margin-top: 8px;
  max-height: 300px;
  overflow: auto;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
  font-size: 12px;
  line-height: 1.6;
}
.log-highlight mark {
  background: #fde047;
  color: #1f2937;
  padding: 0 2px;
}
</style>
