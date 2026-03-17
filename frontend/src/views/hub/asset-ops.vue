<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>资产作战台</h2>
        <p class="page-desc">聚合 CMDB、网络设备、防火墙、堡垒机会话与风控事件，按运维处置流程联动。</p>
      </div>
      <div class="page-actions">
        <el-button type="primary" plain @click="applyRecommendedWorkspace">推荐工作台</el-button>
        <el-button :loading="syncingNetworkFromFirewall" icon="RefreshRight" @click="syncNetworkDevicesFromFirewalls">同步防火墙资产</el-button>
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
        <el-card><div class="metric-title">主机总数</div><div class="metric-value">{{ stats.hostTotal }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">离线主机</div><div class="metric-value danger">{{ stats.hostOffline }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">网络设备</div><div class="metric-value">{{ stats.networkTotal }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">离线网络设备</div><div class="metric-value warning">{{ stats.networkOffline }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">防火墙告警</div><div class="metric-value danger">{{ stats.firewallAlert }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">待审批会话</div><div class="metric-value warning">{{ stats.jumpPending }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">活跃会话</div><div class="metric-value ok">{{ stats.jumpActive }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card>
          <div class="metric-title">待处置积压</div>
          <div class="metric-value warning">{{ stats.pendingBacklog }}</div>
          <div class="metric-sub">高危风控 {{ stats.riskCritical }}</div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="12">
      <el-col :span="10">
        <el-card>
          <template #header>作战健康</template>
          <div class="health-row"><span>主机在线率</span><strong>{{ hostOnlineRate }}%</strong></div>
          <el-progress :percentage="hostOnlineRate" :stroke-width="14" />
          <div class="health-row mtop"><span>网络设备在线率</span><strong>{{ networkOnlineRate }}%</strong></div>
          <el-progress :percentage="networkOnlineRate" :stroke-width="14" status="success" />
          <el-divider />
          <div class="health-row"><span>堡垒机资产</span><strong>{{ jumpAssets.length }}</strong></div>
          <div class="health-row"><span>待复检对象</span><strong>{{ stats.recheckPending }}</strong></div>
          <div class="health-row"><span>超时待审批会话</span><strong>{{ stats.pendingApprovalTimeout }}</strong></div>
          <div class="health-row"><span>风控阻断命令(7天)</span><strong>{{ commandStats.commands_blocked || 0 }}</strong></div>
          <div class="health-row"><span>风控命令总量(7天)</span><strong>{{ commandStats.commands_window || 0 }}</strong></div>
        </el-card>

        <el-card class="mt-12">
          <template #header>待审批会话</template>
          <el-table :fit="true" :data="pendingSessions" size="small" max-height="260" empty-text="暂无待审批会话">
            <el-table-column prop="session_no" label="会话号" min-width="150" />
            <el-table-column prop="asset_name" label="资产" min-width="120" />
            <el-table-column prop="username" label="用户" width="100" />
            <el-table-column label="开始时间" min-width="150">
              <template #default="{ row }">{{ formatTime(row.started_at) }}</template>
            </el-table-column>
            <el-table-column label="等待时长" width="110">
              <template #default="{ row }">
                <el-tag :type="isPendingSessionStale(row) ? 'warning' : 'success'">
                  {{ formatPendingWait(row.started_at) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" min-width="180">
              <template #default="{ row }">
                <div class="inline-actions">
                  <el-button link type="primary" @click="approveJumpSession(row)">通过</el-button>
                  <el-button link type="warning" @click="rejectJumpSession(row)">拒绝</el-button>
                  <el-button link @click="openJumpSession(row)">详情</el-button>
                </div>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>

      <el-col :span="14">
        <el-card>
          <BatchActionBar
            title="高风险操作事件"
            :tags="riskHeaderTags"
            :actions="riskHeaderActions"
            @action="handleRiskHeaderAction"
          />
          <el-table
            ref="riskTableRef"
            :fit="true"
            :data="riskEvents"
            size="small"
            max-height="470"
            empty-text="暂无风险事件"
            @selection-change="onRiskSelectionChange"
          >
            <el-table-column type="selection" width="46" />
            <el-table-column prop="asset_name" label="资产" min-width="120" />
            <el-table-column prop="username" label="用户" width="100" />
            <el-table-column prop="severity" label="级别" width="90">
              <template #default="{ row }">
                <el-tag :type="severityTag(row.severity)">{{ String(row.severity || '').toUpperCase() || '-' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="event_type" label="类型" width="90" />
            <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
            <el-table-column label="时间" min-width="150">
              <template #default="{ row }">{{ formatTime(row.fired_at) }}</template>
            </el-table-column>
            <el-table-column label="操作" min-width="150" fixed="right">
              <template #default="{ row }">
                <div class="inline-actions">
                  <el-button v-if="row.session_id" link type="danger" @click="disconnectRiskSession(row)">断开会话</el-button>
                  <el-button link type="primary" @click="openJumpSessionByID(row.session_id)">详情</el-button>
                </div>
              </template>
            </el-table-column>
          </el-table>
          <QuickGroupTags :groups="riskEventGroups" default-type="warning" @select="batchDisconnectRiskGroup" />
        </el-card>
      </el-col>
    </el-row>

    <el-card class="integration-card">
      <template #header>
        <div class="integration-header">
          <span>资产作战融合视图</span>
          <div class="integration-actions">
            <el-input
              v-model="panelKeyword"
              clearable
              size="small"
              class="panel-search"
              placeholder="筛选会话号、资产、用户、IP、风险..."
            />
            <el-button size="small" type="primary" plain @click="openCurrentPanel">进入完整页面</el-button>
          </div>
        </div>
      </template>

      <el-tabs v-model="activePanel" class="integration-tabs">
        <el-tab-pane label="离线资产" name="offline">
          <el-table :fit="true" :data="filteredOfflineAssets" size="small" max-height="360" empty-text="资产健康良好">
            <el-table-column prop="type" label="类型" width="100" />
            <el-table-column prop="name" label="名称" min-width="140" />
            <el-table-column prop="address" label="地址" min-width="140" />
            <el-table-column label="状态" width="110">
              <template #default="{ row }">
                <el-tag :type="row.level === 'danger' ? 'danger' : 'warning'">{{ row.statusText }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="快速处置" min-width="210">
              <template #default="{ row }">
                <div class="inline-actions">
                  <el-button link type="primary" @click="offlineAction(row)">{{ row.actionLabel || '诊断' }}</el-button>
                  <el-button link @click="go(row.path)">管理页</el-button>
                </div>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="待审批会话" name="pending">
          <el-table :fit="true" :data="filteredPendingSessions" size="small" max-height="360" empty-text="暂无待审批会话">
            <el-table-column prop="session_no" label="会话号" min-width="150" />
            <el-table-column prop="asset_name" label="资产" min-width="140" />
            <el-table-column prop="username" label="用户" width="100" />
            <el-table-column prop="protocol" label="协议" width="90" />
            <el-table-column label="开始时间" min-width="150">
              <template #default="{ row }">{{ formatTime(row.started_at) }}</template>
            </el-table-column>
            <el-table-column label="等待时长" width="110">
              <template #default="{ row }">
                <el-tag :type="isPendingSessionStale(row) ? 'warning' : 'success'">
                  {{ formatPendingWait(row.started_at) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" min-width="210" fixed="right">
              <template #default="{ row }">
                <div class="inline-actions">
                  <el-button link type="primary" @click="approveJumpSession(row)">通过</el-button>
                  <el-button link type="warning" @click="rejectJumpSession(row)">拒绝</el-button>
                  <el-button link @click="openJumpSession(row)">详情</el-button>
                </div>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="活跃会话" name="active">
          <el-table :fit="true" :data="filteredActiveSessions" size="small" max-height="360" empty-text="暂无活跃会话">
            <el-table-column prop="session_no" label="会话号" min-width="150" />
            <el-table-column prop="asset_name" label="资产" min-width="140" />
            <el-table-column prop="username" label="用户" width="100" />
            <el-table-column prop="command_count" label="命令数" width="90" />
            <el-table-column label="最后命令" min-width="150">
              <template #default="{ row }">{{ formatTime(row.last_command_at) }}</template>
            </el-table-column>
            <el-table-column label="开始时间" min-width="150">
              <template #default="{ row }">{{ formatTime(row.started_at) }}</template>
            </el-table-column>
            <el-table-column label="操作" min-width="170" fixed="right">
              <template #default="{ row }">
                <div class="inline-actions">
                  <el-button link type="primary" @click="connectJumpSession(row)">连接</el-button>
                  <el-button link type="danger" @click="disconnectJumpSession(row)">断开</el-button>
                </div>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="风险事件" name="risk">
          <el-table :fit="true" :data="filteredRiskEvents" size="small" max-height="360" empty-text="暂无风险事件">
            <el-table-column prop="asset_name" label="资产" min-width="140" />
            <el-table-column prop="username" label="用户" width="100" />
            <el-table-column label="级别" width="90">
              <template #default="{ row }">
                <el-tag :type="severityTag(row.severity)">{{ String(row.severity || '').toUpperCase() || '-' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="rule_name" label="命中规则" min-width="160" />
            <el-table-column prop="description" label="描述" min-width="180" show-overflow-tooltip />
            <el-table-column label="时间" min-width="150">
              <template #default="{ row }">{{ formatTime(row.fired_at) }}</template>
            </el-table-column>
            <el-table-column label="操作" min-width="180" fixed="right">
              <template #default="{ row }">
                <div class="inline-actions">
                  <el-button v-if="row.session_id" link type="danger" @click="disconnectRiskSession(row)">断开会话</el-button>
                  <el-button link @click="openJumpSessionByID(row.session_id)">详情</el-button>
                </div>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="防火墙" name="firewall">
          <el-table :fit="true" :data="filteredFirewalls" size="small" max-height="360" empty-text="暂无防火墙数据">
            <el-table-column prop="name" label="名称" min-width="140" />
            <el-table-column prop="vendor" label="厂商" min-width="110" />
            <el-table-column prop="ip" label="IP" min-width="140" />
            <el-table-column label="状态" width="110">
              <template #default="{ row }">
                <el-tag :type="firewallStatus(row.status).type">{{ firewallStatus(row.status).text }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="最近检查" min-width="160">
              <template #default="{ row }">{{ formatTime(row.last_check_at) }}</template>
            </el-table-column>
            <el-table-column label="检查时效" width="110">
              <template #default="{ row }">
                <el-tag :type="isCheckStale(row.last_check_at) ? 'warning' : 'success'">
                  {{ isCheckStale(row.last_check_at) ? '待复检' : '及时' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" min-width="180" fixed="right">
              <template #default="{ row }">
                <div class="inline-actions">
                  <el-button link type="primary" @click="collectFirewall(row)">采集</el-button>
                  <el-button link @click="testFirewallSNMP(row)">SNMP测试</el-button>
                </div>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="堡垒机资产" name="jumpAssets">
          <el-table :fit="true" :data="filteredJumpAssets" size="small" max-height="360" empty-text="暂无堡垒机资产数据">
            <el-table-column prop="name" label="名称" min-width="140" />
            <el-table-column prop="asset_type" label="类型" width="110" />
            <el-table-column prop="protocol" label="协议" width="90" />
            <el-table-column label="地址" min-width="170">
              <template #default="{ row }">{{ row.address }}:{{ row.port }}</template>
            </el-table-column>
            <el-table-column prop="source" label="来源" min-width="120" />
            <el-table-column label="启用" width="90">
              <template #default="{ row }">
                <el-tag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? '是' : '否' }}</el-tag>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <BatchResultDrawer
      v-model="batchResultVisible"
      :title="batchResultTitle"
      :summary="batchResultSummary"
      :records="batchResultRecords"
    />
  </el-card>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getErrorMessage, isCancelError } from '@/utils/error'
import { requestApplyWorkspaceCategory } from '@/utils/workspace'
import BatchActionBar from '@/components/hub/BatchActionBar.vue'
import QuickGroupTags from '@/components/hub/QuickGroupTags.vue'
import BatchResultDrawer from '@/components/hub/BatchResultDrawer.vue'

const router = useRouter()
const loading = ref(false)
const hosts = ref([])
const networkDevices = ref([])
const firewalls = ref([])
const jumpSessions = ref([])
const jumpRiskEvents = ref([])
const jumpAssets = ref([])
const commandStats = ref({})
const syncingNetworkFromFirewall = ref(false)
const riskBatching = ref(false)
const activePanel = ref('offline')
const panelKeyword = ref('')
const nowTick = ref(Date.now())
const riskTableRef = ref(null)
const selectedRiskRows = ref([])
const batchResultVisible = ref(false)
const batchResultTitle = ref('')
const batchResultSummary = ref({ total: 0, success: 0, failed: 0 })
const batchResultRecords = ref([])
let freshnessTicker = null

const CHECK_STALE_HOURS = 24
const PENDING_SESSION_STALE_MINUTES = 30

const stats = reactive({
  hostTotal: 0,
  hostOffline: 0,
  networkTotal: 0,
  networkOffline: 0,
  firewallAlert: 0,
  jumpPending: 0,
  jumpActive: 0,
  riskCritical: 0,
  recheckPending: 0,
  pendingApprovalTimeout: 0,
  pendingBacklog: 0
})

const quickTabs = [
  { label: '资产总览', path: '/asset/overview' },
  { label: '主机管理', path: '/host' },
  { label: '网络设备', path: '/cmdb/network-devices' },
  { label: '防火墙管理', path: '/firewall' },
  { label: '堡垒机会话', path: '/jump/sessions' },
  { label: '风控规则', path: '/jump/command-rules' },
  { label: '会话审计', path: '/terminal' }
]

const panelRouteMap = {
  offline: '/host',
  pending: '/jump/sessions',
  active: '/jump/sessions',
  risk: '/jump/sessions',
  firewall: '/firewall',
  jumpAssets: '/jump/assets'
}

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })
const go = (path) => router.push(path)
const applyRecommendedWorkspace = () => requestApplyWorkspaceCategory('asset', 'hub-asset-ops')

const normalizeText = (value) => String(value ?? '').trim().toLowerCase()

const calcAgeHours = (timeValue) => {
  if (!timeValue) return Number.POSITIVE_INFINITY
  const target = new Date(timeValue)
  if (Number.isNaN(target.getTime())) return Number.POSITIVE_INFINITY
  const ageMs = nowTick.value - target.getTime()
  if (ageMs <= 0) return 0
  return ageMs / (60 * 60 * 1000)
}

const calcAgeMinutes = (timeValue) => calcAgeHours(timeValue) * 60

const isCheckStale = (checkedAt) => calcAgeHours(checkedAt) >= CHECK_STALE_HOURS
const isPendingSessionStale = (row) => calcAgeMinutes(row?.started_at) >= PENDING_SESSION_STALE_MINUTES

const formatPendingWait = (startedAt) => {
  const minutes = calcAgeMinutes(startedAt)
  if (!Number.isFinite(minutes)) return '未知'
  if (minutes < 1) return '<1m'
  if (minutes < 60) return `${Math.floor(minutes)}m`
  const hours = Math.floor(minutes / 60)
  const remain = Math.floor(minutes % 60)
  return remain > 0 ? `${hours}h${remain}m` : `${hours}h`
}

const formatCheckAge = (checkedAt) => {
  const hours = calcAgeHours(checkedAt)
  if (!Number.isFinite(hours)) return '未检查'
  if (hours < 1) return '1小时内'
  return `${Math.floor(hours)}h前`
}

const isOnlineStatus = (status) => {
  if (status === null || status === undefined) return false
  if (status === true || status === 1) return true
  const normalized = normalizeText(status)
  return normalized === 'online' || normalized === 'normal' || normalized === 'active' || normalized === '在线'
}

const firewallStatus = (status) => {
  const value = Number(status)
  if (value === 1) return { text: '正常', type: 'success' }
  if (value === 2) return { text: '告警', type: 'danger' }
  return { text: '离线', type: 'warning' }
}

const severityTag = (value) => {
  const v = normalizeText(value)
  if (v === 'critical') return 'danger'
  if (v === 'warning') return 'warning'
  return 'info'
}

const hostOnlineRate = computed(() => {
  if (!stats.hostTotal) return 0
  return Math.round(((stats.hostTotal - stats.hostOffline) / stats.hostTotal) * 100)
})

const networkOnlineRate = computed(() => {
  if (!stats.networkTotal) return 0
  return Math.round(((stats.networkTotal - stats.networkOffline) / stats.networkTotal) * 100)
})

const pendingSessions = computed(() =>
  jumpSessions.value
    .filter((item) => String(item.status) === 'pending_approval')
    .sort((a, b) => new Date(b.started_at || 0).getTime() - new Date(a.started_at || 0).getTime())
    .slice(0, 12)
)

const riskEvents = computed(() =>
  jumpRiskEvents.value
    .sort((a, b) => new Date(b.fired_at || 0).getTime() - new Date(a.fired_at || 0).getTime())
    .slice(0, 12)
)

const riskCriticalCount = computed(() =>
  riskEvents.value.filter((item) => normalizeText(item.severity) === 'critical').length
)

const actionableRiskCount = computed(() => riskEvents.value.filter((item) => Boolean(item.session_id)).length)

const selectedRiskCount = computed(() => selectedRiskRows.value.filter((item) => Boolean(item?.session_id)).length)

const riskHeaderTags = computed(() => [
  { label: `Critical ${riskCriticalCount.value}`, type: 'danger' },
  { label: `可断开 ${actionableRiskCount.value}`, type: 'warning' }
])

const riskHeaderActions = computed(() => [
  {
    key: 'disconnect-selected',
    label: '批量断开已选',
    type: 'warning',
    plain: true,
    loading: riskBatching.value,
    disabled: !selectedRiskCount.value
  },
  {
    key: 'disconnect-critical',
    label: '处置高危',
    type: undefined,
    plain: true,
    loading: riskBatching.value,
    disabled: !riskCriticalCount.value
  }
])

const riskEventGroups = computed(() => {
  const groups = new Map()
  riskEvents.value.forEach((item) => {
    if (!item.session_id) return
    const rule = item.rule_name || item.event_type || '未分类风险'
    const key = `${rule}`
    const current = groups.get(key) || { key, label: rule, count: 0, level: 'warning', rows: [] }
    current.count += 1
    if (normalizeText(item.severity) === 'critical') current.level = 'danger'
    current.rows.push(item)
    groups.set(key, current)
  })
  return [...groups.values()].sort((a, b) => b.count - a.count).slice(0, 8)
})

const staleNetworkDevices = computed(() =>
  networkDevices.value.filter((item) => isOnlineStatus(item.status) && isCheckStale(item.last_check_at))
)

const staleFirewalls = computed(() =>
  firewalls.value.filter((item) => Number(item.status) === 1 && isCheckStale(item.last_check_at))
)

const stalePendingSessions = computed(() => pendingSessions.value.filter((item) => isPendingSessionStale(item)))

const offlineAssets = computed(() => {
  const rows = []
  hosts.value.forEach((item) => {
    if (!isOnlineStatus(item.status)) {
      rows.push({
        type: '主机',
        id: item.id,
        name: item.name || '-',
        address: item.ip || '-',
        statusText: '离线',
        level: 'danger',
        actionLabel: '连通测试',
        path: '/host'
      })
    }
  })
  networkDevices.value.forEach((item) => {
    if (!isOnlineStatus(item.status)) {
      rows.push({
        type: '网络设备',
        id: item.id,
        name: item.name || '-',
        address: item.ip || item.address || '-',
        statusText: '离线',
        level: 'warning',
        actionLabel: '设备诊断',
        path: '/cmdb/network-devices'
      })
    }
  })
  firewalls.value.forEach((item) => {
    const status = Number(item.status)
    if (status !== 1) {
      rows.push({
        type: '防火墙',
        id: item.id,
        name: item.name || '-',
        address: item.ip || '-',
        statusText: status === 2 ? '告警' : '离线',
        level: status === 2 ? 'danger' : 'warning',
        actionLabel: 'SNMP采集',
        path: '/firewall'
      })
    }
  })
  staleNetworkDevices.value.forEach((item) => {
    rows.push({
      type: '网络设备',
      id: item.id,
      name: item.name || '-',
      address: item.ip || item.address || '-',
      statusText: `待复检（${formatCheckAge(item.last_check_at)}）`,
      level: 'warning',
      actionLabel: '设备诊断',
      path: '/cmdb/network-devices'
    })
  })
  staleFirewalls.value.forEach((item) => {
    rows.push({
      type: '防火墙',
      id: item.id,
      name: item.name || '-',
      address: item.ip || '-',
      statusText: `待复检（${formatCheckAge(item.last_check_at)}）`,
      level: 'warning',
      actionLabel: 'SNMP采集',
      path: '/firewall'
    })
  })
  stalePendingSessions.value.forEach((item) => {
    rows.push({
      type: '会话审批',
      id: item.id,
      name: item.session_no || item.id || '-',
      address: item.asset_name || '-',
      statusText: `超时待审批（${formatPendingWait(item.started_at)}）`,
      level: 'warning',
      actionLabel: '进入审批',
      path: '/jump/sessions'
    })
  })
  return rows
    .sort((a, b) => (a.level === 'danger' ? -1 : 1) - (b.level === 'danger' ? -1 : 1))
    .slice(0, 40)
})

const filterRows = (rows, fields) => {
  const keyword = normalizeText(panelKeyword.value)
  const base = Array.isArray(rows) ? rows : []
  if (!keyword) return base.slice(0, 30)
  return base.filter((row) => fields.some((field) => normalizeText(field(row)).includes(keyword))).slice(0, 30)
}

const filteredOfflineAssets = computed(() =>
  filterRows(offlineAssets.value, [(row) => row.type, (row) => row.name, (row) => row.address, (row) => row.statusText])
)

const filteredPendingSessions = computed(() =>
  filterRows(pendingSessions.value, [(row) => row.session_no, (row) => row.asset_name, (row) => row.username, (row) => row.protocol, (row) => row.source_ip])
)

const filteredActiveSessions = computed(() =>
  filterRows(
    jumpSessions.value.filter((item) => String(item.status) === 'active'),
    [(row) => row.session_no, (row) => row.asset_name, (row) => row.username, (row) => row.protocol, (row) => row.source_ip]
  )
)

const filteredRiskEvents = computed(() =>
  filterRows(jumpRiskEvents.value, [(row) => row.asset_name, (row) => row.username, (row) => row.severity, (row) => row.rule_name, (row) => row.description, (row) => row.command])
)

const filteredFirewalls = computed(() =>
  filterRows(firewalls.value, [(row) => row.name, (row) => row.vendor, (row) => row.ip, (row) => firewallStatus(row.status).text])
)

const filteredJumpAssets = computed(() =>
  filterRows(jumpAssets.value, [(row) => row.name, (row) => row.asset_type, (row) => row.protocol, (row) => row.address, (row) => row.source])
)

const openJumpSession = () => {
  router.push('/jump/sessions')
}

const openJumpSessionByID = (sessionID) => {
  if (!sessionID) {
    openJumpSession()
    return
  }
  router.push(`/jump/sessions?id=${encodeURIComponent(sessionID)}`)
}

const connectJumpSession = async (row) => {
  if (!row?.id) return
  try {
    const res = await axios.post(`/api/v1/jump/sessions/${row.id}/connect`, {}, { headers: authHeaders() })
    const openURL = res?.data?.data?.open_url
    if (openURL) {
      router.push(openURL)
      return
    }
    ElMessage.success('会话连接已就绪')
    await refreshAll()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '连接会话失败'))
  }
}

const approveJumpSession = async (row) => {
  if (!row?.id) return
  try {
    await axios.post(`/api/v1/jump/sessions/${row.id}/approve`, {}, { headers: authHeaders() })
    ElMessage.success('会话已通过审批')
    await refreshAll()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '审批失败'))
  }
}

const rejectJumpSession = async (row) => {
  if (!row?.id) return
  try {
    const { value } = await ElMessageBox.prompt('请输入拒绝原因', `拒绝会话 ${row.session_no || row.id}`, {
      inputType: 'textarea',
      inputPlaceholder: '例如：不在变更窗口'
    })
    await axios.post(`/api/v1/jump/sessions/${row.id}/reject`, { reason: value || '' }, { headers: authHeaders() })
    ElMessage.success('会话已拒绝')
    await refreshAll()
  } catch (err) {
    if (!isCancelError(err)) ElMessage.error(getErrorMessage(err, '拒绝失败'))
  }
}

const disconnectJumpSession = async (row) => {
  if (!row?.id) return
  try {
    await ElMessageBox.confirm(`确认断开会话 ${row.session_no || row.id} 吗？`, '提示', { type: 'warning' })
    await axios.post(`/api/v1/jump/sessions/${row.id}/disconnect`, { reason: '资产作战台手动断开' }, { headers: authHeaders() })
    ElMessage.success('会话已断开')
    await refreshAll()
  } catch (err) {
    if (!isCancelError(err)) ElMessage.error(getErrorMessage(err, '断开失败'))
  }
}

const onRiskSelectionChange = (rows) => {
  selectedRiskRows.value = Array.isArray(rows) ? rows : []
}

const handleRiskHeaderAction = async (key) => {
  if (key === 'disconnect-selected') await batchDisconnectSelectedRisk()
  if (key === 'disconnect-critical') await batchDisconnectCriticalRisk()
}

const showBatchResult = (title, targets, settled, successMessage) => {
  const records = targets.map((target, index) => {
    const result = settled[index]
    if (result?.status === 'fulfilled') {
      return { id: target.id, target: target.name, status: 'success', message: successMessage }
    }
    return {
      id: target.id,
      target: target.name,
      status: 'failed',
      message: getErrorMessage(result?.reason, '执行失败')
    }
  })
  const success = records.filter((item) => item.status === 'success').length
  const failed = records.length - success
  batchResultTitle.value = title
  batchResultSummary.value = { total: records.length, success, failed }
  batchResultRecords.value = records
  batchResultVisible.value = true
}

const disconnectRiskSessionByID = async (sessionID, reason) => {
  await axios.post(`/api/v1/jump/sessions/${sessionID}/disconnect`, { reason }, { headers: authHeaders() })
}

const disconnectRiskSession = async (row) => {
  const sessionID = row?.session_id
  if (!sessionID) {
    ElMessage.info('该风险事件没有关联在线会话')
    return
  }
  try {
    await ElMessageBox.confirm(`确认断开风险会话 ${sessionID} 吗？`, '提示', { type: 'warning' })
    await disconnectRiskSessionByID(sessionID, '风险事件触发手动断开')
    ElMessage.success('风险会话已断开')
    await refreshAll()
  } catch (err) {
    if (!isCancelError(err)) ElMessage.error(getErrorMessage(err, '断开风险会话失败'))
  }
}

const runBatchRiskDisconnect = async (rows, title, message) => {
  const sessionIDs = [...new Set(rows.map((item) => item?.session_id).filter(Boolean))]
  const targets = sessionIDs.map((id) => {
    const row = rows.find((item) => item?.session_id === id)
    return { id, name: row?.asset_name || row?.username || `会话-${id}` }
  })
  if (!sessionIDs.length) {
    ElMessage.info('没有可断开的会话')
    return
  }
  riskBatching.value = true
  try {
    await ElMessageBox.confirm(message, title, { type: 'warning' })
    const settled = await Promise.allSettled(
      sessionIDs.map((id) => disconnectRiskSessionByID(id, '资产作战台批量处置断开'))
    )
    const success = settled.filter((item) => item.status === 'fulfilled').length
    const fail = settled.length - success
    ElMessage.success(`批量处置完成：成功 ${success}，失败 ${fail}`)
    showBatchResult(title, targets, settled, '会话已断开')
    selectedRiskRows.value = []
    if (riskTableRef.value?.clearSelection) riskTableRef.value.clearSelection()
    await refreshAll()
  } catch (err) {
    if (!isCancelError(err)) ElMessage.error(getErrorMessage(err, '批量处置失败'))
  } finally {
    riskBatching.value = false
  }
}

const batchDisconnectSelectedRisk = async () => {
  await runBatchRiskDisconnect(
    selectedRiskRows.value,
    '批量断开风险会话',
    `确认断开已选择的 ${selectedRiskCount.value} 个风险会话吗？`
  )
}

const batchDisconnectCriticalRisk = async () => {
  const rows = riskEvents.value.filter((item) => normalizeText(item.severity) === 'critical' && item.session_id)
  await runBatchRiskDisconnect(rows, '处置高危风险', `确认断开 ${rows.length} 个 Critical 风险会话吗？`)
}

const batchDisconnectRiskGroup = async (group) => {
  const rows = Array.isArray(group?.rows) ? group.rows : []
  await runBatchRiskDisconnect(rows, `按规则处置：${group?.label || '-'}`, `确认断开「${group?.label || '-'}」共 ${rows.length} 个风险会话吗？`)
}

const testHostConnectivity = async (id) => {
  if (!id) return
  try {
    await axios.post(`/api/v1/cmdb/hosts/${id}/test`, {}, { headers: authHeaders() })
    ElMessage.success('主机连通测试完成')
    await refreshAll()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '主机连通测试失败'))
  }
}

const testNetworkDevice = async (id) => {
  if (!id) return
  try {
    const res = await axios.post(`/api/v1/cmdb/network-devices/${id}/test`, {}, { headers: authHeaders() })
    const status = Number(res?.data?.data?.status || 0)
    if (status === 1) ElMessage.success('网络设备诊断完成：在线')
    else if (status === 2) ElMessage.warning('网络设备诊断完成：部分可达')
    else ElMessage.warning('网络设备诊断完成：离线')
    await refreshAll()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '网络设备诊断失败'))
  }
}

const collectFirewall = async (row) => {
  if (!row?.id) return
  try {
    await axios.post(`/api/v1/firewall/devices/${row.id}/snmp/collect`, {}, { headers: authHeaders() })
    ElMessage.success('防火墙采集完成')
    await refreshAll()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '防火墙采集失败'))
  }
}

const testFirewallSNMP = async (row) => {
  if (!row?.id) return
  try {
    await axios.post(`/api/v1/firewall/devices/${row.id}/snmp/test`, {}, { headers: authHeaders() })
    ElMessage.success('SNMP 测试通过')
  } catch (err) {
    ElMessage.error(getErrorMessage(err, 'SNMP 测试失败'))
  }
}

const offlineAction = (row) => {
  if (!row) return
  if (row.type === '主机') {
    testHostConnectivity(row.id)
    return
  }
  if (row.type === '网络设备') {
    testNetworkDevice(row.id)
    return
  }
  if (row.type === '防火墙') {
    collectFirewall(row)
    return
  }
  if (row.type === '会话审批') {
    openJumpSession()
    return
  }
  go(row.path)
}

const syncNetworkDevicesFromFirewalls = async () => {
  syncingNetworkFromFirewall.value = true
  try {
    const res = await axios.post('/api/v1/cmdb/network-devices/sync/firewalls', {}, { headers: authHeaders() })
    const data = res?.data?.data || {}
    ElMessage.success(`同步完成：新增 ${data.created || 0}，更新 ${data.updated || 0}`)
    await refreshAll()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '同步失败'))
  } finally {
    syncingNetworkFromFirewall.value = false
  }
}

