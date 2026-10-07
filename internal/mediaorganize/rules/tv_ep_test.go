package rules

import "testing"

// 回归：目录名含嵌入的集数范围（如 Ep01-60）时应识别为剧集，而非电影。
func TestRedSorghumEpRangeRecognizedAsTV(t *testing.T) {
	filename := "红高粱（未删减版）.Red.Sorghum.Ep01.2014.HD1080P.X264.AAC.Mandarin.CHS.Mp4Ba.mp4"
	dirname := "红高粱（未删减版）.全集.Red.Sorghum.Ep01-60.2014.HD1080P.X264.AAC.Mandarin.CHS.Mp4Ba"

	rawFileParsed := NormalizeParsedMedia(ParseFilenameStrict(filename))
	fileParsed := rawFileParsed
	ancestors := []Ancestor{{ID: "3534467323467401056", Name: dirname}}

	dirParsed := NormalizeParsedMedia(ParseDirName(dirname))
	fileParsed = MergeThreeLayerParsed(fileParsed, dirParsed, ParsedMedia{})
	fileParsed = PrepareTVFileParsed(fileParsed, ancestors)

	if tvRule := LooksLikeTVFileWithName(fileParsed, ancestors, filename); !tvRule.Matched {
		t.Fatalf("「红高粱 Ep01-60」应识别为剧集，但 LooksLikeTVFileWithName 返回 Matched=false: %+v", tvRule)
	}
	if !hasTVHintAncestor(ancestors) {
		t.Fatalf("hasTVHintAncestor 应识别目录名中的 Ep01-60 范围标记")
	}
}

func TestHasEmbeddedEpisodeRangeToken(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"红高粱（未删减版）.全集.Red.Sorghum.Ep01-60.2014.HD1080P", true},
		{"某剧.E01-24.1080p", true},
		{"某剧.第1-60集.2014", false}, // 中文范围目前靠 IsEpisodeRangeDirName/ParseDirName 处理
		{"1999-2003 纪录片合集", false}, // 年份范围
		{"720-1080 分辨率", false},      // 分辨率
		{"某电影.2014.HD1080P", false},  // 无范围标记
		{"", false},
	}
	for _, c := range cases {
		if got := HasEmbeddedEpisodeRangeToken(c.name); got != c.want {
			t.Errorf("HasEmbeddedEpisodeRangeToken(%q) = %v, want %v", c.name, got, c.want)
		}
	}
}
