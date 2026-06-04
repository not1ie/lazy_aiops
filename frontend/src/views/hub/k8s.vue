<template>
  <div class="hub-container" v-loading="loading">
    <header class="hub-header">
      <div class="hub-title">
        <h2>容器排障工作台</h2>
        <p>从集群总览到 Pod 日志、从事件诊断到 WebShell，一站式容器排障。</p>
      </div>
      <div class="hub-actions">
        <el-button round icon="Refresh" @click="refreshAll">同步状态</el-button>
        <el-button round icon="Coin" @click="go('/registries')">镜像仓库</el-button>
        <el-button type="primary" round icon="Plus" @click="go('/k8s/clusters')">接入集群</el-button>
      </div>
    </header>

    <div class="hub-summary">
      <div class="summary-item" @click="activePanel = 'clusters'">
        <div class="summary-label">K8s 集群</div>
        <div class="summary-value success">{{ stats.k8sTotal }}</div>
      </div>
      <div class="summary-item" @click="activePanel = 'clusters'">
        <div class="summary-label">Docker 环境</div>
        <div class="summary-value">{{ stats.dockerTotal }}</div>
      </div>
      <div class="summary-item" @click="activePanel = 'quick'">
        <div class="summary-label">集群异常</div>
        <div class="summary-value" :class="stats.k8sUnhealthy > 0 ? 'danger' : 'success'">{{ stats.k8sUnhealthy || 0 }}</div>
      </div>
      <div class="summary-item" @click="activePanel = 'quick'">
        <div class="summary-label">快捷排障</div>
        <div class="summary-value">→</div>
      </div>
    </div>

    <div class="hub-workspace full-width">
      <div class="workspace-content">
        <el-tabs v-model="activePanel" class="hub-tabs">
          <!-- 1. Cluster Overview -->
          <el-tab-pane label="集群总览" name="clusters">
            <el-table :data="clusters" class="hub-table" size="small">
              <el-table-column prop="name" label="集群名称" min-width="150" />
              <el-table-column prop="version" label="版本" width="120" />
              <el-table-column prop="api_server" label="API Server" min-width="200" show-overflow-tooltip />
              <el-table-column label="状态" width="100">
                <template #default="{ row }">
                  <StatusBadge :text="row.status === 1 ? '在线' : '离线'" :type="row.status === 1 ? 'success' : 'danger'" />
                </template>
              </el-table-column>
              <el-table-column label="排障操作" width="280" fixed="right">
                <template #default="{ row }">
                  <el-button link type="primary" size="small" @click="go('/k8s/workloads?cluster=' + row.id)">工作负载</el-button>
                  <el-button link size="small" @click="go('/k8s/pods?cluster=' + row.id)">Pods</el-button>
                  <el-button link size="small" @click="go('/k8s/services?cluster=' + row.id)">服务</el-button>
                  <el-button link type="warning" size="small" @click="go('/k8s/events?cluster=' + row.id)">事件</el-button>
                  <el-button link size="small" @click="goAIAnalysis({ type: 'k8s', title: row.name, id: row.id, summary: `K8s集群: ${row.name}, API: ${row.api_server}, 版本: ${row.version}` })">
                    <el-icon><MagicStick /></el-icon>
                  </el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>

          <!-- 2. Quick Troubleshooting -->
          <el-tab-pane label="快捷排障" name="quick">
            <div class="quick-grid">
              <el-card class="quick-card" shadow="hover" @click="go('/k8s/workloads')">
                <el-icon :size="28" class="qc-icon qc-blue"><Cpu /></el-icon>
                <div class="qc-title">工作负载</div>
                <div class="qc-desc">Deployments, StatefulSets, DaemonSets</div>
              </el-card>
              <el-card class="quick-card" shadow="hover" @click="go('/k8s/pods')">
                <el-icon :size="28" class="qc-icon qc-green"><Box /></el-icon>
                <div class="qc-title">Pod 管理</div>
                <div class="qc-desc">查看日志、重启、Shell 接入</div>
              </el-card>
              <el-card class="quick-card" shadow="hover" @click="go('/k8s/events')">
                <el-icon :size="28" class="qc-icon qc-red"><WarningFilled /></el-icon>
                <div class="qc-title">事件诊断</div>
                <div class="qc-desc">集群事件、错误与告警</div>
              </el-card>
              <el-card class="quick-card" shadow="hover" @click="go('/k8s/services')">
                <el-icon :size="28" class="qc-icon qc-purple"><Share /></el-icon>
                <div class="qc-title">网络服务</div>
                <div class="qc-desc">Service, Ingress, 端口映射</div>
              </el-card>
              <el-card class="quick-card" shadow="hover" @click="go('/k8s/nodes')">
                <el-icon :size="28" class="qc-icon qc-orange"><Grid /></el-icon>
                <div class="qc-title">节点管理</div>
                <div class="qc-desc">节点状态、资源、调度</div>
              </el-card>
              <el-card class="quick-card" shadow="hover" @click="go('/k8s/storage')">
                <el-icon :size="28" class="qc-icon qc-teal"><Coin /></el-icon>
                <div class="qc-title">存储卷</div>
                <div class="qc-desc">PVC, StorageClass</div>
              </el-card>
              <el-card class="quick-card" shadow="hover" @click="go('/k8s/configs')">
                <el-icon :size="28" class="qc-icon qc-gray"><Tickets /></el-icon>
                <div class="qc-title">配置管理</div>
                <div class="qc-desc">ConfigMap, Secret</div>
              </el-card>
              <el-card class="quick-card" shadow="hover" @click="go('/k8s/terminal')">
                <el-icon :size="28" class="qc-icon qc-dark"><Monitor /></el-icon>
                <div class="qc-title">WebShell</div>
                <div class="qc-desc">容器内命令行排障</div>
              </el-card>
            </div>
          </el-tab-pane>

          <!-- 3. Docker Hosts -->
          <el-tab-pane label="Docker 环境" name="docker">
            <div class="pane-header">
              <span class="text-muted">{{ stats.dockerTotal }} 个 Docker 环境</span>
              <el-button size="small" type="primary" @click="go('/docker')">Docker 管理 →</el-button>
            </div>
            <el-table :data="dockerHosts" class="hub-table" size="small">
              <el-table-column prop="name" label="环境名称" min-width="150" />
              <el-table-column prop="ip" label="宿主机 IP" width="140" />
              <el-table-column prop="version" label="版本" width="150" show-overflow-tooltip />
              <el-table-column label="最后检测" width="130">
                <template #default="{ row }">
                  <span :class="{ 'text-warning': isDockerStale(row) }">{{ fmtCheckTime(row.last_check_at) }}</span>
                </template>
              </el-table-column>
              <el-table-column label="操作" width="100">
                <template #default="{ row }">
                  <el-button link type="primary" size="small" @click="go('/docker')">管理</el-button>
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
const activePanel = ref('clusters')

