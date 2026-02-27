<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>服务拓扑</h2>
        <p class="page-desc">维护服务节点与依赖关系，支持导入导出。</p>
      </div>
      <div class="page-actions">
        <el-button type="primary" icon="MagicStick" @click="openDiscoverDialog">自动发现</el-button>
        <el-button icon="Upload" @click="openImportDialog">导入</el-button>
        <el-button icon="Download" @click="exportTopology">导出</el-button>
        <el-button icon="Refresh" @click="refreshAll">刷新</el-button>
      </div>
    </div>

    <el-row :gutter="12" class="mb-12">
      <el-col :span="6"><el-card><div class="k">节点数(可见/总数)</div><div class="v">{{ filteredNodes.length }} / {{ nodes.length }}</div></el-card></el-col>
      <el-col :span="6"><el-card><div class="k">依赖边(可见/总数)</div><div class="v">{{ filteredEdges.length }} / {{ edges.length }}</div></el-card></el-col>
      <el-col :span="6"><el-card><div class="k">视图数</div><div class="v">{{ views.length }}</div></el-card></el-col>
      <el-col :span="6"><el-card><div class="k">异常节点</div><div class="v danger">{{ unhealthyCount }}</div></el-card></el-col>
    </el-row>

    <el-row :gutter="12">
      <el-col :md="14" :sm="24">
        <el-card>
          <template #header>
            <div class="section-header">
              <span>节点清单</span>
              <div>
                <el-button size="small" @click="resetNodeFilter">清空筛选</el-button>
                <el-button size="small" type="primary" icon="Plus" @click="openNodeDialog()">新增节点</el-button>
                <el-button size="small" @click="autoLayout">自动布局</el-button>
              </div>
            </div>
          </template>
          <div class="table-filters">
            <el-input v-model="nodeFilter.keyword" clearable placeholder="按名称/命名空间/集群过滤" class="filter-input" />
            <el-select v-model="nodeFilter.source" class="filter-select" placeholder="来源">
              <el-option label="全部来源" value="all" />
              <el-option v-for="item in sourceOptions" :key="item" :label="item" :value="item" />
            </el-select>
            <el-select v-model="nodeFilter.status" class="filter-select" placeholder="状态">
              <el-option label="全部状态" value="all" />
              <el-option label="正常" value="1" />
              <el-option label="告警" value="2" />
              <el-option label="故障" value="3" />
              <el-option label="未知" value="0" />
            </el-select>
          </div>
          <el-table :data="filteredNodes" v-loading="loading" stripe @row-click="selectNode">
            <el-table-column prop="name" label="名称" min-width="160" />
            <el-table-column label="来源" width="120">
              <template #default="{ row }">
                <el-tag size="small" effect="plain">{{ nodeSource(row) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="type" label="类型" width="110" />
            <el-table-column prop="namespace" label="命名空间" width="120" />
            <el-table-column prop="cluster" label="集群" width="120" />
            <el-table-column label="状态" width="90">
              <template #default="{ row }">
                <el-tag :type="statusType(row.status)">{{ statusText(row.status) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="坐标" width="120">
              <template #default="{ row }">({{ row.x || 0 }}, {{ row.y || 0 }})</template>
            </el-table-column>
            <el-table-column label="操作" width="190" fixed="right">
              <template #default="{ row }">
                <el-button size="small" @click.stop="openNodeDialog(row)">编辑</el-button>
                <el-button size="small" type="danger" plain @click.stop="removeNode(row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>

      <el-col :md="10" :sm="24">
        <el-card>
          <template #header>
            <div class="section-header">
              <span>依赖关系</span>
              <el-button size="small" type="primary" icon="Plus" @click="openEdgeDialog">新增关系</el-button>
            </div>
          </template>
          <el-table :data="filteredEdges" stripe size="small">
            <el-table-column label="源服务" min-width="120">
              <template #default="{ row }">{{ row.source_name || nodeName(row.source_id) }}</template>
            </el-table-column>
            <el-table-column label="目标服务" min-width="120">
              <template #default="{ row }">{{ row.target_name || nodeName(row.target_id) }}</template>
            </el-table-column>
            <el-table-column prop="type" label="类型" width="90" />
            <el-table-column prop="port" label="端口" width="80" />
            <el-table-column label="操作" width="90">
              <template #default="{ row }">
                <el-button size="small" type="danger" plain @click="removeEdge(row)">删</el-button>
              </template>
            </el-table-column>
          </el-table>

          <el-divider />
          <div class="sub-title">节点详情</div>
          <el-empty v-if="!selectedNode" description="选择节点查看详情" :image-size="80" />
          <el-descriptions v-else :column="1" border size="small">
            <el-descriptions-item label="名称">{{ selectedNode.name }}</el-descriptions-item>
            <el-descriptions-item label="类型">{{ selectedNode.type || '-' }}</el-descriptions-item>
            <el-descriptions-item label="描述">{{ selectedNode.description || '-' }}</el-descriptions-item>
            <el-descriptions-item label="健康检查">{{ selectedNode.health_url || '-' }}</el-descriptions-item>
            <el-descriptions-item label="端点">{{ selectedNode.endpoints || '-' }}</el-descriptions-item>
            <el-descriptions-item label="元数据">{{ selectedNode.metadata || '-' }}</el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>
    </el-row>

    <el-dialog v-model="nodeDialogVisible" :title="nodeEditing ? '编辑节点' : '新增节点'" width="760px">
      <el-form :model="nodeForm" label-width="92px">
        <el-row :gutter="12">
          <el-col :span="12"><el-form-item label="名称" required><el-input v-model="nodeForm.name" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="类型"><el-select v-model="nodeForm.type" style="width: 100%"><el-option label="service" value="service" /><el-option label="database" value="database" /><el-option label="cache" value="cache" /><el-option label="mq" value="mq" /><el-option label="gateway" value="gateway" /></el-select></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="命名空间"><el-input v-model="nodeForm.namespace" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="集群"><el-input v-model="nodeForm.cluster" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="状态"><el-select v-model="nodeForm.status" style="width: 100%"><el-option label="正常" :value="1" /><el-option label="告警" :value="2" /><el-option label="故障" :value="3" /><el-option label="未知" :value="0" /></el-select></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="坐标"><el-input-number v-model="nodeForm.x" :step="10" /><span style="padding:0 6px">/</span><el-input-number v-model="nodeForm.y" :step="10" /></el-form-item></el-col>
          <el-col :span="24"><el-form-item label="健康检查"><el-input v-model="nodeForm.health_url" /></el-form-item></el-col>
          <el-col :span="24"><el-form-item label="端点JSON"><el-input v-model="nodeForm.endpoints" type="textarea" :rows="2" placeholder='["http://a:8080"]' /></el-form-item></el-col>
          <el-col :span="24"><el-form-item label="元数据JSON"><el-input v-model="nodeForm.metadata" type="textarea" :rows="2" placeholder='{"team":"ops"}' /></el-form-item></el-col>
          <el-col :span="24"><el-form-item label="描述"><el-input v-model="nodeForm.description" type="textarea" :rows="2" /></el-form-item></el-col>
        </el-row>
      </el-form>
      <template #footer>
        <el-button @click="nodeDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="nodeSaving" @click="saveNode">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="edgeDialogVisible" title="新增依赖关系" width="560px">
      <el-form :model="edgeForm" label-width="96px">
        <el-form-item label="源节点" required>
          <el-select v-model="edgeForm.source_id" filterable style="width: 100%">
            <el-option v-for="item in nodes" :key="item.id" :label="item.name" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="目标节点" required>
          <el-select v-model="edgeForm.target_id" filterable style="width: 100%">
            <el-option v-for="item in nodes" :key="item.id" :label="item.name" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="关系类型">
          <el-input v-model="edgeForm.type" placeholder="http/grpc/tcp/mq" />
        </el-form-item>
        <el-form-item label="协议">
          <el-input v-model="edgeForm.protocol" />
        </el-form-item>
        <el-form-item label="端口">
          <el-input-number v-model="edgeForm.port" :min="0" :max="65535" />
        </el-form-item>
        <el-form-item label="说明">
          <el-input v-model="edgeForm.description" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="edgeDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="edgeSaving" @click="saveEdge">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="discoverDialogVisible" title="自动发现拓扑" width="620px">
      <el-form :model="discoverForm" label-width="110px">
        <el-form-item label="发现源" required>
          <el-checkbox-group v-model="discoverForm.sources">
            <el-checkbox label="k8s">K8s</el-checkbox>
            <el-checkbox label="swarm">Swarm</el-checkbox>
            <el-checkbox label="docker">Docker</el-checkbox>
          </el-checkbox-group>
        </el-form-item>
        <el-form-item label="命名空间">
          <el-input v-model="discoverForm.namespace" placeholder="all / default / 其他命名空间" />
        </el-form-item>
        <el-form-item v-if="discoverForm.sources.includes('k8s')" label="K8s集群">
          <el-select v-model="discoverForm.cluster_ids" multiple collapse-tags collapse-tags-tooltip clearable filterable placeholder="不选=全部集群" style="width: 100%">
            <el-option v-for="item in clusterOptions" :key="item.id" :label="item.display_name || item.name || item.id" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="discoverForm.sources.includes('swarm') || discoverForm.sources.includes('docker')" label="Docker主机">
          <el-select v-model="discoverForm.docker_host_ids" multiple collapse-tags collapse-tags-tooltip clearable filterable placeholder="不选=全部主机" style="width: 100%">
            <el-option v-for="item in dockerHostOptions" :key="item.id" :label="item.name || item.id" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-checkbox v-model="discoverForm.replace">覆盖历史自动发现节点</el-checkbox>
        </el-form-item>
        <el-form-item>
          <el-checkbox v-model="discoverForm.auto_layout">完成后自动布局</el-checkbox>
        </el-form-item>
        <el-alert
          type="info"
          show-icon
          :closable="false"
          title="说明"
          description="自动发现会采集 K8s Service/Ingress/Workload、Swarm Service、Docker Container 的结构关系。"
        />
      </el-form>
      <template #footer>
        <el-button @click="discoverDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="discovering" @click="runDiscover">开始发现</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="importDialogVisible" title="导入拓扑" width="620px">
      <el-input v-model="importPayload" type="textarea" :rows="14" placeholder='{"nodes":[],"edges":[]}' />
      <template #footer>
        <el-button @click="importDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="importing" @click="importTopology">导入</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const loading = ref(false)
const nodeSaving = ref(false)
const edgeSaving = ref(false)
const importing = ref(false)
const discovering = ref(false)

const nodes = ref([])
const edges = ref([])
const views = ref([])
const selectedNode = ref(null)
const clusterOptions = ref([])
const dockerHostOptions = ref([])

const nodeDialogVisible = ref(false)
const nodeEditing = ref(false)
const edgeDialogVisible = ref(false)
const importDialogVisible = ref(false)
const discoverDialogVisible = ref(false)

const nodeForm = reactive({
  id: '',
  name: '',
  type: 'service',
  icon: '',
  description: '',
  namespace: '',
  cluster: '',
  endpoints: '[]',
  metadata: '{}',
  status: 1,
  health_url: '',
  x: 0,
  y: 0
})

const edgeForm = reactive({
  source_id: '',
  target_id: '',
  type: 'http',
  protocol: 'HTTP/1.1',
  port: 80,
  description: ''
})

const importPayload = ref('{"nodes":[],"edges":[]}')

const discoverForm = reactive({
  sources: ['k8s', 'swarm', 'docker'],
  namespace: 'all',
  cluster_ids: [],
  docker_host_ids: [],
  replace: true,
  auto_layout: true
})

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })
const nodeFilter = reactive({
  keyword: '',
  source: 'all',
  status: 'all'
})

const unhealthyCount = computed(() => nodes.value.filter(item => Number(item.status) === 2 || Number(item.status) === 3).length)
const sourceOptions = computed(() => {
  const options = new Set()
  for (const item of nodes.value) options.add(nodeSource(item))
  return Array.from(options).sort()
})
const filteredNodes = computed(() => {
  const keyword = nodeFilter.keyword.trim().toLowerCase()
  const source = nodeFilter.source
  const status = nodeFilter.status
  return nodes.value.filter((item) => {
    if (source !== 'all' && nodeSource(item) !== source) return false
    if (status !== 'all' && String(Number(item.status || 0)) !== status) return false
    if (!keyword) return true
    const text = `${item.name || ''} ${item.namespace || ''} ${item.cluster || ''}`.toLowerCase()
    return text.includes(keyword)
  })
})
const filteredEdges = computed(() => {
  if (filteredNodes.value.length === nodes.value.length) return edges.value
  const idSet = new Set(filteredNodes.value.map(item => item.id))
  const nameSet = new Set(filteredNodes.value.map(item => item.name))
  return edges.value.filter((edge) => {
    const sourceByID = edge.source_id && idSet.has(edge.source_id)
    const targetByID = edge.target_id && idSet.has(edge.target_id)
    if (sourceByID && targetByID) return true
    const sourceName = (edge.source_name || nodeName(edge.source_id)).trim()
    const targetName = (edge.target_name || nodeName(edge.target_id)).trim()
    return nameSet.has(sourceName) && nameSet.has(targetName)
  })
})

const nodeName = (id) => nodes.value.find(item => item.id === id)?.name || id || '-'
const nodeSource = (row) => {
  const parsed = parseNodeMetadata(row?.metadata)
  return parsed.topology_source || 'manual'
}
const parseNodeMetadata = (raw) => {
  if (!raw || typeof raw !== 'string') return {}
  try {
    const parsed = JSON.parse(raw)
    if (parsed && typeof parsed === 'object') return parsed
  } catch {}
  return {}
}

const statusText = (status) => {
  const n = Number(status)
  if (n === 1) return '正常'
  if (n === 2) return '告警'
  if (n === 3) return '故障'
  return '未知'
}

const statusType = (status) => {
  const n = Number(status)
  if (n === 1) return 'success'
  if (n === 2) return 'warning'
  if (n === 3) return 'danger'
  return 'info'
}

const fetchTopology = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/v1/topology/data', { headers: authHeaders() })
    if (res.data?.code === 0) {
      nodes.value = res.data.data?.nodes || []
      edges.value = res.data.data?.edges || []
      if (selectedNode.value) {
        selectedNode.value = nodes.value.find(item => item.id === selectedNode.value.id) || null
      }
    }
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '加载拓扑失败')
  } finally {
    loading.value = false
  }
}

