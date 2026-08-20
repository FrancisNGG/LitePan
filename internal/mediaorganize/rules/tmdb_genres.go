package rules

import (
	"strconv"
	"strings"
)

// tmdbGenreNames TMDB 标准类型 id → 中文名（用于 genre_ids → 类型名映射）
var tmdbGenreNames = map[int]string{
	28: "动作", 12: "冒险", 16: "动画", 35: "喜剧", 80: "犯罪",
	99: "纪录", 18: "剧情", 10751: "家庭", 14: "奇幻", 36: "历史",
	27: "恐怖", 10402: "音乐", 9648: "悬疑", 10749: "爱情", 878: "科幻",
	10770: "电视电影", 53: "惊悚", 10752: "战争", 37: "西部",
	10759: "动作冒险", 10762: "儿童", 10763: "新闻", 10764: "真人秀",
	10765: "科幻奇幻", 10766: "肥皂剧", 10767: "谈话", 10768: "战争政治",
}

// TMDBGenreName 返回类型 id 对应的中文名；未知返回空。
func TMDBGenreName(id int) string {
	return tmdbGenreNames[id]
}

// ExtractTMDBGenres 从 TMDB 原始数据提取类型名称列表。
// 优先取详情字段 genres（[{id,name}]），回退到搜索字段 genre_ids（[28,12]）。
func ExtractTMDBGenres(raw map[string]any) []string {
	if len(raw) == 0 {
		return nil
	}
	if list, ok := raw["genres"].([]any); ok && len(list) > 0 {
		out := make([]string, 0, len(list))
		for _, item := range list {
			if m, ok := item.(map[string]any); ok {
				if name := strVal(m["name"]); name != "" {
					out = append(out, name)
				}
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	if ids, ok := raw["genre_ids"].([]any); ok && len(ids) > 0 {
		out := make([]string, 0, len(ids))
		for _, item := range ids {
			if n, err := strconv.Atoi(toString(item)); err == nil {
				if name := TMDBGenreName(n); name != "" {
					out = append(out, name)
				}
			}
		}
		return out
	}
	return nil
}

// ExtractTMDBGenreIDs 从 TMDB 原始数据提取类型 id 列表（[16, 10751]）。
// 优先取详情 genres（[{id,name}]），回退到搜索 genre_ids。
func ExtractTMDBGenreIDs(raw map[string]any) []int {
	if len(raw) == 0 {
		return nil
	}
	if list, ok := raw["genres"].([]any); ok && len(list) > 0 {
		out := make([]int, 0, len(list))
		for _, item := range list {
			if m, ok := item.(map[string]any); ok {
				if n, err := strconv.Atoi(toString(m["id"])); err == nil {
					out = append(out, n)
				}
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	if ids, ok := raw["genre_ids"].([]any); ok && len(ids) > 0 {
		out := make([]int, 0, len(ids))
		for _, item := range ids {
			if n, err := strconv.Atoi(toString(item)); err == nil {
				out = append(out, n)
			}
		}
		return out
	}
	return nil
}

// ExtractTMDBOriginalLanguage 返回 TMDB 原始语言代码（如 zh/en/ja）。
func ExtractTMDBOriginalLanguage(raw map[string]any) string {
	if len(raw) == 0 {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(strVal(raw["original_language"])))
}

// ExtractTMDBOriginCountries 返回 TMDB 原始地区代码列表（如 CN/US）。
func ExtractTMDBOriginCountries(raw map[string]any) []string {
	if len(raw) == 0 {
		return nil
	}
	if list, ok := raw["origin_country"].([]any); ok && len(list) > 0 {
		out := make([]string, 0, len(list))
		for _, item := range list {
			if c := strings.ToUpper(strings.TrimSpace(toString(item))); c != "" {
				out = append(out, c)
			}
		}
		return out
	}
	// 兼容 production_countries（详情字段）
	if list, ok := raw["production_countries"].([]any); ok && len(list) > 0 {
		out := make([]string, 0, len(list))
		for _, item := range list {
			if m, ok := item.(map[string]any); ok {
				if c := strings.ToUpper(strings.TrimSpace(strVal(m["iso_3166_1"]))); c != "" {
					out = append(out, c)
				}
			}
		}
		return out
	}
	return nil
}
