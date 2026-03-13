<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>成本概览</h2>
        <p class="page-desc">多云成本趋势、账号管理与费用明细。</p>
      </div>
      <div class="page-actions">
        <el-date-picker
          v-model="dateRange"
          type="daterange"
          value-format="YYYY-MM-DD"
          range-separator="至"
          start-placeholder="开始日期"
          end-placeholder="结束日期"
          unlink-panels
          class="w-72"
        />
        <el-select v-model="accountFilter" placeholder="全部账号" clearable filterable class="w-52">
          <el-option v-for="acc in accounts" :key="acc.id" :label="acc.name" :value="acc.id" />
        </el-select>
        <el-button type="primary" icon="Search" @click="reloadAll">查询</el-button>
        <el-button icon="Refresh" @click="reloadAll">刷新</el-button>
      </div>
    </div>

    <el-row :gutter="12" class="summary-row">
      <el-col :xs="24" :sm="12" :lg="6">
        <el-card shadow="never" class="metric-card">
          <div class="metric-title">统计区间总成本</div>
          <div class="metric-value">¥{{ formatCurrency(summaryTotal) }}</div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :lg="6">
        <el-card shadow="never" class="metric-card">
          <div class="metric-title">本月成本</div>
          <div class="metric-value">¥{{ formatCurrency(monthCost) }}</div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :lg="6">
        <el-card shadow="never" class="metric-card">
          <div class="metric-title">云账号数</div>
          <div class="metric-value">{{ accounts.length }}</div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :lg="6">
        <el-card shadow="never" class="metric-card">
          <div class="metric-title">Top 产品</div>
          <div class="metric-value ellipsis">{{ topProductName }}</div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="12" class="chart-row">
      <el-col :xs="24" :lg="14">
        <el-card shadow="never">
          <template #header>
            <div class="card-title">成本日趋势</div>
          </template>
          <div ref="trendChartRef" class="chart-box"></div>
        </el-card>
      </el-col>
      <el-col :xs="24" :lg="10">
        <el-card shadow="never">
          <template #header>
            <div class="card-title">产品成本占比</div>
          </template>
          <div ref="productChartRef" class="chart-box"></div>
        </el-card>
      </el-col>
    </el-row>

    <el-card shadow="never" class="section-card">
      <template #header>
        <div class="section-header">
          <span>云账号管理</span>
          <div class="page-actions">
            <el-button type="primary" icon="Plus" @click="openCreateAccount">新增账号</el-button>
            <el-button icon="Refresh" @click="fetchAccounts">刷新</el-button>
          </div>
        </div>
      </template>
      <div class="table-scroll">
        <el-table :fit="true" :data="accounts" v-loading="accountLoading" stripe style="min-width: 1120px">
        <el-table-column prop="name" label="账号" min-width="140" />
        <el-table-column prop="provider" label="云厂商" width="120" />
        <el-table-column prop="region" label="Region" width="140" />
        <el-table-column prop="description" label="描述" min-width="220" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'info'">{{ row.status === 1 ? '启用' : '停用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="290" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="openEditAccount(row)">编辑</el-button>
            <el-button size="small" type="primary" plain @click="syncAccount(row)">同步费用</el-button>
            <el-button size="small" type="danger" @click="removeAccount(row)">删除</el-button>
          </template>
        </el-table-column>
        </el-table>
      </div>
    </el-card>

    <el-row :gutter="12" class="table-row">
      <el-col :xs="24" :lg="12">
        <el-card shadow="never" class="section-card">
          <template #header>
            <div class="card-title">成本 Top 资源</div>
          </template>
          <div class="table-scroll">
            <el-table :fit="true" :data="topResources" v-loading="resourceLoading" stripe height="320" style="min-width: 720px">
            <el-table-column prop="resource_name" label="资源" min-width="170" show-overflow-tooltip />
            <el-table-column prop="product_name" label="产品" min-width="130" show-overflow-tooltip />
            <el-table-column prop="amount" label="成本" width="120">
              <template #default="{ row }">¥{{ formatCurrency(row.amount) }}</template>
            </el-table-column>
            </el-table>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :lg="12">
        <el-card shadow="never" class="section-card">
          <template #header>
            <div class="card-title">最近费用记录（最多200条）</div>
          </template>
          <div class="table-scroll">
            <el-table :fit="true" :data="records" v-loading="recordLoading" stripe height="320" style="min-width: 840px">
            <el-table-column prop="billing_date" label="账期" width="120">
              <template #default="{ row }">{{ formatDay(row.billing_date) }}</template>
            </el-table-column>
            <el-table-column prop="account_name" label="账号" min-width="120" show-overflow-tooltip />
            <el-table-column prop="product_name" label="产品" min-width="120" show-overflow-tooltip />
            <el-table-column prop="resource_name" label="资源" min-width="160" show-overflow-tooltip />
            <el-table-column prop="amount" label="金额" width="110">
              <template #default="{ row }">¥{{ formatCurrency(row.amount) }}</template>
            </el-table-column>
            </el-table>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-dialog append-to-body v-model="accountDialogVisible" :title="accountDialogTitle" width="680px">
      <el-form :model="accountForm" label-width="110px">
        <el-form-item label="账号名称">
          <el-input v-model="accountForm.name" placeholder="例如：阿里云生产账号" />
        </el-form-item>
        <el-form-item label="云厂商">
          <el-select v-model="accountForm.provider" class="w-52">
            <el-option label="aliyun" value="aliyun" />
            <el-option label="tencent" value="tencent" />
            <el-option label="aws" value="aws" />
            <el-option label="huawei" value="huawei" />
            <el-option label="gcp" value="gcp" />
            <el-option label="azure" value="azure" />
          </el-select>
        </el-form-item>
        <el-form-item label="AccessKey">
          <el-input v-model="accountForm.access_key" />
        </el-form-item>
        <el-form-item label="SecretKey">
          <el-input v-model="accountForm.secret_key" type="password" show-password />
        </el-form-item>
        <el-form-item label="Region">
          <el-input v-model="accountForm.region" placeholder="例如：cn-hangzhou" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="accountForm.description" type="textarea" :rows="2" />
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="accountEnabled" active-text="启用" inactive-text="停用" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="accountDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitAccount">保存</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import axios from 'axios'
import * as echarts from 'echarts'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getErrorMessage, isCancelError } from '@/utils/error'

