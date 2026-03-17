<template>
  <div class="batch-action-bar">
    <div class="batch-action-left">
      <span class="batch-action-title">{{ title }}</span>
      <div v-if="normalizedTags.length" class="batch-action-tags">
        <el-tag
          v-for="tag in normalizedTags"
          :key="`${tag.type}-${tag.label}`"
          :type="tag.type"
          :effect="tag.effect"
        >
          {{ tag.label }}
        </el-tag>
      </div>
    </div>
    <div v-if="normalizedActions.length" class="batch-action-right">
      <el-button
        v-for="action in normalizedActions"
        :key="action.key"
        size="small"
        :type="action.type"
        :plain="action.plain"
        :loading="action.loading"
        :disabled="action.disabled"
        @click="$emit('action', action.key)"
      >
        {{ action.label }}
      </el-button>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  title: {
    type: String,
    default: ''
  },
  tags: {
    type: Array,
    default: () => []
  },
  actions: {
    type: Array,
    default: () => []
  }
})

defineEmits(['action'])

const normalizedTags = computed(() =>
  (Array.isArray(props.tags) ? props.tags : []).map((item) => ({
    label: String(item?.label || ''),
    type: item?.type || 'info',
    effect: item?.effect || 'light'
  }))
)

const normalizedActions = computed(() =>
  (Array.isArray(props.actions) ? props.actions : []).map((item) => ({
    key: String(item?.key || ''),
    label: String(item?.label || ''),
    type: item?.type || undefined,
    plain: item?.plain !== false,
    loading: Boolean(item?.loading),
    disabled: Boolean(item?.disabled)
  }))
)
</script>

<style scoped>
.batch-action-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 10px;
}

.batch-action-left {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.batch-action-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.batch-action-tags,
.batch-action-right {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

@media (max-width: 1100px) {
  .batch-action-bar {
    align-items: flex-start;
    flex-direction: column;
  }

  .batch-action-right {
    width: 100%;
  }
}
</style>
