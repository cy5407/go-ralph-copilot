# 源碼安全掃描修復清單

> 工具：gosec  
> 掃描時間：2026-02-23  
> 總問題數：19 個（HIGH: 1、MEDIUM: 9、LOW: 9）

---

## 🔴 HIGH 優先修復

### G404 - 弱亂數產生器
**檔案**：`internal/ghcopilot/retry_strategy.go:140`  
**問題**：使用 `math/rand.Float64()` 產生 jitter 延遲，屬於可預測的弱亂數  
**現況**：
```go
jitter := time.Duration(rand.Float64() * jitterRange)
```
**建議修復**：改用 `crypto/rand` 產生隨機值，或接受風險加 `#nosec G404` 註解（retry jitter 不涉及安全敏感用途）

---

## 🟡 MEDIUM 建議修復

### G301 - 目錄權限過寬
**檔案**：`internal/ghcopilot/persistence.go:28`  
**問題**：`os.MkdirAll(storageDir, 0755)` 允許其他使用者讀取目錄  
**現況**：
```go
os.MkdirAll(storageDir, 0755)
```
**建議修復**：
```go
os.MkdirAll(storageDir, 0750)
```

---

### G306 - 檔案寫入權限過寬（3 處）
**檔案**：
- `internal/ghcopilot/persistence.go:143`
- `internal/ghcopilot/exit_detector.go:256`
- `internal/ghcopilot/circuit_breaker.go:188`

**問題**：`0644` 允許其他使用者讀取含執行記錄的檔案  
**現況**：
```go
os.WriteFile(outputPath, []byte(jsonStr), 0644)
ioutil.WriteFile(ed.signalFile, jsonData, 0644)
ioutil.WriteFile(cb.stateFile, jsonData, 0644)
```
**建議修復**：全部改為 `0600`（僅擁有者可讀寫）

---

### G304 - 路徑穿越風險（5 處）
**檔案**：`internal/ghcopilot/persistence.go:47, 61, 82, 261, 277`  
**問題**：`os.Open(filename)` / `os.Create(filename)` 使用外部傳入的路徑，理論上可被操控存取任意檔案  
**現況**：
```go
file, err := os.Open(filename)
file, err := os.Create(filename)
```
**建議修復**：驗證 filename 在允許的目錄範圍內（路徑前綴檢查），或使用 Go 1.24 的 `os.Root` 限制存取範圍：
```go
// 驗證路徑在 storageDir 範圍內
absPath, _ := filepath.Abs(filename)
if !strings.HasPrefix(absPath, pm.storageDir) {
    return fmt.Errorf("路徑超出允許範圍: %s", filename)
}
```

---

### G204 - 執行外部程序含變數參數
**檔案**：`internal/ghcopilot/cli_executor.go:396`  
**問題**：`exec.CommandContext(execCtx, "copilot", args...)` 中 args 含有使用者輸入的 prompt  
**現況**：
```go
cmd := exec.CommandContext(execCtx, "copilot", args...)
```
**建議修復**：設計上不可避免，加 `#nosec G204` 並在上方說明 prompt 已透過 `buildArgs()` 組裝，不直接執行 shell 指令（無 shell injection 風險）

---

## 🟢 LOW 可選修復

### G104 - 未處理的 error（9 處）

| 檔案 | 行號 | 未處理的呼叫 |
|------|------|-------------|
| `cmd/ralph-loop/main.go` | 46 | `runCmd.Parse(os.Args[2:])` |
| `cmd/ralph-loop/main.go` | 55 | `statusCmd.Parse(os.Args[2:])` |
| `cmd/ralph-loop/main.go` | 59 | `resetCmd.Parse(os.Args[2:])` |
| `cmd/ralph-loop/main.go` | 63 | `watchCmd.Parse(os.Args[2:])` |
| `cmd/ralph-loop/main.go` | 130 | `os.Setenv("RALPH_SILENT", "1")` |
| `internal/ghcopilot/client.go` | 240 | `parser.Parse()` |
| `internal/ghcopilot/client.go` | 595-597 | `contextManager.UpdateCurrentLoop(...)` |
| `internal/ghcopilot/client.go` | 598 | `contextManager.FinishLoop()` |

**建議修復**（main.go 的 cmd.Parse）：
```go
if err := runCmd.Parse(os.Args[2:]); err != nil {
    fmt.Printf("參數解析失敗: %v\n", err)
    os.Exit(1)
}
```
> 注意：`flag.FlagSet` 使用 `ExitOnError` 時 Parse 不會真正回傳 error，可加 `#nosec G104` 或改用 `_ =` 明確忽略

---

## 執行方式

```bash
# 重新掃描
gosec ./...

# 只看 HIGH/MEDIUM
gosec -severity medium ./...

# 輸出成 JSON 報告
gosec -fmt json -out gosec-report.json ./...
```
