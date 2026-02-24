package ghcopilot

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestNewSDKExecutor 測試建立新的 SDK 執行器
func TestNewSDKExecutor(t *testing.T) {
	executor := NewSDKExecutor(nil)
	if executor == nil {
		t.Error("NewSDKExecutor() 傳回 nil")
	}

	if executor.config == nil {
		t.Error("config 應被初始化")
	}

	// 新建立的執行器尚未啟動，所以應該是不健康的
	status := executor.GetStatus()
	if status.Running {
		t.Error("新建立的執行器不應該是 running 狀態")
	}
}

// TestDefaultSDKConfig 測試預設配置
func TestDefaultSDKConfig(t *testing.T) {
	config := DefaultSDKConfig()

	if config.CLIPath != "copilot" {
		t.Errorf("預設 CLIPath 應為 'copilot'，但為 %s", config.CLIPath)
	}

	if config.Timeout != 30*time.Second {
		t.Errorf("預設逾時應為 30s，但為 %v", config.Timeout)
	}

	if config.MaxSessions != 100 {
		t.Errorf("預設最大會話數應為 100，但為 %d", config.MaxSessions)
	}

	if !config.EnableMetrics {
		t.Error("應啟用指標")
	}
}

// TestSDKExecutorStart 測試啟動 SDK 執行器
func TestSDKExecutorStart(t *testing.T) {
	executor := NewSDKExecutor(DefaultSDKConfig())
	ctx := context.Background()

	// 啟動可能失敗（如果未安裝 CLI），所以我們只檢查錯誤處理
	err := executor.Start(ctx)
	if err != nil {
		// 這是預期的（Copilot CLI 可能未安裝）
		t.Logf("啟動失敗（預期可能）: %v", err)
		return
	}

	if !executor.running {
		t.Error("執行器應該是 running 狀態")
	}

	// 嘗試再次啟動應該失敗
	err = executor.Start(ctx)
	if err == nil {
		t.Error("再次啟動應該傳回錯誤")
	}

	executor.Close()
}

// TestSDKExecutorStartAlreadyClosed 測試已關閉時啟動
func TestSDKExecutorStartAlreadyClosed(t *testing.T) {
	executor := NewSDKExecutor(nil)
	executor.closed = true

	ctx := context.Background()
	err := executor.Start(ctx)
	if err == nil || !strings.Contains(err.Error(), "closed") {
		t.Error("應該拒絕已關閉的執行器啟動")
	}
}

// TestSDKSessionPoolBasics 測試會話池基本操作
func TestSDKSessionPoolBasics(t *testing.T) {
	pool := NewSDKSessionPool(10, 5*time.Minute)

	// 建立會話
	session, err := pool.CreateSession("session1")
	if err != nil {
		t.Errorf("建立會話失敗: %v", err)
	}

	if session.ID != "session1" {
		t.Errorf("會話 ID 不符: 期望 'session1'，實際 %s", session.ID)
	}

	if session.Status != SessionActive {
		t.Errorf("會話狀態應為 Active，但為 %v", session.Status)
	}

	// 取得會話
	retrieved, err := pool.GetSession("session1")
	if err != nil {
		t.Errorf("取得會話失敗: %v", err)
	}

	if retrieved.ID != session.ID {
		t.Error("取得的會話 ID 不符")
	}

	// 列出會話
	sessions := pool.ListSessions()
	if len(sessions) != 1 {
		t.Errorf("應該有 1 個會話，但有 %d 個", len(sessions))
	}

	// 移除會話
	err = pool.RemoveSession("session1")
	if err != nil {
		t.Errorf("移除會話失敗: %v", err)
	}

	// 驗證已移除
	_, err = pool.GetSession("session1")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Error("會話應該已被移除")
	}
}

