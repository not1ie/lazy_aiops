<template>
  <div class="dash">
    <!-- Header -->
    <div class="dash-top">
      <div>
        <h1>{{ greeting }}，{{ username }}</h1>
      </div>
      <div class="dash-top-right">
        <span class="dash-time">{{ nowStr }}</span>
        <el-button size="small" icon="Refresh" circle @click="fetchData" />
      </div>
    </div>

    <!-- Status Chips -->
    <div class="status-row">
      <div class="chip" :class="summary.hostOffline > 0 ? 'warn' : ''" @click="go('/asset')">
        <div class="chip-val">{{ summary.hostOnline }}<small>/{{ summary.hostTotal }}</small></div>
        <div class="chip-label">主机在线</div>
      </div>
      <div class="chip" :class="summary.alertOpen > 0 ? 'bad' : ''" @click="go('/monitor')">
        <div class="chip-val">{{ summary.alertOpen }}</div>
        <div class="chip-label">活跃告警</div>
      </div>
      <div class="chip" @click="go('/k8s')">
        <div class="chip-val">{{ summary.k8sTotal }}<small> / {{ summary.dockerTotal }}</small></div>
        <div class="chip-label">K8s / Docker</div>
      </div>
      <div class="chip" :class="pendingTickets > 0 ? 'warn' : ''" @click="go('/delivery')">
        <div class="chip-val">{{ pendingTickets }}</div>
        <div class="chip-label">待处理工单</div>
      </div>
    </div>

    <!-- Main content: timeline + side -->
    <div class="dash-body">
      <!-- Timeline feed -->
      <div class="timeline-panel glass">
        <div class="tl-head">
          <h3>事件时间线</h3>
          <a @click.prevent="go('/monitor')">查看全部 →</a>
        </div>
        <div v-if="timeline.length === 0" class="tl-empty">暂无事件</div>
        <div v-else class="tl-list">
          <div v-for="(ev, i) in timeline" :key="i" class="tl-item" @click="go(ev._path)">
            <div class="tl-marker" :class="ev._type"></div>
            <div class="tl-body">
              <div class="tl-title">
                <span class="tl-tag" :class="ev._type">{{ ev._tag }}</span>
                {{ ev._title }}
              </div>
              <div class="tl-meta">{{ ev._meta }}</div>
            </div>
            <div class="tl-time">{{ fmtRel(ev._time) }}</div>
          </div>
        </div>
      </div>

      <!-- Side -->
      <div class="dash-side">
        <div class="glass side-card">
          <h3>快捷入口</h3>
          <div class="ql-list">
            <div v-for="h in quickLinks" :key="h.path" class="ql-i" @click="go(h.path)">{{ h.label }}</div>
          </div>
        </div>
        <div class="glass side-card">
          <h3>系统健康</h3>
          <div class="hl-list">
            <div v-for="h in healthItems" :key="h.key" class="hl-i">
              <span class="hl-d" :class="h.ok ? 'ok' : 'bad'"></span>
              <span class="hl-l">{{ h.label }}</span>
              <span class="hl-v">{{ h.val }}</span>
            </div>
          </div>
        </div>
        <div class="glass side-card" v-if="oncallList.length">
          <h3>今日值班</h3>
          <div v-for="o in oncallList" :key="o.id||o.username" class="oc-i">
            <el-avatar :size="18" icon="UserFilled" />
            <span>{{ o.username||o.name }}</span>
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
let timer = null

const username = ref(localStorage.getItem('username') || 'Admin')
const summary = reactive({ hostTotal:0, hostOnline:0, hostOffline:0, k8sTotal:0, dockerTotal:0, alertOpen:0 })
const snapshots = reactive({ alerts:[], cicd_executions:[], workorders:[] })
const recentAIOps = ref([])
const oncallList = ref([])

const go = p => router.push(p)

const greeting = computed(() => {
  const h = new Date().getHours()
  if (h < 6) return '夜深了'
  if (h < 12) return '早上好'
  if (h < 18) return '下午好'
  return '晚上好'
})

const pendingTickets = computed(() => (snapshots.workorders||[]).filter(w => w.status==='pending'||w.status==='open').length)

