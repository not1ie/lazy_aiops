<template>
  <el-card>
    <template #header>
      <div class="flex justify-between items-center">
        <span class="font-bold">Docker 环境列表</span>
        <div>
          <el-button type="primary" icon="Plus" @click="handleAdd">添加环境</el-button>
          <el-button icon="Refresh" @click="syncAll">刷新</el-button>
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
            {{ row.status || 'unknown' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="container_count" label="容器数" width="120" align="center" />
      <el-table-column prop="image_count" label="镜像数" width="120" align="center" />
      <el-table-column prop="version" label="版本" />
      <el-table-column label="操作" width="260" fixed="right">
        <template #default="{ row }">
          <el-space size="8">
            <el-button size="small" type="primary" plain icon="Monitor" @click="handleManage(row)">管理</el-button>
            <el-button size="small" type="warning" plain icon="FirstAidKit" @click="handleDiagnose(row)">诊断</el-button>
            <el-button size="small" type="danger" plain icon="Delete" @click="handleDelete(row)">删除</el-button>
          </el-space>
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

    <!-- 管理抽屉 -->
    <el-drawer v-model="manageVisible" size="70%" :with-header="false">
      <div class="drawer-header">
        <div>
          <div class="drawer-title">{{ activeHost?.name || 'Docker 环境' }}</div>
          <div class="drawer-sub">
            状态：<el-tag size="small" :type="activeHost?.status === 'online' ? 'success' : 'danger'">{{ activeHost?.status || 'unknown' }}</el-tag>
            <span class="drawer-meta">容器：{{ activeHost?.container_count ?? '-' }}</span>
            <span class="drawer-meta">镜像：{{ activeHost?.image_count ?? '-' }}</span>
            <span class="drawer-meta">版本：{{ activeHost?.version || '-' }}</span>
          </div>
        </div>
        <div>
          <el-button size="small" icon="Refresh" @click="refreshManage">刷新</el-button>
        </div>
      </div>

      <el-tabs v-model="manageTab" class="manage-tabs">
        <el-tab-pane label="概览" name="overview">
          <el-descriptions :column="2" border>
            <el-descriptions-item label="名称">{{ activeHost?.name || '-' }}</el-descriptions-item>
            <el-descriptions-item label="状态">{{ activeHost?.status || 'unknown' }}</el-descriptions-item>
            <el-descriptions-item label="容器数">{{ activeHost?.container_count ?? '-' }}</el-descriptions-item>
            <el-descriptions-item label="镜像数">{{ activeHost?.image_count ?? '-' }}</el-descriptions-item>
            <el-descriptions-item label="版本">{{ activeHost?.version || '-' }}</el-descriptions-item>
            <el-descriptions-item label="主机ID">{{ activeHost?.host_id || '-' }}</el-descriptions-item>
          </el-descriptions>
        </el-tab-pane>

        <el-tab-pane label="容器" name="containers">
          <div class="tab-toolbar">
            <el-button type="primary" icon="Plus" @click="openCreateContainer">创建容器</el-button>
            <el-button icon="Refresh" @click="loadContainers">刷新</el-button>
          </div>
          <el-table :data="containers" v-loading="containersLoading" style="width: 100%">
            <el-table-column prop="names" label="名称" min-width="200" />
            <el-table-column prop="image" label="镜像" min-width="180" />
            <el-table-column prop="state" label="状态" width="120" />
            <el-table-column prop="status" label="详情" min-width="180" />
            <el-table-column prop="created" label="创建时间" width="160" />
            <el-table-column label="操作" width="470" fixed="right">
              <template #default="{ row }">
                <el-space size="8">
                  <el-button size="small" @click="openLogs(row)">日志</el-button>
                  <el-button size="small" type="primary" plain @click="openInspect(row)">详情</el-button>
                  <el-button size="small" type="info" plain @click="openExec(row)">执行命令</el-button>
                  <el-button size="small" type="success" plain @click="containerAction(row, 'start')">启动</el-button>
                  <el-button size="small" type="warning" plain @click="containerAction(row, 'stop')">停止</el-button>
                  <el-button size="small" type="primary" plain @click="containerAction(row, 'restart')">重启</el-button>
                  <el-button size="small" type="danger" plain @click="containerAction(row, 'remove')">删除</el-button>
                </el-space>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="镜像" name="images">
          <div class="tab-toolbar">
            <el-button type="danger" plain :disabled="selectedImages.length === 0" @click="removeSelectedImages">
              批量删除
            </el-button>
            <el-button type="primary" icon="Download" @click="openPullImage">拉取镜像</el-button>
            <el-button icon="Refresh" @click="loadImages">刷新</el-button>
          </div>
          <el-table
            ref="imageTableRef"
            :data="images"
            v-loading="imagesLoading"
            style="width: 100%"
            :row-key="row => row.id"
            @selection-change="onImageSelectionChange"
          >
            <el-table-column type="selection" width="48" />
            <el-table-column prop="repository" label="仓库" min-width="200" />
            <el-table-column prop="tag" label="Tag" width="120" />
            <el-table-column prop="id" label="ID" min-width="180" />
            <el-table-column prop="size" label="大小" width="120" />
            <el-table-column prop="created" label="创建时间" min-width="180" />
            <el-table-column label="操作" width="120" fixed="right">
              <template #default="{ row }">
                <el-button size="small" type="danger" plain @click="removeImage(row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="网络" name="networks">
          <div class="tab-toolbar">
            <el-button icon="Refresh" @click="loadNetworks">刷新</el-button>
          </div>
          <el-table :data="networks" v-loading="networksLoading" style="width: 100%">
            <el-table-column prop="name" label="名称" min-width="180" />
            <el-table-column prop="id" label="ID" min-width="200" />
            <el-table-column prop="driver" label="驱动" width="120" />
            <el-table-column prop="scope" label="范围" width="120" />
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="Services" name="services">
          <div class="tab-toolbar">
            <el-select v-model="serviceStackFilter" placeholder="Stack" class="w-40" clearable>
              <el-option v-for="s in serviceStacks" :key="s" :label="s" :value="s" />
            </el-select>
            <el-button icon="Refresh" @click="loadServices">刷新</el-button>
          </div>
          <el-table :data="filteredServices" v-loading="servicesLoading" style="width: 100%">
            <el-table-column prop="Name" label="名称" min-width="200" />
            <el-table-column prop="Mode" label="模式" width="120" />
            <el-table-column prop="Replicas" label="副本" width="120" />
            <el-table-column prop="Image" label="镜像" min-width="180" />
            <el-table-column prop="Ports" label="端口" min-width="160" />
            <el-table-column label="操作" width="360" fixed="right">
              <template #default="{ row }">
                <el-space size="8">
                  <el-button size="small" @click="openServiceDetail(row)">详情</el-button>
                  <el-button size="small" type="info" plain @click="openServiceTasks(row)">任务</el-button>
                  <el-button size="small" type="primary" plain @click="scaleService(row)">扩缩容</el-button>
                  <el-button size="small" type="warning" plain @click="updateServiceImage(row)">更新镜像</el-button>
                  <el-button size="small" type="danger" plain @click="restartService(row)">重启</el-button>
                </el-space>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="Stacks" name="stacks">
          <div class="tab-toolbar">
            <el-button icon="Refresh" @click="loadStacks">刷新</el-button>
          </div>
          <el-table :data="stacks" v-loading="stacksLoading" style="width: 100%">
            <el-table-column prop="Name" label="名称" min-width="200" />
            <el-table-column prop="Services" label="服务数" width="120" />
            <el-table-column prop="Orchestrator" label="编排" width="160" />
            <el-table-column label="操作" width="140" fixed="right">
              <template #default="{ row }">
                <el-button size="small" @click="openStackServices(row)">查看服务</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
      </el-tabs>
    </el-drawer>

    <!-- 创建容器弹窗 -->
    <el-dialog v-model="createVisible" title="创建容器" width="640px">
      <el-form :model="createForm" label-width="100px">
        <el-form-item label="镜像" required>
          <el-input v-model="createForm.image" placeholder="例如 nginx:latest" />
        </el-form-item>
        <el-form-item label="容器名">
          <el-input v-model="createForm.name" placeholder="可选" />
        </el-form-item>
        <el-form-item label="端口映射">
          <el-input v-model="createForm.ports" placeholder="如 8080:80, 8443:443" />
        </el-form-item>
        <el-form-item label="环境变量">
          <el-input v-model="createForm.env" type="textarea" :rows="4" placeholder="KEY=VALUE，每行一个" />
        </el-form-item>
        <el-form-item label="重启策略">
          <el-select v-model="createForm.restart_policy" placeholder="不设置">
            <el-option label="不设置" value="" />
            <el-option label="always" value="always" />
            <el-option label="unless-stopped" value="unless-stopped" />
            <el-option label="on-failure" value="on-failure" />
          </el-select>
        </el-form-item>
        <el-form-item label="自动删除">
          <el-switch v-model="createForm.auto_remove" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" :loading="createLoading" @click="submitCreate">创建</el-button>
      </template>
    </el-dialog>

    <!-- 拉取镜像弹窗 -->
    <el-dialog v-model="pullVisible" title="拉取镜像" width="640px">
      <el-form label-width="100px">
        <el-form-item label="镜像" required>
          <el-input v-model="pullImage" placeholder="例如 redis:7" />
        </el-form-item>
      </el-form>
      <el-input v-model="pullOutput" type="textarea" :rows="8" readonly placeholder="输出" />
      <template #footer>
        <el-button @click="pullVisible = false">关闭</el-button>
        <el-button type="primary" :loading="pullLoading" @click="submitPull">拉取</el-button>
      </template>
    </el-dialog>

    <!-- 诊断弹窗 -->
    <el-dialog v-model="diagnoseVisible" title="Docker 诊断" width="720px">
      <el-alert v-if="diagnoseError" type="error" :closable="false" show-icon>{{ diagnoseError }}</el-alert>
      <el-skeleton v-if="diagnoseLoading" :rows="6" animated />
      <div v-else class="diagnose-block">
        <div class="diagnose-title">Step1: docker info</div>
        <pre class="diagnose-pre">{{ diagnoseResult?.step1_info?.out || '-' }}</pre>
        <div class="diagnose-title">Step2: docker system info (json)</div>
        <pre class="diagnose-pre">{{ diagnoseResult?.step2_sync?.out || '-' }}</pre>
        <div class="diagnose-title">Step3: docker ps -a</div>
        <pre class="diagnose-pre">{{ diagnoseResult?.step3_list?.out || '-' }}</pre>
      </div>
      <template #footer>
        <el-button @click="diagnoseVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- 容器日志弹窗 -->
    <el-dialog v-model="logVisible" title="容器日志" width="720px">
      <div class="log-controls">
        <el-input v-model="logTail" placeholder="tail" style="width: 120px" />
        <el-button icon="Refresh" @click="loadLogs" :loading="logLoading">刷新</el-button>
      </div>
      <el-input v-model="logText" type="textarea" :rows="16" readonly />
    </el-dialog>

    <!-- 容器详情弹窗 -->
    <el-dialog v-model="inspectVisible" title="容器详情" width="880px">
      <el-skeleton v-if="inspectLoading" :rows="6" animated />
      <div v-else>
        <el-descriptions :column="2" border>
          <el-descriptions-item label="ID">{{ inspectData?.Id || '-' }}</el-descriptions-item>
          <el-descriptions-item label="Name">{{ inspectData?.Name || '-' }}</el-descriptions-item>
          <el-descriptions-item label="Image">{{ inspectData?.Config?.Image || '-' }}</el-descriptions-item>
          <el-descriptions-item label="状态">{{ inspectData?.State?.Status || '-' }}</el-descriptions-item>
        </el-descriptions>

        <el-divider content-position="left">Ports</el-divider>
        <el-table :data="inspectPorts" style="width: 100%">
          <el-table-column prop="container" label="容器端口" width="160" />
          <el-table-column prop="host" label="主机端口" width="160" />
          <el-table-column prop="ip" label="Host IP" width="160" />
          <el-table-column label="复制" width="120">
            <template #default="scope">
              <el-button size="small" @click="copyText(`${scope.row.ip}:${scope.row.host}`)">复制</el-button>
            </template>
          </el-table-column>
        </el-table>

        <el-divider content-position="left">Networks</el-divider>
        <el-table :data="inspectNetworks" style="width: 100%">
          <el-table-column prop="name" label="名称" width="200" />
          <el-table-column prop="ip" label="IP" width="180" />
          <el-table-column prop="gateway" label="网关" width="180" />
        </el-table>

        <el-divider content-position="left">Mounts</el-divider>
        <el-table :data="inspectMounts" style="width: 100%">
          <el-table-column prop="type" label="类型" width="120" />
          <el-table-column prop="source" label="Source" min-width="220" />
          <el-table-column prop="destination" label="Destination" min-width="220" />
          <el-table-column prop="mode" label="Mode" width="120" />
          <el-table-column prop="rw" label="RW" width="80" />
        </el-table>

        <el-divider content-position="left">Env</el-divider>
        <el-input v-model="inspectEnvText" type="textarea" :rows="8" readonly />
      </div>
    </el-dialog>

    <!-- 容器执行命令弹窗 -->
    <el-dialog v-model="execVisible" title="执行容器命令" width="720px">
      <el-alert type="info" :closable="false" show-icon>该功能为非交互命令执行（需要容器内存在 /bin/sh）。</el-alert>
      <div class="log-controls">
        <el-input v-model="execCommand" placeholder="例如: ls / 或 ps aux" />
        <el-button type="primary" @click="runExec" :loading="execLoading">执行</el-button>
      </div>
      <el-input v-model="execOutput" type="textarea" :rows="16" readonly placeholder="输出" />
    </el-dialog>

    <!-- Service 详情弹窗 -->
    <el-dialog v-model="serviceVisible" title="Service 详情" width="880px">
      <el-skeleton v-if="serviceLoading" :rows="6" animated />
      <el-input v-else v-model="serviceJson" type="textarea" :rows="16" readonly />
    </el-dialog>

    <!-- Service 任务弹窗 -->
    <el-dialog v-model="tasksVisible" title="Service 任务" width="880px">
      <el-table :data="serviceTasks" v-loading="tasksLoading" style="width: 100%">
        <el-table-column prop="ID" label="ID" min-width="180" />
        <el-table-column prop="Name" label="名称" min-width="200" />
        <el-table-column prop="Node" label="节点" width="160" />
        <el-table-column prop="DesiredState" label="期望状态" width="120" />
        <el-table-column prop="CurrentState" label="当前状态" min-width="200" />
        <el-table-column prop="Error" label="错误" min-width="200" />
      </el-table>
    </el-dialog>

    <!-- Stack 服务弹窗 -->
    <el-dialog v-model="stackVisible" title="Stack 服务" width="880px">
      <el-table :data="stackServices" v-loading="stackLoading" style="width: 100%">
        <el-table-column prop="Name" label="名称" min-width="220" />
        <el-table-column prop="Mode" label="模式" width="120" />
        <el-table-column prop="Replicas" label="副本" width="120" />
        <el-table-column prop="Image" label="镜像" min-width="180" />
        <el-table-column prop="Ports" label="端口" min-width="180" />
      </el-table>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { ref, reactive, onMounted, watch, computed } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const loading = ref(false)
const tableData = ref([])
const dialogVisible = ref(false)
const submitting = ref(false)
const hosts = ref([])

const manageVisible = ref(false)
const manageTab = ref('overview')
const activeHost = ref(null)

const containers = ref([])
const containersLoading = ref(false)
const images = ref([])
const imagesLoading = ref(false)
const imageTableRef = ref(null)
const selectedImages = ref([])
const networks = ref([])
const networksLoading = ref(false)

const services = ref([])
const servicesLoading = ref(false)
const serviceStackFilter = ref('')
const stacks = ref([])
const stacksLoading = ref(false)

const diagnoseVisible = ref(false)
const diagnoseLoading = ref(false)
const diagnoseResult = ref(null)
const diagnoseError = ref('')

const logVisible = ref(false)
const logLoading = ref(false)
const logText = ref('')
const logTail = ref('100')
const logContainerId = ref('')

const inspectVisible = ref(false)
const inspectLoading = ref(false)
const inspectData = ref(null)
const inspectPorts = ref([])
const inspectNetworks = ref([])
const inspectMounts = ref([])
const inspectEnvText = ref('')

const execVisible = ref(false)
const execLoading = ref(false)
const execCommand = ref('ls /')
const execOutput = ref('')
const execContainerId = ref('')

const serviceVisible = ref(false)
const serviceLoading = ref(false)
const serviceJson = ref('')
const tasksVisible = ref(false)
const tasksLoading = ref(false)
const serviceTasks = ref([])

const stackVisible = ref(false)
const stackLoading = ref(false)
const stackServices = ref([])

const serviceStacks = computed(() => {
  const set = new Set()
  services.value.forEach((s) => {
    const name = s.Name || ''
    const parts = name.split('_')
    if (parts.length > 1) set.add(parts[0])
  })
  return Array.from(set)
})

const filteredServices = computed(() => {
  if (!serviceStackFilter.value) return services.value
  return services.value.filter(s => (s.Name || '').startsWith(`${serviceStackFilter.value}_`))
})

const createVisible = ref(false)
const createLoading = ref(false)
const createForm = reactive({
  image: '',
  name: '',
  ports: '',
  env: '',
  restart_policy: '',
  auto_remove: false
})

const pullVisible = ref(false)
const pullLoading = ref(false)
const pullImage = ref('')
const pullOutput = ref('')

const form = reactive({
  name: '',
  host_id: ''
})

const authHeaders = () => ({ Authorization: 'Bearer ' + localStorage.getItem('token') })

const normalizeContainers = (items) => items.map((row) => {
  const id = row.ID || row.Id || row.id
  const namesRaw = row.Names || row.names || row.Name || row.name
  const names = Array.isArray(namesRaw) ? namesRaw.join(',') : (namesRaw || '-')
  return {
    id,
    names,
    image: row.Image || row.image || '-',
    state: row.State || row.state || '-',
    status: row.Status || row.status || '-',
    created: row.Created || row.created || row.CreatedAt || row.created_at || '-'
  }
})

const normalizeImages = (items) => items.map((row) => ({
  id: row.ID || row.Id || row.id || '-',
  repository: row.Repository || row.repository || '-',
  tag: row.Tag || row.tag || '-',
  size: row.Size || row.size || '-',
  created: row.CreatedAt || row.CreatedSince || row.created_at || row.created || '-'
}))

const normalizeNetworks = (items) => items.map((row) => ({
  id: row.ID || row.Id || row.id || '-',
  name: row.Name || row.name || '-',
  driver: row.Driver || row.driver || '-',
  scope: row.Scope || row.scope || '-'
}))

const fetchData = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/v1/docker/hosts', { headers: authHeaders() })
    if (res.data.code === 0) {
      tableData.value = res.data.data
    }
  } catch (e) {
    ElMessage.error('加载失败')
  } finally {
    loading.value = false
  }
}

