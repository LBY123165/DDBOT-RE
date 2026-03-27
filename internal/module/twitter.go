package module

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/cnxysoft/DDBOT-WSa/internal/db"
	"github.com/cnxysoft/DDBOT-WSa/internal/logger"
	"github.com/cnxysoft/DDBOT-WSa/internal/onebot"
)

// ─── 常量和包级变量 ────────────────────────────────────────────────────────────

var (
	twitterBaseURLs  = []string{"https://lightbrd.com/", "https://nitter.net/"}
	twitterUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36 Edg/135.0.0.0"
	twitterCookieJar *cookiejar.Jar
)

// ─── 数据结构 ──────────────────────────────────────────────────────────────────

type twitterTweet struct {
	ID        string        `json:"id"`
	Content   string        `json:"content"`
	CreatedAt time.Time     `json:"created_at"`
	IsRetweet bool          `json:"is_retweet"`
	Media     []twMedia     `json:"media"`
	Url       string        `json:"url"`
	OrgUser   *twUser       `json:"org_user,omitempty"`
	Quote     *twitterTweet `json:"quote,omitempty"`
}

type twMedia struct {
	Type string `json:"type"` // "image","gif","video","video(m3u8)"
	Url  string `json:"url"`
}

type twUser struct {
	ScreenName string `json:"screen_name"`
	Name       string `json:"name"`
}

// ─── Anubis 反爬结构 ──────────────────────────────────────────────────────────

type twAnubisChallenge struct {
	Rules struct {
		Algorithm  string `json:"algorithm"`
		Difficulty int    `json:"difficulty"`
	} `json:"rules"`
	Challenge interface{} `json:"challenge"`
}

type twChallengeSub struct {
	Id         string `json:"id"`
	RandomData string `json:"randomData"`
}

type twChallengeResult struct {
	Hash  string
	Nonce int
	RandT int
	Host  string
	Id    string
}

// ─── TwitterModule ────────────────────────────────────────────────────────────

// TwitterModule Twitter/X 平台模块，通过 Nitter 镜像抓取推文
type TwitterModule struct {
	name    string
	version string
	status  Status
	bot     *onebot.Bot

	stopChan chan struct{}
	wg       sync.WaitGroup

	mu        sync.Mutex
	seenTweet map[string]bool // tweetID -> seen（内存去重）
	isFirst   bool            // 首次拉取标记，不推送历史推文
}

// NewTwitterModule 创建 Twitter 模块
func NewTwitterModule() *TwitterModule {
	jar, _ := cookiejar.New(nil)
	twitterCookieJar = jar
	return &TwitterModule{
		name:      "twitter",
		version:   "1.0.0",
		status:    StatusStopped,
		seenTweet: make(map[string]bool),
		isFirst:   true,
	}
}

func (m *TwitterModule) Name() string           { return m.name }
func (m *TwitterModule) Version() string        { return m.version }
func (m *TwitterModule) Status() Status         { return m.status }
func (m *TwitterModule) SetBot(bot *onebot.Bot) { m.bot = bot }

func (m *TwitterModule) Start() error {
	logger.Infof("启动 twitter 模块 (v%s)，通过 Nitter 镜像监控推文", m.version)
	m.stopChan = make(chan struct{})
	m.status = StatusRunning
	m.isFirst = true
	m.wg.Add(1)
	go m.pollLoop()
	return nil
}

func (m *TwitterModule) Stop() {
	if m.stopChan != nil {
		close(m.stopChan)
		m.wg.Wait()
	}
	m.status = StatusStopped
}

func (m *TwitterModule) Reload() error {
	m.Stop()
	return m.Start()
}

// ─── 轮询主循环 ────────────────────────────────────────────────────────────────

func (m *TwitterModule) pollLoop() {
	defer m.wg.Done()
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	select {
	case <-time.After(5 * time.Second):
	case <-m.stopChan:
		return
	}
	m.checkAll()

	for {
		select {
		case <-ticker.C:
			m.checkAll()
		case <-m.stopChan:
			return
		}
	}
}

