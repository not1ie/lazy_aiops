<template>
  <el-drawer
    v-model="visible"
    :size="isFullscreen ? '100%' : '82%'"
    :with-header="false"
    :destroy-on-close="false"
    :before-close="handleBeforeClose"
    class="web-terminal-drawer"
    direction="rtl"
  >
    <div class="terminal-container" :class="{ 'is-fullscreen': isFullscreen }">
      <!-- 抽屉顶部多功能导航与工具栏 -->
      <div class="terminal-header">
        <div class="header-left">
          <div class="host-badge">
            <el-icon class="host-icon"><Monitor /></el-icon>
            <span class="host-name">{{ hostInfo?.name || hostInfo?.hostname || hostInfo?.ip || 'SSH Terminal' }}</span>
            <span class="host-ip">({{ hostInfo?.username || 'root' }}@{{ hostInfo?.ip || 'localhost' }}:{{ hostInfo?.port || 22 }})</span>
          </div>

          <!-- 模式切换 Tabs: 终端 / SFTP 文件管理 -->
          <div class="mode-tabs">
            <button 
              class="mode-tab-btn" 
              :class="{ active: activeMode === 'terminal' }" 
              @click="switchMode('terminal')"
            >
              <el-icon><Platform /></el-icon> 交互终端
            </button>
            <button 
              class="mode-tab-btn" 
              :class="{ active: activeMode === 'sftp' }" 
              @click="switchMode('sftp')"
            >
              <el-icon><FolderOpened /></el-icon> 文件管理 (SFTP)
            </button>
          </div>

          <el-tag v-if="activeMode === 'terminal'" :type="statusTagType" size="small" effect="dark" class="status-tag">
            <span class="status-dot" :class="statusClass"></span>
            {{ statusText }}
          </el-tag>

          <transition name="fade">
            <span v-if="copyTipVisible" class="copy-tip">
              <el-icon><Check /></el-icon> 已复制选中内容
            </span>
          </transition>
        </div>

        <div class="header-right">
          <!-- 终端模式下的专用操作 -->
          <template v-if="activeMode === 'terminal'">
            <el-tooltip content="常用运维快捷指令" placement="bottom">
              <el-button 
                size="small" 
                :type="showSnippets ? 'primary' : 'default'" 
                plain 
                @click="showSnippets = !showSnippets"
              >
                <el-icon><Collection /></el-icon> 快捷指令
              </el-button>
            </el-tooltip>

            <el-tooltip content="清空终端 (Ctrl+L)" placement="bottom">
              <el-button size="small" plain @click="clearTerminal">
                <el-icon><Delete /></el-icon> 清屏
              </el-button>
            </el-tooltip>

            <el-tooltip content="重新连接当前主机" placement="bottom">
              <el-button size="small" plain :loading="status === 'connecting'" @click="reconnect">
                <el-icon><RefreshRight /></el-icon> 重连
              </el-button>
            </el-tooltip>
          </template>

          <!-- SFTP 模式下的专用操作 -->
          <template v-else>
            <el-button size="small" type="primary" icon="Upload" @click="triggerUploadDialog">
              上传文件
            </el-button>
            <input 
              ref="fileInputRef" 
              type="file" 
              multiple 
              style="display: none;" 
              @change="handleFileInputChange" 
            />

            <el-button size="small" plain icon="FolderAdd" @click="promptNewFolder">
              新建目录
            </el-button>
            <el-button size="small" plain icon="DocumentAdd" @click="promptNewFile">
              新建文件
            </el-button>
            <el-button size="small" plain icon="Refresh" :loading="sftpLoading" @click="loadSFTPList(currentSftpPath)">
              刷新
            </el-button>
          </template>

          <!-- 全屏切换 -->
          <el-tooltip :content="isFullscreen ? '退出全屏' : '全屏模式'" placement="bottom">
            <el-button size="small" plain @click="toggleFullscreen">
              <el-icon v-if="!isFullscreen"><FullScreen /></el-icon>
              <el-icon v-else><Crop /></el-icon>
            </el-button>
          </el-tooltip>

          <!-- 关闭抽屉 -->
          <el-button size="small" type="danger" plain @click="closeDrawer">
            <el-icon><Close /></el-icon> 关闭
          </el-button>
        </div>
      </div>

      <!-- ================= 模式 1: 终端命令行 ================= -->
      <div v-show="activeMode === 'terminal'" class="terminal-body-wrapper">
        <!-- 快捷命令指令条 -->
        <transition name="el-zoom-in-top">
          <div v-if="showSnippets" class="snippets-bar">
            <span class="snippets-label">快捷执行:</span>
            <el-button 
              v-for="item in snippetList" 
              :key="item.cmd" 
              size="small" 
              class="snippet-btn"
              @click="sendSnippet(item.cmd)"
            >
              {{ item.label }}
            </el-button>
          </div>
        </transition>

        <!-- 终端渲染主体 -->
        <div 
          ref="xtermRef" 
          class="xterm-wrapper" 
          @contextmenu.prevent="handleContextMenu"
        ></div>

        <!-- 终端底部状态与提示信息 -->
        <div class="terminal-footer">
          <div class="footer-tips">
            <span>💡 提示：支持鼠标<strong>划选自动复制</strong>，<strong>Ctrl+C</strong>（无选中中断，有选中复制），<strong>Ctrl+V</strong> 直接粘贴</span>
          </div>
          <div class="footer-meta">
            <span>编码: UTF-8</span>
            <span class="meta-sep">|</span>
            <span>终端类型: xterm-256color</span>
          </div>
        </div>
      </div>

      <!-- ================= 模式 2: SFTP 文件管理 ================= -->
      <div 
        v-show="activeMode === 'sftp'" 
        class="sftp-container"
        @dragover.prevent="isDragging = true"
        @dragleave.prevent="isDragging = false"
        @drop.prevent="handleDropUpload"
      >
        <!-- 拖拽上传覆盖遮罩 -->
        <div v-if="isDragging" class="sftp-drop-overlay">
          <el-icon class="drop-icon"><UploadFilled /></el-icon>
          <div class="drop-text">松开鼠标，直接上传到当前目录 ({{ currentSftpPath }})</div>
        </div>

        <!-- 路径导航与常用快捷路径 -->
        <div class="sftp-nav-bar">
          <div class="path-input-group">
            <el-button size="small" icon="Back" :disabled="currentSftpPath === '/'" @click="goParentDir">
              上级
            </el-button>
            <el-input 
              v-model="currentSftpPath" 
              size="small" 
              placeholder="输入绝对路径按回车直达"
              class="path-input"
              @keyup.enter="loadSFTPList(currentSftpPath)"
            >
              <template #prefix>
                <el-icon class="text-gray-400"><Folder /></el-icon>
              </template>
            </el-input>
            <el-button size="small" type="primary" plain @click="loadSFTPList(currentSftpPath)">
              直达
            </el-button>
          </div>

          <div class="quick-paths">
            <span class="quick-label">常用路径:</span>
            <el-tag 
              v-for="qp in quickPaths" 
              :key="qp.path" 
              size="small" 
              effect="plain" 
              class="quick-tag"
              @click="loadSFTPList(qp.path)"
            >
              {{ qp.label }}
            </el-tag>
          </div>
        </div>

        <!-- 面包屑路径条 -->
        <div class="sftp-breadcrumb-bar">
          <span class="crumb-root" @click="loadSFTPList('/')">/ 根目录</span>
          <template v-for="(crumb, idx) in pathCrumbs" :key="crumb.fullPath">
            <span class="crumb-sep">/</span>
            <span 
              class="crumb-item" 
              :class="{ 'is-last': idx === pathCrumbs.length - 1 }"
              @click="loadSFTPList(crumb.fullPath)"
            >
              {{ crumb.name }}
            </span>
          </template>
        </div>

        <!-- 文件列表表格 -->
        <div class="sftp-table-wrapper" v-loading="sftpLoading" element-loading-background="rgba(13, 17, 23, 0.8)">
          <el-table 
            :data="sftpFiles" 
            size="small" 
            style="width: 100%"
            class="sftp-table"
            empty-text="当前目录为空"
            @row-dblclick="handleRowDblClick"
          >
            <el-table-column label="名称" min-width="260">
              <template #default="{ row }">
                <div class="file-name-cell" @click="handleRowClick(row)">
                  <el-icon :class="getFileIconClass(row)"><component :is="getFileIcon(row)" /></el-icon>
                  <span class="file-title" :class="{ 'is-dir': row.is_dir }">{{ row.name }}</span>
                </div>
              </template>
            </el-table-column>

            <el-table-column prop="size_human" label="大小" width="110" align="right">
              <template #default="{ row }">
                <span v-if="!row.is_dir" class="file-size">{{ row.size_human }}</span>
                <span v-else class="text-gray-500">-</span>
              </template>
            </el-table-column>

            <el-table-column prop="mode" label="权限" width="130" align="center">
              <template #default="{ row }">
                <code class="perm-code">{{ row.mode }}</code>
              </template>
            </el-table-column>

            <el-table-column prop="mod_time" label="修改时间" width="165" align="center" />

            <el-table-column label="操作" width="220" fixed="right">
              <template #default="{ row }">
                <div class="sftp-op-cell">
                  <el-button 
                    v-if="!row.is_dir && isEditableFile(row.name)" 
                    size="small" 
                    type="primary" 
                    link 
                    icon="Edit"
                    @click="openFileEditor(row)"
                  >
                    编辑
                  </el-button>
                  <el-button 
                    v-if="!row.is_dir" 
                    size="small" 
                    type="success" 
                    link 
                    icon="Download"
                    @click="downloadSFTPFile(row)"
                  >
                    下载
                  </el-button>
                  <el-button 
                    size="small" 
                    type="warning" 
                    link 
                    icon="EditPen"
                    @click="promptRename(row)"
                  >
                    重命名
                  </el-button>
                  <el-popconfirm 
                    :title="`确定删除 ${row.is_dir ? '目录及其所有内容' : '文件'} [${row.name}] 吗？`"
                    confirm-button-text="删除"
                    cancel-button-text="取消"
                    confirm-button-type="danger"
                    @confirm="deleteSFTPItem(row)"
                  >
                    <template #reference>
                      <el-button size="small" type="danger" link icon="Delete">
                        删除
                      </el-button>
                    </template>
                  </el-popconfirm>
                </div>
              </template>
            </el-table-column>
          </el-table>
        </div>

        <!-- 底部文件统计与提示 -->
        <div class="sftp-footer">
          <div class="sftp-stats">
            <span>当前路径: <code>{{ currentSftpPath }}</code></span>
            <span class="meta-sep">|</span>
            <span>总计: {{ sftpFiles.length }} 项 ({{ dirCount }} 目录, {{ fileCount }} 文件)</span>
          </div>
          <div class="sftp-tips">
            <span>💡 提示：双击文件夹进入，双击文本/配置文件直接在线编辑；支持直接从电脑拖拽文件到窗口上传。</span>
          </div>
        </div>
      </div>

      <!-- 右键上下文菜单 -->
      <div 
        v-if="contextMenuVisible" 
        class="custom-context-menu" 
        :style="{ left: contextMenuPos.x + 'px', top: contextMenuPos.y + 'px' }"
        @click.stop
      >
        <div class="menu-item" @click="handlePasteFromClipboard">
          <el-icon><DocumentCopy /></el-icon> 粘贴剪贴板内容 (Ctrl+V)
        </div>
        <div class="menu-item" @click="handleCopySelection">
          <el-icon><CopyDocument /></el-icon> 复制已选文字 (Ctrl+C)
        </div>
        <div class="menu-divider"></div>
        <div class="menu-item" @click="clearTerminal">
          <el-icon><Delete /></el-icon> 清空当前屏幕 (Ctrl+L)
        </div>
        <div class="menu-item" @click="toggleFullscreen">
          <el-icon><FullScreen /></el-icon> {{ isFullscreen ? '退出全屏模式' : '进入全屏模式' }}
        </div>
        <div class="menu-divider"></div>
        <div class="menu-item" @click="reconnect">
          <el-icon><RefreshRight /></el-icon> 重新建立连接
        </div>
      </div>
    </div>

    <!-- ================= 在线代码 / 配置文件暗黑编辑器弹窗 ================= -->
    <el-dialog
      v-model="editorVisible"
      :title="`在线编辑 - ${currentEditingFile.name}`"
      width="80%"
      top="5vh"
      append-to-body
      :destroy-on-close="true"
      class="sftp-editor-dialog"
    >
      <div class="editor-header-bar">
        <div class="editor-file-info">
          <el-tag size="small" effect="dark" type="info">{{ currentEditingFile.ext || 'text' }}</el-tag>
          <span class="editor-file-path">{{ currentEditingFile.path }}</span>
          <span class="editor-file-size">({{ currentEditingFile.size_human }})</span>
        </div>
        <div class="editor-actions">
          <span class="editor-shortcut-tip">💡 支持 <strong>Ctrl+S / Cmd+S</strong> 快速保存</span>
          <el-button size="small" :loading="editorSaving" type="primary" icon="Check" @click="saveEditingFile">
            保存并应用 (Ctrl+S)
          </el-button>
        </div>
      </div>

      <div class="editor-body">
        <textarea
          ref="editorTextareaRef"
          v-model="editingContent"
          class="code-editor-textarea"
          spellcheck="false"
          @keydown="handleEditorKeyDown"
        ></textarea>
      </div>

      <template #footer>
        <el-button @click="editorVisible = false">关闭</el-button>
        <el-button type="primary" :loading="editorSaving" @click="saveEditingFile">
          保存更改
        </el-button>
      </template>
    </el-dialog>

    <!-- 凭据补充弹窗（若主机未绑定凭据） -->
    <el-dialog
      v-model="credentialDialogVisible"
      title="补充登录凭据"
      width="450px"
      append-to-body
      :close-on-click-modal="false"
    >
      <el-form :model="credentialForm" label-width="80px" size="default">
        <el-form-item label="登录用户">
          <el-input v-model="credentialForm.username" placeholder="默认 root" clearable />
        </el-form-item>
        <el-form-item label="认证方式">
          <el-radio-group v-model="authType">
            <el-radio-button label="password">密码认证</el-radio-button>
            <el-radio-button label="key">私钥认证</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="authType === 'password'" label="登录密码">
          <el-input 
            v-model="credentialForm.password" 
            type="password" 
            placeholder="请输入服务器 SSH 密码" 
            show-password 
            @keyup.enter="submitCredentialAndConnect"
          />
        </el-form-item>
        <el-form-item v-else label="SSH私钥">
          <el-input 
            v-model="credentialForm.key_auth" 
            type="textarea" 
            :rows="4" 
            placeholder="-----BEGIN RSA PRIVATE KEY-----..." 
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="credentialDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="connecting" @click="submitCredentialAndConnect">
          连接服务器
        </el-button>
      </template>
    </el-dialog>
  </el-drawer>
