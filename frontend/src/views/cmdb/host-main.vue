<template>
  <el-card class="host-page-card">
    <template #header>
      <div class="flex justify-between items-center header-wrap">
        <span class="font-bold">CMDB 主机管理</span>
        <div class="header-actions">
          <el-button type="primary" icon="Plus" @click="handleAdd">添加主机</el-button>
          <el-button type="primary" plain icon="Connection" :disabled="selectedRows.length !== 1" @click="openConnectionEditor">
            连接信息
          </el-button>
          <el-button plain icon="FolderOpened" @click="openGroupManager">分组维护</el-button>
          <el-button icon="Upload" @click="openImport">批量导入</el-button>
          <el-button icon="Download" @click="exportCSV">导出</el-button>
          <el-button type="warning" plain icon="Edit" :disabled="selectedRows.length === 0" @click="openBatchStatus">
            批量状态
          </el-button>
          <el-button type="danger" plain icon="Delete" :disabled="selectedRows.length === 0" @click="handleBatchDelete">
            批量删除 ({{ selectedRows.length }})
          </el-button>
          <el-button type="success" plain icon="Promotion" :loading="syncingStatus" @click="syncStatuses()">巡检状态</el-button>
          <el-button icon="Refresh" @click="fetchData">刷新列表</el-button>
        </div>
      </div>
    </template>

    <div class="host-layout">
      <div class="host-aside">
        <el-card shadow="never" class="group-card">
          <template #header>
            <div class="group-card-header">
              <span>资产分组</span>
              <div class="group-card-header-actions">
                <el-button link type="primary" @click="openGroupManager">维护</el-button>
                <el-button link type="primary" @click="clearGroupFilter">重置</el-button>
              </div>
            </div>
          </template>
          <el-input
            v-model="groupKeyword"
            placeholder="筛选分组"
            clearable
            class="group-search"
            @input="handleGroupKeywordChange"
          />
          <el-tree
            ref="groupTreeRef"
            :data="groupTreeData"
            :props="{ children: 'children', label: 'label' }"
            node-key="id"
            default-expand-all
            highlight-current
            :current-node-key="activeGroupId || 'all'"
            :filter-node-method="groupNodeFilter"
            :expand-on-click-node="false"
            @node-click="onGroupNodeClick"
          >
            <template #default="{ data }">
              <div class="group-tree-node">
                <span class="group-tree-label">{{ data.label }}</span>
                <el-tag size="small" effect="plain" type="info">{{ data.count || 0 }}</el-tag>
              </div>
            </template>
          </el-tree>
        </el-card>

        <el-card shadow="never" class="provider-card">
          <template #header>
            <div class="group-card-header">
              <span>云厂商分布</span>
              <span class="provider-total">{{ filteredTableData.length }}</span>
            </div>
          </template>
          <div class="provider-grid">
            <div v-for="item in providerSummaryList" :key="item.key" class="provider-item">
              <span class="provider-name">{{ item.label }}</span>
              <el-tag :type="providerTagType(item.key)" effect="plain">{{ item.count }}</el-tag>
            </div>
          </div>
        </el-card>
      </div>

      <div class="host-main">
        <div class="mb-4">
          <div class="flex gap-2 items-center filters-row">
            <el-input
              v-model="searchKeyword"
              placeholder="搜索主机名或IP"
              class="w-64"
              clearable
              @clear="fetchData"
              @keyup.enter="fetchData"
            >
              <template #append>
                <el-button icon="Search" @click="fetchData" />
              </template>
            </el-input>
            <el-select v-model="activeGroupId" placeholder="分组" clearable class="w-64" @change="onGroupSelectChange">
              <el-option label="全部主机" value="" />
              <el-option label="未分组" :value="UNGROUPED_GROUP_ID" />
              <el-option v-for="g in groups" :key="g.id" :label="g.name" :value="g.id" />
            </el-select>
            <el-tag type="info" effect="plain">总计 {{ filteredTableData.length }}</el-tag>
            <el-tag type="success" effect="plain">在线 {{ onlineCount }}</el-tag>
            <el-tag type="warning" effect="plain">离线 {{ offlineCount }}</el-tag>
          </div>
        </div>

        <div class="table-scroll">
          <el-table
            class="host-table"
            :fit="true"
            :data="filteredTableData"
            v-loading="loading"
            style="width: 100%; min-width: 1420px"
            @selection-change="selectedRows = $event"
          >
            <el-table-column type="selection" width="48" />
            <el-table-column prop="name" label="主机名" min-width="170" show-overflow-tooltip>
              <template #default="{ row }">
                <div class="flex items-center gap-2">
                  <el-icon class="text-gray-500 text-lg"><Monitor /></el-icon>
                  <span class="font-bold">{{ row.name }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="ip" label="IP地址" min-width="140" />
            <el-table-column label="云厂商" min-width="110" align="center">
              <template #default="{ row }">
                <el-tag size="small" :type="providerTagType(hostProvider(row))" effect="plain">
                  {{ providerLabel(hostProvider(row)) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="os" label="操作系统" min-width="160" show-overflow-tooltip />
            <el-table-column label="CPU" width="96" align="center">
              <template #default="{ row }">
                <el-tag size="small" :effect="hasMetricValue(row, 'cpu') ? 'light' : 'plain'" :type="metricTagType(metricValue(row, 'cpu'))">
                  {{ metricText(row, 'cpu') }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="内存" width="96" align="center">
              <template #default="{ row }">
                <el-tag size="small" :effect="hasMetricValue(row, 'memory') ? 'light' : 'plain'" :type="metricTagType(metricValue(row, 'memory'))">
                  {{ metricText(row, 'memory') }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="磁盘" width="96" align="center">
              <template #default="{ row }">
                <el-tag size="small" :effect="hasMetricValue(row, 'disk') ? 'light' : 'plain'" :type="metricTagType(metricValue(row, 'disk'))">
                  {{ metricText(row, 'disk') }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="status" label="状态" min-width="130">
              <template #default="{ row }">
                <StatusBadge
                  :text="hostStatusMeta(row).text"
                  :type="hostStatusMeta(row).type"
                  :source="hostStatusMeta(row).source"
                  :check-at="hostStatusMeta(row).checkAt"
                  :is-stale="hostStatusMeta(row).isStale"
                  :stale-text="hostStatusMeta(row).staleText"
                  :reason="hostStatusMeta(row).reason"
                />
              </template>
            </el-table-column>
            <el-table-column prop="last_check_at" label="最后检测" min-width="170">
              <template #default="{ row }">
                {{ formatTime(hostStatusMeta(row).checkAt || row.last_check_at) }}
              </template>
            </el-table-column>
            <el-table-column prop="status_reason" label="状态说明" min-width="220" show-overflow-tooltip>
              <template #default="{ row }">
                {{ hostStatusMeta(row).reason || '-' }}
              </template>
            </el-table-column>
            <el-table-column prop="group.name" label="分组" min-width="120" show-overflow-tooltip>
              <template #default="{ row }">{{ row?.group?.name || '-' }}</template>
            </el-table-column>
            <el-table-column label="操作" width="320" fixed="right">
              <template #default="{ row }">
                <div class="op-row">
                  <el-button size="small" plain icon="View" @click="openDetail(row)">详情</el-button>
                  <el-button size="small" type="warning" plain icon="FirstAidKit" @click="handleTest(row)">检测</el-button>
                  <el-button size="small" type="primary" plain icon="Edit" @click="handleEdit(row)">编辑</el-button>
                  <el-dropdown trigger="click" @command="(command) => handleRowCommand(row, command)">
                    <el-button size="small" plain>
                      更多
                      <el-icon class="el-icon--right"><ArrowDown /></el-icon>
                    </el-button>
                    <template #dropdown>
                      <el-dropdown-menu>
                        <el-dropdown-item command="process">进程</el-dropdown-item>
                        <el-dropdown-item command="tcp">TCP</el-dropdown-item>
                        <el-dropdown-item command="monitor">监控</el-dropdown-item>
                        <el-dropdown-item command="delete" divided>删除</el-dropdown-item>
                      </el-dropdown-menu>
                    </template>
                  </el-dropdown>
                </div>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>
    </div>

    <el-drawer
      v-model="detailVisible"
      :title="`主机详情 - ${detailHost?.name || '-'}`"
      size="75%"
      :destroy-on-close="false"
      class="host-detail-drawer"
    >
      <div class="detail-toolbar">
        <div class="detail-host-info">
          <el-tag :type="hostStatusMeta(detailHost || {}).type" effect="dark">
            {{ hostStatusMeta(detailHost || {}).text || '未知' }}
          </el-tag>
          <span>IP: {{ detailHost?.ip || '-' }}</span>
          <span>系统: {{ detailHost?.os || '-' }}</span>
          <span>分组: {{ detailHost?.group?.name || '-' }}</span>
          <span>实例: {{ detailInstanceLabel || '-' }}</span>
        </div>
        <div class="detail-actions">
          <el-radio-group v-model="detailRangeHours" size="small" @change="fetchDetailMetrics()">
            <el-radio-button :label="1">1h</el-radio-button>
            <el-radio-button :label="6">6h</el-radio-button>
            <el-radio-button :label="24">24h</el-radio-button>
          </el-radio-group>
          <el-switch v-model="detailAutoRefresh" inline-prompt active-text="自动刷新" inactive-text="手动" />
          <el-button size="small" icon="Refresh" :loading="detailLoading" @click="fetchDetailMetrics()">刷新</el-button>
          <el-button size="small" type="success" plain @click="openInspect(detailHost, 'process')">进程</el-button>
          <el-button size="small" type="info" plain @click="openInspect(detailHost, 'tcp')">TCP</el-button>
        </div>
      </div>

      <div class="detail-metrics">
        <div class="metric-card">
          <span class="metric-label">CPU</span>
          <strong>{{ metricText(detailHost, 'cpu') }}</strong>
        </div>
        <div class="metric-card">
          <span class="metric-label">内存</span>
          <strong>{{ metricText(detailHost, 'memory') }}</strong>
        </div>
        <div class="metric-card">
          <span class="metric-label">磁盘</span>
          <strong>{{ metricText(detailHost, 'disk') }}</strong>
        </div>
        <div class="metric-card">
          <span class="metric-label">最后检测</span>
          <strong>{{ formatTime(hostStatusMeta(detailHost || {}).checkAt || detailHost?.last_check_at) }}</strong>
        </div>
      </div>

      <el-empty v-if="!detailInstanceLabel" description="未匹配到监控实例，无法展示趋势图" />
      <div v-else class="detail-chart-grid" v-loading="detailLoading">
        <div class="chart-card"><div class="chart-title">CPU 使用率</div><div ref="detailCpuChartRef" class="detail-chart" /></div>
        <div class="chart-card"><div class="chart-title">内存使用率</div><div ref="detailMemChartRef" class="detail-chart" /></div>
        <div class="chart-card"><div class="chart-title">磁盘使用率</div><div ref="detailDiskChartRef" class="detail-chart" /></div>
        <div class="chart-card"><div class="chart-title">系统负载</div><div ref="detailLoadChartRef" class="detail-chart" /></div>
        <div class="chart-card"><div class="chart-title">网络接收 (KB/s)</div><div ref="detailNetInChartRef" class="detail-chart" /></div>
        <div class="chart-card"><div class="chart-title">网络发送 (KB/s)</div><div ref="detailNetOutChartRef" class="detail-chart" /></div>
      </div>
    </el-drawer>

    <el-dialog append-to-body v-model="dialogVisible" :title="isEdit ? '编辑主机' : '添加主机'" width="500px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="主机名" required>
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="IP地址" required>
          <el-input v-model="form.ip" />
        </el-form-item>
        <el-form-item label="端口">
          <el-input-number v-model="form.port" :min="1" :max="65535" />
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="form.status" style="width: 100%">
            <el-option label="在线" :value="1" />
            <el-option label="离线" :value="0" />
            <el-option label="维护" :value="2" />
          </el-select>
        </el-form-item>
        <el-form-item label="用户名">
          <el-input v-model="form.username" placeholder="root" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="form.password" type="password" show-password placeholder="如有变更请填写" />
          <div v-if="isEdit" class="helper-row">已加载当前密码，可直接修改。</div>
        </el-form-item>
        <el-form-item label="分组">
          <el-input v-model="form.group_name" placeholder="默认分组" :disabled="isEdit" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitForm" :loading="submitting">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="groupManageVisible" title="分组维护" width="820px">
      <div class="group-manage-toolbar">
        <el-button type="primary" icon="Plus" @click="openCreateGroup">新增分组</el-button>
        <el-button icon="Refresh" @click="fetchGroups">刷新</el-button>
      </div>
      <el-table :fit="true" :data="groups" stripe max-height="420" empty-text="暂无分组">
        <el-table-column prop="name" label="分组名称" min-width="180" />
        <el-table-column prop="description" label="描述" min-width="220" show-overflow-tooltip />
        <el-table-column prop="parent_id" label="父级ID" min-width="160" />
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="primary" plain @click="openEditGroup(row)">编辑</el-button>
            <el-button size="small" type="danger" plain @click="handleGroupDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <el-dialog append-to-body v-model="groupEditorVisible" :title="groupEditorEdit ? '编辑分组' : '新增分组'" width="480px">
      <el-form :model="groupForm" label-width="90px">
        <el-form-item label="分组名称" required>
          <el-input v-model="groupForm.name" placeholder="如：生产环境" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="groupForm.description" type="textarea" :rows="3" />
        </el-form-item>
        <el-form-item label="父级ID">
          <el-input v-model="groupForm.parent_id" placeholder="可选" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="groupEditorVisible = false">取消</el-button>
        <el-button type="primary" :loading="groupSubmitting" @click="saveGroup">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog append-to-body v-model="testVisible" title="主机测试" width="640px">
      <el-alert v-if="testError" type="error" :closable="false" show-icon>{{ testError }}</el-alert>
      <el-skeleton v-if="testLoading" :rows="4" animated />
      <div v-else class="test-block">
        <div class="test-title">uname -a</div>
        <pre class="test-pre">{{ testResult?.uname?.out || '-' }}</pre>
        <div class="test-title">/etc/os-release</div>
        <pre class="test-pre">{{ testResult?.os_release?.out || '-' }}</pre>
      </div>
      <template #footer>
        <el-button @click="testVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <el-dialog append-to-body v-model="inspectVisible" :title="inspectTitle" width="1040px" @closed="clearInspectAutoTimer">
      <div class="inspect-toolbar">
        <div class="inspect-host">
          <strong>{{ inspectHost?.name || '-' }}</strong>
          <span>{{ inspectHost?.ip || '-' }}</span>
          <span>更新时间：{{ inspectData.updatedAt ? formatTime(inspectData.updatedAt) : '-' }}</span>
        </div>
        <div class="inspect-actions">
          <el-radio-group v-model="inspectMode" size="small">
            <el-radio-button label="process">进程监控</el-radio-button>
            <el-radio-button label="tcp">TCP连接</el-radio-button>
          </el-radio-group>
          <el-switch v-model="inspectAutoRefresh" inline-prompt active-text="自动刷新" inactive-text="手动" />
          <el-button size="small" icon="Refresh" :loading="inspectLoading" @click="refreshInspect">手动刷新</el-button>
        </div>
      </div>

      <el-skeleton v-if="inspectLoading" :rows="6" animated />
      <template v-else>
        <div v-if="inspectMode === 'process'" class="inspect-process-grid">
          <el-card class="inspect-card" shadow="never">
            <template #header>Top CPU 进程</template>
            <el-table
              :fit="true"
              :data="inspectData.topCpu"
              size="small"
              max-height="300"
              empty-text="暂无进程数据"
              :row-class-name="processRowClassName"
            >
              <el-table-column prop="pid" label="PID" width="90" />
              <el-table-column prop="command" label="进程" min-width="180" show-overflow-tooltip />
              <el-table-column label="CPU%" width="130">
                <template #default="{ row }">
                  <el-progress
                    :percentage="toPercentNumber(row.cpu)"
                    :stroke-width="8"
                    :show-text="false"
                    :status="toNumber(row.cpu) >= 70 ? 'exception' : undefined"
                  />
                  <div class="mini-percent">{{ formatPercentText(row.cpu) }}</div>
                </template>
              </el-table-column>
              <el-table-column label="内存%" width="120">
                <template #default="{ row }">{{ formatPercentText(row.memory) }}</template>
              </el-table-column>
            </el-table>
          </el-card>
          <el-card class="inspect-card" shadow="never">
            <template #header>Top 内存进程</template>
            <el-table
              :fit="true"
              :data="inspectData.topMem"
              size="small"
              max-height="300"
              empty-text="暂无进程数据"
              :row-class-name="processRowClassName"
            >
              <el-table-column prop="pid" label="PID" width="90" />
              <el-table-column prop="command" label="进程" min-width="180" show-overflow-tooltip />
              <el-table-column label="内存%" width="130">
                <template #default="{ row }">
                  <el-progress
                    :percentage="toPercentNumber(row.memory)"
                    :stroke-width="8"
                    :show-text="false"
                    status="warning"
                  />
                  <div class="mini-percent">{{ formatPercentText(row.memory) }}</div>
                </template>
              </el-table-column>
              <el-table-column label="CPU%" width="120">
                <template #default="{ row }">{{ formatPercentText(row.cpu) }}</template>
              </el-table-column>
            </el-table>
          </el-card>
        </div>

        <div v-else class="inspect-tcp-block">
          <div class="tcp-summary">
            <el-tag type="info" effect="plain">总连接 {{ inspectData.tcpSummary.total || 0 }}</el-tag>
            <el-tag type="success" effect="plain">ESTABLISHED {{ inspectData.tcpSummary.established || 0 }}</el-tag>
            <el-tag type="warning" effect="plain">LISTEN {{ inspectData.tcpSummary.listen || 0 }}</el-tag>
            <el-tag type="danger" effect="plain">TIME_WAIT {{ inspectData.tcpSummary.time_wait || 0 }}</el-tag>
            <el-tag type="danger" effect="plain">风险连接 {{ tcpRiskCount }}</el-tag>
          </div>
          <el-table
            :fit="true"
            :data="inspectData.tcpRows"
            size="small"
            max-height="460"
            empty-text="暂无TCP连接数据"
            :row-class-name="tcpRowClassName"
          >
            <el-table-column prop="proto" label="协议" width="90" />
            <el-table-column label="状态" width="130">
              <template #default="{ row }">
                <el-tag size="small" :type="tcpStateTagType(row.state)" effect="plain">{{ row.state || '-' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="local" label="本地地址" min-width="220" show-overflow-tooltip />
            <el-table-column prop="remote" label="远端地址" min-width="220" show-overflow-tooltip />
            <el-table-column prop="process" label="进程" min-width="220" show-overflow-tooltip />
          </el-table>
        </div>
      </template>

      <template #footer>
        <el-button @click="inspectVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <el-dialog append-to-body v-model="importVisible" title="批量导入主机" width="720px">
      <el-alert type="info" :closable="false" show-icon>
        格式：name,ip,port,username,password,group_name,status,os（第一行可写表头）
      </el-alert>
      <el-input
        v-model="importText"
        type="textarea"
        :rows="10"
        placeholder="例如：&#10;web-1,192.168.1.10,22,root,pass,prod,1,Ubuntu"
      />
      <div class="import-actions">
        <el-button @click="importVisible = false">取消</el-button>
        <el-button type="primary" :loading="importLoading" @click="submitImport">开始导入</el-button>
      </div>
    </el-dialog>

    <el-dialog append-to-body v-model="batchStatusVisible" title="批量修改状态" width="420px">
      <el-form label-width="80px">
        <el-form-item label="状态">
          <el-select v-model="batchStatus" placeholder="选择状态" style="width: 100%">
            <el-option label="在线" :value="1" />
            <el-option label="离线" :value="0" />
            <el-option label="维护" :value="2" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="batchStatusVisible = false">取消</el-button>
        <el-button type="primary" :loading="batchStatusLoading" @click="submitBatchStatus">保存</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { computed, ref, reactive, onBeforeUnmount, onMounted, nextTick, watch } from 'vue'
import axios from 'axios'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import * as echarts from 'echarts'
import StatusBadge from '@/components/common/StatusBadge.vue'
import { cmdbHostStatusMeta } from '@/utils/status'

const router = useRouter()

const UNGROUPED_GROUP_ID = '__ungrouped__'
const DEFAULT_HOST_PORT = 22

const loading = ref(false)
const tableData = ref([])
const groups = ref([])
const selectedRows = ref([])
const searchKeyword = ref('')
const activeGroupId = ref('')
const groupKeyword = ref('')
const groupTreeRef = ref(null)

const dialogVisible = ref(false)
const submitting = ref(false)
const isEdit = ref(false)
const currentId = ref('')
const groupManageVisible = ref(false)
const groupEditorVisible = ref(false)
const groupEditorEdit = ref(false)
const groupSubmitting = ref(false)
const currentGroupId = ref('')

const testVisible = ref(false)
const testLoading = ref(false)
const testResult = ref(null)
const testError = ref('')

const importVisible = ref(false)
const importLoading = ref(false)
const importText = ref('')

const batchStatusVisible = ref(false)
const batchStatusLoading = ref(false)
const batchStatus = ref(null)

const syncingStatus = ref(false)
const hostMetricsMap = ref({})
const hostProviderMap = ref({})

const inspectVisible = ref(false)
const inspectLoading = ref(false)
const inspectHost = ref(null)
const inspectMode = ref('process')
const inspectAutoRefresh = ref(true)
const inspectData = ref({
  updatedAt: '',
  topCpu: [],
  topMem: [],
  tcpRows: [],
  tcpSummary: {}
})

const detailVisible = ref(false)
const detailHost = ref(null)
const detailLoading = ref(false)
const detailRangeHours = ref(1)
const detailAutoRefresh = ref(true)
const detailInstanceLabel = ref('')

const detailCpuChartRef = ref(null)
const detailMemChartRef = ref(null)
const detailDiskChartRef = ref(null)
const detailLoadChartRef = ref(null)
const detailNetInChartRef = ref(null)
const detailNetOutChartRef = ref(null)

let detailCpuChart = null
let detailMemChart = null
let detailDiskChart = null
let detailLoadChart = null
let detailNetInChart = null
let detailNetOutChart = null
let autoSyncTimer = null
let inspectAutoTimer = null
let detailAutoTimer = null

const form = reactive({
  name: '',
  ip: '',
  port: DEFAULT_HOST_PORT,
  status: 1,
  username: '',
  password: '',
  group_name: ''
})

const groupForm = reactive({
  name: '',
  description: '',
  parent_id: ''
})

const providerConfig = {
  aliyun: { label: '阿里云', type: 'warning' },
  aws: { label: 'AWS', type: 'danger' },
  huawei: { label: '华为云', type: 'success' },
  tencent: { label: '腾讯云', type: 'primary' },
  baidu: { label: '百度云', type: 'info' },
  azure: { label: 'Azure', type: 'primary' },
  gcp: { label: 'GCP', type: 'success' },
  onprem: { label: '自建', type: '' },
  unknown: { label: '-', type: 'info' }
}

const authHeaders = () => ({ Authorization: 'Bearer ' + localStorage.getItem('token') })

const getErrorMessage = (e, fallback) => {
  if (typeof e === 'string' && e.trim()) return e.trim()
  if (e?.response?.data?.message) return e.response.data.message
  if (e?.response?.data?.error) return e.response.data.error
  if (e?.message) return e.message
  return fallback
}

const toTime = (value) => {
  if (!value) return null
  const ts = new Date(value).getTime()
  return Number.isNaN(ts) ? null : ts
}

const formatTime = (value) => {
  const ts = toTime(value)
  if (!ts) return '-'
  return new Date(ts).toLocaleString()
}

const toNumber = (value) => {
  if (value === null || value === undefined) return NaN
  if (typeof value === 'number') return value
  const num = Number(String(value).replace('%', '').trim())
  return Number.isFinite(num) ? num : NaN
}

const normalizeHostAddress = (value) => {
  const text = String(value || '').trim()
  if (!text) return ''
  if (text.startsWith('[')) {
    const idx = text.indexOf(']')
    if (idx > 0) return text.slice(1, idx)
  }
  const lastColon = text.lastIndexOf(':')
  if (lastColon > 0 && text.indexOf(':') === lastColon) {
    return text.slice(0, lastColon)
  }
  return text
}

const hostStatusMeta = (row) => cmdbHostStatusMeta(row || {}, { staleMinutes: 3 })

const hostMetric = (row) => {
  const ip = normalizeHostAddress(row?.ip)
  const byIP = hostMetricsMap.value[ip]
  if (byIP) return byIP
  const byName = hostMetricsMap.value[String(row?.name || '').trim()]
  if (byName) return byName
  return {}
}

const normalizeProvider = (value) => {
  const text = String(value || '').trim().toLowerCase()
  if (!text) return 'unknown'
  if (text.includes('aliyun') || text.includes('阿里')) return 'aliyun'
  if (text.includes('aws')) return 'aws'
  if (text.includes('huawei') || text.includes('华为')) return 'huawei'
  if (text.includes('tencent') || text.includes('腾讯')) return 'tencent'
  if (text.includes('baidu') || text.includes('百度')) return 'baidu'
  if (text.includes('azure')) return 'azure'
  if (text.includes('gcp') || text.includes('google')) return 'gcp'
  if (text.includes('onprem') || text.includes('自建') || text.includes('private')) return 'onprem'
  return text
}

const providerLabel = (key) => (providerConfig[key] || providerConfig.unknown).label
const providerTagType = (key) => (providerConfig[key] || providerConfig.unknown).type

const parseProviderFromTags = (tags) => {
  const text = String(tags || '').toLowerCase()
  if (!text) return 'unknown'
  const providerMatch = text.match(/provider\s*[:=]\s*([a-zA-Z0-9_-]+)/)
  if (providerMatch?.[1]) return normalizeProvider(providerMatch[1])
  return normalizeProvider(text)
}

const hostProvider = (row) => {
  const ip = normalizeHostAddress(row?.ip)
  if (ip && hostProviderMap.value[ip]) return hostProviderMap.value[ip]
  const providerFromMetric = normalizeProvider(hostMetric(row)?.provider)
  if (providerFromMetric !== 'unknown') return providerFromMetric
  const providerFromTags = parseProviderFromTags(row?.tags)
  return providerFromTags || 'unknown'
}

const formatPercent = (metricValue, fallback = '') => {
  const metricNum = toNumber(metricValue)
  if (Number.isFinite(metricNum)) return `${metricNum.toFixed(1)}%`
  const fallbackNum = toNumber(fallback)
  if (Number.isFinite(fallbackNum)) return `${fallbackNum.toFixed(1)}%`
  return '-'
}

const metricValue = (row, key) => {
  const metricNum = toNumber(hostMetric(row)?.[key])
  return Number.isFinite(metricNum) ? metricNum : NaN
}

const hasMetricValue = (row, key) => Number.isFinite(metricValue(row, key))

const metricText = (row, key) => {
  const value = metricValue(row, key)
  if (!Number.isFinite(value)) return '--'
  return `${value.toFixed(1)}%`
}

const metricTagType = (value) => {
  const num = toNumber(value)
  if (!Number.isFinite(num)) return 'info'
  if (num >= 85) return 'danger'
  if (num >= 70) return 'warning'
  return 'success'
}

const groupHostCountMap = computed(() => {
  const map = {}
  let ungrouped = 0
  tableData.value.forEach((row) => {
    const gid = String(row?.group?.id || row?.group_id || '').trim()
    if (!gid) {
      ungrouped += 1
      return
    }
    map[gid] = (map[gid] || 0) + 1
  })
  map[UNGROUPED_GROUP_ID] = ungrouped
  return map
})

const groupTreeData = computed(() => {
  const nodes = {}
  const roots = []

  groups.value.forEach((g) => {
    nodes[g.id] = {
      id: g.id,
      label: g.name,
      count: groupHostCountMap.value[g.id] || 0,
      parent_id: g.parent_id || '',
      children: []
    }
  })

  Object.values(nodes).forEach((node) => {
    if (node.parent_id && nodes[node.parent_id]) {
      nodes[node.parent_id].children.push(node)
    } else {
      roots.push(node)
    }
  })

  return [
    { id: 'all', label: '全部主机', count: tableData.value.length, children: [] },
    ...roots,
    { id: UNGROUPED_GROUP_ID, label: '未分组', count: groupHostCountMap.value[UNGROUPED_GROUP_ID] || 0, children: [] }
  ]
})

const groupNodeFilter = (value, data) => {
  if (!value) return true
  return String(data?.label || '').toLowerCase().includes(String(value).toLowerCase())
}

const filteredTableData = computed(() => {
  if (!activeGroupId.value) return tableData.value
  if (activeGroupId.value === UNGROUPED_GROUP_ID) {
    return tableData.value.filter((row) => !row?.group?.id && !row?.group_id)
  }
  return tableData.value.filter((row) => {
    const gid = String(row?.group?.id || row?.group_id || '').trim()
    return gid === String(activeGroupId.value)
  })
})

const providerSummaryList = computed(() => {
  const counter = {}
  filteredTableData.value.forEach((row) => {
    const key = hostProvider(row)
    counter[key] = (counter[key] || 0) + 1
  })
  const baseKeys = ['onprem', 'aliyun', 'tencent', 'huawei', 'aws', 'baidu', 'azure', 'gcp', 'unknown']
  return baseKeys
    .filter((key) => (counter[key] || 0) > 0)
    .map((key) => ({ key, label: providerLabel(key), count: counter[key] || 0 }))
})

const onlineCount = computed(() => filteredTableData.value.filter((row) => hostStatusMeta(row).key === 'online').length)
const offlineCount = computed(() => filteredTableData.value.filter((row) => hostStatusMeta(row).key !== 'online').length)
const inspectTitle = computed(() => (inspectMode.value === 'tcp' ? 'TCP 连接监控' : '进程监控'))
const tcpRiskCount = computed(() => inspectData.value.tcpRows.filter((row) => isRiskTCPRow(row)).length)

const handleGroupKeywordChange = (value) => {
  groupTreeRef.value?.filter(value)
}

const onGroupNodeClick = (node) => {
  if (!node) return
  if (node.id === 'all') {
    activeGroupId.value = ''
    return
  }
  activeGroupId.value = node.id
}

const onGroupSelectChange = (value) => {
  const key = value || 'all'
  nextTick(() => {
    groupTreeRef.value?.setCurrentKey(key)
  })
}

const clearGroupFilter = () => {
  activeGroupId.value = ''
  groupKeyword.value = ''
  groupTreeRef.value?.filter('')
  nextTick(() => {
    groupTreeRef.value?.setCurrentKey('all')
  })
}

const resetGroupForm = () => {
  groupForm.name = ''
  groupForm.description = ''
  groupForm.parent_id = ''
}

const openGroupManager = async () => {
  groupManageVisible.value = true
  await fetchGroups()
}

const openCreateGroup = () => {
  currentGroupId.value = ''
  groupEditorEdit.value = false
  resetGroupForm()
  groupEditorVisible.value = true
}

const openEditGroup = (row) => {
  currentGroupId.value = row.id
  groupEditorEdit.value = true
  groupForm.name = row.name || ''
  groupForm.description = row.description || ''
  groupForm.parent_id = row.parent_id || ''
  groupEditorVisible.value = true
}

const openConnectionEditor = () => {
  if (selectedRows.value.length !== 1) {
    ElMessage.warning('请选择 1 台主机后再编辑连接信息')
    return
  }
  handleEdit(selectedRows.value[0])
}

const saveGroup = async () => {
  if (!groupForm.name.trim()) {
    ElMessage.warning('请填写分组名称')
    return
  }
  groupSubmitting.value = true
  try {
    const url = groupEditorEdit.value ? `/api/v1/cmdb/groups/${currentGroupId.value}` : '/api/v1/cmdb/groups'
    const method = groupEditorEdit.value ? 'put' : 'post'
    const res = await axios({
      url,
      method,
      headers: authHeaders(),
      data: {
        name: groupForm.name.trim(),
        description: groupForm.description,
        parent_id: groupForm.parent_id
      }
    })
    if (res.data?.code === 0) {
      ElMessage.success(groupEditorEdit.value ? '分组更新成功' : '分组创建成功')
      groupEditorVisible.value = false
      await Promise.all([fetchGroups(), fetchData()])
    } else {
      ElMessage.error(res.data?.message || '分组保存失败')
    }
  } catch (e) {
    ElMessage.error(getErrorMessage(e, '分组保存失败'))
  } finally {
    groupSubmitting.value = false
  }
}

const handleGroupDelete = async (row) => {
  try {
    await ElMessageBox.confirm(`确定删除分组“${row.name}”吗？`, '提示', {
      type: 'warning'
    })
    await axios.delete(`/api/v1/cmdb/groups/${row.id}`, { headers: authHeaders() })
    if (String(activeGroupId.value || '') === String(row.id)) {
      clearGroupFilter()
    }
    ElMessage.success('分组删除成功')
    await Promise.all([fetchGroups(), fetchData()])
  } catch (e) {
    if (e !== 'cancel' && e !== 'close') {
      ElMessage.error(getErrorMessage(e, '分组删除失败'))
    }
  }
}

const fetchData = async () => {
  loading.value = true
  try {
    const hostReq = axios.get('/api/v1/cmdb/hosts', {
      headers: authHeaders(),
      params: {
        keyword: searchKeyword.value,
        live: 1
      }
    })
    const metricsReq = axios.get('/api/v1/monitor/servers', { headers: authHeaders() })
    const cloudReq = axios.get('/api/v1/cmdb/cloud/resources', { headers: authHeaders() })
    const [hostRes, metricsRes, cloudRes] = await Promise.allSettled([hostReq, metricsReq, cloudReq])

    if (hostRes.status === 'fulfilled' && hostRes.value.data?.code === 0) {
      tableData.value = hostRes.value.data.data || []
    } else {
      const reason = hostRes.status === 'rejected' ? hostRes.reason : hostRes.value?.data?.message
      ElMessage.error(getErrorMessage(reason, '加载主机列表失败'))
    }

    if (metricsRes.status === 'fulfilled' && metricsRes.value.data?.code === 0) {
      const rows = Array.isArray(metricsRes.value.data.data) ? metricsRes.value.data.data : []
      const map = {}
      rows.forEach((item) => {
        const ipKey = normalizeHostAddress(item?.ip || item?.instance)
        const nameKey = String(item?.hostname || '').trim()
        if (ipKey) map[ipKey] = item
        if (nameKey) map[nameKey] = item
      })
      hostMetricsMap.value = map
    } else {
      hostMetricsMap.value = {}
    }

    if (cloudRes.status === 'fulfilled' && cloudRes.value.data?.code === 0) {
      const rows = Array.isArray(cloudRes.value.data.data) ? cloudRes.value.data.data : []
      const providerMap = {}
      rows.forEach((item) => {
        const ip = normalizeHostAddress(item?.ip)
        if (!ip) return
        const provider = normalizeProvider(item?.account?.provider || item?.provider)
        if (!providerMap[ip] && provider !== 'unknown') providerMap[ip] = provider
      })
      hostProviderMap.value = providerMap
    } else {
      hostProviderMap.value = {}
    }
  } catch (e) {
    ElMessage.error(getErrorMessage(e, '加载失败'))
  } finally {
    loading.value = false
  }
}

const syncStatuses = async (silent = false) => {
  syncingStatus.value = true
  try {
    const res = await axios.post('/api/v1/cmdb/hosts/sync-status', {}, { headers: authHeaders() })
    if (res.data?.code === 0 && !silent) {
      const info = res.data?.data || {}
      ElMessage.success(`巡检完成：在线 ${info.online ?? 0}，离线 ${info.offline ?? 0}，维护 ${info.maintenance ?? 0}`)
    }
  } catch (e) {
    if (!silent) ElMessage.error(getErrorMessage(e, '巡检失败'))
  } finally {
    syncingStatus.value = false
    await fetchData()
  }
}

const fetchGroups = async () => {
  try {
    const res = await axios.get('/api/v1/cmdb/groups', { headers: authHeaders() })
    if (res.data?.code === 0) groups.value = res.data.data || []
  } catch (e) {
    ElMessage.error(getErrorMessage(e, '分组加载失败'))
  }
}

const handleAdd = () => {
  isEdit.value = false
  form.name = ''
  form.ip = ''
  form.port = DEFAULT_HOST_PORT
  form.status = 1
  form.username = 'root'
  form.password = ''
  form.group_name = ''
  dialogVisible.value = true
}

const handleEdit = async (row) => {
  isEdit.value = true
  currentId.value = row.id
  try {
    const res = await axios.get(`/api/v1/cmdb/hosts/${row.id}`, { headers: authHeaders() })
    if (res.data.code === 0) {
      const data = res.data.data
      form.name = data.name
      form.ip = data.ip
      form.port = data.port
      form.status = data.status ?? 1
      form.username = data.credential ? data.credential.username : ''
      form.password = data.credential ? data.credential.password : ''
      form.group_name = data.group ? data.group.name : ''
      dialogVisible.value = true
    }
  } catch (e) {
    ElMessage.error(getErrorMessage(e, '获取详情失败'))
  }
}

const submitForm = async () => {
  submitting.value = true
  try {
    const url = isEdit.value ? `/api/v1/cmdb/hosts/${currentId.value}` : '/api/v1/cmdb/hosts'
    const method = isEdit.value ? 'put' : 'post'

    const res = await axios({
      method,
      url,
      data: form,
      headers: authHeaders()
    })

    if (res.data.code === 0) {
      ElMessage.success(isEdit.value ? '更新成功' : '添加成功')
      dialogVisible.value = false
      fetchData()
    } else {
      ElMessage.error(res.data.message)
    }
  } catch (e) {
    ElMessage.error(getErrorMessage(e, isEdit.value ? '更新失败' : '添加失败'))
  } finally {
    submitting.value = false
  }
}

const handleDelete = async (row) => {
  try {
    await ElMessageBox.confirm('确定删除该主机吗?', '警告', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await axios.delete(`/api/v1/cmdb/hosts/${row.id}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    await fetchData()
  } catch (e) {
    if (e !== 'cancel' && e !== 'close') ElMessage.error(getErrorMessage(e, '删除失败'))
  }
}

const handleBatchDelete = async () => {
  if (selectedRows.value.length === 0) return
  try {
    await ElMessageBox.confirm(`确定删除选中的 ${selectedRows.value.length} 台主机吗?`, '警告', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning'
    })
    for (const row of selectedRows.value) {
      await axios.delete(`/api/v1/cmdb/hosts/${row.id}`, { headers: authHeaders() })
    }
    ElMessage.success('批量删除成功')
    selectedRows.value = []
    await fetchData()
  } catch (e) {
    if (e !== 'cancel' && e !== 'close') ElMessage.error(getErrorMessage(e, '批量删除失败'))
  }
}

const openImport = () => {
  importText.value = ''
  importVisible.value = true
}

const parseCSV = (text) => {
  const lines = text.split(/\r?\n/).map((l) => l.trim()).filter(Boolean)
  if (lines.length === 0) return []
  const delim = lines[0].includes('\t') ? '\t' : ','
  const headers = lines[0].toLowerCase().split(delim).map((s) => s.trim())
  const hasHeader = headers.includes('name') || headers.includes('ip')
  const start = hasHeader ? 1 : 0
  const cols = hasHeader ? headers : ['name', 'ip', 'port', 'username', 'password', 'group_name', 'status', 'os']
  return lines.slice(start).map((line) => {
    const parts = line.split(delim).map((s) => s.trim())
    const obj = {}
    cols.forEach((k, idx) => { obj[k] = parts[idx] || '' })
    return obj
  })
}

const submitImport = async () => {
  const rows = parseCSV(importText.value)
  if (rows.length === 0) {
    ElMessage.warning('请填写导入内容')
    return
  }
  importLoading.value = true
  try {
    for (const row of rows) {
      if (!row.name || !row.ip) continue
      await axios.post('/api/v1/cmdb/hosts', {
        name: row.name,
        ip: row.ip,
        port: row.port ? Number(row.port) : DEFAULT_HOST_PORT,
        username: row.username || '',
        password: row.password || '',
        group_name: row.group_name || '',
        status: row.status ? Number(row.status) : 1,
        os: row.os || ''
      }, {
        headers: authHeaders()
      })
    }
    ElMessage.success('导入完成')
    importVisible.value = false
    await fetchData()
  } catch (e) {
    ElMessage.error(getErrorMessage(e, '导入失败'))
  } finally {
    importLoading.value = false
  }
}

const exportCSV = () => {
  const headers = ['name', 'ip', 'provider', 'port', 'os', 'status', 'group', 'username']
  const rows = filteredTableData.value.map((h) => [
    h.name,
    h.ip,
    providerLabel(hostProvider(h)),
    h.port,
    h.os,
    h.status,
    h.group?.name || '',
    h.credential?.username || ''
  ])
  const csv = [headers.join(','), ...rows.map((r) => r.map((v) => `"${String(v ?? '').replace(/"/g, '""')}"`).join(','))].join('\n')
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'cmdb_hosts.csv'
  a.click()
  URL.revokeObjectURL(url)
}

const openBatchStatus = () => {
  batchStatus.value = 1
  batchStatusVisible.value = true
}

const submitBatchStatus = async () => {
  if (batchStatus.value === null) return
  batchStatusLoading.value = true
  try {
    for (const row of selectedRows.value) {
      await axios.put(`/api/v1/cmdb/hosts/${row.id}`, {
        ...row,
        status: batchStatus.value
      }, {
        headers: authHeaders()
      })
    }
    ElMessage.success('状态更新成功')
    batchStatusVisible.value = false
    await fetchData()
  } catch (e) {
    ElMessage.error(getErrorMessage(e, '状态更新失败'))
  } finally {
    batchStatusLoading.value = false
  }
}

const handleTest = async (row) => {
  testVisible.value = true
  testLoading.value = true
  testResult.value = null
  testError.value = ''
  try {
    const res = await axios.post(`/api/v1/cmdb/hosts/${row.id}/test`, {}, { headers: authHeaders() })
    if (res.data.code === 0) {
      testResult.value = res.data.data
    } else {
      testError.value = res.data.message || '测试失败'
    }
  } catch (e) {
    testError.value = getErrorMessage(e, '测试失败')
  } finally {
    testLoading.value = false
    await fetchData()
  }
}

const toPercentNumber = (value) => {
  const num = toNumber(value)
  if (!Number.isFinite(num) || num < 0) return 0
  if (num > 100) return 100
  return Number(num.toFixed(2))
}

const formatPercentText = (value) => {
  const num = toNumber(value)
  if (!Number.isFinite(num)) return '-'
  return `${num.toFixed(2)}%`
}

const normalizeProcessRows = (rows) =>
  Array.isArray(rows)
    ? rows.map((item) => ({
        pid: item?.pid || '-',
        command: item?.command || '-',
        cpu: item?.cpu || '0',
        memory: item?.memory || '0'
      }))
    : []

const normalizeTCPRows = (rows) =>
  Array.isArray(rows)
    ? rows.map((item) => ({
        proto: item?.proto || '-',
        state: item?.state || '-',
        local: item?.local || '-',
        remote: item?.remote || '-',
        process: item?.process || '-'
      }))
    : []

const getLocalPort = (localAddr) => {
  const text = String(localAddr || '').trim()
  if (!text) return ''
  const idx = text.lastIndexOf(':')
  if (idx < 0) return ''
  return text.slice(idx + 1)
}

const riskyStates = new Set(['syn-recv', 'close-wait', 'fin-wait-2', 'last-ack'])
const commonListenPorts = new Set(['22', '80', '443', '2379', '2380', '6443', '3306', '5432', '6379', '27017'])

const isRiskTCPRow = (row) => {
  const state = String(row?.state || '').toLowerCase()
  if (riskyStates.has(state)) return true
  if (state === 'listen') {
    const port = getLocalPort(row?.local)
    if (port && !commonListenPorts.has(port)) return true
  }
  return false
}

const processRowClassName = ({ row }) => {
  const cpu = toNumber(row?.cpu)
  const mem = toNumber(row?.memory)
  if ((Number.isFinite(cpu) && cpu >= 70) || (Number.isFinite(mem) && mem >= 80)) return 'danger-row'
  if ((Number.isFinite(cpu) && cpu >= 40) || (Number.isFinite(mem) && mem >= 60)) return 'warn-row'
  return ''
}

const tcpRowClassName = ({ row }) => (isRiskTCPRow(row) ? 'danger-row' : '')

const tcpStateTagType = (state) => {
  const text = String(state || '').toLowerCase()
  if (text === 'established') return 'success'
  if (text === 'listen') return 'warning'
  if (text.includes('time')) return 'info'
  if (riskyStates.has(text)) return 'danger'
  return 'info'
}

const loadInspect = async (row, { silent = false } = {}) => {
  if (!row?.id) return
  inspectLoading.value = true
  try {
    const res = await axios.post(`/api/v1/cmdb/hosts/${row.id}/test`, {}, { headers: authHeaders() })
    if (res.data?.code !== 0) throw new Error(res.data?.message || '检测失败')
    const payload = res.data?.data || {}
    inspectData.value = {
      updatedAt: new Date().toISOString(),
      topCpu: normalizeProcessRows(payload?.processes?.top_cpu),
      topMem: normalizeProcessRows(payload?.processes?.top_mem),
      tcpRows: normalizeTCPRows(payload?.tcp_connections),
      tcpSummary: payload?.tcp_summary || {}
    }
  } catch (e) {
    if (!silent) ElMessage.error(getErrorMessage(e, '加载主机监控详情失败'))
  } finally {
    inspectLoading.value = false
  }
}

const clearInspectAutoTimer = () => {
  if (inspectAutoTimer) {
    window.clearInterval(inspectAutoTimer)
    inspectAutoTimer = null
  }
}

const ensureInspectAutoTimer = () => {
  clearInspectAutoTimer()
  if (!inspectVisible.value || !inspectAutoRefresh.value) return
  inspectAutoTimer = window.setInterval(() => {
    if (document.hidden || inspectLoading.value || !inspectHost.value) return
    loadInspect(inspectHost.value, { silent: true })
  }, 15000)
}

const openInspect = async (row, mode = 'process') => {
  if (!row) return
  inspectHost.value = row
  inspectMode.value = mode
  inspectVisible.value = true
  await loadInspect(row)
  ensureInspectAutoTimer()
}

const refreshInspect = async () => {
  if (!inspectHost.value) return
  await loadInspect(inspectHost.value)
}

const openMonitor = (row) => {
  const keyword = String(row?.ip || row?.name || '').trim()
  router.push({
    path: '/monitor/hosts',
    query: keyword ? { keyword } : {}
  })
}

const handleRowCommand = (row, command) => {
  if (command === 'process') {
    openInspect(row, 'process')
    return
  }
  if (command === 'tcp') {
    openInspect(row, 'tcp')
    return
  }
  if (command === 'monitor') {
    openMonitor(row)
    return
  }
  if (command === 'delete') {
    handleDelete(row)
  }
}

const buildSelector = (base, instance) => {
  const baseText = String(base || '').trim()
  if (!instance) return baseText
  return baseText ? `${baseText},instance="${instance}"` : `instance="${instance}"`
}

const queryCpu = (instance) => `100 - (avg by(instance) (irate(node_cpu_seconds_total{${buildSelector('mode="idle"', instance)}}[5m])) * 100)`
const queryMem = (instance) => `100 * (1 - (node_memory_MemAvailable_bytes{${buildSelector('', instance)}} / node_memory_MemTotal_bytes{${buildSelector('', instance)}}))`
const queryDisk = (instance) => `max by(instance) (100 - (node_filesystem_free_bytes{${buildSelector('fstype!="tmpfs",mountpoint="/"', instance)}} / node_filesystem_size_bytes{${buildSelector('fstype!="tmpfs",mountpoint="/"', instance)}}) * 100)`
const queryLoad = (instance) => `node_load1{${buildSelector('', instance)}}`
const queryNetIn = (instance) => `sum by(instance) (rate(node_network_receive_bytes_total{${buildSelector('', instance)}}[5m])) / 1024`
const queryNetOut = (instance) => `sum by(instance) (rate(node_network_transmit_bytes_total{${buildSelector('', instance)}}[5m])) / 1024`

const calcStep = (hours) => {
  if (hours <= 1) return 30
  if (hours <= 6) return 60
  return 120
}

const fetchPromRange = async (query, start, end, step) => {
  const res = await axios.get('/api/v1/monitor/prometheus/query_range', {
    headers: authHeaders(),
    params: {
      query: query.replace(/\s+/g, ' ').trim(),
      start,
      end,
      step
    }
  })
  if (res.data?.status && res.data.status !== 'success') {
    throw new Error(res.data?.error || 'Prometheus 查询失败')
  }
  return res.data?.data?.result || []
}

const resolveChartEl = (value) => {
  if (!value) return null
  if (value?.$el) return resolveChartEl(value.$el)
  if (value instanceof HTMLElement) return value
  return null
}

const ensureChartInstance = (instance, holderRef) => {
  const el = resolveChartEl(holderRef.value)
  if (!el || !el.isConnected || el.clientWidth <= 0 || el.clientHeight <= 0) {
    if (instance) {
      try { instance.dispose() } catch {}
    }
    return null
  }
  const existing = echarts.getInstanceByDom(el)
  if (existing && existing !== instance) {
    try { existing.dispose() } catch {}
  }
  if (instance && instance.getDom && instance.getDom() !== el) {
    try { instance.dispose() } catch {}
    instance = null
  }
  if (!instance) {
    try {
      instance = echarts.init(el, null, { renderer: 'svg' })
    } catch {
      return null
    }
  }
  return instance
}

const parseRangeSeries = (results) => {
  if (!Array.isArray(results) || !results.length) return { labels: [], data: [] }
  const points = Array.isArray(results[0]?.values) ? results[0].values : []
  return {
    labels: points.map((item) => new Date(Number(item[0]) * 1000).toLocaleTimeString()),
    data: points.map((item) => Number(item[1] || 0))
  }
}

const renderDetailLineChart = (chart, title, labels, values, unit = '%', color = '#3b82f6') => {
  if (!chart) return null
  const option = {
    color: [color],
    tooltip: {
      trigger: 'axis',
      valueFormatter: (val) => `${Number(val || 0).toFixed(2)} ${unit}`
    },
    grid: { left: 42, right: 16, top: 24, bottom: 24 },
    xAxis: {
      type: 'category',
      data: labels,
      boundaryGap: false,
      axisLabel: { fontSize: 11 }
    },
    yAxis: {
      type: 'value',
      axisLabel: { fontSize: 11 }
    },
    series: [
      {
        name: title,
        type: 'line',
        showSymbol: false,
        smooth: true,
        data: values,
        areaStyle: { opacity: 0.15 }
      }
    ]
  }
  try {
    chart.setOption(option, true)
    chart.resize()
    return chart
  } catch {
    return chart
  }
}

const initDetailCharts = async () => {
  await nextTick()
  detailCpuChart = ensureChartInstance(detailCpuChart, detailCpuChartRef)
  detailMemChart = ensureChartInstance(detailMemChart, detailMemChartRef)
  detailDiskChart = ensureChartInstance(detailDiskChart, detailDiskChartRef)
  detailLoadChart = ensureChartInstance(detailLoadChart, detailLoadChartRef)
  detailNetInChart = ensureChartInstance(detailNetInChart, detailNetInChartRef)
  detailNetOutChart = ensureChartInstance(detailNetOutChart, detailNetOutChartRef)
}

const disposeDetailCharts = () => {
  const charts = [detailCpuChart, detailMemChart, detailDiskChart, detailLoadChart, detailNetInChart, detailNetOutChart]
  charts.forEach((chart) => {
    if (!chart) return
    try { chart.dispose() } catch {}
  })
  detailCpuChart = null
  detailMemChart = null
  detailDiskChart = null
  detailLoadChart = null
  detailNetInChart = null
  detailNetOutChart = null
}

const resolveHostInstance = (row) => {
  if (!row) return ''
  const metric = hostMetric(row)
  const direct = String(metric?.instance || '').trim()
  if (direct) return direct

  const ip = normalizeHostAddress(row?.ip)
  if (!ip) return ''

  const candidates = [ip, `${ip}:9100`, `${ip}:9101`, `${ip}:9102`]
  const matched = Object.values(hostMetricsMap.value).find((item) => {
    const inst = String(item?.instance || item?.ip || '').trim()
    return candidates.includes(inst) || normalizeHostAddress(inst) === ip
  })
  return String(matched?.instance || '').trim() || `${ip}:9100`
}

const clearDetailAutoTimer = () => {
  if (detailAutoTimer) {
    window.clearInterval(detailAutoTimer)
    detailAutoTimer = null
  }
}

const ensureDetailAutoTimer = () => {
  clearDetailAutoTimer()
  if (!detailVisible.value || !detailAutoRefresh.value) return
  detailAutoTimer = window.setInterval(() => {
    if (document.hidden || detailLoading.value || !detailHost.value) return
    fetchDetailMetrics({ silent: true })
  }, 30000)
}

const fetchDetailMetrics = async ({ silent = false } = {}) => {
  if (!detailHost.value) return
  const instance = resolveHostInstance(detailHost.value)
  detailInstanceLabel.value = instance
  if (!instance) return

  detailLoading.value = true
  try {
    await initDetailCharts()
    const end = Math.floor(Date.now() / 1000)
    const start = end - detailRangeHours.value * 3600
    const step = calcStep(detailRangeHours.value)

    const settled = await Promise.allSettled([
      fetchPromRange(queryCpu(instance), start, end, step),
      fetchPromRange(queryMem(instance), start, end, step),
      fetchPromRange(queryDisk(instance), start, end, step),
      fetchPromRange(queryLoad(instance), start, end, step),
      fetchPromRange(queryNetIn(instance), start, end, step),
      fetchPromRange(queryNetOut(instance), start, end, step)
    ])

    const cpuSeries = parseRangeSeries(settled[0].status === 'fulfilled' ? settled[0].value : [])
    const memSeries = parseRangeSeries(settled[1].status === 'fulfilled' ? settled[1].value : [])
    const diskSeries = parseRangeSeries(settled[2].status === 'fulfilled' ? settled[2].value : [])
    const loadSeries = parseRangeSeries(settled[3].status === 'fulfilled' ? settled[3].value : [])
    const netInSeries = parseRangeSeries(settled[4].status === 'fulfilled' ? settled[4].value : [])
    const netOutSeries = parseRangeSeries(settled[5].status === 'fulfilled' ? settled[5].value : [])

    detailCpuChart = renderDetailLineChart(detailCpuChart, 'CPU', cpuSeries.labels, cpuSeries.data, '%', '#3b82f6')
    detailMemChart = renderDetailLineChart(detailMemChart, 'MEM', memSeries.labels, memSeries.data, '%', '#f59e0b')
    detailDiskChart = renderDetailLineChart(detailDiskChart, 'DISK', diskSeries.labels, diskSeries.data, '%', '#ef4444')
    detailLoadChart = renderDetailLineChart(detailLoadChart, 'LOAD', loadSeries.labels, loadSeries.data, '', '#10b981')
    detailNetInChart = renderDetailLineChart(detailNetInChart, 'NET IN', netInSeries.labels, netInSeries.data, 'KB/s', '#6366f1')
    detailNetOutChart = renderDetailLineChart(detailNetOutChart, 'NET OUT', netOutSeries.labels, netOutSeries.data, 'KB/s', '#22c55e')

    if (!silent) {
      const failedCount = settled.filter((item) => item.status === 'rejected').length
      if (failedCount > 0 && failedCount < settled.length) {
        ElMessage.warning(`部分趋势图查询失败（${failedCount}/${settled.length}）`)
      }
      if (failedCount === settled.length) {
        ElMessage.error('趋势图查询失败，请检查 Prometheus 指标')
      }
    }
  } catch (e) {
    if (!silent) ElMessage.error(getErrorMessage(e, '加载主机趋势失败'))
  } finally {
    detailLoading.value = false
  }
}

const openDetail = async (row) => {
  if (!row) return
  detailHost.value = row
  detailVisible.value = true
  detailRangeHours.value = 1
  await fetchDetailMetrics()
  ensureDetailAutoTimer()
}

const onResize = () => {
  const charts = [detailCpuChart, detailMemChart, detailDiskChart, detailLoadChart, detailNetInChart, detailNetOutChart]
  charts.forEach((chart) => {
    if (!chart) return
    try { chart.resize() } catch {}
  })
}

watch([inspectVisible, inspectAutoRefresh], ensureInspectAutoTimer)
watch([detailVisible, detailAutoRefresh], ensureDetailAutoTimer)
watch(detailVisible, (visible) => {
  if (!visible) {
    clearDetailAutoTimer()
    detailHost.value = null
    detailInstanceLabel.value = ''
  }
})

onMounted(async () => {
  await fetchGroups()
  await syncStatuses(true)
  nextTick(() => {
    groupTreeRef.value?.setCurrentKey('all')
  })

  autoSyncTimer = window.setInterval(() => {
    if (document.hidden || syncingStatus.value || loading.value) return
    syncStatuses(true)
  }, 60 * 1000)

  window.addEventListener('resize', onResize)
})

onBeforeUnmount(() => {
  if (autoSyncTimer) {
    window.clearInterval(autoSyncTimer)
    autoSyncTimer = null
  }
  clearInspectAutoTimer()
  clearDetailAutoTimer()
  disposeDetailCharts()
  window.removeEventListener('resize', onResize)
})
</script>

<style scoped>
.flex { display: flex; }
.justify-between { justify-content: space-between; }
.items-center { align-items: center; }
.gap-2 { gap: 8px; }
.font-bold { font-weight: 600; }
.mb-4 { margin-bottom: 16px; }
.w-64 { width: 256px; }

.host-page-card { min-height: 400px; }
.header-wrap { flex-wrap: wrap; gap: 10px; }
.header-actions { display: flex; gap: 8px; flex-wrap: wrap; justify-content: flex-end; }

.host-layout {
  display: grid;
  grid-template-columns: 280px minmax(0, 1fr);
  gap: 14px;
}

.host-aside {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-width: 0;
}

.group-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-weight: 600;
}

.group-card-header-actions {
  display: flex;
  align-items: center;
  gap: 6px;
}

.group-search { margin-bottom: 10px; }

.group-tree-node {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding-right: 8px;
}

.group-tree-label {
  max-width: 160px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.provider-total {
  font-weight: 600;
  color: var(--el-text-color-secondary);
}

.provider-grid {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.provider-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  background: var(--el-fill-color-extra-light);
  border-radius: 8px;
}

.provider-name {
  font-size: 13px;
  color: var(--el-text-color-regular);
}

.host-main { min-width: 0; }
.filters-row { flex-wrap: wrap; }

.table-scroll {
  overflow-x: auto;
  padding-bottom: 2px;
}

.op-row {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 6px;
}

.group-manage-toolbar {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-bottom: 12px;
}

.detail-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  margin-bottom: 14px;
}

.detail-host-info {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.detail-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.detail-metrics {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
  margin-bottom: 12px;
}

.metric-card {
  background: var(--el-fill-color-extra-light);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 10px;
  padding: 10px 12px;
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.metric-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.detail-chart-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.chart-card {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 10px;
  padding: 10px;
  background: #fff;
}

.chart-title {
  font-weight: 600;
  font-size: 13px;
  margin-bottom: 6px;
}

.detail-chart {
  height: 200px;
  width: 100%;
}

.import-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 12px;
}

.test-block {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.test-title { font-weight: 600; }

.test-pre {
  background: #0f172a;
  color: #e2e8f0;
  padding: 12px;
  border-radius: 6px;
  overflow: auto;
  max-height: 200px;
}

.helper-row {
  margin-top: 6px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.4;
}

.inspect-toolbar {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
  align-items: center;
  flex-wrap: wrap;
}

.inspect-host {
  display: flex;
  gap: 12px;
  align-items: center;
  color: var(--el-text-color-secondary);
  font-size: 13px;
  flex-wrap: wrap;
}

.inspect-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}

.inspect-process-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.inspect-card :deep(.el-card__header) {
  padding: 10px 14px;
  font-weight: 600;
}

.inspect-card :deep(.el-card__body) {
  padding: 10px 12px;
}

.mini-percent {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 3px;
}

.inspect-tcp-block {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.tcp-summary {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

:deep(.danger-row td) {
  background: rgba(245, 108, 108, 0.08);
}

:deep(.warn-row td) {
  background: rgba(230, 162, 60, 0.08);
}

@media (max-width: 1440px) {
  .host-layout { grid-template-columns: 250px minmax(0, 1fr); }
}

@media (max-width: 1280px) {
  .detail-metrics { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}

@media (max-width: 1100px) {
  .host-layout { grid-template-columns: 1fr; }
  .detail-chart-grid { grid-template-columns: 1fr; }
  .inspect-process-grid { grid-template-columns: 1fr; }
}
</style>
