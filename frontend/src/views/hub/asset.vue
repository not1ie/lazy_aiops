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
          <div class="health-row"><span>数据库异常/过期</span><strong>{{ stats.databaseRisk + stats.databaseStale }}</strong></div>
          <div class="health-row"><span>云资源异常/过期</span><strong>{{ stats.cloudRisk + stats.cloudStale }}</strong></div>
          <div class="health-row"><span>堡垒机降级资产</span><strong>{{ stats.jumpRuntimeDegraded }}</strong></div>
          <div class="health-row"><span>Jump同步状态</span><strong>{{ jumpIntegrationStatusText }}</strong></div>
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
                <StatusBadge
                  :text="hostStatusTag(row).text"
                  :type="hostStatusTag(row).type"
                  :reason="hostStatusReason(row)"
                  :updated-at="row.last_check_at || row.updated_at"
                  size="small"
                />
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
          <div class="capability-table-scroll">
            <el-table :fit="true" :data="moduleCapabilityRows" size="small" max-height="320" empty-text="暂无模块能力数据" style="width: 100%; min-width: 1120px">
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
            <el-table-column label="操作" width="90">
              <template #default="{ row }">
                <el-button link type="primary" @click="go(row.path)">进入</el-button>
              </template>
            </el-table-column>
            </el-table>
          </div>
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

      <el-tabs v-model="activePanel" class="integration-tabs">
        <el-tab-pane label="主机" name="hosts">
          <el-table :fit="true" :data="filteredHosts" size="small" max-height="360" empty-text="暂无主机数据">
            <el-table-column prop="name" label="主机名" min-width="180" />
            <el-table-column prop="ip" label="IP" min-width="140" />
            <el-table-column prop="os" label="系统" min-width="140" />
            <el-table-column label="状态" width="100">
              <template #default="{ row }">
                <StatusBadge
                  :text="hostStatusTag(row).text"
                  :type="hostStatusTag(row).type"
                  :reason="hostStatusReason(row)"
                  :updated-at="row.last_check_at || row.updated_at"
                  size="small"
                />
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
                <StatusBadge
                  :text="networkStatusTag(row).text"
                  :type="networkStatusTag(row).type"
                  :reason="networkStatusReason(row)"
                  :updated-at="row.last_check_at || row.updated_at"
                  size="small"
                />
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
            <el-table-column label="状态" width="110">
              <template #default="{ row }">
                <StatusBadge
                  :text="databaseStatusTag(row).text"
                  :type="databaseStatusTag(row).type"
                  :reason="databaseStatusReason(row)"
                  :updated-at="row.updated_at"
                  size="small"
                />
              </template>
            </el-table-column>
            <el-table-column label="状态说明" min-width="180" show-overflow-tooltip>
              <template #default="{ row }">{{ databaseStatusReason(row) }}</template>
            </el-table-column>
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
            <el-table-column label="状态" width="110">
              <template #default="{ row }">
                <StatusBadge
                  :text="cloudStatusTag(row).text"
                  :type="cloudStatusTag(row).type"
                  :reason="cloudStatusReason(row)"
                  :updated-at="row.updated_at"
                  size="small"
                />
              </template>
            </el-table-column>
            <el-table-column label="状态说明" min-width="180" show-overflow-tooltip>
              <template #default="{ row }">{{ cloudStatusReason(row) }}</template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="防火墙" name="firewall">
          <el-table :fit="true" :data="filteredFirewalls" size="small" max-height="360" empty-text="暂无防火墙数据">
            <el-table-column prop="name" label="名称" min-width="150" />
            <el-table-column prop="vendor" label="厂商" min-width="120" />
            <el-table-column prop="ip" label="IP" min-width="140" />
            <el-table-column label="状态" width="110">
              <template #default="{ row }">
                <StatusBadge
                  :text="firewallStatus(row).text"
                  :type="firewallStatus(row).type"
                  :reason="firewallStatusReason(row)"
                  :updated-at="row.last_check_at || row.updated_at"
                  size="small"
                />
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
                <StatusBadge
                  :text="jumpAssetStatus(row).text"
                  :type="jumpAssetStatus(row).type"
                  :reason="jumpAssetStatus(row).reason"
                  :updated-at="row.updated_at"
                  size="small"
                />
              </template>
            </el-table-column>
            <el-table-column label="状态说明" min-width="180" show-overflow-tooltip>
              <template #default="{ row }">{{ jumpAssetStatus(row).reason || '-' }}</template>
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
                <StatusBadge
                  :text="terminalSessionStatus(row).text"
                  :type="terminalSessionStatus(row).type"
                  :reason="terminalSessionStatus(row).reason"
                  :updated-at="row.started_at || row.created_at"
                  size="small"
                />
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
import StatusBadge from '@/components/common/StatusBadge.vue'
import {
  cloudResourceStatusMeta,
  cmdbHostStatusMeta,
  databaseAssetStatusMeta,
  dockerHostStatusMeta,
  jumpIntegrationSyncStatusMeta,
  k8sClusterStatusMeta
} from '@/utils/status'

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
const dockerHosts = ref([])
const k8sClusters = ref([])
const jumpIntegration = ref(null)
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
  databaseRisk: 0,
  databaseStale: 0,
  cloudResourceTotal: 0,
  cloudRisk: 0,
  cloudStale: 0,
  credentialTotal: 0,
  groupTotal: 0,
  firewallDeviceTotal: 0,
  firewallRisk: 0,
  firewallStale: 0,
  jumpAssetTotal: 0,
  jumpRuntimeHealthy: 0,
  jumpRuntimeDegraded: 0,
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
  dockerHosts: 'unknown',
  k8sClusters: 'unknown',
  jumpIntegration: 'unknown',
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
  row?.last_seen ||
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

