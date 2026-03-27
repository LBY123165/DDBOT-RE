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

// BilibiliModule bilibili 平台模块（直播 + 动态 + 番剧 + 专栏监控）
type BilibiliModule struct {
	name    string
	version string
	status  Status
	bot     *onebot.Bot

	stopChan chan struct{}
	wg       sync.WaitGroup

	// liveNotified: uid -> liveTimestamp（已通知时的开播时间戳，避免重复推送）
	mu            sync.Mutex
	liveNotified  map[string]int64  // uid -> start_ts
	dynamicLatest map[string]string // uid -> latest_dynamic_id_str
	bangumiFeed   map[string]string // uid -> latest_episode_id（番剧/追番）
	articleLatest map[string]string // uid -> latest_article_id（专栏）
}

// NewBilibiliModule 创建 bilibili 平台模块
func NewBilibiliModule() *BilibiliModule {
	return &BilibiliModule{
		name:          "bilibili",
		version:       "2.1.0",
		status:        StatusStopped,
		liveNotified:  make(map[string]int64),
		dynamicLatest: make(map[string]string),
		bangumiFeed:   make(map[string]string),
		articleLatest: make(map[string]string),
	}
}

func (m *BilibiliModule) Name() string           { return m.name }
func (m *BilibiliModule) Version() string        { return m.version }
func (m *BilibiliModule) Status() Status         { return m.status }
func (m *BilibiliModule) SetBot(bot *onebot.Bot) { m.bot = bot }

func (m *BilibiliModule) Start() error {
	logger.Infof("启动 bilibili 模块 (v%s)", m.version)
	m.stopChan = make(chan struct{})
	m.status = StatusRunning
	m.wg.Add(4)
	go m.monitorLive()
	go m.monitorDynamic()
	go m.monitorBangumi()
	go m.monitorArticle()
	return nil
}

func (m *BilibiliModule) Stop() {
	if m.stopChan != nil {
		close(m.stopChan)
		m.wg.Wait()
		m.stopChan = nil
	}
	m.status = StatusStopped
	logger.Infof("停止 bilibili 模块")
}

func (m *BilibiliModule) Reload() error {
	m.Stop()
	return m.Start()
}

// ─── 监控循环 ─────────────────────────────────────────────────────────────────

func (m *BilibiliModule) monitorLive() {
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

func (m *BilibiliModule) monitorDynamic() {
	defer m.wg.Done()
	t := time.NewTicker(90 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			m.checkAllDynamics()
		case <-m.stopChan:
			return
		}
	}
}

// ─── 直播监控 ─────────────────────────────────────────────────────────────────

func (m *BilibiliModule) checkAllLives() {
	concerns, err := db.GetAllConcernsBySite(m.name)
	if err != nil {
		logger.Errorf("[bilibili] 查询关注失败: %v", err)
		return
	}
	// 去重：同一 uid 只查一次 API
	queried := make(map[string]*biliLiveStatus)
	for _, c := range concerns {
		if !strings.Contains(c.ConcernType, "live") {
			continue
		}
		if _, done := queried[c.UID]; !done {
			s := m.queryLiveStatus(c.UID)
			queried[c.UID] = s
			// 回填用户名
			if s != nil && s.Name != "" && c.Name != s.Name {
				_ = db.UpdateConcernName(m.name, c.UID, s.Name)
			}
		}
		st := queried[c.UID]
		if st == nil || !st.IsLiving {
			// 不在直播 → 清除已通知记录
			m.mu.Lock()
			delete(m.liveNotified, c.UID)
			m.mu.Unlock()
			continue
		}
		// 防重复：同一开播时间段只推送一次
		m.mu.Lock()
		lastTs := m.liveNotified[c.UID]
		m.mu.Unlock()
		if lastTs == st.LiveStartTS {
			continue
		}
		m.mu.Lock()
		m.liveNotified[c.UID] = st.LiveStartTS
		m.mu.Unlock()

		name := c.Name
		if name == "" {
			name = st.Name
		}
		m.sendLiveNotify(c.GroupCode, c.UID, name, st)
	}
}

