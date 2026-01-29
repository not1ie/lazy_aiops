<template>
  <div class="sql-audit-page">
    <div class="page-header">
      <h1>SQL 审计管理</h1>
      <p>SQL工单管理、智能审核、执行审计</p>
    </div>

    <!-- 统计卡片 -->
    <div class="stats-grid">
      <AppleCard class="stat-card">
        <div class="stat-content">
          <i class="fas fa-file-alt stat-icon"></i>
          <div class="stat-info">
            <div class="stat-value">{{ stats.orders?.total || 0 }}</div>
            <div class="stat-label">总工单数</div>
          </div>
        </div>
      </AppleCard>
      <AppleCard class="stat-card">
        <div class="stat-content">
          <i class="fas fa-clock stat-icon pending"></i>
          <div class="stat-info">
            <div class="stat-value">{{ stats.orders?.pending || 0 }}</div>
            <div class="stat-label">待审核</div>
          </div>
        </div>
      </AppleCard>
      <AppleCard class="stat-card">
        <div class="stat-content">
          <i class="fas fa-check-circle stat-icon success"></i>
          <div class="stat-info">
            <div class="stat-value">{{ stats.orders?.executed || 0 }}</div>
            <div class="stat-label">已执行</div>
          </div>
        </div>
      </AppleCard>
      <AppleCard class="stat-card">
        <div class="stat-content">
          <i class="fas fa-times-circle stat-icon error"></i>
          <div class="stat-info">
            <div class="stat-value">{{ stats.orders?.failed || 0 }}</div>
            <div class="stat-label">执行失败</div>
          </div>
        </div>
      </AppleCard>
    </div>

    <!-- 工单列表 -->
    <AppleCard class="orders-section">
      <div class="section-header">
        <h2>SQL工单</h2>
        <div class="actions">
          <AppleButton @click="showCreateModal = true" type="primary">
            <i class="fas fa-plus"></i> 创建工单
          </AppleButton>
        </div>
      </div>

      <div class="filters">
        <AppleSelect v-model="filters.status" placeholder="状态筛选">
          <option value="">全部状态</option>
          <option value="0">待审核</option>
          <option value="1">审核通过</option>
          <option value="2">审核拒绝</option>
          <option value="3">执行中</option>
          <option value="4">执行成功</option>
          <option value="5">执行失败</option>
        </AppleSelect>
        <AppleButton @click="loadOrders">
          <i class="fas fa-sync"></i> 刷新
        </AppleButton>
      </div>

      <AppleTable :columns="orderColumns" :data="orders" :loading="loading">
        <template #status="{ row }">
          <AppleBadge :type="getStatusType(row.status)">
            {{ getStatusText(row.status) }}
          </AppleBadge>
        </template>
        <template #sql_type="{ row }">
          <AppleBadge :type="getSQLTypeColor(row.sql_type)">
            {{ row.sql_type }}
          </AppleBadge>
        </template>
        <template #audit_level="{ row }">
          <AppleBadge :type="getAuditLevelType(row.audit_level)">
            {{ getAuditLevelText(row.audit_level) }}
          </AppleBadge>
        </template>
        <template #actions="{ row }">
          <div class="action-buttons">
            <AppleButton size="small" @click="viewOrder(row)">
              <i class="fas fa-eye"></i>
            </AppleButton>
            <AppleButton 
              v-if="row.status === 0" 
              size="small" 
              type="success"
              @click="reviewOrder(row, true)">
              <i class="fas fa-check"></i>
            </AppleButton>
            <AppleButton 
              v-if="row.status === 0" 
              size="small" 
              type="danger"
              @click="reviewOrder(row, false)">
              <i class="fas fa-times"></i>
            </AppleButton>
            <AppleButton 
              v-if="row.status === 1" 
              size="small" 
              type="primary"
              @click="executeOrder(row)">
              <i class="fas fa-play"></i>
            </AppleButton>
          </div>
        </template>
      </AppleTable>
    </AppleCard>

    <!-- 创建工单模态框 -->
    <AppleModal v-model="showCreateModal" title="创建SQL工单" width="800px">
      <div class="create-form">
        <div class="form-group">
          <label>工单标题</label>
          <AppleInput v-model="newOrder.title" placeholder="请输入工单标题" />
        </div>
        <div class="form-group">
          <label>数据库实例</label>
          <AppleSelect v-model="newOrder.instance_id" placeholder="选择数据库实例">
            <option v-for="inst in instances" :key="inst.id" :value="inst.id">
              {{ inst.name }} ({{ inst.host }}:{{ inst.port }})
            </option>
          </AppleSelect>
        </div>
        <div class="form-group">
          <label>数据库名</label>
          <AppleInput v-model="newOrder.database" placeholder="请输入数据库名" />
        </div>
        <div class="form-group">
          <label>SQL语句</label>
          <textarea 
            v-model="newOrder.sql_content" 
            class="sql-textarea"
            placeholder="请输入SQL语句"
            rows="10"
          ></textarea>
        </div>
        <div class="form-actions">
          <AppleButton @click="analyzeSQL" :loading="analyzing">
            <i class="fas fa-search"></i> 分析SQL
          </AppleButton>
          <AppleButton type="primary" @click="createOrder" :loading="creating">
            <i class="fas fa-check"></i> 提交工单
          </AppleButton>
        </div>
        
        <!-- 分析结果 -->
        <div v-if="analysisResult" class="analysis-result">
          <h3>分析结果</h3>
          <div class="result-item">
            <span class="label">SQL类型:</span>
            <AppleBadge :type="getSQLTypeColor(analysisResult.sql_type)">
              {{ analysisResult.sql_type }}
            </AppleBadge>
          </div>
          <div class="result-item">
            <span class="label">风险等级:</span>
            <AppleBadge :type="getRiskLevelType(analysisResult.risk_level)">
              {{ analysisResult.risk_level }}
            </AppleBadge>
          </div>
          <div class="result-item">
            <span class="label">涉及表:</span>
            <span>{{ analysisResult.table_names?.join(', ') || '无' }}</span>
          </div>
          <div v-if="analysisResult.issues?.length" class="issues">
            <h4>发现问题:</h4>
            <div v-for="(issue, idx) in analysisResult.issues" :key="idx" class="issue-item">
              <AppleBadge :type="getIssueLevelType(issue.level)">
                {{ issue.type }}
              </AppleBadge>
              <span>{{ issue.message }}</span>
            </div>
          </div>
          <div v-if="analysisResult.suggestions?.length" class="suggestions">
            <h4>建议:</h4>
            <ul>
              <li v-for="(sug, idx) in analysisResult.suggestions" :key="idx">{{ sug }}</li>
            </ul>
          </div>
        </div>
      </div>
    </AppleModal>

    <!-- 工单详情模态框 -->
    <AppleModal v-model="showDetailModal" title="工单详情" width="900px">
      <div v-if="selectedOrder" class="order-detail">
        <div class="detail-section">
          <h3>基本信息</h3>
          <div class="detail-grid">
            <div class="detail-item">
              <span class="label">工单标题:</span>
              <span>{{ selectedOrder.title }}</span>
            </div>
            <div class="detail-item">
              <span class="label">状态:</span>
              <AppleBadge :type="getStatusType(selectedOrder.status)">
                {{ getStatusText(selectedOrder.status) }}
              </AppleBadge>
            </div>
            <div class="detail-item">
              <span class="label">SQL类型:</span>
              <AppleBadge :type="getSQLTypeColor(selectedOrder.sql_type)">
                {{ selectedOrder.sql_type }}
              </AppleBadge>
            </div>
            <div class="detail-item">
              <span class="label">审核级别:</span>
              <AppleBadge :type="getAuditLevelType(selectedOrder.audit_level)">
                {{ getAuditLevelText(selectedOrder.audit_level) }}
              </AppleBadge>
            </div>
          </div>
        </div>

        <div class="detail-section">
          <h3>SQL语句</h3>
          <pre class="sql-content">{{ selectedOrder.sql_content }}</pre>
        </div>

        <div v-if="selectedOrder.audit_result" class="detail-section">
          <h3>审核结果</h3>
          <pre class="audit-result">{{ selectedOrder.audit_result }}</pre>
        </div>

        <div v-if="selectedOrder.rollback_sql" class="detail-section">
          <h3>回滚SQL</h3>
          <pre class="sql-content">{{ selectedOrder.rollback_sql }}</pre>
        </div>

        <div class="detail-section">
          <h3>执行信息</h3>
          <div class="detail-grid">
            <div class="detail-item">
              <span class="label">提交人:</span>
              <span>{{ selectedOrder.submitter }}</span>
            </div>
            <div class="detail-item">
              <span class="label">审核人:</span>
              <span>{{ selectedOrder.reviewer || '-' }}</span>
            </div>
            <div class="detail-item">
              <span class="label">执行人:</span>
              <span>{{ selectedOrder.executor || '-' }}</span>
            </div>
            <div class="detail-item">
              <span class="label">影响行数:</span>
              <span>{{ selectedOrder.affected_rows || 0 }}</span>
            </div>
            <div class="detail-item">
              <span class="label">执行时间:</span>
              <span>{{ selectedOrder.execute_time || 0 }}ms</span>
            </div>
          </div>
        </div>
      </div>
    </AppleModal>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import AppleCard from '../components/AppleCard.vue'
