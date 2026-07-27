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

    <div class="table-scroll">
      <el-table :fit="true" :data="clusters" v-loading="loading" stripe style="width: 100%; min-width: 1240px">
      <el-table-column prop="name" label="名称" min-width="140" />
      <el-table-column prop="display_name" label="显示名" min-width="140" />
      <el-table-column prop="api_server" label="API Server" min-width="200" />
      <el-table-column prop="version" label="版本" width="120" />
      <el-table-column prop="status" label="状态" width="110">
        <template #default="scope">
          <el-tag :type="clusterStatusMeta(scope.row).type">
            {{ clusterStatusMeta(scope.row).text }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="最后检测" width="180">
        <template #default="scope">{{ formatTime(scope.row.last_check_at) }}</template>
      </el-table-column>
      <el-table-column prop="status_reason" label="状态说明" min-width="180" show-overflow-tooltip />
      <el-table-column prop="description" label="描述" min-width="180" />
      <el-table-column label="操作" width="240">
        <template #default="scope">
          <el-button size="small" @click="openEdit(scope.row)">编辑</el-button>
          <el-button size="small" type="primary" :loading="testingId === scope.row.id" @click="testConnection(scope.row)">测试</el-button>
          <el-button size="small" type="danger" @click="removeCluster(scope.row)">删除</el-button>
        </template>
      </el-table-column>
      </el-table>
    </div>

    <el-dialog append-to-body v-model="dialogVisible" :title="dialogTitle" width="680px">
      <el-alert
        type="info"
        :closable="false"
        show-icon
        style="margin-bottom: 16px"
        title="K8s 集群接入指南"
      >
        <template #default>
          <div style="font-size: 12px; line-height: 1.6; margin-top: 4px">
            <strong>推荐获取完整内嵌证书配置的 3 种方式：</strong><br />
            1. <strong>独立文件</strong>：复制 K8s Master 节点上的 <code>/etc/kubernetes/admin.conf</code> 内容并粘贴；<br />
            2. <strong>导出为内嵌证书 YAML</strong>：在节点执行 <code>kubectl config view --flatten</code> 导出包含证书数据的文本粘贴；<br />
            3. <strong>Token 模式</strong>：在 API Server 填入集群 URL（如 <code>https://192.168.10.100:6443</code>），并在 KubeConfig 框内直接粘贴 ServiceAccount 的 Bearer Token。
          </div>
        </template>
      </el-alert>
      <el-form :model="form" label-width="110px">
        <el-form-item label="名称">
          <el-input v-model="form.name" placeholder="例如: prod" />
        </el-form-item>
        <el-form-item label="显示名">
          <el-input v-model="form.display_name" placeholder="例如: 生产集群" />
        </el-form-item>
        <el-form-item label="API Server">
          <el-input v-model="form.api_server" placeholder="https://192.168.10.100:6443 (若配置中为 localhost 请务必在此填写)" />
        </el-form-item>
        <el-form-item label="KubeConfig">
          <el-input v-model="form.kubeconfig" type="textarea" :rows="7" placeholder="粘贴 admin.conf、flatten kubeconfig 内容或 Bearer Token" />
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
import { ref, onBeforeUnmount, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import { k8sClusterStatusMeta } from '@/utils/status'

const clusters = ref([])
const loading = ref(false)
const dialogVisible = ref(false)
const dialogTitle = ref('')
const isEdit = ref(false)
const currentId = ref('')
let autoRefreshTimer = null
const form = ref({
  name: '',
  display_name: '',
  api_server: '',
  kubeconfig: '',
  description: ''
})
const testingId = ref('')

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })
const getErrorMessage = (err, fallback = '操作失败') => err?.response?.data?.message || err?.message || fallback
const clusterStatusMeta = (row) => k8sClusterStatusMeta(row, { staleMinutes: 5 })
const formatTime = (value) => {
  if (!value) return '-'
  const ts = new Date(value).getTime()
  if (Number.isNaN(ts)) return '-'
  return new Date(ts).toLocaleString()
}

const fetchClusters = async () => {
  if (loading.value) return
  loading.value = true
  try {
    const res = await axios.get('/api/v1/k8s/clusters', { headers: authHeaders(), params: { live: 1 } })
    clusters.value = res.data.data || []
  } catch (e) {
    ElMessage.error(getErrorMessage(e, '加载集群失败'))
  } finally {
    loading.value = false
  }
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
  try {
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
    await fetchClusters()
  } catch (e) {
    ElMessage.error(getErrorMessage(e, '保存失败'))
  }
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
    await fetchClusters()
  }
}

const removeCluster = async (row) => {
  try {
    await ElMessageBox.confirm(`确认删除集群 ${row.name} 吗？`, '提示', { type: 'warning' })
    await axios.delete(`/api/v1/k8s/clusters/${row.id}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    await fetchClusters()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error(getErrorMessage(e, '删除失败'))
  }
}

onMounted(() => {
  fetchClusters()
  autoRefreshTimer = window.setInterval(() => {
    if (document.hidden || loading.value) return
    fetchClusters()
  }, 60 * 1000)
})

onBeforeUnmount(() => {
  if (autoRefreshTimer) {
    window.clearInterval(autoRefreshTimer)
    autoRefreshTimer = null
  }
})
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 16px; }
.page-desc { color: #606266; margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; }
.table-scroll { overflow-x: auto; }
</style>
