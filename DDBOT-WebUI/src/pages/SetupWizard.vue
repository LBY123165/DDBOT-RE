<template>
  <div class="wizard">
    <div class="wizard__container">
      <div class="wizard__header">
        <div class="wizard__title">欢迎使用 DDBOT-WSa Desktop</div>
        <div class="wizard__subtitle">首次运行配置向导</div>
      </div>

      <div v-if="step === 1" class="wizard__step">
        <div class="wizard__question">您是否已经部署过 DDBOT-WSa？</div>

        <div class="wizard__options">
          <button
            class="wizard__option"
            :class="{ 'wizard__option--selected': hasExistingDeployment }"
            @click="hasExistingDeployment = true"
          >
            <div class="wizard__option-icon">📁</div>
            <div class="wizard__option-content">
              <div class="wizard__option-title">已部署过</div>
              <div class="wizard__option-desc">导入现有部署目录（配置、数据库等）</div>
            </div>
          </button>

          <button
            class="wizard__option"
            :class="{ 'wizard__option--selected': !hasExistingDeployment }"
            @click="hasExistingDeployment = false"
          >
            <div class="wizard__option-icon">🆕</div>
            <div class="wizard__option-content">
              <div class="wizard__option-title">全新安装</div>
              <div class="wizard__option-desc">仅下载最新版本并创建最小配置</div>
            </div>
          </button>
        </div>

        <div v-if="hasExistingDeployment" class="wizard__field">
          <label class="wizard__label">选择部署目录</label>
          <div class="wizard__input-group">
            <input
              v-model="selectedDir"
              class="wizard__input"
              placeholder="点击按钮选择目录"
              readonly
            />
            <Button @click="selectDirectory">选择</Button>
          </div>
          <div v-if="selectedDir" class="wizard__hint">
            已选择: {{ selectedDir }}
          </div>
        </div>

        <div class="wizard__actions">
          <Button variant="secondary" @click="skipSetup">跳过设置</Button>
          <Button
            variant="primary"
            :disabled="!canProceed"
            :loading="appStore.loading"
            @click="handleNext"
          >
            继续
          </Button>
        </div>
      </div>

      <div v-else-if="step === 2" class="wizard__step">
        <div class="wizard__info">
          <div class="wizard__info-icon">🔐</div>
          <div class="wizard__info-content">
            <div class="wizard__info-title">授权说明</div>
            <div class="wizard__info-text">
              Desktop 需要在配置文件中启用 Admin API 才能获取运行状态。
              这需要修改 <code>application.yaml</code> 文件并生成访问令牌。
            </div>
            <div class="wizard__info-features">
              <div class="wizard__feature">✓ 本地只读访问</div>
              <div class="wizard__feature">✓ 自动生成安全令牌</div>
              <div class="wizard__feature">✓ 仅监听本地 127.0.0.1</div>
            </div>
          </div>
        </div>

        <div class="wizard__actions">
          <Button variant="secondary" @click="step--">返回</Button>
          <Button
            variant="primary"
            :loading="appStore.loading"
            @click="handleApprove"
          >
            我了解并授权
          </Button>
        </div>
      </div>

      <div v-else-if="step === 3" class="wizard__step wizard__step--success">
        <div class="wizard__success-icon">✅</div>
        <div class="wizard__success-title">设置完成！</div>
        <div class="wizard__success-desc">
          DDBOT-WSa Desktop 已准备就绪，您现在可以：
        </div>
        <div class="wizard__success-actions">
          <Button variant="primary" @click="goToHome">进入主页</Button>
          <Button
            v-if="!appStore.isRunning"
            variant="success"
            @click="startAndGoHome"
          >
            启动并进入
          </Button>
        </div>
      </div>
    </div>

    <div v-if="appStore.error" class="wizard__error">
      <span>{{ appStore.error }}</span>
      <Button size="sm" variant="danger" @click="appStore.clearError">关闭</Button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import Button from '../components/Button.vue'
import { useAppStore } from '../stores/app'

const router = useRouter()
const appStore = useAppStore()

const step = ref(1)
const hasExistingDeployment = ref(false)
const selectedDir = ref('')

const canProceed = computed(() => {
  if (hasExistingDeployment.value) {
    return selectedDir.value.length > 0
  }
  return true
})

onMounted(async () => {
  // WebUI 架构下通常由设置页管理路径，此处逻辑保持简单
  if (appStore.isUserApproved) {
    // router.push('/')
  }
})

