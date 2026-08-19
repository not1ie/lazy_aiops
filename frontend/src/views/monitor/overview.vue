<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>监控与系统体检概览</h2>
        <p class="page-desc">全平台资产健康巡检评分、实时资源监控与风险隐患排查。</p>
      </div>
      <div class="page-actions">
        <el-button type="primary" icon="Aim" :loading="inspecting" @click="runInspection">
          一键全面体检
        </el-button>
        <el-button icon="Refresh" :loading="refreshing" @click="handleRefresh">刷新</el-button>
      </div>
    </div>

    <!-- 智能体检评分横幅卡片 -->
    <div class="inspection-banner">
      <div class="score-box">
        <div class="score-circle" :class="scoreClass">
          <span class="score-num">{{ inspectionReport.score || 100 }}</span>
          <span class="score-unit">分</span>
        </div>
        <div class="score-info">
          <div class="score-grade">健康等级: <strong>{{ inspectionReport.grade || 'A (优秀)' }}</strong></div>
          <div class="score-desc">最后巡检时间: {{ inspectionReport.check_time || '刚刚' }}</div>
        </div>
      </div>

      <div class="inspection-kpi">
        <div class="kpi-item">
          <div class="kpi-val">{{ inspectionReport.total_checks || 0 }}</div>
          <div class="kpi-label">已检查指标</div>
        </div>
        <div class="kpi-item text-success">
          <div class="kpi-val">{{ inspectionReport.passed_checks || 0 }}</div>
          <div class="kpi-label">正常项</div>
        </div>
        <div class="kpi-item text-warning">
          <div class="kpi-val">{{ inspectionReport.warning_count || 0 }}</div>
          <div class="kpi-label">预警项</div>
        </div>
        <div class="kpi-item text-danger">
          <div class="kpi-val">{{ inspectionReport.critical_count || 0 }}</div>
          <div class="kpi-label">严重隐患</div>
        </div>
      </div>

      <div class="inspection-btn-area">
        <el-button 
          :type="inspectionReport.critical_count > 0 ? 'danger' : 'primary'" 
          plain 
          icon="Document"
          @click="inspectionDialogVisible = true"
        >
          查看完整体检报告 ({{ (inspectionReport.all_issues || []).length }} 项待关注)
        </el-button>
      </div>
    </div>

    <!-- 基础监控卡片 -->
    <el-row :gutter="16" class="metric-cards">
      <el-col :span="6"><el-card shadow="hover"><div class="card-title">平均 CPU 使用率</div><div class="card-value">{{ realtime.cpu }}%</div></el-card></el-col>
      <el-col :span="6"><el-card shadow="hover"><div class="card-title">平均 内存 使用率</div><div class="card-value">{{ realtime.memory }}%</div></el-card></el-col>
      <el-col :span="6"><el-card shadow="hover"><div class="card-title">平均 磁盘 使用率</div><div class="card-value">{{ realtime.disk }}%</div></el-card></el-col>
      <el-col :span="6"><el-card shadow="hover"><div class="card-title">实时网络流量 (MB/s)</div><div class="card-value">{{ realtime.network }}</div></el-card></el-col>
    </el-row>

    <el-divider />

    <el-row :gutter="16">
      <el-col :span="16">
        <el-card>
          <div class="section-title">资源趋势 (最近1小时)</div>
          <div ref="trendRef" class="chart-box"></div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card>
          <div class="section-title">告警事件汇总</div>
          <div class="stat-grid">
            <el-statistic title="未处理" :value="alertStats.open" />
            <el-statistic title="已恢复" :value="alertStats.closed" />
            <el-statistic title="已抑制" :value="alertStats.ignored" />
          </div>
        </el-card>
        <el-card class="mt-12">
          <div class="section-title">主机探针与心跳状态</div>
          <div class="stat-grid">
            <el-statistic title="在线主机" :value="agentStats.online" value-style="color: #67c23a" />
            <el-statistic title="失联/离线" :value="agentStats.offline" value-style="color: #f56c6c" />
            <el-statistic title="总纳管数" :value="inspectionReport.host_stats?.total || 0" />
          </div>
        </el-card>
        <el-card class="mt-12">
          <div class="section-title">最近活跃告警</div>
          <el-table :fit="true" :data="recentAlerts" size="small" style="width: 100%">
            <el-table-column prop="rule_name" label="规则" min-width="120" />
            <el-table-column prop="severity" label="级别" width="80" />
            <el-table-column prop="status" label="状态" width="90">
              <template #default="scope">
                <StatusBadge v-bind="alertStatusBadge(scope.row)" />
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <!-- 完整体检报告弹窗 -->
    <el-dialog
      v-model="inspectionDialogVisible"
      title="全系统资产与运维健康巡检报告"
      width="820px"
      append-to-body
    >
      <div class="dialog-report-header">
        <div class="header-score">
          <span class="score-large">{{ inspectionReport.score }}</span>
          <span class="score-text">分 ({{ inspectionReport.grade }})</span>
        </div>
        <div class="header-summary">
          <p>共执行 <strong>{{ inspectionReport.total_checks }}</strong> 项指标扫描，其中正常 <strong>{{ inspectionReport.passed_checks }}</strong> 项，发现严重隐患 <strong>{{ inspectionReport.critical_count }}</strong> 项、预警 <strong>{{ inspectionReport.warning_count }}</strong> 项。</p>
        </div>
      </div>

      <div v-if="inspectionReport.recommendations?.length" class="dialog-recom-box">
        <div class="recom-title">💡 SRE 专家处置与优化建议:</div>
        <ul>
          <li v-for="(rec, idx) in inspectionReport.recommendations" :key="idx">{{ rec }}</li>
        </ul>
      </div>

      <div class="dialog-issues-list">
        <div class="issue-title">待关注风险项清单 ({{ (inspectionReport.all_issues || []).length }} 项):</div>
        <el-empty v-if="!inspectionReport.all_issues || inspectionReport.all_issues.length === 0" description="未发现系统隐患，所有指标处于健康状态！" />
        <el-card 
          v-for="(issue, idx) in inspectionReport.all_issues" 
          :key="idx" 
          class="issue-card" 
          :class="'issue-' + issue.level"
          shadow="never"
        >
          <div class="issue-header">
            <el-tag :type="issue.level === 'critical' ? 'danger' : 'warning'" size="small">
              {{ issue.level === 'critical' ? '严重高危' : '一般预警' }}
            </el-tag>
            <strong class="issue-target">{{ issue.title }}</strong>
          </div>
          <div class="issue-desc">{{ issue.description }}</div>
          <div class="issue-sug"><strong>👉 解决建议:</strong> {{ issue.suggestion }}</div>
        </el-card>
      </div>

      <template #footer>
        <el-button @click="inspectionDialogVisible = false">关闭</el-button>
        <el-button type="primary" icon="Aim" :loading="inspecting" @click="runInspection">重新扫描体检</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount, computed } from 'vue'