</template>

<script setup>
import { ref, computed, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { Terminal } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import 'xterm/css/xterm.css'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Monitor,
  Platform,
  FolderOpened,
  Folder,
  FolderAdd,
  DocumentAdd,
  Document,
  DocumentCopy,
  CopyDocument,
  Check,
  Collection,
  Delete,
  RefreshRight,
  Refresh,
  FullScreen,
  Crop,
  Close,
  Upload,
  UploadFilled,
  Download,
  Edit,
  EditPen
} from '@element-plus/icons-vue'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  },
  hostInfo: {
    type: Object,
    default: () => ({})
  },
  sessionId: {
    type: String,
    default: ''
  }
})

const emit = defineEmits(['update:modelValue', 'close'])

const visible = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})

// 模式切换: terminal (命令行) | sftp (文件管理)
const activeMode = ref('terminal')

const xtermRef = ref(null)
const isFullscreen = ref(false)
const showSnippets = ref(false)
const status = ref('idle') // idle, connecting, online, disconnected, error
const copyTipVisible = ref(false)
let copyTipTimer = null

// 凭据补充表单
const credentialDialogVisible = ref(false)
const connecting = ref(false)
const authType = ref('password')
const credentialForm = ref({
  username: 'root',
  password: '',
  key_auth: ''
})

