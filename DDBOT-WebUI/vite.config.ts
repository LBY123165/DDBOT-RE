import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// DDBOT-WebUI WebUI 构建配置
// Go 后端默认运行在 :3000（WebUIService），开发时前端 dev server 在 :5173

export default defineConfig({
  plugins: [vue()],
  clearScreen: false,
  server: {
    host: true,
    port: 5173,
    proxy: {
      // 将所有 /api/* 请求代理到 Go 后端
      '/api': {
        target: 'http://localhost:3000',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
})
