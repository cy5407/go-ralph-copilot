package ghcopilot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// ConfigFile 表示 TOML 配置文件結構
type ConfigFile struct {
	// CLI 相關配置
	CLI CLIConfig `toml:"cli"`
	
	// 上下文與歷史配置
	Context ContextConfig `toml:"context"`
	
	// 熔斷器配置
	CircuitBreaker CircuitBreakerConfig `toml:"circuit_breaker"`
	
	// AI 模型配置
	AI AIConfig `toml:"ai"`
	
	// 輸出配置
	Output OutputConfig `toml:"output"`
	
	// 安全配置
	Security SecurityConfig `toml:"security"`
	
	// 進階功能配置
	Advanced AdvancedConfig `toml:"advanced"`
}

// CLIConfig CLI 執行配置
type CLIConfig struct {
	Timeout    string `toml:"timeout"`    // 如 "60s", "2m"
	MaxRetries int    `toml:"max_retries"`
	WorkDir    string `toml:"work_dir"`
}

// ContextConfig 上下文管理配置
type ContextConfig struct {
	MaxHistorySize    int    `toml:"max_history_size"`
	SaveDir           string `toml:"save_dir"`
	EnablePersistence bool   `toml:"enable_persistence"`
	UseGobFormat      bool   `toml:"use_gob_format"`
}

// CircuitBreakerConfig 熔斷器配置
type CircuitBreakerConfig struct {
	Threshold         int `toml:"threshold"`
	SameErrorThreshold int `toml:"same_error_threshold"`
}

// AIConfig AI 模型配置
type AIConfig struct {
	Model     string `toml:"model"`
	EnableSDK bool   `toml:"enable_sdk"`
	PreferSDK bool   `toml:"prefer_sdk"`
}

// OutputConfig 輸出配置
type OutputConfig struct {
	Silent       bool   `toml:"silent"`
	Verbose      bool   `toml:"verbose"`
	Quiet        bool   `toml:"quiet"`
	Format       string `toml:"format"` // "text", "json", "table"
	UseColors    bool   `toml:"use_colors"`
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	SandboxMode      bool     `toml:"sandbox_mode"`
	AllowedCommands  []string `toml:"allowed_commands"`
	EnableAuditLog   bool     `toml:"enable_audit_log"`
	EncryptCredentials bool   `toml:"encrypt_credentials"`
}

// AdvancedConfig 進階功能配置
type AdvancedConfig struct {
	EnableMetrics   bool   `toml:"enable_metrics"`
	MetricsPort     int    `toml:"metrics_port"`
	EnableWebUI     bool   `toml:"enable_web_ui"`
	WebUIPort       int    `toml:"web_ui_port"`
	PluginDir       string `toml:"plugin_dir"`
}

// LoadConfig 從配置文件載入配置
func LoadConfig(configPath string) (*ClientConfig, error) {
	// 如果文件不存在，返回預設配置
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return DefaultClientConfig(), nil
	}
	
	var configFile ConfigFile
	if _, err := toml.DecodeFile(configPath, &configFile); err != nil {
		return nil, fmt.Errorf("載入配置文件失敗: %w", err)
	}
	
	// 轉換為 ClientConfig
	config, err := convertToClientConfig(&configFile)
	if err != nil {
		return nil, fmt.Errorf("轉換配置失敗: %w", err)
	}
	
	// 應用環境變數覆蓋
	if err := applyEnvironmentVariables(config); err != nil {
		return nil, fmt.Errorf("應用環境變數失敗: %w", err)
	}
	
	// 驗證配置
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("配置驗證失敗: %w", err)
	}
	
	return config, nil
}

