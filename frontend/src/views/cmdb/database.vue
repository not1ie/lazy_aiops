<template>
  <div>
    <el-card class="page-card">
    <template #header>
      <div class="header">
        <div>
          <div class="title">数据库资产</div>
          <div class="desc">管理数据库实例与连接信息</div>
        </div>
        <div class="actions">
          <el-button type="primary" icon="Plus" @click="openCreate">新增资产</el-button>
          <el-button icon="Upload" @click="openImport">批量导入</el-button>
          <el-button icon="Download" @click="exportCSV">导出</el-button>
          <el-button type="danger" plain icon="Delete" :disabled="selectedRows.length === 0" @click="handleBatchDelete">
            批量删除 ({{ selectedRows.length }})
          </el-button>
          <el-button icon="Refresh" @click="fetchData">刷新</el-button>
        </div>
      </div>
    </template>

    <div class="filters">
      <el-input v-model="filters.keyword" placeholder="名称/主机" clearable @clear="fetchData" @keyup.enter="fetchData">
        <template #append>
          <el-button icon="Search" @click="fetchData" />
        </template>
      </el-input>
      <el-select v-model="filters.environment" placeholder="环境" clearable @change="fetchData">
        <el-option label="开发" value="dev" />
        <el-option label="测试" value="test" />
        <el-option label="生产" value="prod" />
      </el-select>
    </div>

    <el-table :fit="true" :data="items" v-loading="loading" stripe style="width: 100%" @selection-change="selectedRows = $event">
      <el-table-column type="selection" width="48" />
      <el-table-column prop="name" label="名称" min-width="160" />
      <el-table-column prop="type" label="类型" width="120" />
      <el-table-column label="地址" min-width="180">
        <template #default="{ row }">
          {{ row.host }}:{{ row.port }}
        </template>
      </el-table-column>
      <el-table-column prop="database" label="库名" min-width="140" />
      <el-table-column prop="environment" label="环境" width="100" />
      <el-table-column prop="owner" label="负责人" width="120" />
      <el-table-column prop="status" label="状态" width="120">
        <template #default="{ row }">
          <el-tooltip v-if="row.status === 2" :content="row.status_reason || '连接失败'" placement="top">
            <el-tag type="danger" style="cursor: pointer">不可用</el-tag>
          </el-tooltip>
          <el-tag v-else-if="row.status === 1" type="success">正常</el-tag>
          <el-tag v-else type="info">禁用</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="420" fixed="right">
        <template #default="{ row }">
          <el-space wrap :size="[8, 8]">
            <el-button size="small" type="warning" plain icon="FirstAidKit" @click="openTest(row)">测试</el-button>
            <el-button size="small" :type="row.slow_log_enabled ? 'danger' : 'success'" plain @click="toggleSlowLog(row)">
              {{ row.slow_log_enabled ? '关闭慢查询' : '开启慢查询' }}
            </el-button>
            <el-button v-if="row.slow_log_enabled" size="small" type="primary" plain @click="openSlowLogAnalysis(row)">
              AI 自治分析
            </el-button>
            <el-button size="small" type="primary" plain @click="openEdit(row)">编辑</el-button>
            <el-button size="small" type="danger" plain @click="handleDelete(row)">删除</el-button>
          </el-space>
        </template>
      </el-table-column>
    </el-table>

    <div class="pagination-wrapper">
      <el-pagination
        v-model:current-page="currentPage"
        v-model:page-size="pageSize"
        :page-sizes="[10, 20, 50, 100]"
        layout="total, sizes, prev, pager, next, jumper"
        :total="totalItems"
        @size-change="fetchData"
        @current-change="fetchData"
      />
    </div>
  </el-card>

  <el-dialog append-to-body v-model="dialogVisible" :title="isEdit ? '编辑数据库资产' : '新增数据库资产'" width="560px">
    <el-form :model="form" label-width="100px">
      <el-form-item label="名称" required>
        <el-input v-model="form.name" />
      </el-form-item>
      <el-form-item label="类型" required>
        <el-select v-model="form.type" placeholder="请选择类型" style="width: 100%">
          <el-option label="MySQL" value="mysql" />
          <el-option label="PostgreSQL" value="postgres" />
          <el-option label="Redis" value="redis" />
          <el-option label="MongoDB" value="mongodb" />
          <el-option label="Oracle" value="oracle" />
        </el-select>
      </el-form-item>
      <el-form-item label="主机" required>
        <el-input v-model="form.host" />
      </el-form-item>
      <el-form-item label="端口">
        <el-input-number v-model="form.port" :min="1" :max="65535" />
      </el-form-item>
      <el-form-item label="库名">
        <el-input v-model="form.database" />
      </el-form-item>
      <el-form-item label="用户名">
        <el-input v-model="form.username" />
      </el-form-item>
      <el-form-item label="密码">
        <el-input v-model="form.password" type="password" show-password />
        <div v-if="isEdit" class="helper-row">已加载当前密码，可直接在原值上修改。</div>
      </el-form-item>
      <el-form-item label="环境">
        <el-select v-model="form.environment" style="width: 100%">
          <el-option label="开发" value="dev" />
          <el-option label="测试" value="test" />
          <el-option label="生产" value="prod" />
        </el-select>
      </el-form-item>
      <el-form-item label="负责人">
        <el-input v-model="form.owner" />
      </el-form-item>
      <el-form-item label="标签">
        <el-input v-model="form.tags" placeholder="逗号分隔" />
      </el-form-item>
      <el-form-item label="状态">
        <el-switch v-model="form.status" :active-value="1" :inactive-value="0" />
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

  <el-dialog append-to-body v-model="importVisible" title="批量导入数据库资产" width="720px">
    <el-alert type="info" :closable="false" show-icon>
      格式：name,type,host,port,username,password,database,environment,owner,tags,status,description（第一行可写表头）
    </el-alert>
    <el-input v-model="importText" type="textarea" :rows="10" />
    <div class="import-actions">
      <el-button @click="importVisible = false">取消</el-button>
      <el-button type="primary" :loading="importLoading" @click="submitImport">开始导入</el-button>
    </div>
  </el-dialog>

  <el-dialog append-to-body v-model="testVisible" title="数据库连通性测试" width="560px">
    <el-alert v-if="testError" type="error" :closable="false" show-icon>{{ testError }}</el-alert>
    <el-alert v-if="testSuccess" type="success" :closable="false" show-icon>{{ testSuccess }}</el-alert>
    <template #footer>
      <el-button @click="testVisible = false">关闭</el-button>
      <el-button type="primary" :loading="testLoading" @click="submitTest">测试</el-button>
    </template>
  </el-dialog>

  <!-- AI 慢日志自治分析弹窗 -->
  <el-dialog append-to-body v-model="slowLogDialogVisible" title="AI 数据库慢日志自治分析" width="800px">
    <div v-loading="slowLogLoading">
      <div v-if="slowLogData">
        <!-- 统计面板 -->
        <div class="stats-row" style="display: flex; gap: 16px; margin-bottom: 20px;">
          <div class="stat-card" style="flex: 1; padding: 16px; background: #f5f7fa; border-radius: 8px; text-align: center;">
            <div style="font-size: 14px; color: #909399; margin-bottom: 8px;">今日慢 SQL 计数</div>
            <div style="font-size: 24px; font-weight: bold; color: #e6a23c;">{{ slowLogData.slow_sql_count }} 次</div>
          </div>
          <div class="stat-card" style="flex: 1; padding: 16px; background: #f5f7fa; border-radius: 8px; text-align: center;">
            <div style="font-size: 14px; color: #909399; margin-bottom: 8px;">平均执行延迟</div>
            <div style="font-size: 24px; font-weight: bold; color: #f56c6c;">{{ slowLogData.avg_query_time_s }} 秒</div>
          </div>
          <div class="stat-card" style="flex: 1; padding: 16px; background: #f5f7fa; border-radius: 8px; text-align: center;">
            <div style="font-size: 14px; color: #909399; margin-bottom: 8px;">无索引扫描数</div>
            <div style="font-size: 24px; font-weight: bold; color: #409eff;">{{ slowLogData.no_index_scans }} 次</div>
          </div>
        </div>

        <!-- AI 诊断摘要 -->
        <el-card shadow="never" style="margin-bottom: 20px; border-color: #dcdfe6; background: #fdf6ec;">
          <template #header>
            <div style="font-weight: bold; color: #e6a23c; display: flex; align-items: center; gap: 8px;">
              <el-icon><Warning /></el-icon>
              <span>AI 慢日志自治诊断报告</span>
            </div>
          </template>
          <div style="font-size: 14px; color: #606266; line-height: 1.6; white-space: pre-wrap;">
            {{ slowLogData.ai_summary_report }}
          </div>
        </el-card>

        <!-- 慢 SQL 详情列表 -->
        <h4 style="margin: 0 0 12px 0; color: #303133;">慢 SQL 排查与索引调优推荐</h4>
        <el-collapse accordion>
          <el-collapse-item v-for="(q, idx) in slowLogData.queries" :key="idx" :name="idx">
            <template #title>
              <div style="display: flex; justify-content: space-between; width: 90%; align-items: center;">
                <span style="font-family: monospace; font-size: 12px; color: #f56c6c; text-overflow: ellipsis; overflow: hidden; white-space: nowrap; width: 60%;">
                  {{ q.sql }}
                </span>
                <el-tag size="small" type="danger">查询耗时: {{ q.query_time }}s</el-tag>
              </div>
            </template>
            <div style="padding: 12px; background: #fafafa; border-radius: 4px;">
              <!-- 基础指标 -->
              <div style="display: flex; gap: 24px; margin-bottom: 12px; font-size: 13px; color: #606266;">
                <span><strong>执行次数:</strong> {{ q.count }} 次</span>
                <span><strong>扫描行数 Examined:</strong> {{ q.rows_examined.toLocaleString() }}</span>
                <span><strong>返回行数 Sent:</strong> {{ q.rows_sent }}</span>
              </div>
              <!-- 慢因分析 -->
              <div style="margin-bottom: 12px;">
                <div style="font-weight: bold; color: #303133; font-size: 13px; margin-bottom: 4px;">慢因解析</div>
                <div style="font-size: 13px; color: #909399;">{{ q.reason }}</div>
              </div>
              <!-- 推荐索引 SQL -->
              <div style="margin-bottom: 12px;">
                <div style="font-weight: bold; color: #67c23a; font-size: 13px; margin-bottom: 4px;">推荐优化索引 SQL</div>
                <pre style="margin: 0; padding: 8px; background: #e8f5e9; color: #2e7d32; border-radius: 4px; font-family: monospace; font-size: 12px; overflow-x: auto;">{{ q.recommendation }}</pre>
              </div>
              <!-- SQL 重写指引 -->
              <div>
                <div style="font-weight: bold; color: #409eff; font-size: 13px; margin-bottom: 4px;">查询改写建议</div>
                <div style="font-size: 13px; color: #606266; line-height: 1.4;">{{ q.rewrite }}</div>
              </div>
            </div>
          </el-collapse-item>
        </el-collapse>
      </div>
    </div>
    <template #footer>
      <el-button type="primary" @click="slowLogDialogVisible = false">完成优化</el-button>
    </template>
  </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, watch } from 'vue'
