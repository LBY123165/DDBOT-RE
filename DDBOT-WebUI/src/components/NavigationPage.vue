<template>
  <nav class="nav">
    <div class="nav__header">
      <div class="nav__title">DDBOT-WebUI</div>
      <div class="nav__subtitle">WebUI</div>
    </div>

    <div class="nav__links">
      <RouterLink
        v-for="item in navItems"
        :key="item.path"
        :to="item.path"
        class="nav__link"
        :class="{ 'nav__link--active': $route.path === item.path }"
      >
        <component :is="item.icon" :size="18" />
        <span>{{ item.name }}</span>
        <div v-if="item.badge" class="nav__badge">{{ item.badge }}</div>
      </RouterLink>
    </div>

    <div class="nav__footer">
      <div class="nav__actions">
        <button @click="toggleTheme" class="nav__link footer-link">
          <component :is="isDark ? Sun : Moon" :size="18" />
          <span>{{ isDark ? '浅色模式' : '深色模式' }}</span>
        </button>
        <a href="https://github.com/LBY123165/DDbot-WebUI" target="_blank" class="nav__link footer-link">
          <Github :size="18" />
          <span>WebUI 源码</span>
        </a>
        <a href="https://github.com/cnxysoft/DDBOT-WebUI" target="_blank" class="nav__link footer-link">
          <Github :size="18" />
          <span>核心端 源码</span>
        </a>
      </div>
      <div class="nav__version">v{{ appStore.version }}</div>
    </div>
  </nav>
</template>

<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { RouterLink } from 'vue-router'
import { Activity, Home, Settings, Zap, FileText, ScrollText, LayoutTemplate, Database, Cpu, Network, Sun, Moon, Github } from 'lucide-vue-next'
import { useAppStore } from '../stores/app'

const appStore = useAppStore()

const isDark = ref(localStorage.getItem('theme') !== 'light')

const toggleTheme = () => {
  isDark.value = !isDark.value
  const theme = isDark.value ? 'dark' : 'light'
  localStorage.setItem('theme', theme)
  document.documentElement.setAttribute('data-theme', theme)
}

onMounted(() => {
  document.documentElement.setAttribute('data-theme', isDark.value ? 'dark' : 'light')
})

const navItems = computed<{name: string, path: string, icon: any, badge?: string}[]>(() => [
  {
    name: '概览',
    path: '/',
    icon: Home,
  },
  {
    name: '订阅',
    path: '/subscriptions',
    icon: Database,
  },
  {
    name: '模块',
    path: '/modules',
    icon: Cpu,
  },
  {
    name: '连接',
    path: '/connection',
    icon: Network,
  },
  {
    name: '配置',
    path: '/config',
    icon: FileText,
  },
  {
    name: '日志',
    path: '/logs',
    icon: ScrollText,
  },
  {
    name: '模板',
    path: '/templates',
    icon: LayoutTemplate,
  },
  {
    name: '设置',
    path: '/settings',
    icon: Settings,
  },
])
</script>

<style scoped>
.nav {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 16px 12px;
  border-right: 1px solid var(--border-color);
  background: var(--bg-secondary);
  backdrop-filter: blur(10px);
  min-width: 200px;
}

.nav__header {
  padding: 8px 12px 16px;
  border-bottom: 1px solid var(--border-color);
  margin-bottom: 8px;
}

.nav__title {
  font-weight: 700;
  font-size: 16px;
  letter-spacing: 0.2px;
}

.nav__subtitle {
  margin-top: 4px;
  font-size: 12px;
  opacity: 0.6;
}

.nav__links {
  display: flex;
  flex-direction: column;
  gap: 4px;
  flex: 1;
}

.nav__link {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border-radius: 10px;
  color: var(--text-secondary);
  text-decoration: none;
  border: 1px solid transparent;
  transition: all 0.2s ease;
  position: relative;
}

.nav__link:hover {
  background: var(--bg-card);
  color: var(--text-primary);
}

.nav__link--active {
  border-color: rgba(124, 92, 255, 0.35);
  background: rgba(124, 92, 255, 0.12);
  color: var(--text-primary);
}

.nav__badge {
  margin-left: auto;
  background: rgba(16, 185, 129, 0.2);
  color: #10b981;
  font-size: 10px;
  padding: 2px 6px;
  border-radius: 10px;
  font-weight: 600;
}

.nav__footer {
  padding-top: 16px;
}

.nav__actions {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-bottom: 12px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--border-color);
}

.footer-link {
  width: 100%;
  text-align: left;
  background: transparent;
  padding: 8px 12px;
  font-size: 13px;
}

.nav__version {
  font-size: 11px;
  opacity: 0.5;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}
</style>
