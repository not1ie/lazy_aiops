<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>Pod 监控</h2>
        <p class="page-desc">基于 Prometheus/cAdvisor 的 Pod 指标（CPU/内存）。</p>
      </div>
      <div class="page-actions">
        <el-button icon="Refresh" @click="fetchMetrics">刷新</el-button>
      </div>
    </div>

    <div class="meta-row" v-if="lastRefresh">
      <span class="meta-text">刷新时间：{{ lastRefresh }}</span>
    </div>

    <el-alert type="info" :closable="false" show-icon>
      如果没有数据，请确认 Prometheus 已采集 kubelet/cAdvisor 指标；若需要重启次数等，请安装 kube-state-metrics。
    </el-alert>

    <div class="filter-bar">
      <el-select v-model="nsFilter" placeholder="命名空间" class="w-40" clearable>
        <el-option v-for="ns in namespaces" :key="ns" :label="ns" :value="ns" />
      </el-select>
      <el-input v-model="keyword" placeholder="搜索Pod/节点" class="w-52" clearable />
      <el-select v-model="topN" class="w-40">
        <el-option label="Top 20" :value="20" />
        <el-option label="Top 50" :value="50" />
        <el-option label="Top 100" :value="100" />
      </el-select>
    </div>

    <el-row :gutter="16" class="summary-row">
      <el-col :span="6"><el-card><div class="card-title">Pod 数</div><div class="card-value">{{ stats.total }}</div></el-card></el-col>
      <el-col :span="6"><el-card><div class="card-title">CPU Top</div><div class="card-value">{{ stats.maxCpu }}</div></el-card></el-col>
      <el-col :span="6"><el-card><div class="card-title">内存 Top(MiB)</div><div class="card-value">{{ stats.maxMem }}</div></el-card></el-col>
    </el-row>

    <el-table :data="filteredRows" v-loading="loading" style="width: 100%; margin-top: 12px">
      <el-table-column prop="namespace" label="命名空间" min-width="160" />
      <el-table-column prop="pod" label="Pod" min-width="220" />
      <el-table-column prop="instance" label="节点" min-width="160" />
      <el-table-column prop="cpu" label="CPU(核)" width="120" sortable />
      <el-table-column prop="memory" label="内存(MiB)" width="140" sortable />
    </el-table>
  </el-card>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'

const rows = ref([])
const loading = ref(false)
const keyword = ref('')
const nsFilter = ref('')
const topN = ref(50)
const lastRefresh = ref('')
const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const cpuQuery = (n) => `topk(${n},
  sum by (pod, namespace, instance) (
    rate(container_cpu_usage_seconds_total{pod!="",container!="POD"}[5m])
  )
  or
  sum by (pod, namespace, instance) (
    label_replace(rate(container_cpu_usage_seconds_total{name!=""}[5m]),
      "pod", "$1", "name", "(.*)"
    )
  )
  or
  sum by (pod, namespace, instance) (
    label_replace(rate(container_cpu_usage_seconds_total{container_label_com_docker_container_name!=""}[5m]),
      "pod", "$1", "container_label_com_docker_container_name", "(.*)"
    )
  )
  or
  sum by (pod, namespace, instance) (
    label_replace(
      label_replace(rate(container_cpu_usage_seconds_total{container_label_com_docker_swarm_service_name!=""}[5m]),
        "pod", "$1", "container_label_com_docker_swarm_service_name", "(.*)"
      ),
      "namespace", "$1", "container_label_com_docker_stack_namespace", "(.*)"
    )
  )
)`

