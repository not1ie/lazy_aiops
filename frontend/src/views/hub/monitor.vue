<template>
  <div class="hub-container" v-loading="loading">
    <header class="hub-header">
      <div class="hub-title">
        <h2>统一观测中心</h2>
        <p>汇聚指标监控、告警事件及域名健康，建立“发现-响应-复盘”的完整观测闭环。</p>
      </div>
      <div class="hub-actions">
        <el-button round icon="Refresh" @click="refreshAll">刷新观测数据</el-button>
        <el-button round icon="Share" @click="go('/topology')">服务拓扑</el-button>
        <el-button type="primary" round icon="Bell" @click="go('/alert/rules')">告警治理</el-button>
      </div>
    </header>

    <div class="hub-summary">
      <div class="summary-item" @click="activePanel = 'alerts'">
        <div class="summary-label">未处理告警</div>
        <div class="summary-value" :class="stats.alertOpen === 0 ? 'success' : 'danger'">{{ stats.alertOpen }}</div>
      </div>
      <div class="summary-item" @click="activePanel = 'alerts'">
        <div class="summary-label">Critical 事件</div>
        <div class="summary-value danger">{{ stats.alertCritical }}</div>
      </div>
      <div class="summary-item" @click="activePanel = 'domain'">
        <div class="summary-label">域名/证书风险</div>
        <div class="summary-value" :class="stats.domainRisk === 0 ? 'success' : 'warning'">{{ stats.domainRisk }}</div>
      </div>
      <div class="summary-item">
        <div class="summary-label">在线 Agent</div>
        <div class="summary-value success">{{ stats.agentOnline }}</div>
      </div>
    </div>

    <div class="hub-workspace full-width">
      <div class="workspace-content">
        <el-tabs v-model="activePanel" class="hub-tabs">
          <el-tab-pane label="活跃告警事件" name="alerts">
            <div v-if="alerts.length === 0 && !loading" class="empty-cta">
              <el-empty description="暂无活跃告警 🎉" :image-size="60">
                <el-button type="primary" @click="go('/alert/rules')">管理告警规则</el-button>
              </el-empty>
            </div>
            <el-table v-else :data="alerts" class="hub-table" size="small">
              <el-table-column label="告警名称" min-width="180">
                <template #default="{ row }">{{ row.alert_name || row.rule_name || '-' }}</template>
              </el-table-column>
              <el-table-column prop="target" label="监控目标" width="150" />
              <el-table-column label="级别" width="100">
                <template #default="{ row }">
                  <el-tag :type="row.severity === 'critical' ? 'danger' : 'warning'" size="small">{{ row.severity }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="时间" width="160">
                <template #default="{ row }">{{ new Date(row.fired_at || row.created_at).toLocaleString() }}</template>
              </el-table-column>
              <el-table-column label="操作" width="220">
                <template #default="{ row }">
                  <el-button link type="primary" @click="handleAck(row)">确认</el-button>
                  <el-button link type="success" @click="handleResolve(row)">恢复</el-button>
                  <el-button link type="warning" @click="handleCreateWorkorder(row)">转工单</el-button>
                  <el-button link @click="goAIAnalysis({ type: 'alert', title: row.alert_name || '告警', id: row.id, summary: `告警: ${row.alert_name || row.rule_name} (${row.severity}), 目标: ${row.target}, 时间: ${new Date(row.fired_at || row.created_at).toLocaleString()}` })">
                    <el-icon><MagicStick /></el-icon>
                  </el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>

          <el-tab-pane label="监控概览" name="metrics">
            <div class="metrics-grid">
               <!-- Placeholder for quick metrics -->
               <el-empty description="指标图表集成中，点击“全屏查看”进入 Grafana 视图" />
            </div>
          </el-tab-pane>

          <el-tab-pane label="域名与证书" name="domain">
            <el-table :data="domainRisks" class="hub-table" size="small">
              <el-table-column prop="name" label="域名/对象" />
              <el-table-column prop="reason" label="风险原因" />
              <el-table-column label="剩余天数" width="100">
                 <template #default="{ row }">{{ row.days_left || '-' }}</template>
              </el-table-column>
              <el-table-column label="操作" width="100">
                <template #default="{ row }">
                  <el-button link type="primary" @click="go('/domain/ssl')">处理</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>

          <el-tab-pane label="故障自愈" name="remediation">
            <div class="pane-header">
              <el-button link type="primary" @click="go('/remediation')">自愈执行日志 →</el-button>
            </div>
            <el-empty description="查看自动修复的执行历史和结果" :image-size="50">
              <el-button type="primary" size="small" @click="go('/remediation')">查看日志</el-button>
            </el-empty>
          </el-tab-pane>

          <el-tab-pane label="通知渠道" name="notify">
            <el-table :data="channels" class="hub-table" size="small">
              <el-table-column prop="name" label="渠道名称" />
              <el-table-column prop="type" label="类型" />
              <el-table-column label="状态">
                <template #default="{ row }"><el-tag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? '启用' : '禁用' }}</el-tag></template>
              </el-table-column>
            </el-table>
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
import { useAIChat } from '@/composables/useAIChat'

const router = useRouter()
const { goAIAnalysis } = useAIChat()
const loading = ref(false)
const alerts = ref([])
const channels = ref([])
const domainRisks = ref([])
const agents = ref([])
const activePanel = ref('alerts')

const stats = reactive({
  alertOpen: 0,
  alertCritical: 0,
  domainRisk: 0,
  agentOnline: 0
})

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })
const go = (path) => router.push(path)

