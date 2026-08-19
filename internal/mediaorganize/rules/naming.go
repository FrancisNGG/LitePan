package rules

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

func SanitizeFilename(name string) string {
	replacer := strings.NewReplacer(
		"/", "",
		"\\", "",
		":", "：",
		"*", "",
		"?", "",
		"\"", "",
		"<", "",
		">", "",
		"|", "",
	)
	return strings.TrimSpace(replacer.Replace(name))
}

func IsSameGeneratedName(currentName, generatedName string) bool {
	return strings.TrimSpace(currentName) == strings.TrimSpace(generatedName)
}

// BuildFolderName 用旧逻辑构建文件夹名：Title (Year) {tmdb-xxx}
func BuildFolderName(parsed ParsedMedia, tmdbID string) string {
	title := strings.TrimSpace(parsed.Title)
	if title == "" {
		return ""
	}
	parts := []string{title}
	if parsed.Year != nil {
		parts = append(parts, fmt.Sprintf("(%d)", *parsed.Year))
	}
	if tmdbID != "" {
		parts = append(parts, fmt.Sprintf("{tmdb-%s}", tmdbID))
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

// BuildFolderNameTpl 用 pongo2/Jinja2 模板构建文件夹名（支持 en_title/if/过滤器）
// tpl 为空时退回默认行为。返回空串表示模板渲染失败或结果为空。
func BuildFolderNameTpl(parsed ParsedMedia, enTitle, tmdbID, tpl string) string {
	if strings.TrimSpace(tpl) == "" {
		return BuildFolderName(parsed, tmdbID)
	}
	ctx := TemplateContext{}
	ctx.FromParsedMedia(parsed, enTitle, tmdbID)
	name, err := RenderTemplate(tpl, ctx)
	if err != nil {
		return ""
	}
	return name
}

// BuildTargetFilename 用旧逻辑构建目标文件名：Title (Year) [marker] SxxExx
func BuildTargetFilename(parsed ParsedMedia, marker, tmdbID string) string {
	title := strings.TrimSpace(parsed.Title)
	if title == "" {
		return ""
	}
	parts := []string{title}
	if parsed.Year != nil {
		parts = append(parts, fmt.Sprintf("(%d)", *parsed.Year))
	}
	tag := ""
	if !IsMarkerOff(marker) {
		switch strings.TrimSpace(marker) {
		case "tmdb", "tmdbid":
			if tmdbID != "" {
				tag = fmt.Sprintf("{tmdb-%s}", tmdbID)
			}
		default:
			if m := strings.TrimSpace(marker); m != "" {
				tag = fmt.Sprintf("[%s]", m)
			}
		}
	}
	if tag != "" {
		parts = append(parts, tag)
	}
	season := asFirstInt(parsed.Season)
	episode := asFirstInt(parsed.Episode)
	if season != nil && episode != nil {
		parts = append(parts, fmt.Sprintf("S%02dE%02d", *season, *episode))
	}
	return strings.Join(parts, " ")
}

// BuildTargetFilenameTpl 用 pongo2/Jinja2 模板构建目标文件名（支持 en_title/if/过滤器）
func BuildTargetFilenameTpl(parsed ParsedMedia, enTitle, marker, tmdbID, tpl string) string {
	if strings.TrimSpace(tpl) == "" {
		return BuildTargetFilename(parsed, marker, tmdbID)
	}
	ctx := TemplateContext{}
	ctx.FromParsedMedia(parsed, enTitle, tmdbID)
	name, err := RenderTemplate(tpl, ctx)
	if err != nil {
		return ""
	}
	return name
}

func BuildDisplayTitle(tmdbTitle, tmdbOriginal, fallbackTitle string) string {
	if tmdbTitle == "" {
		return fallbackTitle
	}
	if tmdbOriginal == "" || strings.EqualFold(tmdbOriginal, tmdbTitle) {
		return tmdbTitle
	}
	if isLatinScript(tmdbOriginal) {
		return tmdbTitle + " - " + tmdbOriginal
	}
	return tmdbTitle
}

func FitFilenameBytes(filename, tmdbLang string) string {
	if len([]byte(filename)) <= MaxFilenameBytes {
		return filename
	}
	m := dualTitleRe.FindStringSubmatch(filename)
	if m != nil {
		title1, title2, year, rest := m[1], m[2], m[3], m[4]
		short := title2 + " (" + year + ")" + rest
		if strings.HasPrefix(strings.ToLower(tmdbLang), "zh") {
			short = title1 + " (" + year + ")" + rest
		}
		if len([]byte(short)) <= MaxFilenameBytes {
			return short
		}
		filename = short
	}
	matches := bracketTagRe.FindAllStringSubmatchIndex(filename, -1)
	if len(matches) == 0 {
		return filename
	}
	last := matches[len(matches)-1]
	tagContent := filename[last[2]:last[3]]
	tags := strings.Fields(tagContent)
	for i := len(tags); i > 0; i-- {
		newTag := "[" + strings.Join(tags[:i], " ") + "]"
		short := filename[:last[0]] + newTag + filename[last[1]:]
		if len([]byte(short)) <= MaxFilenameBytes {
			return short
		}
	}
	return filename[:last[0]] + filename[last[1]:]
}

func IsAlreadyOrganized(filename, marker string) bool {
	if IsMarkerOff(marker) {
		return looksLikeOrganizedStructure(filename) || FindTMDBIDInName(filename) != ""
	}
	m := strings.TrimSpace(marker)
	switch m {
	case "tmdb", "tmdbid":
		return FindTMDBIDInName(filename) != ""
	default:
		return strings.Contains(filename, fmt.Sprintf("[%s]", m))
	}
}

func looksLikeOrganizedStructure(filename string) bool {
	return organizedStructureRe.MatchString(strings.TrimSpace(filename))
}

func IsMarkerOff(marker string) bool {
	switch strings.ToLower(strings.TrimSpace(marker)) {
	case "", "0", "off", "none", "no", "false":
		return true
	default:
		return false
	}
}

func isLatinScript(text string) bool {
	if text == "" {
		return false
	}
	hasLetter := false
	for _, ch := range text {
		if unicode.IsSpace(ch) || strings.ContainsRune(".,:;'\"-_!?()&+/", ch) {
			continue
		}
		if unicode.IsDigit(ch) {
			continue
		}
		if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') ||
			(ch >= 0x00C0 && ch <= 0x024F) || (ch >= 0x1E00 && ch <= 0x1EFF) {
			hasLetter = true
			continue
		}
		return false
	}
	return hasLetter
}

var (
	dualTitleRe = regexp.MustCompile(`^(.+?) - (.+?) \((\d{4})\)(.*)$`)
	bracketTagRe = regexp.MustCompile(`\[([^\]]+)\]`)
)
