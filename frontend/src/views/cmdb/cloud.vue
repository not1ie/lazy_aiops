<template>
  <el-card class="page-card">
    <template #header>
      <div class="header">
        <div>
          <div class="title">云资源管理</div>
          <div class="desc">云账号与资源清单维护</div>
        </div>
        <div class="actions">
          <el-button type="primary" icon="Plus" @click="openCreate(activeTab)">新增</el-button>
          <el-button icon="Upload" @click="openImport">批量导入</el-button>
          <el-button icon="Download" @click="exportCSV">导出</el-button>
          <el-button type="danger" plain icon="Delete" :disabled="activeSelectedCount === 0" @click="handleBatchDelete">
            批量删除 ({{ activeSelectedCount }})
          </el-button>
          <el-button icon="Refresh" @click="refreshActive">刷新</el-button>
        </div>
      </div>
    </template>

    <el-tabs v-model="activeTab" class="cloud-tabs">
      <el-tab-pane label="云账号" name="accounts">
        <el-table :fit="false" :data="accounts" v-loading="loadingAccounts" stripe @selection-change="selectedAccounts = $event">
          <el-table-column type="selection" width="48" />
          <el-table-column prop="name" label="账号名称" min-width="180" />
          <el-table-column prop="provider" label="云厂商" width="120" />
          <el-table-column prop="region" label="区域" width="140" />
          <el-table-column prop="status" label="状态" width="100">
            <template #default="{ row }">
              <el-tag :type="row.status === 1 ? 'success' : 'danger'">{{ row.status === 1 ? '启用' : '禁用' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
          <el-table-column label="操作" width="240" fixed="right">
            <template #default="{ row }">
              <el-space size="8">
                <el-button size="small" type="warning" plain icon="FirstAidKit" @click="openTest(row)">测试</el-button>
                <el-button size="small" type="primary" plain @click="openEdit('accounts', row)">编辑</el-button>
                <el-button size="small" type="danger" plain @click="handleDelete('accounts', row)">删除</el-button>
              </el-space>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="云资源" name="resources">
        <div class="filters">
          <el-select v-model="resourceFilters.account_id" placeholder="账号" clearable @change="fetchResources">
            <el-option v-for="acc in accounts" :key="acc.id" :label="acc.name" :value="acc.id" />
          </el-select>
          <el-input v-model="resourceFilters.keyword" placeholder="名称/ID/IP" clearable @clear="fetchResources" @keyup.enter="fetchResources">
            <template #append>
              <el-button icon="Search" @click="fetchResources" />
            </template>
          </el-input>
        </div>
        <el-table :fit="false" :data="resources" v-loading="loadingResources" stripe @selection-change="selectedResources = $event">
          <el-table-column type="selection" width="48" />
          <el-table-column prop="name" label="资源名称" min-width="180" />
          <el-table-column prop="resource_id" label="资源ID" min-width="180" />
          <el-table-column prop="type" label="类型" width="120" />
          <el-table-column prop="region" label="区域" width="120" />
          <el-table-column prop="ip" label="IP" width="140" />
          <el-table-column prop="status" label="状态" width="120" />
          <el-table-column prop="account.name" label="账号" min-width="140" />
          <el-table-column label="操作" width="200" fixed="right">
            <template #default="{ row }">
              <el-button size="small" type="primary" plain @click="openEdit('resources', row)">编辑</el-button>
              <el-button size="small" type="danger" plain @click="handleDelete('resources', row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
    </el-tabs>
  </el-card>

  <el-dialog append-to-body v-model="dialogVisible" :title="dialogTitle" width="600px">
    <el-form v-if="activeDialog === 'accounts'" :model="accountForm" label-width="110px">
      <el-form-item label="账号名称" required>
        <el-input v-model="accountForm.name" />
      </el-form-item>
      <el-form-item label="云厂商" required>
        <el-select v-model="accountForm.provider" style="width: 100%">
          <el-option label="腾讯云" value="tencent" />
          <el-option label="百度云" value="baidu" />
          <el-option label="阿里云" value="aliyun" />
          <el-option label="华为云" value="huawei" />
          <el-option label="AWS" value="aws" />
        </el-select>
      </el-form-item>
      <el-form-item label="AccessKey">
        <el-input v-model="accountForm.access_key" />
      </el-form-item>
      <el-form-item label="SecretKey">
        <el-input v-model="accountForm.secret_key" type="password" show-password />
      </el-form-item>
      <el-form-item label="区域">
        <el-input v-model="accountForm.region" placeholder="如：ap-guangzhou" />
      </el-form-item>
      <el-form-item label="状态">
        <el-switch v-model="accountForm.status" :active-value="1" :inactive-value="0" />
      </el-form-item>
      <el-form-item label="描述">
        <el-input v-model="accountForm.description" type="textarea" :rows="3" />
      </el-form-item>
    </el-form>

    <el-form v-else :model="resourceForm" label-width="110px">
      <el-form-item label="所属账号" required>
        <el-select v-model="resourceForm.account_id" style="width: 100%">
          <el-option v-for="acc in accounts" :key="acc.id" :label="acc.name" :value="acc.id" />
        </el-select>
      </el-form-item>
      <el-form-item label="资源ID" required>
        <el-input v-model="resourceForm.resource_id" />
      </el-form-item>
      <el-form-item label="资源名称" required>
        <el-input v-model="resourceForm.name" />
      </el-form-item>
      <el-form-item label="类型">
        <el-select v-model="resourceForm.type" style="width: 100%">
          <el-option label="ECS" value="ecs" />
          <el-option label="RDS" value="rds" />
          <el-option label="SLB" value="slb" />
          <el-option label="VPC" value="vpc" />
        </el-select>
      </el-form-item>
      <el-form-item label="区域">
        <el-input v-model="resourceForm.region" />
      </el-form-item>
      <el-form-item label="可用区">
        <el-input v-model="resourceForm.zone" />
      </el-form-item>
      <el-form-item label="IP">
        <el-input v-model="resourceForm.ip" />
      </el-form-item>
      <el-form-item label="状态">
        <el-input v-model="resourceForm.status" />
      </el-form-item>
      <el-form-item label="规格">
        <el-input v-model="resourceForm.spec" />
      </el-form-item>
      <el-form-item label="标签">
        <el-input v-model="resourceForm.tags" placeholder="逗号分隔" />
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="dialogVisible = false">取消</el-button>
      <el-button type="primary" :loading="saving" @click="saveDialog">保存</el-button>
    </template>
  </el-dialog>

  <el-dialog append-to-body v-model="importVisible" title="批量导入" width="720px">
    <el-alert type="info" :closable="false" show-icon>
      账号格式：name,provider,access_key,secret_key,region,status,description
      资源格式：account_id,resource_id,name,type,region,zone,ip,status,spec,tags
    </el-alert>
    <el-input v-model="importText" type="textarea" :rows="10" />
    <div class="import-actions">
      <el-button @click="importVisible = false">取消</el-button>
      <el-button type="primary" :loading="importLoading" @click="submitImport">开始导入</el-button>
    </div>
  </el-dialog>

  <el-dialog append-to-body v-model="testVisible" title="云账号测试" width="560px">
    <el-alert v-if="testError" type="error" :closable="false" show-icon>{{ testError }}</el-alert>
    <el-alert v-if="testSuccess" type="success" :closable="false" show-icon>{{ testSuccess }}</el-alert>
    <div class="test-tip">仅校验 AccessKey/SecretKey 是否填写，不调用真实云 API。</div>
    <template #footer>
      <el-button @click="testVisible = false">关闭</el-button>
      <el-button type="primary" :loading="testLoading" @click="submitTest">测试</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, computed, onMounted, watch } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const activeTab = ref('accounts')
const activeDialog = ref('accounts')
const dialogVisible = ref(false)
const dialogTitle = computed(() => (activeDialog.value === 'accounts' ? (isEdit.value ? '编辑云账号' : '新增云账号') : (isEdit.value ? '编辑云资源' : '新增云资源')))
const isEdit = ref(false)
const currentId = ref('')
const saving = ref(false)

const accounts = ref([])
const resources = ref([])
const loadingAccounts = ref(false)
const loadingResources = ref(false)
const selectedAccounts = ref([])
const selectedResources = ref([])

const importVisible = ref(false)
const importLoading = ref(false)
const importText = ref('')

const testVisible = ref(false)
const testLoading = ref(false)
const testRow = ref(null)
const testError = ref('')
const testSuccess = ref('')

const activeSelectedCount = computed(() => (
  activeTab.value === 'accounts' ? selectedAccounts.value.length : selectedResources.value.length
))

const resourceFilters = reactive({
  account_id: '',
  keyword: ''
})

const accountForm = reactive({
  name: '',
  provider: 'tencent',
  access_key: '',
  secret_key: '',
  region: '',
  status: 1,
  description: ''
})

const resourceForm = reactive({
  account_id: '',
  resource_id: '',
  name: '',
  type: 'ecs',
  region: '',
  zone: '',
  ip: '',
  status: '',
  spec: '',
  tags: ''
})

const headers = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const fetchAccounts = async () => {
  loadingAccounts.value = true
  try {
    const res = await axios.get('/api/v1/cmdb/cloud/accounts', { headers: headers() })
    if (res.data.code === 0) {
      accounts.value = res.data.data
    }
  } catch (error) {
    ElMessage.error('加载云账号失败')
  } finally {
    loadingAccounts.value = false
  }
}

const fetchResources = async () => {
  loadingResources.value = true
  try {
    const res = await axios.get('/api/v1/cmdb/cloud/resources', {
      headers: headers(),
      params: { account_id: resourceFilters.account_id, keyword: resourceFilters.keyword }
    })
    if (res.data.code === 0) {
      resources.value = res.data.data
    }
  } catch (error) {
    ElMessage.error('加载云资源失败')
  } finally {
    loadingResources.value = false
  }
}

const refreshActive = () => {
  if (activeTab.value === 'accounts') {
    fetchAccounts()
  } else {
    fetchResources()
  }
}

const openCreate = (tab) => {
  isEdit.value = false
  currentId.value = ''
  activeDialog.value = tab
  if (tab === 'accounts') {
    Object.assign(accountForm, {
      name: '',
      provider: 'tencent',
      access_key: '',
      secret_key: '',
      region: '',
      status: 1,
      description: ''
    })
  } else {
    Object.assign(resourceForm, {
      account_id: resourceFilters.account_id || '',
      resource_id: '',
      name: '',
      type: 'ecs',
      region: '',
      zone: '',
      ip: '',
      status: '',
      spec: '',
      tags: ''
    })
  }
  dialogVisible.value = true
}

const openEdit = (tab, row) => {
  isEdit.value = true
  currentId.value = row.id
  activeDialog.value = tab
  if (tab === 'accounts') {
    Object.assign(accountForm, row)
  } else {
    Object.assign(resourceForm, row)
  }
  dialogVisible.value = true
}

const saveDialog = async () => {
  saving.value = true
  try {
    if (activeDialog.value === 'accounts') {
      const url = isEdit.value ? `/api/v1/cmdb/cloud/accounts/${currentId.value}` : '/api/v1/cmdb/cloud/accounts'
      const method = isEdit.value ? 'put' : 'post'
      const res = await axios({ url, method, data: accountForm, headers: headers() })
      if (res.data.code === 0) {
        ElMessage.success(isEdit.value ? '更新成功' : '创建成功')
        dialogVisible.value = false
        fetchAccounts()
      }
    } else {
      const url = isEdit.value ? `/api/v1/cmdb/cloud/resources/${currentId.value}` : '/api/v1/cmdb/cloud/resources'
      const method = isEdit.value ? 'put' : 'post'
      const res = await axios({ url, method, data: resourceForm, headers: headers() })
      if (res.data.code === 0) {
        ElMessage.success(isEdit.value ? '更新成功' : '创建成功')
        dialogVisible.value = false
        fetchResources()
      }
    }
  } catch (error) {
    ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

const handleDelete = (tab, row) => {
  const title = tab === 'accounts' ? `确定删除云账号“${row.name}”吗？` : `确定删除云资源“${row.name}”吗？`
  ElMessageBox.confirm(title, '提示', { type: 'warning' }).then(async () => {
    const url = tab === 'accounts' ? `/api/v1/cmdb/cloud/accounts/${row.id}` : `/api/v1/cmdb/cloud/resources/${row.id}`
    await axios.delete(url, { headers: headers() })
    ElMessage.success('删除成功')
    refreshActive()
  })
}

const handleBatchDelete = () => {
  const rows = activeTab.value === 'accounts' ? selectedAccounts.value : selectedResources.value
  if (rows.length === 0) return
  const title = activeTab.value === 'accounts'
    ? `确定删除选中的 ${rows.length} 个云账号吗？`
    : `确定删除选中的 ${rows.length} 个云资源吗？`
  ElMessageBox.confirm(title, '提示', { type: 'warning' }).then(async () => {
    for (const row of rows) {
      const url = activeTab.value === 'accounts'
        ? `/api/v1/cmdb/cloud/accounts/${row.id}`
        : `/api/v1/cmdb/cloud/resources/${row.id}`
      await axios.delete(url, { headers: headers() })
    }
    ElMessage.success('批量删除成功')
    if (activeTab.value === 'accounts') selectedAccounts.value = []
    else selectedResources.value = []
    refreshActive()
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
  const hasHeader = headers.includes('name') || headers.includes('resource_id')
  const start = hasHeader ? 1 : 0
  const cols = hasHeader ? headers : (activeTab.value === 'accounts'
    ? ['name','provider','access_key','secret_key','region','status','description']
    : ['account_id','resource_id','name','type','region','zone','ip','status','spec','tags'])
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
      const url = activeTab.value === 'accounts' ? '/api/v1/cmdb/cloud/accounts' : '/api/v1/cmdb/cloud/resources'
      await axios.post(url, row, { headers: headers() })
    }
    ElMessage.success('导入完成')
    importVisible.value = false
    refreshActive()
  } catch (e) {
    ElMessage.error('导入失败')
  } finally {
    importLoading.value = false
  }
}

const exportCSV = () => {
  const headersArr = activeTab.value === 'accounts'
    ? ['name','provider','region','status','description']
    : ['account_id','resource_id','name','type','region','zone','ip','status']
  const rows = (activeTab.value === 'accounts' ? accounts.value : resources.value)
    .map(r => headersArr.map(k => r[k] || ''))
  const csv = [headersArr.join(','), ...rows.map(r => r.map(v => `"${String(v ?? '').replace(/"/g, '""')}"`).join(','))].join('\n')
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = activeTab.value === 'accounts' ? 'cmdb_cloud_accounts.csv' : 'cmdb_cloud_resources.csv'
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
    const res = await axios.post(`/api/v1/cmdb/cloud/accounts/${testRow.value.id}/test`, {}, { headers: headers() })
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

watch(activeTab, (val) => {
  if (val === 'accounts') {
    fetchAccounts()
  } else {
    fetchResources()
  }
})

onMounted(async () => {
  await fetchAccounts()
  await fetchResources()
})
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.header { display: flex; justify-content: space-between; align-items: center; }
.title { font-size: 18px; font-weight: 600; }
.desc { color: #909399; margin-top: 4px; }
.actions { display: flex; gap: 8px; align-items: center; }
.cloud-tabs { margin-top: 8px; }
.filters { display: flex; gap: 12px; margin-bottom: 16px; }
.filters .el-select { width: 200px; }
.filters .el-input { width: 260px; }
.import-actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 12px; }
.test-tip { color: #909399; margin-top: 8px; }
</style>
