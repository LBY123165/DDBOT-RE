<template>
  <div
    class="select"
    :class="[
      `select--${size}`,
      {
        'select--open': open,
        'select--disabled': disabled,
      },
    ]"
  >
    <div
      class="select__trigger"
      @click="toggleOpen"
      :disabled="disabled"
    >
      <span class="select__value">{{ displayValue }}</span>
      <div class="select__arrow">▼</div>
    </div>
    <div v-if="open" class="select__menu">
      <slot></slot>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'

const props = withDefaults(
  defineProps<{
    modelValue: string | number
    size?: 'sm' | 'md' | 'lg'
    disabled?: boolean
  }>(),
  {
    size: 'md',
    disabled: false,
  }
)

const emit = defineEmits<{
  'update:modelValue': [value: string | number]
  change: [value: string | number]
}>()

const open = ref(false)
const options = ref<any[]>([])

const displayValue = computed(() => {
  const option = options.value.find(opt => opt.value === props.modelValue)
  return option ? option.label : String(props.modelValue)
})

function toggleOpen() {
  if (!props.disabled) {
    open.value = !open.value
  }
}

function selectValue(value: string | number) {
  emit('update:modelValue', value)
  emit('change', value)
  open.value = false
}

function handleClickOutside(event: MouseEvent) {
  const target = event.target as HTMLElement
  if (!target.closest('.select')) {
    open.value = false
  }
}

// Register click outside listener
onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})

// Expose selectValue to Option components
defineExpose({
  selectValue,
  registerOption: (option: any) => {
    options.value.push(option)
  },
  unregisterOption: (value: any) => {
    options.value = options.value.filter(opt => opt.value !== value)
  },
})
</script>

<style scoped>
.select {
  position: relative;
  display: inline-block;
}

.select__trigger {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 8px;
  background: var(--bg-card);
  color: var(--text-primary);
  cursor: pointer;
  transition: all 0.2s ease;
  min-width: 120px;
}

.select--sm .select__trigger {
  padding: 6px 10px;
  font-size: 12px;
}

.select--md .select__trigger {
  padding: 8px 12px;
  font-size: 13px;
}

.select--lg .select__trigger {
  padding: 10px 14px;
  font-size: 14px;
}

.select__trigger:hover:not(.select--disabled) {
  border-color: rgba(124, 92, 255, 0.3);
  background: var(--border-color);
}

.select--open .select__trigger {
  border-color: rgba(124, 92, 255, 0.5);
  background: var(--border-color);
}

.select--disabled .select__trigger {
  opacity: 0.5;
  cursor: not-allowed;
}

.select__value {
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.select__arrow {
  font-size: 10px;
  opacity: 0.6;
  transition: transform 0.2s ease;
}

.select--open .select__arrow {
  transform: rotate(180deg);
}

.select__menu {
  position: absolute;
  top: 100%;
  left: 0;
  right: 0;
  margin-top: 4px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 8px;
  background: rgba(11, 16, 32, 0.95);
  backdrop-filter: blur(10px);
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.3);
  z-index: 100;
  max-height: 200px;
  overflow-y: auto;
}

/* Option styles */
.select__option {
  padding: 8px 12px;
  cursor: pointer;
  transition: background 0.2s ease;
}

.select__option:hover {
  background: rgba(124, 92, 255, 0.1);
}

.select__option--selected {
  background: rgba(124, 92, 255, 0.2);
  font-weight: 500;
}

/* Scrollbar for select menu */
.select__menu::-webkit-scrollbar {
  width: 6px;
}

.select__menu::-webkit-scrollbar-track {
  background: var(--bg-card);
  border-radius: 3px;
}

.select__menu::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.2);
  border-radius: 3px;
}

.select__menu::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.3);
}
</style>