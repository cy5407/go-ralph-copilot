package ghcopilot

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

// CacheItem 緩存項目
type CacheItem struct {
	Value      interface{} // 緩存的值
	ExpiresAt  time.Time   // 過期時間
	AccessedAt time.Time   // 最後訪問時間
	AccessCount int64      // 訪問次數
}

// IsExpired 檢查緩存項目是否已過期
func (ci *CacheItem) IsExpired() bool {
	return time.Now().After(ci.ExpiresAt)
}

// ResponseCache AI 回應緩存系統
//
// 使用 LRU (Least Recently Used) 策略管理緩存，
// 支援 TTL (Time To Live) 自動過期
type ResponseCache struct {
	items    map[string]*CacheItem
	order    []string      // LRU 順序，最新的在後面
	maxSize  int           // 最大緩存項目數
	ttl      time.Duration // 緩存生存時間
	mu       sync.RWMutex  // 讀寫鎖
	hitCount int64         // 命中次數
	requests int64         // 總請求次數
}

// NewResponseCache 創建新的回應緩存
func NewResponseCache(maxSize int, ttl time.Duration) *ResponseCache {
	return &ResponseCache{
		items:   make(map[string]*CacheItem),
		order:   make([]string, 0, maxSize),
		maxSize: maxSize,
		ttl:     ttl,
	}
}

// generateKey 為 prompt 和相關參數生成緩存鍵
func (rc *ResponseCache) generateKey(prompt string, model string, options map[string]interface{}) string {
	h := sha256.New()
	h.Write([]byte(prompt))
	h.Write([]byte(model))
	
	// 將選項也納入鍵的計算
	for k, v := range options {
		h.Write([]byte(k))
		if s, ok := v.(string); ok {
			h.Write([]byte(s))
		}
	}
	
	return hex.EncodeToString(h.Sum(nil))
}

// Get 從緩存中獲取回應
func (rc *ResponseCache) Get(prompt string, model string, options map[string]interface{}) (interface{}, bool) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	
	rc.requests++
	
	key := rc.generateKey(prompt, model, options)
	item, exists := rc.items[key]
	
	if !exists || item.IsExpired() {
		if exists && item.IsExpired() {
			// 清理過期項目
			rc.removeKey(key)
		}
		return nil, false
	}
	
	// 更新訪問信息
	item.AccessedAt = time.Now()
	item.AccessCount++
	rc.hitCount++
	
	// 更新 LRU 順序
	rc.moveToBack(key)
	
	return item.Value, true
}

// Set 設置緩存項目
func (rc *ResponseCache) Set(prompt string, model string, options map[string]interface{}, value interface{}) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	
	key := rc.generateKey(prompt, model, options)
	
	// 如果項目已存在，更新它
	if item, exists := rc.items[key]; exists {
		item.Value = value
		item.ExpiresAt = time.Now().Add(rc.ttl)
		item.AccessedAt = time.Now()
		rc.moveToBack(key)
		return
	}
	
	// 檢查是否需要清理空間
	if len(rc.items) >= rc.maxSize {
		rc.evictLRU()
	}
	
	// 添加新項目
	rc.items[key] = &CacheItem{
		Value:       value,
		ExpiresAt:   time.Now().Add(rc.ttl),
		AccessedAt:  time.Now(),
		AccessCount: 0,
	}
	rc.order = append(rc.order, key)
}

// moveToBack 將鍵移動到 LRU 順序的末尾（最近使用）
func (rc *ResponseCache) moveToBack(key string) {
	// 從當前位置移除
	for i, k := range rc.order {
		if k == key {
			rc.order = append(rc.order[:i], rc.order[i+1:]...)
			break
		}
	}
	// 添加到末尾
	rc.order = append(rc.order, key)
}

// removeKey 從緩存中移除指定鍵
func (rc *ResponseCache) removeKey(key string) {
	delete(rc.items, key)
	for i, k := range rc.order {
		if k == key {
			rc.order = append(rc.order[:i], rc.order[i+1:]...)
			break
		}
	}
}

// evictLRU 清理最少使用的項目
func (rc *ResponseCache) evictLRU() {
	if len(rc.order) == 0 {
		return
	}
	
	// 移除最舊的項目
	oldestKey := rc.order[0]
	rc.removeKey(oldestKey)
}

// Clear 清空緩存
func (rc *ResponseCache) Clear() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	
	rc.items = make(map[string]*CacheItem)
	rc.order = make([]string, 0, rc.maxSize)
	rc.hitCount = 0
	rc.requests = 0
}

