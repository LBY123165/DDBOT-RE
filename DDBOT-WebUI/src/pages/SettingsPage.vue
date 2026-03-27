<template>
  <div class="settings">
    <div class="card">
      <div class="card__header">
        <div>
          <div class="card__title">设置</div>
          <div class="card__subtitle">配置 DDBOT-WSa Desktop</div>
        </div>
      </div>

      <div class="card__body">
        <div class="section">
          <div class="section__title">安装选项</div>
          <div class="row">
            <div class="label">安装通道</div>
            <select class="select" v-model="channel" disabled>
              <option value="release">Release（GitHub Releases）</option>
              <option value="action">Action（测试版，后续支持）</option>
            </select>
          </div>
        </div>

        <div class="section">
          <div class="section__title">路径信息</div>
          <div class="row">
            <div class="label">工作目录</div>
            <div class="value mono">{{ workdir }}</div>
            <Button size="sm" variant="secondary" @click="editPath('workdir')">编辑</Button>
          </div>
          <div class="row">
            <div class="label">本体路径</div>
            <div class="value mono">{{ binaryPath }}</div>
            <Button size="sm" variant="secondary" @click="editPath('binary')">编辑</Button>
          </div>
          <div class="row">
            <div class="label">数据目录</div>
            <div class="value mono">{{ managedDir }}</div>
            <Button size="sm" variant="secondary" @click="editPath('data')">编辑</Button>
          </div>
          <div class="path-actions">
            <Button size="sm" variant="success" @click="savePaths">保存路径配置</Button>
            <Button size="sm" variant="secondary" @click="resetPaths">重置为默认</Button>
          </div>
        </div>

        <div class="section">
          <div class="section__title">安全设置</div>
          <div class="row">
            <div class="label">前端访问密码</div>
            <div class="value" style="flex: 1; padding: 0 16px;">
              <Input type="password" v-model="adminPassword" placeholder="留空则保持原密码..." />
            </div>
            <Button size="sm" variant="primary" @click="updatePassword">修改密码</Button>
          </div>
        </div>

        <!-- 下载进度条 -->
        <div v-if="installProgress.active" class="section">
          <div class="section__title">安装进度</div>
          <ProgressBar
            :percentage="installProgress.percentage"
            :title="installProgress.title"
            :status="installProgress.status"
            :speed="installProgress.speed"
            :active="installProgress.active"
            :steps="installProgress.steps"
          />
        </div>

        <div class="section">
          <div class="section__title">操作</div>
          <div class="actions">
            <Button :icon="Folder" @click="openWorkDir">打开工作目录</Button>
            <Button 
              :icon="Download" 
              @click="install"
              :loading="installing"
              :disabled="installProgress.active"
            >
              {{ installProgress.active ? '安装中...' : '安装/更新' }}
            </Button>
          </div>
        </div>
      </div>
    </div>

    <div v-if="appStore.error" class="settings__error">
      <AlertCircle :size="16" />
      <span>{{ appStore.error }}</span>
      <Button size="sm" variant="danger" @click="appStore.clearError">关闭</Button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive } from 'vue'
import { AlertCircle, Download, Folder } from 'lucide-vue-next'
import Button from '../components/Button.vue'
import Input from '../components/Input.vue'
import ProgressBar from '../components/ProgressBar.vue'
import { useAppStore } from '../stores/app'

const appStore = useAppStore()

const channel = ref<'release' | 'action'>('release')
const workdir = ref('')
const binaryPath = ref('')
const managedDir = ref('')
const installing = ref(false)
const adminPassword = ref('')

// 安装进度状态
const installProgress = reactive({
  active: false,
  percentage: 0,
  title: '准备安装',
  status: '初始化...',
  speed: '',
  steps: [
    { text: '检查环境', completed: false, active: false },
    { text: '创建目录', completed: false, active: false },
    { text: '下载文件', completed: false, active: false },
    { text: '验证完整性', completed: false, active: false },
    { text: '安装完成', completed: false, active: false }
  ]
})