const syncAll = async () => {
  loading.value = true
  try {
    await axios.post('/api/v1/docker/hosts/sync', {}, { headers: authHeaders() })
  } catch (e) {
    ElMessage.error('同步失败')
  } finally {
    await fetchData()
  }
}

const fetchCMDBHosts = async () => {
  try {
    const res = await axios.get('/api/v1/cmdb/hosts', { headers: authHeaders() })
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
    const res = await axios.post('/api/v1/docker/hosts', form, { headers: authHeaders() })
    if (res.data.code === 0) {
      ElMessage.success('添加成功')
      dialogVisible.value = false
      try {
        const id = res.data.data?.id
        if (id) {
          await axios.get(`/api/v1/docker/hosts/${id}/info`, { headers: authHeaders() })
        }
      } catch (e) {}
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
    await axios.delete(`/api/v1/docker/hosts/${row.id}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    fetchData()
  })
}

const handleManage = async (row) => {
  activeHost.value = row
  manageVisible.value = true
  manageTab.value = 'overview'
  await refreshManage()
}

const refreshManage = async () => {
  if (!activeHost.value) return
  try {
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/info`, { headers: authHeaders() })
    if (res.data.code === 0) {
      const idx = tableData.value.findIndex(h => h.id === activeHost.value.id)
      if (idx >= 0) tableData.value[idx] = res.data.data
      activeHost.value = res.data.data
    }
  } catch (e) {}
  if (manageTab.value === 'containers') await loadContainers()
  if (manageTab.value === 'images') await loadImages()
  if (manageTab.value === 'networks') await loadNetworks()
}

const loadContainers = async () => {
  if (!activeHost.value) return
  containersLoading.value = true
  try {
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/containers`, { headers: authHeaders() })
    if (res.data.code === 0) {
      containers.value = normalizeContainers(res.data.data || [])
    }
  } finally {
    containersLoading.value = false
  }
}

const loadImages = async () => {
  if (!activeHost.value) return
  imagesLoading.value = true
  try {
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/images`, { headers: authHeaders() })
    if (res.data.code === 0) {
      images.value = normalizeImages(res.data.data || [])
      selectedImages.value = []
      imageTableRef.value?.clearSelection?.()
    }
  } finally {
    imagesLoading.value = false
  }
}