import request from '../../utils/request'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Warning } from '@element-plus/icons-vue'

const loading = ref(false)
const saving = ref(false)
const items = ref([])
const currentPage = ref(1)
const pageSize = ref(20)
const totalItems = ref(0)

watch([() => filters.keyword, () => filters.environment], () => {
  currentPage.value = 1
  fetchData()
})
const dialogVisible = ref(false)
const isEdit = ref(false)
const currentId = ref('')
const selectedRows = ref([])

const importVisible = ref(false)
const importLoading = ref(false)
const importText = ref('')

const testVisible = ref(false)
const testLoading = ref(false)
const testRow = ref(null)
const testError = ref('')
const testSuccess = ref('')

const slowLogDialogVisible = ref(false)
const slowLogLoading = ref(false)
const slowLogData = ref(null)

const filters = reactive({
  keyword: '',
  environment: ''
})

const form = reactive({
  name: '',
  type: 'mysql',
  host: '',
  port: 3306,
  username: '',
  password: '',
  database: '',
  environment: 'dev',
  owner: '',
  tags: '',
  status: 1,
  description: ''
})

const getErrorMessage = (error, fallback) => {
  if (error?.response?.data?.message) return error.response.data.message
  if (error?.message) return error.message
  return fallback
}

const fetchData = async () => {
  loading.value = true
  try {
    const res = await request.get('/api/v1/cmdb/databases', {
      params: {
        keyword: filters.keyword,
        environment: filters.environment,
        page: currentPage.value,
        page_size: pageSize.value
      }
    })
    if (res.data.code === 0) {
      const resData = res.data.data
      if (resData && typeof resData === 'object' && Array.isArray(resData.list)) {
        items.value = resData.list
        totalItems.value = resData.total || 0
      } else {
        items.value = Array.isArray(resData) ? resData : []
        totalItems.value = items.value.length
      }
    }
  } catch (error) {
    ElMessage.error(getErrorMessage(error, '加载失败'))
  } finally {
    loading.value = false
  }
}

