<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>监控告警中心</h2>
        <p class="page-desc">按“发现-通知-处置”链路汇总可观测数据，减少告警与域名证书排障中的页面切换。</p>
      </div>
      <div class="page-actions">
        <el-button :loading="loading" icon="Refresh" @click="refreshAll">刷新</el-button>
      </div>
    </div>

    <div class="hub-tabs-wrap">
      <el-tabs :model-value="activeWorkbenchTab" @tab-click="handleWorkbenchTabClick">
        <el-tab-pane
          v-for="tab in workbenchTabs"
          :key="tab.path"
          :name="tab.path"
          :label="tab.label"
        />
      </el-tabs>
    </div>

    <el-row :gutter="12" class="summary-row">
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">告警总量</div><div class="metric-value">{{ stats.alertTotal }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">未处理告警</div><div class="metric-value danger">{{ stats.alertOpen }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">Critical</div><div class="metric-value danger">{{ stats.alertCritical }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">告警规则</div><div class="metric-value">{{ stats.ruleTotal }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">在线 Agent</div><div class="metric-value ok">{{ stats.agentOnline }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">通知渠道</div><div class="metric-value">{{ stats.notifyChannelTotal }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">域名风险</div><div class="metric-value warning">{{ stats.domainRisk }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card>
          <div class="metric-title">待处置积压</div>
          <div class="metric-value warning">{{ pendingBacklog }}</div>
          <div class="metric-sub">证书≤30天 {{ stats.certExpiringSoon }}</div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="12">
      <el-col :span="10">
        <el-card>
          <template #header>运行态总览</template>
          <div class="health-row"><span>告警关闭率</span><strong>{{ alertClosedRate }}%</strong></div>
          <el-progress :percentage="alertClosedRate" :stroke-width="14" />
          <div class="health-row mtop"><span>Agent 在线率</span><strong>{{ agentOnlineRate }}%</strong></div>
          <el-progress :percentage="agentOnlineRate" :stroke-width="14" status="success" />
          <el-divider />
          <div class="health-row"><span>通知组</span><strong>{{ stats.notifyGroupTotal }}</strong></div>
          <div class="health-row"><span>模板数量</span><strong>{{ stats.templateTotal }}</strong></div>
          <div class="health-row"><span>健康域名</span><strong>{{ stats.domainHealthy }}</strong></div>
          <div class="health-row"><span>未恢复 Critical</span><strong>{{ stats.alertCriticalOpen }}</strong></div>
          <div class="health-row"><span>超时未恢复告警</span><strong>{{ pendingAlertTimeout }}</strong></div>
          <div class="health-row"><span>待复检风险对象</span><strong>{{ riskCheckStale }}</strong></div>
        </el-card>

        <el-card class="mt-12">
          <template #header>Top 告警目标</template>
          <el-table :fit="true" :data="topTargets" size="small" max-height="220" empty-text="暂无告警数据">
            <el-table-column prop="target" label="目标" min-width="160" />
            <el-table-column prop="count" label="次数" width="90" />
            <el-table-column prop="critical" label="Critical" width="90" />
          </el-table>
        </el-card>
      </el-col>

      <el-col :span="14">
        <el-card>
          <template #header>
            <div class="risk-header">
              <span>域名/证书风险清单</span>
              <div class="risk-actions">
                <el-tag type="danger" effect="light">高危 {{ riskCriticalRows.length }}</el-tag>
                <el-tag type="warning" effect="light">待复检 {{ riskStaleRows.length }}</el-tag>
                <el-button size="small" type="warning" plain :loading="riskBatching" :disabled="!riskCriticalRows.length" @click="batchRecheckRisk('critical')">
                  批量复检高危
                </el-button>
                <el-button size="small" plain :loading="riskBatching" :disabled="!riskStaleRows.length" @click="batchRecheckRisk('stale')">
                  批量复检待复检
                </el-button>
              </div>
            </div>
          </template>
          <el-table :fit="true" :data="riskRows" size="small" max-height="420" empty-text="暂无风险项">
            <el-table-column prop="type" label="类型" width="110" />
            <el-table-column prop="name" label="域名" min-width="180" />
            <el-table-column label="风险" min-width="140">
              <template #default="{ row }">
                <el-tag :type="row.level === 'critical' ? 'danger' : 'warning'">{{ row.reason }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="detail" label="详情" min-width="180" />
            <el-table-column label="操作" width="100">
              <template #default="{ row }">
                <el-button link type="primary" @click="go(row.path)">处理</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="12" class="mt-12">
      <el-col :span="16">
        <el-card class="capability-card">
          <template #header>
            <div class="capability-header">
              <span>功能完整度矩阵</span>
              <div class="capability-tags">
                <el-tag type="success" effect="light">完整 {{ capabilitySummary.complete }}</el-tag>
                <el-tag type="warning" effect="light">待补齐 {{ capabilitySummary.partial }}</el-tag>
                <el-tag type="danger" effect="light">缺口 {{ capabilitySummary.gap }}</el-tag>
              </div>
            </div>
          </template>
          <el-table :fit="true" :data="moduleCapabilityRows" size="small" max-height="320" empty-text="暂无模块能力数据">
            <el-table-column prop="label" label="能力模块" min-width="140" />
            <el-table-column label="链路" width="90">
              <template #default="{ row }">
                <el-tag size="small" :type="row.linkStatus === 'ok' ? 'success' : row.linkStatus === 'warning' ? 'warning' : 'danger'">
                  {{ row.linkStatus === 'ok' ? '正常' : row.linkStatus === 'warning' ? '降级' : '异常' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="状态判定" width="90">
              <template #default="{ row }">
                <el-tag size="small" :type="row.status === 'ok' ? 'success' : row.status === 'warning' ? 'warning' : 'danger'">
                  {{ row.status === 'ok' ? '正常' : row.status === 'warning' ? '预警' : '异常' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="自动处置" width="90">
              <template #default="{ row }">
                <el-tag size="small" :type="row.automationScore >= 90 ? 'success' : row.automationScore >= 70 ? 'warning' : 'info'">
                  {{ row.automationScore >= 90 ? '完善' : row.automationScore >= 70 ? '可用' : '较弱' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="目标值" width="78">
              <template #default="{ row }">{{ row.targetScore }}%</template>
            </el-table-column>
            <el-table-column label="当前值" width="110">
              <template #default="{ row }">
                <el-progress :percentage="row.currentScore" :stroke-width="10" :show-text="false" />
                <div class="capability-mini-percent">{{ row.currentScore }}%</div>
              </template>
            </el-table-column>
            <el-table-column label="趋势" width="94">
              <template #default="{ row }">
                <el-tag size="small" :type="capabilityTrendTag(row.trendDirection)">
                  {{ capabilityTrendArrow(row.trendDirection) }} {{ formatTrendDelta(row.trendDelta) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="suggestion" label="建议" min-width="170" show-overflow-tooltip />
            <el-table-column label="操作" width="90" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" @click="go(row.path)">进入</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card class="capability-gap-card">
          <template #header>
            <div class="capability-header">
              <span>能力缺口追踪</span>
              <el-tag :type="capabilityGapRows.length ? 'warning' : 'success'" effect="light">缺口 {{ capabilityGapRows.length }}</el-tag>
            </div>
          </template>
          <el-table :fit="true" :data="capabilityGapRows" size="small" max-height="320" empty-text="暂无高优先级缺口">
            <el-table-column prop="module" label="模块" width="110" />
            <el-table-column prop="gap" label="缺口" min-width="150" show-overflow-tooltip />
            <el-table-column prop="impact" label="影响" min-width="170" show-overflow-tooltip />
            <el-table-column label="操作" width="90">
              <template #default="{ row }">
                <el-button link type="primary" @click="go(row.path)">修复</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <el-card class="integration-card">
      <template #header>
        <div class="integration-header">
          <div class="integration-title-wrap">
            <span>监控融合视图</span>
            <el-tag size="small" type="info" effect="plain">
              当前：{{ activePanelMeta.label }} · {{ activePanelMeta.count }}
            </el-tag>
          </div>
          <div class="integration-actions">
            <el-input
              v-model="panelKeyword"
              clearable
              size="small"
              class="panel-search"
              placeholder="筛选名称、目标、级别、渠道..."
            />
            <el-button v-if="panelKeyword" size="small" @click="panelKeyword = ''">清空筛选</el-button>
            <el-button size="small" type="primary" plain @click="openCurrentPanel">进入完整页面</el-button>
          </div>
        </div>
      </template>

      <el-tabs v-model="activePanel" class="integration-tabs">
        <el-tab-pane label="告警事件" name="alerts">
          <el-table :fit="true" :data="filteredAlerts" size="small" max-height="360" empty-text="暂无告警事件">
            <el-table-column label="名称" min-width="180">
              <template #default="{ row }">{{ row.alert_name || row.rule_name || row.name || '-' }}</template>
            </el-table-column>
            <el-table-column prop="target" label="目标" min-width="150" />
            <el-table-column label="级别" width="100">
              <template #default="{ row }">
                <el-tag :type="severityTag(row.severity)">{{ severityText(row.severity) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="状态" width="110">
              <template #default="{ row }">
                <StatusBadge v-bind="alertStatusBadge(row)" />
              </template>
            </el-table-column>
            <el-table-column label="时间" min-width="170">
              <template #default="{ row }">{{ formatTime(row.fired_at || row.created_at) }}</template>
            </el-table-column>
            <el-table-column label="待响应" width="110">
              <template #default="{ row }">
                <el-tag :type="isAlertStale(row) ? 'warning' : 'success'">
                  {{ formatWaitDuration(row.fired_at || row.created_at) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" min-width="220">
              <template #default="{ row }">
                <el-space wrap>
                  <el-button
                    v-if="Number(row.status) === 0"
                    size="small"
                    type="primary"
                    link
                    @click="ackAlert(row)"
                  >
                    确认
                  </el-button>
                  <el-button
                    v-if="Number(row.status) === 0 || Number(row.status) === 1"
                    size="small"
                    type="success"
                    link
                    @click="resolveAlert(row)"
                  >
                    恢复
                  </el-button>
                  <el-button size="small" type="warning" link @click="silenceAlert(row)">静默1h</el-button>
                  <el-button size="small" link @click="go('/alert/events')">详情</el-button>
                </el-space>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="告警规则" name="rules">
          <el-table :fit="true" :data="filteredRules" size="small" max-height="360" empty-text="暂无告警规则">
            <el-table-column prop="name" label="名称" min-width="150" />
            <el-table-column prop="type" label="类型" width="110" />
            <el-table-column prop="target" label="目标" min-width="150" />
            <el-table-column prop="metric" label="指标" min-width="120" />
            <el-table-column label="阈值" min-width="120">
              <template #default="{ row }">{{ row.operator || '' }} {{ row.threshold || '-' }}</template>
            </el-table-column>
            <el-table-column label="启用" width="90">
              <template #default="{ row }">
                <el-tag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? '是' : '否' }}</el-tag>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="通知渠道" name="channels">
          <el-table :fit="true" :data="filteredChannels" size="small" max-height="360" empty-text="暂无通知渠道">
            <el-table-column prop="name" label="名称" min-width="150" />
            <el-table-column prop="type" label="类型" width="120" />
            <el-table-column prop="description" label="描述" min-width="220" show-overflow-tooltip />
            <el-table-column label="启用" width="90">
              <template #default="{ row }">
                <el-tag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? '是' : '否' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="120">
              <template #default="{ row }">
                <el-button size="small" type="primary" link @click="testChannel(row)">测试</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="通知组与模板" name="notify">
          <el-row :gutter="12">
            <el-col :span="12">
              <el-table :fit="true" :data="filteredGroups" size="small" max-height="340" empty-text="暂无通知组">
                <el-table-column prop="name" label="通知组" min-width="140" />
                <el-table-column label="渠道数" width="90">
                  <template #default="{ row }">{{ parseChannels(row.channels).length }}</template>
                </el-table-column>
                <el-table-column prop="description" label="描述" min-width="150" show-overflow-tooltip />
              </el-table>
            </el-col>
            <el-col :span="12">
              <el-table :fit="true" :data="filteredTemplates" size="small" max-height="340" empty-text="暂无模板">
                <el-table-column prop="name" label="模板" min-width="140" />
                <el-table-column prop="type" label="类型" width="90" />
                <el-table-column prop="channel_type" label="渠道" width="100" />
                <el-table-column label="启用" width="80">
                  <template #default="{ row }">
                    <el-tag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? '是' : '否' }}</el-tag>
                  </template>
                </el-table-column>
              </el-table>
            </el-col>
          </el-row>
        </el-tab-pane>

        <el-tab-pane label="域名证书风险" name="domain">
          <el-table :fit="true" :data="filteredDomainRisk" size="small" max-height="360" empty-text="暂无域名证书风险">
            <el-table-column prop="type" label="类型" width="100" />
            <el-table-column prop="name" label="对象" min-width="180" />
            <el-table-column label="风险" width="120">
              <template #default="{ row }">
                <el-tag :type="row.level === 'critical' ? 'danger' : 'warning'">{{ row.reason }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="detail" label="详情" min-width="180" />
            <el-table-column label="检查时间" min-width="170">
              <template #default="{ row }">{{ formatTime(row.checked_at) }}</template>
            </el-table-column>
            <el-table-column label="检查时效" width="110">
              <template #default="{ row }">
                <el-tag :type="isRiskCheckStale(row.checked_at) ? 'warning' : 'success'">
                  {{ isRiskCheckStale(row.checked_at) ? '待复检' : '及时' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="140">
              <template #default="{ row }">
                <el-button size="small" type="primary" link @click="checkRisk(row)">复检</el-button>
                <el-button size="small" link @click="go('/domain/ssl')">处理</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </el-card>
</template>

<script setup>
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getErrorMessage, isCancelError } from '@/utils/error'
import { monitorAlertStatusMeta } from '@/utils/status'
import StatusBadge from '@/components/common/StatusBadge.vue'

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const alerts = ref([])
const rules = ref([])
const agents = ref([])
const domains = ref([])
const certs = ref([])
const channels = ref([])
const groups = ref([])
const templates = ref([])
const activePanel = ref('alerts')
const panelKeyword = ref('')
const nowTs = ref(Date.now())
const riskBatching = ref(false)
let minuteTicker = null
let autoRefreshTick = 0

const stats = reactive({
  alertTotal: 0,
  alertOpen: 0,
  alertClosed: 0,
  alertCritical: 0,
  alertCriticalOpen: 0,
  ruleTotal: 0,
  agentOnline: 0,
  agentTotal: 0,
  domainHealthy: 0,
  domainRisk: 0,
  certExpiringSoon: 0,
  notifyChannelTotal: 0,
  notifyGroupTotal: 0,
  templateTotal: 0
})
const dataSourceStatus = reactive({
  alerts: 'unknown',
  rules: 'unknown',
  agents: 'unknown',
  domains: 'unknown',
  certs: 'unknown',
  channels: 'unknown',
  groups: 'unknown',
  templates: 'unknown'
})

const panelRouteMap = {
  alerts: '/alert/events',
  rules: '/alert/rules',
  channels: '/notify/channels',
  notify: '/notify/groups',
  domain: '/domain/ssl'
}

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })
const go = (path) => router.push(path)
const workbenchTabs = [
  { label: '监控告警中心', path: '/monitor/center' },
  { label: '域名监控中心', path: '/domain/center' },
  { label: '告警事件', path: '/alert/events' },
  { label: '告警规则', path: '/alert/rules' },
  { label: '告警静默', path: '/alert/silences' },
  { label: '告警聚合', path: '/alert/aggregation' },
  { label: '告警复盘', path: '/alert/history' },
  { label: '通知渠道', path: '/notify/channels' },
  { label: '通知组', path: '/notify/groups' },
  { label: '通知模板', path: '/notify/templates' },
  { label: '域名与证书', path: '/domain/ssl' },
  { label: '监控概览', path: '/monitor/overview' },
  { label: '主机监控', path: '/monitor/hosts' },
  { label: '指标采集', path: '/monitor/metrics' },
  { label: '容器监控', path: '/monitor/containers' },
  { label: 'Pod监控', path: '/monitor/pods' },
  { label: 'Agent心跳', path: '/monitor/agents' },
  { label: '服务拓扑', path: '/topology' },
  { label: '成本概览', path: '/cost/overview' },
  { label: '预算告警', path: '/cost/budget' }
]
const activeWorkbenchTab = ref('/monitor/center')
const handleWorkbenchTabClick = (pane) => {
  const path = String(pane?.paneName || '').trim()
  if (!path || path === route.path) return
  go(path)
}

const normalizeText = (value) => String(value ?? '').trim().toLowerCase()
const parseChannels = (text) => {
  if (!text) return []
  try {
    return JSON.parse(text)
  } catch {
    return []
  }
}

const filterRows = (rows, fields) => {
  const keyword = normalizeText(panelKeyword.value)
  const base = Array.isArray(rows) ? rows : []
  if (!keyword) return base.slice(0, 20)
  return base
    .filter((row) => fields.some((field) => normalizeText(field(row)).includes(keyword)))
    .slice(0, 20)
}

const severityText = (value) => {
  const v = normalizeText(value)
  if (v === 'critical') return 'Critical'
  if (v === 'warning') return 'Warning'
  if (v === 'info') return 'Info'
  return '-'
}

const severityTag = (value) => {
  const v = normalizeText(value)
  if (v === 'critical') return 'danger'
  if (v === 'warning') return 'warning'
  return 'info'
}

const alertStatusText = (status) => {
  return monitorAlertStatusMeta(status).text
}

const alertStatusBadge = (row) => {
  const meta = monitorAlertStatusMeta(row?.status)
  const parts = []
  if (row?.severity) parts.push(`级别: ${severityText(row.severity)}`)
  if (row?.target) parts.push(`目标: ${row.target}`)
  if (row?.message) parts.push(row.message)
  return {
    text: meta.text,
    type: meta.type,
    reason: parts.join(' | '),
    updatedAt: row?.updated_at || row?.fired_at || row?.created_at
  }
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

const formatWaitDuration = (value) => {
  const minutes = elapsedMinutes(value)
  if (minutes < 60) return `${minutes}m`
  const hours = Math.floor(minutes / 60)
  const remain = minutes % 60
  if (hours < 24) return `${hours}h${remain}m`
  const days = Math.floor(hours / 24)
  return `${days}d${hours % 24}h`
}

const isAlertStale = (row) => {
  const status = Number(row?.status)
  if (status !== 0 && status !== 1) return false
  return elapsedMinutes(row?.fired_at || row?.created_at) >= 60
}

const isRiskCheckStale = (checkedAt) => {
  const ts = parseTimestamp(checkedAt)
  if (!ts) return true
  return (nowTs.value - ts) > 24 * 60 * 60 * 1000
}

const alertClosedRate = computed(() => {
  if (!stats.alertTotal) return 0
  return Math.round((stats.alertClosed / stats.alertTotal) * 100)
})

const agentOnlineRate = computed(() => {
  if (!stats.agentTotal) return 0
  return Math.round((stats.agentOnline / stats.agentTotal) * 100)
})

const topTargets = computed(() => {
  const map = new Map()
  alerts.value.forEach((item) => {
    const key = item.target || item.rule_name || item.alert_name || item.name || 'unknown'
    const existing = map.get(key) || { target: key, count: 0, critical: 0 }
    existing.count += 1
    if (normalizeText(item.severity) === 'critical') existing.critical += 1
    map.set(key, existing)
  })
  return [...map.values()].sort((a, b) => b.count - a.count).slice(0, 8)
})

const riskRows = computed(() => {
  const rows = []
  domains.value.forEach((item) => {
    const health = normalizeText(item.health_status)
    if (health === 'warning' || health === 'critical') {
      rows.push({
        kind: 'domain',
        id: item.id,
        domain: item.domain,
        type: '域名',
        name: item.domain,
        level: health,
        reason: health === 'critical' ? '健康故障' : '健康预警',
        detail: `HTTP ${item.http_status_code || '-'} / ${item.response_time_ms || '-'}ms`,
        path: '/domain/ssl',
        checked_at: item.last_checked_at || item.updated_at
      })
    }
  })
  certs.value.forEach((item) => {
    const days = Number(item.days_to_expire || 0)
    if (days <= 30) {
      rows.push({
        kind: 'cert',
        id: item.id,
        domain: item.domain,
        type: '证书',
        name: item.domain,
        level: days <= 7 ? 'critical' : 'warning',
        reason: days <= 7 ? '即将过期' : '临近到期',
        detail: `剩余 ${days} 天`,
        path: '/domain/ssl',
        checked_at: item.last_check_at || item.updated_at
      })
    }
  })
  return rows
    .sort((a, b) => Number(b.level === 'critical') - Number(a.level === 'critical'))
    .slice(0, 30)
})
const riskCriticalRows = computed(() => riskRows.value.filter((item) => item.level === 'critical'))
const riskStaleRows = computed(() => riskRows.value.filter((item) => isRiskCheckStale(item.checked_at)))

const pendingAlertTimeout = computed(() => alerts.value.filter((item) => isAlertStale(item)).length)
const riskCheckStale = computed(() => riskRows.value.filter((item) => isRiskCheckStale(item.checked_at)).length)
const pendingBacklog = computed(() => pendingAlertTimeout.value + stats.alertCriticalOpen + riskCheckStale.value)

const capabilityStatusSeverity = { ok: 1, warning: 2, error: 3, unknown: 0 }
const clampPercent = (value) => Math.max(0, Math.min(100, Math.round(Number(value) || 0)))
const capabilityScoreByStatus = (value) => {
  const status = normalizeText(value)
  if (status === 'ok') return 100
  if (status === 'warning') return 72
  if (status === 'error') return 38
  return 50
}
const capabilityTrendByTarget = (currentScore, targetScore) => {
  const delta = Math.round((Number(currentScore) || 0) - (Number(targetScore) || 0))
  if (delta >= 3) return { direction: 'up', delta }
  if (delta <= -3) return { direction: 'down', delta }
  return { direction: 'flat', delta }
}
const capabilityTrendArrow = (direction) => {
  if (direction === 'up') return '↑'
  if (direction === 'down') return '↓'
  return '→'
}
const capabilityTrendTag = (direction) => {
  if (direction === 'up') return 'success'
  if (direction === 'down') return 'danger'
  return 'info'
}
const formatTrendDelta = (value) => {
  const delta = Math.round(Number(value) || 0)
  if (delta > 0) return `+${delta}%`
  return `${delta}%`
}
const worstStatus = (...values) =>
  values.reduce((worst, current) => (capabilityStatusSeverity[normalizeText(current)] > capabilityStatusSeverity[normalizeText(worst)] ? current : worst), 'ok')

const moduleCapabilityRows = computed(() => {
  const alertLink = worstStatus(dataSourceStatus.alerts, dataSourceStatus.rules)
  const alertStatus = stats.alertCriticalOpen > 0 ? 'error' : (pendingAlertTimeout.value > 0 ? 'warning' : 'ok')
  const alertAutomation = (stats.alertOpen > 0 || pendingAlertTimeout.value > 0) ? 95 : 82

  const agentLink = dataSourceStatus.agents
  const agentStatus = stats.agentTotal === 0 ? 'warning' : (agentOnlineRate.value >= 90 ? 'ok' : agentOnlineRate.value >= 70 ? 'warning' : 'error')
  const agentAutomation = stats.agentTotal > 0 ? 80 : 65

  const notifyLink = worstStatus(dataSourceStatus.channels, dataSourceStatus.groups, dataSourceStatus.templates)
  const notifyStatus = (stats.notifyChannelTotal > 0 && stats.notifyGroupTotal > 0 && stats.templateTotal > 0) ? 'ok' : 'warning'
  const notifyAutomation = stats.notifyChannelTotal > 0 ? 84 : 62

  const domainLink = worstStatus(dataSourceStatus.domains, dataSourceStatus.certs)
  const domainStatus = riskCriticalRows.value.length > 0 ? 'error' : (riskRows.value.length > 0 || riskCheckStale.value > 0 ? 'warning' : 'ok')
  const domainAutomation = riskRows.value.length > 0 ? 90 : 78

  const rows = [
    {
      key: 'alerts',
      label: '告警事件治理',
      path: '/alert/events',
      linkStatus: alertLink,
      status: alertStatus,
      automationScore: alertAutomation,
      targetScore: 92,
      freshnessScore: clampPercent(100 - pendingAlertTimeout.value * 6),
      suggestion: alertStatus === 'ok' ? '保持规则与静默策略联动' : '优先清理超时未恢复与Critical告警'
    },
    {
      key: 'agents',
      label: 'Agent心跳',
      path: '/monitor/agents',
      linkStatus: agentLink,
      status: agentStatus,
      automationScore: agentAutomation,
      targetScore: 90,
      freshnessScore: agentOnlineRate.value,
      suggestion: agentStatus === 'ok' ? '保持心跳稳定和节点覆盖' : '补齐离线Agent与心跳阈值告警'
    },
    {
      key: 'notify',
      label: '通知链路',
      path: '/notify/channels',
      linkStatus: notifyLink,
      status: notifyStatus,
      automationScore: notifyAutomation,
      targetScore: 88,
      freshnessScore: stats.notifyChannelTotal > 0 ? 100 : 60,
      suggestion: notifyStatus === 'ok' ? '持续验证通知模板可用性' : '补齐渠道/通知组/模板，避免告警丢发'
    },
    {
      key: 'domain',
      label: '域名证书风险',
      path: '/domain/ssl',
      linkStatus: domainLink,
      status: domainStatus,
      automationScore: domainAutomation,
      targetScore: 90,
      freshnessScore: clampPercent(100 - riskCheckStale.value * 10),
      suggestion: domainStatus === 'ok' ? '保持周期复检策略' : '优先复检高危域名与临期证书'
    }
  ]

  return rows.map((item) => {
    const score = clampPercent(
      capabilityScoreByStatus(item.status) * 0.38 +
      capabilityScoreByStatus(item.linkStatus) * 0.32 +
      item.automationScore * 0.2 +
      item.freshnessScore * 0.1
    )
    const targetScore = clampPercent(item.targetScore || 88)
    const trend = capabilityTrendByTarget(score, targetScore)
    const level = score >= 85 ? 'complete' : score >= 65 ? 'partial' : 'gap'
    return {
      ...item,
      targetScore,
      currentScore: score,
      trendDirection: trend.direction,
      trendDelta: trend.delta,
      score,
      level
    }
  }).sort((a, b) => a.score - b.score)
})

const capabilitySummary = computed(() => ({
  complete: moduleCapabilityRows.value.filter((item) => item.level === 'complete').length,
  partial: moduleCapabilityRows.value.filter((item) => item.level === 'partial').length,
  gap: moduleCapabilityRows.value.filter((item) => item.level === 'gap').length
}))

const capabilityGapRows = computed(() =>
  moduleCapabilityRows.value
    .filter((item) => item.level !== 'complete')
    .map((item) => ({
      module: item.label,
      gap: item.level === 'gap' ? '核心链路与告警判定能力不足' : '链路存在降级，自动处置需增强',
      impact: item.level === 'gap' ? '可能造成漏报/误报并延迟处置' : '故障恢复效率下降',
      path: item.path,
      score: item.score
    }))
    .sort((a, b) => a.score - b.score)
    .slice(0, 8)
)

const filteredAlerts = computed(() =>
  filterRows(alerts.value, [(row) => row.alert_name, (row) => row.rule_name, (row) => row.target, (row) => row.severity])
)

const filteredRules = computed(() =>
  filterRows(rules.value, [(row) => row.name, (row) => row.type, (row) => row.target, (row) => row.metric, (row) => row.severity])
)

const filteredChannels = computed(() =>
  filterRows(channels.value, [(row) => row.name, (row) => row.type, (row) => row.description])
)

const filteredGroups = computed(() =>
  filterRows(groups.value, [(row) => row.name, (row) => row.description, (row) => parseChannels(row.channels).join(',')])
)

const filteredTemplates = computed(() =>
  filterRows(templates.value, [(row) => row.name, (row) => row.type, (row) => row.channel_type, (row) => row.title])
)

const filteredDomainRisk = computed(() => filterRows(riskRows.value, [(row) => row.type, (row) => row.name, (row) => row.reason, (row) => row.detail]))

const panelOptions = computed(() => [
  { name: 'alerts', label: '告警事件', count: alerts.value.length },
  { name: 'rules', label: '告警规则', count: rules.value.length },
  { name: 'channels', label: '通知渠道', count: channels.value.length },
  { name: 'notify', label: '通知组与模板', count: groups.value.length + templates.value.length },
  { name: 'domain', label: '域名证书风险', count: riskRows.value.length }
])

const activePanelMeta = computed(
  () => panelOptions.value.find((item) => item.name === activePanel.value) || panelOptions.value[0] || { label: '-', count: 0 }
)

const formatTime = (value) => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString()
}

const safeArray = (res) => (Array.isArray(res?.data?.data) ? res.data.data : [])
const openCurrentPanel = () => go(panelRouteMap[activePanel.value] || '/monitor/overview')

const ackAlert = async (row) => {
  try {
    await axios.post(`/api/v1/alert/alerts/${row.id}/ack`, {}, { headers: authHeaders() })
    ElMessage.success('告警已确认')
    await refreshAll()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '确认告警失败'))
  }
}

const resolveAlert = async (row) => {
  try {
    await axios.post(`/api/v1/alert/alerts/${row.id}/resolve`, {}, { headers: authHeaders() })
    ElMessage.success('告警已恢复')
    await refreshAll()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '恢复告警失败'))
  }
}

const silenceAlert = async (row) => {
  try {
    await ElMessageBox.confirm(
      `确认将告警「${row.alert_name || row.rule_name || row.id}」静默 1 小时吗？`,
      '创建静默',
      { type: 'warning' }
    )
    const now = Date.now()
    const matchers = []
    if (row.rule_id) matchers.push({ name: 'rule_id', op: '=', value: String(row.rule_id) })
    if (row.target) matchers.push({ name: 'target', op: '=', value: String(row.target) })
    if (row.severity) matchers.push({ name: 'severity', op: '=', value: String(row.severity) })
    if (!matchers.length) matchers.push({ name: 'target', op: 'contains', value: '' })
    await axios.post(
      '/api/v1/alert/silences',
      {
        name: `自动静默-${row.rule_name || row.alert_name || row.id}`,
        matchers: JSON.stringify(matchers),
        starts_at: new Date(now).toISOString(),
        ends_at: new Date(now + 60 * 60 * 1000).toISOString(),
        comment: '来自监控告警中心一键静默'
      },
      { headers: authHeaders() }
    )
    ElMessage.success('静默创建成功（1小时）')
    await refreshAll()
  } catch (err) {
    if (!isCancelError(err)) {
      ElMessage.error(getErrorMessage(err, '创建静默失败'))
    }
  }
}

const testChannel = async (row) => {
  try {
    await axios.post(`/api/v1/notify/channels/${row.id}/test`, {}, { headers: authHeaders() })
    ElMessage.success('渠道测试已触发')
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '渠道测试失败'))
  }
}

const triggerRiskRecheck = async (row, { silent = false } = {}) => {
  if (row.kind === 'domain') {
    await axios.post('/api/v1/domain/domains/check', { domain: row.domain || row.name }, { headers: authHeaders() })
    if (!silent) ElMessage.success(`域名 ${row.domain || row.name} 已触发复检`)
    return
  }
  if (row.kind === 'cert') {
    await axios.post(`/api/v1/domain/certs/${row.id}/check`, {}, { headers: authHeaders() })
    if (!silent) ElMessage.success(`证书 ${row.domain || row.name} 已触发复检`)
    return
  }
  throw new Error('暂不支持该风险类型复检')
}

const checkRisk = async (row) => {
  try {
    await triggerRiskRecheck(row)
    await refreshAll()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '触发复检失败'))
  }
}

const batchRecheckRisk = async (mode) => {
  const targetRows = mode === 'stale' ? riskStaleRows.value : riskCriticalRows.value
  if (!targetRows.length) {
    ElMessage.info('没有可复检的目标')
    return
  }
  try {
    await ElMessageBox.confirm(
      `确认批量复检 ${targetRows.length} 个${mode === 'stale' ? '待复检' : '高危'}对象吗？`,
      '批量复检',
      { type: 'warning' }
    )
  } catch (err) {
    if (!isCancelError(err)) ElMessage.error(getErrorMessage(err, '操作已取消'))
    return
  }

  riskBatching.value = true
  try {
    const results = await Promise.allSettled(targetRows.map((row) => triggerRiskRecheck(row, { silent: true })))
    const success = results.filter((item) => item.status === 'fulfilled').length
    const failed = results.length - success
    await refreshAll()
    if (failed > 0) {
      ElMessage.warning(`批量复检完成，成功 ${success}，失败 ${failed}`)
    } else {
      ElMessage.success(`批量复检完成，共 ${success} 项`)
    }
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '批量复检失败'))
  } finally {
    riskBatching.value = false
  }
}

const refreshAll = async () => {
  if (loading.value) return
  loading.value = true
  try {
    const [alertRes, ruleRes, agentRes, domainRes, certRes, channelRes, groupRes, templateRes] = await Promise.allSettled([
      axios.get('/api/v1/alert/alerts', { headers: authHeaders() }),
      axios.get('/api/v1/alert/rules', { headers: authHeaders() }),
      axios.get('/api/v1/monitor/agents', { headers: authHeaders() }),
      axios.get('/api/v1/domain/domains', { headers: authHeaders() }),
      axios.get('/api/v1/domain/certs', { headers: authHeaders() }),
      axios.get('/api/v1/notify/channels', { headers: authHeaders() }),
      axios.get('/api/v1/notify/groups', { headers: authHeaders() }),
      axios.get('/api/v1/notify/templates', { headers: authHeaders() })
    ])

    alerts.value = alertRes.status === 'fulfilled' ? safeArray(alertRes.value) : []
    rules.value = ruleRes.status === 'fulfilled' ? safeArray(ruleRes.value) : []
    agents.value = agentRes.status === 'fulfilled' ? safeArray(agentRes.value) : []
    domains.value = domainRes.status === 'fulfilled' ? safeArray(domainRes.value) : []
    certs.value = certRes.status === 'fulfilled' ? safeArray(certRes.value) : []
    channels.value = channelRes.status === 'fulfilled' ? safeArray(channelRes.value) : []
    groups.value = groupRes.status === 'fulfilled' ? safeArray(groupRes.value) : []
    templates.value = templateRes.status === 'fulfilled' ? safeArray(templateRes.value) : []
    dataSourceStatus.alerts = alertRes.status === 'fulfilled' ? 'ok' : 'error'
    dataSourceStatus.rules = ruleRes.status === 'fulfilled' ? 'ok' : 'error'
    dataSourceStatus.agents = agentRes.status === 'fulfilled' ? 'ok' : 'error'
    dataSourceStatus.domains = domainRes.status === 'fulfilled' ? 'ok' : 'error'
    dataSourceStatus.certs = certRes.status === 'fulfilled' ? 'ok' : 'error'
    dataSourceStatus.channels = channelRes.status === 'fulfilled' ? 'ok' : 'error'
    dataSourceStatus.groups = groupRes.status === 'fulfilled' ? 'ok' : 'error'
    dataSourceStatus.templates = templateRes.status === 'fulfilled' ? 'ok' : 'error'

    stats.alertTotal = alerts.value.length
    stats.alertOpen = alerts.value.filter((item) => Number(item.status) === 0).length
    stats.alertClosed = alerts.value.filter((item) => Number(item.status) === 1 || Number(item.status) === 2).length
    stats.alertCritical = alerts.value.filter((item) => normalizeText(item.severity) === 'critical').length
    stats.alertCriticalOpen = alerts.value.filter((item) => Number(item.status) === 0 && normalizeText(item.severity) === 'critical').length
    stats.ruleTotal = rules.value.length

    stats.agentTotal = agents.value.length
    stats.agentOnline = agents.value.filter((item) => normalizeText(item.status) === 'online').length

    stats.domainHealthy = domains.value.filter((item) => normalizeText(item.health_status) === 'healthy').length
    stats.domainRisk = domains.value.filter((item) => {
      const health = normalizeText(item.health_status)
      return health === 'warning' || health === 'critical'
    }).length

    stats.certExpiringSoon = certs.value.filter((item) => Number(item.days_to_expire || 0) <= 30).length

    stats.notifyChannelTotal = channels.value.length
    stats.notifyGroupTotal = groups.value.length
    stats.templateTotal = templates.value.length

    const failedCount = [alertRes, ruleRes, agentRes, domainRes, certRes, channelRes, groupRes, templateRes].filter((r) => r.status === 'rejected').length
    if (failedCount > 0) {
      ElMessage.warning(`部分监控数据加载失败(${failedCount}项)，已展示可用数据`)
    }
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '加载监控告警中心失败'))
  } finally {
    loading.value = false
  }
}

