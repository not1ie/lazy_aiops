<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>告警事件</h2>
        <p class="page-desc">查看告警列表并进行确认、恢复或故障自愈处理。</p>
      </div>
      <div class="page-actions">
        <el-button icon="Refresh" @click="fetchAlerts">刷新</el-button>
      </div>
    </div>

    <div class="filter-bar">
      <el-select v-model="status" placeholder="状态" class="w-40" clearable @change="fetchAlerts">
        <el-option label="未处理" :value="0" />
        <el-option label="已确认" :value="1" />
        <el-option label="已恢复" :value="2" />
        <el-option label="已抑制" :value="3" />
      </el-select>
      <el-select v-model="severity" placeholder="级别" class="w-40" clearable @change="fetchAlerts">
        <el-option label="critical" value="critical" />
        <el-option label="warning" value="warning" />
        <el-option label="info" value="info" />
      </el-select>
      <el-input v-model="target" placeholder="目标包含" class="w-52" clearable @change="fetchAlerts" />
      <el-button type="primary" @click="fetchAlerts">查询</el-button>
    </div>

    <el-table :fit="true" :data="alerts" stripe style="width: 100%">
      <el-table-column prop="rule_name" label="规则" min-width="160" />
      <el-table-column prop="target" label="目标" min-width="180" />
      <el-table-column prop="metric" label="指标" min-width="130" />
      <el-table-column prop="value" label="值" width="110" />
      <el-table-column prop="severity" label="级别" width="100">
        <template #default="scope">
          <el-tag :type="severityTag(scope.row.severity)">{{ scope.row.severity }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="status" label="状态" width="120">
        <template #default="scope">
          <StatusBadge
            :text="alertStatusMeta(scope.row).text"
            :type="alertStatusMeta(scope.row).type"
            :source="alertStatusMeta(scope.row).source"
            :check-at="alertStatusMeta(scope.row).checkAt"
            :is-stale="alertStatusMeta(scope.row).isStale"
            :stale-text="alertStatusMeta(scope.row).staleText"
            :reason="alertStatusMeta(scope.row).reason"
          />
        </template>
      </el-table-column>
      <el-table-column label="联动" min-width="180">
        <template #default="scope">
          <div class="linkage-wrap" v-if="scope.row.work_order_id">
            <el-tag type="info" size="small">工单 {{ shortID(scope.row.work_order_id) }}</el-tag>
            <el-tag v-if="scope.row.workflow_execution_id" type="success" size="small">执行 {{ shortID(scope.row.workflow_execution_id) }}</el-tag>
          </div>
          <span v-else class="text-gray-400">-</span>
        </template>
      </el-table-column>
      <el-table-column prop="fired_at" label="触发时间" min-width="170" />
      <el-table-column label="操作" width="370" fixed="right">
        <template #default="scope">
          <el-button size="small" @click="openDetail(scope.row)">详情</el-button>
          <el-button size="small" type="primary" plain @click="ack(scope.row)">确认</el-button>
          <el-button size="small" type="success" plain @click="resolve(scope.row)">恢复</el-button>
          <el-button size="small" type="warning" plain icon="MagicStick" @click="openRemediationModal(scope.row)">自愈</el-button>
          <el-button
            size="small"
            type="info"
            plain
            :disabled="!!scope.row.work_order_id"
            @click="createWorkOrder(scope.row)"
          >
            转工单
          </el-button>
          <el-button
            v-if="scope.row.work_order_id"
            size="small"
            plain
            @click="openWorkOrder(scope.row)"
          >
            工单
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 自愈日志与手动触发弹窗 -->
    <el-dialog
      v-model="remediationVisible"
      :title="`故障自愈 - ${currentAlert.rule_name || currentAlert.target}`"
      width="780px"
      append-to-body
    >
      <div class="rem-header">
        <div class="rem-meta">
          <span><strong>目标主机:</strong> {{ currentAlert.target }}</span>
          <span class="ml-4"><strong>告警级别:</strong> <el-tag size="small" :type="severityTag(currentAlert.severity)">{{ currentAlert.severity }}</el-tag></span>
          <span class="ml-4"><strong>当前状态:</strong> {{ currentAlert.status === 2 ? '已恢复' : '触发中' }}</span>
        </div>
        <el-button type="primary" size="small" icon="VideoPlay" :loading="triggeringRem" @click="triggerRemediation">
          立即执行自愈
        </el-button>
      </div>

      <el-divider style="margin: 12px 0;" />

      <div class="rem-logs-section" v-loading="remLogsLoading">
        <div v-if="remediationLogs.length === 0" class="rem-empty">
          <el-empty description="暂无该告警的自愈执行记录，可点击上方「立即执行自愈」进行修复" />
        </div>
        <div v-else class="rem-log-list">
          <el-card v-for="log in remediationLogs" :key="log.id" class="rem-log-card" shadow="never">
            <div class="log-card-header">
              <div class="log-status-badge">
                <el-tag :type="log.status === 'success' ? 'success' : (log.status === 'running' ? 'warning' : 'danger')" size="small">
                  {{ log.status === 'success' ? '执行成功' : (log.status === 'running' ? '执行中...' : '执行失败') }}
                </el-tag>
                <span class="log-time">{{ log.started_at }}</span>
                <span v-if="log.duration" class="log-dur">耗时: {{ log.duration }}s</span>
              </div>
            </div>
            <div class="log-action">
              <div class="log-label">执行动作/脚本:</div>
              <pre class="log-code">{{ log.action }}</pre>
            </div>
            <div v-if="log.stdout" class="log-output">
              <div class="log-label">标准输出 (Stdout):</div>
              <pre class="log-code output-stdout">{{ log.stdout }}</pre>
            </div>
            <div v-if="log.stderr || log.error" class="log-output">
              <div class="log-label text-danger">错误输出 (Stderr / Error):</div>
              <pre class="log-code output-stderr">{{ log.stderr || log.error }}</pre>
            </div>
          </el-card>
        </div>
      </div>
      <template #footer>
        <el-button @click="remediationVisible = false">关闭</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import { MagicStick, VideoPlay } from '@element-plus/icons-vue'
import { getErrorMessage, isCancelError } from '@/utils/error'
import StatusBadge from '@/components/common/StatusBadge.vue'
import { monitorAlertStatusMeta } from '@/utils/status'
import { useAlertsStore } from '@/store/alerts'

const alertsStore = useAlertsStore()
const alerts = ref([])
const status = computed({
  get: () => alertsStore.status,
  set: (val) => { alertsStore.status = val }
})
const severity = computed({
  get: () => alertsStore.severity,
  set: (val) => { alertsStore.severity = val }
})
const target = computed({
  get: () => alertsStore.target,
  set: (val) => { alertsStore.target = val }
})
const router = useRouter()

// 自愈状态
const remediationVisible = ref(false)
const remLogsLoading = ref(false)
const triggeringRem = ref(false)
const currentAlert = ref({})
const remediationLogs = ref([])

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const fetchAlerts = async () => {
  try {
    const res = await axios.get('/api/v1/alert/alerts', {
      headers: authHeaders(),
      params: {
        status: status.value === '' ? undefined : status.value,
        severity: severity.value || undefined,
        target: target.value || undefined
      }
    })
    alerts.value = res.data.data || []
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '加载告警事件失败'))
  }
}

