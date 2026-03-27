package telegram

import (
	"fmt"
	"strings"

	"github.com/cnxysoft/DDBOT-WSa/internal/db"
	"github.com/cnxysoft/DDBOT-WSa/internal/logger"
)

// Channel Telegram 频道绑定记录
// 含义：当 site/uid 有新推送时，同时转发到 chat_id
type Channel struct {
	ID        int64
	ChatID    string // Telegram chat_id（@频道名 或 数字）
	Site      string // 平台
	UID       string // 平台用户 ID
	GroupCode int64  // 来源 QQ 群（可选，0 表示全局）
}

// InitChannelSchema 初始化 telegram_channels 表（由 db.InitSchema 调用，需手动添加）
// 这里提供独立的初始化函数，在 app 启动时调用
func InitChannelSchema() error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS telegram_channels (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			chat_id     VARCHAR(100) NOT NULL,
			site        VARCHAR(50)  NOT NULL,
			uid         VARCHAR(100) NOT NULL,
			group_code  INTEGER      DEFAULT 0,
			create_time DATETIME     DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(chat_id, site, uid)
		);
		CREATE INDEX IF NOT EXISTS idx_tg_site_uid ON telegram_channels(site, uid);
	`)
	return err
}

// BindChannel 绑定 Telegram 频道到指定订阅
func BindChannel(chatID, site, uid string, groupCode int64) error {
	_, err := db.Exec(`
		INSERT INTO telegram_channels (chat_id, site, uid, group_code)
		VALUES (?,?,?,?)
		ON CONFLICT(chat_id, site, uid) DO UPDATE SET group_code=excluded.group_code`,
		chatID, site, uid, groupCode)
	return err
}

// UnbindChannel 解除绑定
func UnbindChannel(chatID, site, uid string) error {
	result, err := db.Exec(`
		DELETE FROM telegram_channels WHERE chat_id=? AND site=? AND uid=?`,
		chatID, site, uid)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("绑定不存在: chat=%s site=%s uid=%s", chatID, site, uid)
	}
	return nil
}

// ListChannels 列出所有绑定（site/uid 可为空表示不过滤）
func ListChannels(site, uid string) ([]*Channel, error) {
	var rows interface {
		Next() bool
		Scan(...interface{}) error
		Close() error
	}
	var err error
	if site == "" {
		rows, err = db.Query(`SELECT id, chat_id, site, uid, group_code FROM telegram_channels ORDER BY site, uid`)
	} else if uid == "" {
		rows, err = db.Query(`SELECT id, chat_id, site, uid, group_code FROM telegram_channels WHERE site=? ORDER BY uid`, site)
	} else {
		rows, err = db.Query(`SELECT id, chat_id, site, uid, group_code FROM telegram_channels WHERE site=? AND uid=?`, site, uid)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Channel
	for rows.Next() {
		c := &Channel{}
		if err := rows.Scan(&c.ID, &c.ChatID, &c.Site, &c.UID, &c.GroupCode); err == nil {
			list = append(list, c)
		}
	}
	return list, nil
}

// GetChannelsBySiteUID 获取指定 site/uid 的所有绑定（推送时调用）
func GetChannelsBySiteUID(site, uid string) ([]*Channel, error) {
	return ListChannels(site, uid)
}

// ForwardToTelegram 将推送消息转发到所有绑定的 Telegram 频道
// 由 notify_helper.SendNotify 调用
func ForwardToTelegram(site, uid, text, imageURL string) {
	client := GetClient()
	if client == nil || !client.IsEnabled() {
		return
	}
	channels, err := GetChannelsBySiteUID(site, uid)
	if err != nil {
		logger.Errorf("[telegram] 查询绑定失败 site=%s uid=%s: %v", site, uid, err)
		return
	}
	for _, ch := range channels {
		go client.SendNotify(ch.ChatID, text, imageURL)
	}
}

// FormatChannelList 格式化频道绑定列表供展示
func FormatChannelList(channels []*Channel) string {
	if len(channels) == 0 {
		return "暂无 Telegram 频道绑定"
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📡 Telegram 绑定列表（共 %d 条）\n\n", len(channels)))
	for _, c := range channels {
		sb.WriteString(fmt.Sprintf("• [%s] %s → %s\n", c.Site, c.UID, c.ChatID))
	}
	return strings.TrimRight(sb.String(), "\n")
}
