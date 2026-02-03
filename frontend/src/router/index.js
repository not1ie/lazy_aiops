import { createRouter, createWebHistory } from 'vue-router'
import Layout from '@/layout/index.vue'

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/login/index.vue')
  },
  {
    path: '/',
    component: Layout,
    redirect: '/dashboard',
    children: [
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('@/views/dashboard/index.vue'),
        meta: { title: '仪表盘', icon: 'Odometer' }
      },
      {
        path: 'host',
        name: 'Host',
        component: () => import('@/views/cmdb/host.vue'),
        meta: { title: '主机管理', icon: 'Monitor' }
      },
      {
        path: 'docker',
        name: 'Docker',
        component: () => import('@/views/docker/index.vue'),
        meta: { title: 'Docker管理', icon: 'Platform' }
      },
      {
        path: 'application',
        name: 'Application',
        component: () => import('@/views/application/index.vue'),
        meta: { title: '应用中心', icon: 'Box' }
      },
      {
        path: 'system/dept',
        name: 'Department',
        component: () => import('@/views/system/dept.vue'),
        meta: { title: '部门管理', icon: 'OfficeBuilding' }
      }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to, from, next) => {
  const token = localStorage.getItem('token')
  if (to.name !== 'Login' && !token) next({ name: 'Login' })
  else next()
})

export default router