import axios from 'axios'
import * as echarts from 'echarts'
import { ElMessage } from 'element-plus'
import { Aim, Document } from '@element-plus/icons-vue'
import { getErrorMessage } from '@/utils/error'
import { monitorAgentStatusMeta, monitorAlertStatusMeta } from '@/utils/status'
import StatusBadge from '@/components/common/StatusBadge.vue'

const realtime = ref({ cpu: 0, memory: 0, disk: 0, network: 0 })
const alertStats = ref({ open: 0, closed: 0, ignored: 0 })
const agentStats = ref({ online: 0, stale: 0, offline: 0, unknown: 0 })
const recentAlerts = ref([])
const topNodes = ref([])
const refreshing = ref(false)

// 智能巡检体检
const inspecting = ref(false)
const inspectionDialogVisible = ref(false)
const inspectionReport = ref({
  score: 100,
  grade: 'A (优秀)',
  check_time: '',
  total_checks: 0,
  passed_checks: 0,
  warning_count: 0,
  critical_count: 0,
  all_issues: [],
  recommendations: []
})

const trendRef = ref(null)
let trendChart = null
let refreshTimer = null

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const scoreClass = computed(() => {
  const s = inspectionReport.value.score || 100
  if (s >= 90) return 'score-green'
  if (s >= 75) return 'score-blue'
  if (s >= 60) return 'score-yellow'
  return 'score-red'
})

