package ghcopilot

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Model 定義可用的 AI 模型
type Model string

const (
	ModelClaudeSonnet45 Model = "claude-sonnet-4.5" // 預設模型
	ModelClaudeHaiku45  Model = "claude-haiku-4.5"
	ModelClaudeOpus45   Model = "claude-opus-4.5"
	ModelClaudeSonnet4  Model = "claude-sonnet-4"
	ModelGPT52Codex     Model = "gpt-5.2-codex"
	ModelGPT51CodexMax  Model = "gpt-5.1-codex-max"
	ModelGPT51Codex     Model = "gpt-5.1-codex"
	ModelGPT52          Model = "gpt-5.2"
	ModelGPT51          Model = "gpt-5.1"
	ModelGPT5           Model = "gpt-5"
	ModelGPT51CodexMini Model = "gpt-5.1-codex-mini"
	ModelGPT5Mini       Model = "gpt-5-mini"
	ModelGPT41          Model = "gpt-4.1"
	ModelGemini3Pro     Model = "gemini-3-pro-preview"
)

// ExecutionResult 代表 CLI 執行的結果
type ExecutionResult struct {
	Command       string        // 執行的指令
	Stdout        string        // 標準輸出
	Stderr        string        // 標準錯誤
	ExitCode      int           // 退出碼
	ExecutionTime time.Duration // 執行時間
	Success       bool          // 是否成功執行
	Error         error         // 任何執行錯誤
	Model         Model         // 使用的模型
}

// ExecutorOptions 定義執行選項
type ExecutorOptions struct {
	Model           Model    // AI 模型
	Silent          bool     // 安靜模式（只輸出結果）
	AllowAllTools   bool     // 允許所有工具自動執行
	AllowAllPaths   bool     // 允許存取所有路徑
	AllowAllURLs    bool     // 允許存取所有 URL
	AllowedTools    []string // 允許的工具列表
	DeniedTools     []string // 禁止的工具列表
	AllowedDirs     []string // 允許存取的目錄
	NoAskUser       bool     // 禁用詢問用戶（自主模式）
	DisableParallel bool     // 禁用平行工具執行
	SessionID       string   // 用於 resume 的 session ID
	SharePath       string   // 分享 session 到檔案
}

// DefaultOptions 回傳預設選項
func DefaultOptions() ExecutorOptions {
	return ExecutorOptions{
		Model:         ModelClaudeSonnet45,
		Silent:        true, // 預設安靜模式，減少輸出
		AllowAllTools: true, // 預設允許所有工具，適合自動化
		AllowAllPaths: true, // 預設允許所有路徑存取，避免 shell 工具被拒
		AllowAllURLs:  true, // 預設允許所有 URL 存取，避免網路工具被拒
		NoAskUser:     true, // 預設自主模式
	}
}

// CLIExecutor 用於執行 GitHub Copilot CLI 指令
type CLIExecutor struct {
	timeout          time.Duration
	workDir          string
	maxRetries       int
	retryDelay       time.Duration
	requestID        string
	telemetryEnabled bool
	options          ExecutorOptions
}

// NewCLIExecutor 建立新的 CLI 執行器
func NewCLIExecutor(workDir string) *CLIExecutor {
	return &CLIExecutor{
		timeout:          60 * time.Second, // 增加到 60 秒以支援複雜任務
		workDir:          workDir,
		maxRetries:       3,
		retryDelay:       1 * time.Second,
		requestID:        generateRequestID(),
		telemetryEnabled: true,
		options:          DefaultOptions(),
	}
}

// NewCLIExecutorWithOptions 建立帶選項的 CLI 執行器
func NewCLIExecutorWithOptions(workDir string, options ExecutorOptions) *CLIExecutor {
	return &CLIExecutor{
		timeout:          60 * time.Second, // 增加到 60 秒以支援複雜任務
		workDir:          workDir,
		maxRetries:       3,
		retryDelay:       1 * time.Second,
		requestID:        generateRequestID(),
		telemetryEnabled: true,
		options:          options,
	}
}

// SetOptions 設定執行選項
func (ce *CLIExecutor) SetOptions(options ExecutorOptions) {
	ce.options = options
}

// SetModel 設定使用的 AI 模型
func (ce *CLIExecutor) SetModel(model Model) {
	ce.options.Model = model
}

// SetSilent 設定安靜模式
func (ce *CLIExecutor) SetSilent(silent bool) {
	ce.options.Silent = silent
}

