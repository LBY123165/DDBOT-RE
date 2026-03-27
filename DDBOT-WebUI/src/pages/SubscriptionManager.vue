<template>
  <div class="subscription-manager">
    <div class="header">
      <div>
        <h2 class="title">订阅管理</h2>
        <p class="subtitle">查看和管理 Bot 当前的所有订阅</p>
      </div>
      <Button :icon="Plus" variant="primary" @click="showAddModal = true">添加订阅</Button>
      <Button :icon="RefreshCw" @click="fetchSubscriptions">刷新列表</Button>
    </div>

    <!-- Error message -->
    <div v-if="error" class="error-banner">
      {{ error }}
      <Button size="sm" variant="danger" @click="error = ''">关闭</Button>
    </div>

    <div class="card">
      <div v-if="loading" class="empty-state">
        <div>正在加载订阅信息...</div>
      </div>
      <div v-else-if="subscriptions.length === 0" class="empty-state">
        <div>暂无订阅数据或无法连接到 Bot。</div>
      </div>
      <div class="table-container" v-else>
        <table class="subs-table">
          <thead>
            <tr>
              <th>群组号</th>
              <th>站点</th>
              <th>账号 UID</th>
              <th>账号名称</th>
              <th>订阅类型</th>
              <th>状态</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="sub in subscriptions" :key="`${sub.group_code}-${sub.site}-${sub.uid}`">
              <td class="mono">{{ sub.group_code }}</td>
              <td>
                <span class="site-badge">{{ sub.site }}</span>
              </td>
              <td class="mono font-semibold">{{ sub.uid }}</td>
              <td>{{ sub.name || '-' }}</td>
              <td>
                <span class="type-badge">{{ formatConcernType(sub.concern_type) }}</span>
              </td>
              <td>
                <span :class="sub.enable ? 'status-on' : 'status-off'">
                  {{ sub.enable ? '启用' : '停用' }}
                </span>
              </td>
              <td>
                <Button size="sm" variant="danger" @click="removeSub(sub)">删除</Button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Add Subscription Modal -->
    <div v-if="showAddModal" class="modal-overlay">
      <div class="modal">
        <h3 class="modal-title">添加订阅</h3>
        <div class="form-group">
          <label>群组号 (group_code)</label>
          <input type="number" v-model.number="newSub.group_code" class="input" placeholder="例如: 123456789" />
        </div>
        <div class="form-group">
          <label>站点 (site)</label>
          <select v-model="newSub.site" class="input">
            <option value="bilibili">bilibili</option>
            <option value="acfun">acfun</option>
            <option value="douyin">douyin</option>
            <option value="weibo">weibo</option>
            <option value="youtube">youtube</option>
            <option value="twitter">twitter</option>
          </select>
        </div>
        <div class="form-group">
          <label>账号 UID (uid)</label>
          <input type="text" v-model="newSub.uid" class="input" placeholder="例如: 226257459" />
        </div>
        <div class="form-group">
          <label>账号名称（可选，用于备注）</label>
          <input type="text" v-model="newSub.name" class="input" placeholder="例如: 某up主（留空将使用uid）" />
        </div>
        <div class="form-group">
          <label>订阅类型 (concern_type)</label>
          <select v-model="newSub.concern_type" class="input">
            <option value="live">直播 (live)</option>
            <option value="news">动态/文章/视频 (news)</option>
            <option value="live,news">直播+动态 (live,news)</option>
          </select>
        </div>

        <div class="modal-actions">
          <Button variant="secondary" @click="showAddModal = false">取消</Button>
          <Button variant="primary" @click="addSub">确定添加</Button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Plus, RefreshCw } from 'lucide-vue-next'
import Button from '../components/Button.vue'

// 与后端 db.Concern 结构对齐
interface SubInfo {
  id: number
  site: string
  uid: string
  name: string
  group_code: number
  concern_type: string
  enable: number
}

// 添加订阅表单（前端独立）
interface NewSubForm {
  uid: string
  name: string
  site: string
  concern_type: string
  group_code: number
}

const subscriptions = ref<SubInfo[]>([])
const loading = ref(false)
const error = ref('')
const showAddModal = ref(false)

const newSub = ref<NewSubForm>({
  uid: '',
  name: '',
  site: 'bilibili',
  concern_type: 'live',
  group_code: 0
})

const API_BASE = '/api/v1'

// ─── 工具函数 ─────────────────────────────────────────────────────────────────

/** 格式化 concern_type 为可读标签 */
function formatConcernType(t: string): string {
  const map: Record<string, string> = {
    live: '直播',
    news: '动态/文章',
    'live,news': '直播+动态',
    'news,live': '动态+直播',
  }
  return map[t] || t
}

// ─── 订阅列表 ─────────────────────────────────────────────────────────────────

const fetchSubscriptions = async () => {
  try {
    loading.value = true
    error.value = ''
    // GET /api/v1/subs  → { subs: [...], total: n }
    const res = await fetch(`${API_BASE}/subs`)
    if (res.ok) {
      const data = await res.json()
      subscriptions.value = Array.isArray(data.subs) ? data.subs : []
    } else {
      throw new Error(`HTTP ${res.status}: ${res.statusText}`)
    }
  } catch (e: any) {
    error.value = `请求失败，请确认 DDBOT 后端正在运行。错误: ${e.message}`
    subscriptions.value = []
  } finally {
    loading.value = false
  }
}

