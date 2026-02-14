package ghcopilot

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// QuotaManager 配額管理器
//
// 管理API調用限制、執行時間限制、並發數限制等配額
type QuotaManager struct {
	quotas    map[string]*TenantQuota // 租戶配額映射
	config    *QuotaConfig           // 配額配置
	usage     map[string]*UsageStats  // 使用統計
	mu        sync.RWMutex           // 讀寫鎖
	started   bool                   // 是否已啟動
	resetTicker *time.Ticker         // 重置定時器
	stopCh    chan struct{}          // 停止通道
}

// QuotaConfig 配額配置
type QuotaConfig struct {
	DefaultLimits    map[string]interface{} // 預設配額限制
	ResetInterval    time.Duration         // 重置間隔
	AlertThreshold   float64              // 告警閾值 (0.0-1.0)
	EnableAlerts     bool                 // 是否啟用告警
}

// TenantQuota 租戶配額
type TenantQuota struct {
	TenantID              string    // 租戶ID
	APICallsPerHour      int64     // 每小時API調用數限制
	APICallsRemaining    int64     // 剩餘API調用數
	MaxConcurrentLoops   int       // 最大併發迴圈數
	CurrentConcurrentLoops int     // 當前併發迴圈數
	MaxLoopDuration      time.Duration // 最大迴圈執行時間
	StorageLimitMB       int64     // 儲存限制 (MB)
	StorageUsedMB        int64     // 已使用儲存 (MB)
	LastReset           time.Time  // 上次重置時間
	mu                  sync.RWMutex // 配額級鎖
}

// UsageStats 使用統計
type UsageStats struct {
	TotalAPICalls        int64     // 總API調用數
	SuccessfulAPICalls   int64     // 成功API調用數
	FailedAPICalls       int64     // 失敗API調用數
	TotalLoops          int64     // 總迴圈數
	SuccessfulLoops     int64     // 成功迴圈數
	FailedLoops         int64     // 失敗迴圈數
	TotalExecutionTime  time.Duration // 總執行時間
	LastUpdated         time.Time // 最後更新時間
	mu                  sync.RWMutex  // 統計級鎖
}

// DefaultQuotaConfig 返回預設配額配置
func DefaultQuotaConfig() *QuotaConfig {
	return &QuotaConfig{
		DefaultLimits: map[string]interface{}{
			"api_calls_per_hour":    int64(1000),
			"max_concurrent_loops":  10,
			"max_loop_duration":     "30m",
			"storage_limit_mb":      int64(1024),
		},
		ResetInterval:  time.Hour,
		AlertThreshold: 0.9, // 90%使用率時告警
		EnableAlerts:   true,
	}
}

// NewQuotaManager 創建新的配額管理器
func NewQuotaManager(config *QuotaConfig) *QuotaManager {
	if config == nil {
		config = DefaultQuotaConfig()
	}

	return &QuotaManager{
		quotas:  make(map[string]*TenantQuota),
		config:  config,
		usage:   make(map[string]*UsageStats),
		started: false,
		stopCh:  make(chan struct{}),
	}
}

// Start 啟動配額管理器
func (qm *QuotaManager) Start(ctx context.Context) error {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	if qm.started {
		return fmt.Errorf("quota manager already started")
	}

	// 啟動配額重置定時器
	qm.resetTicker = time.NewTicker(qm.config.ResetInterval)
	
	go qm.resetLoop()

	qm.started = true
	return nil
}

// Stop 停止配額管理器
func (qm *QuotaManager) Stop(ctx context.Context) error {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	if !qm.started {
		return nil
	}

	if qm.resetTicker != nil {
		qm.resetTicker.Stop()
	}

	close(qm.stopCh)
	qm.started = false
	return nil
}

