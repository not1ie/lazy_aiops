<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>岗位管理</h2>
        <p class="page-desc">维护岗位编码、排序与启用状态。</p>
      </div>
      <div class="page-actions">
        <el-input
          v-model="keyword"
          placeholder="搜索岗位名称/编码"
          clearable
          style="width: 240px"
          @keyup.enter="fetchPosts"
        />
        <el-select v-model="statusFilter" placeholder="状态" clearable style="width: 120px" @change="fetchPosts">
          <el-option label="启用" :value="1" />
          <el-option label="禁用" :value="0" />
        </el-select>
        <el-button type="primary" icon="Plus" @click="openCreate">新增岗位</el-button>
        <el-button icon="Refresh" @click="fetchPosts">刷新</el-button>
      </div>
    </div>

    <div class="table-scroll">
      <el-table :fit="true" :data="posts" v-loading="loading" stripe style="width: 100%; min-width: 1020px">
      <el-table-column prop="name" label="岗位名称" min-width="180" />
      <el-table-column prop="code" label="岗位编码" min-width="180" />
      <el-table-column prop="sort" label="排序" width="100" />
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <StatusBadge v-bind="postStatusBadge(row)" />
        </template>
      </el-table-column>
      <el-table-column prop="description" label="描述" min-width="220" show-overflow-tooltip />
      <el-table-column label="更新时间" width="180">
        <template #default="{ row }">{{ formatTime(row.updated_at) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="180" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="openEdit(row)">编辑</el-button>
          <el-button size="small" type="danger" plain @click="removePost(row)">删除</el-button>
        </template>
      </el-table-column>
      </el-table>
    </div>

    <el-dialog append-to-body v-model="dialogVisible" :title="isEdit ? '编辑岗位' : '新增岗位'" width="500px">
      <el-form :model="form" label-width="88px">
        <el-form-item label="岗位名称" required>
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="岗位编码" required>
          <el-input v-model="form.code" :disabled="isEdit" />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="form.sort" :min="0" :step="1" />
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="form.enabled" inline-prompt active-text="启用" inactive-text="禁用" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="3" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="savePost">保存</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import StatusBadge from '@/components/common/StatusBadge.vue'

const loading = ref(false)
const saving = ref(false)
const posts = ref([])
const keyword = ref('')
const statusFilter = ref('')
const dialogVisible = ref(false)
const isEdit = ref(false)
const currentId = ref('')

const form = reactive({
  name: '',
  code: '',
  sort: 0,
  enabled: true,
  description: ''
})

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })
const getErrorMessage = (err, fallback = '操作失败') => err?.response?.data?.message || err?.message || fallback

const formatTime = (value) => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString()
}

const postStatusBadge = (row) => {
  const enabled = Number(row?.status) === 1
  return {
    text: enabled ? '启用' : '禁用',
    type: enabled ? 'success' : 'info',
    source: '岗位配置',
    checkAt: row?.updated_at || row?.created_at || '',
    reason: enabled ? '岗位可被账号绑定并参与权限继承' : '岗位已停用，不再分配给用户'
  }
}

const resetForm = () => {
  form.name = ''
  form.code = ''
  form.sort = 0
  form.enabled = true
  form.description = ''
  currentId.value = ''
}

const fetchPosts = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/v1/system/posts', {
      headers: authHeaders(),
      params: {
        keyword: keyword.value || undefined,
        status: statusFilter.value === '' ? undefined : statusFilter.value
      }
    })
    if (res.data?.code === 0) {
      posts.value = res.data.data || []
    }
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '获取岗位列表失败'))
  } finally {
    loading.value = false
  }
}

const openCreate = () => {
  isEdit.value = false
  resetForm()
  dialogVisible.value = true
}

const openEdit = (row) => {
  isEdit.value = true
  currentId.value = row.id
  form.name = row.name || ''
  form.code = row.code || ''
  form.sort = Number(row.sort || 0)
  form.enabled = Number(row.status) === 1
  form.description = row.description || ''
  dialogVisible.value = true
}

const savePost = async () => {
  if (!form.name.trim()) {
    ElMessage.warning('请输入岗位名称')
    return
  }
  if (!form.code.trim()) {
    ElMessage.warning('请输入岗位编码')
    return
  }

  saving.value = true
  try {
    const payload = {
      name: form.name.trim(),
      code: form.code.trim(),
      sort: Number(form.sort || 0),
      status: form.enabled ? 1 : 0,
      description: form.description?.trim() || ''
    }

    if (isEdit.value && currentId.value) {
      await axios.put(`/api/v1/system/posts/${currentId.value}`, payload, { headers: authHeaders() })
      ElMessage.success('更新成功')
    } else {
      await axios.post('/api/v1/system/posts', payload, { headers: authHeaders() })
      ElMessage.success('创建成功')
    }

    dialogVisible.value = false
    await fetchPosts()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '保存失败'))
  } finally {
    saving.value = false
  }
}

const removePost = async (row) => {
  try {
    await ElMessageBox.confirm(`确认删除岗位 ${row.name} ?`, '提示', { type: 'warning' })
    await axios.delete(`/api/v1/system/posts/${row.id}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    await fetchPosts()
  } catch (err) {
    if (err !== 'cancel') {
      ElMessage.error(getErrorMessage(err, '删除失败'))
    }
  }
}

onMounted(fetchPosts)
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.page-header { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-bottom: 12px; }
.page-desc { color: #909399; margin: 4px 0 0; }
.page-actions { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.table-scroll { overflow-x: auto; }
</style>