const runInspection = async () => {
  inspecting.value = true
  try {
    const res = await axios.get('/api/v1/monitor/inspection/report', { headers: authHeaders() })
    if (res.data?.code === 0) {
      inspectionReport.value = res.data.data
      ElMessage.success('全系统资产巡检完成！')
    }
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '执行系统体检失败'))
  } finally {
    inspecting.value = false
  }
}

const alertStatusBadge = (row) => {
  const meta = monitorAlertStatusMeta(row)
  const parts = []
  if (row?.severity) parts.push(`级别: ${row.severity}`)
  if (row?.target) parts.push(`目标: ${row.target}`)
  if (row?.message) parts.push(row.message)
  return {
    text: meta.text,
    type: meta.type,
    source: meta.source,
    checkAt: meta.checkAt,
    reason: parts.join(' | ') || meta.reason
  }
}

const fetchRealtime = async () => {
  try {
    const res = await axios.get('/api/v1/monitor/metrics/realtime', { headers: authHeaders() })
    realtime.value = res.data.data || { cpu: 0, memory: 0, disk: 0, network: 0 }
  } catch {}
}

const fetchOverview = async () => {
  try {
    const res = await axios.get('/api/v1/monitor/metrics', { headers: authHeaders() })
    const data = res.data.data || {}
    alertStats.value = data.alerts || { open: 0, closed: 0, ignored: 0 }
    recentAlerts.value = data.recent_alerts || []
    topNodes.value = data.top_nodes || []
  } catch {}
}

const fetchAgents = async () => {
  try {
    const res = await axios.get('/api/v1/monitor/agents', { headers: authHeaders() })
    const list = res.data.data || []
    const stats = { online: 0, stale: 0, offline: 0, unknown: 0 }
    for (const item of list) {
      const meta = monitorAgentStatusMeta(item)
      if (meta.type === 'success') stats.online++
      else if (meta.isStale) stats.stale++
      else if (meta.type === 'danger') stats.offline++
      else stats.unknown++
    }
    agentStats.value = stats
  } catch {}
}

const initChart = () => {
  if (!trendRef.value) return
  trendChart = echarts.init(trendRef.value)
  trendChart.setOption({
    tooltip: { trigger: 'axis' },
    legend: { data: ['CPU', '内存', '磁盘'] },
    xAxis: { type: 'category', data: ['10m', '20m', '30m', '40m', '50m', '60m'] },
    yAxis: { type: 'value', max: 100 },
    series: [
      { name: 'CPU', type: 'line', smooth: true, data: [15, 22, 18, 25, 20, 24], itemStyle: { color: '#409eff' } },
      { name: '内存', type: 'line', smooth: true, data: [45, 46, 48, 47, 49, 48], itemStyle: { color: '#67c23a' } },
      { name: '磁盘', type: 'line', smooth: true, data: [32, 32, 33, 33, 33, 34], itemStyle: { color: '#e6a23c' } }
    ]
  })
}

const handleRefresh = async () => {
  refreshing.value = true
  await Promise.all([fetchRealtime(), fetchOverview(), fetchAgents(), runInspection()])
  refreshing.value = false
}

onMounted(() => {
  initChart()
  handleRefresh()
  refreshTimer = setInterval(() => {
    fetchRealtime()
    fetchOverview()
  }, 15000)
})

