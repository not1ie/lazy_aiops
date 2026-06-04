<template>
  <div class="hub-container" v-loading="loading">
    <header class="hub-header">
      <div class="hub-title">
        <h2>身份授权与治理中心</h2>
        <p>用户 → 角色 → 权限 → 部门 → 岗位，统一身份授权工作流。所有操作有审计。</p>
      </div>
      <div class="hub-actions">
        <el-button round icon="Refresh" @click="refreshAll">刷新</el-button>
        <el-button round icon="Coin" @click="go('/cost/overview')">成本分析</el-button>
        <el-button type="primary" round icon="Plus" @click="handleAddUser">新增用户</el-button>
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
      <div class="summary-item" @click="activePanel = 'org'">
        <div class="summary-label">部门/岗位</div>
        <div class="summary-value">{{ stats.deptTotal }}/{{ stats.postTotal }}</div>
      </div>
      <div class="summary-item" @click="activePanel = 'audit'">
        <div class="summary-label">今日登录</div>
        <div class="summary-value">{{ stats.loginToday }}</div>
      </div>
    </div>

    <div class="hub-workspace full-width">
      <div class="workspace-content">
        <el-tabs v-model="activePanel" class="hub-tabs">
          <!-- 1. Users -->
          <el-tab-pane label="用户管理" name="users">
            <div class="pane-header">
              <el-input v-model="userKeyword" placeholder="搜索用户名..." size="small" style="width:200px" clearable />
              <el-button size="small" type="primary" @click="handleAddUser">新增用户</el-button>
            </div>
            <el-table :data="filteredUsers" class="hub-table" size="small">
              <el-table-column prop="username" label="用户名" width="120" />
              <el-table-column prop="nickname" label="昵称" width="120" />
              <el-table-column label="角色" width="120">
                <template #default="{ row }">
                  <el-tag size="small" effect="plain">{{ row.role?.name || row.role_name || '-' }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="部门" width="120">
                <template #default="{ row }">{{ row.dept?.name || '-' }}</template>
              </el-table-column>
              <el-table-column label="状态" width="80">
                <template #default="{ row }">
                  <el-switch v-model="row.status" :active-value="1" :inactive-value="0" size="small" @change="toggleUserStatus(row)" />
                </template>
              </el-table-column>
              <el-table-column label="最后登录" width="140">
                <template #default="{ row }">{{ fmtTimeAgo(row.last_login_at) }}</template>
              </el-table-column>
              <el-table-column label="操作" width="140" fixed="right">
                <template #default="{ row }">
                  <el-button link type="primary" size="small" @click="handleEditUser(row)">编辑</el-button>
                  <el-button link type="danger" size="small" @click="handleDeleteUser(row)">删除</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>

          <!-- 2. Roles & Permissions -->
          <el-tab-pane label="角色与权限" name="roles">
            <div class="pane-header">
              <span class="text-muted">共 {{ roles.length }} 个角色</span>
              <el-button size="small" type="primary" @click="go('/system/roles')">管理角色 →</el-button>
            </div>
            <el-table :data="roles" class="hub-table" size="small">
              <el-table-column prop="name" label="角色名称" width="140" />
              <el-table-column prop="code" label="编码" width="120" />
              <el-table-column label="权限数" width="80">
                <template #default="{ row }">{{ row.permissions?.length || 0 }}</template>
              </el-table-column>
              <el-table-column label="权限概览" min-width="300">
                <template #default="{ row }">
                  <el-tag v-for="perm in (row.permissions || []).slice(0, 8)" :key="perm.id || perm.code" size="small" effect="plain" style="margin: 2px;">{{ perm.name || perm.code }}</el-tag>
                  <el-tag v-if="(row.permissions || []).length > 8" size="small" type="info" effect="plain">+{{ row.permissions.length - 8 }} more</el-tag>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>

          <!-- 3. Organization -->
          <el-tab-pane label="组织架构" name="org">
            <el-row :gutter="16">
              <el-col :span="12">
                <h4 class="sub-title">部门</h4>
                <el-table :data="depts" size="small">
                  <el-table-column prop="name" label="名称" />
                  <el-table-column label="状态" width="80">
                    <template #default="{ row }"><el-tag :type="row.status === 1 ? 'success' : 'info'" size="small">{{ row.status === 1 ? '正常' : '停用' }}</el-tag></template>
                  </el-table-column>
                </el-table>
                <div class="tab-footer-link">
                  <el-button link type="primary" size="small" @click="go('/system/dept')">管理部门 →</el-button>
                </div>
              </el-col>
              <el-col :span="12">
                <h4 class="sub-title">岗位</h4>
                <el-table :data="posts" size="small">
                  <el-table-column prop="name" label="名称" />
                  <el-table-column label="状态" width="80">
                    <template #default="{ row }"><el-tag :type="row.status === 1 ? 'success' : 'info'" size="small">{{ row.status === 1 ? '正常' : '停用' }}</el-tag></template>
                  </el-table-column>
                </el-table>
                <div class="tab-footer-link">
                  <el-button link type="primary" size="small" @click="go('/system/posts')">管理岗位 →</el-button>
                </div>
              </el-col>
            </el-row>
          </el-tab-pane>

          <!-- 4. Audit -->
          <el-tab-pane label="安全审计" name="audit">
            <el-row :gutter="16">
              <el-col :span="12">
                <h4 class="sub-title">操作日志</h4>
                <el-table :data="auditLogs" size="small">
                  <el-table-column prop="username" label="用户" width="100" />
                  <el-table-column prop="action" label="操作" width="80" />
                  <el-table-column prop="target" label="目标" min-width="150" show-overflow-tooltip />
                  <el-table-column label="时间" width="130">
                    <template #default="{ row }">{{ fmtTimeAgo(row.created_at) }}</template>
                  </el-table-column>
                </el-table>
                <div class="tab-footer-link">
                  <el-button link type="primary" size="small" @click="go('/system/audit-logs')">全部操作日志 →</el-button>
                </div>
              </el-col>
              <el-col :span="12">
                <h4 class="sub-title">登录记录</h4>
                <el-table :data="loginLogs" size="small">
                  <el-table-column prop="username" label="用户" width="100" />
                  <el-table-column prop="ip" label="IP" width="120" />
                  <el-table-column label="时间" width="130">
                    <template #default="{ row }">{{ fmtTimeAgo(row.login_at || row.created_at) }}</template>
                  </el-table-column>
                </el-table>
                <div class="tab-footer-link">
                  <el-button link type="primary" size="small" @click="go('/system/login-logs')">全部登录日志 →</el-button>
                </div>
              </el-col>
            </el-row>
          </el-tab-pane>
        </el-tabs>
      </div>
    </div>

    <!-- Quick User Dialog -->
    <el-dialog title="新增用户" v-model="userDialogVisible" width="480px" append-to-body>
      <el-form :model="userForm" label-width="80px">
        <el-form-item label="用户名" required><el-input v-model="userForm.username" /></el-form-item>
        <el-form-item label="密码" required><el-input v-model="userForm.password" type="password" show-password /></el-form-item>
        <el-form-item label="昵称"><el-input v-model="userForm.nickname" /></el-form-item>
        <el-form-item label="角色">
          <el-select v-model="userForm.role_id" style="width:100%" clearable>
            <el-option v-for="r in roles" :key="r.id" :label="r.name" :value="r.id" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="userDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="userSaving" @click="submitUser">创建</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import './hub-common.css'

const router = useRouter()
const loading = ref(false)
const users = ref([])
const roles = ref([])
const depts = ref([])
const posts = ref([])
const auditLogs = ref([])
const loginLogs = ref([])
const activePanel = ref('users')
const userKeyword = ref('')

const userDialogVisible = ref(false)
const userSaving = ref(false)
const userForm = ref({ username: '', password: '', nickname: '', role_id: '' })

const stats = reactive({ userTotal: 0, roleTotal: 0, deptTotal: 0, postTotal: 0, loginToday: 0 })

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })
const go = (path) => router.push(path)
const fmtTimeAgo = (val) => {
  if (!val) return '-'
  const diff = Math.floor((Date.now() - new Date(val).getTime()) / 1000)
  if (diff < 60) return '刚刚'
  if (diff < 3600) return `${Math.floor(diff / 60)}m 前`
  if (diff < 86400) return `${Math.floor(diff / 3600)}h 前`
  return `${Math.floor(diff / 86400)}d 前`
}

