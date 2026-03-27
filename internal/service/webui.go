package service

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/cnxysoft/DDBOT-WSa/internal/assets"
	"github.com/cnxysoft/DDBOT-WSa/internal/db"
	"github.com/cnxysoft/DDBOT-WSa/internal/logger"
	"github.com/cnxysoft/DDBOT-WSa/internal/module"
	"github.com/cnxysoft/DDBOT-WSa/internal/update"
)

// WebUIService 整合 WebUI 静态服务与后端 API
type WebUIService struct {
	addr          string
	root          string
	configDir     string
	moduleManager *module.Manager
	updateManager *update.Manager
	bot           onebotStatus // 轻量状态接口，避免循环依赖
	server        *http.Server
	startTime     time.Time
	version       string
}

// onebotStatus 轻量 Bot 状态接口（避免直接依赖 onebot 包）
type onebotStatus interface {
	IsConnected() bool
	GetSelfID() int64
}

// NewWebUIService 创建 WebUI 服务
// addr: 监听地址（如 "0.0.0.0:3000"）
// moduleManager: 模块管理器（用于 API 查询/控制模块）
func NewWebUIService(addr string, mm *module.Manager) *WebUIService {
	root := findDistDir()
	cfgDir := findConfigDir()

	return &WebUIService{
		addr:          addr,
		root:          root,
		configDir:     cfgDir,
		moduleManager: mm,
		startTime:     time.Now(),
		version:       "2.0.0",
	}
}

// SetUpdateManager 注入热更新管理器（可选，支持 WebUI 展示待更新列表）
func (s *WebUIService) SetUpdateManager(um *update.Manager) {
	s.updateManager = um
}

// SetBot 注入 OneBot 客户端状态（可选）
func (s *WebUIService) SetBot(b onebotStatus) {
	s.bot = b
}

// Start 启动服务（非阻塞）
func (s *WebUIService) Start() error {
	if _, err := os.Stat(s.root); os.IsNotExist(err) {
		logger.Warnf("WebUI 静态文件目录不存在: %s，仅提供 API 服务", s.root)
	}

	mux := http.NewServeMux()

	// ── API 路由 ─────────────────────────────────────────────────────────────
	// 兼容 backend-js 约定：/api/v1/* 由 Go 直接处理
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/v1/health", s.handleHealth)

	// 进程控制（WebUI 前端 invoke process_start/stop/restart）
	mux.HandleFunc("/api/process/control", s.handleProcessControl)
	mux.HandleFunc("/api/process/status", s.handleProcessStatus)

	// OneBot 状态
	mux.HandleFunc("/api/v1/onebot/status", s.handleOnebotStatus)
	mux.HandleFunc("/api/onebot/status", s.handleOnebotStatus)

	// 订阅概览
	mux.HandleFunc("/api/v1/subs/summary", s.handleSubsSummary)
	mux.HandleFunc("/api/subs/summary", s.handleSubsSummary)

	// 模块管理（热更新核心）
	mux.HandleFunc("/api/v1/modules", s.handleModules)
	mux.HandleFunc("/api/modules", s.handleModules)
	mux.HandleFunc("/api/v1/modules/", s.handleModuleAction)
	mux.HandleFunc("/api/modules/", s.handleModuleActionCompat)

	// 热更新
	mux.HandleFunc("/api/v1/updates", s.handleUpdates)
	mux.HandleFunc("/api/v1/updates/", s.handleApplyUpdate)

	// 配置文件读写
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/v1/config", s.handleConfig)
	mux.HandleFunc("/api/v1/config/reload", s.handleConfigReload)

	// 日志
	mux.HandleFunc("/api/logs", s.handleLogs)
	mux.HandleFunc("/api/v1/logs", s.handleLogs)

	// 认证（内网直通）
	mux.HandleFunc("/api/auth/login", s.handleLogin)
	mux.HandleFunc("/api/auth/logout", s.handleLogout)

	// 连接设置（读写 application.yaml 的 onebot/telegram 部分）
	mux.HandleFunc("/api/v1/settings", s.handleSettings)
	mux.HandleFunc("/api/v1/settings/reconnect", s.handleSettingsReconnect)

	// 订阅管理（CRUD）
	mux.HandleFunc("/api/v1/subs", s.handleSubs)
	mux.HandleFunc("/api/v1/subs/", s.handleSubsItem)

	// ── 静态文件（SPA 路由兜底）────────────────────────────────────────────
	// 优先使用外部目录（开发模式），回退到 embed.FS（生产模式）
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. 优先：外部目录存在则从磁盘读（方便开发调试）
		if s.root != "" {
			if _, err := os.Stat(s.root); err == nil {
				diskPath := filepath.Join(s.root, filepath.FromSlash(r.URL.Path))
				if info, err := os.Stat(diskPath); err == nil && !info.IsDir() {
					http.ServeFile(w, r, diskPath)
					return
				}
				// 文件不存在 → SPA 兜底
				idxPath := filepath.Join(s.root, "index.html")
				if _, err := os.Stat(idxPath); err == nil {
					http.ServeFile(w, r, idxPath)
					return
				}
			}
		}
		// 2. 回退：embed.FS（生产模式，无需外部文件）
		sub, err := fs.Sub(assets.DistFS, "dist")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		urlPath := strings.TrimPrefix(r.URL.Path, "/")
		if urlPath == "" {
			urlPath = "index.html"
		}
		if _, err := sub.(fs.StatFS).Stat(urlPath); err != nil {
			// 路径不存在 → 返回 index.html（SPA hash 路由）
			f, err := sub.Open("index.html")
			if err != nil {
				http.NotFound(w, r)
				return
			}
			f.Close()
			r2 := *r
			r2.URL.Path = "/index.html"
			http.FileServer(http.FS(sub)).ServeHTTP(w, &r2)
			return
		}
		http.FileServer(http.FS(sub)).ServeHTTP(w, r)
	}))

	s.server = &http.Server{
		Addr:         s.addr,
		Handler:      corsMiddleware(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		logger.Infof("WebUI 服务启动: http://%s  静态目录: %s", s.addr, s.root)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Errorf("WebUI 服务异常: %v", err)
		}
	}()

	return nil
}

