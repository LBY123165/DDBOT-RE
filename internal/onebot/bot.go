package onebot

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cnxysoft/DDBOT-WSa/internal/logger"
	"github.com/gorilla/websocket"
)

// EventHandler 事件处理函数类型
type EventHandler func(bot *Bot, event *Event)

// Bot OneBot WebSocket 客户端
// 支持两种模式：
//   - 反向 WS（默认）：Bot 做服务端，等待 OneBot 实现（NapCat/LLOneBot）主动连入
//   - 正向 WS：Bot 主动连接到 OneBot 的 ws-server
type Bot struct {
	// 反向 WS 监听地址，如 "0.0.0.0:8080"（留空则用正向 WS）
	listenAddr string
	// 正向 WS 目标地址，如 "ws://127.0.0.1:3001"（listenAddr 非空时忽略）
	wsURL       string
	accessToken string

	mu     sync.RWMutex
	conn   *websocket.Conn
	closed atomic.Bool
	stopCh chan struct{}
	wg     sync.WaitGroup

	// 反向 WS 服务器（用于 Stop 时关闭）
	httpServer *http.Server
	listener   net.Listener

	echoMu   sync.Mutex
	echoWait map[string]chan *Event

	// 事件处理器列表
	handlers []EventHandler

	// 自身 QQ 号（登录后由 lifecycle 事件填入）
	SelfID int64
}

// NewBot 创建 Bot（正向 WS 模式）
// wsURL: 如 "ws://127.0.0.1:3001"
func NewBot(wsURL, accessToken string) *Bot {
	return &Bot{
		wsURL:       wsURL,
		accessToken: accessToken,
		echoWait:    make(map[string]chan *Event),
	}
}

// NewReverseBot 创建 Bot（反向 WS 模式，服务端监听）
// listenAddr: 如 "0.0.0.0:8080"；accessToken 留空则不鉴权
func NewReverseBot(listenAddr, accessToken string) *Bot {
	return &Bot{
		listenAddr:  listenAddr,
		accessToken: accessToken,
		echoWait:    make(map[string]chan *Event),
	}
}

// OnEvent 注册事件处理器（可多次调用，按注册顺序触发）
func (b *Bot) OnEvent(h EventHandler) {
	b.handlers = append(b.handlers, h)
}

// Start 启动 Bot（非阻塞）
// - 反向 WS 模式：立即开始监听，等待 OneBot 连入
// - 正向 WS 模式：后台尝试连接，失败后自动重连
func (b *Bot) Start() error {
	b.stopCh = make(chan struct{})
	b.closed.Store(false)

	if b.listenAddr != "" {
		// 反向 WS：启动 HTTP 服务等待连入
		return b.startReverseWS()
	}
	// 正向 WS：后台重连
	b.wg.Add(1)
	go b.connectLoop()
	return nil
}

// Stop 断开连接并停止服务
func (b *Bot) Stop() {
	if b.closed.Swap(true) {
		return
	}
	close(b.stopCh)

	// 关闭当前 WS 连接
	b.mu.Lock()
	if b.conn != nil {
		_ = b.conn.Close()
		b.conn = nil
	}
	b.mu.Unlock()

	// 关闭 HTTP 服务器（反向 WS 模式）
	if b.httpServer != nil {
		_ = b.httpServer.Close()
	}
	if b.listener != nil {
		_ = b.listener.Close()
	}

	b.wg.Wait()
	logger.Info("[OneBot] 已停止")
}

// IsConnected 返回当前 WebSocket 是否已连接
func (b *Bot) IsConnected() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.conn != nil && !b.closed.Load()
}

// GetSelfID 返回 Bot 自身 QQ 号（未获取到时返回 0）
func (b *Bot) GetSelfID() int64 {
	return b.SelfID
}

// ─── 消息发送 API ─────────────────────────────────────────────────────────────

// SendGroupMsg 发送群消息
func (b *Bot) SendGroupMsg(groupID int64, segs ...Segment) error {
	return b.callAPI("send_group_msg", sendGroupMsgParams{
		GroupID: groupID,
		Message: segs,
	})
}

// SendGroupText 发送群文本消息（快捷方法）
func (b *Bot) SendGroupText(groupID int64, text string) error {
	return b.SendGroupMsg(groupID, Text(text))
}

// SendPrivateMsg 发送私聊消息
func (b *Bot) SendPrivateMsg(userID int64, segs ...Segment) error {
	return b.callAPI("send_private_msg", sendPrivateMsgParams{
		UserID:  userID,
		Message: segs,
	})
}

// SetGroupLeave 退出群聊（isDismiss=true 时解散群，仅群主可用）
func (b *Bot) SetGroupLeave(groupID int64, isDismiss bool) error {
	return b.callAPI("set_group_leave", map[string]interface{}{
		"group_id":   groupID,
		"is_dismiss": isDismiss,
	})
}

