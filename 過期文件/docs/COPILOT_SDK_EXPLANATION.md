# Copilot Go SDK 用途說明

## 📌 簡短回答

**Copilot Go SDK** (`github.com/github/copilot-sdk/go`) 是 GitHub 官方提供的 **Go 語言函式庫**，允許開發者在 Go 應用程式中以**程式方式**呼叫 Copilot CLI 的功能，而不是僅通過命令行使用。

---

## 🎯 主要用途

### 1. **程式化集成 Copilot 功能**
讓 Go 應用能直接調用 Copilot 的核心功能：
```go
import copilot "github.com/github/copilot-sdk/go"

// 建立客戶端
client := copilot.NewClient(&copilot.ClientOptions{
    CLIPath:  "copilot",  // 或 "gh copilot"
    LogLevel: "info",
})

// 啟動 SDK 客戶端
err := client.Start()
if err != nil {
    log.Fatal(err)
}

// 現在可以呼叫 Copilot 功能
```

### 2. **避免執行外部進程的複雜性**
- ❌ **舊方式**: 使用 `os/exec` 呼叫 `copilot` 命令並解析輸出
  ```go
  cmd := exec.Command("copilot", "explain", "--text", code)
  output, _ := cmd.Output()
  // 手動解析輸出...
  ```

- ✅ **新方式**: 直接使用 SDK API
  ```go
  result, err := client.Explain(ctx, "代碼片段")
  // 直接取得結構化數據
  ```

### 3. **API 標準化和類型安全**
SDK 提供類型安全的 Go 介面，而不是字符串解析：
```go
type CompletionRequest struct {
    // 有類型檢查和自動完成的欄位
}

type CompletionResponse struct {
    // 結構化的回應數據
}
```

---

## 📊 你的項目中的使用情況

### 目前狀態
在 `test/sdk_poc_test.go` 中有 3 個測試：

1. **TestSDKBasicConnection** - 測試基本連接
   ```go
   client := copilot.NewClient(&copilot.ClientOptions{
       CLIPath: "copilot",
   })
   client.Start()
   client.Ping("test")
   ```

2. **TestSDKSessionCreation** - 測試 Session 管理

3. **TestSDKDocumentation** - 文檔測試

### 版本標記
你的項目使用：
- **版本**: v0.1.15-preview.0 (最新開發版) ✅
- **狀態**: 已升級至最新版本
- **發布日期**: 2026-01-22
- **位置**: `go.mod` 中標記為間接依賴

```
github.com/github/copilot-sdk/go v0.1.15-preview.0.0.20260121003103-2415f6f3b828
```

### 升級說明
- **來源**: 直接從 GitHub main 分支取得 (`go get github.com/github/copilot-sdk/go@main`)
- **優勢**: 取得最新的開發版本功能和修復
- **測試狀態**: ✅ 所有 3 個 SDK PoC 測試通過 (1.34s + 1.57s + 0.00s = 3.14s)

---

## 🔄 架構中的角色

在你的 **Ralph Loop 系統**中，Copilot SDK 的角色：

```
Ralph Loop System
├── CLI 層 (cli_executor.go)
│   ├── 使用 "copilot" 命令（獨立 CLI）
│   └── 解析命令輸出
│
└── SDK 層 (sdk_poc_test.go) ✨ 
    ├── 使用 Go SDK 程式化呼叫
    ├── 更結構化的 API
    └── 類型安全的介面
```

---

## 📈 CLI vs SDK 比較

| 特性 | 獨立 CLI | Go SDK |
|-----|--------|--------|
| **安裝方式** | `winget install GitHub.Copilot` | `go get github.com/github/copilot-sdk/go` |
| **呼叫方式** | 命令行指令 | Go 函式庫 API |
| **你的使用** | ✅ 正在使用 (cli_executor.go) | ⚠️ 舊版，供參考 |
| **類型安全** | ❌ 字符串結果 | ✅ 結構化類型 |
| **錯誤處理** | 🟡 Exit code | ✅ Go error 介面 |
| **性能** | 需啟動進程 | 共享進程連接 |
| **學習曲線** | 容易 | 中等 |

