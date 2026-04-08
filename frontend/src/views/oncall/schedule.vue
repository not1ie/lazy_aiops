<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>值班排班</h2>
        <p class="page-desc">管理团队、排班规则和换班处理。</p>
      </div>
      <div class="page-actions">
        <el-button type="primary" icon="Plus" @click="openScheduleDialog()">新增排班</el-button>
        <el-button icon="Refresh" @click="refreshAll">刷新</el-button>
      </div>
    </div>

    <el-row :gutter="12" class="mb-12">
      <el-col :span="6"><el-card><div class="k">排班数</div><div class="v">{{ schedules.length }}</div></el-card></el-col>
      <el-col :span="6"><el-card><div class="k">团队数</div><div class="v">{{ teams.length }}</div></el-card></el-col>
      <el-col :span="6"><el-card><div class="k">当前值班</div><div class="v">{{ currentOncall.length }}</div></el-card></el-col>
      <el-col :span="6"><el-card><div class="k">当前排班</div><div class="v">{{ selectedSchedule?.name || '-' }}</div></el-card></el-col>
    </el-row>

    <el-row :gutter="12">
      <el-col :md="15" :sm="24">
        <el-card>
          <template #header>
            <div class="section-header">
              <span>排班列表</span>
              <el-button size="small" icon="Plus" @click="openScheduleDialog()">新增</el-button>
            </div>
          </template>
          <el-table :fit="true" :data="schedules" v-loading="loading" stripe @row-click="selectSchedule">
            <el-table-column prop="name" label="名称" min-width="160" />
            <el-table-column prop="team_name" label="团队" min-width="120" />
            <el-table-column prop="type" label="类型" width="100" />
            <el-table-column prop="timezone" label="时区" width="130" />
            <el-table-column label="状态" width="90">
              <template #default="{ row }">
                <StatusBadge v-bind="scheduleEnabledBadge(row)" />
              </template>
            </el-table-column>
            <el-table-column label="操作" width="250" fixed="right">
              <template #default="{ row }">
                <el-button size="small" @click.stop="openScheduleDialog(row)">编辑</el-button>
                <el-button size="small" type="primary" plain @click.stop="openGenerateDialog(row)">生成班次</el-button>
                <el-button size="small" type="danger" plain @click.stop="removeSchedule(row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>

      <el-col :md="9" :sm="24">
        <el-card>
          <template #header>
            <div class="section-header">
              <span>值班团队</span>
              <el-button size="small" icon="Plus" @click="openTeamDialog">新增</el-button>
            </div>
          </template>
          <el-table :fit="true" :data="teams" size="small" stripe>
            <el-table-column prop="name" label="团队" min-width="120" />
            <el-table-column prop="description" label="说明" min-width="150" show-overflow-tooltip />
          </el-table>

          <el-divider />
          <div class="current-panel">
            <div class="sub-title">当前值班人</div>
            <el-empty v-if="!currentOncall.length" description="暂无值班人" :image-size="80" />
            <el-tag v-for="item in currentOncall" :key="item.id" class="mr-6 mb-6" effect="plain">
              {{ item.username }} · {{ formatTime(item.end_at) }}
            </el-tag>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-card class="mt-12">
      <template #header>
        <div class="section-header">
          <span>班次明细</span>
          <span class="muted">{{ selectedSchedule ? `排班：${selectedSchedule.name}` : '请选择排班' }}</span>
        </div>
      </template>

      <el-table :fit="true" :data="shifts" v-loading="shiftLoading" stripe>
        <el-table-column prop="username" label="值班人" width="140" />
        <el-table-column prop="phone" label="电话" width="130" />
        <el-table-column prop="email" label="邮箱" min-width="180" />
        <el-table-column label="开始时间" width="180">
          <template #default="{ row }">{{ formatTime(row.start_at) }}</template>
        </el-table-column>
        <el-table-column label="结束时间" width="180">
          <template #default="{ row }">{{ formatTime(row.end_at) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <StatusBadge v-bind="shiftStatusBadge(row)" />
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="openSwapDialog(row)">换班</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog append-to-body v-model="scheduleDialogVisible" :title="scheduleEditing ? '编辑排班' : '新增排班'" width="700px" @closed="handleScheduleDialogClosed">
      <el-form :model="scheduleForm" label-width="96px">
        <el-form-item label="排班名称" required>
          <el-input v-model="scheduleForm.name" />
        </el-form-item>
        <el-form-item label="团队">
          <el-select v-model="scheduleForm.team_id" style="width: 100%" clearable>
            <el-option v-for="team in teams" :key="team.id" :label="team.name" :value="team.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="值班类型">
          <el-select v-model="scheduleForm.type" style="width: 100%">
            <el-option label="按天" value="daily" />
            <el-option label="按周" value="weekly" />
            <el-option label="自定义" value="custom" />
          </el-select>
        </el-form-item>
        <el-form-item label="时区">
          <el-input v-model="scheduleForm.timezone" />
        </el-form-item>
        <el-form-item label="值班时段">
          <el-input v-model="scheduleForm.start_time" style="width: 120px" />
          <span style="padding: 0 8px">~</span>
          <el-input v-model="scheduleForm.end_time" style="width: 120px" />
        </el-form-item>
        <el-form-item label="轮换规则" required>
          <el-input
            v-model="scheduleForm.rotation"
            type="textarea"
            :rows="6"
            placeholder='JSON数组，例如 [{"user_id":"u1","username":"alice","phone":"138..."}]'
          />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="scheduleForm.description" type="textarea" :rows="2" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="scheduleForm.enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="scheduleDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveSchedule">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog append-to-body v-model="teamDialogVisible" title="新增团队" width="520px" @closed="handleTeamDialogClosed">
      <el-form :model="teamForm" label-width="88px">
        <el-form-item label="团队名称" required>
          <el-input v-model="teamForm.name" />
        </el-form-item>
        <el-form-item label="成员JSON">
          <el-input v-model="teamForm.members" type="textarea" :rows="4" placeholder='["u1","u2"]' />
        </el-form-item>
        <el-form-item label="通知组">
          <el-input v-model="teamForm.notify_group" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="teamForm.description" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="teamDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="teamSaving" @click="createTeam">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog append-to-body v-model="generateDialogVisible" title="生成班次" width="460px" @closed="handleGenerateDialogClosed">
      <el-form :model="generateForm" label-width="96px">
        <el-form-item label="开始日期" required>
          <el-date-picker v-model="generateForm.start_date" type="date" value-format="YYYY-MM-DD" style="width: 100%" />
        </el-form-item>
        <el-form-item label="生成天数" required>
          <el-input-number v-model="generateForm.days" :min="1" :max="365" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="generateDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="generating" @click="generateShifts">生成</el-button>
      </template>
    </el-dialog>

    <el-dialog append-to-body v-model="swapDialogVisible" title="换班" width="460px" @closed="handleSwapDialogClosed">
      <el-form :model="swapForm" label-width="96px">
        <el-form-item label="接班用户ID" required>
          <el-input v-model="swapForm.override_user_id" />
        </el-form-item>
        <el-form-item label="接班用户名" required>
          <el-input v-model="swapForm.override_user" />
        </el-form-item>
        <el-form-item label="原因">
          <el-input v-model="swapForm.reason" type="textarea" :rows="3" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="swapDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="swapping" @click="swapShift">确认</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getErrorMessage, isCancelError } from '@/utils/error'
