package command

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cnxysoft/DDBOT-WSa/internal/db"
	"github.com/cnxysoft/DDBOT-WSa/internal/logger"
	"github.com/cnxysoft/DDBOT-WSa/internal/onebot"
)

// ─── 权限检查工具 ─────────────────────────────────────────────────────────────

// checkAdmin 检查事件发送者是否是管理员，若否则回复提示并返回 false
func checkAdmin(b *onebot.Bot, ev *onebot.Event) (bool, error) {
	ok, err := db.IsAdmin(ev.UserID)
	if err != nil {
		logger.Warnf("检查管理员权限失败: %v", err)
		// 数据库错误时放行超级管理员（UserID==0 不可能触发，仅做保险）
		return false, err
	}
	if !ok {
		return false, reply(b, ev, "❌ 权限不足，该命令仅限管理员使用")
	}
	return true, nil
}

// ─── addadmin ────────────────────────────────────────────────────────────────

type AddAdminCommand struct {
	// superAdmins 超级管理员 QQ 号（只有超管才能添加管理员）
	superAdmins map[int64]bool
}

func NewAddAdminCommand(superAdmins []int64) *AddAdminCommand {
	m := make(map[int64]bool)
	for _, id := range superAdmins {
		m[id] = true
	}
	return &AddAdminCommand{superAdmins: m}
}

func (c *AddAdminCommand) Name() string { return "addadmin" }
func (c *AddAdminCommand) Help() string { return "添加管理员（仅超级管理员可用）" }
func (c *AddAdminCommand) Execute(b *onebot.Bot, ev *onebot.Event, args []string) error {
	// 超级管理员检查
	if !c.superAdmins[ev.UserID] {
		// 降级：若没有超管配置，任何管理员可以添加
		ok, err := db.IsAdmin(ev.UserID)
		if err != nil || !ok {
			return reply(b, ev, "❌ 权限不足")
		}
	}
	if len(args) < 1 {
		return reply(b, ev, "❌ 用法：!addadmin <QQ号> [备注]")
	}
	targetID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return reply(b, ev, "❌ QQ 号格式错误")
	}
	note := ""
	if len(args) >= 2 {
		note = strings.Join(args[1:], " ")
	}
	if err := db.AddAdmin(targetID, note); err != nil {
		return fmt.Errorf("添加管理员失败: %w", err)
	}
	logger.Infof("添加管理员: %d (by %d)", targetID, ev.UserID)
	return reply(b, ev, fmt.Sprintf("✅ 已添加管理员 %d", targetID))
}

// ─── removeadmin ─────────────────────────────────────────────────────────────

type RemoveAdminCommand struct {
	superAdmins map[int64]bool
}

func NewRemoveAdminCommand(superAdmins []int64) *RemoveAdminCommand {
	m := make(map[int64]bool)
	for _, id := range superAdmins {
		m[id] = true
	}
	return &RemoveAdminCommand{superAdmins: m}
}

func (c *RemoveAdminCommand) Name() string { return "removeadmin" }
func (c *RemoveAdminCommand) Help() string { return "移除管理员（仅超级管理员可用）" }
func (c *RemoveAdminCommand) Execute(b *onebot.Bot, ev *onebot.Event, args []string) error {
	if !c.superAdmins[ev.UserID] {
		ok, err := db.IsAdmin(ev.UserID)
		if err != nil || !ok {
			return reply(b, ev, "❌ 权限不足")
		}
	}
	if len(args) < 1 {
		return reply(b, ev, "❌ 用法：!removeadmin <QQ号>")
	}
	targetID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return reply(b, ev, "❌ QQ 号格式错误")
	}
	if err := db.RemoveAdmin(targetID); err != nil {
		return fmt.Errorf("移除管理员失败: %w", err)
	}
	logger.Infof("移除管理员: %d (by %d)", targetID, ev.UserID)
	return reply(b, ev, fmt.Sprintf("✅ 已移除管理员 %d", targetID))
}

// ─── listadmins ──────────────────────────────────────────────────────────────

type ListAdminsCommand struct{}

func (c *ListAdminsCommand) Name() string { return "listadmins" }
func (c *ListAdminsCommand) Help() string { return "查看管理员列表（仅管理员可用）" }
func (c *ListAdminsCommand) Execute(b *onebot.Bot, ev *onebot.Event, _ []string) error {
	ok, err := checkAdmin(b, ev)
	if !ok {
		return err
	}
	ids, err := db.ListAdmins()
	if err != nil {
		return fmt.Errorf("查询管理员失败: %w", err)
	}
	if len(ids) == 0 {
		return reply(b, ev, "📋 暂无管理员")
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📋 管理员列表（共 %d 人）\n", len(ids)))
	for i, id := range ids {
		sb.WriteString(fmt.Sprintf("%d. %d\n", i+1, id))
	}
	return reply(b, ev, sb.String())
}
