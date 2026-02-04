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
          <el-button icon="Refresh" @click="fetchData">刷新</el-button>
        </div>
      </div>
    </template>

    <el-table :data="credentials" v-loading="loading" stripe>
      <el-table-column prop="name" label="名称" min-width="180" />
      <el-table-column prop="type" label="类型" width="120">
        <template #default="{ row }">
          <el-tag>{{ typeLabel(row.type) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="username" label="用户名" min-width="140" />
      <el-table-column prop="access_key" label="AccessKey" min-width="160" />
      <el-table-column prop="notes" label="备注" min-width="200" show-overflow-tooltip />
      <el-table-column label="操作" width="200" fixed="right">
        <template #default="{ row }">
          <el-button size="small" type="primary" plain @click="openEdit(row)">编辑</el-button>
          <el-button size="small" type="danger" plain @click="handleDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-card>

  <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑凭据' : '新增凭据'" width="560px">
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
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const loading = ref(false)
const saving = ref(false)
const credentials = ref([])
const dialogVisible = ref(false)
const isEdit = ref(false)
const currentId = ref('')

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

onMounted(fetchData)
</script>

<style scoped>
.page-card { max-width: 1100px; margin: 0 auto; }
.header { display: flex; justify-content: space-between; align-items: center; }
.title { font-size: 18px; font-weight: 600; }
.desc { color: #909399; margin-top: 4px; }
.actions { display: flex; gap: 8px; }
</style>
