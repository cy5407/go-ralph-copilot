# Go SDK `--headless` 相容性問題

> 調查日期：2026-02-24  
> 狀態：⚠️ SDK 本身 bug，暫時停用 SDK，走 CLI 路徑

---

## 問題現象

執行 `ralph-loop run` 時出現：

```
[INFO] ⚠️ SDK 執行器啟動失敗，降級使用 CLI 模式:
       failed to start copilot client: CLI process exited: exit status 1
```

`SDKExecutor.isHealthy()` 永遠回傳 `false`，所有請求都走 CLI 路徑。

---

## 根本原因

### 失敗鏈

```
Go SDK v0.1.26 (client.go:1033)
    ↓ 執行
copilot --headless --no-auto-update --log-level info --stdio
    ↓ 但
系統 PATH = copilot v0.0.415
    ↓ v0.0.415 已移除 --headless flag
exit status 1
    ↓
SDKExecutor.Start() 失敗 → isHealthy() = false
```

### `--headless` 是什麼？

`--headless` 是舊版 Copilot CLI 的 server 模式旗標，搭配 `--stdio` 讓 CLI 以無介面背景服務運行，外部程式（SDK）透過 stdin/stdout 與之溝通。

### 版本演進

| CLI 版本 | server 模式旗標 | 說明 |
|---------|----------------|------|
| ≤ v0.0.409 附近 | `--headless` | 舊協議 |
| v0.0.415（目前） | `--acp` (Agent Client Protocol) | 新協議，移除 `--headless` |

Go SDK v0.1.26 原始碼寫死傳 `--headless`，尚未更新為 `--acp`，**這是 SDK 本身的 bug**。

### 為什麼 TypeScript 沒這個問題？

TypeScript SDK（`@github/copilot-sdk`）的 npm 包內**自動內嵌** `@github/copilot@0.0.403`。
該版本支援 `--headless`，所以 TypeScript 版本完全不受影響。

Go SDK 則沒有自動內嵌，需要開發者自行執行 `go tool bundler` 才能內嵌特定版本 CLI。
未執行 bundler → `embeddedcli.Path()` 回傳 `""` → fallback 到系統 PATH 的 v0.0.415 → 失敗。

---

## 目前處置（2026-02-24）

`DefaultClientConfig()` 已改為停用 SDK：

```go
EnableSDK: false,  // SDK 需要舊版 CLI，目前不支援
PreferSDK: false,  // 走 CLI 路徑（穩定可用）
```

系統功能完全正常，只是走 CLI 路徑而非 SDK 路徑。

---

## 重新啟用的前提條件（擇一）

### 方案一：等 Go SDK 更新（零工作量）

等 SDK 將 `--headless` 改為 `--acp`，就能相容 v0.0.415+。
屆時改回 `EnableSDK: true` 即可。

### 方案二：`COPILOT_CLI_PATH` 指定舊版 binary（最簡單）

從 npm 取得 v0.0.403 binary（該版本支援 `--headless`）：

```bash
npm pack @github/copilot@0.0.403
# 解壓 tgz，取出 CLI binary（通常是 bin/ 目錄下的 JS 包裝器 + node runtime）
```

設定環境變數，不需修改程式碼：

```bash
COPILOT_CLI_PATH=C:\tools\copilot-v0.0.403\bin\copilot ./ralph-loop.exe run ...
```

或在 `sdk_executor.go` 中讓 `CLIPath` 讀取環境變數（已預留 `CLIPath` 欄位）：

```go
// sdk_executor.go DefaultSDKConfig()
CLIPath: func() string {
    if p := os.Getenv("COPILOT_CLI_PATH"); p != "" {
        return p
    }
    return "copilot"
}(),
```

### 方案三：`go tool bundler` 內嵌舊版 CLI（官方推薦方式）

```bash
go get -tool github.com/github/copilot-sdk/go/cmd/bundler
# 準備 v0.0.403 binary（同方案二的 npm pack 步驟）
go tool bundler   # 在 build 前執行，自動呼叫 embeddedcli.Setup()
go build -o ralph-loop.exe ./cmd/ralph-loop
```

適合 CI/CD 流程，確保每次 build 都內嵌固定版本。

---

## 相關檔案

| 檔案 | 說明 |
|------|------|
| `internal/ghcopilot/sdk_executor.go` | SDK 執行器實作，`CLIPath` 欄位位於第 16/30 行 |
| `internal/ghcopilot/client.go` | `DefaultClientConfig()` 中的 `EnableSDK`/`PreferSDK` 開關 |
| `TECHNICAL_DEBT.md` | 技術債追蹤，#3 SDK Executor 啟用 |
