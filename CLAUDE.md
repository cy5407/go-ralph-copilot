# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 語言偏好

**所有回應必須使用繁體中文。**

## 專案概述

Ralph Loop 是一個 AI 驅動的自動程式碼迭代系統，以 **GitHub Copilot SDK（主要）+ CLI（回退）** 實現「觀察→反思→行動」的自主修正迴圈。

**核心概念**：透過 GitHub Copilot SDK v0.1.26 分析任務與執行工具，自動生成修復方案並持續迭代，直到問題解決或達到預設的安全限制。

## 開發命令

### 建置與執行

```bash
# 建置主程式
go build -o ralph-loop.exe ./cmd/ralph-loop

# 執行自動修復迴圈
./ralph-loop.exe run -prompt "修復所有編譯錯誤" -max-loops 10 -timeout 5m

# 查看狀態
./ralph-loop.exe status

# 重置熔斷器
./ralph-loop.exe reset

# 監控模式
./ralph-loop.exe watch -interval 3s
```

### 測試

```bash
# 執行所有測試
go test ./...

# 執行特定套件測試
go test ./internal/ghcopilot

# 詳細輸出
go test -v ./internal/ghcopilot

# 執行特定測試
go test -v -run TestCLIExecutor ./internal/ghcopilot
```

### 除錯模式

```bash
# 啟用詳細日誌（顯示 CLI 執行細節、重試、超時等）
RALPH_DEBUG=1 ./ralph-loop.exe run -prompt "..." -max-loops 5
```

### 模擬模式

```bash
# 使用模擬 Copilot 回應（不消耗 API quota）
COPILOT_MOCK_MODE=true ./ralph-loop.exe run -prompt "測試" -max-loops 3
```

## 架構設計

### 核心迴圈流程

```
使用者啟動 ralph-loop
    ↓
[迴圈 N] RalphLoopClient.ExecuteLoop()
    ↓
├─ SDKExecutor（主要）→ Copilot SDK v0.1.26
│   ├─ SendAndWait + 事件串流顯示工具執行
│   └─ OnPermissionRequest: ApproveAll（自動授權）
│
├─ CLIExecutor（回退，SDK 失敗時）
│   └─ 呼叫 `copilot -p "prompt"`
│
├─ OutputParser → 解析 Copilot 輸出
│   └─ 提取程式碼區塊
│   └─ 提取結構化狀態
│
├─ ResponseAnalyzer → 分析回應
│   └─ 完成偵測（尋找 "✅", "完成" 等關鍵字）
│   └─ 卡住偵測（輸出無變化）
│
├─ CircuitBreaker → 熔斷保護
│   ├─ 無進展迴圈計數（預設 3 次觸發）
│   └─ 相同錯誤計數（預設 5 次觸發）
│
├─ ContextManager → 歷史管理
│   └─ 記錄每個迴圈的 prompt、輸出、錯誤
│
└─ PersistenceManager → 持久化
    └─ 儲存至 .ralph-loop/saves/
```

### 關鍵模組說明

**`internal/ghcopilot/`** - GitHub Copilot 整合層

- `client.go` - **主要 API 入口點**，`RalphLoopClient` 整合所有模組
- `sdk_executor.go` - **GitHub Copilot SDK 執行器（主要）**，SendAndWait + 事件串流
- `cli_executor.go` - GitHub Copilot CLI 執行器（SDK 失敗時回退）
- `output_parser.go` - 解析 Copilot 輸出（程式碼區塊、選項）
- `response_analyzer.go` - 判斷是否應繼續或退出迴圈
- `circuit_breaker.go` - 防止無限迴圈的安全機制
- `context.go` - 管理迴圈歷史與上下文累積
- `persistence.go` - 儲存/載入執行記錄
- `exit_detector.go` - 優雅退出決策（EXIT_SIGNAL 單獨就夠，備用 score ≥ 30）
- `cli_executor.go` - GitHub Copilot CLI 執行器（SDK 失敗時回退）

**`cmd/ralph-loop/main.go`** - CLI 入口

支援的子命令：`run`, `status`, `reset`, `watch`, `version`, `help`

