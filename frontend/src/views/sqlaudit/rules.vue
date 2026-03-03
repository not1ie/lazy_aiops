<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>SQL 审核规则</h2>
        <p class="page-desc">管理 SQL 安全与性能规则，支持在线分析 SQL 风险。</p>
      </div>
      <div class="page-actions">
        <el-button type="primary" icon="Plus" @click="openCreate">新增规则</el-button>
        <el-button icon="Refresh" @click="reloadAll">刷新</el-button>
      </div>
    </div>

    <el-row :gutter="12" class="mb-3">
      <el-col :xs="24" :lg="13">
        <el-card shadow="never">
          <template #header>
            <div class="card-title">规则列表</div>
          </template>
          <el-table :data="rules" v-loading="loading" stripe>
            <el-table-column prop="name" label="名称" min-width="150" />
            <el-table-column prop="type" label="类型" width="110" />
            <el-table-column prop="level" label="级别" width="90">
              <template #default="{ row }">
                <el-tag :type="levelTagType(row.level)">{{ levelText(row.level) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="pattern" label="匹配规则" min-width="220" show-overflow-tooltip />
            <el-table-column label="启用" width="90">
              <template #default="{ row }">
                <el-tag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? '是' : '否' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="180" fixed="right">
              <template #default="{ row }">
                <el-button size="small" @click="openEdit(row)">编辑</el-button>
                <el-button size="small" type="danger" @click="removeRule(row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>

      <el-col :xs="24" :lg="11">
        <el-card shadow="never">
          <template #header>
            <div class="card-title">SQL 风险分析</div>
          </template>
          <el-input v-model="analyzeSql" type="textarea" :rows="12" placeholder="输入 SQL 进行分析" />
          <div class="analyze-actions">
            <el-button type="primary" @click="analyzeNow">分析</el-button>
            <el-button @click="analyzeSql = ''">清空</el-button>
          </div>
          <el-descriptions v-if="analyzeResult" :column="2" border class="mb-2">
            <el-descriptions-item label="通过">{{ analyzeResult.pass ? '是' : '否' }}</el-descriptions-item>
            <el-descriptions-item label="风险等级">{{ analyzeResult.risk_level || '-' }}</el-descriptions-item>
            <el-descriptions-item label="SQL类型">{{ analyzeResult.sql_type || '-' }}</el-descriptions-item>
            <el-descriptions-item label="问题数">{{ analyzeResult.issues?.length || 0 }}</el-descriptions-item>
          </el-descriptions>
          <el-alert v-if="analyzeResult && (analyzeResult.issues?.length || 0) === 0" type="success" :closable="false" title="未检测到明显风险问题" />
          <el-table v-if="analyzeResult && (analyzeResult.issues?.length || 0) > 0" :data="analyzeResult.issues" size="small" stripe>
            <el-table-column prop="type" label="类型" width="110" />
            <el-table-column prop="level" label="级别" width="90">
              <template #default="{ row }">
                <el-tag :type="levelTagType(row.level)">{{ levelText(row.level) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="message" label="问题" min-width="180" show-overflow-tooltip />
            <el-table-column prop="suggestion" label="建议" min-width="180" show-overflow-tooltip />
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <el-dialog append-to-body v-model="dialogVisible" :title="dialogTitle" width="760px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="名称">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="类型">
          <el-select v-model="form.type" class="w-52">
            <el-option label="syntax" value="syntax" />
            <el-option label="security" value="security" />
            <el-option label="performance" value="performance" />
          </el-select>
        </el-form-item>
        <el-form-item label="级别">
          <el-select v-model="form.level" class="w-52">
            <el-option label="INFO" :value="0" />
            <el-option label="WARNING" :value="1" />
            <el-option label="ERROR" :value="2" />
          </el-select>
        </el-form-item>
        <el-form-item label="匹配规则">
          <el-input v-model="form.pattern" placeholder="正则表达式" />
        </el-form-item>
        <el-form-item label="告警信息">
          <el-input v-model="form.message" />
        </el-form-item>
        <el-form-item label="修复建议">
          <el-input v-model="form.suggestion" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="2" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitRule">保存</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const loading = ref(false)
const rules = ref([])

const dialogVisible = ref(false)
const dialogTitle = ref('新增规则')
const isEdit = ref(false)
const currentId = ref('')

const form = ref({
  name: '',
  type: 'security',
  level: 1,
  pattern: '',
  message: '',
  suggestion: '',
  enabled: true,
  description: ''
})

const analyzeSql = ref('')
const analyzeResult = ref(null)

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const levelText = (v) => ({ 0: 'INFO', 1: 'WARNING', 2: 'ERROR' }[v] || '-')
const levelTagType = (v) => ({ 0: 'info', 1: 'warning', 2: 'danger' }[v] || 'info')

const fetchRules = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/v1/sqlaudit/rules', { headers: authHeaders() })
    rules.value = res.data?.data || []
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '获取规则失败')
  } finally {
    loading.value = false
  }
}

const reloadAll = async () => {
  await fetchRules()
}

const openCreate = () => {
  isEdit.value = false
  dialogTitle.value = '新增规则'
  currentId.value = ''
  form.value = {
    name: '',
    type: 'security',
    level: 1,
    pattern: '',
    message: '',
    suggestion: '',
    enabled: true,
    description: ''
  }
  dialogVisible.value = true
}

const openEdit = (row) => {
  isEdit.value = true
  dialogTitle.value = '编辑规则'
  currentId.value = row.id
  form.value = {
    name: row.name || '',
    type: row.type || 'security',
    level: Number(row.level || 0),
    pattern: row.pattern || '',
    message: row.message || '',
    suggestion: row.suggestion || '',
    enabled: !!row.enabled,
    description: row.description || ''
  }
  dialogVisible.value = true
}

const submitRule = async () => {
  if (!form.value.name.trim() || !form.value.pattern.trim()) {
    ElMessage.warning('请填写规则名称和匹配规则')
    return
  }
  try {
    if (isEdit.value) {
      await axios.put(`/api/v1/sqlaudit/rules/${currentId.value}`, form.value, { headers: authHeaders() })
      ElMessage.success('更新成功')
    } else {
      await axios.post('/api/v1/sqlaudit/rules', form.value, { headers: authHeaders() })
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    await fetchRules()
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '保存失败')
  }
}

const removeRule = async (row) => {
  try {
    await ElMessageBox.confirm(`确认删除规则 ${row.name} 吗？`, '提示', { type: 'warning' })
    await axios.delete(`/api/v1/sqlaudit/rules/${row.id}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    await fetchRules()
  } catch (err) {
    if (err !== 'cancel') ElMessage.error(err.response?.data?.message || '删除失败')
  }
}

const analyzeNow = async () => {
  if (!analyzeSql.value.trim()) {
    ElMessage.warning('请输入 SQL')
    return
  }
  try {
    const res = await axios.post('/api/v1/sqlaudit/analyze', { sql_content: analyzeSql.value }, { headers: authHeaders() })
    analyzeResult.value = res.data?.data || null
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '分析失败')
  }
}

onMounted(reloadAll)
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; gap: 12px; margin-bottom: 16px; }
.page-desc { margin: 4px 0 0; color: #606266; }
.page-actions { display: flex; gap: 8px; align-items: center; }
.card-title { font-weight: 600; }
.analyze-actions { margin-top: 10px; display: flex; gap: 8px; margin-bottom: 10px; }
.mb-2 { margin-bottom: 8px; }
.mb-3 { margin-bottom: 12px; }
.w-52 { width: 220px; }
</style>
