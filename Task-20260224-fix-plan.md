# Task-20260224 - Ralph Loop Bug Fix Plan

## 問題來源

在 PornActressDB-Golang-Migration 專案中執行以下指令：
```
.\ralph-loop.exe run -prompt "檢查requirements.txt是否需要更新，若有更新完成就git push" -max-loops 15
```

---

## ✅ Bug 1：Permission denied（已修復）

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
- 根本原因：`--allow-tool shell` 沒有包含子命令語法，正確格式應為 `shell(*)` 或無參數的 `shell`
- 另一原因：個別 allow 旗標組合不可靠

### 修法（已實施）
- `cli_executor.go:156-160` — 改用 `--yolo` 取代所有個別 allow 旗標
- `--yolo` 是 `--allow-all` 的別名，等同 `--allow-all-tools` + `--allow-all-paths` + `--allow-all-urls`
- **Commit**: `97d1e04`（Bug-12）

### 驗證
- `buildArgs()` 確認當 `AllowAllTools || AllowAllPaths || AllowAllURLs` 時統一使用 `--yolo`
- 不再產生個別 `--allow-tool write/shell` 等參數

---

## ✅ Bug 2：任務跑偏（已修復）

### 現象
- 使用者說：「檢查 requirements.txt 是否需要更新」
- Copilot 卻讀了 `Task-20260223.md` → `skill(ralph-loop)` → 開始做 cache manager 整合

### 分析
- 專案根目錄有 `Task-20260223.md` 和 `AGENTS.md`/`.claude/commands/ralph-loop.md`
- Copilot 讀了這些 instruction 檔案，把任務重新詮釋成「按照 task file 執行」
- 問題根源：Copilot 自動載入 AGENTS.md / `.github/copilot-instructions.md` / `CLAUDE.md` 等指令檔

### 修法（已實施）
- `cli_executor.go:168` — 無條件加入 `--no-custom-instructions`
- **Commit**: `97d1e04`（Bug-13）

### `--no-custom-instructions` 封鎖範圍（官方文件確認）
| 被封鎖的路徑 | 說明 |
|---|---|
| `.github/copilot-instructions.md` | 倉庫層級指令 |
| `.github/instructions/**/*.instructions.md` | 路徑層級指令 |
| `AGENTS.md`（git 根目錄 + 工作目錄） | Agent 指令 |
| `CLAUDE.md`（倉庫根目錄） | Claude 相容指令 |
| `GEMINI.md`（倉庫根目錄） | Gemini 相容指令 |
| `$HOME/.copilot/copilot-instructions.md` | 使用者個人指令 |

### ⚠️ 殘留問題
- `--no-custom-instructions` **不能阻止** Copilot 讀取 `.claude/commands/` 下的 skill 檔案
- 這屬於 **Open-02** 待處理問題

---

## ✅ Bug 3：`error: unknown option '--no-warnings'` 造成 exit code 1（已修復）

### 現象
```
● Check git status
  $ git status --short | head -5
  └ 7 lines...

error: unknown option '--no-warnings'
Try 'copilot --help' for more information.
```

### 分析
- 這是 Copilot CLI 的已知 Bug（GitHub Issue #1446，重複自 #1399）
- **根本原因**：Copilot CLI 二進位檔內部設定了 `NODE_OPTIONS=--no-warnings` 環境變數，洩漏到子程序中。git 不認識此 Node.js 專用旗標因此報錯
- 影響版本：v0.0.409（WinGet 安裝）
- **實際影響**：純粹外觀問題，無功能性影響
- **官方修復**：已合併，預期在 2026-02-13 後的版本釋出

### 修法（已實施）
1. `cli_executor.go:656-696` — 實作 `filteredWriter`，在 stderr 輸出前過濾噪音行
2. `cli_executor.go:666-668` — `noisePatterns` 包含 `"error: unknown option '--no-warnings'"`
3. `client.go:253-261` — exit code != 0 但有 stdout 時走正常解析流程，不算失敗
4. **Commit**: `8a56a22`（Open-03 修復）

---

## ❌ Open-01：Permission denied 透過 MCP skill 中轉（待修復）

### 現象
- Copilot 使用 `skill(package-audit)` 等 MCP skill 時，shell 在 skill 沙盒執行
- `--yolo` 只控制 Copilot 主程序的權限，管不到 MCP skill 沙盒內的工具
- 整個任務因 skill 沙盒的權限限制卡死

