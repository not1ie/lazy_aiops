<template>
  <el-card class="page-card">
    <template #header>
      <div class="header">
        <div>
          <div class="title">用户管理</div>
          <div class="desc">管理系统账号、角色与状态</div>
        </div>
        <div class="actions">
          <el-input v-model="keyword" placeholder="搜索用户名" class="w-220" clearable @keyup.enter="fetchUsers" />
          <el-button type="primary" icon="Plus" @click="openCreate">新增用户</el-button>
          <el-button icon="Refresh" @click="fetchUsers">刷新</el-button>
        </div>
      </div>
    </template>

    <el-table :fit="true" :data="users" v-loading="loading" stripe>
      <el-table-column prop="username" label="用户名" min-width="140" />
      <el-table-column prop="nickname" label="昵称" min-width="120" />
      <el-table-column prop="email" label="邮箱" min-width="180" />
      <el-table-column prop="phone" label="电话" min-width="140" />
      <el-table-column label="角色" min-width="120">
        <template #default="{ row }">
          <el-tag v-if="row.role">{{ row.role.name }}</el-tag>
          <span v-else>-</span>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="row.status === 1 ? 'success' : 'info'">
            {{ row.status === 1 ? '启用' : '禁用' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="创建时间" min-width="180">
        <template #default="{ row }">
          {{ formatTime(row.created_at) }}
        </template>
      </el-table-column>
      <el-table-column label="操作" width="280" fixed="right">
        <template #default="{ row }">
          <el-space size="8">
            <el-button size="small" type="primary" plain @click="openEdit(row)">编辑</el-button>
            <el-button size="small" type="warning" plain @click="openPassword(row)">重置密码</el-button>
            <el-button
              size="small"
              :type="row.status === 1 ? 'info' : 'success'"
              plain
              @click="toggleStatus(row)"
            >
              {{ row.status === 1 ? '禁用' : '启用' }}
            </el-button>
            <el-button size="small" type="danger" plain @click="removeUser(row)">删除</el-button>
          </el-space>
        </template>
      </el-table-column>
    </el-table>
  </el-card>

  <el-dialog append-to-body v-model="dialogVisible" :title="isEdit ? '编辑用户' : '新增用户'" width="520px">
    <el-form :model="form" label-width="90px">
      <el-form-item label="用户名" required>
        <el-input v-model="form.username" :disabled="isEdit" />
      </el-form-item>
      <el-form-item v-if="!isEdit" label="密码" required>
        <el-input v-model="form.password" type="password" show-password />
      </el-form-item>
      <el-form-item label="昵称">
        <el-input v-model="form.nickname" />
      </el-form-item>
      <el-form-item label="邮箱">
        <el-input v-model="form.email" />
      </el-form-item>
      <el-form-item label="电话">
        <el-input v-model="form.phone" />
      </el-form-item>
      <el-form-item label="角色">
        <el-select v-model="form.role_id" placeholder="选择角色" class="w-100">
          <el-option v-for="role in roles" :key="role.id" :label="role.name" :value="role.id" />
        </el-select>
      </el-form-item>
      <el-form-item label="状态">
        <el-switch v-model="form.status" :active-value="1" :inactive-value="0" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="dialogVisible = false">取消</el-button>
      <el-button type="primary" :loading="saving" @click="saveUser">保存</el-button>
    </template>
  </el-dialog>

  <el-dialog append-to-body v-model="passwordVisible" title="重置密码" width="420px">
    <el-form :model="passwordForm" label-width="90px">
      <el-form-item label="新密码" required>
        <el-input v-model="passwordForm.new_password" type="password" show-password />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="passwordVisible = false">取消</el-button>
      <el-button type="primary" :loading="passwordLoading" @click="submitPassword">确认</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const loading = ref(false)
const saving = ref(false)
const users = ref([])
const roles = ref([])
const keyword = ref('')

const dialogVisible = ref(false)
const isEdit = ref(false)
const currentId = ref('')

const passwordVisible = ref(false)
const passwordLoading = ref(false)
const passwordForm = reactive({
  id: '',
  new_password: ''
})

const form = reactive({
  username: '',
  password: '',
  nickname: '',
  email: '',
  phone: '',
  status: 1,
  role_id: ''
})

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const fetchRoles = async () => {
  try {
    const res = await axios.get('/api/v1/rbac/roles', { headers: authHeaders() })
    if (res.data.code === 0) roles.value = res.data.data || []
  } catch {
    // ignore
  }
}

const fetchUsers = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/v1/rbac/users', {
      headers: authHeaders(),
      params: { username: keyword.value }
    })
    if (res.data.code === 0) users.value = res.data.data || []
  } catch {
    ElMessage.error('获取用户失败')
  } finally {
    loading.value = false
  }
}