async function selectDirectory() {
  // WebUI 模式下无法直接调用系统对话框
  // 建议引导用户在设置页面输入路径
  alert('WebUI 模式请在主程序启动后，进入设置页面手动配置路径。')
}

async function handleNext() {
  try {
    if (hasExistingDeployment.value) {
      await appStore.importDeployment(selectedDir.value)
    } else {
      await appStore.install()
    }
    step.value = 2
  } catch (e) {
    console.error('Failed to proceed:', e)
  }
}

async function handleApprove() {
  try {
    await appStore.approve()
    step.value = 3
  } catch (e) {
    console.error('Failed to approve:', e)
  }
}

function skipSetup() {
  router.push('/')
}

function goToHome() {
  router.push('/')
}

async function startAndGoHome() {
  try {
    await appStore.start()
    router.push('/')
  } catch (e) {
    console.error('Failed to start:', e)
  }
}
</script>

<style scoped>
.wizard {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  padding: 24px;
}

.wizard__container {
  max-width: 600px;
  width: 100%;
}

.wizard__header {
  text-align: center;
  margin-bottom: 32px;
}

.wizard__title {
  font-size: 28px;
  font-weight: 700;
  margin-bottom: 8px;
}

.wizard__subtitle {
  font-size: 14px;
  opacity: 0.7;
}

.wizard__step {
  border-radius: 16px;
  border: 1px solid var(--border-color);
  background: var(--bg-card);
  padding: 24px;
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.35);
}

.wizard__question {
  font-size: 18px;
  font-weight: 600;
  margin-bottom: 20px;
}

.wizard__options {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-bottom: 24px;
}

.wizard__option {
  display: flex;
  gap: 16px;
  padding: 16px;
  border-radius: 12px;
  border: 2px solid var(--border-color);
  background: var(--bg-card);
  cursor: pointer;
  transition: all 0.2s ease;
  width: 100%;
  text-align: left;
}

.wizard__option:hover {
  border-color: rgba(124, 92, 255, 0.3);
  background: rgba(124, 92, 255, 0.05);
}

.wizard__option--selected {
  border-color: rgba(124, 92, 255, 0.5);
  background: rgba(124, 92, 255, 0.1);
}

.wizard__option-icon {
  font-size: 32px;
}

.wizard__option-content {
  flex: 1;
}

.wizard__option-title {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 4px;
}

.wizard__option-desc {
  font-size: 13px;
  opacity: 0.7;
}

.wizard__field {
  margin-bottom: 24px;
}

.wizard__label {
  display: block;
  font-size: 13px;
  font-weight: 600;
  margin-bottom: 8px;
  opacity: 0.9;
}

.wizard__input-group {
  display: flex;
  gap: 8px;
}

.wizard__input {
  flex: 1;
  padding: 10px 12px;
  border-radius: 10px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  background: var(--bg-card);
  color: var(--text-primary);
  font-size: 13px;
  outline: none;
}

.wizard__input:focus {
  border-color: rgba(124, 92, 255, 0.5);
}

.wizard__hint {
  margin-top: 8px;
  font-size: 12px;
  opacity: 0.7;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}

.wizard__actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.wizard__info {
  display: flex;
  gap: 16px;
  margin-bottom: 24px;
}

.wizard__info-icon {
  font-size: 48px;
}

.wizard__info-content {
  flex: 1;
}

.wizard__info-title {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 8px;
}

.wizard__info-text {
  font-size: 13px;
  opacity: 0.8;
  line-height: 1.6;
  margin-bottom: 16px;
}

.wizard__info-text code {
  padding: 2px 6px;
  border-radius: 4px;
  background: var(--bg-card);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
}

.wizard__info-features {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.wizard__feature {
  font-size: 13px;
  opacity: 0.8;
}

.wizard__step--success {
  text-align: center;
}

.wizard__success-icon {
  font-size: 64px;
  margin-bottom: 16px;
}

.wizard__success-title {
  font-size: 24px;
  font-weight: 700;
  margin-bottom: 8px;
}

.wizard__success-desc {
  font-size: 14px;
  opacity: 0.8;
  margin-bottom: 24px;
}

.wizard__success-actions {
  display: flex;
  justify-content: center;
  gap: 12px;
  flex-wrap: wrap;
}

.wizard__error {
  position: fixed;
  bottom: 24px;
  left: 50%;
  transform: translateX(-50%);
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
}
</style>
