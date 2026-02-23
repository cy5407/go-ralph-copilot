package ghcopilot

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RecoveryStrategyType 定義恢復策略類型
type RecoveryStrategyType int

const (
	// RecoveryAutoReconnect 自動重連恢復
	RecoveryAutoReconnect RecoveryStrategyType = iota
	// RecoverySessionRestore 會話恢復
	RecoverySessionRestore
	// RecoveryFallback 故障轉移恢復
	RecoveryFallback
)

// String 返回恢復策略類型的字串表示
func (r RecoveryStrategyType) String() string {
	switch r {
	case RecoveryAutoReconnect:
		return "auto_reconnect"
	case RecoverySessionRestore:
		return "session_restore"
	case RecoveryFallback:
		return "fallback"
	default:
		return "unknown"
	}
}

// RecoveryStrategy 恢復策略介面
type RecoveryStrategy interface {
	// Recover 嘗試恢復
	Recover(ctx context.Context, err error) error
	// GetType 取得策略類型
	GetType() RecoveryStrategyType
	// GetPriority 取得優先級 (數字越小優先級越高)
	GetPriority() int
}

// AutoReconnectRecovery 自動重連恢復策略
type AutoReconnectRecovery struct {
	maxRetries    int
	retryDelay    time.Duration
	connectFunc   func(ctx context.Context) error
	mu            sync.Mutex
}

// NewAutoReconnectRecovery 建立新的自動重連恢復策略
func NewAutoReconnectRecovery(maxRetries int) *AutoReconnectRecovery {
	return &AutoReconnectRecovery{
		maxRetries:  maxRetries,
		retryDelay:  100 * time.Millisecond,
		connectFunc: func(ctx context.Context) error { return nil },
	}
}

// SetConnectFunc 設定連接函式
func (r *AutoReconnectRecovery) SetConnectFunc(fn func(ctx context.Context) error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.connectFunc = fn
}

// SetRetryDelay 設定重試延遲
func (r *AutoReconnectRecovery) SetRetryDelay(delay time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.retryDelay = delay
}

// Recover 嘗試重新連接
func (r *AutoReconnectRecovery) Recover(ctx context.Context, err error) error {
	r.mu.Lock()
	connectFunc := r.connectFunc
	maxRetries := r.maxRetries
	retryDelay := r.retryDelay
	r.mu.Unlock()

	fmt.Printf("🔄 開始恢復策略（最多重試 %d 次）: %v\n", maxRetries, err)

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			fmt.Printf("⚠️ 恢復策略被取消: %v\n", ctx.Err())
			return ctx.Err()
		default:
		}

		fmt.Printf("🔄 恢復嘗試 %d/%d...\n", attempt, maxRetries)
		lastErr = connectFunc(ctx)
		if lastErr == nil {
			fmt.Printf("✅ 恢復成功（嘗試 %d 次）\n", attempt)
			return nil
		}
		fmt.Printf("⚠️ 恢復嘗試 %d 失敗: %v\n", attempt, lastErr)

		// 指數退避
		delay := retryDelay * time.Duration(attempt)
		fmt.Printf("⏳ 等待 %v 後重試...\n", delay)
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			fmt.Printf("⚠️ 恢復策略被取消: %v\n", ctx.Err())
			return ctx.Err()
		}
	}

	finalErr := fmt.Errorf("自動重連失敗（已嘗試 %d 次）: %w", maxRetries, lastErr)
	fmt.Printf("❌ %v\n", finalErr)
	return finalErr
}

// GetType 取得策略類型
func (r *AutoReconnectRecovery) GetType() RecoveryStrategyType {
	return RecoveryAutoReconnect
}

// GetPriority 取得優先級
func (r *AutoReconnectRecovery) GetPriority() int {
	return 1 // 高優先級
}

// SessionRestoreRecovery 會話恢復策略
type SessionRestoreRecovery struct {
	restoreFunc func(ctx context.Context, sessionID string) error
	sessionID   string
	mu          sync.Mutex
}

// NewSessionRestoreRecovery 建立新的會話恢復策略
func NewSessionRestoreRecovery() *SessionRestoreRecovery {
	return &SessionRestoreRecovery{
		restoreFunc: func(ctx context.Context, sessionID string) error { return nil },
	}
}

// SetRestoreFunc 設定恢復函式
func (r *SessionRestoreRecovery) SetRestoreFunc(fn func(ctx context.Context, sessionID string) error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.restoreFunc = fn
}

// SetSessionID 設定要恢復的會話 ID
func (r *SessionRestoreRecovery) SetSessionID(sessionID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessionID = sessionID
}

