<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>运维知识库</h2>
        <p class="page-desc">Runbook、故障复盘、运维指南的沉淀与 AI 智能检索。</p>
      </div>
      <div class="page-actions">
        <el-button type="warning" icon="Share" @click="handleOpenSources">外部数据源同步</el-button>
        <el-button type="primary" icon="Plus" @click="handleAdd">新增文档</el-button>
        <el-button icon="Refresh" @click="fetchList">刷新</el-button>
      </div>
    </div>

    <!-- Search / Ask -->
    <div class="ask-bar">
      <el-input
        v-model="askQuestion"
        placeholder="向知识库提问，例如：MySQL 连接数过高怎么处理？"
        class="ask-input"
        clearable
        @keyup.enter="handleAsk"
      >
        <template #prepend><el-icon><MagicStick /></el-icon> AI 问答</template>
        <template #append><el-button :loading="asking" @click="handleAsk" icon="Position" /></template>
      </el-input>
    </div>

    <!-- AI Answer -->
    <el-alert v-if="aiAnswer" :title="'AI 答复'" type="success" :closable="true" @close="aiAnswer = null" class="mb-16">
      <div class="ai-answer-content">{{ aiAnswer }}</div>
      <div v-if="aiRefs.length > 0" class="ai-refs">
        <span class="ref-label">参考文档：</span>
        <el-tag v-for="ref in aiRefs" :key="ref.id" size="small" type="info" effect="plain" style="margin-right: 4px;">{{ ref.title }}</el-tag>
      </div>
    </el-alert>

    <!-- Filters -->
    <div class="toolbar">
      <el-input v-model="keyword" placeholder="搜索标题/内容..." class="w-52" clearable @change="fetchList" />
      <el-select v-model="category" placeholder="分类" class="w-40" clearable @change="fetchList">
        <el-option label="Runbook" value="runbook" />
        <el-option label="故障复盘" value="post-mortem" />
        <el-option label="运维指南" value="guide" />
        <el-option label="其他" value="other" />
      </el-select>
    </div>

    <!-- Doc List -->
    <el-table :data="docs" v-loading="loading" stripe>
      <el-table-column prop="title" label="标题" min-width="200">
        <template #default="{ row }">
          <el-button link type="primary" @click="handleView(row)">{{ row.title }}</el-button>
        </template>
      </el-table-column>
      <el-table-column prop="category" label="分类" width="100">
        <template #default="{ row }">
          <el-tag size="small" :type="catType(row.category)" effect="plain">{{ row.category || 'other' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="tags" label="标签" width="180">
        <template #default="{ row }">
          <el-tag v-for="t in tagList(row.tags)" :key="t" size="small" effect="plain" style="margin-right: 4px;">{{ t }}</el-tag>
          <span v-if="!row.tags">-</span>
        </template>
      </el-table-column>
      <el-table-column prop="created_by" label="作者" width="100" />
      <el-table-column label="更新时间" width="160">
        <template #default="{ row }">{{ new Date(row.updated_at).toLocaleString() }}</template>
      </el-table-column>
      <el-table-column label="操作" width="160" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="handleEdit(row)">编辑</el-button>
          <el-button link type="danger" @click="handleDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-empty v-if="docs.length === 0 && !loading" description="知识库为空，点击上方按钮添加第一篇文档" />

    <!-- Form Dialog -->
    <el-dialog :title="form.id ? '编辑文档' : '新增文档'" v-model="dialogVisible" width="680px" append-to-body>
      <el-form :model="form" label-width="80px">
        <el-form-item label="标题" required>
          <el-input v-model="form.title" placeholder="如：MySQL 连接池耗尽紧急处理流程" />
        </el-form-item>
        <el-form-item label="分类">
          <el-select v-model="form.category" style="width: 100%">
            <el-option label="Runbook" value="runbook" />
            <el-option label="故障复盘" value="post-mortem" />
            <el-option label="运维指南" value="guide" />
            <el-option label="其他" value="other" />
          </el-select>
        </el-form-item>
        <el-form-item label="标签">
          <el-input v-model="form.tags" placeholder="逗号分隔，如：mysql,connection,emergency" />
        </el-form-item>
        <el-form-item label="内容" required>
          <el-input v-model="form.content" type="textarea" :rows="14" placeholder="Markdown 格式，包括问题描述、排查步骤、解决方案、回滚方案等" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleSave">保存</el-button>
      </template>
    </el-dialog>

    <!-- View Dialog -->
    <el-dialog :title="viewDoc.title" v-model="viewVisible" width="720px" append-to-body>
      <div class="doc-meta">
        <el-tag size="small" :type="catType(viewDoc.category)" effect="plain">{{ viewDoc.category }}</el-tag>
        <span class="doc-author">作者: {{ viewDoc.created_by }}</span>
        <span class="doc-time">{{ new Date(viewDoc.updated_at).toLocaleString() }}</span>
      </div>
      <div class="doc-content" v-html="renderedContent"></div>
    </el-dialog>

    <!-- 外部数据源管理 Dialog -->
    <el-dialog title="外部知识库同步管理" v-model="sourcesDialogVisible" width="800px" append-to-body>
      <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px;">
        <span style="font-size: 13px; color: var(--el-text-color-secondary);">支持配置 Confluence, Jira 等常见外部知识源，一键拉取并自动生成 AI 检索文档。</span>
        <el-button type="primary" size="small" icon="Plus" @click="sourceFormVisible = true">添加数据源</el-button>
      </div>

      <el-table :data="sources" v-loading="loadingSources" stripe size="small">
        <el-table-column prop="name" label="数据源名称" min-width="120" />
        <el-table-column prop="type" label="类型" width="110">
          <template #default="{ row }">
            <el-tag size="small" effect="plain" type="success">{{ row.type.toUpperCase() }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="url" label="数据源地址" min-width="160" show-overflow-tooltip />
        <el-table-column prop="sync_status" label="同步状态" width="100">
          <template #default="{ row }">
            <el-tag size="small" :type="syncStatusType(row.sync_status)">
              {{ syncStatusLabel(row.sync_status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="上一次同步时间" width="160">
          <template #default="{ row }">
            {{ row.last_synced_at ? new Date(row.last_synced_at).toLocaleString() : '从无' }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="140" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" :loading="row.sync_status === 'syncing'" @click="handleSyncSource(row)">同步</el-button>
            <el-button link type="danger" size="small" @click="handleDeleteSource(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 新建数据源 Dialog (嵌套) -->
      <el-dialog title="添加外部知识源" v-model="sourceFormVisible" width="500px" append-to-body>
        <el-form :model="sourceForm" label-width="100px" size="small">
          <el-form-item label="名称" required>
            <el-input v-model="sourceForm.name" placeholder="如：Confluence 生产排障 Wiki" />
          </el-form-item>
          <el-form-item label="类型" required>
            <el-select v-model="sourceForm.type" style="width: 100%">
              <el-option label="Confluence" value="confluence" />
              <el-option label="Jira (故障单)" value="jira" />
              <el-option label="GitLab Wiki" value="gitlab" />
              <el-option label="Notion" value="notion" />
            </el-select>
          </el-form-item>
          <el-form-item label="地址 / URL" required>
            <el-input v-model="sourceForm.url" placeholder="如: http://wiki.company.com (设为 mock 可离线演示)" />
          </el-form-item>
          <el-form-item label="Token / 凭证">
            <el-input v-model="sourceForm.token" type="password" placeholder="可不填，或填写 API Bearer Token" show-password />
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button size="small" @click="sourceFormVisible = false">取消</el-button>
          <el-button size="small" type="primary" :loading="savingSource" @click="handleSaveSource">保存</el-button>
        </template>
      </el-dialog>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const loading = ref(false)
const docs = ref([])
const keyword = ref('')
const category = ref('')
const askQuestion = ref('')
const asking = ref(false)
const aiAnswer = ref(null)
const aiRefs = ref([])

const dialogVisible = ref(false)
const saving = ref(false)
const form = ref({ title: '', category: 'runbook', tags: '', content: '' })

const viewVisible = ref(false)
const viewDoc = ref({})

const renderedContent = computed(() => {
  const md = viewDoc.value.content || ''
  return md.replace(/\n/g, '<br>').replace(/## (.+)/g, '<h3>$1</h3>').replace(/`(.+?)`/g, '<code>$1</code>')
})

const catType = (c) => {
  if (c === 'runbook') return 'success'
  if (c === 'post-mortem') return 'danger'
  if (c === 'guide') return 'primary'
  return 'info'
}

const tagList = (tags) => {
  if (!tags) return []
  return tags.split(',').map(t => t.trim()).filter(Boolean)
}

const fetchList = async () => {
  loading.value = true
  try {
    const params = {}
    if (keyword.value) params.keyword = keyword.value
    if (category.value) params.category = category.value
    const res = await axios.get('/api/v1/knowledge/docs', { headers: authHeaders(), params })
    docs.value = res.data?.data || []
  } catch (err) {
    ElMessage.error('加载知识库失败')
  } finally {
    loading.value = false
  }
}

const handleAsk = async () => {
  const q = askQuestion.value.trim()
  if (!q) return
  asking.value = true
  try {
    const res = await axios.post('/api/v1/knowledge/ask', { question: q }, { headers: authHeaders() })
    const data = res.data?.data
    aiAnswer.value = data?.answer || '未找到相关答案'
    aiRefs.value = data?.references || []
  } catch (err) {
    ElMessage.error('问答失败: ' + (err.response?.data?.message || err.message))
  } finally {
    asking.value = false
  }
}

const handleAdd = () => {
  form.value = { title: '', category: 'runbook', tags: '', content: '' }
  dialogVisible.value = true
}

const handleEdit = (row) => {
  form.value = { ...row }
  dialogVisible.value = true
}

const handleView = (row) => {
  viewDoc.value = row
  viewVisible.value = true
}

const handleSave = async () => {
  if (!form.value.title || !form.value.content) return ElMessage.warning('标题和内容为必填')
  saving.value = true
  try {
    if (form.value.id) {
      await axios.put(`/api/v1/knowledge/docs/${form.value.id}`, form.value, { headers: authHeaders() })
    } else {
      await axios.post('/api/v1/knowledge/docs', form.value, { headers: authHeaders() })
    }
    ElMessage.success('保存成功')
    dialogVisible.value = false
    fetchList()
  } catch (err) {
    ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

const handleDelete = async (row) => {
  try {
    await ElMessageBox.confirm('确认删除该文档？', '提示', { type: 'warning' })
    await axios.delete(`/api/v1/knowledge/docs/${row.id}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    fetchList()
  } catch (err) { /* cancel */ }
}

// 外部数据源变量与方法
const sourcesDialogVisible = ref(false)
const sources = ref([])
const loadingSources = ref(false)
const sourceFormVisible = ref(false)
const savingSource = ref(false)
const sourceForm = ref({ name: '', type: 'confluence', url: 'mock', token: '' })

const syncStatusType = (status) => {
  if (status === 'syncing') return 'warning'
  if (status === 'success') return 'success'
  if (status === 'failed') return 'danger'
  return 'info'
}

const syncStatusLabel = (status) => {
  if (status === 'syncing') return '同步中...'
  if (status === 'success') return '同步成功'
  if (status === 'failed') return '同步失败'
  return '未同步'
}

const fetchSources = async () => {
  loadingSources.value = true
  try {
    const res = await axios.get('/api/v1/knowledge/sources', { headers: authHeaders() })
    sources.value = res.data?.data || []
  } catch (err) {
    ElMessage.error('加载外部知识源失败')
  } finally {
    loadingSources.value = false
  }
}

const handleOpenSources = () => {
  sourcesDialogVisible.value = true
  fetchSources()
}

const handleSaveSource = async () => {
  if (!sourceForm.value.name || !sourceForm.value.url) {
    return ElMessage.warning('名称和URL为必填')
  }
  savingSource.value = true
  try {
    await axios.post('/api/v1/knowledge/sources', sourceForm.value, { headers: authHeaders() })
    ElMessage.success('创建外部数据源成功')
    sourceFormVisible.value = false
    sourceForm.value = { name: '', type: 'confluence', url: 'mock', token: '' }
    fetchSources()
  } catch (err) {
    ElMessage.error('创建外部数据源失败')
  } finally {
    savingSource.value = false
  }
}

const handleDeleteSource = async (row) => {
  try {
    await ElMessageBox.confirm(`确认删除数据源 [${row.name}] ？`, '提示', { type: 'warning' })
    await axios.delete(`/api/v1/knowledge/sources/${row.id}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    fetchSources()
  } catch (err) { /* cancel */ }
}

const handleSyncSource = async (row) => {
  row.sync_status = 'syncing'
  try {
    await axios.post(`/api/v1/knowledge/sources/${row.id}/sync`, {}, { headers: authHeaders() })
    ElMessage.success('已触发一键拉取同步，请稍后刷新')
    
    // 轮询 3 秒获取最新状态并刷新知识列表，这样让体验极其爽快！
    setTimeout(async () => {
      await fetchSources()
      await fetchList()
    }, 2500)
  } catch (err) {
    ElMessage.error('触发同步失败')
    row.sync_status = 'failed'
  }
}

onMounted(fetchList)
</script>

<style scoped>
.page-card { max-width: 100%; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 16px; }
.page-desc { color: var(--el-text-color-secondary); margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; }
.ask-bar { margin-bottom: 16px; }
.ask-input :deep(.el-input-group__prepend) { font-weight: 700; }
.mb-16 { margin-bottom: 16px; }
.toolbar { display: flex; gap: 12px; margin-bottom: 16px; }
.w-52 { width: 220px; }
.w-40 { width: 160px; }
.doc-meta { display: flex; align-items: center; gap: 16px; margin-bottom: 20px; font-size: 13px; color: var(--el-text-color-secondary); }
.doc-content { line-height: 1.8; max-height: 500px; overflow-y: auto; }
.doc-content :deep(h3) { margin: 16px 0 8px; font-size: 15px; }
.doc-content :deep(code) { background: var(--el-fill-color); padding: 2px 6px; border-radius: 4px; font-size: 13px; }
.ai-answer-content { line-height: 1.7; }
.ai-refs { margin-top: 12px; }
.ref-label { font-size: 12px; color: var(--el-text-color-secondary); margin-right: 4px; }
</style>
