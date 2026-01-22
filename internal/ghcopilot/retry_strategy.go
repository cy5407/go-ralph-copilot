package ghcopilot

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

// RetryStrategyType 定義重試策略類型
type RetryStrategyType int

const (
	// StrategyExponentialBackoff 指數退避策略
	StrategyExponentialBackoff RetryStrategyType = iota
	// StrategyLinearBackoff 線性退避策略
	StrategyLinearBackoff
	// StrategyFixedInterval 固定間隔策略
	StrategyFixedInterval
)

// String 返回策略類型的字串表示
func (s RetryStrategyType) String() string {
	switch s {
	case StrategyExponentialBackoff:
		return "exponential_backoff"
	case StrategyLinearBackoff:
		return "linear_backoff"
	case StrategyFixedInterval:
		return "fixed_interval"
	default:
		return "unknown"
	}
}

// RetryPolicy 定義重試策略配置
type RetryPolicy struct {
	// MaxAttempts 最大重試次數 (包括初始嘗試)
	MaxAttempts int
	// InitialDelay 初始延遲時間
	InitialDelay time.Duration
	// MaxDelay 最大延遲時間
	MaxDelay time.Duration
	// Multiplier 指數退避的乘數 (預設 2.0)
	Multiplier float64
	// Increment 線性退避的增量
	Increment time.Duration
	// Strategy 重試策略類型
	Strategy RetryStrategyType
	// Jitter 是否添加隨機抖動
	Jitter bool
	// JitterFactor 抖動因子 (0.0-1.0)
	JitterFactor float64
	// RetryableErrors 可重試的錯誤類型清單
	RetryableErrors []string
	// NonRetryableErrors 不可重試的錯誤類型清單
	NonRetryableErrors []string
}

// DefaultRetryPolicy 返回預設的重試策略
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxAttempts:  5,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		Increment:    100 * time.Millisecond,
		Strategy:     StrategyExponentialBackoff,
		Jitter:       true,
		JitterFactor: 0.1,
	}
}

// NewExponentialBackoffPolicy 建立指數退避重試策略
func NewExponentialBackoffPolicy(maxAttempts int) *RetryPolicy {
	policy := DefaultRetryPolicy()
	policy.MaxAttempts = maxAttempts
	policy.Strategy = StrategyExponentialBackoff
	return policy
}

// NewLinearBackoffPolicy 建立線性退避重試策略
func NewLinearBackoffPolicy(maxAttempts int) *RetryPolicy {
	policy := DefaultRetryPolicy()
	policy.MaxAttempts = maxAttempts
	policy.Strategy = StrategyLinearBackoff
	policy.MaxDelay = 5 * time.Second
	return policy
}

// NewFixedIntervalPolicy 建立固定間隔重試策略
func NewFixedIntervalPolicy(maxAttempts int, interval time.Duration) *RetryPolicy {
	return &RetryPolicy{
		MaxAttempts:  maxAttempts,
		InitialDelay: interval,
		MaxDelay:     interval,
		Strategy:     StrategyFixedInterval,
		Jitter:       false,
	}
}

// NextWaitDuration 計算下一次重試的等待時間
func (p *RetryPolicy) NextWaitDuration(attempt int) time.Duration {
	if attempt <= 0 {
		attempt = 1
	}

	var delay time.Duration

	switch p.Strategy {
	case StrategyExponentialBackoff:
		// delay = initialDelay * (multiplier ^ (attempt - 1))
		multiplier := p.Multiplier
		if multiplier <= 0 {
			multiplier = 2.0
		}
		delay = time.Duration(float64(p.InitialDelay) * math.Pow(multiplier, float64(attempt-1)))

	case StrategyLinearBackoff:
		// delay = initialDelay + (increment * (attempt - 1))
		delay = p.InitialDelay + p.Increment*time.Duration(attempt-1)

	case StrategyFixedInterval:
		delay = p.InitialDelay

	default:
		delay = p.InitialDelay
	}

	// 應用最大延遲上限
	if delay > p.MaxDelay && p.MaxDelay > 0 {
		delay = p.MaxDelay
	}

	// 添加隨機抖動
	if p.Jitter && p.JitterFactor > 0 {
		jitterRange := float64(delay) * p.JitterFactor
		jitter := time.Duration(rand.Float64() * jitterRange)
		delay += jitter
	}

	return delay
}

// ShouldRetry 判斷是否應該重試
func (p *RetryPolicy) ShouldRetry(attempt int, err error) bool {
	if attempt >= p.MaxAttempts {
		return false
	}
	if err == nil {
		return false
	}

	errMsg := err.Error()

	// 檢查是否在不可重試清單中
	for _, pattern := range p.NonRetryableErrors {
		if containsString(errMsg, pattern) {
			return false
		}
	}

	// 如果有可重試清單，檢查是否匹配
	if len(p.RetryableErrors) > 0 {
		for _, pattern := range p.RetryableErrors {
			if containsString(errMsg, pattern) {
				return true
			}
		}
		return false
	}

	// 預設情況下重試
	return true
}