type biliLiveStatus struct {
	RoomID      int64
	UID         int64
	Name        string
	Title       string
	AreaName    string
	IsLiving    bool
	LiveStartTS int64
	CoverURL    string
}

func (m *BilibiliModule) queryLiveStatus(uid string) *biliLiveStatus {
	// Step 1: 查用户直播间信息
	resp, err := requests.Get(fmt.Sprintf(
		"https://api.live.bilibili.com/live_user/v1/Master/info?uid=%s", uid))
	if err != nil {
		logger.Debugf("[bilibili] 获取用户直播信息失败 uid=%s: %v", uid, err)
		return nil
	}
	data, ok := resp.JSON()
	if !ok || data.Get("code").ToInt() != 0 {
		return nil
	}
	roomID := data.Get("data").Get("room_id").ToInt()
	uname := data.Get("data").Get("info").Get("uname").ToString()
	if roomID == 0 {
		return &biliLiveStatus{Name: uname, IsLiving: false}
	}

	// Step 2: 查直播间详情
	resp2, err := requests.Get(fmt.Sprintf(
		"https://api.live.bilibili.com/xlive/web-room/v1/index/getInfoByRoom?room_id=%d", roomID))
	if err != nil {
		return nil
	}
	d2, ok := resp2.JSON()
	if !ok || d2.Get("code").ToInt() != 0 {
		return nil
	}
	liveStatus := d2.Get("data").Get("room_info").Get("live_status").ToInt()
	startTS := d2.Get("data").Get("room_info").Get("live_start_time").ToInt()

	return &biliLiveStatus{
		RoomID:      roomID,
		Name:        uname,
		Title:       d2.Get("data").Get("room_info").Get("title").ToString(),
		AreaName:    d2.Get("data").Get("room_info").Get("area_name").ToString(),
		IsLiving:    liveStatus == 1,
		LiveStartTS: startTS,
		CoverURL:    d2.Get("data").Get("room_info").Get("cover").ToString(),
	}
}

func (m *BilibiliModule) sendLiveNotify(groupCode int64, uid, name string, s *biliLiveStatus) {
	text := fmt.Sprintf("🔴 %s 开播啦！\n📺 %s\n🏷️ %s\n🔗 https://live.bilibili.com/%d",
		name, s.Title, s.AreaName, s.RoomID)
	logger.Infof("[bilibili] 开播通知 → 群%d: %s 《%s》", groupCode, name, s.Title)
	SendNotify(m.bot, m.name, uid, groupCode, NotifyTypeLive, text, s.CoverURL)
}

// ─── 动态监控 ─────────────────────────────────────────────────────────────────

func (m *BilibiliModule) checkAllDynamics() {
	concerns, err := db.GetAllConcernsBySite(m.name)
	if err != nil {
		logger.Errorf("[bilibili] 查询关注失败: %v", err)
		return
	}
	// 同 uid 只查一次
	queried := make(map[string]bool)
	for _, c := range concerns {
		if !strings.Contains(c.ConcernType, "news") {
			continue
		}
		if queried[c.UID] {
			continue
		}
		queried[c.UID] = true
		m.checkDynamic(c.GroupCode, c.UID, c.Name)
	}
}