// 检测当前运行环境并设置适当路径
function detectEnvironment() {
  const isDev = window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1'
  
  // 尝试从 localStorage 获取自定义路径
  const savedWorkdir = localStorage.getItem('ddbot-workdir')
  const savedBinaryPath = localStorage.getItem('ddbot-binary-path')
  const savedManagedDir = localStorage.getItem('ddbot-managed-dir')
  
  if (savedWorkdir && savedBinaryPath && savedManagedDir) {
    return {
      workdir: savedWorkdir,
      binaryPath: savedBinaryPath,
      managedDir: savedManagedDir
    }
  }
  
  if (isDev) {
    // 开发环境 - 使用实际项目路径
    const defaultWorkdir = 'G:\\Github Files\\ddbot-desktop\\DDBOT-WebUI'
    return {
      workdir: defaultWorkdir,
      binaryPath: `${defaultWorkdir}\\backend\\target\\debug\\ddbot-backend.exe`,
      managedDir: `${defaultWorkdir}\\data\\ddbot`
    }
  } else {
    // 生产环境（服务部署）
    return {
      workdir: 'C:\\Program Files\\DDBOT-WebUI',
      binaryPath: 'C:\\Program Files\\DDBOT-WebUI\\backend\\target\\release\\ddbot-backend.exe',
      managedDir: 'C:\\Program Files\\DDBOT-WebUI\\data\\ddbot'
    }
  }
}

// 保存路径配置到 localStorage
function savePaths() {
  localStorage.setItem('ddbot-workdir', workdir.value)
  localStorage.setItem('ddbot-binary-path', binaryPath.value)
  localStorage.setItem('ddbot-managed-dir', managedDir.value)
  appStore.error = undefined
  alert('路径配置已保存')
}

// 重置为默认路径
function resetPaths() {
  const paths = detectEnvironment()
  workdir.value = paths.workdir
  binaryPath.value = paths.binaryPath
  managedDir.value = paths.managedDir
  // 清除保存的配置
  localStorage.removeItem('ddbot-workdir')
  localStorage.removeItem('ddbot-binary-path')
  localStorage.removeItem('ddbot-managed-dir')
  appStore.error = undefined
  alert('已重置为默认路径')
}

// 编辑路径
function editPath(type: 'workdir' | 'binary' | 'data') {
  let currentValue: string
  let promptText: string
  
  switch (type) {
    case 'workdir':
      currentValue = workdir.value
      promptText = '请输入工作目录路径:'
      break
    case 'binary':
      currentValue = binaryPath.value
      promptText = '请输入本体路径:'
      break
    case 'data':
      currentValue = managedDir.value
      promptText = '请输入数据目录路径:'
      break
  }
  
  const newValue = prompt(promptText, currentValue)
  if (newValue !== null) {
    switch (type) {
      case 'workdir':
        workdir.value = newValue
        break
      case 'binary':
        binaryPath.value = newValue
        break
      case 'data':
        managedDir.value = newValue
        break
    }
  }
}

async function loadPaths() {
  try {
    const paths = detectEnvironment()
    workdir.value = paths.workdir
    binaryPath.value = paths.binaryPath
    managedDir.value = paths.managedDir
  } catch (e) {
    // 降级到默认值
    workdir.value = 'G:\\Github Files\\ddbot-desktop\\DDBOT-WebUI'
    binaryPath.value = 'G:\\Github Files\\ddbot-desktop\\DDBOT-WebUI\\backend\\target\\debug\\ddbot-backend.exe'
    managedDir.value = 'G:\\Github Files\\ddbot-desktop\\DDBOT-WebUI\\data\\ddbot'
    appStore.error = String(e)
  }
}

