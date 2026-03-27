// 内存存储，用于在浏览器环境中模拟配置文件的保存和读取
const memoryStorage: Record<string, string> = {
  'application.yaml': `bot:
  onJoinGroup:
    rename: "【bot】"
  sendFailureReminder:
    enable: false
    times: 3
  offlineQueue:
    enable: false
    expire: 30m
bilibili:
  SESSDATA: 
  bili_jct: 
  qrlogin: true
  interval: 25s
  imageMergeMode: "auto"
  hiddenSub: false
  unsub: false
  minFollowerCap: 0
  disableSub: false
  onlyOnlineNotify: false
  autoParsePosts: false
  secAnalysis: false
logLevel: info`
};

// 仅保留一个Go后端
const GO_API_BASE = `/api/v1`;

// 提供纯前端 Web 面板实现
const invoke: any = async (cmd: string, args?: any) => {
  console.log(`Invoke: ${cmd}`, args);

  try {
    switch (cmd) {
      // 纯 WebUI 无法启动/停止底层 Go 进程
      case 'process_status_text':
        try {
          const res = await fetch(`${GO_API_BASE}/health`);
          if (res.status === 401) { window.location.hash = '#/login'; return 'stopped'; }
          return res.ok ? 'running' : 'stopped';
        } catch {
          return 'stopped';
        }
      case 'process_start':
      case 'process_stop':
      case 'process_restart':
        console.warn('Cannot control process purely from frontend.');
        return;

      // 数据获取 (Go 后端)
      case 'call_onebot_status_api':
        const obRes = await fetch(`${GO_API_BASE}/onebot/status`);
        return await obRes.json();
      case 'onebot_status_text':
        try {
          const obRes = await fetch(`${GO_API_BASE}/onebot/status`);
          if (obRes.status === 401) { window.location.hash = '#/login'; return 'Offline'; }
          const obStatus = await obRes.json();
          return obStatus.online ? 'Online' : 'Offline';
        } catch {
          return 'Offline';
        }
      case 'call_subs_summary_api':
        const subRes = await fetch(`${GO_API_BASE}/subs/summary`);
        return await subRes.json();
      case 'subs_summary_text':
        try {
          const subRes = await fetch(`${GO_API_BASE}/subs/summary`);
          if (subRes.status === 401) { window.location.hash = '#/login'; return 'Offline'; }
          const subData = await subRes.json();
          return `${subData.total} Subs`;
        } catch {
          return 'Offline';
        }

      // 配置文件相关
      case 'read_config_file':
        const filename = args?.filename || 'application.yaml';
        try {
          const cfgRes = await fetch(`/api/config?filename=${filename}`);
          if (cfgRes.status === 401) { window.location.hash = '#/login'; return ''; }
          if (cfgRes.ok) {
            const data = await cfgRes.json();
            return data.content;
          }
        } catch (e) {
          console.error('Failed to fetch config from Rust API', e);
        }
        return memoryStorage[filename] || '';

      case 'write_config_file':
        const writeFilename = args?.filename || 'application.yaml';
        let contentObj = args?.content || '';

        try {
          // If the caller gave an object (from the graphical editor without yaml package earlier), 
          // they'd pass stringified JSON, but now they should pass text config.
          // Wait, if it's already string, we just send it.
          const textContent = typeof contentObj === 'string' ? contentObj : JSON.stringify(contentObj, null, 2);

          const res = await fetch(`/api/config`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ filename: writeFilename, content: textContent })
          });
          if (res.status === 401) { window.location.hash = '#/login'; return null; }

          // Trigger hot-reload in Go Main process if it's running
          if (writeFilename === 'application.yaml') {
            try {
              await fetch(`${GO_API_BASE}/config/reload`, { method: 'POST' });
              console.log('Triggered hot-reload on DDBot');
            } catch (ignore) { }
          }
          return null;

        } catch (e) {
          console.error('Failed to save config via Rust API', e);
          throw e;
        }

      case 'read_logs_tail':
        try {
          const linesLimit = args?.lines || 100;
          const levelQuery = args?.level ? `&level=${args.level}` : '';
          const logRes = await fetch(`/api/logs?lines=${linesLimit}${levelQuery}`);
          if (logRes.status === 401) { window.location.hash = '#/login'; return []; }
          if (logRes.ok) {
            const data = await logRes.json();
            if (data.logs && data.logs.length > 0) {
              return data.logs;
            }
          }
        } catch (e) { }
        return [
          '[INFO] Waiting for logs or DDBOT is not running...'
        ];

      case 'list_config_backups':
        return [];
      case 'check_firewall_rule':
        return false;
      case 'get_current_version':
        return '1.0.0';
      case 'get_admin_password':
        return 'admin123';
      default:
        console.log(`Mock invoke fallback: ${cmd} (unhandled)`, args);
        return null;
    }
  } catch (err) {
    console.error(`Error in invoke ${cmd}:`, err);
    return null;
  }
};

