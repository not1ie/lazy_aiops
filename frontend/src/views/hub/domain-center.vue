<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>域名监控中心</h2>
        <p class="page-desc">集中展示域名可用性、证书到期风险与处置入口，避免分散排查。</p>
      </div>
      <div class="page-actions">
        <el-button :loading="checking" type="primary" plain @click="checkAllDomains">域名一键体检</el-button>
        <el-button :loading="checkingCert" plain @click="checkAllCerts">证书批量检查</el-button>
        <el-button :loading="loading" icon="Refresh" @click="refreshAll">刷新</el-button>
      </div>
    </div>

    <div class="workbench-toolbar">
      <div class="workbench-toolbar-left">
        <span class="workbench-toolbar-label">场景工作台</span>
        <el-check-tag checked @click="go('/domain/center')">域名监控中心</el-check-tag>
        <el-check-tag :checked="false" @click="go('/monitor/center')">监控告警中心</el-check-tag>
        <el-check-tag :checked="false" @click="go('/domain/ssl')">域名与证书</el-check-tag>
      </div>
      <div class="workbench-toolbar-right">
        <el-tag type="danger" effect="light">高危 {{ riskCriticalCount }}</el-tag>
        <el-tag type="warning" effect="light">待复检 {{ riskStaleCount }}</el-tag>
        <el-tag type="info" effect="light">证书≤7天 {{ stats.certCriticalSoon }}</el-tag>
        <el-button link type="primary" @click="focusDomainPanel('domains')">域名体检</el-button>
        <el-button link type="warning" @click="focusDomainPanel('certs')">证书处置</el-button>
      </div>
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
          <template #header>
            <div class="risk-header">
              <span>风险清单（可直接处置）</span>
              <div class="risk-actions">
                <el-tag type="danger" effect="light">高危 {{ riskCriticalCount }}</el-tag>
                <el-tag type="warning" effect="light">待复检 {{ riskStaleCount }}</el-tag>
                <el-button size="small" type="warning" plain :loading="riskBatching" :disabled="!selectedRiskCount" @click="batchHandleSelectedRisk">批量复检已选</el-button>
                <el-button size="small" plain :loading="riskBatching" :disabled="!riskCriticalCount" @click="batchHandleCriticalRisk">处置高危</el-button>
                <el-button size="small" plain :loading="riskBatching" :disabled="!riskStaleCount" @click="batchHandleStaleRisk">复检过期</el-button>
              </div>
            </div>
          </template>
          <el-table
            ref="riskTableRef"
            :fit="true"
            :data="riskRows"
            size="small"
            max-height="470"
            empty-text="暂无风险项"
            @selection-change="onRiskSelectionChange"
          >
            <el-table-column type="selection" width="46" />
            <el-table-column prop="type" label="类型" width="90" />
            <el-table-column prop="name" label="对象" min-width="170" />
            <el-table-column prop="reason" label="风险" min-width="140">
              <template #default="{ row }">
                <el-tag :type="row.level">{{ row.reason }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="detail" label="详情" min-width="180" show-overflow-tooltip />
            <el-table-column label="检查时效" width="110">
              <template #default="{ row }">
                <el-tag :type="row.stale ? 'warning' : 'success'">{{ row.stale ? '待复检' : '及时' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" min-width="150">
              <template #default="{ row }">
                <el-button link type="primary" @click="handleRiskAction(row)">复检</el-button>
                <el-button link type="warning" @click="openDispositionDrawer(row)">处置</el-button>
                <el-button link @click="go(row.path)">去处理</el-button>
              </template>
            </el-table-column>
          </el-table>
          <div class="risk-group-list">
            <el-tag
              v-for="group in riskReasonGroups"
              :key="group.reason"
              :type="group.level"
              effect="plain"
              class="risk-group-tag"
              @click="batchHandleRiskGroup(group)"
            >
              {{ group.reason }} · {{ group.count }}
            </el-tag>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-card class="integration-card">
      <template #header>
        <div class="integration-header">
          <div class="integration-title-wrap">
            <span>域名融合视图</span>
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
              placeholder="筛选域名、证书、账号、告警..."
            />
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
                <el-button size="small" link type="warning" @click="openDispositionDrawer({ kind: 'domain', domain: row.domain, name: row.domain, source: row })">处置</el-button>
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
                <el-button size="small" link type="warning" @click="openDispositionDrawer({ kind: 'cert', id: row.id, domain: row.domain, name: row.domain, source: row })">处置</el-button>
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
                <el-button size="small" link type="warning" @click="openDispositionDrawer({ kind: 'account', id: row.id, name: row.name, source: row })">处置</el-button>
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
                  <el-button size="small" link type="warning" @click="openDispositionDrawer({ kind: 'alert', id: row.id, name: row.rule_name || row.target || row.id, source: row })">处置</el-button>
                  <el-button size="small" link @click="go('/alert/events')">详情</el-button>
                </el-space>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <el-drawer
      v-model="dispositionDrawerVisible"
      title="域名监控处置台"
      size="460px"
      :destroy-on-close="false"
    >
      <template v-if="dispositionTarget">
        <el-descriptions :column="1" border size="small" class="disposition-desc">
          <el-descriptions-item label="对象类型">{{ dispositionTypeText }}</el-descriptions-item>
          <el-descriptions-item label="对象">{{ dispositionTarget.name || dispositionTarget.domain || '-' }}</el-descriptions-item>
          <el-descriptions-item label="风险说明">{{ dispositionReasonText }}</el-descriptions-item>
          <el-descriptions-item label="建议动作">{{ dispositionSuggestionText }}</el-descriptions-item>
        </el-descriptions>

        <div class="disposition-actions">
          <el-button type="primary" :loading="dispositionExecuting" @click="runDispositionPrimaryAction">
            {{ dispositionPrimaryLabel }}
          </el-button>
          <el-button :loading="dispositionExecuting" @click="openDispositionDetail">进入详情页</el-button>
          <el-button
            v-if="dispositionTarget.kind === 'alert'"
            type="warning"
            plain
            :loading="dispositionExecuting"
            @click="runDispositionSecondaryAction"
          >
            一键静默1h
          </el-button>
        </div>
      </template>
      <el-empty v-else description="请选择需要处置的对象" />
    </el-drawer>
  </el-card>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'
import * as echarts from 'echarts'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getErrorMessage, isCancelError } from '@/utils/error'

const router = useRouter()
const loading = ref(false)
const checking = ref(false)
const checkingCert = ref(false)
const riskBatching = ref(false)
const domains = ref([])
const certs = ref([])
const accounts = ref([])
const alerts = ref([])
const nowTick = ref(Date.now())
const riskTableRef = ref(null)
const selectedRiskRows = ref([])
const dispositionDrawerVisible = ref(false)
const dispositionTarget = ref(null)
const dispositionExecuting = ref(false)
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

const panelRouteMap = {
  domains: '/domain/ssl',
  certs: '/domain/ssl',
  accounts: '/domain/ssl',
  alerts: '/alert/events'
}

const dispositionTypeText = computed(() => {
  const kind = dispositionTarget.value?.kind
  if (kind === 'domain') return '域名'
  if (kind === 'cert') return '证书'
  if (kind === 'account') return '云账号'
  if (kind === 'alert') return '告警'
  return '未知'
})

const dispositionReasonText = computed(() => {
  const target = dispositionTarget.value
  if (!target) return '-'
  return target.reason || target.detail || target.source?.rule_name || target.source?.metric || '待人工确认'
})

const dispositionSuggestionText = computed(() => {
  const kind = dispositionTarget.value?.kind
  if (kind === 'domain') return '先执行域名体检，确认 HTTP/DNS 状态后转到证书详情处置'
  if (kind === 'cert') return '立即执行证书检查，确认到期时间并安排续期计划'
  if (kind === 'account') return '先同步账号状态，检查 API 可用性与权限是否正常'
  if (kind === 'alert') return '先确认告警，再按需恢复或静默，避免重复噪声'
  return '建议先执行检测，再进入详情页处理'
})

const dispositionPrimaryLabel = computed(() => {
  const target = dispositionTarget.value
  if (!target) return '执行处置'
  if (target.kind === 'domain') return '执行域名体检'
  if (target.kind === 'cert') return '执行证书检查'
  if (target.kind === 'account') return '同步云账号'
  if (target.kind === 'alert') {
    const status = Number(target.source?.status)
    if (status === 0) return '确认告警'
    return '恢复告警'
  }
  return '执行处置'
})

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
    const domainCheckAt = item.last_check_at || item.updated_at
    if (health === 'warning' || health === 'critical') {
      rows.push({
        key: `domain-health-${item.domain}`,
        kind: 'domain',
        domain: item.domain,
        type: '域名',
        name: item.domain,
        reason: health === 'critical' ? '可用性故障' : '可用性预警',
        detail: `HTTP ${item.http_status_code || '-'} / ${item.response_time_ms || '-'}ms`,
        level: health === 'critical' ? 'danger' : 'warning',
        checkAt: domainCheckAt,
        stale: isCheckStale(domainCheckAt),
        path: '/domain/ssl'
      })
    }

    const certRemain = Number(sslDays(item))
    if (!Number.isNaN(certRemain) && certRemain <= 30) {
      rows.push({
        key: `domain-cert-${item.domain}`,
        kind: 'domain',
        domain: item.domain,
        type: '域名证书',
        name: item.domain,
        reason: certRemain <= 7 ? '即将过期' : '临近到期',
        detail: `剩余 ${certRemain} 天`,
        level: certRemain <= 7 ? 'danger' : 'warning',
        checkAt: domainCheckAt,
        stale: isCheckStale(domainCheckAt),
        path: '/domain/ssl'
      })
    }

    if (isCheckStale(item.last_check_at)) {
      rows.push({
        key: `domain-stale-${item.domain}`,
        kind: 'domain',
        domain: item.domain,
        type: '域名巡检',
        name: item.domain,
        reason: '检查过期',
        detail: `上次检查 ${formatCheckFreshness(item.last_check_at)}`,
        level: 'warning',
        checkAt: item.last_check_at,
        stale: true,
        path: '/domain/ssl'
      })
    }
  })

  certs.value.forEach((item) => {
    const certCheckAt = item.last_check_at || item.updated_at
    const days = Number(certDays(item))
    if (days <= 30) {
      rows.push({
        key: `cert-expire-${item.id}`,
        kind: 'cert',
        id: item.id,
        domain: item.domain,
        type: '证书',
        name: item.domain,
        reason: days <= 7 ? '即将过期' : '临近到期',
        detail: `剩余 ${days} 天`,
        level: days <= 7 ? 'danger' : 'warning',
        checkAt: certCheckAt,
        stale: isCheckStale(certCheckAt),
        path: '/domain/ssl'
      })
    }

    if (isCheckStale(item.last_check_at)) {
      rows.push({
        key: `cert-stale-${item.id}`,
        kind: 'cert',
        id: item.id,
        domain: item.domain,
        type: '证书巡检',
        name: item.domain,
        reason: '检查过期',
        detail: `上次检查 ${formatCheckFreshness(item.last_check_at)}`,
        level: 'warning',
        checkAt: item.last_check_at,
        stale: true,
        path: '/domain/ssl'
      })
    }
  })

  const levelWeight = { danger: 2, warning: 1, info: 0 }
  return rows
    .sort((a, b) => {
      const levelDiff = (levelWeight[b.level] || 0) - (levelWeight[a.level] || 0)
      if (levelDiff !== 0) return levelDiff
      return Number(b.stale) - Number(a.stale)
    })
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
const riskCriticalCount = computed(() => riskRows.value.filter((item) => item.level === 'danger').length)
const riskStaleCount = computed(() => riskRows.value.filter((item) => item.stale).length)
const selectedRiskCount = computed(() => selectedRiskRows.value.length)
const riskReasonGroups = computed(() => {
  const map = new Map()
  riskRows.value.forEach((item) => {
    const key = item.reason || '未分类'
    const current = map.get(key) || { reason: key, count: 0, level: item.level, rows: [] }
    current.count += 1
    if (item.level === 'danger') current.level = 'danger'
    current.rows.push(item)
    map.set(key, current)
  })
  return [...map.values()].sort((a, b) => b.count - a.count).slice(0, 8)
})

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

