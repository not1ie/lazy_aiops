<template>
  <div class="layout-wrapper">
    <!-- Ambient liquid glass blobs -->
    <div class="ambient-blob blob-1"></div>
    <div class="ambient-blob blob-2"></div>
    <div class="ambient-blob blob-3"></div>

    <el-container class="layout-container" v-loading="loading">
      <el-aside :width="isCollapsed ? '64px' : '220px'" class="aside">
        <div class="logo" :style="{ justifyContent: isCollapsed ? 'center' : 'flex-start', padding: isCollapsed ? '0' : '0 20px' }">
          <div class="logo-text">
            <div class="logo-title" v-show="!isCollapsed">LazyOps</div>
            <div class="logo-title" v-show="isCollapsed" style="font-size: 14px; font-weight: 800; letter-spacing: 0;">LO</div>
          </div>
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
      </el-aside>

      <el-container style="flex-direction: column">
        <el-header class="header">
          <div class="header-left">
            <el-button class="icon-btn" :icon="isCollapsed ? 'Expand' : 'Fold'" @click="isCollapsed = !isCollapsed" style="margin-right: 16px;" />
            <el-breadcrumb separator="/" class="breadcrumb-nav">
              <el-breadcrumb-item :to="{ path: '/' }">首页</el-breadcrumb-item>
              <el-breadcrumb-item v-for="(item, idx) in breadcrumbs" :key="idx" :to="item.path ? { path: item.path } : null">
                {{ item.title }}
              </el-breadcrumb-item>
            </el-breadcrumb>
          </div>
          <div class="header-right">
            <el-button class="icon-btn" icon="Search" @click="globalSearchVisible = true" />
            <el-dropdown trigger="click" @command="handleUserCommand">
              <div class="user-chip">
                <el-avatar :size="28" src="https://images.unsplash.com/photo-1534528741775-53994a69daeb?q=80&w=256&auto=format&fit=crop" />
                <span class="user-name">{{ username }}</span>
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
            <component :is="Component" :key="route.fullPath" />
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
    </el-container>
  </div>
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

const isCollapsed = ref(false)
const globalSearchVisible = ref(false)
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

const menuData = [
  { index: '/dashboard', title: '仪表盘', icon: 'Menu' },
  { 
    index: 'asset', title: '资产管理', icon: 'Coin',
    children: [
      { index: '/host', title: '主机管理' },
      { index: '/cmdb/credential', title: '凭据管理' },
      { index: '/cmdb/database', title: '数据库资产' },
      { index: '/cmdb/cloud', title: '云资源' },
      { index: '/cmdb/network-devices', title: '网络与防火墙' },
      { index: '/cmdb/topology', title: '服务拓扑' },
      { index: '/jump/assets', title: '堡垒机接入' }
    ]
  },
  {
    index: 'k8s', title: '集群管理', icon: 'Platform',
    children: [
      { index: '/k8s/clusters', title: 'K8s集群' },
      { index: '/docker', title: 'Docker容器' },
      { index: '/k8s/workloads', title: '工作负载' },
      { index: '/k8s/pods', title: 'Pods' },
      { index: '/k8s/services', title: '服务发现' }
    ]
  },
  {
    index: 'delivery', title: '应用发布', icon: 'Position',
    children: [
      { index: '/cicd/pipelines', title: '流水线' },
      { index: '/cicd/releases', title: '发布管理' }
    ]
  },
  {
    index: 'monitor', title: '监控中心', icon: 'Histogram',
    children: [
      { index: '/monitor/hosts', title: '主机监控' },
      { index: '/monitor/metrics', title: '数据源配置' },
      { index: '/alert/events', title: '告警事件' },
      { index: '/alert/history', title: '告警复盘' },
      { index: '/remediation/logs', title: '故障自愈' },
      { index: '/oncall/schedules', title: '值班排班' },
      { index: '/notify/channels', title: '通知渠道' },
      { index: '/domain/ssl', title: '域名证书' }
    ]
  },
  {
    index: 'log', title: '日志中心', icon: 'Document',
    children: [
      { index: '/log/query', title: '日志查询', icon: 'Search' },
      { index: '/log/library', title: '日志库', icon: 'Collection' },
      { index: '/log/alerts', title: '日志告警', icon: 'Bell' },
      { index: '/log/permissions', title: '日志权限', icon: 'Lock' }
    ]
  },
  {
    index: 'task', title: '作业任务', icon: 'Cpu',
    children: [
      { index: '/executor', title: '批量执行' },
      { index: '/task/schedules', title: '任务调度' },
      { index: '/workorder/orders', title: '运维工单' },
      { index: '/workflow/workflows', title: '运维编排' },
      { index: '/ansible/playbooks', title: 'Ansible管理' },
      { index: '/sqlaudit/requests', title: 'SQL审计' }
    ]
  },
  {
    index: 'ai', title: 'AIOps智能', icon: 'MagicStick',
    children: [
      { index: '/ai/ops', title: '故障诊断' },
      { index: '/knowledge/docs', title: 'AI知识库' }
    ]
  },
  {
    index: 'system', title: '系统管理', icon: 'Setting',
    children: [
      { index: '/system/users', title: '用户管理' },
      { index: '/system/roles', title: '角色管理' },
      { index: '/system/menus', title: '权限管理' },
      { index: '/cost/overview', title: '成本分析' }
    ]
  }
]

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
  const userInfo = JSON.parse(localStorage.getItem('user_info') || '{}')
  if (userInfo.nickname) username.value = userInfo.nickname
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
.layout-wrapper {
  position: relative;
  width: 100vw;
  height: 100vh;
  overflow: hidden;
  background: var(--page-bg);
  background-image: var(--bg-gradient);
}

