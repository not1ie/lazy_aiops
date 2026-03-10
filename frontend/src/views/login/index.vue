<template>
  <div class="login-container">
    <div class="backdrop-glow glow-a"></div>
    <div class="backdrop-glow glow-b"></div>
    <div class="backdrop-glow glow-c"></div>
    <div class="backdrop-glow glow-d"></div>

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

    <div class="login-shell">
      <el-card class="login-card">
        <div class="login-header">
          <div class="brand-mark">L</div>
          <div class="brand-name">Lazy Auto Ops</div>
          <p>统一接入资产、堡垒机与自动化能力</p>
        </div>
        <el-form :model="form" :rules="rules" ref="formRef" class="login-form">
          <el-form-item prop="username">
            <el-input v-model="form.username" class="auth-input" placeholder="用户名" @input="loginError = ''" />
          </el-form-item>
          <el-form-item prop="password">
            <el-input v-model="form.password" class="auth-input" type="password" placeholder="密码" @input="loginError = ''" @keyup.enter="handleLogin" />
          </el-form-item>
          <el-alert v-if="loginError" :title="loginError" type="error" show-icon :closable="false" class="login-error" />
          <el-form-item>
            <el-button type="primary" :loading="loading" class="login-submit" @click="handleLogin">登录</el-button>
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

const router = useRouter()
const formRef = ref(null)
const loading = ref(false)
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

const createParticles = () => Array.from({ length: 42 }).map((_, index) => ({
  id: index + 1,
  left: Math.random() * 100,
  top: Math.random() * 100,
  size: 2 + Math.random() * 3,
  delay: Math.random() * 6,
  duration: 12 + Math.random() * 16
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

const handleLogin = async () => {
  if (!formRef.value) return
  loginError.value = ''
  await formRef.value.validate(async (valid) => {
    if (valid) {
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
            ElMessageBox.alert('当前使用的是默认管理员密码，建议立即修改密码以提升安全性。', '安全提示', {
              confirmButtonText: '我知道了',
              type: 'warning'
            })
          }
          router.push('/')
        } else {
          loginError.value = parseLoginError(res.data.message, 401)
          ElMessage.error(loginError.value)
        }
      } catch (e) {
        const statusCode = Number(e?.response?.status || 0)
        const msg = e?.response?.data?.message || e?.message || ''
        loginError.value = parseLoginError(msg, statusCode, e?.code)
        ElMessage.error(loginError.value)
      } finally {
        loading.value = false
      }
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

onMounted(() => {
  particles.value = createParticles()
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
  background:
    radial-gradient(circle at 12% 18%, rgba(132, 192, 255, 0.42), transparent 36%),
    radial-gradient(circle at 84% 16%, rgba(255, 192, 140, 0.26), transparent 34%),
    radial-gradient(circle at 72% 78%, rgba(190, 179, 255, 0.22), transparent 28%),
    linear-gradient(180deg, #edf3fb 0%, #dfe8f4 48%, #d7e2ef 100%);
}

.backdrop-glow {
  position: absolute;
  border-radius: 50%;
  filter: blur(70px);
  opacity: 0.72;
}

.glow-a {
  width: 420px;
  height: 420px;
  left: -80px;
  top: -40px;
  background: radial-gradient(circle, rgba(112, 185, 255, 0.42), transparent 70%);
}

.glow-b {
  width: 360px;
  height: 360px;
  right: 6%;
  top: 4%;
  background: radial-gradient(circle, rgba(255, 208, 161, 0.38), transparent 72%);
}

.glow-c {
  width: 340px;
  height: 340px;
  left: 14%;
  bottom: -80px;
  background: radial-gradient(circle, rgba(154, 204, 255, 0.28), transparent 70%);
}

.glow-d {
  width: 380px;
  height: 380px;
  right: 18%;
  bottom: -120px;
  background: radial-gradient(circle, rgba(182, 164, 255, 0.24), transparent 72%);
}

.particle-layer {
  position: absolute;
  inset: 0;
  pointer-events: none;
}

.particle-dot {
  position: absolute;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.42);
  filter: blur(0.4px);
  animation-name: drift;
  animation-iteration-count: infinite;
  animation-timing-function: linear;
}

.login-shell {
  z-index: 3;
  width: min(460px, calc(100vw - 40px));
}

.login-card {
  width: 100%;
  border-radius: 24px;
  border: 1px solid rgba(255, 255, 255, 0.36);
  background: rgba(255, 255, 255, 0.5);
  box-shadow:
    0 40px 100px rgba(120, 142, 170, 0.18),
    0 18px 48px rgba(110, 128, 152, 0.08);
  backdrop-filter: blur(26px) saturate(160%);
  -webkit-backdrop-filter: blur(26px) saturate(160%);
  overflow: hidden;
}

.login-header {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  padding: 8px 0 24px;
}

.brand-mark {
  width: 54px;
  height: 54px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 18px;
  background: rgba(255, 255, 255, 0.48);
  border: 1px solid rgba(255, 255, 255, 0.52);
  color: #0f172a;
  font-size: 24px;
  font-weight: 700;
  letter-spacing: 0.04em;
  box-shadow: 0 10px 30px rgba(120, 134, 151, 0.12);
}

.brand-name {
  margin-top: 18px;
  color: #101828;
  font-size: 32px;
  font-weight: 600;
  letter-spacing: -0.03em;
}

.login-header h2 {
  margin: 0;
  color: #12466f;
  font-size: 28px;
  letter-spacing: 0.4px;
}
.login-header p {
  margin: 10px 0 0;
  font-size: 14px;
  color: rgba(15, 23, 42, 0.62);
  letter-spacing: -0.01em;
}

.login-form {
  width: 100%;
}

.login-error {
  margin: 2px 0 18px;
}

.login-submit {
  width: 100%;
  height: 52px;
  border: none;
  border-radius: 16px;
  background: linear-gradient(180deg, #0a84ff 0%, #0071e3 100%);
  font-weight: 700;
  letter-spacing: 0.02em;
  box-shadow: 0 10px 24px rgba(0, 113, 227, 0.22);
}

:deep(.login-card .el-card__body) {
  padding: 34px 32px 30px;
}

:deep(.login-form .el-form-item) {
  margin-bottom: 16px;
}

:deep(.login-form .el-input__wrapper) {
  min-height: 52px;
  border-radius: 16px;
  background: rgba(241, 245, 249, 0.84);
  box-shadow: inset 0 0 0 1px rgba(148, 163, 184, 0.12);
  padding: 0 16px;
}

:deep(.login-form .el-input__wrapper.is-focus) {
  box-shadow:
    inset 0 0 0 1px rgba(10, 132, 255, 0.32),
    0 0 0 4px rgba(10, 132, 255, 0.08);
}

:deep(.login-form .el-input__inner) {
  color: #0f172a;
  font-size: 15px;
}

:deep(.login-form .el-input__inner::placeholder) {
  color: rgba(15, 23, 42, 0.42);
}

:deep(.login-form .el-input__prefix),
:deep(.login-form .el-input__suffix) {
  display: none;
}

@keyframes drift {
  0%, 100% { transform: translate3d(0, 0, 0); opacity: 0.2; }
  50% { transform: translate3d(0, -20px, 0); opacity: 0.55; }
}

@media (max-width: 980px) {
  .login-shell { width: min(460px, calc(100vw - 32px)); }
}

@media (max-width: 640px) {
  .login-shell { width: calc(100vw - 24px); }
  :deep(.login-card .el-card__body) { padding: 26px 22px 22px; }
  .brand-name { font-size: 28px; }
}
</style>
