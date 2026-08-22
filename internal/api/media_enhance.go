package api

import (
	"net/http"
	"strconv"

	"litepan/internal/settings"
)

// getMediaEnhanceStatus 返回「媒体整理增强」卡片状态（开关 + 服务可用性）。
func (h *Handler) getMediaEnhanceStatus(w http.ResponseWriter, r *http.Request) {
	enabled := false
	if h.settings != nil {
		enabled = h.settings.Bool(settings.KeyMOEnhancedEnabled)
	}
	writeOK(w, map[string]any{
		"enabled":   enabled,
		"available": h.mediaOrganize != nil,
	})
}

// setMediaEnhanceEnabled 开启/关闭「媒体整理增强」。
func (h *Handler) setMediaEnhanceEnabled(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Enabled bool `json:"enabled"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeErr(w, err)
		return
	}
	if h.settings != nil {
		if err := h.settings.Update(r.Context(), map[string]string{
			settings.KeyMOEnhancedEnabled: strconv.FormatBool(in.Enabled),
		}); err != nil {
			writeErr(w, err)
			return
		}
	}
	writeOK(w, map[string]any{"enabled": in.Enabled})
}