const openCurrentPanel = () => {
  go(panelRouteMap[activePanel.value] || '/host')
}

const formatTime = (value) => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString()
}

const safeArray = (res) => (Array.isArray(res?.data?.data) ? res.data.data : [])
const safeObject = (res) => (res?.data?.data && typeof res.data.data === 'object' ? res.data.data : {})

const refreshAll = async () => {
  loading.value = true
  try {
    const [hostRes, networkRes, firewallRes, sessionRes, riskRes, jumpAssetRes, commandStatsRes] = await Promise.allSettled([
      axios.get('/api/v1/cmdb/hosts', { headers: authHeaders() }),
      axios.get('/api/v1/cmdb/network-devices', { headers: authHeaders() }),
      axios.get('/api/v1/firewall/devices', { headers: authHeaders() }),
      axios.get('/api/v1/jump/sessions', { headers: authHeaders() }),
      axios.get('/api/v1/jump/risk-events', { headers: authHeaders() }),
      axios.get('/api/v1/jump/assets', { headers: authHeaders() }),
      axios.get('/api/v1/jump/command-rules/stats', { headers: authHeaders(), params: { days: 7 } })
    ])

    hosts.value = hostRes.status === 'fulfilled' ? safeArray(hostRes.value) : []
    networkDevices.value = networkRes.status === 'fulfilled' ? safeArray(networkRes.value) : []
    firewalls.value = firewallRes.status === 'fulfilled' ? safeArray(firewallRes.value) : []
    jumpSessions.value = sessionRes.status === 'fulfilled' ? safeArray(sessionRes.value) : []
    jumpRiskEvents.value = riskRes.status === 'fulfilled' ? safeArray(riskRes.value) : []
    jumpAssets.value = jumpAssetRes.status === 'fulfilled' ? safeArray(jumpAssetRes.value) : []
    commandStats.value = commandStatsRes.status === 'fulfilled' ? safeObject(commandStatsRes.value) : {}

    stats.hostTotal = hosts.value.length
    stats.hostOffline = hosts.value.filter((item) => !isOnlineStatus(item.status)).length
    stats.networkTotal = networkDevices.value.length
    stats.networkOffline = networkDevices.value.filter((item) => !isOnlineStatus(item.status)).length
    stats.firewallAlert = firewalls.value.filter((item) => Number(item.status) === 2).length
    stats.jumpPending = jumpSessions.value.filter((item) => String(item.status) === 'pending_approval').length
    stats.jumpActive = jumpSessions.value.filter((item) => String(item.status) === 'active').length
    stats.riskCritical = jumpRiskEvents.value.filter((item) => normalizeText(item.severity) === 'critical').length
    stats.recheckPending = staleNetworkDevices.value.length + staleFirewalls.value.length
    stats.pendingApprovalTimeout = stalePendingSessions.value.length
    stats.pendingBacklog = stats.recheckPending + stats.pendingApprovalTimeout + stats.riskCritical

    const failedCount = [hostRes, networkRes, firewallRes, sessionRes, riskRes, jumpAssetRes, commandStatsRes].filter((r) => r.status === 'rejected').length
    if (failedCount > 0) {
      ElMessage.warning(`部分作战台数据加载失败(${failedCount}项)，已展示可用数据`)
    }
  } catch (err) {
    ElMessage.error(err?.response?.data?.message || err?.message || '加载资产作战台失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  refreshAll()
  freshnessTicker = window.setInterval(() => {
    nowTick.value = Date.now()
  }, 60 * 1000)
})

onBeforeUnmount(() => {
  if (freshnessTicker) {
    window.clearInterval(freshnessTicker)
    freshnessTicker = null
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
.metric-sub { margin-top: 6px; font-size: 12px; color: var(--muted-text); }
.mt-12 { margin-top: 12px; }
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

  .panel-search {
    width: 100%;
  }

}
</style>
