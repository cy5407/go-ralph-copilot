# Task 3: GitHub Release 產品化

**目標**：讓使用者可以直接從 GitHub Release 下載預編譯的 binary，不需要安裝 Go 環境。

**現狀**：已有 `.github/workflows/release.yml` 和 `test.yml`，架構完整但有多個 bug 導致無法正常運作。

---

## T3-001: 修正 release.yml 🔧

**狀態**: ❌ 待開始
**優先級**: P0（阻擋 release）
**檔案**: `.github/workflows/release.yml`

### Bug 1: 版本號注入失敗（ldflags 變數名不匹配）

**位置**: `release.yml:39` 和 `cmd/ralph-loop/main.go:18`

release.yml 中 ldflags 注入的是大寫 `main.Version`：
```yaml
# release.yml:39 (錯誤)
go build -ldflags="-s -w -X main.Version=${{ steps.version.outputs.VERSION }}"
```

但 main.go 中的變數是小寫 `version`：
```go
// cmd/ralph-loop/main.go:18 (現狀)
var (
    version = "0.1.0"
)
```

**修法（二擇一）**：

方案 A — 改 main.go（推薦，符合 Go 慣例讓 ldflags 用大寫）：
```go
// cmd/ralph-loop/main.go:18
var (
    Version = "0.1.0"
)
```
然後同步修改 main.go 中所有引用 `version` 的地方改成 `Version`（共 2 處）：
- `main.go:149`: `fmt.Printf("Ralph Loop v%s\n", version)` → `Version`
- `main.go:278`: 結尾的 `, version)` → `, Version)`

方案 B — 改 release.yml ldflags（不動 main.go）：
```yaml
# 把所有 main.Version 改成 main.version
-ldflags="-s -w -X main.version=${{ steps.version.outputs.VERSION }}"
```
此修改在 release.yml 中出現 **6 次**（6 個平台各一次，行 39-54）。

### Bug 2: Go 版本過舊

**位置**: `release.yml:22`

```yaml
# 現狀 (錯誤)
go-version: '1.21'

# 修正
go-version: '1.24'
```

`go.mod` 要求 `go 1.24.0`，用 1.21 會直接編譯失敗。

### Bug 3: 壓縮步驟刪除邏輯會誤刪壓縮檔

**位置**: `release.yml:70-71`

```bash
# 現狀 (錯誤) — ralph-loop-* 會匹配到 .zip 和 .tar.gz
rm -f *.exe ralph-loop-*

# 修正 — 只刪除未壓縮的 binary
rm -f ralph-loop-windows-amd64.exe ralph-loop-windows-arm64.exe
rm -f ralph-loop-linux-amd64 ralph-loop-linux-arm64
rm -f ralph-loop-darwin-amd64 ralph-loop-darwin-arm64
```

### Bug 4: Release body 引用已刪除的文件

**位置**: `release.yml:150-154`

```yaml
# 現狀 (錯誤) — 這兩個檔案已被刪除
- [USAGE_GUIDE.md](...)
- [DEPLOYMENT_GUIDE.md](...)

# 修正 — 只保留 README
## 📚 文檔

- [README.md](https://github.com/${{ github.repository }}/blob/${{ steps.version.outputs.VERSION }}/README.md)
```

### Bug 5: Docker job 會失敗

**位置**: `release.yml:167-207`

Docker job 需要 `DOCKER_USERNAME` 和 `DOCKER_PASSWORD` secrets，目前沒有設定。

**修法**：整個 `docker:` job 區塊（行 167-207）暫時刪除或註解掉。等未來真的要推 Docker Hub 時再加回來。

### Bug 6: release.yml 測試命令缺少 -short

**位置**: `release.yml:32`

```yaml
# 現狀 (錯誤) — 會跑需要真實 Copilot 的測試
run: go test -v ./...

# 修正
run: go test -short -timeout 3m ./...
```

### 修改清單

- [ ] 修正版本注入（Bug 1：改 main.go 或改 ldflags）
- [ ] Go 版本 `1.21` → `1.24`（Bug 2：release.yml:22）
- [ ] 修正壓縮步驟的 rm 指令（Bug 3：release.yml:70-71）
- [ ] 修正 release body 移除已刪除文件連結（Bug 4：release.yml:150-154）
- [ ] 移除 Docker job（Bug 5：release.yml:167-207）
- [ ] 測試命令加 `-short -timeout 3m`（Bug 6：release.yml:32）

---

## T3-002: 修正 test.yml 🔧

**狀態**: ❌ 待開始
**優先級**: P0（阻擋 CI）
**檔案**: `.github/workflows/test.yml`

### Bug 1: Go 版本矩陣全部過舊

**位置**: `test.yml:15`

```yaml
# 現狀 (錯誤) — go.mod 要求 1.24.0，以下三個版本全部會失敗
go-version: ['1.21', '1.22', '1.23']

# 修正
go-version: ['1.24']
```

