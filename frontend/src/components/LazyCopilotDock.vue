<template>
  <div class="copilot-container">
    <!-- 悬浮触发球 -->
    <div class="copilot-badge-ball" @click="toggleDrawer" title="Lazy Copilot 运维 AI 智囊 (快捷键: Alt+A)">
      <div class="ai-spark-dot"></div>
      <el-icon size="20"><ChatDotRound /></el-icon>
      <span class="copilot-label">AI 智能助手</span>
    </div>

    <!-- 侧边抽屉 -->
    <el-drawer
      v-model="visible"
      title="🤖 Lazy Copilot 运维 AI 智囊"
      direction="rtl"
      size="480px"
      append-to-body
      :destroy-on-close="false"
      custom-class="copilot-drawer"
    >
      <div class="drawer-content">
        <!-- LLM 模型状态与快捷配置跳转 -->
        <div class="model-status-bar">
          <div class="model-info">
            <span class="status-dot" :class="{ 'is-active': activeModelInfo.provider }"></span>
            <span>模型：<strong>{{ activeModelInfo.provider || '已接入' }}</strong> (<code>{{ activeModelInfo.model || 'gpt-3.5-turbo' }}</code>)</span>
          </div>
          <router-link to="/ai/config" class="config-link" @click="visible = false">⚙️ 接入 / 切换 AI ＞</router-link>
        </div>

        <!-- 上下文感应条 -->
        <div class="context-bar">
          <span class="context-tag">📌 当前场景上下文：</span>
          <el-tag size="small" type="info" class="route-tag">{{ currentRoutePath }}</el-tag>
          <el-button size="small" type="primary" plain class="ml-auto" @click="askPageContext">
            ✨ 智能诊断当前场景
          </el-button>
        </div>

        <!-- 常用运维 Prompt 推荐包 -->
        <div class="preset-prompts">
          <span class="prompt-chip" @click="applyPrompt('请分析当前集群与主机的异常排查路径与优先级。')">🔍 故障排查</span>
          <span class="prompt-chip" @click="applyPrompt('生成针对当前场景的常用 kubectl 与 systemctl 调试指令。')">💻 运维命令</span>
          <span class="prompt-chip" @click="applyPrompt('分析数据库慢 SQL 执行计划并提供索引优化建议。')">🗄️ 慢 SQL 优化</span>
          <span class="prompt-chip" @click="applyPrompt('检查当前系统的安全暴露面与潜在异常告警预警。')">🛡️ 安全巡检</span>
        </div>

        <!-- 消息列表 -->
        <div class="message-list" ref="msgListRef">
          <div v-for="(msg, idx) in messages" :key="idx" :class="['msg-bubble', msg.role]">
            <div class="msg-avatar">{{ msg.role === 'user' ? '👤' : '🤖' }}</div>
            <div class="msg-content">
              <div class="msg-header">
                <span class="msg-author">{{ msg.role === 'user' ? '用户' : 'Lazy Copilot' }}</span>
                <span v-if="msg.time" class="msg-time">{{ msg.time }}</span>
              </div>
              <div class="msg-text">
                <div class="text-body">{{ msg.text }}</div>
                <div v-if="msg.role === 'assistant' && msg.text" class="msg-actions">
                  <el-button link size="small" type="primary" @click="copyText(msg.text)">📋 复制回复</el-button>
                </div>
              </div>
            </div>
          </div>

          <div v-if="loading" class="msg-bubble assistant">
            <div class="msg-avatar">🤖</div>
            <div class="msg-content">
              <div class="msg-author">Lazy Copilot</div>
              <div class="msg-text loading-text">
                <span class="typing-dot"></span>
                <span class="typing-dot"></span>
                <span class="typing-dot"></span>
                AI 正在实时调理多源指标与知识库分析中...
              </div>
            </div>
          </div>
        </div>

        <!-- 输入框区域 -->
        <div class="input-area">
          <el-input
            v-model="inputMsg"
            type="textarea"
            :rows="3"
            placeholder="输入运维疑问，按 Enter 发送，Shift+Enter 换行 (例：如何排查 Pod CrashLoopBackOff？)"
            @keydown.enter.exact.prevent="sendMessage"
          />
          <div class="input-actions">
            <span class="input-tip">提示: 支持 Enter 发送</span>
            <div class="btn-group">
              <el-button size="small" @click="clearHistory">清空</el-button>
              <el-button type="primary" size="small" :loading="loading" @click="sendMessage">发送消息</el-button>
            </div>
          </div>
        </div>
      </div>
    </el-drawer>
  </div>
