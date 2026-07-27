<template>
  <div class="layout-wrapper">
    <!-- Main Central App Window Frame (MANGOS Reference Style) -->
    <el-container class="layout-container" v-loading="loading">
      <!-- Integrated Left Sidebar -->
      <el-aside :width="isCollapsed ? '72px' : '230px'" class="aside">
        <!-- Logo Header (LazyOps Title) -->
        <div class="logo" style="justify-content: center; padding: 0 16px;">
          <span class="logo-brand-title" v-show="!isCollapsed">LazyOps</span>
          <span class="logo-brand-title" v-show="isCollapsed" style="font-size: 16px;">LO</span>
        </div>

        <el-menu
          :default-active="activeMenuIndex"
          class="sider-menu"
          :collapse="isCollapsed"
          :collapse-transition="false"
          @select="handleSelect"
          unique-opened
        >
          <template v-for="menu in menuData" :key="menu.index">
            <el-sub-menu v-if="menu.children && menu.children.length > 0" :index="menu.index">
              <template #title>
                <el-icon v-if="menu.icon"><component :is="menu.icon" /></el-icon>
                <span>{{ menu.title }}</span>
              </template>
              <el-menu-item v-for="child in menu.children" :key="child.index" :index="child.index">
                <el-icon v-if="child.icon"><component :is="child.icon" /></el-icon>
                <span>{{ child.title }}</span>
              </el-menu-item>
            </el-sub-menu>
            <el-menu-item v-else :index="menu.index">
              <el-icon v-if="menu.icon"><component :is="menu.icon" /></el-icon>
              <template #title>{{ menu.title }}</template>
            </el-menu-item>
          </template>
        </el-menu>

        <!-- Bottom Primary Action Button (+ Create Task / Release) -->
        <div class="sidebar-footer" v-show="!isCollapsed">
          <el-button class="sidebar-create-btn" type="primary" icon="Plus" @click="commandPaletteRef?.open()">
            新建运维任务
          </el-button>
        </div>
      </el-aside>

      <!-- Main Container Right Panel -->
      <el-container class="right-panel">
        <el-header class="header">
          <div class="header-left">
            <el-button class="icon-btn collapse-btn" :icon="isCollapsed ? 'Expand' : 'Fold'" @click="isCollapsed = !isCollapsed" />
            <div class="page-title-label">Dashboard</div>
          </div>

          <div class="header-right">
            <!-- MANGOS Sunken Search Input Groove -->
            <div class="search-spotlight-btn" @click="commandPaletteRef?.open()">
              <el-icon><Search /></el-icon>
              <span class="spotlight-placeholder">Search anything...</span>
              <span class="spotlight-kbd">⌘K</span>
            </div>

            <!-- MANGOS Green Status Check-in Badge -->
            <div class="checkin-badge">
              <span class="checkin-dot"></span>
              <span class="checkin-text">SLA 正常</span>
            </div>

            <!-- Notification Icon Button -->
            <div class="notification-btn-box">
              <el-icon><Bell /></el-icon>
              <span class="notification-red-dot"></span>
            </div>

            <el-dropdown trigger="click" @command="handleUserCommand">
              <div class="user-chip">
                <el-avatar :size="32" src="https://images.unsplash.com/photo-1534528741775-53994a69daeb?q=80&w=256&auto=format&fit=crop" />
              </div>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="password">修改密码</el-dropdown-item>
                  <el-dropdown-item command="theme">切换主题</el-dropdown-item>
                  <el-dropdown-item command="logout" divided>退出登录</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </el-header>

        <el-main class="main">
          <router-view v-slot="{ Component, route }">
            <transition name="fade-page" mode="out-in" appear>
              <component :is="Component" :key="route.fullPath" />
            </transition>
          </router-view>
        </el-main>
      </el-container>

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

      <GlobalSearch v-model="globalSearchVisible" />
      <GlobalCommandPalette ref="commandPaletteRef" />
      <LazyCopilotDock />
    </el-container>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref, reactive, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useTheme } from '@/utils/theme'
import axios from 'axios'
import GlobalCommandPalette from '@/components/GlobalCommandPalette.vue'
import LazyCopilotDock from '@/components/LazyCopilotDock.vue'
import GlobalSearch from '@/components/common/GlobalSearch.vue'

const route = useRoute()
const router = useRouter()
const { isDark, toggleTheme } = useTheme()

