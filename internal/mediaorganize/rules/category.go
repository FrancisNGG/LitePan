package rules

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// CategoryRule 单条分类规则（与 MoviePilot category.yaml 完全一致）。
// 条件字段值均为逗号分隔字符串，支持 ! 排除与 YYYY-YYYY 范围；空条件 = 兜底分类。
type CategoryRule struct {
	Name               string // 分类目录名
	GenreIDs           string // 内容类型，如 "16" / "10764,10767"
	OriginalLanguage   string // 语种，如 "zh,cn,bo,za"
	OriginCountry      string // 国家或地区（电视剧），如 "CN,TW,HK"
	ProductionCountries string // 国家或地区（电影），如 "CN,US"
	ReleaseYear        string // 发行年份，如 "2010-2020"
}

// IsCatchAll 无条件规则（兜底分类）。
func (r CategoryRule) IsCatchAll() bool {
	return strings.TrimSpace(r.GenreIDs) == "" &&
		strings.TrimSpace(r.OriginalLanguage) == "" &&
		strings.TrimSpace(r.OriginCountry) == "" &&
		strings.TrimSpace(r.ProductionCountries) == "" &&
		strings.TrimSpace(r.ReleaseYear) == ""
}

// CategoryRules 电影/电视剧两组分类规则（保持配置顺序，按顺序匹配首个命中）。
type CategoryRules struct {
	Movie []CategoryRule
	TV    []CategoryRule
}

// ParseCategoryRules 解析分类策略 JSON（与 MoviePilot 一致，对象键保序）。
// 支持两种形态：
//
//	对象（保序）：{"movie": {"动画电影": {"genre_ids": "16"}, "外语电影": {}}, "tv": {...}}
//	数组（有序）：{"movie": [{"name": "动画电影", "genre_ids": "16"}, {"name": "外语电影"}]}
func ParseCategoryRules(raw string) CategoryRules {
	out := CategoryRules{}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return out
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &root); err != nil {
		return out
	}
	if m, ok := root["movie"]; ok {
		out.Movie = parseCategoryGroup(m)
	}
	if t, ok := root["tv"]; ok {
		out.TV = parseCategoryGroup(t)
	}
	return out
}

func parseCategoryGroup(raw json.RawMessage) []CategoryRule {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil
	}
	if strings.HasPrefix(trimmed, "[") {
		var items []map[string]json.RawMessage
		if err := json.Unmarshal(raw, &items); err != nil {
			return nil
		}
		out := make([]CategoryRule, 0, len(items))
		for _, item := range items {
			rule := CategoryRule{}
			if nameRaw, ok := item["name"]; ok {
				_ = json.Unmarshal(nameRaw, &rule.Name)
			}
			applyRuleFields(&rule, item)
			out = append(out, rule)
		}
		return out
	}
	dec := json.NewDecoder(bytes.NewBuffer(raw))
	dec.UseNumber()
	tok, err := dec.Token()
	if err != nil {
		return nil
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return nil
	}
	out := make([]CategoryRule, 0, 8)
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			break
		}
		name, _ := keyTok.(string)
		var cond json.RawMessage
		if err := dec.Decode(&cond); err != nil {
			break
		}
		rule := CategoryRule{Name: strings.TrimSpace(name)}
		var cm map[string]json.RawMessage
		if json.Unmarshal(cond, &cm) == nil {
			applyRuleFields(&rule, cm)
		}
		out = append(out, rule)
	}
	return out
}

func applyRuleFields(rule *CategoryRule, m map[string]json.RawMessage) {
	get := func(key string) string {
		if raw, ok := m[key]; ok {
			var s string
			if json.Unmarshal(raw, &s) == nil {
				return strings.TrimSpace(s)
			}
			var arr []string
			if json.Unmarshal(raw, &arr) == nil {
				return strings.Join(arr, ",")
			}
		}
		return ""
	}
	rule.GenreIDs = get("genre_ids")
	rule.OriginalLanguage = get("original_language")
	rule.OriginCountry = get("origin_country")
	rule.ProductionCountries = get("production_countries")
	rule.ReleaseYear = get("release_year")
}