// InitializeTenantQuota 初始化租戶配額
func (qm *QuotaManager) InitializeTenantQuota(tenantID string, customLimits map[string]interface{}) error {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	if !qm.started {
		return fmt.Errorf("quota manager not started")
	}

	// 檢查租戶是否已存在配額
	if _, exists := qm.quotas[tenantID]; exists {
		return fmt.Errorf("quota for tenant %s already exists", tenantID)
	}

	// 使用預設配額或自訂配額
	limits := qm.config.DefaultLimits
	if customLimits != nil {
		limits = customLimits
	}

	// 解析配額限制
	maxLoopDuration, _ := time.ParseDuration(limits["max_loop_duration"].(string))

	quota := &TenantQuota{
		TenantID:              tenantID,
		APICallsPerHour:      limits["api_calls_per_hour"].(int64),
		APICallsRemaining:    limits["api_calls_per_hour"].(int64),
		MaxConcurrentLoops:   limits["max_concurrent_loops"].(int),
		CurrentConcurrentLoops: 0,
		MaxLoopDuration:      maxLoopDuration,
		StorageLimitMB:       limits["storage_limit_mb"].(int64),
		StorageUsedMB:        0,
		LastReset:           time.Now(),
	}

	qm.quotas[tenantID] = quota

	// 初始化使用統計
	qm.usage[tenantID] = &UsageStats{
		LastUpdated: time.Now(),
	}

	return nil
}

// CheckQuota 檢查配額是否允許操作
func (qm *QuotaManager) CheckQuota(tenantID, operation string) error {
	qm.mu.RLock()
	quota, exists := qm.quotas[tenantID]
	qm.mu.RUnlock()

	if !exists {
		// 如果租戶配額不存在，使用預設配額初始化
		if err := qm.InitializeTenantQuota(tenantID, nil); err != nil {
			return fmt.Errorf("failed to initialize quota for tenant %s: %w", tenantID, err)
		}
		qm.mu.RLock()
		quota = qm.quotas[tenantID]
		qm.mu.RUnlock()
	}

	quota.mu.RLock()
	defer quota.mu.RUnlock()

	switch operation {
	case "api_call":
		if quota.APICallsRemaining <= 0 {
			return fmt.Errorf("API call quota exceeded for tenant %s", tenantID)
		}

	case "start_loop":
		if quota.CurrentConcurrentLoops >= quota.MaxConcurrentLoops {
			return fmt.Errorf("concurrent loop quota exceeded for tenant %s (%d/%d)", 
				tenantID, quota.CurrentConcurrentLoops, quota.MaxConcurrentLoops)
		}

	case "storage_write":
		// 這裡可以添加儲存空間檢查邏輯
		if quota.StorageUsedMB >= quota.StorageLimitMB {
			return fmt.Errorf("storage quota exceeded for tenant %s (%d/%d MB)", 
				tenantID, quota.StorageUsedMB, quota.StorageLimitMB)
		}

	default:
		// 未知操作，直接允許
	}

	return nil
}

// ConsumeQuota 消耗配額
func (qm *QuotaManager) ConsumeQuota(tenantID, operation string, amount int64) error {
	qm.mu.RLock()
	quota, exists := qm.quotas[tenantID]
	qm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("quota for tenant %s not found", tenantID)
	}

	quota.mu.Lock()
	defer quota.mu.Unlock()

	switch operation {
	case "api_call":
		if quota.APICallsRemaining < amount {
			return fmt.Errorf("insufficient API calls remaining for tenant %s", tenantID)
		}
		quota.APICallsRemaining -= amount

	case "start_loop":
		quota.CurrentConcurrentLoops++
		if quota.CurrentConcurrentLoops > quota.MaxConcurrentLoops {
			quota.CurrentConcurrentLoops--
			return fmt.Errorf("concurrent loop limit exceeded for tenant %s", tenantID)
		}

	case "storage_write":
		quota.StorageUsedMB += amount
		if quota.StorageUsedMB > quota.StorageLimitMB {
			quota.StorageUsedMB -= amount
			return fmt.Errorf("storage limit exceeded for tenant %s", tenantID)
		}

	default:
		return fmt.Errorf("unknown operation: %s", operation)
	}

	return nil
}

// ReleaseQuota 釋放配額
func (qm *QuotaManager) ReleaseQuota(tenantID, operation string, amount int64) error {
	qm.mu.RLock()
	quota, exists := qm.quotas[tenantID]
	qm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("quota for tenant %s not found", tenantID)
	}

	quota.mu.Lock()
	defer quota.mu.Unlock()

	switch operation {
	case "end_loop":
		if quota.CurrentConcurrentLoops > 0 {
			quota.CurrentConcurrentLoops--
		}

	case "storage_delete":
		quota.StorageUsedMB -= amount
		if quota.StorageUsedMB < 0 {
			quota.StorageUsedMB = 0
		}

	default:
		return fmt.Errorf("unknown release operation: %s", operation)
	}

	return nil
}

