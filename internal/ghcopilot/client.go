package ghcopilot

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/cy540/ralph-loop/internal/logger"
	"github.com/cy540/ralph-loop/internal/metrics"
	"github.com/cy540/ralph-loop/internal/security"
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
	exitDetector   *ExitDetector  // 退出偵測器

	// SDK 執行器
	sdkExecutor *SDKExecutor

	// 執行模式選擇器（T-006）
	modeSelector *ExecutionModeSelector

	// 重試執行器（T-008）
	retryExecutor *RetryExecutor

	// 故障檢測器（T-007）
	failureDetectors []FailureDetector

	// 恢復策略（T-007）
	recoveryStrategies []RecoveryStrategy

	// 日誌與監控（T2-007）
	logger           *logger.Logger
	metricsCollector *metrics.MetricsCollector
	
	// 安全管理器（T2-009）
	securityManager *security.SecurityManager

	// 性能優化器（T2-012）
	cacheManager       *CacheManager                 // 緩存管理器
	memoryPool         *MemoryPool                   // 記憶體池
	concurrentManager  *ConcurrentExecutionManager   // 併發執行管理器

	// 插件系統（T2-011）
	pluginManager      *PluginManager                // 插件管理器

	// Promise Detection（參考 doggy8088/copilot-ralph）
	promiseDetector    *PromiseDetector              // 承諾偵測器

	// 配置
	config *ClientConfig
	
	// UI 回調
	uiCallback UICallback

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
	SaveDir        string // 儲存目錄 (預設: ".ralph-loop/saves" 或平台對應路徑)
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
	
	// UI 配置
	Verbose bool // 詳細輸出模式 (預設: false)
	Quiet   bool // 安靜模式 (預設: false)
	
	// 安全配置
	Security security.SecurityConfig // 安全相關設定

	// 性能優化配置（T2-012）
	EnableCaching         bool          // 啟用 AI 回應緩存 (預設: true)
	CacheMaxSize          int           // 緩存最大項目數 (預設: 1000)
	CacheTTL              time.Duration // 緩存生存時間 (預設: 30分鐘)
	EnableConcurrency     bool          // 啟用併發執行 (預設: false)
	MaxConcurrentWorkers  int           // 最大併發工作者數 (預設: CPU核心數)
	EnableMemoryPool      bool          // 啟用記憶體池 (預設: true)
	MemoryOptimization    bool          // 啟用記憶體優化 (預設: true)

	// 插件系統配置（T2-011）
	EnablePluginSystem    bool          // 啟用插件系統 (預設: false)
	PluginDir             string        // 插件目錄 (預設: "./plugins")
	AutoLoadPlugins       bool          // 自動載入插件 (預設: false)
	PreferredExecutor     string        // 首選執行器插件名稱 (預設: "")

	// Promise Detection 配置
	// 參考自 doggy8088/copilot-ralph 的完成偵測設計
	PromisePhrase       string // 完成承諾詞 (預設: "任務完成！🥇")
	EnablePromiseDetect bool   // 啟用 Promise Detection (預設: true)
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

	// 初始化日誌器
	loggerConfig := logger.DefaultConfig()
	loggerConfig.Component = "ralph-loop"
	if config.Verbose {
		loggerConfig.Level = logger.DEBUG
	}
	if config.SaveDir != "" {
		loggerConfig.OutputFile = filepath.Join(config.SaveDir, "ralph-loop.log")
	}
	
	var err error
	client.logger, err = logger.New(loggerConfig)
	if err != nil {
		// 如果創建失敗，使用預設的全域日誌器
		client.logger = logger.WithField("component", "ralph-loop")
	}

	// 初始化指標收集器
	client.metricsCollector = metrics.NewCollector()

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

	// 初始化退出偵測器
	client.exitDetector = NewExitDetector(config.WorkDir)

	if config.EnablePersistence {
		pm, err := NewPersistenceManager(config.SaveDir, config.UseGobFormat)
		if err == nil {
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

	// 初始化執行模式選擇器（T-006）
	client.modeSelector = NewExecutionModeSelector()
	if config.PreferSDK {
		client.modeSelector.SetDefaultMode(ModeSDK)
	} else {
		client.modeSelector.SetDefaultMode(ModeCLI)
	}
	client.modeSelector.SetSDKAvailable(config.EnableSDK && client.sdkExecutor != nil)
	client.modeSelector.SetCLIAvailable(true)  // CLI 始終可用
	client.modeSelector.SetPluginAvailable(config.EnablePluginSystem && client.pluginManager != nil)  // 插件可用性
	client.modeSelector.SetFallbackEnabled(true)  // 啟用故障轉移

	// 初始化重試執行器（T-008）
	retryPolicy := NewExponentialBackoffPolicy(config.CLIMaxRetries)
	retryPolicy.InitialDelay = 100 * time.Millisecond
	retryPolicy.MaxDelay = 30 * time.Second
	retryPolicy.Jitter = true
	client.retryExecutor = NewRetryExecutor(retryPolicy)

	// 初始化故障檢測器（T-007）
	client.failureDetectors = []FailureDetector{
		NewTimeoutDetector(config.CLITimeout),
		NewErrorRateDetector(10, 0.5),  // 窗口 10，錯誤率閾值 50%
	}

	// 初始化恢復策略（T-007）
	client.recoveryStrategies = []RecoveryStrategy{
		NewAutoReconnectRecovery(3),  // 自動重連，最多 3 次
		NewFallbackRecovery(),  // SDK/CLI 故障轉移
	}
	
	// 初始化預設 UI 回調
	client.uiCallback = NewDefaultUICallback(config.Verbose, config.Quiet)

	// 初始化 Promise Detector（參考 doggy8088/copilot-ralph）
	if config.EnablePromiseDetect {
		client.promiseDetector = NewPromiseDetector(config.PromisePhrase)
	}
	
	// 設置串流回調到 CLI 執行器
	client.executor.SetStreamCallback(
		func(line string) {
			// Promise Detection：在串流中即時偵測完成承諾
			if client.promiseDetector != nil {
				client.promiseDetector.Check(line)
			}
			if client.uiCallback != nil {
				client.uiCallback.OnStreamOutput(line)
			}
		},
		func(line string) {
			if client.uiCallback != nil {
				client.uiCallback.OnStreamError(line)
			}
		},
	)
	
	// 初始化性能優化器（T2-012）
	if config.EnableMemoryPool {
		client.memoryPool = NewMemoryPool()
	}
	
	if config.EnableCaching {
		cacheConfig := &CacheConfig{
			MaxSize:         config.CacheMaxSize,
			TTL:             config.CacheTTL,
			CleanupInterval: 5 * time.Minute,
			EnableCaching:   true,
		}
		client.cacheManager = NewCacheManager(cacheConfig)
	}
	
	if config.EnableConcurrency {
		maxWorkers := config.MaxConcurrentWorkers
		if maxWorkers <= 0 {
			maxWorkers = runtime.NumCPU()
		}
		client.concurrentManager = NewConcurrentExecutionManager(maxWorkers, maxWorkers*10)
	}
	
	// 初始化插件系統（T2-011）
	if config.EnablePluginSystem {
		pluginConfig := &PluginConfig{
			PluginDir:           config.PluginDir,
			AutoLoadOnStart:     config.AutoLoadPlugins,
			HealthCheckInterval: 30 * time.Second,
			EnableHotReload:     false, // 暫時不支持熱重載
			DefaultTimeout:      config.CLITimeout,
			MaxPlugins:          10,
			RequiredPlugins:     []string{},
		}
		client.pluginManager = NewPluginManager(pluginConfig)
		
		// 如果啟用自動載入，啟動插件管理器
		if config.AutoLoadPlugins {
			if err := client.pluginManager.Start(); err != nil {
				// 記錄錯誤但不阻止客戶端初始化
				if client.logger != nil {
					client.logger.WithError(err).Warn("插件系統啟動失敗")
				}
			}
		}
	}
	
	// 初始化安全管理器（T2-009）
	sessionID := fmt.Sprintf("session-%d", time.Now().Unix())
	client.securityManager = security.NewSecurityManager(config.Security, sessionID)

	client.initialized = true
	return client
}

// DefaultClientConfig 傳回預設的配置
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		CLITimeout:              60 * time.Second, // 增加到 60 秒以支援複雜任務
		CLIMaxRetries:           3,
		MaxHistorySize:          100,
		SaveDir:                 filepath.Join(".ralph-loop", "saves"),
		UseGobFormat:            false,
		CircuitBreakerThreshold: 3,
		SameErrorThreshold:      5,
		Model:                   "claude-sonnet-4.5",
		Silent:                  false,
		EnablePersistence:       true,
		EnableSDK:               true, // 預設啟用 SDK（主要執行方式）
		PreferSDK:               true, // 預設優先使用 SDK
		Verbose:                 false, // 預設不顯示詳細資訊
		Quiet:                   false, // 預設不安靜模式
		Security:                security.DefaultSecurityConfig(), // 預設安全配置
		
		// 性能優化預設配置（T2-012）
		EnableCaching:         true,                   // 啟用緩存
		CacheMaxSize:          1000,                   // 緩存最多 1000 個回應
		CacheTTL:              30 * time.Minute,       // 30 分鐘 TTL
		EnableConcurrency:     false,                  // 預設不啟用併發（用戶需要明確啟用）
		MaxConcurrentWorkers:  runtime.NumCPU(),       // 預設使用 CPU 核心數
		EnableMemoryPool:      true,                   // 啟用記憶體池
		MemoryOptimization:    true,                   // 啟用記憶體優化
		
		// 插件系統預設配置（T2-011）
		EnablePluginSystem:    false,                  // 預設不啟用插件系統（實驗性功能）
		PluginDir:             "./plugins",            // 插件目錄
		AutoLoadPlugins:       false,                  // 預設不自動載入插件
		PreferredExecutor:     "",                     // 無首選執行器

		// Promise Detection 預設配置
		PromisePhrase:       DefaultPromisePhrase,     // 預設承諾詞
		EnablePromiseDetect: true,                     // 預設啟用
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
		// 記錄熔斷器觸發
		c.metricsCollector.GetLoopMetrics().CircuitBreakerTrips.Inc()
		c.logger.WithFields(map[string]interface{}{
			"circuit_breaker_state": c.breaker.GetState(),
		}).Error("熔斷器已打開，停止執行")
		return nil, fmt.Errorf("circuit breaker is open: %s", c.breaker.GetState())
	}

	// 開始新迴圈
	loopIndex := len(c.contextManager.GetLoopHistory())
	execCtx := c.contextManager.StartLoop(loopIndex, prompt)

	// 記錄迴圈開始指標
	c.metricsCollector.GetLoopMetrics().TotalLoops.Inc()
	c.logger.WithFields(map[string]interface{}{
		"loop_index": loopIndex,
		"prompt":     prompt,
	}).Info("開始執行迴圈")
	
	stopTimer := c.metricsCollector.GetLoopMetrics().LoopExecutionTime.Start()

	defer func() {
		// 完成迴圈
		if err := c.contextManager.FinishLoop(); err != nil {
			// 日誌記錄
			c.logger.WithError(err).Warn("完成迴圈時發生錯誤")
		}

		// 自動持久化整個 ContextManager（如果啟用）
		if c.persistence != nil && c.config.EnablePersistence {
			if err := c.persistence.SaveContextManager(c.contextManager); err != nil {
				// 記錄但不影響主流程
				c.logger.WithError(err).Warn("持久化 ContextManager 失敗")
			}
		}
	}()

	// 使用執行模式選擇器決定執行方式（T-006）
	task := NewTask(fmt.Sprintf("loop-%d", loopIndex), prompt)
	task.WithComplexity(ComplexityMedium)  // 預設中等複雜度
	selectedMode := c.modeSelector.Choose(task)

	// 根據選擇的模式執行，並使用 RetryExecutor（T-008）
	var output string
	var executionErr error
	var usedSDK bool

	startTime := time.Now()
	
	// 使用 RetryExecutor 包裝執行邏輯
	result := c.retryExecutor.ExecuteWithResult(ctx, func() (interface{}, error) {
		switch selectedMode {
		case ModeSDK:
			// 嘗試使用 SDK，如果失敗則降級到 CLI
			if c.config.EnableSDK && c.sdkExecutor != nil && c.sdkExecutor.isHealthy() {
				sdkStart := time.Now()
				out, err := c.executeSecurely(ctx, prompt, func(ctx context.Context, p string) (string, error) {
					return c.sdkExecutor.Complete(ctx, p)
				})
				sdkDuration := time.Since(sdkStart)
				
				if err == nil {
					usedSDK = true
					c.metricsCollector.GetLoopMetrics().SDKExecutions.Inc()
					c.metricsCollector.GetLoopMetrics().SDKExecutionTime.Record(sdkDuration)
					c.logger.WithDuration(sdkDuration).Debug("SDK 執行成功")
					return out, nil
				}
				
				c.logger.WithError(err).WithDuration(sdkDuration).Warn("SDK 執行失敗，降級到 CLI")
				// SDK 失敗，檢測故障並嘗試恢復（T-007）
				c.detectAndRecover(ctx, err, time.Since(startTime))
			}
			
			// SDK 不可用或失敗，降級到 CLI 執行器
			cliStart := time.Now()
			output, err := c.executeSecurely(ctx, prompt, func(ctx context.Context, p string) (string, error) {
				result, execErr := c.executor.ExecutePrompt(ctx, p)
				if execErr != nil {
					return "", execErr
				}
				if result.ExitCode != 0 {
					return "", fmt.Errorf("CLI execution failed with exit code %d: %s", 
						result.ExitCode, result.Stderr)
				}
				return result.Stdout, nil
			})
			cliDuration := time.Since(cliStart)
			
			if err != nil {
				c.logger.WithError(err).WithDuration(cliDuration).Error("CLI 執行失敗")
				return nil, fmt.Errorf("both SDK and CLI execution failed: %w", err)
			}
			
			c.metricsCollector.GetLoopMetrics().CLIExecutions.Inc()
			c.metricsCollector.GetLoopMetrics().CLIExecutionTime.Record(cliDuration)
			c.logger.WithDuration(cliDuration).Debug("CLI 執行成功")
			return output, nil

		case ModeCLI:
			// 使用 CLI
			cliStart := time.Now()
			output, err := c.executeSecurely(ctx, prompt, func(ctx context.Context, p string) (string, error) {
				result, execErr := c.executor.ExecutePrompt(ctx, p)
				if execErr != nil {
					return "", execErr
				}
				if result.ExitCode != 0 {
					return "", fmt.Errorf("CLI execution failed with exit code %d: %s", 
						result.ExitCode, result.Stderr)
				}
				return result.Stdout, nil
			})
			cliDuration := time.Since(cliStart)
			
			if err != nil {
				c.logger.WithError(err).WithDuration(cliDuration).Error("CLI 執行失敗")
				return nil, err
			}
			
			c.metricsCollector.GetLoopMetrics().CLIExecutions.Inc()
			c.metricsCollector.GetLoopMetrics().CLIExecutionTime.Record(cliDuration)
			c.logger.WithDuration(cliDuration).Debug("CLI 執行成功")
			return output, nil

		case ModePlugin:
			// 使用插件執行器
			if c.config.EnablePluginSystem && c.pluginManager != nil {
				pluginName := c.modeSelector.GetPreferredPlugin()
				if pluginName == "" {
					pluginName = c.config.PreferredExecutor
				}
				
				pluginStart := time.Now()
				output, err := c.executeSecurely(ctx, prompt, func(ctx context.Context, p string) (string, error) {
					return c.executeWithPlugin(ctx, pluginName, p)
				})
				pluginDuration := time.Since(pluginStart)
				
				if err == nil {
					// 成功執行插件
					c.logger.WithField("plugin", pluginName).WithDuration(pluginDuration).Debug("插件執行成功")
					return output, nil
				}
				
				c.logger.WithError(err).WithField("plugin", pluginName).WithDuration(pluginDuration).Warn("插件執行失敗，降級到 SDK/CLI")
				// 插件失敗，檢測故障並嘗試恢復
				c.detectAndRecover(ctx, err, time.Since(startTime))
			}
			
			// 插件不可用或失敗，降級到 SDK 或 CLI
			if c.config.EnableSDK && c.sdkExecutor != nil && c.sdkExecutor.isHealthy() {
				sdkStart := time.Now()
				out, err := c.executeSecurely(ctx, prompt, func(ctx context.Context, p string) (string, error) {
					return c.sdkExecutor.Complete(ctx, p)
				})
				sdkDuration := time.Since(sdkStart)
				
				if err == nil {
					usedSDK = true
					c.metricsCollector.GetLoopMetrics().SDKExecutions.Inc()
					c.metricsCollector.GetLoopMetrics().SDKExecutionTime.Record(sdkDuration)
					c.logger.WithDuration(sdkDuration).Debug("SDK 執行成功（插件降級）")
					return out, nil
				}
				
				c.logger.WithError(err).WithDuration(sdkDuration).Warn("SDK 執行失敗，進一步降級到 CLI")
			}
			
			// 最後降級到 CLI
			cliStart := time.Now()
			output, err := c.executeSecurely(ctx, prompt, func(ctx context.Context, p string) (string, error) {
				result, execErr := c.executor.ExecutePrompt(ctx, p)
				if execErr != nil {
					return "", execErr
				}
				if result.ExitCode != 0 {
					return "", fmt.Errorf("CLI execution failed with exit code %d: %s", 
						result.ExitCode, result.Stderr)
				}
				return result.Stdout, nil
			})
			cliDuration := time.Since(cliStart)
			
			if err != nil {
				c.logger.WithError(err).WithDuration(cliDuration).Error("CLI 執行失敗")
				return nil, fmt.Errorf("plugin, SDK, and CLI execution all failed: %w", err)
			}
			
			c.metricsCollector.GetLoopMetrics().CLIExecutions.Inc()
			c.metricsCollector.GetLoopMetrics().CLIExecutionTime.Record(cliDuration)
			c.logger.WithDuration(cliDuration).Debug("CLI 執行成功（插件/SDK 降級）")
			return output, nil

		case ModeAuto, ModeHybrid:
			// 自動模式：優先 SDK，失敗則 CLI
			if c.config.PreferSDK && c.config.EnableSDK && c.sdkExecutor != nil && c.sdkExecutor.isHealthy() {
				sdkStart := time.Now()
				out, err := c.executeSecurely(ctx, prompt, func(ctx context.Context, p string) (string, error) {
					return c.sdkExecutor.Complete(ctx, p)
				})
				sdkDuration := time.Since(sdkStart)
				
				if err == nil {
					usedSDK = true
					c.metricsCollector.GetLoopMetrics().SDKExecutions.Inc()
					c.metricsCollector.GetLoopMetrics().SDKExecutionTime.Record(sdkDuration)
					c.logger.WithDuration(sdkDuration).Debug("SDK 執行成功")
					return out, nil
				}
				
				c.logger.WithError(err).WithDuration(sdkDuration).Warn("SDK 執行失敗，降級到 CLI")
				// SDK 失敗，檢測故障並嘗試恢復
				c.detectAndRecover(ctx, err, time.Since(startTime))
			}
			
			// 使用 CLI
			cliStart := time.Now()
			output, err := c.executeSecurely(ctx, prompt, func(ctx context.Context, p string) (string, error) {
				result, execErr := c.executor.ExecutePrompt(ctx, p)
				if execErr != nil {
					return "", execErr
				}
				if result.ExitCode != 0 {
					return "", fmt.Errorf("CLI execution failed with exit code %d: %s", 
						result.ExitCode, result.Stderr)
				}
				return result.Stdout, nil
			})
			cliDuration := time.Since(cliStart)
			
			if err != nil {
				c.logger.WithError(err).WithDuration(cliDuration).Error("CLI 執行失敗")
				return nil, err
			}
			
			c.metricsCollector.GetLoopMetrics().CLIExecutions.Inc()
			c.metricsCollector.GetLoopMetrics().CLIExecutionTime.Record(cliDuration)
			c.logger.WithDuration(cliDuration).Debug("CLI 執行成功")
			return output, nil

		default:
			return nil, fmt.Errorf("unknown execution mode: %v", selectedMode)
		}
	})

	// 處理執行結果
	if result.Error != nil {
		executionErr = result.Error
		c.breaker.RecordSameError(executionErr.Error())
		execCtx.ExitReason = fmt.Sprintf("執行失敗: %v (嘗試 %d 次)", executionErr, result.Attempts)
		
		// 使用新的錯誤結果方法
		return c.createErrorResult(execCtx, executionErr), nil
	}

	output = result.Value.(string)
	if usedSDK {
		execCtx.CLICommand = "sdk:complete"
	} else {
		execCtx.CLICommand = "cli:execute"
	}
	execCtx.CLIOutput = output
	execCtx.CLIExitCode = 0

	// 解析輸出
	parser := NewOutputParser(output)
	parser.Parse()
	codeBlocks := parser.ExtractCodeBlocks()
	options := parser.GetOptions()

	// 將 CodeBlock 轉換為字串
	var codeBlockStrings []string
	for _, block := range codeBlocks {
		codeBlockStrings = append(codeBlockStrings, block.Content)
	}

	execCtx.ParsedCodeBlocks = options
	execCtx.CleanedOutput = output

	// 使用完整的回應分析器
	analyzer := NewResponseAnalyzer(output)
	analyzer.CalculateCompletionScore()
	
	// 檢查是否卡住（同樣的錯誤重複出現）
	isStuck, stuckReason := analyzer.DetectStuckState()
	
	// 檢查是否完成
	// 優先使用 Promise Detection（參考 doggy8088/copilot-ralph）
	isCompleted := false
	if c.promiseDetector != nil {
		// 串流中可能已經偵測到，再對完整輸出做一次檢查
		c.promiseDetector.CheckFull(output)
		isCompleted = c.promiseDetector.IsDetected()
	}
	// 若 Promise Detection 未偵測到，回退到舊的雙重條件驗證
	if !isCompleted {
		isCompleted = analyzer.IsCompleted()
	}
	
	// 記錄到退出偵測器
	if analyzer.DetectTestOnlyLoop() {
		c.exitDetector.RecordTestOnlyLoop()
	}
	
	// 如果有完成信號，記錄到退出偵測器
	status := analyzer.ParseStructuredOutput()
	if status != nil && status.ExitSignal {
		c.exitDetector.RecordDoneSignal()
	}
	
	// 檢查退出偵測器的優雅退出條件
	analyzerScore := analyzer.CalculateCompletionScore()
	shouldExitGracefully := c.exitDetector.ShouldExitGracefully(analyzerScore)
	var exitReason string
	if shouldExitGracefully {
		exitReason = "graceful exit conditions met"
	}
	
	// 判斷是否應該繼續
	shouldContinue := !isCompleted && !isStuck && !shouldExitGracefully
	
	execCtx.ShouldContinue = shouldContinue
	execCtx.ParsedCodeBlocks = options
	execCtx.ExtractedCodeBlocks = codeBlockStrings  // 使用轉換後的字串
	
	// 根據分析結果更新熔斷器
	if isCompleted {
		c.breaker.RecordSuccess()
		execCtx.ExitReason = "task completed (dual condition verification)"
		// 記錄成功指標
		c.metricsCollector.GetLoopMetrics().SuccessfulLoops.Inc()
		stopTimer()
		c.logger.WithFields(map[string]interface{}{
			"loop_index": loopIndex,
			"reason":     execCtx.ExitReason,
		}).Info("迴圈執行成功")
	} else if shouldExitGracefully {
		c.breaker.RecordSuccess()  // 優雅退出也算成功
		execCtx.ExitReason = fmt.Sprintf("graceful exit: %s", exitReason)
		// 記錄成功指標
		c.metricsCollector.GetLoopMetrics().SuccessfulLoops.Inc()
		stopTimer()
		c.logger.WithFields(map[string]interface{}{
			"loop_index": loopIndex,
			"reason":     execCtx.ExitReason,
		}).Info("迴圈優雅退出")
	} else if isStuck {
		c.breaker.RecordNoProgress()
		execCtx.ExitReason = fmt.Sprintf("stuck state detected: %s", stuckReason)
		// 記錄失敗指標
		c.metricsCollector.GetLoopMetrics().FailedLoops.Inc()
		stopTimer()
		c.logger.WithFields(map[string]interface{}{
			"loop_index": loopIndex,
			"reason":     execCtx.ExitReason,
		}).Warn("迴圈執行失敗（卡住）")
	} else {
		// 有輸出變化表示有進展，即使未完成
		if len(output) > 0 {
			c.breaker.RecordSuccess()  // 有回應就算成功
			c.logger.WithFields(map[string]interface{}{
				"loop_index":    loopIndex,
				"output_length": len(output),
			}).Debug("迴圈有進展，繼續執行")
		} else {
			c.breaker.RecordNoProgress()
			c.logger.WithFields(map[string]interface{}{
				"loop_index": loopIndex,
			}).Warn("迴圈無進展")
		}
	}

	execCtx.CircuitBreakerState = string(c.breaker.GetState())

	// 個別執行上下文的持久化（可選）
	if c.persistence != nil && c.config.EnablePersistence {
		_ = c.persistence.SaveExecutionContext(execCtx)
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
			if c.uiCallback != nil {
				c.uiCallback.OnComplete(i, fmt.Errorf("context cancelled after %d loops", i))
			}
			return results, fmt.Errorf("context cancelled after %d loops", i)
		default:
		}

		// 每次迴圈重置 Promise Detector
		if c.promiseDetector != nil {
			c.promiseDetector.Reset()
		}

		// 構建帶有 System Prompt 的完整 prompt
		prompt := initialPrompt
		if c.config.EnablePromiseDetect {
			prompt = WrapPromptWithSystemInstructions(
				initialPrompt,
				c.config.PromisePhrase,
				i+1,
				maxLoops,
			)
		}

		// UI 回調：迴圈開始
		if c.uiCallback != nil {
			c.uiCallback.OnLoopStart(i+1, maxLoops)
		}

		result, err := c.ExecuteLoop(ctx, prompt)
		if err != nil {
			// UI 回調：錯誤
			if c.uiCallback != nil {
				c.uiCallback.OnError(err)
			}
			return results, err
		}

		results = append(results, result)

		// UI 回調：迴圈完成
		if c.uiCallback != nil {
			c.uiCallback.OnLoopComplete(i+1, result)
		}

		// 檢查是否完成或失敗
		if !result.ShouldContinue {
			if result.IsFailed() {
				// 執行失敗，返回錯誤
				if c.uiCallback != nil {
					c.uiCallback.OnError(result.Error)
					c.uiCallback.OnComplete(i+1, result.Error)
				}
				return results, result.Error
			} else {
				// 任務完成，正常結束
				if c.uiCallback != nil {
					c.uiCallback.OnComplete(i+1, nil)
				}
				return results, nil
			}
		}

		// 檢查熔斷器
		if c.breaker.IsOpen() {
			// 記錄熔斷器觸發
			c.metricsCollector.GetLoopMetrics().CircuitBreakerTrips.Inc()
			c.logger.WithFields(map[string]interface{}{
				"loop_count":            i+1,
				"circuit_breaker_state": c.breaker.GetState(),
			}).Error("熔斷器在迴圈中被觸發")
			
			err := WrapError(ErrorTypeCircuitOpen, fmt.Sprintf("circuit breaker opened after %d loops", i+1), nil)
			if c.uiCallback != nil {
				c.uiCallback.OnError(err)
				c.uiCallback.OnComplete(i+1, err)
			}
			return results, err
		}
	}

	err := WrapError(ErrorTypeRetryExhausted, fmt.Sprintf("reached maximum loops (%d) without completion", maxLoops), nil)
	if c.uiCallback != nil {
		c.uiCallback.OnWarning(fmt.Sprintf("已達到最大迴圈數 (%d)", maxLoops))
		c.uiCallback.OnComplete(maxLoops, err)
	}
	return results, err
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

