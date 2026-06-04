<template>
  <el-container class="layout-container" v-loading="loading">
    <aside class="aside motion-left">
      <div class="logo">
        <div class="logo-icon"><el-icon><MagicStick /></el-icon></div>
        <div class="logo-text">
          <div class="logo-title">Lazy AIOps</div>
          <div class="logo-subtitle">Unified Control</div>
        </div>
      </div>

      <el-scrollbar class="sider-scroll">
        <el-menu
          router
          :default-active="activeMenuIndex"
          background-color="transparent"
          text-color="var(--sider-text)"
          class="el-menu-vertical"
        >
          <el-menu-item index="/dashboard">
            <el-icon><Odometer /></el-icon>
            <span>运行仪表盘</span>
          </el-menu-item>

          <el-menu-item v-if="can('ai')" index="/ai">
            <el-icon><MagicStick /></el-icon>
            <span>AI 智能助手</span>
          </el-menu-item>

          <el-menu-item v-if="can('ai')" index="/ai-skills">
            <el-icon><SetUp /></el-icon>
            <span>AI 技能管理</span>
          </el-menu-item>

          <el-menu-item v-if="can('cmdb')" index="/asset">
            <el-icon><Monitor /></el-icon>
            <span>资产与安全</span>
          </el-menu-item>

          <el-menu-item v-if="can('k8s')" index="/k8s">
            <el-icon><Platform /></el-icon>
            <span>容器编排</span>
          </el-menu-item>

          <el-menu-item v-if="can('monitor')" index="/monitor">
            <el-icon><Histogram /></el-icon>
            <span>统一观测</span>
            <el-badge v-if="sidebarCounts.alerts > 0" :value="sidebarCounts.alerts" class="menu-badge" />
          </el-menu-item>

          <el-menu-item v-if="can('cicd')" index="/delivery">
            <el-icon><Connection /></el-icon>
            <span>交付与流程</span>
            <el-badge v-if="sidebarCounts.tickets > 0" :value="sidebarCounts.tickets" class="menu-badge" type="warning" />
          </el-menu-item>

          <el-menu-item v-if="can('system')" index="/system">
            <el-icon><Setting /></el-icon>
            <span>系统治理</span>
          </el-menu-item>
        </el-menu>
      </el-scrollbar>
    </aside>

    <div class="right-panel">
      <header class="header motion-down">
        <div class="header-left">
          <div class="header-eyebrow">Enterprise</div>
          <el-breadcrumb separator="/">
            <el-breadcrumb-item :to="{ path: '/' }">控制台</el-breadcrumb-item>
            <el-breadcrumb-item v-if="route.meta.title">{{ route.meta.title }}</el-breadcrumb-item>
          </el-breadcrumb>
        </div>

        <div class="header-right">
          <div class="search-trigger" @click="globalSearchVisible = true">
            <el-icon><Search /></el-icon>
            <span class="search-placeholder">搜索...</span>
            <kbd class="search-kbd">⌘K</kbd>
          </div>

          <div class="user-chip" @click="changePasswordVisible = true">
            <el-avatar :size="24" icon="UserFilled" />
            <div class="user-meta">
              <strong>{{ username }}</strong>
              <span>{{ roleCode }}</span>
            </div>
            <el-icon><ArrowDown /></el-icon>
          </div>

          <el-dropdown trigger="click" @command="handleHeaderCommand">
            <el-button circle icon="Operation" />
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="theme">
                  <el-icon><component :is="isDark ? 'Sunny' : 'Moon'" /></el-icon>
                  {{ isDark ? '浅色模式' : '深色模式' }}
                </el-dropdown-item>
                <el-dropdown-item command="logout" divided>
                  <el-icon><Close /></el-icon>
                  退出登录
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </header>

      <main class="main">
        <div class="view-tabs-wrap">
          <el-scrollbar>
            <div class="view-tabs">
              <div
                v-for="item in viewTabs"
                :key="item.path"
                class="view-tab"
                :class="{ 'is-active': route.path === item.path }"
                @click="go(item.path)"
              >
                {{ item.title }}
                <el-icon v-if="item.closable" class="tab-close" @click.stop="closeTab(item.path)"><Close /></el-icon>
              </div>
            </div>
          </el-scrollbar>
        </div>

        <router-view v-slot="{ Component, route }">
          <transition name="app-route-fade" mode="out-in">
            <div class="page-view" :key="route.fullPath">
              <component :is="Component" />
            </div>
          </transition>
        </router-view>
      </main>
    </div>

    <!-- 修改密码对话框 -->
    <el-dialog v-model="changePasswordVisible" title="修改密码" width="480px">
      <el-form :model="passwordForm" label-width="100px" label-position="top">
        <el-form-item label="旧密码">
          <el-input v-model="passwordForm.old" type="password" show-password />
        </el-form-item>
        <el-form-item label="新密码">
          <el-input v-model="passwordForm.new" type="password" show-password />
        </el-form-item>
        <el-form-item label="确认新密码">
          <el-input v-model="passwordForm.confirm" type="password" show-password />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="changePasswordVisible = false">取消</el-button>
        <el-button type="primary" :loading="changePasswordSubmitting" @click="submitChangePassword">确认修改</el-button>
      </template>
    </el-dialog>

    <!-- 全局搜索 -->
    <GlobalSearch v-model="globalSearchVisible" />
  </el-container>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref, reactive, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useTheme } from '@/utils/theme'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import GlobalSearch from '@/components/common/GlobalSearch.vue'

