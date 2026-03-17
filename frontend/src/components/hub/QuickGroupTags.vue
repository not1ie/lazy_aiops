<template>
  <div v-if="normalizedGroups.length" class="group-tag-list">
    <el-tag
      v-for="group in normalizedGroups"
      :key="group.key"
      :type="group.type"
      effect="plain"
      class="group-tag-item"
      @click="$emit('select', group.raw)"
    >
      {{ group.label }}
    </el-tag>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  groups: {
    type: Array,
    default: () => []
  },
  defaultType: {
    type: String,
    default: 'warning'
  }
})

defineEmits(['select'])

const normalizedGroups = computed(() =>
  (Array.isArray(props.groups) ? props.groups : [])
    .filter((item) => item && (item.count === undefined || Number(item.count) >= 0))
    .map((item) => ({
      key: item.key || item.label,
      label: item.count === undefined ? `${item.label || ''}` : `${item.label || ''} · ${item.count}`,
      type: item.level || item.type || props.defaultType,
      raw: item
    }))
)
</script>

<style scoped>
.group-tag-list {
  margin-top: 10px;
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.group-tag-item {
  cursor: pointer;
}
</style>
