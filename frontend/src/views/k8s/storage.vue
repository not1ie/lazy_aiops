<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>存储管理</h2>
        <p class="page-desc">StorageClass / PV / PVC 管理。</p>
      </div>
      <div class="page-actions">
        <el-select v-model="clusterId" placeholder="选择集群" class="w-52" @change="handleClusterChange">
          <el-option v-for="c in clusters" :key="c.id" :label="c.display_name || c.name" :value="c.id" />
        </el-select>
        <el-select v-model="namespace" placeholder="命名空间" class="w-52" @change="fetchData">
          <el-option label="全部" value="" />
          <el-option v-for="ns in namespaces" :key="ns.name" :label="ns.name" :value="ns.name" />
        </el-select>
        <el-input v-model="keyword" placeholder="搜索名称/StorageClass/Claim" class="w-52" clearable />
        <el-button icon="Refresh" @click="fetchData">刷新</el-button>
      </div>
    </div>

    <el-tabs v-model="activeTab">
      <el-tab-pane label="StorageClass" name="sc">
        <el-row :gutter="16" class="summary-row">
          <el-col :span="6"><el-card><div class="card-title">StorageClass</div><div class="card-value">{{ storageStats.sc }}</div></el-card></el-col>
        </el-row>
        <el-table :data="filteredStorageClasses" stripe style="width: 100%">
          <el-table-column prop="name" label="名称" min-width="180" />
          <el-table-column prop="provisioner" label="Provisioner" min-width="220" />
          <el-table-column prop="reclaim_policy" label="回收策略" width="120" />
          <el-table-column prop="volume_binding" label="绑定模式" width="120" />
          <el-table-column prop="allow_expansion" label="扩容" width="100">
            <template #default="scope">
              <el-tag :type="scope.row.allow_expansion ? 'success' : 'info'">
                {{ scope.row.allow_expansion ? '允许' : '不允许' }}
              </el-tag>
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

      <el-tab-pane label="PersistentVolume" name="pv">
        <el-row :gutter="16" class="summary-row">
          <el-col :span="6"><el-card><div class="card-title">PV 总数</div><div class="card-value">{{ storageStats.pv }}</div></el-card></el-col>
          <el-col :span="6"><el-card><div class="card-title">Bound</div><div class="card-value">{{ storageStats.pvBound }}</div></el-card></el-col>
        </el-row>
        <div class="tab-filters">
          <el-select v-model="pvStatusFilter" placeholder="状态" class="w-40" clearable>
            <el-option label="Bound" value="Bound" />
            <el-option label="Available" value="Available" />
            <el-option label="Released" value="Released" />
            <el-option label="Failed" value="Failed" />
          </el-select>
        </div>
        <el-table :data="filteredPvs" stripe style="width: 100%">
          <el-table-column prop="name" label="名称" min-width="180" />
          <el-table-column prop="capacity" label="容量" width="120" />
          <el-table-column label="访问模式" min-width="160">
            <template #default="scope">
              <el-tag v-for="m in scope.row.access_modes" :key="m" size="small" class="mr-2">{{ m }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="status" label="状态" width="120" />
          <el-table-column prop="storage_class" label="StorageClass" min-width="160" />
          <el-table-column prop="claim" label="Claim" min-width="180" />
          <el-table-column label="创建时间" min-width="180">
            <template #default="scope">
              <div>{{ formatTime(scope.row.created_at) }}</div>
              <div class="text-xs text-gray-400">{{ formatSince(scope.row.created_at) }}</div>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="PersistentVolumeClaim" name="pvc">
        <el-row :gutter="16" class="summary-row">
          <el-col :span="6"><el-card><div class="card-title">PVC 总数</div><div class="card-value">{{ storageStats.pvc }}</div></el-card></el-col>
          <el-col :span="6"><el-card><div class="card-title">Bound</div><div class="card-value">{{ storageStats.pvcBound }}</div></el-card></el-col>
        </el-row>
        <div class="tab-filters">
          <el-select v-model="pvcStatusFilter" placeholder="状态" class="w-40" clearable>
            <el-option label="Bound" value="Bound" />
            <el-option label="Pending" value="Pending" />
            <el-option label="Lost" value="Lost" />
          </el-select>
        </div>
        <el-table :data="filteredPvcs" stripe style="width: 100%">
          <el-table-column prop="namespace" label="命名空间" min-width="140" />
          <el-table-column prop="name" label="名称" min-width="180" />
          <el-table-column prop="capacity" label="容量" width="120" />
          <el-table-column label="访问模式" min-width="160">
            <template #default="scope">
              <el-tag v-for="m in scope.row.access_modes" :key="m" size="small" class="mr-2">{{ m }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="status" label="状态" width="120" />
          <el-table-column prop="storage_class" label="StorageClass" min-width="160" />
          <el-table-column prop="volume_name" label="Volume" min-width="180" />
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
const activeTab = ref('sc')
const storageClasses = ref([])
const pvs = ref([])
const pvcs = ref([])
const keyword = ref('')
const pvStatusFilter = ref('')
const pvcStatusFilter = ref('')

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
  const [scRes, pvRes] = await Promise.all([
    axios.get(`/api/v1/k8s/clusters/${clusterId.value}/storageclasses`, { headers: authHeaders() }),
    axios.get(`/api/v1/k8s/clusters/${clusterId.value}/persistentvolumes`, { headers: authHeaders() })
  ])
  storageClasses.value = scRes.data.data || []
  pvs.value = pvRes.data.data || []

  if (namespace.value) {
    const pvcRes = await axios.get(`/api/v1/k8s/clusters/${clusterId.value}/namespaces/${namespace.value}/persistentvolumeclaims`, { headers: authHeaders() })
    pvcs.value = pvcRes.data.data || []
  } else {
    pvcs.value = []
  }
}

const handleClusterChange = async () => {
  await fetchNamespaces()
  namespace.value = ''
  await fetchData()
}

const filteredStorageClasses = computed(() => {
  const key = keyword.value.trim().toLowerCase()
  return storageClasses.value.filter((s) => {
    if (!key) return true
    const hay = `${s.name || ''} ${s.provisioner || ''}`.toLowerCase()
    return hay.includes(key)
  })
})

const filteredPvs = computed(() => {
  const key = keyword.value.trim().toLowerCase()
  return pvs.value.filter((p) => {
    if (pvStatusFilter.value && p.status !== pvStatusFilter.value) return false
    if (!key) return true
    const hay = `${p.name || ''} ${p.storage_class || ''} ${p.claim || ''}`.toLowerCase()
    return hay.includes(key)
  })
})

const filteredPvcs = computed(() => {
  const key = keyword.value.trim().toLowerCase()
  return pvcs.value.filter((p) => {
    if (pvcStatusFilter.value && p.status !== pvcStatusFilter.value) return false
    if (!key) return true
    const hay = `${p.name || ''} ${p.storage_class || ''} ${p.volume_name || ''}`.toLowerCase()
    return hay.includes(key)
  })
})

const storageStats = computed(() => {
  const sc = filteredStorageClasses.value.length
  const pv = filteredPvs.value.length
  const pvc = filteredPvcs.value.length
  const pvBound = filteredPvs.value.filter(p => p.status === 'Bound').length
  const pvcBound = filteredPvcs.value.filter(p => p.status === 'Bound').length
  return { sc, pv, pvc, pvBound, pvcBound }
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
.w-40 { width: 160px; }
.mr-2 { margin-right: 6px; margin-bottom: 6px; }
</style>
