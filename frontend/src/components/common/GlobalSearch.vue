<template>
  <el-dialog
    v-model="visible"
    title=""
    width="620px"
    :show-close="false"
    class="global-search-dialog"
    append-to-body
    @opened="onOpened"
    @closed="onClosed"
  >
    <div class="search-input-wrap">
      <el-icon class="search-icon" :size="18"><Search /></el-icon>
      <input
        ref="inputRef"
        v-model="keyword"
        class="search-input"
        placeholder="搜索主机、IP、告警、流水线、工单..."
        @input="onInput"
        @keydown="onKeydown"
        autocomplete="off"
      />
      <kbd class="esc-hint">esc</kbd>
    </div>

    <div class="search-body" v-loading="searching">
      <!-- 无输入提示 -->
      <div v-if="!keyword.trim()" class="search-empty">
        <div class="hotkey-hints">
          <span class="hint-item"><kbd>⌘K</kbd> 快速搜索</span>
          <span class="hint-item"><kbd>↑↓</kbd> 导航</span>
          <span class="hint-item"><kbd>Enter</kbd> 跳转</span>
        </div>
        <div class="quick-links">
          <div class="ql-title">快捷入口</div>
          <div class="ql-grid">
            <div class="ql-item" v-for="link in quickLinks" :key="link.path" @click="go(link.path)">
              <el-icon :size="16"><component :is="link.icon" /></el-icon>
              <span>{{ link.label }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- 搜索结果 -->
      <template v-else>
        <div v-if="totalCount === 0 && !searching" class="search-empty">
          <el-empty description="未找到匹配结果" :image-size="50" />
        </div>

        <div v-else class="search-results">
          <!-- 主机 -->
          <div class="result-group" v-if="results.hosts.length > 0">
            <div class="group-label">主机资产 ({{ results.hosts.length }})</div>
            <div
              class="result-item"
              v-for="(item, idx) in results.hosts"
              :key="'h-' + item.id"
              :class="{ active: activeIndex === getGlobalIdx('hosts', idx) }"
              @click="go('/host?highlight=' + item.id)"
              @mouseenter="activeIndex = getGlobalIdx('hosts', idx)"
            >
              <el-icon><Monitor /></el-icon>
              <span class="ri-name">{{ item.name }}</span>
              <span class="ri-ip">{{ item.ip }}</span>
              <el-tag size="small" :type="item.status === 1 ? 'success' : 'danger'" effect="plain">{{ item.status === 1 ? '在线' : '离线' }}</el-tag>
            </div>
          </div>

          <!-- 告警 -->
          <div class="result-group" v-if="results.alerts.length > 0">
            <div class="group-label">活跃告警 ({{ results.alerts.length }})</div>
            <div
              class="result-item"
              v-for="(item, idx) in results.alerts"
              :key="'a-' + item.id"
              :class="{ active: activeIndex === getGlobalIdx('alerts', idx) }"
              @click="go('/alert/events')"
              @mouseenter="activeIndex = getGlobalIdx('alerts', idx)"
            >
              <el-icon><WarningFilled /></el-icon>
              <span class="ri-name">{{ item.alert_name || item.rule_name || '告警' }}</span>
              <span class="ri-ip">{{ item.target || '' }}</span>
              <el-tag size="small" :type="item.severity === 'critical' ? 'danger' : 'warning'" effect="plain">{{ item.severity }}</el-tag>
            </div>
          </div>

          <!-- 流水线 -->
          <div class="result-group" v-if="results.pipelines.length > 0">
            <div class="group-label">CI/CD 流水线 ({{ results.pipelines.length }})</div>
            <div
              class="result-item"
              v-for="(item, idx) in results.pipelines"
              :key="'p-' + item.id"
              :class="{ active: activeIndex === getGlobalIdx('pipelines', idx) }"
              @click="go('/cicd/pipelines')"
              @mouseenter="activeIndex = getGlobalIdx('pipelines', idx)"
            >
              <el-icon><Connection /></el-icon>
              <span class="ri-name">{{ item.name }}</span>
              <span class="ri-ip">{{ item.provider || '' }}</span>
              <el-tag size="small" :type="item.status === 1 ? 'success' : 'info'" effect="plain">{{ item.status === 1 ? '启用' : '停用' }}</el-tag>
            </div>
          </div>

          <!-- 工单 -->
          <div class="result-group" v-if="results.workorders.length > 0">
            <div class="group-label">工单 ({{ results.workorders.length }})</div>
            <div
              class="result-item"
              v-for="(item, idx) in results.workorders"
              :key="'w-' + item.id"
              :class="{ active: activeIndex === getGlobalIdx('workorders', idx) }"
              @click="go('/workorder/tickets')"
              @mouseenter="activeIndex = getGlobalIdx('workorders', idx)"
            >
              <el-icon><Tickets /></el-icon>
              <span class="ri-name">{{ item.title || item.id }}</span>
              <span class="ri-ip">{{ item.creator || '' }}</span>
              <el-tag size="small" type="warning" effect="plain">{{ getWorkOrderStatusLabel(item.status) }}</el-tag>
            </div>
          </div>
        </div>
      </template>
    </div>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, computed, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'

const props = defineProps({
  modelValue: Boolean
})
const emit = defineEmits(['update:modelValue'])

const router = useRouter()
const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v)
})

