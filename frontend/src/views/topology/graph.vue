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

    <el-row :gutter="12" class="mb-12">
      <el-col :md="14" :sm="24">
        <el-card class="overview-card">
          <template #header>
            <div class="section-header">
              <span>来源与集群分布</span>
              <el-button text @click="resetNodeFilter">重置过滤</el-button>
            </div>
          </template>
          <div class="source-grid">
            <div
              v-for="item in sourceStats"
              :key="item.source"
              class="source-item"
              :class="{ active: nodeFilter.source === item.source }"
              @click="nodeFilter.source = item.source"
            >
              <div class="source-title">
                <span class="source-dot" :style="{ backgroundColor: sourceColor(item.source) }"></span>
                <span>{{ item.source }}</span>
              </div>
              <div class="source-value">{{ item.count }}</div>
            </div>
          </div>
          <el-divider />
          <div class="cluster-list">
            <div v-for="item in clusterStats.slice(0, 8)" :key="item.cluster" class="cluster-row">
              <span class="cluster-name" :title="item.cluster">{{ item.cluster || '-' }}</span>
              <el-tag size="small">{{ item.count }}</el-tag>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :md="10" :sm="24">
        <el-card class="overview-card">
          <template #header>
            <div class="section-header">
              <span>关键路径/影响分</span>
              <el-button text :loading="dependencyLoading" @click="fetchDependencyInsights">刷新分析</el-button>
            </div>
          </template>
          <el-empty v-if="!dependencyInsights.length" description="暂无分析结果" :image-size="72" />
          <el-table v-else :data="dependencyInsights.slice(0, 6)" size="small" border>
            <el-table-column prop="service_name" label="服务" min-width="160" show-overflow-tooltip />
            <el-table-column prop="impact_score" label="影响分" width="84" />
            <el-table-column label="关键路径" width="92">
              <template #default="{ row }">
                <el-tag :type="row.critical_path ? 'danger' : 'info'" size="small">{{ row.critical_path ? '是' : '否' }}</el-tag>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <el-card class="mb-12">
      <template #header>
        <div class="section-header">
          <span>拓扑画布</span>
          <div class="canvas-actions">
            <el-select v-model="canvasMode" size="small" class="canvas-select">
              <el-option label="按集群泳道" value="lane" />
              <el-option label="按布局坐标" value="manual" />
            </el-select>
            <el-button
              v-if="canvasMode === 'manual'"
              size="small"
              type="primary"
              :loading="savingLayout"
              :disabled="!dirtyNodeIDs.length"
              @click="saveCanvasLayout"
            >
              保存坐标({{ dirtyNodeIDs.length }})
            </el-button>
            <el-button size="small" :disabled="!graphNodes.length" @click="exportCanvasSvg">导出SVG</el-button>
            <el-button size="small" :disabled="!graphNodes.length" @click="exportCanvasPng">导出PNG</el-button>
            <el-switch v-model="showEdgeLabel" size="small" inline-prompt active-text="边标签" inactive-text="边标签" />
          </div>
        </div>
      </template>
      <div v-if="!graphNodes.length" class="canvas-empty">
        <el-empty description="暂无可渲染拓扑数据（请先自动发现或调整筛选）" :image-size="90" />
      </div>
      <div v-else class="topology-canvas">
        <div v-if="canvasMode === 'lane'" class="lane-header">
          <div v-for="lane in laneColumns" :key="lane" class="lane-item">{{ lane || '-' }}</div>
        </div>
        <svg class="edge-layer" :viewBox="`0 0 ${canvasSize.width} ${canvasSize.height}`" preserveAspectRatio="none">
          <defs>
            <marker id="arrow" viewBox="0 0 8 8" refX="7" refY="4" markerWidth="8" markerHeight="8" orient="auto-start-reverse">
              <path d="M0,0 L8,4 L0,8 Z" fill="#9ca3af" />
            </marker>
          </defs>
          <line
            v-for="item in graphEdges"
            :key="item.id"
            :x1="item.x1"
            :y1="item.y1"
            :x2="item.x2"
            :y2="item.y2"
            :stroke="item.critical ? '#f56c6c' : '#9ca3af'"
            :stroke-width="item.critical ? 2.4 : 1.4"
            marker-end="url(#arrow)"
          />
          <text
            v-if="showEdgeLabel"
            v-for="item in graphEdges"
            :key="`${item.id}-label`"
            :x="(item.x1 + item.x2) / 2"
            :y="(item.y1 + item.y2) / 2 - 4"
            class="edge-label"
          >
            {{ item.label }}
          </text>
        </svg>
        <div
          v-for="item in graphNodes"
          :key="item.id"
          class="graph-node"
          :class="{ selected: selectedNode?.id === item.id, critical: criticalNodeNames.has(item.name), draggable: canvasMode === 'manual' }"
          :style="{ left: `${item.left}px`, top: `${item.top}px`, borderColor: sourceColor(item.source) }"
          @mousedown.stop="startNodeDrag($event, item)"
          @click="selectNode(item.raw)"
        >
          <div class="graph-node-title">{{ item.name }}</div>
          <div class="graph-node-meta">{{ item.source }} | {{ item.type }}</div>
        </div>
      </div>
    </el-card>

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
            <el-select v-model="nodeFilter.edge_type" class="filter-select" placeholder="链路类型">
              <el-option label="全部链路" value="all" />
              <el-option v-for="item in edgeTypeOptions" :key="item" :label="item" :value="item" />
            </el-select>
            <el-checkbox v-model="nodeFilter.critical_only" class="filter-toggle">关键路径</el-checkbox>
            <el-checkbox v-model="nodeFilter.alert_chain_only" class="filter-toggle">告警链路</el-checkbox>
            <el-checkbox v-model="nodeFilter.neighbor_only" class="filter-toggle">一跳邻居</el-checkbox>
          </div>
          <el-table :data="filteredNodes" v-loading="loading" stripe @row-click="selectNode" :row-class-name="nodeRowClassName">
            <el-table-column prop="name" label="名称" min-width="160" />
            <el-table-column label="来源" width="120">
              <template #default="{ row }">
                <el-tag size="small" effect="plain" :color="sourceColor(nodeSource(row))" :style="{ color: '#fff', borderColor: 'transparent' }">
                  {{ nodeSource(row) }}
                </el-tag>
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
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const loading = ref(false)
const nodeSaving = ref(false)
const edgeSaving = ref(false)
const importing = ref(false)
const discovering = ref(false)
const dependencyLoading = ref(false)
const canvasMode = ref('lane')
const showEdgeLabel = ref(false)
const savingLayout = ref(false)

