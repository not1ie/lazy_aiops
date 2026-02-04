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
        <el-button icon="Refresh" @click="fetchEvents">刷新</el-button>
      </div>
    </div>

    <el-table :data="events" stripe style="width: 100%">
      <el-table-column prop="type" label="类型" width="100" />
      <el-table-column prop="reason" label="原因" width="160" />
      <el-table-column prop="message" label="信息" min-width="260" />
      <el-table-column prop="involved_object" label="对象" min-width="200" />
      <el-table-column prop="count" label="次数" width="80" />
      <el-table-column prop="last_seen" label="最近时间" min-width="180" />
    </el-table>
  </el-card>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import axios from 'axios'

const clusters = ref([])
const namespaces = ref([])
const clusterId = ref('')
const namespace = ref('')
const events = ref([])

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
    params: { namespace }
  })
  events.value = res.data.data || []
}

const handleClusterChange = async () => {
  await fetchNamespaces()
  namespace.value = ''
  await fetchEvents()
}

onMounted(async () => {
  await fetchClusters()
  await fetchNamespaces()
  await fetchEvents()
})
</script>

<style scoped>
.page-card { max-width: 1180px; margin: 0 auto; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 16px; }
.page-desc { color: #606266; margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; }
.w-52 { width: 220px; }
</style>
