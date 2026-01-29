<template>
  <span :class="['apple-badge', `apple-badge-${type}`]">
    <slot></slot>
  </span>
</template>

<script setup>
defineProps({
  type: {
    type: String,
    default: 'primary',
    validator: (value) => ['primary', 'success', 'warning', 'danger', 'info'].includes(value)
  }
})
</script>

<style scoped>
.apple-badge {
  display: inline-flex;
  align-items: center;
  padding: 4px 12px;
  border-radius: var(--radius-full);
  font-size: 12px;
  font-weight: 500;
  white-space: nowrap;
}

.apple-badge-primary {
  background: rgba(10, 132, 255, 0.2);
  color: var(--apple-accent);
}

.apple-badge-success {
  background: rgba(48, 209, 88, 0.2);
  color: var(--apple-success);
}

.apple-badge-warning {
  background: rgba(255, 159, 10, 0.2);
  color: var(--apple-warning);
}

.apple-badge-danger {
  background: rgba(255, 69, 58, 0.2);
  color: var(--apple-danger);
}

.apple-badge-info {
  background: rgba(255, 255, 255, 0.1);
  color: var(--apple-text-secondary);
}
</style>