// SetGroupBan 单个用户禁言（duration=0 解除禁言，单位秒）
func (b *Bot) SetGroupBan(groupID, userID int64, duration int64) error {
	return b.callAPI("set_group_ban", map[string]interface{}{
		"group_id": groupID,
		"user_id":  userID,
		"duration": duration,
	})
}

// SetGroupWholeBan 全体禁言（enable=true 开启，false 关闭）
func (b *Bot) SetGroupWholeBan(groupID int64, enable bool) error {
	return b.callAPI("set_group_whole_ban", map[string]interface{}{
		"group_id": groupID,
		"enable":   enable,
	})
}

// SetFriendAddRequest 处理好友申请（flag 来自 request 事件，approve=true 同意，remark 备注）
func (b *Bot) SetFriendAddRequest(flag string, approve bool, remark string) error {
	return b.callAPI("set_friend_add_request", map[string]interface{}{
		"flag":    flag,
		"approve": approve,
		"remark":  remark,
	})
}

// SetGroupAddRequest 处理入群申请（subType: "add"|"invite"，reason 拒绝原因）
func (b *Bot) SetGroupAddRequest(flag, subType string, approve bool, reason string) error {
	return b.callAPI("set_group_add_request", map[string]interface{}{
		"flag":     flag,
		"sub_type": subType,
		"approve":  approve,
		"reason":   reason,
	})
}

// SendGroupMsgRich 发送群消息（图片+文字混排）
func (b *Bot) SendGroupMsgRich(groupID int64, segs ...Segment) error {
	return b.SendGroupMsg(groupID, segs...)
}

// ─── 反向 WS（服务端）─────────────────────────────────────────────────────────

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// startReverseWS 启动反向 WS HTTP 服务，等待 OneBot 连入
func (b *Bot) startReverseWS() error {
	ln, err := net.Listen("tcp", b.listenAddr)
	if err != nil {
		return fmt.Errorf("[OneBot] 监听 %s 失败: %w", b.listenAddr, err)
	}
	b.listener = ln

	mux := http.NewServeMux()
	mux.HandleFunc("/", b.handleReverseConn)
	mux.HandleFunc("/ws", b.handleReverseConn)
	mux.HandleFunc("/onebot/v11/ws", b.handleReverseConn)

	b.httpServer = &http.Server{Handler: mux}
	logger.Infof("[OneBot] 反向 WS 服务已启动，监听 %s，等待 OneBot 连入...", b.listenAddr)

	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		if err := b.httpServer.Serve(ln); err != nil && !b.closed.Load() {
			logger.Errorf("[OneBot] 反向 WS 服务异常退出: %v", err)
		}
	}()
	return nil
}

// handleReverseConn 处理 OneBot 连入的 WS 连接
func (b *Bot) handleReverseConn(w http.ResponseWriter, r *http.Request) {
	// 鉴权
	if b.accessToken != "" {
		token := r.Header.Get("Authorization")
		if token == "" {
			token = r.URL.Query().Get("access_token")
		} else if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}
		if token != b.accessToken {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			logger.Warnf("[OneBot] 连入鉴权失败，拒绝连接（来源: %s）", r.RemoteAddr)
			return
		}
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Errorf("[OneBot] WS 握手失败: %v", err)
		return
	}

	// 如果已有旧连接，关闭它
	b.mu.Lock()
	if b.conn != nil {
		logger.Warnf("[OneBot] 新连接覆盖旧连接（来源: %s）", r.RemoteAddr)
		_ = b.conn.Close()
	}
	b.conn = conn
	b.mu.Unlock()

	logger.Infof("[OneBot] OneBot 已连入（来源: %s）", r.RemoteAddr)
	b.readLoop()
	logger.Infof("[OneBot] 连接断开（来源: %s），等待重新连入...", r.RemoteAddr)
}

// ─── 正向 WS（客户端，自动重连）─────────────────────────────────────────────

// connectLoop 持续尝试连接（正向 WS 模式），连接成功后转入 readLoop
func (b *Bot) connectLoop() {
	defer b.wg.Done()
	retryInterval := 5 * time.Second
	attempt := 0
	for {
		if b.closed.Load() {
			return
		}
		if err := b.connect(); err != nil {
			attempt++
			if attempt == 1 {
				logger.Warnf("[OneBot] 连接失败，等待 OneBot 启动后自动重连（每 %v 重试一次）: %v", retryInterval, err)
			} else {
				logger.Debugf("[OneBot] 第 %d 次重连失败: %v", attempt, err)
			}
			select {
			case <-b.stopCh:
				return
			case <-time.After(retryInterval):
			}
			continue
		}
		attempt = 0
		b.readLoop()
		if b.closed.Load() {
			return
		}
		logger.Infof("[OneBot] 连接断开，%v 后重连...", retryInterval)
		select {
		case <-b.stopCh:
			return
		case <-time.After(retryInterval):
		}
	}
}

