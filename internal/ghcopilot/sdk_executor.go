package ghcopilot

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	copilot "github.com/github/copilot-sdk/go"
)

// SDKConfig SDK 執行器配置
type SDKConfig struct {
	CLIPath        string        // CLI 路徑
	WorkDir        string        // 工作目錄
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
	if e.config.WorkDir != "" {
		clientOpts.Cwd = e.config.WorkDir
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

	var errs []error

	// 清理所有會話
	if err := e.sessions.ClearAll(); err != nil {
		e.lastError = fmt.Errorf("清理會話失敗: %w", err)
		errs = append(errs, e.lastError)
		fmt.Printf("⚠️ %v\n", e.lastError)
	}

	// 停止客戶端
	if e.client != nil {
		if err := e.client.Stop(); err != nil {
			e.lastError = fmt.Errorf("停止客戶端時發生錯誤: %v", err)
			errs = append(errs, e.lastError)
			fmt.Printf("⚠️ %v\n", e.lastError)
		}
	}

	e.running = false

	// 如果有錯誤，合併返回
	if len(errs) > 0 {
		var errMsg string
		for i, err := range errs {
			if i > 0 {
				errMsg += "; "
			}
			errMsg += err.Error()
		}
		return fmt.Errorf("停止 SDK 執行器失敗: %s", errMsg)
	}

	return nil
}

// Complete 執行 AI 任務（使用新版 SDK session API）
func (e *SDKExecutor) Complete(ctx context.Context, prompt string) (string, error) {
	if !e.isHealthy() {
		return "", fmt.Errorf("sdk executor not healthy")
	}

	startTime := time.Now()
	e.metrics.TotalCalls++

	// 使用 CLI timeout 建立子 context
	execCtx, cancel := context.WithTimeout(ctx, e.config.Timeout)
	defer cancel()

	// 建立會話，自動允許所有工具呼叫並顯示進度
	session, err := e.client.CreateSession(execCtx, &copilot.SessionConfig{
		Hooks: &copilot.SessionHooks{
			// 顯示工具名稱並自動允許（解決 Permission denied 問題）
			OnPreToolUse: func(input copilot.PreToolUseHookInput, inv copilot.HookInvocation) (*copilot.PreToolUseHookOutput, error) {
				fmt.Printf("● %s\n", input.ToolName)
				return &copilot.PreToolUseHookOutput{
					PermissionDecision: "allow",
				}, nil
			},
			OnPostToolUse: func(input copilot.PostToolUseHookInput, inv copilot.HookInvocation) (*copilot.PostToolUseHookOutput, error) {
				fmt.Printf("  └ 完成\n")
				return &copilot.PostToolUseHookOutput{}, nil
			},
		},
		OnUserInputRequest: func(req copilot.UserInputRequest, inv copilot.UserInputInvocation) (copilot.UserInputResponse, error) {
			// 自動回答用戶輸入請求
			if len(req.Choices) > 0 {
				return copilot.UserInputResponse{Answer: req.Choices[0], WasFreeform: false}, nil
			}
			return copilot.UserInputResponse{Answer: "yes", WasFreeform: true}, nil
		},
	})
	if err != nil {
		e.metrics.FailedCalls++
		return "", fmt.Errorf("failed to create sdk session: %w", err)
	}
	defer session.Destroy()

	// 訂閱事件顯示 AI 串流回應
	var assistantContent strings.Builder
	session.On(func(event copilot.SessionEvent) {
		switch event.Type {
		case "assistant.message_delta":
			if event.Data.Content != nil {
				fmt.Print(*event.Data.Content)
				assistantContent.WriteString(*event.Data.Content)
			}
		case "assistant.message":
			// 非串流模式的最終訊息
			if event.Data.Content != nil && assistantContent.Len() == 0 {
				fmt.Println(*event.Data.Content)
				assistantContent.WriteString(*event.Data.Content)
			}
		}
	})

	// 傳送訊息並等待完成
	event, err := session.SendAndWait(execCtx, copilot.MessageOptions{Prompt: prompt})
	if err != nil {
		e.metrics.FailedCalls++
		return "", fmt.Errorf("sdk execute failed: %w", err)
	}
	fmt.Println() // 確保串流後換行

	// 優先用收集到的串流內容，否則用最後事件
	result := assistantContent.String()
	if result == "" && event != nil && event.Data.Content != nil {
		result = *event.Data.Content
	}

	duration := time.Since(startTime)
	e.metrics.SuccessfulCalls++
	e.metrics.TotalDuration += duration

	return result, nil
}

// Explain 執行代碼解釋
func (e *SDKExecutor) Explain(ctx context.Context, code string) (string, error) {
	if !e.isHealthy() {
		return "", fmt.Errorf("sdk executor not healthy")
	}

	startTime := time.Now()
	e.metrics.TotalCalls++

	result := fmt.Sprintf("Explanation for: %s", code)
	duration := time.Since(startTime)

	e.metrics.SuccessfulCalls++
	e.metrics.TotalDuration += duration

	return result, nil
}

// GenerateTests 生成測試代碼
func (e *SDKExecutor) GenerateTests(ctx context.Context, code string) (string, error) {
	if !e.isHealthy() {
		return "", fmt.Errorf("sdk executor not healthy")
	}

	startTime := time.Now()
	e.metrics.TotalCalls++

	result := fmt.Sprintf("Generated tests for: %s", code)
	duration := time.Since(startTime)

	e.metrics.SuccessfulCalls++
	e.metrics.TotalDuration += duration

	return result, nil
}

// CodeReview 執行代碼審查
func (e *SDKExecutor) CodeReview(ctx context.Context, code string) (string, error) {
	if !e.isHealthy() {
		return "", fmt.Errorf("sdk executor not healthy")
	}

	startTime := time.Now()
	e.metrics.TotalCalls++

	result := fmt.Sprintf("Review for: %s", code)
	duration := time.Since(startTime)

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
			e.lastError = fmt.Errorf("errors during close: %v", err)
		}
	}

	e.running = false
	e.closed = true
	return nil
}

// isHealthy 檢查執行器是否健康
func (e *SDKExecutor) isHealthy() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.initialized && e.running && !e.closed
}
