package rules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/flosch/pongo2/v6"
)

// TemplateContext 提供给命名模板的变量集（MoviePilot/Jinja2 风格，变量名与 MoviePilot 对齐）
type TemplateContext struct {
	Title         string // 本地化标题（TMDB title，可能为中文）
	EnTitle       string // 英文原标题（TMDB original_title）
	Year          *int
	Season        *int
	Episode       *int
	ScreenSize    string // 分辨率，如 2160p / 4K（MoviePilot: videoFormat）
	FrameRate     string // 帧率，如 60fps
	VideoCodec    string
	AudioCodec    string
	AudioChannels string
	AudioEffect   string // Atmos 等音频特效
	VideoBit      string // 色深，如 10bit（MoviePilot: video_bit）
	Source        string // 来源，如 BluRay / WEB-DL
	ReleaseGroup  string
	Edition       string
	WebSource     string // 流媒体平台，如 Netflix（MoviePilot: web_source）
	Type          string // movie / tv
	TMDBID        string
	Part          string // 分集/Part 信息（MoviePilot: part）
	FileExt       string // 文件扩展名，如 .mkv（MoviePilot: fileExt）
	OriginalName  string // 原文件名（MoviePilot: original_name）
}

// seasonEpisode 生成 MoviePilot 风格的 season_episode（如 S01E02）
func (c TemplateContext) seasonEpisode() string {
	if c.Season == nil {
		return ""
	}
	if c.Episode == nil {
		return fmt.Sprintf("S%02d", *c.Season)
	}
	return fmt.Sprintf("S%02dE%02d", *c.Season, *c.Episode)
}

