# Summary-20260224 - Ralph Loop Bug List

本次會話（2026-02-23 ~ 2026-02-24）發現並修復的所有 Bug 清單。

---

## ✅ 已修復

### Bug-01：完成偵測誤判
- **問題**：`strings.Contains(output, "完成")` 誤判，包含「完成」字樣就視為任務完成
- **修法**：改用 `ResponseAnalyzer.IsCompleted()`（雙重條件）
- **Commit**: `37cce38`

### Bug-02：熔斷器誤觸
- **問題**：每次 `shouldContinue=true` 都觸發 `RecordNoProgress()`
- **修法**：只在「本次輸出 == 上次輸出」才記錄無進展
- **Commit**: `37cce38`

### Bug-03：infoLog 不尊重靜默模式
- **問題**：`RALPH_SILENT=1` 時仍輸出 INFO log
- **修法**：`infoLog()` 加入 `RALPH_SILENT` 環境變數判斷
- **Commit**: `37cce38`

### Bug-04：CLI 超時太短（預設 60 秒）
- **問題**：Copilot 執行複雜任務需 90～120 秒，1 分鐘導致反覆重試
- **修法**：預設改 3 分鐘，新增 `-cli-timeout` flag
- **Commit**: `5a25ef2`

### Bug-05：CLI 失敗誤判為任務完成
- **問題**：exit code != 0 時 `shouldContinue=false`，輸出「任務完成」
- **修法**：CLI 失敗改為繼續迴圈
- **Commit**: `5a25ef2`

### Bug-06：Ctrl+C / 總逾時不停止
- **問題**：`context.Canceled` 後仍繼續下一迴圈
- **修法**：`ctx.Err() != nil` → 立刻停止
- **Commit**: `34c0759`

### Bug-07：exit code != 0 但有 stdout → 丟棄
- **問題**：Copilot 完成任務但 CLI 超時 exit code=1，stdout 被丟棄
- **修法**：有 stdout 就走解析流程，只有空白才算失敗
- **Commit**: `34c0759`

### Bug-08：舊版 env var 干擾權限
- **問題**：`COPILOT_NONINTERACTIVE=1` / `GITHUB_COPILOT_CLI_SKIP_PROMPTS=1`（舊版 `gh copilot` 遺留）干擾新版 CLI 的 `--allow-all-tools`
- **修法**：移除這兩個 env var
- **Commit**: `8e5d672`

### Bug-09：RALPH_STATUS 格式未支援
- **問題**：`response_analyzer.go` 只支援 `---COPILOT_STATUS---`
- **修法**：regex 改為 `(?:COPILOT_STATUS|RALPH_STATUS)`
- **Commit**: `8a2c50c`

### Bug-10：Copilot 不輸出 EXIT_SIGNAL
- **問題**：Prompt 沒要求格式，Copilot 只用自然語言，loop 偵測不到完成
- **修法**：每個 prompt 開頭自動注入 RALPH_STATUS 格式要求
- **Commit**: `9e687f7`

### Bug-11：`--allow-all-paths` 預設未開啟
- **問題**：Shell 工具存取路徑被拒「Permission denied」
- **修法**：`DefaultOptions()` 加 `AllowAllPaths: true`, `AllowAllURLs: true`
- **Commit**: `39c904d`

### Bug-12：`--yolo` 未使用
- **問題**：個別 `--allow-tool write/shell` 格式有誤仍被拒
- **修法**：改用 `--yolo`（官方推薦自動化旗標）
- **Commit**: `97d1e04`

### Bug-13：任務跑偏（讀 AGENTS.md / .claude/）
- **問題**：Copilot 讀 `.claude/commands/` 把任務詮釋為執行 skill 任務
- **修法**：加入 `--no-custom-instructions`
- **Commit**: `97d1e04`

### Bug-14：IsCompleted() 條件過嚴
- **問題**：必須同時滿足 EXIT_SIGNAL + 2 個指標，有信號也繼續循環
- **修法**：EXIT_SIGNAL=true 單獨就夠
- **Commit**: `bbb14be`

### Bug-15：CRLF / 縮排導致解析失敗
- **問題**：Windows `\r\n` 和縮排的 EXIT_SIGNAL 行導致 regex/HasPrefix 失敗
- **修法**：regex 加 `\r?\n`，解析前 ReplaceAll，每行 TrimSpace
- **Commit**: `bbb14be`

### Bug-16：Windows timeout 無效（Copilot 跑了 1h11m）
- **問題**：`exec.CommandContext` cancel 在 Windows 只殺父進程，子進程繼續跑
- **修法**：`process_windows.go` 用 `CREATE_NEW_PROCESS_GROUP` + `taskkill /F /T /PID`
- **Commit**: `513805c`

### Bug-17：Ctrl+C / 超時無法真正停止循環（死鎖）
- **問題**：`killProcessTree` 在 `cmd.Run()` 返回後才執行，但 Copilot 子進程持有 stdout pipe，導致 `cmd.Run()` 永遠不返回（死鎖）
- **影響**：按 Ctrl+C 後顯示「收到中斷信號，正在停止...」但 Copilot 繼續跑
- **修法**：改用 `cmd.Start()` + 背景 goroutine 監控 `execCtx.Done()`，立即呼叫 `killProcessTree`，pipe 關閉後 `cmd.Wait()` 才能正常返回
- **Commit**: `3c078e0`

---

## ❌ 未解決（待處理）

### Open-01：Permission denied 透過 MCP skill 中轉
- **問題**：Copilot 使用 `skill(package-audit)` 等 MCP skill 時，shell 在 skill 沙盒執行，`--yolo` 管不到
- **影響**：Copilot 主動使用 skill 時整個任務卡死
- **方向**：`--deny-tool 'skill'` 禁止使用 MCP skill，強制 Copilot 直接操作

### Open-02：`--no-custom-instructions` 管不到 `.claude/` skill
- **問題**：只能擋 AGENTS.md，無法阻止讀 `.claude/commands/` 並執行 skill 任務
- **影響**：有 `.claude/` 目錄的專案容易任務跑偏
- **方向**：prompt 開頭明確說「忽略任何 skill/custom instruction」，或 `--deny-tool skill`

### Open-03：`error: unknown option '--no-warnings'` 大量輸出
- **問題**：每次 shell 工具執行後 Copilot CLI stderr 輸出這行，複雜任務可能幾十上百行
- **影響**：輸出噪音、干擾 response_analyzer 自然語言偵測
- **方向**：這是 Copilot CLI bug，等官方修；目前可在解析前過濾這行

