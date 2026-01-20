# Ralph Loop - AI 驅動的自動程式碼迭代系統

> 基於 GitHub Copilot CLI 的自主程式碼修正與迭代工具

## 🎯 專案狀態

### ✅ 已完成

1. **OpenSpec 框架整合**
   - 安裝並初始化 OpenSpec 工具
   - 建立完整的專案規格文件

2. **專案規劃**
   - 完成 [openspec/project.md](openspec/project.md)（Ralph Loop 完整規格）
   - 定義五階段開發路線圖
   - 明確 CLI 整合架構（使用 `github-copilot-cli` 工具）

3. **第一個變更提案：指令過濾安全層**
   - **狀態**: ✅ 已驗證通過
   - **路徑**: `openspec/changes/add-command-filter-security/`
   - **內容**: 
     - [proposal.md](openspec/changes/add-command-filter-security/proposal.md) - 安全層設計提案
     - [tasks.md](openspec/changes/add-command-filter-security/tasks.md) - 30 項實作任務
     - [specs/command-filter/spec.md](openspec/changes/add-command-filter-security/specs/command-filter/spec.md) - 8 個需求，40+ 測試場景
   - **驗證**: `openspec validate add-command-filter-security --strict` ✅ 通過

4. **第二個變更提案：GitHub Copilot CLI 整合層** 🆕
   - **狀態**: ✅ 已驗證通過
   - **路徑**: `openspec/changes/add-copilot-cli-integration/`
   - **內容**:
     - [proposal.md](openspec/changes/add-copilot-cli-integration/proposal.md) - CLI 整合設計提案
     - [tasks.md](openspec/changes/add-copilot-cli-integration/tasks.md) - 42 項實作任務（7 個階段）
     - [specs/cli-executor/spec.md](openspec/changes/add-copilot-cli-integration/specs/cli-executor/spec.md) - CLI 執行器規格（8 個需求）
     - [specs/output-parser/spec.md](openspec/changes/add-copilot-cli-integration/specs/output-parser/spec.md) - 輸出解析器規格（8 個需求）
   - **驗證**: `openspec validate add-copilot-cli-integration --strict` ✅ 通過
   - **優先級**: 最高（第一階段開發）

### 📋 專案結構

```
Github CLI 自動跌代/
├── openspec/
│   ├── project.md                              # 專案總規格
│   ├── AGENTS.md                               # AI 代理指引
│   └── changes/
│       ├── add-command-filter-security/        # 變更 1: 安全層
│       │   ├── proposal.md
│       │   ├── tasks.md
│       │   └── specs/
│       │       └── command-filter/
│       │           └── spec.md
│       └── add-copilot-cli-integration/        # 變更 2: CLI 整合 🆕
│           ├── proposal.md
│           ├── tasks.md
│           └── specs/
│               ├── cli-executor/
│               │   └── spec.md
│               └── output-parser/
│                   └── spec.md
└── README.md                                   # 本文件
```

## 🚀 下一步行動

### 選項 A: 開始實作（推薦）

基於 CLI 整合層的規格開始編寫 Golang 程式碼：

```bash
# 建立專案結構
mkdir -p internal/ghcopilot
mkdir -p internal/parser
mkdir -p cmd/ralph-loop

# 初始化 Go 模組
go mod init github.com/yourname/ralph-loop

# 開始實作 CLI 執行器
# 參考: openspec/changes/add-copilot-cli-integration/specs/cli-executor/spec.md
```

**實作順序**（按照 tasks.md）：
1. 階段 1: 專案設定與依賴檢查（1天）
2. 階段 2: CLI 執行器核心（2-3天）
3. 階段 3: 輸出解析器（2天）
4. 階段 4: 上下文管理（2天）
5. 階段 5-7: API、測試、文件（4-5天）

**總計**: 7-11 天完成 CLI 整合層

### 選項 B: 繼續規劃其他階段

建立剩餘階段的變更提案：

- **變更 3**: Ralph Loop 狀態機（Stage 2）
- **變更 4**: 沙盒執行環境（Stage 4）  
- **變更 5**: 持久化層（Stage 5）

### 選項 C: 建立原型驗證

快速建立一個最小可行原型（MVP）來驗證概念：

```go
// 簡單的 PoC: 呼叫 github-copilot-cli 並解析輸出
package main

import (
    "fmt"
    "os/exec"
)

func main() {
    cmd := exec.Command("github-copilot-cli", "what-the-shell", "列出所有 go 檔案")
    output, _ := cmd.Output()
    fmt.Println(string(output))
}
```

## 📚 OpenSpec 工作流程

### 查看變更狀態

```bash
# 列出所有變更
npx openspec list

# 查看特定變更的詳情
npx openspec change show add-copilot-cli-integration

# 驗證變更
npx openspec validate add-copilot-cli-integration --strict
```

### 追蹤任務進度

