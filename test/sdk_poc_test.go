package test

import (
	"fmt"
	"os"
	"testing"
	"time"

	copilot "github.com/github/copilot-sdk/go"
)

// TestSDKBasicConnection 測試基本連線
//
// 注意 (2026-01-21 更新)：
//   - 此測試使用舊版 SDK (github.com/github/copilot-sdk/go)
//   - 建議遷移至新版 SDK (github.com/github/copilot-cli-sdk-go)
//   - 新版 CLI 直接使用 "copilot" 命令，無需 wrapper
func TestSDKBasicConnection(t *testing.T) {
	t.Skip("跳過: 舊版 SDK 與新版 CLI 協議不相容，等待遷移至新版 SDK")
	// 新版獨立 Copilot CLI 直接使用 "copilot" 命令
	cliPath := os.Getenv("COPILOT_CLI_PATH")
	if cliPath == "" {
		// 預設使用新版獨立 CLI
		cliPath = "copilot"
	}

	t.Logf("使用 CLI 路徑: %s", cliPath)
	t.Log("注意: 請確保已安裝新版獨立 Copilot CLI (winget install GitHub.Copilot)")

	client := copilot.NewClient(&copilot.ClientOptions{
		CLIPath:  cliPath,
		LogLevel: "info",
	})

	if client == nil {
		t.Fatal("NewClient 返回 nil")
	}

	t.Log("開始啟動 SDK 客戶端...")
	startTime := time.Now()

	// 使用 goroutine 和 channel 來設定超時
	errChan := make(chan error, 1)
	go func() {
		errChan <- client.Start()
	}()

	select {
	case err := <-errChan:
		if err != nil {
			t.Fatalf("啟動 SDK 客戶端失敗 (耗時 %v): %v\n提示: 請確保已安裝新版 Copilot CLI (winget install GitHub.Copilot)", time.Since(startTime), err)
		}
		t.Logf("✅ SDK 客戶端成功啟動 (耗時 %v)", time.Since(startTime))
	case <-time.After(15 * time.Second):
		t.Fatal("❌ 啟動 SDK 客戶端超時（超過 15 秒）")
	}

	defer func() {
		t.Log("停止 SDK 客戶端...")
		errs := client.Stop()
		if len(errs) > 0 {
			t.Logf("停止時發生錯誤: %v", errs)
		}
	}()

	// 測試 Ping
	t.Log("測試 Ping...")
	pong, err := client.Ping("test")
	if err != nil {
		t.Fatalf("Ping 失敗: %v", err)
	}

	t.Logf("✅ Ping 成功: %s (timestamp: %d)", pong.Message, pong.Timestamp)
}

// TestSDKSessionCreation 測試 session 建立與銷毀
//
// 注意 (2026-01-21 更新)：
//   - 此測試使用舊版 SDK (github.com/github/copilot-sdk/go)
//   - 建議遷移至新版 SDK (github.com/github/copilot-cli-sdk-go)
func TestSDKSessionCreation(t *testing.T) {
	t.Skip("跳過: 舊版 SDK 與新版 CLI 協議不相容，等待遷移至新版 SDK")
	
	// 新版獨立 Copilot CLI 直接使用 "copilot" 命令
	cliPath := os.Getenv("COPILOT_CLI_PATH")
	if cliPath == "" {
		cliPath = "copilot"
	}

	t.Logf("使用 CLI 路徑: %s", cliPath)

	client := copilot.NewClient(&copilot.ClientOptions{
		CLIPath:  cliPath,
		LogLevel: "error",
	})

	t.Log("開始啟動 SDK 客戶端...")
	err := client.Start()
	if err != nil {
		t.Fatalf("啟動 SDK 客戶端失敗: %v\n提示: 請確保已安裝新版 Copilot CLI (winget install GitHub.Copilot)", err)
	}
	defer client.Stop()

	t.Log("✅ SDK 客戶端已啟動")

	t.Log("建立 Session...")
	session, err := client.CreateSession(&copilot.SessionConfig{
		Model: "gpt-4",
	})

	if err != nil {
		t.Fatalf("建立 Session 失敗: %v", err)
	}

	if session == nil {
		t.Fatal("CreateSession 返回 nil")
	}

	t.Log("✅ Session 成功建立")
	t.Logf("Session ID: %s", session.SessionID)
}

// TestSDKDecision 決策點：是否繼續 SDK 整合
func TestSDKDecision(t *testing.T) {
	fmt.Println("\n=== SDK PoC 決策報告 (2026-01-21 更新) ===")
	fmt.Println("")
	fmt.Println("⚠️ 重要版本變更:")
	fmt.Println("   - 舊版 'gh copilot' 已於 2025-10-25 停用")
	fmt.Println("   - 本專案使用的 SDK (github.com/github/copilot-sdk/go) 是舊版")
	fmt.Println("   - 建議遷移至新版 SDK: github.com/github/copilot-cli-sdk-go")
	fmt.Println("")
	fmt.Println("📋 遷移步驟:")
	fmt.Println("   1. 安裝新版 CLI: winget install GitHub.Copilot")
	fmt.Println("   2. 移除舊版 SDK: go get github.com/github/copilot-sdk/go@none")
	fmt.Println("   3. 安裝新版 SDK: go get github.com/github/copilot-cli-sdk-go")
	fmt.Println("   4. 更新 import 路徑")
	fmt.Println("")
	fmt.Println("詳見 VERSION_NOTICE.md")
}
