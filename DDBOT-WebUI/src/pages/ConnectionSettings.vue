<template>
  <div class="conn-page">

    <!-- 页面标题 -->
    <div class="page-header">
      <div class="page-header__icon"><Network :size="22" /></div>
      <div>
        <div class="page-header__title">连接管理</div>
        <div class="page-header__sub">管理 OneBot 连接方式及第三方推送集成</div>
      </div>
      <div class="header-spacer" />
      <!-- 当前连接状态徽章 -->
      <div class="status-badge" :class="connected ? 'status-badge--online' : 'status-badge--offline'">
        <span class="status-dot" />
        {{ connected ? `已连接  QQ: ${selfId || '获取中...'}` : '未连接' }}
      </div>
    </div>

    <!-- OneBot 连接卡片 -->
    <div class="card">
      <div class="card__header">
        <div class="card__icon purple"><Plug :size="16" /></div>
        <div>
          <div class="card__title">OneBot v11 连接</div>
          <div class="card__subtitle">支持 NapCat · LLOneBot · go-cqhttp</div>
        </div>
        <div class="header-spacer" />
        <div class="save-row">
          <Button variant="secondary" size="sm" :loading="reconnecting" @click="doReconnect">
            <RefreshCw :size="13" style="margin-right:4px" />重连
          </Button>
          <Button variant="primary" size="sm" :loading="saving" @click="save">
            <Save :size="13" style="margin-right:4px" />保存
          </Button>
        </div>
      </div>

      <div class="card__body">

        <!-- 连接模式选择 -->
        <div class="section">
          <div class="section__title">连接模式</div>
          <div class="mode-grid">
            <!-- 反向 WS -->
            <div
              class="mode-card"
              :class="{ 'mode-card--active': form.onebot.mode === 'reverse' }"
              @click="form.onebot.mode = 'reverse'"
            >
              <div class="mode-card__check"><CheckCircle2 v-if="form.onebot.mode === 'reverse'" :size="16" /><Circle v-else :size="16" /></div>
              <div class="mode-card__icon"><ServerIcon :size="28" /></div>
              <div class="mode-card__name">反向 WebSocket</div>
              <div class="mode-card__desc">Bot 做服务端，等待 OneBot 主动连入<br><span class="tag tag--green">推荐</span></div>
            </div>
            <!-- 正向 WS -->
            <div
              class="mode-card"
              :class="{ 'mode-card--active': form.onebot.mode === 'forward' }"
              @click="form.onebot.mode = 'forward'"
            >
              <div class="mode-card__check"><CheckCircle2 v-if="form.onebot.mode === 'forward'" :size="16" /><Circle v-else :size="16" /></div>
              <div class="mode-card__icon"><ArrowRightFromLine :size="28" /></div>
              <div class="mode-card__name">正向 WebSocket</div>
              <div class="mode-card__desc">Bot 主动连接 OneBot 的 ws-server<br><span class="tag tag--gray">断线后自动重连</span></div>
            </div>
          </div>
        </div>

        <!-- 反向 WS 配置 -->
        <Transition name="slide">
          <div v-if="form.onebot.mode === 'reverse'" class="section">
            <div class="section__title">反向 WS 配置</div>
            <div class="field-group">
              <div class="field">
                <label class="field__label">监听地址</label>
                <div class="field__input-wrap">
                  <input
                    v-model="form.onebot.ws_listen"
                    class="field__input"
                    placeholder="0.0.0.0:8080"
                    spellcheck="false"
                  />
                </div>
                <div class="field__hint">在 NapCat/LLOneBot 中填 <code>ws://&lt;本机IP&gt;:{{ listenPort }}</code></div>
              </div>
              <div class="field">
                <label class="field__label">鉴权 Token <span class="optional">（可选）</span></label>
                <div class="field__input-wrap">
                  <input
                    v-model="form.onebot.access_token"
                    class="field__input"
                    :type="showToken ? 'text' : 'password'"
                    placeholder="留空则不鉴权"
                  />
                  <button class="field__eye" @click="showToken = !showToken">
                    <Eye v-if="!showToken" :size="14" /><EyeOff v-else :size="14" />
                  </button>
                </div>
              </div>
            </div>
            <!-- 引导提示 -->
            <div class="tip-box tip-box--blue">
              <Info :size="14" />
              <div>
                <b>NapCat 配置引导</b><br>
                进入 NapCat WebUI → 网络配置 → 添加连接 → 选择 <b>反向 WebSocket</b><br>
                URL 填写：<code>ws://127.0.0.1:{{ listenPort }}</code>（本机部署时）
              </div>
            </div>
          </div>
        </Transition>

        <!-- 正向 WS 配置 -->
        <Transition name="slide">
          <div v-if="form.onebot.mode === 'forward'" class="section">
            <div class="section__title">正向 WS 配置</div>
            <div class="field-group">
              <div class="field">
                <label class="field__label">WebSocket 地址</label>
                <div class="field__input-wrap">
                  <input
                    v-model="form.onebot.ws_url"
                    class="field__input"
                    placeholder="ws://127.0.0.1:3001"
                    spellcheck="false"
                  />
                </div>
                <div class="field__hint">在 NapCat/LLOneBot 中开启正向 WebSocket，端口默认 3001</div>
              </div>
              <div class="field">
                <label class="field__label">鉴权 Token <span class="optional">（可选）</span></label>
                <div class="field__input-wrap">
                  <input
                    v-model="form.onebot.access_token"
                    class="field__input"
                    :type="showToken ? 'text' : 'password'"
                    placeholder="留空则不鉴权"
                  />
                  <button class="field__eye" @click="showToken = !showToken">
                    <Eye v-if="!showToken" :size="14" /><EyeOff v-else :size="14" />
                  </button>
                </div>
              </div>
            </div>
          </div>
        </Transition>

      </div>
    </div>

    <!-- Telegram Bot 卡片 -->
    <div class="card">
      <div class="card__header">
        <div class="card__icon blue"><Send :size="16" /></div>
        <div>
          <div class="card__title">Telegram Bot 推送</div>
          <div class="card__subtitle">将订阅推送同步转发到 Telegram 频道/群组</div>
        </div>
        <div class="header-spacer" />
        <Toggle v-model="form.telegram.enabled" />
      </div>

      <Transition name="slide">
        <div v-if="form.telegram.enabled" class="card__body">
          <div class="section">
            <div class="field-group">
              <div class="field">
                <label class="field__label">Bot Token</label>
                <div class="field__input-wrap">
                  <input
                    v-model="form.telegram.bot_token"
                    class="field__input"
                    :type="showTgToken ? 'text' : 'password'"
                    placeholder="从 @BotFather 获取，格式：123456:ABC..."
                    spellcheck="false"
                  />
                  <button class="field__eye" @click="showTgToken = !showTgToken">
                    <Eye v-if="!showTgToken" :size="14" /><EyeOff v-else :size="14" />
                  </button>
                </div>
              </div>
              <div class="field">
                <label class="field__label">HTTP 代理 <span class="optional">（可选，国内必填）</span></label>
                <div class="field__input-wrap">
                  <input
                    v-model="form.telegram.proxy"
                    class="field__input"
                    placeholder="http://127.0.0.1:7890"
                    spellcheck="false"
                  />
                </div>
              </div>
            </div>
            <div class="tip-box tip-box--yellow">
              <Info :size="14" />
              <div>
                绑定频道：私聊 Bot 发送 <code>!tg bind &lt;chat_id&gt; &lt;平台&gt; &lt;UID&gt;</code><br>
                测试连接：<code>!tg test</code>&nbsp;&nbsp;·&nbsp;&nbsp;查看状态：<code>!tg status</code>
              </div>
            </div>
          </div>
        </div>
      </Transition>
    </div>

    <!-- 保存按钮区 -->
    <div class="bottom-bar">
      <div class="bottom-bar__hint" v-if="lastSaved">
        <CheckCircle2 :size="13" class="icon-green" /> 已保存 · {{ lastSaved }}
        <span v-if="needRestart" class="restart-hint">（重启生效）</span>
      </div>
      <div class="header-spacer" />
      <Button variant="secondary" size="sm" @click="load">恢复</Button>
      <Button variant="primary" :loading="saving" @click="save">
        <Save :size="13" style="margin-right:4px" />保存所有配置
      </Button>
    </div>

    <!-- Toast 提示 -->
    <Transition name="toast">
      <div v-if="toast.show" class="toast" :class="`toast--${toast.type}`">
        <component :is="toast.type === 'success' ? CheckCircle2 : AlertCircle" :size="15" />
        {{ toast.msg }}
      </div>
    </Transition>

  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import {
  Network, Plug, ServerIcon, ArrowRightFromLine, Send,
  RefreshCw, Save, Eye, EyeOff, Info, CheckCircle2, Circle, AlertCircle
} from 'lucide-vue-next'
import Button from '../components/Button.vue'
import Toggle from '../components/Toggle.vue'

