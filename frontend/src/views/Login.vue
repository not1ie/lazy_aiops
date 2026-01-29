<template>
  <div class="login-page">
    <div class="login-bg"></div>
    <div class="login-container">
      <div class="login-card glass">
        <div class="login-header">
          <h1>Lazy Auto Ops</h1>
          <p>AI驱动的轻量级运维平台</p>
        </div>
        
        <form @submit.prevent="handleLogin" class="login-form">
          <AppleInput
            v-model="username"
            label="用户名"
            placeholder="请输入用户名"
            icon="fas fa-user"
          />
          
          <AppleInput
            v-model="password"
            type="password"
            label="密码"
            placeholder="请输入密码"
            icon="fas fa-lock"
          />
          
          <div class="login-options">
            <label class="remember-me">
              <input type="checkbox" v-model="rememberMe" />
              <span>记住我</span>
            </label>
          </div>
          
          <AppleButton
            type="primary"
            :loading="loading"
            @click="handleLogin"
            style="width: 100%"
          >
            <i class="fas fa-sign-in-alt"></i>
            登录
          </AppleButton>
        </form>
        
        <div v-if="error" class="login-error">
          <i class="fas fa-exclamation-circle"></i>
          {{ error }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '../stores/user'
import AppleButton from '../components/AppleButton.vue'
import AppleInput from '../components/AppleInput.vue'

const router = useRouter()
const userStore = useUserStore()

const username = ref('admin')
const password = ref('')
const rememberMe = ref(false)
const loading = ref(false)
const error = ref('')

const handleLogin = async () => {
  if (!username.value || !password.value) {
    error.value = '请输入用户名和密码'
    return
  }
  
  loading.value = true
  error.value = ''
  
  try {
    const success = await userStore.login(username.value, password.value)
    if (success) {
      router.push('/dashboard')
    } else {
      error.value = '用户名或密码错误'
    }
  } catch (err) {
    error.value = '登录失败，请稍后重试'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
}

.login-bg {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  opacity: 0.8;
}

.login-container {
  position: relative;
  z-index: 1;
  width: 100%;
  max-width: 420px;
  padding: var(--space-lg);
}

.login-card {
  padding: var(--space-2xl);
  border-radius: var(--radius-xl);
  border: 1px solid rgba(255, 255, 255, 0.2);
  animation: scaleIn 0.5s var(--ease-spring);
}

.login-header {
  text-align: center;
  margin-bottom: var(--space-2xl);
}

.login-header h1 {
  font-size: 32px;
  font-weight: 700;
  margin-bottom: var(--space-sm);
  background: linear-gradient(135deg, var(--apple-accent), var(--apple-success));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

.login-header p {
  color: var(--apple-text-secondary);
  font-size: 14px;
}

.login-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-lg);
}

.login-options {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.remember-me {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
  color: var(--apple-text-secondary);
  font-size: 14px;
  cursor: pointer;
}

.remember-me input[type="checkbox"] {
  width: 18px;
  height: 18px;
  cursor: pointer;
}

.login-error {
  margin-top: var(--space-lg);
  padding: var(--space-md);
  background: rgba(255, 69, 58, 0.1);
  border: 1px solid var(--apple-danger);
  border-radius: var(--radius-md);
  color: var(--apple-danger);
  font-size: 14px;
  display: flex;
  align-items: center;
  gap: var(--space-sm);
}

@media (max-width: 768px) {
  .login-card {
    padding: var(--space-lg);
  }
}
</style>
