<template>
  <div class="executor-page">
    <div class="page-header">
      <h1>批量执行</h1>
      <p>SSH批量命令执行、脚本分发</p>
    </div>

    <!-- 快速执行 -->
    <AppleCard class="execute-section">
      <div class="section-header">
        <h2>批量执行命令</h2>
      </div>

      <div class="execute-form">
        <div class="form-row">
          <div class="form-group flex-1">
            <label>执行名称</label>
            <AppleInput v-model="executeForm.name" placeholder="例如: 检查磁盘空间" />
          </div>
          <div class="form-group" style="width: 200px">
            <label>执行类型</label>
            <AppleSelect v-model="executeForm.type">
              <option value="shell">Shell命令</option>
              <option value="script">Shell脚本</option>
            </AppleSelect>
          </div>
        </div>

        <div class="form-group">
          <label>目标主机</label>
          <div class="host-selector">
            <AppleButton @click="showHostModal = true">
              <i class="fas fa-server"></i> 选择主机 (已选 {{ selectedHosts.length }})
            </AppleButton>
            <div v-if="selectedHosts.length" class="selected-hosts">
              <AppleBadge 
                v-for="host in selectedHosts" 
                :key="host.id"
                type="info"
                @click="removeHost(host)"
              >
                {{ host.name }} ({{ host.ip }})
                <i class="fas fa-times"></i>
              </AppleBadge>
            </div>
          </div>
        </div>

        <div class="form-group">
          <label>{{ executeForm.type === 'shell' ? '命令内容' : '脚本内容' }}</label>
          <textarea 
            v-model="executeForm.content" 
            class="command-textarea"
            :placeholder="executeForm.type === 'shell' ? '例如: df -h' : '输入Shell脚本内容'"
            rows="8"
          ></textarea>
        </div>

        <div class="form-row">
          <div class="form-group" style="width: 200px">
            <label>超时时间(秒)</label>
            <AppleInput v-model.number="executeForm.timeout" type="number" min="10" />
          </div>
          <div class="form-group" style="width: 200px">
            <label>并发数</label>
            <AppleInput v-model.number="executeForm.concurrency" type="number" min="1" max="50" />
          </div>
        </div>

        <div class="form-actions">
          <AppleButton @click="loadTemplates">
            <i class="fas fa-list"></i> 使用模板
          </AppleButton>
          <AppleButton type="primary" @click="executeCommand" :loading="executing">
            <i class="fas fa-play"></i> 立即执行
          </AppleButton>
        </div>
      </div>
    </AppleCard>

    <!-- 执行记录 -->
    <AppleCard class="history-section">
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
        <template #progress="{ row }">
          <div class="progress-bar">
            <div class="progress-fill" :style="{ width: row.progress + '%' }"></div>
            <span class="progress-text">{{ row.progress }}%</span>
          </div>
        </template>
        <template #result="{ row }">
          <span class="result-text">
            成功: {{ row.success_count }} / 失败: {{ row.failed_count }}
          </span>
        </template>
        <template #actions="{ row }">
          <div class="action-buttons">
            <AppleButton size="small" @click="viewResults(row)">
              <i class="fas fa-list"></i> 查看结果
            </AppleButton>
            <AppleButton v-if="row.status === 0" size="small" type="danger" @click="cancelExecution(row)">
              <i class="fas fa-stop"></i>
            </AppleButton>
          </div>
        </template>
      </AppleTable>
    </AppleCard>

    <!-- 选择主机模态框 -->
    <AppleModal v-model="showHostModal" title="选择目标主机" width="800px">
      <div class="host-selection">
        <div class="selection-header">
          <AppleInput v-model="hostSearch" placeholder="搜索主机名或IP">
            <template #prefix>
              <i class="fas fa-search"></i>
            </template>
          </AppleInput>
          <AppleButton @click="selectAll">
            <i class="fas fa-check-double"></i> 全选
          </AppleButton>
          <AppleButton @click="clearSelection">
            <i class="fas fa-times"></i> 清空
          </AppleButton>
        </div>

        <div class="host-list">
          <div 
            v-for="host in filteredHosts" 
            :key="host.id"
            :class="['host-item', { selected: isSelected(host) }]"
            @click="toggleHost(host)"
          >
            <i :class="['fas', isSelected(host) ? 'fa-check-square' : 'fa-square']"></i>
            <div class="host-info">
              <div class="host-name">{{ host.name }}</div>
              <div class="host-ip">{{ host.ip }}</div>
            </div>
            <AppleBadge :type="host.status === 1 ? 'success' : 'danger'">
              {{ host.status === 1 ? '在线' : '离线' }}
            </AppleBadge>
          </div>
        </div>

        <div class="selection-footer">
          <span>已选择 {{ selectedHosts.length }} 台主机</span>
          <AppleButton type="primary" @click="confirmSelection">
            <i class="fas fa-check"></i> 确认
          </AppleButton>
        </div>
      </div>
    </AppleModal>

    <!-- 查看结果模态框 -->
    <AppleModal v-model="showResultsModal" title="执行结果" width="1000px">
      <div v-if="selectedExecution" class="results-viewer">
        <div class="results-header">
          <div class="result-stats">
            <div class="stat-item">
              <span class="label">总数:</span>
              <span class="value">{{ selectedExecution.target_count }}</span>
            </div>
            <div class="stat-item success">
              <span class="label">成功:</span>
              <span class="value">{{ selectedExecution.success_count }}</span>
            </div>
            <div class="stat-item failed">
              <span class="label">失败:</span>
              <span class="value">{{ selectedExecution.failed_count }}</span>
            </div>
            <div class="stat-item">
              <span class="label">耗时:</span>
              <span class="value">{{ selectedExecution.duration }}秒</span>
            </div>
          </div>
          <div class="result-filters">
            <AppleSelect v-model="resultFilter">
              <option value="">全部</option>
              <option value="2">成功</option>
              <option value="3">失败</option>
            </AppleSelect>
          </div>
        </div>

        <div class="results-list">
          <div 
            v-for="result in filteredResults" 
            :key="result.id"
            :class="['result-item', getResultClass(result.status)]"
          >
            <div class="result-header">
              <div class="host-info">
                <i :class="['fas', result.status === 2 ? 'fa-check-circle' : 'fa-times-circle']"></i>
                <span class="host-name">{{ result.host_name }}</span>
                <span class="host-ip">{{ result.host_ip }}</span>
              </div>
              <div class="result-meta">
                <span>退出码: {{ result.exit_code }}</span>
                <span>耗时: {{ result.duration }}秒</span>
              </div>
            </div>
            <div v-if="result.stdout" class="result-output">
              <div class="output-label">标准输出:</div>
              <pre>{{ result.stdout }}</pre>
            </div>
            <div v-if="result.stderr" class="result-output error">
              <div class="output-label">错误输出:</div>
              <pre>{{ result.stderr }}</pre>
            </div>
          </div>
        </div>
      </div>
    </AppleModal>

    <!-- 命令模板模态框 -->
    <AppleModal v-model="showTemplatesModal" title="命令模板" width="700px">
      <div class="templates-list">
        <div 
          v-for="template in templates" 
          :key="template.id"
          class="template-item"
          @click="useTemplate(template)"
        >
          <div class="template-header">
            <h4>{{ template.name }}</h4>
            <AppleBadge :type="template.type === 'shell' ? 'info' : 'success'">
              {{ template.type }}
            </AppleBadge>
          </div>
          <div class="template-description">{{ template.description }}</div>
          <pre class="template-content">{{ template.content }}</pre>
        </div>
      </div>
    </AppleModal>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import AppleCard from '../components/AppleCard.vue'
