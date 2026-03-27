package command

import (
	"fmt"
	"strings"

	"github.com/cnxysoft/DDBOT-WSa/internal/cron"
	"github.com/cnxysoft/DDBOT-WSa/internal/db"
	"github.com/cnxysoft/DDBOT-WSa/internal/onebot"
)

// ─── cron 命令 ───────────────────────────────────────────────────────────────
// 用法：
//   !cron add <名称> <cron表达式> <消息模板>   添加定时任务
//   !cron del <名称>                           删除定时任务
//   !cron list                                 列出所有定时任务
//   !cron enable <名称>                        启用任务
//   !cron disable <名称>                       禁用任务
//
// cron 表达式（5段）：分 时 日 月 周
//   示例："0 9 * * 1"   = 每周一 09:00
//         "30 12 * * *"  = 每天 12:30
//         "0 8,20 * * *" = 每天 8:00 和 20:00
//
// 消息模板（支持 Go text/template 变量）：
//   {{.Date}}     当前日期（2006-01-02）
//   {{.Time}}     当前时间（15:04）
//   {{.WeekdayCN}} 星期（中文，如"星期一"）
//   {{.Year}} {{.Month}} {{.Day}} {{.Hour}} {{.Minute}}

type CronCommand struct{}

func (c *CronCommand) Name() string { return "cron" }
func (c *CronCommand) Help() string {
	return `定时任务管理
用法：
  !cron add <名称> <cron表达式> <消息>   添加定时任务
  !cron del <名称>                       删除定时任务
  !cron list                             列出所有任务
  !cron enable/disable <名称>           启用/禁用任务

cron表达式（分 时 日 月 周），例：
  "0 9 * * 1"   每周一09:00
  "30 12 * * *"  每天12:30
  "0 8,20 * * *" 每天8:00和20:00

消息支持模板变量：{{.Date}} {{.Time}} {{.WeekdayCN}}`
}

func (c *CronCommand) Execute(b *onebot.Bot, ev *onebot.Event, args []string) error {
	if !isGroupEvent(ev) {
		return reply(b, ev, "❌ cron 命令只能在群内使用")
	}
	if len(args) == 0 {
		return reply(b, ev, c.Help())
	}

	sub := strings.ToLower(args[0])
	switch sub {
	case "add":
		return c.cmdAdd(b, ev, args[1:])
	case "del", "delete", "remove":
		return c.cmdDel(b, ev, args[1:])
	case "list", "ls":
		return c.cmdList(b, ev)
	case "enable":
		return c.cmdSetEnabled(b, ev, args[1:], true)
	case "disable":
		return c.cmdSetEnabled(b, ev, args[1:], false)
	default:
		return reply(b, ev, "❌ 未知子命令。用法：add/del/list/enable/disable")
	}
}

// cmdAdd 添加定时任务
// 格式：!cron add <名称> <分> <时> <日> <月> <周> <消息...>
// 或：  !cron add <名称> "<cron表达式>" <消息...>
func (c *CronCommand) cmdAdd(b *onebot.Bot, ev *onebot.Event, args []string) error {
	if len(args) < 3 {
		return reply(b, ev, `❌ 用法：!cron add <名称> <cron表达式> <消息>
例：!cron add 早报 "0 9 * * *" 早上好！今天是{{.Date}} {{.WeekdayCN}}`)
	}

	name := args[0]
	// 尝试两种格式：
	// 格式A：!cron add 名称 "0 9 * * *" 消息内容
	// 格式B：!cron add 名称 0 9 * * * 消息内容（6段以上）
	var cronExpr, tmpl string

	if len(args) >= 7 {
		// 格式B：第2-6个参数是 cron 5段
		cronExpr = strings.Join(args[1:6], " ")
		tmpl = strings.Join(args[6:], " ")
	} else {
		// 格式A：第2个参数是带引号的 cron，其余是消息
		cronExpr = args[1]
		tmpl = strings.Join(args[2:], " ")
	}

	if tmpl == "" {
		return reply(b, ev, "❌ 消息内容不能为空")
	}

	if err := cron.ValidateCronExpr(cronExpr); err != nil {
		return reply(b, ev, fmt.Sprintf("❌ cron 表达式格式错误：%v\n正确格式：分 时 日 月 周（5段）\n例：\"0 9 * * 1\" = 每周一9:00", err))
	}

	if err := db.InsertCronJob(ev.GroupID, name, cronExpr, tmpl); err != nil {
		return fmt.Errorf("添加定时任务失败: %w", err)
	}
	return reply(b, ev, fmt.Sprintf("✅ 定时任务 [%s] 已添加\n⏰ 计划：%s\n📝 消息：%s", name, cronExpr, tmpl))
}

// cmdDel 删除定时任务
func (c *CronCommand) cmdDel(b *onebot.Bot, ev *onebot.Event, args []string) error {
	if len(args) == 0 {
		return reply(b, ev, "❌ 用法：!cron del <名称>")
	}
	name := args[0]
	if err := db.DeleteCronJob(ev.GroupID, name); err != nil {
		if strings.Contains(err.Error(), "不存在") {
			return reply(b, ev, fmt.Sprintf("⚠️ 未找到定时任务：%s", name))
		}
		return fmt.Errorf("删除定时任务失败: %w", err)
	}
	return reply(b, ev, fmt.Sprintf("✅ 已删除定时任务：%s", name))
}

// cmdList 列出所有定时任务
func (c *CronCommand) cmdList(b *onebot.Bot, ev *onebot.Event) error {
	jobs, err := db.ListCronJobs(ev.GroupID)
	if err != nil {
		return fmt.Errorf("查询定时任务失败: %w", err)
	}
	if len(jobs) == 0 {
		return reply(b, ev, "📋 暂无定时任务，使用 !cron add 添加")
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📋 定时任务列表（共 %d 条）\n\n", len(jobs)))
	for _, j := range jobs {
		status := "✅"
		if !j.Enabled {
			status = "⏸️"
		}
		lastRun := "从未执行"
		if !j.LastRun.IsZero() {
			lastRun = j.LastRun.Format("01-02 15:04")
		}
		sb.WriteString(fmt.Sprintf("%s [%s]\n   ⏰ %s\n   📝 %s\n   🕐 上次：%s\n\n",
			status, j.Name, j.CronExpr, j.Template, lastRun))
	}
	return reply(b, ev, strings.TrimRight(sb.String(), "\n"))
}

// cmdSetEnabled 启用/禁用定时任务
func (c *CronCommand) cmdSetEnabled(b *onebot.Bot, ev *onebot.Event, args []string, enabled bool) error {
	if len(args) == 0 {
		action := "enable"
		if !enabled {
			action = "disable"
		}
		return reply(b, ev, fmt.Sprintf("❌ 用法：!cron %s <名称>", action))
	}
	name := args[0]
	if err := db.SetCronJobEnabled(ev.GroupID, name, enabled); err != nil {
		if strings.Contains(err.Error(), "不存在") {
			return reply(b, ev, fmt.Sprintf("⚠️ 未找到定时任务：%s", name))
		}
		return fmt.Errorf("操作失败: %w", err)
	}
	action := "已启用"
	if !enabled {
		action = "已禁用"
	}
	return reply(b, ev, fmt.Sprintf("✅ 定时任务 [%s] %s", name, action))
}
