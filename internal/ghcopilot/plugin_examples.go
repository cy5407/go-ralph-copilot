package ghcopilot

import (
	"context"
	"fmt"
	"time"
)

// OpenAIExecutorPlugin OpenAI GPT 執行器插件示例
//
// 這是一個示例插件，展示如何整合 OpenAI GPT 模型到 Ralph Loop 系統中
type OpenAIExecutorPlugin struct {
	metadata     *PluginMetadata
	apiKey       string
	baseURL      string
	defaultModel string
	timeout      time.Duration
	initialized  bool
}

// NewOpenAIExecutorPlugin 創建 OpenAI 執行器插件
func NewOpenAIExecutorPlugin() *OpenAIExecutorPlugin {
	return &OpenAIExecutorPlugin{
		metadata: &PluginMetadata{
			Name:        "openai-executor",
			Version:     "1.0.0",
			Type:        PluginTypeExecutor,
			Author:      "Ralph Loop Team",
			Description: "OpenAI GPT 模型執行器插件",
			Website:     "https://openai.com",
			License:     "MIT",
			Dependencies: []string{"openai-go-sdk"},
			Configuration: map[string]string{
				"api_key":       "OpenAI API Key (必須)",
				"base_url":      "API 基礎 URL (可選)",
				"default_model": "預設使用的模型 (可選，預設: gpt-4)",
				"timeout":       "請求超時時間 (可選，預設: 30s)",
			},
			CreatedAt: time.Now(),
		},
		defaultModel: "gpt-4",
		timeout:      30 * time.Second,
	}
}

// GetMetadata 實作 Plugin 介面
func (p *OpenAIExecutorPlugin) GetMetadata() *PluginMetadata {
	return p.metadata
}

// Initialize 實作 Plugin 介面
func (p *OpenAIExecutorPlugin) Initialize(config map[string]interface{}) error {
	// 解析配置
	if apiKey, ok := config["api_key"].(string); ok && apiKey != "" {
		p.apiKey = apiKey
	} else {
		return fmt.Errorf("api_key is required for OpenAI plugin")
	}

	if baseURL, ok := config["base_url"].(string); ok && baseURL != "" {
		p.baseURL = baseURL
	} else {
		p.baseURL = "https://api.openai.com/v1"
	}

	if model, ok := config["default_model"].(string); ok && model != "" {
		p.defaultModel = model
	}

	if timeout, ok := config["timeout"].(string); ok && timeout != "" {
		if parsedTimeout, err := time.ParseDuration(timeout); err == nil {
			p.timeout = parsedTimeout
		}
	}

	// 驗證 API Key 格式
	if len(p.apiKey) < 20 {
		return fmt.Errorf("invalid API key format")
	}

	p.initialized = true
	return nil
}

// IsHealthy 實作 Plugin 介面
func (p *OpenAIExecutorPlugin) IsHealthy() bool {
	return p.initialized && p.apiKey != ""
}

// Close 實作 Plugin 介面
func (p *OpenAIExecutorPlugin) Close() error {
	p.initialized = false
	// 清理資源（如關閉連接池等）
	return nil
}

// Execute 實作 ExecutorPlugin 介面
func (p *OpenAIExecutorPlugin) Execute(ctx context.Context, prompt string, options PluginExecutorOptions) (*ExecutionResponse, error) {
	if !p.initialized {
		return nil, fmt.Errorf("plugin not initialized")
	}

	// 設定模型
	model := p.defaultModel
	if options.Model != "" {
		model = options.Model
	}

	// 創建上下文與超時
	execCtx := ctx
	if options.Timeout > 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, options.Timeout)
		defer cancel()
	} else {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, p.timeout)
		defer cancel()
	}

	startTime := time.Now()

	// 模擬 OpenAI API 調用
	// 實際實作中，這裡會調用真正的 OpenAI SDK
	response, err := p.callOpenAIAPI(execCtx, prompt, model, options)
	if err != nil {
		return &ExecutionResponse{
			Content:  "",
			Model:    model,
			Error:    err,
			Duration: time.Since(startTime),
		}, err
	}

	return &ExecutionResponse{
		Content:  response.Content,
		Model:    response.Model,
		Metadata: response.Metadata,
		Usage:    response.Usage,
		Duration: time.Since(startTime),
	}, nil
}

// GetCapabilities 實作 ExecutorPlugin 介面
func (p *OpenAIExecutorPlugin) GetCapabilities() *ExecutorCapabilities {
	return &ExecutorCapabilities{
		SupportedModels: []string{
			"gpt-4",
			"gpt-4-turbo",
			"gpt-3.5-turbo",
			"gpt-3.5-turbo-16k",
		},
		MaxTokens:         4096,
		SupportStreaming:  true,
		SupportImages:     true,
		SupportFunctions:  true,
		SupportedLanguages: []string{"en", "zh", "ja", "fr", "de", "es"},
		Features: map[string]bool{
			"function_calling": true,
			"vision":           true,
			"json_mode":        true,
			"tool_use":         true,
		},
	}
}

