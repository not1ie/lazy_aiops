<template>
  <el-card class="page-card">
    <template #header>
      <div class="header">
        <div>
          <div class="title">凭据管理</div>
          <div class="desc">统一管理 SSH/数据库/API 等访问凭据</div>
        </div>
        <div class="actions">
          <el-button type="primary" icon="Plus" @click="openCreate">新增凭据</el-button>
          <el-button icon="Upload" @click="openImport">批量导入</el-button>
          <el-button icon="Download" @click="exportCSV">导出</el-button>
          <el-switch v-model="showSensitive" active-text="显示敏感" inactive-text="隐藏敏感" />
          <el-button type="danger" plain icon="Delete" :disabled="selectedRows.length === 0" @click="handleBatchDelete">
            批量删除 ({{ selectedRows.length }})
          </el-button>
          <el-button icon="Refresh" @click="fetchData">刷新</el-button>
        </div>
      </div>
    </template>

    <el-table :data="credentials" v-loading="loading" stripe @selection-change="selectedRows = $event">
      <el-table-column type="selection" width="48" />
      <el-table-column prop="name" label="名称" min-width="160" />
      <el-table-column prop="type" label="类型" width="120">
        <template #default="{ row }">
          <el-tag>{{ typeLabel(row.type) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="username" label="用户名" min-width="140" />
      <el-table-column prop="access_key" label="AccessKey" min-width="160">
        <template #default="{ row }">
          {{ maskValue(row.access_key) }}
        </template>
      </el-table-column>
      <el-table-column prop="secret_key" label="SecretKey" min-width="160">
        <template #default="{ row }">
          {{ maskValue(row.secret_key) }}
        </template>
      </el-table-column>
      <el-table-column prop="private_key" label="私钥" min-width="160">
        <template #default="{ row }">
          {{ maskValue(row.private_key) }}
        </template>
      </el-table-column>
      <el-table-column prop="notes" label="备注" min-width="180" show-overflow-tooltip />
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

  <el-dialog append-to-body v-model="dialogVisible" :title="isEdit ? '编辑凭据' : '新增凭据'" width="560px">
    <el-form :model="form" label-width="100px">
      <el-form-item label="名称" required>
        <el-input v-model="form.name" placeholder="如：生产SSH" />
      </el-form-item>
      <el-form-item label="类型" required>
        <el-select v-model="form.type" placeholder="请选择类型" style="width: 100%">
          <el-option label="密码" value="password" />
          <el-option label="SSH密钥" value="key" />
          <el-option label="API密钥" value="api" />
        </el-select>
      </el-form-item>
      <el-form-item label="用户名" v-if="form.type !== 'api'">
        <el-input v-model="form.username" placeholder="如：root" />
      </el-form-item>
      <el-form-item label="密码" v-if="form.type === 'password'">
        <el-input v-model="form.password" type="password" show-password />
      </el-form-item>
      <el-form-item label="私钥" v-if="form.type === 'key'">
        <el-input v-model="form.private_key" type="textarea" :rows="4" placeholder="-----BEGIN RSA PRIVATE KEY-----" />
      </el-form-item>
      <el-form-item label="口令" v-if="form.type === 'key'">
        <el-input v-model="form.passphrase" type="password" show-password />
      </el-form-item>
      <el-form-item label="AccessKey" v-if="form.type === 'api'">
        <el-input v-model="form.access_key" />
      </el-form-item>
      <el-form-item label="SecretKey" v-if="form.type === 'api'">
        <el-input v-model="form.secret_key" type="password" show-password />
      </el-form-item>
      <el-form-item label="备注">
        <el-input v-model="form.notes" type="textarea" :rows="3" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="dialogVisible = false">取消</el-button>
      <el-button type="primary" :loading="saving" @click="saveCredential">保存</el-button>
    </template>
  </el-dialog>

  <!-- 批量导入 -->
  <el-dialog append-to-body v-model="importVisible" title="批量导入凭据" width="720px">
    <el-alert type="info" :closable="false" show-icon>
      格式：name,type,username,password,private_key,passphrase,access_key,secret_key,notes（第一行可写表头）
    </el-alert>
    <el-input v-model="importText" type="textarea" :rows="10" placeholder="例如：&#10;prod-ssh,password,root,pass,,,'',,生产环境" />
    <div class="import-actions">
      <el-button @click="importVisible = false">取消</el-button>
      <el-button type="primary" :loading="importLoading" @click="submitImport">开始导入</el-button>
    </div>
  </el-dialog>

  <!-- 测试凭据 -->
  <el-dialog append-to-body v-model="testVisible" title="凭据测试" width="560px">
    <div v-if="testRow?.type === 'api'" class="test-tip">API 凭据仅校验字段是否完整，不做真实调用。</div>
    <el-form v-else label-width="80px">
      <el-form-item label="主机">
        <el-input v-model="testHost" placeholder="如 192.168.1.10" />
      </el-form-item>
      <el-form-item label="端口">
        <el-input-number v-model="testPort" :min="1" :max="65535" />
      </el-form-item>
    </el-form>
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
const credentials = ref([])
const selectedRows = ref([])
const dialogVisible = ref(false)
const isEdit = ref(false)
const currentId = ref('')
const showSensitive = ref(false)

const importVisible = ref(false)
const importLoading = ref(false)
const importText = ref('')

const testVisible = ref(false)
const testLoading = ref(false)
const testRow = ref(null)
const testHost = ref('')
const testPort = ref(22)
const testError = ref('')
const testSuccess = ref('')

const form = reactive({
  name: '',
  type: 'password',
  username: '',
  password: '',
  private_key: '',
  passphrase: '',
  access_key: '',
  secret_key: '',
  notes: ''
})

const typeLabel = (type) => {
  if (type === 'password') return '密码'
  if (type === 'key') return 'SSH密钥'
  if (type === 'api') return 'API密钥'
  return type || '-'
}

const maskValue = (val) => {
  if (!val) return ''
  if (showSensitive.value) return val
  const str = String(val)
  if (str.length <= 6) return '******'
  return str.slice(0, 2) + '****' + str.slice(-2)
}

const fetchData = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/v1/cmdb/credentials', {
      headers: { Authorization: `Bearer ${localStorage.getItem('token')}` }
    })
    if (res.data.code === 0) {
      credentials.value = res.data.data
    }
  } catch (error) {
    ElMessage.error('加载失败')
  } finally {
    loading.value = false
  }
}

