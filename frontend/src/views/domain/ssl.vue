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

        <el-table :fit="true" :data="filteredCerts" v-loading="certLoading" style="width: 100%">
          <el-table-column prop="domain" label="域名" min-width="220" />
          <el-table-column prop="issuer" label="颁发者" min-width="220" />
          <el-table-column label="有效期至" width="180">
            <template #default="{ row }">{{ formatTime(row.not_after) }}</template>
          </el-table-column>
          <el-table-column label="剩余天数" width="120" sortable :sort-method="sortCertDays">
            <template #default="{ row }">{{ certDays(row) }}</template>
          </el-table-column>
          <el-table-column label="状态" width="120">
            <template #default="{ row }">
              <el-tag :type="certStatusTag(row.status, certDays(row))">
                {{ certStatusText(row.status, certDays(row)) }}
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
          <el-col :span="4"><el-card><div class="card-title">域名总数</div><div class="card-value">{{ domainSummary.total }}</div></el-card></el-col>
          <el-col :span="4"><el-card><div class="card-title">健康</div><div class="card-value ok">{{ domainSummary.healthy }}</div></el-card></el-col>
          <el-col :span="4"><el-card><div class="card-title">风险</div><div class="card-value warn">{{ domainSummary.warning }}</div></el-card></el-col>
          <el-col :span="4"><el-card><div class="card-title">故障</div><div class="card-value critical">{{ domainSummary.critical }}</div></el-card></el-col>
          <el-col :span="4"><el-card><div class="card-title">30天内过期</div><div class="card-value warn">{{ domainSummary.expiring }}</div></el-card></el-col>
          <el-col :span="4"><el-card><div class="card-title">云账号</div><div class="card-value">{{ accounts.length }}</div></el-card></el-col>
        </el-row>

        <div class="toolbar">
          <el-select v-model="accountFilter" placeholder="选择云账号" class="w-52" clearable>
            <el-option v-for="item in accounts" :key="item.id" :label="item.name" :value="item.id" />
          </el-select>
          <el-input v-model="domainKeyword" placeholder="搜索域名" class="w-52" clearable />
          <el-button type="primary" @click="checkAllDomains">一键体检</el-button>
        </div>

        <el-table :fit="true" :data="filteredDomains" v-loading="domainLoading" style="width: 100%">
          <el-table-column prop="domain" label="域名" min-width="240" />
          <el-table-column label="云账号" width="180">
            <template #default="{ row }">{{ row.account?.name || '-' }}</template>
          </el-table-column>
          <el-table-column prop="provider" label="厂商" width="120" />
          <el-table-column label="到期时间" width="180">
            <template #default="{ row }">{{ formatTime(row.expiration_at) }}</template>
          </el-table-column>
          <el-table-column label="剩余天数" width="120" sortable :sort-method="sortDomainDays">
            <template #default="{ row }">{{ domainDays(row) }}</template>
          </el-table-column>
          <el-table-column label="状态" width="120">
            <template #default="{ row }">
              <el-tag :type="domainStatusTag(domainDays(row))">
                {{ domainStatusText(domainDays(row)) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="DNS" width="100">
            <template #default="{ row }">
              <el-tag :type="row.dns_resolved ? 'success' : 'danger'">{{ row.dns_resolved ? '正常' : '失败' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="HTTP" width="100">
            <template #default="{ row }">{{ row.http_status_code || '-' }}</template>
          </el-table-column>
          <el-table-column label="响应(ms)" width="110">
            <template #default="{ row }">{{ row.response_time_ms || '-' }}</template>
          </el-table-column>
          <el-table-column label="SSL剩余天" width="120">
            <template #default="{ row }">{{ row.ssl_days_to_expire || '-' }}</template>
          </el-table-column>
          <el-table-column label="健康状态" width="120">
            <template #default="{ row }">
              <el-tag :type="healthTag(row.health_status)">{{ healthText(row.health_status) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="最后体检" width="180">
            <template #default="{ row }">{{ formatTime(row.last_check_at) }}</template>
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
        <el-descriptions-item label="健康状态">
          <el-tag :type="healthTag(domainCheckResult.health_status)">{{ healthText(domainCheckResult.health_status) }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="DNS解析">{{ domainCheckResult.dns_resolved ? '正常' : '失败' }}</el-descriptions-item>
        <el-descriptions-item label="HTTP状态码">{{ domainCheckResult.http_status_code || '-' }}</el-descriptions-item>
        <el-descriptions-item label="响应耗时">{{ domainCheckResult.response_time_ms ? `${domainCheckResult.response_time_ms} ms` : '-' }}</el-descriptions-item>
        <el-descriptions-item label="IP列表" :span="2">{{ (domainCheckResult.ips || []).join(', ') || '-' }}</el-descriptions-item>
        <el-descriptions-item label="证书颁发者">{{ domainCheckResult.ssl?.issuer || '-' }}</el-descriptions-item>
        <el-descriptions-item label="证书到期">{{ formatTime(domainCheckResult.ssl?.not_after) }}</el-descriptions-item>
        <el-descriptions-item label="剩余天数">{{ certDays(domainCheckResult.ssl) }}</el-descriptions-item>
        <el-descriptions-item label="SANs" :span="2">{{ domainCheckResult.ssl?.sans || '-' }}</el-descriptions-item>
      </el-descriptions>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
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
const nowTick = ref(Date.now())
let dayTicker = null

const createCertVisible = ref(false)
const createCertForm = ref({ domain: '' })
const certDetailVisible = ref(false)
const currentCert = ref(null)
const domainCheckVisible = ref(false)
const domainCheckResult = ref(null)

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const getErrorMessage = (err, fallback) => {
  if (err?.response?.data?.message) return err.response.data.message
  if (err?.message) return err.message
  return fallback
}

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

const healthText = (status) => {
  if (status === 'healthy') return '健康'
  if (status === 'warning') return '风险'
  if (status === 'critical') return '故障'
  return '未知'
}

const healthTag = (status) => {
  if (status === 'healthy') return 'success'
  if (status === 'warning') return 'warning'
  if (status === 'critical') return 'danger'
  return 'info'
}

const certSummary = computed(() => {
  const total = certs.value.length
  const expired = certs.value.filter(item => certDays(item) <= 0 || item.status === 0).length
  const expiring = certs.value.filter(item => certDays(item) > 0 && certDays(item) <= 30).length
  const ok = total - expired - expiring
  return { total, expired, expiring, ok }
})

const domainSummary = computed(() => {
  const total = domains.value.length
  const expired = domains.value.filter(item => domainDays(item) <= 0).length
  const expiring = domains.value.filter(item => domainDays(item) > 0 && domainDays(item) <= 30).length
  const healthy = domains.value.filter(item => item.health_status === 'healthy').length
  const warning = domains.value.filter(item => item.health_status === 'warning').length
  const critical = domains.value.filter(item => item.health_status === 'critical').length
  return { total, expired, expiring, healthy, warning, critical }
})

const calcDaysToExpire = (expireValue) => {
  const current = nowTick.value
  if (!expireValue) return 0
  const expireAt = new Date(expireValue)
  if (Number.isNaN(expireAt.getTime())) return 0
  const remainMs = expireAt.getTime() - current
  if (remainMs <= 0) return 0
  return Math.ceil(remainMs / (24 * 60 * 60 * 1000))
}

const certDays = (row) => {
  if (!row?.not_after) return Number(row?.days_to_expire || 0)
  return calcDaysToExpire(row.not_after)
}

const domainDays = (row) => {
  if (!row?.expiration_at) return Number(row?.days_to_expire || 0)
  return calcDaysToExpire(row.expiration_at)
}
const sortCertDays = (a, b) => certDays(a) - certDays(b)
const sortDomainDays = (a, b) => domainDays(a) - domainDays(b)

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
    ElMessage.error(getErrorMessage(err, '加载失败'))
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
    ElMessage.error(getErrorMessage(err, '新增失败'))
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
    ElMessage.error(getErrorMessage(err, '检查失败'))
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
    ElMessage.error(getErrorMessage(err, '批量检查失败'))
  }
}

const deleteCert = async (row) => {
  try {
    await ElMessageBox.confirm(`确认删除证书监控：${row.domain}？`, '删除确认', { type: 'warning' })
    await axios.delete(`/api/v1/domain/certs/${row.id}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    await fetchCerts()
  } catch (err) {
    if (err !== 'cancel' && err !== 'close') {
      ElMessage.error(getErrorMessage(err, '删除失败'))
    }
  }
}

const checkDomain = async (row) => {
  try {
    const res = await axios.post('/api/v1/domain/domains/check', { domain: row.domain }, { headers: authHeaders() })
    if (res.data?.code === 0) {
      domainCheckResult.value = res.data.data
      domainCheckVisible.value = true
      await fetchDomains()
    }
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '检测失败'))
  }
}

const checkAllDomains = async () => {
  try {
    const res = await axios.post('/api/v1/domain/domains/check_all', {}, { headers: authHeaders() })
    if (res.data?.code === 0) {
      const stats = res.data.data || {}
      ElMessage.success(`体检完成: 成功${stats.success || 0}, 失败${stats.failed || 0}`)
    }
    await fetchDomains()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '批量体检失败'))
  }
}

onMounted(() => {
  refreshAll()
  dayTicker = window.setInterval(() => {
    nowTick.value = Date.now()
  }, 60 * 1000)
})

onBeforeUnmount(() => {
  if (dayTicker) window.clearInterval(dayTicker)
})
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
