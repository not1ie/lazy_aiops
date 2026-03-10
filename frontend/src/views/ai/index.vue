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
        <div class="chat-workspace">
          <div class="chat-sidebar">
            <el-card class="chat-panel chat-session-panel h-full">
              <template #header>
                <div class="section-header">
                  <div>
                    <div class="section-title">会话列表</div>
                    <div class="section-meta">保留最近的智能问答上下文</div>
                  </div>
                  <el-button size="small" icon="Plus" @click="startNewSession">新建</el-button>
                </div>
              </template>
              <el-scrollbar max-height="620px">
                <div
                  v-for="item in sessions"
                  :key="item.id"
                  class="session-item"
                  :class="{ active: item.id === activeSessionId }"
                  @click="selectSession(item)"
                >
                  <div class="session-header-row">
                    <div class="session-title">{{ item.title || '未命名会话' }}</div>
                    <el-button link type="danger" size="small" @click.stop="removeSession(item)">删除</el-button>
                  </div>
                  <div class="session-time">{{ formatTime(item.updated_at) }}</div>
                </div>
              </el-scrollbar>
            </el-card>
          </div>

          <div class="chat-main">
            <el-card class="chat-panel h-full">
              <template #header>
                <div class="section-header">
                  <div>
                    <div class="section-title">对话窗口</div>
                    <div class="section-meta">{{ activeSessionTitle }}</div>
                  </div>
                  <el-button size="small" icon="Plus" plain @click="startNewSession">新会话</el-button>
                </div>
              </template>

              <el-scrollbar ref="chatScrollRef" max-height="500px" class="chat-area">
                <el-empty v-if="!messages.length && !chatting" description="输入问题开始对话" :image-size="80" />
                <div class="chat-stream">
                  <div v-for="msg in messages" :key="msg.id" class="msg-shell" :class="msg.role === 'user' ? 'role-user' : 'role-ai'">
                    <div class="msg-bubble">
                      <div class="msg-meta">
                        <span class="msg-role">{{ msg.role === 'user' ? '我' : 'AI 助手' }}</span>
                        <span class="msg-time">{{ formatTime(msg.created_at) }}</span>
                      </div>
                      <div class="msg-content">{{ msg.content }}</div>
                      <div class="msg-actions">
                        <el-button link size="small" icon="DocumentCopy" @click="copyMessage(msg.content)">复制</el-button>
                        <el-button
                          v-if="msg.role === 'user'"
                          link
                          size="small"
                          icon="EditPen"
                          @click="reuseMessage(msg.content)"
                        >
                          再次编辑
                        </el-button>
                      </div>
                    </div>
                  </div>

                  <div v-if="chatting" class="msg-shell role-ai">
                    <div class="msg-bubble pending-bubble">
                      <div class="msg-meta">
                        <span class="msg-role">AI 助手</span>
                        <span class="msg-time">正在响应</span>
                      </div>
                      <div class="msg-content pending-content">正在整理回复，请稍候…</div>
                      <div class="msg-progress">{{ chatProgressText }}</div>
                    </div>
                  </div>
                </div>
              </el-scrollbar>

              <div class="chat-input-shell">
                <div class="quick-prompts">
                  <span class="muted">快捷提问：</span>
                  <el-tag
                    v-for="item in chatQuickPrompts"
                    :key="item"
                    size="small"
                    effect="plain"
                    class="prompt-tag"
                    @click="applyChatPrompt(item)"
                  >
                    {{ item }}
                  </el-tag>
                </div>
                <div class="runbook-assist">
                  <el-select v-model="chatRunbookTemplate" size="small" class="runbook-select">
                    <el-option v-for="item in runbookTemplates" :key="item.value" :label="item.label" :value="item.value" />
                  </el-select>
                  <el-button size="small" plain @click="applyRunbookTemplate">生成排障 Runbook 提示词</el-button>
                </div>
                <el-input
                  ref="chatInputRef"
                  v-model="chatInput"
                  type="textarea"
                  :rows="4"
                  resize="none"
                  placeholder="请输入你的问题，例如：如何排查某服务 CPU 持续过高？"
                  @keydown.enter.exact.prevent="sendChat"
                />
                <el-input v-model="chatContext" placeholder="上下文（可选）" class="mt-8" />
                <div class="composer-footer mt-8">
                  <div class="muted">Enter 发送，Shift + Enter 换行</div>
                  <div class="actions-right">
                    <el-button type="primary" :loading="chatting" :disabled="chatting" @click="sendChat">{{ chatting ? '处理中...' : '发送' }}</el-button>
                  </div>
                </div>
              </div>
            </el-card>
          </div>
        </div>
      </el-tab-pane>

      <el-tab-pane label="日志分析" name="analyze">
        <el-row :gutter="12">
          <el-col :md="14" :sm="24">
            <el-card>
              <template #header>
                <span>日志分析</span>
              </template>
              <el-form label-width="90px">
                <el-form-item label="日志来源">
                  <el-radio-group v-model="analyzeForm.sourceType">
                    <el-radio-button label="manual">手动粘贴</el-radio-button>
                    <el-radio-button label="service">Docker 服务</el-radio-button>
                    <el-radio-button label="container">Docker 容器</el-radio-button>
                    <el-radio-button label="k8s-pod">K8s Pod</el-radio-button>
                  </el-radio-group>
                </el-form-item>
                <template v-if="analyzeForm.sourceType === 'service' || analyzeForm.sourceType === 'container'">
                  <el-form-item label="主机">
                    <el-select
                      v-model="analyzeForm.hostId"
                      placeholder="选择 Docker 环境"
                      filterable
                      clearable
                      style="width: 100%"
                      @change="onAnalyzeHostChange"
                    >
                      <el-option
                        v-for="host in dockerHosts"
                        :key="host.id"
                        :label="`${host.name || host.id} (${host.status || 'unknown'})`"
                        :value="host.id"
                      />
                    </el-select>
                  </el-form-item>
                  <el-form-item :label="analyzeForm.sourceType === 'service' ? '服务' : '容器'">
                    <div class="log-source-row">
                      <el-select
                        v-model="analyzeForm.targetId"
                        placeholder="选择目标"
                        filterable
                        clearable
                        style="flex: 1"
                        :loading="targetLoading"
                      >
                        <el-option
                          v-for="target in analyzeTargets"
                          :key="target.id"
                          :label="target.label"
                          :value="target.id"
                        />
                      </el-select>
                      <el-input-number v-model="analyzeForm.tail" :min="50" :max="2000" :step="50" controls-position="right" />
                      <el-button :loading="pullingLogs" @click="pullTargetLogs">拉取日志</el-button>
                    </div>
                  </el-form-item>
                </template>
                <template v-if="analyzeForm.sourceType === 'k8s-pod'">
                  <el-form-item label="集群">
                    <el-select
                      v-model="analyzeForm.clusterId"
                      placeholder="选择 K8s 集群"
                      filterable
                      clearable
                      style="width: 100%"
                      @change="onAnalyzeClusterChange"
                    >
                      <el-option
                        v-for="cluster in k8sClusters"
                        :key="cluster.id"
                        :label="cluster.display_name || cluster.name || cluster.id"
                        :value="cluster.id"
                      />
                    </el-select>
                  </el-form-item>
                  <el-form-item label="命名空间">
                    <el-select
                      v-model="analyzeForm.namespace"
                      placeholder="选择命名空间"
                      filterable
                      clearable
                      style="width: 100%"
                      :loading="targetLoading"
                      @change="onAnalyzeNamespaceChange"
                    >
                      <el-option
                        v-for="ns in k8sNamespaces"
                        :key="ns.name"
                        :label="ns.name"
                        :value="ns.name"
                      />
                    </el-select>
                  </el-form-item>
                  <el-form-item label="Pod">
                    <div class="log-source-row">
                      <el-select
                        v-model="analyzeForm.podName"
                        placeholder="选择 Pod"
                        filterable
                        clearable
                        style="flex: 1"
                        :loading="targetLoading"
                        @change="onAnalyzePodChange"
                      >
                        <el-option
                          v-for="pod in k8sPods"
                          :key="pod.name"
                          :label="`${pod.name} (${pod.status || '-'})`"
                          :value="pod.name"
                        />
                      </el-select>
                      <el-input-number v-model="analyzeForm.tail" :min="50" :max="2000" :step="50" controls-position="right" />
                      <el-button :loading="pullingLogs" @click="pullTargetLogs">拉取日志</el-button>
                    </div>
                  </el-form-item>
                  <el-form-item label="容器(可选)">
                    <el-select
                      v-model="analyzeForm.containerName"
                      placeholder="默认首个容器"
                      clearable
                      filterable
                      style="width: 100%"
                    >
                      <el-option v-for="ct in k8sContainers" :key="ct.name" :label="ct.name" :value="ct.name" />
                    </el-select>
                  </el-form-item>
                </template>
                <el-form-item label="分析模板">
                  <div class="log-source-row">
                    <el-select v-model="analyzeForm.template" clearable placeholder="选择一个常见故障场景模板" style="flex: 1">
                      <el-option v-for="tpl in analyzeTemplates" :key="tpl.value" :label="tpl.label" :value="tpl.value" />
                    </el-select>
                    <el-button @click="applyAnalyzeTemplate">应用模板</el-button>
                  </div>
                </el-form-item>
                <el-form-item label="服务名">
                  <el-input v-model="analyzeForm.service" placeholder="例如: payment-service" />
                </el-form-item>
                <el-form-item label="日志内容" required>
                  <el-input v-model="analyzeForm.logs" type="textarea" :rows="12" placeholder="粘贴日志内容" />
                </el-form-item>
                <el-form-item label="上下文">
                  <el-input v-model="analyzeForm.context" type="textarea" :rows="3" placeholder="补充环境、版本、变更记录等" />
                </el-form-item>
                <el-form-item label="Prometheus">
                  <div class="prom-switch">
                    <el-switch v-model="analyzeForm.attachPromContext" />
                    <span class="muted">分析时自动附加该服务最近 5 分钟 CPU/内存摘要</span>
                  </div>
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
          <el-table :fit="true" :data="analysisHistory" v-loading="historyLoading" stripe>
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

          <el-table :fit="true" :data="providerConfigs" v-loading="configLoading" stripe>
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

    <el-dialog append-to-body v-model="configDialogVisible" :title="configEditing ? '编辑模型配置' : '新增模型配置'" width="700px" @closed="handleConfigDialogClosed">
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
          <el-input v-model="configForm.api_key" type="password" show-password :placeholder="configEditing ? '已加载当前 API Key，可直接修改' : '请输入 API Key'" />
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
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getErrorMessage, isCancelError } from '@/utils/error'