---

## 🚀 在你的 Ralph Loop 中的應用場景

### 場景 1: 直接集成
如果要在 Ralph Loop 中直接使用 SDK，可以：
```go
// 在 RalphLoopClient 中
func (c *RalphLoopClient) ExecuteWithSDK(ctx context.Context, prompt string) (*LoopResult, error) {
    result, err := c.sdkClient.Complete(ctx, prompt)
    if err != nil {
        return nil, err
    }
    // 處理結果...
}
```

### 場景 2: 混合方式
- CLI 用於簡單任務（更輕量級）
- SDK 用於複雜任務（需要保持連接）

### 場景 3: 遷移計畫
你的項目備註中提到需要遷移：
```
// 目前: github.com/github/copilot-sdk/go v0.1.15-preview.0
// 未來: 等待官方 v1.0 穩定版本
// Stage 8.3+: 整合 SDK 層至 RalphLoopClient (詳見 STAGE_8_3_PLANNING.md)
```

### 如何直接使用 GitHub 仓库

你可以直接從 GitHub 仓库取得 SDK，無需等待 npm/PyPI 發布：

```powershell
# 方式 1: 取得最新發布版本
go get -u github.com/github/copilot-sdk/go@v0.1.15-preview.0

# 方式 2: 取得最新開發版本 (推薦)
go get -u github.com/github/copilot-sdk/go@main

# 方式 3: 複製仓库到本地使用 (高級用法)
git clone https://github.com/github/copilot-sdk.git
# 在 go.mod 中使用 replace
replace github.com/github/copilot-sdk/go => ./copilot-sdk/go
```

---

## ⚙️ SDK 核心功能

根據官方文檔，Go SDK 通常提供：

```go
// 基本操作
client.Start()              // 啟動客戶端
client.Stop()               // 停止連接
client.Ping(msg)            // 心跳檢測

// Copilot 功能
client.Complete(ctx, prompt)      // 代碼完成
client.Explain(ctx, code)         // 解釋代碼
client.Tests(ctx, code)           // 生成測試
client.Review(ctx, code)          // 代碼審查

// Session 管理
client.GetSession(id)       // 取得 Session
client.CreateSession()      // 建立新 Session
```

---

## 🔗 相關文件位置

在你的項目中：
- **SDK 測試**: `test/sdk_poc_test.go` (3 個測試)
- **版本信息**: `VERSION_NOTICE.md` (標記為舊版)
- **技術債**: `TECHNICAL_DEBT.md` (需遷移至新版)
- **CLI 實現**: `internal/ghcopilot/cli_executor.go` (目前使用方式)

---

## 💡 建議

### 當前狀態 ✅
- 你正確地使用獨立 `copilot` CLI（版本 0.0.388）
- CLI 層 (`cli_executor.go`) 運作良好
- SDK PoC 測試提供了參考實現

### 未來計畫 🔮
1. **✅ SDK 升級完成** (2026-01-22)
   - v0.1.14 → v0.1.15-preview.0
   - 所有測試通過

2. **🔄 Stage 8.3: SDK 層整合** (計畫中)
   - 建立 SDKExecutor 模組
   - 實現會話管理
   - 添加容錯機制
   - 詳見 [STAGE_8_3_PLANNING.md](../../STAGE_8_3_PLANNING.md)

3. **🔮 等待官方 v1.0** (2026-Q1 預期)
   - 穩定版本發布
   - 完整功能支援

---

## 📚 額外資源

- 官方 SDK 文檔: https://pkg.go.dev/github.com/github/copilot-sdk/go
- 官方 GitHub 仓库: https://github.com/github/copilot-sdk
- 你的 CLI 執行器: [cli_executor.go](../internal/ghcopilot/cli_executor.go#L1)
- SDK 測試參考: [sdk_poc_test.go](../test/sdk_poc_test.go#L1)
- Stage 8.3 規劃: [STAGE_8_3_PLANNING.md](../../STAGE_8_3_PLANNING.md)
