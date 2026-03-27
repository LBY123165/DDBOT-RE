// Package onebot 实现 OneBot v11 协议客户端
// 支持正向/反向 WebSocket 连接，兼容 NapCat、LLOneBot、go-cqhttp 等实现
package onebot

import "encoding/json"

// ─── 消息段 ───────────────────────────────────────────────────────────────────

// Segment 消息段（CQ码的 Go 结构表示）
type Segment struct {
	Type string            `json:"type"`
	Data map[string]string `json:"data"`
}

// Text 纯文本段
func Text(s string) Segment {
	return Segment{Type: "text", Data: map[string]string{"text": s}}
}

// Image 图片段（url 或 file 路径）
func Image(urlOrFile string) Segment {
	return Segment{Type: "image", Data: map[string]string{"file": urlOrFile}}
}

// At @某人
func At(qq string) Segment {
	return Segment{Type: "at", Data: map[string]string{"qq": qq}}
}

// AtAll @全体成员
func AtAll() Segment {
	return Segment{Type: "at", Data: map[string]string{"qq": "all"}}
}

// Video 视频段（url 或本地文件路径，支持 file:///path 格式）
func Video(urlOrFile string) Segment {
	return Segment{Type: "video", Data: map[string]string{"file": urlOrFile}}
}

// ─── 事件 ─────────────────────────────────────────────────────────────────────

// Event OneBot 上报事件（通用字段）
type Event struct {
	Time        int64           `json:"time"`
	SelfID      int64           `json:"self_id"`
	PostType    string          `json:"post_type"`
	MessageType string          `json:"message_type,omitempty"`
	SubType     string          `json:"sub_type,omitempty"`
	MessageID   int32           `json:"message_id,omitempty"`
	GroupID     int64           `json:"group_id,omitempty"`
	UserID      int64           `json:"user_id,omitempty"`
	Message     json.RawMessage `json:"message,omitempty"`
	RawMessage  string          `json:"raw_message,omitempty"`
	NoticeType  string          `json:"notice_type,omitempty"`
	RequestType string          `json:"request_type,omitempty"`
	MetaType    string          `json:"meta_event_type,omitempty"`
	Flag        string          `json:"flag,omitempty"`    // 请求事件标志（好友/入群申请处理用）
	Comment     string          `json:"comment,omitempty"` // 请求附加消息
	// 响应回包（API调用结果）
	Echo    string          `json:"echo,omitempty"`
	Status  string          `json:"status,omitempty"`
	RetCode int             `json:"retcode,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`

	// 解析后的消息段（由 bot 在接收时填充，避免重复解析）
	Segments []Segment `json:"-"`
}

// IsGroupMessage 判断是否群消息
func (e *Event) IsGroupMessage() bool {
	return e.PostType == "message" && e.MessageType == "group"
}

// IsPrivateMessage 判断是否私聊消息
func (e *Event) IsPrivateMessage() bool {
	return e.PostType == "message" && e.MessageType == "private"
}

// TextContent 从原始消息中提取纯文本（兼容字符串和数组格式）
func (e *Event) TextContent() string {
	if e.RawMessage != "" {
		return e.RawMessage
	}
	if len(e.Message) == 0 {
		return ""
	}
	// 尝试字符串格式
	var s string
	if err := json.Unmarshal(e.Message, &s); err == nil {
		return s
	}
	// 尝试数组格式
	var segs []Segment
	if err := json.Unmarshal(e.Message, &segs); err == nil {
		var out string
		for _, seg := range segs {
			if seg.Type == "text" {
				out += seg.Data["text"]
			}
		}
		return out
	}
	return ""
}

// ─── API 请求 ─────────────────────────────────────────────────────────────────

// apiRequest OneBot API 请求体
type apiRequest struct {
	Action string      `json:"action"`
	Params interface{} `json:"params"`
	Echo   string      `json:"echo,omitempty"`
}

// sendGroupMsgParams send_group_msg 参数
type sendGroupMsgParams struct {
	GroupID int64     `json:"group_id"`
	Message []Segment `json:"message"`
}

// sendPrivateMsgParams send_private_msg 参数
type sendPrivateMsgParams struct {
	UserID  int64     `json:"user_id"`
	Message []Segment `json:"message"`
}
