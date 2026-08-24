package rules

import "strings"

// preprocessProtectedTokens 保护含点号的复合标记：PreprocessDottedFilename 会把点号
// 替换成空格，但 H.265 / DTS-HD.MA / DDP5.1 这类标记的点号是语义的一部分，拆成空格
// 会导致下游 go-fish 识别失败（H.265 → H 265 → video_codec 丢失）。
// 与 go-fish tokenizer 的 groupedPatterns 占位符机制对齐。
var preprocessProtectedTokens = []struct {
	pattern     string // 小写匹配
	placeholder string
	restore     string
}{
	{"h.264", "\x00LPPH264\x00", "H.264"},
	{"h.265", "\x00LPPH265\x00", "H.265"},
	{"dts-hd.ma", "\x00LPPHDTSHDMA\x00", "DTS-HD.MA"},
	{"ddp5.1", "\x00LPPHDDP51\x00", "DDP5.1"},
	{"ddp7.1", "\x00LPPHDDP71\x00", "DDP7.1"},
	{"ddp2.0", "\x00LPPHDDP20\x00", "DDP2.0"},
	{"dd5.1", "\x00LPPHDD51\x00", "DD5.1"},
}

func PreprocessDottedFilename(name string) string {
	if strings.Count(name, ".") < 2 {
		return name
	}
	// 1. 保护含点复合标记（大小写不敏感），避免点号被替换成空格后拆坏
	protected := name
	lower := strings.ToLower(protected)
	for _, t := range preprocessProtectedTokens {
		for {
			idx := strings.Index(lower, t.pattern)
			if idx < 0 {
				break
			}
			protected = protected[:idx] + t.placeholder + protected[idx+len(t.pattern):]
			lower = strings.ToLower(protected)
		}
	}

	// 2. 原有逻辑（作用于 protected）
	m := dottedYearRe.FindStringSubmatchIndex(protected)
	var base, ext string
	if strings.Contains(protected, ".") {
		base, ext = splitStemExt(protected)
	} else {
		base, ext = protected, ""
	}
	var out string
	if m == nil {
		if len(ext) <= 4 {
			out = strings.ReplaceAll(base, ".", " ") + "." + ext
		} else {
			out = strings.ReplaceAll(protected, ".", " ")
		}
	} else {
		yearPos := m[2]
		prefix := strings.ReplaceAll(protected[:yearPos], ".", " ")
		suffix := protected[yearPos:]
		out = prefix + suffix
	}

	// 3. 还原占位符
	for _, t := range preprocessProtectedTokens {
		out = strings.ReplaceAll(out, t.placeholder, t.restore)
	}
	return out
}

func StripKnownIDTags(name string) string {
	if name == "" {
		return name
	}
	return strings.TrimSpace(knownIDTagRe.ReplaceAllString(name, " "))
}

// StripChineseBracketTags 剥掉含汉字的方括号/中文括号标签（如 [全10集] [简繁英字幕] 【广告】），
// 保留纯数字/字母的方括号（如 [2160p] [01]）。
func StripChineseBracketTags(name string) string {
	if name == "" {
		return name
	}
	out := cnBracketTagRe.ReplaceAllString(name, " ")
	return strings.TrimSpace(out)
}

func StripReleaseSitePrefix(name string) string {
	raw := strings.TrimSpace(name)
	if raw == "" {
		return raw
	}
	prev := ""
	for prev != raw {
		prev = raw
		raw = strings.TrimSpace(releaseSiteLeadingBracketRe.ReplaceAllString(raw, ""))
	}
	return raw
}

func StripChineseQualityTags(title string) string {
	raw := strings.TrimSpace(title)
	if raw == "" {
		return raw
	}
	cleaned := cnQualityTagRe.ReplaceAllString(raw, " ")
	cleaned = cnShortTagRe.ReplaceAllString(cleaned, "$1")
	for previous := ""; previous != cleaned; {
		previous = cleaned
		cleaned = enQualityTagRe.ReplaceAllString(cleaned, "${1}${2}")
	}
	cleaned = noiseSpaceRe.ReplaceAllString(cleaned, " ")
	cleaned = trimChars(cleaned, " ._-+")
	for previous := ""; previous != cleaned; {
		previous = cleaned
		cleaned = strings.TrimSpace(enEditionSuffixRe.ReplaceAllString(cleaned, ""))
	}
	if cleaned == "" {
		return raw
	}
	return cleaned
}

