<template>
  <div class="module-manager">
    <!-- 页头 -->
    <div class="page-header">
      <div>
        <h2 class="title">模块管理</h2>
        <p class="subtitle">查看模块运行状态，支持热重载与在线更新</p>
      </div>
      <div class="header-actions">
        <Button :icon="RefreshCw" :loading="loading" @click="fetchAll">刷新</Button>
        <Button variant="success" :icon="Play" :loading="loading" @click="controlAll('start')">全部启动</Button>
        <Button variant="danger" :icon="Square" :loading="loading" @click="controlAll('stop')">全部停止</Button>
      </div>
    </div>

    <!-- 错误提示 -->
    <div v-if="error" class="error-banner">
      {{ error }}
      <Button size="sm" variant="danger" @click="error = ''">关闭</Button>
    </div>

    <!-- 模块列表 -->
    <div class="section-title">
      <Cpu :size="16" />
      运行中的模块
    </div>

    <div class="card" v-if="loading && modules.length === 0">
      <div class="empty-state">正在加载模块信息...</div>
    </div>
    <div class="card" v-else-if="modules.length === 0">
      <div class="empty-state">暂无模块数据，请确认 Bot 正在运行。</div>
    </div>
    <div class="modules-grid" v-else>
      <div
        v-for="mod in modules"
        :key="mod.name"
        class="module-card"
        :class="statusClass(mod.status)"
      >
        <div class="module-top">
          <div class="module-info">
            <div class="module-name">{{ mod.name }}</div>
            <div class="module-version mono">v{{ mod.version }}</div>
          </div>
          <span class="status-dot" :class="statusDotClass(mod.status)"></span>
        </div>
        <div class="module-status-text">{{ statusLabel(mod.status) }}</div>
        <div class="module-error" v-if="mod.error">{{ mod.error }}</div>
        <div class="module-actions">
          <Button size="sm" variant="success" :icon="Play" @click="moduleAction(mod.name, 'start')" :disabled="mod.status === 'running'">启动</Button>
          <Button size="sm" variant="danger" :icon="Square" @click="moduleAction(mod.name, 'stop')" :disabled="mod.status === 'stopped'">停止</Button>
          <Button size="sm" variant="secondary" :icon="RefreshCw" @click="moduleAction(mod.name, 'reload')">重载</Button>
        </div>
      </div>
    </div>

    <!-- 热更新 -->
    <div class="section-title" style="margin-top: 8px;">
      <Download :size="16" />
      热更新队列
    </div>

    <div class="card">
      <div v-if="updates.length === 0" class="empty-state">
        <CheckCircle :size="32" style="color: #10b981; margin-bottom: 8px;" />
        <div>所有模块均为最新版本，无待更新项目。</div>
      </div>
      <div class="table-container" v-else>
        <table class="update-table">
          <thead>
            <tr>
              <th>模块名</th>
              <th>当前版本</th>
              <th>最新版本</th>
              <th>描述</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="upd in updates" :key="upd.module">
              <td class="font-semibold">{{ upd.module }}</td>
              <td class="mono text-secondary">{{ upd.current_version || '-' }}</td>
              <td class="mono text-accent">{{ upd.new_version || '-' }}</td>
              <td class="text-secondary">{{ upd.description || '-' }}</td>
              <td>
                <Button
                  size="sm"
                  variant="primary"
                  :icon="Download"
                  :loading="applyingUpdate === upd.module"
                  @click="applyUpdate(upd.module)"
                >
                  应用更新
                </Button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { RefreshCw, Play, Square, Download, Cpu, CheckCircle } from 'lucide-vue-next'
import Button from '../components/Button.vue'

// ─── 类型 ─────────────────────────────────────────────────────────────────────

interface ModuleInfo {
  name: string
  version: string
  status: string   // "running" | "stopped" | "error" | "loading"
  error?: string
}

interface UpdateInfo {
  module: string
  current_version?: string
  new_version?: string
  description?: string
}

// ─── 状态 ─────────────────────────────────────────────────────────────────────

const modules = ref<ModuleInfo[]>([])
const updates = ref<UpdateInfo[]>([])
const loading = ref(false)
const error = ref('')
const applyingUpdate = ref('')
let pollTimer: number | null = null

// ─── 数据获取 ─────────────────────────────────────────────────────────────────

const fetchModules = async () => {
  try {
    const res = await fetch('/api/v1/modules')
    if (res.ok) {
      const data = await res.json()
      modules.value = Array.isArray(data.modules) ? data.modules : []
    }
  } catch (e) {
    // 静默失败，避免抖动
  }
}

const fetchUpdates = async () => {
  try {
    const res = await fetch('/api/v1/updates')
    if (res.ok) {
      const data = await res.json()
      updates.value = Array.isArray(data.updates) ? data.updates : []
    }
  } catch (e) {
    // 更新接口不是必需的，静默忽略
  }
}

const fetchAll = async () => {
  loading.value = true
  error.value = ''
  try {
    await Promise.all([fetchModules(), fetchUpdates()])
  } catch (e: any) {
    error.value = `数据加载失败: ${e.message}`
  } finally {
    loading.value = false
  }
}

// ─── 模块操作 ─────────────────────────────────────────────────────────────────

