<template>
  <div class="apple-select-wrapper">
    <label v-if="label" class="apple-select-label">{{ label }}</label>
    <select
      :value="modelValue"
      @change="$emit('update:modelValue', $event.target.value)"
      class="apple-select"
      :disabled="disabled"
    >
      <option v-if="placeholder" value="" disabled>{{ placeholder }}</option>
      <option
        v-for="option in options"
        :key="option.value"
        :value="option.value"
      >
        {{ option.label }}
      </option>
    </select>
  </div>
</template>

<script setup>
defineProps({
  modelValue: [String, Number],
  label: String,
  placeholder: String,
  options: {
    type: Array,
    default: () => []
  },
  disabled: Boolean
})

defineEmits(['update:modelValue'])
</script>

<style scoped>
.apple-select-wrapper {
  width: 100%;
}

.apple-select-label {
  display: block;
  margin-bottom: var(--space-sm);
  color: var(--apple-text-secondary);
  font-size: 14px;
  font-weight: 500;
}

.apple-select {
  width: 100%;
  padding: 12px 16px;
  background: var(--apple-bg-tertiary);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: var(--radius-md);
  color: var(--apple-text-primary);
  font-family: var(--apple-font);
  font-size: 15px;
  cursor: pointer;
  transition: all 0.3s var(--ease-standard);
  appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 12 12'%3E%3Cpath fill='%23ffffff' d='M6 9L1 4h10z'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 16px center;
  padding-right: 40px;
}

.apple-select:focus {
  outline: none;
  border-color: var(--apple-accent);
  background: var(--apple-bg-elevated);
  box-shadow: 0 0 0 4px rgba(10, 132, 255, 0.1);
}

.apple-select:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.apple-select option {
  background: var(--apple-bg-tertiary);
  color: var(--apple-text-primary);
}
</style>
