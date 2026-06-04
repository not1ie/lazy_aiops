<template>
  <div class="hub-container" v-loading="loading">
    <header class="hub-header">
      <div class="hub-title">
        <h2>系统治理中心</h2>
        <p>统一管理用户权限、组织架构及安全审计日志，确保平台运行合规可控。</p>
      </div>
      <div class="hub-actions">
        <el-button round icon="Refresh" @click="refreshAll">刷新系统状态</el-button>
        <el-button round icon="Coin" @click="go('/cost/overview')">成本分析</el-button>
        <el-button type="primary" round icon="User" @click="go('/system/users')">新增用户</el-button>
      </div>
    </header>

    <div class="hub-summary">
      <div class="summary-item" @click="activePanel = 'users'">
        <div class="summary-label">注册用户</div>
        <div class="summary-value">{{ stats.userTotal }}</div>
      </div>
      <div class="summary-item" @click="activePanel = 'roles'">
        <div class="summary-label">系统角色</div>
        <div class="summary-value">{{ stats.roleTotal }}</div>
      </div>
      <div class="summary-item" @click="activePanel = 'logs'">
        <div class="summary-label">今日审计日志</div>
        <div class="summary-value success">{{ stats.auditToday }}</div>
      </div>
      <div class="summary-item">
        <div class="summary-label">安全状态</div>
        <div class="summary-value success">合规</div>
      </div>
    </div>

    <div class="hub-workspace full-width">
      <div class="workspace-content">
        <el-tabs v-model="activePanel" class="hub-tabs">
          <el-tab-pane label="用户与权限" name="users">
            <el-table :data="users" class="hub-table" size="small">
              <el-table-column prop="username" label="用户名" />
              <el-table-column prop="nickname" label="昵称" />
              <el-table-column prop="role_name" label="所属角色" />
              <el-table-column label="状态">
                <template #default="{ row }"><el-tag :type="row.status === 1 ? 'success' : 'info'" size="small">{{ row.status === 1 ? '正常' : '禁用' }}</el-tag></template>
              </el-table-column>
              <el-table-column label="操作" width="100">
                <template #default="{ row }">
                   <el-button link type="primary" @click="go('/system/users')">编辑</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>

          <el-tab-pane label="审计日志" name="logs">
            <el-table :data="auditLogs" class="hub-table" size="small">
              <el-table-column prop="username" label="操作人" width="120" />
              <el-table-column prop="operation" label="动作" width="150" />
              <el-table-column prop="path" label="路径" min-width="200" />
              <el-table-column prop="ip" label="IP" width="130" />
              <el-table-column label="时间" prop="created_at" :formatter="formatTime" width="160" />
            </el-table>
          </el-tab-pane>
        </el-tabs>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import './hub-common.css'

const router = useRouter()
const loading = ref(false)
const users = ref([])
const auditLogs = ref([])
const activePanel = ref('users')

const stats = reactive({
  userTotal: 0,
  roleTotal: 0,
  auditToday: 0
})

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })
const go = (path) => router.push(path)
const formatTime = (row, col, val) => val ? new Date(val).toLocaleString() : '-'

const refreshAll = async () => {
  loading.value = true
  try {
    const [usr, rol, log] = await Promise.all([
      axios.get('/api/v1/rbac/users', { headers: authHeaders() }),
      axios.get('/api/v1/rbac/roles', { headers: authHeaders() }),
      axios.get('/api/v1/rbac/logs', { headers: authHeaders(), params: { page: 1, size: 20 } }).catch(() => ({ data: {} }))
    ])
    users.value = usr.data?.data || []
    auditLogs.value = log.data?.data || []
    stats.userTotal = users.value.length
    stats.roleTotal = rol.data?.data?.length || 0
    stats.auditToday = log.data?.total || auditLogs.value.length
  } catch (err) {
    ElMessage.error('加载系统中心失败')
  } finally {
    loading.value = false
  }
}

onMounted(refreshAll)
</script>

<style scoped>
.full-width { grid-template-columns: 1fr; }
</style>