const activeTab = ref('chat')
const chatting = ref(false)
const analyzing = ref(false)
const historyLoading = ref(false)
const targetLoading = ref(false)
const pullingLogs = ref(false)

const sessions = ref([])
const activeSessionId = ref('')
const messages = ref([])
const chatInputRef = ref(null)
const chatScrollRef = ref(null)
const chatInput = ref('')
const chatContext = ref('')
const dockerHosts = ref([])
const dockerServices = ref([])
const dockerContainers = ref([])
const k8sClusters = ref([])
const k8sNamespaces = ref([])
const k8sPods = ref([])
const k8sContainers = ref([])
const chatQuickPrompts = [
  '帮我梳理当前异常的排障路径',
  '给我一份 10 分钟止血方案',
  '把可能根因按概率从高到低排序',
  '给出可直接执行的检查命令清单'
]
const chatRunbookTemplate = ref('k8s_crashloop')
const chatProgressText = ref('等待发送请求')
const runbookTemplates = [
  { value: 'k8s_crashloop', label: 'K8s CrashLoopBackOff' },
  { value: 'k8s_notready', label: 'K8s Pod NotReady' },
  { value: 'docker_restart', label: 'Docker 容器重启风暴' },
  { value: 'latency_spike', label: '接口延迟突增' }
]
const analyzeTemplates = [
  { value: 'cpu', label: 'CPU 持续升高' },
  { value: 'oom', label: '内存上涨/OOM' },
  { value: 'timeout', label: '接口超时/延迟抖动' },
  { value: 'db', label: '数据库连接异常' }
]

