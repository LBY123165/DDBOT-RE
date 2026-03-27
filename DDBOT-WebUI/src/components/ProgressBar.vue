<template>
  <div class="progress-container" :class="{ 'progress-container--active': active }">
    <div class="progress-header">
      <div class="progress-title">{{ title }}</div>
      <div class="progress-percentage">{{ Math.round(percentage) }}%</div>
    </div>
    
    <div class="progress-bar">
      <div 
        class="progress-fill" 
        :style="{ width: percentage + '%' }"
        :class="progressClass"
      ></div>
      
      <!-- 渐变动画效果 -->
      <div 
        class="progress-shine" 
        :style="{ 
          left: percentage + '%',
          opacity: active ? 1 : 0 
        }"
      ></div>
    </div>
    
    <div class="progress-status">
      <div class="status-text">{{ status }}</div>
      <div v-if="speed" class="speed-indicator">{{ speed }}/s</div>
    </div>
    
    <!-- 详细步骤显示 -->
    <div v-if="steps.length > 0" class="progress-steps">
      <div 
        v-for="(step, index) in steps" 
        :key="index"
        class="step"
        :class="{ 
          'step--completed': step.completed,
          'step--active': step.active,
          'step--pending': !step.completed && !step.active
        }"
      >
        <div class="step-icon">
          <Check v-if="step.completed" :size="14" />
          <Loader v-else-if="step.active" :size="14" class="spin" />
          <div v-else class="step-dot"></div>
        </div>
        <div class="step-text">{{ step.text }}</div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Check, Loader } from 'lucide-vue-next'

interface ProgressStep {
  text: string
  completed: boolean
  active: boolean
}

interface Props {
  percentage?: number
  title?: string
  status?: string
  speed?: string
  active?: boolean
  steps?: ProgressStep[]
}

const props = withDefaults(defineProps<Props>(), {
  percentage: 0,
  title: '进度',
  status: '准备中...',
  speed: '',
  active: false,
  steps: () => []
})

const progressClass = computed(() => {
  if (props.percentage < 30) return 'progress-fill--slow'
  if (props.percentage < 70) return 'progress-fill--medium'
  if (props.percentage < 100) return 'progress-fill--fast'
  return 'progress-fill--complete'
})
</script>

<style scoped>
.progress-container {
  background: var(--bg-card);
  border-radius: 12px;
  padding: 16px;
  border: 1px solid var(--border-color);
  transition: all 0.3s ease;
}

.progress-container--active {
  border-color: rgba(124, 92, 255, 0.3);
  box-shadow: 0 0 20px rgba(124, 92, 255, 0.1);
}

.progress-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.progress-title {
  font-weight: 600;
  font-size: 14px;
  color: rgba(232, 234, 240, 0.95);
}

.progress-percentage {
  font-weight: 700;
  font-size: 14px;
  color: #7c5cff;
}

.progress-bar {
  height: 8px;
  background: var(--bg-card);
  border-radius: 4px;
  overflow: hidden;
  position: relative;
  margin-bottom: 12px;
}

.progress-fill {
  height: 100%;
  border-radius: 4px;
  transition: width 0.3s ease, background 0.3s ease;
  position: relative;
}

.progress-fill--slow {
  background: linear-gradient(90deg, #3b82f6, #60a5fa);
}

.progress-fill--medium {
  background: linear-gradient(90deg, #f59e0b, #fbbf24);
}

.progress-fill--fast {
  background: linear-gradient(90deg, #10b981, #34d399);
}

.progress-fill--complete {
  background: linear-gradient(90deg, #8b5cf6, #a78bfa);
}

.progress-shine {
  position: absolute;
  top: 0;
  width: 20px;
  height: 100%;
  background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.4), transparent);
  transform: skewX(-25deg);
  transition: opacity 0.3s ease;
}

.progress-status {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 12px;
  opacity: 0.8;
  margin-bottom: 16px;
}

.status-text {
  color: var(--text-secondary);
}

.speed-indicator {
  color: #7c5cff;
  font-weight: 500;
}

.progress-steps {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.step {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 12px;
  border-radius: 8px;
  transition: all 0.2s ease;
}

.step--completed {
  background: rgba(16, 185, 129, 0.1);
}

.step--active {
  background: rgba(124, 92, 255, 0.1);
}

.step--pending {
  opacity: 0.5;
}

.step-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  flex-shrink: 0;
}

.step--completed .step-icon {
  background: #10b981;
  color: var(--text-primary);
}

.step--active .step-icon {
  background: #7c5cff;
  color: var(--text-primary);
}

.step--pending .step-icon {
  background: rgba(255, 255, 255, 0.1);
}

.step-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: currentColor;
}

.step-text {
  font-size: 13px;
  color: var(--text-primary);
}

.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}
</style>