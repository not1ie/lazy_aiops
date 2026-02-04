<template>
  <el-card class="page-card">
    <template #header>
      <div class="header">
        <div>
          <div class="title">发布管理</div>
          <div class="desc">发布计划与记录</div>
        </div>
        <div class="actions">
          <el-button type="primary" icon="Plus" @click="openCreate">新增发布</el-button>
          <el-button icon="Refresh" @click="fetchData">刷新</el-button>
        </div>
      </div>
    </template>

    <el-table :data="releases" v-loading="loading" stripe>
      <el-table-column prop="name" label="名称" min-width="160" />
      <el-table-column prop="pipeline_name" label="流水线" min-width="180" />
      <el-table-column prop="version" label="版本" width="140" />
      <el-table-column prop="environment" label="环境" width="120" />
      <el-table-column prop="status" label="状态" width="120">
        <template #default="{ row }">
          <el-tag :type="statusType(row.status)">{{ statusLabel(row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="release_at" label="发布时间" width="180" />
      <el-table-column prop="operator" label="操作人" width="120" />
      <el-table-column label="操作" width="200" fixed="right">
        <template #default="{ row }">
          <el-button size="small" type="primary" plain @click="openEdit(row)">编辑</el-button>
          <el-button size="small" type="danger" plain @click="handleDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-card>

  <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑发布' : '新增发布'" width="600px">
    <el-form :model="form" label-width="100px">
      <el-form-item label="名称" required>
        <el-input v-model="form.name" />
      </el-form-item>
      <el-form-item label="流水线" required>
        <el-select v-model="form.pipeline_id" style="width: 100%" @change="syncPipelineName">
          <el-option v-for="pipe in pipelines" :key="pipe.id" :label="pipe.name" :value="pipe.id" />
        </el-select>
      </el-form-item>
      <el-form-item label="版本">
        <el-input v-model="form.version" placeholder="v1.0.0" />
      </el-form-item>
      <el-form-item label="环境">
        <el-select v-model="form.environment" style="width: 100%">
          <el-option label="开发" value="dev" />
          <el-option label="测试" value="test" />
          <el-option label="生产" value="prod" />
        </el-select>
      </el-form-item>
      <el-form-item label="状态">
        <el-select v-model="form.status" style="width: 100%">
          <el-option label="待发布" :value="0" />
          <el-option label="已发布" :value="1" />
          <el-option label="已回滚" :value="2" />
          <el-option label="失败" :value="3" />
        </el-select>
      </el-form-item>
      <el-form-item label="备注">
        <el-input v-model="form.notes" type="textarea" :rows="3" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="dialogVisible = false">取消</el-button>
      <el-button type="primary" :loading="saving" @click="saveRelease">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const releases = ref([])
const pipelines = ref([])
const loading = ref(false)
const saving = ref(false)
const dialogVisible = ref(false)
const isEdit = ref(false)
const currentId = ref('')

const form = reactive({
  name: '',
  pipeline_id: '',
  pipeline_name: '',
  version: '',
  environment: 'dev',
  status: 0,
  notes: ''
})

const headers = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const fetchPipelines = async () => {
  const res = await axios.get('/api/v1/cicd/pipelines', { headers: headers() })
  if (res.data.code === 0) {
    pipelines.value = res.data.data
  }
}

const fetchData = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/v1/cicd/releases', { headers: headers() })
    if (res.data.code === 0) {
      releases.value = res.data.data
    }
  } catch (error) {
    ElMessage.error('加载失败')
  } finally {
    loading.value = false
  }
}

const syncPipelineName = () => {
  const pipe = pipelines.value.find((item) => item.id === form.pipeline_id)
  form.pipeline_name = pipe ? pipe.name : ''
}

const openCreate = async () => {
  await fetchPipelines()
  isEdit.value = false
  currentId.value = ''
  Object.assign(form, { name: '', pipeline_id: '', pipeline_name: '', version: '', environment: 'dev', status: 0, notes: '' })
  dialogVisible.value = true
}

const openEdit = async (row) => {
  await fetchPipelines()
  isEdit.value = true
  currentId.value = row.id
  Object.assign(form, row)
  dialogVisible.value = true
}

const saveRelease = async () => {
  if (!form.name || !form.pipeline_id) {
    ElMessage.warning('请填写名称和流水线')
    return
  }
  saving.value = true
  try {
    const url = isEdit.value ? `/api/v1/cicd/releases/${currentId.value}` : '/api/v1/cicd/releases'
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

const handleDelete = (row) => {
  ElMessageBox.confirm(`确定删除“${row.name}”吗？`, '提示', { type: 'warning' }).then(async () => {
    await axios.delete(`/api/v1/cicd/releases/${row.id}`, { headers: headers() })
    ElMessage.success('删除成功')
    fetchData()
  })
}

const statusLabel = (status) => {
  const map = { 0: '待发布', 1: '已发布', 2: '已回滚', 3: '失败' }
  return map[status] || '未知'
}

const statusType = (status) => {
  if (status === 1) return 'success'
  if (status === 0) return 'warning'
  if (status === 2) return 'info'
  return 'danger'
}

onMounted(async () => {
  await fetchPipelines()
  await fetchData()
})
</script>

<style scoped>
.page-card { max-width: 1200px; margin: 0 auto; }
.header { display: flex; justify-content: space-between; align-items: center; }
.title { font-size: 18px; font-weight: 600; }
.desc { color: #909399; margin-top: 4px; }
.actions { display: flex; gap: 8px; }
</style>
