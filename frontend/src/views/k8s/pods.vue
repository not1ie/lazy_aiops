<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>Pods</h2>
        <p class="page-desc">Pod 列表、日志与删除操作。</p>
      </div>
      <div class="page-actions">
        <el-select v-model="clusterId" placeholder="选择集群" class="w-52" @change="handleClusterChange">
          <el-option v-for="c in clusters" :key="c.id" :label="c.display_name || c.name" :value="c.id" />
        </el-select>
        <el-select v-model="namespace" placeholder="命名空间" class="w-52" @change="fetchPods">
          <el-option v-for="ns in namespaces" :key="ns.name" :label="ns.name" :value="ns.name" />
        </el-select>
        <el-select v-model="nodeFilter" placeholder="节点" class="w-52" clearable>
          <el-option v-for="n in nodes" :key="n" :label="n" :value="n" />
        </el-select>
        <el-input v-model="keyword" placeholder="搜索名称/节点/IP" class="w-52" clearable />
        <el-select v-model="statusFilter" placeholder="状态" class="w-40" clearable>
          <el-option label="Running" value="Running" />
          <el-option label="Pending" value="Pending" />
          <el-option label="Succeeded" value="Succeeded" />
          <el-option label="Failed" value="Failed" />
          <el-option label="Unknown" value="Unknown" />
        </el-select>
        <el-select v-model="ownerKindFilter" placeholder="控制器类型" class="w-40" clearable>
          <el-option label="Deployment" value="Deployment" />
          <el-option label="StatefulSet" value="StatefulSet" />
          <el-option label="DaemonSet" value="DaemonSet" />
          <el-option label="ReplicaSet" value="ReplicaSet" />
          <el-option label="Job" value="Job" />
          <el-option label="CronJob" value="CronJob" />
        </el-select>
        <el-input v-model="ownerNameFilter" placeholder="控制器名称" class="w-52" clearable />
        <el-button icon="Download" @click="exportCSV">导出</el-button>
        <el-button icon="Refresh" @click="fetchPods">刷新</el-button>
        <el-button type="warning" plain :disabled="selectedRows.length === 0" @click="batchRestart">批量重启</el-button>
        <el-button type="danger" plain :disabled="selectedRows.length === 0" @click="batchDelete">批量删除</el-button>
      </div>
    </div>

    <el-row :gutter="16" class="summary-row">
      <el-col :span="4"><el-card><div class="card-title">总数</div><div class="card-value">{{ podStats.total }}</div></el-card></el-col>
      <el-col :span="4"><el-card><div class="card-title">Running</div><div class="card-value">{{ podStats.running }}</div></el-card></el-col>
      <el-col :span="4"><el-card><div class="card-title">Pending</div><div class="card-value">{{ podStats.pending }}</div></el-card></el-col>
      <el-col :span="4"><el-card><div class="card-title">Failed</div><div class="card-value">{{ podStats.failed }}</div></el-card></el-col>
      <el-col :span="4"><el-card><div class="card-title">Succeeded</div><div class="card-value">{{ podStats.succeeded }}</div></el-card></el-col>
      <el-col :span="4"><el-card><div class="card-title">重启次数</div><div class="card-value">{{ podStats.restarts }}</div></el-card></el-col>
    </el-row>

    <el-table :data="filteredPods" stripe style="width: 100%" v-loading="loading" @selection-change="selectedRows = $event">
      <el-table-column type="selection" width="48" />
      <el-table-column prop="namespace" label="命名空间" min-width="140" />
      <el-table-column prop="name" label="名称" min-width="220" />
      <el-table-column label="状态" width="120">
        <template #default="scope">
          <el-tag :type="statusType(scope.row.status)">{{ scope.row.status }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="控制器" min-width="180">
        <template #default="scope">
          <span class="text-xs text-gray-500" v-if="!scope.row.owner_kind">-</span>
          <el-tag
            v-else
            size="small"
            class="owner-tag"
            :type="isWorkloadOwner(scope.row) ? 'primary' : 'info'"
            @click="openOwnerWorkload(scope.row)"
          >
            {{ scope.row.owner_kind }}/{{ scope.row.owner_name }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="node" label="节点" min-width="160" />
      <el-table-column prop="ip" label="IP" width="140" />
      <el-table-column prop="restarts" label="重启" width="80" />
      <el-table-column label="容器" min-width="220">
        <template #default="scope">
          <el-tag
            v-for="c in scope.row.containers"
            :key="c.name"
            size="small"
            class="mr-2"
            :type="c.ready ? 'success' : 'warning'"
          >
            {{ c.name }} ({{ c.state || (c.ready ? 'Ready' : 'NotReady') }})
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="Labels" min-width="200">
        <template #default="scope">
          <el-popover placement="top" width="320" trigger="hover" v-if="Object.keys(scope.row.labels || {}).length">
            <div class="label-grid">
              <el-tag v-for="(v, k) in scope.row.labels" :key="k" size="small" class="mr-2 mb-2">{{ k }}={{ v }}</el-tag>
            </div>
            <template #reference>
              <div class="label-preview">
                <el-tag v-for="(v, k) in previewLabels(scope.row.labels)" :key="k" size="small" class="mr-2">{{ k }}={{ v }}</el-tag>
                <el-tag v-if="labelOverflow(scope.row.labels)" size="small" type="info">更多</el-tag>
              </div>
            </template>
          </el-popover>
          <span v-else class="text-xs text-gray-400">-</span>
        </template>
      </el-table-column>
      <el-table-column label="创建时间" min-width="180">
        <template #default="scope">
          <div>{{ formatTime(scope.row.created_at) }}</div>
          <div class="text-xs text-gray-400">{{ formatSince(scope.row.created_at) }}</div>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="340">
        <template #default="scope">
          <el-button size="small" @click="openDetail(scope.row)">详情</el-button>
          <el-button size="small" @click="openLogs(scope.row)">日志</el-button>
          <el-button size="small" type="primary" plain @click="openTerminal(scope.row)">终端</el-button>
          <el-button size="small" type="warning" @click="restartPod(scope.row)">重启</el-button>
          <el-button size="small" type="danger" @click="deletePod(scope.row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="logVisible" title="Pod 日志" width="880px">
      <div class="log-controls">
        <el-select v-model="logContainer" placeholder="容器" class="w-52">
          <el-option v-for="c in logContainers" :key="c" :label="c" :value="c" />
        </el-select>
        <el-input-number v-model="logTail" :min="10" :max="1000" />
        <el-button type="primary" @click="fetchLogs">获取日志</el-button>
      </div>
      <el-input v-model="logText" type="textarea" :rows="18" readonly />
      <template #footer>
        <el-button @click="logVisible = false">关闭</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const clusters = ref([])
const namespaces = ref([])
const clusterId = ref('')
const namespace = ref('')
const pods = ref([])
const keyword = ref('')
const statusFilter = ref('')
const nodeFilter = ref('')
const ownerKindFilter = ref('')
const ownerNameFilter = ref('')
const selectedRows = ref([])
const loading = ref(false)

const logVisible = ref(false)
const logText = ref('')
const logTail = ref(200)
const logContainer = ref('')
const logContainers = ref([])
const logPod = ref(null)
const router = useRouter()
const route = useRoute()

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
  loading.value = true
  try {
    const res = await axios.get(`/api/v1/k8s/clusters/${clusterId.value}/namespaces/${namespace.value}/pods`, {
      headers: authHeaders()
    })
    pods.value = res.data.data || []
  } finally {
    loading.value = false
  }
}

const filteredPods = computed(() => {
  const key = keyword.value.trim().toLowerCase()
  return pods.value.filter(p => {
    if (statusFilter.value && p.status !== statusFilter.value) return false
    if (nodeFilter.value && p.node !== nodeFilter.value) return false
    if (ownerKindFilter.value && p.owner_kind !== ownerKindFilter.value) return false
    if (ownerNameFilter.value && p.owner_name !== ownerNameFilter.value) return false
    if (!key) return true
    return (
      (p.name || '').toLowerCase().includes(key) ||
      (p.node || '').toLowerCase().includes(key) ||
      (p.ip || '').toLowerCase().includes(key) ||
      (p.owner_name || '').toLowerCase().includes(key) ||
      (p.owner_kind || '').toLowerCase().includes(key)
    )
  })
})

const nodes = computed(() => {
  const set = new Set(pods.value.map(p => p.node).filter(Boolean))
  return Array.from(set)
})

const podStats = computed(() => {
  const stats = { total: filteredPods.value.length, running: 0, pending: 0, failed: 0, succeeded: 0, restarts: 0 }
  filteredPods.value.forEach((p) => {
    if (p.status === 'Running') stats.running += 1
    if (p.status === 'Pending') stats.pending += 1
    if (p.status === 'Failed') stats.failed += 1
    if (p.status === 'Succeeded') stats.succeeded += 1
    stats.restarts += Number(p.restarts || 0)
  })
  return stats
})

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

const handleClusterChange = async () => {
  await fetchNamespaces()
  await fetchPods()
}

const openLogs = (row) => {
  logPod.value = row
  logContainers.value = row.containers?.map(c => c.name) || []
  logContainer.value = logContainers.value[0] || ''
  logText.value = ''
  logVisible.value = true
}

const openDetail = (row) => {
  router.push({
    path: '/k8s/pods/detail',
    query: {
      clusterId: clusterId.value,
      namespace: row.namespace,
      name: row.name
    }
  })
}

const openTerminal = (row) => {
  const container = row.containers?.[0]?.name || ''
  router.push({
    path: '/k8s/terminal',
    query: {
      clusterId: clusterId.value,
      namespace: row.namespace,
      pod: row.name,
      container
    }
  })
}

const isWorkloadOwner = (row) => {
  const kind = String(row?.owner_kind || '')
  return ['Deployment', 'StatefulSet', 'DaemonSet'].includes(kind)
}

const openOwnerWorkload = (row) => {
  if (!isWorkloadOwner(row)) return
  router.push({
    path: '/k8s/workloads/detail',
    query: {
      clusterId: clusterId.value,
      namespace: row.namespace,
      kind: row.owner_kind,
      name: row.owner_name
    }
  })
}

const fetchLogs = async () => {
  if (!logPod.value) return
  const res = await axios.get(`/api/v1/k8s/clusters/${clusterId.value}/namespaces/${logPod.value.namespace}/pods/${logPod.value.name}/logs`, {
    headers: authHeaders(),
    params: { container: logContainer.value, tail: logTail.value }
  })
  logText.value = res.data.data || ''
}

const deletePod = async (row) => {
  await ElMessageBox.confirm(`确认删除 Pod ${row.name} 吗？`, '提示', { type: 'warning' })
  await axios.delete(`/api/v1/k8s/clusters/${clusterId.value}/namespaces/${row.namespace}/pods/${row.name}`, {
    headers: authHeaders()
  })
  ElMessage.success('删除成功')
  fetchPods()
}

const restartPod = async (row) => {
  await ElMessageBox.confirm(`确认重启 Pod ${row.name} 吗？`, '提示', { type: 'warning' })
  await axios.post(`/api/v1/k8s/clusters/${clusterId.value}/namespaces/${row.namespace}/pods/${row.name}/restart`, {}, {
    headers: authHeaders()
  })
  ElMessage.success('已触发重启')
  fetchPods()
}

const batchDelete = async () => {
  if (selectedRows.value.length === 0) return
  await ElMessageBox.confirm(`确认删除选中的 ${selectedRows.value.length} 个 Pod 吗？`, '提示', { type: 'warning' })
  for (const row of selectedRows.value) {
    await axios.delete(`/api/v1/k8s/clusters/${clusterId.value}/namespaces/${row.namespace}/pods/${row.name}`, {
      headers: authHeaders()
    })
  }
  ElMessage.success('批量删除成功')
  selectedRows.value = []
  fetchPods()
}

const batchRestart = async () => {
  if (selectedRows.value.length === 0) return
  await ElMessageBox.confirm(`确认重启选中的 ${selectedRows.value.length} 个 Pod 吗？`, '提示', { type: 'warning' })
  for (const row of selectedRows.value) {
    await axios.post(`/api/v1/k8s/clusters/${clusterId.value}/namespaces/${row.namespace}/pods/${row.name}/restart`, {}, {
      headers: authHeaders()
    })
  }
  ElMessage.success('批量重启已触发')
  selectedRows.value = []
  fetchPods()
}

const exportCSV = () => {
  const headers = ['namespace','name','status','node','ip','owner_kind','owner_name','restarts','created_at']
  const rows = filteredPods.value.map(p => [p.namespace, p.name, p.status, p.node, p.ip, p.owner_kind, p.owner_name, p.restarts, p.created_at])
  const csv = [headers.join(','), ...rows.map(r => r.map(v => `"${String(v ?? '').replace(/"/g, '""')}"`).join(','))].join('\n')
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'k8s_pods.csv'
  a.click()
  URL.revokeObjectURL(url)
}

const previewLabels = (labels) => {
  const entries = Object.entries(labels || {})
  return Object.fromEntries(entries.slice(0, 3))
}

const labelOverflow = (labels) => {
  return Object.keys(labels || {}).length > 3
}

const formatTime = (value) => {
  if (!value) return '-'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return String(value)
  const pad = (v) => String(v).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

const formatSince = (value) => {
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return '-'
  const diff = Date.now() - d.getTime()
  const min = Math.floor(diff / 60000)
  if (min < 1) return '刚刚'
  if (min < 60) return `${min}m`
  const h = Math.floor(min / 60)
  if (h < 24) return `${h}h`
  const days = Math.floor(h / 24)
  return `${days}d`
}

onMounted(async () => {
  if (route.query.clusterId) clusterId.value = String(route.query.clusterId)
  if (route.query.namespace) namespace.value = String(route.query.namespace)
  if (route.query.keyword) keyword.value = String(route.query.keyword)
  if (route.query.ownerKind) ownerKindFilter.value = String(route.query.ownerKind)
  if (route.query.ownerName) ownerNameFilter.value = String(route.query.ownerName)
  await fetchClusters()
  await fetchNamespaces()
  await fetchPods()
})
</script>

<style scoped>
.page-card { max-width: 1180px; margin: 0 auto; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 16px; }
.page-desc { color: #606266; margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; flex-wrap: wrap; }
.summary-row { margin-bottom: 12px; }
.card-title { color: #909399; font-size: 12px; }
.card-value { font-size: 20px; font-weight: 600; margin-top: 6px; }
.w-52 { width: 220px; }
.w-40 { width: 160px; }
.mr-2 { margin-right: 6px; margin-bottom: 6px; }
.label-grid { display: flex; flex-wrap: wrap; }
.label-preview { display: flex; flex-wrap: wrap; }
.log-controls { display: flex; gap: 12px; margin-bottom: 12px; align-items: center; }
.owner-tag { cursor: pointer; }
</style>
