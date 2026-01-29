<template>
  <div class="monitor-page">
    <div class="page-header">
      <h1>监控中心</h1>
      <div class="header-actions">
        <AppleSelect
          v-model="timeRange"
          :options="timeRangeOptions"
          style="width: 150px"
        />
        <AppleButton type="primary" icon="fas fa-sync" @click="refresh">
          刷新
        </AppleButton>
      </div>
    </div>
    
    <!-- 监控概览 -->
    <div class="monitor-overview">
      <AppleCard hoverable>
        <div class="metric-item">
          <i class="fas fa-microchip metric-icon" style="color: var(--apple-accent)"></i>
          <div class="metric-content">
            <div class="metric-value">{{ metrics.cpu }}%</div>
            <div class="metric-label">CPU使用率</div>
            <div class="metric-trend">
              <i class="fas fa-arrow-up" style="color: var(--apple-danger)"></i>
              <span>+5%</span>
            </div>
          </div>
        </div>
      </AppleCard>
      
      <AppleCard hoverable>
        <div class="metric-item">
          <i class="fas fa-memory metric-icon" style="color: var(--apple-success)"></i>
          <div class="metric-content">
            <div class="metric-value">{{ metrics.memory }}%</div>
            <div class="metric-label">内存使用率</div>
            <div class="metric-trend">
              <i class="fas fa-arrow-down" style="color: var(--apple-success)"></i>
              <span>-2%</span>
            </div>
          </div>
        </div>
      </AppleCard>
      
      <AppleCard hoverable>
        <div class="metric-item">
          <i class="fas fa-hdd metric-icon" style="color: var(--apple-warning)"></i>
          <div class="metric-content">
            <div class="metric-value">{{ metrics.disk }}%</div>
            <div class="metric-label">磁盘使用率</div>
            <div class="metric-trend">
              <i class="fas fa-minus" style="color: var(--apple-text-tertiary)"></i>
              <span>0%</span>
            </div>
          </div>
        </div>
      </AppleCard>
      
      <AppleCard hoverable>
        <div class="metric-item">
          <i class="fas fa-network-wired metric-icon" style="color: var(--apple-accent)"></i>
          <div class="metric-content">
            <div class="metric-value">{{ metrics.network }} MB/s</div>
            <div class="metric-label">网络流量</div>
            <div class="metric-trend">
              <i class="fas fa-arrow-up" style="color: var(--apple-success)"></i>
              <span>+12%</span>
            </div>
          </div>
        </div>
      </AppleCard>
    </div>
    
    <!-- 监控图表 -->
    <div class="monitor-charts">
      <AppleCard title="CPU & 内存趋势">
        <div ref="cpuMemoryChart" style="width: 100%; height: 300px;"></div>
      </AppleCard>
      
      <AppleCard title="网络流量">
        <div ref="networkChart" style="width: 100%; height: 300px;"></div>
      </AppleCard>
    </div>
    
    <!-- 服务器列表 -->
    <AppleCard title="服务器监控">
      <AppleTable :columns="columns" :data="servers">
        <template #cell-status="{ value }">
          <AppleBadge :type="value === 'online' ? 'success' : 'danger'">
            {{ value === 'online' ? '在线' : '离线' }}
          </AppleBadge>
        </template>
        <template #cell-cpu="{ value }">
          <div class="progress-cell">
            <div class="progress-bar">
              <div class="progress-fill" :style="{ width: value + '%', background: getProgressColor(value) }"></div>
            </div>
            <span>{{ value }}%</span>
          </div>
        </template>
        <template #cell-memory="{ value }">
          <div class="progress-cell">
            <div class="progress-bar">
              <div class="progress-fill" :style="{ width: value + '%', background: getProgressColor(value) }"></div>
            </div>
            <span>{{ value }}%</span>
          </div>
        </template>
      </AppleTable>
    </AppleCard>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import AppleCard from '../components/AppleCard.vue'
import AppleButton from '../components/AppleButton.vue'
import AppleSelect from '../components/AppleSelect.vue'
import AppleTable from '../components/AppleTable.vue'
import AppleBadge from '../components/AppleBadge.vue'
import * as echarts from 'echarts'

const timeRange = ref('1h')
const timeRangeOptions = [
  { label: '最近1小时', value: '1h' },
  { label: '最近6小时', value: '6h' },
  { label: '最近24小时', value: '24h' },
  { label: '最近7天', value: '7d' }
]

const metrics = ref({
  cpu: 65,
  memory: 78,
  disk: 45,
  network: 125
})

const columns = [
  { key: 'hostname', title: '主机名', width: '150px' },
  { key: 'ip', title: 'IP地址', width: '150px' },
  { key: 'status', title: '状态', width: '100px' },
  { key: 'cpu', title: 'CPU', width: '200px' },
  { key: 'memory', title: '内存', width: '200px' },
  { key: 'uptime', title: '运行时间', width: '150px' }
]

