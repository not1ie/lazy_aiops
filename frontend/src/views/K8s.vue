<template>
  <div class="k8s-page">
    <div class="page-header">
      <h1>Kubernetes 管理</h1>
      <p>集群管理、工作负载、Pod监控</p>
    </div>

    <!-- 集群选择 -->
    <AppleCard class="cluster-selector">
      <div class="selector-content">
        <label>当前集群</label>
        <AppleSelect v-model="selectedCluster" @change="loadClusterData" placeholder="选择集群">
          <option value="">请选择集群</option>
          <option v-for="cluster in clusters" :key="cluster.id" :value="cluster.id">
            {{ cluster.name }} - {{ cluster.version || 'Unknown' }}
          </option>
        </AppleSelect>
        <AppleButton @click="showClusterModal = true">
          <i class="fas fa-plus"></i> 添加集群
        </AppleButton>
      </div>
    </AppleCard>

    <!-- 统计卡片 -->
    <div v-if="selectedCluster" class="stats-grid">
      <AppleCard class="stat-card">
        <div class="stat-content">
          <i class="fas fa-server stat-icon"></i>
          <div class="stat-info">
            <div class="stat-value">{{ stats.nodes || 0 }}</div>
            <div class="stat-label">节点数</div>
          </div>
        </div>
      </AppleCard>
      <AppleCard class="stat-card">
        <div class="stat-content">
          <i class="fas fa-cube stat-icon"></i>
          <div class="stat-info">
            <div class="stat-value">{{ stats.pods || 0 }}</div>
            <div class="stat-label">Pod数</div>
          </div>
        </div>
      </AppleCard>
      <AppleCard class="stat-card">
        <div class="stat-content">
          <i class="fas fa-layer-group stat-icon"></i>
          <div class="stat-info">
            <div class="stat-value">{{ stats.deployments || 0 }}</div>
            <div class="stat-label">Deployment</div>
          </div>
        </div>
      </AppleCard>
      <AppleCard class="stat-card">
        <div class="stat-content">
          <i class="fas fa-folder stat-icon"></i>
          <div class="stat-info">
            <div class="stat-value">{{ stats.namespaces || 0 }}</div>
            <div class="stat-label">命名空间</div>
          </div>
        </div>
      </AppleCard>
    </div>

    <!-- 标签页 -->
    <AppleCard v-if="selectedCluster" class="tabs-section">
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

      <!-- 节点列表 -->
      <div v-if="activeTab === 'nodes'" class="tab-content">
        <AppleTable :columns="nodeColumns" :data="nodes" :loading="loading">
          <template #status="{ row }">
            <AppleBadge :type="row.status === 'Ready' ? 'success' : 'danger'">
              {{ row.status }}
            </AppleBadge>
          </template>
          <template #roles="{ row }">
            <AppleBadge v-for="role in row.roles" :key="role" type="info">
              {{ role }}
            </AppleBadge>
          </template>
        </AppleTable>
      </div>

      <!-- 命名空间列表 -->
      <div v-if="activeTab === 'namespaces'" class="tab-content">
        <AppleTable :columns="namespaceColumns" :data="namespaces" :loading="loading">
          <template #status="{ row }">
            <AppleBadge :type="row.status === 'Active' ? 'success' : 'warning'">
              {{ row.status }}
            </AppleBadge>
          </template>
        </AppleTable>
      </div>

      <!-- Deployment列表 -->
      <div v-if="activeTab === 'deployments'" class="tab-content">
        <div class="filters">
          <AppleSelect v-model="selectedNamespace" @change="loadDeployments" placeholder="命名空间">
            <option value="">所有命名空间</option>
            <option v-for="ns in namespaces" :key="ns.name" :value="ns.name">
              {{ ns.name }}
            </option>
          </AppleSelect>
          <AppleButton @click="loadDeployments">
            <i class="fas fa-sync"></i> 刷新
          </AppleButton>
        </div>
        <AppleTable :columns="deploymentColumns" :data="deployments" :loading="loading">
          <template #replicas="{ row }">
            <span>{{ row.ready }} / {{ row.replicas }}</span>
          </template>
          <template #actions="{ row }">
            <div class="action-buttons">
              <AppleButton size="small" @click="scaleDeployment(row)">
                <i class="fas fa-expand-arrows-alt"></i>
              </AppleButton>
              <AppleButton size="small" type="warning" @click="restartDeployment(row)">
                <i class="fas fa-redo"></i>
              </AppleButton>
              <AppleButton size="small" @click="viewPods(row)">
                <i class="fas fa-list"></i>
              </AppleButton>
            </div>
          </template>
        </AppleTable>
      </div>

      <!-- Pod列表 -->
      <div v-if="activeTab === 'pods'" class="tab-content">
        <div class="filters">
          <AppleSelect v-model="selectedNamespace" @change="loadPods" placeholder="命名空间">
            <option value="">所有命名空间</option>
            <option v-for="ns in namespaces" :key="ns.name" :value="ns.name">
              {{ ns.name }}
            </option>
          </AppleSelect>
          <AppleButton @click="loadPods">
            <i class="fas fa-sync"></i> 刷新
          </AppleButton>
        </div>
        <AppleTable :columns="podColumns" :data="pods" :loading="loading">
          <template #status="{ row }">
            <AppleBadge :type="getPodStatusType(row.status)">
              {{ row.status }}
            </AppleBadge>
          </template>
          <template #actions="{ row }">
            <div class="action-buttons">
              <AppleButton size="small" @click="viewLogs(row)">
                <i class="fas fa-file-alt"></i>
              </AppleButton>
              <AppleButton size="small" type="danger" @click="deletePod(row)">
                <i class="fas fa-trash"></i>
              </AppleButton>
            </div>
          </template>
        </AppleTable>
      </div>
    </AppleCard>

    <!-- 添加集群模态框 -->
    <AppleModal v-model="showClusterModal" title="添加 Kubernetes 集群" width="700px">
      <div class="cluster-form">
        <div class="form-group">
          <label>集群名称</label>
          <AppleInput v-model="newCluster.name" placeholder="例如: production-cluster" />
        </div>
        <div class="form-group">
          <label>描述</label>
          <AppleInput v-model="newCluster.description" placeholder="集群描述" />
        </div>
        <div class="form-group">
          <label>KubeConfig</label>
          <textarea 
            v-model="newCluster.kube_config" 
            class="config-textarea"
            placeholder="粘贴 kubeconfig 内容"
            rows="12"
          ></textarea>
        </div>
        <div class="form-actions">
          <AppleButton @click="testConnection" :loading="testing">
            <i class="fas fa-plug"></i> 测试连接
          </AppleButton>
          <AppleButton type="primary" @click="createCluster" :loading="creating">
            <i class="fas fa-check"></i> 添加集群
          </AppleButton>
        </div>
      </div>
    </AppleModal>

    <!-- 扩缩容模态框 -->
    <AppleModal v-model="showScaleModal" title="扩缩容" width="500px">
      <div v-if="selectedDeployment" class="scale-form">
        <div class="form-group">
          <label>Deployment: {{ selectedDeployment.name }}</label>
          <label>当前副本数: {{ selectedDeployment.replicas }}</label>
        </div>
        <div class="form-group">
          <label>目标副本数</label>
          <AppleInput v-model.number="scaleReplicas" type="number" min="0" />
        </div>
        <div class="form-actions">
          <AppleButton type="primary" @click="confirmScale" :loading="scaling">
            <i class="fas fa-check"></i> 确认
          </AppleButton>
        </div>
      </div>
    </AppleModal>

    <!-- 日志查看模态框 -->
    <AppleModal v-model="showLogsModal" title="Pod 日志" width="900px">
      <div v-if="selectedPod" class="logs-viewer">
        <div class="logs-header">
          <span>Pod: {{ selectedPod.name }}</span>
          <AppleSelect v-model="selectedContainer" @change="loadLogs" placeholder="容器">
            <option v-for="container in selectedPod.containers" :key="container.name" :value="container.name">
              {{ container.name }}
            </option>
          </AppleSelect>
          <AppleButton size="small" @click="loadLogs">
            <i class="fas fa-sync"></i> 刷新
          </AppleButton>
        </div>
        <pre class="logs-content">{{ logs }}</pre>
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

