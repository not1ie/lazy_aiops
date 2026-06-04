<template>
  <div class="dash-wrapper" v-loading="loading">
    <div class="dash-header">
      <div>
        <h1>Overview</h1>
        <p>Real-time insights and operations across your infrastructure.</p>
      </div>
      <div class="dash-actions">
        <el-tag v-if="dataAge" type="info" size="small" effect="plain">{{ dataAge }}</el-tag>
        <el-switch v-model="autoRefresh" size="small" active-text="自动刷新" style="--el-switch-on-color: #10b981" />
        <el-button icon="Refresh" circle @click="fetchData"></el-button>
      </div>
    </div>

    <!-- KPI Cards -->
    <div class="kpi-grid">
      <el-card class="kpi-card" shadow="never" @click="go('/asset')">
        <div class="kpi-top">
          <div class="kpi-icon"><el-icon class="text-green"><Monitor /></el-icon></div>
          <div class="kpi-info">
            <span class="kpi-title">Hosts Online</span>
            <div class="kpi-val">{{ summary.hostOnline }}<span class="kpi-unit">/{{ summary.hostTotal }}</span></div>
          </div>
        </div>
        <div class="kpi-trend" :class="summary.hostOffline > 0 ? 'text-red' : 'text-green'">
          {{ summary.hostOffline > 0 ? `${summary.hostOffline} offline` : 'All healthy' }}
          <span v-if="summary.hostStale > 0" class="text-orange"> &bull; {{ summary.hostStale }} stale</span>
        </div>
        <div class="kpi-sparkline"><svg viewBox="0 0 100 20" preserveAspectRatio="none"><path :d="randomSpark(0)" fill="none" :stroke="summary.hostOffline > 0 ? '#ef4444' : '#10b981'" stroke-width="2" vector-effect="non-scaling-stroke"/></svg></div>
      </el-card>

      <el-card class="kpi-card" shadow="never" @click="go('/monitor')">
        <div class="kpi-top">
          <div class="kpi-icon bg-red-light"><el-icon class="text-red"><WarningFilled /></el-icon></div>
          <div class="kpi-info">
            <span class="kpi-title">Active Alerts</span>
            <div class="kpi-val">{{ summary.alertOpen }}<span class="kpi-unit">/{{ summary.alertTotal }}</span></div>
          </div>
        </div>
        <div class="kpi-trend" :class="summary.alertOpen > 0 ? 'text-red' : 'text-green'">
          {{ summary.alertOpen > 0 ? `${summary.alertOpen} need attention` : 'All clear' }}
        </div>
        <div class="kpi-sparkline"><svg viewBox="0 0 100 20" preserveAspectRatio="none"><path :d="randomSpark(1)" fill="none" :stroke="summary.alertOpen > 3 ? '#ef4444' : '#f59e0b'" stroke-width="2" vector-effect="non-scaling-stroke"/></svg></div>
      </el-card>

      <el-card class="kpi-card" shadow="never" @click="go('/delivery')">
        <div class="kpi-top">
          <div class="kpi-icon bg-purple-light"><el-icon class="text-purple"><Clock /></el-icon></div>
          <div class="kpi-info">
            <span class="kpi-title">Trust Score</span>
            <div class="kpi-val">{{ quality.trustScore }}<span class="kpi-unit">/100</span></div>
          </div>
        </div>
        <div class="kpi-trend" :class="quality.trustScore >= 80 ? 'text-green' : quality.trustScore >= 50 ? 'text-orange' : 'text-red'">
          {{ quality.trustGrade }} &bull; {{ quality.summary }}
        </div>
        <div class="kpi-sparkline"><svg viewBox="0 0 100 20" preserveAspectRatio="none"><path :d="randomSpark(2)" fill="none" :stroke="quality.trustScore >= 80 ? '#10b981' : '#f59e0b'" stroke-width="2" vector-effect="non-scaling-stroke"/></svg></div>
      </el-card>

      <el-card class="kpi-card" shadow="never" @click="go('/delivery')">
        <div class="kpi-top">
          <div class="kpi-icon bg-orange-light"><el-icon class="text-orange"><BellFilled /></el-icon></div>
          <div class="kpi-info">
            <span class="kpi-title">Pending Tickets</span>
            <div class="kpi-val">{{ pendingWorkorders }}</div>
          </div>
        </div>
        <div class="kpi-trend" :class="pendingWorkorders > 0 ? 'text-orange' : 'text-green'">
          {{ pendingWorkorders > 0 ? `${pendingWorkorders} pending` : 'All resolved' }}
        </div>
        <div class="kpi-sparkline"><svg viewBox="0 0 100 20" preserveAspectRatio="none"><path :d="randomSpark(3)" fill="none" :stroke="pendingWorkorders > 0 ? '#f59e0b' : '#10b981'" stroke-width="2" vector-effect="non-scaling-stroke"/></svg></div>
      </el-card>
    </div>

    <div class="main-grid">
      <div class="main-left">
        <el-card class="overview-card" shadow="never">
          <div class="card-title-row">
            <h3>Infrastructure Overview <el-icon class="info-icon"><InfoFilled /></el-icon></h3>
            <a href="#" class="view-all" @click.prevent="go('/asset')">View all</a>
          </div>
          <div class="infra-stats">
            <div class="infra-stat" @click="go('/k8s')">
              <div class="label">K8s Clusters</div>
              <div class="val">{{ summary.k8sTotal }}</div>
              <div class="status"><span class="dot" :class="summary.k8sUnhealthy > 0 ? 'bg-red' : 'bg-green'"></span> {{ summary.k8sHealthy }} healthy{{ summary.k8sUnhealthy > 0 ? `, ${summary.k8sUnhealthy} unhealthy` : '' }}</div>
            </div>
            <div class="infra-stat" @click="go('/host')">
              <div class="label">Host Nodes</div>
              <div class="val">{{ summary.hostTotal }}</div>
              <div class="status"><span class="dot" :class="summary.hostOffline > 0 ? 'bg-orange' : 'bg-green'"></span> {{ summary.hostOnline }} online</div>
            </div>
            <div class="infra-stat" @click="go('/docker')">
              <div class="label">Docker Hosts</div>
              <div class="val">{{ summary.dockerTotal }}</div>
              <div class="status"><span class="dot" :class="summary.dockerOffline > 0 ? 'bg-orange' : 'bg-green'"></span> {{ summary.dockerOnline }} online</div>
            </div>
            <div class="infra-stat" @click="go('/jump/assets')">
              <div class="label">Jump Assets</div>
              <div class="val">{{ snapshots.jump_assets?.length || 0 }}</div>
              <div class="status"><span class="dot bg-green"></span> Synced</div>
            </div>
          </div>
          <!-- Metric history chart -->
          <div class="fake-area-chart" v-if="metricHistory.length > 0">
            <svg viewBox="0 0 400 100" preserveAspectRatio="none">
              <defs>
                <linearGradient id="areaGradient" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stop-color="rgba(0, 113, 227, 0.2)" />
                  <stop offset="100%" stop-color="rgba(0, 113, 227, 0)" />
                </linearGradient>
              </defs>
              <path :d="metricAreaPath" fill="url(#areaGradient)" />
              <path :d="metricLinePath" fill="none" stroke="#0071e3" stroke-width="2" vector-effect="non-scaling-stroke" />
            </svg>
            <div class="chart-labels">
              <span v-for="(m, i) in metricLabels" :key="i">{{ m }}</span>
            </div>
          </div>
          <el-empty v-else description="Waiting for metric data..." :image-size="60" />
        </el-card>

        <div class="sub-grid">
          <!-- Real alerts from API -->
          <el-card class="sub-card" shadow="never">
            <div class="card-title-row"><h3>Recent Alerts</h3><a href="#" class="view-all" @click.prevent="go('/alert/events')">Details</a></div>
            <div v-if="recentAlerts.length === 0" class="empty-hint">No active alerts — all clear 🎉</div>
            <div class="incident-item" v-for="inc in recentAlerts" :key="inc.id" @click="go('/alert/events/detail')">
              <el-icon class="inc-icon" :class="inc._iconColor"><WarningFilled /></el-icon>
              <div class="inc-content">
                <div class="inc-title">{{ inc.alert_name || inc.rule_name || inc.severity }}
                  <el-tag size="small" :type="inc._severityType" class="flat-tag">{{ inc.severity }}</el-tag>
                </div>
                <div class="inc-meta">{{ inc.target || inc.source || '-' }} &bull; {{ fmtTime(inc.fired_at || inc.created_at) }}</div>
              </div>
            </div>
          </el-card>

          <!-- Top Issues: most alerted + recent AIOps -->
          <el-card class="sub-card" shadow="never">
            <div class="card-title-row"><h3>Top Issues</h3><a href="#" class="view-all" @click.prevent="go('/ai/ops')">AIOps</a></div>
            <div v-if="topTargets.length === 0 && recentAIOps.length === 0" class="empty-hint">No issues detected</div>
            <div class="issue-item" v-for="t in topTargets" :key="t.target" @click="go('/monitor')">
              <span class="issue-rank" :class="'rank-' + t.severity">{{ t.count }}</span>
              <div class="issue-info">
                <div class="issue-target">{{ t.target }}</div>
                <div class="issue-meta">{{ t.count }} alerts</div>
              </div>
            </div>
            <div class="issue-item" v-for="inc in recentAIOps" :key="inc.incident_id" @click="go('/ai/ops')">
              <span class="issue-rank" :class="'rank-' + (inc.risk_level || 'medium')">{{ inc.status }}</span>
              <div class="issue-info">
                <div class="issue-target">{{ inc.title || inc.query?.slice(0, 40) }}</div>
                <div class="issue-meta">AIOps &bull; {{ fmtTimeAgo(inc.created_at) }}</div>
              </div>
            </div>
          </el-card>
        </div>
      </div>

      <!-- Right sidebar -->
      <div class="main-right">
        <!-- Action Hints from quality assessment -->
        <el-card class="side-card ai-copilot-card" shadow="never">
          <div class="card-title-row">
            <h3 class="copilot-title"><el-icon><MagicStick /></el-icon> AIOps Copilot</h3>
          </div>
          <div v-if="validActionHints.length > 0">
            <div class="ai-greeting" v-for="hint in validActionHints.slice(0, 3)" :key="hint.key" style="margin-bottom: 12px;">
              <span class="hint-priority" :class="'hint-' + hint.priority_label">{{ hint.priority_label || 'P' + hint.priority }}</span>
              {{ hint.reason }}
              <el-button link type="primary" size="small" @click="go(hint.path)" style="margin-left: 8px;">{{ hint.action }}</el-button>
            </div>
          </div>
          <div v-else>
            <p class="ai-greeting">System is running smoothly. No critical issues detected.</p>
          </div>
          <div class="ai-input-wrap">
            <el-input placeholder="Ask AI anything..." class="copilot-input" @keyup.enter="go('/ai')">
              <template #append><el-button icon="Position" @click="go('/ai')"/></template>
            </el-input>
          </div>
        </el-card>

        <!-- Quality Dimensions -->
        <el-card class="side-card waterline-card" shadow="never">
          <h3>Data Quality</h3>
          <div class="waterline-item" v-for="dim in quality.dimensions" :key="dim.key">
            <div class="wl-label">{{ dim.label }}</div>
            <div class="wl-bar-wrap"><div class="wl-bar" :class="dimBarClass(dim.score)" :style="{ width: dim.score + '%' }"></div></div>
            <div class="wl-val">{{ dim.score }}%</div>
          </div>
          <div v-if="!quality.dimensions || quality.dimensions.length === 0" class="empty-hint">
            Quality assessment in progress...
          </div>
        </el-card>

        <!-- Deployment Health -->
        <el-card class="side-card deploy-health-card" shadow="never">
          <h3>Deploy Health</h3>
          <div v-if="lastExecutions.length === 0" class="empty-hint">No recent deployments</div>
          <div v-else class="deploy-list">
            <div v-for="ex in lastExecutions" :key="ex.id" class="deploy-item">
              <span class="deploy-dot" :class="ex._dotClass"></span>
              <div class="deploy-info">
                <div class="deploy-name">{{ ex.pipeline_name || ex.name || 'Pipeline' }}</div>
                <div class="deploy-meta">{{ fmtTimeAgo(ex.created_at) }}</div>
              </div>
              <el-tag size="small" :type="ex._tagType">{{ ex.status || ex._status }}</el-tag>
            </div>
          </div>
        </el-card>

        <!-- Module Status -->
        <el-card class="side-card module-status-card" shadow="never" v-if="moduleEntries.length > 0">
          <h3>Plugin Status</h3>
          <div class="module-grid">
            <span
              v-for="m in moduleEntries"
              :key="m.name"
              class="module-chip"
              :class="'module-' + m.status"
            >{{ m.name }}</span>
          </div>
          <div class="module-legend">
            <span><span class="dot bg-green"></span> online</span>
            <span><span class="dot bg-red"></span> error</span>
            <span><span class="dot bg-gray"></span> disabled</span>
          </div>
        </el-card>

        <OpsCalendar />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'
