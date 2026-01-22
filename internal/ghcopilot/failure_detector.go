package ghcopilot

import (
	"sync"
	"time"
)

// FailureType 定義故障類型
type FailureType int

const (
	// FailureNone 無故障
	FailureNone FailureType = iota
	// FailureTimeout 逾時故障
	FailureTimeout
	// FailureErrorRate 錯誤率過高
	FailureErrorRate
	// FailureHealthCheck 健康檢查失敗
	FailureHealthCheck
	// FailureConnection 連接故障
	FailureConnection
)

// String 返回故障類型的字串表示
func (f FailureType) String() string {
	switch f {
	case FailureNone:
		return "none"
	case FailureTimeout:
		return "timeout"
	case FailureErrorRate:
		return "error_rate"
	case FailureHealthCheck:
		return "health_check"
	case FailureConnection:
		return "connection"
	default:
		return "unknown"
	}
}

// FailureDetector 故障檢測器介面
type FailureDetector interface {
	// Detect 檢測是否發生故障
	Detect(err error, duration time.Duration) bool
	// GetType 取得檢測器類型
	GetType() FailureType
	// Reset 重置檢測器狀態
	Reset()
}

// TimeoutDetector 逾時檢測器
type TimeoutDetector struct {
	threshold            time.Duration
	consecutiveThreshold int
	consecutiveCount     int
	mu                   sync.Mutex
}

// NewTimeoutDetector 建立新的逾時檢測器
func NewTimeoutDetector(threshold time.Duration) *TimeoutDetector {
	return &TimeoutDetector{
		threshold:            threshold,
		consecutiveThreshold: 3,
		consecutiveCount:     0,
	}
}

// WithConsecutiveThreshold 設定連續逾時閾值
func (d *TimeoutDetector) WithConsecutiveThreshold(count int) *TimeoutDetector {
	d.consecutiveThreshold = count
	return d
}

// Detect 檢測是否發生逾時故障
func (d *TimeoutDetector) Detect(err error, duration time.Duration) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	if duration > d.threshold {
		d.consecutiveCount++
		if d.consecutiveCount >= d.consecutiveThreshold {
			return true
		}
	} else {
		d.consecutiveCount = 0
	}

	return false
}

// GetType 取得檢測器類型
func (d *TimeoutDetector) GetType() FailureType {
	return FailureTimeout
}

// Reset 重置檢測器狀態
func (d *TimeoutDetector) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.consecutiveCount = 0
}

// GetConsecutiveCount 取得連續逾時次數
func (d *TimeoutDetector) GetConsecutiveCount() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.consecutiveCount
}

// ErrorRateDetector 錯誤率檢測器
type ErrorRateDetector struct {
	windowSize int
	threshold  float64
	window     []bool // true = 成功, false = 失敗
	index      int
	mu         sync.Mutex
}

// NewErrorRateDetector 建立新的錯誤率檢測器
func NewErrorRateDetector(windowSize int, threshold float64) *ErrorRateDetector {
	return &ErrorRateDetector{
		windowSize: windowSize,
		threshold:  threshold,
		window:     make([]bool, windowSize),
		index:      0,
	}
}

// Detect 檢測錯誤率是否過高
func (d *ErrorRateDetector) Detect(err error, duration time.Duration) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	// 記錄結果
	d.window[d.index%d.windowSize] = err == nil
	d.index++

	// 計算錯誤率
	if d.index < d.windowSize {
		// 窗口未滿，不檢測
		return false
	}

	failures := 0
	for _, success := range d.window {
		if !success {
			failures++
		}
	}

	errorRate := float64(failures) / float64(d.windowSize)
	return errorRate > d.threshold
}

// GetType 取得檢測器類型
func (d *ErrorRateDetector) GetType() FailureType {
	return FailureErrorRate
}

// Reset 重置檢測器狀態
func (d *ErrorRateDetector) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.window = make([]bool, d.windowSize)
	d.index = 0
}

