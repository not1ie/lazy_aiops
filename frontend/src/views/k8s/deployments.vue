<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>Deployments</h2>
        <p class="page-desc">Deployment 统一运维台：域名解析、环境变量、镜像发布、滚动状态一屏管理。</p>
      </div>
      <div class="page-actions">
        <el-select v-model="clusterId" placeholder="选择集群" class="w-52" @change="handleClusterChange">
          <el-option v-for="c in clusters" :key="c.id" :label="c.display_name || c.name" :value="c.id" />
        </el-select>
        <el-select v-model="namespace" placeholder="命名空间" class="w-52" @change="fetchDeployments">
          <el-option label="全部" value="" />
          <el-option v-for="ns in namespaces" :key="ns.name" :label="ns.name" :value="ns.name" />
        </el-select>
        <el-input v-model="keyword" placeholder="搜索 名称/镜像/域名" class="w-52" clearable />
        <el-button type="primary" icon="Plus" :disabled="!clusterId || !namespace" @click="openCreateDialog">创建 Deployment</el-button>
        <el-button icon="Refresh" @click="fetchDeployments">刷新</el-button>
      </div>
    </div>

    <el-row :gutter="16" class="summary-row">
      <el-col :span="4"><el-card><div class="card-title">总数</div><div class="card-value">{{ stats.total }}</div></el-card></el-col>
      <el-col :span="4"><el-card><div class="card-title">健康</div><div class="card-value">{{ stats.healthy }}</div></el-card></el-col>
      <el-col :span="4"><el-card><div class="card-title">异常</div><div class="card-value">{{ stats.degraded }}</div></el-card></el-col>
      <el-col :span="4"><el-card><div class="card-title">滚动中</div><div class="card-value">{{ stats.rolling }}</div></el-card></el-col>
      <el-col :span="4"><el-card><div class="card-title">已绑定域名</div><div class="card-value">{{ stats.withDomains }}</div></el-card></el-col>
      <el-col :span="4"><el-card><div class="card-title">域名总数</div><div class="card-value">{{ stats.domainCount }}</div></el-card></el-col>
    </el-row>

    <div class="table-scroll">
      <el-table :fit="true" :data="filteredDeployments" stripe v-loading="loading" style="width: 100%; min-width: 1880px">
        <el-table-column prop="namespace" label="命名空间" min-width="130" />
        <el-table-column prop="name" label="名称" min-width="180" />
        <el-table-column label="域名解析" min-width="230">
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
        <el-table-column label="滚动进度" width="190">
          <template #default="{ row }">
            <el-progress :percentage="rolloutPercent(row)" :status="rolloutStatus(row)" :stroke-width="10" />
          </template>
        </el-table-column>
        <el-table-column label="镜像" min-width="300">
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
        <el-table-column label="操作" width="640" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="openDetail(row)">详情</el-button>
            <el-button size="small" type="primary" plain @click="openPods(row)">Pods</el-button>
            <el-button size="small" type="success" plain @click="openOps(row, 'domains')">域名</el-button>
            <el-button size="small" type="warning" plain @click="openOps(row, 'env')">环境变量</el-button>
            <el-button size="small" type="info" plain @click="openOps(row, 'image')">镜像</el-button>
            <el-button size="small" @click="scaleDeployment(row)">扩缩容</el-button>
            <el-button size="small" type="warning" @click="restartDeployment(row)">重启</el-button>
            <el-button size="small" type="danger" plain @click="deleteDeployment(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

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

    <el-dialog append-to-body v-model="opsVisible" width="980px" :title="opsTitle" destroy-on-close>
      <div v-loading="opsLoading">
        <el-descriptions :column="2" border class="ops-overview-desc">
          <el-descriptions-item label="名称">{{ currentDeployment?.name || '-' }}</el-descriptions-item>
          <el-descriptions-item label="命名空间">{{ currentDeployment?.namespace || '-' }}</el-descriptions-item>
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
            <div class="text-xs text-gray-500">保存后会自动创建/更新托管 Ingress，并绑定到选中的 Service。</div>
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
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getErrorMessage, isCancelError } from '@/utils/error'

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

const opsVisible = ref(false)
const opsLoading = ref(false)
const opsSaving = ref(false)
const opsTab = ref('overview')
const currentDeployment = ref(null)
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

const fetchClusters = async () => {
  const res = await axios.get('/api/v1/k8s/clusters', { headers: authHeaders() })
  clusters.value = res.data.data || []
  if (!clusterId.value && clusters.value.length) {
    clusterId.value = clusters.value[0].id
  }
}

