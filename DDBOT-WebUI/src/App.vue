<template>
  <div class="app">
    <Navigation />
    <main class="main">
      <RouterView />
    </main>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { RouterView } from 'vue-router'
import { useRouter } from 'vue-router'
import Navigation from './components/NavigationPage.vue'
import { useAppStore } from './stores/app'

const appStore = useAppStore()
const router = useRouter()

onMounted(async () => {
  await appStore.init()
  // WebUI 模式下不需要检查授权状态
})
</script>

<style scoped>
.app {
  display: flex;
  min-height: 100vh;
  width: 100%;
  color: var(--text-primary);
  background: var(--bg-primary);
  background-image: radial-gradient(1200px 700px at 10% 0%, rgba(124, 92, 255, 0.18), transparent 60%),
    radial-gradient(900px 600px at 90% 10%, rgba(0, 214, 160, 0.14), transparent 55%);
}

.main {
  flex: 1;
  padding: 22px;
  width: 100%;
  max-width: 100vw;
  box-sizing: border-box;
  overflow-x: hidden;
}

@media (max-width: 768px) {
  .app {
    flex-direction: column;
  }
  .main {
    padding: 12px;
  }
}
</style>
