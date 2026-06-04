<template>
  <div class="hub-container" v-loading="loading">
    <header class="hub-header">
      <div class="hub-title">
        <h2>交付与生命周期</h2>
        <p>从流水线构建到应用发布，从工单审批到 SQL 审计，覆盖软件交付的全生命周期。</p>
      </div>
      <div class="hub-actions">
        <el-button round icon="Refresh" @click="refreshAll">刷新流水线</el-button>
        <el-button round icon="Tools" @click="go('/executor')">批量执行</el-button>
        <el-button round icon="Share" @click="go('/gitops/repos')">GitOps</el-button>
        <el-button type="primary" round icon="Promotion" @click="go('/cicd/pipelines')">新建流水线</el-button>
      </div>
    </header>

    <div class="hub-summary">
      <div class="summary-item" @click="activePanel = 'pipelines'">
        <div class="summary-label">活跃流水线</div>
        <div class="summary-value success">{{ stats.pipelineTotal }}</div>
      </div>
      <div class="summary-item" @click="activePanel = 'tickets'">
        <div class="summary-label">待办工单</div>
        <div class="summary-value" :class="stats.ticketPending > 0 ? 'warning' : 'success'">{{ stats.ticketPending }}</div>
      </div>
      <div class="summary-item" @click="activePanel = 'apps'">
        <div class="summary-label">托管应用</div>
        <div class="summary-value">{{ stats.appTotal }}</div>
      </div>
      <div class="summary-item" @click="activePanel = 'sql'">
        <div class="summary-label">SQL 审计异常</div>
        <div class="summary-value" :class="stats.sqlRisk > 0 ? 'danger' : 'success'">{{ stats.sqlRisk }}</div>
      </div>
    </div>

    <div class="hub-workspace full-width">
      <div class="workspace-content">
        <el-tabs v-model="activePanel" class="hub-tabs">
          <el-tab-pane label="应用管理" name="apps">
            <div class="pane-header">
              <el-button link type="primary" @click="go('/sqlaudit/requests')">SQL 审计</el-button>
            </div>
            <el-table :data="apps" class="hub-table" size="small">
              <el-table-column prop="name" label="应用名称" min-width="150" />
              <el-table-column prop="language" label="语言/框架" width="120" />
              <el-table-column prop="owner" label="负责人" width="120" />
              <el-table-column label="操作" width="120">
                <template #default="{ row }">
                  <el-button link type="primary" @click="go('/application')">详情</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>

          <el-tab-pane label="流水线记录" name="pipelines">
            <el-table :data="executions" class="hub-table" size="small">
              <el-table-column prop="pipeline_name" label="流水线" />
              <el-table-column label="状态" width="100">
                <template #default="{ row }">
                  <el-tag :type="row.status === 'success' ? 'success' : 'danger'" size="small">{{ row.status }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="executor" label="执行人" width="120" />
              <el-table-column label="时间" prop="created_at" :formatter="formatTime" width="160" />
            </el-table>
          </el-tab-pane>

          <el-tab-pane label="工单审批" name="tickets">
            <el-table :data="tickets" class="hub-table" size="small">
              <el-table-column prop="id" label="单号" width="100" />
              <el-table-column prop="title" label="标题" min-width="200" />
              <el-table-column prop="creator" label="提单人" width="120" />
              <el-table-column label="状态" width="100">
                <template #default="{ row }"><el-tag size="small">{{ row.status }}</el-tag></template>
              </el-table-column>
              <el-table-column label="操作" width="100">
                <template #default="{ row }">
                  <el-button link type="primary" @click="go('/workorder/tickets')">审批</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>

          <el-tab-pane label="知识库" name="knowledge">
            <div class="pane-header">
              <el-button link type="primary" @click="go('/knowledge')">运维知识库 →</el-button>
            </div>
            <el-empty v-if="knowledgeDocs.length === 0" description="暂无知识库文档" :image-size="50">
              <el-button type="primary" size="small" @click="go('/knowledge')">添加第一篇 Runbook</el-button>
            </el-empty>
          </el-tab-pane>

          <el-tab-pane label="值班与升级" name="oncall">
            <el-table :data="oncallNow" class="hub-table" size="small">
              <el-table-column prop="username" label="当前值班" />
              <el-table-column prop="type" label="级别" />
              <el-table-column label="结束时间" prop="end_at" :formatter="formatTime" />
              <el-table-column label="操作" width="100">
                <template #default="{ row }">
                  <el-button link type="primary" @click="go('/oncall/schedule')">排班</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>
        </el-tabs>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import './hub-common.css'

const router = useRouter()
const loading = ref(false)
const apps = ref([])
const executions = ref([])
const tickets = ref([])
const oncallNow = ref([])
const knowledgeDocs = ref([])
const activePanel = ref('apps')

const stats = reactive({
  pipelineTotal: 0,
  ticketPending: 0,
  appTotal: 0,
  sqlRisk: 0
})

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })
const go = (path) => router.push(path)
const formatTime = (row, col, val) => val ? new Date(val).toLocaleString() : '-'

const refreshAll = async () => {
  loading.value = true
  try {
    const [appRes, exeRes, tktRes, oncRes, sqlRes, knowledgeDocsRes] = await Promise.all([
      axios.get('/api/v1/application/applications', { headers: authHeaders() }),
      axios.get('/api/v1/cicd/executions', { headers: authHeaders() }),
      axios.get('/api/v1/workorder/tickets', { headers: authHeaders() }),
      axios.get('/api/v1/oncall/whoisoncall', { headers: authHeaders() }),
      axios.get('/api/v1/sqlaudit/statistics', { headers: authHeaders() }).catch(() => ({ data: {} })),
      axios.get('/api/v1/knowledge/docs', { headers: authHeaders(), params: { size: 5 } }).catch(() => ({ data: {} }))
    ])
    apps.value = appRes.data?.data || []
    executions.value = exeRes.data?.data || []
    tickets.value = tktRes.data?.data?.filter(t => t.status === 'pending') || []
    oncallNow.value = oncRes.data?.data || []

    stats.appTotal = apps.value.length
    stats.pipelineTotal = [...new Set(executions.value.map(e => e.pipeline_id))].length
    stats.ticketPending = tickets.value.length
    stats.sqlRisk = sqlRes.data?.data?.risk_count || sqlRes.data?.data?.total || 0

    // Knowledge docs
    knowledgeDocs.value = knowledgeDocsRes?.data?.data || []
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
</style>
