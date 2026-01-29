<template>
  <div class="ansible-page">
    <div class="page-header">
      <h1>Ansible 自动化</h1>
      <p>Playbook管理、批量执行、配置管理</p>
    </div>

    <!-- 标签页 -->
    <AppleCard class="tabs-section">
      <div class="tabs">
        <div 
          v-for="tab in tabs" 
          :key="tab.key"
          :class="['tab', { active: activeTab === tab.key }]"
          @click="activeTab = tab.key"
        >
          <i :class="tab.icon"></i>
          {{ tab.label }}
        </div>
      </div>

      <!-- Playbook列表 -->
      <div v-if="activeTab === 'playbooks'" class="tab-content">
        <div class="section-header">
          <h2>Playbook 列表</h2>
          <AppleButton @click="showPlaybookModal = true" type="primary">
            <i class="fas fa-plus"></i> 创建 Playbook
          </AppleButton>
        </div>
        <AppleTable :columns="playbookColumns" :data="playbooks" :loading="loading">
          <template #tags="{ row }">
            <AppleBadge v-for="tag in row.tags" :key="tag" type="info">
              {{ tag }}
            </AppleBadge>
          </template>
          <template #actions="{ row }">
            <div class="action-buttons">
              <AppleButton size="small" @click="viewPlaybook(row)">
                <i class="fas fa-eye"></i>
              </AppleButton>
              <AppleButton size="small" @click="editPlaybook(row)">
                <i class="fas fa-edit"></i>
              </AppleButton>
              <AppleButton size="small" type="success" @click="executePlaybook(row)">
                <i class="fas fa-play"></i>
              </AppleButton>
              <AppleButton size="small" type="danger" @click="deletePlaybook(row)">
                <i class="fas fa-trash"></i>
              </AppleButton>
            </div>
          </template>
        </AppleTable>
      </div>

      <!-- Inventory列表 -->
      <div v-if="activeTab === 'inventories'" class="tab-content">
        <div class="section-header">
          <h2>Inventory 列表</h2>
          <div class="actions">
            <AppleButton @click="syncFromCMDB">
              <i class="fas fa-sync"></i> 从CMDB同步
            </AppleButton>
            <AppleButton @click="showInventoryModal = true" type="primary">
              <i class="fas fa-plus"></i> 创建 Inventory
            </AppleButton>
          </div>
        </div>
        <AppleTable :columns="inventoryColumns" :data="inventories" :loading="loading">
          <template #source="{ row }">
            <AppleBadge :type="row.source === 'cmdb' ? 'success' : 'default'">
              {{ row.source || 'manual' }}
            </AppleBadge>
          </template>
          <template #actions="{ row }">
            <div class="action-buttons">
              <AppleButton size="small" @click="viewInventory(row)">
                <i class="fas fa-eye"></i>
              </AppleButton>
              <AppleButton size="small" @click="editInventory(row)">
                <i class="fas fa-edit"></i>
              </AppleButton>
              <AppleButton size="small" type="danger" @click="deleteInventory(row)">
                <i class="fas fa-trash"></i>
              </AppleButton>
            </div>
          </template>
        </AppleTable>
      </div>

      <!-- 执行记录 -->
      <div v-if="activeTab === 'executions'" class="tab-content">
        <div class="section-header">
          <h2>执行记录</h2>
          <AppleButton @click="loadExecutions">
            <i class="fas fa-sync"></i> 刷新
          </AppleButton>
        </div>
        <AppleTable :columns="executionColumns" :data="executions" :loading="loading">
          <template #status="{ row }">
            <AppleBadge :type="getStatusType(row.status)">
              {{ getStatusText(row.status) }}
            </AppleBadge>
          </template>
          <template #check="{ row }">
            <AppleBadge v-if="row.check" type="warning">Dry Run</AppleBadge>
          </template>
          <template #actions="{ row }">
            <div class="action-buttons">
              <AppleButton size="small" @click="viewExecution(row)">
                <i class="fas fa-eye"></i>
              </AppleButton>
              <AppleButton v-if="row.status === 0" size="small" type="danger" @click="cancelExecution(row)">
                <i class="fas fa-stop"></i>
              </AppleButton>
            </div>
          </template>
        </AppleTable>
      </div>
    </AppleCard>

    <!-- 创建/编辑 Playbook 模态框 -->
    <AppleModal v-model="showPlaybookModal" :title="editingPlaybook ? '编辑 Playbook' : '创建 Playbook'" width="900px">
      <div class="playbook-form">
        <div class="form-group">
          <label>名称</label>
          <AppleInput v-model="playbookForm.name" placeholder="Playbook名称" />
        </div>
        <div class="form-group">
          <label>描述</label>
          <AppleInput v-model="playbookForm.description" placeholder="描述" />
        </div>
        <div class="form-group">
          <label>标签 (逗号分隔)</label>
          <AppleInput v-model="playbookForm.tags" placeholder="例如: deploy,nginx" />
        </div>
        <div class="form-group">
          <label>Playbook 内容</label>
          <textarea 
            v-model="playbookForm.content" 
            class="yaml-textarea"
            placeholder="输入 YAML 格式的 Playbook"
            rows="15"
          ></textarea>
        </div>
        <div class="form-actions">
          <AppleButton @click="validatePlaybook" :loading="validating">
            <i class="fas fa-check-circle"></i> 验证语法
          </AppleButton>
          <AppleButton type="primary" @click="savePlaybook" :loading="saving">
            <i class="fas fa-save"></i> 保存
          </AppleButton>
        </div>
      </div>
    </AppleModal>

    <!-- 创建/编辑 Inventory 模态框 -->
    <AppleModal v-model="showInventoryModal" :title="editingInventory ? '编辑 Inventory' : '创建 Inventory'" width="800px">
      <div class="inventory-form">
        <div class="form-group">
          <label>名称</label>
          <AppleInput v-model="inventoryForm.name" placeholder="Inventory名称" />
        </div>
        <div class="form-group">
          <label>描述</label>
          <AppleInput v-model="inventoryForm.description" placeholder="描述" />
        </div>
        <div class="form-group">
          <label>Inventory 内容 (INI格式)</label>
          <textarea 
            v-model="inventoryForm.content" 
            class="yaml-textarea"
            placeholder="输入 INI 格式的 Inventory"
            rows="12"
          ></textarea>
        </div>
        <div class="form-actions">
          <AppleButton type="primary" @click="saveInventory" :loading="saving">
            <i class="fas fa-save"></i> 保存
          </AppleButton>
        </div>
      </div>
    </AppleModal>

    <!-- 执行 Playbook 模态框 -->
    <AppleModal v-model="showExecuteModal" title="执行 Playbook" width="700px">
      <div v-if="selectedPlaybook" class="execute-form">
        <div class="form-group">
          <label>Playbook: {{ selectedPlaybook.name }}</label>
        </div>
        <div class="form-group">
          <label>Inventory</label>
          <AppleSelect v-model="executeForm.inventory_id" placeholder="选择 Inventory">
            <option value="">直接指定主机</option>
            <option v-for="inv in inventories" :key="inv.id" :value="inv.id">
              {{ inv.name }}
            </option>
          </AppleSelect>
        </div>
        <div v-if="!executeForm.inventory_id" class="form-group">
          <label>主机 (逗号分隔)</label>
          <AppleInput v-model="executeForm.hosts" placeholder="例如: 192.168.1.10,192.168.1.11" />
        </div>
        <div class="form-group">
          <label>Tags (可选)</label>
          <AppleInput v-model="executeForm.tags" placeholder="只执行指定标签的任务" />
        </div>
        <div class="form-group">
          <label>Limit (可选)</label>
          <AppleInput v-model="executeForm.limit" placeholder="限制执行的主机" />
        </div>
        <div class="form-group">
          <label>
            <input type="checkbox" v-model="executeForm.check" />
            Dry Run (检查模式，不实际执行)
          </label>
        </div>
        <div class="form-actions">
          <AppleButton type="primary" @click="confirmExecute" :loading="executing">
            <i class="fas fa-play"></i> 执行
          </AppleButton>
        </div>
      </div>
    </AppleModal>

    <!-- 查看执行详情模态框 -->
    <AppleModal v-model="showExecutionModal" title="执行详情" width="900px">
      <div v-if="selectedExecution" class="execution-detail">
        <div class="detail-section">
          <h3>基本信息</h3>
          <div class="detail-grid">
            <div class="detail-item">
              <span class="label">Playbook:</span>
              <span>{{ selectedExecution.playbook_name }}</span>
            </div>
            <div class="detail-item">
              <span class="label">状态:</span>
              <AppleBadge :type="getStatusType(selectedExecution.status)">
                {{ getStatusText(selectedExecution.status) }}
              </AppleBadge>
            </div>
            <div class="detail-item">
              <span class="label">执行人:</span>
              <span>{{ selectedExecution.executor }}</span>
            </div>
            <div class="detail-item">
              <span class="label">耗时:</span>
              <span>{{ selectedExecution.duration }}秒</span>
            </div>
          </div>
        </div>
        <div class="detail-section">
          <h3>执行输出</h3>
          <pre class="output-content">{{ selectedExecution.output || '暂无输出' }}</pre>
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