import StatusBadge from '@/components/common/StatusBadge.vue'
import { booleanEnabledStatusMeta, oncallShiftStatusMeta } from '@/utils/status'

const loading = ref(false)
const shiftLoading = ref(false)
const schedules = ref([])
const teams = ref([])
const shifts = ref([])
const currentOncall = ref([])
const selectedSchedule = ref(null)

const saving = ref(false)
const teamSaving = ref(false)
const generating = ref(false)
const swapping = ref(false)

const scheduleDialogVisible = ref(false)
const scheduleEditing = ref(false)
const teamDialogVisible = ref(false)
const generateDialogVisible = ref(false)
const swapDialogVisible = ref(false)

const generateTarget = ref(null)
const swapTarget = ref(null)

const scheduleForm = reactive({
  id: '',
  name: '',
  team_id: '',
  team_name: '',
  type: 'daily',
  timezone: 'Asia/Shanghai',
  start_time: '09:00',
  end_time: '18:00',
  rotation: '[{"user_id":"u1","username":"alice"}]',
  enabled: true,
  description: ''
})

const teamForm = reactive({
  name: '',
  members: '[]',
  notify_group: '',
  description: ''
})

const generateForm = reactive({
  start_date: '',
  days: 7
})

const swapForm = reactive({
  override_user_id: '',
  override_user: '',
  reason: ''
})

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const scheduleEnabledBadge = (row) => booleanEnabledStatusMeta(row, {
  source: '值班排班',
  enabledReason: '排班已启用',
  disabledReason: '排班已停用',
  checkAt: row?.updated_at || row?.created_at
})

const shiftStatusBadge = (row) => oncallShiftStatusMeta(row, {
  source: '值班班次',
  checkAt: row?.updated_at || row?.start_at
})

const formatTime = (value) => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString()
}