// Build timeline from alerts + deployments + AIOps
const timeline = computed(() => {
  const items = []
  ;(snapshots.alerts||[]).filter(a => a.status==='firing').forEach(a => {
    items.push({
      _type: 'alert',
      _tag: a.severity==='critical'?'严重':a.severity==='warning'?'警告':'信息',
      _title: a.alert_name||a.rule_name||'告警',
      _meta: a.target||'',
      _time: a.fired_at||a.created_at,
      _path: '/monitor'
    })
  })
  ;(snapshots.cicd_executions||[]).slice(0,5).forEach(e => {
    items.push({
      _type: 'deploy',
      _tag: e.status==='success'?'成功':'失败',
      _title: e.pipeline_name||'Pipeline',
      _meta: e.status||'',
      _time: e.finished_at||e.created_at,
      _path: '/delivery'
    })
  })
  ;recentAIOps.value.forEach(inc => {
    items.push({
      _type: 'aiops',
      _tag: 'AIOps',
      _title: inc.title||inc.query?.slice(0,40)||'诊断',
      _meta: inc.status||'',
      _time: inc.created_at,
      _path: '/ai/ops'
    })
  })
  return items.sort((a,b) => new Date(b._time||0) - new Date(a._time||0)).slice(0, 12)
})

const quickLinks = [
  { label:'资产与安全', path:'/asset' },
  { label:'容器平台', path:'/k8s' },
  { label:'统一观测', path:'/monitor' },
  { label:'变更交付', path:'/delivery' },
  { label:'智能助手', path:'/ai' },
  { label:'系统治理', path:'/system' },
]

const healthItems = computed(() => [
  { key:'hosts', label:'主机', ok:summary.hostOffline===0, val:`${summary.hostOnline}/${summary.hostTotal}` },
  { key:'alerts', label:'告警', ok:summary.alertOpen===0, val:String(summary.alertOpen) },
  { key:'k8s', label:'K8s', ok:true, val:String(summary.k8sTotal) },
  { key:'docker', label:'Docker', ok:true, val:String(summary.dockerTotal) },
])

const fmtRel = v => {
  if (!v) return ''
  const d = Math.floor((Date.now()-new Date(v).getTime())/1000)
  if (d<60) return '刚刚'
  if (d<3600) return `${Math.floor(d/60)}分钟前`
  if (d<86400) return `${Math.floor(d/3600)}小时前`
  return `${Math.floor(d/86400)}天前`
}

const updateClock = () => {
  const n = new Date()
  nowStr.value = `${n.getFullYear()}/${n.getMonth()+1}/${n.getDate()} ${String(n.getHours()).padStart(2,'0')}:${String(n.getMinutes()).padStart(2,'0')}`
}

const fetchData = async () => {
  try {
    const h = { Authorization:'Bearer '+localStorage.getItem('token') }
    const res = await axios.get('/api/v1/dashboard/overview',{headers:h})
    const d = res.data?.data
    if (d?.summary) {
      summary.hostTotal = d.summary.host_total||0
      summary.hostOnline = d.summary.host_online||0
      summary.hostOffline = d.summary.host_offline||0
      summary.k8sTotal = d.summary.k8s_total||0
      summary.dockerTotal = d.summary.docker_total||0
      summary.alertOpen = d.summary.alert_open||0
    }
    if (d?.snapshots) {
      snapshots.alerts = d.snapshots.alerts||[]
      snapshots.cicd_executions = d.snapshots.cicd_executions||[]
      snapshots.workorders = d.snapshots.workorders||[]
    }
    const [ai,onc] = await Promise.all([
      axios.get('/api/v1/ai/ops/incidents',{headers:h}).catch(()=>({data:{}})),
      axios.get('/api/v1/oncall/whoisoncall',{headers:h}).catch(()=>({data:{}}))
    ])
    recentAIOps.value = (ai.data?.data||[]).slice(0,3)
    oncallList.value = onc.data?.data||[]
  } catch(e){}
}

onMounted(()=>{ fetchData(); updateClock(); setInterval(updateClock,30000); timer=setInterval(fetchData,30000) })
onUnmounted(()=>clearInterval(timer))
</script>

<style scoped>
.dash { max-width:1100px; margin:0 auto; padding:20px; }
.dash-top { display:flex; justify-content:space-between; align-items:flex-start; margin-bottom:20px; }
.dash-top h1 { font-size:22px; font-weight:700; margin:0; letter-spacing:-0.02em; }
.dash-top-right { display:flex; align-items:center; gap:8px; }
.dash-time { font-size:12px; color:var(--el-text-color-secondary); }

