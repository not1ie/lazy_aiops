<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>通知模板</h2>
        <p class="page-desc">管理告警/工单/任务等场景的消息模板。</p>
      </div>
      <div class="page-actions">
        <el-button type="primary" icon="Plus" @click="openCreate">新增模板</el-button>
        <el-button icon="Refresh" @click="fetchTemplates">刷新</el-button>
      </div>
    </div>

    <el-table :data="templates" v-loading="loading" stripe style="width: 100%">
      <el-table-column prop="name" label="名称" min-width="160" />
      <el-table-column prop="type" label="类型" width="120" />
      <el-table-column prop="channel_type" label="渠道类型" width="120" />
      <el-table-column prop="title" label="标题模板" min-width="200" />
      <el-table-column label="启用" width="100">
        <template #default="{ row }">
          <el-tag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? '是' : '否' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="200" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="openEdit(row)">编辑</el-button>
          <el-button size="small" type="danger" @click="removeTemplate(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="760px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="名称">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="类型">
          <el-select v-model="form.type" class="w-52">
            <el-option label="alert" value="alert" />
            <el-option label="workorder" value="workorder" />
            <el-option label="task" value="task" />
            <el-option label="custom" value="custom" />
          </el-select>
        </el-form-item>
        <el-form-item label="渠道类型">
          <el-select v-model="form.channel_type" class="w-52">
            <el-option label="webhook" value="webhook" />
            <el-option label="feishu" value="feishu" />
            <el-option label="dingtalk" value="dingtalk" />
            <el-option label="wecom" value="wecom" />
            <el-option label="email" value="email" />
            <el-option label="sms" value="sms" />
          </el-select>
        </el-form-item>
        <el-form-item label="标题模板">
          <el-input v-model="form.title" />
        </el-form-item>
        <el-form-item label="内容模板">
          <el-input v-model="form.content" type="textarea" :rows="8" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
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
import { onMounted, ref } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const loading = ref(false)
const templates = ref([])
const dialogVisible = ref(false)
const dialogTitle = ref('')
const isEdit = ref(false)
const currentId = ref('')

const defaultForm = () => ({
  name: '',
  type: 'alert',
  title: '',
  content: '',
  channel_type: 'webhook',
  enabled: true
})

const form = ref(defaultForm())

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const fetchTemplates = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/v1/notify/templates', { headers: authHeaders() })
    templates.value = res.data?.data || []
  } finally {
    loading.value = false
  }
}

const openCreate = () => {
  isEdit.value = false
  dialogTitle.value = '新增模板'
  form.value = defaultForm()
  dialogVisible.value = true
}

const openEdit = (row) => {
  isEdit.value = true
  dialogTitle.value = '编辑模板'
  currentId.value = row.id
  form.value = { ...defaultForm(), ...row }
  dialogVisible.value = true
}

const submitForm = async () => {
  if (!form.value.name.trim()) {
    ElMessage.warning('请填写名称')
    return
  }
  try {
    if (isEdit.value) {
      await axios.put(`/api/v1/notify/templates/${currentId.value}`, form.value, { headers: authHeaders() })
      ElMessage.success('更新成功')
    } else {
      await axios.post('/api/v1/notify/templates', form.value, { headers: authHeaders() })
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    await fetchTemplates()
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '保存失败')
  }
}

const removeTemplate = async (row) => {
  try {
    await ElMessageBox.confirm(`确认删除模板 ${row.name} 吗？`, '提示', { type: 'warning' })
    await axios.delete(`/api/v1/notify/templates/${row.id}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    await fetchTemplates()
  } catch (err) {
    if (err !== 'cancel') ElMessage.error('删除失败')
  }
}

onMounted(fetchTemplates)
</script>

<style scoped>
.page-card { max-width: 1180px; margin: 0 auto; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 16px; }
.page-desc { color: #606266; margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; }
.w-52 { width: 220px; }
</style>