import OpsCalendar from '@/components/dashboard/OpsCalendar.vue'

const router = useRouter()
const loading = ref(false)
const dataAge = ref('')
const autoRefresh = ref(true)
let ageTimer = null
let refreshTimer = null

const summary = reactive({
  hostTotal: 0, hostOnline: 0, hostOffline: 0, hostStale: 0,
  dockerTotal: 0, dockerOnline: 0, dockerOffline: 0, dockerStale: 0,
  k8sTotal: 0, k8sHealthy: 0, k8sUnhealthy: 0, k8sMaintenance: 0, k8sStale: 0,
  firewallTotal: 0, firewallOnline: 0, firewallOffline: 0, firewallAlert: 0, firewallStale: 0,
  domainTotal: 0, domainHealthy: 0, domainWarning: 0, domainCritical: 0, domainStale: 0,
  alertTotal: 0, alertOpen: 0,
  taskTotal: 0, taskEnabled: 0,
  agentTotal: 0, agentOnline: 0,
  moduleStatus: {}
})

const quality = reactive({
  trustScore: 0,
  trustGrade: 'N/A',
  summary: '',
  dimensions: [],
  actionHints: []
})

const snapshots = reactive({
  alerts: [],
  cicd_executions: [],
  workorders: [],
  metrics: null,
  metric_history: [],
  jump_assets: [],
  domains: [],
  certs: []
})

