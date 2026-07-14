<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>工作负载</h2>
        <p class="page-desc">容器集群中的 Deployment/StatefulSet/DaemonSet 等负载统一管理台。</p>
      </div>
      <div class="page-actions">
        <el-select v-model="clusterId" placeholder="选择集群" class="w-52" @change="handleClusterChange">
          <el-option v-for="c in clusters" :key="c.id" :label="c.display_name || c.name" :value="c.id" />
        </el-select>
        <el-select v-model="namespace" placeholder="命名空间" class="w-52" @change="handleNamespaceChange">
          <el-option label="全部" value="" />
          <el-option v-for="ns in namespaces" :key="ns.name" :label="ns.name" :value="ns.name" />
        </el-select>
        <el-input v-model="keyword" placeholder="搜索 名称/镜像/域名" class="w-52" clearable />
        <el-select v-model="kindFilter" placeholder="类型" class="w-40" clearable>
          <el-option label="Deployment" value="Deployment" />
          <el-option label="StatefulSet" value="StatefulSet" />
          <el-option label="DaemonSet" value="DaemonSet" />
          <el-option label="Job" value="Job" />
          <el-option label="CronJob" value="CronJob" />
        </el-select>
        <el-button type="primary" icon="Plus" :disabled="!clusterId || !namespace" @click="openCreateDialog">创建 Deployment</el-button>
        <el-button icon="Download" @click="exportCSV">导出</el-button>
        <el-button icon="Refresh" @click="fetchWorkloads">刷新</el-button>
      </div>
    </div>

    <!-- Summary Row -->
    <el-row :gutter="16" class="summary-row">
      <el-col :span="4"><el-card><div class="card-title">总数</div><div class="card-value">{{ stats.total }}</div></el-card></el-col>
      <el-col :span="4"><el-card><div class="card-title">健康</div><div class="card-value">{{ stats.healthy }}</div></el-card></el-col>
      <el-col :span="4"><el-card><div class="card-title">异常</div><div class="card-value">{{ stats.degraded }}</div></el-card></el-col>
      <el-col :span="4"><el-card><div class="card-title">滚动中</div><div class="card-value">{{ stats.rolling }}</div></el-card></el-col>
      <el-col :span="4"><el-card><div class="card-title">已绑定域名</div><div class="card-value">{{ stats.withDomains }}</div></el-card></el-col>
      <el-col :span="4"><el-card><div class="card-title">域名总数</div><div class="card-value">{{ stats.domainCount }}</div></el-card></el-col>
    </el-row>

    <!-- Table -->
    <div class="table-scroll">
      <el-table :fit="true" :data="filteredWorkloads" stripe style="width: 100%" v-loading="loading">
        <el-table-column prop="namespace" label="命名空间" min-width="120" />
        <el-table-column prop="name" label="名称" min-width="160" />
        <el-table-column prop="kind" label="类型" width="120" />
        <el-table-column label="域名解析" min-width="180">
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
            <StatusBadge
              :text="workloadStatusMeta(row).text"
              :type="workloadStatusMeta(row).type"
              :source="workloadStatusMeta(row).source"
              :check-at="workloadStatusMeta(row).checkAt"
              :reason="workloadStatusMeta(row).reason"
            />
          </template>
        </el-table-column>
        <el-table-column label="副本" width="90">
          <template #default="{ row }">{{ row.ready }} / {{ row.replicas }}</template>
        </el-table-column>
        <el-table-column label="滚动进度" width="160">
          <template #default="{ row }">
            <el-progress :percentage="rolloutPercent(row)" :status="rolloutStatus(row)" :stroke-width="10" />
          </template>
        </el-table-column>
        <el-table-column label="镜像" min-width="200">
          <template #default="{ row }">
            <el-tooltip v-for="img in row.images || []" :key="img" :content="img" placement="top">
              <el-tag size="small" class="mr-2 image-tag">{{ img }}</el-tag>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" width="180">
          <template #default="{ row }">
            <div>{{ formatTime(row.created_at) }}</div>
            <div class="text-xs text-gray-400">{{ formatSince(row.created_at) }}</div>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="260" fixed="right">
          <template #default="{ row }">
            <div class="action-wrap">
              <el-button size="small" @click="openDetail(row)">详情</el-button>
              <el-button size="small" type="primary" plain @click="openPods(row)">Pods</el-button>
              <el-button size="small" @click="scaleWorkload(row)" :disabled="row.kind === 'DaemonSet'">扩缩容</el-button>
              
              <el-dropdown trigger="click" @command="(cmd) => handleCommand(cmd, row)">
                <el-button size="small">
                  更多<el-icon class="el-icon--right"><arrow-down /></el-icon>
                </el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item command="domains" :disabled="row.kind !== 'Deployment'">域名解析</el-dropdown-item>
                    <el-dropdown-item command="env" :disabled="row.kind !== 'Deployment'">环境变量</el-dropdown-item>
                    <el-dropdown-item command="image" :disabled="row.kind !== 'Deployment'">镜像发布</el-dropdown-item>
                    <el-dropdown-item command="restart">滚动重启</el-dropdown-item>
                    <el-dropdown-item command="delete" :disabled="row.kind !== 'Deployment'" divided type="danger">删除</el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- Create Deployment Dialog -->
    <el-dialog append-to-body v-model="createVisible" title="创建 Deployment" width="720px">
      <el-form :model="createForm" label-width="120px">
        <el-form-item label="命名空间" required>
          <el-select v-model="createForm.namespace" style="width: 100%">
            <el-option v-for="ns in namespaces" :key="ns.name" :label="ns.name" :value="ns.name" />
          </el-select>
        </el-form-item>
        <el-form-item label="名称" required>
          <el-input v-model="createForm.name" placeholder="例如：nginx-web" />
        </el-form-item>
        <el-form-item label="镜像" required>
          <el-input v-model="createForm.image" placeholder="例如：nginx:1.27" />
        </el-form-item>
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="副本数">
              <el-input-number v-model="createForm.replicas" :min="0" :max="1000" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="容器端口">
              <el-input-number v-model="createForm.container_port" :min="1" :max="65535" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="标签 (k=v)">
          <el-input v-model="createForm.labelsText" type="textarea" :rows="2" placeholder="app=nginx,team=ops" />
        </el-form-item>
        <el-form-item label="环境变量 (k=v)">
          <el-input v-model="createForm.envText" type="textarea" :rows="3" placeholder="TZ=Asia/Shanghai&#10;JAVA_OPTS=-Xms256m -Xmx512m" />
        </el-form-item>
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="CPU Request">
              <el-input v-model="createForm.cpu_request" placeholder="100m" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="Memory Request">
              <el-input v-model="createForm.memory_request" placeholder="128Mi" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="CPU Limit">
              <el-input v-model="createForm.cpu_limit" placeholder="500m" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="Memory Limit">
              <el-input v-model="createForm.memory_limit" placeholder="512Mi" />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" :loading="creating" @click="createDeployment">创建</el-button>
      </template>
    </el-dialog>

    <!-- Ops/运维 Dialog for Deployment -->
    <el-dialog append-to-body v-model="opsVisible" width="980px" :title="opsTitle" destroy-on-close>
      <div v-loading="opsLoading">
        <el-descriptions :column="2" border class="ops-overview-desc">
          <el-descriptions-item label="名称">{{ currentWorkload?.name || '-' }}</el-descriptions-item>
          <el-descriptions-item label="命名空间">{{ currentWorkload?.namespace || '-' }}</el-descriptions-item>
          <el-descriptions-item label="副本">{{ runtime.ready || 0 }} / {{ runtime.replicas || 0 }}</el-descriptions-item>
          <el-descriptions-item label="滚动状态">
            <el-tag :type="runtime.rolling ? 'warning' : 'success'">{{ runtime.rolling ? '滚动中' : '稳定' }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="Ingress">{{ runtime.managed_ingress || '-' }}</el-descriptions-item>
          <el-descriptions-item label="Service">{{ runtime.service_name || '-' }}</el-descriptions-item>
        </el-descriptions>

        <el-tabs v-model="opsTab" class="ops-tabs">
          <el-tab-pane label="概览" name="overview">
            <el-row :gutter="12" class="overview-cards">
              <el-col :span="6"><el-card><div class="card-title">Ready</div><div class="card-value">{{ runtime.ready || 0 }}</div></el-card></el-col>
              <el-col :span="6"><el-card><div class="card-title">Updated</div><div class="card-value">{{ runtime.updated || 0 }}</div></el-card></el-col>
              <el-col :span="6"><el-card><div class="card-title">Available</div><div class="card-value">{{ runtime.available || 0 }}</div></el-card></el-col>
              <el-col :span="6"><el-card><div class="card-title">域名数</div><div class="card-value">{{ (runtime.domains || []).length }}</div></el-card></el-col>
            </el-row>
            <div class="ops-section">
              <div class="section-title">当前域名解析</div>
              <div class="domain-list" v-if="(runtime.domains || []).length">
                <el-link v-for="host in runtime.domains" :key="host" :href="`http://${host}`" target="_blank" type="primary" class="domain-link">
                  {{ host }}
                </el-link>
              </div>
              <div v-else class="text-xs text-gray-400">暂无域名解析</div>
            </div>
            <div class="ops-section">
              <div class="section-title">容器镜像</div>
              <el-table :data="runtime.containers || []" size="small" border>
                <el-table-column prop="name" label="容器" width="180" />
                <el-table-column prop="image" label="镜像" min-width="380" />
                <el-table-column label="环境变量" width="120">
                  <template #default="{ row }">{{ (row.env || []).length }}</template>
                </el-table-column>
              </el-table>
            </div>
          </el-tab-pane>

          <el-tab-pane label="环境变量" name="env">
            <div class="ops-toolbar">
              <el-select v-model="envContainer" class="w-52" placeholder="容器">
                <el-option v-for="ctn in runtime.containers || []" :key="ctn.name" :label="ctn.name" :value="ctn.name" />
              </el-select>
              <el-button @click="addEnvRow">新增变量</el-button>
              <el-button type="primary" :loading="opsSaving" @click="saveEnv">保存并滚动更新</el-button>
            </div>
            <el-table :data="envRows" size="small" border>
              <el-table-column label="变量名" min-width="220">
                <template #default="{ row }">
                  <el-input v-model="row.name" placeholder="例如：JAVA_OPTS" />
                </template>
              </el-table-column>
              <el-table-column label="变量值" min-width="360">
                <template #default="{ row }">
                  <el-input v-model="row.value" placeholder="变量值" />
                </template>
              </el-table-column>
              <el-table-column label="操作" width="90">
                <template #default="{ $index }">
                  <el-button text type="danger" @click="removeEnvRow($index)">删除</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>

          <el-tab-pane label="镜像发布" name="image">
            <el-table :data="imageRows" size="small" border>
              <el-table-column prop="container" label="容器" width="180" />
              <el-table-column prop="currentImage" label="当前镜像" min-width="320" />
              <el-table-column label="新镜像" min-width="320">
                <template #default="{ row }">
                  <el-input v-model="row.nextImage" placeholder="例如：repo/app:v2" />
                </template>
              </el-table-column>
            </el-table>
            <div class="ops-toolbar mt-12">
              <el-button type="primary" :loading="opsSaving" @click="saveImage">发布镜像并滚动更新</el-button>
            </div>
          </el-tab-pane>

          <el-tab-pane label="域名解析" name="domains">
            <div class="ops-toolbar wrap">
              <el-select v-model="domainForm.service_name" class="w-52" placeholder="后端 Service">
                <el-option v-for="svc in runtime.service_candidates || []" :key="svc.name" :label="svc.name" :value="svc.name" />
              </el-select>
              <el-input-number v-model="domainForm.service_port" :min="1" :max="65535" placeholder="端口" />
              <el-input v-model="domainForm.ingress_class" class="w-52" placeholder="IngressClass（可选）" />
              <el-input v-model="domainForm.path" class="w-52" placeholder="路径（默认 /）" />
            </div>
            <el-form label-width="120px">
              <el-form-item label="域名列表">
                <el-input
                  v-model="domainForm.domains_text"
                  type="textarea"
                  :rows="5"
                  placeholder="每行一个域名，例如：\napp.example.com\napi.example.com"
                />
              </el-form-item>
            </el-form>
            <div class="text-xs text-gray-500">保存后会自动创建/更新托管 Ingress，并绑定 to 选中的 Service。</div>
            <div class="ops-toolbar mt-12">
              <el-button type="primary" :loading="opsSaving" @click="saveDomains">保存域名解析</el-button>
            </div>
          </el-tab-pane>
        </el-tabs>
      </div>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { ref, onMounted, computed, reactive, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import StatusBadge from '@/components/common/StatusBadge.vue'
import { getErrorMessage, isCancelError } from '@/utils/error'

const clusters = ref([])
const namespaces = ref([])
const clusterId = ref('')
const namespace = ref('')
const workloads = ref([])
const keyword = ref('')
const kindFilter = ref('')
const loading = ref(false)
const router = useRouter()
const route = useRoute()

// Create dialog state
const createVisible = ref(false)
const creating = ref(false)
const createForm = reactive({
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

// Ops/运维 dialog state
const opsVisible = ref(false)
const opsLoading = ref(false)
const opsSaving = ref(false)
const opsTab = ref('overview')
const currentWorkload = ref(null)
const runtime = ref({
  domains: [],
  managed_domains: [],
  service_candidates: [],
  containers: [],
  rolling: false
})
const envContainer = ref('')
const envRows = ref([])
const imageRows = ref([])
const domainForm = reactive({
  service_name: '',
  service_port: 80,
  ingress_class: '',
  path: '/',
  domains_text: ''
})

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

// Common Helpers
const parseKVText = (text) => {
  const result = {}
  String(text || '').split(/\n|,/).forEach((line) => {
    const item = line.trim()
    if (!item) return
    const idx = item.indexOf('=')
    if (idx <= 0) return
    const key = item.slice(0, idx).trim()
    const value = item.slice(idx + 1)
    if (key) result[key] = value
  })
  return result
}

const parseHostText = (text) => {
  const uniq = new Set()
  String(text || '').split(/\n|,/).forEach((line) => {
    const host = line.trim().toLowerCase()
    if (!host) return
    uniq.add(host)
  })
  return Array.from(uniq)
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

// APIs
const fetchClusters = async () => {
  const res = await axios.get('/api/v1/k8s/clusters', { headers: authHeaders() })
  clusters.value = res.data.data || []
  if (!clusterId.value && clusters.value.length > 0) {
    clusterId.value = clusters.value[0].id
  }
}

const fetchNamespaces = async () => {
  if (!clusterId.value) return
  try {
    const res = await axios.get(`/api/v1/k8s/clusters/${clusterId.value}/namespaces`, { headers: authHeaders() })
    namespaces.value = res.data.data || []
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '获取命名空间失败'))
  }
}

const fetchWorkloads = async () => {
  if (!clusterId.value) return
  loading.value = true
  try {
    const res = await axios.get(`/api/v1/k8s/clusters/${clusterId.value}/workloads`, {
      headers: authHeaders(),
      params: { namespace: namespace.value }
    })
    workloads.value = res.data.data || []
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '获取工作负载失败'))
  } finally {
    loading.value = false
  }
}

// Compute table & stats
const filteredWorkloads = computed(() => {
  const key = keyword.value.trim().toLowerCase()
  return workloads.value.filter(w => {
    if (kindFilter.value && w.kind !== kindFilter.value) return false
    if (!key) return true
    const images = (w.images || []).join(',').toLowerCase()
    const domains = (w.domains || []).join(',').toLowerCase()
    return (w.name || '').toLowerCase().includes(key) || images.includes(key) || domains.includes(key)
  })
})

const stats = computed(() => {
  const val = { total: filteredWorkloads.value.length, healthy: 0, degraded: 0, rolling: 0, replicas: 0, withDomains: 0, domainCount: 0 }
  filteredWorkloads.value.forEach((item) => {
    const replicas = Number(item.replicas || 0)
    const ready = Number(item.ready || 0)
    const available = Number(item.available || 0)
    val.replicas += replicas
    val.domainCount += (item.domains || []).length
    if ((item.domains || []).length) val.withDomains += 1
    if (replicas === 0 || ready < replicas || available < replicas) val.degraded += 1
    else val.healthy += 1
    if (ready < replicas || available < replicas) val.rolling += 1
  })
  return val
})

const workloadStatusMeta = (row) => {
  const replicas = Number(row.replicas || 0)
  const ready = Number(row.ready || 0)
  const available = Number(row.available || 0)
  const source = 'K8s 控制面'
  const checkAt = row.updated_at || row.last_transition_time || row.created_at || ''
  if (replicas === 0) {
    return { text: 'Scaled 0', type: 'info', source, checkAt, reason: '副本数为 0，工作负载已缩容' }
  }
  if (ready >= replicas && available >= replicas) {
    return { text: 'Healthy', type: 'success', source, checkAt, reason: 'Ready/Available 副本均满足期望' }
  }
  if (ready > 0 && available > 0) {
    return { text: 'Progressing', type: 'warning', source, checkAt, reason: '工作负载正在滚动发布中' }
  }
  if (ready > 0) {
    return { text: 'NotAvailable', type: 'warning', source, checkAt, reason: '存在 Ready 副本，但可用副本不足' }
  }
  if (available > 0) {
    return { text: 'PartiallyReady', type: 'warning', source, checkAt, reason: '存在可用副本，但 Ready 状态未达标' }
  }
  return { text: 'Degraded', type: 'warning', source, checkAt, reason: '副本未就绪，工作负载异常' }
}

const rolloutPercent = (row) => {
  const replicas = Number(row.replicas || 0)
  const ready = Number(row.ready || 0)
  if (replicas <= 0) return 100
  return Math.max(0, Math.min(100, Math.round((ready / replicas) * 100)))
}

const rolloutStatus = (row) => {
  const p = rolloutPercent(row)
  if (p === 100) return 'success'
  if (p === 0) return 'exception'
  return ''
}

// Handlers
const handleClusterChange = async () => {
  namespace.value = ''
  await fetchNamespaces()
  await fetchWorkloads()
}

const handleNamespaceChange = async () => {
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

const openPods = (row) => {
  router.push({
    path: '/k8s/pods',
    query: {
      clusterId: clusterId.value,
      namespace: row.namespace,
      ownerKind: row.kind,
      ownerName: row.name
    }
  })
}

// More command dropdown handler
const handleCommand = (cmd, row) => {
  if (cmd === 'domains') {
    openOps(row, 'domains')
  } else if (cmd === 'env') {
    openOps(row, 'env')
  } else if (cmd === 'image') {
    openOps(row, 'image')
  } else if (cmd === 'restart') {
    restartWorkload(row)
  } else if (cmd === 'delete') {
    deleteDeployment(row)
  }
}

// Create Logic
const resetCreateForm = () => {
  createForm.namespace = namespace.value || (namespaces.value[0]?.name || '')
  createForm.name = ''
  createForm.image = ''
  createForm.replicas = 1
  createForm.container_port = 80
  createForm.labelsText = ''
  createForm.envText = ''
  createForm.cpu_request = ''
  createForm.memory_request = ''
  createForm.cpu_limit = ''
  createForm.memory_limit = ''
}

const openCreateDialog = () => {
  resetCreateForm()
  createVisible.value = true
}

const createDeployment = async () => {
  if (!createForm.namespace || !createForm.name.trim() || !createForm.image.trim()) {
    ElMessage.warning('命名空间、名称、镜像为必填')
    return
  }
  creating.value = true
  try {
    const payload = {
      name: createForm.name.trim(),
      image: createForm.image.trim(),
      replicas: Number(createForm.replicas || 1),
      container_port: Number(createForm.container_port || 0),
      labels: parseKVText(createForm.labelsText),
      env: parseKVText(createForm.envText),
      cpu_request: createForm.cpu_request.trim(),
      memory_request: createForm.memory_request.trim(),
      cpu_limit: createForm.cpu_limit.trim(),
      memory_limit: createForm.memory_limit.trim()
    }
    await axios.post(`/api/v1/k8s/clusters/${clusterId.value}/namespaces/${createForm.namespace}/deployments`, payload, {
      headers: authHeaders()
    })
    ElMessage.success('创建成功')
    createVisible.value = false
    namespace.value = createForm.namespace
    await fetchWorkloads()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '创建失败'))
  } finally {
    creating.value = false
  }
}

