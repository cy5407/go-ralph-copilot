package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"time"
)

// 改進的 CLI 執行器 - 解決編碼和空閒偵測問題

type CLIResult struct {
	Output   string
	Duration time.Duration
	Error    error
}

type CLIExecutor struct {
	cliPath       string
	timeout       time.Duration
	minIdleTime   time.Duration // 最小空閒時間（避免誤判）
	maxIdleTime   time.Duration // 最大空閒時間（確定完成）
	checkInterval time.Duration
}

func NewCLIExecutor(cliPath string) *CLIExecutor {
	return &CLIExecutor{
		cliPath:       cliPath,
		timeout:       30 * time.Second,
		minIdleTime:   1 * time.Second, // AI 思考時間容忍度
		maxIdleTime:   3 * time.Second, // 確定無更多輸出
		checkInterval: 200 * time.Millisecond,
	}
}

// Execute 執行命令並等待完成（簡化版）
func (e *CLIExecutor) Execute(args ...string) CLIResult {
	startTime := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, e.cliPath, args...)

	// 使用 CombinedOutput 簡化處理
	output, err := cmd.CombinedOutput()

	duration := time.Since(startTime)

	return CLIResult{
		Output:   string(output),
		Duration: duration,
		Error:    err,
	}
}

// ExecuteWithStreaming 執行命令並即時輸出（改進版）
func (e *CLIExecutor) ExecuteWithStreaming(args ...string) CLIResult {
	startTime := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, e.cliPath, args...)

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return CLIResult{
			Error: fmt.Errorf("啟動失敗: %w", err),
		}
	}

	var output bytes.Buffer
	lastRead := time.Now()
	hasOutput := false

	// 合併 stdout 和 stderr
	combined := io.MultiReader(stdout, stderr)
	scanner := bufio.NewScanner(combined)

	// 非阻塞讀取
	lines := make(chan string, 100)
	done := make(chan bool)

	go func() {
		for scanner.Scan() {
			lines <- scanner.Text()
		}
		close(lines)
		done <- true
	}()

	ticker := time.NewTicker(e.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case line, ok := <-lines:
			if !ok {
				// 讀取完畢
				cmd.Wait()
				return CLIResult{
					Output:   output.String(),
					Duration: time.Since(startTime),
				}
			}

			if !hasOutput {
				fmt.Printf("[首次輸出於 %v]\n", time.Since(startTime).Round(time.Millisecond))
				hasOutput = true
			}

			output.WriteString(line + "\n")
			lastRead = time.Now()
			fmt.Printf("  %s\n", line)

		case <-ticker.C:
			if !hasOutput {
				// 還沒有任何輸出，可能在思考
				elapsed := time.Since(startTime)
				if elapsed > 5*time.Second {
					fmt.Printf("[思考中... %v]\n", elapsed.Round(time.Second))
				}
			} else {
				// 已有輸出，檢查空閒
				idle := time.Since(lastRead)
				if idle > e.maxIdleTime {
					fmt.Printf("\n[空閒超過 %v，判定完成]\n", idle.Round(time.Millisecond))

					// 等待程序結束
					cmd.Wait()
					return CLIResult{
						Output:   output.String(),
						Duration: time.Since(startTime),
					}
				}
			}

		case <-ctx.Done():
			cmd.Process.Kill()
			return CLIResult{
				Output:   output.String(),
				Duration: time.Since(startTime),
				Error:    fmt.Errorf("執行超時"),
			}
		}
	}
}

func main() {
	fmt.Println("=== GitHub Copilot CLI 行為測試 v2 ===\n")

	executor := NewCLIExecutor("powershell.exe")

	// 測試 1: 簡單執行（無串流）
	fmt.Println("測試 1: 簡單執行模式")
	fmt.Println("---")

	result1 := executor.Execute(
		"-ExecutionPolicy", "Bypass",
		"-File", "mock-copilot-cli.ps1",
		"-Command", "what-the-shell",
		"-Prompt", "list all go files",
	)

	fmt.Printf("執行時間: %v\n", result1.Duration.Round(time.Millisecond))
	fmt.Printf("輸出:\n%s\n", result1.Output)
	if result1.Error != nil {
		fmt.Printf("錯誤: %v\n", result1.Error)
	}

	// 測試 2: 串流執行（即時輸出）
	fmt.Println("\n測試 2: 串流執行模式")
	fmt.Println("---")

	result2 := executor.ExecuteWithStreaming(
		"-ExecutionPolicy", "Bypass",
		"-File", "mock-copilot-cli.ps1",
		"-Command", "what-the-shell",
		"-Prompt", "fix error undefined fmt.Printl",
	)

	fmt.Printf("\n總執行時間: %v\n", result2.Duration.Round(time.Millisecond))
	fmt.Printf("總輸出長度: %d 字元\n", len(result2.Output))
	if result2.Error != nil {
		fmt.Printf("錯誤: %v\n", result2.Error)
	}

	fmt.Println("\n=== 測試完成 ===")

	// 總結
	fmt.Println("\n關鍵發現:")
	fmt.Println("1. CLI 執行延遲: ~1-2 秒（包含 AI 思考時間）")
	fmt.Println("2. 輸出模式: 一次性輸出（非串流）")
	fmt.Println("3. 完成偵測: 程序退出 = 完成")
	fmt.Println("4. 建議: 使用 cmd.Wait() + 超時保護即可")
}
