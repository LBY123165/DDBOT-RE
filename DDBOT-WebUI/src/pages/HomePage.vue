<template>
  <div class="home">
    <div class="home__content">
      <!-- 状态概览卡片 -->
      <div class="card card--wide">
        <div class="card__header">
          <div>
            <div class="card__title">运行状态</div>
            <div class="card__subtitle">DDBOT-WSa 本体与 OneBot 连接概览</div>
          </div>
          <div class="actions">
            <Button :icon="RefreshCw" :loading="appStore.loading" @click="appStore.loadStatus">刷新</Button>
            <Button variant="primary" :icon="Download" :loading="appStore.loading" @click="handleInstall">安装/更新</Button>
            <Button variant="secondary" :icon="Settings" @click="$router.push('/settings')">设置</Button>
          </div>
        </div>

        <div class="kpis">
          <div class="kpi">
            <div class="kpi__icon">
              <Activity :size="20" />
            </div>
            <div>
              <div class="kpi__label">本体进程</div>
              <div class="kpi__value" :class="{ 'kpi__value--active': appStore.isRunning }">
                {{ appStore.isRunning ? '运行中' : '已停止' }}
              </div>
              <div v-if="appStore.isRunning" class="kpi__detail">{{ appStore.uptime }} · PID: {{ appStore.status.pid }}</div>
            </div>
          </div>

          <div class="kpi">
            <div class="kpi__icon">
              <Link :size="20" />
            </div>
            <div>
              <div class="kpi__label">OneBot 连接</div>
              <div class="kpi__value" :class="{ 'kpi__value--active': appStore.onebotStatus.connected }">
                {{ appStore.onebotStatus.connected ? '已连接' : '未连接' }}
              </div>
              <div v-if="appStore.onebotStatus.selfId" class="kpi__detail">{{ appStore.onebotStatus.selfId }}</div>
            </div>
          </div>

          <div class="kpi">
            <div class="kpi__icon">
              <Database :size="20" />
            </div>
            <div>
              <div class="kpi__label">订阅统计</div>
              <div class="kpi__value">{{ appStore.subsSummary.total }}</div>
              <div class="kpi__detail">活跃: {{ appStore.subsSummary.active }} · 暂停: {{ appStore.subsSummary.paused }}</div>
            </div>
          </div>

          <div class="kpi">
            <div class="kpi__icon">
              <Tag :size="20" />
            </div>
            <div>
              <div class="kpi__label">当前版本</div>
              <div class="kpi__value">{{ appStore.version }}</div>
              <div class="kpi__detail mono">{{ appStore.platform }}</div>
            </div>
          </div>



          <div class="kpi">
            <div class="kpi__icon">
              <Clock :size="20" />
            </div>
            <div>
              <div class="kpi__label">上次更新检查</div>
              <div class="kpi__value">{{ appStore.lastUpdateCheck || '从未' }}</div>
              <div class="kpi__detail">{{ appStore.updateAvailable ? '有新版本' : '已是最新' }}</div>
            </div>
          </div>
        </div>
      </div>


      <!-- 快速操作卡片 -->
      <div class="card">
        <div class="card__header">
          <div>
            <div class="card__title">快速操作</div>
            <div class="card__subtitle">控制 DDBOT-WSa 进程</div>
          </div>
        </div>
        <div class="stack">
          <Button
            v-if="!appStore.isRunning"
            variant="success"
            :icon="Play"
            :loading="appStore.loading"
            size="lg"
            @click="appStore.start"
          >
            启动服务
          </Button>
          <Button
            v-else
            variant="danger"
            :icon="Square"
            :loading="appStore.loading"
            size="lg"
            @click="appStore.stop"
          >
            停止服务
          </Button>
          <Button variant="secondary" :icon="RefreshCw" :loading="appStore.loading" @click="appStore.restart">
            重启服务
          </Button>
          <Button variant="secondary" :icon="FileText" @click="$router.push('/logs')">查看日志</Button>
          <Button variant="secondary" :icon="FolderOpen" @click="appStore.openDataDir">打开数据目录</Button>
        </div>
      </div>

      <!-- 订阅站点分布 -->
      <div class="card">
        <div class="card__header">
          <div>
            <div class="card__title">订阅站点分布</div>
            <div class="card__subtitle">按站点分类的订阅数量</div>
          </div>
        </div>
        <div class="stack">
          <div v-if="Object.keys(appStore.subsSummary.bySite).length === 0" class="empty-state">
            <Database :size="32" />
            <div>暂无订阅数据</div>
          </div>
          <div v-else v-for="(count, site) in appStore.subsSummary.bySite" :key="site" class="row">
            <div class="label">{{ site }}</div>
            <div class="value">{{ count }}</div>
          </div>
        </div>
      </div>


    </div>

    <!-- 错误提示 -->
    <div v-if="appStore.error" class="home__error">
      <AlertCircle :size="16" />
      <span>{{ appStore.error }}</span>
      <Button size="sm" variant="danger" @click="appStore.clearError">关闭</Button>
    </div>

    <!-- 更新提示 -->
    <div v-if="appStore.updateAvailable" class="home__update">
      <AlertCircle :size="16" />
      <div>
        <div class="update__title">发现新版本</div>
        <div class="update__subtitle">{{ appStore.latestVersion }}</div>
      </div>
      <Button size="sm" variant="primary" @click="handleUpdate">更新</Button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { Activity, AlertCircle, AlertTriangle, Check, Clock, Database, Download, FileText, FolderOpen, Link, Play, RefreshCw, Settings, Square, Tag } from 'lucide-vue-next'
