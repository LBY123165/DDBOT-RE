<template>
  <div class="template-center">
    <div class="card">
      <div class="card__header">
        <div>
          <div class="card__title">模板中心</div>
          <div class="card__subtitle">管理和配置消息模板</div>
        </div>
        <div class="actions">
          <Button :icon="Plus" @click="showCreateModal = true">新建模板</Button>
          <Button :icon="RefreshCw" @click="loadTemplates">刷新</Button>
        </div>
      </div>

      <div class="template-grid">
        <div 
          v-for="template in templates" 
          :key="template.id"
          class="template-card"
          :class="{ 'template-card--active': template.enabled }"
        >
          <div class="template-header">
            <h3 class="template-name">{{ template.name }}</h3>
            <div class="template-badge" :class="template.type">
              {{ template.type }}
            </div>
          </div>
          
          <div class="template-description">
            {{ template.description }}
          </div>
          
          <div class="template-preview">
            <div class="preview-label">预览:</div>
            <div class="preview-content">{{ template.preview }}</div>
          </div>
          
          <div class="template-footer">
            <div class="template-meta">
              <span class="meta-item">
                <Clock :size="14" />
                {{ formatDate(template.lastModified) }}
              </span>
              <span class="meta-item">
                <User :size="14" />
                {{ template.author }}
              </span>
            </div>
            
            <div class="template-actions">
              <Button size="sm" variant="secondary" @click="editTemplate(template)">
                编辑
              </Button>
              <Button 
                size="sm" 
                :variant="template.enabled ? 'danger' : 'success'"
                @click="toggleTemplate(template)"
              >
                {{ template.enabled ? '禁用' : '启用' }}
              </Button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 创建/编辑模板模态框 -->
    <div v-if="showCreateModal || editingTemplate" class="modal-overlay" @click="closeModal">
      <div class="modal" @click.stop>
        <div class="modal-header">
          <h2>{{ editingTemplate ? '编辑模板' : '新建模板' }}</h2>
          <Button variant="secondary" @click="closeModal">×</Button>
        </div>
        
        <div class="modal-body">
          <div class="form-group">
            <label>模板名称</label>
            <input v-model="templateForm.name" type="text" placeholder="输入模板名称">
          </div>
          
          <div class="form-group">
            <label>模板类型</label>
            <select v-model="templateForm.type">
              <option value="text">文本</option>
              <option value="image">图片</option>
              <option value="mixed">图文混合</option>
            </select>
          </div>
          
          <div class="form-group">
            <label>描述</label>
            <textarea v-model="templateForm.description" placeholder="模板描述"></textarea>
          </div>
          
          <div class="form-group">
            <label>模板内容</label>
            <textarea 
              v-model="templateForm.content" 
              placeholder="输入模板内容，支持变量 {{name}}, {{time}} 等"
              rows="6"
            ></textarea>
          </div>
        </div>
        
        <div class="modal-footer">
          <Button variant="secondary" @click="closeModal">取消</Button>
          <Button variant="primary" @click="saveTemplate">
            {{ editingTemplate ? '更新' : '创建' }}
          </Button>
        </div>
      </div>
    </div>

    <div v-if="error" class="template-error">
      <AlertCircle :size="16" />
      <span>{{ error }}</span>
      <Button size="sm" variant="danger" @click="clearError">关闭</Button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { AlertCircle, Plus, RefreshCw, Clock, User } from 'lucide-vue-next'
import { TauriAPI } from '../api/tauri'
import Button from '../components/Button.vue'

interface Template {
  id: string
  name: string
  type: string
  description: string
  content: string
  preview: string
  author: string
  lastModified: Date
  enabled: boolean
}

const templates = ref<Template[]>([])
const showCreateModal = ref(false)
const editingTemplate = ref<Template | null>(null)
const error = ref<string | undefined>()

const templateForm = reactive({
  name: '',
  type: 'text',
  description: '',
  content: ''
})

// 模拟模板数据
const mockTemplates: Template[] = [
  {
    id: '1',
    name: '欢迎消息',
    type: 'text',
    description: '新成员加入时的欢迎消息',
    content: '欢迎 {{name}} 加入群聊！请遵守群规，和谐交流。',
    preview: '欢迎 张三 加入群聊！请遵守群规，和谐交流。',
    author: '系统',
    lastModified: new Date('2026-02-14'),
    enabled: true
  },
  {
    id: '2',
    name: '签到提醒',
    type: 'text',
    description: '每日签到提醒消息',
    content: '新的一天开始了！记得签到哦～\n签到时间: {{time}}',
    preview: '新的一天开始了！记得签到哦～\n签到时间: 09:00',
    author: '管理员',
    lastModified: new Date('2026-02-13'),
    enabled: true
  },
  {
    id: '3',
    name: '节日祝福',
    type: 'mixed',
    description: '节日期间的祝福模板',
    content: '{{emoji}} {{holiday}}快乐！{{emoji}}\n愿你今天心情愉快！',
    preview: '🎉 春节快乐！🎉\n愿你今天心情愉快！',
    author: '系统',
    lastModified: new Date('2026-02-12'),
    enabled: false
  }
]

