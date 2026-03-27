<template>
  <div
    class="input-number"
    :class="[
      `input-number--${size}`,
      {
        'input-number--disabled': disabled,
      },
    ]"
  >
    <button
      class="input-number__button input-number__button--decrease"
      @click="decrease"
      :disabled="disabled || value <= min"
    >
      -
    </button>
    <input
      class="input-number__input"
      type="number"
      :value="modelValue"
      :min="min"
      :max="max"
      :step="step"
      :disabled="disabled"
      @input="handleInput"
      @focus="handleFocus"
      @blur="handleBlur"
    />
    <button
      class="input-number__button input-number__button--increase"
      @click="increase"
      :disabled="disabled || value >= max"
    >
      +
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'

const props = withDefaults(
  defineProps<{
    modelValue: number
    min?: number
    max?: number
    step?: number
    size?: 'sm' | 'md' | 'lg'
    disabled?: boolean
  }>(),
  {
    min: -Infinity,
    max: Infinity,
    step: 1,
    size: 'md',
    disabled: false,
  }
)

const emit = defineEmits<{
  'update:modelValue': [value: number]
  change: [value: number]
  focus: [event: FocusEvent]
  blur: [event: FocusEvent]
}>()

const value = ref(props.modelValue)

function handleInput(event: Event) {
  const target = event.target as HTMLInputElement
  const newValue = parseFloat(target.value) || 0
  value.value = clamp(newValue)
  emit('update:modelValue', value.value)
  emit('change', value.value)
}

function handleFocus(event: FocusEvent) {
  emit('focus', event)
}

function handleBlur(event: FocusEvent) {
  emit('blur', event)
}

function increase() {
  if (!props.disabled && value.value < props.max) {
    value.value = clamp(value.value + props.step)
    emit('update:modelValue', value.value)
    emit('change', value.value)
  }
}

function decrease() {
  if (!props.disabled && value.value > props.min) {
    value.value = clamp(value.value - props.step)
    emit('update:modelValue', value.value)
    emit('change', value.value)
  }
}

function clamp(val: number): number {
  return Math.max(props.min, Math.min(props.max, val))
}
</script>

<style scoped>
.input-number {
  display: inline-flex;
  align-items: center;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 8px;
  overflow: hidden;
}

.input-number--sm {
  height: 32px;
}

.input-number--md {
  height: 40px;
}

.input-number--lg {
  height: 48px;
}

.input-number--disabled {
  opacity: 0.5;
}

.input-number__button {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  background: var(--bg-card);
  border: none;
  color: var(--text-primary);
  cursor: pointer;
  transition: all 0.2s ease;
  font-size: 14px;
  font-weight: 600;
}

.input-number--sm .input-number__button {
  width: 28px;
  font-size: 12px;
}

.input-number--lg .input-number__button {
  width: 36px;
  font-size: 16px;
}

.input-number__button:hover:not(:disabled) {
  background: rgba(124, 92, 255, 0.1);
}

.input-number__button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.input-number__input {
  flex: 1;
  padding: 0 8px;
  border: none;
  border-left: 1px solid rgba(255, 255, 255, 0.1);
  border-right: 1px solid rgba(255, 255, 255, 0.1);
  background: var(--bg-card);
  color: var(--text-primary);
  font-size: 13px;
  outline: none;
  text-align: center;
}

.input-number--sm .input-number__input {
  font-size: 12px;
}

.input-number--lg .input-number__input {
  font-size: 14px;
}

.input-number__input:focus {
  background: var(--border-color);
}

.input-number__input::-webkit-inner-spin-button,
.input-number__input::-webkit-outer-spin-button {
  -webkit-appearance: none;
  margin: 0;
}

.input-number__input {
  -moz-appearance: textfield;
}
</style>