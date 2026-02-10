<template>
  <el-card>
    <template #header>
      <div class="flex justify-between items-center">
        <span class="font-bold">Docker 环境列表</span>
        <div>
          <el-button type="primary" icon="Plus" @click="handleAdd">添加环境</el-button>
          <el-button icon="Refresh" @click="syncAll">刷新</el-button>
        </div>
      </div>
    </template>

    <el-table :data="tableData" v-loading="loading" style="width: 100%">
      <el-table-column prop="name" label="名称" width="180">
        <template #default="{ row }">
          <div class="flex items-center gap-2">
            <el-icon class="text-blue-500 text-xl"><Platform /></el-icon>
            <span class="font-bold">{{ row.name }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="status" label="状态" width="120">
        <template #default="{ row }">
          <el-tag :type="row.status === 'online' ? 'success' : 'danger'">
            {{ row.status || 'unknown' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="container_count" label="容器数" width="120" align="center" />
      <el-table-column prop="image_count" label="镜像数" width="120" align="center" />
      <el-table-column prop="version" label="版本" />
      <el-table-column label="操作" width="250" fixed="right">
        <template #default="{ row }">
          <el-button size="small" type="primary" plain icon="Monitor" @click="handleManage(row)">管理</el-button>
          <el-button size="small" type="warning" plain icon="FirstAidKit" @click="handleDiagnose(row)">诊断</el-button>
          <el-button size="small" type="danger" plain icon="Delete" @click="handleDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 添加主机弹窗 -->
    <el-dialog v-model="dialogVisible" title="添加 Docker 环境" width="500px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="名称">
          <el-input v-model="form.name" placeholder="例如: Local Docker" />
        </el-form-item>
        <el-form-item label="关联主机">
          <el-select v-model="form.host_id" placeholder="请选择" class="w-100">
            <el-option label="本机 (Local Socket)" value="local" />
            <el-option v-for="h in hosts" :key="h.id" :label="h.name + ' (' + h.ip + ')'" :value="h.id" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitForm" :loading="submitting">确定</el-button>
      </template>
    </el-dialog>

    <!-- 管理抽屉 -->
    <el-drawer v-model="manageVisible" size="70%" :with-header="false">
      <div class="drawer-header">
        <div>
          <div class="drawer-title">{{ activeHost?.name || 'Docker 环境' }}</div>
          <div class="drawer-sub">
            状态：<el-tag size="small" :type="activeHost?.status === 'online' ? 'success' : 'danger'">{{ activeHost?.status || 'unknown' }}</el-tag>
            <span class="drawer-meta">容器：{{ activeHost?.container_count ?? '-' }}</span>
            <span class="drawer-meta">镜像：{{ activeHost?.image_count ?? '-' }}</span>
            <span class="drawer-meta">版本：{{ activeHost?.version || '-' }}</span>
          </div>
        </div>
        <div>
          <el-button size="small" icon="Refresh" @click="refreshManage">刷新</el-button>
        </div>
      </div>

      <el-tabs v-model="manageTab" class="manage-tabs">
        <el-tab-pane label="概览" name="overview">
          <el-descriptions :column="2" border>
            <el-descriptions-item label="名称">{{ activeHost?.name || '-' }}</el-descriptions-item>
            <el-descriptions-item label="状态">{{ activeHost?.status || 'unknown' }}</el-descriptions-item>
            <el-descriptions-item label="容器数">{{ activeHost?.container_count ?? '-' }}</el-descriptions-item>
            <el-descriptions-item label="镜像数">{{ activeHost?.image_count ?? '-' }}</el-descriptions-item>
            <el-descriptions-item label="版本">{{ activeHost?.version || '-' }}</el-descriptions-item>
            <el-descriptions-item label="主机ID">{{ activeHost?.host_id || '-' }}</el-descriptions-item>
          </el-descriptions>
        </el-tab-pane>

        <el-tab-pane label="容器" name="containers">
          <el-table :data="containers" v-loading="containersLoading" style="width: 100%">
            <el-table-column prop="Names" label="名称" min-width="200">
              <template #default="{ row }">
                <div>{{ Array.isArray(row.Names) ? row.Names.join(',') : row.Names }}</div>
              </template>
            </el-table-column>
            <el-table-column prop="Image" label="镜像" min-width="180" />
            <el-table-column prop="State" label="状态" width="120" />
            <el-table-column prop="Status" label="详情" min-width="180" />
            <el-table-column prop="Created" label="创建时间" width="160" />
            <el-table-column label="操作" width="280" fixed="right">
              <template #default="{ row }">
                <el-button size="small" @click="openLogs(row)">日志</el-button>
                <el-button size="small" type="success" plain @click="containerAction(row, 'start')">启动</el-button>
                <el-button size="small" type="warning" plain @click="containerAction(row, 'stop')">停止</el-button>
                <el-button size="small" type="primary" plain @click="containerAction(row, 'restart')">重启</el-button>
                <el-button size="small" type="danger" plain @click="containerAction(row, 'remove')">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="镜像" name="images">
          <el-table :data="images" v-loading="imagesLoading" style="width: 100%">
            <el-table-column prop="Repository" label="仓库" min-width="200" />
            <el-table-column prop="Tag" label="Tag" width="120" />
            <el-table-column prop="ID" label="ID" min-width="180" />
            <el-table-column prop="Size" label="大小" width="120" />
            <el-table-column label="操作" width="120" fixed="right">
              <template #default="{ row }">
                <el-button size="small" type="danger" plain @click="removeImage(row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="网络" name="networks">
          <el-table :data="networks" v-loading="networksLoading" style="width: 100%">
            <el-table-column prop="Name" label="名称" min-width="180" />
            <el-table-column prop="ID" label="ID" min-width="200" />
            <el-table-column prop="Driver" label="驱动" width="120" />
            <el-table-column prop="Scope" label="范围" width="120" />
          </el-table>
        </el-tab-pane>
      </el-tabs>
    </el-drawer>

    <!-- 诊断弹窗 -->
    <el-dialog v-model="diagnoseVisible" title="Docker 诊断" width="720px">
      <el-alert v-if="diagnoseError" type="error" :closable="false" show-icon>{{ diagnoseError }}</el-alert>
      <el-skeleton v-if="diagnoseLoading" :rows="6" animated />
      <div v-else class="diagnose-block">
        <div class="diagnose-title">Step1: docker info</div>
        <pre class="diagnose-pre">{{ diagnoseResult?.step1_info?.out || '-' }}</pre>
        <div class="diagnose-title">Step2: docker system info (json)</div>
        <pre class="diagnose-pre">{{ diagnoseResult?.step2_sync?.out || '-' }}</pre>
        <div class="diagnose-title">Step3: docker ps -a</div>
        <pre class="diagnose-pre">{{ diagnoseResult?.step3_list?.out || '-' }}</pre>
      </div>
      <template #footer>
        <el-button @click="diagnoseVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- 容器日志弹窗 -->
    <el-dialog v-model="logVisible" title="容器日志" width="720px">
      <div class="log-controls">
        <el-input v-model="logTail" placeholder="tail" style="width: 120px" />
        <el-button icon="Refresh" @click="loadLogs" :loading="logLoading">刷新</el-button>
      </div>
      <el-input v-model="logText" type="textarea" :rows="16" readonly />
    </el-dialog>
  </el-card>
</template>

<script setup>
import { ref, reactive, onMounted, watch } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const loading = ref(false)
const tableData = ref([])
const dialogVisible = ref(false)
const submitting = ref(false)
const hosts = ref([])

const manageVisible = ref(false)
const manageTab = ref('overview')
const activeHost = ref(null)

const containers = ref([])
const containersLoading = ref(false)
const images = ref([])
const imagesLoading = ref(false)
const networks = ref([])
const networksLoading = ref(false)

const diagnoseVisible = ref(false)
const diagnoseLoading = ref(false)
const diagnoseResult = ref(null)
const diagnoseError = ref('')

const logVisible = ref(false)
const logLoading = ref(false)
const logText = ref('')
const logTail = ref('100')
const logContainerId = ref('')

const form = reactive({
  name: '',
  host_id: ''
})

const authHeaders = () => ({ Authorization: 'Bearer ' + localStorage.getItem('token') })

const fetchData = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/v1/docker/hosts', { headers: authHeaders() })
    if (res.data.code === 0) {
      tableData.value = res.data.data
    }
  } catch (e) {
    ElMessage.error('加载失败')
  } finally {
    loading.value = false
  }
}

const syncAll = async () => {
  loading.value = true
  try {
    await axios.post('/api/v1/docker/hosts/sync', {}, { headers: authHeaders() })
  } catch (e) {
    ElMessage.error('同步失败')
  } finally {
    await fetchData()
  }
}

const fetchCMDBHosts = async () => {
  try {
    const res = await axios.get('/api/v1/cmdb/hosts', { headers: authHeaders() })
    if (res.data.code === 0) {
      hosts.value = res.data.data
    }
  } catch (e) {}
}

const handleAdd = () => {
  fetchCMDBHosts()
  form.name = ''
  form.host_id = ''
  dialogVisible.value = true
}

const submitForm = async () => {
  submitting.value = true
  try {
    const res = await axios.post('/api/v1/docker/hosts', form, { headers: authHeaders() })
    if (res.data.code === 0) {
      ElMessage.success('添加成功')
      dialogVisible.value = false
      try {
        const id = res.data.data?.id
        if (id) {
          await axios.get(`/api/v1/docker/hosts/${id}/info`, { headers: authHeaders() })
        }
      } catch (e) {}
      fetchData()
    } else {
      ElMessage.error(res.data.message)
    }
  } finally {
    submitting.value = false
  }
}

const handleDelete = (row) => {
  ElMessageBox.confirm('确定删除该 Docker 环境吗?', '警告', {
    confirmButtonText: '删除',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    await axios.delete(`/api/v1/docker/hosts/${row.id}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    fetchData()
  })
}

const handleManage = async (row) => {
  activeHost.value = row
  manageVisible.value = true
  manageTab.value = 'overview'
  await refreshManage()
}

const refreshManage = async () => {
  if (!activeHost.value) return
  try {
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/info`, { headers: authHeaders() })
    if (res.data.code === 0) {
      const idx = tableData.value.findIndex(h => h.id === activeHost.value.id)
      if (idx >= 0) tableData.value[idx] = res.data.data
      activeHost.value = res.data.data
    }
  } catch (e) {}
  if (manageTab.value === 'containers') await loadContainers()
  if (manageTab.value === 'images') await loadImages()
  if (manageTab.value === 'networks') await loadNetworks()
}

const loadContainers = async () => {
  if (!activeHost.value) return
  containersLoading.value = true
  try {
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/containers`, { headers: authHeaders() })
    if (res.data.code === 0) {
      containers.value = res.data.data || []
    }
  } finally {
    containersLoading.value = false
  }
}

const loadImages = async () => {
  if (!activeHost.value) return
  imagesLoading.value = true
  try {
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/images`, { headers: authHeaders() })
    if (res.data.code === 0) {
      images.value = res.data.data || []
    }
  } finally {
    imagesLoading.value = false
  }
}

const loadNetworks = async () => {
  if (!activeHost.value) return
  networksLoading.value = true
  try {
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/networks`, { headers: authHeaders() })
    if (res.data.code === 0) {
      networks.value = res.data.data || []
    }
  } finally {
    networksLoading.value = false
  }
}

const containerAction = async (row, action) => {
  if (!activeHost.value) return
  const id = row.ID || row.Id || row.id
  if (!id) return
  try {
    await axios.post(`/api/v1/docker/hosts/${activeHost.value.id}/containers/${encodeURIComponent(id)}/${action}`, {}, { headers: authHeaders() })
    ElMessage.success('操作成功')
    loadContainers()
  } catch (e) {
    ElMessage.error('操作失败')
  }
}

const openLogs = (row) => {
  const id = row.ID || row.Id || row.id
  if (!id) return
  logContainerId.value = id
  logText.value = ''
  logVisible.value = true
  loadLogs()
}

const loadLogs = async () => {
  if (!activeHost.value || !logContainerId.value) return
  logLoading.value = true
  try {
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/containers/${encodeURIComponent(logContainerId.value)}/logs`, {
      params: { tail: logTail.value || '100' },
      headers: authHeaders()
    })
    if (res.data.code === 0) {
      logText.value = res.data.data || ''
    }
  } finally {
    logLoading.value = false
  }
}

