package ghcopilot

import (
	"context"
	"errors"
	"fmt"
	"plugin"
	"sync"
	"time"
)

// Plugin 插件介面
//
// 所有插件都必須實作這個介面才能被 Ralph Loop 系統載入和使用。
// 插件可以是執行器、處理器、或其他擴展功能。
type Plugin interface {
	// GetMetadata 返回插件的元數據信息
	GetMetadata() *PluginMetadata
	
	// Initialize 初始化插件
	// config 參數包含插件所需的配置信息
	Initialize(config map[string]interface{}) error
	
	// IsHealthy 檢查插件是否健康
	IsHealthy() bool
	
	// Close 關閉插件並清理資源
	Close() error
}

// ExecutorPlugin 執行器插件介面
//
// 擴展 Plugin 介面，專門用於實作自定義 AI 執行器
// 例如：OpenAI GPT、Anthropic Claude、Google Gemini 等
type ExecutorPlugin interface {
	Plugin
	
	// Execute 執行 AI 請求
	Execute(ctx context.Context, prompt string, options PluginExecutorOptions) (*ExecutionResponse, error)
	
	// GetCapabilities 獲取執行器能力
	GetCapabilities() *ExecutorCapabilities
	
	// SetModel 設置使用的模型
	SetModel(model string) error
	
	// GetAvailableModels 獲取可用的模型列表
	GetAvailableModels() []string
}

// ProcessorPlugin 處理器插件介面
//
// 用於實作自定義的輸出處理、分析或轉換邏輯
type ProcessorPlugin interface {
	Plugin
	
	// Process 處理輸入數據
	Process(ctx context.Context, input interface{}, options ProcessorOptions) (interface{}, error)
	
	// GetSupportedFormats 獲取支持的輸入格式
	GetSupportedFormats() []string
}

// PluginMetadata 插件元數據
type PluginMetadata struct {
	Name          string            `json:"name"`           // 插件名稱
	Version       string            `json:"version"`        // 插件版本
	Type          PluginType        `json:"type"`           // 插件類型
	Author        string            `json:"author"`         // 作者
	Description   string            `json:"description"`    // 描述
	Website       string            `json:"website"`        // 官方網站
	License       string            `json:"license"`        // 許可證
	Dependencies  []string          `json:"dependencies"`   // 依賴項
	Configuration map[string]string `json:"configuration"`  // 配置說明
	CreatedAt     time.Time         `json:"created_at"`     // 創建時間
}

// PluginType 插件類型
type PluginType string

const (
	PluginTypeExecutor  PluginType = "executor"   // 執行器插件
	PluginTypeProcessor PluginType = "processor"  // 處理器插件
	PluginTypeAnalyzer  PluginType = "analyzer"   // 分析器插件
	PluginTypeFormatter PluginType = "formatter"  // 格式化器插件
	PluginTypeExtension PluginType = "extension"  // 擴展插件
)

// PluginExecutorOptions 插件執行器選項
type PluginExecutorOptions struct {
	Model       string                 `json:"model"`       // 使用的模型
	Temperature float64                `json:"temperature"` // 溫度參數
	MaxTokens   int                    `json:"max_tokens"`  // 最大 token 數
	Stream      bool                   `json:"stream"`      // 是否流式輸出
	Context     map[string]interface{} `json:"context"`     // 上下文信息
	Timeout     time.Duration          `json:"timeout"`     // 超時時間
}

// ProcessorOptions 處理器選項
type ProcessorOptions struct {
	Format  string                 `json:"format"`  // 輸入格式
	Options map[string]interface{} `json:"options"` // 其他選項
}

// ExecutionResponse 執行回應
type ExecutionResponse struct {
	Content   string                 `json:"content"`    // 回應內容
	Model     string                 `json:"model"`      // 使用的模型
	Metadata  map[string]interface{} `json:"metadata"`   // 元數據
	Usage     *TokenUsage            `json:"usage"`      // 使用量信息
	Error     error                  `json:"error"`      // 錯誤信息
	Duration  time.Duration          `json:"duration"`   // 執行時間
}

