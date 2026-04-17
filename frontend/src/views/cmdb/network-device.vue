<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>网络设备 CMDB</h2>
        <p class="page-desc">统一纳管交换机与防火墙资产，支持连通性测试与防火墙数据同步。</p>
      </div>
      <div class="page-actions">
        <el-input v-model="keyword" clearable placeholder="搜索名称/IP/型号" style="width: 240px" @keyup.enter="fetchData" />
        <el-select v-model="deviceType" clearable placeholder="设备类型" style="width: 140px" @change="fetchData">
          <el-option label="交换机" value="switch" />
          <el-option label="防火墙" value="firewall" />
        </el-select>
        <el-button type="primary" icon="Plus" @click="openDialog()">新增设备</el-button>
        <el-button icon="Refresh" @click="fetchData">刷新</el-button>
        <el-button icon="Connection" :loading="syncing" @click="syncFromFirewall">同步防火墙</el-button>
      </div>
    </div>

    <el-row :gutter="12" class="mb-12">
      <el-col :span="6"><el-card><div class="k">总设备</div><div class="v">{{ filteredData.length }}</div></el-card></el-col>
      <el-col :span="6"><el-card><div class="k">交换机</div><div class="v">{{ switchCount }}</div></el-card></el-col>
      <el-col :span="6"><el-card><div class="k">防火墙</div><div class="v">{{ firewallCount }}</div></el-card></el-col>
      <el-col :span="6"><el-card><div class="k">在线</div><div class="v success">{{ onlineCount }}</div></el-card></el-col>
    </el-row>

    <div class="table-scroll">
      <el-table :fit="true" :data="filteredData" v-loading="loading" stripe style="width: 100%; min-width: 1760px">
        <el-table-column prop="name" label="设备名" min-width="180" />
        <el-table-column label="类型" width="110">
          <template #default="{ row }">
            <el-tag :type="row.device_type === 'firewall' ? 'warning' : 'info'">
              {{ row.device_type === 'firewall' ? '防火墙' : '交换机' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="vendor" label="厂商" width="120" />
        <el-table-column prop="model" label="型号" min-width="160" />
        <el-table-column prop="ip" label="管理IP" width="150" />
        <el-table-column prop="manage_port" label="端口" width="90" />
        <el-table-column prop="location" label="位置" min-width="140" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="statusTag(row)">{{ statusText(row) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="最后检查" width="180">
          <template #default="{ row }">{{ formatTime(row.last_check_at) }}</template>
        </el-table-column>
        <el-table-column prop="status_reason" label="状态说明" width="240" show-overflow-tooltip />
        <el-table-column label="操作" width="300" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="primary" plain @click="openDialog(row)">编辑</el-button>
            <el-button size="small" type="success" plain :loading="testingId === row.id" @click="testDevice(row)">测试</el-button>
            <el-button size="small" type="danger" plain @click="removeDevice(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <el-dialog append-to-body v-model="dialogVisible" :title="isEdit ? '编辑网络设备' : '新增网络设备'" width="780px" @closed="handleDialogClosed">
      <el-form :model="form" label-width="96px">
        <el-row :gutter="12">
          <el-col :span="12"><el-form-item label="设备名" required><el-input v-model="form.name" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="设备类型" required><el-select v-model="form.device_type" style="width: 100%"><el-option label="交换机" value="switch" /><el-option label="防火墙" value="firewall" /></el-select></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="厂商"><el-input v-model="form.vendor" placeholder="huawei/cisco/h3c" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="型号"><el-input v-model="form.model" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="管理IP" required><el-input v-model="form.ip" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="管理端口"><el-input-number v-model="form.manage_port" :min="1" :max="65535" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="位置"><el-input v-model="form.location" placeholder="机房/区域" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="机柜"><el-input v-model="form.rack" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="序列号"><el-input v-model="form.serial_number" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="固件版本"><el-input v-model="form.firmware_version" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="状态"><el-select v-model="form.status" style="width: 100%"><el-option label="在线" :value="1" /><el-option label="离线" :value="0" /><el-option label="告警" :value="2" /></el-select></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="标签"><el-input v-model="form.tags" placeholder="prod,core" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="SSH用户"><el-input v-model="form.username" placeholder="用于连通性测试" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="SSH密码"><el-input v-model="form.password" type="password" show-password placeholder="空则不更新" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="SNMP版本"><el-select v-model="form.snmp_version" style="width: 100%"><el-option label="v1" value="v1" /><el-option label="v2c" value="v2c" /><el-option label="v3" value="v3" /></el-select></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="SNMP端口"><el-input-number v-model="form.snmp_port" :min="1" :max="65535" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="Community"><el-input v-model="form.snmp_community" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="SNMP用户"><el-input v-model="form.snmp_user" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="Auth协议"><el-select v-model="form.snmp_auth_proto" style="width: 100%"><el-option label="MD5" value="MD5" /><el-option label="SHA" value="SHA" /></el-select></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="Priv协议"><el-select v-model="form.snmp_priv_proto" style="width: 100%"><el-option label="DES" value="DES" /><el-option label="AES" value="AES" /></el-select></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="Auth密码"><el-input v-model="form.snmp_auth_pass" type="password" show-password /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="Priv密码"><el-input v-model="form.snmp_priv_pass" type="password" show-password /></el-form-item></el-col>
          <el-col :span="24"><el-form-item label="描述"><el-input v-model="form.description" type="textarea" :rows="2" /></el-form-item></el-col>
        </el-row>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitForm">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog append-to-body v-model="testVisible" title="连通性测试结果" width="640px">
      <el-skeleton v-if="testLoading" :rows="4" animated />
      <pre v-else class="test-json">{{ JSON.stringify(testResult, null, 2) }}</pre>
      <template #footer>
        <el-button @click="testVisible = false">关闭</el-button>
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
const saving = ref(false)
const syncing = ref(false)
const testLoading = ref(false)
const testingId = ref('')

const tableData = ref([])
const keyword = ref('')
const deviceType = ref('')

const dialogVisible = ref(false)
const isEdit = ref(false)
const currentId = ref('')

const testVisible = ref(false)
const testResult = ref({})

const form = reactive({
  name: '',
  device_type: 'switch',
  vendor: '',
  model: '',
  ip: '',
  manage_port: 22,
  snmp_version: 'v2c',
  snmp_community: 'public',
  snmp_port: 161,
  snmp_user: '',
  snmp_auth_proto: 'MD5',
  snmp_auth_pass: '',
  snmp_priv_proto: 'DES',
  snmp_priv_pass: '',
  username: '',
  password: '',
  location: '',
  rack: '',
  serial_number: '',
  firmware_version: '',
  status: 1,
  tags: '',
  description: ''
})

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const filteredData = computed(() => {
  const key = keyword.value.trim().toLowerCase()
  return tableData.value.filter((item) => {
    if (deviceType.value && item.device_type !== deviceType.value) return false
    if (!key) return true
    return [item.name, item.ip, item.vendor, item.model, item.serial_number]
      .map((v) => String(v || '').toLowerCase())
      .some((v) => v.includes(key))
  })
})

const switchCount = computed(() => filteredData.value.filter((item) => item.device_type === 'switch').length)
const firewallCount = computed(() => filteredData.value.filter((item) => item.device_type === 'firewall').length)
const onlineCount = computed(() => filteredData.value.filter((item) => Number(item.status) === 1 && !isStatusStale(item)).length)

const resetForm = () => {
  form.name = ''
  form.device_type = 'switch'
  form.vendor = ''
  form.model = ''
  form.ip = ''
  form.manage_port = 22
  form.snmp_version = 'v2c'
  form.snmp_community = 'public'
  form.snmp_port = 161
  form.snmp_user = ''
  form.snmp_auth_proto = 'MD5'
  form.snmp_auth_pass = ''
  form.snmp_priv_proto = 'DES'
  form.snmp_priv_pass = ''
  form.username = ''
  form.password = ''
  form.location = ''
  form.rack = ''
  form.serial_number = ''
  form.firmware_version = ''
  form.status = 1
  form.tags = ''
  form.description = ''
}

const formatTime = (value) => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString()
}

const toTime = (value) => {
  if (!value) return null
  const ts = new Date(value).getTime()
  return Number.isNaN(ts) ? null : ts
}

const isStatusStale = (row) => {
  const ts = toTime(row?.last_check_at)
  if (!ts) return true
  return Date.now() - ts > 5 * 60 * 1000
}

const statusText = (row) => {
  const status = Number(row?.status)
  if (status === 1) return isStatusStale(row) ? '在线(过期)' : '在线'
  if (status === 2) return '告警'
  return '离线'
}

const statusTag = (row) => {
  const status = Number(row?.status)
  if (status === 1) return isStatusStale(row) ? 'warning' : 'success'
  if (status === 2) return 'danger'
  return 'info'
}

const fetchData = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/v1/cmdb/network-devices', {
      headers: authHeaders(),
      params: {
        keyword: keyword.value || undefined,
        device_type: deviceType.value || undefined,
        live: 1
      }
    })
    if (res.data?.code === 0) {
      tableData.value = res.data.data || []
    }
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '加载网络设备失败'))
  } finally {
    loading.value = false
  }
}

