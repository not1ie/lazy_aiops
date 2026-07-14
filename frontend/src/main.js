import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus, { ElMessage } from 'element-plus'
import 'element-plus/dist/index.css'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import axios from 'axios'
import App from './App.vue'
import router from './router'
import './style.css'
import { initTheme } from './utils/theme'

initTheme()

// Global Axios Interceptors
axios.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers['Authorization'] = 'Bearer ' + token
    }
    return config
  },
  (error) => Promise.reject(error)
)

axios.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response && (error.response.status === 401 || error.response.status === 403)) {
      const currentPath = window.location.pathname
      if (currentPath !== '/login') {
        localStorage.clear()
        router.push('/login')
        ElMessage.error('登录状态已失效，请重新登录')
      }
    }
    return Promise.reject(error)
  }
)

const app = createApp(App)

// Register ALL icons to avoid missing icons in high-density hubs
for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component)
}

app.use(createPinia())
app.use(router)
app.use(ElementPlus)

app.config.errorHandler = (err, instance, info) => {
  console.error('[VueError]', info, err)
  fetch('/api/v1/debug/error', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      message: `[VueError] info: ${info}, err: ${err?.message || err}`,
      stack: err?.stack || '',
      url: window.location.href
    })
  })
}

window.addEventListener('error', (event) => {
  fetch('/api/v1/debug/error', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      message: event.message,
      stack: event.error?.stack || '',
      url: window.location.href
    })
  })
})

router.onError((err) => {
  console.error('[RouterError]', err)
  fetch('/api/v1/debug/error', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      message: `[RouterError] ${err?.message || err}`,
      stack: err?.stack || '',
      url: window.location.href
    })
  })
  const msg = (err && err.message) || ''
  if (msg.includes('Failed to fetch') || msg.includes('Loading chunk')) {
    window.location.reload()
  }
})

app.mount('#app')
