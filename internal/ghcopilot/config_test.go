package ghcopilot

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	// 測試載入不存在的配置文件（應返回預設配置）
	t.Run("不存在的配置文件", func(t *testing.T) {
		config, err := LoadConfig("nonexistent.toml")
		if err != nil {
			t.Fatalf("載入不存在的配置文件失敗: %v", err)
		}
		
		defaultConfig := DefaultClientConfig()
		if config.CLITimeout != defaultConfig.CLITimeout {
			t.Errorf("超時設定不匹配: got %v, want %v", config.CLITimeout, defaultConfig.CLITimeout)
		}
	})
	
	// 測試載入有效的配置文件
	t.Run("有效的配置文件", func(t *testing.T) {
		// 建立臨時配置文件
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "test-config.toml")
		
		configContent := `
[cli]
timeout = "120s"
max_retries = 5

[context]
max_history_size = 50
save_dir = "./test-saves"
enable_persistence = true

[circuit_breaker]
threshold = 5
same_error_threshold = 10

[ai]
model = "test-model"
enable_sdk = true
prefer_sdk = false

[output]
silent = true
verbose = false
quiet = true
`
		
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("建立測試配置文件失敗: %v", err)
		}
		
		config, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("載入配置文件失敗: %v", err)
		}
		
		// 驗證配置值
		if config.CLITimeout != 120*time.Second {
			t.Errorf("CLI 超時設定錯誤: got %v, want %v", config.CLITimeout, 120*time.Second)
		}
		
		if config.CLIMaxRetries != 5 {
			t.Errorf("最大重試次數錯誤: got %d, want %d", config.CLIMaxRetries, 5)
		}
		
		if config.WorkDir != "" {
			t.Errorf("工作目錄錯誤: got %s, want empty", config.WorkDir)
		}
		
		if config.MaxHistorySize != 50 {
			t.Errorf("最大歷史大小錯誤: got %d, want %d", config.MaxHistorySize, 50)
		}
		
		if config.CircuitBreakerThreshold != 5 {
			t.Errorf("熔斷器閾值錯誤: got %d, want %d", config.CircuitBreakerThreshold, 5)
		}
		
		if config.Model != "test-model" {
			t.Errorf("模型設定錯誤: got %s, want %s", config.Model, "test-model")
		}
		
		if !config.Silent {
			t.Errorf("靜默模式應該為 true")
		}
	})
}

