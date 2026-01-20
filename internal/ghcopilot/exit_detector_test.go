package ghcopilot

import (
	"testing"
	"time"
)

// TestNewExitDetector 測試建立新的退出偵測器
func TestNewExitDetector(t *testing.T) {
	tempDir := t.TempDir()
	ed := NewExitDetector(tempDir)

	if ed == nil {
		t.Error("NewExitDetector() 傳回 nil")
	}

	if ed.signals.TestOnlyLoops != 0 {
		t.Error("初始化時測試迴圈計數應為 0")
	}
}

// TestRecordTestOnlyLoop 測試記錄測試迴圈
func TestRecordTestOnlyLoop(t *testing.T) {
	tempDir := t.TempDir()
	ed := NewExitDetector(tempDir)

	for i := 1; i <= 3; i++ {
		ed.RecordTestOnlyLoop()
		if ed.signals.TestOnlyLoops != i {
			t.Errorf("測試迴圈計數應為 %d，但為 %d", i, ed.signals.TestOnlyLoops)
		}
	}

	// 檢查是否觸發退出條件
	conditions := ed.GetExitConditions()
	if conditions[TestSaturationCondition] == 0 {
		t.Error("應在 3 個測試迴圈後觸發退出條件")
	}
}

// TestRecordDoneSignal 測試記錄完成訊號
func TestRecordDoneSignal(t *testing.T) {
	tempDir := t.TempDir()
	ed := NewExitDetector(tempDir)

	ed.RecordDoneSignal()
	ed.RecordDoneSignal()

	if ed.signals.DoneSignals != 2 {
		t.Errorf("完成訊號計數應為 2，但為 %d", ed.signals.DoneSignals)
	}

	conditions := ed.GetExitConditions()
	if conditions[DoneSignalCondition] == 0 {
		t.Error("應在 2 個完成訊號後觸發退出條件")
	}
}

// TestRecordCompletionIndicator 測試記錄完成指標
func TestRecordCompletionIndicator(t *testing.T) {
	tempDir := t.TempDir()
	ed := NewExitDetector(tempDir)

	ed.RecordCompletionIndicator()
	ed.RecordCompletionIndicator()

	if ed.signals.CompletionCount != 2 {
		t.Errorf("完成指標計數應為 2，但為 %d", ed.signals.CompletionCount)
	}

	conditions := ed.GetExitConditions()
	if conditions[CompletionCondition] == 0 {
		t.Error("應在 2 個完成指標後觸發退出條件")
	}
}

// TestShouldExitGracefully 測試優雅退出判斷
func TestShouldExitGracefully(t *testing.T) {
	tempDir := t.TempDir()
	ed := NewExitDetector(tempDir)

	// 測試 1: 只有指標，分數不足
	ed.RecordCompletionIndicator()
	ed.RecordCompletionIndicator()
	if ed.ShouldExitGracefully(10) {
		t.Error("分數不足不應退出")
	}

	// 測試 2: 指標和分數都足夠
	if !ed.ShouldExitGracefully(25) {
		t.Error("指標和分數都足夠應該退出")
	}

	// 測試 3: 完成訊號
	ed2 := NewExitDetector(tempDir)
	ed2.RecordDoneSignal()
	ed2.RecordDoneSignal()
	if !ed2.ShouldExitGracefully(0) {
		t.Error("2 個完成訊號應該退出")
	}

	// 測試 4: 測試飽和
	ed3 := NewExitDetector(tempDir)
	for i := 0; i < 3; i++ {
		ed3.RecordTestOnlyLoop()
	}
	if !ed3.ShouldExitGracefully(0) {
		t.Error("測試飽和應該退出")
	}
}

// TestGetExitReason 測試取得退出原因
func TestGetExitReason(t *testing.T) {
	tempDir := t.TempDir()
	ed := NewExitDetector(tempDir)

	ed.RecordCompletionIndicator()
	ed.RecordCompletionIndicator()

	reason := ed.GetExitReason(25)
	if len(reason) == 0 {
		t.Error("退出原因不應為空")
	}

	if !contains(reason, "完成條件") {
		t.Error("退出原因應包含 '完成條件'")
	}
}

// TestCheckRateLimit 測試速率限制檢查
func TestCheckRateLimit(t *testing.T) {
	tempDir := t.TempDir()
	ed := NewExitDetector(tempDir)

	// 在限制以下 - 執行 100 次呼叫
	for i := 0; i < 100; i++ {
		allowed, _ := ed.CheckRateLimit(100)
		if !allowed {
			t.Errorf("第 %d 個呼叫應該被允許", i+1)
			break
		}
	}

	// 超過限制 - 第 101 次呼叫
	allowed, timeUntilReset := ed.CheckRateLimit(100)
	if allowed {
		t.Error("第 101 個呼叫應該被拒絕")
	}

	if timeUntilReset <= 0 {
		t.Error("應有重置時間")
	}
}

// TestGetRateLimitStatus 測試取得速率限制狀態
func TestGetRateLimitStatus(t *testing.T) {
	tempDir := t.TempDir()
	ed := NewExitDetector(tempDir)

	for i := 0; i < 30; i++ {
		ed.CheckRateLimit(100)
	}

	status := ed.GetRateLimitStatus(100)

	if used, ok := status["used"].(int); ok {
		if used != 30 {
			t.Errorf("已使用應為 30，但為 %d", used)
		}
	}

	if remaining, ok := status["remaining"].(int); ok {
		if remaining != 70 {
			t.Errorf("剩餘應為 70，但為 %d", remaining)
		}
	}
}

