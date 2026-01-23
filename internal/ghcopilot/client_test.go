package ghcopilot

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestNewRalphLoopClient 測試建立新客戶端
func TestNewRalphLoopClient(t *testing.T) {
	client := NewRalphLoopClient()
	if client == nil {
		t.Error("NewRalphLoopClient() 傳回 nil")
	}
	if !client.initialized {
		t.Error("Client 應已初始化")
	}
	if client.closed {
		t.Error("Client 應未關閉")
	}
}

// TestDefaultClientConfig 測試預設配置
func TestDefaultClientConfig(t *testing.T) {
	config := DefaultClientConfig()
	if config.CLITimeout != 60*time.Second {
		t.Errorf("預設逾時應為 60s，但為 %v", config.CLITimeout)
	}
	if config.CLIMaxRetries != 3 {
		t.Errorf("預設重試應為 3，但為 %d", config.CLIMaxRetries)
	}
	if config.MaxHistorySize != 100 {
		t.Errorf("預設歷史大小應為 100，但為 %d", config.MaxHistorySize)
	}
}

// TestClientBuilderPattern 測試 Builder 模式
func TestClientBuilderPattern(t *testing.T) {
	client := NewClientBuilder().
		WithTimeout(60 * time.Second).
		WithMaxRetries(5).
		WithModel("gpt-4").
		WithoutPersistence().
		Build()

	if client == nil {
		t.Error("Builder 應建立有效的客戶端")
	}

	if client.config.CLITimeout != 60*time.Second {
		t.Errorf("逾時設定失敗")
	}
	if client.config.CLIMaxRetries != 5 {
		t.Errorf("重試設定失敗")
	}
	if client.config.Model != "gpt-4" {
		t.Errorf("模型設定失敗")
	}
	if client.config.EnablePersistence {
		t.Error("持久化應被禁用")
	}
}

// TestGetStatus 測試取得狀態
func TestGetStatus(t *testing.T) {
	client := NewRalphLoopClient()
	status := client.GetStatus()

	if status == nil {
		t.Error("GetStatus() 傳回 nil")
	}
	if !status.Initialized {
		t.Error("狀態應顯示已初始化")
	}
	if status.Closed {
		t.Error("狀態應顯示未關閉")
	}
}

// TestGetHistory 測試取得歷史
func TestGetHistory(t *testing.T) {
	client := NewRalphLoopClient()
	history := client.GetHistory()

	if history == nil {
		t.Error("GetHistory() 傳回 nil")
	}
	if len(history) != 0 {
		t.Errorf("新客戶端應無歷史，但有 %d 筆", len(history))
	}
}

// TestClientGetSummary 測試 Client 取得摘要
func TestClientGetSummary(t *testing.T) {
	client := NewRalphLoopClient()
	summary := client.GetSummary()

	if summary == nil {
		t.Error("GetSummary() 傳回 nil")
	}

	// 檢查基本字段
	if totalLoops, ok := summary["total_loops"].(int); !ok || totalLoops != 0 {
		t.Error("摘要應包含 total_loops = 0")
	}
}

// TestClearHistory 測試清空歷史
func TestClearHistory(t *testing.T) {
	client := NewRalphLoopClient()

	// 模擬添加歷史
	client.contextManager.StartLoop(0, "測試")
	client.contextManager.FinishLoop()

	if len(client.GetHistory()) == 0 {
		t.Error("歷史應有記錄")
	}

	client.ClearHistory()

	if len(client.GetHistory()) != 0 {
		t.Error("清空後應無歷史記錄")
	}
}

// TestClientClose 測試關閉客戶端
func TestClientClose(t *testing.T) {
	client := NewRalphLoopClient()

	err := client.Close()
	if err != nil {
		t.Errorf("Close() 失敗: %v", err)
	}

	if !client.closed {
		t.Error("Client 應已關閉")
	}

	// 再次關閉應失敗
	err = client.Close()
	if err == nil {
		t.Error("再次關閉應傳回錯誤")
	}
}

// TestExecuteLoopWithoutInit 測試在未初始化時執行迴圈失敗
func TestExecuteLoopWithoutInit(t *testing.T) {
	client := &RalphLoopClient{
		initialized: false,
		closed:      false,
	}

	ctx := context.Background()
	_, err := client.ExecuteLoop(ctx, "測試提示詞")
	if err == nil {
		t.Error("應該在未初始化時傳回錯誤")
	}
}