import AppleButton from '../components/AppleButton.vue'
import AppleTable from '../components/AppleTable.vue'
import AppleBadge from '../components/AppleBadge.vue'
import AppleModal from '../components/AppleModal.vue'
import AppleInput from '../components/AppleInput.vue'
import AppleSelect from '../components/AppleSelect.vue'
import api from '../api'

const loading = ref(false)
const executing = ref(false)
const executions = ref([])
const hosts = ref([])
const selectedHosts = ref([])
const results = ref([])
const templates = ref([])
const showHostModal = ref(false)
const showResultsModal = ref(false)
const showTemplatesModal = ref(false)
const selectedExecution = ref(null)
const hostSearch = ref('')
const resultFilter = ref('')

const executeForm = ref({
  name: '',
  type: 'shell',
  content: '',
  timeout: 300,
  concurrency: 10
})

const executionColumns = [
  { key: 'name', label: '名称', width: '200px' },
  { key: 'type', label: '类型', width: '100px' },
  { key: 'status', label: '状态', width: '100px', slot: true },
  { key: 'progress', label: '进度', width: '150px', slot: true },
  { key: 'result', label: '结果', width: '150px', slot: true },
  { key: 'executor', label: '执行人', width: '120px' },
  { key: 'started_at', label: '开始时间', width: '180px' },
  { key: 'actions', label: '操作', width: '180px', slot: true }
]