const accounts = ref([])
const dateRange = ref([])
const accountFilter = ref('')

const summary = ref({ total: 0, by_product: [], daily_trend: [] })
const topResources = ref([])
const records = ref([])

const accountLoading = ref(false)
const summaryLoading = ref(false)
const resourceLoading = ref(false)
const recordLoading = ref(false)

const trendChartRef = ref(null)
const productChartRef = ref(null)
let trendChart = null
let productChart = null

const accountDialogVisible = ref(false)
const accountDialogTitle = ref('新增云账号')
const accountEditId = ref('')
const accountForm = ref({
  name: '',
  provider: 'aliyun',
  access_key: '',
  secret_key: '',
  region: '',
  description: '',
  status: 1
})

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const accountEnabled = computed({
  get: () => accountForm.value.status === 1,
  set: (val) => {
    accountForm.value.status = val ? 1 : 0
  }
})

const summaryTotal = computed(() => Number(summary.value.total || 0))
const monthCost = computed(() => {
  const currentMonth = new Date().toISOString().slice(0, 7)
  return (summary.value.daily_trend || [])
    .filter(item => String(item.date || '').startsWith(currentMonth))
    .reduce((acc, item) => acc + Number(item.amount || 0), 0)
})
const topProductName = computed(() => summary.value.by_product?.[0]?.product_name || '暂无')

const formatCurrency = (v) => Number(v || 0).toFixed(2)

const formatDay = (v) => {
  if (!v) return '-'
  return String(v).slice(0, 10)
}

const buildQuery = () => {
  const params = {}
  if (dateRange.value?.length === 2) {
    params.start_date = dateRange.value[0]
    params.end_date = dateRange.value[1]
  }
  if (accountFilter.value) params.account_id = accountFilter.value
  return params
}

const ensureCharts = () => {
  if (trendChartRef.value && !trendChart) trendChart = echarts.init(trendChartRef.value)
  if (productChartRef.value && !productChart) productChart = echarts.init(productChartRef.value)
}

const renderCharts = () => {
  ensureCharts()
  if (!trendChart || !productChart) return

  const daily = summary.value.daily_trend || []
  trendChart.setOption({
    tooltip: { trigger: 'axis' },
    grid: { left: 42, right: 16, top: 20, bottom: 36 },
    xAxis: {
      type: 'category',
      data: daily.map(item => String(item.date || '').slice(5, 10))
    },
    yAxis: {
      type: 'value',
      axisLabel: { formatter: '{value}' }
    },
    series: [
      {
        type: 'line',
        smooth: true,
        showSymbol: false,
        areaStyle: { opacity: 0.15 },
        data: daily.map(item => Number(item.amount || 0))
      }
    ]
  })

  const products = (summary.value.by_product || []).slice(0, 8)
  productChart.setOption({
    tooltip: { trigger: 'item' },
    legend: { bottom: 0 },
    series: [
      {
        type: 'pie',
        radius: ['45%', '70%'],
        avoidLabelOverlap: true,
        data: products.map(item => ({ name: item.product_name || item.product_code || 'unknown', value: Number(item.amount || 0) }))
      }
    ]
  })
}

const fetchAccounts = async () => {
  accountLoading.value = true
  try {
    const res = await axios.get('/api/v1/cost/accounts', { headers: authHeaders() })
    accounts.value = res.data?.data || []
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '获取账号失败'))
  } finally {
    accountLoading.value = false
  }
}

