<template>
  <el-card class="page-card">
    <template #header>
      <div class="header">
        <div>
          <div class="title">配置管理</div>
          <div class="desc">配置项版本与同步计划</div>
        </div>
        <div class="actions">
          <el-button icon="Refresh" @click="refreshActive">刷新</el-button>
        </div>
      </div>
    </template>

    <el-tabs v-model="activeTab">
      <el-tab-pane label="配置列表" name="configs">
        <div class="filters">
          <el-select v-model="filters.server_id" placeholder="服务器" clearable @change="fetchConfigs">
            <el-option v-for="server in servers" :key="server.id" :label="server.name" :value="server.id" />
          </el-select>
          <el-input v-model="filters.group" placeholder="Group" clearable @clear="fetchConfigs" @keyup.enter="fetchConfigs" />
          <el-input v-model="filters.data_id" placeholder="DataID" clearable @clear="fetchConfigs" @keyup.enter="fetchConfigs" />
          <el-button type="primary" @click="fetchConfigs">查询</el-button>
        </div>

        <el-table :data="configs" v-loading="loadingConfigs" stripe>
          <el-table-column prop="data_id" label="DataID" min-width="200" />
          <el-table-column prop="group" label="Group" width="140" />
          <el-table-column prop="content_type" label="类型" width="100" />
          <el-table-column prop="app_name" label="应用" width="140" />
          <el-table-column prop="updated_at" label="更新时间" width="180" />
          <el-table-column label="操作" width="160" fixed="right">
            <template #default="{ row }">
              <el-button size="small" type="primary" plain @click="openConfig(row)">查看</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="同步计划" name="schedules">
        <div class="actions" style="margin-bottom: 12px;">
          <el-button type="primary" icon="Plus" @click="openScheduleCreate">新增计划</el-button>
        </div>
        <el-table :data="schedules" v-loading="loadingSchedules" stripe>
          <el-table-column prop="name" label="名称" min-width="160" />
          <el-table-column label="服务器" min-width="180">
            <template #default="{ row }">
              {{ serverName(row.server_id) }}
            </template>
          </el-table-column>
          <el-table-column prop="cron" label="Cron" min-width="160" />
          <el-table-column prop="enabled" label="状态" width="120">
            <template #default="{ row }">
              <el-tag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? '启用' : '停用' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="next_run_at" label="下次执行" width="180" />
          <el-table-column label="操作" width="220" fixed="right">
            <template #default="{ row }">
              <el-button size="small" type="primary" plain @click="openScheduleEdit(row)">编辑</el-button>
              <el-button size="small" type="warning" plain @click="toggleSchedule(row)">{{ row.enabled ? '停用' : '启用' }}</el-button>
              <el-button size="small" type="danger" plain @click="deleteSchedule(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
    </el-tabs>
  </el-card>

  <el-dialog v-model="configDialog" title="配置详情" width="760px">
    <el-form :model="currentConfig" label-width="90px">
      <el-form-item label="DataID">
        <el-input v-model="currentConfig.data_id" disabled />
      </el-form-item>
      <el-form-item label="Group">
        <el-input v-model="currentConfig.group" disabled />
      </el-form-item>
      <el-form-item label="内容">
        <el-input v-model="currentConfig.content" type="textarea" :rows="12" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="configDialog = false">关闭</el-button>
      <el-button type="primary" :loading="savingConfig" @click="saveConfig">保存</el-button>
    </template>
  </el-dialog>

  <el-dialog v-model="scheduleDialog" :title="scheduleEdit ? '编辑同步计划' : '新增同步计划'" width="560px">
    <el-form :model="scheduleForm" label-width="100px">
      <el-form-item label="名称" required>
        <el-input v-model="scheduleForm.name" />
      </el-form-item>
      <el-form-item label="服务器" required>
        <el-select v-model="scheduleForm.server_id" style="width: 100%">
          <el-option v-for="server in servers" :key="server.id" :label="server.name" :value="server.id" />
        </el-select>
      </el-form-item>
      <el-form-item label="Cron" required>
        <el-input v-model="scheduleForm.cron" placeholder="0 */10 * * * *" />
      </el-form-item>
      <el-form-item label="描述">
        <el-input v-model="scheduleForm.description" />
      </el-form-item>
      <el-form-item label="启用">
        <el-switch v-model="scheduleForm.enabled" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="scheduleDialog = false">取消</el-button>
      <el-button type="primary" :loading="savingSchedule" @click="saveSchedule">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, onMounted, watch } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const activeTab = ref('configs')
const servers = ref([])

const configs = ref([])
const loadingConfigs = ref(false)
const savingConfig = ref(false)
const configDialog = ref(false)
const currentConfig = reactive({})

