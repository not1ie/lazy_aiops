<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>告警聚合配置</h2>
        <p class="page-desc">配置告警聚合的维度与窗口。</p>
      </div>
      <div class="page-actions">
        <el-button type="primary" icon="Plus" @click="openCreate">新增配置</el-button>
        <el-button icon="Refresh" @click="fetchAggs">刷新</el-button>
      </div>
    </div>

    <el-table :fit="true" :data="aggs" stripe style="width: 100%">
      <el-table-column prop="name" label="名称" min-width="160" />
      <el-table-column prop="group_by" label="聚合字段" min-width="220" />
      <el-table-column prop="interval" label="窗口(秒)" width="120" />
      <el-table-column prop="enabled" label="启用" width="90">
        <template #default="scope">
          <el-switch v-model="scope.row.enabled" @change="toggleAgg(scope.row)" />
        </template>
      </el-table-column>
      <el-table-column prop="description" label="描述" min-width="200" />
      <el-table-column label="操作" width="200">
        <template #default="scope">
          <el-button size="small" @click="openEdit(scope.row)">编辑</el-button>
          <el-button size="small" type="danger" @click="removeAgg(scope.row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog append-to-body v-model="dialogVisible" :title="dialogTitle" width="720px" @closed="handleDialogClosed">
      <el-form :model="form" label-width="110px">
        <el-form-item label="名称">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="聚合字段">
          <el-input v-model="form.group_by" placeholder="如: severity,target,metric" />
        </el-form-item>
        <el-form-item label="窗口(秒)">
          <el-input-number v-model="form.interval" :min="10" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
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
import { getErrorMessage, isCancelError } from '@/utils/error'

const aggs = ref([])
const dialogVisible = ref(false)
const dialogTitle = ref('')
const isEdit = ref(false)
const currentId = ref('')
const defaultForm = () => ({ name: '', group_by: '', interval: 60, enabled: true, description: '' })
const form = ref(defaultForm())

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const handleDialogClosed = () => {
  isEdit.value = false
  currentId.value = ''
  dialogTitle.value = '新增聚合'
  form.value = defaultForm()
}

const fetchAggs = async () => {
  try {
    const res = await axios.get('/api/v1/alert/aggregations', { headers: authHeaders() })
    aggs.value = res.data.data || []
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '加载聚合配置失败'))
  }
}

const openCreate = () => {
  isEdit.value = false
  currentId.value = ''
  dialogTitle.value = '新增聚合'
  form.value = defaultForm()
  dialogVisible.value = true
}

const openEdit = (row) => {
  isEdit.value = true
  dialogTitle.value = '编辑聚合'
  currentId.value = row.id
  form.value = { ...row }
  dialogVisible.value = true
}

const submitForm = async () => {
  try {
    if (isEdit.value) {
      await axios.put(`/api/v1/alert/aggregations/${currentId.value}`, form.value, { headers: authHeaders() })
      ElMessage.success('更新成功')
    } else {
      await axios.post('/api/v1/alert/aggregations', form.value, { headers: authHeaders() })
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    await fetchAggs()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '保存聚合配置失败'))
  }
}

const toggleAgg = async (row) => {
  try {
    await axios.put(`/api/v1/alert/aggregations/${row.id}`, row, { headers: authHeaders() })
  } catch (err) {
    row.enabled = !row.enabled
    ElMessage.error(getErrorMessage(err, '切换聚合状态失败'))
  }
}

const removeAgg = async (row) => {
  try {
    await ElMessageBox.confirm(`确认删除聚合 ${row.name} 吗？`, '提示', { type: 'warning' })
    await axios.delete(`/api/v1/alert/aggregations/${row.id}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    await fetchAggs()
  } catch (err) {
    if (!isCancelError(err)) {
      ElMessage.error(getErrorMessage(err, '删除聚合失败'))
    }
  }
}

onMounted(fetchAggs)
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 16px; }
.page-desc { color: #606266; margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; }
</style>