const generatedAt = ref(null)

const moduleEntries = computed(() => {
  const map = summary.moduleStatus || {}
  return Object.entries(map).map(([name, status]) => ({
    name,
    status: String(status || '').toLowerCase()
  }))
})

const lastExecutions = computed(() => {
  return (snapshots.cicd_executions || []).slice(0, 5).map(e => ({
    ...e,
    _status: e.status || 'unknown',
    _tagType: e.status === 'success' ? 'success' : e.status === 'failed' ? 'danger' : e.status === 'running' ? 'warning' : 'info',
    _dotClass: e.status === 'success' ? 'bg-green' : e.status === 'failed' ? 'bg-red' : e.status === 'running' ? 'bg-blue' : 'bg-gray'
  }))
})

const fmtTimeAgo = (val) => {
  if (!val) return '-'
  const diff = Math.floor((Date.now() - new Date(val).getTime()) / 1000)
  if (diff < 60) return '刚刚'
  if (diff < 3600) return `${Math.floor(diff / 60)}m 前`
  if (diff < 86400) return `${Math.floor(diff / 3600)}h 前`
  return `${Math.floor(diff / 86400)}d 前`
}

const topTargets = computed(() => {
  const alerts = snapshots.alerts || []
  const counts = {}
  alerts.forEach(a => {
    const target = a.target || 'unknown'
    counts[target] = (counts[target] || 0) + 1
  })
  return Object.entries(counts)
    .map(([target, count]) => {
      const maxSeverity = alerts.filter(a => a.target === target).reduce((max, a) => {
        if (a.severity === 'critical') return 'critical'
        if (a.severity === 'warning' && max !== 'critical') return 'warning'
        return max
      }, 'info')
      return { target, count, severity: maxSeverity }
    })
    .sort((a, b) => b.count - a.count)
    .slice(0, 5)
})