// SetAllowAllTools 設定是否允許所有工具
func (ce *CLIExecutor) SetAllowAllTools(allow bool) {
	ce.options.AllowAllTools = allow
}

// SetTimeout 設定執行逾時
func (ce *CLIExecutor) SetTimeout(duration time.Duration) {
	ce.timeout = duration
}

// SetMaxRetries 設定最大重試次數
func (ce *CLIExecutor) SetMaxRetries(retries int) {
	ce.maxRetries = retries
}

// buildArgs 根據選項構建 CLI 參數
func (ce *CLIExecutor) buildArgs(prompt string) []string {
	args := []string{"-p", prompt}

	// 模型選擇
	if ce.options.Model != "" {
		args = append(args, "--model", string(ce.options.Model))
	}

	// 安靜模式
	if ce.options.Silent {
		args = append(args, "-s")
	}

	// 權限控制：使用 --yolo 一次開放所有權限（等同 --allow-all-tools --allow-all-paths --allow-all-urls）
	// 這是官方推薦的自動化腳本用法，比個別旗標更可靠
	if ce.options.AllowAllTools || ce.options.AllowAllPaths || ce.options.AllowAllURLs {
		args = append(args, "--yolo")
	}

	// 自主模式
	if ce.options.NoAskUser {
		args = append(args, "--no-ask-user")
	}

	// 防止 Copilot 自動讀取 AGENTS.md / .claude/ 等指令檔，避免任務跑偏
	args = append(args, "--no-custom-instructions")

	// 禁用平行執行
	if ce.options.DisableParallel {
		args = append(args, "--disable-parallel-tools-execution")
	}

	// 允許的工具
	for _, tool := range ce.options.AllowedTools {
		args = append(args, "--allow-tool", tool)
	}

	// 禁止的工具
	for _, tool := range ce.options.DeniedTools {
		args = append(args, "--deny-tool", tool)
	}

	// 允許的目錄
	for _, dir := range ce.options.AllowedDirs {
		args = append(args, "--add-dir", dir)
	}

	// Session 相關
	if ce.options.SessionID != "" {
		args = append(args, "--resume", ce.options.SessionID)
	}

	// 分享 session
	if ce.options.SharePath != "" {
		args = append(args, "--share", ce.options.SharePath)
	}

	return args
}

// ExecutePrompt 執行任意 prompt（新版 CLI 主要方法）
func (ce *CLIExecutor) ExecutePrompt(ctx context.Context, prompt string) (*ExecutionResult, error) {
	args := ce.buildArgs(prompt)

	if os.Getenv("COPILOT_MOCK_MODE") == "true" {
		return ce.mockExecute("prompt", args)
	}

	return ce.executeWithRetry(ctx, args)
}

// ExecutePromptWithOptions 使用自訂選項執行 prompt
func (ce *CLIExecutor) ExecutePromptWithOptions(ctx context.Context, prompt string, opts ExecutorOptions) (*ExecutionResult, error) {
	// 暫存原選項
	originalOpts := ce.options
	ce.options = opts
	defer func() { ce.options = originalOpts }()

	return ce.ExecutePrompt(ctx, prompt)
}

// SuggestShellCommand 要求 Copilot 建議殼層指令
func (ce *CLIExecutor) SuggestShellCommand(ctx context.Context, description string) (*ExecutionResult, error) {
	prompt := fmt.Sprintf("建議一個殼層指令來完成以下任務: %s\n\n請只回傳指令本身，不要額外解釋。", description)

	if os.Getenv("COPILOT_MOCK_MODE") == "true" {
		return ce.mockExecute("suggest", ce.buildArgs(prompt))
	}

	return ce.executeWithRetry(ctx, ce.buildArgs(prompt))
}

// ExplainShellError 要求 Copilot 解釋殼層錯誤
func (ce *CLIExecutor) ExplainShellError(ctx context.Context, errorOutput string) (*ExecutionResult, error) {
	// 構建描述
	var description strings.Builder
	description.WriteString("解釋以下錯誤輸出並提供修復建議:\n\n")

	// 限制錯誤輸出的大小（最多 1000 字符）
	maxLen := 1000
	if len(errorOutput) > maxLen {
		description.WriteString(errorOutput[:maxLen])
		description.WriteString("...")
	} else {
		description.WriteString(errorOutput)
	}

	prompt := description.String()

	if os.Getenv("COPILOT_MOCK_MODE") == "true" {
		return ce.mockExecute("explain", ce.buildArgs(prompt))
	}

	return ce.executeWithRetry(ctx, ce.buildArgs(prompt))
}