const panelOptions = computed(() => [
  { name: 'domains', label: '域名体检', count: domains.value.length },
  { name: 'certs', label: 'SSL证书', count: certs.value.length },
  { name: 'accounts', label: '云账号', count: accounts.value.length },
  { name: 'alerts', label: '关联告警', count: domainAlerts.value.length }
])

const activePanelMeta = computed(
  () => panelOptions.value.find((item) => item.name === activePanel.value) || panelOptions.value[0] || { label: '-', count: 0 }
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

const focusDomainPanel = (panel) => {
  if (!panel || panel === activePanel.value) return
  activePanel.value = panel
}

const resolveDispositionPath = (target) => {
  if (!target) return '/domain/ssl'
  if (target.kind === 'alert') return '/alert/events'
  if (target.path) return target.path
  return '/domain/ssl'
}

const openDispositionDrawer = (target) => {
  dispositionTarget.value = target ? { ...target } : null
  dispositionDrawerVisible.value = Boolean(target)
}

const openDispositionDetail = () => {
  go(resolveDispositionPath(dispositionTarget.value))
}

const onRiskSelectionChange = (rows) => {
  selectedRiskRows.value = Array.isArray(rows) ? rows : []
}

const recalcStats = () => {
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
}

const resolveCheckTime = (value) => {
  if (!value) return new Date().toISOString()
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) return new Date().toISOString()
  return parsed.toISOString()
}

