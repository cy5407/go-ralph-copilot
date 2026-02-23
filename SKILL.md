# SKILL: ralph-loop 自動迭代工具

> Single binary CLI，無需安裝任何依賴。
> 將 ralph-loop.exe 放在專案根目錄，透過 `./ralph-loop` 呼叫。

---

## 基本語法

```
./ralph-loop run -prompt "<任務描述>" -max-loops <次數> [-workdir <路徑>]
```

---

## SKILL 清單

### ✅ 驗證建置
```bash
./ralph-loop run -prompt "執行 go build ./... 修復所有編譯錯誤，直到建置成功" -max-loops 5
```

### ✅ 驗證測試
```bash
./ralph-loop run -prompt "執行 go test ./... 修復所有失敗的測試，直到全部通過" -max-loops 10
```

### ✅ 完整驗證（建置 + 測試）
```bash
./ralph-loop run -prompt "先執行 go build ./... 確認建置成功，再執行 go test ./... 確認測試全過，兩者都通過才算完成" -max-loops 15
```

### 🔧 修復特定錯誤
```bash
./ralph-loop run -prompt "<貼上錯誤訊息>，請分析並修復" -max-loops 10
```

### 📋 執行任務清單
```bash
./ralph-loop run -prompt "閱讀 task.md 並逐一完成所有任務，每完成一項回報進度" -max-loops 20
```

### 🔍 程式碼審查與修復
```bash
./ralph-loop run -prompt "審查目前專案程式碼，找出潛在問題並修復" -max-loops 10
```

---

## 其他指令

```bash
# 查看目前狀態
./ralph-loop status

# 熔斷器觸發後重置
./ralph-loop reset

# 監控執行中的迴圈
./ralph-loop watch -interval 3s
```

---

## 在其他專案使用

將 ralph-loop.exe 複製到目標專案根目錄，然後在該目錄下執行：

```bash
./ralph-loop run -prompt "修復所有測試失敗" -max-loops 10
```

---

## Agent 呼叫規則

1. 任務描述越具體越好，包含「如何驗證完成」
2. `max-loops` 依任務複雜度調整（簡單修復用 5，複雜任務用 20）
3. 若熔斷器觸發（卡住），先執行 `./ralph-loop reset` 再重試
4. 失敗時查看 `./ralph-loop status` 了解原因
