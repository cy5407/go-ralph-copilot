# Task: SDK Executor MVP

**目標**：讓 SDK Executor 成為真正可用的主要執行路徑，CLI 作為備用並清楚通知使用者。

---

## 現況分析

### 問題一：Fallback 通知不顯眼
`client.go:220`：
```go
log.Printf("⚠️ SDK 執行器啟動失敗，改用 CLI: %v", startErr)
```
- 用 `log.Printf` 而非 `infoLog`，在非 debug 模式下可能看不到
- 沒有 "正在使用 SDK 模式" 的啟動提示
- 使用者不知道當前用的是 SDK 還是 CLI

### 問題二：SDK 從未真正被驗證
- `isHealthy()` = `initialized && running && !closed`
- `Start()` 呼叫 `e.client.Start(ctx)`，實際上是啟動 copilot 子進程作為 agent server
- 目前沒有任何端對端測試確認 SDK → `Complete()` → 收到回應 這條路走得通
- 測試全是 mock，不代表真實行為

### 問題三：沒有 flag 讓使用者選擇
- 只能改 config，沒有 `-use-sdk` / `-no-sdk` CLI flag

---

## MVP 範圍（最小可用）

> 目標：使用者執行 `ralph-loop run` 時，畫面上能看到「用 SDK 還是 CLI」，失敗時有明確訊息。

### T1：Fallback 通知改為 `infoLog`（必做）

**檔案**：`internal/ghcopilot/client.go`

**改動**：
```go
// 改前
log.Printf("⚠️ SDK 執行器啟動失敗，改用 CLI: %v", startErr)

// 改後
infoLog("⚠️ SDK 執行器啟動失敗，降級使用 CLI 模式: %v", startErr)
```

**新增**：在 SDK 成功啟動時也顯示：
```go
infoLog("✅ 使用 SDK 模式執行")
```

在確定走 CLI 時顯示：
```go
infoLog("🔧 使用 CLI 模式執行")
```

---

### T2：每個迴圈開頭顯示執行模式（必做）

**位置**：`client.go ExecuteLoop()` 開頭，在 `infoLog("⏳ 執行 Copilot CLI ...")` 之前

**輸出範例**：
```
🔄 迴圈 1/15 - 正在執行...
[INFO] 📡 執行模式: SDK (claude-sonnet-4.5)
[INFO] ⏳ 執行中 (超時: 3m0s)...
```
或
```
🔄 迴圈 1/15 - 正在執行...
[INFO] 🔧 執行模式: CLI (SDK 不可用)
[INFO] ⏳ 執行中 (超時: 3m0s)...
```

---

### T3：新增 `-use-sdk` / `-no-sdk` flag（必做）

**檔案**：`cmd/ralph-loop/main.go`

```go
useSDK  := flag.Bool("use-sdk",  true,  "優先使用 SDK 執行器（預設開啟）")
noSDK   := flag.Bool("no-sdk",   false, "強制使用 CLI 執行器，跳過 SDK")
```

對應 config：
```go
if *noSDK {
    config.EnableSDK = false
    config.PreferSDK = false
}
```

使用範例：
```bash
# 明確使用 CLI（除錯用）
ralph-loop run -prompt "..." -no-sdk

# 明確嘗試 SDK（預設行為）
ralph-loop run -prompt "..." -use-sdk
```

---

### T4：SDK 端對端煙霧測試（必做）

**檔案**：`test/sdk_smoke_test.go`（新建）

驗證：
1. `SDKExecutor.Start(ctx)` 不回傳錯誤
2. `SDKExecutor.isHealthy()` 回傳 true
3. `SDKExecutor.Complete(ctx, "hello")` 回傳非空字串

```go
//go:build smoke

func TestSDKExecutorSmoke(t *testing.T) {
    // 需要真實 copilot 環境執行
    // 執行方式: go test -tags smoke ./test/...
}
```

> 注意：這個測試需要真實 Copilot 授權，用 build tag `smoke` 隔離，不在一般 `go test ./...` 中執行。

---

## 非 MVP（本次不做）

- SDK Session Pool 多工（多個同時執行的任務）
- SDK 失敗自動重試後再 fallback
- SDK 效能指標 dashboard
- `execution_mode_selector.go` 的動態切換邏輯（目前架構過度設計，MVP 不需要）

---

## 驗收標準

執行以下指令時：
```bash
ralph-loop run -prompt "列出當前目錄下的 Go 檔案" -max-loops 1
```

輸出必須包含：
- `[INFO] 執行模式: SDK` 或 `[INFO] 執行模式: CLI（SDK 不可用）`
- 若用 `-no-sdk` flag，強制顯示 CLI 模式
- SDK 失敗時，錯誤訊息對使用者可見（不是只有 log level debug 才看得到）

---

## 執行順序

```
T1（Fallback 通知）→ T2（迴圈開頭顯示）→ T3（flag）→ T4（煙霧測試）
```

T1、T2、T3 純改現有程式碼，不需要真實 Copilot 環境，可以先做。  
T4 需要真實環境驗證，最後做。
