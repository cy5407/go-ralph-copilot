package ghcopilot

import (
	"context"
	"errors"
	"testing"
	"time"
)

// ========================
// AutoReconnectRecovery 測試
// ========================

func TestNewAutoReconnectRecovery(t *testing.T) {
	recovery := NewAutoReconnectRecovery(3)
	if recovery == nil {
		t.Fatal("recovery should not be nil")
	}
	if recovery.maxRetries != 3 {
		t.Errorf("expected maxRetries 3, got %d", recovery.maxRetries)
	}
}

func TestAutoReconnectRecovery_Success(t *testing.T) {
	recovery := NewAutoReconnectRecovery(3)
	recovery.SetConnectFunc(func(ctx context.Context) error {
		return nil
	})

	ctx := context.Background()
	err := recovery.Recover(ctx, errors.New("original error"))

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestAutoReconnectRecovery_SuccessAfterRetry(t *testing.T) {
	recovery := NewAutoReconnectRecovery(3)
	recovery.SetRetryDelay(10 * time.Millisecond)

	attempt := 0
	recovery.SetConnectFunc(func(ctx context.Context) error {
		attempt++
		if attempt < 2 {
			return errors.New("temporary error")
		}
		return nil
	})

	ctx := context.Background()
	err := recovery.Recover(ctx, errors.New("original error"))

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if attempt != 2 {
		t.Errorf("expected 2 attempts, got %d", attempt)
	}
}

func TestAutoReconnectRecovery_FailAfterMaxRetries(t *testing.T) {
	recovery := NewAutoReconnectRecovery(3)
	recovery.SetRetryDelay(10 * time.Millisecond)
	recovery.SetConnectFunc(func(ctx context.Context) error {
		return errors.New("persistent error")
	})

	ctx := context.Background()
	err := recovery.Recover(ctx, errors.New("original error"))

	if err == nil {
		t.Error("expected error after max retries")
	}
}

func TestAutoReconnectRecovery_ContextCancellation(t *testing.T) {
	recovery := NewAutoReconnectRecovery(10)
	recovery.SetRetryDelay(100 * time.Millisecond)
	recovery.SetConnectFunc(func(ctx context.Context) error {
		return errors.New("error")
	})

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	err := recovery.Recover(ctx, errors.New("original error"))

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context deadline exceeded, got %v", err)
	}
}

func TestAutoReconnectRecovery_GetType(t *testing.T) {
	recovery := NewAutoReconnectRecovery(3)
	if recovery.GetType() != RecoveryAutoReconnect {
		t.Errorf("expected RecoveryAutoReconnect, got %v", recovery.GetType())
	}
}

func TestAutoReconnectRecovery_GetPriority(t *testing.T) {
	recovery := NewAutoReconnectRecovery(3)
	if recovery.GetPriority() != 1 {
		t.Errorf("expected priority 1, got %d", recovery.GetPriority())
	}
}

// ========================
// SessionRestoreRecovery 測試
// ========================

func TestNewSessionRestoreRecovery(t *testing.T) {
	recovery := NewSessionRestoreRecovery()
	if recovery == nil {
		t.Fatal("recovery should not be nil")
	}
}

func TestSessionRestoreRecovery_NoSessionID(t *testing.T) {
	recovery := NewSessionRestoreRecovery()

	ctx := context.Background()
	err := recovery.Recover(ctx, errors.New("original error"))

	if err == nil {
		t.Error("expected error when no session ID")
	}
}

func TestSessionRestoreRecovery_Success(t *testing.T) {
	recovery := NewSessionRestoreRecovery()
	recovery.SetSessionID("test-session-123")
	recovery.SetRestoreFunc(func(ctx context.Context, sessionID string) error {
		if sessionID != "test-session-123" {
			return errors.New("wrong session ID")
		}
		return nil
	})

	ctx := context.Background()
	err := recovery.Recover(ctx, errors.New("original error"))

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestSessionRestoreRecovery_Failure(t *testing.T) {
	recovery := NewSessionRestoreRecovery()
	recovery.SetSessionID("test-session")
	recovery.SetRestoreFunc(func(ctx context.Context, sessionID string) error {
		return errors.New("restore failed")
	})

	ctx := context.Background()
	err := recovery.Recover(ctx, errors.New("original error"))

	if err == nil {
		t.Error("expected error when restore fails")
	}
}

func TestSessionRestoreRecovery_ContextCancellation(t *testing.T) {
	recovery := NewSessionRestoreRecovery()
	recovery.SetSessionID("test-session")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	err := recovery.Recover(ctx, errors.New("original error"))

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context canceled, got %v", err)
	}
}

