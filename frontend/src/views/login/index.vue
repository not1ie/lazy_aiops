<template>
  <div class="login-container">
    <el-card class="login-card">
      <template #header>
        <div class="login-header">
          <h2>Lazy Auto Ops</h2>
          <p>企业级自动化运维平台</p>
        </div>
      </template>
      <el-form :model="form" :rules="rules" ref="formRef" label-position="top">
        <el-form-item label="用户名" prop="username">
          <el-input v-model="form.username" prefix-icon="User" placeholder="admin" />
        </el-form-item>
        <el-form-item label="密码" prop="password">
          <el-input v-model="form.password" prefix-icon="Lock" type="password" placeholder="admin123" show-password @keyup.enter="handleLogin" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" class="w-100" @click="handleLogin">登录</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'
import { ElMessage } from 'element-plus'

const router = useRouter()
const formRef = ref(null)
const loading = ref(false)

const form = reactive({
  username: '',
  password: ''
})

const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }]
}

const handleLogin = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (valid) {
      loading.value = true
      try {
        const res = await axios.post('/api/v1/login', form)
        if (res.data.code === 0) {
          localStorage.setItem('token', res.data.data.token)
          const userInfo = res.data.data.user_info || null
          if (userInfo) {
            localStorage.setItem('user_info', JSON.stringify(userInfo))
            localStorage.setItem('role_code', userInfo.role?.code || '')
            const perms = userInfo.role?.permissions?.map((p) => p.code) || []
            localStorage.setItem('permissions', JSON.stringify(perms))
          }
          ElMessage.success('登录成功')
          router.push('/')
        } else {
          ElMessage.error(res.data.message || '登录失败')
        }
      } catch (e) {
        ElMessage.error('网络错误')
      } finally {
        loading.value = false
      }
    }
  })
}
</script>

<style scoped>
.login-container { display: flex; justify-content: center; align-items: center; height: 100vh; background: #2d3a4b; }
.login-card { width: 400px; }
.login-header { text-align: center; }
.login-header h2 { margin: 0; color: #409eff; }
.login-header p { margin: 10px 0 0; color: #909399; font-size: 14px; }
.w-100 { width: 100%; }
</style>
