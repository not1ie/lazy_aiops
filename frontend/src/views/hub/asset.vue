<template>
  <div class="hub-container" v-loading="loading">
    <header class="hub-header">
      <div class="hub-title">
        <h2>资产与安全中心</h2>
        <p>一站式管理主机、网络、数据库及云端资源，彻底消除跨页面跳转。</p>
      </div>
      <div class="hub-actions">
        <el-button round icon="Refresh" @click="refreshAll">同步状态</el-button>
        <el-button type="primary" round icon="Plus" @click="handleGlobalAdd">新增资产</el-button>
      </div>
    </header>

    <div class="hub-summary">
      <div class="summary-item" @click="activePanel = 'hosts'">
        <div class="summary-label">主机在线率</div>
        <div class="summary-value" :class="hostOnlineRate > 90 ? 'success' : 'warning'">{{ hostOnlineRate }}%</div>
      </div>
      <div class="summary-item" @click="activePanel = 'database'">
        <div class="summary-label">风险数据库</div>
        <div class="summary-value" :class="stats.databaseRisk === 0 ? 'success' : 'danger'">{{ stats.databaseRisk }}</div>
      </div>
      <div class="summary-item" @click="activePanel = 'firewall'">
        <div class="summary-label">防火墙风险</div>
        <div class="summary-value" :class="stats.firewallRisk === 0 ? 'success' : 'danger'">{{ stats.firewallRisk }}</div>
      </div>
      <div class="summary-item" @click="activePanel = 'jump'">
        <div class="summary-label">堡垒机同步</div>
        <div class="summary-value" :class="jumpIntegrationStatusMeta.type">{{ jumpIntegrationStatusMeta.text }}</div>
      </div>
    </div>

    <div class="hub-workspace full-width">
      <div class="workspace-content">
        <el-tabs v-model="activePanel" class="hub-tabs">
          <el-tab-pane label="主机资产" name="hosts">
            <div class="pane-header">
              <el-input v-model="panelKeyword" placeholder="筛选 IP / 名称 / 分组..." style="width: 240px" clearable />
              <el-button link type="primary" @click="go('/host')">全屏管理</el-button>
            </div>
            <div v-if="filteredHosts.length === 0 && !loading" class="empty-cta">
              <el-empty description="暂无主机资产" :image-size="60">
                <el-button type="primary" @click="handleGlobalAdd">添加第一台主机</el-button>
              </el-empty>
            </div>
            <el-table v-else :data="filteredHosts" class="hub-table" size="small">
              <el-table-column prop="name" label="主机名" min-width="130" />
              <el-table-column prop="ip" label="IP" width="130" />
              <el-table-column label="状态" width="130">
                <template #default="{ row }">
                  <div style="display:flex;flex-direction:column;gap:2px">
                    <StatusBadge v-bind="cmdbHostStatusMeta(row)" />
                    <span class="status-detail" v-if="cmdbHostStatusMeta(row).source">{{ cmdbHostStatusMeta(row).source }}</span>
                  </div>
                </template>
              </el-table-column>
              <el-table-column label="最后检测" width="120">
                <template #default="{ row }">
                  <span class="check-time" :class="{ stale: isHostStale(row) }">{{ fmtCheckTime(row.last_check_at) }}</span>
                </template>
              </el-table-column>
              <el-table-column prop="os" label="OS" width="100" />
              <el-table-column label="规格" width="130">
                <template #default="{ row }">
                  <el-tooltip v-if="!row.cpu_count && !row.memory_gb" placement="top" effect="dark" raw-content>
                    <template #content>
                      <div>Agent 未上报或采集失败</div>
                      <div v-if="row.status_reason" style="margin-top:4px;opacity:0.8">原因：{{ row.status_reason }}</div>
                      <div v-if="row.last_check_at" style="opacity:0.7;font-size:11px">最后检测：{{ new Date(row.last_check_at).toLocaleString() }}</div>
                    </template>
                    <span class="spec-tag spec-missing">未采集</span>
                  </el-tooltip>
                  <span v-else class="spec-tag">{{ row.cpu_count || '-' }}C / {{ row.memory_gb || '-' }}G</span>
                </template>
              </el-table-column>
              <el-table-column label="操作" width="160" fixed="right">
                <template #default="{ row }">
                  <el-button link type="primary" @click="openTerminal(row)">终端</el-button>
                  <el-button link type="primary" @click="handleEditHost(row)">编辑</el-button>
                  <el-button link @click="goAIAnalysis({ type: 'host', title: row.name, id: row.id, summary: `主机 ${row.name} (${row.ip})，状态: ${hostStatusTag(row).text}` })">
                    <el-icon><MagicStick /></el-icon>
                  </el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>

          <el-tab-pane label="凭据管理" name="credentials">
             <el-table :data="credentials" class="hub-table" size="small">
               <el-table-column prop="name" label="凭据名称" />
               <el-table-column prop="type" label="类型" />
               <el-table-column prop="username" label="用户名" />
               <el-table-column label="最后更新" prop="updated_at" :formatter="formatTime" />
             </el-table>
          </el-tab-pane>

          <el-tab-pane label="数据库" name="database">
            <el-table :data="filteredDatabases" class="hub-table" size="small">
              <el-table-column prop="name" label="数据库名" />
              <el-table-column prop="type" label="类型" />
              <el-table-column label="地址">
                <template #default="{ row }">{{ row.host }}:{{ row.port }}</template>
              </el-table-column>
              <el-table-column label="状态"><template #default="{ row }"><StatusBadge v-bind="databaseStatusTag(row)" /></template></el-table-column>
            </el-table>
          </el-tab-pane>

          <el-tab-pane label="网络/防火墙" name="firewall">
             <el-table :data="filteredNetworkDevices" class="hub-table" size="small">
               <el-table-column prop="name" label="设备名" />
               <el-table-column prop="ip" label="管理IP" />
               <el-table-column label="状态"><template #default="{ row }"><StatusBadge v-bind="networkStatusTag(row)" /></template></el-table-column>
             </el-table>
          </el-tab-pane>
        </el-tabs>
      </div>
    </div>

    <!-- Quick Create Host Dialog -->
    <el-dialog title="快捷新增主机" v-model="quickCreateVisible" width="480px" append-to-body>
      <el-form :model="quickForm" label-width="80px">
        <el-form-item label="主机名" required><el-input v-model="quickForm.name" placeholder="如 web-01" /></el-form-item>
        <el-form-item label="IP 地址" required><el-input v-model="quickForm.ip" placeholder="如 192.168.1.10" /></el-form-item>
        <el-form-item label="SSH 端口"><el-input-number v-model="quickForm.port" :min="1" :max="65535" /></el-form-item>
        <el-form-item label="操作系统">
          <el-select v-model="quickForm.os" style="width: 100%">
            <el-option label="Linux" value="Linux" />
            <el-option label="Windows" value="Windows" />
            <el-option label="macOS" value="macOS" />
          </el-select>
        </el-form-item>
        <el-form-item label="分组"><el-input v-model="quickForm.group" placeholder="如 production（可选）" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="quickCreateVisible = false">取消</el-button>
        <el-button type="primary" :loading="quickCreating" @click="submitQuickCreate">确认添加</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import StatusBadge from '@/components/common/StatusBadge.vue'