// 右键菜单
const contextMenuVisible = ref(false)
const contextMenuPos = ref({ x: 0, y: 0 })

// 终端实例与 WebSocket
let term = null
let fitAddon = null
let ws = null
let wsGeneration = 0
let resizeObserver = null
let pingTimer = null
const activeSessionId = ref('')

// ================= SFTP 文件管理器相关状态 =================
const fileInputRef = ref(null)
const sftpLoading = ref(false)
const isDragging = ref(false)
const currentSftpPath = ref('/root')
const sftpFiles = ref([])

// 常用快捷路径
const quickPaths = [
  { label: 'root 主目录', path: '/root' },
  { label: '/etc 配置', path: '/etc' },
  { label: '/var/log 日志', path: '/var/log' },
  { label: '/tmp 临时', path: '/tmp' },
  { label: '/data 数据', path: '/data' },
  { label: '/opt 应用', path: '/opt' },
  { label: '/ 根目录', path: '/' }
]

// 在线编辑器
const editorVisible = ref(false)
const editorSaving = ref(false)
const editorTextareaRef = ref(null)
const currentEditingFile = ref({})
const editingContent = ref('')

// 快捷运维指令列表
const snippetList = [
  { label: '系统资源 (top)', cmd: 'top\n' },
  { label: '磁盘用量 (df -h)', cmd: 'df -h\n' },
  { label: '内存状态 (free -m)', cmd: 'free -m\n' },
  { label: '网络端口 (netstat)', cmd: 'netstat -tulnp || ss -tulnp\n' },
  { label: '容器状态 (docker ps)', cmd: 'docker ps\n' },
  { label: 'IP 地址 (ip a)', cmd: 'ip a\n' },
  { label: '系统日志 (dmesg)', cmd: 'dmesg -T | tail -n 30\n' },
  { label: '运行时间 (uptime)', cmd: 'uptime\n' }
]

