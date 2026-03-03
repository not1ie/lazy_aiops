<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>服务与 Ingress</h2>
        <p class="page-desc">Service 与 Ingress 资源管理。</p>
      </div>
      <div class="page-actions">
        <el-select v-model="clusterId" placeholder="选择集群" class="w-52" @change="handleClusterChange">
          <el-option v-for="c in clusters" :key="c.id" :label="c.display_name || c.name" :value="c.id" />
        </el-select>
        <el-select v-model="namespace" placeholder="命名空间" class="w-52" @change="fetchData">
          <el-option label="全部" value="" />
          <el-option v-for="ns in namespaces" :key="ns.name" :label="ns.name" :value="ns.name" />
        </el-select>
        <el-input v-model="keyword" placeholder="搜索名称/端口/Host" class="w-52" clearable />
        <el-button icon="Refresh" @click="fetchData">刷新</el-button>
      </div>
    </div>

    <el-tabs v-model="activeTab">
      <el-tab-pane label="Services" name="services">
        <el-row :gutter="16" class="summary-row">
          <el-col :span="4"><el-card><div class="card-title">总数</div><div class="card-value">{{ serviceStats.total }}</div></el-card></el-col>
          <el-col :span="4"><el-card><div class="card-title">ClusterIP</div><div class="card-value">{{ serviceStats.clusterip }}</div></el-card></el-col>
          <el-col :span="4"><el-card><div class="card-title">NodePort</div><div class="card-value">{{ serviceStats.nodeport }}</div></el-card></el-col>
          <el-col :span="4"><el-card><div class="card-title">LoadBalancer</div><div class="card-value">{{ serviceStats.loadbalancer }}</div></el-card></el-col>
          <el-col :span="4"><el-card><div class="card-title">ExternalName</div><div class="card-value">{{ serviceStats.externalname }}</div></el-card></el-col>
        </el-row>
        <div class="tab-filters">
          <el-select v-model="serviceTypeFilter" placeholder="类型" class="w-40" clearable>
            <el-option label="ClusterIP" value="ClusterIP" />
            <el-option label="NodePort" value="NodePort" />
            <el-option label="LoadBalancer" value="LoadBalancer" />
            <el-option label="ExternalName" value="ExternalName" />
          </el-select>
        </div>
        <el-table :fit="false" :data="filteredServices" stripe style="width: 100%">
          <el-table-column prop="namespace" label="命名空间" min-width="140" />
          <el-table-column prop="name" label="名称" min-width="200" />
          <el-table-column prop="type" label="类型" width="120" />
          <el-table-column prop="cluster_ip" label="ClusterIP" min-width="140" />
          <el-table-column label="端口" min-width="160">
            <template #default="scope">
              <el-tag v-for="p in scope.row.ports" :key="p" size="small" class="mr-2">{{ p }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="Selector" min-width="200">
            <template #default="scope">
              <el-tag v-for="(v, k) in scope.row.selector" :key="k" size="small" class="mr-2">{{ k }}={{ v }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="创建时间" min-width="180">
            <template #default="scope">
              <div>{{ formatTime(scope.row.created_at) }}</div>
              <div class="text-xs text-gray-400">{{ formatSince(scope.row.created_at) }}</div>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="Ingresses" name="ingresses">
        <el-row :gutter="16" class="summary-row">
          <el-col :span="6"><el-card><div class="card-title">总数</div><div class="card-value">{{ ingressStats.total }}</div></el-card></el-col>
          <el-col :span="6"><el-card><div class="card-title">带 Host</div><div class="card-value">{{ ingressStats.withHosts }}</div></el-card></el-col>
          <el-col :span="6"><el-card><div class="card-title">Class 数</div><div class="card-value">{{ ingressStats.classes }}</div></el-card></el-col>
        </el-row>
        <el-table :fit="false" :data="filteredIngresses" stripe style="width: 100%">
          <el-table-column prop="namespace" label="命名空间" min-width="140" />
          <el-table-column prop="name" label="名称" min-width="200" />
          <el-table-column prop="class_name" label="Class" width="160" />
          <el-table-column label="Hosts" min-width="220">
            <template #default="scope">
              <el-tag v-for="h in scope.row.hosts" :key="h" size="small" class="mr-2">{{ h }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="创建时间" min-width="180">
            <template #default="scope">
              <div>{{ formatTime(scope.row.created_at) }}</div>
              <div class="text-xs text-gray-400">{{ formatSince(scope.row.created_at) }}</div>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
    </el-tabs>
  </el-card>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import axios from 'axios'

const clusters = ref([])
const namespaces = ref([])
const clusterId = ref('')
const namespace = ref('')
const activeTab = ref('services')
const services = ref([])
const ingresses = ref([])
const keyword = ref('')
const serviceTypeFilter = ref('')

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

const fetchData = async () => {
  if (!clusterId.value) return
  const params = { namespace: namespace.value || '' }
  const [svcRes, ingRes] = await Promise.all([
    axios.get(`/api/v1/k8s/clusters/${clusterId.value}/services`, { headers: authHeaders(), params }),
    axios.get(`/api/v1/k8s/clusters/${clusterId.value}/ingresses`, { headers: authHeaders(), params })
  ])
  services.value = svcRes.data.data || []
  ingresses.value = ingRes.data.data || []
}

const handleClusterChange = async () => {
  await fetchNamespaces()
  namespace.value = ''
  await fetchData()
}

const filteredServices = computed(() => {
  const key = keyword.value.trim().toLowerCase()
  return services.value.filter((s) => {
    if (serviceTypeFilter.value && s.type !== serviceTypeFilter.value) return false
    if (!key) return true
    const ports = (s.ports || []).join(',')
    const hay = `${s.name || ''} ${ports}`.toLowerCase()
    return hay.includes(key)
  })
})

const filteredIngresses = computed(() => {
  const key = keyword.value.trim().toLowerCase()
  return ingresses.value.filter((i) => {
    if (!key) return true
    const hosts = (i.hosts || []).join(',')
    const hay = `${i.name || ''} ${hosts}`.toLowerCase()
    return hay.includes(key)
  })
})

const serviceStats = computed(() => {
  const stats = { total: filteredServices.value.length, clusterip: 0, nodeport: 0, loadbalancer: 0, externalname: 0 }
  filteredServices.value.forEach((s) => {
    const t = (s.type || '').toLowerCase()
    if (t === 'clusterip') stats.clusterip += 1
    if (t === 'nodeport') stats.nodeport += 1
    if (t === 'loadbalancer') stats.loadbalancer += 1
    if (t === 'externalname') stats.externalname += 1
  })
  return stats
})

const ingressStats = computed(() => {
  const classes = new Set()
  let withHosts = 0
  filteredIngresses.value.forEach((i) => {
    if (i.class_name) classes.add(i.class_name)
    if (i.hosts && i.hosts.length > 0) withHosts += 1
  })
  return { total: filteredIngresses.value.length, withHosts, classes: classes.size }
})

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
  await fetchData()
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
.tab-filters { margin-bottom: 10px; display: flex; gap: 8px; }
.w-52 { width: 220px; }
.mr-2 { margin-right: 6px; margin-bottom: 6px; }
.w-40 { width: 160px; }
</style>