const isCollapsed = ref(false)
const globalSearchVisible = ref(false)
const commandPaletteRef = ref(null)
const loading = ref(false)
const changePasswordVisible = ref(false)
const changePasswordSubmitting = ref(false)
const passwordForm = reactive({ old: '', new: '', confirm: '' })

const username = ref('Admin')
const roleCode = ref(localStorage.getItem('role_code') || '')

const breadcrumbs = computed(() => {
  const result = []
  const currentPath = route.path
  
  for (const menu of menuData) {
    if (menu.index === currentPath) {
      result.push({ title: menu.title, path: menu.index })
      break
    }
    if (menu.children) {
      const child = menu.children.find(c => c.index === currentPath)
      if (child) {
        result.push({ title: menu.title, path: '' })
        result.push({ title: child.title, path: child.index })
        break
      }
    }
  }
  
  if (result.length === 0 && route.meta && route.meta.title) {
    if (currentPath.startsWith('/k8s/')) {
      result.push({ title: '集群管理', path: '' })
    } else if (currentPath.startsWith('/cmdb/') || currentPath === '/host') {
      result.push({ title: '资产管理', path: '' })
    } else if (currentPath.startsWith('/cicd/')) {
      result.push({ title: '应用发布', path: '' })
    } else if (currentPath.startsWith('/monitor/') || currentPath.startsWith('/alert/') || currentPath === '/remediation/logs' || currentPath.startsWith('/oncall/') || currentPath.startsWith('/notify/') || currentPath.startsWith('/domain/')) {
      result.push({ title: '监控中心', path: '' })
    } else if (currentPath.startsWith('/log/')) {
      result.push({ title: '日志中心', path: '' })
    } else if (currentPath.startsWith('/system/') || currentPath.startsWith('/cost/')) {
      result.push({ title: '系统管理', path: '' })
    }
    result.push({ title: route.meta.title, path: currentPath })
  }
  
  return result
})