const listen: any = async (event: string, callback: any) => {
  console.log(`Mock listen: ${event}`);
  return {
    remove: () => console.log(`Mock remove listener: ${event}`)
  };
};

const appDataDir: any = async () => {
  return 'C:\\Users\\User\\AppData\\Roaming\\DDBOT-WebUI';
};

const join: any = async (...paths: string[]) => {
  return paths.join('\\');
};

// 模拟插件功能
const relaunch = async (): Promise<void> => {
  // 实际应用中，这会重启应用
  console.log('Relaunching application...');
};

const check = async (): Promise<any> => {
  // 实际应用中，这会检查更新
  console.log('Checking for updates...');
  return null;
};

let pendingUpdateInstance: any = null;

// 类型定义
interface ProcessInfo {
  pid: number;
  running: boolean;
}

interface DDBOTCoreStatus {
  running: boolean;
  pid?: number;
  version?: string;
}

interface MergedSettings {
  [key: string]: any;
}

interface UpdateCheck {
  hasUpdate: boolean;
  currentVersion: string;
  latestVersion: string;
  releaseDate: string;
  releaseNotes: string;
  assets: any[];
}

interface DownloadProgress {
  downloaded: number;
  total: number;
  percentage: number;
  speed: number;
}

// 使用函数工厂模式创建TauriAPI，确保每次调用都使用最新的invoke变量
export const TauriAPI = {
  // --- File management ---
  get files() {
    return {
      open: (path: string): Promise<boolean> => invoke('open_file', { path }),
      folder: (path: string): Promise<boolean> => invoke('open_folder', { path }),
      urlInBrowser: (url: string): Promise<boolean> => invoke('open_url_in_browser', { url }),
      openDDBotDataDir: (): Promise<boolean> => invoke('open_ddbot_data_dir'),
      openLogsDirectory: (): Promise<boolean> => invoke('open_logs_directory'),
      openConfigFile: (): Promise<boolean> => invoke('open_config_file'),
      openSettingsFile: (): Promise<boolean> => invoke('open_settings_file'),
    };
  },

  // --- DDBOT management ---
  get ddbot() {
    return {
      getPlatformTriple: (): Promise<string> => invoke('get_platform_triple'),
      getWorkdir: (): Promise<string> => invoke('get_workdir'),
      getBinaryPath: (): Promise<string> => invoke('get_binary_path'),
      getManagedDdbotDir: (): Promise<string> => invoke('get_managed_ddbot_dir'),
      isUserApproved: (): Promise<boolean> => invoke('is_user_approved'),
      setUserApproved: (approved: boolean): Promise<void> => invoke('set_user_approved', { approved }),
      importExistingDeployment: (srcDir: string): Promise<void> => invoke('import_existing_deployment', { src_dir: srcDir }),
      ensureDdbotInstalled: (): Promise<void> => invoke('ensure_ddbot_installed'),
      readConfigFile: (filename: string): Promise<string> => invoke('read_config_file', { filename }),
      writeConfigFile: (filename: string, content: string): Promise<void> => invoke('write_config_file', { filename, content }),
      readLogsTail: (lines: number, level?: string): Promise<string[]> => invoke('read_logs_tail', { lines, level }),
      listConfigBackups: (filename: string): Promise<string[]> => invoke('list_config_backups', { filename }),
      restoreConfigBackup: (backupName: string): Promise<void> => invoke('restore_config_backup', { backup_name: backupName }),
      installedVersionText: (): Promise<string> => invoke('installed_version_text'),
    };
  },

  // --- Process management ---
  get process() {
    return {
      start: (): Promise<void> => invoke('process_start'),
      stop: (): Promise<void> => invoke('process_stop'),
      restart: (): Promise<void> => invoke('process_restart'),
      statusText: (): Promise<string> => invoke('process_status_text'),
      onebotStatusText: (): Promise<string> => invoke('onebot_status_text'),
      subsSummaryText: (): Promise<string> => invoke('subs_summary_text'),
      callOnebotStatusApi: (): Promise<any> => invoke('call_onebot_status_api'),
      callSubsSummaryApi: (): Promise<any> => invoke('call_subs_summary_api'),
    };
  },

  // --- Download management ---
  get download() {
    return {
      downloadFile: async (url: string, savePath: string, filename?: string): Promise<string> => {
        return invoke('download_file', { url, save_path: savePath, filename });
      },
    };
  },

  // --- Settings management ---
  get settings() {
    return {
      load: (): Promise<MergedSettings | null> => invoke('load_settings'),
      save: (s: MergedSettings): Promise<boolean> => invoke('save_settings', { settings: s }),
      saveAndRestart: (s: MergedSettings): Promise<boolean> => invoke('save_settings_and_restart', { settings: s }),
      reset: (): Promise<MergedSettings | null> => invoke('reset_settings'),
    };
  },

  // --- Logs management ---
  get logs() {
    return {
      get: (src?: 'ddbot' | 'app' | 'all'): Promise<string[]> =>
        invoke('get_logs', { source: src }),
      clear: (src?: 'ddbot' | 'app' | 'all'): Promise<boolean> =>
        invoke('clear_logs', { source: src }),
      adminPassword: (): Promise<string> => invoke('get_admin_password'),
      resetAdminPassword: (): Promise<string> => invoke('reset_admin_password'),
      setAdminPassword: (password: string): Promise<string> => invoke('set_admin_password', { password }),
    };
  },

  // --- Binary management ---
  get bin() {
    return {
      version: (name: 'ddbot'): Promise<string> => invoke('get_binary_version', { binaryName: name }),
      availableVersions: (tool: string, force: boolean): Promise<string[]> =>
        invoke('get_available_versions', { tool, force }),
      updateVersion: (tool: string, version: string): Promise<string> => invoke('update_tool_version', { tool, version }),
    };
  },

  // --- Firewall management (removed) ---
  get firewall() {
    return {
      check: (): Promise<boolean> => Promise.resolve(false),
      add: (): Promise<boolean> => Promise.resolve(false),
      remove: (): Promise<boolean> => Promise.resolve(false),
    };
  },

  // --- Tray management ---
  get tray() {
    return {
      update: (r: boolean): Promise<void> => invoke('update_tray_menu', { serviceRunning: r }),
    };
  },

  // --- Update management ---
  get updater() {
    return {
      check: async (): Promise<UpdateCheck> => {
        const update = await check();
        const currentVersion = await invoke('get_current_version');

        pendingUpdateInstance = update;

        if (update) {
          return {
            hasUpdate: true,
            currentVersion,
            latestVersion: '1.0.0',
            releaseDate: new Date().toISOString(),
            releaseNotes: 'New features and bug fixes',
            assets: [],
          };
        }

        return {
          hasUpdate: false,
          currentVersion,
          latestVersion: currentVersion,
          releaseDate: '',
          releaseNotes: '',
          assets: [],
        };
      },

      hasPendingUpdate: (): boolean => {
        return pendingUpdateInstance !== null;
      },

      downloadAndInstall: async (onProgress?: (progress: DownloadProgress) => void): Promise<void> => {
        if (!pendingUpdateInstance) {
          throw new Error('No pending update available. Call check() first.');
        }

        // 模拟下载进度
        let downloaded = 0;
        const contentLength = 1000000;

        if (onProgress) {
          onProgress({
            downloaded: 0,
            total: contentLength,
            percentage: 0,
            speed: 0,
          });
        }

        // 模拟下载过程
        for (let i = 0; i < 10; i++) {
          await new Promise(resolve => setTimeout(resolve, 500));
          downloaded += contentLength / 10;
          if (onProgress) {
            onProgress({
              downloaded,
              total: contentLength,
              percentage: (downloaded / contentLength) * 100,
              speed: contentLength / 10 / 0.5,
            });
          }
        }

        if (onProgress) {
          onProgress({
            downloaded: contentLength,
            total: contentLength,
            percentage: 100,
            speed: 0,
          });
        }

        pendingUpdateInstance = null;
      },

      clearPendingUpdate: (): void => {
        pendingUpdateInstance = null;
      },

      relaunch: async (): Promise<void> => {
        await relaunch();
      },

      currentVersion: (): Promise<string> => invoke('get_current_version'),
      setAutoCheck: (e: boolean): Promise<void> => invoke('set_auto_check_enabled', { enabled: e }),
      isAutoCheckEnabled: (): Promise<boolean> => invoke('is_auto_check_enabled'),
      onBackgroundUpdate: (cb: (u: UpdateCheck) => void) =>
        listen('background-update-available', (e: any) => cb(e.payload as UpdateCheck)),
    };
  },

  // --- Utility functions ---
  get util() {
    return {
      defaultDataDir: async (): Promise<string> => {
        const base = await appDataDir();
        return join(base, 'DDBOT-WebUI');
      },
      defaultBinaryDir: async (): Promise<string> => {
        const base = await appDataDir();
        return join(base, 'DDBOT-WebUI', 'binary');
      },
      defaultDataDdbotDir: async (): Promise<string> => {
        const base = await appDataDir();
        return join(base, 'DDBOT-WebUI', 'data', 'ddbot');
      },
      selectDirectory: (title: string): Promise<string | null> => invoke('select_directory', { title }),
    };
  },
};
