<template>
  <el-card class="page-card system-hub-card">
    <div class="page-header">
      <div>
        <h2>系统管理中心</h2>
        <p class="page-desc">账号、角色、权限与审计统一管理入口，减少分散切页。</p>
      </div>
      <div class="page-actions">
        <el-button @click="openProfile">账号信息</el-button>
        <el-button type="primary" plain @click="openPassword">修改密码</el-button>
        <el-button plain @click="refreshCurrentTab">刷新当前</el-button>
      </div>
    </div>

    <div class="hub-tabs-wrap">
      <el-tabs v-model="activeTab" @tab-click="handleSystemTabClick">
        <el-tab-pane
          v-for="tab in visibleTabs"
          :key="tab.key"
          :name="tab.key"
          :label="tab.label"
        />
      </el-tabs>
    </div>

    <el-row :gutter="12" class="mt-12">
      <el-col :span="16">
        <el-card class="capability-card" v-loading="capabilityLoading">
          <template #header>
            <div class="capability-header">
              <span>功能完整度矩阵</span>
              <div class="capability-tags">
                <el-tag type="success" effect="light">完整 {{ capabilitySummary.complete }}</el-tag>
                <el-tag type="warning" effect="light">待补齐 {{ capabilitySummary.partial }}</el-tag>
                <el-tag type="danger" effect="light">缺口 {{ capabilitySummary.gap }}</el-tag>
              </div>
            </div>
          </template>
          <el-table :fit="true" :data="moduleCapabilityRows" size="small" max-height="300" empty-text="暂无模块能力数据">
            <el-table-column prop="label" label="能力模块" min-width="140" />
            <el-table-column label="链路" width="90">
              <template #default="{ row }">
                <el-tag size="small" :type="row.linkStatus === 'ok' ? 'success' : row.linkStatus === 'warning' ? 'warning' : 'danger'">
                  {{ row.linkStatus === 'ok' ? '正常' : row.linkStatus === 'warning' ? '降级' : '异常' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="状态判定" width="90">
              <template #default="{ row }">
                <el-tag size="small" :type="row.status === 'ok' ? 'success' : row.status === 'warning' ? 'warning' : 'danger'">
                  {{ row.status === 'ok' ? '正常' : row.status === 'warning' ? '预警' : '异常' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="自动处置" width="90">
              <template #default="{ row }">
                <el-tag size="small" :type="row.automationScore >= 90 ? 'success' : row.automationScore >= 70 ? 'warning' : 'info'">
                  {{ row.automationScore >= 90 ? '完善' : row.automationScore >= 70 ? '可用' : '较弱' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="目标值" width="78">
              <template #default="{ row }">{{ row.targetScore }}%</template>
            </el-table-column>
            <el-table-column label="当前值" width="110">
              <template #default="{ row }">
                <el-progress :percentage="row.currentScore" :stroke-width="10" :show-text="false" />
                <div class="capability-mini-percent">{{ row.currentScore }}%</div>
              </template>
            </el-table-column>
            <el-table-column label="趋势" width="94">
              <template #default="{ row }">
                <el-tag size="small" :type="capabilityTrendTag(row.trendDirection)">
                  {{ capabilityTrendArrow(row.trendDirection) }} {{ formatTrendDelta(row.trendDelta) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="suggestion" label="建议" min-width="170" show-overflow-tooltip />
            <el-table-column label="操作" width="90" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" @click="openCapability(row.tab)">进入</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card class="capability-gap-card" v-loading="capabilityLoading">
          <template #header>
            <div class="capability-header">
              <span>能力缺口追踪</span>
              <el-tag :type="capabilityGapRows.length ? 'warning' : 'success'" effect="light">缺口 {{ capabilityGapRows.length }}</el-tag>
            </div>
          </template>
          <el-table :fit="true" :data="capabilityGapRows" size="small" max-height="300" empty-text="暂无高优先级缺口">
            <el-table-column prop="module" label="模块" width="110" />
            <el-table-column prop="gap" label="缺口" min-width="150" show-overflow-tooltip />
            <el-table-column prop="impact" label="影响" min-width="170" show-overflow-tooltip />
            <el-table-column label="操作" width="90">
              <template #default="{ row }">
                <el-button link type="primary" @click="openCapability(row.tab)">修复</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <div class="system-hub-panel">
      <component v-if="activeComponent" :is="activeComponent" :key="renderKey" />
      <el-empty v-else description="当前账号暂无系统管理权限" />
    </div>
  </el-card>
</template>

<script setup>
import { computed, defineAsyncComponent, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import axios from 'axios'
import { ElMessage } from 'element-plus'

const route = useRoute()
const router = useRouter()

const tabs = [
  {
    key: 'users',
    label: '用户管理',
    permAny: ['system', 'system:user'],
    component: defineAsyncComponent(() => import('@/views/system/users.vue'))
  },
  {
    key: 'roles',
    label: '角色管理',
    permAny: ['system', 'system:role'],
    component: defineAsyncComponent(() => import('@/views/system/roles.vue'))
  },
  {
    key: 'menus',
    label: '权限管理',
    permAny: ['system', 'system:permission'],
    component: defineAsyncComponent(() => import('@/views/system/menus.vue'))
  },
  {
    key: 'dept',
    label: '部门管理',
    permAny: ['system', 'system:dept'],
    component: defineAsyncComponent(() => import('@/views/system/dept.vue'))
  },
  {
    key: 'posts',
    label: '岗位管理',
    permAny: ['system', 'system:post'],
    component: defineAsyncComponent(() => import('@/views/system/posts.vue'))
  },
  {
    key: 'login-logs',
    label: '登录日志',
    permAny: ['system', 'system:loginlog'],
    component: defineAsyncComponent(() => import('@/views/system/login-logs.vue'))
  },
  {
    key: 'audit-logs',
    label: '操作日志',
    permAny: ['system', 'system:log'],
    component: defineAsyncComponent(() => import('@/views/system/audit-logs.vue'))
  },
  {
    key: 'captcha',
    label: '验证码配置',
    permAny: ['system', 'system:captcha'],
    component: defineAsyncComponent(() => import('@/views/system/captcha.vue'))
  }
]

const isAdmin = computed(() => String(localStorage.getItem('role_code') || '').toLowerCase() === 'admin')
const permissions = computed(() => {
  try {
    const raw = JSON.parse(localStorage.getItem('permissions') || '[]')
    return Array.isArray(raw) ? raw : []
  } catch {
    return []
  }
})

const hasAnyPerm = (codes = []) => {
  if (isAdmin.value) return true
  return codes.some((code) => permissions.value.includes(code))
}

const visibleTabs = computed(() => tabs.filter((tab) => hasAnyPerm(tab.permAny)))
const activeTab = ref('')
const renderKey = ref(0)
const capabilityLoading = ref(false)
const probeStatus = reactive({
  users: 'unknown',
  roles: 'unknown',
  menus: 'unknown',
  dept: 'unknown',
  posts: 'unknown',
  loginLogs: 'unknown',
  auditLogs: 'unknown',
  captcha: 'unknown'
})
const probeCount = reactive({
  users: 0,
  roles: 0,
  menus: 0,
  dept: 0,
  posts: 0,
  loginLogs: 0,
  auditLogs: 0,
  captcha: 0
})

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })
const normalizeText = (value) => String(value ?? '').trim().toLowerCase()
const statusSeverity = { ok: 1, warning: 2, error: 3, unknown: 0 }
const clampPercent = (value) => Math.max(0, Math.min(100, Math.round(Number(value) || 0)))
const capabilityScoreByStatus = (value) => {
  const status = normalizeText(value)
  if (status === 'ok') return 100
  if (status === 'warning') return 72
  if (status === 'error') return 38
  return 50
}
const worstStatus = (...values) =>
  values.reduce((worst, current) => (statusSeverity[normalizeText(current)] > statusSeverity[normalizeText(worst)] ? current : worst), 'ok')
const capabilityTrendByTarget = (currentScore, targetScore) => {
  const delta = Math.round((Number(currentScore) || 0) - (Number(targetScore) || 0))
  if (delta >= 3) return { direction: 'up', delta }
  if (delta <= -3) return { direction: 'down', delta }
  return { direction: 'flat', delta }
}
const capabilityTrendArrow = (direction) => {
  if (direction === 'up') return '↑'
  if (direction === 'down') return '↓'
  return '→'
}
const capabilityTrendTag = (direction) => {
  if (direction === 'up') return 'success'
  if (direction === 'down') return 'danger'
  return 'info'
}
const formatTrendDelta = (value) => {
  const delta = Math.round(Number(value) || 0)
  if (delta > 0) return `+${delta}%`
  return `${delta}%`
}

const tabProbeConfig = {
  users: {
    statusKey: 'users',
    request: () => axios.get('/api/v1/rbac/users', { headers: authHeaders(), params: { page: 1, page_size: 1 } })
  },
  roles: {
    statusKey: 'roles',
    request: () => axios.get('/api/v1/rbac/roles', { headers: authHeaders() })
  },
  menus: {
    statusKey: 'menus',
    request: () => axios.get('/api/v1/rbac/permissions', { headers: authHeaders() })
  },
  dept: {
    statusKey: 'dept',
    request: () => axios.get('/api/v1/system/depts', { headers: authHeaders() })
  },
  posts: {
    statusKey: 'posts',
    request: () => axios.get('/api/v1/system/posts', { headers: authHeaders(), params: { page: 1, page_size: 1 } })
  },
  'login-logs': {
    statusKey: 'loginLogs',
    request: () => axios.get('/api/v1/system/login-logs', { headers: authHeaders(), params: { page: 1, page_size: 1 } })
  },
  'audit-logs': {
    statusKey: 'auditLogs',
    request: () => axios.get('/api/v1/rbac/logs', { headers: authHeaders(), params: { page: 1, page_size: 1 } })
  },
  captcha: {
    statusKey: 'captcha',
    request: () => axios.get('/api/v1/system/captcha', { headers: authHeaders() })
  }
}

const tabStatus = (tabKey) => {
  const config = tabProbeConfig[tabKey]
  return config ? probeStatus[config.statusKey] : 'unknown'
}
const tabCount = (tabKey) => {
  const config = tabProbeConfig[tabKey]
  return config ? Number(probeCount[config.statusKey] || 0) : 0
}
const hasTab = (tabKey) => visibleTabs.value.some((item) => item.key === tabKey)
const openCapability = (tab) => {
  if (!tab) return
  changeTab(tab)
}

const extractProbeCount = (payload) => {
  if (Array.isArray(payload)) return payload.length
  if (Array.isArray(payload?.items)) return payload.items.length
  if (Array.isArray(payload?.list)) return payload.list.length
  if (typeof payload?.total === 'number') return Number(payload.total || 0)
  if (payload && typeof payload === 'object') return 1
  return 0
}

const refreshSystemCapability = async () => {
  const keys = visibleTabs.value.map((item) => item.key).filter((key) => tabProbeConfig[key])
  if (!keys.length) return

  capabilityLoading.value = true
  try {
    const settled = await Promise.allSettled(keys.map((key) => tabProbeConfig[key].request()))
    let failed = 0

    keys.forEach((key, index) => {
      const result = settled[index]
      const { statusKey } = tabProbeConfig[key]
      if (result.status === 'fulfilled' && Number(result.value?.data?.code) === 0) {
        probeStatus[statusKey] = 'ok'
        probeCount[statusKey] = extractProbeCount(result.value?.data?.data)
      } else {
        probeStatus[statusKey] = 'error'
        probeCount[statusKey] = 0
        failed += 1
      }
    })

    if (failed > 0) {
      ElMessage.warning(`系统能力探测存在失败(${failed}项)，已展示可用结果`)
    }
  } finally {
    capabilityLoading.value = false
  }
}

const moduleCapabilityRows = computed(() => {
  const rows = []

  if (hasTab('users') || hasTab('roles')) {
    const linkStatus = worstStatus(tabStatus('users'), tabStatus('roles'))
    const usersCount = tabCount('users')
    const rolesCount = tabCount('roles')
    const status = linkStatus === 'error'
      ? 'error'
      : ((hasTab('users') && usersCount === 0) || (hasTab('roles') && rolesCount === 0) ? 'warning' : 'ok')
    rows.push({
      key: 'identity',
      tab: hasTab('users') ? 'users' : 'roles',
      path: '/system/center?tab=users',
      label: '账号与角色治理',
      linkStatus,
      status,
      automationScore: hasTab('roles') ? 86 : 78,
      targetScore: 90,
      freshnessScore: clampPercent(100 - ((hasTab('users') && usersCount === 0) ? 20 : 0) - ((hasTab('roles') && rolesCount === 0) ? 20 : 0)),
      suggestion: status === 'ok' ? '保持账号生命周期与角色模型同步' : '补齐账号或角色基线，避免权限漂移'
    })
  }

  if (hasTab('menus') || hasTab('roles')) {
    const linkStatus = worstStatus(tabStatus('menus'), tabStatus('roles'))
    const permCount = tabCount('menus')
    const status = linkStatus === 'error' ? 'error' : ((hasTab('menus') && permCount === 0) ? 'warning' : 'ok')
    rows.push({
      key: 'rbac',
      tab: hasTab('menus') ? 'menus' : 'roles',
      path: '/system/center?tab=menus',
      label: '权限模型与授权',
      linkStatus,
      status,
      automationScore: 84,
      targetScore: 89,
      freshnessScore: hasTab('menus') ? (permCount > 0 ? 100 : 72) : 82,
      suggestion: status === 'ok' ? '持续校验权限树与角色绑定一致性' : '补齐权限项与角色授权映射'
    })
  }

  if (hasTab('dept') || hasTab('posts')) {
    const linkStatus = worstStatus(tabStatus('dept'), tabStatus('posts'))
    const deptCount = tabCount('dept')
    const postCount = tabCount('posts')
    const status = linkStatus === 'error'
      ? 'error'
      : ((hasTab('dept') && deptCount === 0) || (hasTab('posts') && postCount === 0) ? 'warning' : 'ok')
    rows.push({
      key: 'org',
      tab: hasTab('dept') ? 'dept' : 'posts',
      path: '/system/center?tab=dept',
      label: '组织与岗位治理',
      linkStatus,
      status,
      automationScore: 82,
      targetScore: 88,
      freshnessScore: clampPercent(100 - ((hasTab('dept') && deptCount === 0) ? 18 : 0) - ((hasTab('posts') && postCount === 0) ? 18 : 0)),
      suggestion: status === 'ok' ? '保持组织结构与岗位基线及时更新' : '补齐部门或岗位数据，避免审批链路断点'
    })
  }

  if (hasTab('login-logs') || hasTab('audit-logs') || hasTab('captcha')) {
    const linkStatus = worstStatus(tabStatus('login-logs'), tabStatus('audit-logs'), tabStatus('captcha'))
    const loginCount = tabCount('login-logs')
    const auditCount = tabCount('audit-logs')
    const captchaCount = tabCount('captcha')
    const status = linkStatus === 'error'
      ? 'error'
      : ((hasTab('login-logs') && loginCount === 0) || (hasTab('audit-logs') && auditCount === 0) || (hasTab('captcha') && captchaCount === 0)
        ? 'warning'
        : 'ok')
    rows.push({
      key: 'audit',
      tab: hasTab('login-logs') ? 'login-logs' : (hasTab('audit-logs') ? 'audit-logs' : 'captcha'),
      path: '/system/center?tab=login-logs',
      label: '审计与安全策略',
      linkStatus,
      status,
      automationScore: 88,
      targetScore: 91,
      freshnessScore: clampPercent(100 - ((hasTab('login-logs') && loginCount === 0) ? 16 : 0) - ((hasTab('audit-logs') && auditCount === 0) ? 16 : 0)),
      suggestion: status === 'ok' ? '保持审计日志与登录策略联动' : '优先修复日志链路与验证码配置可用性'
    })
  }

  return rows.map((item) => {
    const score = clampPercent(
      capabilityScoreByStatus(item.status) * 0.4 +
      capabilityScoreByStatus(item.linkStatus) * 0.3 +
      item.automationScore * 0.2 +
      item.freshnessScore * 0.1
    )
    const targetScore = clampPercent(item.targetScore || 88)
    const trend = capabilityTrendByTarget(score, targetScore)
    const level = score >= 85 ? 'complete' : score >= 65 ? 'partial' : 'gap'
    return {
      ...item,
      targetScore,
      currentScore: score,
      trendDirection: trend.direction,
      trendDelta: trend.delta,
      score,
      level
    }
  }).sort((a, b) => a.score - b.score)
})

const capabilitySummary = computed(() => ({
  complete: moduleCapabilityRows.value.filter((item) => item.level === 'complete').length,
  partial: moduleCapabilityRows.value.filter((item) => item.level === 'partial').length,
  gap: moduleCapabilityRows.value.filter((item) => item.level === 'gap').length
}))

const capabilityGapRows = computed(() =>
  moduleCapabilityRows.value
    .filter((item) => item.level !== 'complete')
    .map((item) => ({
      module: item.label,
      gap: item.level === 'gap' ? '核心管理链路可用性不足' : '存在降级，自动化策略待增强',
      impact: item.level === 'gap' ? '权限与审计链路失真，风险放大' : '处理效率下降，合规追溯成本上升',
      tab: item.tab,
      score: item.score
    }))
    .sort((a, b) => a.score - b.score)
    .slice(0, 8)
)

const syncFromRoute = () => {
  const available = visibleTabs.value
  if (!available.length) {
    activeTab.value = ''
    return
  }
  const queryTab = String(route.query?.tab || '').trim()
  const matched = available.find((item) => item.key === queryTab)
  activeTab.value = matched ? matched.key : available[0].key
}

const updateRouteTab = (tab) => {
  if (!tab) return
  if (String(route.query?.tab || '') === tab) return
  router.replace({ path: '/system/center', query: { ...route.query, tab } })
}

const changeTab = (tab) => {
  if (!tab || activeTab.value === tab) return
  activeTab.value = tab
  updateRouteTab(tab)
}

const handleSystemTabClick = (pane) => {
  const tab = String(pane?.paneName || '').trim()
  if (!tab) return
  changeTab(tab)
}

const activeComponent = computed(() => {
  const item = visibleTabs.value.find((tab) => tab.key === activeTab.value)
  return item?.component || null
})

const refreshCurrentTab = () => {
  renderKey.value += 1
  void refreshSystemCapability()
}

const emitAccountAction = (action) => {
  if (!action || typeof window === 'undefined') return
  window.dispatchEvent(new CustomEvent('lao:system-account-action', { detail: { action } }))
}

const openProfile = () => {
  emitAccountAction('profile')
}

const openPassword = () => {
  emitAccountAction('password')
}

watch(
  () => route.query.tab,
  () => syncFromRoute(),
  { immediate: true }
)

watch(
  () => visibleTabs.value.map((item) => item.key).join(','),
  () => {
    syncFromRoute()
    if (activeTab.value) {
      updateRouteTab(activeTab.value)
    } else if (route.query?.tab) {
      const nextQuery = { ...route.query }
      delete nextQuery.tab
      router.replace({ path: '/system/center', query: nextQuery })
    }
    void refreshSystemCapability()
  },
  { immediate: true }
)
</script>

<style scoped>
.system-hub-card {
  border-radius: 16px;
}

.hub-tabs-wrap {
  margin-bottom: 12px;
}

.hub-tabs-wrap :deep(.el-tabs__header) {
  margin-bottom: 0;
}

.hub-tabs-wrap :deep(.el-tabs__item) {
  height: 36px;
  line-height: 36px;
  font-size: 14px;
  padding: 0 14px;
}

.hub-tabs-wrap :deep(.el-tabs__nav-wrap::after) {
  background-color: var(--el-border-color-light);
}

.hub-tabs-wrap :deep(.el-tabs__active-bar) {
  height: 2px;
}

.hub-tabs-wrap :deep(.el-tabs__content) {
  display: none;
}

.mt-12 {
  margin-top: 12px;
  margin-bottom: 12px;
}

.capability-card,
.capability-gap-card {
  border: 1px solid #e5e7eb;
}

.capability-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.capability-tags {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.capability-mini-percent {
  margin-top: 2px;
  font-size: 11px;
  color: var(--muted-text);
}

.system-hub-panel :deep(.page-card) {
  border: none;
  box-shadow: none;
  padding: 0;
}

.system-hub-panel :deep(.el-card) {
  box-shadow: none;
  border: none;
}
</style>
