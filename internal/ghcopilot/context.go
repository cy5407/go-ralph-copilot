package ghcopilot

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// ExecutionContext 代表單次迴圈執行的完整上下文
type ExecutionContext struct {
	// 基本資訊
	LoopID     string    `json:"loop_id"`     // 迴圈 ID (UUID)
	LoopIndex  int       `json:"loop_index"`  // 迴圈索引 (0, 1, 2, ...)
	Timestamp  time.Time `json:"timestamp"`   // 時間戳
	DurationMs int64     `json:"duration_ms"` // 執行時間（毫秒）

	// 使用者輸入
	UserPrompt   string `json:"user_prompt"`   // 使用者原始請求
	UserFeedback string `json:"user_feedback"` // 使用者反饋（如有）

	// CLI 執行結果
	CLICommand  string `json:"cli_command"`   // 執行的 CLI 指令
	CLIOutput   string `json:"cli_output"`    // CLI 輸出（完整）
	CLIExitCode int    `json:"cli_exit_code"` // 退出碼

	// 輸出解析結果
	ParsedCodeBlocks     []string `json:"parsed_code_blocks"`     // 提取的程式碼區塊
	ExtractedCodeBlocks  []string `json:"extracted_code_blocks"`  // 完整提取的程式碼區塊
	ParsedOptions        []string `json:"parsed_options"`         // 提取的選項
	CleanedOutput        string   `json:"cleaned_output"`         // 清除 Markdown 後的輸出

	// 回應分析
	CompletionScore      int         `json:"completion_score"`      // 完成分數
	CompletionIndicators []string    `json:"completion_indicators"` // 完成指標清單
	StructuredStatus     *LoopStatus `json:"structured_status"`     // 結構化狀態
	IsTestOnlyLoop       bool        `json:"is_test_only_loop"`     // 是否為測試專屬迴圈
	IsStuckState         bool        `json:"is_stuck_state"`        // 是否卡住

	// 熔斷器狀態
	CircuitBreakerState string   `json:"circuit_breaker_state"`  // CLOSED/OPEN/HALF_OPEN
	LoopNoProgressCount int      `json:"loop_no_progress_count"` // 無進展計數
	ErrorHistory        []string `json:"error_history"`          // 錯誤歷史

	// 迴圈決策
	ShouldContinue bool   `json:"should_continue"` // 是否應繼續迴圈
	ExitReason     string `json:"exit_reason"`     // 退出理由（如有）

	// Metadata
	Model    string                 `json:"model,omitempty"` // 使用的 AI 模型
	Metadata map[string]interface{} `json:"metadata"`        // 其他 metadata
}

// LoopStatus 代表結構化的迴圈狀態輸出
type LoopStatus struct {
	Status       string `json:"status"`        // CONTINUE, DONE, ERROR
	ExitSignal   bool   `json:"exit_signal"`   // 是否應退出
	TasksDone    string `json:"tasks_done"`    // 完成的任務數 (e.g., "3/5")
	NextStep     string `json:"next_step"`     // 下一步（如有）
	ErrorMessage string `json:"error_message"` // 錯誤訊息（如有）
}

// ContextManager 管理整個迴圈的上下文歷史記錄
type ContextManager struct {
	mu             sync.RWMutex
	currentLoop    *ExecutionContext
	loopHistory    []*ExecutionContext
	maxHistorySize int
	startTime      time.Time
	totalDuration  time.Duration
	successCount   int
	errorCount     int
}

// NewContextManager 建立新的上下文管理器
func NewContextManager() *ContextManager {
	return &ContextManager{
		currentLoop:    nil,
		loopHistory:    make([]*ExecutionContext, 0),
		maxHistorySize: 100, // 預設保存最後 100 個迴圈
		startTime:      time.Now(),
		successCount:   0,
		errorCount:     0,
	}
}

// NewExecutionContext 建立新的執行上下文
func NewExecutionContext(loopIndex int, userPrompt string) *ExecutionContext {
	return &ExecutionContext{
		LoopID:       fmt.Sprintf("loop-%d-%d", time.Now().Unix(), loopIndex),
		LoopIndex:    loopIndex,
		Timestamp:    time.Now(),
		UserPrompt:   userPrompt,
		Metadata:     make(map[string]interface{}),
		ErrorHistory: make([]string, 0),
	}
}

// StartLoop 開始新的迴圈上下文
func (cm *ContextManager) StartLoop(loopIndex int, userPrompt string) *ExecutionContext {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	ctx := NewExecutionContext(loopIndex, userPrompt)
	cm.currentLoop = ctx
	return ctx
}