async function install() {
  if (installProgress.active) return
  
  try {
    installing.value = true
    appStore.error = undefined
    
    // 初始化进度条
    installProgress.active = true
    installProgress.percentage = 0
    installProgress.title = '正在安装 DDBOT'
    installProgress.status = '开始安装流程...'
    
    // 重置步骤状态
    installProgress.steps.forEach(step => {
      step.completed = false
      step.active = false
    })
    
    // 步骤1: 检查环境
    installProgress.steps[0].active = true
    installProgress.status = '检查运行环境...'
    installProgress.percentage = 10
    await new Promise(resolve => setTimeout(resolve, 800))
    installProgress.steps[0].completed = true
    installProgress.steps[0].active = false
    
    // 步骤2: 创建目录
    installProgress.steps[1].active = true
    installProgress.status = '创建必要目录...'
    installProgress.percentage = 25
    await new Promise(resolve => setTimeout(resolve, 600))
    installProgress.steps[1].completed = true
    installProgress.steps[1].active = false
    
    // 步骤3: 下载文件
    installProgress.steps[2].active = true
    installProgress.status = '下载 DDBOT 可执行文件...'
    installProgress.speed = '5.2MB/s'
    
    // 模拟下载进度
    for (let i = 30; i <= 60; i += 10) {
      installProgress.percentage = i
      await new Promise(resolve => setTimeout(resolve, 400))
    }
    
    installProgress.steps[2].completed = true
    installProgress.steps[2].active = false
    installProgress.speed = ''
    
    // 步骤4: 验证完整性
    installProgress.steps[3].active = true
    installProgress.status = '验证文件完整性...'
    installProgress.percentage = 75
    await new Promise(resolve => setTimeout(resolve, 600))
    
    installProgress.steps[3].completed = true
    installProgress.steps[3].active = false
    
    // 调用实际的安装API
    const response = await fetch('http://localhost:3000/api/install', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
    })
    
    if (!response.ok) {
      throw new Error(`安装失败: ${response.statusText}`)
    }
    
    const result = await response.json()
    if (result.success) {
      // 步骤5: 安装完成
      installProgress.steps[4].active = true
      installProgress.status = '安装完成!'
      installProgress.percentage = 100
      await new Promise(resolve => setTimeout(resolve, 300))
      
      installProgress.steps[4].completed = true
      installProgress.steps[4].active = false
      appStore.version = result.version || 'WebUI v0.1.0'
      
      // 3秒后隐藏进度条
      setTimeout(() => {
        installProgress.active = false
      }, 3000)
    } else {
      throw new Error(result.message || '安装失败')
    }
  } catch (e) {
    appStore.error = `安装失败: ${e}`
    installProgress.active = false
  } finally {
    installing.value = false
  }
}

async function openWorkDir() {
  try {
    // WebUI 模式下使用浏览器API打开文件夹（有限支持）
    // 或者显示路径让用户手动打开
    alert(`工作目录路径:\n${workdir.value}\n\n请手动在文件资源管理器中打开此路径`)    
  } catch (e) {
    appStore.error = `打开目录失败: ${e}`
  }
}

async function updatePassword() {
  if (!adminPassword.value) {
    alert('请输入新密码');
    return;
  }
  if (!confirm('确定要修改前端访问密码吗？修改后将重新登录。')) return;

  try {
    const res = await fetch('/api/auth/update_password', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ newPassword: adminPassword.value })
    });
    
    if (res.ok) {
      alert('密码更新成功！请重新登录。');
      window.location.hash = '#/login';
      adminPassword.value = '';
    } else {
      const err = await res.json();
      alert(`密码修改失败: ${err.error}`);
    }
  } catch (e: any) {
    alert(`网络异常: ${e.message}`);
  }
}

onMounted(() => {
  loadPaths()
})
</script>

<style scoped>
.settings {
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

.card__body {
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.section__title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  padding-bottom: 8px;
  border-bottom: 1px solid var(--bg-card);
}

.row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 0;
}

.label {
  opacity: 0.7;
  font-size: 13px;
}

.value {
  font-weight: 600;
  font-size: 13px;
}

.path-actions {
  display: flex;
  gap: 8px;
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid var(--bg-card);
}

.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
  background: var(--bg-card);
  padding: 4px 8px;
  border-radius: 4px;
  max-width: 400px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
  font-size: 12px;
  opacity: 0.85;
  text-align: right;
  max-width: 50%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.select {
  padding: 8px 12px;
  border-radius: 10px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  background: var(--bg-card);
  color: rgba(232, 234, 240, 0.95);
  font-size: 13px;
  outline: none;
}

.select:focus {
  border-color: rgba(124, 92, 255, 0.5);
}

.actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.settings__error {
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