const hostStatusTag = (row) => cmdbHostStatusMeta(row, { staleMinutes: 3, nowMs: nowMs() })
const hostStatusReason = (row) => {
  const meta = hostStatusTag(row)
  if (meta.key === 'online') return row?.status_reason || '主机状态正常'
  if (meta.key === 'stale') return row?.status_reason || '超过 3 分钟未更新'
  if (meta.key === 'maintenance') return row?.status_reason || '主机处于维护状态'
  if (meta.key === 'offline') return row?.status_reason || '主机离线或不可达'
  return row?.status_reason || '状态未知，建议复核主机连通性'
}

const networkStatusTag = (row, staleMinutes = 5) => {
  if (isMaintenanceStatus(row?.status)) return { key: 'maintenance', text: '维护', type: 'warning' }
  if (isOnlineStatus(row?.status)) {
    if (isStatusStale(row, staleMinutes)) return { key: 'stale', text: '状态过期', type: 'warning' }
    return { key: 'online', text: '在线', type: 'success' }
  }
  const normalized = normalizeText(row?.status)
  if (normalized === 'offline' || Number(row?.status) === 0) return { key: 'offline', text: '离线', type: 'danger' }
  if (isStatusStale(row, staleMinutes)) return { key: 'pending', text: '待检测', type: 'info' }
  return { key: 'unknown', text: '未知', type: 'info' }
}
const networkStatusReason = (row) => {
  const meta = networkStatusTag(row)
  if (meta.key === 'online') return row?.status_reason || '网络设备状态正常'
  if (meta.key === 'stale') return row?.status_reason || '超过 5 分钟未更新'
  if (meta.key === 'maintenance') return row?.status_reason || '设备维护中'
  if (meta.key === 'offline') return row?.status_reason || '设备离线或不可达'
  return row?.status_reason || '状态未知，建议执行连通性巡检'
}

const databaseStatusTag = (row) => databaseAssetStatusMeta(row, { staleDays: 30, nowMs: nowMs() })

const databaseStatusReason = (row) => {
  const meta = databaseStatusTag(row)
  if (meta.key === 'disabled') return '数据库资产被禁用'
  if (meta.key === 'error') return '地址或端口配置缺失'
  if (meta.key === 'stale') return '超过 30 天未更新'
  if (meta.key === 'online') return '配置完整，可用于运维链路'
  return '状态未知，建议复核配置'
}

const cloudStatusTag = (row) => cloudResourceStatusMeta(row, { staleDays: 7, nowMs: nowMs() })

const cloudStatusReason = (row) => {
  const meta = cloudStatusTag(row)
  if (meta.key === 'account_disabled') return '关联云账号已禁用'
  if (meta.key === 'offline') return `云资源状态=${row?.status || 'offline'}`
  if (meta.key === 'error') return `云资源状态=${row?.status || 'error'}`
  if (meta.key === 'pending') return `云资源状态=${row?.status || 'pending'}`
  if (meta.key === 'stale') return '超过 7 天未更新'
  if (meta.key === 'online') return '资源状态正常'
  return '状态未知，建议核对云资源同步任务'
}