const analyzeForm = reactive({
  sourceType: 'manual',
  template: '',
  hostId: '',
  targetId: '',
  clusterId: '',
  namespace: '',
  podName: '',
  containerName: '',
  tail: 300,
  service: '',
  logs: '',
  context: '',
  attachPromContext: true
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
const analyzeTargets = computed(() => (analyzeForm.sourceType === 'service' ? dockerServices.value : dockerContainers.value))

const formatTime = (value) => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString()
}

const progressPhases = [
  '已提交请求，正在构建上下文',
  '上下文已准备，正在调用大模型',
  '模型已返回，正在整理最终回复'
]
let progressTimer = null

const startChatProgress = () => {
  let phaseIndex = 0
  const startedAt = Date.now()
  chatProgressText.value = progressPhases[phaseIndex]
  clearInterval(progressTimer)
  progressTimer = window.setInterval(() => {
    phaseIndex = Math.min(phaseIndex + 1, progressPhases.length - 1)
    const elapsed = Math.max(1, Math.floor((Date.now() - startedAt) / 1000))
    chatProgressText.value = `${progressPhases[phaseIndex]} · 已等待 ${elapsed}s`
  }, 1400)
}

const stopChatProgress = () => {
  clearInterval(progressTimer)
  progressTimer = null
  chatProgressText.value = '等待发送请求'
}

const scrollChatToBottom = async () => {
  await nextTick()
  const scrollbar = chatScrollRef.value
  scrollbar?.setScrollTop?.(10 ** 8)
}

const focusChatInput = async () => {
  await nextTick()
  chatInputRef.value?.focus?.()
}

const copyMessage = async (content) => {
  try {
    await navigator.clipboard.writeText(String(content || ''))
    ElMessage.success('已复制到剪贴板')
  } catch {
    ElMessage.error('复制失败，请检查浏览器权限')
  }
}

const reuseMessage = async (content) => {
  chatInput.value = String(content || '')
  await focusChatInput()
  ElMessage.success('已将内容放回输入框')
}

const fetchSessions = async () => {
  try {
    const res = await axios.get('/api/v1/ai/sessions', { headers: authHeaders() })
    if (res.data?.code === 0) {
      sessions.value = res.data.data || []
      const activeExists = sessions.value.some(item => item.id === activeSessionId.value)
      if (activeSessionId.value && !activeExists) {
        activeSessionId.value = ''
        messages.value = []
      }
      if (!activeSessionId.value && sessions.value.length) {
        await selectSession(sessions.value[0])
      } else if (!sessions.value.length) {
        messages.value = []
      }
    }
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '获取会话失败'))
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
    ElMessage.error(getErrorMessage(err, '获取消息失败'))
  }
}

