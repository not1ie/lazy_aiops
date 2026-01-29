<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="modelValue" class="apple-modal-overlay" @click="handleOverlayClick">
        <div class="apple-modal glass" @click.stop>
          <div class="apple-modal-header">
            <h3>{{ title }}</h3>
            <button class="close-btn" @click="handleClose">
              <i class="fas fa-times"></i>
            </button>
          </div>
          
          <div class="apple-modal-body">
            <slot></slot>
          </div>
          
          <div v-if="$slots.footer" class="apple-modal-footer">
            <slot name="footer"></slot>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
const props = defineProps({
  modelValue: Boolean,
  title: String,
  closeOnOverlay: {
    type: Boolean,
    default: true
  }
})

const emit = defineEmits(['update:modelValue', 'close'])

const handleClose = () => {
  emit('update:modelValue', false)
  emit('close')
}

const handleOverlayClick = () => {
  if (props.closeOnOverlay) {
    handleClose()
  }
}
</script>

<style scoped>
.apple-modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: var(--space-lg);
}

.apple-modal {
  background: var(--apple-bg-secondary);
  border-radius: var(--radius-xl);
  max-width: 600px;
  width: 100%;
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  border: 1px solid rgba(255, 255, 255, 0.1);
}

.apple-modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--space-lg);
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}

.apple-modal-header h3 {
  font-size: 18px;
  font-weight: 600;
  color: var(--apple-text-primary);
  margin: 0;
}

.close-btn {
  background: transparent;
  border: none;
  color: var(--apple-text-secondary);
  cursor: pointer;
  padding: var(--space-sm);
  border-radius: var(--radius-sm);
  transition: all 0.2s var(--ease-standard);
  font-size: 18px;
}

.close-btn:hover {
  background: rgba(255, 255, 255, 0.05);
  color: var(--apple-text-primary);
}

.apple-modal-body {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-lg);
}

.apple-modal-footer {
  padding: var(--space-lg);
  border-top: 1px solid rgba(255, 255, 255, 0.05);
  display: flex;
  justify-content: flex-end;
  gap: var(--space-sm);
}

.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.3s var(--ease-standard);
}

.modal-enter-active .apple-modal,
.modal-leave-active .apple-modal {
  transition: transform 0.3s var(--ease-spring);
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-from .apple-modal,
.modal-leave-to .apple-modal {
  transform: scale(0.9);
}
</style>
