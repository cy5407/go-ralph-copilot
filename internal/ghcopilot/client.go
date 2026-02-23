package ghcopilot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

// RalphLoopClient 是 Ralph Loop 系統的主要公開 API
//
// 它整合了所有內部模組，提供統一的介面用於：
// - CLI 執行與結果解析
// - 上下文管理與歷史追蹤
// - 自動重試與熔斷保護
// - 結果持久化
//
// 典型用法:
//
//	client := NewRalphLoopClient()
//	defer client.Close()
//
//	result, err := client.ExecuteLoop(ctx, "your prompt")
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println(result)
type RalphLoopClient struct {
	// 核心模組
	executor       *CLIExecutor
	parser         *OutputParser
	analyzer       *ResponseAnalyzer
	breaker        *CircuitBreaker
	contextManager *ContextManager
	persistence    *PersistenceManager

	// SDK 執行器（新增）
	sdkExecutor *SDKExecutor

	// 配置
	config *ClientConfig

	// 狀態
	initialized bool
	closed      bool
}

// ClientConfig 包含 Client 的配置選項
type ClientConfig struct {
	// CLI 配置
	CLITimeout    time.Duration // CLI 執行逾時 (預設: 30s)
	CLIMaxRetries int           // 最大重試次數 (預設: 3)
	WorkDir       string        // 工作目錄 (預設: 當前目錄)

	// 上下文配置
	MaxHistorySize int    // 最大歷史記錄 (預設: 100)
	SaveDir        string // 儲存目錄 (預設: ".ralph-loop/saves")
	UseGobFormat   bool   // 是否使用 Gob 格式 (預設: false，使用 JSON)

	// 熔斷器配置
	CircuitBreakerThreshold int // 無進展迴圈數 (預設: 3)
	SameErrorThreshold      int // 相同錯誤數 (預設: 5)

	// AI 模型配置
	Model  string // AI 模型名稱 (預設: "claude-sonnet-4.5")
	Silent bool   // 是否靜默模式 (預設: false)

	// 其他
	EnablePersistence bool // 是否啟用持久化 (預設: true)
	EnableSDK         bool // 是否啟用 SDK 執行器 (預設: true)
	PreferSDK         bool // 是否優先使用 SDK (預設: true)
}

// NewRalphLoopClient 建立新的 Ralph Loop 客戶端
func NewRalphLoopClient() *RalphLoopClient {
	return NewRalphLoopClientWithConfig(DefaultClientConfig())
}

// NewRalphLoopClientWithConfig 使用自訂配置建立客戶端
func NewRalphLoopClientWithConfig(config *ClientConfig) *RalphLoopClient {
	client := &RalphLoopClient{
		config:      config,
		initialized: false,
		closed:      false,
	}

	// 初始化各個模組
	client.executor = NewCLIExecutor(config.WorkDir)
	client.executor.SetTimeout(config.CLITimeout)
	client.executor.SetMaxRetries(config.CLIMaxRetries)
	if config.Model != "" {
		opts := DefaultOptions()
		opts.Model = Model(config.Model)
		opts.Silent = config.Silent
		client.executor.options = opts
	}
	client.executor.SetSilent(config.Silent)

	client.parser = NewOutputParser("")

	client.analyzer = NewResponseAnalyzer("")

	client.breaker = NewCircuitBreaker("")

	client.contextManager = NewContextManager()
	client.contextManager.SetMaxHistorySize(config.MaxHistorySize)

	if config.EnablePersistence {
		pm, err := NewPersistenceManager(config.SaveDir, config.UseGobFormat)
		if err != nil {
			log.Printf("⚠️ 持久化管理器初始化失敗: %v (持久化功能將被禁用)", err)
		} else {
			client.persistence = pm
		}
	}

	// 初始化 SDK 執行器
	sdkConfig := &SDKConfig{
		CLIPath:        "copilot",
		Timeout:        config.CLITimeout,
		SessionTimeout: 5 * time.Minute,
		MaxSessions:    100,
		LogLevel:       "info",
		EnableMetrics:  true,
		AutoReconnect:  true,
		MaxRetries:     config.CLIMaxRetries,
	}
	client.sdkExecutor = NewSDKExecutor(sdkConfig)

	client.initialized = true
	return client
}

