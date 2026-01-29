import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '../stores/user'
import MainLayout from '../layouts/MainLayout.vue'

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('../views/Login.vue'),
    meta: { requiresAuth: false }
  },
  {
    path: '/',
    component: MainLayout,
    redirect: '/dashboard',
    meta: { requiresAuth: true },
    children: [
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('../views/Dashboard.vue')
      },
      {
        path: 'cmdb',
        name: 'CMDB',
        component: () => import('../views/CMDB.vue')
      },
      {
        path: 'monitor',
        name: 'Monitor',
        component: () => import('../views/Monitor.vue')
      },
      {
        path: 'alert',
        name: 'Alert',
        component: () => import('../views/Alert.vue')
      },
      {
        path: 'task',
        name: 'Task',
        component: () => import('../views/Task.vue')
      },
      {
        path: 'ai/log-analysis',
        name: 'AILogAnalysis',
        component: () => import('../views/AI/LogAnalysis.vue')
      },
      {
        path: 'sqlaudit',
        name: 'SQLAudit',
        component: () => import('../views/SQLAudit.vue')
      },
      {
        path: 'k8s',
        name: 'K8s',
        component: () => import('../views/K8s.vue')
      },
      {
        path: 'ansible',
        name: 'Ansible',
        component: () => import('../views/Ansible.vue')
      },
      {
        path: 'executor',
        name: 'Executor',
        component: () => import('../views/Executor.vue')
      }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

// 路由守卫
router.beforeEach((to, from, next) => {
  const userStore = useUserStore()
  
  if (to.meta.requiresAuth && !userStore.isLoggedIn) {
    next('/login')
  } else if (to.path === '/login' && userStore.isLoggedIn) {
    next('/dashboard')
  } else {
    next()
  }
})

export default router
