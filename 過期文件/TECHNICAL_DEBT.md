# 技術債清單

> 已識別但暫未解決的架構改進項目
> 優先級：Medium / 計劃解決時間：階段 9+ 

---

## 1. Context 結構精簡化 (優先級: Medium)

**問題識別時間**: 2026-01-21  
**識別者**: User  
**狀態**: ⏳ 待解決

### 當前問題

`ExecutionContext` 存在資訊重複與冗餘：

```go
// 冗餘部分（可由 SDK 提供）
CLICommand      string
CLIOutput       string
CLIExitCode     int
ParsedCodeBlocks []string
ParsedOptions    []string
CleanedOutput    string
```

SDK (`github.com/github/copilot-sdk/go`) 已提供完整的：
- ✅ 執行結果
- ✅ 錯誤報告  
- ✅ Session 管理
- ✅ 命令執行結果

### 建議解決方案

**改為適配層模式**：

```go
// 簡化後的 ExecutionContext
type ExecutionContext struct {
    // 迴圈身份
    LoopID      string
    LoopIndex   int
    Timestamp   time.Time
    
    // SDK 執行結果 (直接來自 SDK，避免冗餘)
    SDKResponse interface{}  // SDK 的完整返回
    SDKError    error        // SDK 的錯誤
    
    // 本地分析 (自己的邏輯)
    CircuitBreakerState string   // 熔斷器狀態
    ExitReason          string   // 退出理由
    
    // 持久化 (本地存儲)
    SavedAt     time.Time
    SavedPath   string
}
```

### 影響範圍

- context.go (主模組)
- context_test.go (13 個測試需更新)
- persistence.go (適配調整)

### 解決時機

- 完成階段 8 (API 設計) 後
- 當 SDK 集成層清晰時執行

### 相關檔案

- [context.go](./internal/ghcopilot/context.go)
- [context_test.go](./internal/ghcopilot/context_test.go)
- [sdk_poc_test.go](./test/sdk_poc_test.go)

---

## 2. ~~SDK 版本遷移計劃~~ ✅ 已完成

**問題識別時間**: 2026-01-21
**完成時間**: 2026-02-24
**狀態**: ✅ 已解決

### 已完成遷移

```
github.com/github/copilot-sdk/go v0.1.26  (已升級，作為主要執行方式)
```

### 已實作功能

- ✅ `sdk_executor.go` 完整整合
- ✅ `PermissionHandler.ApproveAll` + `OnPreToolUse` hook
- ✅ `SendAndWait` 結構化回應
- ✅ 事件串流顯示工具執行過程
- ✅ Lazy-start 架構 + CLI 自動回退

---

## 待辦清單

| 技術債 | 優先級 | 狀態 | 目標完成 | 預估工時 |
|--------|--------|------|---------|---------|
| Context 結構精簡化 | Medium | ⏳ | 階段 9 | 1-2 天 |
| ~~SDK 版本遷移~~ | ~~Low~~ | ✅ | 2026-02-24 完成 | — |

---

**最後更新**: 2026-02-24
**下次審查**: 階段 9 開始前