// DefaultClientConfig 傳回預設的配置
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		CLITimeout:              60 * time.Second, // 增加到 60 秒以支援複雜任務
		CLIMaxRetries:           3,
		MaxHistorySize:          100,
		SaveDir:                 ".ralph-loop/saves",
		UseGobFormat:            false,
		CircuitBreakerThreshold: 3,
		SameErrorThreshold:      5,
		Model:                   "claude-sonnet-4.5",
		Silent:                  false,
		EnablePersistence:       true,
		EnableSDK:               true, // 預設啟用 SDK（主要執行方式）
		PreferSDK:               true, // 預設優先使用 SDK
	}
}

// ExecuteLoop 執行單個迴圈
//
// 這是最常用的方法。它會：
// 1. 執行 CLI 命令
// 2. 解析輸出
// 3. 分析回應
// 4. 檢查是否應該繼續或退出
// 5. 記錄結果到歷史
//
// 返回值：
// - LoopResult: 迴圈執行結果
// - error: 執行過程中的錯誤
func (c *RalphLoopClient) ExecuteLoop(ctx context.Context, prompt string) (*LoopResult, error) {
	if !c.initialized {
		return nil, fmt.Errorf("client not initialized")
	}
	if c.closed {
		return nil, fmt.Errorf("client is closed")
	}

	// 檢查熔斷器
	if c.breaker.IsOpen() {
		return nil, fmt.Errorf("circuit breaker is open: %s", c.breaker.GetState())
	}

	// 開始新迴圈
	loopIndex := len(c.contextManager.GetLoopHistory())
	execCtx := c.contextManager.StartLoop(loopIndex, prompt)

	defer func() {
		// 完成迴圈
		if err := c.contextManager.FinishLoop(); err != nil {
			log.Printf("⚠️ 迴圈結束記錄失敗: %v", err)
		}

		// 自動持久化整個 ContextManager（如果啟用）
		if c.persistence != nil && c.config.EnablePersistence {
			if err := c.persistence.SaveContextManager(c.contextManager); err != nil {
				log.Printf("⚠️ 上下文持久化失敗 (迴圈 %d): %v", loopIndex, err)
			}
		}
	}()

	// 根據配置決定執行順序：優先使用 SDK 或 CLI
	var output string
	var executionErr error
	var usedSDK bool

	// 如果配置優先使用 SDK，則先嘗試 SDK
	if c.config.PreferSDK && c.config.EnableSDK && c.sdkExecutor != nil && c.sdkExecutor.isHealthy() {
		output, executionErr = c.sdkExecutor.Complete(ctx, prompt)
		if executionErr == nil {
			usedSDK = true
			execCtx.CLICommand = "sdk:complete"
			execCtx.CLIOutput = output
			execCtx.CLIExitCode = 0
		}
	}

	// SDK 失敗/不可用/未啟用，或配置不優先使用 SDK 時，使用 CLI
	if !usedSDK {
		result, err := c.executor.ExecutePrompt(ctx, prompt)
		if err != nil {
			c.breaker.RecordSameError(err.Error())
			if executionErr != nil {
				execCtx.ExitReason = fmt.Sprintf("執行失敗 (SDK: %v, CLI: %v)", executionErr, err)
			} else {
				execCtx.ExitReason = fmt.Sprintf("CLI 執行失敗: %v", err)
			}
			return c.createResult(execCtx, false), nil
		}

		output = result.Stdout
		execCtx.CLICommand = result.Command
		execCtx.CLIOutput = result.Stdout
		execCtx.CLIExitCode = result.ExitCode

		if result.ExitCode != 0 {
			c.breaker.RecordSameError(fmt.Sprintf("exit code %d", result.ExitCode))
			execCtx.ExitReason = fmt.Sprintf("CLI 執行失敗，退出碼 %d", result.ExitCode)
			execCtx.ShouldContinue = false
			return c.createResult(execCtx, false), nil
		}
	}

	// 解析輸出
	parser := NewOutputParser(output)
	parser.Parse()
	codeBlocks := parser.GetOptions()

	execCtx.ParsedCodeBlocks = codeBlocks
	execCtx.CleanedOutput = output

	// 使用 ResponseAnalyzer 分析回應（雙重條件驗證）
	analyzer := NewResponseAnalyzer(output)
	score := analyzer.CalculateCompletionScore()
	execCtx.CompletionScore = score
	completed := analyzer.IsCompleted()

	shouldContinue := !completed
	execCtx.ShouldContinue = shouldContinue

	if !shouldContinue {
		c.breaker.RecordSuccess()
		execCtx.ExitReason = "任務完成 (EXIT_SIGNAL=true)"
	} else {
		// 只在輸出與前一次迴圈完全相同時才記錄無進展（真正卡住）
		history := c.contextManager.GetLoopHistory()
		if len(history) > 0 && history[len(history)-1].CLIOutput == output {
			c.breaker.RecordNoProgress()
		}
	}

	execCtx.CircuitBreakerState = string(c.breaker.GetState())

	// 個別執行上下文的持久化（可選）
	if c.persistence != nil && c.config.EnablePersistence {
		if err := c.persistence.SaveExecutionContext(execCtx); err != nil {
			// 記錄警告但不中斷執行流程
			fmt.Printf("⚠️ 儲存執行上下文失敗: %v\n", err)
		}
	}

	return c.createResult(execCtx, shouldContinue), nil
}

