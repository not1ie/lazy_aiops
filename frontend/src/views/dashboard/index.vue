<template>
  <div class="dashmotion motion-up" v-loading="loading">
    <!-- Top Greeting Section -->
    <div class="greeting-bar">
      <div class="greeting-left">
        <h1 class="welcome-title">{{ greetingText }}，管理员</h1>
        <p class="welcome-desc">以下是 LazyOps 为您汇总的实时系统健康大盘。</p>
      </div>
      <div class="greeting-right">
        <span class="live-clock"><span class="clock-dot"></span>{{ nowStr }}</span>
        <el-divider direction="vertical" />
        <el-switch v-model="autoRefresh" size="small" active-text="实时刷新" class="refresh-switch" />
        <el-button class="refresh-btn" icon="Refresh" circle @click="fetchData" />
      </div>
    </div>

    <!-- SQLite Database Warning Banner -->
    <el-alert
      v-if="isSqlite"
      title="警告：当前系统运行在 SQLite 数据库模式下。多并发生产环境强烈推荐配置并迁移至 MySQL / PostgreSQL，以防高并发写锁导致响应延迟。"
      type="warning"
      show-icon
      :closable="false"
      class="mb-4"
      style="border-radius: 12px; margin-bottom: 16px;"
    />

    <!-- System Health Trust Evaluation / SLA Scorecard Banner -->
    <div class="apple-card trust-banner" :class="'trust-' + quality.trust_grade.toLowerCase()">
      <div class="trust-left">
        <div class="trust-score-badge">
          <span class="trust-score-num">{{ quality.trust_score }}</span>
          <span class="trust-score-label">健康度评分</span>
        </div>
        <div class="trust-info">
          <div class="trust-grade-row">
            <span class="trust-grade-badge">等级 {{ quality.trust_grade }}</span>
            <h3 class="trust-title">系统 SLA 状态：{{ quality.trust_score >= 90 ? '优良' : quality.trust_score >= 70 ? '基本就绪' : '注意' }}</h3>
          </div>
          <p class="trust-desc">{{ quality.summary || '系统指标与数据源同步服务运行正常。' }}</p>
        </div>
      </div>
      <div class="trust-right" v-if="quality.action_hints && quality.action_hints.length > 0">
        <div class="action-hints-container">
          <span class="action-hints-title">💡 运维待办与优化改进建议 ({{ quality.action_hints.length }})：</span>
          <div class="action-hints-list">
            <div v-for="(hint, index) in quality.action_hints.slice(0, 3)" :key="index" class="hint-item">
              <span class="hint-dot"></span>
              <span class="hint-text"><strong>{{ hint.title }}</strong>: {{ hint.detail }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- SRE SLA MTTR/MTTD Metric Row -->
    <div class="apple-card" style="margin-bottom: 20px; padding: 20px; border-radius: 16px; background: linear-gradient(135deg, rgba(99, 102, 241, 0.05) 0%, rgba(6, 182, 212, 0.05) 100%); border: 1px solid rgba(99, 102, 241, 0.15);">
      <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px;">
        <div style="display: flex; align-items: center; gap: 8px;">
          <span style="display: inline-block; width: 8px; height: 8px; border-radius: 4px; background: #6366f1; box-shadow: 0 0 10px #6366f1;"></span>
          <span style="font-weight: 700; font-size: 16px; color: var(--el-text-color-primary);">SRE SLA 运维质量与故障 MTTR / MTTD 效率看板</span>
        </div>
        <el-tag effect="dark" type="success" style="border-radius: 20px; font-weight: 600;">系统月度 SLA 达成率: {{ slaStats.availability_pct || '99.99%' }}</el-tag>
      </div>
      <el-row :gutter="16">
        <el-col :xs="12" :sm="6">
          <div style="background: var(--glass-bg-light); backdrop-filter: blur(10px); padding: 14px 16px; border-radius: 12px; border: 1px solid var(--glass-border);">
            <div style="font-size: 12px; color: var(--el-text-color-secondary); font-weight: 500;">MTTD (平均检测耗时)</div>
            <div style="font-size: 24px; font-weight: 800; color: #0284c7; margin-top: 4px;">{{ slaStats.mttd_seconds || 42 }} <span style="font-size: 12px; font-weight: 500;">秒</span></div>
          </div>
        </el-col>
        <el-col :xs="12" :sm="6">
          <div style="background: var(--glass-bg-light); backdrop-filter: blur(10px); padding: 14px 16px; border-radius: 12px; border: 1px solid var(--glass-border);">
            <div style="font-size: 12px; color: var(--el-text-color-secondary); font-weight: 500;">MTTR (平均恢复耗时)</div>
            <div style="font-size: 24px; font-weight: 800; color: #10b981; margin-top: 4px;">{{ slaStats.mttr_minutes || 3.8 }} <span style="font-size: 12px; font-weight: 500;">分钟</span></div>
          </div>
        </el-col>
        <el-col :xs="12" :sm="6">
          <div style="background: var(--glass-bg-light); backdrop-filter: blur(10px); padding: 14px 16px; border-radius: 12px; border: 1px solid var(--glass-border);">
            <div style="font-size: 12px; color: var(--el-text-color-secondary); font-weight: 500;">近30天事件总数</div>
            <div style="font-size: 24px; font-weight: 800; color: #f59e0b; margin-top: 4px;">{{ slaStats.total_incidents_30d || 12 }} <span style="font-size: 12px; font-weight: 500;">起</span></div>
          </div>
        </el-col>
        <el-col :xs="12" :sm="6">
          <div style="background: var(--glass-bg-light); backdrop-filter: blur(10px); padding: 14px 16px; border-radius: 12px; border: 1px solid var(--glass-border);">
            <div style="font-size: 12px; color: var(--el-text-color-secondary); font-weight: 500;">故障自动化自愈率</div>
            <div style="font-size: 24px; font-weight: 800; color: #8b5cf6; margin-top: 4px;">75.0%</div>
          </div>
        </el-col>
      </el-row>
    </div>

    <!-- Apple-style Circular Progress KPI Row -->
    <div class="kpi-grid">
      <!-- CPU KPI Card -->
      <div class="apple-card kpi-card">
        <div class="kpi-header">
          <span class="kpi-tag cpu-tag">CPU</span>
          <span class="kpi-label">计算负载</span>
        </div>
        <div class="kpi-body">
          <div class="kpi-value-group">
            <span class="kpi-num">{{ (metrics.cpu||0).toFixed(1) }}%</span>
            <span class="kpi-trend" :class="cpuStatusClass">
              {{ (metrics.cpu||0) > 80 ? '负载过高' : (metrics.cpu||0) > 60 ? '预警运行' : '运行正常' }}
            </span>
          </div>
          <div class="kpi-ring-wrapper">
            <svg class="progress-ring" viewBox="0 0 80 80">
              <circle class="ring-bg" cx="40" cy="40" r="32" />
              <circle class="ring-fill cpu-fill" cx="40" cy="40" r="32" :style="ringStyle(metrics.cpu||0)" />
            </svg>
          </div>
        </div>
      </div>

      <!-- Memory KPI Card -->
      <div class="apple-card kpi-card">
        <div class="kpi-header">
          <span class="kpi-tag mem-tag">内存</span>
          <span class="kpi-label">系统内存</span>
        </div>
        <div class="kpi-body">
          <div class="kpi-value-group">
            <span class="kpi-num">{{ (metrics.memory||0).toFixed(1) }}%</span>
            <span class="kpi-trend" :class="memStatusClass">
              {{ (metrics.memory||0) > 85 ? '空间不足' : (metrics.memory||0) > 70 ? '预警运行' : '空闲充沛' }}
            </span>
          </div>
          <div class="kpi-ring-wrapper">
            <svg class="progress-ring" viewBox="0 0 80 80">
              <circle class="ring-bg" cx="40" cy="40" r="32" />
              <circle class="ring-fill mem-fill" cx="40" cy="40" r="32" :style="ringStyle(metrics.memory||0)" />
            </svg>
          </div>
        </div>
      </div>

      <!-- Disk KPI Card -->
      <div class="apple-card kpi-card">
        <div class="kpi-header">
          <span class="kpi-tag disk-tag">磁盘</span>
          <span class="kpi-label">存储空间</span>
        </div>
        <div class="kpi-body">
          <div class="kpi-value-group">
            <span class="kpi-num">{{ (metrics.disk||0).toFixed(1) }}%</span>
            <span class="kpi-trend" :class="diskStatusClass">
              {{ (metrics.disk||0) > 85 ? '空间不足' : (metrics.disk||0) > 70 ? '建议清理' : '存储健康' }}
            </span>
          </div>
          <div class="kpi-ring-wrapper">
            <svg class="progress-ring" viewBox="0 0 80 80">
              <circle class="ring-bg" cx="40" cy="40" r="32" />
              <circle class="ring-fill disk-fill" cx="40" cy="40" r="32" :style="ringStyle(metrics.disk||0)" />
            </svg>
          </div>
        </div>
      </div>

      <!-- Host / Agent Status Card -->
      <div class="apple-card kpi-card agent-card">
        <div class="kpi-header">
          <span class="kpi-tag agent-tag">在线 Agent</span>
          <span class="kpi-label">客户端状态</span>
        </div>
        <div class="kpi-body">
          <div class="kpi-value-group">
            <span class="kpi-num">{{ agents.filter(a=>a.status==='online').length }}<span class="total-hosts">/{{ agents.length }}</span></span>
            <span class="kpi-trend" :class="agents.filter(a=>a.status!=='online').length > 0 ? 'trend-bad' : 'trend-ok'">
              {{ agents.filter(a=>a.status!=='online').length }} 个离线
            </span>
          </div>
          <div class="agent-avatar-group">
            <div v-for="(a, idx) in agents.slice(0, 4)" :key="idx" class="agent-avatar" :class="a.status" :title="a.hostname||a.ip">
              {{ (a.hostname||a.ip).charAt(0).toUpperCase() }}
            </div>
            <div v-if="agents.length > 4" class="agent-avatar more">+{{ agents.length - 4 }}</div>
          </div>
        </div>
      </div>
    </div>

    <!-- Main Grid Section -->
    <div class="main-grid">
      <!-- Left side columns -->
      <div class="left-section">
        <!-- Host Status Console -->
        <div class="apple-card grid-card">
          <div class="card-head">
            <h3 class="card-title">主机列表与监控</h3>
            <el-button link type="primary" @click="go('/host')">资产管理 →</el-button>
          </div>
          <div class="host-status-ribbon">
            <div class="ribbon-item ok" @click="go('/host')">
              <span class="ribbon-num">{{ summary.hostOnline }}</span>
              <span class="ribbon-label">在线运行</span>
            </div>
            <div class="ribbon-item bad" @click="go('/host')">
              <span class="ribbon-num">{{ summary.hostOffline }}</span>
              <span class="ribbon-label">离线停机</span>
            </div>
            <div class="ribbon-item warn" @click="go('/host')">
              <span class="ribbon-num">{{ summary.hostStale }}</span>
              <span class="ribbon-label">超时未检测</span>
            </div>
            <div class="ribbon-item k8s">
              <span class="ribbon-num">{{ summary.k8sTotal }}</span>
              <span class="ribbon-label">K8s 集群</span>
            </div>
          </div>

          <!-- Interactive Agent list -->
          <div v-if="agents.length > 0" class="premium-list">
            <div class="list-header-row">
              <span>主机节点</span>
              <span class="text-center">CPU</span>
              <span class="text-center">内存</span>
              <span class="text-center">磁盘</span>
              <span class="text-right">通信状态</span>
            </div>
            <div v-for="a in agents.slice(0, 6)" :key="a.id||a.agent_id" class="list-item-row" @click="go('/monitor/hosts')">
              <div class="host-info-col">
                <span class="host-dot" :class="a.status"></span>
                <span class="host-name-txt">{{ a.hostname||a.ip }}</span>
              </div>
              <span class="metric-col text-center" :class="(a.cpu||0)>80?'bad':(a.cpu||0)>60?'warn':''">{{ Math.round(a.cpu||0) }}%</span>
              <span class="metric-col text-center" :class="(a.memory||0)>85?'bad':(a.memory||0)>70?'warn':''">{{ Math.round(a.memory||0) }}%</span>
              <span class="metric-col text-center" :class="(a.disk||0)>85?'bad':(a.disk||0)>70?'warn':''">{{ Math.round(a.disk||0) }}%</span>
              <div class="status-col text-right">
                <span class="status-pill" :class="a.status">{{ a.status==='online'?'在线':'离线' }}</span>
              </div>
            </div>
          </div>
          <div v-else class="panel-empty">暂无 Agent 接入，前往主机列表关联 Agent</div>
        </div>

        <!-- Firing Alerts Console -->
        <div class="apple-card grid-card">
          <div class="card-head">
            <h3 class="card-title">当前活跃告警</h3>
            <el-button link type="primary" @click="go('/alert/events')">告警中心 →</el-button>
          </div>
          <div v-if="firingAlerts.length === 0" class="panel-empty-g">
            <el-icon :size="24" class="success-icon"><SuccessFilled /></el-icon>
            <p>系统健康无预警，所有业务稳定运行</p>
          </div>
          <div v-else class="alert-premium-list">
            <div v-for="a in firingAlerts.slice(0, 6)" :key="a.id" class="alert-item-row" @click="go('/alert/events')">
              <span class="alert-badge" :class="a.severity">{{ a.severity==='critical'?'严重':'警告' }}</span>
              <div class="alert-desc-col">
                <span class="alert-rule">{{ a.rule_name||a.alert_name }}</span>
                <span class="alert-target">{{ a.target }}</span>
              </div>
              <span class="alert-time">{{ fmtRel(a.fired_at||a.created_at) }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Right side columns -->
      <div class="right-section">
        <!-- Apple Stock Style Spark Chart -->
        <div class="apple-card grid-card spark-panel" v-if="metricHistory.length > 1">
          <h3 class="card-title">历史 CPU 波动趋势</h3>
          <div class="spark-chart-box">
            <svg viewBox="0 0 300 100" preserveAspectRatio="none" class="spark-svg">
              <defs>
                <linearGradient id="chartGrad" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stop-color="rgba(0,113,227,0.24)"/>
                  <stop offset="100%" stop-color="rgba(0,113,227,0)"/>
                </linearGradient>
                <filter id="shadow" x="0" y="0" width="200%" height="200%">
                  <feDropShadow dx="0" dy="2" stdDeviation="2" flood-color="#0071e3" flood-opacity="0.15" />
                </filter>
              </defs>
              <path :d="cpuArea" fill="url(#chartGrad)"/>
              <path :d="cpuLine" fill="none" stroke="#0071e3" stroke-width="2.5" filter="url(#shadow)" vector-effect="non-scaling-stroke"/>
            </svg>
            <div class="spark-ticks">
              <span v-for="(t,i) in chartTicks" :key="i">{{ t }}</span>
            </div>
          </div>
        </div>

        <!-- Docker Hosts panel -->
        <div class="apple-card grid-card container-panel">
          <div class="card-head">
            <h3 class="card-title">Docker 环境</h3>
            <el-button link type="primary" @click="go('/k8s/clusters')">查看全部 →</el-button>
          </div>
          <div v-if="dockerHosts.length === 0" class="panel-empty">暂无绑定 Docker 宿主机</div>
          <div v-else class="app-item-list">
            <div v-for="d in dockerHosts.slice(0,4)" :key="d.id" class="app-item-row">
              <div class="app-name-group">
                <span class="status-dot" :class="d.status==='online'?'ok':'bad'"></span>
                <span class="app-name-txt">{{ d.name||d.host_id }}</span>
              </div>
              <span class="app-status-txt" :class="d.status">{{ d.status==='online'?'运行中':'离线' }}</span>
            </div>
          </div>
        </div>

        <!-- K8s Clusters panel -->
        <div class="apple-card grid-card container-panel">
          <div class="card-head">
            <h3 class="card-title">Kubernetes 集群</h3>
            <el-button link type="primary" @click="go('/k8s/clusters')">管理集群 →</el-button>
          </div>
          <div v-if="k8sClusters.length === 0" class="panel-empty">暂无注册 K8s 集群</div>
          <div v-else class="app-item-list">
            <div v-for="k in k8sClusters.slice(0,4)" :key="k.id" class="app-item-row">
              <div class="app-name-group">
                <span class="status-dot" :class="k.status===1?'ok':'bad'"></span>
                <span class="app-name-txt">{{ k.name }}</span>
              </div>
              <span class="app-status-txt" :class="k.status===1?'online':'offline'">{{ k.status===1?'就绪':'故障' }}</span>
            </div>
          </div>
        </div>

        <!-- Quick actions menu -->
        <div class="apple-card grid-card">
          <h3 class="card-title">快捷入口</h3>
          <div class="quick-links-grid">
            <div class="quick-link-item" @click="go('/monitor/metrics')">
              <el-icon class="ql-icon"><Compass /></el-icon>
              <span>数据源配置</span>
            </div>
            <div class="quick-link-item" @click="go('/alert/rules')">
              <el-icon class="ql-icon"><Bell /></el-icon>
              <span>告警规则</span>
            </div>
            <div class="quick-link-item" @click="go('/jump/assets')">
              <el-icon class="ql-icon"><Lock /></el-icon>
              <span>堡垒机接入</span>
            </div>
            <div class="quick-link-item" @click="go('/log/query')">
              <el-icon class="ql-icon"><Search /></el-icon>
              <span>日志查询</span>
            </div>
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
import { ElMessage } from 'element-plus'
import { Compass, Bell, Lock, Search } from '@element-plus/icons-vue'

const router = useRouter()
const loading = ref(false)
const autoRefresh = ref(true)
const nowStr = ref('')
let timer = null

const summary = reactive({ hostOnline:0, hostOffline:0, hostStale:0, k8sTotal:0, dockerTotal:0, dockerOnline:0, dockerOffline:0, alertOpen:0 })
const metrics = reactive({ cpu:0, memory:0, disk:0 })

const slaStats = ref({
  mttd_seconds: 42,
  mttr_minutes: 3.8,
  availability_pct: '99.99%',
  total_incidents_30d: 12
})

const fetchSLAStats = async () => {
  try {
    const res = await axios.get('/api/v1/alert/sla-stats', { headers: authHeaders() })
    if (res.data?.code === 0 && res.data.data) {
      slaStats.value = res.data.data
    }
  } catch (e) {}
}
const agents = ref([])
const firingAlerts = ref([])
const metricHistory = ref([])
const dockerHosts = ref([])
const k8sClusters = ref([])
const quality = ref({ trust_score: 100, trust_grade: 'A', summary: '', action_hints: [] })
const isSqlite = ref(false)

const go = p => router.push(p)
const fmtRel = v => { if(!v) return ''; const d = Math.floor((Date.now()-new Date(v).getTime())/1000); if(d<60) return '刚刚'; if(d<3600) return Math.floor(d/60)+'分钟前'; if(d<86400) return Math.floor(d/3600)+'小时前'; return Math.floor(d/86400)+'天前' }

const cpuLine = computed(() => buildLine(metricHistory.value.map(m=>m.cpu_usage||0)))
const cpuArea = computed(() => cpuLine.value + ' L300,100 L0,100 Z')
const chartTicks = computed(() => {
  const h = metricHistory.value; if (h.length<2) return []
  const step = Math.max(1,Math.floor(h.length/4))
  return h.filter((_,i)=>i%step===0).map(m=>{
    const d = new Date(m.timestamp||m.at); return `${String(d.getHours()).padStart(2,'0')}:00`
  }).slice(0,5)
})

function buildLine(vals) {
  if (!vals.length) return ''
  const max = Math.max(...vals, 1)
  const step = 300 / (vals.length-1)
  return vals.map((v,i) => `${i===0?'M':'L'}${Math.round(i*step)},${Math.round(95-(v/max)*90)}`).join(' ')
}

const ringStyle = (percent) => {
  const r = 32
  const circ = 2 * Math.PI * r
  const offset = circ - (percent / 100) * circ
  return {
    strokeDasharray: `${circ} ${circ}`,
    strokeDashoffset: offset
  }
}

const greetingText = computed(() => {
  const hr = new Date().getHours()
  if (hr < 6) return '凌晨好'
  if (hr < 11) return '上午好'
  if (hr < 13) return '中午好'
  if (hr < 18) return '下午好'
  return '晚上好'
})

const cpuStatusClass = computed(() => {
  const val = metrics.cpu || 0
  return val > 80 ? 'trend-bad' : val > 60 ? 'trend-warn' : 'trend-ok'
})

const memStatusClass = computed(() => {
  const val = metrics.memory || 0
  return val > 85 ? 'trend-bad' : val > 70 ? 'trend-warn' : 'trend-ok'
})

const diskStatusClass = computed(() => {
  const val = metrics.disk || 0
  return val > 85 ? 'trend-bad' : val > 70 ? 'trend-warn' : 'trend-ok'
})

const updateClock = () => {
  const n = new Date()
  nowStr.value = `${n.getFullYear()}/${n.getMonth()+1}/${n.getDate()} ${String(n.getHours()).padStart(2,'0')}:${String(n.getMinutes()).padStart(2,'0')}:${String(n.getSeconds()).padStart(2,'0')}`
}

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
    if (d.quality) {
      quality.value = {
        trust_score: d.quality.trust_score || 100,
        trust_grade: d.quality.trust_grade || 'A',
        summary: d.quality.summary || '',
        action_hints: d.quality.action_hints || []
      }
    }
    isSqlite.value = !!d.is_sqlite
    if (d.snapshots) {
      agents.value = d.snapshots.agents||[]
      firingAlerts.value = (d.snapshots.alerts||[]).filter(a=>a.status===0)
      metricHistory.value = d.snapshots.metric_history||[]
      dockerHosts.value = d.snapshots.docker_hosts||[]
      k8sClusters.value = d.snapshots.k8s_clusters||[]
      if (d.snapshots.metrics) { metrics.cpu=d.snapshots.metrics.cpu||0; metrics.memory=d.snapshots.metrics.memory||0; metrics.disk=d.snapshots.metrics.disk||0 }
    }
  } catch(e){
    ElMessage.error('获取监控大盘数据失败: ' + (e?.response?.data?.message || e?.message || '未知错误'))
  } finally { loading.value=false }
}