const nodes = ref([])
const edges = ref([])
const views = ref([])
const selectedNode = ref(null)
const dependencyInsights = ref([])
const clusterOptions = ref([])
const dockerHostOptions = ref([])
const dirtyNodeIDs = ref([])
const dragOverrideMap = ref({})
const dragState = reactive({
  nodeID: '',
  startX: 0,
  startY: 0,
  originLeft: 0,
  originTop: 0
})

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
  status: 'all',
  edge_type: 'all',
  critical_only: false,
  alert_chain_only: false,
  neighbor_only: false
})

const GRAPH_NODE_WIDTH = 190
const GRAPH_NODE_HEIGHT = 56

const unhealthyCount = computed(() => nodes.value.filter(item => Number(item.status) === 2 || Number(item.status) === 3).length)
const criticalNodeNames = computed(() => new Set(
  dependencyInsights.value
    .filter(item => item.critical_path)
    .map(item => item.service_name)
))
const alertChainNames = computed(() => {
  const mapByID = new Map(nodes.value.map(item => [item.id, item.name]))
  const names = new Set()
  for (const node of nodes.value) {
    const st = Number(node.status || 0)
    if (st === 2 || st === 3) names.add(node.name)
  }
  for (const edge of edges.value) {
    const sourceName = (edge.source_name || mapByID.get(edge.source_id) || edge.source_id || '').trim()
    const targetName = (edge.target_name || mapByID.get(edge.target_id) || edge.target_id || '').trim()
    if (!sourceName || !targetName) continue
    if (names.has(sourceName) || names.has(targetName)) {
      names.add(sourceName)
      names.add(targetName)
    }
  }
  return names
})
const sourceOptions = computed(() => {
  const options = new Set()
  for (const item of nodes.value) options.add(nodeSource(item))
  return Array.from(options).sort()
})
const sourceStats = computed(() => {
  const map = new Map()
  for (const item of nodes.value) {
    const source = nodeSource(item)
    map.set(source, (map.get(source) || 0) + 1)
  }
  return Array.from(map.entries())
    .map(([source, count]) => ({ source, count }))
    .sort((a, b) => b.count - a.count)
})
const edgeTypeOptions = computed(() => {
  const set = new Set()
  for (const edge of edges.value) {
    const key = (edge.type || '').trim()
    if (key) set.add(key)
  }
  return Array.from(set).sort((a, b) => a.localeCompare(b))
})
const clusterStats = computed(() => {
  const map = new Map()
  for (const item of nodes.value) {
    const key = item.cluster || '-'
    map.set(key, (map.get(key) || 0) + 1)
  }
  return Array.from(map.entries())
    .map(([cluster, count]) => ({ cluster, count }))
    .sort((a, b) => b.count - a.count)
})
const filteredNodes = computed(() => {
  const keyword = nodeFilter.keyword.trim().toLowerCase()
  const source = nodeFilter.source
  const status = nodeFilter.status
  let list = nodes.value.filter((item) => {
    if (source !== 'all' && nodeSource(item) !== source) return false
    if (status !== 'all' && String(Number(item.status || 0)) !== status) return false
    if (nodeFilter.critical_only && !criticalNodeNames.value.has(item.name)) return false
    if (nodeFilter.alert_chain_only && !alertChainNames.value.has(item.name)) return false
    if (!keyword) return true
    const text = `${item.name || ''} ${item.namespace || ''} ${item.cluster || ''}`.toLowerCase()
    return text.includes(keyword)
  })

  if (nodeFilter.neighbor_only && selectedNode.value) {
    const selectedID = selectedNode.value.id
    const selectedName = selectedNode.value.name || nodeName(selectedID)
    const neighborIDs = new Set([selectedID])
    const neighborNames = new Set([selectedName])
    for (const edge of edges.value) {
      const sourceName = (edge.source_name || nodeName(edge.source_id)).trim()
      const targetName = (edge.target_name || nodeName(edge.target_id)).trim()
      const hitByID = selectedID && (edge.source_id === selectedID || edge.target_id === selectedID)
      const hitByName = selectedName && (sourceName === selectedName || targetName === selectedName)
      if (!hitByID && !hitByName) continue
      if (edge.source_id) neighborIDs.add(edge.source_id)
      if (edge.target_id) neighborIDs.add(edge.target_id)
      if (sourceName) neighborNames.add(sourceName)
      if (targetName) neighborNames.add(targetName)
    }
    list = list.filter(item => neighborIDs.has(item.id) || neighborNames.has(item.name))
  }

  return list
})
const filteredEdges = computed(() => {
  const edgeType = nodeFilter.edge_type
  const edgePool = edgeType === 'all'
    ? edges.value
    : edges.value.filter(edge => (edge.type || '').trim() === edgeType)

  if (filteredNodes.value.length === nodes.value.length && edgeType === 'all') return edgePool
  const idSet = new Set(filteredNodes.value.map(item => item.id))
  const nameSet = new Set(filteredNodes.value.map(item => item.name))
  return edgePool.filter((edge) => {
    const sourceByID = edge.source_id && idSet.has(edge.source_id)
    const targetByID = edge.target_id && idSet.has(edge.target_id)
    if (sourceByID && targetByID) return true
    const sourceName = (edge.source_name || nodeName(edge.source_id)).trim()
    const targetName = (edge.target_name || nodeName(edge.target_id)).trim()
    return nameSet.has(sourceName) && nameSet.has(targetName)
  })
})
const laneColumns = computed(() => {
  const laneSet = new Set()
  for (const item of filteredNodes.value) laneSet.add(item.cluster || '-')
  return Array.from(laneSet).sort((a, b) => a.localeCompare(b))
})
const graphNodes = computed(() => {
  const list = filteredNodes.value
  if (!list.length) return []
  if (canvasMode.value === 'lane') {
    const laneIndex = new Map()
    laneColumns.value.forEach((lane, idx) => laneIndex.set(lane, idx))
    const laneOrder = new Map()
    const sorted = [...list].sort((a, b) => {
      const clusterCmp = (a.cluster || '-').localeCompare(b.cluster || '-')
      if (clusterCmp !== 0) return clusterCmp
      return (a.name || '').localeCompare(b.name || '')
    })
    return sorted.map((raw) => {
      const lane = raw.cluster || '-'
      const col = laneIndex.get(lane) || 0
      const row = laneOrder.get(lane) || 0
      laneOrder.set(lane, row + 1)
      const left = 36 + col * 260
      const top = 64 + row * 84
      const override = dragOverrideMap.value[raw.id]
      return {
        id: raw.id,
        name: raw.name,
        type: raw.type || '-',
        source: nodeSource(raw),
        left: override ? override.left : left,
        top: override ? override.top : top,
        raw
      }
    })
  }
  const sorted = [...list].sort((a, b) => (a.name || '').localeCompare(b.name || ''))
  return sorted.map((raw) => {
    const x = Math.max(0, Number(raw.x || 0))
    const y = Math.max(0, Number(raw.y || 0))
    const left = 36 + x
    const top = 64 + y
    const override = dragOverrideMap.value[raw.id]
    return {
      id: raw.id,
      name: raw.name,
      type: raw.type || '-',
      source: nodeSource(raw),
      left: override ? override.left : left,
      top: override ? override.top : top,
      raw
    }
  })
})
const graphNodeIndex = computed(() => {
  const byID = new Map()
  const byName = new Map()
  for (const item of graphNodes.value) {
    byID.set(item.id, item)
    byName.set(item.name, item)
  }
  return { byID, byName }
})
const graphEdges = computed(() => {
  const result = []
  for (const edge of filteredEdges.value) {
    const sourceName = (edge.source_name || nodeName(edge.source_id)).trim()
    const targetName = (edge.target_name || nodeName(edge.target_id)).trim()
    const sourceNode = edge.source_id ? graphNodeIndex.value.byID.get(edge.source_id) : graphNodeIndex.value.byName.get(sourceName)
    const targetNode = edge.target_id ? graphNodeIndex.value.byID.get(edge.target_id) : graphNodeIndex.value.byName.get(targetName)
    if (!sourceNode || !targetNode) continue
    result.push({
      id: edge.id,
      x1: sourceNode.left + GRAPH_NODE_WIDTH / 2,
      y1: sourceNode.top + GRAPH_NODE_HEIGHT / 2,
      x2: targetNode.left + GRAPH_NODE_WIDTH / 2,
      y2: targetNode.top + GRAPH_NODE_HEIGHT / 2,
      label: [edge.type, edge.port ? `:${edge.port}` : ''].join(''),
      critical: criticalNodeNames.value.has(sourceNode.name) || criticalNodeNames.value.has(targetNode.name)
    })
  }
  return result
})
const canvasSize = computed(() => {
  if (!graphNodes.value.length) return { width: 960, height: 320 }
  const maxLeft = Math.max(...graphNodes.value.map(item => item.left))
  const maxTop = Math.max(...graphNodes.value.map(item => item.top))
  const laneWidth = laneColumns.value.length * 260 + 80
  const width = Math.max(960, maxLeft + GRAPH_NODE_WIDTH + 80, laneWidth)
  const height = Math.max(320, maxTop + GRAPH_NODE_HEIGHT + 80)
  return { width, height }
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
const sourceColor = (source) => {
  const map = {
    k8s: '#409eff',
    swarm: '#67c23a',
    docker: '#e6a23c',
    manual: '#909399'
  }
  return map[source] || '#606266'
}
const nodeRowClassName = ({ row }) => {
  if (criticalNodeNames.value.has(row.name)) return 'critical-node-row'
  return ''
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

const fetchDependencyInsights = async () => {
  dependencyLoading.value = true
  try {
    const res = await axios.get('/api/v1/topology/analyze', { headers: authHeaders() })
    if (res.data?.code === 0) dependencyInsights.value = res.data.data || []
  } catch {
    dependencyInsights.value = []
  } finally {
    dependencyLoading.value = false
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
  await Promise.all([fetchTopology(), fetchViews(), fetchDependencyInsights()])
}

const selectNode = (row) => {
  selectedNode.value = row
}

const resetNodeFilter = () => {
  nodeFilter.keyword = ''
  nodeFilter.source = 'all'
  nodeFilter.status = 'all'
  nodeFilter.edge_type = 'all'
  nodeFilter.critical_only = false
  nodeFilter.alert_chain_only = false
  nodeFilter.neighbor_only = false
}

const startNodeDrag = (event, item) => {
  if (canvasMode.value !== 'manual') return
  dragState.nodeID = item.id
  dragState.startX = event.clientX
  dragState.startY = event.clientY
  dragState.originLeft = item.left
  dragState.originTop = item.top
}

const onMouseMove = (event) => {
  if (!dragState.nodeID) return
  const dx = event.clientX - dragState.startX
  const dy = event.clientY - dragState.startY
  const nextLeft = Math.max(10, Math.round(dragState.originLeft + dx))
  const nextTop = Math.max(10, Math.round(dragState.originTop + dy))
  dragOverrideMap.value = {
    ...dragOverrideMap.value,
    [dragState.nodeID]: { left: nextLeft, top: nextTop }
  }
}

const onMouseUp = () => {
  if (!dragState.nodeID) return
  if (!dirtyNodeIDs.value.includes(dragState.nodeID)) {
    dirtyNodeIDs.value = [...dirtyNodeIDs.value, dragState.nodeID]
  }
  dragState.nodeID = ''
}

const saveCanvasLayout = async () => {
  if (!dirtyNodeIDs.value.length) return
  savingLayout.value = true
  try {
    const requests = dirtyNodeIDs.value.map((id) => {
      const pos = dragOverrideMap.value[id]
      if (!pos) return Promise.resolve()
      return axios.put(`/api/v1/topology/nodes/${id}/position`, {
        x: Math.max(0, Math.round(pos.left - 36)),
        y: Math.max(0, Math.round(pos.top - 64))
      }, { headers: authHeaders() })
    })
    await Promise.all(requests)
    ElMessage.success(`已保存 ${dirtyNodeIDs.value.length} 个节点坐标`)
    dirtyNodeIDs.value = []
    dragOverrideMap.value = {}
    await fetchTopology()
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '保存坐标失败')
  } finally {
    savingLayout.value = false
  }
}

const xmlEscape = (input) => {
  return String(input ?? '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&apos;')
}

const trimText = (text, maxLen = 34) => {
  const value = String(text || '')
  if (value.length <= maxLen) return value
  return `${value.slice(0, maxLen - 1)}…`
}

const drawRoundedRect = (ctx, x, y, width, height, radius) => {
  const r = Math.min(radius, width / 2, height / 2)
  ctx.beginPath()
  ctx.moveTo(x + r, y)
  ctx.arcTo(x + width, y, x + width, y + height, r)
  ctx.arcTo(x + width, y + height, x, y + height, r)
  ctx.arcTo(x, y + height, x, y, r)
  ctx.arcTo(x, y, x + width, y, r)
  ctx.closePath()
}

const drawArrowLine = (ctx, x1, y1, x2, y2, color) => {
  ctx.strokeStyle = color
  ctx.lineWidth = color === '#f56c6c' ? 2.2 : 1.2
  ctx.beginPath()
  ctx.moveTo(x1, y1)
  ctx.lineTo(x2, y2)
  ctx.stroke()

  const headLen = 8
  const angle = Math.atan2(y2 - y1, x2 - x1)
  ctx.beginPath()
  ctx.moveTo(x2, y2)
  ctx.lineTo(x2 - headLen * Math.cos(angle - Math.PI / 6), y2 - headLen * Math.sin(angle - Math.PI / 6))
  ctx.lineTo(x2 - headLen * Math.cos(angle + Math.PI / 6), y2 - headLen * Math.sin(angle + Math.PI / 6))
  ctx.closePath()
  ctx.fillStyle = color
  ctx.fill()
}

const topologyExportFileName = (suffix) => {
  const ts = new Date().toISOString().replaceAll(':', '-').replace('T', '_').slice(0, 19)
  return `topology-canvas-${ts}.${suffix}`
}

const downloadBlob = (blob, name) => {
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = name
  link.click()
  URL.revokeObjectURL(url)
}

const exportCanvasSvg = () => {
  if (!graphNodes.value.length) return
  const width = canvasSize.value.width
  const height = canvasSize.value.height
  const lines = graphEdges.value.map((item) => `
    <line x1="${item.x1}" y1="${item.y1}" x2="${item.x2}" y2="${item.y2}" stroke="${item.critical ? '#f56c6c' : '#9ca3af'}" stroke-width="${item.critical ? 2.2 : 1.2}" marker-end="url(#arrow)" />`).join('')
  const labels = showEdgeLabel.value
    ? graphEdges.value.map((item) => `
    <text x="${(item.x1 + item.x2) / 2}" y="${(item.y1 + item.y2) / 2 - 4}" font-size="11" text-anchor="middle" fill="#6b7280">${xmlEscape(item.label)}</text>`).join('')
    : ''
  const rects = graphNodes.value.map((item) => {
    const fill = criticalNodeNames.value.has(item.name) ? '#fff7f7' : '#ffffff'
    return `
    <rect x="${item.left}" y="${item.top}" width="${GRAPH_NODE_WIDTH}" height="${GRAPH_NODE_HEIGHT}" rx="10" ry="10" fill="${fill}" stroke="${sourceColor(item.source)}" stroke-width="2" />
    <text x="${item.left + 10}" y="${item.top + 20}" font-size="12" fill="#303133">${xmlEscape(trimText(item.name, 32))}</text>
    <text x="${item.left + 10}" y="${item.top + 37}" font-size="10" fill="#909399">${xmlEscape(`${item.source} | ${item.type}`)}</text>`
  }).join('')

  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}" viewBox="0 0 ${width} ${height}">
  <defs>
    <marker id="arrow" viewBox="0 0 8 8" refX="7" refY="4" markerWidth="8" markerHeight="8" orient="auto-start-reverse">
      <path d="M0,0 L8,4 L0,8 Z" fill="#9ca3af" />
    </marker>
  </defs>
  <rect x="0" y="0" width="${width}" height="${height}" fill="#f8f9fb" />
  ${lines}
  ${labels}
  ${rects}
</svg>`
  downloadBlob(new Blob([svg], { type: 'image/svg+xml;charset=utf-8' }), topologyExportFileName('svg'))
}

const exportCanvasPng = async () => {
  if (!graphNodes.value.length) return
  const ratio = window.devicePixelRatio || 1
  const width = canvasSize.value.width
  const height = canvasSize.value.height
  const canvas = document.createElement('canvas')
  canvas.width = Math.round(width * ratio)
  canvas.height = Math.round(height * ratio)
  const ctx = canvas.getContext('2d')
  if (!ctx) {
    ElMessage.error('当前浏览器不支持画布导出')
    return
  }
  ctx.scale(ratio, ratio)
  ctx.fillStyle = '#f8f9fb'
  ctx.fillRect(0, 0, width, height)

  for (const edge of graphEdges.value) {
    drawArrowLine(ctx, edge.x1, edge.y1, edge.x2, edge.y2, edge.critical ? '#f56c6c' : '#9ca3af')
    if (showEdgeLabel.value && edge.label) {
      ctx.fillStyle = '#6b7280'
      ctx.font = '11px sans-serif'
      ctx.textAlign = 'center'
      ctx.fillText(edge.label, (edge.x1 + edge.x2) / 2, (edge.y1 + edge.y2) / 2 - 4)
    }
  }

  for (const node of graphNodes.value) {
    const isCritical = criticalNodeNames.value.has(node.name)
    drawRoundedRect(ctx, node.left, node.top, GRAPH_NODE_WIDTH, GRAPH_NODE_HEIGHT, 10)
    ctx.fillStyle = isCritical ? '#fff7f7' : '#ffffff'
    ctx.fill()
    ctx.lineWidth = 2
    ctx.strokeStyle = sourceColor(node.source)
    ctx.stroke()

    ctx.fillStyle = '#303133'
    ctx.font = '12px sans-serif'
    ctx.textAlign = 'left'
    ctx.fillText(trimText(node.name, 32), node.left + 10, node.top + 20)
    ctx.fillStyle = '#909399'
    ctx.font = '10px sans-serif'
    ctx.fillText(`${node.source} | ${node.type}`, node.left + 10, node.top + 37)
  }

  await new Promise((resolve) => {
    canvas.toBlob((blob) => {
      if (!blob) {
        ElMessage.error('导出PNG失败')
        resolve(null)
        return
      }
      downloadBlob(blob, topologyExportFileName('png'))
      resolve(null)
    }, 'image/png', 0.98)
  })
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
    dirtyNodeIDs.value = []
    dragOverrideMap.value = {}
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
onMounted(() => {
  window.addEventListener('mousemove', onMouseMove)
  window.addEventListener('mouseup', onMouseUp)
})
onBeforeUnmount(() => {
  window.removeEventListener('mousemove', onMouseMove)
  window.removeEventListener('mouseup', onMouseUp)
})
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
.filter-toggle { display: flex; align-items: center; padding: 0 6px; border: 1px solid #ebeef5; border-radius: 6px; background: #fff; }
.overview-card { min-height: 274px; }
.source-grid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 8px; }
.source-item { border: 1px solid #ebeef5; border-radius: 8px; padding: 10px; cursor: pointer; background: #fff; }
.source-item.active { border-color: #409eff; box-shadow: 0 0 0 1px #409eff inset; }
.source-title { display: flex; align-items: center; gap: 6px; color: #606266; font-size: 12px; }
.source-dot { width: 8px; height: 8px; border-radius: 999px; display: inline-block; }
.source-value { font-size: 22px; font-weight: 700; margin-top: 6px; color: #303133; }
.cluster-list { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 8px; }
.cluster-row { display: flex; align-items: center; justify-content: space-between; border: 1px solid #f2f3f5; border-radius: 6px; padding: 6px 8px; }
.cluster-name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 240px; color: #606266; font-size: 12px; }
.canvas-actions { display: flex; align-items: center; gap: 8px; }
.canvas-select { width: 140px; }
.canvas-empty { min-height: 220px; display: flex; align-items: center; justify-content: center; }
.topology-canvas { position: relative; min-height: 360px; border: 1px solid #ebeef5; border-radius: 8px; overflow: auto; background: linear-gradient(180deg, #fcfdff 0%, #f8f9fb 100%); }
.lane-header { position: sticky; top: 0; z-index: 2; display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 8px; padding: 10px; background: rgba(255, 255, 255, 0.9); backdrop-filter: blur(2px); border-bottom: 1px solid #ebeef5; }
.lane-item { background: #f5f7fa; color: #606266; border: 1px solid #e9edf3; border-radius: 6px; padding: 6px 8px; font-size: 12px; text-align: center; font-weight: 600; }
.edge-layer { position: absolute; left: 0; top: 0; width: 100%; height: 100%; pointer-events: none; z-index: 1; }
.edge-label { fill: #6b7280; font-size: 11px; text-anchor: middle; dominant-baseline: middle; }
.graph-node { position: absolute; width: 190px; min-height: 56px; border: 2px solid #dcdfe6; border-radius: 10px; background: #fff; box-shadow: 0 4px 14px rgba(0, 0, 0, 0.04); padding: 8px 10px; z-index: 3; cursor: pointer; transition: all 0.18s ease; }
.graph-node.draggable { cursor: grab; }
.graph-node.draggable:active { cursor: grabbing; }
.graph-node:hover { transform: translateY(-1px); box-shadow: 0 8px 18px rgba(0, 0, 0, 0.08); }
.graph-node.selected { box-shadow: 0 0 0 2px rgba(64, 158, 255, 0.25), 0 10px 20px rgba(64, 158, 255, 0.16); }
.graph-node.critical { background: #fff7f7; }
.graph-node-title { font-size: 13px; font-weight: 600; color: #303133; line-height: 1.35; word-break: break-all; }
.graph-node-meta { margin-top: 4px; font-size: 11px; color: #909399; text-transform: lowercase; }
:deep(.critical-node-row > td) { background: #fff8f8 !important; }
@media (max-width: 900px) {
  .source-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .cluster-list { grid-template-columns: 1fr; }
}
</style>