const selectSession = async (item) => {
  activeSessionId.value = item.id
  await fetchMessages()
  await scrollChatToBottom()
}

const startNewSession = async () => {
  try {
    const res = await axios.post('/api/v1/ai/sessions', { title: '新会话' }, { headers: authHeaders() })
    if (res.data?.code === 0 && res.data.data?.id) {
      activeSessionId.value = res.data.data.id
      chatInput.value = ''
      chatContext.value = ''
      await fetchSessions()
      await fetchMessages()
      await scrollChatToBottom()
      await focusChatInput()
      return
    }
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '创建会话失败'))
  }
  // 兜底：保留本地清空态
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

  const riskyKeywords = ['删除', '重启', '回滚', '扩容', '缩容', '停止', 'drain', 'rm -rf', 'drop table']
  const riskyHit = riskyKeywords.find((item) => message.toLowerCase().includes(item.toLowerCase()))
  if (riskyHit) {
    try {
      await ElMessageBox.confirm(
        `检测到高风险关键词「${riskyHit}」，建议你先让 AI 输出“只读检查步骤 + 回滚方案 + 执行前置条件”。是否继续发送？`,
        '高风险操作提醒',
        { type: 'warning', confirmButtonText: '继续发送', cancelButtonText: '取消' }
      )
    } catch {
      return
    }
  }

  chatting.value = true
  startChatProgress()
  await scrollChatToBottom()
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
      await scrollChatToBottom()
      await focusChatInput()
    }
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '发送失败'))
  } finally {
    chatting.value = false
    stopChatProgress()
  }
}

const applyChatPrompt = (text) => {
  if (!text) return
  if (chatInput.value.trim()) {
    chatInput.value = `${chatInput.value.trim()}\n${text}`
  } else {
    chatInput.value = text
  }
}

const buildRunbookContext = () => {
  if (analyzeForm.sourceType === 'k8s-pod') {
    const cluster = k8sClusters.value.find((item) => item.id === analyzeForm.clusterId)
    return `目标: cluster=${cluster?.display_name || cluster?.name || '-'}, ns=${analyzeForm.namespace || '-'}, pod=${analyzeForm.podName || '-'}, container=${analyzeForm.containerName || '-'}`
  }
  if (analyzeForm.sourceType === 'service' || analyzeForm.sourceType === 'container') {
    const host = dockerHosts.value.find((item) => item.id === analyzeForm.hostId)
    const target = analyzeTargets.value.find((item) => item.id === analyzeForm.targetId)
    return `目标: docker_host=${host?.name || host?.id || '-'}, object=${target?.label || '-'}`
  }
  return '目标: 请先补充环境、版本、最近变更与异常时间窗口'
}

const applyRunbookTemplate = () => {
  const context = buildRunbookContext()
  const promptMap = {
    k8s_crashloop: `请输出 K8s CrashLoopBackOff 的分层 Runbook：\n1) 只读检查命令（kubectl describe/logs/events）\n2) 根因判断分支与判定条件\n3) 10分钟止血动作（最小变更）\n4) 永久修复建议\n5) 回滚与验证清单\n${context}`,
    k8s_notready: `请输出 K8s Pod NotReady 排障 Runbook：\n1) 调度/节点/网络/DNS 四层检查命令\n2) 每一步预期结果与异常分支\n3) 可执行修复动作（带风险说明）\n4) 复盘指标与告警阈值建议\n${context}`,
    docker_restart: `请输出 Docker 容器重启风暴 Runbook：\n1) restart policy / 健康检查 / 退出码核查\n2) 日志与资源指标关联分析\n3) 先止血再修复的执行顺序\n4) 回滚条件与自动化脚本建议\n${context}`,
    latency_spike: `请输出接口延迟突增 Runbook：\n1) 网关、应用、DB、依赖服务逐层定位\n2) 每层最关键3条命令与阈值\n3) 快速降级与限流建议\n4) 事后优化项（容量、连接池、缓存）\n${context}`
  }
  const prompt = promptMap[chatRunbookTemplate.value]
  if (!prompt) return
  applyChatPrompt(prompt)
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
    if (!isCancelError(err)) ElMessage.error(getErrorMessage(err, '删除失败'))
  }
}

const fetchDockerHosts = async () => {
  try {
    const res = await axios.get('/api/v1/docker/hosts', { headers: authHeaders() })
    if (res.data?.code === 0) {
      dockerHosts.value = res.data.data || []
      if (!analyzeForm.hostId && dockerHosts.value.length) {
        analyzeForm.hostId = dockerHosts.value[0].id
      }
    }
  } catch (err) {
    dockerHosts.value = []
  }
}

const fetchK8sClusters = async () => {
  try {
    const res = await axios.get('/api/v1/k8s/clusters', { headers: authHeaders() })
    if (res.data?.code === 0) {
      k8sClusters.value = res.data.data || []
      if (!analyzeForm.clusterId && k8sClusters.value.length) {
        analyzeForm.clusterId = k8sClusters.value[0].id
      }
    }
  } catch (err) {
    k8sClusters.value = []
  }
}

