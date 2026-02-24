package ghcopilot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ExitConditionType 代表退出條件的類型
type ExitConditionType string

const (
	// CompletionCondition 完成條件（基於回應分析）
	CompletionCondition ExitConditionType = "completion"
	// TestSaturationCondition 測試飽和條件（連續測試迴圈）
	TestSaturationCondition ExitConditionType = "test_saturation"
	// DoneSignalCondition 完成信號條件（AI 明確發出 done）
	DoneSignalCondition ExitConditionType = "done_signal"
	// PlanCompleteCondition 計劃完成條件（@fix_plan.md 全部完成）
	PlanCompleteCondition ExitConditionType = "plan_complete"
	// RateLimitCondition 速率限制條件（達到 API 限制）
	RateLimitCondition ExitConditionType = "rate_limit"
)

// ExitSignals 追蹤退出訊號
type ExitSignals struct {
	TestOnlyLoops   int         // 連續測試迴圈數
	DoneSignals     int         // "done" 訊號次數
	CompletionCount int         // 完成指標數量
	LastSignalTime  time.Time   // 最後訊號時間
	SignalWindow    []time.Time // 滾動視窗（最近 5 個訊號的時間）
	RateLimitHits   int         // 速率限制觸發次數
}

// ExitDetector 用於決定是否應該優雅退出
type ExitDetector struct {
	workDir               string
	signalFile            string
	signals               ExitSignals
	exitConditionsTracker map[ExitConditionType]int
	rateLimitResetTime    time.Time
	rateLimitCallCount    int
	mu                    sync.RWMutex
}

// NewExitDetector 建立新的退出偵測器
func NewExitDetector(workDir string) *ExitDetector {
	return &ExitDetector{
		workDir:               workDir,
		signalFile:            filepath.Join(workDir, ".exit_signals"),
		signals:               ExitSignals{},
		exitConditionsTracker: make(map[ExitConditionType]int),
		rateLimitResetTime:    time.Now().Add(1 * time.Hour),
		rateLimitCallCount:    0,
	}
}

// RecordTestOnlyLoop 記錄測試專屬迴圈
func (ed *ExitDetector) RecordTestOnlyLoop() {
	ed.mu.Lock()
	defer ed.mu.Unlock()

	ed.signals.TestOnlyLoops++
	ed.signals.LastSignalTime = time.Now()
	ed.recordSignalTime()

	if ed.signals.TestOnlyLoops >= 3 {
		ed.exitConditionsTracker[TestSaturationCondition]++
	}
}

// RecordDoneSignal 記錄 "done" 訊號
func (ed *ExitDetector) RecordDoneSignal() {
	ed.mu.Lock()
	defer ed.mu.Unlock()

	ed.signals.DoneSignals++
	ed.signals.LastSignalTime = time.Now()
	ed.recordSignalTime()

	if ed.signals.DoneSignals >= 2 {
		ed.exitConditionsTracker[DoneSignalCondition]++
	}
}

// RecordCompletionIndicator 記錄完成指標
func (ed *ExitDetector) RecordCompletionIndicator() {
	ed.mu.Lock()
	defer ed.mu.Unlock()

	ed.signals.CompletionCount++
	ed.signals.LastSignalTime = time.Now()
	ed.recordSignalTime()

	if ed.signals.CompletionCount >= 2 {
		ed.exitConditionsTracker[CompletionCondition]++
	}
}

// RecordRateLimitHit 記錄速率限制觸發
func (ed *ExitDetector) RecordRateLimitHit() {
	ed.mu.Lock()
	defer ed.mu.Unlock()

	ed.signals.RateLimitHits++

	// 檢查是否需要重置計數
	if time.Now().After(ed.rateLimitResetTime) {
		ed.rateLimitCallCount = 0
		ed.rateLimitResetTime = time.Now().Add(1 * time.Hour)
	}

	ed.exitConditionsTracker[RateLimitCondition]++
}

// recordSignalTime 記錄訊號時間到滾動視窗
func (ed *ExitDetector) recordSignalTime() {
	ed.signals.SignalWindow = append(ed.signals.SignalWindow, time.Now())

	// 保持最近 5 個訊號
	if len(ed.signals.SignalWindow) > 5 {
		ed.signals.SignalWindow = ed.signals.SignalWindow[1:]
	}
}

// ShouldExitGracefully 判斷是否應該優雅地退出
func (ed *ExitDetector) ShouldExitGracefully(analyzerScore int) bool {
	ed.mu.RLock()
	defer ed.mu.RUnlock()

	// 條件 1: 完成條件滿足（基於 ralph-claude-code）
	// 需要完成指標 >= 2 且分數 >= 20
	if ed.signals.CompletionCount >= 2 && analyzerScore >= 20 {
		return true
	}

	// 條件 2: 完成信號達到
	if ed.signals.DoneSignals >= 2 {
		return true
	}

	// 條件 3: 測試飽和（連續 3+ 個測試迴圈）
	if ed.signals.TestOnlyLoops >= 3 {
		return true
	}

	// 條件 4: 速率限制達到
	if ed.signals.RateLimitHits > 0 {
		return true
	}

	return false
}

