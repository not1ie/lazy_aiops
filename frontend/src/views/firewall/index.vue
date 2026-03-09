<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>防火墙管理</h2>
        <p class="page-desc">设备资产、SNMP采集和规则清单统一管理。</p>
      </div>
      <div class="page-actions">
        <el-input v-model="keyword" placeholder="搜索设备名/IP" clearable style="width: 240px" />
        <el-button type="primary" icon="Plus" @click="openDeviceDialog()">新增设备</el-button>
        <el-button icon="Refresh" @click="fetchDevices">刷新</el-button>
      </div>
    </div>

    <el-row :gutter="12" class="mb-12">
      <el-col :span="6"><el-card><div class="k">设备总数</div><div class="v">{{ filteredDevices.length }}</div></el-card></el-col>
      <el-col :span="6"><el-card><div class="k">在线</div><div class="v success">{{ onlineCount }}</div></el-card></el-col>
      <el-col :span="6"><el-card><div class="k">告警</div><div class="v warning">{{ alertCount }}</div></el-card></el-col>
      <el-col :span="6"><el-card><div class="k">离线</div><div class="v danger">{{ offlineCount }}</div></el-card></el-col>
    </el-row>

    <el-table :fit="true" :data="filteredDevices" v-loading="loading" stripe @row-click="selectDevice">
      <el-table-column prop="name" label="设备名" min-width="150" />
      <el-table-column prop="vendor" label="厂商" width="120" />
      <el-table-column prop="model" label="型号" width="140" />
      <el-table-column prop="ip" label="管理IP" width="150" />
      <el-table-column label="CPU" width="110">
        <template #default="{ row }">{{ formatPercent(row.cpu_usage) }}</template>
      </el-table-column>
      <el-table-column label="内存" width="110">
        <template #default="{ row }">{{ formatPercent(row.memory_usage) }}</template>
      </el-table-column>
      <el-table-column label="会话数" width="120">
        <template #default="{ row }">{{ row.session_count || 0 }}</template>
      </el-table-column>
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="statusTag(row.status)">{{ statusText(row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="最后采集" width="180">
        <template #default="{ row }">{{ formatTime(row.last_check_at) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="330" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click.stop="openDeviceDialog(row)">编辑</el-button>
          <el-button size="small" type="primary" plain :loading="testingId === row.id" @click.stop="testSNMP(row)">测试SNMP</el-button>
          <el-button size="small" type="success" plain :loading="collectingId === row.id" @click.stop="collectSNMP(row)">采集</el-button>
          <el-button size="small" type="danger" plain @click.stop="removeDevice(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-card class="mt-12" v-if="selectedDevice">
      <template #header>
        <div class="section-header">
          <span>设备详情：{{ selectedDevice.name }} ({{ selectedDevice.ip }})</span>
          <el-button size="small" type="primary" icon="Plus" @click="openRuleDialog">新增规则</el-button>
        </div>
      </template>

      <el-tabs v-model="activeDetailTab" @tab-change="handleDetailTabChange">
        <el-tab-pane label="SNMP指标" name="metrics">
          <div class="toolbar">
            <el-select v-model="metricType" clearable placeholder="指标类型" style="width: 160px" @change="fetchMetrics">
              <el-option label="CPU" value="cpu" />
              <el-option label="Memory" value="memory" />
              <el-option label="Session" value="session" />
              <el-option label="Uptime" value="uptime" />
            </el-select>
            <el-button icon="Refresh" @click="fetchMetrics">刷新指标</el-button>
          </div>
          <el-table :fit="true" :data="metrics" v-loading="metricsLoading" stripe>
            <el-table-column prop="metric_name" label="指标" min-width="180" />
            <el-table-column prop="metric_type" label="类型" width="120" />
            <el-table-column label="值" width="140">
              <template #default="{ row }">{{ row.value }} {{ row.unit || '' }}</template>
            </el-table-column>
            <el-table-column label="采集时间" width="180">
              <template #default="{ row }">{{ formatTime(row.collected_at) }}</template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="访问规则" name="rules">
          <el-table :fit="true" :data="rules" v-loading="rulesLoading" stripe>
            <el-table-column prop="name" label="规则名" min-width="160" />
            <el-table-column prop="priority" label="优先级" width="90" />
            <el-table-column prop="action" label="动作" width="90" />
            <el-table-column prop="src_addr" label="源地址" min-width="160" />
            <el-table-column prop="dst_addr" label="目标地址" min-width="160" />
            <el-table-column prop="service" label="服务" width="120" />
            <el-table-column label="状态" width="90">
              <template #default="{ row }">
                <el-tag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? '启用' : '禁用' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="110" fixed="right">
              <template #default="{ row }">
                <el-button size="small" type="danger" plain @click="removeRule(row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <el-dialog append-to-body v-model="deviceDialogVisible" :title="deviceEditing ? '编辑设备' : '新增设备'" width="760px">
      <el-form :model="deviceForm" label-width="96px">
        <el-row :gutter="12">
          <el-col :span="12"><el-form-item label="设备名" required><el-input v-model="deviceForm.name" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="厂商"><el-select v-model="deviceForm.vendor" style="width: 100%"><el-option label="华为" value="huawei" /><el-option label="Cisco" value="cisco" /><el-option label="Fortinet" value="fortinet" /><el-option label="Paloalto" value="paloalto" /></el-select></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="型号"><el-input v-model="deviceForm.model" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="管理IP" required><el-input v-model="deviceForm.ip" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="管理端口"><el-input-number v-model="deviceForm.manage_port" :min="1" :max="65535" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="SNMP版本"><el-select v-model="deviceForm.snmp_version" style="width: 100%"><el-option label="v2c" value="v2c" /><el-option label="v3" value="v3" /></el-select></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="SNMP端口"><el-input-number v-model="deviceForm.snmp_port" :min="1" :max="65535" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="Community"><el-input v-model="deviceForm.snmp_community" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="SNMP用户"><el-input v-model="deviceForm.snmp_user" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="Auth密码"><el-input v-model="deviceForm.snmp_auth_pass" type="password" show-password /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="Priv密码"><el-input v-model="deviceForm.snmp_priv_pass" type="password" show-password /></el-form-item></el-col>
          <el-col :span="24"><el-form-item label="描述"><el-input v-model="deviceForm.description" type="textarea" :rows="2" /></el-form-item></el-col>
        </el-row>
      </el-form>
      <template #footer>
        <el-button @click="deviceDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="savingDevice" @click="saveDevice">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog append-to-body v-model="ruleDialogVisible" title="新增访问规则" width="700px">
      <el-form :model="ruleForm" label-width="96px">
        <el-row :gutter="12">
          <el-col :span="12"><el-form-item label="规则名" required><el-input v-model="ruleForm.name" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="优先级"><el-input-number v-model="ruleForm.priority" :min="1" :max="10000" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="动作"><el-select v-model="ruleForm.action" style="width: 100%"><el-option label="allow" value="allow" /><el-option label="deny" value="deny" /></el-select></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="协议"><el-input v-model="ruleForm.protocol" placeholder="tcp/udp/icmp" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="源区域"><el-input v-model="ruleForm.src_zone" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="目标区域"><el-input v-model="ruleForm.dst_zone" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="源地址"><el-input v-model="ruleForm.src_addr" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="目标地址"><el-input v-model="ruleForm.dst_addr" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="服务"><el-input v-model="ruleForm.service" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="端口"><el-input v-model="ruleForm.dst_port" /></el-form-item></el-col>
          <el-col :span="24"><el-form-item label="描述"><el-input v-model="ruleForm.description" type="textarea" :rows="2" /></el-form-item></el-col>
        </el-row>
      </el-form>
      <template #footer>
        <el-button @click="ruleDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="savingRule" @click="saveRule">保存</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getErrorMessage, isCancelError } from '@/utils/error'

const loading = ref(false)
const metricsLoading = ref(false)
const rulesLoading = ref(false)
const savingDevice = ref(false)
const savingRule = ref(false)
const testingId = ref('')
const collectingId = ref('')

const keyword = ref('')
const devices = ref([])
const selectedDevice = ref(null)
const metrics = ref([])
const rules = ref([])
const metricType = ref('')
const activeDetailTab = ref('metrics')

const deviceDialogVisible = ref(false)
const deviceEditing = ref(false)

const ruleDialogVisible = ref(false)

const deviceForm = reactive({
  id: '',
  name: '',
  vendor: 'huawei',
  model: '',
  ip: '',
  manage_port: 443,
  snmp_version: 'v2c',
  snmp_community: 'public',
  snmp_port: 161,
  snmp_user: '',
  snmp_auth_pass: '',
  snmp_priv_pass: '',
  description: ''
})

const ruleForm = reactive({
  name: '',
  priority: 100,
  action: 'allow',
  src_zone: '',
  dst_zone: '',
  src_addr: '',
  dst_addr: '',
  service: 'any',
  protocol: 'tcp',
  src_port: '',
  dst_port: '',
  enabled: true,
  description: ''
})

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const filteredDevices = computed(() => {
  const key = keyword.value.trim().toLowerCase()
  if (!key) return devices.value
  return devices.value.filter(item => (
    (item.name || '').toLowerCase().includes(key) ||
    (item.ip || '').toLowerCase().includes(key)
  ))
})

const onlineCount = computed(() => filteredDevices.value.filter(item => Number(item.status) === 1).length)
const alertCount = computed(() => filteredDevices.value.filter(item => Number(item.status) === 2).length)
const offlineCount = computed(() => filteredDevices.value.filter(item => Number(item.status) === 0).length)

const formatTime = (value) => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString()
}

const formatPercent = (value) => {
  const num = Number(value || 0)
  return `${num.toFixed(1)}%`
}

const statusText = (status) => {
  if (Number(status) === 1) return '在线'
  if (Number(status) === 2) return '告警'
  return '离线'
}

const statusTag = (status) => {
  if (Number(status) === 1) return 'success'
  if (Number(status) === 2) return 'warning'
  return 'info'
}

const fetchDevices = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/v1/firewall/devices', { headers: authHeaders() })
    if (res.data?.code === 0) {
      devices.value = res.data.data || []
      if (selectedDevice.value) {
        const latest = devices.value.find(item => item.id === selectedDevice.value.id)
        selectedDevice.value = latest || null
      }
      if (!selectedDevice.value && devices.value.length) {
        selectedDevice.value = devices.value[0]
      }
      await loadDetailData()
    }
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '加载设备失败'))
  } finally {
    loading.value = false
  }
}