const moduleAction = async (name: string, action: 'start' | 'stop' | 'reload') => {
  try {
    const res = await fetch(`/api/v1/modules/${encodeURIComponent(name)}/${action}`, {
      method: 'POST',
    })
    if (!res.ok) {
      const d = await res.json().catch(() => ({}))
      error.value = `操作失败 [${name}/${action}]: ${d.error || res.statusText}`
    } else {
      // 延迟刷新，等后端状态稳定
      setTimeout(fetchModules, 600)
    }
  } catch (e: any) {
    error.value = `网络错误: ${e.message}`
  }
}

const controlAll = async (action: 'start' | 'stop') => {
  loading.value = true
  error.value = ''
  try {
    const res = await fetch('/api/process/control', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ action }),
    })
    if (!res.ok) {
      const d = await res.json().catch(() => ({}))
      error.value = `批量操作失败: ${d.error || res.statusText}`
    } else {
      setTimeout(fetchModules, 800)
    }
  } catch (e: any) {
    error.value = `网络错误: ${e.message}`
  } finally {
    loading.value = false
  }
}

// ─── 热更新 ───────────────────────────────────────────────────────────────────

const applyUpdate = async (moduleName: string) => {
  applyingUpdate.value = moduleName
  try {
    const res = await fetch(`/api/v1/updates/${encodeURIComponent(moduleName)}`, {
      method: 'POST',
    })
    if (res.ok) {
      // 更新成功后刷新列表
      await fetchAll()
    } else {
      const d = await res.json().catch(() => ({}))
      error.value = `更新失败 [${moduleName}]: ${d.error || res.statusText}`
    }
  } catch (e: any) {
    error.value = `网络错误: ${e.message}`
  } finally {
    applyingUpdate.value = ''
  }
}

// ─── 工具函数 ─────────────────────────────────────────────────────────────────

function statusLabel(s: string): string {
  const map: Record<string, string> = {
    running: '运行中',
    stopped: '已停止',
    error: '出错',
    loading: '启动中...',
  }
  return map[s] || s
}

function statusClass(s: string): string {
  if (s === 'running') return 'module-card--running'
  if (s === 'error') return 'module-card--error'
  return 'module-card--stopped'
}

function statusDotClass(s: string): string {
  if (s === 'running') return 'dot-green'
  if (s === 'error') return 'dot-red'
  return 'dot-gray'
}

// ─── 生命周期 ─────────────────────────────────────────────────────────────────

onMounted(() => {
  fetchAll()
  // 每 5 秒自动刷新模块状态
  pollTimer = window.setInterval(fetchModules, 5000)
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})
</script>

<style scoped>
.module-manager {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.page-header {
  display: flex;
  align-items: center;
  gap: 16px;
  background: var(--bg-card);
  padding: 20px;
  border-radius: 16px;
  border: 1px solid var(--border-color);
}

.page-header > div:first-child {
  flex: 1;
}

.header-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.title {
  margin: 0;
  font-size: 20px;
  font-weight: 700;
}

.subtitle {
  margin: 4px 0 0 0;
  font-size: 13px;
  opacity: 0.7;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 600;
  color: var(--text-secondary);
  padding: 0 4px;
}

.card {
  background: var(--bg-card);
  border-radius: 16px;
  border: 1px solid var(--border-color);
  overflow: hidden;
}

.empty-state {
  padding: 40px;
  text-align: center;
  color: var(--text-secondary);
  display: flex;
  flex-direction: column;
  align-items: center;
}

/* ── 模块卡片网格 ── */
.modules-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 12px;
}

.module-card {
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: 14px;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  transition: border-color 0.2s;
}

.module-card--running {
  border-color: rgba(16, 185, 129, 0.35);
}

.module-card--error {
  border-color: rgba(239, 68, 68, 0.35);
}

.module-card--stopped {
  opacity: 0.75;
}

.module-top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
}

.module-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.module-name {
  font-size: 15px;
  font-weight: 700;
}

.module-version {
  font-size: 11px;
  opacity: 0.6;
}

.status-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  margin-top: 4px;
  flex-shrink: 0;
}

.dot-green {
  background: #10b981;
  box-shadow: 0 0 6px rgba(16, 185, 129, 0.7);
}

.dot-red {
  background: #ef4444;
  box-shadow: 0 0 6px rgba(239, 68, 68, 0.7);
}

.dot-gray {
  background: rgba(156, 163, 175, 0.5);
}

.module-status-text {
  font-size: 12px;
  opacity: 0.7;
}

.module-error {
  font-size: 12px;
  color: #f87171;
  background: rgba(239, 68, 68, 0.1);
  padding: 6px 8px;
  border-radius: 6px;
}

.module-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

/* ── 更新表格 ── */
.table-container {
  overflow-x: auto;
}

.update-table {
  width: 100%;
  border-collapse: collapse;
}

.update-table th,
.update-table td {
  padding: 12px 16px;
  text-align: left;
  border-bottom: 1px solid var(--border-color);
}

.update-table th {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-secondary);
}

.update-table td {
  font-size: 14px;
}

.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 13px;
}

.font-semibold {
  font-weight: 600;
}

.text-secondary {
  color: var(--text-secondary);
}

.text-accent {
  color: #7c5cff;
}

.error-banner {
  background: rgba(239, 68, 68, 0.15);
  border: 1px solid rgba(239, 68, 68, 0.3);
  color: #fca5a5;
  padding: 12px 16px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
</style>
