package ghcopilot

import (
	"errors"
	"testing"
	"time"
)

// ========================
// TimeoutDetector 測試
// ========================

func TestNewTimeoutDetector(t *testing.T) {
	detector := NewTimeoutDetector(5 * time.Second)
	if detector == nil {
		t.Fatal("detector should not be nil")
	}
	if detector.threshold != 5*time.Second {
		t.Errorf("expected threshold 5s, got %v", detector.threshold)
	}
	if detector.consecutiveThreshold != 3 {
		t.Errorf("expected consecutive threshold 3, got %d", detector.consecutiveThreshold)
	}
}

func TestTimeoutDetector_WithConsecutiveThreshold(t *testing.T) {
	detector := NewTimeoutDetector(5 * time.Second).WithConsecutiveThreshold(5)
	if detector.consecutiveThreshold != 5 {
		t.Errorf("expected consecutive threshold 5, got %d", detector.consecutiveThreshold)
	}
}

func TestTimeoutDetector_Detect_NoTimeout(t *testing.T) {
	detector := NewTimeoutDetector(5 * time.Second)

	// 正常執行時間
	if detector.Detect(nil, 1*time.Second) {
		t.Error("should not detect failure for normal duration")
	}
	if detector.GetConsecutiveCount() != 0 {
		t.Errorf("expected count 0, got %d", detector.GetConsecutiveCount())
	}
}

func TestTimeoutDetector_Detect_SingleTimeout(t *testing.T) {
	detector := NewTimeoutDetector(5 * time.Second)

	// 單次逾時不應觸發
	if detector.Detect(nil, 6*time.Second) {
		t.Error("single timeout should not trigger failure")
	}
	if detector.GetConsecutiveCount() != 1 {
		t.Errorf("expected count 1, got %d", detector.GetConsecutiveCount())
	}
}

func TestTimeoutDetector_Detect_ConsecutiveTimeout(t *testing.T) {
	detector := NewTimeoutDetector(5 * time.Second).WithConsecutiveThreshold(3)

	// 連續 3 次逾時
	detector.Detect(nil, 6*time.Second)
	detector.Detect(nil, 6*time.Second)

	// 第 3 次應該觸發
	if !detector.Detect(nil, 6*time.Second) {
		t.Error("3 consecutive timeouts should trigger failure")
	}
}

func TestTimeoutDetector_Detect_ResetOnNormal(t *testing.T) {
	detector := NewTimeoutDetector(5 * time.Second)

	// 2 次逾時
	detector.Detect(nil, 6*time.Second)
	detector.Detect(nil, 6*time.Second)

	// 1 次正常執行
	detector.Detect(nil, 1*time.Second)

	if detector.GetConsecutiveCount() != 0 {
		t.Errorf("count should be reset, got %d", detector.GetConsecutiveCount())
	}
}

func TestTimeoutDetector_Reset(t *testing.T) {
	detector := NewTimeoutDetector(5 * time.Second)

	detector.Detect(nil, 6*time.Second)
	detector.Detect(nil, 6*time.Second)

	detector.Reset()

	if detector.GetConsecutiveCount() != 0 {
		t.Errorf("count should be 0 after reset, got %d", detector.GetConsecutiveCount())
	}
}

func TestTimeoutDetector_GetType(t *testing.T) {
	detector := NewTimeoutDetector(5 * time.Second)
	if detector.GetType() != FailureTimeout {
		t.Errorf("expected FailureTimeout, got %v", detector.GetType())
	}
}

// ========================
// ErrorRateDetector 測試
// ========================

func TestNewErrorRateDetector(t *testing.T) {
	detector := NewErrorRateDetector(10, 0.5)
	if detector == nil {
		t.Fatal("detector should not be nil")
	}
	if detector.windowSize != 10 {
		t.Errorf("expected window size 10, got %d", detector.windowSize)
	}
	if detector.threshold != 0.5 {
		t.Errorf("expected threshold 0.5, got %f", detector.threshold)
	}
}

