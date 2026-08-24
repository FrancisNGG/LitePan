package app

import (
	"context"
	"encoding/json"
	"log/slog"
	"path/filepath"
	"strings"

	"litepan/internal/cache"
	"litepan/internal/domain"
	"litepan/internal/eventbus"
	"litepan/internal/file"
	"litepan/internal/logx"
	"litepan/internal/playback"
	"litepan/internal/strm"
)

func wireSTRM(st *storeBundle, files *file.Service, playback *playback.Service, bus *eventbus.Bus, logs *logx.Manager, dataDir, strmDir, listenAddr string, secret []byte) (*strm.Service, *strm.Coordinator) {
	svc := strm.NewService(strm.ServiceOptions{
		Repo:       st.store.StrmTasks,
		Branches:   st.store.StrmBranches,
		DirCache:   st.store.StrmDirCache,
		Files:      files,
		Playback:   playback,
		Settings:   st.settings,
		DataDir:    dataDir,
		StrmDir:    strmDir,
		ListenAddr: listenAddr,
		Secret:     secret,
		Bus:        bus,
		Log:        logs.For(logx.ModuleSystem),
	})
	coord := strm.NewCoordinator(strm.Options{
		Runner: svc,
		Log:    logs.For(logx.ModuleSystem),
	})
	coord.Register(bus)
	return svc, coord
}

// registerStrmWebDAVInvalidation 注册「STRM 任务完成后失效 WebDAV 缓存」订阅者。
//
// strm 文件由 os.WriteFile 直写文件系统，不经过 file.Service 事件总线，
// WebDAV 目录/PROPFIND 缓存无法感知新文件；任务成功后失效指向 strmDir 的
// localfs 账号缓存，保证客户端立即可见。
//
// 该逻辑放在 app 组装层（而非 strm 模块）：localfs 账号配置结构属于账号领域，
// strm 只发布 StrmScanCompleted 领域事件，不感知账号/缓存细节（消除 strm 与
// 账号配置 JSON 的隐式契约）。
func registerStrmWebDAVInvalidation(bus *eventbus.Bus, accounts domain.AccountRepository, cacheSvc *cache.Service, strmDir string, log *slog.Logger) {
	if bus == nil || accounts == nil || cacheSvc == nil {
		return
	}
	defaultRoot := resolvePath(strmDir)
	eventbus.Subscribe(bus, func(ctx context.Context, evt eventbus.StrmScanCompleted) {
		root := resolvePath(evt.StrmDir)
		if root == "" {
			root = defaultRoot
		}
		if root == "" {
			return
		}
		accs, err := accounts.List(ctx)
		if err != nil {
			log.Warn("strm 完成后失效 WebDAV 缓存失败：账号列表读取失败", "task_id", evt.TaskID, "err", err)
			return
		}
		invalidated := 0
		for _, acc := range accs {
			if acc.DriverType != "localfs" {
				continue
			}
			var cfg struct {
				RootPath string `json:"root_path"`
			}
			if json.Unmarshal([]byte(acc.Config), &cfg) != nil {
				continue
			}
			rootPath := strings.TrimSpace(cfg.RootPath)
			if rootPath == "" {
				continue
			}
			if pathsCover(resolvePath(rootPath), root) {
				cacheSvc.InvalidateAccount(acc.ID)
				invalidated++
			}
		}
		if invalidated > 0 {
			log.Info("strm 任务完成，已失效 WebDAV 缓存", "task_id", evt.TaskID, "strm_dir", root, "invalidated_accounts", invalidated)
		}
	})
}

// pathsCover 两条路径 clean 后互为前缀（含相等）即视为覆盖：
// root_path 是 strmDir 或其祖先（strm 新文件落在挂载区域内），
// 或 root_path 是 strmDir 的子路径（strm 新文件可能落在其下）。
func pathsCover(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	if a == b {
		return true
	}
	if strings.HasPrefix(a, b+string(filepath.Separator)) || strings.HasPrefix(b, a+string(filepath.Separator)) {
		return true
	}
	return false
}

// resolvePath 尝试归一符号链接，失败退回 Clean（目录可能尚不存在）。
func resolvePath(p string) string {
	p = filepath.Clean(p)
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return filepath.Clean(r)
	}
	return p
}
