package ghcopilot

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MultiTenantManager 多租戶管理器
//
// 提供多租戶支援，包括租戶隔離、資源分配和權限管理
type MultiTenantManager struct {
	tenants    map[string]*Tenant // 租戶列表
	config     *MultiTenantConfig // 多租戶配置
	mu         sync.RWMutex       // 讀寫鎖
	started    bool               // 是否已啟動
}

// MultiTenantConfig 多租戶配置
type MultiTenantConfig struct {
	DefaultTenant    string        // 預設租戶
	IsolationLevel  string        // 隔離級別 (strict/moderate/basic)
	TenantTimeout   time.Duration // 租戶操作超時
	MaxTenants      int           // 最大租戶數量
	ResourceQuota   map[string]interface{} // 資源配額
}

// Tenant 租戶信息
type Tenant struct {
	ID          string                 // 租戶ID
	Name        string                 // 租戶名稱
	Status      TenantStatus          // 租戶狀態
	CreatedAt   time.Time             // 創建時間
	UpdatedAt   time.Time             // 更新時間
	Config      map[string]interface{} // 租戶配置
	Resources   *TenantResources      // 租戶資源
	Permissions []string              // 權限列表
	Metadata    map[string]string     // 元數據
	mu          sync.RWMutex          // 租戶級鎖
}

// TenantStatus 租戶狀態
type TenantStatus int

const (
	TenantActive   TenantStatus = iota // 活躍
	TenantSuspended                    // 暫停
	TenantDisabled                     // 禁用
)

func (ts TenantStatus) String() string {
	switch ts {
	case TenantActive:
		return "active"
	case TenantSuspended:
		return "suspended"
	case TenantDisabled:
		return "disabled"
	default:
		return "unknown"
	}
}

// TenantResources 租戶資源
type TenantResources struct {
	WorkDir           string    // 工作目錄
	ConfigDir         string    // 配置目錄
	LogDir           string    // 日誌目錄
	TempDir          string    // 臨時目錄
	MaxMemoryMB      int64     // 最大記憶體限制 (MB)
	MaxCPUCores      int       // 最大CPU核心數
	MaxDiskSpaceMB   int64     // 最大磁盤空間 (MB)
	MaxConcurrentOps int       // 最大併發操作數
	LastAccessed     time.Time // 最後訪問時間
}

// DefaultMultiTenantConfig 返回預設多租戶配置
func DefaultMultiTenantConfig() *MultiTenantConfig {
	return &MultiTenantConfig{
		DefaultTenant:  "default",
		IsolationLevel: "moderate",
		TenantTimeout:  30 * time.Second,
		MaxTenants:     100,
		ResourceQuota: map[string]interface{}{
			"max_memory_mb":       1024,
			"max_cpu_cores":       2,
			"max_disk_space_mb":   5120,
			"max_concurrent_ops":  10,
		},
	}
}

// NewMultiTenantManager 創建新的多租戶管理器
func NewMultiTenantManager(config *MultiTenantConfig) *MultiTenantManager {
	if config == nil {
		config = DefaultMultiTenantConfig()
	}

	manager := &MultiTenantManager{
		tenants: make(map[string]*Tenant),
		config:  config,
		started: false,
	}

	// 創建預設租戶
	defaultTenant := &Tenant{
		ID:        config.DefaultTenant,
		Name:      "Default Tenant",
		Status:    TenantActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Config:    make(map[string]interface{}),
		Resources: &TenantResources{
			WorkDir:          "./tenant-" + config.DefaultTenant,
			ConfigDir:        "./tenant-" + config.DefaultTenant + "/config",
			LogDir:          "./tenant-" + config.DefaultTenant + "/logs",
			TempDir:         "./tenant-" + config.DefaultTenant + "/temp",
			MaxMemoryMB:     config.ResourceQuota["max_memory_mb"].(int64),
			MaxCPUCores:     config.ResourceQuota["max_cpu_cores"].(int),
			MaxDiskSpaceMB:  config.ResourceQuota["max_disk_space_mb"].(int64),
			MaxConcurrentOps: config.ResourceQuota["max_concurrent_ops"].(int),
			LastAccessed:    time.Now(),
		},
		Permissions: []string{"execute", "read", "write"},
		Metadata:    make(map[string]string),
	}

	manager.tenants[config.DefaultTenant] = defaultTenant
	return manager
}

// Start 啟動多租戶管理器
func (mtm *MultiTenantManager) Start(ctx context.Context) error {
	mtm.mu.Lock()
	defer mtm.mu.Unlock()

	if mtm.started {
		return fmt.Errorf("multi-tenant manager already started")
	}

	// 初始化租戶資源目錄
	for _, tenant := range mtm.tenants {
		if err := mtm.initializeTenantResources(tenant); err != nil {
			return fmt.Errorf("failed to initialize tenant resources for %s: %w", tenant.ID, err)
		}
	}

	mtm.started = true
	return nil
}