// convertToClientConfig 將 ConfigFile 轉換為 ClientConfig
func convertToClientConfig(configFile *ConfigFile) (*ClientConfig, error) {
	config := DefaultClientConfig()
	
	// CLI 配置
	if configFile.CLI.Timeout != "" {
		duration, err := time.ParseDuration(configFile.CLI.Timeout)
		if err != nil {
			return nil, fmt.Errorf("無效的超時設定 '%s': %w", configFile.CLI.Timeout, err)
		}
		config.CLITimeout = duration
	}
	
	if configFile.CLI.MaxRetries > 0 {
		config.CLIMaxRetries = configFile.CLI.MaxRetries
	}
	
	if configFile.CLI.WorkDir != "" {
		config.WorkDir = configFile.CLI.WorkDir
	}
	
	// 上下文配置
	if configFile.Context.MaxHistorySize > 0 {
		config.MaxHistorySize = configFile.Context.MaxHistorySize
	}
	
	if configFile.Context.SaveDir != "" {
		config.SaveDir = configFile.Context.SaveDir
	}
	
	config.EnablePersistence = configFile.Context.EnablePersistence
	config.UseGobFormat = configFile.Context.UseGobFormat
	
	// 熔斷器配置
	if configFile.CircuitBreaker.Threshold > 0 {
		config.CircuitBreakerThreshold = configFile.CircuitBreaker.Threshold
	}
	
	if configFile.CircuitBreaker.SameErrorThreshold > 0 {
		config.SameErrorThreshold = configFile.CircuitBreaker.SameErrorThreshold
	}
	
	// AI 配置
	if configFile.AI.Model != "" {
		config.Model = configFile.AI.Model
	}
	
	config.EnableSDK = configFile.AI.EnableSDK
	config.PreferSDK = configFile.AI.PreferSDK
	
	// 輸出配置
	config.Silent = configFile.Output.Silent
	config.Verbose = configFile.Output.Verbose
	config.Quiet = configFile.Output.Quiet
	
	// 安全配置
	config.Security.SandboxMode = configFile.Security.SandboxMode
	config.Security.AllowedCommands = configFile.Security.AllowedCommands
	config.Security.EnableAuditLog = configFile.Security.EnableAuditLog
	config.Security.EncryptCredentials = configFile.Security.EncryptCredentials
	if configFile.Security.AllowedCommands == nil {
		config.Security.AllowedCommands = []string{} // 確保不是 nil
	}
	
	return config, nil
}

// applyEnvironmentVariables 應用環境變數覆蓋
func applyEnvironmentVariables(config *ClientConfig) error {
	// CLI 配置
	if timeout := os.Getenv("RALPH_CLI_TIMEOUT"); timeout != "" {
		duration, err := time.ParseDuration(timeout)
		if err != nil {
			return fmt.Errorf("無效的環境變數 RALPH_CLI_TIMEOUT '%s': %w", timeout, err)
		}
		config.CLITimeout = duration
	}
	
	if maxRetries := os.Getenv("RALPH_CLI_MAX_RETRIES"); maxRetries != "" {
		retries, err := strconv.Atoi(maxRetries)
		if err != nil || retries < 0 {
			return fmt.Errorf("無效的環境變數 RALPH_CLI_MAX_RETRIES '%s'", maxRetries)
		}
		config.CLIMaxRetries = retries
	}
	
	if workDir := os.Getenv("RALPH_WORK_DIR"); workDir != "" {
		config.WorkDir = workDir
	}
	
	// 熔斷器配置
	if threshold := os.Getenv("RALPH_CIRCUIT_BREAKER_THRESHOLD"); threshold != "" {
		t, err := strconv.Atoi(threshold)
		if err != nil || t < 1 {
			return fmt.Errorf("無效的環境變數 RALPH_CIRCUIT_BREAKER_THRESHOLD '%s'", threshold)
		}
		config.CircuitBreakerThreshold = t
	}
	
	if sameErrorThreshold := os.Getenv("RALPH_SAME_ERROR_THRESHOLD"); sameErrorThreshold != "" {
		t, err := strconv.Atoi(sameErrorThreshold)
		if err != nil || t < 1 {
			return fmt.Errorf("無效的環境變數 RALPH_SAME_ERROR_THRESHOLD '%s'", sameErrorThreshold)
		}
		config.SameErrorThreshold = t
	}
	
	// AI 配置
	if model := os.Getenv("RALPH_MODEL"); model != "" {
		config.Model = model
	}
	
	if enableSDK := os.Getenv("RALPH_ENABLE_SDK"); enableSDK != "" {
		config.EnableSDK = strings.ToLower(enableSDK) == "true"
	}
	
	if preferSDK := os.Getenv("RALPH_PREFER_SDK"); preferSDK != "" {
		config.PreferSDK = strings.ToLower(preferSDK) == "true"
	}
	
	// 輸出配置
	if silent := os.Getenv("RALPH_SILENT"); silent != "" {
		config.Silent = strings.ToLower(silent) == "true"
	}
	
	if verbose := os.Getenv("RALPH_VERBOSE"); verbose != "" {
		config.Verbose = strings.ToLower(verbose) == "true"
	}
	
	if quiet := os.Getenv("RALPH_QUIET"); quiet != "" {
		config.Quiet = strings.ToLower(quiet) == "true"
	}
	
	// 持久化配置
	if saveDir := os.Getenv("RALPH_SAVE_DIR"); saveDir != "" {
		config.SaveDir = saveDir
	}
	
	if enablePersistence := os.Getenv("RALPH_ENABLE_PERSISTENCE"); enablePersistence != "" {
		config.EnablePersistence = strings.ToLower(enablePersistence) == "true"
	}
	
	return nil
}