const clusters = ref([])
const selectedCluster = ref('')
const stats = ref({})
const activeTab = ref('nodes')
const loading = ref(false)
const nodes = ref([])
const namespaces = ref([])
const deployments = ref([])
const pods = ref([])
const selectedNamespace = ref('')
const showClusterModal = ref(false)
const showScaleModal = ref(false)
const showLogsModal = ref(false)
const selectedDeployment = ref(null)
const selectedPod = ref(null)
const selectedContainer = ref('')
const scaleReplicas = ref(1)
const logs = ref('')
const testing = ref(false)
const creating = ref(false)
const scaling = ref(false)

const newCluster = ref({
  name: '',
  description: '',
  kube_config: ''
})

const tabs = [
  { key: 'nodes', label: '节点', icon: 'fas fa-server' },
  { key: 'namespaces', label: '命名空间', icon: 'fas fa-folder' },
  { key: 'deployments', label: 'Deployments', icon: 'fas fa-layer-group' },
  { key: 'pods', label: 'Pods', icon: 'fas fa-cube' }
]

const nodeColumns = [
  { key: 'name', label: '节点名称', width: '200px' },
  { key: 'status', label: '状态', width: '100px', slot: true },
  { key: 'roles', label: '角色', width: '150px', slot: true },
  { key: 'internal_ip', label: 'IP地址', width: '150px' },
  { key: 'kubelet_ver', label: 'Kubelet版本', width: '120px' },
  { key: 'cpu', label: 'CPU', width: '100px' },
  { key: 'memory', label: '内存', width: '120px' }
]

