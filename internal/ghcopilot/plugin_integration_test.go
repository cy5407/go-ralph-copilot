package ghcopilot

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestPluginSystemIntegration 測試插件系統與 RalphLoopClient 的整合
func TestPluginSystemIntegration(t *testing.T) {
	// 創建臨時插件目錄
	pluginDir, err := os.MkdirTemp("", "ralph-loop-plugins-")
	if err != nil {
		t.Fatalf("Failed to create temp plugin dir: %v", err)
	}
	defer os.RemoveAll(pluginDir)

	// 創建客戶端配置，啟用插件系統
	config := DefaultClientConfig()
	config.EnablePluginSystem = true
	config.PluginDir = pluginDir
	config.AutoLoadPlugins = false // 手動控制插件載入

	// 創建客戶端
	client := NewRalphLoopClientWithConfig(config)
	defer client.Close()

	// 測試插件系統初始化
	t.Run("Plugin System Initialization", func(t *testing.T) {
		status := client.GetPluginStatus()
		if !status["enabled"].(bool) {
			t.Error("Plugin system should be enabled")
		}
		if status["plugin_count"].(int) != 0 {
			t.Error("Should start with no plugins loaded")
		}
	})

	// 測試插件狀態查詢
	t.Run("Plugin Status", func(t *testing.T) {
		plugins := client.ListPlugins()
		if len(plugins) != 0 {
			t.Error("Should have no plugins initially")
		}

		// 嘗試獲取不存在的插件
		_, err := client.GetPlugin("nonexistent")
		if err == nil {
			t.Error("Should return error for nonexistent plugin")
		}
	})

	// 測試插件系統配置
	t.Run("Plugin Configuration", func(t *testing.T) {
		// 測試設定偏好插件（不存在的插件應該返回錯誤）
		err := client.SetPreferredPlugin("nonexistent-plugin")
		if err == nil {
			t.Error("Should return error when setting nonexistent preferred plugin")
		}

		// 測試獲取偏好插件
		preferred := client.GetPreferredPlugin()
		if preferred != "" {
			t.Error("Should have empty preferred plugin initially")
		}
	})

	// 測試插件自動載入功能
	t.Run("Plugin Auto Load", func(t *testing.T) {
		// 禁用自動載入
		err := client.DisablePluginAutoLoad()
		if err != nil {
			t.Errorf("Failed to disable plugin auto load: %v", err)
		}

		// 嘗試啟用自動載入（應該會掃描空目錄）
		err = client.EnablePluginAutoLoad()
		if err != nil {
			t.Errorf("Failed to enable plugin auto load: %v", err)
		}
	})
}

// TestPluginExecutionMode 測試插件執行模式
func TestPluginExecutionMode(t *testing.T) {
	// 創建模擬插件系統的客戶端
	config := DefaultClientConfig()
	config.EnablePluginSystem = false // 先禁用，然後手動測試

	client := NewRalphLoopClientWithConfig(config)
	defer client.Close()

	// 測試沒有插件系統時的行為
	t.Run("No Plugin System", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// 嘗試使用插件執行（應該失敗）
		_, err := client.executeWithPlugin(ctx, "test-plugin", "test prompt")
		if err == nil {
			t.Error("Should fail when plugin system is not initialized")
		}
		if err.Error() != "plugin manager not initialized" {
			t.Errorf("Expected 'plugin manager not initialized', got: %v", err)
		}
	})
}

// TestExecutionModeSelector 測試執行模式選擇器的插件支援
func TestExecutionModeSelector(t *testing.T) {
	selector := NewExecutionModeSelector()

	// 測試插件模式設定
	t.Run("Plugin Mode Settings", func(t *testing.T) {
		// 初始狀態
		if selector.IsPluginAvailable() {
			t.Error("Plugin should not be available initially")
		}

		// 設定插件可用性
		selector.SetPluginAvailable(true)
		if !selector.IsPluginAvailable() {
			t.Error("Plugin should be available after setting")
		}

		// 設定偏好插件
		testPluginName := "test-executor"
		selector.SetPreferredPlugin(testPluginName)
		if selector.GetPreferredPlugin() != testPluginName {
			t.Errorf("Expected preferred plugin '%s', got '%s'", testPluginName, selector.GetPreferredPlugin())
		}
	})

	// 測試插件模式選擇
	t.Run("Plugin Mode Selection", func(t *testing.T) {
		// 設定插件可用且有偏好插件
		selector.SetPluginAvailable(true)
		selector.SetPreferredPlugin("preferred-executor")

		// 創建測試任務
		task := NewTask("test", "test prompt")
		
		// 選擇模式（應該優先選擇插件）
		mode := selector.Choose(task)
		if mode != ModePlugin {
			t.Errorf("Expected ModePlugin, got %v", mode)
		}
	})

	// 測試插件模式降級
	t.Run("Plugin Mode Fallback", func(t *testing.T) {
		// 設定插件不可用，但 SDK 和 CLI 可用
		selector.SetPluginAvailable(false)
		selector.SetSDKAvailable(true)
		selector.SetCLIAvailable(true)

		// 請求插件模式，但應該降級
		mode := selector.validateAndFallback(ModePlugin, true, true, false, true)
		if mode == ModePlugin {
			t.Error("Should fallback from plugin mode when not available")
		}
	})
}