const fetchViews = async () => {
  try {
    const res = await axios.get('/api/v1/topology/views', { headers: authHeaders() })
    if (res.data?.code === 0) views.value = res.data.data || []
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '加载视图失败')
  }
}

const fetchDiscoverOptions = async () => {
  try {
    const [k8sResp, dockerResp] = await Promise.all([
      axios.get('/api/v1/k8s/clusters', { headers: authHeaders() }),
      axios.get('/api/v1/docker/hosts', { headers: authHeaders() })
    ])
    if (k8sResp.data?.code === 0) clusterOptions.value = k8sResp.data.data || []
    if (dockerResp.data?.code === 0) dockerHostOptions.value = dockerResp.data.data || []
  } catch {
    // Keep page available even when one plugin is disabled or unauthorized.
  }
}

const refreshAll = async () => {
  await Promise.all([fetchTopology(), fetchViews()])
}

const selectNode = (row) => {
  selectedNode.value = row
}

const resetNodeFilter = () => {
  nodeFilter.keyword = ''
  nodeFilter.source = 'all'
  nodeFilter.status = 'all'
}

const resetNodeForm = () => {
  nodeForm.id = ''
  nodeForm.name = ''
  nodeForm.type = 'service'
  nodeForm.icon = ''
  nodeForm.description = ''
  nodeForm.namespace = ''
  nodeForm.cluster = ''
  nodeForm.endpoints = '[]'
  nodeForm.metadata = '{}'
  nodeForm.status = 1
  nodeForm.health_url = ''
  nodeForm.x = 0
  nodeForm.y = 0
}