### Bug 2: 測試命令會跑需要真實 Copilot 的測試

**位置**: `test.yml:44`

```yaml
# 現狀 (錯誤)
run: go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

# 修正 — 加上 -short 跳過需要外部服務的測試
run: go test -short -race -coverprofile=coverage.txt -covermode=atomic -timeout 3m ./...
```

### Bug 3: Codecov 條件引用舊版本

**位置**: `test.yml:47`

```yaml
# 現狀 (錯誤) — 版本矩陣已改，這裡的條件也要更新
if: matrix.os == 'ubuntu-latest' && matrix.go-version == '1.21'

# 修正
if: matrix.os == 'ubuntu-latest' && matrix.go-version == '1.24'
```

### Bug 4: build job 的 Go 版本也過舊

**位置**: `test.yml:100`

```yaml
# 現狀 (錯誤)
go-version: '1.21'

# 修正
go-version: '1.24'
```

### Bug 5: lint job 的 Go 版本也過舊

**位置**: `test.yml:67`

```yaml
# 現狀 (錯誤)
go-version: '1.21'

# 修正
go-version: '1.24'
```

### 修改清單

- [ ] Go 版本矩陣改為 `['1.24']`（Bug 1：test.yml:15）
- [ ] 測試命令加 `-short -timeout 3m`（Bug 2：test.yml:44）
- [ ] Codecov 條件更新版本號（Bug 3：test.yml:47）
- [ ] build job Go 版本改 `1.24`（Bug 4：test.yml:100）
- [ ] lint job Go 版本改 `1.24`（Bug 5：test.yml:67）

---

## T3-003: 提交修正並驗證 CI 🧪

**狀態**: ❌ 待開始
**優先級**: P0
**前置條件**: T3-001、T3-002 完成

**步驟**：
- [ ] 提交所有修正到 master 分支
- [ ] 推送並等待 test.yml 通過（在 GitHub Actions 頁面確認）
- [ ] 如果 CI 失敗，根據錯誤訊息繼續修正

**驗證方式**：
```bash
# 推送後，到這個頁面確認 CI 狀態
# https://github.com/cy5407/go-ralph-copilot/actions/workflows/test.yml
```

---

## T3-004: 觸發首次正式 Release 🚀

**狀態**: ❌ 待開始
**優先級**: P1
**前置條件**: T3-003 CI 通過

**步驟**：
- [ ] 刪除舊的 v0.1.0 tag（之前手動建的，沒走 CI）
- [ ] 重新打 tag 並推送，觸發 release.yml

```bash
# 刪除本地和遠端的舊 tag
git tag -d v0.1.0
git push origin :refs/tags/v0.1.0

# 重新打 tag
git tag v0.1.0
git push origin v0.1.0
```

- [ ] 到 GitHub Actions 確認 release workflow 通過
- [ ] 到 GitHub Release 頁面確認有 6 個平台 binary + checksums.txt

**預期產出**：
```
https://github.com/cy5407/go-ralph-copilot/releases/tag/v0.1.0

Release v0.1.0
├── ralph-loop-v0.1.0-windows-amd64.zip
├── ralph-loop-v0.1.0-windows-arm64.zip
├── ralph-loop-v0.1.0-linux-amd64.tar.gz
├── ralph-loop-v0.1.0-linux-arm64.tar.gz
├── ralph-loop-v0.1.0-darwin-amd64.tar.gz
├── ralph-loop-v0.1.0-darwin-arm64.tar.gz
└── checksums.txt
```

- [ ] 下載 Windows AMD64 版本，解壓後執行 `ralph-loop.exe version`，確認顯示 `v0.1.0`

---

## T3-005: 更新 README 安裝說明 📝

**狀態**: ❌ 待開始
**優先級**: P1
**前置條件**: T3-004 Release 成功
**檔案**: `README.md`

在 README.md 的「安裝」區塊，新增「下載預編譯 binary」作為第一選項：

```markdown
## 安裝

### 方法一：下載預編譯 binary（推薦）

到 [Release 頁面](https://github.com/cy5407/go-ralph-copilot/releases/latest) 下載對應平台的檔案，解壓後放到系統 PATH 中。

### 方法二：go install

需要 Go 1.24+：

\```bash
go install github.com/cy5407/go-ralph-copilot/cmd/ralph-loop@latest
\```
```

### 修改清單

- [ ] 新增下載 binary 安裝方式到 README.md
- [ ] 保留 go install 作為替代方案
- [ ] 提交推送

---

## 完成條件

全部任務完成後，使用者只需要：
1. 到 Release 頁面下載 exe
2. 放到 PATH
3. `ralph-loop run -prompt "..." -max-loops 5`

不需要裝 Go、不需要 clone repo、不需要 build。
