package module

import (
	"sync"
	"time"

	"github.com/cnxysoft/DDBOT-WSa/internal/db"
	"github.com/cnxysoft/DDBOT-WSa/internal/logger"
	"github.com/cnxysoft/DDBOT-WSa/internal/onebot"
)

// DouyinModule 抖音平台模块
//
// ⚠️  当前状态：抖音接口有严格反爬机制（设备指纹 + 签名 + Cookie），
//
//	无法通过普通 HTTP 请求获取数据。
//	若需支持抖音，请接入第三方代理 API 或自行实现签名逆向。
type DouyinModule struct {
	name      string
	version   string
	status    Status
	bot       *onebot.Bot
	stopChan  chan struct{}
	wg        sync.WaitGroup
	workCache map[string]string // uid → 最新作品 ID
}

func NewDouyinModule() *DouyinModule {
	return &DouyinModule{
		name:      "douyin",
		version:   "2.0.0",
		status:    StatusStopped,
		workCache: make(map[string]string),
	}
}

func (m *DouyinModule) Name() string           { return m.name }
func (m *DouyinModule) Version() string        { return m.version }
func (m *DouyinModule) Status() Status         { return m.status }
func (m *DouyinModule) SetBot(bot *onebot.Bot) { m.bot = bot }

func (m *DouyinModule) Start() error {
	logger.Warnf("[douyin] ⚠️  抖音模块已启动，但当前为占位实现，无法实际推送。")
	logger.Warnf("[douyin]    原因：抖音接口需要设备指纹签名，普通 HTTP 请求会被拒绝。")
	logger.Warnf("[douyin]    如需支持，请接入第三方抖音 API 并在 updateUserWorks() 中实现。")
	m.stopChan = make(chan struct{})
	m.status = StatusRunning
	m.wg.Add(1)
	go m.monitorWorks()
	return nil
}

func (m *DouyinModule) Stop() {
	if m.stopChan != nil {
		close(m.stopChan)
		m.wg.Wait()
		m.stopChan = nil
	}
	m.status = StatusStopped
	logger.Infof("停止抖音模块")
}

func (m *DouyinModule) Reload() error {
	m.Stop()
	return m.Start()
}

func (m *DouyinModule) monitorWorks() {
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

func (m *DouyinModule) checkAllUsers() {
	concerns, err := db.GetAllConcernsBySite(m.name)
	if err != nil {
		logger.Errorf("[douyin] 查询关注失败: %v", err)
		return
	}
	for _, c := range concerns {
		m.updateUserWorks(c.GroupCode, c.UID, c.Name)
	}
}

// updateUserWorks 检测抖音用户新作品
//
// TODO: 抖音接口需要以下步骤才能正常访问：
//  1. 生成合法的 device_id / iid（设备指纹）
//  2. 使用 X-Bogus / _signature 签名算法对请求 URL 签名
//  3. 维护有效的登录 Cookie（ms_token / ttwid 等）
//
// 参考接口（需签名）：
//
//	https://www.douyin.com/aweme/v1/web/aweme/post/?sec_user_id={uid}&count=1
//
// 如需接入，可考虑使用以下方案：
//   - 自建签名服务（逆向 JS 签名代码）
//   - 使用第三方抖音数据 API（需付费或自建）
func (m *DouyinModule) updateUserWorks(groupCode int64, uid, name string) {
	// 当前为占位实现，不执行任何操作
	_ = m.bot
	_ = groupCode
	_ = uid
	_ = name
}
