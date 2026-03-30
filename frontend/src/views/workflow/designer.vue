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
            <el-table :fit="true" :data="workflows" v-loading="loading" stripe>
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

        <el-table :fit="true" :data="executions" v-loading="executionsLoading" stripe>
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

    <el-dialog append-to-body v-model="workflowDialogVisible" :title="workflowEditing ? '编辑流程' : '新建流程'" width="980px" @closed="handleWorkflowDialogClosed">
      <el-form :model="workflowForm" label-width="96px">
        <el-row :gutter="12">
          <el-col :span="12"><el-form-item label="流程名称" required><el-input v-model="workflowForm.name" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="分类"><el-select v-model="workflowForm.category" style="width: 100%"><el-option label="deploy" value="deploy" /><el-option label="monitor" value="monitor" /><el-option label="backup" value="backup" /><el-option label="custom" value="custom" /></el-select></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="触发方式"><el-select v-model="workflowForm.trigger" style="width: 100%"><el-option label="manual" value="manual" /><el-option label="schedule" value="schedule" /><el-option label="webhook" value="webhook" /><el-option label="alert" value="alert" /></el-select></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="状态"><el-switch v-model="workflowForm.enabled" /></el-form-item></el-col>
          <el-col :span="24"><el-form-item label="描述"><el-input v-model="workflowForm.description" /></el-form-item></el-col>
          <el-col :span="24"><el-form-item label="流程定义" required><el-input v-model="workflowForm.definition" type="textarea" :rows="10" /></el-form-item></el-col>
          <el-col :span="24">
            <el-form-item label="节点编辑器">
              <div class="node-editor-wrap">
                <div class="node-editor-toolbar">
                  <el-button size="small" @click="syncEditorFromDefinition">从 JSON 同步</el-button>
                  <el-button size="small" @click="formatDefinition">格式化 JSON</el-button>
                  <el-button size="small" type="primary" plain @click="appendNode('shell')">新增 Shell</el-button>
                  <el-button size="small" plain @click="appendNode('notify')">新增通知</el-button>
                  <el-button size="small" plain @click="appendNode('http')">新增 HTTP</el-button>
                  <el-button size="small" plain @click="appendNode('approval')">新增审批</el-button>
                </div>
                <el-alert v-if="definitionParseError" type="warning" :closable="false" show-icon class="mb-12" :title="definitionParseError" />
                <div v-if="definitionNodes.length" class="node-editor-list">
                  <el-card v-for="(node, index) in definitionNodes" :key="`${node.id || 'node'}-${index}`" shadow="never" class="node-editor-card">
                    <template #header>
                      <div class="node-editor-card__header">
                        <span>{{ node.name || node.id || `节点 ${index + 1}` }}</span>
                        <div class="node-editor-card__actions">
                          <el-tag size="small" effect="plain">{{ node.type || 'notify' }}</el-tag>
                          <el-button size="small" type="danger" text :disabled="['start', 'end'].includes(node.type)" @click="removeNode(index)">移除</el-button>
                        </div>
                      </div>
                    </template>
                    <el-row :gutter="12">
                      <el-col :span="8"><el-form-item label="ID" label-width="72px"><el-input v-model="node.id" /></el-form-item></el-col>
                      <el-col :span="8"><el-form-item label="名称" label-width="72px"><el-input v-model="node.name" /></el-form-item></el-col>
                      <el-col :span="8"><el-form-item label="类型" label-width="72px"><el-select v-model="node.type" style="width: 100%" @change="handleNodeTypeChange(node)"><el-option v-for="item in nodeTypeOptions" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item></el-col>
                      <el-col :span="24"><el-form-item label="下一节点" label-width="72px"><el-input v-model="node.nextText" placeholder="多个节点用逗号分隔" /></el-form-item></el-col>

                      <el-col v-if="node.type === 'shell'" :span="24"><el-form-item label="脚本" label-width="72px"><el-input v-model="node.config.script" type="textarea" :rows="6" placeholder="sh 脚本内容" /></el-form-item></el-col>
                      <el-col v-if="node.type === 'shell'" :span="8"><el-form-item label="超时" label-width="72px"><el-input-number v-model="node.config.timeout" :min="1" :max="3600" /></el-form-item></el-col>

                      <el-col v-if="node.type === 'notify' || node.type === 'approval'" :span="24"><el-form-item label="标题" label-width="72px"><el-input v-model="node.config.title" /></el-form-item></el-col>
                      <el-col v-if="node.type === 'notify' || node.type === 'approval'" :span="24"><el-form-item label="内容" label-width="72px"><el-input v-model="node.config.content" type="textarea" :rows="4" /></el-form-item></el-col>
                      <el-col v-if="node.type === 'notify' || node.type === 'approval'" :span="12"><el-form-item label="渠道ID" label-width="72px"><el-input v-model="node.config.channel_id" placeholder="可选" /></el-form-item></el-col>

                      <el-col v-if="node.type === 'http'" :span="8"><el-form-item label="方法" label-width="72px"><el-select v-model="node.config.method" style="width: 100%"><el-option label="GET" value="GET" /><el-option label="POST" value="POST" /><el-option label="PUT" value="PUT" /><el-option label="DELETE" value="DELETE" /></el-select></el-form-item></el-col>
                      <el-col v-if="node.type === 'http'" :span="16"><el-form-item label="URL" label-width="72px"><el-input v-model="node.config.url" /></el-form-item></el-col>
                      <el-col v-if="node.type === 'http'" :span="24"><el-form-item label="Body" label-width="72px"><el-input v-model="node.config.body" type="textarea" :rows="4" /></el-form-item></el-col>

                      <el-col v-if="node.type === 'wait'" :span="8"><el-form-item label="等待秒数" label-width="72px"><el-input-number v-model="node.config.seconds" :min="1" :max="86400" /></el-form-item></el-col>
                      <el-col v-if="node.type === 'ai'" :span="24"><el-form-item label="Prompt" label-width="72px"><el-input v-model="node.config.prompt" type="textarea" :rows="4" /></el-form-item></el-col>

                      <el-col v-if="!['start', 'end', 'shell', 'notify', 'approval', 'http', 'wait', 'ai'].includes(node.type)" :span="24">
                        <el-form-item label="配置JSON" label-width="72px">
                          <el-input v-model="node.configJson" type="textarea" :rows="4" @blur="applyRawConfig(node)" />
                        </el-form-item>
                      </el-col>
                    </el-row>
                  </el-card>
                </div>
                <el-empty v-else description="当前流程没有可编辑节点" />
              </div>
            </el-form-item>
          </el-col>
          <el-col :span="24"><el-form-item label="默认变量"><el-input v-model="workflowForm.variables" type="textarea" :rows="4" placeholder='{"service":"nginx"}' /></el-form-item></el-col>
        </el-row>
      </el-form>
      <template #footer>
        <el-button @click="validateWorkflow">验证定义</el-button>
        <el-button @click="workflowDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="workflowSaving" @click="saveWorkflow">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog append-to-body v-model="executeDialogVisible" title="执行流程" width="600px" @closed="handleExecuteDialogClosed">
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

    <el-drawer v-model="executionDetailVisible" title="执行详情" size="55%" append-to-body @closed="handleExecutionDetailClosed">
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
import { onMounted, reactive, ref, watch } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'
import { getErrorMessage, isCancelError } from '@/utils/error'

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
const route = useRoute()
const router = useRouter()