// Stop 优雅关闭
func (s *WebUIService) Stop() {
	if s.server == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.server.Shutdown(ctx); err != nil {
		logger.Errorf("WebUI 关闭失败: %v", err)
	}
	logger.Info("WebUI 服务已停止")
}

// ─── 辅助 ────────────────────────────────────────────────────────────────────

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

// ─── API Handler ─────────────────────────────────────────────────────────────

// handleHealth GET /api/health
func (s *WebUIService) handleHealth(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(s.startTime).Round(time.Second).String()
	writeJSON(w, 200, map[string]any{
		"status":  "healthy",
		"version": s.version,
		"uptime":  uptime,
		"os":      runtime.GOOS,
	})
}

// handleProcessControl POST /api/process/control  {"action":"start"|"stop"|"restart"}
func (s *WebUIService) handleProcessControl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, map[string]string{"error": "method not allowed"})
		return
	}
	var body struct {
		Action string `json:"action"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	switch body.Action {
	case "stop":
		s.moduleManager.StopAll()
		writeJSON(w, 200, map[string]any{"running": false, "status": "stopped"})
	case "start", "restart":
		s.moduleManager.StartAll()
		writeJSON(w, 200, map[string]any{"running": true, "status": "running"})
	default:
		writeJSON(w, 400, map[string]string{"error": "invalid action"})
	}
}

// handleProcessStatus GET /api/process/status
func (s *WebUIService) handleProcessStatus(w http.ResponseWriter, r *http.Request) {
	mods := s.moduleManager.List()
	running := 0
	for _, m := range mods {
		if m.Status == module.StatusRunning {
			running++
		}
	}
	writeJSON(w, 200, map[string]any{
		"running": running > 0,
		"status":  fmt.Sprintf("%d/%d modules running", running, len(mods)),
	})
}

// handleOnebotStatus GET /api/v1/onebot/status
func (s *WebUIService) handleOnebotStatus(w http.ResponseWriter, r *http.Request) {
	connected := false
	var selfID int64
	if s.bot != nil {
		connected = s.bot.IsConnected()
		selfID = s.bot.GetSelfID()
	}
	writeJSON(w, 200, map[string]any{
		"online":    connected,
		"good":      connected,
		"connected": connected,
		"protocol":  "OneBot v11",
		"self_id":   selfID,
	})
}

// handleSubsSummary GET /api/v1/subs/summary
func (s *WebUIService) handleSubsSummary(w http.ResponseWriter, r *http.Request) {
	bySite, err := db.GetConcernCountBySite()
	if err != nil {
		logger.Warnf("查询订阅统计失败: %v", err)
		bySite = map[string]int{}
	}
	total := 0
	for _, v := range bySite {
		total += v
	}
	connected := false
	if s.bot != nil {
		connected = s.bot.IsConnected()
	}
	writeJSON(w, 200, map[string]any{
		"total":   total,
		"active":  total,
		"paused":  0,
		"bySite":  bySite,
		"offline": !connected,
	})
}

// handleModules GET/POST /api/v1/modules
func (s *WebUIService) handleModules(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, 200, map[string]any{"modules": s.moduleManager.List()})
	default:
		writeJSON(w, 405, map[string]string{"error": "method not allowed"})
	}
}

// handleModuleAction POST /api/v1/modules/{name}/{action}
// 支持: start / stop / reload
func (s *WebUIService) handleModuleAction(w http.ResponseWriter, r *http.Request) {
	// 路径: /api/v1/modules/{name}/{action}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/modules/"), "/")
	s.doModuleAction(w, r, parts)
}

// handleModuleActionCompat /api/modules/{name}/{action} 兼容路径
func (s *WebUIService) handleModuleActionCompat(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/modules/"), "/")
	s.doModuleAction(w, r, parts)
}

func (s *WebUIService) doModuleAction(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) < 2 {
		writeJSON(w, 400, map[string]string{"error": "invalid path, expected /api/v1/modules/{name}/{action}"})
		return
	}
	name, action := parts[0], parts[1]

	var err error
	switch action {
	case "start":
		err = s.moduleManager.StartModule(name)
	case "stop":
		err = s.moduleManager.StopModule(name)
	case "reload":
		err = s.moduleManager.ReloadModule(name)
	default:
		writeJSON(w, 400, map[string]string{"error": "unknown action: " + action})
		return
	}

	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

// handleConfig GET/POST /api/config
func (s *WebUIService) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		filename := r.URL.Query().Get("filename")
		if filename == "" {
			filename = "application.yaml"
		}
		path := filepath.Join(s.configDir, filepath.Base(filename))
		content, err := os.ReadFile(path)
		if err != nil {
			writeJSON(w, 404, map[string]string{"error": "config file not found: " + filename})
			return
		}
		writeJSON(w, 200, map[string]string{"content": string(content)})

	case http.MethodPost:
		var body struct {
			Filename string `json:"filename"`
			Content  string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid body"})
			return
		}
		if body.Filename == "" {
			body.Filename = "application.yaml"
		}
		path := filepath.Join(s.configDir, filepath.Base(body.Filename))
		if err := os.WriteFile(path, []byte(body.Content), 0644); err != nil {
			writeJSON(w, 500, map[string]string{"error": "write config failed: " + err.Error()})
			return
		}
		writeJSON(w, 200, map[string]string{"status": "saved"})

	default:
		writeJSON(w, 405, map[string]string{"error": "method not allowed"})
	}
}

// handleUpdates GET /api/v1/updates  获取待更新列表
func (s *WebUIService) handleUpdates(w http.ResponseWriter, r *http.Request) {
	if s.updateManager == nil {
		writeJSON(w, 200, map[string]any{"updates": []any{}})
		return
	}
	writeJSON(w, 200, map[string]any{"updates": s.updateManager.GetPendingUpdates()})
}

// handleApplyUpdate POST /api/v1/updates/{moduleName}  触发指定模块更新
func (s *WebUIService) handleApplyUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, map[string]string{"error": "method not allowed"})
		return
	}
	moduleName := strings.TrimPrefix(r.URL.Path, "/api/v1/updates/")
	if moduleName == "" {
		writeJSON(w, 400, map[string]string{"error": "module name required"})
		return
	}
	if s.updateManager == nil {
		writeJSON(w, 503, map[string]string{"error": "update manager not available"})
		return
	}
	if err := s.updateManager.ApplyUpdate(moduleName); err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]string{"status": "updated"})
}

// handleConfigReload POST /api/v1/config/reload  触发热重载
func (s *WebUIService) handleConfigReload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, map[string]string{"error": "method not allowed"})
		return
	}
	// 触发所有模块 Reload
	for _, mod := range s.moduleManager.ListModules() {
		if err := mod.Reload(); err != nil {
			logger.Errorf("热重载模块失败 [%s]: %v", mod.Name(), err)
		}
	}
	writeJSON(w, 200, map[string]string{"status": "reloaded"})
}

// handleLogs GET /api/logs?lines=100&level=info
func (s *WebUIService) handleLogs(w http.ResponseWriter, r *http.Request) {
	lines := 100
	fmt.Sscanf(r.URL.Query().Get("lines"), "%d", &lines)
	if lines <= 0 || lines > 1000 {
		lines = 100
	}
	level := r.URL.Query().Get("level")

	logs := readLastLines(findLogFile(), lines, level)
	writeJSON(w, 200, map[string]any{"logs": logs})
}

// handleLogin POST /api/auth/login
func (s *WebUIService) handleLogin(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"success": true, "message": "Logged in"})
}

// handleLogout POST /api/auth/logout
func (s *WebUIService) handleLogout(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"success": true})
}

// ─── 订阅管理 API ─────────────────────────────────────────────────────────────

// handleSubs GET  /api/v1/subs?group=<groupID>&site=<site>  查询订阅列表
//
//	POST /api/v1/subs  添加订阅 {"site":"bilibili","uid":"123","group_code":456,"concern_type":"live"}
func (s *WebUIService) handleSubs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		var groupCode int64
		fmt.Sscanf(r.URL.Query().Get("group"), "%d", &groupCode)
		site := r.URL.Query().Get("site")

		var concerns []*db.Concern
		var err error
		if groupCode > 0 {
			concerns, err = db.GetConcerns(groupCode, site)
		} else {
			// 无 group 参数时，按平台返回全量（供管理员视图）
			concerns, err = db.GetAllConcernsBySite(site)
		}
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"subs": concerns, "total": len(concerns)})

	case http.MethodPost:
		var body struct {
			Site        string `json:"site"`
			UID         string `json:"uid"`
			Name        string `json:"name"`
			GroupCode   int64  `json:"group_code"`
			ConcernType string `json:"concern_type"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid body"})
			return
		}
		if body.Site == "" || body.UID == "" || body.GroupCode == 0 {
			writeJSON(w, 400, map[string]string{"error": "site, uid, group_code are required"})
			return
		}
		if body.ConcernType == "" {
			body.ConcernType = "live"
		}
		if err := db.InsertConcern(body.Site, body.UID, body.Name, body.GroupCode, body.ConcernType); err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 201, map[string]string{"status": "created"})

	default:
		writeJSON(w, 405, map[string]string{"error": "method not allowed"})
	}
}