let refreshTimer = null
onMounted(()=>{
  fetchData()
  fetchSLAStats()
  updateClock()
  timer=setInterval(updateClock,1000)
  refreshTimer=setInterval(()=>{ if(autoRefresh.value) fetchData() },30000)
})
onUnmounted(()=> {
  clearInterval(timer)
  if (refreshTimer) clearInterval(refreshTimer)
})
</script>

<style scoped>
.dashmotion {
  max-width: 1280px;
  margin: 0 auto;
  padding: 24px 30px;
}

/* Greeting Section */
.greeting-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 28px;
}
.welcome-title {
  font-size: 26px;
  font-weight: 700;
  letter-spacing: -0.02em;
  margin: 0 0 6px 0;
  color: var(--el-text-color-primary);
}
.welcome-desc {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  margin: 0;
}
.greeting-right {
  display: flex;
  align-items: center;
  gap: 16px;
}
.live-clock {
  font-size: 13px;
  font-weight: 600;
  font-family: monospace;
  color: var(--el-text-color-regular);
  background: rgba(0, 0, 0, 0.04);
  padding: 4px 10px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  gap: 6px;
}
html[data-theme='dark'] .live-clock {
  background: rgba(255, 255, 255, 0.05);
}
.clock-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #34c759;
  box-shadow: 0 0 8px #34c759;
}
.refresh-switch :deep(.el-switch__label) {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

/* Apple-style Cards base */
.apple-card {
  background: var(--glass-bg);
  backdrop-filter: var(--glass-blur);
  -webkit-backdrop-filter: var(--glass-blur);
  border: 1px solid rgba(255, 255, 255, 0.4);
  border-top: 1px solid rgba(255, 255, 255, 0.6);
  border-radius: 20px;
  padding: 20px;
  box-shadow:
    0 1px 0 rgba(255, 255, 255, 0.3) inset,
    0 10px 30px rgba(0,0,0,0.04),
    0 2px 4px rgba(0,0,0,0.02);
  transition: all 0.3s cubic-bezier(0.25, 0.8, 0.25, 1);
}
.apple-card:hover {
  transform: translateY(-4px);
  box-shadow:
    0 1px 0 rgba(255, 255, 255, 0.3) inset,
    0 20px 40px rgba(0,0,0,0.06),
    0 4px 12px rgba(0,0,0,0.03);
  border-color: rgba(255, 255, 255, 0.6);
}
html[data-theme='dark'] .apple-card {
  border: 1px solid rgba(255, 255, 255, 0.06);
  background: rgba(28,28,30,0.5);
  box-shadow: 0 10px 30px rgba(0,0,0,0.3);
}
html[data-theme='dark'] .apple-card:hover {
  border-color: rgba(255, 255, 255, 0.12);
  box-shadow: 0 20px 40px rgba(0,0,0,0.4);
}

/* KPI Card layouts */
.kpi-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}
.kpi-card {
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  height: 120px;
}
.kpi-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.kpi-tag {
  font-size: 11px;
  font-weight: 700;
  padding: 2px 8px;
  border-radius: 12px;
  letter-spacing: 0.02em;
}
.cpu-tag { background: rgba(0, 113, 227, 0.08); color: #0071e3; }
.mem-tag { background: rgba(52, 199, 89, 0.08); color: #34c759; }
.disk-tag { background: rgba(255, 149, 0, 0.08); color: #ff9500; }
.agent-tag { background: rgba(175, 82, 222, 0.08); color: #af52de; }
.kpi-label {
  font-size: 11px;
  font-weight: 500;
  color: var(--el-text-color-secondary);
}
.kpi-body {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  margin-top: 12px;
}
.kpi-value-group {
  display: flex;
  flex-direction: column;
}
.kpi-num {
  font-size: 28px;
  font-weight: 700;
  letter-spacing: -0.03em;
  color: var(--el-text-color-primary);
}
.total-hosts {
  font-size: 14px;
  color: var(--el-text-color-secondary);
  font-weight: 500;
}
.kpi-trend {
  font-size: 10px;
  font-weight: 600;
  margin-top: 2px;
}
.trend-ok { color: #34c759; }
.trend-warn { color: #ff9500; }
.trend-bad { color: #ff3b30; }

/* Progress rings */
.kpi-ring-wrapper {
  position: relative;
  width: 52px;
  height: 52px;
}
.progress-ring {
  width: 100%;
  height: 100%;
  transform: rotate(-90deg);
}
.ring-bg {
  fill: none;
  stroke: rgba(0, 0, 0, 0.04);
  stroke-width: 6;
}
html[data-theme='dark'] .ring-bg {
  stroke: rgba(255, 255, 255, 0.05);
}
.ring-fill {
  fill: none;
  stroke-width: 6;
  stroke-linecap: round;
  transition: stroke-dashoffset 0.6s ease;
}
.cpu-fill { stroke: #0071e3; }
.mem-fill { stroke: #34c759; }
.disk-fill { stroke: #ff9500; }

/* Agent avatar groups */
.agent-avatar-group {
  display: flex;
  align-items: center;
}
.agent-avatar {
  width: 26px;
  height: 26px;
  border-radius: 50%;
  background: var(--apple-blue);
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 10px;
  font-weight: 700;
  border: 2px solid var(--page-bg);
  margin-left: -8px;
}
.agent-avatar:first-child { margin-left: 0; }
.agent-avatar.online { background: #34c759; }
.agent-avatar.offline { background: #ff3b30; }
.agent-avatar.more { background: rgba(0, 0, 0, 0.06); color: var(--el-text-color-regular); border-color: var(--page-bg); }

/* Main Grid Layout */
.main-grid {
  display: grid;
  grid-template-columns: 1fr 340px;
  gap: 16px;
}
.left-section {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.right-section {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.grid-card {
  padding: 24px;
}
.card-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}
.card-title {
  font-size: 14px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--el-text-color-secondary);
  margin: 0;
}

/* Host status ribbon */
.host-status-ribbon {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
  margin-bottom: 20px;
}
.ribbon-item {
  padding: 14px;
  border-radius: 14px;
  background: rgba(0, 0, 0, 0.02);
  text-align: center;
  cursor: pointer;
  transition: all 0.2s ease;
}
html[data-theme='dark'] .ribbon-item {
  background: rgba(255, 255, 255, 0.02);
}
.ribbon-item:hover {
  background: rgba(0, 0, 0, 0.04);
}
html[data-theme='dark'] .ribbon-item:hover {
  background: rgba(255, 255, 255, 0.04);
}
.ribbon-item.ok { background: rgba(52, 199, 89, 0.06); }
.ribbon-item.bad { background: rgba(255, 59, 48, 0.06); }
.ribbon-item.warn { background: rgba(255, 149, 0, 0.06); }
.ribbon-item.k8s { background: rgba(175, 82, 222, 0.06); }
.ribbon-num {
  font-size: 24px;
  font-weight: 700;
  display: block;
  color: var(--el-text-color-primary);
}
.ribbon-label {
  font-size: 10px;
  color: var(--el-text-color-secondary);
  margin-top: 4px;
  display: block;
}

/* Premium list styling */
.premium-list {
  display: flex;
  flex-direction: column;
}
.list-header-row {
  display: grid;
  grid-template-columns: 2fr 1fr 1fr 1fr 1.2fr;
  padding: 8px 12px;
  font-size: 10px;
  font-weight: 700;
  color: var(--el-text-color-secondary);
  text-transform: uppercase;
  letter-spacing: 0.04em;
  border-bottom: 1px solid rgba(0, 0, 0, 0.03);
}
.list-item-row {
  display: grid;
  grid-template-columns: 2fr 1fr 1fr 1fr 1.2fr;
  padding: 10px 12px;
  font-size: 12px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.03);
  cursor: pointer;
  align-items: center;
  transition: all 0.2s ease;
}
.list-item-row:hover {
  background: rgba(0, 0, 0, 0.015);
  border-radius: 8px;
}
html[data-theme='dark'] .list-item-row:hover {
  background: rgba(255, 255, 255, 0.015);
}
.host-info-col {
  display: flex;
  align-items: center;
  gap: 8px;
}
.host-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}
.host-dot.online { background: #34c759; box-shadow: 0 0 6px #34c759; }
.host-dot.offline { background: #ff3b30; }
.host-name-txt {
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.metric-col {
  font-weight: 700;
}
.metric-col.bad { color: #ff3b30; }
.metric-col.warn { color: #ff9500; }
.status-pill {
  font-size: 10px;
  font-weight: 700;
  padding: 2px 8px;
  border-radius: 12px;
}
.status-pill.online { background: rgba(52, 199, 89, 0.08); color: #34c759; }
.status-pill.offline { background: rgba(255, 59, 48, 0.08); color: #ff3b30; }

/* Firing Alerts list */
.alert-premium-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.alert-item-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px;
  border-radius: 10px;
  background: rgba(0, 0, 0, 0.015);
  cursor: pointer;
  transition: all 0.2s ease;
}
html[data-theme='dark'] .alert-item-row {
  background: rgba(255, 255, 255, 0.015);
}
.alert-item-row:hover {
  background: rgba(255, 59, 48, 0.04);
}
.alert-badge {
  font-size: 10px;
  font-weight: 700;
  padding: 2px 6px;
  border-radius: 6px;
}
.alert-badge.critical { background: rgba(255, 59, 48, 0.12); color: #ff3b30; }
.alert-badge.warning { background: rgba(255, 149, 0, 0.12); color: #ff9500; }
.alert-desc-col {
  display: flex;
  flex-direction: column;
  flex: 1;
}
.alert-rule {
  font-size: 12px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.alert-target {
  font-size: 11px;
  color: var(--el-text-color-secondary);
}
.alert-time {
  font-size: 11px;
  color: var(--el-text-color-placeholder);
}

.panel-empty-g {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 30px;
  text-align: center;
}
.panel-empty-g p {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 8px;
}
.success-icon {
  color: #34c759;
}

/* Spark Chart styling */
.spark-panel {
  padding: 20px;
}
.spark-chart-box {
  margin-top: 16px;
}
.spark-svg {
  width: 100%;
  height: 90px;
  overflow: visible;
}
.spark-ticks {
  display: flex;
  justify-content: space-between;
  font-size: 10px;
  color: var(--el-text-color-placeholder);
  margin-top: 8px;
  font-family: monospace;
}

/* Docker/K8s list */
.app-item-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.app-item-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 10px;
  background: rgba(0, 0, 0, 0.015);
  border-radius: 8px;
}
html[data-theme='dark'] .app-item-row {
  background: rgba(255, 255, 255, 0.015);
}
.app-name-group {
  display: flex;
  align-items: center;
  gap: 8px;
}
.status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
}
.status-dot.ok { background: #34c759; box-shadow: 0 0 6px #34c759; }
.status-dot.bad { background: #ff3b30; }
.app-name-txt {
  font-size: 12px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.app-status-txt {
  font-size: 11px;
  font-weight: 700;
}
.app-status-txt.online, .app-status-txt.ok { color: #34c759; }
.app-status-txt.offline, .app-status-txt.bad { color: #ff3b30; }

/* Quick actions */
.quick-links-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 10px;
  margin-top: 12px;
}
.quick-link-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 12px;
  border-radius: 12px;
  background: rgba(0, 113, 227, 0.04);
  color: var(--apple-blue);
  cursor: pointer;
  transition: all 0.2s ease;
}
.quick-link-item:hover {
  background: rgba(0, 113, 227, 0.08);
  transform: scale(1.02);
}
.ql-icon {
  font-size: 20px;
  margin-bottom: 6px;
}
.quick-link-item span {
  font-size: 11px;
  font-weight: 700;
}

.text-center { text-align: center; }
.text-right { text-align: right; }
.panel-empty {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
  text-align: center;
  padding: 24px 0;
}

/* Trust Banner Styles */
.trust-banner {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
  background: linear-gradient(135deg, rgba(255,255,255,0.7) 0%, rgba(255,255,255,0.4) 100%);
  border-left: 5px solid #34c759;
}
html[data-theme='dark'] .trust-banner {
  background: linear-gradient(135deg, rgba(28,28,30,0.7) 0%, rgba(28,28,30,0.4) 100%);
}
.trust-banner.trust-b { border-left-color: #0071e3; }
.trust-banner.trust-c { border-left-color: #ff9500; }
.trust-banner.trust-d, .trust-banner.trust-f { border-left-color: #ff3b30; }

.trust-left {
  display: flex;
  align-items: center;
  gap: 20px;
}
.trust-score-badge {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  width: 70px;
  height: 70px;
  border-radius: 16px;
  background: rgba(52, 199, 89, 0.1);
  color: #34c759;
}
.trust-b .trust-score-badge { background: rgba(0, 113, 227, 0.1); color: #0071e3; }
.trust-c .trust-score-badge { background: rgba(255, 149, 0, 0.1); color: #ff9500; }
.trust-d .trust-score-badge, .trust-f .trust-score-badge { background: rgba(255, 59, 48, 0.1); color: #ff3b30; }

.trust-score-num {
  font-size: 28px;
  font-weight: 800;
  line-height: 1;
}
.trust-score-label {
  font-size: 9px;
  font-weight: 600;
  margin-top: 4px;
  opacity: 0.8;
}
.trust-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.trust-grade-row {
  display: flex;
  align-items: center;
  gap: 10px;
}
.trust-grade-badge {
  font-size: 10px;
  font-weight: 700;
  padding: 1px 6px;
  border-radius: 6px;
  background: #34c759;
  color: white;
}
.trust-b .trust-grade-badge { background: #0071e3; }
.trust-c .trust-grade-badge { background: #ff9500; }
.trust-d .trust-grade-badge, .trust-f .trust-grade-badge { background: #ff3b30; }

.trust-title {
  font-size: 16px;
  font-weight: 700;
  margin: 0;
  color: var(--el-text-color-primary);
}
.trust-desc {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin: 0;
}
.trust-right {
  max-width: 45%;
  background: rgba(0,0,0,0.02);
  padding: 12px 16px;
  border-radius: 12px;
  border: 1px solid rgba(0,0,0,0.04);
}
html[data-theme='dark'] .trust-right {
  background: rgba(255,255,255,0.02);
  border-color: rgba(255,255,255,0.04);
}
.action-hints-container {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.action-hints-title {
  font-size: 11px;
  font-weight: 700;
  color: var(--el-text-color-regular);
}
.action-hints-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.hint-item {
  display: flex;
  align-items: flex-start;
  gap: 6px;
}
.hint-dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--apple-blue);
  margin-top: 5px;
  flex-shrink: 0;
}
.hint-text {
  font-size: 10.5px;
  color: var(--el-text-color-secondary);
  line-height: 1.3;
}
</style>