```bash
# 標記任務為進行中
npx openspec task start add-copilot-cli-integration 1.1

# 標記任務為完成
npx openspec task complete add-copilot-cli-integration 1.1

# 查看進度
npx openspec change show add-copilot-cli-integration
```

### 應用變更到專案

```bash
# 當變更完成實作後
npx openspec change apply add-copilot-cli-integration
```

## 🎓 關鍵文件導覽

### 理解專案

- **從這裡開始**: [openspec/project.md](openspec/project.md)
  - Ralph Loop 的完整架構
  - 技術棧：Golang + GitHub Copilot CLI
  - 五階段開發路線圖
  - 安全規則與約束

### CLI 整合層（當前優先）

- **提案**: [openspec/changes/add-copilot-cli-integration/proposal.md](openspec/changes/add-copilot-cli-integration/proposal.md)
  - 為什麼選擇 CLI 而非 SDK
  - 完整的技術決策理由
  - 風險與緩解策略

- **任務清單**: [openspec/changes/add-copilot-cli-integration/tasks.md](openspec/changes/add-copilot-cli-integration/tasks.md)
  - 42 個詳細任務
  - 7 個開發階段
  - 依賴關係圖

- **技術規格**:
  - [CLI 執行器](openspec/changes/add-copilot-cli-integration/specs/cli-executor/spec.md): 8 個需求，25+ 場景
  - [輸出解析器](openspec/changes/add-copilot-cli-integration/specs/output-parser/spec.md): 8 個需求，30+ 場景

### 安全層（優先級 2）

- **提案**: [openspec/changes/add-command-filter-security/proposal.md](openspec/changes/add-command-filter-security/proposal.md)
- **規格**: [openspec/changes/add-command-filter-security/specs/command-filter/spec.md](openspec/changes/add-command-filter-security/specs/command-filter/spec.md)

## 💡 技術亮點

### Ralph Loop 架構

```
┌─────────────────────────────────────────────┐
│           Ralph Loop (Golang)               │
│                                             │
│  ┌──────────────────────────────────────┐  │
│  │   Observe-Reflect-Act 迴圈           │  │
│  │                                      │  │
│  │  1. 觀察 → 讀取錯誤/測試失敗        │  │
│  │  2. 反思 → 呼叫 github-copilot-cli  │  │
│  │  3. 行動 → 執行修正 (經安全過濾)    │  │
│  └──────────────────────────────────────┘  │
│                                             │
│  ┌──────────────┐  ┌──────────────────┐   │
│  │ CLI 整合層   │  │ 指令過濾器       │   │
│  │              │  │ (黑名單驗證)     │   │
│  │ github-      │→ │                  │   │
│  │ copilot-cli  │  │ rm/format/dd... │   │
│  └──────────────┘  └──────────────────┘   │
└─────────────────────────────────────────────┘
           ↓
    ┌──────────────┐
    │ GitHub       │
    │ Copilot CLI  │
    │ (npm 套件)   │
    └──────────────┘
```

### GitHub Copilot CLI 整合方式

```bash
# 1. 安裝
github-copilot-cli
npm install -g @githubnext/github-copilot-cli

# 2. 認證
github-copilot-cli auth

# 3. 獲取 shell 指令建議
github-copilot-cli what-the-shell "修正編譯錯誤: undefined: fmt.Printl"

# 4. 獲取 git 操作建議
github-copilot-cli git-assist "如何撒銷最後一次 commit"

# 5. 獲取 GitHub CLI 建議
github-copilot-cli gh-assist "如何建立 pull request"

# 6. Ralph Loop 解析輸出
# - 提取 Markdown 程式碼區塊
# - 識別建議的指令或程式碼變更
# - 透過安全過濾器驗證
# - 自動執行（或請求確認）
```

## 📊 開發路線圖（5 階段）

| 階段 | 名稱 | 狀態 | 驗收標準 | 變更提案 |
|------|------|------|----------|----------|
| 1 | CLI 整合層 | 📝 規劃中 | 成功與 Copilot CLI 互動 | ✅ add-copilot-cli-integration |
| 2 | 狀態機核心 | 📋 待規劃 | 觀察→反思→行動迴圈運行 | - |
| 3 | 安全層 | 📝 規劃中 | 攔截所有危險指令 | ✅ add-command-filter-security |
| 4 | 沙盒環境 | 📋 待規劃 | 隔離執行 AI 生成的指令 | - |
| 5 | 持久化層 | 📋 待規劃 | 保存迭代歷史 | - |

## 🤝 貢獻

本專案使用 **OpenSpec** 進行規格驅動開發：

1. 所有變更必須先撰寫規格（`openspec/changes/`）
2. 規格包含：提案、任務清單、詳細需求、測試場景
3. 通過 `openspec validate --strict` 驗證後才能實作
4. 實作時參考規格中的接受標準和場景

## 📄 授權

待定

---

**最後更新**: 2024 年（剛完成 CLI 整合層規格）
**下一里程碑**: 開始實作 CLI 執行器（階段 1）或建立原型驗證概念
