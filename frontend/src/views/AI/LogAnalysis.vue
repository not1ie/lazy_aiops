<template>
  <div class="log-analysis-page">
    <div class="page-header">
      <div>
        <h1>🤖 AI日志分析</h1>
        <p>智能分析日志，自动判断告警并诊断故障</p>
      </div>
    </div>
    
    <!-- 分析配置 -->
    <AppleCard title="分析配置">
      <div class="analysis-form">
        <div class="form-row">
          <div class="form-item">
            <label>服务名称</label>
            <select v-model="config.service" class="apple-select">
              <option value="user-service">user-service</option>
              <option value="order-service">order-service</option>
              <option value="payment-service">payment-service</option>
            </select>
          </div>
          
          <div class="form-item">
            <label>日志级别</label>
            <select v-model="config.level" class="apple-select">
              <option value="ERROR">ERROR</option>
              <option value="WARNING">WARNING</option>
              <option value="INFO">INFO</option>
            </select>
          </div>
          
          <div class="form-item">
            <label>时间范围</label>
            <select v-model="config.timeRange" class="apple-select">
              <option value="5m">最近5分钟</option>
              <option value="15m">最近15分钟</option>
              <option value="1h">最近1小时</option>
              <option value="24h">最近24小时</option>
            </select>
          </div>
        </div>
        
        <AppleButton
          type="primary"
          icon="fas fa-robot"
          :loading="analyzing"
          @click="startAnalysis"
        >
          开始分析
        </AppleButton>
      </div>
    </AppleCard>
    
    <!-- 分析进度 -->
    <AppleCard v-if="analyzing" title="分析中...">
      <div class="progress-container">
        <div class="apple-progress">
          <div class="apple-progress-bar" :style="{ width: progress + '%' }"></div>
        </div>
        <p class="progress-text">已分析 {{ analyzedLogs }} 条日志，发现 {{ anomalies }} 个异常模式</p>
      </div>
    </AppleCard>
    
    <!-- 分析结果 -->
    <AppleCard v-if="result" class="result-card">
      <div class="result-header">
        <h3>
          <i class="fas fa-exclamation-triangle" style="color: var(--apple-danger)"></i>
          发现异常
        </h3>
        <span :class="['apple-badge', `apple-badge-${result.level}`]">
          {{ result.levelText }}
        </span>
      </div>
      
      <div class="result-section">
        <h4>🔍 故障原因</h4>
        <p>{{ result.rootCause }}</p>
      </div>
      
      <div class="result-section">
        <h4>📊 影响范围</h4>
        <ul>
          <li v-for="(impact, index) in result.impact" :key="index">{{ impact }}</li>
        </ul>
      </div>
      
      <div class="result-section">
        <h4>💡 解决建议</h4>
        <ol>
          <li v-for="(solution, index) in result.solutions" :key="index">{{ solution }}</li>
        </ol>
      </div>
      
      <div class="result-section">
        <h4>🛡️ 预防措施</h4>
        <ul>
          <li v-for="(prevention, index) in result.prevention" :key="index">{{ prevention }}</li>
        </ul>
      </div>
      
      <div class="result-footer">
        <div class="confidence">
          <span>AI置信度: </span>
          <strong>{{ result.confidence }}%</strong>
        </div>
        <div class="actions">
          <AppleButton type="ghost">查看详细日志</AppleButton>
          <AppleButton type="secondary">生成报告</AppleButton>
          <AppleButton type="primary">创建告警</AppleButton>
        </div>
      </div>
    </AppleCard>
    
    <!-- 历史分析 -->
    <AppleCard title="历史分析记录">
      <div class="history-table">
        <table>
          <thead>
            <tr>
              <th>时间</th>
              <th>服务</th>
              <th>级别</th>
              <th>故障原因</th>
              <th>状态</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="record in history" :key="record.id">
              <td>{{ record.time }}</td>
              <td>{{ record.service }}</td>
              <td>
                <span :class="['apple-badge', `apple-badge-${record.level}`]">
                  {{ record.levelText }}
                </span>
              </td>
              <td>{{ record.cause }}</td>
              <td>
                <span :class="['status-badge', `status-${record.status}`]">
                  {{ record.statusText }}
                </span>
              </td>
              <td>
                <button class="action-btn">
                  <i class="fas fa-eye"></i>
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </AppleCard>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import AppleCard from '../../components/AppleCard.vue'
import AppleButton from '../../components/AppleButton.vue'

const config = ref({
  service: 'user-service',
  level: 'ERROR',
  timeRange: '15m'
})

const analyzing = ref(false)
const progress = ref(0)
const analyzedLogs = ref(0)
const anomalies = ref(0)

const result = ref(null)

const history = ref([
  {
    id: 1,
    time: '2026-01-22 18:30',
    service: 'user-service',
    level: 'danger',
    levelText: '严重',
    cause: '数据库连接池耗尽',
    status: 'pending',
    statusText: '待处理'
  },
  {
    id: 2,
    time: '2026-01-22 17:15',
    service: 'order-service',
    level: 'warning',
    levelText: '警告',
    cause: '响应时间过长',
    status: 'resolved',
    statusText: '已处理'
  }
])

