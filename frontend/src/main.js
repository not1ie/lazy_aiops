import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import App from './App.vue'
import router from './router'
import './style.css'
import { initTheme } from './utils/theme'

initTheme()

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
}

router.onError((err) => {
  const msg = (err && err.message) || ''
  if (msg.includes('Failed to fetch') || msg.includes('Loading chunk')) {
    window.location.reload()
  }
})

app.mount('#app')