// ExecuteUntilCompletion 持續執行迴圈直到完成或錯誤
//
// 這個方法會自動處理迴圈，直到：
// - 系統回報完成
// - 熔斷器打開
// - Context 被取消
// - 達到最大迴圈次數
func (c *RalphLoopClient) ExecuteUntilCompletion(ctx context.Context, initialPrompt string, maxLoops int) ([]*LoopResult, error) {
	var results []*LoopResult

	for i := 0; i < maxLoops; i++ {
		select {
		case <-ctx.Done():
			return results, fmt.Errorf("context cancelled after %d loops", i)
		default:
		}

		// 顯示進度
		if !c.config.Silent {
			fmt.Printf("\n🔄 迴圈 %d/%d - 正在執行...\n", i+1, maxLoops)
		}

		result, err := c.ExecuteLoop(ctx, initialPrompt)
		if err != nil {
			if !c.config.Silent {
				fmt.Printf("❌ 迴圈 %d 失敗: %v\n", i+1, err)
			}
			return results, err
		}

		results = append(results, result)

		// 顯示迴圈結果
		if !c.config.Silent {
			if result.ShouldContinue {
				fmt.Printf("✓ 迴圈 %d 完成 - 繼續下一個迴圈\n", i+1)
			} else {
				fmt.Printf("✓ 迴圈 %d 完成 - 任務完成: %s\n", i+1, result.ExitReason)
			}
		}

		// 檢查是否完成
		if !result.ShouldContinue {
			return results, nil
		}

		// 檢查熔斷器
		if c.breaker.IsOpen() {
			return results, fmt.Errorf("circuit breaker opened after %d loops", i+1)
		}
	}

	return results, fmt.Errorf("reached maximum loops (%d) without completion", maxLoops)
}

// GetHistory 取得執行歷史
func (c *RalphLoopClient) GetHistory() []*ExecutionContext {
	return c.contextManager.GetLoopHistory()
}

// GetSummary 取得執行摘要
func (c *RalphLoopClient) GetSummary() map[string]interface{} {
	return c.contextManager.GetSummary()
}

// GetStatus 取得當前狀態
func (c *RalphLoopClient) GetStatus() *ClientStatus {
	return &ClientStatus{
		Initialized:         c.initialized,
		Closed:              c.closed,
		CircuitBreakerOpen:  c.breaker.IsOpen(),
		CircuitBreakerState: c.breaker.GetState(),
		LoopsExecuted:       len(c.contextManager.GetLoopHistory()),
		Summary:             c.GetSummary(),
	}
}

// ResetCircuitBreaker 重置熔斷器
func (c *RalphLoopClient) ResetCircuitBreaker() error {
	if !c.initialized {
		return fmt.Errorf("client not initialized")
	}
	c.breaker.Reset()
	return nil
}