// Validate 驗證策略配置的有效性
func (p *RetryPolicy) Validate() error {
	if p.MaxAttempts < 1 {
		return fmt.Errorf("max attempts must be at least 1, got %d", p.MaxAttempts)
	}
	if p.InitialDelay < 0 {
		return fmt.Errorf("initial delay cannot be negative")
	}
	if p.MaxDelay < 0 {
		return fmt.Errorf("max delay cannot be negative")
	}
	if p.MaxDelay > 0 && p.InitialDelay > p.MaxDelay {
		return fmt.Errorf("initial delay cannot exceed max delay")
	}
	if p.Multiplier < 0 {
		return fmt.Errorf("multiplier cannot be negative")
	}
	if p.JitterFactor < 0 || p.JitterFactor > 1 {
		return fmt.Errorf("jitter factor must be between 0 and 1")
	}
	return nil
}

// Clone 複製策略配置
func (p *RetryPolicy) Clone() *RetryPolicy {
	return &RetryPolicy{
		MaxAttempts:        p.MaxAttempts,
		InitialDelay:       p.InitialDelay,
		MaxDelay:           p.MaxDelay,
		Multiplier:         p.Multiplier,
		Increment:          p.Increment,
		Strategy:           p.Strategy,
		Jitter:             p.Jitter,
		JitterFactor:       p.JitterFactor,
		RetryableErrors:    append([]string{}, p.RetryableErrors...),
		NonRetryableErrors: append([]string{}, p.NonRetryableErrors...),
	}
}

// RetryExecutor 執行帶重試的操作
type RetryExecutor struct {
	policy  *RetryPolicy
	metrics *RetryMetrics
	mu      sync.RWMutex
}

// RetryMetrics 重試指標統計
type RetryMetrics struct {
	TotalAttempts     int64         // 總嘗試次數
	SuccessfulRetries int64         // 成功重試次數
	FailedRetries     int64         // 失敗重試次數
	TotalWaitTime     time.Duration // 總等待時間
	LastError         error         // 最後一次錯誤
	LastAttemptTime   time.Time     // 最後一次嘗試時間
}

// NewRetryExecutor 建立新的重試執行器
func NewRetryExecutor(policy *RetryPolicy) *RetryExecutor {
	if policy == nil {
		policy = DefaultRetryPolicy()
	}
	return &RetryExecutor{
		policy:  policy,
		metrics: &RetryMetrics{},
	}
}

// Execute 執行帶重試的操作
func (e *RetryExecutor) Execute(ctx context.Context, fn func() error) error {
	return e.ExecuteWithResult(ctx, func() (interface{}, error) {
		return nil, fn()
	}).Error
}

// RetryResult 重試結果
type RetryResult struct {
	Value    interface{}
	Error    error
	Attempts int
	Duration time.Duration
}

// ExecuteWithResult 執行帶重試的操作並返回結果
func (e *RetryExecutor) ExecuteWithResult(ctx context.Context, fn func() (interface{}, error)) *RetryResult {
	startTime := time.Now()
	result := &RetryResult{
		Attempts: 0,
	}

	for attempt := 1; attempt <= e.policy.MaxAttempts; attempt++ {
		// 檢查上下文是否已取消
		select {
		case <-ctx.Done():
			result.Error = ctx.Err()
			result.Duration = time.Since(startTime)
			return result
		default:
		}

		result.Attempts = attempt

		// 執行操作
		value, err := fn()

		e.mu.Lock()
		e.metrics.TotalAttempts++
		e.metrics.LastAttemptTime = time.Now()
		e.mu.Unlock()

		if err == nil {
			// 成功
			result.Value = value
			result.Error = nil
			result.Duration = time.Since(startTime)

			if attempt > 1 {
				e.mu.Lock()
				e.metrics.SuccessfulRetries++
				e.mu.Unlock()
			}

			return result
		}

		e.mu.Lock()
		e.metrics.LastError = err
		e.mu.Unlock()

		// 檢查是否應該重試
		if !e.policy.ShouldRetry(attempt, err) {
			result.Error = fmt.Errorf("operation failed after %d attempts: %w", attempt, err)
			result.Duration = time.Since(startTime)

			e.mu.Lock()
			e.metrics.FailedRetries++
			e.mu.Unlock()

			return result
		}

		// 計算等待時間
		waitDuration := e.policy.NextWaitDuration(attempt)

		e.mu.Lock()
		e.metrics.TotalWaitTime += waitDuration
		e.mu.Unlock()

		// 等待
		select {
		case <-time.After(waitDuration):
		case <-ctx.Done():
			result.Error = ctx.Err()
			result.Duration = time.Since(startTime)
			return result
		}
	}

	// 耗盡所有重試
	result.Error = fmt.Errorf("max attempts (%d) exceeded", e.policy.MaxAttempts)
	result.Duration = time.Since(startTime)

	e.mu.Lock()
	e.metrics.FailedRetries++
	e.mu.Unlock()

	return result
}

