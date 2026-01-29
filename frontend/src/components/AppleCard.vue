<template>
  <div :class="['apple-card', { 'apple-card-hoverable': hoverable }]">
    <div v-if="title || $slots.extra" class="apple-card-header">
      <h3 v-if="title">{{ title }}</h3>
      <div v-if="$slots.extra" class="apple-card-extra">
        <slot name="extra"></slot>
      </div>
    </div>
    <div class="apple-card-body">
      <slot></slot>
    </div>
  </div>
</template>

<script setup>
defineProps({
  title: String,
  hoverable: Boolean
})
</script>

<style scoped>
.apple-card {
  background: var(--apple-bg-secondary);
  border-radius: var(--radius-lg);
  padding: var(--space-lg);
  transition: all 0.3s var(--ease-standard);
  border: 1px solid rgba(255, 255, 255, 0.05);
}

.apple-card-hoverable:hover {
  transform: translateY(-4px);
  box-shadow: var(--shadow-lg);
  border-color: rgba(255, 255, 255, 0.1);
}

.apple-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-md);
}

.apple-card-header h3 {
  font-size: 18px;
  font-weight: 600;
  color: var(--apple-text-primary);
}

.apple-card-body {
  color: var(--apple-text-secondary);
}
</style>
