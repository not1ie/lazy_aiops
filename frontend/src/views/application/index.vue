<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>应用中心</h2>
        <p class="page-desc">管理 DevOps 核心服务，关联代码仓库、构建工具与 K8s 部署环境。</p>
      </div>
      <div class="page-actions">
        <el-button type="primary" icon="Plus" @click="openCreate">创建应用</el-button>
        <el-button icon="Refresh" @click="fetchData">刷新</el-button>
      </div>
    </div>

    <div v-if="list.length === 0" class="empty-cta">
      <el-empty description="暂无应用">
        <el-button type="primary" @click="openCreate">创建第一个应用</el-button>
      </el-empty>
    </div>

    <div v-else class="app-grid">
      <el-card v-for="app in list" :key="app.id" shadow="hover" class="app-card">
        <div class="app-card-header">
          <div class="app-icon" :style="{ background: langColor(app.language) }">
            {{ langIcon(app.language) }}
          </div>
          <div class="app-info">
            <div class="app-name">{{ app.name }}</div>
            <div class="app-code">{{ app.code }}</div>
          </div>
          <el-dropdown trigger="click" @command="(cmd) => handleCardAction(cmd, app)">
            <el-button link icon="MoreFilled" />
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="edit">编辑</el-dropdown-item>
                <el-dropdown-item command="config">环境配置</el-dropdown-item>
                <el-dropdown-item command="delete" divided>删除</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
        <div class="app-desc">{{ app.description || '暂无描述' }}</div>
        <div class="app-meta">
          <el-tag size="small" effect="plain" type="info">{{ app.language || '-' }}</el-tag>
          <el-tag v-if="app.build_tool" size="small" effect="plain">{{ app.build_tool }}</el-tag>
          <span class="app-owner">{{ app.owner || '-' }}</span>
        </div>
        <div class="app-footer" v-if="app.git_repo">
          <el-icon><Share /></el-icon>
          <span class="git-url">{{ app.git_repo }}</span>
        </div>
      </el-card>
    </div>

    <el-dialog append-to-body v-model="visible" :title="editingApp ? '编辑应用' : '创建应用'" width="560px" @closed="handleDialogClosed">
      <el-form :model="form" label-width="90px">
        <el-form-item label="应用名称" required>
          <el-input v-model="form.name" placeholder="如: 用户中心" />
        </el-form-item>
        <el-form-item label="唯一标识" required>
          <el-input v-model="form.code" placeholder="如: user-service" :disabled="!!editingApp" />
        </el-form-item>
        <el-form-item label="开发语言">
          <el-select v-model="form.language" style="width: 100%">
            <el-option label="Java" value="java" />
            <el-option label="Go" value="go" />
            <el-option label="NodeJS" value="nodejs" />
            <el-option label="Python" value="python" />
            <el-option label="Rust" value="rust" />
          </el-select>
        </el-form-item>
        <el-form-item label="构建工具">
          <el-select v-model="form.build_tool" style="width: 100%" clearable>
            <el-option label="Maven" value="maven" />
            <el-option label="Gradle" value="gradle" />
            <el-option label="Go Build" value="go build" />
            <el-option label="npm" value="npm" />
            <el-option label="pip" value="pip" />
            <el-option label="Cargo" value="cargo" />
          </el-select>
        </el-form-item>
        <el-form-item label="Git 仓库">
          <el-input v-model="form.git_repo" placeholder="git@github.com:org/repo.git" />
        </el-form-item>
        <el-form-item label="负责人">
          <el-input v-model="form.owner" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="visible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submit">保存</el-button>
      </template>
    </el-dialog>

    <!-- Environment Config Dialog -->
    <el-dialog :title="'环境配置: ' + configAppName" v-model="envDialogVisible" width="640px" append-to-body>
      <el-table :data="envConfigs" size="small" v-if="envConfigs.length > 0">
        <el-table-column prop="env_name" label="环境" width="80" />
        <el-table-column prop="namespace" label="命名空间" width="120" />
        <el-table-column label="副本数" width="70">
          <template #default="{ row }">{{ row.replicas || 1 }}</template>
        </el-table-column>
        <el-table-column label="CPU" width="90">
          <template #default="{ row }">{{ row.cpu_quota || '-' }}</template>
        </el-table-column>
        <el-table-column label="内存" width="90">
          <template #default="{ row }">{{ row.mem_quota || '-' }}</template>
        </el-table-column>
        <el-table-column label="健康检查" min-width="150">
          <template #default="{ row }">{{ row.health_check || '-' }}</template>
        </el-table-column>
      </el-table>
      <el-empty v-else description="暂无环境配置" :image-size="40" />

      <el-divider />

      <el-form :model="envForm" label-width="90px" inline>
        <el-form-item label="环境">
          <el-select v-model="envForm.env_name" style="width:100px">
            <el-option label="dev" value="dev" />
            <el-option label="test" value="test" />
            <el-option label="staging" value="staging" />
            <el-option label="prod" value="prod" />
          </el-select>
        </el-form-item>
        <el-form-item label="K8s 集群">
          <el-select v-model="envForm.cluster_id" style="width:140px" clearable>
            <el-option v-for="c in k8sClusters" :key="c.id" :label="c.name" :value="c.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="命名空间">
          <el-input v-model="envForm.namespace" style="width:120px" placeholder="default" />
        </el-form-item>
        <el-form-item label="副本数">
          <el-input-number v-model="envForm.replicas" :min="1" :max="99" style="width:80px" />
        </el-form-item>
        <el-form-item label="CPU">
          <el-input v-model="envForm.cpu_quota" style="width:100px" placeholder="500m" />
        </el-form-item>
        <el-form-item label="内存">
          <el-input v-model="envForm.mem_quota" style="width:100px" placeholder="512Mi" />
        </el-form-item>
        <el-form-item label="健康检查">
          <el-input v-model="envForm.health_check" style="width:160px" placeholder="/health" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="envSaving" @click="submitEnvConfig">添加</el-button>
        </el-form-item>
      </el-form>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const list = ref([])