func StripReleaseGroupFromTitle(title string) string {
	raw := strings.TrimSpace(title)
	if raw == "" {
		return raw
	}
	if m := releaseGroupTailRe.FindStringSubmatch(raw); len(m) >= 3 {
		if isKnownReleaseGroup(m[2]) {
			out := trimChars(m[1], " ._-")
			if out != "" {
				return out
			}
		}
	}
	if m := releaseGroupDashRe.FindStringSubmatch(raw); len(m) >= 3 {
		if isKnownReleaseGroup(m[2]) {
			out := trimChars(m[1], " ._-")
			if out != "" {
				return out
			}
		}
	}
	return raw
}

func StripReleaseGroupFromStem(stem string, parsed ParsedMedia) string {
	raw := strings.TrimSpace(stem)
	if raw == "" {
		return raw
	}
	if parsed.ReleaseGroup != "" {
		for _, sep := range []string{"-", ".", "_"} {
			suffix := sep + parsed.ReleaseGroup
			if strings.HasSuffix(raw, suffix) && len(raw) > len(suffix) {
				return trimChars(raw[:len(raw)-len(suffix)], "._- ")
			}
		}
	}
	if m := releaseGroupTailRe.FindStringSubmatch(raw); len(m) >= 3 {
		if isKnownReleaseGroup(m[2]) {
			out := trimChars(m[1], "._- ")
			if out != "" {
				return out
			}
		}
	}
	if m := releaseGroupDashRe.FindStringSubmatch(raw); len(m) >= 3 {
		if isKnownReleaseGroup(m[2]) {
			out := trimChars(m[1], "._- ")
			if out != "" {
				return out
			}
		}
	}
	if m := releaseGroupGenericRe.FindStringSubmatch(raw); len(m) >= 3 {
		tail := m[2]
		if !seTokenRe.MatchString(tail) && !metaTailBlocklist[strings.ToLower(tail)] {
			head := trimChars(m[1], "._- ")
			if head != "" && len(head) >= max(8, len(raw)/3) {
				return head
			}
		}
	}
	return raw
}

func StripSeasonSuffix(name string) (string, *int) {
	raw := strings.TrimSpace(name)
	if raw == "" {
		return raw, nil
	}
	if m := cnSeasonSuffixRe.FindStringSubmatch(raw); len(m) >= 3 {
		if season := ChineseNumberToInt(m[2]); season != nil {
			return trimChars(m[1], " ._-"), season
		}
	}
	if m := enSeasonSuffixRe.FindStringSubmatch(raw); len(m) >= 3 {
		if n, err := parseInt(m[2]); err == nil {
			return trimChars(m[1], " ._-"), intPtr(n)
		}
	}
	return raw, nil
}

func StripTrailingNumber(name string) (string, *int) {
	raw := strings.TrimSpace(name)
	if raw == "" {
		return raw, nil
	}
	m := trailingNumberRe.FindStringSubmatch(raw)
	if len(m) < 3 {
		return raw, nil
	}
	n, err := parseInt(m[2])
	if err != nil {
		return raw, nil
	}
	titlePart := trimChars(m[1], " ._-")
	if titlePart == "" {
		return raw, nil
	}
	return titlePart, intPtr(n)
}

var metaTailBlocklist = map[string]bool{
	"1": true, "2": true, "5": true, "7": true,
	"atmos": true, "ddp": true, "dts": true, "hevc": true,
	"sdr": true, "hdr": true, "dv": true, "nf": true, "web": true,
}

func parseInt(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errInvalidInt
	}
	n := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return 0, errInvalidInt
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

type parseIntError struct{}

func (e *parseIntError) Error() string { return "invalid int" }

var errInvalidInt = &parseIntError{}
