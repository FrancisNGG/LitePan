package rules

import (
	"testing"
)

func TestDVEffectRecognition(t *testing.T) {
	cases := []struct {
		name string
		want string // 期望 edition 字段（effect 存这里）
	}{
		{"某剧.S01E01.2160p.DV.HEVC.DDP5.1.mkv", "DoVi"},
		{"某剧.S01E01.2160p.Dolby.Vision.HEVC.DDP5.1.mkv", "DoVi"},
		{"某剧.S01E01.2160p.DoVi.HEVC.mkv", "DoVi"},
		{"某剧.S01E01.2160p.HDR10+.HEVC.mkv", "HDR10+"},
		{"某剧.S01E01.2160p.HDR10.HEVC.mkv", "HDR10"},
		{"某电影.2014.DVDRip.x264.AC3.mkv", ""}, // DVDRip 不得被误识别为 DV
		{"某电影.2014.BluRay.1080p.x264.mkv", ""},
	}
	for _, c := range cases {
		m := map[string]any{}
		EnrichMediaTagsFromFilename(c.name, m)
		got := ""
		if v, ok := m["edition"]; ok && v != nil {
			got = v.(string)
		}
		if got != c.want {
			t.Errorf("EnrichMediaTagsFromFilename(%q) edition=%q, want %q", c.name, got, c.want)
		}
	}
}