func (b *Bot) connect() error {
	header := http.Header{}
	if b.accessToken != "" {
		header.Set("Authorization", "Bearer "+b.accessToken)
	}

	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	conn, _, err := dialer.Dial(b.wsURL, header)
	if err != nil {
		return fmt.Errorf("[OneBot] 连接失败 %s: %w", b.wsURL, err)
	}
	b.mu.Lock()
	b.conn = conn
	b.mu.Unlock()
	logger.Infof("[OneBot] 已连接到 %s", b.wsURL)
	return nil
}

// ─── 读循环（正向/反向共用）──────────────────────────────────────────────────

func (b *Bot) readLoop() {
	for {
		b.mu.RLock()
		conn := b.conn
		b.mu.RUnlock()
		if conn == nil {
			return
		}

		_, raw, err := conn.ReadMessage()
		if err != nil {
			if b.closed.Load() {
				return
			}
			logger.Warnf("[OneBot] 读取断开: %v", err)
			b.mu.Lock()
			b.conn = nil
			b.mu.Unlock()
			return
		}

		var ev Event
		if err := json.Unmarshal(raw, &ev); err != nil {
			logger.Debugf("[OneBot] 解析事件失败: %v", err)
			continue
		}

		// 预解析消息段（供命令层使用，如倒放命令读取视频 URL）
		if len(ev.Message) > 0 && (ev.IsGroupMessage() || ev.IsPrivateMessage()) {
			var segs []Segment
			if err := json.Unmarshal(ev.Message, &segs); err == nil {
				ev.Segments = segs
			}
		}

		// 处理 API 回包
		if ev.Echo != "" {
			b.echoMu.Lock()
			ch, ok := b.echoWait[ev.Echo]
			b.echoMu.Unlock()
			if ok {
				ch <- &ev
			}
			continue
		}

		// 记录 SelfID
		if ev.SelfID != 0 && b.SelfID == 0 {
			b.SelfID = ev.SelfID
			logger.Infof("[OneBot] 已登录，Bot QQ: %d", b.SelfID)
		}

		// lifecycle connect 事件
		if ev.PostType == "meta_event" && ev.MetaType == "lifecycle" && ev.SubType == "connect" {
			logger.Infof("[OneBot] lifecycle 连接成功，Bot QQ: %d", ev.SelfID)
			if b.SelfID == 0 {
				go b.fetchSelfID()
			}
		}

		// 心跳事件静默处理
		if ev.PostType == "meta_event" && ev.MetaType == "heartbeat" {
			continue
		}

		// 分发事件
		for _, h := range b.handlers {
			go h(b, &ev)
		}
	}
}

// ─── API 调用 ─────────────────────────────────────────────────────────────────

// callAPI 向 OneBot 发送 API 调用（fire-and-forget，不等回包）
func (b *Bot) callAPI(action string, params interface{}) error {
	req := apiRequest{Action: action, Params: params}
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	b.mu.RLock()
	conn := b.conn
	b.mu.RUnlock()
	if conn == nil {
		return fmt.Errorf("[OneBot] 未连接")
	}
	return conn.WriteMessage(websocket.TextMessage, data)
}

// fetchSelfID 主动调用 get_login_info 获取 Bot QQ 号
func (b *Bot) fetchSelfID() {
	echo := "get_login_info_init"
	req := apiRequest{Action: "get_login_info", Echo: echo}
	data, err := json.Marshal(req)
	if err != nil {
		return
	}
	ch := make(chan *Event, 1)
	b.echoMu.Lock()
	b.echoWait[echo] = ch
	b.echoMu.Unlock()
	defer func() {
		b.echoMu.Lock()
		delete(b.echoWait, echo)
		b.echoMu.Unlock()
	}()

	b.mu.RLock()
	conn := b.conn
	b.mu.RUnlock()
	if conn == nil {
		return
	}
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return
	}

	select {
	case ev := <-ch:
		if ev.SelfID != 0 {
			b.SelfID = ev.SelfID
			logger.Infof("[OneBot] 获取到 Bot QQ: %d", b.SelfID)
		} else {
			var loginInfo struct {
				UserID int64 `json:"user_id"`
			}
			if json.Unmarshal(ev.Data, &loginInfo) == nil && loginInfo.UserID != 0 {
				b.SelfID = loginInfo.UserID
				logger.Infof("[OneBot] 获取到 Bot QQ: %d", b.SelfID)
			}
		}
	case <-time.After(5 * time.Second):
		logger.Warnf("[OneBot] 获取 SelfID 超时")
	}
}
