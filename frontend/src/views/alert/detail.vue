<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>告警详情</h2>
        <p class="page-desc">告警事件的详细信息与处理。</p>
      </div>
      <div class="page-actions">
        <el-button icon="Refresh" @click="fetchDetail">刷新</el-button>
        <el-button type="primary" @click="ack">确认</el-button>
        <el-button type="success" @click="resolve">恢复</el-button>
        <el-button type="warning" plain :disabled="!!alert.work_order_id" @click="createWorkOrder">转工单</el-button>
        <el-button v-if="alert.work_order_id" plain @click="openWorkOrder">工单</el-button>
      </div>
    </div>

    <el-descriptions :column="2" border>
      <el-descriptions-item label="规则">{{ alert.rule_name }}</el-descriptions-item>
      <el-descriptions-item label="目标">{{ alert.target }}</el-descriptions-item>
      <el-descriptions-item label="指标">{{ alert.metric }}</el-descriptions-item>
      <el-descriptions-item label="阈值">{{ alert.threshold }}</el-descriptions-item>
      <el-descriptions-item label="值">{{ alert.value }}</el-descriptions-item>
      <el-descriptions-item label="级别">{{ alert.severity }}</el-descriptions-item>
      <el-descriptions-item label="状态">{{ statusText(alert.status) }}</el-descriptions-item>
      <el-descriptions-item label="触发时间">{{ alert.fired_at }}</el-descriptions-item>
      <el-descriptions-item label="确认人">{{ alert.acked_by || '-' }}</el-descriptions-item>
      <el-descriptions-item label="确认时间">{{ alert.acked_at || '-' }}</el-descriptions-item>
      <el-descriptions-item label="恢复时间">{{ alert.resolved_at || '-' }}</el-descriptions-item>
      <el-descriptions-item label="关联工单">{{ alert.work_order_id || '-' }}</el-descriptions-item>
      <el-descriptions-item label="Runbook执行">{{ alert.workflow_execution_id || '-' }}</el-descriptions-item>
      <el-descriptions-item label="状态说明" :span="2">{{ alert.status_reason || '-' }}</el-descriptions-item>
    </el-descriptions>

    <el-divider />

    <h3 class="section-title">Labels</h3>
    <el-input v-model="labelsText" type="textarea" :rows="8" readonly />

    <el-divider />

    <h3 class="section-title">Annotations</h3>
    <el-input v-model="annotationsText" type="textarea" :rows="8" readonly />
  </el-card>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const route = useRoute()
const router = useRouter()
const alert = ref({})
const labelsText = ref('')
const annotationsText = ref('')

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const fetchDetail = async () => {
  const id = route.query.id
  if (!id) return
  const res = await axios.get(`/api/v1/alert/alerts/${id}`, { headers: authHeaders() })
  alert.value = res.data.data || {}
  labelsText.value = formatJSON(alert.value.labels)
  annotationsText.value = formatJSON(alert.value.annotations)
}

const ack = async () => {
  const id = route.query.id
  if (!id) return
  await ElMessageBox.confirm('确认该告警？', '提示', { type: 'warning' })
  await axios.post(`/api/v1/alert/alerts/${id}/ack`, {}, { headers: authHeaders() })
  ElMessage.success('已确认')
  fetchDetail()
}

const resolve = async () => {
  const id = route.query.id
  if (!id) return
  await ElMessageBox.confirm('标记为已恢复？', '提示', { type: 'warning' })
  await axios.post(`/api/v1/alert/alerts/${id}/resolve`, {}, { headers: authHeaders() })
  ElMessage.success('已恢复')
  fetchDetail()
}

const createWorkOrder = async () => {
  const id = route.query.id
  if (!id) return
  try {
    await ElMessageBox.confirm('确认将该告警转换为工单吗？工单将进入审批流程。', '告警联动', { type: 'warning' })
    const priority = alert.value.severity === 'critical' ? 1 : (alert.value.severity === 'warning' ? 2 : 3)
    await axios.post(`/api/v1/alert/alerts/${id}/create-workorder`, { type_code: 'incident', priority }, { headers: authHeaders() })
    ElMessage.success('已生成联动工单')
    await fetchDetail()
  } catch (err) {
    if (err !== 'cancel' && err !== 'close') {
      ElMessage.error('告警转工单失败')
    }
  }
}

const openWorkOrder = () => {
  if (!alert.value?.work_order_id) return
  router.push({ path: '/workorder/tickets', query: { workorder_id: alert.value.work_order_id } })
}

const formatJSON = (txt) => {
  if (!txt) return ''
  try {
    const obj = typeof txt === 'string' ? JSON.parse(txt) : txt
    return JSON.stringify(obj, null, 2)
  } catch {
    return String(txt)
  }
}

const statusText = (s) => {
  if (s === 0) return '未处理'
  if (s === 1) return '已确认'
  if (s === 2) return '已恢复'
  return '已抑制'
}

onMounted(fetchDetail)
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 16px; }
.page-desc { color: #606266; margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; }
.section-title { margin: 12px 0; }
</style>