func TestSessionRestoreRecovery_GetType(t *testing.T) {
	recovery := NewSessionRestoreRecovery()
	if recovery.GetType() != RecoverySessionRestore {
		t.Errorf("expected RecoverySessionRestore, got %v", recovery.GetType())
	}
}

func TestSessionRestoreRecovery_GetPriority(t *testing.T) {
	recovery := NewSessionRestoreRecovery()
	if recovery.GetPriority() != 2 {
		t.Errorf("expected priority 2, got %d", recovery.GetPriority())
	}
}

// ========================
// FallbackRecovery 測試
// ========================

func TestNewFallbackRecovery(t *testing.T) {
	recovery := NewFallbackRecovery()
	if recovery == nil {
		t.Fatal("recovery should not be nil")
	}
}

func TestFallbackRecovery_Success(t *testing.T) {
	recovery := NewFallbackRecovery()
	recovery.SetFallbackFunc(func(ctx context.Context) (interface{}, error) {
		return "fallback result", nil
	})

	ctx := context.Background()
	err := recovery.Recover(ctx, errors.New("original error"))

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	result := recovery.GetLastResult()
	if result != "fallback result" {
		t.Errorf("expected 'fallback result', got %v", result)
	}
}

func TestFallbackRecovery_Failure(t *testing.T) {
	recovery := NewFallbackRecovery()
	recovery.SetFallbackFunc(func(ctx context.Context) (interface{}, error) {
		return nil, errors.New("fallback failed")
	})

	ctx := context.Background()
	err := recovery.Recover(ctx, errors.New("original error"))

	if err == nil {
		t.Error("expected error when fallback fails")
	}
}

func TestFallbackRecovery_NoFallbackConfigured(t *testing.T) {
	recovery := NewFallbackRecovery()

	ctx := context.Background()
	err := recovery.Recover(ctx, errors.New("original error"))

	if err == nil {
		t.Error("expected error when no fallback configured")
	}
}