const filteredUsers = computed(() => {
  if (!userKeyword.value) return users.value
  const kw = userKeyword.value.toLowerCase()
  return users.value.filter(u => (u.username || '').toLowerCase().includes(kw) || (u.nickname || '').toLowerCase().includes(kw))
})

const refreshAll = async () => {
  loading.value = true
  try {
    const headers = authHeaders()
    const [usrRes, rolRes, deptRes, postRes, logRes, loginRes] = await Promise.all([
      axios.get('/api/v1/rbac/users', { headers }),
      axios.get('/api/v1/rbac/roles', { headers }),
      axios.get('/api/v1/system/depts', { headers }).catch(() => ({ data: {} })),
      axios.get('/api/v1/system/posts', { headers }).catch(() => ({ data: {} })),
      axios.get('/api/v1/rbac/logs', { headers, params: { page: 1, size: 10 } }).catch(() => ({ data: {} })),
      axios.get('/api/v1/system/login-logs', { headers, params: { size: 10 } }).catch(() => ({ data: {} }))
    ])
    users.value = usrRes.data?.data || []
    roles.value = rolRes.data?.data || []
    depts.value = deptRes.data?.data || []
    posts.value = postRes.data?.data || []
    auditLogs.value = logRes.data?.data || []
    loginLogs.value = loginRes.data?.data || []

    stats.userTotal = users.value.length
    stats.roleTotal = roles.value.length
    stats.deptTotal = depts.value.length
    stats.postTotal = posts.value.length
    stats.loginToday = loginLogs.value.length
  } catch (err) {
    ElMessage.error('加载系统中心失败')
  } finally {
    loading.value = false
  }
}