// validateConfig 驗證配置合法性
func validateConfig(config *ClientConfig) error {
	// 驗證超時設定
	if config.CLITimeout < time.Second {
		return fmt.Errorf("CLI 超時設定過小，最小值為 1 秒")
	}
	
	if config.CLITimeout > 10*time.Minute {
		return fmt.Errorf("CLI 超時設定過大，最大值為 10 分鐘")
	}
	
	// 驗證重試次數
	if config.CLIMaxRetries < 0 || config.CLIMaxRetries > 10 {
		return fmt.Errorf("重試次數必須在 0-10 之間")
	}
	
	// 驗證歷史大小
	if config.MaxHistorySize < 1 || config.MaxHistorySize > 1000 {
		return fmt.Errorf("歷史記錄大小必須在 1-1000 之間")
	}
	
	// 驗證熔斷器閾值
	if config.CircuitBreakerThreshold < 1 || config.CircuitBreakerThreshold > 50 {
		return fmt.Errorf("熔斷器閾值必須在 1-50 之間")
	}
	
	if config.SameErrorThreshold < 1 || config.SameErrorThreshold > 100 {
		return fmt.Errorf("相同錯誤閾值必須在 1-100 之間")
	}
	
	// 驗證工作目錄
	if config.WorkDir != "" {
		if !filepath.IsAbs(config.WorkDir) {
			// 如果是相對路徑，轉換為絕對路徑
			absPath, err := filepath.Abs(config.WorkDir)
			if err != nil {
				return fmt.Errorf("無法解析工作目錄路徑: %w", err)
			}
			config.WorkDir = absPath
		}
		
		if _, err := os.Stat(config.WorkDir); os.IsNotExist(err) {
			return fmt.Errorf("工作目錄不存在: %s", config.WorkDir)
		}
	}
	
	// 驗證儲存目錄
	if config.SaveDir != "" {
		dir := filepath.Dir(config.SaveDir)
		if dir != "." {
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				// 嘗試建立目錄
				if err := os.MkdirAll(dir, 0755); err != nil {
					return fmt.Errorf("無法建立儲存目錄: %w", err)
				}
			}
		}
	}
	
	return nil
}

