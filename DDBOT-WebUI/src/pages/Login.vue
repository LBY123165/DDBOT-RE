<template>
  <div class="login-container">
    <div class="login-card">
      <div class="login-header">
        <ShieldCheck class="login-icon" :size="48" />
        <h2>访问受限 (安全验证)</h2>
        <p>您正在通过外部网络访问面板，请输入验证密码。</p>
      </div>

      <div class="login-body">
        <form @submit.prevent="handleLogin">
          <div class="form-group">
            <label>管理密码</label>
            <Input type="password" v-model="password" placeholder="请输入 WEBUI_PASSWORD..." required autofocus />
          </div>

          <div v-if="error" class="error-msg">{{ error }}</div>

          <Button type="submit" variant="primary" class="w-full mt-4" :disabled="loading">
            <span v-if="loading">验证中...</span>
            <span v-else>立即登入</span>
          </Button>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ShieldCheck } from 'lucide-vue-next'
import Input from '../components/Input.vue'
import Button from '../components/Button.vue'

const router = useRouter()
const route = useRoute()
const password = ref('')
const loading = ref(false)
const error = ref('')

const handleLogin = async () => {
  if (!password.value) return

  loading.value = true
  error.value = ''

  try {
    const res = await fetch('/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ password: password.value })
    })

    if (res.ok) {
      const redirect = route.query.redirect as string || '/'
      router.push(redirect)
    } else {
      const data = await res.json()
      error.value = data.error || '验证失败，密码错误'
    }
  } catch (err: any) {
    error.value = `网络请求失败: ${err.message}`
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  padding: 20px;
}

.login-card {
  width: 100%;
  max-width: 400px;
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: 16px;
  padding: 32px;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.4);
}

.login-header {
  text-align: center;
  margin-bottom: 32px;
}

.login-icon {
  color: var(--accent-color);
  margin-bottom: 16px;
  display: inline-block;
}

.login-header h2 {
  font-size: 24px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 8px;
}

.login-header p {
  color: var(--text-secondary);
  font-size: 14px;
}

.form-group {
  margin-bottom: 20px;
}

.form-group label {
  display: block;
  font-size: 13px;
  font-weight: 500;
  color: var(--text-secondary);
  margin-bottom: 8px;
}

.w-full {
  width: 100%;
}

.mt-4 {
  margin-top: 16px;
}

.error-msg {
  color: #ef4444;
  font-size: 13px;
  margin-top: 12px;
  text-align: center;
  padding: 8px;
  background: rgba(239, 68, 68, 0.1);
  border-radius: 6px;
}
</style>
