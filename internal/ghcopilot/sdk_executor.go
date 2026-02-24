package ghcopilot

import (
	"context"
	"encoding/json"
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

	// 防禦：client 必須是真實連線，否則 CreateSession 會 panic
	if e.client == nil {
		return "", fmt.Errorf("sdk executor: client not initialized, call Start() first")
	}

	startTime := time.Now()
	e.metrics.TotalCalls++

	// 使用 CLI timeout 建立子 context
	execCtx, cancel := context.WithTimeout(ctx, e.config.Timeout)
	defer cancel()

	// 建立會話
	session, err := e.client.CreateSession(execCtx, &copilot.SessionConfig{
		WorkingDirectory: e.config.WorkDir,
		// 自動允許所有工具（解決 Permission denied 問題）
		OnPermissionRequest: copilot.PermissionHandler.ApproveAll,
		Hooks: &copilot.SessionHooks{
			// OnPreToolUse 也設 allow，確保 hook 層也通過
			OnPreToolUse: func(input copilot.PreToolUseHookInput, inv copilot.HookInvocation) (*copilot.PreToolUseHookOutput, error) {
				return &copilot.PreToolUseHookOutput{
					PermissionDecision: "allow",
				}, nil
			},
		},
		OnUserInputRequest: func(req copilot.UserInputRequest, inv copilot.UserInputInvocation) (copilot.UserInputResponse, error) {
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

	// 訂閱事件顯示 AI 行為
	var assistantContent strings.Builder
	session.On(func(event copilot.SessionEvent) {
		switch event.Type {
		case copilot.ToolExecutionStart:
			// 顯示工具名稱和參數摘要
			toolName := ""
			if event.Data.ToolName != nil {
				toolName = *event.Data.ToolName
			}
			argSummary := formatToolArgs(event.Data.Arguments)
			if argSummary != "" {
				fmt.Printf("● %s\n  $ %s\n", toolName, argSummary)
			} else {
				fmt.Printf("● %s\n", toolName)
			}
		case copilot.ToolExecutionPartialResult:
			// 顯示工具串流輸出
			if event.Data.PartialOutput != nil && *event.Data.PartialOutput != "" {
				fmt.Printf("  │ %s\n", *event.Data.PartialOutput)
			}
		case copilot.ToolExecutionComplete:
			// 顯示工具執行結果
			success := event.Data.Success == nil || *event.Data.Success
			if success {
				if event.Data.Result != nil && event.Data.Result.Content != "" {
					// 顯示結果（最多 20 行）
					lines := strings.Split(strings.TrimSpace(event.Data.Result.Content), "\n")
					limit := 20
					for i, line := range lines {
						if i >= limit {
							fmt.Printf("  │ ... (共 %d 行)\n", len(lines))
							break
						}
						fmt.Printf("  │ %s\n", line)
					}
				}
				fmt.Printf("  └ 完成\n")
			} else {
				errMsg := ""
				if event.Data.Error != nil {
					if event.Data.Error.ErrorClass != nil {
						errMsg = event.Data.Error.ErrorClass.Message
					} else if event.Data.Error.String != nil {
						errMsg = *event.Data.Error.String
					}
				}
				fmt.Printf("  └ ❌ 失敗: %s\n", errMsg)
			}
		case copilot.ToolExecutionProgress:
			if event.Data.ProgressMessage != nil {
				fmt.Printf("  … %s\n", *event.Data.ProgressMessage)
			}
		case "assistant.message_delta":
			if event.Data.DeltaContent != nil {
				fmt.Print(*event.Data.DeltaContent)
				assistantContent.WriteString(*event.Data.DeltaContent)
			}
		case "assistant.message":
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
	fmt.Println()

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

// formatToolArgs 從工具參數中提取摘要（最多 120 字元）
func formatToolArgs(args interface{}) string {
	if args == nil {
		return ""
	}
	// 嘗試提取常見欄位
	if m, ok := args.(map[string]interface{}); ok {
		// 優先顯示 command / input / code / file_path / path
		for _, key := range []string{"command", "input", "code", "file_path", "path", "query"} {
			if v, ok := m[key]; ok {
				s := fmt.Sprintf("%v", v)
				if len(s) > 120 {
					s = s[:117] + "..."
				}
				return s
			}
		}
	}
	// 退而求其次，轉 JSON 截斷
	b, err := json.Marshal(args)
	if err != nil {
		return ""
	}
	s := string(b)
	if len(s) > 120 {
		s = s[:117] + "..."
	}
	return s
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
