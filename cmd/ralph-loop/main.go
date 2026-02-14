package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/cy5407/go-ralph-copilot/internal/ghcopilot"
	"github.com/cy5407/go-ralph-copilot/internal/metrics"
)

var (
	version = "0.1.0"
)

// SecurityOptions 安全選項結構
type SecurityOptions struct {
	SandboxMode         bool
	AllowedCommands     string
	EnableAudit         bool
	AuditLogDir         string
	EncryptCredentials  bool
	EncryptionPassword  string
}

func main() {
	// 定義子命令
	runCmd := flag.NewFlagSet("run", flag.ExitOnError)
	runPrompt := runCmd.String("prompt", "", "初始提示 (必填)")
	runMaxLoops := runCmd.Int("max-loops", 10, "最大迴圈次數")
	runTimeout := runCmd.Duration("timeout", 5*time.Minute, "總執行逾時")
	runCliTimeout := runCmd.Duration("cli-timeout", 0, "單次 CLI 執行超時 (預設: 根據總超時自動調整)")
	runWorkDir := runCmd.String("workdir", ".", "工作目錄")
	runSilent := runCmd.Bool("silent", false, "靜默模式")
	runVerbose := runCmd.Bool("verbose", false, "詳細輸出模式")
	runQuiet := runCmd.Bool("quiet", false, "安靜模式（僅輸出結果）")
	runNoColor := runCmd.Bool("no-color", false, "禁用彩色輸出")
	runFormat := runCmd.String("format", "text", "輸出格式 (text/json/table)")
	
	// 安全選項 (T2-009)
	runSandbox := runCmd.Bool("sandbox", false, "啟用沙箱模式")
	runAllowedCommands := runCmd.String("allowed-commands", "", "允許執行的命令列表（逗號分隔）")
	runEnableAudit := runCmd.Bool("enable-audit", false, "啟用審計日誌")
	runAuditLogDir := runCmd.String("audit-log-dir", "", "審計日誌目錄")
	runEncryptCredentials := runCmd.Bool("encrypt-credentials", false, "啟用憑證加密")
	runEncryptionPassword := runCmd.String("encryption-password", "", "加密密碼（留空使用預設）")

	statusCmd := flag.NewFlagSet("status", flag.ExitOnError)
	statusWorkDir := statusCmd.String("workdir", ".", "工作目錄")
	statusCheckSDK := statusCmd.Bool("check-sdk", false, "檢查 SDK 健康狀況")

	resetCmd := flag.NewFlagSet("reset", flag.ExitOnError)
	resetWorkDir := resetCmd.String("workdir", ".", "工作目錄")

	watchCmd := flag.NewFlagSet("watch", flag.ExitOnError)
	watchWorkDir := watchCmd.String("workdir", ".", "工作目錄")
	watchInterval := watchCmd.Duration("interval", 5*time.Second, "檢查間隔")

	configCmd := flag.NewFlagSet("config", flag.ExitOnError)
	configAction := configCmd.String("action", "show", "配置操作 (show/init/validate)")
	configPath := configCmd.String("path", "", "配置文件路徑 (預設: 自動尋找)")
	configOutput := configCmd.String("output", "text", "輸出格式 (text/json)")

	metricsCmd := flag.NewFlagSet("metrics", flag.ExitOnError)
	metricsOutput := metricsCmd.String("output", "text", "輸出格式 (text/json)")
	metricsReset := metricsCmd.Bool("reset", false, "重置所有指標")

	dashboardCmd := flag.NewFlagSet("dashboard", flag.ExitOnError)
	dashboardPort := dashboardCmd.Int("port", 8080, "HTTP 服務器端口")
	dashboardHost := dashboardCmd.String("host", "localhost", "HTTP 服務器主機")
	dashboardRefresh := dashboardCmd.Int("refresh", 5, "自動刷新間隔 (秒)")

	// 插件管理命令 (T2-011)
	pluginCmd := flag.NewFlagSet("plugin", flag.ExitOnError)
	pluginAction := pluginCmd.String("action", "list", "插件操作 (list/load/unload/status/enable/disable)")
	pluginName := pluginCmd.String("name", "", "插件名稱")
	pluginPath := pluginCmd.String("path", "", "插件檔案路徑")
	pluginDir := pluginCmd.String("dir", "./plugins", "插件目錄")
	pluginAutoLoad := pluginCmd.Bool("auto-load", false, "啟用自動載入")
	pluginWorkDir := pluginCmd.String("workdir", ".", "工作目錄")

	// 檢查參數
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		runCmd.Parse(os.Args[2:])
		if *runPrompt == "" {
			PrintError("缺少必填參數: -prompt")
			runCmd.Usage()
			os.Exit(1)
		}
		
		// 設置 UI 選項
		SetColorEnabled(!*runNoColor)
		SetVerbose(*runVerbose)
		SetQuiet(*runQuiet)
		SetOutputFormat(*runFormat)
		
		// 創建安全配置
		securityConfig := SecurityOptions{
			SandboxMode:         *runSandbox,
			AllowedCommands:     *runAllowedCommands,
			EnableAudit:         *runEnableAudit,
			AuditLogDir:         *runAuditLogDir,
			EncryptCredentials:  *runEncryptCredentials,
			EncryptionPassword:  *runEncryptionPassword,
		}
		
		cmdRun(*runPrompt, *runMaxLoops, *runTimeout, *runCliTimeout, *runWorkDir, *runSilent, *runVerbose, *runQuiet, *runFormat, securityConfig)

	case "status":
		statusCmd.Parse(os.Args[2:])
		cmdStatus(*statusWorkDir, *statusCheckSDK)

	case "reset":
		resetCmd.Parse(os.Args[2:])
		cmdReset(*resetWorkDir)

	case "watch":
		watchCmd.Parse(os.Args[2:])
		cmdWatch(*watchWorkDir, *watchInterval)

	case "config":
		configCmd.Parse(os.Args[2:])
		cmdConfig(*configAction, *configPath, *configOutput)

	case "metrics":
		metricsCmd.Parse(os.Args[2:])
		cmdMetrics(*metricsOutput, *metricsReset)

	case "dashboard":
		dashboardCmd.Parse(os.Args[2:])
		cmdDashboard(*dashboardHost, *dashboardPort, *dashboardRefresh)

	case "plugin":
		pluginCmd.Parse(os.Args[2:])
		cmdPlugin(*pluginAction, *pluginName, *pluginPath, *pluginDir, *pluginAutoLoad, *pluginWorkDir)

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
  config    配置管理 (顯示、初始化、驗證配置)
  metrics   顯示性能指標
  dashboard 啟動 Web 監控儀表板
  plugin    插件管理 (載入、卸載、列出插件)
  version   顯示版本資訊
  help      顯示此幫助訊息

run 命令選項:
  -prompt string       初始提示 (必填)
  -max-loops int       最大迴圈次數 (預設: 10)
  -timeout duration    總執行逾時 (預設: 5m)
  -cli-timeout duration CLI 執行超時 (預設: 自動調整)
  -workdir string      工作目錄 (預設: ".")
  -silent              靜默模式
  -verbose             詳細輸出模式
  -quiet               安靜模式（僅輸出結果）
  -no-color            禁用彩色輸出
  -format string       輸出格式 text/json/table (預設: "text")

config 命令選項:
  -action string       配置操作 show/init/validate (預設: "show")
  -path string         配置文件路徑 (預設: 自動尋找)
  -output string       輸出格式 text/json (預設: "text")

plugin 命令選項:
  -action string       插件操作 list/load/unload/status/enable/disable/set-preferred (預設: "list")
  -name string         插件名稱
  -path string         插件檔案路徑
  -dir string          插件目錄 (預設: "./plugins")
  -auto-load           啟用自動載入
  -workdir string      工作目錄 (預設: ".")

範例:
  # 基礎用法：啟動自動迴圈
  ralph-loop run -prompt "修正所有編譯錯誤" -max-loops 20

  # 詳細輸出模式
  ralph-loop run -prompt "優化性能" -verbose

  # 使用 JSON 格式輸出
  ralph-loop run -prompt "重構程式碼" -format json

  # 使用表格格式輸出
  ralph-loop run -prompt "修復測試" -format table

  # 安靜模式（僅輸出結果）
  ralph-loop run -prompt "完成任務" -quiet

  # 查看當前狀態（支援 JSON 格式）
  ralph-loop status

  # 監控模式
  ralph-loop watch -interval 3s

  # 重置熔斷器
  ralph-loop reset

  # 配置管理
  ralph-loop config -action init              # 建立預設配置文件
  ralph-loop config -action show              # 顯示當前配置
  ralph-loop config -action show -output json # 以 JSON 格式顯示
  ralph-loop config -action validate          # 驗證配置文件

  # 查看性能指標
  ralph-loop metrics                          # 顯示所有指標統計
  ralph-loop metrics -output json             # 以 JSON 格式輸出
  ralph-loop metrics -reset                   # 重置所有指標

  # 啟動 Web 監控儀表板
  ralph-loop dashboard                         # 在 localhost:8080 啟動
  ralph-loop dashboard -port 9090             # 指定端口
  ralph-loop dashboard -host 0.0.0.0 -port 8080 # 允許外部訪問

  # 插件管理
  ralph-loop plugin -action list              # 列出所有載入的插件
  ralph-loop plugin -action status            # 顯示插件系統狀態
  ralph-loop plugin -action load -path ./plugins/openai-executor.so  # 載入插件
  ralph-loop plugin -action unload -name openai-executor              # 卸載插件
  ralph-loop plugin -action enable -auto-load # 啟用插件自動載入
  ralph-loop plugin -action disable           # 禁用插件自動載入
  ralph-loop plugin -action set-preferred -name openai-executor       # 設定偏好插件

  # 設定超時與最大迴圈數
  ralph-loop run -prompt "測試" -max-loops 5 -timeout 10m -cli-timeout 2m

進階用法:
  # 結合管道使用
  ralph-loop run -prompt "fix bugs" -format json | jq .

  # 環境變數控制
  RALPH_DEBUG=1 ralph-loop run -prompt "test"
  COPILOT_MOCK_MODE=true ralph-loop run -prompt "test"
  
  # 環境變數覆蓋配置
  RALPH_CLI_TIMEOUT=120s ralph-loop run -prompt "test"
  RALPH_MODEL=gpt-4 ralph-loop run -prompt "test"
  RALPH_VERBOSE=true ralph-loop status

錯誤處理提示:
  - 執行超時：增加 -timeout 或 -cli-timeout
  - API quota 超限：等待重置或檢查訂閱
  - 熔斷器觸發：使用 'ralph-loop reset' 重置
  - CLI 未安裝：winget install GitHub.Copilot
  - 認證失敗：copilot auth

更多資訊請參考: https://github.com/cy5407/go-ralph-copilot
`, version)
}

func cmdRun(prompt string, maxLoops int, timeout time.Duration, cliTimeout time.Duration, workDir string, silent bool, verbose bool, quiet bool, format string, securityOptions SecurityOptions) {
	startTime := time.Now()
	
	// 打印標題
	if !quietMode {
		fmt.Println(Colorize("========================================", ColorBold))
		fmt.Println(Colorize("  Ralph Loop - 自動程式碼迭代系統", ColorBold))
		fmt.Println(Colorize("========================================", ColorBold))
		PrintInfo("提示: %s", prompt)
		PrintInfo("最大迴圈: %d", maxLoops)
		PrintInfo("逾時: %v", timeout)
		PrintInfo("工作目錄: %s", workDir)
		fmt.Println(Colorize("----------------------------------------", ColorBold))
	}

	// 檢查依賴
	spinner := NewSpinner("檢查依賴環境...")
	spinner.Start()
	
	checker := ghcopilot.NewDependencyChecker()
	if err := checker.CheckAll(); err != nil {
		spinner.Stop("")
		PrintError("依賴檢查失敗: %v", err)
		fmt.Println()
		PrintInfo("安裝指引:")
		fmt.Println("1. 安裝 GitHub Copilot CLI:")
		fmt.Println("   Windows: winget install GitHub.Copilot")
		fmt.Println("   或者: npm install -g @github/copilot")
		fmt.Println()
		fmt.Println("2. 驗證安裝:")
		fmt.Println("   copilot --version")
		fmt.Println()
		fmt.Println("3. 認證 (需要有效的 GitHub Copilot 訂閱):")
		fmt.Println("   copilot auth")
		fmt.Println()
		os.Exit(1)
	}
	spinner.Stop(Colorize("✅ 依賴環境檢查通過", ColorGreen))

	// 建立配置 - 優先使用配置文件
	var config *ghcopilot.ClientConfig
	
	// 嘗試載入配置文件
	configPath := ghcopilot.GetDefaultConfigPath()
	loadedConfig, err := ghcopilot.LoadConfig(configPath)
	if err != nil {
		if verboseMode {
			PrintVerbose("載入配置文件失敗，使用預設配置: %v", err)
		}
		config = ghcopilot.DefaultClientConfig()
	} else {
		config = loadedConfig
		if verboseMode {
			PrintVerbose("已載入配置文件: %s", configPath)
		}
	}
	
	// 命令列參數覆蓋配置文件設定
	if workDir != "." {
		config.WorkDir = workDir
	}
	config.Silent = silent || config.Silent
	config.Verbose = verbose || config.Verbose
	config.Quiet = quiet || config.Quiet
	
	// 應用安全配置 (T2-009)
	if securityOptions.SandboxMode {
		config.Security.SandboxMode = true
		config.Security.WorkDir = config.WorkDir // 使用工作目錄作為沙箱限制
		
		// 解析允許的命令列表
		if securityOptions.AllowedCommands != "" {
			commands := strings.Split(securityOptions.AllowedCommands, ",")
			for i, cmd := range commands {
				commands[i] = strings.TrimSpace(cmd)
			}
			config.Security.AllowedCommands = commands
		}
		
		if !quietMode {
			PrintInfo("🔒 沙箱模式已啟用")
			if len(config.Security.AllowedCommands) > 0 {
				PrintInfo("   允許的命令: %v", config.Security.AllowedCommands)
			}
		}
	}
	
	if securityOptions.EnableAudit {
		config.Security.EnableAuditLog = true
		if securityOptions.AuditLogDir != "" {
			config.Security.AuditLogDir = securityOptions.AuditLogDir
		}
		
		if !quietMode {
			PrintInfo("📋 審計日誌已啟用")
			if config.Security.AuditLogDir != "" {
				PrintInfo("   日誌目錄: %s", config.Security.AuditLogDir)
			}
		}
	}
	
	if securityOptions.EncryptCredentials {
		config.Security.EncryptCredentials = true
		if securityOptions.EncryptionPassword != "" {
			config.Security.EncryptionPassword = securityOptions.EncryptionPassword
		}
		
		if !quietMode {
			PrintInfo("🔐 憑證加密已啟用")
		}
	}
	
	// 動態調整 CLI 超時設定
	if cliTimeout > 0 {
		// 用戶明確指定 CLI 超時
		config.CLITimeout = cliTimeout
	} else {
		// 根據總超時自動調整 CLI 超時
		// 考慮重試機制：給每個迴圈預留足夠時間，包含重試
		totalTimePerLoop := timeout / time.Duration(maxLoops)
		// CLI單次超時應該是總時間除以可能的重試次數
		autoCliTimeout := totalTimePerLoop / time.Duration(config.CLIMaxRetries+1)
		
		// 設定最小和最大邊界
		if autoCliTimeout < 60*time.Second {
			autoCliTimeout = 60 * time.Second
		}
		if autoCliTimeout > 5*time.Minute {
			autoCliTimeout = 5 * time.Minute
		}
		
		config.CLITimeout = autoCliTimeout
		if verboseMode {
			PrintVerbose("CLI 超時自動調整為: %v (每個迴圈預算: %v, 最大重試: %d)", 
				autoCliTimeout, totalTimePerLoop, config.CLIMaxRetries)
		}
	}

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
		PrintWarning("收到中斷信號，正在停止...")
		cancel()
	}()

	if !quietMode {
		PrintProgress("開始執行迴圈...")
		fmt.Println()
	}

	// 執行迴圈（使用 UI 回調）
	results, err := client.ExecuteUntilCompletion(ctx, prompt, maxLoops)

	// 使用輸出格式化器顯示結果
	totalTime := time.Since(startTime)
	formatter := ghcopilot.NewOutputFormatter(ghcopilot.OutputFormat(format))
	if formatterErr := formatter.FormatResults(results, totalTime, err); formatterErr != nil {
		PrintError("格式化輸出失敗: %v", formatterErr)
	}
	
	// 如果執行失敗，顯示友善的錯誤訊息並退出
	if err != nil {
		fmt.Println()
		PrintError("%s", ghcopilot.FormatUserFriendlyError(err))
		os.Exit(1)
	}
}

// loadConfigWithOverrides 載入配置並應用命令列覆蓋
func loadConfigWithOverrides(workDir string) *ghcopilot.ClientConfig {
	// 嘗試載入配置文件
	configPath := ghcopilot.GetDefaultConfigPath()
	config, err := ghcopilot.LoadConfig(configPath)
	if err != nil {
		// 載入失敗，使用預設配置
		config = ghcopilot.DefaultClientConfig()
	}
	
	// 命令列參數覆蓋
	if workDir != "." {
		config.WorkDir = workDir
	}
	
	return config
}

func cmdStatus(workDir string, checkSDK bool) {
	config := loadConfigWithOverrides(workDir)

	client := ghcopilot.NewRalphLoopClientWithConfig(config)
	defer client.Close()

	// 嘗試載入歷史
	_ = client.LoadHistoryFromDisk()

	status := client.GetStatus()

	// SDK 健康檢查
	if checkSDK {
		sdkHealth := client.CheckSDKHealth()
		if sdkHealth != nil {
			fmt.Printf("SDK 健康檢查:\n")
			fmt.Printf("  版本: %s\n", sdkHealth["version"])
			fmt.Printf("  狀態: %s\n", sdkHealth["status"])
			fmt.Printf("  連接: %s\n", sdkHealth["connection"])
			if sdkHealth["error"] != "" {
				fmt.Printf("  錯誤: %s\n", sdkHealth["error"])
			}
		}
		return
	}

	// 使用輸出格式化器
	formatter := ghcopilot.NewOutputFormatter(ghcopilot.OutputFormat(outputFormat))
	if err := formatter.FormatStatus(status); err != nil {
		PrintError("格式化輸出失敗: %v", err)
	}
}

func cmdReset(workDir string) {
	config := loadConfigWithOverrides(workDir)

	client := ghcopilot.NewRalphLoopClientWithConfig(config)
	defer client.Close()

	err := client.ResetCircuitBreaker()
	if err != nil {
		PrintError("重置失敗: %v", err)
		os.Exit(1)
	}

	PrintSuccess("熔斷器已重置")
}

func cmdWatch(workDir string, interval time.Duration) {
	config := loadConfigWithOverrides(workDir)

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

func cmdConfig(action string, configPath string, outputFormat string) {
	// 如果沒有指定配置路徑，使用預設路徑
	if configPath == "" {
		configPath = ghcopilot.GetDefaultConfigPath()
	}

	switch action {
	case "show":
		// 顯示當前配置
		config, err := ghcopilot.LoadConfig(configPath)
		if err != nil {
			if os.IsNotExist(err) {
				PrintWarning("配置文件不存在: %s", configPath)
				PrintInfo("使用 'ralph-loop config -action init' 建立預設配置文件")
				return
			}
			PrintError("載入配置文件失敗: %v", err)
			os.Exit(1)
		}

		fmt.Printf("配置文件路徑: %s\n", configPath)
		fmt.Println("----------------------------------------")

		if outputFormat == "json" {
			// JSON 格式輸出
			jsonData, err := ghcopilot.FormatConfigAsJSON(config)
			if err != nil {
				PrintError("格式化為 JSON 失敗: %v", err)
				os.Exit(1)
			}
			fmt.Println(jsonData)
		} else {
			// 文字格式輸出
			printConfigText(config)
		}

	case "init":
		// 初始化配置文件
		if _, err := os.Stat(configPath); err == nil {
			PrintWarning("配置文件已存在: %s", configPath)
			fmt.Print("是否要覆蓋? (y/N): ")
			
			var response string
			fmt.Scanln(&response)
			if response != "y" && response != "Y" {
				fmt.Println("已取消操作")
				return
			}
		}

		if err := ghcopilot.GenerateDefaultConfigFile(configPath); err != nil {
			PrintError("建立配置文件失敗: %v", err)
			os.Exit(1)
		}

		PrintSuccess("已建立預設配置文件: %s", configPath)
		PrintInfo("您可以編輯此文件來自訂設定")

	case "validate":
		// 驗證配置文件
		config, err := ghcopilot.LoadConfig(configPath)
		if err != nil {
			PrintError("載入配置文件失敗: %v", err)
			os.Exit(1)
		}

		if err := ghcopilot.ValidateConfigPublic(config); err != nil {
			PrintError("配置驗證失敗: %v", err)
			os.Exit(1)
		}

		PrintSuccess("配置文件驗證通過: %s", configPath)

	default:
		PrintError("未知的配置操作: %s", action)
		fmt.Println("可用操作: show, init, validate")
		os.Exit(1)
	}
}

// printConfigText 以文字格式顯示配置
func printConfigText(config *ghcopilot.ClientConfig) {
	fmt.Println("CLI 配置:")
	fmt.Printf("  超時設定: %v\n", config.CLITimeout)
	fmt.Printf("  最大重試: %d\n", config.CLIMaxRetries)
	if config.WorkDir != "" {
		fmt.Printf("  工作目錄: %s\n", config.WorkDir)
	} else {
		fmt.Printf("  工作目錄: (當前目錄)\n")
	}
	fmt.Println()

	fmt.Println("上下文配置:")
	fmt.Printf("  最大歷史: %d\n", config.MaxHistorySize)
	fmt.Printf("  儲存目錄: %s\n", config.SaveDir)
	fmt.Printf("  啟用持久化: %t\n", config.EnablePersistence)
	fmt.Printf("  使用 Gob 格式: %t\n", config.UseGobFormat)
	fmt.Println()

	fmt.Println("熔斷器配置:")
	fmt.Printf("  閾值: %d\n", config.CircuitBreakerThreshold)
	fmt.Printf("  相同錯誤閾值: %d\n", config.SameErrorThreshold)
	fmt.Println()

	fmt.Println("AI 配置:")
	fmt.Printf("  模型: %s\n", config.Model)
	fmt.Printf("  啟用 SDK: %t\n", config.EnableSDK)
	fmt.Printf("  偏好 SDK: %t\n", config.PreferSDK)
	fmt.Println()

	fmt.Println("輸出配置:")
	fmt.Printf("  靜默模式: %t\n", config.Silent)
	fmt.Printf("  詳細模式: %t\n", config.Verbose)
	fmt.Printf("  安靜模式: %t\n", config.Quiet)
}

func cmdMetrics(outputFormat string, reset bool) {
	if reset {
		PrintWarning("正在重置所有指標...")
		metrics.ResetGlobalMetrics()
		PrintSuccess("已重置所有指標")
		return
	}

	// 獲取指標摘要
	summary := metrics.GetSummary()

	fmt.Println("========================================")
	fmt.Println("  Ralph Loop 性能指標")
	fmt.Println("========================================")

	if outputFormat == "json" {
		// JSON 格式輸出
		jsonData, err := summary.ToJSON()
		if err != nil {
			PrintError("格式化為 JSON 失敗: %v", err)
			os.Exit(1)
		}
		fmt.Println(jsonData)
	} else {
		// 文字格式輸出
		fmt.Print(summary.ToText())
	}
}

func cmdDashboard(host string, port int, refreshInterval int) {
	fmt.Println("========================================")
	fmt.Printf("  Ralph Loop Web 監控儀表板\n")
	fmt.Println("========================================")
	fmt.Printf("正在啟動 HTTP 服務器於 %s:%d\n", host, port)
	fmt.Printf("自動刷新間隔: %d 秒\n", refreshInterval)
	fmt.Println("----------------------------------------")

	// TODO: 實作 HTTP 服務器
	// 這裡先提供基本的實作框架
	
	fmt.Printf("瀏覽器訪問: http://%s:%d\n", host, port)
	fmt.Println("按 Ctrl+C 停止服務器")
	
	// 處理中斷信號
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	// 簡單的 HTTP 服務器實作
	go func() {
		// 這裡將來會實作完整的 Web UI
		PrintInfo("Web 服務器功能將在後續版本中完整實作")
		PrintInfo("當前可以使用以下命令查看指標:")
		PrintInfo("  ralph-loop metrics")
		PrintInfo("  ralph-loop metrics -output json")
	}()
	
	// 等待中斷信號
	<-sigChan
	fmt.Println("\nWeb 儀表板已停止")
}

func cmdPlugin(action, pluginName, pluginPath, pluginDir string, autoLoad bool, workDir string) {
	config := loadConfigWithOverrides(workDir)
	
	// 如果插件系統未啟用，嘗試啟用它
	if !config.EnablePluginSystem {
		config.EnablePluginSystem = true
		config.PluginDir = pluginDir
	}

	client := ghcopilot.NewRalphLoopClientWithConfig(config)
	defer client.Close()

	fmt.Println("========================================")
	fmt.Printf("  Ralph Loop 插件管理 - %s\n", action)
	fmt.Println("========================================")

	switch action {
	case "list":
		// 列出所有已載入的插件
		plugins := client.ListPlugins()
		if len(plugins) == 0 {
			PrintInfo("沒有載入的插件")
			return
		}

		fmt.Printf("已載入插件數量: %d\n", len(plugins))
		fmt.Println("----------------------------------------")
		for i, plugin := range plugins {
			metadata := plugin.GetMetadata()
			fmt.Printf("%d. %s v%s\n", i+1, metadata.Name, metadata.Version)
			fmt.Printf("   作者: %s\n", metadata.Author)
			fmt.Printf("   描述: %s\n", metadata.Description)
			fmt.Printf("   類型: %s\n", metadata.Type)
			fmt.Printf("   健康狀況: %s\n", func() string {
				if plugin.IsHealthy() {
					return "正常"
				}
				return "異常"
			}())
			if i < len(plugins)-1 {
				fmt.Println()
			}
		}

	case "status":
		// 顯示插件系統狀態
		status := client.GetPluginStatus()
		if !status["enabled"].(bool) {
			PrintError("插件系統未啟用")
			PrintInfo("在配置中啟用插件系統: EnablePluginSystem = true")
			return
		}

		fmt.Printf("插件系統狀態: 已啟用\n")
		fmt.Printf("插件目錄: %s\n", status["plugin_dir"].(string))
		fmt.Printf("自動載入: %t\n", status["auto_load"].(bool))
		fmt.Printf("已載入插件: %d\n", status["plugin_count"].(int))
		
		// 顯示偏好插件
		preferred := client.GetPreferredPlugin()
		if preferred != "" {
			fmt.Printf("偏好插件: %s\n", preferred)
		} else {
			fmt.Printf("偏好插件: 未設定\n")
		}

		fmt.Println("----------------------------------------")
		
		// 顯示插件詳細信息
		if plugins, ok := status["plugins"].([]map[string]interface{}); ok && len(plugins) > 0 {
			fmt.Println("插件詳細信息:")
			for _, pluginInfo := range plugins {
				fmt.Printf("  • %s v%s (%s)\n", 
					pluginInfo["name"], 
					pluginInfo["version"],
					pluginInfo["type"])
			}
		}

	case "load":
		// 載入插件
		if pluginPath == "" {
			PrintError("缺少必要參數: -path")
			PrintInfo("使用方式: ralph-loop plugin -action load -path <插件檔案路徑>")
			os.Exit(1)
		}

		if !strings.HasSuffix(pluginPath, ".so") {
			PrintWarning("插件檔案應該是 .so 檔案")
		}

		PrintInfo("正在載入插件: %s", pluginPath)
		err := client.LoadPlugin(pluginPath)
		if err != nil {
			PrintError("載入插件失敗: %v", err)
			os.Exit(1)
		}

		PrintSuccess("插件載入成功: %s", pluginPath)

	case "unload":
		// 卸載插件
		if pluginName == "" {
			PrintError("缺少必要參數: -name")
			PrintInfo("使用方式: ralph-loop plugin -action unload -name <插件名稱>")
			os.Exit(1)
		}

		PrintInfo("正在卸載插件: %s", pluginName)
		err := client.UnloadPlugin(pluginName)
		if err != nil {
			PrintError("卸載插件失敗: %v", err)
			os.Exit(1)
		}

		PrintSuccess("插件卸載成功: %s", pluginName)

	case "enable":
		// 啟用插件自動載入
		if autoLoad {
			err := client.EnablePluginAutoLoad()
			if err != nil {
				PrintError("啟用插件自動載入失敗: %v", err)
				os.Exit(1)
			}
			PrintSuccess("已啟用插件自動載入")
		} else {
			PrintError("請使用 -auto-load 參數啟用自動載入")
			PrintInfo("使用方式: ralph-loop plugin -action enable -auto-load")
		}

	case "disable":
		// 禁用插件自動載入
		err := client.DisablePluginAutoLoad()
		if err != nil {
			PrintError("禁用插件自動載入失敗: %v", err)
			os.Exit(1)
		}
		PrintSuccess("已禁用插件自動載入")

	case "set-preferred":
		// 設定偏好插件
		if pluginName == "" {
			PrintError("缺少必要參數: -name")
			PrintInfo("使用方式: ralph-loop plugin -action set-preferred -name <插件名稱>")
			os.Exit(1)
		}

		err := client.SetPreferredPlugin(pluginName)
		if err != nil {
			PrintError("設定偏好插件失敗: %v", err)
			os.Exit(1)
		}

		PrintSuccess("已設定偏好插件: %s", pluginName)

	default:
		PrintError("未知的插件操作: %s", action)
		PrintInfo("可用操作: list, status, load, unload, enable, disable, set-preferred")
		os.Exit(1)
	}
}
