package i18n

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// TypeScript i18n 格式支持
// 支持以下格式:
//   export default { key: "value" }
//   export const messages = { key: "value" }
//   const messages = { key: "value" }

// tsObjectPattern 匹配 TS 中的对象字面量
var tsObjectPattern = regexp.MustCompile(`(?s)((?:export\s+default|export\s+const\s+\w+\s*=|const\s+\w+\s*=)\s*)(\{.+\})(\s*;?\s*$)`)

// TranslateTS 翻译 TypeScript i18n 内容
func TranslateTS(content string, targetLangs []string, sourceLang string, translateFn TranslateFunc) (map[string]string, error) {
	// 提取 TS 中的对象部分
	matches := tsObjectPattern.FindStringSubmatch(strings.TrimSpace(content))
	if matches == nil {
		return nil, fmt.Errorf("cannot parse TypeScript content: expected 'export default {...}' or 'export const xxx = {...}'")
	}

	prefix := matches[1]  // "export default " 或 "export const xxx = "
	objStr := matches[2]  // { ... }
	suffix := matches[3]  // 结尾分号等

	// TS 对象字面量转 JSON (处理无引号 key 和尾逗号)
	jsonStr := tsObjectToJSON(objStr)

	// 用 JSON 方式翻译
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, fmt.Errorf("parse TS object failed: %w (converted JSON: %s)", err, jsonStr)
	}

	texts, paths := flattenValues(data, "")
	result := make(map[string]string)

	for _, lang := range targetLangs {
		translated := make(map[string]string)
		for i, text := range texts {
			t, err := translateFn(text, lang, sourceLang)
			if err != nil {
				return nil, fmt.Errorf("translate to %s failed at key %s: %w", lang, paths[i], err)
			}
			translated[paths[i]] = t
		}

		output := rebuildJSON(data, translated)
		b, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return nil, err
		}

		// 转回 TS 格式
		tsOutput := prefix + string(b) + suffix
		result[lang] = tsOutput
	}

	return result, nil
}

// tsObjectToJSON 将 TS 对象字面量转换为合法 JSON
func tsObjectToJSON(s string) string {
	// 移除单行注释
	lines := strings.Split(s, "\n")
	var cleaned []string
	for _, line := range lines {
		// 移除 // 注释 (但不影响 URL 中的 //)
		if idx := strings.Index(line, "//"); idx >= 0 {
			// 简单判断：如果 // 前面不是 : 或 http 就当注释
			before := strings.TrimSpace(line[:idx])
			if !strings.HasSuffix(before, ":") && !strings.Contains(before, "http") {
				line = line[:idx]
			}
		}
		cleaned = append(cleaned, line)
	}
	s = strings.Join(cleaned, "\n")

	// 给无引号的 key 加引号: word: -> "word":
	reKey := regexp.MustCompile(`(?m)^(\s*)(\w+)\s*:`)
	s = reKey.ReplaceAllString(s, `$1"$2":`)

	// 单引号转双引号
	s = strings.ReplaceAll(s, "'", "\"")

	// 移除尾逗号
	reTrailing := regexp.MustCompile(`,(\s*[}\]])`)
	s = reTrailing.ReplaceAllString(s, "$1")

	return s
}