func (m *TwitterModule) checkAll() {
	concerns, err := db.GetAllConcernsBySite("twitter")
	if err != nil {
		logger.Errorf("twitter: 获取订阅列表失败: %v", err)
		return
	}
	if len(concerns) == 0 {
		return
	}

	// 按 uid 去重
	type uidEntry struct{ uid string }
	seen := make(map[string]bool)
	var uids []string
	for _, c := range concerns {
		if !seen[c.UID] {
			seen[c.UID] = true
			uids = append(uids, c.UID)
		}
	}

	first := m.isFirst
	if first {
		m.isFirst = false
	}

	for _, uid := range uids {
		select {
		case <-m.stopChan:
			return
		default:
		}
		tweets, err := m.fetchTweets(uid)
		if err != nil {
			logger.Warnf("twitter: 抓取 @%s 推文失败: %v", uid, err)
			continue
		}

		// 首次拉取：仅标记，不推送
		if first {
			m.mu.Lock()
			for _, tw := range tweets {
				m.seenTweet[tw.ID] = true
			}
			m.mu.Unlock()
			continue
		}

		for _, tw := range tweets {
			m.mu.Lock()
			alreadySeen := m.seenTweet[tw.ID]
			if !alreadySeen {
				m.seenTweet[tw.ID] = true
			}
			m.mu.Unlock()
			if alreadySeen {
				continue
			}

			// 回填用户名到 DB（尽力而为）
			_ = db.UpdateConcernName("twitter", uid, uid)

			msg := m.formatTweet(uid, tw)
			for _, c := range concerns {
				if c.UID != uid {
					continue
				}
				SendNotify(m.bot, "twitter", uid, c.GroupCode, NotifyTypeNews, msg, "")
			}
			time.Sleep(500 * time.Millisecond)
		}
		// 每个账号之间稍作等待，避免请求过于密集
		time.Sleep(time.Duration(rand.Intn(3)+1) * time.Second)
	}
}

// ─── 格式化推文消息 ────────────────────────────────────────────────────────────

