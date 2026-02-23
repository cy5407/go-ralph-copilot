# Task-20260224 - Ralph Loop Bug Fix Plan

## 問題來源

在 PornActressDB-Golang-Migration 專案中執行以下指令：
```
.\ralph-loop.exe run -prompt "檢查requirements.txt是否需要更新，若有更新完成就git push" -max-loops 15
```

---

## Bug 1：Permission denied（高優先）

### 現象
```
✗ Edit src\services\go_bridge.py
  Permission denied and could not request permission from user

✗ Test write access via Python
  $ python -c "..."
  Permission denied and could not request permission from user
```

### 分析
- 已加入 `--allow-all-tools`, `--allow-all-paths`, `--allow-all-urls`, `--allow-tool write`, `--allow-tool shell`
- 但 Edit 工具和 Shell 工具仍被拒絕
- 懷疑原因：`--allow-tool shell` 沒有包含子命令語法，正確格式應為 `shell(*)` 或無參數的 `shell`
- 另一可能：Copilot CLI 版本對 `--allow-tool write` 的 `write` 關鍵字有不同解釋

### 修法
```go
// 改用 --yolo 一次搞定所有權限，等同 --allow-all-tools --allow-all-paths --allow-all-urls
// 這是官方推薦的自動化腳本用法
args = append(args, "--yolo")
// 移除多餘的個別 allow-tool
```

---

## Bug 2：任務跑偏（高優先）

### 現象
- 使用者說：「檢查 requirements.txt 是否需要更新」
- Copilot 卻讀了 `Task-20260223.md` → `skill(ralph-loop)` → 開始做 cache manager 整合

### 分析
- 專案根目錄有 `Task-20260223.md` 和 `AGENTS.md`/`.claude/commands/ralph-loop.md`
- Copilot 讀了這些 instruction 檔案，把任務重新詮釋成「按照 task file 執行」
- 問題根源：`--no-custom-instructions` 沒有加，Copilot 自動載入 AGENTS.md

### 修法
```go
// 加入 --no-custom-instructions 防止 Copilot 自動讀取 AGENTS.md / .claude/ 等指令檔
args = append(args, "--no-custom-instructions")
```

---

## Bug 3：`error: unknown option '--no-warnings'` 造成 exit code 1

### 現象
```
● Check git status
  $ git status --short | head -5
  └ 7 lines...

error: unknown option '--no-warnings'
Try 'copilot --help' for more information.
```

### 分析
- Shell 工具執行成功，有輸出（7 lines）
- 但 Copilot 本身因某個內部問題輸出了 `error: unknown option '--no-warnings'` 並以 exit code 1 結束
- 這是 Copilot CLI 的 bug，不是我們的問題
- 影響：ralph-loop 把有輸出但 exit code 1 當作失敗重試

### 修法
- 已有「exit code != 0 但有 stdout → 先解析」的邏輯，應已緩解
- 確認 `response_analyzer.go` 能從有部分成功輸出的結果中偵測完成狀態

---

## 行動清單

| 優先 | 檔案 | 修改 |
|------|------|------|
| P0 | `cli_executor.go` buildArgs() | 改用 `--yolo` 取代所有個別 allow 旗標 |
| P0 | `cli_executor.go` buildArgs() | 加入 `--no-custom-instructions` |
| P1 | 驗證 | 重新測試 Edit 工具是否能正常執行 |
