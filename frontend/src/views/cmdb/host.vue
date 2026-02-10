<template>
  <el-card>
    <template #header>
      <div class="flex justify-between items-center">
        <span class="font-bold">CMDB 主机管理</span>
        <div>
          <el-button type="primary" icon="Plus" @click="handleAdd">添加主机</el-button>
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

    <el-table :data="tableData" v-loading="loading" style="width: 100%" @selection-change="selectedRows = $event">
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
          <el-tag :type="row.status === 1 ? 'success' : 'danger'">
            {{ row.status === 1 ? '在线' : '离线' }}
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
    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑主机' : '添加主机'" width="500px">
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

    <el-dialog v-model="testVisible" title="主机测试" width="640px">
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

const form = reactive({
  name: '',
  ip: '',
  port: 22,
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
.test-block { display: flex; flex-direction: column; gap: 10px; }
.test-title { font-weight: 600; }
.test-pre { background: #0f172a; color: #e2e8f0; padding: 12px; border-radius: 6px; overflow: auto; max-height: 200px; }
</style>