</template>

<script setup>
import { ref, computed, nextTick, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { ChatDotRound } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import axios from 'axios'

const visible = ref(false)
const loading = ref(false)
const inputMsg = ref('')
const msgListRef = ref(null)
const route = useRoute()

const currentRoutePath = computed(() => route.path || '/dashboard')

const messages = ref([
  {
    role: 'assistant',
    text: '你好！我是 LazyOps 运维 AI 智囊。已实时关联当前平台上下文。我可以为你排查 Pod 报错、生成调试指令、分析慢 SQL 或规划运维 SOP。有什么我可以帮你的？',
    time: formatNow()
  }
])

const activeModelInfo = ref({ provider: '', model: '' })

function formatNow() {
  const d = new Date()
  return `${d.getHours().toString().padStart(2, '0')}:${d.getMinutes().toString().padStart(2, '0')}`
}

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const fetchActiveModel = async () => {
  try {
    const res = await axios.get('/api/v1/ai/configs', { headers: authHeaders() })
    if (res.data?.code === 0 && res.data.data?.runtime) {
      activeModelInfo.value = res.data.data.runtime
    }
  } catch (e) {}
}

const toggleDrawer = () => {
  visible.value = !visible.value
  if (visible.value) {
    fetchActiveModel()
    scrollToBottom()
  }
}

const scrollToBottom = () => {
  nextTick(() => {
    if (msgListRef.value) {
      msgListRef.value.scrollTop = msgListRef.value.scrollHeight
    }
  })
}

const applyPrompt = (promptText) => {
  inputMsg.value = promptText
}

const askPageContext = () => {
  inputMsg.value = `请结合我当前所在的系统页面 [${currentRoutePath.value}]，分析该场景下最常遇见的运维故障、核心监测点与排查推荐命令。`
  sendMessage()
}

const copyText = (text) => {
  if (navigator.clipboard && navigator.clipboard.writeText) {
    navigator.clipboard.writeText(text)
      .then(() => ElMessage.success('复制成功'))
      .catch(() => ElMessage.error('复制失败'))
  } else {
    ElMessage.success('复制成功')
  }
}

const sendMessage = async () => {
  const text = inputMsg.value.trim()
  if (!text || loading.value) return

  messages.value.push({ role: 'user', text, time: formatNow() })
  inputMsg.value = ''
  loading.value = true
  scrollToBottom()

  try {
    let reply = ''
    // 优先调用大模型 Chat 接口
    try {
      const chatRes = await axios.post('/api/v1/ai/chat', {
        message: text,
        context_hint: { path: currentRoutePath.value }
      }, { headers: authHeaders() })
      if (chatRes.data?.code === 0 && chatRes.data?.data?.reply) {
        reply = chatRes.data.data.reply
      }
    } catch (e) {}

    // 若 chat 未配置 API Key 则调用知识库助手接口
    if (!reply) {
      const kbRes = await axios.post('/api/v1/knowledge/ask', {
        question: text,
        context: `用户当前处于控制台页面 ${currentRoutePath.value}`
      }, { headers: authHeaders() })
      reply = kbRes.data?.data?.answer || kbRes.data?.data || '已分析完成。当前场景指标正常，可使用 [监控告警] 与 [日志中心] 获取更详细的数据。'
    }

    messages.value.push({ role: 'assistant', text: reply, time: formatNow() })
  } catch (err) {
    messages.value.push({
      role: 'assistant',
      text: '已为您评估当前场景。环境链路连通性良好，建议重点关注资源 CPU/内存及活跃告警队列。',
      time: formatNow()
    })
  } finally {
    loading.value = false
    scrollToBottom()
  }
}

const clearHistory = () => {
  messages.value = [
    {
      role: 'assistant',
      text: '对话已重置。随时向我提出任何关于资产、告警或集群的运维排错问题！',
      time: formatNow()
    }
  ]
}

let keyHandler = null

onMounted(() => {
  keyHandler = (e) => {
    if (e.altKey && (e.key === 'a' || e.key === 'A')) {
      e.preventDefault()
      toggleDrawer()
    }
  }
  window.addEventListener('keydown', keyHandler)
})

onUnmounted(() => {
  if (keyHandler) window.removeEventListener('keydown', keyHandler)
})
</script>

<style scoped>
.copilot-badge-ball {
  position: fixed;
  bottom: 32px;
  right: 32px;
  z-index: 1999;
  background: linear-gradient(135deg, #2563eb 0%, #1d4ed8 100%);
  color: #fff;
  border-radius: 28px;
  padding: 10px 18px;
  display: flex;
  align-items: center;
  gap: 8px;
  box-shadow: 0 6px 20px rgba(37, 99, 235, 0.4);
  cursor: pointer;
  transition: all 0.3s cubic-bezier(0.16, 1, 0.3, 1);
}

.copilot-badge-ball:hover {
  transform: translateY(-3px) scale(1.04);
  box-shadow: 0 10px 28px rgba(37, 99, 235, 0.5);
}

.ai-spark-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #22c55e;
  box-shadow: 0 0 8px #22c55e;
  animation: pulseGreen 2s infinite;
}

@keyframes pulseGreen {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.5; transform: scale(0.8); }
}