// Workload operations
const scaleWorkload = async (row) => {
  if (!row || row.kind === 'DaemonSet') return
  try {
    const { value } = await ElMessageBox.prompt('输入副本数', `扩缩容 ${row.name}`, {
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
    await fetchWorkloads()
  } catch (err) {
    if (!isCancelError(err)) ElMessage.error(getErrorMessage(err, '扩缩容失败'))
  }
}

const restartWorkload = async (row) => {
  if (!row) return
  try {
    await ElMessageBox.confirm(`确认重启 ${row.kind} ${row.name} 吗？`, '提示', { type: 'warning' })
    await axios.post(`/api/v1/k8s/clusters/${clusterId.value}/namespaces/${row.namespace}/workloads/${row.kind}/${row.name}/restart`, {}, {
      headers: authHeaders()
    })
    ElMessage.success('已触发滚动重启')
    await fetchWorkloads()
  } catch (err) {
    if (!isCancelError(err)) ElMessage.error(getErrorMessage(err, '重启失败'))
  }
}

const deleteDeployment = async (row) => {
  try {
    await ElMessageBox.confirm(`确认删除 Deployment ${row.name} 吗？此操作不可恢复。`, '提示', { type: 'warning' })
    await axios.delete(`/api/v1/k8s/clusters/${clusterId.value}/namespaces/${row.namespace}/deployments/${row.name}`, {
      headers: authHeaders()
    })
    ElMessage.success('删除成功')
    await fetchWorkloads()
  } catch (err) {
    if (!isCancelError(err)) ElMessage.error(getErrorMessage(err, '删除失败'))
  }
}

// Ops logic (Deployment specialized)
const opsTitle = computed(() => {
  if (!currentWorkload.value) return 'Deployment 运维中心'
  return `Deployment 运维中心 · ${currentWorkload.value.name}`
})

const fetchDeploymentRuntime = async (row) => {
  const res = await axios.get(`/api/v1/k8s/clusters/${clusterId.value}/namespaces/${row.namespace}/deployments/${row.name}/runtime`, {
    headers: authHeaders()
  })
  runtime.value = res.data.data || {
    domains: [],
    managed_domains: [],
    service_candidates: [],
    containers: [],
    rolling: false
  }
  if (runtime.value.containers?.length) {
    envContainer.value = runtime.value.containers[0].name
  } else {
    envContainer.value = ''
  }
  refreshEnvRowsByContainer()
  imageRows.value = (runtime.value.containers || []).map((item) => ({
    container: item.name,
    currentImage: item.image,
    nextImage: item.image
  }))
  domainForm.service_name = runtime.value.service_name || runtime.value.service_candidates?.[0]?.name || ''
  const servicePort = runtime.value.service_candidates?.find((svc) => svc.name === domainForm.service_name)?.ports?.[0]
  domainForm.service_port = Number(servicePort || 80)
  domainForm.ingress_class = runtime.value.ingress_class || ''
  domainForm.path = '/'
  domainForm.domains_text = (runtime.value.managed_domains || runtime.value.domains || []).join('\n')
}

const openOps = async (row, tab = 'overview') => {
  currentWorkload.value = row
  opsTab.value = tab
  opsVisible.value = true
  opsLoading.value = true
  try {
    await fetchDeploymentRuntime(row)
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '加载 Deployment 运行信息失败'))
  } finally {
    opsLoading.value = false
  }
}

