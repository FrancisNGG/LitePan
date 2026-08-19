package rules

import (
	"testing"
)

func yearPtr(y int) *int { return &y }

func TestRenderTemplate_LegacyFormat(t *testing.T) {
	ctx := TemplateContext{}
	ctx.FromParsedMedia(ParsedMedia{
		Title:      "流浪地球",
		Year:       yearPtr(2019),
		Season:     yearPtr(1),
		Episode:    yearPtr(2),
		ScreenSize: "2160p",
		Edition:    "Theatrical",
	}, "The Wandering Earth", "12345")

	cases := []struct {
		name string
		tpl  string
		want string
	}{
		{"旧格式-文件夹", "Season {season:02d}", "Season 01"},
		{"旧格式-裸变量", "{title} ({year})", "流浪地球 (2019)"},
		{"旧格式-混合", "{title} S{season}E{episode}", "流浪地球 S1E2"},
		{"Jinja2-基础", "{{ title }} ({{ year }})", "流浪地球 (2019)"},
		{"Jinja2-en_title", "{{ en_title }} ({{ year }})", "The Wandering Earth (2019)"},
		{"Jinja2-条件", `{% if quality == "2160p" %}{{ title }} 4K{% else %}{{ title }}{% endif %}`, "流浪地球 4K"},
		{"Jinja2-条件否", `{% if quality == "1080p" %}{{ title }} 4K{% else %}{{ title }} HD{% endif %}`, "流浪地球 HD"},
		{"Jinja2-过滤器", "{{ title }} {{ year|default('') }}", "流浪地球 2019"},
		{"空模板", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := RenderTemplate(c.tpl, ctx)
			if err != nil {
				t.Fatalf("RenderTemplate(%q) 错误: %v", c.tpl, err)
			}
			if got != c.want {
				t.Errorf("RenderTemplate(%q) = %q, 期望 %q", c.tpl, got, c.want)
			}
		})
	}
}

func TestRenderTemplate_TMDBIDAndFilters(t *testing.T) {
	ctx := TemplateContext{}
	ctx.FromParsedMedia(ParsedMedia{
		Title:   "暗战",
		Year:    yearPtr(1999),
		Edition: "Criterion",
	}, "Running Out of Time", "9615")

	got, err := RenderTemplate(`{{ title }} ({{ year }}) [tmdb-{{ tmdb_id }}]{% if edition %} {{ edition }}{% endif %}`, ctx)
	if err != nil {
		t.Fatalf("渲染错误: %v", err)
	}
	want := "暗战 (1999) [tmdb-9615] Criterion"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestNormalizeTemplate(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Season {season:02d}", `Season {{ season|stringformat:"%02d" }}`},
		{"{title} ({year})", "{{ title }} ({{ year }})"},
		{"S{season}E{episode}", "S{{ season }}E{{ episode }}"},
	}
	for _, c := range cases {
		if got := normalizeTemplate(c.in); got != c.want {
			t.Errorf("normalizeTemplate(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestRenderTemplate_InvalidSyntax(t *testing.T) {
	ctx := TemplateContext{}
	_, err := RenderTemplate(`{{ title }`, ctx)
	if err == nil {
		t.Error("非法模板应返回错误")
	}
}