const recentAIOps = ref([])

const fetchAIOps = async () => {
  try {
    const headers = { Authorization: 'Bearer ' + localStorage.getItem('token') }
    const res = await axios.get('/api/v1/ai/ops/incidents', { headers }).catch(() => ({ data: {} }))
    recentAIOps.value = (res.data?.data || []).slice(0, 3)
  } catch (e) { /* silent */ }
}

const pendingWorkorders = computed(() => {
  const wos = snapshots.workorders || []
  return wos.filter(w => w.status === 'pending' || w.status === 'open').length
})

const recentAlerts = computed(() => {
  const alerts = snapshots.alerts || []
  return alerts.filter(a => a.status === 'firing' || a.status === 'open').slice(0, 5).map(a => ({
    ...a,
    _severityType: a.severity === 'critical' ? 'danger' : a.severity === 'warning' ? 'warning' : 'info',
    _iconColor: a.severity === 'critical' ? 'text-red' : a.severity === 'warning' ? 'text-orange' : 'text-blue'
  }))
})

const topActionHint = computed(() => {
  const hints = quality.actionHints || []
  if (hints.length === 0) return null
  return hints.sort((a, b) => (a.priority || 99) - (b.priority || 99))[0]
})

const validActionHints = computed(() => {
  const hints = quality.actionHints || []
  return hints
    .filter(h => h.path && h.action)
    .sort((a, b) => (a.priority || 99) - (b.priority || 99))
})