const openNodeDialog = (row) => {
  nodeEditing.value = !!row
  resetNodeForm()
  if (row) {
    nodeForm.id = row.id
    nodeForm.name = row.name || ''
    nodeForm.type = row.type || 'service'
    nodeForm.icon = row.icon || ''
    nodeForm.description = row.description || ''
    nodeForm.namespace = row.namespace || ''
    nodeForm.cluster = row.cluster || ''
    nodeForm.endpoints = row.endpoints || '[]'
    nodeForm.metadata = row.metadata || '{}'
    nodeForm.status = Number(row.status || 1)
    nodeForm.health_url = row.health_url || ''
    nodeForm.x = Number(row.x || 0)
    nodeForm.y = Number(row.y || 0)
  }
  nodeDialogVisible.value = true
}

const saveNode = async () => {
  if (!nodeForm.name.trim()) {
    ElMessage.warning('请输入节点名称')
    return
  }
  try {
    JSON.parse(nodeForm.endpoints || '[]')
    JSON.parse(nodeForm.metadata || '{}')
  } catch {
    ElMessage.warning('端点或元数据JSON格式不正确')
    return
  }

  nodeSaving.value = true
  try {
    const payload = {
      name: nodeForm.name.trim(),
      type: nodeForm.type,
      icon: nodeForm.icon,
      description: nodeForm.description,
      namespace: nodeForm.namespace,
      cluster: nodeForm.cluster,
      endpoints: nodeForm.endpoints,
      metadata: nodeForm.metadata,
      status: Number(nodeForm.status),
      health_url: nodeForm.health_url,
      x: Number(nodeForm.x || 0),
      y: Number(nodeForm.y || 0)
    }

    if (nodeEditing.value && nodeForm.id) {
      await axios.put(`/api/v1/topology/nodes/${nodeForm.id}`, payload, { headers: authHeaders() })
      ElMessage.success('更新成功')
    } else {
      await axios.post('/api/v1/topology/nodes', payload, { headers: authHeaders() })
      ElMessage.success('创建成功')
    }

    nodeDialogVisible.value = false
    await fetchTopology()
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '保存失败')
  } finally {
    nodeSaving.value = false
  }
}