// 状态文字与样式映射
const statusText = computed(() => {
  switch (status.value) {
    case 'connecting': return '连接中...'
    case 'online': return '已连接'
    case 'reconnected': return '已重连'
    case 'disconnected': return '已断开'
    case 'error': return '连接异常'
    default: return '未连接'
  }
})

const statusTagType = computed(() => {
  switch (status.value) {
    case 'online':
    case 'reconnected': return 'success'
    case 'connecting': return 'warning'
    case 'disconnected': return 'info'
    case 'error': return 'danger'
    default: return 'info'
  }
})

const statusClass = computed(() => {
  switch (status.value) {
    case 'online':
    case 'reconnected': return 'is-online'
    case 'connecting': return 'is-connecting'
    case 'disconnected': return 'is-disconnected'
    case 'error': return 'is-error'
    default: return ''
  }
})

// 面包屑路径拆分
const pathCrumbs = computed(() => {
  const p = currentSftpPath.value.trim()
  if (!p || p === '/') return []
  const segments = p.split('/').filter(Boolean)
  let accum = ''
  return segments.map(seg => {
    accum += '/' + seg
    return {
      name: seg,
      fullPath: accum
    }
  })
})

const dirCount = computed(() => sftpFiles.value.filter(f => f.is_dir).length)
const fileCount = computed(() => sftpFiles.value.filter(f => !f.is_dir).length)

const authHeaders = () => {
  const token = localStorage.getItem('token') || ''
  return { Authorization: `Bearer ${token}` }
}

const triggerCopyTip = () => {
  copyTipVisible.value = true
  if (copyTipTimer) clearTimeout(copyTipTimer)
  copyTipTimer = setTimeout(() => {
    copyTipVisible.value = false
  }, 2000)
}

// 模式切换
const switchMode = (mode) => {
  activeMode.value = mode
  if (mode === 'terminal') {
    nextTick(() => {
      if (fitAddon) {
        fitAddon.fit()
        sendResize()
      }
      term?.focus()
    })
  } else if (mode === 'sftp') {
    if (sftpFiles.value.length === 0) {
      loadSFTPList(currentSftpPath.value)
    }
  }
}

// 初始化 xterm 终端实例
const initTerminal = async () => {
  await nextTick()
  if (!xtermRef.value) return

  if (term) {
    term.dispose()
    term = null
  }

  term = new Terminal({
    cursorBlink: true,
    cursorStyle: 'block',
    fontSize: 13.5,
    fontFamily: 'Consolas, "Fira Code", Monaco, Menlo, "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", "WenQuanYi Micro Hei", monospace',
    lineHeight: 1.22,
    letterSpacing: 0,
    allowProposedApi: true,
    scrollback: 10000,
    theme: {
      background: '#0d1117',
      foreground: '#e6edf3',
      cursor: '#58a6ff',
      cursorAccent: '#0d1117',
      selectionBackground: 'rgba(88, 166, 255, 0.35)',
      black: '#484f58',
      red: '#ff7b72',
      green: '#3fb950',
      yellow: '#d29922',
      blue: '#58a6ff',
      magenta: '#bc8cff',
      cyan: '#39c5cf',
      white: '#b1bac4',
      brightBlack: '#6e7681',
      brightRed: '#ffa198',
      brightGreen: '#56d364',
      brightYellow: '#e3b341',
      brightBlue: '#79c0ff',
      brightMagenta: '#d2a8ff',
      brightCyan: '#56d4dd',
      brightWhite: '#f0f6fc'
    }
  })

  fitAddon = new FitAddon()
  term.loadAddon(fitAddon)
  term.open(xtermRef.value)
  fitAddon.fit()

  // 1. 选中即复制 (Copy on Select)
  term.onSelectionChange(() => {
    const selection = term.getSelection()?.trim()
    if (selection) {
      navigator.clipboard?.writeText(selection).then(() => {
        triggerCopyTip()
      }).catch(() => {})
    }
  })

  // 2. 快捷键拦截适配 (Ctrl+C / Ctrl+V / Cmd+C / Cmd+V)
  term.attachCustomKeyEventHandler((e) => {
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'c') {
      if (term.hasSelection()) {
        const text = term.getSelection()
        navigator.clipboard?.writeText(text).catch(() => {})
        triggerCopyTip()
        return false
      }
      return true
    }

    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'v') {
      if (e.type === 'keydown') {
        navigator.clipboard?.readText().then(text => {
          if (text && ws && ws.readyState === WebSocket.OPEN) {
            ws.send(text)
          }
        }).catch(() => {})
      }
      return false
    }

    return true
  })

  // 3. 用户输入发送
  term.onData((data) => {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(data)
    }
  })

  // 4. 自适应尺寸监听
  if (!resizeObserver && xtermRef.value) {
    resizeObserver = new ResizeObserver(() => {
      if (fitAddon && term && activeMode.value === 'terminal') {
        fitAddon.fit()
        sendResize()
      }
    })
    resizeObserver.observe(xtermRef.value)
  }
}

const sendResize = () => {
  if (term && ws && ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({
      type: 'resize',
      cols: term.cols,
      rows: term.rows
    }))
  }
}

const buildWsUrl = (sessionId) => {
  const token = localStorage.getItem('token') || ''
  const proto = window.location.protocol === 'https:' ? 'wss' : 'ws'
  return `${proto}://${window.location.host}/api/v1/terminal/ws/${sessionId}?token=${encodeURIComponent(token)}`
}

const connectWebSocket = (sessionId) => {
  closeWebSocket()
  status.value = 'connecting'
  const url = buildWsUrl(sessionId)
  const currentGen = ++wsGeneration

  ws = new WebSocket(url)
  ws.binaryType = 'arraybuffer'

  ws.onopen = () => {
    if (currentGen !== wsGeneration) return
    status.value = 'online'
    term?.writeln('\x1b[32m[系统] SSH 终端连接成功，已建立交互式会话。\x1b[0m\r\n')
    sendResize()

    if (pingTimer) clearInterval(pingTimer)
    pingTimer = setInterval(() => {
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'ping', ts: Date.now() }))
      }
    }, 15000)
  }

  ws.onmessage = (event) => {
    if (currentGen !== wsGeneration) return
    if (typeof event.data === 'string') {
      try {
        const msg = JSON.parse(event.data)
        if (msg?.type === 'pong') return
      } catch {}
      term?.write(event.data)
      return
    }
    term?.write(new Uint8Array(event.data))
  }

  ws.onerror = () => {
    if (currentGen !== wsGeneration) return
    status.value = 'error'
    term?.writeln('\r\n\x1b[31m[系统] WebSocket 连接发生异常，请检查网络或认证状态。\x1b[0m')
  }

  ws.onclose = (event) => {
    if (currentGen !== wsGeneration) return
    status.value = 'disconnected'
    if (pingTimer) clearInterval(pingTimer)
    term?.writeln(`\r\n\x1b[33m[系统] 会话连接已断开 (代码: ${event.code})。\x1b[0m`)
  }
}

