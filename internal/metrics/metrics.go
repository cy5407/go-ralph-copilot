package metrics

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"
)

// MetricType 定義指標類型
type MetricType string

const (
	Counter   MetricType = "counter"
	Gauge     MetricType = "gauge"
	Histogram MetricType = "histogram"
	Timer     MetricType = "timer"
)

// Metric 表示一個指標
type Metric interface {
	Name() string
	Type() MetricType
	Value() interface{}
	Reset()
}

// CounterMetric 計數器指標
type CounterMetric struct {
	name  string
	value int64
	mu    sync.RWMutex
}

// NewCounter 創建新的計數器
func NewCounter(name string) *CounterMetric {
	return &CounterMetric{name: name}
}

func (c *CounterMetric) Name() string     { return c.name }
func (c *CounterMetric) Type() MetricType { return Counter }

func (c *CounterMetric) Value() interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.value
}

func (c *CounterMetric) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value = 0
}

// Inc 遞增計數器
func (c *CounterMetric) Inc() {
	c.Add(1)
}

// Add 增加計數器值
func (c *CounterMetric) Add(delta int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value += delta
}

// Get 獲取當前值
func (c *CounterMetric) Get() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.value
}

// GaugeMetric 標量指標
type GaugeMetric struct {
	name  string
	value float64
	mu    sync.RWMutex
}

// NewGauge 創建新的標量
func NewGauge(name string) *GaugeMetric {
	return &GaugeMetric{name: name}
}

func (g *GaugeMetric) Name() string     { return g.name }
func (g *GaugeMetric) Type() MetricType { return Gauge }

func (g *GaugeMetric) Value() interface{} {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.value
}

func (g *GaugeMetric) Reset() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.value = 0
}

// Set 設置標量值
func (g *GaugeMetric) Set(value float64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.value = value
}

// Inc 遞增標量
func (g *GaugeMetric) Inc() {
	g.Add(1)
}

// Dec 遞減標量
func (g *GaugeMetric) Dec() {
	g.Add(-1)
}

// Add 增加標量值
func (g *GaugeMetric) Add(delta float64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.value += delta
}

// Get 獲取當前值
func (g *GaugeMetric) Get() float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.value
}

// TimerMetric 計時器指標
type TimerMetric struct {
	name      string
	durations []time.Duration
	mu        sync.RWMutex
}

// NewTimer 創建新的計時器
func NewTimer(name string) *TimerMetric {
	return &TimerMetric{
		name:      name,
		durations: make([]time.Duration, 0),
	}
}

func (t *TimerMetric) Name() string     { return t.name }
func (t *TimerMetric) Type() MetricType { return Timer }

func (t *TimerMetric) Value() interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	if len(t.durations) == 0 {
		return map[string]interface{}{
			"count": 0,
			"min":   0,
			"max":   0,
			"avg":   0,
			"p50":   0,
			"p95":   0,
			"p99":   0,
		}
	}

	sorted := make([]time.Duration, len(t.durations))
	copy(sorted, t.durations)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	var total time.Duration
	for _, d := range sorted {
		total += d
	}

	count := len(sorted)
	return map[string]interface{}{
		"count": count,
		"min":   sorted[0].Milliseconds(),
		"max":   sorted[count-1].Milliseconds(),
		"avg":   total.Milliseconds() / int64(count),
		"p50":   sorted[count*50/100].Milliseconds(),
		"p95":   sorted[count*95/100].Milliseconds(),
		"p99":   sorted[count*99/100].Milliseconds(),
	}
}

func (t *TimerMetric) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.durations = t.durations[:0]
}

// Record 記錄一個時間值
func (t *TimerMetric) Record(duration time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.durations = append(t.durations, duration)
}

// Time 測量函數執行時間
func (t *TimerMetric) Time(fn func()) {
	start := time.Now()
	fn()
	t.Record(time.Since(start))
}

// Start 開始計時，返回停止函數
func (t *TimerMetric) Start() func() {
	start := time.Now()
	return func() {
		t.Record(time.Since(start))
	}
}

