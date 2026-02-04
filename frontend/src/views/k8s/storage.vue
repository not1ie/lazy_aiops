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
        <el-button icon="Refresh" @click="fetchData">刷新</el-button>
      </div>
    </div>

    <el-tabs v-model="activeTab">
      <el-tab-pane label="StorageClass" name="sc">
        <el-table :data="storageClasses" stripe style="width: 100%">
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
          <el-table-column prop="created_at" label="创建时间" min-width="180" />
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="PersistentVolume" name="pv">
        <el-table :data="pvs" stripe style="width: 100%">
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
          <el-table-column prop="created_at" label="创建时间" min-width="180" />
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="PersistentVolumeClaim" name="pvc">
        <el-table :data="pvcs" stripe style="width: 100%">
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
const activeTab = ref('sc')
const storageClasses = ref([])
const pvs = ref([])
const pvcs = ref([])

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