const selectDevice = async (row) => {
  selectedDevice.value = row
  await loadDetailData()
}

const handleDetailTabChange = async () => {
  await loadDetailData()
}

const loadDetailData = async () => {
  if (!selectedDevice.value?.id) {
    metrics.value = []
    rules.value = []
    return
  }
  if (activeDetailTab.value === 'metrics') {
    await fetchMetrics()
  } else {
    await fetchRules()
  }
}

const fetchMetrics = async () => {
  if (!selectedDevice.value?.id) return
  metricsLoading.value = true
  try {
    const res = await axios.get(`/api/v1/firewall/devices/${selectedDevice.value.id}/snmp/metrics`, {
      headers: authHeaders(),
      params: { type: metricType.value || undefined }
    })
    if (res.data?.code === 0) metrics.value = res.data.data || []
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '加载指标失败'))
  } finally {
    metricsLoading.value = false
  }
}

const fetchRules = async () => {
  if (!selectedDevice.value?.id) return
  rulesLoading.value = true
  try {
    const res = await axios.get(`/api/v1/firewall/devices/${selectedDevice.value.id}/rules`, { headers: authHeaders() })
    if (res.data?.code === 0) rules.value = res.data.data || []
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '加载规则失败'))
  } finally {
    rulesLoading.value = false
  }
}

