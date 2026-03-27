package db

import (
	"fmt"
	"time"

	"github.com/cnxysoft/DDBOT-WSa/internal/logger"
)

// Concern 关注记录
type Concern struct {
	ID          int64
	Site        string
	UID         string
	Name        string
	GroupCode   int64
	ConcernType string // "live" / "news" / "live,news" 等
	Enable      bool
	CreateTime  time.Time
	UpdateTime  time.Time
}

// InsertConcern 添加关注（重复则忽略）
func InsertConcern(site, uid, name string, groupCode int64, concernType string) error {
	if concernType == "" {
		concernType = "live"
	}
	_, err := Exec(`
		INSERT OR IGNORE INTO concerns (site, uid, name, group_code, concern_type, enable)
		VALUES (?, ?, ?, ?, ?, 1)`,
		site, uid, name, groupCode, concernType)
	return err
}

// DeleteConcern 删除关注
func DeleteConcern(site, uid string, groupCode int64) error {
	result, err := Exec(
		`DELETE FROM concerns WHERE site = ? AND uid = ? AND group_code = ?`,
		site, uid, groupCode)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("关注记录不存在: site=%s uid=%s group=%d", site, uid, groupCode)
	}
	return nil
}

// UpdateConcernName 更新关注对象的显示名称
func UpdateConcernName(site, uid, name string) error {
	_, err := Exec(
		`UPDATE concerns SET name = ?, update_time = CURRENT_TIMESTAMP WHERE site = ? AND uid = ?`,
		name, site, uid)
	return err
}

// GetConcerns 查询群组的关注列表（platform 为空时查全部）
func GetConcerns(groupCode int64, platform string) ([]*Concern, error) {
	var (
		query string
		args  []interface{}
	)
	if platform == "" {
		query = `SELECT id, site, uid, name, group_code, concern_type, enable, create_time, update_time
		          FROM concerns WHERE group_code = ? AND enable = 1
		          ORDER BY site, create_time DESC`
		args = []interface{}{groupCode}
	} else {
		query = `SELECT id, site, uid, name, group_code, concern_type, enable, create_time, update_time
		          FROM concerns WHERE group_code = ? AND site = ? AND enable = 1
		          ORDER BY create_time DESC`
		args = []interface{}{groupCode, platform}
	}

	rows, err := Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询关注失败: %w", err)
	}
	defer rows.Close()

	var list []*Concern
	for rows.Next() {
		c := &Concern{}
		var enableInt int
		if err := rows.Scan(&c.ID, &c.Site, &c.UID, &c.Name, &c.GroupCode,
			&c.ConcernType, &enableInt, &c.CreateTime, &c.UpdateTime); err != nil {
			logger.Warnf("扫描关注记录失败: %v", err)
			continue
		}
		c.Enable = enableInt == 1
		list = append(list, c)
	}
	return list, nil
}

// GetAllConcernsBySite 查询指定平台所有启用的关注（供模块轮询用）
func GetAllConcernsBySite(site string) ([]*Concern, error) {
	rows, err := Query(`
		SELECT id, site, uid, name, group_code, concern_type, enable, create_time, update_time
		FROM concerns WHERE site = ? AND enable = 1
		ORDER BY uid, group_code`, site)
	if err != nil {
		return nil, fmt.Errorf("查询关注失败: %w", err)
	}
	defer rows.Close()

	var list []*Concern
	for rows.Next() {
		c := &Concern{}
		var enableInt int
		if err := rows.Scan(&c.ID, &c.Site, &c.UID, &c.Name, &c.GroupCode,
			&c.ConcernType, &enableInt, &c.CreateTime, &c.UpdateTime); err != nil {
			continue
		}
		c.Enable = enableInt == 1
		list = append(list, c)
	}
	return list, nil
}

