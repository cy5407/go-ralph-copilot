package ghcopilot

import (
	"context"
	"sync"
	"time"
)

// ExecutionMode 定義執行模式
type ExecutionMode int

const (
	// ModeCLI 使用 CLI 執行（輕量級）
	ModeCLI ExecutionMode = iota
	// ModeSDK 使用 SDK 執行（類型安全）
	ModeSDK
	// ModePlugin 使用插件執行（可擴展）
	ModePlugin
	// ModeAuto 自動選擇最佳模式
	ModeAuto
	// ModeHybrid 混合模式
	ModeHybrid
)

// String 返回執行模式的字串表示
func (m ExecutionMode) String() string {
	switch m {
	case ModeCLI:
		return "cli"
	case ModeSDK:
		return "sdk"
	case ModePlugin:
		return "plugin"
	case ModeAuto:
		return "auto"
	case ModeHybrid:
		return "hybrid"
	default:
		return "unknown"
	}
}

// TaskComplexity 定義任務複雜度
type TaskComplexity int

const (
	// ComplexitySimple 簡單任務
	ComplexitySimple TaskComplexity = iota
	// ComplexityMedium 中等任務
	ComplexityMedium
	// ComplexityComplex 複雜任務
	ComplexityComplex
)

// String 返回任務複雜度的字串表示
func (c TaskComplexity) String() string {
	switch c {
	case ComplexitySimple:
		return "simple"
	case ComplexityMedium:
		return "medium"
	case ComplexityComplex:
		return "complex"
	default:
		return "unknown"
	}
}

// Task 定義任務結構
type Task struct {
	ID             string
	Prompt         string
	Complexity     TaskComplexity
	RequiresSDK    bool
	PreferredMode  ExecutionMode
	Timeout        time.Duration
	Priority       int
	Tags           []string
}

// NewTask 建立新任務
func NewTask(id, prompt string) *Task {
	return &Task{
		ID:            id,
		Prompt:        prompt,
		Complexity:    ComplexitySimple,
		PreferredMode: ModeAuto,
		Timeout:       30 * time.Second,
		Priority:      5,
	}
}

// WithComplexity 設定任務複雜度
func (t *Task) WithComplexity(complexity TaskComplexity) *Task {
	t.Complexity = complexity
	return t
}

// WithTimeout 設定任務逾時
func (t *Task) WithTimeout(timeout time.Duration) *Task {
	t.Timeout = timeout
	return t
}

// WithPreferredMode 設定偏好模式
func (t *Task) WithPreferredMode(mode ExecutionMode) *Task {
	t.PreferredMode = mode
	return t
}

// WithPriority 設定優先級
func (t *Task) WithPriority(priority int) *Task {
	t.Priority = priority
	return t
}

// WithTags 設定標籤
func (t *Task) WithTags(tags ...string) *Task {
	t.Tags = tags
	return t
}

// SetRequiresSDK 設定是否需要 SDK
func (t *Task) SetRequiresSDK(requires bool) *Task {
	t.RequiresSDK = requires
	return t
}

// ExecutionModeSelector 執行模式選擇器
type ExecutionModeSelector struct {
	defaultMode      ExecutionMode
	fallbackEnabled  bool
	sdkAvailable     bool
	cliAvailable     bool
	pluginAvailable  bool
	preferredPlugin  string
	metrics          *SelectorMetrics
	rules            []SelectionRule
	mu               sync.RWMutex
}

// SelectorMetrics 選擇器指標統計
type SelectorMetrics struct {
	TotalSelections   int64
	CLISelections     int64
	SDKSelections     int64
	PluginSelections  int64
	FallbackCount     int64
	LastSelection     ExecutionMode
	LastSelectionTime time.Time
	mu                sync.RWMutex
}

// SelectionRule 選擇規則
type SelectionRule struct {
	Name        string
	Priority    int
	Condition   func(task *Task) bool
	Mode        ExecutionMode
}

// NewExecutionModeSelector 建立新的執行模式選擇器
func NewExecutionModeSelector() *ExecutionModeSelector {
	return &ExecutionModeSelector{
		defaultMode:     ModeAuto,
		fallbackEnabled: true,
		sdkAvailable:    true,
		cliAvailable:    true,
		metrics:         &SelectorMetrics{},
		rules:           make([]SelectionRule, 0),
	}
}