import AppleButton from '../components/AppleButton.vue'
import AppleTable from '../components/AppleTable.vue'
import AppleBadge from '../components/AppleBadge.vue'
import AppleModal from '../components/AppleModal.vue'
import AppleInput from '../components/AppleInput.vue'
import AppleSelect from '../components/AppleSelect.vue'
import api from '../api'

const stats = ref({})
const orders = ref([])
const instances = ref([])
const loading = ref(false)
const showCreateModal = ref(false)
const showDetailModal = ref(false)
const selectedOrder = ref(null)
const analyzing = ref(false)
const creating = ref(false)
const analysisResult = ref(null)

const filters = ref({
  status: ''
})

const newOrder = ref({
  title: '',
  instance_id: '',
  database: '',
  sql_content: ''
})

const orderColumns = [
  { key: 'title', label: '标题', width: '200px' },
  { key: 'sql_type', label: 'SQL类型', width: '120px', slot: true },
  { key: 'database', label: '数据库', width: '120px' },
  { key: 'status', label: '状态', width: '100px', slot: true },
  { key: 'audit_level', label: '审核级别', width: '100px', slot: true },
  { key: 'submitter', label: '提交人', width: '100px' },
  { key: 'created_at', label: '创建时间', width: '160px' },
  { key: 'actions', label: '操作', width: '180px', slot: true }
]

