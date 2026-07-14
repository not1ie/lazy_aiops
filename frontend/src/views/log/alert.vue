<template>
  <div class="log-alert-container" v-loading="loading">
    <div class="page-header">
      <div class="header-title">
        <div class="icon-box"><el-icon><Bell /></el-icon></div>
        <div>
          <h2>日志告警规则</h2>
          <p>基于 ES 或 Loki 查询语句，设置阈值与通知渠道</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button type="primary" icon="Plus" @click="handleAdd">新建告警规则</el-button>
      </div>
    </div>

    <el-card class="table-card">
      <div class="toolbar">
        <el-input v-model="searchKey" placeholder="搜索规则名称..." prefix-icon="Search" style="width: 240px" clearable @input="filterData" />
        <el-select v-model="levelFilter" placeholder="告警级别" style="width: 140px; margin-left: 12px" clearable @change="filterData">
          <el-option label="P0 严重" value="P0" />
          <el-option label="P1 紧急" value="P1" />
          <el-option label="P2 警告" value="P2" />
        </el-select>
        <el-button icon="Refresh" circle style="margin-left: auto" @click="fetchData" />
      </div>

      <el-table :data="filteredTableData" style="width: 100%" v-loading="tableLoading">
        <el-table-column prop="name" label="规则名称" min-width="150">
          <template #default="{ row }">
            <div class="rule-name">{{ row.name }}</div>
            <div class="rule-lib text-muted">{{ row.library_name }}</div>
          </template>
        </el-table-column>
        <el-table-column prop="level" label="级别" width="100">
          <template #default="{ row }">
            <el-tag :type="getLevelType(row.level)" effect="dark" size="small">{{ row.level }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="condition" label="触发条件" min-width="250">
          <template #default="{ row }">
            <div class="condition-expr">
              <span class="keyword">When</span> {{ row.query }}
              <br/>
              <span class="keyword">Count</span> {{ row.operator }} {{ row.threshold }} <span class="keyword">in</span> {{ row.duration }}
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="channels" label="通知渠道" min-width="150">
          <template #default="{ row }">
            <el-tag v-for="c in formatChannels(row.channels)" :key="c" size="small" type="info" class="channel-tag">{{ c }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="enabled" label="启用状态" width="100">
          <template #default="{ row }">
            <el-switch v-model="row.enabled" @change="handleToggle(row)" />
          </template>
        </el-table-column>
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="handleEdit(row)">编辑</el-button>
            <el-button link type="danger" size="small" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- Dialog for Adding/Editing Alert Rule -->
    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="600px" destroy-on-close>
      <el-form :model="form" :rules="rules" ref="formRef" label-width="100px">
        <el-form-item label="规则名称" prop="name">
          <el-input v-model="form.name" placeholder="请输入规则名称，如: Nginx 5xx错误过多" />
        </el-form-item>
        <el-form-item label="日志库" prop="library_id">
          <el-select v-model="form.library_id" placeholder="选择关联的日志库" style="width: 100%">
            <el-option v-for="lib in libraries" :key="lib.id" :label="`${lib.name} (${lib.type.toUpperCase()})`" :value="lib.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="告警级别" prop="level">
          <el-radio-group v-model="form.level">
            <el-radio-button label="P0">P0 严重</el-radio-button>
            <el-radio-button label="P1">P1 紧急</el-radio-button>
            <el-radio-button label="P2">P2 警告</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="检索查询句" prop="query">
          <el-input v-model="form.query" type="textarea" :rows="3" placeholder="请输入 ES Lucene 语法或 Loki LogQL 表达式，如: {app='nginx'} |= '500'" />
        </el-form-item>
        <el-form-item label="触发阈值">
          <div style="display: flex; gap: 10px; width: 100%">
            <el-select v-model="form.operator" placeholder="操作符" style="width: 120px">
              <el-option label="大于 (>)" value=">" />
              <el-option label="小于 (<)" value="<" />
              <el-option label="大于等于 (>=)" value=">=" />
              <el-option label="小于等于 (<=)" value="<=" />
              <el-option label="等于 (==)" value="==" />
            </el-select>
            <el-input-number v-model="form.threshold" :min="0" style="flex: 1" />
            <el-select v-model="form.duration" placeholder="时间窗口" style="width: 140px">
              <el-option label="1分钟" value="1m" />
              <el-option label="5分钟" value="5m" />
              <el-option label="10分钟" value="10m" />
              <el-option label="30分钟" value="30m" />
            </el-select>
          </div>
        </el-form-item>
        <el-form-item label="通知渠道" prop="channelsList">
          <el-select v-model="form.channelsList" multiple placeholder="请选择通知渠道" style="width: 100%">
            <el-option label="钉钉" value="钉钉" />
            <el-option label="邮件" value="邮件" />
            <el-option label="短信" value="短信" />
            <el-option label="电话" value="电话" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" :loading="submitLoading" @click="handleSubmit">确定</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Bell, Plus, Search, Refresh } from '@element-plus/icons-vue'

const loading = ref(false)
const tableLoading = ref(false)
const searchKey = ref('')
const levelFilter = ref('')

const tableData = ref([])
const filteredTableData = ref([])
const libraries = ref([])

const dialogVisible = ref(false)
const dialogTitle = ref('')
const isEdit = ref(false)
const submitLoading = ref(false)
const formRef = ref(null)

const form = ref({
  id: '',
  name: '',
  library_id: '',
  level: 'P1',
  query: '',
  operator: '>',
  threshold: 5,
  duration: '5m',
  channelsList: []
})

const rules = {
  name: [{ required: true, message: '请输入规则名称', trigger: 'blur' }],
  library_id: [{ required: true, message: '请选择关联的日志库', trigger: 'change' }],
  query: [{ required: true, message: '请输入检索查询句', trigger: 'blur' }],
  channelsList: [{ required: true, type: 'array', message: '请至少选择一个通知渠道', trigger: 'change' }]
}

const authHeaders = () => ({
  Authorization: `Bearer ${localStorage.getItem('token')}`
})

const formatChannels = (chStr) => {
  if (!chStr) return []
  return chStr.split(',').filter(Boolean)
}

const getLevelType = (level) => {
  if (level === 'P0') return 'danger'
  if (level === 'P1') return 'warning'
  return 'info'
}

const filterData = () => {
  let data = [...tableData.value]
  if (searchKey.value) {
    const key = searchKey.value.toLowerCase()
    data = data.filter(item => item.name.toLowerCase().includes(key) || item.query.toLowerCase().includes(key))
  }
  if (levelFilter.value) {
    data = data.filter(item => item.level === levelFilter.value)
  }
  filteredTableData.value = data
}

const fetchData = async () => {
  tableLoading.value = true
  try {
    const res = await axios.get('/api/v1/log/alerts', { headers: authHeaders() })
    if (res.data.code === 0) {
      tableData.value = res.data.data
      filterData()
    } else {
      ElMessage.error(res.data.message || '获取告警规则失败')
    }
  } catch (err) {
    ElMessage.error('加载告警规则出错')
  } finally {
    tableLoading.value = false
  }
}

const fetchLibraries = async () => {
  try {
    const res = await axios.get('/api/v1/log/libraries', { headers: authHeaders() })
    if (res.data.code === 0) {
      libraries.value = res.data.data
    }
  } catch (err) {
    console.error('获取日志库失败', err)
  }
}

const handleAdd = () => {
  isEdit.value = false
  dialogTitle.value = '新建告警规则'
  form.value = {
    id: '',
    name: '',
    library_id: '',
    level: 'P1',
    query: '',
    operator: '>',
    threshold: 10,
    duration: '5m',
    channelsList: ['邮件', '钉钉']
  }
  dialogVisible.value = true
  fetchLibraries()
}

const handleEdit = (row) => {
  isEdit.value = true
  dialogTitle.value = '编辑告警规则'
  form.value = {
    ...row,
    channelsList: formatChannels(row.channels)
  }
  dialogVisible.value = true
  fetchLibraries()
}

const handleToggle = async (row) => {
  try {
    const res = await axios.post(`/api/v1/log/alerts/${row.id}/toggle`, {}, { headers: authHeaders() })
    if (res.data.code === 0) {
      ElMessage.success(`${row.name} 已${row.enabled ? '启用' : '禁用'}`)
    } else {
      ElMessage.error(res.data.message || '状态切换失败')
      row.enabled = !row.enabled
    }
  } catch (err) {
    ElMessage.error('接口请求失败')
    row.enabled = !row.enabled
  }
}

const handleDelete = (row) => {
  ElMessageBox.confirm(`确定要删除告警规则 "${row.name}" 吗？`, '删除规则', {
    type: 'warning',
    confirmButtonText: '删除',
    cancelButtonText: '取消'
  }).then(async () => {
    try {
      const res = await axios.delete(`/api/v1/log/alerts/${row.id}`, { headers: authHeaders() })
      if (res.data.code === 0) {
        ElMessage.success('删除成功')
        fetchData()
      } else {
        ElMessage.error(res.data.message || '删除失败')
      }
    } catch (err) {
      ElMessage.error('删除请求出错')
    }
  }).catch(() => {})
}

const handleSubmit = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    submitLoading.value = true
    const payload = {
      ...form.value,
      channels: form.value.channelsList.join(',')
    }
    try {
      let res
      if (isEdit.value) {
        res = await axios.put(`/api/v1/log/alerts/${form.value.id}`, payload, { headers: authHeaders() })
      } else {
        res = await axios.post('/api/v1/log/alerts', payload, { headers: authHeaders() })
      }
      if (res.data.code === 0) {
        ElMessage.success('保存成功')
        dialogVisible.value = false
        fetchData()
      } else {
        ElMessage.error(res.data.message || '保存失败')
      }
    } catch (err) {
      ElMessage.error('操作接口出错，请重试')
    } finally {
      submitLoading.value = false
    }
  })
}

onMounted(() => {
  fetchData()
})
</script>

<style scoped>
.log-alert-container {
  padding-bottom: 20px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}
.header-title {
  display: flex;
  align-items: center;
  gap: 16px;
}
.icon-box {
  width: 48px;
  height: 48px;
  background-color: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
}
.header-title h2 {
  margin: 0 0 4px 0;
  font-size: 20px;
  color: #1f2329;
}
.header-title p {
  margin: 0;
  font-size: 13px;
  color: #8f959e;
}

.table-card {
  border-radius: 12px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.02) !important;
}

.toolbar {
  display: flex;
  align-items: center;
  margin-bottom: 16px;
}

.rule-name {
  font-weight: 500;
  color: #1f2329;
}
.text-muted {
  font-size: 12px;
  color: #8f959e;
  margin-top: 4px;
}

.condition-expr {
  font-family: monospace;
  font-size: 12px;
  background: #f8f9fa;
  padding: 6px 10px;
  border-radius: 6px;
  line-height: 1.6;
}
.keyword {
  color: var(--el-color-primary);
  font-weight: bold;
}

.channel-tag {
  margin-right: 4px;
  margin-bottom: 2px;
}
</style>