.ambient-blob {
  position: absolute;
  border-radius: 50%;
  filter: blur(140px);
  opacity: 0.35;
  z-index: 0;
  pointer-events: none;
  animation: drift 25s infinite alternate ease-in-out;
}
html[data-theme='dark'] .ambient-blob {
  opacity: 0.12;
}
.blob-1 {
  width: 500px;
  height: 500px;
  background: radial-gradient(circle, #ff9500 0%, rgba(255,149,0,0) 70%);
  top: -100px;
  left: 20%;
  animation-duration: 28s;
}
.blob-2 {
  width: 600px;
  height: 600px;
  background: radial-gradient(circle, #0071e3 0%, rgba(0,113,227,0) 70%);
  bottom: -150px;
  right: 15%;
  animation-duration: 35s;
}
.blob-3 {
  width: 450px;
  height: 450px;
  background: radial-gradient(circle, #34c759 0%, rgba(52,199,89,0) 70%);
  top: 30%;
  right: -100px;
  animation-duration: 22s;
}

@keyframes drift {
  0% {
    transform: translate(0, 0) scale(1) rotate(0deg);
  }
  50% {
    transform: translate(80px, 60px) scale(1.15) rotate(180deg);
  }
  100% {
    transform: translate(-40px, -80px) scale(0.9) rotate(360deg);
  }
}

.layout-container {
  height: 100vh;
  width: 100vw;
  background-color: transparent;
  position: relative;
  z-index: 1;
}

.aside {
  background: var(--glass-bg);
  backdrop-filter: var(--glass-blur);
  transition: width 0.3s cubic-bezier(0.2, 0, 0, 1);
}

.breadcrumb-nav {
  font-size: 13px;
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
}

.breadcrumb-nav :deep(.el-breadcrumb__inner) {
  font-weight: 500;
  color: var(--el-text-color-regular) !important;
}

.breadcrumb-nav :deep(.el-breadcrumb__inner.is-link:hover) {
  color: var(--el-color-primary) !important;
}

.aside {
  background: var(--glass-bg);
  backdrop-filter: var(--glass-blur);
  -webkit-backdrop-filter: var(--glass-blur);
  border-right: 1px solid var(--glass-border);
  display: flex;
  flex-direction: column;
  z-index: 10;
  border-top-right-radius: 24px;
  border-bottom-right-radius: 24px;
  box-shadow: 4px 0 24px rgba(0, 0, 0, 0.04);
  overflow: hidden;
  transition: width 0.3s cubic-bezier(0.2, 0, 0, 1);
}

.logo {
  height: 60px;
  display: flex;
  align-items: center;
  padding: 0 20px;
  gap: 12px;
  border-bottom: 1px solid var(--glass-outline);
  transition: padding 0.3s, justify-content 0.3s;
}



.logo-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.sider-menu {
  flex: 1;
  border-right: none;
  background: transparent;
  --el-menu-bg-color: transparent;
  --el-menu-hover-bg-color: rgba(0, 0, 0, 0.04);
}

.sider-menu :deep(.el-menu-item) {
  height: 40px;
  line-height: 40px;
  margin: 4px 12px;
  border-radius: 6px;
}

.sider-menu :deep(.el-menu-item.is-active) {
  background-color: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
  font-weight: 500;
}

.sider-menu :deep(.el-sub-menu__title) {
  height: 44px;
  line-height: 44px;
}

.header {
  height: 60px;
  background: var(--glass-bg-light);
  backdrop-filter: var(--glass-blur-light);
  -webkit-backdrop-filter: var(--glass-blur-light);
  border-bottom: 1px solid var(--glass-border);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  z-index: 10;
}

.header-left {
  display: flex;
  align-items: center;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.icon-btn {
  border: none;
  background: transparent;
  color: var(--el-text-color-secondary);
}

.icon-btn:hover {
  background-color: rgba(0, 0, 0, 0.05);
  color: var(--el-text-color-primary);
}

.user-chip {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 6px;
  transition: background-color 0.2s;
}

.user-chip:hover {
  background-color: rgba(0, 0, 0, 0.05);
}

.user-name {
  font-size: 14px;
  color: var(--el-text-color-primary);
}

.main {
  padding: 24px;
  overflow: auto;
}

.app-route-fade-enter-active,
.app-route-fade-leave-active {
  transition: opacity 0.2s, transform 0.2s;
}
.app-route-fade-enter-from { opacity: 0; transform: translateY(5px); }
.app-route-fade-leave-to { opacity: 0; transform: translateY(-5px); }
</style>
