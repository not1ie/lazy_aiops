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
      <el-table-column label="触发策略" width="130">
        <template #default="{ row }">
          <el-tag :type="row.require_approval ? 'warning' : 'success'">{{ row.require_approval ? '审批后执行' : '直接执行' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="credential_name" label="统一凭据" min-width="160" show-overflow-tooltip />
      <el-table-column prop="notify_target_id" label="通知目标" min-width="180" show-overflow-tooltip>
        <template #default="{ row }">
          <span>{{ notifyTargetLabel(row.notify_target_id) }}</span>
        </template>
      </el-table-column>
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

  <el-dialog append-to-body v-model="dialogVisible" :title="isEdit ? '编辑流水线' : '新增流水线'" width="720px" @closed="handleDialogClosed">
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
      <el-form-item label="统一凭据">
        <el-select v-model="form.credential_id" clearable filterable placeholder="可选，选择后优先使用凭据中的账号/Token" style="width: 100%" @change="handleCredentialChange">
          <el-option
            v-for="item in credentialOptions"
            :key="item.id"
            :label="`${item.name}${item.username ? ` (${item.username})` : ''}`"
            :value="item.id"
          />
        </el-select>
        <div class="form-tip">统一凭据在「资产管理 -> 凭据管理」维护，流水线仅做引用。</div>
      </el-form-item>
      <el-alert
        v-if="usingUnifiedCredential"
        title="已启用统一凭据：Token 将从凭据中心注入，建议不再手填散落 Token。"
        type="success"
        :closable="false"
        show-icon
      />
      <el-form-item label="审批触发">
        <el-switch v-model="form.require_approval" />
        <div class="form-tip">开启后，触发将先生成工单，审批通过后再执行流水线。</div>
      </el-form-item>
      <el-form-item v-if="form.require_approval" label="工单类型">
        <el-select v-model="form.workorder_type_id" clearable filterable placeholder="不选则默认使用变更申请" style="width: 100%">
          <el-option v-for="item in workorderTypeOptions" :key="item.id" :label="`${item.name} (${item.code})`" :value="item.id" />
        </el-select>
      </el-form-item>
      <el-form-item label="通知目标">
        <el-select v-model="form.notify_target_id" clearable filterable placeholder="选择通知组或通知渠道（可选）" style="width: 100%">
          <el-option-group v-if="notifyGroupOptions.length" label="通知组">
            <el-option v-for="item in notifyGroupOptions" :key="`group-${item.id}`" :label="item.name" :value="item.id" />
          </el-option-group>
          <el-option-group v-if="notifyChannelOptions.length" label="通知渠道">
            <el-option v-for="item in notifyChannelOptions" :key="`channel-${item.id}`" :label="`${item.name} (${item.type})`" :value="item.id" />
          </el-option-group>
        </el-select>
      </el-form-item>
      <el-form-item label="通知接收人">
        <el-input v-model="form.notify_receiver" placeholder="可选：邮箱/手机号/@用户，由渠道决定格式" />
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
          <el-input v-model="form.jenkins_token" type="password" show-password :disabled="usingUnifiedCredential" :placeholder="usingUnifiedCredential ? '由统一凭据提供' : ''" />
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
          <el-input v-model="form.gitlab_token" type="password" show-password :disabled="usingUnifiedCredential" :placeholder="usingUnifiedCredential ? '由统一凭据提供' : ''" />
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
          <el-input v-model="form.argocd_token" type="password" show-password :disabled="usingUnifiedCredential" :placeholder="usingUnifiedCredential ? '由统一凭据提供' : ''" />
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
          <el-input v-model="form.github_token" type="password" show-password :disabled="usingUnifiedCredential" :placeholder="usingUnifiedCredential ? '由统一凭据提供' : ''" />
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

  <el-dialog append-to-body v-model="triggerVisible" title="触发流水线" width="520px" @closed="handleTriggerDialogClosed">
    <el-form :model="triggerForm" label-width="90px">
      <el-form-item label="参数(JSON)">
        <el-input v-model="triggerForm.parameters" type="textarea" :rows="4" placeholder='{"env":"prod"}' />
      </el-form-item>
      <el-form-item label="触发说明">
        <el-input v-model="triggerForm.reason" type="textarea" :rows="3" placeholder="可选：本次触发原因（审批场景会写入工单）" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="triggerVisible = false">取消</el-button>
      <el-button type="primary" :loading="triggering" @click="triggerPipeline">触发</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, onMounted, computed, watch } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const pipelines = ref([])
const credentialOptions = ref([])
const workorderTypeOptions = ref([])
const notifyChannelOptions = ref([])
const notifyGroupOptions = ref([])
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
  credential_id: '',
  credential_name: '',
  require_approval: false,
  workorder_type_id: '',
  notify_target_id: '',
  notify_receiver: '',
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
  parameters: '',
  reason: ''
})

