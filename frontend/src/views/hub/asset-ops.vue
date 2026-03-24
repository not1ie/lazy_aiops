<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>资产作战台</h2>
        <p class="page-desc">聚合 CMDB、网络设备、防火墙、堡垒机会话与风控事件，按运维处置流程联动。</p>
      </div>
      <div class="page-actions">
        <el-button :loading="syncingNetworkFromFirewall" icon="RefreshRight" @click="syncNetworkDevicesFromFirewalls">同步防火墙资产</el-button>
        <el-button :loading="loading" icon="Refresh" @click="refreshAll">刷新</el-button>
      </div>
    </div>

    <div class="workbench-toolbar">
      <div class="workbench-toolbar-left">
        <span class="workbench-toolbar-label">场景工作台</span>
        <el-check-tag checked @click="go('/asset/ops')">资产作战台</el-check-tag>
        <el-check-tag :checked="false" @click="go('/asset/overview')">资产总览</el-check-tag>
        <el-check-tag :checked="false" @click="go('/k8s/overview')">容器总览</el-check-tag>
        <el-check-tag :checked="false" @click="go('/domain/center')">域名中心</el-check-tag>
      </div>
      <div class="workbench-toolbar-right">
        <el-tag type="warning" effect="light">待处置 {{ stats.pendingBacklog }}</el-tag>
        <el-tag type="danger" effect="light">高危 {{ riskCriticalCount }}</el-tag>
        <el-tag type="info" effect="light">待审批 {{ stats.jumpPending }}</el-tag>
        <el-button link type="primary" @click="focusAssetPanel('risk')">风险处置</el-button>
        <el-button link type="warning" @click="focusAssetPanel('pending')">审批队列</el-button>
      </div>
    </div>

    <div class="workbench-layout">
      <aside class="asset-tree-panel">
        <div class="asset-tree-head">
          <div class="asset-tree-title">资产分组</div>
          <el-tag size="small" type="info" effect="plain">{{ selectedTreeNodeMeta.label || '全部资产' }}</el-tag>
        </div>
        <el-input
          v-model="treeKeyword"
          clearable
          size="small"
          placeholder="搜索分组或焦点"
          class="asset-tree-search"
        />
        <el-scrollbar class="asset-tree-scroll">
          <el-tree
            ref="assetTreeRef"
            :data="assetTreeData"
            node-key="id"
            :current-node-key="selectedTreeNode"
            :expand-on-click-node="false"
            :props="{ label: 'label', children: 'children' }"
            :default-expanded-keys="['all', 'group-root', 'type-root', 'focus-root']"
            :filter-node-method="filterAssetTreeNode"
            highlight-current
            @node-click="handleAssetTreeNodeClick"
          >
            <template #default="{ data }">
              <div class="asset-tree-node">
                <span class="asset-tree-node-label">{{ data.label }}</span>
                <el-tag
                  v-if="typeof data.count === 'number'"
                  size="small"
                  effect="plain"
                  type="info"
                >
                  {{ data.count }}
                </el-tag>
              </div>
            </template>
          </el-tree>
        </el-scrollbar>
      </aside>

      <div class="workbench-main">
    <el-row :gutter="12" class="summary-row">
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">主机总数</div><div class="metric-value">{{ stats.hostTotal }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">离线主机</div><div class="metric-value danger">{{ stats.hostOffline }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">网络设备</div><div class="metric-value">{{ stats.networkTotal }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">离线网络设备</div><div class="metric-value warning">{{ stats.networkOffline }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">防火墙告警</div><div class="metric-value danger">{{ stats.firewallAlert }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">待审批会话</div><div class="metric-value warning">{{ stats.jumpPending }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">活跃会话</div><div class="metric-value ok">{{ stats.jumpActive }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card>
          <div class="metric-title">待处置积压</div>
          <div class="metric-value warning">{{ stats.pendingBacklog }}</div>
          <div class="metric-sub">高危风控 {{ stats.riskCritical }}</div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="12">
      <el-col :span="10">
        <el-card>
          <template #header>作战健康</template>
          <div class="health-row"><span>主机在线率</span><strong>{{ hostOnlineRate }}%</strong></div>
          <el-progress :percentage="hostOnlineRate" :stroke-width="14" />
          <div class="health-row mtop"><span>网络设备在线率</span><strong>{{ networkOnlineRate }}%</strong></div>
          <el-progress :percentage="networkOnlineRate" :stroke-width="14" status="success" />
          <el-divider />
          <div class="health-row"><span>堡垒机资产</span><strong>{{ jumpAssets.length }}</strong></div>
          <div class="health-row"><span>待复检对象</span><strong>{{ stats.recheckPending }}</strong></div>
          <div class="health-row"><span>超时待审批会话</span><strong>{{ stats.pendingApprovalTimeout }}</strong></div>
          <div class="health-row"><span>风控阻断命令(7天)</span><strong>{{ commandStats.commands_blocked || 0 }}</strong></div>
          <div class="health-row"><span>风控命令总量(7天)</span><strong>{{ commandStats.commands_window || 0 }}</strong></div>
        </el-card>

        <el-card class="mt-12">
          <template #header>待审批会话</template>
          <el-table :fit="true" :data="pendingSessions" size="small" max-height="260" empty-text="暂无待审批会话">
            <el-table-column prop="session_no" label="会话号" min-width="150" />
            <el-table-column prop="asset_name" label="资产" min-width="120" />
            <el-table-column prop="username" label="用户" width="100" />
            <el-table-column label="开始时间" min-width="150">
              <template #default="{ row }">{{ formatTime(row.started_at) }}</template>
            </el-table-column>
            <el-table-column label="等待时长" width="110">
              <template #default="{ row }">
                <el-tag :type="isPendingSessionStale(row) ? 'warning' : 'success'">
                  {{ formatPendingWait(row.started_at) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" min-width="180">
              <template #default="{ row }">
                <div class="inline-actions">
                  <el-button link type="primary" @click="approveJumpSession(row)">通过</el-button>
                  <el-button link type="warning" @click="rejectJumpSession(row)">拒绝</el-button>
                  <el-button link @click="openAssetDetail('pendingSession', row)">详情</el-button>
                </div>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>

      <el-col :span="14">
        <el-card>
          <BatchActionBar
            title="高风险操作事件"
            :tags="riskHeaderTags"
            :actions="riskHeaderActions"
            @action="handleRiskHeaderAction"
          />
          <el-table
            ref="riskTableRef"
            :fit="true"
            :data="riskEvents"
            size="small"
            max-height="470"
            empty-text="暂无风险事件"
            @selection-change="onRiskSelectionChange"
          >
            <el-table-column type="selection" width="46" />
            <el-table-column prop="asset_name" label="资产" min-width="120" />
            <el-table-column prop="username" label="用户" width="100" />
            <el-table-column prop="severity" label="级别" width="90">
              <template #default="{ row }">
                <el-tag :type="severityTag(row.severity)">{{ String(row.severity || '').toUpperCase() || '-' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="event_type" label="类型" width="90" />
            <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
            <el-table-column label="时间" min-width="150">
              <template #default="{ row }">{{ formatTime(row.fired_at) }}</template>
            </el-table-column>
            <el-table-column label="操作" min-width="150" fixed="right">
              <template #default="{ row }">
                <div class="inline-actions">
                  <el-button v-if="row.session_id" link type="danger" @click="disconnectRiskSession(row)">断开会话</el-button>
                  <el-button link type="primary" @click="openAssetDetail('riskEvent', row)">详情</el-button>
                </div>
              </template>
            </el-table-column>
          </el-table>
          <QuickGroupTags :groups="riskEventGroups" default-type="warning" @select="batchDisconnectRiskGroup" />
        </el-card>
      </el-col>
    </el-row>

    <el-card class="integration-card">
      <template #header>
        <div class="integration-header">
          <div class="integration-title-wrap">
            <span>资产作战融合视图</span>
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
              placeholder="筛选会话号、资产、用户、IP、风险..."
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
        <el-tab-pane label="离线资产" name="offline">
          <el-table :fit="true" :data="filteredOfflineAssets" size="small" max-height="360" empty-text="资产健康良好">
            <el-table-column prop="type" label="类型" width="100" />
            <el-table-column prop="name" label="名称" min-width="140" />
            <el-table-column prop="address" label="地址" min-width="140" />
            <el-table-column label="状态" width="110">
              <template #default="{ row }">
                <el-tag :type="row.level === 'danger' ? 'danger' : 'warning'">{{ row.statusText }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="快速处置" min-width="210">
              <template #default="{ row }">
                <div class="inline-actions">
                  <el-button link type="primary" @click="offlineAction(row)">{{ row.actionLabel || '诊断' }}</el-button>
                  <el-button link @click="openOfflineDetail(row)">详情</el-button>
                  <el-button link @click="go(row.path)">管理页</el-button>
                </div>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="待审批会话" name="pending">
          <el-table :fit="true" :data="filteredPendingSessions" size="small" max-height="360" empty-text="暂无待审批会话">
            <el-table-column prop="session_no" label="会话号" min-width="150" />
            <el-table-column prop="asset_name" label="资产" min-width="140" />
            <el-table-column prop="username" label="用户" width="100" />
            <el-table-column prop="protocol" label="协议" width="90" />
            <el-table-column label="开始时间" min-width="150">
              <template #default="{ row }">{{ formatTime(row.started_at) }}</template>
            </el-table-column>
            <el-table-column label="等待时长" width="110">
              <template #default="{ row }">
                <el-tag :type="isPendingSessionStale(row) ? 'warning' : 'success'">
                  {{ formatPendingWait(row.started_at) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" min-width="210" fixed="right">
              <template #default="{ row }">
                <div class="inline-actions">
                  <el-button link type="primary" @click="approveJumpSession(row)">通过</el-button>
                  <el-button link type="warning" @click="rejectJumpSession(row)">拒绝</el-button>
                  <el-button link @click="openAssetDetail('pendingSession', row)">详情</el-button>
                </div>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="活跃会话" name="active">
          <el-table :fit="true" :data="filteredActiveSessions" size="small" max-height="360" empty-text="暂无活跃会话">
            <el-table-column prop="session_no" label="会话号" min-width="150" />
            <el-table-column prop="asset_name" label="资产" min-width="140" />
            <el-table-column prop="username" label="用户" width="100" />
            <el-table-column prop="command_count" label="命令数" width="90" />
            <el-table-column label="最后命令" min-width="150">
              <template #default="{ row }">{{ formatTime(row.last_command_at) }}</template>
            </el-table-column>
            <el-table-column label="开始时间" min-width="150">
              <template #default="{ row }">{{ formatTime(row.started_at) }}</template>
            </el-table-column>
            <el-table-column label="操作" min-width="170" fixed="right">
              <template #default="{ row }">
                <div class="inline-actions">
                  <el-button link type="primary" @click="connectJumpSession(row)">连接</el-button>
                  <el-button link type="danger" @click="disconnectJumpSession(row)">断开</el-button>
                  <el-button link @click="openAssetDetail('activeSession', row)">详情</el-button>
                </div>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="风险事件" name="risk">
          <el-table :fit="true" :data="filteredRiskEvents" size="small" max-height="360" empty-text="暂无风险事件">
            <el-table-column prop="asset_name" label="资产" min-width="140" />
            <el-table-column prop="username" label="用户" width="100" />
            <el-table-column label="级别" width="90">
              <template #default="{ row }">
                <el-tag :type="severityTag(row.severity)">{{ String(row.severity || '').toUpperCase() || '-' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="rule_name" label="命中规则" min-width="160" />
            <el-table-column prop="description" label="描述" min-width="180" show-overflow-tooltip />
            <el-table-column label="时间" min-width="150">
              <template #default="{ row }">{{ formatTime(row.fired_at) }}</template>
            </el-table-column>
            <el-table-column label="操作" min-width="180" fixed="right">
              <template #default="{ row }">
                <div class="inline-actions">
                  <el-button v-if="row.session_id" link type="danger" @click="disconnectRiskSession(row)">断开会话</el-button>
                  <el-button link @click="openAssetDetail('riskEvent', row)">详情</el-button>
                </div>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="防火墙" name="firewall">
          <el-table :fit="true" :data="filteredFirewalls" size="small" max-height="360" empty-text="暂无防火墙数据">
            <el-table-column prop="name" label="名称" min-width="140" />
            <el-table-column prop="vendor" label="厂商" min-width="110" />
            <el-table-column prop="ip" label="IP" min-width="140" />
            <el-table-column label="状态" width="110">
              <template #default="{ row }">
                <el-tag :type="firewallStatus(row.status).type">{{ firewallStatus(row.status).text }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="最近检查" min-width="160">
              <template #default="{ row }">{{ formatTime(row.last_check_at) }}</template>
            </el-table-column>
            <el-table-column label="检查时效" width="110">
              <template #default="{ row }">
                <el-tag :type="isCheckStale(row.last_check_at) ? 'warning' : 'success'">
                  {{ isCheckStale(row.last_check_at) ? '待复检' : '及时' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" min-width="180" fixed="right">
              <template #default="{ row }">
                <div class="inline-actions">
                  <el-button link type="primary" @click="collectFirewall(row)">采集</el-button>
                  <el-button link @click="testFirewallSNMP(row)">SNMP测试</el-button>
                  <el-button link @click="openAssetDetail('firewall', row)">详情</el-button>
                </div>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="堡垒机资产" name="jumpAssets">
          <el-table :fit="true" :data="filteredJumpAssets" size="small" max-height="360" empty-text="暂无堡垒机资产数据">
            <el-table-column prop="name" label="名称" min-width="140" />
            <el-table-column prop="asset_type" label="类型" width="110" />
            <el-table-column prop="protocol" label="协议" width="90" />
            <el-table-column label="地址" min-width="170">
              <template #default="{ row }">{{ row.address }}:{{ row.port }}</template>
            </el-table-column>
            <el-table-column prop="source" label="来源" min-width="120" />
            <el-table-column label="启用" width="90">
              <template #default="{ row }">
                <el-tag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? '是' : '否' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" min-width="120" fixed="right">
              <template #default="{ row }">
                <div class="inline-actions">
                  <el-button link type="primary" @click="openAssetDetail('jumpAsset', row)">详情</el-button>
                  <el-button link @click="go('/jump/assets')">管理页</el-button>
                </div>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
      </el-tabs>
    </el-card>
      </div>
    </div>

    <el-drawer
      v-model="assetDetailVisible"
      :title="assetDetailTitle"
      size="720px"
      :destroy-on-close="false"
    >
      <el-skeleton v-if="assetDetailLoading" :rows="8" animated />
      <template v-else-if="assetDetailData">
        <el-tabs v-model="assetDetailTab">
          <el-tab-pane label="概览" name="overview">
            <el-descriptions :column="2" border size="small">
              <el-descriptions-item v-for="item in assetDetailRows" :key="item.label" :label="item.label">
                {{ item.value }}
              </el-descriptions-item>
            </el-descriptions>
          </el-tab-pane>

          <el-tab-pane :label="`关联会话 (${assetRelatedSessions.length})`" name="sessions">
            <el-table :fit="true" :data="assetRelatedSessions" size="small" max-height="300" empty-text="暂无关联会话">
              <el-table-column prop="session_no" label="会话号" min-width="140" />
              <el-table-column prop="asset_name" label="资产" min-width="130" />
              <el-table-column prop="username" label="用户" width="90" />
              <el-table-column prop="status" label="状态" width="100" />
              <el-table-column label="开始时间" min-width="150">
                <template #default="{ row }">{{ formatTime(row.started_at) }}</template>
              </el-table-column>
              <el-table-column label="操作" width="130">
                <template #default="{ row }">
                  <el-button size="small" link type="primary" @click="openJumpSessionByID(row.id)">会话详情</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>

          <el-tab-pane :label="`关联风险 (${assetRelatedRisks.length})`" name="risks">
            <el-table :fit="true" :data="assetRelatedRisks" size="small" max-height="300" empty-text="暂无关联风险">
              <el-table-column prop="severity" label="级别" width="90">
                <template #default="{ row }">
                  <el-tag :type="severityTag(row.severity)">{{ String(row.severity || '').toUpperCase() || '-' }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="rule_name" label="命中规则" min-width="160" />
              <el-table-column prop="description" label="描述" min-width="190" show-overflow-tooltip />
              <el-table-column label="触发时间" min-width="150">
                <template #default="{ row }">{{ formatTime(row.fired_at) }}</template>
              </el-table-column>
              <el-table-column label="操作" width="130">
                <template #default="{ row }">
                  <el-button size="small" link type="warning" @click="openAssetDetail('riskEvent', row)">查看详情</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>

          <el-tab-pane label="快捷操作" name="actions">
            <div class="asset-action-grid">
              <el-card shadow="never">
                <div class="asset-action-title">常用操作</div>
                <div class="asset-detail-actions">
                  <el-button
                    v-for="action in assetDetailActions"
                    :key="action.key"
                    :type="action.type || 'default'"
                    :plain="action.plain !== false"
                    @click="runAssetDetailQuickAction(action.key)"
                  >
                    {{ action.label }}
                  </el-button>
                </div>
              </el-card>
            </div>
          </el-tab-pane>
        </el-tabs>
      </template>
      <el-empty v-else description="暂无详情数据" />
    </el-drawer>

    <BatchResultDrawer
      v-model="batchResultVisible"
      :title="batchResultTitle"
      :summary="batchResultSummary"
      :records="batchResultRecords"
    />
  </el-card>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getErrorMessage, isCancelError } from '@/utils/error'
import BatchActionBar from '@/components/hub/BatchActionBar.vue'
import QuickGroupTags from '@/components/hub/QuickGroupTags.vue'
import BatchResultDrawer from '@/components/hub/BatchResultDrawer.vue'

const router = useRouter()
const loading = ref(false)
const hosts = ref([])
const groups = ref([])
const networkDevices = ref([])
const firewalls = ref([])
const jumpSessions = ref([])
const jumpRiskEvents = ref([])
const jumpAssets = ref([])
const commandStats = ref({})
const syncingNetworkFromFirewall = ref(false)
const riskBatching = ref(false)
const activePanel = ref('offline')
const panelKeyword = ref('')
const nowTick = ref(Date.now())
const riskTableRef = ref(null)
const assetTreeRef = ref(null)
const selectedRiskRows = ref([])
const selectedTreeNode = ref('all')
const treeKeyword = ref('')
const batchResultVisible = ref(false)
const batchResultTitle = ref('')
const batchResultSummary = ref({ total: 0, success: 0, failed: 0 })
const batchResultRecords = ref([])
const assetDetailVisible = ref(false)
const assetDetailLoading = ref(false)
const assetDetailType = ref('')
const assetDetailData = ref(null)
const assetDetailTab = ref('overview')
let freshnessTicker = null

const CHECK_STALE_HOURS = 24
const PENDING_SESSION_STALE_MINUTES = 30

const stats = reactive({
  hostTotal: 0,
  hostOffline: 0,
  networkTotal: 0,
  networkOffline: 0,
  firewallAlert: 0,
  jumpPending: 0,
  jumpActive: 0,
  riskCritical: 0,
  recheckPending: 0,
  pendingApprovalTimeout: 0,
  pendingBacklog: 0
})


const panelRouteMap = {
  offline: '/host',
  pending: '/jump/sessions',
  active: '/jump/sessions',
  risk: '/jump/sessions',
  firewall: '/firewall',
  jumpAssets: '/jump/assets'
}

const assetDetailRouteMap = {
  host: '/host',
  network: '/cmdb/network-devices',
  firewall: '/firewall',
  jumpAsset: '/jump/assets',
  pendingSession: '/jump/sessions',
  activeSession: '/jump/sessions',
  riskEvent: '/jump/sessions'
}

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })
const go = (path) => router.push(path)
const focusAssetPanel = (panel) => {
  if (!panel || panel === activePanel.value) return
  activePanel.value = panel
}

const normalizeText = (value) => String(value ?? '').trim().toLowerCase()

const calcAgeHours = (timeValue) => {
  if (!timeValue) return Number.POSITIVE_INFINITY
  const target = new Date(timeValue)
  if (Number.isNaN(target.getTime())) return Number.POSITIVE_INFINITY
  const ageMs = nowTick.value - target.getTime()
  if (ageMs <= 0) return 0
  return ageMs / (60 * 60 * 1000)
}

const calcAgeMinutes = (timeValue) => calcAgeHours(timeValue) * 60

const isCheckStale = (checkedAt) => calcAgeHours(checkedAt) >= CHECK_STALE_HOURS
const isPendingSessionStale = (row) => calcAgeMinutes(row?.started_at) >= PENDING_SESSION_STALE_MINUTES

const formatPendingWait = (startedAt) => {
  const minutes = calcAgeMinutes(startedAt)
  if (!Number.isFinite(minutes)) return '未知'
  if (minutes < 1) return '<1m'
  if (minutes < 60) return `${Math.floor(minutes)}m`
  const hours = Math.floor(minutes / 60)
  const remain = Math.floor(minutes % 60)
  return remain > 0 ? `${hours}h${remain}m` : `${hours}h`
}

const formatCheckAge = (checkedAt) => {
  const hours = calcAgeHours(checkedAt)
  if (!Number.isFinite(hours)) return '未检查'
  if (hours < 1) return '1小时内'
  return `${Math.floor(hours)}h前`
}

const isOnlineStatus = (status) => {
  if (status === null || status === undefined) return false
  if (status === true || status === 1) return true
  const normalized = normalizeText(status)
  return normalized === 'online' || normalized === 'normal' || normalized === 'active' || normalized === '在线'
}

const firewallStatus = (status) => {
  const value = Number(status)
  if (value === 1) return { text: '正常', type: 'success' }
  if (value === 2) return { text: '告警', type: 'danger' }
  return { text: '离线', type: 'warning' }
}

const severityTag = (value) => {
  const v = normalizeText(value)
  if (v === 'critical') return 'danger'
  if (v === 'warning') return 'warning'
  return 'info'
}

const hostGroupMeta = computed(() => {
  const countMap = new Map()
  const labelMap = new Map()
  groups.value.forEach((item) => {
    labelMap.set(item.id, item.name || item.id)
  })
  hosts.value.forEach((item) => {
    const key = item.group_id || 'ungrouped'
    countMap.set(key, (countMap.get(key) || 0) + 1)
    if (item.group_id && item.group?.name && !labelMap.has(item.group_id)) {
      labelMap.set(item.group_id, item.group.name)
    }
  })
  return { countMap, labelMap }
})

const hostGroupNodes = computed(() => {
  const { countMap, labelMap } = hostGroupMeta.value
  const nodes = []
  labelMap.forEach((label, id) => {
    nodes.push({
      id: `group:${id}`,
      label,
      count: countMap.get(id) || 0,
      nodeType: 'hostGroup',
      groupId: id
    })
  })
  nodes.sort((a, b) => b.count - a.count || a.label.localeCompare(b.label))
  const ungroupedCount = countMap.get('ungrouped') || 0
  if (ungroupedCount > 0) {
    nodes.push({
      id: 'group:ungrouped',
      label: '未分组主机',
      count: ungroupedCount,
      nodeType: 'hostUngrouped'
    })
  }
  return nodes
})

const assetTreeData = computed(() => {
  const hostCount = hosts.value.length
  const networkCount = networkDevices.value.length
  const firewallCount = firewalls.value.length
  const firewallOfflineCount = firewalls.value.filter((item) => Number(item.status) !== 1).length
  const jumpAssetCount = jumpAssets.value.length
  const sessionCount = jumpSessions.value.length
  const total = hostCount + networkCount + firewallCount + jumpAssetCount + sessionCount

  return [
    { id: 'all', label: '全部资产', count: total, nodeType: 'all' },
    {
      id: 'group-root',
      label: '主机分组',
      count: hostCount,
      nodeType: 'groupRoot',
      children: hostGroupNodes.value
    },
    {
      id: 'type-root',
      label: '资产类型',
      count: total,
      nodeType: 'typeRoot',
      children: [
        { id: 'type:host', label: '主机', count: hostCount, nodeType: 'assetType', assetType: 'host' },
        { id: 'type:network', label: '网络设备', count: networkCount, nodeType: 'assetType', assetType: 'network' },
        { id: 'type:firewall', label: '防火墙', count: firewallCount, nodeType: 'assetType', assetType: 'firewall' },
        { id: 'type:jumpAsset', label: '堡垒机资产', count: jumpAssetCount, nodeType: 'assetType', assetType: 'jumpAsset' },
        { id: 'type:jumpSession', label: '堡垒机会话', count: sessionCount, nodeType: 'assetType', assetType: 'jumpSession' }
      ]
    },
    {
      id: 'focus-root',
      label: '处置焦点',
      count: stats.pendingBacklog,
      nodeType: 'focusRoot',
      children: [
        { id: 'focus:offline', label: '离线资产', count: stats.hostOffline + stats.networkOffline + firewallOfflineCount, nodeType: 'focus', focusKey: 'offline' },
        { id: 'focus:pending', label: '待审批会话', count: stats.jumpPending, nodeType: 'focus', focusKey: 'pending' },
        { id: 'focus:riskCritical', label: '高危风险', count: stats.riskCritical, nodeType: 'focus', focusKey: 'riskCritical' }
      ]
    }
  ]
})

const assetTreeNodeMap = computed(() => {
  const map = {}
  const walk = (nodes) => {
    nodes.forEach((node) => {
      map[node.id] = node
      if (Array.isArray(node.children) && node.children.length) walk(node.children)
    })
  }
  walk(assetTreeData.value)
  return map
})

const selectedTreeNodeMeta = computed(() => assetTreeNodeMap.value[selectedTreeNode.value] || assetTreeNodeMap.value.all || { nodeType: 'all', label: '全部资产' })

const filterAssetTreeNode = (value, data) => {
  const keyword = normalizeText(value)
  if (!keyword) return true
  return normalizeText(data?.label).includes(keyword)
}

const handleAssetTreeNodeClick = (node) => {
  if (!node?.id) return
  selectedTreeNode.value = node.id
}

const scopeMatcher = (kind, row) => {
  const scope = selectedTreeNodeMeta.value
  if (!scope || scope.nodeType === 'all' || scope.nodeType === 'typeRoot' || scope.nodeType === 'focusRoot') return true
  if (scope.nodeType === 'groupRoot') return kind === 'host' || kind === 'offlineAsset'
  if (scope.nodeType === 'hostGroup') {
    if (kind === 'host') return row?.group_id === scope.groupId
    if (kind === 'offlineAsset') return row?.scopeType === 'host' && row?.group_id === scope.groupId
    return false
  }
  if (scope.nodeType === 'hostUngrouped') {
    if (kind === 'host') return !row?.group_id
    if (kind === 'offlineAsset') return row?.scopeType === 'host' && !row?.group_id
    return false
  }
  if (scope.nodeType === 'assetType') {
    const matchTypeMap = {
      host: ['host', 'offlineHost'],
      network: ['network', 'offlineNetwork'],
      firewall: ['firewall', 'offlineFirewall'],
      jumpAsset: ['jumpAsset'],
      jumpSession: ['pendingSession', 'activeSession', 'riskEvent', 'offlineSession']
    }
    const matched = matchTypeMap[scope.assetType] || []
    return matched.includes(kind)
  }
  if (scope.nodeType === 'focus') {
    if (scope.focusKey === 'offline') return kind.startsWith('offline')
    if (scope.focusKey === 'pending') return kind === 'pendingSession'
    if (scope.focusKey === 'riskCritical') return kind === 'riskEvent' && normalizeText(row?.severity) === 'critical'
  }
  return true
}

const scopedHosts = computed(() => hosts.value.filter((item) => scopeMatcher('host', item)))
const scopedNetworkDevices = computed(() => networkDevices.value.filter((item) => scopeMatcher('network', item)))
const scopedFirewalls = computed(() => firewalls.value.filter((item) => scopeMatcher('firewall', item)))
const scopedJumpAssets = computed(() => jumpAssets.value.filter((item) => scopeMatcher('jumpAsset', item)))
const scopedPendingSessionsRaw = computed(() =>
  jumpSessions.value
    .filter((item) => String(item.status) === 'pending_approval' && scopeMatcher('pendingSession', item))
    .sort((a, b) => new Date(b.started_at || 0).getTime() - new Date(a.started_at || 0).getTime())
)
const scopedActiveSessionsRaw = computed(() =>
  jumpSessions.value
    .filter((item) => String(item.status) === 'active' && scopeMatcher('activeSession', item))
    .sort((a, b) => new Date(b.started_at || 0).getTime() - new Date(a.started_at || 0).getTime())
)
const scopedRiskEventsRaw = computed(() =>
  jumpRiskEvents.value
    .filter((item) => scopeMatcher('riskEvent', item))
    .sort((a, b) => new Date(b.fired_at || 0).getTime() - new Date(a.fired_at || 0).getTime())
)

const hostOnlineRate = computed(() => {
  if (!stats.hostTotal) return 0
  return Math.round(((stats.hostTotal - stats.hostOffline) / stats.hostTotal) * 100)
})

const networkOnlineRate = computed(() => {
  if (!stats.networkTotal) return 0
  return Math.round(((stats.networkTotal - stats.networkOffline) / stats.networkTotal) * 100)
})

const pendingSessions = computed(() => scopedPendingSessionsRaw.value.slice(0, 12))

const riskEvents = computed(() => scopedRiskEventsRaw.value.slice(0, 12))

const riskCriticalCount = computed(() =>
  riskEvents.value.filter((item) => normalizeText(item.severity) === 'critical').length
)

const actionableRiskCount = computed(() => riskEvents.value.filter((item) => Boolean(item.session_id)).length)

const selectedRiskCount = computed(() => selectedRiskRows.value.filter((item) => Boolean(item?.session_id)).length)

const riskHeaderTags = computed(() => [
  { label: `Critical ${riskCriticalCount.value}`, type: 'danger' },
  { label: `可断开 ${actionableRiskCount.value}`, type: 'warning' }
])

const riskHeaderActions = computed(() => [
  {
    key: 'disconnect-selected',
    label: '批量断开已选',
    type: 'warning',
    plain: true,
    loading: riskBatching.value,
    disabled: !selectedRiskCount.value
  },
  {
    key: 'disconnect-critical',
    label: '处置高危',
    type: undefined,
    plain: true,
    loading: riskBatching.value,
    disabled: !riskCriticalCount.value
  }
])

const riskEventGroups = computed(() => {
  const groups = new Map()
  riskEvents.value.forEach((item) => {
    if (!item.session_id) return
    const rule = item.rule_name || item.event_type || '未分类风险'
    const key = `${rule}`
    const current = groups.get(key) || { key, label: rule, count: 0, level: 'warning', rows: [] }
    current.count += 1
    if (normalizeText(item.severity) === 'critical') current.level = 'danger'
    current.rows.push(item)
    groups.set(key, current)
  })
  return [...groups.values()].sort((a, b) => b.count - a.count).slice(0, 8)
})

const staleNetworkDevices = computed(() =>
  networkDevices.value.filter((item) => isOnlineStatus(item.status) && isCheckStale(item.last_check_at))
)

const staleFirewalls = computed(() =>
  firewalls.value.filter((item) => Number(item.status) === 1 && isCheckStale(item.last_check_at))
)

const scopedStaleNetworkDevices = computed(() =>
  scopedNetworkDevices.value.filter((item) => isOnlineStatus(item.status) && isCheckStale(item.last_check_at))
)

const scopedStaleFirewalls = computed(() =>
  scopedFirewalls.value.filter((item) => Number(item.status) === 1 && isCheckStale(item.last_check_at))
)

const stalePendingSessions = computed(() =>
  jumpSessions.value.filter((item) => String(item.status) === 'pending_approval' && isPendingSessionStale(item))
)

const offlineAssets = computed(() => {
  const rows = []
  scopedHosts.value.forEach((item) => {
    if (!isOnlineStatus(item.status)) {
      rows.push({
        type: '主机',
        id: item.id,
        name: item.name || '-',
        address: item.ip || '-',
        group_id: item.group_id,
        scopeType: 'host',
        source: item,
        statusText: '离线',
        level: 'danger',
        actionLabel: '连通测试',
        path: '/host'
      })
    }
  })
  scopedNetworkDevices.value.forEach((item) => {
    if (!isOnlineStatus(item.status)) {
      rows.push({
        type: '网络设备',
        id: item.id,
        name: item.name || '-',
        address: item.ip || item.address || '-',
        scopeType: 'network',
        source: item,
        statusText: '离线',
        level: 'warning',
        actionLabel: '设备诊断',
        path: '/cmdb/network-devices'
      })
    }
  })
  scopedFirewalls.value.forEach((item) => {
    const status = Number(item.status)
    if (status !== 1) {
      rows.push({
        type: '防火墙',
        id: item.id,
        name: item.name || '-',
        address: item.ip || '-',
        scopeType: 'firewall',
        source: item,
        statusText: status === 2 ? '告警' : '离线',
        level: status === 2 ? 'danger' : 'warning',
        actionLabel: 'SNMP采集',
        path: '/firewall'
      })
    }
  })
  scopedStaleNetworkDevices.value.forEach((item) => {
    rows.push({
      type: '网络设备',
      id: item.id,
      name: item.name || '-',
      address: item.ip || item.address || '-',
      scopeType: 'network',
      source: item,
      statusText: `待复检（${formatCheckAge(item.last_check_at)}）`,
      level: 'warning',
      actionLabel: '设备诊断',
      path: '/cmdb/network-devices'
    })
  })
  scopedStaleFirewalls.value.forEach((item) => {
    rows.push({
      type: '防火墙',
      id: item.id,
      name: item.name || '-',
      address: item.ip || '-',
      scopeType: 'firewall',
      source: item,
      statusText: `待复检（${formatCheckAge(item.last_check_at)}）`,
      level: 'warning',
      actionLabel: 'SNMP采集',
      path: '/firewall'
    })
  })
  scopedPendingSessionsRaw.value.filter((item) => isPendingSessionStale(item)).forEach((item) => {
    rows.push({
      type: '会话审批',
      id: item.id,
      name: item.session_no || item.id || '-',
      address: item.asset_name || '-',
      scopeType: 'session',
      source: item,
      statusText: `超时待审批（${formatPendingWait(item.started_at)}）`,
      level: 'warning',
      actionLabel: '进入审批',
      path: '/jump/sessions'
    })
  })
  return rows
    .sort((a, b) => (a.level === 'danger' ? -1 : 1) - (b.level === 'danger' ? -1 : 1))
    .slice(0, 40)
})

const filterRows = (rows, fields) => {
  const keyword = normalizeText(panelKeyword.value)
  const base = Array.isArray(rows) ? rows : []
  if (!keyword) return base.slice(0, 30)
  return base.filter((row) => fields.some((field) => normalizeText(field(row)).includes(keyword))).slice(0, 30)
}

const filteredOfflineAssets = computed(() =>
  filterRows(offlineAssets.value, [(row) => row.type, (row) => row.name, (row) => row.address, (row) => row.statusText])
)

const filteredPendingSessions = computed(() =>
  filterRows(scopedPendingSessionsRaw.value, [(row) => row.session_no, (row) => row.asset_name, (row) => row.username, (row) => row.protocol, (row) => row.source_ip])
)

const filteredActiveSessions = computed(() =>
  filterRows(
    scopedActiveSessionsRaw.value,
    [(row) => row.session_no, (row) => row.asset_name, (row) => row.username, (row) => row.protocol, (row) => row.source_ip]
  )
)

const filteredRiskEvents = computed(() =>
  filterRows(scopedRiskEventsRaw.value, [(row) => row.asset_name, (row) => row.username, (row) => row.severity, (row) => row.rule_name, (row) => row.description, (row) => row.command])
)

const filteredFirewalls = computed(() =>
  filterRows(scopedFirewalls.value, [(row) => row.name, (row) => row.vendor, (row) => row.ip, (row) => firewallStatus(row.status).text])
)

const filteredJumpAssets = computed(() =>
  filterRows(scopedJumpAssets.value, [(row) => row.name, (row) => row.asset_type, (row) => row.protocol, (row) => row.address, (row) => row.source])
)

const panelOptions = computed(() => [
  { name: 'offline', label: '离线资产', count: offlineAssets.value.length },
  { name: 'pending', label: '待审批会话', count: scopedPendingSessionsRaw.value.length },
  { name: 'active', label: '活跃会话', count: scopedActiveSessionsRaw.value.length },
  { name: 'risk', label: '风险事件', count: scopedRiskEventsRaw.value.length },
  { name: 'firewall', label: '防火墙', count: scopedFirewalls.value.length },
  { name: 'jumpAssets', label: '堡垒机资产', count: scopedJumpAssets.value.length }
])

const activePanelMeta = computed(
  () => panelOptions.value.find((item) => item.name === activePanel.value) || panelOptions.value[0] || { label: '-', count: 0 }
)

const assetDetailTitleMap = {
  host: '主机详情',
  network: '网络设备详情',
  firewall: '防火墙详情',
  jumpAsset: '堡垒机资产详情',
  pendingSession: '待审批会话详情',
  activeSession: '活跃会话详情',
  riskEvent: '风险事件详情'
}

const assetDetailTitle = computed(() => assetDetailTitleMap[assetDetailType.value] || '资产详情')
const assetDetailPath = computed(() => assetDetailRouteMap[assetDetailType.value] || '/asset/ops')
const assetDetailSessionID = computed(() => {
  if (!assetDetailData.value) return ''
  if (assetDetailType.value === 'riskEvent') return assetDetailData.value.session_id || ''
  if (assetDetailType.value === 'pendingSession' || assetDetailType.value === 'activeSession') return assetDetailData.value.id || ''
  return ''
})

const assetDetailRows = computed(() => {
  const item = assetDetailData.value
  if (!item) return []
  if (assetDetailType.value === 'host') {
    return [
      { label: '主机名', value: item.name || '-' },
      { label: 'IP', value: item.ip || '-' },
      { label: '端口', value: item.port || '-' },
      { label: '状态', value: isOnlineStatus(item.status) ? '在线' : '离线/维护' },
      { label: '操作系统', value: item.os || '-' },
      { label: '分组', value: item.group?.name || item.group_name || '-' },
      { label: 'CPU', value: item.cpu || '-' },
      { label: '内存', value: item.memory || '-' },
      { label: '磁盘', value: item.disk || '-' },
      { label: '更新时间', value: formatTime(item.updated_at) }
    ]
  }
  if (assetDetailType.value === 'network') {
    return [
      { label: '设备名', value: item.name || '-' },
      { label: '类型', value: item.device_type || '-' },
      { label: '厂商/型号', value: `${item.vendor || '-'} / ${item.model || '-'}` },
      { label: '管理IP', value: item.ip || '-' },
      { label: '管理端口', value: item.manage_port || '-' },
      { label: '状态', value: isOnlineStatus(item.status) ? '在线' : '离线/告警' },
      { label: 'SNMP版本', value: item.snmp_version || '-' },
      { label: '位置', value: item.location || '-' },
      { label: '最近检查', value: formatTime(item.last_check_at) },
      { label: '更新时间', value: formatTime(item.updated_at) }
    ]
  }
  if (assetDetailType.value === 'firewall') {
    return [
      { label: '设备名', value: item.name || '-' },
      { label: '厂商/型号', value: `${item.vendor || '-'} / ${item.model || '-'}` },
      { label: '管理IP', value: item.ip || '-' },
      { label: '状态', value: firewallStatus(item.status).text },
      { label: 'CPU使用率', value: item.cpu_usage !== undefined ? `${item.cpu_usage}%` : '-' },
      { label: '内存使用率', value: item.memory_usage !== undefined ? `${item.memory_usage}%` : '-' },
      { label: '会话数', value: item.session_count ?? '-' },
      { label: '吞吐量', value: item.throughput ?? '-' },
      { label: '最近检查', value: formatTime(item.last_check_at) },
      { label: '更新时间', value: formatTime(item.updated_at) }
    ]
  }
  if (assetDetailType.value === 'jumpAsset') {
    return [
      { label: '资产名', value: item.name || '-' },
      { label: '资产类型', value: item.asset_type || '-' },
      { label: '协议', value: item.protocol || '-' },
      { label: '地址', value: `${item.address || '-'}:${item.port || '-'}` },
      { label: '来源', value: item.source || '-' },
      { label: '标签', value: item.tags || '-' },
      { label: '启用', value: item.enabled ? '是' : '否' },
      { label: '更新时间', value: formatTime(item.updated_at) }
    ]
  }
  if (assetDetailType.value === 'pendingSession' || assetDetailType.value === 'activeSession') {
    return [
      { label: '会话号', value: item.session_no || '-' },
      { label: '资产', value: item.asset_name || '-' },
      { label: '用户', value: item.username || '-' },
      { label: '协议', value: item.protocol || '-' },
      { label: '来源IP', value: item.source_ip || '-' },
      { label: '状态', value: item.status || '-' },
      { label: '命令数', value: item.command_count ?? '-' },
      { label: '开始时间', value: formatTime(item.started_at) },
      { label: '最后命令', value: formatTime(item.last_command_at) }
    ]
  }
  if (assetDetailType.value === 'riskEvent') {
    return [
      { label: '风险级别', value: String(item.severity || '-').toUpperCase() },
      { label: '资产', value: item.asset_name || '-' },
      { label: '用户', value: item.username || '-' },
      { label: '事件类型', value: item.event_type || '-' },
      { label: '命中规则', value: item.rule_name || '-' },
      { label: '命令', value: item.command || '-' },
      { label: '描述', value: item.description || '-' },
      { label: '触发时间', value: formatTime(item.fired_at) },
      { label: '关联会话', value: item.session_id || '-' }
    ]
  }
  return []
})

const assetIdentityKeywords = computed(() => {
  const item = assetDetailData.value || {}
  const pool = [
    item.name,
    item.asset_name,
    item.ip,
    item.address,
    item.session_no
  ]
  return Array.from(
    new Set(
      pool
        .map((one) => normalizeText(one))
        .filter((one) => one && one !== '-')
    )
  )
})

const assetRelatedSessions = computed(() => {
  const type = assetDetailType.value
  const item = assetDetailData.value || {}
  if (type === 'pendingSession' || type === 'activeSession') {
    return item.id ? jumpSessions.value.filter((row) => row.id === item.id) : []
  }
  if (type === 'riskEvent') {
    return item.session_id ? jumpSessions.value.filter((row) => row.id === item.session_id) : []
  }
  const keys = assetIdentityKeywords.value
  if (!keys.length) return []
  return jumpSessions.value
    .filter((row) => {
      const content = normalizeText([row.asset_name, row.session_no, row.source_ip].filter(Boolean).join(' '))
      return keys.some((key) => content.includes(key))
    })
    .slice(0, 30)
})

const assetRelatedRisks = computed(() => {
  const type = assetDetailType.value
  const item = assetDetailData.value || {}
  if (type === 'riskEvent') {
    return item.id ? jumpRiskEvents.value.filter((row) => row.id === item.id) : [item].filter(Boolean)
  }
  if (type === 'pendingSession' || type === 'activeSession') {
    if (!item.id) return []
    return jumpRiskEvents.value.filter((row) => row.session_id === item.id).slice(0, 30)
  }
  const keys = assetIdentityKeywords.value
  if (!keys.length) return []
  return jumpRiskEvents.value
    .filter((row) => {
      const content = normalizeText([row.asset_name, row.description, row.command].filter(Boolean).join(' '))
      return keys.some((key) => content.includes(key))
    })
    .slice(0, 30)
})

const assetDetailActions = computed(() => {
  const type = assetDetailType.value
  const actions = [{ key: 'goManage', label: '进入管理页', type: 'primary', plain: false }]
  if (type === 'host') actions.push({ key: 'testHost', label: '连通测试' })
  if (type === 'network') actions.push({ key: 'testNetwork', label: '设备诊断' })
  if (type === 'firewall') {
    actions.push({ key: 'collectFirewall', label: 'SNMP采集', type: 'success' })
    actions.push({ key: 'testFirewall', label: 'SNMP测试', type: 'warning' })
  }
  if (type === 'pendingSession') {
    actions.push({ key: 'approveSession', label: '审批通过', type: 'success' })
    actions.push({ key: 'rejectSession', label: '拒绝会话', type: 'warning' })
  }
  if (type === 'activeSession') {
    actions.push({ key: 'connectSession', label: '连接会话', type: 'primary' })
    actions.push({ key: 'disconnectSession', label: '断开会话', type: 'danger' })
  }
  if (type === 'riskEvent') {
    actions.push({ key: 'disconnectRisk', label: '断开风险会话', type: 'danger' })
  }
  if (assetDetailSessionID.value) actions.push({ key: 'sessionDetail', label: '会话详情' })
  actions.push({ key: 'openJump', label: '打开会话中心' })
  return actions
})

const openOfflineDetail = (row) => {
  if (!row) return
  if (row.scopeType === 'host') openAssetDetail('host', row.source || row)
  else if (row.scopeType === 'network') openAssetDetail('network', row.source || row)
  else if (row.scopeType === 'firewall') openAssetDetail('firewall', row.source || row)
  else if (row.scopeType === 'session') openAssetDetail('pendingSession', row.source || row)
}

const openAssetDetail = async (type, row) => {
  if (!row) return
  assetDetailType.value = type
  assetDetailData.value = row
  assetDetailTab.value = 'overview'
  assetDetailVisible.value = true
  assetDetailLoading.value = false

  const id = row.id || row.session_id
  let api = ''
  if (type === 'host' && row.id) api = `/api/v1/cmdb/hosts/${row.id}`
  if (type === 'network' && row.id) api = `/api/v1/cmdb/network-devices/${row.id}`
  if (type === 'firewall' && row.id) api = `/api/v1/firewall/devices/${row.id}`
  if (type === 'jumpAsset' && row.id) api = `/api/v1/jump/assets/${row.id}`
  if ((type === 'pendingSession' || type === 'activeSession') && row.id) api = `/api/v1/jump/sessions/${row.id}`
  if (type === 'riskEvent' && id) api = `/api/v1/jump/sessions/${id}`

  if (!api) return

  assetDetailLoading.value = true
  try {
    const res = await axios.get(api, { headers: authHeaders() })
    if (res?.data?.code === 0 && res.data.data) {
      if (type === 'riskEvent') {
        assetDetailData.value = { ...row, ...(res.data.data || {}) }
      } else {
        assetDetailData.value = res.data.data
      }
    }
  } catch (err) {
    ElMessage.warning(getErrorMessage(err, '详情加载失败，已展示缓存数据'))
  } finally {
    assetDetailLoading.value = false
  }
}

const runAssetDetailQuickAction = async (key) => {
  const row = assetDetailData.value
  if (!row) return
  if (key === 'goManage') {
    go(assetDetailPath.value)
    return
  }
  if (key === 'sessionDetail') {
    openJumpSessionByID(assetDetailSessionID.value)
    return
  }
  if (key === 'openJump') {
    go('/jump/sessions')
    return
  }
  if (key === 'testHost') {
    await testHostConnectivity(row.id)
    return
  }
  if (key === 'testNetwork') {
    await testNetworkDevice(row.id)
    return
  }
  if (key === 'collectFirewall') {
    await collectFirewall(row)
    return
  }
  if (key === 'testFirewall') {
    await testFirewallSNMP(row)
    return
  }
  if (key === 'approveSession') {
    await approveJumpSession(row)
    return
  }
  if (key === 'rejectSession') {
    await rejectJumpSession(row)
    return
  }
  if (key === 'connectSession') {
    await connectJumpSession(row)
    return
  }
  if (key === 'disconnectSession') {
    await disconnectJumpSession(row)
    return
  }
  if (key === 'disconnectRisk') {
    await disconnectRiskSession(row)
  }
}

const openJumpSession = () => {
  router.push('/jump/sessions')
}

const openJumpSessionByID = (sessionID) => {
  if (!sessionID) {
    openJumpSession()
    return
  }
  router.push(`/jump/sessions?id=${encodeURIComponent(sessionID)}`)
}

const connectJumpSession = async (row) => {
  if (!row?.id) return
  try {
    const res = await axios.post(`/api/v1/jump/sessions/${row.id}/connect`, {}, { headers: authHeaders() })
    const openURL = res?.data?.data?.open_url
    if (openURL) {
      router.push(openURL)
      return
    }
    ElMessage.success('会话连接已就绪')
    await refreshAll()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '连接会话失败'))
  }
}

const approveJumpSession = async (row) => {
  if (!row?.id) return
  try {
    await axios.post(`/api/v1/jump/sessions/${row.id}/approve`, {}, { headers: authHeaders() })
    ElMessage.success('会话已通过审批')
    await refreshAll()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '审批失败'))
  }
}

const rejectJumpSession = async (row) => {
  if (!row?.id) return
  try {
    const { value } = await ElMessageBox.prompt('请输入拒绝原因', `拒绝会话 ${row.session_no || row.id}`, {
      inputType: 'textarea',
      inputPlaceholder: '例如：不在变更窗口'
    })
    await axios.post(`/api/v1/jump/sessions/${row.id}/reject`, { reason: value || '' }, { headers: authHeaders() })
    ElMessage.success('会话已拒绝')
    await refreshAll()
  } catch (err) {
    if (!isCancelError(err)) ElMessage.error(getErrorMessage(err, '拒绝失败'))
  }
}

const disconnectJumpSession = async (row) => {
  if (!row?.id) return
  try {
    await ElMessageBox.confirm(`确认断开会话 ${row.session_no || row.id} 吗？`, '提示', { type: 'warning' })
    await axios.post(`/api/v1/jump/sessions/${row.id}/disconnect`, { reason: '资产作战台手动断开' }, { headers: authHeaders() })
    ElMessage.success('会话已断开')
    await refreshAll()
  } catch (err) {
    if (!isCancelError(err)) ElMessage.error(getErrorMessage(err, '断开失败'))
  }
}

const onRiskSelectionChange = (rows) => {
  selectedRiskRows.value = Array.isArray(rows) ? rows : []
}

const handleRiskHeaderAction = async (key) => {
  if (key === 'disconnect-selected') await batchDisconnectSelectedRisk()
  if (key === 'disconnect-critical') await batchDisconnectCriticalRisk()
}

const showBatchResult = (title, targets, settled, successMessage) => {
  const records = targets.map((target, index) => {
    const result = settled[index]
    if (result?.status === 'fulfilled') {
      return { id: target.id, target: target.name, status: 'success', message: successMessage }
    }
    return {
      id: target.id,
      target: target.name,
      status: 'failed',
      message: getErrorMessage(result?.reason, '执行失败')
    }
  })
  const success = records.filter((item) => item.status === 'success').length
  const failed = records.length - success
  batchResultTitle.value = title
  batchResultSummary.value = { total: records.length, success, failed }
  batchResultRecords.value = records
  batchResultVisible.value = true
}

const disconnectRiskSessionByID = async (sessionID, reason) => {
  await axios.post(`/api/v1/jump/sessions/${sessionID}/disconnect`, { reason }, { headers: authHeaders() })
}

const disconnectRiskSession = async (row) => {
  const sessionID = row?.session_id
  if (!sessionID) {
    ElMessage.info('该风险事件没有关联在线会话')
    return
  }
  try {
    await ElMessageBox.confirm(`确认断开风险会话 ${sessionID} 吗？`, '提示', { type: 'warning' })
    await disconnectRiskSessionByID(sessionID, '风险事件触发手动断开')
    ElMessage.success('风险会话已断开')
    await refreshAll()
  } catch (err) {
    if (!isCancelError(err)) ElMessage.error(getErrorMessage(err, '断开风险会话失败'))
  }
}

const runBatchRiskDisconnect = async (rows, title, message) => {
  const sessionIDs = [...new Set(rows.map((item) => item?.session_id).filter(Boolean))]
  const targets = sessionIDs.map((id) => {
    const row = rows.find((item) => item?.session_id === id)
    return { id, name: row?.asset_name || row?.username || `会话-${id}` }
  })
  if (!sessionIDs.length) {
    ElMessage.info('没有可断开的会话')
    return
  }
  riskBatching.value = true
  try {
    await ElMessageBox.confirm(message, title, { type: 'warning' })
    const settled = await Promise.allSettled(
      sessionIDs.map((id) => disconnectRiskSessionByID(id, '资产作战台批量处置断开'))
    )
    const success = settled.filter((item) => item.status === 'fulfilled').length
    const fail = settled.length - success
    ElMessage.success(`批量处置完成：成功 ${success}，失败 ${fail}`)
    showBatchResult(title, targets, settled, '会话已断开')
    selectedRiskRows.value = []
    if (riskTableRef.value?.clearSelection) riskTableRef.value.clearSelection()
    await refreshAll()
  } catch (err) {
    if (!isCancelError(err)) ElMessage.error(getErrorMessage(err, '批量处置失败'))
  } finally {
    riskBatching.value = false
  }
}

const batchDisconnectSelectedRisk = async () => {
  await runBatchRiskDisconnect(
    selectedRiskRows.value,
    '批量断开风险会话',
    `确认断开已选择的 ${selectedRiskCount.value} 个风险会话吗？`
  )
}

const batchDisconnectCriticalRisk = async () => {
  const rows = riskEvents.value.filter((item) => normalizeText(item.severity) === 'critical' && item.session_id)
  await runBatchRiskDisconnect(rows, '处置高危风险', `确认断开 ${rows.length} 个 Critical 风险会话吗？`)
}

const batchDisconnectRiskGroup = async (group) => {
  const rows = Array.isArray(group?.rows) ? group.rows : []
  await runBatchRiskDisconnect(rows, `按规则处置：${group?.label || '-'}`, `确认断开「${group?.label || '-'}」共 ${rows.length} 个风险会话吗？`)
}

const testHostConnectivity = async (id) => {
  if (!id) return
  try {
    await axios.post(`/api/v1/cmdb/hosts/${id}/test`, {}, { headers: authHeaders() })
    ElMessage.success('主机连通测试完成')
    await refreshAll()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '主机连通测试失败'))
  }
}

const testNetworkDevice = async (id) => {
  if (!id) return
  try {
    const res = await axios.post(`/api/v1/cmdb/network-devices/${id}/test`, {}, { headers: authHeaders() })
    const status = Number(res?.data?.data?.status || 0)
    if (status === 1) ElMessage.success('网络设备诊断完成：在线')
    else if (status === 2) ElMessage.warning('网络设备诊断完成：部分可达')
    else ElMessage.warning('网络设备诊断完成：离线')
    await refreshAll()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '网络设备诊断失败'))
  }
}

const collectFirewall = async (row) => {
  if (!row?.id) return
  try {
    await axios.post(`/api/v1/firewall/devices/${row.id}/snmp/collect`, {}, { headers: authHeaders() })
    ElMessage.success('防火墙采集完成')
    await refreshAll()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '防火墙采集失败'))
  }
}

const testFirewallSNMP = async (row) => {
  if (!row?.id) return
  try {
    await axios.post(`/api/v1/firewall/devices/${row.id}/snmp/test`, {}, { headers: authHeaders() })
    ElMessage.success('SNMP 测试通过')
  } catch (err) {
    ElMessage.error(getErrorMessage(err, 'SNMP 测试失败'))
  }
}

const offlineAction = (row) => {
  if (!row) return
  if (row.type === '主机') {
    testHostConnectivity(row.id)
    return
  }
  if (row.type === '网络设备') {
    testNetworkDevice(row.id)
    return
  }
  if (row.type === '防火墙') {
    collectFirewall(row)
    return
  }
  if (row.type === '会话审批') {
    openJumpSession()
    return
  }
  go(row.path)
}

const syncNetworkDevicesFromFirewalls = async () => {
  syncingNetworkFromFirewall.value = true
  try {
    const res = await axios.post('/api/v1/cmdb/network-devices/sync/firewalls', {}, { headers: authHeaders() })
    const data = res?.data?.data || {}
    ElMessage.success(`同步完成：新增 ${data.created || 0}，更新 ${data.updated || 0}`)
    await refreshAll()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '同步失败'))
  } finally {
    syncingNetworkFromFirewall.value = false
  }
}

const openCurrentPanel = () => {
  go(panelRouteMap[activePanel.value] || '/host')
}

const formatTime = (value) => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString()
}