## 重要配置

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
```

### 執行器選項

```go
opts := ghcopilot.DefaultOptions()
opts.Model = ghcopilot.ModelClaudeSonnet45
opts.Silent = true              // 靜默模式（減少輸出）
opts.AllowAllTools = true       // 允許 Copilot 使用所有工具
opts.NoAskUser = true           // 自主模式（不詢問使用者）
```

## 完成檢測機制

系統透過以下方式判斷任務是否完成：

1. **結構化退出信號**（最可靠，單獨就夠）
   - `---RALPH_STATUS---` 區塊中 `EXIT_SIGNAL: true`
   - `REASON:` 欄位說明退出原因
2. **自然語言備用**（score ≥ 30 + 至少 2 個指標）
   - 關鍵字：「完成」、「沒有更多工作」等
3. **熔斷器** - 無進展或重複錯誤達閾值

## 常見問題處理

### Copilot CLI 超時

**現象**：執行日誌顯示 "⚠️ 執行超時"

**解決**：
1. 增加超時設定：`config.CLITimeout = 120 * time.Second`
2. 檢查 Copilot CLI 狀態：`copilot --version`
3. 檢查網路連線與認證

### API Quota 超限

**現象**：錯誤訊息 "402 You have no quota"

**解決**：
1. 等待 quota 重置（通常每小時或每月）
2. 使用模擬模式測試：`COPILOT_MOCK_MODE=true`
3. 檢查 GitHub Copilot 訂閱狀態

### 熔斷器觸發

**現象**："circuit breaker opened after X loops"

**原因**：偵測到無進展或重複錯誤

**解決**：
1. 重置熔斷器：`./ralph-loop.exe reset`
2. 調整閾值：`config.CircuitBreakerThreshold = 5`
3. 改善 prompt 明確度

## 依賴需求

### 必須安裝

- **Go 1.21+** - 專案使用 Go 1.24.5
- **GitHub Copilot CLI** - 獨立版本 (`copilot` 命令)
  - 安裝：`winget install GitHub.Copilot` 或 `npm install -g @github/copilot`
  - 驗證：`copilot --version` (需要 ≥ 0.0.389)
  - 認證：`copilot auth` (需要有效的 GitHub Copilot 訂閱)

### 版本注意事項

- ❌ **舊版 `gh copilot`** 已於 2025-10-25 停用
- ❌ **`@githubnext/github-copilot-cli`** 早已棄用
- ✅ 使用 **獨立 `copilot` CLI** (最新版)

## 開發工作流程

### 添加新功能

1. 在 `internal/ghcopilot/` 中實作核心邏輯
2. 撰寫單元測試（`*_test.go`）
3. 更新 `RalphLoopClient` 整合新模組
4. 在 `cmd/ralph-loop/main.go` 添加 CLI 命令（如需要）
5. 執行測試驗證：`go test ./...`

### 修改執行邏輯

- **修改 Copilot 呼叫方式** → `cli_executor.go`
- **調整完成判斷邏輯** → `response_analyzer.go`
- **改變熔斷條件** → `circuit_breaker.go`
- **新增持久化欄位** → `context.go` + `persistence.go`

## 測試策略

### 多迴圈測試場景

要測試真正的多迴圈處理，需要設計**需要多次驗證的任務**：

```bash
# 範例：創建包含多個 bug 的程式和測試
# 強制 AI 必須「修復 → 測試 → 發現新錯誤 → 修復」循環

cd test-multiloop
go test -v  # 應該失敗
../ralph-loop.exe run -prompt "逐一修復所有測試失敗，每次修復後執行 go test 驗證" -max-loops 10
```

**實際觀察**：Copilot 通常很聰明，會一次性修復多個問題。要真正分步，需要：
- 外部強制驗證機制
- 漸進式目標設定
- 明確的分階段 prompt

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

## 安全考量

- **自動執行程式碼**：系統會執行 Copilot 建議的程式碼修改，務必在安全環境測試
- **熔斷機制**：防止無限迴圈消耗資源
- **工作目錄隔離**：建議在測試專案中執行，避免影響重要程式碼
- **API 成本**：每次迴圈消耗 GitHub Copilot API quota，注意用量

## 相關資源

- **README.md** - 專案總覽與快速開始
- **診斷報告.md** - 已知問題與修復記錄
- **OpenSpec/** - 原始設計規格（已整合 SDK）
- **archive/old-tests/** - 舊版測試檔案（含 build tags）
