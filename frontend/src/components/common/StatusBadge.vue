<template>
  <el-tooltip
    v-if="tooltipContent"
    :content="tooltipContent"
    :placement="placement"
    effect="dark"
  >
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

const tooltipContent = computed(() => {
  const parts = []
  if (props.reason && String(props.reason).trim()) parts.push(String(props.reason).trim())
  const timeText = formatTime(props.updatedAt)
  if (timeText) parts.push(`更新时间: ${timeText}`)
  return parts.join(' | ')
})
</script>
