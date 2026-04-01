<template>
  <div class="login-container">
    <div class="backdrop-glow glow-a"></div>
    <div class="backdrop-glow glow-b"></div>
    <div class="backdrop-glow glow-c"></div>
    <div class="backdrop-glow glow-d"></div>
    <div class="backdrop-grid"></div>
    <div class="backdrop-vignette"></div>

    <div class="particle-layer">
      <span
        v-for="p in particles"
        :key="p.id"
        class="particle-dot"
        :style="{
          left: `${p.left}%`,
          top: `${p.top}%`,
          width: `${p.size}px`,
          height: `${p.size}px`,
          animationDelay: `${p.delay}s`,
          animationDuration: `${p.duration}s`
        }"
      />
    </div>

    <div class="login-topbar">
      <button class="theme-switch" type="button" @click="toggleTheme">
        <el-icon><component :is="isDark ? 'Sunny' : 'Moon'" /></el-icon>
        <span>{{ isDark ? '浅色模式' : '深色模式' }}</span>
      </button>
    </div>

    <div class="login-shell">
      <el-card class="login-card">
        <div class="login-header">
          <div class="brand-eyebrow">运维统一入口</div>
          <div class="brand-name">Lazy Auto Ops</div>
          <p>统一接入资产、堡垒机、自动化与智能诊断能力</p>
        </div>

        <el-form ref="formRef" :model="form" :rules="rules" class="login-form" @keyup.enter="handleLogin">
          <el-form-item prop="username">
            <el-input
              ref="usernameRef"
              v-model="form.username"
              class="auth-input"
              placeholder="用户名"
              @input="loginError = ''"
            />
          </el-form-item>

          <el-form-item prop="password" class="password-item">
            <el-input
              v-model="form.password"
              class="auth-input"
              :type="showPassword ? 'text' : 'password'"
              placeholder="密码"
              @input="loginError = ''"
              @keydown="handlePasswordKeyState"
              @keyup="handlePasswordKeyState"
            >
              <template #suffix>
                <button class="password-visibility" type="button" @click="showPassword = !showPassword">
                  {{ showPassword ? '隐藏' : '查看' }}
                </button>
              </template>
            </el-input>
            <div v-if="capsLockOn" class="caps-indicator">
              <el-icon><WarningFilled /></el-icon>
              <span>Caps Lock 已开启</span>
            </div>
          </el-form-item>

          <el-alert v-if="loginError" :title="loginError" type="error" :closable="false" class="login-error" />

          <el-form-item>
            <el-button
              type="primary"
              :loading="loading"
              :disabled="loading"
              class="login-submit"
              @click="handleLogin"
            >
              {{ loading ? '登录中...' : '登录' }}
            </el-button>
          </el-form-item>
        </el-form>
      </el-card>
    </div>

    <el-dialog
      v-model="changePwdVisible"
      title="首次登录请修改默认密码"
      width="460px"
      :close-on-click-modal="false"
      :close-on-press-escape="false"
      :show-close="false"
    >
      <el-alert type="warning" show-icon :closable="false" title="检测到当前账号仍在使用默认密码，需先修改后再进入系统。" />
      <el-form ref="changePwdRef" :model="changePwdForm" :rules="changePwdRules" label-position="top" style="margin-top: 12px;">
        <el-form-item label="新密码" prop="new_password">
          <el-input v-model="changePwdForm.new_password" type="password" show-password placeholder="请输入新密码（至少8位）" />
        </el-form-item>
        <el-form-item label="确认新密码" prop="confirm_password">
          <el-input v-model="changePwdForm.confirm_password" type="password" show-password placeholder="请再次输入新密码" @keyup.enter="submitChangePassword" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="cancelForceChange">退出登录</el-button>
        <el-button type="primary" :loading="changePwdLoading" @click="submitChangePassword">确认修改</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useTheme } from '@/utils/theme'

const router = useRouter()
const { isDark, toggleTheme } = useTheme()
const formRef = ref(null)
const usernameRef = ref(null)
const loading = ref(false)
const showPassword = ref(false)
const capsLockOn = ref(false)
const changePwdVisible = ref(false)
const changePwdRef = ref(null)
const changePwdLoading = ref(false)
const pendingToken = ref('')
const pendingUserID = ref('')
const loginPassword = ref('')
const loginError = ref('')
const particles = ref([])

const form = reactive({
  username: '',
  password: ''
})

const changePwdForm = reactive({
  new_password: '',
  confirm_password: ''
})

const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }]
}

