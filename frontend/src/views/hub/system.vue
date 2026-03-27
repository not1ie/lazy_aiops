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

    <div class="workbench-toolbar">
      <div class="workbench-toolbar-left">
        <span class="workbench-toolbar-label">系统工作台</span>
        <el-check-tag
          v-for="tab in visibleTabs"
          :key="tab.key"
          :checked="activeTab === tab.key"
          @change="() => changeTab(tab.key)"
        >
          {{ tab.label }}
        </el-check-tag>
      </div>
    </div>

    <div class="system-hub-panel">
      <component v-if="activeComponent" :is="activeComponent" :key="renderKey" />
      <el-empty v-else description="当前账号暂无系统管理权限" />
    </div>
  </el-card>
</template>

<script setup>
import { computed, defineAsyncComponent, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

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

const activeComponent = computed(() => {
  const item = visibleTabs.value.find((tab) => tab.key === activeTab.value)
  return item?.component || null
})

const refreshCurrentTab = () => {
  renderKey.value += 1
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
  },
  { immediate: true }
)
</script>

<style scoped>
.system-hub-card {
  border-radius: 16px;
}

.workbench-toolbar {
  margin-bottom: 12px;
  padding: 10px 12px;
  border-radius: 12px;
  border: 1px solid var(--el-border-color-light);
  background: var(--el-fill-color-extra-light);
}

.workbench-toolbar-left {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.workbench-toolbar-label {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  margin-right: 4px;
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
