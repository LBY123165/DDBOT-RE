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

// YoutubeModule YouTube 平台模块
type YoutubeModule struct {
	name       string
	version    string
	status     Status
	bot        *onebot.Bot
	stopChan   chan struct{}
	wg         sync.WaitGroup
	mu         sync.Mutex
	videoCache map[string]string // channelID → 最新视频 ID
}

func NewYoutubeModule() *YoutubeModule {
	return &YoutubeModule{
		name:       "youtube",
		version:    "2.0.0",
		status:     StatusStopped,
		videoCache: make(map[string]string),
	}
}

func (m *YoutubeModule) Name() string           { return m.name }
func (m *YoutubeModule) Version() string        { return m.version }
func (m *YoutubeModule) Status() Status         { return m.status }
func (m *YoutubeModule) SetBot(bot *onebot.Bot) { m.bot = bot }

func (m *YoutubeModule) Start() error {
	logger.Infof("启动 YouTube 模块 (v%s)", m.version)
	m.stopChan = make(chan struct{})
	m.status = StatusRunning
	m.wg.Add(1)
	go m.monitorVideos()
	return nil
}

func (m *YoutubeModule) Stop() {
	if m.stopChan != nil {
		close(m.stopChan)
		m.wg.Wait()
		m.stopChan = nil
	}
	m.status = StatusStopped
	logger.Infof("停止 YouTube 模块")
}

func (m *YoutubeModule) Reload() error {
	m.Stop()
	return m.Start()
}

func (m *YoutubeModule) monitorVideos() {
	defer m.wg.Done()
	t := time.NewTicker(60 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			m.checkAllChannels()
		case <-m.stopChan:
			return
		}
	}
}

func (m *YoutubeModule) checkAllChannels() {
	concerns, err := db.GetAllConcernsBySite(m.name)
	if err != nil {
		logger.Errorf("[youtube] 查询关注失败: %v", err)
		return
	}
	for _, c := range concerns {
		m.updateChannelVideos(c.GroupCode, c.UID, c.Name)
	}
}

func (m *YoutubeModule) updateChannelVideos(groupCode int64, channelID, name string) {
	resp, err := requests.Get(fmt.Sprintf(
		"https://www.youtube.com/feeds/videos.xml?channel_id=%s", channelID))
	if err != nil {
		logger.Debugf("[youtube] 获取频道 RSS 失败: %v", err)
		return
	}
	xml := resp.Text()
	if xml == "" {
		return
	}
	// 简单提取第一个 <yt:videoId> 标签
	const tag = "<yt:videoId>"
	videoID := findTag(xml, tag)
	if videoID == "" {
		return
	}

	m.mu.Lock()
	cachedID := m.videoCache[channelID]
	if videoID == cachedID {
		m.mu.Unlock()
		return
	}
	m.videoCache[channelID] = videoID
	isFirst := cachedID == "" // 首次拉取不通知，避免历史刷屏
	m.mu.Unlock()

	if isFirst {
		return
	}

	// 尝试提取视频标题
	title := findTag(xml, "<title>")
	dispName := name
	if dispName == "" {
		dispName = channelID
	}
	var msg string
	if title != "" && title != dispName {
		msg = fmt.Sprintf("🎬 %s 发布了新视频\n📹 %s\n🔗 https://www.youtube.com/watch?v=%s", dispName, title, videoID)
	} else {
		msg = fmt.Sprintf("🎬 %s 发布了新视频\n🔗 https://www.youtube.com/watch?v=%s", dispName, videoID)
	}
	logger.Infof("[youtube] 新视频通知 → 群%d: %s", groupCode, videoID)
	SendNotify(m.bot, m.name, channelID, groupCode, NotifyTypeNews, msg, "")
}

// findTag 从 XML 字符串中提取第一个指定标签的值
func findTag(xml, tag string) string {
	start := 0
	for i := 0; i < len(xml)-len(tag); i++ {
		if xml[i:i+len(tag)] == tag {
			start = i + len(tag)
			break
		}
	}
	if start == 0 {
		return ""
	}
	end := start
	for end < len(xml) && xml[end] != '<' {
		end++
	}
	return xml[start:end]
}
