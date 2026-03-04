<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>升级策略</h2>
        <p class="page-desc">按告警级别和时间窗口定义通知升级流程。</p>
      </div>
      <div class="page-actions">
        <el-select v-model="scheduleFilter" clearable placeholder="按排班过滤" style="width: 220px" @change="fetchEscalations">
          <el-option v-for="item in schedules" :key="item.id" :label="item.name" :value="item.id" />
        </el-select>
        <el-button type="primary" icon="Plus" @click="openDialog()">新增策略</el-button>
        <el-button icon="Refresh" @click="fetchEscalations">刷新</el-button>
      </div>
    </div>

    <el-row :gutter="12" class="mb-12">
      <el-col :span="8"><el-card><div class="k">策略总数</div><div class="v">{{ escalations.length }}</div></el-card></el-col>
      <el-col :span="8"><el-card><div class="k">启用策略</div><div class="v">{{ enabledCount }}</div></el-card></el-col>
      <el-col :span="8"><el-card><div class="k">关联排班</div><div class="v">{{ linkedSchedules }}</div></el-card></el-col>
    </el-row>

    <el-table :fit="true" :data="escalations" v-loading="loading" stripe>
      <el-table-column prop="name" label="策略名称" min-width="180" />
      <el-table-column label="排班" min-width="160">
        <template #default="{ row }">{{ scheduleName(row.schedule_id) }}</template>
      </el-table-column>
      <el-table-column prop="rules" label="规则" min-width="360" show-overflow-tooltip />
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? '启用' : '停用' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="description" label="说明" min-width="180" show-overflow-tooltip />
      <el-table-column label="操作" width="190" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="openDialog(row)">编辑</el-button>
          <el-button size="small" type="danger" plain @click="removeItem(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog append-to-body v-model="dialogVisible" :title="editing ? '编辑升级策略' : '新增升级策略'" width="700px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="策略名称" required>
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="关联排班">
          <el-select v-model="form.schedule_id" clearable style="width: 100%">
            <el-option v-for="item in schedules" :key="item.id" :label="item.name" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="升级规则" required>
          <el-input
            v-model="form.rules"
            type="textarea"
            :rows="8"
            placeholder='JSON数组，例如 [{"level":"critical","delay_min":5,"notify":["sms","phone"]}]'
          />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
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
  </el-card>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const loading = ref(false)
const saving = ref(false)
const editing = ref(false)
const dialogVisible = ref(false)
const currentId = ref('')

const schedules = ref([])
const escalations = ref([])
const scheduleFilter = ref('')

const form = reactive({
  name: '',
  schedule_id: '',
  rules: '[{"level":"critical","delay_min":5,"notify":["sms"]}]',
  enabled: true,
  description: ''
})

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const enabledCount = computed(() => escalations.value.filter(item => item.enabled).length)

const linkedSchedules = computed(() => {
  const ids = new Set(escalations.value.filter(item => item.schedule_id).map(item => item.schedule_id))
  return ids.size
})

const resetForm = () => {
  currentId.value = ''
  form.name = ''
  form.schedule_id = ''
  form.rules = '[{"level":"critical","delay_min":5,"notify":["sms"]}]'
  form.enabled = true
  form.description = ''
}

const scheduleName = (id) => schedules.value.find(item => item.id === id)?.name || '-'

const fetchSchedules = async () => {
  try {
    const res = await axios.get('/api/v1/oncall/schedules', { headers: authHeaders() })
    if (res.data?.code === 0) schedules.value = res.data.data || []
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '加载排班失败')
  }
}

const fetchEscalations = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/v1/oncall/escalations', {
      headers: authHeaders(),
      params: { schedule_id: scheduleFilter.value || undefined }
    })
    if (res.data?.code === 0) escalations.value = res.data.data || []
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '加载策略失败')
  } finally {
    loading.value = false
  }
}

const openDialog = (row) => {
  editing.value = !!row
  resetForm()
  if (row) {
    currentId.value = row.id
    form.name = row.name || ''
    form.schedule_id = row.schedule_id || ''
    form.rules = row.rules || '[]'
    form.enabled = !!row.enabled
    form.description = row.description || ''
  }
  dialogVisible.value = true
}

const saveItem = async () => {
  if (!form.name.trim()) {
    ElMessage.warning('请输入策略名称')
    return
  }
  try {
    JSON.parse(form.rules || '[]')
  } catch {
    ElMessage.warning('升级规则必须是合法JSON')
    return
  }

  saving.value = true
  try {
    const payload = {
      name: form.name.trim(),
      schedule_id: form.schedule_id,
      rules: form.rules,
      enabled: form.enabled,
      description: form.description?.trim() || ''
    }

    if (editing.value && currentId.value) {
      await axios.put(`/api/v1/oncall/escalations/${currentId.value}`, payload, { headers: authHeaders() })
      ElMessage.success('更新成功')
    } else {
      await axios.post('/api/v1/oncall/escalations', payload, { headers: authHeaders() })
      ElMessage.success('创建成功')
    }

    dialogVisible.value = false
    await fetchEscalations()
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '保存失败')
  } finally {
    saving.value = false
  }
}

const removeItem = async (row) => {
  try {
    await ElMessageBox.confirm(`确认删除策略 ${row.name} ?`, '提示', { type: 'warning' })
    await axios.delete(`/api/v1/oncall/escalations/${row.id}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    await fetchEscalations()
  } catch (err) {
    if (err !== 'cancel') ElMessage.error(err.response?.data?.message || '删除失败')
  }
}

onMounted(async () => {
  await fetchSchedules()
  await fetchEscalations()
})
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.page-header { display: flex; justify-content: space-between; align-items: center; gap: 12px; margin-bottom: 12px; }
.page-desc { color: #909399; margin: 4px 0 0; }
.page-actions { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.k { color: #909399; font-size: 12px; }
.v { font-size: 26px; font-weight: 700; margin-top: 4px; }
.mb-12 { margin-bottom: 12px; }
</style>