const keyword = ref('')
const searching = ref(false)
const activeIndex = ref(0)
const inputRef = ref(null)
let debounceTimer = null

const results = reactive({
  hosts: [],
  alerts: [],
  pipelines: [],
  workorders: []
})

const totalCount = computed(() =>
  results.hosts.length + results.alerts.length + results.pipelines.length + results.workorders.length
)

// Flatten all results for keyboard navigation
const flatResults = computed(() => {
  const flat = []
  results.hosts.forEach((item, i) => flat.push({ group: 'hosts', idx: i, item, path: '/host?highlight=' + item.id }))
  results.alerts.forEach((item, i) => flat.push({ group: 'alerts', idx: i, item, path: '/alert/events' }))
  results.pipelines.forEach((item, i) => flat.push({ group: 'pipelines', idx: i, item, path: '/cicd/pipelines' }))
  results.workorders.forEach((item, i) => flat.push({ group: 'workorders', idx: i, item, path: '/workorder/tickets' }))
  return flat
})

const getGlobalIdx = (group, idx) => {
  let offset = 0
  if (group === 'hosts') return offset + idx
  offset += results.hosts.length
  if (group === 'alerts') return offset + idx
  offset += results.alerts.length
  if (group === 'pipelines') return offset + idx
  offset += results.pipelines.length
  if (group === 'workorders') return offset + idx
  return idx
}

const quickLinks = [
  { label: '仪表盘', path: '/dashboard', icon: 'Odometer' },
  { label: '资产中心', path: '/asset', icon: 'Monitor' },
  { label: '容器平台', path: '/k8s', icon: 'Platform' },
  { label: '监控中心', path: '/monitor', icon: 'Histogram' },
  { label: '交付中心', path: '/delivery', icon: 'Connection' },
  { label: '系统治理', path: '/system', icon: 'Setting' },
  { label: 'AI 助手', path: '/ai', icon: 'MagicStick' },
  { label: '终端', path: '/terminal', icon: 'Monitor' }
]

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const onInput = () => {
  clearTimeout(debounceTimer)
  const kw = keyword.value.trim()
  if (!kw) {
    results.hosts = []
    results.alerts = []
    results.pipelines = []
    results.workorders = []
    searching.value = false
    return
  }
  searching.value = true
  activeIndex.value = 0
  debounceTimer = setTimeout(() => performSearch(kw), 250)
}

const performSearch = async (kw) => {
  try {
    const headers = authHeaders()
    const [hostsRes, alertsRes, pipelinesRes, workordersRes] = await Promise.all([
      axios.get('/api/v1/cmdb/hosts', { headers, params: { keyword: kw, size: 5 } }).catch(() => ({ data: {} })),
      axios.get('/api/v1/alert/alerts', { headers }).catch(() => ({ data: {} })),
      axios.get('/api/v1/cicd/pipelines', { headers }).catch(() => ({ data: {} })),
      axios.get('/api/v1/workorder/tickets', { headers, params: { size: 5 } }).catch(() => ({ data: {} }))
    ])

    const hostList = hostsRes.data?.data || []
    const alertList = (alertsRes.data?.data || []).filter(a => a.status === 0)
    const pipelineList = pipelinesRes.data?.data || []
    const workorderList = (workordersRes.data?.data || []).filter(w => w.status === 0 || w.status === 1 || w.status === 4)

    const lower = kw.toLowerCase()
    results.hosts = hostList.filter(h =>
      (h.name || '').toLowerCase().includes(lower) ||
      (h.ip || '').includes(lower) ||
      (h.group_name || '').toLowerCase().includes(lower)
    ).slice(0, 5)

    results.alerts = alertList.filter(a =>
      (a.alert_name || a.rule_name || '').toLowerCase().includes(lower) ||
      (a.target || '').toLowerCase().includes(lower) ||
      (a.severity || '').toLowerCase().includes(lower)
    ).slice(0, 5)

    results.pipelines = pipelineList.filter(p =>
      (p.name || '').toLowerCase().includes(lower) ||
      (p.provider || '').toLowerCase().includes(lower)
    ).slice(0, 5)

    results.workorders = workorderList.filter(w =>
      (w.title || '').toLowerCase().includes(lower) ||
      (w.creator || '').toLowerCase().includes(lower) ||
      getWorkOrderStatusLabel(w.status).includes(lower)
    ).slice(0, 5)
  } catch (e) {
    // silent
  } finally {
    searching.value = false
  }
}

