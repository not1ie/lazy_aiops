<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>告警规则</h2>
        <p class="page-desc">管理告警规则并配置阈值与通知。</p>
      </div>
      <div class="page-actions">
        <el-button type="primary" icon="Plus" @click="openCreate">新增规则</el-button>
        <el-button icon="Refresh" @click="fetchRules">刷新</el-button>
      </div>
    </div>

    <el-table :data="rules" stripe style="width: 100%">
      <el-table-column prop="name" label="名称" min-width="160" />
      <el-table-column prop="type" label="类型" width="120" />
      <el-table-column prop="target" label="目标" min-width="200" />
      <el-table-column prop="metric" label="指标" width="140" />
      <el-table-column prop="operator" label="操作符" width="90" />
      <el-table-column prop="threshold" label="阈值" width="100" />
      <el-table-column prop="severity" label="级别" width="120" />
      <el-table-column prop="enabled" label="启用" width="90">
        <template #default="scope">
          <el-switch v-model="scope.row.enabled" @change="toggleRule(scope.row)" />
        </template>
      </el-table-column>
      <el-table-column label="操作" width="200">
        <template #default="scope">
          <el-button size="small" @click="openEdit(scope.row)">编辑</el-button>
          <el-button size="small" type="danger" @click="removeRule(scope.row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="720px">
      <el-form :model="form" label-width="110px">
        <el-form-item label="名称">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="类型">
          <el-select v-model="form.type" class="w-52">
            <el-option label="host" value="host" />
            <el-option label="k8s" value="k8s" />
            <el-option label="domain" value="domain" />
            <el-option label="ssl" value="ssl" />
            <el-option label="custom" value="custom" />
          </el-select>
        </el-form-item>
        <el-form-item label="目标">
          <el-input v-model="form.target" />
        </el-form-item>
        <el-form-item label="指标">
          <el-input v-model="form.metric" />
        </el-form-item>
        <el-form-item label="操作符">
          <el-select v-model="form.operator" class="w-40">
            <el-option label=">" value=">" />
            <el-option label=">=" value=">=" />
            <el-option label="<" value="<" />
            <el-option label="<=" value="<=" />
            <el-option label="==" value="==" />
            <el-option label="!=" value="!=" />
          </el-select>
        </el-form-item>
        <el-form-item label="阈值">
          <el-input v-model="form.threshold" />
        </el-form-item>
        <el-form-item label="持续(秒)">
          <el-input-number v-model="form.duration" :min="0" />
        </el-form-item>
        <el-form-item label="级别">
          <el-select v-model="form.severity" class="w-40">
            <el-option label="critical" value="critical" />
            <el-option label="warning" value="warning" />
            <el-option label="info" value="info" />
          </el-select>
        </el-form-item>
        <el-form-item label="通知组">
          <el-select v-model="form.notify_group_id" class="w-52" filterable clearable>
            <el-option v-for="g in groups" :key="g.id" :label="g.name" :value="g.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="通知模板">
          <el-select v-model="selectedTemplateId" class="w-52" filterable clearable>
            <el-option v-for="t in templates" :key="t.id" :label="t.name" :value="t.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>

      <el-divider />

      <div class="preview-block">
        <div class="preview-title">通知预览</div>
        <div class="preview-actions">
          <el-switch v-model="markdownEnabled" active-text="Markdown预览" />
          <el-button size="small" @click="testNotify" :disabled="!form.notify_group_id">测试通知</el-button>
        </div>
        <div class="preview-hint" v-pre>
          可用变量：{{.name}} {{.type}} {{.target}} {{.metric}} {{.operator}} {{.threshold}} {{.duration}} {{.severity}} {{.description}}
        </div>
        <el-input v-model="previewTitle" readonly />
        <el-input v-model="previewContent" type="textarea" :rows="6" readonly />
        <div v-if="markdownEnabled" class="preview-markdown" v-html="previewHtml"></div>
      </div>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitForm">保存</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import { marked } from 'marked'
import DOMPurify from 'dompurify'

const rules = ref([])
const dialogVisible = ref(false)
const dialogTitle = ref('')
const isEdit = ref(false)
const currentId = ref('')
const groups = ref([])
const templates = ref([])
const selectedTemplateId = ref('')
const markdownEnabled = ref(true)
const form = ref({
  name: '',
  type: 'host',
  target: '',
  metric: '',
  operator: '>',
  threshold: '',
  duration: 0,
  severity: 'warning',
  notify_group_id: '',
  enabled: true,
  description: ''
})

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const fetchRules = async () => {
  const res = await axios.get('/api/v1/alert/rules', { headers: authHeaders() })
  rules.value = res.data.data || []
}

