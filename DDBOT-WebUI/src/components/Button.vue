<template>
  <button
    class="btn"
    :class="[
      `btn--${variant}`,
      `btn--${size}`,
      {
        'btn--loading': loading,
        'btn--disabled': disabled || loading,
      },
    ]"
    :disabled="disabled || loading"
    @click="handleClick"
  >
    <div v-if="loading" class="btn__spinner" />
    <component v-else-if="icon" :is="icon" :size="iconSize" class="btn__icon" />
    <span v-if="$slots.default" class="btn__text"><slot /></span>
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    variant?: 'primary' | 'secondary' | 'danger' | 'success'
    size?: 'sm' | 'md' | 'lg'
    disabled?: boolean
    loading?: boolean
    icon?: any
  }>(),
  {
    variant: 'secondary',
    size: 'md',
  }
)

const emit = defineEmits<{
  click: [event: MouseEvent]
}>()

const iconSize = computed(() => {
  const sizes = { sm: 14, md: 16, lg: 18 }
  return sizes[props.size]
})

function handleClick(event: MouseEvent) {
  if (!props.disabled && !props.loading) {
    emit('click', event)
  }
}
</script>

<style scoped>
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border-radius: 10px;
  border: 1px solid transparent;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
  position: relative;
}

.btn--sm {
  padding: 6px 12px;
  font-size: 12px;
}

.btn--md {
  padding: 10px 16px;
  font-size: 13px;
}

.btn--lg {
  padding: 12px 20px;
  font-size: 14px;
}

.btn--primary {
  border-color: rgba(124, 92, 255, 0.35);
  background: rgba(124, 92, 255, 0.2);
  color: var(--text-primary);
}

.btn--primary:hover:not(.btn--disabled) {
  background: rgba(124, 92, 255, 0.3);
}

.btn--secondary {
  border-color: rgba(255, 255, 255, 0.1);
  background: var(--bg-card);
  color: var(--text-primary);
}

.btn--secondary:hover:not(.btn--disabled) {
  background: var(--border-color);
}

.btn--danger {
  border-color: rgba(239, 68, 68, 0.35);
  background: rgba(239, 68, 68, 0.2);
  color: var(--text-primary);
}

.btn--danger:hover:not(.btn--disabled) {
  background: rgba(239, 68, 68, 0.3);
}

.btn--success {
  border-color: rgba(16, 185, 129, 0.35);
  background: rgba(16, 185, 129, 0.2);
  color: var(--text-primary);
}

.btn--success:hover:not(.btn--disabled) {
  background: rgba(16, 185, 129, 0.3);
}

.btn--disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn__spinner {
  width: 14px;
  height: 14px;
  border: 2px solid currentColor;
  border-top-color: transparent;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