const filteredHosts = computed(() => {
  if (!hostSearch.value) return hosts.value
  const search = hostSearch.value.toLowerCase()
  return hosts.value.filter(h => 
    h.name.toLowerCase().includes(search) || 
    h.ip.includes(search)
  )
})

const filteredResults = computed(() => {
  if (!resultFilter.value) return results.value
  return results.value.filter(r => r.status === parseInt(resultFilter.value))
})

const getStatusType = (status) => {
  const types = { 0: 'info', 1: 'success', 2: 'warning', 3: 'danger', 4: 'default' }
  return types[status] || 'default'
}

const getStatusText = (status) => {
  const texts = { 0: '执行中', 1: '全部成功', 2: '部分失败', 3: '全部失败', 4: '已取消' }
  return texts[status] || '未知'
}

const getResultClass = (status) => {
  return status === 2 ? 'success' : 'failed'
}

const isSelected = (host) => {
  return selectedHosts.value.some(h => h.id === host.id)
}

const toggleHost = (host) => {
  const index = selectedHosts.value.findIndex(h => h.id === host.id)
  if (index > -1) {
    selectedHosts.value.splice(index, 1)
  } else {
    selectedHosts.value.push(host)
  }
}

const removeHost = (host) => {
  const index = selectedHosts.value.findIndex(h => h.id === host.id)
  if (index > -1) {
    selectedHosts.value.splice(index, 1)
  }
}

const selectAll = () => {
  selectedHosts.value = [...filteredHosts.value]
}

const clearSelection = () => {
  selectedHosts.value = []
}

const confirmSelection = () => {
  showHostModal.value = false
}

const loadHosts = async () => {
  try {
    const res = await api.get('/cmdb/hosts')
    if (res.data.code === 0) {
      hosts.value = res.data.data || []
    }
  } catch (error) {
    console.error('加载主机失败:', error)
  }
}

const loadExecutions = async () => {
  loading.value = true
  try {
    const res = await api.get('/executor/executions')
    if (res.data.code === 0) {
      executions.value = res.data.data || []
    }
  } catch (error) {
    console.error('加载执行记录失败:', error)
  } finally {
    loading.value = false
  }
}

const loadTemplates = async () => {
  try {
    const res = await api.get('/executor/templates')
    if (res.data.code === 0) {
      templates.value = res.data.data || []
      showTemplatesModal.value = true
    }
  } catch (error) {
    console.error('加载模板失败:', error)
  }
}

const useTemplate = (template) => {
  executeForm.value.type = template.type
  executeForm.value.content = template.content
  executeForm.value.name = template.name
  showTemplatesModal.value = false
}

const executeCommand = async () => {
  if (!executeForm.value.content) {
    alert('请输入命令或脚本内容')
    return
  }
  if (selectedHosts.value.length === 0) {
    alert('请选择目标主机')
    return
  }

  executing.value = true
  try {
    const data = {
      ...executeForm.value,
      host_ids: selectedHosts.value.map(h => h.id)
    }
    const res = await api.post('/executor/execute', data)
    if (res.data.code === 0) {
      alert('执行已启动')
      executeForm.value = { name: '', type: 'shell', content: '', timeout: 300, concurrency: 10 }
      selectedHosts.value = []
      loadExecutions()
    }
  } catch (error) {
    alert('执行失败: ' + (error.response?.data?.message || error.message))
  } finally {
    executing.value = false
  }
}

const viewResults = async (execution) => {
  selectedExecution.value = execution
  resultFilter.value = ''
  
  try {
    const res = await api.get(`/executor/executions/${execution.id}/results`)
    if (res.data.code === 0) {
      results.value = res.data.data || []
      showResultsModal.value = true
    }
  } catch (error) {
    alert('加载结果失败')
  }
}

const cancelExecution = async (execution) => {
  if (!confirm('确认取消执行?')) return
  
  try {
    const res = await api.post(`/executor/executions/${execution.id}/cancel`)
    if (res.data.code === 0) {
      alert('已取消')
      loadExecutions()
    }
  } catch (error) {
    alert('取消失败')
  }
}

