package ghcopilot

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PersistenceData 用於序列化的數據結構（導出欄位以支援 Gob）
type PersistenceData struct {
	Summary     map[string]interface{} `json:"summary"`
	LoopHistory []*ExecutionContext    `json:"history"`
}

// PersistenceManager 管理上下文的序列化和持久化
type PersistenceManager struct {
	storageDir string // 儲存目錄
	useGob     bool   // 是否使用 Gob 編碼（比 JSON 更快且緊湊）
	maxBackups int    // 最多保留的備份數量
}

// NewPersistenceManager 建立新的持久化管理器
func NewPersistenceManager(storageDir string, useGob bool) (*PersistenceManager, error) {
	// 建立儲存目錄 (0750: 僅擁有者和群組可讀取)
	if err := os.MkdirAll(storageDir, 0750); err != nil {
		return nil, fmt.Errorf("無法建立儲存目錄: %w", err)
	}

	return &PersistenceManager{
		storageDir: storageDir,
		useGob:     useGob,
		maxBackups: 10,
	}, nil
}

// validatePath 驗證檔案路徑在允許的儲存目錄範圍內，防止路徑穿越攻擊
func (pm *PersistenceManager) validatePath(filename string) error {
	// 取得絕對路徑
	absPath, err := filepath.Abs(filename)
	if err != nil {
		return fmt.Errorf("無法解析路徑: %w", err)
	}
	
	// 取得儲存目錄的絕對路徑
	absStorageDir, err := filepath.Abs(pm.storageDir)
	if err != nil {
		return fmt.Errorf("無法解析儲存目錄: %w", err)
	}
	
	// 驗證路徑在儲存目錄範圍內
	if !strings.HasPrefix(absPath, absStorageDir) {
		return fmt.Errorf("路徑超出允許範圍: %s (允許: %s)", absPath, absStorageDir)
	}
	
	return nil
}

// SaveContextManager 儲存整個上下文管理器到檔案
func (pm *PersistenceManager) SaveContextManager(cm *ContextManager) error {
	if cm == nil {
		return fmt.Errorf("上下文管理器不能為 nil")
	}

	filename := filepath.Join(pm.storageDir, "context_manager_"+time.Now().Format("20060102_150405")+pm.getExtension())

	// #nosec G304 -- filename 由 filepath.Join 從 storageDir 構建，範圍受限
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("無法建立檔案: %w", err)
	}
	defer file.Close()

	if pm.useGob {
		return pm.saveAsGobData(cm, file)
	}
	return pm.saveAsJSON(cm, file)
}

// LoadContextManager 從檔案載入上下文管理器
func (pm *PersistenceManager) LoadContextManager(filename string) (*ContextManager, error) {
	if err := pm.validatePath(filename); err != nil {
		return nil, err
	}
	
	file, err := os.Open(filename) // #nosec G304 -- 路徑已透過 validatePath 驗證在 storageDir 範圍內
	if err != nil {
		return nil, fmt.Errorf("無法打開檔案: %w", err)
	}
	defer file.Close()

	// 根據副檔名自動決定解碼方式
	if filepath.Ext(filename) == ".gob" {
		return pm.loadFromGobData(file)
	}
	return pm.loadFromJSON(file)
}

// SaveExecutionContext 儲存單個執行上下文
func (pm *PersistenceManager) SaveExecutionContext(ctx *ExecutionContext) error {
	if ctx == nil {
		return fmt.Errorf("執行上下文不能為 nil")
	}

	filename := filepath.Join(pm.storageDir, "loop_"+ctx.LoopID+pm.getExtension())

	// #nosec G304 -- filename 由 filepath.Join 從 storageDir 構建，範圍受限
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("無法建立檔案: %w", err)
	}
	defer file.Close()

	if pm.useGob {
		encoder := gob.NewEncoder(file)
		return encoder.Encode(ctx)
	}

	bytes, err := json.MarshalIndent(ctx, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON 編碼失敗: %w", err)
	}

	_, err = file.Write(bytes)
	return err
}

// LoadExecutionContext 載入單個執行上下文
func (pm *PersistenceManager) LoadExecutionContext(loopID string) (*ExecutionContext, error) {
	// 嘗試載入 Gob 格式
	filename := filepath.Join(pm.storageDir, "loop_"+loopID+".gob")
	if _, err := os.Stat(filename); err == nil {
		return pm.loadContextFromGob(filename)
	}

	// 嘗試載入 JSON 格式
	filename = filepath.Join(pm.storageDir, "loop_"+loopID+".json")
	if _, err := os.Stat(filename); err == nil {
		return pm.loadContextFromJSON(filename)
	}

	return nil, fmt.Errorf("找不到迴圈上下文: %s", loopID)
}

// ListSavedContexts 列出所有已儲存的上下文檔案
func (pm *PersistenceManager) ListSavedContexts() ([]string, error) {
	entries, err := os.ReadDir(pm.storageDir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && (filepath.Ext(entry.Name()) == ".json" || filepath.Ext(entry.Name()) == ".gob") {
			files = append(files, entry.Name())
		}
	}

	return files, nil
}

