<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>容器监控</h2>
        <p class="page-desc">基于 Prometheus/cAdvisor 的容器指标（CPU/内存）。</p>
      </div>
      <div class="page-actions">
        <el-button icon="Refresh" @click="fetchMetrics">刷新</el-button>
      </div>
    </div>

    <el-alert type="info" :closable="false" show-icon>
      如果没有数据，请确认 Prometheus 已采集 cAdvisor 或 kubelet/cAdvisor 指标。
    </el-alert>

    <div class="filter-bar">
      <el-input v-model="keyword" placeholder="搜索容器/镜像/节点" class="w-52" clearable />
      <el-select v-model="topN" class="w-40">
        <el-option label="Top 20" :value="20" />
        <el-option label="Top 50" :value="50" />
        <el-option label="Top 100" :value="100" />
      </el-select>
    </div>

    <el-table :data="filteredRows" v-loading="loading" style="width: 100%; margin-top: 12px">
      <el-table-column prop="container" label="容器" min-width="220" />
      <el-table-column prop="image" label="镜像" min-width="200" />
      <el-table-column prop="instance" label="节点" min-width="160" />
      <el-table-column prop="cpu" label="CPU(核)" width="120" />
      <el-table-column prop="memory" label="内存(MiB)" width="140" />
    </el-table>
  </el-card>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import axios from 'axios'

const rows = ref([])
const loading = ref(false)
const keyword = ref('')
const topN = ref(50)
const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const cpuQuery = (n) =>
  `topk(${n}, sum by (container, instance, image) (rate(container_cpu_usage_seconds_total{container!="",container!="POD"}[5m])))`
const memQuery = (n) =>
  `topk(${n}, sum by (container, instance, image) (container_memory_working_set_bytes{container!="",container!="POD"}))`

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
    const [cpuRes, memRes] = await Promise.all([fetchProm(cpuQuery(topN.value)), fetchProm(memQuery(topN.value))])
    const map = {}
    cpuRes.forEach((item) => {
      const m = item.metric || {}
      const key = `${m.container || ''}|${m.instance || ''}|${m.image || ''}`
      map[key] = map[key] || {
        container: m.container || '-',
        image: m.image || '-',
        instance: m.instance || '-',
        cpu: 0,
        memory: 0
      }
      map[key].cpu = Number(item.value?.[1] || 0).toFixed(3)
    })
    memRes.forEach((item) => {
      const m = item.metric || {}
      const key = `${m.container || ''}|${m.instance || ''}|${m.image || ''}`
      map[key] = map[key] || {
        container: m.container || '-',
        image: m.image || '-',
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

const filteredRows = computed(() => {
  const key = keyword.value.trim().toLowerCase()
  if (!key) return rows.value
  return rows.value.filter(r =>
    (r.container || '').toLowerCase().includes(key) ||
    (r.image || '').toLowerCase().includes(key) ||
    (r.instance || '').toLowerCase().includes(key)
  )
})

onMounted(fetchMetrics)
</script>

<style scoped>
.page-card { max-width: 1180px; margin: 0 auto; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 16px; }
.page-desc { color: #606266; margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; }
.filter-bar { display: flex; gap: 8px; margin-top: 12px; }
.w-52 { width: 220px; }
.w-40 { width: 140px; }
</style>
