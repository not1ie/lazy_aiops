<template>
  <div class="main-layout">
    <!-- 侧边栏 -->
    <aside class="sidebar glass">
      <div class="sidebar-header">
        <h2>Lazy Auto Ops</h2>
      </div>
      
      <nav class="sidebar-nav">
        <div v-for="group in menuGroups" :key="group.title" class="nav-group">
          <div class="nav-group-title">{{ group.title }}</div>
          <router-link
            v-for="item in group.items"
            :key="item.path"
            :to="item.path"
            class="nav-item"
            active-class="nav-item-active"
          >
            <i :class="item.icon"></i>
            <span>{{ item.label }}</span>
          </router-link>
        </div>
      </nav>
      
      <div class="sidebar-footer">
        <div class="user-info">
          <i class="fas fa-user-circle"></i>
          <span>{{ userStore.userInfo?.username || 'Admin' }}</span>
        </div>
        <button @click="handleLogout" class="logout-btn">
          <i class="fas fa-sign-out-alt"></i>
        </button>
      </div>
    </aside>
    
    <!-- 主内容区 -->
    <main class="main-content">
      <router-view />
    </main>
  </div>
</template>

<script setup>
import { useRouter } from 'vue-router'
import { useUserStore } from '../stores/user'

const router = useRouter()
const userStore = useUserStore()

const menuGroups = [
  {
    title: '概览',
    items: [
      { path: '/dashboard', label: '仪表板', icon: 'fas fa-chart-line' }
    ]
  },
  {
    title: '资源管理',
    items: [
      { path: '/cmdb', label: 'CMDB', icon: 'fas fa-database' },
      { path: '/monitor', label: '监控中心', icon: 'fas fa-desktop' },
      { path: '/alert', label: '告警管理', icon: 'fas fa-bell' }
    ]
  },
  {
    title: 'AI智能',
    items: [
      { path: '/ai/log-analysis', label: '日志分析', icon: 'fas fa-robot' }
    ]
  },
  {
    title: '自动化',
    items: [
      { path: '/task', label: '任务调度', icon: 'fas fa-tasks' },
      { path: '/executor', label: '批量执行', icon: 'fas fa-terminal' },
      { path: '/ansible', label: 'Ansible', icon: 'fas fa-cogs' },
      { path: '/k8s', label: 'Kubernetes', icon: 'fas fa-dharmachakra' },
      { path: '/sqlaudit', label: 'SQL审计', icon: 'fas fa-file-code' }
    ]
  }
]

const handleLogout = () => {
  userStore.logout()
  router.push('/login')
}
</script>

<style scoped>
.main-layout {
  display: flex;
  min-height: 100vh;
  background: var(--apple-bg-primary);
}

.sidebar {
  width: 260px;
  background: var(--apple-bg-secondary);
  border-right: 1px solid rgba(255, 255, 255, 0.05);
  display: flex;
  flex-direction: column;
  position: fixed;
  left: 0;
  top: 0;
  bottom: 0;
  z-index: 100;
}

.sidebar-header {
  padding: var(--space-xl) var(--space-lg);
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}

.sidebar-header h2 {
  font-size: 20px;
  font-weight: 700;
  background: linear-gradient(135deg, var(--apple-accent), var(--apple-success));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

.sidebar-nav {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-lg) 0;
}

.nav-group {
  margin-bottom: var(--space-xl);
}

.nav-group-title {
  padding: 0 var(--space-lg);
  margin-bottom: var(--space-sm);
  font-size: 12px;
  font-weight: 600;
  color: var(--apple-text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: var(--space-md);
  padding: var(--space-md) var(--space-lg);
  color: var(--apple-text-secondary);
  text-decoration: none;
  transition: all 0.2s var(--ease-standard);
  position: relative;
}

.nav-item::before {
  content: '';
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 3px;
  background: var(--apple-accent);
  transform: scaleY(0);
  transition: transform 0.2s var(--ease-standard);
}

.nav-item:hover {
  background: rgba(255, 255, 255, 0.05);
  color: var(--apple-text-primary);
}

.nav-item-active {
  background: rgba(10, 132, 255, 0.1);
  color: var(--apple-accent);
}

.nav-item-active::before {
  transform: scaleY(1);
}

.nav-item i {
  width: 20px;
  text-align: center;
}

.sidebar-footer {
  padding: var(--space-lg);
  border-top: 1px solid rgba(255, 255, 255, 0.05);
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.user-info {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
  color: var(--apple-text-secondary);
  font-size: 14px;
}

.user-info i {
  font-size: 24px;
}

.logout-btn {
  background: transparent;
  border: none;
  color: var(--apple-text-secondary);
  cursor: pointer;
  padding: var(--space-sm);
  border-radius: var(--radius-sm);
  transition: all 0.2s var(--ease-standard);
}

.logout-btn:hover {
  background: rgba(255, 69, 58, 0.1);
  color: var(--apple-danger);
}

.main-content {
  flex: 1;
  margin-left: 260px;
  min-height: 100vh;
}

@media (max-width: 768px) {
  .sidebar {
    transform: translateX(-100%);
  }
  
  .main-content {
    margin-left: 0;
  }
}
</style>