const removeNode = async (row) => {
  try {
    await ElMessageBox.confirm(`确认删除节点 ${row.name} ?`, '提示', { type: 'warning' })
    await axios.delete(`/api/v1/topology/nodes/${row.id}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    if (selectedNode.value?.id === row.id) selectedNode.value = null
    await fetchTopology()
  } catch (err) {
    if (err !== 'cancel') ElMessage.error(err.response?.data?.message || '删除失败')
  }
}

const openEdgeDialog = () => {
  edgeForm.source_id = selectedNode.value?.id || ''
  edgeForm.target_id = ''
  edgeForm.type = 'http'
  edgeForm.protocol = 'HTTP/1.1'
  edgeForm.port = 80
  edgeForm.description = ''
  edgeDialogVisible.value = true
}

const saveEdge = async () => {
  if (!edgeForm.source_id || !edgeForm.target_id) {
    ElMessage.warning('请选择源节点和目标节点')
    return
  }

  edgeSaving.value = true
  try {
    const sourceName = nodeName(edgeForm.source_id)
    const targetName = nodeName(edgeForm.target_id)
    await axios.post('/api/v1/topology/edges', {
      source_id: edgeForm.source_id,
      target_id: edgeForm.target_id,
      source_name: sourceName,
      target_name: targetName,
      type: edgeForm.type,
      protocol: edgeForm.protocol,
      port: Number(edgeForm.port || 0),
      description: edgeForm.description
    }, { headers: authHeaders() })
    ElMessage.success('关系创建成功')
    edgeDialogVisible.value = false
    await fetchTopology()
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '创建关系失败')
  } finally {
    edgeSaving.value = false
  }
}

const removeEdge = async (row) => {
  try {
    await ElMessageBox.confirm('确认删除该依赖关系?', '提示', { type: 'warning' })
    await axios.delete(`/api/v1/topology/edges/${row.id}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    await fetchTopology()
  } catch (err) {
    if (err !== 'cancel') ElMessage.error(err.response?.data?.message || '删除失败')
  }
}

const autoLayout = async () => {
  try {
    const res = await axios.post('/api/v1/topology/layout/auto', {}, { headers: authHeaders() })
    ElMessage.success(res.data?.message || '已触发自动布局')
    await fetchTopology()
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '自动布局失败')
  }
}

