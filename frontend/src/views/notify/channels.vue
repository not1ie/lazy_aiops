<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>通知渠道</h2>
        <p class="page-desc">管理 webhook、飞书、钉钉、企微、邮件、短信渠道。</p>
      </div>
      <div class="page-actions">
        <el-button type="primary" icon="Plus" @click="openCreate">新增渠道</el-button>
        <el-button icon="Refresh" @click="fetchChannels">刷新</el-button>
      </div>
    </div>

    <el-table :data="channels" v-loading="loading" stripe style="width: 100%">
      <el-table-column prop="name" label="名称" min-width="160" />
      <el-table-column prop="type" label="类型" width="120" />
      <el-table-column prop="description" label="描述" min-width="240" />
      <el-table-column label="启用" width="100">
        <template #default="{ row }">
          <el-tag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? '是' : '否' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="260" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="openEdit(row)">编辑</el-button>
          <el-button size="small" @click="openTest(row)">测试</el-button>
          <el-button size="small" type="danger" @click="removeChannel(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog append-to-body v-model="dialogVisible" :title="dialogTitle" width="720px">
      <el-form :model="form" label-width="110px">
        <el-form-item label="名称">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="类型">
          <el-select v-model="form.type" class="w-52">
            <el-option label="webhook" value="webhook" />
            <el-option label="feishu" value="feishu" />
            <el-option label="dingtalk" value="dingtalk" />
            <el-option label="wecom" value="wecom" />
            <el-option label="email" value="email" />
            <el-option label="sms" value="sms" />
          </el-select>
        </el-form-item>
        <el-form-item label="Webhook" v-if="showWebhook">
          <el-input v-model="form.webhook" />
        </el-form-item>
        <el-form-item label="签名密钥" v-if="showWebhook">
          <el-input v-model="form.secret" show-password />
        </el-form-item>
        <el-form-item label="AppID" v-if="showAppAuth">
          <el-input v-model="form.app_id" />
        </el-form-item>
        <el-form-item label="AppSecret" v-if="showAppAuth">
          <el-input v-model="form.app_secret" show-password />
        </el-form-item>
        <el-form-item label="SMTP Host" v-if="form.type === 'email'">
          <el-input v-model="form.smtp_host" />
        </el-form-item>
        <el-form-item label="SMTP Port" v-if="form.type === 'email'">
          <el-input-number v-model="form.smtp_port" :min="1" :max="65535" />
        </el-form-item>
        <el-form-item label="SMTP 用户" v-if="form.type === 'email'">
          <el-input v-model="form.smtp_user" />
        </el-form-item>
        <el-form-item label="SMTP 密码" v-if="form.type === 'email'">
          <el-input v-model="form.smtp_pass" show-password />
        </el-form-item>
        <el-form-item label="短信厂商" v-if="form.type === 'sms'">
          <el-select v-model="form.sms_provider" class="w-52">
            <el-option label="aliyun" value="aliyun" />
            <el-option label="tencent" value="tencent" />
          </el-select>
        </el-form-item>
        <el-form-item label="短信签名" v-if="form.type === 'sms'">
          <el-input v-model="form.sms_sign" />
        </el-form-item>
        <el-form-item label="短信模板" v-if="form.type === 'sms'">
          <el-input v-model="form.sms_template" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="2" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitForm">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog append-to-body v-model="testVisible" title="测试通知渠道" width="420px">
      <el-form :model="testForm" label-width="80px">
        <el-form-item label="接收人">
          <el-input v-model="testForm.receiver" placeholder="可选：邮箱/手机号/用户ID" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="testVisible = false">取消</el-button>
        <el-button type="primary" @click="testChannel">发送测试</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const channels = ref([])
const loading = ref(false)
const dialogVisible = ref(false)
const dialogTitle = ref('')
const isEdit = ref(false)
const currentId = ref('')
const testVisible = ref(false)
const testId = ref('')
const testForm = ref({ receiver: '' })

const defaultForm = () => ({
  name: '',
  type: 'webhook',
  webhook: '',
  secret: '',
  app_id: '',
  app_secret: '',
  smtp_host: '',
  smtp_port: 25,
  smtp_user: '',
  smtp_pass: '',
  sms_provider: 'aliyun',
  sms_sign: '',
  sms_template: '',
  enabled: true,
  description: ''
})

const form = ref(defaultForm())

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const showWebhook = computed(() => ['webhook', 'feishu', 'dingtalk', 'wecom'].includes(form.value.type))
const showAppAuth = computed(() => ['feishu', 'wecom'].includes(form.value.type))

const fetchChannels = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/v1/notify/channels', { headers: authHeaders() })
    channels.value = res.data?.data || []
  } finally {
    loading.value = false
  }
}

const openCreate = () => {
  isEdit.value = false
  dialogTitle.value = '新增渠道'
  form.value = defaultForm()
  dialogVisible.value = true
}

const openEdit = (row) => {
  isEdit.value = true
  dialogTitle.value = '编辑渠道'
  currentId.value = row.id
  form.value = {
    ...defaultForm(),
    ...row
  }
  dialogVisible.value = true
}

const submitForm = async () => {
  if (!form.value.name.trim()) {
    ElMessage.warning('请填写名称')
    return
  }
  try {
    if (isEdit.value) {
      await axios.put(`/api/v1/notify/channels/${currentId.value}`, form.value, { headers: authHeaders() })
      ElMessage.success('更新成功')
    } else {
      await axios.post('/api/v1/notify/channels', form.value, { headers: authHeaders() })
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    await fetchChannels()
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '保存失败')
  }
}

const removeChannel = async (row) => {
  try {
    await ElMessageBox.confirm(`确认删除渠道 ${row.name} 吗？`, '提示', { type: 'warning' })
    await axios.delete(`/api/v1/notify/channels/${row.id}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    await fetchChannels()
  } catch (err) {
    if (err !== 'cancel') ElMessage.error('删除失败')
  }
}

const openTest = (row) => {
  testId.value = row.id
  testForm.value.receiver = ''
  testVisible.value = true
}

const testChannel = async () => {
  try {
    await axios.post(`/api/v1/notify/channels/${testId.value}/test`, testForm.value, { headers: authHeaders() })
    ElMessage.success('测试发送成功')
    testVisible.value = false
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '测试发送失败')
  }
}

onMounted(fetchChannels)
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 16px; }
.page-desc { color: #606266; margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; }
.w-52 { width: 220px; }
</style>
