package ghcopilot

import (
	"context"
	"errors"
	"testing"
	"time"
)

// ========================
// RetryPolicy 測試
// ========================

func TestDefaultRetryPolicy(t *testing.T) {
	policy := DefaultRetryPolicy()

	if policy.MaxAttempts != 5 {
		t.Errorf("expected MaxAttempts 5, got %d", policy.MaxAttempts)
	}
	if policy.Strategy != StrategyExponentialBackoff {
		t.Errorf("expected StrategyExponentialBackoff, got %v", policy.Strategy)
	}
	if policy.InitialDelay != 100*time.Millisecond {
		t.Errorf("expected InitialDelay 100ms, got %v", policy.InitialDelay)
	}
	if policy.MaxDelay != 30*time.Second {
		t.Errorf("expected MaxDelay 30s, got %v", policy.MaxDelay)
	}
	if !policy.Jitter {
		t.Error("expected Jitter to be true")
	}
}

func TestNewExponentialBackoffPolicy(t *testing.T) {
	policy := NewExponentialBackoffPolicy(3)

	if policy.MaxAttempts != 3 {
		t.Errorf("expected MaxAttempts 3, got %d", policy.MaxAttempts)
	}
	if policy.Strategy != StrategyExponentialBackoff {
		t.Errorf("expected StrategyExponentialBackoff, got %v", policy.Strategy)
	}
}

func TestNewLinearBackoffPolicy(t *testing.T) {
	policy := NewLinearBackoffPolicy(4)

	if policy.MaxAttempts != 4 {
		t.Errorf("expected MaxAttempts 4, got %d", policy.MaxAttempts)
	}
	if policy.Strategy != StrategyLinearBackoff {
		t.Errorf("expected StrategyLinearBackoff, got %v", policy.Strategy)
	}
	if policy.MaxDelay != 5*time.Second {
		t.Errorf("expected MaxDelay 5s, got %v", policy.MaxDelay)
	}
}

func TestNewFixedIntervalPolicy(t *testing.T) {
	policy := NewFixedIntervalPolicy(3, 500*time.Millisecond)

	if policy.MaxAttempts != 3 {
		t.Errorf("expected MaxAttempts 3, got %d", policy.MaxAttempts)
	}
	if policy.Strategy != StrategyFixedInterval {
		t.Errorf("expected StrategyFixedInterval, got %v", policy.Strategy)
	}
	if policy.InitialDelay != 500*time.Millisecond {
		t.Errorf("expected InitialDelay 500ms, got %v", policy.InitialDelay)
	}
	if policy.Jitter {
		t.Error("expected Jitter to be false for fixed interval")
	}
}

func TestNextWaitDuration_ExponentialBackoff(t *testing.T) {
	policy := &RetryPolicy{
		Strategy:     StrategyExponentialBackoff,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     10 * time.Second,
		Multiplier:   2.0,
		Jitter:       false, // 禁用抖動以便精確測試
	}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{1, 100 * time.Millisecond},
		{2, 200 * time.Millisecond},
		{3, 400 * time.Millisecond},
		{4, 800 * time.Millisecond},
		{5, 1600 * time.Millisecond},
	}

	for _, tt := range tests {
		delay := policy.NextWaitDuration(tt.attempt)
		if delay != tt.expected {
			t.Errorf("attempt %d: expected %v, got %v", tt.attempt, tt.expected, delay)
		}
	}
}

func TestNextWaitDuration_LinearBackoff(t *testing.T) {
	policy := &RetryPolicy{
		Strategy:     StrategyLinearBackoff,
		InitialDelay: 100 * time.Millisecond,
		Increment:    100 * time.Millisecond,
		MaxDelay:     10 * time.Second,
		Jitter:       false,
	}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{1, 100 * time.Millisecond},
		{2, 200 * time.Millisecond},
		{3, 300 * time.Millisecond},
		{4, 400 * time.Millisecond},
		{5, 500 * time.Millisecond},
	}

	for _, tt := range tests {
		delay := policy.NextWaitDuration(tt.attempt)
		if delay != tt.expected {
			t.Errorf("attempt %d: expected %v, got %v", tt.attempt, tt.expected, delay)
		}
	}
}

