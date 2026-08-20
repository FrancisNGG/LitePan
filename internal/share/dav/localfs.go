package dav

import (
	"context"
	"html"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
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

// serveLocalRead 处理本地目录模式的 GET/HEAD：目录返回 HTML 列表，文件直接读盘返回。
func (s *Server) serveLocalRead(w http.ResponseWriter, r *http.Request, root string) bool {
	name := strings.TrimPrefix(pathClean(r.URL.Path), "/")
	full := filepath.Join(root, filepath.FromSlash(name))
	info, err := os.Stat(full)
	if err != nil {
		return false
	}
	if r.Method == http.MethodHead {
		if info.IsDir() {
			w.Header().Set("Content-Type", "httpd/unix-directory")
		} else {
			w.Header().Set("Content-Type", mime.TypeByExtension(filepath.Ext(info.Name())))
			w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
		}
		w.WriteHeader(http.StatusOK)
		return true
	}
	if info.IsDir() {
		s.serveLocalDirListing(w, r, root, name)
		return true
	}
	http.ServeFile(w, r, full)
	return true
}

func (s *Server) serveLocalDirListing(w http.ResponseWriter, r *http.Request, root, name string) {
	dir := filepath.Join(root, filepath.FromSlash(name))
	entries, err := os.ReadDir(dir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	base := "/" + name
	if base == "/" {
		base = "/"
	}
	var b strings.Builder
	b.WriteString("<!DOCTYPE html><html><head><meta charset=\"utf-8\"><title>Index of ")
	b.WriteString(html.EscapeString(base))
	b.WriteString("</title></head><body><h1>Index of ")
	b.WriteString(html.EscapeString(base))
	b.WriteString("</h1><ul>")
	if name != "" {
		parent := parentPath(base)
		b.WriteString(`<li><a href="`)
		b.WriteString(html.EscapeString(publicHref(parent)))
		b.WriteString(`">../</a></li>`)
	}
	for _, e := range entries {
		display := e.Name()
		hrefName := display
		if e.IsDir() {
			display += "/"
			hrefName += "/"
		}
		href := publicHref(joinHref(base, hrefName))
		b.WriteString(`<li><a href="`)
		b.WriteString(html.EscapeString(href))
		b.WriteString(`">`)
		b.WriteString(html.EscapeString(display))
		b.WriteString("</a></li>")
	}
	b.WriteString("</ul></body></html>")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}