const executeWorkflow = ref(null)
const executionDetail = ref({ execution: null, nodes: [] })
const definitionNodes = ref([])
const definitionParseError = ref('')
const openingWorkflowID = ref('')
let syncingFromEditor = false
let syncingFromText = false

const nodeTypeOptions = [
  { label: 'start', value: 'start' },
  { label: 'end', value: 'end' },
  { label: 'shell', value: 'shell' },
  { label: 'notify', value: 'notify' },
  { label: 'approval', value: 'approval' },
  { label: 'http', value: 'http' },
  { label: 'wait', value: 'wait' },
  { label: 'ai', value: 'ai' },
  { label: 'condition', value: 'condition' },
  { label: 'parallel', value: 'parallel' }
]

const executionFilter = reactive({ workflow_id: '', status: '' })

const workflowForm = reactive({
  id: '',
  name: '',
  description: '',
  category: 'custom',
  definition: '',
  variables: '{}',
  trigger: 'manual',
  enabled: true
})

const executeForm = reactive({
  variables: '{}'
})

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const defaultDefinitionObject = () => ({
  nodes: [
    { id: 'start', type: 'start', name: '开始', next: ['end'] },
    { id: 'end', type: 'end', name: '结束' }
  ]
})

const stringifyDefinition = (value) => JSON.stringify(value, null, 2)