func (m *TwitterModule) formatTweet(uid string, tw *twitterTweet) string {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	if loc == nil {
		loc = time.Local
	}
	createdAt := tw.CreatedAt.In(loc).Format("2006-01-02 15:04:05")

	var sb strings.Builder
	if tw.IsRetweet && tw.OrgUser != nil {
		sb.WriteString(fmt.Sprintf("🔁 X-%s 转发了 %s 的推文：\n", uid, tw.OrgUser.Name))
	} else {
		sb.WriteString(fmt.Sprintf("🐦 X-%s 发布了新推文：\n", uid))
	}
	sb.WriteString(createdAt + "\n")
	if tw.Content != "" {
		sb.WriteString(tw.Content + "\n")
	}
	for _, md := range tw.Media {
		switch md.Type {
		case "image":
			sb.WriteString(fmt.Sprintf("[图片] %s\n", md.Url))
		case "gif":
			sb.WriteString(fmt.Sprintf("[GIF] %s\n", md.Url))
		case "video", "video(m3u8)":
			sb.WriteString(fmt.Sprintf("[视频] %s\n", md.Url))
		}
	}
	if tw.Quote != nil && tw.Quote.OrgUser != nil {
		sb.WriteString(fmt.Sprintf("└ 引用 @%s: %s\n", tw.Quote.OrgUser.ScreenName, tw.Quote.Content))
	}
	if tw.Url != "" {
		sb.WriteString(tw.Url + "\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

// ─── 抓取推文（Nitter 镜像 HTML 解析） ────────────────────────────────────────

func (m *TwitterModule) fetchTweets(screenName string) ([]*twitterTweet, error) {
	mirrorBase := twitterBaseURLs[rand.Intn(len(twitterBaseURLs))]
	mirrorURL := mirrorBase + screenName

	body, err := m.doGet(mirrorURL)
	if err != nil {
		return nil, err
	}
	return parseTweetsFromHTML(body, mirrorURL)
}

// doGet 执行 HTTP GET
func (m *TwitterModule) doGet(rawURL string) ([]byte, error) {
	client := &http.Client{
		Timeout: 15 * time.Second,
		Jar:     twitterCookieJar,
	}
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", twitterUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 503 {
		return nil, fmt.Errorf("503 Service Unavailable")
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("http %d", resp.StatusCode)
	}
	var buf bytes.Buffer
	_, err = buf.ReadFrom(resp.Body)
	return buf.Bytes(), err
}

// ─── HTML 解析（纯正则，无 goquery 依赖） ──────────────────────────────────────

var (
	reTweetLink    = regexp.MustCompile(`class="tweet-link"\s+href="([^"]+)"`)
	reTweetContent = regexp.MustCompile(`class="tweet-content[^"]*"[^>]*>([\s\S]*?)</div>`)
	reTweetDate    = regexp.MustCompile(`class="tweet-date"[\s\S]*?title="([^"]+)"`)
	reRetweetHdr   = regexp.MustCompile(`class="retweet-header"`)
	reImgSrc       = regexp.MustCompile(`class="attachment[^"]*"[\s\S]*?<img[^>]+src="([^"]+)"`)
	reGifSrc       = regexp.MustCompile(`class="gif"[\s\S]*?<source[^>]+src="([^"]+)"`)
	reVideoSrc     = regexp.MustCompile(`class="gallery-video"[\s\S]*?<source[^>]+src="([^"]+)"`)
	reM3u8         = regexp.MustCompile(`class="gallery-video"[\s\S]*?<video[^>]+data-url="([^"]+)"`)
	rePinnedItem   = regexp.MustCompile(`class="pinned"`)
	reTimelineItem = regexp.MustCompile(`(?s)class="timeline-item"([\s\S]*?)(?:class="timeline-item"|</main>)`)
	reTweetID      = regexp.MustCompile(`/status/(\d+)`)
	reHTMLTag      = regexp.MustCompile(`<[^>]+>`)
	reOgTitle      = regexp.MustCompile(`<meta\s+property="og:title"\s+content="([^"]+)"`)
	reAnubisScript = regexp.MustCompile(`id="anubis_challenge"[^>]*>([\s\S]*?)</script>`)
	reTitle        = regexp.MustCompile(`<title>([^<]+)</title>`)
)

func parseTweetsFromHTML(htmlContent []byte, rawURL string) ([]*twitterTweet, error) {
	html := string(htmlContent)

	// 检查 title
	titleMatch := reTitle.FindStringSubmatch(html)
	if len(titleMatch) >= 2 {
		title := titleMatch[1]
		if strings.Contains(title, "Just a moment") {
			return nil, fmt.Errorf("触发 CF 验证")
		}
		if strings.HasPrefix(title, "Error") || strings.Contains(title, "正在确认") {
			// 尝试处理 Anubis
			if m := reAnubisScript.FindStringSubmatch(html); len(m) >= 2 {
				solveAndApplyAnubis(m[1], rawURL)
			}
			return nil, fmt.Errorf("反爬拦截，已尝试处理")
		}
	}

	// 从 og:title 获取账号 screenName
	parsedURL, _ := url.Parse(rawURL)
	screenName := strings.Trim(parsedURL.Path, "/")

	// 按 timeline-item 分割（用 <!-- --> 或自定义分隔符）
	// 改用 FindAllIndex 找到每个 timeline-item div 的起始位置
	type itemRange struct{ start, end int }
	var items []itemRange
	{
		reStart := regexp.MustCompile(`class="timeline-item`)
		allStarts := reStart.FindAllStringIndex(html, -1)
		for i, st := range allStarts {
			var end int
			if i+1 < len(allStarts) {
				end = allStarts[i+1][0]
			} else {
				end = len(html)
			}
			items = append(items, itemRange{st[0], end})
		}
	}

	var tweets []*twitterTweet
	for _, item := range items {
		chunk := html[item.start:item.end]
		tw := parseSingleTweet(chunk, screenName, parsedURL.Host)
		if tw != nil && tw.ID != "" {
			tweets = append(tweets, tw)
		}
	}
	return tweets, nil
}

func parseSingleTweet(chunk, screenName, host string) *twitterTweet {
	tw := &twitterTweet{}

	// ID
	if m := reTweetLink.FindStringSubmatch(chunk); len(m) >= 2 {
		tw.ID = extractTweetIDStr(m[1])
	}
	if tw.ID == "" {
		return nil
	}

	// 内容（去除 HTML 标签）
	if m := reTweetContent.FindStringSubmatch(chunk); len(m) >= 2 {
		tw.Content = strings.TrimSpace(reHTMLTag.ReplaceAllString(m[1], ""))
	}

	// 时间
	tw.CreatedAt = time.Now()
	if m := reTweetDate.FindStringSubmatch(chunk); len(m) >= 2 {
		// 格式：Jan 2, 2006 · 3:04 PM UTC
		if t, err := time.Parse("Jan 2, 2006 · 3:04 PM MST", m[1]); err == nil {
			tw.CreatedAt = t
		}
	}

	// 转推
	tw.IsRetweet = reRetweetHdr.MatchString(chunk)

	// 图片
	for _, m := range reImgSrc.FindAllStringSubmatch(chunk, -1) {
		if len(m) >= 2 {
			tw.Media = append(tw.Media, twMedia{Type: "image", Url: fixTwitterMediaURL(m[1], host)})
		}
	}
	// GIF
	for _, m := range reGifSrc.FindAllStringSubmatch(chunk, -1) {
		if len(m) >= 2 {
			tw.Media = append(tw.Media, twMedia{Type: "gif", Url: fixTwitterMediaURL(m[1], host)})
		}
	}
	// 视频
	for _, m := range reVideoSrc.FindAllStringSubmatch(chunk, -1) {
		if len(m) >= 2 {
			tw.Media = append(tw.Media, twMedia{Type: "video", Url: fixTwitterMediaURL(m[1], host)})
		}
	}
	// m3u8
	for _, m := range reM3u8.FindAllStringSubmatch(chunk, -1) {
		if len(m) >= 2 {
			tw.Media = append(tw.Media, twMedia{Type: "video(m3u8)", Url: fixTwitterMediaURL(m[1], host)})
		}
	}

	// URL
	if tw.IsRetweet {
		if m := reTweetLink.FindStringSubmatch(chunk); len(m) >= 2 {
			tw.Url = "https://x.com" + strings.TrimRight(m[1], "#m")
		}
	} else {
		tw.Url = fmt.Sprintf("https://x.com/%s/status/%s", screenName, tw.ID)
	}

	return tw
}

func extractTweetIDStr(href string) string {
	m := reTweetID.FindStringSubmatch(href)
	if len(m) >= 2 {
		return m[1]
	}
	return ""
}

func fixTwitterMediaURL(mediaURL, host string) string {
	if strings.HasPrefix(mediaURL, "http://") || strings.HasPrefix(mediaURL, "https://") {
		return mediaURL
	}
	if strings.HasPrefix(mediaURL, "/pic/") {
		tail := strings.TrimPrefix(mediaURL, "/pic/")
		if decoded, err := url.PathUnescape(tail); err == nil {
			return "https://pbs.twimg.com/" + strings.TrimPrefix(decoded, "/")
		}
	}
	if strings.HasPrefix(mediaURL, "/") && host != "" {
		return "https://" + host + mediaURL
	}
	return mediaURL
}

// ─── Anubis PoW 反爬处理 ──────────────────────────────────────────────────────

func solveAndApplyAnubis(challengeJSON, rawURL string) {
	var ch twAnubisChallenge
	if err := json.Unmarshal([]byte(challengeJSON), &ch); err != nil {
		return
	}
	var challengeStr string
	var subID string
	switch v := ch.Challenge.(type) {
	case string:
		challengeStr = v
	case map[string]interface{}:
		b, _ := json.Marshal(v)
		var sub twChallengeSub
		if err := json.Unmarshal(b, &sub); err == nil {
			challengeStr = sub.RandomData
			subID = sub.Id
		}
	}
	if challengeStr == "" {
		return
	}
	nonce, hash := computeTwitterPoW(challengeStr, ch.Rules.Difficulty)
	parsedURL, _ := url.Parse(rawURL)
	result := &twChallengeResult{
		Hash:  hash,
		Nonce: nonce,
		RandT: rand.Intn(100),
		Host:  parsedURL.Hostname(),
		Id:    subID,
	}
	applyAnubisCookieToJar(result)
}

func applyAnubisCookieToJar(result *twChallengeResult) {
	if twitterCookieJar == nil || result == nil {
		return
	}
	addID := ""
	if result.Id != "" {
		addID = "id=" + result.Id + "&"
	}
	passURL := fmt.Sprintf(
		"https://%s/.within.website/x/cmd/anubis/api/pass-challenge?%sresponse=%s&nonce=%d&redir=https://%s/&elapsedTime=%d",
		result.Host, addID, result.Hash, result.Nonce, url.QueryEscape(result.Host), result.RandT,
	)
	client := &http.Client{Timeout: 10 * time.Second, Jar: twitterCookieJar}
	req, _ := http.NewRequest("GET", passURL, nil)
	if req != nil {
		req.Header.Set("User-Agent", twitterUserAgent)
		_, _ = client.Do(req)
	}
	logger.Debugf("twitter: 已尝试刷新 anubis cookie, host=%s", result.Host)
}

func computeTwitterPoW(challenge string, difficulty int) (nonce int, hash string) {
	prefix := strings.Repeat("0", difficulty)
	for {
		data := challenge + fmt.Sprintf("%d", nonce)
		sum := sha256.Sum256([]byte(data))
		hash = hex.EncodeToString(sum[:])
		if strings.HasPrefix(hash, prefix) {
			return nonce, hash
		}
		nonce++
	}
}
