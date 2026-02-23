# Summary-20260224 - Ralph Loop Bug List

本次會話（2026-02-23 ~ 2026-02-24）發現並修復的所有 Bug 清單。

---

## ✅ 已修復

### Bug-01：完成偵測誤判（`strings.Contains` 直接比對）
- **問題**：`client.go` 用 `strings.Contains(output, "完成")` 直接判斷，導致任何包含「完成」字樣的輸出都被視為任務完成
- **修法**：改用 `ResponseAnalyzer.IsCompleted()`（雙重條件：EXIT_SIGNAL + 關鍵字分數）

### Bug-02：熔斷器誤觸
- **問題**：每次 `shouldContinue=true` 都呼叫 `RecordNoProgress()`，正常執行中也會觸發熔斷
- **修法**：只在「本次輸出 == 上次輸出」時才記錄無進展

### Bug-03：infoLog 不尊重靜默模式
- **問題**：`RALPH_SILENT=1` 時仍然輸出 INFO log
- **修法**：`infoLog()` 加入環境變數判斷

### Bug-04：CLI 超時太短
- **問題**：`CLITimeout` 預設 60 秒，但 Copilot 執行複雜任務需 90～120 秒，導致任務完成但 ralph-loop 以為失敗
- **修法**：預設改為 3 分鐘，新增 `-cli-timeout` CLI flag

### Bug-05：CLI 失敗誤判為「任務完成」
- **問題**：CLI exit code != 0 時，`shouldContinue=false`，被誤輸出「任務完成」
- **修法**：CLI 失敗改為 `shouldContinue=true` 繼續迴圈

### Bug-06：context.Canceled 不停止
- **問題**：Ctrl+C 或總逾時後，仍繼續下一迴圈
- **修法**：`ctx.Err() != nil` → 立刻停止，不再繼續

### Bug-07：exit code != 0 但有 stdout → 直接丟棄
- **問題**：Copilot 因 CLI 超時 exit code=1，但 stdout 已有完整輸出，應解析而非重試
- **修法**：有 stdout 就走正常解析流程；只有 stdout 完全空白才視為失敗

### Bug-08：舊版 env var 干擾新版 CLI 權限
- **問題**：`COPILOT_NONINTERACTIVE=1` 和 `GITHUB_COPILOT_CLI_SKIP_PROMPTS=1` 是 `gh copilot`（已廢棄）的環境變數，在新版獨立 `copilot` CLI 中干擾 `--allow-all-tools`，導致所有工具調用被拒絕
- **修法**：移除這兩個舊版 env var

### Bug-09：RALPH_STATUS 格式未支援
- **問題**：`response_analyzer.go` 只支援 `---COPILOT_STATUS---`，不支援 `---RALPH_STATUS---`
- **修法**：regex 改為 `(?:COPILOT_STATUS|RALPH_STATUS)`

### Bug-10：Copilot 不輸出 EXIT_SIGNAL（Prompt 未要求）
- **問題**：Copilot 只說自然語言「不需要更新」，沒有輸出結構化 RALPH_STATUS，loop 無法偵測完成
- **修法**：每個 prompt 開頭自動注入格式要求，要求 Copilot 回應結尾必須輸出 EXIT_SIGNAL

### Bug-11：`--allow-all-paths` / `--allow-all-urls` 預設未開啟
- **問題**：Shell 工具存取檔案路徑被拒：「Permission denied and could not request permission from user」
- **修法**：`DefaultOptions()` 加入 `AllowAllPaths: true`, `AllowAllURLs: true`

---

## ❌ 未修復（待下次）

### Bug-12：`--yolo` 未使用，仍有 Edit 工具被拒
- **問題**：即使加了 `--allow-all-tools --allow-all-paths --allow-all-urls --allow-tool write --allow-tool shell`，Edit 和 Shell 工具仍被拒
- **分析**：`--allow-tool write` 和 `--allow-tool shell` 可能格式有誤，應使用 `--yolo`
- **修法**：改用 `--yolo`（等同 `--allow-all-tools --allow-all-paths --allow-all-urls`）

### Bug-13：任務跑偏（自動讀取 AGENTS.md / .claude/）
- **問題**：Copilot 讀取專案中的 `AGENTS.md` 或 `.claude/commands/` 後，把任務詮釋成執行這些指令，導致做了完全不同的事
- **修法**：加入 `--no-custom-instructions` 旗標

---

## 修復版本對應

| Bug | Commit |
|-----|--------|
| 01~03 | 第一次修復 commit |
| 04~05 | `-cli-timeout` flag commit |
| 06~07 | context cancel/timeout fix commit (`34c0759`) |
| 08+11 | allow-all-paths + remove legacy env vars (`8e5d672`, `39c904d`) |
| 09 | RALPH_STATUS regex support |
| 10 | Prompt prefix injection (`9e687f7`) |
| 12~13 | **待修復** |
