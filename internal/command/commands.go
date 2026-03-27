package command

import (
	"fmt"
	"strings"

	"github.com/cnxysoft/DDBOT-WSa/internal/db"
	"github.com/cnxysoft/DDBOT-WSa/internal/onebot"
)

// ─── ping ───────────────────────────────────────────────────────────────────

type PingCommand struct{}

func (c *PingCommand) Name() string { return "ping" }
func (c *PingCommand) Help() string { return "测试机器人是否在线" }
func (c *PingCommand) Execute(b *onebot.Bot, ev *onebot.Event, _ []string) error {
	return reply(b, ev, "pong! 🏓")
}

// ─── help ───────────────────────────────────────────────────────────────────

type HelpCommand struct{}

func (c *HelpCommand) Name() string { return "help" }
func (c *HelpCommand) Help() string { return "显示帮助信息" }
func (c *HelpCommand) Execute(b *onebot.Bot, ev *onebot.Event, _ []string) error {
	help := `📖 DDBOT 帮助

【订阅命令（群内）】
  subscribe <平台> <UID> [类型]  关注主播/UP主
  unsubscribe <平台> <UID>       取消关注
  list [平台]                    查看关注列表

【娱乐命令（群内）】
  roll [范围/选项]               随机数/抽签
  签到                           每日签到（+1积分）
  积分 [排行]                    查看积分/排行榜

【管理命令（群内，需管理员）】
  config <平台> <UID> <选项>     配置推送方式（at/notify/filter）
  enable <命令>                  启用群内命令
  disable <命令>                 禁用群内命令
  silence [off|status]           群静音模式开关
  block user|group <ID> [原因]   封禁用户/群
  grant <QQ号> [remove]          管理员授权/撤销
  清除订阅 [list|<平台> <UID>]   查看/删除订阅
  addadmin <QQ号>                添加管理员
  removeadmin <QQ号>             移除管理员
  listadmins                     查看管理员列表

【管理命令（私聊，需管理员）】
  status                         查看 BOT 运行状态
  sysinfo                        查看系统信息
  quit <群号>                    退出指定群
  log <级别>                     调整日志级别
  invite on|off|status           控制入群邀请自动接受

【通用】
  ping                           测试机器人
  help                           显示此帮助

支持平台：
  bilibili  acfun  youtube
  douyin    douyu  huya  weibo  twitter  twitcasting

关注类型（bilibili 专属）：
  live（默认）  news  bangumi  article
  可组合：live,news  live,bangumi  live,news,bangumi,article

示例：
  !subscribe bilibili 123456
  !subscribe bilibili 123456 live,news
  !unsubscribe bilibili 123456
  !list bilibili`
	return reply(b, ev, help)
}

// ─── subscribe ──────────────────────────────────────────────────────────────

var validPlatforms = map[string]bool{
	"bilibili": true, "acfun": true, "youtube": true,
	"douyin": true, "douyu": true, "huya": true, "weibo": true, "twitter": true, "twitcasting": true,
}

var validConcernTypes = map[string]bool{
	"live": true, "news": true, "live,news": true, "news,live": true,
	"bangumi": true, "article": true,
	"live,bangumi": true, "bangumi,live": true,
	"live,news,bangumi": true, "live,news,article": true,
	"live,news,bangumi,article": true,
}

type SubscribeCommand struct{}

