<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>Pods</h2>
        <p class="page-desc">Pod 列表、日志与删除操作。</p>
      </div>
      <div class="page-actions">
        <el-select v-model="clusterId" placeholder="选择集群" class="w-52" @change="handleClusterChange">
          <el-option v-for="c in clusters" :key="c.id" :label="c.display_name || c.name" :value="c.id" />
        </el-select>
        <el-select v-model="namespace" placeholder="命名空间" class="w-52" @change="fetchPods">
          <el-option v-for="ns in namespaces" :key="ns.name" :label="ns.name" :value="ns.name" />
        </el-select>
        <el-button icon="Refresh" @click="fetchPods">刷新</el-button>
      </div>
    </div>

    <el-table :data="pods" stripe style="width: 100%">
      <el-table-column prop="namespace" label="命名空间" min-width="140" />
      <el-table-column prop="name" label="名称" min-width="220" />
      <el-table-column prop="status" label="状态" width="120" />
      <el-table-column prop="node" label="节点" min-width="160" />
      <el-table-column prop="ip" label="IP" width="140" />
      <el-table-column prop="restarts" label="重启" width="80" />
      <el-table-column label="容器" min-width="220">
        <template #default="scope">
          <el-tag v-for="c in scope.row.containers" :key="c.name" size="small" class="mr-2">
            {{ c.name }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="created_at" label="创建时间" min-width="180" />
      <el-table-column label="操作" width="220">
        <template #default="scope">
          <el-button size="small" @click="openDetail(scope.row)">详情</el-button>
          <el-button size="small" @click="openLogs(scope.row)">日志</el-button>
          <el-button size="small" type="danger" @click="deletePod(scope.row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="logVisible" title="Pod 日志" width="880px">
      <div class="log-controls">
        <el-select v-model="logContainer" placeholder="容器" class="w-52">
          <el-option v-for="c in logContainers" :key="c" :label="c" :value="c" />
        </el-select>
        <el-input-number v-model="logTail" :min="10" :max="1000" />
        <el-button type="primary" @click="fetchLogs">获取日志</el-button>
      </div>
      <el-input v-model="logText" type="textarea" :rows="18" readonly />
      <template #footer>
        <el-button @click="logVisible = false">关闭</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const clusters = ref([])
const namespaces = ref([])
const clusterId = ref('')
const namespace = ref('')
const pods = ref([])

const logVisible = ref(false)
const logText = ref('')
const logTail = ref(200)
const logContainer = ref('')
const logContainers = ref([])
const logPod = ref(null)
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
  if (!namespace.value && namespaces.value.length > 0) {
    namespace.value = namespaces.value[0].name
  }
}

const fetchPods = async () => {
  if (!clusterId.value || !namespace.value) return
  const res = await axios.get(`/api/v1/k8s/clusters/${clusterId.value}/namespaces/${namespace.value}/pods`, {
    headers: authHeaders()
  })
  pods.value = res.data.data || []
}

const handleClusterChange = async () => {
  await fetchNamespaces()
  await fetchPods()
}

const openLogs = (row) => {
  logPod.value = row
  logContainers.value = row.containers?.map(c => c.name) || []
  logContainer.value = logContainers.value[0] || ''
  logText.value = ''
  logVisible.value = true
}

const openDetail = (row) => {
  router.push({
    path: '/k8s/pods/detail',
    query: {
      clusterId: clusterId.value,
      namespace: row.namespace,
      name: row.name
    }
  })
}

const fetchLogs = async () => {
  if (!logPod.value) return
  const res = await axios.get(`/api/v1/k8s/clusters/${clusterId.value}/namespaces/${logPod.value.namespace}/pods/${logPod.value.name}/logs`, {
    headers: authHeaders(),
    params: { container: logContainer.value, tail: logTail.value }
  })
  logText.value = res.data.data || ''
}

const deletePod = async (row) => {
  await ElMessageBox.confirm(`确认删除 Pod ${row.name} 吗？`, '提示', { type: 'warning' })
  await axios.delete(`/api/v1/k8s/clusters/${clusterId.value}/namespaces/${row.namespace}/pods/${row.name}`, {
    headers: authHeaders()
  })
  ElMessage.success('删除成功')
  fetchPods()
}

onMounted(async () => {
  await fetchClusters()
  await fetchNamespaces()
  await fetchPods()
})
</script>

<style scoped>
.page-card { max-width: 1180px; margin: 0 auto; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 16px; }
.page-desc { color: #606266; margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; }
.w-52 { width: 220px; }
.mr-2 { margin-right: 6px; margin-bottom: 6px; }
.log-controls { display: flex; gap: 12px; margin-bottom: 12px; align-items: center; }
</style>
