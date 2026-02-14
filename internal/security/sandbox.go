package security

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// SandboxManager 管理沙箱執行環境
type SandboxManager struct {
	allowedCommands []string
	workDir         string
	restrictedPaths []string
}

// NewSandboxManager 創建新的沙箱管理器
func NewSandboxManager(allowedCommands []string, workDir string) *SandboxManager {
	// 如果沒有指定允許的命令，使用預設安全命令列表
	if len(allowedCommands) == 0 {
		allowedCommands = []string{
			"go", "gofmt", "git", "npm", "node", "python", "pip",
			"cat", "echo", "ls", "dir", "cd", "pwd", "mkdir", "cp", "mv",
			"test", "grep", "find", "diff",
			"copilot", "gh", // GitHub 工具
		}
	}
	
	// 預設的受限路徑
	restrictedPaths := []string{
		"/etc",      // Linux 系統配置
		"/sys",      // Linux 系統文件
		"/proc",     // Linux 進程信息
		"/dev",      // Linux 設備文件
		"C:\\Windows", // Windows 系統目錄
		"C:\\Program Files", // Windows 程序目錄
		"C:\\Users\\Public", // Windows 公共目錄
		"/usr/bin",  // Linux 系統二進制文件
		"/sbin",     // Linux 系統管理工具
	}
	
	return &SandboxManager{
		allowedCommands: allowedCommands,
		workDir:         workDir,
		restrictedPaths: restrictedPaths,
	}
}

// IsCommandAllowed 檢查命令是否被允許執行
func (sm *SandboxManager) IsCommandAllowed(command string) (bool, string) {
	if len(sm.allowedCommands) == 0 {
		// 如果沒有設定白名單，使用預設安全命令列表
		sm.allowedCommands = getDefaultAllowedCommands()
	}
	
	// 提取命令的基本名稱（移除路徑和參數）
	baseCommand := extractBaseCommand(command)
	
	// 檢查是否在白名單中
	for _, allowed := range sm.allowedCommands {
		if matchCommand(baseCommand, allowed) {
			return true, ""
		}
	}
	
	return false, fmt.Sprintf("命令 '%s' 不在允許執行的白名單中", baseCommand)
}

// IsPathAllowed 檢查路徑是否被允許訪問
func (sm *SandboxManager) IsPathAllowed(path string) (bool, string) {
	// 獲取絕對路徑
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false, fmt.Sprintf("無法解析路徑: %s", err.Error())
	}
	
	// 檢查是否在工作目錄內
	if sm.workDir != "" {
		absWorkDir, err := filepath.Abs(sm.workDir)
		if err != nil {
			return false, fmt.Sprintf("無法解析工作目錄: %s", err.Error())
		}
		
		// 檢查路徑是否在工作目錄下
		relPath, err := filepath.Rel(absWorkDir, absPath)
		if err != nil || strings.HasPrefix(relPath, "..") {
			return false, fmt.Sprintf("路徑 '%s' 在工作目錄 '%s' 之外", absPath, absWorkDir)
		}
	}
	
	// 檢查是否訪問受限路徑
	for _, restricted := range sm.restrictedPaths {
		if strings.HasPrefix(strings.ToLower(absPath), strings.ToLower(restricted)) {
			return false, fmt.Sprintf("路徑 '%s' 被安全策略禁止訪問", absPath)
		}
	}
	
	return true, ""
}

// ValidateCommandForSandbox 驗證命令是否適合沙箱執行
func (sm *SandboxManager) ValidateCommandForSandbox(command string) error {
	// 先檢查危險操作（這包含了整個命令的檢查）
	if containsDangerousOperations(command) {
		return fmt.Errorf("命令包含危險操作，不允許在沙箱中執行")
	}
	
	// 檢查是否包含命令連接符（&& || ; |）
	if regexp.MustCompile(`[&|;]`).MatchString(command) {
		return fmt.Errorf("沙箱模式不允許執行複合命令")
	}
	
	// 檢查基本命令是否允許
	allowed, reason := sm.IsCommandAllowed(command)
	if !allowed {
		return fmt.Errorf("沙箱安全檢查失敗: %s", reason)
	}
	
	// 檢查命令中的路徑
	paths := extractPathsFromCommand(command)
	for _, path := range paths {
		if allowed, reason := sm.IsPathAllowed(path); !allowed {
			return fmt.Errorf("沙箱路徑檢查失敗: %s", reason)
		}
	}
	
	return nil
}