const exportTopology = async () => {
  try {
    const res = await axios.get('/api/v1/topology/export', {
      headers: authHeaders(),
      responseType: 'blob'
    })

    const blob = new Blob([res.data], { type: 'application/json' })
    const link = document.createElement('a')
    link.href = URL.createObjectURL(blob)
    link.download = `topology-${Date.now()}.json`
    link.click()
    URL.revokeObjectURL(link.href)
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '导出失败')
  }
}

const openDiscoverDialog = () => {
  discoverDialogVisible.value = true
}

const runDiscover = async () => {
  if (!discoverForm.sources.length) {
    ElMessage.warning('请至少选择一个发现源')
    return
  }
  discovering.value = true
  try {
    const payload = {
      sources: discoverForm.sources,
      namespace: (discoverForm.namespace || 'all').trim(),
      cluster_ids: discoverForm.cluster_ids,
      docker_host_ids: discoverForm.docker_host_ids,
      replace: !!discoverForm.replace,
      auto_layout: !!discoverForm.auto_layout
    }
    const res = await axios.post('/api/v1/topology/discover', payload, { headers: authHeaders() })
    const data = res.data?.data || {}
    const msg = `发现完成：节点 ${data.discovered_nodes || 0}，关系 ${data.discovered_edges || 0}`
    ElMessage.success(msg)
    if (Array.isArray(data.warnings) && data.warnings.length) {
      ElMessage.warning(`发现告警：${data.warnings.slice(0, 2).join('；')}`)
    }
    discoverDialogVisible.value = false
    await refreshAll()
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '自动发现失败')
  } finally {
    discovering.value = false
  }
}

