<template>
  <div class="apple-input-wrapper">
    <label v-if="label" class="apple-input-label">{{ label }}</label>
    <div class="apple-input-container">
      <i v-if="icon" :class="['apple-input-icon', icon]"></i>
      <input
        :type="type"
        :value="modelValue"
        :placeholder="placeholder"
        :disabled="disabled"
        class="apple-input"
        @input="$emit('update:modelValue', $event.target.value)"
        @focus="$emit('focus', $event)"
        @blur="$emit('blur', $event)"
      />
    </div>
  </div>
</template>

<script setup>
defineProps({
  modelValue: [String, Number],
  type: {
    type: String,
    default: 'text'
  },
  label: String,
  placeholder: String,
  icon: String,
  disabled: Boolean
})

defineEmits(['update:modelValue', 'focus', 'blur'])
</script>

<style scoped>
.apple-input-wrapper {
  width: 100%;
}

.apple-input-label {
  display: block;
  margin-bottom: var(--space-sm);
  color: var(--apple-text-secondary);
  font-size: 14px;
  font-weight: 500;
}

.apple-input-container {
  position: relative;
}

.apple-input-icon {
  position: absolute;
  left: 16px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--apple-text-tertiary);
}

.apple-input {
  width: 100%;
  padding: 12px 16px;
  padding-left: 44px;
  background: var(--apple-bg-tertiary);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: var(--radius-md);
  color: var(--apple-text-primary);
  font-family: var(--apple-font);
  font-size: 15px;
  transition: all 0.3s var(--ease-standard);
}

.apple-input:not(:has(+ .apple-input-icon)) {
  padding-left: 16px;
}

.apple-input:focus {
  outline: none;
  border-color: var(--apple-accent);
  background: var(--apple-bg-elevated);
  box-shadow: 0 0 0 4px rgba(10, 132, 255, 0.1);
}

.apple-input::placeholder {
  color: var(--apple-text-tertiary);
}

.apple-input:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