// getDefaultAllowedCommands 獲取預設允許的命令列表
func getDefaultAllowedCommands() []string {
	return []string{
		// Go 相關工具
		"go", "gofmt", "golint", "goimports",
		
		// 版本控制
		"git",
		
		// GitHub Copilot
		"copilot", "gh",
		
		// 基本文本處理
		"cat", "type", "echo", "print",
		"grep", "find", "findstr", "where",
		
		// 文件操作（限制在工作目錄內）
		"ls", "dir", "pwd", "cd",
		"mkdir", "rmdir", "cp", "copy", "mv", "move",
		"touch", "new-item",
		
		// 測試相關
		"test", "jest", "npm", "yarn", "node",
		
		// 安全的系統工具
		"which", "whoami", "hostname", "uname",
	}
}

// extractBaseCommand 從命令行中提取基本命令名稱
func extractBaseCommand(command string) string {
	// 移除前後空白
	command = strings.TrimSpace(command)
	if command == "" {
		return ""
	}
	
	// 處理帶引號的命令路徑（如 "C:\Program Files\Git\bin\git.exe" status）
	if strings.HasPrefix(command, `"`) {
		// 查找結束引號
		if endQuote := strings.Index(command[1:], `"`); endQuote != -1 {
			baseCommand := command[1:endQuote+1]
			return extractCommandName(baseCommand)
		}
	}
	
	// 對於沒有引號但可能有空格的 Windows 路徑，嘗試智能解析
	// 如果命令以 C:\ 或包含 .exe，嘗試解析為 Windows 路徑
	if strings.Contains(command, ".exe ") {
		parts := strings.Split(command, ".exe ")
		if len(parts) >= 2 {
			exePath := parts[0] + ".exe"
			return extractCommandName(exePath)
		}
	}
	
	// 分割命令和參數
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return ""
	}
	
	baseCommand := parts[0]
	return extractCommandName(baseCommand)
}

// extractCommandName 從路徑中提取命令名稱
func extractCommandName(path string) string {
	// 移除路徑，只保留命令名稱
	baseCommand := filepath.Base(path)
	
	// 移除 Windows 的 .exe 副檔名
	if strings.HasSuffix(strings.ToLower(baseCommand), ".exe") {
		baseCommand = baseCommand[:len(baseCommand)-4]
	}
	
	return baseCommand
}

// matchCommand 檢查命令是否匹配允許列表中的模式
func matchCommand(command, pattern string) bool {
	// 支援簡單的萬用字元匹配
	if pattern == "*" {
		return true
	}
	
	// 精確匹配
	if strings.EqualFold(command, pattern) {
		return true
	}
	
	// 支援萬用字元 *
	if strings.Contains(pattern, "*") {
		regex := strings.ReplaceAll(regexp.QuoteMeta(pattern), `\*`, `.*`)
		matched, err := regexp.MatchString("(?i)^"+regex+"$", command)
		return err == nil && matched
	}
	
	return false
}

// isAbsolutePath 檢查路徑是否為絕對路徑（支持 Unix 和 Windows）
func isAbsolutePath(path string) bool {
	// 標準的 filepath.IsAbs 檢查
	if filepath.IsAbs(path) {
		return true
	}
	
	// 額外檢查 Unix 風格路徑（在 Windows 上也可能遇到）
	if strings.HasPrefix(path, "/") {
		return true
	}
	
	return false
}

