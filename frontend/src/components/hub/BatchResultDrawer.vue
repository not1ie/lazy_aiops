<template>
  <el-drawer
    :model-value="modelValue"
    :title="title || '批量处置结果'"
    size="560px"
    direction="rtl"
    @close="$emit('update:modelValue', false)"
  >
    <div class="result-summary">
      <el-tag type="info" effect="light">总计 {{ summary.total || 0 }}</el-tag>
      <el-tag type="success" effect="light">成功 {{ summary.success || 0 }}</el-tag>
      <el-tag type="danger" effect="light">失败 {{ summary.failed || 0 }}</el-tag>
    </div>

    <el-table :data="records" size="small" max-height="520" empty-text="暂无执行明细">
      <el-table-column prop="target" label="对象" min-width="160" show-overflow-tooltip />
      <el-table-column prop="id" label="ID" min-width="120" show-overflow-tooltip />
      <el-table-column label="结果" width="90">
        <template #default="{ row }">
          <el-tag :type="row.status === 'success' ? 'success' : 'danger'">
            {{ row.status === 'success' ? '成功' : '失败' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="message" label="说明" min-width="220" show-overflow-tooltip />
    </el-table>
  </el-drawer>
</template>

<script setup>
defineProps({
  modelValue: {
    type: Boolean,
    default: false
  },
  title: {
    type: String,
    default: ''
  },
  summary: {
    type: Object,
    default: () => ({ total: 0, success: 0, failed: 0 })
  },
  records: {
    type: Array,
    default: () => []
  }
})

defineEmits(['update:modelValue'])
</script>

<style scoped>
.result-summary {
  margin-bottom: 10px;
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
</style>
