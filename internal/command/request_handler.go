package command

import (
	"fmt"
	"strings"

	"github.com/cnxysoft/DDBOT-WSa/internal/db"
	"github.com/cnxysoft/DDBOT-WSa/internal/logger"
	"github.com/cnxysoft/DDBOT-WSa/internal/onebot"
)

// RequestHandler 处理好友申请和入群邀请/申请
// 默认策略：
//   - 好友申请：仅当对方是群内管理员时自动同意，其余忽略
//   - 入群申请（他人申请加入）：忽略（不自动处理）
//   - 入群邀请（BOT 被邀请）：自动同意（如配置 auto_accept_invite=true）
type RequestHandler struct {
	superAdmins map[int64]bool
	autoInvite  bool // 是否自动接受群邀请
}

func NewRequestHandler(superAdmins []int64, autoInvite bool) *RequestHandler {
	m := make(map[int64]bool)
	for _, id := range superAdmins {
		m[id] = true
	}
	return &RequestHandler{superAdmins: m, autoInvite: autoInvite}
}

// HandleRequest 处理 request 事件（由 manager 调用）
func (h *RequestHandler) HandleRequest(b *onebot.Bot, ev *onebot.Event) {
	switch ev.RequestType {
	case "friend":
		h.handleFriendRequest(b, ev)
	case "group":
		h.handleGroupRequest(b, ev)
	}
}

func (h *RequestHandler) handleFriendRequest(b *onebot.Bot, ev *onebot.Event) {
	// 如果申请人是超级管理员，自动同意
	if h.superAdmins[ev.UserID] {
		logger.Infof("request: 自动同意超级管理员 %d 的好友申请", ev.UserID)
		_ = b.SetFriendAddRequest(ev.Flag, true, "")
		return
	}
	// 如果是已注册管理员，自动同意
	isAdmin, _ := db.IsAdmin(ev.UserID)
	if isAdmin {
		logger.Infof("request: 自动同意管理员 %d 的好友申请", ev.UserID)
		_ = b.SetFriendAddRequest(ev.Flag, true, "")
		return
	}
	logger.Debugf("request: 忽略 %d 的好友申请（非管理员）", ev.UserID)
}

func (h *RequestHandler) handleGroupRequest(b *onebot.Bot, ev *onebot.Event) {
	switch ev.SubType {
	case "add":
		// 他人申请加入群，一般 BOT 没权限批准，忽略
		logger.Debugf("request: 收到 %d 的入群申请（group=%d），已忽略", ev.UserID, ev.GroupID)
	case "invite":
		// BOT 被邀请入群
		if h.autoInvite || h.superAdmins[ev.UserID] {
			logger.Infof("request: 自动接受入群邀请（group=%d，inviter=%d）", ev.GroupID, ev.UserID)
			_ = b.SetGroupAddRequest(ev.Flag, "invite", true, "")
		} else {
			logger.Infof("request: 拒绝入群邀请（group=%d，inviter=%d）", ev.GroupID, ev.UserID)
			_ = b.SetGroupAddRequest(ev.Flag, "invite", false, "未经授权，请联系超级管理员")
		}
	}
}

// ─── invite 命令（运行时控制入群邀请策略）────────────────────────────────────

// InviteCommand 运行时控制是否自动接受入群邀请
type InviteCommand struct {
	handler *RequestHandler
}

func NewInviteCommand(handler *RequestHandler) *InviteCommand {
	return &InviteCommand{handler: handler}
}

func (c *InviteCommand) Name() string { return "invite" }
func (c *InviteCommand) Help() string {
	return `控制 BOT 入群邀请自动接受策略（私聊管理员命令）
用法：
  !invite on    自动接受所有入群邀请
  !invite off   拒绝非超管发起的入群邀请
  !invite status 查看当前状态`
}

func (c *InviteCommand) Execute(b *onebot.Bot, ev *onebot.Event, args []string) error {
	if !ev.IsPrivateMessage() {
		return nil
	}
	isAdmin, _ := db.IsAdmin(ev.UserID)
	if !isAdmin {
		return nil
	}

	if len(args) == 0 || args[0] == "status" {
		state := "关闭"
		if c.handler.autoInvite {
			state = "开启"
		}
		return reply(b, ev, fmt.Sprintf("入群邀请自动接受：%s", state))
	}

	switch strings.ToLower(args[0]) {
	case "on", "开":
		c.handler.autoInvite = true
		logger.Infof("invite: 自动接受入群邀请已开启 by %d", ev.UserID)
		return reply(b, ev, "✅ 已开启自动接受入群邀请")
	case "off", "关":
		c.handler.autoInvite = false
		logger.Infof("invite: 自动接受入群邀请已关闭 by %d", ev.UserID)
		return reply(b, ev, "✅ 已关闭自动接受入群邀请（仅超管可邀入群）")
	}
	return reply(b, ev, "❌ 用法：!invite on|off|status")
}