const loadNetworks = async () => {
  if (!activeHost.value) return
  networksLoading.value = true
  try {
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/networks`, { headers: authHeaders() })
    if (res.data.code === 0) {
      networks.value = normalizeNetworks(res.data.data || [])
    }
  } finally {
    networksLoading.value = false
  }
}

const loadServices = async () => {
  if (!activeHost.value) return
  servicesLoading.value = true
  try {
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/services`, { headers: authHeaders() })
    if (res.data.code === 0) {
      services.value = res.data.data || []
    }
  } finally {
    servicesLoading.value = false
  }
}

const loadStacks = async () => {
  if (!activeHost.value) return
  stacksLoading.value = true
  try {
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/stacks`, { headers: authHeaders() })
    if (res.data.code === 0) {
      stacks.value = res.data.data || []
    }
  } finally {
    stacksLoading.value = false
  }
}

const openCreateContainer = () => {
  createForm.image = ''
  createForm.name = ''
  createForm.ports = ''
  createForm.env = ''
  createForm.restart_policy = ''
  createForm.auto_remove = false
  createVisible.value = true
}

const submitCreate = async () => {
  if (!activeHost.value || !createForm.image) {
    ElMessage.warning('请填写镜像')
    return
  }
  createLoading.value = true
  try {
    const ports = createForm.ports
      .split(',')
      .map(v => v.trim())
      .filter(Boolean)
    const env = {}
    createForm.env.split('\n').map(v => v.trim()).filter(Boolean).forEach((line) => {
      const idx = line.indexOf('=')
      if (idx > 0) {
        const k = line.slice(0, idx).trim()
        const v = line.slice(idx + 1).trim()
        env[k] = v
      }
    })
    const payload = {
      name: createForm.name,
      image: createForm.image,
      ports,
      env,
      restart_policy: createForm.restart_policy,
      auto_remove: createForm.auto_remove
    }
    const res = await axios.post(`/api/v1/docker/hosts/${activeHost.value.id}/containers`, payload, { headers: authHeaders() })
    if (res.data.code === 0) {
      ElMessage.success('容器创建成功')
      createVisible.value = false
      loadContainers()
      refreshManage()
    } else {
      ElMessage.error(res.data.message || '创建失败')
    }
  } catch (e) {
    ElMessage.error('创建失败')
  } finally {
    createLoading.value = false
  }
}

const containerAction = async (row, action) => {
  if (!activeHost.value) return
  const id = row.id
  if (!id) return
  try {
    await axios.post(`/api/v1/docker/hosts/${activeHost.value.id}/containers/${encodeURIComponent(id)}/${action}`, {}, { headers: authHeaders() })
    ElMessage.success('操作成功')
    loadContainers()
  } catch (e) {
    ElMessage.error('操作失败')
  }
}

const openLogs = (row) => {
  const id = row.id
  if (!id) return
  logContainerId.value = id
  logText.value = ''
  logVisible.value = true
  loadLogs()
}

const openInspect = async (row) => {
  const id = row.id
  if (!activeHost.value || !id) return
  inspectVisible.value = true
  inspectLoading.value = true
  inspectData.value = null
  inspectPorts.value = []
  inspectNetworks.value = []
  inspectMounts.value = []
  inspectEnvText.value = ''
  try {
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/containers/${encodeURIComponent(id)}`, { headers: authHeaders() })
    if (res.data.code === 0) {
      inspectData.value = res.data.data || null
      const ports = []
      const portMap = inspectData.value?.NetworkSettings?.Ports || {}
      Object.entries(portMap).forEach(([containerPort, hostBindings]) => {
        if (!hostBindings || hostBindings.length === 0) {
          ports.push({ container: containerPort, host: '-', ip: '-' })
          return
        }
        hostBindings.forEach((b) => {
          ports.push({ container: containerPort, host: b.HostPort || '-', ip: b.HostIp || '-' })
        })
      })
      inspectPorts.value = ports

      const networks = []
      const nets = inspectData.value?.NetworkSettings?.Networks || {}
      Object.entries(nets).forEach(([name, info]) => {
        networks.push({ name, ip: info.IPAddress || '-', gateway: info.Gateway || '-' })
      })
      inspectNetworks.value = networks

      const mounts = (inspectData.value?.Mounts || []).map(m => ({
        type: m.Type,
        source: m.Source,
        destination: m.Destination,
        mode: m.Mode,
        rw: m.RW ? 'true' : 'false'
      }))
      inspectMounts.value = mounts

      const env = inspectData.value?.Config?.Env || []
      inspectEnvText.value = env.join('\n')
    }
  } finally {
    inspectLoading.value = false
  }
}