func TestFallbackRecovery_ContextCancellation(t *testing.T) {
	recovery := NewFallbackRecovery()
	recovery.SetFallbackFunc(func(ctx context.Context) (interface{}, error) {
		return "result", nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := recovery.Recover(ctx, errors.New("original error"))

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context canceled, got %v", err)
	}
}

func TestFallbackRecovery_GetType(t *testing.T) {
	recovery := NewFallbackRecovery()
	if recovery.GetType() != RecoveryFallback {
		t.Errorf("expected RecoveryFallback, got %v", recovery.GetType())
	}
}

func TestFallbackRecovery_GetPriority(t *testing.T) {
	recovery := NewFallbackRecovery()
	if recovery.GetPriority() != 3 {
		t.Errorf("expected priority 3, got %d", recovery.GetPriority())
	}
}

// ========================
// RecoveryCoordinator 測試
// ========================

func TestNewRecoveryCoordinator(t *testing.T) {
	coordinator := NewRecoveryCoordinator()
	if coordinator == nil {
		t.Fatal("coordinator should not be nil")
	}
	if coordinator.GetStrategyCount() != 0 {
		t.Errorf("expected 0 strategies, got %d", coordinator.GetStrategyCount())
	}
}

func TestRecoveryCoordinator_AddStrategy(t *testing.T) {
	coordinator := NewRecoveryCoordinator()
	coordinator.AddStrategy(NewAutoReconnectRecovery(3))

	if coordinator.GetStrategyCount() != 1 {
		t.Errorf("expected 1 strategy, got %d", coordinator.GetStrategyCount())
	}
}

func TestRecoveryCoordinator_AddStrategy_PrioritySorted(t *testing.T) {
	coordinator := NewRecoveryCoordinator()

	// 先添加低優先級
	coordinator.AddStrategy(NewFallbackRecovery())         // priority 3
	coordinator.AddStrategy(NewSessionRestoreRecovery())   // priority 2
	coordinator.AddStrategy(NewAutoReconnectRecovery(3))   // priority 1

	if coordinator.GetStrategyCount() != 3 {
		t.Errorf("expected 3 strategies, got %d", coordinator.GetStrategyCount())
	}
}

func TestRecoveryCoordinator_Recover_NoStrategies(t *testing.T) {
	coordinator := NewRecoveryCoordinator()

	ctx := context.Background()
	err := coordinator.Recover(ctx, errors.New("original error"))

	if err == nil {
		t.Error("expected error when no strategies configured")
	}
}

func TestRecoveryCoordinator_Recover_FirstSuccess(t *testing.T) {
	coordinator := NewRecoveryCoordinator()

	reconnect := NewAutoReconnectRecovery(3)
	reconnect.SetConnectFunc(func(ctx context.Context) error {
		return nil
	})
	coordinator.AddStrategy(reconnect)

	ctx := context.Background()
	err := coordinator.Recover(ctx, errors.New("original error"))

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	metrics := coordinator.GetMetrics()
	if metrics.SuccessfulRecoveries != 1 {
		t.Errorf("expected 1 successful recovery, got %d", metrics.SuccessfulRecoveries)
	}
}

func TestRecoveryCoordinator_Recover_FallbackToSecond(t *testing.T) {
	coordinator := NewRecoveryCoordinator()

	// 第一個失敗
	reconnect := NewAutoReconnectRecovery(1)
	reconnect.SetRetryDelay(1 * time.Millisecond)
	reconnect.SetConnectFunc(func(ctx context.Context) error {
		return errors.New("reconnect failed")
	})
	coordinator.AddStrategy(reconnect)

	// 第二個成功
	fallback := NewFallbackRecovery()
	fallback.SetFallbackFunc(func(ctx context.Context) (interface{}, error) {
		return "fallback result", nil
	})
	coordinator.AddStrategy(fallback)

	ctx := context.Background()
	err := coordinator.Recover(ctx, errors.New("original error"))

	if err != nil {
		t.Errorf("expected no error (fallback should succeed), got %v", err)
	}

	metrics := coordinator.GetMetrics()
	if metrics.LastRecoveryType != RecoveryFallback {
		t.Errorf("expected RecoveryFallback, got %v", metrics.LastRecoveryType)
	}
}

func TestRecoveryCoordinator_Recover_AllFail(t *testing.T) {
	coordinator := NewRecoveryCoordinator()

	reconnect := NewAutoReconnectRecovery(1)
	reconnect.SetRetryDelay(1 * time.Millisecond)
	reconnect.SetConnectFunc(func(ctx context.Context) error {
		return errors.New("failed")
	})
	coordinator.AddStrategy(reconnect)

	ctx := context.Background()
	err := coordinator.Recover(ctx, errors.New("original error"))

	if err == nil {
		t.Error("expected error when all strategies fail")
	}

	metrics := coordinator.GetMetrics()
	if metrics.FailedRecoveries != 1 {
		t.Errorf("expected 1 failed recovery, got %d", metrics.FailedRecoveries)
	}
}

func TestRecoveryCoordinator_GetMetrics(t *testing.T) {
	coordinator := NewRecoveryCoordinator()

	reconnect := NewAutoReconnectRecovery(1)
	reconnect.SetConnectFunc(func(ctx context.Context) error { return nil })
	coordinator.AddStrategy(reconnect)

	ctx := context.Background()
	_ = coordinator.Recover(ctx, errors.New("error"))

	metrics := coordinator.GetMetrics()
	if metrics.TotalAttempts != 1 {
		t.Errorf("expected 1 total attempt, got %d", metrics.TotalAttempts)
	}
}

func TestRecoveryCoordinator_ResetMetrics(t *testing.T) {
	coordinator := NewRecoveryCoordinator()

	reconnect := NewAutoReconnectRecovery(1)
	reconnect.SetConnectFunc(func(ctx context.Context) error { return nil })
	coordinator.AddStrategy(reconnect)

	ctx := context.Background()
	_ = coordinator.Recover(ctx, errors.New("error"))

	coordinator.ResetMetrics()

	metrics := coordinator.GetMetrics()
	if metrics.TotalAttempts != 0 {
		t.Errorf("expected 0 attempts after reset, got %d", metrics.TotalAttempts)
	}
}

// ========================
// FaultTolerantExecutor 測試
// ========================

func TestNewFaultTolerantExecutor(t *testing.T) {
	executor := NewFaultTolerantExecutor(nil, nil)
	if executor == nil {
		t.Fatal("executor should not be nil")
	}
}

func TestNewFaultTolerantExecutor_WithConfig(t *testing.T) {
	retryPolicy := NewExponentialBackoffPolicy(3)
	detectorConfig := DefaultFailureDetectorConfig()

	executor := NewFaultTolerantExecutor(retryPolicy, detectorConfig)
	if executor == nil {
		t.Fatal("executor should not be nil")
	}
}

func TestFaultTolerantExecutor_Execute_Success(t *testing.T) {
	executor := NewFaultTolerantExecutor(
		NewFixedIntervalPolicy(3, 10*time.Millisecond),
		DefaultFailureDetectorConfig(),
	)

	ctx := context.Background()
	err := executor.Execute(ctx, func() error {
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	metrics := executor.GetMetrics()
	if metrics.SuccessfulExecutions != 1 {
		t.Errorf("expected 1 successful execution, got %d", metrics.SuccessfulExecutions)
	}
}

func TestFaultTolerantExecutor_Execute_SuccessWithRetry(t *testing.T) {
	executor := NewFaultTolerantExecutor(
		NewFixedIntervalPolicy(5, 10*time.Millisecond),
		DefaultFailureDetectorConfig(),
	)

	ctx := context.Background()
	callCount := 0
	err := executor.Execute(ctx, func() error {
		callCount++
		if callCount < 3 {
			return errors.New("temporary error")
		}
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	metrics := executor.GetMetrics()
	if metrics.TotalRetries != 2 {
		t.Errorf("expected 2 retries, got %d", metrics.TotalRetries)
	}
}

func TestFaultTolerantExecutor_Execute_FailureWithRecovery(t *testing.T) {
	executor := NewFaultTolerantExecutor(
		NewFixedIntervalPolicy(2, 10*time.Millisecond),
		DefaultFailureDetectorConfig(),
	)

	// 添加恢復策略
	reconnect := NewAutoReconnectRecovery(1)
	reconnect.SetConnectFunc(func(ctx context.Context) error { return nil })
	executor.AddRecoveryStrategy(reconnect)

	ctx := context.Background()
	callCount := 0
	err := executor.Execute(ctx, func() error {
		callCount++
		if callCount < 2 {
			return errors.New("error")
		}
		return nil
	})

	// 第一次失敗，第二次成功（通過重試機制）
	if err != nil {
		t.Errorf("expected success with retry, got %v", err)
	}
}

func TestFaultTolerantExecutor_GetMetrics(t *testing.T) {
	executor := NewFaultTolerantExecutor(nil, nil)

	ctx := context.Background()
	_ = executor.Execute(ctx, func() error { return nil })
	_ = executor.Execute(ctx, func() error { return nil })

	metrics := executor.GetMetrics()
	if metrics.TotalExecutions != 2 {
		t.Errorf("expected 2 total executions, got %d", metrics.TotalExecutions)
	}
}

func TestFaultTolerantExecutor_GetRetryMetrics(t *testing.T) {
	executor := NewFaultTolerantExecutor(nil, nil)

	ctx := context.Background()
	_ = executor.Execute(ctx, func() error { return nil })

	metrics := executor.GetRetryMetrics()
	if metrics.TotalAttempts != 1 {
		t.Errorf("expected 1 total attempt, got %d", metrics.TotalAttempts)
	}
}

func TestFaultTolerantExecutor_GetRecoveryMetrics(t *testing.T) {
	executor := NewFaultTolerantExecutor(nil, nil)

	metrics := executor.GetRecoveryMetrics()
	if metrics.TotalAttempts != 0 {
		t.Errorf("expected 0 recovery attempts initially, got %d", metrics.TotalAttempts)
	}
}

func TestFaultTolerantExecutor_SetRetryPolicy(t *testing.T) {
	executor := NewFaultTolerantExecutor(nil, nil)

	newPolicy := NewLinearBackoffPolicy(10)
	err := executor.SetRetryPolicy(newPolicy)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestFaultTolerantExecutor_ResetAllMetrics(t *testing.T) {
	executor := NewFaultTolerantExecutor(nil, nil)

	ctx := context.Background()
	_ = executor.Execute(ctx, func() error { return nil })

	executor.ResetAllMetrics()

	metrics := executor.GetMetrics()
	if metrics.TotalExecutions != 0 {
		t.Errorf("expected 0 executions after reset, got %d", metrics.TotalExecutions)
	}
}

func TestFaultTolerantExecutor_ResetDetectors(t *testing.T) {
	executor := NewFaultTolerantExecutor(nil, nil)

	// 這不應該 panic
	executor.ResetDetectors()
}

// ========================
// RecoveryStrategyType 測試
// ========================

func TestRecoveryStrategyTypeString(t *testing.T) {
	tests := []struct {
		strategyType RecoveryStrategyType
		expected     string
	}{
		{RecoveryAutoReconnect, "auto_reconnect"},
		{RecoverySessionRestore, "session_restore"},
		{RecoveryFallback, "fallback"},
		{RecoveryStrategyType(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.strategyType.String(); got != tt.expected {
			t.Errorf("RecoveryStrategyType(%d).String() = %q, expected %q",
				tt.strategyType, got, tt.expected)
		}
	}
}

// ========================
// 整合測試
// ========================

func TestFaultTolerance_Integration(t *testing.T) {
	// 建立容錯執行器
	executor := NewFaultTolerantExecutor(
		NewRetryPolicyBuilder().
			WithMaxAttempts(3).
			WithInitialDelay(10 * time.Millisecond).
			WithStrategy(StrategyExponentialBackoff).
			MustBuild(),
		&FailureDetectorConfig{
			EnableTimeout:       true,
			TimeoutThreshold:    100 * time.Millisecond,
			TimeoutConsecutive:  2,
			EnableConnection:    true,
			ConnectionThreshold: 2,
		},
	)

	// 添加恢復策略
	reconnect := NewAutoReconnectRecovery(2)
	reconnect.SetRetryDelay(10 * time.Millisecond)
	reconnect.SetConnectFunc(func(ctx context.Context) error {
		return nil
	})
	executor.AddRecoveryStrategy(reconnect)

	// 測試場景：暫時故障後成功
	ctx := context.Background()
	callCount := 0
	err := executor.Execute(ctx, func() error {
		callCount++
		if callCount < 3 {
			return errors.New("temporary error")
		}
		return nil
	})

	if err != nil {
		t.Errorf("expected success after retries, got %v", err)
	}

	metrics := executor.GetMetrics()
	if metrics.SuccessfulExecutions != 1 {
		t.Errorf("expected 1 successful execution, got %d", metrics.SuccessfulExecutions)
	}
}

func TestFaultTolerance_Concurrent(t *testing.T) {
	executor := NewFaultTolerantExecutor(
		NewFixedIntervalPolicy(3, 10*time.Millisecond),
		DefaultFailureDetectorConfig(),
	)

	ctx := context.Background()
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			_ = executor.Execute(ctx, func() error {
				return nil
			})
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	metrics := executor.GetMetrics()
	if metrics.TotalExecutions != 10 {
		t.Errorf("expected 10 executions, got %d", metrics.TotalExecutions)
	}
}