// UpdateCurrentLoop 更新當前迴圈的上下文
func (cm *ContextManager) UpdateCurrentLoop(fn func(*ExecutionContext)) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.currentLoop == nil {
		return fmt.Errorf("no active loop context")
	}

	fn(cm.currentLoop)
	return nil
}

// FinishLoop 完成當前迴圈，將其加入歷史記錄
func (cm *ContextManager) FinishLoop() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.currentLoop == nil {
		return fmt.Errorf("no active loop context to finish")
	}

	// 計算執行時間
	duration := time.Since(cm.currentLoop.Timestamp)
	cm.currentLoop.DurationMs = duration.Milliseconds()

	// 統計成功/失敗
	if cm.currentLoop.ShouldContinue || cm.currentLoop.ExitReason == "" {
		cm.successCount++
	} else {
		cm.errorCount++
	}

	// 加入歷史記錄
	cm.loopHistory = append(cm.loopHistory, cm.currentLoop)

	// 如果歷史記錄超過最大值，刪除最早的
	if len(cm.loopHistory) > cm.maxHistorySize {
		cm.loopHistory = cm.loopHistory[1:]
	}

	cm.totalDuration += duration
	cm.currentLoop = nil
	return nil
}

// GetCurrentLoop 取得當前迴圈的上下文
func (cm *ContextManager) GetCurrentLoop() *ExecutionContext {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.currentLoop
}

// GetLoopHistory 取得迴圈歷史記錄
func (cm *ContextManager) GetLoopHistory() []*ExecutionContext {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// 傳回副本，避免外部修改
	history := make([]*ExecutionContext, len(cm.loopHistory))
	copy(history, cm.loopHistory)
	return history
}

// GetLoopByIndex 根據迴圈索引取得特定的迴圈上下文
func (cm *ContextManager) GetLoopByIndex(index int) *ExecutionContext {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	for _, ctx := range cm.loopHistory {
		if ctx.LoopIndex == index {
			return ctx
		}
	}
	return nil
}

// GetSummary 取得整體執行摘要
func (cm *ContextManager) GetSummary() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	totalLoops := len(cm.loopHistory)
	successRate := 0.0
	if totalLoops > 0 {
		successRate = float64(cm.successCount) / float64(totalLoops) * 100
	}

	return map[string]interface{}{
		"total_loops":       totalLoops,
		"success_count":     cm.successCount,
		"error_count":       cm.errorCount,
		"success_rate":      fmt.Sprintf("%.1f%%", successRate),
		"total_duration_ms": cm.totalDuration.Milliseconds(),
		"avg_duration_ms": func() int64 {
			if totalLoops > 0 {
				return cm.totalDuration.Milliseconds() / int64(totalLoops)
			}
			return 0
		}(),
		"start_time": cm.startTime.Format(time.RFC3339),
		"elapsed":    fmt.Sprintf("%.2f s", time.Since(cm.startTime).Seconds()),
	}
}

// GetLastErrorContext 取得最後一個包含錯誤的迴圈
func (cm *ContextManager) GetLastErrorContext() *ExecutionContext {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	for i := len(cm.loopHistory) - 1; i >= 0; i-- {
		if cm.loopHistory[i].ExitReason != "" {
			return cm.loopHistory[i]
		}
	}
	return nil
}

// Clear 清空所有上下文（用於重新開始）
func (cm *ContextManager) Clear() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.currentLoop = nil
	cm.loopHistory = make([]*ExecutionContext, 0)
	cm.startTime = time.Now()
	cm.totalDuration = 0
	cm.successCount = 0
	cm.errorCount = 0
}

// SetMaxHistorySize 設定最大歷史記錄大小
func (cm *ContextManager) SetMaxHistorySize(size int) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.maxHistorySize = size
	// 如果當前歷史記錄超過新的大小，截截
	if len(cm.loopHistory) > size {
		cm.loopHistory = cm.loopHistory[len(cm.loopHistory)-size:]
	}
}

// ToJSON 將整個上下文歷史轉換為 JSON
func (cm *ContextManager) ToJSON() (string, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	data := map[string]interface{}{
		"summary": cm.getSummaryUnlocked(),
		"history": cm.loopHistory,
	}

	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

// getSummaryUnlocked 內部使用的摘要取得（不加鎖）
func (cm *ContextManager) getSummaryUnlocked() map[string]interface{} {
	totalLoops := len(cm.loopHistory)
	successRate := 0.0
	if totalLoops > 0 {
		successRate = float64(cm.successCount) / float64(totalLoops) * 100
	}

	return map[string]interface{}{
		"total_loops":       totalLoops,
		"success_count":     cm.successCount,
		"error_count":       cm.errorCount,
		"success_rate":      fmt.Sprintf("%.1f%%", successRate),
		"total_duration_ms": cm.totalDuration.Milliseconds(),
	}
}
