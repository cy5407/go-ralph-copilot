package ghcopilot

import (
	"context"
	"fmt"
	"sync"
	"time"

	copilot "github.com/github/copilot-sdk/go"
)

// SDKConfig SDK 執行器配置
type SDKConfig struct {
	CLIPath        string        // CLI 路徑
	Timeout        time.Duration // 執行逾時
	SessionTimeout time.Duration // 會話逾時
	MaxSessions    int           // 最大會話數
	LogLevel       string        // 日誌級別
	EnableMetrics  bool          // 啟用指標
	AutoReconnect  bool          // 自動重新連接
	MaxRetries     int           // 最大重試次數
}

// DefaultSDKConfig 預設 SDK 配置
func DefaultSDKConfig() *SDKConfig {
	return &SDKConfig{
		CLIPath:        "copilot",
		Timeout:        30 * time.Second,
		SessionTimeout: 5 * time.Minute,
		MaxSessions:    100,
		LogLevel:       "info",
		EnableMetrics:  true,
		AutoReconnect:  true,
		MaxRetries:     3,
	}
}

// SDKExecutor SDK 執行器
type SDKExecutor struct {
	client      *copilot.Client
	config      *SDKConfig
	sessions    *SDKSessionPool
	mu          sync.RWMutex
	initialized bool
	running     bool
	closed      bool
	lastError   error
	metrics     *SDKExecutorMetrics
}

// SDKExecutorMetrics 執行器指標
type SDKExecutorMetrics struct {
	TotalCalls      int64
	SuccessfulCalls int64
	FailedCalls     int64
	TotalDuration   time.Duration
	StartTime       time.Time
}

// NewSDKExecutor 建立新的 SDK 執行器
func NewSDKExecutor(config *SDKConfig) *SDKExecutor {
	if config == nil {
		config = DefaultSDKConfig()
	}

	return &SDKExecutor{
		config:   config,
		sessions: NewSDKSessionPool(config.MaxSessions, config.SessionTimeout),
		metrics:  &SDKExecutorMetrics{StartTime: time.Now()},
	}
}

// Start 啟動 SDK 執行器
func (e *SDKExecutor) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		return fmt.Errorf("sdk executor already running")
	}

	if e.closed {
		return fmt.Errorf("sdk executor already closed")
	}

	// 建立客戶端
	clientOpts := &copilot.ClientOptions{
		CLIPath:  e.config.CLIPath,
		LogLevel: e.config.LogLevel,
	}

	e.client = copilot.NewClient(clientOpts)
	if e.client == nil {
		e.lastError = fmt.Errorf("failed to create copilot client")
		return e.lastError
	}

	// 啟動客戶端
	if err := e.client.Start(ctx); err != nil {
		e.lastError = fmt.Errorf("failed to start copilot client: %w", err)
		return e.lastError
	}

	e.initialized = true
	e.running = true
	return nil
}

// Stop 停止 SDK 執行器
func (e *SDKExecutor) Stop(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return fmt.Errorf("sdk executor not running")
	}

	// 清理所有會話
	if err := e.sessions.ClearAll(); err != nil {
		e.lastError = fmt.Errorf("failed to clear sessions: %w", err)
	}

	// 停止客戶端
	if e.client != nil {
		if err := e.client.Stop(); err != nil {
			e.lastError = fmt.Errorf("error during client stop: %w", err)
		}
	}

	e.running = false
	return nil
}

// Complete 執行代碼完成
func (e *SDKExecutor) Complete(ctx context.Context, prompt string) (string, error) {
	startTime := time.Now()
	e.metrics.TotalCalls++

	if !e.isHealthy() {
		e.metrics.FailedCalls++
		return "", fmt.Errorf("sdk executor not healthy")
	}

	// 使用通用會話處理
	result, err := e.executeWithSession(ctx, prompt)
	
	duration := time.Since(startTime)
	if err != nil {
		e.metrics.FailedCalls++
		return "", fmt.Errorf("completion failed: %w", err)
	}

	e.metrics.SuccessfulCalls++
	e.metrics.TotalDuration += duration
	return result, nil
}