// GetExitReason 取得退出原因
func (ed *ExitDetector) GetExitReason(analyzerScore int) string {
	ed.mu.RLock()
	defer ed.mu.RUnlock()

	// 按優先順序檢查
	if ed.signals.CompletionCount >= 2 && analyzerScore >= 20 {
		return fmt.Sprintf("完成條件滿足 (分數: %d, 指標: %d)", analyzerScore, ed.signals.CompletionCount)
	}

	if ed.signals.DoneSignals >= 2 {
		return fmt.Sprintf("完成訊號達到 (%d 次)", ed.signals.DoneSignals)
	}

	if ed.signals.TestOnlyLoops >= 3 {
		return fmt.Sprintf("測試飽和 (%d 個連續測試迴圈)", ed.signals.TestOnlyLoops)
	}

	if ed.signals.RateLimitHits > 0 {
		return "達到 API 速率限制"
	}

	return "未知原因"
}

// CheckRateLimit 檢查是否超過速率限制
func (ed *ExitDetector) CheckRateLimit(callsPerHour int) (allowed bool, timeUntilReset time.Duration) {
	ed.mu.Lock()
	defer ed.mu.Unlock()

	now := time.Now()

	// 重置時間已到
	if now.After(ed.rateLimitResetTime) {
		ed.rateLimitCallCount = 0
		ed.rateLimitResetTime = now.Add(1 * time.Hour)
	}

	ed.rateLimitCallCount++

	if ed.rateLimitCallCount > callsPerHour {
		return false, ed.rateLimitResetTime.Sub(now)
	}

	return true, 0
}

// GetRateLimitStatus 取得速率限制狀態
func (ed *ExitDetector) GetRateLimitStatus(callsPerHour int) map[string]interface{} {
	ed.mu.RLock()
	defer ed.mu.RUnlock()

	used := ed.rateLimitCallCount
	remaining := callsPerHour - used
	if remaining < 0 {
		remaining = 0
	}

	return map[string]interface{}{
		"used":                used,
		"remaining":           remaining,
		"limit_per_hour":      callsPerHour,
		"reset_time":          ed.rateLimitResetTime.Format(time.RFC3339),
		"time_until_reset":    ed.rateLimitResetTime.Sub(time.Now()).String(),
		"rate_limit_exceeded": ed.rateLimitCallCount > callsPerHour,
	}
}

// Reset 重置所有訊號
func (ed *ExitDetector) Reset() {
	ed.mu.Lock()
	defer ed.mu.Unlock()

	ed.signals = ExitSignals{}
	ed.exitConditionsTracker = make(map[ExitConditionType]int)
	ed.rateLimitCallCount = 0
}

// SaveSignals 儲存訊號到檔案
func (ed *ExitDetector) SaveSignals() error {
	ed.mu.RLock()
	defer ed.mu.RUnlock()

	data := map[string]interface{}{
		"test_only_loops":  ed.signals.TestOnlyLoops,
		"done_signals":     ed.signals.DoneSignals,
		"completion_count": ed.signals.CompletionCount,
		"last_signal_time": ed.signals.LastSignalTime.Unix(),
		"signal_window":    len(ed.signals.SignalWindow),
		"rate_limit_hits":  ed.signals.RateLimitHits,
		"timestamp":        time.Now().Unix(),
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("無法序列化訊號: %w", err)
	}

	return os.WriteFile(ed.signalFile, jsonData, 0600)
}

// LoadSignals 從檔案載入訊號
func (ed *ExitDetector) LoadSignals() error {
	ed.mu.Lock()
	defer ed.mu.Unlock()

	data, err := os.ReadFile(ed.signalFile)
	if err != nil {
		return nil // 檔案不存在是正常的
	}

	var signals map[string]interface{}
	if err := json.Unmarshal(data, &signals); err != nil {
		return fmt.Errorf("無法解析訊號檔案: %w", err)
	}

	// 恢復訊號
	if test, ok := signals["test_only_loops"].(float64); ok {
		ed.signals.TestOnlyLoops = int(test)
	}

	if done, ok := signals["done_signals"].(float64); ok {
		ed.signals.DoneSignals = int(done)
	}

	if comp, ok := signals["completion_count"].(float64); ok {
		ed.signals.CompletionCount = int(comp)
	}

	if hits, ok := signals["rate_limit_hits"].(float64); ok {
		ed.signals.RateLimitHits = int(hits)
	}

	return nil
}

// GetSignalsSummary 取得訊號摘要
func (ed *ExitDetector) GetSignalsSummary() map[string]interface{} {
	ed.mu.RLock()
	defer ed.mu.RUnlock()

	return map[string]interface{}{
		"test_only_loops":    ed.signals.TestOnlyLoops,
		"done_signals":       ed.signals.DoneSignals,
		"completion_count":   ed.signals.CompletionCount,
		"rate_limit_hits":    ed.signals.RateLimitHits,
		"signal_window_size": len(ed.signals.SignalWindow),
		"last_signal_time":   ed.signals.LastSignalTime.Format(time.RFC3339),
		"conditions_met":     len(ed.exitConditionsTracker),
	}
}

// GetExitConditions 取得所有觸發的退出條件
func (ed *ExitDetector) GetExitConditions() map[ExitConditionType]int {
	ed.mu.RLock()
	defer ed.mu.RUnlock()

	// 返回副本避免外部修改
	result := make(map[ExitConditionType]int)
	for k, v := range ed.exitConditionsTracker {
		result[k] = v
	}

	return result
}