// Recover 嘗試恢復會話
func (r *SessionRestoreRecovery) Recover(ctx context.Context, err error) error {
	r.mu.Lock()
	restoreFunc := r.restoreFunc
	sessionID := r.sessionID
	r.mu.Unlock()

	if sessionID == "" {
		return fmt.Errorf("恢復會話失敗：未設定會話 ID")
	}

	select {
	case <-ctx.Done():
		fmt.Printf("⚠️ 會話恢復被取消: %v\n", ctx.Err())
		return ctx.Err()
	default:
	}

	fmt.Printf("🔄 嘗試恢復會話: %s\n", sessionID)
	if restoreErr := restoreFunc(ctx, sessionID); restoreErr != nil {
		finalErr := fmt.Errorf("會話恢復失敗: %w", restoreErr)
		fmt.Printf("❌ %v\n", finalErr)
		return finalErr
	}

	fmt.Printf("✅ 會話恢復成功: %s\n", sessionID)
	return nil
}

// GetType 取得策略類型
func (r *SessionRestoreRecovery) GetType() RecoveryStrategyType {
	return RecoverySessionRestore
}

// GetPriority 取得優先級
func (r *SessionRestoreRecovery) GetPriority() int {
	return 2 // 中優先級
}

// FallbackRecovery 故障轉移恢復策略
type FallbackRecovery struct {
	fallbackFunc func(ctx context.Context) (interface{}, error)
	lastResult   interface{}
	mu           sync.Mutex
}

// NewFallbackRecovery 建立新的故障轉移恢復策略
func NewFallbackRecovery() *FallbackRecovery {
	return &FallbackRecovery{
		fallbackFunc: func(ctx context.Context) (interface{}, error) {
			return nil, fmt.Errorf("no fallback configured")
		},
	}
}

// SetFallbackFunc 設定故障轉移函式
func (r *FallbackRecovery) SetFallbackFunc(fn func(ctx context.Context) (interface{}, error)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.fallbackFunc = fn
}

// Recover 執行故障轉移
func (r *FallbackRecovery) Recover(ctx context.Context, err error) error {
	r.mu.Lock()
	fallbackFunc := r.fallbackFunc
	r.mu.Unlock()

	select {
	case <-ctx.Done():
		fmt.Printf("⚠️ 故障轉移被取消: %v\n", ctx.Err())
		return ctx.Err()
	default:
	}

	fmt.Printf("🔄 執行故障轉移策略: %v\n", err)
	result, fallbackErr := fallbackFunc(ctx)
	if fallbackErr != nil {
		finalErr := fmt.Errorf("故障轉移失敗: %w", fallbackErr)
		fmt.Printf("❌ %v\n", finalErr)
		return finalErr
	}

	r.mu.Lock()
	r.lastResult = result
	r.mu.Unlock()

	fmt.Printf("✅ 故障轉移成功\n")
	return nil
}

// GetLastResult 取得最後一次故障轉移的結果
func (r *FallbackRecovery) GetLastResult() interface{} {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.lastResult
}

// GetType 取得策略類型
func (r *FallbackRecovery) GetType() RecoveryStrategyType {
	return RecoveryFallback
}

// GetPriority 取得優先級
func (r *FallbackRecovery) GetPriority() int {
	return 3 // 低優先級
}

// RecoveryCoordinator 恢復協調器
type RecoveryCoordinator struct {
	strategies []RecoveryStrategy
	metrics    *RecoveryMetrics
	mu         sync.RWMutex
}

// RecoveryMetrics 恢復指標統計
type RecoveryMetrics struct {
	TotalAttempts        int64
	SuccessfulRecoveries int64
	FailedRecoveries     int64
	LastRecoveryTime     time.Time
	LastRecoveryType     RecoveryStrategyType
	LastError            error
	mu                   sync.RWMutex
}

// NewRecoveryCoordinator 建立新的恢復協調器
func NewRecoveryCoordinator() *RecoveryCoordinator {
	return &RecoveryCoordinator{
		strategies: make([]RecoveryStrategy, 0),
		metrics:    &RecoveryMetrics{},
	}
}

// AddStrategy 添加恢復策略
func (c *RecoveryCoordinator) AddStrategy(strategy RecoveryStrategy) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.strategies = append(c.strategies, strategy)

	// 按優先級排序
	for i := len(c.strategies) - 1; i > 0; i-- {
		if c.strategies[i].GetPriority() < c.strategies[i-1].GetPriority() {
			c.strategies[i], c.strategies[i-1] = c.strategies[i-1], c.strategies[i]
		}
	}
}

