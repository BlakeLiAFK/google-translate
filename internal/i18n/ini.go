package i18n

import (
	"fmt"
	"strings"
)

// INI 格式支持
// 支持以下格式:
//   [section]
//   key = value
//   key=value
//   ; 注释
//   # 注释

// TranslateINI 翻译 INI 格式内容
func TranslateINI(content string, targetLangs []string, sourceLang string, translateFn TranslateFunc) (map[string]string, error) {
	// 解析 INI
	entries, err := parseINI(content)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)

	for _, lang := range targetLangs {
		// 翻译所有值
		translatedEntries := make([]iniEntry, len(entries))
		copy(translatedEntries, entries)

		for i, e := range translatedEntries {
			if e.entryType != "keyval" {
				continue
			}
			t, err := translateFn(e.value, lang, sourceLang)
			if err != nil {
				return nil, fmt.Errorf("translate to %s failed at [%s] %s: %w", lang, e.section, e.key, err)
			}
			translatedEntries[i].value = t
		}

		result[lang] = buildINI(translatedEntries)
	}

	return result, nil
}

// iniEntry 表示 INI 文件中的一行
type iniEntry struct {
	entryType string // "section", "keyval", "comment", "empty"
	raw       string // 原始行内容
	section   string // 当前所属 section
	key       string
	value     string
	separator string // " = " 或 "=" 保持原格式
}

// parseINI 解析 INI 内容
func parseINI(content string) ([]iniEntry, error) {
	lines := strings.Split(content, "\n")
	var entries []iniEntry
	currentSection := ""

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// 空行
		if trimmed == "" {
			entries = append(entries, iniEntry{entryType: "empty", raw: line})
			continue
		}

		// 注释
		if strings.HasPrefix(trimmed, ";") || strings.HasPrefix(trimmed, "#") {
			entries = append(entries, iniEntry{entryType: "comment", raw: line})
			continue
		}

		// Section
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			currentSection = trimmed[1 : len(trimmed)-1]
			entries = append(entries, iniEntry{entryType: "section", raw: line, section: currentSection})
			continue
		}

		// Key = Value
		eqIdx := strings.Index(line, "=")
		if eqIdx > 0 {
			key := strings.TrimSpace(line[:eqIdx])
			value := strings.TrimSpace(line[eqIdx+1:])

			// 保持原始分隔符格式
			sep := "="
			if eqIdx+1 < len(line) && line[eqIdx+1] == ' ' {
				sep = "= "
			}
			if eqIdx > 0 && line[eqIdx-1] == ' ' {
				sep = " " + sep
			}

			entries = append(entries, iniEntry{
				entryType: "keyval",
				section:   currentSection,
				key:       key,
				value:     value,
				separator: sep,
			})
			continue
		}

		// 未知行，保持原样
		entries = append(entries, iniEntry{entryType: "comment", raw: line})
	}

	return entries, nil
}

// buildINI 从 entries 重建 INI 内容
func buildINI(entries []iniEntry) string {
	var lines []string
	for _, e := range entries {
		switch e.entryType {
		case "empty":
			lines = append(lines, "")
		case "comment":
			lines = append(lines, e.raw)
		case "section":
			lines = append(lines, e.raw)
		case "keyval":
			lines = append(lines, e.key+e.separator+e.value)
		}
	}
	return strings.Join(lines, "\n")
}