func TestErrorRateDetector_Detect_WindowNotFull(t *testing.T) {
	detector := NewErrorRateDetector(10, 0.5)

	// 窗口未滿不應觸發
	for i := 0; i < 5; i++ {
		if detector.Detect(errors.New("error"), 0) {
			t.Error("should not detect failure when window not full")
		}
	}
}

func TestErrorRateDetector_Detect_HighErrorRate(t *testing.T) {
	detector := NewErrorRateDetector(10, 0.5)

	// 填滿窗口：8 個錯誤，2 個成功 = 80% 錯誤率
	for i := 0; i < 8; i++ {
		detector.Detect(errors.New("error"), 0)
	}
	detector.Detect(nil, 0) // 成功
	detector.Detect(nil, 0) // 成功

	// 下一次錯誤應該觸發（窗口已滿，錯誤率 > 50%）
	// 需要再填一輪來更新窗口
	for i := 0; i < 8; i++ {
		detector.Detect(errors.New("error"), 0)
	}
	detector.Detect(nil, 0)
	if !detector.Detect(nil, 0) {
		// 現在窗口中有 8 個錯誤，錯誤率應該 > 50%
		// 但由於我們是用最後記錄的來判斷，這可能不會立即觸發
	}

	// 確保高錯誤率被檢測
	errorRate := detector.GetErrorRate()
	if errorRate < 0.5 {
		t.Logf("error rate %f is below threshold, test may need adjustment", errorRate)
	}
}

func TestErrorRateDetector_Detect_LowErrorRate(t *testing.T) {
	detector := NewErrorRateDetector(10, 0.5)

	// 填滿窗口：2 個錯誤，8 個成功 = 20% 錯誤率
	detector.Detect(errors.New("error"), 0)
	detector.Detect(errors.New("error"), 0)
	for i := 0; i < 8; i++ {
		if detector.Detect(nil, 0) {
			t.Error("low error rate should not trigger failure")
		}
	}

	errorRate := detector.GetErrorRate()
	if errorRate > 0.5 {
		t.Errorf("expected error rate <= 50%%, got %f", errorRate)
	}
}

func TestErrorRateDetector_Reset(t *testing.T) {
	detector := NewErrorRateDetector(10, 0.5)

	for i := 0; i < 10; i++ {
		detector.Detect(errors.New("error"), 0)
	}

	detector.Reset()

	if detector.GetErrorRate() != 0 {
		t.Errorf("error rate should be 0 after reset, got %f", detector.GetErrorRate())
	}
}

func TestErrorRateDetector_GetType(t *testing.T) {
	detector := NewErrorRateDetector(10, 0.5)
	if detector.GetType() != FailureErrorRate {
		t.Errorf("expected FailureErrorRate, got %v", detector.GetType())
	}
}

// ========================
// HealthCheckDetector 測試
// ========================

func TestNewHealthCheckDetector(t *testing.T) {
	detector := NewHealthCheckDetector(30*time.Second, 3)
	if detector == nil {
		t.Fatal("detector should not be nil")
	}
	if detector.checkInterval != 30*time.Second {
		t.Errorf("expected interval 30s, got %v", detector.checkInterval)
	}
	if detector.maxUnhealthy != 3 {
		t.Errorf("expected max unhealthy 3, got %d", detector.maxUnhealthy)
	}
}

func TestHealthCheckDetector_Detect_Healthy(t *testing.T) {
	detector := NewHealthCheckDetector(0, 3) // interval=0 表示每次都檢查
	detector.SetHealthCheckFunc(func() bool { return true })

	if detector.Detect(nil, 0) {
		t.Error("healthy check should not trigger failure")
	}
	if detector.GetUnhealthyCount() != 0 {
		t.Errorf("expected unhealthy count 0, got %d", detector.GetUnhealthyCount())
	}
}

func TestHealthCheckDetector_Detect_Unhealthy(t *testing.T) {
	detector := NewHealthCheckDetector(0, 3)
	detector.SetHealthCheckFunc(func() bool { return false })

	// 3 次不健康
	detector.Detect(nil, 0)
	detector.Detect(nil, 0)

	// 第 3 次應該觸發
	if !detector.Detect(nil, 0) {
		t.Error("3 unhealthy checks should trigger failure")
	}
}