func TestNextWaitDuration_FixedInterval(t *testing.T) {
	policy := NewFixedIntervalPolicy(5, 500*time.Millisecond)

	for attempt := 1; attempt <= 5; attempt++ {
		delay := policy.NextWaitDuration(attempt)
		if delay != 500*time.Millisecond {
			t.Errorf("attempt %d: expected 500ms, got %v", attempt, delay)
		}
	}
}

func TestNextWaitDuration_MaxDelayLimit(t *testing.T) {
	policy := &RetryPolicy{
		Strategy:     StrategyExponentialBackoff,
		InitialDelay: 1 * time.Second,
		MaxDelay:     5 * time.Second,
		Multiplier:   10.0,
		Jitter:       false,
	}

	// 第 3 次嘗試: 1s * 10^2 = 100s，應該被限制為 5s
	delay := policy.NextWaitDuration(3)
	if delay != 5*time.Second {
		t.Errorf("expected max delay 5s, got %v", delay)
	}
}

func TestNextWaitDuration_WithJitter(t *testing.T) {
	policy := &RetryPolicy{
		Strategy:     StrategyFixedInterval,
		InitialDelay: 1 * time.Second,
		MaxDelay:     5 * time.Second,
		Jitter:       true,
		JitterFactor: 0.1,
	}

	// 執行多次以確保抖動在合理範圍內
	for i := 0; i < 10; i++ {
		delay := policy.NextWaitDuration(1)
		// 延遲應該在 1s 到 1.1s 之間
		if delay < 1*time.Second || delay > 1100*time.Millisecond {
			t.Errorf("delay %v out of expected range [1s, 1.1s]", delay)
		}
	}
}

func TestNextWaitDuration_ZeroAttempt(t *testing.T) {
	policy := DefaultRetryPolicy()
	policy.Jitter = false

	// attempt <= 0 應該被當作 1
	delay := policy.NextWaitDuration(0)
	if delay != policy.InitialDelay {
		t.Errorf("expected initial delay for attempt 0, got %v", delay)
	}
}

func TestShouldRetry_MaxAttemptsExceeded(t *testing.T) {
	policy := &RetryPolicy{MaxAttempts: 3}
	err := errors.New("test error")

	if policy.ShouldRetry(3, err) {
		t.Error("should not retry when max attempts reached")
	}
	if policy.ShouldRetry(4, err) {
		t.Error("should not retry when exceeding max attempts")
	}
}

func TestShouldRetry_NoError(t *testing.T) {
	policy := &RetryPolicy{MaxAttempts: 5}

	if policy.ShouldRetry(1, nil) {
		t.Error("should not retry when no error")
	}
}

func TestShouldRetry_WithRetryableErrors(t *testing.T) {
	policy := &RetryPolicy{
		MaxAttempts:     5,
		RetryableErrors: []string{"timeout", "connection refused"},
	}

	if !policy.ShouldRetry(1, errors.New("connection timeout")) {
		t.Error("should retry on timeout error")
	}
	if !policy.ShouldRetry(1, errors.New("connection refused by server")) {
		t.Error("should retry on connection refused error")
	}
	if policy.ShouldRetry(1, errors.New("invalid input")) {
		t.Error("should not retry on non-retryable error")
	}
}

func TestShouldRetry_WithNonRetryableErrors(t *testing.T) {
	policy := &RetryPolicy{
		MaxAttempts:        5,
		NonRetryableErrors: []string{"authentication failed", "invalid"},
	}

	if policy.ShouldRetry(1, errors.New("authentication failed")) {
		t.Error("should not retry on authentication error")
	}
	if policy.ShouldRetry(1, errors.New("invalid input")) {
		t.Error("should not retry on invalid error")
	}
	if !policy.ShouldRetry(1, errors.New("temporary error")) {
		t.Error("should retry on other errors")
	}
}