func (m *BilibiliModule) checkDynamic(groupCode int64, uid, name string) {
	// 使用 x/polymer API（新动态接口）
	resp, err := requests.Get(fmt.Sprintf(
		"https://api.bilibili.com/x/polymer/web-dynamic/v1/feed/space?host_mid=%s&timezone_offset=-480", uid))
	if err != nil {
		return
	}
	data, ok := resp.JSON()
	if !ok || data.Get("code").ToInt() != 0 {
		return
	}
	items := data.Get("data").Get("items").ToArray()
	if len(items) == 0 {
		return
	}
	latestItem := items[0]
	dynIDStr := latestItem.Get("id_str").ToString()
	if dynIDStr == "" {
		return
	}

	// 直接用 id_str 字符串做去重（B站动态 ID 是单调递增的字符串，无需转数字）
	m.mu.Lock()
	cachedID := m.dynamicLatest[uid]
	if dynIDStr == cachedID {
		m.mu.Unlock()
		return
	}
	isFirst := cachedID == "" // 首次拉取只缓存，不推送，避免历史刷屏
	m.dynamicLatest[uid] = dynIDStr
	m.mu.Unlock()

	if isFirst {
		return
	}

	// 提取动态文字内容
	desc := latestItem.Get("modules").Get("module_dynamic").Get("desc")
	text := ""
	if desc.ToString() != "" {
		text = desc.Get("text").ToString()
	}
	dynURL := fmt.Sprintf("https://t.bilibili.com/%s", dynIDStr)
	if text != "" && len(text) > 100 {
		text = text[:100] + "..."
	}
	dispName := name
	if dispName == "" {
		dispName = uid
	}
	var msg string
	if text != "" {
		msg = fmt.Sprintf("📢 %s 发布了新动态\n%s\n🔗 %s", dispName, text, dynURL)
	} else {
		msg = fmt.Sprintf("📢 %s 发布了新动态\n🔗 %s", dispName, dynURL)
	}

	logger.Infof("[bilibili] 动态通知 → 群%d: %s", groupCode, dispName)
	SendNotify(m.bot, m.name, uid, groupCode, NotifyTypeNews, msg, "")
}

// ─── 番剧监控 ─────────────────────────────────────────────────────────────────
// 监控用户追番列表中最新更新的番剧集（concern_type 含 "bangumi"）

func (m *BilibiliModule) monitorBangumi() {
	defer m.wg.Done()
	t := time.NewTicker(10 * time.Minute) // 番剧更新频率低，10分钟轮询一次
	defer t.Stop()
	for {
		select {
		case <-t.C:
			m.checkAllBangumi()
		case <-m.stopChan:
			return
		}
	}
}

func (m *BilibiliModule) checkAllBangumi() {
	concerns, err := db.GetAllConcernsBySite(m.name)
	if err != nil {
		return
	}
	queried := make(map[string]bool)
	for _, c := range concerns {
		if !strings.Contains(c.ConcernType, "bangumi") {
			continue
		}
		if queried[c.UID] {
			continue
		}
		queried[c.UID] = true
		m.checkBangumi(c.GroupCode, c.UID, c.Name)
	}
}

// checkBangumi 查询用户追番列表，检测最新更新集
// 使用公开 API：https://api.bilibili.com/x/space/bangumi/follow/list
func (m *BilibiliModule) checkBangumi(groupCode int64, uid, name string) {
	resp, err := requests.Get(fmt.Sprintf(
		"https://api.bilibili.com/x/space/bangumi/follow/list?vmid=%s&pn=1&ps=1&type=1", uid))
	if err != nil {
		return
	}
	data, ok := resp.JSON()
	if !ok || data.Get("code").ToInt() != 0 {
		return
	}
	list := data.Get("data").Get("list").ToArray()
	if len(list) == 0 {
		return
	}
	item := list[0]
	ssid := item.Get("season_id").ToString()
	if ssid == "" {
		return
	}
	// 获取最新集 ID（用 new_ep.id 作为去重键）
	newEpID := item.Get("new_ep").Get("id").ToString()
	if newEpID == "" {
		newEpID = ssid
	}

	m.mu.Lock()
	cached := m.bangumiFeed[uid]
	if newEpID == cached {
		m.mu.Unlock()
		return
	}
	isFirst := cached == ""
	m.bangumiFeed[uid] = newEpID
	m.mu.Unlock()

	if isFirst {
		return
	}

	title := item.Get("title").ToString()
	newEpTitle := item.Get("new_ep").Get("index_show").ToString() // "第12话" 等
	coverURL := item.Get("cover").ToString()
	url := fmt.Sprintf("https://www.bilibili.com/bangumi/play/ss%s", ssid)

	dispName := name
	if dispName == "" {
		dispName = uid
	}

	var msg string
	if newEpTitle != "" {
		msg = fmt.Sprintf("🎬 %s 追番更新啦！\n📺 %s 更新了%s\n🔗 %s", dispName, title, newEpTitle, url)
	} else {
		msg = fmt.Sprintf("🎬 %s 追番更新啦！\n📺 %s\n🔗 %s", dispName, title, url)
	}

	logger.Infof("[bilibili] 番剧通知 → 群%d: %s 《%s》", groupCode, dispName, title)
	SendNotify(m.bot, m.name, uid, groupCode, NotifyTypeNews, msg, coverURL)
}

