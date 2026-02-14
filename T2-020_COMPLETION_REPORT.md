# T2-020 完成報告：CLI 即時串流輸出

**完成時間**: 2026-02-14  
**狀態**: ✅ 完成  
**優先級**: P0（MVP 必備）

## 📋 任務概述

實現 CLI 即時串流輸出功能，解決 Copilot CLI 執行期間（可能長達 60 秒以上）使用者完全看不到任何進度的問題。

## ✅ 完成內容

### 1. UICallback 介面擴展

**檔案**: `internal/ghcopilot/ui_callback.go`

擴展 `UICallback` 介面，新增兩個串流方法：
```go
type UICallback interface {
    // ... 現有方法
    OnStreamOutput(line string)  // 串流 stdout
    OnStreamError(line string)   // 串流 stderr
}
```

在 `DefaultUICallback` 中：
- 新增 `streamEnabled` 欄位控制串流開關
- 實作即時輸出顯示，帶 `[copilot]` 和 `[copilot:err]` 前綴
- 新增 `NewDefaultUICallbackWithStream` 建構函數

### 2. lineWriter 串流處理器

**檔案**: `internal/ghcopilot/cli_executor.go`

實作 `lineWriter` 結構，提供逐行串流輸出功能：

```go
type lineWriter struct {
    buffer   *bytes.Buffer  // 原始 buffer，保留完整輸出
    callback func(string)   // 每行的回調函數
    scanner  *bufio.Scanner // 逐行掃描器
    mu       sync.Mutex     // 保護並發寫入
    pipe     io.WriteCloser // 管道寫入端
}
```

**技術特點**：
- 使用 `io.Pipe` 和 `bufio.Scanner` 實現逐行處理
- 後台 goroutine 異步處理，不阻塞主執行流程
- `sync.Mutex` 保護並發寫入安全
- 同時寫入原始 buffer 和串流處理器

### 3. CLIExecutor 整合串流

**檔案**: `internal/ghcopilot/cli_executor.go`

修改 `CLIExecutor`：
- 新增 `streamCallback` 和 `streamErrCallback` 欄位
- 新增 `SetStreamCallback(stdout, stderr func(string))` 方法
- 修改 `execute` 方法使用 `lineWriter`：

```go
if ce.streamCallback != nil {
    stdoutLW = newLineWriter(&stdout, ce.streamCallback)
    stdoutWriter = stdoutLW
} else {
    stdoutWriter = &stdout
}
```

### 4. Client 自動整合

**檔案**: `internal/ghcopilot/client.go`

在 `NewRalphLoopClientWithConfig` 初始化時：
```go
// 設置串流回調到 CLI 執行器
client.executor.SetStreamCallback(
    func(line string) {
        if client.uiCallback != nil {
            client.uiCallback.OnStreamOutput(line)
        }
    },
    func(line string) {
        if client.uiCallback != nil {
            client.uiCallback.OnStreamError(line)
        }
    },
)
```

更新 `SetUICallback` 方法確保更換 UI 回調時同步更新串流回調。

### 5. 完整測試套件

**檔案**: `internal/ghcopilot/streaming_test.go`（新檔案）

包含 8 個測試：
- ✅ `TestLineWriter` - 基本逐行功能
- ✅ `TestLineWriterEmptyLines` - 空行處理
- ✅ `TestUICallbackStreamOutput` - 串流輸出格式
- ✅ `TestUICallbackStreamError` - 串流錯誤輸出格式
- ✅ `TestUICallbackStreamQuietMode` - quiet 模式禁用
- ✅ `TestCLIExecutorStreamCallback` - 執行器回調設置
- ✅ `TestCLIExecutorStreamingIntegration` - 整合測試
- ✅ `BenchmarkLineWriter` - 性能測試

**所有測試通過** ✅

## 🎯 驗收標準

### 預期行為
執行：
```bash
.\ralph-loop.exe run -prompt "修復所有測試" -max-loops 5
```

**即時看到輸出**：
```
⏳ 執行 Copilot CLI (單次超時: 1m0s)...
[copilot] 正在分析專案結構...
[copilot] 找到 3 個失敗的測試...
[copilot] 修改 xxx_test.go ...
✅ 執行成功 (耗時: 25s)
```

### 功能驗證

✅ **即時顯示**：執行過程中持續更新，不必等待結束  
✅ **完整保留**：最終結果仍包含完整的 stdout/stderr  
✅ **自動控制**：非 quiet 模式下自動啟用串流  
✅ **向後相容**：不影響現有功能和測試  
✅ **性能優化**：後台處理不阻塞主流程  

## 📊 技術亮點

### 1. 並發安全設計
- 使用 `sync.Mutex` 保護並發寫入
- 使用 `io.Pipe` 實現安全的數據傳遞
- 後台 goroutine 異步處理回調

### 2. 雙重寫入機制
```
輸入數據
   ↓
lineWriter.Write()
   ├→ buffer (完整保存)
   └→ pipe → scanner → callback (即時顯示)
```

### 3. 優雅降級
- quiet 模式自動禁用串流
- 回調為 nil 時回退到傳統模式
- 不影響錯誤處理和重試機制

## 📈 影響範圍

### 修改的檔案
- `internal/ghcopilot/ui_callback.go` - 介面擴展
- `internal/ghcopilot/cli_executor.go` - 核心串流邏輯
- `internal/ghcopilot/client.go` - 自動整合
- `internal/ghcopilot/streaming_test.go` - 新增測試

### 相容性
- ✅ 完全向後相容
- ✅ 所有現有測試通過
- ✅ 不改變現有 API
- ✅ 可選啟用/禁用

## 🎉 成果

### 使用者體驗提升
- **執行透明度**：即時看到 Copilot 在做什麼
- **進度感知**：不再有「黑盒子」體驗
- **更好的除錯**：即時輸出幫助發現問題

### 開發者體驗提升
- **測試覆蓋**：完整的單元測試
- **清晰設計**：模組化、易維護
- **文檔完善**：程式碼註解清楚

## 🔧 使用方式

### 預設行為（自動啟用）
```bash
# 串流輸出自動啟用（非 quiet 模式）
.\ralph-loop.exe run -prompt "任務描述" -max-loops 5
```

### 禁用串流
```bash
# quiet 模式自動禁用串流
.\ralph-loop.exe run -prompt "任務描述" -max-loops 5 -quiet
```

### 程式化使用
```go
client := ghcopilot.NewRalphLoopClient()

// 自訂 UI 回調（自動包含串流功能）
customCallback := ghcopilot.NewDefaultUICallbackWithStream(verbose, quiet, stream)
client.SetUICallback(customCallback)

// 執行迴圈，即時看到輸出
result, err := client.ExecuteLoop(ctx, prompt)
```

## 📝 後續改進建議

1. **進度條整合**：結合串流輸出顯示進度條
2. **彩色編碼**：根據輸出類型使用不同顏色
3. **過濾選項**：允許使用者自訂輸出過濾規則
4. **日誌級別**：支援更細粒度的輸出控制

## ✅ 結論

T2-020 已完成實作並測試通過。串流輸出功能現已整合到 Ralph Loop 系統中，大幅提升了使用者體驗和可觀測性。這是 MVP 必備功能，為後續功能（如進度條、即時除錯）奠定了基礎。