const safeArray = (res) => (Array.isArray(res?.data?.data) ? res.data.data : [])
const safeObject = (res) => (res?.data?.data && typeof res.data.data === 'object' ? res.data.data : {})

watch(treeKeyword, (value) => {
  if (assetTreeRef.value?.filter) assetTreeRef.value.filter(value)
})

watch(assetTreeData, (nodes) => {
  const exists = nodes.some((item) => item.id === selectedTreeNode.value) || Boolean(assetTreeNodeMap.value[selectedTreeNode.value])
  if (!exists) selectedTreeNode.value = 'all'
})

const refreshAll = async () => {
  loading.value = true
  try {
    const [hostRes, groupRes, networkRes, firewallRes, sessionRes, riskRes, jumpAssetRes, commandStatsRes] = await Promise.allSettled([
      axios.get('/api/v1/cmdb/hosts', { headers: authHeaders() }),
      axios.get('/api/v1/cmdb/groups', { headers: authHeaders() }),
      axios.get('/api/v1/cmdb/network-devices', { headers: authHeaders() }),
      axios.get('/api/v1/firewall/devices', { headers: authHeaders() }),
      axios.get('/api/v1/jump/sessions', { headers: authHeaders() }),
      axios.get('/api/v1/jump/risk-events', { headers: authHeaders() }),
      axios.get('/api/v1/jump/assets', { headers: authHeaders() }),
      axios.get('/api/v1/jump/command-rules/stats', { headers: authHeaders(), params: { days: 7 } })
    ])

    hosts.value = hostRes.status === 'fulfilled' ? safeArray(hostRes.value) : []
    groups.value = groupRes.status === 'fulfilled' ? safeArray(groupRes.value) : []
    networkDevices.value = networkRes.status === 'fulfilled' ? safeArray(networkRes.value) : []
    firewalls.value = firewallRes.status === 'fulfilled' ? safeArray(firewallRes.value) : []
    jumpSessions.value = sessionRes.status === 'fulfilled' ? safeArray(sessionRes.value) : []
    jumpRiskEvents.value = riskRes.status === 'fulfilled' ? safeArray(riskRes.value) : []
    jumpAssets.value = jumpAssetRes.status === 'fulfilled' ? safeArray(jumpAssetRes.value) : []
    commandStats.value = commandStatsRes.status === 'fulfilled' ? safeObject(commandStatsRes.value) : {}

    stats.hostTotal = hosts.value.length
    stats.hostOffline = hosts.value.filter((item) => !isOnlineStatus(item.status)).length
    stats.networkTotal = networkDevices.value.length
    stats.networkOffline = networkDevices.value.filter((item) => !isOnlineStatus(item.status)).length
    stats.firewallAlert = firewalls.value.filter((item) => Number(item.status) === 2).length
    stats.jumpPending = jumpSessions.value.filter((item) => String(item.status) === 'pending_approval').length
    stats.jumpActive = jumpSessions.value.filter((item) => String(item.status) === 'active').length
    stats.riskCritical = jumpRiskEvents.value.filter((item) => normalizeText(item.severity) === 'critical').length
    stats.recheckPending = staleNetworkDevices.value.length + staleFirewalls.value.length
    stats.pendingApprovalTimeout = stalePendingSessions.value.length
    stats.pendingBacklog = stats.recheckPending + stats.pendingApprovalTimeout + stats.riskCritical

    const failedCount = [hostRes, groupRes, networkRes, firewallRes, sessionRes, riskRes, jumpAssetRes, commandStatsRes].filter((r) => r.status === 'rejected').length
    if (failedCount > 0) {
      ElMessage.warning(`部分作战台数据加载失败(${failedCount}项)，已展示可用数据`)
    }
  } catch (err) {
    ElMessage.error(err?.response?.data?.message || err?.message || '加载资产作战台失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  refreshAll()
  freshnessTicker = window.setInterval(() => {
    nowTick.value = Date.now()
  }, 60 * 1000)
})

onBeforeUnmount(() => {
  if (freshnessTicker) {
    window.clearInterval(freshnessTicker)
    freshnessTicker = null
  }
})
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 12px; gap: 12px; }
.page-desc { color: var(--muted-text); margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; }
.workbench-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 12px;
  background: color-mix(in srgb, var(--card-bg) 86%, #ffffff 14%);
  padding: 10px 12px;
  margin-bottom: 12px;
}
.workbench-toolbar-left,
.workbench-toolbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.workbench-toolbar-label {
  font-size: 12px;
  color: var(--muted-text);
  margin-right: 4px;
}