const resetDeviceForm = () => {
  deviceForm.id = ''
  deviceForm.name = ''
  deviceForm.vendor = 'huawei'
  deviceForm.model = ''
  deviceForm.ip = ''
  deviceForm.manage_port = 443
  deviceForm.snmp_version = 'v2c'
  deviceForm.snmp_community = 'public'
  deviceForm.snmp_port = 161
  deviceForm.snmp_user = ''
  deviceForm.snmp_auth_pass = ''
  deviceForm.snmp_priv_pass = ''
  deviceForm.description = ''
}

const openDeviceDialog = (row) => {
  deviceEditing.value = !!row
  resetDeviceForm()
  if (row) {
    deviceForm.id = row.id
    deviceForm.name = row.name || ''
    deviceForm.vendor = row.vendor || 'huawei'
    deviceForm.model = row.model || ''
    deviceForm.ip = row.ip || ''
    deviceForm.manage_port = Number(row.manage_port || 443)
    deviceForm.snmp_version = row.snmp_version || 'v2c'
    deviceForm.snmp_community = row.snmp_community || 'public'
    deviceForm.snmp_port = Number(row.snmp_port || 161)
    deviceForm.snmp_user = row.snmp_user || ''
    deviceForm.description = row.description || ''
  }
  deviceDialogVisible.value = true
}

const saveDevice = async () => {
  if (!deviceForm.name.trim() || !deviceForm.ip.trim()) {
    ElMessage.warning('请填写设备名和管理IP')
    return
  }

  savingDevice.value = true
  try {
    const payload = {
      name: deviceForm.name.trim(),
      vendor: deviceForm.vendor,
      model: deviceForm.model,
      ip: deviceForm.ip.trim(),
      manage_port: Number(deviceForm.manage_port || 443),
      snmp_version: deviceForm.snmp_version,
      snmp_community: deviceForm.snmp_community,
      snmp_port: Number(deviceForm.snmp_port || 161),
      snmp_user: deviceForm.snmp_user,
      snmp_auth_pass: deviceForm.snmp_auth_pass,
      snmp_priv_pass: deviceForm.snmp_priv_pass,
      description: deviceForm.description
    }

    if (deviceEditing.value && deviceForm.id) {
      await axios.put(`/api/v1/firewall/devices/${deviceForm.id}`, payload, { headers: authHeaders() })
      ElMessage.success('更新成功')
    } else {
      await axios.post('/api/v1/firewall/devices', payload, { headers: authHeaders() })
      ElMessage.success('创建成功')
    }

    deviceDialogVisible.value = false
    await fetchDevices()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '保存设备失败'))
  } finally {
    savingDevice.value = false
  }
}