// LoopMetrics Ralph Loop 專用指標
type LoopMetrics struct {
	// 計數器
	TotalLoops           *CounterMetric
	SuccessfulLoops      *CounterMetric
	FailedLoops          *CounterMetric
	TimeoutLoops         *CounterMetric
	CircuitBreakerTrips  *CounterMetric
	RetryAttempts        *CounterMetric
	CLIExecutions        *CounterMetric
	SDKExecutions        *CounterMetric

	// 標量
	CurrentActiveLoops   *GaugeMetric
	CircuitBreakerState  *GaugeMetric // 0=closed, 1=open, 2=half-open
	ErrorRate            *GaugeMetric
	AverageLoopDuration  *GaugeMetric

	// 計時器
	LoopExecutionTime    *TimerMetric
	CLIExecutionTime     *TimerMetric
	SDKExecutionTime     *TimerMetric
	AIResponseTime       *TimerMetric
}

// NewLoopMetrics 創建新的 Ralph Loop 指標集合
func NewLoopMetrics() *LoopMetrics {
	return &LoopMetrics{
		// 計數器
		TotalLoops:           NewCounter("ralph_loops_total"),
		SuccessfulLoops:      NewCounter("ralph_loops_successful"),
		FailedLoops:          NewCounter("ralph_loops_failed"),
		TimeoutLoops:         NewCounter("ralph_loops_timeout"),
		CircuitBreakerTrips:  NewCounter("ralph_circuit_breaker_trips"),
		RetryAttempts:        NewCounter("ralph_retry_attempts"),
		CLIExecutions:        NewCounter("ralph_cli_executions"),
		SDKExecutions:        NewCounter("ralph_sdk_executions"),

		// 標量
		CurrentActiveLoops:   NewGauge("ralph_active_loops"),
		CircuitBreakerState:  NewGauge("ralph_circuit_breaker_state"),
		ErrorRate:           NewGauge("ralph_error_rate"),
		AverageLoopDuration: NewGauge("ralph_avg_loop_duration_ms"),

		// 計時器
		LoopExecutionTime:   NewTimer("ralph_loop_execution_time"),
		CLIExecutionTime:    NewTimer("ralph_cli_execution_time"),
		SDKExecutionTime:    NewTimer("ralph_sdk_execution_time"),
		AIResponseTime:      NewTimer("ralph_ai_response_time"),
	}
}

// UpdateErrorRate 更新錯誤率
func (lm *LoopMetrics) UpdateErrorRate() {
	total := lm.TotalLoops.Get()
	failed := lm.FailedLoops.Get()
	
	if total > 0 {
		rate := float64(failed) / float64(total) * 100
		lm.ErrorRate.Set(rate)
	}
}

// UpdateAverageLoopDuration 更新平均迴圈執行時間
func (lm *LoopMetrics) UpdateAverageLoopDuration() {
	timerValue := lm.LoopExecutionTime.Value().(map[string]interface{})
	if avg, ok := timerValue["avg"].(int64); ok && avg > 0 {
		lm.AverageLoopDuration.Set(float64(avg))
	}
}

// MetricsCollector 指標收集器
type MetricsCollector struct {
	metrics    map[string]Metric
	loopMetrics *LoopMetrics
	mu         sync.RWMutex
	startTime  time.Time
}

// NewCollector 創建新的指標收集器
func NewCollector() *MetricsCollector {
	loopMetrics := NewLoopMetrics()
	collector := &MetricsCollector{
		metrics:     make(map[string]Metric),
		loopMetrics: loopMetrics,
		startTime:   time.Now(),
	}

	// 註冊 LoopMetrics 中的所有指標
	collector.registerLoopMetrics(loopMetrics)

	return collector
}

