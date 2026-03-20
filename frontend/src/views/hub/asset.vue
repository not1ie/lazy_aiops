<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>资产总览</h2>
        <p class="page-desc">把主机、网络设备、数据库、云资源、防火墙与堡垒机资产融合在一个工作台里。</p>
      </div>
      <div class="page-actions">
        <el-button :loading="loading" icon="Refresh" @click="refreshAll">刷新</el-button>
      </div>
    </div>

    <el-row :gutter="12" class="summary-row">
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">主机总数</div><div class="metric-value">{{ stats.hostTotal }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">在线主机</div><div class="metric-value ok">{{ stats.hostOnline }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">网络设备</div><div class="metric-value">{{ stats.networkDeviceTotal }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">数据库资产</div><div class="metric-value">{{ stats.databaseTotal }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">云资源</div><div class="metric-value">{{ stats.cloudResourceTotal }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">凭据</div><div class="metric-value">{{ stats.credentialTotal }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">防火墙设备</div><div class="metric-value">{{ stats.firewallDeviceTotal }}</div></el-card>
      </el-col>
      <el-col :xl="3" :lg="6" :md="6" :sm="12" :xs="12">
        <el-card><div class="metric-title">堡垒机资产</div><div class="metric-value">{{ stats.jumpAssetTotal }}</div></el-card>
      </el-col>
    </el-row>

    <el-row :gutter="12">
      <el-col :span="10">
        <el-card>
          <template #header>资产健康</template>
          <div class="health-row">
            <span>主机在线率</span>
            <strong>{{ hostOnlineRate }}%</strong>
          </div>
          <el-progress :percentage="hostOnlineRate" :stroke-width="14" />
          <div class="health-row mtop">
            <span>网络设备在线率</span>
            <strong>{{ networkOnlineRate }}%</strong>
          </div>
          <el-progress :percentage="networkOnlineRate" :stroke-width="14" status="success" />
          <el-divider />
          <div class="health-row"><span>资产分组</span><strong>{{ stats.groupTotal }}</strong></div>
          <div class="health-row"><span>风险防火墙</span><strong>{{ stats.firewallRisk }}</strong></div>
        </el-card>
      </el-col>
      <el-col :span="14">
        <el-card>
          <template #header>最近资产变更主机</template>
          <el-table :fit="true" :data="recentHosts" size="small" max-height="300">
            <el-table-column prop="name" label="主机名" min-width="130" />
            <el-table-column prop="ip" label="IP" min-width="130" />
            <el-table-column label="状态" width="100">
              <template #default="{ row }">
                <el-tag :type="isOnline(row.status) ? 'success' : 'info'">{{ isOnline(row.status) ? '在线' : '离线' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="os" label="系统" min-width="120" />
            <el-table-column label="更新时间" min-width="170">
              <template #default="{ row }">{{ formatTime(row.updated_at) }}</template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <el-card class="integration-card">
      <template #header>
        <div class="integration-header">
          <div class="integration-title-wrap">
            <span>资产融合视图</span>
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
              placeholder="筛选名称、IP、类型、区域..."
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
        <el-tab-pane label="主机" name="hosts">
          <el-table :fit="true" :data="filteredHosts" size="small" max-height="360" empty-text="暂无主机数据">
            <el-table-column prop="name" label="主机名" min-width="180" />
            <el-table-column prop="ip" label="IP" min-width="140" />
            <el-table-column prop="os" label="系统" min-width="140" />
            <el-table-column label="状态" width="100">
              <template #default="{ row }">
                <el-tag :type="isOnline(row.status) ? 'success' : 'info'">{{ isOnline(row.status) ? '在线' : '离线' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="更新时间" min-width="170">
              <template #default="{ row }">{{ formatTime(row.updated_at) }}</template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="网络设备" name="network">
          <el-table :fit="true" :data="filteredNetworkDevices" size="small" max-height="360" empty-text="暂无网络设备数据">
            <el-table-column prop="name" label="设备名" min-width="160" />
            <el-table-column prop="device_type" label="类型" width="120" />
            <el-table-column prop="vendor" label="厂商" width="120" />
            <el-table-column prop="ip" label="管理IP" min-width="140" />
            <el-table-column label="状态" width="100">
              <template #default="{ row }">
                <el-tag :type="isOnline(row.status) ? 'success' : 'warning'">{{ isOnline(row.status) ? '在线' : '离线' }}</el-tag>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="数据库资产" name="database">
          <el-table :fit="true" :data="filteredDatabases" size="small" max-height="360" empty-text="暂无数据库资产数据">
            <el-table-column prop="name" label="名称" min-width="160" />
            <el-table-column prop="type" label="类型" width="120" />
            <el-table-column label="地址" min-width="180">
              <template #default="{ row }">{{ row.host }}:{{ row.port }}</template>
            </el-table-column>
            <el-table-column prop="database" label="库名" min-width="120" />
            <el-table-column prop="environment" label="环境" width="100" />
            <el-table-column prop="owner" label="负责人" width="120" />
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="云资源" name="cloud">
          <el-table :fit="true" :data="filteredCloudResources" size="small" max-height="360" empty-text="暂无云资源数据">
            <el-table-column prop="name" label="资源名称" min-width="180" />
            <el-table-column prop="type" label="类型" width="110" />
            <el-table-column prop="region" label="区域" min-width="120" />
            <el-table-column prop="ip" label="IP" min-width="140" />
            <el-table-column label="账号" min-width="140">
              <template #default="{ row }">{{ row.account?.name || '-' }}</template>
            </el-table-column>
            <el-table-column prop="status" label="状态" width="110" />
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="防火墙" name="firewall">
          <el-table :fit="true" :data="filteredFirewalls" size="small" max-height="360" empty-text="暂无防火墙数据">
            <el-table-column prop="name" label="名称" min-width="150" />
            <el-table-column prop="vendor" label="厂商" min-width="120" />
            <el-table-column prop="ip" label="IP" min-width="140" />
            <el-table-column label="状态" width="110">
              <template #default="{ row }">
                <el-tag :type="firewallStatus(row.status).type">{{ firewallStatus(row.status).text }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="最近检查" min-width="170">
              <template #default="{ row }">{{ formatTime(row.last_check_at) }}</template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="堡垒机资产" name="jump">
          <el-table :fit="true" :data="filteredJumpAssets" size="small" max-height="360" empty-text="暂无堡垒机资产数据">
            <el-table-column prop="name" label="名称" min-width="170" />
            <el-table-column prop="asset_type" label="类型" width="110" />
            <el-table-column prop="protocol" label="协议" width="110" />
            <el-table-column label="地址" min-width="170">
              <template #default="{ row }">{{ row.address }}:{{ row.port }}</template>
            </el-table-column>
            <el-table-column prop="source" label="来源" width="120" />
            <el-table-column label="状态" width="100">
              <template #default="{ row }">
                <el-tag :type="jumpAssetStatus(row).type">{{ jumpAssetStatus(row).text }}</el-tag>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="WebTerminal会话" name="terminal">
          <el-table :fit="true" :data="filteredTerminalSessions" size="small" max-height="360" empty-text="暂无会话数据">
            <el-table-column prop="operator" label="操作人" min-width="120" />
            <el-table-column prop="host" label="主机" min-width="170" />
            <el-table-column prop="username" label="登录用户" min-width="120" />
            <el-table-column prop="port" label="端口" width="90" />
            <el-table-column label="状态" width="100">
              <template #default="{ row }">
                <el-tag :type="terminalSessionStatus(row.status).type">{{ terminalSessionStatus(row.status).text }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="开始时间" min-width="170">
              <template #default="{ row }">{{ formatTime(row.started_at || row.created_at) }}</template>
            </el-table-column>
            <el-table-column prop="last_error" label="失败原因" min-width="180" show-overflow-tooltip>
              <template #default="{ row }">{{ row.last_error || '-' }}</template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </el-card>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'
import { ElMessage } from 'element-plus'

const router = useRouter()
const loading = ref(false)
const hosts = ref([])
const networkDevices = ref([])
const databases = ref([])
const cloudResources = ref([])
const credentials = ref([])
const groups = ref([])
const firewallDevices = ref([])
const jumpAssets = ref([])
const terminalSessions = ref([])
const activePanel = ref('hosts')
const panelKeyword = ref('')

const stats = reactive({
  hostTotal: 0,
  hostOnline: 0,
  networkDeviceTotal: 0,
  networkDeviceOnline: 0,
  databaseTotal: 0,
  cloudResourceTotal: 0,
  credentialTotal: 0,
  groupTotal: 0,
  firewallDeviceTotal: 0,
  firewallRisk: 0,
  jumpAssetTotal: 0,
  terminalSessionTotal: 0
})


const panelRouteMap = {
  hosts: '/host',
  network: '/cmdb/network-devices',
  database: '/cmdb/database',
  cloud: '/cmdb/cloud',
  firewall: '/firewall',
  jump: '/jump/assets',
  terminal: '/terminal'
}

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })
const go = (path) => router.push(path)

const isOnline = (status) => status === 1 || status === 'online' || status === '在线' || status === true

const firewallStatus = (status) => {
  const value = Number(status)
  if (value === 1) return { text: '正常', type: 'success' }
  if (value === 2) return { text: '告警', type: 'danger' }
  return { text: '离线', type: 'info' }
}

const jumpAssetStatus = (row) => {
  if (row?.enabled === false || row?.status === 0 || String(row?.status || '').toLowerCase() === 'offline') {
    return { text: '禁用', type: 'info' }
  }
  return { text: '可用', type: 'success' }
}

const terminalSessionStatus = (status) => {
  const value = Number(status)
  if (value === 1) return { text: '在线', type: 'success' }
  if (value === 2) return { text: '已关闭', type: 'info' }
  if (value === 3) return { text: '失败', type: 'danger' }
  return { text: '待连', type: 'warning' }
}

const normalizeText = (value) => String(value ?? '').trim().toLowerCase()

const filterRows = (rows, fields) => {
  const keyword = normalizeText(panelKeyword.value)
  const base = Array.isArray(rows) ? rows : []
  if (!keyword) return base.slice(0, 20)
  return base
    .filter((row) => fields.some((field) => normalizeText(field(row)).includes(keyword)))
    .slice(0, 20)
}

const hostOnlineRate = computed(() => {
  if (!stats.hostTotal) return 0
  return Math.round((stats.hostOnline / stats.hostTotal) * 100)
})

const networkOnlineRate = computed(() => {
  if (!stats.networkDeviceTotal) return 0
  return Math.round((stats.networkDeviceOnline / stats.networkDeviceTotal) * 100)
})

const recentHosts = computed(() => {
  return [...hosts.value]
    .sort((a, b) => new Date(b.updated_at || 0).getTime() - new Date(a.updated_at || 0).getTime())
    .slice(0, 8)
})

const filteredHosts = computed(() =>
  filterRows(hosts.value, [(row) => row.name, (row) => row.ip, (row) => row.os, (row) => row.group])
)

const filteredNetworkDevices = computed(() =>
  filterRows(networkDevices.value, [(row) => row.name, (row) => row.device_type, (row) => row.ip, (row) => row.vendor])
)

const filteredDatabases = computed(() =>
  filterRows(databases.value, [(row) => row.name, (row) => row.type, (row) => row.host, (row) => row.database, (row) => row.owner])
)

const filteredCloudResources = computed(() =>
  filterRows(cloudResources.value, [(row) => row.name, (row) => row.type, (row) => row.ip, (row) => row.region, (row) => row.account?.name])
)

const filteredFirewalls = computed(() =>
  filterRows(firewallDevices.value, [(row) => row.name, (row) => row.vendor, (row) => row.ip])
)

const filteredJumpAssets = computed(() =>
  filterRows(jumpAssets.value, [(row) => row.name, (row) => row.asset_type, (row) => row.protocol, (row) => row.address, (row) => row.source])
)

const filteredTerminalSessions = computed(() =>
  filterRows(terminalSessions.value, [(row) => row.operator, (row) => row.host, (row) => row.username, (row) => row.last_error, (row) => row.session_no])
)

const panelOptions = computed(() => [
  { name: 'hosts', label: '主机', count: hosts.value.length },
  { name: 'network', label: '网络设备', count: networkDevices.value.length },
  { name: 'database', label: '数据库资产', count: databases.value.length },
  { name: 'cloud', label: '云资源', count: cloudResources.value.length },
  { name: 'firewall', label: '防火墙', count: firewallDevices.value.length },
  { name: 'jump', label: '堡垒机资产', count: jumpAssets.value.length },
  { name: 'terminal', label: 'WebTerminal会话', count: terminalSessions.value.length }
])

const activePanelMeta = computed(
  () => panelOptions.value.find((item) => item.name === activePanel.value) || panelOptions.value[0] || { label: '-', count: 0 }
)

const formatTime = (value) => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString()
}

const fetchList = async (url) => {
  const res = await axios.get(url, { headers: authHeaders() })
  return Array.isArray(res.data?.data) ? res.data.data : []
}

const openCurrentPanel = () => {
  go(panelRouteMap[activePanel.value] || '/host')
}

const safeData = (result) => (result.status === 'fulfilled' && Array.isArray(result.value) ? result.value : [])

const refreshAll = async () => {
  loading.value = true
  try {
    const settled = await Promise.allSettled([
      fetchList('/api/v1/cmdb/hosts'),
      fetchList('/api/v1/cmdb/groups'),
      fetchList('/api/v1/cmdb/credentials'),
      fetchList('/api/v1/cmdb/databases'),
      fetchList('/api/v1/cmdb/cloud/resources'),
      fetchList('/api/v1/cmdb/network-devices'),
      fetchList('/api/v1/firewall/devices'),
      fetchList('/api/v1/jump/assets'),
      fetchList('/api/v1/terminal/sessions')
    ])

    const [hostList, groupList, credentialList, databaseList, cloudResourceList, networkList, firewallList, jumpAssetList, sessionList] = settled.map(safeData)

    hosts.value = hostList
    groups.value = groupList
    credentials.value = credentialList
    databases.value = databaseList
    cloudResources.value = cloudResourceList
    networkDevices.value = networkList
    firewallDevices.value = firewallList
    jumpAssets.value = jumpAssetList
    terminalSessions.value = sessionList

    stats.hostTotal = hostList.length
    stats.hostOnline = hostList.filter((item) => isOnline(item.status)).length
    stats.groupTotal = groupList.length
    stats.credentialTotal = credentialList.length
    stats.databaseTotal = databaseList.length
    stats.cloudResourceTotal = cloudResourceList.length
    stats.networkDeviceTotal = networkList.length
    stats.networkDeviceOnline = networkList.filter((item) => isOnline(item.status)).length
    stats.firewallDeviceTotal = firewallList.length
    stats.firewallRisk = firewallList.filter((item) => firewallStatus(item.status).type === 'danger').length
    stats.jumpAssetTotal = jumpAssetList.length
    stats.terminalSessionTotal = sessionList.length

    const failed = settled.filter((item) => item.status === 'rejected').length
    if (failed > 0) {
      ElMessage.warning(`部分资产源加载失败（${failed}项），已展示可用数据`)
    }
  } catch (err) {
    ElMessage.error(err?.response?.data?.message || err?.message || '加载资产总览失败')
  } finally {
    loading.value = false
  }
}

onMounted(refreshAll)
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 12px; gap: 12px; }
.page-desc { color: var(--muted-text); margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; }
.summary-row { margin-bottom: 12px; }
.summary-row :deep(.el-card) { margin-bottom: 8px; }
.metric-title { color: var(--muted-text); font-size: 12px; }
.metric-value { font-size: 20px; font-weight: 600; margin-top: 6px; color: var(--el-text-color-primary); }
.ok { color: #67c23a; }
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

@media (max-width: 1100px) {
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
}
</style>
