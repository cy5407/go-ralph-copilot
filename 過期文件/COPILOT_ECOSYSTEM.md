# GitHub Copilot 生態系統完整說明

> **重要更新 (2026-01-21)**：GitHub 已於 2025 年 10 月發布全新的獨立 Copilot CLI，取代了舊版 `gh copilot` 擴充功能。

## 概述

GitHub Copilot 有多個不同的工具和介面，很容易混淆。本文檔整理所有相關工具及其關係。

---

## 1. GitHub Copilot (核心服務)

### 是什麼？
GitHub Copilot 是 AI 編程助手**服務**，運行在 GitHub 雲端。

### 訪問方式
- **IDE 擴充功能**：VS Code、Visual Studio、JetBrains、Neovim 等
- **GitHub.com**：網頁介面
- **CLI**：命令列介面（本文重點）

### 官方文檔
- 主頁：https://github.com/features/copilot
- 文檔：https://docs.github.com/en/copilot

---

## 2. GitHub Copilot CLI（新版獨立工具）

### 是什麼？
**全新的獨立命令列工具**，提供完整的 Agentic AI 體驗。於 2025 年 10 月發布，取代了舊版 `gh copilot` 擴充功能。

### 安裝方式

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

**macOS/Linux (Install Script)**：
```bash
curl -fsSL https://gh.io/copilot-install | bash
```

### 驗證安裝
```powershell
copilot --version
# 輸出範例：GitHub Copilot CLI v1.x.x
```

### 主要功能
```bash
# 啟動互動式 CLI
copilot

# 登入 GitHub
/login

# 選擇 AI 模型
/model
# 可選: Claude Sonnet 4.5 (預設), Claude Sonnet 4, GPT-5, GPT-5 mini, GPT-4.1
```

### 特點
- 完整的 **Agentic AI 助手**
- 支援多種 AI 模型
- 內建專門化代理：Explore (快速程式碼分析)、Task (執行測試和建置)
- **不需要** GitHub CLI (`gh`) 作為依賴
- **不需要** wrapper script

### 官方文檔
- **安裝指南**：https://docs.github.com/en/copilot/how-tos/set-up/install-copilot-cli
- **Repository**：https://github.com/github/copilot-cli
- **功能頁面**：https://github.com/features/copilot/cli

---

## 3. GitHub CLI (`gh`)（非必須）

### 是什麼？
GitHub 官方的命令列工具，用於與 GitHub 平台互動（管理 repo、PR、issues 等）。

### 與新版 Copilot CLI 的關係
- **新版 Copilot CLI 不需要 `gh` 作為依賴**
- 兩者是獨立的工具
- 如果你需要管理 GitHub 資源（PR、issues），仍可安裝 `gh`

### 安裝方式（可選）
```powershell
winget install --id GitHub.cli
# 或從 https://cli.github.com/ 下載
```

### 官方文檔
- 主頁：https://cli.github.com/
- 安裝指南：https://github.com/cli/cli#installation

---

## 4. ~~`gh copilot`~~（已停用）

### 是什麼？
**已於 2025 年 10 月 25 日停止運作**。這曾是 GitHub CLI 的擴充功能。

### 狀態
- 2025 年 9 月 25 日公告棄用
- 2025 年 10 月 25 日停止運作
- 已被新版獨立 Copilot CLI 取代

### 官方公告
- https://github.blog/changelog/2025-09-25-upcoming-deprecation-of-gh-copilot-cli-extension/

---

## 5. Copilot CLI Server Mode

### 是什麼？
新版 Copilot CLI 支援以 **server 模式** 運行，通過 JSON-RPC 協議與外部程式通訊。

### 用途
- 不是給終端使用者直接使用
- 是給**開發者**用來整合 Copilot 到自己的應用程式中
- 這就是 **GitHub Copilot SDK** 所依賴的

### 架構圖
```
你的應用程式 (例如 Ralph Loop)
       ↓
  Copilot SDK (Go/Python/Node.js/.NET)
       ↓ JSON-RPC
  copilot CLI (server mode)
       ↓
  GitHub Copilot 雲端服務
```

### 如何啟動 Server 模式？
**SDK 會自動處理**，你不需要手動啟動。SDK 內部會：
1. 找到 `copilot` 命令
2. 執行 `copilot` 並以 server 模式啟動
3. 通過 stdin/stdout 或 TCP 進行 JSON-RPC 通訊

---

## 6. GitHub Copilot SDK

### 是什麼？
多語言的 SDK，讓開發者可以**程式化**地使用 Copilot CLI。

### 狀態
於 2026 年 1 月 14 日進入 **Technical Preview**。

