<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>预算与告警</h2>
        <p class="page-desc">预算执行状态、超支告警、成本优化建议闭环。</p>
      </div>
      <div class="page-actions">
        <el-button type="primary" icon="Plus" @click="openCreateBudget">新增预算</el-button>
        <el-button icon="Histogram" @click="runOptimizationAnalyze">分析优化建议</el-button>
        <el-button icon="Refresh" @click="reloadAll">刷新</el-button>
      </div>
    </div>

    <el-row :gutter="12" class="status-row">
      <el-col v-for="item in budgetStatus" :key="item.budget_id" :xs="24" :sm="12" :lg="8">
        <el-card shadow="never" class="status-card" :class="item.is_alert ? 'alert-card' : ''">
          <div class="status-title">{{ item.budget_name }}</div>
          <div class="status-values">
            <span>¥{{ formatCurrency(item.current_cost) }}</span>
            <span>/</span>
            <span>¥{{ formatCurrency(item.budget_amount) }}</span>
          </div>
          <el-progress :percentage="Math.min(Number(item.percentage || 0), 100)" :status="item.is_alert ? 'exception' : ''" />
          <div class="status-foot">
            <span>阈值 {{ Number(item.alert_at || 0).toFixed(0) }}%</span>
            <el-tag size="small" :type="item.is_alert ? 'danger' : 'success'">{{ item.is_alert ? '超阈值' : '正常' }}</el-tag>
          </div>
        </el-card>
      </el-col>
      <el-col v-if="budgetStatus.length === 0" :span="24">
        <el-empty description="暂无预算状态数据" />
      </el-col>
    </el-row>

    <el-card shadow="never" class="section-card">
      <template #header>
        <div class="section-header">
          <span>预算规则</span>
          <el-button type="primary" plain icon="Plus" @click="openCreateBudget">新增预算</el-button>
        </div>
      </template>
      <el-table :fit="true" :data="budgets" v-loading="budgetLoading" stripe>
        <el-table-column prop="name" label="名称" min-width="160" />
        <el-table-column prop="budget_type" label="类型" width="120" />
        <el-table-column prop="amount" label="预算金额" width="130">
          <template #default="{ row }">¥{{ formatCurrency(row.amount) }}</template>
        </el-table-column>
        <el-table-column prop="alert_at" label="告警阈值" width="110">
          <template #default="{ row }">{{ Number(row.alert_at || 0).toFixed(0) }}%</template>
        </el-table-column>
        <el-table-column label="时间区间" min-width="220">
          <template #default="{ row }">
            {{ formatDay(row.start_date) }} ~ {{ formatDay(row.end_date) }}
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'info'">{{ row.status === 1 ? '启用' : '停用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="openEditBudget(row)">编辑</el-button>
            <el-button size="small" type="danger" @click="removeBudget(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-row :gutter="12" class="table-row">
      <el-col :xs="24" :lg="12">
        <el-card shadow="never" class="section-card">
          <template #header>
            <div class="card-title">费用告警</div>
          </template>
          <el-table :fit="true" :data="alerts" v-loading="alertLoading" stripe height="320">
            <el-table-column prop="budget_name" label="预算" min-width="130" show-overflow-tooltip />
            <el-table-column prop="message" label="内容" min-width="180" show-overflow-tooltip />
            <el-table-column prop="percentage" label="占比" width="90">
              <template #default="{ row }">{{ Number(row.percentage || 0).toFixed(1) }}%</template>
            </el-table-column>
            <el-table-column prop="alert_at" label="时间" width="120">
              <template #default="{ row }">{{ formatDay(row.alert_at) }}</template>
            </el-table-column>
            <el-table-column label="状态" width="90">
              <template #default="{ row }">
                <el-tag :type="row.status === 1 ? 'success' : 'danger'">{{ row.status === 1 ? '已确认' : '未确认' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="100">
              <template #default="{ row }">
                <el-button v-if="row.status !== 1" size="small" type="primary" link @click="ackAlert(row)">确认</el-button>
                <span v-else>-</span>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
      <el-col :xs="24" :lg="12">
        <el-card shadow="never" class="section-card">
          <template #header>
            <div class="card-title">优化建议</div>
          </template>
          <el-table :fit="true" :data="optimizations" v-loading="optimizationLoading" stripe height="320">
            <el-table-column prop="resource_name" label="资源" min-width="150" show-overflow-tooltip />
            <el-table-column prop="resource_type" label="类型" width="110" />
            <el-table-column prop="opt_type" label="建议" width="100" />
            <el-table-column prop="save_amount" label="预计节省" width="110">
              <template #default="{ row }">¥{{ formatCurrency(row.save_amount) }}</template>
            </el-table-column>
            <el-table-column label="状态" width="100">
              <template #default="{ row }">
                <el-tag :type="optimizationStatusType(row.status)">{{ optimizationStatusText(row.status) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="140">
              <template #default="{ row }">
                <el-dropdown trigger="click" @command="(cmd) => updateOptimization(row, cmd)">
                  <el-button size="small">处理</el-button>
                  <template #dropdown>
                    <el-dropdown-menu>
                      <el-dropdown-item :disabled="row.status === 1" command="1">标记已采纳</el-dropdown-item>
                      <el-dropdown-item :disabled="row.status === 2" command="2">标记已忽略</el-dropdown-item>
                      <el-dropdown-item :disabled="row.status === 0" command="0">重置为待处理</el-dropdown-item>
                    </el-dropdown-menu>
                  </template>
                </el-dropdown>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <el-dialog append-to-body v-model="budgetDialogVisible" :title="budgetDialogTitle" width="720px">
      <el-form :model="budgetForm" label-width="110px">
        <el-form-item label="预算名称">
          <el-input v-model="budgetForm.name" placeholder="例如：生产环境月度预算" />
        </el-form-item>
        <el-form-item label="云账号">
          <el-select v-model="budgetForm.account_id" class="w-52" clearable placeholder="可选">
            <el-option v-for="acc in accounts" :key="acc.id" :label="acc.name" :value="acc.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="产品编码">
          <el-input v-model="budgetForm.product_code" placeholder="可选，例如 ecs/rds" />
        </el-form-item>
        <el-form-item label="预算类型">
          <el-select v-model="budgetForm.budget_type" class="w-52">
            <el-option label="monthly" value="monthly" />
            <el-option label="quarterly" value="quarterly" />
            <el-option label="yearly" value="yearly" />
          </el-select>
        </el-form-item>
        <el-form-item label="预算金额">
          <el-input-number v-model="budgetForm.amount" :min="0" :precision="2" :step="100" class="w-52" />
        </el-form-item>
        <el-form-item label="告警阈值(%)">
          <el-input-number v-model="budgetForm.alert_at" :min="1" :max="100" :step="5" class="w-52" />
        </el-form-item>
        <el-form-item label="预算周期">
          <el-date-picker
            v-model="budgetDateRange"
            type="daterange"
            range-separator="至"
            start-placeholder="开始日期"
            end-placeholder="结束日期"
            unlink-panels
            class="w-72"
          />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="budgetEnabled" active-text="启用" inactive-text="停用" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="budgetDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitBudget">保存</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const accounts = ref([])
const budgets = ref([])
const budgetStatus = ref([])
const alerts = ref([])
const optimizations = ref([])

const budgetLoading = ref(false)
const alertLoading = ref(false)
const optimizationLoading = ref(false)

const budgetDialogVisible = ref(false)
const budgetDialogTitle = ref('新增预算')
const budgetEditId = ref('')
const budgetForm = ref({
  name: '',
  account_id: '',
  product_code: '',
  budget_type: 'monthly',
  amount: 0,
  alert_at: 80,
  start_date: '',
  end_date: '',
  status: 1
})
const budgetDateRange = ref([])

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const budgetEnabled = computed({
  get: () => budgetForm.value.status === 1,
  set: (val) => {
    budgetForm.value.status = val ? 1 : 0
  }
})

const formatCurrency = (v) => Number(v || 0).toFixed(2)
const formatDay = (v) => (v ? String(v).slice(0, 10) : '-')

const optimizationStatusText = (status) => {
  if (status === 1) return '已采纳'
  if (status === 2) return '已忽略'
  return '待处理'
}

const optimizationStatusType = (status) => {
  if (status === 1) return 'success'
  if (status === 2) return 'info'
  return 'warning'
}

const fetchAccounts = async () => {
  try {
    const res = await axios.get('/api/v1/cost/accounts', { headers: authHeaders() })
    accounts.value = res.data?.data || []
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '获取账号失败')
  }
}

const fetchBudgets = async () => {
  budgetLoading.value = true
  try {
    const [listRes, statusRes] = await Promise.all([
      axios.get('/api/v1/cost/budgets', { headers: authHeaders() }),
      axios.get('/api/v1/cost/budgets/status', { headers: authHeaders() })
    ])
    budgets.value = listRes.data?.data || []
    budgetStatus.value = statusRes.data?.data || []
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '获取预算失败')
  } finally {
    budgetLoading.value = false
  }
}

const fetchAlerts = async () => {
  alertLoading.value = true
  try {
    const res = await axios.get('/api/v1/cost/alerts', { headers: authHeaders() })
    alerts.value = res.data?.data || []
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '获取告警失败')
  } finally {
    alertLoading.value = false
  }
}

const fetchOptimizations = async () => {
  optimizationLoading.value = true
  try {
    const res = await axios.get('/api/v1/cost/optimizations', { headers: authHeaders() })
    optimizations.value = res.data?.data || []
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '获取优化建议失败')
  } finally {
    optimizationLoading.value = false
  }
}

