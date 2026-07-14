<template>
  <div class="log-permission-container" v-loading="loading">
    <div class="page-header">
      <div class="header-title">
        <div class="icon-box"><el-icon><Lock /></el-icon></div>
        <div>
          <h2>日志权限管理</h2>
          <p>控制用户和角色对不同日志库的访问权限，确保数据安全</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button type="primary" icon="Plus" @click="handleAdd">分配权限</el-button>
      </div>
    </div>

    <el-card class="table-card">
      <div class="toolbar">
        <el-input v-model="searchKey" placeholder="搜索授权主体..." prefix-icon="Search" style="width: 260px" clearable @input="filterData" />
        <el-select v-model="typeFilter" placeholder="授权类型" style="width: 140px; margin-left: 12px" clearable @change="filterData">
          <el-option label="角色授权" value="role" />
          <el-option label="人员授权" value="user" />
        </el-select>
        <el-button icon="Refresh" circle style="margin-left: auto" @click="fetchData" />
      </div>

      <el-table :data="filteredTableData" style="width: 100%" v-loading="tableLoading">
        <el-table-column prop="subject" label="授权主体" min-width="150">
          <template #default="{ row }">
            <div class="subject-info">
              <el-icon class="subject-icon" :class="row.type">
                <User v-if="row.type === 'user'" />
                <UserFilled v-else />
              </el-icon>
              <div>
                <div class="subject-name">{{ row.subject }}</div>
                <div class="subject-type text-muted">{{ row.type === 'user' ? '具体用户' : '系统角色' }}</div>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="libraries" label="可访问的日志库" min-width="250">
          <template #default="{ row }">
            <div class="lib-tags">
              <el-tag v-for="lib in formatCommaList(row.library_ids)" :key="lib" size="small" type="info" class="lib-tag">
                <el-icon><Collection /></el-icon> {{ lib }}
              </el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="actions" label="操作权限" width="200">
          <template #default="{ row }">
            <div class="action-tags">
              <el-tag v-for="act in formatCommaList(row.actions)" :key="act" size="small" :type="getActionType(act)" effect="plain" class="action-tag">
                {{ act }}
              </el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="creator" label="授权人" width="120" />
        <el-table-column prop="created_at" label="授权时间" width="180">
          <template #default="{ row }">
            {{ formatTime(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <el-button link type="danger" size="small" @click="handleRevoke(row)">撤销</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- Dialog for Allocating Permissions -->
    <el-dialog v-model="dialogVisible" title="分配日志权限" width="550px" destroy-on-close>
      <el-form :model="form" :rules="rules" ref="formRef" label-width="100px">
        <el-form-item label="授权类型" prop="type">
          <el-radio-group v-model="form.type">
            <el-radio label="role">系统角色</el-radio>
            <el-radio label="user">具体用户</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="授权主体" prop="subject">
          <el-input v-model="form.subject" placeholder="请输入角色名（如: Developer）或用户名（如: lisi）" />
        </el-form-item>
        <el-form-item label="可访问日志库" prop="librariesList">
          <el-select v-model="form.librariesList" multiple placeholder="请选择日志库" style="width: 100%">
            <el-option v-for="lib in libraries" :key="lib.id" :label="lib.name" :value="lib.name" />
          </el-select>
        </el-form-item>
        <el-form-item label="操作权限" prop="actionsList">
          <el-select v-model="form.actionsList" multiple placeholder="请选择授权的操作" style="width: 100%">
            <el-option label="查询" value="查询" />
            <el-option label="Tail" value="Tail" />
            <el-option label="下载" value="下载" />
            <el-option label="配置告警" value="配置告警" />
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
import { Lock, Plus, Search, Refresh, User, UserFilled, Collection } from '@element-plus/icons-vue'

const loading = ref(false)
const tableLoading = ref(false)
const searchKey = ref('')
const typeFilter = ref('')

const tableData = ref([])
const filteredTableData = ref([])
const libraries = ref([])

const dialogVisible = ref(false)
const submitLoading = ref(false)
const formRef = ref(null)

const form = ref({
  type: 'role',
  subject: '',
  librariesList: [],
  actionsList: []
})

const rules = {
  subject: [{ required: true, message: '请输入授权主体', trigger: 'blur' }],
  librariesList: [{ required: true, type: 'array', message: '请至少选择一个日志库', trigger: 'change' }],
  actionsList: [{ required: true, type: 'array', message: '请选择授予的操作权限', trigger: 'change' }]
}

const authHeaders = () => ({
  Authorization: `Bearer ${localStorage.getItem('token')}`
})

const formatCommaList = (str) => {
  if (!str) return []
  return str.split(',').filter(Boolean)
}

const formatTime = (timeStr) => {
  if (!timeStr) return '-'
  const date = new Date(timeStr)
  return date.toLocaleString()
}

const getActionType = (act) => {
  if (act.includes('告警') || act.includes('下载')) return 'warning'
  return 'success'
}

const filterData = () => {
  let data = [...tableData.value]
  if (searchKey.value) {
    const key = searchKey.value.toLowerCase()
    data = data.filter(item => item.subject.toLowerCase().includes(key))
  }
  if (typeFilter.value) {
    data = data.filter(item => item.type === typeFilter.value)
  }
  filteredTableData.value = data
}

const fetchData = async () => {
  tableLoading.value = true
  try {
    const res = await axios.get('/api/v1/log/permissions', { headers: authHeaders() })
    if (res.data.code === 0) {
      tableData.value = res.data.data
      filterData()
    } else {
      ElMessage.error(res.data.message || '获取权限列表失败')
    }
  } catch (err) {
    ElMessage.error('加载日志权限出错')
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
  form.value = {
    type: 'role',
    subject: '',
    librariesList: [],
    actionsList: ['查询', 'Tail']
  }
  dialogVisible.value = true
  fetchLibraries()
}

const handleRevoke = (row) => {
  ElMessageBox.confirm(`确定要撤销对 "${row.subject}" 的日志权限分配吗？`, '确认撤销', {
    type: 'warning',
    confirmButtonText: '确定撤销',
    cancelButtonText: '取消'
  }).then(async () => {
    try {
      const res = await axios.delete(`/api/v1/log/permissions/${row.id}`, { headers: authHeaders() })
      if (res.data.code === 0) {
        ElMessage.success('撤销权限成功')
        fetchData()
      } else {
        ElMessage.error(res.data.message || '撤销失败')
      }
    } catch (err) {
      ElMessage.error('撤销权限请求出错')
    }
  }).catch(() => {})
}

const handleSubmit = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    submitLoading.value = true
    const payload = {
      type: form.value.type,
      subject: form.value.subject,
      library_ids: form.value.librariesList.join(','),
      actions: form.value.actionsList.join(',')
    }
    try {
      const res = await axios.post('/api/v1/log/permissions', payload, { headers: authHeaders() })
      if (res.data.code === 0) {
        ElMessage.success('授权成功')
        dialogVisible.value = false
        fetchData()
      } else {
        ElMessage.error(res.data.message || '授权失败')
      }
    } catch (err) {
      ElMessage.error('提交请求失败')
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
.log-permission-container {
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

.subject-info {
  display: flex;
  align-items: center;
  gap: 10px;
}
.subject-icon {
  font-size: 24px;
  color: #8f959e;
  background: #f5f7fa;
  padding: 6px;
  border-radius: 50%;
}
.subject-icon.role {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}
.subject-name {
  font-weight: 500;
  color: #1f2329;
}
.text-muted {
  font-size: 12px;
  color: #8f959e;
  margin-top: 2px;
}

.lib-tags, .action-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
</style>