onMounted(refreshAll)
watch(
  () => route.path,
  (path) => {
    const matched = workbenchTabs.find((item) => item.path === path)
    activeWorkbenchTab.value = matched ? matched.path : '/monitor/center'
  },
  { immediate: true }
)
onMounted(() => {
  minuteTicker = window.setInterval(() => {
    nowTs.value = Date.now()
    autoRefreshTick += 1
    if (autoRefreshTick >= 2) {
      autoRefreshTick = 0
      if (!document.hidden && !loading.value) {
        refreshAll()
      }
    }
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
.hub-tabs-wrap {
  margin-bottom: 12px;
}

.hub-tabs-wrap :deep(.el-tabs__header) {
  margin-bottom: 0;
}

.hub-tabs-wrap :deep(.el-tabs__item) {
  height: 36px;
  line-height: 36px;
  font-size: 14px;
  padding: 0 14px;
}

.hub-tabs-wrap :deep(.el-tabs__nav-wrap::after) {
  background-color: var(--el-border-color-light);
}

.hub-tabs-wrap :deep(.el-tabs__active-bar) {
  height: 2px;
}

.hub-tabs-wrap :deep(.el-tabs__content) {
  display: none;
}
.risk-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.risk-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

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

.capability-card,
.capability-gap-card {
  border: 1px solid #e5e7eb;
}

.capability-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.capability-tags {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.capability-mini-percent {
  margin-top: 2px;
  font-size: 11px;
  color: var(--muted-text);
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

.integration-title-wrap {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
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
