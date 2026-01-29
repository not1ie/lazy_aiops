<template>
  <div class="alert-page">
    <div class="page-header">
      <h1>告警管理</h1>
      <AppleButton type="primary" icon="fas fa-plus" @click="showAddModal = true">
        新建告警规则
      </AppleButton>
    </div>
    
    <!-- 告警统计 -->
    <div class="alert-stats">
      <AppleCard hoverable @click="filterLevel = 'all'">
        <div class="stat-item">
          <i class="fas fa-bell stat-icon" style="color: var(--apple-accent)"></i>
          <div class="stat-content">
            <div class="stat-value">{{ stats.total }}</div>
            <div class="stat-label">总告警</div>
          </div>
        </div>
      </AppleCard>
      
      <AppleCard hoverable @click="filterLevel = 'critical'">
        <div class="stat-item">
          <i class="fas fa-exclamation-circle stat-icon" style="color: var(--apple-danger)"></i>
          <div class="stat-content">
            <div class="stat-value">{{ stats.critical }}</div>
            <div class="stat-label">严重</div>
          </div>
        </div>
      </AppleCard>
      
      <AppleCard hoverable @click="filterLevel = 'warning'">
        <div class="stat-item">
          <i class="fas fa-exclamation-triangle stat-icon" style="color: var(--apple-warning)"></i>
          <div class="stat-content">
            <div class="stat-value">{{ stats.warning }}</div>
            <div class="stat-label">警告</div>
          </div>
        </div>
      </AppleCard>
      
      <AppleCard hoverable @click="filterLevel = 'info'">
        <div class="stat-item">
          <i class="fas fa-info-circle stat-icon" style="color: var(--apple-success)"></i>
          <div class="stat-content">
            <div class="stat-value">{{ stats.info }}</div>
            <div class="stat-label">提示</div>
          </div>
        </div>
      </AppleCard>
    </div>
    
    <!-- 告警列表 -->
    <AppleCard title="告警列表">
      <template #extra>
        <div class="filter-bar">
          <AppleSelect
            v-model="filterStatus"
            :options="statusOptions"
            style="width: 120px"
          />
          <AppleInput
            v-model="searchQuery"
            placeholder="搜索告警..."
            icon="fas fa-search"
            style="width: 250px"
          />
        </div>
      </template>
      
      <AppleTable :columns="columns" :data="filteredAlerts" @row-click="handleAlertClick">
        <template #cell-level="{ value }">
          <AppleBadge :type="getLevelType(value)">
            {{ getLevelText(value) }}
          </AppleBadge>
        </template>
        <template #cell-status="{ value }">
          <AppleBadge :type="value === 'resolved' ? 'success' : 'warning'">
            {{ value === 'resolved' ? '已处理' : '待处理' }}
          </AppleBadge>
        </template>
        <template #cell-actions="{ row }">
          <div class="action-buttons">
            <button class="action-btn" @click.stop="handleResolve(row)">
              <i class="fas fa-check"></i>
            </button>
            <button class="action-btn" @click.stop="handleDelete(row)">
              <i class="fas fa-trash"></i>
            </button>
          </div>
        </template>
      </AppleTable>
    </AppleCard>
    
    <!-- 新建告警规则模态框 -->
    <AppleModal v-model="showAddModal" title="新建告警规则">
      <div class="form-group">
        <AppleInput v-model="newRule.name" label="规则名称" placeholder="请输入规则名称" />
      </div>
      <div class="form-group">
        <AppleSelect
          v-model="newRule.metric"
          label="监控指标"
          :options="metricOptions"
        />
      </div>
      <div class="form-group">
        <AppleSelect
          v-model="newRule.operator"
          label="条件"
          :options="operatorOptions"
        />
      </div>
      <div class="form-group">
        <AppleInput v-model="newRule.threshold" label="阈值" type="number" placeholder="请输入阈值" />
      </div>
      <div class="form-group">
        <AppleSelect
          v-model="newRule.level"
          label="告警级别"
          :options="levelOptions"
        />
      </div>
      
      <template #footer>
        <AppleButton type="ghost" @click="showAddModal = false">取消</AppleButton>
        <AppleButton type="primary" @click="handleAddRule">确定</AppleButton>
      </template>
    </AppleModal>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import AppleCard from '../components/AppleCard.vue'
import AppleButton from '../components/AppleButton.vue'
import AppleInput from '../components/AppleInput.vue'
import AppleSelect from '../components/AppleSelect.vue'
import AppleTable from '../components/AppleTable.vue'
import AppleBadge from '../components/AppleBadge.vue'
import AppleModal from '../components/AppleModal.vue'

const filterLevel = ref('all')
const filterStatus = ref('all')
const searchQuery = ref('')
const showAddModal = ref(false)

