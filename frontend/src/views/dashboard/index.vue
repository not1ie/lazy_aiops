<template>
  <div class="dash">
    <!-- Header -->
    <div class="dash-top">
      <div>
        <h1>{{ greeting }}</h1>
        <p>{{ statusSummary }}</p>
      </div>
      <div class="dash-top-right">
        <span class="dash-time">{{ nowStr }}</span>
        <el-button icon="Refresh" circle size="small" @click="fetchData" />
      </div>
    </div>

    <!-- KPI Cards -->
    <div class="kpi-row">
      <div class="kpi" @click="go('/asset')">
        <div class="kpi-num">{{ summary.hostOnline }}<span class="kpi-den">/{{ summary.hostTotal }}</span></div>
        <div class="kpi-label">Hosts Online</div>
        <div class="kpi-sub" :class="summary.hostOffline > 0 ? 'warn' : ''">{{ summary.hostOffline > 0 ? `${summary.hostOffline} offline` : 'All healthy' }}</div>
      </div>
      <div class="kpi" @click="go('/monitor')">
        <div class="kpi-num" :class="summary.alertOpen > 0 ? 'danger' : ''">{{ summary.alertOpen }}</div>
        <div class="kpi-label">Active Alerts</div>
        <div class="kpi-sub" :class="summary.alertOpen > 0 ? 'danger' : ''">{{ summary.alertOpen > 0 ? `${summary.alertCritical} critical` : 'All clear' }}</div>
      </div>
      <div class="kpi" @click="go('/delivery')">
        <div class="kpi-num">{{ summary.k8sTotal + summary.dockerTotal }}</div>
        <div class="kpi-label">Clusters & Hosts</div>
        <div class="kpi-sub">{{ summary.k8sTotal }} K8s &bull; {{ summary.dockerTotal }} Docker</div>
      </div>
      <div class="kpi" @click="go('/delivery')">
        <div class="kpi-num">{{ pendingWorkorders }}</div>
        <div class="kpi-label">Pending Tickets</div>
        <div class="kpi-sub">{{ lastExecutions.length }} recent deploys</div>
      </div>
    </div>

    <!-- Main Content -->
    <div class="dash-grid">
      <!-- Left: Alerts + Activity -->
      <div class="dash-main">
        <div class="card">
          <div class="card-head">
            <h3>Active Alerts</h3>
            <a @click.prevent="go('/monitor')">View all →</a>
          </div>
          <div v-if="recentAlerts.length === 0" class="card-empty">No active alerts — all clear</div>
          <div v-else class="alert-list">
            <div v-for="a in recentAlerts" :key="a.id" class="alert-row" @click="go('/alert/events/detail?id=' + a.id)">
              <span class="alert-dot" :class="'dot-' + a.severity"></span>
              <span class="alert-name">{{ a.alert_name || a.rule_name }}</span>
              <span class="alert-target">{{ a.target }}</span>
              <span class="alert-time">{{ fmtTimeAgo(a.fired_at || a.created_at) }}</span>
            </div>
          </div>
        </div>

        <div class="card">
          <div class="card-head">
            <h3>Recent Activity</h3>
            <a @click.prevent="go('/delivery')">View all →</a>
          </div>
          <div v-if="lastExecutions.length === 0" class="card-empty">No recent activity</div>
          <div v-else class="activity-list">
            <div v-for="ex in lastExecutions" :key="ex.id" class="act-row" @click="go('/cicd/executions')">
              <span class="act-dot" :class="ex._dotClass"></span>
              <span class="act-name">{{ ex.pipeline_name || 'Pipeline' }}</span>
              <span class="act-status" :class="'status-' + (ex.status || 'unknown')">{{ ex.status }}</span>
              <span class="act-time">{{ fmtTimeAgo(ex.created_at) }}</span>
            </div>
          </div>
          <!-- AIOps incidents -->
          <div v-if="recentAIOps.length > 0" style="margin-top:12px;border-top:1px solid var(--el-border-color-lighter);padding-top:12px">
            <div v-for="inc in recentAIOps" :key="inc.incident_id" class="act-row" @click="go('/ai/ops')">
              <span class="act-dot" :class="inc.risk_level === 'critical' ? 'bg-red' : 'bg-blue'"></span>
              <span class="act-name">{{ inc.title || inc.query?.slice(0, 50) }}</span>
              <span class="act-status status-aiops">AIOps</span>
              <span class="act-time">{{ fmtTimeAgo(inc.created_at) }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Right: Quick Nav + Status -->
      <div class="dash-side">
        <div class="card">
          <h3 style="margin:0 0 16px 0;font-size:14px;font-weight:700">Quick Access</h3>
          <div class="quick-nav">
            <div class="qn-item" v-for="hub in quickHubs" :key="hub.path" @click="go(hub.path)">
              <el-icon :size="16"><component :is="hub.icon" /></el-icon>
              <span>{{ hub.label }}</span>
              <el-icon :size="12"><ArrowRight /></el-icon>
            </div>
          </div>
        </div>

        <div class="card">
          <h3 style="margin:0 0 12px 0;font-size:14px;font-weight:700">Health</h3>
          <div class="health-list">
            <div v-for="h in healthItems" :key="h.key" class="health-row">
              <span class="health-dot" :class="h.ok ? 'bg-green' : 'bg-red'"></span>
              <span class="health-label">{{ h.label }}</span>
              <span class="health-val">{{ h.val }}</span>
            </div>
          </div>
        </div>

        <div class="card" v-if="oncallNow.length > 0">
          <h3 style="margin:0 0 8px 0;font-size:14px;font-weight:700">On Call</h3>
          <div v-for="o in oncallNow" :key="o.id || o.username" class="oncall-chip">
            <el-avatar :size="20" icon="UserFilled" />
            <span>{{ o.username || o.name }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'

const router = useRouter()
const loading = ref(false)
let refreshTimer = null

const summary = reactive({ hostTotal: 0, hostOnline: 0, hostOffline: 0, k8sTotal: 0, dockerTotal: 0, alertOpen: 0, alertCritical: 0 })
const snapshots = reactive({ alerts: [], cicd_executions: [], workorders: [] })
const recentAIOps = ref([])
const oncallNow = ref([])
const nowStr = ref('')

const authHeaders = () => ({ Authorization: 'Bearer ' + localStorage.getItem('token') })
const go = (path) => router.push(path)

const greeting = computed(() => {
  const h = new Date().getHours()
  if (h < 12) return 'Good morning'
  if (h < 18) return 'Good afternoon'
  return 'Good evening'
})

const statusSummary = computed(() => {
  const parts = []
  if (summary.alertOpen > 0) parts.push(`${summary.alertOpen} alerts need attention`)
  else parts.push('All systems operational')
  if (summary.hostOffline > 0) parts.push(`${summary.hostOffline} hosts offline`)
  return parts.join(' · ')
})

const recentAlerts = computed(() => (snapshots.alerts || []).filter(a => a.status === 'firing').slice(0, 6))

const lastExecutions = computed(() => (snapshots.cicd_executions || []).slice(0, 4).map(e => ({
  ...e,
  _dotClass: e.status === 'success' ? 'bg-green' : e.status === 'failed' ? 'bg-red' : 'bg-blue'
})))

const pendingWorkorders = computed(() => (snapshots.workorders || []).filter(w => w.status === 'pending' || w.status === 'open').length)

const quickHubs = [
  { label: '资产与安全', path: '/asset', icon: 'Monitor' },
  { label: '容器平台', path: '/k8s', icon: 'Platform' },
  { label: '告警运营台', path: '/monitor', icon: 'Histogram' },
  { label: '变更交付', path: '/delivery', icon: 'Connection' },
  { label: '系统治理', path: '/system', icon: 'Setting' },
  { label: 'AIOps 诊断', path: '/ai/ops', icon: 'MagicStick' }
]

const healthItems = computed(() => [
  { key: 'hosts', label: 'Hosts', ok: summary.hostOffline === 0, val: `${summary.hostOnline}/${summary.hostTotal}` },
  { key: 'alerts', label: 'Alerts', ok: summary.alertOpen === 0, val: `${summary.alertOpen}` },
  { key: 'k8s', label: 'K8s', ok: true, val: `${summary.k8sTotal}` },
  { key: 'docker', label: 'Docker', ok: true, val: `${summary.dockerTotal}` }
])

const fmtTimeAgo = (val) => {
  if (!val) return ''
  const diff = Math.floor((Date.now() - new Date(val).getTime()) / 1000)
  if (diff < 60) return 'now'
  if (diff < 3600) return `${Math.floor(diff / 60)}m`
  if (diff < 86400) return `${Math.floor(diff / 3600)}h`
  return `${Math.floor(diff / 86400)}d`
}

const updateClock = () => {
  nowStr.value = new Date().toLocaleString('en-US', { weekday: 'short', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
}

const fetchData = async () => {
  loading.value = true
  try {
    const headers = authHeaders()
    const res = await axios.get('/api/v1/dashboard/overview', { headers })
    const d = res.data?.data
    if (d?.summary) {
      summary.hostTotal = d.summary.host_total || 0
      summary.hostOnline = d.summary.host_online || 0
      summary.hostOffline = d.summary.host_offline || 0
      summary.k8sTotal = d.summary.k8s_total || 0
      summary.dockerTotal = d.summary.docker_total || 0
      summary.alertOpen = d.summary.alert_open || 0
      summary.alertCritical = (d.snapshots?.alerts || []).filter(a => a.severity === 'critical').length
    }
    if (d?.snapshots) {
      snapshots.alerts = d.snapshots.alerts || []
      snapshots.cicd_executions = d.snapshots.cicd_executions || []
      snapshots.workorders = d.snapshots.workorders || []
    }
  } catch (e) { /* silent */ }
  finally { loading.value = false }

  // Also fetch oncall + AIOps
  try {
    const h = authHeaders()
    const [oncRes, aiRes] = await Promise.all([
      axios.get('/api/v1/oncall/whoisoncall', { headers: h }).catch(() => ({ data: {} })),
      axios.get('/api/v1/ai/ops/incidents', { headers: h }).catch(() => ({ data: {} }))
    ])
    oncallNow.value = oncRes.data?.data || []
    recentAIOps.value = (aiRes.data?.data || []).slice(0, 2)
  } catch (e) { /* silent */ }
}

onMounted(() => {
  fetchData()
  updateClock()
  setInterval(updateClock, 30000)
  refreshTimer = setInterval(fetchData, 30000)
})

onUnmounted(() => { clearInterval(refreshTimer) })
</script>

<style scoped>
.dash { max-width: 1100px; margin: 0 auto; padding: 32px 24px; }

/* Header */
.dash-top { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 32px; }
.dash-top h1 { font-size: 28px; font-weight: 700; margin: 0 0 4px; letter-spacing: -0.02em; color: var(--el-text-color-primary); }
.dash-top p { font-size: 14px; color: var(--el-text-color-secondary); margin: 0; }
.dash-top-right { display: flex; align-items: center; gap: 12px; }
.dash-time { font-size: 13px; color: var(--el-text-color-secondary); }

/* KPI Row */
.kpi-row { display: grid; grid-template-columns: repeat(4, 1fr); gap: 16px; margin-bottom: 32px; }
.kpi { padding: 20px; border-radius: 16px; background: var(--el-bg-color); border: 1px solid var(--el-border-color-lighter); cursor: pointer; transition: box-shadow 0.2s; }
.kpi:hover { box-shadow: 0 2px 12px rgba(0,0,0,0.06); }
.kpi-num { font-size: 36px; font-weight: 700; letter-spacing: -0.02em; color: var(--el-text-color-primary); }
.kpi-num.danger { color: #ef4444; }
.kpi-den { font-size: 16px; font-weight: 500; color: var(--el-text-color-placeholder); margin-left: 4px; }
.kpi-label { font-size: 13px; color: var(--el-text-color-secondary); margin-top: 4px; font-weight: 500; }
.kpi-sub { font-size: 12px; color: var(--el-text-color-placeholder); margin-top: 4px; }
.kpi-sub.warn { color: #f59e0b; }
.kpi-sub.danger { color: #ef4444; }

/* Main Grid */
.dash-grid { display: grid; grid-template-columns: 1fr 280px; gap: 20px; }
.dash-main { display: flex; flex-direction: column; gap: 16px; }

/* Cards */
.card { background: var(--el-bg-color); border: 1px solid var(--el-border-color-lighter); border-radius: 16px; padding: 20px; }
.card-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.card-head h3 { font-size: 15px; font-weight: 700; margin: 0; }
.card-head a { font-size: 13px; font-weight: 600; color: var(--el-color-primary); text-decoration: none; cursor: pointer; }
.card-empty { font-size: 13px; color: var(--el-text-color-placeholder); padding: 16px 0; text-align: center; }

/* Alert rows */
.alert-row { display: flex; align-items: center; gap: 10px; padding: 8px 0; border-bottom: 1px solid var(--el-border-color-lighter); cursor: pointer; font-size: 13px; }
.alert-row:last-child { border-bottom: none; }
.alert-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
.dot-critical { background: #ef4444; }
.dot-warning { background: #f59e0b; }
.dot-info { background: #6b7280; }
.alert-name { font-weight: 600; flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.alert-target { color: var(--el-text-color-secondary); font-size: 12px; max-width: 120px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.alert-time { color: var(--el-text-color-placeholder); font-size: 11px; flex-shrink: 0; }

/* Activity rows */
.activity-list { display: flex; flex-direction: column; }
.act-row { display: flex; align-items: center; gap: 10px; padding: 7px 0; border-bottom: 1px solid var(--el-border-color-lighter); cursor: pointer; font-size: 13px; }
.act-row:last-child { border-bottom: none; }
.act-dot { width: 7px; height: 7px; border-radius: 50%; flex-shrink: 0; }
.act-name { font-weight: 600; flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.act-status { font-size: 11px; font-weight: 600; padding: 2px 8px; border-radius: 6px; }
.status-success { background: #f0fdf4; color: #16a34a; }
.status-failed { background: #fef2f2; color: #dc2626; }
.status-running { background: #eff6ff; color: #2563eb; }
.status-aiops { background: #faf5ff; color: #7c3aed; }
.act-time { color: var(--el-text-color-placeholder); font-size: 11px; flex-shrink: 0; }

/* Quick Nav */
.quick-nav { display: flex; flex-direction: column; gap: 2px; }
.qn-item { display: flex; align-items: center; gap: 10px; padding: 10px 8px; border-radius: 10px; cursor: pointer; font-size: 13px; font-weight: 600; color: var(--el-text-color-regular); transition: background 0.15s; }
.qn-item:hover { background: var(--el-fill-color-light); }
.qn-item span { flex: 1; }

/* Health */
.health-list { display: flex; flex-direction: column; gap: 4px; }
.health-row { display: flex; align-items: center; gap: 8px; font-size: 13px; }
.health-dot { width: 7px; height: 7px; border-radius: 50%; flex-shrink: 0; }
.health-label { flex: 1; color: var(--el-text-color-secondary); }
.health-val { font-weight: 600; color: var(--el-text-color-regular); }

/* Oncall */
.oncall-chip { display: flex; align-items: center; gap: 8px; padding: 6px 0; font-size: 13px; font-weight: 600; }

.bg-green { background: #10b981; }
.bg-red { background: #ef4444; }
.bg-blue { background: #2563eb; }
</style>
