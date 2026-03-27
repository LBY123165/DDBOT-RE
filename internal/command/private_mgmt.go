package command

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/cnxysoft/DDBOT-WSa/internal/logger"
	"github.com/cnxysoft/DDBOT-WSa/internal/onebot"
)

// ─── sysinfo ─────────────────────────────────────────────────────────────────

type SysinfoCommand struct {
	startTime time.Time
}

func NewSysinfoCommand() *SysinfoCommand {
	return &SysinfoCommand{startTime: time.Now()}
}

func (c *SysinfoCommand) Name() string { return "sysinfo" }
func (c *SysinfoCommand) Help() string { return "查看系统运行信息（仅管理员）" }
func (c *SysinfoCommand) Execute(b *onebot.Bot, ev *onebot.Event, _ []string) error {
	ok, err := checkAdmin(b, ev)
	if !ok {
		return err
	}
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	uptime := time.Since(c.startTime).Round(time.Second)
	msg := fmt.Sprintf(`📊 系统信息
运行时长：%s
Go 版本：%s
操作系统：%s/%s
CPU 核心：%d
Goroutine：%d
堆内存使用：%.1f MB
堆内存总分配：%.1f MB
GC 次数：%d`,
		uptime,
		runtime.Version(),
		runtime.GOOS, runtime.GOARCH,
		runtime.NumCPU(),
		runtime.NumGoroutine(),
		float64(ms.HeapInuse)/1024/1024,
		float64(ms.TotalAlloc)/1024/1024,
		ms.NumGC,
	)
	return reply(b, ev, msg)
}

// ─── status ──────────────────────────────────────────────────────────────────

type StatusCommand struct {
	startTime time.Time
	bot       *onebot.Bot
}

func NewStatusCommand(bot *onebot.Bot) *StatusCommand {
	return &StatusCommand{startTime: time.Now(), bot: bot}
}

func (c *StatusCommand) Name() string { return "status" }
func (c *StatusCommand) Help() string { return "查看 BOT 运行状态（仅管理员）" }
func (c *StatusCommand) Execute(b *onebot.Bot, ev *onebot.Event, _ []string) error {
	ok, err := checkAdmin(b, ev)
	if !ok {
		return err
	}
	uptime := time.Since(c.startTime).Round(time.Second)

	connected := "✅ 已连接"
	selfID := int64(0)
	if c.bot != nil {
		if !c.bot.IsConnected() {
			connected = "❌ 未连接"
		}
		selfID = c.bot.GetSelfID()
	}

	msg := fmt.Sprintf(`🤖 BOT 状态
运行时长：%s
OneBot 连接：%s
Bot QQ号：%d`,
		uptime, connected, selfID)
	return reply(b, ev, msg)
}

// ─── quit（退出群聊） ──────────────────────────────────────────────────────────

type QuitCommand struct{}

func (c *QuitCommand) Name() string { return "quit" }
func (c *QuitCommand) Help() string {
	return "让 BOT 退出指定群（仅管理员）\n用法：!quit <群号>"
}
func (c *QuitCommand) Execute(b *onebot.Bot, ev *onebot.Event, args []string) error {
	ok, err := checkAdmin(b, ev)
	if !ok {
		return err
	}
	if len(args) < 1 {
		return reply(b, ev, "❌ 用法：!quit <群号>")
	}
	var groupID int64
	if _, e := fmt.Sscanf(args[0], "%d", &groupID); e != nil {
		return reply(b, ev, "❌ 群号格式错误")
	}
	if err := b.SetGroupLeave(groupID, false); err != nil {
		return fmt.Errorf("退出群聊失败: %w", err)
	}
	logger.Infof("quit: group=%d (by %d)", groupID, ev.UserID)
	return reply(b, ev, fmt.Sprintf("✅ 已发送退出群 %d 的请求", groupID))
}

// ─── log（调整日志级别） ──────────────────────────────────────────────────────

type LogCommand struct{}

func (c *LogCommand) Name() string { return "log" }
func (c *LogCommand) Help() string {
	return "调整日志级别（仅管理员）\n用法：!log <debug|info|warn|error>"
}
func (c *LogCommand) Execute(b *onebot.Bot, ev *onebot.Event, args []string) error {
	ok, err := checkAdmin(b, ev)
	if !ok {
		return err
	}
	if len(args) < 1 {
		return reply(b, ev, "❌ 用法：!log <debug|info|warn|error>")
	}
	lvl := strings.ToLower(args[0])
	switch lvl {
	case "debug":
		logger.SetLevel("debug")
	case "info":
		logger.SetLevel("info")
	case "warn", "warning":
		logger.SetLevel("warn")
	case "error":
		logger.SetLevel("error")
	default:
		return reply(b, ev, "❌ 不支持的级别，可选：debug / info / warn / error")
	}
	logger.Infof("log level changed to %s (by %d)", lvl, ev.UserID)
	return reply(b, ev, fmt.Sprintf("✅ 日志级别已设置为 %s", lvl))
}
