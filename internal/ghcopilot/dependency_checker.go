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
	// 新版 CLI 使用自己的認證機制，先嘗試 gh auth，如失敗則提示使用 copilot /login
	cmd := exec.Command("gh", "auth", "status")
	_, err := cmd.CombinedOutput()
	if err != nil {
		// gh 認證失敗不一定是問題，因為新版 CLI 有自己的認證
		// 這裡只是警告，不阻止執行
		dc.errors = append(dc.errors, &DependencyError{
			Component: "GitHub Auth",
			Message:   "GitHub CLI 未認證（新版 Copilot CLI 可使用自己的認證）",
			Help: `認證方式：

   方法 1: 使用新版 Copilot CLI 認證（推薦）
      執行 'copilot' 然後輸入 '/login'

   方法 2: 使用 GitHub CLI 認證
      執行 'gh auth login -w'（使用瀏覽器認證）`,
		})
	}
}

// isVersionValid 檢查版本是否大於等於最低要求版本
func (dc *DependencyChecker) isVersionValid(current, minimum string) bool {
	currentParts := strings.Split(current, ".")
	minimumParts := strings.Split(minimum, ".")

	for i := 0; i < len(currentParts) && i < len(minimumParts); i++ {
		currentNum, err1 := strconv.Atoi(currentParts[i])
		minimumNum, err2 := strconv.Atoi(minimumParts[i])
		
		// 如果版本號包含非數字，視為格式無效，返回 false
		if err1 != nil || err2 != nil {
			return false
		}

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

	return fmt.Errorf("依賴檢查失敗:\n%s", output.String())
}

// GetErrors 取得所有檢查到的錯誤
func (dc *DependencyChecker) GetErrors() []*DependencyError {
	return dc.errors
}

// HasErrors 檢查是否有錯誤
func (dc *DependencyChecker) HasErrors() bool {
	return len(dc.errors) > 0
}