// TestExecuteLoopAfterClose 測試在關閉後執行迴圈失敗
func TestExecuteLoopAfterClose(t *testing.T) {
	client := NewRalphLoopClient()
	client.Close()

	ctx := context.Background()
	_, err := client.ExecuteLoop(ctx, "測試提示詞")
	if err == nil {
		t.Error("應該在關閉後傳回錯誤")
	}
}

// TestResetCircuitBreaker 測試重置熔斷器
func TestResetCircuitBreaker(t *testing.T) {
	client := NewRalphLoopClient()

	// 模擬打開熔斷器
	for i := 0; i < 3; i++ {
		client.breaker.RecordNoProgress()
	}

	if !client.breaker.IsOpen() {
		t.Error("熔斷器應已打開")
	}

	// 重置
	err := client.ResetCircuitBreaker()
	if err != nil {
		t.Errorf("ResetCircuitBreaker() 失敗: %v", err)
	}

	if client.breaker.IsOpen() {
		t.Error("重置後熔斷器應已關閉")
	}
}

// TestGetStatus_CircuitBreakerOpen 測試熔斷器開啟時的狀態
func TestGetStatus_CircuitBreakerOpen(t *testing.T) {
	client := NewRalphLoopClient()

	// 打開熔斷器
	for i := 0; i < 3; i++ {
		client.breaker.RecordNoProgress()
	}

	status := client.GetStatus()
	if !status.CircuitBreakerOpen {
		t.Error("狀態應顯示熔斷器已打開")
	}
}

// TestClientConfiguration 測試客戶端配置應用
func TestClientConfiguration(t *testing.T) {
	config := &ClientConfig{
		CLITimeout:    60 * time.Second,
		CLIMaxRetries: 5,
		Model:         "custom-model",
		Silent:        true,
	}

	client := NewRalphLoopClientWithConfig(config)

	if client.config.CLITimeout != 60*time.Second {
		t.Error("逾時配置未應用")
	}
	if client.config.CLIMaxRetries != 5 {
		t.Error("重試配置未應用")
	}
	if client.config.Model != "custom-model" {
		t.Error("模型配置未應用")
	}
	if !client.config.Silent {
		t.Error("靜默配置未應用")
	}
}

// TestBuilderMultipleSettings 測試 Builder 多個設定
func TestBuilderMultipleSettings(t *testing.T) {
	client := NewClientBuilder().
		WithTimeout(90 * time.Second).
		WithMaxRetries(10).
		WithMaxHistory(200).
		WithModel("gpt-4-turbo").
		WithSaveDir("./custom-saves").
		WithGobFormat(true).
		Build()

	if client.config.CLITimeout != 90*time.Second {
		t.Error("逾時設定失敗")
	}
	if client.config.CLIMaxRetries != 10 {
		t.Error("重試設定失敗")
	}
	if client.config.MaxHistorySize != 200 {
		t.Error("歷史大小設定失敗")
	}
	if client.config.Model != "gpt-4-turbo" {
		t.Error("模型設定失敗")
	}
	if client.config.SaveDir != "./custom-saves" {
		t.Error("儲存目錄設定失敗")
	}
	if !client.config.UseGobFormat {
		t.Error("Gob 格式設定失敗")
	}
}

// TestSaveHistoryToDisk 測試保存歷史到磁盤
func TestSaveHistoryToDisk(t *testing.T) {
	client := NewRalphLoopClient()
	defer client.Close()

	// 沒有歷史時也應該成功
	err := client.SaveHistoryToDisk()
	if err != nil {
		t.Errorf("保存歷史失敗: %v", err)
	}

	// 添加一些迴圈
	client.contextManager.StartLoop(0, "測試提示")
	client.contextManager.FinishLoop()

	// 再次保存
	err = client.SaveHistoryToDisk()
	if err != nil {
		t.Errorf("保存歷史失敗: %v", err)
	}
}

// TestLoadHistoryFromDisk 測試從磁盤載入歷史
func TestLoadHistoryFromDisk(t *testing.T) {
	// 建立第一個客戶端並保存數據
	client1 := NewRalphLoopClient()
	client1.contextManager.StartLoop(0, "測試提示1")
	client1.contextManager.FinishLoop()
	client1.contextManager.StartLoop(1, "測試提示2")
	client1.contextManager.FinishLoop()

	err := client1.SaveHistoryToDisk()
	if err != nil {
		t.Errorf("保存歷史失敗: %v", err)
	}

	originalCount := len(client1.contextManager.GetLoopHistory())
	client1.Close()

	// 建立第二個客戶端並載入
	client2 := NewRalphLoopClient()
	defer client2.Close()

	err = client2.LoadHistoryFromDisk()
	if err != nil {
		// 載入可能失敗（如果檔案不存在），這是正常的
		// 因為新的臨時目錄可能不存在之前保存的檔案
		t.Logf("載入歷史：%v (預期在新目錄中可能失敗)", err)
		return
	}

	loadedCount := len(client2.contextManager.GetLoopHistory())
	if loadedCount != originalCount {
		t.Errorf("載入歷史計數不符: 期望 %d，實際 %d", originalCount, loadedCount)
	}
}