const openDialog = async (row) => {
  resetForm()
  if (!row) {
    isEdit.value = false
    currentId.value = ''
    dialogVisible.value = true
    return
  }
  isEdit.value = true
  currentId.value = row.id
  try {
    const res = await axios.get(`/api/v1/cmdb/network-devices/${row.id}`, { headers: authHeaders() })
    if (res.data?.code === 0) {
      const data = res.data.data || {}
      form.name = data.name || ''
      form.device_type = data.device_type || 'switch'
      form.vendor = data.vendor || ''
      form.model = data.model || ''
      form.ip = data.ip || ''
      form.manage_port = data.manage_port || 22
      form.snmp_version = data.snmp_version || 'v2c'
      form.snmp_community = data.snmp_community || ''
      form.snmp_port = data.snmp_port || 161
      form.snmp_user = data.snmp_user || ''
      form.snmp_auth_proto = data.snmp_auth_proto || 'MD5'
      form.snmp_auth_pass = data.snmp_auth_pass || ''
      form.snmp_priv_proto = data.snmp_priv_proto || 'DES'
      form.snmp_priv_pass = data.snmp_priv_pass || ''
      form.location = data.location || ''
      form.rack = data.rack || ''
      form.serial_number = data.serial_number || ''
      form.firmware_version = data.firmware_version || ''
      form.status = data.status ?? 1
      form.tags = data.tags || ''
      form.description = data.description || ''
      form.username = data.credential?.username || ''
      form.password = data.credential?.password || ''
      dialogVisible.value = true
    }
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '加载设备详情失败'))
  }
}

