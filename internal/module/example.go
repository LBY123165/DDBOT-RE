package module

import (
	"github.com/cnxysoft/DDBOT-WSa/internal/logger"
	"github.com/cnxysoft/DDBOT-WSa/internal/onebot"
)

// ExampleModule 示例模块（用于演示模块系统）
type ExampleModule struct {
	name    string
	version string
	status  Status
	bot     *onebot.Bot
	stop    chan struct{}
}

// NewExampleModule 创建示例模块
func NewExampleModule() *ExampleModule {
	return &ExampleModule{
		name:    "example",
		version: "1.0.0",
		status:  StatusStopped,
	}
}

func (m *ExampleModule) Name() string           { return m.name }
func (m *ExampleModule) Version() string        { return m.version }
func (m *ExampleModule) Status() Status         { return m.status }
func (m *ExampleModule) SetBot(bot *onebot.Bot) { m.bot = bot }

func (m *ExampleModule) Start() error {
	logger.Infof("启动示例模块: %s (版本: %s)", m.name, m.version)
	m.stop = make(chan struct{})
	m.status = StatusRunning
	return nil
}

func (m *ExampleModule) Stop() {
	if m.stop != nil {
		close(m.stop)
		m.stop = nil
	}
	m.status = StatusStopped
	logger.Infof("停止示例模块: %s", m.name)
}

func (m *ExampleModule) Reload() error {
	m.Stop()
	return m.Start()
}
