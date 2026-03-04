<template>
  <div class="executor-page">
    <el-card class="section-card">
      <template #header>
        <div class="header">
          <div>
            <div class="title">批量执行</div>
            <div class="desc">批量执行命令并实时查看结果</div>
          </div>
        </div>
      </template>

      <el-form :model="form" label-width="110px" class="exec-form">
        <el-form-item label="任务名称">
          <el-input v-model="form.name" placeholder="可选" />
        </el-form-item>
        <el-form-item label="命令模板">
          <el-select v-model="selectedTemplate" placeholder="选择模板" clearable style="width: 100%" @change="applyTemplate">
            <el-option v-for="tpl in templates" :key="tpl.id" :label="tpl.name" :value="tpl.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="执行命令" required>
          <el-input v-model="form.content" type="textarea" :rows="4" placeholder="请输入命令" />
        </el-form-item>
        <el-form-item label="目标主机" required>
          <el-select v-model="form.host_ids" multiple filterable placeholder="选择主机" style="width: 100%">
            <el-option v-for="host in hosts" :key="host.id" :label="`${host.name} (${host.ip})`" :value="host.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="超时(秒)">
          <el-input-number v-model="form.timeout" :min="30" :max="3600" />
        </el-form-item>
        <el-form-item label="并发数">
          <el-input-number v-model="form.concurrency" :min="1" :max="50" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="submitting" @click="submitExecution">开始执行</el-button>
          <el-button @click="resetForm">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card class="section-card">
      <template #header>
        <div class="header">
          <div>
            <div class="title">执行记录</div>
            <div class="desc">选择记录查看详细结果</div>
          </div>
          <div class="actions">
            <el-button icon="Refresh" @click="fetchExecutions">刷新</el-button>
          </div>
        </div>
      </template>

      <el-table :fit="true" :data="executions" v-loading="loadingExecutions" @row-click="handleSelectExecution" highlight-current-row>
        <el-table-column prop="name" label="名称" min-width="160" />
        <el-table-column prop="target_count" label="目标数" width="100" />
        <el-table-column prop="progress" label="进度" width="120">
          <template #default="{ row }">
            <el-progress :percentage="row.progress" :status="progressStatus(row.status)" />
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="120">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)">{{ statusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="executor" label="执行人" width="140" />
        <el-table-column prop="started_at" label="开始时间" width="180" />
        <el-table-column label="操作" width="120" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="danger" plain @click.stop="cancelExecution(row)">取消</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card class="section-card" v-if="currentExecution">
      <template #header>
        <div class="header">
          <div>
            <div class="title">执行详情</div>
            <div class="desc">{{ currentExecution.name || '未命名任务' }}</div>
          </div>
          <div class="actions">
            <el-tag :type="statusType(currentExecution.status)">{{ statusLabel(currentExecution.status) }}</el-tag>
          </div>
        </div>
      </template>

      <el-table :fit="true" :data="results" v-loading="loadingResults" stripe>
        <el-table-column prop="host_name" label="主机" min-width="160" />
        <el-table-column prop="host_ip" label="IP" width="140" />
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="resultStatusType(row.status)">{{ resultStatusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="duration" label="耗时" width="100" />
        <el-table-column label="输出" min-width="260">
          <template #default="{ row }">
            <el-button size="small" type="primary" text @click="openOutput(row)">查看输出</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>

  <el-dialog append-to-body v-model="outputVisible" title="执行输出" width="720px">
    <el-tabs v-model="outputTab">
      <el-tab-pane label="Stdout" name="stdout">
        <pre class="log-block">{{ selectedOutput.stdout || '-' }}</pre>
      </el-tab-pane>
      <el-tab-pane label="Stderr" name="stderr">
        <pre class="log-block">{{ selectedOutput.stderr || '-' }}</pre>
      </el-tab-pane>
    </el-tabs>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, onMounted, onBeforeUnmount } from 'vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'

const form = reactive({
  name: '',
  type: 'shell',
  content: '',
  host_ids: [],
  timeout: 300,
  concurrency: 10
})

const hosts = ref([])
const templates = ref([])
const selectedTemplate = ref('')
const submitting = ref(false)

const executions = ref([])
const loadingExecutions = ref(false)
const currentExecution = ref(null)
const results = ref([])
const loadingResults = ref(false)

const outputVisible = ref(false)
const outputTab = ref('stdout')
const selectedOutput = ref({ stdout: '', stderr: '' })

let eventSource = null

const headers = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const fetchHosts = async () => {
  const res = await axios.get('/api/v1/cmdb/hosts', { headers: headers() })
  if (res.data.code === 0) {
    hosts.value = res.data.data
  }
}

const fetchTemplates = async () => {
  const res = await axios.get('/api/v1/executor/templates', { headers: headers() })
  if (res.data.code === 0) {
    templates.value = res.data.data
  }
}

const fetchExecutions = async () => {
  loadingExecutions.value = true
  try {
    const res = await axios.get('/api/v1/executor/executions', { headers: headers() })
    if (res.data.code === 0) {
      executions.value = res.data.data
    }
  } catch (error) {
    ElMessage.error('加载执行记录失败')
  } finally {
    loadingExecutions.value = false
  }
}

const fetchResults = async (executionId) => {
  loadingResults.value = true
  try {
    const res = await axios.get(`/api/v1/executor/executions/${executionId}/results`, { headers: headers() })
    if (res.data.code === 0) {
      results.value = res.data.data
    }
  } catch (error) {
    ElMessage.error('加载执行结果失败')
  } finally {
    loadingResults.value = false
  }
}

const applyTemplate = (id) => {
  const tpl = templates.value.find((item) => item.id === id)
  if (tpl) {
    form.name = tpl.name
    form.content = tpl.content
  }
}

const resetForm = () => {
  form.name = ''
  form.content = ''
  form.host_ids = []
  form.timeout = 300
  form.concurrency = 10
  selectedTemplate.value = ''
}

const submitExecution = async () => {
  if (!form.content || form.host_ids.length === 0) {
    ElMessage.warning('请填写命令并选择目标主机')
    return
  }
  submitting.value = true
  try {
    const res = await axios.post('/api/v1/executor/execute', form, { headers: headers() })
    if (res.data.code === 0) {
      ElMessage.success('任务已提交')
      await fetchExecutions()
      handleSelectExecution(res.data.data)
    }
  } catch (error) {
    ElMessage.error('提交失败')
  } finally {
    submitting.value = false
  }
}

const handleSelectExecution = async (row) => {
  if (!row || !row.id) return
  currentExecution.value = row
  await fetchResults(row.id)
  openStream(row.id)
}

const openStream = (id) => {
  if (eventSource) {
    eventSource.close()
  }
  eventSource = new EventSource(`/api/v1/executor/executions/${id}/stream`)
  eventSource.onmessage = (event) => {
    try {
      const payload = JSON.parse(event.data)
      if (payload.execution) {
        currentExecution.value = payload.execution
      }
      if (payload.results) {
        results.value = payload.results
      }
    } catch (error) {
      // ignore parse errors
    }
  }
  eventSource.onerror = () => {
    eventSource.close()
  }
}

const cancelExecution = async (row) => {
  if (!row || row.status !== 0) return
  await axios.post(`/api/v1/executor/executions/${row.id}/cancel`, {}, { headers: headers() })
  ElMessage.success('已取消')
  fetchExecutions()
}

const openOutput = (row) => {
  selectedOutput.value = { stdout: row.stdout, stderr: row.stderr }
  outputTab.value = 'stdout'
  outputVisible.value = true
}

const statusLabel = (status) => {
  const map = { 0: '运行中', 1: '成功', 2: '部分失败', 3: '失败', 4: '已取消' }
  return map[status] || '未知'
}

const statusType = (status) => {
  if (status === 1) return 'success'
  if (status === 0) return 'warning'
  if (status === 2) return 'warning'
  if (status === 4) return 'info'
  return 'danger'
}

const progressStatus = (status) => {
  if (status === 1) return 'success'
  if (status === 3) return 'exception'
  if (status === 4) return 'warning'
  return ''
}

const resultStatusLabel = (status) => {
  const map = { 0: '等待', 1: '运行中', 2: '成功', 3: '失败', 4: '超时' }
  return map[status] || '未知'
}

const resultStatusType = (status) => {
  if (status === 2) return 'success'
  if (status === 1) return 'warning'
  if (status === 4) return 'info'
  if (status === 0) return ''
  return 'danger'
}

onMounted(async () => {
  await fetchHosts()
  await fetchTemplates()
  await fetchExecutions()
})

onBeforeUnmount(() => {
  if (eventSource) {
    eventSource.close()
  }
})
</script>

<style scoped>
.executor-page { display: flex; flex-direction: column; gap: 16px; }
.section-card { max-width: 100%; margin: 0; }
.header { display: flex; justify-content: space-between; align-items: center; }
.title { font-size: 18px; font-weight: 600; }
.desc { color: #909399; margin-top: 4px; }
.exec-form { max-width: 880px; }
.log-block { background: #0f172a; color: #e2e8f0; padding: 12px; border-radius: 6px; font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace; font-size: 12px; white-space: pre-wrap; }
</style>
