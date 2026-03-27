<template>
  <div
    class="select__option"
    :class="{
      'select__option--selected': isSelected,
    }"
    @click="select"
  >
    <slot></slot>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue'

const props = defineProps<{
  value: string | number
  label?: string
}>()

const emit = defineEmits<{
  click: [value: string | number]
}>()

const parent = computed(() => {
  // Get the parent Select component
  let el = (document.querySelector('.select') as any)
  return el ? el.__vueParentComponent : null
})

const isSelected = computed(() => {
  // Check if this option is selected
  return parent.value?.props.modelValue === props.value
})

function select() {
  // Notify parent Select component to select this value
  if (parent.value?.exposed?.selectValue) {
    parent.value.exposed.selectValue(props.value)
  }
  emit('click', props.value)
}

// Register this option with parent Select component
onMounted(() => {
  if (parent.value?.exposed?.registerOption) {
    parent.value.exposed.registerOption({
      value: props.value,
      label: props.label || props.value.toString(),
    })
  }
})

onUnmounted(() => {
  if (parent.value?.exposed?.unregisterOption) {
    parent.value.exposed.unregisterOption(props.value)
  }
})
</script>

<style scoped>
/* Option styles are defined in Select.vue */
</style>