const openImportDialog = () => {
  importPayload.value = '{"nodes":[],"edges":[]}'
  importDialogVisible.value = true
}

const importTopology = async () => {
  try {
    JSON.parse(importPayload.value || '{}')
  } catch {
    ElMessage.warning('导入内容必须是合法JSON')
    return
  }

  importing.value = true
  try {
    await axios.post('/api/v1/topology/import', JSON.parse(importPayload.value || '{}'), { headers: authHeaders() })
    ElMessage.success('导入请求已提交')
    importDialogVisible.value = false
    await fetchTopology()
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '导入失败')
  } finally {
    importing.value = false
  }
}

onMounted(refreshAll)
onMounted(fetchDiscoverOptions)
</script>

<style scoped>
.page-card { max-width: 1400px; margin: 0 auto; }
.page-header { display: flex; justify-content: space-between; align-items: center; gap: 12px; margin-bottom: 12px; }
.page-desc { color: #909399; margin: 4px 0 0; }
.page-actions { display: flex; align-items: center; gap: 8px; }
.k { color: #909399; font-size: 12px; }
.v { font-size: 26px; font-weight: 700; margin-top: 4px; }
.v.danger { color: #f56c6c; }
.mb-12 { margin-bottom: 12px; }
.section-header { display: flex; justify-content: space-between; align-items: center; gap: 8px; }
.sub-title { font-size: 13px; color: #606266; margin-bottom: 8px; }
.table-filters { display: flex; gap: 8px; margin-bottom: 8px; }
.filter-input { flex: 1; }
.filter-select { width: 140px; }
</style>
