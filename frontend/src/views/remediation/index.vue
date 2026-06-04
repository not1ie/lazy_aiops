<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>故障自愈日志</h2>
        <p class="page-desc">自动修复执行的脚本记录与结果，支持手动重试。</p>
      </div>
      <div class="page-actions">
        <el-button icon="Refresh" @click="fetchList">刷新</el-button>
      </div>
    </div>

    <el-table :data="logs" v-loading="loading" stripe>
      <el-table-column prop="alert_id" label="告警ID" width="120" />
      <el-table-column prop="target" label="目标" width="150" />
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="row.status === 'success' ? 'success' : row.status === 'running' ? 'warning' : 'danger'" size="small">
            {{ row.status === 'success' ? '成功' : row.status === 'running' ? '运行中' : '失败' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="action" label="修复脚本" min-width="250" show-overflow-tooltip />
      <el-table-column label="耗时" width="80">
        <template #default="{ row }">{{ row.duration || 0 }}s</template>
      </el-table-column>
      <el-table-column label="时间" min-width="160">
        <template #default="{ row }">{{ new Date(row.started_at).toLocaleString() }}</template>
      </el-table-column>
      <el-table-column label="操作" width="100">
        <template #default="{ row }">
          <el-button link type="primary" @click="handleView(row)">详情</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-empty v-if="logs.length === 0 && !loading" description="暂无自愈记录。当告警规则开启自动修复且触发时，这里会出现执行记录。" />

    <!-- Detail Dialog -->
    <el-dialog title="修复详情" v-model="detailVisible" width="680px" append-to-body>
      <el-descriptions :column="2" border>
        <el-descriptions-item label="告警ID">{{ detail.alert_id }}</el-descriptions-item>
        <el-descriptions-item label="状态">
          <el-tag :type="detail.status === 'success' ? 'success' : 'danger'" size="small">{{ detail.status }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="目标">{{ detail.target }}</el-descriptions-item>
        <el-descriptions-item label="耗时">{{ detail.duration || 0 }}s</el-descriptions-item>
        <el-descriptions-item label="开始时间" :span="2">{{ new Date(detail.started_at).toLocaleString() }}</el-descriptions-item>
        <el-descriptions-item label="修复脚本" :span="2">
          <code class="script-block">{{ detail.action }}</code>
        </el-descriptions-item>
        <el-descriptions-item label="输出 (stdout)" :span="2">
          <pre class="output-block">{{ detail.stdout || '(空)' }}</pre>
        </el-descriptions-item>
        <el-descriptions-item label="错误 (stderr)" :span="2">
          <pre class="output-block error">{{ detail.stderr || '(空)' }}</pre>
        </el-descriptions-item>
      </el-descriptions>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })
const loading = ref(false)
const logs = ref([])
const detailVisible = ref(false)
const detail = ref({})

const fetchList = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/v1/remediation/logs', { headers: authHeaders() })
    logs.value = res.data?.data || []
  } catch (err) {
    ElMessage.error('加载自愈日志失败')
  } finally {
    loading.value = false
  }
}

const handleView = async (row) => {
  try {
    const res = await axios.get(`/api/v1/remediation/logs/${row.id}`, { headers: authHeaders() })
    detail.value = res.data?.data || row
    detailVisible.value = true
  } catch {
    detail.value = row
    detailVisible.value = true
  }
}

onMounted(fetchList)
</script>

<style scoped>
.page-card { max-width: 100%; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 16px; }
.page-desc { color: var(--el-text-color-secondary); margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; }
.script-block { background: var(--el-fill-color); padding: 8px 12px; border-radius: 6px; font-family: monospace; font-size: 13px; display: block; max-height: 100px; overflow-y: auto; }
.output-block { margin: 0; font-family: monospace; font-size: 12px; max-height: 200px; overflow-y: auto; white-space: pre-wrap; word-break: break-all; }
.output-block.error { color: var(--el-color-danger); }
</style>