const loadDockerTargets = async () => {
  if (!analyzeForm.hostId) {
    dockerServices.value = []
    dockerContainers.value = []
    analyzeForm.targetId = ''
    return
  }
  targetLoading.value = true
  try {
    const [servicesRes, containersRes] = await Promise.all([
      axios.get(`/api/v1/docker/hosts/${analyzeForm.hostId}/services`, { headers: authHeaders() }),
      axios.get(`/api/v1/docker/hosts/${analyzeForm.hostId}/containers`, { headers: authHeaders() })
    ])
    const services = servicesRes.data?.code === 0 ? (servicesRes.data.data || []) : []
    const containers = containersRes.data?.code === 0 ? (containersRes.data.data || []) : []
    dockerServices.value = services.map((item) => ({
      id: item.ID || item.id,
      label: `${item.Name || item.name || item.ID || item.id} (${item.Replicas || '-'})`
    }))
    dockerContainers.value = containers.map((item) => {
      const name = Array.isArray(item.names) ? (item.names[0] || item.id) : (item.name || item.id)
      return {
        id: item.id,
        label: `${name || item.id} (${item.status || '-'})`
      }
    })
    if (analyzeForm.targetId) {
      const exists = analyzeTargets.value.some((item) => item.id === analyzeForm.targetId)
      if (!exists) analyzeForm.targetId = ''
    }
  } catch (err) {
    dockerServices.value = []
    dockerContainers.value = []
    analyzeForm.targetId = ''
  } finally {
    targetLoading.value = false
  }
}

const loadK8sNamespaces = async () => {
  if (!analyzeForm.clusterId) {
    k8sNamespaces.value = []
    analyzeForm.namespace = ''
    return
  }
  targetLoading.value = true
  try {
    const res = await axios.get(`/api/v1/k8s/clusters/${analyzeForm.clusterId}/namespaces`, { headers: authHeaders() })
    const list = res.data?.code === 0 ? (res.data.data || []) : []
    k8sNamespaces.value = list
    if (!analyzeForm.namespace && k8sNamespaces.value.length) {
      analyzeForm.namespace = k8sNamespaces.value[0].name
    }
  } catch (err) {
    k8sNamespaces.value = []
    analyzeForm.namespace = ''
  } finally {
    targetLoading.value = false
  }
}

const loadK8sPods = async () => {
  if (!analyzeForm.clusterId || !analyzeForm.namespace) {
    k8sPods.value = []
    k8sContainers.value = []
    analyzeForm.podName = ''
    analyzeForm.containerName = ''
    return
  }
  targetLoading.value = true
  try {
    const res = await axios.get(`/api/v1/k8s/clusters/${analyzeForm.clusterId}/namespaces/${analyzeForm.namespace}/pods`, {
      headers: authHeaders()
    })
    k8sPods.value = res.data?.code === 0 ? (res.data.data || []) : []
    if (!k8sPods.value.some((p) => p.name === analyzeForm.podName)) {
      analyzeForm.podName = ''
      analyzeForm.containerName = ''
      k8sContainers.value = []
    }
  } catch (err) {
    k8sPods.value = []
    k8sContainers.value = []
    analyzeForm.podName = ''
    analyzeForm.containerName = ''
  } finally {
    targetLoading.value = false
  }
}

const loadAnalyzeTargets = async () => {
  if (analyzeForm.sourceType === 'service' || analyzeForm.sourceType === 'container') {
    await loadDockerTargets()
    return
  }
  if (analyzeForm.sourceType === 'k8s-pod') {
    await loadK8sNamespaces()
    await loadK8sPods()
  }
}

const onAnalyzeHostChange = async () => {
  analyzeForm.targetId = ''
  await loadDockerTargets()
}

const onAnalyzeClusterChange = async () => {
  analyzeForm.namespace = ''
  analyzeForm.podName = ''
  analyzeForm.containerName = ''
  await loadK8sNamespaces()
  await loadK8sPods()
}

const onAnalyzeNamespaceChange = async () => {
  analyzeForm.podName = ''
  analyzeForm.containerName = ''
  await loadK8sPods()
}

const onAnalyzePodChange = () => {
  const pod = k8sPods.value.find((item) => item.name === analyzeForm.podName)
  const containers = Array.isArray(pod?.containers) ? pod.containers : []
  k8sContainers.value = containers
  if (containers.length === 1) {
    analyzeForm.containerName = containers[0].name
  } else if (!containers.some((c) => c.name === analyzeForm.containerName)) {
    analyzeForm.containerName = ''
  }
}

const selectedTargetLabel = () => {
  if (analyzeForm.sourceType === 'k8s-pod') return analyzeForm.podName || ''
  const target = analyzeTargets.value.find((item) => item.id === analyzeForm.targetId)
  if (!target) return ''
  return String(target.label || '').split('(')[0].trim()
}