const defaultNodeConfig = (type) => {
  if (type === 'shell') return { script: '', timeout: 300 }
  if (type === 'notify' || type === 'approval') return { title: '', content: '', channel_id: '' }
  if (type === 'http') return { method: 'GET', url: '', body: '' }
  if (type === 'wait') return { seconds: 10 }
  if (type === 'ai') return { prompt: '' }
  return {}
}

const normalizeConfig = (type, config) => ({ ...defaultNodeConfig(type), ...((config && typeof config === 'object') ? config : {}) })

const normalizeNode = (node = {}, index = 0) => {
  const type = node.type || 'notify'
  const config = normalizeConfig(type, node.config)
  return {
    id: String(node.id || `${type}_${index + 1}`),
    type,
    name: String(node.name || `节点 ${index + 1}`),
    nextText: Array.isArray(node.next) ? node.next.join(', ') : '',
    config,
    configJson: JSON.stringify(config, null, 2)
  }
}

const buildNodePayload = (node, index) => {
  const id = String(node.id || `node_${index + 1}`).trim()
  const type = String(node.type || 'notify').trim() || 'notify'
  let config = {}
  if (!['start', 'end'].includes(type)) {
    if (!['start', 'end', 'shell', 'notify', 'approval', 'http', 'wait', 'ai'].includes(type) && node.configJson?.trim()) {
      try {
        config = JSON.parse(node.configJson)
      } catch (_) {
        config = normalizeConfig(type, node.config)
      }
    } else {
      config = normalizeConfig(type, node.config)
    }
  }
  const payload = {
    id,
    type,
    name: String(node.name || id).trim() || id
  }
  if (Object.keys(config).length > 0) payload.config = config
  const next = String(node.nextText || '')
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)
  if (next.length > 0) payload.next = next
  return payload
}

const syncEditorFromDefinition = () => {
  try {
    const parsed = JSON.parse(workflowForm.definition || '{}')
    const nodes = Array.isArray(parsed.nodes) ? parsed.nodes : []
    syncingFromText = true
    definitionNodes.value = nodes.map((node, index) => normalizeNode(node, index))
    definitionParseError.value = ''
    Promise.resolve().then(() => {
      syncingFromText = false
    })
  } catch (err) {
    definitionParseError.value = '流程定义 JSON 解析失败，节点编辑器保留上一次可用内容。'
  }
}

const syncDefinitionFromEditor = () => {
  const nodes = definitionNodes.value.map((node, index) => buildNodePayload(node, index))
  syncingFromEditor = true
  workflowForm.definition = stringifyDefinition({ nodes })
  Promise.resolve().then(() => {
    syncingFromEditor = false
  })
}

const formatDefinition = () => {
  try {
    workflowForm.definition = stringifyDefinition(JSON.parse(workflowForm.definition || '{}'))
    syncEditorFromDefinition()
    ElMessage.success('JSON 已格式化')
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '格式化失败'))
  }
}

const appendNode = (type) => {
  const index = definitionNodes.value.length + 1
  definitionNodes.value.push(normalizeNode({
    id: `${type}_${Date.now().toString(36)}`,
    type,
    name: `新${type}节点`,
    config: defaultNodeConfig(type)
  }, index))
}

const removeNode = (index) => {
  definitionNodes.value.splice(index, 1)
}

const handleNodeTypeChange = (node) => {
  node.config = normalizeConfig(node.type, {})
  node.configJson = JSON.stringify(node.config, null, 2)
}

const applyRawConfig = (node) => {
  if (!node.configJson?.trim()) {
    node.config = {}
    return
  }
  try {
    node.config = JSON.parse(node.configJson)
  } catch (err) {
    ElMessage.error('配置 JSON 非法，已保留原值')
    node.configJson = JSON.stringify(node.config || {}, null, 2)
  }
}

const openWorkflowFromRoute = () => {
  const workflowID = String(route.query.workflow_id || '').trim()
  if (!workflowID || openingWorkflowID.value === workflowID) return
  const row = workflows.value.find((item) => item.id === workflowID)
  if (!row) return
  openingWorkflowID.value = workflowID
  openWorkflowDialog(row)
  if (String(route.query.auto_open || '') === '1') {
    ElMessage.success('已打开 AI Runbook，可在节点编辑器中补充为可执行流程。')
  }
}

const handleWorkflowDialogClosed = () => {
  workflowEditing.value = false
  openingWorkflowID.value = ''
  resetWorkflowForm()
}

