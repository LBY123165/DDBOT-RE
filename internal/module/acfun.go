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

// AcfunModule Acfun 平台模块
type AcfunModule struct {
	name       string
	version    string
	status     Status
	bot        *onebot.Bot
	stopChan   chan struct{}
	wg         sync.WaitGroup
	videoCache map[string]int64
}

func NewAcfunModule() *AcfunModule {
	return &AcfunModule{
		name:       "acfun",
		version:    "2.0.0",
		status:     StatusStopped,
		videoCache: make(map[string]int64),
	}
}

func (m *AcfunModule) Name() string           { return m.name }
func (m *AcfunModule) Version() string        { return m.version }
func (m *AcfunModule) Status() Status         { return m.status }
func (m *AcfunModule) SetBot(bot *onebot.Bot) { m.bot = bot }

func (m *AcfunModule) Start() error {
	logger.Infof("启动 acfun 模块 (v%s)", m.version)
	m.stopChan = make(chan struct{})
	m.status = StatusRunning
	m.wg.Add(1)
	go m.monitorVideos()
	return nil
}

func (m *AcfunModule) Stop() {
	if m.stopChan != nil {
		close(m.stopChan)
		m.wg.Wait()
		m.stopChan = nil
	}
	m.status = StatusStopped
	logger.Infof("停止 acfun 模块")
}

func (m *AcfunModule) Reload() error {
	m.Stop()
	return m.Start()
}

func (m *AcfunModule) monitorVideos() {
	defer m.wg.Done()
	t := time.NewTicker(60 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			m.checkAllVideos()
		case <-m.stopChan:
			return
		}
	}
}

func (m *AcfunModule) checkAllVideos() {
	concerns, err := db.GetAllConcernsBySite(m.name)
	if err != nil {
		logger.Errorf("[acfun] 查询关注失败: %v", err)
		return
	}
	for _, c := range concerns {
		m.updateVideos(c.GroupCode, c.UID, c.Name)
	}
}

func (m *AcfunModule) updateVideos(groupCode int64, uid, name string) {
	resp, err := requests.Get(fmt.Sprintf(
		"https://www.acfun.cn/rest/pc-direct/u/video?userId=%s", uid))
	if err != nil {
		return
	}
	data, ok := resp.JSON()
	if !ok {
		return
	}
	videos := data.Get("videoList").ToArray()
	if len(videos) == 0 {
		return
	}
	latest := videos[0]
	videoID := latest.Get("id").ToInt()
	if videoID == 0 {
		return
	}

	cachedID, exists := m.videoCache[uid]
	if exists && videoID <= cachedID {
		return
	}
	m.videoCache[uid] = videoID
	if !exists {
		// 首次拉取只缓存，不推送，避免历史视频刷屏
		return
	}

	title := latest.Get("title").ToString()
	vid := fmt.Sprintf("%d", videoID)
	msg := fmt.Sprintf("🎬 %s 发布了新视频\n📹 %s\n🔗 https://www.acfun.cn/v/ac%s", name, title, vid)
	logger.Infof("[acfun] 新视频通知 → 群%d: %s", groupCode, title)
	SendNotify(m.bot, m.name, uid, groupCode, NotifyTypeNews, msg, "")
}
