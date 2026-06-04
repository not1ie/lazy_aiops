<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>镜像仓库管理</h2>
        <p class="page-desc">统一管理容器镜像分发中心，支持 Harbor 与标准 Docker Registry 资产浏览与版本查询。</p>
      </div>
      <div class="page-actions">
        <el-button type="primary" icon="Plus" @click="handleAddRegistry">新增仓库</el-button>
        <el-button icon="Refresh" @click="getRegistryList">刷新列表</el-button>
      </div>
    </div>

    <div class="page-body mt-20">
      <el-row :gutter="20">
        <!-- 左侧：仓库列表 -->
        <el-col :xl="7" :lg="8">
          <el-card shadow="never" header="资产目录">
            <el-table
              v-loading="registryLoading"
              :data="registryList"
              @row-click="handleRegistryClick"
              highlight-current-row
              border
            >
              <el-table-column prop="name" label="名称" show-overflow-tooltip />
              <el-table-column prop="provider" label="类型" width="90">
                <template #default="{ row }">
                  <el-tag size="small" :type="row.provider === 'harbor' ? 'primary' : 'info'">{{ row.provider }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="操作" width="110" align="center">
                <template #default="{ row }">
                  <el-button type="primary" link @click.stop="handleEditRegistry(row)">编辑</el-button>
                  <el-button type="danger" link @click.stop="handleDeleteRegistry(row)">删除</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-card>
        </el-col>

        <!-- 右侧：镜像查询 -->
        <el-col :xl="17" :lg="16">
          <el-card shadow="never">
            <template #header>
              <div class="card-header">
                <span>镜像版本查询 <span v-if="currentRegistry" class="current-name">/ {{ currentRegistry.name }}</span></span>
              </div>
            </template>
            
            <div v-if="currentRegistry" class="search-bar">
              <el-input
                v-model="searchRepo"
                placeholder="输入完整镜像名称，例如：library/nginx"
                style="width: 360px;"
                clearable
                @keyup.enter="handleSearchTags"
              >
                <template #prefix><el-icon><Search /></el-icon></template>
              </el-input>
              <el-button type="primary" @click="handleSearchTags" :loading="tagsLoading">查询 Tags</el-button>
            </div>
            
            <div class="tags-table-wrap">
              <el-empty v-if="!currentRegistry" description="请先在左侧选择镜像仓库资产" :image-size="120" />

              <el-table v-if="currentRegistry" v-loading="tagsLoading" :data="tagsList" border stripe>
                <el-table-column prop="name" label="Tag 版本" width="180">
                  <template #default="{ row }">
                    <span class="tag-name">{{ row.name }}</span>
                  </template>
                </el-table-column>
                <el-table-column prop="created" label="推送时间" width="200">
                  <template #default="{ row }">{{ new Date(row.created).toLocaleString() }}</template>
                </el-table-column>
                <el-table-column prop="digest" label="制品哈希 (Digest)" show-overflow-tooltip />
              </el-table>
            </div>
          </el-card>
        </el-col>
      </el-row>
    </div>

    <!-- 仓库表单 -->
    <el-dialog :title="regForm.id ? '编辑仓库资产' : '录入仓库资产'" v-model="regDialogVisible" width="550px" append-to-body>
      <el-form ref="regFormRef" :model="regForm" :rules="regRules" label-width="110px">
        <el-form-item label="显示名称" prop="name">
          <el-input v-model="regForm.name" placeholder="例如：上海中心私有 Harbor" />
        </el-form-item>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="仓库类型" prop="provider">
              <el-select v-model="regForm.provider" style="width: 100%;">
                <el-option label="Harbor" value="harbor" />
                <el-option label="Docker Registry V2" value="docker-registry" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
             <el-form-item label="设为默认">
               <el-switch v-model="regForm.is_default" />
             </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="仓库 API 地址" prop="url">
          <el-input v-model="regForm.url" placeholder="https://harbor.example.com" />
        </el-form-item>
        <el-form-item label="认证用户名">
          <el-input v-model="regForm.username" placeholder="留空则使用匿名访问" />
        </el-form-item>
        <el-form-item label="访问凭据/密码">
          <el-input type="password" v-model="regForm.password" placeholder="不修改请留空" show-password />
        </el-form-item>
        <el-form-item label="描述说明">
          <el-input type="textarea" v-model="regForm.description" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="regDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitRegForm">保存资产</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<style scoped>
.page-body {
  margin-top: 24px;
}
.current-name {
  color: var(--el-color-primary);
  font-weight: 600;
  margin-left: 10px;
  background: var(--el-color-primary-light-8);
  padding: 2px 10px;
  border-radius: 6px;
  font-size: 13px;
}
.search-bar {
  margin-bottom: 28px;
  display: flex;
  gap: 12px;
  padding: 4px;
}
.tags-table-wrap {
  min-height: 400px;
}
.tag-name {
  font-family: "SF Mono", Menlo, monospace;
  font-weight: 600;
  color: var(--el-color-primary);
  font-size: 13px;
}
.mt-20 {
  margin-top: 20px;
}
:deep(.el-card) {
  border-color: var(--el-border-color-lighter);
}
:deep(.el-table) {
  border: none;
}
</style>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import axios from 'axios'

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

// 仓库管理
const registryLoading = ref(false)
const registryList = ref([])
const regDialogVisible = ref(false)
const regFormRef = ref(null)
const regForm = ref({})
const currentRegistry = ref(null)

// Tags 查询
const searchRepo = ref('')
const tagsLoading = ref(false)
const tagsList = ref([])

const regRules = {
  name: [{ required: true, message: '必填', trigger: 'blur' }],
  provider: [{ required: true, message: '必填', trigger: 'change' }],
  url: [{ required: true, message: '必填', trigger: 'blur' }]
}

const getRegistryList = async () => {
  registryLoading.value = true
  try {
    const res = await axios.get('/api/v1/cicd/registries', { headers: authHeaders() })
    registryList.value = res.data?.data || res.data || []
  } finally {
    registryLoading.value = false
  }
}

const handleAddRegistry = () => {
  regForm.value = { provider: 'harbor' }
  regDialogVisible.value = true
}

const handleEditRegistry = (row) => {
  regForm.value = { ...row, password: '' }
  regDialogVisible.value = true
}

const handleDeleteRegistry = (row) => {
  ElMessageBox.confirm('确认删除仓库?', '提示', { type: 'warning' }).then(async () => {
    await axios.delete(`/api/v1/cicd/registries/${row.id}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    if (currentRegistry.value?.id === row.id) {
      currentRegistry.value = null
      tagsList.value = []
    }
    getRegistryList()
  })
}

const submitRegForm = async () => {
  if (!regFormRef.value) return
  await regFormRef.value.validate()

  const method = regForm.value.id ? 'put' : 'post'
  const url = regForm.value.id ? `/api/v1/cicd/registries/${regForm.value.id}` : '/api/v1/cicd/registries'
  await axios[method](url, regForm.value, { headers: authHeaders() })
  ElMessage.success('保存成功')
  regDialogVisible.value = false
  getRegistryList()
}

const handleRegistryClick = (row) => {
  currentRegistry.value = row
  tagsList.value = []
}

const handleSearchTags = async () => {
  if (!searchRepo.value) return ElMessage.warning('请输入镜像名称')
  tagsLoading.value = true
  try {
    const res = await axios.get(`/api/v1/cicd/registries/${currentRegistry.value.id}/tags`, {
      params: { repository: searchRepo.value },
      headers: authHeaders()
    })
    tagsList.value = res.data?.data || res.data || []
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '获取镜像版本失败')
  } finally {
    tagsLoading.value = false
  }
}

onMounted(() => {
  getRegistryList()
})
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>e>e>