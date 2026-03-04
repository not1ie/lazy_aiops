<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>节点管理</h2>
        <p class="page-desc">查看集群节点状态与资源信息。</p>
      </div>
      <div class="page-actions">
        <el-select v-model="clusterId" placeholder="选择集群" class="w-64" @change="fetchNodes">
          <el-option v-for="c in clusters" :key="c.id" :label="c.display_name || c.name" :value="c.id" />
        </el-select>
        <el-button icon="Refresh" @click="fetchNodes">刷新</el-button>
      </div>
    </div>

    <el-table :fit="true" :data="nodes" stripe style="width: 100%">
      <el-table-column prop="name" label="节点" min-width="200" />
      <el-table-column prop="status" label="状态" width="120" />
      <el-table-column label="角色" min-width="140">
        <template #default="scope">
          <el-tag v-for="r in scope.row.roles" :key="r" size="small" class="mr-2">{{ r }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="internal_ip" label="内网IP" min-width="140" />
      <el-table-column prop="os" label="OS" min-width="160" />
      <el-table-column prop="kubelet_version" label="Kubelet" min-width="140" />
      <el-table-column prop="cpu" label="CPU" width="100" />
      <el-table-column prop="memory" label="内存" width="120" />
      <el-table-column prop="creation_time" label="创建时间" min-width="180" />
    </el-table>
  </el-card>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import axios from 'axios'

const clusters = ref([])
const clusterId = ref('')
const nodes = ref([])

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const fetchClusters = async () => {
  const res = await axios.get('/api/v1/k8s/clusters', { headers: authHeaders() })
  clusters.value = res.data.data || []
  if (!clusterId.value && clusters.value.length > 0) {
    clusterId.value = clusters.value[0].id
  }
}

const fetchNodes = async () => {
  if (!clusterId.value) return
  const res = await axios.get(`/api/v1/k8s/clusters/${clusterId.value}/nodes`, { headers: authHeaders() })
  nodes.value = res.data.data || []
}

onMounted(async () => {
  await fetchClusters()
  await fetchNodes()
})
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 16px; }
.page-desc { color: #606266; margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; }
.w-64 { width: 260px; }
.mr-2 { margin-right: 6px; margin-bottom: 6px; }
</style>
