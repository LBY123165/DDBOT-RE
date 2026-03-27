<template>
  <div class="status-card">
    <div class="status-card__header">
      <div class="status-card__title">{{ title }}</div>
      <div class="status-card__status" :class="`status-card__status--${status}`">
        <div v-if="loading" class="status-card__spinner" />
        <component v-else :is="statusIcon" :size="16" />
        <span>{{ displayStatusText }}</span>
      </div>
    </div>
    <div v-if="slots.default" class="status-card__content">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Check, X } from 'lucide-vue-next'

const props = withDefaults(
  defineProps<{
    title: string
    status: 'online' | 'offline' | 'unknown' | 'loading'
    statusText?: string
    loading?: boolean
  }>(),
  {}
)

const slots = defineSlots()

const statusIcons: Record<string, any> = {
  online: Check,
  offline: X,
  unknown: X,
}

const statusTexts: Record<string, string> = {
  online: '在线',
  offline: '离线',
  unknown: '未知',
  loading: '加载中',
}

const statusIcon = computed(() => statusIcons[props.status] || X)
const displayStatusText = computed(() => props.statusText || statusTexts[props.status] || '未知')
</script>

<style scoped>
.status-card {
  border-radius: 14px;
  border: 1px solid var(--border-color);
  background: var(--bg-card);
  padding: 16px;
  transition: all 0.2s ease;
}

.status-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}

.status-card__title {
  font-size: 14px;
  font-weight: 600;
  opacity: 0.9;
}

.status-card__status {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 600;
}

.status-card__status--online {
  background: rgba(16, 185, 129, 0.15);
  color: #10b981;
  border: 1px solid rgba(16, 185, 129, 0.3);
}

.status-card__status--offline {
  background: rgba(239, 68, 68, 0.15);
  color: #ef4444;
  border: 1px solid rgba(239, 68, 68, 0.3);
}

.status-card__status--unknown {
  background: rgba(156, 163, 175, 0.15);
  color: #9ca3af;
  border: 1px solid rgba(156, 163, 175, 0.3);
}

.status-card__status--loading {
  background: rgba(124, 92, 255, 0.15);
  color: #7c5cff;
  border: 1px solid rgba(124, 92, 255, 0.3);
}

.status-card__spinner {
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

.status-card__content {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
</style>
