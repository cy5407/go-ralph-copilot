# Ralph Loop - 程式碼架構總結

## 核心設計原則

### 1. 迴圈控制的三個層次

```
┌─────────────────────────────────┐
│   應用程式層 (app)              │
│   - 整合所有模組                │
│   - 實作業務邏輯                │
└────────────┬────────────────────┘
             │
┌────────────▼────────────────────┐
│   迴圈控制層 (ghcopilot)         │
│   ├─ SDK 執行器（主要）          │
│   ├─ CLI 執行器（回退）          │
│   ├─ 輸出解析器                 │
│   ├─ 回應分析器                 │
│   ├─ 熔斷器                    │
│   └─ 退出偵測                  │
└────────────┬────────────────────┘
             │
┌────────────▼────────────────────┐
│   執行層                        │
│   ├─ Copilot SDK v0.1.26（主）  │
│   └─ github-copilot-cli（備用） │
└─────────────────────────────────┘
```

### 2. 完成退出邏輯 🔑

```go
func (ra *ResponseAnalyzer) IsCompleted() bool {
    status := ra.ParseStructuredOutput()

    // 條件 1: EXIT_SIGNAL = true 單獨就夠（最可靠）
    if status != nil && status.ExitSignal {
        return true
    }

    // 條件 2: 自然語言備用（分數 >= 30 + 至少 2 個指標）
    if ra.completionScore >= 30 && len(ra.completionIndicators) >= 2 {
        return true
    }

    return false
}
```

**完成信號層次**：

```
層次 1: 結構化信號（最可靠，單獨就夠）
└─ EXIT_SIGNAL = true → 直接退出
   來自 ---RALPH_STATUS--- 區塊，包含 REASON 欄位

層次 2: 自然語言備用（需 score ≥ 30 + 2 個指標）
├─ 完成 / 完全完成 (10 分)
├─ 沒有更多工作 (15 分)
└─ 準備就緒 (10 分)
```

### 3. RALPH_STATUS 格式

```
---RALPH_STATUS---
EXIT_SIGNAL: true
REASON: 已完成所有工作，無待辦項目
---END_RALPH_STATUS---
```

對應的 Go 結構：

```go
type CopilotStatus struct {
    Status     string
    ExitSignal bool
    Reason     string
}
```

## 模組互動流程

```
1. 取得使用者請求
   ↓
2. SDKExecutor.Complete()（優先）/ CLIExecutor（回退）
   ├─ SDK: SendAndWait + 事件串流顯示工具執行
   └─ CLI: 執行 github-copilot-cli what-the-shell
   ↓
3. ResponseAnalyzer
   ├─ 解析 RALPH_STATUS 結構化輸出
   ├─ 計算完成分數
   ├─ 偵測卡住狀態
   └─ IsCompleted() ← 關鍵決策點
   ↓
4. CircuitBreaker
   ├─ 記錄進度或錯誤
   └─ 決定是否熔斷
   ↓
5. 判斷:
   ├─ IsCompleted() = true → 😊 退出迴圈
   ├─ CircuitBreaker.IsOpen() = true → ⚠️ 熔斷停止
   └─ 其他 → ↻ 進入下一迴圈
```

## 故障恢復策略

### 熔斷器狀態轉換

```
             ┌─────────────┐
             │   CLOSED    │ ← 正常運作
             └──────┬──────┘
                    │ 失敗×3（無進展/相同錯誤）
                    ↓
             ┌─────────────┐
             │    OPEN     │ ← 停止執行
             └──────┬──────┘
                    │ 成功×1
                    ↓
             ┌─────────────┐
  失敗 ←────┤ HALF_OPEN   │ ← 試探恢復
             └──────┬──────┘
                    │ 成功×1
                    ↓
             ┌─────────────┐
             │   CLOSED    │ ← 恢復正常
             └─────────────┘
```

## 擴展點

### 新增完成指標

在 `response_analyzer.go` 的 `CalculateCompletionScore()` 中新增關鍵字：

```go
newKeywords := []string{"新指標1", "新指標2"}
for _, keyword := range newKeywords {
    if strings.Contains(response, keyword) {
        score += 10
        completionIndicators = append(completionIndicators, keyword)
        break
    }
}
```

### 新增 RALPH_STATUS 欄位

在 `response_analyzer.go` 的 `CopilotStatus` 結構新增欄位，並在 `ParseStructuredOutput()` 中加入對應解析。

---

**設計文件版本**: 2.0  
**上次更新**: 2026-02-24  
**相關文件**: [TECHNICAL_DEBT.md](./TECHNICAL_DEBT.md), [task3.md](./task3.md)
