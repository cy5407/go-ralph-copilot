package ghcopilot

import (
	"regexp"
	"strings"
)

// OutputParser 用於解析 Copilot CLI 的輸出
type OutputParser struct {
	rawOutput string
	options   []string
}

// NewOutputParser 建立新的輸出解析器
func NewOutputParser(rawOutput string) *OutputParser {
	return &OutputParser{
		rawOutput: rawOutput,
		options:   []string{},
	}
}

// Parse 解析輸出
func (op *OutputParser) Parse() error {
	op.extractOptions()
	return nil
}

// GetOptions 取得所有選項
func (op *OutputParser) GetOptions() []string {
	return op.options
}

// extractOptions 提取選項
func (op *OutputParser) extractOptions() {
	lines := strings.Split(op.rawOutput, "\n")
	var currentOption strings.Builder
	var inCodeBlock bool

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// 檢查程式碼區塊開始/結束
		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			if inCodeBlock && currentOption.Len() > 0 {
				op.options = append(op.options, strings.TrimSpace(currentOption.String()))
				currentOption.Reset()
			}
		} else if inCodeBlock {
			if currentOption.Len() > 0 {
				currentOption.WriteString("\n")
			}
			currentOption.WriteString(line)
		} else if isNumberedItem(trimmed) || isBulletItem(trimmed) {
			if currentOption.Len() > 0 {
				op.options = append(op.options, strings.TrimSpace(currentOption.String()))
				currentOption.Reset()
			}
			currentOption.WriteString(trimmed)
		} else if currentOption.Len() > 0 && trimmed != "" {
			currentOption.WriteString(" ")
			currentOption.WriteString(trimmed)
		}
	}

	if currentOption.Len() > 0 {
		op.options = append(op.options, strings.TrimSpace(currentOption.String()))
	}
}

// ExtractCodeBlocks 提取所有程式碼區塊
func (op *OutputParser) ExtractCodeBlocks() []CodeBlock {
	var blocks []CodeBlock
	lines := strings.Split(op.rawOutput, "\n")
	var inBlock bool
	var language string
	var content strings.Builder

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") {
			if !inBlock {
				inBlock = true
				language = strings.TrimPrefix(trimmed, "```")
				language = strings.TrimSpace(language)
				content.Reset()
			} else {
				blocks = append(blocks, CodeBlock{
					Language: language,
					Content:  strings.TrimSpace(content.String()),
				})
				inBlock = false
			}
		} else if inBlock {
			if content.Len() > 0 {
				content.WriteString("\n")
			}
			content.WriteString(line)
		}
	}

	return blocks
}

// RemoveMarkdown 移除 Markdown 格式標記
func (op *OutputParser) RemoveMarkdown() string {
	text := op.rawOutput

	// 移除程式碼區塊標記
	text = regexp.MustCompile("```[^`]*```").ReplaceAllString(text, "")

	// 移除粗體
	text = regexp.MustCompile(`\*\*(.*?)\*\*`).ReplaceAllString(text, "$1")

	// 移除斜體
	text = regexp.MustCompile(`\*(.*?)\*`).ReplaceAllString(text, "$1")

	// 移除標題標記
	text = regexp.MustCompile(`^#+\s+`).ReplaceAllString(text, "")

	// 移除超連結標記
	text = regexp.MustCompile(`\[(.*?)\]\((.*?)\)`).ReplaceAllString(text, "$1")

	return text
}

// CodeBlock 代表一個程式碼區塊
type CodeBlock struct {
	Language string
	Content  string
}

// isNumberedItem 檢查是否為編號項目
func isNumberedItem(line string) bool {
	// 匹配 "1.", "2." 等格式
	matched, _ := regexp.MatchString(`^\d+\.\s+`, line)
	return matched
}

// isBulletItem 檢查是否為項目符號
func isBulletItem(line string) bool {
	// 匹配 "-" 或 "*" 開頭
	return strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ")
}