const applyAnalyzeTemplate = () => {
  const tpl = analyzeForm.template
  if (!tpl) return
  const serviceName = analyzeForm.service?.trim() || selectedTargetLabel()
  if (serviceName) analyzeForm.service = serviceName

  const templateMap = {
    cpu: '症状: CPU 持续高于 80%。请重点判断是否存在热点接口、无限重试、批任务突增、GC 抖动，并给出可验证步骤。',
    oom: '症状: 内存持续上涨或 OOMKilled。请分析内存泄漏、缓存膨胀、请求堆积、对象生命周期问题，并给出止血建议。',
    timeout: '症状: 接口超时与延迟波动。请分析上下游依赖、连接池、DNS/网络抖动、线程池耗尽等因素，并给出排查顺序。',
    db: '症状: 数据库连接错误/慢查询。请分析连接池、锁等待、慢 SQL、突发流量与配置项风险，并给出优化建议。'
  }
  const append = templateMap[tpl] || ''
  if (!append) return
  analyzeForm.context = analyzeForm.context ? `${analyzeForm.context}\n${append}` : append
}

const buildSourceContext = () => {
  if (analyzeForm.sourceType === 'service' || analyzeForm.sourceType === 'container') {
    const host = dockerHosts.value.find((item) => item.id === analyzeForm.hostId)
    const target = analyzeTargets.value.find((item) => item.id === analyzeForm.targetId)
    return `日志来源: docker/${analyzeForm.sourceType}, 主机=${host?.name || host?.id || '-'}, 目标=${target?.label || '-'}, tail=${analyzeForm.tail}`
  }
  if (analyzeForm.sourceType === 'k8s-pod') {
    const cluster = k8sClusters.value.find((item) => item.id === analyzeForm.clusterId)
    const containerName = analyzeForm.containerName || '-'
    return `日志来源: k8s/pod, 集群=${cluster?.display_name || cluster?.name || analyzeForm.clusterId || '-'}, 命名空间=${analyzeForm.namespace || '-'}, Pod=${analyzeForm.podName || '-'}, 容器=${containerName}, tail=${analyzeForm.tail}`
  }
  return '日志来源: manual/paste'
}

const pullTargetLogs = async () => {
  let endpoint = ''
  let params = { tail: analyzeForm.tail, timestamps: 1 }
  let warnText = ''

  if (analyzeForm.sourceType === 'service' || analyzeForm.sourceType === 'container') {
    if (!analyzeForm.hostId) {
      ElMessage.warning('请先选择主机')
      return
    }
    if (!analyzeForm.targetId) {
      ElMessage.warning(`请先选择${analyzeForm.sourceType === 'service' ? '服务' : '容器'}`)
      return
    }
    endpoint = analyzeForm.sourceType === 'service'
      ? `/api/v1/docker/hosts/${analyzeForm.hostId}/services/${analyzeForm.targetId}/logs`
      : `/api/v1/docker/hosts/${analyzeForm.hostId}/containers/${analyzeForm.targetId}/logs`
    warnText = analyzeForm.sourceType === 'service'
      ? '未获取到服务日志（请确认当前节点是 Swarm manager）'
      : '未获取到容器日志'
  } else if (analyzeForm.sourceType === 'k8s-pod') {
    if (!analyzeForm.clusterId || !analyzeForm.namespace || !analyzeForm.podName) {
      ElMessage.warning('请先选择集群、命名空间和 Pod')
      return
    }
    endpoint = `/api/v1/k8s/clusters/${analyzeForm.clusterId}/namespaces/${analyzeForm.namespace}/pods/${analyzeForm.podName}/logs`
    params = { tail: analyzeForm.tail }
    if (analyzeForm.containerName) params.container = analyzeForm.containerName
    warnText = '未获取到 Pod 日志'
  } else {
    ElMessage.warning('手动粘贴模式无需拉取日志')
    return
  }

  pullingLogs.value = true
  try {
    const res = await axios.get(endpoint, { headers: authHeaders(), params })
    if (res.data?.code === 0) {
      analyzeForm.logs = String(res.data.data || '').trim()
      if (!analyzeForm.logs) {
        ElMessage.warning(warnText)
        return
      }
      if (!analyzeForm.service.trim()) {
        analyzeForm.service = selectedTargetLabel()
      }
      ElMessage.success('日志拉取成功')
    } else {
      ElMessage.error(res.data?.message || '日志拉取失败')
    }
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '日志拉取失败'))
  } finally {
    pullingLogs.value = false
  }
}

const escapePromRegex = (v) => String(v || '').replace(/[.*+?^${}()|[\]\\]/g, '\\$&')

const fetchPromInstant = async (query) => {
  const res = await axios.get('/api/v1/monitor/prometheus/query', {
    headers: authHeaders(),
    params: { query }
  })
  if (res.data?.status === 'success') {
    return res.data?.data?.result || []
  }
  return []
}

