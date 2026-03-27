package concern

import (
	"github.com/cnxysoft/DDBOT-WSa/internal/db"
	"github.com/cnxysoft/DDBOT-WSa/internal/logger"
)

// Concern 关注接口（各平台模块可选实现）
type Concern interface {
	Site() string
	Start() error
	Stop()
	FreshIndex()
}

// Manager 关注管理器（统一管理各平台 Concern）
type Manager struct {
	concerns map[string]Concern
}

// NewManager 创建关注管理器
func NewManager() *Manager {
	return &Manager{concerns: make(map[string]Concern)}
}

// Register 注册关注模块
func (m *Manager) Register(c Concern) {
	m.concerns[c.Site()] = c
	logger.Infof("注册关注模块: %s", c.Site())
}

// Start 启动所有关注模块
func (m *Manager) Start() error {
	for _, c := range m.concerns {
		if err := c.Start(); err != nil {
			logger.Errorf("启动关注失败 [%s]: %v", c.Site(), err)
			continue
		}
		logger.Infof("启动关注成功: %s", c.Site())
	}
	return nil
}

// Stop 停止所有关注模块
func (m *Manager) Stop() {
	for _, c := range m.concerns {
		c.Stop()
		logger.Infof("停止关注: %s", c.Site())
	}
	m.concerns = make(map[string]Concern)
}

// FreshIndex 刷新所有关注的索引
func (m *Manager) FreshIndex() {
	for _, c := range m.concerns {
		c.FreshIndex()
	}
}

// ─── 便捷查询（直接操作 DB）─────────────────────────────────────────────────

// AddConcern 添加关注（site/uid/name/groupCode/concernType）
func (m *Manager) AddConcern(site, uid, name string, groupCode int64, concernType string) error {
	return db.InsertConcern(site, uid, name, groupCode, concernType)
}

// RemoveConcern 取消关注
func (m *Manager) RemoveConcern(site, uid string, groupCode int64) error {
	return db.DeleteConcern(site, uid, groupCode)
}

// GetGroupConcerns 获取某群的所有关注
func (m *Manager) GetGroupConcerns(groupCode int64) ([]*db.Concern, error) {
	return db.GetConcerns(groupCode, "")
}

// Summary 订阅统计
func (m *Manager) Summary() (map[string]int, error) {
	return db.GetConcernCountBySite()
}
