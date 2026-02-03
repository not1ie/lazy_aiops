<template>
  <el-card>
    <template #header>
      <div class="flex justify-between items-center">
        <span class="font-bold">Docker 环境列表</span>
        <div>
          <el-button type="primary" icon="Plus" @click="handleAdd">添加环境</el-button>
          <el-button icon="Refresh" @click="fetchData">刷新</el-button>
        </div>
      </div>
    </template>

    <el-table :data="tableData" v-loading="loading" style="width: 100%">
      <el-table-column prop="name" label="名称" width="180">
        <template #default="{ row }">
          <div class="flex items-center gap-2">
            <el-icon class="text-blue-500 text-xl"><Platform /></el-icon>
            <span class="font-bold">{{ row.name }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="status" label="状态" width="120">
        <template #default="{ row }">
          <el-tag :type="row.status === 'online' ? 'success' : 'danger'">
            {{ row.status }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="container_count" label="容器数" width="120" align="center" />
      <el-table-column prop="image_count" label="镜像数" width="120" align="center" />
      <el-table-column prop="version" label="版本" />
      <el-table-column label="操作" width="250" fixed="right">
        <template #default="{ row }">
          <el-button size="small" type="primary" plain icon="Monitor" @click="handleManage(row)">管理</el-button>
          <el-button size="small" type="warning" plain icon="FirstAidKit" @click="handleDiagnose(row)">诊断</el-button>
          <el-button size="small" type="danger" plain icon="Delete" @click="handleDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 添加主机弹窗 -->
    <el-dialog v-model="dialogVisible" title="添加 Docker 环境" width="500px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="名称">
          <el-input v-model="form.name" placeholder="例如: Local Docker" />
        </el-form-item>
        <el-form-item label="关联主机">
          <el-select v-model="form.host_id" placeholder="请选择" class="w-100">
            <el-option label="本机 (Local Socket)" value="local" />
            <el-option v-for="h in hosts" :key="h.id" :label="h.name + ' (' + h.ip + ')'" :value="h.id" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitForm" :loading="submitting">确定</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const loading = ref(false)
const tableData = ref([])
const dialogVisible = ref(false)
const submitting = ref(false)
const hosts = ref([]) // CMDB hosts

const form = reactive({
  name: '',
  host_id: ''
})

const fetchData = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/v1/docker/hosts', {
      headers: { Authorization: 'Bearer ' + localStorage.getItem('token') }
    })
    if (res.data.code === 0) {
      tableData.value = res.data.data
    }
  } catch (e) {
    ElMessage.error('加载失败')
  } finally {
    loading.value = false
  }
}

const fetchCMDBHosts = async () => {
  try {
    const res = await axios.get('/api/v1/cmdb/hosts', {
      headers: { Authorization: 'Bearer ' + localStorage.getItem('token') }
    })
    if (res.data.code === 0) {
      hosts.value = res.data.data
    }
  } catch (e) {}
}

const handleAdd = () => {
  fetchCMDBHosts()
  form.name = ''
  form.host_id = ''
  dialogVisible.value = true
}

const submitForm = async () => {
  submitting.value = true
  try {
    const res = await axios.post('/api/v1/docker/hosts', form, {
      headers: { Authorization: 'Bearer ' + localStorage.getItem('token') }
    })
    if (res.data.code === 0) {
      ElMessage.success('添加成功')
      dialogVisible.value = false
      fetchData()
    } else {
      ElMessage.error(res.data.message)
    }
  } finally {
    submitting.value = false
  }
}

const handleDelete = (row) => {
  ElMessageBox.confirm('确定删除该 Docker 环境吗?', '警告', {
    confirmButtonText: '删除',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    await axios.delete(`/api/v1/docker/hosts/${row.id}`, {
      headers: { Authorization: 'Bearer ' + localStorage.getItem('token') }
    })
    ElMessage.success('删除成功')
    fetchData()
  })
}

const handleManage = (row) => {
  // TODO: 跳转到容器列表页 (下一阶段实现)
  ElMessage.info('进入管理界面: ' + row.name)
}

const handleDiagnose = (row) => {
  ElMessage.info('开始诊断: ' + row.name)
  // TODO: 调用诊断API并展示弹窗
}

onMounted(() => {
  fetchData()
})
</script>

<style scoped>
.flex { display: flex; }
.justify-between { justify-content: space-between; }
.items-center { align-items: center; }
.gap-2 { gap: 8px; }
.font-bold { font-weight: bold; }
.w-100 { width: 100%; }
</style>