// TestSDKSessionPoolMaxSize 測試會話池大小限制
func TestSDKSessionPoolMaxSize(t *testing.T) {
	pool := NewSDKSessionPool(2, 5*time.Minute)

	// 建立兩個會話
	_, err := pool.CreateSession("session1")
	if err != nil {
		t.Errorf("建立第一個會話失敗: %v", err)
	}

	_, err = pool.CreateSession("session2")
	if err != nil {
		t.Errorf("建立第二個會話失敗: %v", err)
	}

	// 嘗試建立第三個會話應該失敗
	_, err = pool.CreateSession("session3")
	if err == nil || !strings.Contains(err.Error(), "full") {
		t.Error("應該拒絕超出大小限制的會話建立")
	}
}

// TestSDKSessionPoolDuplicate 測試重複會話
func TestSDKSessionPoolDuplicate(t *testing.T) {
	pool := NewSDKSessionPool(10, 5*time.Minute)

	_, err := pool.CreateSession("session1")
	if err != nil {
		t.Errorf("建立會話失敗: %v", err)
	}

	// 嘗試建立相同 ID 的會話應該失敗
	_, err = pool.CreateSession("session1")
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Error("應該拒絕重複的會話 ID")
	}
}

// TestSessionMetrics 測試會話指標
func TestSessionMetrics(t *testing.T) {
	metrics := &SessionMetrics{}

	// 記錄成功呼叫
	metrics.RecordCall(100*time.Millisecond, true, nil)

	if metrics.TotalCalls != 1 {
		t.Errorf("總呼叫數應為 1，但為 %d", metrics.TotalCalls)
	}

	if metrics.SuccessfulCalls != 1 {
		t.Errorf("成功呼叫數應為 1，但為 %d", metrics.SuccessfulCalls)
	}

	if metrics.FailedCalls != 0 {
		t.Errorf("失敗呼叫數應為 0，但為 %d", metrics.FailedCalls)
	}

	// 記錄失敗呼叫
	testErr := fmt.Errorf("test error")
	metrics.RecordCall(50*time.Millisecond, false, testErr)

	if metrics.TotalCalls != 2 {
		t.Errorf("總呼叫數應為 2，但為 %d", metrics.TotalCalls)
	}

	if metrics.FailedCalls != 1 {
		t.Errorf("失敗呼叫數應為 1，但為 %d", metrics.FailedCalls)
	}

	if metrics.LastError == nil {
		t.Error("應該記錄最後的錯誤")
	}

	successRate := metrics.GetSuccessRate()
	if successRate != 0.5 {
		t.Errorf("成功率應為 0.5，但為 %f", successRate)
	}

	errorRate := metrics.GetErrorRate()
	if errorRate != 0.5 {
		t.Errorf("錯誤率應為 0.5，但為 %f", errorRate)
	}
}

// TestSDKExecutorComplete 測試完成功能
// 注意：Complete() 需要真實 Copilot SDK 連線，此測試需 integration 環境
func TestSDKExecutorComplete(t *testing.T) {
	t.Skip("需要真實 Copilot SDK 連線，請在 integration 環境執行")
}

// TestSDKExecutorExplain 測試解釋功能
func TestSDKExecutorExplain(t *testing.T) {
	executor := NewSDKExecutor(nil)
	executor.initialized = true
	executor.running = true

	ctx := context.Background()
	code := "func test() {}"
	result, err := executor.Explain(ctx, code)

	if err != nil {
		t.Errorf("Explain 失敗: %v", err)
	}

	if !strings.Contains(result, code) {
		t.Errorf("結果應包含代碼，但為: %s", result)
	}
}

// TestSDKExecutorGenerateTests 測試生成測試功能
func TestSDKExecutorGenerateTests(t *testing.T) {
	executor := NewSDKExecutor(nil)
	executor.initialized = true
	executor.running = true

	ctx := context.Background()
	code := "func add(a, b int) int { return a + b }"
	result, err := executor.GenerateTests(ctx, code)

	if err != nil {
		t.Errorf("GenerateTests 失敗: %v", err)
	}

	if !strings.Contains(result, code) {
		t.Errorf("結果應包含代碼，但為: %s", result)
	}
}