func TestPolicyValidate_Valid(t *testing.T) {
	policy := DefaultRetryPolicy()
	if err := policy.Validate(); err != nil {
		t.Errorf("valid policy should pass validation: %v", err)
	}
}

func TestPolicyValidate_InvalidMaxAttempts(t *testing.T) {
	policy := &RetryPolicy{MaxAttempts: 0}
	if err := policy.Validate(); err == nil {
		t.Error("expected validation error for MaxAttempts < 1")
	}
}

func TestPolicyValidate_NegativeInitialDelay(t *testing.T) {
	policy := &RetryPolicy{MaxAttempts: 3, InitialDelay: -1}
	if err := policy.Validate(); err == nil {
		t.Error("expected validation error for negative InitialDelay")
	}
}

func TestPolicyValidate_NegativeMaxDelay(t *testing.T) {
	policy := &RetryPolicy{MaxAttempts: 3, MaxDelay: -1}
	if err := policy.Validate(); err == nil {
		t.Error("expected validation error for negative MaxDelay")
	}
}

func TestPolicyValidate_InitialExceedsMax(t *testing.T) {
	policy := &RetryPolicy{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Second,
		MaxDelay:     5 * time.Second,
	}
	if err := policy.Validate(); err == nil {
		t.Error("expected validation error when InitialDelay > MaxDelay")
	}
}

func TestPolicyValidate_NegativeMultiplier(t *testing.T) {
	policy := &RetryPolicy{MaxAttempts: 3, Multiplier: -1}
	if err := policy.Validate(); err == nil {
		t.Error("expected validation error for negative Multiplier")
	}
}

func TestPolicyValidate_InvalidJitterFactor(t *testing.T) {
	policy := &RetryPolicy{MaxAttempts: 3, JitterFactor: 1.5}
	if err := policy.Validate(); err == nil {
		t.Error("expected validation error for JitterFactor > 1")
	}

	policy.JitterFactor = -0.1
	if err := policy.Validate(); err == nil {
		t.Error("expected validation error for JitterFactor < 0")
	}
}

func TestPolicyClone(t *testing.T) {
	original := DefaultRetryPolicy()
	original.RetryableErrors = []string{"error1", "error2"}

	clone := original.Clone()

	// 修改克隆不應影響原本
	clone.MaxAttempts = 10
	clone.RetryableErrors[0] = "modified"

	if original.MaxAttempts == 10 {
		t.Error("clone modification should not affect original")
	}
	if original.RetryableErrors[0] == "modified" {
		t.Error("clone slice modification should not affect original")
	}
}