const refreshAll = async () => {
  loading.value = true
  try {
    const [alt, chn, dom, agt] = await Promise.all([
      axios.get('/api/v1/alert/alerts', { headers: authHeaders() }),
      axios.get('/api/v1/notify/channels', { headers: authHeaders() }),
      axios.get('/api/v1/domain/domains', { headers: authHeaders() }),
      axios.get('/api/v1/monitor/agents', { headers: authHeaders() })
    ])
    alerts.value = alt.data?.data?.filter(a => a.status === 'firing') || []
    channels.value = chn.data?.data || []
    domainRisks.value = dom.data?.data?.filter(d => d.health_status !== 'healthy') || []
    agents.value = agt.data?.data || []

    stats.alertOpen = alerts.value.length
    stats.alertCritical = alerts.value.filter(a => a.severity === 'critical').length
    stats.domainRisk = domainRisks.value.length
    stats.agentOnline = agents.value.filter(a => a.status === 'online').length
  } catch (err) {
    ElMessage.error('加载观测中心失败')
  } finally {
    loading.value = false
  }
}

const handleAck = async (row) => {
  try {
    await axios.post(`/api/v1/alert/alerts/${row.id}/ack`, {}, { headers: authHeaders() })
    ElMessage.success('告警已确认')
    refreshAll()
  } catch (err) {
    ElMessage.error('确认失败: ' + (err.response?.data?.message || err.message))
  }
}
const handleResolve = async (row) => {
  try {
    await axios.post(`/api/v1/alert/alerts/${row.id}/resolve`, {}, { headers: authHeaders() })
    ElMessage.success('告警已恢复')
    refreshAll()
  } catch (err) {
    ElMessage.error('恢复失败: ' + (err.response?.data?.message || err.message))
  }
}
const handleCreateWorkorder = async (row) => {
  try {
    const res = await axios.post(`/api/v1/alert/alerts/${row.id}/create-workorder`, {}, { headers: authHeaders() })
    ElMessage.success('已转为工单: ' + (res.data?.data?.id || ''))
  } catch (err) {
    ElMessage.error('转工单失败: ' + (err.response?.data?.message || err.message))
  }
}

onMounted(refreshAll)
</script>

<style scoped>
.full-width { grid-template-columns: 1fr; }
.metrics-grid { padding: 40px 0; }
.empty-cta { padding: 40px 0; }
</style>