const usingUnifiedCredential = computed(() => Boolean(form.credential_id))

const resetForm = () => {
  Object.assign(form, {
    name: '',
    description: '',
    provider: 'jenkins',
    credential_id: '',
    credential_name: '',
    require_approval: false,
    workorder_type_id: '',
    notify_target_id: '',
    notify_receiver: '',
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
}

const resetTriggerForm = () => {
  triggerPipelineId.value = ''
  triggerForm.parameters = ''
  triggerForm.reason = ''
}

const headers = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const getErrorMessage = (error, fallback) => {
  if (error?.response?.data?.message) return error.response.data.message
  if (error?.message) return error.message
  return fallback
}

const fetchData = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/v1/cicd/pipelines', { headers: headers() })
    if (res.data.code === 0) {
      pipelines.value = res.data.data
    }
  } catch (error) {
    ElMessage.error(getErrorMessage(error, '加载失败'))
  } finally {
    loading.value = false
  }
}

const fetchCredentialOptions = async () => {
  try {
    const res = await axios.get('/api/v1/cicd/credentials', {
      headers: headers(),
      params: { provider: form.provider || undefined }
    })
    if (res.data.code === 0) {
      credentialOptions.value = Array.isArray(res.data.data) ? res.data.data : []
    }
  } catch (error) {
    ElMessage.error(getErrorMessage(error, '加载凭据失败'))
  }
}

const fetchWorkorderTypes = async () => {
  try {
    const res = await axios.get('/api/v1/workorder/types', { headers: headers() })
    if (res.data.code === 0) {
      const list = Array.isArray(res.data.data) ? res.data.data : []
      workorderTypeOptions.value = list.filter((item) => item.enabled !== false)
    }
  } catch (error) {
    ElMessage.error(getErrorMessage(error, '加载工单类型失败'))
  }
}

const fetchNotifyTargets = async () => {
  try {
    const [groupsRes, channelsRes] = await Promise.all([
      axios.get('/api/v1/notify/groups', { headers: headers() }),
      axios.get('/api/v1/notify/channels', { headers: headers() })
    ])
    if (groupsRes.data.code === 0) {
      notifyGroupOptions.value = Array.isArray(groupsRes.data.data) ? groupsRes.data.data : []
    }
    if (channelsRes.data.code === 0) {
      const channelList = Array.isArray(channelsRes.data.data) ? channelsRes.data.data : []
      notifyChannelOptions.value = channelList.filter((item) => item.enabled !== false)
    }
  } catch (error) {
    ElMessage.error(getErrorMessage(error, '加载通知目标失败'))
  }
}

const openCreate = () => {
  isEdit.value = false
  currentId.value = ''
  resetForm()
  Promise.all([fetchCredentialOptions(), fetchWorkorderTypes(), fetchNotifyTargets()])
  dialogVisible.value = true
}

const openEdit = async (row) => {
  isEdit.value = true
  currentId.value = row.id
  try {
    const res = await axios.get(`/api/v1/cicd/pipelines/${row.id}`, { headers: headers() })
    if (res.data.code === 0) {
      Object.assign(form, res.data.data || {})
      await Promise.all([fetchCredentialOptions(), fetchWorkorderTypes(), fetchNotifyTargets()])
      dialogVisible.value = true
      return
    }
  } catch (error) {
    ElMessage.error(getErrorMessage(error, '加载流水线详情失败'))
  }
  isEdit.value = false
  currentId.value = ''
  resetForm()
}

