<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>工单管理</h2>
        <p class="page-desc">提交、审批、执行与评论，形成完整工单闭环。</p>
      </div>
      <div class="page-actions">
        <el-button type="primary" icon="Plus" @click="openCreate">新建工单</el-button>
        <el-button icon="Refresh" @click="reloadAll">刷新</el-button>
      </div>
    </div>

    <el-row :gutter="12" class="summary-row">
      <el-col :xs="12" :sm="6">
        <el-card shadow="never" class="metric-card">
          <div class="metric-title">总工单</div>
          <div class="metric-value">{{ stats.total || 0 }}</div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="6">
        <el-card shadow="never" class="metric-card">
          <div class="metric-title">待审批</div>
          <div class="metric-value">{{ stats.pending || 0 }}</div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="6">
        <el-card shadow="never" class="metric-card">
          <div class="metric-title">执行中</div>
          <div class="metric-value">{{ stats.processing || 0 }}</div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="6">
        <el-card shadow="never" class="metric-card">
          <div class="metric-title">已完成</div>
          <div class="metric-value">{{ stats.completed || 0 }}</div>
        </el-card>
      </el-col>
    </el-row>

    <div class="filter-row">
      <el-select v-model="statusFilter" clearable placeholder="状态" class="w-40">
        <el-option v-for="opt in statusOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
      </el-select>
      <el-select v-model="typeFilter" clearable placeholder="类型" class="w-52">
        <el-option v-for="t in types" :key="t.id" :label="t.name" :value="t.id" />
      </el-select>
      <el-switch v-model="myPending" active-text="仅我的待办" />
      <el-button type="primary" icon="Search" @click="fetchOrders">查询</el-button>
    </div>

    <el-table :fit="true" :data="orders" v-loading="loading" stripe>
      <el-table-column prop="title" label="标题" min-width="220" show-overflow-tooltip />
      <el-table-column prop="type_name" label="类型" width="130" />
      <el-table-column label="优先级" width="100">
        <template #default="{ row }">
          <el-tag :type="priorityType(row.priority)">{{ priorityText(row.priority) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="110">
        <template #default="{ row }">
          <el-tag :type="statusType(row.status)">{{ statusText(row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="submitter" label="提交人" width="100" />
      <el-table-column prop="assignee" label="处理人" width="100" />
      <el-table-column label="创建时间" width="170">
        <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="250" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="openDetail(row)">详情</el-button>
          <el-button size="small" v-if="canApprove(row)" @click="openApprove(row, true)">通过</el-button>
          <el-button size="small" type="warning" v-if="canApprove(row)" @click="openApprove(row, false)">拒绝</el-button>
          <el-button size="small" type="primary" v-if="row.status === 2" @click="executeOrder(row)">执行</el-button>
          <el-button size="small" type="success" v-if="row.status === 4" @click="completeOrder(row)">完成</el-button>
          <el-button size="small" type="danger" v-if="canCancel(row)" @click="cancelOrder(row)">取消</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog append-to-body v-model="createVisible" title="新建工单" width="760px">
      <el-form :model="createForm" label-width="100px">
        <el-form-item label="标题">
          <el-input v-model="createForm.title" />
        </el-form-item>
        <el-form-item label="工单类型">
          <el-select v-model="createForm.type_id" class="w-52" filterable>
            <el-option v-for="t in types" :key="t.id" :label="t.name" :value="t.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="优先级">
          <el-select v-model="createForm.priority" class="w-40">
            <el-option label="紧急" :value="1" />
            <el-option label="高" :value="2" />
            <el-option label="中" :value="3" />
            <el-option label="低" :value="4" />
          </el-select>
        </el-form-item>
        <el-form-item label="内容">
          <el-input v-model="createForm.content" type="textarea" :rows="6" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" @click="submitCreate">提交</el-button>
      </template>
    </el-dialog>

    <el-dialog append-to-body v-model="approveVisible" :title="approveForm.approved ? '审批通过' : '审批拒绝'" width="560px">
      <el-form :model="approveForm" label-width="80px">
        <el-form-item label="备注">
          <el-input v-model="approveForm.comment" type="textarea" :rows="4" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="approveVisible = false">取消</el-button>
        <el-button type="primary" @click="submitApprove">确认</el-button>
      </template>
    </el-dialog>

    <el-drawer v-model="detailVisible" title="工单详情" size="62%" destroy-on-close append-to-body>
      <el-descriptions :column="2" border class="mb-3">
        <el-descriptions-item label="标题">{{ detail.order?.title || '-' }}</el-descriptions-item>
        <el-descriptions-item label="状态">{{ statusText(detail.order?.status) }}</el-descriptions-item>
        <el-descriptions-item label="类型">{{ detail.order?.type_name || '-' }}</el-descriptions-item>
        <el-descriptions-item label="优先级">{{ priorityText(detail.order?.priority) }}</el-descriptions-item>
        <el-descriptions-item label="提交人">{{ detail.order?.submitter || '-' }}</el-descriptions-item>
        <el-descriptions-item label="处理人">{{ detail.order?.assignee || '-' }}</el-descriptions-item>
      </el-descriptions>

      <el-card shadow="never" class="mb-3">
        <template #header>工单内容</template>
        <div class="pre-wrap">{{ detail.order?.content || '-' }}</div>
      </el-card>

      <el-card shadow="never" class="mb-3">
        <template #header>审批步骤</template>
        <el-steps :active="detail.order?.current_step || 1" finish-status="success" align-center>
          <el-step
            v-for="s in detail.steps"
            :key="s.id"
            :title="s.name"
            :description="`${stepStatusText(s.status)} ${s.approver ? `(${s.approver})` : ''}`"
          />
        </el-steps>
      </el-card>

      <el-card shadow="never">
        <template #header>
          <div class="section-header">
            <span>评论</span>
            <el-button type="primary" plain @click="submitComment">发表评论</el-button>
          </div>
        </template>
        <el-input v-model="commentInput" type="textarea" :rows="3" placeholder="输入评论内容" class="mb-3" />
        <el-timeline>
          <el-timeline-item v-for="item in detail.comments" :key="item.id" :timestamp="formatTime(item.created_at)">
            <span class="comment-user">{{ item.username || 'system' }}</span>
            <span class="comment-type">[{{ item.type }}]</span>
            <div class="pre-wrap">{{ item.content }}</div>
          </el-timeline-item>
        </el-timeline>
      </el-card>
    </el-drawer>
  </el-card>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const loading = ref(false)
const orders = ref([])
const types = ref([])
const stats = ref({ total: 0, pending: 0, processing: 0, completed: 0 })

const statusFilter = ref('')
const typeFilter = ref('')
const myPending = ref(false)

const createVisible = ref(false)
const createForm = ref({
  title: '',
  type_id: '',
  content: '',
  priority: 3
})

const approveVisible = ref(false)
const approveOrderId = ref('')
const approveForm = ref({ approved: true, comment: '' })

const detailVisible = ref(false)
const detailOrderId = ref('')
const detail = ref({ order: null, steps: [], comments: [] })
const commentInput = ref('')

const statusOptions = [
  { label: '待审批', value: 0 },
  { label: '审批中', value: 1 },
  { label: '已通过', value: 2 },
  { label: '已拒绝', value: 3 },
  { label: '执行中', value: 4 },
  { label: '已完成', value: 5 },
  { label: '已取消', value: 6 }
]

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const formatTime = (v) => {
  if (!v) return '-'
  return String(v).slice(0, 19).replace('T', ' ')
}

const priorityText = (v) => ({ 1: '紧急', 2: '高', 3: '中', 4: '低' }[v] || '-')
const priorityType = (v) => ({ 1: 'danger', 2: 'warning', 3: 'primary', 4: 'info' }[v] || 'info')

const statusText = (v) => ({
  0: '待审批',
  1: '审批中',
  2: '已通过',
  3: '已拒绝',
  4: '执行中',
  5: '已完成',
  6: '已取消'
}[v] || '-')

const statusType = (v) => ({ 0: 'warning', 1: 'primary', 2: 'success', 3: 'danger', 4: 'primary', 5: 'success', 6: 'info' }[v] || 'info')
const stepStatusText = (v) => ({ 0: '待审批', 1: '通过', 2: '拒绝' }[v] || '-')

const canApprove = (row) => row.status === 0 || row.status === 1
const canCancel = (row) => [0, 1, 2, 4].includes(row.status)

const fetchTypes = async () => {
  try {
    const res = await axios.get('/api/v1/workorder/types', { headers: authHeaders() })
    types.value = res.data?.data || []
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '获取类型失败')
  }
}

const fetchStats = async () => {
  try {
    const res = await axios.get('/api/v1/workorder/stats', { headers: authHeaders() })
    stats.value = res.data?.data || {}
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '获取统计失败')
  }
}

const fetchOrders = async () => {
  loading.value = true
  try {
    const params = {}
    if (statusFilter.value !== '' && statusFilter.value !== null) params.status = statusFilter.value
    if (typeFilter.value) params.type_id = typeFilter.value
    if (myPending.value) params.my_pending = true
    const res = await axios.get('/api/v1/workorder/orders', { headers: authHeaders(), params })
    orders.value = res.data?.data || []
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '获取工单失败')
  } finally {
    loading.value = false
  }
}

const reloadAll = async () => {
  await Promise.all([fetchTypes(), fetchStats(), fetchOrders()])
}

const openCreate = () => {
  createForm.value = { title: '', type_id: '', content: '', priority: 3 }
  createVisible.value = true
}

const submitCreate = async () => {
  if (!createForm.value.title.trim() || !createForm.value.type_id) {
    ElMessage.warning('请填写标题和工单类型')
    return
  }
  try {
    await axios.post('/api/v1/workorder/orders', createForm.value, { headers: authHeaders() })
    ElMessage.success('创建成功')
    createVisible.value = false
    await reloadAll()
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '创建失败')
  }
}

const openApprove = (row, approved) => {
  approveOrderId.value = row.id
  approveForm.value = { approved, comment: '' }
  approveVisible.value = true
}

const submitApprove = async () => {
  try {
    await axios.post(`/api/v1/workorder/orders/${approveOrderId.value}/approve`, approveForm.value, { headers: authHeaders() })
    ElMessage.success('审批完成')
    approveVisible.value = false
    await reloadAll()
    if (detailOrderId.value === approveOrderId.value) await fetchDetail(detailOrderId.value)
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '审批失败')
  }
}

