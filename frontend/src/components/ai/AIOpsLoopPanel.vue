<template>
  <div class="ops-layout">
    <el-row :gutter="12">
      <el-col :lg="14" :md="24">
        <el-card shadow="never" class="ops-card">
          <template #header>
            <div class="ops-head">
              <div>
                <div class="ops-title">1) 故障诊断</div>
                <div class="ops-sub">输入自然语言，自动生成证据、计划与事件轨迹</div>
              </div>
              <el-tag type="info" effect="plain">Evidence First</el-tag>
            </div>
          </template>

          <el-form label-width="96px">
            <el-form-item label="故障描述">
              <el-input
                v-model="diagnoseForm.query"
                type="textarea"
                :rows="5"
                resize="none"
                placeholder="例如：支付服务响应变慢，帮我定位原因并给出修复建议"
              />
            </el-form-item>
            <el-form-item label="上下文">
              <el-input
                v-model="diagnoseForm.context"
                placeholder="例如：env=prod namespace=payment recent_change=deploy-20260430"
              />
            </el-form-item>
            <el-form-item label="事件编号">
              <el-input v-model="diagnoseForm.incident_id" placeholder="不填将自动生成 CHG-xxxxxx" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" :loading="loading.diagnose" @click="runDiagnose">开始诊断</el-button>
              <el-button plain :disabled="!opsState.incident_id" @click="loadTimeline">刷新时间轴</el-button>
            </el-form-item>
          </el-form>
        </el-card>

        <el-card shadow="never" class="ops-card mt-12">
          <template #header>
            <div class="ops-head">
              <div>
                <div class="ops-title">2) 变更前风险评分</div>
                <div class="ops-sub">高分自动升级审批要求</div>
              </div>
            </div>
          </template>
          <el-form label-width="96px">
            <el-form-item label="执行命令">
              <el-input
                v-model="preflightForm.command"
                type="textarea"
                :rows="3"
                resize="none"
                placeholder="例如：kubectl rollout restart deploy/payment -n payment"
              />
            </el-form-item>
            <el-form-item label="环境">
              <el-select v-model="preflightForm.context" style="width: 180px">
                <el-option label="prod" value="prod" />
                <el-option label="staging" value="staging" />
                <el-option label="dev" value="dev" />
              </el-select>
              <el-button class="ml-8" type="warning" plain :loading="loading.preflight" @click="runPreflight">预检评分</el-button>
            </el-form-item>
          </el-form>

          <div v-if="preflightResult" class="risk-box">
            <div class="risk-row">
              <div class="risk-item">
                <div class="risk-label">Risk Score</div>
                <div class="risk-value" :class="riskScoreClass">{{ preflightResult.risk_score }}</div>
              </div>
              <div class="risk-item">
                <div class="risk-label">审批升级</div>
                <div class="risk-text">{{ preflightResult.escalate_approval ? '是（>=70）' : '否' }}</div>
              </div>
            </div>
            <div class="risk-text">影响范围：{{ preflightResult.blast_radius || '-' }}</div>
            <div class="risk-text">建议窗口：{{ preflightResult.recommended_time || '-' }}</div>
            <div class="risk-text">更安全替代：{{ preflightResult.safer_alternative || '-' }}</div>
          </div>
        </el-card>
      </el-col>

      <el-col :lg="10" :md="24">
        <el-card shadow="never" class="ops-card">
          <template #header>
            <div class="ops-head">
              <div>
                <div class="ops-title">3) 审批与执行回写</div>
                <div class="ops-sub">关联工单并写回 apply/verify/rollback</div>
              </div>
            </div>
          </template>

          <el-form label-width="96px">
            <el-form-item label="Incident ID">
              <el-input v-model="opsState.incident_id" />
            </el-form-item>
            <el-form-item label="审批意见">
              <el-input v-model="approveForm.comment" placeholder="例如：风险可控，同意在低峰窗口执行" />
            </el-form-item>
            <el-form-item>
              <el-button type="success" :loading="loading.approve" @click="submitApprove(true)">审批通过</el-button>
              <el-button type="danger" plain :loading="loading.approve" @click="submitApprove(false)">审批拒绝</el-button>
            </el-form-item>

            <el-divider />
            <el-form-item label="阶段">
              <el-select v-model="executeForm.stage" style="width: 140px">
                <el-option label="apply" value="apply" />
                <el-option label="verify" value="verify" />
                <el-option label="rollback" value="rollback" />
              </el-select>
              <el-switch v-model="executeForm.success" class="ml-8" active-text="成功" inactive-text="失败" />
            </el-form-item>
            <el-form-item label="结果">
              <el-input v-model="executeForm.result" type="textarea" :rows="3" resize="none" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" plain :loading="loading.execute" @click="submitExecute">回写阶段结果</el-button>
            </el-form-item>
          </el-form>
        </el-card>

        <el-card shadow="never" class="ops-card mt-12">
          <template #header>
            <div class="ops-head">
              <div>
                <div class="ops-title">诊断摘要</div>
                <div class="ops-sub">根因推断时刻 + 首次修复动作</div>
              </div>
            </div>
          </template>
          <el-descriptions :column="1" border size="small">
            <el-descriptions-item label="事件编号">{{ opsState.incident_id || '-' }}</el-descriptions-item>
            <el-descriptions-item label="状态">{{ opsState.status || '-' }}</el-descriptions-item>
            <el-descriptions-item label="根因时刻">{{ formatTime(opsState.root_cause_at) }}</el-descriptions-item>
            <el-descriptions-item label="首次修复时刻">{{ formatTime(opsState.first_fix_action_at) }}</el-descriptions-item>
            <el-descriptions-item label="MTTD">{{ opsState.mttd_seconds ?? '-' }}s</el-descriptions-item>
            <el-descriptions-item label="MTTR">{{ opsState.mttr_seconds ?? '-' }}s</el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>
    </el-row>

    <el-card shadow="never" class="ops-card mt-12">
      <template #header>
        <div class="ops-head">
          <div>
            <div class="ops-title">4) 故障时间轴</div>
            <div class="ops-sub">支持 rich / mermaid / json；可用于复盘与文档沉淀</div>
          </div>
          <div class="ops-actions">
            <el-select v-model="timelineForm.format" style="width: 140px">
              <el-option label="rich" value="rich" />
              <el-option label="mermaid" value="mermaid" />
              <el-option label="json" value="json" />
            </el-select>
            <el-input v-model="timelineForm.compare_files" style="width: 320px" placeholder="compare file1,file2 (可选)" />
            <el-button :loading="loading.timeline" @click="loadTimeline">加载时间轴</el-button>
          </div>
        </div>
      </template>

      <el-table :fit="true" :data="timelineEvents" size="small" stripe>
        <el-table-column label="时间" width="180">
          <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column prop="stage" label="阶段" width="120" />
        <el-table-column prop="status" label="状态" width="90" />
        <el-table-column label="耗时(ms)" width="100">
          <template #default="{ row }">{{ row.duration_ms || 0 }}</template>
        </el-table-column>
        <el-table-column prop="detail" label="详情" min-width="360" show-overflow-tooltip />
      </el-table>

      <el-input
        v-if="timelineText"
        v-model="timelineText"
        type="textarea"
        :rows="14"
        resize="none"
        readonly
        class="mt-12"
      />
    </el-card>
  </div>
