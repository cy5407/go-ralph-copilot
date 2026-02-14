package security

import (
	"fmt"
	"os"
	"path/filepath"
)

// SecurityManager 統一的安全管理器
type SecurityManager struct {
	encryptionManager *EncryptionManager
	sandboxManager    *SandboxManager
	auditLogger       *AuditLogger
	config            SecurityConfig
	user              string
	session           string
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	// 沙箱模式
	SandboxMode      bool     `json:"sandbox_mode"`
	AllowedCommands  []string `json:"allowed_commands"`
	WorkDir          string   `json:"work_dir"`
	
	// 審計日誌
	EnableAuditLog   bool   `json:"enable_audit_log"`
	AuditLogDir      string `json:"audit_log_dir"`
	
	// 加密
	EncryptCredentials bool   `json:"encrypt_credentials"`
	EncryptionPassword string `json:"encryption_password,omitempty"`
	
	// 其他安全選項
	MaskSensitiveInfo bool `json:"mask_sensitive_info"`
	StrictPathChecking bool `json:"strict_path_checking"`
}

// NewSecurityManager 創建新的安全管理器
func NewSecurityManager(config SecurityConfig, sessionID string) *SecurityManager {
	user := GetCurrentUser()
	
	// 初始化加密管理器
	var encryptionManager *EncryptionManager
	if config.EncryptCredentials {
		password := config.EncryptionPassword
		if password == "" {
			password = GetDefaultPassword()
		}
		encryptionManager = NewEncryptionManager(password)
	}
	
	// 初始化沙箱管理器
	var sandboxManager *SandboxManager
	if config.SandboxMode {
		sandboxManager = NewSandboxManager(config.AllowedCommands, config.WorkDir)
	}
	
	// 初始化審計日誌
	auditLogDir := config.AuditLogDir
	if auditLogDir == "" {
		homeDir, _ := os.UserHomeDir()
		auditLogDir = filepath.Join(homeDir, ".ralph-loop", "audit")
	}
	auditLogger := NewAuditLogger(auditLogDir, config.EnableAuditLog)
	
	return &SecurityManager{
		encryptionManager: encryptionManager,
		sandboxManager:    sandboxManager,
		auditLogger:       auditLogger,
		config:            config,
		user:              user,
		session:           sessionID,
	}
}

// ValidateCommand 驗證命令執行
func (sm *SecurityManager) ValidateCommand(command string) error {
	// 記錄命令驗證嘗試
	sm.auditLogger.LogCommandExecution(sm.user, sm.session, command, "VALIDATING", 
		map[string]interface{}{
			"sandbox_mode": sm.config.SandboxMode,
		})
	
	// 如果啟用沙箱模式，進行額外檢查
	if sm.config.SandboxMode && sm.sandboxManager != nil {
		if err := sm.sandboxManager.ValidateCommandForSandbox(command); err != nil {
			// 記錄沙箱違規
			sm.auditLogger.LogSandboxViolation(sm.user, sm.session, command, err.Error())
			return fmt.Errorf("沙箱安全檢查失敗: %w", err)
		}
	}
	
	// 記錄命令驗證成功
	sm.auditLogger.LogCommandExecution(sm.user, sm.session, command, "VALIDATED", nil)
	
	return nil
}

// ExecuteCommand 安全執行命令（包裝函數）
func (sm *SecurityManager) ExecuteCommand(command string, executor func(string) (string, error)) (string, error) {
	// 預執行驗證
	if err := sm.ValidateCommand(command); err != nil {
		return "", err
	}
	
	// 執行命令
	output, err := executor(command)
	
	// 記錄執行結果
	outcome := "SUCCESS"
	if err != nil {
		outcome = "FAILURE"
	}
	
	details := map[string]interface{}{
		"output_length": len(output),
		"has_error":     err != nil,
	}
	if err != nil {
		details["error"] = err.Error()
	}
	
	sm.auditLogger.LogCommandExecution(sm.user, sm.session, command, outcome, details)
	
	// 如果需要，遮罩輸出中的敏感資訊
	if sm.config.MaskSensitiveInfo {
		output = MaskSensitiveInfo(output)
	}
	
	return output, err
}

// EncryptCredential 加密憑證
func (sm *SecurityManager) EncryptCredential(credential string) (string, error) {
	if sm.encryptionManager == nil {
		return credential, nil // 未啟用加密，直接返回
	}
	
	encrypted, err := sm.encryptionManager.EncryptString(credential)
	
	// 記錄加密操作
	outcome := "SUCCESS"
	if err != nil {
		outcome = "FAILURE"
	}
	sm.auditLogger.LogEncryption(sm.user, sm.session, "ENCRYPT", "credential", outcome)
	
	return encrypted, err
}