.workbench-layout {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.workbench-main {
  flex: 1;
  min-width: 0;
}

.asset-tree-panel {
  width: 280px;
  flex: 0 0 280px;
  padding: 12px;
  border-radius: 14px;
  border: 1px solid var(--el-border-color-light);
  background: color-mix(in srgb, var(--card-bg) 86%, #ffffff 14%);
}

.asset-tree-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.asset-tree-title {
  font-size: 14px;
  font-weight: 600;
}

.asset-tree-search {
  margin-top: 10px;
}

.asset-tree-scroll {
  max-height: 760px;
  margin-top: 10px;
}

.asset-tree-node {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.asset-tree-node-label {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.summary-row { margin-bottom: 12px; }
.summary-row :deep(.el-card) { margin-bottom: 8px; }
.metric-title { color: var(--muted-text); font-size: 12px; }
.metric-value { font-size: 20px; font-weight: 600; margin-top: 6px; color: var(--el-text-color-primary); }
.metric-value.ok { color: #67c23a; }
.metric-value.warning { color: #e6a23c; }
.metric-value.danger { color: #f56c6c; }
.metric-sub { margin-top: 6px; font-size: 12px; color: var(--muted-text); }
.mt-12 { margin-top: 12px; }
.health-row { display: flex; justify-content: space-between; align-items: center; margin-bottom: 6px; }
.health-row strong { font-size: 15px; }
.mtop { margin-top: 12px; }

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
  width: 280px;
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

.inline-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.asset-detail-actions {
  margin-top: 12px;
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.asset-action-grid {
  display: grid;
  gap: 10px;
}

.asset-action-title {
  font-weight: 600;
  margin-bottom: 10px;
}

.integration-tabs :deep(.el-tabs__header) { display: none; }

@media (max-width: 1100px) {
  .workbench-layout {
    flex-direction: column;
  }

  .asset-tree-panel {
    width: 100%;
    flex: 1 1 auto;
  }

  .asset-tree-scroll {
    max-height: 280px;
  }

  .integration-header {
    align-items: flex-start;
    flex-direction: column;
  }

  .workbench-toolbar {
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

}
</style>
