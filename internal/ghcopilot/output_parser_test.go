package ghcopilot

import (
	"strings"
	"testing"
)

// TestNewOutputParser 測試建立新的輸出解析器
func TestNewOutputParser(t *testing.T) {
	parser := NewOutputParser("test output")
	if parser == nil {
		t.Error("NewOutputParser() 傳回 nil")
	}
	if parser.rawOutput != "test output" {
		t.Errorf("rawOutput 應為 'test output'，但為 '%s'", parser.rawOutput)
	}
}

// TestExtractCodeBlocks 測試提取程式碼區塊
func TestExtractCodeBlocks(t *testing.T) {
	output := "示例輸出\n\n```bash\necho \"Hello\"\nls -la\n```\n\n更多文本\n\n```go\npackage main\nfunc main() {}\n```"

	parser := NewOutputParser(output)
	blocks := parser.ExtractCodeBlocks()

	if len(blocks) != 2 {
		t.Errorf("應提取 2 個程式碼區塊，但提取了 %d 個", len(blocks))
	}

	if len(blocks) > 0 && blocks[0].Language != "bash" {
		t.Errorf("第一個區塊語言應為 'bash'，但為 '%s'", blocks[0].Language)
	}

	if len(blocks) > 1 && blocks[1].Language != "go" {
		t.Errorf("第二個區塊語言應為 'go'，但為 '%s'", blocks[1].Language)
	}
}

// TestRemoveMarkdown 測試移除 Markdown 格式
func TestRemoveMarkdown(t *testing.T) {
	output := "# 標題\n這是**粗體**和*斜體*文本\n[連結](http://example.com)"

	parser := NewOutputParser(output)
	cleaned := parser.RemoveMarkdown()

	if strings.Contains(cleaned, "**") {
		t.Error("應移除粗體標記 **")
	}

	if strings.Contains(cleaned, "http://") {
		t.Error("應移除超連結格式")
	}
}

// TestExtractOptions 測試提取選項
func TestExtractOptions(t *testing.T) {
	output := "建議 1: 第一個選項\n建議 2: 第二個選項\n3. 第三個選項\n- 項目一\n* 項目二"

	parser := NewOutputParser(output)
	parser.Parse()
	options := parser.GetOptions()

	if len(options) == 0 {
		t.Error("應提取至少一個選項")
	}
}

// TestOutputParserWithCodeBlocks 測試帶程式碼區塊的解析
func TestOutputParserWithCodeBlocks(t *testing.T) {
	output := "為了完成這個任務，請運行以下指令:\n\n```bash\ngit add .\ngit commit -m \"Update code\"\ngit push origin main\n```\n\n這將推送您的變更。"

	parser := NewOutputParser(output)
	blocks := parser.ExtractCodeBlocks()

	if len(blocks) != 1 {
		t.Errorf("應有 1 個程式碼區塊，但有 %d 個", len(blocks))
		return
	}

	if !strings.Contains(blocks[0].Content, "git add") {
		t.Error("程式碼區塊應包含 'git add'")
	}

	if blocks[0].Language != "bash" {
		t.Errorf("語言應為 'bash'，但為 '%s'", blocks[0].Language)
	}
}

// TestNumberedItemDetection 測試編號項目偵測
func TestNumberedItemDetection(t *testing.T) {
	tests := []struct {
		line     string
		expected bool
	}{
		{"1. First item", true},
		{"2. Second item", true},
		{"10. Tenth item", true},
		{"Not numbered", false},
		{"-item", false},
		{"", false},
	}

	for _, tt := range tests {
		result := isNumberedItem(tt.line)
		if result != tt.expected {
			t.Errorf("isNumberedItem('%s') = %v，期望 %v", tt.line, result, tt.expected)
		}
	}
}

// TestBulletItemDetection 測試項目符號偵測
func TestBulletItemDetection(t *testing.T) {
	tests := []struct {
		line     string
		expected bool
	}{
		{"- item", true},
		{"* item", true},
		{"- - nested", true},
		{"No bullet", false},
		{"--double dash", false},
	}

	for _, tt := range tests {
		result := isBulletItem(tt.line)
		if result != tt.expected {
			t.Errorf("isBulletItem('%s') = %v，期望 %v", tt.line, result, tt.expected)
		}
	}
}