// handleSubsItem DELETE /api/v1/subs/{site}/{uid}?group=<groupCode>
func (s *WebUIService) handleSubsItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeJSON(w, 405, map[string]string{"error": "method not allowed"})
		return
	}
	// 路径：/api/v1/subs/{site}/{uid}
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/subs/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 {
		writeJSON(w, 400, map[string]string{"error": "path must be /api/v1/subs/{site}/{uid}"})
		return
	}
	site, uid := parts[0], parts[1]
	var groupCode int64
	fmt.Sscanf(r.URL.Query().Get("group"), "%d", &groupCode)
	if groupCode == 0 {
		writeJSON(w, 400, map[string]string{"error": "query param group (group_code) is required"})
		return
	}
	if err := db.DeleteConcern(site, uid, groupCode); err != nil {
		if strings.Contains(err.Error(), "不存在") {
			writeJSON(w, 404, map[string]string{"error": "subscription not found"})
			return
		}
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

// ─── 文件工具 ─────────────────────────────────────────────────────────────────

func findDistDir() string {
	candidates := []string{
		"./DDBOT-WebUI/dist",
		"./webui/dist",
		"./webui",
		"./static",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "./webui"
}

func findConfigDir() string {
	candidates := []string{"./configs", "./data", "."}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "."
}

func findLogFile() string {
	candidates := []string{
		"./logs/ddbot.log",
		"./data/logs/ddbot.log",
		"./ddbot.log",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// ─── 连接设置 API ─────────────────────────────────────────────────────────────

// settingsPayload 连接设置读写的 JSON 结构
type settingsPayload struct {
	OneBot   onebotSettings   `json:"onebot"`
	Telegram telegramSettings `json:"telegram"`
}

type onebotSettings struct {
	Mode        string `json:"mode"`         // "reverse" | "forward"
	WsListen    string `json:"ws_listen"`    // 反向 WS 监听地址
	WsURL       string `json:"ws_url"`       // 正向 WS 目标地址
	AccessToken string `json:"access_token"` // 鉴权 token
}

type telegramSettings struct {
	Enabled  bool   `json:"enabled"`
	BotToken string `json:"bot_token"`
	Proxy    string `json:"proxy"`
}

// handleSettings GET/POST /api/v1/settings
// 读写 application.yaml 中的连接相关配置
func (s *WebUIService) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// 从 viper 实时读取（config.C）
		mode := "reverse"
		wsListen := getViperString("onebot.ws_listen", "0.0.0.0:8080")
		wsURL := getViperString("onebot.ws_url", "ws://127.0.0.1:3001")
		if wsListen == "" {
			mode = "forward"
		}
		payload := settingsPayload{
			OneBot: onebotSettings{
				Mode:        mode,
				WsListen:    wsListen,
				WsURL:       wsURL,
				AccessToken: getViperString("onebot.access_token", ""),
			},
			Telegram: telegramSettings{
				Enabled:  getViperBool("telegram.enabled", false),
				BotToken: getViperString("telegram.bot_token", ""),
				Proxy:    getViperString("telegram.proxy", ""),
			},
		}
		writeJSON(w, 200, payload)

	case http.MethodPost:
		var payload settingsPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid body"})
			return
		}
		// 写回 application.yaml
		cfgPath := filepath.Join(".", "application.yaml")
		if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
			cfgPath = filepath.Join(s.configDir, "application.yaml")
		}
		if err := patchYAML(cfgPath, payload); err != nil {
			writeJSON(w, 500, map[string]string{"error": "写入配置失败: " + err.Error()})
			return
		}
		writeJSON(w, 200, map[string]string{"status": "saved", "hint": "重启 Bot 生效（或点击重连）"})

	default:
		writeJSON(w, 405, map[string]string{"error": "method not allowed"})
	}
}

