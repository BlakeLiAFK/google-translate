package i18n

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// TranslateFunc 翻译函数签名
type TranslateFunc func(text, target, source string) (string, error)

// 支持的格式
const (
	FormatJSON       = "json"
	FormatTypeScript = "ts"
	FormatMarkdown   = "md"
	FormatINI        = "ini"
)

// DetectFormat 根据文件扩展名或内容检测格式
func DetectFormat(filename, content string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".json":
		return FormatJSON
	case ".ts", ".js":
		return FormatTypeScript
	case ".md", ".markdown":
		return FormatMarkdown
	case ".ini", ".cfg", ".conf":
		return FormatINI
	}
	// 根据内容推测
	trimmed := strings.TrimSpace(content)
	if strings.HasPrefix(trimmed, "{") {
		return FormatJSON
	}
	if strings.Contains(trimmed, "export default") || strings.Contains(trimmed, "export const") {
		return FormatTypeScript
	}
	if strings.HasPrefix(trimmed, "#") || strings.Contains(trimmed, "```") {
		return FormatMarkdown
	}
	if strings.Contains(trimmed, "[") && strings.Contains(trimmed, "=") {
		return FormatINI
	}
	return FormatJSON
}

// SupportedFormats 返回支持的格式列表
func SupportedFormats() []string {
	return []string{FormatJSON, FormatTypeScript, FormatMarkdown, FormatINI}
}

// TranslateByFormat 按格式翻译内容
func TranslateByFormat(format, content string, targetLangs []string, sourceLang string, translateFn TranslateFunc) (map[string]string, error) {
	switch format {
	case FormatJSON:
		return TranslateJSON(content, targetLangs, sourceLang, translateFn)
	case FormatTypeScript:
		return TranslateTS(content, targetLangs, sourceLang, translateFn)
	case FormatMarkdown:
		return TranslateMarkdown(content, targetLangs, sourceLang, translateFn)
	case FormatINI:
		return TranslateINI(content, targetLangs, sourceLang, translateFn)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// TranslateJSON 翻译 i18n JSON 内容
// 输入 JSON 字符串，返回 map[语言代码]JSON字符串
func TranslateJSON(content string, targetLangs []string, sourceLang string, translateFn TranslateFunc) (map[string]string, error) {
	// 解析 JSON
	var data map[string]any
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// 提取所有需要翻译的文本
	texts, paths := flattenValues(data, "")

	result := make(map[string]string)

	for _, lang := range targetLangs {
		// 翻译所有文本
		translated := make(map[string]string)
		for i, text := range texts {
			t, err := translateFn(text, lang, sourceLang)
			if err != nil {
				return nil, fmt.Errorf("translate to %s failed at key %s: %w", lang, paths[i], err)
			}
			translated[paths[i]] = t
		}

		// 重建 JSON 结构
		output := rebuildJSON(data, translated)
		b, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return nil, err
		}
		result[lang] = string(b)
	}

	return result, nil
}

// flattenValues 递归提取 JSON 中所有字符串值
func flattenValues(data map[string]any, prefix string) (texts []string, paths []string) {
	// 按 key 排序保证顺序一致
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := data[k]
		path := k
		if prefix != "" {
			path = prefix + "." + k
		}
		switch val := v.(type) {
		case string:
			texts = append(texts, val)
			paths = append(paths, path)
		case map[string]any:
			t, p := flattenValues(val, path)
			texts = append(texts, t...)
			paths = append(paths, p...)
		}
	}
	return
}

// rebuildJSON 用翻译后的文本重建 JSON 结构
func rebuildJSON(original map[string]any, translated map[string]string) map[string]any {
	return rebuildMap(original, translated, "")
}

func rebuildMap(original map[string]any, translated map[string]string, prefix string) map[string]any {
	result := make(map[string]any)
	for k, v := range original {
		path := k
		if prefix != "" {
			path = prefix + "." + k
		}
		switch val := v.(type) {
		case string:
			if t, ok := translated[path]; ok {
				result[k] = t
			} else {
				result[k] = val
			}
		case map[string]any:
			result[k] = rebuildMap(val, translated, path)
		default:
			result[k] = v
		}
	}
	return result
}
