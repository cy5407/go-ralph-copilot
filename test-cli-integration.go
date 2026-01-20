package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

// 測試 CLI 執行和輸出偵測

type CLIExecutor struct {
	cliPath     string
	timeout     time.Duration
	idleTimeout time.Duration
}

func NewCLIExecutor(cliPath string) *CLIExecutor {
	return &CLIExecutor{
		cliPath:     cliPath,
		timeout:     30 * time.Second,
		idleTimeout: 500 * time.Millisecond,
	}
}

// ExecuteWithIdleDetection 執行命令並偵測輸出空閒
func (e *CLIExecutor) ExecuteWithIdleDetection(args ...string) (string, time.Duration, error) {
	startTime := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, e.cliPath, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", 0, fmt.Errorf("建立 stdout pipe 失敗: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", 0, fmt.Errorf("啟動命令失敗: %w", err)
	}

	var output bytes.Buffer
	scanner := bufio.NewScanner(stdout)
	lastOutputTime := time.Now()

	// 使用 goroutine 非阻塞讀取
	lines := make(chan string, 100)
	done := make(chan bool)

	go func() {
		for scanner.Scan() {
			lines <- scanner.Text()
		}
		close(lines)
		done <- true
	}()

	// 空閒偵測計時器
	idleTicker := time.NewTicker(100 * time.Millisecond)
	defer idleTicker.Stop()

	for {
		select {
		case line, ok := <-lines:
			if !ok {
				// 所有輸出已讀取完畢
				cmd.Wait()
				elapsed := time.Since(startTime)
				return output.String(), elapsed, nil
			}

			output.WriteString(line + "\n")
			lastOutputTime = time.Now()
			fmt.Printf("[%s] %s\n", time.Since(startTime).Round(time.Millisecond), line)

		case <-idleTicker.C:
			idle := time.Since(lastOutputTime)
			if idle > e.idleTimeout {
				// 空閒超時 - 可能等待輸入或已完成
				fmt.Printf("\n[偵測到空閒: %v]\n", idle.Round(time.Millisecond))

				// 檢查程序是否還在執行
				select {
				case <-done:
					// 程序已結束
					cmd.Wait()
					elapsed := time.Since(startTime)
					return output.String(), elapsed, nil
				default:
					// 程序還在執行，可能等待輸入
					fmt.Println("[警告: 程序仍在執行但無輸出]")
				}
			}

		case <-ctx.Done():
			// 絕對超時
			cmd.Process.Kill()
			elapsed := time.Since(startTime)
			return output.String(), elapsed, fmt.Errorf("執行超時")
		}
	}
}

func main() {
	fmt.Println("=== GitHub Copilot CLI 行為測試 ===\n")

	// 使用模擬 CLI
	executor := NewCLIExecutor("powershell.exe")

	// 測試 1: Shell 指令建議
	fmt.Println("測試 1: Shell 指令建議")
	fmt.Println("命令: what-the-shell '列出 go 檔案'")
	fmt.Println("---")

	output, duration, err := executor.ExecuteWithIdleDetection(
		"-File", "mock-copilot-cli.ps1",
		"-Command", "what-the-shell",
		"-Prompt", "列出當前目錄所有 go 檔案",
	)

	if err != nil {
		fmt.Printf("錯誤: %v\n", err)
	}

	fmt.Printf("\n執行時間: %v\n", duration.Round(time.Millisecond))
	fmt.Printf("輸出長度: %d 字元\n", len(output))
	fmt.Printf("輸出行數: %d\n", bytes.Count([]byte(output), []byte("\n")))

	// 測試 2: 錯誤修正建議
	fmt.Println("\n\n測試 2: 錯誤修正建議")
	fmt.Println("命令: what-the-shell '修正錯誤'")
	fmt.Println("---")

	output2, duration2, err2 := executor.ExecuteWithIdleDetection(
		"-File", "mock-copilot-cli.ps1",
		"-Command", "what-the-shell",
		"-Prompt", "修正錯誤: undefined: fmt.Printl",
	)

	if err2 != nil {
		fmt.Printf("錯誤: %v\n", err2)
	}

	fmt.Printf("\n執行時間: %v\n", duration2.Round(time.Millisecond))
	fmt.Printf("輸出長度: %d 字元\n", len(output2))

	fmt.Println("\n=== 測試完成 ===")
}