// extractPathsFromCommand 從命令中提取可能的路徑
func extractPathsFromCommand(command string) []string {
	var paths []string
	
	// 分割命令為參數
	args := strings.Fields(command)
	
	for _, arg := range args {
		// 清理參數
		cleaned := strings.Trim(arg, `"'`)
		
		// 檢查是否為絕對路徑
		if isAbsolutePath(cleaned) {
			// 排除明顯不是路徑的東西（如 URL）
			if !strings.HasPrefix(cleaned, "http://") && 
			   !strings.HasPrefix(cleaned, "https://") &&
			   !strings.HasPrefix(cleaned, "ftp://") {
				paths = append(paths, cleaned)
			}
		}
	}
	
	// 額外檢查帶引號的參數
	quotedPattern := regexp.MustCompile(`['"]([^'"]*[/\\][^'"]*?)['"]`)
	quotedMatches := quotedPattern.FindAllStringSubmatch(command, -1)
	for _, match := range quotedMatches {
		if len(match) > 1 {
			path := match[1]
			if isAbsolutePath(path) {
				// 檢查是否已存在
				found := false
				for _, existing := range paths {
					if existing == path {
						found = true
						break
					}
				}
				if !found {
					paths = append(paths, path)
				}
			}
		}
	}
	
	return paths
}

// containsDangerousOperations 檢查命令是否包含危險操作
func containsDangerousOperations(command string) bool {
	dangerousPatterns := []string{
		// 系統控制
		`\bshutdown\b`, `\breboot\b`, `\bhalt\b`, `\bpoweroff\b`,
		
		// 網路操作（包含 curl -X POST）
		`curl\s+.*-x\s+(post|put|delete)`, `wget\s+.*--post`, `\bnc\s+`, `\bnetcat\b`,
		
		// 進程操作
		`\bkill\s+`, `\bkillall\b`, `\bpkill\b`, `\btaskkill\b`,
		
		// 權限提升（只匹配作為命令，不匹配引號內或文件名）
		`^\s*sudo\s+`, `^\s*su\s+`, `\brunas\b`,
		
		// 系統修改
		`chmod\s+.*777`, `chown\s+.*root`, `\bpasswd\b`,
		
		// 危險的刪除操作
		`\brm\s+[^']*-rf\b`, `\bdel\s+[^/]*[/\\]\w*\s*[/\\]\w*`, 
		`\brmdir\s+[^']*[/\\]s`,
		
		// 格式化操作
		`\bformat\s+[A-Za-z]:`, `\bmkfs\b`, `\bfdisk\b`,
		
		// 服務操作
		`\bsystemctl\b`, `\bservice\s+`, `\bsc\s+`,
		
		// 環境變數修改（可能危險的）
		`export\s+[^=]*PATH`, `set\s+[^=]*PATH`,
	}
	
	lowerCommand := strings.ToLower(command)
	
	// 不檢查引號內的內容
	if strings.Contains(command, "'") || strings.Contains(command, "\"") {
		// 移除引號內的內容進行檢查
		singleQuoteRegex := regexp.MustCompile(`'[^']*'`)
		doubleQuoteRegex := regexp.MustCompile(`"[^"]*"`)
		
		cleanCommand := singleQuoteRegex.ReplaceAllString(lowerCommand, "")
		cleanCommand = doubleQuoteRegex.ReplaceAllString(cleanCommand, "")
		
		for _, pattern := range dangerousPatterns {
			matched, err := regexp.MatchString(pattern, cleanCommand)
			if err == nil && matched {
				return true
			}
		}
		return false
	}
	
	// 正常檢查
	for _, pattern := range dangerousPatterns {
		matched, err := regexp.MatchString(pattern, lowerCommand)
		if err == nil && matched {
			return true
		}
	}
	
	return false
}

// CreateRestrictedEnvironment 創建受限的執行環境
func (sm *SandboxManager) CreateRestrictedEnvironment() map[string]string {
	env := make(map[string]string)
	
	// 只保留必要的環境變數
	essentialVars := []string{
		"PATH", "HOME", "USER", "USERNAME", "TEMP", "TMP",
		"GO111MODULE", "GOPROXY", "GOSUMDB", "GOPATH", "GOROOT",
		"COPILOT_CLI_PATH",
	}
	
	for _, varName := range essentialVars {
		if value := os.Getenv(varName); value != "" {
			env[varName] = value
		}
	}
	
	// 如果設定了工作目錄，限制某些路徑相關的環境變數
	if sm.workDir != "" {
		env["PWD"] = sm.workDir
		env["OLDPWD"] = sm.workDir
	}
	
	// 添加沙箱標識
	env["RALPH_SANDBOX_MODE"] = "true"
	
	return env
}