// SetDefaultMode 設定預設模式
func (s *ExecutionModeSelector) SetDefaultMode(mode ExecutionMode) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.defaultMode = mode
}

// GetDefaultMode 取得預設模式
func (s *ExecutionModeSelector) GetDefaultMode() ExecutionMode {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.defaultMode
}

// SetFallbackEnabled 設定是否啟用故障轉移
func (s *ExecutionModeSelector) SetFallbackEnabled(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fallbackEnabled = enabled
}

// IsFallbackEnabled 檢查是否啟用故障轉移
func (s *ExecutionModeSelector) IsFallbackEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.fallbackEnabled
}

// SetSDKAvailable 設定 SDK 是否可用
func (s *ExecutionModeSelector) SetSDKAvailable(available bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sdkAvailable = available
}

// IsSDKAvailable 檢查 SDK 是否可用
func (s *ExecutionModeSelector) IsSDKAvailable() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sdkAvailable
}

// SetCLIAvailable 設定 CLI 是否可用
func (s *ExecutionModeSelector) SetCLIAvailable(available bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cliAvailable = available
}

// IsCLIAvailable 檢查 CLI 是否可用
func (s *ExecutionModeSelector) IsCLIAvailable() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cliAvailable
}

// SetPluginAvailable 設定插件是否可用
func (s *ExecutionModeSelector) SetPluginAvailable(available bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pluginAvailable = available
}

// IsPluginAvailable 檢查插件是否可用
func (s *ExecutionModeSelector) IsPluginAvailable() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.pluginAvailable
}

// SetPreferredPlugin 設定偏好的插件
func (s *ExecutionModeSelector) SetPreferredPlugin(plugin string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.preferredPlugin = plugin
}

// GetPreferredPlugin 取得偏好的插件
func (s *ExecutionModeSelector) GetPreferredPlugin() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.preferredPlugin
}

// AddRule 添加選擇規則
func (s *ExecutionModeSelector) AddRule(rule SelectionRule) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.rules = append(s.rules, rule)

	// 按優先級排序（優先級數字越小越優先）
	for i := len(s.rules) - 1; i > 0; i-- {
		if s.rules[i].Priority < s.rules[i-1].Priority {
			s.rules[i], s.rules[i-1] = s.rules[i-1], s.rules[i]
		}
	}
}

// ClearRules 清除所有規則
func (s *ExecutionModeSelector) ClearRules() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rules = make([]SelectionRule, 0)
}

// GetRuleCount 取得規則數量
func (s *ExecutionModeSelector) GetRuleCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.rules)
}

// Choose 為任務選擇最佳執行模式
func (s *ExecutionModeSelector) Choose(task *Task) ExecutionMode {
	s.mu.RLock()
	rules := make([]SelectionRule, len(s.rules))
	copy(rules, s.rules)
	defaultMode := s.defaultMode
	fallbackEnabled := s.fallbackEnabled
	sdkAvailable := s.sdkAvailable
	cliAvailable := s.cliAvailable
	pluginAvailable := s.pluginAvailable
	preferredPlugin := s.preferredPlugin
	s.mu.RUnlock()

	// 記錄選擇時間 (不在這裡增加 TotalSelections，由 recordSelection 負責)
	defer func() {
		s.metrics.mu.Lock()
		s.metrics.LastSelectionTime = time.Now()
		s.metrics.mu.Unlock()
	}()

	// 如果任務指定了偏好模式且不是自動模式
	if task != nil && task.PreferredMode != ModeAuto {
		mode := s.validateAndFallback(task.PreferredMode, sdkAvailable, cliAvailable, pluginAvailable, fallbackEnabled)
		s.recordSelection(mode)
		return mode
	}

	// 如果有偏好插件且插件可用
	if preferredPlugin != "" && pluginAvailable {
		s.recordSelection(ModePlugin)
		return ModePlugin
	}

	// 如果任務需要 SDK
	if task != nil && task.RequiresSDK {
		if sdkAvailable {
			s.recordSelection(ModeSDK)
			return ModeSDK
		}
		if fallbackEnabled && cliAvailable {
			s.recordFallback()
			s.recordSelection(ModeCLI)
			return ModeCLI
		}
	}

	// 應用規則
	for _, rule := range rules {
		if task != nil && rule.Condition(task) {
			mode := s.validateAndFallback(rule.Mode, sdkAvailable, cliAvailable, pluginAvailable, fallbackEnabled)
			s.recordSelection(mode)
			return mode
		}
	}

	// 根據任務複雜度選擇
	if task != nil {
		switch task.Complexity {
		case ComplexitySimple:
			// 簡單任務使用 CLI
			mode := s.validateAndFallback(ModeCLI, sdkAvailable, cliAvailable, pluginAvailable, fallbackEnabled)
			s.recordSelection(mode)
			return mode
		case ComplexityComplex:
			// 複雜任務使用 SDK
			mode := s.validateAndFallback(ModeSDK, sdkAvailable, cliAvailable, pluginAvailable, fallbackEnabled)
			s.recordSelection(mode)
			return mode
		}
	}

	// 使用預設模式
	mode := s.resolveAutoMode(defaultMode, sdkAvailable, cliAvailable)
	s.recordSelection(mode)
	return mode
}