// GetMetrics 取得重試指標
func (e *RetryExecutor) GetMetrics() *RetryMetrics {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return &RetryMetrics{
		TotalAttempts:     e.metrics.TotalAttempts,
		SuccessfulRetries: e.metrics.SuccessfulRetries,
		FailedRetries:     e.metrics.FailedRetries,
		TotalWaitTime:     e.metrics.TotalWaitTime,
		LastError:         e.metrics.LastError,
		LastAttemptTime:   e.metrics.LastAttemptTime,
	}
}

// ResetMetrics 重置重試指標
func (e *RetryExecutor) ResetMetrics() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.metrics = &RetryMetrics{}
}

// SetPolicy 設定新的重試策略
func (e *RetryExecutor) SetPolicy(policy *RetryPolicy) error {
	if policy == nil {
		return fmt.Errorf("policy cannot be nil")
	}
	if err := policy.Validate(); err != nil {
		return err
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	e.policy = policy
	return nil
}

// GetPolicy 取得當前重試策略
func (e *RetryExecutor) GetPolicy() *RetryPolicy {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.policy.Clone()
}

// RetryPolicyBuilder 重試策略建構器
type RetryPolicyBuilder struct {
	policy *RetryPolicy
}

// NewRetryPolicyBuilder 建立新的策略建構器
func NewRetryPolicyBuilder() *RetryPolicyBuilder {
	return &RetryPolicyBuilder{
		policy: DefaultRetryPolicy(),
	}
}

// WithMaxAttempts 設定最大重試次數
func (b *RetryPolicyBuilder) WithMaxAttempts(attempts int) *RetryPolicyBuilder {
	b.policy.MaxAttempts = attempts
	return b
}

// WithInitialDelay 設定初始延遲
func (b *RetryPolicyBuilder) WithInitialDelay(delay time.Duration) *RetryPolicyBuilder {
	b.policy.InitialDelay = delay
	return b
}

// WithMaxDelay 設定最大延遲
func (b *RetryPolicyBuilder) WithMaxDelay(delay time.Duration) *RetryPolicyBuilder {
	b.policy.MaxDelay = delay
	return b
}

// WithMultiplier 設定指數退避乘數
func (b *RetryPolicyBuilder) WithMultiplier(multiplier float64) *RetryPolicyBuilder {
	b.policy.Multiplier = multiplier
	return b
}

// WithIncrement 設定線性退避增量
func (b *RetryPolicyBuilder) WithIncrement(increment time.Duration) *RetryPolicyBuilder {
	b.policy.Increment = increment
	return b
}

// WithStrategy 設定重試策略類型
func (b *RetryPolicyBuilder) WithStrategy(strategy RetryStrategyType) *RetryPolicyBuilder {
	b.policy.Strategy = strategy
	return b
}

// WithJitter 設定是否使用隨機抖動
func (b *RetryPolicyBuilder) WithJitter(enabled bool) *RetryPolicyBuilder {
	b.policy.Jitter = enabled
	return b
}

// WithJitterFactor 設定抖動因子
func (b *RetryPolicyBuilder) WithJitterFactor(factor float64) *RetryPolicyBuilder {
	b.policy.JitterFactor = factor
	return b
}

// WithRetryableErrors 設定可重試的錯誤模式
func (b *RetryPolicyBuilder) WithRetryableErrors(patterns ...string) *RetryPolicyBuilder {
	b.policy.RetryableErrors = patterns
	return b
}

// WithNonRetryableErrors 設定不可重試的錯誤模式
func (b *RetryPolicyBuilder) WithNonRetryableErrors(patterns ...string) *RetryPolicyBuilder {
	b.policy.NonRetryableErrors = patterns
	return b
}

// Build 建立重試策略
func (b *RetryPolicyBuilder) Build() (*RetryPolicy, error) {
	if err := b.policy.Validate(); err != nil {
		return nil, err
	}
	return b.policy.Clone(), nil
}

// MustBuild 建立重試策略，若失敗則 panic
func (b *RetryPolicyBuilder) MustBuild() *RetryPolicy {
	policy, err := b.Build()
	if err != nil {
		panic(err)
	}
	return policy
}

// 輔助函式

// containsString 檢查字串是否包含子字串（不區分大小寫）
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && containsIgnoreCase(s, substr)))
}

// containsIgnoreCase 不區分大小寫的字串包含檢查
func containsIgnoreCase(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if equalIgnoreCase(s[i:i+len(substr)], substr) {
			return true
		}
	}
	return false
}

// equalIgnoreCase 不區分大小寫的字串相等檢查
func equalIgnoreCase(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca := a[i]
		cb := b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 32
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 32
		}
		if ca != cb {
			return false
		}
	}
	return true
}
