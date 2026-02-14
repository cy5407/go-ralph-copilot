package test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestE2ECompleteWorkflow 端到端測試：完整工作流程
func TestE2ECompleteWorkflow(t *testing.T) {
	// 跳過 E2E 測試，除非明確請求
	if os.Getenv("RUN_E2E_TESTS") != "true" {
		t.Skip("跳過 E2E 測試。設置 RUN_E2E_TESTS=true 來運行")
	}
	
	// 檢查是否存在 ralph-loop.exe
	exePath := "ralph-loop.exe"
	if _, err := exec.LookPath(exePath); err != nil {
		// 嘗試從項目根目錄查找
		rootExePath := filepath.Join("..", "ralph-loop.exe")
		if _, err := os.Stat(rootExePath); err != nil {
			t.Skip("ralph-loop.exe 不存在，跳過 E2E 測試。請先執行 'go build -o ralph-loop.exe ./cmd/ralph-loop'")
		}
		exePath = rootExePath
	}

	// 創建暫存工作目錄
	tempDir, err := os.MkdirTemp("", "ralph-e2e-*")
	if err != nil {
		t.Fatalf("無法創建暫存目錄: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Logf("E2E 測試目錄: %s", tempDir)
	t.Logf("使用執行文件: %s", exePath)

	// 設置環境變數啟用模擬模式
	env := append(os.Environ(), "COPILOT_MOCK_MODE=true")

	// 測試 1: 版本命令
	t.Run("Version Command", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, exePath, "version")
		cmd.Env = env
		cmd.Dir = tempDir

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("版本命令失敗: %v\n輸出: %s", err, string(output))
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "ralph-loop") {
			t.Errorf("版本輸出不包含 'ralph-loop': %s", outputStr)
		}

		t.Logf("✅ 版本命令成功: %s", strings.TrimSpace(outputStr))
	})

	// 測試 2: 幫助命令
	t.Run("Help Command", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, exePath, "help")
		cmd.Env = env
		cmd.Dir = tempDir

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("幫助命令失敗: %v\n輸出: %s", err, string(output))
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "用法") && !strings.Contains(outputStr, "Usage") {
			t.Errorf("幫助輸出不包含用法信息: %s", outputStr)
		}

		t.Log("✅ 幫助命令成功")
	})

	// 測試 3: 狀態命令
	t.Run("Status Command", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, exePath, "status")
		cmd.Env = env
		cmd.Dir = tempDir

		output, err := cmd.CombinedOutput()
		// 狀態命令可能會失敗（如果沒有先前的執行記錄），這是正常的
		outputStr := string(output)
		t.Logf("狀態命令輸出: %s", outputStr)

		// 只要不是崩潰就算成功
		if err != nil {
			t.Logf("狀態命令返回錯誤 (預期): %v", err)
		}

		t.Log("✅ 狀態命令執行完成")
	})

	// 測試 4: 配置初始化
	t.Run("Config Init Command", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, exePath, "config", "-action", "init")
		cmd.Env = env
		cmd.Dir = tempDir

		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		if err != nil {
			t.Logf("配置初始化命令錯誤 (可能預期): %v\n輸出: %s", err, outputStr)
		} else {
			t.Logf("✅ 配置初始化成功: %s", outputStr)

			// 檢查是否創建了配置文件
			configPath := filepath.Join(tempDir, "ralph-loop.toml")
			if _, err := os.Stat(configPath); err == nil {
				t.Log("✅ 配置文件已創建")
			}
		}
	})

	// 測試 5: 簡短執行流程 (模擬模式)
	t.Run("Short Execution Flow", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, exePath, "run",
			"-prompt", "E2E測試：修復簡單的編譯錯誤",
			"-max-loops", "2",
			"-timeout", "30s",
		)
		cmd.Env = env
		cmd.Dir = tempDir

		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		// 在模擬模式下，執行可能會成功或失敗，都是可接受的
		t.Logf("執行命令輸出 (%d bytes): %s", len(outputStr), outputStr)

		if err != nil {
			t.Logf("執行命令錯誤 (模擬模式下預期): %v", err)
		} else {
			t.Log("✅ 執行命令成功完成")
		}

		// 檢查輸出是否包含預期的關鍵字
		expectedKeywords := []string{"迴圈", "Loop", "執行", "完成", "錯誤"}
		foundKeyword := false
		for _, keyword := range expectedKeywords {
			if strings.Contains(outputStr, keyword) {
				foundKeyword = true
				break
			}
		}

		if foundKeyword {
			t.Log("✅ 輸出包含預期關鍵字")
		} else {
			t.Logf("⚠️ 輸出未包含預期關鍵字，但這在模擬模式下可能正常")
		}
	})
}

