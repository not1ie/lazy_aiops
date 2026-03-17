<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>协作中心</h2>
        <p class="page-desc">把 AI 问答、工单流转、流程执行、值班升级与终端会话放进同一协同工作台。</p>
      </div>
      <div class="page-actions">
        <el-button type="primary" plain @click="applyRecommendedWorkspace">推荐工作台</el-button>
        <el-button :loading="loading" icon="Refresh" @click="refreshAll">刷新</el-button>
      </div>
    </div>

    <div class="module-tabs">
      <el-tag v-for="item in quickTabs" :key="item.path" class="tab-item" effect="plain" @click="go(item.path)">
        {{ item.label }}
      </el-tag>
    </div>

    <el-row :gutter="12" class="summary-row">
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">AI会话</div><div class="metric-value">{{ stats.aiSessions }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">工单总数</div><div class="metric-value">{{ stats.workorderTotal }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">待处理工单</div><div class="metric-value warning">{{ stats.workorderPending }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">流程执行中</div><div class="metric-value warning">{{ stats.workflowRunning }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">流程失败</div><div class="metric-value danger">{{ stats.workflowFailed }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">当前值班</div><div class="metric-value ok">{{ stats.oncallNow }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">活跃终端会话</div><div class="metric-value ok">{{ stats.terminalActive }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card>
          <div class="metric-title">待处置积压</div>
          <div class="metric-value warning">{{ pendingBacklog }}</div>
          <div class="metric-sub">待审批 {{ pendingApprovalTimeout }}</div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="12">
      <el-col :span="10">
        <el-card>
          <template #header>协同健康</template>
          <div class="health-row">
            <span>工单闭环率</span>
            <strong>{{ workorderCloseRate }}%</strong>
          </div>
          <el-progress :percentage="workorderCloseRate" :stroke-width="14" />
          <div class="health-row mtop">
            <span>流程成功率</span>
            <strong>{{ workflowSuccessRate }}%</strong>
          </div>
          <el-progress :percentage="workflowSuccessRate" :stroke-width="14" status="success" />
          <el-divider />
          <div class="health-row"><span>排班数量</span><strong>{{ stats.scheduleTotal }}</strong></div>
          <div class="health-row"><span>终端会话总数</span><strong>{{ stats.terminalTotal }}</strong></div>
          <div class="health-row"><span>超时待审批工单</span><strong>{{ pendingApprovalTimeout }}</strong></div>
          <div class="health-row"><span>长时间流程执行</span><strong>{{ workflowRunningTimeout }}</strong></div>
          <div class="health-row"><span>待连接超时会话</span><strong>{{ terminalPendingTimeout }}</strong></div>
          <div class="health-row"><span>连接失败会话</span><strong>{{ terminalFailedCount }}</strong></div>
          <div class="health-row"><span>AI最近活跃时间</span><strong>{{ latestAISessionTime }}</strong></div>
        </el-card>
      </el-col>
      <el-col :span="14">
        <el-card>
          <template #header>近期协同动态</template>
          <el-table :fit="true" :data="activityRows" size="small" max-height="300" empty-text="暂无协同动态">
            <el-table-column prop="type" label="类型" width="110" />
            <el-table-column prop="title" label="标题" min-width="200" show-overflow-tooltip />
            <el-table-column prop="status" label="状态" width="120">
              <template #default="{ row }">
                <el-tag :type="row.level">{{ row.status }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="time" label="时间" min-width="160" />
            <el-table-column label="操作" width="90">
              <template #default="{ row }">
                <el-button link type="primary" @click="go(row.path)">查看</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <el-card class="integration-card">
      <template #header>
        <div class="integration-header">
          <span>协作融合视图</span>
          <div class="integration-actions">
            <el-input
              v-model="panelKeyword"
              clearable
              size="small"
              class="panel-search"
              placeholder="筛选会话、工单、流程、值班成员..."
            />
            <el-button size="small" type="primary" plain @click="openCurrentPanel">进入完整页面</el-button>
          </div>
        </div>
      </template>

      <el-tabs v-model="activePanel" class="integration-tabs">
        <el-tab-pane label="AI会话" name="ai">
          <el-table :fit="true" :data="filteredAISessions" size="small" max-height="360" empty-text="暂无 AI 会话">
            <el-table-column prop="title" label="会话" min-width="180" />
            <el-table-column prop="type" label="类型" width="100" />
            <el-table-column label="更新时间" min-width="165">
              <template #default="{ row }">{{ formatTime(row.updated_at) }}</template>
            </el-table-column>
            <el-table-column prop="context" label="上下文" min-width="220" show-overflow-tooltip />
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="工单" name="workorder">
          <div class="panel-toolbar">
            <div class="panel-toolbar-left">
              <el-tag type="warning" effect="light">待审批 {{ workorderPendingCount }}</el-tag>
              <el-tag type="danger" effect="light">超时 {{ pendingApprovalTimeout }}</el-tag>
            </div>
            <div class="panel-toolbar-right">
              <el-button size="small" type="success" plain :loading="workorderBatching" :disabled="!selectedWorkorderApprovableCount" @click="batchApproveSelectedWorkorders">批量通过已选</el-button>
              <el-button size="small" type="danger" plain :loading="workorderBatching" :disabled="!selectedWorkorderCancelableCount" @click="batchCancelSelectedWorkorders">批量取消已选</el-button>
              <el-button size="small" plain :loading="workorderBatching" :disabled="!pendingApprovalTimeout" @click="batchApproveTimeoutWorkorders">处置超时审批</el-button>
            </div>
          </div>
          <el-table
            ref="workorderTableRef"
            :fit="true"
            :data="filteredOrders"
            size="small"
            max-height="360"
            empty-text="暂无工单数据"
            @selection-change="onWorkorderSelectionChange"
          >
            <el-table-column type="selection" width="46" />
            <el-table-column prop="title" label="标题" min-width="180" show-overflow-tooltip />
            <el-table-column prop="type_name" label="类型" min-width="120" />
            <el-table-column label="优先级" width="90">
              <template #default="{ row }">
                <el-tag :type="priorityTag(row.priority)">{{ priorityText(row.priority) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="状态" width="110">
              <template #default="{ row }">
                <el-tag :type="workorderStatusTag(row.status)">{{ workorderStatusText(row.status) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="submitter" label="提交人" width="110" />
            <el-table-column label="创建时间" min-width="165">
              <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
            </el-table-column>
            <el-table-column label="等待时长" width="110">
              <template #default="{ row }">
                <el-tag :type="isWorkorderPendingTimeout(row) ? 'warning' : 'success'">
                  {{ formatDuration(row.created_at) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" min-width="220" fixed="right">
              <template #default="{ row }">
                <div class="inline-actions">
                  <el-button v-if="canApprove(row)" link type="primary" @click="approveOrder(row, true)">通过</el-button>
                  <el-button v-if="canApprove(row)" link type="warning" @click="approveOrder(row, false)">拒绝</el-button>
                  <el-button v-if="Number(row.status) === 2" link type="primary" @click="executeOrder(row)">执行</el-button>
                  <el-button v-if="Number(row.status) === 4" link type="success" @click="completeOrder(row)">完成</el-button>
                  <el-button v-if="canCancel(row)" link type="danger" @click="cancelOrder(row)">取消</el-button>
                </div>
              </template>
            </el-table-column>
          </el-table>
          <div class="group-tag-list">
            <el-tag
              v-for="group in workorderApproveGroups"
              :key="group.key"
              type="warning"
              effect="plain"
              class="group-tag-item"
              @click="batchApproveWorkorderGroup(group)"
            >
              {{ group.label }} · {{ group.count }}
            </el-tag>
          </div>
        </el-tab-pane>

        <el-tab-pane label="流程执行" name="workflow">
          <el-table :fit="true" :data="filteredWorkflows" size="small" max-height="180" empty-text="暂无流程定义">
            <el-table-column prop="name" label="流程名称" min-width="160" />
            <el-table-column prop="category" label="分类" width="100" />
            <el-table-column prop="trigger" label="触发" width="100" />
            <el-table-column label="状态" width="90">
              <template #default="{ row }">
                <el-tag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? '启用' : '停用' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="120" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" @click="executeWorkflow(row)">执行</el-button>
              </template>
            </el-table-column>
          </el-table>

          <el-divider content-position="left">近期执行</el-divider>

          <el-table :fit="true" :data="filteredExecutions" size="small" max-height="360" empty-text="暂无流程执行">
            <el-table-column prop="workflow_name" label="流程" min-width="170" />
            <el-table-column prop="trigger_by" label="触发人" width="110" />
            <el-table-column prop="trigger" label="触发方式" width="100" />
            <el-table-column label="状态" width="110">
              <template #default="{ row }">
                <el-tag :type="workflowStatusTag(row.status)">{{ workflowStatusText(row.status) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="duration" label="耗时(s)" width="100" />
            <el-table-column label="开始时间" min-width="165">
              <template #default="{ row }">{{ formatTime(row.started_at) }}</template>
            </el-table-column>
            <el-table-column label="持续时长" width="110">
              <template #default="{ row }">
                <el-tag :type="isWorkflowRunningTimeout(row) ? 'warning' : 'success'">
                  {{ formatDuration(row.started_at) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="130" fixed="right">
              <template #default="{ row }">
                <el-button v-if="Number(row.status) === 0" link type="danger" @click="cancelExecution(row)">取消</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="值班与升级" name="oncall">
          <el-row :gutter="12">
            <el-col :span="12">
              <el-table :fit="true" :data="filteredOncallNow" size="small" max-height="340" empty-text="当前无人值班">
                <el-table-column prop="username" label="当前值班" min-width="120" />
                <el-table-column prop="type" label="类型" width="90" />
                <el-table-column label="结束时间" min-width="160">
                  <template #default="{ row }">{{ formatTime(row.end_at) }}</template>
                </el-table-column>
              </el-table>
            </el-col>
            <el-col :span="12">
              <el-table :fit="true" :data="filteredEscalations" size="small" max-height="340" empty-text="暂无升级策略">
                <el-table-column prop="name" label="升级策略" min-width="130" />
                <el-table-column label="状态" width="90">
                  <template #default="{ row }">
                    <el-tag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? '启用' : '禁用' }}</el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="schedule_id" label="关联排班" min-width="140" show-overflow-tooltip />
              </el-table>
            </el-col>
          </el-row>
        </el-tab-pane>

        <el-tab-pane label="WebTerminal" name="terminal">
          <div class="panel-toolbar">
            <div class="panel-toolbar-left">
              <el-tag type="success" effect="light">已连接 {{ terminalActiveCount }}</el-tag>
              <el-tag type="warning" effect="light">待连接超时 {{ terminalPendingTimeout }}</el-tag>
              <el-tag type="danger" effect="light">失败 {{ terminalFailedCount }}</el-tag>
            </div>
            <div class="panel-toolbar-right">
              <el-button size="small" type="warning" plain :loading="terminalBatching" :disabled="!selectedTerminalClosableCount" @click="batchCloseSelectedTerminals">批量关闭已选</el-button>
              <el-button size="small" type="danger" plain :loading="terminalBatching" :disabled="!selectedTerminalPurgeableCount" @click="batchPurgeSelectedTerminals">批量删除已选</el-button>
              <el-button size="small" plain :loading="terminalBatching" :disabled="!terminalFailedCount" @click="batchPurgeFailedTerminals">清理失败会话</el-button>
            </div>
          </div>
          <el-table
            ref="terminalTableRef"
            :fit="true"
            :data="filteredTerminalSessions"
            size="small"
            max-height="360"
            empty-text="暂无终端会话"
            @selection-change="onTerminalSelectionChange"
          >
            <el-table-column type="selection" width="46" />
            <el-table-column prop="operator" label="操作人" min-width="110" />
            <el-table-column prop="host" label="主机" min-width="150" />
            <el-table-column prop="username" label="登录用户" width="110" />
            <el-table-column label="状态" width="100">
              <template #default="{ row }">
                <el-tag :type="terminalStatusTag(row.status)">{{ terminalStatusText(row.status) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="开始时间" min-width="160">
              <template #default="{ row }">{{ formatTime(row.started_at || row.created_at) }}</template>
            </el-table-column>
            <el-table-column label="持续时长" width="110">
              <template #default="{ row }">
                <el-tag :type="isTerminalPendingTimeout(row) ? 'warning' : 'success'">
                  {{ formatDuration(row.started_at || row.created_at) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="错误信息" min-width="180" show-overflow-tooltip>
              <template #default="{ row }">{{ row.last_error || '-' }}</template>
            </el-table-column>
            <el-table-column label="操作" min-width="170" fixed="right">
              <template #default="{ row }">
                <div class="inline-actions">
                  <el-button link type="primary" @click="connectTerminal(row)">连接</el-button>
                  <el-button v-if="Number(row.status) === 1" link type="warning" @click="closeTerminalSession(row)">关闭</el-button>
                  <el-button link type="danger" @click="purgeTerminalSession(row)">删除</el-button>
                </div>
              </template>
            </el-table-column>
          </el-table>
          <div class="group-tag-list">
            <el-tag
              v-for="group in terminalIssueGroups"
              :key="group.key"
              :type="group.level"
              effect="plain"
              class="group-tag-item"
              @click="batchPurgeTerminalGroup(group)"
            >
              {{ group.label }} · {{ group.count }}
            </el-tag>
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </el-card>
</template>

<script setup>
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getErrorMessage, isCancelError } from '@/utils/error'
import { requestApplyWorkspaceCategory } from '@/utils/workspace'

const router = useRouter()
const loading = ref(false)
const activePanel = ref('workorder')
const panelKeyword = ref('')
const workorderBatching = ref(false)
const terminalBatching = ref(false)
const nowTs = ref(Date.now())
const workorderTableRef = ref(null)
const terminalTableRef = ref(null)
const selectedWorkorderRows = ref([])
const selectedTerminalRows = ref([])
let minuteTicker = null

const aiSessions = ref([])
const workorders = ref([])
const workorderStats = ref({})
const workflows = ref([])
const workflowExecutions = ref([])
const workflowStats = ref({})
const oncallNow = ref([])
const schedules = ref([])
const escalations = ref([])
const terminalSessions = ref([])

const stats = reactive({
  aiSessions: 0,
  workorderTotal: 0,
  workorderPending: 0,
  workflowRunning: 0,
  workflowFailed: 0,
  oncallNow: 0,
  terminalActive: 0,
  escalationTotal: 0,
  scheduleTotal: 0,
  terminalTotal: 0
})

const quickTabs = [
  { label: 'AI运维助手', path: '/ai' },
  { label: '工单管理', path: '/workorder/tickets' },
  { label: '工作流编排', path: '/workflow/designer' },
  { label: '值班排班', path: '/oncall/schedule' },
  { label: '升级策略', path: '/oncall/escalation' },
  { label: 'WebTerminal', path: '/terminal' }
]

const panelRouteMap = {
  ai: '/ai',
  workorder: '/workorder/tickets',
  workflow: '/workflow/designer',
  oncall: '/oncall/schedule',
  terminal: '/terminal'
}

const applyRecommendedWorkspace = () => requestApplyWorkspaceCategory('collab', 'hub-collab')

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })
const go = (path) => router.push(path)
const normalizeText = (value) => String(value ?? '').trim().toLowerCase()

const priorityText = (value) => {
  const v = Number(value)
  if (v === 1) return '紧急'
  if (v === 2) return '高'
  if (v === 3) return '中'
  return '低'
}

const priorityTag = (value) => {
  const v = Number(value)
  if (v === 1) return 'danger'
  if (v === 2) return 'warning'
  if (v === 3) return ''
  return 'info'
}

const workorderStatusText = (value) => {
  const v = Number(value)
  if (v === 0) return '待审批'
  if (v === 1) return '审批中'
  if (v === 2) return '已通过'
  if (v === 3) return '已拒绝'
  if (v === 4) return '执行中'
  if (v === 5) return '已完成'
  if (v === 6) return '已取消'
  return '未知'
}

const workorderStatusTag = (value) => {
  const v = Number(value)
  if (v === 5) return 'success'
  if (v === 3 || v === 6) return 'danger'
  if (v === 0 || v === 1 || v === 4) return 'warning'
  return 'info'
}

const workflowStatusText = (value) => {
  const v = Number(value)
  if (v === 0) return '运行中'
  if (v === 1) return '成功'
  if (v === 2) return '失败'
  if (v === 3) return '取消'
  if (v === 4) return '待审批'
  return '未知'
}

const workflowStatusTag = (value) => {
  const v = Number(value)
  if (v === 1) return 'success'
  if (v === 2) return 'danger'
  if (v === 0 || v === 4) return 'warning'
  return 'info'
}

const terminalStatusText = (value) => {
  const v = Number(value)
  if (v === 0) return '待连接'
  if (v === 1) return '已连接'
  if (v === 2) return '已关闭'
  if (v === 3) return '连接失败'
  return '未知'
}

const terminalStatusTag = (value) => {
  const v = Number(value)
  if (v === 1) return 'success'
  if (v === 3) return 'danger'
  if (v === 0) return 'warning'
  return 'info'
}

const parseTimestamp = (value) => {
  if (!value) return null
  const ts = new Date(value).getTime()
  return Number.isNaN(ts) ? null : ts
}

const elapsedMinutes = (value) => {
  const ts = parseTimestamp(value)
  if (!ts) return 0
  const diff = Math.floor((nowTs.value - ts) / 60000)
  return diff > 0 ? diff : 0
}

const formatDuration = (value) => {
  const minutes = elapsedMinutes(value)
  if (minutes < 60) return `${minutes}m`
  const hours = Math.floor(minutes / 60)
  const remain = minutes % 60
  if (hours < 24) return `${hours}h${remain}m`
  const days = Math.floor(hours / 24)
  return `${days}d${hours % 24}h`
}

const isWorkorderPendingTimeout = (row) => {
  const status = Number(row?.status)
  if (status !== 0 && status !== 1 && status !== 4) return false
  return elapsedMinutes(row?.created_at) >= 120
}

const isWorkflowRunningTimeout = (row) => Number(row?.status) === 0 && elapsedMinutes(row?.started_at) >= 15

const isTerminalPendingTimeout = (row) => Number(row?.status) === 0 && elapsedMinutes(row?.started_at || row?.created_at) >= 10

const workorderCloseRate = computed(() => {
  const total = Number(workorderStats.value.total || 0)
  const completed = Number(workorderStats.value.completed || 0)
  if (!total) return 0
  return Math.round((completed / total) * 100)
})

const workflowSuccessRate = computed(() => {
  const success = Number(workflowStats.value.success || 0)
  const failed = Number(workflowStats.value.failed || 0)
  const running = Number(workflowStats.value.running || 0)
  const total = success + failed + running
  if (!total) return 0
  return Math.round((success / total) * 100)
})

const latestAISessionTime = computed(() => {
  if (!aiSessions.value.length) return '-'
  return formatTime(aiSessions.value[0].updated_at)
})

const pendingApprovalTimeout = computed(() => workorders.value.filter((item) => isWorkorderPendingTimeout(item)).length)
const workflowRunningTimeout = computed(() => workflowExecutions.value.filter((item) => isWorkflowRunningTimeout(item)).length)
const terminalPendingTimeout = computed(() => terminalSessions.value.filter((item) => isTerminalPendingTimeout(item)).length)
const terminalFailedCount = computed(() => terminalSessions.value.filter((item) => Number(item.status) === 3).length)
const workorderPendingCount = computed(() => workorders.value.filter((item) => canApprove(item)).length)
const terminalActiveCount = computed(() => terminalSessions.value.filter((item) => Number(item.status) === 1).length)
const pendingBacklog = computed(
  () => pendingApprovalTimeout.value + workflowRunningTimeout.value + terminalPendingTimeout.value + terminalFailedCount.value
)

const selectedWorkorderApprovableCount = computed(() =>
  selectedWorkorderRows.value.filter((item) => canApprove(item)).length
)

const selectedWorkorderCancelableCount = computed(() =>
  selectedWorkorderRows.value.filter((item) => canCancel(item)).length
)

const selectedTerminalClosableCount = computed(() =>
  selectedTerminalRows.value.filter((item) => Number(item?.status) === 1).length
)

const selectedTerminalPurgeableCount = computed(() =>
  selectedTerminalRows.value.filter((item) => Number(item?.status) !== 1).length
)

const workorderApproveGroups = computed(() => {
  const groups = new Map()
  filteredOrders.value
    .filter((item) => canApprove(item))
    .forEach((item) => {
      const label = item.type_name || '未分类'
      const key = String(label)
      const current = groups.get(key) || { key, label, count: 0, rows: [] }
      current.count += 1
      current.rows.push(item)
      groups.set(key, current)
    })
  return [...groups.values()].sort((a, b) => b.count - a.count).slice(0, 8)
})

const terminalIssueGroups = computed(() => {
  const groups = new Map()
  filteredTerminalSessions.value
    .filter((item) => Number(item.status) !== 1)
    .forEach((item) => {
      const label = item.host || item.operator || '未命名主机'
      const key = String(label)
      const current = groups.get(key) || { key, label, count: 0, rows: [], level: 'warning' }
      current.count += 1
      if (Number(item.status) === 3) current.level = 'danger'
      current.rows.push(item)
      groups.set(key, current)
    })
  return [...groups.values()].sort((a, b) => b.count - a.count).slice(0, 8)
})

const activityRows = computed(() => {
  const rows = []
  workorders.value.slice(0, 6).forEach((item) => {
    rows.push({
      type: '工单',
      title: item.title || '未命名工单',
      status: workorderStatusText(item.status),
      level: workorderStatusTag(item.status) || 'info',
      time: formatTime(item.updated_at || item.created_at),
      sortAt: new Date(item.updated_at || item.created_at || 0).getTime(),
      path: '/workorder/tickets'
    })
  })
  workflowExecutions.value.slice(0, 6).forEach((item) => {
    rows.push({
      type: '流程',
      title: item.workflow_name || '工作流执行',
      status: workflowStatusText(item.status),
      level: workflowStatusTag(item.status),
      time: formatTime(item.started_at),
      sortAt: new Date(item.started_at || 0).getTime(),
      path: '/workflow/designer'
    })
  })
  terminalSessions.value.slice(0, 6).forEach((item) => {
    rows.push({
      type: '终端',
      title: `${item.operator || '-'}@${item.host || '-'}`,
      status: terminalStatusText(item.status),
      level: terminalStatusTag(item.status),
      time: formatTime(item.started_at || item.created_at),
      sortAt: new Date(item.started_at || item.created_at || 0).getTime(),
      path: '/terminal'
    })
  })
  return rows.sort((a, b) => b.sortAt - a.sortAt).slice(0, 12)
})

const filterRows = (rows, fields) => {
  const keyword = normalizeText(panelKeyword.value)
  const base = Array.isArray(rows) ? rows : []
  if (!keyword) return base.slice(0, 30)
  return base.filter((row) => fields.some((field) => normalizeText(field(row)).includes(keyword))).slice(0, 30)
}

const filteredAISessions = computed(() =>
  filterRows(aiSessions.value, [(row) => row.title, (row) => row.type, (row) => row.context])
)

const filteredOrders = computed(() =>
  filterRows(workorders.value, [(row) => row.title, (row) => row.type_name, (row) => row.submitter, (row) => workorderStatusText(row.status), (row) => priorityText(row.priority)])
)

const filteredExecutions = computed(() =>
  filterRows(workflowExecutions.value, [(row) => row.workflow_name, (row) => row.trigger_by, (row) => row.trigger, (row) => workflowStatusText(row.status)])
)

const filteredWorkflows = computed(() =>
  filterRows(workflows.value, [(row) => row.name, (row) => row.category, (row) => row.trigger, (row) => row.description])
)

const filteredOncallNow = computed(() =>
  filterRows(oncallNow.value, [(row) => row.username, (row) => row.type, (row) => row.phone, (row) => row.email])
)

const filteredEscalations = computed(() =>
  filterRows(escalations.value, [(row) => row.name, (row) => row.schedule_id, (row) => row.description])
)

const filteredTerminalSessions = computed(() =>
  filterRows(terminalSessions.value, [(row) => row.operator, (row) => row.host, (row) => row.username, (row) => terminalStatusText(row.status), (row) => row.last_error])
)

const formatTime = (value) => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString()
}

const openCurrentPanel = () => {
  go(panelRouteMap[activePanel.value] || '/ai')
}

const canApprove = (row) => [0, 1].includes(Number(row?.status))
const canCancel = (row) => [0, 1, 2, 4].includes(Number(row?.status))
const onWorkorderSelectionChange = (rows) => {
  selectedWorkorderRows.value = Array.isArray(rows) ? rows : []
}

const onTerminalSelectionChange = (rows) => {
  selectedTerminalRows.value = Array.isArray(rows) ? rows : []
}

const approveOrderByID = (orderID, comment = '协作中心批量审批通过') =>
  axios.post(`/api/v1/workorder/orders/${orderID}/approve`, { approved: true, comment }, { headers: authHeaders() })

const cancelOrderByID = (orderID) =>
  axios.post(`/api/v1/workorder/orders/${orderID}/cancel`, {}, { headers: authHeaders() })

const closeTerminalSessionByID = (sessionID) =>
  axios.delete(`/api/v1/terminal/sessions/${sessionID}`, { headers: authHeaders() })

const purgeTerminalSessionByID = (sessionID) =>
  axios.delete(`/api/v1/terminal/sessions/${sessionID}/purge`, { headers: authHeaders() })

const runBatchWorkorderAction = async (rows, options) => {
  const actionRows = rows.filter((row) => row?.id && (options.filter ? options.filter(row) : true))
  const orderIDs = [...new Set(actionRows.map((row) => row.id))]
  if (!orderIDs.length) {
    ElMessage.info('没有可处置的工单')
    return
  }
  workorderBatching.value = true
  try {
    await ElMessageBox.confirm(options.message(orderIDs.length), options.title, { type: 'warning' })
    const settled = await Promise.allSettled(orderIDs.map((id) => options.action(id)))
    const success = settled.filter((item) => item.status === 'fulfilled').length
    const fail = settled.length - success
    ElMessage.success(`${options.successText}：成功 ${success}，失败 ${fail}`)
    selectedWorkorderRows.value = []
    if (workorderTableRef.value?.clearSelection) workorderTableRef.value.clearSelection()
    await refreshAll()
  } catch (err) {
    if (!isCancelError(err)) ElMessage.error(getErrorMessage(err, options.failText))
  } finally {
    workorderBatching.value = false
  }
}

const batchApproveSelectedWorkorders = async () => {
  await runBatchWorkorderAction(selectedWorkorderRows.value, {
    title: '批量审批通过',
    message: (count) => `确认批量通过 ${count} 个已选工单吗？`,
    filter: (row) => canApprove(row),
    action: (id) => approveOrderByID(id),
    successText: '批量审批完成',
    failText: '批量审批失败'
  })
}

const batchCancelSelectedWorkorders = async () => {
  await runBatchWorkorderAction(selectedWorkorderRows.value, {
    title: '批量取消工单',
    message: (count) => `确认批量取消 ${count} 个已选工单吗？`,
    filter: (row) => canCancel(row),
    action: (id) => cancelOrderByID(id),
    successText: '批量取消完成',
    failText: '批量取消失败'
  })
}

const batchApproveTimeoutWorkorders = async () => {
  const rows = filteredOrders.value.filter((row) => canApprove(row) && isWorkorderPendingTimeout(row))
  await runBatchWorkorderAction(rows, {
    title: '处置超时审批',
    message: (count) => `确认批量通过 ${count} 个超时审批工单吗？`,
    action: (id) => approveOrderByID(id, '协作中心自动处置超时审批'),
    successText: '超时审批处置完成',
    failText: '处置超时审批失败'
  })
}

const batchApproveWorkorderGroup = async (group) => {
  const rows = Array.isArray(group?.rows) ? group.rows : []
  await runBatchWorkorderAction(rows, {
    title: `按类型处置：${group?.label || '-'}`,
    message: (count) => `确认批量通过「${group?.label || '-'}」${count} 个工单吗？`,
    action: (id) => approveOrderByID(id, `按类型(${group?.label || '-'})批量处置`),
    successText: '分组处置完成',
    failText: '分组处置失败'
  })
}

const runBatchTerminalAction = async (rows, options) => {
  const actionRows = rows.filter((row) => row?.id && (options.filter ? options.filter(row) : true))
  const sessionIDs = [...new Set(actionRows.map((row) => row.id))]
  if (!sessionIDs.length) {
    ElMessage.info('没有可处置的终端会话')
    return
  }
  terminalBatching.value = true
  try {
    await ElMessageBox.confirm(options.message(sessionIDs.length), options.title, { type: 'warning' })
    const settled = await Promise.allSettled(sessionIDs.map((id) => options.action(id)))
    const success = settled.filter((item) => item.status === 'fulfilled').length
    const fail = settled.length - success
    ElMessage.success(`${options.successText}：成功 ${success}，失败 ${fail}`)
    selectedTerminalRows.value = []
    if (terminalTableRef.value?.clearSelection) terminalTableRef.value.clearSelection()
    await refreshAll()
  } catch (err) {
    if (!isCancelError(err)) ElMessage.error(getErrorMessage(err, options.failText))
  } finally {
    terminalBatching.value = false
  }
}

const batchCloseSelectedTerminals = async () => {
  await runBatchTerminalAction(selectedTerminalRows.value, {
    title: '批量关闭会话',
    message: (count) => `确认批量关闭 ${count} 个已连接终端会话吗？`,
    filter: (row) => Number(row.status) === 1,
    action: (id) => closeTerminalSessionByID(id),
    successText: '批量关闭完成',
    failText: '批量关闭失败'
  })
}

const batchPurgeSelectedTerminals = async () => {
  await runBatchTerminalAction(selectedTerminalRows.value, {
    title: '批量删除会话',
    message: (count) => `确认批量删除 ${count} 个已选终端会话吗？`,
    filter: (row) => Number(row.status) !== 1,
    action: (id) => purgeTerminalSessionByID(id),
    successText: '批量删除完成',
    failText: '批量删除失败'
  })
}

const batchPurgeFailedTerminals = async () => {
  const rows = filteredTerminalSessions.value.filter((row) => Number(row.status) === 3)
  await runBatchTerminalAction(rows, {
    title: '清理失败会话',
    message: (count) => `确认删除 ${count} 个连接失败会话吗？`,
    action: (id) => purgeTerminalSessionByID(id),
    successText: '失败会话清理完成',
    failText: '失败会话清理失败'
  })
}

const batchPurgeTerminalGroup = async (group) => {
  const rows = Array.isArray(group?.rows) ? group.rows : []
  await runBatchTerminalAction(rows, {
    title: `按主机清理：${group?.label || '-'}`,
    message: (count) => `确认删除主机「${group?.label || '-'}」${count} 个非活跃会话吗？`,
    action: (id) => purgeTerminalSessionByID(id),
    successText: '分组清理完成',
    failText: '分组清理失败'
  })
}

const isCICDWorkOrder = (row) => {
  if (!row?.form_data) return false
  try {
    const data = JSON.parse(row.form_data)
    return String(data?.source || '').toLowerCase() === 'cicd'
  } catch (_) {
    return false
  }
}

const approveOrder = async (row, approved) => {
  try {
    const { value } = await ElMessageBox.prompt(
      approved ? '可选：审批备注' : '请输入拒绝原因',
      approved ? '审批通过' : '审批拒绝',
      {
        inputType: 'textarea',
        inputValue: '',
        inputPlaceholder: approved ? '例如：同意执行' : '例如：变更窗口不符合要求'
      }
    )
    await axios.post(`/api/v1/workorder/orders/${row.id}/approve`, {
      approved,
      comment: value || ''
    }, { headers: authHeaders() })
    ElMessage.success('审批已提交')
    await refreshAll()
  } catch (err) {
    if (!isCancelError(err)) ElMessage.error(getErrorMessage(err, '审批失败'))
  }
}

const executeOrder = async (row) => {
  try {
    if (isCICDWorkOrder(row)) {
      await axios.post(`/api/v1/cicd/orders/${row.id}/execute`, {}, { headers: authHeaders() })
      ElMessage.success('已触发流水线执行')
    } else {
      await axios.post(`/api/v1/workorder/orders/${row.id}/execute`, {}, { headers: authHeaders() })
      ElMessage.success('已开始执行')
    }
    await refreshAll()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '执行失败'))
  }
}

const completeOrder = async (row) => {
  try {
    const { value } = await ElMessageBox.prompt('请输入执行结果', '完成工单', {
      inputType: 'textarea',
      inputPlaceholder: '例如：已完成并验证通过'
    })
    await axios.post(`/api/v1/workorder/orders/${row.id}/complete`, { result: value || '' }, { headers: authHeaders() })
    ElMessage.success('工单已完成')
    await refreshAll()
  } catch (err) {
    if (!isCancelError(err)) ElMessage.error(getErrorMessage(err, '操作失败'))
  }
}

const cancelOrder = async (row) => {
  try {
    await ElMessageBox.confirm(`确认取消工单「${row.title || row.id}」吗？`, '提示', { type: 'warning' })
    await axios.post(`/api/v1/workorder/orders/${row.id}/cancel`, {}, { headers: authHeaders() })
    ElMessage.success('工单已取消')
    await refreshAll()
  } catch (err) {
    if (!isCancelError(err)) ElMessage.error(getErrorMessage(err, '操作失败'))
  }
}

const parseJSONSafe = (text, fallback = {}) => {
  try {
    return JSON.parse(text || '{}')
  } catch (_) {
    return fallback
  }
}

const executeWorkflow = async (workflow) => {
  try {
    const defaultVars = workflow?.variables || '{}'
    const { value } = await ElMessageBox.prompt(
      '可选：覆盖本次运行变量（JSON）',
      `执行流程：${workflow?.name || ''}`,
      {
        inputType: 'textarea',
        inputValue: defaultVars,
        inputPlaceholder: '{"service":"nginx"}'
      }
    )
    const variables = parseJSONSafe((value || '').trim(), parseJSONSafe(defaultVars, {}))
    await axios.post(`/api/v1/workflow/workflows/${workflow.id}/execute`, { variables }, { headers: authHeaders() })
    ElMessage.success('流程已触发执行')
    await refreshAll()
    activePanel.value = 'workflow'
  } catch (err) {
    if (!isCancelError(err)) ElMessage.error(getErrorMessage(err, '执行流程失败'))
  }
}

const cancelExecution = async (row) => {
  try {
    await axios.post(`/api/v1/workflow/executions/${row.id}/cancel`, {}, { headers: authHeaders() })
    ElMessage.success('执行已取消')
    await refreshAll()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '取消执行失败'))
  }
}

const connectTerminal = (row) => {
  go(`/terminal?session_id=${row.id}`)
}

const closeTerminalSession = async (row) => {
  try {
    await closeTerminalSessionByID(row.id)
    ElMessage.success('会话已关闭')
    await refreshAll()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '关闭会话失败'))
  }
}

const purgeTerminalSession = async (row) => {
  try {
    await ElMessageBox.confirm(`确认删除终端会话 ${row.host || row.id} 吗？`, '提示', { type: 'warning' })
    await purgeTerminalSessionByID(row.id)
    ElMessage.success('会话已删除')
    await refreshAll()
  } catch (err) {
    if (!isCancelError(err)) ElMessage.error(getErrorMessage(err, '删除会话失败'))
  }
}

const safeArray = (res) => (Array.isArray(res?.data?.data) ? res.data.data : [])
const safeObject = (res) => (res?.data?.data && typeof res.data.data === 'object' ? res.data.data : {})

const refreshAll = async () => {
  loading.value = true
  try {
    const [aiSessionRes, orderRes, orderStatsRes, workflowRes, executionRes, workflowStatsRes, oncallNowRes, scheduleRes, escalationRes, terminalRes] = await Promise.allSettled([
      axios.get('/api/v1/ai/sessions', { headers: authHeaders() }),
      axios.get('/api/v1/workorder/orders', { headers: authHeaders() }),
      axios.get('/api/v1/workorder/stats', { headers: authHeaders() }),
      axios.get('/api/v1/workflow/workflows', { headers: authHeaders() }),
      axios.get('/api/v1/workflow/executions', { headers: authHeaders() }),
      axios.get('/api/v1/workflow/stats', { headers: authHeaders() }),
      axios.get('/api/v1/oncall/whoisoncall', { headers: authHeaders() }),
      axios.get('/api/v1/oncall/schedules', { headers: authHeaders() }),
      axios.get('/api/v1/oncall/escalations', { headers: authHeaders() }),
      axios.get('/api/v1/terminal/sessions', { headers: authHeaders() })
    ])

    aiSessions.value = aiSessionRes.status === 'fulfilled' ? safeArray(aiSessionRes.value) : []
    workorders.value = orderRes.status === 'fulfilled' ? safeArray(orderRes.value) : []
    workorderStats.value = orderStatsRes.status === 'fulfilled' ? safeObject(orderStatsRes.value) : {}
    workflows.value = workflowRes.status === 'fulfilled' ? safeArray(workflowRes.value) : []
    workflowExecutions.value = executionRes.status === 'fulfilled' ? safeArray(executionRes.value) : []
    workflowStats.value = workflowStatsRes.status === 'fulfilled' ? safeObject(workflowStatsRes.value) : {}
    oncallNow.value = oncallNowRes.status === 'fulfilled' ? safeArray(oncallNowRes.value) : []
    schedules.value = scheduleRes.status === 'fulfilled' ? safeArray(scheduleRes.value) : []
    escalations.value = escalationRes.status === 'fulfilled' ? safeArray(escalationRes.value) : []
    terminalSessions.value = terminalRes.status === 'fulfilled' ? safeArray(terminalRes.value) : []

    stats.aiSessions = aiSessions.value.length
    stats.workorderTotal = Number(workorderStats.value.total || workorders.value.length || 0)
    stats.workorderPending = Number(workorderStats.value.pending || workorders.value.filter((item) => [0, 1, 4].includes(Number(item.status))).length || 0)
    stats.workflowRunning = Number(workflowStats.value.running || 0)
    stats.workflowFailed = Number(workflowStats.value.failed || 0)
    stats.oncallNow = oncallNow.value.length
    stats.terminalActive = terminalSessions.value.filter((item) => Number(item.status) === 1).length
    stats.escalationTotal = escalations.value.length
    stats.scheduleTotal = schedules.value.length
    stats.terminalTotal = terminalSessions.value.length

    const failedCount = [aiSessionRes, orderRes, orderStatsRes, workflowRes, executionRes, workflowStatsRes, oncallNowRes, scheduleRes, escalationRes, terminalRes]
      .filter((item) => item.status === 'rejected').length
    if (failedCount > 0) {
      ElMessage.warning(`部分协作中心数据加载失败（${failedCount}项），已展示可用数据`)
    }
  } catch (err) {
    ElMessage.error(err?.response?.data?.message || err?.message || '加载协作中心失败')
  } finally {
    loading.value = false
  }
}

onMounted(refreshAll)
onMounted(() => {
  minuteTicker = window.setInterval(() => {
    nowTs.value = Date.now()
  }, 60 * 1000)
})
onUnmounted(() => {
  if (minuteTicker) {
    window.clearInterval(minuteTicker)
    minuteTicker = null
  }
})
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 12px; gap: 12px; }
.page-desc { color: var(--muted-text); margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; }
.module-tabs { display: flex; gap: 8px; flex-wrap: wrap; margin-bottom: 12px; }
.tab-item { cursor: pointer; user-select: none; }
.summary-row { margin-bottom: 12px; }
.summary-row :deep(.el-card) { margin-bottom: 8px; }
.metric-title { color: var(--muted-text); font-size: 12px; }
.metric-value { font-size: 20px; font-weight: 600; margin-top: 6px; color: var(--el-text-color-primary); }
.metric-value.ok { color: #67c23a; }
.metric-value.warning { color: #e6a23c; }
.metric-value.danger { color: #f56c6c; }
.metric-sub { margin-top: 4px; color: var(--muted-text); font-size: 12px; }
.health-row { display: flex; justify-content: space-between; align-items: center; margin-bottom: 6px; }
.health-row strong { font-size: 15px; }
.mtop { margin-top: 12px; }

.integration-card {
  margin-top: 12px;
}

.integration-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.integration-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.panel-search {
  width: 280px;
}

.panel-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 10px;
}

.panel-toolbar-left,
.panel-toolbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.group-tag-list {
  margin-top: 10px;
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.group-tag-item {
  cursor: pointer;
}

.inline-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.integration-tabs :deep(.el-tabs__header) {
  margin-bottom: 10px;
}

@media (max-width: 1100px) {
  .integration-header {
    align-items: flex-start;
    flex-direction: column;
  }

  .integration-actions {
    width: 100%;
    flex-wrap: wrap;
  }

  .panel-toolbar {
    align-items: flex-start;
    flex-direction: column;
  }

  .panel-toolbar-left,
  .panel-toolbar-right {
    width: 100%;
  }

  .panel-search {
    width: 100%;
  }
}
</style>
