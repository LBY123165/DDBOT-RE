package command

import (
	"strings"

	"github.com/cnxysoft/DDBOT-WSa/internal/db"
	"github.com/cnxysoft/DDBOT-WSa/internal/logger"
	"github.com/cnxysoft/DDBOT-WSa/internal/onebot"
)

// Command 命令接口
type Command interface {
	Name() string
	Help() string
	Execute(bot *onebot.Bot, event *onebot.Event, args []string) error
}

// Manager 命令管理器
type Manager struct {
	commands       map[string]Command
	superAdmins    []int64 // 超级管理员 QQ 号（从配置读取）
	bot            *onebot.Bot
	requestHandler *RequestHandler
}

// NewManager 创建命令管理器
func NewManager(superAdmins ...int64) *Manager {
	return &Manager{
		commands:    make(map[string]Command),
		superAdmins: superAdmins,
	}
}

// Register 注册命令
func (m *Manager) Register(cmd Command) {
	m.commands[cmd.Name()] = cmd
	logger.Infof("注册命令: %s", cmd.Name())
}

// Start 启动命令管理器，向 OneBot 客户端注册事件处理器
func (m *Manager) Start(bot *onebot.Bot) error {
	m.bot = bot
	logger.Info("启动命令管理器")

	// 基础命令
	m.Register(&PingCommand{})
	m.Register(&HelpCommand{})
	m.Register(&SubscribeCommand{})
	m.Register(&UnsubscribeCommand{})
	m.Register(&ListCommand{})

	// Admin 命令
	m.Register(NewAddAdminCommand(m.superAdmins))
	m.Register(NewRemoveAdminCommand(m.superAdmins))
	m.Register(&ListAdminsCommand{})

	// 群管理命令
	m.Register(&ConfigCommand{})
	m.Register(&EnableCommand{})
	m.Register(&DisableCommand{})
	m.Register(&BlockCommand{})
	m.Register(NewGrantCommand(m.superAdmins))
	m.Register(&SilenceCommand{})

	// 娱乐/积分命令
	m.Register(&RollCommand{})
	m.Register(&CheckinCommand{})
	m.Register(&ScoreCommand{})
	m.Register(&CheckConcernsCommand{})
	m.Register(&ReverseCommand{})

	// 定时任务命令
	m.Register(&CronCommand{})

	// Telegram 管理命令（超管）
	m.Register(NewTelegramCommand(m.superAdmins))

	// 私聊管理命令
	sysinfoCmd := NewSysinfoCommand()
	m.Register(sysinfoCmd)
	m.Register(NewStatusCommand(bot))
	m.Register(&QuitCommand{})
	m.Register(&LogCommand{})

	// 请求处理器（好友申请/入群邀请）
	m.requestHandler = NewRequestHandler(m.superAdmins, false)
	inviteCmd := NewInviteCommand(m.requestHandler)
	m.Register(inviteCmd)

	// 消息事件处理
	bot.OnEvent(func(b *onebot.Bot, ev *onebot.Event) {
		// 处理请求事件（好友/群邀请）
		if ev.PostType == "request" {
			m.requestHandler.HandleRequest(b, ev)
			return
		}

		if !ev.IsGroupMessage() && !ev.IsPrivateMessage() {
			return
		}

		// 黑名单检查
		if blocked, _ := db.IsBlocked("user", ev.UserID); blocked {
			logger.Debugf("command: blocked user %d", ev.UserID)
			return
		}
		if ev.IsGroupMessage() {
			if blocked, _ := db.IsBlocked("group", ev.GroupID); blocked {
				logger.Debugf("command: blocked group %d", ev.GroupID)
				return
			}
		}

		text := strings.TrimSpace(ev.TextContent())
		if len(text) == 0 {
			return
		}
		// 支持 ! / . / / 前缀触发命令
		if text[0] == '!' || text[0] == '/' || text[0] == '.' {
			text = text[1:]
		} else {
			return
		}
		parts := strings.Fields(text)
		if len(parts) == 0 {
			return
		}
		cmdName := strings.ToLower(parts[0])
		args := parts[1:]

		cmd, ok := m.commands[cmdName]
		if !ok {
			logger.Debugf("未知命令: %s", cmdName)
			return
		}

		// 群静音检查（管理员命令跳过：silence/grant/block/enable/disable）
		if ev.IsGroupMessage() {
			adminCmds := map[string]bool{
				"silence": true, "grant": true, "block": true,
				"enable": true, "disable": true, "addadmin": true,
				"removeadmin": true, "listadmins": true,
			}
			if !adminCmds[cmdName] {
				if silenced, _ := db.IsGroupSilenced(ev.GroupID); silenced {
					logger.Debugf("command: group %d silenced, ignoring %s", ev.GroupID, cmdName)
					return
				}
			}
		}

		// 群命令开关检查（群聊中检查，私聊跳过）
		if ev.IsGroupMessage() {
			enabled, _ := db.IsCommandEnabled(ev.GroupID, cmdName)
			if !enabled {
				logger.Debugf("command: disabled in group %d: %s", ev.GroupID, cmdName)
				return
			}
		}

		go func() {
			if err := cmd.Execute(b, ev, args); err != nil {
				logger.Errorf("执行命令 [%s] 失败: %v", cmdName, err)
				_ = reply(b, ev, "❌ 执行失败："+err.Error())
			}
		}()
	})

	return nil
}

// Stop 停止命令管理器
func (m *Manager) Stop() {
	m.commands = make(map[string]Command)
}

// RegisterSuperAdmin 初始化时从 DB 补充超管列表（可选）
func (m *Manager) RegisterSuperAdmin(userID int64) {
	for _, id := range m.superAdmins {
		if id == userID {
			return
		}
	}
	m.superAdmins = append(m.superAdmins, userID)
	// 同步写入 DB
	_ = db.AddAdmin(userID, "super")
}