const activeTab = ref('playbooks')
const loading = ref(false)
const playbooks = ref([])
const inventories = ref([])
const executions = ref([])
const showPlaybookModal = ref(false)
const showInventoryModal = ref(false)
const showExecuteModal = ref(false)
const showExecutionModal = ref(false)
const editingPlaybook = ref(null)
const editingInventory = ref(null)
const selectedPlaybook = ref(null)
const selectedExecution = ref(null)
const validating = ref(false)
const saving = ref(false)
const executing = ref(false)

const playbookForm = ref({
  name: '',
  description: '',
  tags: '',
  content: ''
})

const inventoryForm = ref({
  name: '',
  description: '',
  content: ''
})

const executeForm = ref({
  inventory_id: '',
  hosts: '',
  tags: '',
  limit: '',
  check: false
})

const tabs = [
  { key: 'playbooks', label: 'Playbooks', icon: 'fas fa-file-code' },
  { key: 'inventories', label: 'Inventories', icon: 'fas fa-list' },
  { key: 'executions', label: '执行记录', icon: 'fas fa-history' }
]

const playbookColumns = [
  { key: 'name', label: '名称', width: '200px' },
  { key: 'description', label: '描述', width: '300px' },
  { key: 'tags', label: '标签', width: '200px', slot: true },
  { key: 'created_at', label: '创建时间', width: '180px' },
  { key: 'actions', label: '操作', width: '220px', slot: true }
]