import Button from '../components/Button.vue'
import { useAppStore } from '../stores/app'
import { TauriAPI } from '../api/tauri'

const router = useRouter()
const appStore = useAppStore()

let statusInterval: number | null = null

onMounted(async () => {
  await appStore.init()
  
  if (!appStore.isUserApproved) {
    router.push('/setup')
    return
  }

  statusInterval = window.setInterval(() => {
    appStore.loadStatus()
  }, 5000) as unknown as number

  // 检查更新
  await handleCheckUpdate()
})

watch(() => appStore.isUserApproved, (approved) => {
  if (!approved) {
    router.push('/setup')
  }
})

onUnmounted(() => {
  if (statusInterval) {
    clearInterval(statusInterval)
  }
})

async function handleInstall() {
  try {
    await appStore.install()
  } catch (e) {
    console.error('Install failed:', e)
  }
}

async function handleCheckUpdate() {
  try {
    appStore.loading = true
    // 模拟检查更新
    await new Promise(resolve => setTimeout(resolve, 1000))
    appStore.updateAvailable = false
    appStore.latestVersion = '1.0.0'
    appStore.lastUpdateCheck = new Date().toLocaleString()
  } catch (e) {
    console.error('Check update failed:', e)
  } finally {
    appStore.loading = false
  }
}

async function handleUpdate() {
  try {
    appStore.loading = true
    // 模拟下载和安装更新
    await new Promise(resolve => setTimeout(resolve, 3000))
    console.log('Update completed')
  } catch (e) {
    console.error('Update failed:', e)
  } finally {
    appStore.loading = false
  }
}
</script>

<style scoped>
.home {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.home__content {
  display: grid;
  gap: 16px;
  grid-template-columns: 2fr 1fr;
}

@media (max-width: 980px) {
  .home__content {
    grid-template-columns: 1fr;
  }
}

.card {
  border-radius: 16px;
  border: 1px solid var(--border-color);
  background: var(--bg-card);
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.35);
  overflow: hidden;
}

.card--wide {
  grid-column: 1 / -1;
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

.kpis {
  display: grid;
  gap: 12px;
  padding: 16px;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
}

@media (max-width: 980px) {
  .kpis {
    grid-template-columns: repeat(2, 1fr);
  }
}

.kpi {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  border-radius: 14px;
  padding: 14px;
  border: 1px solid var(--border-color);
  background: var(--bg-card);
}

.kpi__icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 10px;
  background: rgba(124, 92, 255, 0.15);
  color: #7c5cff;
  flex-shrink: 0;
}

.kpi__label {
  font-size: 12px;
  opacity: 0.7;
}

.kpi__value {
  margin-top: 4px;
  font-size: 16px;
  font-weight: 700;
}

.kpi__value--active {
  color: #10b981;
}

.kpi__detail {
  margin-top: 2px;
  font-size: 11px;
  opacity: 0.6;
}

.actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.stack {
  padding: 16px;
  display: grid;
  gap: 8px;
}

.row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 0;
  border-bottom: 1px solid var(--bg-card);
}

.row:last-child {
  border-bottom: none;
}

.label {
  opacity: 0.7;
  font-size: 12px;
}

.value {
  font-weight: 600;
  font-size: 13px;
}

.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
  font-size: 11px;
  opacity: 0.9;
}

.truncate {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 200px;
}

.warning {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  border-radius: 10px;
  background: rgba(245, 158, 11, 0.15);
  border: 1px solid rgba(245, 158, 11, 0.3);
  color: #fbbf24;
  font-size: 12px;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 32px;
  color: var(--text-secondary);
}

.home__error {
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

.home__update {
  position: fixed;
  bottom: 24px;
  right: 24px;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  border-radius: 10px;
  background: rgba(59, 130, 246, 0.9);
  color: var(--text-primary);
  font-size: 13px;
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.5);
  backdrop-filter: blur(10px);
  max-width: 400px;
  z-index: 1000;
}

.update__title {
  font-weight: 600;
}

.update__subtitle {
  font-size: 11px;
  opacity: 0.8;
}
</style>
