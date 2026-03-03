<template>
  <el-card class="page-card">
    <template #header>
      <div class="header">
        <div>
          <div class="title">主机分组</div>
          <div class="desc">维护主机分组与归属关系</div>
        </div>
        <div class="actions">
          <el-button type="primary" icon="Plus" @click="openCreate">新增分组</el-button>
          <el-button icon="Refresh" @click="fetchData">刷新</el-button>
        </div>
      </div>
    </template>

    <el-table :data="groups" v-loading="loading" stripe>
      <el-table-column prop="name" label="分组名称" min-width="200" />
      <el-table-column prop="description" label="描述" min-width="240" />
      <el-table-column prop="parent_id" label="父级ID" min-width="200" />
      <el-table-column label="操作" width="200" fixed="right">
        <template #default="{ row }">
          <el-button size="small" type="primary" plain @click="openEdit(row)">编辑</el-button>
          <el-button size="small" type="danger" plain @click="handleDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-card>

  <el-dialog append-to-body v-model="dialogVisible" :title="isEdit ? '编辑分组' : '新增分组'" width="480px">
    <el-form :model="form" label-width="90px">
      <el-form-item label="分组名称" required>
        <el-input v-model="form.name" placeholder="如：生产环境" />
      </el-form-item>
      <el-form-item label="描述">
        <el-input v-model="form.description" type="textarea" :rows="3" />
      </el-form-item>
      <el-form-item label="父级ID">
        <el-input v-model="form.parent_id" placeholder="可选" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="dialogVisible = false">取消</el-button>
      <el-button type="primary" :loading="saving" @click="saveGroup">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const loading = ref(false)
const saving = ref(false)
const groups = ref([])
const dialogVisible = ref(false)
const isEdit = ref(false)
const currentId = ref('')

const form = reactive({
  name: '',
  description: '',
  parent_id: ''
})

const fetchData = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/v1/cmdb/groups', {
      headers: { Authorization: `Bearer ${localStorage.getItem('token')}` }
    })
    if (res.data.code === 0) {
      groups.value = res.data.data
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
  form.name = ''
  form.description = ''
  form.parent_id = ''
  dialogVisible.value = true
}

const openEdit = (row) => {
  isEdit.value = true
  currentId.value = row.id
  form.name = row.name
  form.description = row.description
  form.parent_id = row.parent_id
  dialogVisible.value = true
}

const saveGroup = async () => {
  if (!form.name) {
    ElMessage.warning('请填写分组名称')
    return
  }
  saving.value = true
  try {
    const url = isEdit.value ? `/api/v1/cmdb/groups/${currentId.value}` : '/api/v1/cmdb/groups'
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
  ElMessageBox.confirm(`确定删除分组“${row.name}”吗？`, '提示', {
    type: 'warning'
  }).then(async () => {
    await axios.delete(`/api/v1/cmdb/groups/${row.id}`, {
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
