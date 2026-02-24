# Ralph Loop 下一步待辦事項

**更新日期**: 2026-02-24  
**當前版本**: v0.1.0-stable（SDK v0.1.26）

---

## 🎯 待辦事項（按優先順序）

### 1. 創建 GitHub Release ⭐（P0 - 最高優先）

**目標**: 讓使用者可直接下載編譯好的 binary，詳見 `task3.md`

- [ ] 訪問 https://github.com/cy5407/go-ralph-copilot/releases/new
- [ ] 選擇 tag: `v0.1.0-stable`
- [ ] 編譯並上傳 Windows binary:
  ```powershell
  go build -ldflags="-s -w -X main.Version=0.1.0-stable" -o ralph-loop-windows-amd64.exe ./cmd/ralph-loop
  ```

---

### 2. 驗證用戶體驗（P0）

在乾淨環境測試安裝與基本執行流程是否正常。

---

### 3. 創建 GitHub Workflows（P1）

**目標**: 自動化測試與發布流程，詳見 `task3.md`（T3-001 ~ T3-005 全部未完成）

- [ ] `.github/workflows/test.yml`（每次 push 執行 `go test ./...`）
- [ ] `.github/workflows/release.yml`（tag push 時自動建置多平台 binary）

---

### 4. 改進 System Prompt 機制（P3 - 可選）

研究 System Prompt 最佳實踐，避免 AI 將用戶 prompt 誤解為文件說明。

---

## ✅ 已完成

- [x] SDK 版本升級至 v0.1.26（含 lazy-start、事件串流、自動權限放行）
- [x] Permission denied 修復（`PermissionHandler.ApproveAll`）
- [x] RALPH_STATUS / REASON 欄位解析
- [x] Promise Detection 評估 → **暫緩**（詳見 `Task-20260224-fix-Promise-Detection.md`）

---

**相關文件**: `task3.md`（GitHub Release）、`ARCHITECTURE.md`、`TECHNICAL_DEBT.md`  
**維護者**: [@cy5407](https://github.com/cy5407)