// FixCode 要求 Copilot 修復程式碼問題
func (ce *CLIExecutor) FixCode(ctx context.Context, errorMessage string, filePath string) (*ExecutionResult, error) {
	prompt := fmt.Sprintf(`請修復以下錯誤:

錯誤訊息:
%s

檔案路徑: %s

請直接修復程式碼，不要詢問。`, errorMessage, filePath)

	if os.Getenv("COPILOT_MOCK_MODE") == "true" {
		return ce.mockExecute("fix", ce.buildArgs(prompt))
	}

	return ce.executeWithRetry(ctx, ce.buildArgs(prompt))
}

// AnalyzeAndFix 分析錯誤並自動修復（Ralph Loop 核心功能）
func (ce *CLIExecutor) AnalyzeAndFix(ctx context.Context, buildOutput string, testOutput string) (*ExecutionResult, error) {
	var prompt strings.Builder
	prompt.WriteString("分析以下輸出並修復所有錯誤:\n\n")

	if buildOutput != "" {
		prompt.WriteString("=== 建置輸出 ===\n")
		prompt.WriteString(truncateString(buildOutput, 2000))
		prompt.WriteString("\n\n")
	}

	if testOutput != "" {
		prompt.WriteString("=== 測試輸出 ===\n")
		prompt.WriteString(truncateString(testOutput, 2000))
		prompt.WriteString("\n\n")
	}

	prompt.WriteString(`請執行以下步驟:
1. 分析錯誤原因
2. 修復所有問題
3. 完成後回報修復結果

---COPILOT_STATUS---
STATUS: CONTINUE
EXIT_SIGNAL: false
TASKS_DONE: 0/1
---END_STATUS---`)

	if os.Getenv("COPILOT_MOCK_MODE") == "true" {
		return ce.mockExecute("analyze", ce.buildArgs(prompt.String()))
	}

	return ce.executeWithRetry(ctx, ce.buildArgs(prompt.String()))
}

// truncateString 截斷字串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// ResumeSession 恢復之前的 session
func (ce *CLIExecutor) ResumeSession(ctx context.Context, sessionID string) (*ExecutionResult, error) {
	args := []string{"--resume", sessionID}

	if ce.options.AllowAllTools {
		args = append(args, "--allow-all-tools")
	}

	return ce.execute(ctx, args)
}

// ContinueLastSession 繼續最近的 session
func (ce *CLIExecutor) ContinueLastSession(ctx context.Context) (*ExecutionResult, error) {
	args := []string{"--continue"}

	if ce.options.AllowAllTools {
		args = append(args, "--allow-all-tools")
	}

	return ce.execute(ctx, args)
}

// executeWithRetry 執行指令並在失敗時重試
func (ce *CLIExecutor) executeWithRetry(ctx context.Context, args []string) (*ExecutionResult, error) {
	var lastErr error
	var result *ExecutionResult

	debugLog("重試設定: 最大重試 %d 次, 延遲倍數 %v", ce.maxRetries, ce.retryDelay)

	for attempt := 0; attempt <= ce.maxRetries; attempt++ {
		if attempt > 0 {
			retryDelay := ce.retryDelay * time.Duration(attempt)
			infoLog("🔄 第 %d/%d 次重試，等待 %v...", attempt, ce.maxRetries, retryDelay)
			debugLog("重試原因: %v", lastErr)

			select {
			case <-time.After(retryDelay):
			case <-ctx.Done():
				debugLog("上下文已取消，停止重試")
				return nil, ctx.Err()
			}
		}

		result, err := ce.execute(ctx, args)

		if err == nil && result.Success {
			if attempt > 0 {
				infoLog("✅ 重試成功！")
			}
			return result, nil
		}

		lastErr = err
		result.Error = err

		// 如果達到最大重試次數，返回結果
		if attempt == ce.maxRetries {
			infoLog("❌ 已達最大重試次數 (%d), 放棄執行", ce.maxRetries)
			return result, lastErr
		}

		debugLog("執行失敗，準備重試...")
	}

	return result, lastErr
}