const firewallStatus = (row) => {
  const value = Number(row?.status)
  if (value === 2) return { key: 'alert', text: '告警', type: 'danger' }
  if (value === 1) {
    if (isStatusStale(row, 5)) return { key: 'stale', text: '状态过期', type: 'warning' }
    return { key: 'online', text: '在线', type: 'success' }
  }
  if (value === 0) return { key: 'offline', text: '离线', type: 'danger' }
  return { key: 'unknown', text: '未知', type: 'info' }
}
const firewallStatusReason = (row) => {
  const meta = firewallStatus(row)
  if (meta.key === 'online') return row?.status_reason || '防火墙状态正常'
  if (meta.key === 'stale') return row?.status_reason || '超过 5 分钟未更新'
  if (meta.key === 'alert') return row?.status_reason || '存在策略或设备告警'
  if (meta.key === 'offline') return row?.status_reason || '防火墙离线或不可达'
  return row?.status_reason || '状态未知，建议执行SNMP巡检'
}

const jumpIntegrationStatusMeta = computed(() =>
  jumpIntegrationSyncStatusMeta(jumpIntegration.value?.last_sync_status, {
    enabled: jumpIntegration.value?.enabled !== false,
    lastSyncAt: jumpIntegration.value?.last_sync_at,
    nowMs: nowMs(),
    staleMinutes: 30
  })
)

const jumpIntegrationStatusText = computed(() => jumpIntegrationStatusMeta.value.text)

const findCMDBHostByJumpAsset = (row) => {
  const sourceRef = String(row?.source_ref || '').trim()
  if (sourceRef) {
    const byID = hosts.value.find((item) => String(item.id || '') === sourceRef)
    if (byID) return byID
  }
  const address = String(row?.address || '').trim()
  if (address) {
    return hosts.value.find((item) => String(item.ip || '').trim() === address) || null
  }
  return null
}

const findDockerHostByJumpAsset = (row) => {
  const sourceRef = String(row?.source_ref || '').trim()
  if (sourceRef) {
    const byID = dockerHosts.value.find((item) => String(item.id || '') === sourceRef)
    if (byID) return byID
  }
  const address = String(row?.address || '').trim()
  if (address) {
    const byCMDB = hosts.value.find((item) => String(item.ip || '').trim() === address)
    if (byCMDB) {
      return dockerHosts.value.find((item) => String(item.host_id || '') === String(byCMDB.id || '')) || null
    }
  }
  return null
}

const findK8sClusterByJumpAsset = (row) => {
  const sourceRef = String(row?.source_ref || '').trim()
  if (sourceRef) {
    const byID = k8sClusters.value.find((item) => String(item.id || '') === sourceRef)
    if (byID) return byID
  }
  const address = normalizeText(row?.address)
  if (address) {
    return k8sClusters.value.find((item) => normalizeText(item.api_server) === address || normalizeText(item.name) === address) || null
  }
  return null
}