const stats = ref({
  total: 156,
  critical: 12,
  warning: 45,
  info: 99
})

const statusOptions = [
  { label: '全部状态', value: 'all' },
  { label: '待处理', value: 'pending' },
  { label: '已处理', value: 'resolved' }
]

const metricOptions = [
  { label: 'CPU使用率', value: 'cpu' },
  { label: '内存使用率', value: 'memory' },
  { label: '磁盘使用率', value: 'disk' },
  { label: '网络流量', value: 'network' }
]

const operatorOptions = [
  { label: '大于', value: '>' },
  { label: '大于等于', value: '>=' },
  { label: '小于', value: '<' },
  { label: '小于等于', value: '<=' },
  { label: '等于', value: '=' }
]

const levelOptions = [
  { label: '严重', value: 'critical' },
  { label: '警告', value: 'warning' },
  { label: '提示', value: 'info' }
]

const newRule = ref({
  name: '',
  metric: 'cpu',
  operator: '>',
  threshold: '',
  level: 'warning'
})

const columns = [
  { key: 'time', title: '时间', width: '180px' },
  { key: 'title', title: '告警标题', width: '250px' },
  { key: 'level', title: '级别', width: '100px' },
  { key: 'service', title: '服务', width: '150px' },
  { key: 'status', title: '状态', width: '100px' },
  { key: 'actions', title: '操作', width: '120px' }
]

const alerts = ref([
  {
    id: 1,
    time: '2026-01-23 18:30:15',
    title: 'CPU使用率过高',
    level: 'critical',
    service: 'web-01',
    status: 'pending'
  },
  {
    id: 2,
    time: '2026-01-23 18:25:10',
    title: '磁盘空间不足',
    level: 'warning',
    service: 'db-01',
    status: 'pending'
  },
  {
    id: 3,
    time: '2026-01-23 18:20:05',
    title: '服务响应缓慢',
    level: 'warning',
    service: 'api-01',
    status: 'resolved'
  },
  {
    id: 4,
    time: '2026-01-23 18:15:00',
    title: '内存使用率正常',
    level: 'info',
    service: 'cache-01',
    status: 'resolved'
  }
])

const filteredAlerts = computed(() => {
  return alerts.value.filter(alert => {
    const matchLevel = filterLevel.value === 'all' || alert.level === filterLevel.value
    const matchStatus = filterStatus.value === 'all' || alert.status === filterStatus.value
    const matchSearch = !searchQuery.value || 
      alert.title.includes(searchQuery.value) ||
      alert.service.includes(searchQuery.value)
    return matchLevel && matchStatus && matchSearch
  })
})

const getLevelType = (level) => {
  const map = {
    critical: 'danger',
    warning: 'warning',
    info: 'success'
  }
  return map[level] || 'info'
}

const getLevelText = (level) => {
  const map = {
    critical: '严重',
    warning: '警告',
    info: '提示'
  }
  return map[level] || level
}

const handleAlertClick = (alert) => {
  console.log('Alert clicked:', alert)
}

const handleResolve = (alert) => {
  alert.status = 'resolved'
  console.log('Resolved:', alert)
}

const handleDelete = (alert) => {
  const index = alerts.value.findIndex(a => a.id === alert.id)
  if (index > -1) {
    alerts.value.splice(index, 1)
  }
}

const handleAddRule = () => {
  console.log('Add rule:', newRule.value)
  showAddModal.value = false
}
</script>

<style scoped>
.alert-page {
  padding: var(--space-xl);
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-2xl);
}

.page-header h1 {
  font-size: 32px;
  font-weight: 700;
}

.alert-stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: var(--space-lg);
  margin-bottom: var(--space-2xl);
}

.stat-item {
  display: flex;
  align-items: center;
  gap: var(--space-lg);
  cursor: pointer;
}

.stat-icon {
  font-size: 32px;
}

.stat-content {
  flex: 1;
}

.stat-value {
  font-size: 28px;
  font-weight: 700;
  color: var(--apple-text-primary);
}

.stat-label {
  font-size: 14px;
  color: var(--apple-text-secondary);
  margin-top: var(--space-xs);
}

.filter-bar {
  display: flex;
  gap: var(--space-md);
  align-items: center;
}

.action-buttons {
  display: flex;
  gap: var(--space-xs);
}

.action-btn {
  background: transparent;
  border: none;
  color: var(--apple-text-secondary);
  cursor: pointer;
  padding: var(--space-sm);
  border-radius: var(--radius-sm);
  transition: all 0.2s var(--ease-standard);
}

.action-btn:hover {
  background: rgba(10, 132, 255, 0.1);
  color: var(--apple-accent);
}

.form-group {
  margin-bottom: var(--space-lg);
}
</style>
