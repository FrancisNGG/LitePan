// Package enhance 提供媒体整理增强能力（Jinja2 命名模板 + 自动分类）。
//
// 该包以「辅助工具」形态独立于主功能代码：由调用方（planner）根据
// mo_enhanced_enabled 开关决定是否构造 Enhancer；关闭时不构造（nil），
// 调用方走官方原始逻辑，行为与上游一致。本包不读取任何设置，
// 全部输入通过 EnhancerConfig 显式传入（对设置注册表零依赖）。
package enhance

import (
	"strings"

	"litepan/internal/mediaorganize/rules"
)

// EnhancerConfig 增强器配置（由调用方从设置翻译，本包不感知设置来源）。
type EnhancerConfig struct {
	MovieNamingTpl string
	TVNamingTpl    string
	CategoryRules  rules.CategoryRules
}

// Enhancer 收拢媒体整理增强逻辑：电影/电视剧命名模板 + 分类规则。
type Enhancer struct {
	movieNamingTpl string
	tvNamingTpl    string
	categoryRules  rules.CategoryRules
}

// New 从配置构建增强器（纯值对象，无副作用）。
func New(cfg EnhancerConfig) *Enhancer {
	return &Enhancer{
		movieNamingTpl: strings.TrimSpace(cfg.MovieNamingTpl),
		tvNamingTpl:    strings.TrimSpace(cfg.TVNamingTpl),
		categoryRules:  cfg.CategoryRules,
	}
}

// NamingTplFor 按媒体类型返回完整路径命名模板（未配置返回空）。
func (e *Enhancer) NamingTplFor(isTV bool) string {
	if isTV {
		return e.tvNamingTpl
	}
	return e.movieNamingTpl
}

// CategoryRules 返回分类规则（供调用方做配置诊断，如不生效条件提示）。
func (e *Enhancer) CategoryRules() rules.CategoryRules {
	return e.categoryRules
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

// SeasonFolderName 季目录名：优先用 TV 全局模板渲染路径中含季号的目录段
// （模板可含剧名/类型/分类等段，季段由 IsSeasonDirName 识别，不依赖位置），
// 否则用任务级季目录模板（legacy 字符串替换）。
func (e *Enhancer) SeasonFolderName(season *int, fallbackTpl string) string {
	if strings.TrimSpace(e.tvNamingTpl) != "" && season != nil {
		parsed := rules.ParsedMedia{Season: season}
		if dirs, _, ok := e.ResolveNamingParts(true, parsed, "", "", "", ""); ok {
			for _, d := range dirs {
				if rules.IsSeasonDirName(d) {
					if name := rules.SanitizeFilename(d); name != "" {
						return name
					}
				}
			}
		}
	}
	return rules.BuildSeasonFolderName(season, fallbackTpl)
}

// MatchCategory 按 TMDB 原始数据匹配分类目录名（未命中返回空）。
func (e *Enhancer) MatchCategory(isTV bool, tmdbInfo map[string]any) string {
	return e.categoryRules.MatchCategory(isTV, tmdbInfo)
}