// ClearHistory 清空歷史記錄
func (c *RalphLoopClient) ClearHistory() {
	if c.initialized {
		c.contextManager.Clear()
	}
}

// ExportHistory 匯出歷史為 JSON
func (c *RalphLoopClient) ExportHistory(outputPath string) error {
	if c.persistence == nil {
		return fmt.Errorf("persistence not enabled")
	}
	return c.persistence.ExportAsJSON(c.contextManager, outputPath)
}

// LoadHistoryFromDisk 從磁盤載入歷史記錄
//
// 此方法將從儲存目錄載入所有保存的執行上下文，
// 並恢復 ContextManager 的狀態。
//
// 使用時機：
// - 客戶端初始化後，需要恢復之前的迴圈歷史
// - 重啟應用程序時恢復狀態
func (c *RalphLoopClient) LoadHistoryFromDisk() error {
	if !c.initialized {
		return fmt.Errorf("client not initialized")
	}
	if c.closed {
		return fmt.Errorf("client is closed")
	}
	if c.persistence == nil {
		return fmt.Errorf("persistence not enabled")
	}

	// 從磁盤載入 ContextManager (使用預設檔名)
	loadedManager, err := c.persistence.LoadContextManager("context_manager.json")
	if err != nil {
		return fmt.Errorf("failed to load context manager: %w", err)
	}

	// 使用載入的管理器替換當前的
	c.contextManager = loadedManager
	return nil
}

// SaveHistoryToDisk 立即將歷史記錄儲存到磁盤
//
// 此方法強制將目前的執行歷史記錄保存到磁盤，
// 即使自動持久化未啟用。
//
// 使用時機：
// - 在應用程序關閉前確保所有數據已保存
// - 定期備份關鍵狀態
// - 手動觸發保存
func (c *RalphLoopClient) SaveHistoryToDisk() error {
	if !c.initialized {
		return fmt.Errorf("client not initialized")
	}
	if c.persistence == nil {
		return fmt.Errorf("persistence not enabled")
	}

	// 保存 ContextManager
	if err := c.persistence.SaveContextManager(c.contextManager); err != nil {
		return fmt.Errorf("failed to save context manager: %w", err)
	}

	// 同時保存當前迴圈（如果有）
	if len(c.contextManager.GetLoopHistory()) > 0 {
		lastLoop := c.contextManager.GetLoopByIndex(len(c.contextManager.GetLoopHistory()) - 1)
		if lastLoop != nil {
			if err := c.persistence.SaveExecutionContext(lastLoop); err != nil {
				// 不影響主流程，只記錄警告
				return fmt.Errorf("warning: failed to save last execution context: %w", err)
			}
		}
	}

	return nil
}

// GetPersistenceStats 取得持久化統計資訊
//
// 傳回持久化層的統計資訊，包括：
// - 儲存目錄路徑
// - 儲存的上下文數量
// - 最後保存時間
// - 使用的格式 (JSON/Gob)
func (c *RalphLoopClient) GetPersistenceStats() map[string]interface{} {
	stats := make(map[string]interface{})

	if c.persistence == nil {
		stats["enabled"] = false
		return stats
	}

	stats["enabled"] = true
	stats["storage_dir"] = c.persistence.GetStorageDir()
	stats["format"] = "json"
	if c.config.UseGobFormat {
		stats["format"] = "gob"
	}

	// 列出已保存的上下文
	savedContexts, err := c.persistence.ListSavedContexts()
	if err == nil {
		stats["saved_count"] = len(savedContexts)
		stats["saved_contexts"] = savedContexts
	}

	return stats
}

// CleanupOldBackups 清理舊的備份檔案
//
// 此方法會清理舊於指定天數的備份，
// 或根據 maxBackups 設定保留最新的備份。
//
// 參數:
// - prefix: 備份檔名前綴 (如 "context_manager" 或 "execution_context")
//
// 返回值:
// - error: 清理過程中的錯誤
func (c *RalphLoopClient) CleanupOldBackups(prefix string) error {
	if !c.initialized {
		return fmt.Errorf("client not initialized")
	}
	if c.persistence == nil {
		return fmt.Errorf("persistence not enabled")
	}

	return c.persistence.ClearOldBackups(prefix)
}