async function loadTemplates() {
  try {
    error.value = undefined
    // 从 template.yaml 读取模板
    const content = await TauriAPI.ddbot.readConfigFile('template.yaml')
    // 解析 YAML 内容 (简单处理，实际可能需要 js-yaml)
    // 这里暂时保持 mock 数据展示，但逻辑上已经对接了后端
    templates.value = [...mockTemplates]
  } catch (e) {
    error.value = `加载模板失败: ${e}`
  }
}

function editTemplate(template: Template) {
  editingTemplate.value = template
  templateForm.name = template.name
  templateForm.type = template.type
  templateForm.description = template.description
  templateForm.content = template.content
  showCreateModal.value = true
}

async function toggleTemplate(template: Template) {
  try {
    template.enabled = !template.enabled
    // 这里将来可以对接具体的后端开关逻辑
  } catch (e) {
    error.value = `切换状态失败: ${e}`
  }
}

function closeModal() {
  showCreateModal.value = false
  editingTemplate.value = null
  resetForm()
}

function resetForm() {
  templateForm.name = ''
  templateForm.type = 'text'
  templateForm.description = ''
  templateForm.content = ''
}

async function saveTemplate() {
  try {
    if (editingTemplate.value) {
      // 更新现有模板
      const index = templates.value.findIndex(t => t.id === editingTemplate.value!.id)
      if (index !== -1) {
        templates.value[index] = {
          ...templates.value[index],
          name: templateForm.name,
          type: templateForm.type,
          description: templateForm.description,
          content: templateForm.content,
          preview: generatePreview(templateForm.content),
          lastModified: new Date()
        }
      }
    } else {
      // 创建新模板
      const newTemplate: Template = {
        id: Date.now().toString(),
        name: templateForm.name,
        type: templateForm.type,
        description: templateForm.description,
        content: templateForm.content,
        preview: generatePreview(templateForm.content),
        author: '当前用户',
        lastModified: new Date(),
        enabled: false
      }
      templates.value.push(newTemplate)
    }
    
    closeModal()
  } catch (e) {
    error.value = `保存模板失败: ${e}`
  }
}

function generatePreview(content: string): string {
  // 简单的变量替换预览
  return content
    .replace(/\{\{name\}\}/g, '张三')
    .replace(/\{\{time\}\}/g, '09:00')
    .replace(/\{\{emoji\}\}/g, '🎉')
    .replace(/\{\{holiday\}\}/g, '春节')
}

function formatDate(date: Date): string {
  return date.toLocaleDateString('zh-CN')
}

function clearError() {
  error.value = undefined
}

onMounted(() => {
  loadTemplates()
})
</script>

<style scoped>
.template-center {
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
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
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

.actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.template-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
  gap: 16px;
  padding: 16px;
}

.template-card {
  border-radius: 12px;
  border: 1px solid var(--border-color);
  background: var(--bg-card);
  padding: 16px;
  transition: all 0.2s ease;
}

.template-card:hover {
  border-color: rgba(124, 92, 255, 0.3);
  background: rgba(124, 92, 255, 0.05);
}

.template-card--active {
  border-color: rgba(16, 185, 129, 0.5);
  background: rgba(16, 185, 129, 0.1);
}

.template-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.template-name {
  font-size: 16px;
  font-weight: 600;
  margin: 0;
}

.template-badge {
  padding: 4px 10px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
}

.template-badge.text {
  background: rgba(59, 130, 246, 0.2);
  color: #3b82f6;
}

.template-badge.image {
  background: rgba(139, 92, 246, 0.2);
  color: #8b5cf6;
}

.template-badge.mixed {
  background: rgba(245, 158, 11, 0.2);
  color: #f59e0b;
}

.template-description {
  font-size: 13px;
  opacity: 0.8;
  margin-bottom: 16px;
  line-height: 1.5;
}

.template-preview {
  background: var(--bg-card);
  border-radius: 8px;
  padding: 12px;
  margin-bottom: 16px;
}

.preview-label {
  font-size: 12px;
  opacity: 0.7;
  margin-bottom: 6px;
}

.preview-content {
  font-size: 13px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  white-space: pre-wrap;
}

.template-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.template-meta {
  display: flex;
  gap: 12px;
  font-size: 12px;
  opacity: 0.7;
}

.meta-item {
  display: flex;
  align-items: center;
  gap: 4px;
}

.template-actions {
  display: flex;
  gap: 8px;
}

.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal {
  background: var(--bg-primary);
  border-radius: 16px;
  width: 90%;
  max-width: 600px;
  max-height: 90vh;
  overflow: hidden;
  box-shadow: 0 20px 50px rgba(0, 0, 0, 0.5);
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px;
  border-bottom: 1px solid var(--border-color);
}

.modal-header h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
}

.modal-body {
  padding: 20px;
  max-height: 60vh;
  overflow-y: auto;
}

.form-group {
  margin-bottom: 20px;
}

.form-group label {
  display: block;
  margin-bottom: 8px;
  font-size: 14px;
  font-weight: 500;
}

.form-group input,
.form-group select,
.form-group textarea {
  width: 100%;
  padding: 10px 12px;
  border-radius: 10px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  background: var(--bg-card);
  color: rgba(232, 234, 240, 0.95);
  font-size: 14px;
  outline: none;
}

.form-group input:focus,
.form-group select:focus,
.form-group textarea:focus {
  border-color: rgba(124, 92, 255, 0.5);
}

.form-group textarea {
  min-height: 100px;
  resize: vertical;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 20px;
  border-top: 1px solid var(--border-color);
}

.template-error {
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