func TestRetryStrategyTypeString(t *testing.T) {
	tests := []struct {
		strategy RetryStrategyType
		expected string
	}{
		{StrategyExponentialBackoff, "exponential_backoff"},
		{StrategyLinearBackoff, "linear_backoff"},
		{StrategyFixedInterval, "fixed_interval"},
		{RetryStrategyType(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.strategy.String(); got != tt.expected {
			t.Errorf("strategy %d: expected %q, got %q", tt.strategy, tt.expected, got)
		}
	}
}

// ========================
// RetryExecutor 測試
// ========================

func TestNewRetryExecutor(t *testing.T) {
	executor := NewRetryExecutor(nil)
	if executor == nil {
		t.Fatal("executor should not be nil")
	}
	if executor.policy == nil {
		t.Error("should use default policy when nil")
	}
}

func TestNewRetryExecutorWithPolicy(t *testing.T) {
	policy := NewExponentialBackoffPolicy(3)
	executor := NewRetryExecutor(policy)

	if executor.policy.MaxAttempts != 3 {
		t.Errorf("expected MaxAttempts 3, got %d", executor.policy.MaxAttempts)
	}
}

func TestRetryExecutor_SuccessOnFirstAttempt(t *testing.T) {
	executor := NewRetryExecutor(NewFixedIntervalPolicy(3, 10*time.Millisecond))
	ctx := context.Background()

	callCount := 0
	err := executor.Execute(ctx, func() error {
		callCount++
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}

	metrics := executor.GetMetrics()
	if metrics.SuccessfulRetries != 0 {
		t.Errorf("expected 0 successful retries, got %d", metrics.SuccessfulRetries)
	}
}

func TestRetryExecutor_SuccessAfterRetry(t *testing.T) {
	executor := NewRetryExecutor(NewFixedIntervalPolicy(5, 10*time.Millisecond))
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
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}

	metrics := executor.GetMetrics()
	if metrics.SuccessfulRetries != 1 {
		t.Errorf("expected 1 successful retry, got %d", metrics.SuccessfulRetries)
	}
}

func TestRetryExecutor_FailAfterMaxAttempts(t *testing.T) {
	executor := NewRetryExecutor(NewFixedIntervalPolicy(3, 10*time.Millisecond))
	ctx := context.Background()

	callCount := 0
	err := executor.Execute(ctx, func() error {
		callCount++
		return errors.New("persistent error")
	})

	if err == nil {
		t.Error("expected error after max attempts")
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}

	metrics := executor.GetMetrics()
	if metrics.FailedRetries != 1 {
		t.Errorf("expected 1 failed retry, got %d", metrics.FailedRetries)
	}
}

func TestRetryExecutor_ContextCancellation(t *testing.T) {
	executor := NewRetryExecutor(NewFixedIntervalPolicy(10, 100*time.Millisecond))
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	callCount := 0
	err := executor.Execute(ctx, func() error {
		callCount++
		return errors.New("error")
	})

	if err == nil {
		t.Error("expected context error")
	}
	// 應該執行了 1-2 次然後超時
	if callCount > 3 {
		t.Errorf("expected less than 3 calls due to timeout, got %d", callCount)
	}
}

func TestRetryExecutor_ExecuteWithResult(t *testing.T) {
	executor := NewRetryExecutor(NewFixedIntervalPolicy(3, 10*time.Millisecond))
	ctx := context.Background()

	result := executor.ExecuteWithResult(ctx, func() (interface{}, error) {
		return "success value", nil
	})

	if result.Error != nil {
		t.Errorf("expected no error, got %v", result.Error)
	}
	if result.Value != "success value" {
		t.Errorf("expected 'success value', got %v", result.Value)
	}
	if result.Attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", result.Attempts)
	}
}

func TestRetryExecutor_GetMetrics(t *testing.T) {
	executor := NewRetryExecutor(NewFixedIntervalPolicy(3, 10*time.Millisecond))
	ctx := context.Background()

	// 執行一些操作
	callCount := 0
	_ = executor.Execute(ctx, func() error {
		callCount++
		if callCount < 2 {
			return errors.New("error")
		}
		return nil
	})

	metrics := executor.GetMetrics()
	if metrics.TotalAttempts != 2 {
		t.Errorf("expected TotalAttempts 2, got %d", metrics.TotalAttempts)
	}
}

func TestRetryExecutor_ResetMetrics(t *testing.T) {
	executor := NewRetryExecutor(NewFixedIntervalPolicy(3, 10*time.Millisecond))
	ctx := context.Background()

	_ = executor.Execute(ctx, func() error { return nil })

	executor.ResetMetrics()
	metrics := executor.GetMetrics()

	if metrics.TotalAttempts != 0 {
		t.Errorf("expected TotalAttempts 0 after reset, got %d", metrics.TotalAttempts)
	}
}

func TestRetryExecutor_SetPolicy(t *testing.T) {
	executor := NewRetryExecutor(nil)

	newPolicy := NewLinearBackoffPolicy(10)
	err := executor.SetPolicy(newPolicy)

	if err != nil {
		t.Errorf("expected no error setting policy, got %v", err)
	}

	policy := executor.GetPolicy()
	if policy.MaxAttempts != 10 {
		t.Errorf("expected MaxAttempts 10, got %d", policy.MaxAttempts)
	}
}

func TestRetryExecutor_SetPolicy_Nil(t *testing.T) {
	executor := NewRetryExecutor(nil)
	err := executor.SetPolicy(nil)

	if err == nil {
		t.Error("expected error when setting nil policy")
	}
}

func TestRetryExecutor_SetPolicy_Invalid(t *testing.T) {
	executor := NewRetryExecutor(nil)
	invalidPolicy := &RetryPolicy{MaxAttempts: 0}
	err := executor.SetPolicy(invalidPolicy)

	if err == nil {
		t.Error("expected error when setting invalid policy")
	}
}

// ========================
// RetryPolicyBuilder 測試
// ========================

func TestRetryPolicyBuilder_Basic(t *testing.T) {
	policy, err := NewRetryPolicyBuilder().
		WithMaxAttempts(5).
		WithInitialDelay(200 * time.Millisecond).
		WithStrategy(StrategyLinearBackoff).
		Build()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy.MaxAttempts != 5 {
		t.Errorf("expected MaxAttempts 5, got %d", policy.MaxAttempts)
	}
	if policy.InitialDelay != 200*time.Millisecond {
		t.Errorf("expected InitialDelay 200ms, got %v", policy.InitialDelay)
	}
	if policy.Strategy != StrategyLinearBackoff {
		t.Errorf("expected StrategyLinearBackoff, got %v", policy.Strategy)
	}
}

func TestRetryPolicyBuilder_AllOptions(t *testing.T) {
	policy, err := NewRetryPolicyBuilder().
		WithMaxAttempts(10).
		WithInitialDelay(50 * time.Millisecond).
		WithMaxDelay(5 * time.Second).
		WithMultiplier(3.0).
		WithIncrement(50 * time.Millisecond).
		WithStrategy(StrategyExponentialBackoff).
		WithJitter(true).
		WithJitterFactor(0.2).
		WithRetryableErrors("timeout", "connection").
		WithNonRetryableErrors("auth failed").
		Build()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy.MaxAttempts != 10 {
		t.Errorf("expected MaxAttempts 10, got %d", policy.MaxAttempts)
	}
	if policy.Multiplier != 3.0 {
		t.Errorf("expected Multiplier 3.0, got %f", policy.Multiplier)
	}
	if len(policy.RetryableErrors) != 2 {
		t.Errorf("expected 2 retryable errors, got %d", len(policy.RetryableErrors))
	}
	if len(policy.NonRetryableErrors) != 1 {
		t.Errorf("expected 1 non-retryable error, got %d", len(policy.NonRetryableErrors))
	}
}

func TestRetryPolicyBuilder_InvalidBuild(t *testing.T) {
	_, err := NewRetryPolicyBuilder().
		WithMaxAttempts(0).
		Build()

	if err == nil {
		t.Error("expected error for invalid MaxAttempts")
	}
}

func TestRetryPolicyBuilder_MustBuild_Success(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Error("MustBuild should not panic for valid config")
		}
	}()

	policy := NewRetryPolicyBuilder().
		WithMaxAttempts(3).
		MustBuild()

	if policy.MaxAttempts != 3 {
		t.Errorf("expected MaxAttempts 3, got %d", policy.MaxAttempts)
	}
}

