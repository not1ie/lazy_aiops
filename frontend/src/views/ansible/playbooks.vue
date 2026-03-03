<template>
  <div class="ansible-page">
    <el-card class="section-card">
      <template #header>
        <div class="header">
          <div>
            <div class="title">Ansible Playbook</div>
            <div class="desc">Playbook 管理与执行记录</div>
          </div>
          <div class="actions">
            <el-button type="primary" icon="Plus" @click="openCreate">新增Playbook</el-button>
            <el-button icon="Refresh" @click="fetchPlaybooks">刷新</el-button>
          </div>
        </div>
      </template>

      <el-table :data="playbooks" v-loading="loading" stripe>
        <el-table-column prop="name" label="名称" min-width="200" />
        <el-table-column prop="description" label="描述" min-width="240" />
        <el-table-column prop="tags" label="Tags" min-width="160" />
        <el-table-column prop="category" label="分类" width="140" />
        <el-table-column label="操作" width="260" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="primary" plain @click="openEdit(row)">编辑</el-button>
            <el-button size="small" type="success" plain @click="openExecute(row)">执行</el-button>
            <el-button size="small" type="danger" plain @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card class="section-card">
      <template #header>
        <div class="header">
          <div>
            <div class="title">执行记录</div>
            <div class="desc">最近执行历史</div>
          </div>
          <div class="actions">
            <el-button icon="Refresh" @click="fetchExecutions">刷新</el-button>
          </div>
        </div>
      </template>
      <el-table :data="executions" v-loading="loadingExecutions" stripe>
        <el-table-column prop="playbook_name" label="Playbook" min-width="200" />
        <el-table-column prop="status" label="状态" width="120">
          <template #default="{ row }">
            <el-tag :type="executionStatusType(row.status)">{{ executionStatusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="started_at" label="开始时间" width="180" />
        <el-table-column prop="finished_at" label="结束时间" width="180" />
        <el-table-column prop="duration" label="耗时" width="100" />
        <el-table-column prop="executor" label="执行人" width="120" />
        <el-table-column label="输出" width="120" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="primary" text @click="openOutput(row)">查看</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>

  <el-dialog append-to-body v-model="dialogVisible" :title="isEdit ? '编辑Playbook' : '新增Playbook'" width="760px">
    <el-form :model="form" label-width="90px">
      <el-form-item label="名称" required>
        <el-input v-model="form.name" />
      </el-form-item>
      <el-form-item label="描述">
        <el-input v-model="form.description" />
      </el-form-item>
      <el-form-item label="Tags">
        <el-input v-model="form.tags" placeholder="逗号分隔" />
      </el-form-item>
      <el-form-item label="分类">
        <el-input v-model="form.category" />
      </el-form-item>
      <el-form-item label="内容" required>
        <el-input v-model="form.content" type="textarea" :rows="12" placeholder="- hosts: all" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="dialogVisible = false">取消</el-button>
      <el-button type="primary" :loading="saving" @click="savePlaybook">保存</el-button>
    </template>
  </el-dialog>

  <el-dialog append-to-body v-model="executeVisible" title="执行Playbook" width="560px">
    <el-form :model="executeForm" label-width="90px">
      <el-form-item label="Inventory">
        <el-select v-model="executeForm.inventory_id" style="width: 100%" clearable>
          <el-option v-for="inv in inventories" :key="inv.id" :label="inv.name" :value="inv.id" />
        </el-select>
      </el-form-item>
      <el-form-item label="Hosts">
        <el-input v-model="executeForm.hosts" placeholder="如：10.0.0.1,10.0.0.2" />
      </el-form-item>
      <el-form-item label="Extra Vars">
        <el-input v-model="executeForm.extra_vars" type="textarea" :rows="4" placeholder='{"env":"prod"}' />
      </el-form-item>
      <el-form-item label="Tags">
        <el-input v-model="executeForm.tags" />
      </el-form-item>
      <el-form-item label="Limit">
        <el-input v-model="executeForm.limit" />
      </el-form-item>
      <el-form-item label="Check">
        <el-switch v-model="executeForm.check" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="executeVisible = false">取消</el-button>
      <el-button type="primary" :loading="executing" @click="executePlaybook">执行</el-button>
    </template>
  </el-dialog>

  <el-dialog append-to-body v-model="outputVisible" title="执行输出" width="760px">
    <pre class="log-block">{{ outputText }}</pre>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, onMounted, onBeforeUnmount } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const playbooks = ref([])
const inventories = ref([])
const executions = ref([])
const loading = ref(false)
const loadingExecutions = ref(false)
const dialogVisible = ref(false)
const isEdit = ref(false)
const currentId = ref('')
const saving = ref(false)

const executeVisible = ref(false)
const executing = ref(false)
const selectedPlaybookId = ref('')

const outputVisible = ref(false)
const outputText = ref('')
let outputSource = null

const form = reactive({
  name: '',
  description: '',
  content: '',
  tags: '',
  category: ''
})

const executeForm = reactive({
  inventory_id: '',
  hosts: '',
  extra_vars: '',
  tags: '',
  limit: '',
  check: false
})

const headers = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const fetchPlaybooks = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/v1/ansible/playbooks', { headers: headers() })
    if (res.data.code === 0) {
      playbooks.value = res.data.data
    }
  } catch (error) {
    ElMessage.error('加载失败')
  } finally {
    loading.value = false
  }
}

