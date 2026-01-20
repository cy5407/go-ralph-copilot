package ghcopilot

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestNewCLIExecutor 測試建立新的 CLI 執行器
func TestNewCLIExecutor(t *testing.T) {
	ce := NewCLIExecutor("/tmp")
	if ce == nil {
		t.Error("NewCLIExecutor() 傳回 nil")
	}
	if ce.timeout != 30*time.Second {
		t.Errorf("預設逾時應為 30s，但為 %v", ce.timeout)
	}
	if ce.workDir != "/tmp" {
		t.Errorf("工作目錄應為 /tmp，但為 %s", ce.workDir)
	}
}

// TestSetTimeout 測試設定逾時
func TestSetTimeout(t *testing.T) {
	ce := NewCLIExecutor("/tmp")
	ce.SetTimeout(60 * time.Second)
	if ce.timeout != 60*time.Second {
		t.Errorf("逾時應為 60s，但為 %v", ce.timeout)
	}
}

// TestSetMaxRetries 測試設定最大重試次數
func TestSetMaxRetries(t *testing.T) {
	ce := NewCLIExecutor("/tmp")
	ce.SetMaxRetries(5)
	if ce.maxRetries != 5 {
		t.Errorf("最大重試次數應為 5，但為 %d", ce.maxRetries)
	}
}

// TestValidateWorkDir 測試驗證工作目錄
func TestValidateWorkDir(t *testing.T) {
	// 測試有效的工作目錄
	wd, _ := os.Getwd()
	ce := NewCLIExecutor(wd)
	err := ce.ValidateWorkDir()
	if err != nil {
		t.Errorf("驗證現有目錄失敗: %v", err)
	}

	// 測試無效的工作目錄
	ce2 := NewCLIExecutor("/nonexistent/path/12345")
	err2 := ce2.ValidateWorkDir()
	if err2 == nil {
		t.Error("驗證不存在的目錄應傳回錯誤")
	}
}

// TestSetWorkDir 測試設定工作目錄
func TestSetWorkDir(t *testing.T) {
	wd, _ := os.Getwd()
	ce := NewCLIExecutor("/tmp")

	err := ce.SetWorkDir(wd)
	if err != nil {
		t.Errorf("設定現有目錄失敗: %v", err)
	}

	if ce.workDir != wd {
		t.Errorf("工作目錄應為 %s，但為 %s", wd, ce.workDir)
	}

	// 測試設定無效的路徑
	err2 := ce.SetWorkDir("/nonexistent/path/12345")
	if err2 == nil {
		t.Error("設定不存在的路徑應傳回錯誤")
	}
}

// TestGetWorkDir 測試取得工作目錄
func TestGetWorkDir(t *testing.T) {
	ce := NewCLIExecutor("/tmp")
	wd := ce.GetWorkDir()
	if wd != "/tmp" {
		t.Errorf("工作目錄應為 /tmp，但為 %s", wd)
	}

	// 測試空工作目錄（應傳回當前目錄）
	ce2 := NewCLIExecutor("")
	wd2 := ce2.GetWorkDir()
	if wd2 == "" {
		t.Error("取得空工作目錄應傳回有效的目錄")
	}
}

// TestMockExecute 測試模擬執行
func TestMockExecute(t *testing.T) {
	os.Setenv("COPILOT_MOCK_MODE", "true")
	defer os.Unsetenv("COPILOT_MOCK_MODE")

	wd, _ := os.Getwd()
	ce := NewCLIExecutor(wd)

	ctx := context.Background()
	result, err := ce.SuggestShellCommand(ctx, "列出所有檔案")

	if err != nil {
		t.Errorf("模擬執行失敗: %v", err)
	}

	if !result.Success {
		t.Error("模擬執行應成功")
	}

	if result.Stdout == "" {
		t.Error("模擬執行應產生輸出")
	}

	if result.ExitCode != 0 {
		t.Errorf("模擬執行的退出碼應為 0，但為 %d", result.ExitCode)
	}
}

// TestExecutionResult 測試 ExecutionResult 結構
func TestExecutionResult(t *testing.T) {
	result := &ExecutionResult{
		Command:       "test",
		Stdout:        "output",
		Stderr:        "",
		ExitCode:      0,
		Success:       true,
		Error:         nil,
		ExecutionTime: 100 * time.Millisecond,
	}

	if result.Command != "test" {
		t.Error("Command 應為 'test'")
	}

	if result.Stdout != "output" {
		t.Error("Stdout 應為 'output'")
	}

	if !result.Success {
		t.Error("Success 應為 true")
	}
}

// TestGenerateMockResponse 測試模擬響應生成
func TestGenerateMockResponse(t *testing.T) {
	ce := NewCLIExecutor("/tmp")
	response := ce.generateMockResponse("suggest", []string{
		"-p", "測試描述",
	})

	if len(response) == 0 {
		t.Error("模擬響應應不為空")
	}

	if !contains(response, "COPILOT_STATUS") {
		t.Error("模擬響應應包含 COPILOT_STATUS")
	}

	if !contains(response, "測試描述") {
		t.Error("模擬響應應包含描述")
	}
}