const openExec = (row) => {
  const id = row.id
  if (!id) return
  execContainerId.value = id
  execCommand.value = 'ls /'
  execOutput.value = ''
  execVisible.value = true
}

const copyText = async (text) => {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('已复制')
  } catch (e) {
    ElMessage.warning('复制失败')
  }
}

const runExec = async () => {
  if (!activeHost.value || !execContainerId.value || !execCommand.value) {
    ElMessage.warning('请输入命令')
    return
  }
  execLoading.value = true
  try {
    const res = await axios.post(
      `/api/v1/docker/hosts/${activeHost.value.id}/containers/${encodeURIComponent(execContainerId.value)}/exec`,
      { command: execCommand.value },
      { headers: authHeaders() }
    )
    if (res.data.code === 0) {
      execOutput.value = res.data.data || ''
    } else {
      execOutput.value = res.data.message || '执行失败'
    }
  } catch (e) {
    execOutput.value = '执行失败'
  } finally {
    execLoading.value = false
  }
}

const loadLogs = async () => {
  if (!activeHost.value || !logContainerId.value) return
  logLoading.value = true
  try {
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/containers/${encodeURIComponent(logContainerId.value)}/logs`, {
      params: { tail: logTail.value || '100' },
      headers: authHeaders()
    })
    if (res.data.code === 0) {
      logText.value = res.data.data || ''
    }
  } finally {
    logLoading.value = false
  }
}

const openServiceDetail = async (row) => {
  if (!activeHost.value || !row?.ID) return
  serviceVisible.value = true
  serviceLoading.value = true
  serviceJson.value = ''
  try {
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/services/${encodeURIComponent(row.ID)}`, { headers: authHeaders() })
    if (res.data.code === 0) {
      serviceJson.value = JSON.stringify(res.data.data, null, 2)
    }
  } finally {
    serviceLoading.value = false
  }
}