const changePwdRules = {
  new_password: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 8, message: '新密码至少 8 位', trigger: 'blur' }
  ],
  confirm_password: [
    { required: true, message: '请再次输入新密码', trigger: 'blur' },
    {
      validator: (rule, value, callback) => {
        void rule
        if (value !== changePwdForm.new_password) {
          callback(new Error('两次输入的新密码不一致'))
          return
        }
        callback()
      },
      trigger: 'blur'
    }
  ]
}

const createParticles = () => Array.from({ length: 18 }).map((_, index) => ({
  id: index + 1,
  left: Math.random() * 100,
  top: Math.random() * 100,
  size: 1 + Math.random() * 2,
  delay: Math.random() * 6,
  duration: 16 + Math.random() * 24
}))

const setLoginSession = (payload) => {
  localStorage.setItem('token', payload.token)
  const userInfo = payload.user_info || null
  if (userInfo) {
    localStorage.setItem('user_info', JSON.stringify(userInfo))
    localStorage.setItem('role_code', userInfo.role?.code || '')
    const perms = userInfo.role?.permissions?.map((p) => p.code) || []
    localStorage.setItem('permissions', JSON.stringify(perms))
  }
}

const clearLoginSession = () => {
  localStorage.removeItem('token')
  localStorage.removeItem('permissions')
  localStorage.removeItem('user_info')
  localStorage.removeItem('role_code')
}

const cancelForceChange = () => {
  clearLoginSession()
  changePwdVisible.value = false
  changePwdForm.new_password = ''
  changePwdForm.confirm_password = ''
  pendingToken.value = ''
  pendingUserID.value = ''
  loginPassword.value = ''
}

const submitChangePassword = async () => {
  if (!changePwdRef.value || !pendingToken.value || !pendingUserID.value) return
  await changePwdRef.value.validate(async (valid) => {
    if (!valid) return
    changePwdLoading.value = true
    try {
      const res = await axios.put(
        `/api/v1/rbac/users/${pendingUserID.value}/password`,
        {
          old_password: loginPassword.value,
          new_password: changePwdForm.new_password
        },
        {
          headers: { Authorization: `Bearer ${pendingToken.value}` }
        }
      )
      if (res.data.code !== 0) {
        ElMessage.error(res.data.message || '修改密码失败')
        return
      }
      ElMessage.success('密码修改成功，请重新登录')
      cancelForceChange()
    } catch (e) {
      ElMessage.error(e?.response?.data?.message || '修改密码失败')
    } finally {
      changePwdLoading.value = false
    }
  })
}

const parseLoginError = (rawMessage, statusCode = 0, errorCode = '') => {
  const message = String(rawMessage || '').trim()
  const lower = message.toLowerCase()

  if (errorCode === 'ECONNABORTED') return '请求超时，请稍后重试'
  if (!statusCode) return '网络错误：无法连接到服务端'
  if (statusCode >= 500) return '服务端异常，请稍后重试'
  if (lower.includes('用户不存在')) return '账号不存在，请检查用户名'
  if (lower.includes('密码错误')) return '密码错误，请重新输入'
  if (lower.includes('用户已被禁用')) return '账号已被禁用，请联系管理员'
  if (lower.includes('token')) return '登录态无效，请重新登录'
  if (lower.includes('参数错误')) return '登录参数错误，请检查输入'
  return message || '登录失败，请稍后重试'
}

const handlePasswordKeyState = (event) => {
  capsLockOn.value = Boolean(event?.getModifierState?.('CapsLock'))
}

const handleLogin = async () => {
  if (!formRef.value || loading.value) return
  loginError.value = ''
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    loading.value = true
    try {
      const res = await axios.post('/api/v1/login', form)
      if (res.data.code === 0) {
        loginError.value = ''
        const payload = res.data.data || {}
        setLoginSession(payload)
        ElMessage.success('登录成功')
        if (payload.must_change_password) {
          pendingToken.value = payload.token || ''
          pendingUserID.value = payload.user_info?.id || ''
          loginPassword.value = form.password
          changePwdVisible.value = true
          return
        }
        if (payload.recommend_change_password) {
          await ElMessageBox.alert('当前使用的是默认管理员密码，建议立即修改密码以提升安全性。', '安全提示', {
            confirmButtonText: '我知道了',
            type: 'warning'
          })
        }
        router.push('/')
        return
      }
      loginError.value = parseLoginError(res.data.message, 401)
      ElMessage.error(loginError.value)
    } catch (e) {
      const statusCode = Number(e?.response?.status || 0)
      const msg = e?.response?.data?.message || e?.message || ''
      loginError.value = parseLoginError(msg, statusCode, e?.code)
      ElMessage.error(loginError.value)
    } finally {
      loading.value = false
    }
  })
}