const metricHistory = computed(() => snapshots.metric_history || [])

const metricLabels = computed(() => {
  if (metricHistory.value.length === 0) return []
  const step = Math.max(1, Math.floor(metricHistory.value.length / 6))
  return metricHistory.value.filter((_, i) => i % step === 0).map(m => {
    const d = new Date(m.at)
    return `${d.getMonth() + 1}/${d.getDate()} ${d.getHours()}:00`
  }).slice(0, 7)
})

const metricLinePath = computed(() => {
  if (metricHistory.value.length === 0) return ''
  const cpuValues = metricHistory.value.map(m => m.cpu_percent || 0)
  return buildSvgPath(cpuValues, 100, 0, 100)
})

const metricAreaPath = computed(() => {
  if (metricHistory.value.length === 0) return ''
  return metricLinePath.value + ' L400,100 L0,100 Z'
})

function buildSvgPath(values, maxY, width, height) {
  if (values.length === 0) return ''
  const step = width / Math.max(1, values.length - 1)
  const clamp = (v) => Math.max(2, Math.min(height - 2, height - (v / maxY) * height))
  let d = `M0,${clamp(values[0])}`
  for (let i = 1; i < values.length; i++) {
    d += ` L${Math.round(i * step)},${clamp(values[i])}`
  }
  return d
}

const randomSpark = (seed) => {
  // Use real metric data if available, otherwise fallback
  const history = metricHistory.value
  if (history.length > 1) {
    const values = history.map(m => m.cpu_percent || 0)
    // Interpolate to 100px width
    const step = 100 / (values.length - 1)
    const maxVal = Math.max(...values, 1)
    let d = ''
    values.forEach((v, i) => {
      const x = Math.round(i * step)
      const y = Math.round(18 - (v / maxVal) * 16) // scale to 2-18 range
      d += `${i === 0 ? 'M' : 'L'}${x},${y}`
    })
    return d
  }
  // Fallback patterns
  const patterns = [
    'M0,15 L20,15 L30,5 L40,18 L50,12 L70,12 L80,8 L100,10',
    'M0,5 L20,8 L40,2 L60,12 L80,10 L100,15',
    'M0,18 L15,10 L30,12 L50,4 L70,15 L85,8 L100,12',
    'M0,10 L20,12 L40,8 L60,15 L80,5 L100,10'
  ]
  return patterns[seed % patterns.length]
}

const dimBarClass = (score) => {
  if (score >= 80) return 'bg-green'
  if (score >= 50) return 'bg-blue'
  return 'bg-red'
}