// TestPluginSystemMetrics 測試插件系統指標
func TestPluginSystemMetrics(t *testing.T) {
	selector := NewExecutionModeSelector()
	
	// 設定插件可用
	selector.SetPluginAvailable(true)
	
	// 模擬插件選擇
	selector.recordSelection(ModePlugin)
	selector.recordSelection(ModePlugin)
	selector.recordSelection(ModeCLI)

	// 檢查指標
	metrics := selector.GetMetrics()
	if metrics.PluginSelections != 2 {
		t.Errorf("Expected 2 plugin selections, got %d", metrics.PluginSelections)
	}
	if metrics.CLISelections != 1 {
		t.Errorf("Expected 1 CLI selection, got %d", metrics.CLISelections)
	}
	if metrics.TotalSelections != 3 {
		t.Errorf("Expected 3 total selections, got %d", metrics.TotalSelections)
	}

	// 重置指標
	selector.ResetMetrics()
	metrics = selector.GetMetrics()
	if metrics.PluginSelections != 0 {
		t.Error("Plugin selections should be reset to 0")
	}
}

// TestPluginSystemConfiguration 測試插件系統配置
func TestPluginSystemConfiguration(t *testing.T) {
	// 測試預設配置
	t.Run("Default Configuration", func(t *testing.T) {
		config := DefaultClientConfig()
		if config.EnablePluginSystem {
			t.Error("Plugin system should be disabled by default")
		}
		if config.AutoLoadPlugins {
			t.Error("Auto load should be disabled by default")
		}
		if config.PluginDir != "./plugins" {
			t.Errorf("Expected default plugin dir './plugins', got '%s'", config.PluginDir)
		}
	})

	// 測試 ClientBuilder 插件配置
	t.Run("ClientBuilder Plugin Config", func(t *testing.T) {
		builder := NewClientBuilder()
		
		// 測試設定插件目錄
		customPluginDir := "/custom/plugins"
		config := builder.config
		config.EnablePluginSystem = true
		config.PluginDir = customPluginDir
		config.AutoLoadPlugins = true
		config.PreferredExecutor = "custom-executor"
		
		client := builder.Build()
		defer client.Close()
		
		if client.config.PluginDir != customPluginDir {
			t.Errorf("Expected plugin dir '%s', got '%s'", customPluginDir, client.config.PluginDir)
		}
	})
}

// MockExecutorPlugin 用於測試的模擬執行器插件
type MockExecutorPlugin struct {
	metadata     PluginMetadata
	capabilities []string
}

// GetMetadata 實作 Plugin 介面
func (m *MockExecutorPlugin) GetMetadata() PluginMetadata {
	return m.metadata
}

// GetType 實作 Plugin 介面
func (m *MockExecutorPlugin) GetType() string {
	return "executor"
}

// GetCapabilities 實作 Plugin 介面
func (m *MockExecutorPlugin) GetCapabilities() []string {
	return m.capabilities
}

// Start 實作 Plugin 介面
func (m *MockExecutorPlugin) Start() error {
	return nil
}

// Stop 實作 Plugin 介面
func (m *MockExecutorPlugin) Stop() error {
	return nil
}

// IsHealthy 實作 Plugin 介面
func (m *MockExecutorPlugin) IsHealthy() bool {
	return true
}

// Execute 實作 ExecutorPlugin 介面
func (m *MockExecutorPlugin) Execute(ctx context.Context, prompt string) (string, error) {
	return "Mock plugin response for: " + prompt, nil
}

// TestMockPluginExecution 測試模擬插件執行
func TestMockPluginExecution(t *testing.T) {
	// 創建模擬插件
	mockPlugin := &MockExecutorPlugin{
		metadata: PluginMetadata{
			Name:        "mock-executor",
			Version:     "1.0.0",
			Author:      "Test Author",
			Description: "Mock executor plugin for testing",
		},
		capabilities: []string{"execute", "completion"},
	}

	// 測試插件基本功能
	t.Run("Mock Plugin Basic Functions", func(t *testing.T) {
		if mockPlugin.GetType() != "executor" {
			t.Error("Mock plugin should be executor type")
		}
		
		if !mockPlugin.IsHealthy() {
			t.Error("Mock plugin should be healthy")
		}
		
		err := mockPlugin.Start()
		if err != nil {
			t.Errorf("Mock plugin start failed: %v", err)
		}
		
		err = mockPlugin.Stop()
		if err != nil {
			t.Errorf("Mock plugin stop failed: %v", err)
		}
	})

	// 測試插件執行
	t.Run("Mock Plugin Execution", func(t *testing.T) {
		ctx := context.Background()
		prompt := "test prompt"
		
		result, err := mockPlugin.Execute(ctx, prompt)
		if err != nil {
			t.Errorf("Mock plugin execution failed: %v", err)
		}
		
		expected := "Mock plugin response for: " + prompt
		if result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	})
}