func TestRetryPolicyBuilder_MustBuild_Panic(t *testing.T) {
	// 測試無效配置時，MustBuild 應返回預設策略而不是 panic
	policy := NewRetryPolicyBuilder().
		WithMaxAttempts(0). // 無效配置
		MustBuild()

	// 應該返回預設策略
	if policy == nil {
		t.Error("MustBuild should return default policy for invalid config, not nil")
	}
	
	// 驗證返回的是預設策略
	defaultPolicy := DefaultRetryPolicy()
	if policy.MaxAttempts != defaultPolicy.MaxAttempts {
		t.Errorf("Expected default MaxAttempts %d, got %d", defaultPolicy.MaxAttempts, policy.MaxAttempts)
	}
}

// ========================
// 輔助函式測試
// ========================

func TestContainsString(t *testing.T) {
	tests := []struct {
		s        string
		substr   string
		expected bool
	}{
		{"hello world", "world", true},
		{"hello world", "WORLD", true}, // 不區分大小寫
		{"Hello World", "hello", true},
		{"hello", "hello world", false},
		{"", "", true},
		{"hello", "", true},
		{"", "hello", false},
	}

	for _, tt := range tests {
		result := containsString(tt.s, tt.substr)
		if result != tt.expected {
			t.Errorf("containsString(%q, %q) = %v, expected %v",
				tt.s, tt.substr, result, tt.expected)
		}
	}
}

