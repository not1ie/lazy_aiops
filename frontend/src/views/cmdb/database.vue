<template>
  <el-card class="page-card">
    <template #header>
      <div class="header">
        <div>
          <div class="title">数据库资产</div>
          <div class="desc">管理数据库实例与连接信息</div>
        </div>
        <div class="actions">
          <el-button type="primary" icon="Plus" @click="openCreate">新增资产</el-button>
          <el-button icon="Upload" @click="openImport">批量导入</el-button>
          <el-button icon="Download" @click="exportCSV">导出</el-button>
          <el-button type="danger" plain icon="Delete" :disabled="selectedRows.length === 0" @click="handleBatchDelete">
            批量删除 ({{ selectedRows.length }})
          </el-button>
          <el-button icon="Refresh" @click="fetchData">刷新</el-button>
        </div>
      </div>
    </template>

    <div class="filters">
      <el-input v-model="filters.keyword" placeholder="名称/主机" clearable @clear="fetchData" @keyup.enter="fetchData">
        <template #append>
          <el-button icon="Search" @click="fetchData" />
        </template>
      </el-input>
      <el-select v-model="filters.environment" placeholder="环境" clearable @change="fetchData">
        <el-option label="开发" value="dev" />
        <el-option label="测试" value="test" />
        <el-option label="生产" value="prod" />
      </el-select>
    </div>

    <el-table :fit="true" :data="items" v-loading="loading" stripe @selection-change="selectedRows = $event">
      <el-table-column type="selection" width="48" />
      <el-table-column prop="name" label="名称" min-width="160" />
      <el-table-column prop="type" label="类型" width="120" />
      <el-table-column label="地址" min-width="180">
        <template #default="{ row }">
          {{ row.host }}:{{ row.port }}
        </template>
      </el-table-column>
      <el-table-column prop="database" label="库名" min-width="140" />
      <el-table-column prop="environment" label="环境" width="100" />
      <el-table-column prop="owner" label="负责人" width="120" />
      <el-table-column prop="status" label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="row.status === 1 ? 'success' : 'danger'">{{ row.status === 1 ? '正常' : '禁用' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="240" fixed="right">
        <template #default="{ row }">
          <el-space size="8">
            <el-button size="small" type="warning" plain icon="FirstAidKit" @click="openTest(row)">测试</el-button>
            <el-button size="small" type="primary" plain @click="openEdit(row)">编辑</el-button>
            <el-button size="small" type="danger" plain @click="handleDelete(row)">删除</el-button>
          </el-space>
        </template>
      </el-table-column>
    </el-table>
  </el-card>

  <el-dialog append-to-body v-model="dialogVisible" :title="isEdit ? '编辑数据库资产' : '新增数据库资产'" width="560px">
    <el-form :model="form" label-width="100px">
      <el-form-item label="名称" required>
        <el-input v-model="form.name" />
      </el-form-item>
      <el-form-item label="类型" required>
        <el-select v-model="form.type" placeholder="请选择类型" style="width: 100%">
          <el-option label="MySQL" value="mysql" />
          <el-option label="PostgreSQL" value="postgres" />
          <el-option label="Redis" value="redis" />
          <el-option label="MongoDB" value="mongodb" />
          <el-option label="Oracle" value="oracle" />
        </el-select>
      </el-form-item>
      <el-form-item label="主机" required>
        <el-input v-model="form.host" />
      </el-form-item>
      <el-form-item label="端口">
        <el-input-number v-model="form.port" :min="1" :max="65535" />
      </el-form-item>
      <el-form-item label="库名">
        <el-input v-model="form.database" />
      </el-form-item>
      <el-form-item label="用户名">
        <el-input v-model="form.username" />
      </el-form-item>
      <el-form-item label="密码">
        <el-input v-model="form.password" type="password" show-password />
      </el-form-item>
      <el-form-item label="环境">
        <el-select v-model="form.environment" style="width: 100%">
          <el-option label="开发" value="dev" />
          <el-option label="测试" value="test" />
          <el-option label="生产" value="prod" />
        </el-select>
      </el-form-item>
      <el-form-item label="负责人">
        <el-input v-model="form.owner" />
      </el-form-item>
      <el-form-item label="标签">
        <el-input v-model="form.tags" placeholder="逗号分隔" />
      </el-form-item>
      <el-form-item label="状态">
        <el-switch v-model="form.status" :active-value="1" :inactive-value="0" />
      </el-form-item>
      <el-form-item label="描述">
        <el-input v-model="form.description" type="textarea" :rows="3" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="dialogVisible = false">取消</el-button>
      <el-button type="primary" :loading="saving" @click="saveItem">保存</el-button>
    </template>
  </el-dialog>

  <el-dialog append-to-body v-model="importVisible" title="批量导入数据库资产" width="720px">
    <el-alert type="info" :closable="false" show-icon>
      格式：name,type,host,port,username,password,database,environment,owner,tags,status,description（第一行可写表头）
    </el-alert>
    <el-input v-model="importText" type="textarea" :rows="10" />
    <div class="import-actions">
      <el-button @click="importVisible = false">取消</el-button>
      <el-button type="primary" :loading="importLoading" @click="submitImport">开始导入</el-button>
    </div>
  </el-dialog>

  <el-dialog append-to-body v-model="testVisible" title="数据库连通性测试" width="560px">
    <el-alert v-if="testError" type="error" :closable="false" show-icon>{{ testError }}</el-alert>
    <el-alert v-if="testSuccess" type="success" :closable="false" show-icon>{{ testSuccess }}</el-alert>
    <template #footer>
      <el-button @click="testVisible = false">关闭</el-button>
      <el-button type="primary" :loading="testLoading" @click="submitTest">测试</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const loading = ref(false)
const saving = ref(false)
const items = ref([])
const dialogVisible = ref(false)
const isEdit = ref(false)
const currentId = ref('')
const selectedRows = ref([])

const importVisible = ref(false)
const importLoading = ref(false)
const importText = ref('')

const testVisible = ref(false)
const testLoading = ref(false)
const testRow = ref(null)
const testError = ref('')
const testSuccess = ref('')

const filters = reactive({
  keyword: '',
  environment: ''
})

const form = reactive({
  name: '',
  type: 'mysql',
  host: '',
  port: 3306,
  username: '',
  password: '',
  database: '',
  environment: 'dev',
  owner: '',
  tags: '',
  status: 1,
  description: ''
})

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const getErrorMessage = (error, fallback) => {
  if (error?.response?.data?.message) return error.response.data.message
  if (error?.message) return error.message
  return fallback
}

const fetchData = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/v1/cmdb/databases', {
      params: { keyword: filters.keyword, environment: filters.environment },
      headers: authHeaders()
    })
    if (res.data.code === 0) {
      items.value = res.data.data
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
  Object.assign(form, {
    name: '',
    type: 'mysql',
    host: '',
    port: 3306,
    username: '',
    password: '',
    database: '',
    environment: 'dev',
    owner: '',
    tags: '',
    status: 1,
    description: ''
  })
  dialogVisible.value = true
}