// Stop 停止多租戶管理器
func (mtm *MultiTenantManager) Stop(ctx context.Context) error {
	mtm.mu.Lock()
	defer mtm.mu.Unlock()

	if !mtm.started {
		return nil
	}

	mtm.started = false
	return nil
}

// CreateTenant 創建新租戶
func (mtm *MultiTenantManager) CreateTenant(tenantID, tenantName string, config map[string]interface{}) (*Tenant, error) {
	mtm.mu.Lock()
	defer mtm.mu.Unlock()

	if !mtm.started {
		return nil, fmt.Errorf("multi-tenant manager not started")
	}

	// 檢查租戶是否已存在
	if _, exists := mtm.tenants[tenantID]; exists {
		return nil, fmt.Errorf("tenant %s already exists", tenantID)
	}

	// 檢查租戶數量限制
	if len(mtm.tenants) >= mtm.config.MaxTenants {
		return nil, fmt.Errorf("maximum number of tenants (%d) reached", mtm.config.MaxTenants)
	}

	// 創建租戶資源配置
	resources := &TenantResources{
		WorkDir:           "./tenant-" + tenantID,
		ConfigDir:         "./tenant-" + tenantID + "/config",
		LogDir:           "./tenant-" + tenantID + "/logs",
		TempDir:          "./tenant-" + tenantID + "/temp",
		MaxMemoryMB:      mtm.config.ResourceQuota["max_memory_mb"].(int64),
		MaxCPUCores:      mtm.config.ResourceQuota["max_cpu_cores"].(int),
		MaxDiskSpaceMB:   mtm.config.ResourceQuota["max_disk_space_mb"].(int64),
		MaxConcurrentOps: mtm.config.ResourceQuota["max_concurrent_ops"].(int),
		LastAccessed:     time.Now(),
	}

	// 創建租戶
	tenant := &Tenant{
		ID:          tenantID,
		Name:        tenantName,
		Status:      TenantActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Config:      config,
		Resources:   resources,
		Permissions: []string{"execute", "read"},
		Metadata:    make(map[string]string),
	}

	// 初始化租戶資源
	if err := mtm.initializeTenantResources(tenant); err != nil {
		return nil, fmt.Errorf("failed to initialize tenant resources: %w", err)
	}

	mtm.tenants[tenantID] = tenant
	return tenant, nil
}

// GetTenant 獲取租戶信息
func (mtm *MultiTenantManager) GetTenant(tenantID string) (*Tenant, error) {
	mtm.mu.RLock()
	defer mtm.mu.RUnlock()

	tenant, exists := mtm.tenants[tenantID]
	if !exists {
		return nil, fmt.Errorf("tenant %s not found", tenantID)
	}

	// 更新最後訪問時間
	tenant.mu.Lock()
	tenant.Resources.LastAccessed = time.Now()
	tenant.mu.Unlock()

	return tenant, nil
}

// ListTenants 列出所有租戶
func (mtm *MultiTenantManager) ListTenants() []*Tenant {
	mtm.mu.RLock()
	defer mtm.mu.RUnlock()

	tenants := make([]*Tenant, 0, len(mtm.tenants))
	for _, tenant := range mtm.tenants {
		tenants = append(tenants, tenant)
	}

	return tenants
}

// UpdateTenant 更新租戶信息
func (mtm *MultiTenantManager) UpdateTenant(tenantID string, updates map[string]interface{}) error {
	mtm.mu.Lock()
	defer mtm.mu.Unlock()

	tenant, exists := mtm.tenants[tenantID]
	if !exists {
		return fmt.Errorf("tenant %s not found", tenantID)
	}

	tenant.mu.Lock()
	defer tenant.mu.Unlock()

	// 更新租戶配置
	for key, value := range updates {
		tenant.Config[key] = value
	}

	tenant.UpdatedAt = time.Now()
	return nil
}

// DeleteTenant 刪除租戶
func (mtm *MultiTenantManager) DeleteTenant(tenantID string) error {
	mtm.mu.Lock()
	defer mtm.mu.Unlock()

	// 不能刪除預設租戶
	if tenantID == mtm.config.DefaultTenant {
		return fmt.Errorf("cannot delete default tenant")
	}

	tenant, exists := mtm.tenants[tenantID]
	if !exists {
		return fmt.Errorf("tenant %s not found", tenantID)
	}

	// 清理租戶資源
	if err := mtm.cleanupTenantResources(tenant); err != nil {
		return fmt.Errorf("failed to cleanup tenant resources: %w", err)
	}

	delete(mtm.tenants, tenantID)
	return nil
}