### 根本原因
- MCP skill 有自己獨立的權限系統
- `--yolo` / `--allow-all-tools` 只對 Copilot 主程序的直接工具呼叫有效
- 當 Copilot 主動決定使用 MCP skill 時，skill 內部的 shell/write 操作不受主程序權限控制

### 修復方案

**方案 A：禁用所有內建 MCP 伺服器（推薦）**
```go
// cli_executor.go buildArgs()
args = append(args, "--disable-builtin-mcps")
```
- 完全禁用所有內建 MCP 伺服器（目前是 `github-mcp-server`）
- 強制 Copilot 只使用直接的 shell/write 工具，不透過 MCP 中轉

**方案 B：使用 `--deny-tool` 禁止特定 MCP 伺服器**
```go
// 如果只想禁止特定的 MCP 伺服器
for _, server := range ce.options.DeniedMCPServers {
    args = append(args, "--disable-mcp-server", server)
}
```

**方案 C：使用 `--excluded-tools` 排除特定工具**
```go
args = append(args, "--excluded-tools", "skill")
```

**建議**：採用 **方案 A**（`--disable-builtin-mcps`），因為 ralph-loop 的使用場景不需要 MCP 伺服器。如果日後需要 GitHub MCP 功能，可用 `ExecutorOptions.DisableBuiltinMCPs` 旗標控制。

### 實作計畫

| 步驟 | 檔案 | 修改 |
|------|------|------|
| 1 | `cli_executor.go` ExecutorOptions | 新增 `DisableBuiltinMCPs bool` 欄位 |
| 2 | `cli_executor.go` DefaultOptions() | 設定 `DisableBuiltinMCPs: true` |
| 3 | `cli_executor.go` buildArgs() | 當 `DisableBuiltinMCPs` 為 true 時加入 `--disable-builtin-mcps` |
| 4 | 測試 | 驗證 Copilot 不再使用 MCP skill |

---

## ❌ Open-02：`--no-custom-instructions` 管不到 `.claude/` skill（待修復）

### 現象
- `--no-custom-instructions` 只阻擋 AGENTS.md / CLAUDE.md 等指令檔
- 無法阻止 Copilot 讀取 `.claude/commands/` 並載入/執行 skill 任務
- 有 `.claude/` 目錄的專案容易任務跑偏

### 根本原因
- `.claude/commands/` 中的 skill（slash 命令）使用不同的載入機制
- 它們不在 `--no-custom-instructions` 的封鎖清單中
- Copilot 可以主動「發現」這些 skill 並自行決定使用

### 修復方案（多層防禦）

**Layer 1：禁用 MCP（與 Open-01 同步）**
- `--disable-builtin-mcps` 可以阻止 MCP-based 的 skill 執行
- 但不能阻止 Copilot 讀取 `.claude/commands/` 的內容

**Layer 2：Prompt 防禦注入（推薦）**
- 在 `ralphStatusInstruction` 中增加明確指令，告訴 Copilot 不要使用任何 skill
```go
const ralphStatusInstruction = `【系統要求】
1. 不要使用任何 skill 或 slash command（如 /ralph-loop、/package-audit 等）
2. 不要讀取或執行 .claude/commands/ 目錄下的任何檔案
3. 只使用直接的 shell 和 write 工具來完成任務
4. 完成任務後，回應最後必須輸出以下格式：
---RALPH_STATUS---
EXIT_SIGNAL: true
REASON: <完成原因>
---END_RALPH_STATUS---
若任務尚未完成，輸出 EXIT_SIGNAL: false。

【任務】
`
```

**Layer 3：使用 `--excluded-tools` 排除 skill 類工具**
```go
// 排除 skill 類工具（如果 Copilot CLI 支援此語法）
args = append(args, "--excluded-tools", "skill")
```

### 實作計畫

| 步驟 | 檔案 | 修改 |
|------|------|------|
| 1 | `client.go` ralphStatusInstruction | 加入「禁止使用 skill」的 prompt 指令 |
| 2 | `cli_executor.go` buildArgs() | 加入 `--disable-builtin-mcps`（與 Open-01 合併） |
| 3 | 測試 | 在含 `.claude/commands/` 的專案中驗證 Copilot 不再讀取 skill |

---

## ❌ Open-03：`error: unknown option '--no-warnings'` 大量輸出（已緩解，待官方修復）

### 現象
- 每次 shell 工具執行後 Copilot CLI stderr 輸出這行
- 複雜任務可能幾十上百行噪音

