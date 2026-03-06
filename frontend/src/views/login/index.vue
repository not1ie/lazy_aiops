<template>
  <div class="login-container">
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
      <div class="login-illustration">
        <div class="scene-card">
          <div class="scene-left">
            <div class="scene-door">
              <span class="door-window top"></span>
              <span class="door-window bottom"></span>
              <span class="door-knob"></span>
            </div>
          </div>
          <div class="scene-right">
            <div class="scene-tip">SECURE ACCESS</div>
            <div class="scene-sub">统一入口，集中审计</div>
          </div>
        </div>
      </div>

      <el-card class="login-card">
        <template #header>
          <div class="login-header">
            <h2>Lazy Auto Ops</h2>
            <p>企业级自动化运维平台</p>
          </div>
        </template>
        <el-form :model="form" :rules="rules" ref="formRef" label-position="top">
          <el-form-item label="用户名" prop="username">
            <el-input v-model="form.username" prefix-icon="User" placeholder="请输入用户名" />
          </el-form-item>
          <el-form-item label="密码" prop="password">
            <el-input v-model="form.password" prefix-icon="Lock" type="password" placeholder="请输入密码" show-password @keyup.enter="handleLogin" />
          </el-form-item>
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
  await formRef.value.validate(async (valid) => {
    if (valid) {
      loading.value = true
      try {
        const res = await axios.post('/api/v1/login', form)
        if (res.data.code === 0) {
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
          ElMessage.error(res.data.message || '登录失败')
        }
      } catch (e) {
        const msg = e?.response?.data?.message
        if (msg) {
          ElMessage.error(msg)
        } else if (e?.code === 'ECONNABORTED') {
          ElMessage.error('请求超时，请稍后重试')
        } else if (!e?.response) {
          ElMessage.error('网络错误：无法连接到服务端')
        } else {
          ElMessage.error('登录失败，请稍后重试')
        }
      } finally {
        loading.value = false
      }
    }
  })
}

onMounted(() => {
  particles.value = createParticles()
})
</script>

<style scoped>
.login-container {
  position: relative;
  overflow: hidden;
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100vh;
  background: linear-gradient(120deg, #0d213f 0%, #132d57 45%, #203866 100%);
}

.particle-layer {
  position: absolute;
  inset: 0;
  pointer-events: none;
}

.particle-dot {
  position: absolute;
  border-radius: 999px;
  background: rgba(176, 213, 255, 0.58);
  animation-name: drift;
  animation-iteration-count: infinite;
  animation-timing-function: ease-in-out;
}

.login-shell {
  z-index: 2;
  width: min(1180px, 92vw);
  display: grid;
  grid-template-columns: 1.2fr 0.9fr;
  gap: 34px;
  align-items: center;
}

.login-illustration {
  display: flex;
  justify-content: center;
}

.scene-card {
  width: min(650px, 100%);
  min-height: 430px;
  border-radius: 18px;
  overflow: hidden;
  box-shadow: 0 24px 54px rgba(7, 19, 43, 0.45);
  display: grid;
  grid-template-columns: 1fr 1fr;
  background: #edf3fb;
}

.scene-left {
  position: relative;
  background: linear-gradient(160deg, #1d5060 0%, #0f3a48 100%);
}

.scene-left::before {
  content: '';
  position: absolute;
  inset: 0;
  background: radial-gradient(circle at 20% 20%, rgba(255, 255, 255, 0.15), transparent 55%);
}

.scene-right {
  background: linear-gradient(180deg, #ffcd45 0%, #f5b923 100%);
  color: #0d3442;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  gap: 14px;
}

.scene-tip {
  font-size: 28px;
  font-weight: 800;
  letter-spacing: 0.8px;
}

.scene-sub {
  font-size: 16px;
  font-weight: 500;
}

.scene-door {
  position: absolute;
  right: 18%;
  bottom: 16%;
  width: 42%;
  height: 64%;
  border-radius: 5px;
  background: linear-gradient(180deg, #1c7288 0%, #135d6f 100%);
  box-shadow: inset -8px 0 0 rgba(7, 45, 57, 0.35);
}

.door-window {
  position: absolute;
  left: 17%;
  width: 66%;
  height: 18%;
  border-radius: 4px;
  background: rgba(11, 62, 76, 0.58);
}

.door-window.top {
  top: 18%;
}

.door-window.bottom {
  top: 48%;
}

.door-knob {
  position: absolute;
  right: 10%;
  top: 54%;
  width: 10px;
  height: 10px;
  border-radius: 999px;
  background: #b4e2f2;
}

.login-card {
  position: relative;
  width: min(420px, 100%);
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.9);
  backdrop-filter: blur(10px);
  box-shadow: 0 20px 42px rgba(9, 22, 42, 0.32);
}

.login-header { text-align: center; }
.login-header h2 { margin: 0; color: #409eff; }
.login-header p { margin: 10px 0 0; color: #909399; font-size: 14px; }
.w-100 { width: 100%; }

@keyframes drift {
  0%,
  100% {
    transform: translate3d(0, 0, 0);
  }
  50% {
    transform: translate3d(0, -14px, 0);
  }
}

@media (max-width: 980px) {
  .login-shell {
    grid-template-columns: 1fr;
    width: min(460px, 92vw);
  }
  .login-illustration {
    display: none;
  }
}
</style>
