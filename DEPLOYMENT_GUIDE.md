# Ralph Loop 部署指南

> AI 驅動的自動程式碼迭代系統 - 完整部署文檔

## 📋 目錄

- [系統需求](#系統需求)
- [安裝方式](#安裝方式)
- [配置設定](#配置設定)
- [驗證安裝](#驗證安裝)
- [Docker 部署](#docker-部署)
- [生產環境建議](#生產環境建議)

---

## 系統需求

### 最低需求

| 項目 | 需求 |
|------|------|
| **作業系統** | Windows 10+, macOS 11+, Linux (kernel 4.x+) |
| **處理器** | 2 核心 CPU |
| **記憶體** | 4 GB RAM |
| **磁碟空間** | 500 MB 可用空間 |
| **網路** | 穩定的網際網路連線 |

### 建議配置

| 項目 | 建議 |
|------|------|
| **處理器** | 4+ 核心 CPU |
| **記憶體** | 8+ GB RAM |
| **磁碟空間** | 2+ GB 可用空間（含日誌） |
| **網路** | 高速穩定連線 |

### 軟體依賴

#### 必須安裝

1. **Go 1.21+**（如需從源碼建置）
   ```bash
   # 驗證安裝
   go version  # 應顯示 go version go1.21 或更高
   ```

2. **GitHub Copilot CLI** (獨立版本)
   ```bash
   # Windows
   winget install GitHub.Copilot
   
   # macOS
   brew install github-copilot-cli
   
   # 或使用 npm (跨平台)
   npm install -g @github/copilot
   
   # 驗證安裝（需要 ≥ 0.0.389）
   copilot --version
   ```

3. **有效的 GitHub Copilot 訂閱**
   - 個人訂閱: $10/月
   - 企業訂閱: 透過組織管理員
   - 驗證: https://github.com/settings/copilot

#### 認證設定

```bash
# 執行 Copilot CLI 認證
copilot auth

# 驗證認證狀態
copilot --version
```

---

## 安裝方式

### 方式 1: 下載預編譯執行檔（推薦）

#### Windows

```powershell
# 下載最新版本
$version = "v0.2.0"  # 替換為最新版本號
$url = "https://github.com/yourusername/ralph-loop/releases/download/$version/ralph-loop-windows-amd64.exe"
Invoke-WebRequest -Uri $url -OutFile ralph-loop.exe

# 驗證檔案雜湊值（選擇性）
$checksumUrl = "https://github.com/yourusername/ralph-loop/releases/download/$version/checksums.txt"
Invoke-WebRequest -Uri $checksumUrl -OutFile checksums.txt
Get-FileHash ralph-loop.exe -Algorithm SHA256

# 移動到系統路徑
Move-Item ralph-loop.exe C:\Windows\System32\

# 驗證安裝
ralph-loop version
```

#### macOS

```bash
# 下載最新版本
VERSION="v0.2.0"  # 替換為最新版本號
ARCH="darwin-arm64"  # Apple Silicon 使用 arm64，Intel 使用 amd64

curl -L "https://github.com/yourusername/ralph-loop/releases/download/$VERSION/ralph-loop-$ARCH" -o ralph-loop

# 驗證檔案雜湊值
curl -L "https://github.com/yourusername/ralph-loop/releases/download/$VERSION/checksums.txt" -o checksums.txt
shasum -a 256 -c checksums.txt --ignore-missing

# 賦予執行權限
chmod +x ralph-loop

# 移動到系統路徑
sudo mv ralph-loop /usr/local/bin/

# 驗證安裝
ralph-loop version
```

#### Linux

```bash
# 下載最新版本
VERSION="v0.2.0"  # 替換為最新版本號
ARCH="linux-amd64"  # 或 linux-arm64

wget "https://github.com/yourusername/ralph-loop/releases/download/$VERSION/ralph-loop-$ARCH" -O ralph-loop

# 驗證檔案雜湊值
wget "https://github.com/yourusername/ralph-loop/releases/download/$VERSION/checksums.txt"
sha256sum -c checksums.txt --ignore-missing

# 賦予執行權限
chmod +x ralph-loop

# 移動到系統路徑
sudo mv ralph-loop /usr/local/bin/

# 驗證安裝
ralph-loop version
```

---

### 方式 2: 從源碼建置

```bash
# 克隆儲存庫
git clone https://github.com/yourusername/ralph-loop.git
cd ralph-loop

# 下載依賴
go mod download

# 建置執行檔
go build -o ralph-loop ./cmd/ralph-loop

# 驗證建置
./ralph-loop version

# 安裝到系統路徑（選擇性）
go install ./cmd/ralph-loop
```

---

### 方式 3: 使用 Go Install

```bash
# 直接安裝最新版本
go install github.com/yourusername/ralph-loop/cmd/ralph-loop@latest

# 驗證安裝
ralph-loop version
```

---

## 配置設定

### 基本配置

創建配置文件 `ralph-loop.toml`（選擇性）：

```toml
[client]
# Copilot CLI 超時設定
cli_timeout = "60s"

# 最大重試次數
cli_max_retries = 3

# 熔斷器閾值
circuit_breaker_threshold = 3
same_error_threshold = 5

# AI 模型選擇
model = "claude-sonnet-4.5"

# 工作目錄
work_dir = "."

# 儲存目錄
save_dir = ".ralph-loop/saves"

[executor]
# 啟用 SDK 執行器
enable_sdk = true

# 優先使用 SDK
prefer_sdk = true

[logging]
# 日誌等級 (debug, info, warn, error)
level = "info"

# 日誌輸出格式 (text, json)
format = "text"

# 日誌檔案路徑
file = ".ralph-loop/logs/ralph-loop.log"
```

### 環境變數配置

```bash
# Windows (PowerShell)
$env:RALPH_CLI_TIMEOUT = "120s"
$env:RALPH_DEBUG = "1"
$env:COPILOT_MOCK_MODE = "false"

# macOS/Linux (Bash)
export RALPH_CLI_TIMEOUT="120s"
export RALPH_DEBUG="1"
export COPILOT_MOCK_MODE="false"
```

支援的環境變數：

| 環境變數 | 說明 | 預設值 |
|---------|------|--------|
| `RALPH_CLI_TIMEOUT` | CLI 執行超時 | `60s` |
| `RALPH_MAX_LOOPS` | 最大迴圈數 | `10` |
| `RALPH_DEBUG` | 啟用除錯日誌 | `0` |
| `COPILOT_MOCK_MODE` | 模擬模式 | `false` |
| `RALPH_WORK_DIR` | 工作目錄 | `.` |
| `RALPH_SAVE_DIR` | 儲存目錄 | `.ralph-loop/saves` |

---

## 驗證安裝

### 基本驗證

```bash
# 檢查版本
ralph-loop version

# 檢查 Copilot CLI
copilot --version

# 執行簡單測試
ralph-loop run -prompt "列出當前目錄檔案" -max-loops 1
```

### 健康檢查

```bash
# 執行內建健康檢查
ralph-loop status

# 預期輸出：
# ✅ Ralph Loop 運行正常
# ✅ GitHub Copilot CLI 已安裝 (版本 0.0.xxx)
# ✅ 認證狀態: 已認證
# ℹ️  配置檔案: ralph-loop.toml
```

---

## Docker 部署

### 使用預建映像

```bash
# 拉取最新映像
docker pull yourusername/ralph-loop:latest

# 執行容器
docker run -it --rm \
  -v $(pwd):/workspace \
  -e RALPH_DEBUG=1 \
  yourusername/ralph-loop:latest \
  run -prompt "測試任務" -max-loops 5
```

### 自行建置映像

創建 `Dockerfile`：

```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ralph-loop ./cmd/ralph-loop

FROM alpine:latest
RUN apk --no-cache add ca-certificates nodejs npm
RUN npm install -g @github/copilot

WORKDIR /workspace
COPY --from=builder /app/ralph-loop /usr/local/bin/

ENTRYPOINT ["ralph-loop"]
```

建置與執行：

```bash
# 建置映像
docker build -t ralph-loop:local .

# 執行容器
docker run -it --rm \
  -v $(pwd):/workspace \
  ralph-loop:local run -prompt "測試" -max-loops 3
```

### Docker Compose

創建 `docker-compose.yml`：

```yaml
version: '3.8'

services:
  ralph-loop:
    image: yourusername/ralph-loop:latest
    volumes:
      - ./:/workspace
    environment:
      - RALPH_DEBUG=1
      - RALPH_CLI_TIMEOUT=120s
    command: run -prompt "測試任務" -max-loops 5
```

執行：

```bash
docker-compose up
```

---

## 生產環境建議

### 1. 資源配置

```toml
[client]
cli_timeout = "120s"          # 增加超時
cli_max_retries = 5           # 增加重試次數
circuit_breaker_threshold = 5  # 放寬熔斷閾值

[logging]
level = "info"                # 生產環境使用 info
format = "json"               # 使用 JSON 格式便於分析
file = "/var/log/ralph-loop/app.log"
```

### 2. 監控與日誌

```bash
# 設定日誌輪轉（Linux）
cat > /etc/logrotate.d/ralph-loop << 'EOF'
/var/log/ralph-loop/*.log {
    daily
    rotate 30
    compress
    delaycompress
    notifempty
    create 0644 root root
}
EOF

# 執行日誌輪轉
logrotate /etc/logrotate.d/ralph-loop
```

### 3. 系統服務設定（Systemd）

創建 `/etc/systemd/system/ralph-loop.service`：

```ini
[Unit]
Description=Ralph Loop AI Agent
After=network.target

[Service]
Type=simple
User=ralph-loop
Group=ralph-loop
WorkingDirectory=/opt/ralph-loop
ExecStart=/usr/local/bin/ralph-loop run --config /etc/ralph-loop/config.toml
Restart=on-failure
RestartSec=5s
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

啟用服務：

```bash
sudo systemctl daemon-reload
sudo systemctl enable ralph-loop
sudo systemctl start ralph-loop
sudo systemctl status ralph-loop
```

### 4. 安全性設定

```bash
# 建立專用使用者
sudo useradd -r -s /bin/false ralph-loop

# 設定檔案權限
sudo chown -R ralph-loop:ralph-loop /opt/ralph-loop
sudo chmod 700 /opt/ralph-loop

# 限制 API 金鑰存取
sudo chmod 600 ~/.copilot/credentials
```

### 5. 備份策略

```bash
# 自動備份腳本
cat > /opt/ralph-loop/backup.sh << 'EOF'
#!/bin/bash
BACKUP_DIR="/backup/ralph-loop/$(date +%Y%m%d)"
mkdir -p "$BACKUP_DIR"

# 備份配置
cp /etc/ralph-loop/config.toml "$BACKUP_DIR/"

# 備份執行記錄
cp -r /opt/ralph-loop/.ralph-loop/saves "$BACKUP_DIR/"

# 壓縮備份
tar -czf "$BACKUP_DIR.tar.gz" "$BACKUP_DIR"
rm -rf "$BACKUP_DIR"
EOF

chmod +x /opt/ralph-loop/backup.sh

# 設定 cron 每日備份
echo "0 2 * * * /opt/ralph-loop/backup.sh" | sudo crontab -
```

### 6. 效能調校

```bash
# 增加檔案描述符限制
ulimit -n 65536

# 設定 Go 執行時參數
export GOMAXPROCS=4
export GOGC=200
```

---

## 故障排除

遇到問題？請參閱 [TROUBLESHOOTING.md](./TROUBLESHOOTING.md)

---

## 升級指南

### 從舊版本升級

```bash
# 1. 備份現有配置
cp ralph-loop.toml ralph-loop.toml.backup
cp -r .ralph-loop/saves .ralph-loop/saves.backup

# 2. 下載新版本
VERSION="v0.2.0"
# ... 依照上述安裝步驟 ...

# 3. 驗證升級
ralph-loop version

# 4. 測試執行
ralph-loop run -prompt "測試升級" -max-loops 1
```

### 版本相容性

| Ralph Loop 版本 | Go 版本需求 | Copilot CLI 版本 |
|----------------|------------|-----------------|
| v0.1.x | Go 1.21+ | ≥ 0.0.389 |
| v0.2.x | Go 1.21+ | ≥ 0.0.400 |

---

## 支援

- **文檔**: [README.md](./README.md)
- **使用指南**: [USAGE_GUIDE.md](./USAGE_GUIDE.md)
- **故障排除**: [TROUBLESHOOTING.md](./TROUBLESHOOTING.md)
- **問題回報**: [GitHub Issues](https://github.com/yourusername/ralph-loop/issues)
- **討論區**: [GitHub Discussions](https://github.com/yourusername/ralph-loop/discussions)

---

## 授權

MIT License - 詳見 [LICENSE](./LICENSE)
