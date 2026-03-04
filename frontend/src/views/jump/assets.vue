<template>
  <el-card class="page-card">
    <template #header>
      <div class="header">
        <div>
          <div class="title">堡垒机资产</div>
          <div class="desc">统一纳管主机、K8s、Docker资产，供会话与授权复用。</div>
        </div>
        <div class="actions">
          <el-button type="primary" icon="Plus" @click="openCreate">新增资产</el-button>
          <el-button icon="Refresh" @click="fetchAssets">刷新</el-button>
          <el-button type="success" plain @click="syncAll">一键同步</el-button>
        </div>
      </div>
    </template>

    <div class="toolbar">
      <el-input v-model="filters.keyword" clearable placeholder="搜索名称/IP/来源ID" class="filter-item" @change="fetchAssets" />
      <el-select v-model="filters.asset_type" clearable placeholder="资产类型" class="filter-item" @change="fetchAssets">
        <el-option label="主机" value="host" />
        <el-option label="K8s" value="k8s" />
        <el-option label="数据库" value="database" />
      </el-select>
      <el-select v-model="filters.protocol" clearable placeholder="协议" class="filter-item" @change="fetchAssets">
        <el-option v-for="p in protocols" :key="p" :label="p" :value="p" />
      </el-select>
      <el-button @click="syncCMDB">同步CMDB</el-button>
      <el-button @click="syncK8s">同步K8s</el-button>
      <el-button @click="syncDocker">同步Docker</el-button>
    </div>

    <el-table :fit="true" :data="assets" v-loading="loading" stripe>
      <el-table-column prop="name" label="名称" min-width="180" show-overflow-tooltip />
      <el-table-column prop="asset_type" label="类型" width="100" />
      <el-table-column prop="protocol" label="协议" width="110" />
      <el-table-column label="地址" min-width="200">
        <template #default="{ row }">{{ row.address || '-' }}:{{ row.port || '-' }}</template>
      </el-table-column>
      <el-table-column prop="cluster" label="集群" min-width="120" />
      <el-table-column prop="source" label="来源" width="130" />
      <el-table-column label="状态" width="90">
        <template #default="{ row }">
          <el-tag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? '启用' : '禁用' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="190" fixed="right">
        <template #default="{ row }">
          <el-button size="small" type="primary" plain @click="openEdit(row)">编辑</el-button>
          <el-button size="small" type="danger" plain @click="removeAsset(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-card>

  <el-dialog append-to-body v-model="dialogVisible" :title="editing ? '编辑资产' : '新增资产'" width="720px">
    <el-form :model="form" label-width="96px">
      <el-form-item label="名称" required>
        <el-input v-model="form.name" />
      </el-form-item>
      <el-form-item label="资产类型">
        <el-select v-model="form.asset_type" style="width: 100%">
          <el-option label="主机" value="host" />
          <el-option label="K8s" value="k8s" />
          <el-option label="数据库" value="database" />
        </el-select>
      </el-form-item>
      <el-form-item label="协议" required>
        <el-select v-model="form.protocol" style="width: 100%" @change="handleProtocolChange">
          <el-option v-for="p in protocols" :key="p" :label="p" :value="p" />
        </el-select>
      </el-form-item>
      <el-form-item label="地址">
        <el-input v-model="form.address" placeholder="例如 10.0.0.1 / https://api.k8s.local" />
      </el-form-item>
      <el-form-item label="端口">
        <el-input-number v-model="form.port" :min="1" :max="65535" />
      </el-form-item>
      <el-form-item label="集群/命名空间">
        <div class="inline-fields">
          <el-input v-model="form.cluster" placeholder="集群" />
          <el-input v-model="form.namespace" placeholder="命名空间" />
        </div>
      </el-form-item>
      <el-form-item label="来源">
        <div class="inline-fields">
          <el-input v-model="form.source" placeholder="manual/cmdb_host/k8s_cluster/docker_host" />
          <el-input v-model="form.source_ref" placeholder="来源ID" />
        </div>
      </el-form-item>
      <el-form-item label="标签">
        <el-input v-model="form.tags" placeholder="逗号分隔" />
      </el-form-item>
      <el-form-item label="描述">
        <el-input v-model="form.description" type="textarea" :rows="3" />
      </el-form-item>
      <el-form-item label="启用">
        <el-switch v-model="form.enabled" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="dialogVisible = false">取消</el-button>
      <el-button type="primary" :loading="saving" @click="saveAsset">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const loading = ref(false)
const saving = ref(false)
const dialogVisible = ref(false)
const editing = ref(false)
const currentID = ref('')

const protocols = ['ssh', 'docker', 'k8s', 'mysql', 'postgres', 'redis', 'mongodb']

const filters = reactive({
  keyword: '',
  asset_type: '',
  protocol: ''
})

const assets = ref([])

