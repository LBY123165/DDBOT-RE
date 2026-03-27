<template>
  <div class="log-viewer">
    <div class="card">
      <div class="card__header">
        <div>
          <div class="card__title">日志查看器</div>
          <div class="card__subtitle">实时查看 DDBOT 运行日志</div>
        </div>
        <div class="actions">
          <select v-model="logLevel" class="log-level-select" @change="loadLogs">
            <option value="all">全部日志</option>
            <option value="error">错误</option>
            <option value="warn">警告</option>
            <option value="info">信息</option>
            <option value="debug">调试</option>
          </select>
          <Button :icon="RefreshCw" @click="loadLogs">刷新</Button>
          <Button :icon="Trash2" @click="clearLogs">清空</Button>
        </div>
      </div>

      <div class="log-controls">
        <label class="checkbox-container">
          <input type="checkbox" v-model="autoRefresh" @change="toggleAutoRefresh">
          <span class="checkmark"></span>
          自动刷新
        </label>
        <label class="checkbox-container">
          <input type="checkbox" v-model="wrapLines">
          <span class="checkmark"></span>
          自动换行
        </label>
        <div class="log-stats">
          显示 {{ filteredLogs.length }} 条，共 {{ logs.length }} 条
        </div>
      </div>

      <div class="log-container">
        <div 
          v-for="(log, index) in filteredLogs" 
          :key="index"
          :class="['log-entry', `log-${log.level.toLowerCase()}`]"
        >
          <span class="log-timestamp">{{ log.timestamp }}</span>
          <span class="log-level">{{ log.level }}</span>
          <span class="log-message" :class="{ 'wrap-lines': wrapLines }">{{ log.message }}</span>
        </div>
        <div v-if="filteredLogs.length === 0" class="no-logs">
          暂无日志数据
        </div>
      </div>
    </div>

    <div v-if="error" class="log-error">
      <AlertCircle :size="16" />
      <span>{{ error }}</span>
      <Button size="sm" variant="danger" @click="clearError">关闭</Button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { AlertCircle, RefreshCw, Trash2 } from 'lucide-vue-next'
import Button from '../components/Button.vue'

const API_BASE = 'http://localhost:3000/api'

interface LogEntry {
  timestamp: string
  level: string
  message: string
}

const logs = ref<LogEntry[]>([])
const logLevel = ref('all')
const autoRefresh = ref(true)
const wrapLines = ref(false)
const error = ref<string | undefined>()

let refreshInterval: number | null = null

const filteredLogs = computed(() => {
  if (logLevel.value === 'all') {
    return logs.value
  }
  return logs.value.filter(log => log.level.toLowerCase() === logLevel.value.toLowerCase())
})

async function loadLogs() {
  try {
    error.value = undefined
    const response = await fetch(`${API_BASE}/logs?lines=100`)
    if (!response.ok) throw new Error('无法加载日志')
    const data = await response.json()
    logs.value = data.logs.map((line: string) => parseLogLine(line))
  } catch (e) {
    error.value = `加载日志失败: ${e}`
  }
}

function parseLogLine(line: string): LogEntry {
  const match = line.match(/^\[(.*?)\]\s*\[(.*?)\]\s*(.*)$/)
  if (match) {
    return {
      timestamp: match[1],
      level: match[2],
      message: match[3]
    }
  }
  return {
    timestamp: '',
    level: 'INFO',
    message: line
  }
}

function clearLogs() {
  logs.value = []
}

function clearError() {
  error.value = undefined
}

function toggleAutoRefresh() {
  if (autoRefresh.value) {
    startAutoRefresh()
  } else {
    stopAutoRefresh()
  }
}

function startAutoRefresh() {
  if (refreshInterval) {
    clearInterval(refreshInterval)
  }
  refreshInterval = window.setInterval(() => {
    loadLogs()
  }, 3000) as unknown as number
}

function stopAutoRefresh() {
  if (refreshInterval) {
    clearInterval(refreshInterval)
    refreshInterval = null
  }
}

onMounted(() => {
  loadLogs()
  if (autoRefresh.value) {
    startAutoRefresh()
  }
})

onUnmounted(() => {
  stopAutoRefresh()
})
</script>

<style scoped>
.log-viewer {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.card {
  border-radius: 16px;
  border: 1px solid var(--border-color);
  background: var(--bg-card);
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.35);
  overflow: hidden;
}

.card__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 16px;
  border-bottom: 1px solid var(--border-color);
}

.card__title {
  font-weight: 700;
  font-size: 15px;
}

.card__subtitle {
  margin-top: 4px;
  font-size: 12px;
  opacity: 0.7;
}

.actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.log-level-select {
  padding: 8px 12px;
  border-radius: 10px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  background: var(--bg-card);
  color: rgba(232, 234, 240, 0.95);
  font-size: 13px;
  outline: none;
}

.log-level-select:focus {
  border-color: rgba(124, 92, 255, 0.5);
}

.log-controls {
  display: flex;
  align-items: center;
  gap: 20px;
  padding: 12px 16px;
  border-bottom: 1px solid var(--bg-card);
  background: rgba(0, 0, 0, 0.1);
  flex-wrap: wrap;
}

.checkbox-container {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  cursor: pointer;
  user-select: none;
}

.checkbox-container input {
  appearance: none;
  width: 16px;
  height: 16px;
  border: 2px solid rgba(255, 255, 255, 0.2);
  border-radius: 4px;
  background: transparent;
  position: relative;
}

.checkbox-container input:checked {
  background: #7c5cff;
  border-color: #7c5cff;
}

.checkbox-container input:checked::after {
  content: '✓';
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  color: var(--text-primary);
  font-size: 12px;
}

.log-stats {
  margin-left: auto;
  font-size: 12px;
  opacity: 0.7;
}

.log-container {
  height: 500px;
  overflow-y: auto;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
}

.log-entry {
  display: flex;
  gap: 12px;
  padding: 8px 16px;
  border-bottom: 1px solid var(--bg-card);
}

.log-entry:hover {
  background: var(--bg-card);
}

.log-timestamp {
  color: #888;
  min-width: 140px;
}

.log-level {
  font-weight: bold;
  min-width: 60px;
}

.log-error .log-level {
  color: #ef4444;
}

.log-warn .log-level {
  color: #f59e0b;
}

.log-info .log-level {
  color: #3b82f6;
}

.log-debug .log-level {
  color: #8b5cf6;
}

.log-message {
  flex: 1;
  color: var(--text-primary);
}

.log-message.wrap-lines {
  white-space: pre-wrap;
  word-break: break-word;
}

.no-logs {
  text-align: center;
  padding: 40px;
  color: rgba(232, 234, 240, 0.5);
  font-style: italic;
}

.log-error {
  position: fixed;
  bottom: 24px;
  right: 24px;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  border-radius: 10px;
  background: rgba(239, 68, 68, 0.9);
  color: var(--text-primary);
  font-size: 13px;
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.5);
  backdrop-filter: blur(10px);
  max-width: 400px;
  z-index: 1000;
}
</style>