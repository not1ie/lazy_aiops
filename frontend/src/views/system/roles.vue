<template>
  <el-card class="page-card">
    <template #header>
      <div class="header">
        <div>
          <div class="title">角色管理</div>
          <div class="desc">创建角色并分配权限</div>
        </div>
        <div class="actions">
          <el-button type="primary" icon="Plus" @click="openCreate">新增角色</el-button>
          <el-button icon="Refresh" @click="fetchRoles">刷新</el-button>
        </div>
      </div>
    </template>

    <el-table :fit="true" :data="roles" v-loading="loading" stripe>
      <el-table-column prop="name" label="角色名称" min-width="160" />
      <el-table-column prop="code" label="角色编码" min-width="160" />
      <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
      <el-table-column label="权限数" width="100">
        <template #default="{ row }">
          {{ row.permissions?.length || 0 }}
        </template>
      </el-table-column>
      <el-table-column label="创建时间" min-width="180">
        <template #default="{ row }">
          {{ formatTime(row.created_at) }}
        </template>
      </el-table-column>
      <el-table-column label="操作" width="240" fixed="right">
        <template #default="{ row }">
          <el-space size="8">
            <el-button size="small" type="primary" plain @click="openEdit(row)">编辑</el-button>
            <el-button size="small" type="success" plain @click="openPermissions(row)">分配权限</el-button>
            <el-button size="small" type="danger" plain @click="removeRole(row)">删除</el-button>
          </el-space>
        </template>
      </el-table-column>
    </el-table>
  </el-card>

  <el-dialog append-to-body v-model="dialogVisible" :title="isEdit ? '编辑角色' : '新增角色'" width="520px">
    <el-form :model="form" label-width="90px">
      <el-form-item label="名称" required>
        <el-input v-model="form.name" />
      </el-form-item>
      <el-form-item label="编码" required>
        <el-input v-model="form.code" :disabled="isEdit" />
      </el-form-item>
      <el-form-item label="描述">
        <el-input v-model="form.description" type="textarea" :rows="3" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="dialogVisible = false">取消</el-button>
      <el-button type="primary" :loading="saving" @click="saveRole">保存</el-button>
    </template>
  </el-dialog>

  <el-dialog append-to-body v-model="permVisible" title="分配权限" width="640px">
    <el-tree
      ref="permTreeRef"
      :data="permissionTree"
      node-key="id"
      show-checkbox
      default-expand-all
      :props="{ label: 'name', children: 'children' }"
    />
    <template #footer>
      <el-button @click="permVisible = false">取消</el-button>
      <el-button type="primary" :loading="permSaving" @click="submitPermissions">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const roles = ref([])
const loading = ref(false)
const saving = ref(false)
const dialogVisible = ref(false)
const isEdit = ref(false)
const currentId = ref('')

const permVisible = ref(false)
const permSaving = ref(false)
const permTreeRef = ref(null)
const permissionTree = ref([])
const currentRole = ref(null)

const form = reactive({
  name: '',
  code: '',
  description: ''
})

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })
const getErrorMessage = (err, fallback = '操作失败') => err?.response?.data?.message || err?.message || fallback

const fetchRoles = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/v1/rbac/roles', { headers: authHeaders() })
    if (res.data.code === 0) roles.value = res.data.data || []
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '获取角色失败'))
  } finally {
    loading.value = false
  }
}

const fetchPermissionTree = async () => {
  try {
    const res = await axios.get('/api/v1/rbac/permissions/tree', { headers: authHeaders() })
    if (res.data.code === 0) permissionTree.value = res.data.data || []
  } catch {
    permissionTree.value = []
  }
}

const openCreate = () => {
  isEdit.value = false
  currentId.value = ''
  form.name = ''
  form.code = ''
  form.description = ''
  dialogVisible.value = true
}

const openEdit = (row) => {
  isEdit.value = true
  currentId.value = row.id
  form.name = row.name
  form.code = row.code
  form.description = row.description || ''
  dialogVisible.value = true
}

const saveRole = async () => {
  saving.value = true
  try {
    if (isEdit.value) {
      await axios.put(`/api/v1/rbac/roles/${currentId.value}`, {
        name: form.name,
        code: form.code,
        description: form.description
      }, { headers: authHeaders() })
    } else {
      await axios.post('/api/v1/rbac/roles', {
        name: form.name,
        code: form.code,
        description: form.description
      }, { headers: authHeaders() })
    }
    ElMessage.success('保存成功')
    dialogVisible.value = false
    await fetchRoles()
  } catch (e) {
    ElMessage.error(getErrorMessage(e, '保存失败'))
  } finally {
    saving.value = false
  }
}

const openPermissions = async (row) => {
  currentRole.value = row
  await fetchPermissionTree()
  permVisible.value = true
  const checked = (row.permissions || []).map((p) => p.id)
  setTimeout(() => {
    permTreeRef.value?.setCheckedKeys(checked, false)
  }, 50)
}

const submitPermissions = async () => {
  if (!currentRole.value) return
  permSaving.value = true
  try {
    const keys = permTreeRef.value?.getCheckedKeys() || []
    await axios.put(`/api/v1/rbac/roles/${currentRole.value.id}/permissions`, {
      permission_ids: keys
    }, { headers: authHeaders() })
    ElMessage.success('权限已更新')
    permVisible.value = false
    await fetchRoles()
  } catch (e) {
    ElMessage.error(getErrorMessage(e, '保存失败'))
  } finally {
    permSaving.value = false
  }
}

const removeRole = async (row) => {
  try {
    await ElMessageBox.confirm(`确认删除角色 ${row.name}？`, '提示', { type: 'warning' })
    await axios.delete(`/api/v1/rbac/roles/${row.id}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    await fetchRoles()
  } catch (e) {
    if (e !== 'cancel') {
      ElMessage.error(getErrorMessage(e, '删除失败'))
    }
  }
}

const formatTime = (val) => {
  if (!val) return '-'
  return new Date(val).toLocaleString()
}

onMounted(fetchRoles)
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.header { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.title { font-size: 18px; font-weight: 600; }
.desc { color: #909399; font-size: 12px; margin-top: 4px; }
.actions { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
</style>
