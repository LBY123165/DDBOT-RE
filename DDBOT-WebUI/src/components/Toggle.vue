<template>
  <div
    class="toggle"
    :class="{
      'toggle--checked': modelValue,
      'toggle--disabled': disabled,
    }"
    :disabled="disabled"
    @click="handleClick"
  >
    <div class="toggle__track">
      <div class="toggle__thumb"></div>
    </div>
  </div>
</template>

<script setup lang="ts">
const props = withDefaults(
  defineProps<{
    modelValue: boolean
    disabled?: boolean
  }>(),
  {
    disabled: false,
  }
)

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  change: [value: boolean]
}>()

function handleClick() {
  if (!props.disabled) {
    const newValue = !props.modelValue
    emit('update:modelValue', newValue)
    emit('change', newValue)
  }
}
</script>

<style scoped>
.toggle {
  position: relative;
  display: inline-flex;
  align-items: center;
  cursor: pointer;
  transition: all 0.2s ease;
}

.toggle--disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.toggle__track {
  position: relative;
  width: 44px;
  height: 24px;
  background: rgba(255, 255, 255, 0.1);
  border-radius: 12px;
  transition: all 0.2s ease;
}

.toggle--checked .toggle__track {
  background: rgba(124, 92, 255, 0.5);
}

.toggle__thumb {
  position: absolute;
  top: 2px;
  left: 2px;
  width: 20px;
  height: 20px;
  background: rgba(232, 234, 240, 0.95);
  border-radius: 50%;
  transition: all 0.2s ease;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.3);
}

.toggle--checked .toggle__thumb {
  transform: translateX(20px);
}

.toggle:hover:not(.toggle--disabled) .toggle__track {
  background: var(--border-color);
}

.toggle:hover:not(.toggle--disabled).toggle--checked .toggle__track {
  background: rgba(124, 92, 255, 0.6);
}
</style>