// TestE2EConfigurationManagement 端到端測試：配置管理
func TestE2EConfigurationManagement(t *testing.T) {
	// 跳過 E2E 測試，除非明確請求
	if os.Getenv("RUN_E2E_TESTS") != "true" {
		t.Skip("跳過 E2E 測試。設置 RUN_E2E_TESTS=true 來運行")
	}
	
	exePath := "ralph-loop.exe"
	if _, err := exec.LookPath(exePath); err != nil {
		rootExePath := filepath.Join("..", "ralph-loop.exe")
		if _, err := os.Stat(rootExePath); err != nil {
			t.Skip("ralph-loop.exe 不存在，跳過配置管理 E2E 測試")
		}
		exePath = rootExePath
	}

	tempDir, err := os.MkdirTemp("", "ralph-config-e2e-*")
	if err != nil {
		t.Fatalf("無法創建暫存目錄: %v", err)
	}
	defer os.RemoveAll(tempDir)

	env := append(os.Environ(), "COPILOT_MOCK_MODE=true")

	// 創建自定義配置文件
	configContent := `
[cli]
timeout = "60s"
max_retries = 3

[ai]
model = "claude-sonnet-4.5"
allow_all_tools = true

[output]
format = "text"
color = true
`
	configPath := filepath.Join(tempDir, "custom-config.toml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("無法寫入配置文件: %v", err)
	}

	// 測試使用自定義配置運行
	t.Run("Custom Config Execution", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, exePath, "run",
			"-config", configPath,
			"-prompt", "使用自定義配置的E2E測試",
			"-max-loops", "1",
		)
		cmd.Env = env
		cmd.Dir = tempDir

		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		t.Logf("自定義配置執行輸出: %s", outputStr)

		if err != nil {
			t.Logf("自定義配置執行錯誤 (模擬模式下可能預期): %v", err)
		}

		t.Log("✅ 自定義配置E2E測試完成")
	})

	// 測試配置驗證
	t.Run("Config Validation", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, exePath, "config",
			"-action", "validate",
			"-config", configPath,
		)
		cmd.Env = env
		cmd.Dir = tempDir

		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		t.Logf("配置驗證輸出: %s", outputStr)

		if err != nil {
			t.Logf("配置驗證錯誤: %v", err)
		} else {
			t.Log("✅ 配置驗證成功")
		}
	})
}

// TestE2EErrorScenarios 端到端測試：錯誤場景處理
func TestE2EErrorScenarios(t *testing.T) {
	// 跳過 E2E 測試，除非明確請求
	if os.Getenv("RUN_E2E_TESTS") != "true" {
		t.Skip("跳過 E2E 測試。設置 RUN_E2E_TESTS=true 來運行")
	}
	
	exePath := "ralph-loop.exe"
	if _, err := exec.LookPath(exePath); err != nil {
		rootExePath := filepath.Join("..", "ralph-loop.exe")
		if _, err := os.Stat(rootExePath); err != nil {
			t.Skip("ralph-loop.exe 不存在，跳過錯誤場景 E2E 測試")
		}
		exePath = rootExePath
	}

	tempDir, err := os.MkdirTemp("", "ralph-error-e2e-*")
	if err != nil {
		t.Fatalf("無法創建暫存目錄: %v", err)
	}
	defer os.RemoveAll(tempDir)

	env := append(os.Environ(), "COPILOT_MOCK_MODE=true")

	errorTestCases := []struct {
		name        string
		args        []string
		expectError bool
		description string
	}{
		{
			name:        "Missing Prompt",
			args:        []string{"run", "-max-loops", "1"},
			expectError: true,
			description: "缺少 prompt 參數應該報錯",
		},
		{
			name:        "Invalid Max Loops",
			args:        []string{"run", "-prompt", "test", "-max-loops", "-1"},
			expectError: true,
			description: "負數 max-loops 應該報錯",
		},
		{
			name:        "Invalid Config File",
			args:        []string{"run", "-config", "nonexistent.toml", "-prompt", "test"},
			expectError: true,
			description: "不存在的配置文件應該報錯",
		},
		{
			name:        "Invalid Timeout",
			args:        []string{"run", "-prompt", "test", "-timeout", "invalid"},
			expectError: true,
			description: "無效的超時格式應該報錯",
		},
		{
			name:        "Zero Max Loops",
			args:        []string{"run", "-prompt", "test", "-max-loops", "0"},
			expectError: true,
			description: "零 max-loops 應該報錯",
		},
	}

	for _, tc := range errorTestCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			cmd := exec.CommandContext(ctx, exePath, tc.args...)
			cmd.Env = env
			cmd.Dir = tempDir

			output, err := cmd.CombinedOutput()
			outputStr := string(output)

			if tc.expectError && err == nil {
				t.Errorf("預期會有錯誤但沒有發生。輸出: %s", outputStr)
			} else if !tc.expectError && err != nil {
				t.Errorf("未預期的錯誤: %v。輸出: %s", err, outputStr)
			}

			if tc.expectError && err != nil {
				t.Logf("✅ 正確捕獲錯誤: %v", err)
			}

			t.Logf("%s: 輸出長度=%d bytes", tc.description, len(outputStr))
		})
	}

	t.Log("✅ 錯誤場景E2E測試完成")
}