// registerLoopMetrics 註冊 LoopMetrics 中的所有指標
func (c *MetricsCollector) registerLoopMetrics(lm *LoopMetrics) {
	c.Register(lm.TotalLoops)
	c.Register(lm.SuccessfulLoops)
	c.Register(lm.FailedLoops)
	c.Register(lm.TimeoutLoops)
	c.Register(lm.CircuitBreakerTrips)
	c.Register(lm.RetryAttempts)
	c.Register(lm.CLIExecutions)
	c.Register(lm.SDKExecutions)
	c.Register(lm.CurrentActiveLoops)
	c.Register(lm.CircuitBreakerState)
	c.Register(lm.ErrorRate)
	c.Register(lm.AverageLoopDuration)
	c.Register(lm.LoopExecutionTime)
	c.Register(lm.CLIExecutionTime)
	c.Register(lm.SDKExecutionTime)
	c.Register(lm.AIResponseTime)
}

// Register 註冊指標
func (c *MetricsCollector) Register(metric Metric) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.metrics[metric.Name()] = metric
}

// Unregister 取消註冊指標
func (c *MetricsCollector) Unregister(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.metrics, name)
}

// Get 獲取指標
func (c *MetricsCollector) Get(name string) Metric {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.metrics[name]
}

// GetLoopMetrics 獲取 LoopMetrics
func (c *MetricsCollector) GetLoopMetrics() *LoopMetrics {
	return c.loopMetrics
}

// GetAll 獲取所有指標
func (c *MetricsCollector) GetAll() map[string]Metric {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	result := make(map[string]Metric)
	for name, metric := range c.metrics {
		result[name] = metric
	}
	return result
}

// Reset 重置所有指標
func (c *MetricsCollector) Reset() {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	for _, metric := range c.metrics {
		metric.Reset()
	}
	c.startTime = time.Now()
}

// Summary 生成指標摘要
type Summary struct {
	Timestamp     time.Time              `json:"timestamp"`
	Uptime        string                 `json:"uptime"`
	TotalMetrics  int                    `json:"total_metrics"`
	Metrics       map[string]interface{} `json:"metrics"`
}

// GetSummary 獲取指標摘要
func (c *MetricsCollector) GetSummary() *Summary {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 更新計算指標
	c.loopMetrics.UpdateErrorRate()
	c.loopMetrics.UpdateAverageLoopDuration()

	metrics := make(map[string]interface{})
	for name, metric := range c.metrics {
		metrics[name] = map[string]interface{}{
			"type":  string(metric.Type()),
			"value": metric.Value(),
		}
	}

	return &Summary{
		Timestamp:    time.Now(),
		Uptime:       time.Since(c.startTime).String(),
		TotalMetrics: len(c.metrics),
		Metrics:      metrics,
	}
}