const buildPayload = () => {
  const payload = { ...form }
  delete payload.credential_name
  if (payload.credential_id) {
    if (payload.provider === 'gitlab') payload.gitlab_token = ''
    if (payload.provider === 'argocd') payload.argocd_token = ''
    if (payload.provider === 'github') payload.github_token = ''
    if (payload.provider === 'jenkins') payload.jenkins_token = ''
  }
  return payload
}

const findCredentialOption = (id) => credentialOptions.value.find((item) => item.id === id)

const notifyTargetLabel = (targetID) => {
  if (!targetID) return '-'
  const group = notifyGroupOptions.value.find((item) => item.id === targetID)
  if (group) return `通知组: ${group.name}`
  const channel = notifyChannelOptions.value.find((item) => item.id === targetID)
  if (channel) return `渠道: ${channel.name}`
  return targetID
}

const handleCredentialChange = (credentialID) => {
  if (!credentialID) return
  const selected = findCredentialOption(credentialID)
  if (form.provider === 'jenkins' && selected?.username && !form.jenkins_user) {
    form.jenkins_user = selected.username
  }
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
    const res = await axios({ url, method, data: buildPayload(), headers: headers() })
    if (res.data.code === 0) {
      ElMessage.success(isEdit.value ? '更新成功' : '创建成功')
      dialogVisible.value = false
      await fetchData()
    }
  } catch (error) {
    ElMessage.error(getErrorMessage(error, '保存失败'))
  } finally {
    saving.value = false
  }
}

const openTrigger = (row) => {
  resetTriggerForm()
  triggerPipelineId.value = row.id
  triggerVisible.value = true
}

const handleDialogClosed = () => {
  isEdit.value = false
  currentId.value = ''
  resetForm()
}

const handleTriggerDialogClosed = () => {
  resetTriggerForm()
}

const triggerPipeline = async () => {
  triggering.value = true
  try {
    const payload = { parameters: {}, reason: triggerForm.reason }
    if (triggerForm.parameters) {
      payload.parameters = JSON.parse(triggerForm.parameters)
    }
    const res = await axios.post(`/api/v1/cicd/pipelines/${triggerPipelineId.value}/trigger`, payload, { headers: headers() })
    if (res.data.code === 0) {
      if (res.data?.data?.mode === 'approval_required') {
        const workorderID = res.data.data.workorder_id
        ElMessage.success(`已提交审批工单：${workorderID}`)
      } else {
        ElMessage.success('触发成功')
      }
      triggerVisible.value = false
    }
  } catch (error) {
    if (error instanceof SyntaxError) {
      ElMessage.error('触发失败：参数 JSON 格式错误')
    } else {
      ElMessage.error(getErrorMessage(error, '触发失败'))
    }
  } finally {
    triggering.value = false
  }
}

const handleDelete = async (row) => {
  try {
    await ElMessageBox.confirm(`确定删除流水线“${row.name}”吗？`, '提示', { type: 'warning' })
    await axios.delete(`/api/v1/cicd/pipelines/${row.id}`, { headers: headers() })
    ElMessage.success('删除成功')
    await fetchData()
  } catch (error) {
    if (error !== 'cancel' && error !== 'close') {
      ElMessage.error(getErrorMessage(error, '删除失败'))
    }
  }
}

onMounted(async () => {
  await Promise.all([fetchData(), fetchCredentialOptions(), fetchWorkorderTypes(), fetchNotifyTargets()])
})

watch(() => form.provider, async () => {
  form.credential_id = ''
  await fetchCredentialOptions()
})

watch(() => form.require_approval, (enabled) => {
  if (!enabled) {
    form.workorder_type_id = ''
  }
})
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.header { display: flex; justify-content: space-between; align-items: center; }
.title { font-size: 18px; font-weight: 600; }
.desc { color: #909399; margin-top: 4px; }
.actions { display: flex; gap: 8px; }
.form-tip { font-size: 12px; color: var(--el-text-color-secondary); margin-top: 6px; }
</style>
