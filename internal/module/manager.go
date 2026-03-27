package module

import (
	"fmt"
	"sync"

	"github.com/cnxysoft/DDBOT-WSa/internal/logger"
	"github.com/cnxysoft/DDBOT-WSa/internal/onebot"
)

// Manager 模块管理器（并发安全）
type Manager struct {
	mu      sync.RWMutex
	modules map[string]Module
}

// NewManager 创建模块管理器
func NewManager() *Manager {
	return &Manager{modules: make(map[string]Module)}
}

// Register 注册模块
func (m *Manager) Register(mod Module) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.modules[mod.Name()] = mod
	logger.Infof("注册模块: %s (v%s)", mod.Name(), mod.Version())
}

// Get 获取模块
func (m *Manager) Get(name string) (Module, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	mod, ok := m.modules[name]
	return mod, ok
}

// List 返回所有模块信息（WebUI用）
func (m *Manager) List() []ModuleInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]ModuleInfo, 0, len(m.modules))
	for _, mod := range m.modules {
		list = append(list, ModuleInfo{
			Name:    mod.Name(),
			Version: mod.Version(),
			Status:  mod.Status(),
		})
	}
	return list
}

// ListModules 返回模块实例列表（内部用）
func (m *Manager) ListModules() []Module {
	m.mu.RLock()
	defer m.mu.RUnlock()
	mods := make([]Module, 0, len(m.modules))
	for _, mod := range m.modules {
		mods = append(mods, mod)
	}
	return mods
}

// SetBotAll 向所有模块注入 OneBot 客户端（必须在 StartAll 之前调用）
func (m *Manager) SetBotAll(bot *onebot.Bot) {
	for _, mod := range m.ListModules() {
		mod.SetBot(bot)
	}
}

// StartAll 启动所有模块
func (m *Manager) StartAll() {
	for _, mod := range m.ListModules() {
		if err := mod.Start(); err != nil {
			logger.Errorf("启动模块失败 [%s]: %v", mod.Name(), err)
		} else {
			logger.Infof("启动模块成功: %s", mod.Name())
		}
	}
}

// StopAll 停止所有模块
func (m *Manager) StopAll() {
	for _, mod := range m.ListModules() {
		mod.Stop()
		logger.Infof("停止模块: %s", mod.Name())
	}
}

// StartModule 启动指定模块
func (m *Manager) StartModule(name string) error {
	mod, ok := m.Get(name)
	if !ok {
		return fmt.Errorf("模块不存在: %s", name)
	}
	if err := mod.Start(); err != nil {
		return fmt.Errorf("启动模块失败 [%s]: %w", name, err)
	}
	logger.Infof("启动模块: %s", name)
	return nil
}

// StopModule 停止指定模块
func (m *Manager) StopModule(name string) error {
	mod, ok := m.Get(name)
	if !ok {
		return fmt.Errorf("模块不存在: %s", name)
	}
	mod.Stop()
	logger.Infof("停止模块: %s", name)
	return nil
}

// ReloadModule 热重载指定模块
func (m *Manager) ReloadModule(name string) error {
	mod, ok := m.Get(name)
	if !ok {
		return fmt.Errorf("模块不存在: %s", name)
	}
	if err := mod.Reload(); err != nil {
		return fmt.Errorf("重载模块失败 [%s]: %w", name, err)
	}
	logger.Infof("重载模块: %s", name)
	return nil
}

// ReplaceModule 热替换模块（零停机更新核心）
func (m *Manager) ReplaceModule(name string, newMod Module) error {
	m.mu.Lock()
	old, ok := m.modules[name]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("模块不存在: %s", name)
	}
	old.Stop()
	m.modules[name] = newMod
	m.mu.Unlock()

	logger.Infof("替换模块: %s (v%s → v%s)", name, old.Version(), newMod.Version())

	if err := newMod.Start(); err != nil {
		// 回滚
		m.mu.Lock()
		m.modules[name] = old
		m.mu.Unlock()
		if rerr := old.Start(); rerr != nil {
			logger.Errorf("回滚模块失败 [%s]: %v", name, rerr)
		}
		return fmt.Errorf("启动新模块失败，已回滚 [%s]: %w", name, err)
	}
	return nil
}