// GetConcernCountBySite 统计各平台关注数（WebUI用）
func GetConcernCountBySite() (map[string]int, error) {
	rows, err := Query(`SELECT site, COUNT(DISTINCT uid || ':' || group_code) FROM concerns WHERE enable=1 GROUP BY site`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]int)
	for rows.Next() {
		var site string
		var cnt int
		if err := rows.Scan(&site, &cnt); err != nil {
			continue
		}
		result[site] = cnt
	}
	return result, nil
}

// GetConcernsByGroup 获取指定群的所有订阅
func GetConcernsByGroup(groupCode int64) ([]*Concern, error) {
	rows, err := Query(`
		SELECT id, site, uid, name, group_code, concern_type, enable, create_time, update_time
		FROM concerns WHERE group_code = ? AND enable = 1
		ORDER BY site, uid`, groupCode)
	if err != nil {
		return nil, fmt.Errorf("查询群订阅失败: %w", err)
	}
	defer rows.Close()
	var list []*Concern
	for rows.Next() {
		c := &Concern{}
		var enableInt int
		if err := rows.Scan(&c.ID, &c.Site, &c.UID, &c.Name, &c.GroupCode,
			&c.ConcernType, &enableInt, &c.CreateTime, &c.UpdateTime); err != nil {
			continue
		}
		c.Enable = enableInt == 1
		list = append(list, c)
	}
	return list, nil
}