const visible = ref(false)
const saving = ref(false)
const editingApp = ref(null)

const defaultForm = () => ({
  name: '', code: '', language: 'java', build_tool: '', git_repo: '', owner: '', description: ''
})
const form = reactive(defaultForm())

const authHeaders = () => ({ Authorization: 'Bearer ' + localStorage.getItem('token') })

const langIcon = (lang) => {
  const icons = { go: 'Go', java: 'Jv', nodejs: 'JS', python: 'Py', rust: 'Rs' }
  return icons[lang] || lang?.charAt(0)?.toUpperCase() || '?'
}
const langColor = (lang) => {
  const colors = { go: '#00ADD8', java: '#ED8B00', nodejs: '#339933', python: '#3776AB', rust: '#DEA584' }
  return colors[lang] || '#6b7280'
}

const handleCardAction = (cmd, app) => {
  if (cmd === 'edit') {
    editingApp.value = app
    Object.assign(form, {
      name: app.name, code: app.code, language: app.language || 'java',
      build_tool: app.build_tool || '', git_repo: app.git_repo || '',
      owner: app.owner || '', description: app.description || ''
    })
    visible.value = true
  } else if (cmd === 'config') {
    openEnvConfig(app)
  } else if (cmd === 'delete') {
    handleDelete(app)
  }
}

// --- Environment Config ---
const envDialogVisible = ref(false)
const envSaving = ref(false)
const configAppId = ref('')
const configAppName = ref('')
const envConfigs = ref([])
const k8sClusters = ref([])
const envForm = ref({ env_name: 'dev', cluster_id: '', namespace: 'default', replicas: 1, cpu_quota: '', mem_quota: '', health_check: '' })

