<template>
  <div class="dashboard">
    <div class="dashboard-header">
      <h1>仪表板</h1>
      <p>系统运行状态总览</p>
    </div>
    
    <div class="dashboard-stats">
      <AppleCard hoverable>
        <div class="stat-item">
          <i class="fas fa-server stat-icon" style="color: var(--apple-accent)"></i>
          <div class="stat-content">
            <div class="stat-value">{{ stats.servers }}</div>
            <div class="stat-label">服务器</div>
          </div>
        </div>
      </AppleCard>
      
      <AppleCard hoverable>
        <div class="stat-item">
          <i class="fas fa-exclamation-triangle stat-icon" style="color: var(--apple-warning)"></i>
          <div class="stat-content">
            <div class="stat-value">{{ stats.alerts }}</div>
            <div class="stat-label">告警</div>
          </div>
        </div>
      </AppleCard>
      
      <AppleCard hoverable>
        <div class="stat-item">
          <i class="fas fa-tasks stat-icon" style="color: var(--apple-success)"></i>
          <div class="stat-content">
            <div class="stat-value">{{ stats.tasks }}</div>
            <div class="stat-label">任务</div>
          </div>
        </div>
      </AppleCard>
      
      <AppleCard hoverable>
        <div class="stat-item">
          <i class="fas fa-chart-line stat-icon" style="color: var(--apple-danger)"></i>
          <div class="stat-content">
            <div class="stat-value">{{ stats.uptime }}%</div>
            <div class="stat-label">可用率</div>
          </div>
        </div>
      </AppleCard>
    </div>
    
    <div class="dashboard-content">
      <AppleCard title="系统监控">
        <div class="chart-container">
          <div ref="cpuChart" style="width: 100%; height: 300px;"></div>
        </div>
      </AppleCard>
      
      <AppleCard title="最近告警">
        <div class="alert-list">
          <div v-for="alert in recentAlerts" :key="alert.id" class="alert-item">
            <span :class="['alert-badge', `alert-${alert.level}`]"></span>
            <div class="alert-info">
              <div class="alert-title">{{ alert.title }}</div>
              <div class="alert-time">{{ alert.time }}</div>
            </div>
          </div>
        </div>
      </AppleCard>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import AppleCard from '../components/AppleCard.vue'
import * as echarts from 'echarts'

const stats = ref({
  servers: 24,
  alerts: 3,
  tasks: 156,
  uptime: 99.9
})

const recentAlerts = ref([
  { id: 1, title: 'CPU使用率过高', level: 'warning', time: '5分钟前' },
  { id: 2, title: '磁盘空间不足', level: 'danger', time: '10分钟前' },
  { id: 3, title: '服务响应缓慢', level: 'warning', time: '15分钟前' }
])

const cpuChart = ref(null)

onMounted(() => {
  initChart()
})

const initChart = () => {
  if (!cpuChart.value) return
  
  const chart = echarts.init(cpuChart.value)
  const option = {
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'axis',
      backgroundColor: 'rgba(28, 28, 30, 0.9)',
      borderColor: 'rgba(255, 255, 255, 0.1)',
      textStyle: { color: '#fff' }
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: ['00:00', '04:00', '08:00', '12:00', '16:00', '20:00', '24:00'],
      axisLine: { lineStyle: { color: 'rgba(255, 255, 255, 0.1)' } },
      axisLabel: { color: '#98989d' }
    },
    yAxis: {
      type: 'value',
      axisLine: { lineStyle: { color: 'rgba(255, 255, 255, 0.1)' } },
      axisLabel: { color: '#98989d' },
      splitLine: { lineStyle: { color: 'rgba(255, 255, 255, 0.05)' } }
    },
    series: [{
      name: 'CPU使用率',
      type: 'line',
      smooth: true,
      data: [30, 45, 35, 50, 65, 55, 40],
      areaStyle: {
        color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
          { offset: 0, color: 'rgba(10, 132, 255, 0.3)' },
          { offset: 1, color: 'rgba(10, 132, 255, 0)' }
        ])
      },
      lineStyle: { color: '#0a84ff', width: 2 },
      itemStyle: { color: '#0a84ff' }
    }]
  }
  
  chart.setOption(option)
  
  window.addEventListener('resize', () => chart.resize())
}
</script>

<style scoped>
.dashboard {
  padding: var(--space-xl);
}

.dashboard-header {
  margin-bottom: var(--space-2xl);
}

.dashboard-header h1 {
  font-size: 32px;
  font-weight: 700;
  margin-bottom: var(--space-sm);
}

.dashboard-header p {
  color: var(--apple-text-secondary);
}

.dashboard-stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: var(--space-lg);
  margin-bottom: var(--space-2xl);
}

.stat-item {
  display: flex;
  align-items: center;
  gap: var(--space-lg);
}

.stat-icon {
  font-size: 32px;
}

.stat-content {
  flex: 1;
}

.stat-value {
  font-size: 28px;
  font-weight: 700;
  color: var(--apple-text-primary);
}

.stat-label {
  font-size: 14px;
  color: var(--apple-text-secondary);
  margin-top: var(--space-xs);
}

.dashboard-content {
  display: grid;
  grid-template-columns: 2fr 1fr;
  gap: var(--space-lg);
}

.alert-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-md);
}

.alert-item {
  display: flex;
  align-items: center;
  gap: var(--space-md);
  padding: var(--space-md);
  background: var(--apple-bg-tertiary);
  border-radius: var(--radius-md);
  transition: all 0.2s var(--ease-standard);
}

.alert-item:hover {
  background: var(--apple-bg-elevated);
}

.alert-badge {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.alert-warning {
  background: var(--apple-warning);
}

.alert-danger {
  background: var(--apple-danger);
}

.alert-info {
  flex: 1;
}

.alert-title {
  font-size: 14px;
  color: var(--apple-text-primary);
  margin-bottom: var(--space-xs);
}

.alert-time {
  font-size: 12px;
  color: var(--apple-text-tertiary);
}

@media (max-width: 1024px) {
  .dashboard-content {
    grid-template-columns: 1fr;
  }
}
</style>
