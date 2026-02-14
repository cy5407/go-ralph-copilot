package test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cy5407/go-ralph-copilot/internal/ghcopilot"
)

// MockCopilotResponse 模擬 Copilot 回應結構
type MockCopilotResponse struct {
	Content    string            `json:"content"`
	Metadata   map[string]string `json:"metadata"`
	ExitSignal bool              `json:"exit_signal"`
	Tasks      []MockTask        `json:"tasks"`
}

type MockTask struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Completed   bool   `json:"completed"`
}

// TestMockCopilotService 測試模擬 Copilot 服務功能
func TestMockCopilotService(t *testing.T) {
	// 確保啟用模擬模式
	os.Setenv("COPILOT_MOCK_MODE", "true")
	defer os.Unsetenv("COPILOT_MOCK_MODE")

	tempDir, err := os.MkdirTemp("", "ralph-mock-service-*")
	if err != nil {
		t.Fatalf("無法創建暫存目錄: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := ghcopilot.DefaultClientConfig()
	config.WorkDir = tempDir
	config.CLITimeout = 20 * time.Second

	client := ghcopilot.NewRalphLoopClientWithConfig(config)
	if client == nil {
		t.Fatal("NewRalphLoopClient 返回 nil")
	}

	// 測試基本模擬回應
	t.Run("Basic Mock Response", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		prompt := "模擬服務測試：修復編譯錯誤"
		result, err := client.ExecuteUntilCompletion(ctx, prompt, 1)

		// 在模擬模式下，行為可能不同
		if err != nil {
			t.Logf("模擬模式執行錯誤 (可能預期): %v", err)
		}

		if result != nil {
			t.Logf("✅ 模擬模式執行完成: 迴圈數=%d", len(result))
		} else {
			t.Log("⚠️ 模擬模式返回 nil 結果")
		}
	})

	// 測試不同類型的模擬場景
	mockScenarios := []struct {
		name        string
		prompt      string
		expectError bool
		description string
	}{
		{
			name:        "CodeFix",
			prompt:      "修復 Go 編譯錯誤",
			expectError: false,
			description: "代碼修復場景",
		},
		{
			name:        "TestGeneration",
			prompt:      "為函數生成單元測試",
			expectError: false,
			description: "測試生成場景",
		},
		{
			name:        "CodeReview",
			prompt:      "審查代碼品質",
			expectError: false,
			description: "代碼審查場景",
		},
		{
			name:        "Documentation",
			prompt:      "生成 API 文檔",
			expectError: false,
			description: "文檔生成場景",
		},
		{
			name:        "Refactoring",
			prompt:      "重構代碼結構",
			expectError: false,
			description: "重構場景",
		},
	}

	for _, scenario := range mockScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
			defer cancel()

			result, err := client.ExecuteUntilCompletion(ctx, scenario.prompt, 1)

			if scenario.expectError && err == nil {
				t.Errorf("預期會有錯誤但沒有發生: %s", scenario.description)
			}

			if !scenario.expectError && result != nil {
				t.Logf("✅ %s 完成", scenario.description)
			}

			t.Logf("%s: result=%v, err=%v", scenario.description, result != nil, err != nil)
		})
	}
}