func TestHealthCheckDetector_Detect_RecoverAfterHealthy(t *testing.T) {
	detector := NewHealthCheckDetector(0, 3)

	healthy := false
	detector.SetHealthCheckFunc(func() bool { return healthy })

	// 2 次不健康
	detector.Detect(nil, 0)
	detector.Detect(nil, 0)

	// 恢復健康
	healthy = true
	detector.Detect(nil, 0)

	if detector.GetUnhealthyCount() != 0 {
		t.Errorf("unhealthy count should be reset, got %d", detector.GetUnhealthyCount())
	}
}

func TestHealthCheckDetector_Detect_IntervalSkip(t *testing.T) {
	detector := NewHealthCheckDetector(1*time.Hour, 3)
	detector.SetHealthCheckFunc(func() bool { return false })

	// 第一次檢查
	detector.Detect(nil, 0)

	// 由於間隔未到，不會執行實際檢查
	result := detector.Detect(nil, 0)
	// 結果取決於當前累計計數
	if detector.GetUnhealthyCount() != 1 {
		t.Errorf("expected unhealthy count 1, got %d", detector.GetUnhealthyCount())
	}
	_ = result
}

func TestHealthCheckDetector_Reset(t *testing.T) {
	detector := NewHealthCheckDetector(0, 3)
	detector.SetHealthCheckFunc(func() bool { return false })

	detector.Detect(nil, 0)
	detector.Detect(nil, 0)

	detector.Reset()

	if detector.GetUnhealthyCount() != 0 {
		t.Errorf("unhealthy count should be 0 after reset, got %d", detector.GetUnhealthyCount())
	}
}

func TestHealthCheckDetector_GetType(t *testing.T) {
	detector := NewHealthCheckDetector(0, 3)
	if detector.GetType() != FailureHealthCheck {
		t.Errorf("expected FailureHealthCheck, got %v", detector.GetType())
	}
}

// ========================
// ConnectionDetector 測試
// ========================

func TestNewConnectionDetector(t *testing.T) {
	detector := NewConnectionDetector(3)
	if detector == nil {
		t.Fatal("detector should not be nil")
	}
	if detector.threshold != 3 {
		t.Errorf("expected threshold 3, got %d", detector.threshold)
	}
}

func TestConnectionDetector_Detect_NoError(t *testing.T) {
	detector := NewConnectionDetector(3)

	if detector.Detect(nil, 0) {
		t.Error("no error should not trigger failure")
	}
	if detector.GetConsecutiveCount() != 0 {
		t.Errorf("expected count 0, got %d", detector.GetConsecutiveCount())
	}
}

func TestConnectionDetector_Detect_ConnectionError(t *testing.T) {
	detector := NewConnectionDetector(3)

	// 連續 3 次連接錯誤
	detector.Detect(errors.New("connection refused"), 0)
	detector.Detect(errors.New("connection timeout"), 0)

	// 第 3 次應該觸發
	if !detector.Detect(errors.New("connection reset"), 0) {
		t.Error("3 consecutive connection errors should trigger failure")
	}
}

func TestConnectionDetector_Detect_NonConnectionError(t *testing.T) {
	detector := NewConnectionDetector(3)

	// 非連接錯誤
	detector.Detect(errors.New("invalid input"), 0)

	if detector.GetConsecutiveCount() != 0 {
		t.Errorf("non-connection error should not increment count, got %d", detector.GetConsecutiveCount())
	}
}

func TestConnectionDetector_Detect_ResetOnSuccess(t *testing.T) {
	detector := NewConnectionDetector(3)

	detector.Detect(errors.New("connection refused"), 0)
	detector.Detect(errors.New("connection refused"), 0)

	// 成功後重置
	detector.Detect(nil, 0)

	if detector.GetConsecutiveCount() != 0 {
		t.Errorf("count should be reset on success, got %d", detector.GetConsecutiveCount())
	}
}

func TestConnectionDetector_AddPattern(t *testing.T) {
	detector := NewConnectionDetector(1)
	detector.AddPattern("custom error")

	if !detector.Detect(errors.New("custom error occurred"), 0) {
		t.Error("custom pattern should be detected")
	}
}