// ── 状态 ──────────────────────────────────────────────────────────────────────

const connected = ref(false)
const selfId = ref<number | null>(null)
const saving = ref(false)
const reconnecting = ref(false)
const showToken = ref(false)
const showTgToken = ref(false)
const lastSaved = ref('')
const needRestart = ref(false)

const form = reactive({
  onebot: {
    mode: 'reverse' as 'reverse' | 'forward',
    ws_listen: '0.0.0.0:8080',
    ws_url: 'ws://127.0.0.1:3001',
    access_token: '',
  },
  telegram: {
    enabled: false,
    bot_token: '',
    proxy: '',
  },
})

const toast = reactive({ show: false, msg: '', type: 'success' as 'success' | 'error' })

// ── 计算 ──────────────────────────────────────────────────────────────────────

const listenPort = computed(() => {
  const parts = form.onebot.ws_listen.split(':')
  return parts[parts.length - 1] || '8080'
})

// ── 数据加载 ──────────────────────────────────────────────────────────────────

async function loadStatus() {
  try {
    const r = await fetch('/api/v1/onebot/status')
    if (r.ok) {
      const d = await r.json()
      connected.value = d.connected || d.online || false
      selfId.value = d.self_id || null
    }
  } catch {}
}

async function load() {
  try {
    const r = await fetch('/api/v1/settings')
    if (!r.ok) return
    const d = await r.json()
    form.onebot.mode = d.onebot?.mode || 'reverse'
    form.onebot.ws_listen = d.onebot?.ws_listen || '0.0.0.0:8080'
    form.onebot.ws_url = d.onebot?.ws_url || 'ws://127.0.0.1:3001'
    form.onebot.access_token = d.onebot?.access_token || ''
    form.telegram.enabled = d.telegram?.enabled || false
    form.telegram.bot_token = d.telegram?.bot_token || ''
    form.telegram.proxy = d.telegram?.proxy || ''
  } catch (e) {
    showToast('加载配置失败: ' + e, 'error')
  }
}