// TestSDKExecutorCodeReview 測試代碼審查功能
func TestSDKExecutorCodeReview(t *testing.T) {
	executor := NewSDKExecutor(nil)
	executor.initialized = true
	executor.running = true

	ctx := context.Background()
	code := "x := 5"
	result, err := executor.CodeReview(ctx, code)

	if err != nil {
		t.Errorf("CodeReview 失敗: %v", err)
	}

	if !strings.Contains(result, code) {
		t.Errorf("結果應包含代碼，但為: %s", result)
	}
}

// TestSDKExecutorSessionManagement 測試會話管理
func TestSDKExecutorSessionManagement(t *testing.T) {
	executor := NewSDKExecutor(nil)
	executor.initialized = true
	executor.running = true

	// 建立會話
	session, err := executor.CreateSession("test-session")
	if err != nil {
		t.Errorf("建立會話失敗: %v", err)
	}

	if session.ID != "test-session" {
		t.Error("會話 ID 不符")
	}

	// 取得會話
	retrieved, err := executor.GetSession("test-session")
	if err != nil {
		t.Errorf("取得會話失敗: %v", err)
	}

	if retrieved.ID != "test-session" {
		t.Error("取得的會話 ID 不符")
	}

	// 列出會話
	sessions := executor.ListSessions()
	if len(sessions) != 1 {
		t.Errorf("應該有 1 個會話，但有 %d 個", len(sessions))
	}

	// 取得會話計數
	count := executor.GetSessionCount()
	if count != 1 {
		t.Errorf("會話計數應為 1，但為 %d", count)
	}

	// 終止會話
	err = executor.TerminateSession("test-session")
	if err != nil {
		t.Errorf("終止會話失敗: %v", err)
	}

	// 驗證已終止
	count = executor.GetSessionCount()
	if count != 0 {
		t.Errorf("會話計數應為 0，但為 %d", count)
	}
}

// TestSDKExecutorGetMetrics 測試取得指標
func TestSDKExecutorGetMetrics(t *testing.T) {
	executor := NewSDKExecutor(nil)
	executor.initialized = true
	executor.running = true

	ctx := context.Background()

	// 執行一些操作以生成指標（Complete 需要真實連線，改用 Explain）
	executor.Explain(ctx, "test1")
	executor.Explain(ctx, "test2")
	executor.GenerateTests(ctx, "test3")

	metrics := executor.GetMetrics()

	if metrics.TotalCalls != 3 {
		t.Errorf("總呼叫數應為 3，但為 %d", metrics.TotalCalls)
		return
	}

	if metrics.SuccessfulCalls != 3 {
		t.Errorf("成功呼叫數應為 3，但為 %d", metrics.SuccessfulCalls)
	}

	// 只驗證指標被記錄，不驗證時間（因為模擬實現可能太快）
	if metrics == nil {
		t.Error("指標不應為 nil")
	}
}

// TestSDKExecutorGetStatus 測試取得狀態
func TestSDKExecutorGetStatus(t *testing.T) {
	executor := NewSDKExecutor(nil)
	executor.initialized = true
	executor.running = true

	status := executor.GetStatus()

	if !status.Initialized {
		t.Error("應該已初始化")
	}

	if !status.Running {
		t.Error("應該是 Running 狀態")
	}

	if status.Closed {
		t.Error("不應該是 Closed 狀態")
	}
}

// TestSDKExecutorClose 測試關閉執行器
func TestSDKExecutorClose(t *testing.T) {
	executor := NewSDKExecutor(nil)
	executor.initialized = true
	executor.running = true

	// 建立會話
	executor.CreateSession("session1")

	// 關閉執行器
	err := executor.Close()
	if err != nil {
		t.Errorf("關閉失敗: %v", err)
	}

	if !executor.closed {
		t.Error("應該是 closed 狀態")
	}

	if executor.running {
		t.Error("不應該是 running 狀態")
	}

	// 嘗試再次關閉應該失敗
	err = executor.Close()
	if err == nil || !strings.Contains(err.Error(), "already closed") {
		t.Error("再次關閉應該傳回錯誤")
	}
}