const handleAddUser = () => {
  userForm.value = { username: '', password: '', nickname: '', role_id: '' }
  userDialogVisible.value = true
}

const handleEditUser = (row) => router.push(`/system/users?edit=${row.id}`)
const handleDeleteUser = async (row) => {
  try {
    await ElMessageBox.confirm(`确认删除用户「${row.username}」？`, '提示', { type: 'warning' })
    await axios.delete(`/api/v1/rbac/users/${row.id}`, { headers: authHeaders() })
    ElMessage.success('已删除')
    refreshAll()
  } catch (e) { /* cancel */ }
}

const toggleUserStatus = async (row) => {
  try {
    await axios.put(`/api/v1/rbac/users/${row.id}/status`, { status: row.status }, { headers: authHeaders() })
  } catch (err) {
    row.status = row.status === 1 ? 0 : 1
    ElMessage.error('切换失败')
  }
}

const submitUser = async () => {
  if (!userForm.value.username || !userForm.value.password) return ElMessage.warning('用户名和密码为必填')
  userSaving.value = true
  try {
    await axios.post('/api/v1/rbac/users', userForm.value, { headers: authHeaders() })
    ElMessage.success('用户创建成功')
    userDialogVisible.value = false
    refreshAll()
  } catch (err) {
    ElMessage.error('创建失败: ' + (err.response?.data?.message || err.message))
  } finally {
    userSaving.value = false
  }
}

onMounted(refreshAll)
</script>

<style scoped>
.full-width { grid-template-columns: 1fr; }
.pane-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.sub-title { font-size: 14px; font-weight: 700; margin: 0 0 12px 0; color: var(--el-text-color-primary); }
.text-muted { color: var(--el-text-color-secondary); font-size: 13px; }
.tab-footer-link { margin-top: 12px; text-align: right; }
</style>
