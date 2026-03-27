<template>
  <input
    class="input"
    :class="[
      `input--${size}`,
      {
        'input--error': error,
        'input--disabled': disabled,
      },
    ]"
    :type="type"
    :value="modelValue"
    :placeholder="placeholder"
    :disabled="disabled"
    @input="handleInput"
    @focus="handleFocus"
    @blur="handleBlur"
  />
</template>

<script setup lang="ts">
const props = withDefaults(
  defineProps<{
    modelValue: string | number | any
    type?: string
    size?: 'sm' | 'md' | 'lg'
    placeholder?: string
    disabled?: boolean
    error?: boolean
  }>(),
  {
    type: 'text',
    size: 'md',
  }
)

const emit = defineEmits<{
  'update:modelValue': [value: string]
  focus: [event: FocusEvent]
  blur: [event: FocusEvent]
}>()

function handleInput(event: Event) {
  const target = event.target as HTMLInputElement
  emit('update:modelValue', target.value)
}

function handleFocus(event: FocusEvent) {
  emit('focus', event)
}

function handleBlur(event: FocusEvent) {
  emit('blur', event)
}
</script>

<style scoped>
.input {
  padding: 8px 12px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 8px;
  background: var(--bg-card);
  color: var(--text-primary);
  font-size: 13px;
  outline: none;
  transition: all 0.2s ease;
}

.input:focus {
  border-color: rgba(124, 92, 255, 0.5);
  background: var(--border-color);
}

.input--error {
  border-color: rgba(239, 68, 68, 0.5);
}

.input--disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.input--sm {
  padding: 6px 10px;
  font-size: 12px;
}

.input--md {
  padding: 8px 12px;
  font-size: 13px;
}

.input--lg {
  padding: 10px 14px;
  font-size: 14px;
}

.input::placeholder {
  color: var(--text-secondary);
}
</style>