const menuData = computed(() => {
  const allMenus = [
    { index: '/dashboard', title: '仪表盘', icon: 'Menu', perm: 'dashboard' },
    { 
      index: 'asset', title: '资产管理', icon: 'Coin', perm: 'cmdb',
      children: [
        { index: '/host', title: '主机管理', perm: 'cmdb' },
        { index: '/cmdb/credential', title: '凭据管理', perm: 'cmdb' },
        { index: '/cmdb/database', title: '数据库资产', perm: 'cmdb' },
        { index: '/cmdb/cloud', title: '云资源', perm: 'cmdb' },
        { index: '/cmdb/network-devices', title: '网络与防火墙', perm: 'cmdb' },
        { index: '/cmdb/topology', title: '服务拓扑', perm: 'cmdb' },
        { index: '/jump/assets', title: '堡垒机接入', perm: 'jump:asset' }
      ]
    },
    {
      index: 'k8s', title: '集群管理', icon: 'Platform', perm: 'k8s',
      children: [
        { index: '/k8s/clusters', title: 'K8s集群', perm: 'k8s' },
        { index: '/docker', title: 'Docker容器', perm: 'docker' },
        { index: '/k8s/workloads', title: '工作负载', perm: 'k8s' },
        { index: '/k8s/pods', title: 'Pods', perm: 'k8s' },
        { index: '/k8s/services', title: '服务发现', perm: 'k8s' }
      ]
    },
    {
      index: 'delivery', title: '应用发布', icon: 'Position', perm: 'cicd',
      children: [
        { index: '/cicd/pipelines', title: '流水线', perm: 'cicd' },
        { index: '/cicd/releases', title: '发布管理', perm: 'cicd' }
      ]
    },
    {
      index: 'monitor', title: '监控中心', icon: 'Histogram', perm: 'monitor',
      children: [
        { index: '/monitor/hosts', title: '主机监控', perm: 'monitor' },
        { index: '/monitor/metrics', title: '数据源配置', perm: 'monitor' },
        { index: '/alert/events', title: '告警事件', perm: 'alert' },
        { index: '/alert/history', title: '告警复盘', perm: 'alert' },
        { index: '/remediation/logs', title: '故障自愈', perm: 'remediation' },
        { index: '/oncall/schedules', title: '值班排班', perm: 'oncall' },
        { index: '/notify/channels', title: '通知渠道', perm: 'notify' },
        { index: '/domain/ssl', title: '域名证书', perm: 'domain' }
      ]
    },
    {
      index: 'log', title: '日志中心', icon: 'Document', perm: 'log',
      children: [
        { index: '/log/query', title: '日志查询', perm: 'log' },
        { index: '/log/library', title: '日志库', perm: 'log' },
        { index: '/log/alerts', title: '日志告警', perm: 'log' },
        { index: '/log/permissions', title: '日志权限', perm: 'log' }
      ]
    },
    {
      index: 'task', title: '作业任务', icon: 'Cpu', perm: 'task',
      children: [
        { index: '/executor', title: '批量执行', perm: 'executor' },
        { index: '/task/schedules', title: '任务调度', perm: 'task' },
        { index: '/workorder/orders', title: '运维工单', perm: 'workorder' },
        { index: '/workflow/workflows', title: '运维编排', perm: 'workflow' },
        { index: '/ansible/playbooks', title: 'Ansible管理', perm: 'ansible' },
        { index: '/sqlaudit/requests', title: 'SQL审计', perm: 'sqlaudit' }
      ]
    },
    {
      index: 'ai', title: 'AIOps智能', icon: 'MagicStick', perm: 'ai',
      children: [
        { index: '/ai/assistant', title: 'AI智能助手', perm: 'ai' },
        { index: '/ai/config', title: '模型接入配置', perm: 'ai' },
        { index: '/ai/ops', title: '故障诊断', perm: 'ai' },
        { index: '/knowledge/docs', title: 'AI知识库', perm: 'knowledge' },
        { index: '/ai-skills', title: 'AI技能管理', perm: 'ai' }
      ]
    },
    {
      index: 'system', title: '系统管理', icon: 'Setting', perm: 'system',
      children: [
        { index: '/system/users', title: '用户管理', perm: 'system:user' },
        { index: '/system/roles', title: '角色管理', perm: 'system:role' },
        { index: '/system/menus', title: '权限管理', perm: 'system:permission' },
        { index: '/cost/overview', title: '成本分析', perm: 'cost' }
      ]
    }
  ]

  let perms = []
  try {
    perms = JSON.parse(localStorage.getItem('permissions') || '[]')
    if (!Array.isArray(perms)) perms = []
  } catch (e) {
    perms = []
  }
  
  const hasPerm = (code) => {
    if (!code) return true
    if (roleCode.value === 'admin') return true
    try {
      const userStr = localStorage.getItem('user_info')
      if (userStr) {
        const user = JSON.parse(userStr)
        if (user?.role === 'admin' || user?.username === 'admin') return true
      }
    } catch (e) {}
    if (perms.includes('*') || perms.includes('all') || perms.includes(code)) return true
    const parts = code.split(':')
    while (parts.length > 1) {
      parts.pop()
      if (perms.includes(parts.join(':'))) return true
    }
    return false
  }

  const filterMenu = (list) => {
    return list.map(item => {
      const copy = { ...item }
      if (copy.children) {
        copy.children = filterMenu(copy.children)
      }
      return copy
    }).filter(item => {
      if (item.children && item.children.length === 0) return false
      return hasPerm(item.perm)
    })
  }

  return filterMenu(allMenus)
})

const activeMenuIndex = computed(() => {
  return route.path
})

const handleSelect = (index) => {
  router.push(index)
}

const go = (path) => router.push(path)

const handleUserCommand = (cmd) => {
  if (cmd === 'password') { changePasswordVisible.value = true; return }
  if (cmd === 'theme') { toggleTheme(); return }
  if (cmd === 'logout') {
    localStorage.clear()
    router.push('/login')
  }
}

onMounted(() => {
  let userInfo = {}
  try {
    userInfo = JSON.parse(localStorage.getItem('user_info') || '{}')
  } catch (e) {
    userInfo = {}
  }
  if (userInfo && userInfo.nickname) username.value = userInfo.nickname
})