const onKeydown = (e) => {
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    activeIndex.value = Math.min(activeIndex.value + 1, flatResults.value.length - 1)
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    activeIndex.value = Math.max(activeIndex.value - 1, 0)
  } else if (e.key === 'Enter') {
    e.preventDefault()
    const selected = flatResults.value[activeIndex.value]
    if (selected) {
      visible.value = false
      router.push(selected.path)
    }
  }
}

const go = (path) => {
  visible.value = false
  router.push(path)
}

const onOpened = () => {
  nextTick(() => {
    inputRef.value?.focus()
  })
}

const getWorkOrderStatusLabel = (status) => {
  const map = {
    0: '待审批',
    1: '审批中',
    2: '已通过',
    3: '已拒绝',
    4: '执行中',
    5: '已完成',
    6: '已取消'
  }
  return map[status] || '未知'
}

const onClosed = () => {
  keyword.value = ''
  activeIndex.value = 0
  results.hosts = []
  results.alerts = []
  results.pipelines = []
  results.workorders = []
}
</script>

<style scoped>
.global-search-dialog :deep(.el-dialog__body) { padding: 0; }
.global-search-dialog :deep(.el-dialog__header) { display: none; }

.search-input-wrap {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 24px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.search-icon { color: var(--el-text-color-secondary); flex-shrink: 0; }
.search-input {
  flex: 1;
  border: none;
  outline: none;
  font-size: 16px;
  font-weight: 500;
  background: transparent;
  color: var(--el-text-color-primary);
}
.search-input::placeholder { color: var(--el-text-color-placeholder); font-weight: 400; }
.esc-hint {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 4px;
  background: var(--el-fill-color);
  color: var(--el-text-color-secondary);
  font-family: monospace;
}

.search-body { min-height: 200px; max-height: 460px; overflow-y: auto; }
.search-empty { padding: 40px 24px; }
.hotkey-hints { display: flex; gap: 16px; margin-bottom: 24px; }
.hint-item { font-size: 12px; color: var(--el-text-color-secondary); }
.hint-item kbd {
  font-family: monospace;
  font-size: 11px;
  padding: 1px 6px;
  border-radius: 3px;
  border: 1px solid var(--el-border-color);
  background: var(--el-fill-color);
}

.quick-links { border-top: 1px solid var(--el-border-color-lighter); padding-top: 20px; }
.ql-title { font-size: 12px; font-weight: 700; color: var(--el-text-color-secondary); margin-bottom: 12px; text-transform: uppercase; }
.ql-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 8px; }
.ql-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  border-radius: 8px;
  cursor: pointer;
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-regular);
  transition: background 0.15s;
}
.ql-item:hover { background: var(--el-fill-color-light); }

.search-results { padding: 12px 24px 24px; }
.result-group { margin-bottom: 16px; }
.group-label {
  font-size: 11px;
  font-weight: 700;
  color: var(--el-text-color-secondary);
  text-transform: uppercase;
  margin-bottom: 8px;
  padding: 0 4px;
}
.result-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border-radius: 8px;
  cursor: pointer;
  margin-bottom: 2px;
  transition: background 0.1s;
  font-size: 14px;
}
.result-item:hover, .result-item.active { background: var(--el-fill-color-light); }
.ri-name { font-weight: 600; color: var(--el-text-color-primary); flex: 1; }
.ri-ip { font-size: 12px; color: var(--el-text-color-secondary); font-family: monospace; }
</style>
