package ghcopilot

import (
	"context"
	"fmt"
	"runtime"
	"time"
)

// 性能優化相關方法（T2-012）

// GetPerformanceStats 獲取性能統計信息
func (c *RalphLoopClient) GetPerformanceStats() map[string]interface{} {
	stats := make(map[string]interface{})
	
	// 記憶體統計
	memStats := GetMemoryStats()
	stats["memory"] = map[string]interface{}{
		"allocated_bytes":   memStats.AllocatedBytes,
		"total_alloc_bytes": memStats.TotalAllocBytes,
		"gc_cycles":         memStats.GCCycles,
		"goroutine_count":   memStats.GoroutineCount,
		"heap_in_use":       memStats.HeapInUse,
	}
	
	// 緩存統計
	if c.cacheManager != nil {
		cacheStats := c.cacheManager.GetStats()
		stats["cache"] = cacheStats
	}
	
	// 併發統計
	if c.concurrentManager != nil {
		concurrentStats := c.concurrentManager.GetStats()
		stats["concurrency"] = map[string]interface{}{
			"max_workers":       concurrentStats.MaxWorkers,
			"active_workers":    concurrentStats.ActiveWorkers,
			"queued_tasks":      concurrentStats.QueuedTasks,
			"available_workers": concurrentStats.AvailableWorkers,
			"is_running":        concurrentStats.IsRunning,
		}
	}
	
	// 配置信息
	stats["config"] = map[string]interface{}{
		"enable_caching":         c.config.EnableCaching,
		"enable_concurrency":     c.config.EnableConcurrency,
		"enable_memory_pool":     c.config.EnableMemoryPool,
		"memory_optimization":    c.config.MemoryOptimization,
		"max_concurrent_workers": c.config.MaxConcurrentWorkers,
		"cache_max_size":         c.config.CacheMaxSize,
		"cache_ttl":              c.config.CacheTTL.String(),
	}
	
	return stats
}

// SetCacheEnabled 啟用或禁用緩存
func (c *RalphLoopClient) SetCacheEnabled(enabled bool) error {
	if !c.initialized {
		return fmt.Errorf("client not initialized")
	}
	
	c.config.EnableCaching = enabled
	
	if enabled && c.cacheManager == nil {
		// 創建緩存管理器
		cacheConfig := &CacheConfig{
			MaxSize:         c.config.CacheMaxSize,
			TTL:             c.config.CacheTTL,
			CleanupInterval: 5 * time.Minute,
			EnableCaching:   true,
		}
		c.cacheManager = NewCacheManager(cacheConfig)
	} else if !enabled && c.cacheManager != nil {
		// 關閉緩存管理器
		c.cacheManager.Close()
		c.cacheManager = nil
	}
	
	return nil
}

// ClearCache 清空所有緩存
func (c *RalphLoopClient) ClearCache() error {
	if c.cacheManager != nil {
		c.cacheManager.Clear()
		return nil
	}
	return fmt.Errorf("cache not enabled")
}

// StartConcurrentManager 啟動併發執行管理器
func (c *RalphLoopClient) StartConcurrentManager() error {
	if c.concurrentManager == nil {
		maxWorkers := c.config.MaxConcurrentWorkers
		if maxWorkers <= 0 {
			maxWorkers = runtime.NumCPU()
		}
		c.concurrentManager = NewConcurrentExecutionManager(maxWorkers, maxWorkers*10)
	}
	
	return c.concurrentManager.Start()
}

// StopConcurrentManager 停止併發執行管理器
func (c *RalphLoopClient) StopConcurrentManager() {
	if c.concurrentManager != nil {
		c.concurrentManager.Stop()
	}
}

// SubmitConcurrentTask 提交併發任務
func (c *RalphLoopClient) SubmitConcurrentTask(taskID string, prompt string, maxLoops int, callback func(*ConcurrentExecutionResult)) error {
	if c.concurrentManager == nil || !c.concurrentManager.GetStats().IsRunning {
		return fmt.Errorf("concurrent manager not running")
	}
	
	task := &ExecutionTask{
		ID:       taskID,
		Prompt:   prompt,
		MaxLoops: maxLoops,
		Context:  context.Background(),
		Config:   c.config,
		Callback: callback,
	}
	
	return c.concurrentManager.SubmitTask(task)
}

// GetConcurrentResult 獲取併發任務結果
func (c *RalphLoopClient) GetConcurrentResult() *ConcurrentExecutionResult {
	if c.concurrentManager == nil {
		return nil
	}
	
	return c.concurrentManager.GetResult()
}

// ForceGarbageCollection 強制執行垃圾回收
func (c *RalphLoopClient) ForceGarbageCollection() {
	ForceGC()
}

// EnablePerformanceOptimization 啟用性能優化
func (c *RalphLoopClient) EnablePerformanceOptimization() error {
	if !c.initialized {
		return fmt.Errorf("client not initialized")
	}
	
	// 啟用記憶體池
	if c.memoryPool == nil {
		c.memoryPool = NewMemoryPool()
		c.config.EnableMemoryPool = true
	}
	
	// 啟用緩存
	if c.cacheManager == nil {
		cacheConfig := &CacheConfig{
			MaxSize:         c.config.CacheMaxSize,
			TTL:             c.config.CacheTTL,
			CleanupInterval: 5 * time.Minute,
			EnableCaching:   true,
		}
		c.cacheManager = NewCacheManager(cacheConfig)
		c.config.EnableCaching = true
	}
	
	// 啟用記憶體優化
	c.config.MemoryOptimization = true
	
	return nil
}

// DisablePerformanceOptimization 禁用性能優化
func (c *RalphLoopClient) DisablePerformanceOptimization() {
	// 禁用緩存
	if c.cacheManager != nil {
		c.cacheManager.Close()
		c.cacheManager = nil
	}
	c.config.EnableCaching = false
	
	// 禁用記憶體池（保留實例但標記為禁用）
	c.config.EnableMemoryPool = false
	
	// 停止併發管理器
	if c.concurrentManager != nil {
		c.concurrentManager.Stop()
		c.concurrentManager = nil
	}
	c.config.EnableConcurrency = false
	
	// 禁用記憶體優化
	c.config.MemoryOptimization = false
}

// WithPerformanceOptimization 客戶端建構器的性能優化擴展
func (b *ClientBuilder) WithPerformanceOptimization() *ClientBuilder {
	b.config.EnableCaching = true
	b.config.EnableMemoryPool = true
	b.config.MemoryOptimization = true
	return b
}

// WithCaching 啟用緩存
func (b *ClientBuilder) WithCaching(maxSize int, ttl time.Duration) *ClientBuilder {
	b.config.EnableCaching = true
	b.config.CacheMaxSize = maxSize
	b.config.CacheTTL = ttl
	return b
}

// WithConcurrency 啟用併發執行
func (b *ClientBuilder) WithConcurrency(maxWorkers int) *ClientBuilder {
	b.config.EnableConcurrency = true
	b.config.MaxConcurrentWorkers = maxWorkers
	return b
}

// WithMemoryPool 啟用記憶體池
func (b *ClientBuilder) WithMemoryPool() *ClientBuilder {
	b.config.EnableMemoryPool = true
	return b
}

// WithMemoryOptimization 啟用記憶體優化
func (b *ClientBuilder) WithMemoryOptimization() *ClientBuilder {
	b.config.MemoryOptimization = true
	return b
}