// ========================
// 整合測試
// ========================

func TestRetryExecutor_Integration_ExponentialBackoff(t *testing.T) {
	policy := NewRetryPolicyBuilder().
		WithMaxAttempts(5).
		WithInitialDelay(10 * time.Millisecond).
		WithMaxDelay(100 * time.Millisecond).
		WithMultiplier(2.0).
		WithStrategy(StrategyExponentialBackoff).
		WithJitter(false).
		MustBuild()

	executor := NewRetryExecutor(policy)
	ctx := context.Background()

	startTime := time.Now()
	callCount := 0

	err := executor.Execute(ctx, func() error {
		callCount++
		if callCount < 4 {
			return errors.New("error")
		}
		return nil
	})

	elapsed := time.Since(startTime)

	if err != nil {
		t.Errorf("expected success, got %v", err)
	}
	if callCount != 4 {
		t.Errorf("expected 4 calls, got %d", callCount)
	}

	// 總等待時間應該約為 10 + 20 + 40 = 70ms
	// 加上一些執行時間餘量
	if elapsed < 60*time.Millisecond || elapsed > 200*time.Millisecond {
		t.Errorf("elapsed time %v out of expected range", elapsed)
	}
}

func TestRetryExecutor_Integration_LinearBackoff(t *testing.T) {
	policy := NewRetryPolicyBuilder().
		WithMaxAttempts(4).
		WithInitialDelay(10 * time.Millisecond).
		WithIncrement(10 * time.Millisecond).
		WithStrategy(StrategyLinearBackoff).
		WithJitter(false).
		MustBuild()

	executor := NewRetryExecutor(policy)
	ctx := context.Background()

	startTime := time.Now()
	callCount := 0

	err := executor.Execute(ctx, func() error {
		callCount++
		if callCount < 3 {
			return errors.New("error")
		}
		return nil
	})

	elapsed := time.Since(startTime)

	if err != nil {
		t.Errorf("expected success, got %v", err)
	}

	// 總等待時間應該約為 10 + 20 = 30ms
	if elapsed < 25*time.Millisecond || elapsed > 100*time.Millisecond {
		t.Errorf("elapsed time %v out of expected range", elapsed)
	}
}

func TestRetryExecutor_Integration_NonRetryableError(t *testing.T) {
	policy := NewRetryPolicyBuilder().
		WithMaxAttempts(5).
		WithInitialDelay(10 * time.Millisecond).
		WithNonRetryableErrors("fatal").
		MustBuild()

	executor := NewRetryExecutor(policy)
	ctx := context.Background()

	callCount := 0
	err := executor.Execute(ctx, func() error {
		callCount++
		return errors.New("fatal error occurred")
	})

	if err == nil {
		t.Error("expected error")
	}
	if callCount != 1 {
		t.Errorf("expected 1 call for non-retryable error, got %d", callCount)
	}
}

func TestRetryExecutor_Concurrent(t *testing.T) {
	policy := NewFixedIntervalPolicy(3, 10*time.Millisecond)
	executor := NewRetryExecutor(policy)
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
	if metrics.TotalAttempts != 10 {
		t.Errorf("expected 10 total attempts, got %d", metrics.TotalAttempts)
	}
}
