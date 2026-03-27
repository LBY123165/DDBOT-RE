package module

import (
	"fmt"
	"sync"
	"time"

	"github.com/cnxysoft/DDBOT-WSa/internal/db"
	"github.com/cnxysoft/DDBOT-WSa/internal/logger"
	"github.com/cnxysoft/DDBOT-WSa/internal/onebot"
	"github.com/cnxysoft/DDBOT-WSa/requests"
)

// DouyuModule 斗鱼平台模块
type DouyuModule struct {
	name      string
	version   string
	status    Status
	bot       *onebot.Bot
	stopChan  chan struct{}
	wg        sync.WaitGroup
	mu        sync.Mutex
	liveCache map[string]bool // roomID → 是否已通知
}

// DouyuLiveInfo 直播间信息
type DouyuLiveInfo struct {
	RoomID   int64
	Title    string
	Category string
	IsLiving bool
}

func NewDouyuModule() *DouyuModule {
	return &DouyuModule{
		name:      "douyu",
		version:   "2.0.0",
		status:    StatusStopped,
		liveCache: make(map[string]bool),
	}
}

func (m *DouyuModule) Name() string           { return m.name }
func (m *DouyuModule) Version() string        { return m.version }
func (m *DouyuModule) Status() Status         { return m.status }
func (m *DouyuModule) SetBot(bot *onebot.Bot) { m.bot = bot }

func (m *DouyuModule) Start() error {
	logger.Infof("启动斗鱼模块 (v%s)", m.version)
	m.stopChan = make(chan struct{})
	m.status = StatusRunning
	m.wg.Add(1)
	go m.monitorLive()
	return nil
}

func (m *DouyuModule) Stop() {
	if m.stopChan != nil {
		close(m.stopChan)
		m.wg.Wait()
		m.stopChan = nil
	}
	m.status = StatusStopped
	logger.Infof("停止斗鱼模块")
}

func (m *DouyuModule) Reload() error {
	m.Stop()
	return m.Start()
}

func (m *DouyuModule) monitorLive() {
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

func (m *DouyuModule) checkAllLives() {
	concerns, err := db.GetAllConcernsBySite(m.name)
	if err != nil {
		logger.Errorf("[douyu] 查询关注失败: %v", err)
		return
	}
	for _, c := range concerns {
		if living, info := m.getLiveStatus(c.UID); living {
			m.sendLiveNotify(c.GroupCode, c.Name, c.UID, info)
		} else {
			// 下播时清除缓存，以便下次开播能重新通知
			m.mu.Lock()
			delete(m.liveCache, c.UID)
			m.mu.Unlock()
		}
	}
}

func (m *DouyuModule) getLiveStatus(roomID string) (bool, *DouyuLiveInfo) {
	resp, err := requests.Get(fmt.Sprintf("https://www.douyu.com/betard/%s", roomID))
	if err != nil {
		return false, nil
	}
	data, ok := resp.JSON()
	if !ok {
		return false, nil
	}
	room := data.Get("room")
	isLiving := room.Get("show_status").ToInt() == 1
	info := &DouyuLiveInfo{
		RoomID:   room.Get("room_id").ToInt(),
		Title:    room.Get("room_name").ToString(),
		Category: room.Get("cate_name").ToString(),
		IsLiving: isLiving,
	}
	return isLiving, info
}

func (m *DouyuModule) sendLiveNotify(groupCode int64, name, uid string, info *DouyuLiveInfo) {
	m.mu.Lock()
	alreadyNotified := m.liveCache[uid]
	if !alreadyNotified {
		m.liveCache[uid] = true
	}
	m.mu.Unlock()
	if alreadyNotified {
		return
	}

	msg := fmt.Sprintf("🔴 %s 在斗鱼开播啦！\n📺 %s\n🏷️ %s\n🔗 https://www.douyu.com/%d",
		name, info.Title, info.Category, info.RoomID)
	logger.Infof("[douyu] 开播通知 → 群%d: %s", groupCode, info.Title)
	SendNotify(m.bot, m.name, uid, groupCode, NotifyTypeLive, msg, "")
}