/* Chips */
.status-row { display:flex; gap:10px; margin-bottom:20px; }
.chip {
  flex:1; padding:12px 14px; border-radius:14px; cursor:pointer;
  background:var(--glass-bg); backdrop-filter:var(--glass-blur); -webkit-backdrop-filter:var(--glass-blur);
  border:1px solid rgba(255,255,255,0.5); box-shadow:var(--shadow-sm); transition:all .15s;
}
.chip:hover { box-shadow:var(--shadow-md); }
.chip.bad { border-left:3px solid #ff3b30; }
.chip.warn { border-left:3px solid #ff9500; }
.chip-val { font-size:26px; font-weight:700; color:var(--el-text-color-primary); }
.chip-val small { font-size:14px; font-weight:500; color:var(--el-text-color-secondary); }
.chip.bad .chip-val { color:#ff3b30; }
.chip.warn .chip-val { color:#ff9500; }
.chip-label { font-size:11px; color:var(--el-text-color-secondary); margin-top:2px; font-weight:500; }

/* Body */
.dash-body { display:grid; grid-template-columns:1fr 240px; gap:14px; }
.dash-side { display:flex; flex-direction:column; gap:12px; }

/* Glass card */
.glass {
  background:var(--glass-bg); backdrop-filter:var(--glass-blur); -webkit-backdrop-filter:var(--glass-blur);
  border:1px solid rgba(255,255,255,0.5); border-top:1px solid rgba(255,255,255,0.7);
  border-radius:16px; padding:16px;
  box-shadow:0 1px 0 rgba(255,255,255,0.4) inset, var(--shadow-sm);
}

/* Timeline */
.timeline-panel { padding:14px 16px; }
.tl-head { display:flex; justify-content:space-between; align-items:center; margin-bottom:12px; }
.tl-head h3 { font-size:13px; font-weight:700; margin:0; text-transform:uppercase; letter-spacing:0.04em; color:var(--el-text-color-secondary); }
.tl-head a { font-size:12px; font-weight:600; color:var(--apple-blue); text-decoration:none; cursor:pointer; }
.tl-empty { font-size:13px; color:var(--el-text-color-placeholder); text-align:center; padding:24px 0; }
.tl-list { display:flex; flex-direction:column; }
.tl-item { display:flex; gap:10px; padding:8px 0; cursor:pointer; border-bottom:1px solid rgba(0,0,0,0.04); align-items:flex-start; }
.tl-item:last-child { border:none; }
.tl-marker { width:8px; height:8px; border-radius:50%; margin-top:4px; flex-shrink:0; }
.tl-marker.alert { background:#ff3b30; }
.tl-marker.deploy { background:#34c759; }
.tl-marker.aiops { background:#7c3aed; }
.tl-body { flex:1; min-width:0; }
.tl-title { font-size:13px; font-weight:600; display:flex; align-items:center; gap:6px; }
.tl-tag { font-size:10px; font-weight:700; padding:0 4px; border-radius:3px; text-transform:uppercase; }
.tl-tag.alert { background:#fee2e2; color:#dc2626; }
.tl-tag.deploy { background:#dcfce7; color:#16a34a; }
.tl-tag.aiops { background:#ede9fe; color:#7c3aed; }
.tl-meta { font-size:11px; color:var(--el-text-color-secondary); margin-top:2px; }
.tl-time { font-size:11px; color:var(--el-text-color-placeholder); flex-shrink:0; margin-top:2px; }

/* Side */
.side-card h3 { font-size:12px; font-weight:700; margin:0 0 10px; text-transform:uppercase; letter-spacing:0.04em; color:var(--el-text-color-secondary); }
.ql-list { display:flex; flex-direction:column; gap:1px; }
.ql-i { padding:7px 8px; border-radius:6px; cursor:pointer; font-size:12px; font-weight:500; color:var(--el-text-color-regular); }
.ql-i:hover { background:rgba(0,0,0,0.04); }
.hl-list { display:flex; flex-direction:column; gap:5px; }
.hl-i { display:flex; align-items:center; gap:6px; font-size:12px; }
.hl-d { width:5px; height:5px; border-radius:50%; flex-shrink:0; }
.hl-d.ok { background:#34c759; }
.hl-d.bad { background:#ff3b30; }
.hl-l { flex:1; color:var(--el-text-color-secondary); }
.hl-v { font-weight:600; }
.oc-i { display:flex; align-items:center; gap:6px; padding:3px 0; font-size:12px; font-weight:600; }
</style>
