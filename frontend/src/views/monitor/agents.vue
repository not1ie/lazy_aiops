<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>Agent 心跳</h2>
        <p class="page-desc">采集器在线状态与心跳时间。</p>
      </div>
      <div class="page-actions">
        <el-button icon="Refresh" @click="fetchAgents">刷新</el-button>
      </div>
    </div>

    <el-table :data="agents" stripe style="width: 100%">
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
          <el-tag :type="scope.row.status === 'online' ? 'success' : 'info'">
            {{ scope.row.status }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="last_seen" label="最后心跳" min-width="180" />
      <el-table-column label="操作" width="120">
        <template #default="scope">
          <el-button size="small" @click="openDetail(scope.row)">详情</el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-card>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'

const agents = ref([])
const router = useRouter()
const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const fetchAgents = async () => {
  const res = await axios.get('/api/v1/monitor/agents', { headers: authHeaders() })
  agents.value = res.data.data || []
}

onMounted(fetchAgents)

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
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 16px; }
.page-desc { color: #606266; margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; }
</style>
