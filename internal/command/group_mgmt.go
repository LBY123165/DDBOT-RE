package command

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cnxysoft/DDBOT-WSa/internal/db"
	"github.com/cnxysoft/DDBOT-WSa/internal/logger"
	"github.com/cnxysoft/DDBOT-WSa/internal/onebot"
)

// ─── config 命令（群推送配置） ─────────────────────────────────────────────────

// ConfigCommand 配置订阅推送方式
// 用法：!config <平台> <UID> <选项> [值]
// 选项：
//   at none|all|<QQ号>    设置 @ 模式
//   notify live|news|all|none  设置通知类型开关
//   filter add <关键词>    添加关键词过滤
//   filter del <关键词>    移除关键词过滤
//   filter clear           清空过滤规则
//   show                   显示当前配置
type ConfigCommand struct{}

func (c *ConfigCommand) Name() string { return "config" }
func (c *ConfigCommand) Help() string {
	return `配置订阅推送方式（仅管理员可用）
用法：
  !config <平台> <UID> show            查看当前配置
  !config <平台> <UID> at none         关闭 @
  !config <平台> <UID> at all          开播时 @全体成员
  !config <平台> <UID> at <QQ号>       开播时 @ 指定成员（多个用逗号分隔）
  !config <平台> <UID> notify live on|off   开播通知开关
  !config <平台> <UID> notify news on|off   动态通知开关
  !config <平台> <UID> filter add <关键词>  添加关键词过滤（仅含关键词的内容才推送）
  !config <平台> <UID> filter del <关键词>  删除关键词过滤
  !config <平台> <UID> filter clear         清空关键词过滤`
}

func (c *ConfigCommand) Execute(b *onebot.Bot, ev *onebot.Event, args []string) error {
	if !isGroupEvent(ev) {
		return reply(b, ev, "❌ 该命令只能在群内使用")
	}
	ok, err := checkAdmin(b, ev)
	if !ok {
		return err
	}
	if len(args) < 3 {
		return reply(b, ev, "❌ 参数不足\n"+c.Help())
	}

	platform := strings.ToLower(args[0])
	uid := args[1]
	action := strings.ToLower(args[2])
	rest := args[3:] // 剩余参数

	cfg, err2 := db.GetConcernConfig(ev.GroupID, platform, uid)
	if err2 != nil {
		return fmt.Errorf("读取配置失败: %w", err2)
	}

	switch action {
	case "show":
		return c.showConfig(b, ev, cfg, platform, uid)

	case "at":
		if len(rest) == 0 {
			return reply(b, ev, "❌ 用法：!config <平台> <UID> at none|all|<QQ号>")
		}
		switch strings.ToLower(rest[0]) {
		case "none":
			cfg.AtMode = 0
			cfg.AtMembers = ""
		case "all":
			cfg.AtMode = 1
			cfg.AtMembers = ""
		default:
			// 解析 QQ 号列表
			qqs := strings.Split(rest[0], ",")
			var ids []string
			for _, q := range qqs {
				q = strings.TrimSpace(q)
				if _, e := strconv.ParseInt(q, 10, 64); e != nil {
					return reply(b, ev, fmt.Sprintf("❌ QQ号格式错误：%s", q))
				}
				ids = append(ids, q)
			}
			cfg.AtMode = 2
			cfg.AtMembers = "[" + strings.Join(ids, ",") + "]"
		}

	case "notify":
		if len(rest) < 2 {
			return reply(b, ev, "❌ 用法：!config <平台> <UID> notify live|news on|off")
		}
		enabled := strings.ToLower(rest[1]) == "on" || rest[1] == "1"
		switch strings.ToLower(rest[0]) {
		case "live":
			cfg.NotifyLive = enabled
		case "news":
			cfg.NotifyNews = enabled
		case "all":
			cfg.NotifyLive = enabled
			cfg.NotifyNews = enabled
		default:
			return reply(b, ev, "❌ 通知类型只支持：live / news / all")
		}

	case "filter":
		if len(rest) == 0 {
			return reply(b, ev, "❌ 用法：!config <平台> <UID> filter add|del|clear [关键词]")
		}
		sub := strings.ToLower(rest[0])
		switch sub {
		case "clear":
			cfg.FilterText = ""
		case "add":
			if len(rest) < 2 {
				return reply(b, ev, "❌ 用法：!config <平台> <UID> filter add <关键词>")
			}
			kw := strings.Join(rest[1:], " ")
			cfg.FilterText = addKeyword(cfg.FilterText, kw)
		case "del", "remove":
			if len(rest) < 2 {
				return reply(b, ev, "❌ 用法：!config <平台> <UID> filter del <关键词>")
			}
			kw := strings.Join(rest[1:], " ")
			cfg.FilterText = removeKeyword(cfg.FilterText, kw)
		default:
			return reply(b, ev, "❌ filter 子命令：add / del / clear")
		}

	default:
		return reply(b, ev, "❌ 未知操作："+action+"\n"+c.Help())
	}

	if e := db.SetConcernConfig(cfg); e != nil {
		return fmt.Errorf("保存配置失败: %w", e)
	}
	logger.Infof("config: group=%d site=%s uid=%s action=%s (by %d)", ev.GroupID, platform, uid, action, ev.UserID)
	return reply(b, ev, "✅ 配置已更新")
}

