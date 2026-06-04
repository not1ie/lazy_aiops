<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>AI 技能管理</h2>
        <p class="page-desc">定义 AI Agent 的系统提示词、参数定义以及可调用的运维动作工具。</p>
      </div>
      <div class="page-actions">
        <el-button type="primary" icon="Plus" @click="handleAdd">新增技能</el-button>
        <el-button icon="Refresh" @click="getList">刷新</el-button>
      </div>
    </div>

    <div class="page-body">
      <el-table v-loading="loading" :data="list" border stripe>
        <el-table-column prop="name" label="技能名称" width="180" />
        <el-table-column prop="description" label="描述" />
        <el-table-column label="系统预设" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="row.is_system ? 'info' : 'success'" effect="plain">{{ row.is_system ? '是' : '否' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" align="center" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="handleRun(row)">执行测试</el-button>
            <el-button type="primary" link @click="handleEdit(row)" :disabled="row.is_system">编辑</el-button>
            <el-button type="danger" link @click="handleDelete(row)" :disabled="row.is_system">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 表单弹窗 -->
    <el-dialog :title="form.id ? '编辑技能' : '新增技能'" v-model="dialogVisible" width="650px" append-to-body>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="110px">
        <el-form-item label="名称" prop="name">
          <el-input v-model="form.name" placeholder="请输入技能名称，如 check_node_load" />
        </el-form-item>
        <el-form-item label="描述" prop="description">
          <el-input v-model="form.description" placeholder="简单描述该技能的作用" />
        </el-form-item>
        <el-form-item label="系统提示词" prop="system_prompt">
          <el-input type="textarea" :rows="5" v-model="form.system_prompt" placeholder="定义 AI 扮演的角色和回复逻辑" />
        </el-form-item>
        <el-form-item label="绑定动作" prop="tool_bindings">
          <el-input v-model="form.tool_bindings" placeholder='JSON数组，例如 ["get_k8s_pods", "search_hosts"]' />
        </el-form-item>
        <el-form-item label="参数定义" prop="parameters_schema">
          <el-input type="textarea" :rows="3" v-model="form.parameters_schema" placeholder="输入参数的 JSON Schema (用于前端表单生成，可选)" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitForm">确定保存</el-button>
      </template>
    </el-dialog>

    <!-- 执行弹窗 -->
    <el-dialog title="测试执行技能" v-model="runDialogVisible" width="600px" append-to-body>
      <el-form label-width="110px">
        <el-alert type="info" :closable="false" show-icon title="模拟输入参数，系统将展示 AI 决策过程和最终答复。" class="mb-12" />
        <el-form-item label="输入参数 (JSON)">
          <el-input type="textarea" :rows="6" v-model="runParams" placeholder='{"host": "192.168.1.1"}' />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="runDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="running" @click="submitRun">开始执行</el-button>
      </template>
    </el-dialog>

    <!-- 执行结果抽屉 -->
    <el-drawer title="执行轨迹与结果" v-model="resultDrawerVisible" size="45%" append-to-body>
      <div v-if="runResult" class="drawer-content">
        <div class="result-section">
          <h3><el-icon><ChatLineRound /></el-icon> AI 答复</h3>
          <div class="reply-box">{{ runResult.reply }}</div>
        </div>
        
        <div class="result-section">
          <h3><el-icon><Connection /></el-icon> 动作轨迹 (Tool Traces)</h3>
          <el-table :data="runResult.tool_calls || []" border size="small">
            <el-table-column prop="name" label="工具" width="140" />
            <el-table-column prop="status" label="状态" width="90">
              <template #default="{ row }">
                <el-tag :type="row.status === 'success' ? 'success' : 'danger'" size="small">{{ row.status }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="summary" label="执行摘要" show-overflow-tooltip />
          </el-table>
        </div>
      </div>
    </el-drawer>
  </el-card>
</template>

<style scoped>
.page-body {
  margin-top: 24px;
}
.drawer-content {
  padding: 0 24px 24px;
}
.result-section h3 {
  margin: 24px 0 12px;
  font-size: 15px;
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--el-text-color-primary);
}
.reply-box {
  background: var(--el-fill-color-lighter);
  padding: 20px;
  border-radius: 14px;
  line-height: 1.6;
  white-space: pre-wrap;
  color: var(--el-text-color-primary);
  border: 1px solid var(--el-border-color-lighter);
  box-shadow: inset 0 1px 4px rgba(0, 0, 0, 0.02);
}
.mb-12 {
  margin-bottom: 12px;
}
:deep(.el-table) {
  border: none;
}
:deep(.el-table th.el-table__cell) {
  background-color: transparent !important;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
</style>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import axios from 'axios'

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const loading = ref(false)
const list = ref([])
const dialogVisible = ref(false)
const formRef = ref(null)
const form = ref({})

const runDialogVisible = ref(false)
const resultDrawerVisible = ref(false)
const running = ref(false)
const runParams = ref('{}')
const currentRunSkill = ref(null)
const runResult = ref(null)

const rules = {
  name: [{ required: true, message: '必填', trigger: 'blur' }],
  system_prompt: [{ required: true, message: '必填', trigger: 'blur' }]
}

const getList = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/v1/ai/skills', { headers: authHeaders() })
    list.value = res.data?.data || res.data || []
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '获取技能列表失败')
  } finally {
    loading.value = false
  }
}

const handleAdd = () => {
  form.value = { tool_bindings: '[]' }
  dialogVisible.value = true
}

const handleEdit = (row) => {
  form.value = { ...row }
  dialogVisible.value = true
}

const handleDelete = (row) => {
  ElMessageBox.confirm('确认删除该技能?', '提示', { type: 'warning' }).then(async () => {
    await axios.delete(`/api/v1/ai/skills/${row.id}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    getList()
  })
}

const submitForm = async () => {
  if (!formRef.value) return
  await formRef.value.validate()
  
  const method = form.value.id ? 'put' : 'post'
  const url = form.value.id ? `/api/v1/ai/skills/${form.value.id}` : '/api/v1/ai/skills'
  await axios[method](url, form.value, { headers: authHeaders() })
  ElMessage.success('保存成功')
  dialogVisible.value = false
  getList()
}

const handleRun = (row) => {
  currentRunSkill.value = row
  runParams.value = '{}'
  runResult.value = null
  runDialogVisible.value = true
}

const submitRun = async () => {
  let params = {}
  try {
    params = JSON.parse(runParams.value)
  } catch (e) {
    return ElMessage.error('参数必须为有效 JSON')
  }

  running.value = true
  try {
    const res = await axios.post(`/api/v1/ai/skills/${currentRunSkill.value.id}/run`, { parameters: params }, { headers: authHeaders() })
    runResult.value = res.data?.data || res.data
    runDialogVisible.value = false
    resultDrawerVisible.value = true
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '执行失败')
  } finally {
    running.value = false
  }
}

onMounted(() => {
  getList()
})
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>