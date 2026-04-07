<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>资产总览</h2>
        <p class="page-desc">把主机、网络设备、数据库、云资源、防火墙与堡垒机资产融合在一个工作台里。</p>
      </div>
      <div class="page-actions">
        <el-button :loading="loading" icon="Refresh" @click="refreshAll">刷新</el-button>
      </div>
    </div>

    <el-row :gutter="12" class="summary-row">
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">主机总数</div><div class="metric-value">{{ stats.hostTotal }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">在线主机</div><div class="metric-value ok">{{ stats.hostOnline }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">网络设备</div><div class="metric-value">{{ stats.networkDeviceTotal }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">数据库资产</div><div class="metric-value">{{ stats.databaseTotal }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">云资源</div><div class="metric-value">{{ stats.cloudResourceTotal }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">凭据</div><div class="metric-value">{{ stats.credentialTotal }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">防火墙设备</div><div class="metric-value">{{ stats.firewallDeviceTotal }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">堡垒机资产</div><div class="metric-value">{{ stats.jumpAssetTotal }}</div></el-card>
      </el-col>
    </el-row>

    <el-row :gutter="12">
      <el-col :span="10">
        <el-card>
          <template #header>资产健康</template>
          <div class="health-row">
            <span>主机在线率</span>
            <strong>{{ hostOnlineRate }}%</strong>
          </div>
          <el-progress :percentage="hostOnlineRate" :stroke-width="14" />
          <div class="health-row mtop">
            <span>网络设备在线率</span>
            <strong>{{ networkOnlineRate }}%</strong>
          </div>
          <el-progress :percentage="networkOnlineRate" :stroke-width="14" status="success" />
          <el-divider />
          <div class="health-row"><span>资产分组</span><strong>{{ stats.groupTotal }}</strong></div>
          <div class="health-row"><span>风险防火墙</span><strong>{{ stats.firewallRisk }}</strong></div>
          <div class="health-row"><span>主机状态过期</span><strong>{{ stats.hostStale }}</strong></div>
          <div class="health-row"><span>网络状态过期</span><strong>{{ stats.networkDeviceStale }}</strong></div>
          <div class="health-row"><span>防火墙状态过期</span><strong>{{ stats.firewallStale }}</strong></div>
        </el-card>
      </el-col>
      <el-col :span="14">
        <el-card>
          <template #header>最近资产变更主机</template>
          <el-table :fit="true" :data="recentHosts" size="small" max-height="300">
            <el-table-column prop="name" label="主机名" min-width="130" />
            <el-table-column prop="ip" label="IP" min-width="130" />
            <el-table-column label="状态" width="100">
              <template #default="{ row }">
                <el-tag :type="assetStatusTag(row).type">{{ assetStatusTag(row).text }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="os" label="系统" min-width="120" />
            <el-table-column label="更新时间" min-width="170">
              <template #default="{ row }">{{ formatTime(row.updated_at) }}</template>
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
            <span>资产融合视图</span>
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
              placeholder="筛选名称、IP、类型、区域..."
            />
            <el-button v-if="panelKeyword" size="small" @click="panelKeyword = ''">清空筛选</el-button>
            <el-button size="small" type="primary" plain @click="openCurrentPanel">进入完整页面</el-button>
          </div>
        </div>
      </template>

      <div class="panel-switch">
        <el-check-tag
          v-for="item in panelOptions"
          :key="item.name"
          :checked="activePanel === item.name"
          @change="activePanel = item.name"
        >
          {{ item.label }}
          <span class="panel-switch-count">{{ item.count }}</span>
        </el-check-tag>
      </div>

      <el-tabs v-model="activePanel" class="integration-tabs">
        <el-tab-pane label="主机" name="hosts">
          <el-table :fit="true" :data="filteredHosts" size="small" max-height="360" empty-text="暂无主机数据">
            <el-table-column prop="name" label="主机名" min-width="180" />
            <el-table-column prop="ip" label="IP" min-width="140" />
            <el-table-column prop="os" label="系统" min-width="140" />
            <el-table-column label="状态" width="100">
              <template #default="{ row }">
                <el-tag :type="assetStatusTag(row).type">{{ assetStatusTag(row).text }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="最近检查" min-width="170">
              <template #default="{ row }">{{ formatTime(row.last_check_at) }}</template>
            </el-table-column>
            <el-table-column prop="status_reason" label="异常原因" min-width="170" show-overflow-tooltip>
              <template #default="{ row }">{{ row.status_reason || '-' }}</template>
            </el-table-column>
            <el-table-column label="更新时间" min-width="170">
              <template #default="{ row }">{{ formatTime(row.updated_at) }}</template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="网络设备" name="network">
          <el-table :fit="true" :data="filteredNetworkDevices" size="small" max-height="360" empty-text="暂无网络设备数据">
            <el-table-column prop="name" label="设备名" min-width="160" />
            <el-table-column prop="device_type" label="类型" width="120" />
            <el-table-column prop="vendor" label="厂商" width="120" />
            <el-table-column prop="ip" label="管理IP" min-width="140" />
            <el-table-column label="状态" width="100">
              <template #default="{ row }">
                <el-tag :type="assetStatusTag(row).type">{{ assetStatusTag(row).text }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="最近检查" min-width="170">
              <template #default="{ row }">{{ formatTime(row.last_check_at) }}</template>
            </el-table-column>
            <el-table-column prop="status_reason" label="异常原因" min-width="170" show-overflow-tooltip>
              <template #default="{ row }">{{ row.status_reason || '-' }}</template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="数据库资产" name="database">
          <el-table :fit="true" :data="filteredDatabases" size="small" max-height="360" empty-text="暂无数据库资产数据">
            <el-table-column prop="name" label="名称" min-width="160" />
            <el-table-column prop="type" label="类型" width="120" />
            <el-table-column label="地址" min-width="180">
              <template #default="{ row }">{{ row.host }}:{{ row.port }}</template>
            </el-table-column>
            <el-table-column prop="database" label="库名" min-width="120" />
            <el-table-column prop="environment" label="环境" width="100" />
            <el-table-column prop="owner" label="负责人" width="120" />
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="云资源" name="cloud">
          <el-table :fit="true" :data="filteredCloudResources" size="small" max-height="360" empty-text="暂无云资源数据">
            <el-table-column prop="name" label="资源名称" min-width="180" />
            <el-table-column prop="type" label="类型" width="110" />
            <el-table-column prop="region" label="区域" min-width="120" />
            <el-table-column prop="ip" label="IP" min-width="140" />
            <el-table-column label="账号" min-width="140">
              <template #default="{ row }">{{ row.account?.name || '-' }}</template>
            </el-table-column>
            <el-table-column prop="status" label="状态" width="110" />
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="防火墙" name="firewall">
          <el-table :fit="true" :data="filteredFirewalls" size="small" max-height="360" empty-text="暂无防火墙数据">
            <el-table-column prop="name" label="名称" min-width="150" />
            <el-table-column prop="vendor" label="厂商" min-width="120" />
            <el-table-column prop="ip" label="IP" min-width="140" />
            <el-table-column label="状态" width="110">
              <template #default="{ row }">
                <el-tag :type="firewallStatus(row).type">{{ firewallStatus(row).text }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="最近检查" min-width="170">
              <template #default="{ row }">{{ formatTime(row.last_check_at) }}</template>
            </el-table-column>
            <el-table-column prop="status_reason" label="异常原因" min-width="180" show-overflow-tooltip>
              <template #default="{ row }">{{ row.status_reason || '-' }}</template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="堡垒机资产" name="jump">
          <el-table :fit="true" :data="filteredJumpAssets" size="small" max-height="360" empty-text="暂无堡垒机资产数据">
            <el-table-column prop="name" label="名称" min-width="170" />
            <el-table-column prop="asset_type" label="类型" width="110" />
            <el-table-column prop="protocol" label="协议" width="110" />
            <el-table-column label="地址" min-width="170">
              <template #default="{ row }">{{ row.address }}:{{ row.port }}</template>
            </el-table-column>
            <el-table-column prop="source" label="来源" width="120" />
            <el-table-column label="状态" width="100">
              <template #default="{ row }">
                <el-tag :type="jumpAssetStatus(row).type">{{ jumpAssetStatus(row).text }}</el-tag>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="WebTerminal会话" name="terminal">
          <el-table :fit="true" :data="filteredTerminalSessions" size="small" max-height="360" empty-text="暂无会话数据">
            <el-table-column prop="operator" label="操作人" min-width="120" />
            <el-table-column prop="host" label="主机" min-width="170" />
            <el-table-column prop="username" label="登录用户" min-width="120" />
            <el-table-column prop="port" label="端口" width="90" />
            <el-table-column label="状态" width="100">
              <template #default="{ row }">
                <el-tag :type="terminalSessionStatus(row.status).type">{{ terminalSessionStatus(row.status).text }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="开始时间" min-width="170">
              <template #default="{ row }">{{ formatTime(row.started_at || row.created_at) }}</template>
            </el-table-column>
            <el-table-column prop="last_error" label="失败原因" min-width="180" show-overflow-tooltip>
              <template #default="{ row }">{{ row.last_error || '-' }}</template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </el-card>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'
import { ElMessage } from 'element-plus'

const router = useRouter()
const loading = ref(false)
const hosts = ref([])
const networkDevices = ref([])
const databases = ref([])
const cloudResources = ref([])
const credentials = ref([])
const groups = ref([])
const firewallDevices = ref([])
const jumpAssets = ref([])
const terminalSessions = ref([])
const activePanel = ref('hosts')
const panelKeyword = ref('')
let autoRefreshTimer = null

const stats = reactive({
  hostTotal: 0,
  hostOnline: 0,
  hostStale: 0,
  networkDeviceTotal: 0,
  networkDeviceOnline: 0,
  networkDeviceStale: 0,
  databaseTotal: 0,
  cloudResourceTotal: 0,
  credentialTotal: 0,
  groupTotal: 0,
  firewallDeviceTotal: 0,
  firewallRisk: 0,
  firewallStale: 0,
  jumpAssetTotal: 0,
  terminalSessionTotal: 0
})
const dataSourceStatus = reactive({
  hosts: 'unknown',
  groups: 'unknown',
  credentials: 'unknown',
  databases: 'unknown',
  cloudResources: 'unknown',
  networkDevices: 'unknown',
  firewallDevices: 'unknown',
  jumpAssets: 'unknown',
  terminalSessions: 'unknown'
})


const panelRouteMap = {
  hosts: '/host',
  network: '/cmdb/network-devices',
  database: '/cmdb/database',
  cloud: '/cmdb/cloud',
  firewall: '/firewall',
  jump: '/jump/assets',
  terminal: '/terminal'
}

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })
const go = (path) => router.push(path)
const normalizeText = (value) => String(value ?? '').trim().toLowerCase()
const nowMs = () => Date.now()
const parseTimestamp = (value) => {
  if (!value) return null
  const ts = new Date(value).getTime()
  return Number.isNaN(ts) ? null : ts
}

const statusFreshnessTs = (row) =>
  row?.last_check_at ||
  row?.last_seen_at ||
  row?.last_heartbeat_at ||
  row?.updated_at

const isStatusStale = (row, minutes = 5) => {
  const ts = parseTimestamp(statusFreshnessTs(row))
  if (!ts) return true
  return nowMs() - ts > minutes * 60 * 1000
}

const isOnlineStatus = (status) => {
  const normalized = normalizeText(status)
  return status === true || normalized === 'online' || normalized === 'connected' || normalized === 'running' || Number(status) === 1
}

const isMaintenanceStatus = (status) => {
  const normalized = normalizeText(status)
  return normalized === 'maintenance' || normalized === 'maintain' || Number(status) === 2
}

const assetStatusTag = (row, staleMinutes = 5) => {
  if (isMaintenanceStatus(row?.status)) return { text: '维护', type: 'warning' }
  if (isOnlineStatus(row?.status)) {
    if (isStatusStale(row, staleMinutes)) return { text: '状态过期', type: 'warning' }
    return { text: '在线', type: 'success' }
  }
  const normalized = normalizeText(row?.status)
  if (normalized === 'offline' || Number(row?.status) === 0) return { text: '离线', type: 'danger' }
  if (isStatusStale(row, staleMinutes)) return { text: '待检测', type: 'info' }
  return { text: '未知', type: 'info' }
}

const firewallStatus = (row) => {
  const value = Number(row?.status)
  if (value === 2) return { text: '告警', type: 'danger' }
  if (value === 1) {
    if (isStatusStale(row, 5)) return { text: '状态过期', type: 'warning' }
    return { text: '在线', type: 'success' }
  }
  if (value === 0) return { text: '离线', type: 'danger' }
  return { text: '未知', type: 'info' }
}

const jumpAssetStatus = (row) => {
  if (row?.enabled === false || row?.status === 0 || String(row?.status || '').toLowerCase() === 'offline') {
    return { text: '禁用', type: 'info' }
  }
  return { text: '可用', type: 'success' }
}

const terminalSessionStatus = (status) => {
  const value = Number(status)
  if (value === 1) return { text: '在线', type: 'success' }
  if (value === 2) return { text: '已关闭', type: 'info' }
  if (value === 3) return { text: '失败', type: 'danger' }
  return { text: '待连', type: 'warning' }
}

const filterRows = (rows, fields) => {
  const keyword = normalizeText(panelKeyword.value)
  const base = Array.isArray(rows) ? rows : []
  if (!keyword) return base.slice(0, 20)
  return base
    .filter((row) => fields.some((field) => normalizeText(field(row)).includes(keyword)))
    .slice(0, 20)
}

const hostOnlineRate = computed(() => {
  if (!stats.hostTotal) return 0
  return Math.round((stats.hostOnline / stats.hostTotal) * 100)
})

const networkOnlineRate = computed(() => {
  if (!stats.networkDeviceTotal) return 0
  return Math.round((stats.networkDeviceOnline / stats.networkDeviceTotal) * 100)
})

const jumpEnabledRate = computed(() => {
  if (!stats.jumpAssetTotal) return 0
  const enabled = jumpAssets.value.filter((item) => jumpAssetStatus(item).type === 'success').length
  return Math.round((enabled / stats.jumpAssetTotal) * 100)
})

const terminalFailureCount = computed(() =>
  terminalSessions.value.filter((item) => Number(item?.status) === 3).length
)

const capabilityStatusSeverity = { ok: 1, warning: 2, error: 3, unknown: 0 }
const clampPercent = (value) => Math.max(0, Math.min(100, Math.round(Number(value) || 0)))
const capabilityScoreByStatus = (value) => {
  const status = normalizeText(value)
  if (status === 'ok') return 100
  if (status === 'warning') return 72
  if (status === 'error') return 38
  return 50
}
const worstStatus = (...values) =>
  values.reduce((worst, current) => (capabilityStatusSeverity[normalizeText(current)] > capabilityStatusSeverity[normalizeText(worst)] ? current : worst), 'ok')
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

const moduleCapabilityRows = computed(() => {
  const cmdbLink = worstStatus(dataSourceStatus.hosts, dataSourceStatus.groups, dataSourceStatus.credentials)
  const cmdbStatus = stats.hostTotal === 0 ? 'warning' : (hostOnlineRate.value >= 85 ? 'ok' : hostOnlineRate.value >= 60 ? 'warning' : 'error')

  const networkLink = worstStatus(dataSourceStatus.networkDevices, dataSourceStatus.firewallDevices)
  const networkStatus = stats.firewallRisk > 0
    ? 'error'
    : ((stats.networkDeviceStale + stats.firewallStale) > 0
      ? 'warning'
      : (networkOnlineRate.value >= 80 ? 'ok' : networkOnlineRate.value >= 50 ? 'warning' : 'error'))

  const serviceLink = worstStatus(dataSourceStatus.databases, dataSourceStatus.cloudResources)
  const serviceStatus = (stats.databaseTotal > 0 && stats.cloudResourceTotal > 0)
    ? 'ok'
    : ((stats.databaseTotal + stats.cloudResourceTotal) > 0 ? 'warning' : 'error')

  const jumpLink = worstStatus(dataSourceStatus.jumpAssets, dataSourceStatus.hosts)
  const jumpStatus = stats.jumpAssetTotal === 0 ? 'warning' : (jumpEnabledRate.value >= 80 ? 'ok' : jumpEnabledRate.value >= 50 ? 'warning' : 'error')

  const terminalLink = dataSourceStatus.terminalSessions
  const terminalStatus = stats.terminalSessionTotal === 0
    ? 'warning'
    : (terminalFailureCount.value > 3 ? 'error' : terminalFailureCount.value > 0 ? 'warning' : 'ok')

  const rows = [
    {
      key: 'cmdb',
      label: 'CMDB主机资产',
      path: '/host',
      linkStatus: cmdbLink,
      status: cmdbStatus,
      automationScore: stats.credentialTotal > 0 ? 86 : 72,
      targetScore: 90,
      freshnessScore: clampPercent(100 - stats.hostStale * 8),
      suggestion: cmdbStatus === 'ok' ? '保持主机状态巡检与凭据轮换' : '优先修复离线/过期状态主机与凭据覆盖'
    },
    {
      key: 'network',
      label: '网络与防火墙',
      path: '/firewall',
      linkStatus: networkLink,
      status: networkStatus,
      automationScore: stats.firewallDeviceTotal > 0 ? 84 : 70,
      targetScore: 89,
      freshnessScore: clampPercent(100 - stats.networkDeviceStale * 8 - stats.firewallStale * 10 - stats.firewallRisk * 6),
      suggestion: networkStatus === 'ok' ? '保持网络巡检与策略审计' : '先处理防火墙高风险与状态过期设备'
    },
    {
      key: 'service',
      label: '数据库与云资源',
      path: '/cmdb/database',
      linkStatus: serviceLink,
      status: serviceStatus,
      automationScore: (stats.databaseTotal > 0 && stats.cloudResourceTotal > 0) ? 82 : 68,
      targetScore: 86,
      freshnessScore: (stats.databaseTotal + stats.cloudResourceTotal) > 0 ? 100 : 70,
      suggestion: serviceStatus === 'ok' ? '保持资源账实一致' : '补齐数据库/云资源接入，避免盲区'
    },
    {
      key: 'jump',
      label: '堡垒机资产融合',
      path: '/jump/assets',
      linkStatus: jumpLink,
      status: jumpStatus,
      automationScore: stats.jumpAssetTotal > 0 ? 86 : 64,
      targetScore: 88,
      freshnessScore: jumpEnabledRate.value,
      suggestion: jumpStatus === 'ok' ? '持续验证来源映射与授权链路' : '修复禁用资产并校验来源映射关系'
    },
    {
      key: 'terminal',
      label: 'WebTerminal会话审计',
      path: '/terminal',
      linkStatus: terminalLink,
      status: terminalStatus,
      automationScore: stats.terminalSessionTotal > 0 ? 80 : 66,
      targetScore: 85,
      freshnessScore: stats.terminalSessionTotal > 0
        ? clampPercent(100 - Math.round((terminalFailureCount.value / Math.max(1, stats.terminalSessionTotal)) * 100))
        : 70,
      suggestion: terminalStatus === 'ok' ? '保持会话审计持续采集' : '补齐失败会话诊断并提升连通稳定性'
    }
  ]

  return rows.map((item) => {
    const score = clampPercent(
      capabilityScoreByStatus(item.status) * 0.4 +
      capabilityScoreByStatus(item.linkStatus) * 0.3 +
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
      gap: item.level === 'gap' ? '核心资产链路与状态感知不足' : '链路存在降级，自动巡检能力待增强',
      impact: item.level === 'gap' ? '资产状态失真，影响故障定位与授权' : '排障效率下降，异常发现延迟',
      path: item.path,
      score: item.score
    }))
    .sort((a, b) => a.score - b.score)
    .slice(0, 8)
)

const recentHosts = computed(() => {
  return [...hosts.value]
    .sort((a, b) => new Date(b.updated_at || 0).getTime() - new Date(a.updated_at || 0).getTime())
    .slice(0, 8)
})

const filteredHosts = computed(() =>
  filterRows(hosts.value, [(row) => row.name, (row) => row.ip, (row) => row.os, (row) => row.group])
)

const filteredNetworkDevices = computed(() =>
  filterRows(networkDevices.value, [(row) => row.name, (row) => row.device_type, (row) => row.ip, (row) => row.vendor])
)

const filteredDatabases = computed(() =>
  filterRows(databases.value, [(row) => row.name, (row) => row.type, (row) => row.host, (row) => row.database, (row) => row.owner])
)

const filteredCloudResources = computed(() =>
  filterRows(cloudResources.value, [(row) => row.name, (row) => row.type, (row) => row.ip, (row) => row.region, (row) => row.account?.name])
)

const filteredFirewalls = computed(() =>
  filterRows(firewallDevices.value, [(row) => row.name, (row) => row.vendor, (row) => row.ip])
)

const filteredJumpAssets = computed(() =>
  filterRows(jumpAssets.value, [(row) => row.name, (row) => row.asset_type, (row) => row.protocol, (row) => row.address, (row) => row.source])
)

const filteredTerminalSessions = computed(() =>
  filterRows(terminalSessions.value, [(row) => row.operator, (row) => row.host, (row) => row.username, (row) => row.last_error, (row) => row.session_no])
)

const panelOptions = computed(() => [
  { name: 'hosts', label: '主机', count: hosts.value.length },
  { name: 'network', label: '网络设备', count: networkDevices.value.length },
  { name: 'database', label: '数据库资产', count: databases.value.length },
  { name: 'cloud', label: '云资源', count: cloudResources.value.length },
  { name: 'firewall', label: '防火墙', count: firewallDevices.value.length },
  { name: 'jump', label: '堡垒机资产', count: jumpAssets.value.length },
  { name: 'terminal', label: 'WebTerminal会话', count: terminalSessions.value.length }
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

const fetchList = async (url, params = {}) => {
  const res = await axios.get(url, { headers: authHeaders(), params })
  return Array.isArray(res.data?.data) ? res.data.data : []
}

const openCurrentPanel = () => {
  go(panelRouteMap[activePanel.value] || '/host')
}

const safeData = (result) => (result.status === 'fulfilled' && Array.isArray(result.value) ? result.value : [])

const refreshAll = async () => {
  if (loading.value) return
  loading.value = true
  try {
    const settled = await Promise.allSettled([
      fetchList('/api/v1/cmdb/hosts', { live: 1 }),
      fetchList('/api/v1/cmdb/groups'),
      fetchList('/api/v1/cmdb/credentials'),
      fetchList('/api/v1/cmdb/databases'),
      fetchList('/api/v1/cmdb/cloud/resources'),
      fetchList('/api/v1/cmdb/network-devices', { live: 1 }),
      fetchList('/api/v1/firewall/devices', { live: 1 }),
      fetchList('/api/v1/jump/assets'),
      fetchList('/api/v1/terminal/sessions')
    ])

    const [hostList, groupList, credentialList, databaseList, cloudResourceList, networkList, firewallList, jumpAssetList, sessionList] = settled.map(safeData)

    hosts.value = hostList
    groups.value = groupList
    credentials.value = credentialList
    databases.value = databaseList
    cloudResources.value = cloudResourceList
    networkDevices.value = networkList
    firewallDevices.value = firewallList
    jumpAssets.value = jumpAssetList
    terminalSessions.value = sessionList
    dataSourceStatus.hosts = settled[0].status === 'fulfilled' ? 'ok' : 'error'
    dataSourceStatus.groups = settled[1].status === 'fulfilled' ? 'ok' : 'error'
    dataSourceStatus.credentials = settled[2].status === 'fulfilled' ? 'ok' : 'error'
    dataSourceStatus.databases = settled[3].status === 'fulfilled' ? 'ok' : 'error'
    dataSourceStatus.cloudResources = settled[4].status === 'fulfilled' ? 'ok' : 'error'
    dataSourceStatus.networkDevices = settled[5].status === 'fulfilled' ? 'ok' : 'error'
    dataSourceStatus.firewallDevices = settled[6].status === 'fulfilled' ? 'ok' : 'error'
    dataSourceStatus.jumpAssets = settled[7].status === 'fulfilled' ? 'ok' : 'error'
    dataSourceStatus.terminalSessions = settled[8].status === 'fulfilled' ? 'ok' : 'error'

    stats.hostTotal = hostList.length
    stats.hostOnline = hostList.filter((item) => isOnlineStatus(item.status)).length
    stats.hostStale = hostList.filter((item) => isStatusStale(item, 3)).length
    stats.groupTotal = groupList.length
    stats.credentialTotal = credentialList.length
    stats.databaseTotal = databaseList.length
    stats.cloudResourceTotal = cloudResourceList.length
    stats.networkDeviceTotal = networkList.length
    stats.networkDeviceOnline = networkList.filter((item) => isOnlineStatus(item.status)).length
    stats.networkDeviceStale = networkList.filter((item) => isStatusStale(item, 5)).length
    stats.firewallDeviceTotal = firewallList.length
    stats.firewallRisk = firewallList.filter((item) => firewallStatus(item).type === 'danger').length
    stats.firewallStale = firewallList.filter((item) => isStatusStale(item, 5)).length
    stats.jumpAssetTotal = jumpAssetList.length
    stats.terminalSessionTotal = sessionList.length

    const failed = settled.filter((item) => item.status === 'rejected').length
    if (failed > 0) {
      ElMessage.warning(`部分资产源加载失败（${failed}项），已展示可用数据`)
    }
  } catch (err) {
    ElMessage.error(err?.response?.data?.message || err?.message || '加载资产总览失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  refreshAll()
  autoRefreshTimer = window.setInterval(() => {
    if (document.hidden || loading.value) return
    refreshAll()
  }, 90 * 1000)
})

onBeforeUnmount(() => {
  if (autoRefreshTimer) {
    window.clearInterval(autoRefreshTimer)
    autoRefreshTimer = null
  }
})
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 12px; gap: 12px; }
.page-desc { color: var(--muted-text); margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; }
.summary-row { margin-bottom: 12px; }
.summary-row :deep(.el-card) { margin-bottom: 8px; }
.metric-title { color: var(--muted-text); font-size: 12px; }
.metric-value { font-size: 20px; font-weight: 600; margin-top: 6px; color: var(--el-text-color-primary); }
.ok { color: #67c23a; }
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

.panel-switch {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 10px;
}

.panel-switch-count {
  margin-left: 6px;
  opacity: 0.8;
}

.integration-tabs :deep(.el-tabs__header) { display: none; }

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