const openServiceTasks = async (row) => {
  if (!activeHost.value || !row?.ID) return
  tasksVisible.value = true
  tasksLoading.value = true
  serviceTasks.value = []
  try {
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/services/${encodeURIComponent(row.ID)}/tasks`, { headers: authHeaders() })
    if (res.data.code === 0) {
      serviceTasks.value = res.data.data || []
    }
  } finally {
    tasksLoading.value = false
  }
}

const openStackServices = async (row) => {
  if (!activeHost.value || !row?.Name) return
  stackVisible.value = true
  stackLoading.value = true
  stackServices.value = []
  try {
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/stacks/${encodeURIComponent(row.Name)}/services`, { headers: authHeaders() })
    if (res.data.code === 0) {
      stackServices.value = res.data.data || []
    }
  } finally {
    stackLoading.value = false
  }
}

const scaleService = async (row) => {
  if (!activeHost.value || !row?.ID) return
  try {
    const { value } = await ElMessageBox.prompt('输入副本数', '扩缩容', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      inputPattern: /^[0-9]+$/,
      inputErrorMessage: '请输入数字'
    })
    await axios.post(`/api/v1/docker/hosts/${activeHost.value.id}/services/${encodeURIComponent(row.ID)}/scale`, {
      replicas: Number(value)
    }, { headers: authHeaders() })
    ElMessage.success('已提交扩缩容')
    loadServices()
  } catch (e) {}
}

