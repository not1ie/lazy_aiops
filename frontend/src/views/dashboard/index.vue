<template>
  <div class="dash" v-loading="loading">
    <!-- Top bar -->
    <div class="top">
      <h1>系统监控</h1>
      <div class="top-r">
        <span class="top-time">{{ nowStr }}</span>
        <el-switch v-model="autoRefresh" size="small" active-text="自动刷新" />
        <el-button size="small" icon="Refresh" circle @click="fetchData" />
      </div>
    </div>

    <!-- Aggregate metrics -->
    <div class="kpi-row">
      <div class="kpi">
        <div class="kpi-title">CPU 使用率</div>
        <div class="kpi-val" :class="(metrics.cpu||0) > 80 ? 'bad' : (metrics.cpu||0) > 60 ? 'warn' : ''">{{ (metrics.cpu||0).toFixed(1) }}%</div>
        <div class="kpi-bar"><span class="bar-fill" :class="(metrics.cpu||0) > 80 ? 'bad' : (metrics.cpu||0) > 60 ? 'warn' : ''" :style="{width: Math.min(metrics.cpu||0,100)+'%'}"></span></div>
      </div>
      <div class="kpi">
        <div class="kpi-title">内存使用率</div>
        <div class="kpi-val" :class="(metrics.memory||0) > 85 ? 'bad' : (metrics.memory||0) > 70 ? 'warn' : ''">{{ (metrics.memory||0).toFixed(1) }}%</div>
        <div class="kpi-bar"><span class="bar-fill" :class="(metrics.memory||0) > 85 ? 'bad' : (metrics.memory||0) > 70 ? 'warn' : ''" :style="{width: Math.min(metrics.memory||0,100)+'%'}"></span></div>
      </div>
      <div class="kpi">
        <div class="kpi-title">磁盘使用率</div>
        <div class="kpi-val" :class="(metrics.disk||0) > 85 ? 'bad' : (metrics.disk||0) > 70 ? 'warn' : ''">{{ (metrics.disk||0).toFixed(1) }}%</div>
        <div class="kpi-bar"><span class="bar-fill" :class="(metrics.disk||0) > 85 ? 'bad' : (metrics.disk||0) > 70 ? 'warn' : ''" :style="{width: Math.min(metrics.disk||0,100)+'%'}"></span></div>
      </div>
      <div class="kpi">
        <div class="kpi-title">在线 Agent</div>
        <div class="kpi-val">{{ agents.filter(a=>a.status==='online').length }}<small> / {{ agents.length }}</small></div>
        <div class="kpi-sub">{{ agents.filter(a=>a.status!=='online').length }} 离线</div>
      </div>
    </div>

    <!-- Main grid -->
    <div class="grid">
      <!-- Left: hosts + alerts -->
      <div class="main-col">
        <!-- Host status -->
        <div class="panel glass">
          <div class="panel-head">
            <h3>主机状态</h3>
            <a @click.prevent="go('/asset')">全部 →</a>
          </div>
          <div class="host-grid">
            <div class="host-stat ok" @click="go('/asset')">
              <span class="hs-num">{{ summary.hostOnline }}</span>
              <span class="hs-label">在线</span>
            </div>
            <div class="host-stat bad" @click="go('/asset')">
              <span class="hs-num">{{ summary.hostOffline }}</span>
              <span class="hs-label">离线</span>
            </div>
            <div class="host-stat warn">
              <span class="hs-num">{{ summary.hostStale }}</span>
              <span class="hs-label">超时未检测</span>
            </div>
            <div class="host-stat">
              <span class="hs-num">{{ summary.k8sTotal }}</span>
              <span class="hs-label">K8s 集群</span>
            </div>
          </div>
          <!-- Agent list -->
          <div v-if="agents.length > 0" class="agent-list">
            <div class="agent-row header">
              <span>主机</span><span>CPU</span><span>内存</span><span>磁盘</span><span>状态</span>
            </div>
            <div v-for="a in agents.slice(0, 8)" :key="a.id||a.agent_id" class="agent-row" @click="go('/monitor/agents')">
              <span class="a-name">{{ a.hostname||a.ip }}</span>
              <span class="a-metric" :class="(a.cpu||0)>80?'bad':(a.cpu||0)>60?'warn':''">{{ Math.round(a.cpu||0) }}%</span>
              <span class="a-metric" :class="(a.memory||0)>85?'bad':(a.memory||0)>70?'warn':''">{{ Math.round(a.memory||0) }}%</span>
              <span class="a-metric" :class="(a.disk||0)>85?'bad':(a.disk||0)>70?'warn':''">{{ Math.round(a.disk||0) }}%</span>
              <span class="a-status" :class="a.status">{{ a.status==='online'?'在线':'离线' }}</span>
            </div>
          </div>
          <div v-else class="panel-empty">暂无 Agent 数据</div>
        </div>

        <!-- Active alerts -->
        <div class="panel glass">
          <div class="panel-head">
            <h3>活跃告警</h3>
            <a @click.prevent="go('/monitor')">全部 →</a>
          </div>
          <div v-if="firingAlerts.length === 0" class="panel-empty">🎉 暂无活跃告警</div>
          <div v-else class="alert-list">
            <div v-for="a in firingAlerts.slice(0, 8)" :key="a.id" class="alert-row" @click="go('/monitor')">
              <span class="a-sev" :class="a.severity">{{ a.severity==='critical'?'严重':'警告' }}</span>
              <span class="a-name">{{ a.rule_name||a.alert_name }}</span>
              <span class="a-target">{{ a.target }}</span>
              <span class="a-time">{{ fmtRel(a.fired_at||a.created_at) }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Right: charts + health -->
      <div class="side-col">
        <!-- Metric history chart -->
        <div class="panel glass chart-panel" v-if="metricHistory.length > 1">
          <h3 class="chart-title">CPU 历史趋势</h3>
          <div class="chart-wrap">
            <svg viewBox="0 0 300 100" preserveAspectRatio="none" class="spark-chart">
              <defs>
                <linearGradient id="cpuGrad" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stop-color="rgba(0,113,227,0.2)"/>
                  <stop offset="100%" stop-color="rgba(0,113,227,0)"/>
                </linearGradient>
              </defs>
              <path :d="cpuArea" fill="url(#cpuGrad)"/>
              <path :d="cpuLine" fill="none" stroke="#0071e3" stroke-width="2" vector-effect="non-scaling-stroke"/>
            </svg>
            <div class="chart-ticks">
              <span v-for="(t,i) in chartTicks" :key="i">{{ t }}</span>
            </div>
          </div>
        </div>

        <!-- Docker hosts -->
        <div class="panel glass">
          <div class="panel-head">
            <h3>Docker 环境</h3>
            <a @click.prevent="go('/k8s')">全部 →</a>
          </div>
          <div v-if="dockerHosts.length === 0" class="panel-empty">无 Docker 环境</div>
          <div v-else class="docker-list">
            <div v-for="d in dockerHosts.slice(0,6)" :key="d.id" class="docker-row">
              <span class="d-dot" :class="d.status==='online'?'ok':'bad'"></span>
              <span class="d-name">{{ d.name||d.host_id }}</span>
              <span class="d-status" :class="d.status">{{ d.status==='online'?'在线':'离线' }}</span>
            </div>
          </div>
        </div>

        <!-- K8s clusters -->
        <div class="panel glass">
          <div class="panel-head">
            <h3>K8s 集群</h3>
            <a @click.prevent="go('/k8s')">全部 →</a>
          </div>
          <div v-if="k8sClusters.length === 0" class="panel-empty">无 K8s 集群</div>
          <div v-else>
            <div v-for="k in k8sClusters.slice(0,6)" :key="k.id" class="docker-row">
              <span class="d-dot" :class="k.status===1?'ok':'bad'"></span>
              <span class="d-name">{{ k.name }}</span>
              <span class="d-status" :class="k.status===1?'online':'offline'">{{ k.status===1?'在线':'离线' }}</span>
            </div>
          </div>
        </div>

        <!-- Quick actions -->
        <div class="panel glass">
          <h3 style="font-size:12px;font-weight:700;margin:0 0 8px;text-transform:uppercase;color:var(--el-text-color-secondary)">快捷操作</h3>
          <div class="quick-row">
            <span class="ql" @click="go('/monitor/hosts')">主机监控</span>
            <span class="ql" @click="go('/monitor/agents')">Agent 管理</span>
            <span class="ql" @click="go('/alert/rules')">告警规则</span>
            <span class="ql" @click="go('/monitor/metrics')">指标采集</span>
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
const autoRefresh = ref(true)
const nowStr = ref('')
let timer = null

const summary = reactive({ hostOnline:0, hostOffline:0, hostStale:0, k8sTotal:0, dockerTotal:0, dockerOnline:0, dockerOffline:0, alertOpen:0 })
const metrics = reactive({ cpu:0, memory:0, disk:0 })
const agents = ref([])
const firingAlerts = ref([])
const metricHistory = ref([])
const dockerHosts = ref([])
const k8sClusters = ref([])

const go = p => router.push(p)
const fmtRel = v => { if(!v) return ''; const d = Math.floor((Date.now()-new Date(v).getTime())/1000); if(d<60) return '刚刚'; if(d<3600) return Math.floor(d/60)+'分钟前'; if(d<86400) return Math.floor(d/3600)+'小时前'; return Math.floor(d/86400)+'天前' }

const cpuLine = computed(() => buildLine(metricHistory.value.map(m=>m.cpu_usage||0)))
const cpuArea = computed(() => cpuLine.value + ' L300,100 L0,100 Z')
const chartTicks = computed(() => {
  const h = metricHistory.value; if (h.length<2) return []
  const step = Math.max(1,Math.floor(h.length/5))
  return h.filter((_,i)=>i%step===0).map(m=>{
    const d = new Date(m.timestamp||m.at); return (d.getMonth()+1)+'/'+d.getDate()+' '+d.getHours()+':00'
  }).slice(0,6)
})

function buildLine(vals) {
  if (!vals.length) return ''
  const max = Math.max(...vals, 1)
  const step = 300 / (vals.length-1)
  return vals.map((v,i) => `${i===0?'M':'L'}${Math.round(i*step)},${Math.round(95-(v/max)*90)}`).join(' ')
}

const updateClock = () => { const n=new Date(); nowStr.value=n.getFullYear()+'/'+(n.getMonth()+1)+'/'+n.getDate()+' '+String(n.getHours()).padStart(2,'0')+':'+String(n.getMinutes()).padStart(2,'0') }

const fetchData = async () => {
  loading.value = true
  try {
    const h = { Authorization:'Bearer '+localStorage.getItem('token') }
    const res = await axios.get('/api/v1/dashboard/overview',{headers:h})
    const d = res.data?.data; if (!d) return
    if (d.summary) {
      summary.hostOnline = d.summary.host_online||0
      summary.hostOffline = d.summary.host_offline||0
      summary.hostStale = d.summary.host_stale||0
      summary.k8sTotal = d.summary.k8s_total||0
      summary.dockerTotal = d.summary.docker_total||0
      summary.dockerOnline = d.summary.docker_online||0
      summary.dockerOffline = d.summary.docker_offline||0
      summary.alertOpen = d.summary.alert_open||0
    }
    if (d.snapshots) {
      agents.value = d.snapshots.agents||[]
      firingAlerts.value = (d.snapshots.alerts||[]).filter(a=>a.status===0)
      metricHistory.value = d.snapshots.metric_history||[]
      dockerHosts.value = d.snapshots.docker_hosts||[]
      k8sClusters.value = d.snapshots.k8s_clusters||[]
      if (d.snapshots.metrics) { metrics.cpu=d.snapshots.metrics.cpu||0; metrics.memory=d.snapshots.metrics.memory||0; metrics.disk=d.snapshots.metrics.disk||0 }
    }
  } catch(e){} finally { loading.value=false }
}

onMounted(()=>{ fetchData(); updateClock(); setInterval(updateClock,30000); timer=setInterval(()=>{ if(autoRefresh.value) fetchData() },30000) })
onUnmounted(()=>clearInterval(timer))
</script>

<style scoped>
.dash { max-width:1200px; margin:0 auto; padding:16px 20px; }

.top { display:flex; justify-content:space-between; align-items:center; margin-bottom:16px; }
.top h1 { font-size:20px; font-weight:700; margin:0; }
.top-r { display:flex; align-items:center; gap:10px; }
.top-time { font-size:12px; color:var(--el-text-color-secondary); }

/* KPI */
.kpi-row { display:grid; grid-template-columns:repeat(4,1fr); gap:12px; margin-bottom:14px; }
.kpi { padding:12px 14px; border-radius:14px; background:var(--glass-bg); backdrop-filter:var(--glass-blur); -webkit-backdrop-filter:var(--glass-blur); border:1px solid rgba(255,255,255,0.5); box-shadow:var(--shadow-sm); }
.kpi-title { font-size:11px; color:var(--el-text-color-secondary); font-weight:600; text-transform:uppercase; letter-spacing:0.04em; }
.kpi-val { font-size:28px; font-weight:700; margin:4px 0; }
.kpi-val.bad { color:#ff3b30; }
.kpi-val.warn { color:#ff9500; }
.kpi-val small { font-size:14px; font-weight:500; color:var(--el-text-color-secondary); }
.kpi-bar { height:3px; background:rgba(0,0,0,0.06); border-radius:2px; margin-top:6px; }
.bar-fill { display:block; height:100%; border-radius:2px; background:#34c759; transition:width .5s; }
.bar-fill.warn { background:#ff9500; }
.bar-fill.bad { background:#ff3b30; }
.kpi-sub { font-size:11px; color:var(--el-text-color-secondary); }

/* Grid */
.grid { display:grid; grid-template-columns:1fr 320px; gap:12px; }
.main-col { display:flex; flex-direction:column; gap:12px; }
.side-col { display:flex; flex-direction:column; gap:10px; }

/* Glass panel */
.glass { background:var(--glass-bg); backdrop-filter:var(--glass-blur); -webkit-backdrop-filter:var(--glass-blur); border:1px solid rgba(255,255,255,0.5); border-top:1px solid rgba(255,255,255,0.7); border-radius:14px; padding:14px; box-shadow:0 1px 0 rgba(255,255,255,0.4) inset, var(--shadow-sm); }
.panel-head { display:flex; justify-content:space-between; align-items:center; margin-bottom:10px; }
.panel-head h3 { font-size:12px; font-weight:700; margin:0; text-transform:uppercase; letter-spacing:0.04em; color:var(--el-text-color-secondary); }
.panel-head a { font-size:11px; font-weight:600; color:var(--apple-blue); text-decoration:none; cursor:pointer; }
.panel-empty { font-size:12px; color:var(--el-text-color-placeholder); text-align:center; padding:16px 0; }

/* Host stats */
.host-grid { display:grid; grid-template-columns:repeat(4,1fr); gap:8px; margin-bottom:12px; }
.host-stat { padding:10px; border-radius:10px; background:rgba(0,0,0,0.02); text-align:center; cursor:pointer; }
.host-stat.ok { background:rgba(52,199,89,0.06); }
.host-stat.bad { background:rgba(255,59,48,0.06); }
.host-stat.warn { background:rgba(255,149,0,0.06); }
.hs-num { font-size:20px; font-weight:700; display:block; }
.hs-label { font-size:10px; color:var(--el-text-color-secondary); margin-top:2px; display:block; }

/* Agent list */
.agent-list { margin-top:8px; }
.agent-row { display:grid; grid-template-columns:2fr 1fr 1fr 1fr 1fr; gap:4px; padding:5px 8px; font-size:12px; border-bottom:1px solid rgba(0,0,0,0.04); cursor:pointer; align-items:center; }
.agent-row.header { font-size:10px; font-weight:700; color:var(--el-text-color-secondary); text-transform:uppercase; cursor:default; }
.a-name { font-weight:600; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
.a-metric { font-weight:600; text-align:center; }
.a-metric.bad { color:#ff3b30; }
.a-metric.warn { color:#ff9500; }
.a-status { font-size:11px; font-weight:600; }
.a-status.online { color:#34c759; }
.a-status.offline { color:#ff3b30; }

/* Alerts */
.alert-list { display:flex; flex-direction:column; }
.alert-row { display:flex; align-items:center; gap:6px; padding:6px 0; border-bottom:1px solid rgba(0,0,0,0.04); cursor:pointer; font-size:12px; }
.alert-row:last-child { border:none; }
.a-sev { font-size:10px; font-weight:700; padding:0 4px; border-radius:3px; }
.a-sev.critical { background:#fee2e2; color:#dc2626; }
.a-sev.warning { background:#fef3c7; color:#d97706; }
.a-name { font-weight:600; flex:1; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
.a-target { font-size:11px; color:var(--el-text-color-secondary); max-width:80px; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
.a-time { font-size:10px; color:var(--el-text-color-placeholder); flex-shrink:0; }

/* Chart */
.chart-panel { padding:12px 14px; }
.chart-title { font-size:11px; font-weight:700; margin:0 0 8px; text-transform:uppercase; color:var(--el-text-color-secondary); }
.chart-wrap { }
.spark-chart { width:100%; height:80px; }
.chart-ticks { display:flex; justify-content:space-between; font-size:9px; color:var(--el-text-color-placeholder); margin-top:4px; }

/* Docker/K8s */
.docker-list { display:flex; flex-direction:column; }
.docker-row { display:flex; align-items:center; gap:8px; padding:5px 0; border-bottom:1px solid rgba(0,0,0,0.03); font-size:12px; }
.d-dot { width:6px; height:6px; border-radius:50%; flex-shrink:0; }
.d-dot.ok { background:#34c759; }
.d-dot.bad { background:#ff3b30; }
.d-name { flex:1; font-weight:600; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
.d-status { font-size:11px; }
.d-status.online { color:#34c759; }
.d-status.offline { color:#ff3b30; }

/* Quick ops */
.quick-row { display:flex; flex-wrap:wrap; gap:6px; }
.ql { padding:4px 10px; border-radius:6px; font-size:11px; font-weight:600; color:var(--apple-blue); cursor:pointer; background:rgba(0,113,227,0.06); }
.ql:hover { background:rgba(0,113,227,0.12); }
</style>
