package module

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cnxysoft/DDBOT-WSa/internal/db"
	"github.com/cnxysoft/DDBOT-WSa/internal/logger"
	"github.com/cnxysoft/DDBOT-WSa/internal/onebot"
	"github.com/cnxysoft/DDBOT-WSa/requests"
)

// TwitcastingModule TwitCasting 直播监控模块
// 使用 TwitCasting 官方公开 API（无需 OAuth，404=未直播，200=直播中）
type TwitcastingModule struct {
	name    string
	version string
	status  Status
	bot     *onebot.Bot
	running bool

	stopCh    chan struct{}
	mu        sync.Mutex
	liveState map[string]bool // uid -> 是否正在直播
}

// twitcastingMovieResp TwitCasting API /movies/user 响应
type twitcastingMovieResp struct {
	Movie *struct {
		ID             string `json:"id"`
		Title          string `json:"title"`
		IsLive         bool   `json:"is_live"`
		LargeThumbnail string `json:"large_thumbnail"`
	} `json:"movie"`
	Broadcaster *struct {
		ID       string `json:"id"`
		ScreenID string `json:"screen_id"`
		Name     string `json:"name"`
		Image    string `json:"image"`
	} `json:"broadcaster"`
}

func NewTwitcastingModule() *TwitcastingModule {
	return &TwitcastingModule{
		name:      "twitcasting",
		version:   "1.0.0",
		liveState: make(map[string]bool),
	}
}

func (m *TwitcastingModule) Name() string    { return m.name }
func (m *TwitcastingModule) Version() string { return m.version }
func (m *TwitcastingModule) Status() Status  { return m.status }

func (m *TwitcastingModule) Start() error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return nil
	}
	m.running = true
	m.status = StatusRunning
	m.stopCh = make(chan struct{})
	m.liveState = make(map[string]bool)
	m.mu.Unlock()

	logger.Info("[twitcasting] 模块启动，轮询间隔 60s")
	go m.poll()
	return nil
}

func (m *TwitcastingModule) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.running {
		return
	}
	m.running = false
	m.status = StatusStopped
	close(m.stopCh)
	logger.Info("[twitcasting] 模块已停止")
}

func (m *TwitcastingModule) Reload() error {
	m.Stop()
	time.Sleep(200 * time.Millisecond)
	return m.Start()
}

func (m *TwitcastingModule) SetBot(bot *onebot.Bot) {
	m.bot = bot
}

func (m *TwitcastingModule) poll() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.checkAll()
		}
	}
}

func (m *TwitcastingModule) checkAll() {
	concerns, err := db.GetAllConcernsBySite("twitcasting")
	if err != nil {
		logger.Errorf("[twitcasting] 获取订阅失败: %v", err)
		return
	}
	// 按 uid 聚合群
	userMap := make(map[string][]db.Concern)
	for _, c := range concerns {
		userMap[c.UID] = append(userMap[c.UID], *c)
	}
	for uid, groups := range userMap {
		m.checkUser(uid, groups)
	}
}

func (m *TwitcastingModule) checkUser(uid string, groups []db.Concern) {
	live, title, thumbURL, err := m.fetchLiveStatus(uid)
	if err != nil {
		logger.Debugf("[twitcasting] 获取 %s 直播状态失败: %v", uid, err)
		return
	}

	m.mu.Lock()
	prev, existed := m.liveState[uid]
	m.liveState[uid] = live
	m.mu.Unlock()

	// 首次或无变化不推送
	if !existed {
		return
	}
	if prev == live {
		return
	}

	for _, c := range groups {
		if live {
			m.sendNotify(c.GroupCode, uid, title, thumbURL)
		}
	}
}

// fetchLiveStatus 使用 TwitCasting 官方 API v2
// GET https://apiv2.twitcasting.tv/users/{user_id}/current_live
// 无需 OAuth，404 = 未在直播，200 = 正在直播
func (m *TwitcastingModule) fetchLiveStatus(uid string) (live bool, title, thumbURL string, err error) {
	url := fmt.Sprintf("https://apiv2.twitcasting.tv/users/%s/current_live", uid)
	resp, err := requests.Get(url)
	if err != nil {
		return false, "", "", err
	}
	// 404 = 不在直播
	if resp.Status() == 404 {
		return false, "", "", nil
	}
	if resp.Status() != 200 {
		return false, "", "", fmt.Errorf("twitcasting API 返回 %d", resp.Status())
	}

	var result twitcastingMovieResp
	if err := json.Unmarshal([]byte(resp.Text()), &result); err != nil {
		return false, "", "", fmt.Errorf("解析响应失败: %w", err)
	}

	if result.Movie == nil || !result.Movie.IsLive {
		return false, "", "", nil
	}
	return true, result.Movie.Title, result.Movie.LargeThumbnail, nil
}

func (m *TwitcastingModule) sendNotify(groupCode int64, uid, title, thumbURL string) {
	displayName := uid
	link := fmt.Sprintf("https://twitcasting.tv/%s", uid)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🔴 %s 正在 TwitCasting 直播！\n", displayName))
	if title != "" {
		sb.WriteString(fmt.Sprintf("📺 %s\n", title))
	}
	sb.WriteString(fmt.Sprintf("🔗 %s", link))

	text := sb.String()
	logger.Infof("[twitcasting] 开播通知 → 群%d: %s 《%s》", groupCode, uid, title)
	SendNotify(m.bot, m.name, uid, groupCode, NotifyTypeLive, text, thumbURL)
}
