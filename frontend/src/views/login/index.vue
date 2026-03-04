<template>
  <div class="login-container">
    <div class="bg-layer bg-grid"></div>
    <div class="bg-layer bg-glow bg-glow-a"></div>
    <div class="bg-layer bg-glow bg-glow-b"></div>
    <div class="bg-layer bg-glow bg-glow-c"></div>
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
import { ref, reactive } from 'vue'
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
        ElMessage.error('网络错误')
      } finally {
        loading.value = false
      }
    }
  })
}
</script>

<style scoped>
.login-container {
  position: relative;
  overflow: hidden;
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100vh;
  background:
    radial-gradient(1200px 600px at 90% -10%, rgba(23, 109, 255, 0.3) 0%, rgba(23, 109, 255, 0) 70%),
    radial-gradient(900px 500px at -10% 110%, rgba(0, 210, 255, 0.22) 0%, rgba(0, 210, 255, 0) 68%),
    #0f172a;
}

.bg-layer {
  position: absolute;
  inset: 0;
  pointer-events: none;
}

.bg-grid {
  opacity: 0.18;
  background-image:
    linear-gradient(rgba(255, 255, 255, 0.16) 1px, transparent 1px),
    linear-gradient(90deg, rgba(255, 255, 255, 0.16) 1px, transparent 1px);
  background-size: 36px 36px;
  mask-image: radial-gradient(circle at center, #000 30%, transparent 80%);
}

.bg-glow {
  filter: blur(60px);
  transform: translate3d(0, 0, 0);
  animation: floatGlow 14s ease-in-out infinite;
}

.bg-glow-a {
  width: 420px;
  height: 420px;
  left: -120px;
  top: 10%;
  background: rgba(59, 130, 246, 0.45);
}

.bg-glow-b {
  width: 500px;
  height: 500px;
  right: -180px;
  top: 35%;
  background: rgba(34, 197, 94, 0.35);
  animation-delay: -4s;
}

.bg-glow-c {
  width: 320px;
  height: 320px;
  left: 40%;
  bottom: -140px;
  background: rgba(236, 72, 153, 0.32);
  animation-delay: -8s;
}

.login-card {
  position: relative;
  z-index: 2;
  width: 400px;
  border-radius: 14px;
  backdrop-filter: blur(8px);
}

.login-header { text-align: center; }
.login-header h2 { margin: 0; color: #409eff; }
.login-header p { margin: 10px 0 0; color: #909399; font-size: 14px; }
.w-100 { width: 100%; }

@keyframes floatGlow {
  0%,
  100% {
    transform: translate3d(0, 0, 0) scale(1);
  }
  50% {
    transform: translate3d(12px, -18px, 0) scale(1.06);
  }
}
</style>