onBeforeUnmount(() => {
  if (refreshTimer) clearInterval(refreshTimer)
  if (trendChart) trendChart.dispose()
})
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 16px; }
.page-desc { color: #606266; margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; }

/* 巡检评分 Banner 样式 */
.inspection-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: linear-gradient(135deg, #1e293b 0%, #0f172a 100%);
  color: #fff;
  padding: 16px 24px;
  border-radius: 8px;
  margin-bottom: 16px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  flex-wrap: wrap;
  gap: 16px;
}

.score-box {
  display: flex;
  align-items: center;
  gap: 16px;
}

.score-circle {
  width: 68px;
  height: 68px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
  border: 3px solid;
}
.score-green { color: #56d364; border-color: #56d364; box-shadow: 0 0 12px rgba(86, 211, 100, 0.4); }
.score-blue { color: #38bdf8; border-color: #38bdf8; box-shadow: 0 0 12px rgba(56, 189, 248, 0.4); }
.score-yellow { color: #fbbf24; border-color: #fbbf24; box-shadow: 0 0 12px rgba(251, 191, 36, 0.4); }
.score-red { color: #f87171; border-color: #f87171; box-shadow: 0 0 12px rgba(248, 113, 113, 0.4); }

.score-num { font-size: 26px; }
.score-unit { font-size: 13px; margin-left: 2px; }

.score-grade { font-size: 16px; margin-bottom: 4px; }
.score-desc { font-size: 12px; color: #94a3b8; }

.inspection-kpi {
  display: flex;
  align-items: center;
  gap: 28px;
}
.kpi-item { text-align: center; }
.kpi-val { font-size: 22px; font-weight: 700; color: #f8fafc; }
.kpi-label { font-size: 12px; color: #94a3b8; margin-top: 2px; }
.text-success .kpi-val { color: #4ade80; }
.text-warning .kpi-val { color: #facc15; }
.text-danger .kpi-val { color: #f87171; }

.metric-cards .card-title { color: #64748b; font-size: 13px; }
.metric-cards .card-value { font-size: 24px; font-weight: 700; margin-top: 8px; color: #0f172a; }

.chart-box { height: 320px; }
.section-title { font-weight: 600; font-size: 14px; margin-bottom: 12px; }
.stat-grid { display: flex; justify-content: space-around; }
.mt-12 { margin-top: 12px; }

/* 报告弹窗内部样式 */
.dialog-report-header {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 12px 16px;
  background: #f1f5f9;
  border-radius: 6px;
  margin-bottom: 12px;
}
.score-large { font-size: 32px; font-weight: 700; color: #0284c7; }
.score-text { font-size: 14px; font-weight: 600; margin-left: 4px; }
.header-summary p { margin: 0; font-size: 13px; color: #334155; }

.dialog-recom-box {
  background: #f0fdf4;
  border: 1px solid #bbf7d0;
  border-radius: 6px;
  padding: 10px 16px;
  margin-bottom: 14px;
}
.recom-title { font-weight: 600; color: #166534; margin-bottom: 6px; font-size: 13px; }
.dialog-recom-box ul { margin: 0; padding-left: 20px; font-size: 12.5px; color: #15803d; }

.issue-title { font-size: 13.5px; font-weight: 600; margin-bottom: 10px; color: #1e293b; }
.issue-card { margin-bottom: 8px; border-left: 4px solid #cbd5e1; }
.issue-critical { border-left-color: #ef4444; background: #fef2f2; }
.issue-warning { border-left-color: #f59e0b; background: #fffbeb; }

.issue-header { display: flex; align-items: center; gap: 8px; margin-bottom: 4px; }
.issue-target { font-size: 13px; color: #1e293b; }
.issue-desc { font-size: 12px; color: #64748b; margin-bottom: 4px; }
.issue-sug { font-size: 12px; color: #0369a1; }
</style>
