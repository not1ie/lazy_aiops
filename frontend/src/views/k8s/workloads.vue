<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>工作负载</h2>
        <p class="page-desc">Deployment/StatefulSet/DaemonSet 等工作负载。</p>
      </div>
      <div class="page-actions">
        <el-select v-model="clusterId" placeholder="选择集群" class="w-52" @change="handleClusterChange">
          <el-option v-for="c in clusters" :key="c.id" :label="c.display_name || c.name" :value="c.id" />
        </el-select>
        <el-select v-model="namespace" placeholder="命名空间" class="w-52" @change="fetchWorkloads">
          <el-option label="全部" value="" />
          <el-option v-for="ns in namespaces" :key="ns.name" :label="ns.name" :value="ns.name" />
        </el-select>
        <el-input v-model="keyword" placeholder="搜索名称/镜像" class="w-52" clearable />
        <el-select v-model="kindFilter" placeholder="类型" class="w-40" clearable>
          <el-option label="Deployment" value="Deployment" />
          <el-option label="StatefulSet" value="StatefulSet" />
          <el-option label="DaemonSet" value="DaemonSet" />
          <el-option label="Job" value="Job" />
          <el-option label="CronJob" value="CronJob" />
        </el-select>
        <el-button icon="Download" @click="exportCSV">导出</el-button>
        <el-button icon="Refresh" @click="fetchWorkloads">刷新</el-button>
      </div>
    </div>

    <el-row :gutter="16" class="summary-row">
      <el-col :span="4"><el-card><div class="card-title">总数</div><div class="card-value">{{ workloadStats.total }}</div></el-card></el-col>
      <el-col :span="4"><el-card><div class="card-title">健康</div><div class="card-value">{{ workloadStats.healthy }}</div></el-card></el-col>
      <el-col :span="4"><el-card><div class="card-title">告警</div><div class="card-value">{{ workloadStats.degraded }}</div></el-card></el-col>
      <el-col :span="4"><el-card><div class="card-title">无就绪</div><div class="card-value">{{ workloadStats.zeroReady }}</div></el-card></el-col>
      <el-col :span="4"><el-card><div class="card-title">Deployment</div><div class="card-value">{{ workloadStats.deployment }}</div></el-card></el-col>
      <el-col :span="4"><el-card><div class="card-title">StatefulSet</div><div class="card-value">{{ workloadStats.statefulset }}</div></el-card></el-col>
    </el-row>

    <el-table :data="filteredWorkloads" stripe style="width: 100%" v-loading="loading">
      <el-table-column prop="namespace" label="命名空间" min-width="140" />
      <el-table-column prop="name" label="名称" min-width="200" />
      <el-table-column prop="kind" label="类型" width="140" />
      <el-table-column label="状态" width="120">
        <template #default="scope">
          <el-tag :type="workloadStatus(scope.row).type">{{ workloadStatus(scope.row).text }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="replicas" label="副本" width="90" />
      <el-table-column prop="ready" label="就绪" width="90" />
      <el-table-column prop="available" label="可用" width="90" />
      <el-table-column label="镜像" min-width="220">
        <template #default="scope">
          <el-tag v-for="img in scope.row.images" :key="img" size="small" class="mr-2">{{ img }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="创建时间" min-width="180">
        <template #default="scope">
          <div>{{ formatTime(scope.row.created_at) }}</div>
          <div class="text-xs text-gray-400">{{ formatSince(scope.row.created_at) }}</div>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="240">
        <template #default="scope">
          <el-button size="small" @click="openDetail(scope.row)">详情</el-button>
          <el-button size="small" type="primary" plain @click="scaleWorkload(scope.row)" :disabled="scope.row.kind === 'DaemonSet'">扩缩容</el-button>
          <el-button size="small" type="warning" plain @click="restartWorkload(scope.row)">重启</el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-card>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const clusters = ref([])
const namespaces = ref([])
const clusterId = ref('')
const namespace = ref('')
const workloads = ref([])
const keyword = ref('')
const kindFilter = ref('')
const loading = ref(false)
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
}

const fetchWorkloads = async () => {
  if (!clusterId.value) return
  loading.value = true
  try {
    const res = await axios.get(`/api/v1/k8s/clusters/${clusterId.value}/workloads`, {
      headers: authHeaders(),
      params: { namespace }
    })
    workloads.value = res.data.data || []
  } finally {
    loading.value = false
  }
}

const filteredWorkloads = computed(() => {
  const key = keyword.value.trim().toLowerCase()
  return workloads.value.filter(w => {
    if (kindFilter.value && w.kind !== kindFilter.value) return false
    if (!key) return true
    const images = (w.images || []).join(',').toLowerCase()
    return (w.name || '').toLowerCase().includes(key) || images.includes(key)
  })
})

const handleClusterChange = async () => {
  await fetchNamespaces()
  namespace.value = ''
  await fetchWorkloads()
}

const openDetail = (row) => {
  router.push({
    path: '/k8s/workloads/detail',
    query: {
      clusterId: clusterId.value,
      namespace: row.namespace,
      kind: row.kind,
      name: row.name
    }
  })
}

const workloadStatus = (row) => {
  if (!row) return { text: '-', type: 'info' }
  if (row.ready === 0 && row.replicas > 0) return { text: 'Degraded', type: 'danger' }
  if (row.ready < row.replicas) return { text: 'Updating', type: 'warning' }
  if (row.replicas === 0) return { text: 'Scaled 0', type: 'info' }
  return { text: 'Healthy', type: 'success' }
}

const workloadStats = computed(() => {
  const stats = {
    total: filteredWorkloads.value.length,
    healthy: 0,
    degraded: 0,
    zeroReady: 0,
    deployment: 0,
    statefulset: 0
  }
  filteredWorkloads.value.forEach((w) => {
    const status = workloadStatus(w).text
    if (status === 'Healthy') stats.healthy += 1
    if (status === 'Degraded') stats.degraded += 1
    if (w.ready === 0) stats.zeroReady += 1
    if (w.kind === 'Deployment') stats.deployment += 1
    if (w.kind === 'StatefulSet') stats.statefulset += 1
  })
  return stats
})

const scaleWorkload = async (row) => {
  if (!row || row.kind === 'DaemonSet') return
  try {
    const { value } = await ElMessageBox.prompt('输入副本数', '扩缩容', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      inputValue: String(row.replicas ?? 1),
      inputPattern: /^[0-9]+$/,
      inputErrorMessage: '请输入数字'
    })
    await axios.put(`/api/v1/k8s/clusters/${clusterId.value}/namespaces/${row.namespace}/workloads/${row.kind}/${row.name}/scale`, {
      replicas: Number(value)
    }, { headers: authHeaders() })
    ElMessage.success('已提交扩缩容')
    fetchWorkloads()
  } catch (e) {}
}

const restartWorkload = async (row) => {
  if (!row) return
  await ElMessageBox.confirm('确认执行滚动重启吗？', '提示', { type: 'warning' })
  await axios.post(`/api/v1/k8s/clusters/${clusterId.value}/namespaces/${row.namespace}/workloads/${row.kind}/${row.name}/restart`, {}, {
    headers: authHeaders()
  })
  ElMessage.success('已触发滚动重启')
  fetchWorkloads()
}

const exportCSV = () => {
  const headers = ['namespace','name','kind','replicas','ready','available','images','created_at','status']
  const rows = filteredWorkloads.value.map(w => [
    w.namespace, w.name, w.kind, w.replicas, w.ready, w.available,
    (w.images || []).join('|'), w.created_at, workloadStatus(w).text
  ])
  const csv = [headers.join(','), ...rows.map(r => r.map(v => `"${String(v ?? '').replace(/"/g, '""')}"`).join(','))].join('\n')
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'k8s_workloads.csv'
  a.click()
  URL.revokeObjectURL(url)
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
  await fetchClusters()
  await fetchNamespaces()
  await fetchWorkloads()
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
</style>