func (c *SubscribeCommand) Name() string { return "subscribe" }
func (c *SubscribeCommand) Help() string { return "关注主播/UP主" }
func (c *SubscribeCommand) Execute(b *onebot.Bot, ev *onebot.Event, args []string) error {
	if !isGroupEvent(ev) {
		return reply(b, ev, "❌ 该命令只能在群内使用")
	}
	if len(args) < 2 {
		return reply(b, ev, "❌ 用法：!subscribe <平台> <用户ID> [类型]\n例：!subscribe bilibili 123456 live")
	}

	platform := strings.ToLower(args[0])
	uid := args[1]
	concernType := "live"
	if len(args) >= 3 {
		concernType = strings.ToLower(args[2])
	}

	if !validPlatforms[platform] {
		return reply(b, ev, fmt.Sprintf("❌ 不支持的平台：%s\n支持：bilibili/acfun/youtube/douyin/douyu/huya/weibo", platform))
	}
	if !validConcernTypes[concernType] {
		return reply(b, ev, fmt.Sprintf("❌ 不支持的关注类型：%s\n支持：live / news / live,news", concernType))
	}

	if err := db.InsertConcern(platform, uid, "", ev.GroupID, concernType); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") || strings.Contains(err.Error(), "IGNORE") {
			return reply(b, ev, "⚠️ 已经关注过了")
		}
		return fmt.Errorf("添加关注失败: %w", err)
	}
	return reply(b, ev, fmt.Sprintf("✅ 成功关注 [%s] 平台用户 %s（类型：%s）", platform, uid, concernType))
}

// ─── unsubscribe ────────────────────────────────────────────────────────────

type UnsubscribeCommand struct{}

func (c *UnsubscribeCommand) Name() string { return "unsubscribe" }
func (c *UnsubscribeCommand) Help() string { return "取消关注主播/UP主" }
func (c *UnsubscribeCommand) Execute(b *onebot.Bot, ev *onebot.Event, args []string) error {
	if !isGroupEvent(ev) {
		return reply(b, ev, "❌ 该命令只能在群内使用")
	}
	if len(args) < 2 {
		return reply(b, ev, "❌ 用法：!unsubscribe <平台> <用户ID>")
	}
	platform, uid := strings.ToLower(args[0]), args[1]
	if err := db.DeleteConcern(platform, uid, ev.GroupID); err != nil {
		if strings.Contains(err.Error(), "不存在") {
			return reply(b, ev, fmt.Sprintf("⚠️ 未找到 [%s] 平台用户 %s 的关注记录", platform, uid))
		}
		return fmt.Errorf("取消关注失败: %w", err)
	}
	return reply(b, ev, fmt.Sprintf("✅ 已取消关注 [%s] 平台用户 %s", platform, uid))
}

// ─── list ───────────────────────────────────────────────────────────────────

type ListCommand struct{}

func (c *ListCommand) Name() string { return "list" }
func (c *ListCommand) Help() string { return "查看关注列表" }
func (c *ListCommand) Execute(b *onebot.Bot, ev *onebot.Event, args []string) error {
	if !isGroupEvent(ev) {
		return reply(b, ev, "❌ 该命令只能在群内使用")
	}
	var platform string
	if len(args) > 0 {
		platform = strings.ToLower(args[0])
	}
	concerns, err := db.GetConcerns(ev.GroupID, platform)
	if err != nil {
		return fmt.Errorf("查询关注列表失败: %w", err)
	}
	if len(concerns) == 0 {
		if platform != "" {
			return reply(b, ev, fmt.Sprintf("📋 [%s] 暂无关注", platform))
		}
		return reply(b, ev, "📋 暂无关注，使用 !subscribe 添加")
	}
	var sb strings.Builder
	sb.WriteString("📋 关注列表\n\n")
	for i, c := range concerns {
		name := c.Name
		if name == "" {
			name = c.UID
		}
		sb.WriteString(fmt.Sprintf("%d. [%s] %s（ID: %s，类型: %s）\n",
			i+1, c.Site, name, c.UID, c.ConcernType))
	}
	return reply(b, ev, sb.String())
}

// ─── 工具函数 ─────────────────────────────────────────────────────────────────

// reply 根据消息来源自动选择发送到群/私聊
func reply(b *onebot.Bot, ev *onebot.Event, text string) error {
	if ev.IsGroupMessage() {
		return b.SendGroupText(ev.GroupID, text)
	}
	return b.SendPrivateMsg(ev.UserID, onebot.Text(text))
}

// isGroupEvent 判断事件是否来自群聊
func isGroupEvent(ev *onebot.Event) bool {
	return ev.IsGroupMessage()
}