</template>

<script setup>
import { computed, reactive, ref } from 'vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import { getErrorMessage } from '@/utils/error'

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const loading = reactive({
  diagnose: false,
  preflight: false,
  approve: false,
  execute: false,
  timeline: false
})

const diagnoseForm = reactive({
  query: '',
  context: '',
  incident_id: '',
  title: ''
})

const preflightForm = reactive({
  command: '',
  context: 'prod'
})

const approveForm = reactive({
  comment: ''
})

const executeForm = reactive({
  stage: 'apply',
  success: true,
  result: ''
})

const timelineForm = reactive({
  format: 'rich',
  compare_files: ''
})

const opsState = reactive({
  incident_id: '',
  status: '',
  root_cause_at: '',
  first_fix_action_at: '',
  mttd_seconds: null,
  mttr_seconds: null
})

const preflightResult = ref(null)
const timelineEvents = ref([])
const timelineText = ref('')

const riskScoreClass = computed(() => {
  const score = Number(preflightResult.value?.risk_score || 0)
  if (score >= 80) return 'risk-high'
  if (score >= 50) return 'risk-mid'
  return 'risk-low'
})

const formatTime = (value) => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString()
}

const incidentIdRequired = () => {
  if (!opsState.incident_id.trim()) {
    ElMessage.warning('请先执行诊断或填写 Incident ID')
    return false
  }
  return true
}

const runDiagnose = async () => {
  if (!diagnoseForm.query.trim()) {
    ElMessage.warning('请输入故障描述')
    return
  }
  loading.diagnose = true
  try {
    const payload = {
      query: diagnoseForm.query.trim(),
      context: diagnoseForm.context.trim(),
      incident_id: diagnoseForm.incident_id.trim() || '',
      title: diagnoseForm.title.trim() || ''
    }
    const res = await axios.post('/api/v1/ai/ops/diagnose', payload, { headers: authHeaders() })
    if (res.data?.code === 0) {
      const data = res.data.data || {}
      opsState.incident_id = data.incident_id || ''
      opsState.status = data.status || ''
      opsState.root_cause_at = data.root_cause_at || ''
      opsState.first_fix_action_at = data.first_fix_action_at || ''
      opsState.mttd_seconds = data.mttd_seconds ?? null
      opsState.mttr_seconds = data.mttr_seconds ?? null
      diagnoseForm.incident_id = opsState.incident_id
      if (data.reply) {
        ElMessage.success('诊断完成，已生成建议与计划')
      }
      await loadTimeline()
    }
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '诊断失败'))
  } finally {
    loading.diagnose = false
  }
}

