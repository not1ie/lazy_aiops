<template>
  <el-card>
    <template #header>
      <div class="flex justify-between items-center">
        <span class="font-bold">应用列表</span>
        <el-button type="primary" icon="Plus" @click="openCreate">创建应用</el-button>
      </div>
    </template>

    <div class="grid grid-cols-4 gap-4">
      <el-card v-for="app in list" :key="app.id" shadow="hover" class="cursor-pointer">
        <div class="flex items-center gap-3 mb-3">
          <div class="w-10 h-10 rounded bg-blue-100 flex items-center justify-center text-blue-600 text-xl">
            <i class="fab fa-git"></i>
          </div>
          <div>
            <div class="font-bold text-lg">{{ app.name }}</div>
            <div class="text-xs text-gray-500">{{ app.code }}</div>
          </div>
        </div>
        <div class="text-sm text-gray-600 mb-3 h-10 overflow-hidden">{{ app.description || '暂无描述' }}</div>
        <div class="flex justify-between text-xs text-gray-400 border-t pt-2">
          <span>{{ app.language }}</span>
          <span>{{ app.owner }}</span>
        </div>
      </el-card>
    </div>

    <el-dialog append-to-body v-model="visible" title="创建应用" width="600px" @closed="handleDialogClosed">
      <el-form :model="form" label-width="100px">
        <el-form-item label="应用名称" required>
          <el-input v-model="form.name" placeholder="如: 用户中心" />
        </el-form-item>
        <el-form-item label="唯一标识" required>
          <el-input v-model="form.code" placeholder="如: user-service" />
        </el-form-item>
        <el-form-item label="开发语言">
          <el-select v-model="form.language" class="w-full">
            <el-option label="Java" value="java" />
            <el-option label="Go" value="go" />
            <el-option label="NodeJS" value="nodejs" />
            <el-option label="Python" value="python" />
          </el-select>
        </el-form-item>
        <el-form-item label="Git仓库">
          <el-input v-model="form.git_repo" placeholder="git@github.com:org/repo.git" />
        </el-form-item>
        <el-form-item label="负责人">
          <el-input v-model="form.owner" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="visible = false">取消</el-button>
        <el-button type="primary" @click="submit">确定</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import { getErrorMessage } from '@/utils/error'

const list = ref([])
const visible = ref(false)
const defaultForm = () => ({
  name: '', code: '', language: 'java', git_repo: '', owner: '', description: ''
})
const form = reactive(defaultForm())

const authHeaders = () => ({ Authorization: 'Bearer ' + localStorage.getItem('token') })

const resetForm = () => {
  Object.assign(form, defaultForm())
}

const openCreate = () => {
  resetForm()
  visible.value = true
}

const handleDialogClosed = () => {
  resetForm()
}

const fetchData = async () => {
  try {
    const res = await axios.get('/api/v1/application/apps', {
      headers: authHeaders()
    })
    if (res.data.code === 0) list.value = res.data.data
  } catch (e) {
    ElMessage.error(getErrorMessage(e, '加载应用失败'))
  }
}

const submit = async () => {
  try {
    await axios.post('/api/v1/application/apps', form, {
      headers: authHeaders()
    })
    ElMessage.success('创建成功')
    visible.value = false
    await fetchData()
  } catch (e) {
    ElMessage.error(getErrorMessage(e, '创建应用失败'))
  }
}

onMounted(fetchData)
</script>

<style scoped>
.grid { display: grid; }
.grid-cols-4 { grid-template-columns: repeat(4, minmax(0, 1fr)); }
.gap-4 { gap: 1rem; }
.mb-3 { margin-bottom: 0.75rem; }
.text-lg { font-size: 1.125rem; }
.text-xs { font-size: 0.75rem; }
.text-gray-500 { color: #6b7280; }
.h-10 { height: 2.5rem; }
.overflow-hidden { overflow: hidden; }
.border-t { border-top-width: 1px; }
.pt-2 { padding-top: 0.5rem; }
.w-full { width: 100%; }
</style>