const fetchSummary = async () => {
  summaryLoading.value = true
  try {
    const res = await axios.get('/api/v1/cost/summary', { headers: authHeaders(), params: buildQuery() })
    summary.value = res.data?.data || { total: 0, by_product: [], daily_trend: [] }
    nextTick(renderCharts)
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '获取汇总失败'))
  } finally {
    summaryLoading.value = false
  }
}

const fetchTopResources = async () => {
  resourceLoading.value = true
  try {
    const res = await axios.get('/api/v1/cost/top-resources', { headers: authHeaders(), params: buildQuery() })
    topResources.value = res.data?.data || []
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '获取Top资源失败'))
  } finally {
    resourceLoading.value = false
  }
}

const fetchRecords = async () => {
  recordLoading.value = true
  try {
    const res = await axios.get('/api/v1/cost/records', { headers: authHeaders(), params: buildQuery() })
    records.value = res.data?.data || []
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '获取费用记录失败'))
  } finally {
    recordLoading.value = false
  }
}

const reloadAll = async () => {
  await Promise.all([fetchSummary(), fetchTopResources(), fetchRecords()])
}

const resetAccountForm = () => {
  accountEditId.value = ''
  accountForm.value = {
    name: '',
    provider: 'aliyun',
    access_key: '',
    secret_key: '',
    region: '',
    description: '',
    status: 1
  }
}

const openCreateAccount = () => {
  accountDialogTitle.value = '新增云账号'
  resetAccountForm()
  accountDialogVisible.value = true
}

const openEditAccount = (row) => {
  accountDialogTitle.value = '编辑云账号'
  accountEditId.value = row.id
  accountForm.value = {
    name: row.name || '',
    provider: row.provider || 'aliyun',
    access_key: row.access_key || '',
    secret_key: row.secret_key || '',
    region: row.region || '',
    description: row.description || '',
    status: row.status === 1 ? 1 : 0
  }
  accountDialogVisible.value = true
}

const submitAccount = async () => {
  if (!accountForm.value.name.trim()) {
    ElMessage.warning('请填写账号名称')
    return
  }
  try {
    if (accountEditId.value) {
      await axios.put(`/api/v1/cost/accounts/${accountEditId.value}`, accountForm.value, { headers: authHeaders() })
      ElMessage.success('更新成功')
    } else {
      await axios.post('/api/v1/cost/accounts', accountForm.value, { headers: authHeaders() })
      ElMessage.success('创建成功')
    }
    accountDialogVisible.value = false
    await fetchAccounts()
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '保存失败'))
  }
}

const removeAccount = async (row) => {
  try {
    await ElMessageBox.confirm(`确认删除云账号 ${row.name} 吗？`, '提示', { type: 'warning' })
    await axios.delete(`/api/v1/cost/accounts/${row.id}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    if (accountFilter.value === row.id) accountFilter.value = ''
    await fetchAccounts()
    await reloadAll()
  } catch (err) {
    if (!isCancelError(err)) ElMessage.error(getErrorMessage(err, '删除失败'))
  }
}

const syncAccount = async (row) => {
  try {
    const payload = {}
    if (dateRange.value?.length === 2) {
      payload.start_date = dateRange.value[0]
      payload.end_date = dateRange.value[1]
    }
    await axios.post(`/api/v1/cost/accounts/${row.id}/sync`, payload, { headers: authHeaders() })
    ElMessage.success('同步任务已提交')
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '同步失败'))
  }
}

const handleResize = () => {
  if (trendChart) trendChart.resize()
  if (productChart) productChart.resize()
}

onMounted(async () => {
  const now = new Date()
  const start = new Date(now.getFullYear(), now.getMonth(), 1)
  dateRange.value = [start.toISOString().slice(0, 10), now.toISOString().slice(0, 10)]
  await fetchAccounts()
  await reloadAll()
  window.addEventListener('resize', handleResize)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', handleResize)
  if (trendChart) trendChart.dispose()
  if (productChart) productChart.dispose()
  trendChart = null
  productChart = null
})
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; gap: 12px; margin-bottom: 12px; }
.page-desc { margin: 4px 0 0; color: #606266; }
.page-actions { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.summary-row,
.chart-row,
.table-row { margin-bottom: 12px; }
.metric-card { min-height: 90px; }
.metric-title { color: #909399; font-size: 13px; margin-bottom: 8px; }
.metric-value { font-size: 26px; font-weight: 700; color: #303133; line-height: 1.2; }
.section-card { margin-bottom: 12px; }
.section-header { display: flex; justify-content: space-between; align-items: center; gap: 8px; }
.card-title { font-weight: 600; }
.chart-box { height: 300px; width: 100%; }
.ellipsis { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.w-52 { width: 220px; }
.w-72 { width: 300px; }

@media (max-width: 992px) {
  .chart-box { height: 260px; }
}
</style>