import {
  cmdbHostStatusMeta,
  databaseAssetStatusMeta,
  jumpIntegrationSyncStatusMeta
} from '@/utils/status'
import './hub-common.css'
import { useAIChat } from '@/composables/useAIChat'

const router = useRouter()
const { goAIAnalysis } = useAIChat()
const loading = ref(false)
const hosts = ref([])
const databases = ref([])
const cloudResources = ref([])
const credentials = ref([])
const networkDevices = ref([])
const firewallDevices = ref([])
const jumpIntegration = ref(null)
const activePanel = ref('hosts')
const panelKeyword = ref('')

const stats = reactive({
  hostTotal: 0,
  hostOnline: 0,
  databaseTotal: 0,
  databaseRisk: 0,
  firewallRisk: 0
})

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })
const go = (path) => router.push(path)
const formatTime = (row, col, val) => val ? new Date(val).toLocaleString() : '-'

const hostOnlineRate = computed(() => stats.hostTotal ? Math.round((stats.hostOnline / stats.hostTotal) * 100) : 0)

const hostStatusTag = (row) => cmdbHostStatusMeta(row)
const databaseStatusTag = (row) => databaseAssetStatusMeta(row)
const networkStatusTag = (row) => ({ text: row.status === 1 ? '在线' : '离线', type: row.status === 1 ? 'success' : 'danger' })

