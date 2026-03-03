<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>工作流编排</h2>
        <p class="page-desc">维护流程定义、模板和执行记录。</p>
      </div>
      <div class="page-actions">
        <el-button type="primary" icon="Plus" @click="openWorkflowDialog()">新建流程</el-button>
        <el-button icon="Refresh" @click="refreshAll">刷新</el-button>
      </div>
    </div>

    <el-row :gutter="12" class="mb-12">
      <el-col :span="6"><el-card><div class="k">总执行数</div><div class="v">{{ stats.total || 0 }}</div></el-card></el-col>
      <el-col :span="6"><el-card><div class="k">运行中</div><div class="v info">{{ stats.running || 0 }}</div></el-card></el-col>
      <el-col :span="6"><el-card><div class="k">成功</div><div class="v success">{{ stats.success || 0 }}</div></el-card></el-col>
      <el-col :span="6"><el-card><div class="k">失败</div><div class="v danger">{{ stats.failed || 0 }}</div></el-card></el-col>
    </el-row>

    <el-tabs v-model="activeTab">
      <el-tab-pane label="流程定义" name="workflows">
        <el-row :gutter="12">
          <el-col :md="16" :sm="24">
            <el-table :data="workflows" v-loading="loading" stripe>
              <el-table-column prop="name" label="名称" min-width="160" />
              <el-table-column prop="category" label="分类" width="120" />
              <el-table-column prop="trigger" label="触发方式" width="120" />
              <el-table-column label="状态" width="90">
                <template #default="{ row }">
                  <el-tag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? '启用' : '停用' }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="version" label="版本" width="80" />
              <el-table-column prop="description" label="描述" min-width="220" show-overflow-tooltip />
              <el-table-column label="操作" width="290" fixed="right">
                <template #default="{ row }">
                  <el-button size="small" @click="openWorkflowDialog(row)">编辑</el-button>
                  <el-button size="small" type="primary" plain @click="openExecuteDialog(row)">执行</el-button>
                  <el-button size="small" type="success" plain @click="openExecutionList(row)">记录</el-button>
                  <el-button size="small" type="danger" plain @click="removeWorkflow(row)">删除</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-col>
          <el-col :md="8" :sm="24">
            <el-card>
              <template #header>
                <span>流程模板</span>
              </template>
              <el-scrollbar max-height="460px">
                <div v-for="tpl in templates" :key="tpl.id" class="tpl-item">
                  <div class="tpl-title">{{ tpl.name }}</div>
                  <div class="tpl-desc">{{ tpl.description || '-' }}</div>
                  <div class="tpl-meta">{{ tpl.category }}</div>
                  <el-button size="small" type="primary" plain @click="createFromTemplate(tpl)">从模板创建</el-button>
                </div>
              </el-scrollbar>
            </el-card>
          </el-col>
        </el-row>
      </el-tab-pane>

      <el-tab-pane label="执行记录" name="executions">
        <div class="toolbar">
          <el-select v-model="executionFilter.workflow_id" clearable placeholder="流程" style="width: 240px">
            <el-option v-for="item in workflows" :key="item.id" :label="item.name" :value="item.id" />
          </el-select>
          <el-select v-model="executionFilter.status" clearable placeholder="状态" style="width: 120px">
            <el-option label="运行中" value="0" />
            <el-option label="成功" value="1" />
            <el-option label="失败" value="2" />
            <el-option label="取消" value="3" />
          </el-select>
          <el-button type="primary" icon="Search" @click="fetchExecutions">查询</el-button>
          <el-button icon="Refresh" @click="fetchExecutions">刷新</el-button>
        </div>

        <el-table :data="executions" v-loading="executionsLoading" stripe>
          <el-table-column prop="workflow_name" label="流程" min-width="180" />
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <el-tag :type="executionStatusType(row.status)">{{ executionStatusText(row.status) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="trigger" label="触发" width="100" />
          <el-table-column prop="trigger_by" label="触发人" width="120" />
          <el-table-column label="开始时间" width="180">
            <template #default="{ row }">{{ formatTime(row.started_at) }}</template>
          </el-table-column>
          <el-table-column label="结束时间" width="180">
            <template #default="{ row }">{{ formatTime(row.finished_at) }}</template>
          </el-table-column>
          <el-table-column prop="error" label="错误信息" min-width="200" show-overflow-tooltip />
          <el-table-column label="操作" width="180" fixed="right">
            <template #default="{ row }">
              <el-button size="small" @click="showExecutionDetail(row)">详情</el-button>
              <el-button size="small" type="danger" plain v-if="Number(row.status) === 0" @click="cancelExecution(row)">取消</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
    </el-tabs>

    <el-dialog append-to-body v-model="workflowDialogVisible" :title="workflowEditing ? '编辑流程' : '新建流程'" width="860px">
      <el-form :model="workflowForm" label-width="96px">
        <el-row :gutter="12">
          <el-col :span="12"><el-form-item label="流程名称" required><el-input v-model="workflowForm.name" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="分类"><el-select v-model="workflowForm.category" style="width: 100%"><el-option label="deploy" value="deploy" /><el-option label="monitor" value="monitor" /><el-option label="backup" value="backup" /><el-option label="custom" value="custom" /></el-select></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="触发方式"><el-select v-model="workflowForm.trigger" style="width: 100%"><el-option label="manual" value="manual" /><el-option label="schedule" value="schedule" /><el-option label="webhook" value="webhook" /><el-option label="alert" value="alert" /></el-select></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="状态"><el-switch v-model="workflowForm.enabled" /></el-form-item></el-col>
          <el-col :span="24"><el-form-item label="描述"><el-input v-model="workflowForm.description" /></el-form-item></el-col>
          <el-col :span="24"><el-form-item label="流程定义" required><el-input v-model="workflowForm.definition" type="textarea" :rows="10" /></el-form-item></el-col>
          <el-col :span="24"><el-form-item label="默认变量"><el-input v-model="workflowForm.variables" type="textarea" :rows="4" placeholder='{"service":"nginx"}' /></el-form-item></el-col>
        </el-row>
      </el-form>
      <template #footer>
        <el-button @click="validateWorkflow">验证定义</el-button>
        <el-button @click="workflowDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="workflowSaving" @click="saveWorkflow">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog append-to-body v-model="executeDialogVisible" title="执行流程" width="600px">
      <el-form :model="executeForm" label-width="90px">
        <el-form-item label="流程">
          <el-input :model-value="executeWorkflow?.name || '-'" disabled />
        </el-form-item>
        <el-form-item label="运行变量">
          <el-input v-model="executeForm.variables" type="textarea" :rows="8" placeholder='{"service":"nginx"}' />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="executeDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="executing" @click="executeWorkflowNow">执行</el-button>
      </template>
    </el-dialog>

    <el-drawer v-model="executionDetailVisible" title="执行详情" size="55%" append-to-body>
      <el-descriptions :column="2" border v-if="executionDetail.execution">
        <el-descriptions-item label="流程">{{ executionDetail.execution.workflow_name }}</el-descriptions-item>
        <el-descriptions-item label="状态">{{ executionStatusText(executionDetail.execution.status) }}</el-descriptions-item>
        <el-descriptions-item label="触发方式">{{ executionDetail.execution.trigger }}</el-descriptions-item>
        <el-descriptions-item label="触发人">{{ executionDetail.execution.trigger_by }}</el-descriptions-item>
        <el-descriptions-item label="开始时间">{{ formatTime(executionDetail.execution.started_at) }}</el-descriptions-item>
        <el-descriptions-item label="结束时间">{{ formatTime(executionDetail.execution.finished_at) }}</el-descriptions-item>
        <el-descriptions-item label="错误" :span="2">{{ executionDetail.execution.error || '-' }}</el-descriptions-item>
      </el-descriptions>

      <el-divider>节点执行</el-divider>
      <el-timeline>
        <el-timeline-item v-for="node in executionDetail.nodes || []" :key="node.id" :timestamp="formatTime(node.started_at)" :type="nodeStatusType(node.status)">
          <div class="node-title">{{ node.node_name }} ({{ node.node_type }})</div>
          <div class="muted">状态：{{ nodeStatusText(node.status) }}</div>
          <div class="muted" v-if="node.error">错误：{{ node.error }}</div>
          <div class="muted" v-if="node.output">输出：{{ node.output }}</div>
        </el-timeline-item>
      </el-timeline>
    </el-drawer>
  </el-card>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const loading = ref(false)
const executionsLoading = ref(false)
const workflowSaving = ref(false)
const executing = ref(false)

const activeTab = ref('workflows')
const workflows = ref([])
const templates = ref([])
const executions = ref([])
const stats = ref({ total: 0, running: 0, success: 0, failed: 0 })

const workflowDialogVisible = ref(false)
const workflowEditing = ref(false)
const executeDialogVisible = ref(false)
const executionDetailVisible = ref(false)

const executeWorkflow = ref(null)
const executionDetail = ref({ execution: null, nodes: [] })

const executionFilter = reactive({ workflow_id: '', status: '' })

const workflowForm = reactive({
  id: '',
  name: '',
  description: '',
  category: 'custom',
  definition: '{"nodes":[{"id":"start","type":"start","name":"开始","next":["end"]},{"id":"end","type":"end","name":"结束"}]}',
  variables: '{}',
  trigger: 'manual',
  enabled: true
})

const executeForm = reactive({
  variables: '{}'
})

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const formatTime = (value) => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString()
}

const executionStatusText = (status) => {
  const n = Number(status)
  if (n === 0) return '运行中'
  if (n === 1) return '成功'
  if (n === 2) return '失败'
  if (n === 3) return '取消'
  if (n === 4) return '等待审批'
  return '-'
}

const executionStatusType = (status) => {
  const n = Number(status)
  if (n === 1) return 'success'
  if (n === 2) return 'danger'
  if (n === 3) return 'warning'
  if (n === 0) return 'primary'
  return 'info'
}

const nodeStatusText = (status) => {
  const n = Number(status)
  if (n === 0) return '运行中'
  if (n === 1) return '成功'
  if (n === 2) return '失败'
  if (n === 3) return '跳过'
  return '-'
}

const nodeStatusType = (status) => {
  const n = Number(status)
  if (n === 1) return 'success'
  if (n === 2) return 'danger'
  if (n === 3) return 'warning'
  return 'primary'
}

const fetchStats = async () => {
  const res = await axios.get('/api/v1/workflow/stats', { headers: authHeaders() })
  if (res.data?.code === 0) stats.value = res.data.data || stats.value
}

const fetchWorkflows = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/v1/workflow/workflows', { headers: authHeaders() })
    if (res.data?.code === 0) workflows.value = res.data.data || []
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '加载流程失败')
  } finally {
    loading.value = false
  }
}