const handleExecuteDialogClosed = () => {
  executeWorkflow.value = null
  executeForm.variables = '{}'
}

const handleExecutionDetailClosed = () => {
  executionDetail.value = { execution: null, nodes: [] }
}

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
  try {
    const res = await axios.get('/api/v1/workflow/stats', { headers: authHeaders() })
    if (res.data?.code === 0) stats.value = res.data.data || stats.value
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '加载统计失败'))
  }
}

const fetchWorkflows = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/v1/workflow/workflows', { headers: authHeaders() })
    if (res.data?.code === 0) workflows.value = res.data.data || []
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '加载流程失败'))
  } finally {
    loading.value = false
  }
}

const fetchTemplates = async () => {
  try {
    const res = await axios.get('/api/v1/workflow/templates', { headers: authHeaders() })
    if (res.data?.code === 0) templates.value = res.data.data || []
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '加载模板失败'))
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
    ElMessage.error(getErrorMessage(err, '加载执行记录失败'))
  } finally {
    executionsLoading.value = false
  }
}

const resetWorkflowForm = () => {
  workflowForm.id = ''
  workflowForm.name = ''
  workflowForm.description = ''
  workflowForm.category = 'custom'
  workflowForm.definition = stringifyDefinition(defaultDefinitionObject())
  workflowForm.variables = '{}'
  workflowForm.trigger = 'manual'
  workflowForm.enabled = true
  definitionNodes.value = defaultDefinitionObject().nodes.map((node, index) => normalizeNode(node, index))
  definitionParseError.value = ''
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
  syncEditorFromDefinition()
  workflowDialogVisible.value = true
}

const validateWorkflow = async () => {
  try {
    const res = await axios.post('/api/v1/workflow/validate', {
      definition: workflowForm.definition
    }, { headers: authHeaders() })
    ElMessage.success(`验证通过，节点数 ${res.data?.data?.node_count || 0}`)
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '验证失败'))
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
    let savedWorkflowID = workflowForm.id
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
      const res = await axios.put(`/api/v1/workflow/workflows/${workflowForm.id}`, payload, { headers: authHeaders() })
      savedWorkflowID = res.data?.data?.id || workflowForm.id
      ElMessage.success('更新成功')
    } else {
      const res = await axios.post('/api/v1/workflow/workflows', payload, { headers: authHeaders() })
      savedWorkflowID = res.data?.data?.id || ''
      ElMessage.success('创建成功')
    }
    workflowDialogVisible.value = false
    await fetchWorkflows()
    if (savedWorkflowID) {
      router.replace({ query: { ...route.query, workflow_id: savedWorkflowID } })
    }
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '保存失败'))
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
    if (!isCancelError(err)) ElMessage.error(getErrorMessage(err, '删除失败'))
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
    ElMessage.error(getErrorMessage(err, '执行失败'))
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
    ElMessage.error(getErrorMessage(err, '加载详情失败'))
  }
}

const cancelExecution = async (row) => {
  try {
    await axios.post(`/api/v1/workflow/executions/${row.id}/cancel`, {}, { headers: authHeaders() })
    ElMessage.success('已取消')
    await Promise.all([fetchExecutions(), fetchStats()])
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '取消失败'))
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
    if (!isCancelError(err)) ElMessage.error(getErrorMessage(err, '创建失败'))
  }
}

const refreshAll = async () => {
  await Promise.all([fetchStats(), fetchWorkflows(), fetchTemplates(), fetchExecutions()])
}

watch(definitionNodes, () => {
  if (syncingFromText) return
  syncDefinitionFromEditor()
}, { deep: true })

watch(() => workflowForm.definition, () => {
  if (!workflowDialogVisible.value) return
  if (syncingFromEditor) return
  syncEditorFromDefinition()
})

watch(() => [route.query.workflow_id, workflows.value.length], () => {
  openWorkflowFromRoute()
})

onMounted(async () => {
  resetWorkflowForm()
  await refreshAll()
  openWorkflowFromRoute()
})
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
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
.node-editor-wrap { width: 100%; }
.node-editor-toolbar { display: flex; gap: 8px; flex-wrap: wrap; margin-bottom: 12px; }
.node-editor-list { display: flex; flex-direction: column; gap: 12px; }
.node-editor-card { border-radius: 10px; }
.node-editor-card__header { display: flex; justify-content: space-between; align-items: center; gap: 12px; }
.node-editor-card__actions { display: flex; align-items: center; gap: 8px; }
</style>
