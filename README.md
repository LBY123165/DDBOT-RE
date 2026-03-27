# DDBOT-WSa 重构版

> 基于原版 [DDBOT-WSa](https://github.com/cnxysoft/DDBOT-WSa) 的完整重构，移除 MiraiGo 依赖，采用 OneBot v11 协议 + 纯 Go SQLite，配套内嵌 WebUI 管理面板。

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue)](LICENSE)
[![Build](https://img.shields.io/badge/Build-CGO_ENABLED%3D0-success)](#编译)

---

## 目录

- [与原版对比](#与原版对比)
- [功能概览](#功能概览)
- [支持平台](#支持平台)
- [快速开始](#快速开始)
- [配置说明](#配置说明)
- [命令列表](#命令列表)
- [WebUI](#webui)
- [项目结构](#项目结构)
- [编译](#编译)

---

## 与原版对比

| 项目 | 原版 DDBOT-WSa | 本重构版 |
|------|--------------|---------|
| **Bot 协议** | MiraiGo（需要登录 QQ） | OneBot v11（对接 NapCat/LLOneBot） |
| **协议依赖** | MiraiGo + Mirai 生态 | 无第三方 Bot 框架，纯 WS 通信 |
| **数据库** | buntdb（KV 嵌入式） | SQLite3（纯 Go，CGO_ENABLED=0） |
| **WS 连接方式** | 主动连接 | 反向 WS（Bot 做服务端，等待连入）/ 正向 WS |
| **WebUI** | Electron 桌面版 | 内嵌静态 SPA，浏览器访问（端口 3000） |
| **前端构建** | 需要 Electron 环境 | 只需 `npm run build`，产物嵌入二进制 |
| **编译方式** | 需要 CGO（go-sqlite3） | **纯 Go**，`CGO_ENABLED=0` 跨平台编译 |
| **多平台发布** | 手动 | 一键 `.\build.ps1 -All` 构建 6 平台 |
| **平台模块数** | 7 个（含部分占位） | **9 个**（新增 Twitter / TwitCasting） |
| **Telegram 推送** | ❌ 无 | ✅ 内置，与订阅推送联动 |
| **CronJob** | ❌ 无 | ✅ 内置 5 段 Cron 定时任务 |
| **配置热读** | 需重启 | WebUI 支持在线读写配置 |

---

## 功能概览

### 核心架构

- **OneBot v11 协议**：兼容 [NapCat](https://github.com/NapNeko/NapCatQQ)、[LLOneBot](https://github.com/LLOneBot/LLOneBot)、[go-cqhttp](https://github.com/Mrs4s/go-cqhttp) 等任意实现
- **反向 WS**（默认）：Bot 做 WebSocket 服务端，OneBot 实现主动连入，断线静默等待
- **纯 Go SQLite**：`modernc.org/sqlite` 无需 CGO，Windows/Linux/macOS/ARM 全平台直接编译
- **模块化**：每个平台独立 goroutine，支持 Start / Stop / Reload 生命周期控制
- **内嵌 WebUI**：前端 `dist` 通过 `go:embed` 打包进二进制，双击即用，无需额外部署

### 推送系统

- 统一推送入口 `SendNotify()`，所有模块共享推送逻辑
- 支持每群独立配置：@全体 / @指定成员 / 关键词过滤 / 开关直播/动态分类推送
- **Telegram Bot 同步推送**：绑定 Channel/群组，订阅推送自动同步到 Telegram

---

## 支持平台

| 平台 | 支持内容 | 轮询间隔 | 备注 |
|------|---------|---------|------|
| 🔵 Bilibili | 直播开播 + 动态（含图文推送）+ 番剧追番 + 专栏文章 | 30s / 90s | 直播通知含封面图 |
| 🔴 斗鱼 | 直播开播 | 30s | |
| 🟠 虎牙 | 直播开播 | 30s | 含降级方案 |
| ⚫ 微博 | 动态推送 | 60s | |
| 🟣 AcFun | 视频更新 | 60s | |
| 🔴 YouTube | 视频更新（RSS） | 60s | |
| 🐦 Twitter / X | 推文（Nitter 镜像 + Anubis PoW 反爬） | 60s | 无需 API Key |
| 🎥 TwitCasting | 直播开播（官方公开 API） | 60s | 无需 OAuth |
| 🎵 抖音 | ⚠️ 暂不可用 | — | 接口反爬严格，仅占位 |

---

## 快速开始

### 1. 下载

从 [Releases](../../releases) 下载对应平台的可执行文件：

```
ddbot-windows-amd64.exe   # Windows x64
ddbot-linux-amd64         # Linux x64
ddbot-linux-arm64         # Linux ARM64（树莓派等）
ddbot-darwin-arm64        # macOS Apple Silicon
```

### 2. 配置 OneBot 实现

推荐使用 **NapCat**（QQ 官方客户端注入方案）：

1. 安装并启动 NapCat
2. 在 NapCat 配置中添加 **反向 WebSocket** 连接：
   - 地址：`ws://127.0.0.1:8080`（Bot 和 NapCat 在同一台机器时）

### 3. 配置 Bot

首次运行会自动生成 `application.yaml`，编辑关键配置：

```yaml
onebot:
  ws_listen: "0.0.0.0:8080"   # 反向 WS 监听地址

bot:
  super_admins: [你的QQ号]    # 超级管理员

webui:
  addr: "0.0.0.0:3000"        # WebUI 访问地址
```

### 4. 运行

```bash
# Windows
.\ddbot-windows-amd64.exe

# Linux
chmod +x ddbot-linux-amd64
./ddbot-linux-amd64
```

启动后访问 `http://127.0.0.1:3000` 打开 WebUI 管理面板。

---

## 配置说明

`application.yaml` 完整示例：

```yaml
# OneBot v11 连接（二选一）
onebot:
  # 【推荐】反向 WS：等待 NapCat/LLOneBot 主动连入
  ws_listen: "0.0.0.0:8080"
  # 【备用】正向 WS：Bot 主动连接
  # ws_url: "ws://127.0.0.1:3001"
  access_token: ""    # 与 OneBot 端保持一致，留空不鉴权

bot:
  super_admins: [123456789]   # 超级管理员 QQ 号，可多个

bilibili:
  SESSDATA: ""        # B 站 Cookie（提高请求成功率，可选）
  bili_jct: ""
  interval: "25s"

log_level: "info"     # debug / info / warn / error

webui:
  addr: "0.0.0.0:3000"

# Telegram 推送（可选）
telegram:
  enabled: false
  bot_token: ""       # 从 @BotFather 获取
  proxy: ""           # HTTP 代理，如 http://127.0.0.1:7890
```

---

## 命令列表

命令前缀支持 `!` / `.` / `/`，例如 `!订阅` 或 `.订阅`。

### 订阅管理（管理员）

| 命令 | 说明 |
|------|------|
| `!订阅 <平台> <UID>` | 在当前群订阅指定主播/用户 |
| `!取消订阅 <平台> <UID>` | 取消订阅 |
| `!查看订阅` | 查看当前群的所有订阅 |
| `!清除订阅` | 清除当前群所有订阅 |

**支持的平台名**：`bilibili` / `douyu` / `huya` / `weibo` / `acfun` / `youtube` / `twitter` / `twitcasting`

### 推送配置（管理员）

| 命令 | 说明 |
|------|------|
| `!config at all` | 开播通知@全体成员 |
| `!config at <QQ号>` | 开播通知@指定成员 |
| `!config notify live on/off` | 开关直播通知 |
| `!config notify news on/off` | 开关动态通知 |
| `!config filter add <关键词>` | 添加推送过滤词 |
| `!config filter del <关键词>` | 删除过滤词 |
| `!config filter list` | 查看过滤词列表 |
| `!enable <命令名>` | 启用群内某命令 |
| `!disable <命令名>` | 禁用群内某命令 |

### 管理命令（管理员）

| 命令 | 说明 |
|------|------|
| `!grant <QQ号>` | 授予管理员权限 |
| `!block add <QQ号>` | 加入黑名单 |
| `!block del <QQ号>` | 移出黑名单 |
| `!block list` | 查看黑名单 |
| `!silence on/off` | 开关群静音模式 |
| `!invite on/off` | 允许/拒绝自动接受入群邀请 |

### 定时任务（管理员）

| 命令 | 说明 |
|------|------|
| `!cron add <cron表达式> <消息>` | 添加定时任务（5段 Cron） |
| `!cron list` | 查看当前群定时任务 |
| `!cron del <ID>` | 删除定时任务 |
| `!cron enable/disable <ID>` | 启用/禁用定时任务 |

Cron 消息支持 Go 模板，例如：`{{.Date}}` 展开为当天日期。

### Telegram 绑定（超管私聊）

| 命令 | 说明 |
|------|------|
| `!tg bind <chat_id> <平台> <UID>` | 绑定 Telegram 频道/群组 |
| `!tg unbind <chat_id>` | 解绑 |
| `!tg list` | 查看绑定列表 |
| `!tg test <chat_id>` | 发送测试消息 |
| `!tg status` | 查看 Telegram Bot 状态 |

### 娱乐命令

| 命令 | 说明 |
|------|------|
| `!roll [N]` | 掷骰子（1~N，默认100） |
| `!roll <选项1> <选项2> ...` | 多选项随机抽签 |
| `!签到` | 每日签到，获得积分 |
| `!积分` | 查看个人积分 |
| `!积分榜` | 查看群积分排行榜 |
| `!倒放` | 回复视频消息，调用 ffmpeg 倒放（需安装 ffmpeg） |

### 系统命令（超管私聊）

| 命令 | 说明 |
|------|------|
| `!sysinfo` | 查看系统信息（CPU/内存/运行时间） |
| `!status` | 查看 Bot 连接状态和模块状态 |
| `!log debug/info/warn/error` | 动态调整日志级别 |
| `!quit` | 安全退出 Bot |

---

## WebUI

访问 `http://<Bot_IP>:3000` 打开管理面板。

### 页面功能

| 页面 | 功能 |
|------|------|
| **概览** | Bot 在线状态、订阅统计、运行时间 |
| **订阅管理** | 可视化查看/添加/删除各群订阅 |
| **模块管理** | 各平台模块状态、启停控制 |
| **连接管理** | 切换反向/正向 WS 模式、配置 Telegram，无需手动编辑 yaml |
| **配置编辑** | 在线编辑 `application.yaml`（图形化 + 文本两种模式） |
| **日志查看** | 实时查看 Bot 日志，支持级别过滤 |

### API 端点

| 端点 | 说明 |
|------|------|
| `GET /api/v1/health` | 健康检查 |
| `GET /api/v1/onebot/status` | OneBot 连接状态 |
| `GET /api/v1/modules` | 所有模块状态 |
| `POST /api/v1/modules/{name}/start\|stop\|reload` | 模块生命周期控制 |
| `GET /api/v1/subs` | 查询订阅列表 |
| `POST /api/v1/subs` | 添加订阅 |
| `DELETE /api/v1/subs/{site}/{uid}?group=` | 删除订阅 |
| `GET/POST /api/v1/settings` | 读写连接配置（OneBot + Telegram） |
| `GET/POST /api/v1/config` | 读写完整配置文件 |
| `GET /api/v1/logs` | 获取日志 |

---

## 项目结构

```
DDBOT-RE - claw/
├── cmd/
│   └── main.go              # 程序入口
├── internal/
│   ├── app/                 # 应用核心（启动流程编排）
│   ├── assets/              # 嵌入式静态文件（embed.FS）
│   │   ├── embed.go
│   │   └── dist/            # 前端构建产物（自动同步）
│   ├── command/             # 命令解析与执行（~20 条命令）
│   ├── config/              # viper 配置加载
│   ├── cron/                # CronJob 定时任务调度器
│   ├── db/                  # SQLite 数据库（纯 Go）
│   ├── logger/              # logrus 日志
│   ├── module/              # 平台模块接口 + 9 个平台实现
│   │   ├── module.go        # Module 接口定义
│   │   ├── manager.go       # 模块管理器
│   │   ├── notify_helper.go # 统一推送入口
│   │   ├── bilibili.go
│   │   ├── douyu.go
│   │   ├── huya.go
│   │   ├── weibo.go
│   │   ├── acfun.go
│   │   ├── youtube.go
│   │   ├── twitter.go
│   │   ├── twitcasting.go
│   │   └── douyin.go        # ⚠️ 占位，接口反爬暂不可用
│   ├── onebot/              # OneBot v11 客户端（正向/反向 WS）
│   ├── service/             # WebUI HTTP 服务 + REST API
│   ├── telegram/            # Telegram Bot 推送集成
│   └── update/              # 模块热更新管理器
├── requests/                # 轻量 HTTP 工具包
├── DDBOT-WSa-Desktop/       # 前端源码（Vue 3 + TypeScript）
├── configs/
│   └── application.yaml     # 配置示例
├── build.ps1                # 一键多平台构建脚本
└── go.mod                   # github.com/cnxysoft/DDBOT-WSa
```

---

## 编译

### 前置要求

- Go 1.21+
- Node.js 18+（构建前端时需要）

### 一键构建（推荐）

```powershell
# 先构建前端（有前端改动时）
cd DDBOT-WSa-Desktop
npm install
npm run build
cd ..

# 构建当前平台（windows/amd64）
.\build.ps1

# 构建所有平台（6 个）
.\build.ps1 -All

# 构建指定平台
.\build.ps1 -Target linux-amd64
```

构建产物在 `dist/` 目录，同时在项目根目录生成 `ddbot.exe` 快捷副本。

### 手动编译

```bash
# 纯 Go，无 CGO 依赖
CGO_ENABLED=0 go build -o ddbot ./cmd/main.go

# 带裁剪（减小体积）
CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o ddbot ./cmd/main.go
```

### 开发模式（带热重载前端）

```bash
# 终端 1：运行 Go 后端
go run ./cmd/main.go

# 终端 2：Vite 开发服务器（前端热重载）
cd DDBOT-WSa-Desktop
npm run dev
```

---

## 数据库

数据存储在运行目录下的 `ddbot.db`（SQLite）：

| 表名 | 用途 |
|------|------|
| `concerns` | 订阅列表 |
| `concern_configs` | 每群推送配置（@/过滤词/开关） |
| `command_configs` | 每群命令开关 |
| `admins` | 管理员列表 |
| `blocklist` | 黑名单 |
| `scores` | 签到积分 |
| `group_silence` | 群静音状态 |
| `cron_jobs` | 定时任务 |
| `telegram_channels` | Telegram 绑定关系 |

---

## 致谢

- [DDBOT-WSa](https://github.com/cnxysoft/DDBOT-WSa) — 原版项目，提供平台 API 参考
- [NapCat](https://github.com/NapNeko/NapCatQQ) — 推荐搭配的 OneBot 实现
- [modernc.org/sqlite](https://gitlab.com/cznic/sqlite) — 纯 Go SQLite 驱动
- [gorilla/websocket](https://github.com/gorilla/websocket) — WebSocket 通信
- [spf13/viper](https://github.com/spf13/viper) — 配置管理

---

## License

MIT