const removeImage = (row) => {
  if (!activeHost.value) return
  const id = row.ID || row.Id || row.id
  if (!id) return
  ElMessageBox.confirm('确定删除镜像吗?', '警告', {
    confirmButtonText: '删除',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    await axios.delete(`/api/v1/docker/hosts/${activeHost.value.id}/images/${encodeURIComponent(id)}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    loadImages()
  })
}

const handleDiagnose = async (row) => {
  diagnoseVisible.value = true
  diagnoseLoading.value = true
  diagnoseResult.value = null
  diagnoseError.value = ''
  try {
    const res = await axios.post(`/api/v1/docker/hosts/${row.id}/test`, {}, { headers: authHeaders() })
    if (res.data.code === 0) {
      diagnoseResult.value = res.data.data
    } else {
      diagnoseError.value = res.data.message || '诊断失败'
    }
  } catch (e) {
    diagnoseError.value = '诊断失败'
  } finally {
    diagnoseLoading.value = false
  }
}

watch(manageTab, (tab) => {
  if (tab === 'containers') loadContainers()
  if (tab === 'images') loadImages()
  if (tab === 'networks') loadNetworks()
})

onMounted(() => {
  fetchData()
})
</script>

<style scoped>
.flex { display: flex; }
.justify-between { justify-content: space-between; }
.items-center { align-items: center; }
.gap-2 { gap: 8px; }
.font-bold { font-weight: bold; }
.w-100 { width: 100%; }
.text-blue-500 { color: #409eff; }
.text-xl { font-size: 18px; }
.drawer-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.drawer-title { font-size: 18px; font-weight: 600; }
.drawer-sub { color: #606266; margin-top: 6px; display: flex; gap: 12px; align-items: center; flex-wrap: wrap; }
.drawer-meta { color: #909399; }
.manage-tabs { margin-top: 8px; }
.diagnose-block { display: flex; flex-direction: column; gap: 12px; }
.diagnose-title { font-weight: 600; }
.diagnose-pre { background: #0f172a; color: #e2e8f0; padding: 12px; border-radius: 6px; overflow: auto; max-height: 200px; }
.log-controls { display: flex; gap: 8px; margin-bottom: 10px; }
</style>
