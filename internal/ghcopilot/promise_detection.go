package ghcopilot

import (
	"strings"
	"sync"
)

// Promise Detection 機制
// 參考自 doggy8088/copilot-ralph 專案的完成偵測設計
// https://github.com/doggy8088/copilot-ralph
//
// 核心概念：透過 System Prompt 約束 AI 在任務完成時輸出特定的
// <promise>phrase</promise> 標籤，再由程式端進行硬匹配偵測。
// 這比關鍵字評分更可靠，因為：
// 1. 不依賴 AI 的自然語言輸出格式
// 2. <promise> 標籤不會在正常輸出中意外出現
// 3. AI 自己決定何時完成，而非程式猜測

const (
	// DefaultPromisePhrase 預設的完成承諾詞
	DefaultPromisePhrase = "任務完成！🥇"
)

// PromiseDetector 用於偵測 AI 輸出中的完成承諾
type PromiseDetector struct {
	promisePhrase string
	detected      bool
	mu            sync.RWMutex
}

// NewPromiseDetector 建立新的承諾偵測器
func NewPromiseDetector(phrase string) *PromiseDetector {
	if phrase == "" {
		phrase = DefaultPromisePhrase
	}
	return &PromiseDetector{
		promisePhrase: phrase,
	}
}

// DetectPromise 檢查文字中是否包含完成承諾
func DetectPromise(text string, promisePhrase string) bool {
	if promisePhrase == "" {
		return false
	}
	wrapped := "<promise>" + promisePhrase + "</promise>"
	return strings.Contains(text, wrapped)
}

// Check 檢查一行文字是否包含完成承諾（串流模式用）
// 每當收到新的串流行時呼叫此方法
func (pd *PromiseDetector) Check(line string) bool {
	if DetectPromise(line, pd.promisePhrase) {
		pd.mu.Lock()
		pd.detected = true
		pd.mu.Unlock()
		return true
	}
	return false
}

// CheckFull 檢查完整輸出是否包含完成承諾
func (pd *PromiseDetector) CheckFull(fullOutput string) bool {
	if DetectPromise(fullOutput, pd.promisePhrase) {
		pd.mu.Lock()
		pd.detected = true
		pd.mu.Unlock()
		return true
	}
	return false
}

// IsDetected 回傳是否已偵測到完成承諾
func (pd *PromiseDetector) IsDetected() bool {
	pd.mu.RLock()
	defer pd.mu.RUnlock()
	return pd.detected
}

// Reset 重置偵測狀態
func (pd *PromiseDetector) Reset() {
	pd.mu.Lock()
	defer pd.mu.Unlock()
	pd.detected = false
}

// GetPhrase 取得承諾詞
func (pd *PromiseDetector) GetPhrase() string {
	return pd.promisePhrase
}