const fetchTemplates = async () => {
  try {
    const res = await axios.get('/api/v1/workflow/templates', { headers: authHeaders() })
    if (res.data?.code === 0) templates.value = res.data.data || []
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '加载模板失败')
  }
}

const fetchExecutions = async () => {
  executionsLoading.value = true
  try {
    const res = await axios.get('/api/v1/workflow/executions', {
      headers: authHeaders(),
      params: {
        workflow_id: executionFilter.workflow_id || undefined,
        status: executionFilter.status || undefined
      }
    })
    if (res.data?.code === 0) executions.value = res.data.data || []
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '加载执行记录失败')
  } finally {
    executionsLoading.value = false
  }
}

const resetWorkflowForm = () => {
  workflowForm.id = ''
  workflowForm.name = ''
  workflowForm.description = ''
  workflowForm.category = 'custom'
  workflowForm.definition = '{"nodes":[{"id":"start","type":"start","name":"开始","next":["end"]},{"id":"end","type":"end","name":"结束"}]}'
  workflowForm.variables = '{}'
  workflowForm.trigger = 'manual'
  workflowForm.enabled = true
}

const openWorkflowDialog = (row) => {
  workflowEditing.value = !!row
  resetWorkflowForm()
  if (row) {
    workflowForm.id = row.id
    workflowForm.name = row.name || ''
    workflowForm.description = row.description || ''
    workflowForm.category = row.category || 'custom'
    workflowForm.definition = row.definition || workflowForm.definition
    workflowForm.variables = row.variables || '{}'
    workflowForm.trigger = row.trigger || 'manual'
    workflowForm.enabled = !!row.enabled
  }
  workflowDialogVisible.value = true
}

