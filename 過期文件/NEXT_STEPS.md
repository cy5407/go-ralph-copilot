# Ralph Loop 下一步待辦事項

**更新日期**: 2026-02-14  
**當前版本**: v0.1.0-stable  
**狀態**: ✅ 穩定版本已發布

---

## 🎉 已完成工作（v0.1.0-stable 發布）

- [x] 清理臨時檔案（.ralph-loop/, .claude/ 等）
- [x] Commit 修改（版本變數 + 輸出顯示）- `f51ba61`
- [x] 創建穩定分支 `stable/v0.1.0-working`
- [x] 創建版本標籤 `v0.1.0-stable`
- [x] 推送 master 分支到 GitHub
- [x] 推送穩定分支到 GitHub
- [x] 推送版本標籤到 GitHub
- [x] 更新 README.md 標註穩定版本 - `72b6ae6`
- [x] 添加版本 badges（Version, Go, License）
- [x] 添加穩定版本安裝說明
- [x] 添加版本歷史章節

---

## 📦 發布資源連結

- **Repository**: https://github.com/cy5407/go-ralph-copilot
- **穩定版本 Tag**: https://github.com/cy5407/go-ralph-copilot/releases/tag/v0.1.0-stable
- **穩定分支**: https://github.com/cy5407/go-ralph-copilot/tree/stable/v0.1.0-working
- **Master 分支**: https://github.com/cy5407/go-ralph-copilot/tree/master

---

## 🎯 優先待辦事項（按順序執行）

### 1. 創建 GitHub Release ⭐ (P0 - 最高優先級)

**目標**: 讓用戶可以直接下載編譯好的 binary

**步驟**:
- [ ] 訪問 https://github.com/cy5407/go-ralph-copilot/releases/new
- [ ] 選擇 tag: `v0.1.0-stable`
- [ ] 標題填寫: `Ralph Loop v0.1.0-stable - 首個穩定版本`
- [ ] 複製 README.md 中「版本歷史」章節內容作為 Release Notes
- [ ] 編譯並上傳 Windows binary:
  ```powershell
  go build -ldflags="-s -w -X main.Version=0.1.0-stable" -o ralph-loop-windows-amd64.exe ./cmd/ralph-loop
  ```
- [ ] 可選：編譯並上傳 Linux/macOS binary
- [ ] 發布 Release

**預期成果**: 用戶可以直接下載 `ralph-loop-windows-amd64.exe` 使用，無需安裝 Go

---

### 2. 驗證用戶體驗 (P0 - 高優先級)

**目標**: 確保新用戶可以順利安裝與使用

**步驟**:
- [ ] 在乾淨的環境測試克隆安裝:
  ```powershell
  cd $env:TEMP
  git clone https://github.com/cy5407/go-ralph-copilot.git test-install
  cd test-install
  git checkout v0.1.0-stable
  go build -o ralph-loop.exe ./cmd/ralph-loop
  .\ralph-loop.exe version
  ```
- [ ] 驗證輸出: `Ralph Loop v0.1.0`
- [ ] 測試基本功能:
  ```powershell
  .\ralph-loop.exe run -prompt "輸出 Hello World" -max-loops 1
  ```
- [ ] 確認 Copilot 輸出正常顯示
- [ ] 清理測試環境:
  ```powershell
  cd ..
  Remove-Item -Recurse -Force test-install
  ```

**預期成果**: 確認安裝流程順暢，無錯誤

---

### 3. 創建 GitHub Workflows (P1 - 中優先級)

**目標**: 自動化測試與發布流程（task3.md T3-001）

**相關文件**: `task3.md`

#### 3.1 創建 `.github/workflows/test.yml`

- [ ] 自動執行 `go test ./...` 在每次 push/PR
- [ ] 測試多個 Go 版本（1.21, 1.24）
- [ ] 報告測試覆蓋率

#### 3.2 創建 `.github/workflows/release.yml`

**注意**: 根據 task3.md，需要修正以下 bug：

- [ ] **Bug 1**: 版本號注入（已修正 main.go，確認 ldflags 使用 `main.Version`）
- [ ] **Bug 2**: Go 版本設為 `1.24`（不是 `1.21`）
- [ ] **Bug 3**: 測試命令使用 `go test ./...`（不是 `go test`）
- [ ] **Bug 4**: 壓縮前刪除檔案邏輯修正（不誤刪 .zip/.tar.gz）
- [ ] **Bug 5**: Release body 格式修正
- [ ] **Bug 6**: 只在 tag push 時觸發（`refs/tags/v*`）

