# Task-20260224 - Promise Detection 完成偵測機制重構

## 問題背景

目前的完成偵測機制有兩套：
1. **RALPH_STATUS 結構化輸出** — 要求 Copilot 輸出 `---RALPH_STATUS--- / EXIT_SIGNAL: true`
2. **自然語言分數系統** — 掃描關鍵字（"完成"、"無需 push" 等）累計分數

**實際結果**：兩套都不可靠。
- Copilot 經常不輸出 RALPH_STATUS（它不是被強制的）
- 自然語言分數系統問題多：Markdown 干擾、門檻調不好、關鍵字清單永遠不夠全、容易誤判也容易漏判
- 結果：任務明明完成了，迴圈卻一直跑到 maxLoops 才停

## 參考：保哥的 copilot-ralph 做法

保哥的 `doggy8088/copilot-ralph` 使用 **Promise Detection**，核心只有一招：

```typescript
// 偵測邏輯 — 就這麼簡單
const detectPromise = (text: string, promisePhrase: string): boolean => {
  const wrapped = `<promise>${promisePhrase}</promise>`;
  return text.includes(wrapped);
};
```

在 System Prompt 中告訴 AI：
> 任務完全完成時，在回應的**最後**輸出：`<promise>任務完成！</promise>`
> - 必須是回應的最後字元
> - 不要包在 code block 裡
> - 任務沒完成就**不要輸出**

**優點**：
- 極其精確，不可能被自然語言誤觸
- 偵測邏輯只有一行 `text.includes()`
- 不需要維護關鍵字清單、不需要分數計算

**缺點**：
- 完全依賴 AI 遵守指示，如果 AI 不輸出標記就只能靠 maxIterations 兜底
- 沒有熔斷器、沒有卡住偵測

## 設計方向：結合雙方優點

### 核心思路

| 層級 | 機制 | 來源 | 說明 |
|------|------|------|------|
| **Layer 1** | `<ralph-done>` 精確標記偵測 | 學自保哥 | 主要偵測手段，簡潔可靠 |
| **Layer 2** | RALPH_STATUS 結構化輸出 | 我們原有 | 保留作為備用，向下相容 |
| **Layer 3** | 自然語言關鍵字偵測 | 我們原有 | **大幅簡化**，僅作最後防線 |
| **安全網** | 熔斷器 + maxLoops + timeout | 我們原有 | 保留不動 |

### 為什麼不完全照抄保哥？

1. 保哥用的是 **Copilot SDK**（程式化 API），可以設定 System Prompt；我們用的是 **Copilot CLI**（`copilot -p`），只能透過 user prompt 傳遞指示，AI 遵從度較低
2. 我們已經有熔斷器和卡住偵測，這些是保哥沒有的安全機制，值得保留
3. 完全砍掉 RALPH_STATUS 會破壞向下相容（舊版 prompt 仍然會輸出這個格式）

---

## 具體修改計畫

### 1. 新增 Promise Detection（`response_analyzer.go`）

新增 `detectPromise()` 函式，偵測 `<ralph-done>` 標記：

```go
// promiseTag 是精確完成標記，不可能被自然語言誤觸
const promiseTag = "<ralph-done>"

// detectPromise 偵測 AI 輸出中是否包含完成標記
func detectPromise(text string) bool {
    return strings.Contains(text, promiseTag)
}
```

**為什麼用 `<ralph-done>` 而不是保哥的 `<promise>任務完成！</promise>`？**
- 更短，AI 更容易遵守
- 不含中文和 emoji，避免編碼問題
- 固定字串，不需要配置 promisePhrase
- 單一 tag 不需要配對開合（`<promise>...</promise>` 中間的內容變成額外變數）

### 2. 修改 `IsCompleted()` 優先級（`response_analyzer.go`）

```go
func (ra *ResponseAnalyzer) IsCompleted() bool {
    // Layer 1：精確標記偵測（最高優先，學自保哥）
    if detectPromise(ra.response) {
        return true
    }

    // Layer 2：RALPH_STATUS 結構化輸出（向下相容）
    status := ra.ParseStructuredOutput()
    if status != nil && status.ExitSignal {
        return true
    }

    // Layer 3：自然語言備用（僅作最後防線，門檻維持 ≥ 20 + 1 指標）
    if ra.completionScore >= 20 && len(ra.completionIndicators) >= 1 {
        return true
    }

    return false
}
```

### 3. 修改 Prompt 引導（`client.go`）

將 `ralphStatusSuffix` 改為同時引導 `<ralph-done>` 和 RALPH_STATUS：

```go
const ralphStatusSuffix = `

任務完成時，請在回應的最後一行輸出：
<ralph-done>

如果任務尚未完成，不要輸出上面的標記，繼續執行任務。`
```

**關鍵變更**：
- 移除 RALPH_STATUS 格式要求（AI 通常不會遵守多行結構化格式）
- 改用單行 `<ralph-done>` 標記（簡單到 AI 幾乎一定會配合）
- 指示放在 prompt 尾部（保持現有的「使用者 prompt 在前」策略）

### 4. 簡化 `CalculateCompletionScore()`（`response_analyzer.go`）