const updateServiceImage = async (row) => {
  if (!activeHost.value || !row?.ID) return
  try {
    const { value } = await ElMessageBox.prompt('输入镜像 (如 nginx:latest)', '更新镜像', {
      confirmButtonText: '确定',
      cancelButtonText: '取消'
    })
    if (!value) return
    await axios.post(`/api/v1/docker/hosts/${activeHost.value.id}/services/${encodeURIComponent(row.ID)}/update_image`, {
      image: value
    }, { headers: authHeaders() })
    ElMessage.success('已提交镜像更新')
    loadServices()
  } catch (e) {}
}

const restartService = async (row) => {
  if (!activeHost.value || !row?.ID) return
  try {
    await ElMessageBox.confirm('确认滚动重启该服务吗？', '提示', { type: 'warning' })
    await axios.post(`/api/v1/docker/hosts/${activeHost.value.id}/services/${encodeURIComponent(row.ID)}/restart`, {}, { headers: authHeaders() })
    ElMessage.success('已触发重启')
    loadServices()
  } catch (e) {}
}

const openPullImage = () => {
  pullImage.value = ''
  pullOutput.value = ''
  pullVisible.value = true
}

const submitPull = async () => {
  if (!activeHost.value || !pullImage.value) {
    ElMessage.warning('请填写镜像')
    return
  }
  pullLoading.value = true
  try {
    const res = await axios.post(`/api/v1/docker/hosts/${activeHost.value.id}/images/pull`, { image: pullImage.value }, { headers: authHeaders() })
    if (res.data.code === 0) {
      pullOutput.value = res.data.output || res.data.message || '拉取完成'
      loadImages()
      refreshManage()
    } else {
      ElMessage.error(res.data.message || '拉取失败')
    }
  } catch (e) {
    ElMessage.error('拉取失败')
  } finally {
    pullLoading.value = false
  }
}

