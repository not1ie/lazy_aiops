<template>
  <div class="login-container">
    <div class="backdrop-grid"></div>
    <div class="backdrop-glow glow-a"></div>
    <div class="backdrop-glow glow-b"></div>
    <div class="backdrop-glow glow-c"></div>

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
      <section class="hero-panel">
        <div class="hero-badge">Unified Access Gateway</div>
        <h1>Lazy Auto Ops</h1>
        <p>统一接入资产、堡垒机、自动化与审计能力。</p>
        <div class="hero-metrics">
          <span>终端审计</span>
          <span>命令风控</span>
          <span>自动化编排</span>
        </div>
        <div class="hero-scene">
          <div class="scene-door">
            <span class="door-strip"></span>
            <span class="door-light"></span>
            <span class="door-knob"></span>
          </div>
          <div class="scene-character">
            <span class="head"></span>
            <span class="body"></span>
            <span class="arm"></span>
            <span class="leg left"></span>
            <span class="leg right"></span>
          </div>
          <div class="scene-floating-card">
            <div class="line short"></div>
            <div class="line"></div>
            <div class="line"></div>
            <div class="pill"></div>
          </div>
          <div class="scene-shadow"></div>
        </div>
      </section>

      <el-card class="login-card">
        <template #header>
          <div class="login-header">
            <h2>账号登录</h2>
            <p>请输入账号与密码</p>
          </div>
        </template>
        <el-form :model="form" :rules="rules" ref="formRef" label-position="top">
          <el-form-item label="用户名" prop="username">
            <el-input v-model="form.username" prefix-icon="User" placeholder="请输入用户名" @input="loginError = ''" />
          </el-form-item>
          <el-form-item label="密码" prop="password">
            <el-input v-model="form.password" prefix-icon="Lock" type="password" placeholder="请输入密码" show-password @input="loginError = ''" @keyup.enter="handleLogin" />
          </el-form-item>
          <el-alert v-if="loginError" :title="loginError" type="error" show-icon :closable="false" class="login-error" />
          <el-form-item>
            <el-button type="primary" :loading="loading" class="w-100" @click="handleLogin">登录</el-button>
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
  size: 2 + Math.random() * 4,
  delay: Math.random() * 6,
  duration: 8 + Math.random() * 14
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
  background: radial-gradient(circle at 8% 14%, #183f78 0%, #0f2c57 36%, #0b1d3b 100%);
}

.backdrop-grid {
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(rgba(255, 255, 255, 0.06) 1px, transparent 1px),
    linear-gradient(90deg, rgba(255, 255, 255, 0.06) 1px, transparent 1px);
  background-size: 32px 32px;
  mask-image: radial-gradient(circle at center, rgba(0, 0, 0, 1), rgba(0, 0, 0, 0.25));
  opacity: 0.38;
}

.backdrop-glow {
  position: absolute;
  border-radius: 50%;
  filter: blur(12px);
  opacity: 0.55;
}

.glow-a {
  width: 520px;
  height: 520px;
  left: -140px;
  top: 10%;
  background: radial-gradient(circle, rgba(54, 189, 255, 0.62), transparent 72%);
}

.glow-b {
  width: 460px;
  height: 460px;
  right: 8%;
  top: -120px;
  background: radial-gradient(circle, rgba(255, 169, 55, 0.5), transparent 72%);
}

.glow-c {
  width: 360px;
  height: 360px;
  right: 20%;
  bottom: -130px;
  background: radial-gradient(circle, rgba(128, 106, 255, 0.55), transparent 72%);
}

.particle-layer {
  position: absolute;
  inset: 0;
  pointer-events: none;
}

.particle-dot {
  position: absolute;
  border-radius: 999px;
  background: rgba(196, 226, 255, 0.75);
  animation-name: drift;
  animation-iteration-count: infinite;
  animation-timing-function: linear;
}

.login-shell {
  z-index: 3;
  width: min(1200px, 92vw);
  display: grid;
  grid-template-columns: 1.15fr 0.85fr;
  gap: 28px;
  align-items: center;
}

.hero-panel {
  border-radius: 22px;
  padding: 34px 36px;
  color: #d9e7ff;
  background: linear-gradient(145deg, rgba(13, 37, 74, 0.74), rgba(10, 28, 58, 0.62));
  border: 1px solid rgba(154, 195, 255, 0.26);
  box-shadow: 0 32px 68px rgba(4, 11, 27, 0.46);
  backdrop-filter: blur(8px);
}

.hero-badge {
  display: inline-flex;
  padding: 6px 12px;
  border-radius: 999px;
  background: rgba(120, 174, 255, 0.18);
  border: 1px solid rgba(152, 192, 255, 0.42);
  color: #b4d4ff;
  font-size: 12px;
  letter-spacing: 0.6px;
  margin-bottom: 16px;
}

.hero-panel h1 {
  margin: 0;
  font-size: 40px;
  font-weight: 700;
  letter-spacing: 0.4px;
  color: #f0f6ff;
}