// ── 保存 ──────────────────────────────────────────────────────────────────────

async function save() {
  saving.value = true
  try {
    const r = await fetch('/api/v1/settings', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(form),
    })
    const d = await r.json()
    if (!r.ok) throw new Error(d.error || r.statusText)
    lastSaved.value = new Date().toLocaleTimeString()
    needRestart.value = true
    showToast('配置已保存，重启 Bot 后生效', 'success')
  } catch (e) {
    showToast('保存失败: ' + e, 'error')
  } finally {
    saving.value = false
  }
}

// ── 重连 ──────────────────────────────────────────────────────────────────────

async function doReconnect() {
  reconnecting.value = true
  try {
    const r = await fetch('/api/v1/settings/reconnect', { method: 'POST' })
    const d = await r.json()
    if (d.status === 'reconnecting') {
      showToast('正在重连...', 'success')
      setTimeout(loadStatus, 2000)
    } else {
      showToast(d.hint || '请重启 Bot 进程使配置生效', 'success')
    }
  } catch (e) {
    showToast('重连请求失败: ' + e, 'error')
  } finally {
    reconnecting.value = false
  }
}

// ── Toast ─────────────────────────────────────────────────────────────────────

function showToast(msg: string, type: 'success' | 'error' = 'success') {
  toast.msg = msg
  toast.type = type
  toast.show = true
  setTimeout(() => { toast.show = false }, 3500)
}