// ValidateTenantAccess 驗證租戶訪問權限
func (mtm *MultiTenantManager) ValidateTenantAccess(tenantID, operation string) error {
	mtm.mu.RLock()
	defer mtm.mu.RUnlock()

	tenant, exists := mtm.tenants[tenantID]
	if !exists {
		return fmt.Errorf("tenant %s not found", tenantID)
	}

	tenant.mu.RLock()
	defer tenant.mu.RUnlock()

	// 檢查租戶狀態
	if tenant.Status != TenantActive {
		return fmt.Errorf("tenant %s is %s", tenantID, tenant.Status)
	}

	// 檢查操作權限
	hasPermission := false
	for _, permission := range tenant.Permissions {
		if permission == operation || permission == "admin" {
			hasPermission = true
			break
		}
	}

	if !hasPermission {
		return fmt.Errorf("tenant %s does not have permission for operation %s", tenantID, operation)
	}

	return nil
}

// SetTenantStatus 設置租戶狀態
func (mtm *MultiTenantManager) SetTenantStatus(tenantID string, status TenantStatus) error {
	mtm.mu.Lock()
	defer mtm.mu.Unlock()

	tenant, exists := mtm.tenants[tenantID]
	if !exists {
		return fmt.Errorf("tenant %s not found", tenantID)
	}

	tenant.mu.Lock()
	defer tenant.mu.Unlock()

	tenant.Status = status
	tenant.UpdatedAt = time.Now()
	return nil
}

// GetTenantResourceUsage 獲取租戶資源使用情況
func (mtm *MultiTenantManager) GetTenantResourceUsage(tenantID string) (map[string]interface{}, error) {
	tenant, err := mtm.GetTenant(tenantID)
	if err != nil {
		return nil, err
	}

	tenant.mu.RLock()
	defer tenant.mu.RUnlock()

	usage := map[string]interface{}{
		"tenant_id":         tenant.ID,
		"work_dir":         tenant.Resources.WorkDir,
		"max_memory_mb":    tenant.Resources.MaxMemoryMB,
		"max_cpu_cores":    tenant.Resources.MaxCPUCores,
		"max_disk_space_mb": tenant.Resources.MaxDiskSpaceMB,
		"max_concurrent_ops": tenant.Resources.MaxConcurrentOps,
		"last_accessed":    tenant.Resources.LastAccessed,
		"status":           tenant.Status.String(),
	}

	return usage, nil
}

// GetStatus 獲取多租戶管理器狀態
func (mtm *MultiTenantManager) GetStatus() map[string]interface{} {
	mtm.mu.RLock()
	defer mtm.mu.RUnlock()

	status := map[string]interface{}{
		"started":         mtm.started,
		"total_tenants":   len(mtm.tenants),
		"max_tenants":     mtm.config.MaxTenants,
		"default_tenant":  mtm.config.DefaultTenant,
		"isolation_level": mtm.config.IsolationLevel,
	}

	// 統計租戶狀態
	statusCounts := map[string]int{
		"active":    0,
		"suspended": 0,
		"disabled":  0,
	}

	for _, tenant := range mtm.tenants {
		tenant.mu.RLock()
		statusCounts[tenant.Status.String()]++
		tenant.mu.RUnlock()
	}

	status["tenant_status_counts"] = statusCounts
	return status
}

// 私有輔助方法

// initializeTenantResources 初始化租戶資源目錄
func (mtm *MultiTenantManager) initializeTenantResources(tenant *Tenant) error {
	// 這裡應該創建租戶的工作目錄、配置目錄等
	// 為了簡化實作，這裡僅做基本檢查
	if tenant.Resources == nil {
		return fmt.Errorf("tenant resources not configured")
	}

	// 在實際實作中，這裡會創建必要的目錄結構
	// os.MkdirAll(tenant.Resources.WorkDir, 0755)
	// os.MkdirAll(tenant.Resources.ConfigDir, 0755)
	// 等等...

	return nil
}

// cleanupTenantResources 清理租戶資源
func (mtm *MultiTenantManager) cleanupTenantResources(tenant *Tenant) error {
	// 這裡應該清理租戶的資源目錄
	// 為了安全考慮，這裡僅做標記
	
	// 在實際實作中，這裡會清理目錄
	// os.RemoveAll(tenant.Resources.WorkDir)
	// 等等...

	return nil
}

// GetTenantWorkDir 獲取租戶工作目錄
func (mtm *MultiTenantManager) GetTenantWorkDir(tenantID string) (string, error) {
	tenant, err := mtm.GetTenant(tenantID)
	if err != nil {
		return "", err
	}

	tenant.mu.RLock()
	defer tenant.mu.RUnlock()

	return tenant.Resources.WorkDir, nil
}

// IsTenantActive 檢查租戶是否活躍
func (mtm *MultiTenantManager) IsTenantActive(tenantID string) bool {
	mtm.mu.RLock()
	defer mtm.mu.RUnlock()

	tenant, exists := mtm.tenants[tenantID]
	if !exists {
		return false
	}

	tenant.mu.RLock()
	defer tenant.mu.RUnlock()

	return tenant.Status == TenantActive
}