**檔案位置**: `.github/workflows/release.yml`

**建置命令模板**:
```yaml
- name: Build for ${{ matrix.platform }}
  run: |
    go build -ldflags="-s -w -X main.Version=${{ steps.version.outputs.VERSION }}" \
      -o ralph-loop-${{ matrix.platform }} ./cmd/ralph-loop
```

**支援平台**:
- Windows (amd64, arm64)
- Linux (amd64, arm64)
- macOS (amd64, arm64)

---

### 4. 實作 Promise Detection (P2 - 低優先級)

**目標**: 改進完成檢測機制（task2.md 提到但未在當前版本實作）

**背景**: 當前版本使用舊版完成檢測機制，依賴關鍵字匹配

**改進方向**:
- [ ] 研究 Promise Detection 機制設計
- [ ] 實作結構化退出信號 `<promise>任務完成！🥇</promise>`
- [ ] 整合到 `ResponseAnalyzer`
- [ ] 新增單元測試驗證
- [ ] 更新 ARCHITECTURE.md 文檔

**參考**: 
- `internal/ghcopilot/response_analyzer.go`
- commit `d2c8ec1` (Promise Detection 原始實作，但造成問題已回退)

---

### 5. SDK 版本升級 (P2 - 低優先級)

**目標**: 遷移到新版 GitHub Copilot SDK（task2.md T2-019）

**當前狀態**: 
- SDK executor 已實作但因版本不兼容無法使用
- 使用舊版 SDK: `github.com/cy5407/copilot-cli-agent-go v0.1.15-preview.0`

**升級計劃**:
- [ ] 研究新版 SDK: `github.com/github/copilot-cli-sdk-go`
- [ ] 檢查 API 變更與遷移需求
- [ ] 更新 `go.mod` 依賴
- [ ] 修改 `sdk_executor.go` 適配新 API
- [ ] 更新所有相關測試
- [ ] 驗證 SDK/CLI 混合執行器正常工作

**風險**: 可能需要大量程式碼修改，建議在新分支開發

---

### 6. 改進 System Prompt 機制 (P3 - 可選)

**目標**: 解決 System Prompt 導致 AI 忽略用戶任務的問題

**背景**: 
- commit `d2c8ec1` 添加的 System Prompt 導致 AI 將用戶 prompt 當作文檔說明
- 已在 v0.1.0-stable 中移除

**改進方向**:
- [ ] 研究 System Prompt 最佳實踐
- [ ] 實作更清晰的 prompt 結構（System + User）
- [ ] 測試不同的 prompt 順序
- [ ] 驗證 AI 能正確理解並執行任務
- [ ] 添加單元測試與整合測試

**參考**: 
- `internal/ghcopilot/system_prompt.go` (已在 a13543d 之前移除)

---

## 📊 版本資訊

- **當前穩定版本**: v0.1.0-stable
- **Commit**: `72b6ae6` (master) / `f51ba61` (stable tag)
- **分支**: `stable/v0.1.0-working`
- **發布日期**: 2026-02-14

---

## 🔄 開發工作流程建議

### 開發新功能

1. 從 master 創建 feature 分支
   ```bash
   git checkout master
   git pull
   git checkout -b feature/your-feature-name
   ```

2. 開發並測試
   ```bash
   go test ./...
   go build -o ralph-loop.exe ./cmd/ralph-loop
   .\ralph-loop.exe run -prompt "測試新功能" -max-loops 2
   ```

3. Commit 並推送
   ```bash
   git add .
   git commit -m "feat: 添加新功能說明"
   git push origin feature/your-feature-name
   ```

4. 在 GitHub 創建 Pull Request

### 發布新版本

1. 確認所有測試通過
   ```bash
   go test ./...
   ```

2. 更新版本號（main.go + README.md）
   
3. Commit 版本變更
   ```bash
   git commit -m "chore: bump version to v0.2.0"
   ```

4. 創建 tag 並推送
   ```bash
   git tag -a v0.2.0 -m "Release v0.2.0"
   git push origin master
   git push origin v0.2.0
   ```

5. GitHub Actions 自動建置並發布（需要先完成待辦 #3）

---

## 📝 相關文檔

- **task3.md** - GitHub Release 產品化任務清單
- **task2.md** - 技術債務與改進項目
- **ARCHITECTURE.md** - 系統架構說明
- **README.md** - 專案總覽與使用說明

---

**最後更新**: 2026-02-14  
**維護者**: [@cy5407](https://github.com/cy5407)