const validateWorkflow = async () => {
  try {
    const res = await axios.post('/api/v1/workflow/validate', {
      definition: workflowForm.definition
    }, { headers: authHeaders() })
    ElMessage.success(`验证通过，节点数 ${res.data?.data?.node_count || 0}`)
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '验证失败')
  }
}

const saveWorkflow = async () => {
  if (!workflowForm.name.trim()) {
    ElMessage.warning('请输入流程名称')
    return
  }
  try {
    JSON.parse(workflowForm.definition || '{}')
  } catch {
    ElMessage.warning('流程定义不是合法JSON')
    return
  }
  try {
    JSON.parse(workflowForm.variables || '{}')
  } catch {
    ElMessage.warning('默认变量不是合法JSON')
    return
  }

  workflowSaving.value = true
  try {
    const payload = {
      name: workflowForm.name.trim(),
      description: workflowForm.description,
      category: workflowForm.category,
      definition: workflowForm.definition,
      variables: workflowForm.variables,
      trigger: workflowForm.trigger,
      enabled: workflowForm.enabled
    }
    if (workflowEditing.value && workflowForm.id) {
      await axios.put(`/api/v1/workflow/workflows/${workflowForm.id}`, payload, { headers: authHeaders() })
      ElMessage.success('更新成功')
    } else {
      await axios.post('/api/v1/workflow/workflows', payload, { headers: authHeaders() })
      ElMessage.success('创建成功')
    }
    workflowDialogVisible.value = false
    await fetchWorkflows()
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '保存失败')
  } finally {
    workflowSaving.value = false
  }
}

