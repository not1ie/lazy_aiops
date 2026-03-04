<template>
  <el-card class="page-card">
    <template #header>
      <div class="header">
        <div>
          <div class="title">流水线管理</div>
          <div class="desc">CI/CD 流水线配置与触发</div>
        </div>
        <div class="actions">
          <el-button type="primary" icon="Plus" @click="openCreate">新增流水线</el-button>
          <el-button icon="Refresh" @click="fetchData">刷新</el-button>
        </div>
      </div>
    </template>

    <el-table :fit="true" :data="pipelines" v-loading="loading" stripe>
      <el-table-column prop="name" label="名称" min-width="180" />
      <el-table-column prop="provider" label="Provider" width="120" />
      <el-table-column prop="status" label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="row.status === 1 ? 'success' : 'info'">{{ row.status === 1 ? '启用' : '禁用' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="description" label="描述" min-width="220" show-overflow-tooltip />
      <el-table-column label="操作" width="260" fixed="right">
        <template #default="{ row }">
          <el-button size="small" type="primary" plain @click="openEdit(row)">编辑</el-button>
          <el-button size="small" type="success" plain @click="openTrigger(row)">触发</el-button>
          <el-button size="small" type="danger" plain @click="handleDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-card>

  <el-dialog append-to-body v-model="dialogVisible" :title="isEdit ? '编辑流水线' : '新增流水线'" width="720px">
    <el-form :model="form" label-width="110px">
      <el-form-item label="名称" required>
        <el-input v-model="form.name" />
      </el-form-item>
      <el-form-item label="描述">
        <el-input v-model="form.description" />
      </el-form-item>
      <el-form-item label="Provider" required>
        <el-select v-model="form.provider" style="width: 100%">
          <el-option label="Jenkins" value="jenkins" />
          <el-option label="GitLab" value="gitlab" />
          <el-option label="ArgoCD" value="argocd" />
          <el-option label="GitHub" value="github" />
        </el-select>
      </el-form-item>

      <template v-if="form.provider === 'jenkins'">
        <el-form-item label="Jenkins URL">
          <el-input v-model="form.jenkins_url" />
        </el-form-item>
        <el-form-item label="Job Name">
          <el-input v-model="form.jenkins_job" />
        </el-form-item>
        <el-form-item label="User">
          <el-input v-model="form.jenkins_user" />
        </el-form-item>
        <el-form-item label="Token">
          <el-input v-model="form.jenkins_token" type="password" show-password />
        </el-form-item>
      </template>

      <template v-if="form.provider === 'gitlab'">
        <el-form-item label="GitLab URL">
          <el-input v-model="form.gitlab_url" />
        </el-form-item>
        <el-form-item label="Project ID">
          <el-input v-model="form.gitlab_project_id" />
        </el-form-item>
        <el-form-item label="Token">
          <el-input v-model="form.gitlab_token" type="password" show-password />
        </el-form-item>
        <el-form-item label="Ref">
          <el-input v-model="form.gitlab_ref" placeholder="main" />
        </el-form-item>
      </template>

      <template v-if="form.provider === 'argocd'">
        <el-form-item label="ArgoCD URL">
          <el-input v-model="form.argocd_url" />
        </el-form-item>
        <el-form-item label="App Name">
          <el-input v-model="form.argocd_app" />
        </el-form-item>
        <el-form-item label="Token">
          <el-input v-model="form.argocd_token" type="password" show-password />
        </el-form-item>
      </template>

      <template v-if="form.provider === 'github'">
        <el-form-item label="Repo">
          <el-input v-model="form.github_repo" placeholder="org/repo" />
        </el-form-item>
        <el-form-item label="Workflow">
          <el-input v-model="form.github_workflow" />
        </el-form-item>
        <el-form-item label="Token">
          <el-input v-model="form.github_token" type="password" show-password />
        </el-form-item>
      </template>

      <el-form-item label="参数(JSON)">
        <el-input v-model="form.parameters" type="textarea" :rows="3" placeholder='{"env":"prod"}' />
      </el-form-item>
      <el-form-item label="环境变量">
        <el-input v-model="form.env_vars" type="textarea" :rows="3" placeholder='KEY=VALUE' />
      </el-form-item>
      <el-form-item label="状态">
        <el-switch v-model="form.status" :active-value="1" :inactive-value="0" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="dialogVisible = false">取消</el-button>
      <el-button type="primary" :loading="saving" @click="savePipeline">保存</el-button>
    </template>
  </el-dialog>

  <el-dialog append-to-body v-model="triggerVisible" title="触发流水线" width="520px">
    <el-form :model="triggerForm" label-width="90px">
      <el-form-item label="参数(JSON)">
        <el-input v-model="triggerForm.parameters" type="textarea" :rows="4" placeholder='{"env":"prod"}' />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="triggerVisible = false">取消</el-button>
      <el-button type="primary" :loading="triggering" @click="triggerPipeline">触发</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const pipelines = ref([])
const loading = ref(false)
const saving = ref(false)
const dialogVisible = ref(false)
const isEdit = ref(false)
const currentId = ref('')

const triggerVisible = ref(false)
const triggering = ref(false)
const triggerPipelineId = ref('')

const form = reactive({
  name: '',
  description: '',
  provider: 'jenkins',
  jenkins_url: '',
  jenkins_job: '',
  jenkins_user: '',
  jenkins_token: '',
  gitlab_url: '',
  gitlab_project_id: '',
  gitlab_token: '',
  gitlab_ref: 'main',
  argocd_url: '',
  argocd_app: '',
  argocd_token: '',
  github_repo: '',
  github_token: '',
  github_workflow: '',
  parameters: '',
  env_vars: '',
  status: 1
})

const triggerForm = reactive({
  parameters: ''
})

const headers = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const fetchData = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/v1/cicd/pipelines', { headers: headers() })
    if (res.data.code === 0) {
      pipelines.value = res.data.data
    }
  } catch (error) {
    ElMessage.error('加载失败')
  } finally {
    loading.value = false
  }
}