// Stats 獲取緩存統計信息
func (rc *ResponseCache) Stats() CacheStats {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	
	hitRate := float64(0)
	if rc.requests > 0 {
		hitRate = float64(rc.hitCount) / float64(rc.requests) * 100
	}
	
	return CacheStats{
		Size:        len(rc.items),
		MaxSize:     rc.maxSize,
		HitCount:    rc.hitCount,
		Requests:    rc.requests,
		HitRate:     hitRate,
		TTL:         rc.ttl,
	}
}

// CleanExpired 清理過期的緩存項目
func (rc *ResponseCache) CleanExpired() int {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	
	cleaned := 0
	now := time.Now()
	
	// 收集過期的鍵
	expiredKeys := make([]string, 0)
	for key, item := range rc.items {
		if now.After(item.ExpiresAt) {
			expiredKeys = append(expiredKeys, key)
		}
	}
	
	// 移除過期項目
	for _, key := range expiredKeys {
		rc.removeKey(key)
		cleaned++
	}
	
	return cleaned
}

// CacheStats 緩存統計信息
type CacheStats struct {
	Size        int           // 當前緩存項目數
	MaxSize     int           // 最大緩存項目數
	HitCount    int64         // 命中次數
	Requests    int64         // 總請求次數
	HitRate     float64       // 命中率 (%)
	TTL         time.Duration // 緩存 TTL
}

// DefaultCacheConfig 預設緩存配置
func DefaultCacheConfig() *CacheConfig {
	return &CacheConfig{
		MaxSize:           1000,              // 最多緩存 1000 個回應
		TTL:               30 * time.Minute,  // 30 分鐘 TTL
		CleanupInterval:   5 * time.Minute,   // 每 5 分鐘清理一次過期項目
		EnableCaching:     true,              // 預設啟用緩存
	}
}

// CacheConfig 緩存配置
type CacheConfig struct {
	MaxSize           int           // 最大緩存項目數
	TTL               time.Duration // 緩存生存時間
	CleanupInterval   time.Duration // 清理過期項目的間隔
	EnableCaching     bool          // 是否啟用緩存
}

// CacheManager 緩存管理器
//
// 管理多種類型的緩存，並提供統一的介面
type CacheManager struct {
	responseCache *ResponseCache   // AI 回應緩存
	config        *CacheConfig     // 緩存配置
	stopCleanup   chan struct{}    // 停止清理 goroutine
	mu            sync.RWMutex     // 讀寫鎖
}

// NewCacheManager 創建新的緩存管理器
func NewCacheManager(config *CacheConfig) *CacheManager {
	if config == nil {
		config = DefaultCacheConfig()
	}
	
	cm := &CacheManager{
		config:      config,
		stopCleanup: make(chan struct{}),
	}
	
	if config.EnableCaching {
		cm.responseCache = NewResponseCache(config.MaxSize, config.TTL)
		
		// 啟動清理 goroutine
		go cm.cleanupLoop()
	}
	
	return cm
}

// GetResponse 獲取緩存的 AI 回應
func (cm *CacheManager) GetResponse(prompt string, model string, options map[string]interface{}) (interface{}, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	if !cm.config.EnableCaching || cm.responseCache == nil {
		return nil, false
	}
	
	return cm.responseCache.Get(prompt, model, options)
}

// SetResponse 設置 AI 回應緩存
func (cm *CacheManager) SetResponse(prompt string, model string, options map[string]interface{}, response interface{}) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	if !cm.config.EnableCaching || cm.responseCache == nil {
		return
	}
	
	cm.responseCache.Set(prompt, model, options, response)
}

// GetStats 獲取緩存統計信息
func (cm *CacheManager) GetStats() map[string]CacheStats {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	stats := make(map[string]CacheStats)
	
	if cm.responseCache != nil {
		stats["response"] = cm.responseCache.Stats()
	}
	
	return stats
}

// Clear 清空所有緩存
func (cm *CacheManager) Clear() {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	if cm.responseCache != nil {
		cm.responseCache.Clear()
	}
}

// Close 關閉緩存管理器
func (cm *CacheManager) Close() {
	close(cm.stopCleanup)
}

// cleanupLoop 清理過期項目的循環
func (cm *CacheManager) cleanupLoop() {
	ticker := time.NewTicker(cm.config.CleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			cm.cleanupExpired()
		case <-cm.stopCleanup:
			return
		}
	}
}

// cleanupExpired 清理過期的緩存項目
func (cm *CacheManager) cleanupExpired() {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	if cm.responseCache != nil {
		cleaned := cm.responseCache.CleanExpired()
		if cleaned > 0 {
			// 這裡可以記錄日誌
			_ = cleaned
		}
	}
}