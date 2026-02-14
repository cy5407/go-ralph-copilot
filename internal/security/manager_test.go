package security

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSecurityManager_Basic(t *testing.T) {
	config := DefaultSecurityConfig()
	config.SandboxMode = true
	config.AllowedCommands = []string{"git", "go"}
	config.EnableAuditLog = true
	config.EncryptCredentials = true
	
	sm := NewSecurityManager(config, "test-session")
	
	if sm == nil {
		t.Fatal("NewSecurityManager 應該返回有效的實例")
	}
	
	// 檢查各個功能是否正確啟用
	if !sm.IsSandboxEnabled() {
		t.Error("沙箱模式應該被啟用")
	}
	
	if !sm.IsAuditEnabled() {
		t.Error("審計日誌應該被啟用")
	}
	
	if !sm.IsEncryptionEnabled() {
		t.Error("加密功能應該被啟用")
	}
}

func TestSecurityManager_ValidateCommand(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "security-test")
	if err != nil {
		t.Fatalf("創建臨時目錄失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	config := SecurityConfig{
		SandboxMode:     true,
		AllowedCommands: []string{"git", "go", "echo"},
		WorkDir:         tempDir,
		EnableAuditLog:  true,
		AuditLogDir:     filepath.Join(tempDir, "audit"),
	}
	
	sm := NewSecurityManager(config, "test-session")
	
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
		{"curl evil.com", true},
	}
	
	for _, tc := range testCases {
		err := sm.ValidateCommand(tc.command)
		hasErr := (err != nil)
		if hasErr != tc.shouldErr {
			t.Errorf("ValidateCommand('%s'): 期望錯誤 %v，實際錯誤 %v (錯誤: %v)", 
				tc.command, tc.shouldErr, hasErr, err)
		}
	}
}

func TestSecurityManager_ExecuteCommand(t *testing.T) {
	config := SecurityConfig{
		SandboxMode:     true,
		AllowedCommands: []string{"echo", "cat"},
		EnableAuditLog:  true,
		MaskSensitiveInfo: true,
	}
	
	sm := NewSecurityManager(config, "test-session")
	
	// 模擬執行器
	mockExecutor := func(command string) (string, error) {
		if strings.Contains(command, "echo") {
			return "password=secret123\ntoken=abc456", nil
		}
		return "", nil
	}
	
	// 測試允許的命令
	output, err := sm.ExecuteCommand("echo test", mockExecutor)
	if err != nil {
		t.Fatalf("執行允許的命令失敗: %v", err)
	}
	
	// 檢查敏感資訊遮罩
	if strings.Contains(output, "secret123") || strings.Contains(output, "abc456") {
		t.Error("輸出應該遮罩敏感資訊")
	}
	
	// 測試不允許的命令
	_, err = sm.ExecuteCommand("rm -rf /", mockExecutor)
	if err == nil {
		t.Error("不允許的命令應該被拒絕")
	}
}

func TestSecurityManager_CredentialEncryption(t *testing.T) {
	config := SecurityConfig{
		EncryptCredentials: true,
		EncryptionPassword: "test-password",
	}
	
	sm := NewSecurityManager(config, "test-session")
	
	testCredential := "secret-api-key-12345"
	
	// 測試加密
	encrypted, err := sm.EncryptCredential(testCredential)
	if err != nil {
		t.Fatalf("加密憑證失敗: %v", err)
	}
	
	if encrypted == testCredential {
		t.Fatal("加密後的憑證應該與原憑證不同")
	}
	
	// 測試解密
	decrypted, err := sm.DecryptCredential(encrypted)
	if err != nil {
		t.Fatalf("解密憑證失敗: %v", err)
	}
	
	if decrypted != testCredential {
		t.Fatalf("解密結果不匹配，期望: %s，實際: %s", testCredential, decrypted)
	}
}

func TestSecurityManager_ValidateFilePath(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "file-security-test")
	if err != nil {
		t.Fatalf("創建臨時目錄失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	config := SecurityConfig{
		SandboxMode: true,
		WorkDir:     tempDir,
		EnableAuditLog: true,
		AuditLogDir: filepath.Join(tempDir, "audit"),
	}
	
	sm := NewSecurityManager(config, "test-session")
	
	// 允許的路徑（工作目錄內）
	allowedPath := filepath.Join(tempDir, "test.txt")
	err = sm.ValidateFilePath(allowedPath, "READ")
	if err != nil {
		t.Errorf("工作目錄內的路徑應該被允許: %v", err)
	}
	
	// 禁止的路徑（系統目錄）
	deniedPath := "/etc/passwd"
	err = sm.ValidateFilePath(deniedPath, "READ")
	if err == nil {
		t.Error("系統敏感路徑應該被禁止")
	}
}