func (c *ConfigCommand) showConfig(b *onebot.Bot, ev *onebot.Event, cfg *db.ConcernConfig, site, uid string) error {
	atDesc := "关闭"
	switch cfg.AtMode {
	case 1:
		atDesc = "@全体成员"
	case 2:
		atDesc = "@指定成员 " + cfg.AtMembers
	}
	notifyDesc := []string{}
	if cfg.NotifyLive {
		notifyDesc = append(notifyDesc, "开播")
	}
	if cfg.NotifyNews {
		notifyDesc = append(notifyDesc, "动态")
	}
	if len(notifyDesc) == 0 {
		notifyDesc = []string{"全部关闭"}
	}
	filterDesc := "无"
	if cfg.FilterText != "" && cfg.FilterText != "[]" {
		filterDesc = cfg.FilterText
	}
	msg := fmt.Sprintf("📋 [%s] %s 推送配置\n\n@模式：%s\n通知类型：%s\n关键词过滤：%s",
		site, uid, atDesc, strings.Join(notifyDesc, "+"), filterDesc)
	return reply(b, ev, msg)
}

// ─── enable / disable 命令（群命令开关） ──────────────────────────────────────

type EnableCommand struct{}

func (c *EnableCommand) Name() string { return "enable" }
func (c *EnableCommand) Help() string {
	return "启用指定命令（仅管理员可用）\n用法：!enable <命令名>"
}
func (c *EnableCommand) Execute(b *onebot.Bot, ev *onebot.Event, args []string) error {
	return setCommandSwitch(b, ev, args, true)
}

type DisableCommand struct{}

func (c *DisableCommand) Name() string { return "disable" }
func (c *DisableCommand) Help() string {
	return "禁用指定命令（仅管理员可用）\n用法：!disable <命令名>"
}
func (c *DisableCommand) Execute(b *onebot.Bot, ev *onebot.Event, args []string) error {
	return setCommandSwitch(b, ev, args, false)
}

