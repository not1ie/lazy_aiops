<template>
  <el-card class="page-card">
    <template #header>
      <div class="header">
        <div>
          <div class="title">Ansible Inventory</div>
          <div class="desc">维护主机清单与分组</div>
        </div>
        <div class="actions">
          <el-button type="primary" icon="Plus" @click="openCreate">新增</el-button>
          <el-button icon="Refresh" @click="fetchData">刷新</el-button>
          <el-button type="success" plain @click="syncFromCMDB">从CMDB同步</el-button>
        </div>
      </div>
    </template>

    <el-table :data="items" v-loading="loading" stripe>
      <el-table-column prop="name" label="名称" min-width="200" />
      <el-table-column prop="description" label="描述" min-width="240" />
      <el-table-column prop="source" label="来源" width="120" />
      <el-table-column label="操作" width="200" fixed="right">
        <template #default="{ row }">
          <el-button size="small" type="primary" plain @click="openEdit(row)">编辑</el-button>
          <el-button size="small" type="danger" plain @click="handleDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-card>

  <el-dialog append-to-body v-model="dialogVisible" :title="isEdit ? '编辑Inventory' : '新增Inventory'" width="700px">
    <el-form :model="form" label-width="90px">
      <el-form-item label="名称" required>
        <el-input v-model="form.name" />
      </el-form-item>
      <el-form-item label="描述">
        <el-input v-model="form.description" />
      </el-form-item>
      <el-form-item label="内容" required>
        <el-input v-model="form.content" type="textarea" :rows="10" placeholder="[web]\n10.0.0.1" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="dialogVisible = false">取消</el-button>
      <el-button type="primary" :loading="saving" @click="saveItem">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const items = ref([])
const loading = ref(false)
const saving = ref(false)
const dialogVisible = ref(false)
const isEdit = ref(false)
const currentId = ref('')

const form = reactive({
  name: '',
  description: '',
  content: ''
})

const headers = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const fetchData = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/v1/ansible/inventories', { headers: headers() })
    if (res.data.code === 0) {
      items.value = res.data.data
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
  Object.assign(form, { name: '', description: '', content: '' })
  dialogVisible.value = true
}

const openEdit = (row) => {
  isEdit.value = true
  currentId.value = row.id
  Object.assign(form, { name: row.name, description: row.description, content: row.content || '' })
  dialogVisible.value = true
}

const saveItem = async () => {
  if (!form.name || !form.content) {
    ElMessage.warning('请填写名称和内容')
    return
  }
  saving.value = true
  try {
    const url = isEdit.value ? `/api/v1/ansible/inventories/${currentId.value}` : '/api/v1/ansible/inventories'
    const method = isEdit.value ? 'put' : 'post'
    const res = await axios({ url, method, data: form, headers: headers() })
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
  ElMessageBox.confirm(`确定删除“${row.name}”吗？`, '提示', { type: 'warning' }).then(async () => {
    await axios.delete(`/api/v1/ansible/inventories/${row.id}`, { headers: headers() })
    ElMessage.success('删除成功')
    fetchData()
  })
}

const syncFromCMDB = async () => {
  try {
    const defaultName = `cmdb-sync-${new Date().toISOString().slice(0, 19).replace(/[-:T]/g, '')}`
    const res = await axios.post('/api/v1/ansible/inventories/sync-cmdb', { name: defaultName }, { headers: headers() })
    if (res.data.code === 0) {
      ElMessage.success('同步成功')
      fetchData()
      return
    }
    ElMessage.error(res.data.message || '同步失败')
  } catch (error) {
    ElMessage.error(error?.response?.data?.message || '同步失败')
  }
}

onMounted(fetchData)
</script>

<style scoped>
.page-card { max-width: 1200px; margin: 0 auto; }
.header { display: flex; justify-content: space-between; align-items: center; }
.title { font-size: 18px; font-weight: 600; }
.desc { color: #909399; margin-top: 4px; }
.actions { display: flex; gap: 8px; }
</style>