// ExportAsJSON 以 JSON 格式匯出上下文管理器
func (pm *PersistenceManager) ExportAsJSON(cm *ContextManager, outputPath string) error {
	jsonStr, err := cm.ToJSON()
	if err != nil {
		return err
	}

	return os.WriteFile(outputPath, []byte(jsonStr), 0600)
}

// ClearOldBackups 清理舊的備份檔案，只保留最新的 maxBackups 個
func (pm *PersistenceManager) ClearOldBackups(prefix string) error {
	entries, err := os.ReadDir(pm.storageDir)
	if err != nil {
		return err
	}

	var files []os.DirEntry
	for _, entry := range entries {
		if !entry.IsDir() {
			name := entry.Name()
			// 檢查是否符合前綴
			if len(name) > len(prefix) && name[:len(prefix)] == prefix {
				files = append(files, entry)
			}
		}
	}

	// 如果檔案數超過限制，刪除最舊的
	if len(files) > pm.maxBackups {
		// 按修改時間排序（最舊的在前）
		// 簡單實現：直接刪除超出限制的檔案
		for i := 0; i < len(files)-pm.maxBackups; i++ {
			filePath := filepath.Join(pm.storageDir, files[i].Name())
			if err := os.Remove(filePath); err != nil {
				return fmt.Errorf("無法刪除檔案 %s: %w", filePath, err)
			}
		}
	}

	return nil
}

// 私有輔助函式

func (pm *PersistenceManager) getExtension() string {
	if pm.useGob {
		return ".gob"
	}
	return ".json"
}

func (pm *PersistenceManager) saveAsJSON(cm *ContextManager, file *os.File) error {
	jsonStr, err := cm.ToJSON()
	if err != nil {
		return err
	}

	_, err = file.WriteString(jsonStr)
	return err
}

func (pm *PersistenceManager) saveAsGobData(cm *ContextManager, file *os.File) error {
	// 建立可導出的數據結構
	data := &PersistenceData{
		Summary:     cm.GetSummary(),
		LoopHistory: cm.GetLoopHistory(),
	}

	encoder := gob.NewEncoder(file)
	return encoder.Encode(data)
}

func (pm *PersistenceManager) loadFromJSON(file *os.File) (*ContextManager, error) {
	data := make(map[string]interface{})
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		return nil, fmt.Errorf("JSON 解碼失敗: %w", err)
	}

	// 重新構建 ContextManager
	cm := NewContextManager()

	// 驗證數據完整性
	if data == nil || len(data) == 0 {
		return nil, fmt.Errorf("加載的數據為空")
	}

	// 注意: 這是簡化版本，實際應用可能需要更完善的反序列化邏輯
	if history, ok := data["history"].([]interface{}); ok {
		loadedCount := 0
		for _, item := range history {
			if itemMap, ok := item.(map[string]interface{}); ok {
				bytes, _ := json.Marshal(itemMap)
				var ctx ExecutionContext
				if err := json.Unmarshal(bytes, &ctx); err == nil {
					cm.loopHistory = append(cm.loopHistory, &ctx)
					loadedCount++
				}
			}
		}
		// 如果沒有成功加載任何歷史記錄，記錄警告
		if len(history) > 0 && loadedCount == 0 {
			return cm, fmt.Errorf("無法加載任何歷史記錄 (共 %d 條)", len(history))
		}
	}

	return cm, nil
}

func (pm *PersistenceManager) loadFromGobData(file *os.File) (*ContextManager, error) {
	var data PersistenceData
	decoder := gob.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		return nil, fmt.Errorf("Gob 解碼失敗: %w", err)
	}

	// 重新構建 ContextManager
	cm := NewContextManager()
	cm.loopHistory = data.LoopHistory

	return cm, nil
}

func (pm *PersistenceManager) loadContextFromJSON(filename string) (*ExecutionContext, error) {
	if err := pm.validatePath(filename); err != nil {
		return nil, err
	}
	
	file, err := os.Open(filename) // #nosec G304 -- 路徑已透過 validatePath 驗證在 storageDir 範圍內
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var ctx ExecutionContext
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&ctx); err != nil {
		return nil, fmt.Errorf("JSON 解碼失敗: %w", err)
	}

	return &ctx, nil
}

func (pm *PersistenceManager) loadContextFromGob(filename string) (*ExecutionContext, error) {
	if err := pm.validatePath(filename); err != nil {
		return nil, err
	}
	
	file, err := os.Open(filename) // #nosec G304 -- 路徑已透過 validatePath 驗證在 storageDir 範圍內
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var ctx ExecutionContext
	decoder := gob.NewDecoder(file)
	if err := decoder.Decode(&ctx); err != nil {
		return nil, fmt.Errorf("Gob 解碼失敗: %w", err)
	}

	return &ctx, nil
}

// SetMaxBackups 設定最多保留的備份數量
func (pm *PersistenceManager) SetMaxBackups(count int) {
	pm.maxBackups = count
}

// GetStorageDir 取得儲存目錄
func (pm *PersistenceManager) GetStorageDir() string {
	return pm.storageDir
}