const go = (path) => router.push(path)
const fmtTime = (val) => {
  if (!val) return '-'
  const d = new Date(val)
  const now = new Date()
  const diffMs = now - d
  const diffMin = Math.floor(diffMs / 60000)
  if (diffMin < 1) return 'just now'
  if (diffMin < 60) return `${diffMin}m ago`
  if (diffMin < 1440) return `${Math.floor(diffMin / 60)}h ago`
  return d.toLocaleDateString()
}

const fetchData = async () => {
  loading.value = true
  try {
    const authHeaders = { Authorization: 'Bearer ' + localStorage.getItem('token') }
    const res = await axios.get('/api/v1/dashboard/overview', { headers: authHeaders })
    const d = res.data?.data

    if (d) {
      // Summary
      Object.assign(summary, d.summary || {})

      // Quality
      quality.trustScore = d.quality?.trust_score ?? 0
      quality.trustGrade = d.quality?.trust_grade ?? 'N/A'
      quality.summary = d.quality?.summary ?? ''
      quality.dimensions = d.quality?.dimensions || []
      quality.actionHints = d.quality?.action_hints || []

      // Snapshots
      if (d.snapshots) {
        snapshots.alerts = d.snapshots.alerts || []
        snapshots.cicd_executions = d.snapshots.cicd_executions || []
        snapshots.workorders = d.snapshots.workorders || []
        snapshots.metrics = d.snapshots.metrics || null
        snapshots.metric_history = d.snapshots.metric_history || []
        snapshots.jump_assets = d.snapshots.jump_assets || []
        snapshots.domains = d.snapshots.domains || []
        snapshots.certs = d.snapshots.certs || []
      }

      generatedAt.value = d.generated_at ? new Date(d.generated_at) : new Date()
    }
  } catch (err) {
    console.error('[Dashboard] Failed to load overview:', err)
  } finally {
    loading.value = false
  }
  fetchAIOps()
}

const updateAge = () => {
  if (!generatedAt.value) { dataAge.value = ''; return }
  const diff = Math.floor((Date.now() - generatedAt.value) / 1000)
  if (diff < 60) dataAge.value = `updated ${diff}s ago`
  else if (diff < 3600) dataAge.value = `updated ${Math.floor(diff / 60)}m ago`
  else dataAge.value = `updated ${Math.floor(diff / 3600)}h ago`
}

onMounted(() => {
  fetchData()
  ageTimer = setInterval(updateAge, 30000)
  refreshTimer = setInterval(() => {
    if (autoRefresh.value) fetchData()
  }, 30000)
})

onUnmounted(() => {
  clearInterval(ageTimer)
  clearInterval(refreshTimer)
})
</script>

<style scoped>
.dash-wrapper { max-width: 1400px; margin: 0 auto; padding: 20px; }
.dash-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 24px; }
.dash-header h1 { font-size: 26px; font-weight: 800; margin: 0 0 4px 0; color: var(--el-text-color-primary); }
.dash-header p { font-size: 14px; color: var(--el-text-color-secondary); margin: 0; }
.dash-actions { display: flex; align-items: center; gap: 12px; }

.kpi-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 20px; margin-bottom: 20px; }
.kpi-card { padding: 20px; position: relative; overflow: hidden; cursor: pointer; }
.kpi-top { display: flex; align-items: flex-start; gap: 16px; margin-bottom: 12px; }
.kpi-icon { width: 40px; height: 40px; border-radius: 12px; background: rgba(0,0,0,0.04); display: flex; align-items: center; justify-content: center; font-size: 20px; }
.kpi-info { display: flex; flex-direction: column; }
.kpi-title { font-size: 13px; font-weight: 600; color: var(--el-text-color-secondary); }
.kpi-val { font-size: 28px; font-weight: 800; color: var(--el-text-color-primary); line-height: 1.1; }
.kpi-unit { font-size: 14px; font-weight: 600; color: var(--el-text-color-secondary); margin-left: 4px; }
.kpi-trend { font-size: 12px; font-weight: 600; line-height: 1.4; }
.kpi-sparkline { position: absolute; bottom: 0; left: 0; right: 0; height: 40px; opacity: 0.2; }

