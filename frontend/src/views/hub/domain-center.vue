<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>域名监控中心</h2>
        <p class="page-desc">集中展示域名可用性、证书到期风险与处置入口，避免分散排查。</p>
      </div>
      <div class="page-actions">
        <el-button type="primary" plain @click="applyRecommendedWorkspace">推荐工作台</el-button>
        <el-button :loading="checking" type="primary" plain @click="checkAllDomains">域名一键体检</el-button>
        <el-button :loading="checkingCert" plain @click="checkAllCerts">证书批量检查</el-button>
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
        <el-card><div class="metric-title">域名总数</div><div class="metric-value">{{ stats.domainTotal }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">健康域名</div><div class="metric-value ok">{{ stats.domainHealthy }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">风险域名</div><div class="metric-value warning">{{ stats.domainWarning }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">故障域名</div><div class="metric-value danger">{{ stats.domainCritical }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">证书总数</div><div class="metric-value">{{ stats.certTotal }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">证书≤30天</div><div class="metric-value warning">{{ stats.certExpiringSoon }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">证书≤7天</div><div class="metric-value danger">{{ stats.certCriticalSoon }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card>
          <div class="metric-title">待复检对象</div>
          <div class="metric-value warning">{{ stats.pendingRecheck }}</div>
          <div class="metric-sub">云账号 {{ stats.accountTotal }}</div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="12">
      <el-col :span="10">
        <el-card>
          <template #header>健康与到期结构</template>
          <div ref="healthChartRef" class="chart-box"></div>
          <el-divider />
          <div class="health-row"><span>域名可用率</span><strong>{{ domainHealthyRate }}%</strong></div>
          <el-progress :percentage="domainHealthyRate" :stroke-width="14" />
          <div class="health-row mtop"><span>Domain/SSL 相关告警</span><strong>{{ domainAlertCount }}</strong></div>
        </el-card>

        <el-card class="mt-12">
          <template #header>证书到期分布</template>
          <el-table :fit="true" :data="expiryBuckets" size="small" max-height="220">
            <el-table-column prop="label" label="区间" min-width="120" />
            <el-table-column prop="count" label="数量" width="90" />
            <el-table-column label="风险" width="100">
              <template #default="{ row }">
                <el-tag :type="row.level">{{ row.levelText }}</el-tag>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>

      <el-col :span="14">
        <el-card>
          <template #header>风险清单（可直接处置）</template>
          <el-table :fit="true" :data="riskRows" size="small" max-height="470" empty-text="暂无风险项">
            <el-table-column prop="type" label="类型" width="90" />
            <el-table-column prop="name" label="对象" min-width="170" />
            <el-table-column prop="reason" label="风险" min-width="140">
              <template #default="{ row }">
                <el-tag :type="row.level">{{ row.reason }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="detail" label="详情" min-width="180" show-overflow-tooltip />
            <el-table-column label="操作" min-width="150">
              <template #default="{ row }">
                <el-button link type="primary" @click="handleRiskAction(row)">复检</el-button>
                <el-button link @click="go(row.path)">去处理</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <el-card class="integration-card">
      <template #header>
        <div class="integration-header">
          <span>域名融合视图</span>
          <div class="integration-actions">
            <el-input
              v-model="panelKeyword"
              clearable
              size="small"
              class="panel-search"
              placeholder="筛选域名、证书、账号、告警..."
            />
            <el-button size="small" type="primary" plain @click="openCurrentPanel">进入完整页面</el-button>
          </div>
        </div>
      </template>

      <el-tabs v-model="activePanel" class="integration-tabs">
        <el-tab-pane label="域名体检" name="domains">
          <el-table :fit="true" :data="filteredDomains" size="small" max-height="360" empty-text="暂无域名数据">
            <el-table-column prop="domain" label="域名" min-width="180" />
            <el-table-column label="云账号" min-width="130">
              <template #default="{ row }">{{ row.account?.name || '-' }}</template>
            </el-table-column>
            <el-table-column prop="provider" label="厂商" width="100" />
            <el-table-column label="DNS" width="80">
              <template #default="{ row }">
                <el-tag :type="row.dns_resolved ? 'success' : 'danger'">{{ row.dns_resolved ? '正常' : '失败' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="HTTP" width="80">
              <template #default="{ row }">{{ row.http_status_code || '-' }}</template>
            </el-table-column>
            <el-table-column label="证书剩余天" width="100">
              <template #default="{ row }">{{ sslDays(row) }}</template>
            </el-table-column>
            <el-table-column label="健康状态" width="110">
              <template #default="{ row }">
                <el-tag :type="healthTag(row.health_status)">{{ healthText(row.health_status) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="最后体检" min-width="165">
              <template #default="{ row }">{{ formatTime(row.last_check_at) }}</template>
            </el-table-column>
            <el-table-column label="检查时效" width="110">
              <template #default="{ row }">
                <el-tag :type="isCheckStale(row.last_check_at) ? 'warning' : 'success'">
                  {{ isCheckStale(row.last_check_at) ? '待复检' : '及时' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="150">
              <template #default="{ row }">
                <el-button size="small" link type="primary" @click="checkDomainRow(row)">检测</el-button>
                <el-button size="small" link @click="go('/domain/ssl')">详情</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="SSL证书" name="certs">
          <el-table :fit="true" :data="filteredCerts" size="small" max-height="360" empty-text="暂无证书数据">
            <el-table-column prop="domain" label="域名" min-width="170" />
            <el-table-column prop="issuer" label="颁发者" min-width="160" />
            <el-table-column label="到期时间" min-width="165">
              <template #default="{ row }">{{ formatTime(row.not_after) }}</template>
            </el-table-column>
            <el-table-column label="剩余天数" width="95">
              <template #default="{ row }">{{ certDays(row) }}</template>
            </el-table-column>
            <el-table-column label="状态" width="110">
              <template #default="{ row }">
                <el-tag :type="certStatusTag(row)">{{ certStatusText(row) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="最后检查" min-width="165">
              <template #default="{ row }">{{ formatTime(row.last_check_at) }}</template>
            </el-table-column>
            <el-table-column label="检查时效" width="110">
              <template #default="{ row }">
                <el-tag :type="isCheckStale(row.last_check_at) ? 'warning' : 'success'">
                  {{ isCheckStale(row.last_check_at) ? '待复检' : '及时' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="150">
              <template #default="{ row }">
                <el-button size="small" link type="primary" @click="checkCertRow(row)">检查</el-button>
                <el-button size="small" link @click="go('/domain/ssl')">详情</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="云账号" name="accounts">
          <el-table :fit="true" :data="filteredAccounts" size="small" max-height="360" empty-text="暂无云账号数据">
            <el-table-column prop="name" label="账号名" min-width="150" />
            <el-table-column prop="provider" label="厂商" width="120" />
            <el-table-column prop="region" label="区域" min-width="120" />
            <el-table-column prop="domain_count" label="域名数" width="90" />
            <el-table-column label="状态" width="100">
              <template #default="{ row }">
                <el-tag :type="Number(row.status) === 1 ? 'success' : 'warning'">{{ Number(row.status) === 1 ? '可用' : '异常' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="最近同步" min-width="165">
              <template #default="{ row }">{{ formatTime(row.last_sync_at) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="150">
              <template #default="{ row }">
                <el-button size="small" link type="primary" @click="syncAccount(row)">同步</el-button>
                <el-button size="small" link @click="go('/domain/ssl')">详情</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="关联告警" name="alerts">
          <el-table :fit="true" :data="filteredDomainAlerts" size="small" max-height="360" empty-text="暂无域名相关告警">
            <el-table-column prop="rule_name" label="规则" min-width="170" />
            <el-table-column prop="target" label="目标" min-width="160" />
            <el-table-column prop="metric" label="指标" width="110" />
            <el-table-column label="级别" width="90">
              <template #default="{ row }">
                <el-tag :type="severityTag(row.severity)">{{ severityText(row.severity) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="状态" width="100">
              <template #default="{ row }">
                <el-tag :type="alertStatusTag(row.status)">{{ alertStatusText(row.status) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="触发时间" min-width="165">
              <template #default="{ row }">{{ formatTime(row.fired_at || row.created_at) }}</template>
            </el-table-column>
            <el-table-column label="操作" min-width="230">
              <template #default="{ row }">
                <el-space wrap>
                  <el-button
                    v-if="Number(row.status) === 0"
                    size="small"
                    link
                    type="primary"
                    @click="ackAlert(row)"
                  >
                    确认
                  </el-button>
                  <el-button
                    v-if="Number(row.status) === 0 || Number(row.status) === 1"
                    size="small"
                    link
                    type="success"
                    @click="resolveAlert(row)"
                  >
                    恢复
                  </el-button>
                  <el-button size="small" link type="warning" @click="silenceAlert(row)">静默1h</el-button>
                  <el-button size="small" link @click="go('/alert/events')">详情</el-button>
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
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'
import * as echarts from 'echarts'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getErrorMessage, isCancelError } from '@/utils/error'
import { requestApplyWorkspaceCategory } from '@/utils/workspace'

const router = useRouter()
const loading = ref(false)
const checking = ref(false)
const checkingCert = ref(false)
const domains = ref([])
const certs = ref([])
const accounts = ref([])
const alerts = ref([])
const nowTick = ref(Date.now())
let dayTicker = null

const activePanel = ref('domains')
const panelKeyword = ref('')

const stats = reactive({
  domainTotal: 0,
  domainHealthy: 0,
  domainWarning: 0,
  domainCritical: 0,
  certTotal: 0,
  certExpiringSoon: 0,
  certCriticalSoon: 0,
  accountTotal: 0,
  pendingRecheck: 0
})

const CHECK_STALE_HOURS = 24

const quickTabs = [
  { label: '域名与证书', path: '/domain/ssl' },
  { label: '告警事件', path: '/alert/events' },
  { label: '通知模板', path: '/notify/templates' },
  { label: '监控告警中心', path: '/monitor/center' }
]

const panelRouteMap = {
  domains: '/domain/ssl',
  certs: '/domain/ssl',
  accounts: '/domain/ssl',
  alerts: '/alert/events'
}

const applyRecommendedWorkspace = () => requestApplyWorkspaceCategory('monitor', 'hub-domain-center')

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })
const go = (path) => router.push(path)

const healthChartRef = ref(null)
let healthChart = null

const normalizeText = (value) => String(value ?? '').trim().toLowerCase()

const calcDaysToExpire = (expireValue, fallback = 0) => {
  const current = nowTick.value
  if (!expireValue) return Number(fallback || 0)
  const expireAt = new Date(expireValue)
  if (Number.isNaN(expireAt.getTime())) return Number(fallback || 0)
  const remainMs = expireAt.getTime() - current
  if (remainMs <= 0) return 0
  return Math.ceil(remainMs / (24 * 60 * 60 * 1000))
}

const calcCheckAgeHours = (checkedAt) => {
  if (!checkedAt) return Number.POSITIVE_INFINITY
  const checkTime = new Date(checkedAt)
  if (Number.isNaN(checkTime.getTime())) return Number.POSITIVE_INFINITY
  const ageMs = nowTick.value - checkTime.getTime()
  if (ageMs < 0) return 0
  return ageMs / (60 * 60 * 1000)
}

const isCheckStale = (checkedAt) => calcCheckAgeHours(checkedAt) >= CHECK_STALE_HOURS

const formatCheckFreshness = (checkedAt) => {
  if (!checkedAt) return '从未检查'
  const hours = calcCheckAgeHours(checkedAt)
  if (!Number.isFinite(hours)) return '从未检查'
  if (hours < 1) return '1小时内'
  return `${Math.floor(hours)}h前`
}

const certDays = (row) => calcDaysToExpire(row?.not_after, row?.days_to_expire)
const sslDays = (row) => calcDaysToExpire(row?.ssl_not_after, row?.ssl_days_to_expire)

const certStatusText = (row) => {
  const days = certDays(row)
  if (days <= 0 || Number(row?.status) === 0) return '已过期'
  if (days <= 30 || Number(row?.status) === 2) return '即将过期'
  return '正常'
}

const certStatusTag = (row) => {
  const days = certDays(row)
  if (days <= 0 || Number(row?.status) === 0) return 'danger'
  if (days <= 30 || Number(row?.status) === 2) return 'warning'
  return 'success'
}

const healthText = (status) => {
  const v = normalizeText(status)
  if (v === 'healthy') return '健康'
  if (v === 'warning') return '风险'
  if (v === 'critical') return '故障'
  return '未知'
}

const healthTag = (status) => {
  const v = normalizeText(status)
  if (v === 'healthy') return 'success'
  if (v === 'warning') return 'warning'
  if (v === 'critical') return 'danger'
  return 'info'
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
  const v = Number(status)
  if (v === 0) return '触发'
  if (v === 1) return '确认'
  if (v === 2) return '恢复'
  if (v === 3) return '抑制'
  return '未知'
}

const alertStatusTag = (status) => {
  const v = Number(status)
  if (v === 0) return 'danger'
  if (v === 2) return 'success'
  return 'info'
}

const domainHealthyRate = computed(() => {
  if (!stats.domainTotal) return 0
  return Math.round((stats.domainHealthy / stats.domainTotal) * 100)
})

const expiryBuckets = computed(() => {
  const values = { critical: 0, warning: 0, safe: 0, unknown: 0 }
  certs.value.forEach((item) => {
    const days = Number(certDays(item))
    if (Number.isNaN(days)) {
      values.unknown += 1
      return
    }
    if (days <= 7) values.critical += 1
    else if (days <= 30) values.warning += 1
    else values.safe += 1
  })
  return [
    { label: '0-7天', count: values.critical, level: 'danger', levelText: '高危' },
    { label: '8-30天', count: values.warning, level: 'warning', levelText: '预警' },
    { label: '31天以上', count: values.safe, level: 'success', levelText: '正常' },
    { label: '未知', count: values.unknown, level: 'info', levelText: '待检查' }
  ]
})

const riskRows = computed(() => {
  const rows = []
  domains.value.forEach((item) => {
    const health = normalizeText(item.health_status)
    if (health === 'warning' || health === 'critical') {
      rows.push({
        kind: 'domain',
        domain: item.domain,
        type: '域名',
        name: item.domain,
        reason: health === 'critical' ? '可用性故障' : '可用性预警',
        detail: `HTTP ${item.http_status_code || '-'} / ${item.response_time_ms || '-'}ms`,
        level: health === 'critical' ? 'danger' : 'warning',
        path: '/domain/ssl'
      })
    }

    const certRemain = Number(sslDays(item))
    if (!Number.isNaN(certRemain) && certRemain <= 30) {
      rows.push({
        kind: 'domain',
        domain: item.domain,
        type: '域名证书',
        name: item.domain,
        reason: certRemain <= 7 ? '即将过期' : '临近到期',
        detail: `剩余 ${certRemain} 天`,
        level: certRemain <= 7 ? 'danger' : 'warning',
        path: '/domain/ssl'
      })
    }

    if (isCheckStale(item.last_check_at)) {
      rows.push({
        kind: 'domain',
        domain: item.domain,
        type: '域名巡检',
        name: item.domain,
        reason: '检查过期',
        detail: `上次检查 ${formatCheckFreshness(item.last_check_at)}`,
        level: 'warning',
        path: '/domain/ssl'
      })
    }
  })

  certs.value.forEach((item) => {
    const days = Number(certDays(item))
    if (days <= 30) {
      rows.push({
        kind: 'cert',
        id: item.id,
        domain: item.domain,
        type: '证书',
        name: item.domain,
        reason: days <= 7 ? '即将过期' : '临近到期',
        detail: `剩余 ${days} 天`,
        level: days <= 7 ? 'danger' : 'warning',
        path: '/domain/ssl'
      })
    }

    if (isCheckStale(item.last_check_at)) {
      rows.push({
        kind: 'cert',
        id: item.id,
        domain: item.domain,
        type: '证书巡检',
        name: item.domain,
        reason: '检查过期',
        detail: `上次检查 ${formatCheckFreshness(item.last_check_at)}`,
        level: 'warning',
        path: '/domain/ssl'
      })
    }
  })

  return rows
    .sort((a, b) => (a.level === 'danger' ? -1 : 1) - (b.level === 'danger' ? -1 : 1))
    .slice(0, 30)
})

const domainAlerts = computed(() => {
  const allDomains = new Set(domains.value.map((item) => normalizeText(item.domain)).filter(Boolean))
  return alerts.value.filter((item) => {
    const typeHint = normalizeText(item.type)
    const target = normalizeText(item.target)
    const metric = normalizeText(item.metric)
    const ruleName = normalizeText(item.rule_name)

    if (typeHint === 'domain' || typeHint === 'ssl') return true
    if (metric.includes('ssl') || metric.includes('http') || metric.includes('dns')) return true
    if (ruleName.includes('domain') || ruleName.includes('ssl') || ruleName.includes('证书') || ruleName.includes('域名')) return true
    if ([...allDomains].some((domain) => domain && target.includes(domain))) return true
    return false
  })
})

const domainAlertCount = computed(() => domainAlerts.value.length)

const filterRows = (rows, fields) => {
  const keyword = normalizeText(panelKeyword.value)
  const base = Array.isArray(rows) ? rows : []
  if (!keyword) return base.slice(0, 30)
  return base.filter((row) => fields.some((field) => normalizeText(field(row)).includes(keyword))).slice(0, 30)
}

const filteredDomains = computed(() =>
  filterRows(domains.value, [(row) => row.domain, (row) => row.provider, (row) => row.account?.name, (row) => row.health_status, (row) => row.http_status_code])
)

const filteredCerts = computed(() =>
  filterRows(certs.value, [(row) => row.domain, (row) => row.issuer, (row) => certStatusText(row), (row) => certDays(row)])
)

const filteredAccounts = computed(() =>
  filterRows(accounts.value, [(row) => row.name, (row) => row.provider, (row) => row.region, (row) => row.domain_count])
)

const filteredDomainAlerts = computed(() =>
  filterRows(domainAlerts.value, [(row) => row.rule_name, (row) => row.target, (row) => row.metric, (row) => row.severity, (row) => alertStatusText(row.status)])
)

const renderHealthChart = () => {
  if (!healthChartRef.value) return
  if (!healthChart) healthChart = echarts.init(healthChartRef.value)
  healthChart.setOption({
    tooltip: { trigger: 'item' },
    legend: { bottom: 0 },
    series: [
      {
        name: '域名健康',
        type: 'pie',
        radius: ['42%', '70%'],
        itemStyle: { borderRadius: 8, borderColor: 'transparent', borderWidth: 2 },
        label: { formatter: '{b}: {c}' },
        data: [
          { value: stats.domainHealthy, name: '健康', itemStyle: { color: '#67c23a' } },
          { value: stats.domainWarning, name: '预警', itemStyle: { color: '#e6a23c' } },
          { value: stats.domainCritical, name: '故障', itemStyle: { color: '#f56c6c' } }
        ]
      }
    ]
  })
}

const formatTime = (value) => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString()
}

const safeArray = (res) => (Array.isArray(res?.data?.data) ? res.data.data : [])

const openCurrentPanel = () => {
  go(panelRouteMap[activePanel.value] || '/domain/ssl')
}

const checkDomainRow = async (row) => {
  try {
    await axios.post('/api/v1/domain/domains/check', { domain: row.domain }, { headers: authHeaders() })
    ElMessage.success(`域名 ${row.domain} 已触发体检`)
    await refreshAll()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '域名体检失败'))
  }
}

const checkCertRow = async (row) => {
  try {
    await axios.post(`/api/v1/domain/certs/${row.id}/check`, {}, { headers: authHeaders() })
    ElMessage.success(`证书 ${row.domain} 已触发检查`)
    await refreshAll()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '证书检查失败'))
  }
}

const syncAccount = async (row) => {
  try {
    await axios.post(`/api/v1/domain/accounts/${row.id}/sync`, {}, { headers: authHeaders() })
    ElMessage.success(`云账号 ${row.name} 同步已触发`)
    await refreshAll()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '云账号同步失败'))
  }
}

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
      `确认将告警「${row.rule_name || row.target || row.id}」静默 1 小时吗？`,
      '创建静默',
      { type: 'warning' }
    )
    const now = Date.now()
    const matchers = []
    if (row.rule_id) matchers.push({ name: 'rule_id', op: '=', value: String(row.rule_id) })
    if (row.target) matchers.push({ name: 'target', op: '=', value: String(row.target) })
    if (row.severity) matchers.push({ name: 'severity', op: '=', value: String(row.severity) })
    if (!matchers.length) matchers.push({ name: 'target', op: '=~', value: String(row.target || '') })
    await axios.post(
      '/api/v1/alert/silences',
      {
        name: `域名中心静默-${row.rule_name || row.id}`,
        matchers: JSON.stringify(matchers),
        starts_at: new Date(now).toISOString(),
        ends_at: new Date(now + 60 * 60 * 1000).toISOString(),
        comment: '来自域名监控中心的一键静默'
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

const handleRiskAction = async (row) => {
  if (row.kind === 'cert' && row.id) {
    await checkCertRow({ id: row.id, domain: row.domain || row.name })
    return
  }
  await checkDomainRow({ domain: row.domain || row.name })
}

const refreshAll = async () => {
  loading.value = true
  try {
    const [domainRes, certRes, accountRes, alertRes] = await Promise.allSettled([
      axios.get('/api/v1/domain/domains', { headers: authHeaders() }),
      axios.get('/api/v1/domain/certs', { headers: authHeaders() }),
      axios.get('/api/v1/domain/accounts', { headers: authHeaders() }),
      axios.get('/api/v1/alert/alerts', { headers: authHeaders() })
    ])

    domains.value = domainRes.status === 'fulfilled' ? safeArray(domainRes.value) : []
    certs.value = certRes.status === 'fulfilled' ? safeArray(certRes.value) : []
    accounts.value = accountRes.status === 'fulfilled' ? safeArray(accountRes.value) : []
    alerts.value = alertRes.status === 'fulfilled' ? safeArray(alertRes.value) : []

    stats.domainTotal = domains.value.length
    stats.domainHealthy = domains.value.filter((item) => normalizeText(item.health_status) === 'healthy').length
    stats.domainWarning = domains.value.filter((item) => normalizeText(item.health_status) === 'warning').length
    stats.domainCritical = domains.value.filter((item) => normalizeText(item.health_status) === 'critical').length
    stats.certTotal = certs.value.length
    stats.certExpiringSoon = certs.value.filter((item) => Number(certDays(item)) <= 30).length
    stats.certCriticalSoon = certs.value.filter((item) => Number(certDays(item)) <= 7).length
    stats.accountTotal = accounts.value.length
    stats.pendingRecheck =
      domains.value.filter((item) => isCheckStale(item.last_check_at)).length +
      certs.value.filter((item) => isCheckStale(item.last_check_at)).length

    renderHealthChart()

    const failedCount = [domainRes, certRes, accountRes, alertRes].filter((r) => r.status === 'rejected').length
    if (failedCount > 0) {
      ElMessage.warning(`部分域名中心数据加载失败（${failedCount}项），已展示可用数据`)
    }
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '加载域名监控中心失败'))
  } finally {
    loading.value = false
  }
}

const checkAllDomains = async () => {
  checking.value = true
  try {
    const res = await axios.post('/api/v1/domain/domains/check_all', {}, { headers: authHeaders() })
    const data = res?.data?.data || {}
    ElMessage.success(`域名体检完成：成功 ${data.success || 0} / 失败 ${data.failed || 0}`)
    await refreshAll()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '域名体检失败'))
  } finally {
    checking.value = false
  }
}

const checkAllCerts = async () => {
  checkingCert.value = true
  try {
    const res = await axios.post('/api/v1/domain/certs/check_all', {}, { headers: authHeaders() })
    const data = res?.data?.data || {}
    ElMessage.success(`证书检查完成：成功 ${data.success || 0} / 失败 ${data.failed || 0}`)
    await refreshAll()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '证书检查失败'))
  } finally {
    checkingCert.value = false
  }
}

onMounted(() => {
  refreshAll()
  dayTicker = window.setInterval(() => {
    nowTick.value = Date.now()
  }, 60 * 1000)
  window.addEventListener('resize', resizeChart)
})

const resizeChart = () => {
  if (healthChart) healthChart.resize()
}

onBeforeUnmount(() => {
  if (dayTicker) {
    window.clearInterval(dayTicker)
    dayTicker = null
  }
  window.removeEventListener('resize', resizeChart)
  if (healthChart) {
    healthChart.dispose()
    healthChart = null
  }
})
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 12px; gap: 12px; }
.page-desc { color: var(--muted-text); margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; flex-wrap: wrap; }
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
.chart-box { width: 100%; height: 240px; }
.health-row { display: flex; justify-content: space-between; align-items: center; margin-bottom: 6px; }
.health-row strong { font-size: 15px; }
.mt-12 { margin-top: 12px; }
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
