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
        <el-button icon="Refresh" @click="fetchData">刷新</el-button>
      </div>
    </div>

    <el-tabs v-model="activeTab">
      <el-tab-pane label="ConfigMaps" name="configmaps">
        <el-table :data="configmaps" stripe style="width: 100%">
          <el-table-column prop="namespace" label="命名空间" min-width="140" />
          <el-table-column prop="name" label="名称" min-width="200" />
          <el-table-column label="Keys" min-width="220">
            <template #default="scope">
              <el-tag v-for="k in scope.row.data_keys" :key="k" size="small" class="mr-2">{{ k }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="created_at" label="创建时间" min-width="180" />
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="Secrets" name="secrets">
        <el-table :data="secrets" stripe style="width: 100%">
          <el-table-column prop="namespace" label="命名空间" min-width="140" />
          <el-table-column prop="name" label="名称" min-width="200" />
          <el-table-column prop="type" label="类型" width="180" />
          <el-table-column label="Keys" min-width="220">
            <template #default="scope">
              <el-tag v-for="k in scope.row.data_keys" :key="k" size="small" class="mr-2">{{ k }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="created_at" label="创建时间" min-width="180" />
        </el-table>
      </el-tab-pane>
    </el-tabs>
  </el-card>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import axios from 'axios'

const clusters = ref([])
const namespaces = ref([])
const clusterId = ref('')
const namespace = ref('')
const activeTab = ref('configmaps')
const configmaps = ref([])
const secrets = ref([])

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

onMounted(async () => {
  await fetchClusters()
  await fetchNamespaces()
  await fetchData()
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