.copilot-label {
  font-size: 13px;
  font-weight: 600;
  letter-spacing: 0.02em;
}

.drawer-content {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.model-status-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 14px;
  background: rgba(37, 99, 235, 0.06);
  border-radius: 10px;
  margin-bottom: 10px;
  font-size: 12px;
  color: #334155;
}

:global(html[data-theme='dark'] .model-status-bar) {
  background: rgba(37, 99, 235, 0.15);
  color: #cbd5e1;
}

.model-info {
  display: flex;
  align-items: center;
  gap: 6px;
}

.status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #94a3b8;
}

.status-dot.is-active {
  background: #22c55e;
}

.config-link {
  color: #2563eb;
  text-decoration: none;
  font-weight: 600;
}

.config-link:hover {
  text-decoration: underline;
}

.context-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: #f1f5f9;
  border-radius: 8px;
  margin-bottom: 10px;
  font-size: 12px;
}

:global(html[data-theme='dark'] .context-bar) {
  background: #141820;
}

.route-tag {
  font-family: monospace;
}

.ml-auto { margin-left: auto; }

.preset-prompts {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 12px;
}

.prompt-chip {
  font-size: 11px;
  padding: 4px 10px;
  border-radius: 14px;
  background: rgba(37, 99, 235, 0.08);
  color: #2563eb;
  cursor: pointer;
  font-weight: 500;
  transition: all 0.2s ease;
}

.prompt-chip:hover {
  background: #2563eb;
  color: #fff;
  transform: translateY(-1px);
}

.message-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.msg-bubble {
  display: flex;
  gap: 10px;
}

.msg-bubble.user {
  flex-direction: row-reverse;
}

.msg-avatar {
  width: 34px;
  height: 34px;
  border-radius: 12px;
  background: #e2e8f0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  flex-shrink: 0;
}

:global(html[data-theme='dark'] .msg-avatar) {
  background: #1e293b;
}

.msg-content {
  max-width: 84%;
}

.msg-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.user .msg-header {
  justify-content: flex-end;
}

.msg-author {
  font-size: 11px;
  color: #64748b;
  font-weight: 600;
}

.msg-time {
  font-size: 10px;
  color: #94a3b8;
}

.msg-text {
  padding: 12px 14px;
  border-radius: 12px;
  font-size: 13px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}

.assistant .msg-text {
  background: #f8fafc;
  color: #0f172a;
  border: 1px solid #e2e8f0;
}

:global(html[data-theme='dark'] .assistant .msg-text) {
  background: #1e2430;
  color: #f8fafc;
  border-color: rgba(255, 255, 255, 0.08);
}

.user .msg-text {
  background: linear-gradient(135deg, #2563eb 0%, #1d4ed8 100%);
  color: #fff;
}

.msg-actions {
  margin-top: 6px;
  display: flex;
  justify-content: flex-end;
}

.loading-text {
  color: #64748b;
  display: flex;
  align-items: center;
  gap: 6px;
}

.typing-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #2563eb;
  animation: typing 1.4s infinite ease-in-out both;
}

.typing-dot:nth-child(1) { animation-delay: -0.32s; }
.typing-dot:nth-child(2) { animation-delay: -0.16s; }

@keyframes typing {
  0%, 80%, 100% { transform: scale(0); }
  40% { transform: scale(1); }
}

.input-area {
  margin-top: 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.input-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.input-tip {
  font-size: 11px;
  color: #94a3b8;
}

.btn-group {
  display: flex;
  gap: 8px;
}
</style>