const fetchInventories = async () => {
  const res = await axios.get('/api/v1/ansible/inventories', { headers: headers() })
  if (res.data.code === 0) {
    inventories.value = res.data.data
  }
}

const fetchExecutions = async () => {
  loadingExecutions.value = true
  try {
    const res = await axios.get('/api/v1/ansible/executions', { headers: headers() })
    if (res.data.code === 0) {
      executions.value = res.data.data
    }
  } catch (error) {
    ElMessage.error('加载执行记录失败')
  } finally {
    loadingExecutions.value = false
  }
}

const openCreate = () => {
  isEdit.value = false
  currentId.value = ''
  Object.assign(form, { name: '', description: '', content: '', tags: '', category: '' })
  dialogVisible.value = true
}

const openEdit = (row) => {
  isEdit.value = true
  currentId.value = row.id
  Object.assign(form, row)
  dialogVisible.value = true
}

const savePlaybook = async () => {
  if (!form.name || !form.content) {
    ElMessage.warning('请填写名称和内容')
    return
  }
  saving.value = true
  try {
    const url = isEdit.value ? `/api/v1/ansible/playbooks/${currentId.value}` : '/api/v1/ansible/playbooks'
    const method = isEdit.value ? 'put' : 'post'
    const res = await axios({ url, method, data: form, headers: headers() })
    if (res.data.code === 0) {
      ElMessage.success(isEdit.value ? '更新成功' : '创建成功')
      dialogVisible.value = false
      fetchPlaybooks()
    }
  } catch (error) {
    ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

const handleDelete = (row) => {
  ElMessageBox.confirm(`确定删除“${row.name}”吗？`, '提示', { type: 'warning' }).then(async () => {
    await axios.delete(`/api/v1/ansible/playbooks/${row.id}`, { headers: headers() })
    ElMessage.success('删除成功')
    fetchPlaybooks()
  })
}

const openExecute = async (row) => {
  selectedPlaybookId.value = row.id
  Object.assign(executeForm, { inventory_id: '', hosts: '', extra_vars: '', tags: '', limit: '', check: false })
  await fetchInventories()
  executeVisible.value = true
}

const executePlaybook = async () => {
  executing.value = true
  try {
    const payload = {
      inventory_id: executeForm.inventory_id,
      hosts: executeForm.hosts,
      extra_vars: executeForm.extra_vars ? JSON.parse(executeForm.extra_vars) : {},
      tags: executeForm.tags,
      limit: executeForm.limit,
      check: executeForm.check
    }
    const res = await axios.post(`/api/v1/ansible/playbooks/${selectedPlaybookId.value}/execute`, payload, { headers: headers() })
    if (res.data.code === 0) {
      ElMessage.success('已提交执行')
      executeVisible.value = false
      fetchExecutions()
    }
  } catch (error) {
    ElMessage.error('执行失败：请检查Extra Vars格式')
  } finally {
    executing.value = false
  }
}

const openOutput = (row) => {
  outputText.value = ''
  outputVisible.value = true
  if (outputSource) {
    outputSource.close()
  }
  outputSource = new EventSource(`/api/v1/ansible/executions/${row.id}/stream`)
  outputSource.addEventListener('output', (event) => {
    outputText.value += event.data
  })
  outputSource.addEventListener('done', () => {
    outputSource.close()
  })
}

const executionStatusLabel = (status) => {
  const map = { 0: '运行中', 1: '成功', 2: '失败', 3: '取消' }
  return map[status] || '未知'
}

const executionStatusType = (status) => {
  if (status === 1) return 'success'
  if (status === 0) return 'warning'
  if (status === 3) return 'info'
  return 'danger'
}

onMounted(async () => {
  await fetchPlaybooks()
  await fetchExecutions()
})

onBeforeUnmount(() => {
  if (outputSource) {
    outputSource.close()
  }
})
</script>

<style scoped>
.ansible-page { display: flex; flex-direction: column; gap: 16px; }
.section-card { max-width: 100%; margin: 0; }
.header { display: flex; justify-content: space-between; align-items: center; }
.title { font-size: 18px; font-weight: 600; }
.desc { color: #909399; margin-top: 4px; }
.actions { display: flex; gap: 8px; }
.log-block { background: #0f172a; color: #e2e8f0; padding: 12px; border-radius: 6px; font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace; font-size: 12px; white-space: pre-wrap; }
</style>