const removeDevice = async (row) => {
  try {
    await ElMessageBox.confirm(`确认删除设备 ${row.name} ?`, '提示', { type: 'warning' })
    await axios.delete(`/api/v1/firewall/devices/${row.id}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    if (selectedDevice.value?.id === row.id) selectedDevice.value = null
    await fetchDevices()
  } catch (err) {
    if (!isCancelError(err)) ElMessage.error(getErrorMessage(err, '删除失败'))
  }
}

const testSNMP = async (row) => {
  testingId.value = row.id
  try {
    const res = await axios.post(`/api/v1/firewall/devices/${row.id}/snmp/test`, {}, { headers: authHeaders() })
    ElMessage.success(`SNMP连接成功：${res.data?.data?.sys_desc || ''}`)
  } catch (err) {
    ElMessage.error(getErrorMessage(err, 'SNMP连接失败'))
  } finally {
    testingId.value = ''
  }
}

const collectSNMP = async (row) => {
  collectingId.value = row.id
  try {
    await axios.post(`/api/v1/firewall/devices/${row.id}/snmp/collect`, {}, { headers: authHeaders() })
    ElMessage.success('采集成功')
    if (selectedDevice.value?.id === row.id) await loadDetailData()
    await fetchDevices()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '采集失败'))
  } finally {
    collectingId.value = ''
  }
}

const resetRuleForm = () => {
  ruleForm.name = ''
  ruleForm.priority = 100
  ruleForm.action = 'allow'
  ruleForm.src_zone = ''
  ruleForm.dst_zone = ''
  ruleForm.src_addr = ''
  ruleForm.dst_addr = ''
  ruleForm.service = 'any'
  ruleForm.protocol = 'tcp'
  ruleForm.src_port = ''
  ruleForm.dst_port = ''
  ruleForm.enabled = true
  ruleForm.description = ''
}

const openRuleDialog = () => {
  if (!selectedDevice.value?.id) {
    ElMessage.warning('请先选择设备')
    return
  }
  resetRuleForm()
  ruleDialogVisible.value = true
}

const saveRule = async () => {
  if (!selectedDevice.value?.id) return
  if (!ruleForm.name.trim()) {
    ElMessage.warning('请输入规则名')
    return
  }

  savingRule.value = true
  try {
    await axios.post(`/api/v1/firewall/devices/${selectedDevice.value.id}/rules`, {
      name: ruleForm.name.trim(),
      priority: Number(ruleForm.priority || 100),
      action: ruleForm.action,
      src_zone: ruleForm.src_zone,
      dst_zone: ruleForm.dst_zone,
      src_addr: ruleForm.src_addr,
      dst_addr: ruleForm.dst_addr,
      service: ruleForm.service,
      protocol: ruleForm.protocol,
      src_port: ruleForm.src_port,
      dst_port: ruleForm.dst_port,
      enabled: ruleForm.enabled,
      description: ruleForm.description
    }, { headers: authHeaders() })

    ElMessage.success('规则创建成功')
    ruleDialogVisible.value = false
    await fetchRules()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '创建规则失败'))
  } finally {
    savingRule.value = false
  }
}

const removeRule = async (row) => {
  if (!selectedDevice.value?.id) return
  try {
    await ElMessageBox.confirm(`确认删除规则 ${row.name} ?`, '提示', { type: 'warning' })
    await axios.delete(`/api/v1/firewall/devices/${selectedDevice.value.id}/rules/${row.id}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    await fetchRules()
  } catch (err) {
    if (!isCancelError(err)) ElMessage.error(getErrorMessage(err, '删除规则失败'))
  }
}

onMounted(fetchDevices)
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.page-header { display: flex; justify-content: space-between; align-items: center; gap: 12px; margin-bottom: 12px; }
.page-desc { color: #909399; margin: 4px 0 0; }
.page-actions { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.k { color: #909399; font-size: 12px; }
.v { font-size: 26px; font-weight: 700; margin-top: 4px; }
.v.success { color: #67c23a; }
.v.warning { color: #e6a23c; }
.v.danger { color: #f56c6c; }
.mb-12 { margin-bottom: 12px; }
.mt-12 { margin-top: 12px; }
.section-header { display: flex; justify-content: space-between; align-items: center; gap: 8px; }
.toolbar { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; }
</style>
