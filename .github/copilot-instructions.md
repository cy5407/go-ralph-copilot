# GitHub Copilot Instructions - Ralph Loop

> AI 驅動的自動程式碼迭代系統 - 基於 GitHub Copilot SDK 與 CLI

## 語言偏好

**所有回應必須使用繁體中文。**

## 建置與測試

### 建置專案

```bash
# 建置主程式
go build -o ralph-loop.exe ./cmd/ralph-loop

# 建置所有套件
go build ./...
```

### 執行測試

```bash
# 執行所有測試
go test ./...

# 執行特定套件測試
go test ./internal/ghcopilot

# 詳細輸出
go test -v ./internal/ghcopilot

# 執行特定測試
go test -v -run TestCLIExecutor ./internal/ghcopilot

# 測試覆蓋率
go test -cover ./internal/ghcopilot
```

### 除錯與模擬

```bash
# 啟用詳細日誌（顯示 CLI 執行細節、重試、超時等）
RALPH_DEBUG=1 ./ralph-loop.exe run -prompt "..." -max-loops 5

# 使用模擬 Copilot 回應（不消耗 API quota）
COPILOT_MOCK_MODE=true ./ralph-loop.exe run -prompt "測試" -max-loops 3
```

## 架構概覽

### ORA 循環流程（Observe → Reflect → Act）

```
使用者啟動 ralph-loop
    ↓
[迴圈 N] RalphLoopClient.ExecuteLoop()
    ↓
├─ ExecutionModeSelector → 選擇最佳執行器 (SDK/CLI/Hybrid)
│   ├─ SDKExecutor (主要) - GitHub Copilot SDK
│   └─ CLIExecutor (備用) - GitHub Copilot CLI
│
├─ OutputParser → 解析 AI 輸出
│   └─ 提取程式碼區塊、結構化狀態
│
├─ ResponseAnalyzer → 分析回應
│   ├─ 完成偵測（雙重條件驗證）
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

### 核心模組位置

| 模組 | 檔案 | 職責 |
|------|------|------|
| **RalphLoopClient** | `internal/ghcopilot/client.go` | 主要 API 入口點，整合所有模組 |
| **SDKExecutor** | `internal/ghcopilot/sdk_executor.go` | GitHub Copilot SDK 執行器（主要） |
| **CLIExecutor** | `internal/ghcopilot/cli_executor.go` | GitHub Copilot CLI 執行器（備用） |
| **ExecutionModeSelector** | `internal/ghcopilot/execution_mode_selector.go` | 智能執行模式選擇與降級 |
| **RetryExecutor** | `internal/ghcopilot/retry_strategy.go` | 重試機制（指數/線性/固定間隔） |
| **CircuitBreaker** | `internal/ghcopilot/circuit_breaker.go` | 熔斷器保護（三狀態模式） |
| **ResponseAnalyzer** | `internal/ghcopilot/response_analyzer.go` | 完成判斷與卡住偵測 |
| **ContextManager** | `internal/ghcopilot/context.go` | 上下文與歷史管理 |
| **PersistenceManager** | `internal/ghcopilot/persistence.go` | 狀態保存/載入（JSON/Gob） |
| **SDKSessionPool** | `internal/ghcopilot/sdk_session.go` | SDK 會話生命週期管理 |

## 關鍵設計模式

### 1. 雙重條件退出驗證

系統使用**雙重條件驗證**來決定是否完成任務，防止過早退出或無限循環：

```go
func (ra *ResponseAnalyzer) IsCompleted() bool {
    // 條件 1: 有足夠的完成指標（分數 >= 20）
    if len(ra.completionIndicators) < 2 {
        return false
    }
    
    // 條件 2: AI 明確發出 EXIT_SIGNAL = true
    status := ra.ParseStructuredOutput()
    if status == nil || !status.ExitSignal {
        return false
    }
    
    // 雙重條件都滿足才退出
    return true
}
```

**為什麼需要雙重條件？**
- 防止誤判：避免僅因輸出包含「完成」等關鍵字就退出
- 防止無限循環：避免 AI 未發出明確退出信號時持續運行
- 提高可靠性：結構化信號（`EXIT_SIGNAL`）+ 自然語言關鍵字

### 2. 三狀態熔斷器

```
CLOSED (正常運作)
    ↓ 失敗×3 (無進展/相同錯誤)
OPEN (停止執行)
    ↓ 成功×1
HALF_OPEN (試探恢復)
    ↓ 成功×1
CLOSED (恢復正常)
```

### 3. 智能執行模式選擇

系統根據性能指標自動選擇最佳執行器：

- **SDK 優先**：型別安全、原生 Go 整合、更好的錯誤處理
- **CLI 降級**：SDK 失敗時自動降級
- **混合模式**：根據執行時間、錯誤率動態調整

### 4. 完成信號層次

```
層次 1: 結構化信號（最可靠）
├─ EXIT_SIGNAL = true (100 分)
└─ 來自 ---COPILOT_STATUS--- 區塊

