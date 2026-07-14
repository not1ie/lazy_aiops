<template>
  <div class="log-library-container" v-loading="loading">
    <div class="page-header">
      <div class="header-title">
        <div class="icon-box"><el-icon><Collection /></el-icon></div>
        <div>
          <h2>日志库管理</h2>
          <p>统一管理系统中的 Elasticsearch 索引与 Loki 数据流</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button type="primary" icon="Plus" @click="handleAdd">接入日志库</el-button>
      </div>
    </div>

    <el-card class="table-card">
      <div class="toolbar">
        <el-input v-model="searchKey" placeholder="搜索库名称..." prefix-icon="Search" style="width: 240px" clearable @input="filterData" />
        <el-select v-model="typeFilter" placeholder="存储类型" style="width: 140px; margin-left: 12px" clearable @change="filterData">
          <el-option label="Elasticsearch" value="es" />
          <el-option label="Loki" value="loki" />
        </el-select>
        <el-button icon="Refresh" circle style="margin-left: auto" @click="fetchData" />
      </div>

      <el-table :data="filteredTableData" style="width: 100%" v-loading="tableLoading">
        <el-table-column prop="name" label="日志库名称" min-width="150">
          <template #default="{ row }">
            <span class="lib-name">{{ row.name }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="type" label="存储类型" width="120">
          <template #default="{ row }">
            <el-tag :type="row.type === 'es' ? 'warning' : 'primary'" size="small">
              {{ row.type === 'es' ? 'Elasticsearch' : 'Loki' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="source" label="数据源" min-width="180" show-overflow-tooltip />
        <el-table-column prop="retention" label="保留策略" width="100" />
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tooltip :content="row.status_reason || '正常'" placement="top" :disabled="!row.status_reason">
              <div class="status-indicator" :class="row.status">
                <span class="dot"></span>
                {{ row.status === 'active' ? '正常' : '异常' }}
              </div>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="接入时间" width="180">
          <template #default="{ row }">
            {{ formatTime(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="handleEdit(row)">编辑</el-button>
            <el-button link type="primary" size="small" @click="handleQuery(row)">查询</el-button>
            <el-button link type="danger" size="small" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- Dialog to Add/Edit Log Library -->
    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="550px" destroy-on-close>
      <el-form :model="form" :rules="rules" ref="formRef" label-width="100px">
        <el-form-item label="库名称" prop="name">
          <el-input v-model="form.name" placeholder="请输入库名称，例如: k8s-app-logs" :disabled="isEdit" />
        </el-form-item>
        <el-form-item label="存储类型" prop="type">
          <el-select v-model="form.type" placeholder="选择存储类型" style="width: 100%" :disabled="isEdit">
            <el-option label="Elasticsearch" value="es" />
            <el-option label="Loki" value="loki" />
          </el-select>
        </el-form-item>
        <el-form-item label="数据源" prop="source">
          <div class="source-input-group">
            <el-input v-model="form.source" placeholder="请输入数据源地址，例如: http://localhost:9200" style="flex: 1" />
            <el-button type="success" :loading="testing" @click="testConnection" style="margin-left: 8px">测试连接</el-button>
          </div>
        </el-form-item>
        <el-form-item label="保留策略" prop="retention">
          <el-input v-model="form.retention" placeholder="请输入保留天数/时长，例如: 30天" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" :loading="submitLoading" @click="handleSubmit">确定</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Collection, Plus, Search, Refresh } from '@element-plus/icons-vue'

const router = useRouter()

const loading = ref(false)
const tableLoading = ref(false)
const searchKey = ref('')
const typeFilter = ref('')

const tableData = ref([])
const filteredTableData = ref([])

const dialogVisible = ref(false)
const dialogTitle = ref('')
const isEdit = ref(false)
const submitLoading = ref(false)
const testing = ref(false)
const formRef = ref(null)

const form = ref({
  id: '',
  name: '',
  type: 'es',
  source: '',
  retention: '30天'
})

const rules = {
  name: [
    { required: true, message: '请输入日志库名称', trigger: 'blur' },
    { pattern: /^[a-zA-Z0-9_-]+$/, message: '库名称只能包含字母、数字、中划线和下划线', trigger: 'blur' }
  ],
  type: [{ required: true, message: '请选择存储类型', trigger: 'change' }],
  source: [{ required: true, message: '请输入数据源地址', trigger: 'blur' }],
  retention: [{ required: true, message: '请输入保留策略', trigger: 'blur' }]
}

const authHeaders = () => ({
  Authorization: `Bearer ${localStorage.getItem('token')}`
})

const formatTime = (timeStr) => {
  if (!timeStr) return '-'
  const date = new Date(timeStr)
  return date.toLocaleString()
}

const filterData = () => {
  let data = [...tableData.value]
  if (searchKey.value) {
    const key = searchKey.value.toLowerCase()
    data = data.filter(item => 
      item.name.toLowerCase().includes(key) || 
      item.source.toLowerCase().includes(key)
    )
  }
  if (typeFilter.value) {
    data = data.filter(item => item.type === typeFilter.value)
  }
  filteredTableData.value = data
}

const fetchData = async () => {
  tableLoading.value = true
  try {
    const res = await axios.get('/api/v1/log/libraries', { headers: authHeaders() })
    if (res.data.code === 0) {
      tableData.value = res.data.data
      filterData()
    } else {
      ElMessage.error(res.data.message || '获取日志库列表失败')
    }
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '获取数据失败，请重试')
  } finally {
    tableLoading.value = false
  }
}

const handleAdd = () => {
  isEdit.value = false
  dialogTitle.value = '接入日志库'
  form.value = {
    id: '',
    name: '',
    type: 'es',
    source: '',
    retention: '30天'
  }
  dialogVisible.value = true
}

const handleEdit = (row) => {
  isEdit.value = true
  dialogTitle.value = '编辑日志库'
  form.value = { ...row }
  dialogVisible.value = true
}

const handleQuery = (row) => {
  router.push({ name: 'LogQuery', query: { library_id: row.id } })
}

const handleDelete = (row) => {
  ElMessageBox.confirm(`确定删除日志库 "${row.name}" 吗？此操作不可逆。`, '删除确认', {
    type: 'warning',
    confirmButtonText: '删除',
    cancelButtonText: '取消'
  }).then(async () => {
    try {
      const res = await axios.delete(`/api/v1/log/libraries/${row.id}`, { headers: authHeaders() })
      if (res.data.code === 0) {
        ElMessage.success('删除成功')
        fetchData()
      } else {
        ElMessage.error(res.data.message || '删除失败')
      }
    } catch (err) {
      ElMessage.error('删除请求出错，请重试')
    }
  }).catch(() => {})
}

const testConnection = async () => {
  if (!form.value.source) {
    ElMessage.warning('请输入数据源地址进行测试')
    return
  }
  testing.value = true
  try {
    const res = await axios.post('/api/v1/log/libraries/test', {
      source: form.value.source,
      type: form.value.type
    }, { headers: authHeaders() })
    if (res.data.code === 0) {
      if (res.data.status === 'active') {
        ElMessage.success(res.data.message || '连通性测试成功！')
      } else {
        ElMessage.error(res.data.message || '测试失败：无法连通该地址')
      }
    } else {
      ElMessage.error(res.data.message || '连通性测试接口出错')
    }
  } catch (err) {
    ElMessage.error('测试请求超时或失败')
  } finally {
    testing.value = false
  }
}

const handleSubmit = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    submitLoading.value = true
    try {
      let res
      if (isEdit.value) {
        res = await axios.put(`/api/v1/log/libraries/${form.value.id}`, form.value, { headers: authHeaders() })
      } else {
        res = await axios.post('/api/v1/log/libraries', form.value, { headers: authHeaders() })
      }
      if (res.data.code === 0) {
        ElMessage.success(isEdit.value ? '保存成功' : '接入成功')
        dialogVisible.value = false
        fetchData()
      } else {
        ElMessage.error(res.data.message || '保存失败')
      }
    } catch (err) {
      ElMessage.error(err.response?.data?.message || '操作请求失败')
    } finally {
      submitLoading.value = false
    }
  })
}

let timer = null
onMounted(() => {
  fetchData()
  timer = setInterval(fetchData, 10000)
})

onBeforeUnmount(() => {
  if (timer) {
    clearInterval(timer)
  }
})
</script>

<style scoped>
.log-library-container {
  padding-bottom: 20px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}
.header-title {
  display: flex;
  align-items: center;
  gap: 16px;
}
.icon-box {
  width: 48px;
  height: 48px;
  background-color: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
}
.header-title h2 {
  margin: 0 0 4px 0;
  font-size: 20px;
  color: #1f2329;
}
.header-title p {
  margin: 0;
  font-size: 13px;
  color: #8f959e;
}

.table-card {
  border-radius: 12px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.02) !important;
}

.toolbar {
  display: flex;
  align-items: center;
  margin-bottom: 16px;
}

.lib-name {
  font-weight: 500;
  color: #1f2329;
}

.status-indicator {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
}
.status-indicator .dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
}
.status-indicator.active .dot { background-color: var(--el-color-success); }
.status-indicator.error .dot { background-color: var(--el-color-danger); }

.source-input-group {
  display: flex;
  width: 100%;
}
</style>