const closeWebSocket = () => {
  if (pingTimer) clearInterval(pingTimer)
  if (ws) {
    try {
      ws.close()
    } catch {}
    ws = null
  }
}

// 快速创建会话并连接
const startSessionConnect = async (customCreds = null) => {
  if (!props.hostInfo?.id) {
    if (props.sessionId) {
      activeSessionId.value = props.sessionId
      await initTerminal()
      connectWebSocket(props.sessionId)
      return
    }
    return
  }

  status.value = 'connecting'
  await initTerminal()
  term?.writeln(`\x1b[36m[系统] 正在向服务器 ${props.hostInfo.name || props.hostInfo.ip} 发起 SSH 连接...\x1b[0m`)

  try {
    const payload = customCreds || {}
    const res = await axios.post(`/api/v1/terminal/quick-connect-host/${props.hostInfo.id}`, payload, {
      headers: authHeaders()
    })

    if (res.data?.code === 0 && res.data?.data?.id) {
      activeSessionId.value = res.data.data.id
      connectWebSocket(res.data.data.id)
    } else if (res.data?.code === 4001) {
      credentialForm.value.username = res.data.data?.username || 'root'
      credentialDialogVisible.value = true
      status.value = 'idle'
      term?.writeln('\x1b[33m[系统] 该主机未绑定已保存凭据，请在弹出的对话框中输入密码或私钥。\x1b[0m')
    } else {
      status.value = 'error'
      term?.writeln(`\r\n\x1b[31m[系统] 连接失败: ${res.data?.message || '未知错误'}\x1b[0m`)
    }
  } catch (err) {
    status.value = 'error'
    const msg = err.response?.data?.message || err.message || '连接失败'
    term?.writeln(`\r\n\x1b[31m[系统] 初始化终端会话异常: ${msg}\x1b[0m`)
  }
}

const submitCredentialAndConnect = async () => {
  if (authType.value === 'password' && !credentialForm.value.password.trim()) {
    ElMessage.warning('请输入登录密码')
    return
  }
  if (authType.value === 'key' && !credentialForm.value.key_auth.trim()) {
    ElMessage.warning('请输入 SSH 私钥')
    return
  }

  connecting.value = true
  try {
    await startSessionConnect({
      username: credentialForm.value.username || 'root',
      password: authType.value === 'password' ? credentialForm.value.password : '',
      key_auth: authType.value === 'key' ? credentialForm.value.key_auth : ''
    })
    credentialDialogVisible.value = false
  } finally {
    connecting.value = false
  }
}

const clearTerminal = () => {
  term?.clear()
}

const reconnect = () => {
  if (activeSessionId.value) {
    connectWebSocket(activeSessionId.value)
  } else {
    startSessionConnect()
  }
}

const sendSnippet = (cmd) => {
  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(cmd)
    term?.focus()
  } else {
    ElMessage.warning('终端尚未连接')
  }
}

const toggleFullscreen = () => {
  isFullscreen.value = !isFullscreen.value
  nextTick(() => {
    if (fitAddon && activeMode.value === 'terminal') {
      fitAddon.fit()
      sendResize()
    }
  })
}

const handleContextMenu = (e) => {
  contextMenuPos.value = { x: e.clientX, y: e.clientY }
  contextMenuVisible.value = true
}

const handlePasteFromClipboard = () => {
  contextMenuVisible.value = false
  navigator.clipboard?.readText().then(text => {
    if (text && ws && ws.readyState === WebSocket.OPEN) {
      ws.send(text)
    }
  }).catch(() => {
    ElMessage.warning('未能读取剪贴板，请使用快捷键 Ctrl+V / Cmd+V 粘贴')
  })
}

const handleCopySelection = () => {
  contextMenuVisible.value = false
  const text = term?.getSelection()
  if (text) {
    navigator.clipboard?.writeText(text).then(() => {
      triggerCopyTip()
    }).catch(() => {})
  } else {
    ElMessage.info('未选中文本')
  }
}

const closeContextMenu = () => {
  contextMenuVisible.value = false
}

// ================= SFTP 文件管理核心方法 =================
const getActiveSession = async () => {
  if (activeSessionId.value) return activeSessionId.value
  if (props.hostInfo?.id) {
    const res = await axios.post(`/api/v1/terminal/quick-connect-host/${props.hostInfo.id}`, {}, {
      headers: authHeaders()
    })
    if (res.data?.code === 0 && res.data?.data?.id) {
      activeSessionId.value = res.data.data.id
      return res.data.data.id
    }
  }
  return ''
}

const loadSFTPList = async (targetPath = '/root') => {
  const sessId = await getActiveSession()
  if (!sessId) {
    ElMessage.warning('无法建立 SFTP 会话，请确认主机凭据')
    return
  }

  sftpLoading.value = true
  try {
    const res = await axios.get(`/api/v1/terminal/sftp/${sessId}/list`, {
      params: { path: targetPath },
      headers: authHeaders()
    })

    if (res.data?.code === 0) {
      currentSftpPath.value = res.data.data.path
      sftpFiles.value = res.data.data.files || []
    } else {
      ElMessage.error(res.data?.message || '读取目录失败')
    }
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '无法访问该目录')
  } finally {
    sftpLoading.value = false
  }
}

const goParentDir = () => {
  const p = currentSftpPath.value
  if (p === '/') return
  const parent = p.substring(0, p.lastIndexOf('/')) || '/'
  loadSFTPList(parent)
}