const inventoryColumns = [
  { key: 'name', label: '名称', width: '200px' },
  { key: 'description', label: '描述', width: '300px' },
  { key: 'source', label: '来源', width: '100px', slot: true },
  { key: 'created_at', label: '创建时间', width: '180px' },
  { key: 'actions', label: '操作', width: '180px', slot: true }
]

const executionColumns = [
  { key: 'playbook_name', label: 'Playbook', width: '200px' },
  { key: 'status', label: '状态', width: '100px', slot: true },
  { key: 'check', label: '模式', width: '100px', slot: true },
  { key: 'executor', label: '执行人', width: '120px' },
  { key: 'duration', label: '耗时(秒)', width: '100px' },
  { key: 'started_at', label: '开始时间', width: '180px' },
  { key: 'actions', label: '操作', width: '120px', slot: true }
]

const getStatusType = (status) => {
  const types = { 0: 'info', 1: 'success', 2: 'danger', 3: 'warning' }
  return types[status] || 'default'
}

const getStatusText = (status) => {
  const texts = { 0: '执行中', 1: '成功', 2: '失败', 3: '已取消' }
  return texts[status] || '未知'
}

const loadPlaybooks = async () => {
  loading.value = true
  try {
    const res = await api.get('/ansible/playbooks')
    if (res.data.code === 0) {
      playbooks.value = res.data.data || []
    }
  } catch (error) {
    console.error('加载Playbook失败:', error)
  } finally {
    loading.value = false
  }
}