.main-grid { display: grid; grid-template-columns: 2fr 1fr; gap: 20px; }
.main-left { display: flex; flex-direction: column; gap: 20px; }
.overview-card { display: flex; flex-direction: column; }
.card-title-row { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }
.card-title-row h3 { font-size: 16px; font-weight: 700; margin: 0; color: var(--el-text-color-primary); }
.view-all { font-size: 13px; font-weight: 600; color: var(--apple-blue); text-decoration: none; }
.info-icon { color: var(--el-text-color-secondary); font-size: 14px; }

.infra-stats { display: grid; grid-template-columns: repeat(4, 1fr); gap: 24px; margin-bottom: 30px; }
.infra-stat { display: flex; flex-direction: column; gap: 4px; cursor: pointer; }
.infra-stat .label { font-size: 12px; color: var(--el-text-color-secondary); font-weight: 600; text-transform: uppercase; }
.infra-stat .val { font-size: 22px; font-weight: 800; color: var(--el-text-color-primary); }
.infra-stat .status { font-size: 12px; display: flex; align-items: center; font-weight: 600; }
.dot { width: 6px; height: 6px; border-radius: 50%; margin-right: 6px; }

.fake-area-chart { height: 160px; position: relative; }
.fake-area-chart svg { width: 100%; height: 100%; }
.chart-labels { display: flex; justify-content: space-between; font-size: 11px; color: var(--el-text-color-secondary); margin-top: 10px; }