// Explain 執行代碼解釋
func (e *SDKExecutor) Explain(ctx context.Context, code string) (string, error) {
	startTime := time.Now()
	e.metrics.TotalCalls++

	if !e.isHealthy() {
		e.metrics.FailedCalls++
		return "", fmt.Errorf("sdk executor not healthy")
	}

	// 構建解釋 prompt
	prompt := fmt.Sprintf("請解釋以下代碼的功能和工作原理：\n\n```\n%s\n```", code)

	// 使用通用會話處理
	result, err := e.executeWithSession(ctx, prompt)
	
	duration := time.Since(startTime)
	if err != nil {
		e.metrics.FailedCalls++
		return "", fmt.Errorf("explanation failed: %w", err)
	}

	e.metrics.SuccessfulCalls++
	e.metrics.TotalDuration += duration
	return result, nil
}

// GenerateTests 生成測試代碼
func (e *SDKExecutor) GenerateTests(ctx context.Context, code string) (string, error) {
	startTime := time.Now()
	e.metrics.TotalCalls++

	if !e.isHealthy() {
		e.metrics.FailedCalls++
		return "", fmt.Errorf("sdk executor not healthy")
	}

	// 構建測試生成 prompt
	prompt := fmt.Sprintf("為以下代碼生成完整的單元測試，包含邊界條件和錯誤處理測試：\n\n```\n%s\n```\n\n請使用適當的測試框架並包含測試說明。", code)

	// 使用通用會話處理
	result, err := e.executeWithSession(ctx, prompt)
	
	duration := time.Since(startTime)
	if err != nil {
		e.metrics.FailedCalls++
		return "", fmt.Errorf("test generation failed: %w", err)
	}

	e.metrics.SuccessfulCalls++
	e.metrics.TotalDuration += duration
	return result, nil
}

// CodeReview 執行代碼審查
func (e *SDKExecutor) CodeReview(ctx context.Context, code string) (string, error) {
	startTime := time.Now()
	e.metrics.TotalCalls++

	if !e.isHealthy() {
		e.metrics.FailedCalls++
		return "", fmt.Errorf("sdk executor not healthy")
	}

	// 構建代碼審查 prompt
	prompt := fmt.Sprintf("請審查以下代碼，指出潛在的問題、改進建議和最佳實踐：\n\n```\n%s\n```\n\n請重點關注：\n1. 安全性問題\n2. 性能問題\n3. 代碼品質\n4. 錯誤處理\n5. 可讀性和維護性", code)

	// 使用通用會話處理
	result, err := e.executeWithSession(ctx, prompt)
	
	duration := time.Since(startTime)
	if err != nil {
		e.metrics.FailedCalls++
		return "", fmt.Errorf("code review failed: %w", err)
	}

	e.metrics.SuccessfulCalls++
	e.metrics.TotalDuration += duration
	return result, nil
}

// CreateSession 建立新會話
func (e *SDKExecutor) CreateSession(sessionID string) (*SDKSession, error) {
	if !e.isHealthy() {
		return nil, fmt.Errorf("sdk executor not healthy")
	}

	return e.sessions.CreateSession(sessionID)
}

// GetSession 取得會話
func (e *SDKExecutor) GetSession(sessionID string) (*SDKSession, error) {
	if !e.initialized {
		return nil, fmt.Errorf("sdk executor not initialized")
	}

	return e.sessions.GetSession(sessionID)
}

// ListSessions 列出所有會話
func (e *SDKExecutor) ListSessions() []*SDKSession {
	return e.sessions.ListSessions()
}

// TerminateSession 終止會話
func (e *SDKExecutor) TerminateSession(sessionID string) error {
	if !e.initialized {
		return fmt.Errorf("sdk executor not initialized")
	}

	return e.sessions.RemoveSession(sessionID)
}

// GetSessionCount 取得會話計數
func (e *SDKExecutor) GetSessionCount() int {
	return e.sessions.GetSessionCount()
}

// CleanupExpiredSessions 清理過期會話
func (e *SDKExecutor) CleanupExpiredSessions() int {
	return e.sessions.CleanupExpiredSessions()
}

