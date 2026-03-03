<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>工单类型</h2>
        <p class="page-desc">维护工单分类、编码与模板，作为工单创建入口。</p>
      </div>
      <div class="page-actions">
        <el-button type="primary" icon="Plus" @click="openCreate">新增类型</el-button>
        <el-button icon="Refresh" @click="fetchTypes">刷新</el-button>
      </div>
    </div>

    <el-table :data="types" v-loading="loading" stripe>
      <el-table-column prop="name" label="名称" min-width="150" />
      <el-table-column prop="code" label="编码" width="140" />
      <el-table-column prop="icon" label="图标" width="100" />
      <el-table-column prop="flow_id" label="流程ID" min-width="160" show-overflow-tooltip />
      <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
      <el-table-column label="启用" width="100">
        <template #default="{ row }">
          <el-tag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? '是' : '否' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="200" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="openEdit(row)">编辑</el-button>
          <el-button size="small" type="danger" @click="removeType(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog append-to-body v-model="dialogVisible" :title="dialogTitle" width="760px">
      <el-form :model="form" label-width="110px">
        <el-form-item label="名称">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="编码">
          <el-input v-model="form.code" :disabled="isEdit" placeholder="如: change_apply" />
        </el-form-item>
        <el-form-item label="图标">
          <el-input v-model="form.icon" placeholder="如: edit" />
        </el-form-item>
        <el-form-item label="流程ID">
          <el-input v-model="form.flow_id" placeholder="可选" />
        </el-form-item>
        <el-form-item label="表单模板">
          <el-input v-model="form.template" type="textarea" :rows="5" placeholder="JSON 字符串，可选" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="2" />
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
const types = ref([])
const dialogVisible = ref(false)
const dialogTitle = ref('新增类型')
const isEdit = ref(false)
const currentId = ref('')

const form = ref({
  name: '',
  code: '',
  icon: '',
  flow_id: '',
  template: '',
  enabled: true,
  description: ''
})

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const fetchTypes = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/v1/workorder/types', { headers: authHeaders() })
    types.value = res.data?.data || []
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '获取类型失败')
  } finally {
    loading.value = false
  }
}

const openCreate = () => {
  isEdit.value = false
  dialogTitle.value = '新增类型'
  currentId.value = ''
  form.value = {
    name: '',
    code: '',
    icon: '',
    flow_id: '',
    template: '',
    enabled: true,
    description: ''
  }
  dialogVisible.value = true
}

const openEdit = (row) => {
  isEdit.value = true
  dialogTitle.value = '编辑类型'
  currentId.value = row.id
  form.value = {
    name: row.name || '',
    code: row.code || '',
    icon: row.icon || '',
    flow_id: row.flow_id || '',
    template: row.template || '',
    enabled: !!row.enabled,
    description: row.description || ''
  }
  dialogVisible.value = true
}

const submitForm = async () => {
  if (!form.value.name.trim() || !form.value.code.trim()) {
    ElMessage.warning('请填写名称和编码')
    return
  }
  try {
    if (isEdit.value) {
      await axios.put(`/api/v1/workorder/types/${currentId.value}`, form.value, { headers: authHeaders() })
      ElMessage.success('更新成功')
    } else {
      await axios.post('/api/v1/workorder/types', form.value, { headers: authHeaders() })
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    await fetchTypes()
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '保存失败')
  }
}

const removeType = async (row) => {
  try {
    await ElMessageBox.confirm(`确认删除类型 ${row.name} 吗？`, '提示', { type: 'warning' })
    await axios.delete(`/api/v1/workorder/types/${row.id}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    await fetchTypes()
  } catch (err) {
    if (err !== 'cancel') ElMessage.error(err.response?.data?.message || '删除失败')
  }
}

onMounted(fetchTypes)
</script>

<style scoped>
.page-card { max-width: 1280px; margin: 0 auto; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; gap: 12px; margin-bottom: 16px; }
.page-desc { margin: 4px 0 0; color: #606266; }
.page-actions { display: flex; gap: 8px; align-items: center; }
</style>