const fetchSchedules = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/v1/oncall/schedules', { headers: authHeaders() })
    if (res.data?.code === 0) {
      schedules.value = res.data.data || []
      if (selectedSchedule.value?.id) {
        selectedSchedule.value = schedules.value.find(item => item.id === selectedSchedule.value.id) || null
      }
      if (!selectedSchedule.value && schedules.value.length) {
        selectedSchedule.value = schedules.value[0]
      }
    }
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '加载排班失败'))
  } finally {
    loading.value = false
  }
}

const fetchTeams = async () => {
  try {
    const res = await axios.get('/api/v1/oncall/teams', { headers: authHeaders() })
    if (res.data?.code === 0) teams.value = res.data.data || []
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '加载团队失败'))
  }
}

const fetchCurrentOncall = async () => {
  try {
    const res = await axios.get('/api/v1/oncall/whoisoncall', { headers: authHeaders() })
    if (res.data?.code === 0) currentOncall.value = res.data.data || []
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '获取当前值班失败'))
  }
}

const fetchShifts = async () => {
  if (!selectedSchedule.value?.id) {
    shifts.value = []
    return
  }
  shiftLoading.value = true
  try {
    const res = await axios.get(`/api/v1/oncall/schedules/${selectedSchedule.value.id}/shifts`, { headers: authHeaders() })
    if (res.data?.code === 0) shifts.value = res.data.data || []
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '加载班次失败'))
  } finally {
    shiftLoading.value = false
  }
}

const refreshAll = async () => {
  await Promise.all([fetchSchedules(), fetchTeams(), fetchCurrentOncall()])
  await fetchShifts()
}

const selectSchedule = async (row) => {
  selectedSchedule.value = row
  await fetchShifts()
}

const resetScheduleForm = () => {
  scheduleForm.id = ''
  scheduleForm.name = ''
  scheduleForm.team_id = ''
  scheduleForm.team_name = ''
  scheduleForm.type = 'daily'
  scheduleForm.timezone = 'Asia/Shanghai'
  scheduleForm.start_time = '09:00'
  scheduleForm.end_time = '18:00'
  scheduleForm.rotation = '[{"user_id":"u1","username":"alice"}]'
  scheduleForm.enabled = true
  scheduleForm.description = ''
}

const resetTeamForm = () => {
  teamForm.name = ''
  teamForm.members = '[]'
  teamForm.notify_group = ''
  teamForm.description = ''
}

const resetGenerateForm = () => {
  generateTarget.value = null
  generateForm.start_date = ''
  generateForm.days = 7
}

const resetSwapForm = () => {
  swapTarget.value = null
  swapForm.override_user_id = ''
  swapForm.override_user = ''
  swapForm.reason = ''
}

const handleScheduleDialogClosed = () => {
  scheduleEditing.value = false
  resetScheduleForm()
}

const handleTeamDialogClosed = () => {
  resetTeamForm()
}

const handleGenerateDialogClosed = () => {
  resetGenerateForm()
}

const handleSwapDialogClosed = () => {
  resetSwapForm()
}

const openScheduleDialog = (row) => {
  scheduleEditing.value = !!row
  resetScheduleForm()
  if (row) {
    scheduleForm.id = row.id
    scheduleForm.name = row.name || ''
    scheduleForm.team_id = row.team_id || ''
    scheduleForm.team_name = row.team_name || ''
    scheduleForm.type = row.type || 'daily'
    scheduleForm.timezone = row.timezone || 'Asia/Shanghai'
    scheduleForm.start_time = row.start_time || '09:00'
    scheduleForm.end_time = row.end_time || '18:00'
    scheduleForm.rotation = row.rotation || '[]'
    scheduleForm.enabled = !!row.enabled
    scheduleForm.description = row.description || ''
  }
  scheduleDialogVisible.value = true
}

const saveSchedule = async () => {
  if (!scheduleForm.name.trim()) {
    ElMessage.warning('请输入排班名称')
    return
  }
  try {
    JSON.parse(scheduleForm.rotation || '[]')
  } catch {
    ElMessage.warning('轮换规则必须是合法JSON')
    return
  }

  saving.value = true
  try {
    const team = teams.value.find(item => item.id === scheduleForm.team_id)
    const payload = {
      name: scheduleForm.name.trim(),
      team_id: scheduleForm.team_id,
      team_name: team?.name || scheduleForm.team_name || '',
      type: scheduleForm.type,
      timezone: scheduleForm.timezone,
      start_time: scheduleForm.start_time,
      end_time: scheduleForm.end_time,
      rotation: scheduleForm.rotation,
      enabled: scheduleForm.enabled,
      description: scheduleForm.description
    }

    if (scheduleEditing.value && scheduleForm.id) {
      await axios.put(`/api/v1/oncall/schedules/${scheduleForm.id}`, payload, { headers: authHeaders() })
      ElMessage.success('更新成功')
    } else {
      await axios.post('/api/v1/oncall/schedules', payload, { headers: authHeaders() })
      ElMessage.success('创建成功')
    }

    scheduleDialogVisible.value = false
    await fetchSchedules()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '保存排班失败'))
  } finally {
    saving.value = false
  }
}