// handleSettingsReconnect POST /api/v1/settings/reconnect
// 重启 OneBot 连接（热切换，无需重启进程）
func (s *WebUIService) handleSettingsReconnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, map[string]string{"error": "method not allowed"})
		return
	}
	if s.bot == nil {
		writeJSON(w, 503, map[string]string{"error": "bot not initialized"})
		return
	}
	// 调用可选的 Restart 接口（如果 bot 实现了 Restarter 接口）
	type Restarter interface {
		Restart() error
	}
	if rs, ok := s.bot.(Restarter); ok {
		if err := rs.Restart(); err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]string{"status": "reconnecting"})
	} else {
		writeJSON(w, 200, map[string]string{"status": "not_supported", "hint": "请重启 Bot 进程使新配置生效"})
	}
}

// getViperString 从 viper 读字符串，带默认值（避免引入 config 包的循环依赖）
func getViperString(key, defaultVal string) string {
	// 通过已加载的配置文件直接读，使用反射式动态方式
	v := viperGet(key)
	if v == "" {
		return defaultVal
	}
	return v
}

func getViperBool(key string, defaultVal bool) bool {
	v := viperGet(key)
	if v == "" {
		return defaultVal
	}
	return v == "true" || v == "1" || v == "yes"
}

// viperGet 通过读取配置文件获取值（不引入 viper 包依赖，直接读 YAML 文件）
func viperGet(key string) string {
	content, err := os.ReadFile(findAppYAML())
	if err != nil {
		return ""
	}
	return parseYAMLValue(string(content), key)
}

