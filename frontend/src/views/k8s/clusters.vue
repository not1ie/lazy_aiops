<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>集群管理</h2>
        <p class="page-desc">管理多集群接入、连接状态与基本信息。</p>
      </div>
      <div class="page-actions">
        <el-button type="primary" icon="Plus" @click="openCreate">新增集群</el-button>
        <el-button icon="Refresh" @click="fetchClusters">刷新</el-button>
      </div>
    </div>

    <el-table :fit="true" :data="clusters" stripe style="width: 100%">
      <el-table-column prop="name" label="名称" min-width="140" />
      <el-table-column prop="display_name" label="显示名" min-width="140" />
      <el-table-column prop="api_server" label="API Server" min-width="200" />
      <el-table-column prop="version" label="版本" width="120" />
      <el-table-column prop="status" label="状态" width="100">
        <template #default="scope">
          <el-tag :type="scope.row.status === 1 ? 'success' : 'warning'">
            {{ scope.row.status === 1 ? '正常' : '异常' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="description" label="描述" min-width="180" />
      <el-table-column label="操作" width="240">
        <template #default="scope">
          <el-button size="small" @click="openEdit(scope.row)">编辑</el-button>
          <el-button size="small" type="primary" :loading="testingId === scope.row.id" @click="testConnection(scope.row)">测试</el-button>
          <el-button size="small" type="danger" @click="removeCluster(scope.row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog append-to-body v-model="dialogVisible" :title="dialogTitle" width="640px">
      <el-form :model="form" label-width="110px">
        <el-form-item label="名称">
          <el-input v-model="form.name" placeholder="例如: prod" />
        </el-form-item>
        <el-form-item label="显示名">
          <el-input v-model="form.display_name" placeholder="例如: 生产集群" />
        </el-form-item>
        <el-form-item label="API Server">
          <el-input v-model="form.api_server" placeholder="https://x.x.x.x:6443" />
        </el-form-item>
        <el-form-item label="KubeConfig">
          <el-input v-model="form.kubeconfig" type="textarea" :rows="6" placeholder="粘贴 kubeconfig 内容" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitForm">保存</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const clusters = ref([])
const dialogVisible = ref(false)
const dialogTitle = ref('')
const isEdit = ref(false)
const currentId = ref('')
const form = ref({
  name: '',
  display_name: '',
  api_server: '',
  kubeconfig: '',
  description: ''
})
const testingId = ref('')

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const fetchClusters = async () => {
  const res = await axios.get('/api/v1/k8s/clusters', { headers: authHeaders() })
  clusters.value = res.data.data || []
}

const openCreate = () => {
  isEdit.value = false
  dialogTitle.value = '新增集群'
  form.value = { name: '', display_name: '', api_server: '', kubeconfig: '', description: '' }
  dialogVisible.value = true
}

const openEdit = (row) => {
  isEdit.value = true
  dialogTitle.value = '编辑集群'
  currentId.value = row.id
  form.value = {
    name: row.name,
    display_name: row.display_name,
    api_server: row.api_server,
    kubeconfig: '',
    description: row.description
  }
  dialogVisible.value = true
}

const submitForm = async () => {
  const payload = {
    name: form.value.name,
    display_name: form.value.display_name,
    api_server: form.value.api_server,
    kube_config: form.value.kubeconfig,
    description: form.value.description
  }

  if (isEdit.value) {
    await axios.put(`/api/v1/k8s/clusters/${currentId.value}`, payload, { headers: authHeaders() })
    ElMessage.success('更新成功')
  } else {
    await axios.post('/api/v1/k8s/clusters', payload, { headers: authHeaders() })
    ElMessage.success('创建成功')
  }

  dialogVisible.value = false
  fetchClusters()
}

const testConnection = async (row) => {
  testingId.value = row.id
  try {
    const res = await axios.post(`/api/v1/k8s/clusters/${row.id}/test`, {}, { headers: authHeaders() })
    ElMessage.success(`连接成功: ${res.data.data.version}`)
  } catch (e) {
    const msg = e?.response?.data?.message || '测试失败，请检查 kubeconfig / API Server'
    ElMessage.error(msg)
  } finally {
    testingId.value = ''
    fetchClusters()
  }
}

const removeCluster = async (row) => {
  await ElMessageBox.confirm(`确认删除集群 ${row.name} 吗？`, '提示', { type: 'warning' })
  await axios.delete(`/api/v1/k8s/clusters/${row.id}`, { headers: authHeaders() })
  ElMessage.success('删除成功')
  fetchClusters()
}

onMounted(fetchClusters)
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 16px; }
.page-desc { color: #606266; margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; }
</style>