// GetErrorRate 取得當前錯誤率
func (d *ErrorRateDetector) GetErrorRate() float64 {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.index == 0 {
		return 0
	}

	count := d.index
	if count > d.windowSize {
		count = d.windowSize
	}

	failures := 0
	for i := 0; i < count; i++ {
		if !d.window[i] {
			failures++
		}
	}

	return float64(failures) / float64(count)
}

// HealthCheckDetector 健康檢查檢測器
type HealthCheckDetector struct {
	checkInterval   time.Duration
	maxUnhealthy    int
	unhealthyCount  int
	lastCheck       time.Time
	healthCheckFunc func() bool
	mu              sync.Mutex
}

// NewHealthCheckDetector 建立新的健康檢查檢測器
func NewHealthCheckDetector(interval time.Duration, maxUnhealthy int) *HealthCheckDetector {
	return &HealthCheckDetector{
		checkInterval:   interval,
		maxUnhealthy:    maxUnhealthy,
		unhealthyCount:  0,
		lastCheck:       time.Time{},
		healthCheckFunc: func() bool { return true },
	}
}

// SetHealthCheckFunc 設定健康檢查函式
func (d *HealthCheckDetector) SetHealthCheckFunc(fn func() bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.healthCheckFunc = fn
}

// Detect 執行健康檢查並判斷是否故障
func (d *HealthCheckDetector) Detect(err error, duration time.Duration) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	if now.Sub(d.lastCheck) < d.checkInterval && !d.lastCheck.IsZero() {
		// 未到檢查間隔
		return d.unhealthyCount >= d.maxUnhealthy
	}

	d.lastCheck = now

	// 執行健康檢查
	if d.healthCheckFunc() {
		d.unhealthyCount = 0
		return false
	}

	d.unhealthyCount++
	return d.unhealthyCount >= d.maxUnhealthy
}

// GetType 取得檢測器類型
func (d *HealthCheckDetector) GetType() FailureType {
	return FailureHealthCheck
}

// Reset 重置檢測器狀態
func (d *HealthCheckDetector) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.unhealthyCount = 0
	d.lastCheck = time.Time{}
}

// GetUnhealthyCount 取得連續不健康次數
func (d *HealthCheckDetector) GetUnhealthyCount() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.unhealthyCount
}

// ConnectionDetector 連接故障檢測器
type ConnectionDetector struct {
	failurePatterns  []string
	consecutiveCount int
	threshold        int
	mu               sync.Mutex
}

// NewConnectionDetector 建立新的連接故障檢測器
func NewConnectionDetector(threshold int) *ConnectionDetector {
	return &ConnectionDetector{
		failurePatterns: []string{
			"connection refused",
			"connection reset",
			"connection timeout",
			"no such host",
			"network unreachable",
			"EOF",
		},
		threshold:        threshold,
		consecutiveCount: 0,
	}
}

// AddPattern 添加故障模式
func (d *ConnectionDetector) AddPattern(pattern string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.failurePatterns = append(d.failurePatterns, pattern)
}

// Detect 檢測是否發生連接故障
func (d *ConnectionDetector) Detect(err error, duration time.Duration) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	if err == nil {
		d.consecutiveCount = 0
		return false
	}

	errMsg := err.Error()
	isConnectionError := false

	for _, pattern := range d.failurePatterns {
		if containsString(errMsg, pattern) {
			isConnectionError = true
			break
		}
	}

	if isConnectionError {
		d.consecutiveCount++
		if d.consecutiveCount >= d.threshold {
			return true
		}
	} else {
		d.consecutiveCount = 0
	}

	return false
}

// GetType 取得檢測器類型
func (d *ConnectionDetector) GetType() FailureType {
	return FailureConnection
}

// Reset 重置檢測器狀態
func (d *ConnectionDetector) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.consecutiveCount = 0
}