const openEnvConfig = async (app) => {
  configAppId.value = app.id
  configAppName.value = app.name
  envForm.value = { env_name: 'dev', cluster_id: '', namespace: 'default', replicas: 1, cpu_quota: '', mem_quota: '', health_check: '' }
  envDialogVisible.value = true
  try {
    const [cfgRes, k8sRes] = await Promise.all([
      axios.get(`/api/v1/application/apps/${app.id}/configs`, { headers: authHeaders() }),
      axios.get('/api/v1/k8s/clusters', { headers: authHeaders() }).catch(() => ({ data: {} }))
    ])
    envConfigs.value = cfgRes.data?.data || []
    k8sClusters.value = k8sRes.data?.data || []
  } catch (e) { /* silent */ }
}

const submitEnvConfig = async () => {
  if (!configAppId.value) return
  envSaving.value = true
  try {
    await axios.post(`/api/v1/application/apps/${configAppId.value}/configs`, envForm.value, { headers: authHeaders() })
    ElMessage.success('环境配置已添加')
    // Refresh env list
    const cfgRes = await axios.get(`/api/v1/application/apps/${configAppId.value}/configs`, { headers: authHeaders() })
    envConfigs.value = cfgRes.data?.data || []
  } catch (err) {
    ElMessage.error('添加失败: ' + (err.response?.data?.message || err.message))
  } finally {
    envSaving.value = false
  }
}

const openCreate = () => {
  editingApp.value = null
  Object.assign(form, defaultForm())
  visible.value = true
}

const handleDialogClosed = () => {
  editingApp.value = null
  Object.assign(form, defaultForm())
}

const handleDelete = async (app) => {
  try {
    await ElMessageBox.confirm(`确认删除应用「${app.name}」？`, '提示', { type: 'warning' })
    await axios.delete(`/api/v1/application/apps/${app.id}`, { headers: authHeaders() })
    ElMessage.success('已删除')
    fetchData()
  } catch (e) { /* cancel */ }
}

const fetchData = async () => {
  try {
    const res = await axios.get('/api/v1/application/apps', { headers: authHeaders() })
    list.value = res.data?.data || []
  } catch (e) {
    ElMessage.error('加载应用失败')
  }
}

const submit = async () => {
  if (!form.name || !form.code) return ElMessage.warning('名称和标识为必填')
  saving.value = true
  try {
    if (editingApp.value) {
      await axios.put(`/api/v1/application/apps/${editingApp.value.id}`, form, { headers: authHeaders() })
      ElMessage.success('更新成功')
    } else {
      await axios.post('/api/v1/application/apps', form, { headers: authHeaders() })
      ElMessage.success('创建成功')
    }
    visible.value = false
    fetchData()
  } catch (e) {
    ElMessage.error((e.response?.data?.message) || '保存失败')
  } finally {
    saving.value = false
  }
}

onMounted(fetchData)
</script>

<style scoped>
.page-card { max-width: 100%; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 20px; }
.page-desc { color: var(--el-text-color-secondary); margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; }
.empty-cta { padding: 60px 0; }

.app-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 16px; }
.app-card { cursor: default; }
.app-card-header { display: flex; align-items: flex-start; gap: 12px; margin-bottom: 12px; }
.app-icon {
  width: 40px; height: 40px; border-radius: 10px;
  display: flex; align-items: center; justify-content: center;
  color: #fff; font-weight: 800; font-size: 14px; flex-shrink: 0;
}
.app-info { flex: 1; min-width: 0; }
.app-name { font-size: 16px; font-weight: 700; color: var(--el-text-color-primary); }
.app-code { font-size: 12px; color: var(--el-text-color-secondary); font-family: monospace; }
.app-desc { font-size: 13px; color: var(--el-text-color-regular); margin-bottom: 12px; min-height: 36px; }
.app-meta { display: flex; align-items: center; gap: 6px; margin-bottom: 8px; }
.app-owner { font-size: 12px; color: var(--el-text-color-secondary); margin-left: auto; }
.app-footer { display: flex; align-items: center; gap: 6px; font-size: 12px; color: var(--el-text-color-secondary); }
.git-url { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
</style>
