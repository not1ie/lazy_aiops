<template>
  <el-card class="page-card">
    <div class="page-header">
      <div>
        <h2>复盘详情</h2>
        <p class="page-desc">编辑根因与处理方式。</p>
      </div>
      <div class="page-actions">
        <el-button icon="Refresh" @click="fetchDetail">刷新</el-button>
        <el-button type="primary" @click="save">保存</el-button>
      </div>
    </div>

    <el-descriptions :column="2" border>
      <el-descriptions-item label="规则ID">{{ item.rule_id }}</el-descriptions-item>
      <el-descriptions-item label="目标">{{ item.target }}</el-descriptions-item>
      <el-descriptions-item label="级别">{{ item.severity }}</el-descriptions-item>
      <el-descriptions-item label="触发时间">{{ item.fired_at }}</el-descriptions-item>
      <el-descriptions-item label="恢复时间">{{ item.resolved_at || '-' }}</el-descriptions-item>
      <el-descriptions-item label="持续(s)">{{ item.duration }}</el-descriptions-item>
    </el-descriptions>

    <el-divider />

    <el-form label-width="110px">
      <el-form-item label="处理方式">
        <el-input v-model="resolution" type="textarea" :rows="4" />
      </el-form-item>
      <el-form-item label="根因">
        <el-input v-model="rootCause" type="textarea" :rows="4" />
      </el-form-item>
    </el-form>
  </el-card>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import axios from 'axios'
import { ElMessage } from 'element-plus'

const route = useRoute()
const item = ref({})
const resolution = ref('')
const rootCause = ref('')

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const fetchDetail = async () => {
  const id = route.query.id
  if (!id) return
  const res = await axios.get(`/api/v1/alert/history/${id}`, { headers: authHeaders() })
  item.value = res.data.data || {}
  resolution.value = item.value.resolution || ''
  rootCause.value = item.value.root_cause || ''
}

const save = async () => {
  const id = route.query.id
  if (!id) return
  await axios.put(`/api/v1/alert/history/${id}`, {
    resolution: resolution.value,
    root_cause: rootCause.value
  }, { headers: authHeaders() })
  ElMessage.success('保存成功')
  fetchDetail()
}

onMounted(fetchDetail)
</script>

<style scoped>
.page-card { max-width: 100%; margin: 0; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 16px; }
.page-desc { color: #606266; margin: 4px 0 0; }
.page-actions { display: flex; gap: 8px; }
</style>