### 已實施的緩解措施
1. `filteredWriter` 過濾 stderr 中的噪音行 — `cli_executor.go:656-696`
2. exit code != 0 但有 stdout 時正常解析 — `client.go:253-261`
3. **Commit**: `8a56a22`

### 根治方案
- **等待 Copilot CLI 升級**：官方已修復（Issue #1446），在 v0.0.409 後的版本釋出
- **臨時加速方案**：在 `execute()` 啟動子程序前清除 `NODE_OPTIONS` 環境變數
```go
// 在 cmd.Env 中移除或清空 NODE_OPTIONS，防止洩漏到 git 子程序
env := os.Environ()
for i, e := range env {
    if strings.HasPrefix(e, "NODE_OPTIONS=") {
        env[i] = "NODE_OPTIONS="
        break
    }
}
cmd.Env = append(env, envVars...)
```

### 實作計畫

| 步驟 | 檔案 | 修改 |
|------|------|------|
| 1 | `cli_executor.go` execute() | 在設定 cmd.Env 時清除 `NODE_OPTIONS` |
| 2 | 保留 `filteredWriter` | 作為防禦層繼續保留，以防其他噪音 |
| 3 | 升級 Copilot CLI | 升級後可移除 `NODE_OPTIONS` 清除邏輯 |

---

## 🆕 發現的其他問題

### Issue-A：`ioutil` 已棄用

- `circuit_breaker.go` 和 `exit_detector.go` 使用了 `io/ioutil`
- `ioutil.ReadFile` / `ioutil.WriteFile` 在 Go 1.16+ 已棄用
- 應改用 `os.ReadFile` / `os.WriteFile`
- **影響**：無功能性影響，但編譯器警告，且不符合 Go 1.24.5 最佳實踐

### Issue-B：`ExecuteUntilCompletion` 每次迴圈重複注入 ralphStatusInstruction

- `client.go:177-186` — `ExecuteLoop` 每次被呼叫都會在 prompt 前面注入 `ralphStatusInstruction`
- `client.go:328` — `ExecuteUntilCompletion` 每次迴圈都呼叫 `ExecuteLoop(ctx, initialPrompt)`
- **結果**：隨著迴圈增加，prompt 不會累積（因為每次都是原始 prompt + 指令），這是正確的
- **但**：如果使用 session resume（`--resume`），之前的 prompt 會留在 session 中，新的注入可能重複
- **建議**：目前無需修改，但如果啟用 session resume 功能需要注意

### Issue-C：`ResumeSession` 和 `ContinueLastSession` 未使用 `--yolo`

- `cli_executor.go:321-339` — `ResumeSession()` 和 `ContinueLastSession()` 只加了 `--allow-all-tools`
- 它們沒有走 `buildArgs()` 流程，因此不會加入 `--yolo`、`--no-custom-instructions`、`--disable-builtin-mcps`
- **影響**：如果使用 session resume，會回到舊的權限模式
- **修法**：讓這些方法也走 `buildArgs()` 或至少複製相同的旗標邏輯

---

## 行動清單

### 已完成

| 狀態 | 優先 | 項目 | Commit |
|------|------|------|--------|
| ✅ | P0 | Bug 1: `--yolo` 取代個別 allow 旗標 | `97d1e04` |
| ✅ | P0 | Bug 2: `--no-custom-instructions` 防止任務跑偏 | `97d1e04` |
| ✅ | P1 | Bug 3: `filteredWriter` 過濾 stderr 噪音 | `8a56a22` |

### 待執行

| 優先 | 項目 | 檔案 | 說明 |
|------|------|------|------|
| P0 | Open-01 修復 | `cli_executor.go` | 新增 `DisableBuiltinMCPs` 選項，buildArgs() 加入 `--disable-builtin-mcps` |
| P0 | Open-02 修復 | `client.go` | ralphStatusInstruction 加入「禁止使用 skill」指令 |
| P1 | Open-03 加強 | `cli_executor.go` | 清除 `NODE_OPTIONS` 環境變數 |
| P2 | Issue-C 修復 | `cli_executor.go` | `ResumeSession`/`ContinueLastSession` 改走 `buildArgs()` 流程 |
| P3 | Issue-A 清理 | `circuit_breaker.go`, `exit_detector.go` | `ioutil` → `os.ReadFile`/`os.WriteFile` |
| P3 | 驗證 | 整合測試 | 在含 `.claude/commands/` 的專案中驗證所有修復 |
