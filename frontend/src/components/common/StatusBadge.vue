<template>
  <el-tooltip
    v-if="tooltipLines.length"
    :placement="placement"
    effect="dark"
  >
    <template #content>
      <div class="status-tooltip-content">
        <div v-for="(line, idx) in tooltipLines" :key="idx">{{ line }}</div>
      </div>
    </template>
    <el-tag :type="type" :size="size" :effect="effect">{{ text || '-' }}</el-tag>
  </el-tooltip>
  <el-tag v-else :type="type" :size="size" :effect="effect">{{ text || '-' }}</el-tag>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  text: {
    type: String,
    default: ''
  },
  type: {
    type: String,
    default: 'info'
  },
  reason: {
    type: String,
    default: ''
  },
  source: {
    type: String,
    default: ''
  },
  checkAt: {
    type: [String, Number, Date],
    default: ''
  },
  isStale: {
    type: Boolean,
    default: false
  },
  staleText: {
    type: String,
    default: ''
  },
  updatedAt: {
    type: [String, Number, Date],
    default: ''
  },
  size: {
    type: String,
    default: 'default'
  },
  effect: {
    type: String,
    default: 'light'
  },
  placement: {
    type: String,
    default: 'top'
  }
})

const formatTime = (value) => {
  if (!value) return ''
  const ts = new Date(value).getTime()
  if (Number.isNaN(ts)) return ''
  return new Date(ts).toLocaleString()
}

const tooltipLines = computed(() => {
  const lines = []
  const sourceText = String(props.source || '').trim()
  if (sourceText) lines.push(`来源: ${sourceText}`)

  const checkText = formatTime(props.checkAt)
  const updateText = formatTime(props.updatedAt)
  if (checkText) lines.push(`检测时间: ${checkText}`)
  else if (updateText) lines.push(`更新时间: ${updateText}`)

  if (props.isStale) {
    const stale = String(props.staleText || '').trim() || '状态已过期，请重新检测'
    lines.push(stale)
  }

  const reasonText = String(props.reason || '').trim()
  if (reasonText) lines.push(`说明: ${reasonText}`)
  return lines
})
</script>

<style scoped>
.status-tooltip-content {
  line-height: 1.5;
  max-width: 380px;
}
</style>
