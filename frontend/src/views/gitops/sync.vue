<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>配置同步</h2>
        <p class="page-desc">管理配置文件、在线编辑与变更追踪。</p>
      </div>
      <div class="page-actions">
        <el-select v-model="repoFilter" class="w-52" clearable placeholder="全部仓库" @change="fetchConfigs">
          <el-option v-for="repo in repos" :key="repo.id" :label="repo.name" :value="repo.id" />
        </el-select>
        <el-button type="primary" icon="Plus" @click="openCreateConfig">新增配置</el-button>
        <el-button icon="Refresh" @click="reloadAll">刷新</el-button>
      </div>
    </div>

    <el-card shadow="never" class="section-card">
      <template #header>
        <div class="card-title">配置文件</div>
      </template>
      <el-table :data="configs" v-loading="configLoading" stripe>
        <el-table-column prop="name" label="名称" min-width="140" />
        <el-table-column label="仓库" min-width="130">
          <template #default="{ row }">{{ row.repo?.name || '-' }}</template>
        </el-table-column>
        <el-table-column prop="file_path" label="文件路径" min-width="220" show-overflow-tooltip />
        <el-table-column prop="type" label="类型" width="100" />
        <el-table-column prop="environment" label="环境" width="110" />
        <el-table-column prop="description" label="描述" min-width="180" show-overflow-tooltip />
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="openEditConfig(row)">编辑内容</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card shadow="never" class="section-card">
      <template #header>
        <div class="card-title">变更历史</div>
      </template>
      <el-table :data="changes" v-loading="changeLoading" stripe>
        <el-table-column prop="created_at" label="时间" width="160">
          <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column prop="config_name" label="配置" min-width="140" />
        <el-table-column prop="change_type" label="类型" width="90" />
        <el-table-column prop="commit_by" label="提交人" width="110" />
        <el-table-column prop="commit_message" label="提交信息" min-width="280" show-overflow-tooltip />
      </el-table>
    </el-card>

    <el-dialog v-model="createVisible" title="新增配置" width="700px">
      <el-form :model="createForm" label-width="110px">
        <el-form-item label="配置名称">
          <el-input v-model="createForm.name" />
        </el-form-item>
        <el-form-item label="所属仓库">
          <el-select v-model="createForm.repo_id" class="w-52" filterable>
            <el-option v-for="repo in repos" :key="repo.id" :label="repo.name" :value="repo.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="文件路径">
          <el-input v-model="createForm.file_path" placeholder="例如：k8s/deploy.yaml" />
        </el-form-item>
        <el-form-item label="文件类型">
          <el-select v-model="createForm.type" class="w-52">
            <el-option label="yaml" value="yaml" />
            <el-option label="json" value="json" />
            <el-option label="toml" value="toml" />
            <el-option label="ini" value="ini" />
          </el-select>
        </el-form-item>
        <el-form-item label="环境">
          <el-input v-model="createForm.environment" placeholder="例如：prod" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="createForm.description" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" @click="submitCreateConfig">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="editVisible" :title="`编辑配置：${editForm.name || ''}`" width="960px" top="6vh">
      <el-alert type="info" :closable="false" class="mb-3">
        <template #default>
          路径：{{ editForm.file_path || '-' }} | 仓库：{{ editForm.repo?.name || '-' }}
        </template>
      </el-alert>
      <el-form :model="editForm" label-width="90px">
        <el-form-item label="提交信息">
          <el-input v-model="editForm.commit_message" placeholder="可选，不填则自动生成" />
        </el-form-item>
        <el-form-item label="文件内容">
          <el-input v-model="editForm.content" type="textarea" :rows="18" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">取消</el-button>
        <el-button type="primary" @click="submitConfigUpdate">提交并推送</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'

const repos = ref([])
const configs = ref([])
const changes = ref([])
const repoFilter = ref('')

const configLoading = ref(false)
const changeLoading = ref(false)

const createVisible = ref(false)
const createForm = ref({
  repo_id: '',
  name: '',
  file_path: '',
  type: 'yaml',
  environment: '',
  description: ''
})

const editVisible = ref(false)
const editId = ref('')
const editForm = ref({
  name: '',
  file_path: '',
  repo: null,
  content: '',
  commit_message: ''
})

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const formatTime = (v) => {
  if (!v) return '-'
  return String(v).slice(0, 19).replace('T', ' ')
}

const fetchRepos = async () => {
  try {
    const res = await axios.get('/api/v1/gitops/repos', { headers: authHeaders() })
    repos.value = res.data?.data || []
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '获取仓库失败')
  }
}

const fetchConfigs = async () => {
  configLoading.value = true
  try {
    const params = {}
    if (repoFilter.value) params.repo_id = repoFilter.value
    const res = await axios.get('/api/v1/gitops/configs', { headers: authHeaders(), params })
    configs.value = res.data?.data || []
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '获取配置失败')
  } finally {
    configLoading.value = false
  }
}

const fetchChanges = async () => {
  changeLoading.value = true
  try {
    const res = await axios.get('/api/v1/gitops/changes', { headers: authHeaders() })
    changes.value = res.data?.data || []
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '获取变更历史失败')
  } finally {
    changeLoading.value = false
  }
}

const reloadAll = async () => {
  await Promise.all([fetchRepos(), fetchConfigs(), fetchChanges()])
}

const openCreateConfig = () => {
  createForm.value = {
    repo_id: repoFilter.value || '',
    name: '',
    file_path: '',
    type: 'yaml',
    environment: '',
    description: ''
  }
  createVisible.value = true
}

const submitCreateConfig = async () => {
  if (!createForm.value.repo_id || !createForm.value.name.trim() || !createForm.value.file_path.trim()) {
    ElMessage.warning('请填写仓库、名称和文件路径')
    return
  }
  try {
    await axios.post('/api/v1/gitops/configs', createForm.value, { headers: authHeaders() })
    ElMessage.success('创建成功')
    createVisible.value = false
    await fetchConfigs()
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '创建失败')
  }
}

const openEditConfig = async (row) => {
  try {
    const res = await axios.get(`/api/v1/gitops/configs/${row.id}`, { headers: authHeaders() })
    const data = res.data?.data || {}
    editId.value = row.id
    editForm.value = {
      ...row,
      content: data.content || '',
      commit_message: ''
    }
    editVisible.value = true
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '读取配置失败')
  }
}

const submitConfigUpdate = async () => {
  if (!editId.value) return
  try {
    await axios.put(
      `/api/v1/gitops/configs/${editId.value}`,
      {
        content: editForm.value.content,
        commit_message: editForm.value.commit_message
      },
      { headers: authHeaders() }
    )
    ElMessage.success('更新成功')
    editVisible.value = false
    await Promise.all([fetchConfigs(), fetchChanges()])
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '更新失败')
  }
}

onMounted(reloadAll)
</script>

<style scoped>
.page-card { max-width: 1280px; margin: 0 auto; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; gap: 12px; margin-bottom: 16px; }
.page-desc { margin: 4px 0 0; color: #606266; }
.page-actions { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
.section-card { margin-bottom: 12px; }
.card-title { font-weight: 600; }
.w-52 { width: 220px; }
.mb-3 { margin-bottom: 12px; }
</style>
