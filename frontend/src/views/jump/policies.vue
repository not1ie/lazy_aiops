<template>
  <el-row :gutter="12">
    <el-col :md="10" :sm="24">
      <el-card class="page-card">
        <template #header>
          <div class="header">
            <div>
              <div class="title">登录账号</div>
              <div class="desc">维护堡垒机使用的目标登录账号。</div>
            </div>
            <div class="actions">
              <el-button type="primary" icon="Plus" @click="openAccountCreate">新增账号</el-button>
              <el-button icon="Refresh" @click="loadAccounts">刷新</el-button>
            </div>
          </div>
        </template>

        <el-table :fit="false" :data="accounts" v-loading="accountLoading" stripe>
          <el-table-column prop="name" label="名称" min-width="130" />
          <el-table-column prop="username" label="登录名" min-width="120" />
          <el-table-column prop="auth_type" label="认证" width="100" />
          <el-table-column label="Secret" width="90">
            <template #default="{ row }">
              <el-tag :type="row.has_secret ? 'success' : 'info'" size="small">{{ row.has_secret ? '已配置' : '未配置' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="80">
            <template #default="{ row }">
              <el-tag :type="row.enabled ? 'success' : 'info'" size="small">{{ row.enabled ? '启用' : '禁用' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="160" fixed="right">
            <template #default="{ row }">
              <el-button size="small" type="primary" plain @click="openAccountEdit(row)">编辑</el-button>
              <el-button size="small" type="danger" plain @click="removeAccount(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-card>
    </el-col>

    <el-col :md="14" :sm="24">
      <el-card class="page-card">
        <template #header>
          <div class="header">
            <div>
              <div class="title">授权策略</div>
              <div class="desc">按用户/角色绑定可访问资产与账号。</div>
            </div>
            <div class="actions">
              <el-button type="primary" icon="Plus" @click="openPolicyCreate">新增策略</el-button>
              <el-button icon="Refresh" @click="loadPolicies">刷新</el-button>
            </div>
          </div>
        </template>

        <el-table :fit="false" :data="policies" v-loading="policyLoading" stripe>
          <el-table-column prop="name" label="策略" min-width="140" />
          <el-table-column label="授权对象" min-width="140">
            <template #default="{ row }">{{ principalText(row) }}</template>
          </el-table-column>
          <el-table-column prop="asset_name" label="资产" min-width="140" show-overflow-tooltip />
          <el-table-column prop="account_name" label="账号" min-width="120" />
          <el-table-column prop="protocol" label="协议" width="90" />
          <el-table-column label="审批" width="70">
            <template #default="{ row }">{{ row.require_approve ? '是' : '否' }}</template>
          </el-table-column>
          <el-table-column label="状态" width="80">
            <template #default="{ row }">
              <el-tag :type="row.status === 1 ? 'success' : 'info'" size="small">{{ row.status === 1 ? '启用' : '禁用' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="160" fixed="right">
            <template #default="{ row }">
              <el-button size="small" type="primary" plain @click="openPolicyEdit(row)">编辑</el-button>
              <el-button size="small" type="danger" plain @click="removePolicy(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-card>
    </el-col>
  </el-row>

  <el-dialog append-to-body v-model="accountDialogVisible" :title="accountEditing ? '编辑账号' : '新增账号'" width="560px">
    <el-form :model="accountForm" label-width="90px">
      <el-form-item label="名称" required>
        <el-input v-model="accountForm.name" />
      </el-form-item>
      <el-form-item label="登录名" required>
        <el-input v-model="accountForm.username" />
      </el-form-item>
      <el-form-item label="认证方式">
        <el-select v-model="accountForm.auth_type" style="width: 100%">
          <el-option label="password" value="password" />
          <el-option label="key" value="key" />
          <el-option label="token" value="token" />
        </el-select>
      </el-form-item>
      <el-form-item label="密钥/口令">
        <el-input v-model="accountForm.secret" type="textarea" :rows="3" placeholder="password/私钥/token" show-password />
      </el-form-item>
      <el-form-item label="描述">
        <el-input v-model="accountForm.description" type="textarea" :rows="2" />
      </el-form-item>
      <el-form-item label="启用">
        <el-switch v-model="accountForm.enabled" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="accountDialogVisible = false">取消</el-button>
      <el-button type="primary" :loading="accountSaving" @click="saveAccount">保存</el-button>
    </template>
  </el-dialog>

  <el-dialog append-to-body v-model="policyDialogVisible" :title="policyEditing ? '编辑策略' : '新增策略'" width="720px">
    <el-form :model="policyForm" label-width="98px">
      <el-form-item label="策略名称" required>
        <el-input v-model="policyForm.name" />
      </el-form-item>
      <el-form-item label="资产/账号" required>
        <div class="inline-fields">
          <el-select v-model="policyForm.asset_id" filterable placeholder="选择资产">
            <el-option v-for="item in assets" :key="item.id" :label="`${item.name}(${item.protocol})`" :value="item.id" />
          </el-select>
          <el-select v-model="policyForm.account_id" filterable placeholder="选择账号">
            <el-option v-for="item in accounts" :key="item.id" :label="`${item.name}/${item.username}`" :value="item.id" />
          </el-select>
        </div>
      </el-form-item>
      <el-form-item label="授权对象" required>
        <div class="inline-fields">
          <el-select v-model="policyForm.user_id" clearable filterable placeholder="用户（可选）">
            <el-option v-for="item in users" :key="item.id" :label="`${item.username}${item.nickname ? `(${item.nickname})` : ''}`" :value="item.id" />
          </el-select>
          <el-select v-model="policyForm.role_code" clearable filterable placeholder="角色（可选）">
            <el-option v-for="item in roles" :key="item.code" :label="`${item.name}(${item.code})`" :value="item.code" />
          </el-select>
        </div>
      </el-form-item>
      <el-form-item label="协议">
        <el-select v-model="policyForm.protocol" style="width: 100%" clearable placeholder="为空时跟随资产协议">
          <el-option label="ssh" value="ssh" />
          <el-option label="docker" value="docker" />
          <el-option label="k8s" value="k8s" />
          <el-option label="mysql" value="mysql" />
          <el-option label="postgres" value="postgres" />
          <el-option label="redis" value="redis" />
          <el-option label="mongodb" value="mongodb" />
        </el-select>
      </el-form-item>
      <el-form-item label="过期时间">
        <el-date-picker
          v-model="policyForm.expires_at"
          type="datetime"
          value-format="YYYY-MM-DDTHH:mm:ssZ"
          placeholder="可选"
          style="width: 100%"
        />
      </el-form-item>
      <el-form-item label="策略选项">
        <div class="inline-fields">
          <el-select v-model="policyForm.status">
            <el-option label="启用" :value="1" />
            <el-option label="禁用" :value="0" />
          </el-select>
          <el-switch v-model="policyForm.require_approve" active-text="需要审批" inactive-text="无需审批" />
        </div>
      </el-form-item>
      <el-form-item label="说明">
        <el-input v-model="policyForm.description" type="textarea" :rows="3" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="policyDialogVisible = false">取消</el-button>
      <el-button type="primary" :loading="policySaving" @click="savePolicy">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const accountLoading = ref(false)
const policyLoading = ref(false)
const accountSaving = ref(false)
const policySaving = ref(false)

const accounts = ref([])
const policies = ref([])
const assets = ref([])
const users = ref([])
const roles = ref([])

const accountDialogVisible = ref(false)
const policyDialogVisible = ref(false)
const accountEditing = ref(false)
const policyEditing = ref(false)
const accountID = ref('')
const policyID = ref('')

const accountForm = reactive({
  name: '',
  username: '',
  auth_type: 'password',
  secret: '',
  description: '',
  enabled: true
})

const policyForm = reactive({
  name: '',
  user_id: '',
  role_code: '',
  asset_id: '',
  account_id: '',
  protocol: '',
  require_approve: false,
  expires_at: null,
  status: 1,
  description: ''
})

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const principalText = (row) => {
  if (row.user_id && row.role_code) return `用户:${row.user_id} / 角色:${row.role_code}`
  if (row.user_id) return `用户:${row.user_id}`
  if (row.role_code) return `角色:${row.role_code}`
  return '-'
}

const loadAssets = async () => {
  const res = await axios.get('/api/v1/jump/assets', { headers: authHeaders() })
  if (res.data.code === 0) assets.value = res.data.data || []
}

const loadAccounts = async () => {
  accountLoading.value = true
  try {
    const res = await axios.get('/api/v1/jump/accounts', { headers: authHeaders() })
    if (res.data.code === 0) {
      accounts.value = res.data.data || []
    }
  } catch (error) {
    ElMessage.error(error?.response?.data?.message || '加载账号失败')
  } finally {
    accountLoading.value = false
  }
}

const loadPolicies = async () => {
  policyLoading.value = true
  try {
    const res = await axios.get('/api/v1/jump/policies', { headers: authHeaders() })
    if (res.data.code === 0) {
      policies.value = res.data.data || []
    }
  } catch (error) {
    ElMessage.error(error?.response?.data?.message || '加载策略失败')
  } finally {
    policyLoading.value = false
  }
}

const loadUsersAndRoles = async () => {
  try {
    const [usersRes, rolesRes] = await Promise.all([
      axios.get('/api/v1/rbac/users', { headers: authHeaders() }),
      axios.get('/api/v1/rbac/roles', { headers: authHeaders() })
    ])
    if (usersRes.data.code === 0) users.value = usersRes.data.data || []
    if (rolesRes.data.code === 0) roles.value = rolesRes.data.data || []
  } catch {
    // 非管理员角色可能无权限读取，忽略即可
  }
}

const resetAccountForm = () => {
  Object.assign(accountForm, {
    name: '',
    username: '',
    auth_type: 'password',
    secret: '',
    description: '',
    enabled: true
  })
}

const resetPolicyForm = () => {
  Object.assign(policyForm, {
    name: '',
    user_id: '',
    role_code: '',
    asset_id: '',
    account_id: '',
    protocol: '',
    require_approve: false,
    expires_at: null,
    status: 1,
    description: ''
  })
}

const openAccountCreate = () => {
  accountEditing.value = false
  accountID.value = ''
  resetAccountForm()
  accountDialogVisible.value = true
}

const openAccountEdit = (row) => {
  accountEditing.value = true
  accountID.value = row.id
  Object.assign(accountForm, {
    name: row.name || '',
    username: row.username || '',
    auth_type: row.auth_type || 'password',
    secret: '',
    description: row.description || '',
    enabled: row.enabled !== false
  })
  accountDialogVisible.value = true
}

const saveAccount = async () => {
  if (!accountForm.name || !accountForm.username) {
    ElMessage.warning('请填写账号名称与登录名')
    return
  }
  accountSaving.value = true
  try {
    const payload = { ...accountForm }
    if (accountEditing.value && !payload.secret) {
      delete payload.secret
    }
    let res
    if (accountEditing.value) {
      res = await axios.put(`/api/v1/jump/accounts/${accountID.value}`, payload, { headers: authHeaders() })
    } else {
      res = await axios.post('/api/v1/jump/accounts', payload, { headers: authHeaders() })
    }
    if (res.data.code === 0) {
      ElMessage.success(accountEditing.value ? '账号已更新' : '账号已创建')
      accountDialogVisible.value = false
      loadAccounts()
      loadPolicies()
    }
  } catch (error) {
    ElMessage.error(error?.response?.data?.message || '保存账号失败')
  } finally {
    accountSaving.value = false
  }
}

const removeAccount = (row) => {
  ElMessageBox.confirm(`确认删除账号「${row.name}」吗？`, '提示', { type: 'warning' }).then(async () => {
    await axios.delete(`/api/v1/jump/accounts/${row.id}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    loadAccounts()
    loadPolicies()
  }).catch(() => {})
}

const openPolicyCreate = () => {
  policyEditing.value = false
  policyID.value = ''
  resetPolicyForm()
  policyDialogVisible.value = true
}

const openPolicyEdit = (row) => {
  policyEditing.value = true
  policyID.value = row.id
  Object.assign(policyForm, {
    name: row.name || '',
    user_id: row.user_id || '',
    role_code: row.role_code || '',
    asset_id: row.asset_id || '',
    account_id: row.account_id || '',
    protocol: row.protocol || '',
    require_approve: row.require_approve === true,
    expires_at: row.expires_at || null,
    status: row.status === 0 ? 0 : 1,
    description: row.description || ''
  })
  policyDialogVisible.value = true
}

const savePolicy = async () => {
  if (!policyForm.name || !policyForm.asset_id || !policyForm.account_id) {
    ElMessage.warning('请填写策略名、资产、账号')
    return
  }
  if (!policyForm.user_id && !policyForm.role_code) {
    ElMessage.warning('请至少选择一个用户或角色')
    return
  }

  policySaving.value = true
  try {
    const payload = {
      ...policyForm,
      protocol: policyForm.protocol || undefined,
      expires_at: policyForm.expires_at || null
    }
    let res
    if (policyEditing.value) {
      res = await axios.put(`/api/v1/jump/policies/${policyID.value}`, payload, { headers: authHeaders() })
    } else {
      res = await axios.post('/api/v1/jump/policies', payload, { headers: authHeaders() })
    }
    if (res.data.code === 0) {
      ElMessage.success(policyEditing.value ? '策略已更新' : '策略已创建')
      policyDialogVisible.value = false
      loadPolicies()
    }
  } catch (error) {
    ElMessage.error(error?.response?.data?.message || '保存策略失败')
  } finally {
    policySaving.value = false
  }
}

const removePolicy = (row) => {
  ElMessageBox.confirm(`确认删除策略「${row.name}」吗？`, '提示', { type: 'warning' }).then(async () => {
    await axios.delete(`/api/v1/jump/policies/${row.id}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    loadPolicies()
  }).catch(() => {})
}

onMounted(async () => {
  await Promise.all([
    loadAssets(),
    loadAccounts(),
    loadPolicies(),
    loadUsersAndRoles()
  ])
})
</script>

<style scoped>
.page-card { margin-bottom: 12px; }
.header { display: flex; justify-content: space-between; align-items: center; gap: 12px; }
.title { font-size: 17px; font-weight: 600; }
.desc { color: #909399; margin-top: 4px; }
.actions { display: flex; gap: 8px; }
.inline-fields { width: 100%; display: grid; grid-template-columns: 1fr 1fr; gap: 8px; }
@media (max-width: 768px) {
  .header { flex-direction: column; align-items: flex-start; }
  .inline-fields { grid-template-columns: 1fr; }
}
</style>