// TestSignalWindow 測試訊號滾動視窗
func TestSignalWindow(t *testing.T) {
	tempDir := t.TempDir()
	ed := NewExitDetector(tempDir)

	// 記錄 7 個訊號（超過 5 個限制）
	for i := 0; i < 7; i++ {
		ed.RecordCompletionIndicator()
	}

	// 應只保持最近 5 個
	if len(ed.signals.SignalWindow) != 5 {
		t.Errorf("訊號視窗應有 5 個，但有 %d 個", len(ed.signals.SignalWindow))
	}
}

// TestExitDetectorReset 測試重置
func TestExitDetectorReset(t *testing.T) {
	tempDir := t.TempDir()
	ed := NewExitDetector(tempDir)

	// 記錄一些訊號
	ed.RecordCompletionIndicator()
	ed.RecordDoneSignal()
	ed.RecordTestOnlyLoop()

	if ed.signals.CompletionCount == 0 || ed.signals.DoneSignals == 0 || ed.signals.TestOnlyLoops == 0 {
		t.Error("記錄訊號失敗")
	}

	// 重置
	ed.Reset()

	if ed.signals.CompletionCount != 0 || ed.signals.DoneSignals != 0 || ed.signals.TestOnlyLoops != 0 {
		t.Error("重置後訊號應全為 0")
	}
}

// TestSaveAndLoadSignals 測試儲存和載入訊號
func TestSaveAndLoadSignals(t *testing.T) {
	tempDir := t.TempDir()
	ed1 := NewExitDetector(tempDir)

	// 記錄訊號
	ed1.RecordCompletionIndicator()
	ed1.RecordCompletionIndicator()
	ed1.RecordDoneSignal()

	err := ed1.SaveSignals()
	if err != nil {
		t.Fatalf("儲存訊號失敗: %v", err)
	}

	// 建立新的偵測器並載入訊號
	ed2 := NewExitDetector(tempDir)
	err = ed2.LoadSignals()
	if err != nil {
		t.Fatalf("載入訊號失敗: %v", err)
	}

	if ed2.signals.CompletionCount != 2 {
		t.Errorf("載入後完成指標應為 2，但為 %d", ed2.signals.CompletionCount)
	}

	if ed2.signals.DoneSignals != 1 {
		t.Errorf("載入後完成訊號應為 1，但為 %d", ed2.signals.DoneSignals)
	}
}

// TestGetSignalsSummary 測試取得訊號摘要
func TestGetSignalsSummary(t *testing.T) {
	tempDir := t.TempDir()
	ed := NewExitDetector(tempDir)

	ed.RecordCompletionIndicator()
	ed.RecordDoneSignal()

	summary := ed.GetSignalsSummary()

	if _, ok := summary["test_only_loops"]; !ok {
		t.Error("摘要應包含 test_only_loops")
	}

	if _, ok := summary["done_signals"]; !ok {
		t.Error("摘要應包含 done_signals")
	}

	if _, ok := summary["completion_count"]; !ok {
		t.Error("摘要應包含 completion_count")
	}
}

// TestGetExitConditions 測試取得退出條件
func TestGetExitConditions(t *testing.T) {
	tempDir := t.TempDir()
	ed := NewExitDetector(tempDir)

	ed.RecordDoneSignal()
	ed.RecordDoneSignal()

	conditions := ed.GetExitConditions()

	if count, ok := conditions[DoneSignalCondition]; !ok || count == 0 {
		t.Error("應有完成訊號條件")
	}
}

// TestMultipleExitConditions 測試多個退出條件
func TestMultipleExitConditions(t *testing.T) {
	tempDir := t.TempDir()
	ed := NewExitDetector(tempDir)

	// 觸發多個條件
	for i := 0; i < 3; i++ {
		ed.RecordTestOnlyLoop()
	}
	ed.RecordDoneSignal()
	ed.RecordDoneSignal()

	if !ed.ShouldExitGracefully(0) {
		t.Error("多個條件滿足應該退出")
	}

	conditions := ed.GetExitConditions()
	if len(conditions) < 2 {
		t.Errorf("應有至少 2 個退出條件，但只有 %d 個", len(conditions))
	}
}

// TestRateLimitResetAfterHour 測試速率限制在一小時後重置
func TestRateLimitResetAfterHour(t *testing.T) {
	tempDir := t.TempDir()
	ed := NewExitDetector(tempDir)

	// 使用完全部額度
	for i := 0; i < 100; i++ {
		ed.CheckRateLimit(100)
	}

	// 應該超過限制
	allowed, _ := ed.CheckRateLimit(100)
	if allowed {
		t.Error("應超過限制")
	}

	// 手動調整重置時間（模擬一小時後）
	ed.rateLimitResetTime = time.Now().Add(-1 * time.Second)

	// 現在應該被允許
	allowed, _ = ed.CheckRateLimit(100)
	if !allowed {
		t.Error("重置後應被允許")
	}
}