// findAppYAML 找到运行时的 application.yaml 路径
func findAppYAML() string {
	candidates := []string{"./application.yaml", "./configs/application.yaml"}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "./application.yaml"
}

// parseYAMLValue 极简 YAML key 提取（支持 "onebot.ws_listen" 这类两级 key）
func parseYAMLValue(content, key string) string {
	parts := strings.SplitN(key, ".", 2)
	if len(parts) == 1 {
		// 顶层 key
		for _, line := range strings.Split(content, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, parts[0]+":") {
				val := strings.TrimPrefix(line, parts[0]+":")
				val = strings.TrimSpace(val)
				val = strings.Trim(val, `"'`)
				return val
			}
		}
		return ""
	}
	// 两级 key：先找父节点，再在其缩进块内找子 key
	section, subKey := parts[0], parts[1]
	inSection := false
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if trimmed == section+":" || strings.HasPrefix(trimmed, section+":") {
			// 父节点本身
			if trimmed == section+":" {
				inSection = true
				continue
			}
		}
		if inSection {
			// 如果这一行是顶层（无缩进）且不是本 section，则退出
			if len(line) > 0 && line[0] != ' ' && line[0] != '\t' {
				inSection = false
				continue
			}
			if strings.HasPrefix(trimmed, subKey+":") {
				val := strings.TrimPrefix(trimmed, subKey+":")
				val = strings.TrimSpace(val)
				val = strings.Trim(val, `"'`)
				// 跳过注释
				if idx := strings.Index(val, " #"); idx >= 0 {
					val = strings.TrimSpace(val[:idx])
				}
				return val
			}
		}
	}
	return ""
}

