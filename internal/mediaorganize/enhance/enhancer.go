// Package enhance 提供媒体整理增强能力（Jinja2 命名模板 + 自动分类）。
//
// 该包以「辅助工具」形态独立于主功能代码：由 mo_enhanced_enabled 开关控制，
// 关闭时 NewFromSettings 返回 nil，调用方走官方原始逻辑，行为与上游一致。
package enhance

import (
	"strings"

	"litepan/internal/mediaorganize/rules"
)

// Enhancer 收拢媒体整理增强逻辑：电影/电视剧命名模板 + 分类规则。
type Enhancer struct {
	movieNamingTpl string
	tvNamingTpl    string
	categoryRules  rules.CategoryRules
}

// NewFromSettings 从设置构建增强器；开关未开启时返回 nil（=增强关闭，官方行为）。
func NewFromSettings(settings map[string]any) *Enhancer {
	if !rules.SettingBool(settings["mo_enhanced_enabled"], false) {
		return nil
	}
	return &Enhancer{
		movieNamingTpl: strSetting(settings, "mo_movie_naming_format", ""),
		tvNamingTpl:    strSetting(settings, "mo_tv_naming_format", ""),
		categoryRules:  rules.ParseCategoryRules(strSetting(settings, "mo_category_map", "")),
	}
}

func strSetting(settings map[string]any, key, fallback string) string {
	if v, ok := settings[key]; ok {
		if s := strings.TrimSpace(toString(v)); s != "" {
			return s
		}
	}
	return fallback
}

func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// NamingTplFor 按媒体类型返回完整路径命名模板（未配置返回空）。
func (e *Enhancer) NamingTplFor(isTV bool) string {
	if isTV {
		return e.tvNamingTpl
	}
	return e.movieNamingTpl
}

// ResolveNamingParts 渲染完整路径模板，拆分为目录段和文件名段。
// ext 为不含点的扩展名（如 mkv），会映射到模板的 fileExt 变量。
func (e *Enhancer) ResolveNamingParts(isTV bool, parsed rules.ParsedMedia, enTitle, tmdbID, ext, originalName string) (dirs []string, filename string, ok bool) {
	tpl := e.NamingTplFor(isTV)
	if strings.TrimSpace(tpl) == "" {
		return nil, "", false
	}
	ctx := rules.TemplateContext{}
	ctx.FromParsedMedia(parsed, enTitle, tmdbID)
	if ext != "" {
		ctx.FileExt = "." + ext
	}
	ctx.OriginalName = originalName
	out, err := rules.RenderTemplate(tpl, ctx)
	if err != nil {
		return nil, "", false
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return nil, "", false
	}
	parts := strings.Split(out, "/")
	if len(parts) == 0 {
		return nil, "", false
	}
	return parts[:len(parts)-1], parts[len(parts)-1], true
}

// SeasonFolderName 季目录名：优先用 TV 全局模板渲染结果的目录段第二段（如 "Season 1"），
// 否则用任务级季目录模板。
func (e *Enhancer) SeasonFolderName(season *int, fallbackTpl string) string {
	if strings.TrimSpace(e.tvNamingTpl) != "" && season != nil {
		parsed := rules.ParsedMedia{Season: season}
		if dirs, _, ok := e.ResolveNamingParts(true, parsed, "", "", "", ""); ok && len(dirs) > 1 {
			name := rules.SanitizeFilename(dirs[1])
			if name != "" {
				return name
			}
		}
	}
	return rules.BuildSeasonFolderNameTpl(season, "", fallbackTpl)
}

// MatchCategory 按 TMDB 原始数据匹配分类目录名（未命中返回空）。
func (e *Enhancer) MatchCategory(isTV bool, tmdbInfo map[string]any) string {
	return e.categoryRules.MatchCategory(isTV, tmdbInfo)
}