const startAnalysis = async () => {
  analyzing.value = true
  progress.value = 0
  analyzedLogs.value = 0
  anomalies.value = 0
  result.value = null
  
  // 模拟分析过程
  const interval = setInterval(() => {
    progress.value += 10
    analyzedLogs.value += 15
    anomalies.value = Math.floor(Math.random() * 5)
    
    if (progress.value >= 100) {
      clearInterval(interval)
      analyzing.value = false
      
      // 显示分析结果
      result.value = {
        level: 'danger',
        levelText: '严重',
        rootCause: '数据库连接池耗尽，导致新请求无法建立连接。根据日志分析，连接池配置的最大连接数为100，但当前活跃连接已达到上限，且存在多个长时间未释放的连接。',
        impact: [
          '用户登录功能不可用',
          '订单查询响应超时',
          '预计影响 1000+ 用户'
        ],
        solutions: [
          '立即扩容数据库连接池（建议增加到200）',
          '检查是否有慢查询占用连接（执行时间>5s）',
          '重启应用释放僵死连接',
          '添加连接池监控告警'
        ],
        prevention: [
          '配置连接池自动回收机制',
          '设置连接超时时间',
          '添加慢查询监控',
          '定期检查数据库性能'
        ],
        confidence: 85
      }
    }
  }, 300)
}
</script>

<style scoped>
.log-analysis-page {
  padding: var(--space-xl);
}

.page-header {
  margin-bottom: var(--space-2xl);
}

.page-header h1 {
  font-size: 32px;
  font-weight: 700;
  margin-bottom: var(--space-sm);
}

.page-header p {
  color: var(--apple-text-secondary);
}

.analysis-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-lg);
}

.form-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: var(--space-lg);
}

.form-item label {
  display: block;
  margin-bottom: var(--space-sm);
  color: var(--apple-text-secondary);
  font-size: 14px;
  font-weight: 500;
}

.apple-select {
  width: 100%;
  padding: 12px 16px;
  background: var(--apple-bg-tertiary);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: var(--radius-md);
  color: var(--apple-text-primary);
  font-family: var(--apple-font);
  font-size: 15px;
  cursor: pointer;
  transition: all 0.3s var(--ease-standard);
}

.apple-select:focus {
  outline: none;
  border-color: var(--apple-accent);
  background: var(--apple-bg-elevated);
}

.progress-container {
  display: flex;
  flex-direction: column;
  gap: var(--space-md);
}

.apple-progress {
  width: 100%;
  height: 4px;
  background: var(--apple-bg-tertiary);
  border-radius: var(--radius-full);
  overflow: hidden;
}

.apple-progress-bar {
  height: 100%;
  background: var(--apple-accent);
  border-radius: var(--radius-full);
  transition: width 0.3s var(--ease-standard);
}

.progress-text {
  color: var(--apple-text-secondary);
  font-size: 14px;
}

.result-card {
  margin-top: var(--space-lg);
}

.result-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-xl);
  padding-bottom: var(--space-lg);
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}

.result-header h3 {
  font-size: 20px;
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: var(--space-sm);
}

.result-section {
  margin-bottom: var(--space-xl);
}

.result-section h4 {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: var(--space-md);
  color: var(--apple-text-primary);
}

.result-section p {
  color: var(--apple-text-secondary);
  line-height: 1.6;
}

.result-section ul,
.result-section ol {
  padding-left: var(--space-lg);
  color: var(--apple-text-secondary);
  line-height: 1.8;
}

.result-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: var(--space-lg);
  border-top: 1px solid rgba(255, 255, 255, 0.05);
}

.confidence {
  color: var(--apple-text-secondary);
}

.confidence strong {
  color: var(--apple-accent);
  font-size: 18px;
}

.actions {
  display: flex;
  gap: var(--space-sm);
}

.apple-badge {
  padding: 4px 12px;
  border-radius: var(--radius-full);
  font-size: 12px;
  font-weight: 500;
}

.apple-badge-danger {
  background: rgba(255, 69, 58, 0.2);
  color: var(--apple-danger);
}

.apple-badge-warning {
  background: rgba(255, 159, 10, 0.2);
  color: var(--apple-warning);
}

.history-table {
  overflow-x: auto;
}

table {
  width: 100%;
  border-collapse: collapse;
}

thead {
  background: var(--apple-bg-tertiary);
}

th {
  padding: var(--space-md);
  text-align: left;
  font-weight: 600;
  color: var(--apple-text-secondary);
  font-size: 14px;
}

td {
  padding: var(--space-md);
  border-top: 1px solid rgba(255, 255, 255, 0.05);
  color: var(--apple-text-primary);
}

.status-badge {
  padding: 4px 12px;
  border-radius: var(--radius-full);
  font-size: 12px;
  font-weight: 500;
}

.status-pending {
  background: rgba(255, 159, 10, 0.2);
  color: var(--apple-warning);
}

.status-resolved {
  background: rgba(48, 209, 88, 0.2);
  color: var(--apple-success);
}

.action-btn {
  background: transparent;
  border: none;
  color: var(--apple-text-secondary);
  cursor: pointer;
  padding: var(--space-sm);
  border-radius: var(--radius-sm);
  transition: all 0.2s var(--ease-standard);
}

.action-btn:hover {
  background: rgba(10, 132, 255, 0.1);
  color: var(--apple-accent);
}
</style>
