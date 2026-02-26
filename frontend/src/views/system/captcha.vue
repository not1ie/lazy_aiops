<template>
  <el-card class="page-card" v-loading="loading">
    <div class="page-header">
      <div>
        <h2>验证码配置</h2>
        <p class="page-desc">配置登录验证码策略，变更后立即生效。</p>
      </div>
      <div class="page-actions">
        <el-button icon="Refresh" @click="fetchConfig">刷新</el-button>
        <el-button type="primary" :loading="saving" @click="saveConfig">保存配置</el-button>
      </div>
    </div>

    <el-row :gutter="16">
      <el-col :md="16" :sm="24">
        <el-form :model="form" label-width="130px" class="form-panel">
          <el-form-item label="启用验证码">
            <el-switch v-model="form.enabled" inline-prompt active-text="启用" inactive-text="关闭" />
          </el-form-item>
          <el-form-item label="验证码类型">
            <el-select v-model="form.type" style="width: 220px">
              <el-option label="算术验证码" value="math" />
              <el-option label="随机字符串" value="string" />
            </el-select>
          </el-form-item>
          <el-form-item label="长度">
            <el-input-number v-model="form.length" :min="2" :max="8" />
          </el-form-item>
          <el-form-item label="过期时间(秒)">
            <el-input-number v-model="form.expire_seconds" :min="30" :max="600" :step="10" />
          </el-form-item>
          <el-form-item label="噪点等级">
            <el-slider v-model="form.noise_level" :min="0" :max="5" style="width: 280px" />
          </el-form-item>
          <el-form-item label="背景色">
            <el-select v-model="form.background" style="width: 220px">
              <el-option label="白色" value="white" />
              <el-option label="灰色" value="gray" />
              <el-option label="深色" value="dark" />
            </el-select>
          </el-form-item>
          <el-form-item label="区分大小写">
            <el-switch v-model="form.case_sensitive" inline-prompt active-text="是" inactive-text="否" />
          </el-form-item>
        </el-form>
      </el-col>
      <el-col :md="8" :sm="24">
        <el-card class="preview-card">
          <template #header>
            <span>配置预览</span>
          </template>
          <el-descriptions :column="1" border>
            <el-descriptions-item label="状态">{{ form.enabled ? '已启用' : '已关闭' }}</el-descriptions-item>
            <el-descriptions-item label="类型">{{ form.type === 'math' ? '算术验证码' : '随机字符串' }}</el-descriptions-item>
            <el-descriptions-item label="长度">{{ form.length }}</el-descriptions-item>
            <el-descriptions-item label="过期">{{ form.expire_seconds }} 秒</el-descriptions-item>
            <el-descriptions-item label="噪点">{{ form.noise_level }}</el-descriptions-item>
            <el-descriptions-item label="背景">{{ backgroundText }}</el-descriptions-item>
            <el-descriptions-item label="大小写">{{ form.case_sensitive ? '区分' : '不区分' }}</el-descriptions-item>
          </el-descriptions>

          <div class="mock-captcha" :class="`bg-${form.background}`">
            <span>{{ mockText }}</span>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </el-card>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'

const loading = ref(false)
const saving = ref(false)

const form = reactive({
  enabled: true,
  type: 'math',
  length: 4,
  expire_seconds: 120,
  noise_level: 1,
  background: 'white',
  case_sensitive: false
})

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const backgroundText = computed(() => {
  if (form.background === 'gray') return '灰色'
  if (form.background === 'dark') return '深色'
  return '白色'
})

const mockText = computed(() => {
  if (form.type === 'math') return '8 + 3 = ?'
  const base = form.case_sensitive ? 'aB3kX9Zp' : 'AB3KX9ZP'
  return base.slice(0, Number(form.length || 4))
})

const fetchConfig = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/v1/system/captcha', { headers: authHeaders() })
    if (res.data?.code === 0 && res.data.data) {
      const data = res.data.data
      form.enabled = !!data.enabled
      form.type = data.type || 'math'
      form.length = Number(data.length || 4)
      form.expire_seconds = Number(data.expire_seconds || 120)
      form.noise_level = Number(data.noise_level || 1)
      form.background = data.background || 'white'
      form.case_sensitive = !!data.case_sensitive
    }
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '加载配置失败')
  } finally {
    loading.value = false
  }
}

const saveConfig = async () => {
  saving.value = true
  try {
    await axios.put('/api/v1/system/captcha', {
      enabled: form.enabled,
      type: form.type,
      length: Number(form.length || 4),
      expire_seconds: Number(form.expire_seconds || 120),
      noise_level: Number(form.noise_level || 1),
      background: form.background,
      case_sensitive: form.case_sensitive
    }, { headers: authHeaders() })
    ElMessage.success('保存成功')
    await fetchConfig()
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '保存失败')
  } finally {
    saving.value = false
  }
}

onMounted(fetchConfig)
</script>

<style scoped>
.page-card { max-width: 1200px; margin: 0 auto; }
.page-header { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-bottom: 12px; }
.page-desc { color: #909399; margin: 4px 0 0; }
.page-actions { display: flex; align-items: center; gap: 8px; }
.form-panel { padding-top: 4px; }
.preview-card { height: 100%; }
.mock-captcha {
  margin-top: 16px;
  border: 1px dashed #dcdfe6;
  border-radius: 6px;
  height: 72px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  letter-spacing: 4px;
  user-select: none;
}
.bg-white { background: #fff; color: #303133; }
.bg-gray { background: #f2f3f5; color: #303133; }
.bg-dark { background: #303133; color: #fff; }
</style>