const buildPromContext = async (serviceName) => {
  const key = escapePromRegex(serviceName)
  if (!key) return ''
  const cpuQuery = `sum(rate(container_cpu_usage_seconds_total{name=~".*${key}.*"}[5m]))`
  const memQuery = `sum(container_memory_working_set_bytes{name=~".*${key}.*"}) / 1024 / 1024`
  const [cpuRes, memRes] = await Promise.allSettled([fetchPromInstant(cpuQuery), fetchPromInstant(memQuery)])
  const cpu = cpuRes.status === 'fulfilled' ? Number(cpuRes.value?.[0]?.value?.[1] || 0) : 0
  const mem = memRes.status === 'fulfilled' ? Number(memRes.value?.[0]?.value?.[1] || 0) : 0
  if (cpu <= 0 && mem <= 0) return ''
  return `Prometheus 摘要(最近5分钟): CPU=${cpu.toFixed(3)} 核, 内存=${mem.toFixed(1)} MiB`
}

const analyzeLogs = async () => {
  if (!analyzeForm.logs.trim()) {
    ElMessage.warning('请输入日志内容')
    return
  }
  const resolvedService = analyzeForm.service.trim() || selectedTargetLabel() || 'unknown-service'
  analyzing.value = true
  try {
    const logs = analyzeForm.logs
      .split('\n')
      .map(item => item.trim())
      .filter(Boolean)
    let contextText = analyzeForm.context || ''
    const sourceContext = buildSourceContext()
    if (sourceContext) {
      contextText = contextText ? `${contextText}\n${sourceContext}` : sourceContext
    }
    if (analyzeForm.attachPromContext) {
      const promContext = await buildPromContext(resolvedService)
      if (promContext) {
        contextText = contextText ? `${contextText}\n${promContext}` : promContext
      }
    }
    const res = await axios.post('/api/v1/ai/analyze/logs-detailed', {
      logs,
      service: resolvedService,
      context: contextText
    }, { headers: authHeaders() })

    if (res.data?.code === 0) {
      analysisResult.value = res.data.data || null
      ElMessage.success('分析完成')
      await fetchHistory()
    }
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '分析失败'))
  } finally {
    analyzing.value = false
  }
}

const fetchHistory = async () => {
  historyLoading.value = true
  try {
    const service = analyzeForm.service || selectedTargetLabel() || undefined
    const res = await axios.get('/api/v1/ai/analyze/history', {
      headers: authHeaders(),
      params: { service }
    })
    if (res.data?.code === 0) analysisHistory.value = res.data.data || []
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '加载历史失败'))
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
    ElMessage.error(getErrorMessage(err, '加载模型配置失败'))
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

const handleConfigDialogClosed = () => {
  configEditing.value = false
  resetConfigForm()
}

const openConfigDialog = async (row) => {
  resetConfigForm()
  configEditing.value = !!row
  if (row) {
    try {
      const res = await axios.get(`/api/v1/ai/configs/${row.id}`, { headers: authHeaders() })
      if (res.data.code === 0) {
        const data = res.data.data || {}
        configForm.id = data.id
        configForm.name = data.name || ''
        configForm.provider = data.provider || 'openai'
        configForm.base_url = data.base_url || ''
        configForm.model = data.model || 'gpt-3.5-turbo'
        configForm.auth_type = data.auth_type || 'bearer'
        configForm.api_key = data.api_key || ''
        configForm.timeout_second = Number(data.timeout_second || 60)
        configForm.extra_headers = data.extra_headers || '{}'
        configForm.description = data.description || ''
      }
    } catch (err) {
      ElMessage.error(getErrorMessage(err, '加载模型配置详情失败'))
      return
    }
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
    ElMessage.error(getErrorMessage(err, '保存配置失败'))
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
    ElMessage.error(getErrorMessage(err, '激活失败'))
  }
}

const testConfig = async (row) => {
  try {
    const res = await axios.post(`/api/v1/ai/configs/${row.id}/test`, {}, { headers: authHeaders() })
    const info = res.data?.data?.reply ? `，返回: ${String(res.data.data.reply).slice(0, 30)}` : ''
    ElMessage.success(`测试通过${info}`)
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '测试失败'))
  }
}