const resolveCertStatusByDays = (days) => {
  const num = Number(days)
  if (!Number.isFinite(num)) return 1
  if (num <= 0) return 0
  if (num <= 30) return 2
  return 1
}

const applyRiskCheckPatch = (result) => {
  if (!result) return
  const checkedAt = resolveCheckTime(result.checkedAt || result.payload?.checked_at || result.payload?.last_check_at)
  nowTick.value = Date.now()

  if (result.kind === 'domain') {
    const domain = normalizeText(result.domain || result.payload?.domain)
    if (!domain) return
    domains.value = domains.value.map((item) => {
      if (normalizeText(item.domain) !== domain) return item
      return {
        ...item,
        dns_resolved: result.payload?.dns_resolved ?? item.dns_resolved,
        http_status_code: result.payload?.http_status_code ?? item.http_status_code,
        response_time_ms: result.payload?.response_time_ms ?? item.response_time_ms,
        ssl_days_to_expire: result.payload?.ssl_days_to_expire ?? item.ssl_days_to_expire,
        health_status: result.payload?.health_status || item.health_status,
        last_check_at: checkedAt
      }
    })
  }

  if (result.kind === 'cert') {
    const certID = String(result.id || result.payload?.id || '').trim()
    const certDomain = normalizeText(result.domain || result.payload?.domain)
    const nextDays = Number(result.payload?.days_to_expire)
    const nextStatus = result.payload?.status ?? resolveCertStatusByDays(nextDays)

    certs.value = certs.value.map((item) => {
      const idMatched = certID && String(item.id || '').trim() === certID
      const domainMatched = certDomain && normalizeText(item.domain) === certDomain
      if (!idMatched && !domainMatched) return item
      return {
        ...item,
        issuer: result.payload?.issuer || item.issuer,
        subject: result.payload?.subject || item.subject,
        sans: result.payload?.sans || item.sans,
        not_before: result.payload?.not_before ?? item.not_before,
        not_after: result.payload?.not_after ?? item.not_after,
        days_to_expire: Number.isFinite(nextDays) ? nextDays : item.days_to_expire,
        serial_number: result.payload?.serial_number || item.serial_number,
        status: nextStatus,
        last_check_at: checkedAt
      }
    })

    domains.value = domains.value.map((item) => {
      if (certDomain && normalizeText(item.domain) !== certDomain) return item
      return {
        ...item,
        ssl_days_to_expire: Number.isFinite(nextDays) ? nextDays : item.ssl_days_to_expire,
        last_check_at: checkedAt
      }
    })
  }

  recalcStats()
}