// SetMaxBackupCount 設定最多保留的備份數量
//
// 此方法會設定持久化管理器最多保留多少個備份檔案。
// 預設值為 10。
//
// 參數:
// - count: 最多保留的備份數量 (必須 > 0)
//
// 範例:
//
//	client.SetMaxBackupCount(20)  // 最多保留 20 個備份
func (c *RalphLoopClient) SetMaxBackupCount(count int) error {
	if !c.initialized {
		return fmt.Errorf("client not initialized")
	}
	if c.persistence == nil {
		return fmt.Errorf("persistence not enabled")
	}
	if count <= 0 {
		return fmt.Errorf("backup count must be greater than 0")
	}

	c.persistence.SetMaxBackups(count)
	return nil
}

// ListBackups 列出所有備份
//
// 傳回指定前綴的所有備份檔案列表。
//
// 參數:
// - prefix: 備份檔名前綴
//
// 返回值:
// - []string: 備份檔案名稱列表
// - error: 列舉過程中的錯誤
func (c *RalphLoopClient) ListBackups(prefix string) ([]string, error) {
	if !c.initialized {
		return nil, fmt.Errorf("client not initialized")
	}
	if c.persistence == nil {
		return nil, fmt.Errorf("persistence not enabled")
	}

	// 使用 ListSavedContexts 作為備份列表
	contexts, err := c.persistence.ListSavedContexts()
	if err != nil {
		return nil, err
	}

	// 過濾符合前綴的備份
	var backups []string
	for _, ctx := range contexts {
		if strings.HasPrefix(ctx, prefix) {
			backups = append(backups, ctx)
		}
	}

	return backups, nil
}

// RecoverFromBackup 從備份恢復狀態
//
// 此方法從指定的備份檔案恢復執行上下文和系統狀態。
// 可用於故障恢復或狀態復制。
//
// 參數:
// - filename: 備份檔名
//
// 返回值:
// - error: 恢復過程中的錯誤
func (c *RalphLoopClient) RecoverFromBackup(filename string) error {
	if !c.initialized {
		return fmt.Errorf("client not initialized")
	}
	if c.closed {
		return fmt.Errorf("client is closed")
	}
	if c.persistence == nil {
		return fmt.Errorf("persistence not enabled")
	}

	// 從備份載入
	execCtx, err := c.persistence.LoadExecutionContext(filename)
	if err != nil {
		return fmt.Errorf("failed to load backup: %w", err)
	}

	if execCtx == nil {
		return fmt.Errorf("loaded backup is empty")
	}

	// 恢復迴圈索引到該執行上下文
	// 清空當前歷史並添加恢復的上下文
	c.contextManager.Clear()
	c.contextManager.StartLoop(execCtx.LoopIndex, execCtx.UserPrompt)
	c.contextManager.UpdateCurrentLoop(func(ctx *ExecutionContext) {
		*ctx = *execCtx
	})
	c.contextManager.FinishLoop()

	return nil
}

// VerifyStateConsistency 驗證狀態一致性
//
// 此方法檢查保存的狀態與當前狀態是否一致，
// 用於檢測損毀或不一致的備份。
//
// 返回值:
// - bool: 狀態是否一致
// - error: 驗證過程中的錯誤
func (c *RalphLoopClient) VerifyStateConsistency() (bool, error) {
	if !c.initialized {
		return false, fmt.Errorf("client not initialized")
	}
	if c.persistence == nil {
		return false, fmt.Errorf("persistence not enabled")
	}

	// 取得當前狀態
	currentCount := len(c.contextManager.GetLoopHistory())

	// 列出已保存的備份
	savedContexts, err := c.persistence.ListSavedContexts()
	if err != nil {
		return false, fmt.Errorf("failed to list saved contexts: %w", err)
	}

	// 基本一致性檢查：備份計數不應遠大於當前迴圈計數
	// (允許某些差異是因為備份可能更新)
	if len(savedContexts) > currentCount*2 {
		return false, fmt.Errorf("saved backups count significantly exceeds current loops")
	}

	return true, nil
}