const ack = async (row) => {
  try {
    await ElMessageBox.confirm('确认该告警？', '提示', { type: 'warning' })
    await axios.post(`/api/v1/alert/alerts/${row.id}/ack`, {}, { headers: authHeaders() })
    ElMessage.success('已确认')
    await fetchAlerts()
  } catch (err) {
    if (!isCancelError(err)) {
      ElMessage.error(getErrorMessage(err, '确认告警失败'))
    }
  }
}

const resolve = async (row) => {
  try {
    await ElMessageBox.confirm('标记为已恢复？', '提示', { type: 'warning' })
    await axios.post(`/api/v1/alert/alerts/${row.id}/resolve`, {}, { headers: authHeaders() })
    ElMessage.success('已恢复')
    await fetchAlerts()
  } catch (err) {
    if (!isCancelError(err)) {
      ElMessage.error(getErrorMessage(err, '恢复告警失败'))
    }
  }
}

const openDetail = (row) => {
  router.push({ path: '/alert/events/detail', query: { id: row.id } })
}

// 打开自愈详情弹窗
const openRemediationModal = async (row) => {
  currentAlert.value = row
  remediationVisible.value = true
  await fetchRemediationLogs(row.id)
}

const fetchRemediationLogs = async (alertId) => {
  remLogsLoading.value = true
  try {
    const res = await axios.get('/api/v1/remediation/logs', {
      params: { alert_id: alertId },
      headers: authHeaders()
    })
    remediationLogs.value = res.data.data || []
  } catch (err) {
    ElMessage.error('加载自愈日志失败')
  } finally {
    remLogsLoading.value = false
  }
}

