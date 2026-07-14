import axios from 'axios'
import router from '../router'
import { ElMessage } from 'element-plus'

// Create custom axios instance
const service = axios.create({
  baseURL: '',
  timeout: 30000
})

// Request interceptor
service.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers['Authorization'] = 'Bearer ' + token
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// Response interceptor
service.interceptors.response.use(
  (response) => {
    return response
  },
  (error) => {
    if (error.response && (error.response.status === 401 || error.response.status === 403)) {
      const currentPath = window.location.pathname
      if (currentPath !== '/login') {
        localStorage.clear()
        router.push('/login')
        ElMessage.error('登录状态已失效，请重新登录')
      }
    } else {
      const msg = error.response?.data?.message || error.message || '请求失败'
      ElMessage.error(msg)
    }
    return Promise.reject(error)
  }
)

export default service