// IsAdmin 检查用户是否是管理员
func IsAdmin(userID int64) (bool, error) {
	row := QueryRow(`SELECT COUNT(*) FROM admins WHERE user_id = ?`, userID)
	var count int
	if err := row.Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

// AddAdmin 添加管理员
func AddAdmin(userID int64, note string) error {
	_, err := Exec(`INSERT OR IGNORE INTO admins (user_id, note) VALUES (?, ?)`, userID, note)
	return err
}

// RemoveAdmin 移除管理员
func RemoveAdmin(userID int64) error {
	_, err := Exec(`DELETE FROM admins WHERE user_id = ?`, userID)
	return err
}

// ListAdmins 列出所有管理员 ID
func ListAdmins() ([]int64, error) {
	rows, err := Query(`SELECT user_id FROM admins ORDER BY create_time`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			continue
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// MigrateFromBuntDB 从旧版 buntdb 数据库迁移数据到 SQLite（占位实现）
// 如需启用，请参考文档自行实现迁移逻辑。
func MigrateFromBuntDB(_ string) error {
	return fmt.Errorf("buntdb 迁移功能暂未实现，如需迁移旧版数据请参考文档")
}

// ─────────────────────────── CronJob ─────────────────────────────────────────

// CronJob 定时任务记录
type CronJob struct {
	ID        int64
	Name      string
	GroupCode int64
	CronExpr  string // cron 表达式（5段，分/时/日/月/周）
	Template  string // 消息模板
	Enabled   bool
	LastRun   time.Time
}

// InsertCronJob 添加定时任务（同群同名则覆盖）
func InsertCronJob(groupCode int64, name, cronExpr, template string) error {
	_, err := Exec(`
		INSERT INTO cron_jobs (group_code, name, cron_expr, template)
		VALUES (?,?,?,?)
		ON CONFLICT(group_code, name) DO UPDATE SET
			cron_expr=excluded.cron_expr, template=excluded.template,
			enabled=1, update_time=CURRENT_TIMESTAMP`,
		groupCode, name, cronExpr, template)
	return err
}

// DeleteCronJob 删除定时任务
func DeleteCronJob(groupCode int64, name string) error {
	result, err := Exec(`DELETE FROM cron_jobs WHERE group_code=? AND name=?`, groupCode, name)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("定时任务不存在: group=%d name=%s", groupCode, name)
	}
	return nil
}

// ListCronJobs 列出群内所有定时任务
func ListCronJobs(groupCode int64) ([]*CronJob, error) {
	rows, err := Query(`
		SELECT id, name, group_code, cron_expr, template, enabled, COALESCE(last_run,'')
		FROM cron_jobs WHERE group_code=? ORDER BY name`, groupCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*CronJob
	for rows.Next() {
		j := &CronJob{}
		var enabledInt int
		var lastRunStr string
		if err := rows.Scan(&j.ID, &j.Name, &j.GroupCode, &j.CronExpr, &j.Template, &enabledInt, &lastRunStr); err != nil {
			continue
		}
		j.Enabled = enabledInt == 1
		if lastRunStr != "" {
			j.LastRun, _ = time.Parse("2006-01-02 15:04:05", lastRunStr)
		}
		list = append(list, j)
	}
	return list, nil
}

// GetAllEnabledCronJobs 获取所有启用的定时任务（调度器用）
func GetAllEnabledCronJobs() ([]*CronJob, error) {
	rows, err := Query(`
		SELECT id, name, group_code, cron_expr, template, enabled, COALESCE(last_run,'')
		FROM cron_jobs WHERE enabled=1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*CronJob
	for rows.Next() {
		j := &CronJob{}
		var enabledInt int
		var lastRunStr string
		if err := rows.Scan(&j.ID, &j.Name, &j.GroupCode, &j.CronExpr, &j.Template, &enabledInt, &lastRunStr); err != nil {
			continue
		}
		j.Enabled = enabledInt == 1
		if lastRunStr != "" {
			j.LastRun, _ = time.Parse("2006-01-02 15:04:05", lastRunStr)
		}
		list = append(list, j)
	}
	return list, nil
}

// UpdateCronJobLastRun 更新定时任务最后执行时间
func UpdateCronJobLastRun(id int64) error {
	_, err := Exec(`UPDATE cron_jobs SET last_run=CURRENT_TIMESTAMP, update_time=CURRENT_TIMESTAMP WHERE id=?`, id)
	return err
}

// SetCronJobEnabled 启用/禁用定时任务
func SetCronJobEnabled(groupCode int64, name string, enabled bool) error {
	val := 0
	if enabled {
		val = 1
	}
	result, err := Exec(`UPDATE cron_jobs SET enabled=?, update_time=CURRENT_TIMESTAMP WHERE group_code=? AND name=?`,
		val, groupCode, name)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("定时任务不存在: group=%d name=%s", groupCode, name)
	}
	return nil
}

// ─────────────────────────── ConcernConfig ───────────────────────────────────

// ConcernConfig 群订阅推送配置
type ConcernConfig struct {
	GroupCode  int64
	Site       string
	UID        string
	AtMode     int    // 0=不At, 1=@全体, 2=@指定成员
	AtMembers  string // JSON int64 数组
	NotifyLive bool
	NotifyNews bool
	FilterText string // JSON string 数组（关键词过滤）
}

// GetConcernConfig 获取群推送配置（不存在时返回默认值）
func GetConcernConfig(groupCode int64, site, uid string) (*ConcernConfig, error) {
	row := QueryRow(`
		SELECT group_code, site, uid, at_mode, at_members, notify_live, notify_news, filter_text
		FROM concern_configs WHERE group_code=? AND site=? AND uid=?`,
		groupCode, site, uid)
	cfg := &ConcernConfig{}
	var notifyLive, notifyNews int
	err := row.Scan(&cfg.GroupCode, &cfg.Site, &cfg.UID,
		&cfg.AtMode, &cfg.AtMembers, &notifyLive, &notifyNews, &cfg.FilterText)
	if err != nil {
		// 不存在时返回默认配置
		return &ConcernConfig{
			GroupCode:  groupCode,
			Site:       site,
			UID:        uid,
			NotifyLive: true,
			NotifyNews: true,
		}, nil
	}
	cfg.NotifyLive = notifyLive == 1
	cfg.NotifyNews = notifyNews == 1
	return cfg, nil
}

// SetConcernConfig 写入/更新群推送配置
func SetConcernConfig(cfg *ConcernConfig) error {
	notifyLive, notifyNews := 0, 0
	if cfg.NotifyLive {
		notifyLive = 1
	}
	if cfg.NotifyNews {
		notifyNews = 1
	}
	_, err := Exec(`
		INSERT INTO concern_configs
			(group_code, site, uid, at_mode, at_members, notify_live, notify_news, filter_text, update_time)
		VALUES (?,?,?,?,?,?,?,?,CURRENT_TIMESTAMP)
		ON CONFLICT(group_code, site, uid) DO UPDATE SET
			at_mode=excluded.at_mode, at_members=excluded.at_members,
			notify_live=excluded.notify_live, notify_news=excluded.notify_news,
			filter_text=excluded.filter_text, update_time=CURRENT_TIMESTAMP`,
		cfg.GroupCode, cfg.Site, cfg.UID,
		cfg.AtMode, cfg.AtMembers, notifyLive, notifyNews, cfg.FilterText)
	return err
}

// ─────────────────────────── CommandConfig ───────────────────────────────────

// IsCommandEnabled 检查群内命令是否已启用（默认 true）
func IsCommandEnabled(groupCode int64, command string) (bool, error) {
	row := QueryRow(`SELECT enabled FROM command_configs WHERE group_code=? AND command=?`, groupCode, command)
	var enabled int
	if err := row.Scan(&enabled); err != nil {
		// 不存在 = 默认启用
		return true, nil
	}
	return enabled == 1, nil
}

// SetCommandEnabled 设置群内命令开关
func SetCommandEnabled(groupCode int64, command string, enabled bool) error {
	val := 0
	if enabled {
		val = 1
	}
	_, err := Exec(`
		INSERT INTO command_configs (group_code, command, enabled)
		VALUES (?,?,?)
		ON CONFLICT(group_code, command) DO UPDATE SET enabled=excluded.enabled`,
		groupCode, command, val)
	return err
}

// GetDisabledCommands 获取群内所有已禁用的命令列表
func GetDisabledCommands(groupCode int64) ([]string, error) {
	rows, err := Query(`SELECT command FROM command_configs WHERE group_code=? AND enabled=0`, groupCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cmds []string
	for rows.Next() {
		var cmd string
		if err := rows.Scan(&cmd); err == nil {
			cmds = append(cmds, cmd)
		}
	}
	return cmds, nil
}

// ─────────────────────────── Blocklist ───────────────────────────────────────

// IsBlocked 检查用户或群是否在黑名单中
func IsBlocked(targetType string, targetID int64) (bool, error) {
	row := QueryRow(`SELECT COUNT(*) FROM blocklist WHERE target_type=? AND target_id=?`, targetType, targetID)
	var count int
	if err := row.Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

// AddBlock 将用户/群加入黑名单
func AddBlock(targetType string, targetID int64, reason string) error {
	_, err := Exec(`
		INSERT OR IGNORE INTO blocklist (target_type, target_id, reason)
		VALUES (?,?,?)`, targetType, targetID, reason)
	return err
}

// RemoveBlock 从黑名单移除
func RemoveBlock(targetType string, targetID int64) error {
	_, err := Exec(`DELETE FROM blocklist WHERE target_type=? AND target_id=?`, targetType, targetID)
	return err
}

// ListBlocks 列出黑名单（可按类型过滤）
func ListBlocks(targetType string) ([]struct {
	Type   string
	ID     int64
	Reason string
}, error) {
	var rows_ interface {
		Next() bool
		Scan(...interface{}) error
		Close() error
	}
	var err error
	if targetType == "" {
		rows_, err = Query(`SELECT target_type, target_id, reason FROM blocklist ORDER BY create_time DESC`)
	} else {
		rows_, err = Query(`SELECT target_type, target_id, reason FROM blocklist WHERE target_type=? ORDER BY create_time DESC`, targetType)
	}
	if err != nil {
		return nil, err
	}
	defer rows_.Close()
	type entry struct {
		Type   string
		ID     int64
		Reason string
	}
	var list []entry
	for rows_.Next() {
		var e entry
		if err := rows_.Scan(&e.Type, &e.ID, &e.Reason); err == nil {
			list = append(list, e)
		}
	}
	var result []struct {
		Type   string
		ID     int64
		Reason string
	}
	for _, e := range list {
		result = append(result, struct {
			Type   string
			ID     int64
			Reason string
		}{e.Type, e.ID, e.Reason})
	}
	return result, nil
}

// ─────────────────────────── GroupSilence ────────────────────────────────────

// IsGroupSilenced 检查群是否处于静音状态（BOT 不回复非管理员命令）
func IsGroupSilenced(groupCode int64) (bool, error) {
	row := QueryRow(`SELECT COUNT(*) FROM group_silence WHERE group_code=?`, groupCode)
	var count int
	if err := row.Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

// SetGroupSilence 设置群静音（enable=true 开启，false 解除）
func SetGroupSilence(groupCode int64, enable bool) error {
	if enable {
		_, err := Exec(`INSERT OR IGNORE INTO group_silence (group_code) VALUES (?)`, groupCode)
		return err
	}
	_, err := Exec(`DELETE FROM group_silence WHERE group_code=?`, groupCode)
	return err
}

// ─────────────────────────── Score / Checkin ─────────────────────────────────

// GetScore 获取用户积分（不存在时返回 0）
func GetScore(groupCode, userID int64) (int64, error) {
	row := QueryRow(`SELECT score FROM scores WHERE group_code=? AND user_id=?`, groupCode, userID)
	var score int64
	if err := row.Scan(&score); err != nil {
		return 0, nil
	}
	return score, nil
}

// CheckinResult 签到结果
type CheckinResult struct {
	Success bool   // 今日是否首次签到
	Score   int64  // 签到后积分
	Date    string // 今日日期 "2006-01-02"
}

// Checkin 签到（每日一次），返回签到结果
func Checkin(groupCode, userID int64) (*CheckinResult, error) {
	today := time.Now().Format("2006-01-02")

	// 检查是否已签到
	row := QueryRow(`SELECT score, last_checkin FROM scores WHERE group_code=? AND user_id=?`, groupCode, userID)
	var currentScore int64
	var lastCheckin string
	err := row.Scan(&currentScore, &lastCheckin)
	if err == nil && lastCheckin == today {
		// 今日已签到
		return &CheckinResult{Success: false, Score: currentScore, Date: today}, nil
	}

	// 执行签到（+1 积分）
	newScore := currentScore + 1
	_, err = Exec(`
		INSERT INTO scores (group_code, user_id, score, last_checkin, update_time)
		VALUES (?, ?, 1, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(group_code, user_id) DO UPDATE SET
			score=score+1, last_checkin=excluded.last_checkin, update_time=CURRENT_TIMESTAMP`,
		groupCode, userID, today)
	if err != nil {
		return nil, err
	}
	return &CheckinResult{Success: true, Score: newScore, Date: today}, nil
}

// ScoreRanking 积分排行榜（返回前 N 名）
func ScoreRanking(groupCode int64, limit int) ([]struct {
	UserID int64
	Score  int64
}, error) {
	rows, err := Query(`
		SELECT user_id, score FROM scores
		WHERE group_code=?
		ORDER BY score DESC LIMIT ?`, groupCode, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type entry struct {
		UserID int64
		Score  int64
	}
	var list []entry
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.UserID, &e.Score); err == nil {
			list = append(list, e)
		}
	}
	var result []struct {
		UserID int64
		Score  int64
	}
	for _, e := range list {
		result = append(result, struct {
			UserID int64
			Score  int64
		}{e.UserID, e.Score})
	}
	return result, nil
}
