<template>
  <el-card class="page-card">
    <template #header>
      <div class="header">
        <div>
          <div class="title">操作日志</div>
          <div class="desc">记录平台内关键操作与权限审计</div>
        </div>
        <div class="actions">
          <el-input v-model="filters.username" placeholder="用户名" class="w-160" clearable />
          <el-input v-model="filters.module" placeholder="模块" class="w-160" clearable />
          <el-button type="primary" icon="Search" @click="fetchLogs">查询</el-button>
          <el-button icon="Refresh" @click="fetchLogs">刷新</el-button>
        </div>
      </div>
    </template>

    <el-table :fit="true" :data="logs" v-loading="loading" stripe>
      <el-table-column prop="username" label="用户" width="140" />
      <el-table-column prop="module" label="模块" width="140" />
      <el-table-column prop="action" label="动作" width="120" />
      <el-table-column prop="target" label="目标" min-width="220" show-overflow-tooltip />
      <el-table-column prop="detail" label="详情" min-width="220" show-overflow-tooltip />
      <el-table-column prop="ip" label="IP" width="140" />
      <el-table-column label="状态" width="90">
        <template #default="{ row }">
          <el-tag :type="row.status === 1 ? 'success' : 'danger'">
            {{ row.status === 1 ? '成功' : '失败' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="时间" width="180">
        <template #default="{ row }">
          {{ formatTime(row.created_at) }}
        </template>
      </el-table-column>
    </el-table>
  </el-card>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'

const logs = ref([])
const loading = ref(false)
const filters = reactive({
  username: '',
  module: ''
})

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })
const getErrorMessage = (err, fallback = '操作失败') => err?.response?.data?.message || err?.message || fallback

const fetchLogs = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/v1/rbac/logs', {
      headers: authHeaders(),
      params: { username: filters.username, module: filters.module }
    })
    if (res.data.code === 0) logs.value = res.data.data || []
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '获取日志失败'))
  } finally {
    loading.value = false
  }
}

const formatTime = (val) => {
  if (!val) return '-'
  return new Date(val).toLocaleString()
}

onMounted(fetchLogs)
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.header { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.title { font-size: 18px; font-weight: 600; }
.desc { color: #909399; font-size: 12px; margin-top: 4px; }
.actions { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.w-160 { width: 160px; }
</style>