const namespaceColumns = [
  { key: 'name', label: '命名空间', width: '200px' },
  { key: 'status', label: '状态', width: '100px', slot: true },
  { key: 'created_at', label: '创建时间', width: '180px' }
]

const deploymentColumns = [
  { key: 'namespace', label: '命名空间', width: '150px' },
  { key: 'name', label: '名称', width: '200px' },
  { key: 'replicas', label: '副本数', width: '100px', slot: true },
  { key: 'images', label: '镜像', width: '300px' },
  { key: 'created_at', label: '创建时间', width: '180px' },
  { key: 'actions', label: '操作', width: '180px', slot: true }
]

const podColumns = [
  { key: 'namespace', label: '命名空间', width: '150px' },
  { key: 'name', label: 'Pod名称', width: '250px' },
  { key: 'status', label: '状态', width: '100px', slot: true },
  { key: 'node', label: '节点', width: '150px' },
  { key: 'ip', label: 'IP', width: '120px' },
  { key: 'restarts', label: '重启次数', width: '100px' },
  { key: 'created_at', label: '创建时间', width: '180px' },
  { key: 'actions', label: '操作', width: '120px', slot: true }
]

const getPodStatusType = (status) => {
  if (status === 'Running') return 'success'
  if (status === 'Pending') return 'warning'
  if (status === 'Failed' || status === 'Error') return 'danger'
  return 'default'
}

const loadClusters = async () => {
  try {
    const res = await api.get('/k8s/clusters')
    if (res.data.code === 0) {
      clusters.value = res.data.data || []
    }
  } catch (error) {
    console.error('加载集群失败:', error)
  }
}

const loadClusterData = async () => {
  if (!selectedCluster.value) return
  
  loading.value = true
  try {
    await Promise.all([
      loadNodes(),
      loadNamespaces()
    ])
    updateStats()
  } finally {
    loading.value = false
  }
}

const loadNodes = async () => {
  try {
    const res = await api.get(`/k8s/clusters/${selectedCluster.value}/nodes`)
    if (res.data.code === 0) {
      nodes.value = res.data.data || []
    }
  } catch (error) {
    console.error('加载节点失败:', error)
  }
}

const loadNamespaces = async () => {
  try {
    const res = await api.get(`/k8s/clusters/${selectedCluster.value}/namespaces`)
    if (res.data.code === 0) {
      namespaces.value = res.data.data || []
    }
  } catch (error) {
    console.error('加载命名空间失败:', error)
  }
}

const loadDeployments = async () => {
  loading.value = true
  try {
    const params = selectedNamespace.value ? { namespace: selectedNamespace.value } : {}
    const res = await api.get(`/k8s/clusters/${selectedCluster.value}/workloads`, { params })
    if (res.data.code === 0) {
      deployments.value = res.data.data || []
    }
  } catch (error) {
    console.error('加载Deployment失败:', error)
  } finally {
    loading.value = false
  }
}

const loadPods = async () => {
  if (!selectedNamespace.value) {
    alert('请选择命名空间')
    return
  }
  loading.value = true
  try {
    const res = await api.get(`/k8s/clusters/${selectedCluster.value}/namespaces/${selectedNamespace.value}/pods`)
    if (res.data.code === 0) {
      pods.value = res.data.data || []
    }
  } catch (error) {
    console.error('加载Pod失败:', error)
  } finally {
    loading.value = false
  }
}

const updateStats = () => {
  stats.value = {
    nodes: nodes.value.length,
    namespaces: namespaces.value.length,
    deployments: deployments.value.length,
    pods: pods.value.length
  }
}