// validateAndFallback 驗證模式並在必要時進行故障轉移
func (s *ExecutionModeSelector) validateAndFallback(
	mode ExecutionMode,
	sdkAvailable, cliAvailable, pluginAvailable, fallbackEnabled bool,
) ExecutionMode {
	switch mode {
	case ModeSDK:
		if sdkAvailable {
			return ModeSDK
		}
		if fallbackEnabled && cliAvailable {
			s.recordFallback()
			return ModeCLI
		}
		return ModeSDK // 返回請求的模式，即使不可用

	case ModeCLI:
		if cliAvailable {
			return ModeCLI
		}
		if fallbackEnabled && sdkAvailable {
			s.recordFallback()
			return ModeSDK
		}
		return ModeCLI

	case ModePlugin:
		if pluginAvailable {
			return ModePlugin
		}
		// 插件不可用時優先降級到 SDK，再到 CLI
		if fallbackEnabled && sdkAvailable {
			s.recordFallback()
			return ModeSDK
		}
		if fallbackEnabled && cliAvailable {
			s.recordFallback()
			return ModeCLI
		}
		return ModePlugin

	case ModeHybrid:
		// 混合模式優先使用 SDK
		if sdkAvailable {
			return ModeHybrid
		}
		if fallbackEnabled && cliAvailable {
			s.recordFallback()
			return ModeCLI
		}
		return ModeHybrid

	default:
		return s.resolveAutoMode(ModeAuto, sdkAvailable, cliAvailable)
	}
}

// resolveAutoMode 解析自動模式
func (s *ExecutionModeSelector) resolveAutoMode(mode ExecutionMode, sdkAvailable, cliAvailable bool) ExecutionMode {
	if mode == ModeAuto {
		// 自動模式：優先 CLI（輕量級）
		if cliAvailable {
			return ModeCLI
		}
		if sdkAvailable {
			return ModeSDK
		}
	}
	return mode
}

// recordSelection 記錄選擇
func (s *ExecutionModeSelector) recordSelection(mode ExecutionMode) {
	s.metrics.mu.Lock()
	defer s.metrics.mu.Unlock()

	s.metrics.LastSelection = mode
	s.metrics.TotalSelections++ // 增加總選擇計數
	switch mode {
	case ModeCLI:
		s.metrics.CLISelections++
	case ModeSDK:
		s.metrics.SDKSelections++
	case ModePlugin:
		s.metrics.PluginSelections++
	}
}

// recordFallback 記錄故障轉移
func (s *ExecutionModeSelector) recordFallback() {
	s.metrics.mu.Lock()
	defer s.metrics.mu.Unlock()
	s.metrics.FallbackCount++
}

