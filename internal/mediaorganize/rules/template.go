package rules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/flosch/pongo2/v6"
)

// TemplateContext 提供给命名模板的变量集（MoviePilot/Jinja2 风格）
type TemplateContext struct {
	Title         string // 本地化标题（TMDB title，可能为中文）
	EnTitle       string // 英文原标题（TMDB original_title）
	Year          *int
	Season        *int
	Episode       *int
	ScreenSize    string // 分辨率，如 2160p / 4K
	FrameRate     string // 帧率，如 60fps
	VideoCodec    string
	AudioCodec    string
	AudioChannels string
	Source        string // 来源，如 BluRay / WEB-DL
	ReleaseGroup  string
	Edition       string
	Type          string // movie / tv
	TMDBID        string
}

// intOrNil 解引用 *int，nil 保持 nil（避免 pongo2 stringformat 打印指针地址）
func intOrNil(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

func (c TemplateContext) toPongo2Context() pongo2.Context {
	return pongo2.Context{
		"title":          c.Title,
		"en_title":       c.EnTitle,
		"year":           intOrNil(c.Year),
		"season":         intOrNil(c.Season),
		"episode":        intOrNil(c.Episode),
		"quality":        c.ScreenSize,
		"screen_size":    c.ScreenSize,
		"frame_rate":     c.FrameRate,
		"video_codec":    c.VideoCodec,
		"audio_codec":    c.AudioCodec,
		"audio_channels": c.AudioChannels,
		"source":         c.Source,
		"release_group":  c.ReleaseGroup,
		"edition":        c.Edition,
		"type":           c.Type,
		"tmdb_id":        c.TMDBID,
	}
}

// FromParsedMedia 从 ParsedMedia + 附加信息构建模板上下文
func (c *TemplateContext) FromParsedMedia(p ParsedMedia, enTitle, tmdbID string) {
	c.Title = p.Title
	c.EnTitle = enTitle
	c.Year = p.Year
	c.Season = p.Season
	c.Episode = p.Episode
	c.ScreenSize = p.ScreenSize
	c.FrameRate = p.FrameRate
	c.VideoCodec = p.VideoCodec
	c.AudioCodec = p.AudioCodec
	c.AudioChannels = p.AudioChannels
	c.Source = p.Source
	c.ReleaseGroup = p.ReleaseGroup
	c.Edition = p.Edition
	c.Type = p.Type
	c.TMDBID = tmdbID
}

// legacyVarRe 匹配旧格式占位符：{title} / {season:02d}
var legacyVarRe = regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z0-9_]*)(?::([0-9]+)d)?\}`)

// jinjaFilterCallRe 匹配 Jinja2 括号形式过滤器：|default('x') 或 |default("x")
// pongo2 只支持冒号形式 |default:'x'，需转换以兼容 MoviePilot 模板
var jinjaFilterCallRe = regexp.MustCompile(`\|([a-zA-Z_][a-zA-Z0-9_]*)\(([^)]*)\)`)

// normalizeTemplate 将旧格式 {var} / {var:02d} 转换为 Jinja2/pongo2 语法，
// 并将 Jinja2 括号形式过滤器转换为 pongo2 冒号形式
func normalizeTemplate(tpl string) string {
	tpl = legacyVarRe.ReplaceAllStringFunc(tpl, func(m string) string {
		parts := legacyVarRe.FindStringSubmatch(m)
		name := parts[1]
		if parts[2] != "" {
			// {season:02d} -> {{ season|stringformat:"%02d" }}
			return fmt.Sprintf(`{{ %s|stringformat:"%%%sd" }}`, name, parts[2])
		}
		return fmt.Sprintf(`{{ %s }}`, name)
	})
	// |default('x') -> |default:'x'  (Jinja2 -> pongo2)
	tpl = jinjaFilterCallRe.ReplaceAllString(tpl, `|$1:$2`)
	return tpl
}

// RenderTemplate 用 pongo2 (Jinja2 兼容) 渲染命名模板。
// 自动兼容旧格式 {title} / {season:02d}。
// 空模板返回空字符串；渲染失败返回错误。
func RenderTemplate(tpl string, ctx TemplateContext) (string, error) {
	tpl = strings.TrimSpace(tpl)
	if tpl == "" {
		return "", nil
	}
	normalized := normalizeTemplate(tpl)
	tmpl, err := pongo2.FromString(normalized)
	if err != nil {
		return "", fmt.Errorf("模板语法错误: %w", err)
	}
	out, err := tmpl.Execute(ctx.toPongo2Context())
	if err != nil {
		return "", fmt.Errorf("模板渲染失败: %w", err)
	}
	return strings.TrimSpace(out), nil
}