const executeOrder = async (row) => {
  try {
    await axios.post(`/api/v1/workorder/orders/${row.id}/execute`, {}, { headers: authHeaders() })
    ElMessage.success('已开始执行')
    await reloadAll()
    if (detailOrderId.value === row.id) await fetchDetail(row.id)
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '执行失败')
  }
}

const completeOrder = async (row) => {
  try {
    const { value } = await ElMessageBox.prompt('请输入执行结果', '完成工单', { inputPlaceholder: '例如：已完成并验证通过' })
    await axios.post(`/api/v1/workorder/orders/${row.id}/complete`, { result: value || '' }, { headers: authHeaders() })
    ElMessage.success('工单已完成')
    await reloadAll()
    if (detailOrderId.value === row.id) await fetchDetail(row.id)
  } catch (err) {
    if (err !== 'cancel') ElMessage.error(err.response?.data?.message || '操作失败')
  }
}

const cancelOrder = async (row) => {
  try {
    await ElMessageBox.confirm(`确认取消工单 ${row.title} 吗？`, '提示', { type: 'warning' })
    await axios.post(`/api/v1/workorder/orders/${row.id}/cancel`, {}, { headers: authHeaders() })
    ElMessage.success('已取消')
    await reloadAll()
    if (detailOrderId.value === row.id) await fetchDetail(row.id)
  } catch (err) {
    if (err !== 'cancel') ElMessage.error(err.response?.data?.message || '操作失败')
  }
}

