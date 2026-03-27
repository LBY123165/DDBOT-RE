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

// WeiboModule 微博平台模块
type WeiboModule struct {
	name       string
	version    string
	status     Status
	bot        *onebot.Bot
	stopChan   chan struct{}
	wg         sync.WaitGroup
	mu         sync.Mutex
	weiboCache map[string]int64 // uid → 最新微博 ID
}

func NewWeiboModule() *WeiboModule {
	return &WeiboModule{
		name:       "weibo",
		version:    "2.0.0",
		status:     StatusStopped,
		weiboCache: make(map[string]int64),
	}
}

func (m *WeiboModule) Name() string           { return m.name }
func (m *WeiboModule) Version() string        { return m.version }
func (m *WeiboModule) Status() Status         { return m.status }
func (m *WeiboModule) SetBot(bot *onebot.Bot) { m.bot = bot }

func (m *WeiboModule) Start() error {
	logger.Infof("启动微博模块 (v%s)", m.version)
	m.stopChan = make(chan struct{})
	m.status = StatusRunning
	m.wg.Add(1)
	go m.monitorWeibo()
	return nil
}

func (m *WeiboModule) Stop() {
	if m.stopChan != nil {
		close(m.stopChan)
		m.wg.Wait()
		m.stopChan = nil
	}
	m.status = StatusStopped
	logger.Infof("停止微博模块")
}

func (m *WeiboModule) Reload() error {
	m.Stop()
	return m.Start()
}

func (m *WeiboModule) monitorWeibo() {
	defer m.wg.Done()
	t := time.NewTicker(60 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			m.checkAllUsers()
		case <-m.stopChan:
			return
		}
	}
}

func (m *WeiboModule) checkAllUsers() {
	concerns, err := db.GetAllConcernsBySite(m.name)
	if err != nil {
		logger.Errorf("[weibo] 查询关注失败: %v", err)
		return
	}
	for _, c := range concerns {
		m.updateUserWeibo(c.GroupCode, c.UID, c.Name)
	}
}

func (m *WeiboModule) updateUserWeibo(groupCode int64, uid, name string) {
	resp, err := requests.Get(fmt.Sprintf(
		"https://m.weibo.cn/api/container/getIndex?containerid=107603%s&count=1", uid))
	if err != nil {
		logger.Debugf("[weibo] 获取动态失败: %v", err)
		return
	}
	data, ok := resp.JSON()
	if !ok {
		return
	}
	cards := data.Get("data").Get("cards").ToArray()
	if len(cards) == 0 {
		return
	}
	mblog := cards[0].Get("mblog")
	wid := mblog.Get("id").ToInt()
	if wid == 0 {
		return
	}

	m.mu.Lock()
	cachedID, exists := m.weiboCache[uid]
	if exists && wid <= cachedID {
		m.mu.Unlock()
		return
	}
	m.weiboCache[uid] = wid
	m.mu.Unlock()

	if !exists {
		// 首次拉取只缓存，不推送，避免历史刷屏
		return
	}

	text := mblog.Get("text").ToString()
	if len(text) > 200 {
		text = text[:200] + "..."
	}
	msg := fmt.Sprintf("📢 %s 发了新微博\n%s\n🔗 https://weibo.com/%s/%d",
		name, text, uid, wid)
	logger.Infof("[weibo] 微博通知 → 群%d", groupCode)
	SendNotify(m.bot, m.name, uid, groupCode, NotifyTypeNews, msg, "")
}
