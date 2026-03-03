<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>告警事件</h2>
        <p class="page-desc">查看告警列表并进行确认/恢复处理。</p>
      </div>
      <div class="page-actions">
        <el-button icon="Refresh" @click="fetchAlerts">刷新</el-button>
      </div>
    </div>

    <div class="filter-bar">
      <el-select v-model="status" placeholder="状态" class="w-40" clearable @change="fetchAlerts">
        <el-option label="未处理" :value="0" />
        <el-option label="已确认" :value="1" />
        <el-option label="已恢复" :value="2" />
        <el-option label="已抑制" :value="3" />
      </el-select>
      <el-select v-model="severity" placeholder="级别" class="w-40" clearable @change="fetchAlerts">
        <el-option label="critical" value="critical" />
        <el-option label="warning" value="warning" />
        <el-option label="info" value="info" />
      </el-select>
      <el-input v-model="target" placeholder="目标包含" class="w-52" clearable @change="fetchAlerts" />
      <el-button type="primary" @click="fetchAlerts">查询</el-button>
    </div>

    <el-table :fit="false" :data="alerts" stripe style="width: 100%">
      <el-table-column prop="rule_name" label="规则" min-width="160" />
      <el-table-column prop="target" label="目标" min-width="200" />
      <el-table-column prop="metric" label="指标" min-width="140" />
      <el-table-column prop="value" label="值" width="120" />
      <el-table-column prop="severity" label="级别" width="120">
        <template #default="scope">
          <el-tag :type="severityTag(scope.row.severity)">{{ scope.row.severity }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="status" label="状态" width="120">
        <template #default="scope">
          <el-tag :type="statusTag(scope.row.status)">{{ statusText(scope.row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="fired_at" label="触发时间" min-width="180" />
      <el-table-column label="操作" width="220">
        <template #default="scope">
          <el-button size="small" @click="openDetail(scope.row)">详情</el-button>
          <el-button size="small" type="primary" @click="ack(scope.row)">确认</el-button>
          <el-button size="small" type="success" @click="resolve(scope.row)">恢复</el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-card>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const alerts = ref([])
const status = ref('')
const severity = ref('')
const target = ref('')
const router = useRouter()

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const fetchAlerts = async () => {
  const res = await axios.get('/api/v1/alert/alerts', {
    headers: authHeaders(),
    params: {
      status: status.value === '' ? undefined : status.value,
      severity: severity.value || undefined,
      target: target.value || undefined
    }
  })
  alerts.value = res.data.data || []
}

const ack = async (row) => {
  await ElMessageBox.confirm('确认该告警？', '提示', { type: 'warning' })
  await axios.post(`/api/v1/alert/alerts/${row.id}/ack`, {}, { headers: authHeaders() })
  ElMessage.success('已确认')
  fetchAlerts()
}

const resolve = async (row) => {
  await ElMessageBox.confirm('标记为已恢复？', '提示', { type: 'warning' })
  await axios.post(`/api/v1/alert/alerts/${row.id}/resolve`, {}, { headers: authHeaders() })
  ElMessage.success('已恢复')
  fetchAlerts()
}

const openDetail = (row) => {
  router.push({ path: '/alert/events/detail', query: { id: row.id } })
}

const statusText = (s) => {
  if (s === 0) return '未处理'
  if (s === 1) return '已确认'
  if (s === 2) return '已恢复'
  return '已抑制'
}

const statusTag = (s) => {
  if (s === 0) return 'danger'
  if (s === 1) return 'warning'
  if (s === 2) return 'success'
  return 'info'
}

const severityTag = (s) => {
  if (s === 'critical') return 'danger'
  if (s === 'warning') return 'warning'
  return 'info'
}

onMounted(fetchAlerts)
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 16px; }
.page-desc { color: #606266; margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; }
.filter-bar { display: flex; gap: 8px; flex-wrap: wrap; margin-bottom: 12px; }
.w-40 { width: 160px; }
.w-52 { width: 220px; }
</style>