const refreshEnvRowsByContainer = () => {
  const ctn = (runtime.value.containers || []).find((item) => item.name === envContainer.value)
  envRows.value = (ctn?.env || []).map((item) => ({ name: item.name, value: item.value }))
}

const addEnvRow = () => {
  envRows.value.push({ name: '', value: '' })
}

const removeEnvRow = (idx) => {
  envRows.value.splice(idx, 1)
}

const saveEnv = async () => {
  if (!currentWorkload.value) return
  if (!envContainer.value) {
    ElMessage.warning('请选择容器')
    return
  }
  const payloadEnv = {}
  for (const item of envRows.value) {
    const key = String(item.name || '').trim()
    if (!key) continue
    payloadEnv[key] = String(item.value ?? '')
  }
  opsSaving.value = true
  try {
    await axios.put(`/api/v1/k8s/clusters/${clusterId.value}/namespaces/${currentWorkload.value.namespace}/deployments/${currentWorkload.value.name}/env`, {
      container: envContainer.value,
      env: payloadEnv
    }, { headers: authHeaders() })
    ElMessage.success('环境变量已更新，滚动发布已触发')
    await fetchDeploymentRuntime(currentWorkload.value)
    await fetchWorkloads()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '保存环境变量失败'))
  } finally {
    opsSaving.value = false
  }
}