const removeConfig = async (row) => {
  try {
    await ElMessageBox.confirm(`确认删除模型配置 ${row.name} ?`, '提示', { type: 'warning' })
    await axios.delete(`/api/v1/ai/configs/${row.id}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    await fetchConfigs()
  } catch (err) {
    if (!isCancelError(err)) ElMessage.error(getErrorMessage(err, '删除失败'))
  }
}

const refreshAll = async () => {
  await Promise.all([fetchSessions(), fetchHistory(), fetchConfigs(), fetchDockerHosts(), fetchK8sClusters()])
  await loadAnalyzeTargets()
  await fetchMessages()
  await scrollChatToBottom()
}

watch(() => analyzeForm.sourceType, async () => {
  analyzeForm.targetId = ''
  analyzeForm.podName = ''
  analyzeForm.containerName = ''
  await loadAnalyzeTargets()
})

onMounted(refreshAll)

onBeforeUnmount(() => {
  stopChatProgress()
})
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.page-header { display: flex; justify-content: space-between; align-items: center; gap: 12px; margin-bottom: 12px; }
.page-desc { color: var(--muted-text); margin: 4px 0 0; }
.page-actions { display: flex; align-items: center; gap: 8px; }
.section-header { display: flex; justify-content: space-between; align-items: center; gap: 8px; }
.section-title { font-size: 15px; font-weight: 700; }
.section-meta { color: var(--muted-text); font-size: 12px; margin-top: 4px; }
.h-full { height: 100%; }
.chat-workspace { display: grid; grid-template-columns: 320px minmax(0, 1fr); gap: 16px; }
.chat-panel { background: var(--card-bg); }
.chat-session-panel :deep(.el-card__body) { padding-top: 14px; }
.session-item {
  border: 1px solid var(--el-border-color-light);
  border-radius: 16px;
  padding: 12px 14px;
  margin-bottom: 10px;
  cursor: pointer;
  background: rgba(255, 255, 255, 0.4);
  transition: border-color 0.2s ease, background-color 0.2s ease, transform 0.2s ease;
}
.session-item:hover { transform: translateY(-1px); }
.session-item.active {
  border-color: rgba(10, 132, 255, 0.4);
  background: rgba(10, 132, 255, 0.08);
  box-shadow: 0 12px 28px rgba(10, 132, 255, 0.08);
}
.session-header-row { display: flex; align-items: flex-start; justify-content: space-between; gap: 8px; }
.session-title { font-weight: 700; font-size: 14px; }
.session-time { color: var(--muted-text); font-size: 12px; margin-top: 8px; }
.chat-area {
  border: 1px solid var(--el-border-color-light);
  border-radius: 20px;
  padding: 12px;
  background: rgba(255, 255, 255, 0.32);
}
.chat-stream { display: flex; flex-direction: column; gap: 14px; }
.msg-shell { display: flex; }
.role-user { justify-content: flex-end; }
.role-ai { justify-content: flex-start; }
.msg-bubble {
  max-width: min(78%, 860px);
  padding: 14px 16px;
  border-radius: 20px;
  border: 1px solid var(--el-border-color-light);
  background: rgba(255, 255, 255, 0.58);
  box-shadow: 0 10px 22px rgba(15, 23, 42, 0.06);
}
.role-user .msg-bubble {
  background: linear-gradient(180deg, rgba(36, 146, 255, 0.18) 0%, rgba(36, 146, 255, 0.1) 100%);
  border-top-right-radius: 8px;
}
.role-ai .msg-bubble {
  border-top-left-radius: 8px;
}
.pending-bubble {
  border-style: dashed;
}
.msg-meta { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-bottom: 8px; }
.msg-role { font-weight: 700; font-size: 12px; letter-spacing: 0.02em; }
.msg-content { white-space: pre-wrap; line-height: 1.75; font-size: 14px; }
.msg-time { color: var(--muted-text); font-size: 12px; }
.msg-actions { display: flex; align-items: center; gap: 8px; margin-top: 10px; }
.msg-progress { color: var(--muted-text); font-size: 12px; margin-top: 8px; }
.pending-content { color: var(--el-text-color-secondary); }
.chat-input-shell {
  margin-top: 14px;
  padding: 14px;
  border-radius: 20px;
  border: 1px solid var(--el-border-color-light);
  background: rgba(255, 255, 255, 0.44);
}
.quick-prompts { display: flex; gap: 6px; align-items: center; flex-wrap: wrap; margin-bottom: 10px; }
.prompt-tag { cursor: pointer; border-radius: 999px; }
.runbook-assist { display: flex; gap: 8px; align-items: center; margin-bottom: 10px; flex-wrap: wrap; }
.runbook-select { min-width: 240px; }
.composer-footer { display: flex; justify-content: space-between; align-items: center; gap: 12px; }
.actions-right { display: flex; justify-content: flex-end; }
.log-source-row { display: flex; width: 100%; gap: 8px; align-items: center; }
.prom-switch { display: flex; align-items: center; gap: 8px; }
.muted { color: var(--muted-text); font-size: 12px; }
.mt-8 { margin-top: 8px; }
.mt-12 { margin-top: 12px; }
.mb-12 { margin-bottom: 12px; }

:global(html[data-theme='dark'] .session-item),
:global(html[data-theme='dark'] .chat-area),
:global(html[data-theme='dark'] .chat-input-shell),
:global(html[data-theme='dark'] .msg-bubble) {
  background: rgba(15, 23, 42, 0.58);
}

:global(html[data-theme='dark'] .role-user .msg-bubble) {
  background: linear-gradient(180deg, rgba(36, 146, 255, 0.2) 0%, rgba(36, 146, 255, 0.12) 100%);
}

@media (max-width: 1200px) {
  .chat-workspace { grid-template-columns: 1fr; }
}

@media (max-width: 768px) {
  .msg-bubble { max-width: 100%; }
  .composer-footer { flex-direction: column; align-items: stretch; }
  .runbook-select { min-width: 100%; }
}
</style>