const stats = reactive({ k8sTotal: 0, k8sUnhealthy: 0, dockerTotal: 0 })

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })
const go = (path) => router.push(path)

const fmtCheckTime = (val) => {
  if (!val) return '从未检测'
  const diff = Math.floor((Date.now() - new Date(val).getTime()) / 1000)
  if (diff < 60) return '刚刚'
  if (diff < 3600) return `${Math.floor(diff / 60)}m 前`
  if (diff < 86400) return `${Math.floor(diff / 3600)}h 前`
  return `${Math.floor(diff / 86400)}d 前`
}

const isDockerStale = (row) => {
  if (!row.last_check_at) return true
  return Date.now() - new Date(row.last_check_at).getTime() > 120000
}

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
    stats.k8sUnhealthy = clusters.value.filter(c => c.status !== 1).length
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
.pane-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.text-muted { color: var(--el-text-color-secondary); font-size: 13px; }
.text-warning { color: var(--el-color-warning); font-weight: 600; }

.quick-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 14px; }
.quick-card { cursor: pointer; text-align: center; padding: 10px; }
.quick-card:hover { border-color: var(--apple-blue); }
.qc-icon { margin-bottom: 10px; }
.qc-blue { color: #0071e3; }
.qc-green { color: #10b981; }
.qc-red { color: #ef4444; }
.qc-purple { color: #8b5cf6; }
.qc-orange { color: #f59e0b; }
.qc-teal { color: #14b8a6; }
.qc-gray { color: #6b7280; }
.qc-dark { color: #1f2937; }
.qc-title { font-size: 15px; font-weight: 700; margin-bottom: 4px; color: var(--el-text-color-primary); }
.qc-desc { font-size: 12px; color: var(--el-text-color-secondary); }

@media (max-width: 900px) {
  .quick-grid { grid-template-columns: repeat(2, 1fr); }
}
</style>
