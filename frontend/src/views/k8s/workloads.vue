<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>工作负载</h2>
        <p class="page-desc">Deployment/StatefulSet/DaemonSet 等工作负载。</p>
      </div>
      <div class="page-actions">
        <el-select v-model="clusterId" placeholder="选择集群" class="w-52" @change="handleClusterChange">
          <el-option v-for="c in clusters" :key="c.id" :label="c.display_name || c.name" :value="c.id" />
        </el-select>
        <el-select v-model="namespace" placeholder="命名空间" class="w-52" @change="fetchWorkloads">
          <el-option label="全部" value="" />
          <el-option v-for="ns in namespaces" :key="ns.name" :label="ns.name" :value="ns.name" />
        </el-select>
        <el-button icon="Refresh" @click="fetchWorkloads">刷新</el-button>
      </div>
    </div>

    <el-table :data="workloads" stripe style="width: 100%">
      <el-table-column prop="namespace" label="命名空间" min-width="140" />
      <el-table-column prop="name" label="名称" min-width="200" />
      <el-table-column prop="kind" label="类型" width="140" />
      <el-table-column prop="replicas" label="副本" width="90" />
      <el-table-column prop="ready" label="就绪" width="90" />
      <el-table-column prop="available" label="可用" width="90" />
      <el-table-column label="镜像" min-width="220">
        <template #default="scope">
          <el-tag v-for="img in scope.row.images" :key="img" size="small" class="mr-2">{{ img }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="created_at" label="创建时间" min-width="180" />
      <el-table-column label="操作" width="160">
        <template #default="scope">
          <el-button size="small" @click="openDetail(scope.row)">详情</el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-card>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'

const clusters = ref([])
const namespaces = ref([])
const clusterId = ref('')
const namespace = ref('')
const workloads = ref([])
const router = useRouter()

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

const fetchWorkloads = async () => {
  if (!clusterId.value) return
  const res = await axios.get(`/api/v1/k8s/clusters/${clusterId.value}/workloads`, {
    headers: authHeaders(),
    params: { namespace }
  })
  workloads.value = res.data.data || []
}

const handleClusterChange = async () => {
  await fetchNamespaces()
  namespace.value = ''
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

onMounted(async () => {
  await fetchClusters()
  await fetchNamespaces()
  await fetchWorkloads()
})
</script>

<style scoped>
.page-card { max-width: 1180px; margin: 0 auto; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 16px; }
.page-desc { color: #606266; margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; }
.w-52 { width: 220px; }
.mr-2 { margin-right: 6px; margin-bottom: 6px; }
</style>