// GetMetrics 取得選擇器指標
func (s *ExecutionModeSelector) GetMetrics() *SelectorMetrics {
	s.metrics.mu.RLock()
	defer s.metrics.mu.RUnlock()

	return &SelectorMetrics{
		TotalSelections:   s.metrics.TotalSelections,
		CLISelections:     s.metrics.CLISelections,
		SDKSelections:     s.metrics.SDKSelections,
		PluginSelections:  s.metrics.PluginSelections,
		FallbackCount:     s.metrics.FallbackCount,
		LastSelection:     s.metrics.LastSelection,
		LastSelectionTime: s.metrics.LastSelectionTime,
	}
}

// ResetMetrics 重置指標
func (s *ExecutionModeSelector) ResetMetrics() {
	s.metrics.mu.Lock()
	defer s.metrics.mu.Unlock()

	s.metrics.TotalSelections = 0
	s.metrics.CLISelections = 0
	s.metrics.SDKSelections = 0
	s.metrics.PluginSelections = 0
	s.metrics.FallbackCount = 0
	s.metrics.LastSelection = 0
	s.metrics.LastSelectionTime = time.Time{}
}

// PerformanceMetrics 效能指標
type PerformanceMetrics struct {
	CLITime      time.Duration
	SDKTime      time.Duration
	MemoryUsage  uint64
	ErrorRate    float64
	Throughput   float64
}

// PerformanceMonitor 效能監控器
type PerformanceMonitor struct {
	cliMetrics  *modeMetrics
	sdkMetrics  *modeMetrics
	mu          sync.RWMutex
}

// modeMetrics 模式指標
type modeMetrics struct {
	totalExecutions  int64
	totalTime        time.Duration
	errorCount       int64
	mu               sync.Mutex
}

// NewPerformanceMonitor 建立新的效能監控器
func NewPerformanceMonitor() *PerformanceMonitor {
	return &PerformanceMonitor{
		cliMetrics: &modeMetrics{},
		sdkMetrics: &modeMetrics{},
	}
}

// RecordExecution 記錄執行
func (p *PerformanceMonitor) RecordExecution(mode ExecutionMode, duration time.Duration, err error) {
	var metrics *modeMetrics

	p.mu.RLock()
	switch mode {
	case ModeCLI:
		metrics = p.cliMetrics
	case ModeSDK:
		metrics = p.sdkMetrics
	default:
		p.mu.RUnlock()
		return
	}
	p.mu.RUnlock()

	metrics.mu.Lock()
	defer metrics.mu.Unlock()

	metrics.totalExecutions++
	metrics.totalTime += duration
	if err != nil {
		metrics.errorCount++
	}
}

// GetPerformanceMetrics 取得效能指標
func (p *PerformanceMonitor) GetPerformanceMetrics() *PerformanceMetrics {
	p.mu.RLock()
	cliMetrics := p.cliMetrics
	sdkMetrics := p.sdkMetrics
	p.mu.RUnlock()

	cliMetrics.mu.Lock()
	cliTime := time.Duration(0)
	if cliMetrics.totalExecutions > 0 {
		cliTime = cliMetrics.totalTime / time.Duration(cliMetrics.totalExecutions)
	}
	cliMetrics.mu.Unlock()

	sdkMetrics.mu.Lock()
	sdkTime := time.Duration(0)
	if sdkMetrics.totalExecutions > 0 {
		sdkTime = sdkMetrics.totalTime / time.Duration(sdkMetrics.totalExecutions)
	}
	sdkMetrics.mu.Unlock()

	// 計算整體錯誤率
	totalExecs := cliMetrics.totalExecutions + sdkMetrics.totalExecutions
	totalErrors := cliMetrics.errorCount + sdkMetrics.errorCount
	overallErrorRate := float64(0)
	if totalExecs > 0 {
		overallErrorRate = float64(totalErrors) / float64(totalExecs)
	}

	return &PerformanceMetrics{
		CLITime:   cliTime,
		SDKTime:   sdkTime,
		ErrorRate: overallErrorRate,
	}
}

