<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>交付中心</h2>
        <p class="page-desc">围绕“流水线-执行-发布-工单”构建一屏闭环，贴近企业变更管控流程。</p>
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
        <el-card><div class="metric-title">流水线总数</div><div class="metric-value">{{ stats.pipelineTotal }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">启用流水线</div><div class="metric-value ok">{{ stats.pipelineEnabled }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">需审批流水线</div><div class="metric-value warning">{{ stats.pipelineNeedApprove }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">执行中</div><div class="metric-value">{{ stats.executionRunning }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">失败执行</div><div class="metric-value danger">{{ stats.executionFailed }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">待处理工单</div><div class="metric-value danger">{{ stats.workorderPending }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">已发布版本</div><div class="metric-value ok">{{ stats.released }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card>
          <div class="metric-title">待处置积压</div>
          <div class="metric-value warning">{{ pendingBacklog }}</div>
          <div class="metric-sub">超时审批 {{ pendingApprovalTimeout }}</div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="12">
      <el-col :span="14">
        <el-card>
          <template #header>
            <div class="order-header">
              <span>待处理工单（优先）</span>
              <div class="order-actions">
                <el-tag type="warning" effect="light">待处理 {{ pendingOrders.length }}</el-tag>
                <el-tag type="danger" effect="light">超时审批 {{ pendingApprovalTimeout }}</el-tag>
                <el-button size="small" type="success" plain :loading="orderBatching" :disabled="!selectedApprovableCount" @click="batchApproveSelectedOrders">批量通过已选</el-button>
                <el-button size="small" type="danger" plain :loading="orderBatching" :disabled="!selectedCancelableCount" @click="batchCancelSelectedOrders">批量取消已选</el-button>
                <el-button size="small" plain :loading="orderBatching" :disabled="!pendingApprovalTimeout" @click="batchApproveTimeoutOrders">处置超时审批</el-button>
              </div>
            </div>
          </template>
          <el-table
            ref="pendingOrderTableRef"
            :fit="true"
            :data="pendingOrders"
            size="small"
            max-height="300"
            empty-text="暂无待处理工单"
            @selection-change="onPendingOrderSelectionChange"
          >
            <el-table-column type="selection" width="46" />
            <el-table-column prop="title" label="工单" min-width="170" />
            <el-table-column prop="type_name" label="类型" width="110" />
            <el-table-column label="优先级" width="90">
              <template #default="{ row }">
                <el-tag :type="priorityTag(row.priority)">{{ priorityText(row.priority) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="状态" width="100">
              <template #default="{ row }">
                <el-tag :type="workorderStatusTag(row.status)">{{ workorderStatusText(row.status) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="创建时间" min-width="150">
              <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
            </el-table-column>
            <el-table-column label="等待时长" width="110">
              <template #default="{ row }">
                <el-tag :type="isOrderApprovalTimeout(row) ? 'warning' : 'success'">
                  {{ formatWaitDuration(row.created_at) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" min-width="240">
              <template #default="{ row }">
                <el-space wrap>
                  <el-button size="small" link type="primary" @click="openOrder(row)">详情</el-button>
                  <el-button v-if="canApprove(row)" size="small" link type="success" @click="approveOrder(row, true)">通过</el-button>
                  <el-button v-if="canApprove(row)" size="small" link type="warning" @click="approveOrder(row, false)">拒绝</el-button>
                  <el-button v-if="Number(row.status) === 2" size="small" link type="primary" @click="executeOrder(row)">执行</el-button>
                  <el-button v-if="Number(row.status) === 4" size="small" link type="success" @click="completeOrder(row)">完成</el-button>
                  <el-button v-if="canCancel(row)" size="small" link type="danger" @click="cancelOrder(row)">取消</el-button>
                </el-space>
              </template>
            </el-table-column>
          </el-table>
          <div class="order-group-list">
            <el-tag
              v-for="group in pendingOrderApproveGroups"
              :key="group.key"
              type="warning"
              effect="plain"
              class="order-group-tag"
              @click="batchApproveOrderGroup(group)"
            >
              {{ group.label }} · 待审批 {{ group.count }}
            </el-tag>
          </div>
        </el-card>

        <el-card class="mt-12">
          <template #header>最近执行记录</template>
          <el-table :fit="true" :data="recentExecutions" size="small" max-height="250" empty-text="暂无执行记录">
            <el-table-column prop="pipeline_name" label="流水线" min-width="170" />
            <el-table-column prop="trigger" label="触发方式" width="100" />
            <el-table-column label="状态" width="100">
              <template #default="{ row }">
                <el-tag :type="executionStatusTag(row.status)">{{ executionStatusText(row.status) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="发起人" width="110" prop="trigger_by" />
            <el-table-column label="开始时间" min-width="150">
              <template #default="{ row }">{{ formatTime(row.started_at) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="130">
              <template #default="{ row }">
                <el-button v-if="Number(row.status) === 0" size="small" link type="danger" @click="cancelExecution(row)">取消</el-button>
                <el-button size="small" link type="primary" @click="go('/cicd/executions')">详情</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>

      <el-col :span="10">
        <el-card>
          <template #header>交付健康</template>
          <div class="health-row"><span>执行成功率</span><strong>{{ executionSuccessRate }}%</strong></div>
          <el-progress :percentage="executionSuccessRate" :stroke-width="14" />
          <div class="health-row mtop"><span>按计划发布率</span><strong>{{ releaseOnPlanRate }}%</strong></div>
          <el-progress :percentage="releaseOnPlanRate" :stroke-width="14" status="success" />
          <el-divider />
          <div class="health-row"><span>定时任务</span><strong>{{ stats.scheduleTotal }}</strong></div>
          <div class="health-row"><span>待审批工单</span><strong>{{ stats.workorderPending }}</strong></div>
          <div class="health-row"><span>超时审批工单</span><strong>{{ pendingApprovalTimeout }}</strong></div>
          <div class="health-row"><span>长时间执行</span><strong>{{ executionLongRunning }}</strong></div>
          <div class="health-row"><span>异常定时任务</span><strong>{{ scheduleStale }}</strong></div>
          <div class="health-row"><span>已完成工单</span><strong>{{ stats.workorderDone }}</strong></div>
        </el-card>

        <el-card class="mt-12">
          <template #header>即将执行的定时发布</template>
          <el-table :fit="true" :data="nextSchedules" size="small" max-height="240" empty-text="暂无计划任务">
            <el-table-column prop="name" label="任务" min-width="140" />
            <el-table-column prop="cron" label="CRON" min-width="130" />
            <el-table-column label="下次执行" min-width="150">
              <template #default="{ row }">{{ formatTime(row.next_run_at) }}</template>
            </el-table-column>
            <el-table-column label="计划时效" width="110">
              <template #default="{ row }">
                <el-tag :type="isScheduleStale(row) ? 'warning' : 'success'">
                  {{ isScheduleStale(row) ? '待修复' : '正常' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="130">
              <template #default="{ row }">
                <el-button size="small" link type="primary" @click="toggleSchedule(row)">{{ row.enabled ? '停用' : '启用' }}</el-button>
                <el-button size="small" link type="success" @click="runScheduleNow(row)">立即执行</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <el-card class="integration-card">
      <template #header>
        <div class="integration-header">
          <span>交付融合视图</span>
          <div class="integration-actions">
            <el-input
              v-model="panelKeyword"
              clearable
              size="small"
              class="panel-search"
              placeholder="筛选名称、分支、状态、负责人..."
            />
            <el-button size="small" type="primary" plain @click="openCurrentPanel">进入完整页面</el-button>
          </div>
        </div>
      </template>

      <el-tabs v-model="activePanel" class="integration-tabs">
        <el-tab-pane label="流水线" name="pipelines">
          <el-table :fit="true" :data="filteredPipelines" size="small" max-height="360" empty-text="暂无流水线">
            <el-table-column prop="name" label="名称" min-width="170" />
            <el-table-column prop="repo_url" label="仓库" min-width="180" show-overflow-tooltip />
            <el-table-column prop="branch" label="分支" min-width="120" />
            <el-table-column label="启用" width="90">
              <template #default="{ row }">
                <el-tag :type="Number(row.status) === 1 ? 'success' : 'info'">{{ Number(row.status) === 1 ? '是' : '否' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="需审批" width="90">
              <template #default="{ row }">
                <el-tag :type="row.require_approval ? 'warning' : 'info'">{{ row.require_approval ? '是' : '否' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" min-width="180">
              <template #default="{ row }">
                <el-button size="small" link type="primary" @click="triggerPipeline(row)">触发</el-button>
                <el-button size="small" link type="success" @click="syncPipeline(row)">同步</el-button>
                <el-button size="small" link @click="go('/cicd/pipelines')">详情</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="执行记录" name="executions">
          <el-table :fit="true" :data="filteredExecutions" size="small" max-height="360" empty-text="暂无执行记录">
            <el-table-column prop="pipeline_name" label="流水线" min-width="170" />
            <el-table-column prop="trigger" label="触发方式" width="110" />
            <el-table-column label="状态" width="100">
              <template #default="{ row }">
                <el-tag :type="executionStatusTag(row.status)">{{ executionStatusText(row.status) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="trigger_by" label="发起人" width="110" />
            <el-table-column label="开始时间" min-width="160">
              <template #default="{ row }">{{ formatTime(row.started_at) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="130">
              <template #default="{ row }">
                <el-button v-if="Number(row.status) === 0" size="small" link type="danger" @click="cancelExecution(row)">取消</el-button>
                <el-button size="small" link @click="go('/cicd/executions')">详情</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="定时发布" name="schedules">
          <el-table :fit="true" :data="filteredSchedules" size="small" max-height="360" empty-text="暂无定时任务">
            <el-table-column prop="name" label="任务名称" min-width="160" />
            <el-table-column prop="cron" label="CRON" min-width="140" />
            <el-table-column label="启用" width="90">
              <template #default="{ row }">
                <el-tag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? '是' : '否' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="下次执行" min-width="170">
              <template #default="{ row }">{{ formatTime(row.next_run_at) }}</template>
            </el-table-column>
            <el-table-column label="计划时效" width="110">
              <template #default="{ row }">
                <el-tag :type="isScheduleStale(row) ? 'warning' : 'success'">
                  {{ isScheduleStale(row) ? '待修复' : '正常' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" min-width="180">
              <template #default="{ row }">
                <el-button size="small" link type="primary" @click="toggleSchedule(row)">{{ row.enabled ? '停用' : '启用' }}</el-button>
                <el-button size="small" link type="success" @click="runScheduleNow(row)">执行</el-button>
                <el-button size="small" link @click="go('/cicd/schedules')">详情</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="发布管理" name="releases">
          <el-table :fit="true" :data="filteredReleases" size="small" max-height="360" empty-text="暂无发布记录">
            <el-table-column prop="name" label="发布名称" min-width="160" />
            <el-table-column prop="version" label="版本" width="110" />
            <el-table-column prop="environment" label="环境" width="110" />
            <el-table-column label="状态" width="100">
              <template #default="{ row }">
                <el-tag :type="releaseStatusTag(row.status)">{{ releaseStatusText(row.status) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="发布时间" min-width="170">
              <template #default="{ row }">{{ formatTime(row.released_at || row.created_at) }}</template>
            </el-table-column>
            <el-table-column label="操作" min-width="180">
              <template #default="{ row }">
                <el-button v-if="Number(row.status) === 0" size="small" link type="success" @click="publishRelease(row)">发布</el-button>
                <el-button v-if="Number(row.status) === 1" size="small" link type="warning" @click="rollbackRelease(row)">回滚</el-button>
                <el-button size="small" link @click="go('/cicd/releases')">详情</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="工单审批" name="workorders">
          <el-table :fit="true" :data="filteredOrders" size="small" max-height="360" empty-text="暂无工单">
            <el-table-column prop="title" label="标题" min-width="180" />
            <el-table-column prop="type_name" label="类型" width="120" />
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
            <el-table-column label="创建时间" min-width="170">
              <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
            </el-table-column>
            <el-table-column label="操作" min-width="240">
              <template #default="{ row }">
                <el-space wrap>
                  <el-button size="small" link type="primary" @click="openOrder(row)">详情</el-button>
                  <el-button v-if="canApprove(row)" size="small" link type="success" @click="approveOrder(row, true)">通过</el-button>
                  <el-button v-if="canApprove(row)" size="small" link type="warning" @click="approveOrder(row, false)">拒绝</el-button>
                  <el-button v-if="Number(row.status) === 2" size="small" link type="primary" @click="executeOrder(row)">执行</el-button>
                  <el-button v-if="Number(row.status) === 4" size="small" link type="success" @click="completeOrder(row)">完成</el-button>
                  <el-button v-if="canCancel(row)" size="small" link type="danger" @click="cancelOrder(row)">取消</el-button>
                </el-space>
              </template>
            </el-table-column>
          </el-table>
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
import { requestApplyWorkspaceCategory } from '@/utils/workspace'
import { getErrorMessage, isCancelError } from '@/utils/error'

const router = useRouter()
const loading = ref(false)
const pipelines = ref([])
const executions = ref([])
const schedules = ref([])
const releases = ref([])
const orders = ref([])
const orderStats = ref({ total: 0, pending: 0, processing: 0, completed: 0 })
const activePanel = ref('pipelines')
const panelKeyword = ref('')
const orderBatching = ref(false)
const nowTs = ref(Date.now())
const pendingOrderTableRef = ref(null)
const selectedPendingOrders = ref([])
let minuteTicker = null

const stats = reactive({
  pipelineTotal: 0,
  pipelineEnabled: 0,
  pipelineNeedApprove: 0,
  executionRunning: 0,
  executionFailed: 0,
  executionSuccess: 0,
  scheduleTotal: 0,
  scheduleEnabled: 0,
  released: 0,
  releaseTotal: 0,
  workorderPending: 0,
  workorderDone: 0
})

const quickTabs = [
  { label: '流水线管理', path: '/cicd/pipelines' },
  { label: '执行记录', path: '/cicd/executions' },
  { label: '定时发布', path: '/cicd/schedules' },
  { label: '发布管理', path: '/cicd/releases' },
  { label: '工单管理', path: '/workorder/tickets' }
]

const panelRouteMap = {
  pipelines: '/cicd/pipelines',
  executions: '/cicd/executions',
  schedules: '/cicd/schedules',
  releases: '/cicd/releases',
  workorders: '/workorder/tickets'
}

const applyRecommendedWorkspace = () => requestApplyWorkspaceCategory('delivery', 'hub-delivery')

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })
const go = (path) => router.push(path)

const normalizeText = (value) => String(value ?? '').trim().toLowerCase()
const filterRows = (rows, fields) => {
  const keyword = normalizeText(panelKeyword.value)
  const base = Array.isArray(rows) ? rows : []
  if (!keyword) return base.slice(0, 20)
  return base
    .filter((row) => fields.some((field) => normalizeText(field(row)).includes(keyword)))
    .slice(0, 20)
}

const pendingOrders = computed(() => {
  return orders.value
    .filter((item) => Number(item.status) === 0 || Number(item.status) === 1 || Number(item.status) === 4)
    .sort((a, b) => Number(a.priority || 4) - Number(b.priority || 4))
    .slice(0, 12)
})

const selectedApprovableCount = computed(() =>
  selectedPendingOrders.value.filter((item) => canApprove(item)).length
)

const selectedCancelableCount = computed(() =>
  selectedPendingOrders.value.filter((item) => canCancel(item)).length
)

const pendingOrderApproveGroups = computed(() => {
  const groups = new Map()
  pendingOrders.value
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

const recentExecutions = computed(() => executions.value.slice(0, 12))

const nextSchedules = computed(() => {
  return schedules.value
    .filter((item) => !!item.next_run_at)
    .sort((a, b) => new Date(a.next_run_at).getTime() - new Date(b.next_run_at).getTime())
    .slice(0, 10)
})

const filteredPipelines = computed(() =>
  filterRows(pipelines.value, [(row) => row.name, (row) => row.repo_url, (row) => row.branch, (row) => row.description])
)

const filteredExecutions = computed(() =>
  filterRows(executions.value, [(row) => row.pipeline_name, (row) => row.trigger, (row) => row.trigger_by, (row) => executionStatusText(row.status)])
)

const filteredSchedules = computed(() =>
  filterRows(schedules.value, [(row) => row.name, (row) => row.cron, (row) => row.pipeline_name, (row) => row.next_run_at])
)

const filteredReleases = computed(() =>
  filterRows(releases.value, [(row) => row.name, (row) => row.version, (row) => row.environment, (row) => releaseStatusText(row.status)])
)

const filteredOrders = computed(() =>
  filterRows(orders.value, [(row) => row.title, (row) => row.type_name, (row) => row.applicant, (row) => workorderStatusText(row.status)])
)

const executionSuccessRate = computed(() => {
  const finished = stats.executionSuccess + stats.executionFailed
  if (!finished) return 0
  return Math.round((stats.executionSuccess / finished) * 100)
})

const releaseOnPlanRate = computed(() => {
  if (!stats.releaseTotal) return 0
  return Math.round((stats.released / stats.releaseTotal) * 100)
})

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

const formatWaitDuration = (value) => {
  const minutes = elapsedMinutes(value)
  if (minutes < 60) return `${minutes}m`
  const hours = Math.floor(minutes / 60)
  const remain = minutes % 60
  if (hours < 24) return `${hours}h${remain}m`
  const days = Math.floor(hours / 24)
  return `${days}d${hours % 24}h`
}

const isOrderApprovalTimeout = (row) => {
  const status = Number(row?.status)
  if (status !== 0 && status !== 1) return false
  return elapsedMinutes(row?.created_at) >= 120
}

const isExecutionLongRunning = (row) => Number(row?.status) === 0 && elapsedMinutes(row?.started_at) >= 30

const isScheduleStale = (row) => {
  if (!row?.enabled) return false
  const nextTs = parseTimestamp(row?.next_run_at)
  if (!nextTs) return true
  return nextTs < (nowTs.value - 5 * 60 * 1000)
}

const pendingApprovalTimeout = computed(() => orders.value.filter((item) => isOrderApprovalTimeout(item)).length)
const executionLongRunning = computed(() => executions.value.filter((item) => isExecutionLongRunning(item)).length)
const scheduleStale = computed(() => schedules.value.filter((item) => isScheduleStale(item)).length)
const pendingBacklog = computed(() => pendingApprovalTimeout.value + executionLongRunning.value + scheduleStale.value)

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
  if (v === 3) return 'primary'
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
  return '-'
}

const workorderStatusTag = (value) => {
  const v = Number(value)
  if (v === 0 || v === 1) return 'warning'
  if (v === 2) return 'success'
  if (v === 4) return 'primary'
  if (v === 5) return 'success'
  if (v === 3 || v === 6) return 'info'
  return ''
}

const executionStatusText = (value) => {
  const v = Number(value)
  if (v === 0) return '运行中'
  if (v === 1) return '成功'
  if (v === 2) return '失败'
  if (v === 3) return '取消'
  return '-'
}

const executionStatusTag = (value) => {
  const v = Number(value)
  if (v === 1) return 'success'
  if (v === 0) return 'warning'
  if (v === 3) return 'info'
  return 'danger'
}

const releaseStatusText = (value) => {
  const v = Number(value)
  if (v === 0) return '待发布'
  if (v === 1) return '已发布'
  if (v === 2) return '已回滚'
  if (v === 3) return '失败'
  return '-'
}

const releaseStatusTag = (value) => {
  const v = Number(value)
  if (v === 1) return 'success'
  if (v === 0) return 'warning'
  if (v === 2) return 'info'
  return 'danger'
}

const formatTime = (value) => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString()
}

const openOrder = (row) => {
  router.push({ path: '/workorder/tickets', query: row?.id ? { id: row.id } : undefined })
}

const canApprove = (row) => [0, 1].includes(Number(row?.status))
const canCancel = (row) => [0, 1, 2, 4].includes(Number(row?.status))
const onPendingOrderSelectionChange = (rows) => {
  selectedPendingOrders.value = Array.isArray(rows) ? rows : []
}

const approveOrderByID = (orderID, comment = '交付中心批量审批通过') =>
  axios.post(`/api/v1/workorder/orders/${orderID}/approve`, { approved: true, comment }, { headers: authHeaders() })

const cancelOrderByID = (orderID) =>
  axios.post(`/api/v1/workorder/orders/${orderID}/cancel`, {}, { headers: authHeaders() })

const runBatchOrderAction = async (rows, options) => {
  const actionRows = rows.filter((row) => row?.id && (options.filter ? options.filter(row) : true))
  const orderIDs = [...new Set(actionRows.map((row) => row.id))]
  if (!orderIDs.length) {
    ElMessage.info('没有可处置的工单')
    return
  }
  orderBatching.value = true
  try {
    await ElMessageBox.confirm(options.message(orderIDs.length), options.title, { type: 'warning' })
    const settled = await Promise.allSettled(orderIDs.map((id) => options.action(id)))
    const success = settled.filter((item) => item.status === 'fulfilled').length
    const fail = settled.length - success
    ElMessage.success(`${options.successText}：成功 ${success}，失败 ${fail}`)
    selectedPendingOrders.value = []
    if (pendingOrderTableRef.value?.clearSelection) pendingOrderTableRef.value.clearSelection()
    await refreshAll()
  } catch (err) {
    if (!isCancelError(err)) ElMessage.error(getErrorMessage(err, options.failText))
  } finally {
    orderBatching.value = false
  }
}

const batchApproveSelectedOrders = async () => {
  await runBatchOrderAction(selectedPendingOrders.value, {
    title: '批量审批通过',
    message: (count) => `确认批量通过 ${count} 个已选工单吗？`,
    filter: (row) => canApprove(row),
    action: (id) => approveOrderByID(id),
    successText: '批量审批完成',
    failText: '批量审批失败'
  })
}

const batchCancelSelectedOrders = async () => {
  await runBatchOrderAction(selectedPendingOrders.value, {
    title: '批量取消工单',
    message: (count) => `确认批量取消 ${count} 个已选工单吗？`,
    filter: (row) => canCancel(row),
    action: (id) => cancelOrderByID(id),
    successText: '批量取消完成',
    failText: '批量取消失败'
  })
}

const batchApproveTimeoutOrders = async () => {
  const rows = pendingOrders.value.filter((row) => isOrderApprovalTimeout(row) && canApprove(row))
  await runBatchOrderAction(rows, {
    title: '处置超时审批',
    message: (count) => `确认批量通过 ${count} 个超时待审批工单吗？`,
    action: (id) => approveOrderByID(id, '交付中心自动处置超时审批'),
    successText: '超时审批处置完成',
    failText: '处置超时审批失败'
  })
}

const batchApproveOrderGroup = async (group) => {
  const rows = Array.isArray(group?.rows) ? group.rows : []
  await runBatchOrderAction(rows, {
    title: `按类型处置：${group?.label || '-'}`,
    message: (count) => `确认批量通过「${group?.label || '-'}」${count} 个工单吗？`,
    action: (id) => approveOrderByID(id, `按类型(${group?.label || '-'})批量处置通过`),
    successText: '分组处置完成',
    failText: '分组处置失败'
  })
}

const safeArray = (res) => (Array.isArray(res?.data?.data) ? res.data.data : [])
const openCurrentPanel = () => go(panelRouteMap[activePanel.value] || '/delivery/center')

const approveOrder = async (row, approved) => {
  try {
    const title = approved ? '审批通过' : '审批拒绝'
    const prompt = approved ? `确认通过工单「${row.title || row.id}」吗？` : `请输入拒绝原因（工单：${row.title || row.id}）`
    let comment = ''
    if (approved) {
      await ElMessageBox.confirm(prompt, title, { type: 'warning' })
    } else {
      const { value } = await ElMessageBox.prompt(prompt, title, {
        inputType: 'textarea',
        inputPlaceholder: '请输入拒绝原因',
        inputValidator: (v) => (String(v || '').trim() ? true : '拒绝原因不能为空')
      })
      comment = value
    }
    await axios.post(
      `/api/v1/workorder/orders/${row.id}/approve`,
      { approved, comment },
      { headers: authHeaders() }
    )
    ElMessage.success('审批完成')
    await refreshAll()
  } catch (err) {
    if (!isCancelError(err)) ElMessage.error(getErrorMessage(err, '审批失败'))
  }
}

const executeOrder = async (row) => {
  try {
    let executedByCICD = false
    try {
      await axios.post(`/api/v1/cicd/orders/${row.id}/execute`, {}, { headers: authHeaders() })
      executedByCICD = true
    } catch {
      await axios.post(`/api/v1/workorder/orders/${row.id}/execute`, {}, { headers: authHeaders() })
    }
    ElMessage.success(executedByCICD ? '已触发CI/CD执行' : '工单已进入执行')
    await refreshAll()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '执行工单失败'))
  }
}

const completeOrder = async (row) => {
  try {
    const { value } = await ElMessageBox.prompt(`请输入工单「${row.title || row.id}」执行结果`, '完成工单', {
      inputType: 'textarea',
      inputPlaceholder: '例如：发布完成，验证通过',
      inputValidator: (v) => (String(v || '').trim() ? true : '执行结果不能为空')
    })
    await axios.post(
      `/api/v1/workorder/orders/${row.id}/complete`,
      { result: value },
      { headers: authHeaders() }
    )
    ElMessage.success('工单已完成')
    await refreshAll()
  } catch (err) {
    if (!isCancelError(err)) ElMessage.error(getErrorMessage(err, '完成工单失败'))
  }
}

const cancelOrder = async (row) => {
  try {
    await ElMessageBox.confirm(`确认取消工单「${row.title || row.id}」吗？`, '提示', { type: 'warning' })
    await axios.post(`/api/v1/workorder/orders/${row.id}/cancel`, {}, { headers: authHeaders() })
    ElMessage.success('工单已取消')
    await refreshAll()
  } catch (err) {
    if (!isCancelError(err)) ElMessage.error(getErrorMessage(err, '取消工单失败'))
  }
}

const triggerPipeline = async (row) => {
  try {
    await axios.post(
      `/api/v1/cicd/pipelines/${row.id}/trigger`,
      { reason: '来自交付中心一键触发' },
      { headers: authHeaders() }
    )
    ElMessage.success('已触发流水线')
    await refreshAll()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '触发流水线失败'))
  }
}

const syncPipeline = async (row) => {
  try {
    await axios.post(`/api/v1/cicd/pipelines/${row.id}/sync`, {}, { headers: authHeaders() })
    ElMessage.success('流水线已同步')
    await refreshAll()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '同步流水线失败'))
  }
}

const cancelExecution = async (row) => {
  try {
    await ElMessageBox.confirm(`确认取消执行「${row.pipeline_name || row.id}」吗？`, '提示', { type: 'warning' })
    await axios.post(`/api/v1/cicd/executions/${row.id}/cancel`, {}, { headers: authHeaders() })
    ElMessage.success('执行已取消')
    await refreshAll()
  } catch (err) {
    if (!isCancelError(err)) ElMessage.error(getErrorMessage(err, '取消执行失败'))
  }
}

const toggleSchedule = async (row) => {
  try {
    await axios.post(`/api/v1/cicd/schedules/${row.id}/toggle`, {}, { headers: authHeaders() })
    ElMessage.success(`定时任务已${row.enabled ? '停用' : '启用'}`)
    await refreshAll()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '切换定时任务失败'))
  }
}

const runScheduleNow = async (row) => {
  try {
    if (!row.pipeline_id) {
      ElMessage.warning('该定时任务未关联流水线')
      return
    }
    await axios.post(
      `/api/v1/cicd/pipelines/${row.pipeline_id}/trigger`,
      { reason: `定时任务手动触发: ${row.name || row.id}` },
      { headers: authHeaders() }
    )
    ElMessage.success('已触发对应流水线执行')
    await refreshAll()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '立即执行失败'))
  }
}

const updateReleaseStatus = async (row, status, successText) => {
  await axios.put(
    `/api/v1/cicd/releases/${row.id}`,
    {
      ...row,
      status,
      release_at: status === 1 ? new Date().toISOString() : row.release_at
    },
    { headers: authHeaders() }
  )
  ElMessage.success(successText)
  await refreshAll()
}

const publishRelease = async (row) => {
  try {
    await ElMessageBox.confirm(`确认将版本 ${row.version || row.name} 标记为已发布吗？`, '发布确认', { type: 'warning' })
    await updateReleaseStatus(row, 1, '发布状态已更新')
  } catch (err) {
    if (!isCancelError(err)) ElMessage.error(getErrorMessage(err, '发布失败'))
  }
}

const rollbackRelease = async (row) => {
  try {
    await ElMessageBox.confirm(`确认回滚发布 ${row.version || row.name} 吗？`, '回滚确认', { type: 'warning' })
    await updateReleaseStatus(row, 2, '已标记为回滚')
  } catch (err) {
    if (!isCancelError(err)) ElMessage.error(getErrorMessage(err, '回滚失败'))
  }
}

const refreshAll = async () => {
  loading.value = true
  try {
    const [pipelineRes, executionRes, scheduleRes, releaseRes, orderRes, orderStatsRes] = await Promise.allSettled([
      axios.get('/api/v1/cicd/pipelines', { headers: authHeaders() }),
      axios.get('/api/v1/cicd/executions', { headers: authHeaders() }),
      axios.get('/api/v1/cicd/schedules', { headers: authHeaders() }),
      axios.get('/api/v1/cicd/releases', { headers: authHeaders() }),
      axios.get('/api/v1/workorder/orders', { headers: authHeaders(), params: { my_pending: true } }),
      axios.get('/api/v1/workorder/stats', { headers: authHeaders() })
    ])

    pipelines.value = pipelineRes.status === 'fulfilled' ? safeArray(pipelineRes.value) : []
    executions.value = executionRes.status === 'fulfilled' ? safeArray(executionRes.value) : []
    schedules.value = scheduleRes.status === 'fulfilled' ? safeArray(scheduleRes.value) : []
    releases.value = releaseRes.status === 'fulfilled' ? safeArray(releaseRes.value) : []
    orders.value = orderRes.status === 'fulfilled' ? safeArray(orderRes.value) : []
    orderStats.value = orderStatsRes.status === 'fulfilled' ? (orderStatsRes.value?.data?.data || orderStats.value) : orderStats.value

    stats.pipelineTotal = pipelines.value.length
    stats.pipelineEnabled = pipelines.value.filter((item) => Number(item.status) === 1).length
    stats.pipelineNeedApprove = pipelines.value.filter((item) => Boolean(item.require_approval)).length

    stats.executionRunning = executions.value.filter((item) => Number(item.status) === 0).length
    stats.executionSuccess = executions.value.filter((item) => Number(item.status) === 1).length
    stats.executionFailed = executions.value.filter((item) => Number(item.status) === 2).length

    stats.scheduleTotal = schedules.value.length
    stats.scheduleEnabled = schedules.value.filter((item) => Boolean(item.enabled)).length

    stats.releaseTotal = releases.value.length
    stats.released = releases.value.filter((item) => Number(item.status) === 1).length

    stats.workorderPending = Number(orderStats.value?.pending || 0)
    stats.workorderDone = Number(orderStats.value?.completed || 0)

    const failedCount = [pipelineRes, executionRes, scheduleRes, releaseRes, orderRes, orderStatsRes].filter((r) => r.status === 'rejected').length
    if (failedCount > 0) {
      ElMessage.warning(`部分交付数据加载失败(${failedCount}项)，已展示可用数据`)
    }
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '加载交付中心失败'))
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
.mt-12 { margin-top: 12px; }

.order-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.order-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.order-group-list {
  margin-top: 10px;
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.order-group-tag {
  cursor: pointer;
}

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
  width: 260px;
}

.integration-tabs :deep(.el-tabs__header) {
  margin-bottom: 10px;
}

@media (max-width: 1100px) {
  .order-header {
    align-items: flex-start;
    flex-direction: column;
  }

  .order-actions {
    width: 100%;
  }

  .integration-header {
    align-items: flex-start;
    flex-direction: column;
  }

  .integration-actions {
    width: 100%;
    flex-wrap: wrap;
  }

  .panel-search {
    width: 100%;
  }
}
</style>
