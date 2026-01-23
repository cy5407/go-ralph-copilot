package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cy540/ralph-loop/internal/ghcopilot"
)

var (
	version = "0.1.0"
)

func main() {
	// 定義子命令
	runCmd := flag.NewFlagSet("run", flag.ExitOnError)
	runPrompt := runCmd.String("prompt", "", "初始提示 (必填)")
	runMaxLoops := runCmd.Int("max-loops", 10, "最大迴圈次數")
	runTimeout := runCmd.Duration("timeout", 5*time.Minute, "總執行逾時")
	runWorkDir := runCmd.String("workdir", ".", "工作目錄")
	runSilent := runCmd.Bool("silent", false, "靜默模式")

	statusCmd := flag.NewFlagSet("status", flag.ExitOnError)
	statusWorkDir := statusCmd.String("workdir", ".", "工作目錄")

	resetCmd := flag.NewFlagSet("reset", flag.ExitOnError)
	resetWorkDir := resetCmd.String("workdir", ".", "工作目錄")

	watchCmd := flag.NewFlagSet("watch", flag.ExitOnError)
	watchWorkDir := watchCmd.String("workdir", ".", "工作目錄")
	watchInterval := watchCmd.Duration("interval", 5*time.Second, "檢查間隔")

	// 檢查參數
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		runCmd.Parse(os.Args[2:])
		if *runPrompt == "" {
			fmt.Println("錯誤: -prompt 為必填參數")
			runCmd.Usage()
			os.Exit(1)
		}
		cmdRun(*runPrompt, *runMaxLoops, *runTimeout, *runWorkDir, *runSilent)

	case "status":
		statusCmd.Parse(os.Args[2:])
		cmdStatus(*statusWorkDir)

	case "reset":
		resetCmd.Parse(os.Args[2:])
		cmdReset(*resetWorkDir)

	case "watch":
		watchCmd.Parse(os.Args[2:])
		cmdWatch(*watchWorkDir, *watchInterval)

	case "version":
		fmt.Printf("Ralph Loop v%s\n", version)

	case "help", "-h", "--help":
		printUsage()

	default:
		fmt.Printf("未知命令: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(`Ralph Loop v%s - AI 驅動的自動程式碼迭代系統

使用方式:
  ralph-loop <command> [options]

可用命令:
  run       啟動自動迴圈執行
  status    查看當前狀態
  reset     重置熔斷器
  watch     監控模式 (持續顯示狀態)
  version   顯示版本資訊
  help      顯示此幫助訊息

範例:
  # 啟動自動迴圈
  ralph-loop run -prompt "修正所有編譯錯誤" -max-loops 20

  # 查看狀態
  ralph-loop status

  # 監控模式
  ralph-loop watch -interval 3s

  # 重置熔斷器
  ralph-loop reset

更多資訊請參考: https://github.com/cy540/ralph-loop
`, version)
}

func cmdRun(prompt string, maxLoops int, timeout time.Duration, workDir string, silent bool) {
	fmt.Println("========================================")
	fmt.Println("  Ralph Loop - 自動程式碼迭代系統")
	fmt.Println("========================================")
	fmt.Printf("提示: %s\n", prompt)
	fmt.Printf("最大迴圈: %d\n", maxLoops)
	fmt.Printf("逾時: %v\n", timeout)
	fmt.Printf("工作目錄: %s\n", workDir)
	fmt.Println("----------------------------------------")

	// 建立配置
	config := ghcopilot.DefaultClientConfig()
	config.WorkDir = workDir
	config.Silent = silent
	config.CLIMaxRetries = 3
	config.CircuitBreakerThreshold = 3
	config.SameErrorThreshold = 5

	// 建立客戶端
	client := ghcopilot.NewRalphLoopClientWithConfig(config)
	defer client.Close()

	// 建立 context 與取消機制
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 處理中斷信號
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n收到中斷信號，正在停止...")
		cancel()
	}()

	fmt.Println("開始執行迴圈...")
	fmt.Println()

	// 執行迴圈（顯示進度）
	fmt.Println("⏳ 正在初始化 Copilot CLI...")
	results, err := client.ExecuteUntilCompletion(ctx, prompt, maxLoops)

	// 顯示結果摘要
	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("  執行結果摘要")
	fmt.Println("========================================")
	fmt.Printf("總迴圈數: %d\n", len(results))

	if err != nil {
		fmt.Printf("結束原因: %v\n", err)
	} else {
		fmt.Println("結束原因: 任務完成")
	}

	// 顯示狀態
	status := client.GetStatus()
	fmt.Printf("熔斷器狀態: %s\n", status.CircuitBreakerState)

	// 顯示每個迴圈的簡要
	if len(results) > 0 {
		fmt.Println()
		fmt.Println("迴圈歷史:")
		for i, r := range results {
			continueStr := "否"
			if r.ShouldContinue {
				continueStr = "是"
			}
			fmt.Printf("  [%d] 繼續=%s, 原因=%s\n", i+1, continueStr, r.ExitReason)
		}
	}

	fmt.Println("========================================")
}

func cmdStatus(workDir string) {
	config := ghcopilot.DefaultClientConfig()
	config.WorkDir = workDir

	client := ghcopilot.NewRalphLoopClientWithConfig(config)
	defer client.Close()

	// 嘗試載入歷史
	_ = client.LoadHistoryFromDisk()

	status := client.GetStatus()

	fmt.Println("========================================")
	fmt.Println("  Ralph Loop 狀態")
	fmt.Println("========================================")
	fmt.Printf("初始化: %v\n", status.Initialized)
	fmt.Printf("已關閉: %v\n", status.Closed)
	fmt.Printf("熔斷器狀態: %s\n", status.CircuitBreakerState)
	fmt.Printf("熔斷器打開: %v\n", status.CircuitBreakerOpen)
	fmt.Printf("已執行迴圈數: %d\n", status.LoopsExecuted)

	if status.Summary != nil {
		fmt.Println()
		fmt.Println("摘要:")
		for k, v := range status.Summary {
			fmt.Printf("  %s: %v\n", k, v)
		}
	}
	fmt.Println("========================================")
}

func cmdReset(workDir string) {
	config := ghcopilot.DefaultClientConfig()
	config.WorkDir = workDir

	client := ghcopilot.NewRalphLoopClientWithConfig(config)
	defer client.Close()

	err := client.ResetCircuitBreaker()
	if err != nil {
		fmt.Printf("重置失敗: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("熔斷器已重置")
}

func cmdWatch(workDir string, interval time.Duration) {
	config := ghcopilot.DefaultClientConfig()
	config.WorkDir = workDir

	client := ghcopilot.NewRalphLoopClientWithConfig(config)
	defer client.Close()

	fmt.Println("========================================")
	fmt.Println("  Ralph Loop 監控模式")
	fmt.Println("========================================")
	fmt.Printf("工作目錄: %s\n", workDir)
	fmt.Printf("更新間隔: %v\n", interval)
	fmt.Println("按 Ctrl+C 停止")
	fmt.Println("----------------------------------------")

	// 處理中斷信號
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-sigChan:
			fmt.Println("\n監控已停止")
			return
		case <-ticker.C:
			// 重新載入狀態
			_ = client.LoadHistoryFromDisk()
			status := client.GetStatus()

			// 清除並重新顯示
			fmt.Print("\033[H\033[2J") // 清除終端
			fmt.Println("========================================")
			fmt.Printf("  Ralph Loop 監控 - %s\n", time.Now().Format("15:04:05"))
			fmt.Println("========================================")
			fmt.Printf("熔斷器: %s", status.CircuitBreakerState)
			if status.CircuitBreakerOpen {
				fmt.Print(" (打開)")
			}
			fmt.Println()
			fmt.Printf("已執行迴圈: %d\n", status.LoopsExecuted)

			if status.Summary != nil {
				fmt.Println()
				for k, v := range status.Summary {
					fmt.Printf("  %s: %v\n", k, v)
				}
			}
			fmt.Println("----------------------------------------")
			fmt.Println("按 Ctrl+C 停止監控")
		}
	}
}
