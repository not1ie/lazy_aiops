<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>SQL 工单</h2>
        <p class="page-desc">数据库实例管理、SQL工单审核执行与审计日志。</p>
      </div>
      <div class="page-actions">
        <el-button type="primary" icon="Plus" @click="openCreateOrder">新建工单</el-button>
        <el-button icon="Refresh" @click="reloadAll">刷新</el-button>
      </div>
    </div>

    <el-row :gutter="12" class="summary-row">
      <el-col :xs="12" :sm="6"><el-card shadow="never" class="metric-card"><div class="metric-title">总工单</div><div class="metric-value">{{ statOrders.total || 0 }}</div></el-card></el-col>
      <el-col :xs="12" :sm="6"><el-card shadow="never" class="metric-card"><div class="metric-title">待审核</div><div class="metric-value">{{ statOrders.pending || 0 }}</div></el-card></el-col>
      <el-col :xs="12" :sm="6"><el-card shadow="never" class="metric-card"><div class="metric-title">执行成功</div><div class="metric-value">{{ statOrders.executed || 0 }}</div></el-card></el-col>
      <el-col :xs="12" :sm="6"><el-card shadow="never" class="metric-card"><div class="metric-title">执行失败</div><div class="metric-value">{{ statOrders.failed || 0 }}</div></el-card></el-col>
    </el-row>

    <el-tabs v-model="activeTab">
      <el-tab-pane label="SQL工单" name="orders">
        <div class="filter-row">
          <el-select v-model="orderInstanceFilter" clearable placeholder="实例" class="w-52">
            <el-option v-for="ins in instances" :key="ins.id" :label="ins.name" :value="ins.id" />
          </el-select>
          <el-select v-model="orderStatusFilter" clearable placeholder="状态" class="w-40">
            <el-option v-for="s in orderStatusOptions" :key="s.value" :label="s.label" :value="s.value" />
          </el-select>
          <el-button type="primary" icon="Search" @click="fetchOrders">查询</el-button>
        </div>

        <el-table :data="orders" v-loading="orderLoading" stripe>
          <el-table-column prop="title" label="标题" min-width="220" show-overflow-tooltip />
          <el-table-column label="实例" min-width="140">
            <template #default="{ row }">{{ row.instance?.name || '-' }}</template>
          </el-table-column>
          <el-table-column prop="database" label="库" width="130" />
          <el-table-column prop="sql_type" label="SQL类型" width="110" />
          <el-table-column prop="audit_level" label="风险" width="90">
            <template #default="{ row }">
              <el-tag :type="auditLevelType(row.audit_level)">{{ auditLevelText(row.audit_level) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="status" label="状态" width="110">
            <template #default="{ row }">
              <el-tag :type="orderStatusType(row.status)">{{ orderStatusText(row.status) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="submitter" label="提交人" width="100" />
          <el-table-column prop="created_at" label="创建时间" width="170">
            <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="250" fixed="right">
            <template #default="{ row }">
              <el-button size="small" @click="openOrderDetail(row)">详情</el-button>
              <el-button size="small" v-if="row.status === 0" @click="openReview(row, true)">通过</el-button>
              <el-button size="small" type="warning" v-if="row.status === 0" @click="openReview(row, false)">拒绝</el-button>
              <el-button size="small" type="primary" v-if="row.status === 1" @click="executeOrder(row)">执行</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="数据库实例" name="instances">
        <div class="filter-row">
          <el-button type="primary" icon="Plus" @click="openCreateInstance">新增实例</el-button>
          <el-button icon="Refresh" @click="fetchInstances">刷新</el-button>
        </div>
        <el-table :data="instances" v-loading="instanceLoading" stripe>
          <el-table-column prop="name" label="名称" min-width="140" />
          <el-table-column prop="type" label="类型" width="100" />
          <el-table-column prop="host" label="地址" min-width="160" />
          <el-table-column prop="port" label="端口" width="90" />
          <el-table-column prop="database" label="数据库" width="120" />
          <el-table-column prop="environment" label="环境" width="100" />
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <el-tag :type="row.status === 1 ? 'success' : 'danger'">{{ row.status === 1 ? '正常' : '异常' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="260" fixed="right">
            <template #default="{ row }">
              <el-button size="small" @click="openEditInstance(row)">编辑</el-button>
              <el-button size="small" type="primary" plain @click="testInstance(row)">测试连接</el-button>
              <el-button size="small" type="danger" @click="removeInstance(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="审计日志" name="logs">
        <div class="filter-row">
          <el-select v-model="logInstanceFilter" clearable placeholder="实例" class="w-52">
            <el-option v-for="ins in instances" :key="ins.id" :label="ins.name" :value="ins.id" />
          </el-select>
          <el-select v-model="logTypeFilter" clearable placeholder="SQL类型" class="w-40">
            <el-option label="DQL" value="DQL" />
            <el-option label="DML" value="DML" />
            <el-option label="DDL" value="DDL" />
            <el-option label="OTHER" value="OTHER" />
          </el-select>
          <el-button type="primary" icon="Search" @click="fetchLogs">查询</el-button>
        </div>
        <el-table :data="logs" v-loading="logLoading" stripe>
          <el-table-column prop="executed_at" label="执行时间" width="170">
            <template #default="{ row }">{{ formatTime(row.executed_at) }}</template>
          </el-table-column>
          <el-table-column prop="instance_name" label="实例" min-width="130" />
          <el-table-column prop="database" label="库" width="120" />
          <el-table-column prop="username" label="执行人" width="100" />
          <el-table-column prop="sql_type" label="类型" width="90" />
          <el-table-column prop="execute_time" label="耗时(ms)" width="100" />
          <el-table-column prop="affected_rows" label="影响行数" width="100" />
          <el-table-column label="状态" width="90">
            <template #default="{ row }">
              <el-tag :type="row.status === 1 ? 'success' : 'danger'">{{ row.status === 1 ? '成功' : '失败' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="sql_content" label="SQL" min-width="220" show-overflow-tooltip />
        </el-table>
      </el-tab-pane>
    </el-tabs>

    <el-dialog append-to-body v-model="instanceVisible" :title="instanceDialogTitle" width="760px">
      <el-form :model="instanceForm" label-width="100px">
        <el-form-item label="名称"><el-input v-model="instanceForm.name" /></el-form-item>
        <el-form-item label="类型">
          <el-select v-model="instanceForm.type" class="w-52">
            <el-option label="mysql" value="mysql" />
          </el-select>
        </el-form-item>
        <el-form-item label="地址"><el-input v-model="instanceForm.host" /></el-form-item>
        <el-form-item label="端口"><el-input-number v-model="instanceForm.port" :min="1" :max="65535" /></el-form-item>
        <el-form-item label="用户名"><el-input v-model="instanceForm.username" /></el-form-item>
        <el-form-item label="密码"><el-input v-model="instanceForm.password" type="password" show-password /></el-form-item>
        <el-form-item label="数据库"><el-input v-model="instanceForm.database" /></el-form-item>
        <el-form-item label="字符集"><el-input v-model="instanceForm.charset" /></el-form-item>
        <el-form-item label="环境"><el-input v-model="instanceForm.environment" /></el-form-item>
        <el-form-item label="描述"><el-input v-model="instanceForm.description" type="textarea" :rows="2" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="instanceVisible = false">取消</el-button>
        <el-button type="primary" @click="submitInstance">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog append-to-body v-model="orderVisible" title="新建 SQL 工单" width="860px">
      <el-form :model="orderForm" label-width="100px">
        <el-form-item label="标题"><el-input v-model="orderForm.title" /></el-form-item>
        <el-form-item label="数据库实例">
          <el-select v-model="orderForm.instance_id" class="w-52" filterable>
            <el-option v-for="ins in instances" :key="ins.id" :label="ins.name" :value="ins.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="数据库"><el-input v-model="orderForm.database" /></el-form-item>
        <el-form-item label="SQL内容"><el-input v-model="orderForm.sql_content" type="textarea" :rows="10" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="orderVisible = false">取消</el-button>
        <el-button type="primary" @click="submitOrder">提交工单</el-button>
      </template>
    </el-dialog>

    <el-dialog append-to-body v-model="reviewVisible" :title="reviewForm.approved ? '审核通过' : '审核拒绝'" width="560px">
      <el-form :model="reviewForm" label-width="80px">
        <el-form-item label="备注"><el-input v-model="reviewForm.remark" type="textarea" :rows="4" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="reviewVisible = false">取消</el-button>
        <el-button type="primary" @click="submitReview">确认</el-button>
      </template>
    </el-dialog>

    <el-drawer v-model="detailVisible" title="SQL工单详情" size="62%" append-to-body>
      <el-descriptions :column="2" border class="mb-3">
        <el-descriptions-item label="标题">{{ detailOrder?.title || '-' }}</el-descriptions-item>
        <el-descriptions-item label="状态">{{ orderStatusText(detailOrder?.status) }}</el-descriptions-item>
        <el-descriptions-item label="实例">{{ detailOrder?.instance?.name || '-' }}</el-descriptions-item>
        <el-descriptions-item label="数据库">{{ detailOrder?.database || '-' }}</el-descriptions-item>
        <el-descriptions-item label="SQL类型">{{ detailOrder?.sql_type || '-' }}</el-descriptions-item>
        <el-descriptions-item label="风险等级">{{ auditLevelText(detailOrder?.audit_level) }}</el-descriptions-item>
      </el-descriptions>
      <el-card shadow="never" class="mb-3">
        <template #header>SQL 内容</template>
        <div class="pre-wrap">{{ detailOrder?.sql_content || '-' }}</div>
      </el-card>
      <el-card shadow="never" class="mb-3">
        <template #header>审核结果</template>
        <div class="pre-wrap">{{ detailOrder?.audit_result || '-' }}</div>
      </el-card>
      <el-card shadow="never">
        <template #header>回滚 SQL</template>
        <div class="pre-wrap">{{ detailOrder?.rollback_sql || '-' }}</div>
      </el-card>
    </el-drawer>
  </el-card>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const activeTab = ref('orders')

const instances = ref([])
const orders = ref([])
const logs = ref([])
const stats = ref({})

const instanceLoading = ref(false)
const orderLoading = ref(false)
const logLoading = ref(false)

const orderInstanceFilter = ref('')
const orderStatusFilter = ref('')
const logInstanceFilter = ref('')
const logTypeFilter = ref('')

const instanceVisible = ref(false)
const instanceDialogTitle = ref('新增实例')
const instanceEditId = ref('')
const instanceForm = ref({
  name: '',
  type: 'mysql',
  host: '',
  port: 3306,
  username: '',
  password: '',
  database: '',
  charset: 'utf8mb4',
  environment: 'prod',
  description: ''
})

const orderVisible = ref(false)
const orderForm = ref({
  title: '',
  instance_id: '',
  database: '',
  sql_content: ''
})

const reviewVisible = ref(false)
const reviewOrderId = ref('')
const reviewForm = ref({ approved: true, remark: '' })

const detailVisible = ref(false)
const detailOrder = ref(null)

const orderStatusOptions = [
  { label: '待审核', value: 0 },
  { label: '审核通过', value: 1 },
  { label: '审核拒绝', value: 2 },
  { label: '执行中', value: 3 },
  { label: '执行成功', value: 4 },
  { label: '执行失败', value: 5 },
  { label: '已回滚', value: 6 }
]

const statOrders = computed(() => stats.value.orders || {})

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const formatTime = (v) => {
  if (!v) return '-'
  return String(v).slice(0, 19).replace('T', ' ')
}

const orderStatusText = (v) => ({
  0: '待审核',
  1: '审核通过',
  2: '审核拒绝',
  3: '执行中',
  4: '执行成功',
  5: '执行失败',
  6: '已回滚'
}[v] || '-')

const orderStatusType = (v) => ({ 0: 'warning', 1: 'success', 2: 'danger', 3: 'primary', 4: 'success', 5: 'danger', 6: 'info' }[v] || 'info')
const auditLevelText = (v) => ({ 0: '通过', 1: '警告', 2: '错误' }[v] || '-')
const auditLevelType = (v) => ({ 0: 'success', 1: 'warning', 2: 'danger' }[v] || 'info')

const fetchStats = async () => {
  try {
    const res = await axios.get('/api/v1/sqlaudit/statistics', { headers: authHeaders() })
    stats.value = res.data?.data || {}
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '获取统计失败')
  }
}

const fetchInstances = async () => {
  instanceLoading.value = true
  try {
    const res = await axios.get('/api/v1/sqlaudit/instances', { headers: authHeaders() })
    instances.value = res.data?.data || []
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '获取实例失败')
  } finally {
    instanceLoading.value = false
  }
}

const fetchOrders = async () => {
  orderLoading.value = true
  try {
    const params = {}
    if (orderInstanceFilter.value) params.instance_id = orderInstanceFilter.value
    if (orderStatusFilter.value !== '' && orderStatusFilter.value !== null) params.status = orderStatusFilter.value
    const res = await axios.get('/api/v1/sqlaudit/orders', { headers: authHeaders(), params })
    orders.value = res.data?.data || []
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '获取工单失败')
  } finally {
    orderLoading.value = false
  }
}

const fetchLogs = async () => {
  logLoading.value = true
  try {
    const params = {}
    if (logInstanceFilter.value) params.instance_id = logInstanceFilter.value
    if (logTypeFilter.value) params.sql_type = logTypeFilter.value
    const res = await axios.get('/api/v1/sqlaudit/logs', { headers: authHeaders(), params })
    logs.value = res.data?.data || []
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '获取日志失败')
  } finally {
    logLoading.value = false
  }
}

const reloadAll = async () => {
  await Promise.all([fetchStats(), fetchInstances(), fetchOrders(), fetchLogs()])
}

const openCreateInstance = () => {
  instanceDialogTitle.value = '新增实例'
  instanceEditId.value = ''
  instanceForm.value = {
    name: '', type: 'mysql', host: '', port: 3306, username: '', password: '', database: '', charset: 'utf8mb4', environment: 'prod', description: ''
  }
  instanceVisible.value = true
}

const openEditInstance = (row) => {
  instanceDialogTitle.value = '编辑实例'
  instanceEditId.value = row.id
  instanceForm.value = {
    name: row.name || '',
    type: row.type || 'mysql',
    host: row.host || '',
    port: Number(row.port || 3306),
    username: row.username || '',
    password: row.password || '',
    database: row.database || '',
    charset: row.charset || 'utf8mb4',
    environment: row.environment || 'prod',
    description: row.description || ''
  }
  instanceVisible.value = true
}

const submitInstance = async () => {
  if (!instanceForm.value.name.trim() || !instanceForm.value.host.trim()) {
    ElMessage.warning('请填写实例名称和地址')
    return
  }
  try {
    if (instanceEditId.value) {
      await axios.put(`/api/v1/sqlaudit/instances/${instanceEditId.value}`, instanceForm.value, { headers: authHeaders() })
      ElMessage.success('更新成功')
    } else {
      await axios.post('/api/v1/sqlaudit/instances', instanceForm.value, { headers: authHeaders() })
      ElMessage.success('创建成功')
    }
    instanceVisible.value = false
    await fetchInstances()
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '保存失败')
  }
}

const testInstance = async (row) => {
  try {
    await axios.post(`/api/v1/sqlaudit/instances/${row.id}/test`, {}, { headers: authHeaders() })
    ElMessage.success('连接成功')
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '连接失败')
  }
}

const removeInstance = async (row) => {
  try {
    await ElMessageBox.confirm(`确认删除实例 ${row.name} 吗？`, '提示', { type: 'warning' })
    await axios.delete(`/api/v1/sqlaudit/instances/${row.id}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    await fetchInstances()
  } catch (err) {
    if (err !== 'cancel') ElMessage.error(err.response?.data?.message || '删除失败')
  }
}

const openCreateOrder = () => {
  orderForm.value = { title: '', instance_id: '', database: '', sql_content: '' }
  orderVisible.value = true
}

const submitOrder = async () => {
  if (!orderForm.value.title.trim() || !orderForm.value.instance_id || !orderForm.value.sql_content.trim()) {
    ElMessage.warning('请填写标题、实例和 SQL 内容')
    return
  }
  try {
    await axios.post('/api/v1/sqlaudit/orders', orderForm.value, { headers: authHeaders() })
    ElMessage.success('工单创建成功')
    orderVisible.value = false
    await Promise.all([fetchOrders(), fetchStats()])
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '创建工单失败')
  }
}

const openOrderDetail = async (row) => {
  try {
    const res = await axios.get(`/api/v1/sqlaudit/orders/${row.id}`, { headers: authHeaders() })
    detailOrder.value = res.data?.data || null
    detailVisible.value = true
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '读取详情失败')
  }
}

const openReview = (row, approved) => {
  reviewOrderId.value = row.id
  reviewForm.value = { approved, remark: '' }
  reviewVisible.value = true
}

const submitReview = async () => {
  try {
    await axios.post(`/api/v1/sqlaudit/orders/${reviewOrderId.value}/review`, reviewForm.value, { headers: authHeaders() })
    ElMessage.success('审核完成')
    reviewVisible.value = false
    await Promise.all([fetchOrders(), fetchStats()])
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '审核失败')
  }
}

const executeOrder = async (row) => {
  try {
    await axios.post(`/api/v1/sqlaudit/orders/${row.id}/execute`, {}, { headers: authHeaders() })
    ElMessage.success('执行完成')
    await Promise.all([fetchOrders(), fetchLogs(), fetchStats()])
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '执行失败')
    await Promise.all([fetchOrders(), fetchLogs(), fetchStats()])
  }
}

onMounted(reloadAll)
</script>

<style scoped>
.page-card { max-width: 1280px; margin: 0 auto; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; gap: 12px; margin-bottom: 12px; }
.page-desc { margin: 4px 0 0; color: #606266; }
.page-actions { display: flex; gap: 8px; align-items: center; }
.summary-row { margin-bottom: 12px; }
.metric-card { min-height: 86px; }
.metric-title { color: #909399; font-size: 13px; margin-bottom: 8px; }
.metric-value { font-size: 24px; font-weight: 700; }
.filter-row { display: flex; align-items: center; gap: 8px; margin-bottom: 12px; flex-wrap: wrap; }
.w-40 { width: 160px; }
.w-52 { width: 220px; }
.mb-3 { margin-bottom: 12px; }
.pre-wrap { white-space: pre-wrap; line-height: 1.5; }
</style>
