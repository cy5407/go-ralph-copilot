# Ralph Loop - AI 驅動的自動程式碼迭代系統

[![Version](https://img.shields.io/badge/version-0.1.0--stable-green.svg)](https://github.com/cy5407/go-ralph-copilot/releases/tag/v0.1.0-stable)
[![Go Version](https://img.shields.io/badge/Go-1.24+-blue.svg)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

> 在看到保哥於群組說他用SDK建立了像是Ralph的工具後，基於 GitHub Copilot SDK 的自主程式碼修正與迭代工具，並於後續在維護及產品實用化的過程中，參考了非常多保哥的設計，如果保哥希望我下架，我會盡快處理。
 (這是我用Vibe coding 出來的AI垃圾，主要是我想用Go來建立這個工具)

## 📦 穩定版本

**當前穩定版本**: `v0.1.0-stable` ([Release Notes](https://github.com/cy5407/go-ralph-copilot/releases/tag/v0.1.0-stable))

此版本已驗證可正常運作，包含：
- ✅ AI 正常執行任務（無無限循環）
- ✅ Copilot 輸出完整顯示
- ✅ 完成檢測機制有效
- ✅ 版本變數符合 ldflags 注入需求

安裝穩定版本：
```bash
git clone https://github.com/cy5407/go-ralph-copilot.git
cd go-ralph-copilot
git checkout v0.1.0-stable
go build -o ralph-loop.exe ./cmd/ralph-loop
```

## 🎯 專案概述

Ralph Loop 是一個參考了Ralph-Loop，想拿來在Copilot上仿造的 AI 驅動自動化系統，透過「觀察→反思→行動」(ORA) 循環實現自主程式碼迭代與修正。
基本上就是我的AI 小垃圾，希望大家見諒

### 靈感來源

本專案受到以下專案的啟發並參考：
- [ralph-claude-code](https://github.com/frankbria/ralph-claude-code) - Ralph Loop 的原始概念與設計思想
- [copilot-ralph](https://github.com/doggy8088/copilot-ralph) - Will 保哥以 TypeScript/Node.js 實作的 GitHub Copilot 版 Ralph Loop，本專案在以下概念上高度參考此作：
  - **結構化完成信號設計**：保哥的 `<promise>phrase</promise>` 精確字串偵測機制，啟發本專案以 `EXIT_SIGNAL: true` 嵌入 `---RALPH_STATUS---` 區塊的同等設計哲學——都是讓 AI 在輸出中嵌入特殊標記來明確表達任務已完成，避免誤判，因實測許多方法，無法有效的進行暫停/繼續迴圈的邏輯判斷，參考了保哥專案的信號設計。
  - **Copilot SDK 事件串流架構**：兩者都迭代 `TextEvent` / `ToolCallEvent` / `ToolResultEvent` 事件流，並在終端機即時顯示工具執行過程；保哥的 `LoopEngine.executeIteration()` 事件處理邏輯與本專案的 SDK executor 事件迭代在結構上高度一致。
  - **LoopConfig 核心欄位設計**：`maxIterations`、`timeoutMs`、`model`、`workingDir`、`promisePhrase` 這些欄位並非 GitHub Copilot SDK 本身定義的欄位，而是保哥在 `copilot-ralph` 的 `LoopConfig` 介面中自行設計的命名語意；本專案的 `ClientConfig` 直接沿用了相同的欄位命名與概念。
- 採用 **Vibe Coding** 開發方法論 - 以 AI 輔助的快速迭代開發流程

### 核心技術

- **語言**: Go 1.24.5
- **AI 整合**: GitHub Copilot SDK (v0.1.26)
- **架構**: 雙執行器（SDK + CLI）+ 智能模式選擇
- **測試**: 351 個測試，93% 覆蓋率

## ✨ 主要功能

### 1. 智能執行模式
- **SDK 執行器** (主要) - 使用 GitHub Copilot SDK 進行 AI 互動
- **CLI 執行器** (備用) - 整合 GitHub Copilot CLI 工具
- **混合執行器** - SDK 失敗時自動降級到 CLI
- **執行模式選擇器** - 根據性能指標智能選擇最佳執行方式

### 2. 自動化迭代
- **ORA 循環** - Observe (觀察錯誤) → Reflect (AI 分析) → Act (自動修正)
- **重試機制** - 三種重試策略（指數退避、線性退避、固定間隔）
- **熔斷器保護** - 防止無限循環，三狀態模式（CLOSED/HALF_OPEN/OPEN）
- **完成檢測** - 智能判斷任務是否完成

### 3. 狀態管理
- **上下文管理** - 完整的執行歷史追蹤
- **持久化系統** - 支援 JSON/Gob 格式的狀態保存與恢復
- **會話池管理** - SDK 會話生命週期管理
- **備份與恢復** - 自動備份與狀態恢復機制

### 4. 性能監控
- **執行時間追蹤** - SDK/CLI 執行器性能統計
- **錯誤率監控** - 自動收集錯誤統計
- **健康檢查** - 執行器可用性監控

## 🚀 快速開始

### 先決條件

1. **Go 1.21+** (專案使用 Go 1.24.5)
2. **GitHub Copilot CLI** - 獨立版本
   ```bash
   # Windows (使用 winget)
   winget install GitHub.Copilot

   # 或使用 npm
   npm install -g @github/copilot

   # 驗證安裝
   copilot --version

   # 認證 (需要有效的 GitHub Copilot 訂閱)
   copilot auth
   ```

### 安裝

**方式 1：使用穩定版本（推薦）**

```bash
# 克隆專案
git clone https://github.com/cy5407/go-ralph-copilot.git
cd go-ralph-copilot

# 切換到穩定版本
git checkout v0.1.0-stable

# 建置執行檔
go build -o ralph-loop.exe ./cmd/ralph-loop

# 驗證安裝
.\ralph-loop.exe version
```

**方式 2：使用最新開發版**

```bash
# 克隆專案
git clone https://github.com/cy5407/go-ralph-copilot.git
cd go-ralph-copilot

# 使用 master 分支（包含最新功能，可能不穩定）
go build -o ralph-loop.exe ./cmd/ralph-loop
```

### 基本使用

```bash
# 執行自動修復迴圈
./ralph-loop.exe run -prompt "修復所有編譯錯誤" -max-loops 10 -timeout 5m

# 查看系統狀態
./ralph-loop.exe status

# 重置熔斷器
./ralph-loop.exe reset

# 監控模式
./ralph-loop.exe watch -interval 3s

# 查看版本
./ralph-loop.exe version
```

### 進階選項

```bash
# 啟用詳細日誌（除錯模式）
RALPH_DEBUG=1 ./ralph-loop.exe run -prompt "..." -max-loops 5

# 使用模擬模式（測試用，不消耗 API quota）
COPILOT_MOCK_MODE=true ./ralph-loop.exe run -prompt "測試" -max-loops 3
```

## 🏗️ 架構設計

### 執行流程

```
使用者啟動 ralph-loop
    ↓
[迴圈 N] RalphLoopClient.ExecuteLoop()
    ↓
├─ ExecutionModeSelector → 選擇最佳執行器
│   ├─ SDKExecutor (主要)
│   └─ CLIExecutor (備用)
│
├─ OutputParser → 解析 AI 輸出
│   └─ 提取程式碼區塊與結構化狀態
│
├─ ResponseAnalyzer → 分析回應
│   ├─ 完成偵測
│   └─ 卡住偵測
│
├─ CircuitBreaker → 熔斷保護
│   ├─ 無進展檢測（預設 3 次觸發）
│   └─ 相同錯誤檢測（預設 5 次觸發）
│
├─ ContextManager → 歷史管理
│   └─ 記錄每個迴圈的輸入/輸出/錯誤
│
└─ PersistenceManager → 持久化
    └─ 儲存至 .ralph-loop/saves/
```

### 核心模組

| 模組 | 功能 | 檔案 |
|------|------|------|
| **RalphLoopClient** | 主要 API 入口，整合所有功能 | `internal/ghcopilot/client.go` |
| **SDKExecutor** | GitHub Copilot SDK 執行器（主要） | `internal/ghcopilot/sdk_executor.go` |
| **CLIExecutor** | GitHub Copilot CLI 執行器（備用） | `internal/ghcopilot/cli_executor.go` |
| **ExecutionModeSelector** | 智能執行模式選擇 | `internal/ghcopilot/execution_mode_selector.go` |
| **HybridExecutor** | 混合執行器（SDK→CLI降級） | `internal/ghcopilot/execution_mode_selector.go` |
| **RetryExecutor** | 重試機制（3種策略） | `internal/ghcopilot/retry_executor.go` |
| **SDKSessionPool** | SDK 會話管理池 | `internal/ghcopilot/sdk_session_pool.go` |
| **CircuitBreaker** | 熔斷器保護 | `internal/ghcopilot/circuit_breaker.go` |
| **ContextManager** | 上下文與歷史管理 | `internal/ghcopilot/context.go` |
| **PersistenceManager** | 持久化管理 | `internal/ghcopilot/persistence.go` |

## 🧪 測試

```bash
# 執行所有測試
go test ./...

# 執行特定套件測試
go test ./internal/ghcopilot

# 詳細輸出
go test -v ./internal/ghcopilot

# 測試覆蓋率
go test -cover ./internal/ghcopilot
```

### 測試統計

- **總測試數**: 351 個
- **通過率**: 100%
- **覆蓋率**: 93%

## 📊 專案結構

```
ralph-loop/
├── cmd/ralph-loop/              # CLI 主程式入口
│   └── main.go
├── internal/ghcopilot/          # 核心業務邏輯 (33 個 Go 文件)
│   ├── client.go                # 主 API
│   ├── sdk_executor.go          # SDK 執行器
│   ├── cli_executor.go          # CLI 執行器
│   ├── execution_mode_selector.go
│   ├── retry_executor.go
│   ├── circuit_breaker.go
│   ├── context.go
│   ├── persistence.go
│   └── ...
├── test/                        # 整合測試
│   └── sdk_poc_test.go
├── 過期文件/                    # 已完成/已取代的歷史文件
├── .ralph-loop/                 # 執行時資料
│   └── saves/                   # 執行歷史保存
├── go.mod                       # Go 模組定義
└── README.md                    # 本文件
```

## ⚙️ 配置

### ClientConfig 參數

```go
config := ghcopilot.DefaultClientConfig()
config.CLITimeout = 60 * time.Second      // Copilot 單次執行超時
config.CLIMaxRetries = 3                  // 失敗重試次數
config.CircuitBreakerThreshold = 3        // 無進展迴圈數觸發熔斷
config.SameErrorThreshold = 5             // 相同錯誤次數觸發熔斷
config.Model = "claude-sonnet-4.5"        // AI 模型
config.WorkDir = "."                      // 工作目錄
config.SaveDir = ".ralph-loop/saves"      // 歷史儲存位置
config.EnableSDK = true                   // 啟用 SDK 執行器
config.PreferSDK = true                   // 優先使用 SDK
```

## 📖 文檔

- **[ARCHITECTURE.md](ARCHITECTURE.md)** - 系統架構說明
- **[CLAUDE.md](CLAUDE.md)** - Claude Code 開發指南
- **[TECHNICAL_DEBT.md](TECHNICAL_DEBT.md)** - 技術債清單
- **[NEXT_STEPS.md](NEXT_STEPS.md)** - 下一步待辦事項
- **[過期文件/](過期文件/)** - 已完成/已取代的歷史文件

## 🔧 技術亮點

### Vibe Coding 開發流程

本專案採用 Vibe Coding 方法論，特點包括：
- AI 輔助的快速原型開發
- 測試驅動的迭代式實作
- 即時回饋與持續改進
- 人機協作的程式碼生成

### SDK 優先設計

系統優先使用 GitHub Copilot SDK，具備以下優勢：
- 更好的型別安全
- 原生 Go 整合
- 更細緻的錯誤處理
- 自動會話管理

### 智能降級機制

當 SDK 不可用時，系統自動降級到 CLI 執行器：
```go
if c.config.PreferSDK && c.sdkExecutor != nil && c.sdkExecutor.isHealthy() {
    output, err = c.sdkExecutor.Complete(ctx, prompt)
    if err != nil {
        // 降級到 CLI
        output, err = c.cliExecutor.ExecutePrompt(ctx, prompt)
    }
}
```

## 🚨 安全考量

- **自動執行程式碼**: 系統會執行 AI 建議的程式碼修改，建議在安全環境中測試
- **熔斷機制**: 防止無限迴圈消耗資源
- **工作目錄隔離**: 建議在測試專案中執行，避免影響重要程式碼
- **API 成本**: 每次迴圈消耗 GitHub Copilot API quota，請注意用量

## 🤝 貢獻

歡迎提交 Issue 和 Pull Request！

### 開發工作流程

1. Fork 本專案
2. 創建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交變更 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 開啟 Pull Request

## 📄 授權

MIT License

## 🙏 致謝

- [ralph-claude-code](https://github.com/frankbria/ralph-claude-code) - 原始靈感與概念設計
- [copilot-ralph](https://github.com/doggy8088/copilot-ralph) (Will 保哥) - 結構化完成信號設計哲學、Copilot SDK 事件串流架構、LoopConfig 核心欄位語意，均為本專案的重要參考來源
- GitHub Copilot Team - 提供強大的 AI SDK
- Vibe Coding 社群 - 創新的開發方法論

## 📞 聯繫

- GitHub Issues: [問題追蹤](https://github.com/cy5407/go-ralph-copilot/issues)
- 專案維護者: [@cy5407](https://github.com/cy5407)

## 📜 版本歷史

### master (2026-02-24) - SDK 遷移與功能改進

- ✅ 遷移至 GitHub Copilot SDK v0.1.26（主要執行方式）
- ✅ 修復 Permission denied（`PermissionHandler.ApproveAll`）
- ✅ 新增 `RALPH_STATUS` / `REASON` 欄位解析
- ✅ 工具執行過程即時顯示（事件串流）
- ✅ CLI 自動降級機制

### v0.1.0-stable (2026-02-14) 🎉

**首個穩定版本發布**

**新增功能**：
- ✅ 修正 Version 變數為大寫（符合 ldflags 注入）
- ✅ 新增 io.MultiWriter 讓 Copilot 輸出同時顯示到終端
- ✅ 設定 Silent: false 啟用完整輸出顯示

**驗證結果**：
- AI 正常執行任務（無無限循環問題）
- Copilot 輸出完整顯示（可觀察 AI 行為）
- 完成檢測機制有效（單次迴圈完成任務）
- 版本變數符合 ldflags 注入需求

**已知限制**：
- 基於舊版完成檢測機制（無 Promise Detection）
- 無 System Prompt（依賴 `.github/copilot-instructions.md`）

[查看完整 Release Notes](https://github.com/cy5407/go-ralph-copilot/releases/tag/v0.1.0-stable)

---

**最後更新**: 2026-02-24  
**當前穩定版本**: v0.1.0-stable（master 持續更新中）  
**狀態**: ✅ 已驗證可用（SDK v0.1.26，Permission denied 已修復）