const schedules = ref([])
const loadingSchedules = ref(false)
const savingSchedule = ref(false)
const scheduleDialog = ref(false)
const scheduleEdit = ref(false)
const scheduleId = ref('')

const filters = reactive({
  server_id: '',
  group: '',
  data_id: ''
})

const scheduleForm = reactive({
  name: '',
  server_id: '',
  cron: '',
  description: '',
  enabled: true
})

const headers = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const fetchServers = async () => {
  const res = await axios.get('/api/v1/nacos/servers', { headers: headers() })
  if (res.data.code === 0) {
    servers.value = res.data.data
  }
}

const fetchConfigs = async () => {
  loadingConfigs.value = true
  try {
    const res = await axios.get('/api/v1/nacos/configs', {
      headers: headers(),
      params: { server_id: filters.server_id, group: filters.group, data_id: filters.data_id }
    })
    if (res.data.code === 0) {
      configs.value = res.data.data
    }
  } catch (error) {
    ElMessage.error('加载配置失败')
  } finally {
    loadingConfigs.value = false
  }
}

const openConfig = (row) => {
  Object.assign(currentConfig, row)
  configDialog.value = true
}

const saveConfig = async () => {
  if (!currentConfig.id) return
  savingConfig.value = true
  try {
    const res = await axios.put(`/api/v1/nacos/configs/${currentConfig.id}`, currentConfig, { headers: headers() })
    if (res.data.code === 0) {
      ElMessage.success('更新成功')
      configDialog.value = false
      fetchConfigs()
    }
  } catch (error) {
    ElMessage.error('保存失败')
  } finally {
    savingConfig.value = false
  }
}

const fetchSchedules = async () => {
  loadingSchedules.value = true
  try {
    const res = await axios.get('/api/v1/nacos/schedules', { headers: headers() })
    if (res.data.code === 0) {
      schedules.value = res.data.data
    }
  } catch (error) {
    ElMessage.error('加载计划失败')
  } finally {
    loadingSchedules.value = false
  }
}

const serverName = (id) => {
  const server = servers.value.find((item) => item.id === id)
  return server ? server.name : id
}

const openScheduleCreate = () => {
  scheduleEdit.value = false
  scheduleId.value = ''
  Object.assign(scheduleForm, { name: '', server_id: '', cron: '', description: '', enabled: true })
  scheduleDialog.value = true
}

const openScheduleEdit = (row) => {
  scheduleEdit.value = true
  scheduleId.value = row.id
  Object.assign(scheduleForm, row)
  scheduleDialog.value = true
}

const saveSchedule = async () => {
  if (!scheduleForm.name || !scheduleForm.server_id || !scheduleForm.cron) {
    ElMessage.warning('请补全必填项')
    return
  }
  savingSchedule.value = true
  try {
    const url = scheduleEdit.value ? `/api/v1/nacos/schedules/${scheduleId.value}` : '/api/v1/nacos/schedules'
    const method = scheduleEdit.value ? 'put' : 'post'
    const res = await axios({ url, method, data: scheduleForm, headers: headers() })
    if (res.data.code === 0) {
      ElMessage.success('保存成功')
      scheduleDialog.value = false
      fetchSchedules()
    }
  } catch (error) {
    ElMessage.error('保存失败')
  } finally {
    savingSchedule.value = false
  }
}

const deleteSchedule = (row) => {
  ElMessageBox.confirm(`确定删除“${row.name}”吗？`, '提示', { type: 'warning' }).then(async () => {
    await axios.delete(`/api/v1/nacos/schedules/${row.id}`, { headers: headers() })
    ElMessage.success('删除成功')
    fetchSchedules()
  })
}

const toggleSchedule = async (row) => {
  await axios.post(`/api/v1/nacos/schedules/${row.id}/toggle`, {}, { headers: headers() })
  ElMessage.success('已更新状态')
  fetchSchedules()
}

const refreshActive = () => {
  if (activeTab.value === 'configs') {
    fetchConfigs()
  } else {
    fetchSchedules()
  }
}

watch(activeTab, (val) => {
  if (val === 'configs') {
    fetchConfigs()
  } else {
    fetchSchedules()
  }
})

onMounted(async () => {
  await fetchServers()
  await fetchConfigs()
  await fetchSchedules()
})
</script>

<style scoped>
.page-card { max-width: 1200px; margin: 0 auto; }
.header { display: flex; justify-content: space-between; align-items: center; }
.title { font-size: 18px; font-weight: 600; }
.desc { color: #909399; margin-top: 4px; }
.actions { display: flex; gap: 8px; }
.filters { display: flex; gap: 12px; margin-bottom: 16px; }
.filters .el-select { width: 200px; }
.filters .el-input { width: 240px; }
</style>