const saveImage = async () => {
  if (!currentWorkload.value) return
  const changed = imageRows.value.filter((item) => String(item.nextImage || '').trim() && String(item.nextImage || '').trim() !== item.currentImage)
  if (!changed.length) {
    ElMessage.warning('没有检测到镜像变更')
    return
  }
  opsSaving.value = true
  try {
    for (const item of changed) {
      await axios.put(`/api/v1/k8s/clusters/${clusterId.value}/namespaces/${currentWorkload.value.namespace}/deployments/${currentWorkload.value.name}/image`, {
        container: item.container,
        image: String(item.nextImage || '').trim()
      }, { headers: authHeaders() })
    }
    ElMessage.success(`已发布 ${changed.length} 个容器镜像，滚动更新进行中`)
    await fetchDeploymentRuntime(currentWorkload.value)
    await fetchWorkloads()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '镜像发布失败'))
  } finally {
    opsSaving.value = false
  }
}

const saveDomains = async () => {
  if (!currentWorkload.value) return
  const domains = parseHostText(domainForm.domains_text)
  opsSaving.value = true
  try {
    await axios.put(`/api/v1/k8s/clusters/${clusterId.value}/namespaces/${currentWorkload.value.namespace}/deployments/${currentWorkload.value.name}/domains`, {
      domains,
      service_name: domainForm.service_name,
      service_port: Number(domainForm.service_port || 80),
      ingress_class: String(domainForm.ingress_class || '').trim(),
      path: String(domainForm.path || '/').trim() || '/'
    }, { headers: authHeaders() })
    ElMessage.success('域名解析已更新')
    await fetchDeploymentRuntime(currentWorkload.value)
    await fetchWorkloads()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '更新域名解析失败'))
  } finally {
    opsSaving.value = false
  }
}

