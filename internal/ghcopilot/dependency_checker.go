package ghcopilot

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// DependencyError 代表依賴檢查失敗的錯誤
type DependencyError struct {
	Component string // 元件名稱 (e.g., "Node.js", "GitHub Copilot CLI", "GitHub CLI")
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
	// 注意: Node.js 不再是必須的,因為 copilot CLI 可以通過 winget/brew 安裝
	// 但如果存在,仍然檢查版本
	dc.CheckGitHubCopilotCLI() // 優先檢查 Copilot CLI
	dc.CheckGitHubCLI()
	dc.CheckGitHubAuth()

	if len(dc.errors) > 0 {
		return dc.formatErrors()
	}
	return nil
}

// CheckNodeJS 檢查 Node.js 是否已安裝
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
func (dc *DependencyChecker) CheckGitHubCopilotCLI() {
	cmd := exec.Command("copilot", "--version")
	_, err := cmd.Output()
	if err != nil {
		dc.errors = append(dc.errors, &DependencyError{
			Component: "GitHub Copilot CLI",
			Message:   "未找到 copilot CLI,請先安裝",
			Help:      "運行以下其中一個指令:\n   - Windows: winget install GitHub.Copilot\n   - macOS/Linux: brew install copilot-cli\n   - 跨平台: npm install -g @github/copilot\n   更多資訊: https://docs.github.com/en/copilot/how-tos/set-up/install-copilot-cli",
		})
		return
	}
}

// CheckGitHubCLI 檢查 GitHub CLI 是否已安裝
func (dc *DependencyChecker) CheckGitHubCLI() {
	cmd := exec.Command("gh", "--version")
	_, err := cmd.Output()
	if err != nil {
		dc.errors = append(dc.errors, &DependencyError{
			Component: "GitHub CLI",
			Message:   "未找到 GitHub CLI (gh)，請先安裝",
			Help:      "訪問 https://cli.github.com/ 下載安裝程式",
		})
	}
}

// CheckGitHubAuth 檢查 GitHub 認證狀態
func (dc *DependencyChecker) CheckGitHubAuth() {
	cmd := exec.Command("gh", "auth", "status")
	_, err := cmd.CombinedOutput()
	if err != nil {
		dc.errors = append(dc.errors, &DependencyError{
			Component: "GitHub Auth",
			Message:   "未認證或認證已過期",
			Help:      "運行: gh auth login -w (使用瀏覽器認證)",
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