const getStatusType = (status) => {
  const types = {
    0: 'warning', 1: 'info', 2: 'danger',
    3: 'info', 4: 'success', 5: 'danger'
  }
  return types[status] || 'default'
}

const getStatusText = (status) => {
  const texts = {
    0: '待审核', 1: '审核通过', 2: '审核拒绝',
    3: '执行中', 4: '执行成功', 5: '执行失败'
  }
  return texts[status] || '未知'
}

const getSQLTypeColor = (type) => {
  if (type?.startsWith('DDL')) return 'danger'
  if (type?.startsWith('DML')) return 'warning'
  if (type === 'DQL') return 'info'
  return 'default'
}

const getAuditLevelType = (level) => {
  return level === 0 ? 'success' : level === 1 ? 'warning' : 'danger'
}

const getAuditLevelText = (level) => {
  return level === 0 ? '通过' : level === 1 ? '警告' : '错误'
}

const getRiskLevelType = (level) => {
  const types = { low: 'success', medium: 'warning', high: 'danger', critical: 'danger' }
  return types[level] || 'default'
}

const getIssueLevelType = (level) => {
  return level === 0 ? 'info' : level === 1 ? 'warning' : 'danger'
}

const loadStats = async () => {
  try {
    const res = await api.get('/sqlaudit/statistics')
    if (res.data.code === 0) {
      stats.value = res.data.data
    }
  } catch (error) {
    console.error('加载统计失败:', error)
  }
}

const loadOrders = async () => {
  loading.value = true
  try {
    const params = {}
    if (filters.value.status) params.status = filters.value.status
    const res = await api.get('/sqlaudit/orders', { params })
    if (res.data.code === 0) {
      orders.value = res.data.data || []
    }
  } catch (error) {
    console.error('加载工单失败:', error)
  } finally {
    loading.value = false
  }
}

const loadInstances = async () => {
  try {
    const res = await api.get('/sqlaudit/instances')
    if (res.data.code === 0) {
      instances.value = res.data.data || []
    }
  } catch (error) {
    console.error('加载实例失败:', error)
  }
}

const analyzeSQL = async () => {
  if (!newOrder.value.sql_content) {
    alert('请输入SQL语句')
    return
  }
  analyzing.value = true
  try {
    const res = await api.post('/sqlaudit/analyze', {
      sql_content: newOrder.value.sql_content
    })
    if (res.data.code === 0) {
      analysisResult.value = res.data.data
    }
  } catch (error) {
    console.error('分析SQL失败:', error)
    alert('分析失败')
  } finally {
    analyzing.value = false
  }
}

const createOrder = async () => {
  if (!newOrder.value.title || !newOrder.value.instance_id || 
      !newOrder.value.database || !newOrder.value.sql_content) {
    alert('请填写完整信息')
    return
  }
  creating.value = true
  try {
    const res = await api.post('/sqlaudit/orders', newOrder.value)
    if (res.data.code === 0) {
      alert('工单创建成功')
      showCreateModal.value = false
      newOrder.value = { title: '', instance_id: '', database: '', sql_content: '' }
      analysisResult.value = null
      loadOrders()
      loadStats()
    }
  } catch (error) {
    console.error('创建工单失败:', error)
    alert('创建失败')
  } finally {
    creating.value = false
  }
}

