package module

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cnxysoft/DDBOT-WSa/internal/db"
	"github.com/cnxysoft/DDBOT-WSa/internal/logger"
	"github.com/cnxysoft/DDBOT-WSa/internal/onebot"
	"github.com/cnxysoft/DDBOT-WSa/requests"
)

// HuyaModule 虎牙平台模块
type HuyaModule struct {
	name      string
	version   string
	status    Status
	bot       *onebot.Bot
	stopChan  chan struct{}
	wg        sync.WaitGroup
	mu        sync.Mutex
	liveCache map[string]bool // uid → 是否已通知
}

// HuyaLiveInfo 直播间信息
type HuyaLiveInfo struct {
	Nick     string
	Title    string
	GameName string
	IsLiving bool
	RoomURL  string
}

func NewHuyaModule() *HuyaModule {
	return &HuyaModule{
		name:      "huya",
		version:   "2.0.0",
		status:    StatusStopped,
		liveCache: make(map[string]bool),
	}
}

func (m *HuyaModule) Name() string           { return m.name }
func (m *HuyaModule) Version() string        { return m.version }
func (m *HuyaModule) Status() Status         { return m.status }
func (m *HuyaModule) SetBot(bot *onebot.Bot) { m.bot = bot }

func (m *HuyaModule) Start() error {
	logger.Infof("启动虎牙模块 (v%s)", m.version)
	m.stopChan = make(chan struct{})
	m.status = StatusRunning
	m.wg.Add(1)
	go m.monitorLive()
	return nil
}

func (m *HuyaModule) Stop() {
	if m.stopChan != nil {
		close(m.stopChan)
		m.wg.Wait()
		m.stopChan = nil
	}
	m.status = StatusStopped
	logger.Infof("停止虎牙模块")
}

func (m *HuyaModule) Reload() error {
	m.Stop()
	return m.Start()
}

func (m *HuyaModule) monitorLive() {
	defer m.wg.Done()
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			m.checkAllLives()
		case <-m.stopChan:
			return
		}
	}
}

func (m *HuyaModule) checkAllLives() {
	concerns, err := db.GetAllConcernsBySite(m.name)
	if err != nil {
		logger.Errorf("[huya] 查询关注失败: %v", err)
		return
	}
	for _, c := range concerns {
		info := m.getLiveStatus(c.UID)
		if info == nil || !info.IsLiving {
			// 下播时清除缓存
			m.mu.Lock()
			delete(m.liveCache, c.UID)
			m.mu.Unlock()
			continue
		}
		// 回填用户名
		if info.Nick != "" && c.Name != info.Nick {
			_ = db.UpdateConcernName(m.name, c.UID, info.Nick)
		}
		name := c.Name
		if name == "" {
			name = info.Nick
		}
		if name == "" {
			name = c.UID
		}
		m.sendLiveNotify(c.GroupCode, name, c.UID, info)
	}
}

// getLiveStatus 通过虎牙移动端 API 获取直播状态
// 虎牙移动端房间号查询接口（channel 对应房间号/昵称）
func (m *HuyaModule) getLiveStatus(uid string) *HuyaLiveInfo {
	// 虎牙 PC Web API（获取直播间信息）
	resp, err := requests.Get(fmt.Sprintf(
		"https://www.huya.com/cache.php?m=Live&do=profileRoom&roomid=%s", uid))
	if err != nil {
		logger.Debugf("[huya] 获取直播状态失败 uid=%s: %v", uid, err)
		return nil
	}
	data, ok := resp.JSON()
	if !ok {
		// 降级：尝试解析页面文本判断直播状态
		return m.getLiveStatusFromPage(uid)
	}
	status := data.Get("status").ToInt()
	if status != 200 {
		return m.getLiveStatusFromPage(uid)
	}
	roomData := data.Get("data")
	isLiving := roomData.Get("isOn").ToInt() == 1
	return &HuyaLiveInfo{
		Nick:     roomData.Get("nick").ToString(),
		Title:    roomData.Get("introduction").ToString(),
		GameName: roomData.Get("gameFullName").ToString(),
		IsLiving: isLiving,
		RoomURL:  fmt.Sprintf("https://www.huya.com/%s", uid),
	}
}

// getLiveStatusFromPage 降级方案：解析 HTML 页面判断直播状态
func (m *HuyaModule) getLiveStatusFromPage(uid string) *HuyaLiveInfo {
	resp, err := requests.Get(fmt.Sprintf("https://m.huya.com/%s", uid))
	if err != nil {
		return nil
	}
	text := resp.Text()
	if text == "" {
		return nil
	}
	// 通过页面标题包含特征词判断直播状态
	// 虎牙移动端直播中的页面 title 包含 "正在直播" 或 "直播中"
	isLiving := strings.Contains(text, "正在直播") || strings.Contains(text, "liveStatus:1")

	// 尝试提取主播昵称（移动端页面 var hyPlayerConfig 中有 nick 字段）
	nick := extractBetween(text, `"nick":"`, `"`)
	title := extractBetween(text, `"introduction":"`, `"`)

	return &HuyaLiveInfo{
		Nick:     nick,
		Title:    title,
		IsLiving: isLiving,
		RoomURL:  fmt.Sprintf("https://www.huya.com/%s", uid),
	}
}

func (m *HuyaModule) sendLiveNotify(groupCode int64, name, uid string, info *HuyaLiveInfo) {
	m.mu.Lock()
	alreadyNotified := m.liveCache[uid]
	if !alreadyNotified {
		m.liveCache[uid] = true
	}
	m.mu.Unlock()
	if alreadyNotified {
		return
	}

	var msg string
	if info.Title != "" {
		msg = fmt.Sprintf("🔴 %s 在虎牙开播啦！\n📺 %s\n🏷️ %s\n🔗 %s",
			name, info.Title, info.GameName, info.RoomURL)
	} else {
		msg = fmt.Sprintf("🔴 %s 在虎牙开播啦！\n🔗 %s", name, info.RoomURL)
	}
	logger.Infof("[huya] 开播通知 → 群%d: %s", groupCode, name)
	SendNotify(m.bot, m.name, uid, groupCode, NotifyTypeLive, msg, "")
}

// extractBetween 从字符串中提取 start 和 end 之间的内容
func extractBetween(s, start, end string) string {
	i := strings.Index(s, start)
	if i < 0 {
		return ""
	}
	s = s[i+len(start):]
	j := strings.Index(s, end)
	if j < 0 {
		return ""
	}
	return s[:j]
}
