<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>命名空间</h2>
        <p class="page-desc">查看各集群命名空间与状态。</p>
      </div>
      <div class="page-actions">
        <el-select v-model="clusterId" placeholder="选择集群" class="w-64" @change="fetchNamespaces">
          <el-option v-for="c in clusters" :key="c.id" :label="c.display_name || c.name" :value="c.id" />
        </el-select>
        <el-button icon="Refresh" @click="fetchNamespaces">刷新</el-button>
      </div>
    </div>

    <el-table :data="namespaces" stripe style="width: 100%">
      <el-table-column prop="name" label="名称" min-width="180" />
      <el-table-column prop="status" label="状态" width="120" />
      <el-table-column label="标签" min-width="220">
        <template #default="scope">
          <el-tag v-for="(v, k) in scope.row.labels" :key="k" size="small" class="mr-2">{{ k }}={{ v }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="created_at" label="创建时间" min-width="180" />
    </el-table>
  </el-card>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import axios from 'axios'

const clusters = ref([])
const clusterId = ref('')
const namespaces = ref([])

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

onMounted(async () => {
  await fetchClusters()
  await fetchNamespaces()
})
</script>

<style scoped>
.page-card { max-width: 1180px; margin: 0 auto; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 16px; }
.page-desc { color: #606266; margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; }
.w-64 { width: 260px; }
.mr-2 { margin-right: 6px; margin-bottom: 6px; }
</style>