const form = reactive({
  name: '',
  asset_type: 'host',
  protocol: 'ssh',
  address: '',
  port: 22,
  cluster: '',
  namespace: '',
  source: 'manual',
  source_ref: '',
  tags: '',
  description: '',
  enabled: true
})

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const inferType = (protocol) => {
  if (protocol === 'k8s') return 'k8s'
  if (['mysql', 'postgres', 'redis', 'mongodb'].includes(protocol)) return 'database'
  return 'host'
}

const inferPort = (protocol) => {
  const m = {
    ssh: 22,
    docker: 22,
    k8s: 443,
    mysql: 3306,
    postgres: 5432,
    redis: 6379,
    mongodb: 27017
  }
  return m[protocol] || 22
}

const handleProtocolChange = (protocol) => {
  form.asset_type = inferType(protocol)
  form.port = inferPort(protocol)
}

const fetchAssets = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/v1/jump/assets', {
      headers: authHeaders(),
      params: {
        keyword: filters.keyword || undefined,
        asset_type: filters.asset_type || undefined,
        protocol: filters.protocol || undefined
      }
    })
    if (res.data.code === 0) {
      assets.value = res.data.data || []
    }
  } catch (error) {
    ElMessage.error(error?.response?.data?.message || '加载资产失败')
  } finally {
    loading.value = false
  }
}

const resetForm = () => {
  Object.assign(form, {
    name: '',
    asset_type: 'host',
    protocol: 'ssh',
    address: '',
    port: 22,
    cluster: '',
    namespace: '',
    source: 'manual',
    source_ref: '',
    tags: '',
    description: '',
    enabled: true
  })
}

const openCreate = () => {
  editing.value = false
  currentID.value = ''
  resetForm()
  dialogVisible.value = true
}

const openEdit = (row) => {
  editing.value = true
  currentID.value = row.id
  Object.assign(form, {
    name: row.name || '',
    asset_type: row.asset_type || 'host',
    protocol: row.protocol || 'ssh',
    address: row.address || '',
    port: row.port || inferPort(row.protocol),
    cluster: row.cluster || '',
    namespace: row.namespace || '',
    source: row.source || 'manual',
    source_ref: row.source_ref || '',
    tags: row.tags || '',
    description: row.description || '',
    enabled: row.enabled !== false
  })
  dialogVisible.value = true
}

const saveAsset = async () => {
  if (!form.name || !form.protocol) {
    ElMessage.warning('请填写名称和协议')
    return
  }
  saving.value = true
  try {
    const payload = { ...form }
    let res
    if (editing.value) {
      res = await axios.put(`/api/v1/jump/assets/${currentID.value}`, payload, { headers: authHeaders() })
    } else {
      res = await axios.post('/api/v1/jump/assets', payload, { headers: authHeaders() })
    }
    if (res.data.code === 0) {
      ElMessage.success(editing.value ? '更新成功' : '创建成功')
      dialogVisible.value = false
      fetchAssets()
    }
  } catch (error) {
    ElMessage.error(error?.response?.data?.message || '保存失败')
  } finally {
    saving.value = false
  }
}

const removeAsset = (row) => {
  ElMessageBox.confirm(`确认删除资产「${row.name}」吗？`, '提示', { type: 'warning' }).then(async () => {
    await axios.delete(`/api/v1/jump/assets/${row.id}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    fetchAssets()
  }).catch(() => {})
}

const runSync = async (url, okText = '同步成功') => {
  try {
    const res = await axios.post(url, {}, { headers: authHeaders() })
    if (res.data.code === 0) {
      ElMessage.success(res.data.message || okText)
      fetchAssets()
    }
  } catch (error) {
    ElMessage.error(error?.response?.data?.message || '同步失败')
  }
}

const syncCMDB = () => runSync('/api/v1/jump/sync/cmdb-hosts')
const syncK8s = () => runSync('/api/v1/jump/sync/k8s-clusters')
const syncDocker = () => runSync('/api/v1/jump/sync/docker-hosts')
const syncAll = () => runSync('/api/v1/jump/sync/all', '同步完成')

onMounted(fetchAssets)
</script>

<style scoped>
.page-card { max-width: 100%; }
.header { display: flex; justify-content: space-between; align-items: center; gap: 12px; }
.title { font-size: 18px; font-weight: 600; }
.desc { color: #909399; margin-top: 4px; }
.actions { display: flex; gap: 8px; }
.toolbar { margin-bottom: 12px; display: flex; gap: 8px; flex-wrap: wrap; }
.filter-item { width: 220px; }
.inline-fields { display: grid; gap: 8px; width: 100%; grid-template-columns: 1fr 1fr; }
@media (max-width: 768px) {
  .header { flex-direction: column; align-items: flex-start; }
  .filter-item { width: 100%; }
  .inline-fields { grid-template-columns: 1fr; }
}
</style>