Layer 3 自然語言偵測保留但簡化：
- **保留** `stripMarkdown()` 和 `completionKeywords` / `noWorkPatterns` 比對
- **移除** `short_output`（輸出長度 < 500 加分）— 這個指標太不可靠
- **保留** 現有門檻 `≥ 20 + 1 指標`

```go
func (ra *ResponseAnalyzer) CalculateCompletionScore() int {
    score := 0

    // Promise 標記命中
    if detectPromise(ra.response) {
        score += 100
        ra.completionIndicators = append(ra.completionIndicators, "promise_tag")
    }

    // RALPH_STATUS 結構化輸出
    status := ra.ParseStructuredOutput()
    if status != nil && status.ExitSignal {
        score += 100
        ra.completionIndicators = append(ra.completionIndicators, "explicit_exit_signal")
    }

    cleaned := strings.ToLower(stripMarkdown(ra.response))

    // 完成關鍵字（+10）
    // ... 保持不變 ...

    // 無工作模式（+15）
    // ... 保持不變 ...

    // 移除 short_output 指標

    ra.completionScore = score
    return score
}
```

### 5. 更新測試（`response_analyzer_test.go`）

新增測試案例：

```go
func TestPromiseDetection(t *testing.T) {
    // 有 <ralph-done> 標記 → 完成
    ra1 := NewResponseAnalyzer("任務已完成，所有檔案已更新。\n<ralph-done>")
    ra1.CalculateCompletionScore()
    if !ra1.IsCompleted() {
        t.Error("有 <ralph-done> 標記應視為完成")
    }

    // 標記嵌在中間也可以偵測
    ra2 := NewResponseAnalyzer("結果如下：\n<ralph-done>\n以上。")
    ra2.CalculateCompletionScore()
    if !ra2.IsCompleted() {
        t.Error("<ralph-done> 在中間也應被偵測")
    }

    // 沒有標記、沒有其他信號 → 不完成
    ra3 := NewResponseAnalyzer("我正在處理中，請稍候。")
    ra3.CalculateCompletionScore()
    if ra3.IsCompleted() {
        t.Error("無任何完成信號不應視為完成")
    }

    // <ralph-done> 在 code block 裡不應該被偵測？
    // → 不需要特別處理，因為這個標記本身就不是自然語言，
    //   AI 不會在 code block 裡「意外」產生它
}
```

更新 `TestDualConditionVerification` 以反映新的優先級。

### 6. 不需要修改的部分

| 模組 | 原因 |
|------|------|
| `circuit_breaker.go` | 熔斷器邏輯不變，繼續保護無限迴圈 |
| `exit_detector.go` | 退出偵測器不變 |
| `cli_executor.go` | CLI 執行邏輯不變 |
| `output_parser.go` | 輸出解析不變 |
| `client.go` ExecuteLoop 流程 | 只改 `ralphStatusSuffix` 常數，流程不動 |

---

## 修改檔案清單

| 檔案 | 修改項目 |
|------|----------|
| `internal/ghcopilot/response_analyzer.go` | 新增 `detectPromise()`；修改 `IsCompleted()` 加入 Layer 1；`CalculateCompletionScore()` 移除 short_output |
| `internal/ghcopilot/client.go` | 修改 `ralphStatusSuffix` 改用 `<ralph-done>` 引導 |
| `internal/ghcopilot/response_analyzer_test.go` | 新增 `TestPromiseDetection`；更新 `TestDualConditionVerification` |

---

## 預期效果

### 修改前（現狀）
```
Copilot 回覆：「分支與 origin/master **已同步**（**無需 push**）」
→ RALPH_STATUS：未輸出 ❌
→ 自然語言分數：15 分（< 20 門檻）❌
→ 結果：繼續迴圈 → 跑滿 maxLoops 才停
```

### 修改後（預期）
```
Copilot 回覆：「分支與 origin/master 已同步，無需 push。\n<ralph-done>」
→ Promise 標記：命中 ✅
→ 結果：第 1 迴圈就停止
```

### 最差情況（AI 不配合）
```
Copilot 回覆：「不需要 push」（沒有標記，沒有 RALPH_STATUS）
→ Promise 標記：未命中 ❌
→ RALPH_STATUS：未輸出 ❌
→ 自然語言分數：15 分 + 「不需要 push」命中 noWorkPatterns
→ 如果同時命中 completionKeywords → 25 分 ≥ 20 + 2 指標 → 完成 ✅
→ 如果只命中一類 → 15 分 < 20 → 繼續迴圈 → 靠 maxLoops 兜底
```

---

## 風險評估

| 風險 | 可能性 | 影響 | 緩解 |
|------|--------|------|------|
| AI 完全不輸出 `<ralph-done>` | 中 | 退化到 Layer 2/3 或 maxLoops | Prompt 指示夠簡潔，遵從率應高於 RALPH_STATUS |
| AI 在未完成時誤輸出標記 | 極低 | 提前停止迴圈 | `<ralph-done>` 不是自然語言，不會被意外產生 |
| 破壞現有測試 | 低 | CI 失敗 | 計畫中已包含測試更新 |
| 向下相容問題 | 無 | — | RALPH_STATUS 仍然作為 Layer 2 保留 |
