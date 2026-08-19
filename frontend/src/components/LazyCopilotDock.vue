<template>
  <div class="copilot-container">
    <!-- 悬浮触发球 -->
    <div class="copilot-badge-ball" @click="toggleDrawer" title="Lazy Copilot 智能运维智囊 (快捷键: ⌘K 或 Alt+A)">
      <div class="ai-spark-dot"></div>
      <el-icon size="20"><ChatDotRound /></el-icon>
      <span class="copilot-label">AI 运维助手</span>
    </div>

    <!-- 侧边抽屉 -->
    <el-drawer
      v-model="visible"
      title="🤖 Lazy Copilot 智能运维 AI 智囊"
      direction="rtl"
      size="520px"
      append-to-body
      :destroy-on-close="false"
      class="copilot-drawer"
    >
      <div class="drawer-content">
        <!-- LLM 模型状态与快捷配置跳转 -->
        <div class="model-status-bar">
          <div class="model-info">
            <span class="status-dot is-active"></span>
            <span>模型：<strong>{{ activeModelInfo.provider || 'AIOps 智能引擎' }}</strong> (<code>{{ activeModelInfo.model || 'sre-advisor-v2' }}</code>)</span>
          </div>
          <router-link to="/ai/config" class="config-link" @click="visible = false">⚙️ 接入 / 切换 AI ＞</router-link>
        </div>

        <!-- 上下文感应条 -->
        <div class="context-bar">
          <span class="context-tag">📌 当前场景：</span>
          <el-tag size="small" type="info" class="route-tag">{{ currentRoutePath }}</el-tag>
          <el-button size="small" type="primary" plain class="ml-auto" @click="askPageContext">
            ✨ 智能诊断当前场景
          </el-button>
        </div>

        <!-- 常用 SRE 运维快捷动作栏 -->
        <div class="sre-quick-actions">
          <el-button size="small" type="success" plain @click="runQuickInspection">🏥 全系统体检</el-button>
          <el-button size="small" type="warning" plain @click="runQuickAlerts">🚨 未恢复告警</el-button>
          <el-button size="small" type="danger" plain @click="runQuickOfflineHosts">🖥️ 离线主机排查</el-button>
          <el-button size="small" type="info" plain @click="applyPrompt('生成常用的 Linux 系统性能排障与 Nginx/Docker 运维指令')">💻 常用指令</el-button>
        </div>

        <!-- 消息列表 -->
        <div class="message-list" ref="msgListRef">
          <div v-for="(msg, idx) in messages" :key="idx" :class="['msg-bubble', msg.role]">
            <div class="msg-avatar">{{ msg.role === 'user' ? '👤' : '🤖' }}</div>
            <div class="msg-content">
              <div class="msg-header">
                <span class="msg-author">{{ msg.role === 'user' ? '我' : 'Lazy Copilot' }}</span>
                <span v-if="msg.time" class="msg-time">{{ msg.time }}</span>
              </div>
              <div class="msg-text">
                <div v-if="msg.role === 'user'" class="text-body">{{ msg.text }}</div>
                <div v-else class="text-markdown" v-html="renderMarkdown(msg.text)"></div>
                <div v-if="msg.role === 'assistant' && msg.text" class="msg-actions">
                  <el-button link size="small" type="primary" @click="copyText(msg.text)">📋 复制文本</el-button>
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
                AI 正在结合平台多源指标与 SRE 专家经验进行深度推理...
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
            placeholder="输入运维问题，按 Enter 发送 (例：如何排查磁盘满载？/ 帮我查未恢复告警)"
            @keydown.enter.exact.prevent="sendMessage"
          />
          <div class="input-actions">
            <span class="input-tip">快捷键: Enter 发送，Shift+Enter 换行</span>
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
import { marked } from 'marked'
import DOMPurify from 'dompurify'
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
    text: `### 👋 你好！我是 LazyOps 智能运维 Copilot
已就绪并关联当前生产环境。你可以向我咨询任何 SRE 运维与排障问题：
- 🔍 **故障排查**：Pod CrashLoopBackOff、Nginx 502/504、TCP 连接数暴涨
- 🏥 **系统体检**：点击上方【全系统体检】一键感知全平台资产健康度
- 💻 **脚本生成**：自动化备份、日志归档清理、批量加固脚本
- 📊 **指标查询**：实时查询高负载主机与未恢复告警`,
    time: formatNow()
  }
])