// execute 執行殼層指令並捕獲輸出
func (ce *CLIExecutor) execute(ctx context.Context, args []string) (*ExecutionResult, error) {
	start := time.Now()

	// 建立帶逾時的上下文
	execCtx, cancel := context.WithTimeout(ctx, ce.timeout)
	defer cancel()

	// 建立指令
	// #nosec G204 -- args 已透過 buildArgs() 組裝，參數經過引號處理，無 shell injection 風險
	cmd := exec.CommandContext(execCtx, "copilot", args...)
	cmd.Dir = ce.workDir

	// 設定環境變數
	envVars := []string{
		fmt.Sprintf("REQUEST_ID=%s", ce.requestID),
	}

	// 如果啟用除錯模式，添加 copilot 除錯環境變數
	if os.Getenv("RALPH_DEBUG") == "1" {
		envVars = append(envVars,
			"COPILOT_DEBUG=1",
			"COPILOT_LOG_LEVEL=debug",
		)
	}

	cmd.Env = append(os.Environ(), envVars...)

	// 捕獲輸出並同時顯示到終端
	var stdout, stderr bytes.Buffer
	cmd.Stdout = io.MultiWriter(&stdout, os.Stdout) // 同時寫入 buffer 和終端
	cmd.Stderr = io.MultiWriter(&stderr, os.Stderr)
	cmd.Stdin = nil // 明確設定沒有輸入，防止卡在等待輸入

	// 執行前日誌
	debugLog("========================================")
	debugLog("開始執行 Copilot CLI")
	debugLog("工作目錄: %s", ce.workDir)
	debugLog("超時設定: %v", ce.timeout)
	debugLog("Request ID: %s", ce.requestID)
	debugLog("模型: %s", ce.options.Model)

	// 顯示命令參數（隱藏過長的 prompt）
	cmdStr := "copilot"
	for i, arg := range args {
		if i > 0 && args[i-1] == "-p" && len(arg) > 100 {
			cmdStr += fmt.Sprintf(" %s \"%.100s...\"", args[i-1], arg)
			i++ // 跳過下一個參數
		} else if arg != "-p" && i > 0 && args[i-1] != "-p" {
			cmdStr += " " + arg
		}
	}
	debugLog("指令: %s", cmdStr)
	debugLog("環境變數: %v", envVars)
	debugLog("----------------------------------------")

	infoLog("⏳ 執行 Copilot CLI (超時: %v)...", ce.timeout)

	// 執行指令
	err := cmd.Run()
	executionTime := time.Since(start)

	// 檢查是否超時
	if execCtx.Err() == context.DeadlineExceeded {
		debugLog("⚠️  執行超時！已達到 %v 的限制", ce.timeout)
		infoLog("⚠️  執行超時 - 可能需要增加超時設定或檢查 Copilot CLI 狀態")
	}

	result := &ExecutionResult{
		Command:       fmt.Sprintf("copilot %s", strings.Join(args, " ")),
		Stdout:        stdout.String(),
		Stderr:        stderr.String(),
		ExecutionTime: executionTime,
		Success:       err == nil,
		Error:         err,
		Model:         ce.options.Model,
	}

	// 提取退出碼
	if exitErr, ok := err.(*exec.ExitError); ok {
		result.ExitCode = exitErr.ExitCode()
	}

	// 執行後日誌
	debugLog("----------------------------------------")
	debugLog("執行完成")
	debugLog("執行時間: %v", executionTime)
	debugLog("成功: %v", result.Success)
	debugLog("退出碼: %d", result.ExitCode)
	debugLog("Stdout 長度: %d bytes", len(result.Stdout))
	debugLog("Stderr 長度: %d bytes", len(result.Stderr))

	if result.Success {
		infoLog("✅ 執行成功 (耗時: %v)", executionTime)
	} else {
		infoLog("❌ 執行失敗 (耗時: %v, 退出碼: %d)", executionTime, result.ExitCode)
		if len(result.Stderr) > 0 {
			debugLog("錯誤輸出: %s", truncateString(result.Stderr, 500))
		}
	}

	// 顯示輸出摘要
	if len(result.Stdout) > 0 {
		debugLog("輸出摘要: %s", truncateString(result.Stdout, 200))
	}

	debugLog("========================================")

	return result, nil
}

// mockExecute 用於測試的模擬執行
func (ce *CLIExecutor) mockExecute(command string, args []string) (*ExecutionResult, error) {
	// 根據參數產生模擬響應
	mockResponse := ce.generateMockResponse(command, args)

	return &ExecutionResult{
		Command:       fmt.Sprintf("copilot %s", strings.Join(args, " ")),
		Stdout:        mockResponse,
		Stderr:        "",
		ExitCode:      0,
		ExecutionTime: 100 * time.Millisecond,
		Success:       true,
		Error:         nil,
		Model:         ce.options.Model,
	}, nil
}