const reloadAll = async () => {
  await Promise.all([fetchAccounts(), fetchBudgets(), fetchAlerts(), fetchOptimizations()])
}

const resetBudgetForm = () => {
  budgetEditId.value = ''
  budgetForm.value = {
    name: '',
    account_id: '',
    product_code: '',
    budget_type: 'monthly',
    amount: 0,
    alert_at: 80,
    start_date: '',
    end_date: '',
    status: 1
  }
  budgetDateRange.value = []
}

const openCreateBudget = () => {
  resetBudgetForm()
  budgetDialogTitle.value = '新增预算'
  budgetDialogVisible.value = true
}

const openEditBudget = (row) => {
  budgetEditId.value = row.id
  budgetDialogTitle.value = '编辑预算'
  budgetForm.value = {
    name: row.name || '',
    account_id: row.account_id || '',
    product_code: row.product_code || '',
    budget_type: row.budget_type || 'monthly',
    amount: Number(row.amount || 0),
    alert_at: Number(row.alert_at || 80),
    start_date: row.start_date || '',
    end_date: row.end_date || '',
    status: row.status === 1 ? 1 : 0
  }
  budgetDateRange.value = [row.start_date ? new Date(row.start_date) : null, row.end_date ? new Date(row.end_date) : null].filter(Boolean)
  budgetDialogVisible.value = true
}