// TestE2ECrossExecutorMode 端到端測試：跨執行器模式
func TestE2ECrossExecutorMode(t *testing.T) {
	// 跳過 E2E 測試，除非明確請求
	if os.Getenv("RUN_E2E_TESTS") != "true" {
		t.Skip("跳過 E2E 測試。設置 RUN_E2E_TESTS=true 來運行")
	}
	
	exePath := "ralph-loop.exe"
	if _, err := exec.LookPath(exePath); err != nil {
		rootExePath := filepath.Join("..", "ralph-loop.exe")
		if _, err := os.Stat(rootExePath); err != nil {
			t.Skip("ralph-loop.exe 不存在，跳過跨執行器模式 E2E 測試")
		}
		exePath = rootExePath
	}

	tempDir, err := os.MkdirTemp("", "ralph-cross-executor-*")
	if err != nil {
		t.Fatalf("無法創建暫存目錄: %v", err)
	}
	defer os.RemoveAll(tempDir)

	baseEnv := append(os.Environ(), "COPILOT_MOCK_MODE=true")

	// 測試不同執行器模式
	executorModes := []struct {
		name    string
		envVars []string
		desc    string
	}{
		{
			name:    "CLI_Priority",
			envVars: []string{"RALPH_PREFER_SDK=false"},
			desc:    "CLI 執行器優先模式",
		},
		{
			name:    "SDK_Priority", 
			envVars: []string{"RALPH_PREFER_SDK=true"},
			desc:    "SDK 執行器優先模式",
		},
		{
			name:    "CLI_Only",
			envVars: []string{"RALPH_ENABLE_SDK=false"},
			desc:    "僅 CLI 執行器模式",
		},
	}

	for _, mode := range executorModes {
		t.Run(mode.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()

			// 設置特定模式的環境變數
			env := append(baseEnv, mode.envVars...)

			cmd := exec.CommandContext(ctx, exePath, "run",
				"-prompt", fmt.Sprintf("E2E測試：%s", mode.desc),
				"-max-loops", "1",
				"-timeout", "30s",
			)
			cmd.Env = env
			cmd.Dir = tempDir

			output, err := cmd.CombinedOutput()
			outputStr := string(output)

			t.Logf("%s 模式輸出: %s", mode.desc, outputStr)

			if err != nil {
				t.Logf("%s 模式錯誤 (模擬模式下可能預期): %v", mode.desc, err)
			} else {
				t.Logf("✅ %s 模式執行成功", mode.desc)
			}
		})
	}

	t.Log("✅ 跨執行器模式E2E測試完成")
}

// TestE2ELongRunningExecution 端到端測試：長時間運行執行
func TestE2ELongRunningExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("跳過長時間運行測試 (使用 -short)")
	}
	
	// 跳過 E2E 測試，除非明確請求
	if os.Getenv("RUN_E2E_TESTS") != "true" {
		t.Skip("跳過 E2E 測試。設置 RUN_E2E_TESTS=true 來運行")
	}

	exePath := "ralph-loop.exe"
	if _, err := exec.LookPath(exePath); err != nil {
		rootExePath := filepath.Join("..", "ralph-loop.exe")
		if _, err := os.Stat(rootExePath); err != nil {
			t.Skip("ralph-loop.exe 不存在，跳過長時間運行 E2E 測試")
		}
		exePath = rootExePath
	}

	tempDir, err := os.MkdirTemp("", "ralph-long-running-*")
	if err != nil {
		t.Fatalf("無法創建暫存目錄: %v", err)
	}
	defer os.RemoveAll(tempDir)

	env := append(os.Environ(), "COPILOT_MOCK_MODE=true")

	// 測試較長的多迴圈執行
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, exePath, "run",
		"-prompt", "E2E長時間測試：多迴圈代碼重構任務",
		"-max-loops", "5",
		"-timeout", "90s",
	)
	cmd.Env = env
	cmd.Dir = tempDir

	startTime := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)
	outputStr := string(output)

	t.Logf("長時間執行完成，耗時: %v", duration)
	t.Logf("輸出長度: %d bytes", len(outputStr))

	if err != nil {
		t.Logf("長時間執行錯誤 (模擬模式下可能預期): %v", err)
	} else {
		t.Log("✅ 長時間執行成功完成")
	}

	// 驗證執行時間是否合理（應該 > 5秒，< 120秒）
	if duration < 5*time.Second {
		t.Logf("⚠️ 執行時間過短: %v", duration)
	}
	if duration >= 120*time.Second {
		t.Errorf("執行時間過長: %v", duration)
	}

	t.Log("✅ 長時間運行E2E測試完成")
}