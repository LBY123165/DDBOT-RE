package module

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cnxysoft/DDBOT-WSa/internal/db"
	"github.com/cnxysoft/DDBOT-WSa/internal/logger"
	"github.com/cnxysoft/DDBOT-WSa/internal/onebot"
	"github.com/cnxysoft/DDBOT-WSa/internal/telegram"
)

// NotifyType 推送类型枚举
type NotifyType int

const (
	NotifyTypeLive NotifyType = iota // 直播开播
	NotifyTypeNews                   // 动态/视频/推文
)

// SendNotify 统一推送入口，自动应用 concern config：
//   - NotifyLive/NotifyNews 开关：关闭时静默跳过
//   - 关键词过滤：filterText 有内容时，text 必须包含至少一个关键词才推送
//   - At 模式：AtMode=1 在消息前加 [CQ:at,qq=all]，AtMode=2 加指定成员 @
//
// bot     : OneBot 机器人实例
// site    : 平台名称（用于查询配置）
// uid     : 平台用户 ID
// groupCode: 目标群号
// nType   : 通知类型（NotifyTypeLive / NotifyTypeNews）
// text    : 要发送的文字内容
// imageURL: 可选封面图 URL（空字符串表示纯文本）
func SendNotify(bot *onebot.Bot, site, uid string, groupCode int64,
	nType NotifyType, text string, imageURL string) {

	if bot == nil {
		return
	}

	// ── 1. 读取推送配置 ────────────────────────────────────────────────────────
	cfg, err := db.GetConcernConfig(groupCode, site, uid)
	if err != nil {
		logger.Warnf("[notify] 读取 concern config 失败 site=%s uid=%s group=%d: %v",
			site, uid, groupCode, err)
		// 读取失败时仍然推送，使用默认行为
		cfg = &db.ConcernConfig{
			GroupCode:  groupCode,
			Site:       site,
			UID:        uid,
			NotifyLive: true,
			NotifyNews: true,
		}
	}

	// ── 2. 通知类型开关 ────────────────────────────────────────────────────────
	switch nType {
	case NotifyTypeLive:
		if !cfg.NotifyLive {
			logger.Debugf("[notify] skip live (notify_live=off) site=%s uid=%s group=%d", site, uid, groupCode)
			return
		}
	case NotifyTypeNews:
		if !cfg.NotifyNews {
			logger.Debugf("[notify] skip news (notify_news=off) site=%s uid=%s group=%d", site, uid, groupCode)
			return
		}
	}

	// ── 3. 关键词过滤 ──────────────────────────────────────────────────────────
	if cfg.FilterText != "" && cfg.FilterText != "[]" {
		keywords := parseFilterKeywords(cfg.FilterText)
		if len(keywords) > 0 {
			matched := false
			for _, kw := range keywords {
				if strings.Contains(text, kw) {
					matched = true
					break
				}
			}
			if !matched {
				logger.Debugf("[notify] filtered by keyword site=%s uid=%s group=%d keywords=%v",
					site, uid, groupCode, keywords)
				return
			}
		}
	}

	// ── 4. 构建消息段 ──────────────────────────────────────────────────────────
	var segs []onebot.Segment

	// At 模式处理
	switch cfg.AtMode {
	case 1:
		// @全体成员
		segs = append(segs, onebot.AtAll())
		segs = append(segs, onebot.Text("\n"))
	case 2:
		// @指定成员
		members := parseAtMembers(cfg.AtMembers)
		for _, memberID := range members {
			segs = append(segs, onebot.At(fmt.Sprintf("%d", memberID)))
			segs = append(segs, onebot.Text(" "))
		}
		if len(members) > 0 {
			segs = append(segs, onebot.Text("\n"))
		}
	}

	// 文本内容
	if imageURL != "" {
		segs = append(segs, onebot.Text(text+"\n"))
		segs = append(segs, onebot.Image(imageURL))
	} else {
		segs = append(segs, onebot.Text(text))
	}

	// ── 5. 发送到 QQ 群 ────────────────────────────────────────────────────────
	if err := bot.SendGroupMsg(groupCode, segs...); err != nil {
		logger.Errorf("[notify] 发送失败 site=%s uid=%s group=%d: %v", site, uid, groupCode, err)
	}

	// ── 6. 同步转发到 Telegram ──────────────────────────────────────────────────
	go telegram.ForwardToTelegram(site, uid, text, imageURL)
}

// parseFilterKeywords 解析 JSON 关键词数组字符串 → []string
// 格式："["kw1","kw2"]" 或 ""
func parseFilterKeywords(filterJSON string) []string {
	filterJSON = strings.TrimSpace(filterJSON)
	if filterJSON == "" || filterJSON == "[]" {
		return nil
	}
	var kws []string
	if err := json.Unmarshal([]byte(filterJSON), &kws); err != nil {
		// 降级：按逗号分割
		parts := strings.Split(strings.Trim(filterJSON, "[]"), ",")
		for _, p := range parts {
			p = strings.Trim(strings.TrimSpace(p), `"`)
			if p != "" {
				kws = append(kws, p)
			}
		}
	}
	return kws
}

// parseAtMembers 解析 JSON int64 数组字符串 → []int64
// 格式："[123456,789012]" 或 ""
func parseAtMembers(membersJSON string) []int64 {
	membersJSON = strings.TrimSpace(membersJSON)
	if membersJSON == "" || membersJSON == "[]" {
		return nil
	}
	var ids []int64
	if err := json.Unmarshal([]byte(membersJSON), &ids); err != nil {
		return nil
	}
	return ids
}