const servers = ref([
  { hostname: 'web-01', ip: '192.168.1.10', status: 'online', cpu: 45, memory: 62, uptime: '15天' },
  { hostname: 'web-02', ip: '192.168.1.11', status: 'online', cpu: 38, memory: 55, uptime: '15天' },
  { hostname: 'db-01', ip: '192.168.1.20', status: 'online', cpu: 78, memory: 85, uptime: '30天' },
  { hostname: 'cache-01', ip: '192.168.1.30', status: 'offline', cpu: 0, memory: 0, uptime: '-' }
])

const cpuMemoryChart = ref(null)
const networkChart = ref(null)

const getProgressColor = (value) => {
  if (value >= 80) return 'var(--apple-danger)'
  if (value >= 60) return 'var(--apple-warning)'
  return 'var(--apple-success)'
}

const refresh = () => {
  console.log('Refreshing...')
}

onMounted(() => {
  initCharts()
})

const initCharts = () => {
  // CPU & 内存图表
  if (cpuMemoryChart.value) {
    const chart1 = echarts.init(cpuMemoryChart.value)
    chart1.setOption({
      backgroundColor: 'transparent',
      tooltip: { trigger: 'axis' },
      legend: {
        data: ['CPU', '内存'],
        textStyle: { color: '#98989d' }
      },
      grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
      xAxis: {
        type: 'category',
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
      series: [
        {
          name: 'CPU',
          type: 'line',
          smooth: true,
          data: [30, 45, 35, 50, 65, 55, 40],
          lineStyle: { color: '#0a84ff', width: 2 },
          itemStyle: { color: '#0a84ff' }
        },
        {
          name: '内存',
          type: 'line',
          smooth: true,
          data: [50, 60, 55, 70, 78, 72, 65],
          lineStyle: { color: '#30d158', width: 2 },
          itemStyle: { color: '#30d158' }
        }
      ]
    })
  }
  
  // 网络流量图表
  if (networkChart.value) {
    const chart2 = echarts.init(networkChart.value)
    chart2.setOption({
      backgroundColor: 'transparent',
      tooltip: { trigger: 'axis' },
      legend: {
        data: ['入站', '出站'],
        textStyle: { color: '#98989d' }
      },
      grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
      xAxis: {
        type: 'category',
        data: ['00:00', '04:00', '08:00', '12:00', '16:00', '20:00', '24:00'],
        axisLine: { lineStyle: { color: 'rgba(255, 255, 255, 0.1)' } },
        axisLabel: { color: '#98989d' }
      },
      yAxis: {
        type: 'value',
        axisLine: { lineStyle: { color: 'rgba(255, 255, 255, 0.1)' } },
        axisLabel: { color: '#98989d', formatter: '{value} MB/s' },
        splitLine: { lineStyle: { color: 'rgba(255, 255, 255, 0.05)' } }
      },
      series: [
        {
          name: '入站',
          type: 'bar',
          data: [80, 95, 85, 110, 125, 115, 100],
          itemStyle: { color: '#0a84ff' }
        },
        {
          name: '出站',
          type: 'bar',
          data: [60, 75, 65, 90, 105, 95, 80],
          itemStyle: { color: '#30d158' }
        }
      ]
    })
  }
}
</script>

<style scoped>
.monitor-page {
  padding: var(--space-xl);
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-2xl);
}

.page-header h1 {
  font-size: 32px;
  font-weight: 700;
}

.header-actions {
  display: flex;
  gap: var(--space-md);
  align-items: center;
}

.monitor-overview {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: var(--space-lg);
  margin-bottom: var(--space-2xl);
}

.metric-item {
  display: flex;
  align-items: center;
  gap: var(--space-lg);
}

.metric-icon {
  font-size: 32px;
}

.metric-content {
  flex: 1;
}

.metric-value {
  font-size: 28px;
  font-weight: 700;
  color: var(--apple-text-primary);
}

.metric-label {
  font-size: 14px;
  color: var(--apple-text-secondary);
  margin-top: var(--space-xs);
}

.metric-trend {
  display: flex;
  align-items: center;
  gap: var(--space-xs);
  margin-top: var(--space-sm);
  font-size: 12px;
  color: var(--apple-text-tertiary);
}

.monitor-charts {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(500px, 1fr));
  gap: var(--space-lg);
  margin-bottom: var(--space-2xl);
}

.progress-cell {
  display: flex;
  align-items: center;
  gap: var(--space-md);
}

.progress-bar {
  flex: 1;
  height: 6px;
  background: var(--apple-bg-tertiary);
  border-radius: var(--radius-full);
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  border-radius: var(--radius-full);
  transition: width 0.3s var(--ease-standard);
}

@media (max-width: 1024px) {
  .monitor-charts {
    grid-template-columns: 1fr;
  }
}
</style>
