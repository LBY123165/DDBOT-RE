package module

import "github.com/cnxysoft/DDBOT-WSa/internal/onebot"

// Status 模块运行状态
type Status string

const (
	StatusStopped  Status = "stopped"
	StatusRunning  Status = "running"
	StatusError    Status = "error"
	StatusUpdating Status = "updating"
)

// Module 模块接口（所有平台模块必须实现此接口）
type Module interface {
	// 基本信息
	Name() string
	Version() string

	// 生命周期
	Start() error
	Stop()
	Reload() error

	// 热更新支持
	Status() Status

	// SetBot 注入 OneBot 客户端（启动前由 Manager 统一注入）
	SetBot(bot *onebot.Bot)
}

// ModuleInfo 模块信息（用于 WebUI 展示）
type ModuleInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Status  Status `json:"status"`
}