// ── 生命周期 ──────────────────────────────────────────────────────────────────

onMounted(() => {
  load()
  loadStatus()
  // 每 5 秒刷新连接状态
  setInterval(loadStatus, 5000)
})
</script>

<style scoped>
/* ── 布局 ────────────────────────────────────────────────────────────────── */
.conn-page {
  display: flex;
  flex-direction: column;
  gap: 18px;
  max-width: 780px;
}

/* ── 页面标题 ────────────────────────────────────────────────────────────── */
.page-header {
  display: flex;
  align-items: center;
  gap: 12px;
}
.page-header__icon {
  width: 40px; height: 40px;
  border-radius: 12px;
  background: rgba(124, 92, 255, 0.15);
  border: 1px solid rgba(124, 92, 255, 0.3);
  display: flex; align-items: center; justify-content: center;
  color: #a78bfa;
}
.page-header__title { font-size: 18px; font-weight: 700; }
.page-header__sub { font-size: 12px; opacity: .6; margin-top: 2px; }

.status-badge {
  display: flex; align-items: center; gap: 7px;
  padding: 6px 14px; border-radius: 20px;
  font-size: 12px; font-weight: 600;
}
.status-badge--online  { background: rgba(16,185,129,.15); border: 1px solid rgba(16,185,129,.35); color: #10b981; }
.status-badge--offline { background: rgba(239,68,68,.12);  border: 1px solid rgba(239,68,68,.3);  color: #ef4444; }
.status-dot {
  width: 8px; height: 8px; border-radius: 50%;
  background: currentColor;
  animation: pulse 2s infinite;
}
@keyframes pulse {
  0%,100% { opacity:1 } 50% { opacity:.4 }
}

/* ── 卡片 ────────────────────────────────────────────────────────────────── */
.card {
  border-radius: 16px;
  border: 1px solid var(--border-color);
  background: var(--bg-card);
  box-shadow: 0 6px 24px rgba(0,0,0,.25);
  overflow: hidden;
}
.card__header {
  display: flex; align-items: center; gap: 12px;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
}
.card__icon {
  width: 32px; height: 32px; border-radius: 9px;
  display: flex; align-items: center; justify-content: center;
}
.card__icon.purple { background: rgba(124,92,255,.18); color: #a78bfa; }
.card__icon.blue   { background: rgba(59,130,246,.18); color: #60a5fa; }
.card__title   { font-size: 14px; font-weight: 700; }
.card__subtitle { font-size: 11px; opacity: .6; margin-top: 2px; }
.card__body    { padding: 20px; display: flex; flex-direction: column; gap: 20px; }

/* ── Section ─────────────────────────────────────────────────────────────── */
.section { display: flex; flex-direction: column; gap: 14px; }
.section__title {
  font-size: 12px; font-weight: 600; letter-spacing: .6px;
  text-transform: uppercase; opacity: .55;
}

/* ── 模式选择卡 ──────────────────────────────────────────────────────────── */
.mode-grid {
  display: grid; grid-template-columns: 1fr 1fr; gap: 12px;
}
.mode-card {
  position: relative;
  padding: 18px 16px;
  border-radius: 12px;
  border: 1.5px solid var(--border-color);
  background: var(--bg-secondary);
  cursor: pointer;
  transition: border-color .2s, background .2s;
  display: flex; flex-direction: column; gap: 6px;
}
.mode-card:hover { border-color: rgba(124,92,255,.4); }
.mode-card--active {
  border-color: rgba(124,92,255,.7);
  background: rgba(124,92,255,.09);
}
.mode-card__check {
  position: absolute; top: 12px; right: 12px;
  color: #a78bfa;
}
.mode-card__icon { opacity: .7; margin-bottom: 4px; }
.mode-card--active .mode-card__icon { opacity: 1; color: #a78bfa; }
.mode-card__name { font-size: 14px; font-weight: 600; }
.mode-card__desc { font-size: 12px; opacity: .6; line-height: 1.5; }

/* ── 字段 ────────────────────────────────────────────────────────────────── */
.field-group { display: flex; flex-direction: column; gap: 14px; }
.field { display: flex; flex-direction: column; gap: 6px; }
.field__label { font-size: 13px; font-weight: 600; }
.optional { font-weight: 400; opacity: .5; font-size: 11px; }
.field__input-wrap { position: relative; }
.field__input {
  width: 100%; box-sizing: border-box;
  padding: 9px 36px 9px 12px;
  border-radius: 9px;
  border: 1px solid rgba(255,255,255,.1);
  background: var(--bg-secondary);
  color: var(--text-primary);
  font-size: 13px; font-family: ui-monospace, monospace;
  outline: none; transition: border-color .2s;
}
.field__input:focus { border-color: rgba(124,92,255,.5); }
.field__eye {
  position: absolute; right: 10px; top: 50%; transform: translateY(-50%);
  background: none; border: none; cursor: pointer;
  color: var(--text-secondary); padding: 2px;
}
.field__hint { font-size: 11px; opacity: .5; }
.field__hint code, .tip-box code {
  background: rgba(255,255,255,.08); border-radius: 4px;
  padding: 1px 5px; font-family: ui-monospace, monospace;
}

/* ── 提示框 ──────────────────────────────────────────────────────────────── */
.tip-box {
  display: flex; gap: 10px; align-items: flex-start;
  padding: 12px 14px; border-radius: 10px;
  font-size: 12px; line-height: 1.6;
}
.tip-box--blue   { background: rgba(59,130,246,.1);  border: 1px solid rgba(59,130,246,.25); color: #93c5fd; }
.tip-box--yellow { background: rgba(234,179,8,.08); border: 1px solid rgba(234,179,8,.2);  color: #fde68a; }

/* ── 标签 ────────────────────────────────────────────────────────────────── */
.tag { font-size: 10px; font-weight: 600; padding: 2px 7px; border-radius: 5px; }
.tag--green { background: rgba(16,185,129,.18); color: #10b981; }
.tag--gray  { background: rgba(255,255,255,.08); color: rgba(255,255,255,.5); }

/* ── 底部操作栏 ──────────────────────────────────────────────────────────── */
.bottom-bar {
  display: flex; align-items: center; gap: 10px;
  padding: 14px 20px;
  border-radius: 12px;
  border: 1px solid var(--border-color);
  background: var(--bg-card);
}
.bottom-bar__hint { font-size: 12px; opacity: .6; display: flex; align-items: center; gap: 6px; }
.restart-hint { color: #f59e0b; font-size: 11px; }
.icon-green { color: #10b981; }

/* ── 保存行 ──────────────────────────────────────────────────────────────── */
.save-row { display: flex; gap: 8px; }
.header-spacer { flex: 1; }

/* ── Toast ───────────────────────────────────────────────────────────────── */
.toast {
  position: fixed; bottom: 28px; right: 28px;
  display: flex; align-items: center; gap: 8px;
  padding: 11px 16px; border-radius: 10px;
  font-size: 13px; font-weight: 500;
  backdrop-filter: blur(12px);
  box-shadow: 0 8px 24px rgba(0,0,0,.4);
  z-index: 9999;
}
.toast--success { background: rgba(16,185,129,.92); color: #fff; }
.toast--error   { background: rgba(239,68,68,.9);   color: #fff; }

/* ── 动画 ────────────────────────────────────────────────────────────────── */
.slide-enter-active, .slide-leave-active { transition: all .25s ease; }
.slide-enter-from, .slide-leave-to { opacity: 0; transform: translateY(-8px); }

.toast-enter-active, .toast-leave-active { transition: all .3s ease; }
.toast-enter-from, .toast-leave-to { opacity: 0; transform: translateY(12px); }
</style>
