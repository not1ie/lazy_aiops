<template>
  <div class="task-page">
    <el-card class="section-card">
      <template #header>
        <div class="header">
          <div>
            <div class="title">任务调度</div>
            <div class="desc">定时任务与执行历史</div>
          </div>
          <div class="actions">
            <el-button type="primary" icon="Plus" @click="openCreate">新增任务</el-button>
            <el-button icon="Refresh" @click="fetchTasks">刷新</el-button>
          </div>
        </div>
      </template>

      <el-table :fit="true" :data="tasks" v-loading="loading" @row-click="selectTask" highlight-current-row>
        <el-table-column prop="name" label="任务名称" min-width="180" />
        <el-table-column prop="type" label="类型" width="120" />
        <el-table-column prop="cron" label="Cron" min-width="180" />
        <el-table-column prop="enabled" label="启用" width="100">
          <template #default="{ row }">
            <el-tag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? '启用' : '停用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="last_run_at" label="上次执行" width="180" />
        <el-table-column prop="next_run_at" label="下次执行" width="180" />
        <el-table-column label="操作" width="280" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="primary" plain @click.stop="openEdit(row)">编辑</el-button>
            <el-button size="small" type="success" plain @click.stop="runTask(row)">执行</el-button>
            <el-button size="small" type="warning" plain @click.stop="toggleTask(row)">{{ row.enabled ? '停用' : '启用' }}</el-button>
            <el-button size="small" type="danger" plain @click.stop="deleteTask(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card class="section-card">
      <template #header>
        <div class="header">
          <div>
            <div class="title">执行记录</div>
            <div class="desc">{{ currentTask ? currentTask.name : '全部任务' }}</div>
          </div>
          <div class="actions">
            <el-button icon="Refresh" @click="fetchExecutions">刷新</el-button>
          </div>
        </div>
      </template>

      <el-table :fit="true" :data="executions" v-loading="loadingExecutions" stripe>
        <el-table-column prop="task_name" label="任务" min-width="180" />
        <el-table-column prop="status" label="状态" width="120">
          <template #default="{ row }">
            <el-tag :type="executionStatusType(row.status)">{{ executionStatusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="start_at" label="开始时间" width="180" />
        <el-table-column prop="end_at" label="结束时间" width="180" />
        <el-table-column prop="duration" label="耗时" width="100" />
        <el-table-column prop="executor" label="执行人" width="120" />
      </el-table>
    </el-card>
  </div>

  <el-dialog append-to-body v-model="dialogVisible" :title="isEdit ? '编辑任务' : '新增任务'" width="640px" @closed="handleDialogClosed">
    <el-form :model="form" label-width="110px">
      <el-form-item label="任务名称" required>
        <el-input v-model="form.name" />
      </el-form-item>
      <el-form-item label="任务类型" required>
        <el-select v-model="form.type" style="width: 100%">
          <el-option label="Shell" value="shell" />
          <el-option label="Python" value="python" />
          <el-option label="Ansible" value="ansible" />
        </el-select>
      </el-form-item>
      <el-form-item label="任务内容" required>
        <el-input v-model="form.content" type="textarea" :rows="4" placeholder="脚本/命令内容" />
      </el-form-item>
      <el-form-item label="目标主机">
        <el-select v-model="form.targets" multiple filterable placeholder="选择主机" style="width: 100%">
          <el-option v-for="host in hosts" :key="host.id" :label="`${host.name} (${host.ip})`" :value="host.id" />
        </el-select>
      </el-form-item>
      <el-form-item label="Cron">
        <el-input v-model="form.cron" placeholder="如：0 */5 * * * *" />
      </el-form-item>
      <el-form-item label="超时(秒)">
        <el-input-number v-model="form.timeout" :min="30" :max="3600" />
      </el-form-item>
      <el-form-item label="启用">
        <el-switch v-model="form.enabled" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="dialogVisible = false">取消</el-button>
      <el-button type="primary" :loading="saving" @click="saveTask">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const tasks = ref([])
const executions = ref([])
const loading = ref(false)
const loadingExecutions = ref(false)
const dialogVisible = ref(false)
const isEdit = ref(false)
const currentId = ref('')
const currentTask = ref(null)
const saving = ref(false)
const hosts = ref([])

const form = reactive({
  name: '',
  type: 'shell',
  content: '',
  targets: [],
  cron: '',
  timeout: 300,
  enabled: true
})

const resetForm = () => {
  Object.assign(form, {
    name: '',
    type: 'shell',
    content: '',
    targets: [],
    cron: '',
    timeout: 300,
    enabled: true
  })
}

const headers = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const getErrorMessage = (error, fallback) => {
  if (error?.response?.data?.message) return error.response.data.message
  if (error?.message) return error.message
  return fallback
}

const fetchHosts = async () => {
  try {
    const res = await axios.get('/api/v1/cmdb/hosts', { headers: headers() })
    if (res.data.code === 0) {
      hosts.value = res.data.data
    }
  } catch (error) {
    ElMessage.error(getErrorMessage(error, '加载主机失败'))
  }
}

const fetchTasks = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/v1/task/tasks', { headers: headers() })
    if (res.data.code === 0) {
      tasks.value = res.data.data
      if (currentTask.value?.id) {
        currentTask.value = tasks.value.find(item => item.id === currentTask.value.id) || null
      }
    }
  } catch (error) {
    ElMessage.error(getErrorMessage(error, '加载任务失败'))
  } finally {
    loading.value = false
  }
}