const removeSchedule = async (row) => {
  try {
    await ElMessageBox.confirm(`确认删除排班 ${row.name} ?`, '提示', { type: 'warning' })
    await axios.delete(`/api/v1/oncall/schedules/${row.id}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    if (selectedSchedule.value?.id === row.id) selectedSchedule.value = null
    await refreshAll()
  } catch (err) {
    if (!isCancelError(err)) ElMessage.error(getErrorMessage(err, '删除失败'))
  }
}

const openTeamDialog = () => {
  resetTeamForm()
  teamDialogVisible.value = true
}

const createTeam = async () => {
  if (!teamForm.name.trim()) {
    ElMessage.warning('请输入团队名称')
    return
  }
  try {
    JSON.parse(teamForm.members || '[]')
  } catch {
    ElMessage.warning('成员必须是合法JSON')
    return
  }

  teamSaving.value = true
  try {
    await axios.post('/api/v1/oncall/teams', {
      name: teamForm.name.trim(),
      members: teamForm.members,
      notify_group: teamForm.notify_group,
      description: teamForm.description
    }, { headers: authHeaders() })
    ElMessage.success('创建团队成功')
    teamDialogVisible.value = false
    await fetchTeams()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '创建团队失败'))
  } finally {
    teamSaving.value = false
  }
}

const openGenerateDialog = (row) => {
  generateTarget.value = row
  generateForm.start_date = new Date().toISOString().slice(0, 10)
  generateForm.days = 7
  generateDialogVisible.value = true
}

const generateShifts = async () => {
  if (!generateTarget.value?.id) return
  if (!generateForm.start_date || Number(generateForm.days) <= 0) {
    ElMessage.warning('请填写开始日期和生成天数')
    return
  }
  generating.value = true
  try {
    const res = await axios.post(`/api/v1/oncall/schedules/${generateTarget.value.id}/generate`, {
      start_date: generateForm.start_date,
      days: Number(generateForm.days)
    }, { headers: authHeaders() })
    ElMessage.success(`已生成 ${res.data?.data?.count || 0} 条班次`)
    generateDialogVisible.value = false
    if (selectedSchedule.value?.id === generateTarget.value.id) await fetchShifts()
    await fetchCurrentOncall()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '生成失败'))
  } finally {
    generating.value = false
  }
}

const openSwapDialog = (row) => {
  swapTarget.value = row
  resetSwapForm()
  swapTarget.value = row
  swapDialogVisible.value = true
}

const swapShift = async () => {
  if (!swapTarget.value?.id) return
  if (!swapForm.override_user_id.trim() || !swapForm.override_user.trim()) {
    ElMessage.warning('请填写接班用户信息')
    return
  }
  swapping.value = true
  try {
    await axios.post(`/api/v1/oncall/shifts/${swapTarget.value.id}/swap`, {
      override_user_id: swapForm.override_user_id.trim(),
      override_user: swapForm.override_user.trim(),
      reason: swapForm.reason.trim()
    }, { headers: authHeaders() })
    ElMessage.success('换班成功')
    swapDialogVisible.value = false
    await fetchShifts()
    await fetchCurrentOncall()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '换班失败'))
  } finally {
    swapping.value = false
  }
}

onMounted(refreshAll)
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.page-header { display: flex; justify-content: space-between; align-items: center; gap: 12px; margin-bottom: 12px; }
.page-desc { color: #909399; margin: 4px 0 0; }
.page-actions { display: flex; align-items: center; gap: 8px; }
.section-header { display: flex; justify-content: space-between; align-items: center; gap: 12px; }
.k { color: #909399; font-size: 12px; }
.v { font-size: 26px; font-weight: 700; margin-top: 4px; }
.current-panel { min-height: 120px; }
.sub-title { font-size: 13px; color: #606266; margin-bottom: 8px; }
.muted { color: #909399; font-size: 12px; }
.mb-12 { margin-bottom: 12px; }
.mt-12 { margin-top: 12px; }
.mr-6 { margin-right: 6px; }
.mb-6 { margin-bottom: 6px; }
</style>