// SetModel 實作 ExecutorPlugin 介面
func (p *OpenAIExecutorPlugin) SetModel(model string) error {
	capabilities := p.GetCapabilities()
	for _, supportedModel := range capabilities.SupportedModels {
		if supportedModel == model {
			p.defaultModel = model
			return nil
		}
	}
	return fmt.Errorf("model %s is not supported", model)
}

// GetAvailableModels 實作 ExecutorPlugin 介面
func (p *OpenAIExecutorPlugin) GetAvailableModels() []string {
	return p.GetCapabilities().SupportedModels
}

// callOpenAIAPI 模擬 OpenAI API 調用
func (p *OpenAIExecutorPlugin) callOpenAIAPI(ctx context.Context, prompt, model string, options PluginExecutorOptions) (*ExecutionResponse, error) {
	// 模擬 API 延遲
	select {
	case <-time.After(100 * time.Millisecond):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// 模擬回應
	mockResponse := fmt.Sprintf("OpenAI %s 模型回應: %s", model, prompt)

	return &ExecutionResponse{
		Content: mockResponse,
		Model:   model,
		Metadata: map[string]interface{}{
			"finish_reason": "stop",
			"provider":      "openai",
			"api_version":   "2024-02-01",
		},
		Usage: &TokenUsage{
			PromptTokens:     len(prompt) / 4, // 粗略估算
			CompletionTokens: len(mockResponse) / 4,
			TotalTokens:      (len(prompt) + len(mockResponse)) / 4,
		},
	}, nil
}

// ClaudeExecutorPlugin Anthropic Claude 執行器插件示例
type ClaudeExecutorPlugin struct {
	metadata     *PluginMetadata
	apiKey       string
	defaultModel string
	timeout      time.Duration
	initialized  bool
}

// NewClaudeExecutorPlugin 創建 Claude 執行器插件
func NewClaudeExecutorPlugin() *ClaudeExecutorPlugin {
	return &ClaudeExecutorPlugin{
		metadata: &PluginMetadata{
			Name:        "claude-executor",
			Version:     "1.0.0",
			Type:        PluginTypeExecutor,
			Author:      "Ralph Loop Team",
			Description: "Anthropic Claude 模型執行器插件",
			Website:     "https://anthropic.com",
			License:     "MIT",
			Dependencies: []string{"anthropic-go-sdk"},
			Configuration: map[string]string{
				"api_key":       "Anthropic API Key (必須)",
				"default_model": "預設使用的模型 (可選，預設: claude-3-sonnet)",
				"timeout":       "請求超時時間 (可選，預設: 30s)",
			},
			CreatedAt: time.Now(),
		},
		defaultModel: "claude-3-sonnet",
		timeout:      30 * time.Second,
	}
}

// GetMetadata 實作 Plugin 介面
func (p *ClaudeExecutorPlugin) GetMetadata() *PluginMetadata {
	return p.metadata
}

// Initialize 實作 Plugin 介面
func (p *ClaudeExecutorPlugin) Initialize(config map[string]interface{}) error {
	if apiKey, ok := config["api_key"].(string); ok && apiKey != "" {
		p.apiKey = apiKey
	} else {
		return fmt.Errorf("api_key is required for Claude plugin")
	}

	if model, ok := config["default_model"].(string); ok && model != "" {
		p.defaultModel = model
	}

	if timeout, ok := config["timeout"].(string); ok && timeout != "" {
		if parsedTimeout, err := time.ParseDuration(timeout); err == nil {
			p.timeout = parsedTimeout
		}
	}

	p.initialized = true
	return nil
}

// IsHealthy 實作 Plugin 介面
func (p *ClaudeExecutorPlugin) IsHealthy() bool {
	return p.initialized && p.apiKey != ""
}

// Close 實作 Plugin 介面
func (p *ClaudeExecutorPlugin) Close() error {
	p.initialized = false
	return nil
}

// Execute 實作 ExecutorPlugin 介面
func (p *ClaudeExecutorPlugin) Execute(ctx context.Context, prompt string, options PluginExecutorOptions) (*ExecutionResponse, error) {
	if !p.initialized {
		return nil, fmt.Errorf("plugin not initialized")
	}

	model := p.defaultModel
	if options.Model != "" {
		model = options.Model
	}

	execCtx := ctx
	if options.Timeout > 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, options.Timeout)
		defer cancel()
	} else {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, p.timeout)
		defer cancel()
	}

	startTime := time.Now()

	// 模擬 Claude API 調用
	response, err := p.callClaudeAPI(execCtx, prompt, model, options)
	if err != nil {
		return &ExecutionResponse{
			Content:  "",
			Model:    model,
			Error:    err,
			Duration: time.Since(startTime),
		}, err
	}

	return response, nil
}

