<template>
  <el-card class="side-card ops-calendar-card" shadow="never">
    <div class="card-title-row">
      <h3>📅 Ops Calendar</h3>
      <span class="week-label">{{ weekLabel }}</span>
    </div>

    <div v-loading="loading" class="cal-body">
      <!-- Oncall -->
      <div class="cal-section">
        <div class="cal-section-title">
          <el-icon><Calendar /></el-icon> 当前值班
        </div>
        <div v-if="oncallList.length === 0" class="cal-empty">未设置值班</div>
        <div v-for="item in oncallList" :key="item.id || item.username" class="cal-item">
          <span class="cal-name">{{ item.username || item.name }}</span>
          <span class="cal-meta">{{ item.type || item.level || '主值班' }}</span>
          <span v-if="item.end_at" class="cal-time">至 {{ fmtShort(item.end_at) }}</span>
        </div>
      </div>

      <!-- Upcoming Releases -->
      <div class="cal-section">
        <div class="cal-section-title">
          <el-icon><Promotion /></el-icon> 发布窗口
        </div>
        <div v-if="schedules.length === 0" class="cal-empty">本周无计划发布</div>
        <div v-for="item in schedules.slice(0, 5)" :key="item.id" class="cal-item">
          <span class="cal-name">{{ item.name }}</span>
          <span class="cal-meta">
            <el-tag size="small" :type="item.enabled ? 'success' : 'info'" effect="plain">{{ item.enabled ? '启用' : '停用' }}</el-tag>
          </span>
          <span v-if="item.next_run_at" class="cal-time">{{ fmtShort(item.next_run_at) }}</span>
        </div>
      </div>

      <!-- Active Silences -->
      <div class="cal-section">
        <div class="cal-section-title">
          <el-icon><MuteNotification /></el-icon> 告警静默
        </div>
        <div v-if="silences.length === 0" class="cal-empty">无活跃静默</div>
        <div v-for="item in silences.slice(0, 5)" :key="item.id" class="cal-item">
          <span class="cal-name">{{ item.comment || item.matchers || '静默规则' }}</span>
          <span class="cal-meta">
            <el-tag size="small" type="warning" effect="plain">静默中</el-tag>
          </span>
          <span v-if="item.ends_at" class="cal-time">至 {{ fmtShort(item.ends_at) }}</span>
        </div>
      </div>
    </div>
  </el-card>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import axios from 'axios'

const loading = ref(false)
const oncallList = ref([])
const schedules = ref([])
const silences = ref([])

const weekLabel = computed(() => {
  const now = new Date()
  const monday = new Date(now)
  monday.setDate(now.getDate() - now.getDay() + 1)
  const friday = new Date(monday)
  friday.setDate(monday.getDate() + 4)
  return `${monday.getMonth() + 1}/${monday.getDate()} – ${friday.getMonth() + 1}/${friday.getDate()}`
})

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const fmtShort = (val) => {
  if (!val) return ''
  const d = new Date(val)
  return `${d.getMonth() + 1}/${d.getDate()} ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
}

const fetchData = async () => {
  loading.value = true
  try {
    const headers = authHeaders()
    const [oncallRes, schedRes, silenceRes] = await Promise.all([
      axios.get('/api/v1/oncall/whoisoncall', { headers }).catch(() => ({ data: {} })),
      axios.get('/api/v1/cicd/schedules', { headers }).catch(() => ({ data: {} })),
      axios.get('/api/v1/alert/silences', { headers }).catch(() => ({ data: {} }))
    ])
    oncallList.value = oncallRes.data?.data || []
    schedules.value = (schedRes.data?.data || []).filter(s => s.enabled)
    silences.value = (silenceRes.data?.data || []).filter(s => s.status === 'active' || s.active)
  } catch (e) {
    // silent
  } finally {
    loading.value = false
  }
}

onMounted(fetchData)
</script>

<style scoped>
.ops-calendar-card { margin-top: 0; }
.card-title-row { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.card-title-row h3 { font-size: 16px; font-weight: 700; margin: 0; }
.week-label { font-size: 12px; font-weight: 600; color: var(--el-text-color-secondary); background: var(--el-fill-color); padding: 4px 10px; border-radius: 6px; }
.cal-body { min-height: 120px; }
.cal-section { margin-bottom: 18px; }
.cal-section:last-child { margin-bottom: 0; }
.cal-section-title {
  font-size: 12px;
  font-weight: 700;
  color: var(--el-text-color-secondary);
  text-transform: uppercase;
  margin-bottom: 10px;
  display: flex;
  align-items: center;
  gap: 6px;
}
.cal-empty { font-size: 12px; color: var(--el-text-color-placeholder); padding: 8px 0; }
.cal-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 0;
  font-size: 13px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.cal-item:last-child { border-bottom: none; }
.cal-name { font-weight: 600; color: var(--el-text-color-primary); flex: 1; }
.cal-meta { flex-shrink: 0; }
.cal-time { font-size: 11px; color: var(--el-text-color-secondary); font-family: monospace; }
</style>
