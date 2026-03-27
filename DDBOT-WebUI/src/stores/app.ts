import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { TauriAPI } from '../api/tauri'

const API_BASE = 'http://localhost:3000/api'

// HTTP API 调用函数
async function callApi<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
  const response = await fetch(`${API_BASE}${endpoint}`, {
    headers: {
      'Content-Type': 'application/json',
    },
    ...options,
  })

  if (!response.ok) {
    if (response.status === 401) {
      window.location.hash = '#/login'
      throw new Error('未授权访问，需要登录')
    }
    const errorText = await response.text().catch(() => response.statusText)
    throw new Error(errorText || `API 调用失败: ${response.status}`)
  }

  try {
    return await response.json()
  } catch (e) {
    const errorText = await response.text().catch(() => '无效的响应格式')
    throw new Error(`API 响应解析失败: ${errorText}`)
  }
}

export interface DDBOTStatus {
  running: boolean
  pid?: string
  startTime?: number
}

export interface OneBotStatus {
  connected: boolean
  protocol?: string
  selfId?: string
  online: boolean
  good: boolean
}

export interface SubsSummary {
  total: number
  active: number
  paused: number
  bySite: Record<string, number>
}

export const useAppStore = defineStore('app', () => {
  const status = ref<DDBOTStatus>({ running: false })
  const onebotStatus = ref<OneBotStatus>({ connected: false, online: false, good: false })
  const subsSummary = ref<SubsSummary>({ total: 0, active: 0, paused: 0, bySite: {} })
  const version = ref<string>('-')
  const platform = ref<string>('WebUI')
  const loading = ref(false)
  const error = ref<string | undefined>()
  const isUserApproved = ref(true) // WebUI 架构下默认授权
  const initialized = ref(false)
  const firewallStatus = ref(false)
  const updateAvailable = ref(false)
  const latestVersion = ref<string>('-')
  const lastUpdateCheck = ref<string | undefined>()
  const dataDir = ref<string | undefined>()

  const isRunning = computed(() => status.value.running)
  const uptime = computed(() => {
    if (!status.value.startTime) return '-'
    const seconds = Math.floor((Date.now() - status.value.startTime) / 1000)
    const hours = Math.floor(seconds / 3600)
    const minutes = Math.floor((seconds % 3600) / 60)
    if (hours > 0) return `${hours}h ${minutes}m`
    return `${minutes}m`
  })

  async function init() {
    if (initialized.value) return
    try {
      await loadStatus()
      await loadFirewallStatus()
      await loadDataDir()
      initialized.value = true
    } catch (e) {
      console.error('Failed to init app store:', e)
    }
  }

  async function loadStatus() {
    try {
      loading.value = true

      // Fetch process status from the backend API
      let processRunning = false;
      let processPid: string | undefined = undefined;
      let processStartTime: number | undefined = undefined;

      try {
        const res = await fetch('/api/process/status');
        if (res.ok) {
          const data = await res.json();
          processRunning = data.running;
          processPid = data.pid;
          processStartTime = data.startTime;
        } else {
          console.warn('Failed to fetch process status:', res.status, await res.text());
        }
      } catch (e) {
        console.error('Error fetching process status:', e);
      }

      status.value = {
        running: processRunning,
        pid: processPid,
        startTime: processRunning ? (processStartTime || status.value.startTime || Date.now()) : undefined
      };

      // Original TauriAPI calls (if still relevant for other data)
      // Note: The original snippet had `const processStat = await TauriAPI.process.statusText()`
      // and then `status.value = { running: processStat === 'running', ... }`.
      // The new instruction replaces this with a direct API call for process status.
      // Assuming `TauriAPI.process.statusText()` is no longer the primary source for `running` status.
      // The `obStat` and `subsStat` calls are kept as they are for other data.

      const obStat = await TauriAPI.process.callOnebotStatusApi()
      const subsStat = await TauriAPI.process.callSubsSummaryApi()
      const healthVer = await TauriAPI.updater.currentVersion()

      if (healthVer) {
        version.value = healthVer
      }

      if (obStat) {
        onebotStatus.value = {
          connected: obStat.connected ?? obStat.online,
          online: obStat.online,
          good: obStat.good,
          protocol: obStat.protocol || 'OneBot v11',
          selfId: obStat.self_id || obStat.selfId
        }
      }

      if (subsStat) {
        subsSummary.value = {
          total: subsStat.total || 0,
          active: subsStat.active || 0,
          paused: subsStat.paused || 0,
          bySite: subsStat.by_site || subsStat.bySite || {}
        }
      }
    } catch (e) {
      console.error('Failed to load status:', e)
    } finally {
      loading.value = false
    }
  }

  async function loadFirewallStatus() {
    // 防火墙功能已移除，直接设置为 false
    firewallStatus.value = false
  }

  async function loadDataDir() {
    try {
      // 模拟数据目录，不再调用 TauriAPI
      dataDir.value = 'C:\\Users\\User\\AppData\\Roaming\\DDBOT-WebUI\\data\\ddbot'
    } catch (e) {
      console.error('Failed to load data dir:', e)
    }
  }

  async function openDataDir() {
    try {
      // 模拟打开数据目录，不再调用 TauriAPI
      console.log('Opening data directory:', dataDir.value)
    } catch (e) {
      console.error('Failed to open data dir:', e)
    }
  }

  async function approve() {
    // WebUI 架构下不需要前端授权逻辑，由后端处理
    isUserApproved.value = true
  }
  // 后端 API 地址
  const BACKEND_URL = '/api'

  async function install() {
    try {
      loading.value = true
      error.value = undefined
      const res = await fetch(`${BACKEND_URL}/install`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        }
      })
      if (!res.ok) {
        const data = await res.json()
        throw new Error(data.error || '安装请求失败')
      }
      await new Promise(resolve => setTimeout(resolve, 1000))
      await loadStatus()
    } catch (e: any) {
      error.value = `安装失败: ${e.message || e}`
      throw e
    } finally {
      loading.value = false
    }
  }

  async function start() {
    try {
      loading.value = true
      error.value = undefined
      const res = await fetch(`${BACKEND_URL}/process/control`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ action: 'start' })
      })
      if (!res.ok) {
        const data = await res.json()
        throw new Error(data.error || '启动请求失败')
      }
      await new Promise(resolve => setTimeout(resolve, 1000))
      await loadStatus()
    } catch (e) {
      error.value = `启动失败: ${e}`
      throw e
    } finally {
      loading.value = false
    }
  }

  async function stop() {
    try {
      loading.value = true
      error.value = undefined
      const res = await fetch(`${BACKEND_URL}/process/control`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ action: 'stop' })
      })
      if (!res.ok) {
        const data = await res.json()
        throw new Error(data.error || '停止请求失败')
      }
      await new Promise(resolve => setTimeout(resolve, 1000))
      await loadStatus()
    } catch (e: any) {
      error.value = `停止失败: ${e.message || e}`
      throw e
    } finally {
      loading.value = false
    }
  }

  async function restart() {
    try {
      loading.value = true
      error.value = undefined
      const res = await fetch(`${BACKEND_URL}/process/control`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ action: 'restart' })
      })
      if (!res.ok) {
        const data = await res.json()
        throw new Error(data.error || '重启请求失败')
      }
      await new Promise(resolve => setTimeout(resolve, 2000))
      await loadStatus()
    } catch (e: any) {
      error.value = `重启失败: ${e.message || e}`
      throw e
    } finally {
      loading.value = false
    }
  }


  function clearError() {
    error.value = undefined
  }

  return {
    status,
    onebotStatus,
    subsSummary,
    version,
    platform,
    loading,
    error,
    isUserApproved,
    initialized,
    firewallStatus,
    updateAvailable,
    latestVersion,
    lastUpdateCheck,
    dataDir,
    isRunning,
    uptime,
    init,
    loadStatus,
    loadFirewallStatus,
    loadDataDir,
    openDataDir,
    approve,
    install,
    start,
    stop,
    restart,
    clearError,
  }
})