const route = useRoute()
const router = useRouter()
const { isDark, toggleTheme } = useTheme()

const globalSearchVisible = ref(false)

// Sidebar notification counts
const sidebarCounts = reactive({ alerts: 0, tickets: 0 })
let sidebarPollTimer = null

const fetchSidebarCounts = async () => {
  try {
    const headers = { Authorization: 'Bearer ' + localStorage.getItem('token') }
    const [alertRes, ticketRes] = await Promise.all([
      axios.get('/api/v1/alert/alerts', { headers }).catch(() => ({ data: {} })),
      axios.get('/api/v1/workorder/tickets', { headers }).catch(() => ({ data: {} }))
    ])
    const alertList = alertRes.data?.data || []
    const ticketList = ticketRes.data?.data || []
    sidebarCounts.alerts = alertList.filter(a => a.status === 'firing').length
    sidebarCounts.tickets = ticketList.filter(t => t.status === 'pending' || t.status === 'open').length
  } catch (e) { /* silent */ }
}

const onGlobalKeydown = (e) => {
  if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
    e.preventDefault()
    globalSearchVisible.value = true
  }
}
const username = ref('Admin')
const roleCode = ref(localStorage.getItem('role_code') || '')
const permissions = ref(new Set(JSON.parse(localStorage.getItem('permissions') || '[]')))

const TABS_STORAGE_KEY = 'layout_view_tabs_v4'
const TAB_LIMIT = 15

const loading = ref(false)
const changePasswordVisible = ref(false)
const changePasswordSubmitting = ref(false)
const passwordForm = reactive({ old: '', new: '', confirm: '' })

const viewTabs = ref(JSON.parse(localStorage.getItem(TABS_STORAGE_KEY) || '[]'))
if (viewTabs.value.length === 0) {
  viewTabs.value.push({ title: '仪表盘', path: '/dashboard', closable: false })
}

const can = (perm) => permissions.value.has(perm) || roleCode.value === 'admin'

const activeMenuIndex = computed(() => {
  const path = route.path
  if (path === '/dashboard' || path === '/ai') return path
  if (path.startsWith('/asset') || path.startsWith('/host')) return '/asset'
  if (path.startsWith('/k8s') || path.startsWith('/docker')) return '/k8s'
  if (path.startsWith('/monitor') || path.startsWith('/alert') || path.startsWith('/notify') || path.startsWith('/domain')) return '/monitor'
  if (path.startsWith('/delivery') || path.startsWith('/cicd') || path.startsWith('/workorder') || path.startsWith('/workflow')) return '/delivery'
  if (path.startsWith('/system')) return '/system'
  return path
})

const handleHeaderCommand = (cmd) => {
  if (cmd === 'theme') toggleTheme()
  if (cmd === 'logout') {
    localStorage.clear()
    router.push('/login')
  }
}

const go = (path) => router.push(path)

const closeTab = (path) => {
  const idx = viewTabs.value.findIndex(t => t.path === path)
  if (idx > -1) {
    viewTabs.value.splice(idx, 1)
    if (route.path === path) {
      router.push(viewTabs.value[viewTabs.value.length - 1].path)
    }
  }
}

watch(() => route.path, (path) => {
  if (path === '/login') return
  const title = route.meta.title || '新页面'
  if (route.meta.hidden) return // Don't add hidden sub-routes to tabs
  if (!viewTabs.value.some(t => t.path === path)) {
    viewTabs.value.push({ title, path, closable: path !== '/dashboard' })
    if (viewTabs.value.length > TAB_LIMIT) viewTabs.value.splice(1, 1)
  }
  localStorage.setItem(TABS_STORAGE_KEY, JSON.stringify(viewTabs.value))
}, { immediate: true })

onMounted(() => {
  const userInfo = JSON.parse(localStorage.getItem('user_info') || '{}')
  if (userInfo.nickname) username.value = userInfo.nickname
  document.addEventListener('keydown', onGlobalKeydown)
  fetchSidebarCounts()
  sidebarPollTimer = setInterval(fetchSidebarCounts, 60000)
})

onUnmounted(() => {
  document.removeEventListener('keydown', onGlobalKeydown)
  clearInterval(sidebarPollTimer)
})

