<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>ConfigMap / Secret</h2>
        <p class="page-desc">配置与密钥资源管理。</p>
      </div>
      <div class="page-actions">
        <el-select v-model="clusterId" placeholder="选择集群" class="w-52" @change="handleClusterChange">
          <el-option v-for="c in clusters" :key="c.id" :label="c.display_name || c.name" :value="c.id" />
        </el-select>
        <el-select v-model="namespace" placeholder="命名空间" class="w-52" @change="fetchData">
          <el-option label="全部" value="" />
          <el-option v-for="ns in namespaces" :key="ns.name" :label="ns.name" :value="ns.name" />
        </el-select>
        <el-input v-model="keyword" placeholder="搜索名称/Key" class="w-52" clearable />
        <el-button icon="Refresh" @click="fetchData">刷新</el-button>
      </div>
    </div>

    <el-tabs v-model="activeTab">
      <el-tab-pane label="ConfigMaps" name="configmaps">
        <el-row :gutter="16" class="summary-row">
          <el-col :span="6"><el-card><div class="card-title">总数</div><div class="card-value">{{ configStats.total }}</div></el-card></el-col>
          <el-col :span="6"><el-card><div class="card-title">Keys 总数</div><div class="card-value">{{ configStats.keys }}</div></el-card></el-col>
        </el-row>
        <el-table :fit="false" :data="filteredConfigmaps" stripe style="width: 100%">
          <el-table-column prop="namespace" label="命名空间" min-width="140" />
          <el-table-column prop="name" label="名称" min-width="200" />
          <el-table-column label="Keys" min-width="220">
            <template #default="scope">
              <el-tag v-for="k in scope.row.data_keys" :key="k" size="small" class="mr-2">{{ k }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="Key 数" width="100">
            <template #default="scope">
              {{ (scope.row.data_keys || []).length }}
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

      <el-tab-pane label="Secrets" name="secrets">
        <el-row :gutter="16" class="summary-row">
          <el-col :span="6"><el-card><div class="card-title">总数</div><div class="card-value">{{ secretStats.total }}</div></el-card></el-col>
          <el-col :span="6"><el-card><div class="card-title">类型数</div><div class="card-value">{{ secretStats.types }}</div></el-card></el-col>
          <el-col :span="6"><el-card><div class="card-title">Keys 总数</div><div class="card-value">{{ secretStats.keys }}</div></el-card></el-col>
        </el-row>
        <el-table :fit="false" :data="filteredSecrets" stripe style="width: 100%">
          <el-table-column prop="namespace" label="命名空间" min-width="140" />
          <el-table-column prop="name" label="名称" min-width="200" />
          <el-table-column prop="type" label="类型" width="180" />
          <el-table-column label="Keys" min-width="220">
            <template #default="scope">
              <el-tag v-for="k in scope.row.data_keys" :key="k" size="small" class="mr-2">{{ k }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="Key 数" width="100">
            <template #default="scope">
              {{ (scope.row.data_keys || []).length }}
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
const activeTab = ref('configmaps')
const configmaps = ref([])
const secrets = ref([])
const keyword = ref('')

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
  const [cmRes, secRes] = await Promise.all([
    axios.get(`/api/v1/k8s/clusters/${clusterId.value}/configmaps`, { headers: authHeaders(), params }),
    axios.get(`/api/v1/k8s/clusters/${clusterId.value}/secrets`, { headers: authHeaders(), params })
  ])
  configmaps.value = cmRes.data.data || []
  secrets.value = secRes.data.data || []
}

const handleClusterChange = async () => {
  await fetchNamespaces()
  namespace.value = ''
  await fetchData()
}

const filteredConfigmaps = computed(() => {
  const key = keyword.value.trim().toLowerCase()
  return configmaps.value.filter((c) => {
    if (!key) return true
    const keys = (c.data_keys || []).join(',')
    const hay = `${c.name || ''} ${keys}`.toLowerCase()
    return hay.includes(key)
  })
})

const filteredSecrets = computed(() => {
  const key = keyword.value.trim().toLowerCase()
  return secrets.value.filter((s) => {
    if (!key) return true
    const keys = (s.data_keys || []).join(',')
    const hay = `${s.name || ''} ${keys}`.toLowerCase()
    return hay.includes(key)
  })
})

const configStats = computed(() => {
  const total = filteredConfigmaps.value.length
  const keys = filteredConfigmaps.value.reduce((sum, c) => sum + (c.data_keys || []).length, 0)
  return { total, keys }
})

const secretStats = computed(() => {
  const total = filteredSecrets.value.length
  const types = new Set(filteredSecrets.value.map(s => s.type).filter(Boolean)).size
  const keys = filteredSecrets.value.reduce((sum, s) => sum + (s.data_keys || []).length, 0)
  return { total, types, keys }
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
.w-52 { width: 220px; }
.mr-2 { margin-right: 6px; margin-bottom: 6px; }
</style>
