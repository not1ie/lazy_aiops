<template>
  <el-card>
    <template #header>
      <div class="flex justify-between items-center">
        <span class="font-bold">CMDB 主机管理</span>
        <div>
          <el-button type="primary" icon="Plus" @click="handleAdd">添加主机</el-button>
          <el-button icon="Upload" @click="openImport">批量导入</el-button>
          <el-button icon="Download" @click="exportCSV">导出</el-button>
          <el-button type="warning" plain icon="Edit" :disabled="selectedRows.length === 0" @click="openBatchStatus">
            批量状态
          </el-button>
          <el-button type="danger" plain icon="Delete" :disabled="selectedRows.length === 0" @click="handleBatchDelete">
            批量删除 ({{ selectedRows.length }})
          </el-button>
          <el-button icon="Refresh" @click="fetchData">刷新</el-button>
        </div>
      </div>
    </template>

    <div class="mb-4">
      <div class="flex gap-2 items-center">
        <el-input v-model="searchKeyword" placeholder="搜索主机名或IP" class="w-64" clearable @clear="fetchData" @keyup.enter="fetchData">
          <template #append>
            <el-button icon="Search" @click="fetchData" />
          </template>
        </el-input>
        <el-select v-model="filterGroupId" placeholder="分组" clearable class="w-64" @change="fetchData">
          <el-option v-for="g in groups" :key="g.id" :label="g.name" :value="g.id" />
        </el-select>
      </div>
    </div>

    <el-table :fit="true" :data="tableData" v-loading="loading" style="width: 100%" @selection-change="selectedRows = $event">
      <el-table-column type="selection" width="48" />
      <el-table-column prop="name" label="主机名" width="180">
        <template #default="{ row }">
          <div class="flex items-center gap-2">
            <el-icon class="text-gray-500 text-lg"><Monitor /></el-icon>
            <span class="font-bold">{{ row.name }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="ip" label="IP地址" width="150" />
      <el-table-column prop="os" label="操作系统" width="150" />
      <el-table-column prop="status" label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="row.status === 1 ? 'success' : row.status === 2 ? 'warning' : 'danger'">
            {{ row.status === 1 ? '在线' : row.status === 2 ? '维护' : '离线' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="group.name" label="分组" width="150" />
      <el-table-column label="操作" width="260" fixed="right">
        <template #default="{ row }">
          <el-button size="small" type="warning" plain icon="FirstAidKit" @click="handleTest(row)">测试</el-button>
          <el-button size="small" type="primary" plain icon="Edit" @click="handleEdit(row)">编辑</el-button>
          <el-button size="small" type="danger" plain icon="Delete" @click="handleDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 添加/编辑主机弹窗 -->
    <el-dialog append-to-body v-model="dialogVisible" :title="isEdit ? '编辑主机' : '添加主机'" width="500px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="主机名" required>
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="IP地址" required>
          <el-input v-model="form.ip" />
        </el-form-item>
        <el-form-item label="端口">
          <el-input-number v-model="form.port" :min="1" :max="65535" />
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="form.status" style="width: 100%">
            <el-option label="在线" :value="1" />
            <el-option label="离线" :value="0" />
            <el-option label="维护" :value="2" />
          </el-select>
        </el-form-item>
        <el-form-item label="用户名">
          <el-input v-model="form.username" placeholder="root" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="form.password" type="password" show-password placeholder="如有变更请填写" />
        </el-form-item>
        <el-form-item label="分组">
          <el-input v-model="form.group_name" placeholder="默认分组" :disabled="isEdit" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitForm" :loading="submitting">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog append-to-body v-model="testVisible" title="主机测试" width="640px">
      <el-alert v-if="testError" type="error" :closable="false" show-icon>{{ testError }}</el-alert>
      <el-skeleton v-if="testLoading" :rows="4" animated />
      <div v-else class="test-block">
        <div class="test-title">uname -a</div>
        <pre class="test-pre">{{ testResult?.uname?.out || '-' }}</pre>
        <div class="test-title">/etc/os-release</div>
        <pre class="test-pre">{{ testResult?.os_release?.out || '-' }}</pre>
      </div>
      <template #footer>
        <el-button @click="testVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- 批量导入 -->
    <el-dialog append-to-body v-model="importVisible" title="批量导入主机" width="720px">
      <el-alert type="info" :closable="false" show-icon>
        格式：name,ip,port,username,password,group_name,status,os（第一行可写表头）
      </el-alert>
      <el-input v-model="importText" type="textarea" :rows="10" placeholder="例如：&#10;web-1,192.168.1.10,22,root,pass,prod,1,Ubuntu" />
      <div class="import-actions">
        <el-button @click="importVisible = false">取消</el-button>
        <el-button type="primary" :loading="importLoading" @click="submitImport">开始导入</el-button>
      </div>
    </el-dialog>

    <!-- 批量状态 -->
    <el-dialog append-to-body v-model="batchStatusVisible" title="批量修改状态" width="420px">
      <el-form label-width="80px">
        <el-form-item label="状态">
          <el-select v-model="batchStatus" placeholder="选择状态" style="width: 100%">
            <el-option label="在线" :value="1" />
            <el-option label="离线" :value="0" />
            <el-option label="维护" :value="2" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="batchStatusVisible = false">取消</el-button>
        <el-button type="primary" :loading="batchStatusLoading" @click="submitBatchStatus">保存</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const loading = ref(false)
const tableData = ref([])
const groups = ref([])
const filterGroupId = ref('')
const selectedRows = ref([])
const searchKeyword = ref('')
const dialogVisible = ref(false)
const submitting = ref(false)
const isEdit = ref(false)
const currentId = ref('')
const testVisible = ref(false)
const testLoading = ref(false)
const testResult = ref(null)
const testError = ref('')
const importVisible = ref(false)
const importLoading = ref(false)
const importText = ref('')
const batchStatusVisible = ref(false)
const batchStatusLoading = ref(false)
const batchStatus = ref(null)

const form = reactive({
  name: '',
  ip: '',
  port: 22,
  status: 1,
  username: '',
  password: '',
  group_name: ''
})

const fetchData = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/v1/cmdb/hosts', {
      headers: { Authorization: 'Bearer ' + localStorage.getItem('token') },
      params: { keyword: searchKeyword.value, group_id: filterGroupId.value }
    })
    if (res.data.code === 0) {
      tableData.value = res.data.data
    }
  } catch (e) {
    ElMessage.error('加载失败')
  } finally {
    loading.value = false
  }
}