const testConnection = async () => {
  if (!newCluster.value.kube_config) {
    alert('请输入 KubeConfig')
    return
  }
  testing.value = true
  try {
    // 先创建临时集群测试
    const res = await api.post('/k8s/clusters', newCluster.value)
    if (res.data.code === 0) {
      const clusterId = res.data.data.id
      const testRes = await api.post(`/k8s/clusters/${clusterId}/test`)
      if (testRes.data.code === 0) {
        alert('连接成功！版本: ' + testRes.data.data.version)
      }
    }
  } catch (error) {
    alert('连接失败: ' + (error.response?.data?.message || error.message))
  } finally {
    testing.value = false
  }
}

const createCluster = async () => {
  if (!newCluster.value.name || !newCluster.value.kube_config) {
    alert('请填写完整信息')
    return
  }
  creating.value = true
  try {
    const res = await api.post('/k8s/clusters', newCluster.value)
    if (res.data.code === 0) {
      alert('集群添加成功')
      showClusterModal.value = false
      newCluster.value = { name: '', description: '', kube_config: '' }
      loadClusters()
    }
  } catch (error) {
    alert('添加失败: ' + (error.response?.data?.message || error.message))
  } finally {
    creating.value = false
  }
}

const scaleDeployment = (deployment) => {
  selectedDeployment.value = deployment
  scaleReplicas.value = deployment.replicas
  showScaleModal.value = true
}

const confirmScale = async () => {
  scaling.value = true
  try {
    const res = await api.post(
      `/k8s/clusters/${selectedCluster.value}/namespaces/${selectedDeployment.value.namespace}/deployments/${selectedDeployment.value.name}/scale`,
      { replicas: scaleReplicas.value }
    )
    if (res.data.code === 0) {
      alert('扩缩容成功')
      showScaleModal.value = false
      loadDeployments()
    }
  } catch (error) {
    alert('操作失败: ' + (error.response?.data?.message || error.message))
  } finally {
    scaling.value = false
  }
}

const restartDeployment = async (deployment) => {
  if (!confirm(`确认重启 ${deployment.name}?`)) return
  
  try {
    const res = await api.post(
      `/k8s/clusters/${selectedCluster.value}/namespaces/${deployment.namespace}/deployments/${deployment.name}/restart`
    )
    if (res.data.code === 0) {
      alert('重启成功')
      loadDeployments()
    }
  } catch (error) {
    alert('操作失败: ' + (error.response?.data?.message || error.message))
  }
}

const viewPods = async (deployment) => {
  selectedNamespace.value = deployment.namespace
  activeTab.value = 'pods'
  await loadPods()
}

const viewLogs = (pod) => {
  selectedPod.value = pod
  if (pod.containers && pod.containers.length > 0) {
    selectedContainer.value = pod.containers[0].name
  }
  showLogsModal.value = true
  loadLogs()
}

const loadLogs = async () => {
  if (!selectedPod.value || !selectedContainer.value) return
  
  try {
    const res = await api.get(
      `/k8s/clusters/${selectedCluster.value}/namespaces/${selectedPod.value.namespace}/pods/${selectedPod.value.name}/logs`,
      { params: { container: selectedContainer.value, tail: 500 } }
    )
    if (res.data.code === 0) {
      logs.value = res.data.data || '暂无日志'
    }
  } catch (error) {
    logs.value = '加载日志失败: ' + (error.response?.data?.message || error.message)
  }
}

const deletePod = async (pod) => {
  if (!confirm(`确认删除 Pod ${pod.name}?`)) return
  
  try {
    const res = await api.delete(
      `/k8s/clusters/${selectedCluster.value}/namespaces/${pod.namespace}/pods/${pod.name}`
    )
    if (res.data.code === 0) {
      alert('删除成功')
      loadPods()
    }
  } catch (error) {
    alert('删除失败: ' + (error.response?.data?.message || error.message))
  }
}

onMounted(() => {
  loadClusters()
})
</script>

<style scoped>
.k8s-page {
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

.cluster-selector {
  padding: 20px;
  margin-bottom: 24px;
}

.selector-content {
  display: flex;
  align-items: center;
  gap: 16px;
}

.selector-content label {
  color: #fff;
  font-weight: 500;
  min-width: 80px;
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

.filters {
  display: flex;
  gap: 12px;
  margin-bottom: 20px;
}

.action-buttons {
  display: flex;
  gap: 8px;
}

.cluster-form,
.scale-form {
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

.config-textarea {
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

.logs-viewer {
  padding: 20px;
}

.logs-header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 16px;
  padding-bottom: 16px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.logs-header span {
  color: #fff;
  font-weight: 500;
}

.logs-content {
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
