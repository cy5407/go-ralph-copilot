# 技術債清單

> 已識別但暫未解決的架構改進項目

---

## 1. Context 結構精簡化（優先級: Medium）

**識別時間**: 2026-01-21  
**狀態**: ⏳ 待解決（階段 9+）

### 問題

`ExecutionContext` 存在資訊重複與冗餘，SDK 已提供完整執行結果，不需自行保存：

```go
// 可移除的冗餘欄位（SDK 已涵蓋）
CLICommand        string
CLIOutput         string
CLIExitCode       int
ParsedCodeBlocks  []string
ParsedOptions     []string
CleanedOutput     string
```

### 建議解法（適配層模式）

```go
type ExecutionContext struct {
    LoopID              string
    LoopIndex           int
    Timestamp           time.Time
    SDKResponse         interface{}  // SDK 完整返回
    SDKError            error
    CircuitBreakerState string
    ExitReason          string
    SavedAt             time.Time
}
```

### 影響範圍

- `context.go`、`context_test.go`（需更新測試）、`persistence.go`

---

## ~~2. SDK 版本遷移~~ ✅ 已完成（2026-02-24）

遷移至 `github.com/github/copilot-sdk/go v0.1.26`，`sdk_executor.go` 完整整合。

---

## 3. SDK Executor 啟用（優先級: Low）

**識別時間**: 2026-02-24  
**狀態**: ⏸ 暫停（待 SDK 更新或 bundler 流程建立）

### 問題根因

`SDKExecutor` 啟動永遠失敗，根因如下：

1. `embeddedcli.Setup()` 從未被呼叫 → `embeddedcli.Path()` 回傳 `""`
2. Fallback 到系統 PATH 的 `copilot`（目前為 v0.0.415）
3. Go SDK 執行 `copilot --headless --stdio ...`，但 v0.0.415 已移除 `--headless` flag → `exit status 1`

TypeScript SDK 不受影響，因為 `@github/copilot` npm 包**自動內嵌** v0.0.403（支援 `--headless`）。

### 暫時處置（2026-02-24）

`DefaultClientConfig()` 已改為 `EnableSDK: false` / `PreferSDK: false`，系統走 CLI 路徑（穩定可用）。

### 未來重新啟用條件（擇一）

**方案 A：等 Go SDK 更新**（被動）  
等 SDK 移除 `--headless` flag，相容 v0.0.415+。

**方案 B：使用 `go tool bundler` 內嵌舊版 CLI**（主動）
```bash
# 1. 取得支援 --headless 的 CLI binary（例如從 @github/copilot@0.0.403 npm 包解壓）
# 2. 安裝 bundler
go get -tool github.com/github/copilot-sdk/go/cmd/bundler
# 3. 執行 bundler（在 build 前）
go tool bundler
# 4. 在程式啟動時呼叫 Setup
embeddedcli.Setup(embeddedcli.Config{Cli: cliReader, CliHash: hash})
# 5. 重新啟用
config.EnableSDK = true
config.PreferSDK = true
```

---

## 待辦清單

| 技術債 | 優先級 | 狀態 |
|--------|--------|------|
| Context 結構精簡化 | Medium | ⏳ 待解決 |
| SDK Executor 啟用 | Low | ⏸ 暫停 |
| ~~SDK 版本遷移~~ | ~~Low~~ | ✅ 完成 |

---

**最後更新**: 2026-02-24