const openCreate = () => {
  isEdit.value = false
  currentId.value = ''
  Object.assign(form, {
    name: '',
    type: 'mysql',
    host: '',
    port: 3306,
    username: '',
    password: '',
    database: '',
    environment: 'dev',
    owner: '',
    tags: '',
    status: 1,
    description: ''
  })
  dialogVisible.value = true
}

const openEdit = async (row) => {
  isEdit.value = true
  currentId.value = row.id
  try {
    const res = await request.get(`/api/v1/cmdb/databases/${row.id}`)
    if (res.data.code === 0) {
      Object.assign(form, res.data.data || {})
      dialogVisible.value = true
    }
  } catch (error) {
    ElMessage.error(getErrorMessage(error, '获取详情失败'))
  }
}

const saveItem = async () => {
  if (!form.name || !form.host) {
    ElMessage.warning('请填写名称与主机')
    return
  }
  saving.value = true
  try {
    const url = isEdit.value ? `/api/v1/cmdb/databases/${currentId.value}` : '/api/v1/cmdb/databases'
    const method = isEdit.value ? 'put' : 'post'
    const res = await request({
      url,
      method,
      data: form
    })
    if (res.data.code === 0) {
      ElMessage.success(isEdit.value ? '更新成功' : '创建成功')
      dialogVisible.value = false
      await fetchData()
    }
  } catch (error) {
    ElMessage.error(getErrorMessage(error, '保存失败'))
  } finally {
    saving.value = false
  }
}

