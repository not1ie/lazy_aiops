<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>事件与诊断</h2>
        <p class="page-desc">集群事件与排障信息。</p>
      </div>
      <div class="page-actions">
        <el-select v-model="clusterId" placeholder="选择集群" class="w-52" @change="handleClusterChange">
          <el-option v-for="c in clusters" :key="c.id" :label="c.display_name || c.name" :value="c.id" />
        </el-select>
        <el-select v-model="namespace" placeholder="命名空间" class="w-52" @change="fetchEvents">
          <el-option label="全部" value="" />
          <el-option v-for="ns in namespaces" :key="ns.name" :label="ns.name" :value="ns.name" />
        </el-select>
        <el-select v-model="typeFilter" placeholder="类型" class="w-40" clearable>
          <el-option label="Normal" value="Normal" />
          <el-option label="Warning" value="Warning" />
        </el-select>
        <el-input v-model="keyword" placeholder="搜索 reason/message/对象" class="w-52" clearable />
        <el-button icon="Refresh" @click="fetchEvents">刷新</el-button>
      </div>
    </div>

    <el-row :gutter="16" class="summary-row">
      <el-col :span="6"><el-card><div class="card-title">总数</div><div class="card-value">{{ eventStats.total }}</div></el-card></el-col>
      <el-col :span="6"><el-card><div class="card-title">Warning</div><div class="card-value">{{ eventStats.warning }}</div></el-card></el-col>
      <el-col :span="6"><el-card><div class="card-title">Normal</div><div class="card-value">{{ eventStats.normal }}</div></el-card></el-col>
    </el-row>

    <el-table :data="filteredEvents" stripe style="width: 100%">
      <el-table-column label="类型" width="100">
        <template #default="scope">
          <el-tag :type="scope.row.type === 'Warning' ? 'danger' : 'success'">{{ scope.row.type }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="reason" label="原因" width="160" />
      <el-table-column prop="message" label="信息" min-width="260" />
      <el-table-column prop="involved_object" label="对象" min-width="200" />
      <el-table-column prop="count" label="次数" width="80" />
      <el-table-column label="最近时间" min-width="180">
        <template #default="scope">
          <div>{{ formatTime(scope.row.last_seen) }}</div>
          <div class="text-xs text-gray-400">{{ formatSince(scope.row.last_seen) }}</div>
        </template>
      </el-table-column>
    </el-table>
  </el-card>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import axios from 'axios'

const clusters = ref([])
const namespaces = ref([])
const clusterId = ref('')
const namespace = ref('')
const events = ref([])
const typeFilter = ref('')
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

const fetchEvents = async () => {
  if (!clusterId.value) return
  const res = await axios.get(`/api/v1/k8s/clusters/${clusterId.value}/events`, {
    headers: authHeaders(),
    params: { namespace: namespace.value || '' }
  })
  events.value = res.data.data || []
}

const handleClusterChange = async () => {
  await fetchNamespaces()
  namespace.value = ''
  await fetchEvents()
}

const filteredEvents = computed(() => {
  const key = keyword.value.trim().toLowerCase()
  return events.value.filter((e) => {
    if (typeFilter.value && e.type !== typeFilter.value) return false
    if (!key) return true
    const hay = `${e.reason || ''} ${e.message || ''} ${e.involved_object || ''}`.toLowerCase()
    return hay.includes(key)
  })
})

const eventStats = computed(() => {
  const stats = { total: filteredEvents.value.length, warning: 0, normal: 0 }
  filteredEvents.value.forEach((e) => {
    if (e.type === 'Warning') stats.warning += 1
    if (e.type === 'Normal') stats.normal += 1
  })
  return stats
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
  await fetchEvents()
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
</style>