// RecordUsage 記錄使用量
func (qm *QuotaManager) RecordUsage(usage *QuotaUsage) {
	qm.mu.RLock()
	stats, exists := qm.usage[usage.TenantID]
	qm.mu.RUnlock()

	if !exists {
		// 初始化使用統計
		qm.mu.Lock()
		qm.usage[usage.TenantID] = &UsageStats{
			LastUpdated: time.Now(),
		}
		stats = qm.usage[usage.TenantID]
		qm.mu.Unlock()
	}

	stats.mu.Lock()
	defer stats.mu.Unlock()

	switch usage.Operation {
	case "api_call":
		stats.TotalAPICalls++
		if usage.Success {
			stats.SuccessfulAPICalls++
		} else {
			stats.FailedAPICalls++
		}

	case "loop_execution":
		stats.TotalLoops++
		stats.TotalExecutionTime += usage.Duration
		if usage.Success {
			stats.SuccessfulLoops++
		} else {
			stats.FailedLoops++
		}
	}

	stats.LastUpdated = time.Now()
}

// GetQuotaStatus 獲取租戶配額狀態
func (qm *QuotaManager) GetQuotaStatus(tenantID string) (map[string]interface{}, error) {
	qm.mu.RLock()
	quota, exists := qm.quotas[tenantID]
	qm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("quota for tenant %s not found", tenantID)
	}

	quota.mu.RLock()
	defer quota.mu.RUnlock()

	status := map[string]interface{}{
		"tenant_id":               quota.TenantID,
		"api_calls_per_hour":     quota.APICallsPerHour,
		"api_calls_remaining":    quota.APICallsRemaining,
		"api_calls_used":         quota.APICallsPerHour - quota.APICallsRemaining,
		"api_calls_usage_percent": float64(quota.APICallsPerHour-quota.APICallsRemaining) / float64(quota.APICallsPerHour) * 100,
		"max_concurrent_loops":   quota.MaxConcurrentLoops,
		"current_concurrent_loops": quota.CurrentConcurrentLoops,
		"concurrent_usage_percent": float64(quota.CurrentConcurrentLoops) / float64(quota.MaxConcurrentLoops) * 100,
		"max_loop_duration":      quota.MaxLoopDuration.String(),
		"storage_limit_mb":       quota.StorageLimitMB,
		"storage_used_mb":        quota.StorageUsedMB,
		"storage_usage_percent":  float64(quota.StorageUsedMB) / float64(quota.StorageLimitMB) * 100,
		"last_reset":            quota.LastReset,
	}

	return status, nil
}

// GetUsageStats 獲取使用統計
func (qm *QuotaManager) GetUsageStats(tenantID string) (map[string]interface{}, error) {
	qm.mu.RLock()
	stats, exists := qm.usage[tenantID]
	qm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("usage stats for tenant %s not found", tenantID)
	}

	stats.mu.RLock()
	defer stats.mu.RUnlock()

	result := map[string]interface{}{
		"tenant_id":              tenantID,
		"total_api_calls":        stats.TotalAPICalls,
		"successful_api_calls":   stats.SuccessfulAPICalls,
		"failed_api_calls":       stats.FailedAPICalls,
		"total_loops":           stats.TotalLoops,
		"successful_loops":      stats.SuccessfulLoops,
		"failed_loops":          stats.FailedLoops,
		"total_execution_time":  stats.TotalExecutionTime.String(),
		"last_updated":          stats.LastUpdated,
	}

	// 計算成功率
	if stats.TotalAPICalls > 0 {
		result["api_success_rate"] = float64(stats.SuccessfulAPICalls) / float64(stats.TotalAPICalls) * 100
	}
	if stats.TotalLoops > 0 {
		result["loop_success_rate"] = float64(stats.SuccessfulLoops) / float64(stats.TotalLoops) * 100
		result["avg_execution_time"] = (stats.TotalExecutionTime / time.Duration(stats.TotalLoops)).String()
	}

	return result, nil
}

