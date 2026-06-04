<template>
  <div class="dash">
    <!-- Top bar -->
    <div class="dash-top">
      <div>
        <h1>{{ greeting }}，{{ username }}</h1>
        <p>{{ statusSummary }}</p>
      </div>
      <div class="dash-top-right">
        <span class="dash-time">{{ nowStr }}</span>
        <el-button size="small" icon="Refresh" circle @click="fetchData" />
      </div>
    </div>

    <!-- Status row -->
    <div class="status-row">
      <div class="status-chip ok" @click="go('/asset')">
        <span class="chip-num">{{ summary.hostOnline }}</span>
        <span class="chip-label">主机在线</span>
        <span class="chip-sub">共 {{ summary.hostTotal }} 台</span>
      </div>
      <div class="status-chip" :class="summary.alertOpen > 0 ? 'bad' : 'ok'" @click="go('/monitor')">
        <span class="chip-num">{{ summary.alertOpen }}</span>
        <span class="chip-label">活跃告警</span>
        <span class="chip-sub" v-if="summary.alertCritical > 0">{{ summary.alertCritical }} 严重</span>
      </div>
      <div class="status-chip ok" @click="go('/k8s')">
        <span class="chip-num">{{ summary.k8sTotal + summary.dockerTotal }}</span>
        <span class="chip-label">集群/主机</span>
        <span class="chip-sub">{{ summary.k8sTotal }} K8s · {{ summary.dockerTotal }} Docker</span>
      </div>
      <div class="status-chip" :class="pendingWorkorders > 0 ? 'warn' : 'ok'" @click="go('/delivery')">
        <span class="chip-num">{{ pendingWorkorders }}</span>
        <span class="chip-label">待处理工单</span>
        <span class="chip-sub" v-if="lastExecs.length > 0">{{ lastExecs.length }} 次部署</span>
      </div>
      <div class="status-chip ok" @click="go('/ai/ops')">
        <span class="chip-num">{{ recentAIOps.length }}</span>
        <span class="chip-label">AIOps 事件</span>
        <span class="chip-sub">最近诊断</span>
      </div>
    </div>

    <!-- Main grid -->
    <div class="dash-grid">
      <!-- Left column -->
      <div class="dash-main">
        <!-- Active alerts -->
        <div class="glass-panel">
          <div class="panel-head">
            <h3>活跃告警</h3>
            <a @click.prevent="go('/monitor')">全部 →</a>
          </div>
          <div v-if="recentAlerts.length === 0" class="panel-empty">暂无活跃告警</div>
          <div v-else class="alert-list">
            <div v-for="a in recentAlerts" :key="a.id" class="alert-row" @click="go('/alert/events/detail?id=' + a.id)">
              <span class="a-sev" :class="a.severity">{{ a.severity === 'critical' ? '严重' : a.severity === 'warning' ? '警告' : '信息' }}</span>
              <span class="a-name">{{ a.alert_name || a.rule_name }}</span>
              <span class="a-target">{{ a.target }}</span>
              <span class="a-time">{{ fmtRel(a.fired_at || a.created_at) }}</span>
            </div>
          </div>
        </div>

        <!-- Recent deployments + AIOps -->
        <div class="panel-grid">
          <div class="glass-panel">
            <div class="panel-head">
              <h3>最近部署</h3>
              <a @click.prevent="go('/delivery')">全部 →</a>
            </div>
            <div v-if="lastExecs.length === 0" class="panel-empty">暂无部署记录</div>
            <div v-else>
              <div v-for="ex in lastExecs" :key="ex.id" class="act-row" @click="go('/cicd/executions')">
                <span class="act-dot" :class="ex._dotClass"></span>
                <span class="act-name">{{ ex.pipeline_name }}</span>
                <span class="act-stat" :class="ex.status">{{ ex.status }}</span>
                <span class="act-time">{{ fmtRel(ex.created_at) }}</span>
              </div>
            </div>
          </div>
          <div class="glass-panel">
            <div class="panel-head">
              <h3>AIOps 诊断</h3>
              <a @click.prevent="go('/ai/ops')">全部 →</a>
            </div>
            <div v-if="recentAIOps.length === 0" class="panel-empty">暂无诊断记录</div>
            <div v-else>
              <div v-for="inc in recentAIOps" :key="inc.incident_id" class="act-row" @click="go('/ai/ops')">
                <span class="act-dot ai"></span>
                <span class="act-name">{{ inc.title || inc.query?.slice(0, 40) }}</span>
                <span class="act-stat">{{ inc.status }}</span>
                <span class="act-time">{{ fmtRel(inc.created_at) }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Right column -->
      <div class="dash-side">
        <div class="glass-panel">
          <h3 style="margin:0 0 12px;font-size:14px;font-weight:700">快捷入口</h3>
          <div class="quick-list">
            <div v-for="h in quickLinks" :key="h.path" class="ql-item" @click="go(h.path)">{{ h.label }}</div>
          </div>
        </div>

        <div class="glass-panel">
          <h3 style="margin:0 0 10px;font-size:14px;font-weight:700">系统健康</h3>
          <div class="health-list">
            <div class="hl-row" v-for="h in healthItems" :key="h.key">
              <span class="hl-dot" :class="h.ok ? 'ok' : 'bad'"></span>
              <span class="hl-label">{{ h.label }}</span>
              <span class="hl-val">{{ h.val }}</span>
            </div>
          </div>
        </div>

        <div class="glass-panel" v-if="oncallList.length > 0">
          <h3 style="margin:0 0 8px;font-size:14px;font-weight:700">今日值班</h3>
          <div v-for="o in oncallList" :key="o.id || o.username" class="oc-row">
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
const nowStr = ref('')
let refreshTimer = null

const username = ref(localStorage.getItem('username') || 'Admin')
const summary = reactive({ hostTotal: 0, hostOnline: 0, hostOffline: 0, k8sTotal: 0, dockerTotal: 0, alertOpen: 0, alertCritical: 0 })
const snapshots = reactive({ alerts: [], cicd_executions: [], workorders: [] })
const recentAIOps = ref([])
const oncallList = ref([])

const go = (p) => router.push(p)

const greeting = computed(() => {
  const h = new Date().getHours()
  if (h < 6) return '夜深了'
  if (h < 12) return '早上好'
  if (h < 18) return '下午好'
  return '晚上好'
})

const statusSummary = computed(() => {
  const p = []
  if (summary.alertOpen > 0) p.push(`${summary.alertOpen} 条告警需处理`)
  else p.push('所有系统运行正常')
  if (summary.hostOffline > 0) p.push(`${summary.hostOffline} 台离线`)
  return p.join('  ·  ')
})

const recentAlerts = computed(() => (snapshots.alerts || []).filter(a => a.status === 'firing').slice(0, 6))

const lastExecs = computed(() => (snapshots.cicd_executions || []).slice(0, 4).map(e => ({
  ...e,
  _dotClass: e.status === 'success' ? 'ok' : e.status === 'failed' ? 'bad' : 'warn'
})))

const pendingWorkorders = computed(() => (snapshots.workorders || []).filter(w => w.status === 'pending' || w.status === 'open').length)

const quickLinks = [
  { label: '资产与安全', path: '/asset' },
  { label: '容器平台', path: '/k8s' },
  { label: '告警运营台', path: '/monitor' },
  { label: '变更交付', path: '/delivery' },
  { label: '智能助手', path: '/ai' },
  { label: '系统治理', path: '/system' }
]

const healthItems = computed(() => [
  { key: 'hosts', label: '主机', ok: summary.hostOffline === 0, val: `${summary.hostOnline}/${summary.hostTotal}` },
  { key: 'alerts', label: '告警', ok: summary.alertOpen === 0, val: String(summary.alertOpen) },
  { key: 'k8s', label: 'K8s', ok: true, val: String(summary.k8sTotal) },
  { key: 'docker', label: 'Docker', ok: true, val: String(summary.dockerTotal) }
])

const fmtRel = (v) => {
  if (!v) return ''
  const d = Math.floor((Date.now() - new Date(v).getTime()) / 1000)
  if (d < 60) return '刚刚'
  if (d < 3600) return `${Math.floor(d / 60)}分钟前`
  if (d < 86400) return `${Math.floor(d / 3600)}小时前`
  return `${Math.floor(d / 86400)}天前`
}

const updateClock = () => {
  const n = new Date()
  nowStr.value = `${n.getFullYear()}/${n.getMonth()+1}/${n.getDate()} ${String(n.getHours()).padStart(2,'0')}:${String(n.getMinutes()).padStart(2,'0')}`
}

const fetchData = async () => {
  try {
    const h = { Authorization: 'Bearer ' + localStorage.getItem('token') }
    const res = await axios.get('/api/v1/dashboard/overview', { headers: h })
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
    // AIOps + oncall (non-blocking)
    const [ai, onc] = await Promise.all([
      axios.get('/api/v1/ai/ops/incidents', { headers: h }).catch(() => ({ data: {} })),
      axios.get('/api/v1/oncall/whoisoncall', { headers: h }).catch(() => ({ data: {} }))
    ])
    recentAIOps.value = (ai.data?.data || []).slice(0, 3)
    oncallList.value = onc.data?.data || []
  } catch (e) { /* silent */ }
}

onMounted(() => {
  fetchData()
  updateClock()
  setInterval(updateClock, 30000)
  refreshTimer = setInterval(fetchData, 30000)
})
onUnmounted(() => clearInterval(refreshTimer))
</script>

<style scoped>
.dash { max-width: 1100px; margin: 0 auto; padding: 24px 20px; }

/* Top */
.dash-top { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 24px; }
.dash-top h1 { font-size: 24px; font-weight: 700; margin: 0 0 4px; letter-spacing: -0.02em; }
.dash-top p { font-size: 13px; color: var(--el-text-color-secondary); margin: 0; }
.dash-top-right { display: flex; align-items: center; gap: 10px; }
.dash-time { font-size: 12px; color: var(--el-text-color-secondary); }

/* Status chips */
.status-row { display: flex; gap: 12px; margin-bottom: 24px; }
.status-chip {
  flex: 1; padding: 14px 16px; border-radius: 14px;
  background: var(--glass-bg); backdrop-filter: var(--glass-blur);
  -webkit-backdrop-filter: var(--glass-blur);
  border: 1px solid rgba(255,255,255,0.5); cursor: pointer;
  box-shadow: var(--shadow-sm);
  transition: all 0.15s;
}
.status-chip:hover { box-shadow: var(--shadow-md); transform: translateY(-1px); }
.status-chip.ok .chip-num { color: #34c759; }
.status-chip.bad .chip-num { color: #ff3b30; }
.status-chip.warn .chip-num { color: #ff9500; }
.chip-num { font-size: 28px; font-weight: 700; display: block; }
.chip-label { font-size: 12px; color: var(--el-text-color-secondary); margin-top: 2px; display: block; }
.chip-sub { font-size: 11px; color: var(--el-text-color-placeholder); display: block; }

/* Grid */
.dash-grid { display: grid; grid-template-columns: 1fr 260px; gap: 16px; }
.dash-main { display: flex; flex-direction: column; gap: 16px; }
.dash-side { display: flex; flex-direction: column; gap: 14px; }

/* Glass panel — the core frosted glass look */
.glass-panel {
  background: var(--glass-bg); backdrop-filter: var(--glass-blur);
  -webkit-backdrop-filter: var(--glass-blur);
  border: 1px solid rgba(255,255,255,0.5); border-top: 1px solid rgba(255,255,255,0.7);
  border-radius: 16px; padding: 16px;
  box-shadow: 0 1px 0 rgba(255,255,255,0.4) inset, var(--shadow-sm);
}
.panel-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.panel-head h3 { font-size: 14px; font-weight: 700; margin: 0; }
.panel-head a { font-size: 12px; font-weight: 600; color: var(--apple-blue); text-decoration: none; cursor: pointer; }
.panel-empty { font-size: 13px; color: var(--el-text-color-placeholder); padding: 20px 0; text-align: center; }
.panel-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; }

/* Alerts */
.alert-list { display: flex; flex-direction: column; }
.alert-row { display: flex; align-items: center; gap: 8px; padding: 7px 0; border-bottom: 1px solid rgba(0,0,0,0.04); cursor: pointer; font-size: 13px; }
.alert-row:last-child { border: none; }
.a-sev { font-size: 10px; font-weight: 700; padding: 1px 6px; border-radius: 4px; text-transform: uppercase; }
.a-sev.critical { background: #fee2e2; color: #dc2626; }
.a-sev.warning { background: #fef3c7; color: #d97706; }
.a-sev.info { background: #e0e7ff; color: #4f46e5; }
.a-name { flex: 1; font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.a-target { font-size: 12px; color: var(--el-text-color-secondary); max-width: 100px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.a-time { font-size: 11px; color: var(--el-text-color-placeholder); flex-shrink: 0; }

/* Activity */
.act-row { display: flex; align-items: center; gap: 8px; padding: 6px 0; border-bottom: 1px solid rgba(0,0,0,0.04); cursor: pointer; font-size: 12px; }
.act-row:last-child { border: none; }
.act-dot { width: 6px; height: 6px; border-radius: 50%; flex-shrink: 0; }
.act-dot.ok { background: #34c759; }
.act-dot.bad { background: #ff3b30; }
.act-dot.warn { background: #ff9500; }
.act-dot.ai { background: #7c3aed; }
.act-name { flex: 1; font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.act-stat { font-size: 10px; font-weight: 600; padding: 1px 6px; border-radius: 4px; text-transform: uppercase; }
.act-stat.success { background: #dcfce7; color: #16a34a; }
.act-stat.failed { background: #fee2e2; color: #dc2626; }
.act-stat.running { background: #dbeafe; color: #2563eb; }
.act-time { font-size: 11px; color: var(--el-text-color-placeholder); flex-shrink: 0; }

/* Quick links */
.quick-list { display: flex; flex-direction: column; gap: 2px; }
.ql-item { padding: 8px 10px; border-radius: 8px; cursor: pointer; font-size: 13px; font-weight: 500; color: var(--el-text-color-regular); }
.ql-item:hover { background: rgba(0,0,0,0.04); }

/* Health */
.health-list { display: flex; flex-direction: column; gap: 6px; }
.hl-row { display: flex; align-items: center; gap: 8px; font-size: 13px; }
.hl-dot { width: 6px; height: 6px; border-radius: 50%; flex-shrink: 0; }
.hl-dot.ok { background: #34c759; }
.hl-dot.bad { background: #ff3b30; }
.hl-label { flex: 1; color: var(--el-text-color-secondary); }
.hl-val { font-weight: 600; }

/* Oncall */
.oc-row { display: flex; align-items: center; gap: 8px; padding: 4px 0; font-size: 13px; font-weight: 600; }
</style>