const fetchGroups = async () => {
  try {
    const res = await axios.get('/api/v1/cmdb/groups', {
      headers: { Authorization: 'Bearer ' + localStorage.getItem('token') }
    })
    if (res.data.code === 0) {
      groups.value = res.data.data
    }
  } catch (e) {}
}

const handleAdd = () => {
  isEdit.value = false
  form.name = ''
  form.ip = ''
  form.port = 22
  form.status = 1
  form.username = 'root'
  form.password = ''
  form.group_name = ''
  dialogVisible.value = true
}

const handleEdit = async (row) => {
  isEdit.value = true
  currentId.value = row.id
  // Load detail
  try {
    const res = await axios.get(`/api/v1/cmdb/hosts/${row.id}`, {
      headers: { Authorization: 'Bearer ' + localStorage.getItem('token') }
    })
    if (res.data.code === 0) {
      const data = res.data.data
      form.name = data.name
      form.ip = data.ip
      form.port = data.port
      form.status = data.status ?? 1
      form.username = data.credential ? data.credential.username : ''
      form.password = data.credential ? data.credential.password : ''
      form.group_name = data.group ? data.group.name : ''
      dialogVisible.value = true
    }
  } catch (e) {
    ElMessage.error('获取详情失败')
  }
}

const submitForm = async () => {
  submitting.value = true
  try {
    const url = isEdit.value ? `/api/v1/cmdb/hosts/${currentId.value}` : '/api/v1/cmdb/hosts'
    const method = isEdit.value ? 'put' : 'post'
    
    const res = await axios({
      method,
      url,
      data: form,
      headers: { Authorization: 'Bearer ' + localStorage.getItem('token') }
    })
    
    if (res.data.code === 0) {
      ElMessage.success(isEdit.value ? '更新成功' : '添加成功')
      dialogVisible.value = false
      fetchData()
    } else {
      ElMessage.error(res.data.message)
    }
  } finally {
    submitting.value = false
  }
}