// GetCapabilities 實作 ExecutorPlugin 介面
func (p *ClaudeExecutorPlugin) GetCapabilities() *ExecutorCapabilities {
	return &ExecutorCapabilities{
		SupportedModels: []string{
			"claude-3-opus",
			"claude-3-sonnet",
			"claude-3-haiku",
			"claude-2.1",
		},
		MaxTokens:         200000,
		SupportStreaming:  true,
		SupportImages:     true,
		SupportFunctions:  true,
		SupportedLanguages: []string{"en", "zh", "ja", "fr", "de", "es", "it", "pt", "ru", "ko"},
		Features: map[string]bool{
			"large_context":    true,
			"vision":           true,
			"tool_use":         true,
			"code_generation":  true,
			"analysis":         true,
		},
	}
}

// SetModel 實作 ExecutorPlugin 介面
func (p *ClaudeExecutorPlugin) SetModel(model string) error {
	capabilities := p.GetCapabilities()
	for _, supportedModel := range capabilities.SupportedModels {
		if supportedModel == model {
			p.defaultModel = model
			return nil
		}
	}
	return fmt.Errorf("model %s is not supported", model)
}

// GetAvailableModels 實作 ExecutorPlugin 介面
func (p *ClaudeExecutorPlugin) GetAvailableModels() []string {
	return p.GetCapabilities().SupportedModels
}

// callClaudeAPI 模擬 Claude API 調用
func (p *ClaudeExecutorPlugin) callClaudeAPI(ctx context.Context, prompt, model string, options PluginExecutorOptions) (*ExecutionResponse, error) {
	// 模擬 API 延遲
	select {
	case <-time.After(150 * time.Millisecond):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// 模擬回應
	mockResponse := fmt.Sprintf("Claude %s 模型回應: 我是 Claude，一個 AI 助理。您的問題「%s」已收到，正在處理中...", model, prompt)

	return &ExecutionResponse{
		Content: mockResponse,
		Model:   model,
		Metadata: map[string]interface{}{
			"stop_reason":   "end_turn",
			"provider":      "anthropic",
			"api_version":   "2024-02-15",
			"safety_level":  "standard",
		},
		Usage: &TokenUsage{
			PromptTokens:     len(prompt) / 3, // Claude 通常有更好的 tokenization
			CompletionTokens: len(mockResponse) / 3,
			TotalTokens:      (len(prompt) + len(mockResponse)) / 3,
		},
	}, nil
}

// OutputProcessorPlugin 輸出處理器插件示例
type OutputProcessorPlugin struct {
	metadata    *PluginMetadata
	initialized bool
}

// NewOutputProcessorPlugin 創建輸出處理器插件
func NewOutputProcessorPlugin() *OutputProcessorPlugin {
	return &OutputProcessorPlugin{
		metadata: &PluginMetadata{
			Name:        "output-processor",
			Version:     "1.0.0",
			Type:        PluginTypeProcessor,
			Author:      "Ralph Loop Team",
			Description: "輸出後處理插件，支持格式化、驗證和轉換",
			License:     "MIT",
			Configuration: map[string]string{
				"enable_markdown": "是否啟用 Markdown 格式化 (可選，預設: true)",
				"enable_syntax":   "是否啟用語法高亮 (可選，預設: true)",
			},
			CreatedAt: time.Now(),
		},
	}
}

// GetMetadata 實作 Plugin 介面
func (p *OutputProcessorPlugin) GetMetadata() *PluginMetadata {
	return p.metadata
}

// Initialize 實作 Plugin 介面
func (p *OutputProcessorPlugin) Initialize(config map[string]interface{}) error {
	p.initialized = true
	return nil
}

// IsHealthy 實作 Plugin 介面
func (p *OutputProcessorPlugin) IsHealthy() bool {
	return p.initialized
}

// Close 實作 Plugin 介面
func (p *OutputProcessorPlugin) Close() error {
	p.initialized = false
	return nil
}

// Process 實作 ProcessorPlugin 介面
func (p *OutputProcessorPlugin) Process(ctx context.Context, input interface{}, options ProcessorOptions) (interface{}, error) {
	if !p.initialized {
		return nil, fmt.Errorf("plugin not initialized")
	}

	// 處理字符串輸入
	inputStr, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("input must be a string")
	}

	// 模擬處理邏輯
	processed := fmt.Sprintf("處理後的輸出:\n```\n%s\n```\n\n> 由 output-processor 插件處理", inputStr)

	return processed, nil
}

// GetSupportedFormats 實作 ProcessorPlugin 介面
func (p *OutputProcessorPlugin) GetSupportedFormats() []string {
	return []string{"text/plain", "text/markdown", "application/json"}
}