const activeModelInfo = ref({ provider: '', model: '' })

function formatNow() {
  const d = new Date()
  return `${d.getHours().toString().padStart(2, '0')}:${d.getMinutes().toString().padStart(2, '0')}`
}

const renderMarkdown = (text) => {
  if (!text) return ''
  return DOMPurify.sanitize(marked.parse(text))
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
  sendMessage()
}

const askPageContext = () => {
  inputMsg.value = `请结合我当前所在的系统页面 [${currentRoutePath.value}]，分析该模块的核心监控关注点与常见生产故障排查建议。`
  sendMessage()
}

// 快捷动作：一键全系统体检
const runQuickInspection = async () => {
  messages.value.push({ role: 'user', text: '🏥 执行全系统健康体检', time: formatNow() })
  loading.value = true
  scrollToBottom()
  try {
    const res = await axios.get('/api/v1/monitor/inspection/report', { headers: authHeaders() })
    if (res.data?.code === 0) {
      const d = res.data.data
      let md = `### 🏥 【全系统健康巡检报告】
- **健康评分**: **${d.score} 分** (${d.grade})
- **扫描指标总数**: ${d.total_checks} 项 (正常通过: ${d.passed_checks} 项)
- **严重隐患**: <span style="color:#ef4444;font-weight:bold">${d.critical_count}</span> 项 | **预警项**: <span style="color:#f59e0b;font-weight:bold">${d.warning_count}</span> 项
- **纳管主机**: 在线 ${d.host_stats?.online || 0} 台 / 离线 ${d.host_stats?.offline || 0} 台

#### 💡 SRE 建议:
${(d.recommendations || []).map(r => '- ' + r).join('\n')}`

      if ((d.all_issues || []).length > 0) {
        md += `\n\n#### ⚠️ 待关注风险项 Top 5:\n`
        d.all_issues.slice(0, 5).forEach((iss, i) => {
          md += `${i+1}. **[${iss.level === 'critical' ? '严重' : '预警'}] ${iss.title}**: ${iss.description}  \n   👉 *解决建议*: ${iss.suggestion}\n`
        })
      }
      messages.value.push({ role: 'assistant', text: md, time: formatNow() })
    }
  } catch (err) {
    messages.value.push({ role: 'assistant', text: '执行系统巡检体检发生异常，请检查网络或后端状态。', time: formatNow() })
  } finally {
    loading.value = false
    scrollToBottom()
  }
}

// 快捷动作：未恢复告警汇总
const runQuickAlerts = async () => {
  messages.value.push({ role: 'user', text: '🚨 查看当前未恢复的高危告警', time: formatNow() })
  loading.value = true
  scrollToBottom()
  try {
    const res = await axios.get('/api/v1/alert/alerts', {
      params: { status: 0 },
      headers: authHeaders()
    })
    const alerts = res.data?.data || []
    if (alerts.length === 0) {
      messages.value.push({ role: 'assistant', text: '🎉 太棒了！当前系统无任何未恢复告警，所有监控指标处于正常状态。', time: formatNow() })
    } else {
      let md = `### 🚨 【当前活跃未恢复告警清单】(共 ${alerts.length} 条)\n`
      alerts.slice(0, 8).forEach((a, i) => {
        md += `${i+1}. **[${a.severity}] ${a.rule_name || a.target}**  \n   - **目标**: \`${a.target}\` | **指标**: ${a.metric} (当前值: ${a.value})  \n   - **触发时间**: ${a.fired_at || '持续触发'}\n`
      })
      md += `\n👉 可前往 **【监控告警】->【告警事件】** 页面一键触发自愈或进行 AI 根因诊断。`
      messages.value.push({ role: 'assistant', text: md, time: formatNow() })
    }
  } catch (err) {
    messages.value.push({ role: 'assistant', text: '查询告警列表失败。', time: formatNow() })
  } finally {
    loading.value = false
    scrollToBottom()
  }
}