// Close 關閉客戶端並清理資源
func (c *RalphLoopClient) Close() error {
	if c.closed {
		return fmt.Errorf("client already closed")
	}

	var errs []error

	// 執行最後的持久化
	if c.persistence != nil && c.config.EnablePersistence {
		if err := c.persistence.SaveContextManager(c.contextManager); err != nil {
			errs = append(errs, fmt.Errorf("儲存上下文管理器失敗: %w", err))
		}
	}

	// 關閉 SDK 執行器
	if c.sdkExecutor != nil {
		if err := c.sdkExecutor.Close(); err != nil {
			errs = append(errs, fmt.Errorf("關閉 SDK 執行器失敗: %w", err))
		}
	}

	c.closed = true

	// 如果有錯誤，合併返回
	if len(errs) > 0 {
		var errMsg string
		for i, err := range errs {
			if i > 0 {
				errMsg += "; "
			}
			errMsg += err.Error()
		}
		return fmt.Errorf("關閉客戶端時發生錯誤: %s", errMsg)
	}

	return nil
}

// StartSDKExecutor 啟動 SDK 執行器
// 這使用新的 SDK 層進行程式碼執行，提供更細粒度的控制
func (c *RalphLoopClient) StartSDKExecutor(ctx context.Context) error {
	if !c.initialized {
		return fmt.Errorf("client not initialized")
	}
	if c.closed {
		return fmt.Errorf("client is closed")
	}
	if c.sdkExecutor == nil {
		return fmt.Errorf("SDK executor not available")
	}

	return c.sdkExecutor.Start(ctx)
}

// StopSDKExecutor 停止 SDK 執行器
func (c *RalphLoopClient) StopSDKExecutor(ctx context.Context) error {
	if c.sdkExecutor == nil {
		return fmt.Errorf("SDK executor not available")
	}

	return c.sdkExecutor.Stop(ctx)
}

// ExecuteWithSDK 使用 SDK 執行程式碼完成
// 提供比標準 ExecuteLoop 更直接的程式碼執行介面
func (c *RalphLoopClient) ExecuteWithSDK(ctx context.Context, prompt string) (string, error) {
	if !c.initialized {
		return "", fmt.Errorf("client not initialized")
	}
	if c.closed {
		return "", fmt.Errorf("client is closed")
	}
	if c.sdkExecutor == nil {
		return "", fmt.Errorf("SDK executor not available")
	}

	return c.sdkExecutor.Complete(ctx, prompt)
}

// ExplainWithSDK 使用 SDK 解釋程式碼
func (c *RalphLoopClient) ExplainWithSDK(ctx context.Context, code string) (string, error) {
	if !c.initialized {
		return "", fmt.Errorf("client not initialized")
	}
	if c.closed {
		return "", fmt.Errorf("client is closed")
	}
	if c.sdkExecutor == nil {
		return "", fmt.Errorf("SDK executor not available")
	}

	return c.sdkExecutor.Explain(ctx, code)
}

// GenerateTestsWithSDK 使用 SDK 生成測試
func (c *RalphLoopClient) GenerateTestsWithSDK(ctx context.Context, code string) (string, error) {
	if !c.initialized {
		return "", fmt.Errorf("client not initialized")
	}
	if c.closed {
		return "", fmt.Errorf("client is closed")
	}
	if c.sdkExecutor == nil {
		return "", fmt.Errorf("SDK executor not available")
	}

	return c.sdkExecutor.GenerateTests(ctx, code)
}

// CodeReviewWithSDK 使用 SDK 進行程式碼審查
func (c *RalphLoopClient) CodeReviewWithSDK(ctx context.Context, code string) (string, error) {
	if !c.initialized {
		return "", fmt.Errorf("client not initialized")
	}
	if c.closed {
		return "", fmt.Errorf("client is closed")
	}
	if c.sdkExecutor == nil {
		return "", fmt.Errorf("SDK executor not available")
	}

	return c.sdkExecutor.CodeReview(ctx, code)
}

