<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>AI 运维助手</h2>
        <p class="page-desc">对话问答、日志分析和优化建议。</p>
      </div>
      <div class="page-actions">
        <el-button icon="Refresh" @click="refreshAll">刷新</el-button>
      </div>
    </div>

    <el-tabs v-model="activeTab">
      <el-tab-pane label="智能问答" name="chat">
        <el-row :gutter="12">
          <el-col :md="7" :sm="24">
            <el-card class="h-full">
              <template #header>
                <div class="section-header">
                  <span>会话列表</span>
                  <el-button size="small" icon="Plus" @click="startNewSession">新建</el-button>
                </div>
              </template>
              <el-scrollbar max-height="520px">
                <div
                  v-for="item in sessions"
                  :key="item.id"
                  class="session-item"
                  :class="{ active: item.id === activeSessionId }"
                  @click="selectSession(item)"
                >
                  <div class="session-title">{{ item.title || '未命名会话' }}</div>
                  <div class="session-time">{{ formatTime(item.updated_at) }}</div>
                  <el-button link type="danger" @click.stop="removeSession(item)">删除</el-button>
                </div>
              </el-scrollbar>
            </el-card>
          </el-col>
          <el-col :md="17" :sm="24">
            <el-card class="h-full">
              <template #header>
                <div class="section-header">
                  <span>对话窗口</span>
                  <span class="muted">{{ activeSessionTitle }}</span>
                </div>
              </template>

              <el-scrollbar max-height="430px" class="chat-area">
                <el-empty v-if="!messages.length" description="输入问题开始对话" :image-size="80" />
                <div v-for="msg in messages" :key="msg.id" class="msg" :class="msg.role === 'user' ? 'msg-user' : 'msg-ai'">
                  <div class="msg-role">{{ msg.role === 'user' ? '我' : 'AI' }}</div>
                  <div class="msg-content">{{ msg.content }}</div>
                  <div class="msg-time">{{ formatTime(msg.created_at) }}</div>
                </div>
              </el-scrollbar>

              <div class="chat-input">
                <el-input
                  v-model="chatInput"
                  type="textarea"
                  :rows="3"
                  placeholder="请输入你的问题，例如：如何排查某服务CPU持续过高？"
                />
                <el-input v-model="chatContext" placeholder="上下文（可选）" class="mt-8" />
                <div class="actions-right mt-8">
                  <el-button type="primary" :loading="chatting" @click="sendChat">发送</el-button>
                </div>
              </div>
            </el-card>
          </el-col>
        </el-row>
      </el-tab-pane>

      <el-tab-pane label="日志分析" name="analyze">
        <el-row :gutter="12">
          <el-col :md="14" :sm="24">
            <el-card>
              <template #header>
                <span>日志分析</span>
              </template>
              <el-form label-width="90px">
                <el-form-item label="服务名">
                  <el-input v-model="analyzeForm.service" placeholder="例如: payment-service" />
                </el-form-item>
                <el-form-item label="日志内容" required>
                  <el-input v-model="analyzeForm.logs" type="textarea" :rows="12" placeholder="粘贴日志内容" />
                </el-form-item>
                <el-form-item label="上下文">
                  <el-input v-model="analyzeForm.context" type="textarea" :rows="3" placeholder="补充环境、版本、变更记录等" />
                </el-form-item>
              </el-form>
              <div class="actions-right">
                <el-button type="primary" :loading="analyzing" @click="analyzeLogs">开始分析</el-button>
              </div>
            </el-card>
          </el-col>
          <el-col :md="10" :sm="24">
            <el-card>
              <template #header>
                <span>分析结果</span>
              </template>
              <el-empty v-if="!analysisResult" description="暂无分析结果" :image-size="80" />
              <template v-else>
                <el-descriptions :column="1" border>
                  <el-descriptions-item label="服务">{{ analysisResult.service || analyzeForm.service || '-' }}</el-descriptions-item>
                  <el-descriptions-item label="告警级别">{{ analysisResult.alert_level || '-' }}</el-descriptions-item>
                  <el-descriptions-item label="置信度">{{ analysisResult.confidence || '-' }}</el-descriptions-item>
                  <el-descriptions-item label="日志统计">
                    共 {{ analysisResult.log_count || 0 }} 条，错误 {{ analysisResult.error_count || 0 }}，警告 {{ analysisResult.warning_count || 0 }}
                  </el-descriptions-item>
                  <el-descriptions-item label="根因">{{ analysisResult.root_cause || '-' }}</el-descriptions-item>
                  <el-descriptions-item label="影响">{{ (analysisResult.impact || []).join('；') || '-' }}</el-descriptions-item>
                  <el-descriptions-item label="处理建议">{{ (analysisResult.solutions || []).join('；') || '-' }}</el-descriptions-item>
                </el-descriptions>
              </template>
            </el-card>
          </el-col>
        </el-row>

        <el-card class="mt-12">
          <template #header>
            <span>分析历史</span>
          </template>
          <el-table :data="analysisHistory" v-loading="historyLoading" stripe>
            <el-table-column prop="service" label="服务" width="180" />
            <el-table-column label="告警" width="100">
              <template #default="{ row }">
                <el-tag :type="row.need_alert ? 'danger' : 'success'">{{ row.need_alert ? '需要告警' : '正常' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="alert_level" label="级别" width="100" />
            <el-table-column prop="root_cause" label="根因" min-width="220" show-overflow-tooltip />
            <el-table-column prop="confidence" label="置信度" width="100" />
            <el-table-column label="时间" width="180">
              <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-tab-pane>

      <el-tab-pane label="模型配置" name="config">
        <el-card>
          <template #header>
            <div class="section-header">
              <span>自定义 API 接入</span>
              <el-button size="small" type="primary" icon="Plus" @click="openConfigDialog()">新增配置</el-button>
            </div>
          </template>

          <el-alert
            type="info"
            :closable="false"
            show-icon
            class="mb-12"
            title="支持 OpenAI 兼容 API / 私有网关 / 三方模型代理。"
          />

          <el-descriptions v-if="runtimeConfig" :column="2" border size="small" class="mb-12">
            <el-descriptions-item label="当前 Provider">{{ runtimeConfig.provider || '-' }}</el-descriptions-item>
            <el-descriptions-item label="当前 Model">{{ runtimeConfig.model || '-' }}</el-descriptions-item>
            <el-descriptions-item label="Auth 类型">{{ runtimeConfig.auth_type || '-' }}</el-descriptions-item>
            <el-descriptions-item label="超时">{{ runtimeConfig.timeout_second || 0 }}s</el-descriptions-item>
            <el-descriptions-item label="Base URL" :span="2">{{ runtimeConfig.base_url || '-' }}</el-descriptions-item>
          </el-descriptions>

          <el-table :data="providerConfigs" v-loading="configLoading" stripe>
            <el-table-column prop="name" label="名称" min-width="140" />
            <el-table-column prop="provider" label="Provider" width="110" />
            <el-table-column prop="base_url" label="Base URL" min-width="240" show-overflow-tooltip />
            <el-table-column prop="model" label="Model" min-width="160" show-overflow-tooltip />
            <el-table-column prop="auth_type" label="Auth" width="110" />
            <el-table-column prop="timeout_second" label="超时(s)" width="90" />
            <el-table-column label="当前" width="80">
              <template #default="{ row }">
                <el-tag v-if="row.active" type="success">激活</el-tag>
                <span v-else>-</span>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="330" fixed="right">
              <template #default="{ row }">
                <el-button size="small" @click="openConfigDialog(row)">编辑</el-button>
                <el-button size="small" type="primary" plain @click="activateConfig(row)" :disabled="row.active">设为当前</el-button>
                <el-button size="small" type="warning" plain @click="testConfig(row)">测试</el-button>
                <el-button size="small" type="danger" plain @click="removeConfig(row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-tab-pane>
    </el-tabs>

    <el-dialog v-model="configDialogVisible" :title="configEditing ? '编辑模型配置' : '新增模型配置'" width="700px">
      <el-form :model="configForm" label-width="110px">
        <el-form-item label="配置名称" required>
          <el-input v-model="configForm.name" placeholder="如：OpenAI-Prod" />
        </el-form-item>
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="Provider">
              <el-select v-model="configForm.provider" style="width: 100%">
                <el-option label="openai" value="openai" />
                <el-option label="ollama" value="ollama" />
                <el-option label="custom" value="custom" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="Auth 类型">
              <el-select v-model="configForm.auth_type" style="width: 100%">
                <el-option label="bearer" value="bearer" />
                <el-option label="x-api-key" value="x-api-key" />
                <el-option label="api-key" value="api-key" />
                <el-option label="none" value="none" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="Base URL" required>
          <el-input v-model="configForm.base_url" placeholder="如：https://api.openai.com/v1" />
        </el-form-item>
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="模型" required>
              <el-input v-model="configForm.model" placeholder="如：gpt-4o-mini" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="超时(秒)">
              <el-input-number v-model="configForm.timeout_second" :min="5" :max="300" :step="5" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="API Key">
          <el-input v-model="configForm.api_key" type="password" show-password placeholder="编辑时留空则保持原值" />
        </el-form-item>
        <el-form-item label="附加请求头">
          <el-input
            v-model="configForm.extra_headers"
            type="textarea"
            :rows="3"
            placeholder='JSON对象，如 {"x-tenant":"ops","x-region":"cn"}'
          />
        </el-form-item>
        <el-form-item label="说明">
          <el-input v-model="configForm.description" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="configDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="configSaving" @click="saveConfig">保存</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const activeTab = ref('chat')
const chatting = ref(false)
const analyzing = ref(false)
const historyLoading = ref(false)

const sessions = ref([])
const activeSessionId = ref('')
const messages = ref([])
const chatInput = ref('')
const chatContext = ref('')

const analyzeForm = reactive({
  service: '',
  logs: '',
  context: ''
})
const analysisResult = ref(null)
const analysisHistory = ref([])
const providerConfigs = ref([])
const runtimeConfig = ref(null)
const configLoading = ref(false)
const configSaving = ref(false)
const configDialogVisible = ref(false)
const configEditing = ref(false)
const configForm = reactive({
  id: '',
  name: '',
  provider: 'openai',
  base_url: '',
  model: 'gpt-3.5-turbo',
  auth_type: 'bearer',
  api_key: '',
  timeout_second: 60,
  extra_headers: '{}',
  description: ''
})

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const activeSessionTitle = computed(() => sessions.value.find(item => item.id === activeSessionId.value)?.title || '新会话')

const formatTime = (value) => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString()
}

const fetchSessions = async () => {
  try {
    const res = await axios.get('/api/v1/ai/sessions', { headers: authHeaders() })
    if (res.data?.code === 0) {
      sessions.value = res.data.data || []
      if (!activeSessionId.value && sessions.value.length) {
        await selectSession(sessions.value[0])
      }
    }
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '获取会话失败')
  }
}

