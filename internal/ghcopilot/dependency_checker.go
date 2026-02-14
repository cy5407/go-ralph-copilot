package ghcopilot

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// DependencyError 代表依賴檢查失敗的錯誤
type DependencyError struct {
	Component string // 元件名稱 (e.g., "GitHub Copilot CLI", "GitHub Auth")
	Message   string // 錯誤訊息
	Help      string // 幫助文本
}

// Error 實作 error 介面
func (e *DependencyError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Component, e.Message)
}

// DependencyChecker 用於檢查所有依賴項
type DependencyChecker struct {
	errors []*DependencyError
}

// NewDependencyChecker 建立新的依賴檢查器
func NewDependencyChecker() *DependencyChecker {
	return &DependencyChecker{
		errors: []*DependencyError{},
	}
}

// CheckAll 檢查所有必需的依賴項
func (dc *DependencyChecker) CheckAll() error {
	// 注意: 新版獨立 Copilot CLI 不需要 gh CLI 或 Node.js
	dc.CheckGitHubCopilotCLI() // 檢查獨立 Copilot CLI
	dc.CheckGitHubAuth()       // 檢查認證狀態

	if len(dc.errors) > 0 {
		return dc.formatErrors()
	}
	return nil
}

// CheckNodeJS 檢查 Node.js 是否已安裝（可選，新版 CLI 不需要）
func (dc *DependencyChecker) CheckNodeJS() {
	cmd := exec.Command("node", "--version")
	output, err := cmd.Output()
	if err != nil {
		dc.errors = append(dc.errors, &DependencyError{
			Component: "Node.js",
			Message:   "未找到 Node.js，請先安裝",
			Help:      "訪問 https://nodejs.org/ 下載最新版本（>= 14.0.0）",
		})
		return
	}

	version := strings.TrimSpace(string(output))
	version = strings.TrimPrefix(version, "v")

	if !dc.isVersionValid(version, "14.0.0") {
		dc.errors = append(dc.errors, &DependencyError{
			Component: "Node.js",
			Message:   fmt.Sprintf("版本過舊：%s，需要 >= 14.0.0", version),
			Help:      "運行 'node --version' 檢查版本，然後從 https://nodejs.org/ 升級",
		})
	}
}

// CheckGitHubCopilotCLI 檢查 GitHub Copilot CLI 是否已安裝
//
// 版本說明 (2026-01-21 更新)：
//   - 本專案使用 **新版獨立** GitHub Copilot CLI (`copilot` 命令)
//   - 安裝方式：`winget install GitHub.Copilot` 或 `npm install -g @github/copilot`
//   - **舊版 `gh copilot` 已於 2025-10-25 停用**
//   - **`@githubnext/github-copilot-cli` 早已棄用**
//   - 詳見 VERSION_NOTICE.md
func (dc *DependencyChecker) CheckGitHubCopilotCLI() {
	cmd := exec.Command("copilot", "--version")
	_, err := cmd.Output()
	if err != nil {
		dc.errors = append(dc.errors, &DependencyError{
			Component: "GitHub Copilot CLI",
			Message:   "未找到 copilot 命令",
			Help: `請安裝新版獨立 GitHub Copilot CLI：

   Windows (Winget):
      winget install GitHub.Copilot

   macOS/Linux (Homebrew):
      brew install copilot-cli

   npm (全平台):
      npm install -g @github/copilot

   macOS/Linux (Install Script):
      curl -fsSL https://gh.io/copilot-install | bash

   安裝後執行 'copilot --version' 驗證。

   ⚠️ 注意：
   - 舊版 'gh copilot' 已於 2025-10-25 停用
   - 舊版 '@githubnext/github-copilot-cli' 已棄用
   - 詳見 VERSION_NOTICE.md`,
		})
		return
	}
}

// CheckGitHubCLI 檢查 GitHub CLI 是否已安裝（可選，新版 CLI 不需要）
func (dc *DependencyChecker) CheckGitHubCLI() {
	cmd := exec.Command("gh", "--version")
	_, err := cmd.Output()
	if err != nil {
		dc.errors = append(dc.errors, &DependencyError{
			Component: "GitHub CLI",
			Message:   "未找到 GitHub CLI (gh)，請先安裝（可選）",
			Help:      "訪問 https://cli.github.com/ 下載安裝程式（新版 Copilot CLI 不需要此依賴）",
		})
	}
}

// CheckGitHubAuth 檢查 GitHub 認證狀態
func (dc *DependencyChecker) CheckGitHubAuth() {
	// 測試 Copilot CLI 是否能實際執行（這比檢查 gh auth 更可靠）
	cmd := exec.Command("copilot", "-p", "test", "--model", "claude-haiku-4.5", "--silent")
	_, err := cmd.CombinedOutput()
	
	// 如果命令失敗且不是「找不到命令」錯誤，可能是認證問題
	if err != nil {
		if strings.Contains(err.Error(), "executable file not found") {
			// 這種情況已經在 CheckGitHubCopilotCLI() 中處理了
			return
		}
		
		// 其他錯誤可能是認證問題，但不一定阻止執行
		// 某些情況下 Copilot 可能因為其他原因失敗（如網路問題）
		// 所以這裡改為警告而不是錯誤
		dc.errors = append(dc.errors, &DependencyError{
			Component: "GitHub Auth",
			Message:   "無法驗證 Copilot CLI 認證狀態",
			Help: `如果遇到認證問題，請嘗試：

   方法 1: 使用 Copilot CLI 認證（推薦）
      執行 'copilot' 然後輸入 '/login'

   方法 2: 使用 GitHub CLI 認證
      執行 'gh auth login -w'（使用瀏覽器認證）
      
   方法 3: 直接測試
      執行 'copilot -p "hello" --silent' 看是否正常工作`,
		})
	}
}

// isVersionValid 檢查版本是否大於等於最低要求版本
func (dc *DependencyChecker) isVersionValid(current, minimum string) bool {
	currentParts := strings.Split(current, ".")
	minimumParts := strings.Split(minimum, ".")

	for i := 0; i < len(currentParts) && i < len(minimumParts); i++ {
		currentNum, _ := strconv.Atoi(currentParts[i])
		minimumNum, _ := strconv.Atoi(minimumParts[i])

		if currentNum > minimumNum {
			return true
		}
		if currentNum < minimumNum {
			return false
		}
	}

	return len(currentParts) >= len(minimumParts)
}

// formatErrors 格式化所有錯誤為用戶友善的訊息
func (dc *DependencyChecker) formatErrors() error {
	var output strings.Builder
	output.WriteString("\n❌ 依賴檢查失敗，找到以下問題：\n\n")

	for i, err := range dc.errors {
		output.WriteString(fmt.Sprintf("%d. %s\n", i+1, err.Error()))
		output.WriteString(fmt.Sprintf("   💡 解決方案: %s\n\n", err.Help))
	}

	output.WriteString("✅ 解決所有問題後，請重新運行本程式\n")

	return fmt.Errorf("%s", output.String())
}

// GetErrors 取得所有檢查到的錯誤
func (dc *DependencyChecker) GetErrors() []*DependencyError {
	return dc.errors
}

// HasErrors 檢查是否有錯誤
func (dc *DependencyChecker) HasErrors() bool {
	return len(dc.errors) > 0
}