.hero-panel p {
  margin: 14px 0 0;
  color: #afc7ea;
  font-size: 16px;
}

.hero-metrics {
  margin-top: 18px;
  display: flex;
  gap: 10px;
}

.hero-metrics span {
  font-size: 12px;
  color: #d7e8ff;
  border-radius: 999px;
  border: 1px solid rgba(166, 205, 255, 0.38);
  background: rgba(137, 185, 255, 0.14);
  padding: 4px 10px;
}

.hero-scene {
  margin-top: 26px;
  min-height: 300px;
  border-radius: 16px;
  position: relative;
  overflow: hidden;
  background: linear-gradient(110deg, #1a4f63 0%, #194556 48%, #f8b932 48%, #f3c446 100%);
}

.scene-door {
  position: absolute;
  left: 42%;
  top: 19%;
  width: 20%;
  height: 62%;
  border-radius: 6px;
  background: linear-gradient(180deg, #0f4f5f 0%, #0d3f4f 100%);
  box-shadow: inset -8px 0 0 rgba(6, 31, 39, 0.35), 0 8px 18px rgba(4, 18, 26, 0.36);
}

.door-strip {
  position: absolute;
  left: 18%;
  top: 14%;
  width: 64%;
  height: 56%;
  border-radius: 4px;
  background: rgba(14, 84, 100, 0.68);
}

.door-light {
  position: absolute;
  left: -28%;
  top: 12%;
  width: 22%;
  height: 70%;
  background: linear-gradient(90deg, rgba(255, 231, 127, 0.78), rgba(255, 231, 127, 0));
}

.door-knob {
  position: absolute;
  right: 12%;
  top: 52%;
  width: 10px;
  height: 10px;
  border-radius: 999px;
  background: #bfe8f7;
}

.scene-character {
  position: absolute;
  left: 20%;
  bottom: 16%;
  width: 90px;
  height: 160px;
}

.scene-character .head {
  position: absolute;
  left: 28px;
  top: 0;
  width: 34px;
  height: 34px;
  border-radius: 50%;
  background: #f6d7b5;
}

.scene-character .body {
  position: absolute;
  left: 24px;
  top: 32px;
  width: 42px;
  height: 68px;
  border-radius: 10px;
  background: linear-gradient(180deg, #0e6e84 0%, #0a5568 100%);
}

.scene-character .arm {
  position: absolute;
  left: 56px;
  top: 56px;
  width: 36px;
  height: 10px;
  border-radius: 5px;
  background: #0a5568;
  transform: rotate(-28deg);
  transform-origin: left center;
}

.scene-character .leg {
  position: absolute;
  top: 94px;
  width: 12px;
  height: 56px;
  border-radius: 6px;
  background: #163447;
}

.scene-character .leg.left { left: 30px; }
.scene-character .leg.right { left: 50px; }

.scene-floating-card {
  position: absolute;
  right: 12%;
  top: 26%;
  width: 188px;
  padding: 14px;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.88);
  box-shadow: 0 12px 24px rgba(17, 34, 66, 0.26);
  animation: floatCard 3.6s ease-in-out infinite;
}

.scene-floating-card .line {
  height: 8px;
  border-radius: 4px;
  margin-bottom: 10px;
  background: rgba(18, 57, 84, 0.18);
}

.scene-floating-card .line.short {
  width: 62%;
  background: rgba(245, 175, 37, 0.52);
}

.scene-floating-card .pill {
  margin-top: 4px;
  width: 100%;
  height: 26px;
  border-radius: 999px;
  background: linear-gradient(90deg, #0f6f83, #1780b0);
}

.scene-shadow {
  position: absolute;
  left: 16%;
  bottom: 8%;
  width: 180px;
  height: 14px;
  border-radius: 50%;
  background: rgba(6, 20, 32, 0.28);
}

.login-card {
  width: min(430px, 100%);
  border-radius: 18px;
  border: 1px solid rgba(179, 210, 255, 0.26);
  background: rgba(242, 248, 255, 0.94);
  box-shadow: 0 24px 46px rgba(7, 16, 35, 0.36);
  backdrop-filter: blur(12px);
}

.login-header { text-align: center; }
.login-header h2 {
  margin: 0;
  color: #12466f;
  font-size: 28px;
  letter-spacing: 0.4px;
}
.login-header p {
  margin: 10px 0 0;
  font-size: 14px;
  color: #5f7286;
}
.login-error { margin-bottom: 16px; }
.w-100 {
  width: 100%;
  height: 42px;
}

@keyframes drift {
  0%, 100% { transform: translate3d(0, 0, 0); opacity: 0.65; }
  50% { transform: translate3d(0, -18px, 0); opacity: 1; }
}

@keyframes floatCard {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-10px); }
}

@media (max-width: 980px) {
  .login-shell {
    grid-template-columns: 1fr;
    width: min(480px, 92vw);
  }
  .hero-panel {
    display: none;
  }
  .login-card {
    margin: 0 auto;
  }
}
</style>
