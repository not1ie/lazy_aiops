<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>容器平台总览</h2>
        <p class="page-desc">把集群、工作负载、服务与诊断入口放在一个视图里，按运维排障链路组织信息。</p>
      </div>
      <div class="page-actions">
        <el-select v-model="clusterId" class="w-52" placeholder="选择集群" @change="refreshClusterData">
          <el-option
            v-for="c in clusters"
            :key="c.id"
            :label="c.display_name || c.name"
            :value="c.id"
          />
        </el-select>
        <el-button :loading="loading" icon="Refresh" @click="refreshAll">刷新</el-button>
      </div>
    </div>

    <el-row :gutter="12" class="summary-row">
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">集群总数</div><div class="metric-value">{{ stats.clusterTotal }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">在线集群</div><div class="metric-value ok">{{ stats.clusterOnline }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">维护集群</div><div class="metric-value warning">{{ stats.clusterMaintenance }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">节点总数</div><div class="metric-value">{{ stats.nodeTotal }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">命名空间</div><div class="metric-value">{{ stats.namespaceTotal }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">工作负载</div><div class="metric-value">{{ stats.workloadTotal }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">异常工作负载</div><div class="metric-value danger">{{ stats.degradedWorkloads }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card>
          <div class="metric-title">待处置积压</div>
          <div class="metric-value warning">{{ pendingBacklog }}</div>
          <div class="metric-sub">Warning {{ stats.warningEvents }}</div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="12">
      <el-col :span="10">
        <el-card>
          <template #header>平台健康</template>
          <div class="health-row">
            <span>集群在线率</span>
            <strong>{{ clusterOnlineRate }}%</strong>
          </div>
          <el-progress :percentage="clusterOnlineRate" :stroke-width="14" />
          <div class="health-row mtop">
            <span>工作负载健康率</span>
            <strong>{{ workloadHealthyRate }}%</strong>
          </div>
          <el-progress :percentage="workloadHealthyRate" :stroke-width="14" status="success" />
          <el-divider />
          <div class="health-row"><span>Service 数量</span><strong>{{ stats.serviceTotal }}</strong></div>
          <div class="health-row"><span>Ingress 数量</span><strong>{{ stats.ingressTotal }}</strong></div>
          <div class="health-row"><span>Ready Pod（估算）</span><strong>{{ stats.podReadyTotal }}</strong></div>
          <div class="health-row"><span>NotReady 节点</span><strong>{{ notReadyNodes }}</strong></div>
          <div class="health-row"><span>超时 Warning 事件</span><strong>{{ warningEventTimeout }}</strong></div>
          <div class="health-row"><span>待复检集群</span><strong>{{ clusterStaleCount }}</strong></div>
        </el-card>
      </el-col>
      <el-col :span="14">
        <el-card>
          <template #header>重点异常工作负载</template>
            <el-table :fit="true" :data="degradedRows" size="small" max-height="300" empty-text="暂无异常工作负载">
            <el-table-column prop="namespace" label="命名空间" min-width="130" />
            <el-table-column prop="name" label="名称" min-width="170" />
            <el-table-column prop="kind" label="类型" width="130" />
            <el-table-column label="副本/就绪" width="110">
              <template #default="{ row }">{{ row.ready || 0 }}/{{ row.replicas || 0 }}</template>
            </el-table-column>
            <el-table-column label="状态" width="100">
              <template #default="{ row }">
                <el-tag :type="workloadStatus(row).type">{{ workloadStatus(row).text }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" min-width="190">
              <template #default="{ row }">
                <el-button link type="primary" @click="openWorkload(row)">详情</el-button>
                <el-button link @click="goToWorkloadDetail(row)">页面</el-button>
                <el-button link type="success" @click="restartWorkload(row)">重启</el-button>
                <el-button
                  v-if="normalizeText(row.kind) !== 'daemonset'"
                  link
                  type="warning"
                  @click="scaleWorkload(row)"
                >
                  扩缩容
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <el-card class="integration-card">
      <template #header>
        <div class="integration-header">
          <div class="integration-title-wrap">
            <span>K8s 融合视图</span>
            <el-tag size="small" type="info" effect="plain">
              当前：{{ activePanelMeta.label }} · {{ activePanelMeta.count }}
            </el-tag>
          </div>
          <div class="integration-actions">
            <el-input
              v-model="panelKeyword"
              clearable
              size="small"
              class="panel-search"
              placeholder="筛选名称、命名空间、类型、状态..."
            />
            <el-button v-if="panelKeyword" size="small" @click="panelKeyword = ''">清空筛选</el-button>
            <el-button size="small" type="primary" plain @click="openCurrentPanel">进入完整页面</el-button>
          </div>
        </div>
      </template>

      <div class="panel-switch">
        <el-check-tag
          v-for="item in panelOptions"
          :key="item.name"
          :checked="activePanel === item.name"
          @change="activePanel = item.name"
        >
          {{ item.label }}
          <span class="panel-switch-count">{{ item.count }}</span>
        </el-check-tag>
      </div>

      <el-tabs v-model="activePanel" class="integration-tabs">
        <el-tab-pane label="集群" name="clusters">
          <el-table :fit="true" :data="filteredClusters" size="small" max-height="360" empty-text="暂无集群数据">
            <el-table-column label="名称" min-width="170">
              <template #default="{ row }">{{ row.display_name || row.name || '-' }}</template>
            </el-table-column>
            <el-table-column prop="api_server" label="API Server" min-width="220" show-overflow-tooltip />
            <el-table-column prop="version" label="版本" width="120" />
            <el-table-column prop="node_count" label="节点数" width="100" />
            <el-table-column label="状态" width="110">
              <template #default="{ row }">
                <el-tag :type="clusterStatus(row).type">{{ clusterStatus(row).text }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="检查时效" width="110">
              <template #default="{ row }">
                <el-tag :type="isClusterStale(row) ? 'warning' : 'success'">
                  {{ isClusterStale(row) ? '待复检' : '及时' }}
                </el-tag>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="工作负载" name="workloads">
          <el-table :fit="true" :data="filteredWorkloads" size="small" max-height="360" empty-text="暂无工作负载数据">
            <el-table-column prop="namespace" label="命名空间" min-width="130" />
            <el-table-column prop="name" label="名称" min-width="170" />
            <el-table-column prop="kind" label="类型" width="120" />
            <el-table-column label="副本/就绪" width="110">
              <template #default="{ row }">{{ row.ready || 0 }}/{{ row.replicas || 0 }}</template>
            </el-table-column>
            <el-table-column label="状态" width="100">
              <template #default="{ row }">
                <el-tag :type="workloadStatus(row).type">{{ workloadStatus(row).text }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="镜像" min-width="180">
              <template #default="{ row }">{{ (row.images || []).join(', ') || '-' }}</template>
            </el-table-column>
            <el-table-column label="操作" min-width="190">
              <template #default="{ row }">
                <el-button size="small" link type="primary" @click="openWorkload(row)">详情</el-button>
                <el-button size="small" link @click="goToWorkloadDetail(row)">页面</el-button>
                <el-button size="small" link type="success" @click="restartWorkload(row)">重启</el-button>
                <el-button
                  v-if="normalizeText(row.kind) !== 'daemonset'"
                  size="small"
                  link
                  type="warning"
                  @click="scaleWorkload(row)"
                >
                  扩缩容
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="Service/Ingress" name="serviceIngress">
          <el-row :gutter="12">
            <el-col :span="12">
              <el-table :fit="true" :data="filteredServices" size="small" max-height="340" empty-text="暂无 Service">
                <el-table-column prop="namespace" label="命名空间" min-width="110" />
                <el-table-column prop="name" label="Service" min-width="140" />
                <el-table-column prop="type" label="类型" width="100" />
                <el-table-column label="端口" min-width="130">
                  <template #default="{ row }">{{ (row.ports || []).join(', ') || '-' }}</template>
                </el-table-column>
              </el-table>
            </el-col>
            <el-col :span="12">
              <el-table :fit="true" :data="filteredIngresses" size="small" max-height="340" empty-text="暂无 Ingress">
                <el-table-column prop="namespace" label="命名空间" min-width="110" />
                <el-table-column prop="name" label="Ingress" min-width="140" />
                <el-table-column prop="class_name" label="Class" width="100" />
                <el-table-column label="Hosts" min-width="140">
                  <template #default="{ row }">{{ (row.hosts || []).join(', ') || '-' }}</template>
                </el-table-column>
              </el-table>
            </el-col>
          </el-row>
        </el-tab-pane>

        <el-tab-pane label="节点" name="nodes">
          <el-table :fit="true" :data="filteredNodes" size="small" max-height="360" empty-text="暂无节点数据">
            <el-table-column prop="name" label="节点" min-width="170" />
            <el-table-column prop="internal_ip" label="内网IP" min-width="130" />
            <el-table-column prop="os" label="系统" min-width="130" />
            <el-table-column prop="kubelet_version" label="Kubelet" width="120" />
            <el-table-column label="角色" min-width="130">
              <template #default="{ row }">{{ (row.roles || []).join(', ') || '-' }}</template>
            </el-table-column>
            <el-table-column label="状态" width="90">
              <template #default="{ row }">
                <el-tag :type="nodeReady(row) ? 'success' : 'warning'">{{ nodeReady(row) ? 'Ready' : 'NotReady' }}</el-tag>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="事件" name="events">
          <el-table :fit="true" :data="filteredEvents" size="small" max-height="360" empty-text="暂无事件数据">
            <el-table-column label="类型" width="90">
              <template #default="{ row }">
                <el-tag :type="eventTag(row.type)">{{ row.type || '-' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="reason" label="原因" min-width="130" />
            <el-table-column prop="message" label="消息" min-width="220" show-overflow-tooltip />
            <el-table-column label="对象" min-width="150">
              <template #default="{ row }">{{ row.involved_object || row.involved || '-' }}</template>
            </el-table-column>
            <el-table-column prop="count" label="次数" width="80" />
            <el-table-column label="最近时间" min-width="165">
              <template #default="{ row }">{{ formatTime(row.last_seen) }}</template>
            </el-table-column>
            <el-table-column label="持续时长" width="110">
              <template #default="{ row }">
                <el-tag :type="isWarningEventTimeout(row) ? 'warning' : 'success'">
                  {{ formatDuration(row.last_seen || row.first_seen || row.created_at) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="120">
              <template #default="{ row }">
                <el-button size="small" link type="primary" @click="openEventTarget(row)">定位</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <el-drawer
      v-model="workloadCockpitVisible"
      title="工作负载驾驶舱"
      size="68%"
      append-to-body
      destroy-on-close
    >
      <template #header>
        <div class="cockpit-header">
          <div>
            <div class="cockpit-title">
              {{ workloadCockpit?.kind || '-' }}/{{ workloadCockpit?.name || '-' }}
            </div>
            <div class="cockpit-subtitle">
              {{ workloadCockpit?.namespace || '-' }} · {{ clusterName }}
            </div>
          </div>
          <div class="cockpit-header-actions">
            <el-button size="small" @click="goToWorkloadDetail(workloadCockpit)">进入详情页</el-button>
            <el-button size="small" type="success" plain @click="restartWorkload(workloadCockpit)">重启</el-button>
            <el-button
              v-if="normalizeText(workloadCockpit?.kind) !== 'daemonset'"
              size="small"
              type="warning"
              plain
              @click="scaleWorkload(workloadCockpit)"
            >
              扩缩容
            </el-button>
          </div>
        </div>
      </template>

      <div v-loading="workloadCockpitLoading">
        <el-tabs v-model="workloadCockpitTab">
          <el-tab-pane label="概览" name="overview">
            <el-row :gutter="12" class="summary-row">
              <el-col :span="6"><el-card><div class="metric-title">副本</div><div class="metric-value">{{ workloadCockpit?.replicas || 0 }}</div></el-card></el-col>
              <el-col :span="6"><el-card><div class="metric-title">就绪</div><div class="metric-value">{{ workloadCockpit?.ready || 0 }}</div></el-card></el-col>
              <el-col :span="6"><el-card><div class="metric-title">状态</div><div class="metric-value">{{ workloadStatus(workloadCockpit).text }}</div></el-card></el-col>
              <el-col :span="6"><el-card><div class="metric-title">关联域名</div><div class="metric-value">{{ relatedIngressHosts.length }}</div></el-card></el-col>
            </el-row>

            <el-descriptions :column="2" border>
              <el-descriptions-item label="工作负载">{{ workloadCockpit?.kind || '-' }}/{{ workloadCockpit?.name || '-' }}</el-descriptions-item>
              <el-descriptions-item label="命名空间">{{ workloadCockpit?.namespace || '-' }}</el-descriptions-item>
              <el-descriptions-item label="镜像">{{ (workloadCockpit?.images || []).join(', ') || '-' }}</el-descriptions-item>
              <el-descriptions-item label="最近更新时间">{{ formatTime(workloadCockpit?.updated_at || workloadCockpit?.created_at) }}</el-descriptions-item>
              <el-descriptions-item label="关联域名">
                <span v-if="!relatedIngressHosts.length" class="muted-inline">-</span>
                <el-link
                  v-for="host in relatedIngressHosts"
                  :key="host"
                  :href="`http://${host}`"
                  type="primary"
                  target="_blank"
                  class="mr-2"
                >
                  {{ host }}
                </el-link>
              </el-descriptions-item>
              <el-descriptions-item label="关联 Ingress">{{ relatedIngressNames.join(', ') || '-' }}</el-descriptions-item>
            </el-descriptions>
          </el-tab-pane>

          <el-tab-pane :label="`Pods (${workloadCockpitPods.length})`" name="pods">
            <el-table :fit="true" :data="workloadCockpitPods" size="small" max-height="380" empty-text="暂无关联 Pod">
              <el-table-column prop="name" label="Pod" min-width="180" />
              <el-table-column prop="status" label="状态" width="110" />
              <el-table-column prop="node" label="节点" min-width="150" />
              <el-table-column prop="ip" label="IP" min-width="130" />
              <el-table-column prop="restarts" label="重启" width="70" />
              <el-table-column label="创建时间" min-width="160">
                <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
              </el-table-column>
              <el-table-column label="操作" width="110">
                <template #default="{ row }">
                  <el-button
                    size="small"
                    link
                    @click="go('/k8s/pods')"
                  >
                    打开 Pods
                  </el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>

          <el-tab-pane :label="`事件 (${workloadCockpitEvents.length})`" name="events">
            <el-table :fit="true" :data="workloadCockpitEvents" size="small" max-height="380" empty-text="暂无关联事件">
              <el-table-column prop="type" label="类型" width="90">
                <template #default="{ row }">
                  <el-tag :type="eventTag(row.type)">{{ row.type || '-' }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="reason" label="原因" min-width="130" />
              <el-table-column prop="message" label="消息" min-width="220" show-overflow-tooltip />
              <el-table-column prop="count" label="次数" width="80" />
              <el-table-column label="最近时间" min-width="165">
                <template #default="{ row }">{{ formatTime(row.last_seen || row.created_at) }}</template>
              </el-table-column>
            </el-table>
          </el-tab-pane>

          <el-tab-pane label="快捷操作" name="actions">
            <div class="cockpit-action-grid">
              <el-card shadow="never">
                <div class="action-title">工作负载操作</div>
                <div class="inline-actions">
                  <el-button type="primary" plain @click="goToWorkloadDetail(workloadCockpit)">进入详情页</el-button>
                  <el-button type="success" plain @click="restartWorkload(workloadCockpit)">滚动重启</el-button>
                  <el-button
                    v-if="normalizeText(workloadCockpit?.kind) !== 'daemonset'"
                    type="warning"
                    plain
                    @click="scaleWorkload(workloadCockpit)"
                  >
                    扩缩容
                  </el-button>
                </div>
              </el-card>
              <el-card shadow="never">
                <div class="action-title">关联排障入口</div>
                <div class="inline-actions">
                  <el-button plain @click="go('/k8s/deployments')">Deployment 运维台</el-button>
                  <el-button plain @click="go('/k8s/pods')">Pod 列表</el-button>
                  <el-button plain @click="go('/k8s/events')">事件诊断</el-button>
                  <el-button plain @click="go('/k8s/services')">服务与 Ingress</el-button>
                </div>
              </el-card>
            </div>
          </el-tab-pane>
        </el-tabs>
      </div>
    </el-drawer>
  </el-card>
</template>

<script setup>
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getErrorMessage, isCancelError } from '@/utils/error'

const router = useRouter()
const loading = ref(false)
const clusters = ref([])
const namespaces = ref([])
const workloads = ref([])
const services = ref([])
const ingresses = ref([])
const nodes = ref([])
const events = ref([])
const clusterId = ref('')
const activePanel = ref('workloads')
const panelKeyword = ref('')
const nowTs = ref(Date.now())
const workloadCockpitVisible = ref(false)
const workloadCockpitLoading = ref(false)
const workloadCockpitTab = ref('overview')
const workloadCockpit = ref(null)
const workloadCockpitPods = ref([])
let minuteTicker = null

const stats = reactive({
  clusterTotal: 0,
  clusterOnline: 0,
  clusterMaintenance: 0,
  namespaceTotal: 0,
  workloadTotal: 0,
  healthyWorkloads: 0,
  degradedWorkloads: 0,
  serviceTotal: 0,
  ingressTotal: 0,
  podReadyTotal: 0,
  nodeTotal: 0,
  warningEvents: 0
})


const panelRouteMap = {
  clusters: '/k8s/clusters',
  workloads: '/k8s/workloads',
  serviceIngress: '/k8s/services',
  nodes: '/k8s/nodes',
  events: '/k8s/events'
}


const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })
const go = (path) => router.push(path)

const normalizeText = (value) => String(value ?? '').trim().toLowerCase()

const isOnlineCluster = (status) => {
  const normalized = normalizeText(status)
  return normalized === 'online' || normalized === 'connected' || normalized === 'normal' || Number(status) === 1 || status === true
}

const isMaintenanceCluster = (status) => {
  const normalized = normalizeText(status)
  return normalized === 'maintenance' || normalized === 'maintain' || Number(status) === 2
}

const clusterStatus = (row) => {
  if (!row) return { text: '-', type: 'info' }
  if (isOnlineCluster(row.status)) return { text: '在线', type: 'success' }
  if (isMaintenanceCluster(row.status)) return { text: '维护', type: 'warning' }
  return { text: '异常', type: 'danger' }
}

const nodeReady = (row) => {
  const v = normalizeText(row?.status)
  return v === 'ready' || v === 'true' || v === 'online' || v === 'running'
}

const workloadStatus = (row) => {
  if (!row) return { text: '-', type: 'info' }
  const replicas = Number(row.replicas || 0)
  const ready = Number(row.ready || 0)
  if (replicas === 0) return { text: 'Scaled 0', type: 'info' }
  if (ready === 0 && replicas > 0) return { text: 'Degraded', type: 'danger' }
  if (ready < replicas) return { text: 'Updating', type: 'warning' }
  return { text: 'Healthy', type: 'success' }
}

const eventTag = (type) => {
  const v = normalizeText(type)
  if (v === 'warning') return 'danger'
  if (v === 'normal') return 'success'
  return 'info'
}

const clusterOnlineRate = computed(() => {
  if (!stats.clusterTotal) return 0
  return Math.round((stats.clusterOnline / stats.clusterTotal) * 100)
})

const workloadHealthyRate = computed(() => {
  if (!stats.workloadTotal) return 0
  return Math.round((stats.healthyWorkloads / stats.workloadTotal) * 100)
})

const clusterName = computed(() => {
  const current = clusters.value.find((item) => item.id === clusterId.value)
  return current?.display_name || current?.name || '-'
})

const parseTimestamp = (value) => {
  if (!value) return null
  const ts = new Date(value).getTime()
  return Number.isNaN(ts) ? null : ts
}

const elapsedMinutes = (value) => {
  const ts = parseTimestamp(value)
  if (!ts) return 0
  const diff = Math.floor((nowTs.value - ts) / 60000)
  return diff > 0 ? diff : 0
}

const formatDuration = (value) => {
  const minutes = elapsedMinutes(value)
  if (minutes < 60) return `${minutes}m`
  const hours = Math.floor(minutes / 60)
  const remain = minutes % 60
  if (hours < 24) return `${hours}h${remain}m`
  const days = Math.floor(hours / 24)
  return `${days}d${hours % 24}h`
}

const clusterFreshnessTs = (row) =>
  row?.last_sync_at ||
  row?.last_seen_at ||
  row?.last_heartbeat_at ||
  row?.last_checked_at ||
  row?.updated_at

const isClusterStale = (row) => {
  if (!isOnlineCluster(row?.status)) return false
  return elapsedMinutes(clusterFreshnessTs(row)) >= 15
}

const isWarningEventTimeout = (row) => {
  if (normalizeText(row?.type) !== 'warning') return false
  return elapsedMinutes(row?.last_seen || row?.first_seen || row?.created_at) >= 30
}

const notReadyNodes = computed(() => nodes.value.filter((item) => !nodeReady(item)).length)
const warningEventTimeout = computed(() => events.value.filter((item) => isWarningEventTimeout(item)).length)
const clusterStaleCount = computed(() => clusters.value.filter((item) => isClusterStale(item)).length)
const pendingBacklog = computed(
  () => stats.degradedWorkloads + notReadyNodes.value + warningEventTimeout.value + clusterStaleCount.value
)

const degradedRows = computed(() =>
  workloads.value
    .filter((item) => workloadStatus(item).text !== 'Healthy')
    .sort((a, b) => Number(b.replicas || 0) - Number(a.replicas || 0))
    .slice(0, 10)
)

const filterRows = (rows, fields) => {
  const keyword = normalizeText(panelKeyword.value)
  const base = Array.isArray(rows) ? rows : []
  if (!keyword) return base.slice(0, 30)
  return base.filter((row) => fields.some((field) => normalizeText(field(row)).includes(keyword))).slice(0, 30)
}

const filteredClusters = computed(() =>
  filterRows(clusters.value, [(row) => row.display_name || row.name, (row) => row.api_server, (row) => row.version, (row) => clusterStatus(row).text])
)

const filteredWorkloads = computed(() =>
  filterRows(workloads.value, [(row) => row.namespace, (row) => row.name, (row) => row.kind, (row) => (row.images || []).join(','), (row) => workloadStatus(row).text])
)

const filteredServices = computed(() =>
  filterRows(services.value, [(row) => row.namespace, (row) => row.name, (row) => row.type, (row) => (row.ports || []).join(','), (row) => row.cluster_ip])
)

const filteredIngresses = computed(() =>
  filterRows(ingresses.value, [(row) => row.namespace, (row) => row.name, (row) => row.class_name, (row) => (row.hosts || []).join(',')])
)

const filteredNodes = computed(() =>
  filterRows(nodes.value, [(row) => row.name, (row) => row.internal_ip, (row) => row.os, (row) => row.kubelet_version, (row) => (row.roles || []).join(','), (row) => row.status])
)

const filteredEvents = computed(() =>
  filterRows(events.value, [(row) => row.type, (row) => row.reason, (row) => row.message, (row) => row.involved_object || row.involved, (row) => row.namespace])
)

const panelOptions = computed(() => [
  { name: 'clusters', label: '集群', count: clusters.value.length },
  { name: 'workloads', label: '工作负载', count: workloads.value.length },
  { name: 'serviceIngress', label: '服务与Ingress', count: services.value.length + ingresses.value.length },
  { name: 'nodes', label: '节点', count: nodes.value.length },
  { name: 'events', label: '事件', count: events.value.length }
])

const activePanelMeta = computed(
  () => panelOptions.value.find((item) => item.name === activePanel.value) || panelOptions.value[0] || { label: '-', count: 0 }
)

const eventMatchesWorkload = (event, row) => {
  if (!event || !row) return false
  if (normalizeText(event.namespace) !== normalizeText(row.namespace)) return false
  const involved = normalizeText(event.involved_object || event.involved)
  const expect = `${normalizeText(row.kind)}/${normalizeText(row.name)}`
  if (involved === expect) return true
  if (!involved) return false
  if (normalizeText(row.kind) === 'deployment' && involved.startsWith('replicaset/')) {
    const rsName = involved.split('/')[1] || ''
    return rsName.startsWith(`${normalizeText(row.name)}-`)
  }
  return false
}

const workloadCockpitEvents = computed(() =>
  events.value
    .filter((item) => eventMatchesWorkload(item, workloadCockpit.value))
    .sort(
      (a, b) =>
        (parseTimestamp(b?.last_seen || b?.created_at) || 0) -
        (parseTimestamp(a?.last_seen || a?.created_at) || 0)
    )
)

const relatedIngresses = computed(() => {
  if (!workloadCockpit.value) return []
  const ns = normalizeText(workloadCockpit.value.namespace)
  const wkName = normalizeText(workloadCockpit.value.name)
  return ingresses.value.filter((item) => {
    if (normalizeText(item.namespace) !== ns) return false
    const nameMatched = normalizeText(item.name).includes(wkName)
    if (nameMatched) return true
    try {
      return JSON.stringify(item).toLowerCase().includes(wkName)
    } catch (err) {
      return false
    }
  })
})

const relatedIngressHosts = computed(() =>
  Array.from(
    new Set(
      relatedIngresses.value
        .flatMap((item) => (Array.isArray(item.hosts) ? item.hosts : []))
        .filter((item) => String(item || '').trim() !== '')
    )
  )
)

const relatedIngressNames = computed(() =>
  relatedIngresses.value.map((item) => item.name).filter((item) => String(item || '').trim() !== '')
)

const podOwnedByWorkload = (pod, row) => {
  if (!pod || !row) return false
  if (normalizeText(pod.namespace) !== normalizeText(row.namespace)) return false
  const ownerKind = normalizeText(pod.owner_kind)
  const ownerName = normalizeText(pod.owner_name)
  const kind = normalizeText(row.kind)
  const name = normalizeText(row.name)
  if (ownerKind === kind && ownerName === name) return true
  if (kind === 'deployment' && ownerKind === 'replicaset') return ownerName.startsWith(`${name}-`)
  if (kind === 'cronjob' && ownerKind === 'job') return ownerName.startsWith(`${name}-`)
  const labels = pod.labels || {}
  const appLabel = normalizeText(labels.app || labels['app.kubernetes.io/name'])
  return appLabel !== '' && appLabel === name
}

const loadWorkloadPods = async (row) => {
  if (!row || !clusterId.value || !row.namespace) {
    workloadCockpitPods.value = []
    return
  }
  workloadCockpitLoading.value = true
  try {
    const list = await fetchList(`/api/v1/k8s/clusters/${clusterId.value}/namespaces/${row.namespace}/pods`)
    workloadCockpitPods.value = list.filter((item) => podOwnedByWorkload(item, row))
  } catch (err) {
    workloadCockpitPods.value = []
    ElMessage.warning(getErrorMessage(err, '获取关联 Pod 失败'))
  } finally {
    workloadCockpitLoading.value = false
  }
}

const fetchList = async (url, params = undefined) => {
  const res = await axios.get(url, { headers: authHeaders(), params })
  return Array.isArray(res.data?.data) ? res.data.data : []
}

const fetchClusters = async () => {
  const list = await fetchList('/api/v1/k8s/clusters')
  clusters.value = list
  stats.clusterTotal = list.length
  stats.clusterOnline = list.filter((item) => isOnlineCluster(item.status)).length
  stats.clusterMaintenance = list.filter((item) => isMaintenanceCluster(item.status)).length
  if (!clusterId.value && list.length) {
    clusterId.value = list[0].id
  }
}

const toSafeList = (result) => (result.status === 'fulfilled' && Array.isArray(result.value) ? result.value : [])

const refreshClusterData = async () => {
  if (!clusterId.value) {
    namespaces.value = []
    workloads.value = []
    services.value = []
    ingresses.value = []
    nodes.value = []
    events.value = []
    stats.namespaceTotal = 0
    stats.workloadTotal = 0
    stats.healthyWorkloads = 0
    stats.degradedWorkloads = 0
    stats.serviceTotal = 0
    stats.ingressTotal = 0
    stats.podReadyTotal = 0
    stats.nodeTotal = 0
    stats.warningEvents = 0
    return
  }

  const settled = await Promise.allSettled([
    fetchList(`/api/v1/k8s/clusters/${clusterId.value}/namespaces`),
    fetchList(`/api/v1/k8s/clusters/${clusterId.value}/workloads`),
    fetchList(`/api/v1/k8s/clusters/${clusterId.value}/services`),
    fetchList(`/api/v1/k8s/clusters/${clusterId.value}/ingresses`),
    fetchList(`/api/v1/k8s/clusters/${clusterId.value}/nodes`),
    fetchList(`/api/v1/k8s/clusters/${clusterId.value}/events`)
  ])

  const [nsList, workloadList, serviceList, ingressList, nodeList, eventList] = settled.map(toSafeList)

  namespaces.value = nsList
  workloads.value = workloadList
  services.value = serviceList
  ingresses.value = ingressList
  nodes.value = nodeList
  events.value = eventList

  stats.namespaceTotal = nsList.length
  stats.workloadTotal = workloadList.length
  stats.healthyWorkloads = workloadList.filter((item) => workloadStatus(item).text === 'Healthy').length
  stats.degradedWorkloads = workloadList.filter((item) => workloadStatus(item).text !== 'Healthy').length
  stats.serviceTotal = serviceList.length
  stats.ingressTotal = ingressList.length
  stats.podReadyTotal = workloadList.reduce((sum, item) => sum + Number(item.ready || 0), 0)
  stats.nodeTotal = nodeList.length
  stats.warningEvents = eventList.filter((item) => normalizeText(item.type) === 'warning').length

  const failedCount = settled.filter((item) => item.status === 'rejected').length
  if (failedCount > 0) {
    ElMessage.warning(`部分 K8s 数据源加载失败（${failedCount}项），已展示可用数据`)
  }
}

const goToWorkloadDetail = (row) => {
  if (!row) return
  router.push({
    path: '/k8s/workloads/detail',
    query: {
      clusterId: clusterId.value,
      namespace: row.namespace,
      kind: row.kind,
      name: row.name
    }
  })
}

const openWorkload = async (row) => {
  if (!row) return
  workloadCockpit.value = { ...row }
  workloadCockpitVisible.value = true
  workloadCockpitTab.value = 'overview'
  await loadWorkloadPods(row)
}

const openCurrentPanel = () => {
  go(panelRouteMap[activePanel.value] || '/k8s/clusters')
}

const scaleWorkload = async (row) => {
  if (!row || normalizeText(row.kind) === 'daemonset') return
  try {
    const { value } = await ElMessageBox.prompt('输入目标副本数', `扩缩容 ${row.kind}/${row.name}`, {
      inputValue: String(row.replicas ?? 1),
      inputPattern: /^[0-9]+$/,
      inputErrorMessage: '请输入非负整数'
    })
    await axios.put(
      `/api/v1/k8s/clusters/${clusterId.value}/namespaces/${row.namespace}/workloads/${row.kind}/${row.name}/scale`,
      { replicas: Number(value) },
      { headers: authHeaders() }
    )
    ElMessage.success('扩缩容已提交')
    await refreshClusterData()
  } catch (err) {
    if (!isCancelError(err)) {
      ElMessage.error(getErrorMessage(err, '扩缩容失败'))
    }
  }
}

const restartWorkload = async (row) => {
  if (!row) return
  try {
    await ElMessageBox.confirm(`确认重启 ${row.kind}/${row.name} 吗？`, '提示', { type: 'warning' })
    await axios.post(
      `/api/v1/k8s/clusters/${clusterId.value}/namespaces/${row.namespace}/workloads/${row.kind}/${row.name}/restart`,
      {},
      { headers: authHeaders() }
    )
    ElMessage.success('已触发滚动重启')
    await refreshClusterData()
  } catch (err) {
    if (!isCancelError(err)) {
      ElMessage.error(getErrorMessage(err, '重启失败'))
    }
  }
}

const openEventTarget = (row) => {
  const involved = String(row?.involved_object || row?.involved || '').trim()
  const [kind, name] = involved.includes('/') ? involved.split('/') : ['', '']
  if (!kind || !name) {
    go('/k8s/events')
    return
  }
  const allowedKinds = ['deployment', 'statefulset', 'daemonset']
  if (!allowedKinds.includes(normalizeText(kind))) {
    go('/k8s/events')
    return
  }
  const matched = workloads.value.find(
    (w) =>
      normalizeText(w.namespace) === normalizeText(row?.namespace) &&
      normalizeText(w.kind) === normalizeText(kind) &&
      normalizeText(w.name) === normalizeText(name)
  )
  if (matched) {
    openWorkload(matched)
    return
  }
  router.push({
    path: '/k8s/workloads/detail',
    query: {
      clusterId: clusterId.value,
      namespace: row?.namespace || '',
      kind,
      name
    }
  })
}

const formatTime = (value) => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return String(value)
  return date.toLocaleString()
}

const refreshAll = async () => {
  loading.value = true
  try {
    await fetchClusters()
    await refreshClusterData()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '加载容器平台总览失败'))
  } finally {
    loading.value = false
  }
}

onMounted(refreshAll)
onMounted(() => {
  minuteTicker = window.setInterval(() => {
    nowTs.value = Date.now()
  }, 60 * 1000)
})
onUnmounted(() => {
  if (minuteTicker) {
    window.clearInterval(minuteTicker)
    minuteTicker = null
  }
})
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 12px; gap: 12px; }
.page-desc { color: var(--muted-text); margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; flex-wrap: wrap; }
.summary-row { margin-bottom: 12px; }
.summary-row :deep(.el-card) { margin-bottom: 8px; }
.metric-title { color: var(--muted-text); font-size: 12px; }
.metric-value { font-size: 20px; font-weight: 600; margin-top: 6px; color: var(--el-text-color-primary); }
.metric-value.ok { color: #67c23a; }
.metric-value.warning { color: #e6a23c; }
.metric-value.danger { color: #f56c6c; }
.metric-sub { margin-top: 4px; color: var(--muted-text); font-size: 12px; }
.health-row { display: flex; justify-content: space-between; align-items: center; margin-bottom: 6px; }
.health-row strong { font-size: 15px; }
.mtop { margin-top: 12px; }
.w-52 { width: 220px; }

.integration-card {
  margin-top: 12px;
}

.integration-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.integration-title-wrap {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.integration-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.panel-search {
  width: 260px;
}

.panel-switch {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 10px;
}

.panel-switch-count {
  margin-left: 6px;
  opacity: 0.8;
}

.integration-tabs :deep(.el-tabs__header) { display: none; }

.cockpit-header {
  width: 100%;
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
}

.cockpit-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.cockpit-subtitle {
  margin-top: 4px;
  color: var(--muted-text);
  font-size: 12px;
}

.cockpit-header-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.cockpit-action-grid {
  display: grid;
  gap: 10px;
}

.action-title {
  font-weight: 600;
  margin-bottom: 10px;
}

.inline-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.muted-inline {
  color: var(--muted-text);
  font-size: 12px;
}

@media (max-width: 1100px) {
  .page-actions {
    width: 100%;
  }

  .w-52 {
    width: 100%;
  }

  .integration-header {
    align-items: flex-start;
    flex-direction: column;
  }

  .integration-actions {
    width: 100%;
    flex-wrap: wrap;
  }

  .panel-search {
    width: 100%;
  }

  .cockpit-header {
    flex-direction: column;
  }
}
</style>