const exportCSV = () => {
  const headers = ['namespace','name','kind','replicas','ready','available','images','created_at','status']
  const rows = filteredWorkloads.value.map(w => [
    w.namespace, w.name, w.kind, w.replicas, w.ready, w.available,
    (w.images || []).join('|'), w.created_at, workloadStatusMeta(w).text
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

watch(envContainer, () => {
  refreshEnvRowsByContainer()
})

watch(() => domainForm.service_name, (val) => {
  const servicePort = runtime.value.service_candidates?.find((svc) => svc.name === val)?.ports?.[0]
  if (servicePort) {
    domainForm.service_port = Number(servicePort)
  }
})

onMounted(async () => {
  if (route.query.clusterId) clusterId.value = String(route.query.clusterId)
  if (route.query.namespace) namespace.value = String(route.query.namespace)
  await fetchClusters()
  await fetchNamespaces()
  await fetchWorkloads()
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
.w-40 { width: 160px; }
.mr-2 { margin-right: 6px; margin-bottom: 6px; }
.table-scroll { overflow-x: auto; }
.domain-list { display: flex; flex-wrap: wrap; gap: 6px; }
.domain-link { font-size: 12px; }
.image-tag {
  max-width: 320px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  vertical-align: top;
}
.action-wrap {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
}
.ops-overview-desc { margin-bottom: 16px; }
.ops-toolbar { display: flex; align-items: center; gap: 10px; margin-bottom: 12px; }
.ops-toolbar.wrap { flex-wrap: wrap; }
.ops-tabs { margin-top: 8px; }
.overview-cards { margin-bottom: 12px; }
.ops-section { margin-top: 12px; }
.section-title { font-size: 13px; color: #606266; margin-bottom: 8px; }
.mt-12 { margin-top: 12px; }

@media (max-width: 1600px) {
  .image-tag {
    max-width: 180px;
  }
}
</style>
