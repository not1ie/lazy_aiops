<template>
  <div class="hub-container" v-loading="loading">
    <header class="hub-header">
      <div class="hub-title">
        <h2>容器编排平台</h2>
        <p>一体化管理 Kubernetes 集群与独立 Docker 环境，实现资源视图的深度融合。</p>
      </div>
      <div class="hub-actions">
        <el-button round icon="Refresh" @click="refreshAll">同步集群状态</el-button>
        <el-button round icon="Coin" @click="go('/registries')">镜像仓库</el-button>
        <el-button type="primary" round icon="Plus" @click="go('/k8s/clusters')">接入集群</el-button>
      </div>
    </header>

    <div class="hub-summary">
      <div class="summary-item" @click="activePanel = 'k8s'">
        <div class="summary-label">K8s 集群</div>
        <div class="summary-value success">{{ stats.k8sTotal }}</div>
      </div>
      <div class="summary-item" @click="activePanel = 'docker'">
        <div class="summary-label">Docker 环境</div>
        <div class="summary-value">{{ stats.dockerTotal }}</div>
      </div>
      <div class="summary-item">
        <div class="summary-label">Pod 总数</div>
        <div class="summary-value">1,245</div>
      </div>
      <div class="summary-item">
        <div class="summary-label">集群异常</div>
        <div class="summary-value danger">0</div>
      </div>
    </div>

    <div class="hub-workspace full-width">
      <div class="workspace-content">
        <el-tabs v-model="activePanel" class="hub-tabs">
          <el-tab-pane label="Kubernetes 资源" name="k8s">
            <div class="pane-header">
               <el-button-group>
                 <el-button size="small" type="primary" plain @click="go('/k8s/workloads')">工作负载</el-button>
                 <el-button size="small" type="primary" plain @click="go('/k8s/services')">网络服务</el-button>
                 <el-button size="small" type="primary" plain @click="go('/k8s/storage')">存储卷</el-button>
               </el-button-group>
            </div>
            <el-table :data="clusters" class="hub-table" size="small">
              <el-table-column prop="name" label="集群名称" min-width="150" />
              <el-table-column prop="version" label="版本" width="120" />
              <el-table-column prop="api_server" label="API Server" min-width="200" />
              <el-table-column label="状态" width="100">
                <template #default="{ row }"><StatusBadge :text="row.status === 1 ? '在线' : '离线'" :type="row.status === 1 ? 'success' : 'danger'" /></template>
              </el-table-column>
              <el-table-column label="操作" width="180">
                 <template #default="{ row }">
                   <el-button link type="primary" @click="go('/k8s/workloads')">管理</el-button>
                   <el-button link type="primary" @click="go('/k8s/events')">事件</el-button>
                   <el-button link @click="goAIAnalysis({ type: 'k8s', title: row.name, id: row.id, summary: `K8s集群: ${row.name}, API: ${row.api_server}, 版本: ${row.version}` })">
                     <el-icon><MagicStick /></el-icon>
                   </el-button>
                 </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>

          <el-tab-pane label="Docker 独立环境" name="docker">
            <el-table :data="dockerHosts" class="hub-table" size="small">
              <el-table-column prop="name" label="环境名称" />
              <el-table-column prop="ip" label="宿主机 IP" />
              <el-table-column prop="version" label="Docker 版本" />
              <el-table-column label="操作" width="100">
                 <template #default="{ row }">
                   <el-button link type="primary" @click="go('/docker')">管理</el-button>
                 </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>
        </el-tabs>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import StatusBadge from '@/components/common/StatusBadge.vue'
import './hub-common.css'
import { useAIChat } from '@/composables/useAIChat'

const router = useRouter()
const { goAIAnalysis } = useAIChat()
const loading = ref(false)
const clusters = ref([])
const dockerHosts = ref([])
const activePanel = ref('k8s')

const stats = reactive({
  k8sTotal: 0,
  dockerTotal: 0
})

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })
const go = (path) => router.push(path)

const refreshAll = async () => {
  loading.value = true
  try {
    const [k8s, dkr] = await Promise.all([
      axios.get('/api/v1/k8s/clusters', { headers: authHeaders() }),
      axios.get('/api/v1/docker/hosts', { headers: authHeaders() })
    ])
    clusters.value = k8s.data?.data || []
    dockerHosts.value = dkr.data?.data || []
    stats.k8sTotal = clusters.value.length
    stats.dockerTotal = dockerHosts.value.length
  } catch (err) {
    ElMessage.error('加载容器平台失败')
  } finally {
    loading.value = false
  }
}

onMounted(refreshAll)
</script>

<style scoped>
.full-width { grid-template-columns: 1fr; }
.pane-header { margin-bottom: 16px; }
</style>
