# Task-20260224: SDK Executor MVP

**日期**：2026-02-24  
**目標**：讓 SDK Executor 成為真正可用的主要執行路徑，CLI 作為備用並清楚通知使用者。  
**執行模型建議**：Haiku（T1~T3 純改現有程式碼）

---

## 現況分析

| 問題 | 位置 | 影響 |
|------|------|------|
| Fallback 用 `log.Printf`，使用者看不到 | `client.go:220` | 不知道走哪條路 |
| 沒有執行模式顯示 | `client.go:213~232` | 無法判斷 SDK 是否有用 |
| 沒有 `-no-sdk` flag | `cmd/ralph-loop/main.go:21~27` | 無法強制選擇模式 |
| SDK 真實路徑從未端對端驗證 | `sdk_executor.go:160` | 不確定能不能跑 |

---

## T1：修正 Fallback 通知（10 分鐘）

### 目標
讓使用者在終端機看到「SDK 失敗，改用 CLI」的提示訊息，而非靜默 fallback。

### 修改位置
**檔案**：`internal/ghcopilot/client.go`  
**函數**：`ExecuteLoop()`  
**行數**：第 215~232 行

### 改前（原始碼）
```go
// 如果配置優先使用 SDK，嘗試啟動並使用 SDK
if c.config.PreferSDK && c.config.EnableSDK && c.sdkExecutor != nil {
    // Lazy-start：第一次呼叫時才啟動 SDK 執行器
    if !c.sdkExecutor.isHealthy() {
        if startErr := c.sdkExecutor.Start(ctx); startErr != nil {
            log.Printf("⚠️ SDK 執行器啟動失敗，改用 CLI: %v", startErr)
        }
    }
    if c.sdkExecutor.isHealthy() {
        output, executionErr = c.sdkExecutor.Complete(ctx, prompt)
        if executionErr == nil {
            usedSDK = true
            execCtx.CLICommand = "sdk:complete"
            execCtx.CLIOutput = output
            execCtx.CLIExitCode = 0
        }
    }
}
```

### 改後（目標程式碼）
```go
// 如果配置優先使用 SDK，嘗試啟動並使用 SDK
if c.config.PreferSDK && c.config.EnableSDK && c.sdkExecutor != nil {
    // Lazy-start：第一次呼叫時才啟動 SDK 執行器
    if !c.sdkExecutor.isHealthy() {
        if startErr := c.sdkExecutor.Start(ctx); startErr != nil {
            infoLog("⚠️ SDK 執行器啟動失敗，降級使用 CLI 模式: %v", startErr)
        }
    }
    if c.sdkExecutor.isHealthy() {
        infoLog("📡 使用 SDK 模式執行")
        output, executionErr = c.sdkExecutor.Complete(ctx, prompt)
        if executionErr == nil {
            usedSDK = true
            execCtx.CLICommand = "sdk:complete"
            execCtx.CLIOutput = output
            execCtx.CLIExitCode = 0
        } else {
            infoLog("⚠️ SDK 執行失敗，降級使用 CLI 模式: %v", executionErr)
        }
    }
}

// SDK 失敗/不可用/未啟用，或配置不優先使用 SDK 時，使用 CLI
if !usedSDK {
    infoLog("🔧 使用 CLI 模式執行")
```

> **注意**：`infoLog` 函數已存在於 `internal/ghcopilot/cli_executor.go`，定義為：
> ```go
> func infoLog(format string, args ...interface{}) {
>     if os.Getenv("RALPH_SILENT") != "1" {
>         log.Printf("[INFO %s] "+format, append([]interface{}{time.Now().Format("15:04:05.000")}, args...)...)
>     }
> }
> ```
> 不需要引入 `"log"` 套件（如果 `client.go` 目前有 `import "log"` 且只有這一行用，可以移除）。

### 驗收
- 執行 `ralph-loop run -prompt "test" -max-loops 1` 後終端機應出現以下其中之一：
  - `[INFO xx:xx:xx] 📡 使用 SDK 模式執行`
  - `[INFO xx:xx:xx] ⚠️ SDK 執行器啟動失敗，降級使用 CLI 模式: ...`
  - `[INFO xx:xx:xx] 🔧 使用 CLI 模式執行`

---

## T2：新增 `-no-sdk` flag（10 分鐘）

### 目標
讓使用者可以透過 `-no-sdk` 強制跳過 SDK，方便除錯。

### 修改位置
**檔案**：`cmd/ralph-loop/main.go`

