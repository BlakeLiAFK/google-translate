package i18n

import (
	"fmt"
	"strings"
)

// TranslateMarkdown 翻译 Markdown 文档
// 保留 Markdown 语法结构，只翻译文本内容
func TranslateMarkdown(content string, targetLangs []string, sourceLang string, translateFn TranslateFunc) (map[string]string, error) {
	blocks := splitMarkdownBlocks(content)

	result := make(map[string]string)

	for _, lang := range targetLangs {
		var translated []string
		for _, block := range blocks {
			t, err := translateBlock(block, lang, sourceLang, translateFn)
			if err != nil {
				return nil, fmt.Errorf("translate to %s failed: %w", lang, err)
			}
			translated = append(translated, t)
		}
		result[lang] = strings.Join(translated, "\n")
	}

	return result, nil
}

// markdownBlock 表示一个 Markdown 内容块
type markdownBlock struct {
	content    string
	blockType  string // "text", "code", "empty", "html"
}

// splitMarkdownBlocks 将 Markdown 按块分割
func splitMarkdownBlocks(content string) []markdownBlock {
	lines := strings.Split(content, "\n")
	var blocks []markdownBlock
	inCodeBlock := false
	var codeLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// 代码块开始/结束
		if strings.HasPrefix(trimmed, "```") {
			if inCodeBlock {
				// 代码块结束
				codeLines = append(codeLines, line)
				blocks = append(blocks, markdownBlock{
					content:   strings.Join(codeLines, "\n"),
					blockType: "code",
				})
				codeLines = nil
				inCodeBlock = false
			} else {
				// 代码块开始
				inCodeBlock = true
				codeLines = []string{line}
			}
			continue
		}

		if inCodeBlock {
			codeLines = append(codeLines, line)
			continue
		}

		// 空行
		if trimmed == "" {
			blocks = append(blocks, markdownBlock{content: "", blockType: "empty"})
			continue
		}

		// HTML 标签行
		if strings.HasPrefix(trimmed, "<") && strings.HasSuffix(trimmed, ">") {
			blocks = append(blocks, markdownBlock{content: line, blockType: "html"})
			continue
		}

		// 普通文本行（包含标题、列表等）
		blocks = append(blocks, markdownBlock{content: line, blockType: "text"})
	}

	// 未关闭的代码块
	if inCodeBlock && len(codeLines) > 0 {
		blocks = append(blocks, markdownBlock{
			content:   strings.Join(codeLines, "\n"),
			blockType: "code",
		})
	}

	return blocks
}

// translateBlock 翻译单个 Markdown 块
func translateBlock(block markdownBlock, targetLang, sourceLang string, translateFn TranslateFunc) (string, error) {
	switch block.blockType {
	case "code", "empty", "html":
		// 不翻译
		return block.content, nil
	case "text":
		return translateMarkdownLine(block.content, targetLang, sourceLang, translateFn)
	}
	return block.content, nil
}

// translateMarkdownLine 翻译一行 Markdown 文本，保留语法前缀
func translateMarkdownLine(line, targetLang, sourceLang string, translateFn TranslateFunc) (string, error) {
	// 提取 Markdown 前缀（标题、列表等）
	prefix, text := extractMarkdownPrefix(line)

	if strings.TrimSpace(text) == "" {
		return line, nil
	}

	// 纯分隔线不翻译
	trimmed := strings.TrimSpace(text)
	if isMarkdownSeparator(trimmed) {
		return line, nil
	}

	// 翻译文本部分
	translated, err := translateFn(text, targetLang, sourceLang)
	if err != nil {
		return "", err
	}

	return prefix + translated, nil
}

// extractMarkdownPrefix 提取行首的 Markdown 语法前缀
func extractMarkdownPrefix(line string) (prefix, text string) {
	// 标题: # ## ### 等
	for i := 1; i <= 6; i++ {
		p := strings.Repeat("#", i) + " "
		if strings.HasPrefix(line, p) {
			return p, line[len(p):]
		}
	}

	// 无序列表: - * +
	for _, ch := range []string{"- ", "* ", "+ "} {
		trimmed := strings.TrimLeft(line, " \t")
		indent := line[:len(line)-len(trimmed)]
		if strings.HasPrefix(trimmed, ch) {
			return indent + ch, trimmed[len(ch):]
		}
	}

	// 有序列表: 1. 2. 等
	trimmed := strings.TrimLeft(line, " \t")
	indent := line[:len(line)-len(trimmed)]
	for j := 0; j < len(trimmed); j++ {
		if trimmed[j] >= '0' && trimmed[j] <= '9' {
			continue
		}
		if trimmed[j] == '.' && j > 0 && j+1 < len(trimmed) && trimmed[j+1] == ' ' {
			p := indent + trimmed[:j+2]
			return p, trimmed[j+2:]
		}
		break
	}

	// 引用: >
	if strings.HasPrefix(trimmed, "> ") {
		return indent + "> ", trimmed[2:]
	}

	return "", line
}

// isMarkdownSeparator 判断是否是分隔线
func isMarkdownSeparator(s string) bool {
	s = strings.ReplaceAll(s, " ", "")
	if len(s) < 3 {
		return false
	}
	allSame := true
	ch := s[0]
	if ch != '-' && ch != '*' && ch != '_' {
		return false
	}
	for _, c := range s {
		if byte(c) != ch {
			allSame = false
			break
		}
	}
	return allSame
}
