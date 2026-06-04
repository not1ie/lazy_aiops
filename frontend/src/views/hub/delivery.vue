<template>
  <div class="hub-container" v-loading="loading">
    <header class="hub-header">
      <div class="hub-title">
        <h2>变更交付中心</h2>
        <p>构建 → 发布 → 审批 → 执行 → 复盘，软件交付全生命周期闭环。</p>
      </div>
      <div class="hub-actions">
        <el-button round icon="Refresh" @click="refreshAll">刷新</el-button>
        <el-button round icon="Tools" @click="go('/executor')">批量执行</el-button>
        <el-button round icon="Share" @click="go('/gitops/repos')">GitOps</el-button>
        <el-button type="primary" round icon="Promotion" @click="go('/cicd/pipelines')">新建流水线</el-button>
      </div>
    </header>

    <div class="hub-summary">
      <div class="summary-item" @click="activePanel = 'executions'">
        <div class="summary-label">今日执行</div>
        <div class="summary-value">{{ stats.todayExecutions }}</div>
      </div>
      <div class="summary-item" :class="{ 'has-pending': stats.ticketPending > 0 }" @click="activePanel = 'tickets'">
        <div class="summary-label">待审批</div>
        <div class="summary-value" :class="stats.ticketPending > 0 ? 'warning' : 'success'">{{ stats.ticketPending }}</div>
      </div>
      <div class="summary-item" @click="activePanel = 'releases'">
        <div class="summary-label">近期发布</div>
        <div class="summary-value">{{ stats.releaseCount }}</div>
      </div>
      <div class="summary-item" @click="activePanel = 'apps'">
        <div class="summary-label">托管应用</div>
        <div class="summary-value">{{ stats.appTotal }}</div>
      </div>
    </div>

    <div class="hub-workspace full-width">
      <div class="workspace-content">
        <el-tabs v-model="activePanel" class="hub-tabs">
          <!-- 1. Executions (most viewed) -->
          <el-tab-pane label="执行记录" name="executions">
            <div class="pane-header">
              <el-select v-model="execFilter" placeholder="状态" size="small" style="width:120px" clearable>
                <el-option label="成功" value="success" />
                <el-option label="失败" value="failed" />
                <el-option label="运行中" value="running" />
              </el-select>
              <el-button size="small" @click="go('/cicd/executions')">全部记录 →</el-button>
            </div>
            <el-table :data="filteredExecutions" class="hub-table" size="small">
              <el-table-column prop="pipeline_name" label="流水线" min-width="160" />
              <el-table-column label="状态" width="90">
                <template #default="{ row }">
                  <el-tag :type="execStatusType(row.status)" size="small">{{ row.status || 'unknown' }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="触发" width="80">
                <template #default="{ row }">{{ row.trigger || '-' }}</template>
              </el-table-column>
              <el-table-column label="耗时" width="80">
                <template #default="{ row }">{{ row.duration ? row.duration + 's' : '-' }}</template>
              </el-table-column>
              <el-table-column label="时间" width="130">
                <template #default="{ row }">{{ fmtTimeAgo(row.created_at) }}</template>
              </el-table-column>
              <el-table-column label="操作" width="100">
                <template #default="{ row }">
                  <el-button link type="primary" size="small" @click="go('/cicd/executions')">日志</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>

          <!-- 2. Pipelines -->
          <el-tab-pane label="流水线" name="pipelines">
            <div class="pane-header">
              <span class="text-muted">{{ pipelines.length }} 条流水线</span>
              <el-button size="small" type="primary" @click="go('/cicd/pipelines')">管理 →</el-button>
            </div>
            <el-table :data="pipelines" class="hub-table" size="small">
              <el-table-column prop="name" label="名称" min-width="180" />
              <el-table-column prop="provider" label="类型" width="90">
                <template #default="{ row }">
                  <el-tag size="small" effect="plain">{{ row.provider }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="状态" width="80">
                <template #default="{ row }">
                  <el-tag :type="row.status === 1 ? 'success' : 'info'" size="small">{{ row.status === 1 ? '启用' : '停用' }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="操作" width="160">
                <template #default="{ row }">
                  <el-button link type="primary" size="small" @click="handleTriggerPipeline(row)">触发</el-button>
                  <el-button link size="small" @click="go('/cicd/pipelines')">编辑</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>

          <!-- 3. Releases -->
          <el-tab-pane label="发布管理" name="releases">
            <div class="pane-header">
              <span class="text-muted">最近 {{ releases.length }} 次发布</span>
              <el-button size="small" @click="go('/cicd/releases')">全部发布 →</el-button>
            </div>
            <el-table :data="releases" class="hub-table" size="small">
              <el-table-column prop="name" label="发布名称" min-width="160" />
              <el-table-column prop="version" label="版本" width="120" />
              <el-table-column prop="environment" label="环境" width="80" />
              <el-table-column label="状态" width="90">
                <template #default="{ row }">
                  <el-tag :type="releaseStatusType(row.status)" size="small">{{ releaseStatusText(row.status) }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="operator" label="操作人" width="100" />
              <el-table-column label="时间" width="130">
                <template #default="{ row }">{{ fmtTimeAgo(row.release_at || row.created_at) }}</template>
              </el-table-column>
            </el-table>
          </el-tab-pane>

          <!-- 4. Applications -->
          <el-tab-pane label="应用中心" name="apps">
            <div class="pane-header">
              <span class="text-muted">{{ stats.appTotal }} 个应用</span>
              <el-button size="small" type="primary" @click="go('/application')">管理应用 →</el-button>
            </div>
            <el-table :data="apps" class="hub-table" size="small">
              <el-table-column prop="name" label="应用" min-width="140" />
              <el-table-column prop="language" label="语言" width="80" />
              <el-table-column prop="owner" label="负责人" width="100" />
              <el-table-column label="操作" width="100">
                <template #default="{ row }">
                  <el-button link type="primary" size="small" @click="go('/application')">详情</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>

          <!-- 5. Workorders + Oncall -->
          <el-tab-pane label="工单与值班" name="tickets">
            <el-row :gutter="16">
              <el-col :span="14">
                <h4 class="sub-title">待办工单</h4>
                <el-table :data="tickets" size="small">
                  <el-table-column prop="title" label="标题" min-width="180" show-overflow-tooltip />
                  <el-table-column prop="creator" label="提单人" width="90" />
                  <el-table-column label="操作" width="80">
                    <template #default><el-button link type="primary" size="small" @click="go('/workorder/tickets')">审批</el-button></template>
                  </el-table-column>
                </el-table>
                <div class="tab-footer-link">
                  <el-button link type="primary" size="small" @click="go('/workorder/tickets')">全部工单 →</el-button>
                </div>
              </el-col>
              <el-col :span="10">
                <h4 class="sub-title">今日值班</h4>
                <div v-if="oncallNow.length === 0" class="text-muted" style="padding:12px 0">未设置值班</div>
                <div v-for="item in oncallNow" :key="item.id || item.username" class="oncall-chip">
                  <el-avatar :size="24" icon="UserFilled" />
                  <div>
                    <div class="oc-name">{{ item.username || item.name }}</div>
                    <div class="oc-type">{{ item.type || '主值班' }}</div>
                  </div>
                </div>
                <div class="tab-footer-link">
                  <el-button link size="small" @click="go('/oncall/schedule')">排班管理 →</el-button>
                </div>
              </el-col>
            </el-row>
          </el-tab-pane>

          <!-- 6. Tools -->
          <el-tab-pane label="变更工具" name="tools">
            <div class="quick-grid">
              <el-card class="quick-card" shadow="hover" @click="go('/sqlaudit/requests')">
                <el-icon :size="24" class="qc-red"><Document /></el-icon>
                <span class="qc-label">SQL 审核</span>
                <span class="qc-badge" v-if="stats.sqlRisk > 0">{{ stats.sqlRisk }}</span>
              </el-card>
              <el-card class="quick-card" shadow="hover" @click="go('/gitops/repos')">
                <el-icon :size="24" class="qc-blue"><Share /></el-icon>
                <span class="qc-label">GitOps</span>
              </el-card>
              <el-card class="quick-card" shadow="hover" @click="go('/knowledge')">
                <el-icon :size="24" class="qc-purple"><Reading /></el-icon>
                <span class="qc-label">知识库</span>
              </el-card>
              <el-card class="quick-card" shadow="hover" @click="go('/ansible/playbooks')">
                <el-icon :size="24" class="qc-green"><SetUp /></el-icon>
                <span class="qc-label">Ansible</span>
              </el-card>
              <el-card class="quick-card" shadow="hover" @click="go('/task/schedules')">
                <el-icon :size="24" class="qc-orange"><Clock /></el-icon>
                <span class="qc-label">任务调度</span>
              </el-card>
              <el-card class="quick-card" shadow="hover" @click="go('/workflow/orchestrator')">
                <el-icon :size="24" class="qc-teal"><Operation /></el-icon>
                <span class="qc-label">编排中心</span>
              </el-card>
            </div>
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
import { ElMessage } from 'element-plus'
import './hub-common.css'

const router = useRouter()
const loading = ref(false)
const apps = ref([])
const executions = ref([])
const pipelines = ref([])
const releases = ref([])
const tickets = ref([])
const oncallNow = ref([])
const activePanel = ref('executions')
const execFilter = ref('')

const stats = reactive({ pipelineTotal: 0, ticketPending: 0, appTotal: 0, sqlRisk: 0, todayExecutions: 0, releaseCount: 0 })

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })
const go = (path) => router.push(path)

const fmtTimeAgo = (val) => {
  if (!val) return '-'
  const diff = Math.floor((Date.now() - new Date(val).getTime()) / 1000)
  if (diff < 60) return '刚刚'
  if (diff < 3600) return `${Math.floor(diff / 60)}m 前`
  if (diff < 86400) return `${Math.floor(diff / 3600)}h 前`
  return `${Math.floor(diff / 86400)}d 前`
}

const execStatusType = (s) => {
  if (s === 'success') return 'success'
  if (s === 'failed') return 'danger'
  if (s === 'running') return 'warning'
  return 'info'
}

const releaseStatusText = (s) => {
  const map = { 0: '待发布', 1: '已发布', 2: '已回滚', 3: '失败' }
  return map[s] || '未知'
}
const releaseStatusType = (s) => {
  const map = { 0: 'info', 1: 'success', 2: 'warning', 3: 'danger' }
  return map[s] || 'info'
}

const filteredExecutions = computed(() => {
  if (!execFilter.value) return executions.value.slice(0, 20)
  return executions.value.filter(e => e.status === execFilter.value).slice(0, 20)
})

const handleTriggerPipeline = async (row) => {
  try {
    await axios.post(`/api/v1/cicd/pipelines/${row.id}/trigger`, {}, { headers: authHeaders() })
    ElMessage.success('流水线已触发')
    refreshAll()
  } catch (err) {
    ElMessage.error('触发失败: ' + (err.response?.data?.message || err.message))
  }
}

const refreshAll = async () => {
  loading.value = true
  try {
    const headers = authHeaders()
    const [appRes, exeRes, pipeRes, relRes, tktRes, oncRes, sqlRes] = await Promise.all([
      axios.get('/api/v1/application/applications', { headers }),
      axios.get('/api/v1/cicd/executions', { headers }),
      axios.get('/api/v1/cicd/pipelines', { headers }).catch(() => ({ data: {} })),
      axios.get('/api/v1/cicd/releases', { headers }).catch(() => ({ data: {} })),
      axios.get('/api/v1/workorder/tickets', { headers }),
      axios.get('/api/v1/oncall/whoisoncall', { headers }).catch(() => ({ data: {} })),
      axios.get('/api/v1/sqlaudit/statistics', { headers }).catch(() => ({ data: {} }))
    ])
    apps.value = appRes.data?.data || []
    executions.value = exeRes.data?.data || []
    pipelines.value = pipeRes.data?.data || []
    releases.value = (relRes.data?.data || []).slice(0, 20)
    tickets.value = (tktRes.data?.data || []).filter(t => t.status === 'pending').slice(0, 10)
    oncallNow.value = oncRes.data?.data || []

    stats.appTotal = apps.value.length
    stats.pipelineTotal = pipelines.value.filter(p => p.status === 1).length
    stats.ticketPending = tickets.value.length
    stats.todayExecutions = executions.value.length
    stats.releaseCount = releases.value.length
    stats.sqlRisk = sqlRes.data?.data?.risk_count || sqlRes.data?.data?.total || 0
  } catch (err) {
    ElMessage.error('加载交付中心失败')
  } finally {
    loading.value = false
  }
}

onMounted(refreshAll)
</script>

<style scoped>
.full-width { grid-template-columns: 1fr; }
.pane-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.text-muted { color: var(--el-text-color-secondary); font-size: 13px; }
.sub-title { font-size: 14px; font-weight: 700; margin: 0 0 12px 0; color: var(--el-text-color-primary); }
.tab-footer-link { margin-top: 12px; text-align: right; }

.summary-item.has-pending { background: rgba(245, 158, 11, 0.06); }

.oncall-chip { display: flex; align-items: center; gap: 10px; padding: 8px 0; border-bottom: 1px solid var(--el-border-color-lighter); }
.oncall-chip:last-child { border-bottom: none; }
.oc-name { font-size: 13px; font-weight: 600; }
.oc-type { font-size: 11px; color: var(--el-text-color-secondary); }

.quick-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 12px; }
.quick-card { cursor: pointer; display: flex; align-items: center; gap: 12px; padding: 16px; }
.quick-card:hover { border-color: var(--apple-blue); }
.qc-label { font-size: 14px; font-weight: 600; color: var(--el-text-color-primary); flex: 1; }
.qc-badge { font-size: 12px; font-weight: 700; color: #fff; background: #ef4444; padding: 2px 8px; border-radius: 10px; }
.qc-red { color: #ef4444; }
.qc-blue { color: #0071e3; }
.qc-purple { color: #8b5cf6; }
.qc-green { color: #10b981; }
.qc-orange { color: #f59e0b; }
.qc-teal { color: #14b8a6; }
</style>
