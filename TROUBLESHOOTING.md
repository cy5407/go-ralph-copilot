# Ralph Loop 故障排除指南

> 常見問題診斷與解決方案

## 📋 目錄

- [快速診斷](#快速診斷)
- [常見錯誤](#常見錯誤)
- [效能問題](#效能問題)
- [連接問題](#連接問題)
- [配置問題](#配置問題)
- [平台特定問題](#平台特定問題)
- [日誌分析](#日誌分析)

---

## 快速診斷

### 診斷檢查清單

執行以下命令進行快速診斷：

```bash
# 1. 檢查 Ralph Loop 版本
ralph-loop version

# 2. 檢查 Copilot CLI 安裝
copilot --version

# 3. 檢查認證狀態
copilot auth status

# 4. 檢查系統狀態
ralph-loop status

# 5. 執行簡單測試
ralph-loop run -prompt "echo hello" -max-loops 1 -v
```

### 系統資訊收集

當需要回報問題時，請收集以下資訊：

```bash
# Windows (PowerShell)
@"
=== Ralph Loop 診斷資訊 ===
Ralph Loop 版本: $(ralph-loop version)
Copilot CLI 版本: $(copilot --version)
作業系統: $([System.Environment]::OSVersion.VersionString)
Go 版本: $(go version)
"@ | Out-File -FilePath ralph-loop-diagnostic.txt

# macOS/Linux (Bash)
cat > ralph-loop-diagnostic.txt << EOF
=== Ralph Loop 診斷資訊 ===
Ralph Loop 版本: $(ralph-loop version)
Copilot CLI 版本: $(copilot --version)
作業系統: $(uname -a)
Go 版本: $(go version)
EOF
```

---

## 常見錯誤

### ❌ 錯誤 1: "copilot: command not found"

**現象**：
```
[EXECUTION_ERROR] 執行 Copilot CLI 失敗: exec: "copilot": executable file not found
```

**原因**：GitHub Copilot CLI 未安裝或不在 PATH 中

**解決方案**：

1. **安裝 Copilot CLI**：
   ```bash
   # Windows
   winget install GitHub.Copilot
   
   # macOS
   brew install github-copilot-cli
   
   # 或使用 npm
   npm install -g @github/copilot
   ```

2. **驗證安裝**：
   ```bash
   copilot --version
   # 應輸出: Copilot CLI version 0.0.xxx
   ```

3. **檢查 PATH**（若仍無法找到）：
   ```bash
   # Windows
   $env:PATH -split ';' | Select-String copilot
   
   # macOS/Linux
   which copilot
   echo $PATH | grep copilot
   ```

---

### ❌ 錯誤 2: "You have no quota"

**現象**：
```
[QUOTA_ERROR] API 配額已超限: 402 You have no quota for model claude-sonnet-4.5
```

**原因**：GitHub Copilot API 配額耗盡

**解決方案**：

1. **檢查訂閱狀態**：
   - 前往 https://github.com/settings/copilot
   - 確認訂閱是否有效
   - 檢查計費狀態

2. **等待配額重置**：
   - 個人帳戶：通常每小時或每月重置
   - 企業帳戶：聯繫管理員

3. **使用模擬模式測試**（不消耗配額）：
   ```bash
   export COPILOT_MOCK_MODE=true
   ralph-loop run -prompt "測試" -max-loops 3
   ```

4. **切換較輕量的模型**：
   ```bash
   ralph-loop run -prompt "..." --model gpt-4o-mini -max-loops 5
   ```

---

### ❌ 錯誤 3: "authentication failed"

**現象**：
```
[AUTH_ERROR] 認證失敗: please run 'copilot auth' to authenticate
```

**原因**：未認證或認證過期

**解決方案**：

1. **執行認證**：
   ```bash
   copilot auth
   ```

2. **驗證認證狀態**：
   ```bash
   copilot auth status
   ```

3. **重新認證**（若過期）：
   ```bash
   # 登出
   copilot auth logout
   
   # 重新登入
   copilot auth
   ```

4. **檢查認證檔案權限**：
   ```bash
   # macOS/Linux
   ls -la ~/.config/github-copilot/
   chmod 600 ~/.config/github-copilot/hosts.json
   
   # Windows
   icacls %USERPROFILE%\.config\github-copilot\hosts.json
   ```

---

### ❌ 錯誤 4: "circuit breaker opened"

**現象**：
```
[CIRCUIT_OPEN] 熔斷器已開啟，停止執行
💡 建議: 請執行 'ralph-loop reset' 重置熔斷器
```

**原因**：系統偵測到連續失敗或無進展

**解決方案**：

1. **重置熔斷器**：
   ```bash
   ralph-loop reset
   ```

2. **檢查根本原因**：
   ```bash
   # 查看最近的日誌
   tail -100 .ralph-loop/logs/ralph-loop.log
   ```

3. **調整閾值**（如需要）：
   ```toml
   # ralph-loop.toml
   [client]
   circuit_breaker_threshold = 5  # 預設 3，增加容錯
   same_error_threshold = 10      # 預設 5，增加容錯
   ```

4. **改善 prompt 明確度**：
   - 避免模糊的指令
   - 提供明確的完成標準
   - 分解複雜任務

---

### ❌ 錯誤 5: "operation timeout"

**現象**：
```
[TIMEOUT] 操作超時
💡 建議: 請增加超時設定 (--timeout) 或檢查網路連線
```

**原因**：CLI 執行時間超過設定的超時時間

**解決方案**：

1. **增加超時時間**：
   ```bash
   ralph-loop run -prompt "..." -timeout 5m
   ```

2. **設定環境變數**：
   ```bash
   export RALPH_CLI_TIMEOUT="300s"
   ```

3. **檢查網路連線**：
   ```bash
   # 測試 GitHub API 連線
   curl -I https://api.github.com
   
   # 測試 DNS 解析
   nslookup github.com
   ```

4. **檢查防火牆設定**：
   - 確保允許 HTTPS 連線（port 443）
   - 檢查企業代理設定

---

### ❌ 錯誤 6: "invalid configuration"

**現象**：
```
[CONFIG_ERROR] 配置無效: invalid value for cli_timeout
```

**原因**：配置檔案格式錯誤或參數無效

**解決方案**：

1. **驗證 TOML 格式**：
   ```bash
   # 使用線上驗證器
   # https://www.toml-lint.com/
   
   # 或使用 Go 工具
   go run -c ralph-loop.toml
   ```

2. **檢查常見錯誤**：
   ```toml
   # ❌ 錯誤：時間格式
   cli_timeout = 60  # 應為 "60s"
   
   # ✅ 正確
   cli_timeout = "60s"
   
   # ❌ 錯誤：路徑分隔符
   work_dir = "C:\Users\..."  # Windows 需要跳脫
   
   # ✅ 正確
   work_dir = "C:\\Users\\..." # 或使用
   work_dir = 'C:\Users\...'   # 單引號字串
   ```

3. **使用預設配置測試**：
   ```bash
   ralph-loop run -prompt "..." --no-config
   ```

---

## 效能問題

### 🐌 問題 1: 執行速度過慢

**現象**：每個迴圈執行時間 > 2 分鐘

**診斷**：

```bash
# 啟用除錯日誌查看時間分布
RALPH_DEBUG=1 ralph-loop run -prompt "..." -max-loops 3 2>&1 | grep "took"
```

**可能原因與解決方案**：

1. **網路延遲**：
   ```bash
   # 測試延遲
   ping api.github.com
   
   # 使用代理（如適用）
   export HTTP_PROXY=http://proxy.example.com:8080
   export HTTPS_PROXY=http://proxy.example.com:8080
   ```

2. **模型選擇**：
   ```bash
   # 切換較快的模型
   ralph-loop run --model gpt-4o-mini ...
   ```

3. **重試次數過高**：
   ```toml
   [client]
   cli_max_retries = 1  # 降低重試次數
   ```

---

### 💾 問題 2: 記憶體使用過高

**現象**：程式使用 > 1GB 記憶體

**診斷**：

```bash
# Windows
Get-Process ralph-loop | Select-Object WorkingSet,VirtualMemorySize

# macOS/Linux
ps aux | grep ralph-loop
```

**解決方案**：

1. **限制迴圈數**：
   ```bash
   ralph-loop run -max-loops 5  # 降低最大迴圈數
   ```

2. **清理舊記錄**：
   ```bash
   # 清理舊的執行記錄
   find .ralph-loop/saves -mtime +7 -delete
   ```

3. **調整 Go 記憶體參數**：
   ```bash
   export GOGC=50  # 更積極的 GC
   ralph-loop run ...
   ```

---

## 連接問題

### 🌐 問題 1: 無法連接到 GitHub API

**現象**：
```
[NETWORK_ERROR] 網路連線失敗: dial tcp: lookup api.github.com: no such host
```

**解決方案**：

1. **檢查 DNS**：
   ```bash
   nslookup api.github.com
   # 應解析到 GitHub 的 IP 地址
   ```

2. **檢查 /etc/hosts**（macOS/Linux）：
   ```bash
   cat /etc/hosts | grep github
   # 移除任何 GitHub 相關的錯誤條目
   ```

3. **檢查防火牆**：
   ```bash
   # Windows
   netsh advfirewall show allprofiles
   
   # macOS
   /usr/libexec/ApplicationFirewall/socketfilterfw --getglobalstate
   
   # Linux
   sudo iptables -L
   ```

---

### 🔒 問題 2: 企業代理問題

**現象**：在企業網路環境中無法連線

**解決方案**：

1. **設定代理環境變數**：
   ```bash
   # Windows
   $env:HTTP_PROXY = "http://proxy.company.com:8080"
   $env:HTTPS_PROXY = "http://proxy.company.com:8080"
   $env:NO_PROXY = "localhost,127.0.0.1"
   
   # macOS/Linux
   export HTTP_PROXY="http://proxy.company.com:8080"
   export HTTPS_PROXY="http://proxy.company.com:8080"
   export NO_PROXY="localhost,127.0.0.1"
   ```

2. **配置 Git 代理**：
   ```bash
   git config --global http.proxy http://proxy.company.com:8080
   git config --global https.proxy http://proxy.company.com:8080
   ```

3. **信任企業憑證**（若使用 HTTPS 攔截）：
   ```bash
   # 將企業根憑證加入系統信任
   # Windows: certmgr.msc
   # macOS: Keychain Access
   # Linux: /etc/ssl/certs/
   ```

---

## 配置問題

### ⚙️ 問題 1: 配置不生效

**現象**：修改配置後行為未改變

**檢查清單**：

1. **確認配置檔案位置**：
   ```bash
   ralph-loop run --config ralph-loop.toml -v
   # 查看日誌確認載入的配置檔案
   ```

2. **環境變數優先級**：
   - 環境變數 > 命令列參數 > 配置檔案 > 預設值
   ```bash
   # 取消環境變數測試
   unset RALPH_CLI_TIMEOUT
   ```

3. **配置語法正確性**：
   ```bash
   # 驗證 TOML 語法
   cat ralph-loop.toml
   ```

---

## 平台特定問題

### 🪟 Windows 問題

#### 問題：路徑分隔符錯誤

```powershell
# ❌ 錯誤
ralph-loop run -work-dir C:/Users/...

# ✅ 正確
ralph-loop run -work-dir C:\Users\...
```

#### 問題：PowerShell 執行政策

```powershell
# 如果無法執行腳本
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

---

### 🍎 macOS 問題

#### 問題："ralph-loop" cannot be opened because the developer cannot be verified

```bash
# 移除隔離屬性
xattr -d com.apple.quarantine ralph-loop

# 或在系統偏好設定 > 安全性與隱私中允許
```

#### 問題：Gatekeeper 阻擋

```bash
# 允許未簽名的應用程式
sudo spctl --master-disable
# 執行後再啟用
sudo spctl --master-enable
```

---

### 🐧 Linux 問題

#### 問題：權限不足

```bash
# 賦予執行權限
chmod +x ralph-loop

# 檢查 SELinux（CentOS/RHEL）
sestatus
sudo chcon -t bin_t ralph-loop
```

#### 問題：缺少依賴

```bash
# Debian/Ubuntu
sudo apt-get install -y ca-certificates

# CentOS/RHEL
sudo yum install -y ca-certificates
```

---

## 日誌分析

### 啟用詳細日誌

```bash
# 方式 1: 環境變數
export RALPH_DEBUG=1
ralph-loop run ...

# 方式 2: 命令列參數
ralph-loop run -v ...

# 方式 3: 配置文件
# ralph-loop.toml
[logging]
level = "debug"
```

### 日誌位置

```
.ralph-loop/
├── logs/
│   ├── ralph-loop.log          # 主日誌
│   └── cli-output-*.log        # CLI 原始輸出
└── saves/
    ├── context_*.json          # 執行上下文
    └── loop_*.json             # 迴圈記錄
```

### 常見日誌模式

#### 1. 成功執行

```
INFO  開始迴圈 1/10
DEBUG CLI 命令: copilot -p "..."
DEBUG CLI 輸出: [...成功輸出...]
INFO  迴圈 1 完成，耗時 45.2s
```

#### 2. 超時

```
INFO  開始迴圈 3/10
DEBUG CLI 命令: copilot -p "..."
WARN  ⚠️  執行超時（60s）
ERROR [TIMEOUT] 操作超時
```

#### 3. 熔斷器觸發

```
WARN  無進展迴圈計數: 3/3
ERROR [CIRCUIT_OPEN] 熔斷器已開啟
INFO  結束執行：熔斷器保護
```

### 分析工具

```bash
# 統計錯誤類型
grep ERROR .ralph-loop/logs/ralph-loop.log | cut -d']' -f1 | sort | uniq -c

# 查看最慢的迴圈
grep "耗時" .ralph-loop/logs/ralph-loop.log | sort -t'耗' -k2 -n

# 查看完成率
grep -c "迴圈.*完成" .ralph-loop/logs/ralph-loop.log
```

---

## 進階診斷

### 啟用 pprof 效能分析

```bash
# 編譯時啟用 pprof
go build -tags=pprof -o ralph-loop ./cmd/ralph-loop

# 執行並收集效能資料
ralph-loop run ... &
RALPH_PID=$!

# 等待一段時間後收集
go tool pprof http://localhost:6060/debug/pprof/profile
```

### 追蹤 Copilot CLI 呼叫

```bash
# macOS/Linux
strace -e trace=execve -f ralph-loop run ... 2>&1 | grep copilot

# Windows
# 使用 Process Monitor (procmon.exe)
```

---

## 取得協助

如果上述方法都無法解決問題：

1. **收集診斷資訊**：
   ```bash
   ralph-loop status > diagnostic.txt
   cat .ralph-loop/logs/ralph-loop.log >> diagnostic.txt
   ```

2. **建立 GitHub Issue**：
   - 前往: https://github.com/yourusername/ralph-loop/issues/new
   - 提供：
     - 診斷資訊檔案
     - 重現步驟
     - 預期行為 vs 實際行為
     - 環境資訊（OS、版本等）

3. **社群支援**：
   - GitHub Discussions: https://github.com/yourusername/ralph-loop/discussions
   - Discord/Slack（如有）

---

## 參考資源

- [部署指南](./DEPLOYMENT_GUIDE.md)
- [使用指南](./USAGE_GUIDE.md)
- [架構文檔](./ARCHITECTURE.md)
- [GitHub Copilot CLI 文檔](https://docs.github.com/copilot/using-github-copilot/using-github-copilot-in-the-command-line)

---

最後更新：2026-02-12
