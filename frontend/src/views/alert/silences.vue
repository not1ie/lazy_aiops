<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>告警静默</h2>
        <p class="page-desc">在指定时间窗口内静默匹配的告警。</p>
      </div>
      <div class="page-actions">
        <el-button type="primary" icon="Plus" @click="openCreate">新增静默</el-button>
        <el-button icon="Refresh" @click="fetchSilences">刷新</el-button>
      </div>
    </div>

    <el-table :data="silences" stripe style="width: 100%">
      <el-table-column prop="name" label="名称" min-width="160" />
      <el-table-column prop="matchers" label="匹配规则" min-width="240" />
      <el-table-column prop="starts_at" label="开始时间" min-width="180" />
      <el-table-column prop="ends_at" label="结束时间" min-width="180" />
      <el-table-column prop="status" label="状态" width="120">
        <template #default="scope">
          <el-tag :type="scope.row.status === 1 ? 'success' : 'info'">
            {{ scope.row.status === 1 ? '生效中' : '过期' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="200">
        <template #default="scope">
          <el-button size="small" @click="openEdit(scope.row)">编辑</el-button>
          <el-button size="small" type="danger" @click="removeSilence(scope.row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog append-to-body v-model="dialogVisible" :title="dialogTitle" width="720px">
      <el-form :model="form" label-width="110px">
        <el-form-item label="名称">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="匹配规则">
          <el-input v-model="form.matchers" type="textarea" :rows="3" placeholder='例如: {"severity":"critical","target":"db"}' />
        </el-form-item>
        <el-form-item label="开始时间">
          <el-date-picker v-model="form.starts_at" type="datetime" />
        </el-form-item>
        <el-form-item label="结束时间">
          <el-date-picker v-model="form.ends_at" type="datetime" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="form.comment" type="textarea" :rows="2" />
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

const silences = ref([])
const dialogVisible = ref(false)
const dialogTitle = ref('')
const isEdit = ref(false)
const currentId = ref('')
const form = ref({
  name: '',
  matchers: '',
  starts_at: '',
  ends_at: '',
  comment: ''
})

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const fetchSilences = async () => {
  const res = await axios.get('/api/v1/alert/silences', { headers: authHeaders() })
  silences.value = res.data.data || []
}

const openCreate = () => {
  isEdit.value = false
  dialogTitle.value = '新增静默'
  form.value = { name: '', matchers: '', starts_at: '', ends_at: '', comment: '' }
  dialogVisible.value = true
}

const openEdit = (row) => {
  isEdit.value = true
  dialogTitle.value = '编辑静默'
  currentId.value = row.id
  form.value = { name: row.name, matchers: row.matchers, starts_at: row.starts_at, ends_at: row.ends_at, comment: row.comment }
  dialogVisible.value = true
}

const submitForm = async () => {
  if (isEdit.value) {
    await axios.put(`/api/v1/alert/silences/${currentId.value}`, form.value, { headers: authHeaders() })
    ElMessage.success('更新成功')
  } else {
    await axios.post('/api/v1/alert/silences', form.value, { headers: authHeaders() })
    ElMessage.success('创建成功')
  }
  dialogVisible.value = false
  fetchSilences()
}

const removeSilence = async (row) => {
  await ElMessageBox.confirm(`确认删除静默 ${row.name} 吗？`, '提示', { type: 'warning' })
  await axios.delete(`/api/v1/alert/silences/${row.id}`, { headers: authHeaders() })
  ElMessage.success('删除成功')
  fetchSilences()
}

onMounted(fetchSilences)
</script>

<style scoped>
.page-card { max-width: 1180px; margin: 0 auto; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 16px; }
.page-desc { color: #606266; margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; }
</style>
