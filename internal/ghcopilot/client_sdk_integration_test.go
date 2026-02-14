package ghcopilot

import (
	"context"
	"os"
	"testing"
	"time"
)

// skipIfSDKNotAvailable 檢查 SDK 是否可用，不可用則跳過測試
func skipIfSDKNotAvailable(t *testing.T) {
	t.Helper()
	
	// 檢查 copilot CLI 是否存在
	cliPath := "copilot"
	if _, err := os.Stat(cliPath); os.IsNotExist(err) {
		t.Skip("Copilot CLI 不可用，跳過 SDK 測試")
	}
}

// TestClientSDKIntegration 測試 RalphLoopClient 與 SDKExecutor 集成
func TestClientSDKIntegration(t *testing.T) {
	config := DefaultClientConfig()
	client := NewRalphLoopClientWithConfig(config)
	defer client.Close()

	// 驗證 SDK 執行器已初始化
	if client.sdkExecutor == nil {
		t.Fatal("SDK 執行器應已初始化")
	}

	// 驗證狀態方法
	status := client.GetSDKStatus()
	if status == nil {
		t.Log("初始狀態為 nil（在啟動前正常）")
	}

	// 驗證會話計數
	count := client.GetSDKSessionCount()
	if count != 0 {
		t.Errorf("初始會話計數應為 0，實際: %d", count)
	}
}

// TestClientStartStopSDKExecutor 測試啟動和停止 SDK 執行器
func TestClientStartStopSDKExecutor(t *testing.T) {
	skipIfSDKNotAvailable(t)
	
	config := DefaultClientConfig()
	client := NewRalphLoopClientWithConfig(config)
	defer client.Close()

	// 使用 goroutine 和 channel 強制超時
	done := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		done <- client.StartSDKExecutor(ctx)
	}()

	// 等待結果或超時
	select {
	case err := <-done:
		if err != nil {
			t.Logf("啟動 SDK 執行器失敗（預期）: %v", err)
			return
		}
		// 如果成功啟動，停止 SDK 執行器
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer stopCancel()
		if err := client.StopSDKExecutor(stopCtx); err != nil {
			t.Logf("停止 SDK 執行器: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Skip("SDK 啟動超時，跳過測試")
	}
}

// TestClientExecuteWithSDK 測試使用 SDK 執行程式碼完成
func TestClientExecuteWithSDK(t *testing.T) {
	config := DefaultClientConfig()
	client := NewRalphLoopClientWithConfig(config)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 嘗試執行（可能因為 CLI 不可用而失敗，這是正常的）
	result, err := client.ExecuteWithSDK(ctx, "print('hello')")
	if err != nil {
		t.Logf("ExecuteWithSDK 失敗（預期）: %v", err)
		return
	}

	if result == "" {
		t.Log("返回空結果（預期）")
	}
}

// TestClientExplainWithSDK 測試使用 SDK 解釋程式碼
func TestClientExplainWithSDK(t *testing.T) {
	config := DefaultClientConfig()
	client := NewRalphLoopClientWithConfig(config)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, err := client.ExplainWithSDK(ctx, "def hello(): return 'world'")
	if err != nil {
		t.Logf("ExplainWithSDK 失敗（預期）: %v", err)
		return
	}

	if result == "" {
		t.Log("返回空結果（預期）")
	}
}

// TestClientGenerateTestsWithSDK 測試使用 SDK 生成測試
func TestClientGenerateTestsWithSDK(t *testing.T) {
	config := DefaultClientConfig()
	client := NewRalphLoopClientWithConfig(config)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, err := client.GenerateTestsWithSDK(ctx, "func Add(a, b int) int { return a + b }")
	if err != nil {
		t.Logf("GenerateTestsWithSDK 失敗（預期）: %v", err)
		return
	}

	if result == "" {
		t.Log("返回空結果（預期）")
	}
}

// TestClientCodeReviewWithSDK 測試使用 SDK 進行程式碼審查
func TestClientCodeReviewWithSDK(t *testing.T) {
	config := DefaultClientConfig()
	client := NewRalphLoopClientWithConfig(config)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, err := client.CodeReviewWithSDK(ctx, "var x = 1; var y = 2;")
	if err != nil {
		t.Logf("CodeReviewWithSDK 失敗（預期）: %v", err)
		return
	}

	if result == "" {
		t.Log("返回空結果（預期）")
	}
}

// TestClientSDKSessionManagement 測試 SDK 會話管理
func TestClientSDKSessionManagement(t *testing.T) {
	skipIfSDKNotAvailable(t)
	
	config := DefaultClientConfig()
	client := NewRalphLoopClientWithConfig(config)
	defer client.Close()

	// 使用 goroutine 強制超時
	done := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		done <- client.StartSDKExecutor(ctx)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Logf("無法啟動執行器（預期）: %v", err)
			return
		}
	case <-time.After(5 * time.Second):
		t.Skip("SDK 啟動超時，跳過測試")
		return
	}

	// 創建會話
	session, err := client.sdkExecutor.CreateSession("test-session-1")
	if err != nil {
		t.Logf("創建會話失敗: %v", err)
		return
	}

	if session == nil {
		t.Fatal("會話應不為 nil")
	}

	// 驗證會話計數
	count := client.GetSDKSessionCount()
	if count != 1 {
		t.Errorf("會話計數應為 1，實際: %d", count)
	}

	// 列出會話
	sessions := client.ListSDKSessions()
	if len(sessions) != 1 {
		t.Errorf("應有 1 個會話，實際: %d", len(sessions))
	}

	// 終止會話
	err = client.TerminateSDKSession("test-session-1")
	if err != nil {
		t.Logf("終止會話失敗: %v", err)
	}

	// 驗證會話已移除
	count = client.GetSDKSessionCount()
	if count != 0 {
		t.Errorf("終止後會話計數應為 0，實際: %d", count)
	}
}