.sub-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 20px; }
.issue-item { display: flex; align-items: flex-start; gap: 12px; padding: 10px 0; border-bottom: 1px solid var(--glass-outline); cursor: pointer; }
.issue-item:last-child { border-bottom: none; }
.issue-rank { width: 36px; height: 24px; border-radius: 6px; display: flex; align-items: center; justify-content: center; font-size: 11px; font-weight: 800; color: #fff; flex-shrink: 0; text-transform: uppercase; }
.issue-rank.rank-critical { background: #ef4444; }
.issue-rank.rank-warning { background: #f59e0b; }
.issue-rank.rank-info, .issue-rank.rank-medium { background: #6b7280; }
.issue-rank.rank-low { background: #10b981; }
.issue-info { flex: 1; min-width: 0; }
.issue-target { font-size: 13px; font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.issue-meta { font-size: 11px; color: var(--el-text-color-secondary); margin-top: 2px; }

.empty-hint { padding: 24px 0; text-align: center; color: var(--el-text-color-secondary); font-size: 13px; }
.incident-item { display: flex; align-items: flex-start; gap: 12px; padding: 12px 0; border-bottom: 1px solid var(--glass-outline); cursor: pointer; }
.incident-item:last-child { border-bottom: none; }
.inc-icon { font-size: 18px; margin-top: 2px; }
.inc-content { flex: 1; }
.inc-title { font-size: 14px; font-weight: 700; display: flex; align-items: center; justify-content: space-between; }
.inc-meta { font-size: 12px; color: var(--el-text-color-secondary); margin-top: 4px; }

.change-item { display: flex; align-items: flex-start; gap: 12px; padding: 12px 0; }
.chg-dot { width: 8px; height: 8px; border-radius: 50%; margin-top: 6px; flex-shrink: 0; }
.chg-desc { font-size: 14px; font-weight: 600; color: var(--el-text-color-primary); }
.chg-meta { font-size: 12px; color: var(--el-text-color-secondary); margin-top: 4px; }

.ai-copilot-card { background: linear-gradient(135deg, rgba(0, 113, 227, 0.05) 0%, rgba(139, 92, 246, 0.05) 100%); }
.copilot-title { background: linear-gradient(135deg, #0071e3, #8b5cf6); -webkit-background-clip: text; -webkit-text-fill-color: transparent; }
.ai-greeting { font-size: 14px; color: var(--el-text-color-regular); line-height: 1.6; margin: 0 0 16px 0; }
.hint-priority {
  display: inline-block;
  font-size: 10px;
  font-weight: 800;
  padding: 1px 6px;
  border-radius: 4px;
  margin-right: 4px;
  text-transform: uppercase;
}
.hint-priority.hint-p0, .hint-priority.hint-critical { background: #fef2f2; color: #dc2626; }
.hint-priority.hint-p1, .hint-priority.hint-high { background: #fff7ed; color: #ea580c; }
.hint-priority.hint-p2, .hint-priority.hint-medium { background: #fefce8; color: #ca8a04; }
.hint-priority.hint-p3, .hint-priority.hint-low { background: #f0fdf4; color: #16a34a; }
.ai-actions { display: flex; gap: 10px; margin-bottom: 24px; }
.ai-btn { height: 36px; padding: 0 20px; font-weight: 700; box-shadow: 0 4px 12px rgba(0, 113, 227, 0.2); }
.ai-btn-alt { height: 36px; padding: 0 20px; font-weight: 600; background: transparent; border: 1px solid var(--glass-outline) !important; }
.ai-input-wrap { margin-top: auto; }
.copilot-input :deep(.el-input__wrapper) { border-radius: 12px; background: rgba(255,255,255,0.6); }

.waterline-item { display: flex; align-items: center; gap: 12px; margin-bottom: 16px; }
.wl-label { width: 80px; font-size: 12px; font-weight: 700; color: var(--el-text-color-secondary); text-transform: uppercase; margin-bottom: 0; }
.wl-bar-wrap { height: 6px; background: rgba(0,0,0,0.05); border-radius: 3px; overflow: hidden; flex: 1; }
.wl-bar { height: 100%; border-radius: 3px; }
.wl-val { width: 40px; font-size: 13px; font-weight: 700; text-align: right; }

.bg-red-light { background: rgba(239, 68, 68, 0.08); }
.bg-purple-light { background: rgba(139, 92, 246, 0.08); }
.bg-orange-light { background: rgba(245, 158, 11, 0.08); }
.text-green { color: #10b981; }
.text-red { color: #ef4444; }
.text-orange { color: #f59e0b; }
.text-purple { color: #8b5cf6; }
.bg-green { background: #10b981; }
.bg-blue { background: #0071e3; }
.bg-red { background: #ef4444; }
.bg-purple { background: #8b5cf6; }
.bg-orange { background: #f59e0b; }
.bg-gray { background: #9ca3af; }
.flat-tag { background: rgba(0,0,0,0.04) !important; border: none !important; border-radius: 6px !important; font-weight: 700 !important; }

.deploy-health-card { margin-bottom: 16px; }
.deploy-health-card h3 { font-size: 14px; font-weight: 700; margin: 0 0 12px 0; }
.deploy-list { display: flex; flex-direction: column; gap: 2px; }
.deploy-item { display: flex; align-items: center; gap: 10px; padding: 6px 0; }
.deploy-dot { width: 7px; height: 7px; border-radius: 50%; flex-shrink: 0; }
.deploy-info { flex: 1; min-width: 0; }
.deploy-name { font-size: 13px; font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.deploy-meta { font-size: 11px; color: var(--el-text-color-secondary); }

.module-status-card { margin-bottom: 16px; }
.module-status-card h3 { font-size: 14px; font-weight: 700; margin: 0 0 12px 0; }
.module-grid { display: flex; flex-wrap: wrap; gap: 6px; margin-bottom: 12px; }
.module-chip {
  font-size: 11px;
  font-weight: 600;
  padding: 3px 10px;
  border-radius: 6px;
  text-transform: uppercase;
}
.module-chip.module-online, .module-chip.module-ok, .module-chip.module-enabled { background: #f0fdf4; color: #16a34a; }
.module-chip.module-offline, .module-chip.module-error, .module-chip.module-failed { background: #fef2f2; color: #dc2626; }
.module-chip.module-disabled { background: #f3f4f6; color: #9ca3af; }
.module-legend { display: flex; gap: 16px; font-size: 11px; color: var(--el-text-color-secondary); }
.module-legend .dot { display: inline-block; width: 6px; height: 6px; border-radius: 50%; margin-right: 4px; }
</style>
