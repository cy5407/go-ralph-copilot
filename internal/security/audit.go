package security

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// AuditEvent 審計事件類型
type AuditEvent string

const (
	EventCommandExecution AuditEvent = "COMMAND_EXECUTION"
	EventFileAccess       AuditEvent = "FILE_ACCESS"
	EventConfigChange     AuditEvent = "CONFIG_CHANGE"
	EventSecurityViolation AuditEvent = "SECURITY_VIOLATION"
	EventAuthentication   AuditEvent = "AUTHENTICATION"
	EventEncryption       AuditEvent = "ENCRYPTION"
	EventSandboxViolation AuditEvent = "SANDBOX_VIOLATION"
)

// AuditLevel 審計級別
type AuditLevel string

const (
	LevelInfo     AuditLevel = "INFO"
	LevelWarning  AuditLevel = "WARNING"
	LevelError    AuditLevel = "ERROR"
	LevelCritical AuditLevel = "CRITICAL"
)

// AuditEntry 審計日誌條目
type AuditEntry struct {
	Timestamp    time.Time              `json:"timestamp"`
	Event        AuditEvent            `json:"event"`
	Level        AuditLevel            `json:"level"`
	User         string                `json:"user"`
	Session      string                `json:"session,omitempty"`
	Message      string                `json:"message"`
	Details      map[string]interface{} `json:"details,omitempty"`
	Source       string                `json:"source,omitempty"`
	Outcome      string                `json:"outcome"` // SUCCESS, FAILURE, BLOCKED
}

// AuditLogger 審計日誌記錄器
type AuditLogger struct {
	logFile    string
	enabled    bool
	maskSensitive bool
}

// NewAuditLogger 創建新的審計日誌記錄器
func NewAuditLogger(logDir string, enabled bool) *AuditLogger {
	if !enabled {
		return &AuditLogger{enabled: false}
	}
	
	// 確保日誌目錄存在
	if err := os.MkdirAll(logDir, 0700); err != nil {
		// 如果無法創建目錄，禁用審計日誌
		return &AuditLogger{enabled: false}
	}
	
	// 生成日誌文件名（按日期分割）
	logFile := filepath.Join(logDir, fmt.Sprintf("audit_%s.json", 
		time.Now().Format("2006-01-02")))
	
	return &AuditLogger{
		logFile:       logFile,
		enabled:       true,
		maskSensitive: true,
	}
}

// LogCommandExecution 記錄命令執行
func (al *AuditLogger) LogCommandExecution(user, session, command, outcome string, details map[string]interface{}) {
	if !al.enabled {
		return
	}
	
	entry := AuditEntry{
		Timestamp: time.Now().UTC(),
		Event:     EventCommandExecution,
		Level:     LevelInfo,
		User:      user,
		Session:   session,
		Message:   fmt.Sprintf("執行命令: %s", al.maskCommand(command)),
		Details:   al.maskSensitiveData(details),
		Source:    "cli_executor",
		Outcome:   outcome,
	}
	
	al.writeEntry(entry)
}

// LogFileAccess 記錄文件訪問
func (al *AuditLogger) LogFileAccess(user, session, path, operation, outcome string) {
	if !al.enabled {
		return
	}
	
	level := LevelInfo
	if outcome != "SUCCESS" {
		level = LevelWarning
	}
	
	entry := AuditEntry{
		Timestamp: time.Now().UTC(),
		Event:     EventFileAccess,
		Level:     level,
		User:      user,
		Session:   session,
		Message:   fmt.Sprintf("文件%s: %s", operation, path),
		Details: map[string]interface{}{
			"path":      path,
			"operation": operation,
		},
		Source:  "file_system",
		Outcome: outcome,
	}
	
	al.writeEntry(entry)
}

// LogConfigChange 記錄配置變更
func (al *AuditLogger) LogConfigChange(user, session, configKey, oldValue, newValue string) {
	if !al.enabled {
		return
	}
	
	entry := AuditEntry{
		Timestamp: time.Now().UTC(),
		Event:     EventConfigChange,
		Level:     LevelWarning,
		User:      user,
		Session:   session,
		Message:   fmt.Sprintf("配置變更: %s", configKey),
		Details: map[string]interface{}{
			"config_key": configKey,
			"old_value":  al.maskSensitiveValue(configKey, oldValue),
			"new_value":  al.maskSensitiveValue(configKey, newValue),
		},
		Source:  "config_manager",
		Outcome: "SUCCESS",
	}
	
	al.writeEntry(entry)
}

// LogSecurityViolation 記錄安全違規
func (al *AuditLogger) LogSecurityViolation(user, session, violation, details string) {
	if !al.enabled {
		return
	}
	
	entry := AuditEntry{
		Timestamp: time.Now().UTC(),
		Event:     EventSecurityViolation,
		Level:     LevelCritical,
		User:      user,
		Session:   session,
		Message:   fmt.Sprintf("安全違規: %s", violation),
		Details: map[string]interface{}{
			"violation_type": violation,
			"details":       details,
		},
		Source:  "security_manager",
		Outcome: "BLOCKED",
	}
	
	al.writeEntry(entry)
}