// SaveConfig 儲存配置到文件
func SaveConfig(config *ClientConfig, configPath string) error {
	configFile := &ConfigFile{
		CLI: CLIConfig{
			Timeout:    config.CLITimeout.String(),
			MaxRetries: config.CLIMaxRetries,
			WorkDir:    config.WorkDir,
		},
		Context: ContextConfig{
			MaxHistorySize:    config.MaxHistorySize,
			SaveDir:           config.SaveDir,
			EnablePersistence: config.EnablePersistence,
			UseGobFormat:      config.UseGobFormat,
		},
		CircuitBreaker: CircuitBreakerConfig{
			Threshold:          config.CircuitBreakerThreshold,
			SameErrorThreshold: config.SameErrorThreshold,
		},
		AI: AIConfig{
			Model:     config.Model,
			EnableSDK: config.EnableSDK,
			PreferSDK: config.PreferSDK,
		},
		Output: OutputConfig{
			Silent:  config.Silent,
			Verbose: config.Verbose,
			Quiet:   config.Quiet,
		},
		Security: SecurityConfig{
			SandboxMode:        config.Security.SandboxMode,
			AllowedCommands:    config.Security.AllowedCommands,
			EnableAuditLog:     config.Security.EnableAuditLog,
			EncryptCredentials: config.Security.EncryptCredentials,
		},
		Advanced: AdvancedConfig{
			EnableMetrics: false, // 將來實作
			MetricsPort:   8080,  // 將來實作
			EnableWebUI:   false, // 將來實作
			WebUIPort:     3000,  // 將來實作
			PluginDir:     "plugins", // 將來實作
		},
	}
	
	// 確保目錄存在
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("建立配置目錄失敗: %w", err)
	}
	
	// 建立文件
	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("建立配置文件失敗: %w", err)
	}
	defer file.Close()
	
	// 寫入 TOML
	encoder := toml.NewEncoder(file)
	encoder.Indent = "  "
	if err := encoder.Encode(configFile); err != nil {
		return fmt.Errorf("寫入配置文件失敗: %w", err)
	}
	
	return nil
}

// GetDefaultConfigPath 取得預設配置文件路徑
func GetDefaultConfigPath() string {
	// 嘗試以下順序：
	// 1. 當前目錄的 ralph-loop.toml
	// 2. 使用者 HOME 目錄的 .ralph-loop/config.toml
	
	// 首先檢查當前目錄
	currentDir := "ralph-loop.toml"
	if _, err := os.Stat(currentDir); err == nil {
		return currentDir
	}
	
	// 檢查 HOME 目錄
	homeDir, err := os.UserHomeDir()
	if err == nil {
		configPath := filepath.Join(homeDir, ".ralph-loop", "config.toml")
		return configPath
	}
	
	// 預設返回當前目錄
	return currentDir
}

// ValidateConfigPublic 公開的配置驗證函數
func ValidateConfigPublic(config *ClientConfig) error {
	return validateConfig(config)
}

// FormatConfigAsJSON 將配置格式化為 JSON
func FormatConfigAsJSON(config *ClientConfig) (string, error) {
	configFile := &ConfigFile{
		CLI: CLIConfig{
			Timeout:    config.CLITimeout.String(),
			MaxRetries: config.CLIMaxRetries,
			WorkDir:    config.WorkDir,
		},
		Context: ContextConfig{
			MaxHistorySize:    config.MaxHistorySize,
			SaveDir:           config.SaveDir,
			EnablePersistence: config.EnablePersistence,
			UseGobFormat:      config.UseGobFormat,
		},
		CircuitBreaker: CircuitBreakerConfig{
			Threshold:          config.CircuitBreakerThreshold,
			SameErrorThreshold: config.SameErrorThreshold,
		},
		AI: AIConfig{
			Model:     config.Model,
			EnableSDK: config.EnableSDK,
			PreferSDK: config.PreferSDK,
		},
		Output: OutputConfig{
			Silent:  config.Silent,
			Verbose: config.Verbose,
			Quiet:   config.Quiet,
		},
		Security: SecurityConfig{
			SandboxMode:        config.Security.SandboxMode,
			AllowedCommands:    config.Security.AllowedCommands,
			EnableAuditLog:     config.Security.EnableAuditLog,
			EncryptCredentials: config.Security.EncryptCredentials,
		},
		Advanced: AdvancedConfig{
			EnableMetrics: false, // 將來實作
			MetricsPort:   8080,  // 將來實作
			EnableWebUI:   false, // 將來實作
			WebUIPort:     3000,  // 將來實作
			PluginDir:     "plugins", // 將來實作
		},
	}
	
	jsonBytes, err := json.MarshalIndent(configFile, "", "  ")
	if err != nil {
		return "", fmt.Errorf("JSON 編碼失敗: %w", err)
	}
	
	return string(jsonBytes), nil
}

// GenerateDefaultConfigFile 生成預設配置文件
func GenerateDefaultConfigFile(configPath string) error {
	config := DefaultClientConfig()
	return SaveConfig(config, configPath)
}