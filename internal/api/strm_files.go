package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// strmFileEntry 是 STRM 文件列表中的一个条目。
type strmFileEntry struct {
	Name    string `json:"name"`
	Path    string `json:"path"` // 相对 STRM 根目录的路径（斜杠分隔）
	Size    int64  `json:"size"`
	ModTime string `json:"mod_time"`
	IsDir   bool   `json:"is_dir"`
}

// listStrmFiles 返回 STRM 输出目录下的文件树（相对路径列表）。
// 可选查询参数：?dir=<相对目录> 只看某个子目录；?recurse=1 递归全部。
func (h *Handler) listStrmFiles(w http.ResponseWriter, r *http.Request) {
	if !ensureServiceReady(w, h.strm != nil) {
		return
	}
	root := strings.TrimSpace(h.strm.StrmDir())
	if root == "" {
		writeErr(w, fmt.Errorf("STRM 输出目录未配置"))
		return
	}
	sub := strings.Trim(strings.TrimSpace(r.URL.Query().Get("dir")), "/")
	recurse := r.URL.Query().Get("recurse") == "1"

	base := filepath.Clean(root)
	target := base
	if sub != "" {
		target = filepath.Join(base, filepath.FromSlash(sub))
		if !strings.HasPrefix(filepath.Clean(target), base+string(filepath.Separator)) && filepath.Clean(target) != base {
			writeErr(w, fmt.Errorf("非法路径"))
			return
		}
	}

	entries, err := collectStrmFiles(base, target, recurse)
	if err != nil {
		writeErr(w, err)
		return
	}
	if entries == nil {
		entries = []strmFileEntry{}
	}
	writeOK(w, entries)
}

// collectStrmFiles 递归收集目录下的文件。
func collectStrmFiles(root, target string, recurse bool) ([]strmFileEntry, error) {
	infos, err := os.ReadDir(target)
	if err != nil {
		if os.IsNotExist(err) {
			return []strmFileEntry{}, nil
		}
		return nil, err
	}
	out := make([]strmFileEntry, 0, len(infos))
	for _, info := range infos {
		abs := filepath.Join(target, info.Name())
		rel, _ := filepath.Rel(root, abs)
		relSlash := filepath.ToSlash(rel)
		entry := strmFileEntry{
			Name:  info.Name(),
			Path:  relSlash,
			IsDir: info.IsDir(),
		}
		if fi, err := info.Info(); err == nil {
			entry.Size = fi.Size()
			entry.ModTime = fi.ModTime().Format("2006-01-02 15:04:05")
		}
		out = append(out, entry)
		if info.IsDir() && recurse {
			children, err := collectStrmFiles(root, abs, recurse)
			if err != nil {
				return nil, err
			}
			out = append(out, children...)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].IsDir != out[j].IsDir {
			return out[i].IsDir
		}
		return out[i].Path < out[j].Path
	})
	return out, nil
}

// deleteStrmFile 删除 STRM 输出目录下的文件或目录。
// 请求体：{"path": "相对路径"}。仅允许删除根目录范围内的内容。
func (h *Handler) deleteStrmFile(w http.ResponseWriter, r *http.Request) {
	if !ensureServiceReady(w, h.strm != nil) {
		return
	}
	var req struct {
		Path string `json:"path"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, err)
		return
	}
	root := strings.TrimSpace(h.strm.StrmDir())
	if root == "" {
		writeErr(w, fmt.Errorf("STRM 输出目录未配置"))
		return
	}
	base := filepath.Clean(root)
	target := filepath.Join(base, filepath.FromSlash(strings.Trim(strings.TrimSpace(req.Path), "/")))
	if !strings.HasPrefix(filepath.Clean(target), base+string(filepath.Separator)) || filepath.Clean(target) == base {
		writeErr(w, fmt.Errorf("非法路径"))
		return
	}
	if err := os.RemoveAll(target); err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, map[string]any{"deleted": req.Path})
}
