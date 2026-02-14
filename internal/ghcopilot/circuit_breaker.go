package ghcopilot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CircuitBreakerState 代表熔斷器的狀態
type CircuitBreakerState string

const (
	// StateClosed 正常運作
	StateClosed CircuitBreakerState = "CLOSED"
	// StateHalfOpen 試探性恢復
	StateHalfOpen CircuitBreakerState = "HALF_OPEN"
	// StateOpen 停止執行
	StateOpen CircuitBreakerState = "OPEN"
)

// CircuitBreaker 用於防止失控迴圈
type CircuitBreaker struct {
	state            CircuitBreakerState
	noProgressLoops  int
	sameErrorLoops   int
	totalErrors      int
	lastStateChange  time.Time
	stateFile        string
	failureThreshold int      // 無進展或相同錯誤達到此閾值時打開
	successThreshold int      // 成功達到此次數時關閉
	successCount     int      // 目前成功計數
	lastErrors       []string // 最後 3 個錯誤
}

// NewCircuitBreaker 建立新的熔斷器
func NewCircuitBreaker(workDir string) *CircuitBreaker {
	return &CircuitBreaker{
		state:            StateClosed,
		noProgressLoops:  0,
		sameErrorLoops:   0,
		totalErrors:      0,
		lastStateChange:  time.Now(),
		stateFile:        filepath.Join(workDir, ".circuit_breaker_state"),
		failureThreshold: 3, // 3 次無進展或相同錯誤
		successThreshold: 1, // 1 次成功即可關閉
		successCount:     0,
		lastErrors:       []string{},
	}
}

// GetState 取得目前狀態
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	return cb.state
}

// IsClosed 檢查是否為關閉狀態（正常運作）
func (cb *CircuitBreaker) IsClosed() bool {
	return cb.state == StateClosed
}

// IsOpen 檢查是否為開啟狀態（停止執行）
func (cb *CircuitBreaker) IsOpen() bool {
	return cb.state == StateOpen
}

// IsHalfOpen 檢查是否為半開狀態（試探性恢復）
func (cb *CircuitBreaker) IsHalfOpen() bool {
	return cb.state == StateHalfOpen
}

// RecordSuccess 記錄成功執行
func (cb *CircuitBreaker) RecordSuccess() {
	if cb.state == StateOpen {
		// 如果在開啟狀態，轉換為半開狀態試探
		cb.state = StateHalfOpen
		cb.lastStateChange = time.Now()
	}

	if cb.state == StateHalfOpen {
		cb.successCount++
		if cb.successCount >= cb.successThreshold {
			cb.state = StateClosed
			cb.lastStateChange = time.Now()
			cb.noProgressLoops = 0
			cb.sameErrorLoops = 0
			cb.successCount = 0
		}
	}

	cb.noProgressLoops = 0
	cb.sameErrorLoops = 0
}

// RecordNoProgress 記錄無進展
func (cb *CircuitBreaker) RecordNoProgress() {
	cb.noProgressLoops++
	cb.successCount = 0 // 重置成功計數

	if cb.noProgressLoops >= cb.failureThreshold {
		cb.openCircuit("無進展迴圈已達 3 次")
	}
}

// RecordSameError 記錄相同錯誤
func (cb *CircuitBreaker) RecordSameError(errorMsg string) {
	normalized := normalizeErrorMsg(errorMsg)

	// 檢查是否與最後一個錯誤相同
	if len(cb.lastErrors) > 0 && cb.lastErrors[len(cb.lastErrors)-1] == normalized {
		cb.sameErrorLoops++
	} else {
		cb.sameErrorLoops = 1
	}

	// 保持最後 3 個錯誤
	cb.lastErrors = append(cb.lastErrors, normalized)
	if len(cb.lastErrors) > 3 {
		cb.lastErrors = cb.lastErrors[1:]
	}

	cb.totalErrors++
	cb.successCount = 0 // 重置成功計數

	if cb.sameErrorLoops >= 5 {
		cb.openCircuit("相同錯誤已出現 5 次")
	}
}

// openCircuit 打開熔斷器
func (cb *CircuitBreaker) openCircuit(reason string) {
	if cb.state != StateOpen {
		cb.state = StateOpen
		cb.lastStateChange = time.Now()
		fmt.Printf("⚠️ 熔斷器打開: %s\n", reason)
		cb.SaveState()
	}
}

// Reset 手動重置熔斷器
func (cb *CircuitBreaker) Reset() {
	cb.state = StateClosed
	cb.noProgressLoops = 0
	cb.sameErrorLoops = 0
	cb.successCount = 0
	cb.lastStateChange = time.Now()
	cb.totalErrors = 0
	cb.lastErrors = []string{}
	cb.SaveState()
	fmt.Println("✅ 熔斷器已重置")
}

// GetStats 取得統計資訊
func (cb *CircuitBreaker) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"state":             cb.state,
		"no_progress_loops": cb.noProgressLoops,
		"same_error_loops":  cb.sameErrorLoops,
		"total_errors":      cb.totalErrors,
		"last_state_change": cb.lastStateChange.Format(time.RFC3339),
		"time_in_state":     time.Since(cb.lastStateChange).String(),
	}
}

// SaveState 儲存狀態到檔案
func (cb *CircuitBreaker) SaveState() error {
	data := map[string]interface{}{
		"state":             cb.state,
		"no_progress_loops": cb.noProgressLoops,
		"same_error_loops":  cb.sameErrorLoops,
		"total_errors":      cb.totalErrors,
		"last_errors":       cb.lastErrors,
		"timestamp":         time.Now().Unix(),
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("無法序列化狀態: %w", err)
	}

	return os.WriteFile(cb.stateFile, jsonData, 0644)
}

// LoadState 從檔案載入狀態
func (cb *CircuitBreaker) LoadState() error {
	if _, err := os.Stat(cb.stateFile); err != nil {
		return nil // 檔案不存在，使用預設值
	}

	data, err := os.ReadFile(cb.stateFile)
	if err != nil {
		return fmt.Errorf("無法讀取狀態檔案: %w", err)
	}

	var state map[string]interface{}
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("無法解析狀態檔案: %w", err)
	}

	// 恢復狀態
	if s, ok := state["state"].(string); ok {
		cb.state = CircuitBreakerState(s)
	}

	if n, ok := state["no_progress_loops"].(float64); ok {
		cb.noProgressLoops = int(n)
	}

	if s, ok := state["same_error_loops"].(float64); ok {
		cb.sameErrorLoops = int(s)
	}

	if t, ok := state["total_errors"].(float64); ok {
		cb.totalErrors = int(t)
	}

	if errs, ok := state["last_errors"].([]interface{}); ok {
		cb.lastErrors = []string{}
		for _, e := range errs {
			if errStr, ok := e.(string); ok {
				cb.lastErrors = append(cb.lastErrors, errStr)
			}
		}
	}

	return nil
}

// normalizeErrorMsg 正規化錯誤訊息
func normalizeErrorMsg(msg string) string {
	// 簡單的正規化：轉小寫並移除多餘空白
	normalized := strings.ToLower(msg)
	normalized = strings.TrimSpace(normalized)

	// 只取前 100 字符
	if len(normalized) > 100 {
		normalized = normalized[:100]
	}

	return normalized
}