const handleRowClick = (row) => {
  if (row.is_dir) {
    loadSFTPList(row.path)
  }
}

const handleRowDblClick = (row) => {
  if (row.is_dir) {
    loadSFTPList(row.path)
  } else if (isEditableFile(row.name)) {
    openFileEditor(row)
  }
}

// 图标适配
const getFileIcon = (row) => {
  if (row.is_dir) return Folder
  return Document
}

const getFileIconClass = (row) => {
  if (row.is_dir) return 'icon-dir'
  const ext = (row.ext || '').toLowerCase()
  if (['conf', 'yaml', 'yml', 'json', 'ini', 'toml', 'env'].includes(ext)) return 'icon-config'
  if (['sh', 'bash', 'py', 'go', 'js', 'sql'].includes(ext)) return 'icon-code'
  if (['log', 'txt', 'md'].includes(ext)) return 'icon-text'
  if (['tar', 'gz', 'zip', 'rar', 'tgz'].includes(ext)) return 'icon-archive'
  return 'icon-file'
}

const isEditableFile = (fileName) => {
  const name = (fileName || '').toLowerCase()
  const exts = ['.conf', '.yaml', '.yml', '.json', '.sh', '.bash', '.py', '.go', '.js', '.sql', '.log', '.txt', '.md', '.env', '.ini', '.toml', '.xml', '.properties', 'dockerfile', 'makefile', 'hosts']
  return exts.some(ext => name.endsWith(ext) || name === ext)
}

// 上传处理
const triggerUploadDialog = () => {
  fileInputRef.value?.click()
}

const handleFileInputChange = (e) => {
  const files = e.target.files
  if (files && files.length > 0) {
    uploadFiles(files)
  }
  e.target.value = ''
}

const handleDropUpload = (e) => {
  isDragging.value = false
  const files = e.dataTransfer.files
  if (files && files.length > 0) {
    uploadFiles(files)
  }
}

const uploadFiles = async (fileList) => {
  const sessId = await getActiveSession()
  if (!sessId) return

  const formData = new FormData()
  formData.append('target_dir', currentSftpPath.value)
  for (let i = 0; i < fileList.length; i++) {
    formData.append('files', fileList[i])
  }

  sftpLoading.value = true
  try {
    const res = await axios.post(`/api/v1/terminal/sftp/${sessId}/upload`, formData, {
      headers: {
        ...authHeaders(),
        'Content-Type': 'multipart/form-data'
      }
    })
    if (res.data?.code === 0) {
      ElMessage.success(res.data.message || '上传成功')
      await loadSFTPList(currentSftpPath.value)
    } else {
      ElMessage.error(res.data?.message || '上传失败')
    }
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '文件上传发生异常')
  } finally {
    sftpLoading.value = false
  }
}

// 下载处理
const downloadSFTPFile = async (row) => {
  const sessId = await getActiveSession()
  if (!sessId) return

  const downloadUrl = `/api/v1/terminal/sftp/${sessId}/download?path=${encodeURIComponent(row.path)}`
  const link = document.createElement('a')
  link.href = downloadUrl
  link.setAttribute('download', row.name)
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
}

// 新建目录
const promptNewFolder = async () => {
  try {
    const { value } = await ElMessageBox.prompt('请输入新建目录名称', '新建目录', {
      confirmButtonText: '创建',
      cancelButtonText: '取消',
      inputPattern: /^[^/]+$/,
      inputErrorMessage: '目录名不能包含 / 字符'
    })

    if (!value?.trim()) return
    const sessId = await getActiveSession()
    const targetDir = currentSftpPath.value === '/' ? `/${value.trim()}` : `${currentSftpPath.value}/${value.trim()}`

    const res = await axios.post(`/api/v1/terminal/sftp/${sessId}/mkdir`, { path: targetDir }, {
      headers: authHeaders()
    })
    if (res.data?.code === 0) {
      ElMessage.success('目录创建成功')
      await loadSFTPList(currentSftpPath.value)
    } else {
      ElMessage.error(res.data?.message || '创建失败')
    }
  } catch {}
}

// 新建文件
const promptNewFile = async () => {
  try {
    const { value } = await ElMessageBox.prompt('请输入新建文件名 (如 test.conf, app.sh)', '新建文件', {
      confirmButtonText: '创建并编辑',
      cancelButtonText: '取消',
      inputPattern: /^[^/]+$/,
      inputErrorMessage: '文件名不能包含 / 字符'
    })

    if (!value?.trim()) return
    const sessId = await getActiveSession()
    const targetFile = currentSftpPath.value === '/' ? `/${value.trim()}` : `${currentSftpPath.value}/${value.trim()}`

    const res = await axios.post(`/api/v1/terminal/sftp/${sessId}/write`, {
      path: targetFile,
      content: ''
    }, { headers: authHeaders() })

    if (res.data?.code === 0) {
      ElMessage.success('文件创建成功')
      await loadSFTPList(currentSftpPath.value)
      // 直接打开在线编辑
      openFileEditor({
        name: value.trim(),
        path: targetFile,
        size_human: '0 B',
        ext: value.trim().split('.').pop()
      })
    } else {
      ElMessage.error(res.data?.message || '创建失败')
    }
  } catch {}
}

// 重命名
const promptRename = async (row) => {
  try {
    const { value } = await ElMessageBox.prompt(`重命名 [${row.name}]`, '重命名', {
      confirmButtonText: '确认',
      cancelButtonText: '取消',
      inputValue: row.name,
      inputPattern: /^[^/]+$/,
      inputErrorMessage: '文件名不能包含 / 字符'
    })

    if (!value || value.trim() === row.name) return
    const sessId = await getActiveSession()
    const parentDir = currentSftpPath.value === '/' ? '' : currentSftpPath.value
    const newPath = `${parentDir}/${value.trim()}`

    const res = await axios.post(`/api/v1/terminal/sftp/${sessId}/rename`, {
      old_path: row.path,
      new_path: newPath
    }, { headers: authHeaders() })

    if (res.data?.code === 0) {
      ElMessage.success('重命名成功')
      await loadSFTPList(currentSftpPath.value)
    } else {
      ElMessage.error(res.data?.message || '重命名失败')
    }
  } catch {}
}