const openCreate = () => {
  isEdit.value = false
  currentId.value = ''
  Object.assign(form, {
    name: '',
    description: '',
    provider: 'jenkins',
    jenkins_url: '',
    jenkins_job: '',
    jenkins_user: '',
    jenkins_token: '',
    gitlab_url: '',
    gitlab_project_id: '',
    gitlab_token: '',
    gitlab_ref: 'main',
    argocd_url: '',
    argocd_app: '',
    argocd_token: '',
    github_repo: '',
    github_token: '',
    github_workflow: '',
    parameters: '',
    env_vars: '',
    status: 1
  })
  dialogVisible.value = true
}

const openEdit = (row) => {
  isEdit.value = true
  currentId.value = row.id
  Object.assign(form, row)
  dialogVisible.value = true
}

const savePipeline = async () => {
  if (!form.name) {
    ElMessage.warning('请填写名称')
    return
  }
  saving.value = true
  try {
    const url = isEdit.value ? `/api/v1/cicd/pipelines/${currentId.value}` : '/api/v1/cicd/pipelines'
    const method = isEdit.value ? 'put' : 'post'
    const res = await axios({ url, method, data: form, headers: headers() })
    if (res.data.code === 0) {
      ElMessage.success(isEdit.value ? '更新成功' : '创建成功')
      dialogVisible.value = false
      fetchData()
    }
  } catch (error) {
    ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

const openTrigger = (row) => {
  triggerPipelineId.value = row.id
  triggerForm.parameters = ''
  triggerVisible.value = true
}

const triggerPipeline = async () => {
  triggering.value = true
  try {
    const payload = { parameters: {} }
    if (triggerForm.parameters) {
      payload.parameters = JSON.parse(triggerForm.parameters)
    }
    const res = await axios.post(`/api/v1/cicd/pipelines/${triggerPipelineId.value}/trigger`, payload, { headers: headers() })
    if (res.data.code === 0) {
      ElMessage.success('触发成功')
      triggerVisible.value = false
    }
  } catch (error) {
    ElMessage.error('触发失败：参数格式错误')
  } finally {
    triggering.value = false
  }
}

const handleDelete = (row) => {
  ElMessageBox.confirm(`确定删除流水线“${row.name}”吗？`, '提示', { type: 'warning' }).then(async () => {
    await axios.delete(`/api/v1/cicd/pipelines/${row.id}`, { headers: headers() })
    ElMessage.success('删除成功')
    fetchData()
  })
}

onMounted(fetchData)
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.header { display: flex; justify-content: space-between; align-items: center; }
.title { font-size: 18px; font-weight: 600; }
.desc { color: #909399; margin-top: 4px; }
.actions { display: flex; gap: 8px; }
</style>
