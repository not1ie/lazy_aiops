<template>
  <el-card class="page-card">
    <template #header>
      <div class="header">
        <div>
          <div class="title">定时发布</div>
          <div class="desc">按计划触发流水线执行</div>
        </div>
        <div class="actions">
          <el-button type="primary" icon="Plus" @click="openCreate">新增计划</el-button>
          <el-button icon="Refresh" @click="fetchData">刷新</el-button>
        </div>
      </div>
    </template>

    <el-table :data="schedules" v-loading="loading" stripe>
      <el-table-column prop="name" label="名称" min-width="180" />
      <el-table-column prop="pipeline_id" label="流水线" min-width="200">
        <template #default="{ row }">
          {{ pipelineName(row.pipeline_id) }}
        </template>
      </el-table-column>
      <el-table-column prop="cron" label="Cron" min-width="180" />
      <el-table-column prop="enabled" label="状态" width="120">
        <template #default="{ row }">
          <el-tag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? '启用' : '停用' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="next_run_at" label="下次执行" width="180" />
      <el-table-column label="操作" width="240" fixed="right">
        <template #default="{ row }">
          <el-button size="small" type="primary" plain @click="openEdit(row)">编辑</el-button>
          <el-button size="small" type="warning" plain @click="toggle(row)">{{ row.enabled ? '停用' : '启用' }}</el-button>
          <el-button size="small" type="danger" plain @click="handleDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-card>

  <el-dialog append-to-body v-model="dialogVisible" :title="isEdit ? '编辑计划' : '新增计划'" width="600px">
    <el-form :model="form" label-width="100px">
      <el-form-item label="名称" required>
        <el-input v-model="form.name" />
      </el-form-item>
      <el-form-item label="流水线" required>
        <el-select v-model="form.pipeline_id" style="width: 100%">
          <el-option v-for="pipe in pipelines" :key="pipe.id" :label="pipe.name" :value="pipe.id" />
        </el-select>
      </el-form-item>
      <el-form-item label="Cron" required>
        <el-input v-model="form.cron" placeholder="0 0 * * *" />
      </el-form-item>
      <el-form-item label="参数(JSON)">
        <el-input v-model="form.parameters" type="textarea" :rows="3" placeholder='{"env":"prod"}' />
      </el-form-item>
      <el-form-item label="描述">
        <el-input v-model="form.description" />
      </el-form-item>
      <el-form-item label="启用">
        <el-switch v-model="form.enabled" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="dialogVisible = false">取消</el-button>
      <el-button type="primary" :loading="saving" @click="saveSchedule">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const schedules = ref([])
const pipelines = ref([])
const loading = ref(false)
const saving = ref(false)
const dialogVisible = ref(false)
const isEdit = ref(false)
const currentId = ref('')

const form = reactive({
  name: '',
  pipeline_id: '',
  cron: '',
  parameters: '',
  description: '',
  enabled: true
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
    const res = await axios.get('/api/v1/cicd/schedules', { headers: headers() })
    if (res.data.code === 0) {
      schedules.value = res.data.data
    }
  } catch (error) {
    ElMessage.error('加载失败')
  } finally {
    loading.value = false
  }
}

const pipelineName = (id) => {
  const target = pipelines.value.find((item) => item.id === id)
  return target ? target.name : id
}

const openCreate = async () => {
  await fetchPipelines()
  isEdit.value = false
  currentId.value = ''
  Object.assign(form, { name: '', pipeline_id: '', cron: '', parameters: '', description: '', enabled: true })
  dialogVisible.value = true
}

const openEdit = async (row) => {
  await fetchPipelines()
  isEdit.value = true
  currentId.value = row.id
  Object.assign(form, row)
  dialogVisible.value = true
}

const saveSchedule = async () => {
  if (!form.name || !form.pipeline_id || !form.cron) {
    ElMessage.warning('请补全必填项')
    return
  }
  saving.value = true
  try {
    const url = isEdit.value ? `/api/v1/cicd/schedules/${currentId.value}` : '/api/v1/cicd/schedules'
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

const toggle = async (row) => {
  await axios.post(`/api/v1/cicd/schedules/${row.id}/toggle`, {}, { headers: headers() })
  ElMessage.success('已更新状态')
  fetchData()
}

const handleDelete = (row) => {
  ElMessageBox.confirm(`确定删除“${row.name}”吗？`, '提示', { type: 'warning' }).then(async () => {
    await axios.delete(`/api/v1/cicd/schedules/${row.id}`, { headers: headers() })
    ElMessage.success('删除成功')
    fetchData()
  })
}

onMounted(async () => {
  await fetchPipelines()
  await fetchData()
})
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.header { display: flex; justify-content: space-between; align-items: center; }
.title { font-size: 18px; font-weight: 600; }
.desc { color: #909399; margin-top: 4px; }
.actions { display: flex; gap: 8px; }
</style>