// LogAuthentication 記錄認證事件
func (al *AuditLogger) LogAuthentication(user, method, outcome string, details map[string]interface{}) {
	if !al.enabled {
		return
	}
	
	level := LevelInfo
	if outcome != "SUCCESS" {
		level = LevelError
	}
	
	entry := AuditEntry{
		Timestamp: time.Now().UTC(),
		Event:     EventAuthentication,
		Level:     level,
		User:      user,
		Message:   fmt.Sprintf("認證嘗試: %s", method),
		Details:   al.maskSensitiveData(details),
		Source:    "auth_manager",
		Outcome:   outcome,
	}
	
	al.writeEntry(entry)
}

// LogEncryption 記錄加密/解密操作
func (al *AuditLogger) LogEncryption(user, session, operation, target, outcome string) {
	if !al.enabled {
		return
	}
	
	level := LevelInfo
	if outcome != "SUCCESS" {
		level = LevelWarning
	}
	
	entry := AuditEntry{
		Timestamp: time.Now().UTC(),
		Event:     EventEncryption,
		Level:     level,
		User:      user,
		Session:   session,
		Message:   fmt.Sprintf("加密操作: %s %s", operation, target),
		Details: map[string]interface{}{
			"operation": operation,
			"target":    target,
		},
		Source:  "encryption_manager",
		Outcome: outcome,
	}
	
	al.writeEntry(entry)
}

// LogSandboxViolation 記錄沙箱違規
func (al *AuditLogger) LogSandboxViolation(user, session, command, violation string) {
	if !al.enabled {
		return
	}
	
	entry := AuditEntry{
		Timestamp: time.Now().UTC(),
		Event:     EventSandboxViolation,
		Level:     LevelCritical,
		User:      user,
		Session:   session,
		Message:   fmt.Sprintf("沙箱違規: %s", violation),
		Details: map[string]interface{}{
			"command":   al.maskCommand(command),
			"violation": violation,
		},
		Source:  "sandbox_manager",
		Outcome: "BLOCKED",
	}
	
	al.writeEntry(entry)
}

// writeEntry 寫入審計日誌條目
func (al *AuditLogger) writeEntry(entry AuditEntry) {
	if !al.enabled {
		return
	}
	
	// 序列化為 JSON
	jsonData, err := json.Marshal(entry)
	if err != nil {
		// 寫入失敗，嘗試記錄到標準錯誤
		fmt.Fprintf(os.Stderr, "審計日誌序列化失敗: %v\n", err)
		return
	}
	
	// 追加到日誌文件
	file, err := os.OpenFile(al.logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "無法打開審計日誌文件: %v\n", err)
		return
	}
	defer file.Close()
	
	// 寫入 JSON 行
	if _, err := file.WriteString(string(jsonData) + "\n"); err != nil {
		fmt.Fprintf(os.Stderr, "寫入審計日誌失敗: %v\n", err)
	}
}

// maskCommand 遮罩命令中的敏感資訊
func (al *AuditLogger) maskCommand(command string) string {
	if !al.maskSensitive {
		return command
	}
	
	return MaskSensitiveInfo(command)
}

// maskSensitiveValue 根據鍵名遮罩敏感值
func (al *AuditLogger) maskSensitiveValue(key, value string) string {
	if !al.maskSensitive {
		return value
	}
	
	sensitiveKeys := []string{
		"password", "secret", "key", "token", "auth",
		"credential", "api_key", "access_key", "private_key",
	}
	
	keyLower := strings.ToLower(key)
	for _, sensitive := range sensitiveKeys {
		if strings.Contains(keyLower, sensitive) {
			if len(value) > 6 {
				return value[:2] + "***" + value[len(value)-2:]
			} else if len(value) > 0 {
				return "***"
			}
		}
	}
	
	return value
}

// maskSensitiveData 遮罩詳細資訊中的敏感資料
func (al *AuditLogger) maskSensitiveData(details map[string]interface{}) map[string]interface{} {
	if !al.maskSensitive || details == nil {
		return details
	}
	
	masked := make(map[string]interface{})
	
	for key, value := range details {
		if strValue, ok := value.(string); ok {
			masked[key] = al.maskSensitiveValue(key, strValue)
		} else {
			masked[key] = value
		}
	}
	
	return masked
}

// GetAuditLogPath 獲取當前的審計日誌路徑
func (al *AuditLogger) GetAuditLogPath() string {
	return al.logFile
}

// IsEnabled 檢查審計日誌是否啟用
func (al *AuditLogger) IsEnabled() bool {
	return al.enabled
}

// GetCurrentUser 獲取當前使用者名稱
func GetCurrentUser() string {
	if user := os.Getenv("USER"); user != "" {
		return user
	}
	if user := os.Getenv("USERNAME"); user != "" {
		return user
	}
	return "unknown"
}