const loadInventories = async () => {
  loading.value = true
  try {
    const res = await api.get('/ansible/inventories')
    if (res.data.code === 0) {
      inventories.value = res.data.data || []
    }
  } catch (error) {
    console.error('加载Inventory失败:', error)
  } finally {
    loading.value = false
  }
}

const loadExecutions = async () => {
  loading.value = true
  try {
    const res = await api.get('/ansible/executions')
    if (res.data.code === 0) {
      executions.value = res.data.data || []
    }
  } catch (error) {
    console.error('加载执行记录失败:', error)
  } finally {
    loading.value = false
  }
}

const viewPlaybook = async (playbook) => {
  try {
    const res = await api.get(`/ansible/playbooks/${playbook.id}`)
    if (res.data.code === 0) {
      playbookForm.value = { ...res.data.data, tags: res.data.data.tags?.join(',') || '' }
      editingPlaybook.value = playbook
      showPlaybookModal.value = true
    }
  } catch (error) {
    alert('加载失败')
  }
}

const editPlaybook = (playbook) => {
  viewPlaybook(playbook)
}

const deletePlaybook = async (playbook) => {
  if (!confirm(`确认删除 ${playbook.name}?`)) return
  
  try {
    const res = await api.delete(`/ansible/playbooks/${playbook.id}`)
    if (res.data.code === 0) {
      alert('删除成功')
      loadPlaybooks()
    }
  } catch (error) {
    alert('删除失败')
  }
}

const validatePlaybook = async () => {
  if (!editingPlaybook.value) {
    alert('请先保存Playbook')
    return
  }
  validating.value = true
  try {
    const res = await api.post(`/ansible/playbooks/${editingPlaybook.value.id}/validate`)
    if (res.data.code === 0) {
      if (res.data.data.valid) {
        alert('语法验证通过')
      } else {
        alert('语法错误:\n' + res.data.data.output)
      }
    }
  } catch (error) {
    alert('验证失败')
  } finally {
    validating.value = false
  }
}

const savePlaybook = async () => {
  if (!playbookForm.value.name || !playbookForm.value.content) {
    alert('请填写完整信息')
    return
  }
  
  saving.value = true
  try {
    const data = {
      ...playbookForm.value,
      tags: playbookForm.value.tags ? playbookForm.value.tags.split(',').map(t => t.trim()) : []
    }
    
    const res = editingPlaybook.value
      ? await api.put(`/ansible/playbooks/${editingPlaybook.value.id}`, data)
      : await api.post('/ansible/playbooks', data)
    
    if (res.data.code === 0) {
      alert('保存成功')
      showPlaybookModal.value = false
      editingPlaybook.value = null
      playbookForm.value = { name: '', description: '', tags: '', content: '' }
      loadPlaybooks()
    }
  } catch (error) {
    alert('保存失败')
  } finally {
    saving.value = false
  }
}

const viewInventory = async (inventory) => {
  try {
    const res = await api.get(`/ansible/inventories/${inventory.id}`)
    if (res.data.code === 0) {
      inventoryForm.value = res.data.data
      editingInventory.value = inventory
      showInventoryModal.value = true
    }
  } catch (error) {
    alert('加载失败')
  }
}

const editInventory = (inventory) => {
  viewInventory(inventory)
}