onMounted(() => {
  particles.value = createParticles()
  usernameRef.value?.focus?.()
})
</script>

<style scoped>
.login-container {
  position: relative;
  min-height: 100vh;
  display: flex;
  justify-content: center;
  align-items: center;
  overflow: hidden;
  padding: 32px 20px;
  font-family: "SF Pro Text", "PingFang SC", "Segoe UI", "Helvetica Neue", sans-serif;
  background:
    radial-gradient(circle at 14% 16%, rgba(116, 171, 255, 0.26), transparent 36%),
    radial-gradient(circle at 86% 14%, rgba(255, 211, 163, 0.24), transparent 32%),
    radial-gradient(circle at 50% 112%, rgba(144, 191, 255, 0.22), transparent 40%),
    linear-gradient(180deg, #f8fafd 0%, #edf2f8 52%, #e9eff7 100%);
}

:global(html[data-theme='dark'] .login-container) {
  background:
    radial-gradient(circle at 16% 18%, rgba(57, 112, 214, 0.33), transparent 34%),
    radial-gradient(circle at 84% 14%, rgba(76, 122, 188, 0.2), transparent 28%),
    radial-gradient(circle at 48% 112%, rgba(29, 74, 135, 0.28), transparent 42%),
    linear-gradient(180deg, #070d19 0%, #0b1424 54%, #101a2a 100%);
}

.backdrop-glow {
  position: absolute;
  border-radius: 999px;
  filter: blur(92px);
  opacity: 0.62;
  pointer-events: none;
}

.glow-a { width: 440px; height: 440px; left: -70px; top: -40px; background: rgba(120, 176, 255, 0.42); }
.glow-b { width: 340px; height: 340px; right: 8%; top: 8%; background: rgba(255, 206, 158, 0.32); }
.glow-c { width: 300px; height: 300px; left: 14%; bottom: -80px; background: rgba(162, 199, 255, 0.28); }
.glow-d { width: 400px; height: 400px; right: 14%; bottom: -130px; background: rgba(123, 168, 245, 0.24); }

.backdrop-grid {
  position: absolute;
  inset: 0;
  pointer-events: none;
  opacity: 0.52;
  background-image:
    linear-gradient(to right, rgba(51, 87, 132, 0.08) 1px, transparent 1px),
    linear-gradient(to bottom, rgba(51, 87, 132, 0.07) 1px, transparent 1px);
  background-size: 56px 56px;
  mask-image: radial-gradient(circle at 50% 42%, black 28%, transparent 86%);
}

.backdrop-vignette {
  position: absolute;
  inset: 0;
  pointer-events: none;
  background: radial-gradient(circle at 50% 44%, transparent 52%, rgba(16, 31, 55, 0.12) 100%);
}

:global(html[data-theme='dark'] .backdrop-grid) {
  opacity: 0.36;
  background-image:
    linear-gradient(to right, rgba(152, 179, 214, 0.09) 1px, transparent 1px),
    linear-gradient(to bottom, rgba(152, 179, 214, 0.08) 1px, transparent 1px);
}

:global(html[data-theme='dark'] .backdrop-vignette) {
  background: radial-gradient(circle at 50% 44%, transparent 44%, rgba(2, 6, 23, 0.5) 100%);
}

.particle-layer {
  position: absolute;
  inset: 0;
  pointer-events: none;
}

.particle-dot {
  position: absolute;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.28);
  box-shadow: 0 0 8px rgba(255, 255, 255, 0.34);
  animation: drift linear infinite;
}

:global(html[data-theme='dark'] .particle-dot) {
  background: rgba(148, 189, 255, 0.26);
  box-shadow: 0 0 10px rgba(83, 138, 228, 0.34);
}

.login-topbar {
  position: absolute;
  inset: 24px 24px auto auto;
  z-index: 4;
}

.theme-switch {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  border: 1px solid rgba(255, 255, 255, 0.42);
  border-radius: 999px;
  padding: 10px 14px;
  background: rgba(255, 255, 255, 0.42);
  color: #0f172a;
  font: inherit;
  cursor: pointer;
  backdrop-filter: blur(18px);
  box-shadow: 0 18px 48px rgba(104, 126, 156, 0.12);
}

:global(html[data-theme='dark'] .theme-switch) {
  background: rgba(13, 24, 42, 0.62);
  border-color: rgba(148, 163, 184, 0.18);
  color: #e2e8f0;
}

.login-shell {
  position: relative;
  z-index: 3;
  width: min(460px, calc(100vw - 40px));
}

.login-card {
  width: 100%;
  border-radius: 24px;
  border: 1px solid rgba(255, 255, 255, 0.34);
  background: rgba(255, 255, 255, 0.52);
  box-shadow: 0 40px 120px rgba(112, 130, 157, 0.18);
  backdrop-filter: blur(26px) saturate(160%);
  -webkit-backdrop-filter: blur(26px) saturate(160%);
}

:global(html[data-theme='dark'] .login-card) {
  border-color: rgba(148, 163, 184, 0.16);
  background: rgba(11, 18, 32, 0.68);
  box-shadow: 0 48px 120px rgba(2, 6, 23, 0.52);
}

.login-header {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  padding: 8px 0 24px;
}

.brand-eyebrow {
  margin-bottom: 16px;
  border-radius: 999px;
  padding: 6px 12px;
  background: rgba(255, 255, 255, 0.54);
  color: rgba(15, 23, 42, 0.72);
  font-size: 12px;
  letter-spacing: 0.08em;
}

:global(html[data-theme='dark'] .brand-eyebrow) {
  background: rgba(30, 41, 59, 0.64);
  color: rgba(226, 232, 240, 0.82);
}

.brand-name {
  color: #111827;
  font-size: 38px;
  font-weight: 700;
  letter-spacing: -0.04em;
}

:global(html[data-theme='dark'] .brand-name) {
  color: #f8fafc;
}

.login-header p {
  margin: 10px 0 0;
  font-size: 14px;
  color: rgba(15, 23, 42, 0.58);
}

:global(html[data-theme='dark'] .login-header p) {
  color: rgba(226, 232, 240, 0.7);
}

.login-form {
  width: 100%;
}

.login-error {
  margin: 4px 0 18px;
}

.login-submit {
  width: 100%;
  height: 54px;
  border: none;
  border-radius: 18px;
  background: linear-gradient(180deg, #2492ff 0%, #0a84ff 50%, #0071e3 100%);
  font-weight: 700;
  letter-spacing: 0.02em;
  box-shadow: 0 16px 40px rgba(0, 113, 227, 0.26);
}

:deep(.login-card .el-card__body) {
  padding: 36px 34px 32px;
}

:deep(.login-form .el-form-item) {
  margin-bottom: 16px;
}

:deep(.login-form .el-input__wrapper) {
  min-height: 54px;
  border-radius: 16px;
  background: rgba(248, 250, 252, 0.86);
  box-shadow: inset 0 0 0 1px rgba(148, 163, 184, 0.12);
  padding: 0 16px;
}

:global(html[data-theme='dark'] .login-form .el-input__wrapper) {
  background: rgba(15, 23, 42, 0.58);
  box-shadow: inset 0 0 0 1px rgba(148, 163, 184, 0.18);
}

:deep(.login-form .el-input__wrapper.is-focus) {
  box-shadow: inset 0 0 0 1px rgba(10, 132, 255, 0.36), 0 0 0 4px rgba(10, 132, 255, 0.08);
}

:deep(.login-form .el-input__inner) {
  color: #0f172a;
  font-size: 15px;
}

:global(html[data-theme='dark'] .login-form .el-input__inner) {
  color: #f8fafc;
}

:deep(.login-form .el-input__inner::placeholder) {
  color: rgba(15, 23, 42, 0.38);
}

:global(html[data-theme='dark'] .login-form .el-input__inner::placeholder) {
  color: rgba(226, 232, 240, 0.44);
}

:deep(.login-form .el-input__prefix) {
  display: none;
}

.password-item :deep(.el-form-item__content) {
  display: block;
}

.password-visibility {
  border: none;
  background: transparent;
  color: #0a84ff;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}

.caps-indicator {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-top: 10px;
  color: #d97706;
  font-size: 12px;
}

:global(html[data-theme='dark'] .caps-indicator) {
  color: #fbbf24;
}

@keyframes drift {
  0%, 100% { transform: translate3d(0, 0, 0); opacity: 0.14; }
  50% { transform: translate3d(0, -14px, 0); opacity: 0.42; }
}

@media (max-width: 640px) {
  .login-topbar { inset: 18px 18px auto auto; }
  .login-shell { width: calc(100vw - 24px); }
  :deep(.login-card .el-card__body) { padding: 28px 22px 24px; }
  .brand-name { font-size: 32px; }
}
</style>