const fmtCheckTime = (val) => {
  if (!val) return '从未检测'
  const diff = Math.floor((Date.now() - new Date(val).getTime()) / 1000)
  if (diff < 60) return '刚刚'
  if (diff < 3600) return `${Math.floor(diff / 60)} 分钟前`
  if (diff < 86400) return `${Math.floor(diff / 3600)} 小时前`
  return `${Math.floor(diff / 86400)} 天前`
}
const isHostStale = (row) => {
  if (!row.last_check_at) return true
  return Date.now() - new Date(row.last_check_at).getTime() > 5 * 60 * 1000
}

const jumpIntegrationStatusMeta = computed(() =>
  jumpIntegrationSyncStatusMeta(jumpIntegration.value?.last_sync_status, {
    enabled: jumpIntegration.value?.enabled !== false,
    lastSyncAt: jumpIntegration.value?.last_sync_at,
    nowMs: Date.now()
  })
)

const filteredHosts = computed(() => {
  const kw = panelKeyword.value.toLowerCase()
  return hosts.value.filter(h => !kw || h.name.toLowerCase().includes(kw) || h.ip.includes(kw))
})

const filteredDatabases = computed(() => databases.value)
const filteredNetworkDevices = computed(() => networkDevices.value)

const refreshAll = async () => {
  loading.value = true
  try {
    const res = await Promise.all([
      axios.get('/api/v1/cmdb/hosts', { headers: authHeaders() }),
      axios.get('/api/v1/cmdb/databases', { headers: authHeaders() }),
      axios.get('/api/v1/cmdb/credentials', { headers: authHeaders() }),
      axios.get('/api/v1/cmdb/network-devices', { headers: authHeaders() }),
      axios.get('/api/v1/jump/integration/config', { headers: authHeaders() })
    ])
    hosts.value = res[0].data?.data || []
    databases.value = res[1].data?.data || []
    credentials.value = res[2].data?.data || []
    networkDevices.value = res[3].data?.data || []
    jumpIntegration.value = res[4].data?.data

    stats.hostTotal = hosts.value.length
    stats.hostOnline = hosts.value.filter(h => h.status === 1).length
    stats.databaseRisk = databases.value.filter(d => d.status === 0).length
  } catch (err) {
    ElMessage.error('加载资产中心失败')
  } finally {
    loading.value = false
  }
}

const openTerminal = (row) => router.push(`/terminal?host_id=${row.id}`)
const handleEditHost = (row) => router.push('/host')

// --- Quick Create Host ---
const quickCreateVisible = ref(false)
const quickCreating = ref(false)
const quickForm = ref({ name: '', ip: '', port: 22, os: 'Linux', group: '' })

const handleGlobalAdd = () => {
  quickForm.value = { name: '', ip: '', port: 22, os: 'Linux', group: '' }
  quickCreateVisible.value = true
}

const submitQuickCreate = async () => {
  if (!quickForm.value.name || !quickForm.value.ip) {
    ElMessage.warning('主机名和 IP 为必填项')
    return
  }
  quickCreating.value = true
  try {
    await axios.post('/api/v1/cmdb/hosts', {
      name: quickForm.value.name,
      ip: quickForm.value.ip,
      port: quickForm.value.port,
      os: quickForm.value.os,
      group_name: quickForm.value.group || undefined
    }, { headers: authHeaders() })
    ElMessage.success('主机添加成功')
    quickCreateVisible.value = false
    refreshAll()
  } catch (err) {
    ElMessage.error('添加失败: ' + (err.response?.data?.message || err.message))
  } finally {
    quickCreating.value = false
  }
}

onMounted(refreshAll)
</script>

<style scoped>
.full-width { grid-template-columns: 1fr; }
.pane-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.spec-tag { font-family: monospace; font-size: 11px; background: rgba(0,0,0,0.05); padding: 2px 6px; border-radius: 4px; }
.spec-tag.spec-missing { color: var(--el-color-warning); background: rgba(245,158,11,0.1); cursor: help; }
.status-detail { font-size: 11px; color: var(--el-text-color-secondary); }
.check-time { font-size: 12px; color: var(--el-text-color-secondary); }
.check-time.stale { color: var(--el-color-warning); font-weight: 600; }
.empty-cta { padding: 40px 0; }
</style>