const scheduleConsistencyRefresh = () => {
  window.setTimeout(() => {
    refreshAll({ silent: true })
  }, 450)
}

const doRiskCheck = async (row) => {
  const domain = row.domain || row.name
  if (row.kind === 'cert' && row.id) {
    const res = await axios.post(`/api/v1/domain/certs/${row.id}/check`, {}, { headers: authHeaders() })
    return {
      kind: 'cert',
      id: row.id,
      domain,
      payload: res?.data?.data || {},
      checkedAt: res?.data?.data?.checked_at || res?.data?.data?.last_check_at || new Date().toISOString()
    }
  }
  if (!domain) {
    return Promise.reject(new Error('缺少域名信息'))
  }
  const res = await axios.post('/api/v1/domain/domains/check', { domain }, { headers: authHeaders() })
  return {
    kind: 'domain',
    domain,
    payload: res?.data?.data || {},
    checkedAt: res?.data?.data?.checked_at || new Date().toISOString()
  }
}

const runBatchRiskAction = async (rows, title, message) => {
  const uniqueRows = []
  const seen = new Set()
  rows.forEach((item) => {
    const key = item.key || `${item.kind}-${item.id || item.domain || item.name}`
    if (seen.has(key)) return
    seen.add(key)
    uniqueRows.push(item)
  })
  if (!uniqueRows.length) {
    ElMessage.info('没有可处置项')
    return
  }

  riskBatching.value = true
  try {
    await ElMessageBox.confirm(message, title, { type: 'warning' })
    const settled = await Promise.allSettled(uniqueRows.map((item) => doRiskCheck(item)))
    const success = settled.filter((item) => item.status === 'fulfilled').length
    const fail = settled.length - success
    settled.forEach((item) => {
      if (item.status === 'fulfilled') applyRiskCheckPatch(item.value)
    })
    ElMessage.success(`批量处置完成：成功 ${success}，失败 ${fail}`)
    selectedRiskRows.value = []
    if (riskTableRef.value?.clearSelection) riskTableRef.value.clearSelection()
    scheduleConsistencyRefresh()
  } catch (err) {
    if (!isCancelError(err)) {
      ElMessage.error(getErrorMessage(err, '批量处置失败'))
    }
  } finally {
    riskBatching.value = false
  }
}