const submitBudget = async () => {
  if (!budgetForm.value.name.trim()) {
    ElMessage.warning('请填写预算名称')
    return
  }
  if (!budgetDateRange.value || budgetDateRange.value.length !== 2) {
    ElMessage.warning('请选择预算周期')
    return
  }

  const payload = {
    ...budgetForm.value,
    start_date: budgetDateRange.value[0],
    end_date: budgetDateRange.value[1]
  }

  try {
    if (budgetEditId.value) {
      await axios.put(`/api/v1/cost/budgets/${budgetEditId.value}`, payload, { headers: authHeaders() })
      ElMessage.success('更新成功')
    } else {
      await axios.post('/api/v1/cost/budgets', payload, { headers: authHeaders() })
      ElMessage.success('创建成功')
    }
    budgetDialogVisible.value = false
    await fetchBudgets()
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '保存失败')
  }
}

const removeBudget = async (row) => {
  try {
    await ElMessageBox.confirm(`确认删除预算 ${row.name} 吗？`, '提示', { type: 'warning' })
    await axios.delete(`/api/v1/cost/budgets/${row.id}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    await fetchBudgets()
  } catch (err) {
    if (err !== 'cancel') ElMessage.error(err.response?.data?.message || '删除失败')
  }
}

const ackAlert = async (row) => {
  try {
    await axios.post(`/api/v1/cost/alerts/${row.id}/ack`, {}, { headers: authHeaders() })
    ElMessage.success('已确认')
    await fetchAlerts()
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '确认失败')
  }
}

const updateOptimization = async (row, status) => {
  const target = Number(status)
  try {
    await axios.put(`/api/v1/cost/optimizations/${row.id}`, { status: target }, { headers: authHeaders() })
    ElMessage.success('更新成功')
    await fetchOptimizations()
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '更新失败')
  }
}

const runOptimizationAnalyze = async () => {
  try {
    await axios.post('/api/v1/cost/optimizations/analyze', {}, { headers: authHeaders() })
    ElMessage.success('分析任务已提交')
    await fetchOptimizations()
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '分析失败')
  }
}

onMounted(reloadAll)
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; gap: 12px; margin-bottom: 12px; }
.page-desc { margin: 4px 0 0; color: #606266; }
.page-actions { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
.status-row,
.table-row { margin-bottom: 12px; }
.status-card { margin-bottom: 12px; }
.alert-card { border-color: #f56c6c; }
.status-title { font-weight: 600; margin-bottom: 8px; }
.status-values { display: flex; gap: 6px; align-items: baseline; font-size: 18px; font-weight: 700; margin-bottom: 8px; }
.status-foot { margin-top: 8px; display: flex; justify-content: space-between; align-items: center; color: #909399; font-size: 12px; }
.section-card { margin-bottom: 12px; }
.section-header { display: flex; justify-content: space-between; align-items: center; }
.card-title { font-weight: 600; }
.w-52 { width: 220px; }
.w-72 { width: 300px; }
</style>
