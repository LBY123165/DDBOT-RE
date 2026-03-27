package command

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/cnxysoft/DDBOT-WSa/internal/db"
	"github.com/cnxysoft/DDBOT-WSa/internal/logger"
	"github.com/cnxysoft/DDBOT-WSa/internal/onebot"
)

// ─── roll 命令 ────────────────────────────────────────────────────────────────

// RollCommand roll 随机数/抽签命令
// 用法：
//   !roll           → 0-100 随机数
//   !roll 50        → 0-50 随机数
//   !roll 10-50     → 10-50 随机数
//   !roll A B C     → 从 A/B/C 中随机选一个
type RollCommand struct{}

func (c *RollCommand) Name() string { return "roll" }
func (c *RollCommand) Help() string {
	return `随机数/抽签
用法：
  !roll           随机 0-100
  !roll 50        随机 0-50
  !roll 10-50     随机 10-50
  !roll A B C     从选项中随机选一个`
}

func (c *RollCommand) Execute(b *onebot.Bot, ev *onebot.Event, args []string) error {
	// 多选项模式：!roll A B C
	if len(args) > 1 {
		result := args[rand.Intn(len(args))]
		return reply(b, ev, result)
	}

	// 数字范围模式
	var min, max int64 = 0, 100
	if len(args) == 1 {
		arg := args[0]
		if strings.Contains(arg, "-") {
			parts := strings.SplitN(arg, "-", 2)
			var err error
			min, err = strconv.ParseInt(parts[0], 10, 64)
			if err != nil {
				return reply(b, ev, "❌ 格式错误，例：!roll 10-50")
			}
			max, err = strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return reply(b, ev, "❌ 格式错误，例：!roll 10-50")
			}
		} else {
			var err error
			max, err = strconv.ParseInt(arg, 10, 64)
			if err != nil {
				// 当成单选项处理
				return reply(b, ev, arg)
			}
		}
	}
	if min > max {
		min, max = max, min
	}
	result := rand.Int63n(max-min+1) + min
	return reply(b, ev, fmt.Sprintf("🎲 %d", result))
}

// ─── 签到命令 ─────────────────────────────────────────────────────────────────

// CheckinCommand 每日签到命令（仅群聊可用）
type CheckinCommand struct{}

func (c *CheckinCommand) Name() string { return "签到" }
func (c *CheckinCommand) Help() string {
	return `每日签到，获得 1 积分（每天仅限一次）
用法：!签到`
}

func (c *CheckinCommand) Execute(b *onebot.Bot, ev *onebot.Event, _ []string) error {
	if !ev.IsGroupMessage() {
		return reply(b, ev, "❌ 签到仅在群聊中使用")
	}
	result, err := db.Checkin(ev.GroupID, ev.UserID)
	if err != nil {
		logger.Errorf("签到失败 group=%d user=%d: %v", ev.GroupID, ev.UserID, err)
		return reply(b, ev, "❌ 签到失败，请稍后重试")
	}
	if !result.Success {
		return reply(b, ev, fmt.Sprintf("📅 今日已签到，当前积分：%d", result.Score))
	}
	return reply(b, ev, fmt.Sprintf("✅ 签到成功！当前积分：%d（+1）", result.Score))
}

// ─── 查询积分命令 ──────────────────────────────────────────────────────────────

// ScoreCommand 查询积分命令（仅群聊可用）
type ScoreCommand struct{}

func (c *ScoreCommand) Name() string { return "积分" }
func (c *ScoreCommand) Help() string {
	return `查询积分（仅群聊）
用法：
  !积分           查看自己的积分
  !积分 排行      查看群积分排行榜（前10名）`
}

func (c *ScoreCommand) Execute(b *onebot.Bot, ev *onebot.Event, args []string) error {
	if !ev.IsGroupMessage() {
		return reply(b, ev, "❌ 积分查询仅在群聊中使用")
	}

	// 排行榜
	if len(args) > 0 && (args[0] == "排行" || args[0] == "rank") {
		ranking, err := db.ScoreRanking(ev.GroupID, 10)
		if err != nil {
			return reply(b, ev, "❌ 查询排行失败")
		}
		if len(ranking) == 0 {
			return reply(b, ev, "暂无积分记录")
		}
		var sb strings.Builder
		sb.WriteString("🏆 积分排行榜 Top10\n")
		for i, r := range ranking {
			sb.WriteString(fmt.Sprintf("%d. QQ %d — %d 分\n", i+1, r.UserID, r.Score))
		}
		return reply(b, ev, strings.TrimRight(sb.String(), "\n"))
	}

	score, err := db.GetScore(ev.GroupID, ev.UserID)
	if err != nil {
		return reply(b, ev, "❌ 查询失败")
	}
	return reply(b, ev, fmt.Sprintf("💰 当前积分：%d", score))
}