const batchHandleSelectedRisk = async () => {
  await runBatchRiskAction(
    selectedRiskRows.value,
    '批量复检',
    `确认复检已选择的 ${selectedRiskRows.value.length} 个风险对象吗？`
  )
}

const batchHandleCriticalRisk = async () => {
  const rows = riskRows.value.filter((item) => item.level === 'danger')
  await runBatchRiskAction(rows, '处置高危风险', `确认对 ${rows.length} 个高危风险对象执行复检吗？`)
}

const batchHandleStaleRisk = async () => {
  const rows = riskRows.value.filter((item) => item.stale)
  await runBatchRiskAction(rows, '复检过期对象', `确认对 ${rows.length} 个待复检对象执行复检吗？`)
}

const batchHandleRiskGroup = async (group) => {
  const rows = Array.isArray(group?.rows) ? group.rows : []
  await runBatchRiskAction(rows, `批量处置：${group?.reason || '风险组'}`, `确认对「${group?.reason || '风险组'}」共 ${rows.length} 项执行复检吗？`)
}

const checkDomainRow = async (row) => {
  try {
    const result = await doRiskCheck({ kind: 'domain', domain: row.domain })
    applyRiskCheckPatch(result)
    ElMessage.success(`域名 ${row.domain} 已触发体检`)
    scheduleConsistencyRefresh()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '域名体检失败'))
  }
}

const checkCertRow = async (row) => {
  try {
    const result = await doRiskCheck({ kind: 'cert', id: row.id, domain: row.domain })
    applyRiskCheckPatch(result)
    ElMessage.success(`证书 ${row.domain} 已触发检查`)
    scheduleConsistencyRefresh()
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

const runDispositionPrimaryAction = async () => {
  const target = dispositionTarget.value
  if (!target) return
  dispositionExecuting.value = true
  try {
    if (target.kind === 'cert') {
      await checkCertRow({ id: target.id, domain: target.domain || target.name })
    } else if (target.kind === 'domain') {
      await checkDomainRow({ domain: target.domain || target.name })
    } else if (target.kind === 'account') {
      await syncAccount(target.source || { id: target.id, name: target.name })
    } else if (target.kind === 'alert') {
      const status = Number(target.source?.status)
      if (status === 0) await ackAlert(target.source)
      else await resolveAlert(target.source)
    }
  } finally {
    dispositionExecuting.value = false
  }
}

const runDispositionSecondaryAction = async () => {
  const target = dispositionTarget.value
  if (!target || target.kind !== 'alert') return
  dispositionExecuting.value = true
  try {
    await silenceAlert(target.source)
  } finally {
    dispositionExecuting.value = false
  }
}

const refreshAll = async ({ silent = false } = {}) => {
  if (!silent) loading.value = true
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

    recalcStats()

    renderHealthChart()

    const failedCount = [domainRes, certRes, accountRes, alertRes].filter((r) => r.status === 'rejected').length
    if (failedCount > 0) {
      ElMessage.warning(`部分域名中心数据加载失败（${failedCount}项），已展示可用数据`)
    }
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '加载域名监控中心失败'))
  } finally {
    if (!silent) loading.value = false
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
.workbench-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 12px;
  background: color-mix(in srgb, var(--card-bg) 86%, #ffffff 14%);
  padding: 10px 12px;
  margin-bottom: 12px;
}
.workbench-toolbar-left,
.workbench-toolbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.workbench-toolbar-label {
  font-size: 12px;
  color: var(--muted-text);
  margin-right: 4px;
}
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

.risk-group-list {
  margin-top: 10px;
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.risk-group-tag {
  cursor: pointer;
}

.disposition-desc {
  margin-bottom: 14px;
}

.disposition-actions {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
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

.integration-tabs :deep(.el-tabs__header) {
  margin-bottom: 10px;
}

@media (max-width: 1100px) {
  .workbench-toolbar {
    align-items: flex-start;
    flex-direction: column;
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

  .risk-header {
    align-items: flex-start;
    flex-direction: column;
  }

  .risk-actions {
    width: 100%;
  }
}
</style>