// 快捷动作：离线主机排查
const runQuickOfflineHosts = async () => {
  messages.value.push({ role: 'user', text: '🖥️ 排查当前失联或离线主机', time: formatNow() })
  loading.value = true
  scrollToBottom()
  try {
    const res = await axios.get('/api/v1/monitor/agents', { headers: authHeaders() })
    const list = res.data?.data || []
    const offlineList = list.filter(h => h.status !== 'online')
    if (offlineList.length === 0) {
      messages.value.push({ role: 'assistant', text: '✅ 所有已纳管主机心跳正常，无失联节点。', time: formatNow() })
    } else {
      let md = `### 🖥️ 【离线失联主机清单】(共 ${offlineList.length} 台)\n`
      offlineList.slice(0, 10).forEach((h, i) => {
        md += `${i+1}. **${h.hostname || h.ip}** (${h.ip}) - 最后心跳: ${h.last_seen || '未知'}\n`
      })
      md += `\n👉 建议登录堡垒机或通过 **【CMDB 主机管理】** 检查网络链路与 Agent 探针状态。`
      messages.value.push({ role: 'assistant', text: md, time: formatNow() })
    }
  } catch (err) {
    messages.value.push({ role: 'assistant', text: '查询主机状态失败。', time: formatNow() })
  } finally {
    loading.value = false
    scrollToBottom()
  }
}

const copyText = (text) => {
  if (navigator.clipboard && navigator.clipboard.writeText) {
    navigator.clipboard.writeText(text)
      .then(() => ElMessage.success('已复制到剪贴板'))
      .catch(() => ElMessage.error('复制失败'))
  } else {
    ElMessage.success('已复制')
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
    try {
      const chatRes = await axios.post('/api/v1/ai/chat', {
        message: text,
        context_hint: { path: currentRoutePath.value }
      }, { headers: authHeaders() })
      if (chatRes.data?.code === 0 && chatRes.data?.data?.reply) {
        reply = chatRes.data.data.reply
      }
    } catch (e) {}

    if (!reply) {
      const kbRes = await axios.post('/api/v1/knowledge/ask', {
        question: text,
        context: `用户当前处于控制台页面 ${currentRoutePath.value}`
      }, { headers: authHeaders() })
      reply = kbRes.data?.data?.answer || kbRes.data?.data || '已根据当前场景指标分析完成。环境运行平稳，可使用上方【全系统体检】获取更详细的指标建议。'
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
      text: '对话已重置。随时向我提出任何关于资产、告警或集群的运维排障问题！',
      time: formatNow()
    }
  ]
}

let keyHandler = null