const removeWorkflow = async (row) => {
  try {
    await ElMessageBox.confirm(`确认删除流程 ${row.name} ?`, '提示', { type: 'warning' })
    await axios.delete(`/api/v1/workflow/workflows/${row.id}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    await fetchWorkflows()
  } catch (err) {
    if (err !== 'cancel') ElMessage.error(err.response?.data?.message || '删除失败')
  }
}

const openExecuteDialog = (row) => {
  executeWorkflow.value = row
  executeForm.variables = row.variables || '{}'
  executeDialogVisible.value = true
}

const executeWorkflowNow = async () => {
  if (!executeWorkflow.value?.id) return
  try {
    JSON.parse(executeForm.variables || '{}')
  } catch {
    ElMessage.warning('运行变量不是合法JSON')
    return
  }

  executing.value = true
  try {
    await axios.post(`/api/v1/workflow/workflows/${executeWorkflow.value.id}/execute`, {
      variables: JSON.parse(executeForm.variables || '{}')
    }, { headers: authHeaders() })
    ElMessage.success('已触发执行')
    executeDialogVisible.value = false
    activeTab.value = 'executions'
    await Promise.all([fetchExecutions(), fetchStats()])
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '执行失败')
  } finally {
    executing.value = false
  }
}

const openExecutionList = async (row) => {
  executionFilter.workflow_id = row.id
  activeTab.value = 'executions'
  await fetchExecutions()
}

const showExecutionDetail = async (row) => {
  try {
    const res = await axios.get(`/api/v1/workflow/executions/${row.id}`, { headers: authHeaders() })
    if (res.data?.code === 0) {
      executionDetail.value = res.data.data || { execution: null, nodes: [] }
      executionDetailVisible.value = true
    }
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '加载详情失败')
  }
}

const cancelExecution = async (row) => {
  try {
    await axios.post(`/api/v1/workflow/executions/${row.id}/cancel`, {}, { headers: authHeaders() })
    ElMessage.success('已取消')
    await Promise.all([fetchExecutions(), fetchStats()])
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '取消失败')
  }
}

const createFromTemplate = async (tpl) => {
  try {
    const { value } = await ElMessageBox.prompt('请输入新流程名称', '从模板创建', {
      inputValue: `${tpl.name}-副本`,
      confirmButtonText: '创建',
      cancelButtonText: '取消'
    })
    await axios.post(`/api/v1/workflow/templates/${tpl.id}/create`, { name: value }, { headers: authHeaders() })
    ElMessage.success('创建成功')
    await fetchWorkflows()
  } catch (err) {
    if (err !== 'cancel') ElMessage.error(err.response?.data?.message || '创建失败')
  }
}

const refreshAll = async () => {
  await Promise.all([fetchStats(), fetchWorkflows(), fetchTemplates(), fetchExecutions()])
}

onMounted(refreshAll)
</script>

<style scoped>
.page-card { max-width: 1400px; margin: 0 auto; }
.page-header { display: flex; justify-content: space-between; align-items: center; gap: 12px; margin-bottom: 12px; }
.page-desc { color: #909399; margin: 4px 0 0; }
.page-actions { display: flex; align-items: center; gap: 8px; }
.k { color: #909399; font-size: 12px; }
.v { font-size: 26px; font-weight: 700; margin-top: 4px; }
.v.success { color: #67c23a; }
.v.info { color: #409eff; }
.v.danger { color: #f56c6c; }
.mb-12 { margin-bottom: 12px; }
.mt-12 { margin-top: 12px; }
.toolbar { display: flex; gap: 8px; margin-bottom: 8px; align-items: center; }
.tpl-item { border: 1px solid #ebeef5; border-radius: 6px; padding: 10px; margin-bottom: 8px; }
.tpl-title { font-weight: 600; margin-bottom: 6px; }
.tpl-desc { color: #606266; font-size: 12px; margin-bottom: 6px; }
.tpl-meta { font-size: 12px; color: #909399; margin-bottom: 8px; }
.muted { color: #909399; font-size: 12px; }
.node-title { font-weight: 600; margin-bottom: 4px; }
</style>