// Recover 嘗試恢復，按優先級依次嘗試各策略
func (c *RecoveryCoordinator) Recover(ctx context.Context, originalErr error) error {
	c.mu.RLock()
	strategies := make([]RecoveryStrategy, len(c.strategies))
	copy(strategies, c.strategies)
	c.mu.RUnlock()

	if len(strategies) == 0 {
		return fmt.Errorf("no recovery strategies configured")
	}

	c.metrics.mu.Lock()
	c.metrics.TotalAttempts++
	c.metrics.mu.Unlock()

	var lastErr error
	for _, strategy := range strategies {
		select {
		case <-ctx.Done():
			c.recordFailure(ctx.Err())
			return ctx.Err()
		default:
		}

		err := strategy.Recover(ctx, originalErr)
		if err == nil {
			c.recordSuccess(strategy.GetType())
			return nil
		}
		lastErr = err
	}

	c.recordFailure(lastErr)
	return fmt.Errorf("all recovery strategies failed: %w", lastErr)
}

// recordSuccess 記錄成功恢復
func (c *RecoveryCoordinator) recordSuccess(recoveryType RecoveryStrategyType) {
	c.metrics.mu.Lock()
	defer c.metrics.mu.Unlock()
	c.metrics.SuccessfulRecoveries++
	c.metrics.LastRecoveryTime = time.Now()
	c.metrics.LastRecoveryType = recoveryType
	c.metrics.LastError = nil
}

// recordFailure 記錄失敗恢復
func (c *RecoveryCoordinator) recordFailure(err error) {
	c.metrics.mu.Lock()
	defer c.metrics.mu.Unlock()
	c.metrics.FailedRecoveries++
	c.metrics.LastError = err
}

// GetMetrics 取得恢復指標
func (c *RecoveryCoordinator) GetMetrics() *RecoveryMetrics {
	c.metrics.mu.RLock()
	defer c.metrics.mu.RUnlock()

	return &RecoveryMetrics{
		TotalAttempts:        c.metrics.TotalAttempts,
		SuccessfulRecoveries: c.metrics.SuccessfulRecoveries,
		FailedRecoveries:     c.metrics.FailedRecoveries,
		LastRecoveryTime:     c.metrics.LastRecoveryTime,
		LastRecoveryType:     c.metrics.LastRecoveryType,
		LastError:            c.metrics.LastError,
	}
}

// GetStrategyCount 取得策略數量
func (c *RecoveryCoordinator) GetStrategyCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.strategies)
}

// ResetMetrics 重置指標
func (c *RecoveryCoordinator) ResetMetrics() {
	c.metrics.mu.Lock()
	defer c.metrics.mu.Unlock()
	c.metrics = &RecoveryMetrics{}
}

// FaultTolerantExecutor 容錯執行器
type FaultTolerantExecutor struct {
	retryExecutor *RetryExecutor
	detector      *MultiDetector
	coordinator   *RecoveryCoordinator
	metrics       *FaultToleranceMetrics
	mu            sync.RWMutex
}

// FaultToleranceMetrics 容錯指標統計
type FaultToleranceMetrics struct {
	TotalExecutions      int64
	SuccessfulExecutions int64
	FailedExecutions     int64
	RecoveredExecutions  int64
	TotalRetries         int64
	TotalRecoveryAttempts int64
	AverageExecutionTime time.Duration
	mu                   sync.RWMutex
}

// NewFaultTolerantExecutor 建立新的容錯執行器
func NewFaultTolerantExecutor(
	retryPolicy *RetryPolicy,
	detectorConfig *FailureDetectorConfig,
) *FaultTolerantExecutor {
	if retryPolicy == nil {
		retryPolicy = DefaultRetryPolicy()
	}
	if detectorConfig == nil {
		detectorConfig = DefaultFailureDetectorConfig()
	}

	return &FaultTolerantExecutor{
		retryExecutor: NewRetryExecutor(retryPolicy),
		detector:      BuildMultiDetector(detectorConfig),
		coordinator:   NewRecoveryCoordinator(),
		metrics:       &FaultToleranceMetrics{},
	}
}

// AddRecoveryStrategy 添加恢復策略
func (e *FaultTolerantExecutor) AddRecoveryStrategy(strategy RecoveryStrategy) {
	e.coordinator.AddStrategy(strategy)
}