// 删除
const deleteSFTPItem = async (row) => {
  const sessId = await getActiveSession()
  if (!sessId) return

  try {
    const res = await axios.delete(`/api/v1/terminal/sftp/${sessId}/delete`, {
      params: { path: row.path },
      headers: authHeaders()
    })
    if (res.data?.code === 0) {
      ElMessage.success('删除成功')
      await loadSFTPList(currentSftpPath.value)
    } else {
      ElMessage.error(res.data?.message || '删除失败')
    }
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '删除发生异常')
  }
}

// 打开在线编辑器
const openFileEditor = async (row) => {
  const sessId = await getActiveSession()
  if (!sessId) return

  currentEditingFile.value = row
  editingContent.value = ''
  editorVisible.value = true

  try {
    const res = await axios.get(`/api/v1/terminal/sftp/${sessId}/read`, {
      params: { path: row.path },
      headers: authHeaders()
    })
    if (res.data?.code === 0) {
      editingContent.value = res.data.data.content || ''
      nextTick(() => {
        editorTextareaRef.value?.focus()
      })
    } else {
      ElMessage.error(res.data?.message || '读取文件失败')
      editorVisible.value = false
    }
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '读取文件发生异常')
    editorVisible.value = false
  }
}

// 快捷键 Ctrl+S / Tab 键拦截
const handleEditorKeyDown = (e) => {
  // Ctrl+S / Cmd+S 保存
  if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 's') {
    e.preventDefault()
    saveEditingFile()
    return
  }

  // Tab 键缩进 2 空格
  if (e.key === 'Tab') {
    e.preventDefault()
    const textarea = editorTextareaRef.value
    if (!textarea) return
    const start = textarea.selectionStart
    const end = textarea.selectionEnd
    editingContent.value = editingContent.value.substring(0, start) + '  ' + editingContent.value.substring(end)
    nextTick(() => {
      textarea.selectionStart = textarea.selectionEnd = start + 2
    })
  }
}

// 保存编辑文件
const saveEditingFile = async () => {
  const sessId = await getActiveSession()
  if (!sessId || !currentEditingFile.value?.path) return

  editorSaving.value = true
  try {
    const res = await axios.post(`/api/v1/terminal/sftp/${sessId}/write`, {
      path: currentEditingFile.value.path,
      content: editingContent.value
    }, { headers: authHeaders() })

    if (res.data?.code === 0) {
      ElMessage.success('🎉 文件已成功保存至远程服务器！')
    } else {
      ElMessage.error(res.data?.message || '保存失败')
    }
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '保存文件发生异常')
  } finally {
    editorSaving.value = false
  }
}

const handleBeforeClose = (done) => {
  closeWebSocket()
  done()
}

const closeDrawer = () => {
  closeWebSocket()
  visible.value = false
  emit('close')
}

watch(() => props.modelValue, (val) => {
  if (val) {
    isFullscreen.value = false
    status.value = 'idle'
    activeMode.value = 'terminal'
    sftpFiles.value = []
    nextTick(() => {
      startSessionConnect()
    })
  } else {
    closeWebSocket()
  }
})

onMounted(() => {
  window.addEventListener('click', closeContextMenu)
})

onBeforeUnmount(() => {
  window.removeEventListener('click', closeContextMenu)
  if (resizeObserver) resizeObserver.disconnect()
  closeWebSocket()
  if (term) term.dispose()
})
</script>

<style scoped>
.web-terminal-drawer :deep(.el-drawer__body) {
  padding: 0;
  background-color: #0d1117;
}

.terminal-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  background-color: #0d1117;
  color: #e6edf3;
  overflow: hidden;
  user-select: none;
}

.terminal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 16px;
  background: #161b22;
  border-bottom: 1px solid #30363d;
  flex-shrink: 0;
  gap: 12px;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 14px;
  flex-wrap: wrap;
}

.host-badge {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 600;
  color: #f0f6fc;
}

.host-icon {
  font-size: 17px;
  color: #58a6ff;
}

.host-ip {
  font-size: 12px;
  color: #8b949e;
  font-weight: normal;
}

/* 模式切换 Tab 样式 */
.mode-tabs {
  display: flex;
  align-items: center;
  background: #0d1117;
  border: 1px solid #30363d;
  border-radius: 6px;
  padding: 2px;
  gap: 2px;
}

.mode-tab-btn {
  display: flex;
  align-items: center;
  gap: 5px;
  border: none;
  background: transparent;
  color: #8b949e;
  padding: 4px 10px;
  font-size: 12px;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.mode-tab-btn:hover {
  color: #c9d1d9;
}

.mode-tab-btn.active {
  background: #21262d;
  color: #58a6ff;
  font-weight: 600;
  box-shadow: 0 1px 3px rgba(0,0,0,0.3);
}

.status-tag {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
}

.status-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background-color: #8b949e;
}

.status-dot.is-online {
  background-color: #3fb950;
  box-shadow: 0 0 6px rgba(63, 185, 80, 0.8);
}

.status-dot.is-connecting {
  background-color: #d29922;
  animation: blink 1.2s infinite;
}

.status-dot.is-error {
  background-color: #f85149;
}

@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.3; }
}

.copy-tip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: #3fb950;
  background: rgba(63, 185, 80, 0.15);
  padding: 2px 8px;
  border-radius: 4px;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.terminal-body-wrapper {
  display: flex;
  flex-direction: column;
  flex: 1;
  overflow: hidden;
}

.snippets-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  background: #1c2128;
  border-bottom: 1px solid #30363d;
  flex-wrap: wrap;
}

.snippets-label {
  font-size: 12px;
  color: #8b949e;
  margin-right: 4px;
}

.snippet-btn {
  background: #21262d;
  border-color: #30363d;
  color: #c9d1d9;
}

.snippet-btn:hover {
  background: #30363d;
  border-color: #58a6ff;
  color: #58a6ff;
}

.xterm-wrapper {
  flex: 1;
  padding: 8px 12px;
  background-color: #0d1117;
  overflow: hidden;
  user-select: text;
}

.xterm-wrapper :deep(.xterm) {
  height: 100%;
}