func TestSecurityManager_GetSecurityStatus(t *testing.T) {
	config := SecurityConfig{
		SandboxMode:        true,
		AllowedCommands:    []string{"git", "go"},
		EnableAuditLog:     true,
		EncryptCredentials: true,
	}
	
	sm := NewSecurityManager(config, "test-session")
	
	status := sm.GetSecurityStatus()
	
	// 檢查返回的狀態
	if status["sandbox_enabled"] != true {
		t.Error("安全狀態應該顯示沙箱已啟用")
	}
	
	if status["audit_enabled"] != true {
		t.Error("安全狀態應該顯示審計已啟用")
	}
	
	if status["encryption_enabled"] != true {
		t.Error("安全狀態應該顯示加密已啟用")
	}
	
	if status["allowed_commands"] != 2 {
		t.Errorf("應該顯示 2 個允許的命令，實際: %v", status["allowed_commands"])
	}
}

func TestSecurityManager_NoSecurityFeatures(t *testing.T) {
	// 創建沒有啟用任何安全功能的管理器
	config := SecurityConfig{
		SandboxMode:        false,
		EnableAuditLog:     false,
		EncryptCredentials: false,
	}
	
	sm := NewSecurityManager(config, "test-session")
	
	// 檢查功能狀態
	if sm.IsSandboxEnabled() {
		t.Error("沙箱模式應該被禁用")
	}
	
	if sm.IsAuditEnabled() {
		t.Error("審計日誌應該被禁用")
	}
	
	if sm.IsEncryptionEnabled() {
		t.Error("加密功能應該被禁用")
	}
	
	// 驗證命令（應該總是通過，因為沙箱禁用）
	err := sm.ValidateCommand("rm -rf /")
	if err != nil {
		t.Error("沙箱禁用時，所有命令都應該被允許")
	}
	
	// 憑證加密（應該返回原值）
	testCred := "secret"
	encrypted, err := sm.EncryptCredential(testCred)
	if err != nil {
		t.Fatalf("加密禁用時不應該有錯誤: %v", err)
	}
	if encrypted != testCred {
		t.Error("加密禁用時應該返回原憑證")
	}
}

func TestDefaultSecurityConfig(t *testing.T) {
	config := DefaultSecurityConfig()
	
	// 檢查預設值
	if config.SandboxMode {
		t.Error("預設應該禁用沙箱模式")
	}
	
	if config.EnableAuditLog {
		t.Error("預設應該禁用審計日誌")
	}
	
	if config.EncryptCredentials {
		t.Error("預設應該禁用憑證加密")
	}
	
	if !config.MaskSensitiveInfo {
		t.Error("預設應該啟用敏感資訊遮罩")
	}
	
	if !config.StrictPathChecking {
		t.Error("預設應該啟用嚴格路徑檢查")
	}
}

func TestValidateSecurityConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config-test")
	if err != nil {
		t.Fatalf("創建臨時目錄失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	testCases := []struct {
		name      string
		config    SecurityConfig
		shouldErr bool
	}{
		{
			"有效配置",
			SecurityConfig{
				SandboxMode: true,
				WorkDir:     tempDir,
				EnableAuditLog: true,
				AuditLogDir: filepath.Join(tempDir, "audit"),
			},
			false,
		},
		{
			"沙箱模式但工作目錄不存在",
			SecurityConfig{
				SandboxMode: true,
				WorkDir:     "/nonexistent/directory",
			},
			true,
		},
		{
			"沙箱模式但工作目錄是相對路徑",
			SecurityConfig{
				SandboxMode: true,
				WorkDir:     "relative/path",
			},
			true,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateSecurityConfig(tc.config)
			hasErr := (err != nil)
			if hasErr != tc.shouldErr {
				t.Errorf("期望錯誤 %v，實際錯誤 %v (錯誤: %v)", 
					tc.shouldErr, hasErr, err)
			}
		})
	}
}

func TestSecurityManager_AuditLogPath(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "audit-test")
	if err != nil {
		t.Fatalf("創建臨時目錄失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	auditDir := filepath.Join(tempDir, "audit")
	
	config := SecurityConfig{
		EnableAuditLog: true,
		AuditLogDir:    auditDir,
	}
	
	sm := NewSecurityManager(config, "test-session")
	
	// 檢查審計日誌路徑
	logPath := sm.GetAuditLogPath()
	if logPath == "" {
		t.Error("應該返回有效的審計日誌路徑")
	}
	
	// 檢查路徑是否在指定目錄內
	if !strings.Contains(logPath, auditDir) {
		t.Errorf("審計日誌路徑 '%s' 應該在 '%s' 目錄內", logPath, auditDir)
	}
}

func TestSecurityManager_MaskSensitiveOutput(t *testing.T) {
	config := SecurityConfig{
		MaskSensitiveInfo: true,
	}
	
	sm := NewSecurityManager(config, "test-session")
	
	testInput := "password=secret123\napi_key=sk-1234567890\nusername=john"
	expectedOutput := "password=***MASKED***\napi_key=***MASKED***\nusername=john"
	
	result := sm.MaskSensitiveOutput(testInput)
	if result != expectedOutput {
		t.Errorf("敏感資訊遮罩結果不匹配\n期望: %s\n實際: %s", expectedOutput, result)
	}
	
	// 測試禁用遮罩
	config.MaskSensitiveInfo = false
	sm = NewSecurityManager(config, "test-session")
	
	result = sm.MaskSensitiveOutput(testInput)
	if result != testInput {
		t.Error("禁用遮罩時應該返回原輸出")
	}
}