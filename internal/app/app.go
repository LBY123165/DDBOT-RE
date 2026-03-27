package app

import (
	"fmt"

	"github.com/cnxysoft/DDBOT-WSa/internal/command"
	"github.com/cnxysoft/DDBOT-WSa/internal/concern"
	"github.com/cnxysoft/DDBOT-WSa/internal/config"
	"github.com/cnxysoft/DDBOT-WSa/internal/cron"
	"github.com/cnxysoft/DDBOT-WSa/internal/db"
	"github.com/cnxysoft/DDBOT-WSa/internal/logger"
	"github.com/cnxysoft/DDBOT-WSa/internal/module"
	"github.com/cnxysoft/DDBOT-WSa/internal/onebot"
	"github.com/cnxysoft/DDBOT-WSa/internal/service"
	"github.com/cnxysoft/DDBOT-WSa/internal/telegram"
	"github.com/cnxysoft/DDBOT-WSa/internal/update"
)

// App 应用核心：整合所有子系统
type App struct {
	bot            *onebot.Bot
	moduleManager  *module.Manager
	updateManager  *update.Manager
	commandManager *command.Manager
	concernManager *concern.Manager
	webui          *service.WebUIService
	cronScheduler  *cron.Scheduler
}

// New 创建应用实例
func New() *App {
	mm := module.NewManager()
	superAdmins := config.GetSuperAdmins()
	return &App{
		moduleManager:  mm,
		updateManager:  update.NewManager(mm),
		commandManager: command.NewManager(superAdmins...),
		concernManager: concern.NewManager(),
		cronScheduler:  cron.NewScheduler(),
	}
}

// Start 按顺序启动所有子系统
func (a *App) Start() error {
	logger.Info("正在启动 DDBOT...")

	// 1. 数据库
	if err := db.Init(); err != nil {
		return fmt.Errorf("初始化数据库: %w", err)
	}
	if err := db.InitSchema(); err != nil {
		return fmt.Errorf("初始化数据库表: %w", err)
	}
	// 初始化 Telegram 频道表
	if err := telegram.InitChannelSchema(); err != nil {
		logger.Warnf("初始化 Telegram 表失败（可忽略）: %v", err)
	}

	// 2. Telegram 客户端
	tgCfg := config.GetTelegramConfig()
	telegram.Init(tgCfg.BotToken, tgCfg.Proxy, tgCfg.Enabled)
	if tgCfg.Enabled {
		logger.Info("Telegram 推送已启用")
	}

	// 2. OneBot 客户端
	obCfg := config.GetOneBotConfig()
	if obCfg.WsListen != "" {
		// 反向 WS：Bot 做服务端，等待 OneBot 主动连入（推荐）
		a.bot = onebot.NewReverseBot(obCfg.WsListen, obCfg.AccessToken)
	} else {
		// 正向 WS：Bot 主动连接到 OneBot 的 ws-server
		a.bot = onebot.NewBot(obCfg.WsURL, obCfg.AccessToken)
	}
	if err := a.bot.Start(); err != nil {
		return fmt.Errorf("启动 OneBot: %w", err)
	}

	// 3. 注册平台模块并注入 Bot
	a.registerModules()
	a.moduleManager.SetBotAll(a.bot)

	// 4. WebUI
	webuiAddr := config.GetWebUIAddr()
	a.webui = service.NewWebUIService(webuiAddr, a.moduleManager)
	a.webui.SetUpdateManager(a.updateManager)
	a.webui.SetBot(a.bot) // 注入 Bot 状态（连接/selfID）
	if err := a.webui.Start(); err != nil {
		return fmt.Errorf("启动 WebUI: %w", err)
	}

	// 5. 启动所有平台模块
	a.moduleManager.StartAll()

	// 6. 热更新
	a.updateManager.Start()

	// 7. 命令管理器（向 OneBot 注册事件处理器）
	if err := a.commandManager.Start(a.bot); err != nil {
		return fmt.Errorf("启动命令管理器: %w", err)
	}

	// 8. 定时任务调度器
	a.cronScheduler.SetBot(a.bot)
	a.cronScheduler.Start()

	// 8. 关注管理器
	// 注意：各平台的关注轮询已内置于各 module 中（调用 db.GetAllConcernsBySite），
	// concern.Manager 作为扩展预留，当前无需主动 Start（内部 concerns map 为空）。
	// 若未来需要统一由 concernManager 管理轮询，可在此处向其注册各平台实现。
	_ = a.concernManager

	logger.Info("DDBOT 启动完成 ✓")
	return nil
}

// Stop 按逆序停止所有子系统
func (a *App) Stop() {
	logger.Info("正在停止 DDBOT...")

	a.updateManager.Stop()
	// concernManager 未主动启动（关注轮询已内置于各模块），无需 Stop
	a.cronScheduler.Stop()
	a.commandManager.Stop()
	a.moduleManager.StopAll()

	if a.webui != nil {
		a.webui.Stop()
	}
	if a.bot != nil {
		a.bot.Stop()
	}
	if err := db.Close(); err != nil {
		logger.Errorf("关闭数据库: %v", err)
	}

	logger.Info("DDBOT 已停止")
}

// registerModules 注册所有平台模块
func (a *App) registerModules() {
	a.moduleManager.Register(module.NewBilibiliModule())
	a.moduleManager.Register(module.NewAcfunModule())
	a.moduleManager.Register(module.NewYoutubeModule())
	a.moduleManager.Register(module.NewDouyuModule())
	a.moduleManager.Register(module.NewHuyaModule())
	a.moduleManager.Register(module.NewWeiboModule())
	a.moduleManager.Register(module.NewDouyinModule())
	a.moduleManager.Register(module.NewTwitterModule())
	a.moduleManager.Register(module.NewTwitcastingModule())
}