const viewOrder = (order) => {
  selectedOrder.value = order
  showDetailModal.value = true
}

const reviewOrder = async (order, approved) => {
  const remark = prompt(approved ? '请输入审核意见:' : '请输入拒绝原因:')
  if (remark === null) return
  
  try {
    const res = await api.post(`/sqlaudit/orders/${order.id}/review`, {
      approved,
      remark
    })
    if (res.data.code === 0) {
      alert('审核完成')
      loadOrders()
      loadStats()
    }
  } catch (error) {
    console.error('审核失败:', error)
    alert('审核失败')
  }
}

const executeOrder = async (order) => {
  if (!confirm('确认执行此SQL工单?')) return
  
  try {
    const res = await api.post(`/sqlaudit/orders/${order.id}/execute`)
    if (res.data.code === 0) {
      alert(`执行成功\n影响行数: ${res.data.data.affected_rows}\n执行时间: ${res.data.data.execute_time}ms`)
      loadOrders()
      loadStats()
    }
  } catch (error) {
    console.error('执行失败:', error)
    alert('执行失败: ' + (error.response?.data?.message || error.message))
  }
}

onMounted(() => {
  loadStats()
  loadOrders()
  loadInstances()
})
</script>

<style scoped>
.sql-audit-page {
  padding: 24px;
}

.page-header {
  margin-bottom: 24px;
}

.page-header h1 {
  font-size: 28px;
  font-weight: 600;
  color: #fff;
  margin: 0 0 8px 0;
}

.page-header p {
  color: rgba(255, 255, 255, 0.6);
  margin: 0;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}

.stat-card {
  padding: 20px;
}

.stat-content {
  display: flex;
  align-items: center;
  gap: 16px;
}

.stat-icon {
  font-size: 32px;
  color: #007AFF;
}

.stat-icon.pending { color: #FF9500; }
.stat-icon.success { color: #34C759; }
.stat-icon.error { color: #FF3B30; }

.stat-info {
  flex: 1;
}

.stat-value {
  font-size: 32px;
  font-weight: 600;
  color: #fff;
  line-height: 1;
  margin-bottom: 4px;
}

.stat-label {
  font-size: 14px;
  color: rgba(255, 255, 255, 0.6);
}

.orders-section {
  padding: 24px;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.section-header h2 {
  font-size: 20px;
  font-weight: 600;
  color: #fff;
  margin: 0;
}

.filters {
  display: flex;
  gap: 12px;
  margin-bottom: 20px;
}

.action-buttons {
  display: flex;
  gap: 8px;
}

.create-form {
  padding: 20px;
}

.form-group {
  margin-bottom: 20px;
}

.form-group label {
  display: block;
  margin-bottom: 8px;
  color: #fff;
  font-weight: 500;
}

.sql-textarea {
  width: 100%;
  padding: 12px;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 8px;
  color: #fff;
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 13px;
  resize: vertical;
}

.form-actions {
  display: flex;
  gap: 12px;
  margin-bottom: 20px;
}

.analysis-result {
  padding: 20px;
  background: rgba(255, 255, 255, 0.03);
  border-radius: 8px;
  border: 1px solid rgba(255, 255, 255, 0.1);
}

.analysis-result h3 {
  margin: 0 0 16px 0;
  color: #fff;
  font-size: 16px;
}

.result-item {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.result-item .label {
  color: rgba(255, 255, 255, 0.6);
  min-width: 80px;
}

.issues, .suggestions {
  margin-top: 16px;
}

.issues h4, .suggestions h4 {
  margin: 0 0 12px 0;
  color: #fff;
  font-size: 14px;
}

.issue-item {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 8px;
  padding: 8px;
  background: rgba(255, 255, 255, 0.03);
  border-radius: 6px;
}

.suggestions ul {
  margin: 0;
  padding-left: 20px;
  color: rgba(255, 255, 255, 0.8);
}

.suggestions li {
  margin-bottom: 8px;
}

.order-detail {
  padding: 20px;
}

.detail-section {
  margin-bottom: 24px;
}

.detail-section h3 {
  margin: 0 0 16px 0;
  color: #fff;
  font-size: 16px;
  font-weight: 600;
}

.detail-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}

.detail-item {
  display: flex;
  align-items: center;
  gap: 12px;
}

.detail-item .label {
  color: rgba(255, 255, 255, 0.6);
  min-width: 80px;
}

.sql-content, .audit-result {
  padding: 16px;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 8px;
  color: #fff;
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 13px;
  white-space: pre-wrap;
  word-break: break-all;
  margin: 0;
}
</style>
