<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>Agent 心跳</h2>
        <p class="page-desc">采集器在线状态与心跳时间。</p>
      </div>
      <div class="page-actions">
        <el-tag type="success" effect="plain">在线 {{ onlineCount }}</el-tag>
        <el-tag type="warning" effect="plain">状态过期 {{ staleCount }}</el-tag>
        <el-tag type="danger" effect="plain">离线 {{ offlineCount }}</el-tag>
        <el-tag effect="plain">未知 {{ unknownCount }}</el-tag>
        <el-button icon="Refresh" @click="fetchAgents">刷新</el-button>
      </div>
    </div>

    <el-table :fit="true" :data="agents" v-loading="loading" stripe style="width: 100%">
      <el-table-column prop="agent_id" label="Agent ID" min-width="160" />
      <el-table-column prop="hostname" label="主机名" min-width="160" />
      <el-table-column prop="ip" label="IP" width="140" />
      <el-table-column prop="cpu" label="CPU(%)" width="110">
        <template #default="scope">
          <el-tag :type="tagType(scope.row.cpu)">{{ fmt(scope.row.cpu) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="memory" label="内存(%)" width="110">
        <template #default="scope">
          <el-tag :type="tagType(scope.row.memory)">{{ fmt(scope.row.memory) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="disk" label="磁盘(%)" width="110">
        <template #default="scope">
          <el-tag :type="tagType(scope.row.disk)">{{ fmt(scope.row.disk) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="net_in" label="入流量" width="120">
        <template #default="scope">
          <span>{{ fmt(scope.row.net_in) }}</span>
        </template>
      </el-table-column>
      <el-table-column prop="net_out" label="出流量" width="120">
        <template #default="scope">
          <span>{{ fmt(scope.row.net_out) }}</span>
        </template>
      </el-table-column>
      <el-table-column prop="version" label="版本" width="120" />
      <el-table-column prop="os" label="OS" min-width="160" />
      <el-table-column prop="status" label="状态" width="120">
        <template #default="scope">
          <StatusBadge v-bind="agentStatusBadge(scope.row)" />
        </template>
      </el-table-column>
      <el-table-column prop="last_seen" label="最后心跳" min-width="180">
        <template #default="scope">{{ formatTime(scope.row.last_seen) }}</template>
      </el-table-column>
      <el-table-column label="状态说明" min-width="180" show-overflow-tooltip>
        <template #default="scope">{{ statusReason(scope.row) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="120">
        <template #default="scope">
          <el-button size="small" @click="openDetail(scope.row)">详情</el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-card>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import { getErrorMessage } from '@/utils/error'
import { monitorAgentStatusMeta } from '@/utils/status'
import StatusBadge from '@/components/common/StatusBadge.vue'

const agents = ref([])
const loading = ref(false)
const nowTick = ref(Date.now())
let statusTicker = null
let autoRefreshTicker = null
const router = useRouter()
const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })
const agentStatusMeta = (row) => monitorAgentStatusMeta(row, { staleMinutes: 3, nowMs: nowTick.value })
const onlineCount = computed(() => agents.value.filter((item) => agentStatusMeta(item).key === 'online').length)
const staleCount = computed(() => agents.value.filter((item) => agentStatusMeta(item).key === 'stale').length)
const offlineCount = computed(() => agents.value.filter((item) => agentStatusMeta(item).key === 'offline').length)
const unknownCount = computed(() => agents.value.filter((item) => agentStatusMeta(item).key === 'unknown').length)

const fetchAgents = async () => {
  if (loading.value) return
  loading.value = true
  try {
    const res = await axios.get('/api/v1/monitor/agents', { headers: authHeaders() })
    agents.value = res.data.data || []
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '加载 Agent 列表失败'))
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchAgents()
  statusTicker = window.setInterval(() => {
    nowTick.value = Date.now()
  }, 60 * 1000)
  autoRefreshTicker = window.setInterval(() => {
    if (document.hidden || loading.value) return
    fetchAgents()
  }, 60 * 1000)
})

onUnmounted(() => {
  if (statusTicker) {
    window.clearInterval(statusTicker)
    statusTicker = null
  }
  if (autoRefreshTicker) {
    window.clearInterval(autoRefreshTicker)
    autoRefreshTicker = null
  }
})

const openDetail = (row) => {
  router.push({ path: '/monitor/agents/detail', query: { id: row.agent_id } })
}

const tagType = (val) => {
  if (val === undefined || val === null || Number.isNaN(val)) return 'info'
  if (val >= 80) return 'danger'
  if (val >= 60) return 'warning'
  return 'success'
}

const fmt = (val) => {
  if (val === undefined || val === null || Number.isNaN(val)) return '-'
  return Number(val).toFixed(1)
}

const formatTime = (value) => {
  if (!value) return '-'
  const ts = new Date(value).getTime()
  if (Number.isNaN(ts)) return '-'
  return new Date(ts).toLocaleString()
}

const statusReason = (row) => {
  const meta = agentStatusMeta(row)
  if (meta.key === 'online') return '心跳正常'
  if (meta.key === 'stale') return '超过 3 分钟未上报心跳'
  if (meta.key === 'offline') return 'Agent 已离线或不可达'
  return '状态未知，建议检查采集器进程'
}

const agentStatusBadge = (row) => {
  const meta = agentStatusMeta(row)
  return {
    text: meta.text,
    type: meta.type,
    reason: statusReason(row),
    updatedAt: row?.last_seen || row?.last_heartbeat || row?.updated_at
  }
}
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 16px; }
.page-desc { color: #606266; margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; }
</style>
