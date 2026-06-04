<template>
  <div class="hub-container" v-loading="loading">
    <header class="hub-header">
      <div class="hub-title">
        <h2>告警运营台</h2>
        <p>事件发现 → 响应处置 → 静默策略 → 聚合分析 → 复盘改进，完整闭环。</p>
      </div>
      <div class="hub-actions">
        <el-button round icon="Refresh" @click="refreshAll">刷新</el-button>
        <el-button round icon="Share" @click="go('/topology')">服务拓扑</el-button>
        <el-button round icon="Bell" @click="go('/monitor/hosts')">主机监控</el-button>
        <el-button type="primary" round icon="Plus" @click="go('/alert/rules')">告警治理</el-button>
      </div>
    </header>

    <!-- Ops Summary -->
    <div class="hub-summary ops-summary">
      <div class="summary-item" :class="{ 'has-alert': stats.alertOpen > 0 }" @click="activePanel = 'alerts'">
        <div class="summary-label">待处理</div>
        <div class="summary-value" :class="stats.alertOpen > 0 ? 'danger' : 'success'">{{ stats.alertOpen }}</div>
      </div>
      <div class="summary-item" :class="{ 'has-alert': stats.alertCritical > 0 }" @click="activePanel = 'alerts'; severityFilter = 'critical'">
        <div class="summary-label">Critical</div>
        <div class="summary-value" :class="stats.alertCritical > 0 ? 'danger' : 'success'">{{ stats.alertCritical }}</div>
      </div>
      <div class="summary-item" :class="{ 'has-alert': stats.alertWarning > 0 }" @click="activePanel = 'alerts'; severityFilter = 'warning'">
        <div class="summary-label">Warning</div>
        <div class="summary-value warning">{{ stats.alertWarning }}</div>
      </div>
      <div class="summary-item" @click="activePanel = 'silences'">
        <div class="summary-label">活跃静默</div>
        <div class="summary-value">{{ stats.silenceCount }}</div>
      </div>
      <div class="summary-item" @click="activePanel = 'rules'">
        <div class="summary-label">启用规则</div>
        <div class="summary-value success">{{ stats.rulesEnabled }}</div>
      </div>
      <div class="summary-item" @click="activePanel = 'history'">
        <div class="summary-label">近期恢复</div>
        <div class="summary-value">{{ stats.recentResolved }}</div>
      </div>
    </div>

    <div class="hub-workspace full-width">
      <div class="workspace-content">
        <el-tabs v-model="activePanel" class="hub-tabs" @tab-change="onTabChange">
          <!-- 1. Active Alerts -->
          <el-tab-pane label="活跃事件" name="alerts">
            <div class="pane-header">
              <el-select v-model="severityFilter" placeholder="级别" size="small" style="width:120px" clearable @change="() => {}">
                <el-option label="critical" value="critical" />
                <el-option label="warning" value="warning" />
                <el-option label="info" value="info" />
              </el-select>
              <div class="pane-actions">
                <el-button size="small" @click="go('/alert/events')">全屏管理 →</el-button>
              </div>
            </div>
            <div v-if="filteredAlerts.length === 0 && !loading" class="empty-cta">
              <el-empty description="暂无活跃告警 🎉" :image-size="50">
                <el-button type="primary" size="small" @click="go('/alert/rules')">管理告警规则</el-button>
              </el-empty>
            </div>
            <el-table v-else :data="filteredAlerts" class="hub-table" size="small">
              <el-table-column type="selection" width="40" />
              <el-table-column label="告警名称" min-width="180">
                <template #default="{ row }">
                  <el-button link type="primary" @click="go('/alert/events/detail?id=' + row.id)">{{ row.alert_name || row.rule_name || '-' }}</el-button>
                </template>
              </el-table-column>
              <el-table-column prop="target" label="目标" width="150" />
              <el-table-column label="级别" width="90">
                <template #default="{ row }">
                  <el-tag :type="row.severity === 'critical' ? 'danger' : row.severity === 'warning' ? 'warning' : 'info'" size="small">{{ row.severity }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="触发时间" width="140">
                <template #default="{ row }">{{ fmtTimeAgo(row.fired_at || row.created_at) }}</template>
              </el-table-column>
              <el-table-column label="操作" width="240" fixed="right">
                <template #default="{ row }">
                  <el-button link type="primary" size="small" @click="handleAck(row)">确认</el-button>
                  <el-button link type="success" size="small" @click="handleResolve(row)">恢复</el-button>
                  <el-button link type="warning" size="small" @click="handleCreateWorkorder(row)">转工单</el-button>
                  <el-button link size="small" @click="goAIAnalysis({ type: 'alert', title: row.alert_name || '告警', id: row.id, summary: `告警: ${row.alert_name || row.rule_name} (${row.severity}), 目标: ${row.target}` })">
                    <el-icon><MagicStick /></el-icon>
                  </el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>

          <!-- 2. Alert Rules Summary -->
          <el-tab-pane label="告警规则" name="rules">
            <div class="pane-header">
              <el-input v-model="ruleKeyword" placeholder="搜索规则..." size="small" style="width:200px" clearable />
              <el-button size="small" type="primary" @click="go('/alert/rules')">管理规则 →</el-button>
            </div>
            <el-table :data="filteredRules" class="hub-table" size="small">
              <el-table-column prop="name" label="规则名称" min-width="180" />
              <el-table-column prop="target" label="目标" width="160" />
              <el-table-column prop="metric" label="指标" width="120" />
              <el-table-column label="级别" width="80">
                <template #default="{ row }">
                  <el-tag :type="row.severity === 'critical' ? 'danger' : 'warning'" size="small">{{ row.severity }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="启用" width="70">
                <template #default="{ row }">
                  <el-switch v-model="row.enabled" size="small" @change="toggleRule(row)" />
                </template>
              </el-table-column>
              <el-table-column label="自愈" width="60">
                <template #default="{ row }">
                  <el-tag v-if="row.auto_recover" type="success" size="small" effect="plain">ON</el-tag>
                  <span v-else class="text-muted">-</span>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>

          <!-- 3. Active Silences -->
          <el-tab-pane label="静默策略" name="silences">
            <div class="pane-header">
              <el-button size="small" type="primary" @click="go('/alert/silences')">管理静默 →</el-button>
            </div>
            <div v-if="silences.length === 0 && !loading" class="empty-cta">
              <el-empty description="无活跃静默" :image-size="40" />
            </div>
            <el-table v-else :data="silences" class="hub-table" size="small">
              <el-table-column prop="comment" label="说明" min-width="200">
                <template #default="{ row }">{{ row.comment || row.matchers || '-' }}</template>
              </el-table-column>
              <el-table-column prop="created_by" label="创建人" width="100" />
              <el-table-column label="剩余时间" width="140">
                <template #default="{ row }">
                  <span :class="{ 'text-danger': isSilenceExpiring(row) }">{{ silenceRemaining(row) }}</span>
                </template>
              </el-table-column>
              <el-table-column label="操作" width="80">
                <template #default="{ row }">
                  <el-button link type="danger" size="small" @click="handleDeleteSilence(row)">解除</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>

          <!-- 4. Aggregated Groups -->
          <el-tab-pane label="聚合视图" name="aggregations">
            <div class="pane-header">
              <el-button size="small" type="primary" @click="go('/alert/aggregation')">聚合管理 →</el-button>
            </div>
            <div v-if="aggregations.length === 0 && !loading" class="empty-cta">
              <el-empty description="暂无聚合告警组" :image-size="40">
                <el-button type="primary" size="small" @click="go('/alert/aggregation')">创建聚合规则</el-button>
              </el-empty>
            </div>
            <el-table v-else :data="aggregations" class="hub-table" size="small">
              <el-table-column prop="name" label="聚合名称" min-width="180" />
              <el-table-column label="关联告警数" width="100">
                <template #default="{ row }">{{ row.alert_count || row.count || '-' }}</template>
              </el-table-column>
              <el-table-column label="状态" width="80">
                <template #default="{ row }">
                  <el-tag :type="row.enabled ? 'success' : 'info'" size="small">{{ row.enabled ? '启用' : '停用' }}</el-tag>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>

          <!-- 5. Alert History / Retrospectives -->
          <el-tab-pane label="近期复盘" name="history">
            <div class="pane-header">
              <el-button size="small" type="primary" @click="go('/alert/history')">全部复盘 →</el-button>
            </div>
            <div v-if="alertHistory.length === 0 && !loading" class="empty-cta">
              <el-empty description="暂无已恢复告警记录" :image-size="40" />
            </div>
            <el-table v-else :data="alertHistory" class="hub-table" size="small">
              <el-table-column label="告警" min-width="180">
                <template #default="{ row }">{{ row.rule_name || row.alert_name || '-' }}</template>
              </el-table-column>
              <el-table-column prop="target" label="目标" width="150" />
              <el-table-column label="级别" width="80">
                <template #default="{ row }">
                  <el-tag :type="row.severity === 'critical' ? 'danger' : 'warning'" size="small">{{ row.severity }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="恢复时间" width="140">
                <template #default="{ row }">{{ fmtTimeAgo(row.resolved_at) }}</template>
              </el-table-column>
              <el-table-column label="持续" width="80">
                <template #default="{ row }">{{ durationText(row) }}</template>
              </el-table-column>
            </el-table>
          </el-tab-pane>

          <!-- 域名/证书 -->
          <el-tab-pane label="域名证书" name="domain">
            <el-table :data="domainRisks" class="hub-table" size="small">
              <el-table-column prop="name" label="域名" />
              <el-table-column prop="reason" label="风险" />
              <el-table-column label="剩余天数" width="100">
                <template #default="{ row }">{{ row.days_left || '-' }}</template>
              </el-table-column>
              <el-table-column label="操作" width="80">
                <template #default="{ row }">
                  <el-button link type="primary" size="small" @click="go('/domain/ssl')">处理</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>

          <!-- 通知 & 自愈 -->
          <el-tab-pane label="通知与自愈" name="tools">
            <el-row :gutter="16">
              <el-col :span="12">
                <h4 class="sub-title">通知渠道</h4>
                <el-table :data="channels" size="small">
                  <el-table-column prop="name" label="渠道" />
                  <el-table-column label="状态" width="80">
                    <template #default="{ row }"><el-tag :type="row.enabled ? 'success' : 'info'" size="small">{{ row.enabled ? '启用' : '停用' }}</el-tag></template>
                  </el-table-column>
                </el-table>
              </el-col>
              <el-col :span="12">
                <h4 class="sub-title">故障自愈</h4>
                <el-empty description="自愈日志" :image-size="40">
                  <el-button type="primary" size="small" @click="go('/remediation')">查看日志</el-button>
                </el-empty>
              </el-col>
            </el-row>
          </el-tab-pane>
        </el-tabs>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import './hub-common.css'
import { useAIChat } from '@/composables/useAIChat'

const router = useRouter()
const { goAIAnalysis } = useAIChat()
const loading = ref(false)
const alerts = ref([])
const rules = ref([])
const silences = ref([])
const aggregations = ref([])
const alertHistory = ref([])
const channels = ref([])
const domainRisks = ref([])
const agents = ref([])
const activePanel = ref('alerts')
const severityFilter = ref('')
const ruleKeyword = ref('')

const stats = reactive({
  alertOpen: 0, alertCritical: 0, alertWarning: 0,
  silenceCount: 0, rulesEnabled: 0, rulesTotal: 0, recentResolved: 0,
  domainRisk: 0, agentOnline: 0
})

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })
const go = (path) => router.push(path)

const filteredAlerts = computed(() => {
  let list = alerts.value
  if (severityFilter.value) list = list.filter(a => a.severity === severityFilter.value)
  return list
})

const filteredRules = computed(() => {
  if (!ruleKeyword.value) return rules.value
  const kw = ruleKeyword.value.toLowerCase()
  return rules.value.filter(r => (r.name || '').toLowerCase().includes(kw))
})

const fmtTimeAgo = (val) => {
  if (!val) return '-'
  const diff = Math.floor((Date.now() - new Date(val).getTime()) / 1000)
  if (diff < 60) return '刚刚'
  if (diff < 3600) return `${Math.floor(diff / 60)}m 前`
  if (diff < 86400) return `${Math.floor(diff / 3600)}h 前`
  return `${Math.floor(diff / 86400)}d 前`
}

const silenceRemaining = (row) => {
  if (!row.ends_at) return '-'
  const diff = Math.floor((new Date(row.ends_at).getTime() - Date.now()) / 1000)
  if (diff <= 0) return '已过期'
  if (diff < 3600) return `${Math.floor(diff / 60)}m`
  if (diff < 86400) return `${Math.floor(diff / 3600)}h`
  return `${Math.floor(diff / 86400)}d`
}

const isSilenceExpiring = (row) => {
  if (!row.ends_at) return false
  const diff = Math.floor((new Date(row.ends_at).getTime() - Date.now()) / 1000)
  return diff > 0 && diff < 3600
}

const durationText = (row) => {
  if (!row.fired_at || !row.resolved_at) return '-'
  const diff = Math.floor((new Date(row.resolved_at).getTime() - new Date(row.fired_at).getTime()) / 1000)
  if (diff < 60) return `${diff}s`
  if (diff < 3600) return `${Math.floor(diff / 60)}m`
  return `${Math.floor(diff / 3600)}h`
}

const onTabChange = (tab) => {
  if (tab === 'rules' && rules.value.length === 0) fetchRules()
  if (tab === 'silences' && silences.value.length === 0) fetchSilences()
  if (tab === 'aggregations' && aggregations.value.length === 0) fetchAggregations()
  if (tab === 'history' && alertHistory.value.length === 0) fetchHistory()
}

const refreshAll = async () => {
  loading.value = true
  try {
    const [alt, chn, dom, agt, sil, statsRes] = await Promise.all([
      axios.get('/api/v1/alert/alerts', { headers: authHeaders() }),
      axios.get('/api/v1/notify/channels', { headers: authHeaders() }),
      axios.get('/api/v1/domain/domains', { headers: authHeaders() }),
      axios.get('/api/v1/monitor/agents', { headers: authHeaders() }),
      axios.get('/api/v1/alert/silences', { headers: authHeaders() }).catch(() => ({ data: {} })),
      axios.get('/api/v1/alert/stats', { headers: authHeaders() }).catch(() => ({ data: {} }))
    ])
    alerts.value = (alt.data?.data || []).filter(a => a.status === 'firing')
    channels.value = chn.data?.data || []
    domainRisks.value = (dom.data?.data || []).filter(d => d.health_status !== 'healthy')
    agents.value = agt.data?.data || []
    silences.value = (sil.data?.data || []).filter(s => s.status === 'active' || s.active)

    stats.alertOpen = alerts.value.length
    stats.alertCritical = alerts.value.filter(a => a.severity === 'critical').length
    stats.alertWarning = alerts.value.filter(a => a.severity === 'warning').length
    stats.silenceCount = silences.value.length
    stats.domainRisk = domainRisks.value.length
    stats.agentOnline = agents.value.filter(a => a.status === 'online').length
    if (statsRes.data?.data) {
      stats.rulesEnabled = statsRes.data.data.rules_enabled || statsRes.data.data.active_rules || 0
      stats.rulesTotal = statsRes.data.data.rules_total || 0
    }
  } catch (err) {
    ElMessage.error('加载观测中心失败')
  } finally {
    loading.value = false
  }
}

const fetchRules = async () => {
  try {
    const res = await axios.get('/api/v1/alert/rules', { headers: authHeaders() })
    rules.value = res.data?.data || []
    stats.rulesEnabled = rules.value.filter(r => r.enabled).length
    stats.rulesTotal = rules.value.length
  } catch (e) { /* silent */ }
}

const fetchSilences = async () => {
  try {
    const res = await axios.get('/api/v1/alert/silences', { headers: authHeaders() })
    silences.value = (res.data?.data || []).filter(s => s.status === 'active' || s.active)
    stats.silenceCount = silences.value.length
  } catch (e) { /* silent */ }
}

const fetchAggregations = async () => {
  try {
    const res = await axios.get('/api/v1/alert/aggregations', { headers: authHeaders() })
    aggregations.value = res.data?.data || []
  } catch (e) { /* silent */ }
}

const fetchHistory = async () => {
  try {
    const res = await axios.get('/api/v1/alert/history', { headers: authHeaders() })
    alertHistory.value = (res.data?.data || []).slice(0, 20)
    stats.recentResolved = alertHistory.value.length
  } catch (e) { /* silent */ }
}

const toggleRule = async (row) => {
  try {
    await axios.put(`/api/v1/alert/rules/${row.id}`, { enabled: row.enabled }, { headers: authHeaders() })
  } catch (err) {
    row.enabled = !row.enabled
    ElMessage.error('切换失败')
  }
}

const handleAck = async (row) => {
  try {
    await axios.post(`/api/v1/alert/alerts/${row.id}/ack`, {}, { headers: authHeaders() })
    ElMessage.success('已确认')
    refreshAll()
  } catch (err) {
    ElMessage.error('确认失败')
  }
}

const handleResolve = async (row) => {
  try {
    await axios.post(`/api/v1/alert/alerts/${row.id}/resolve`, {}, { headers: authHeaders() })
    ElMessage.success('已恢复')
    refreshAll()
  } catch (err) {
    ElMessage.error('恢复失败')
  }
}

const handleCreateWorkorder = async (row) => {
  try {
    const res = await axios.post(`/api/v1/alert/alerts/${row.id}/create-workorder`, {}, { headers: authHeaders() })
    ElMessage.success('已转工单: ' + (res.data?.data?.id || ''))
  } catch (err) {
    ElMessage.error('转工单失败')
  }
}

const handleDeleteSilence = async (row) => {
  try {
    await ElMessageBox.confirm('确认解除该静默策略？', '提示', { type: 'warning' })
    await axios.delete(`/api/v1/alert/silences/${row.id}`, { headers: authHeaders() })
    ElMessage.success('已解除')
    fetchSilences()
  } catch (e) { /* cancel */ }
}

onMounted(refreshAll)
</script>

<style scoped>
.full-width { grid-template-columns: 1fr; }
.empty-cta { padding: 30px 0; }

.ops-summary { grid-template-columns: repeat(6, 1fr); }
.summary-item.has-alert { background: rgba(239, 68, 68, 0.04); }

.pane-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.pane-actions { display: flex; gap: 8px; }
.sub-title { font-size: 14px; font-weight: 700; margin: 0 0 12px 0; color: var(--el-text-color-primary); }
.text-muted { color: var(--el-text-color-placeholder); }
.text-danger { color: var(--el-color-danger); font-weight: 600; }

@media (max-width: 1200px) {
  .ops-summary { grid-template-columns: repeat(3, 1fr); }
}
</style>