// ─── silence 命令 ─────────────────────────────────────────────────────────────

// SilenceCommand 群静音命令（BOT 在群内静默，不回复非管理员消息）
// 需要管理员权限
type SilenceCommand struct{}

func (c *SilenceCommand) Name() string { return "silence" }
func (c *SilenceCommand) Help() string {
	return `群静音/恢复（管理员命令）
BOT 在静音状态下不会回复任何命令（管理员命令除外）
用法：
  !silence        开启静音模式
  !silence off    关闭静音模式
  !silence status 查看当前状态`
}

func (c *SilenceCommand) Execute(b *onebot.Bot, ev *onebot.Event, args []string) error {
	if !ev.IsGroupMessage() {
		return reply(b, ev, "❌ silence 命令仅在群聊中使用")
	}

	// 权限检查：需要管理员
	isAdmin, _ := db.IsAdmin(ev.UserID)
	if !isAdmin {
		return nil // 静默拒绝
	}

	groupCode := ev.GroupID

	if len(args) > 0 {
		switch strings.ToLower(args[0]) {
		case "off", "关闭", "取消":
			if err := db.SetGroupSilence(groupCode, false); err != nil {
				return fmt.Errorf("解除静音失败: %w", err)
			}
			logger.Infof("silence: group %d 解除静音 by %d", groupCode, ev.UserID)
			return reply(b, ev, "✅ 已解除静音模式，BOT 恢复正常响应")
		case "status", "状态":
			silenced, _ := db.IsGroupSilenced(groupCode)
			if silenced {
				return reply(b, ev, "🔇 当前状态：静音中")
			}
			return reply(b, ev, "🔊 当前状态：正常")
		}
	}

	// 开启静音
	if err := db.SetGroupSilence(groupCode, true); err != nil {
		return fmt.Errorf("开启静音失败: %w", err)
	}
	logger.Infof("silence: group %d 开启静音 by %d", groupCode, ev.UserID)
	return reply(b, ev, "🔇 静音模式已开启，BOT 将不回复普通命令\n使用 !silence off 关闭")
}

// ─── 检测异常订阅命令 ──────────────────────────────────────────────────────────

// CheckConcernsCommand 检测并清理异常订阅（无效/残留关注）
type CheckConcernsCommand struct{}

func (c *CheckConcernsCommand) Name() string { return "清除订阅" }
func (c *CheckConcernsCommand) Help() string {
	return `检测和清理异常订阅（管理员命令）
用法：
  !清除订阅 list        列出当前群所有订阅
  !清除订阅 <平台> <UID>  删除指定订阅`
}

func (c *CheckConcernsCommand) Execute(b *onebot.Bot, ev *onebot.Event, args []string) error {
	if !ev.IsGroupMessage() {
		return reply(b, ev, "❌ 该命令仅在群聊中使用")
	}

	// 权限检查
	isAdmin, _ := db.IsAdmin(ev.UserID)
	if !isAdmin {
		return reply(b, ev, "❌ 权限不足，该命令仅限管理员使用")
	}

	if len(args) == 0 || args[0] == "list" {
		return c.listConcerns(b, ev)
	}

	if len(args) >= 2 {
		site := strings.ToLower(args[0])
		uid := args[1]
		if err := db.DeleteConcern(site, uid, ev.GroupID); err != nil {
			return fmt.Errorf("删除订阅失败: %w", err)
		}
		logger.Infof("清除订阅: group=%d site=%s uid=%s by %d", ev.GroupID, site, uid, ev.UserID)
		return reply(b, ev, fmt.Sprintf("✅ 已删除 %s/%s 的订阅", site, uid))
	}

	return reply(b, ev, "❌ 用法：!清除订阅 <平台> <UID>")
}

