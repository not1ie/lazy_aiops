<template>
  <button 
    :class="['apple-btn', `apple-btn-${type}`, { 'apple-btn-loading': loading }]"
    :disabled="disabled || loading"
    @click="handleClick"
  >
    <i v-if="icon && !loading" :class="icon"></i>
    <span v-if="loading" class="apple-spinner"></span>
    <slot></slot>
  </button>
</template>

<script setup>
const props = defineProps({
  type: {
    type: String,
    default: 'primary',
    validator: (value) => ['primary', 'secondary', 'ghost'].includes(value)
  },
  icon: String,
  loading: Boolean,
  disabled: Boolean
})

const emit = defineEmits(['click'])

const handleClick = (e) => {
  if (!props.loading && !props.disabled) {
    emit('click', e)
  }
}
</script>

<style scoped>
.apple-btn {
  padding: 10px 20px;
  border-radius: var(--radius-full);
  border: none;
  font-family: var(--apple-font);
  font-size: 15px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s var(--ease-standard);
  display: inline-flex;
  align-items: center;
  gap: 8px;
  position: relative;
  overflow: hidden;
}

.apple-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.apple-btn-primary {
  background: var(--apple-accent);
  color: white;
}

.apple-btn-primary:hover:not(:disabled) {
  background: var(--apple-accent-hover);
  transform: scale(1.02);
}

.apple-btn-secondary {
  background: var(--apple-bg-tertiary);
  color: var(--apple-text-primary);
}

.apple-btn-secondary:hover:not(:disabled) {
  background: var(--apple-bg-elevated);
}

.apple-btn-ghost {
  background: transparent;
  color: var(--apple-accent);
  border: 1px solid var(--apple-accent);
}

.apple-btn-ghost:hover:not(:disabled) {
  background: rgba(10, 132, 255, 0.1);
}

.apple-spinner {
  width: 16px;
  height: 16px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-top-color: white;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