#### Step 1：在 runCmd flag 定義區新增 flag（第 22~27 行附近）
在第 27 行（`runSilent` 後面）加入：
```go
runNoSDK := runCmd.Bool("no-sdk", false, "強制使用 CLI 執行器，跳過 SDK（除錯用）")
```

#### Step 2：修改 `cmdRun` 函數簽名（第 115 行）
```go
// 改前
func cmdRun(prompt string, maxLoops int, timeout time.Duration, cliTimeout time.Duration, workDir string, silent bool) {

// 改後
func cmdRun(prompt string, maxLoops int, timeout time.Duration, cliTimeout time.Duration, workDir string, silent bool, noSDK bool) {
```

#### Step 3：在 `cmdRun` 的 config 設定區加入（第 126~133 行附近，`config.SameErrorThreshold = 5` 之後）：
```go
if noSDK {
    config.EnableSDK = false
    config.PreferSDK = false
}
```

#### Step 4：修改呼叫 `cmdRun` 的地方（第 54 行）
```go
// 改前
cmdRun(*runPrompt, *runMaxLoops, *runTimeout, *runCLITimeout, *runWorkDir, *runSilent)

// 改後
cmdRun(*runPrompt, *runMaxLoops, *runTimeout, *runCLITimeout, *runWorkDir, *runSilent, *runNoSDK)
```

### 驗收
```bash
# 應在輸出中看到 CLI 模式（不嘗試 SDK）
ralph-loop run -prompt "test" -max-loops 1 -no-sdk
# 預期輸出包含: [INFO] 🔧 使用 CLI 模式執行
# 不應出現: [INFO] 📡 使用 SDK 模式執行
```

---

## T3：build 驗證（2 分鐘）

T1、T2 完成後執行：
```bash
go build -o ralph-loop.exe ./cmd/ralph-loop
go test ./internal/ghcopilot/... ./cmd/...
```

兩者均須 **無錯誤** 通過。

---

## T4：SDK 端對端煙霧測試（需真實 Copilot 環境）

### 目標
確認 SDK 路徑真的能跑通：`Start()` → `isHealthy()=true` → `Complete()` 回傳非空字串。

### 新建檔案
**檔案**：`test/sdk_smoke_test.go`

```go
//go:build smoke

package test

import (
    "context"
    "testing"
    "time"

    "github.com/cy540/ralph-loop/internal/ghcopilot"
)

// 執行方式（需要真實 copilot 授權）:
//   go test -tags smoke -timeout 60s ./test/...
func TestSDKExecutorSmoke(t *testing.T) {
    config := ghcopilot.DefaultSDKConfig()
    config.Timeout = 30 * time.Second

    executor := ghcopilot.NewSDKExecutor(config)

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // 測試 Start
    if err := executor.Start(ctx); err != nil {
        t.Fatalf("SDK Start() 失敗: %v", err)
    }
    defer executor.Close()

    // 測試 isHealthy（透過 GetStatus 間接驗證）
    status := executor.GetStatus()
    if !status["running"].(bool) {
        t.Fatal("SDK 啟動後 running 應為 true")
    }

    // 測試 Complete
    output, err := executor.Complete(ctx, "請回覆：hello")
    if err != nil {
        t.Fatalf("SDK Complete() 失敗: %v", err)
    }
    if output == "" {
        t.Fatal("SDK Complete() 回傳空字串")
    }
    t.Logf("SDK 回應: %s", output)
}
```

> `NewSDKExecutor`、`DefaultSDKConfig`、`GetStatus` 均已存在於 `internal/ghcopilot/sdk_executor.go`。  
> `Close()` 方法需確認存在，若無則用 `executor.Stop(ctx)`。

---

## 非 MVP（本次不做）

- SDK Session Pool 多工並發
- SDK 失敗後智能重試再 fallback  
- `execution_mode_selector.go` 的動態規則切換（目前過度設計）
- SDK 效能 dashboard

---

## 執行順序與估時

```
T1（15 分鐘）→ T2（10 分鐘）→ T3 build 驗證（2 分鐘）→ git commit → T4（需環境）
```

| 任務 | 檔案 | 行數變動 | 需要真實 Copilot |
|------|------|----------|-----------------|
| T1 Fallback 通知 | `internal/ghcopilot/client.go` | ~5 行 | ❌ 不需要 |
| T2 `-no-sdk` flag | `cmd/ralph-loop/main.go` | ~6 行 | ❌ 不需要 |
| T3 Build 驗證 | — | — | ❌ 不需要 |
| T4 煙霧測試 | `test/sdk_smoke_test.go`（新建） | ~40 行 | ✅ 需要 |
