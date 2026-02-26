<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>登录日志</h2>
        <p class="page-desc">审计登录行为，定位异常来源与失败原因。</p>
      </div>
      <div class="page-actions">
        <el-input v-model="filters.username" placeholder="用户名" clearable style="width: 160px" @keyup.enter="fetchLogs" />
        <el-input v-model="filters.ip" placeholder="IP" clearable style="width: 160px" @keyup.enter="fetchLogs" />
        <el-select v-model="filters.status" placeholder="状态" clearable style="width: 120px" @change="fetchLogs">
          <el-option label="成功" :value="1" />
          <el-option label="失败" :value="0" />
        </el-select>
        <el-date-picker
          v-model="timeRange"
          type="datetimerange"
          range-separator="至"
          start-placeholder="开始时间"
          end-placeholder="结束时间"
          style="width: 340px"
          value-format="YYYY-MM-DDTHH:mm:ssZ"
        />
        <el-button type="primary" icon="Search" @click="handleQuery">查询</el-button>
        <el-button icon="Refresh" @click="resetQuery">重置</el-button>
      </div>
    </div>

    <el-table :data="logs" v-loading="loading" stripe>
      <el-table-column prop="username" label="用户名" width="140" />
      <el-table-column prop="ip" label="IP" width="150" />
      <el-table-column label="状态" width="90">
        <template #default="{ row }">
          <el-tag :type="row.status === 1 ? 'success' : 'danger'">{{ row.status === 1 ? '成功' : '失败' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="message" label="结果" min-width="160" show-overflow-tooltip />
      <el-table-column prop="user_agent" label="User Agent" min-width="320" show-overflow-tooltip />
      <el-table-column label="登录时间" width="180">
        <template #default="{ row }">{{ formatTime(row.login_at) }}</template>
      </el-table-column>
    </el-table>

    <div class="pager">
      <el-pagination
        background
        layout="total, prev, pager, next, sizes"
        :total="total"
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :page-sizes="[20, 50, 100]"
        @current-change="fetchLogs"
        @size-change="fetchLogs"
      />
    </div>
  </el-card>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'

const loading = ref(false)
const logs = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const timeRange = ref([])

const filters = reactive({
  username: '',
  ip: '',
  status: ''
})

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const formatTime = (value) => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString()
}

const fetchLogs = async () => {
  loading.value = true
  try {
    const params = {
      page: page.value,
      page_size: pageSize.value,
      username: filters.username || undefined,
      ip: filters.ip || undefined,
      status: filters.status === '' ? undefined : filters.status,
      start_at: timeRange.value?.[0] || undefined,
      end_at: timeRange.value?.[1] || undefined
    }

    const res = await axios.get('/api/v1/system/login-logs', {
      headers: authHeaders(),
      params
    })

    if (res.data?.code === 0) {
      logs.value = res.data.data?.items || []
      total.value = Number(res.data.data?.total || 0)
    }
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '获取登录日志失败')
  } finally {
    loading.value = false
  }
}

const handleQuery = () => {
  page.value = 1
  fetchLogs()
}

const resetQuery = () => {
  filters.username = ''
  filters.ip = ''
  filters.status = ''
  timeRange.value = []
  page.value = 1
  fetchLogs()
}

onMounted(fetchLogs)
</script>

<style scoped>
.page-card { max-width: 1400px; margin: 0 auto; }
.page-header { display: flex; justify-content: space-between; align-items: center; gap: 12px; margin-bottom: 12px; }
.page-desc { color: #909399; margin: 4px 0 0; }
.page-actions { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.pager { display: flex; justify-content: flex-end; margin-top: 12px; }
</style>