const deleteInventory = async (inventory) => {
  if (!confirm(`确认删除 ${inventory.name}?`)) return
  
  try {
    const res = await api.delete(`/ansible/inventories/${inventory.id}`)
    if (res.data.code === 0) {
      alert('删除成功')
      loadInventories()
    }
  } catch (error) {
    alert('删除失败')
  }
}

const saveInventory = async () => {
  if (!inventoryForm.value.name || !inventoryForm.value.content) {
    alert('请填写完整信息')
    return
  }
  
  saving.value = true
  try {
    const res = editingInventory.value
      ? await api.put(`/ansible/inventories/${editingInventory.value.id}`, inventoryForm.value)
      : await api.post('/ansible/inventories', inventoryForm.value)
    
    if (res.data.code === 0) {
      alert('保存成功')
      showInventoryModal.value = false
      editingInventory.value = null
      inventoryForm.value = { name: '', description: '', content: '' }
      loadInventories()
    }
  } catch (error) {
    alert('保存失败')
  } finally {
    saving.value = false
  }
}

const syncFromCMDB = async () => {
  const name = prompt('请输入Inventory名称:')
  if (!name) return
  
  try {
    const res = await api.post('/ansible/inventories/sync', { name })
    if (res.data.code === 0) {
      alert('同步成功')
      loadInventories()
    }
  } catch (error) {
    alert('同步失败')
  }
}

const executePlaybook = (playbook) => {
  selectedPlaybook.value = playbook
  executeForm.value = { inventory_id: '', hosts: '', tags: '', limit: '', check: false }
  showExecuteModal.value = true
}

const confirmExecute = async () => {
  if (!executeForm.value.inventory_id && !executeForm.value.hosts) {
    alert('请选择Inventory或指定主机')
    return
  }
  
  executing.value = true
  try {
    const res = await api.post(`/ansible/playbooks/${selectedPlaybook.value.id}/execute`, executeForm.value)
    if (res.data.code === 0) {
      alert('执行已启动')
      showExecuteModal.value = false
      activeTab.value = 'executions'
      loadExecutions()
    }
  } catch (error) {
    alert('执行失败')
  } finally {
    executing.value = false
  }
}

const viewExecution = async (execution) => {
  try {
    const res = await api.get(`/ansible/executions/${execution.id}`)
    if (res.data.code === 0) {
      selectedExecution.value = res.data.data
      showExecutionModal.value = true
    }
  } catch (error) {
    alert('加载失败')
  }
}

const cancelExecution = async (execution) => {
  if (!confirm('确认取消执行?')) return
  
  try {
    const res = await api.post(`/ansible/executions/${execution.id}/cancel`)
    if (res.data.code === 0) {
      alert('已取消')
      loadExecutions()
    }
  } catch (error) {
    alert('取消失败')
  }
}

onMounted(() => {
  loadPlaybooks()
  loadInventories()
  loadExecutions()
})
</script>

<style scoped>
.ansible-page {
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

.tabs-section {
  padding: 0;
}

.tabs {
  display: flex;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  padding: 0 24px;
}

.tab {
  padding: 16px 24px;
  color: rgba(255, 255, 255, 0.6);
  cursor: pointer;
  border-bottom: 2px solid transparent;
  transition: all 0.3s;
  display: flex;
  align-items: center;
  gap: 8px;
}

.tab:hover {
  color: #fff;
}

.tab.active {
  color: #007AFF;
  border-bottom-color: #007AFF;
}

.tab-content {
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

.actions {
  display: flex;
  gap: 12px;
}

.action-buttons {
  display: flex;
  gap: 8px;
}

.playbook-form,
.inventory-form,
.execute-form {
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

.yaml-textarea {
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
  justify-content: flex-end;
}

.execution-detail {
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

.output-content {
  padding: 16px;
  background: rgba(0, 0, 0, 0.3);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 8px;
  color: #fff;
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 12px;
  max-height: 500px;
  overflow-y: auto;
  white-space: pre-wrap;
  word-break: break-all;
  margin: 0;
}
</style>