// TokenUsage Token 使用量
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`     // 提示詞 token 數
	CompletionTokens int `json:"completion_tokens"` // 完成 token 數
	TotalTokens      int `json:"total_tokens"`      // 總 token 數
}

// ExecutorCapabilities 執行器能力
type ExecutorCapabilities struct {
	SupportedModels   []string          `json:"supported_models"`   // 支持的模型
	MaxTokens         int               `json:"max_tokens"`         // 最大 token 數
	SupportStreaming  bool              `json:"support_streaming"`  // 是否支持流式
	SupportImages     bool              `json:"support_images"`     // 是否支持圖像
	SupportFunctions  bool              `json:"support_functions"`  // 是否支持函數調用
	SupportedLanguages []string         `json:"supported_languages"` // 支持的語言
	Features          map[string]bool   `json:"features"`           // 其他特性
}

// PluginRegistry 插件註冊表
//
// 管理所有已載入的插件，提供註冊、查找和卸載功能
type PluginRegistry struct {
	plugins       map[string]Plugin     // 插件映射 (名稱 -> 插件)
	pluginsByType map[PluginType][]Plugin // 按類型分類的插件
	pluginPaths   map[string]string     // 插件路徑映射
	mu            sync.RWMutex          // 讀寫鎖
}

// NewPluginRegistry 創建新的插件註冊表
func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{
		plugins:       make(map[string]Plugin),
		pluginsByType: make(map[PluginType][]Plugin),
		pluginPaths:   make(map[string]string),
	}
}

// LoadPlugin 載入插件
//
// path: 插件文件路徑 (.so 文件)
// config: 插件配置參數
func (pr *PluginRegistry) LoadPlugin(path string, config map[string]interface{}) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	// 載入動態庫
	p, err := plugin.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open plugin %s: %w", path, err)
	}

	// 查找插件符號
	symbol, err := p.Lookup("Plugin")
	if err != nil {
		return fmt.Errorf("plugin %s does not export 'Plugin' symbol: %w", path, err)
	}

	// 轉換為插件介面
	pluginInstance, ok := symbol.(Plugin)
	if !ok {
		return fmt.Errorf("plugin %s does not implement Plugin interface", path)
	}

	// 初始化插件
	if err := pluginInstance.Initialize(config); err != nil {
		return fmt.Errorf("failed to initialize plugin %s: %w", path, err)
	}

	// 獲取插件元數據
	metadata := pluginInstance.GetMetadata()
	if metadata == nil {
		return fmt.Errorf("plugin %s returned nil metadata", path)
	}

	// 檢查是否已有同名插件
	if _, exists := pr.plugins[metadata.Name]; exists {
		return fmt.Errorf("plugin with name %s already exists", metadata.Name)
	}

	// 註冊插件
	pr.plugins[metadata.Name] = pluginInstance
	pr.pluginPaths[metadata.Name] = path

	// 按類型分類
	if pr.pluginsByType[metadata.Type] == nil {
		pr.pluginsByType[metadata.Type] = make([]Plugin, 0)
	}
	pr.pluginsByType[metadata.Type] = append(pr.pluginsByType[metadata.Type], pluginInstance)

	return nil
}

// UnloadPlugin 卸載插件
func (pr *PluginRegistry) UnloadPlugin(name string) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	pluginInstance, exists := pr.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	// 關閉插件
	if err := pluginInstance.Close(); err != nil {
		return fmt.Errorf("failed to close plugin %s: %w", name, err)
	}

	// 從映射中移除
	metadata := pluginInstance.GetMetadata()
	delete(pr.plugins, name)
	delete(pr.pluginPaths, name)

	// 從類型分類中移除
	if plugins, exists := pr.pluginsByType[metadata.Type]; exists {
		for i, p := range plugins {
			if p.GetMetadata().Name == name {
				pr.pluginsByType[metadata.Type] = append(plugins[:i], plugins[i+1:]...)
				break
			}
		}
	}

	return nil
}

// GetPlugin 獲取插件
func (pr *PluginRegistry) GetPlugin(name string) (Plugin, error) {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	pluginInstance, exists := pr.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", name)
	}

	return pluginInstance, nil
}

// GetExecutorPlugin 獲取執行器插件
func (pr *PluginRegistry) GetExecutorPlugin(name string) (ExecutorPlugin, error) {
	pluginInstance, err := pr.GetPlugin(name)
	if err != nil {
		return nil, err
	}

	executor, ok := pluginInstance.(ExecutorPlugin)
	if !ok {
		return nil, fmt.Errorf("plugin %s is not an executor plugin", name)
	}

	return executor, nil
}

// GetProcessorPlugin 獲取處理器插件
func (pr *PluginRegistry) GetProcessorPlugin(name string) (ProcessorPlugin, error) {
	pluginInstance, err := pr.GetPlugin(name)
	if err != nil {
		return nil, err
	}

	processor, ok := pluginInstance.(ProcessorPlugin)
	if !ok {
		return nil, fmt.Errorf("plugin %s is not a processor plugin", name)
	}

	return processor, nil
}

// ListPlugins 列出所有插件
func (pr *PluginRegistry) ListPlugins() []*PluginMetadata {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	metadata := make([]*PluginMetadata, 0, len(pr.plugins))
	for _, pluginInstance := range pr.plugins {
		metadata = append(metadata, pluginInstance.GetMetadata())
	}

	return metadata
}

// ListPluginsByType 按類型列出插件
func (pr *PluginRegistry) ListPluginsByType(pluginType PluginType) []Plugin {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	plugins, exists := pr.pluginsByType[pluginType]
	if !exists {
		return []Plugin{}
	}

	// 返回副本以避免並發修改
	result := make([]Plugin, len(plugins))
	copy(result, plugins)
	return result
}

// GetPluginCount 獲取插件數量
func (pr *PluginRegistry) GetPluginCount() int {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	return len(pr.plugins)
}

// CheckHealth 檢查所有插件健康狀態
func (pr *PluginRegistry) CheckHealth() map[string]bool {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	health := make(map[string]bool)
	for name, pluginInstance := range pr.plugins {
		health[name] = pluginInstance.IsHealthy()
	}

	return health
}

// CloseAll 關閉所有插件
func (pr *PluginRegistry) CloseAll() error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	var errors []error

	for name, pluginInstance := range pr.plugins {
		if err := pluginInstance.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close plugin %s: %w", name, err))
		}
	}

	// 清空所有映射
	pr.plugins = make(map[string]Plugin)
	pr.pluginsByType = make(map[PluginType][]Plugin)
	pr.pluginPaths = make(map[string]string)

	if len(errors) > 0 {
		return fmt.Errorf("errors closing plugins: %v", errors)
	}
	return nil
}

// List 列出所有插件名稱
func (pr *PluginRegistry) List() []string {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	names := make([]string, 0, len(pr.plugins))
	for name := range pr.plugins {
		names = append(names, name)
	}
	return names
}

// Get 獲取指定名稱的插件（GetPlugin 的別名）
func (pr *PluginRegistry) Get(name string) (Plugin, error) {
	return pr.GetPlugin(name)
}

// Register 註冊插件
func (pr *PluginRegistry) Register(plugin Plugin) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	metadata := plugin.GetMetadata()
	if metadata == nil {
		return fmt.Errorf("plugin metadata is nil")
	}

	if _, exists := pr.plugins[metadata.Name]; exists {
		return ErrPluginAlreadyExists
	}

	pr.plugins[metadata.Name] = plugin
	pr.pluginsByType[metadata.Type] = append(pr.pluginsByType[metadata.Type], plugin)

	return nil
}

// Unregister 取消註冊插件（UnloadPlugin 的別名）
func (pr *PluginRegistry) Unregister(name string) error {
	return pr.UnloadPlugin(name)
}

// PluginManager 插件管理器
//
// 提供插件的完整生命週期管理，包括載入、卸載、配置和監控
type PluginManager struct {
	registry    *PluginRegistry
	config      *PluginConfig
	healthCheck *time.Ticker
	stopChan    chan struct{}
	mu          sync.RWMutex
}

// PluginConfig 插件管理器配置
type PluginConfig struct {
	PluginDir             string        // 插件目錄
	AutoLoadOnStart       bool          // 是否在啟動時自動載入
	HealthCheckInterval   time.Duration // 健康檢查間隔
	EnableHotReload       bool          // 是否啟用熱重載
	DefaultTimeout        time.Duration // 預設超時時間
	MaxPlugins            int           // 最大插件數量
	RequiredPlugins       []string      // 必須載入的插件
}

// DefaultPluginConfig 預設插件配置
func DefaultPluginConfig() *PluginConfig {
	return &PluginConfig{
		PluginDir:           "./plugins",
		AutoLoadOnStart:     true,
		HealthCheckInterval: 30 * time.Second,
		EnableHotReload:     false,
		DefaultTimeout:      30 * time.Second,
		MaxPlugins:          10,
		RequiredPlugins:     []string{},
	}
}

// NewPluginManager 創建插件管理器
func NewPluginManager(config *PluginConfig) *PluginManager {
	if config == nil {
		config = DefaultPluginConfig()
	}

	return &PluginManager{
		registry: NewPluginRegistry(),
		config:   config,
		stopChan: make(chan struct{}),
	}
}

// Start 啟動插件管理器
func (pm *PluginManager) Start() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 如果啟用自動載入，掃描插件目錄
	if pm.config.AutoLoadOnStart {
		if err := pm.scanAndLoadPlugins(); err != nil {
			return fmt.Errorf("failed to auto-load plugins: %w", err)
		}
	}

	// 啟動健康檢查
	pm.startHealthCheck()

	return nil
}

// Stop 停止插件管理器
func (pm *PluginManager) Stop() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 停止健康檢查
	if pm.healthCheck != nil {
		pm.healthCheck.Stop()
	}

	close(pm.stopChan)

	// 關閉所有插件
	return pm.registry.CloseAll()
}

// GetRegistry 獲取插件註冊表
func (pm *PluginManager) GetRegistry() *PluginRegistry {
	return pm.registry
}

// scanAndLoadPlugins 掃描並載入插件目錄中的所有插件
func (pm *PluginManager) scanAndLoadPlugins() error {
	// 實際實作中需要掃描文件系統
	// 這裡提供基本框架
	return nil
}

// startHealthCheck 啟動插件健康檢查
func (pm *PluginManager) startHealthCheck() {
	if pm.config.HealthCheckInterval <= 0 {
		return
	}

	pm.healthCheck = time.NewTicker(pm.config.HealthCheckInterval)

	go func() {
		for {
			select {
			case <-pm.healthCheck.C:
				pm.performHealthCheck()
			case <-pm.stopChan:
				return
			}
		}
	}()
}

// performHealthCheck 執行健康檢查
func (pm *PluginManager) performHealthCheck() {
	health := pm.registry.CheckHealth()

	for name, isHealthy := range health {
		if !isHealthy {
			// 記錄不健康的插件
			// 實際實作中可能需要重啟或移除
			_ = name
		}
	}
}

// ErrPluginNotFound 插件未找到錯誤
var ErrPluginNotFound = errors.New("plugin not found")

// ErrPluginAlreadyExists 插件已存在錯誤  
var ErrPluginAlreadyExists = errors.New("plugin already exists")

// ErrInvalidPluginType 無效插件類型錯誤
var ErrInvalidPluginType = errors.New("invalid plugin type")

// ErrPluginInitFailed 插件初始化失敗錯誤
var ErrPluginInitFailed = errors.New("plugin initialization failed")

// ListPlugins 列出所有已載入的插件
func (pm *PluginManager) ListPlugins() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.registry.List()
}

// GetPlugin 獲取指定名稱的插件
func (pm *PluginManager) GetPlugin(name string) (Plugin, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.registry.Get(name)
}

// LoadPlugin 載入插件
func (pm *PluginManager) LoadPlugin(name string, plugin Plugin) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.registry.Register(plugin)
}

// UnloadPlugin 卸載插件
func (pm *PluginManager) UnloadPlugin(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.registry.Unregister(name)
}
