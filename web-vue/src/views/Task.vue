<template>
  <div class="task-page">
    <div class="page-header">
      <h1>任务调度</h1>
      <AppleButton type="primary" icon="fas fa-plus" @click="showAddModal = true">
        新建任务
      </AppleButton>
    </div>
    
    <!-- 任务统计 -->
    <div class="task-stats">
      <AppleCard hoverable>
        <div class="stat-item">
          <i class="fas fa-tasks stat-icon" style="color: var(--apple-accent)"></i>
          <div class="stat-content">
            <div class="stat-value">{{ stats.total }}</div>
            <div class="stat-label">总任务</div>
          </div>
        </div>
      </AppleCard>
      
      <AppleCard hoverable>
        <div class="stat-item">
          <i class="fas fa-play-circle stat-icon" style="color: var(--apple-success)"></i>
          <div class="stat-content">
            <div class="stat-value">{{ stats.running }}</div>
            <div class="stat-label">运行中</div>
          </div>
        </div>
      </AppleCard>
      
      <AppleCard hoverable>
        <div class="stat-item">
          <i class="fas fa-check-circle stat-icon" style="color: var(--apple-success)"></i>
          <div class="stat-content">
            <div class="stat-value">{{ stats.success }}</div>
            <div class="stat-label">成功</div>
          </div>
        </div>
      </AppleCard>
      
      <AppleCard hoverable>
        <div class="stat-item">
          <i class="fas fa-times-circle stat-icon" style="color: var(--apple-danger)"></i>
          <div class="stat-content">
            <div class="stat-value">{{ stats.failed }}</div>
            <div class="stat-label">失败</div>
          </div>
        </div>
      </AppleCard>
    </div>
    
    <!-- 任务列表 -->
    <AppleCard title="任务列表">
      <template #extra>
        <AppleInput
          v-model="searchQuery"
          placeholder="搜索任务..."
          icon="fas fa-search"
          style="width: 250px"
        />
      </template>
      
      <AppleTable :columns="columns" :data="filteredTasks" @row-click="handleTaskClick">
        <template #cell-status="{ value }">
          <AppleBadge :type="getStatusType(value)">
            {{ getStatusText(value) }}
          </AppleBadge>
        </template>
        <template #cell-enabled="{ value }">
          <AppleBadge :type="value ? 'success' : 'info'">
            {{ value ? '启用' : '禁用' }}
          </AppleBadge>
        </template>
        <template #cell-actions="{ row }">
          <div class="action-buttons">
            <button class="action-btn" @click.stop="handleRun(row)" :title="'立即执行'">
              <i class="fas fa-play"></i>
            </button>
            <button class="action-btn" @click.stop="handleEdit(row)" :title="'编辑'">
              <i class="fas fa-edit"></i>
            </button>
            <button class="action-btn" @click.stop="handleToggle(row)" :title="row.enabled ? '禁用' : '启用'">
              <i :class="row.enabled ? 'fas fa-pause' : 'fas fa-play'"></i>
            </button>
            <button class="action-btn danger" @click.stop="handleDelete(row)" :title="'删除'">
              <i class="fas fa-trash"></i>
            </button>
          </div>
        </template>
      </AppleTable>
    </AppleCard>
    
    <!-- 新建任务模态框 -->
    <AppleModal v-model="showAddModal" title="新建任务">
      <div class="form-group">
        <AppleInput v-model="newTask.name" label="任务名称" placeholder="请输入任务名称" />
      </div>
      <div class="form-group">
        <AppleSelect
          v-model="newTask.type"
          label="任务类型"
          :options="typeOptions"
        />
      </div>
      <div class="form-group">
        <AppleInput v-model="newTask.cron" label="Cron表达式" placeholder="0 0 * * *" />
        <small style="color: var(--apple-text-tertiary); margin-top: 4px; display: block;">
          例如: 0 0 * * * (每天0点执行)
        </small>
      </div>
      <div class="form-group">
        <AppleInput v-model="newTask.command" label="执行命令" placeholder="请输入命令" />
      </div>
      <div class="form-group">
        <AppleSelect
          v-model="newTask.target"
          label="目标服务器"
          :options="targetOptions"
        />
      </div>
      
      <template #footer>
        <AppleButton type="ghost" @click="showAddModal = false">取消</AppleButton>
        <AppleButton type="primary" @click="handleAddTask">确定</AppleButton>
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