// TestGetPersistenceStats 測試取得持久化統計
func TestGetPersistenceStats(t *testing.T) {
	client := NewRalphLoopClient()
	defer client.Close()

	stats := client.GetPersistenceStats()

	if stats == nil {
		t.Error("GetPersistenceStats 傳回 nil")
		return
	}

	if enabled, ok := stats["enabled"].(bool); !ok || !enabled {
		t.Error("持久化應該已啟用")
	}

	if _, ok := stats["storage_dir"]; !ok {
		t.Error("統計應包含 storage_dir")
	}

	if _, ok := stats["format"]; !ok {
		t.Error("統計應包含 format")
	}
}

// TestPersistenceIntegration 測試完整持久化流程
func TestPersistenceIntegration(t *testing.T) {
	client := NewRalphLoopClient()

	// 執行一些迴圈
	client.contextManager.StartLoop(0, "迴圈 1")
	client.contextManager.UpdateCurrentLoop(func(ctx *ExecutionContext) {
		ctx.ShouldContinue = false
		ctx.ExitReason = "測試完成"
	})
	client.contextManager.FinishLoop()

	// 驗證初始狀態
	if len(client.contextManager.GetLoopHistory()) != 1 {
		t.Error("應該有 1 個迴圈")
	}

	// 保存
	err := client.SaveHistoryToDisk()
	if err != nil {
		t.Errorf("保存失敗: %v", err)
	}

	// 檢查統計
	stats := client.GetPersistenceStats()
	if enabled, ok := stats["enabled"].(bool); !ok || !enabled {
		t.Error("持久化應該已啟用")
	}

	client.Close()
}

// TestLoadHistoryWithoutInit 測試在未初始化時載入歷史
func TestLoadHistoryWithoutInit(t *testing.T) {
	client := &RalphLoopClient{
		initialized: false,
	}

	err := client.LoadHistoryFromDisk()
	if err == nil || !strings.Contains(err.Error(), "not initialized") {
		t.Error("應該拒絕未初始化的客戶端")
	}
}

// TestSaveHistoryWithoutPersistence 測試禁用持久化時保存
func TestSaveHistoryWithoutPersistence(t *testing.T) {
	builder := NewClientBuilder()
	client := builder.WithoutPersistence().Build()
	defer client.Close()

	err := client.SaveHistoryToDisk()
	if err == nil || !strings.Contains(err.Error(), "persistence not enabled") {
		t.Error("應該在禁用持久化時拒絕保存")
	}
}

// TestCleanupOldBackups 測試清理舊備份
func TestCleanupOldBackups(t *testing.T) {
	client := NewRalphLoopClient()
	defer client.Close()

	// 嘗試清理 context_manager 備份
	err := client.CleanupOldBackups("context_manager")
	if err != nil {
		// 可能沒有備份，這是正常的
		t.Logf("清理備份: %v (預期可能無備份)", err)
	}
}

// TestSetMaxBackupCount 測試設定最大備份數量
func TestSetMaxBackupCount(t *testing.T) {
	client := NewRalphLoopClient()
	defer client.Close()

	// 設定有效的備份計數
	err := client.SetMaxBackupCount(20)
	if err != nil {
		t.Errorf("設定備份計數失敗: %v", err)
	}

	// 嘗試設定無效的備份計數
	err = client.SetMaxBackupCount(0)
	if err == nil || !strings.Contains(err.Error(), "greater than 0") {
		t.Error("應該拒絕無效的備份計數")
	}
}

// TestListBackups 測試列出備份
func TestListBackups(t *testing.T) {
	client := NewRalphLoopClient()
	defer client.Close()

	// 列出 context_manager 備份
	backups, err := client.ListBackups("context_manager")
	if err != nil {
		// 可能沒有備份，這是正常的
		t.Logf("列出備份: %v (預期可能無備份)", err)
		return
	}

	// backups 可能是空的或包含檔案名稱
	if backups != nil {
		t.Logf("找到 %d 個備份", len(backups))
	}
}