.terminal-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 16px;
  background: #161b22;
  border-top: 1px solid #30363d;
  font-size: 11.5px;
  color: #8b949e;
  flex-shrink: 0;
}

.footer-meta {
  display: flex;
  align-items: center;
  gap: 8px;
}

.meta-sep {
  color: #30363d;
  margin: 0 4px;
}

/* ================= SFTP 文件管理样式 ================= */
.sftp-container {
  position: relative;
  display: flex;
  flex-direction: column;
  flex: 1;
  background: #0d1117;
  overflow: hidden;
}

.sftp-drop-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(13, 17, 23, 0.92);
  border: 2px dashed #58a6ff;
  border-radius: 8px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  z-index: 100;
  color: #58a6ff;
  pointer-events: none;
}

.drop-icon {
  font-size: 48px;
  margin-bottom: 12px;
  animation: bounce 1.2s infinite ease-in-out;
}

.drop-text {
  font-size: 16px;
  font-weight: 600;
}

@keyframes bounce {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-8px); }
}

.sftp-nav-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 16px;
  background: #161b22;
  border-bottom: 1px solid #30363d;
  gap: 16px;
  flex-wrap: wrap;
}

.path-input-group {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  max-width: 540px;
}

.path-input :deep(.el-input__wrapper) {
  background-color: #0d1117;
  border-color: #30363d;
  box-shadow: none;
  color: #f0f6fc;
}

.quick-paths {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.quick-label {
  font-size: 12px;
  color: #8b949e;
}

.quick-tag {
  background: #21262d;
  border-color: #30363d;
  color: #c9d1d9;
  cursor: pointer;
  transition: all 0.15s;
}

.quick-tag:hover {
  background: #30363d;
  color: #58a6ff;
  border-color: #58a6ff;
}

.sftp-breadcrumb-bar {
  display: flex;
  align-items: center;
  padding: 6px 16px;
  background: #11161d;
  border-bottom: 1px solid #21262d;
  font-size: 12px;
  overflow-x: auto;
  white-space: nowrap;
}

.crumb-root, .crumb-item {
  color: #58a6ff;
  cursor: pointer;
}

.crumb-root:hover, .crumb-item:hover {
  text-decoration: underline;
}

.crumb-item.is-last {
  color: #f0f6fc;
  font-weight: 600;
  cursor: default;
  text-decoration: none;
}

.crumb-sep {
  color: #484f58;
  margin: 0 6px;
}

.sftp-table-wrapper {
  flex: 1;
  overflow: auto;
  background: #0d1117;
}

.sftp-table {
  background-color: transparent !important;
}

.sftp-table :deep(tr),
.sftp-table :deep(th.el-table__cell) {
  background-color: #0d1117 !important;
  color: #c9d1d9 !important;
  border-bottom: 1px solid #21262d !important;
}

.sftp-table :deep(.el-table__row:hover > td.el-table__cell) {
  background-color: #161b22 !important;
}

.file-name-cell {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  user-select: text;
}

.file-title {
  color: #e6edf3;
}

.file-title.is-dir {
  color: #58a6ff;
  font-weight: 500;
}

.file-title:hover {
  text-decoration: underline;
}

.icon-dir { color: #58a6ff; font-size: 16px; }
.icon-config { color: #f59e0b; font-size: 16px; }
.icon-code { color: #34d399; font-size: 16px; }
.icon-text { color: #60a5fa; font-size: 16px; }
.icon-archive { color: #a855f7; font-size: 16px; }
.icon-file { color: #8b949e; font-size: 16px; }

.file-size {
  color: #8b949e;
  font-family: monospace;
}

.perm-code {
  font-family: monospace;
  font-size: 11.5px;
  color: #8b949e;
  background: #161b22;
  padding: 2px 6px;
  border-radius: 4px;
}

.sftp-op-cell {
  display: flex;
  align-items: center;
  gap: 4px;
}

.sftp-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 16px;
  background: #161b22;
  border-top: 1px solid #30363d;
  font-size: 11.5px;
  color: #8b949e;
  flex-shrink: 0;
}

.sftp-stats code {
  color: #58a6ff;
  background: #0d1117;
  padding: 1px 5px;
  border-radius: 3px;
}

/* ================= 在线编辑器暗黑样式 ================= */
.sftp-editor-dialog :deep(.el-dialog__body) {
  padding: 12px 20px;
  background: #0d1117;
}

.editor-header-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  background: #161b22;
  border: 1px solid #30363d;
  border-radius: 6px 6px 0 0;
}

.editor-file-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.editor-file-path {
  font-weight: 600;
  color: #f0f6fc;
  font-size: 13px;
}

.editor-file-size {
  font-size: 12px;
  color: #8b949e;
}

.editor-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.editor-shortcut-tip {
  font-size: 12px;
  color: #8b949e;
}

.editor-body {
  border: 1px solid #30363d;
  border-top: none;
  border-radius: 0 0 6px 6px;
  background: #0d1117;
}

.code-editor-textarea {
  width: 100%;
  height: 480px;
  background-color: #0d1117;
  color: #e6edf3;
  border: none;
  padding: 12px;
  font-family: Consolas, "Fira Code", Monaco, Menlo, monospace;
  font-size: 13.5px;
  line-height: 1.5;
  resize: vertical;
  outline: none;
  box-sizing: border-box;
}

.code-editor-textarea:focus {
  background-color: #090d13;
}

/* 右键菜单 */
.custom-context-menu {
  position: fixed;
  z-index: 3000;
  background: #1c2128;
  border: 1px solid #30363d;
  border-radius: 6px;
  padding: 4px 0;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.45);
  min-width: 180px;
}

.menu-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 7px 14px;
  font-size: 12.5px;
  color: #c9d1d9;
  cursor: pointer;
  transition: all 0.15s;
}

.menu-item:hover {
  background: #30363d;
  color: #58a6ff;
}

.menu-divider {
  height: 1px;
  background: #30363d;
  margin: 4px 0;
}

.fade-enter-active, .fade-leave-active {
  transition: opacity 0.3s;
}
.fade-enter-from, .fade-leave-to {
  opacity: 0;
}
</style>
