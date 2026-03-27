package update

import (
	"fmt"
	"sync"
	"time"

	"github.com/cnxysoft/DDBOT-WSa/internal/logger"
	"github.com/cnxysoft/DDBOT-WSa/internal/module"
)

// UpdateInfo 模块更新信息
type UpdateInfo struct {
	ModuleName     string `json:"module_name"`
	CurrentVersion string `json:"current_version"`
	NewVersion     string `json:"new_version"`
	DownloadURL    string `json:"download_url"`
	Changelog      string `json:"changelog"`
}

// Manager 热更新管理器（可安全并发访问）
type Manager struct {
	moduleManager  *module.Manager
	mu             sync.RWMutex
	pendingUpdates map[string]UpdateInfo
	checkInterval  time.Duration
	stopChan       chan struct{}
	// notifyFn 可选：外部注入的通知函数（如发送 QQ 消息）
	notifyFn func(info UpdateInfo)
}

// NewManager 创建热更新管理器
func NewManager(mm *module.Manager) *Manager {
	return &Manager{
		moduleManager:  mm,
		pendingUpdates: make(map[string]UpdateInfo),
		checkInterval:  24 * time.Hour,
	}
}

// SetCheckInterval 设置检查间隔
func (m *Manager) SetCheckInterval(d time.Duration) { m.checkInterval = d }

// SetNotifyFunc 设置更新通知回调（可选）
func (m *Manager) SetNotifyFunc(fn func(UpdateInfo)) { m.notifyFn = fn }

// Start 启动热更新检查循环（非阻塞）
func (m *Manager) Start() {
	if m.stopChan != nil {
		return // 已启动
	}
	m.stopChan = make(chan struct{})
	go m.loop()
	logger.Info("热更新管理器已启动")
}

// Stop 停止热更新检查
func (m *Manager) Stop() {
	if m.stopChan != nil {
		close(m.stopChan)
		m.stopChan = nil
	}
	logger.Info("热更新管理器已停止")
}

// GetPendingUpdates 获取待更新列表（供 WebUI API 使用）
func (m *Manager) GetPendingUpdates() []UpdateInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]UpdateInfo, 0, len(m.pendingUpdates))
	for _, u := range m.pendingUpdates {
		list = append(list, u)
	}
	return list
}

// ApplyUpdate 执行指定模块的更新
func (m *Manager) ApplyUpdate(moduleName string) error {
	m.mu.Lock()
	info, ok := m.pendingUpdates[moduleName]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("无待更新模块: %s", moduleName)
	}
	delete(m.pendingUpdates, moduleName)
	m.mu.Unlock()

	logger.Infof("开始更新模块 [%s]: v%s → v%s", moduleName, info.CurrentVersion, info.NewVersion)

	// 下载
	dest := fmt.Sprintf("./modules/%s/%s.zip", moduleName, info.NewVersion)
	if err := DownloadModule(info.DownloadURL, dest); err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}

	// 解压
	extractDest := fmt.Sprintf("./modules/%s/%s", moduleName, info.NewVersion)
	if err := ExtractModule(dest, extractDest); err != nil {
		return fmt.Errorf("解压失败: %w", err)
	}

	// 创建新模块实例
	newMod, err := CreateModule(moduleName, info.NewVersion)
	if err != nil {
		return fmt.Errorf("创建模块实例失败: %w", err)
	}

	// 热替换（零停机）
	if err := m.moduleManager.ReplaceModule(moduleName, newMod); err != nil {
		return fmt.Errorf("替换模块失败: %w", err)
	}

	logger.Infof("模块更新完成 [%s] v%s", moduleName, info.NewVersion)
	return nil
}

// ─── 私有方法 ──────────────────────────────────────────────────────────────

func (m *Manager) loop() {
	ticker := time.NewTicker(m.checkInterval)
	defer ticker.Stop()

	// 首次立即检查
	m.checkAll()

	for {
		select {
		case <-ticker.C:
			m.checkAll()
		case <-m.stopChan:
			return
		}
	}
}

func (m *Manager) checkAll() {
	for _, mod := range m.moduleManager.ListModules() {
		m.checkOne(mod)
	}
}

func (m *Manager) checkOne(mod module.Module) {
	// TODO: 从远端 API 获取最新版本信息
	// 当前为占位逻辑：模拟远端返回固定版本
	remoteVersion := "2.0.1"
	if mod.Version() == remoteVersion {
		return
	}

	info := UpdateInfo{
		ModuleName:     mod.Name(),
		CurrentVersion: mod.Version(),
		NewVersion:     remoteVersion,
		DownloadURL:    fmt.Sprintf("https://example.com/modules/%s/%s.zip", mod.Name(), remoteVersion),
		Changelog:      "修复若干 Bug，提升稳定性",
	}

	m.mu.Lock()
	m.pendingUpdates[mod.Name()] = info
	m.mu.Unlock()

	logger.Infof("检测到模块更新: [%s] v%s → v%s", mod.Name(), mod.Version(), remoteVersion)

	if m.notifyFn != nil {
		m.notifyFn(info)
	}
}