// ToJSON 轉換為 JSON 格式
func (s *Summary) ToJSON() (string, error) {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ToText 轉換為文字格式
func (s *Summary) ToText() string {
	result := fmt.Sprintf("=== Ralph Loop 指標摘要 ===\n")
	result += fmt.Sprintf("時間戳: %s\n", s.Timestamp.Format("2006-01-02 15:04:05"))
	result += fmt.Sprintf("執行時間: %s\n", s.Uptime)
	result += fmt.Sprintf("指標總數: %d\n\n", s.TotalMetrics)

	// 按類型分組顯示
	counters := make(map[string]interface{})
	gauges := make(map[string]interface{})
	timers := make(map[string]interface{})

	for name, metric := range s.Metrics {
		metricMap := metric.(map[string]interface{})
		switch metricMap["type"] {
		case "counter":
			counters[name] = metricMap["value"]
		case "gauge":
			gauges[name] = metricMap["value"]
		case "timer":
			timers[name] = metricMap["value"]
		}
	}

	// 顯示計數器
	if len(counters) > 0 {
		result += "📊 計數器:\n"
		for name, value := range counters {
			result += fmt.Sprintf("  %s: %v\n", name, value)
		}
		result += "\n"
	}

	// 顯示標量
	if len(gauges) > 0 {
		result += "📈 標量:\n"
		for name, value := range gauges {
			result += fmt.Sprintf("  %s: %.2f\n", name, value)
		}
		result += "\n"
	}

	// 顯示計時器
	if len(timers) > 0 {
		result += "⏱️  計時器:\n"
		for name, value := range timers {
			timerValue := value.(map[string]interface{})
			result += fmt.Sprintf("  %s:\n", name)
			result += fmt.Sprintf("    計數: %v\n", timerValue["count"])
			result += fmt.Sprintf("    最小值: %v ms\n", timerValue["min"])
			result += fmt.Sprintf("    最大值: %v ms\n", timerValue["max"])
			result += fmt.Sprintf("    平均值: %v ms\n", timerValue["avg"])
			result += fmt.Sprintf("    P50: %v ms\n", timerValue["p50"])
			result += fmt.Sprintf("    P95: %v ms\n", timerValue["p95"])
			result += fmt.Sprintf("    P99: %v ms\n", timerValue["p99"])
		}
		result += "\n"
	}

	return result
}

// 全域指標收集器
var GlobalCollector *MetricsCollector

// init 初始化全域指標收集器
func init() {
	GlobalCollector = NewCollector()
}

// 全域函數簡化使用

// RecordLoopStart 記錄迴圈開始
func RecordLoopStart() func() {
	GlobalCollector.GetLoopMetrics().TotalLoops.Inc()
	GlobalCollector.GetLoopMetrics().CurrentActiveLoops.Inc()
	return GlobalCollector.GetLoopMetrics().LoopExecutionTime.Start()
}

// RecordLoopSuccess 記錄迴圈成功
func RecordLoopSuccess(stopTimer func()) {
	if stopTimer != nil {
		stopTimer()
	}
	GlobalCollector.GetLoopMetrics().SuccessfulLoops.Inc()
	GlobalCollector.GetLoopMetrics().CurrentActiveLoops.Dec()
}

// RecordLoopFailure 記錄迴圈失敗
func RecordLoopFailure(stopTimer func()) {
	if stopTimer != nil {
		stopTimer()
	}
	GlobalCollector.GetLoopMetrics().FailedLoops.Inc()
	GlobalCollector.GetLoopMetrics().CurrentActiveLoops.Dec()
}

// RecordLoopTimeout 記錄迴圈超時
func RecordLoopTimeout(stopTimer func()) {
	if stopTimer != nil {
		stopTimer()
	}
	GlobalCollector.GetLoopMetrics().TimeoutLoops.Inc()
	GlobalCollector.GetLoopMetrics().CurrentActiveLoops.Dec()
}

// RecordCircuitBreakerTrip 記錄熔斷器觸發
func RecordCircuitBreakerTrip() {
	GlobalCollector.GetLoopMetrics().CircuitBreakerTrips.Inc()
}

// RecordRetryAttempt 記錄重試嘗試
func RecordRetryAttempt() {
	GlobalCollector.GetLoopMetrics().RetryAttempts.Inc()
}

// RecordCLIExecution 記錄 CLI 執行
func RecordCLIExecution(duration time.Duration) {
	GlobalCollector.GetLoopMetrics().CLIExecutions.Inc()
	GlobalCollector.GetLoopMetrics().CLIExecutionTime.Record(duration)
}

// RecordSDKExecution 記錄 SDK 執行
func RecordSDKExecution(duration time.Duration) {
	GlobalCollector.GetLoopMetrics().SDKExecutions.Inc()
	GlobalCollector.GetLoopMetrics().SDKExecutionTime.Record(duration)
}

// SetCircuitBreakerState 設置熔斷器狀態
func SetCircuitBreakerState(state int) {
	GlobalCollector.GetLoopMetrics().CircuitBreakerState.Set(float64(state))
}

// GetSummary 獲取全域指標摘要
func GetSummary() *Summary {
	return GlobalCollector.GetSummary()
}

// ResetGlobalMetrics 重置全域指標
func ResetGlobalMetrics() {
	GlobalCollector.Reset()
}