const handleDelete = async (row) => {
  try {
    await ElMessageBox.confirm(`确定删除资产“${row.name}”吗？`, '提示', {
      type: 'warning'
    })
    await request.delete(`/api/v1/cmdb/databases/${row.id}`)
    ElMessage.success('删除成功')
    await fetchData()
  } catch (error) {
    if (error !== 'cancel' && error !== 'close') {
      ElMessage.error(getErrorMessage(error, '删除失败'))
    }
  }
}

const handleBatchDelete = async () => {
  if (selectedRows.value.length === 0) return
  try {
    await ElMessageBox.confirm(`确定删除选中的 ${selectedRows.value.length} 个资产吗？`, '提示', {
      type: 'warning'
    })
    for (const row of selectedRows.value) {
      await request.delete(`/api/v1/cmdb/databases/${row.id}`)
    }
    ElMessage.success('批量删除成功')
    selectedRows.value = []
    await fetchData()
  } catch (error) {
    if (error !== 'cancel' && error !== 'close') {
      ElMessage.error(getErrorMessage(error, '批量删除失败'))
    }
  }
}

const openImport = () => {
  importText.value = ''
  importVisible.value = true
}

const parseCSV = (text) => {
  const lines = text.split(/\r?\n/).map(l => l.trim()).filter(Boolean)
  if (lines.length === 0) return []
  const delim = lines[0].includes('\t') ? '\t' : ','
  const headers = lines[0].toLowerCase().split(delim).map(s => s.trim())
  const hasHeader = headers.includes('name') || headers.includes('host')
  const start = hasHeader ? 1 : 0
  const cols = hasHeader ? headers : ['name','type','host','port','username','password','database','environment','owner','tags','status','description']
  return lines.slice(start).map(line => {
    const parts = line.split(delim).map(s => s.trim())
    const obj = {}
    cols.forEach((k, idx) => { obj[k] = parts[idx] || '' })
    return obj
  })
}