### 支援的語言
| 語言 | 套件名稱 | 安裝指令 |
|------|---------|---------|
| **Go** | `copilot-cli-sdk-go` | `go get github.com/github/copilot-cli-sdk-go` |
| **Node.js** | `@github/copilot-cli-sdk` | `npm install @github/copilot-cli-sdk` |
| **Python** | `copilot` | `pip install copilot` |
| **.NET** | `GitHub.Copilot.SDK` | `dotnet add package GitHub.Copilot.SDK` |

### 依賴關係
- **必須先安裝**：新版獨立 `copilot` CLI
- SDK 會自動啟動 `copilot` 作為 server

### Go SDK 範例
```go
import copilot "github.com/github/copilot-cli-sdk-go"

client := copilot.NewClient(nil)
client.Start() // SDK 自動找到並啟動 "copilot"

session, _ := client.CreateSession(&copilot.SessionConfig{
    Model: "claude-sonnet-4.5",
})

session.Send(copilot.MessageOptions{
    Prompt: "What is 2+2?",
})
```

### 官方文檔
- **SDK Repository**：https://github.com/github/copilot-sdk
- **公告**：https://github.blog/changelog/2026-01-14-copilot-sdk-in-technical-preview/

---

## 7. 已棄用的工具

### ❌ `@githubnext/github-copilot-cli`

**這是什麼？**
- 一個**獨立的**互動式命令列工具（早期實驗性）
- 提供 `what-the-shell`、`git-assist` 等子命令
- **已不再維護**

**為什麼混淆？**
- 名字很像 "Copilot CLI"
- 實際上是早期的實驗性工具
- **SDK 不使用這個套件**

### ❌ `gh copilot`

**這是什麼？**
- 曾是 GitHub CLI 的擴充功能
- **已於 2025 年 10 月 25 日停止運作**

---

## 8. Ralph Loop 應該使用什麼？

### 推薦架構（2026 年 1 月）

**前提條件**：
1. ✅ 安裝新版獨立 `copilot` CLI
2. ✅ 安裝 Go SDK：`go get github.com/github/copilot-cli-sdk-go`

**檢查是否已安裝**：
```powershell
copilot --version   # 應該有輸出 (例如 GitHub Copilot CLI v1.x.x)
```

### 依賴關係圖

```
┌─────────────────────────────────────────────────────────────┐
│                     Ralph Loop 應用程式                      │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│              internal/ghcopilot (本專案模組)                 │
└─────────────────────────────────────────────────────────────┘
                              │
            ┌─────────────────┴─────────────────┐
            ▼                                   ▼
┌───────────────────────────┐       ┌───────────────────────────┐
│   copilot CLI (獨立版)    │       │  GitHub Copilot SDK (Go)  │
│   ✅ 直接安裝             │       │  copilot-cli-sdk-go       │
└───────────────────────────┘       └───────────────────────────┘
            │                                   │
            └───────────────┬───────────────────┘
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                GitHub Copilot 雲端服務                       │
└─────────────────────────────────────────────────────────────┘
```

---

## 9. 總結對照表

| 工具名稱 | 類型 | 狀態 | 安裝方式 |
|---------|------|------|---------|
| **`copilot`** | 獨立 CLI | ✅ **官方推薦** | `winget install GitHub.Copilot` |
| **Copilot SDK (Go)** | Go 套件 | ✅ Technical Preview | `go get github.com/github/copilot-cli-sdk-go` |
| `gh` | CLI 工具 | ⚠️ 非必須 | `winget install GitHub.cli` |
| ~~`gh copilot`~~ | gh 擴充功能 | ❌ **已停用** | - |
| ~~`@githubnext/github-copilot-cli`~~ | npm 套件 | ❌ **已棄用** | - |

---

## 10. 下一步行動

1. **安裝新版 CLI**：
```powershell
winget install GitHub.Copilot
```

2. **驗證**：
```powershell
copilot --version
# 應該輸出版本號
```

3. **更新 SDK（如使用）**：
```bash
go get github.com/github/copilot-cli-sdk-go
```

4. **移除舊版依賴（如有）**：
```bash
# 移除舊版 SDK
go get github.com/github/copilot-sdk/go@none

# 移除 gh copilot 擴充功能
gh extension remove gh-copilot
```

---

## 11. 官方文檔總整理

1. **Copilot CLI 安裝**：https://docs.github.com/en/copilot/how-tos/set-up/install-copilot-cli
2. **Copilot CLI Repository**：https://github.com/github/copilot-cli
3. **Copilot CLI 功能頁面**：https://github.com/features/copilot/cli
4. **Copilot SDK Repository**：https://github.com/github/copilot-sdk
5. **gh-copilot 棄用公告**：https://github.blog/changelog/2025-09-25-upcoming-deprecation-of-gh-copilot-cli-extension/
6. **Copilot CLI 增強功能 (2026-01)**：https://github.blog/changelog/2026-01-14-github-copilot-cli-enhanced-agents-context-management-and-new-ways-to-install/

---

**最後更新**：2026-01-21