// GetStatus 獲取配額管理器狀態
func (qm *QuotaManager) GetStatus() map[string]interface{} {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	status := map[string]interface{}{
		"started":          qm.started,
		"total_tenants":    len(qm.quotas),
		"reset_interval":   qm.config.ResetInterval.String(),
		"alert_threshold":  qm.config.AlertThreshold,
		"alerts_enabled":   qm.config.EnableAlerts,
	}

	// 統計配額使用情況
	totalQuotas := len(qm.quotas)
	alertCount := 0

	for _, quota := range qm.quotas {
		quota.mu.RLock()
		
		// 檢查是否需要告警
		apiUsageRate := float64(quota.APICallsPerHour-quota.APICallsRemaining) / float64(quota.APICallsPerHour)
		concurrentUsageRate := float64(quota.CurrentConcurrentLoops) / float64(quota.MaxConcurrentLoops)
		storageUsageRate := float64(quota.StorageUsedMB) / float64(quota.StorageLimitMB)

		if apiUsageRate >= qm.config.AlertThreshold || 
		   concurrentUsageRate >= qm.config.AlertThreshold ||
		   storageUsageRate >= qm.config.AlertThreshold {
			alertCount++
		}
		
		quota.mu.RUnlock()
	}

	status["tenants_over_threshold"] = alertCount
	status["alert_percentage"] = float64(alertCount) / float64(totalQuotas) * 100

	return status
}

// ResetTenantQuota 重置租戶配額
func (qm *QuotaManager) ResetTenantQuota(tenantID string) error {
	qm.mu.RLock()
	quota, exists := qm.quotas[tenantID]
	qm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("quota for tenant %s not found", tenantID)
	}

	quota.mu.Lock()
	defer quota.mu.Unlock()

	// 重置API調用配額
	quota.APICallsRemaining = quota.APICallsPerHour
	quota.LastReset = time.Now()

	return nil
}

// ResetAllQuotas 重置所有租戶的配額
func (qm *QuotaManager) ResetAllQuotas() {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	for _, quota := range qm.quotas {
		quota.mu.Lock()
		quota.APICallsRemaining = quota.APICallsPerHour
		quota.LastReset = time.Now()
		quota.mu.Unlock()
	}
}

// 私有方法

// resetLoop 配額重置循環
func (qm *QuotaManager) resetLoop() {
	for {
		select {
		case <-qm.resetTicker.C:
			qm.ResetAllQuotas()
		case <-qm.stopCh:
			return
		}
	}
}

// UpdateTenantQuota 更新租戶配額限制
func (qm *QuotaManager) UpdateTenantQuota(tenantID string, newLimits map[string]interface{}) error {
	qm.mu.RLock()
	quota, exists := qm.quotas[tenantID]
	qm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("quota for tenant %s not found", tenantID)
	}

	quota.mu.Lock()
	defer quota.mu.Unlock()

	// 更新配額限制
	if apiCalls, ok := newLimits["api_calls_per_hour"].(int64); ok {
		// 按比例調整剩餘額度
		ratio := float64(apiCalls) / float64(quota.APICallsPerHour)
		quota.APICallsRemaining = int64(float64(quota.APICallsRemaining) * ratio)
		quota.APICallsPerHour = apiCalls
	}

	if maxConcurrent, ok := newLimits["max_concurrent_loops"].(int); ok {
		quota.MaxConcurrentLoops = maxConcurrent
	}

	if maxDuration, ok := newLimits["max_loop_duration"].(string); ok {
		if duration, err := time.ParseDuration(maxDuration); err == nil {
			quota.MaxLoopDuration = duration
		}
	}

	if storageLimit, ok := newLimits["storage_limit_mb"].(int64); ok {
		quota.StorageLimitMB = storageLimit
	}

	return nil
}

// ListTenantQuotas 列出所有租戶配額
func (qm *QuotaManager) ListTenantQuotas() []string {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	tenantIDs := make([]string, 0, len(qm.quotas))
	for tenantID := range qm.quotas {
		tenantIDs = append(tenantIDs, tenantID)
	}

	return tenantIDs
}