const fetchNamespaces = async () => {
  if (!clusterId.value) return
  try {
    const res = await axios.get(`/api/v1/k8s/clusters/${clusterId.value}/namespaces`, { headers: authHeaders() })
    namespaces.value = res.data.data || []
    if (!namespace.value && namespaces.value.length) {
      namespace.value = namespaces.value[0].name
    }
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '获取命名空间失败'))
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
    ElMessage.error(getErrorMessage(err, '获取 Deployment 失败'))
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
  const val = { total: filteredDeployments.value.length, healthy: 0, degraded: 0, rolling: 0, replicas: 0, withDomains: 0, domainCount: 0 }
  filteredDeployments.value.forEach((item) => {
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

const statusText = (row) => {
  const replicas = Number(row.replicas || 0)
  const ready = Number(row.ready || 0)
  const available = Number(row.available || 0)
  if (replicas === 0) return 'Scaled 0'
  if (ready >= replicas && available >= replicas) return 'Healthy'
  if (ready > 0 && available > 0) return 'Progressing'
  if (ready > 0) return 'NotAvailable'
  if (available > 0) return 'PartiallyReady'
  return 'Degraded'
}

const statusType = (row) => {
  const text = statusText(row)
  if (text === 'Healthy') return 'success'
  if (text === 'Scaled 0') return 'info'
  if (text === 'Progressing' || text === 'NotAvailable' || text === 'PartiallyReady') return 'warning'
  return 'warning'
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
  currentDeployment.value = row
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
  if (!currentDeployment.value) return
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
    await axios.put(`/api/v1/k8s/clusters/${clusterId.value}/namespaces/${currentDeployment.value.namespace}/deployments/${currentDeployment.value.name}/env`, {
      container: envContainer.value,
      env: payloadEnv
    }, { headers: authHeaders() })
    ElMessage.success('环境变量已更新，滚动发布已触发')
    await fetchDeploymentRuntime(currentDeployment.value)
    await fetchDeployments()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '保存环境变量失败'))
  } finally {
    opsSaving.value = false
  }
}

const saveImage = async () => {
  if (!currentDeployment.value) return
  const changed = imageRows.value.filter((item) => String(item.nextImage || '').trim() && String(item.nextImage || '').trim() !== item.currentImage)
  if (!changed.length) {
    ElMessage.warning('没有检测到镜像变更')
    return
  }
  opsSaving.value = true
  try {
    for (const item of changed) {
      await axios.put(`/api/v1/k8s/clusters/${clusterId.value}/namespaces/${currentDeployment.value.namespace}/deployments/${currentDeployment.value.name}/image`, {
        container: item.container,
        image: String(item.nextImage || '').trim()
      }, { headers: authHeaders() })
    }
    ElMessage.success(`已发布 ${changed.length} 个容器镜像，滚动更新进行中`)
    await fetchDeploymentRuntime(currentDeployment.value)
    await fetchDeployments()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '镜像发布失败'))
  } finally {
    opsSaving.value = false
  }
}

const saveDomains = async () => {
  if (!currentDeployment.value) return
  const domains = parseHostText(domainForm.domains_text)
  opsSaving.value = true
  try {
    await axios.put(`/api/v1/k8s/clusters/${clusterId.value}/namespaces/${currentDeployment.value.namespace}/deployments/${currentDeployment.value.name}/domains`, {
      domains,
      service_name: domainForm.service_name,
      service_port: Number(domainForm.service_port || 80),
      ingress_class: String(domainForm.ingress_class || '').trim(),
      path: String(domainForm.path || '/').trim() || '/'
    }, { headers: authHeaders() })
    ElMessage.success('域名解析已更新')
    await fetchDeploymentRuntime(currentDeployment.value)
    await fetchDeployments()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '更新域名解析失败'))
  } finally {
    opsSaving.value = false
  }
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
    if (!isCancelError(err)) ElMessage.error(getErrorMessage(err, '扩缩容失败'))
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
    if (!isCancelError(err)) ElMessage.error(getErrorMessage(err, '重启失败'))
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
    if (!isCancelError(err)) ElMessage.error(getErrorMessage(err, '删除失败'))
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
    ElMessage.error(getErrorMessage(err, '创建失败'))
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

const opsTitle = computed(() => {
  if (!currentDeployment.value) return 'Deployment 运维中心'
  return `Deployment 运维中心 · ${currentDeployment.value.name}`
})

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
.table-scroll { overflow-x: auto; }
.mr-2 { margin-right: 6px; margin-bottom: 6px; }
.mt-12 { margin-top: 12px; }
.domain-list { display: flex; flex-wrap: wrap; gap: 6px; }
.domain-link { font-size: 12px; }
.ops-overview-desc { margin-bottom: 16px; }
.ops-toolbar { display: flex; align-items: center; gap: 10px; margin-bottom: 12px; }
.ops-toolbar.wrap { flex-wrap: wrap; }
.ops-tabs { margin-top: 8px; }
.overview-cards { margin-bottom: 12px; }
.ops-section { margin-top: 12px; }
.section-title { font-size: 13px; color: #606266; margin-bottom: 8px; }
</style>