// 立即触发自愈
const triggerRemediation = async () => {
  if (!currentAlert.value?.id) return
  triggeringRem.value = true
  try {
    const res = await axios.post(`/api/v1/remediation/trigger/${currentAlert.value.id}`, {}, {
      headers: authHeaders()
    })
    if (res.data?.code === 0) {
      ElMessage.success('已下发自愈任务，正在远程执行...')
      // 轮询 3 秒后刷新日志
      setTimeout(async () => {
        await fetchRemediationLogs(currentAlert.value.id)
        await fetchAlerts()
      }, 3000)
    } else {
      ElMessage.error(res.data?.message || '触发失败')
    }
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '触发自愈发生异常')
  } finally {
    triggeringRem.value = false
  }
}

const shortID = (value) => {
  const text = String(value || '').trim()
  if (!text) return '-'
  return text.slice(0, 8)
}

const createWorkOrder = async (row) => {
  try {
    await ElMessageBox.confirm('确认将该告警转换为工单吗？工单将进入审批流程。', '告警联动', { type: 'warning' })
    const priority = row.severity === 'critical' ? 1 : (row.severity === 'warning' ? 2 : 3)
    await axios.post(
      `/api/v1/alert/alerts/${row.id}/create-workorder`,
      { type_code: 'incident', priority },
      { headers: authHeaders() }
    )
    ElMessage.success('已生成联动工单')
    await fetchAlerts()
  } catch (err) {
    if (!isCancelError(err)) {
      ElMessage.error(getErrorMessage(err, '告警转工单失败'))
    }
  }
}

const openWorkOrder = (row) => {
  if (!row?.work_order_id) return
  router.push({ path: '/workorder/tickets', query: { workorder_id: row.work_order_id } })
}

const alertStatusMeta = (row) => {
  const meta = monitorAlertStatusMeta(row)
  const reason = [
    row?.status_reason,
    row?.message,
    row?.target ? `目标: ${row.target}` : '',
    row?.metric ? `指标: ${row.metric}` : ''
  ].filter(Boolean).join(' | ')
  return {
    ...meta,
    reason: reason || meta.reason
  }
}

const severityTag = (s) => {
  if (s === 'critical') return 'danger'
  if (s === 'warning') return 'warning'
  return 'info'
}

onMounted(fetchAlerts)
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 16px; }
.page-desc { color: #606266; margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; }
.filter-bar { display: flex; gap: 8px; flex-wrap: wrap; margin-bottom: 12px; }
.w-40 { width: 160px; }
.w-52 { width: 220px; }
.linkage-wrap { display: flex; gap: 6px; flex-wrap: wrap; }

.rem-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.rem-meta {
  font-size: 13px;
  color: #303133;
}
.ml-4 { margin-left: 16px; }
.rem-log-card {
  margin-bottom: 12px;
  background-color: #f8fafc;
}
.log-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}
.log-status-badge {
  display: flex;
  align-items: center;
  gap: 10px;
}
.log-time {
  font-size: 12px;
  color: #909399;
}
.log-dur {
  font-size: 12px;
  color: #409eff;
}
.log-label {
  font-size: 12px;
  font-weight: 600;
  color: #606266;
  margin: 6px 0 2px;
}
.text-danger { color: #f56c6c; }
.log-code {
  background: #1e293b;
  color: #e2e8f0;
  padding: 8px 12px;
  border-radius: 4px;
  font-family: Consolas, monospace;
  font-size: 12px;
  white-space: pre-wrap;
  word-break: break-all;
  margin: 0;
}
.output-stdout {
  background: #0f172a;
  color: #38bdf8;
}
.output-stderr {
  background: #2a1215;
  color: #fca5a5;
}
</style>