const submitChangePassword = async () => {
  if (passwordForm.new !== passwordForm.confirm) return ElMessage.error('两次输入密码不一致')
  changePasswordSubmitting.value = true
  try {
    const authHeaders = { Authorization: 'Bearer ' + localStorage.getItem('token') }
    await axios.post('/api/v1/system/user/password', { old_password: passwordForm.old, new_password: passwordForm.new }, { headers: authHeaders })
    ElMessage.success('修改成功，请重新登录')
    localStorage.clear()
    router.push('/login')
  } catch (e) { ElMessage.error('修改失败') }
  finally { changePasswordSubmitting.value = false }
}
</script>

<style scoped>
.layout-container {
  height: 100vh;
  width: 100vw;
  display: flex;
  overflow: hidden;
}

.aside {
  margin: 16px 0 16px 16px;
  width: 240px !important;
  border-radius: 24px;
  background: var(--glass-bg) !important;
  backdrop-filter: var(--glass-blur);
  -webkit-backdrop-filter: var(--glass-blur);
  border: 1px solid var(--glass-outline);
  box-shadow: var(--surface-shadow), inset 0 1px 1px var(--glass-border);
  display: flex;
  flex-direction: column;
  z-index: 100;
  overflow: hidden;
}

.logo {
  padding: 28px 24px 20px;
  display: flex;
  align-items: center;
  gap: 12px;
}

.logo-icon {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  background: var(--apple-blue);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  box-shadow: 0 4px 12px rgba(0, 113, 227, 0.3);
}

.logo-title {
  color: var(--el-text-color-primary);
  font-size: 18px;
  font-weight: 700;
  letter-spacing: -0.02em;
}

.logo-subtitle {
  color: var(--el-text-color-secondary);
  font-size: 9px;
  font-weight: 600;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

.right-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  overflow: hidden;
}

.header {
  height: 52px !important;
  margin: 16px 16px 0 16px;
  padding: 0 20px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-radius: 14px;
  background: var(--glass-bg);
  backdrop-filter: var(--glass-blur);
  -webkit-backdrop-filter: var(--glass-blur);
  box-shadow: var(--surface-shadow), inset 0 1px 1px var(--glass-border);
  border: 1px solid var(--glass-outline);
}

.header-eyebrow {
  color: #fff;
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  background: var(--apple-blue);
  padding: 4px 8px;
  border-radius: 8px;
}

.search-trigger {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 14px;
  border-radius: 10px;
  background: rgba(0, 0, 0, 0.04);
  cursor: pointer;
  font-size: 13px;
  color: var(--el-text-color-secondary);
  transition: all 0.15s;
  min-width: 200px;
}
.search-trigger:hover { background: rgba(0, 0, 0, 0.06); color: var(--el-text-color-regular); }
.search-placeholder { flex: 1; }
.search-kbd {
  font-family: monospace;
  font-size: 11px;
  padding: 2px 6px;
  border-radius: 4px;
  border: 1px solid var(--el-border-color);
  background: var(--el-fill-color);
  color: var(--el-text-color-secondary);
}

.user-chip {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 12px;
  border-radius: 10px;
  background: rgba(0, 0, 0, 0.04);
  cursor: pointer;
}

.view-tabs-wrap {
  height: 44px;
  margin: 12px 16px 0;
  padding: 0 8px;
  display: flex;
  align-items: center;
  background: var(--glass-bg);
  backdrop-filter: var(--glass-blur);
  border-radius: 12px;
  border: 1px solid var(--glass-outline);
}

.view-tabs {
  display: flex;
  gap: 6px;
  padding: 4px 0;
}

.view-tab {
  height: 28px;
  padding: 0 12px;
  border-radius: 8px;
  font-size: 12px;
  display: flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  color: var(--el-text-color-regular);
  transition: all 0.2s;
}

.view-tab.is-active {
  background: var(--apple-blue);
  color: #fff;
  font-weight: 600;
}

.tab-close { font-size: 10px; }

.main {
  flex: 1;
  padding: 16px;
  overflow: auto;
}

:deep(.el-menu) {
  border-right: none;
  background: transparent !important;
}

:deep(.el-menu-item) {
  height: 40px !important;
  line-height: 40px !important;
  margin: 4px 12px !important;
  border-radius: 10px !important;
  color: var(--el-text-color-regular) !important;
  font-weight: 500;
}

:deep(.el-menu-item.is-active) {
  background: var(--apple-blue) !important;
  color: #fff !important;
}

.page-view {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.app-route-fade-enter-active,
.app-route-fade-leave-active {
  transition: opacity 0.3s cubic-bezier(0.32, 0.72, 0, 1), transform 0.3s;
}

.app-route-fade-enter-from { opacity: 0; transform: translateY(10px); }
.app-route-fade-leave-to { opacity: 0; transform: translateY(-10px); }
.menu-badge {
  position: absolute;
  right: 16px;
  top: 50%;
  transform: translateY(-50%);
}
:deep(.el-menu-item) { position: relative; }
</style>
