<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>通知组管理</h2>
        <p class="page-desc">聚合多个通知渠道并用于告警规则。</p>
      </div>
      <div class="page-actions">
        <el-button type="primary" icon="Plus" @click="openCreate">新增通知组</el-button>
        <el-button icon="Refresh" @click="fetchGroups">刷新</el-button>
      </div>
    </div>

    <el-table :data="groups" stripe style="width: 100%">
      <el-table-column prop="name" label="名称" min-width="160" />
      <el-table-column prop="description" label="描述" min-width="240" />
      <el-table-column prop="channels" label="渠道数" width="100">
        <template #default="scope">
          {{ parseList(scope.row.channels).length }}
        </template>
      </el-table-column>
      <el-table-column label="操作" width="200">
        <template #default="scope">
          <el-button size="small" @click="openEdit(scope.row)">编辑</el-button>
          <el-button size="small" type="danger" @click="removeGroup(scope.row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog append-to-body v-model="dialogVisible" :title="dialogTitle" width="720px">
      <el-form :model="form" label-width="110px">
        <el-form-item label="名称">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="2" />
        </el-form-item>
        <el-form-item label="通知渠道">
          <el-select v-model="form.channels" multiple filterable class="w-52">
            <el-option v-for="c in channels" :key="c.id" :label="c.name" :value="c.id" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitForm">保存</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const groups = ref([])
const channels = ref([])
const dialogVisible = ref(false)
const dialogTitle = ref('')
const isEdit = ref(false)
const currentId = ref('')
const form = ref({ name: '', description: '', channels: [] })

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const parseList = (txt) => {
  if (!txt) return []
  try {
    return JSON.parse(txt)
  } catch {
    return []
  }
}

const fetchGroups = async () => {
  const res = await axios.get('/api/v1/notify/groups', { headers: authHeaders() })
  groups.value = res.data.data || []
}

const fetchChannels = async () => {
  const res = await axios.get('/api/v1/notify/channels', { headers: authHeaders() })
  channels.value = res.data.data || []
}

const openCreate = () => {
  isEdit.value = false
  dialogTitle.value = '新增通知组'
  form.value = { name: '', description: '', channels: [] }
  dialogVisible.value = true
}

const openEdit = (row) => {
  isEdit.value = true
  dialogTitle.value = '编辑通知组'
  currentId.value = row.id
  form.value = { name: row.name, description: row.description, channels: parseList(row.channels) }
  dialogVisible.value = true
}

const submitForm = async () => {
  const payload = { ...form.value, channels: JSON.stringify(form.value.channels || []) }
  if (isEdit.value) {
    await axios.put(`/api/v1/notify/groups/${currentId.value}`, payload, { headers: authHeaders() })
    ElMessage.success('更新成功')
  } else {
    await axios.post('/api/v1/notify/groups', payload, { headers: authHeaders() })
    ElMessage.success('创建成功')
  }
  dialogVisible.value = false
  fetchGroups()
}

const removeGroup = async (row) => {
  await ElMessageBox.confirm(`确认删除通知组 ${row.name} 吗？`, '提示', { type: 'warning' })
  await axios.delete(`/api/v1/notify/groups/${row.id}`, { headers: authHeaders() })
  ElMessage.success('删除成功')
  fetchGroups()
}

onMounted(() => {
  fetchGroups()
  fetchChannels()
})
</script>

<style scoped>
.page-card { max-width: 1180px; margin: 0 auto; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 16px; }
.page-desc { color: #606266; margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; }
.w-52 { width: 220px; }
</style>