const submitImport = async () => {
  const rows = parseCSV(importText.value)
  if (rows.length === 0) {
    ElMessage.warning('请填写导入内容')
    return
  }
  importLoading.value = true
  try {
    for (const row of rows) {
      if (!row.name || !row.host) continue
      await request.post('/api/v1/cmdb/databases', {
        name: row.name,
        type: row.type || 'mysql',
        host: row.host,
        port: row.port ? Number(row.port) : 3306,
        username: row.username || '',
        password: row.password || '',
        database: row.database || '',
        environment: row.environment || 'dev',
        owner: row.owner || '',
        tags: row.tags || '',
        status: row.status ? Number(row.status) : 1,
        description: row.description || ''
      })
    }
    ElMessage.success('导入完成')
    importVisible.value = false
    await fetchData()
  } catch (e) {
    ElMessage.error(getErrorMessage(e, '导入失败'))
  } finally {
    importLoading.value = false
  }
}

const exportCSV = () => {
  const headers = ['name','type','host','port','database','environment','owner','status']
  const rows = items.value.map(d => [d.name, d.type, d.host, d.port, d.database, d.environment, d.owner, d.status])
  const csv = [headers.join(','), ...rows.map(r => r.map(v => `"${String(v ?? '').replace(/"/g, '""')}"`).join(','))].join('\n')
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'cmdb_databases.csv'
  a.click()
  URL.revokeObjectURL(url)
}

const openTest = (row) => {
  testRow.value = row
  testError.value = ''
  testSuccess.value = ''
  testVisible.value = true
}

const submitTest = async () => {
  if (!testRow.value) return
  testLoading.value = true
  try {
    const res = await request.post(`/api/v1/cmdb/databases/${testRow.value.id}/test`, {})
    if (res.data.code === 0) {
      testSuccess.value = res.data.message || '连接成功'
    } else {
      testError.value = res.data.message || '连接失败'
    }
  } catch (e) {
    testError.value = getErrorMessage(e, '连接失败')
  } finally {
    testLoading.value = false
  }
}

const toggleSlowLog = async (row) => {
  try {
    const res = await request.post(`/api/v1/cmdb/databases/${row.id}/slowlog/toggle`, {})
    if (res.data.code === 0) {
      ElMessage.success(res.data.message)
      await fetchData()
    }
  } catch (e) {
    ElMessage.error(getErrorMessage(e, '操作失败'))
  }
}

const openSlowLogAnalysis = async (row) => {
  slowLogLoading.value = true
  slowLogDialogVisible.value = true
  slowLogData.value = null
  try {
    const res = await request.get(`/api/v1/cmdb/databases/${row.id}/slowlog/analysis`)
    if (res.data.code === 0) {
      slowLogData.value = res.data.data
    }
  } catch (e) {
    ElMessage.error(getErrorMessage(e, '加载自治分析报告失败'))
  } finally {
    slowLogLoading.value = false
  }
}

onMounted(fetchData)
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.header { display: flex; justify-content: space-between; align-items: center; }
.title { font-size: 18px; font-weight: 600; }
.desc { color: #909399; margin-top: 4px; }
.actions { display: flex; gap: 8px; align-items: center; }
.filters { display: flex; gap: 12px; margin-bottom: 16px; }
.filters .el-input { width: 240px; }
.filters .el-select { width: 160px; }
.import-actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 12px; }
.helper-row { margin-top: 6px; color: var(--el-text-color-secondary); font-size: 12px; line-height: 1.4; }
.pagination-wrapper { display: flex; justify-content: flex-end; margin-top: 16px; }
</style>