const fetchGroups = async () => {
  const res = await axios.get('/api/v1/notify/groups', { headers: authHeaders() })
  groups.value = res.data.data || []
}

const fetchTemplates = async () => {
  const res = await axios.get('/api/v1/notify/templates', { headers: authHeaders() })
  templates.value = res.data.data || []
}

const openCreate = () => {
  isEdit.value = false
  dialogTitle.value = '新增规则'
  form.value = { name: '', type: 'host', target: '', metric: '', operator: '>', threshold: '', duration: 0, severity: 'warning', notify_group_id: '', enabled: true, description: '' }
  dialogVisible.value = true
}

const openEdit = (row) => {
  isEdit.value = true
  dialogTitle.value = '编辑规则'
  currentId.value = row.id
  form.value = { ...row }
  dialogVisible.value = true
}

const submitForm = async () => {
  if (isEdit.value) {
    await axios.put(`/api/v1/alert/rules/${currentId.value}`, form.value, { headers: authHeaders() })
    ElMessage.success('更新成功')
  } else {
    await axios.post('/api/v1/alert/rules', form.value, { headers: authHeaders() })
    ElMessage.success('创建成功')
  }
  dialogVisible.value = false
  fetchRules()
}

const toggleRule = async (row) => {
  await axios.put(`/api/v1/alert/rules/${row.id}`, row, { headers: authHeaders() })
}

const removeRule = async (row) => {
  await ElMessageBox.confirm(`确认删除规则 ${row.name} 吗？`, '提示', { type: 'warning' })
  await axios.delete(`/api/v1/alert/rules/${row.id}`, { headers: authHeaders() })
  ElMessage.success('删除成功')
  fetchRules()
}

onMounted(fetchRules)
onMounted(fetchGroups)
onMounted(fetchTemplates)

const previewTitle = computed(() => {
  const tpl = templates.value.find(t => t.id === selectedTemplateId.value)
  const raw = tpl?.title || `告警规则预览: ${form.value.name || '未命名规则'}`
  return applyTemplate(raw)
})

const previewContent = computed(() => {
  const tpl = templates.value.find(t => t.id === selectedTemplateId.value)
  const raw = tpl?.content || [
    `规则: ${form.value.name || '-'}`,
    `类型: ${form.value.type || '-'}`,
    `目标: ${form.value.target || '-'}`,
    `指标: ${form.value.metric || '-'}`,
    `条件: ${form.value.operator || ''} ${form.value.threshold || ''}`,
    `持续: ${form.value.duration || 0}s`,
    `级别: ${form.value.severity || '-'}`,
    `描述: ${form.value.description || '-'}`,
  ].join('\\n')
  return applyTemplate(raw)
})

const previewHtml = computed(() => {
  const html = marked.parse(previewContent.value || '')
  return DOMPurify.sanitize(html)
})

const applyTemplate = (text) => {
  const vars = {
    name: form.value.name || '-',
    type: form.value.type || '-',
    target: form.value.target || '-',
    metric: form.value.metric || '-',
    operator: form.value.operator || '',
    threshold: form.value.threshold || '',
    duration: form.value.duration || 0,
    severity: form.value.severity || '-',
    description: form.value.description || '-'
  }
  return text.replace(/\\{\\{\\.(\\w+)\\}\\}/g, (_, key) => {
    if (Object.prototype.hasOwnProperty.call(vars, key)) return String(vars[key])
    return ''
  })
}

const testNotify = async () => {
  if (!form.value.notify_group_id) return
  await axios.post(`/api/v1/notify/groups/${form.value.notify_group_id}/test`, {
    title: previewTitle.value,
    content: previewContent.value
  }, { headers: authHeaders() })
  ElMessage.success('测试通知已发送')
}
</script>

<style scoped>
.page-card { max-width: 1180px; margin: 0 auto; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 16px; }
.page-desc { color: #606266; margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; }
.w-52 { width: 220px; }
.w-40 { width: 160px; }
.preview-block { display: flex; flex-direction: column; gap: 8px; }
.preview-title { font-weight: 600; }
.preview-actions { display: flex; justify-content: flex-end; }
.preview-hint { color: #909399; font-size: 12px; }
.preview-markdown {
  border: 1px solid #ebeef5;
  padding: 12px;
  border-radius: 6px;
  background: #fafafa;
  font-size: 13px;
  line-height: 1.6;
}
</style>