層次 2: 自然語言關鍵字（次可靠）
├─ 完成 / 完全完成 (10 分)
├─ 沒有更多工作 (15 分)
└─ 準備就緒 (10 分)

層次 3: 上下文線索（輔助）
└─ 輸出短小 (10 分)

決策: 必須層次 1 + (層次 2 或層次 3)
```

## 程式碼慣例

### 執行邏輯修改指南

- **修改 Copilot 呼叫方式** → `cli_executor.go` 或 `sdk_executor.go`
- **調整完成判斷邏輯** → `response_analyzer.go`
- **改變熔斷條件** → `circuit_breaker.go`
- **新增持久化欄位** → `context.go` + `persistence.go`
- **執行模式調整** → `execution_mode_selector.go`
- **重試策略修改** → `retry_strategy.go`

### 添加新功能的流程

1. 在 `internal/ghcopilot/` 中實作核心邏輯
2. 撰寫對應的單元測試（`*_test.go`）
3. 更新 `RalphLoopClient` 整合新模組
4. 在 `cmd/ralph-loop/main.go` 添加 CLI 命令（如需要）
5. 執行測試驗證：`go test ./...`

### 重要配置參數

```go
// ClientConfig 預設值
config := ghcopilot.DefaultClientConfig()
config.CLITimeout = 60 * time.Second      // Copilot 單次執行超時
config.CLIMaxRetries = 3                  // 失敗重試次數
config.CircuitBreakerThreshold = 3        // 無進展觸發熔斷
config.SameErrorThreshold = 5             // 相同錯誤觸發熔斷
config.Model = "claude-sonnet-4.5"        // AI 模型
config.WorkDir = "."                      // 工作目錄
config.SaveDir = ".ralph-loop/saves"      // 歷史儲存位置
config.EnableSDK = true                   // 啟用 SDK 執行器
config.PreferSDK = true                   // 優先使用 SDK
```

### 測試覆蓋率要求

- 目標覆蓋率：≥ 90%
- 當前覆蓋率：93%
- 總測試數：351 個

## 依賴需求

### 必須安裝

- **Go 1.21+** (專案使用 Go 1.24.5)
- **GitHub Copilot CLI** - 獨立版本 (`copilot` 命令)
  ```bash
  # Windows
  winget install GitHub.Copilot
  
  # 或使用 npm
  npm install -g @github/copilot
  
  # 驗證安裝（需要 ≥ 0.0.389）
  copilot --version
  
  # 認證（需要有效的 GitHub Copilot 訂閱）
  copilot auth
  ```

### 版本注意事項

- ❌ 舊版 `gh copilot` 已於 2025-10-25 停用
- ❌ `@githubnext/github-copilot-cli` 早已棄用
- ✅ 使用獨立 `copilot` CLI（最新版）

## 常見問題處理

### Copilot CLI 超時

**現象**：執行日誌顯示 "⚠️ 執行超時"

**解決方案**：
1. 增加超時設定：`config.CLITimeout = 120 * time.Second`
2. 檢查 Copilot CLI 狀態：`copilot --version`
3. 檢查網路連線與認證

### API Quota 超限

**現象**：錯誤訊息 "402 You have no quota"

**解決方案**：
1. 等待 quota 重置（通常每小時或每月）
2. 使用模擬模式測試：`COPILOT_MOCK_MODE=true`
3. 檢查 GitHub Copilot 訂閱狀態

### 熔斷器觸發

**現象**："circuit breaker opened after X loops"

**原因**：偵測到無進展或重複錯誤

**解決方案**：
1. 重置熔斷器：`./ralph-loop.exe reset`
2. 調整閾值：`config.CircuitBreakerThreshold = 5`
3. 改善 prompt 明確度

## 安全考量

- **自動執行程式碼**：系統會執行 AI 建議的程式碼修改，務必在安全環境測試
- **熔斷機制**：防止無限迴圈消耗資源
- **工作目錄隔離**：建議在測試專案中執行，避免影響重要程式碼
- **API 成本**：每次迴圈消耗 GitHub Copilot API quota，注意用量

## 專案狀態追蹤

執行記錄儲存於 `.ralph-loop/saves/`：

```
.ralph-loop/saves/
├── context_manager_YYYYMMDD_HHMMSS.json  # 完整上下文快照
└── loop_loop-TIMESTAMP-N.json            # 個別迴圈記錄
```

可用於：
- 除錯失敗的迴圈
- 分析 Copilot 回應模式
- 恢復中斷的執行

## 相關文檔

- **README.md** - 專案總覽與快速開始
- **ARCHITECTURE.md** - 詳細架構說明與設計原則
- **CLAUDE.md** - Claude Code 專用開發指南
- **IMPLEMENTATION_COMPLETE.md** - 階段 8 完成報告
- **VERSION_NOTICE.md** - 版本資訊與遷移指南
- **docs/INDEX.md** - 文檔導航索引
