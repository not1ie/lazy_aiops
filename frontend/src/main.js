import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import {
  ArrowDown,
  Close,
  Coin,
  Connection,
  Delete,
  DocumentCopy,
  Download,
  Edit,
  EditPen,
  FirstAidKit,
  Histogram,
  MagicStick,
  Monitor,
  Moon,
  Odometer,
  Operation,
  Platform,
  Plus,
  Refresh,
  Search,
  Setting,
  Share,
  Sunny,
  Tickets,
  Upload,
  User,
  VideoPlay,
  WarningFilled
} from '@element-plus/icons-vue'
import App from './App.vue'
import router from './router'
import './style.css'
import { initTheme } from './utils/theme'

initTheme()

const app = createApp(App)

const iconRegistry = {
  ArrowDown,
  Close,
  Coin,
  Connection,
  Delete,
  DocumentCopy,
  Download,
  Edit,
  EditPen,
  FirstAidKit,
  Histogram,
  MagicStick,
  Monitor,
  Moon,
  Odometer,
  Operation,
  Platform,
  Plus,
  Refresh,
  Search,
  Setting,
  Share,
  Sunny,
  Tickets,
  Upload,
  User,
  VideoPlay,
  WarningFilled
}

for (const [key, component] of Object.entries(iconRegistry)) {
  app.component(key, component)
}

app.use(createPinia())
app.use(router)
app.use(ElementPlus)

// Global error hooks to surface silent render failures
app.config.errorHandler = (err, instance, info) => {
  // eslint-disable-next-line no-console
  console.error('[VueError]', info, err)
}

router.onError((err) => {
  const msg = (err && err.message) || ''
  // Handle failed async chunk loads or transient network issues
  if (msg.includes('Failed to fetch') || msg.includes('Loading chunk') || msg.includes('import')) {
    // eslint-disable-next-line no-console
    console.error('[RouterError]', err)
    window.location.reload()
  }
})

// Expose for quick debugging in production
window.__APP__ = app
window.__ROUTER__ = router

app.mount('#app')