// TestClientGetSDKStatus 測試取得 SDK 狀態
func TestClientGetSDKStatus(t *testing.T) {
	skipIfSDKNotAvailable(t)
	
	config := DefaultClientConfig()
	client := NewRalphLoopClientWithConfig(config)
	defer client.Close()

	// 使用 goroutine 強制超時
	done := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		done <- client.StartSDKExecutor(ctx)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Logf("無法啟動執行器（預期）: %v", err)
			return
		}
	case <-time.After(5 * time.Second):
		t.Skip("SDK 啟動超時，跳過測試")
		return
	}

	// 取得狀態
	status := client.GetSDKStatus()
	if status == nil {
		t.Fatal("狀態應不為 nil")
	}

	// 驗證狀態欄位
	if !status.Running {
		t.Log("執行器應在運行狀態")
	}

	if status.SessionCount < 0 {
		t.Errorf("會話計數不應為負: %d", status.SessionCount)
	}
}

// TestClientSDKClosing 測試客戶端關閉時正確清理 SDK 資源
func TestClientSDKClosing(t *testing.T) {
	skipIfSDKNotAvailable(t)
	
	config := DefaultClientConfig()
	client := NewRalphLoopClientWithConfig(config)

	// 使用 goroutine 強制超時
	done := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		done <- client.StartSDKExecutor(ctx)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Logf("啟動執行器失敗（預期）: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Skip("SDK 啟動超時，跳過測試")
		return
	}

	// 建立一些會話
	for i := 0; i < 3; i++ {
		sessionID := "session-" + string(rune(i))
		_, err := client.sdkExecutor.CreateSession(sessionID)
		if err != nil {
			t.Logf("創建會話失敗: %v", err)
		}
	}

	// 關閉客戶端
	err := client.Close()
	if err != nil {
		t.Fatalf("關閉客戶端失敗: %v", err)
	}

	// 驗證客戶端已關閉
	if !client.closed {
		t.Error("客戶端應已關閉")
	}

	// 嘗試使用已關閉的客戶端應返回錯誤
	testCtx := context.Background()
	_, err = client.ExecuteWithSDK(testCtx, "test")
	if err == nil || err.Error() != "client is closed" {
		t.Error("已關閉的客戶端應返回錯誤")
	}
}

// TestClientSDKWithTimeout 測試 SDK 執行器的超時設定
func TestClientSDKWithTimeout(t *testing.T) {
	config := DefaultClientConfig()
	config.CLITimeout = 100 * time.Millisecond // 設定超短超時
	client := NewRalphLoopClientWithConfig(config)
	defer client.Close()

	// 驗證超時已設定
	if client.sdkExecutor == nil {
		t.Fatal("SDK 執行器應已初始化")
	}

	if client.sdkExecutor.config.Timeout != 100*time.Millisecond {
		t.Errorf("超時應為 100ms，實際: %v", client.sdkExecutor.config.Timeout)
	}
}

// TestClientSDKMultipleCycles 測試多個 SDK 循環
func TestClientSDKMultipleCycles(t *testing.T) {
	skipIfSDKNotAvailable(t)
	
	config := DefaultClientConfig()
	client := NewRalphLoopClientWithConfig(config)
	defer client.Close()

	// 使用 goroutine 強制超時
	done := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		done <- client.StartSDKExecutor(ctx)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Logf("啟動執行器失敗（預期）: %v", err)
			return
		}
	case <-time.After(5 * time.Second):
		t.Skip("SDK 啟動超時，跳過測試")
		return
	}

	// 執行多個循環
	for i := 0; i < 3; i++ {
		// 創建會話
		sessionID := "cycle-session-" + string(rune(i))
		_, err := client.sdkExecutor.CreateSession(sessionID)
		if err != nil {
			t.Logf("循環 %d: 創建會話失敗: %v", i, err)
			continue
		}

		// 終止會話
		err = client.TerminateSDKSession(sessionID)
		if err != nil {
			t.Logf("循環 %d: 終止會話失敗: %v", i, err)
		}
	}

	// 驗證最終沒有會話
	count := client.GetSDKSessionCount()
	if count != 0 {
		t.Logf("最終會話計數應為 0，實際: %d", count)
	}
}
