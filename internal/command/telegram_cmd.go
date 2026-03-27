package command

import (
	"fmt"
	"strings"

	"github.com/cnxysoft/DDBOT-WSa/internal/onebot"
	"github.com/cnxysoft/DDBOT-WSa/internal/telegram"
)

// ─── tg 命令（Telegram 频道绑定管理）────────────────────────────────────────
// 用法（仅超级管理员，私聊）：
//   !tg bind <chat_id> <site> <uid>     绑定 Telegram 频道到订阅
//   !tg unbind <chat_id> <site> <uid>   解除绑定
//   !tg list [<site> [<uid>]]           列出所有绑定
//   !tg test <chat_id>                  发送测试消息
//
// chat_id 格式：
//   @频道用户名（如 @mychannel）
//   数字 ID（如 -1001234567890）

type TelegramCommand struct {
	superAdmins []int64
}

func NewTelegramCommand(superAdmins []int64) *TelegramCommand {
	return &TelegramCommand{superAdmins: superAdmins}
}

func (c *TelegramCommand) Name() string { return "tg" }
func (c *TelegramCommand) Help() string {
	return `Telegram 频道绑定管理（超管命令，私聊使用）
用法：
  !tg bind <chat_id> <平台> <UID>    绑定 Telegram 频道到订阅
  !tg unbind <chat_id> <平台> <UID>  解除绑定
  !tg list [<平台> [<UID>]]          列出所有绑定
  !tg test <chat_id>                 发送测试消息

chat_id 格式：@频道名 或 数字群组ID`
}

func (c *TelegramCommand) Execute(b *onebot.Bot, ev *onebot.Event, args []string) error {
	// 权限检查：仅超级管理员可用
	if !c.isSuperAdmin(ev.UserID) {
		return nil // 静默拒绝
	}

	if len(args) == 0 {
		return reply(b, ev, c.Help())
	}

	sub := strings.ToLower(args[0])
	switch sub {
	case "bind":
		return c.cmdBind(b, ev, args[1:])
	case "unbind":
		return c.cmdUnbind(b, ev, args[1:])
	case "list", "ls":
		return c.cmdList(b, ev, args[1:])
	case "test":
		return c.cmdTest(b, ev, args[1:])
	case "status":
		return c.cmdStatus(b, ev)
	default:
		return reply(b, ev, "❌ 未知子命令。用法：bind/unbind/list/test/status")
	}
}

func (c *TelegramCommand) cmdBind(b *onebot.Bot, ev *onebot.Event, args []string) error {
	if len(args) < 3 {
		return reply(b, ev, "❌ 用法：!tg bind <chat_id> <平台> <UID>")
	}
	chatID := args[0]
	site := strings.ToLower(args[1])
	uid := args[2]

	groupCode := ev.GroupID // 群聊使用当前群，私聊为 0
	if err := telegram.BindChannel(chatID, site, uid, groupCode); err != nil {
		return fmt.Errorf("绑定失败: %w", err)
	}
	return reply(b, ev, fmt.Sprintf("✅ 已绑定：[%s] %s → Telegram %s", site, uid, chatID))
}

func (c *TelegramCommand) cmdUnbind(b *onebot.Bot, ev *onebot.Event, args []string) error {
	if len(args) < 3 {
		return reply(b, ev, "❌ 用法：!tg unbind <chat_id> <平台> <UID>")
	}
	chatID, site, uid := args[0], strings.ToLower(args[1]), args[2]
	if err := telegram.UnbindChannel(chatID, site, uid); err != nil {
		if strings.Contains(err.Error(), "不存在") {
			return reply(b, ev, "⚠️ 该绑定不存在")
		}
		return fmt.Errorf("解绑失败: %w", err)
	}
	return reply(b, ev, fmt.Sprintf("✅ 已解绑：[%s] %s → Telegram %s", site, uid, chatID))
}

func (c *TelegramCommand) cmdList(b *onebot.Bot, ev *onebot.Event, args []string) error {
	var site, uid string
	if len(args) >= 1 {
		site = args[0]
	}
	if len(args) >= 2 {
		uid = args[1]
	}
	channels, err := telegram.ListChannels(site, uid)
	if err != nil {
		return fmt.Errorf("查询失败: %w", err)
	}
	return reply(b, ev, telegram.FormatChannelList(channels))
}

func (c *TelegramCommand) cmdTest(b *onebot.Bot, ev *onebot.Event, args []string) error {
	if len(args) == 0 {
		return reply(b, ev, "❌ 用法：!tg test <chat_id>")
	}
	chatID := args[0]
	client := telegram.GetClient()
	if client == nil || !client.IsEnabled() {
		return reply(b, ev, "❌ Telegram 未启用，请在 application.yaml 中配置 telegram.bot_token")
	}
	if err := client.SendMessage(chatID, "🤖 DDBOT Telegram 连接测试成功！\nBot 已就绪，将为您推送订阅更新。"); err != nil {
		return reply(b, ev, fmt.Sprintf("❌ 发送失败：%v", err))
	}
	return reply(b, ev, fmt.Sprintf("✅ 测试消息已发送到 %s", chatID))
}

func (c *TelegramCommand) cmdStatus(b *onebot.Bot, ev *onebot.Event) error {
	client := telegram.GetClient()
	if client == nil || !client.IsEnabled() {
		return reply(b, ev, "📡 Telegram 状态：❌ 未启用\n请在 application.yaml 配置 telegram.bot_token")
	}
	channels, _ := telegram.ListChannels("", "")
	return reply(b, ev, fmt.Sprintf("📡 Telegram 状态：✅ 已启用\n绑定频道数：%d\n使用 !tg list 查看详情", len(channels)))
}

func (c *TelegramCommand) isSuperAdmin(userID int64) bool {
	for _, id := range c.superAdmins {
		if id == userID {
			return true
		}
	}
	return false
}