// CheckSDKHealth 檢查 SDK 執行器的健康狀況
func (c *RalphLoopClient) CheckSDKHealth() map[string]string {
	if !c.initialized {
		return map[string]string{
			"status": "未初始化",
			"error":  "客戶端未初始化",
		}
	}

	// 使用更短的超時來避免卡住
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 使用 goroutine 和 channel 來避免 deadlock
	result := make(chan map[string]string, 1)
	
	go func() {
		defer func() {
			if r := recover(); r != nil {
				result <- map[string]string{
					"version":    "v0.1.23",
					"status":     "錯誤",
					"connection": "崩潰",
					"error":      fmt.Sprintf("SDK 測試時發生 panic: %v", r),
				}
			}
		}()

		// 創建臨時 SDK 執行器進行測試
		sdkConfig := DefaultSDKConfig()
		sdkConfig.Timeout = 3 * time.Second // 更短的超時
		sdkExecutor := NewSDKExecutor(sdkConfig)

		// 嘗試啟動 SDK
		err := sdkExecutor.Start(ctx)
		if err != nil {
			result <- map[string]string{
				"version":    "v0.1.23",
				"status":     "不可用",
				"connection": "失敗",
				"error":      err.Error(),
			}
			return
		}

		// 清理
		_ = sdkExecutor.Close()

		result <- map[string]string{
			"version":    "v0.1.23",
			"status":     "正常",
			"connection": "已連接",
			"error":      "",
		}
	}()

	// 等待結果或超時
	select {
	case res := <-result:
		return res
	case <-ctx.Done():
		return map[string]string{
			"version":    "v0.1.23",
			"status":     "超時",
			"connection": "超時",
			"error":      "SDK 健康檢查超時",
		}
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

// detectAndRecover 檢測故障並嘗試恢復（T-007）
//
// 此方法使用 FailureDetector 檢測故障類型，
// 並根據 RecoveryStrategy 嘗試恢復。
//
// 參數:
// - ctx: 執行上下文
// - err: 發生的錯誤
// - duration: 執行時長
func (c *RalphLoopClient) detectAndRecover(ctx context.Context, err error, duration time.Duration) {
	if err == nil {
		return
	}

	// 使用故障檢測器識別故障類型
	var detectedFailure FailureType = FailureNone
	for _, detector := range c.failureDetectors {
		if detector.Detect(err, duration) {
			detectedFailure = detector.GetType()
			break
		}
	}

	// 如果檢測到故障，按優先級嘗試恢復策略
	if detectedFailure != FailureNone {
		for _, strategy := range c.recoveryStrategies {
			recoveryErr := strategy.Recover(ctx, err)
			if recoveryErr == nil {
				// 恢復成功，重置檢測器
				for _, detector := range c.failureDetectors {
					detector.Reset()
				}
				return
			}
		}
	}
}

// Close 關閉客戶端並清理資源
func (c *RalphLoopClient) Close() error {
	if c.closed {
		return fmt.Errorf("client already closed") // 返回錯誤而不是 nil
	}

	c.closed = true

	var errors []error

	// 停止併發執行管理器 (from client_performance.go)
	if c.concurrentManager != nil {
		c.concurrentManager.Stop()
	}

	// 關閉緩存管理器 (from client_performance.go)
	if c.cacheManager != nil {
		c.cacheManager.Close()
	}

	// 關閉 SDK 執行器
	if c.sdkExecutor != nil {
		if err := c.sdkExecutor.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close SDK executor: %w", err))
		}
	}

	// 執行持久化 (from client.go 原始邏輯)
	if c.persistence != nil && c.config.EnablePersistence {
		if err := c.SaveHistoryToDisk(); err != nil {
			errors = append(errors, fmt.Errorf("failed to save state: %w", err))
		}
	}

	// 記憶體優化：強制執行垃圾回收 (from client_performance.go)
	if c.config.MemoryOptimization {
		runtime.GC()
	}

	// 如果有錯誤，合併返回
	if len(errors) > 0 {
		errorMessages := make([]string, len(errors))
		for i, err := range errors {
			errorMessages[i] = err.Error()
		}
		return fmt.Errorf("errors during close: %s", strings.Join(errorMessages, "; "))
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
		Error:           nil,            // 預設無錯誤
		IsSuccess:       !shouldContinue, // 如果不應繼續且無錯誤，表示成功完成
		Timestamp:       execCtx.Timestamp,
	}
}

// createErrorResult 建立錯誤結果
func (c *RalphLoopClient) createErrorResult(execCtx *ExecutionContext, err error) *LoopResult {
	// 包裝為 RalphLoopError（如果還不是）
	var ralphErr *RalphLoopError
	if !errors.As(err, &ralphErr) {
		ralphErr = WrapError(ErrorTypeExecutionError, "execution failed", err)
	}
	
	return &LoopResult{
		LoopID:          execCtx.LoopID,
		LoopIndex:       execCtx.LoopIndex,
		ShouldContinue:  false,         // 錯誤時不應繼續
		CompletionScore: 0,
		Output:          execCtx.CLIOutput,
		ExitReason:      ralphErr.Error(),
		Error:           ralphErr,      // 明確設定錯誤
		IsSuccess:       false,         // 明確標記為失敗
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
	Error           error  // 新增：明確的錯誤欄位
	IsSuccess       bool   // 新增：明確的成功狀態
	Timestamp       time.Time
}

// IsCompleted 檢查迴圈是否因為任務完成而結束 (非錯誤)
func (r *LoopResult) IsCompleted() bool {
	return !r.ShouldContinue && r.Error == nil
}

// IsFailed 檢查迴圈是否因為錯誤而結束
func (r *LoopResult) IsFailed() bool {
	return !r.ShouldContinue && r.Error != nil
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

// WithSandboxMode 啟用沙箱模式
func (b *ClientBuilder) WithSandboxMode(allowedCommands []string) *ClientBuilder {
	b.config.Security.SandboxMode = true
	b.config.Security.AllowedCommands = allowedCommands
	return b
}

// WithAuditLog 啟用審計日誌
func (b *ClientBuilder) WithAuditLog(logDir string) *ClientBuilder {
	b.config.Security.EnableAuditLog = true
	if logDir != "" {
		b.config.Security.AuditLogDir = logDir
	}
	return b
}

// WithCredentialEncryption 啟用憑證加密
func (b *ClientBuilder) WithCredentialEncryption(password string) *ClientBuilder {
	b.config.Security.EncryptCredentials = true
	if password != "" {
		b.config.Security.EncryptionPassword = password
	}
	return b
}

// WithSecurityConfig 設定完整的安全配置
func (b *ClientBuilder) WithSecurityConfig(securityConfig security.SecurityConfig) *ClientBuilder {
	b.config.Security = securityConfig
	return b
}

// Build 建立客戶端
func (b *ClientBuilder) Build() *RalphLoopClient {
	return NewRalphLoopClientWithConfig(b.config)
}

// SetUICallback 設置 UI 回調介面
//
// 允許自訂 UI 回調以控制如何顯示進度、錯誤和完成訊息。
// 如果傳入 nil，則使用預設的 UI 回調。
//
// 參數:
// - callback: 自訂的 UI 回調介面實作
//
// 範例:
//
//	customCallback := &MyCustomUICallback{}
//	client.SetUICallback(customCallback)
func (c *RalphLoopClient) SetUICallback(callback UICallback) {
	if callback == nil {
		// 使用預設回調
		c.uiCallback = NewDefaultUICallback(c.config.Verbose, c.config.Quiet)
	} else {
		c.uiCallback = callback
	}
	
	// 更新 CLI 執行器的串流回調
	if c.executor != nil {
		c.executor.SetStreamCallback(
			func(line string) {
				if c.uiCallback != nil {
					c.uiCallback.OnStreamOutput(line)
				}
			},
			func(line string) {
				if c.uiCallback != nil {
					c.uiCallback.OnStreamError(line)
				}
			},
		)
	}
}

// GetUICallback 取得當前的 UI 回調
func (c *RalphLoopClient) GetUICallback() UICallback {
	return c.uiCallback
}

// Security related methods (T2-009)

// executeSecurely 安全地執行 prompt，包含所有安全檢查
func (c *RalphLoopClient) executeSecurely(ctx context.Context, prompt string, executor func(context.Context, string) (string, error)) (string, error) {
	// 安全驗證（如果啟用）
	if c.securityManager != nil {
		// 將 prompt 當作偽命令進行驗證
		fakeCommand := fmt.Sprintf("copilot -p \"%s\"", prompt)
		if err := c.securityManager.ValidateCommand(fakeCommand); err != nil {
			return "", fmt.Errorf("安全檢查失敗: %w", err)
		}
	}
	
	// 執行命令
	output, err := executor(ctx, prompt)
	
	// 遮罩輸出中的敏感資訊
	if c.securityManager != nil {
		output = c.securityManager.MaskSensitiveOutput(output)
	}
	
	return output, err
}

// GetSecurityStatus 獲取安全狀態
func (c *RalphLoopClient) GetSecurityStatus() map[string]interface{} {
	if c.securityManager == nil {
		return map[string]interface{}{
			"security_enabled": false,
		}
	}
	
	status := c.securityManager.GetSecurityStatus()
	status["security_enabled"] = true
	return status
}

// EnableSandboxMode 啟用沙箱模式
func (c *RalphLoopClient) EnableSandboxMode(allowedCommands []string) error {
	if c.securityManager == nil {
		return fmt.Errorf("security manager not initialized")
	}
	
	c.config.Security.SandboxMode = true
	c.config.Security.AllowedCommands = allowedCommands
	
	// 重新創建安全管理器以應用新設置
	sessionID := fmt.Sprintf("session-%d", time.Now().Unix())
	c.securityManager = security.NewSecurityManager(c.config.Security, sessionID)
	
	return nil
}

// DisableSandboxMode 禁用沙箱模式
func (c *RalphLoopClient) DisableSandboxMode() error {
	if c.securityManager == nil {
		return fmt.Errorf("security manager not initialized")
	}
	
	c.config.Security.SandboxMode = false
	
	// 重新創建安全管理器
	sessionID := fmt.Sprintf("session-%d", time.Now().Unix())
	c.securityManager = security.NewSecurityManager(c.config.Security, sessionID)
	
	return nil
}

// EnableAuditLog 啟用審計日誌
func (c *RalphLoopClient) EnableAuditLog(logDir string) error {
	if c.securityManager == nil {
		return fmt.Errorf("security manager not initialized")
	}
	
	c.config.Security.EnableAuditLog = true
	if logDir != "" {
		c.config.Security.AuditLogDir = logDir
	}
	
	// 重新創建安全管理器以應用新設置
	sessionID := fmt.Sprintf("session-%d", time.Now().Unix())
	c.securityManager = security.NewSecurityManager(c.config.Security, sessionID)
	
	return nil
}

// DisableAuditLog 禁用審計日誌
func (c *RalphLoopClient) DisableAuditLog() error {
	if c.securityManager == nil {
		return fmt.Errorf("security manager not initialized")
	}
	
	c.config.Security.EnableAuditLog = false
	
	// 重新創建安全管理器
	sessionID := fmt.Sprintf("session-%d", time.Now().Unix())
	c.securityManager = security.NewSecurityManager(c.config.Security, sessionID)
	
	return nil
}

// EnableCredentialEncryption 啟用憑證加密
func (c *RalphLoopClient) EnableCredentialEncryption(password string) error {
	if c.securityManager == nil {
		return fmt.Errorf("security manager not initialized")
	}
	
	c.config.Security.EncryptCredentials = true
	if password != "" {
		c.config.Security.EncryptionPassword = password
	}
	
	// 重新創建安全管理器以應用新設置
	sessionID := fmt.Sprintf("session-%d", time.Now().Unix())
	c.securityManager = security.NewSecurityManager(c.config.Security, sessionID)
	
	return nil
}

// DisableCredentialEncryption 禁用憑證加密
func (c *RalphLoopClient) DisableCredentialEncryption() error {
	if c.securityManager == nil {
		return fmt.Errorf("security manager not initialized")
	}
	
	c.config.Security.EncryptCredentials = false
	c.config.Security.EncryptionPassword = ""
	
	// 重新創建安全管理器
	sessionID := fmt.Sprintf("session-%d", time.Now().Unix())
	c.securityManager = security.NewSecurityManager(c.config.Security, sessionID)
	
	return nil
}

// executeWithPlugin 使用插件執行器執行 prompt
//
// 此方法查找指定的插件並使用它來處理 prompt。
// 如果未指定插件名稱，將嘗試使用第一個可用的執行器插件。
//
// 參數:
// - ctx: 執行上下文
// - pluginName: 插件名稱（空字串表示使用第一個可用插件）
// - prompt: 要執行的 prompt
//
// 返回值:
// - string: 插件的輸出結果
// - error: 執行過程中的錯誤
func (c *RalphLoopClient) executeWithPlugin(ctx context.Context, pluginName string, prompt string) (string, error) {
	if c.pluginManager == nil {
		return "", fmt.Errorf("plugin manager not initialized")
	}

	// 獲取所有載入的插件名稱
	pluginNames := c.pluginManager.ListPlugins()
	if len(pluginNames) == 0 {
		return "", fmt.Errorf("no plugins loaded")
	}

	var targetPlugin Plugin
	
	// 如果指定了插件名稱，查找特定插件
	if pluginName != "" {
		plugin, err := c.pluginManager.GetPlugin(pluginName)
		if err != nil {
			return "", fmt.Errorf("plugin '%s' not found: %w", pluginName, err)
		}
		targetPlugin = plugin
	} else {
		// 未指定插件名稱，使用第一個執行器插件
		for _, name := range pluginNames {
			plugin, err := c.pluginManager.GetPlugin(name)
			if err != nil {
				continue
			}
			metadata := plugin.GetMetadata()
			if metadata != nil && metadata.Type == "executor" {
				targetPlugin = plugin
				break
			}
		}
		if targetPlugin == nil {
			return "", fmt.Errorf("no executor plugins available")
		}
	}

	// 檢查插件是否為執行器插件
	executorPlugin, ok := targetPlugin.(ExecutorPlugin)
	if !ok {
		return "", fmt.Errorf("plugin '%s' is not an executor plugin", targetPlugin.GetMetadata().Name)
	}

	// 使用插件執行 prompt
	c.logger.WithField("plugin", targetPlugin.GetMetadata().Name).Debug("使用插件執行 prompt")
	
	// 準備插件執行選項
	options := PluginExecutorOptions{
		Model:       string(c.config.Model),
		Temperature: 0.7,
		MaxTokens:   4096,
		Stream:      false,
		Context:     make(map[string]interface{}),
		Timeout:     c.config.CLITimeout,
	}
	
	result, err := executorPlugin.Execute(ctx, prompt, options)
	if err != nil {
		return "", fmt.Errorf("plugin execution failed: %w", err)
	}

	// 提取輸出文字
	if result == nil || result.Content == "" {
		return "", fmt.Errorf("plugin returned empty result")
	}

	return result.Content, nil
}

// 插件管理相關方法

// LoadPlugin 載入指定的插件
//
// 此方法動態載入一個插件並將其註冊到插件管理器中。
//
// 參數:
// - pluginPath: 插件檔案路徑
//
// 返回值:
// - error: 載入過程中的錯誤
func (c *RalphLoopClient) LoadPlugin(pluginPath string) error {
	if !c.initialized {
		return fmt.Errorf("client not initialized")
	}
	if c.pluginManager == nil {
		return fmt.Errorf("plugin system not enabled")
	}

	// TODO: 實作從路徑載入插件的邏輯
	return fmt.Errorf("LoadPlugin from path not yet implemented")
}

// UnloadPlugin 卸載指定的插件
//
// 此方法從插件管理器中移除一個插件。
//
// 參數:
// - pluginName: 插件名稱
//
// 返回值:
// - error: 卸載過程中的錯誤
func (c *RalphLoopClient) UnloadPlugin(pluginName string) error {
	if !c.initialized {
		return fmt.Errorf("client not initialized")
	}
	if c.pluginManager == nil {
		return fmt.Errorf("plugin system not enabled")
	}

	return c.pluginManager.UnloadPlugin(pluginName)
}

// ListPlugins 列出所有已載入的插件
//
// 返回值:
// - []Plugin: 已載入的插件列表
func (c *RalphLoopClient) ListPlugins() []Plugin {
	if c.pluginManager == nil {
		return nil
	}

	// 獲取所有插件名稱並轉換為 Plugin 列表
	pluginNames := c.pluginManager.ListPlugins()
	plugins := make([]Plugin, 0, len(pluginNames))
	
	for _, name := range pluginNames {
		plugin, err := c.pluginManager.GetPlugin(name)
		if err == nil {
			plugins = append(plugins, plugin)
		}
	}
	
	return plugins
}

// GetPlugin 獲取指定的插件
//
// 參數:
// - pluginName: 插件名稱
//
// 返回值:
// - Plugin: 插件實例
// - error: 獲取過程中的錯誤
func (c *RalphLoopClient) GetPlugin(pluginName string) (Plugin, error) {
	if c.pluginManager == nil {
		return nil, fmt.Errorf("plugin system not enabled")
	}

	return c.pluginManager.GetPlugin(pluginName)
}

// GetPluginStatus 獲取插件系統狀態
//
// 返回值:
// - map[string]interface{}: 插件系統狀態信息
func (c *RalphLoopClient) GetPluginStatus() map[string]interface{} {
	if c.pluginManager == nil {
		return map[string]interface{}{
			"enabled": false,
			"error":   "plugin system not enabled",
		}
	}

	plugins := c.pluginManager.ListPlugins()
	status := map[string]interface{}{
		"enabled":       true,
		"plugin_count":  len(plugins),
		"plugin_dir":    c.config.PluginDir,
		"auto_load":     c.config.AutoLoadPlugins,
		"plugins":       make([]map[string]interface{}, 0, len(plugins)),
	}

	for _, pluginName := range plugins {
		plugin, err := c.pluginManager.GetPlugin(pluginName)
		if err != nil {
			continue
		}
		metadata := plugin.GetMetadata()
		pluginInfo := map[string]interface{}{
			"name":        metadata.Name,
			"version":     metadata.Version,
			"author":      metadata.Author,
			"description": metadata.Description,
			"type":        metadata.Type,
			"healthy":     plugin.IsHealthy(),
		}
		status["plugins"] = append(status["plugins"].([]map[string]interface{}), pluginInfo)
	}

	return status
}

// EnablePluginAutoLoad 啟用插件自動載入
//
// 此方法會啟用插件自動載入功能，並掃描插件目錄載入所有可用插件。
//
// 返回值:
// - error: 啟用過程中的錯誤
func (c *RalphLoopClient) EnablePluginAutoLoad() error {
	if !c.initialized {
		return fmt.Errorf("client not initialized")
	}
	if c.pluginManager == nil {
		return fmt.Errorf("plugin system not enabled")
	}

	c.config.AutoLoadPlugins = true
	return c.pluginManager.Start()
}

// DisablePluginAutoLoad 禁用插件自動載入
//
// 此方法會禁用插件自動載入功能，但不會卸載已載入的插件。
//
// 返回值:
// - error: 禁用過程中的錯誤
func (c *RalphLoopClient) DisablePluginAutoLoad() error {
	if c.pluginManager == nil {
		return fmt.Errorf("plugin system not enabled")
	}

	c.config.AutoLoadPlugins = false
	return nil
}

// SetPreferredPlugin 設定偏好的插件
//
// 此方法設定執行模式選擇器偏好使用的插件。
//
// 參數:
// - pluginName: 插件名稱
//
// 返回值:
// - error: 設定過程中的錯誤
func (c *RalphLoopClient) SetPreferredPlugin(pluginName string) error {
	if !c.initialized {
		return fmt.Errorf("client not initialized")
	}

	// 驗證插件是否存在
	if c.pluginManager != nil {
		_, err := c.pluginManager.GetPlugin(pluginName)
		if err != nil {
			return fmt.Errorf("plugin '%s' not found: %w", pluginName, err)
		}
	}

	c.config.PreferredExecutor = pluginName
	c.modeSelector.SetPreferredPlugin(pluginName)
	c.modeSelector.SetPluginAvailable(true)
	
	return nil
}

// GetPreferredPlugin 獲取偏好的插件
//
// 返回值:
// - string: 偏好插件的名稱
func (c *RalphLoopClient) GetPreferredPlugin() string {
	return c.config.PreferredExecutor
}