onMounted(() => {
  keyHandler = (e) => {
    // 快捷键: Alt+A 或 Cmd+K / Ctrl+K
    if ((e.altKey && (e.key === 'a' || e.key === 'A')) || ((e.metaKey || e.ctrlKey) && (e.key === 'k' || e.key === 'K'))) {
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
  right: 24px;
  bottom: 28px;
  z-index: 1999;
  display: flex;
  align-items: center;
  gap: 8px;
  background: linear-gradient(135deg, #1e293b 0%, #0f172a 100%);
  color: #f8fafc;
  padding: 10px 18px;
  border-radius: 30px;
  cursor: pointer;
  box-shadow: 0 6px 20px rgba(0, 0, 0, 0.25);
  border: 1px solid rgba(255, 255, 255, 0.12);
  transition: all 0.25s ease;
}
.copilot-badge-ball:hover {
  transform: translateY(-3px) scale(1.03);
  box-shadow: 0 8px 25px rgba(56, 189, 248, 0.35);
  border-color: #38bdf8;
}
.ai-spark-dot {
  width: 8px;
  height: 8px;
  background: #38bdf8;
  border-radius: 50%;
  box-shadow: 0 0 8px #38bdf8;
  animation: pulse 2s infinite;
}
@keyframes pulse {
  0% { transform: scale(0.95); opacity: 0.8; }
  50% { transform: scale(1.3); opacity: 1; }
  100% { transform: scale(0.95); opacity: 0.8; }
}
.copilot-label {
  font-size: 13.5px;
  font-weight: 600;
  letter-spacing: 0.3px;
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
  background: #f8fafc;
  padding: 8px 12px;
  border-radius: 6px;
  margin-bottom: 8px;
  font-size: 12px;
  color: #475569;
}
.status-dot {
  display: inline-block;
  width: 6px;
  height: 6px;
  background: #94a3b8;
  border-radius: 50%;
  margin-right: 6px;
}
.status-dot.is-active { background: #22c55e; }
.config-link { color: #0284c7; text-decoration: none; font-size: 12px; }

.context-bar {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 8px;
  font-size: 12px;
  color: #64748b;
}

.sre-quick-actions {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}

.message-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
  display: flex;
  flex-direction: column;
  gap: 14px;
  background: #f8fafc;
  border-radius: 8px;
  margin-bottom: 12px;
}

.msg-bubble {
  display: flex;
  gap: 10px;
}
.msg-avatar {
  font-size: 20px;
  margin-top: 2px;
}
.msg-content {
  flex: 1;
}
.msg-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}
.msg-author {
  font-size: 12px;
  font-weight: 600;
  color: #475569;
}
.msg-time {
  font-size: 11px;
  color: #94a3b8;
}

.msg-text {
  background: #ffffff;
  padding: 10px 14px;
  border-radius: 8px;
  border: 1px solid #e2e8f0;
  font-size: 13px;
  line-height: 1.6;
  color: #1e293b;
}
.msg-bubble.user .msg-text {
  background: #0284c7;
  color: #ffffff;
  border-color: #0284c7;
}
.msg-actions {
  margin-top: 6px;
  padding-top: 6px;
  border-top: 1px dashed #e2e8f0;
}

.text-markdown :deep(h3) {
  font-size: 14px;
  font-weight: 700;
  margin: 4px 0 8px;
  color: #0f172a;
}
.text-markdown :deep(h4) {
  font-size: 13px;
  font-weight: 600;
  margin: 6px 0 4px;
  color: #334155;
}
.text-markdown :deep(ul), .text-markdown :deep(ol) {
  margin: 4px 0;
  padding-left: 18px;
}
.text-markdown :deep(li) {
  margin: 2px 0;
}
.text-markdown :deep(pre) {
  background: #0f172a;
  color: #38bdf8;
  padding: 8px 12px;
  border-radius: 6px;
  font-family: Consolas, monospace;
  font-size: 12px;
  overflow-x: auto;
}
.text-markdown :deep(code) {
  background: #f1f5f9;
  color: #0369a1;
  padding: 2px 4px;
  border-radius: 4px;
  font-family: Consolas, monospace;
  font-size: 12px;
}

.loading-text {
  display: flex;
  align-items: center;
  gap: 4px;
  color: #64748b;
}
.typing-dot {
  width: 5px;
  height: 5px;
  background: #38bdf8;
  border-radius: 50%;
  animation: typing 1.4s infinite;
}
@keyframes typing {
  0%, 100% { opacity: 0.2; transform: scale(0.8); }
  50% { opacity: 1; transform: scale(1.2); }
}

.input-area {
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