// GetSDKStatus 取得 SDK 執行器狀態
func (c *RalphLoopClient) GetSDKStatus() *SDKStatus {
	if c.sdkExecutor == nil {
		return nil
	}

	return c.sdkExecutor.GetStatus()
}

// ListSDKSessions 列出所有 SDK 會話
func (c *RalphLoopClient) ListSDKSessions() []*SDKSession {
	if c.sdkExecutor == nil {
		return nil
	}

	return c.sdkExecutor.ListSessions()
}

// GetSDKSessionCount 取得 SDK 會話數
func (c *RalphLoopClient) GetSDKSessionCount() int {
	if c.sdkExecutor == nil {
		return 0
	}

	return c.sdkExecutor.GetSessionCount()
}

// TerminateSDKSession 終止特定的 SDK 會話
func (c *RalphLoopClient) TerminateSDKSession(sessionID string) error {
	if c.sdkExecutor == nil {
		return fmt.Errorf("SDK executor not available")
	}

	session, err := c.sdkExecutor.GetSession(sessionID)
	if err != nil {
		return err
	}

	return c.sdkExecutor.sessions.RemoveSession(session.ID)
}

// 私有輔助函式

func (c *RalphLoopClient) createResult(execCtx *ExecutionContext, shouldContinue bool) *LoopResult {
	return &LoopResult{
		LoopID:          execCtx.LoopID,
		LoopIndex:       execCtx.LoopIndex,
		ShouldContinue:  shouldContinue,
		CompletionScore: execCtx.CompletionScore,
		Output:          execCtx.CLIOutput,
		ExitReason:      execCtx.ExitReason,
		Timestamp:       execCtx.Timestamp,
	}
}

// LoopResult 表示單個迴圈的結果
type LoopResult struct {
	LoopID          string
	LoopIndex       int
	ShouldContinue  bool
	CompletionScore int
	Output          string
	ExitReason      string
	Timestamp       time.Time
}

// ClientStatus 表示客戶端的當前狀態
type ClientStatus struct {
	Initialized         bool
	Closed              bool
	CircuitBreakerOpen  bool
	CircuitBreakerState CircuitBreakerState
	LoopsExecuted       int
	Summary             map[string]interface{}
}

// ClientBuilder 用於建立自訂配置的客戶端
type ClientBuilder struct {
	config *ClientConfig
}

// NewClientBuilder 建立新的客戶端建構器
func NewClientBuilder() *ClientBuilder {
	return &ClientBuilder{
		config: DefaultClientConfig(),
	}
}

// WithTimeout 設定 CLI 執行逾時
func (b *ClientBuilder) WithTimeout(duration time.Duration) *ClientBuilder {
	b.config.CLITimeout = duration
	return b
}

// WithMaxRetries 設定最大重試次數
func (b *ClientBuilder) WithMaxRetries(count int) *ClientBuilder {
	b.config.CLIMaxRetries = count
	return b
}

// WithWorkDir 設定工作目錄
func (b *ClientBuilder) WithWorkDir(dir string) *ClientBuilder {
	b.config.WorkDir = dir
	return b
}

// WithModel 設定 AI 模型
func (b *ClientBuilder) WithModel(model string) *ClientBuilder {
	b.config.Model = model
	return b
}

// WithSaveDir 設定儲存目錄
func (b *ClientBuilder) WithSaveDir(dir string) *ClientBuilder {
	b.config.SaveDir = dir
	return b
}

// WithMaxHistory 設定最大歷史記錄
func (b *ClientBuilder) WithMaxHistory(size int) *ClientBuilder {
	b.config.MaxHistorySize = size
	return b
}

// WithGobFormat 啟用 Gob 格式
func (b *ClientBuilder) WithGobFormat(enabled bool) *ClientBuilder {
	b.config.UseGobFormat = enabled
	return b
}

// WithoutPersistence 禁用持久化
func (b *ClientBuilder) WithoutPersistence() *ClientBuilder {
	b.config.EnablePersistence = false
	return b
}

// Build 建立客戶端
func (b *ClientBuilder) Build() *RalphLoopClient {
	return NewRalphLoopClientWithConfig(b.config)
}