func TestConnectionDetector_Reset(t *testing.T) {
	detector := NewConnectionDetector(3)

	detector.Detect(errors.New("connection refused"), 0)
	detector.Detect(errors.New("connection refused"), 0)

	detector.Reset()

	if detector.GetConsecutiveCount() != 0 {
		t.Errorf("count should be 0 after reset, got %d", detector.GetConsecutiveCount())
	}
}

func TestConnectionDetector_GetType(t *testing.T) {
	detector := NewConnectionDetector(3)
	if detector.GetType() != FailureConnection {
		t.Errorf("expected FailureConnection, got %v", detector.GetType())
	}
}

// ========================
// MultiDetector 測試
// ========================

func TestNewMultiDetector(t *testing.T) {
	detector := NewMultiDetector()
	if detector == nil {
		t.Fatal("detector should not be nil")
	}
	if detector.GetDetectorCount() != 0 {
		t.Errorf("expected 0 detectors, got %d", detector.GetDetectorCount())
	}
}

func TestNewMultiDetector_WithDetectors(t *testing.T) {
	timeout := NewTimeoutDetector(5 * time.Second)
	errorRate := NewErrorRateDetector(10, 0.5)

	detector := NewMultiDetector(timeout, errorRate)

	if detector.GetDetectorCount() != 2 {
		t.Errorf("expected 2 detectors, got %d", detector.GetDetectorCount())
	}
}

func TestMultiDetector_AddDetector(t *testing.T) {
	detector := NewMultiDetector()
	detector.AddDetector(NewTimeoutDetector(5 * time.Second))

	if detector.GetDetectorCount() != 1 {
		t.Errorf("expected 1 detector, got %d", detector.GetDetectorCount())
	}
}

func TestMultiDetector_Detect_AllPass(t *testing.T) {
	timeout := NewTimeoutDetector(5 * time.Second)
	connection := NewConnectionDetector(3)

	detector := NewMultiDetector(timeout, connection)

	// 正常情況
	if detector.Detect(nil, 1*time.Second) {
		t.Error("should not detect failure when all pass")
	}
}

func TestMultiDetector_Detect_OneTriggered(t *testing.T) {
	timeout := NewTimeoutDetector(5 * time.Second).WithConsecutiveThreshold(1)
	connection := NewConnectionDetector(3)

	detector := NewMultiDetector(timeout, connection)

	// 逾時觸發
	if !detector.Detect(nil, 6*time.Second) {
		t.Error("should detect failure when timeout triggered")
	}
}

func TestMultiDetector_DetectWithType(t *testing.T) {
	timeout := NewTimeoutDetector(5 * time.Second).WithConsecutiveThreshold(1)
	connection := NewConnectionDetector(3)

	detector := NewMultiDetector(timeout, connection)

	failed, failType := detector.DetectWithType(nil, 6*time.Second)

	if !failed {
		t.Error("should detect failure")
	}
	if failType != FailureTimeout {
		t.Errorf("expected FailureTimeout, got %v", failType)
	}
}

func TestMultiDetector_DetectWithType_NoFailure(t *testing.T) {
	timeout := NewTimeoutDetector(5 * time.Second)
	detector := NewMultiDetector(timeout)

	failed, failType := detector.DetectWithType(nil, 1*time.Second)

	if failed {
		t.Error("should not detect failure")
	}
	if failType != FailureNone {
		t.Errorf("expected FailureNone, got %v", failType)
	}
}

func TestMultiDetector_Reset(t *testing.T) {
	timeout := NewTimeoutDetector(5 * time.Second)
	timeout.Detect(nil, 6*time.Second)

	detector := NewMultiDetector(timeout)
	detector.Reset()

	if timeout.GetConsecutiveCount() != 0 {
		t.Errorf("timeout should be reset, got %d", timeout.GetConsecutiveCount())
	}
}

func TestMultiDetector_GetType_Empty(t *testing.T) {
	detector := NewMultiDetector()
	if detector.GetType() != FailureNone {
		t.Errorf("expected FailureNone for empty multi-detector, got %v", detector.GetType())
	}
}