const runPreflight = async () => {
  if (!preflightForm.command.trim()) {
    ElMessage.warning('请输入待执行命令')
    return
  }
  loading.preflight = true
  try {
    const res = await axios.post('/api/v1/ai/ops/preflight', {
      command: preflightForm.command.trim(),
      context: preflightForm.context
    }, { headers: authHeaders() })
    if (res.data?.code === 0) {
      preflightResult.value = res.data.data || null
      ElMessage.success('预检完成')
    }
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '预检失败'))
  } finally {
    loading.preflight = false
  }
}

const submitApprove = async (approved) => {
  if (!incidentIdRequired()) return
  loading.approve = true
  try {
    const res = await axios.post('/api/v1/ai/ops/approve', {
      incident_id: opsState.incident_id.trim(),
      approved,
      comment: approveForm.comment.trim()
    }, { headers: authHeaders() })
    if (res.data?.code === 0) {
      ElMessage.success(approved ? '审批通过并已创建/关联工单' : '已记录审批拒绝')
      await loadTimeline()
    }
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '审批提交失败'))
  } finally {
    loading.approve = false
  }
}

const submitExecute = async () => {
  if (!incidentIdRequired()) return
  loading.execute = true
  try {
    const res = await axios.post('/api/v1/ai/ops/execute', {
      incident_id: opsState.incident_id.trim(),
      stage: executeForm.stage,
      success: executeForm.success,
      result: executeForm.result.trim()
    }, { headers: authHeaders() })
    if (res.data?.code === 0) {
      const item = res.data.data || {}
      opsState.status = item.status || opsState.status
      opsState.root_cause_at = item.root_cause_at || opsState.root_cause_at
      opsState.first_fix_action_at = item.first_fix_action_at || opsState.first_fix_action_at
      opsState.mttd_seconds = item.mttd_seconds ?? opsState.mttd_seconds
      opsState.mttr_seconds = item.mttr_seconds ?? opsState.mttr_seconds
      ElMessage.success('阶段结果已回写')
      await loadTimeline()
    }
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '回写失败'))
  } finally {
    loading.execute = false
  }
}

const loadTimeline = async () => {
  if (!incidentIdRequired()) return
  loading.timeline = true
  try {
    const compareFiles = String(timelineForm.compare_files || '')
      .split(',')
      .map((item) => item.trim())
      .filter(Boolean)
    const res = await axios.post('/api/v1/ai/ops/timeline', {
      incident_id: opsState.incident_id.trim(),
      format: timelineForm.format,
      compare_files: compareFiles
    }, { headers: authHeaders() })
    if (res.data?.code === 0) {
      const data = res.data.data || {}
      timelineEvents.value = data.events || []
      timelineText.value = data.timeline || JSON.stringify(data, null, 2)
      const markers = data.markers || {}
      opsState.status = markers.incident_status || opsState.status
      opsState.mttd_seconds = markers.mttd_persisted_seconds ?? opsState.mttd_seconds
      opsState.mttr_seconds = markers.mttr_persisted_seconds ?? opsState.mttr_seconds
      opsState.root_cause_at = markers.root_cause_at || opsState.root_cause_at
      opsState.first_fix_action_at = markers.first_fix_action_at || opsState.first_fix_action_at
    }
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '加载时间轴失败'))
  } finally {
    loading.timeline = false
  }
}
</script>

<style scoped>
.ops-layout { display: block; }
.ops-card {
  border-radius: 16px;
  border: 1px solid rgba(15, 23, 42, 0.08);
}
.ops-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}
.ops-title {
  font-size: 15px;
  font-weight: 700;
  color: #0f172a;
}
.ops-sub {
  font-size: 12px;
  color: #64748b;
  margin-top: 4px;
}
.ops-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}
.risk-box {
  margin-top: 10px;
  padding: 12px;
  border-radius: 14px;
  background: rgba(15, 23, 42, 0.04);
  border: 1px solid rgba(15, 23, 42, 0.08);
}
.risk-row {
  display: flex;
  gap: 20px;
  margin-bottom: 8px;
}
.risk-item {
  min-width: 120px;
}
.risk-label {
  font-size: 12px;
  color: #64748b;
}
.risk-value {
  font-size: 30px;
  line-height: 1.1;
  font-weight: 700;
}
.risk-value.risk-high { color: #dc2626; }
.risk-value.risk-mid { color: #d97706; }
.risk-value.risk-low { color: #16a34a; }
.risk-text {
  margin-top: 6px;
  font-size: 13px;
  color: #334155;
}
.ml-8 { margin-left: 8px; }
.mt-12 { margin-top: 12px; }

@media (max-width: 768px) {
  .ops-actions {
    width: 100%;
    flex-wrap: wrap;
  }
  .risk-row {
    flex-direction: column;
    gap: 8px;
  }
}
</style>
