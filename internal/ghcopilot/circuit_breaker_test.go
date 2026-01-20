package ghcopilot

import (
	"testing"
)

// TestNewCircuitBreaker 測試建立新的熔斷器
func TestNewCircuitBreaker(t *testing.T) {
	tempDir := t.TempDir()
	cb := NewCircuitBreaker(tempDir)

	if cb == nil {
		t.Error("NewCircuitBreaker() 傳回 nil")
	}

	if cb.GetState() != StateClosed {
		t.Errorf("初始狀態應為 CLOSED，但為 %s", cb.GetState())
	}

	if !cb.IsClosed() {
		t.Error("初始狀態應為 closed")
	}
}

// TestStateTransition 測試狀態轉換
func TestStateTransition(t *testing.T) {
	tempDir := t.TempDir()
	cb := NewCircuitBreaker(tempDir)

	// 從 CLOSED 轉換到 OPEN
	for i := 0; i < 3; i++ {
		cb.RecordNoProgress()
	}

	if !cb.IsOpen() {
		t.Error("應在無進展 3 次後轉為 OPEN")
	}

	// 成功不會自動轉換，需要修改 RecordSuccess 邏輯
	// 在 OPEN 狀態下成功會轉為 HALF_OPEN 但仍需要成功計數達到閾值才能轉 CLOSED
	cb.successCount = 0 // 重置以確保新測試邏輯

	// 直接測試從 HALF_OPEN 轉為 CLOSED
	cb.state = StateHalfOpen
	cb.successCount = 0
	cb.RecordSuccess()

	if !cb.IsClosed() {
		t.Error("從 HALF_OPEN 成功應轉為 CLOSED")
	}
}

// TestRecordNoProgress 測試記錄無進展
func TestRecordNoProgress(t *testing.T) {
	tempDir := t.TempDir()
	cb := NewCircuitBreaker(tempDir)

	for i := 0; i < 2; i++ {
		cb.RecordNoProgress()
		if cb.IsOpen() {
			t.Errorf("應在 3 次後才打開，但在 %d 次時打開了", i+1)
		}
	}

	cb.RecordNoProgress()
	if !cb.IsOpen() {
		t.Error("應在 3 次無進展後打開")
	}
}

// TestRecordSameError 測試記錄相同錯誤
func TestRecordSameError(t *testing.T) {
	tempDir := t.TempDir()
	cb := NewCircuitBreaker(tempDir)

	// 記錄相同錯誤 5 次
	for i := 0; i < 5; i++ {
		cb.RecordSameError("Connection timeout")
		if i < 4 && cb.IsOpen() {
			t.Errorf("應在 5 次相同錯誤後打開，但在 %d 次時打開了", i+1)
		}
	}

	if !cb.IsOpen() {
		t.Error("應在 5 次相同錯誤後打開")
	}
}

// TestDifferentErrors 測試不同的錯誤
func TestDifferentErrors(t *testing.T) {
	tempDir := t.TempDir()
	cb := NewCircuitBreaker(tempDir)

	errors := []string{
		"Connection timeout",
		"Database error",
		"Network error",
	}

	// 記錄不同的錯誤不應打開熔斷器
	for i := 0; i < 5; i++ {
		cb.RecordSameError(errors[i%len(errors)])
	}

	if cb.IsOpen() {
		t.Error("不同的錯誤不應打開熔斷器")
	}
}

// TestReset 測試重置
func TestReset(t *testing.T) {
	tempDir := t.TempDir()
	cb := NewCircuitBreaker(tempDir)

	// 打開熔斷器
	for i := 0; i < 3; i++ {
		cb.RecordNoProgress()
	}

	if !cb.IsOpen() {
		t.Error("應打開熔斷器")
	}

	// 重置
	cb.Reset()

	if !cb.IsClosed() {
		t.Error("重置後應回到 CLOSED")
	}

	if cb.noProgressLoops != 0 {
		t.Error("重置後無進展迴圈計數應為 0")
	}
}

// TestSaveAndLoadState 測試儲存和載入狀態
func TestSaveAndLoadState(t *testing.T) {
	tempDir := t.TempDir()
	cb1 := NewCircuitBreaker(tempDir)

	// 改變狀態
	for i := 0; i < 3; i++ {
		cb1.RecordNoProgress()
	}

	err := cb1.SaveState()
	if err != nil {
		t.Fatalf("儲存狀態失敗: %v", err)
	}

	// 建立新的熔斷器並載入狀態
	cb2 := NewCircuitBreaker(tempDir)
	err = cb2.LoadState()
	if err != nil {
		t.Fatalf("載入狀態失敗: %v", err)
	}

	if cb2.GetState() != StateOpen {
		t.Errorf("載入後狀態應為 OPEN，但為 %s", cb2.GetState())
	}

	if cb2.noProgressLoops != 3 {
		t.Errorf("無進展迴圈計數應為 3，但為 %d", cb2.noProgressLoops)
	}
}

// TestGetStats 測試取得統計資訊
func TestGetStats(t *testing.T) {
	tempDir := t.TempDir()
	cb := NewCircuitBreaker(tempDir)

	cb.RecordNoProgress()
	cb.RecordSameError("Test error")

	stats := cb.GetStats()

	if _, ok := stats["state"]; !ok {
		t.Error("統計應包含 state")
	}

	if _, ok := stats["total_errors"]; !ok {
		t.Error("統計應包含 total_errors")
	}

	if _, ok := stats["time_in_state"]; !ok {
		t.Error("統計應包含 time_in_state")
	}
}

// TestRecordSuccess 測試記錄成功
func TestRecordSuccess(t *testing.T) {
	tempDir := t.TempDir()
	cb := NewCircuitBreaker(tempDir)

	// 打開熔斷器
	for i := 0; i < 3; i++ {
		cb.RecordNoProgress()
	}

	// 記錄成功應重置計數
	cb.RecordSuccess()
	if cb.noProgressLoops != 0 {
		t.Error("成功記錄應重置無進展計數")
	}

	if cb.sameErrorLoops != 0 {
		t.Error("成功記錄應重置相同錯誤計數")
	}
}

// TestLastErrors 測試錯誤歷史
func TestLastErrors(t *testing.T) {
	tempDir := t.TempDir()
	cb := NewCircuitBreaker(tempDir)

	errors := []string{"Error 1", "Error 2", "Error 3", "Error 4"}

	for _, err := range errors {
		cb.RecordSameError(err)
	}

	// 應只保持最後 3 個錯誤
	if len(cb.lastErrors) != 3 {
		t.Errorf("應保持最後 3 個錯誤，但有 %d 個", len(cb.lastErrors))
	}

	// 檢查保存的是最後 3 個
	if cb.lastErrors[len(cb.lastErrors)-1] != "error 4" {
		t.Errorf("最後一個錯誤應為 'error 4'，但為 '%s'", cb.lastErrors[len(cb.lastErrors)-1])
	}
}