// patchYAML 将 payload 中的值写回 application.yaml
// 采用行替换策略，保留原文件注释和格式
func patchYAML(path string, payload settingsPayload) error {
	content, err := os.ReadFile(path)
	if err != nil {
		// 文件不存在时新建
		content = []byte{}
	}

	lines := strings.Split(string(content), "\n")

	// 根据 mode 决定 ws_listen / ws_url 的值
	wsListen := ""
	wsURL := ""
	if payload.OneBot.Mode == "reverse" {
		wsListen = payload.OneBot.WsListen
		wsURL = "" // 反向模式下注释掉正向地址
	} else {
		wsListen = "" // 正向模式下注释掉监听地址
		wsURL = payload.OneBot.WsURL
	}

	// 需要更新的 key-value 映射（section -> key -> newValue）
	patches := map[string]map[string]string{
		"onebot": {
			"ws_listen":    wsListen,
			"ws_url":       wsURL,
			"access_token": payload.OneBot.AccessToken,
		},
		"telegram": {
			"enabled":   boolToStr(payload.Telegram.Enabled),
			"bot_token": payload.Telegram.BotToken,
			"proxy":     payload.Telegram.Proxy,
		},
	}

	result := patchYAMLLines(lines, patches)
	out := strings.Join(result, "\n")
	return os.WriteFile(path, []byte(out), 0644)
}

func boolToStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// patchYAMLLines 对 YAML 行数组做 in-place 替换
func patchYAMLLines(lines []string, patches map[string]map[string]string) []string {
	currentSection := ""
	result := make([]string, len(lines))
	copy(result, lines)

	for i, line := range result {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		// 检测顶层 section
		if len(line) > 0 && line[0] != ' ' && line[0] != '\t' && strings.Contains(trimmed, ":") {
			kv := strings.SplitN(trimmed, ":", 2)
			currentSection = strings.TrimSpace(kv[0])
			continue
		}
		// 在 section 内部寻找 key
		if sectionPatches, ok := patches[currentSection]; ok {
			for subKey, newVal := range sectionPatches {
				prefix := subKey + ":"
				if strings.HasPrefix(trimmed, prefix) || strings.HasPrefix(trimmed, "# "+subKey+":") || strings.HasPrefix(trimmed, "#"+subKey+":") {
					// 计算缩进
					indent := ""
					for _, ch := range line {
						if ch == ' ' || ch == '\t' {
							indent += string(ch)
						} else {
							break
						}
					}
					// 去掉注释前缀
					indent = strings.TrimLeft(indent, "#")
					indent = "  " // 标准两空格缩进
					if newVal == "" {
						// 注释掉此行
						result[i] = "  # " + subKey + ": \"\""
					} else {
						result[i] = indent + subKey + ": \"" + newVal + "\""
					}
					delete(sectionPatches, subKey)
					break
				}
			}
		}
	}
	return result
}

// readLastLines 从文件末尾读取 n 行（可按 level 过滤）
func readLastLines(path string, n int, level string) []string {
	if path == "" {
		return []string{"[INFO] No log file found."}
	}
	f, err := os.Open(path)
	if err != nil {
		return []string{"[ERROR] Cannot open log file: " + err.Error()}
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if level != "" && !strings.Contains(strings.ToUpper(line), strings.ToUpper(level)) {
			continue
		}
		lines = append(lines, line)
	}

	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines
}