// intOrNil 解引用 *int，nil 保持 nil（避免 pongo2 stringformat 打印指针地址）
func intOrNil(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

func (c TemplateContext) toPongo2Context() pongo2.Context {
	// en_title 去重：与标题相同，或标题已包含英文名（如 displayTitle 拼好的“疯狂动物城2 - Zootopia 2”）时置空
	enTitle := c.EnTitle
	if enTitle != "" {
		t := strings.TrimSpace(c.Title)
		e := strings.TrimSpace(enTitle)
		if t == "" || strings.EqualFold(e, t) {
			enTitle = ""
		} else if len(e) >= 2 && strings.Contains(strings.ToLower(t), strings.ToLower(e)) {
			enTitle = ""
		}
	}
	// MoviePilot 语义：videoCodec = 编码 + 色深（如 H265 10bit）；audioCodec = 编码 + 声道 + 特效（如 AAC 2.0 / DDP 5.1 Atmos）
	videoCodec := c.VideoCodec
	if videoCodec != "" && c.VideoBit != "" {
		videoCodec = videoCodec + " " + c.VideoBit
	}
	audioCodec := c.AudioCodec
	if audioCodec != "" && c.AudioChannels != "" {
		audioCodec = audioCodec + " " + c.AudioChannels
	}
	if audioCodec != "" && c.AudioEffect != "" {
		audioCodec = audioCodec + " " + c.AudioEffect
	}
	ctx := pongo2.Context{
		// MoviePilot 兼容变量名（可直接复制 MoviePilot 模板）
		"title":           c.Title,
		"name":            c.Title,
		"original_title":  enTitle,
		"originalTitle":   enTitle,
		"en_title":        enTitle,
		"en_name":         enTitle,
		"original_name":   c.OriginalName,
		"year":            intOrNil(c.Year),
		"season":          intOrNil(c.Season),
		"episode":         intOrNil(c.Episode),
		"season_episode":  c.seasonEpisode(),
		"season_fmt":      c.seasonEpisode(),
		"videoFormat":     c.ScreenSize,
		"videoCodec":      videoCodec,
		"audioCodec":      audioCodec,
		"audioChannels":   c.AudioChannels,
		"videoBit":        c.VideoBit,
		"video_bit":       c.VideoBit,
		"releaseGroup":    c.ReleaseGroup,
		"resourceType":    c.Source,
		"webSource":       c.WebSource,
		"effect":          c.Edition,
		"edition":         c.Edition,
		"resource_term":   c.Source,
		"fps":             strings.TrimSuffix(c.FrameRate, "fps"),
		"fps_text":        c.FrameRate,
		"fileExt":         c.FileExt,
		"part":            c.Part,
		"tmdbid":          c.TMDBID,
		"tmdb_id":         c.TMDBID,
		// LitePan 原有变量名（向后兼容）
		"quality":        c.ScreenSize,
		"screen_size":    c.ScreenSize,
		"frame_rate":     c.FrameRate,
		"video_codec":    videoCodec,
		"audio_codec":    audioCodec,
		"audio_channels": c.AudioChannels,
		"source":         c.Source,
		"release_group":  c.ReleaseGroup,
		"type":           c.Type,
	}
	return ctx
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
	c.VideoBit = p.VideoBit
	c.AudioEffect = p.AudioEffect
	c.Source = p.Source
	c.ReleaseGroup = p.ReleaseGroup
	c.Edition = p.Edition
	c.WebSource = p.WebSource
	c.Type = p.Type
	c.TMDBID = tmdbID
}

// legacyVarRe 匹配旧格式占位符：{title} / {season:02d}
var legacyVarRe = regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z0-9_]*)(?::([0-9]+)d)?\}`)

// jinjaBlockRe 匹配 Jinja2 的 {{...}} / {%...%} 块（转换旧格式前先保护，避免误伤 MoviePilot 模板）
var jinjaBlockRe = regexp.MustCompile(`\{\{.*?\}\}|\{%.*?%\}`)

// jinjaFilterCallRe 匹配 Jinja2 括号形式过滤器：|default('x') 或 |default("x")
// pongo2 只支持冒号形式 |default:'x'，需转换以兼容 MoviePilot 模板
var jinjaFilterCallRe = regexp.MustCompile(`\|([a-zA-Z_][a-zA-Z0-9_]*)\(([^)]*)\)`)

// jinjaMethodCallRe 匹配 Jinja2 方法调用语法：var.lower() / var.strip() 等
// MoviePilot 用 Python Jinja2 支持方法调用；pongo2 不支持，需转成过滤器
var jinjaMethodCallRe = regexp.MustCompile(`([a-zA-Z_][a-zA-Z0-9_]*)\.(lower|upper|title|capitalize|strip|lstrip|rstrip|trim)\(\)`)

// methodCallToFilter 把 Jinja2 方法调用转成 pongo2 过滤器：original_name.lower() -> original_name|lower
func methodCallToFilter(m string) string {
	parts := jinjaMethodCallRe.FindStringSubmatch(m)
	if len(parts) != 3 {
		return m
	}
	f := parts[2]
	if f == "strip" || f == "lstrip" || f == "rstrip" {
		f = "trim"
	}
	return parts[1] + "|" + f
}

// jinjaOrRe 匹配 Jinja2 的 or 表达式：{{ a or b or '默认' }}
// pongo2 的 or 返回布尔值而非第一个真值，需转换为 if/elif/else
var jinjaOrRe = regexp.MustCompile(`\{\{\s*([^}]*?)\s*\}\}`)

// convertJinjaOr 将 {{ a or b or 'x' }} 转为 {% if a %}{{ a }}{% elif b %}{{ b }}{% else %}x{% endif %}
func convertJinjaOr(tpl string) string {
	return jinjaOrRe.ReplaceAllStringFunc(tpl, func(m string) string {
		inner := strings.TrimSpace(m[2 : len(m)-2])
		if !strings.Contains(inner, " or ") {
			return m
		}
		parts := strings.Split(inner, " or ")
		var sb strings.Builder
		for i, p := range parts {
			p = strings.TrimSpace(p)
			if i == 0 {
				sb.WriteString("{% if ")
				sb.WriteString(p)
				sb.WriteString(" %}")
				sb.WriteString("{{ ")
				sb.WriteString(p)
				sb.WriteString(" }}")
			} else if i < len(parts)-1 {
				sb.WriteString("{% elif ")
				sb.WriteString(p)
				sb.WriteString(" %}")
				sb.WriteString("{{ ")
				sb.WriteString(p)
				sb.WriteString(" }}")
			} else {
				// 末位可能是字面量 'Unknown Artist'：包一层 {{ }} 避免引号原样输出
				sb.WriteString("{% else %}")
				sb.WriteString("{{ ")
				sb.WriteString(p)
				sb.WriteString(" }}")
				sb.WriteString("{% endif %}")
			}
		}
		return sb.String()
	})
}

// normalizeTemplate 将旧格式 {var} / {var:02d} 转换为 Jinja2/pongo2 语法，
// 并将 Jinja2 括号形式过滤器转换为 pongo2 冒号形式，or 表达式转换为 if/elif/else
func normalizeTemplate(tpl string) string {
	// 1. 保护 Jinja2 块（{{...}} / {%...%}），避免被旧格式正则误伤（Go regexp 无 lookbehind）
	var blocks []string
	ph := "\x00"
	tpl = jinjaBlockRe.ReplaceAllStringFunc(tpl, func(m string) string {
		idx := len(blocks)
		blocks = append(blocks, m)
		return ph + fmt.Sprintf("%d", idx) + ph
	})
	// 2. 旧格式占位符转换（此时剩余 {var} 均为旧格式）
	tpl = legacyVarRe.ReplaceAllStringFunc(tpl, func(m string) string {
		parts := legacyVarRe.FindStringSubmatch(m)
		name := parts[1]
		if parts[2] != "" {
			// {season:02d} -> {{ season|stringformat:"%02d" }}
			return fmt.Sprintf(`{{ %s|stringformat:"%%%sd" }}`, name, parts[2])
		}
		return fmt.Sprintf(`{{ %s }}`, name)
	})
	// 3. 还原 Jinja2 块
	for i, b := range blocks {
		tpl = strings.ReplaceAll(tpl, ph+fmt.Sprintf("%d", i)+ph, b)
	}
	// 3.5 Jinja2 方法调用转过滤器：original_name.lower() -> original_name|lower
	tpl = jinjaMethodCallRe.ReplaceAllStringFunc(tpl, methodCallToFilter)
	// 4. |default('x') -> |default:'x'  (Jinja2 -> pongo2)
	tpl = jinjaFilterCallRe.ReplaceAllString(tpl, `|$1:$2`)
	// 5. {{ a or b or 'x' }} -> {% if a %}...{% endif %}  (Jinja2 or -> pongo2 兼容)
	tpl = convertJinjaOr(tpl)
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
