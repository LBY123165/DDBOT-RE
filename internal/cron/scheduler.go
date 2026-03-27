// Package cron 提供轻量定时任务调度器，不依赖第三方 cron 库。
// cron 表达式格式（5段）：分 时 日 月 周
// 例："0 9 * * 1" = 每周一上午9:00
//     "30 12 * * *" = 每天中午12:30
//     "0 8,20 * * *" = 每天8:00和20:00
package cron

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/cnxysoft/DDBOT-WSa/internal/db"
	"github.com/cnxysoft/DDBOT-WSa/internal/logger"
	"github.com/cnxysoft/DDBOT-WSa/internal/onebot"
)

// Scheduler 定时任务调度器
type Scheduler struct {
	bot      *onebot.Bot
	mu       sync.Mutex
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// NewScheduler 创建调度器
func NewScheduler() *Scheduler {
	return &Scheduler{}
}

// SetBot 注入 Bot 实例
func (s *Scheduler) SetBot(bot *onebot.Bot) {
	s.mu.Lock()
	s.bot = bot
	s.mu.Unlock()
}

// Start 启动调度器（每分钟检查一次到期任务）
func (s *Scheduler) Start() {
	s.stopChan = make(chan struct{})
	s.wg.Add(1)
	go s.loop()
	logger.Info("[cron] 定时任务调度器已启动")
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	if s.stopChan != nil {
		close(s.stopChan)
		s.wg.Wait()
		s.stopChan = nil
	}
	logger.Info("[cron] 定时任务调度器已停止")
}

func (s *Scheduler) loop() {
	defer s.wg.Done()

	// 对齐到下一整分钟再开始，保证时间点准确
	now := time.Now()
	nextMin := now.Truncate(time.Minute).Add(time.Minute)
	select {
	case <-time.After(time.Until(nextMin)):
	case <-s.stopChan:
		return
	}

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	// 首次触发
	s.tick()

	for {
		select {
		case <-ticker.C:
			s.tick()
		case <-s.stopChan:
			return
		}
	}
}

// tick 每分钟执行一次，检查所有到期任务
func (s *Scheduler) tick() {
	now := time.Now()
	jobs, err := db.GetAllEnabledCronJobs()
	if err != nil {
		logger.Errorf("[cron] 读取定时任务失败: %v", err)
		return
	}
	for _, job := range jobs {
		if matchCron(job.CronExpr, now) {
			go s.runJob(job, now)
		}
	}
}

// runJob 执行单个任务：渲染模板 → 发送群消息 → 更新 last_run
func (s *Scheduler) runJob(job *db.CronJob, now time.Time) {
	s.mu.Lock()
	bot := s.bot
	s.mu.Unlock()

	if bot == nil {
		logger.Warnf("[cron] bot 未就绪，跳过任务 %s", job.Name)
		return
	}

	msg, err := renderTemplate(job.Template, now)
	if err != nil {
		logger.Errorf("[cron] 渲染模板失败 job=%s: %v", job.Name, err)
		return
	}

	if err := bot.SendGroupText(job.GroupCode, msg); err != nil {
		logger.Errorf("[cron] 发送消息失败 job=%s group=%d: %v", job.Name, job.GroupCode, err)
		return
	}

	logger.Infof("[cron] 执行任务 %s → 群%d", job.Name, job.GroupCode)
	_ = db.UpdateCronJobLastRun(job.ID)
}

// ─── 模板渲染 ─────────────────────────────────────────────────────────────────

// TemplateData 消息模板可用变量
type TemplateData struct {
	Now       time.Time // 当前时间
	Date      string    // 日期 "2006-01-02"
	Time      string    // 时间 "15:04"
	Weekday   string    // 星期 "Monday"
	WeekdayCN string    // 星期（中文）"星期一"
	Year      int
	Month     int
	Day       int
	Hour      int
	Minute    int
}

var weekdayCN = map[time.Weekday]string{
	time.Monday: "星期一", time.Tuesday: "星期二", time.Wednesday: "星期三",
	time.Thursday: "星期四", time.Friday: "星期五", time.Saturday: "星期六",
	time.Sunday: "星期日",
}

func renderTemplate(tmplStr string, now time.Time) (string, error) {
	data := TemplateData{
		Now:       now,
		Date:      now.Format("2006-01-02"),
		Time:      now.Format("15:04"),
		Weekday:   now.Weekday().String(),
		WeekdayCN: weekdayCN[now.Weekday()],
		Year:      now.Year(),
		Month:     int(now.Month()),
		Day:       now.Day(),
		Hour:      now.Hour(),
		Minute:    now.Minute(),
	}
	tmpl, err := template.New("msg").Parse(tmplStr)
	if err != nil {
		return tmplStr, nil // 解析失败时原样发送
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return tmplStr, nil
	}
	return buf.String(), nil
}

// ─── cron 表达式解析 ──────────────────────────────────────────────────────────
// 支持：数字、*、*/n、逗号列表（如 8,20）
// 格式：分(0-59) 时(0-23) 日(1-31) 月(1-12) 周(0-7，0和7均=周日)

func matchCron(expr string, t time.Time) bool {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return false
	}
	checks := []struct {
		field string
		value int
	}{
		{fields[0], t.Minute()},
		{fields[1], t.Hour()},
		{fields[2], t.Day()},
		{fields[3], int(t.Month())},
		{fields[4], int(t.Weekday())},
	}
	for _, c := range checks {
		if !matchField(c.field, c.value) {
			return false
		}
	}
	return true
}

func matchField(field string, value int) bool {
	if field == "*" {
		return true
	}
	// */n 步进
	if strings.HasPrefix(field, "*/") {
		n, err := strconv.Atoi(field[2:])
		if err != nil || n <= 0 {
			return false
		}
		return value%n == 0
	}
	// 逗号列表
	for _, part := range strings.Split(field, ",") {
		part = strings.TrimSpace(part)
		if n, err := strconv.Atoi(part); err == nil {
			if n == value {
				return true
			}
			// 周日兼容：0 和 7 都代表周日
			if value == 0 && n == 7 {
				return true
			}
			if value == 7 && n == 0 {
				return true
			}
		}
	}
	return false
}

// ValidateCronExpr 校验 cron 表达式格式
func ValidateCronExpr(expr string) error {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return fmt.Errorf("cron 表达式必须包含5个字段（分 时 日 月 周），当前：%d 个", len(fields))
	}
	limits := [][2]int{{0, 59}, {0, 23}, {1, 31}, {1, 12}, {0, 7}}
	names := []string{"分钟(0-59)", "小时(0-23)", "日(1-31)", "月(1-12)", "星期(0-7)"}
	for i, f := range fields {
		if err := validateField(f, limits[i][0], limits[i][1]); err != nil {
			return fmt.Errorf("第%d段[%s]格式错误: %v", i+1, names[i], err)
		}
	}
	return nil
}

func validateField(field string, min, max int) error {
	if field == "*" {
		return nil
	}
	if strings.HasPrefix(field, "*/") {
		n, err := strconv.Atoi(field[2:])
		if err != nil {
			return fmt.Errorf("无效步进: %s", field)
		}
		if n <= 0 {
			return fmt.Errorf("步进值必须大于0")
		}
		return nil
	}
	for _, part := range strings.Split(field, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return fmt.Errorf("无效值: %s", part)
		}
		if n < min || n > max {
			return fmt.Errorf("值 %d 超出范围 [%d,%d]", n, min, max)
		}
	}
	return nil
}
