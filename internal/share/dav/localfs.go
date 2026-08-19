package dav

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/net/webdav"
)

// localFileSystem 是 WebDAV 的本地目录实现。
// 当配置了 webdav_root 设置项时，/dav 直接暴露该本地目录
// （典型用途：把 STRM 输出目录挂给 Infuse/VidHub 等客户端读取）。
type localFileSystem struct {
	root string
}

func (fs *localFileSystem) join(name string) string {
	clean := filepath.FromSlash(strings.TrimPrefix(strings.TrimSpace(name), "/"))
	return filepath.Join(fs.root, clean)
}

func (fs *localFileSystem) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	return os.MkdirAll(fs.join(name), perm)
}

func (fs *localFileSystem) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	return os.OpenFile(fs.join(name), flag, perm)
}

func (fs *localFileSystem) RemoveAll(ctx context.Context, name string) error {
	return os.RemoveAll(fs.join(name))
}

func (fs *localFileSystem) Rename(ctx context.Context, oldName, newName string) error {
	return os.Rename(fs.join(oldName), fs.join(newName))
}

func (fs *localFileSystem) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	return os.Stat(fs.join(name))
}

var _ webdav.FileSystem = (*localFileSystem)(nil)
