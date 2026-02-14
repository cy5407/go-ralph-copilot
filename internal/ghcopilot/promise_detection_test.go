package ghcopilot

import (
	"strings"
	"testing"
)

// TestDetectPromise 測試基本的 Promise 偵測函數
func TestDetectPromise(t *testing.T) {
	tests := []struct {
		name          string
		text          string
		promisePhrase string
		expected      bool
	}{
		{
			name:          "偵測到完成承諾",
			text:          "所有任務已完成。\n<promise>任務完成！🥇</promise>",
			promisePhrase: "任務完成！🥇",
			expected:      true,
		},
		{
			name:          "沒有完成承諾",
			text:          "正在處理中...",
			promisePhrase: "任務完成！🥇",
			expected:      false,
		},
		{
			name:          "只有部分匹配不算",
			text:          "任務完成！🥇",
			promisePhrase: "任務完成！🥇",
			expected:      false, // 沒有 <promise> 標籤包裹
		},
		{
			name:          "空承諾詞",
			text:          "<promise></promise>",
			promisePhrase: "",
			expected:      false,
		},
		{
			name:          "自訂承諾詞",
			text:          "Done! <promise>COMPLETED</promise>",
			promisePhrase: "COMPLETED",
			expected:      true,
		},
		{
			name:          "承諾詞在代碼塊中也會被偵測",
			text:          "```\n<promise>任務完成！🥇</promise>\n```",
			promisePhrase: "任務完成！🥇",
			expected:      true,
		},
		{
			name:          "錯誤的承諾詞不匹配",
			text:          "<promise>不同的詞</promise>",
			promisePhrase: "任務完成！🥇",
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectPromise(tt.text, tt.promisePhrase)
			if result != tt.expected {
				t.Errorf("DetectPromise() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestPromiseDetector 測試 PromiseDetector 的狀態管理
func TestPromiseDetector(t *testing.T) {
	pd := NewPromiseDetector("任務完成！🥇")

	// 初始狀態
	if pd.IsDetected() {
		t.Error("新建的 PromiseDetector 不應該已偵測")
	}

	// 檢查不匹配的行
	pd.Check("正在處理中...")
	if pd.IsDetected() {
		t.Error("不匹配的行不應觸發偵測")
	}

	// 檢查匹配的行
	pd.Check("<promise>任務完成！🥇</promise>")
	if !pd.IsDetected() {
		t.Error("匹配的行應該觸發偵測")
	}

	// 重置
	pd.Reset()
	if pd.IsDetected() {
		t.Error("重置後不應該已偵測")
	}
}

// TestPromiseDetectorCheckFull 測試完整輸出偵測
func TestPromiseDetectorCheckFull(t *testing.T) {
	pd := NewPromiseDetector("DONE")

	fullOutput := `修復完成摘要:
- 修改了 file1.go
- 修改了 file2.go
所有測試通過。
<promise>DONE</promise>`

	result := pd.CheckFull(fullOutput)
	if !result {
		t.Error("CheckFull 應該回傳 true")
	}
	if !pd.IsDetected() {
		t.Error("CheckFull 後應該標記為已偵測")
	}
}

// TestPromiseDetectorDefaultPhrase 測試預設承諾詞
func TestPromiseDetectorDefaultPhrase(t *testing.T) {
	pd := NewPromiseDetector("")
	if pd.GetPhrase() != DefaultPromisePhrase {
		t.Errorf("空字串應該使用預設承諾詞，got %q", pd.GetPhrase())
	}
}

// TestBuildSystemPrompt 測試 System Prompt 構建
func TestBuildSystemPrompt(t *testing.T) {
	prompt := BuildSystemPrompt("任務完成！🥇")

	// 檢查承諾詞是否被正確嵌入
	if !strings.Contains(prompt, `<promise>任務完成！🥇</promise>`) {
		t.Error("System prompt 應該包含嵌入的承諾詞")
	}

	// 檢查不包含模板佔位符
	if strings.Contains(prompt, "{{PROMISE}}") {
		t.Error("System prompt 不應包含未替換的佔位符")
	}
}

// TestWrapPromptWithSystemInstructions 測試 prompt 包裝
func TestWrapPromptWithSystemInstructions(t *testing.T) {
	wrapped := WrapPromptWithSystemInstructions("修復所有錯誤", "任務完成！🥇", 3, 10)

	// 應該包含原始 prompt
	if !strings.Contains(wrapped, "修復所有錯誤") {
		t.Error("包裝後的 prompt 應該包含原始 prompt")
	}

	// 應該包含迭代資訊
	if !strings.Contains(wrapped, "[Iteration 3/10]") {
		t.Error("包裝後的 prompt 應該包含迭代資訊")
	}

	// 應該包含 system prompt 的關鍵內容
	if !strings.Contains(wrapped, "Ralph Loop System Instructions") {
		t.Error("包裝後的 prompt 應該包含 system prompt")
	}

	// 應該包含承諾詞說明
	if !strings.Contains(wrapped, `<promise>任務完成！🥇</promise>`) {
		t.Error("包裝後的 prompt 應該包含承諾詞說明")
	}
}

// TestItoa 測試簡易整數轉字串
func TestItoa(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{10, "10"},
		{123, "123"},
		{-5, "-5"},
	}

	for _, tt := range tests {
		result := itoa(tt.input)
		if result != tt.expected {
			t.Errorf("itoa(%d) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