const fetchDetail = async (id) => {
  const res = await axios.get(`/api/v1/workorder/orders/${id}`, { headers: authHeaders() })
  detail.value = res.data?.data || { order: null, steps: [], comments: [] }
}

const openDetail = async (row) => {
  detailOrderId.value = row.id
  commentInput.value = ''
  try {
    await fetchDetail(row.id)
    detailVisible.value = true
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '读取详情失败')
  }
}

const submitComment = async () => {
  if (!commentInput.value.trim() || !detailOrderId.value) {
    ElMessage.warning('请输入评论内容')
    return
  }
  try {
    await axios.post(`/api/v1/workorder/orders/${detailOrderId.value}/comment`, { content: commentInput.value }, { headers: authHeaders() })
    commentInput.value = ''
    await fetchDetail(detailOrderId.value)
    ElMessage.success('评论成功')
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '评论失败')
  }
}

onMounted(reloadAll)
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; gap: 12px; margin-bottom: 12px; }
.page-desc { margin: 4px 0 0; color: #606266; }
.page-actions { display: flex; gap: 8px; align-items: center; }
.summary-row { margin-bottom: 12px; }
.metric-card { min-height: 88px; }
.metric-title { color: #909399; font-size: 13px; margin-bottom: 8px; }
.metric-value { font-size: 24px; font-weight: 700; }
.filter-row { display: flex; align-items: center; gap: 8px; margin-bottom: 12px; flex-wrap: wrap; }
.w-40 { width: 160px; }
.w-52 { width: 220px; }
.mb-3 { margin-bottom: 12px; }
.section-header { display: flex; justify-content: space-between; align-items: center; }
.pre-wrap { white-space: pre-wrap; line-height: 1.5; }
.comment-user { font-weight: 600; margin-right: 6px; }
.comment-type { color: #909399; margin-right: 8px; }
</style>