func setCommandSwitch(b *onebot.Bot, ev *onebot.Event, args []string, enabled bool) error {
	if !isGroupEvent(ev) {
		return reply(b, ev, "❌ 该命令只能在群内使用")
	}
	ok, err := checkAdmin(b, ev)
	if !ok {
		return err
	}
	if len(args) < 1 {
		return reply(b, ev, "❌ 用法：!enable <命令名> 或 !disable <命令名>")
	}
	cmdName := strings.ToLower(args[0])
	// 防止禁用关键管理命令
	protected := map[string]bool{"enable": true, "disable": true, "addadmin": true, "removeadmin": true}
	if protected[cmdName] {
		return reply(b, ev, "❌ 该命令不允许被禁用")
	}
	if err := db.SetCommandEnabled(ev.GroupID, cmdName, enabled); err != nil {
		return fmt.Errorf("设置命令开关失败: %w", err)
	}
	action := "已禁用"
	if enabled {
		action = "已启用"
	}
	logger.Infof("command switch: group=%d cmd=%s enabled=%v (by %d)", ev.GroupID, cmdName, enabled, ev.UserID)
	return reply(b, ev, fmt.Sprintf("✅ 命令 [%s] %s", cmdName, action))
}

// ─── block / unblock 命令（黑名单） ──────────────────────────────────────────

type BlockCommand struct{}

func (c *BlockCommand) Name() string { return "block" }
func (c *BlockCommand) Help() string {
	return `封禁用户或群（仅管理员可用）
用法：
  !block user <QQ号> [原因]   封禁用户
  !block group <群号> [原因]  封禁群
  !block list [user|group]    查看黑名单
  !block remove user <QQ号>   解除封禁
  !block remove group <群号>  解除封禁`
}
func (c *BlockCommand) Execute(b *onebot.Bot, ev *onebot.Event, args []string) error {
	ok, err := checkAdmin(b, ev)
	if !ok {
		return err
	}
	if len(args) < 1 {
		return reply(b, ev, "❌ 参数不足\n"+c.Help())
	}

	sub := strings.ToLower(args[0])
	switch sub {
	case "list":
		filterType := ""
		if len(args) >= 2 {
			filterType = strings.ToLower(args[1])
		}
		blocks, err := db.ListBlocks(filterType)
		if err != nil {
			return fmt.Errorf("查询黑名单失败: %w", err)
		}
		if len(blocks) == 0 {
			return reply(b, ev, "📋 黑名单为空")
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("📋 黑名单（共 %d 条）\n", len(blocks)))
		for i, e := range blocks {
			reason := ""
			if e.Reason != "" {
				reason = " 原因：" + e.Reason
			}
			sb.WriteString(fmt.Sprintf("%d. [%s] %d%s\n", i+1, e.Type, e.ID, reason))
		}
		return reply(b, ev, strings.TrimRight(sb.String(), "\n"))

	case "remove":
		if len(args) < 3 {
			return reply(b, ev, "❌ 用法：!block remove user|group <ID>")
		}
		targetType, id, parseErr := parseBlockTarget(args[1], args[2])
		if parseErr != nil {
			return reply(b, ev, "❌ "+parseErr.Error())
		}
		if err := db.RemoveBlock(targetType, id); err != nil {
			return fmt.Errorf("移除黑名单失败: %w", err)
		}
		logger.Infof("unblock: type=%s id=%d (by %d)", targetType, id, ev.UserID)
		return reply(b, ev, fmt.Sprintf("✅ 已移除黑名单 [%s] %d", targetType, id))

	case "user", "group":
		if len(args) < 2 {
			return reply(b, ev, fmt.Sprintf("❌ 用法：!block %s <ID> [原因]", sub))
		}
		targetType, id, parseErr := parseBlockTarget(sub, args[1])
		if parseErr != nil {
			return reply(b, ev, "❌ "+parseErr.Error())
		}
		reason := ""
		if len(args) >= 3 {
			reason = strings.Join(args[2:], " ")
		}
		if err := db.AddBlock(targetType, id, reason); err != nil {
			return fmt.Errorf("添加黑名单失败: %w", err)
		}
		logger.Infof("block: type=%s id=%d reason=%s (by %d)", targetType, id, reason, ev.UserID)
		return reply(b, ev, fmt.Sprintf("✅ 已封禁 [%s] %d", targetType, id))

	default:
		return reply(b, ev, "❌ 未知操作："+sub+"\n"+c.Help())
	}
}