onMounted(() => {
  loadHosts()
  loadExecutions()
})
</script>

<style scoped>
.executor-page {
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

.execute-section,
.history-section {
  padding: 24px;
  margin-bottom: 24px;
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

.execute-form {
  max-width: 1000px;
}

.form-row {
  display: flex;
  gap: 16px;
}

.form-group {
  margin-bottom: 20px;
}

.form-group.flex-1 {
  flex: 1;
}

.form-group label {
  display: block;
  margin-bottom: 8px;
  color: #fff;
  font-weight: 500;
}

.host-selector {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.selected-hosts {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.command-textarea {
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

.action-buttons {
  display: flex;
  gap: 8px;
}

.progress-bar {
  position: relative;
  width: 100%;
  height: 24px;
  background: rgba(255, 255, 255, 0.1);
  border-radius: 12px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, #007AFF, #00C7FF);
  transition: width 0.3s;
}

.progress-text {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  color: #fff;
  font-size: 12px;
  font-weight: 600;
}

.result-text {
  color: rgba(255, 255, 255, 0.8);
  font-size: 13px;
}

.host-selection {
  padding: 20px;
}

.selection-header {
  display: flex;
  gap: 12px;
  margin-bottom: 20px;
}

.host-list {
  max-height: 400px;
  overflow-y: auto;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 8px;
  padding: 8px;
}

.host-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;
}

.host-item:hover {
  background: rgba(255, 255, 255, 0.05);
}

.host-item.selected {
  background: rgba(0, 122, 255, 0.2);
}

.host-item i {
  font-size: 18px;
  color: #007AFF;
}

.host-info {
  flex: 1;
}

.host-name {
  color: #fff;
  font-weight: 500;
}

.host-ip {
  color: rgba(255, 255, 255, 0.6);
  font-size: 13px;
}

.selection-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 20px;
  padding-top: 20px;
  border-top: 1px solid rgba(255, 255, 255, 0.1);
}

.selection-footer span {
  color: rgba(255, 255, 255, 0.8);
}

.results-viewer {
  padding: 20px;
}

.results-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding-bottom: 20px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.result-stats {
  display: flex;
  gap: 24px;
}

.stat-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.stat-item .label {
  color: rgba(255, 255, 255, 0.6);
}

.stat-item .value {
  color: #fff;
  font-weight: 600;
  font-size: 18px;
}

.stat-item.success .value {
  color: #34C759;
}

.stat-item.failed .value {
  color: #FF3B30;
}

.results-list {
  max-height: 600px;
  overflow-y: auto;
}

.result-item {
  margin-bottom: 16px;
  padding: 16px;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 8px;
}

.result-item.success {
  border-left: 3px solid #34C759;
}

.result-item.failed {
  border-left: 3px solid #FF3B30;
}

.result-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.result-header .host-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.result-header i {
  font-size: 20px;
}

.result-item.success i {
  color: #34C759;
}

.result-item.failed i {
  color: #FF3B30;
}

.result-meta {
  display: flex;
  gap: 16px;
  color: rgba(255, 255, 255, 0.6);
  font-size: 13px;
}

.result-output {
  margin-top: 12px;
}

.output-label {
  color: rgba(255, 255, 255, 0.6);
  font-size: 13px;
  margin-bottom: 8px;
}

.result-output pre {
  padding: 12px;
  background: rgba(0, 0, 0, 0.3);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 6px;
  color: #fff;
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 12px;
  white-space: pre-wrap;
  word-break: break-all;
  margin: 0;
}

.result-output.error pre {
  color: #FF3B30;
}

.templates-list {
  padding: 20px;
  max-height: 500px;
  overflow-y: auto;
}

.template-item {
  padding: 16px;
  margin-bottom: 16px;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
}

.template-item:hover {
  background: rgba(255, 255, 255, 0.05);
  border-color: #007AFF;
}

.template-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.template-header h4 {
  margin: 0;
  color: #fff;
  font-size: 16px;
}

.template-description {
  color: rgba(255, 255, 255, 0.6);
  font-size: 13px;
  margin-bottom: 12px;
}

.template-content {
  padding: 12px;
  background: rgba(0, 0, 0, 0.3);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 6px;
  color: #fff;
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 12px;
  margin: 0;
}
</style>