const jumpAssetStatus = (row) => {
  if (!row) return { text: '未知', type: 'info', reason: '资产数据为空' }
  if (row?.enabled === false || row?.status === 0 || normalizeText(row?.status) === 'offline') {
    return { text: '禁用', type: 'info', reason: '资产已禁用' }
  }
  const source = normalizeText(row?.source)
  if (source === 'cmdb_host') {
    const host = findCMDBHostByJumpAsset(row)
    if (!host) return { text: '未映射', type: 'warning', reason: '未匹配到 CMDB 主机' }
    const meta = hostStatusTag(host)
    if (meta.key === 'online') return { text: meta.text, type: meta.type, reason: host.status_reason || '主机状态正常' }
    if (meta.key === 'stale') return { text: meta.text, type: meta.type, reason: host.status_reason || '主机状态检测超时' }
    if (meta.key === 'maintenance') return { text: meta.text, type: meta.type, reason: host.status_reason || '主机维护中' }
    if (meta.key === 'offline') return { text: meta.text, type: meta.type, reason: host.status_reason || '主机不可达' }
    return { text: meta.text, type: meta.type, reason: host.status_reason || '主机状态未知' }
  }
  if (source === 'docker_host') {
    const env = findDockerHostByJumpAsset(row)
    if (!env) return { text: '未映射', type: 'warning', reason: '未匹配到 Docker 环境' }
    const meta = dockerHostStatusMeta(env, { staleMinutes: 3, nowMs: nowMs() })
    if (meta.key === 'online') return { text: meta.text, type: meta.type, reason: env.last_error || 'Docker 环境在线' }
    if (meta.key === 'stale') return { text: meta.text, type: meta.type, reason: env.last_error || 'Docker 状态检测超时' }
    if (meta.key === 'error') return { text: meta.text, type: meta.type, reason: env.last_error || 'Docker 环境异常' }
    if (meta.key === 'offline') return { text: meta.text, type: meta.type, reason: env.last_error || 'Docker 环境离线' }
    if (meta.key === 'maintenance') return { text: meta.text, type: meta.type, reason: env.last_error || 'Docker 环境维护中' }
    return { text: meta.text, type: meta.type, reason: env.last_error || 'Docker 状态未知' }
  }
  if (source === 'k8s_cluster') {
    const cluster = findK8sClusterByJumpAsset(row)
    if (!cluster) return { text: '未映射', type: 'warning', reason: '未匹配到 K8s 集群' }
    const meta = k8sClusterStatusMeta(cluster, { staleMinutes: 5, nowMs: nowMs() })
    if (meta.key === 'online') return { text: meta.text, type: meta.type, reason: cluster.status_reason || '集群状态正常' }
    if (meta.key === 'stale') return { text: meta.text, type: meta.type, reason: cluster.status_reason || 'K8s 状态检测超时' }
    if (meta.key === 'maintenance') return { text: meta.text, type: meta.type, reason: cluster.status_reason || '集群维护中' }
    if (meta.key === 'abnormal') return { text: meta.text, type: meta.type, reason: cluster.status_reason || '集群连接异常' }
    return { text: meta.text, type: meta.type, reason: cluster.status_reason || '集群状态未知' }
  }
  if (source === 'jumpserver' || source === 'jumpserver_host' || source === 'jumpserver_database') {
    const syncMeta = jumpIntegrationStatusMeta.value
    if (syncMeta.key === 'failed') return { text: '同步异常', type: 'danger', reason: jumpIntegration.value?.last_sync_msg || 'JumpServer 最近同步失败' }
    if (syncMeta.key === 'partial' || syncMeta.key === 'stale' || syncMeta.key === 'unknown') {
      return { text: '同步降级', type: 'warning', reason: jumpIntegration.value?.last_sync_msg || 'JumpServer 同步部分成功或状态过期' }
    }
    return { text: '已同步', type: 'success', reason: jumpIntegration.value?.last_sync_msg || 'JumpServer 同步正常' }
  }
  if (source === 'cmdb_database') return { text: '可用', type: 'success', reason: 'CMDB 数据库资产已接入' }
  return { text: '可用', type: 'success', reason: '手工资产，待接入实时检测链路' }
}