// MatchCategory 根据 TMDB 原始数据匹配分类（逻辑与 MoviePilot get_category 一致）。
// 未配置规则或未命中时返回空（不分类）。
func (c CategoryRules) MatchCategory(isTV bool, tmdbInfo map[string]any) string {
	catRules := c.Movie
	if isTV {
		catRules = c.TV
	}
	if len(catRules) == 0 || len(tmdbInfo) == 0 {
		return ""
	}
	for _, rule := range catRules {
		if rule.IsCatchAll() {
			return rule.Name
		}
		matchFlag := true
		for _, cond := range []struct {
			attr  string
			value string
		}{
			{"genre_ids", rule.GenreIDs},
			{"original_language", rule.OriginalLanguage},
			{"origin_country", rule.OriginCountry},
			{"production_countries", rule.ProductionCountries},
			{"release_year", rule.ReleaseYear},
		} {
			if strings.TrimSpace(cond.value) == "" {
				continue
			}
			var infoValue any
			if cond.attr == "release_year" {
				infoValue = tmdbInfo["release_date"]
				if infoValue == nil {
					infoValue = tmdbInfo["first_air_date"]
				}
				if infoValue != nil {
					s := fmt.Sprint(infoValue)
					if len(s) >= 4 {
						infoValue = s[:4]
					}
				}
			} else {
				infoValue = tmdbInfo[cond.attr]
			}
			if infoValue == nil || fmt.Sprint(infoValue) == "" {
				matchFlag = false
				continue
			}
			infoValues := categoryInfoValues(cond.attr, infoValue)
			include, exclude := splitCategoryValues(cond.value)
			if len(include) > 0 && !stringSetsIntersect(include, infoValues) {
				matchFlag = false
			}
			if len(exclude) > 0 && stringSetsIntersect(exclude, infoValues) {
				matchFlag = false
			}
			if !matchFlag {
				break
			}
		}
		if matchFlag {
			return rule.Name
		}
	}
	return ""
}

// categoryInfoValues 提取 TMDB 字段的可比对值列表（统一转大写，与 MoviePilot 一致）。
func categoryInfoValues(attr string, infoValue any) []string {
	switch attr {
	case "production_countries":
		var out []string
		if list, ok := infoValue.([]any); ok {
			for _, item := range list {
				if m, ok := item.(map[string]any); ok {
					if c := strings.ToUpper(strings.TrimSpace(fmt.Sprint(m["iso_3166_1"]))); c != "" {
						out = append(out, c)
					}
				}
			}
		}
		return out
	case "genre_ids":
		var out []string
		if list, ok := infoValue.([]any); ok {
			for _, item := range list {
				if m, ok := item.(map[string]any); ok {
					if id := strings.TrimSpace(fmt.Sprint(m["id"])); id != "" {
						out = append(out, strings.ToUpper(id))
					}
				} else {
					out = append(out, strings.ToUpper(strings.TrimSpace(fmt.Sprint(item))))
				}
			}
		}
		return out
	default:
		if list, ok := infoValue.([]any); ok {
			out := make([]string, 0, len(list))
			for _, item := range list {
				out = append(out, strings.ToUpper(strings.TrimSpace(fmt.Sprint(item))))
			}
			return out
		}
		return []string{strings.ToUpper(strings.TrimSpace(fmt.Sprint(infoValue)))}
	}
}

// splitCategoryValues 解析逗号分隔值，支持 ! 排除与 YYYY-YYYY 范围（与 MoviePilot 一致）。
func splitCategoryValues(value string) (include, exclude []string) {
	var values []string
	for _, v := range strings.Split(value, ",") {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		values = append(values, v)
	}
	var expanded []string
	for _, v := range values {
		if !strings.Contains(v, "-") {
			expanded = append(expanded, v)
			continue
		}
		parts := strings.SplitN(v, "-", 2)
		begin, end := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		prefix := ""
		if strings.HasPrefix(begin, "!") {
			prefix = "!"
			begin = strings.TrimPrefix(begin, "!")
		}
		b, berr := strconv.Atoi(begin)
		e, eerr := strconv.Atoi(end)
		if berr == nil && eerr == nil {
			for i := b; i <= e; i++ {
				expanded = append(expanded, fmt.Sprintf("%s%d", prefix, i))
			}
		} else {
			expanded = append(expanded, prefix+begin, prefix+end)
		}
	}
	for _, v := range expanded {
		v = strings.ToUpper(v)
		if strings.HasPrefix(v, "!") {
			exclude = append(exclude, strings.TrimPrefix(v, "!"))
		} else {
			include = append(include, v)
		}
	}
	return include, exclude
}

func stringSetsIntersect(a, b []string) bool {
	set := map[string]struct{}{}
	for _, v := range a {
		set[v] = struct{}{}
	}
	for _, v := range b {
		if _, ok := set[v]; ok {
			return true
		}
	}
	return false
}