const removeImage = (row) => {
  if (!activeHost.value) return
  const id = row.id
  if (!id) return
  ElMessageBox.confirm('确定删除镜像吗?', '警告', {
    confirmButtonText: '删除',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    await axios.delete(`/api/v1/docker/hosts/${activeHost.value.id}/images/${encodeURIComponent(id)}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    loadImages()
  })
}

const onImageSelectionChange = (rows) => {
  selectedImages.value = rows || []
}

const removeSelectedImages = async () => {
  if (!activeHost.value) return
  const rows = selectedImages.value.filter(r => r.id && r.id !== '-')
  if (rows.length === 0) {
    ElMessage.warning('请选择要删除的镜像')
    return
  }
  try {
    await ElMessageBox.confirm(`确定删除选中的 ${rows.length} 个镜像吗?`, '警告', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning'
    })
  } catch (e) {
    return
  }
  let ok = 0
  let fail = 0
  for (const row of rows) {
    try {
      await axios.delete(`/api/v1/docker/hosts/${activeHost.value.id}/images/${encodeURIComponent(row.id)}`, { headers: authHeaders() })
      ok += 1
    } catch (e) {
      fail += 1
    }
  }
  if (fail === 0) {
    ElMessage.success(`已删除 ${ok} 个镜像`)
  } else {
    ElMessage.warning(`已删除 ${ok} 个，失败 ${fail} 个`)
  }
  loadImages()
}

const handleDiagnose = async (row) => {
  diagnoseVisible.value = true
  diagnoseLoading.value = true
  diagnoseResult.value = null
  diagnoseError.value = ''
  try {
    const res = await axios.post(`/api/v1/docker/hosts/${row.id}/test`, {}, { headers: authHeaders() })
    if (res.data.code === 0) {
      diagnoseResult.value = res.data.data
    } else {
      diagnoseError.value = res.data.message || '诊断失败'
    }
  } catch (e) {
    diagnoseError.value = '诊断失败'
  } finally {
    diagnoseLoading.value = false
  }
}

watch(manageTab, (tab) => {
  if (tab === 'containers') loadContainers()
  if (tab === 'images') loadImages()
  if (tab === 'networks') loadNetworks()
  if (tab === 'services') loadServices()
  if (tab === 'stacks') loadStacks()
})

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
.text-blue-500 { color: #409eff; }
.text-xl { font-size: 18px; }
.drawer-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.drawer-title { font-size: 18px; font-weight: 600; }
.drawer-sub { color: #606266; margin-top: 6px; display: flex; gap: 12px; align-items: center; flex-wrap: wrap; }
.drawer-meta { color: #909399; }
.w-40 { width: 140px; }
.manage-tabs { margin-top: 8px; }
.tab-toolbar { display: flex; justify-content: flex-end; gap: 8px; margin-bottom: 10px; }
.diagnose-block { display: flex; flex-direction: column; gap: 12px; }
.diagnose-title { font-weight: 600; }
.diagnose-pre { background: #0f172a; color: #e2e8f0; padding: 12px; border-radius: 6px; overflow: auto; max-height: 200px; }
.log-controls { display: flex; gap: 8px; margin-bottom: 10px; }
:deep(.el-drawer__body) { padding: 16px; }
</style>
