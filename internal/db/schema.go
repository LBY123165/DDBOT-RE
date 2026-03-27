package db

import (
	"github.com/cnxysoft/DDBOT-WSa/internal/logger"
)

// InitSchema 初始化数据库表结构
func InitSchema() error {
	logger.Info("正在初始化数据库表结构")

	// 关注表（support concern_type 字段）
	concernTableSQL := `
	CREATE TABLE IF NOT EXISTS concerns (
		id           INTEGER PRIMARY KEY AUTOINCREMENT,
		site         VARCHAR(50)  NOT NULL,
		uid          VARCHAR(100) NOT NULL,
		name         VARCHAR(200) DEFAULT '',
		group_code   INTEGER      NOT NULL,
		concern_type VARCHAR(50)  DEFAULT 'live',
		enable       INTEGER      DEFAULT 1,
		create_time  DATETIME     DEFAULT CURRENT_TIMESTAMP,
		update_time  DATETIME     DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(site, uid, group_code, concern_type)
	);
	CREATE INDEX IF NOT EXISTS idx_concerns_site  ON concerns(site);
	CREATE INDEX IF NOT EXISTS idx_concerns_group ON concerns(group_code);
	CREATE INDEX IF NOT EXISTS idx_concerns_uid   ON concerns(uid);
	`

	// 配置表
	configTableSQL := `
	CREATE TABLE IF NOT EXISTS configs (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		key_name    VARCHAR(100) UNIQUE NOT NULL,
		value       TEXT,
		description TEXT,
		create_time DATETIME DEFAULT CURRENT_TIMESTAMP,
		update_time DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_configs_key ON configs(key_name);
	`

	// 模块状态表
	moduleStatusTableSQL := `
	CREATE TABLE IF NOT EXISTS module_status (
		id             INTEGER PRIMARY KEY AUTOINCREMENT,
		module_name    VARCHAR(100) UNIQUE NOT NULL,
		version        VARCHAR(50),
		status         VARCHAR(20),
		last_heartbeat DATETIME,
		create_time    DATETIME DEFAULT CURRENT_TIMESTAMP,
		update_time    DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_module_status_name ON module_status(module_name);
	`

	// 管理员表（存储有权操作 BOT 的 QQ 号）
	adminTableSQL := `
	CREATE TABLE IF NOT EXISTS admins (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id     INTEGER UNIQUE NOT NULL,
		note        TEXT,
		create_time DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	// 群关注推送配置表（at/notify/filter）
	concernConfigTableSQL := `
	CREATE TABLE IF NOT EXISTS concern_configs (
		id           INTEGER PRIMARY KEY AUTOINCREMENT,
		group_code   INTEGER NOT NULL,
		site         VARCHAR(50) NOT NULL,
		uid          VARCHAR(100) NOT NULL,
		at_mode      INTEGER DEFAULT 0,    -- 0=不At, 1=At全体, 2=At指定成员
		at_members   TEXT    DEFAULT '',   -- JSON 数组，at_mode=2 时有效
		notify_live  INTEGER DEFAULT 1,    -- 开播通知
		notify_news  INTEGER DEFAULT 1,    -- 动态/推送通知
		filter_text  TEXT    DEFAULT '',   -- 关键词过滤（JSON 字符串数组），空=不过滤
		create_time  DATETIME DEFAULT CURRENT_TIMESTAMP,
		update_time  DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(group_code, site, uid)
	);
	CREATE INDEX IF NOT EXISTS idx_concern_cfg_group ON concern_configs(group_code);
	`

	// 群命令开关配置表（enable/disable）
	commandConfigTableSQL := `
	CREATE TABLE IF NOT EXISTS command_configs (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		group_code INTEGER NOT NULL,
		command    VARCHAR(100) NOT NULL,
		enabled    INTEGER DEFAULT 1,
		create_time DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(group_code, command)
	);
	CREATE INDEX IF NOT EXISTS idx_cmd_cfg_group ON command_configs(group_code);
	`

	// 黑名单表（block 系统）
	blocklistTableSQL := `
	CREATE TABLE IF NOT EXISTS blocklist (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		target_type VARCHAR(20) NOT NULL,  -- "user" | "group"
		target_id   INTEGER NOT NULL,
		reason      TEXT DEFAULT '',
		create_time DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(target_type, target_id)
	);
	`

	// 群静音表（silence 命令）
	groupSilenceTableSQL := `
	CREATE TABLE IF NOT EXISTS group_silence (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		group_code  INTEGER UNIQUE NOT NULL,
		create_time DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	// 积分表（签到/score 系统）
	scoreTableSQL := `
	CREATE TABLE IF NOT EXISTS scores (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		group_code  INTEGER NOT NULL,
		user_id     INTEGER NOT NULL,
		score       INTEGER DEFAULT 0,
		last_checkin DATE,
		create_time DATETIME DEFAULT CURRENT_TIMESTAMP,
		update_time DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(group_code, user_id)
	);
	CREATE INDEX IF NOT EXISTS idx_scores_group ON scores(group_code);
	`

	// 定时任务表（cronjob 系统）
	cronJobTableSQL := `
	CREATE TABLE IF NOT EXISTS cron_jobs (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		name        VARCHAR(100) NOT NULL,
		group_code  INTEGER NOT NULL,
		cron_expr   VARCHAR(100) NOT NULL,    -- cron 表达式，如 "0 9 * * 1"（每周一9点）
		template    TEXT NOT NULL,             -- 消息模板（支持 {{.Now}} 等变量）
		enabled     INTEGER DEFAULT 1,
		last_run    DATETIME,
		create_time DATETIME DEFAULT CURRENT_TIMESTAMP,
		update_time DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(group_code, name)
	);
	CREATE INDEX IF NOT EXISTS idx_cron_group ON cron_jobs(group_code);
	`

	tables := []string{
		concernTableSQL,
		configTableSQL,
		moduleStatusTableSQL,
		adminTableSQL,
		concernConfigTableSQL,
		commandConfigTableSQL,
		blocklistTableSQL,
		groupSilenceTableSQL,
		scoreTableSQL,
		cronJobTableSQL,
	}

	for _, sqlStr := range tables {
		if _, err := DB.Exec(sqlStr); err != nil {
			logger.Errorf("创建表失败：%v", err)
			return err
		}
	}

	logger.Info("数据库表结构初始化完成")
	return nil
}