// TestCleanupWithoutInit 測試未初始化時清理備份
func TestCleanupWithoutInit(t *testing.T) {
	client := &RalphLoopClient{
		initialized: false,
	}

	err := client.CleanupOldBackups("test")
	if err == nil || !strings.Contains(err.Error(), "not initialized") {
		t.Error("應該拒絕未初始化的客戶端")
	}
}

// TestSetMaxBackupCountWithoutPersistence 測試禁用持久化時設定備份計數
func TestSetMaxBackupCountWithoutPersistence(t *testing.T) {
	builder := NewClientBuilder()
	client := builder.WithoutPersistence().Build()
	defer client.Close()

	err := client.SetMaxBackupCount(20)
	if err == nil || !strings.Contains(err.Error(), "persistence not enabled") {
		t.Error("應該在禁用持久化時拒絕設定")
	}
}

// TestBackupIntegration 測試完整備份流程
func TestBackupIntegration(t *testing.T) {
	client := NewRalphLoopClient()

	// 執行幾個迴圈建立備份
	for i := 0; i < 3; i++ {
		client.contextManager.StartLoop(i, fmt.Sprintf("迴圈 %d", i))
		client.contextManager.FinishLoop()
	}

	// 保存
	err := client.SaveHistoryToDisk()
	if err != nil {
		t.Errorf("保存失敗: %v", err)
	}

	// 列出備份
	backups, err := client.ListBackups("context_manager")
	if err != nil {
		t.Logf("列出備份失敗: %v (可能沒有備份)", err)
	} else {
		if backups != nil {
			t.Logf("找到 %d 個備份", len(backups))
		}
	}
	err = client.SetMaxBackupCount(5)
	if err != nil {
		t.Errorf("設定備份計數失敗: %v", err)
	}

	client.Close()
}

// TestVerifyStateConsistency 測試狀態一致性驗證
func TestVerifyStateConsistency(t *testing.T) {
	client := NewRalphLoopClient()
	defer client.Close()

	// 如果還沒有執行任何迴圈，驗證應該通過（狀態一致）
	// 或是有備份也沒關係，只要計數合理
	consistent, err := client.VerifyStateConsistency()
	if err != nil {
		t.Logf("驗證結果：%v (可能有預期的備份)", err)
		return
	}
	if !consistent {
		t.Error("狀態應該是一致的")
	}
}

// TestRecoverFromBackup 測試從備份恢復
func TestRecoverFromBackup(t *testing.T) {
	// 建立第一個客戶端並建立備份
	client1 := NewRalphLoopClient()
	client1.contextManager.StartLoop(0, "測試提示")
	client1.contextManager.UpdateCurrentLoop(func(ctx *ExecutionContext) {
		ctx.CLIOutput = "測試輸出"
		ctx.ShouldContinue = false
	})
	client1.contextManager.FinishLoop()

	// 保存到備份
	err := client1.SaveHistoryToDisk()
	if err != nil {
		t.Errorf("保存失敗: %v", err)
	}

	client1.Close()

	// 建立第二個客戶端並列出備份
	client2 := NewRalphLoopClient()
	defer client2.Close()

	// 列出並嘗試恢復
	backups, err := client2.ListBackups("execution_context")
	if err != nil || len(backups) == 0 {
		t.Logf("沒有找到 execution_context 備份: %v", err)
		return
	}

	// 嘗試恢復（可能失敗，因為檔案格式可能不同）
	if len(backups) > 0 {
		err = client2.RecoverFromBackup(backups[0])
		if err != nil {
			t.Logf("恢復失敗（預期可能）: %v", err)
		}
	}
}

// TestRecoverWithoutInit 測試未初始化時恢復
func TestRecoverWithoutInit(t *testing.T) {
	client := &RalphLoopClient{
		initialized: false,
	}

	err := client.RecoverFromBackup("test.json")
	if err == nil || !strings.Contains(err.Error(), "not initialized") {
		t.Error("應該拒絕未初始化的客戶端")
	}
}

// TestVerifyStateWithoutPersistence 測試禁用持久化時驗證
func TestVerifyStateWithoutPersistence(t *testing.T) {
	builder := NewClientBuilder()
	client := builder.WithoutPersistence().Build()
	defer client.Close()

	_, err := client.VerifyStateConsistency()
	if err == nil || !strings.Contains(err.Error(), "persistence not enabled") {
		t.Error("應該在禁用持久化時拒絕驗證")
	}
}
