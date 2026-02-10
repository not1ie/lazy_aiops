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

    <el-alert type="info" :closable="false" show-icon>
      如果没有数据，请确认 Prometheus 已采集 kubelet/cAdvisor 指标；若需要重启次数等，请安装 kube-state-metrics。
    </el-alert>

    <el-table :data="rows" v-loading="loading" style="width: 100%; margin-top: 12px">
      <el-table-column prop="namespace" label="命名空间" min-width="160" />
      <el-table-column prop="pod" label="Pod" min-width="220" />
      <el-table-column prop="instance" label="节点" min-width="160" />
      <el-table-column prop="cpu" label="CPU(核)" width="120" />
      <el-table-column prop="memory" label="内存(MiB)" width="140" />
    </el-table>
  </el-card>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import axios from 'axios'

const rows = ref([])
const loading = ref(false)
const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const cpuQuery =
  'topk(50, sum by (pod, namespace, instance) (rate(container_cpu_usage_seconds_total{pod!=\"\",container!=\"POD\"}[5m])))'
const memQuery =
  'topk(50, sum by (pod, namespace, instance) (container_memory_working_set_bytes{pod!=\"\",container!=\"POD\"}))'

const fetchProm = async (query) => {
  const res = await axios.get('/api/v1/monitor/prometheus/query', {
    headers: authHeaders(),
    params: { query }
  })
  return res.data?.data?.result || []
}

const fetchMetrics = async () => {
  loading.value = true
  try {
    const [cpuRes, memRes] = await Promise.all([fetchProm(cpuQuery), fetchProm(memQuery)])
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
  } finally {
    loading.value = false
  }
}

onMounted(fetchMetrics)
</script>

<style scoped>
.page-card { max-width: 1180px; margin: 0 auto; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 16px; }
.page-desc { color: #606266; margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; }
</style>