const terminalSessionStatus = (row) => {
  const value = Number(row?.status)
  if (value === 1) return { key: 'online', text: '在线', type: 'success', reason: '会话进行中' }
  if (value === 2) return { key: 'closed', text: '已关闭', type: 'info', reason: row?.close_reason || '会话已结束' }
  if (value === 3) return { key: 'failed', text: '失败', type: 'danger', reason: row?.last_error || '会话连接或执行失败' }
  return { key: 'pending', text: '待连', type: 'warning', reason: row?.last_error || '等待建立会话连接' }
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

const jumpHealthyRate = computed(() => {
  if (!stats.jumpAssetTotal) return 0
  return Math.round((stats.jumpRuntimeHealthy / stats.jumpAssetTotal) * 100)
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
  const serviceRiskTotal = stats.databaseRisk + stats.cloudRisk
  const serviceStaleTotal = stats.databaseStale + stats.cloudStale
  const serviceStatus = (stats.databaseTotal + stats.cloudResourceTotal) === 0
    ? 'error'
    : (
      serviceRiskTotal > 0
        ? 'error'
        : (serviceStaleTotal > 0 ? 'warning' : 'ok')
    )

  const jumpLink = worstStatus(
    dataSourceStatus.jumpAssets,
    dataSourceStatus.hosts,
    dataSourceStatus.dockerHosts,
    dataSourceStatus.k8sClusters,
    dataSourceStatus.jumpIntegration
  )
  const jumpRuntimeDegradedRate = stats.jumpAssetTotal > 0
    ? Math.round((stats.jumpRuntimeDegraded / stats.jumpAssetTotal) * 100)
    : 0
  const jumpSyncRisk = jumpIntegrationStatusMeta.value.key === 'failed'
    ? 'error'
    : (['partial', 'stale', 'unknown'].includes(jumpIntegrationStatusMeta.value.key) ? 'warning' : 'ok')
  const jumpStatus = stats.jumpAssetTotal === 0
    ? 'warning'
    : (
      jumpSyncRisk === 'error'
        ? 'error'
        : (jumpRuntimeDegradedRate >= 35 ? 'error' : (jumpRuntimeDegradedRate > 0 || jumpSyncRisk === 'warning' ? 'warning' : 'ok'))
    )

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
      automationScore: (stats.databaseTotal > 0 && stats.cloudResourceTotal > 0) ? 84 : 70,
      targetScore: 86,
      freshnessScore: (stats.databaseTotal + stats.cloudResourceTotal) > 0
        ? clampPercent(100 - serviceRiskTotal * 12 - serviceStaleTotal * 8)
        : 70,
      suggestion: serviceStatus === 'ok'
        ? '保持数据库与云资源账实一致'
        : (serviceRiskTotal > 0 ? '优先修复数据库/云资源异常状态' : '处理过期状态并补齐资源同步')
    },
    {
      key: 'jump',
      label: '堡垒机资产融合',
      path: '/jump/assets',
      linkStatus: jumpLink,
      status: jumpStatus,
      automationScore: stats.jumpAssetTotal > 0 ? 86 : 64,
      targetScore: 88,
      freshnessScore: jumpHealthyRate.value,
      suggestion: jumpStatus === 'ok'
        ? '持续验证来源映射与授权链路'
        : `优先修复降级资产并处理同步状态（${jumpIntegrationStatusMeta.value.text}）`
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
  if (Object.prototype.hasOwnProperty.call(res.data || {}, 'code') && Number(res.data?.code) !== 0) {
    throw new Error(res.data?.message || `${url} 返回异常`)
  }
  return Array.isArray(res.data?.data) ? res.data.data : []
}

const fetchObject = async (url, params = {}) => {
  const res = await axios.get(url, { headers: authHeaders(), params })
  if (Object.prototype.hasOwnProperty.call(res.data || {}, 'code') && Number(res.data?.code) !== 0) {
    throw new Error(res.data?.message || `${url} 返回异常`)
  }
  return res.data?.data && typeof res.data.data === 'object' ? res.data.data : {}
}

const openCurrentPanel = () => {
  go(panelRouteMap[activePanel.value] || '/host')
}

const safeData = (result) => (result.status === 'fulfilled' && Array.isArray(result.value) ? result.value : [])
const safeObject = (result) => (result.status === 'fulfilled' && result.value && typeof result.value === 'object' ? result.value : null)
const settledToStatus = (result) => (result.status === 'fulfilled' ? 'ok' : 'error')

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
      fetchList('/api/v1/terminal/sessions'),
      fetchList('/api/v1/docker/hosts', { sync: 1 }),
      fetchList('/api/v1/k8s/clusters', { live: 1 }),
      fetchObject('/api/v1/jump/integration/config')
    ])

    const [
      hostList,
      groupList,
      credentialList,
      databaseList,
      cloudResourceList,
      networkList,
      firewallList,
      jumpAssetList,
      sessionList,
      dockerHostList,
      k8sClusterList
    ] = settled.map((item, idx) => (idx <= 10 ? safeData(item) : []))
    const jumpIntegrationCfg = safeObject(settled[11])

    hosts.value = hostList
    groups.value = groupList
    credentials.value = credentialList
    databases.value = databaseList
    cloudResources.value = cloudResourceList
    networkDevices.value = networkList
    firewallDevices.value = firewallList
    jumpAssets.value = jumpAssetList
    terminalSessions.value = sessionList
    dockerHosts.value = dockerHostList
    k8sClusters.value = k8sClusterList
    jumpIntegration.value = jumpIntegrationCfg

    dataSourceStatus.hosts = settledToStatus(settled[0])
    dataSourceStatus.groups = settledToStatus(settled[1])
    dataSourceStatus.credentials = settledToStatus(settled[2])
    dataSourceStatus.databases = settledToStatus(settled[3])
    dataSourceStatus.cloudResources = settledToStatus(settled[4])
    dataSourceStatus.networkDevices = settledToStatus(settled[5])
    dataSourceStatus.firewallDevices = settledToStatus(settled[6])
    dataSourceStatus.jumpAssets = settledToStatus(settled[7])
    dataSourceStatus.terminalSessions = settledToStatus(settled[8])
    dataSourceStatus.dockerHosts = settledToStatus(settled[9])
    dataSourceStatus.k8sClusters = settledToStatus(settled[10])
    dataSourceStatus.jumpIntegration = settledToStatus(settled[11])

    stats.hostTotal = hostList.length
    const hostRuntime = hostList.map((item) => hostStatusTag(item))
    stats.hostOnline = hostRuntime.filter((item) => item.key === 'online').length
    stats.hostStale = hostRuntime.filter((item) => item.key === 'stale').length
    stats.groupTotal = groupList.length
    stats.credentialTotal = credentialList.length
    stats.databaseTotal = databaseList.length
    const databaseRuntime = databaseList.map((item) => databaseStatusTag(item))
    stats.databaseRisk = databaseRuntime.filter((item) => item.key === 'error').length
    stats.databaseStale = databaseRuntime.filter((item) => item.key === 'stale').length
    stats.cloudResourceTotal = cloudResourceList.length
    const cloudRuntime = cloudResourceList.map((item) => cloudStatusTag(item))
    stats.cloudRisk = cloudRuntime.filter((item) => item.key === 'offline' || item.key === 'error' || item.key === 'account_disabled').length
    stats.cloudStale = cloudRuntime.filter((item) => item.key === 'stale').length
    stats.networkDeviceTotal = networkList.length
    const networkRuntime = networkList.map((item) => networkStatusTag(item))
    stats.networkDeviceOnline = networkRuntime.filter((item) => item.type === 'success').length
    stats.networkDeviceStale = networkRuntime.filter((item) => item.text === '状态过期').length
    stats.firewallDeviceTotal = firewallList.length
    stats.firewallRisk = firewallList.filter((item) => firewallStatus(item).type === 'danger').length
    stats.firewallStale = firewallList.filter((item) => isStatusStale(item, 5)).length
    stats.jumpAssetTotal = jumpAssetList.length
    const jumpRuntime = jumpAssetList.map((item) => jumpAssetStatus(item))
    stats.jumpRuntimeHealthy = jumpRuntime.filter((item) => item.type === 'success').length
    stats.jumpRuntimeDegraded = jumpRuntime.filter((item) => item.type === 'warning' || item.type === 'danger').length
    stats.terminalSessionTotal = sessionList.length

    if (dataSourceStatus.jumpAssets === 'ok' && stats.jumpRuntimeDegraded > 0) {
      dataSourceStatus.jumpAssets = 'warning'
    }
    if (dataSourceStatus.databases === 'ok' && (stats.databaseRisk > 0 || stats.databaseStale > 0)) {
      dataSourceStatus.databases = 'warning'
    }
    if (dataSourceStatus.cloudResources === 'ok' && (stats.cloudRisk > 0 || stats.cloudStale > 0)) {
      dataSourceStatus.cloudResources = 'warning'
    }
    if (dataSourceStatus.jumpIntegration === 'ok') {
      const syncMeta = jumpIntegrationSyncStatusMeta(jumpIntegrationCfg?.last_sync_status, {
        enabled: jumpIntegrationCfg?.enabled !== false,
        lastSyncAt: jumpIntegrationCfg?.last_sync_at,
        nowMs: nowMs(),
        staleMinutes: 30
      })
      const hasJumpServerAsset = jumpAssetList.some((item) => {
        const source = normalizeText(item?.source)
        return source === 'jumpserver' || source === 'jumpserver_host' || source === 'jumpserver_database'
      })
      if (syncMeta.key === 'failed') {
        dataSourceStatus.jumpIntegration = 'error'
      } else if (
        syncMeta.key === 'partial' ||
        syncMeta.key === 'stale' ||
        syncMeta.key === 'unknown' ||
        (syncMeta.key === 'disabled' && hasJumpServerAsset)
      ) {
        dataSourceStatus.jumpIntegration = 'warning'
      }
    }

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

.capability-table-scroll {
  overflow-x: auto;
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
