<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>GitOps 仓库</h2>
        <p class="page-desc">管理配置仓库、同步状态与本地目录。</p>
      </div>
      <div class="page-actions">
        <el-button type="primary" icon="Plus" @click="openCreate">新增仓库</el-button>
        <el-button icon="Refresh" @click="fetchRepos">刷新</el-button>
      </div>
    </div>

    <el-table :fit="true" :data="repos" v-loading="loading" stripe>
      <el-table-column prop="name" label="名称" min-width="140" />
      <el-table-column prop="url" label="URL" min-width="260" show-overflow-tooltip />
      <el-table-column prop="branch" label="分支" width="120" />
      <el-table-column prop="local_path" label="本地路径" min-width="220" show-overflow-tooltip />
      <el-table-column label="状态" width="110">
        <template #default="{ row }">
          <StatusBadge v-bind="repoStatusBadge(row)" />
        </template>
      </el-table-column>
      <el-table-column label="最后同步" width="130">
        <template #default="{ row }">{{ formatTime(row.last_sync_at) }}</template>
      </el-table-column>
      <el-table-column prop="description" label="描述" min-width="180" show-overflow-tooltip />
      <el-table-column label="操作" width="260" fixed="right">
        <template #default="{ row }">
          <el-button size="small" type="primary" plain @click="syncRepo(row)">同步</el-button>
          <el-button size="small" @click="openDetail(row)">详情</el-button>
          <el-button size="small" type="danger" @click="removeRepo(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog append-to-body v-model="dialogVisible" title="新增仓库" width="720px">
      <el-form :model="form" label-width="110px">
        <el-form-item label="仓库名称">
          <el-input v-model="form.name" placeholder="例如：cmdb-config" />
        </el-form-item>
        <el-form-item label="仓库URL">
          <el-input v-model="form.url" placeholder="https://... 或 git@..." />
        </el-form-item>
        <el-form-item label="分支">
          <el-input v-model="form.branch" placeholder="main" />
        </el-form-item>
        <el-form-item label="SSH私钥">
          <el-input v-model="form.ssh_key" type="textarea" :rows="4" placeholder="可选：私有仓库可填" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitCreate">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog append-to-body v-model="detailVisible" title="仓库详情" width="760px" @closed="handleDetailClosed">
      <el-descriptions :column="1" border>
        <el-descriptions-item label="名称">{{ detailRow?.name || '-' }}</el-descriptions-item>
        <el-descriptions-item label="URL">{{ detailRow?.url || '-' }}</el-descriptions-item>
        <el-descriptions-item label="分支">{{ detailRow?.branch || '-' }}</el-descriptions-item>
        <el-descriptions-item label="本地路径">{{ detailRow?.local_path || '-' }}</el-descriptions-item>
        <el-descriptions-item label="状态">{{ repoStatusLabel(detailRow?.status) }}</el-descriptions-item>
        <el-descriptions-item label="最后同步">{{ formatTime(detailRow?.last_sync_at) }}</el-descriptions-item>
        <el-descriptions-item label="描述">{{ detailRow?.description || '-' }}</el-descriptions-item>
      </el-descriptions>
      <template #footer>
        <el-button @click="detailVisible = false">关闭</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getErrorMessage, isCancelError } from '@/utils/error'
import StatusBadge from '@/components/common/StatusBadge.vue'
import { gitRepoStatusMeta } from '@/utils/status'

const repos = ref([])
const loading = ref(false)
const dialogVisible = ref(false)
const detailVisible = ref(false)
const detailRow = ref(null)

const form = ref({
  name: '',
  url: '',
  branch: 'main',
  ssh_key: '',
  description: ''
})

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const repoStatusBadge = (row) => gitRepoStatusMeta(row, {
  source: 'GitOps同步',
  checkAt: row?.last_sync_at || row?.updated_at
})

const repoStatusLabel = (status) => gitRepoStatusMeta(status).text

const formatTime = (v) => {
  if (!v) return '-'
  return String(v).slice(0, 19).replace('T', ' ')
}

const fetchRepos = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/v1/gitops/repos', { headers: authHeaders() })
    repos.value = res.data?.data || []
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '获取仓库失败'))
  } finally {
    loading.value = false
  }
}

const openCreate = () => {
  form.value = {
    name: '',
    url: '',
    branch: 'main',
    ssh_key: '',
    description: ''
  }
  dialogVisible.value = true
}

const submitCreate = async () => {
  if (!form.value.name.trim() || !form.value.url.trim()) {
    ElMessage.warning('请填写名称和URL')
    return
  }
  try {
    await axios.post('/api/v1/gitops/repos', form.value, { headers: authHeaders() })
    ElMessage.success('创建成功，已异步克隆仓库')
    dialogVisible.value = false
    await fetchRepos()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '创建失败'))
  }
}

const openDetail = async (row) => {
  try {
    const res = await axios.get(`/api/v1/gitops/repos/${row.id}`, { headers: authHeaders() })
    detailRow.value = res.data?.data || row
    detailVisible.value = true
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '获取详情失败'))
  }
}

const handleDetailClosed = () => {
  detailRow.value = null
}

const syncRepo = async (row) => {
  try {
    await axios.post(`/api/v1/gitops/repos/${row.id}/sync`, {}, { headers: authHeaders() })
    ElMessage.success('同步成功')
    await fetchRepos()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '同步失败'))
  }
}

const removeRepo = async (row) => {
  try {
    await ElMessageBox.confirm(`确认删除仓库 ${row.name} 吗？会同时删除本地目录。`, '提示', { type: 'warning' })
    await axios.delete(`/api/v1/gitops/repos/${row.id}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    if (detailRow.value?.id === row.id) {
      detailVisible.value = false
    }
    await fetchRepos()
  } catch (err) {
    if (!isCancelError(err)) ElMessage.error(getErrorMessage(err, '删除失败'))
  }
}

onMounted(fetchRepos)
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; gap: 12px; margin-bottom: 16px; }
.page-desc { margin: 4px 0 0; color: #606266; }
.page-actions { display: flex; gap: 8px; align-items: center; }
</style>