// GetConsecutiveCount 取得連續故障次數
func (d *ConnectionDetector) GetConsecutiveCount() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.consecutiveCount
}

// MultiDetector 多檢測器組合
type MultiDetector struct {
	detectors []FailureDetector
	mu        sync.RWMutex
}

// NewMultiDetector 建立新的多檢測器組合
func NewMultiDetector(detectors ...FailureDetector) *MultiDetector {
	return &MultiDetector{
		detectors: detectors,
	}
}

// AddDetector 添加檢測器
func (d *MultiDetector) AddDetector(detector FailureDetector) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.detectors = append(d.detectors, detector)
}

// Detect 執行所有檢測器，任一檢測到故障即返回 true
func (d *MultiDetector) Detect(err error, duration time.Duration) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	for _, detector := range d.detectors {
		if detector.Detect(err, duration) {
			return true
		}
	}
	return false
}

// DetectWithType 執行所有檢測器並返回檢測到的故障類型
func (d *MultiDetector) DetectWithType(err error, duration time.Duration) (bool, FailureType) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	for _, detector := range d.detectors {
		if detector.Detect(err, duration) {
			return true, detector.GetType()
		}
	}
	return false, FailureNone
}

// GetType 取得檢測器類型（返回第一個的類型）
func (d *MultiDetector) GetType() FailureType {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if len(d.detectors) > 0 {
		return d.detectors[0].GetType()
	}
	return FailureNone
}

// Reset 重置所有檢測器
func (d *MultiDetector) Reset() {
	d.mu.RLock()
	defer d.mu.RUnlock()

	for _, detector := range d.detectors {
		detector.Reset()
	}
}

// GetDetectorCount 取得檢測器數量
func (d *MultiDetector) GetDetectorCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.detectors)
}

// FailureDetectorConfig 故障檢測器配置
type FailureDetectorConfig struct {
	// 逾時檢測
	EnableTimeout        bool
	TimeoutThreshold     time.Duration
	TimeoutConsecutive   int
	// 錯誤率檢測
	EnableErrorRate      bool
	ErrorRateWindow      int
	ErrorRateThreshold   float64
	// 健康檢查
	EnableHealthCheck    bool
	HealthCheckInterval  time.Duration
	HealthCheckMaxFails  int
	// 連接檢測
	EnableConnection     bool
	ConnectionThreshold  int
}

// DefaultFailureDetectorConfig 返回預設的故障檢測器配置
func DefaultFailureDetectorConfig() *FailureDetectorConfig {
	return &FailureDetectorConfig{
		EnableTimeout:       true,
		TimeoutThreshold:    5 * time.Second,
		TimeoutConsecutive:  3,
		EnableErrorRate:     true,
		ErrorRateWindow:     10,
		ErrorRateThreshold:  0.5,
		EnableHealthCheck:   false,
		HealthCheckInterval: 30 * time.Second,
		HealthCheckMaxFails: 3,
		EnableConnection:    true,
		ConnectionThreshold: 3,
	}
}

// BuildMultiDetector 根據配置建構多檢測器
func BuildMultiDetector(config *FailureDetectorConfig) *MultiDetector {
	detectors := make([]FailureDetector, 0)

	if config.EnableTimeout {
		detector := NewTimeoutDetector(config.TimeoutThreshold).
			WithConsecutiveThreshold(config.TimeoutConsecutive)
		detectors = append(detectors, detector)
	}

	if config.EnableErrorRate {
		detector := NewErrorRateDetector(config.ErrorRateWindow, config.ErrorRateThreshold)
		detectors = append(detectors, detector)
	}

	if config.EnableHealthCheck {
		detector := NewHealthCheckDetector(config.HealthCheckInterval, config.HealthCheckMaxFails)
		detectors = append(detectors, detector)
	}

	if config.EnableConnection {
		detector := NewConnectionDetector(config.ConnectionThreshold)
		detectors = append(detectors, detector)
	}

	return NewMultiDetector(detectors...)
}