const fetchExecutions = async () => {
  loadingExecutions.value = true
  try {
    const res = await axios.get('/api/v1/task/executions', {
      headers: headers(),
      params: { task_id: currentTask.value ? currentTask.value.id : '' }
    })
    if (res.data.code === 0) {
      executions.value = res.data.data
    }
  } catch (error) {
    ElMessage.error(getErrorMessage(error, '加载执行记录失败'))
  } finally {
    loadingExecutions.value = false
  }
}

const selectTask = (row) => {
  currentTask.value = row
  fetchExecutions()
}

const openCreate = () => {
  isEdit.value = false
  currentId.value = ''
  resetForm()
  dialogVisible.value = true
}

const openEdit = (row) => {
  isEdit.value = true
  currentId.value = row.id
  Object.assign(form, {
    name: row.name,
    type: row.type,
    content: row.content,
    targets: row.targets ? row.targets.split(',') : [],
    cron: row.cron,
    timeout: row.timeout,
    enabled: row.enabled
  })
  dialogVisible.value = true
}

const saveTask = async () => {
  if (!form.name || !form.content) {
    ElMessage.warning('请填写任务名称与内容')
    return
  }
  saving.value = true
  try {
    const payload = {
      ...form,
      targets: form.targets.join(',')
    }
    const url = isEdit.value ? `/api/v1/task/tasks/${currentId.value}` : '/api/v1/task/tasks'
    const method = isEdit.value ? 'put' : 'post'
    const res = await axios({ url, method, data: payload, headers: headers() })
    if (res.data.code === 0) {
      ElMessage.success(isEdit.value ? '更新成功' : '创建成功')
      dialogVisible.value = false
      await fetchTasks()
    }
  } catch (error) {
    ElMessage.error(getErrorMessage(error, '保存失败'))
  } finally {
    saving.value = false
  }
}

const handleDialogClosed = () => {
  isEdit.value = false
  currentId.value = ''
  resetForm()
}

const runTask = async (row) => {
  try {
    await axios.post(`/api/v1/task/tasks/${row.id}/run`, {}, { headers: headers() })
    ElMessage.success('已提交执行')
    await fetchExecutions()
  } catch (error) {
    ElMessage.error(getErrorMessage(error, '执行失败'))
  }
}

const toggleTask = async (row) => {
  const action = row.enabled ? 'disable' : 'enable'
  try {
    await axios.post(`/api/v1/task/tasks/${row.id}/${action}`, {}, { headers: headers() })
    ElMessage.success(row.enabled ? '已停用' : '已启用')
    await fetchTasks()
  } catch (error) {
    ElMessage.error(getErrorMessage(error, '更新状态失败'))
  }
}

const deleteTask = async (row) => {
  try {
    await ElMessageBox.confirm(`确定删除任务“${row.name}”吗？`, '提示', { type: 'warning' })
    await axios.delete(`/api/v1/task/tasks/${row.id}`, { headers: headers() })
    ElMessage.success('删除成功')
    const deletingCurrentTask = currentTask.value?.id === row.id
    await fetchTasks()
    if (deletingCurrentTask) {
      currentTask.value = null
      await fetchExecutions()
    }
  } catch (error) {
    if (error !== 'cancel' && error !== 'close') {
      ElMessage.error(getErrorMessage(error, '删除失败'))
    }
  }
}

const executionStatusLabel = (status) => {
  const map = { 0: '运行中', 1: '成功', 2: '失败', 3: '超时' }
  return map[status] || '未知'
}

const executionStatusType = (status) => {
  if (status === 1) return 'success'
  if (status === 0) return 'warning'
  if (status === 3) return 'info'
  return 'danger'
}

onMounted(async () => {
  await fetchHosts()
  await fetchTasks()
  await fetchExecutions()
})
</script>

<style scoped>
.task-page { display: flex; flex-direction: column; gap: 16px; }
.section-card { max-width: 100%; margin: 0; }
.header { display: flex; justify-content: space-between; align-items: center; }
.title { font-size: 18px; font-weight: 600; }
.desc { color: #909399; margin-top: 4px; }
.actions { display: flex; gap: 8px; }
</style>
