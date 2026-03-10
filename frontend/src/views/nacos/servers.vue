<template>
  <el-card class="page-card">
    <template #header>
      <div class="header">
        <div>
          <div class="title">Nacos 服务器</div>
          <div class="desc">Nacos 实例与命名空间管理</div>
        </div>
        <div class="actions">
          <el-button type="primary" icon="Plus" @click="openCreate">新增服务器</el-button>
          <el-button icon="Refresh" @click="fetchData">刷新</el-button>
        </div>
      </div>
    </template>

    <el-table :fit="true" :data="servers" v-loading="loading" stripe>
      <el-table-column prop="name" label="名称" min-width="160" />
      <el-table-column prop="address" label="地址" min-width="220" />
      <el-table-column prop="namespace" label="Namespace" min-width="160" />
      <el-table-column prop="status" label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="row.status === 1 ? 'success' : 'danger'">{{ row.status === 1 ? '正常' : '异常' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
      <el-table-column label="操作" width="300" fixed="right">
        <template #default="{ row }">
          <el-button size="small" type="primary" plain @click="openEdit(row)">编辑</el-button>
          <el-button size="small" type="success" plain @click="testConnection(row)">测试</el-button>
          <el-button size="small" type="warning" plain @click="syncConfigs(row)">同步配置</el-button>
          <el-button size="small" type="danger" plain @click="handleDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-card>

  <el-dialog append-to-body v-model="dialogVisible" :title="isEdit ? '编辑服务器' : '新增服务器'" width="600px">
    <el-form :model="form" label-width="100px">
      <el-form-item label="名称" required>
        <el-input v-model="form.name" />
      </el-form-item>
      <el-form-item label="地址" required>
        <el-input v-model="form.address" placeholder="http://127.0.0.1:8848" />
      </el-form-item>
      <el-form-item label="Namespace">
        <el-input v-model="form.namespace" placeholder="可选" />
      </el-form-item>
      <el-form-item label="用户名">
        <el-input v-model="form.username" />
      </el-form-item>
      <el-form-item label="密码">
        <el-input v-model="form.password" type="password" show-password />
        <div v-if="isEdit" class="helper-row">已加载当前密码，可直接修改。</div>
      </el-form-item>
      <el-form-item label="描述">
        <el-input v-model="form.description" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="dialogVisible = false">取消</el-button>
      <el-button type="primary" :loading="saving" @click="saveServer">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const servers = ref([])
const loading = ref(false)
const saving = ref(false)
const dialogVisible = ref(false)
const isEdit = ref(false)
const currentId = ref('')

const form = reactive({
  name: '',
  address: '',
  namespace: '',
  username: '',
  password: '',
  description: ''
})

const headers = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const getErrorMessage = (error, fallback) => {
  if (error?.response?.data?.message) return error.response.data.message
  if (error?.message) return error.message
  return fallback
}

const fetchData = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/v1/nacos/servers', { headers: headers() })
    if (res.data.code === 0) {
      servers.value = res.data.data
    }
  } catch (error) {
    ElMessage.error(getErrorMessage(error, '加载失败'))
  } finally {
    loading.value = false
  }
}

const openCreate = () => {
  isEdit.value = false
  currentId.value = ''
  Object.assign(form, { name: '', address: '', namespace: '', username: '', password: '', description: '' })
  dialogVisible.value = true
}

const openEdit = async (row) => {
  isEdit.value = true
  currentId.value = row.id
  try {
    const res = await axios.get(`/api/v1/nacos/servers/${row.id}`, { headers: headers() })
    if (res.data.code === 0) {
      Object.assign(form, res.data.data || {})
      dialogVisible.value = true
    }
  } catch (error) {
    ElMessage.error(getErrorMessage(error, '获取详情失败'))
  }
}

const saveServer = async () => {
  if (!form.name || !form.address) {
    ElMessage.warning('请填写名称与地址')
    return
  }
  saving.value = true
  try {
    const url = isEdit.value ? `/api/v1/nacos/servers/${currentId.value}` : '/api/v1/nacos/servers'
    const method = isEdit.value ? 'put' : 'post'
    const res = await axios({ url, method, data: form, headers: headers() })
    if (res.data.code === 0) {
      ElMessage.success(isEdit.value ? '更新成功' : '创建成功')
      dialogVisible.value = false
      await fetchData()
    }
  } catch (error) {
    ElMessage.error(getErrorMessage(error, '保存失败'))
  } finally {
    saving.value = false
  }
}

const testConnection = async (row) => {
  try {
    const res = await axios.post(`/api/v1/nacos/servers/${row.id}/test`, {}, { headers: headers() })
    if (res.data.code === 0 && res.data.data.success) {
      ElMessage.success('连接成功')
    } else {
      ElMessage.error(`连接失败: ${res.data.data?.error || res.data.message || '未知错误'}`)
    }
    await fetchData()
  } catch (error) {
    ElMessage.error(getErrorMessage(error, '连接失败'))
  }
}

const syncConfigs = async (row) => {
  try {
    await axios.post(`/api/v1/nacos/servers/${row.id}/sync-configs`, {}, { headers: headers() })
    ElMessage.success('同步完成')
  } catch (error) {
    ElMessage.error(getErrorMessage(error, '同步失败'))
  }
}

const handleDelete = async (row) => {
  try {
    await ElMessageBox.confirm(`确定删除“${row.name}”吗？`, '提示', { type: 'warning' })
    await axios.delete(`/api/v1/nacos/servers/${row.id}`, { headers: headers() })
    ElMessage.success('删除成功')
    await fetchData()
  } catch (error) {
    if (error !== 'cancel' && error !== 'close') {
      ElMessage.error(getErrorMessage(error, '删除失败'))
    }
  }
}

onMounted(fetchData)
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.header { display: flex; justify-content: space-between; align-items: center; }
.title { font-size: 18px; font-weight: 600; }
.desc { color: #909399; margin-top: 4px; }
.actions { display: flex; gap: 8px; }
.helper-row { margin-top: 6px; color: var(--el-text-color-secondary); font-size: 12px; line-height: 1.4; }
</style>