// Execute 執行帶容錯的操作
func (e *FaultTolerantExecutor) Execute(ctx context.Context, fn func() error) error {
	startTime := time.Now()

	e.metrics.mu.Lock()
	e.metrics.TotalExecutions++
	e.metrics.mu.Unlock()

	// 使用重試執行器執行
	result := e.retryExecutor.ExecuteWithResult(ctx, func() (interface{}, error) {
		return nil, fn()
	})

	e.metrics.mu.Lock()
	e.metrics.TotalRetries += int64(result.Attempts - 1)
	e.metrics.mu.Unlock()

	if result.Error == nil {
		e.recordSuccess(time.Since(startTime))
		return nil
	}

	// 檢測是否是可恢復的故障
	failed, _ := e.detector.DetectWithType(result.Error, result.Duration)
	if !failed {
		e.recordFailure()
		return result.Error
	}

	// 嘗試恢復
	e.metrics.mu.Lock()
	e.metrics.TotalRecoveryAttempts++
	e.metrics.mu.Unlock()

	recoveryErr := e.coordinator.Recover(ctx, result.Error)
	if recoveryErr != nil {
		e.recordFailure()
		return fmt.Errorf("execution failed and recovery unsuccessful: %w", result.Error)
	}

	// 恢復成功後重新執行一次
	retryResult := e.retryExecutor.ExecuteWithResult(ctx, func() (interface{}, error) {
		return nil, fn()
	})

	if retryResult.Error == nil {
		e.recordRecovery(time.Since(startTime))
		return nil
	}

	e.recordFailure()
	return retryResult.Error
}

// recordSuccess 記錄成功執行
func (e *FaultTolerantExecutor) recordSuccess(duration time.Duration) {
	e.metrics.mu.Lock()
	defer e.metrics.mu.Unlock()
	e.metrics.SuccessfulExecutions++
	e.updateAverageTime(duration)
}

// recordFailure 記錄失敗執行
func (e *FaultTolerantExecutor) recordFailure() {
	e.metrics.mu.Lock()
	defer e.metrics.mu.Unlock()
	e.metrics.FailedExecutions++
}

// recordRecovery 記錄恢復成功的執行
func (e *FaultTolerantExecutor) recordRecovery(duration time.Duration) {
	e.metrics.mu.Lock()
	defer e.metrics.mu.Unlock()
	e.metrics.RecoveredExecutions++
	e.metrics.SuccessfulExecutions++
	e.updateAverageTime(duration)
}

// updateAverageTime 更新平均執行時間
func (e *FaultTolerantExecutor) updateAverageTime(duration time.Duration) {
	totalSuccessful := e.metrics.SuccessfulExecutions
	if totalSuccessful == 0 {
		return
	}
	// 簡單的移動平均
	if e.metrics.AverageExecutionTime == 0 {
		e.metrics.AverageExecutionTime = duration
	} else {
		e.metrics.AverageExecutionTime = (e.metrics.AverageExecutionTime + duration) / 2
	}
}

// GetMetrics 取得容錯指標
func (e *FaultTolerantExecutor) GetMetrics() *FaultToleranceMetrics {
	e.metrics.mu.RLock()
	defer e.metrics.mu.RUnlock()

	return &FaultToleranceMetrics{
		TotalExecutions:       e.metrics.TotalExecutions,
		SuccessfulExecutions:  e.metrics.SuccessfulExecutions,
		FailedExecutions:      e.metrics.FailedExecutions,
		RecoveredExecutions:   e.metrics.RecoveredExecutions,
		TotalRetries:          e.metrics.TotalRetries,
		TotalRecoveryAttempts: e.metrics.TotalRecoveryAttempts,
		AverageExecutionTime:  e.metrics.AverageExecutionTime,
	}
}

// GetRetryMetrics 取得重試指標
func (e *FaultTolerantExecutor) GetRetryMetrics() *RetryMetrics {
	return e.retryExecutor.GetMetrics()
}

// GetRecoveryMetrics 取得恢復指標
func (e *FaultTolerantExecutor) GetRecoveryMetrics() *RecoveryMetrics {
	return e.coordinator.GetMetrics()
}

// SetRetryPolicy 設定重試策略
func (e *FaultTolerantExecutor) SetRetryPolicy(policy *RetryPolicy) error {
	return e.retryExecutor.SetPolicy(policy)
}

// ResetAllMetrics 重置所有指標
func (e *FaultTolerantExecutor) ResetAllMetrics() {
	e.metrics.mu.Lock()
	e.metrics.TotalExecutions = 0
	e.metrics.SuccessfulExecutions = 0
	e.metrics.FailedExecutions = 0
	e.metrics.RecoveredExecutions = 0
	e.metrics.TotalRetries = 0
	e.metrics.TotalRecoveryAttempts = 0
	e.metrics.AverageExecutionTime = 0
	e.metrics.mu.Unlock()

	e.retryExecutor.ResetMetrics()
	e.coordinator.ResetMetrics()
}

// ResetDetectors 重置所有檢測器
func (e *FaultTolerantExecutor) ResetDetectors() {
	e.detector.Reset()
}
