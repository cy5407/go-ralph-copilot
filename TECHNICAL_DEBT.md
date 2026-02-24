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

## 待辦清單

| 技術債 | 優先級 | 狀態 |
|--------|--------|------|
| Context 結構精簡化 | Medium | ⏳ 待解決 |
| ~~SDK 版本遷移~~ | ~~Low~~ | ✅ 完成 |

---

**最後更新**: 2026-02-24