// GetMetrics 取得執行器指標
func (e *SDKExecutor) GetMetrics() *SDKExecutorMetrics {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return &SDKExecutorMetrics{
		TotalCalls:      e.metrics.TotalCalls,
		SuccessfulCalls: e.metrics.SuccessfulCalls,
		FailedCalls:     e.metrics.FailedCalls,
		TotalDuration:   e.metrics.TotalDuration,
		StartTime:       e.metrics.StartTime,
	}
}

// GetStatus 取得執行器狀態
func (e *SDKExecutor) GetStatus() *SDKStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return &SDKStatus{
		Initialized:  e.initialized,
		Running:      e.running,
		Closed:       e.closed,
		SessionCount: e.sessions.GetSessionCount(),
		LastError:    e.lastError,
		Uptime:       time.Since(e.metrics.StartTime),
	}
}

// SDKStatus SDK 執行器狀態
type SDKStatus struct {
	Initialized  bool
	Running      bool
	Closed       bool
	SessionCount int
	LastError    error
	Uptime       time.Duration
}

// Close 關閉執行器
func (e *SDKExecutor) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.closed {
		return fmt.Errorf("sdk executor already closed")
	}

	// 清理會話
	_ = e.sessions.ClearAll()

	// 停止客戶端
	if e.client != nil && e.running {
		if err := e.client.Stop(); err != nil {
			e.lastError = fmt.Errorf("error during close: %w", err)
		}
	}

	e.running = false
	e.closed = true
	return nil
}

// executeWithSession 使用會話執行 prompt 的通用方法
func (e *SDKExecutor) executeWithSession(ctx context.Context, prompt string) (string, error) {
	// 檢查 client 是否有效
	if e.client == nil {
		return "", fmt.Errorf("sdk client not initialized")
	}

	var lastErr error
	maxRetries := e.config.MaxRetries
	if maxRetries == 0 {
		maxRetries = 3
	}

	for retry := 0; retry < maxRetries; retry++ {
		// 創建會話配置
		sessionConfig := &copilot.SessionConfig{
			Model: "claude-sonnet-4.5",
		}

		// 創建會話
		session, err := e.client.CreateSession(ctx, sessionConfig)
		if err != nil {
			lastErr = fmt.Errorf("failed to create session (attempt %d/%d): %w", retry+1, maxRetries, err)
			if retry < maxRetries-1 {
				// 等待一段時間後重試
				time.Sleep(time.Duration(retry+1) * time.Second)
				continue
			}
			return "", lastErr
		}

		// 設定超時
		timeout := e.config.Timeout
		if timeout == 0 {
			timeout = 30 * time.Second
		}

		// 發送請求並等待回應，使用帶 timeout 的 context
		sendCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		
		event, err := session.SendAndWait(sendCtx, copilot.MessageOptions{
			Prompt: prompt,
		})

		// 清理會話
		if destroyErr := session.Destroy(); destroyErr != nil {
			// 記錄錯誤但不影響主要邏輯
		}

		if err != nil {
			lastErr = fmt.Errorf("failed to get response (attempt %d/%d): %w", retry+1, maxRetries, err)
			if retry < maxRetries-1 {
				time.Sleep(time.Duration(retry+1) * time.Second)
				continue
			}
			return "", lastErr
		}

		// 檢查回應
		if event == nil {
			lastErr = fmt.Errorf("empty event from session (attempt %d/%d)", retry+1, maxRetries)
			if retry < maxRetries-1 {
				time.Sleep(time.Duration(retry+1) * time.Second)
				continue
			}
			return "", lastErr
		}

		// 檢查內容
		if event.Data.Content == nil {
			lastErr = fmt.Errorf("empty content from session (attempt %d/%d)", retry+1, maxRetries)
			if retry < maxRetries-1 {
				time.Sleep(time.Duration(retry+1) * time.Second)
				continue
			}
			return "", lastErr
		}

		// 成功
		return *event.Data.Content, nil
	}

	return "", lastErr
}

// isHealthy 檢查執行器是否健康
func (e *SDKExecutor) isHealthy() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.initialized && e.running && !e.closed
}
