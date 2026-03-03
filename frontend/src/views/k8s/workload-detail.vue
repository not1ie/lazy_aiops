<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>工作负载详情</h2>
        <p class="page-desc">Deployment / StatefulSet / DaemonSet 详情与操作。</p>
      </div>
      <div class="page-actions">
        <el-button icon="Refresh" @click="reloadAll">刷新</el-button>
        <el-button type="primary" @click="restartWorkload">滚动重启</el-button>
      </div>
    </div>

    <el-form label-width="110px" class="form-block">
      <el-form-item label="集群">
        <el-select v-model="clusterId" placeholder="选择集群" class="w-52" @change="handleClusterChange">
          <el-option v-for="c in clusters" :key="c.id" :label="c.display_name || c.name" :value="c.id" />
        </el-select>
      </el-form-item>
      <el-form-item label="命名空间">
        <el-select v-model="namespace" placeholder="命名空间" class="w-52" @change="fetchWorkloadList">
          <el-option v-for="ns in namespaces" :key="ns.name" :label="ns.name" :value="ns.name" />
        </el-select>
      </el-form-item>
      <el-form-item label="类型">
        <el-select v-model="kind" placeholder="类型" class="w-52" @change="fetchWorkloadList">
          <el-option label="Deployment" value="Deployment" />
          <el-option label="StatefulSet" value="StatefulSet" />
          <el-option label="DaemonSet" value="DaemonSet" />
        </el-select>
      </el-form-item>
      <el-form-item label="工作负载">
        <el-select v-model="name" placeholder="选择工作负载" class="w-52" @change="fetchWorkloadDetail">
          <el-option v-for="w in workloadList" :key="w.name" :label="w.name" :value="w.name" />
        </el-select>
      </el-form-item>
    </el-form>

    <el-descriptions :column="2" border>
      <el-descriptions-item label="名称">{{ workload.name }}</el-descriptions-item>
      <el-descriptions-item label="命名空间">{{ workload.namespace }}</el-descriptions-item>
      <el-descriptions-item label="类型">{{ workload.kind }}</el-descriptions-item>
      <el-descriptions-item label="副本">{{ workload.replicas }}</el-descriptions-item>
      <el-descriptions-item label="就绪">{{ workload.ready }}</el-descriptions-item>
      <el-descriptions-item label="可用">{{ workload.available }}</el-descriptions-item>
      <el-descriptions-item label="Selector">
        <el-tag v-for="(v, k) in workload.selector || {}" :key="k" size="small" class="mr-2">{{ k }}={{ v }}</el-tag>
      </el-descriptions-item>
      <el-descriptions-item label="创建时间">{{ workload.created_at }}</el-descriptions-item>
    </el-descriptions>

    <el-divider />

    <div class="scale-block">
      <el-input-number v-model="targetReplicas" :min="0" :disabled="!canScale" />
      <el-button type="primary" :disabled="!canScale" @click="scaleWorkload">扩缩容</el-button>
      <span v-if="!canScale" class="scale-hint">DaemonSet 不支持手动扩缩容</span>
    </div>

    <el-divider />

    <el-tabs v-model="activeTab" class="detail-tabs">
      <el-tab-pane label="概览" name="overview">
        <h3 class="section-title">镜像</h3>
        <el-tag v-for="img in workload.images || []" :key="img" class="mr-2" size="small">{{ img }}</el-tag>

        <el-divider />

        <h3 class="section-title">关联 Pods</h3>
        <el-table :data="pods" stripe style="width: 100%">
          <el-table-column prop="name" label="Pod" min-width="200" />
          <el-table-column label="状态" width="120">
            <template #default="scope">
              <el-tag :type="statusType(scope.row.status)">{{ scope.row.status }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="node" label="节点" min-width="160" />
          <el-table-column prop="ip" label="IP" width="140" />
          <el-table-column prop="restarts" label="重启" width="80" />
          <el-table-column prop="created_at" label="创建时间" min-width="180" />
          <el-table-column label="操作" width="140">
            <template #default="scope">
              <el-button size="small" @click="openPodDetail(scope.row)">详情</el-button>
            </template>
          </el-table-column>
        </el-table>
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
        <el-table :data="events" stripe style="width: 100%">
          <el-table-column prop="type" label="类型" width="100" />
          <el-table-column prop="reason" label="原因" width="160" />
          <el-table-column prop="message" label="信息" min-width="260" />
          <el-table-column prop="involved_object" label="对象" min-width="200" />
          <el-table-column prop="count" label="次数" width="80" />
          <el-table-column prop="last_seen" label="最近时间" min-width="180" />
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="日志" name="logs">
        <div class="log-controls">
          <el-select v-model="logPod" placeholder="Pod" class="w-52" @change="handleLogPodChange">
            <el-option v-for="p in pods" :key="p.name" :label="p.name" :value="p.name" />
          </el-select>
          <el-select v-model="logContainer" placeholder="容器" class="w-52">
            <el-option v-for="c in logContainers" :key="c" :label="c" :value="c" />
          </el-select>
          <el-input-number v-model="logTail" :min="10" :max="2000" />
          <el-button type="primary" @click="fetchLogs">获取日志</el-button>
          <el-button @click="logText = ''">清空</el-button>
        </div>
        <el-input v-model="logText" type="textarea" :rows="18" readonly />
      </el-tab-pane>

      <el-tab-pane label="YAML/JSON" name="manifest">
        <div class="log-controls">
          <el-select v-model="manifestFormat" placeholder="格式" class="w-52" @change="fetchManifest">
            <el-option label="YAML" value="yaml" />
            <el-option label="JSON" value="json" />
          </el-select>
          <el-button type="primary" @click="fetchManifest">获取</el-button>
          <el-button @click="manifestEditable = !manifestEditable">{{ manifestEditable ? '取消编辑' : '编辑' }}</el-button>
          <el-button @click="copyManifest">复制</el-button>
          <el-button @click="downloadManifest">下载</el-button>
          <el-button @click="downloadDiff">导出补丁</el-button>
          <el-button type="danger" :disabled="!manifestEditable" @click="openApplyDialog">应用变更</el-button>
        </div>
        <el-input v-model="manifestText" type="textarea" :rows="18" :readonly="!manifestEditable" />
      </el-tab-pane>
    </el-tabs>
  </el-card>

  <el-dialog append-to-body v-model="applyVisible" title="确认应用变更" width="980px">
    <p class="apply-hint">请确认以下变更将会写入集群，操作不可逆。</p>
    <div class="diff-block">
      <div class="apply-title">差异预览</div>
      <div class="diff-toolbar">
        <el-switch v-model="diffHideEqual" active-text="隐藏未变行" />
        <el-switch v-model="diffCollapseEqual" active-text="折叠未变行" />
      </div>
      <div class="diff-view">
        <div class="diff-header">
          <span class="diff-no">OLD</span>
          <span class="diff-no">NEW</span>
          <span class="diff-content">CONTENT</span>
        </div>
        <pre class="diff-body">
        <template v-for="(line, idx) in diffDisplayLines" :key="idx">
          <span :class="['diff-line', line.type, line.block ? 'diff-block-line' : '']">
            <template v-if="line.block">
              <span class="diff-no"></span>
              <span class="diff-no"></span>
              <span class="diff-content">
                {{ line.text }}
                <el-button text size="small" class="diff-expand" @click="toggleBlock(line.blockId)">
                  {{ expandedBlocks.has(line.blockId) ? '收起' : '展开' }}
                </el-button>
              </span>
            </template>
            <template v-else>
              <span class="diff-no">{{ line.oldNo || '' }}</span>
              <span class="diff-no">{{ line.newNo || '' }}</span>
              <span class="diff-content">{{ line.prefix }}{{ line.text }}</span>
            </template>
          </span>
        </template>
        </pre>
      </div>
    </div>
    <div class="apply-grid">
      <div>
        <div class="apply-title">原始内容</div>
        <el-input v-model="manifestOriginal" type="textarea" :rows="16" readonly />
      </div>
      <div>
        <div class="apply-title">修改后内容</div>
        <el-input v-model="manifestText" type="textarea" :rows="16" readonly />
      </div>
    </div>
    <template #footer>
      <el-button @click="applyVisible = false">取消</el-button>
      <el-button type="danger" @click="applyManifest">确认应用</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const clusters = ref([])
const namespaces = ref([])
const workloadList = ref([])

const clusterId = ref('')
const namespace = ref('')
const kind = ref('Deployment')
const name = ref('')
const workload = ref({})
const targetReplicas = ref(1)
const pods = ref([])
const activeTab = ref('overview')

const events = ref([])
const eventType = ref('')
const eventKeyword = ref('')

const logText = ref('')
const logTail = ref(200)
const logPod = ref('')
const logContainer = ref('')
const logContainers = ref([])

const manifestFormat = ref('yaml')
const manifestText = ref('')
const manifestOriginal = ref('')
const manifestEditable = ref(false)
const applyVisible = ref(false)
const diffLines = computed(() => buildDiff(manifestOriginal.value || '', manifestText.value || ''))
const diffHideEqual = ref(false)
const diffCollapseEqual = ref(true)
const filteredDiffLinesWithNo = computed(() => {
  const lines = diffLines.value
  let oldNo = 0
  let newNo = 0
  const mapped = lines.map(line => {
    const item = { ...line, oldNo: '', newNo: '' }
    if (line.type === 'equal') {
      oldNo += 1
      newNo += 1
      item.oldNo = oldNo
      item.newNo = newNo
    } else if (line.type === 'del') {
      oldNo += 1
      item.oldNo = oldNo
    } else if (line.type === 'add') {
      newNo += 1
      item.newNo = newNo
    }
    return item
  })
  if (diffHideEqual.value) return mapped.filter(line => line.type !== 'equal')
  return mapped
})
const expandedBlocks = ref(new Set())
const diffDisplayLines = computed(() => {
  const lines = filteredDiffLinesWithNo.value
  if (diffHideEqual.value || !diffCollapseEqual.value) return lines
  const output = []
  let buffer = []
  let blockId = 0
  const flush = () => {
    if (buffer.length <= 6) {
      output.push(...buffer)
    } else {
      const id = `b${blockId++}`
      if (expandedBlocks.value.has(id)) {
        output.push(...buffer)
        buffer = []
        return
      }
      output.push(...buffer.slice(0, 3))
      output.push({ block: true, blockId: id, type: 'block', text: `... 已折叠 ${buffer.length - 6} 行未变内容 ...` })
      output.push(...buffer.slice(-3))
    }
    buffer = []
  }
  for (const line of lines) {
    if (line.type === 'equal') {
      buffer.push(line)
    } else {
      if (buffer.length) flush()
      output.push(line)
    }
  }
  if (buffer.length) flush()
  return output
})

const toggleBlock = (id) => {
  if (!id) return
  if (expandedBlocks.value.has(id)) {
    expandedBlocks.value.delete(id)
  } else {
    expandedBlocks.value.add(id)
  }
}
const canScale = computed(() => kind.value !== 'DaemonSet')

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

const fetchWorkloadList = async () => {
  if (!clusterId.value || !namespace.value) return
  const res = await axios.get(`/api/v1/k8s/clusters/${clusterId.value}/workloads`, {
    headers: authHeaders(),
    params: { namespace: namespace.value }
  })
  workloadList.value = (res.data.data || []).filter(w => w.kind === kind.value)
  if (!name.value && workloadList.value.length > 0) {
    name.value = workloadList.value[0].name
  }
  await fetchWorkloadDetail()
}

const fetchWorkloadDetail = async () => {
  if (!clusterId.value || !namespace.value || !name.value) return
  const res = await axios.get(`/api/v1/k8s/clusters/${clusterId.value}/namespaces/${namespace.value}/workloads/${kind.value}/${name.value}`, {
    headers: authHeaders()
  })
  workload.value = res.data.data || {}
  if (typeof workload.value.replicas === 'number') {
    targetReplicas.value = workload.value.replicas
  }
  await fetchPodsBySelector()
}

const fetchPodsBySelector = async () => {
  if (!clusterId.value || !namespace.value) return
  const selector = workload.value.selector || {}
  const selectorStr = Object.entries(selector).map(([k, v]) => `${k}=${v}`).join(',')
  if (!selectorStr) {
    pods.value = []
    events.value = []
    return
  }
  const res = await axios.get(`/api/v1/k8s/clusters/${clusterId.value}/namespaces/${namespace.value}/pods`, {
    headers: authHeaders(),
    params: { selector: selectorStr }
  })
  pods.value = res.data.data || []
  if (!logPod.value && pods.value.length > 0) {
    logPod.value = pods.value[0].name
    handleLogPodChange()
  }
}

const openPodDetail = (row) => {
  router.push({
    path: '/k8s/pods/detail',
    query: {
      clusterId: clusterId.value,
      namespace: row.namespace,
      name: row.name
    }
  })
}

const statusType = (status) => {
  switch (status) {
    case 'Running':
      return 'success'
    case 'Pending':
      return 'warning'
    case 'Failed':
      return 'danger'
    case 'Succeeded':
      return 'info'
    default:
      return 'info'
  }
}

const fetchEvents = async () => {
  if (!clusterId.value || !namespace.value) return
  const res = await axios.get(`/api/v1/k8s/clusters/${clusterId.value}/events`, {
    headers: authHeaders(),
    params: { namespace: namespace.value }
  })
  const raw = res.data.data || []
  const podNames = new Set(pods.value.map(p => p.name))
  events.value = raw.filter((e) => {
    const involved = e.involved_object || ''
    const hasPod = Array.from(podNames).some((p) => involved.includes(p))
    if (!hasPod) return false
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

const handleLogPodChange = () => {
  const pod = pods.value.find(p => p.name === logPod.value)
  logContainers.value = pod?.containers?.map(c => c.name) || []
  if (!logContainer.value && logContainers.value.length > 0) {
    logContainer.value = logContainers.value[0]
  }
}

const fetchLogs = async () => {
  if (!clusterId.value || !namespace.value || !logPod.value) return
  const res = await axios.get(`/api/v1/k8s/clusters/${clusterId.value}/namespaces/${namespace.value}/pods/${logPod.value}/logs`, {
    headers: authHeaders(),
    params: { container: logContainer.value, tail: logTail.value }
  })
  logText.value = res.data.data || ''
}

const fetchManifest = async () => {
  if (!clusterId.value || !namespace.value || !name.value) return
  const res = await axios.get(`/api/v1/k8s/clusters/${clusterId.value}/namespaces/${namespace.value}/workloads/${kind.value}/${name.value}/manifest`, {
    headers: authHeaders(),
    params: { format: manifestFormat.value }
  })
  manifestText.value = res.data?.data?.content || ''
  manifestOriginal.value = manifestText.value
  manifestEditable.value = false
}

const copyManifest = async () => {
  if (!manifestText.value) return
  await navigator.clipboard.writeText(manifestText.value)
  ElMessage.success('已复制')
}

const downloadManifest = () => {
  if (!manifestText.value) return
  const blob = new Blob([manifestText.value], { type: manifestFormat.value === 'json' ? 'application/json' : 'text/yaml' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `${name.value || 'workload'}.${manifestFormat.value === 'json' ? 'json' : 'yaml'}`
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(url)
}

const openApplyDialog = () => {
  if (!manifestEditable.value) return
  if (manifestText.value === manifestOriginal.value) {
    ElMessage.info('未检测到变更')
    return
  }
  applyVisible.value = true
}

const applyManifest = async () => {
  await axios.post(`/api/v1/k8s/clusters/${clusterId.value}/namespaces/${namespace.value}/workloads/${kind.value}/${name.value}/manifest/apply`, {
    format: manifestFormat.value,
    content: manifestText.value
  }, { headers: authHeaders() })
  ElMessage.success('已应用变更')
  applyVisible.value = false
  manifestEditable.value = false
  await fetchManifest()
  await fetchWorkloadDetail()
}

const downloadDiff = () => {
  const lines = diffLines.value
  if (!lines.length) return
  const text = lines.map(l => `${l.prefix}${l.text}`).join('\n')
  const blob = new Blob([text], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `${name.value || 'workload'}.diff`
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(url)
}

const buildDiff = (oldText, newText) => {
  const a = oldText.split('\n')
  const b = newText.split('\n')
  const n = a.length
  const m = b.length
  const max = n + m
  const v = new Array(2 * max + 1).fill(0)
  const trace = []

  for (let d = 0; d <= max; d++) {
    trace.push([...v])
    for (let k = -d; k <= d; k += 2) {
      const kIndex = k + max
      let x
      if (k === -d || (k !== d && v[kIndex - 1] < v[kIndex + 1])) {
        x = v[kIndex + 1]
      } else {
        x = v[kIndex - 1] + 1
      }
      let y = x - k
      while (x < n && y < m && a[x] === b[y]) {
        x++
        y++
      }
      v[kIndex] = x
      if (x >= n && y >= m) {
        return backtrackDiff(trace, a, b, max)
      }
    }
  }
  return []
}

const backtrackDiff = (trace, a, b, max) => {
  let x = a.length
  let y = b.length
  const result = []
  for (let d = trace.length - 1; d >= 0; d--) {
    const v = trace[d]
    const k = x - y
    const kIndex = k + max
    let prevK
    if (k === -d || (k !== d && v[kIndex - 1] < v[kIndex + 1])) {
      prevK = k + 1
    } else {
      prevK = k - 1
    }
    const prevX = v[prevK + max]
    const prevY = prevX - prevK

    while (x > prevX && y > prevY) {
      result.push({ type: 'equal', prefix: ' ', text: a[x - 1] })
      x--
      y--
    }

    if (d === 0) break

    if (x === prevX) {
      result.push({ type: 'add', prefix: '+', text: b[y - 1] })
      y--
    } else {
      result.push({ type: 'del', prefix: '-', text: a[x - 1] })
      x--
    }
  }
  return result.reverse()
}

const scaleWorkload = async () => {
  if (!clusterId.value || !namespace.value || !name.value) return
  await axios.put(`/api/v1/k8s/clusters/${clusterId.value}/namespaces/${namespace.value}/workloads/${kind.value}/${name.value}/scale`, {
    replicas: targetReplicas.value
  }, { headers: authHeaders() })
  ElMessage.success('扩缩容成功')
  await fetchWorkloadDetail()
}

const restartWorkload = async () => {
  if (!clusterId.value || !namespace.value || !name.value) return
  await ElMessageBox.confirm('确认执行滚动重启吗？', '提示', { type: 'warning' })
  await axios.post(`/api/v1/k8s/clusters/${clusterId.value}/namespaces/${namespace.value}/workloads/${kind.value}/${name.value}/restart`, {}, {
    headers: authHeaders()
  })
  ElMessage.success('已触发滚动重启')
}

const handleClusterChange = async () => {
  await fetchNamespaces()
  await fetchWorkloadList()
}

const reloadAll = async () => {
  await fetchWorkloadDetail()
  await fetchEvents()
  await fetchManifest()
}

onMounted(async () => {
  clusterId.value = route.query.clusterId || ''
  namespace.value = route.query.namespace || ''
  kind.value = route.query.kind || 'Deployment'
  name.value = route.query.name || ''

  await fetchClusters()
  await fetchNamespaces()
  await fetchWorkloadList()
  await fetchEvents()
  await fetchManifest()
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
.mr-2 { margin-right: 6px; margin-bottom: 6px; }
.scale-block { display: flex; gap: 12px; align-items: center; }
.scale-hint { color: #909399; font-size: 12px; }
.detail-tabs { margin-top: 12px; }
.event-controls { display: flex; gap: 12px; margin-bottom: 12px; align-items: center; }
.log-controls { display: flex; gap: 12px; margin-bottom: 12px; align-items: center; }
.apply-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; margin-top: 12px; }
.apply-title { font-weight: 600; margin-bottom: 8px; }
.apply-hint { color: #e6a23c; margin-bottom: 8px; }
.diff-block { margin-bottom: 12px; }
.diff-toolbar { display: flex; justify-content: flex-end; gap: 12px; margin-bottom: 6px; }
.diff-view {
  background: #0f172a;
  color: #e2e8f0;
  padding: 12px;
  border-radius: 8px;
  max-height: 240px;
  overflow: auto;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
  font-size: 12px;
}
.diff-header {
  display: grid;
  grid-template-columns: 46px 46px 1fr;
  gap: 8px;
  position: sticky;
  top: 0;
  background: #0b1220;
  padding-bottom: 6px;
  margin-bottom: 6px;
  z-index: 1;
  text-transform: uppercase;
  font-size: 11px;
  letter-spacing: 0.6px;
  color: #94a3b8;
}
.diff-body { white-space: pre; margin: 0; }
.diff-line { display: grid; grid-template-columns: 46px 46px 1fr; gap: 8px; }
.diff-no { color: #94a3b8; text-align: right; }
.diff-content { white-space: pre; }
.diff-view .add .diff-content { color: #86efac; }
.diff-view .del .diff-content { color: #fca5a5; }
.diff-view .equal .diff-content { color: #cbd5f5; }
.diff-view .block .diff-content { color: #fbbf24; font-style: italic; }
</style>