const memQuery = (n) => `topk(${n},
  sum by (pod, namespace, instance) (
    container_memory_working_set_bytes{pod!="",container!="POD"}
  )
  or
  sum by (pod, namespace, instance) (
    label_replace(container_memory_working_set_bytes{name!=""}
      , "pod", "$1", "name", "(.*)"
    )
  )
  or
  sum by (pod, namespace, instance) (
    label_replace(container_memory_working_set_bytes{container_label_com_docker_container_name!=""}
      , "pod", "$1", "container_label_com_docker_container_name", "(.*)"
    )
  )
  or
  sum by (pod, namespace, instance) (
    label_replace(
      label_replace(container_memory_working_set_bytes{container_label_com_docker_swarm_service_name!=""}
        , "pod", "$1", "container_label_com_docker_swarm_service_name", "(.*)"
      ),
      "namespace", "$1", "container_label_com_docker_stack_namespace", "(.*)"
    )
  )
)`

const fetchProm = async (query) => {
  const res = await axios.get('/api/v1/monitor/prometheus/query', {
    headers: authHeaders(),
    params: { query: query.replace(/\s+/g, ' ').trim() }
  })
  if (res.data?.status && res.data.status !== 'success') {
    throw new Error(res.data?.error || 'Prometheus 查询失败')
  }
  return res.data?.data?.result || []
}

const fetchMetrics = async () => {
  loading.value = true
  try {
    const [cpuRes, memRes] = await Promise.all([fetchProm(cpuQuery(topN.value)), fetchProm(memQuery(topN.value))])
    const map = {}
    cpuRes.forEach((item) => {
      const m = item.metric || {}
      const key = `${m.namespace || ''}|${m.pod || ''}|${m.instance || ''}`
      map[key] = map[key] || {
        namespace: m.namespace || '-',
        pod: m.pod || '-',
        instance: m.instance || '-',
        cpu: 0,
        memory: 0
      }
      map[key].cpu = Number(item.value?.[1] || 0).toFixed(3)
    })
    memRes.forEach((item) => {
      const m = item.metric || {}
      const key = `${m.namespace || ''}|${m.pod || ''}|${m.instance || ''}`
      map[key] = map[key] || {
        namespace: m.namespace || '-',
        pod: m.pod || '-',
        instance: m.instance || '-',
        cpu: 0,
        memory: 0
      }
      map[key].memory = (Number(item.value?.[1] || 0) / 1024 / 1024).toFixed(1)
    })
    rows.value = Object.values(map)
    lastRefresh.value = new Date().toLocaleString()
    if (!rows.value.length) {
      ElMessage.warning('未获取到 Pod 指标，请确认 Prometheus 已采集 kubelet/cAdvisor 指标')
    }
  } catch (err) {
    ElMessage.error('拉取 Pod 指标失败')
  } finally {
    loading.value = false
  }
}

const namespaces = computed(() => {
  const set = new Set(rows.value.map(r => r.namespace).filter(Boolean))
  return Array.from(set)
})

const filteredRows = computed(() => {
  const key = keyword.value.trim().toLowerCase()
  return rows.value.filter(r => {
    if (nsFilter.value && r.namespace !== nsFilter.value) return false
    if (!key) return true
    return (
      (r.pod || '').toLowerCase().includes(key) ||
      (r.instance || '').toLowerCase().includes(key)
    )
  })
})

const stats = computed(() => {
  const total = filteredRows.value.length
  const maxCpu = filteredRows.value.reduce((max, r) => Math.max(max, Number(r.cpu || 0)), 0).toFixed(3)
  const maxMem = filteredRows.value.reduce((max, r) => Math.max(max, Number(r.memory || 0)), 0).toFixed(1)
  return { total, maxCpu, maxMem }
})

onMounted(fetchMetrics)
</script>

<style scoped>
.page-card { max-width: 1180px; margin: 0 auto; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 16px; }
.page-desc { color: #606266; margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; }
.filter-bar { display: flex; gap: 8px; margin-top: 12px; }
.summary-row { margin-top: 12px; }
.meta-row { display: flex; align-items: center; margin-top: 8px; color: #606266; font-size: 12px; }
.card-title { color: #909399; font-size: 12px; }
.card-value { font-size: 20px; font-weight: 600; margin-top: 6px; }
.w-52 { width: 220px; }
.w-40 { width: 140px; }
</style>