func (c *CheckConcernsCommand) listConcerns(b *onebot.Bot, ev *onebot.Event) error {
	concerns, err := db.GetConcernsByGroup(ev.GroupID)
	if err != nil {
		return reply(b, ev, "❌ 获取订阅列表失败")
	}
	if len(concerns) == 0 {
		return reply(b, ev, "当前群无任何订阅")
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📋 本群订阅列表（共 %d 条）\n", len(concerns)))
	for _, c := range concerns {
		name := c.Name
		if utf8.RuneCountInString(name) > 10 {
			name = string([]rune(name)[:10]) + "…"
		}
		sb.WriteString(fmt.Sprintf("  [%s] %s (%s) — %s\n",
			c.Site, name, c.UID, c.ConcernType))
	}
	return reply(b, ev, strings.TrimRight(sb.String(), "\n"))
}

// ─── 初始化 rand seed ─────────────────────────────────────────────────────────

func init() {
	rand.Seed(time.Now().UnixNano()) //nolint:staticcheck
}

// ─── 倒放命令 ─────────────────────────────────────────────────────────────────
// 对群内最近一条视频消息进行倒放，需要本地安装 ffmpeg。
// 用法：回复视频消息后发送 !倒放
// 工作流程：
//   1. 从被回复消息中提取视频 URL（CQ:video 或 CQ:file）
//   2. 用 ffmpeg 下载并倒放视频（vreverse+areverse）
//   3. 将倒放后的视频发回群内

type ReverseCommand struct{}

func (c *ReverseCommand) Name() string { return "倒放" }
func (c *ReverseCommand) Help() string {
	return `倒放视频消息（需要本地安装 ffmpeg）
用法：回复一条视频消息，再发送 !倒放`
}

func (c *ReverseCommand) Execute(b *onebot.Bot, ev *onebot.Event, _ []string) error {
	if !ev.IsGroupMessage() {
		return reply(b, ev, "❌ 倒放命令仅在群聊中使用")
	}

	// 检查 ffmpeg 是否可用
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return reply(b, ev, "❌ 服务器未安装 ffmpeg，倒放功能不可用\n请在服务器上安装 ffmpeg 后重试")
	}

	// 从被回复消息中提取视频 URL
	videoURL := extractVideoURL(ev)
	if videoURL == "" {
		return reply(b, ev, "❌ 请回复一条视频消息后使用 !倒放")
	}

	_ = reply(b, ev, "⏳ 正在处理视频，请稍候...")

	// 在后台处理，避免阻塞（视频处理可能耗时较长）
	go func() {
		result, err := reverseVideo(videoURL)
		if err != nil {
			logger.Errorf("[倒放] 处理失败: %v", err)
			_ = b.SendGroupText(ev.GroupID, "❌ 倒放处理失败："+err.Error())
			return
		}
		defer os.Remove(result) // 发送后清理临时文件

		// 发送倒放后的视频（使用 file:// 本地路径）
		absPath, _ := filepath.Abs(result)
		if err := b.SendGroupMsg(ev.GroupID, onebot.Video("file://"+absPath)); err != nil {
			logger.Errorf("[倒放] 发送视频失败: %v", err)
			_ = b.SendGroupText(ev.GroupID, "❌ 视频发送失败，请检查 OneBot 实现是否支持发送本地视频")
		}
	}()
	return nil
}

// extractVideoURL 从事件消息段中提取视频/文件 URL
func extractVideoURL(ev *onebot.Event) string {
	// 使用 bot.go 预解析的 Segments
	for _, seg := range ev.Segments {
		if seg.Type == "video" {
			if url, ok := seg.Data["url"]; ok && url != "" {
				return url
			}
			if file, ok := seg.Data["file"]; ok && file != "" {
				return file
			}
		}
	}
	return ""
}

// reverseVideo 使用 ffmpeg 对远程/本地视频进行倒放
// 返回临时输出文件路径
func reverseVideo(videoURL string) (string, error) {
	tmpDir := os.TempDir()
	inputFile := filepath.Join(tmpDir, fmt.Sprintf("ddbot_rev_in_%d.mp4", time.Now().UnixNano()))
	outputFile := filepath.Join(tmpDir, fmt.Sprintf("ddbot_rev_out_%d.mp4", time.Now().UnixNano()))

	// Step 1: 下载视频（直接让 ffmpeg 从 URL 读取）
	// ffmpeg -i <url> -vf reverse -af areverse -c:v libx264 -c:a aac <output>
	// 注意：reverse 滤镜需要将整个视频加载到内存，大视频可能 OOM；
	// 添加 -t 60 限制最多处理 60 秒
	args := []string{
		"-y",           // 覆盖输出
		"-i", videoURL, // 输入（支持 HTTP URL 和本地路径）
		"-t", "60", // 最多处理 60 秒，防止大视频 OOM
		"-vf", "reverse",
		"-af", "areverse",
		"-c:v", "libx264",
		"-preset", "fast",
		"-c:a", "aac",
		"-movflags", "+faststart",
		outputFile,
	}

	cmd := exec.Command("ffmpeg", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		// 清理可能存在的临时文件
		os.Remove(inputFile)
		os.Remove(outputFile)
		return "", fmt.Errorf("ffmpeg 执行失败: %v\n输出：%s", err, string(out))
	}

	return outputFile, nil
}
