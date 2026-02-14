# Ralph Loop 實用指南

## 🚀 快速開始

### 第一步：設定環境

```bash
# 1. 確認 Ralph Loop 已編譯
cd "C:\Users\cy5407\Desktop\Github CLI 自動跌代"
.\ralph-loop.exe version
# 應該顯示: Ralph Loop v0.1.0

# 2. 設定 GitHub Copilot 認證
copilot auth
# 或者在 copilot 中輸入 /login
```

---

## 📋 實際應用場景

### 1. 修復編譯錯誤 🔧

**場景**：專案有多個編譯錯誤需要逐一修復

```bash
# 基本用法
.\ralph-loop.exe run -prompt "修復所有 Go 編譯錯誤" -max-loops 15

# 詳細版本
.\ralph-loop.exe run \
  -prompt "逐一修復編譯錯誤，每次修復後執行 go build 驗證" \
  -max-loops 20 \
  -timeout 10m \
  -workdir "."
```

**預期流程**：
1. AI 分析第一個編譯錯誤
2. 生成修復程式碼
3. 執行 `go build` 驗證
4. 如果還有錯誤，繼續下一輪
5. 直到編譯成功或達到迴圈限制

---

### 2. 通過測試 ✅

**場景**：有失敗的測試需要修復

```bash
# 修復特定測試
.\ralph-loop.exe run \
  -prompt "修復 TestCalculator 中的失敗測試，執行 go test -v 驗證" \
  -max-loops 10

# 修復所有測試
.\ralph-loop.exe run \
  -prompt "修復所有失敗的測試，每次修復後執行 go test ./... 驗證，直到全部通過" \
  -max-loops 25
```

**預期流程**：
1. AI 執行 `go test` 查看失敗
2. 分析失敗原因
3. 修復程式碼
4. 重新執行測試
5. 重複直到所有測試通過

---

### 3. 程式碼重構 🔄

**場景**：需要改善程式碼結構但保持功能不變

```bash
.\ralph-loop.exe run \
  -prompt "重構 user.go 中的 GetUser 函數，拆分為更小的函數，保持測試通過" \
  -max-loops 8
```

---

### 4. 實作新功能 ⭐

**場景**：根據規格實作新功能

```bash
.\ralph-loop.exe run \
  -prompt "實作一個計算器類別，支援加減乘除，並寫對應的單元測試，確保測試通過" \
  -max-loops 15
```

---

### 5. 修復安全問題 🛡️

**場景**：修復程式碼掃描發現的安全漏洞

```bash
.\ralph-loop.exe run \
  -prompt "修復 SQL 注入漏洞，使用參數化查詢替換字串拼接，確保功能正常" \
  -max-loops 12
```

---

## 🎯 最佳實踐

### Prompt 撰寫技巧

#### ✅ 好的 Prompt

```bash
# 明確 + 驗證步驟 + 限制範圍
.\ralph-loop.exe run \
  -prompt "修復 models/user.go 中的 ValidateEmail 函數，支援國際化域名，執行 go test ./models -v 驗證，不要修改其他檔案" \
  -max-loops 5
```

#### ❌ 避免的 Prompt

```bash
# 太模糊
.\ralph-loop.exe run -prompt "修復程式碼" -max-loops 10

# 太複雜
.\ralph-loop.exe run -prompt "重寫整個專案架構並加上快取和日誌還有監控" -max-loops 50
```

### 參數調整指南

| 場景類型 | max-loops | timeout | 說明 |
|----------|-----------|---------|------|
| **簡單錯誤修復** | 5-8 | 5m | 1-2 個檔案的小問題 |
| **複雜除錯** | 15-25 | 15m | 多檔案或邏輯複雜 |
| **新功能實作** | 10-20 | 20m | 從零實作功能 |
| **重構** | 8-15 | 10m | 保持功能不變的改進 |
| **測試修復** | 10-20 | 12m | 修復失敗測試 |

---

## 🔍 監控與除錯

### 即時監控

```bash
# 在另一個終端監控狀態
.\ralph-loop.exe watch -interval 3s

# 查看詳細狀態
.\ralph-loop.exe status
```

### 除錯模式

```bash
# 啟用詳細日誌
$env:RALPH_DEBUG = "1"
.\ralph-loop.exe run -prompt "..." -max-loops 5

# 檢視執行歷史
Get-Content .ralph-loop\saves\*.json | ConvertFrom-Json | Format-Table
```

### 熔斷器管理

```bash
# 如果意外觸發熔斷器
.\ralph-loop.exe reset

# 查看熔斷器狀態
.\ralph-loop.exe status
```

---

## ⚠️ 重要注意事項

### 安全考量

1. **在測試環境使用**
   ```bash
   # 先備份重要程式碼
   git add . && git commit -m "備份：使用 Ralph Loop 前"
   
   # 在測試分支執行
   git checkout -b ralph-loop-experiment
   .\ralph-loop.exe run -prompt "..." -max-loops 10
   ```

2. **限制工作目錄**
   ```bash
   # 只在特定目錄工作
   .\ralph-loop.exe run -workdir ".\test-project" -prompt "..." -max-loops 10
   ```

### API 用量管理

- 每次迴圈消耗 GitHub Copilot quota
- 建議在重要工作前檢查 quota 狀態
- 使用合理的 `max-loops` 設定

### 當 Ralph Loop 卡住時

```bash
# 檢查狀態
.\ralph-loop.exe status

# 如果無回應，強制重置
.\ralph-loop.exe reset

# 檢查是否有活躍進程
Get-Process ralph-loop -ErrorAction SilentlyContinue
```

---

## 📝 實際範例工作流

### 範例：修復一個 Go 專案

```bash
# 1. 備份現狀
git add . && git commit -m "開始 Ralph Loop 修復"

# 2. 檢查當前問題
go build ./...
go test ./...

# 3. 啟動自動修復
.\ralph-loop.exe run \
  -prompt "首先執行 go build ./... 檢查編譯錯誤，然後逐一修復。修復完成後執行 go test ./... 檢查測試。確保所有編譯錯誤和測試都通過。" \
  -max-loops 20 \
  -timeout 15m

# 4. 監控進度 (另一終端)
.\ralph-loop.exe watch -interval 5s

# 5. 驗證結果
go build ./...
go test ./...

# 6. 提交改動
git add . && git commit -m "Ralph Loop 自動修復完成"
```

---

## 🎯 成功使用的關鍵

1. **明確的目標** - 具體說明要做什麼
2. **驗證步驟** - 告訴 AI 如何驗證修復
3. **適當範圍** - 不要一次做太多事
4. **安全備份** - 使用 git 保護重要程式碼
5. **漸進式改善** - 從小問題開始，累積信心

**Ralph Loop 現在可以成為你的 AI 程式設計助手，自動處理重複性的修復工作！** 🚀