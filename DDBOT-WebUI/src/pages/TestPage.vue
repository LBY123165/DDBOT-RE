<template>
  <div style="padding: 20px; color: var(--text-primary);">
    <h1>测试页面</h1>
    <p>如果你能看到这个页面，说明基本的 Vue + Tauri 环境是正常的。</p>
    <div style="margin-top: 20px;">
      <p>Store 状态: {{ initialized ? '已初始化' : '未初始化' }}</p>
      <p>平台: {{ platform }}</p>
      <p v-if="error" style="color: red;">错误: {{ error }}</p>
      <p v-if="downloadResult" style="color: green;">下载结果: {{ downloadResult }}</p>
      <p v-if="downloadProgress" style="color: yellow;">下载进度: {{ downloadProgress }}</p>
    </div>
    <button @click="testInvoke" style="margin-top: 20px; padding: 10px;">测试 Tauri 调用</button>
    
    <div style="margin-top: 30px; padding: 20px; border: 1px solid #444; border-radius: 8px;">
      <h2>下载测试</h2>
      <div style="margin-top: 10px;">
        <label for="downloadUrl">下载 URL:</label>
        <input 
          id="downloadUrl" 
          v-model="downloadUrl" 
          style="margin-left: 10px; padding: 5px; width: 400px; background: #333; color: var(--text-primary); border: 1px solid #555;"
        />
      </div>
      <div style="margin-top: 10px;">
        <label for="savePath">保存路径:</label>
        <input 
          id="savePath" 
          v-model="savePath" 
          style="margin-left: 10px; padding: 5px; width: 400px; background: #333; color: var(--text-primary); border: 1px solid #555;"
        />
      </div>
      <div style="margin-top: 10px;">
        <label for="filename">文件名:</label>
        <input 
          id="filename" 
          v-model="filename" 
          style="margin-left: 10px; padding: 5px; width: 400px; background: #333; color: var(--text-primary); border: 1px solid #555;"
        />
      </div>
      <button @click="testDownload" style="margin-top: 20px; padding: 10px;">开始下载</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { TauriAPI } from '@/api/tauri'

const initialized = ref(false)
const platform = ref('未知')
const error = ref<string | null>(null)
const downloadResult = ref<string | null>(null)
const downloadProgress = ref<string | null>(null)
const downloadUrl = ref('https://github.com/cnxysoft/DDBOT-WSa/releases/latest')
const savePath = ref('')
const filename = ref('')

async function testInvoke() {
  try {
    error.value = null
    const triple = await window.__TAURI__.core.invoke('get_platform_triple')
    platform.value = triple as string
    initialized.value = true
  } catch (e: any) {
    error.value = String(e)
    console.error('Tauri invoke error:', e)
  }
}

async function testDownload() {
  try {
    error.value = null
    downloadResult.value = null
    downloadProgress.value = '开始下载...'
    
    if (!savePath.value) {
      // 如果没有指定保存路径，使用默认数据目录
      const defaultDir = await TauriAPI.util.defaultDataDir()
      savePath.value = defaultDir
    }
    
    const result = await TauriAPI.download.downloadFile(
      downloadUrl.value,
      savePath.value,
      filename.value || undefined
    )
    
    downloadProgress.value = '下载完成'
    downloadResult.value = `文件已保存到: ${result}`
  } catch (e: any) {
    error.value = String(e)
    downloadProgress.value = null
    console.error('下载错误:', e)
  }
}

onMounted(async () => {
  try {
    const defaultDir = await TauriAPI.util.defaultDataDir()
    savePath.value = defaultDir
    
    const triple = await TauriAPI.ddbot.getPlatformTriple()
    platform.value = triple
    initialized.value = true
  } catch (e: any) {
    error.value = String(e)
    console.error('初始化错误:', e)
  }
})
</script>