const fetchMessages = async () => {
  if (!activeSessionId.value) {
    messages.value = []
    return
  }
  try {
    const res = await axios.get(`/api/v1/ai/sessions/${activeSessionId.value}/messages`, { headers: authHeaders() })
    if (res.data?.code === 0) messages.value = res.data.data || []
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '获取消息失败')
  }
}

const selectSession = async (item) => {
  activeSessionId.value = item.id
  await fetchMessages()
}

const startNewSession = () => {
  activeSessionId.value = ''
  messages.value = []
  chatInput.value = ''
  chatContext.value = ''
}

const sendChat = async () => {
  const message = chatInput.value.trim()
  if (!message) {
    ElMessage.warning('请输入消息')
    return
  }

  chatting.value = true
  try {
    const res = await axios.post('/api/v1/ai/chat', {
      session_id: activeSessionId.value || '',
      message,
      context: chatContext.value?.trim() || ''
    }, { headers: authHeaders() })

    if (res.data?.code === 0 && res.data.data) {
      activeSessionId.value = res.data.data.session_id || activeSessionId.value
      chatInput.value = ''
      await fetchSessions()
      await fetchMessages()
    }
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '发送失败')
  } finally {
    chatting.value = false
  }
}

const removeSession = async (item) => {
  try {
    await ElMessageBox.confirm('确认删除该会话?', '提示', { type: 'warning' })
    await axios.delete(`/api/v1/ai/sessions/${item.id}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    if (activeSessionId.value === item.id) {
      activeSessionId.value = ''
      messages.value = []
    }
    await fetchSessions()
  } catch (err) {
    if (err !== 'cancel') ElMessage.error(err.response?.data?.message || '删除失败')
  }
}

const analyzeLogs = async () => {
  if (!analyzeForm.service.trim()) {
    ElMessage.warning('请输入服务名')
    return
  }
  if (!analyzeForm.logs.trim()) {
    ElMessage.warning('请输入日志内容')
    return
  }
  analyzing.value = true
  try {
    const logs = analyzeForm.logs
      .split('\n')
      .map(item => item.trim())
      .filter(Boolean)
    const res = await axios.post('/api/v1/ai/analyze/logs-detailed', {
      logs,
      service: analyzeForm.service.trim(),
      context: analyzeForm.context
    }, { headers: authHeaders() })

    if (res.data?.code === 0) {
      analysisResult.value = res.data.data || null
      ElMessage.success('分析完成')
      await fetchHistory()
    }
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '分析失败')
  } finally {
    analyzing.value = false
  }
}

const fetchHistory = async () => {
  historyLoading.value = true
  try {
    const res = await axios.get('/api/v1/ai/analyze/history', {
      headers: authHeaders(),
      params: { service: analyzeForm.service || undefined }
    })
    if (res.data?.code === 0) analysisHistory.value = res.data.data || []
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '加载历史失败')
  } finally {
    historyLoading.value = false
  }
}

const fetchConfigs = async () => {
  configLoading.value = true
  try {
    const res = await axios.get('/api/v1/ai/configs', { headers: authHeaders() })
    if (res.data?.code === 0) {
      providerConfigs.value = res.data.data?.configs || []
      runtimeConfig.value = res.data.data?.runtime || null
    }
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '加载模型配置失败')
  } finally {
    configLoading.value = false
  }
}

const resetConfigForm = () => {
  configForm.id = ''
  configForm.name = ''
  configForm.provider = 'openai'
  configForm.base_url = ''
  configForm.model = 'gpt-3.5-turbo'
  configForm.auth_type = 'bearer'
  configForm.api_key = ''
  configForm.timeout_second = 60
  configForm.extra_headers = '{}'
  configForm.description = ''
}

const openConfigDialog = (row) => {
  resetConfigForm()
  configEditing.value = !!row
  if (row) {
    configForm.id = row.id
    configForm.name = row.name || ''
    configForm.provider = row.provider || 'openai'
    configForm.base_url = row.base_url || ''
    configForm.model = row.model || 'gpt-3.5-turbo'
    configForm.auth_type = row.auth_type || 'bearer'
    configForm.timeout_second = Number(row.timeout_second || 60)
    configForm.extra_headers = row.extra_headers || '{}'
    configForm.description = row.description || ''
  }
  configDialogVisible.value = true
}

const saveConfig = async () => {
  if (!configForm.name.trim()) {
    ElMessage.warning('请输入配置名称')
    return
  }
  if (!configForm.base_url.trim()) {
    ElMessage.warning('请输入 Base URL')
    return
  }
  if (!configForm.model.trim()) {
    ElMessage.warning('请输入模型名称')
    return
  }
  let headersPayload = {}
  const rawHeaders = (configForm.extra_headers || '').trim()
  if (rawHeaders) {
    try {
      headersPayload = JSON.parse(rawHeaders)
    } catch {
      ElMessage.warning('附加请求头必须是 JSON 对象')
      return
    }
  }

  configSaving.value = true
  try {
    const payload = {
      name: configForm.name.trim(),
      provider: configForm.provider,
      base_url: configForm.base_url.trim(),
      model: configForm.model.trim(),
      auth_type: configForm.auth_type,
      api_key: configForm.api_key || '',
      timeout_second: Number(configForm.timeout_second || 60),
      extra_headers: headersPayload,
      description: configForm.description
    }
    if (configEditing.value && configForm.id) {
      await axios.put(`/api/v1/ai/configs/${configForm.id}`, payload, { headers: authHeaders() })
      ElMessage.success('配置已更新')
    } else {
      await axios.post('/api/v1/ai/configs', payload, { headers: authHeaders() })
      ElMessage.success('配置已创建')
    }
    configDialogVisible.value = false
    await fetchConfigs()
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '保存配置失败')
  } finally {
    configSaving.value = false
  }
}

const activateConfig = async (row) => {
  try {
    await axios.post(`/api/v1/ai/configs/${row.id}/activate`, {}, { headers: authHeaders() })
    ElMessage.success('已切换为当前模型配置')
    await fetchConfigs()
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '激活失败')
  }
}

const testConfig = async (row) => {
  try {
    const res = await axios.post(`/api/v1/ai/configs/${row.id}/test`, {}, { headers: authHeaders() })
    const info = res.data?.data?.reply ? `，返回: ${String(res.data.data.reply).slice(0, 30)}` : ''
    ElMessage.success(`测试通过${info}`)
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '测试失败')
  }
}

const removeConfig = async (row) => {
  try {
    await ElMessageBox.confirm(`确认删除模型配置 ${row.name} ?`, '提示', { type: 'warning' })
    await axios.delete(`/api/v1/ai/configs/${row.id}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    await fetchConfigs()
  } catch (err) {
    if (err !== 'cancel') ElMessage.error(err.response?.data?.message || '删除失败')
  }
}

const refreshAll = async () => {
  await Promise.all([fetchSessions(), fetchHistory(), fetchConfigs()])
  await fetchMessages()
}

onMounted(refreshAll)
</script>

<style scoped>
.page-card { max-width: 1400px; margin: 0 auto; }
.page-header { display: flex; justify-content: space-between; align-items: center; gap: 12px; margin-bottom: 12px; }
.page-desc { color: #909399; margin: 4px 0 0; }
.page-actions { display: flex; align-items: center; gap: 8px; }
.section-header { display: flex; justify-content: space-between; align-items: center; gap: 8px; }
.h-full { height: 100%; }
.session-item { border: 1px solid #ebeef5; border-radius: 6px; padding: 8px; margin-bottom: 8px; cursor: pointer; }
.session-item.active { border-color: #409eff; background: #ecf5ff; }
.session-title { font-weight: 600; font-size: 13px; }
.session-time { color: #909399; font-size: 12px; margin-top: 4px; margin-bottom: 4px; }
.chat-area { border: 1px solid #ebeef5; border-radius: 6px; padding: 8px; }
.msg { margin-bottom: 10px; padding: 8px 10px; border-radius: 6px; }
.msg-user { background: #ecf5ff; }
.msg-ai { background: #f5f7fa; }
.msg-role { font-weight: 600; font-size: 12px; margin-bottom: 4px; }
.msg-content { white-space: pre-wrap; line-height: 1.5; }
.msg-time { color: #909399; font-size: 12px; margin-top: 4px; }
.chat-input { margin-top: 12px; }
.actions-right { display: flex; justify-content: flex-end; }
.muted { color: #909399; font-size: 12px; }
.mt-8 { margin-top: 8px; }
.mt-12 { margin-top: 12px; }
.mb-12 { margin-bottom: 12px; }
</style>
