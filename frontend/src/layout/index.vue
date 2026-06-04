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

      <nav class="sider-nav">
        <a class="nav-item" :class="{ active: route.path === '/dashboard' }" @click="go('/dashboard')">仪表盘</a>
        <a class="nav-item" :class="{ active: route.path.startsWith('/ai') }" @click="go('/ai')">智能助手</a>
        <a class="nav-item" :class="{ active: route.path.startsWith('/asset') || route.path === '/host' }" @click="go('/asset')">资产与安全</a>
        <a class="nav-item" :class="{ active: route.path.startsWith('/k8s') }" @click="go('/k8s')">容器平台</a>
        <a class="nav-item" :class="{ active: route.path.startsWith('/monitor') }" @click="go('/monitor')">
          统一观测
          <span v-if="sidebarCounts.alerts > 0" class="nav-badge">{{ sidebarCounts.alerts }}</span>
        </a>
        <a class="nav-item" :class="{ active: route.path.startsWith('/delivery') }" @click="go('/delivery')">
          变更交付
          <span v-if="sidebarCounts.tickets > 0" class="nav-badge warn">{{ sidebarCounts.tickets }}</span>
        </a>
        <a class="nav-item" :class="{ active: route.path.startsWith('/system') }" @click="go('/system')">系统治理</a>
      </nav>
    </aside>

    <div class="right-panel">
      <header class="header motion-down">
        <div class="header-left">
          <el-breadcrumb separator="/">
            <el-breadcrumb-item :to="{ path: '/' }">控制台</el-breadcrumb-item>
            <el-breadcrumb-item v-if="route.meta.title">{{ route.meta.title }}</el-breadcrumb-item>
          </el-breadcrumb>
        </div>

        <div class="header-right">
          <el-button class="search-btn" circle @click="globalSearchVisible = true">
            <el-icon :size="16"><Search /></el-icon>
          </el-button>

          <el-dropdown trigger="click" @command="handleUserCommand">
            <div class="user-chip">
              <el-avatar :size="28" icon="UserFilled" />
              <el-icon><ArrowDown /></el-icon>
            </div>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item disabled>
                  <div class="user-dropdown-header">
                    <div class="ud-name">{{ username }}</div>
                    <div class="ud-role">{{ roleCode }}</div>
                  </div>
                </el-dropdown-item>
                <el-dropdown-item divided command="password">
                  <el-icon><Lock /></el-icon>
                  修改密码
                </el-dropdown-item>
                <el-dropdown-item command="theme">
                  <el-icon><component :is="isDark ? 'Sunny' : 'Moon'" /></el-icon>
                  {{ isDark ? '浅色模式' : '深色模式' }}
                </el-dropdown-item>
                <el-dropdown-item command="logout" divided>
                  <el-icon><SwitchButton /></el-icon>
                  退出登录
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </header>

      <main class="main">
        <router-view v-slot="{ Component, route }">
          <transition name="app-route-fade" mode="out-in">
            <component :is="Component" :key="route.fullPath" />
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

const handleUserCommand = (cmd) => {
  if (cmd === 'password') { changePasswordVisible.value = true; return }
  if (cmd === 'theme') { toggleTheme(); return }
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
  margin: 10px 0 10px 10px;
  width: 200px;
  border-radius: 20px;
  background: var(--glass-bg-heavy);
  backdrop-filter: var(--glass-blur);
  -webkit-backdrop-filter: var(--glass-blur);
  border: 1px solid var(--glass-border);
  box-shadow: var(--shadow-lg);
  display: flex;
  flex-direction: column;
  z-index: 100;
}

.logo { padding: 18px 16px 12px; display: flex; align-items: center; gap: 8px; }
.logo-icon {
  width: 30px; height: 30px; border-radius: 8px;
  background: var(--apple-blue); color: #fff;
  display: flex; align-items: center; justify-content: center;
  font-size: 16px; box-shadow: 0 3px 10px rgba(0,113,227,0.35);
}
.logo-title { font-size: 15px; font-weight: 700; letter-spacing: -0.02em; }
.logo-subtitle { font-size: 9px; font-weight: 600; letter-spacing: 0.1em; color: var(--el-text-color-secondary); }

.right-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  overflow: hidden;
}

.header {
  height: 42px; margin: 10px 14px 0 14px;
  padding: 0 10px 0 14px;
  display: flex; align-items: center; justify-content: space-between;
  border-radius: 12px;
  background: var(--glass-bg); backdrop-filter: var(--glass-blur);
  -webkit-backdrop-filter: var(--glass-blur);
  border: 1px solid var(--glass-outline); box-shadow: var(--shadow-sm);
  flex-shrink: 0;
}
.header-left { display: flex; align-items: center; }
.header-right { display: flex; align-items: center; gap: 2px; }
.search-btn { color: var(--el-text-color-secondary); background: transparent !important; border: none !important; box-shadow: none !important; }
.search-btn:hover { color: var(--el-text-color-primary); background: rgba(0,0,0,0.04) !important; }
.user-chip { display: flex; align-items: center; gap: 4px; padding: 3px 6px 3px 3px; border-radius: 8px; cursor: pointer; }
.user-chip:hover { background: rgba(0,0,0,0.04); }
.user-dropdown-header { padding: 4px 0; }
.ud-name { font-size: 13px; font-weight: 700; }
.ud-role { font-size: 11px; color: var(--el-text-color-secondary); }

.main { flex: 1; padding: 12px 14px 14px 14px; overflow: auto; }

/* Sidebar Nav */
.sider-nav {
  display: flex;
  flex-direction: column;
  padding: 8px 12px;
  gap: 2px;
}
.nav-section {
  font-size: 10px;
  font-weight: 700;
  color: var(--el-text-color-placeholder);
  text-transform: uppercase;
  letter-spacing: 0.06em;
  padding: 16px 8px 6px;
}
.nav-item {
  display: flex;
  align-items: center;
  padding: 8px 12px;
  border-radius: 8px;
  font-size: 13px;
  font-weight: 500;
  color: var(--el-text-color-regular);
  cursor: pointer;
  text-decoration: none;
  transition: all 0.15s;
}
.nav-item:hover { background: rgba(0,0,0,0.04); color: var(--el-text-color-primary); }
.nav-item.active { background: var(--apple-blue); color: #fff; font-weight: 600; }
.nav-badge {
  margin-left: auto;
  background: #ef4444;
  color: #fff;
  font-size: 10px;
  font-weight: 700;
  min-width: 18px;
  height: 18px;
  border-radius: 9px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 5px;
}
.nav-badge.warn { background: #f59e0b; }

.app-route-fade-enter-active,
.app-route-fade-leave-active {
  transition: opacity 0.3s cubic-bezier(0.32, 0.72, 0, 1), transform 0.3s;
}

.app-route-fade-enter-from { opacity: 0; transform: translateY(10px); }
.app-route-fade-leave-to { opacity: 0; transform: translateY(-10px); }
</style>