const submitChangePassword = async () => {
  if (!passwordForm.old) return ElMessage.error('请输入旧密码')
  if (!passwordForm.new) return ElMessage.error('请输入新密码')
  if (passwordForm.new.length < 6) return ElMessage.error('新密码长度不能小于 6 位')
  if (passwordForm.new === passwordForm.old) return ElMessage.error('新密码不能与旧密码相同')
  if (passwordForm.new === 'admin123') return ElMessage.error('密码过于简单，系统不允许使用 admin123')
  if (passwordForm.new !== passwordForm.confirm) return ElMessage.error('两次输入密码不一致')
  changePasswordSubmitting.value = true
  try {
    const authHeaders = { Authorization: 'Bearer ' + localStorage.getItem('token') }
    await axios.post('/api/v1/system/user/password', { old_password: passwordForm.old, new_password: passwordForm.new }, { headers: authHeaders })
    ElMessage.success('修改成功，请重新登录')
    localStorage.clear()
    router.push('/login')
  } catch (e) {
    ElMessage.error(e?.response?.data?.message || '修改失败')
  } finally { changePasswordSubmitting.value = false }
}
</script>

<style scoped>
/* Direct Viewport Fill (NO outer margin or outer gradient) */
.layout-wrapper {
  position: relative;
  width: 100vw;
  height: 100vh;
  overflow: hidden;
  background: var(--page-bg) !important;
  padding: 0 !important;
  margin: 0 !important;
  box-sizing: border-box;
}

/* Full Viewport App Window */
.layout-container {
  height: 100vh !important;
  width: 100vw !important;
  margin: 0 !important;
  border-radius: 0 !important;
  background: var(--page-bg) !important;
  box-shadow: none !important;
  overflow: hidden;
  position: relative;
  z-index: 10;
}

/* ── Floating Rounded Neumorphic Sidebar ── */
.aside {
  margin: 12px 0 12px 12px;
  border-radius: 20px !important;
  background: #ffffff !important;
  box-shadow: 4px 6px 18px var(--neu-dark), -4px -4px 14px var(--neu-light) !important;
  border: 1px solid rgba(255, 255, 255, 0.6) !important;
  display: flex;
  flex-direction: column;
  z-index: 100;
  overflow: hidden;
  padding: 16px 0 16px 0;
  transition: width 0.3s cubic-bezier(0.16, 1, 0.3, 1);
}
html[data-theme='dark'] .aside {
  background: #1e2430 !important;
  border: 1px solid rgba(255, 255, 255, 0.05) !important;
}

/* Logo Header */
.logo {
  height: 52px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 16px;
  margin-bottom: 8px;
}
.logo-brand-title {
  font-family: 'Outfit', sans-serif !important;
  font-size: 22px !important;
  font-weight: 800 !important;
  letter-spacing: -0.03em !important;
  color: #0f172a !important;
  text-align: center;
}
html[data-theme='dark'] .logo-brand-title {
  color: #f8fafc !important;
}

/* ── Menu Navigation ── */
.sider-menu {
  flex: 1;
  border-right: none;
  background: transparent;
  --el-menu-bg-color: transparent;
  --el-menu-hover-bg-color: rgba(0, 0, 0, 0.03);
  padding: 4px 0;
  overflow-y: auto;
}

.sider-menu :deep(.el-menu) {
  background-color: transparent !important;
}

.sider-menu :deep(.el-menu--inline) {
  background-color: transparent !important;
  padding: 2px 0 !important;
}

.sider-menu :deep(.el-menu-item) {
  height: 42px !important;
  line-height: 42px !important;
  margin: 4px 12px !important;
  border-radius: 12px !important;
  color: #64748b !important;
  font-size: 13px !important;
  font-weight: 500 !important;
  background: transparent !important;
  box-shadow: none !important;
  transition: all 0.2s cubic-bezier(0.16, 1, 0.3, 1) !important;
}

.sider-menu :deep(.el-menu-item:hover) {
  color: #0f172a !important;
  background: rgba(255, 255, 255, 0.6) !important;
}

/* MANGOS Extruded 3D Pure White Active Pill */
.sider-menu :deep(.el-menu-item.is-active) {
  background: #ffffff !important;
  box-shadow: 4px 6px 18px rgba(166, 180, 200, 0.45), -4px -4px 12px #ffffff !important;
  color: #0f172a !important;
  font-weight: 700 !important;
  border-left: 4px solid #2563eb !important;
  border-radius: 12px !important;
}
html[data-theme='dark'] .sider-menu :deep(.el-menu-item.is-active) {
  background: #1e2430 !important;
  color: #f8fafc !important;
  box-shadow: 4px 6px 18px rgba(0,0,0,0.5), -4px -4px 12px rgba(40,48,64,0.4) !important;
}