// DecryptCredential 解密憑證
func (sm *SecurityManager) DecryptCredential(encryptedCredential string) (string, error) {
	if sm.encryptionManager == nil {
		return encryptedCredential, nil // 未啟用加密，直接返回
	}
	
	// 檢查是否已加密
	if !IsEncrypted(encryptedCredential) {
		return encryptedCredential, nil
	}
	
	decrypted, err := sm.encryptionManager.DecryptString(encryptedCredential)
	
	// 記錄解密操作
	outcome := "SUCCESS"
	if err != nil {
		outcome = "FAILURE"
	}
	sm.auditLogger.LogEncryption(sm.user, sm.session, "DECRYPT", "credential", outcome)
	
	return decrypted, err
}

// ValidateFilePath 驗證文件路徑訪問
func (sm *SecurityManager) ValidateFilePath(path, operation string) error {
	// 如果啟用沙箱模式，檢查路徑訪問
	if sm.config.SandboxMode && sm.sandboxManager != nil {
		if allowed, reason := sm.sandboxManager.IsPathAllowed(path); !allowed {
			sm.auditLogger.LogSecurityViolation(sm.user, sm.session, 
				"UNAUTHORIZED_PATH_ACCESS", reason)
			return fmt.Errorf("路徑訪問被拒絕: %s", reason)
		}
	}
	
	// 記錄文件訪問
	sm.auditLogger.LogFileAccess(sm.user, sm.session, path, operation, "ALLOWED")
	
	return nil
}

// LogConfigChange 記錄配置變更
func (sm *SecurityManager) LogConfigChange(key, oldValue, newValue string) {
	sm.auditLogger.LogConfigChange(sm.user, sm.session, key, oldValue, newValue)
}

// CreateRestrictedEnvironment 創建受限環境
func (sm *SecurityManager) CreateRestrictedEnvironment() map[string]string {
	if sm.config.SandboxMode && sm.sandboxManager != nil {
		return sm.sandboxManager.CreateRestrictedEnvironment()
	}
	return nil
}

// MaskSensitiveOutput 遮罩輸出中的敏感資訊
func (sm *SecurityManager) MaskSensitiveOutput(output string) string {
	if sm.config.MaskSensitiveInfo {
		return MaskSensitiveInfo(output)
	}
	return output
}

// GetAuditLogPath 獲取審計日誌路徑
func (sm *SecurityManager) GetAuditLogPath() string {
	if sm.auditLogger != nil {
		return sm.auditLogger.GetAuditLogPath()
	}
	return ""
}

// IsAuditEnabled 檢查審計是否啟用
func (sm *SecurityManager) IsAuditEnabled() bool {
	return sm.config.EnableAuditLog && sm.auditLogger != nil && sm.auditLogger.IsEnabled()
}

// IsSandboxEnabled 檢查沙箱是否啟用
func (sm *SecurityManager) IsSandboxEnabled() bool {
	return sm.config.SandboxMode && sm.sandboxManager != nil
}

// IsEncryptionEnabled 檢查加密是否啟用
func (sm *SecurityManager) IsEncryptionEnabled() bool {
	return sm.config.EncryptCredentials && sm.encryptionManager != nil
}

// GetSecurityStatus 獲取安全狀態摘要
func (sm *SecurityManager) GetSecurityStatus() map[string]interface{} {
	return map[string]interface{}{
		"sandbox_enabled":    sm.IsSandboxEnabled(),
		"audit_enabled":      sm.IsAuditEnabled(),
		"encryption_enabled": sm.IsEncryptionEnabled(),
		"user":               sm.user,
		"session":            sm.session,
		"audit_log_path":     sm.GetAuditLogPath(),
		"allowed_commands":   len(sm.config.AllowedCommands),
		"work_dir":           sm.config.WorkDir,
	}
}

// DefaultSecurityConfig 返回預設安全配置
func DefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		SandboxMode:        false,
		AllowedCommands:    []string{},
		WorkDir:            "",
		EnableAuditLog:     false,
		AuditLogDir:        "",
		EncryptCredentials: false,
		EncryptionPassword: "",
		MaskSensitiveInfo:  true,
		StrictPathChecking: true,
	}
}

// ValidateSecurityConfig 驗證安全配置
func ValidateSecurityConfig(config SecurityConfig) error {
	// 檢查工作目錄
	if config.SandboxMode && config.WorkDir != "" {
		if !filepath.IsAbs(config.WorkDir) {
			return fmt.Errorf("沙箱模式要求絕對路徑的工作目錄")
		}
		
		if _, err := os.Stat(config.WorkDir); os.IsNotExist(err) {
			return fmt.Errorf("工作目錄不存在: %s", config.WorkDir)
		}
	}
	
	// 檢查審計日誌目錄
	if config.EnableAuditLog && config.AuditLogDir != "" {
		dir := config.AuditLogDir
		if !filepath.IsAbs(dir) {
			absDir, err := filepath.Abs(dir)
			if err != nil {
				return fmt.Errorf("無法解析審計日誌目錄路徑: %w", err)
			}
			dir = absDir
		}
		
		// 嘗試創建目錄以驗證權限
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("無法創建審計日誌目錄: %w", err)
		}
	}
	
	return nil
}