const resetForm = () => {
  form.username = ''
  form.password = ''
  form.nickname = ''
  form.email = ''
  form.phone = ''
  form.status = 1
  form.role_id = ''
}

const openCreate = () => {
  isEdit.value = false
  currentId.value = ''
  resetForm()
  dialogVisible.value = true
}

const openEdit = (row) => {
  isEdit.value = true
  currentId.value = row.id
  form.username = row.username
  form.password = ''
  form.nickname = row.nickname || ''
  form.email = row.email || ''
  form.phone = row.phone || ''
  form.status = row.status
  form.role_id = row.role_id || row.role?.id || ''
  dialogVisible.value = true
}

const saveUser = async () => {
  if (!isEdit.value && !form.password) {
    ElMessage.warning('请输入初始密码')
    return
  }
  saving.value = true
  try {
    if (isEdit.value) {
      await axios.put(`/api/v1/rbac/users/${currentId.value}`, {
        nickname: form.nickname,
        email: form.email,
        phone: form.phone,
        status: form.status,
        role_id: form.role_id
      }, { headers: authHeaders() })
    } else {
      await axios.post('/api/v1/rbac/users', {
        username: form.username,
        password: form.password,
        nickname: form.nickname,
        email: form.email,
        phone: form.phone,
        status: form.status,
        role_id: form.role_id
      }, { headers: authHeaders() })
    }
    ElMessage.success('保存成功')
    dialogVisible.value = false
    fetchUsers()
  } catch (e) {
    ElMessage.error(e?.response?.data?.message || '保存失败')
  } finally {
    saving.value = false
  }
}

const openPassword = (row) => {
  passwordForm.id = row.id
  passwordForm.new_password = ''
  passwordVisible.value = true
}

const submitPassword = async () => {
  if (!passwordForm.new_password) {
    ElMessage.warning('请输入新密码')
    return
  }
  passwordLoading.value = true
  try {
    await axios.put(`/api/v1/rbac/users/${passwordForm.id}/password`, {
      new_password: passwordForm.new_password
    }, { headers: authHeaders() })
    ElMessage.success('密码已更新')
    passwordVisible.value = false
  } catch (e) {
    ElMessage.error(e?.response?.data?.message || '更新失败')
  } finally {
    passwordLoading.value = false
  }
}

const toggleStatus = async (row) => {
  const nextStatus = row.status === 1 ? 0 : 1
  try {
    await axios.put(`/api/v1/rbac/users/${row.id}/status`, { status: nextStatus }, { headers: authHeaders() })
    ElMessage.success('状态已更新')
    fetchUsers()
  } catch (e) {
    ElMessage.error(e?.response?.data?.message || '更新失败')
  }
}

const removeUser = async (row) => {
  try {
    await ElMessageBox.confirm(`确认删除用户 ${row.username}？`, '提示', { type: 'warning' })
    await axios.delete(`/api/v1/rbac/users/${row.id}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    fetchUsers()
  } catch (e) {
    if (e !== 'cancel') {
      ElMessage.error(e?.response?.data?.message || '删除失败')
    }
  }
}

const formatTime = (val) => {
  if (!val) return '-'
  return new Date(val).toLocaleString()
}

onMounted(() => {
  fetchRoles()
  fetchUsers()
})
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.header { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.title { font-size: 18px; font-weight: 600; }
.desc { color: #909399; font-size: 12px; margin-top: 4px; }
.actions { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.w-220 { width: 220px; }
.w-100 { width: 100%; }
</style>