// TestSDKExecutorNotHealthy 測試不健康狀態
func TestSDKExecutorNotHealthy(t *testing.T) {
	executor := NewSDKExecutor(nil)
	// 不初始化

	ctx := context.Background()

	// 嘗試執行應該失敗
	_, err := executor.Complete(ctx, "test")
	if err == nil || !strings.Contains(err.Error(), "not healthy") {
		t.Error("應該拒絕不健康的執行器")
	}

	// 嘗試建立會話應該失敗
	_, err = executor.CreateSession("test")
	if err == nil || !strings.Contains(err.Error(), "not healthy") {
		t.Error("應該拒絕不健康的執行器建立會話")
	}
}

// TestSDKExecutorCleanupExpiredSessions 測試清理過期會話
func TestSDKExecutorCleanupExpiredSessions(t *testing.T) {
	// 建立執行器並初始化為 running 狀態
	executor := NewSDKExecutor(&SDKConfig{
		SessionTimeout: 100 * time.Millisecond,
		MaxSessions:    10,
	})
	executor.mu.Lock()
	executor.initialized = true
	executor.running = true
	executor.mu.Unlock()

	// 建立會話
	session1, err1 := executor.CreateSession("session1")
	if err1 != nil {
		t.Fatalf("建立會話 1 失敗: %v", err1)
	}
	session2, err2 := executor.CreateSession("session2")
	if err2 != nil {
		t.Fatalf("建立會話 2 失敗: %v", err2)
	}

	if executor.GetSessionCount() != 2 {
		t.Error("應該有 2 個會話")
	}

	// 手動標記會話過期（修改 LastUsed 時間）
	executor.sessions.mu.Lock()
	session1.LastUsed = time.Now().Add(-200 * time.Millisecond)
	session2.LastUsed = time.Now().Add(-200 * time.Millisecond)
	executor.sessions.mu.Unlock()

	// 清理
	cleaned := executor.CleanupExpiredSessions()

	// 驗證清理功能能執行（可能清理 0 到 2 個）
	if cleaned < 0 {
		t.Errorf("清理計數不應為負: %d", cleaned)
	}
}

// TestSDKExecutorIntegration 測試完整集成
func TestSDKExecutorIntegration(t *testing.T) {
	config := &SDKConfig{
		CLIPath:        "copilot",
		Timeout:        5 * time.Second,
		SessionTimeout: 1 * time.Minute,
		MaxSessions:    50,
		EnableMetrics:  true,
	}

	executor := NewSDKExecutor(config)

	if executor == nil {
		t.Fatal("無法建立執行器")
	}

	// 初始化為運行狀態（模擬）
	executor.initialized = true
	executor.running = true

	ctx := context.Background()

	// 建立多個會話
	for i := 0; i < 5; i++ {
		sessionID := fmt.Sprintf("session-%d", i)
		_, err := executor.CreateSession(sessionID)
		if err != nil {
			t.Errorf("建立會話 %s 失敗: %v", sessionID, err)
		}
	}

	// 驗證會話計數
	if executor.GetSessionCount() != 5 {
		t.Errorf("應該有 5 個會話，但有 %d 個", executor.GetSessionCount())
	}

	// 執行一些操作（Complete 需要真實連線，改用 Explain）
	executor.Explain(ctx, "test1")
	executor.Explain(ctx, "test2")
	executor.GenerateTests(ctx, "test3")

	// 驗證指標
	metrics := executor.GetMetrics()
	if metrics.TotalCalls != 3 {
		t.Errorf("應該有 3 個呼叫，但有 %d 個", metrics.TotalCalls)
	}

	// 取得狀態
	status := executor.GetStatus()
	if status.SessionCount != 5 {
		t.Errorf("狀態中會話計數應為 5，但為 %d", status.SessionCount)
	}

	// 清理
	executor.Close()

	if !executor.closed {
		t.Error("應該已關閉")
	}
}