// generateMockResponse 產生模擬響應
func (ce *CLIExecutor) generateMockResponse(command string, args []string) string {
	var response strings.Builder

	// 根據描述產生建議（新的 copilot CLI 使用 -p 參數）
	prompt := ""
	for i, arg := range args {
		if arg == "-p" && i+1 < len(args) {
			prompt = args[i+1]
			break
		}
	}

	// 根據 command 類型產生不同的響應
	switch command {
	case "suggest":
		response.WriteString("根據您的需求，建議使用以下指令：\n\n")
		response.WriteString("```bash\n")
		response.WriteString("# 模擬建議的指令\n")
		response.WriteString("echo 'Mock suggestion for: ")
		if prompt != "" {
			response.WriteString(truncateString(prompt, 50))
		}
		response.WriteString("'\n")
		response.WriteString("```\n\n")

	case "explain":
		response.WriteString("## 錯誤分析\n\n")
		response.WriteString("這個錯誤的原因是...\n\n")
		response.WriteString("## 修復建議\n\n")
		response.WriteString("1. 檢查相關設定\n")
		response.WriteString("2. 確認依賴版本\n")
		response.WriteString("3. 重新執行指令\n\n")

	case "fix":
		response.WriteString("已修復以下問題：\n\n")
		response.WriteString("- 修正了語法錯誤\n")
		response.WriteString("- 更新了相關引用\n\n")

	case "analyze":
		response.WriteString("## 分析結果\n\n")
		response.WriteString("發現 1 個問題需要修復。\n\n")
		response.WriteString("### 問題 1\n")
		response.WriteString("- 位置: main.go:10\n")
		response.WriteString("- 類型: 語法錯誤\n")
		response.WriteString("- 狀態: 已修復\n\n")

	default:
		if prompt != "" {
			response.WriteString(fmt.Sprintf("根據您的要求: %s\n\n", truncateString(prompt, 100)))
		}
		response.WriteString("任務已完成。\n\n")
	}

	// 添加結構化狀態輸出
	response.WriteString("---COPILOT_STATUS---\n")
	if command == "analyze" || command == "fix" {
		response.WriteString("STATUS: COMPLETED\n")
		response.WriteString("EXIT_SIGNAL: true\n")
		response.WriteString("TASKS_DONE: 1/1\n")
	} else {
		response.WriteString("STATUS: CONTINUE\n")
		response.WriteString("EXIT_SIGNAL: false\n")
		response.WriteString("TASKS_DONE: 0/5\n")
	}
	response.WriteString("---END_STATUS---\n")

	return response.String()
}

// generateRequestID 產生唯一的請求 ID
func generateRequestID() string {
	return fmt.Sprintf("copilot-req-%d", time.Now().UnixNano())
}

// debugLog 輸出除錯日誌（僅在 RALPH_DEBUG=1 時）
func debugLog(format string, args ...interface{}) {
	if os.Getenv("RALPH_DEBUG") == "1" {
		timestamp := time.Now().Format("15:04:05.000")
		fmt.Printf("[DEBUG %s] ", timestamp)
		fmt.Printf(format, args...)
		fmt.Println()
	}
}

// infoLog 輸出資訊日誌（靜默模式時不顯示）
func infoLog(format string, args ...interface{}) {
	if os.Getenv("RALPH_SILENT") == "1" {
		return
	}
	timestamp := time.Now().Format("15:04:05.000")
	fmt.Printf("[INFO %s] ", timestamp)
	fmt.Printf(format, args...)
	fmt.Println()
}

// GetWorkDir 取得工作目錄
func (ce *CLIExecutor) GetWorkDir() string {
	if ce.workDir == "" {
		wd, _ := os.Getwd()
		return wd
	}
	return ce.workDir
}

// ValidateWorkDir 驗證工作目錄是否存在
func (ce *CLIExecutor) ValidateWorkDir() error {
	workDir := ce.GetWorkDir()
	_, err := os.Stat(workDir)
	if err != nil {
		return fmt.Errorf("工作目錄無效 %s: %w", workDir, err)
	}
	return nil
}

// SetWorkDir 設定工作目錄
func (ce *CLIExecutor) SetWorkDir(workDir string) error {
	absPath, err := filepath.Abs(workDir)
	if err != nil {
		return fmt.Errorf("無法解析工作目錄: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("工作目錄不存在: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("路徑不是目錄: %s", absPath)
	}

	ce.workDir = absPath
	return nil
}