// GetCLIMetrics 取得 CLI 模式指標
func (p *PerformanceMonitor) GetCLIMetrics() (executions int64, avgTime time.Duration, errorRate float64) {
	p.cliMetrics.mu.Lock()
	defer p.cliMetrics.mu.Unlock()

	executions = p.cliMetrics.totalExecutions
	if executions > 0 {
		avgTime = p.cliMetrics.totalTime / time.Duration(executions)
		errorRate = float64(p.cliMetrics.errorCount) / float64(executions)
	}
	return
}

// GetSDKMetrics 取得 SDK 模式指標
func (p *PerformanceMonitor) GetSDKMetrics() (executions int64, avgTime time.Duration, errorRate float64) {
	p.sdkMetrics.mu.Lock()
	defer p.sdkMetrics.mu.Unlock()

	executions = p.sdkMetrics.totalExecutions
	if executions > 0 {
		avgTime = p.sdkMetrics.totalTime / time.Duration(executions)
		errorRate = float64(p.sdkMetrics.errorCount) / float64(executions)
	}
	return
}

// Reset 重置所有指標
func (p *PerformanceMonitor) Reset() {
	p.cliMetrics.mu.Lock()
	p.cliMetrics.totalExecutions = 0
	p.cliMetrics.totalTime = 0
	p.cliMetrics.errorCount = 0
	p.cliMetrics.mu.Unlock()

	p.sdkMetrics.mu.Lock()
	p.sdkMetrics.totalExecutions = 0
	p.sdkMetrics.totalTime = 0
	p.sdkMetrics.errorCount = 0
	p.sdkMetrics.mu.Unlock()
}

// HybridExecutor 混合執行器
type HybridExecutor struct {
	selector   *ExecutionModeSelector
	monitor    *PerformanceMonitor
	cliFunc    func(ctx context.Context, prompt string) (string, error)
	sdkFunc    func(ctx context.Context, prompt string) (string, error)
	mu         sync.RWMutex
}

// NewHybridExecutor 建立新的混合執行器
func NewHybridExecutor(selector *ExecutionModeSelector) *HybridExecutor {
	if selector == nil {
		selector = NewExecutionModeSelector()
	}
	return &HybridExecutor{
		selector: selector,
		monitor:  NewPerformanceMonitor(),
		cliFunc:  func(ctx context.Context, prompt string) (string, error) { return "", nil },
		sdkFunc:  func(ctx context.Context, prompt string) (string, error) { return "", nil },
	}
}

// SetCLIExecutor 設定 CLI 執行函式
func (h *HybridExecutor) SetCLIExecutor(fn func(ctx context.Context, prompt string) (string, error)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cliFunc = fn
}

// SetSDKExecutor 設定 SDK 執行函式
func (h *HybridExecutor) SetSDKExecutor(fn func(ctx context.Context, prompt string) (string, error)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.sdkFunc = fn
}

// Execute 執行任務
func (h *HybridExecutor) Execute(ctx context.Context, task *Task) (string, error) {
	// 選擇執行模式
	mode := h.selector.Choose(task)

	h.mu.RLock()
	cliFunc := h.cliFunc
	sdkFunc := h.sdkFunc
	h.mu.RUnlock()

	start := time.Now()
	var result string
	var err error

	prompt := ""
	if task != nil {
		prompt = task.Prompt
	}

	switch mode {
	case ModeCLI:
		result, err = cliFunc(ctx, prompt)
	case ModeSDK:
		result, err = sdkFunc(ctx, prompt)
	case ModeHybrid:
		// 混合模式：先嘗試 SDK，失敗則使用 CLI
		result, err = sdkFunc(ctx, prompt)
		if err != nil && h.selector.IsFallbackEnabled() && h.selector.IsCLIAvailable() {
			result, err = cliFunc(ctx, prompt)
			mode = ModeCLI // 更新記錄的模式
		}
	default:
		result, err = cliFunc(ctx, prompt)
		mode = ModeCLI
	}

	// 記錄效能
	h.monitor.RecordExecution(mode, time.Since(start), err)

	return result, err
}

// GetSelector 取得選擇器
func (h *HybridExecutor) GetSelector() *ExecutionModeSelector {
	return h.selector
}

// GetPerformanceMonitor 取得效能監控器
func (h *HybridExecutor) GetPerformanceMonitor() *PerformanceMonitor {
	return h.monitor
}
