<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>Deployments</h2>
        <p class="page-desc">Deployment 视图，支持创建、扩缩容、重启和跳转 Pod/详情。</p>
      </div>
      <div class="page-actions">
        <el-select v-model="clusterId" placeholder="选择集群" class="w-52" @change="handleClusterChange">
          <el-option v-for="c in clusters" :key="c.id" :label="c.display_name || c.name" :value="c.id" />
        </el-select>
        <el-select v-model="namespace" placeholder="命名空间" class="w-52" @change="fetchDeployments">
          <el-option label="全部" value="" />
          <el-option v-for="ns in namespaces" :key="ns.name" :label="ns.name" :value="ns.name" />
        </el-select>
        <el-input v-model="keyword" placeholder="搜索名称/镜像" class="w-52" clearable />
        <el-button type="primary" icon="Plus" :disabled="!clusterId || !namespace" @click="openCreateDialog">创建 Deployment</el-button>
        <el-button icon="Refresh" @click="fetchDeployments">刷新</el-button>
      </div>
    </div>

    <el-row :gutter="16" class="summary-row">
      <el-col :span="6"><el-card><div class="card-title">总数</div><div class="card-value">{{ stats.total }}</div></el-card></el-col>
      <el-col :span="6"><el-card><div class="card-title">健康</div><div class="card-value">{{ stats.healthy }}</div></el-card></el-col>
      <el-col :span="6"><el-card><div class="card-title">异常</div><div class="card-value">{{ stats.degraded }}</div></el-card></el-col>
      <el-col :span="6"><el-card><div class="card-title">副本总数</div><div class="card-value">{{ stats.replicas }}</div></el-card></el-col>
    </el-row>

    <el-table :fit="false" :data="filteredDeployments" stripe v-loading="loading">
      <el-table-column prop="namespace" label="命名空间" min-width="130" />
      <el-table-column prop="name" label="名称" min-width="180" />
      <el-table-column label="域名解析" min-width="220">
        <template #default="{ row }">
          <span v-if="!(row.domains || []).length" class="text-xs text-gray-400">-</span>
          <div v-else class="domain-list">
            <el-link
              v-for="host in row.domains"
              :key="host"
              :href="`http://${host}`"
              target="_blank"
              type="primary"
              class="domain-link"
            >
              {{ host }}
            </el-link>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="120">
        <template #default="{ row }">
          <el-tag :type="statusType(row)">{{ statusText(row) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="副本" width="120">
        <template #default="{ row }">{{ row.ready }} / {{ row.replicas }}</template>
      </el-table-column>
      <el-table-column prop="available" label="可用" width="90" />
      <el-table-column label="镜像" min-width="280">
        <template #default="{ row }">
          <el-tag v-for="img in row.images || []" :key="img" size="small" class="mr-2">{{ img }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="创建时间" min-width="170">
        <template #default="{ row }">
          <div>{{ formatTime(row.created_at) }}</div>
          <div class="text-xs text-gray-400">{{ formatSince(row.created_at) }}</div>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="360" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="openDetail(row)">详情</el-button>
          <el-button size="small" type="primary" plain @click="openPods(row)">Pods</el-button>
          <el-button size="small" type="success" plain @click="scaleDeployment(row)">扩缩容</el-button>
          <el-button size="small" type="warning" plain @click="restartDeployment(row)">重启</el-button>
          <el-button size="small" type="danger" plain @click="deleteDeployment(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog append-to-body v-model="createVisible" title="创建 Deployment" width="720px">
      <el-form :model="form" label-width="120px">
        <el-form-item label="命名空间" required>
          <el-select v-model="form.namespace" style="width: 100%">
            <el-option v-for="ns in namespaces" :key="ns.name" :label="ns.name" :value="ns.name" />
          </el-select>
        </el-form-item>
        <el-form-item label="名称" required>
          <el-input v-model="form.name" placeholder="例如：nginx-web" />
        </el-form-item>
        <el-form-item label="镜像" required>
          <el-input v-model="form.image" placeholder="例如：nginx:1.27" />
        </el-form-item>
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="副本数">
              <el-input-number v-model="form.replicas" :min="0" :max="1000" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="容器端口">
              <el-input-number v-model="form.container_port" :min="1" :max="65535" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="标签 (k=v)">
          <el-input v-model="form.labelsText" type="textarea" :rows="2" placeholder="app=nginx,team=ops" />
        </el-form-item>
        <el-form-item label="环境变量 (k=v)">
          <el-input v-model="form.envText" type="textarea" :rows="3" placeholder="TZ=Asia/Shanghai&#10;JAVA_OPTS=-Xms256m -Xmx512m" />
        </el-form-item>
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="CPU Request">
              <el-input v-model="form.cpu_request" placeholder="100m" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="Memory Request">
              <el-input v-model="form.memory_request" placeholder="128Mi" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="CPU Limit">
              <el-input v-model="form.cpu_limit" placeholder="500m" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="Memory Limit">
              <el-input v-model="form.memory_limit" placeholder="512Mi" />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" :loading="creating" @click="createDeployment">创建</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const clusters = ref([])
const namespaces = ref([])
const clusterId = ref('')
const namespace = ref('')
const deployments = ref([])
const keyword = ref('')
const loading = ref(false)
const createVisible = ref(false)
const creating = ref(false)
const router = useRouter()
const route = useRoute()

const form = reactive({
  namespace: '',
  name: '',
  image: '',
  replicas: 1,
  container_port: 80,
  labelsText: '',
  envText: '',
  cpu_request: '',
  memory_request: '',
  cpu_limit: '',
  memory_limit: ''
})

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const parseKVText = (text) => {
  const result = {}
  String(text || '').split(/\n|,/).forEach((line) => {
    const item = line.trim()
    if (!item) return
    const idx = item.indexOf('=')
    if (idx <= 0) return
    const key = item.slice(0, idx).trim()
    const value = item.slice(idx + 1).trim()
    if (key && value) result[key] = value
  })
  return result
}

const fetchClusters = async () => {
  const res = await axios.get('/api/v1/k8s/clusters', { headers: authHeaders() })
  clusters.value = res.data.data || []
  if (!clusterId.value && clusters.value.length) {
    clusterId.value = clusters.value[0].id
  }
}

const fetchNamespaces = async () => {
  if (!clusterId.value) return
  const res = await axios.get(`/api/v1/k8s/clusters/${clusterId.value}/namespaces`, { headers: authHeaders() })
  namespaces.value = res.data.data || []
  if (!namespace.value && namespaces.value.length) {
    namespace.value = namespaces.value[0].name
  }
}

const fetchDeployments = async () => {
  if (!clusterId.value) return
  loading.value = true
  try {
    const res = await axios.get(`/api/v1/k8s/clusters/${clusterId.value}/workloads`, {
      headers: authHeaders(),
      params: { namespace: namespace.value || '' }
    })
    deployments.value = (res.data.data || []).filter((item) => item.kind === 'Deployment')
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '获取 Deployment 失败')
  } finally {
    loading.value = false
  }
}

const filteredDeployments = computed(() => {
  const key = keyword.value.trim().toLowerCase()
  return deployments.value.filter((item) => {
    if (!key) return true
    const images = (item.images || []).join(',').toLowerCase()
    const domains = (item.domains || []).join(',').toLowerCase()
    return (item.name || '').toLowerCase().includes(key) || images.includes(key) || domains.includes(key)
  })
})

const stats = computed(() => {
  const val = { total: filteredDeployments.value.length, healthy: 0, degraded: 0, replicas: 0 }
  filteredDeployments.value.forEach((item) => {
    const replicas = Number(item.replicas || 0)
    const ready = Number(item.ready || 0)
    val.replicas += replicas
    if (replicas === 0 || ready < replicas) val.degraded += 1
    else val.healthy += 1
  })
  return val
})

const statusText = (row) => {
  const replicas = Number(row.replicas || 0)
  const ready = Number(row.ready || 0)
  if (replicas === 0) return 'Scaled 0'
  if (ready < replicas) return 'Degraded'
  return 'Healthy'
}

const statusType = (row) => {
  const text = statusText(row)
  if (text === 'Healthy') return 'success'
  if (text === 'Scaled 0') return 'info'
  return 'warning'
}

const handleClusterChange = async () => {
  namespace.value = ''
  await fetchNamespaces()
  await fetchDeployments()
}

const openDetail = (row) => {
  router.push({
    path: '/k8s/workloads/detail',
    query: {
      clusterId: clusterId.value,
      namespace: row.namespace,
      kind: 'Deployment',
      name: row.name
    }
  })
}

const openPods = (row) => {
  router.push({
    path: '/k8s/pods',
    query: {
      clusterId: clusterId.value,
      namespace: row.namespace,
      ownerKind: 'Deployment',
      ownerName: row.name
    }
  })
}

const scaleDeployment = async (row) => {
  try {
    const { value } = await ElMessageBox.prompt('输入副本数', `扩缩容 ${row.name}`, {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      inputValue: String(row.replicas ?? 1),
      inputPattern: /^[0-9]+$/,
      inputErrorMessage: '请输入数字'
    })
    await axios.put(`/api/v1/k8s/clusters/${clusterId.value}/namespaces/${row.namespace}/deployments/${row.name}/scale`, {
      replicas: Number(value)
    }, { headers: authHeaders() })
    ElMessage.success('扩缩容已提交')
    await fetchDeployments()
  } catch (err) {
    if (err !== 'cancel') ElMessage.error(err.response?.data?.message || '扩缩容失败')
  }
}

const restartDeployment = async (row) => {
  try {
    await ElMessageBox.confirm(`确认重启 Deployment ${row.name} 吗？`, '提示', { type: 'warning' })
    await axios.post(`/api/v1/k8s/clusters/${clusterId.value}/namespaces/${row.namespace}/deployments/${row.name}/restart`, {}, {
      headers: authHeaders()
    })
    ElMessage.success('已触发滚动重启')
    await fetchDeployments()
  } catch (err) {
    if (err !== 'cancel') ElMessage.error(err.response?.data?.message || '重启失败')
  }
}

const deleteDeployment = async (row) => {
  try {
    await ElMessageBox.confirm(`确认删除 Deployment ${row.name} 吗？`, '提示', { type: 'warning' })
    await axios.delete(`/api/v1/k8s/clusters/${clusterId.value}/namespaces/${row.namespace}/deployments/${row.name}`, {
      headers: authHeaders()
    })
    ElMessage.success('删除成功')
    await fetchDeployments()
  } catch (err) {
    if (err !== 'cancel') ElMessage.error(err.response?.data?.message || '删除失败')
  }
}

const resetForm = () => {
  form.namespace = namespace.value || (namespaces.value[0]?.name || '')
  form.name = ''
  form.image = ''
  form.replicas = 1
  form.container_port = 80
  form.labelsText = ''
  form.envText = ''
  form.cpu_request = ''
  form.memory_request = ''
  form.cpu_limit = ''
  form.memory_limit = ''
}

const openCreateDialog = () => {
  resetForm()
  createVisible.value = true
}

const createDeployment = async () => {
  if (!form.namespace || !form.name.trim() || !form.image.trim()) {
    ElMessage.warning('命名空间、名称、镜像为必填')
    return
  }
  creating.value = true
  try {
    const payload = {
      name: form.name.trim(),
      image: form.image.trim(),
      replicas: Number(form.replicas || 1),
      container_port: Number(form.container_port || 0),
      labels: parseKVText(form.labelsText),
      env: parseKVText(form.envText),
      cpu_request: form.cpu_request.trim(),
      memory_request: form.memory_request.trim(),
      cpu_limit: form.cpu_limit.trim(),
      memory_limit: form.memory_limit.trim()
    }
    await axios.post(`/api/v1/k8s/clusters/${clusterId.value}/namespaces/${form.namespace}/deployments`, payload, {
      headers: authHeaders()
    })
    ElMessage.success('创建成功')
    createVisible.value = false
    namespace.value = form.namespace
    await fetchDeployments()
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '创建失败')
  } finally {
    creating.value = false
  }
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
  await fetchClusters()
  await fetchNamespaces()
  await fetchDeployments()
})
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 16px; }
.page-desc { color: #606266; margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; flex-wrap: wrap; }
.summary-row { margin-bottom: 12px; }
.card-title { color: #909399; font-size: 12px; }
.card-value { font-size: 20px; font-weight: 600; margin-top: 6px; }
.w-52 { width: 220px; }
.mr-2 { margin-right: 6px; margin-bottom: 6px; }
.domain-list { display: flex; flex-wrap: wrap; gap: 6px; }
.domain-link { font-size: 12px; }
</style>