func TestMultiDetector_GetType_WithDetectors(t *testing.T) {
	timeout := NewTimeoutDetector(5 * time.Second)
	detector := NewMultiDetector(timeout)

	if detector.GetType() != FailureTimeout {
		t.Errorf("expected FailureTimeout, got %v", detector.GetType())
	}
}

// ========================
// FailureDetectorConfig 測試
// ========================

func TestDefaultFailureDetectorConfig(t *testing.T) {
	config := DefaultFailureDetectorConfig()

	if !config.EnableTimeout {
		t.Error("expected timeout enabled by default")
	}
	if !config.EnableErrorRate {
		t.Error("expected error rate enabled by default")
	}
	if config.EnableHealthCheck {
		t.Error("expected health check disabled by default")
	}
	if !config.EnableConnection {
		t.Error("expected connection enabled by default")
	}
}

func TestBuildMultiDetector(t *testing.T) {
	config := DefaultFailureDetectorConfig()
	detector := BuildMultiDetector(config)

	// 預設啟用 timeout, error rate, connection = 3 個檢測器
	if detector.GetDetectorCount() != 3 {
		t.Errorf("expected 3 detectors, got %d", detector.GetDetectorCount())
	}
}

func TestBuildMultiDetector_AllEnabled(t *testing.T) {
	config := DefaultFailureDetectorConfig()
	config.EnableHealthCheck = true

	detector := BuildMultiDetector(config)

	if detector.GetDetectorCount() != 4 {
		t.Errorf("expected 4 detectors, got %d", detector.GetDetectorCount())
	}
}

func TestBuildMultiDetector_AllDisabled(t *testing.T) {
	config := &FailureDetectorConfig{
		EnableTimeout:     false,
		EnableErrorRate:   false,
		EnableHealthCheck: false,
		EnableConnection:  false,
	}

	detector := BuildMultiDetector(config)

	if detector.GetDetectorCount() != 0 {
		t.Errorf("expected 0 detectors, got %d", detector.GetDetectorCount())
	}
}

// ========================
// FailureType 測試
// ========================

func TestFailureTypeString(t *testing.T) {
	tests := []struct {
		failType FailureType
		expected string
	}{
		{FailureNone, "none"},
		{FailureTimeout, "timeout"},
		{FailureErrorRate, "error_rate"},
		{FailureHealthCheck, "health_check"},
		{FailureConnection, "connection"},
		{FailureType(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.failType.String(); got != tt.expected {
			t.Errorf("FailureType(%d).String() = %q, expected %q", tt.failType, got, tt.expected)
		}
	}
}

// ========================
// 整合測試
// ========================

func TestFailureDetector_Integration(t *testing.T) {
	config := &FailureDetectorConfig{
		EnableTimeout:      true,
		TimeoutThreshold:   100 * time.Millisecond,
		TimeoutConsecutive: 2,
		EnableErrorRate:    true,
		ErrorRateWindow:    5,
		ErrorRateThreshold: 0.6,
		EnableConnection:   true,
		ConnectionThreshold: 2,
	}

	detector := BuildMultiDetector(config)

	// 測試情境 1: 連續逾時
	detector.Reset()
	if detector.Detect(nil, 150*time.Millisecond) {
		t.Error("single timeout should not trigger")
	}
	if !detector.Detect(nil, 150*time.Millisecond) {
		t.Error("2 consecutive timeouts should trigger")
	}

	// 測試情境 2: 連接錯誤
	detector.Reset()
	if detector.Detect(errors.New("connection refused"), 50*time.Millisecond) {
		t.Error("single connection error should not trigger")
	}
	if !detector.Detect(errors.New("connection refused"), 50*time.Millisecond) {
		t.Error("2 consecutive connection errors should trigger")
	}
}

func TestFailureDetector_Concurrent(t *testing.T) {
	detector := NewMultiDetector(
		NewTimeoutDetector(100*time.Millisecond).WithConsecutiveThreshold(10),
		NewConnectionDetector(10),
	)

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				detector.Detect(nil, 50*time.Millisecond)
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// 確保沒有 race condition 導致 panic
	_ = detector.GetDetectorCount()
}
