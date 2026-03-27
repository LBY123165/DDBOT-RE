// Package telegram 提供 Telegram Bot API 集成。
// 功能：将 DDBOT 推送消息同步转发到 Telegram 频道或群组。
//
// 配置（application.yaml）：
//   telegram:
//     bot_token: "123456789:AAAB..."   # Telegram Bot Token（从 @BotFather 获取）
//     enabled: true
//     proxy: ""                        # 可选 HTTP 代理（如 "http://127.0.0.1:7890"）
//
// 频道绑定存储在 DB telegram_channels 表中。
// 命令（私聊或群内，超管）：
//   !tgbind <chat_id> <site> <uid>   将指定订阅绑定到 Telegram 频道
//   !tgunbind <chat_id> <site> <uid> 解除绑定
//   !tglist                          列出所有绑定

package telegram

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/cnxysoft/DDBOT-WSa/internal/logger"
	neturl "net/url"
)

// Client Telegram Bot API 客户端（纯 HTTP，无第三方依赖）
type Client struct {
	token      string
	httpClient *http.Client
	mu         sync.RWMutex
	enabled    bool
}

var defaultClient *Client
var once sync.Once

// Init 初始化全局 Telegram 客户端
func Init(token string, proxy string, enabled bool) {
	once.Do(func() {
		transport := &http.Transport{}
		if proxy != "" {
			proxyURL, err := neturl.Parse(proxy)
			if err == nil {
				transport.Proxy = http.ProxyURL(proxyURL)
			}
		}
		defaultClient = &Client{
			token:   token,
			enabled: enabled && token != "",
			httpClient: &http.Client{
				Timeout:   30 * time.Second,
				Transport: transport,
			},
		}
		if defaultClient.enabled {
			logger.Infof("[telegram] 客户端已初始化，Token: %s...", token[:min(10, len(token))])
		}
	})
}

// GetClient 获取全局客户端（Init 后调用）
func GetClient() *Client {
	return defaultClient
}

// IsEnabled 检查是否启用
func (c *Client) IsEnabled() bool {
	if c == nil {
		return false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.enabled
}

// SendMessage 向指定 chat_id 发送文本消息
// chatID 可以是 @频道名 或 数字群组 ID
func (c *Client) SendMessage(chatID, text string) error {
	if !c.IsEnabled() {
		return nil
	}
	return c.callAPI("sendMessage", map[string]string{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "HTML",
	})
}

// SendPhoto 向指定 chat_id 发送图片+说明文字
func (c *Client) SendPhoto(chatID, photoURL, caption string) error {
	if !c.IsEnabled() {
		return nil
	}
	return c.callAPI("sendPhoto", map[string]string{
		"chat_id":    chatID,
		"photo":      photoURL,
		"caption":    caption,
		"parse_mode": "HTML",
	})
}

// callAPI 调用 Telegram Bot API
func (c *Client) callAPI(method string, params map[string]string) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/%s", c.token, method)

	form := url.Values{}
	for k, v := range params {
		form.Set(k, v)
	}

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Telegram API 请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode != 200 {
		return fmt.Errorf("Telegram API 返回 %d: %s", resp.StatusCode, string(body))
	}

	// 简单检查 ok 字段
	if !bytes.Contains(body, []byte(`"ok":true`)) {
		return fmt.Errorf("Telegram API 返回错误: %s", string(body))
	}
	return nil
}

// SendNotify 统一推送接口（供 notify_helper 调用）
// 自动判断是否有图片，选择 sendMessage 或 sendPhoto
func (c *Client) SendNotify(chatID, text, imageURL string) {
	if !c.IsEnabled() || chatID == "" {
		return
	}
	var err error
	if imageURL != "" {
		err = c.SendPhoto(chatID, imageURL, htmlEscape(text))
	} else {
		err = c.SendMessage(chatID, htmlEscape(text))
	}
	if err != nil {
		logger.Errorf("[telegram] 推送失败 chat=%s: %v", chatID, err)
	}
}

// htmlEscape 将文本中的 HTML 特殊字符转义（parse_mode=HTML 时需要）
func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
