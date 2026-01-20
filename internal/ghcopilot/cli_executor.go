package ghcopilot

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
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
}

// CLIExecutor 用於執行 GitHub Copilot CLI 指令
type CLIExecutor struct {
	timeout          time.Duration
	workDir          string
	maxRetries       int
	retryDelay       time.Duration
	requestID        string
	telemetryEnabled bool
}

// NewCLIExecutor 建立新的 CLI 執行器
func NewCLIExecutor(workDir string) *CLIExecutor {
	return &CLIExecutor{
		timeout:          30 * time.Second,
		workDir:          workDir,
		maxRetries:       3,
		retryDelay:       1 * time.Second,
		requestID:        generateRequestID(),
		telemetryEnabled: true,
	}
}

// SetTimeout 設定執行逾時
func (ce *CLIExecutor) SetTimeout(duration time.Duration) {
	ce.timeout = duration
}

// SetMaxRetries 設定最大重試次數
func (ce *CLIExecutor) SetMaxRetries(retries int) {
	ce.maxRetries = retries
}

// SuggestShellCommand 要求 Copilot 建議殼層指令
func (ce *CLIExecutor) SuggestShellCommand(ctx context.Context, description string) (*ExecutionResult, error) {
	prompt := fmt.Sprintf("建議一個殼層指令來完成以下任務: %s", description)
	args := []string{
		"-p", prompt,
	}

	if os.Getenv("COPILOT_MOCK_MODE") == "true" {
		return ce.mockExecute("suggest", args)
	}

	return ce.executeWithRetry(ctx, args)
}

// ExplainShellError 要求 Copilot 解釋殼層錯誤
func (ce *CLIExecutor) ExplainShellError(ctx context.Context, errorOutput string) (*ExecutionResult, error) {
	// 構建描述
	var description strings.Builder
	description.WriteString("解釋以下錯誤輸出: ")

	// 限制錯誤輸出的大小（最多 1000 字符）
	maxLen := 1000
	if len(errorOutput) > maxLen {
		description.WriteString(errorOutput[:maxLen])
		description.WriteString("...")
	} else {
		description.WriteString(errorOutput)
	}

	args := []string{
		"-p", description.String(),
	}

	if os.Getenv("COPILOT_MOCK_MODE") == "true" {
		return ce.mockExecute("explain", args)
	}

	return ce.executeWithRetry(ctx, args)
}

// executeWithRetry 執行指令並在失敗時重試
func (ce *CLIExecutor) executeWithRetry(ctx context.Context, args []string) (*ExecutionResult, error) {
	var lastErr error
	var result *ExecutionResult

	for attempt := 0; attempt <= ce.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-time.After(ce.retryDelay * time.Duration(attempt)):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		result, err := ce.execute(ctx, args)

		if err == nil && result.Success {
			return result, nil
		}

		lastErr = err
		result.Error = err

		// 如果達到最大重試次數，返回結果
		if attempt == ce.maxRetries {
			return result, lastErr
		}
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
	cmd := exec.CommandContext(execCtx, "copilot", args...)
	cmd.Dir = ce.workDir

	// 設定環境變數
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("REQUEST_ID=%s", ce.requestID),
	)

	// 捕獲輸出
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 執行指令
	err := cmd.Run()
	executionTime := time.Since(start)

	result := &ExecutionResult{
		Command:       fmt.Sprintf("copilot %s", strings.Join(args, " ")),
		Stdout:        stdout.String(),
		Stderr:        stderr.String(),
		ExecutionTime: executionTime,
		Success:       err == nil,
		Error:         err,
	}

	// 提取退出碼
	if exitErr, ok := err.(*exec.ExitError); ok {
		result.ExitCode = exitErr.ExitCode()
	}

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
	}, nil
}

// generateMockResponse 產生模擬響應
func (ce *CLIExecutor) generateMockResponse(command string, args []string) string {
	var response strings.Builder
	response.WriteString("---COPILOT_STATUS---\n")
	response.WriteString("STATUS: CONTINUE\n")
	response.WriteString("EXIT_SIGNAL: false\n")
	response.WriteString("TASKS_DONE: 0/5\n")
	response.WriteString("---END_STATUS---\n\n")

	// 根據描述產生建議（新的 copilot CLI 使用 -p 參數）
	prompt := ""
	for i, arg := range args {
		if arg == "-p" && i+1 < len(args) {
			prompt = args[i+1]
			break
		}
	}

	if prompt != "" {
		response.WriteString(fmt.Sprintf("根據您的要求: %s\n\n", prompt))
		response.WriteString("建議的指令:\n\n")
		response.WriteString("```bash\n")
		response.WriteString("# 模擬建議\n")
		response.WriteString("echo 'Mock suggestion'\n")
		response.WriteString("```\n")
	}

	return response.String()
}

// generateRequestID 產生唯一的請求 ID
func generateRequestID() string {
	return fmt.Sprintf("copilot-req-%d", time.Now().UnixNano())
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
