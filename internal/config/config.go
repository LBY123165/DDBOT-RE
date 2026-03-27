package config

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/viper"
)

// C 全局配置实例（用 viper 读取 application.yaml）
var C = viper.New()

// OneBotConfig OneBot 连接配置
type OneBotConfig struct {
	// WsListen 反向 WS 监听地址（推荐），如 "0.0.0.0:8080"
	// 非空时以反向 WS 模式运行，等待 OneBot 实现主动连入
	WsListen string
	// WsURL 正向 WS 目标地址（WsListen 为空时生效），如 "ws://127.0.0.1:3001"
	WsURL string
	// AccessToken 鉴权 token（与 NapCat/LLOneBot 端 access_token 保持一致，留空不鉴权）
	AccessToken string
}

// GetOneBotConfig 读取 OneBot 配置
func GetOneBotConfig() OneBotConfig {
	return OneBotConfig{
		WsListen:    C.GetString("onebot.ws_listen"),
		WsURL:       C.GetString("onebot.ws_url"),
		AccessToken: C.GetString("onebot.access_token"),
	}
}

// GetSuperAdmins 读取超级管理员 QQ 号列表
// 对应配置：bot.super_admins: [123456, 789012]
func GetSuperAdmins() []int64 {
	raw := C.GetIntSlice("bot.super_admins")
	result := make([]int64, 0, len(raw))
	for _, v := range raw {
		result = append(result, int64(v))
	}
	return result
}

// GetWebUIAddr 读取 WebUI 监听地址
func GetWebUIAddr() string {
	addr := C.GetString("webui.addr")
	if addr == "" {
		return "0.0.0.0:3000"
	}
	return addr
}

// TelegramConfig Telegram Bot 配置
type TelegramConfig struct {
	BotToken string
	Enabled  bool
	Proxy    string
}

// GetTelegramConfig 读取 Telegram 配置
func GetTelegramConfig() TelegramConfig {
	token := C.GetString("telegram.bot_token")
	return TelegramConfig{
		BotToken: token,
		Enabled:  C.GetBool("telegram.enabled") && token != "",
		Proxy:    C.GetString("telegram.proxy"),
	}
}

// Load 加载配置文件（application.yaml）
func Load() error {
	if err := ensureConfigFile(); err != nil {
		return err
	}

	C.SetConfigName("application")
	C.SetConfigType("yaml")
	C.AddConfigPath(".")
	C.AddConfigPath("./configs")

	if err := C.ReadInConfig(); err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 监听配置变化（热重载）
	C.WatchConfig()
	return nil
}

func ensureConfigFile() error {
	fi, err := os.Stat("application.yaml")
	if os.IsNotExist(err) {
		fmt.Println("提示：未检测到 application.yaml，正在生成默认配置...")
		content := defaultConfig
		if runtime.GOOS == "windows" {
			content = strings.ReplaceAll(content, "\n", "\r\n")
		}
		if err2 := os.WriteFile("application.yaml", []byte(content), 0644); err2 != nil {
			return fmt.Errorf("生成 application.yaml 失败: %w", err2)
		}
		fmt.Println("已生成默认 application.yaml，请按需修改后重启")
		return nil
	}
	if err != nil {
		return fmt.Errorf("检查 application.yaml 失败: %w", err)
	}
	if fi.IsDir() {
		return fmt.Errorf("application.yaml 是一个目录，请删除后重试")
	}
	return nil
}

// defaultConfig 默认配置模板
var defaultConfig = `# DDBOT-WSa 配置文件
# OneBot v11 协议连接
# 兼容 NapCat / LLOneBot / go-cqhttp 等实现
onebot:
  # 【推荐】反向 WS 监听地址：Bot 做服务端，等待 OneBot 实现主动连入
  # 在 NapCat/LLOneBot 中配置"反向 WebSocket"，地址填 ws://<本机IP>:8080
  ws_listen: "0.0.0.0:8080"
  # 【备用】正向 WS 地址（ws_listen 非空时此项无效）
  # ws_url: "ws://127.0.0.1:3001"
  # 鉴权 Token，与 OneBot 端 access_token 保持一致，留空则不鉴权
  access_token: ""

# Bot 全局配置
bot:
  # 超级管理员 QQ 号列表（可使用 !addadmin / !removeadmin 命令管理普通管理员）
  super_admins: []

# B 站相关配置
bilibili:
  SESSDATA: ""
  bili_jct: ""
  interval: "25s"

# 日志级别: debug / info / warn / error
log_level: "info"

# WebUI 监听地址
webui:
  addr: "0.0.0.0:3000"

# Telegram Bot 集成（可选）
# 配置后，订阅推送将同步发送到 Telegram 频道/群组
# 使用 !tg bind <chat_id> <平台> <UID> 绑定频道
telegram:
  enabled: false
  bot_token: ""   # 从 @BotFather 获取
  proxy: ""       # 可选 HTTP 代理，如 "http://127.0.0.1:7890"
`