const handleDialogClosed = () => {
  isEdit.value = false
  currentId.value = ''
  resetForm()
}

const submitForm = async () => {
  if (!form.name || !form.ip) {
    ElMessage.warning('请填写设备名和管理IP')
    return
  }
  saving.value = true
  try {
    const url = isEdit.value ? `/api/v1/cmdb/network-devices/${currentId.value}` : '/api/v1/cmdb/network-devices'
    const method = isEdit.value ? 'put' : 'post'
    const res = await axios({
      method,
      url,
      headers: authHeaders(),
      data: { ...form }
    })
    if (res.data?.code === 0) {
      ElMessage.success(isEdit.value ? '更新成功' : '创建成功')
      dialogVisible.value = false
      await fetchData()
    } else {
      ElMessage.error(res.data?.message || '保存失败')
    }
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '保存失败'))
  } finally {
    saving.value = false
  }
}

const removeDevice = async (row) => {
  try {
    await ElMessageBox.confirm(`确定删除设备 ${row.name} 吗？`, '删除确认', {
      type: 'warning',
      confirmButtonText: '删除',
      cancelButtonText: '取消'
    })
    await axios.delete(`/api/v1/cmdb/network-devices/${row.id}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    await fetchData()
  } catch (err) {
    if (!isCancelError(err)) {
      ElMessage.error(getErrorMessage(err, '删除失败'))
    }
  }
}

const testDevice = async (row) => {
  testingId.value = row.id
  testLoading.value = true
  testVisible.value = true
  testResult.value = {}
  try {
    const res = await axios.post(`/api/v1/cmdb/network-devices/${row.id}/test`, {}, { headers: authHeaders() })
    if (res.data?.code === 0) {
      testResult.value = res.data.data || {}
      ElMessage.success('测试完成')
      await fetchData()
    } else {
      testResult.value = { error: res.data?.message || '测试失败' }
      ElMessage.error(res.data?.message || '测试失败')
    }
  } catch (err) {
    testResult.value = { error: getErrorMessage(err, '测试失败') }
    ElMessage.error(getErrorMessage(err, '测试失败'))
  } finally {
    testingId.value = ''
    testLoading.value = false
  }
}

const syncFromFirewall = async () => {
  syncing.value = true
  try {
    const res = await axios.post('/api/v1/cmdb/network-devices/sync/firewalls', {}, { headers: authHeaders() })
    if (res.data?.code === 0) {
      const d = res.data.data || {}
      ElMessage.success(`同步完成：新增 ${d.created || 0}，更新 ${d.updated || 0}，跳过 ${d.skipped || 0}`)
      await fetchData()
    } else {
      ElMessage.error(res.data?.message || '同步失败')
    }
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '同步失败'))
  } finally {
    syncing.value = false
  }
}

onMounted(fetchData)
</script>

<style scoped>
.page-card {
  border-radius: 16px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
  margin-bottom: 12px;
}

.page-header h2 {
  margin: 0;
  font-size: 22px;
}

.page-desc {
  margin-top: 6px;
  color: var(--el-text-color-secondary);
}

.page-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.mb-12 {
  margin-bottom: 12px;
}

.k {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.v {
  margin-top: 6px;
  font-size: 28px;
  font-weight: 700;
}

.v.success {
  color: var(--el-color-success);
}

.table-scroll {
  overflow-x: auto;
}

.test-json {
  margin: 0;
  max-height: 360px;
  overflow: auto;
  background: rgba(15, 23, 42, 0.95);
  color: #e2e8f0;
  padding: 12px;
  border-radius: 10px;
}
</style>
