<template>
  <el-card class="page-card">
    <template #header>
      <div class="header">
        <div>
          <div class="title">构建历史</div>
          <div class="desc">流水线执行记录与日志</div>
        </div>
        <div class="actions">
          <el-button icon="Refresh" @click="fetchData">刷新</el-button>
        </div>
      </div>
    </template>

    <el-table :fit="false" :data="executions" v-loading="loading" stripe>
      <el-table-column prop="pipeline_name" label="流水线" min-width="200" />
      <el-table-column prop="provider" label="Provider" width="120" />
      <el-table-column prop="status" label="状态" width="120">
        <template #default="{ row }">
          <el-tag :type="statusType(row.status)">{{ statusLabel(row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="trigger" label="触发方式" width="120" />
      <el-table-column prop="trigger_by" label="触发人" width="120" />
      <el-table-column prop="started_at" label="开始时间" width="180" />
      <el-table-column prop="finished_at" label="结束时间" width="180" />
      <el-table-column prop="duration" label="耗时" width="100" />
      <el-table-column label="操作" width="200" fixed="right">
        <template #default="{ row }">
          <el-button size="small" type="primary" text @click="openLogs(row)">日志</el-button>
          <el-button size="small" type="danger" text @click="cancelExecution(row)">取消</el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-card>

  <el-dialog append-to-body v-model="logVisible" title="执行日志" width="760px">
    <pre class="log-block">{{ logText }}</pre>
  </el-dialog>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'

const executions = ref([])
const loading = ref(false)
const logVisible = ref(false)
const logText = ref('')

const headers = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const fetchData = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/v1/cicd/executions', { headers: headers() })
    if (res.data.code === 0) {
      executions.value = res.data.data
    }
  } catch (error) {
    ElMessage.error('加载失败')
  } finally {
    loading.value = false
  }
}

const openLogs = async (row) => {
  logText.value = ''
  const res = await axios.get(`/api/v1/cicd/executions/${row.id}/logs`, { headers: headers() })
  if (res.data.code === 0) {
    logText.value = res.data.data.logs || '-'
  }
  logVisible.value = true
}

const cancelExecution = async (row) => {
  if (row.status !== 0) {
    return
  }
  await axios.post(`/api/v1/cicd/executions/${row.id}/cancel`, {}, { headers: headers() })
  ElMessage.success('已取消')
  fetchData()
}

const statusLabel = (status) => {
  const map = { 0: '运行中', 1: '成功', 2: '失败', 3: '取消' }
  return map[status] || '未知'
}

const statusType = (status) => {
  if (status === 1) return 'success'
  if (status === 0) return 'warning'
  if (status === 3) return 'info'
  return 'danger'
}

onMounted(fetchData)
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.header { display: flex; justify-content: space-between; align-items: center; }
.title { font-size: 18px; font-weight: 600; }
.desc { color: #909399; margin-top: 4px; }
.actions { display: flex; gap: 8px; }
.log-block { background: #0f172a; color: #e2e8f0; padding: 12px; border-radius: 6px; font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace; font-size: 12px; white-space: pre-wrap; }
</style>