func parseBlockTarget(typeStr, idStr string) (string, int64, error) {
	typeStr = strings.ToLower(typeStr)
	if typeStr != "user" && typeStr != "group" {
		return "", 0, fmt.Errorf("类型必须是 user 或 group")
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("ID 格式错误：%s", idStr)
	}
	return typeStr, id, nil
}

// ─── 关键词过滤工具函数 ────────────────────────────────────────────────────────

func addKeyword(filterJSON, kw string) string {
	kws := parseKeywords(filterJSON)
	for _, k := range kws {
		if k == kw {
			return filterJSON // 已存在
		}
	}
	kws = append(kws, kw)
	return marshalKeywords(kws)
}

func removeKeyword(filterJSON, kw string) string {
	kws := parseKeywords(filterJSON)
	var result []string
	for _, k := range kws {
		if k != kw {
			result = append(result, k)
		}
	}
	return marshalKeywords(result)
}

func parseKeywords(filterJSON string) []string {
	if filterJSON == "" || filterJSON == "[]" {
		return nil
	}
	// 简单解析 JSON 数组 ["kw1","kw2"]
	filterJSON = strings.TrimSpace(filterJSON)
	filterJSON = strings.TrimPrefix(filterJSON, "[")
	filterJSON = strings.TrimSuffix(filterJSON, "]")
	parts := strings.Split(filterJSON, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.Trim(p, `"`)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func marshalKeywords(kws []string) string {
	if len(kws) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("[")
	for i, k := range kws {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(`"`)
		sb.WriteString(strings.ReplaceAll(k, `"`, `\"`))
		sb.WriteString(`"`)
	}
	sb.WriteString("]")
	return sb.String()
}

// ─── grant 命令（群内授权管理员） ─────────────────────────────────────────────

// GrantCommand 在群内临时授权某用户作为管理员（BOT全局管理员级别）
// 用法：!grant <QQ号> [add|remove]
type GrantCommand struct {
	superAdmins map[int64]bool
}

func NewGrantCommand(superAdmins []int64) *GrantCommand {
	m := make(map[int64]bool)
	for _, id := range superAdmins {
		m[id] = true
	}
	return &GrantCommand{superAdmins: m}
}

func (c *GrantCommand) Name() string { return "grant" }
func (c *GrantCommand) Help() string {
	return `授权/撤销管理员（仅超级管理员可用）
用法：
  !grant <QQ号>         授予管理员权限
  !grant <QQ号> remove  撤销管理员权限`
}

func (c *GrantCommand) Execute(b *onebot.Bot, ev *onebot.Event, args []string) error {
	// 需要超级管理员或已有管理员
	isSuperAdmin := c.superAdmins[ev.UserID]
	if !isSuperAdmin {
		ok, err := db.IsAdmin(ev.UserID)
		if err != nil || !ok {
			return reply(b, ev, "❌ 权限不足，该命令仅限超级管理员使用")
		}
	}
	if len(args) < 1 {
		return reply(b, ev, "❌ 用法：!grant <QQ号> [remove]")
	}
	targetID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return reply(b, ev, "❌ QQ 号格式错误")
	}

	remove := len(args) >= 2 && strings.ToLower(args[1]) == "remove"
	if remove {
		if err := db.RemoveAdmin(targetID); err != nil {
			return fmt.Errorf("撤销管理员失败: %w", err)
		}
		logger.Infof("grant remove: %d (by %d)", targetID, ev.UserID)
		return reply(b, ev, fmt.Sprintf("✅ 已撤销 %d 的管理员权限", targetID))
	}
	if err := db.AddAdmin(targetID, fmt.Sprintf("granted by %d", ev.UserID)); err != nil {
		return fmt.Errorf("授权管理员失败: %w", err)
	}
	logger.Infof("grant add: %d (by %d)", targetID, ev.UserID)
	return reply(b, ev, fmt.Sprintf("✅ 已授予 %d 管理员权限", targetID))
}