// ─── 添加订阅 ─────────────────────────────────────────────────────────────────

const addSub = async () => {
  if (!newSub.value.uid || !newSub.value.group_code) {
    alert('请填写账号 ID 和群组号')
    return
  }
  try {
    // POST /api/v1/subs  body: { site, uid, name, group_code, concern_type }
    const res = await fetch(`${API_BASE}/subs`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        site: newSub.value.site,
        uid: newSub.value.uid,
        name: newSub.value.name || newSub.value.uid,
        group_code: newSub.value.group_code,
        concern_type: newSub.value.concern_type,
      }),
    })

    if (res.ok) {
      showAddModal.value = false
      newSub.value.uid = ''
      newSub.value.name = ''
      await fetchSubscriptions()
    } else {
      const errData = await res.json().catch(() => ({}))
      alert(`添加失败: ${errData.error || res.statusText}`)
    }
  } catch (e: any) {
    alert(`网络错误: ${e.message}`)
  }
}

// ─── 删除订阅 ─────────────────────────────────────────────────────────────────

const removeSub = async (sub: SubInfo) => {
  if (!confirm(`确定要删除群 ${sub.group_code} 中对 ${sub.site}/${sub.uid} 的订阅吗？`)) return

  try {
    // DELETE /api/v1/subs/{site}/{uid}?group=<groupCode>
    const res = await fetch(
      `${API_BASE}/subs/${encodeURIComponent(sub.site)}/${encodeURIComponent(sub.uid)}?group=${sub.group_code}`,
      { method: 'DELETE' }
    )

    if (res.ok) {
      await fetchSubscriptions()
    } else {
      const errData = await res.json().catch(() => ({}))
      alert(`删除失败: ${errData.error || res.statusText}`)
    }
  } catch (e: any) {
    alert(`网络错误: ${e.message}`)
  }
}

onMounted(() => {
  fetchSubscriptions()
})
</script>

<style scoped>
.subscription-manager {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.header {
  display: flex;
  align-items: center;
  gap: 16px;
  background: var(--bg-card);
  padding: 20px;
  border-radius: 16px;
  border: 1px solid var(--border-color);
}

.header > div {
  flex: 1;
}

.title {
  margin: 0;
  font-size: 20px;
  font-weight: 700;
}

.subtitle {
  margin: 4px 0 0 0;
  font-size: 13px;
  opacity: 0.7;
}

.card {
  background: var(--bg-card);
  border-radius: 16px;
  border: 1px solid var(--border-color);
  padding: 0;
  overflow: hidden;
}

.empty-state {
  padding: 40px;
  text-align: center;
  color: var(--text-secondary);
}

.table-container {
  overflow-x: auto;
}

.subs-table {
  width: 100%;
  border-collapse: collapse;
}

.subs-table th,
.subs-table td {
  padding: 12px 16px;
  text-align: left;
  border-bottom: 1px solid var(--border-color);
}

.subs-table th {
  background: var(--bg-card);
  font-size: 13px;
  font-weight: 600;
  color: var(--text-secondary);
}

.subs-table td {
  font-size: 14px;
}

.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}
.font-semibold {
  font-weight: 600;
}

.site-badge {
  display: inline-block;
  padding: 3px 8px;
  background: rgba(0, 214, 160, 0.12);
  color: #34d399;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 500;
}

.type-badge {
  display: inline-block;
  padding: 4px 8px;
  background: rgba(124, 92, 255, 0.15);
  color: #c9bcff;
  border-radius: 6px;
  font-size: 12px;
}

.status-on {
  display: inline-block;
  padding: 3px 8px;
  background: rgba(16, 185, 129, 0.15);
  color: #10b981;
  border-radius: 6px;
  font-size: 12px;
}

.status-off {
  display: inline-block;
  padding: 3px 8px;
  background: rgba(239, 68, 68, 0.12);
  color: #f87171;
  border-radius: 6px;
  font-size: 12px;
}

.error-banner {
  background: rgba(239, 68, 68, 0.15);
  border: 1px solid rgba(239, 68, 68, 0.3);
  color: #fca5a5;
  padding: 12px 16px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.6);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal {
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  width: 90%;
  max-width: 480px;
  border-radius: 16px;
  padding: 24px;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.4);
}

.modal-title {
  margin: 0 0 20px 0;
  font-size: 18px;
  font-weight: 700;
}

.form-group {
  margin-bottom: 16px;
}

.form-group label {
  display: block;
  font-size: 13px;
  opacity: 0.8;
  margin-bottom: 6px;
}

.input {
  width: 100%;
  padding: 10px 12px;
  border-radius: 8px;
  border: 1px solid var(--border-color);
  background: var(--bg-card);
  color: var(--text-primary);
  font-family: inherit;
  font-size: 14px;
  box-sizing: border-box;
}

.input:focus {
  outline: none;
  border-color: var(--accent-color);
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 24px;
}
</style>
