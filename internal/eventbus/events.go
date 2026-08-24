package eventbus

import "time"

// 跨模块事件契约；发布者不感知订阅者。

// AccountAuthFailed 账号认证失败。Fatal 表示需要用户重新授权而非自动恢复。
type AccountAuthFailed struct {
	AccountID int64
	Reason    string
	Fatal     bool
}

// AccountAuthRecovered 账号认证恢复可用。
type AccountAuthRecovered struct {
	AccountID int64
}

// FileMutated 文件发生增删改，用于触发缓存失效等。
type FileMutated struct {
	AccountID   int64
	Op          string // create/delete/move/rename/copy/upload
	ParentID    string
	FileID      string
	FileName    string
	FileSize    int64
	IsDir       bool
	ModTime     time.Time
	FileIDs     []string
	OldParentID string
}

// OfflineDownloadCompleted 离线下载任务首次进入成功状态。
type OfflineDownloadCompleted struct {
	TaskID            string
	AccountID         int64
	TargetParentID    string
	TargetDisplayPath string
	FileID            string
	FileName          string
}

// NotificationCreated 新通知产生。
type NotificationCreated struct {
	Level     string
	Category  string
	Title     string
	Message   string
	AccountID int64
	RefID     int64
}

// StrmScanCompleted STRM 任务成功完成。strm 文件由 os.WriteFile 直写文件系统，
// 不经过 file.Service 事件总线，WebDAV 缓存无法感知；订阅者（app 层）据此
// 失效指向 strmDir 的 localfs 账号缓存，保证客户端立即可见。
// 发布者（strm）不感知订阅者，也不感知账号/缓存细节。
type StrmScanCompleted struct {
	TaskID    int64
	AccountID int64
	StrmDir   string
}
