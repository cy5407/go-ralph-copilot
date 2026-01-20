package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

// 支援互動式 REPL 的 CLI 執行器

type InteractiveCLIExecutor struct {
	cliPath       string
	timeout       time.Duration
	maxIdleTime   time.Duration
	checkInterval time.Duration

	// 互動式偵測
	promptPatterns []string // 提示輸入的關鍵字
}

func NewInteractiveCLIExecutor(cliPath string) *InteractiveCLIExecutor {
	return &InteractiveCLIExecutor{
		cliPath:       cliPath,
		timeout:       60 * time.Second,
		maxIdleTime:   3 * time.Second,
		checkInterval: 200 * time.Millisecond,

		// 偵測等待輸入的模式
		promptPatterns: []string{
			"? ",        // 常見問題提示
			": ",        // 冒號提示
			"> ",        // Shell 提示
			"Enter",     // "Press Enter" 等
			"Continue?", // 確認提示
			"[Y/n]",     // 選擇提示
			"選擇",        // 中文提示
		},
	}
}

// ExecuteInteractive 執行互動式 CLI 並處理輸入
func (e *InteractiveCLIExecutor) ExecuteInteractive(args []string, inputs []string) CLIResult {
	startTime := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, e.cliPath, args...)

	// 建立 stdin, stdout, stderr pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return CLIResult{Error: fmt.Errorf("建立 stdin pipe 失敗: %w", err)}
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return CLIResult{Error: fmt.Errorf("建立 stdout pipe 失敗: %w", err)}
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return CLIResult{Error: fmt.Errorf("建立 stderr pipe 失敗: %w", err)}
	}

	if err := cmd.Start(); err != nil {
		return CLIResult{Error: fmt.Errorf("啟動失敗: %w", err)}
	}

	var output bytes.Buffer
	lastRead := time.Now()
	hasOutput := false
	inputIndex := 0

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
				stdin.Close()
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

			// 🔍 偵測是否在等待輸入
			if e.isWaitingForInput(line) && inputIndex < len(inputs) {
				time.Sleep(500 * time.Millisecond) // 模擬人類思考

				inputData := inputs[inputIndex]
				fmt.Printf("\n[偵測到提示，輸入: %q]\n", inputData)

				_, err := stdin.Write([]byte(inputData + "\n"))
				if err != nil {
					fmt.Printf("[輸入錯誤: %v]\n", err)
				}

				inputIndex++
				lastRead = time.Now() // 重置空閒計時
			}

		case <-ticker.C:
			if !hasOutput {
				// 還沒有任何輸出，可能在思考
				elapsed := time.Since(startTime)
				if elapsed > 5*time.Second {
					fmt.Printf("[等待首次輸出... %v]\n", elapsed.Round(time.Second))
				}
			} else {
				// 已有輸出，檢查空閒
				idle := time.Since(lastRead)
				if idle > e.maxIdleTime {
					// 空閒可能代表兩種情況：
					// 1. 等待輸入（但沒有明確提示）
					// 2. 真的完成了

					if inputIndex < len(inputs) {
						// 還有輸入要送，嘗試送出
						fmt.Printf("\n[空閒 %v，嘗試送入剩餘輸入]\n", idle.Round(time.Millisecond))

						inputData := inputs[inputIndex]
						stdin.Write([]byte(inputData + "\n"))
						inputIndex++
						lastRead = time.Now()
					} else {
						// 沒有更多輸入，判定完成
						fmt.Printf("\n[空閒超過 %v，判定完成]\n", idle.Round(time.Millisecond))

						stdin.Close()
						cmd.Wait()
						return CLIResult{
							Output:   output.String(),
							Duration: time.Since(startTime),
						}
					}
				}
			}

		case <-ctx.Done():
			stdin.Close()
			cmd.Process.Kill()
			return CLIResult{
				Output:   output.String(),
				Duration: time.Since(startTime),
				Error:    fmt.Errorf("執行超時"),
			}
		}
	}
}

// isWaitingForInput 偵測輸出是否顯示等待輸入的訊號
func (e *InteractiveCLIExecutor) isWaitingForInput(line string) bool {
	for _, pattern := range e.promptPatterns {
		if strings.Contains(line, pattern) {
			return true
		}
	}
	return false
}

type CLIResult struct {
	Output   string
	Duration time.Duration
	Error    error
}

func main() {
	fmt.Println("=== 互動式 CLI 測試 ===\n")

	executor := NewInteractiveCLIExecutor("powershell.exe")

	// 測試場景：模擬需要多次輸入的互動式 CLI
	fmt.Println("測試: 互動式問答")
	fmt.Println("---")

	// 準備輸入資料
	inputs := []string{
		"yes",      // 第一個問題的回答
		"option 2", // 第二個問題的選擇
		"continue", // 確認繼續
	}

	result := executor.ExecuteInteractive(
		[]string{
			"-ExecutionPolicy", "Bypass",
			"-Command", `
				Write-Host "Question 1: Do you want to continue? [Y/n]"
				$answer1 = Read-Host
				Write-Host "You answered: $answer1"
				
				Write-Host ""
				Write-Host "Question 2: Choose an option:"
				Write-Host "  1. Option 1"
				Write-Host "  2. Option 2"
				Write-Host "Enter choice: "
				$answer2 = Read-Host
				Write-Host "You chose: $answer2"
				
				Write-Host ""
				Write-Host "Press any key to continue..."
				$answer3 = Read-Host
				Write-Host "Done!"
			`,
		},
		inputs,
	)

	fmt.Printf("\n總執行時間: %v\n", result.Duration.Round(time.Millisecond))
	fmt.Printf("總輸出:\n%s\n", result.Output)
	if result.Error != nil {
		fmt.Printf("錯誤: %v\n", result.Error)
	}

	fmt.Println("\n=== 測試完成 ===")

	// 關鍵發現
	fmt.Println("\n🔑 互動式 REPL 的關鍵機制:")
	fmt.Println("1. ✅ 建立 StdinPipe 用於送入資料")
	fmt.Println("2. ✅ 偵測輸出中的提示模式（如 '? ', '[Y/n]'）")
	fmt.Println("3. ✅ 空閒偵測 + 提示偵測雙重機制")
	fmt.Println("4. ✅ 按順序送入預先準備的輸入")
	fmt.Println("5. ✅ 在完成或超時時關閉 stdin 並等待程序")
}