const handleDelete = (row) => {
  ElMessageBox.confirm('确定删除该主机吗?', '警告', {
    confirmButtonText: '删除',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    await axios.delete(`/api/v1/cmdb/hosts/${row.id}`, {
      headers: { Authorization: 'Bearer ' + localStorage.getItem('token') }
    })
    ElMessage.success('删除成功')
    fetchData()
  })
}

const handleBatchDelete = () => {
  if (selectedRows.value.length === 0) return
  ElMessageBox.confirm(`确定删除选中的 ${selectedRows.value.length} 台主机吗?`, '警告', {
    confirmButtonText: '删除',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    for (const row of selectedRows.value) {
      await axios.delete(`/api/v1/cmdb/hosts/${row.id}`, {
        headers: { Authorization: 'Bearer ' + localStorage.getItem('token') }
      })
    }
    ElMessage.success('批量删除成功')
    selectedRows.value = []
    fetchData()
  })
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
  const hasHeader = headers.includes('name') || headers.includes('ip')
  const start = hasHeader ? 1 : 0
  const cols = hasHeader ? headers : ['name','ip','port','username','password','group_name','status','os']
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
      if (!row.name || !row.ip) continue
      await axios.post('/api/v1/cmdb/hosts', {
        name: row.name,
        ip: row.ip,
        port: row.port ? Number(row.port) : 22,
        username: row.username || '',
        password: row.password || '',
        group_name: row.group_name || '',
        status: row.status ? Number(row.status) : 1,
        os: row.os || ''
      }, {
        headers: { Authorization: 'Bearer ' + localStorage.getItem('token') }
      })
    }
    ElMessage.success('导入完成')
    importVisible.value = false
    fetchData()
  } catch (e) {
    ElMessage.error('导入失败')
  } finally {
    importLoading.value = false
  }
}

const exportCSV = () => {
  const headers = ['name','ip','port','os','status','group','username']
  const rows = tableData.value.map(h => [
    h.name, h.ip, h.port, h.os, h.status,
    h.group?.name || '', h.credential?.username || ''
  ])
  const csv = [headers.join(','), ...rows.map(r => r.map(v => `"${String(v ?? '').replace(/"/g, '""')}"`).join(','))].join('\\n')
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'cmdb_hosts.csv'
  a.click()
  URL.revokeObjectURL(url)
}

const openBatchStatus = () => {
  batchStatus.value = 1
  batchStatusVisible.value = true
}

const submitBatchStatus = async () => {
  if (batchStatus.value === null) return
  batchStatusLoading.value = true
  try {
    for (const row of selectedRows.value) {
      await axios.put(`/api/v1/cmdb/hosts/${row.id}`, {
        ...row,
        status: batchStatus.value
      }, {
        headers: { Authorization: 'Bearer ' + localStorage.getItem('token') }
      })
    }
    ElMessage.success('状态更新成功')
    batchStatusVisible.value = false
    fetchData()
  } catch (e) {
    ElMessage.error('状态更新失败')
  } finally {
    batchStatusLoading.value = false
  }
}

const handleTest = async (row) => {
  testVisible.value = true
  testLoading.value = true
  testResult.value = null
  testError.value = ''
  try {
    const res = await axios.post(`/api/v1/cmdb/hosts/${row.id}/test`, {}, {
      headers: { Authorization: 'Bearer ' + localStorage.getItem('token') }
    })
    if (res.data.code === 0) {
      testResult.value = res.data.data
    } else {
      testError.value = res.data.message || '测试失败'
    }
  } catch (e) {
    testError.value = '测试失败'
  } finally {
    testLoading.value = false
  }
}

onMounted(() => {
  fetchGroups()
  fetchData()
})
</script>

<style scoped>
.flex { display: flex; }
.justify-between { justify-content: space-between; }
.items-center { align-items: center; }
.gap-2 { gap: 8px; }
.font-bold { font-weight: bold; }
.mb-4 { margin-bottom: 16px; }
.w-64 { width: 256px; }
.w-100 { width: 100%; }
.import-actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 12px; }
.test-block { display: flex; flex-direction: column; gap: 10px; }
.test-title { font-weight: 600; }
.test-pre { background: #0f172a; color: #e2e8f0; padding: 12px; border-radius: 6px; overflow: auto; max-height: 200px; }
</style>