// ─── 专栏监控 ─────────────────────────────────────────────────────────────────
// 监控 UP 主发布的专栏文章（concern_type 含 "article"）

func (m *BilibiliModule) monitorArticle() {
	defer m.wg.Done()
	t := time.NewTicker(5 * time.Minute) // 专栏更新，5分钟轮询
	defer t.Stop()
	for {
		select {
		case <-t.C:
			m.checkAllArticles()
		case <-m.stopChan:
			return
		}
	}
}

func (m *BilibiliModule) checkAllArticles() {
	concerns, err := db.GetAllConcernsBySite(m.name)
	if err != nil {
		return
	}
	queried := make(map[string]bool)
	for _, c := range concerns {
		if !strings.Contains(c.ConcernType, "article") {
			continue
		}
		if queried[c.UID] {
			continue
		}
		queried[c.UID] = true
		m.checkArticle(c.GroupCode, c.UID, c.Name)
	}
}

// checkArticle 查询 UP 主的专栏列表，检测最新文章
// 使用公开 API：https://api.bilibili.com/x/space/article
func (m *BilibiliModule) checkArticle(groupCode int64, uid, name string) {
	resp, err := requests.Get(fmt.Sprintf(
		"https://api.bilibili.com/x/space/article?mid=%s&pn=1&ps=1&sort=publish_time", uid))
	if err != nil {
		return
	}
	data, ok := resp.JSON()
	if !ok || data.Get("code").ToInt() != 0 {
		return
	}
	articles := data.Get("data").Get("articles").ToArray()
	if len(articles) == 0 {
		return
	}
	art := articles[0]
	artID := art.Get("id").ToString()
	if artID == "" {
		return
	}

	m.mu.Lock()
	cached := m.articleLatest[uid]
	if artID == cached {
		m.mu.Unlock()
		return
	}
	isFirst := cached == ""
	m.articleLatest[uid] = artID
	m.mu.Unlock()

	if isFirst {
		return
	}

	title := art.Get("title").ToString()
	summary := art.Get("summary").ToString()
	coverURL := ""
	// 封面可能在 image_urls 数组
	imgList := art.Get("image_urls").ToArray()
	if len(imgList) > 0 {
		coverURL = imgList[0].ToString()
	}
	url := fmt.Sprintf("https://www.bilibili.com/read/cv%s", artID)

	dispName := name
	if dispName == "" {
		dispName = uid
	}

	var msg string
	if summary != "" {
		if len(summary) > 80 {
			summary = summary[:80] + "..."
		}
		msg = fmt.Sprintf("📝 %s 发布了新专栏\n《%s》\n%s\n🔗 %s", dispName, title, summary, url)
	} else {
		msg = fmt.Sprintf("📝 %s 发布了新专栏\n《%s》\n🔗 %s", dispName, title, url)
	}

	logger.Infof("[bilibili] 专栏通知 → 群%d: %s 《%s》", groupCode, dispName, title)
	SendNotify(m.bot, m.name, uid, groupCode, NotifyTypeNews, msg, coverURL)
}
