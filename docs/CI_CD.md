# CI/CD 指南

Ralph Loop 使用 GitHub Actions 實現完整的 CI/CD 流程，包含自動測試、程式碼品質檢查、安全掃描和自動發布。

## 工作流程概覽

### 1. 測試流程 (`.github/workflows/test.yml`)

**觸發條件**：
- 推送到 `main` 或 `develop` 分支
- 對 `main` 或 `develop` 分支的 Pull Request

**執行內容**：
- ✅ **跨平台測試**：Linux、macOS、Windows
- ✅ **多版本測試**：Go 1.21、1.22、1.23
- ✅ **競態檢測**：使用 `-race` 標誌
- ✅ **測試覆蓋率**：上傳到 Codecov
- ✅ **程式碼檢查**：golangci-lint、go vet、go fmt
- ✅ **建置驗證**：確保所有平台可正常編譯

**檢視結果**：
```bash
# 在 GitHub 上檢視
https://github.com/<your-repo>/actions/workflows/test.yml
```

### 2. 發布流程 (`.github/workflows/release.yml`)

**觸發條件**：
- 推送符合 `v*.*.*` 格式的標籤（例如：`v0.2.0`）

**執行內容**：
- ✅ **執行完整測試**
- ✅ **多平台建置**：
  - Windows (AMD64/ARM64)
  - Linux (AMD64/ARM64)
  - macOS (AMD64/ARM64)
- ✅ **建立壓縮檔**：
  - Windows: `.zip`
  - Linux/macOS: `.tar.gz`
- ✅ **計算 SHA256 checksums**
- ✅ **生成變更日誌**
- ✅ **創建 GitHub Release**
- ✅ **建置並推送 Docker 映像**

**發布新版本**：
```bash
# 1. 標記版本
git tag v0.2.0

# 2. 推送標籤
git push --tags

# 3. GitHub Actions 自動執行發布流程
# 4. 檢視發布結果
https://github.com/<your-repo>/releases
```

### 3. 安全掃描 (`.github/workflows/codeql.yml`)

**觸發條件**：
- 推送到 `main` 或 `develop` 分支
- 對 `main` 分支的 Pull Request
- 每週一自動執行

**執行內容**：
- ✅ **CodeQL 程式碼分析**
- ✅ **安全漏洞掃描**
- ✅ **程式碼品質檢查**

### 4. 依賴更新 (`.github/workflows/dependencies.yml`)

**觸發條件**：
- 每週一自動執行
- 手動觸發

**執行內容**：
- ✅ **檢查可更新的依賴**
- ✅ **自動更新依賴**
- ✅ **執行測試驗證**
- ✅ **自動創建 Pull Request**

## 配置 Secrets

某些功能需要配置 GitHub Secrets：

### Codecov（可選）

```bash
# 在 GitHub repository settings > Secrets and variables > Actions
# 新增 secret：CODECOV_TOKEN

# 1. 前往 https://codecov.io
# 2. 連接您的 GitHub repository
# 3. 複製 token
# 4. 在 GitHub 設定中新增 secret
```

### Docker Hub（可選）

```bash
# 如果要推送到 Docker Hub，需要設定：
# DOCKER_USERNAME: Docker Hub 用戶名
# DOCKER_PASSWORD: Docker Hub 密碼或 token
```

## 本地測試

在推送到 GitHub 之前，建議先在本地執行測試：

```bash
# 執行所有測試
go test ./...

# 執行測試並檢查覆蓋率
go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

# 執行 linter
golangci-lint run

# 模擬跨平台建置
GOOS=linux GOARCH=amd64 go build ./cmd/ralph-loop
GOOS=windows GOARCH=amd64 go build ./cmd/ralph-loop
GOOS=darwin GOARCH=arm64 go build ./cmd/ralph-loop
```

## 持續整合最佳實踐

### 1. 提交前檢查

```bash
# 格式化程式碼
go fmt ./...

# 執行測試
go test ./...

# 執行 linter
golangci-lint run
```

### 2. Pull Request 流程

1. 建立功能分支：`git checkout -b feature/xxx`
2. 開發並提交變更
3. 推送到 GitHub：`git push origin feature/xxx`
4. 建立 Pull Request
5. 等待 CI 檢查通過
6. Code Review
7. 合併到 `main` 或 `develop`

### 3. 版本發布流程

```bash
# 1. 確保所有測試通過
go test ./...

# 2. 更新版本號（如果需要）
# 編輯 cmd/ralph-loop/main.go 中的版本號

# 3. 提交變更
git add .
git commit -m "chore: bump version to v0.2.0"
git push

# 4. 建立標籤
git tag -a v0.2.0 -m "Release v0.2.0"
git push --tags

# 5. GitHub Actions 自動建置並發布
```

## 監控 CI/CD

### 檢視工作流程執行狀態

```bash
# 使用 GitHub CLI
gh run list
gh run view <run-id>

# 或在瀏覽器中檢視
https://github.com/<your-repo>/actions
```

### CI 徽章

在 README.md 中新增 CI 狀態徽章：

```markdown
![Test](https://github.com/<your-repo>/actions/workflows/test.yml/badge.svg)
![CodeQL](https://github.com/<your-repo>/actions/workflows/codeql.yml/badge.svg)
[![codecov](https://codecov.io/gh/<your-repo>/branch/main/graph/badge.svg)](https://codecov.io/gh/<your-repo>)
```

## 故障排除

### 測試失敗

```bash
# 檢視失敗的測試
gh run view <run-id> --log-failed

# 在本地重現問題
go test -v -race ./...
```

### Linter 錯誤

```bash
# 執行 linter 並查看所有問題
golangci-lint run --max-issues-per-linter 0

# 自動修復部分問題
golangci-lint run --fix
```

### 建置失敗

```bash
# 檢查依賴
go mod verify
go mod tidy

# 清除快取重新建置
go clean -cache
go build ./...
```

## 進階配置

### 自訂 golangci-lint 規則

編輯 `.golangci.yml`：

```yaml
linters-settings:
  gocyclo:
    min-complexity: 20  # 調整複雜度閾值
  
  lll:
    line-length: 150  # 調整行長度限制
```

### 新增自訂 workflow

在 `.github/workflows/` 目錄下建立新的 `.yml` 檔案：

```yaml
name: 自訂檢查

on:
  push:
    branches: [ main ]

jobs:
  custom-check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: 執行自訂腳本
        run: ./scripts/custom-check.sh
```

## 效能優化

### 1. 使用快取

```yaml
- name: 快取 Go 模組
  uses: actions/cache@v4
  with:
    path: |
      ~/.cache/go-build
      ~/go/pkg/mod
    key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
```

### 2. 並行執行

```yaml
strategy:
  matrix:
    os: [ubuntu-latest, macos-latest, windows-latest]
  fail-fast: false  # 不要因為一個失敗就停止所有
```

### 3. 條件執行

```yaml
- name: 上傳覆蓋率
  if: matrix.os == 'ubuntu-latest'  # 只在一個平台執行
  uses: codecov/codecov-action@v4
```

## 參考資源

- [GitHub Actions 文檔](https://docs.github.com/en/actions)
- [golangci-lint 文檔](https://golangci-lint.run/)
- [Codecov 文檔](https://docs.codecov.com/)
- [Docker Build Push Action](https://github.com/docker/build-push-action)