const openEdit = (row) => {
  isEdit.value = true
  currentId.value = row.id
  Object.assign(form, row)
  dialogVisible.value = true
}

const saveItem = async () => {
  if (!form.name || !form.host) {
    ElMessage.warning('请填写名称与主机')
    return
  }
  saving.value = true
  try {
    const url = isEdit.value ? `/api/v1/cmdb/databases/${currentId.value}` : '/api/v1/cmdb/databases'
    const method = isEdit.value ? 'put' : 'post'
    const res = await axios({
      url,
      method,
      data: form,
      headers: authHeaders()
    })
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

const handleDelete = async (row) => {
  try {
    await ElMessageBox.confirm(`确定删除资产“${row.name}”吗？`, '提示', {
      type: 'warning'
    })
    await axios.delete(`/api/v1/cmdb/databases/${row.id}`, {
      headers: authHeaders()
    })
    ElMessage.success('删除成功')
    await fetchData()
  } catch (error) {
    if (error !== 'cancel' && error !== 'close') {
      ElMessage.error(getErrorMessage(error, '删除失败'))
    }
  }
}

const handleBatchDelete = async () => {
  if (selectedRows.value.length === 0) return
  try {
    await ElMessageBox.confirm(`确定删除选中的 ${selectedRows.value.length} 个资产吗？`, '提示', {
      type: 'warning'
    })
    for (const row of selectedRows.value) {
      await axios.delete(`/api/v1/cmdb/databases/${row.id}`, { headers: authHeaders() })
    }
    ElMessage.success('批量删除成功')
    selectedRows.value = []
    await fetchData()
  } catch (error) {
    if (error !== 'cancel' && error !== 'close') {
      ElMessage.error(getErrorMessage(error, '批量删除失败'))
    }
  }
}

const openImport = () => {
  importText.value = ''
  importVisible.value = true
}

const parseCSV = (text) => {
  const lines = text.split(/\r?\n/).map(l => l.trim()).filter(Boolean)
  if (lines.length === 0) return []
  const delim = lines[0].includes('\t') ? '\t' : ','
  const headers = lines[0].toLowerCase().split(delim).map(s => s.trim())
  const hasHeader = headers.includes('name') || headers.includes('host')
  const start = hasHeader ? 1 : 0
  const cols = hasHeader ? headers : ['name','type','host','port','username','password','database','environment','owner','tags','status','description']
  return lines.slice(start).map(line => {
    const parts = line.split(delim).map(s => s.trim())
    const obj = {}
    cols.forEach((k, idx) => { obj[k] = parts[idx] || '' })
    return obj
  })
}

const submitImport = async () => {
  const rows = parseCSV(importText.value)
  if (rows.length === 0) {
    ElMessage.warning('请填写导入内容')
    return
  }
  importLoading.value = true
  try {
    for (const row of rows) {
      if (!row.name || !row.host) continue
      await axios.post('/api/v1/cmdb/databases', {
        name: row.name,
        type: row.type || 'mysql',
        host: row.host,
        port: row.port ? Number(row.port) : 3306,
        username: row.username || '',
        password: row.password || '',
        database: row.database || '',
        environment: row.environment || 'dev',
        owner: row.owner || '',
        tags: row.tags || '',
        status: row.status ? Number(row.status) : 1,
        description: row.description || ''
      }, { headers: authHeaders() })
    }
    ElMessage.success('导入完成')
    importVisible.value = false
    await fetchData()
  } catch (e) {
    ElMessage.error(getErrorMessage(e, '导入失败'))
  } finally {
    importLoading.value = false
  }
}

const exportCSV = () => {
  const headers = ['name','type','host','port','database','environment','owner','status']
  const rows = items.value.map(d => [d.name, d.type, d.host, d.port, d.database, d.environment, d.owner, d.status])
  const csv = [headers.join(','), ...rows.map(r => r.map(v => `"${String(v ?? '').replace(/"/g, '""')}"`).join(','))].join('\n')
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'cmdb_databases.csv'
  a.click()
  URL.revokeObjectURL(url)
}

const openTest = (row) => {
  testRow.value = row
  testError.value = ''
  testSuccess.value = ''
  testVisible.value = true
}

const submitTest = async () => {
  if (!testRow.value) return
  testLoading.value = true
  try {
    const res = await axios.post(`/api/v1/cmdb/databases/${testRow.value.id}/test`, {}, { headers: authHeaders() })
    if (res.data.code === 0) {
      testSuccess.value = res.data.message || '连接成功'
    } else {
      testError.value = res.data.message || '连接失败'
    }
  } catch (e) {
    testError.value = getErrorMessage(e, '连接失败')
  } finally {
    testLoading.value = false
  }
}

onMounted(fetchData)
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.header { display: flex; justify-content: space-between; align-items: center; }
.title { font-size: 18px; font-weight: 600; }
.desc { color: #909399; margin-top: 4px; }
.actions { display: flex; gap: 8px; align-items: center; }
.filters { display: flex; gap: 12px; margin-bottom: 16px; }
.filters .el-input { width: 240px; }
.filters .el-select { width: 160px; }
.import-actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 12px; }
</style>