.sider-menu :deep(.el-sub-menu__title) {
  height: 42px !important;
  line-height: 42px !important;
  margin: 4px 12px !important;
  border-radius: 12px !important;
  color: #64748b !important;
  font-size: 13px !important;
  font-weight: 500 !important;
  transition: all 0.2s ease !important;
}

.sider-menu :deep(.el-sub-menu__title:hover) {
  color: #0f172a !important;
  background: rgba(255, 255, 255, 0.6) !important;
}

.sider-menu :deep(.el-sub-menu.is-active > .el-sub-menu__title) {
  color: #0f172a !important;
  font-weight: 700 !important;
}

.sider-menu :deep(.el-icon) {
  font-size: 16px !important;
  margin-right: 10px !important;
}

/* Sidebar Footer Primary Action Button */
.sidebar-footer {
  padding: 16px 16px 4px 16px;
  margin-top: auto;
}
.sidebar-create-btn {
  width: 100% !important;
  height: 42px !important;
  border-radius: 20px !important;
  background: linear-gradient(135deg, #2563eb 0%, #1d4ed8 100%) !important;
  color: #ffffff !important;
  font-size: 13px !important;
  font-weight: 700 !important;
  border: none !important;
  box-shadow: 0 6px 18px rgba(37, 99, 235, 0.35), var(--neu-convex-sm) !important;
}

/* ── Right Panel Container ── */
.right-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  overflow: hidden;
}

/* ── Header Toolbar ── */
.header {
  height: 64px;
  padding: 0 24px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: transparent !important;
  border-bottom: none !important;
  box-shadow: none !important;
  flex-shrink: 0;
  z-index: 10;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 14px;
}

.collapse-btn {
  border: none !important;
  background: #ffffff !important;
  box-shadow: var(--neu-convex-sm) !important;
  color: #64748b !important;
  width: 34px !important;
  height: 34px !important;
  border-radius: 10px !important;
  padding: 0 !important;
}
html[data-theme='dark'] .collapse-btn { background: #1e2430 !important; }

.page-title-label {
  font-size: 18px;
  font-weight: 700;
  color: #0f172a;
  font-family: 'Outfit', sans-serif;
}
html[data-theme='dark'] .page-title-label { color: #f8fafc; }

.header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

/* Sunken Search Input Groove */
.search-spotlight-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 7px 14px;
  border-radius: 14px;
  background: #e6eaef;
  box-shadow: var(--neu-concave-sm);
  color: #64748b;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s ease;
}
html[data-theme='dark'] .search-spotlight-btn { background: #141820; }
.search-spotlight-btn:hover {
  color: #0f172a;
}
.spotlight-placeholder {
  font-size: 12px;
}
.spotlight-kbd {
  font-size: 10px;
  font-weight: 700;
  padding: 2px 5px;
  border-radius: 4px;
  background: #ffffff;
  box-shadow: 1px 1px 4px rgba(0,0,0,0.1);
  color: #64748b;
  font-family: monospace;
}

/* Green Status Check-in Badge */
.checkin-badge {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 14px;
  border-radius: 20px;
  background: #22c55e;
  color: #ffffff;
  font-size: 11px;
  font-weight: 700;
  box-shadow: 0 4px 12px rgba(34, 197, 94, 0.35);
}
.checkin-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #ffffff;
}

/* Notification Circle Button */
.notification-btn-box {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: #ffffff;
  box-shadow: var(--neu-convex-sm);
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  cursor: pointer;
  color: #64748b;
  transition: all 0.2s ease;
}
html[data-theme='dark'] .notification-btn-box { background: #1e2430; }
.notification-btn-box:hover { color: #0f172a; transform: translateY(-1px); }
.notification-red-dot {
  position: absolute;
  top: 8px;
  right: 8px;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #ef4444;
  border: 1.5px solid #ffffff;
}

/* User Chip */
.user-chip {
  display: flex;
  align-items: center;
  cursor: pointer;
  border-radius: 50%;
  box-shadow: var(--neu-convex-sm);
}

/* Main View Container */
.main {
  flex: 1;
  padding: 0 !important;
  overflow: auto;
  box-sizing: border-box;
}

/* Transitions */
.app-route-fade-enter-active,
.app-route-fade-leave-active {
  transition: opacity 0.25s ease, transform 0.25s cubic-bezier(0.16, 1, 0.3, 1);
}
.app-route-fade-enter-from { opacity: 0; transform: translateY(6px); }
.app-route-fade-leave-to { opacity: 0; transform: translateY(-4px); }
</style>