func TestApplyEnvironmentVariables(t *testing.T) {
	// 保存原始環境變數
	originalVars := map[string]string{
		"RALPH_CLI_TIMEOUT":                os.Getenv("RALPH_CLI_TIMEOUT"),
		"RALPH_CLI_MAX_RETRIES":           os.Getenv("RALPH_CLI_MAX_RETRIES"),
		"RALPH_MODEL":                     os.Getenv("RALPH_MODEL"),
		"RALPH_CIRCUIT_BREAKER_THRESHOLD": os.Getenv("RALPH_CIRCUIT_BREAKER_THRESHOLD"),
	}
	
	// 清理函數
	defer func() {
		for key, value := range originalVars {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()
	
	// 設定測試環境變數
	os.Setenv("RALPH_CLI_TIMEOUT", "90s")
	os.Setenv("RALPH_CLI_MAX_RETRIES", "7")
	os.Setenv("RALPH_MODEL", "env-model")
	os.Setenv("RALPH_CIRCUIT_BREAKER_THRESHOLD", "8")
	
	config := DefaultClientConfig()
	if err := applyEnvironmentVariables(config); err != nil {
		t.Fatalf("應用環境變數失敗: %v", err)
	}
	
	// 驗證環境變數已應用
	if config.CLITimeout != 90*time.Second {
		t.Errorf("環境變數 CLI 超時設定錯誤: got %v, want %v", config.CLITimeout, 90*time.Second)
	}
	
	if config.CLIMaxRetries != 7 {
		t.Errorf("環境變數最大重試次數錯誤: got %d, want %d", config.CLIMaxRetries, 7)
	}
	
	if config.Model != "env-model" {
		t.Errorf("環境變數模型設定錯誤: got %s, want %s", config.Model, "env-model")
	}
	
	if config.CircuitBreakerThreshold != 8 {
		t.Errorf("環境變數熔斷器閾值錯誤: got %d, want %d", config.CircuitBreakerThreshold, 8)
	}
}

func TestValidateConfig(t *testing.T) {
	testCases := []struct {
		name    string
		setup   func(*ClientConfig)
		wantErr bool
	}{
		{
			name: "有效配置",
			setup: func(config *ClientConfig) {
				// 使用預設配置，應該是有效的
			},
			wantErr: false,
		},
		{
			name: "超時設定過小",
			setup: func(config *ClientConfig) {
				config.CLITimeout = 500 * time.Millisecond
			},
			wantErr: true,
		},
		{
			name: "超時設定過大",
			setup: func(config *ClientConfig) {
				config.CLITimeout = 15 * time.Minute
			},
			wantErr: true,
		},
		{
			name: "重試次數過大",
			setup: func(config *ClientConfig) {
				config.CLIMaxRetries = 15
			},
			wantErr: true,
		},
		{
			name: "重試次數為負數",
			setup: func(config *ClientConfig) {
				config.CLIMaxRetries = -1
			},
			wantErr: true,
		},
		{
			name: "歷史記錄大小過大",
			setup: func(config *ClientConfig) {
				config.MaxHistorySize = 2000
			},
			wantErr: true,
		},
		{
			name: "熔斷器閾值過小",
			setup: func(config *ClientConfig) {
				config.CircuitBreakerThreshold = 0
			},
			wantErr: true,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := DefaultClientConfig()
			tc.setup(config)
			
			err := validateConfig(config)
			if tc.wantErr && err == nil {
				t.Errorf("期望錯誤但沒有發生錯誤")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("不期望錯誤但發生了錯誤: %v", err)
			}
		})
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	// 建立臨時目錄
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.toml")
	
	// 建立自訂配置
	originalConfig := DefaultClientConfig()
	originalConfig.CLITimeout = 75 * time.Second
	originalConfig.CLIMaxRetries = 4
	originalConfig.Model = "test-save-model"
	originalConfig.MaxHistorySize = 75
	originalConfig.Silent = true
	
	// 儲存配置
	if err := SaveConfig(originalConfig, configPath); err != nil {
		t.Fatalf("儲存配置失敗: %v", err)
	}
	
	// 載入配置
	loadedConfig, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("載入配置失敗: %v", err)
	}
	
	// 比較配置值
	if loadedConfig.CLITimeout != originalConfig.CLITimeout {
		t.Errorf("CLI 超時設定不匹配: got %v, want %v", loadedConfig.CLITimeout, originalConfig.CLITimeout)
	}
	
	if loadedConfig.CLIMaxRetries != originalConfig.CLIMaxRetries {
		t.Errorf("最大重試次數不匹配: got %d, want %d", loadedConfig.CLIMaxRetries, originalConfig.CLIMaxRetries)
	}
	
	if loadedConfig.Model != originalConfig.Model {
		t.Errorf("模型設定不匹配: got %s, want %s", loadedConfig.Model, originalConfig.Model)
	}
	
	if loadedConfig.MaxHistorySize != originalConfig.MaxHistorySize {
		t.Errorf("最大歷史大小不匹配: got %d, want %d", loadedConfig.MaxHistorySize, originalConfig.MaxHistorySize)
	}
	
	if loadedConfig.Silent != originalConfig.Silent {
		t.Errorf("靜默模式不匹配: got %v, want %v", loadedConfig.Silent, originalConfig.Silent)
	}
}

func TestGetDefaultConfigPath(t *testing.T) {
	configPath := GetDefaultConfigPath()
	if configPath == "" {
		t.Errorf("預設配置路徑不應該為空")
	}
	
	// 應該是絕對路徑或相對路徑
	if !filepath.IsAbs(configPath) && !filepath.IsLocal(configPath) {
		t.Errorf("預設配置路徑格式錯誤: %s", configPath)
	}
}

func TestGenerateDefaultConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "generated-config.toml")
	
	// 生成預設配置文件
	if err := GenerateDefaultConfigFile(configPath); err != nil {
		t.Fatalf("生成預設配置文件失敗: %v", err)
	}
	
	// 檢查文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("配置文件未生成")
	}
	
	// 嘗試載入生成的配置
	loadedConfig, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("載入生成的配置文件失敗: %v", err)
	}
	
	// 應該與預設配置相同
	defaultConfig := DefaultClientConfig()
	if loadedConfig.CLITimeout != defaultConfig.CLITimeout {
		t.Errorf("生成的配置與預設配置不匹配")
	}
}

func TestInvalidEnvironmentVariables(t *testing.T) {
	// 保存原始環境變數
	originalTimeout := os.Getenv("RALPH_CLI_TIMEOUT")
	defer func() {
		if originalTimeout == "" {
			os.Unsetenv("RALPH_CLI_TIMEOUT")
		} else {
			os.Setenv("RALPH_CLI_TIMEOUT", originalTimeout)
		}
	}()
	
	testCases := []struct {
		name    string
		envVar  string
		envVal  string
		wantErr bool
	}{
		{
			name:    "無效超時格式",
			envVar:  "RALPH_CLI_TIMEOUT",
			envVal:  "invalid-duration",
			wantErr: true,
		},
		{
			name:    "有效超時格式",
			envVar:  "RALPH_CLI_TIMEOUT",
			envVal:  "60s",
			wantErr: false,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			os.Setenv(tc.envVar, tc.envVal)
			
			config := DefaultClientConfig()
			err := applyEnvironmentVariables(config)
			
			if tc.wantErr && err == nil {
				t.Errorf("期望錯誤但沒有發生錯誤")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("不期望錯誤但發生了錯誤: %v", err)
			}
		})
	}
}