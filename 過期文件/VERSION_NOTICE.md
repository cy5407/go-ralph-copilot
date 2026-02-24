# GitHub Copilot 版本標示

> **重要提醒**：本文檔用於標示專案使用的 GitHub Copilot 版本，避免後續閱讀者混淆。

---

## 本專案應使用的版本

| 組件 | 版本 | 狀態 |
|------|------|------|
| **GitHub Copilot CLI** | `copilot` (獨立 CLI) | ✅ **新版（官方推薦）** |
| **GitHub Copilot SDK (Go)** | `github.com/github/copilot-cli-sdk-go` | ✅ **新版 (Technical Preview)** |
| **GitHub CLI** | `gh` v2.x | ⚠️ 非必須（新版 CLI 獨立運作） |

---

## 新舊版本對照表

### ⚠️ 避免混淆 - 2026 年 1 月更新

| 名稱 | 類型 | 狀態 | 說明 |
|------|------|------|------|
| **`copilot`** | 獨立 CLI 工具 | ✅ **新版/官方推薦** | 2025 年 10 月發布，本專案應遷移至此 |
| `github.com/github/copilot-cli-sdk-go` | Go SDK | ✅ **新版 (Technical Preview)** | 2026 年 1 月發布的新 SDK |
| `gh copilot` | GitHub CLI 擴充功能 | ❌ **已停用 (2025-10-25)** | 已被獨立 CLI 取代 |
| `github.com/github/copilot-sdk/go` | Go SDK | ⚠️ **舊版** | 本專案目前使用，需遷移 |
| `@githubnext/github-copilot-cli` | npm 套件 | ❌ **已淘汰** | 早期實驗性工具 |

### 詳細說明

#### ✅ 新版：獨立 GitHub Copilot CLI（本專案應遷移至此）

- **官方名稱**：GitHub Copilot CLI
- **安裝方式**：
  - Windows: `winget install GitHub.Copilot`
  - macOS/Linux: `brew install copilot-cli`
  - npm (全平台): `npm install -g @github/copilot`
- **呼叫方式**：直接執行 `copilot`
- **特點**：
  - 完整的 Agentic AI 助手
  - 支援多種 AI 模型（Claude Sonnet 4.5、GPT-5 等）
  - 內建 Explore agent 和 Task agent
  - **不需要** `gh` CLI 作為依賴
  - **不需要** wrapper script
- **文檔**：
  - https://docs.github.com/en/copilot/how-tos/set-up/install-copilot-cli
  - https://github.com/github/copilot-cli

#### ❌ 已停用：`gh copilot`（2025 年 10 月 25 日停止運作）

- **類型**：GitHub CLI (`gh`) 的擴充功能
- **安裝方式**：~~`gh extension install github/gh-copilot`~~
- **狀態**：
  - 2025 年 9 月 25 日公告棄用
  - 2025 年 10 月 25 日停止運作
- **官方公告**：https://github.blog/changelog/2025-09-25-upcoming-deprecation-of-gh-copilot-cli-extension/

#### ❌ 已淘汰：`@githubnext/github-copilot-cli`

- **類型**：獨立的 npm 套件
- **安裝方式**：~~`npm install -g @githubnext/github-copilot-cli`~~
- **狀態**：早期實驗性工具，已不再維護

---

## 程式碼中的 `copilot` 命令

本專案在 `cli_executor.go` 中直接呼叫 `copilot` 命令：

```go
cmd := exec.CommandContext(execCtx, "copilot", args...)
```

### 新版解決方案（推薦）

安裝新版獨立 CLI 後，系統路徑中會自動有 `copilot` 命令：

**Windows (Winget)**：
```powershell
winget install GitHub.Copilot
```

**macOS/Linux (Homebrew)**：
```bash
brew install copilot-cli
```

**npm (全平台)**：
```bash
npm install -g @github/copilot
```

安裝後直接執行 `copilot` 即可，**不需要任何 wrapper**。

### 舊版解決方案（已過時，僅供參考）

~~建立 wrapper 讓 `copilot` 命令指向 `gh copilot`~~

> ⚠️ 此方案已不適用，因為 `gh copilot` 已於 2025 年 10 月 25 日停止運作。

---

## 依賴關係圖（新版架構）

```
┌─────────────────────────────────────────────────────────────┐
│                     Ralph Loop 應用程式                      │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│              internal/ghcopilot (本專案模組)                 │
│  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────┐│
│  │  cli_executor   │  │ response_analyzer│  │circuit_breaker││
│  └─────────────────┘  └─────────────────┘  └──────────────┘│
└─────────────────────────────────────────────────────────────┘
                              │
            ┌─────────────────┴─────────────────┐
            ▼                                   ▼
┌───────────────────────────┐       ┌───────────────────────────┐
│   copilot CLI (獨立版)    │       │  GitHub Copilot SDK (Go)  │
│   ✅ 新版 - 直接安裝      │       │  copilot-cli-sdk-go       │
└───────────────────────────┘       └───────────────────────────┘
            │                                   │
            └───────────────┬───────────────────┘
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                GitHub Copilot 雲端服務                       │
└─────────────────────────────────────────────────────────────┘
```

---

## 常見混淆情境

### 情境 1：「找不到 copilot 命令」

**錯誤訊息**：
```
copilot : 無法辨識 'copilot' 詞彙是否為 Cmdlet、函式、指令檔或可執行程式的名稱。
```

**原因**：沒有安裝新版獨立 Copilot CLI。

**解決**：
```powershell
# Windows
winget install GitHub.Copilot

# macOS/Linux
brew install copilot-cli

# 或使用 npm
npm install -g @github/copilot
```

### 情境 2：「SDK protocol version mismatch」

**錯誤訊息**：
```
SDK protocol version mismatch: SDK expects version 1,
but server does not report a protocol version
```

**原因**：使用舊版 SDK (`github.com/github/copilot-sdk/go`) 或舊版 CLI (`gh copilot`)。

**解決**：
1. 安裝新版獨立 CLI：`winget install GitHub.Copilot`
2. 遷移至新版 SDK：`go get github.com/github/copilot-cli-sdk-go`

### 情境 3：混淆「哪個是 Copilot CLI」

**正確理解（2026 年 1 月）**：
- `copilot` (獨立 CLI) = 新版（✅ 使用這個）
- `gh copilot` = 已停用（❌ 不再可用）
- `@githubnext/github-copilot-cli` = 已淘汰（❌ 不用這個）

---

## 版本檢查指令

```powershell
# 檢查新版獨立 Copilot CLI
copilot --version
# 輸出範例: GitHub Copilot CLI v1.x.x

# 檢查 GitHub CLI（新版 CLI 不需要此依賴）
gh --version
# 輸出範例: gh version 2.xx.x (2025-xx-xx)
```

---

## 官方文檔連結

- [GitHub Copilot CLI - 安裝指南](https://docs.github.com/en/copilot/how-tos/set-up/install-copilot-cli)
- [GitHub Copilot CLI Repository](https://github.com/github/copilot-cli)
- [GitHub Copilot SDK Repository](https://github.com/github/copilot-sdk)
- [gh-copilot 棄用公告](https://github.blog/changelog/2025-09-25-upcoming-deprecation-of-gh-copilot-cli-extension/)
- [Copilot CLI 增強功能公告 (2026-01-14)](https://github.blog/changelog/2026-01-14-github-copilot-cli-enhanced-agents-context-management-and-new-ways-to-install/)

---

**最後更新**：2026-01-21
