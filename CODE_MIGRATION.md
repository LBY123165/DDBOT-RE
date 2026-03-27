# 代码迁移对照文档

> **面向原版开发者**：本文档详细说明原版 `DDBOT-WSa`（`DDBOT-WSa/` 目录）与重构版（`internal/` 目录）之间的代码对应关系，方便你在熟悉原版结构的基础上快速理解、修改和扩展新版代码。

---

## 目录

- [总体架构变化](#总体架构变化)
- [文件规模对比](#文件规模对比)
- [模块逐一对照](#模块逐一对照)
  - [1. Bot 协议层](#1-bot-协议层)
  - [2. 命令系统](#2-命令系统)
  - [3. 关注/订阅系统](#3-关注订阅系统)
  - [4. 平台模块](#4-平台模块)
  - [5. 数据库层](#5-数据库层)
  - [6. 推送通知](#6-推送通知)
  - [7. Telegram 集成](#7-telegram-集成)
  - [8. 定时任务](#8-定时任务)
  - [9. 权限系统](#9-权限系统)
  - [10. 配置系统](#10-配置系统)
  - [11. WebUI 管理面板](#11-webui-管理面板)
  - [12. 消息构造层](#12-消息构造层)
  - [13. 模板引擎](#13-模板引擎)
  - [14. 工具函数](#14-工具函数)
- [依赖对比](#依赖对比)
- [已移除的模块](#已移除的模块)
- [新增的能力](#新增的能力)
- [开发注意事项](#开发注意事项)

---

## 总体架构变化

```
原版架构（lsp 中心化）           重构版架构（internal 分层）
─────────────────────────       ─────────────────────────────
bot/bot/bot.go                   internal/onebot/bot.go
  └─ MiraiGo（直连QQ协议）         └─ OneBot v11 WebSocket（对接NapCat）

lsp/                             internal/
  ├─ module.go        (34KB)       ├─ module/      (13个文件，每平台1文件)
  ├─ groupCommand.go  (34KB)       ├─ command/     (9个文件，按功能分类)
  ├─ privateCommand.go(48KB)       ├─ db/          (3个文件，SQL+迁移)
  ├─ iCommand.go      (33KB)       ├─ service/     (webui.go)
  ├─ concern/         (13文件)      ├─ telegram/    (2个文件)
  └─ {platform}/      (每平台多文件) └─ cron/        (调度器)

数据库: BuntDB (KV)              数据库: SQLite（纯Go，关系型）
代码量: 500+ .go 文件            代码量: 39 .go 文件
```

**核心差异**：原版以 `lsp/` 包为中心，所有业务逻辑都注册到这里，高度耦合；新版按功能域分包，每个平台独立文件，依赖通过接口注入。

---

## 文件规模对比

| 模块 | 原版文件数 | 原版总大小 | 新版文件数 | 新版总大小 | 压缩比 |
|------|---------|---------|---------|---------|------|
| 平台模块（所有平台） | 120+ | ~2 MB | 13 | ~80 KB | ~96% 压缩 |
| 命令系统 | 4（共 ~120KB） | 120 KB | 9 | ~56 KB | ~54% 压缩 |
| 关注/状态管理 | 13（concern/） | ~90 KB | 1（concern/manager.go） | 2 KB | ~98% 压缩 |
| 数据库 | lsp/buntdb/ 10文件 | ~60 KB | 3 | ~26 KB | ~57% 压缩 |
| MiraiGo 协议栈 | 100+ | ~1.5 MB | **已完全移除** | 0 | — |

> **为什么压缩这么大？** 原版大量 `.pb.go` Protobuf 生成代码（bilibili 375KB、acfun 163KB、douyin 109KB、weibo 65KB）已全部移除，改用直接 HTTP + JSON 解析。

---

## 模块逐一对照

### 1. Bot 协议层

#### 原版：`bot/bot/bot.go`
```
bot/bot/
├── bot.go       # Bot 结构体，封装 *client.QQClient（MiraiGo）
├── login.go     # QQ 登录逻辑（二维码/密码/token）
├── module.go    # 模块注册机制（MiraiGo 模板）
└── moduleinfo.go
```
- 核心：`Bot struct { *client.QQClient ... }`
- 直连 QQ 协议，需要维护 session.token 做断线重连
- `ReLogin()` 做自动重连

#### 新版：`internal/onebot/`
```
internal/onebot/
├── bot.go     # Bot 结构体，封装 WebSocket 连接
└── types.go   # OneBot 事件/消息类型定义
```
- 核心：`Bot struct { listenAddr/wsURL string; conn *websocket.Conn; ... }`
- **反向 WS 模式**（默认）：`NewReverseBot(addr, token)` — Bot 做 HTTP 服务端，等 NapCat 连入
- **正向 WS 模式**：`NewBot(wsURL, token)` — Bot 主动连接
- 事件通过 `RegisterHandler(fn EventHandler)` 注册回调

#### 关键 API 对照

| 原版 API | 新版 API | 说明 |
|---------|---------|------|
| `bot.SendGroupMessage(groupID, msg)` | `bot.SendGroupMsg(groupID, msg)` | 发群消息 |
| `bot.SendPrivateMessage(userID, msg)` | `bot.SendPrivateMsg(userID, msg)` | 发私聊消息 |
| `bot.SetGroupBan(gid, uid, dur)` | `bot.SetGroupBan(gid, uid, dur)` | 禁言 |
| `client.GroupMemberInfo` | `Event.Sender` 字段 | 发送者信息 |
| `message.NewSendingMessage()` | `onebot.Text()/Image()/At()` | 消息构造 |
| `MiraiGo event handlers` | `bot.RegisterHandler(fn)` | 事件注册 |

---

### 2. 命令系统

#### 原版：`lsp/groupCommand.go` + `lsp/privateCommand.go` + `lsp/iCommand.go`
```
lsp/
├── groupCommand.go   # 34KB，群命令处理（所有群命令堆在一个文件）
├── privateCommand.go # 48KB，私聊命令处理（所有私聊命令堆在一个文件）
├── iCommand.go       # 33KB，命令接口定义 + 大量业务逻辑
├── command.go        # 命令注册
└── commandRuntime.go # 命令运行时
```
- 命令注册通过 `bot/bot/module.go` 的模块系统
- 所有命令逻辑混在单个巨型文件中
- 与 BuntDB 状态管理强耦合

#### 新版：`internal/command/`
```
internal/command/
├── manager.go         # 命令管理器（注册/路由/中间件）
├── commands.go        # 命令总注册入口 + 黑名单/静音中间件
├── group_mgmt.go      # 群管理命令：订阅/配置/禁言/黑名单等
├── admin.go           # 管理员命令：grant/addadmin/removeadmin
├── private_mgmt.go    # 私聊管理命令：sysinfo/status/quit/log
├── fun_cmds.go        # 娱乐命令：签到/积分/roll/倒放
├── cron_cmd.go        # 定时任务命令：cron add/del/list
├── telegram_cmd.go    # Telegram 命令：tg bind/unbind/list
└── request_handler.go # 好友申请/入群邀请自动处理
```

**命令注册方式对比：**

```go
// 原版（lsp/module.go 中注册，MiraiGo 回调模式）
func (l *Lsp) GroupMessage(qqClient *client.QQClient, msg *message.GroupMessage) {
    // 直接处理，没有路由抽象
}

// 新版（命令管理器注册）
manager.RegisterGroupCommand("订阅", handleSubscribe)
manager.RegisterGroupCommand("config", handleConfig)
// 中间件：黑名单 → 静音检查 → 权限检查 → 命令执行
```

---

### 3. 关注/订阅系统

#### 原版：`lsp/concern/`（13个文件，~90KB）
```
lsp/concern/
├── stateManager.go   # 25KB，BuntDB 状态管理（核心）
├── registry.go       # 关注注册表（全局注册中心）
├── config.go         # 推送配置基类
├── config_at.go      # @ 配置
├── config_filter.go  # 过滤器配置
├── config_notify.go  # 通知开关
├── hook.go           # 事件钩子
├── identity.go       # 关注对象身份标识
├── keyset.go         # BuntDB key 命名规则
└── ...
```
- 核心是 `stateManager.go`，管理 BuntDB 中的所有关注状态
- 复杂的 keyset 命名空间机制（`concern:<site>:<uid>:<group>`）
- 基于事件总线（`eventbus`）做平台间解耦

#### 新版：`internal/concern/manager.go` + `internal/db/migration.go`
```go
// concern/manager.go（仅2KB，薄包装层）
type Manager struct{}
func (m *Manager) GetAll() ([]db.Concern, error)
func (m *Manager) Add(site, uid, name string, groupCode int64, ct string) error
func (m *Manager) Remove(site, uid string, groupCode int64) error

// db/migration.go（订阅CRUD的实际实现）
// 关注配置直接用 SQL 表存储，不再需要 keyset/stateManager
```

**原版 keyset → 新版 SQL 表对照：**

| 原版 BuntDB key | 新版 SQL 表 | 含义 |
|--------------|-----------|------|
| `concern:<site>:<uid>:<group>` | `concerns` 表 | 关注记录 |
| `concern_config:at:<group>` | `concern_configs` 表 `at_*` 字段 | @配置 |
| `concern_config:filter:<group>` | `concern_configs` 表 `filter_words` 字段 | 过滤词 |
| `concern_config:notify:<group>` | `concern_configs` 表 `notify_live/news` 字段 | 推送开关 |
| `permission:admin:<uid>` | `admins` 表 | 管理员列表 |
| `blocklist:<uid>` | `blocklist` 表 | 黑名单 |

---

### 4. 平台模块

#### 原版：`lsp/{platform}/`（每个平台 5~15 个文件）

以 bilibili 为例：
```
lsp/bilibili/
├── bilibili.go          # 平台入口注册
├── bilibili.pb.go       # Protobuf 定义（375KB！）
├── concern.go           # 关注逻辑（19KB）
├── concern_fresher.go   # 数据刷新器（15KB）
├── model.go             # 数据模型（45KB）
├── stateManager.go      # BuntDB 状态管理
├── config.go            # 推送配置
├── keyset.go            # BuntDB key
├── login.go             # B站登录
└── ... (20+ 文件)
```

#### 新版：`internal/module/{platform}.go`（每平台单文件）

```
internal/module/
├── bilibili.go      # 13KB，完整 B 站模块（直播+动态+番剧+专栏）
├── douyu.go         # 3KB，斗鱼直播
├── huya.go          # 5KB，虎牙直播
├── weibo.go         # 3KB，微博动态
├── acfun.go         # 3KB，AcFun 视频
├── youtube.go       # 4KB，YouTube RSS
├── twitter.go       # 16KB，Twitter（Nitter镜像）
├── twitcasting.go   # 5KB，TwitCasting
├── douyin.go        # 3KB，抖音（占位，反爬限制）
├── module.go        # 模块接口定义
├── manager.go       # 模块管理器
└── notify_helper.go # 统一推送入口
```

**平台模块接口对比：**

```go
// 原版（通过 lsp/concern/registry.go 注册，与 lsp.Lsp 强耦合）
func init() {
    concern.RegisterConcern(newBilibiliConcern())
}
type BilibiliConcern struct {
    *concern.StateManager
    // ...
}
func (c *BilibiliConcern) FreshCount() int { ... }
func (c *BilibiliConcern) EmitFresh() error { ... }

// 新版（实现 Module 接口，通过 SetBot 注入 Bot）
type BilibiliModule struct {
    bot    *onebot.Bot
    stopCh chan struct{}
}
func (m *BilibiliModule) Name() string    { return "bilibili" }
func (m *BilibiliModule) Start() error    { go m.pollLive(); go m.pollDynamic(); return nil }
func (m *BilibiliModule) Stop()           { close(m.stopCh) }
func (m *BilibiliModule) Reload() error   { m.Stop(); return m.Start() }
func (m *BilibiliModule) SetBot(b *onebot.Bot) { m.bot = b }
```

**各平台关键 API 参考（原版 → 新版）：**

| 平台 | 原版核心文件 | 新版文件 | API 变化 |
|------|-----------|---------|---------|
| bilibili | `lsp/bilibili/concern.go` + `.pb.go` | `module/bilibili.go` | Protobuf→JSON，直接调用公开 REST API |
| 斗鱼 | `lsp/douyu/concern.go` + `.pb.go` | `module/douyu.go` | Protobuf→HTTP，`betard.xin` API |
| 虎牙 | `lsp/huya/concern.go` | `module/huya.go` | HTML解析+降级方案 |
| 微博 | `lsp/weibo/concern.go` + `json_compat.go` | `module/weibo.go` | 同原版逻辑，简化字段解析 |
| AcFun | `lsp/acfun/concern.go` + `.pb.go` | `module/acfun.go` | Protobuf→JSON REST |
| YouTube | `lsp/youtube/fetchInfo.go` | `module/youtube.go` | 保持 RSS 解析方式 |
| Twitter | `lsp/twitter/fetchInfo.go` | `module/twitter.go` | 增加 Anubis PoW 反爬处理 |
| TwitCasting | `lsp/twitcasting/concern.go` | `module/twitcasting.go` | 官方公开 API，无需 OAuth |

---

### 5. 数据库层

#### 原版：BuntDB（KV 嵌入式）
```
lsp/buntdb/
├── buntdb.go     # BuntDB 封装
├── key.go        # key 命名规则（9KB，大量常量）
├── option.go     # 查询选项
└── shortcut.go   # 快捷操作（17KB）
```
- 所有数据用 JSON 字符串序列化存到 KV
- 每种数据类型都有独立的 keyset 命名空间
- 跨表查询需要在应用层做 join

#### 新版：SQLite（纯 Go，`modernc.org/sqlite`）
```
internal/db/
├── db.go         # 连接管理，WAL模式，Exec/Query/QueryRow
├── schema.go     # 建表 DDL（所有表结构集中定义）
└── migration.go  # 所有 CRUD 函数（600+ 行）
```

**Schema 速查：**

```sql
-- 订阅关注
CREATE TABLE concerns (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    site TEXT NOT NULL,          -- 平台名（bilibili/douyu/...）
    uid TEXT NOT NULL,           -- 平台 UID
    name TEXT,                   -- 显示名称
    group_code INTEGER NOT NULL, -- QQ 群号
    concern_type TEXT DEFAULT 'live',  -- live/news/live,news
    enable INTEGER DEFAULT 1,
    create_time DATETIME DEFAULT CURRENT_TIMESTAMP,
    update_time DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(site, uid, group_code)
);

-- 推送配置（每群每平台）
CREATE TABLE concern_configs (
    group_code INTEGER NOT NULL,
    site TEXT NOT NULL,
    at_all INTEGER DEFAULT 0,
    at_members TEXT DEFAULT '',   -- JSON 数组
    notify_live INTEGER DEFAULT 1,
    notify_news INTEGER DEFAULT 1,
    filter_words TEXT DEFAULT '',  -- JSON 数组
    PRIMARY KEY (group_code, site)
);

-- 管理员 / 黑名单 / 签到积分 / 群静音 / 定时任务 / Telegram绑定
-- 见 internal/db/schema.go
```

---

### 6. 推送通知

#### 原版：`lsp/notify.go` + 各平台 `notify.go`
```go
// 原版：通过 lsp.Lsp 的 SendGroupMessage 发送
// 每个平台在自己的 notify.go 里处理推送逻辑
func (l *Lsp) notifyBilibili(msg *BilibiliNotify) {
    groupMsg := buildBilibiliMsg(msg)
    l.bot.SendGroupMessage(msg.GroupCode, groupMsg)
}
```

#### 新版：`internal/module/notify_helper.go`（统一入口）
```go
// 新版：所有模块调用同一个函数
func SendNotify(bot *onebot.Bot, groupCode int64, site, uid string, msg NotifyMsg) {
    // 1. 检查关注是否存在
    // 2. 读取该群的推送配置（concern_configs）
    // 3. 应用关键词过滤
    // 4. 构建消息（根据 at_all/at_members 决定是否@）
    // 5. 发送群消息
    // 6. 可选：转发到 Telegram（telegram.ForwardToTelegram）
}
```

**配置参数对照：**

| 原版命令 | 新版命令 | 存储位置 |
|---------|---------|---------|
| `lsp config bilibili at all` | `!config at all` | `concern_configs.at_all` |
| `lsp config bilibili notify live off` | `!config notify live off` | `concern_configs.notify_live` |
| `lsp config bilibili filter add 词` | `!config filter add 词` | `concern_configs.filter_words` |

---

### 7. Telegram 集成

#### 原版：`lsp/telegram/` + `lsp/telegram_commands.go`
```
lsp/telegram/
├── telegram.go     # 使用 go-telegram-bot-api/v5 第三方库
└── publisher.go    # 11KB，推送逻辑
lsp/telegram_commands.go  # 15KB，TG 命令处理
```
- 依赖 `go-telegram-bot-api/v5`（外部库）
- 命令混在 `lsp/` 中

#### 新版：`internal/telegram/`
```
internal/telegram/
├── client.go    # 纯 HTTP，直接调用 Telegram Bot API（无第三方依赖）
└── channels.go  # Telegram 绑定的 DB CRUD
internal/command/telegram_cmd.go  # Telegram 命令
```
- **零外部依赖**，纯 `net/http` 调用 Telegram Bot API
- 配置通过 `application.yaml` 的 `telegram.bot_token` 设置

---

### 8. 定时任务

#### 原版：`lsp/cron.go`（使用 `robfig/cron/v3`）
```go
// 原版：依赖 robfig/cron 第三方库
import "github.com/robfig/cron/v3"
// cron 表达式与原版相同（5段格式）
```

#### 新版：`internal/cron/scheduler.go`
```go
// 新版：纯 Go 实现，无第三方依赖
// 支持标准 5 段 cron 表达式：分 时 日 月 周
// 消息内容支持 Go text/template：{{.Date}} {{.Time}} 等
// DB 表：cron_jobs
```

---

### 9. 权限系统

#### 原版：`lsp/permission/`（6个文件，~30KB）
```
lsp/permission/
├── permission.go      # 权限级别定义
├── stateManager.go    # 11KB，BuntDB权限状态
└── keyset.go          # BuntDB key 命名
```
权限级别：`Admin > GroupAdmin > Member`

#### 新版：集成在命令层 + `admins` 表
```go
// internal/command/manager.go
func isAdmin(userID int64) bool {
    return db.IsAdmin(userID)
}
func isSuperAdmin(userID int64) bool {
    for _, id := range config.GetBotConfig().SuperAdmins {
        if id == userID { return true }
    }
    return false
}
// 权限检查内联在命令处理函数中，无独立权限包
```

超级管理员在 `application.yaml` 配置：
```yaml
bot:
  super_admins: [123456789]  # 原版 lsp.yaml 的 admin 列表
```

---

### 10. 配置系统

#### 原版：`bot/config/config.go` + `lsp/cfg/config.go`
- 使用 `lsp.yaml` 做 Bot 主配置
- 各模块有独立的配置文件（`lsp/bilibili/config.go` 等）

#### 新版：`internal/config/config.go`
```go
// 单一配置文件 application.yaml（viper 读取）
type BotConfig struct {
    SuperAdmins []int64
}
type OneBotConfig struct {
    WsListen    string  // 反向 WS 监听地址
    WsURL       string  // 正向 WS 地址
    AccessToken string
}
type BilibiliConfig struct {
    SESSDATA string
    BiliJct  string
    Interval string
}
```

**配置文件对照：**

| 原版 `lsp.yaml` 字段 | 新版 `application.yaml` 字段 |
|--------------------|--------------------------|
| `admin` | `bot.super_admins` |
| `ws_url` | `onebot.ws_listen`（反向WS）或 `onebot.ws_url`（正向WS）|
| `bilibili.sessdata` | `bilibili.SESSDATA` |
| `telegram.token` | `telegram.bot_token` |

---

### 11. WebUI 管理面板

#### 原版：`admin/server.go`（49KB）+ Electron
```
admin/server.go   # 49KB，所有管理 API 堆在一个文件
DDBOT-WSa-Desktop/ # Electron 桌面应用
```
- 需要安装 Electron 运行时
- 前后端分离，Electron 打包桌面 App

#### 新版：`internal/service/webui.go`（27KB）+ embed.FS
```go
// webui.go 同时提供：
// 1. REST API（/api/v1/*）
// 2. 前端静态文件（embed.FS，不需要外部文件）
// 端口 3000（可配置）
```
- 浏览器直接访问 `http://IP:3000`
- 前端 Vue3+TypeScript 构建产物通过 `//go:embed dist` 打包进二进制
- 新增"连接管理"页面（`/connection`）：可视化切换反向/正向 WS 模式

**API 路由对照：**

| 原版 admin API | 新版 API | 功能 |
|-------------|---------|------|
| `GET /api/bot/status` | `GET /api/v1/onebot/status` | Bot 状态 |
| `GET /api/modules` | `GET /api/v1/modules` | 模块列表 |
| `POST /api/module/start` | `POST /api/v1/modules/{name}/start` | 启动模块 |
| `GET /api/subs` | `GET /api/v1/subs` | 订阅列表 |
| 无 | `GET/POST /api/v1/settings` | 连接配置读写 |
| 无 | `POST /api/v1/settings/reconnect` | 热重连 |

---

### 12. 消息构造层

#### 原版：`lsp/mmsg/`（13 个文件，消息构造 DSL）
```
lsp/mmsg/
├── writer.go    # 6KB，消息构造器
├── image.go     # 5KB，图片消息
├── at.go        # @消息
├── video.go     # 视频消息
├── target.go    # 发送目标
└── ...
```

#### 新版：`internal/onebot/types.go`（直接构造 CQ 码）
```go
// onebot/types.go 和 bot.go 中的辅助函数
func Text(text string) string { return text }
func Image(url string) string { return fmt.Sprintf("[CQ:image,url=%s]", url) }
func At(uid int64) string     { return fmt.Sprintf("[CQ:at,qq=%d]", uid) }
func AtAll() string           { return "[CQ:at,qq=all]" }
func Video(url string) string { return fmt.Sprintf("[CQ:video,file=%s]", url) }
```
- 直接拼接 CQ 码字符串，比 mmsg 更直观
- 没有复杂的 DSL，发送时直接传字符串

---

### 13. 模板引擎

#### 原版：`lsp/template/`（20+ 文件，~140KB）
```
lsp/template/
├── exec.go          # 31KB，模板执行引擎
├── funcs.go         # 28KB，内置函数库
├── funcs_ext.go     # 29KB，扩展函数
└── ... (20 个文件)
```
- 高度定制的 Go template 扩展
- 支持数据库查询、HTTP 请求等高级模板函数
- CronJob 消息模板通过此引擎渲染

#### 新版：标准 `text/template`（无独立包）
```go
// internal/cron/scheduler.go 中直接使用 text/template
import "text/template"

type cronTemplateData struct {
    Date    string
    Time    string
    WeekDay string
    // ...
}
// 消息模板：{{.Date}} 每日早报 {{.Time}}
```
- 只保留基本模板功能，不支持原版的 DB 查询/HTTP 请求模板函数
- 如需复杂模板，可在此基础上扩展

---

### 14. 工具函数

#### 原版：`utils/`（大量工具，强依赖 MiraiGo）
```
utils/
├── helper.go    # 16KB，通用工具（含 MiraiGo 消息工具）
├── miraigo.go   # MiraiGo 特有工具
├── hackedBot.go # MiraiGo 内部 API 暴力调用
├── image.go     # 图片处理
└── ...
```

#### 新版：`requests/requests.go`（轻量 HTTP 工具）
```go
// requests/requests.go（自有包，非第三方）
// 提供：Get/Post/GetJSON/PostJSON + 重试/超时/UA伪装
// 不依赖任何外部 HTTP 库
```
- 不依赖 MiraiGo 的工具函数已迁移到 `requests/` 包
- 图片处理、视频处理等功能按需在各模块内实现

---

## 依赖对比

### 原版核心依赖（`DDBOT-WSa/go.mod`）

```
MiraiGo                    # ← 已完全移除
BuntDB                     # ← 已完全移除
go-telegram-bot-api/v5     # ← 已完全移除（改用纯HTTP）
robfig/cron/v3             # ← 已完全移除（纯Go重写）
google.golang.org/protobuf # ← 已完全移除（不再使用Protobuf）
guonaihong/gout            # ← 已完全移除
samber/lo                  # ← 已完全移除
```

### 新版依赖（精简后）

```
modernc.org/sqlite   # 纯Go SQLite（替代 buntdb + mattn/go-sqlite3）
gorilla/websocket    # WebSocket 客户端/服务端
spf13/viper          # 配置文件读取
sirupsen/logrus      # 日志
lestrrat-go/file-rotatelogs  # 日志轮转
```

> 依赖数量从 40+ 个减少到 5 个核心依赖，无 CGO，无 Protobuf。

---

## 已移除的模块

以下原版模块在新版中**不存在**，若需要可重新实现：

| 模块 | 原版位置 | 说明 |
|------|---------|------|
| MiraiGo 协议栈 | `miraigo/` (100+文件) | 已用 OneBot WS 替代 |
| Protobuf 定义 | `lsp/*/_.pb.go` | 已改用 JSON API |
| 二维码登录 | `bot/bot/login.go` | 由 NapCat 负责登录 |
| lolicon 图池 | `image_pool/lolicon_pool/` | 可按需添加 |
| 代理池 | `proxy_pool/` | 可按需添加 |
| msg-marker | `msg-marker/` | OneBot 侧自动处理 |
| 复杂模板引擎 | `lsp/template/` | 改用标准 text/template |
| BuntDB 状态管理 | `lsp/buntdb/` | 改用 SQLite |
| eventbus | `lsp/eventbus/` | 模块间直接调用 |

---

## 新增的能力

以下是**原版没有、新版新增**的功能：

| 功能 | 实现位置 | 说明 |
|------|---------|------|
| 反向 WS 服务端 | `internal/onebot/bot.go` | Bot 做 WS 服务端，等 NapCat 连入 |
| Twitter 模块 | `internal/module/twitter.go` | Nitter镜像+Anubis PoW 反爬 |
| TwitCasting 模块 | `internal/module/twitcasting.go` | 官方公开 API |
| B站番剧/专栏 | `internal/module/bilibili.go` | bangumi/article concern_type |
| 连接管理页面 | `ConnectionSettings.vue` | WebUI 可视化切换WS模式 |
| embed.FS 内嵌前端 | `internal/assets/embed.go` | 二进制自包含，无外部依赖 |
| 签到/积分系统 | `internal/command/fun_cmds.go` | 每日签到+排行榜 |
| 群静音模式 | `command/group_mgmt.go` + `db/` | `!silence on/off` |
| 邀请策略 | `command/group_mgmt.go` | `!invite on/off` |
| 申请自动处理 | `command/request_handler.go` | 好友申请/入群邀请策略 |
| 倒放命令 | `command/fun_cmds.go` | 调用 ffmpeg，回复视频消息 |
| 热配置读写 | `service/webui.go` | WebUI 在线修改 application.yaml |

---

## 开发注意事项

### 1. 添加新平台模块

参考 `internal/module/douyu.go`（最简单的模块），实现 `Module` 接口：

```go
type MyModule struct {
    bot    *onebot.Bot
    stopCh chan struct{}
}

func (m *MyModule) Name() string  { return "myplatform" }
func (m *MyModule) SetBot(b *onebot.Bot) { m.bot = b }
func (m *MyModule) Start() error {
    m.stopCh = make(chan struct{})
    go m.poll()
    return nil
}
func (m *MyModule) Stop()         { close(m.stopCh) }
func (m *MyModule) Reload() error { m.Stop(); return m.Start() }
func (m *MyModule) Status() Status { return StatusRunning }

func (m *MyModule) poll() {
    ticker := time.NewTicker(60 * time.Second)
    for {
        select {
        case <-ticker.C:
            m.check()
        case <-m.stopCh:
            return
        }
    }
}

func (m *MyModule) check() {
    concerns, _ := db.GetAllConcernsBySite("myplatform")
    for _, c := range concerns {
        // 调用平台 API 检查是否有新内容
        // 有则调用 module.SendNotify(m.bot, c.GroupCode, "myplatform", c.UID, msg)
    }
}
```

然后在 `internal/update/module.go` 的 `CreateModule()` 中注册：
```go
case "myplatform":
    return &module.MyModule{}
```

### 2. 添加新命令

在对应的 command 文件中添加，然后在 `internal/command/commands.go` 的 `RegisterAll()` 注册：

```go
func RegisterAll(manager *Manager, bot *onebot.Bot) {
    // 群命令
    manager.RegisterGroupCommand("新命令", func(event *onebot.Event, args []string) {
        bot.SendGroupMsg(event.GroupID, "响应")
    })
}
```

### 3. 注意事项

1. **不要对 `DDBOT-WSa/` 目录下的代码做修改**，它仅作参考；新版代码全部在 `internal/` 和 `cmd/` 中
2. **`go mod tidy` 慎用**：`DDBOT-WSa/` 子目录中有 MiraiGo 的 local replace，跑 tidy 会扫进来导致 go.mod 被污染
3. **前端修改后需要重新构建**：修改 `DDBOT-WSa-Desktop/src/` 后需要运行 `npm run build`，然后运行 `.\build.ps1` 会自动同步 dist
4. **CGO_ENABLED=0**：现在的 SQLite 驱动是纯 Go，不需要 CGO，所有平台直接交叉编译
5. **数据库文件**：运行目录下的 `ddbot.db`，SQLite WAL 模式，Schema 在 `internal/db/schema.go`
