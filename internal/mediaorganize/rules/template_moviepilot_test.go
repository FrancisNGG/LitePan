package rules

import (
	"testing"
)

// 用 MoviePilot 的默认 RENAME_FORMAT 直接验证兼容性（目标：配置复制过来直接用）
func TestRenderTemplate_MoviePilotMovieFormat(t *testing.T) {
	ctx := TemplateContext{}
	ctx.FromParsedMedia(ParsedMedia{
		Title:      "流浪地球",
		Year:       yearPtr(2019),
		ScreenSize: "2160p",
	}, "The Wandering Earth", "12345")
	ctx.FileExt = ".mkv"

	// MoviePilot MOVIE_RENAME_FORMAT
	tpl := `{{title}}{% if year %} ({{year}}){% endif %}/{{title}}{% if year %} ({{year}}){% endif %}{% if part %}-{{part}}{% endif %}{% if videoFormat %} - {{videoFormat}}{% endif %}{{fileExt}}`
	got, err := RenderTemplate(tpl, ctx)
	if err != nil {
		t.Fatalf("MoviePilot 电影模板渲染失败: %v", err)
	}
	want := "流浪地球 (2019)/流浪地球 (2019) - 2160p.mkv"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderTemplate_MoviePilotTVFormat(t *testing.T) {
	ctx := TemplateContext{}
	ctx.FromParsedMedia(ParsedMedia{
		Title:   "三体",
		Year:    yearPtr(2023),
		Season:  yearPtr(1),
		Episode: yearPtr(2),
	}, "Three-Body", "45678")
	ctx.FileExt = ".mkv"

	// MoviePilot TV_RENAME_FORMAT
	tpl := `{{title}}{% if year %} ({{year}}){% endif %}/Season {{season}}/{{title}} - {{season_episode}}{% if part %}-{{part}}{% endif %}{% if episode %} - 第 {{episode}} 集{% endif %}{{fileExt}}`
	got, err := RenderTemplate(tpl, ctx)
	if err != nil {
		t.Fatalf("MoviePilot 剧集模板渲染失败: %v", err)
	}
	want := "三体 (2023)/Season 1/三体 - S01E02 - 第 2 集.mkv"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderTemplate_MoviePilotMusicFormat(t *testing.T) {
	ctx := TemplateContext{}
	ctx.Title = "爱在西元前"
	ctx.Year = yearPtr(2001)
	ctx.FileExt = ".flac"

	// MoviePilot MUSIC_RENAME_FORMAT（含 or 表达式，验证 convertJinjaOr）
	tpl := `{{album_artist or artist or 'Unknown Artist'}}/{{album or 'Unknown Album'}}{% if year %} ({{year}}){% endif %}/{% if track %}{{track}} - {% endif %}{{title}}{{fileExt}}`
	got, err := RenderTemplate(tpl, ctx)
	if err != nil {
		t.Fatalf("MoviePilot 音乐模板渲染失败: %v", err)
	}
	want := "Unknown Artist/Unknown Album (2001)/爱在西元前.flac"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	// 有 artist 时走 or 的第二分支
	ctx2 := ctx
	ctx2.EnTitle = "" // 无关
	// 直接构造：把 artist 放进 context 需要额外字段——用 or 表达式验证 first truthy
	got2, err := RenderTemplate(`{{album_artist or artist or 'Unknown Artist'}}`, TemplateContext{})
	if err != nil {
		t.Fatal(err)
	}
	if got2 != "Unknown Artist" {
		t.Errorf("or 表达式全空时 got %q, want %q", got2, "Unknown Artist")
	}
}

func TestRenderTemplate_MoviePilotVariableAliases(t *testing.T) {
	ctx := TemplateContext{}
	ctx.FromParsedMedia(ParsedMedia{
		Title:      "暗战",
		Year:       yearPtr(1999),
		Season:     yearPtr(1),
		Episode:    yearPtr(3),
		ScreenSize: "1080p",
	}, "Running Out of Time", "9615")
	ctx.Part = "B"
	ctx.FileExt = ".mkv"

	// 验证 MoviePilot 变量名全部可用
	tpl := `{{title}}|{{original_title}}|{{year}}|{{season}}|{{episode}}|{{season_episode}}|{{videoFormat}}|{{fileExt}}|{{part}}|{{tmdbid}}`
	got, err := RenderTemplate(tpl, ctx)
	if err != nil {
		t.Fatalf("变量别名渲染失败: %v", err)
	}
	want := "暗战|Running Out of Time|1999|1|3|S01E03|1080p|.mkv|B|9615"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
