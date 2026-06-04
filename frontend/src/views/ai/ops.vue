<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>AIOps 故障诊断</h2>
        <p class="page-desc">自然语言输入故障描述，AI 自动诊断、预检风险、生成修复计划并沉淀 Runbook。</p>
      </div>
      <div class="page-actions">
        <el-button icon="Refresh" @click="fetchIncidents">刷新</el-button>
      </div>
    </div>

    <!-- Diagnose Input -->
    <div class="diagnose-bar">
      <el-input
        v-model="diagnoseQuery"
        placeholder="描述故障现象，如：支付服务在 10:30 开始大量 502，怀疑数据库连接池耗尽..."
        class="diagnose-input"
        clearable
        @keyup.enter="handleDiagnose"
      >
        <template #prepend>🔍 故障诊断</template>
        <template #append>
          <el-button :loading="diagnosing" @click="handleDiagnose" type="primary">开始诊断</el-button>
        </template>
      </el-input>
    </div>

    <!-- Diagnose Result -->
    <el-alert v-if="diagnoseResult" :title="'诊断结果: ' + (diagnoseResult.status || 'analyzing')" :type="diagnoseResult.risk_level === 'critical' ? 'error' : 'warning'" :closable="true" @close="diagnoseResult = null" class="mb-16">
      <div class="diagnose-reply">{{ diagnoseResult.reply }}</div>
      <div class="diagnose-meta" v-if="diagnoseResult.incident_id">
        <span>Incident: <code>{{ diagnoseResult.incident_id }}</code></span>
        <span v-if="diagnoseResult.mttd_seconds">MTTD: {{ fmtDuration(diagnoseResult.mttd_seconds) }}</span>
        <span v-if="diagnoseResult.mttr_seconds">MTTR: {{ fmtDuration(diagnoseResult.mttr_seconds) }}</span>
      </div>
      <div class="diagnose-actions" v-if="diagnoseResult.incident_id">
        <el-button size="small" type="primary" @click="handlePreflight(diagnoseResult.incident_id)">Preflight 风险评分</el-button>
        <el-button size="small" type="success" @click="handleGenerateRunbook(diagnoseResult.incident_id)">生成 Runbook</el-button>
        <el-button size="small" @click="selectedIncidentId = diagnoseResult.incident_id; activeTab = 'timeline'">查看时间轴</el-button>
      </div>
      <div v-if="diagnoseResult.execution_plan" class="execution-plan">
        <h4>修复计划: {{ diagnoseResult.execution_plan.title }}</h4>
        <p>{{ diagnoseResult.execution_plan.summary }}</p>
        <el-table :data="diagnoseResult.execution_plan.steps || []" size="small">
          <el-table-column prop="title" label="步骤" />
          <el-table-column prop="action" label="动作" width="100" />
          <el-table-column prop="risk" label="风险" width="120" />
          <el-table-column label="需确认" width="80">
            <template #default="{ row }"><el-tag :type="row.requires_confirmation ? 'warning' : 'success'" size="small">{{ row.requires_confirmation ? '是' : '否' }}</el-tag></template>
          </el-table-column>
        </el-table>
      </div>
    </el-alert>

    <!-- Preflight Result -->
    <el-alert v-if="preflightResult" title="Preflight 风险评分" :type="preflightResult.risk_level === 'critical' ? 'error' : preflightResult.risk_level === 'high' ? 'warning' : 'success'" :closable="true" @close="preflightResult = null" class="mb-16">
      <div class="preflight-score">
        <span class="score-num" :class="'score-' + preflightResult.risk_level">{{ preflightResult.risk_score }}/100</span>
        <span class="score-label">{{ preflightResult.risk_level?.toUpperCase() }}</span>
        <span v-if="preflightResult.blocked" class="score-blocked">🚫 BLOCKED</span>
      </div>
      <ul class="preflight-reasons">
        <li v-for="r in preflightResult.reasons" :key="r">{{ r }}</li>
      </ul>
      <div v-if="preflightResult.suggestion" class="preflight-suggestion">💡 {{ preflightResult.suggestion }}</div>
    </el-alert>

    <!-- Runbook Result -->
    <el-alert v-if="runbookResult" title="Runbook 已生成" type="success" :closable="true" @close="runbookResult = null" class="mb-16">
      <div class="runbook-preview" v-html="runbookHtml"></div>
      <el-button size="small" type="primary" style="margin-top:8px" @click="handleSaveRunbook">保存到知识库</el-button>
    </el-alert>

    <!-- Tabs: Incidents / Timeline / Preflight -->
    <el-tabs v-model="activeTab" class="mb-16">
      <el-tab-pane label="故障记录" name="incidents">
        <el-table :data="incidents" v-loading="loadingIncidents" stripe size="small">
          <el-table-column label="标题" min-width="200">
            <template #default="{ row }">
              <el-button link type="primary" @click="viewIncident(row)">{{ row.title || row.query?.slice(0, 60) }}</el-button>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <el-tag :type="incidentStatusType(row.status)" size="small">{{ row.status }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="风险" width="80">
            <template #default="{ row }">{{ row.risk_level || '-' }}</template>
          </el-table-column>
          <el-table-column label="MTTD" width="80">
            <template #default="{ row }">{{ row.mttd_seconds ? fmtDuration(row.mttd_seconds) : '-' }}</template>
          </el-table-column>
          <el-table-column label="MTTR" width="80">
            <template #default="{ row }">{{ row.mttr_seconds ? fmtDuration(row.mttr_seconds) : '-' }}</template>
          </el-table-column>
          <el-table-column label="时间" width="140">
            <template #default="{ row }">{{ new Date(row.created_at).toLocaleString() }}</template>
          </el-table-column>
          <el-table-column label="操作" width="180" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" size="small" @click="viewIncident(row)">详情</el-button>
              <el-button link type="success" size="small" @click="handleGenerateRunbook(row.incident_id)">Runbook</el-button>
              <el-button link type="warning" size="small" @click="handlePreflight(row.incident_id)">Preflight</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="时间轴" name="timeline">
        <div class="pane-header">
          <el-input v-model="selectedIncidentId" placeholder="输入 Incident ID" size="small" style="width:300px" />
          <el-button size="small" type="primary" @click="fetchTimeline">查看</el-button>
        </div>
        <el-timeline v-if="timelineEvents.length > 0">
          <el-timeline-item
            v-for="ev in timelineEvents"
            :key="ev.id"
            :timestamp="new Date(ev.created_at).toLocaleString()"
            :color="ev.status === 'success' ? '#10b981' : ev.status === 'fail' ? '#ef4444' : '#6b7280'"
          >
            <div class="tl-item">
              <el-tag size="small" effect="plain">{{ ev.stage }}</el-tag>
              <span>{{ ev.detail }}</span>
              <span class="tl-meta">{{ ev.actor }} · {{ ev.duration_ms }}ms</span>
            </div>
          </el-timeline-item>
        </el-timeline>
        <el-empty v-else description="输入 Incident ID 查看故障时间轴" :image-size="40" />
      </el-tab-pane>
    </el-tabs>

    <!-- Incident Detail Dialog -->
    <el-dialog :title="'Incident: ' + (incidentDetail?.incident?.incident_id || '')" v-model="detailVisible" width="720px" append-to-body>
      <el-descriptions v-if="incidentDetail?.incident" :column="2" border size="small">
        <el-descriptions-item label="标题">{{ incidentDetail.incident.title }}</el-descriptions-item>
        <el-descriptions-item label="状态">{{ incidentDetail.incident.status }}</el-descriptions-item>
        <el-descriptions-item label="查询" :span="2">{{ incidentDetail.incident.query }}</el-descriptions-item>
        <el-descriptions-item label="根因">{{ incidentDetail.incident.root_cause_summary || '-' }}</el-descriptions-item>
        <el-descriptions-item label="风险">{{ incidentDetail.incident.risk_level || '-' }}</el-descriptions-item>
        <el-descriptions-item label="MTTD">{{ fmtDuration(incidentDetail.incident.mttd_seconds) }}</el-descriptions-item>
        <el-descriptions-item label="MTTR">{{ fmtDuration(incidentDetail.incident.mttr_seconds) }}</el-descriptions-item>
        <el-descriptions-item label="工单">{{ incidentDetail.incident.workorder_id || '-' }}</el-descriptions-item>
        <el-descriptions-item label="Preflight 评分">{{ incidentDetail.incident.last_preflight_score || '-' }}</el-descriptions-item>
      </el-descriptions>
      <div v-if="incidentDetail?.events?.length" style="margin-top:16px">
        <h4>时间轴</h4>
        <el-timeline>
          <el-timeline-item v-for="ev in incidentDetail.events" :key="ev.id" :timestamp="new Date(ev.created_at).toLocaleString()" :color="ev.status === 'success' ? '#10b981' : '#ef4444'">
            <el-tag size="small">{{ ev.stage }}</el-tag> {{ ev.detail }}
          </el-timeline-item>
        </el-timeline>
      </div>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const diagnoseQuery = ref('')
const diagnosing = ref(false)
const diagnoseResult = ref(null)
const preflightResult = ref(null)
const runbookResult = ref(null)

const activeTab = ref('incidents')
const incidents = ref([])
const loadingIncidents = ref(false)
const selectedIncidentId = ref('')
const timelineEvents = ref([])

const detailVisible = ref(false)
const incidentDetail = ref(null)

const fmtDuration = (sec) => {
  if (!sec || sec <= 0) return '-'
  if (sec < 60) return `${sec}s`
  if (sec < 3600) return `${Math.floor(sec / 60)}m`
  return `${Math.floor(sec / 3600)}h${Math.floor((sec % 3600) / 60)}m`
}

const incidentStatusType = (s) => {
  const map = { diagnosing: 'warning', preflight: 'info', approved: 'success', executing: 'warning', resolved: 'success', failed: 'danger' }
  return map[s] || 'info'
}

const runbookHtml = computed(() => {
  if (!runbookResult.value?.markdown && !runbookResult.value?.content) return ''
  const md = runbookResult.value.markdown || runbookResult.value.content || ''
  return md.replace(/\n/g, '<br>').replace(/## (.+)/g, '<h3>$1</h3>').replace(/`(.+?)`/g, '<code>$1</code>')
})

const handleDiagnose = async () => {
  const q = diagnoseQuery.value.trim()
  if (!q) return ElMessage.warning('请输入故障描述')
  diagnosing.value = true
  diagnoseResult.value = null
  try {
    const res = await axios.post('/api/v1/ai/ops/diagnose', { query: q }, { headers: authHeaders() })
    diagnoseResult.value = res.data?.data || res.data
    fetchIncidents()
  } catch (err) {
    ElMessage.error('诊断失败: ' + (err.response?.data?.message || err.message))
  } finally {
    diagnosing.value = false
  }
}

const handlePreflight = async (incidentId) => {
  if (!incidentId) return
  preflightResult.value = null
  try {
    const res = await axios.post('/api/v1/ai/ops/preflight', { incident_id: incidentId }, { headers: authHeaders() })
    preflightResult.value = res.data?.data || res.data
  } catch (err) {
    ElMessage.error('Preflight 失败: ' + (err.response?.data?.message || err.message))
  }
}

const handleGenerateRunbook = async (incidentId) => {
  if (!incidentId) return
  runbookResult.value = null
  try {
    const res = await axios.post('/api/v1/ai/ops/runbook/generate', { incident_id: incidentId }, { headers: authHeaders() })
    runbookResult.value = res.data?.data || res.data
  } catch (err) {
    ElMessage.error('Runbook 生成失败: ' + (err.response?.data?.message || err.message))
  }
}

const handleSaveRunbook = async () => {
  if (!runbookResult.value) return
  try {
    await axios.post('/api/v1/knowledge/docs', {
      title: 'Runbook: ' + (diagnoseResult.value?.incident_id || 'incident'),
      content: runbookResult.value.markdown || runbookResult.value.content || '',
      category: 'runbook',
      tags: 'aiops,auto-generated'
    }, { headers: authHeaders() })
    ElMessage.success('Runbook 已保存到知识库')
  } catch (err) {
    ElMessage.error('保存失败')
  }
}

const fetchIncidents = async () => {
  loadingIncidents.value = true
  try {
    const res = await axios.get('/api/v1/ai/ops/incidents', { headers: authHeaders() })
    incidents.value = res.data?.data || []
  } catch (err) { /* silent */ }
  finally { loadingIncidents.value = false }
}

const fetchTimeline = async () => {
  if (!selectedIncidentId.value) return
  try {
    const res = await axios.post('/api/v1/ai/ops/timeline', { incident_id: selectedIncidentId.value }, { headers: authHeaders() })
    timelineEvents.value = res.data?.data?.events || res.data?.data || []
  } catch (err) {
    ElMessage.error('获取时间轴失败')
  }
}

const viewIncident = async (row) => {
  try {
    const res = await axios.get(`/api/v1/ai/ops/incidents/${row.incident_id || row.id}`, { headers: authHeaders() })
    incidentDetail.value = res.data?.data || res.data
    detailVisible.value = true
  } catch (err) {
    ElMessage.error('获取详情失败')
  }
}

onMounted(fetchIncidents)
</script>

<style scoped>
.page-card { max-width: 100%; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 16px; }
.page-desc { color: var(--el-text-color-secondary); margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; }

.diagnose-bar { margin-bottom: 20px; }
.diagnose-input :deep(.el-input-group__prepend) { font-weight: 700; background: rgba(0,113,227,0.05); }

.mb-16 { margin-bottom: 16px; }
.pane-header { display: flex; gap: 8px; align-items: center; margin-bottom: 12px; }

.diagnose-reply { line-height: 1.7; margin-bottom: 8px; }
.diagnose-meta { display: flex; gap: 16px; font-size: 12px; color: var(--el-text-color-secondary); margin-bottom: 8px; }
.diagnose-meta code { background: var(--el-fill-color); padding: 1px 6px; border-radius: 3px; }
.diagnose-actions { display: flex; gap: 8px; margin-top: 8px; }

.execution-plan { margin-top: 12px; padding: 12px; background: var(--el-fill-color-light); border-radius: 8px; }
.execution-plan h4 { margin: 0 0 4px; font-size: 14px; }
.execution-plan p { font-size: 13px; color: var(--el-text-color-secondary); margin: 0 0 8px; }

.preflight-score { display: flex; align-items: center; gap: 12px; margin-bottom: 8px; }
.score-num { font-size: 32px; font-weight: 800; }
.score-num.score-low { color: #10b981; }
.score-num.score-medium { color: #f59e0b; }
.score-num.score-high { color: #ef4444; }
.score-num.score-critical { color: #dc2626; }
.score-label { font-size: 14px; font-weight: 700; color: var(--el-text-color-secondary); }
.score-blocked { font-size: 14px; font-weight: 800; color: #dc2626; }
.preflight-reasons { margin: 8px 0; padding-left: 20px; }
.preflight-suggestion { font-size: 13px; color: var(--el-text-color-secondary); }

.runbook-preview { max-height: 300px; overflow-y: auto; line-height: 1.7; }
.runbook-preview :deep(h3) { margin: 12px 0 4px; font-size: 14px; }
.runbook-preview :deep(code) { background: var(--el-fill-color); padding: 2px 6px; border-radius: 3px; }

.tl-item { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.tl-meta { font-size: 11px; color: var(--el-text-color-secondary); }
</style>