const searchQuery = ref('')
const showAddModal = ref(false)

const stats = ref({
  total: 24,
  running: 3,
  success: 18,
  failed: 3
})

const typeOptions = [
  { label: 'Shell脚本', value: 'shell' },
  { label: 'Python脚本', value: 'python' },
  { label: 'HTTP请求', value: 'http' },
  { label: 'SQL查询', value: 'sql' }
]

const targetOptions = [
  { label: '所有服务器', value: 'all' },
  { label: 'web-01', value: 'web-01' },
  { label: 'web-02', value: 'web-02' },
  { label: 'db-01', value: 'db-01' }
]

const newTask = ref({
  name: '',
  type: 'shell',
  cron: '',
  command: '',
  target: 'all'
})

const columns = [
  { key: 'name', title: '任务名称', width: '200px' },
  { key: 'type', title: '类型', width: '100px' },
  { key: 'cron', title: 'Cron', width: '150px' },
  { key: 'status', title: '状态', width: '100px' },
  { key: 'enabled', title: '启用', width: '80px' },
  { key: 'lastRun', title: '最后执行', width: '180px' },
  { key: 'nextRun', title: '下次执行', width: '180px' },
  { key: 'actions', title: '操作', width: '180px' }
]

const tasks = ref([
  {
    id: 1,
    name: '数据库备份',
    type: 'shell',
    cron: '0 2 * * *',
    status: 'success',
    enabled: true,
    lastRun: '2026-01-23 02:00:00',
    nextRun: '2026-01-24 02:00:00'
  },
  {
    id: 2,
    name: '日志清理',
    type: 'shell',
    cron: '0 0 * * 0',
    status: 'success',
    enabled: true,
    lastRun: '2026-01-21 00:00:00',
    nextRun: '2026-01-28 00:00:00'
  },
  {
    id: 3,
    name: '健康检查',
    type: 'http',
    cron: '*/5 * * * *',
    status: 'running',
    enabled: true,
    lastRun: '2026-01-23 18:25:00',
    nextRun: '2026-01-23 18:30:00'
  },
  {
    id: 4,
    name: '数据同步',
    type: 'python',
    cron: '0 */6 * * *',
    status: 'failed',
    enabled: true,
    lastRun: '2026-01-23 12:00:00',
    nextRun: '2026-01-23 18:00:00'
  }
])

const filteredTasks = computed(() => {
  if (!searchQuery.value) return tasks.value
  return tasks.value.filter(task =>
    task.name.includes(searchQuery.value) ||
    task.type.includes(searchQuery.value)
  )
})

const getStatusType = (status) => {
  const map = {
    running: 'primary',
    success: 'success',
    failed: 'danger',
    pending: 'warning'
  }
  return map[status] || 'info'
}

const getStatusText = (status) => {
  const map = {
    running: '运行中',
    success: '成功',
    failed: '失败',
    pending: '等待中'
  }
  return map[status] || status
}

const handleTaskClick = (task) => {
  console.log('Task clicked:', task)
}

const handleRun = (task) => {
  task.status = 'running'
  console.log('Run task:', task)
}

const handleEdit = (task) => {
  console.log('Edit task:', task)
}

const handleToggle = (task) => {
  task.enabled = !task.enabled
  console.log('Toggle task:', task)
}

const handleDelete = (task) => {
  const index = tasks.value.findIndex(t => t.id === task.id)
  if (index > -1) {
    tasks.value.splice(index, 1)
  }
}

const handleAddTask = () => {
  console.log('Add task:', newTask.value)
  showAddModal.value = false
}
</script>

<style scoped>
.task-page {
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

.task-stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: var(--space-lg);
  margin-bottom: var(--space-2xl);
}

.stat-item {
  display: flex;
  align-items: center;
  gap: var(--space-lg);
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

.action-btn.danger:hover {
  background: rgba(255, 69, 58, 0.1);
  color: var(--apple-danger);
}

.form-group {
  margin-bottom: var(--space-lg);
}
</style>
