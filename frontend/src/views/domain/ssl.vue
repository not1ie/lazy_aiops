<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>域名与证书</h2>
        <p class="page-desc">证书到期与云域名到期的统一看板。</p>
      </div>
      <div class="page-actions">
        <el-button icon="Refresh" @click="refreshAll">刷新</el-button>
      </div>
    </div>

    <el-tabs v-model="activeTab">
      <el-tab-pane label="SSL证书" name="certs">
        <el-row :gutter="12" class="summary-row">
          <el-col :span="6"><el-card><div class="card-title">证书总数</div><div class="card-value">{{ certSummary.total }}</div></el-card></el-col>
          <el-col :span="6"><el-card><div class="card-title">正常</div><div class="card-value ok">{{ certSummary.ok }}</div></el-card></el-col>
          <el-col :span="6"><el-card><div class="card-title">30天内过期</div><div class="card-value warn">{{ certSummary.expiring }}</div></el-card></el-col>
          <el-col :span="6"><el-card><div class="card-title">已过期</div><div class="card-value critical">{{ certSummary.expired }}</div></el-card></el-col>
        </el-row>

        <div class="toolbar">
          <el-input v-model="certKeyword" placeholder="搜索域名/颁发者" class="w-52" clearable />
          <el-button type="primary" @click="openCreateCert">新增证书监控</el-button>
          <el-button @click="checkAllCerts">批量检查</el-button>
        </div>

        <el-table :data="filteredCerts" v-loading="certLoading" style="width: 100%">
          <el-table-column prop="domain" label="域名" min-width="220" />
          <el-table-column prop="issuer" label="颁发者" min-width="220" />
          <el-table-column label="有效期至" width="180">
            <template #default="{ row }">{{ formatTime(row.not_after) }}</template>
          </el-table-column>
          <el-table-column prop="days_to_expire" label="剩余天数" width="120" sortable />
          <el-table-column label="状态" width="120">
            <template #default="{ row }">
              <el-tag :type="certStatusTag(row.status, row.days_to_expire)">
                {{ certStatusText(row.status, row.days_to_expire) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="最后检查" width="180">
            <template #default="{ row }">{{ formatTime(row.last_check_at) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="220" fixed="right">
            <template #default="{ row }">
              <el-button size="small" @click="inspectCert(row)">详情</el-button>
              <el-button size="small" type="primary" plain @click="checkCert(row)">检查</el-button>
              <el-button size="small" type="danger" plain @click="deleteCert(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="云域名" name="domains">
        <el-row :gutter="12" class="summary-row">
          <el-col :span="6"><el-card><div class="card-title">域名总数</div><div class="card-value">{{ domainSummary.total }}</div></el-card></el-col>
          <el-col :span="6"><el-card><div class="card-title">30天内过期</div><div class="card-value warn">{{ domainSummary.expiring }}</div></el-card></el-col>
          <el-col :span="6"><el-card><div class="card-title">已过期</div><div class="card-value critical">{{ domainSummary.expired }}</div></el-card></el-col>
          <el-col :span="6"><el-card><div class="card-title">云账号</div><div class="card-value">{{ accounts.length }}</div></el-card></el-col>
        </el-row>

        <div class="toolbar">
          <el-select v-model="accountFilter" placeholder="选择云账号" class="w-52" clearable>
            <el-option v-for="item in accounts" :key="item.id" :label="item.name" :value="item.id" />
          </el-select>
          <el-input v-model="domainKeyword" placeholder="搜索域名" class="w-52" clearable />
        </div>

        <el-table :data="filteredDomains" v-loading="domainLoading" style="width: 100%">
          <el-table-column prop="domain" label="域名" min-width="240" />
          <el-table-column label="云账号" width="180">
            <template #default="{ row }">{{ row.account?.name || '-' }}</template>
          </el-table-column>
          <el-table-column prop="provider" label="厂商" width="120" />
          <el-table-column label="到期时间" width="180">
            <template #default="{ row }">{{ formatTime(row.expiration_at) }}</template>
          </el-table-column>
          <el-table-column prop="days_to_expire" label="剩余天数" width="120" sortable />
          <el-table-column label="状态" width="120">
            <template #default="{ row }">
              <el-tag :type="domainStatusTag(row.days_to_expire)">
                {{ domainStatusText(row.days_to_expire) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="140" fixed="right">
            <template #default="{ row }">
              <el-button size="small" @click="checkDomain(row)">检测</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
    </el-tabs>

    <el-dialog append-to-body v-model="createCertVisible" title="新增证书监控" width="460px">
      <el-form :model="createCertForm" label-width="80px">
        <el-form-item label="域名">
          <el-input v-model="createCertForm.domain" placeholder="example.com" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createCertVisible = false">取消</el-button>
        <el-button type="primary" @click="createCert">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog append-to-body v-model="certDetailVisible" title="证书详情" width="700px">
      <el-descriptions :column="2" border v-if="currentCert">
        <el-descriptions-item label="域名">{{ currentCert.domain }}</el-descriptions-item>
        <el-descriptions-item label="颁发者">{{ currentCert.issuer || '-' }}</el-descriptions-item>
        <el-descriptions-item label="主题">{{ currentCert.subject || '-' }}</el-descriptions-item>
        <el-descriptions-item label="序列号">{{ currentCert.serial_number || '-' }}</el-descriptions-item>
        <el-descriptions-item label="生效时间">{{ formatTime(currentCert.not_before) }}</el-descriptions-item>
        <el-descriptions-item label="到期时间">{{ formatTime(currentCert.not_after) }}</el-descriptions-item>
        <el-descriptions-item label="SANs" :span="2">{{ currentCert.sans || '-' }}</el-descriptions-item>
      </el-descriptions>
    </el-dialog>

    <el-dialog append-to-body v-model="domainCheckVisible" title="域名检测结果" width="760px">
      <el-descriptions :column="2" border v-if="domainCheckResult">
        <el-descriptions-item label="域名">{{ domainCheckResult.domain || '-' }}</el-descriptions-item>
        <el-descriptions-item label="DNS解析">{{ domainCheckResult.dns_resolved ? '正常' : '失败' }}</el-descriptions-item>
        <el-descriptions-item label="IP列表" :span="2">{{ (domainCheckResult.ips || []).join(', ') || '-' }}</el-descriptions-item>
        <el-descriptions-item label="证书颁发者">{{ domainCheckResult.ssl?.issuer || '-' }}</el-descriptions-item>
        <el-descriptions-item label="证书到期">{{ formatTime(domainCheckResult.ssl?.not_after) }}</el-descriptions-item>
        <el-descriptions-item label="剩余天数">{{ domainCheckResult.ssl?.days_to_expire ?? '-' }}</el-descriptions-item>
        <el-descriptions-item label="SANs" :span="2">{{ domainCheckResult.ssl?.sans || '-' }}</el-descriptions-item>
      </el-descriptions>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const activeTab = ref('certs')
const certLoading = ref(false)
const domainLoading = ref(false)
const certs = ref([])
const domains = ref([])
const accounts = ref([])
const certKeyword = ref('')
const domainKeyword = ref('')
const accountFilter = ref('')

const createCertVisible = ref(false)
const createCertForm = ref({ domain: '' })
const certDetailVisible = ref(false)
const currentCert = ref(null)
const domainCheckVisible = ref(false)
const domainCheckResult = ref(null)

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const formatTime = (value) => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString()
}

const certStatusText = (status, days) => {
  if (status === 0 || Number(days) <= 0) return '已过期'
  if (status === 2 || Number(days) <= 30) return '即将过期'
  return '正常'
}

const certStatusTag = (status, days) => {
  if (status === 0 || Number(days) <= 0) return 'danger'
  if (status === 2 || Number(days) <= 30) return 'warning'
  return 'success'
}

const domainStatusText = (days) => {
  if (Number(days) <= 0) return '已过期'
  if (Number(days) <= 30) return '即将过期'
  return '正常'
}

const domainStatusTag = (days) => {
  if (Number(days) <= 0) return 'danger'
  if (Number(days) <= 30) return 'warning'
  return 'success'
}

const certSummary = computed(() => {
  const total = certs.value.length
  const expired = certs.value.filter(item => Number(item.days_to_expire) <= 0 || item.status === 0).length
  const expiring = certs.value.filter(item => Number(item.days_to_expire) > 0 && Number(item.days_to_expire) <= 30).length
  const ok = total - expired - expiring
  return { total, expired, expiring, ok }
})

const domainSummary = computed(() => {
  const total = domains.value.length
  const expired = domains.value.filter(item => Number(item.days_to_expire) <= 0).length
  const expiring = domains.value.filter(item => Number(item.days_to_expire) > 0 && Number(item.days_to_expire) <= 30).length
  return { total, expired, expiring }
})

const filteredCerts = computed(() => {
  const key = certKeyword.value.trim().toLowerCase()
  if (!key) return certs.value
  return certs.value.filter(item => (
    (item.domain || '').toLowerCase().includes(key) ||
    (item.issuer || '').toLowerCase().includes(key)
  ))
})

const filteredDomains = computed(() => {
  const key = domainKeyword.value.trim().toLowerCase()
  return domains.value.filter(item => {
    if (accountFilter.value && item.account_id !== accountFilter.value) return false
    if (!key) return true
    return (item.domain || '').toLowerCase().includes(key)
  })
})

const fetchAccounts = async () => {
  const res = await axios.get('/api/v1/domain/accounts', { headers: authHeaders() })
  if (res.data?.code === 0) accounts.value = res.data.data || []
}

const fetchCerts = async () => {
  certLoading.value = true
  try {
    const res = await axios.get('/api/v1/domain/certs', { headers: authHeaders() })
    if (res.data?.code === 0) certs.value = res.data.data || []
  } finally {
    certLoading.value = false
  }
}

const fetchDomains = async () => {
  domainLoading.value = true
  try {
    const res = await axios.get('/api/v1/domain/domains', { headers: authHeaders() })
    if (res.data?.code === 0) domains.value = res.data.data || []
  } finally {
    domainLoading.value = false
  }
}

const refreshAll = async () => {
  try {
    await Promise.all([fetchAccounts(), fetchCerts(), fetchDomains()])
  } catch (err) {
    ElMessage.error('加载失败')
  }
}

const openCreateCert = () => {
  createCertForm.value.domain = ''
  createCertVisible.value = true
}

const createCert = async () => {
  const domain = createCertForm.value.domain.trim()
  if (!domain) {
    ElMessage.warning('请输入域名')
    return
  }
  try {
    await axios.post('/api/v1/domain/certs', { domain }, { headers: authHeaders() })
    ElMessage.success('新增成功')
    createCertVisible.value = false
    await fetchCerts()
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '新增失败')
  }
}

const inspectCert = (row) => {
  currentCert.value = row
  certDetailVisible.value = true
}

const checkCert = async (row) => {
  try {
    await axios.post(`/api/v1/domain/certs/${row.id}/check`, {}, { headers: authHeaders() })
    ElMessage.success('检查完成')
    await fetchCerts()
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '检查失败')
  }
}

const checkAllCerts = async () => {
  try {
    const res = await axios.post('/api/v1/domain/certs/check_all', {}, { headers: authHeaders() })
    if (res.data?.code === 0) {
      const stats = res.data.data || {}
      ElMessage.success(`检查完成: 成功${stats.success || 0}, 失败${stats.failed || 0}`)
    } else {
      ElMessage.warning('批量检查已执行')
    }
    await fetchCerts()
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '批量检查失败')
  }
}

const deleteCert = async (row) => {
  try {
    await ElMessageBox.confirm(`确认删除证书监控：${row.domain}？`, '删除确认', { type: 'warning' })
    await axios.delete(`/api/v1/domain/certs/${row.id}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    await fetchCerts()
  } catch (err) {
    if (err !== 'cancel') ElMessage.error('删除失败')
  }
}

const checkDomain = async (row) => {
  try {
    const res = await axios.post('/api/v1/domain/domains/check', { domain: row.domain }, { headers: authHeaders() })
    if (res.data?.code === 0) {
      domainCheckResult.value = res.data.data
      domainCheckVisible.value = true
    }
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '检测失败')
  }
}

onMounted(refreshAll)
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 16px; }
.page-desc { color: #606266; margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; }
.summary-row { margin-bottom: 12px; }
.toolbar { display: flex; gap: 8px; flex-wrap: wrap; margin-bottom: 12px; }
.card-title { color: #909399; font-size: 12px; }
.card-value { font-size: 20px; font-weight: 600; margin-top: 6px; }
.ok { color: #67C23A; }
.warn { color: #E6A23C; }
.critical { color: #F56C6C; }
.w-52 { width: 220px; }
</style>