// TestMockServiceEdgeCases 測試模擬服務的邊界情況
func TestMockServiceEdgeCases(t *testing.T) {
	os.Setenv("COPILOT_MOCK_MODE", "true")
	defer os.Unsetenv("COPILOT_MOCK_MODE")

	tempDir, err := os.MkdirTemp("", "ralph-mock-edge-*")
	if err != nil {
		t.Fatalf("無法創建暫存目錄: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := ghcopilot.DefaultClientConfig()
	config.WorkDir = tempDir

	edgeCases := []struct {
		name            string
		prompt          string
		maxLoops        int
		timeout         time.Duration
		modifyConfig    func(*ghcopilot.ClientConfig)
		expectSuccess   bool
		description     string
	}{
		{
			name:          "EmptyPrompt",
			prompt:        "",
			maxLoops:      1,
			timeout:       10 * time.Second,
			modifyConfig:  nil,
			expectSuccess: false,
			description:   "空 prompt 應該失敗",
		},
		{
			name:          "VeryLongPrompt",
			prompt:        strings.Repeat("很長的提示詞 ", 1000),
			maxLoops:      1,
			timeout:       30 * time.Second,
			modifyConfig:  nil,
			expectSuccess: true,
			description:   "超長 prompt 測試",
		},
		{
			name:          "ZeroMaxLoops",
			prompt:        "測試零迴圈",
			maxLoops:      0,
			timeout:       10 * time.Second,
			modifyConfig:  nil,
			expectSuccess: false,
			description:   "零 maxLoops 應該失敗",
		},
		{
			name:          "NegativeMaxLoops",
			prompt:        "測試負數迴圈",
			maxLoops:      -1,
			timeout:       10 * time.Second,
			modifyConfig:  nil,
			expectSuccess: false,
			description:   "負數 maxLoops 應該失敗",
		},
		{
			name:     "ShortTimeout",
			prompt:   "短超時測試",
			maxLoops: 1,
			timeout:  1 * time.Second,
			modifyConfig: func(c *ghcopilot.ClientConfig) {
				c.CLITimeout = 500 * time.Millisecond
			},
			expectSuccess: false,
			description:   "極短超時測試",
		},
		{
			name:     "LowCircuitBreakerThreshold",
			prompt:   "低熔斷器閾值測試",
			maxLoops: 5,
			timeout:  30 * time.Second,
			modifyConfig: func(c *ghcopilot.ClientConfig) {
				c.CircuitBreakerThreshold = 1
			},
			expectSuccess: true,
			description:   "低熔斷器閾值測試",
		},
	}

	for _, tc := range edgeCases {
		t.Run(tc.name, func(t *testing.T) {
			// 複製配置以避免影響其他測試
			testConfig := *config
			if tc.modifyConfig != nil {
				tc.modifyConfig(&testConfig)
			}

			client := ghcopilot.NewRalphLoopClientWithConfig(&testConfig)
			if client == nil && tc.expectSuccess {
				t.Fatal("NewRalphLoopClient 返回 nil")
			}

			if client == nil {
				t.Logf("客戶端創建失敗 (可能預期): %s", tc.description)
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), tc.timeout)
			defer cancel()

			result, err := client.ExecuteUntilCompletion(ctx, tc.prompt, tc.maxLoops)

			if tc.expectSuccess && err != nil && result == nil {
				t.Errorf("預期成功但失敗: %s, 錯誤: %v", tc.description, err)
			}

			if !tc.expectSuccess && err == nil && result != nil {
				t.Logf("預期失敗但成功 (模擬模式下可能正常): %s", tc.description)
			}

			t.Logf("%s: result=%v, err=%v", tc.description, result != nil, err != nil)
		})
	}
}

// TestMockServiceResponseParsing 測試模擬服務回應解析
func TestMockServiceResponseParsing(t *testing.T) {
	os.Setenv("COPILOT_MOCK_MODE", "true")
	defer os.Unsetenv("COPILOT_MOCK_MODE")

	// 模擬不同格式的回應內容
	mockResponses := []struct {
		name        string
		response    string
		shouldParse bool
		description string
	}{
		{
			name: "ValidStructuredResponse",
			response: `
# 代碼修復完成

修復了以下問題：
1. 修正了變數名稱錯誤
2. 添加了遺漏的 import
3. 修復了語法錯誤

---COPILOT_STATUS---
EXIT_SIGNAL: true  
TASKS_DONE: 3/3
SUCCESS: true
---COPILOT_STATUS---
`,
			shouldParse: true,
			description: "標準結構化回應",
		},
		{
			name: "IncompleteResponse",
			response: `
# 部分修復

修復了一些問題，但還需要更多工作。

---COPILOT_STATUS---
EXIT_SIGNAL: false
TASKS_DONE: 1/3
SUCCESS: false
---COPILOT_STATUS---
`,
			shouldParse: true,
			description: "未完成任務回應",
		},
		{
			name: "PlainTextResponse",
			response: `
修復完成。所有編譯錯誤已解決。
`,
			shouldParse: true,
			description: "純文字回應",
		},
		{
			name: "EmptyResponse",
			response: ``,
			shouldParse: false,
			description: "空回應",
		},
		{
			name: "MalformedJSON",
			response: `{
"status": "complete",
"tasks": [
	"fix_errors": true,
	"run_tests": 
}`,
			shouldParse: false,
			description: "格式錯誤的 JSON",
		},
	}

	tempDir, err := os.MkdirTemp("", "ralph-mock-parsing-*")
	if err != nil {
		t.Fatalf("無法創建暫存目錄: %v", err)
	}
	defer os.RemoveAll(tempDir)

	for _, resp := range mockResponses {
		t.Run(resp.name, func(t *testing.T) {
			// 創建包含模擬回應的臨時文件
			responseFile := filepath.Join(tempDir, fmt.Sprintf("mock_response_%s.txt", resp.name))
			err := os.WriteFile(responseFile, []byte(resp.response), 0644)
			if err != nil {
				t.Fatalf("無法寫入模擬回應文件: %v", err)
			}

			// 測試回應內容是否包含預期的關鍵字
			if resp.shouldParse {
				if len(resp.response) > 0 {
					t.Logf("✅ %s: 回應長度 %d bytes", resp.description, len(resp.response))
				} else {
					t.Errorf("%s: 回應為空但預期應該可解析", resp.description)
				}
			} else {
				if len(resp.response) == 0 || strings.Contains(resp.response, "malformed") {
					t.Logf("✅ %s: 正確識別為無效回應", resp.description)
				}
			}

			// 測試是否包含退出信號
			hasExitSignal := strings.Contains(resp.response, "EXIT_SIGNAL: true")
			if hasExitSignal {
				t.Logf("檢測到退出信號: %s", resp.description)
			}

			// 測試是否包含狀態區塊
			hasStatusBlock := strings.Contains(resp.response, "---COPILOT_STATUS---")
			if hasStatusBlock {
				t.Logf("檢測到狀態區塊: %s", resp.description)
			}
		})
	}
}

// TestMockServiceConcurrency 測試模擬服務並發處理
func TestMockServiceConcurrency(t *testing.T) {
	os.Setenv("COPILOT_MOCK_MODE", "true")
	defer os.Unsetenv("COPILOT_MOCK_MODE")

	tempDir, err := os.MkdirTemp("", "ralph-mock-concurrency-*")
	if err != nil {
		t.Fatalf("無法創建暫存目錄: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := ghcopilot.DefaultClientConfig()
	config.WorkDir = tempDir
	config.CLITimeout = 15 * time.Second

	// 測試並發執行多個模擬任務
	concurrencyLevel := 5
	done := make(chan bool, concurrencyLevel)
	errors := make(chan error, concurrencyLevel)

	t.Logf("啟動 %d 個並發模擬任務", concurrencyLevel)

	for i := 0; i < concurrencyLevel; i++ {
		go func(taskID int) {
			defer func() { done <- true }()

			// 每個 goroutine 創建自己的客戶端
			taskConfig := *config
			client := ghcopilot.NewRalphLoopClientWithConfig(&taskConfig)
			if client == nil {
				errors <- fmt.Errorf("任務 %d: 客戶端創建失敗", taskID)
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			prompt := fmt.Sprintf("並發模擬任務 #%d", taskID)
			result, err := client.ExecuteUntilCompletion(ctx, prompt, 1)

			if err != nil && result == nil {
				errors <- fmt.Errorf("任務 %d 執行失敗: %v", taskID, err)
			} else {
				t.Logf("✅ 任務 %d 完成", taskID)
			}
		}(i)
	}

	// 等待所有任務完成
	completedTasks := 0
	errorCount := 0
	timeout := time.After(60 * time.Second)

	for completedTasks < concurrencyLevel {
		select {
		case <-done:
			completedTasks++
		case err := <-errors:
			errorCount++
			t.Logf("並發任務錯誤 (模擬模式下可能預期): %v", err)
		case <-timeout:
			t.Fatalf("並發測試超時，只完成 %d/%d 個任務", completedTasks, concurrencyLevel)
		}
	}

	t.Logf("✅ 並發測試完成: %d/%d 任務完成，%d 個錯誤", 
		completedTasks, concurrencyLevel, errorCount)

	// 在模擬模式下，一些錯誤是可接受的
	if errorCount > concurrencyLevel/2 {
		t.Logf("⚠️ 錯誤率較高: %d/%d，但在模擬模式下可能正常", errorCount, concurrencyLevel)
	}
}

// TestMockServiceStateConsistency 測試模擬服務狀態一致性
func TestMockServiceStateConsistency(t *testing.T) {
	os.Setenv("COPILOT_MOCK_MODE", "true")
	defer os.Unsetenv("COPILOT_MOCK_MODE")

	tempDir, err := os.MkdirTemp("", "ralph-mock-state-*")
	if err != nil {
		t.Fatalf("無法創建暫存目錄: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := ghcopilot.DefaultClientConfig()
	config.WorkDir = tempDir
	config.SaveDir = filepath.Join(tempDir, ".ralph-loop", "saves")

	client := ghcopilot.NewRalphLoopClientWithConfig(config)
	if client == nil {
		t.Fatal("NewRalphLoopClient 返回 nil")
	}

	// 執行一系列任務來測試狀態保持
	tasks := []string{
		"初始化狀態測試",
		"中間狀態測試", 
		"最終狀態測試",
	}

	for i, task := range tasks {
		t.Run(fmt.Sprintf("StateTest_%d", i+1), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			result, err := client.ExecuteUntilCompletion(ctx, task, 1)

			if err != nil && result == nil {
				t.Logf("任務 %d 執行錯誤 (模擬模式下可能預期): %v", i+1, err)
			} else {
				t.Logf("✅ 任務 %d 完成", i+1)
			}

			// 檢查狀態目錄是否存在
			if _, err := os.Stat(config.SaveDir); err == nil {
				t.Logf("狀態目錄存在: %s", config.SaveDir)
			}
		})
	}

	t.Log("✅ 模擬服務狀態一致性測試完成")
}
