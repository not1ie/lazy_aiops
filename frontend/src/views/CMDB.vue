<template>
  <div class="cmdb-page">
    <div class="page-header">
      <h1>CMDB 资产管理</h1>
      <AppleButton type="primary" icon="fas fa-plus" @click="showAddDialog = true">
        添加资产
      </AppleButton>
    </div>
    
    <AppleCard>
      <div class="search-bar">
        <AppleInput
          v-model="searchQuery"
          placeholder="搜索资产..."
          icon="fas fa-search"
        />
      </div>
      
      <AppleTable :columns="columns" :data="filteredAssets" @row-click="handleRowClick">
        <template #cell-status="{ value }">
          <AppleBadge :type="value === 'online' ? 'success' : 'danger'">
            {{ value === 'online' ? '在线' : '离线' }}
          </AppleBadge>
        </template>
        <template #cell-cpu="{ value }">
          <div class="progress-cell">
            <div class="progress-bar">
              <div class="progress-fill" :style="{ width: value + '%', background: getProgressColor(value) }"></div>
            </div>
            <span>{{ value }}%</span>
          </div>
        </template>
        <template #cell-memory="{ value }">
          <div class="progress-cell">
            <div class="progress-bar">
              <div class="progress-fill" :style="{ width: value + '%', background: getProgressColor(value) }"></div>
            </div>
            <span>{{ value }}%</span>
          </div>
        </template>
        <template #cell-actions="{ row }">
          <div class="action-buttons">
            <button class="action-btn" @click.stop="editAsset(row)">
              <i class="fas fa-edit"></i>
            </button>
            <button class="action-btn danger" @click.stop="deleteAsset(row)">
              <i class="fas fa-trash"></i>
            </button>
          </div>
        </template>
      </AppleTable>
    </AppleCard>
    
    <!-- 添加资产模态框 -->
    <AppleModal v-model="showAddDialog" title="添加资产">
      <div class="form-group">
        <AppleInput v-model="newAsset.hostname" label="主机名" placeholder="请输入主机名" />
      </div>
      <div class="form-group">
        <AppleInput v-model="newAsset.ip" label="IP地址" placeholder="请输入IP地址" />
      </div>
      <div class="form-group">
        <AppleSelect
          v-model="newAsset.type"
          label="类型"
          :options="typeOptions"
        />
      </div>
      
      <template #footer>
        <AppleButton type="ghost" @click="showAddDialog = false">取消</AppleButton>
        <AppleButton type="primary" @click="handleAddAsset">确定</AppleButton>
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
const showAddDialog = ref(false)

const typeOptions = [
  { label: 'Web服务器', value: 'web' },
  { label: '数据库', value: 'database' },
  { label: '缓存服务器', value: 'cache' },
  { label: '应用服务器', value: 'app' }
]

const newAsset = ref({
  hostname: '',
  ip: '',
  type: 'web'
})

const columns = [
  { key: 'hostname', title: '主机名', width: '150px' },
  { key: 'ip', title: 'IP地址', width: '150px' },
  { key: 'type', title: '类型', width: '120px' },
  { key: 'status', title: '状态', width: '100px' },
  { key: 'cpu', title: 'CPU', width: '200px' },
  { key: 'memory', title: '内存', width: '200px' },
  { key: 'actions', title: '操作', width: '120px' }
]

const assets = ref([
  { id: 1, hostname: 'web-01', ip: '192.168.1.10', type: 'Web服务器', status: 'online', cpu: 45, memory: 62 },
  { id: 2, hostname: 'db-01', ip: '192.168.1.20', type: '数据库', status: 'online', cpu: 78, memory: 85 },
  { id: 3, hostname: 'cache-01', ip: '192.168.1.30', type: '缓存服务器', status: 'offline', cpu: 0, memory: 0 }
])

const filteredAssets = computed(() => {
  if (!searchQuery.value) return assets.value
  return assets.value.filter(asset =>
    asset.hostname.includes(searchQuery.value) ||
    asset.ip.includes(searchQuery.value)
  )
})

const getProgressColor = (value) => {
  if (value >= 80) return 'var(--apple-danger)'
  if (value >= 60) return 'var(--apple-warning)'
  return 'var(--apple-success)'
}

const handleRowClick = (row) => {
  console.log('Row clicked:', row)
}

const editAsset = (asset) => {
  console.log('Edit asset:', asset)
}

const deleteAsset = (asset) => {
  const index = assets.value.findIndex(a => a.id === asset.id)
  if (index > -1) {
    assets.value.splice(index, 1)
  }
}

const handleAddAsset = () => {
  console.log('Add asset:', newAsset.value)
  showAddDialog.value = false
}
</script>

<style scoped>
.cmdb-page {
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

.search-bar {
  margin-bottom: var(--space-lg);
  max-width: 400px;
}

.progress-cell {
  display: flex;
  align-items: center;
  gap: var(--space-md);
}

.progress-bar {
  flex: 1;
  height: 6px;
  background: var(--apple-bg-tertiary);
  border-radius: var(--radius-full);
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  border-radius: var(--radius-full);
  transition: width 0.3s var(--ease-standard);
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