const openCreate = () => {
  isEdit.value = false
  currentId.value = ''
  Object.assign(form, {
    name: '',
    type: 'password',
    username: '',
    password: '',
    private_key: '',
    passphrase: '',
    access_key: '',
    secret_key: '',
    notes: ''
  })
  dialogVisible.value = true
}

const openEdit = (row) => {
  isEdit.value = true
  currentId.value = row.id
  Object.assign(form, {
    name: row.name,
    type: row.type || 'password',
    username: row.username,
    password: row.password,
    private_key: row.private_key,
    passphrase: row.passphrase,
    access_key: row.access_key,
    secret_key: row.secret_key,
    notes: row.notes
  })
  dialogVisible.value = true
}

const saveCredential = async () => {
  if (!form.name) {
    ElMessage.warning('请填写名称')
    return
  }
  saving.value = true
  try {
    const url = isEdit.value ? `/api/v1/cmdb/credentials/${currentId.value}` : '/api/v1/cmdb/credentials'
    const method = isEdit.value ? 'put' : 'post'
    const res = await axios({
      url,
      method,
      data: form,
      headers: { Authorization: `Bearer ${localStorage.getItem('token')}` }
    })
    if (res.data.code === 0) {
      ElMessage.success(isEdit.value ? '更新成功' : '创建成功')
      dialogVisible.value = false
      fetchData()
    }
  } catch (error) {
    ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

const handleDelete = (row) => {
  ElMessageBox.confirm(`确定删除凭据“${row.name}”吗？`, '提示', {
    type: 'warning'
  }).then(async () => {
    await axios.delete(`/api/v1/cmdb/credentials/${row.id}`, {
      headers: { Authorization: `Bearer ${localStorage.getItem('token')}` }
    })
    ElMessage.success('删除成功')
    fetchData()
  })
}

const handleBatchDelete = () => {
  if (selectedRows.value.length === 0) return
  ElMessageBox.confirm(`确定删除选中的 ${selectedRows.value.length} 个凭据吗？`, '提示', {
    type: 'warning'
  }).then(async () => {
    for (const row of selectedRows.value) {
      await axios.delete(`/api/v1/cmdb/credentials/${row.id}`, {
        headers: { Authorization: `Bearer ${localStorage.getItem('token')}` }
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
  const hasHeader = headers.includes('name') || headers.includes('type')
  const start = hasHeader ? 1 : 0
  const cols = hasHeader ? headers : ['name','type','username','password','private_key','passphrase','access_key','secret_key','notes']
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
      if (!row.name) continue
      await axios.post('/api/v1/cmdb/credentials', row, {
        headers: { Authorization: `Bearer ${localStorage.getItem('token')}` }
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
  const headers = ['name','type','username','access_key','notes']
  const rows = credentials.value.map(c => [c.name, c.type, c.username, c.access_key, c.notes])
  const csv = [headers.join(','), ...rows.map(r => r.map(v => `"${String(v ?? '').replace(/"/g, '""')}"`).join(','))].join('\n')
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'cmdb_credentials.csv'
  a.click()
  URL.revokeObjectURL(url)
}

const openTest = (row) => {
  testRow.value = row
  testHost.value = ''
  testPort.value = 22
  testError.value = ''
  testSuccess.value = ''
  testVisible.value = true
}

const submitTest = async () => {
  if (!testRow.value) return
  testLoading.value = true
  testError.value = ''
  testSuccess.value = ''
  try {
    const payload = testRow.value.type === 'api' ? {} : { host: testHost.value, port: testPort.value }
    const res = await axios.post(`/api/v1/cmdb/credentials/${testRow.value.id}/test`, payload, {
      headers: { Authorization: `Bearer ${localStorage.getItem('token')}` }
    })
    if (res.data.code === 0) {
      testSuccess.value = res.data.message || '测试成功'
    } else {
      testError.value = res.data.message || '测试失败'
    }
  } catch (e) {
    testError.value = '测试失败'
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
.import-actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 12px; }
.test-tip { color: #909399; margin-bottom: 12px; }
</style>
