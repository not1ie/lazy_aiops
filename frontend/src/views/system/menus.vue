<template>
  <el-card class="page-card">
    <template #header>
      <div class="header">
        <div>
          <div class="title">权限管理</div>
          <div class="desc">维护菜单/接口权限，用于角色授权</div>
        </div>
        <div class="actions">
          <el-button type="primary" icon="Plus" @click="openCreate">新增权限</el-button>
          <el-button icon="Refresh" @click="fetchPermissions">刷新</el-button>
        </div>
      </div>
    </template>

    <el-table :fit="true" :data="permissions" v-loading="loading" stripe>
      <el-table-column prop="name" label="名称" min-width="160" />
      <el-table-column prop="code" label="编码" min-width="200" />
      <el-table-column prop="type" label="类型" width="120">
        <template #default="{ row }">
          <el-tag>{{ typeLabel(row.type) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="父级" min-width="160">
        <template #default="{ row }">
          {{ parentName(row.parent_id) || '-' }}
        </template>
      </el-table-column>
      <el-table-column label="创建时间" min-width="180">
        <template #default="{ row }">
          {{ formatTime(row.created_at) }}
        </template>
      </el-table-column>
      <el-table-column label="操作" width="200" fixed="right">
        <template #default="{ row }">
          <el-space size="8">
            <el-button size="small" type="primary" plain @click="openEdit(row)">编辑</el-button>
            <el-button size="small" type="danger" plain @click="removePermission(row)">删除</el-button>
          </el-space>
        </template>
      </el-table-column>
    </el-table>
  </el-card>

  <el-dialog append-to-body v-model="dialogVisible" :title="isEdit ? '编辑权限' : '新增权限'" width="520px">
    <el-form :model="form" label-width="90px">
      <el-form-item label="名称" required>
        <el-input v-model="form.name" />
      </el-form-item>
      <el-form-item label="编码" required>
        <el-input v-model="form.code" :disabled="isEdit" />
      </el-form-item>
      <el-form-item label="类型">
        <el-select v-model="form.type" class="w-100">
          <el-option label="菜单" value="menu" />
          <el-option label="按钮" value="button" />
          <el-option label="接口" value="api" />
        </el-select>
      </el-form-item>
      <el-form-item label="父级">
        <el-select v-model="form.parent_id" class="w-100" clearable>
          <el-option label="无" value="" />
          <el-option v-for="perm in permissions" :key="perm.id" :label="perm.name" :value="perm.id" />
        </el-select>
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="dialogVisible = false">取消</el-button>
      <el-button type="primary" :loading="saving" @click="savePermission">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const permissions = ref([])
const loading = ref(false)
const saving = ref(false)
const dialogVisible = ref(false)
const isEdit = ref(false)
const currentId = ref('')

const form = reactive({
  name: '',
  code: '',
  type: 'menu',
  parent_id: ''
})

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const fetchPermissions = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/v1/rbac/permissions', { headers: authHeaders() })
    if (res.data.code === 0) permissions.value = res.data.data || []
  } catch {
    ElMessage.error('获取权限失败')
  } finally {
    loading.value = false
  }
}

const openCreate = () => {
  isEdit.value = false
  currentId.value = ''
  form.name = ''
  form.code = ''
  form.type = 'menu'
  form.parent_id = ''
  dialogVisible.value = true
}

const openEdit = (row) => {
  isEdit.value = true
  currentId.value = row.id
  form.name = row.name
  form.code = row.code
  form.type = row.type || 'menu'
  form.parent_id = row.parent_id || ''
  dialogVisible.value = true
}

const savePermission = async () => {
  saving.value = true
  try {
    if (isEdit.value) {
      await axios.put(`/api/v1/rbac/permissions/${currentId.value}`, {
        name: form.name,
        code: form.code,
        type: form.type,
        parent_id: form.parent_id || ''
      }, { headers: authHeaders() })
    } else {
      await axios.post('/api/v1/rbac/permissions', {
        name: form.name,
        code: form.code,
        type: form.type,
        parent_id: form.parent_id || ''
      }, { headers: authHeaders() })
    }
    ElMessage.success('保存成功')
    dialogVisible.value = false
    fetchPermissions()
  } catch (e) {
    ElMessage.error(e?.response?.data?.message || '保存失败')
  } finally {
    saving.value = false
  }
}

const removePermission = async (row) => {
  try {
    await ElMessageBox.confirm(`确认删除权限 ${row.name}？`, '提示', { type: 'warning' })
    await axios.delete(`/api/v1/rbac/permissions/${row.id}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    fetchPermissions()
  } catch (e) {
    if (e !== 'cancel') {
      ElMessage.error(e?.response?.data?.message || '删除失败')
    }
  }
}

const parentName = (id) => permissions.value.find((p) => p.id === id)?.name

const typeLabel = (type) => {
  if (type === 'api') return '接口'
  if (type === 'button') return '按钮'
  return '菜单'
}

const formatTime = (val) => {
  if (!val) return '-'
  return new Date(val).toLocaleString()
}

onMounted(fetchPermissions)
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.header { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.title { font-size: 18px; font-weight: 600; }
.desc { color: #909399; font-size: 12px; margin-top: 4px; }
.actions { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.w-100 { width: 100%; }
</style>
