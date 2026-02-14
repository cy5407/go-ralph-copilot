package ghcopilot

import (
	"bytes"
	"context"
	"errors"
	"runtime"
	"sync"
	"time"
)

// MemoryPool 記憶體池，用於減少頻繁的記憶體分配
type MemoryPool struct {
	smallBuffers  sync.Pool // 小緩衝區池 (< 1KB)
	mediumBuffers sync.Pool // 中等緩衝區池 (1KB - 10KB)
	largeBuffers  sync.Pool // 大緩衝區池 (> 10KB)
}

// NewMemoryPool 創建新的記憶體池
func NewMemoryPool() *MemoryPool {
	mp := &MemoryPool{}
	
	// 初始化小緩衝區池
	mp.smallBuffers.New = func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 1024)) // 1KB
	}
	
	// 初始化中等緩衝區池
	mp.mediumBuffers.New = func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 10*1024)) // 10KB
	}
	
	// 初始化大緩衝區池
	mp.largeBuffers.New = func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 100*1024)) // 100KB
	}
	
	return mp
}

// GetBuffer 根據預期大小獲取合適的緩衝區
func (mp *MemoryPool) GetBuffer(expectedSize int) *bytes.Buffer {
	switch {
	case expectedSize <= 1024:
		return mp.smallBuffers.Get().(*bytes.Buffer)
	case expectedSize <= 10*1024:
		return mp.mediumBuffers.Get().(*bytes.Buffer)
	default:
		return mp.largeBuffers.Get().(*bytes.Buffer)
	}
}

// PutBuffer 歸還緩衝區到池中
func (mp *MemoryPool) PutBuffer(buf *bytes.Buffer) {
	if buf == nil {
		return
	}
	
	// 重置緩衝區但保留容量
	buf.Reset()
	
	// 根據容量歸還到對應的池
	cap := buf.Cap()
	switch {
	case cap <= 1024:
		mp.smallBuffers.Put(buf)
	case cap <= 10*1024:
		mp.mediumBuffers.Put(buf)
	default:
		// 對於過大的緩衝區，不歸還到池中以避免記憶體浪費
		if cap <= 1024*1024 { // 1MB 以下才歸還
			mp.largeBuffers.Put(buf)
		}
	}
}

// ConcurrentExecutionManager 併發執行管理器
//
// 支援並發執行多個 Ralph Loop 迴圈，
// 提供工作者池、任務排程和資源管理
type ConcurrentExecutionManager struct {
	workerPool    chan *ConcurrentWorker         // 工作者池
	taskQueue     chan *ExecutionTask            // 任務佇列
	results       chan *ConcurrentExecutionResult // 結果通道
	maxWorkers    int                     // 最大工作者數量
	maxQueueSize  int                     // 最大佇列大小
	workers       []*ConcurrentWorker     // 工作者切片
	stopChan      chan struct{}           // 停止信號
	wg            sync.WaitGroup          // 等待群組
	mu            sync.RWMutex            // 讀寫鎖
	started       bool                    // 是否已啟動
	memoryPool    *MemoryPool             // 記憶體池
}

// ExecutionTask 執行任務
type ExecutionTask struct {
	ID        string                            // 任務 ID
	Prompt    string                            // AI 提示詞
	MaxLoops  int                               // 最大迴圈數
	Context   context.Context                   // 執行上下文
	Config    *ClientConfig                     // 客戶端配置
	Callback  func(*ConcurrentExecutionResult) // 結果回調
}

// ConcurrentExecutionResult 併發執行結果
type ConcurrentExecutionResult struct {
	TaskID    string         // 任務 ID
	Results   []*LoopResult  // 迴圈結果列表
	Error     error          // 錯誤信息
	Duration  time.Duration  // 執行時間
	WorkerID  int            // 工作者 ID
}

// ConcurrentWorker 併發工作者
type ConcurrentWorker struct {
	ID             int                                // 工作者 ID
	client         *RalphLoopClient                   // Ralph Loop 客戶端
	taskChan       chan *ExecutionTask                // 任務通道
	resultChan     chan *ConcurrentExecutionResult    // 結果通道
	stopChan       chan struct{}            // 停止信號
	memoryPool     *MemoryPool              // 記憶體池
}

// NewConcurrentExecutionManager 創建併發執行管理器
func NewConcurrentExecutionManager(maxWorkers, maxQueueSize int) *ConcurrentExecutionManager {
	if maxWorkers <= 0 {
		maxWorkers = runtime.NumCPU()
	}
	if maxQueueSize <= 0 {
		maxQueueSize = maxWorkers * 10
	}
	
	return &ConcurrentExecutionManager{
		workerPool:   make(chan *ConcurrentWorker, maxWorkers),
		taskQueue:    make(chan *ExecutionTask, maxQueueSize),
		results:      make(chan *ConcurrentExecutionResult, maxQueueSize),
		maxWorkers:   maxWorkers,
		maxQueueSize: maxQueueSize,
		workers:      make([]*ConcurrentWorker, 0, maxWorkers),
		stopChan:     make(chan struct{}),
		memoryPool:   NewMemoryPool(),
	}
}

// Start 啟動併發執行管理器
func (cem *ConcurrentExecutionManager) Start() error {
	cem.mu.Lock()
	defer cem.mu.Unlock()
	
	if cem.started {
		return errors.New("併發執行管理器已經啟動")
	}
	
	// 創建並啟動工作者
	for i := 0; i < cem.maxWorkers; i++ {
		worker := &ConcurrentWorker{
			ID:         i,
			taskChan:   make(chan *ExecutionTask, 1),
			resultChan: cem.results,
			stopChan:   make(chan struct{}),
			memoryPool: cem.memoryPool,
		}
		
		cem.workers = append(cem.workers, worker)
		cem.workerPool <- worker
		
		cem.wg.Add(1)
		go cem.workerLoop(worker)
	}
	
	// 啟動任務分派器
	cem.wg.Add(1)
	go cem.dispatchLoop()
	
	cem.started = true
	return nil
}

// Stop 停止併發執行管理器
func (cem *ConcurrentExecutionManager) Stop() {
	cem.mu.Lock()
	defer cem.mu.Unlock()
	
	if !cem.started {
		return
	}
	
	// 發送停止信號
	close(cem.stopChan)
	
	// 停止所有工作者
	for _, worker := range cem.workers {
		close(worker.stopChan)
	}
	
	// 等待所有 goroutine 完成
	cem.wg.Wait()
	
	// 關閉通道
	close(cem.taskQueue)
	close(cem.results)
	
	cem.started = false
}

// SubmitTask 提交執行任務
func (cem *ConcurrentExecutionManager) SubmitTask(task *ExecutionTask) error {
	cem.mu.RLock()
	defer cem.mu.RUnlock()
	
	if !cem.started {
		return errors.New("併發執行管理器未啟動")
	}
	
	select {
	case cem.taskQueue <- task:
		return nil
	default:
		return errors.New("任務佇列已滿")
	}
}

// GetResult 獲取執行結果
func (cem *ConcurrentExecutionManager) GetResult() *ConcurrentExecutionResult {
	select {
	case result := <-cem.results:
		return result
	default:
		return nil
	}
}

// GetStats 獲取併發執行統計信息
func (cem *ConcurrentExecutionManager) GetStats() ConcurrentStats {
	cem.mu.RLock()
	defer cem.mu.RUnlock()
	
	return ConcurrentStats{
		MaxWorkers:      cem.maxWorkers,
		ActiveWorkers:   len(cem.workers),
		QueuedTasks:     len(cem.taskQueue),
		MaxQueueSize:    cem.maxQueueSize,
		AvailableWorkers: len(cem.workerPool),
		IsRunning:       cem.started,
	}
}

// dispatchLoop 任務分派循環
func (cem *ConcurrentExecutionManager) dispatchLoop() {
	defer cem.wg.Done()
	
	for {
		select {
		case task := <-cem.taskQueue:
			// 獲取可用工作者
			select {
			case worker := <-cem.workerPool:
				// 分派任務給工作者
				select {
				case worker.taskChan <- task:
					// 任務分派成功
				default:
					// 工作者忙碌，歸還到池中並重新排隊任務
					cem.workerPool <- worker
					cem.taskQueue <- task
				}
			case <-cem.stopChan:
				return
			}
		case <-cem.stopChan:
			return
		}
	}
}

// workerLoop 工作者循環
func (cem *ConcurrentExecutionManager) workerLoop(worker *ConcurrentWorker) {
	defer cem.wg.Done()
	
	for {
		select {
		case task := <-worker.taskChan:
			// 執行任務
			result := cem.executeTask(worker, task)
			
			// 發送結果
			select {
			case worker.resultChan <- result:
			default:
				// 結果通道已滿，丟棄結果（或記錄錯誤）
			}
			
			// 歸還工作者到池中
			select {
			case cem.workerPool <- worker:
			default:
				// 池已滿，這不應該發生
			}
			
		case <-worker.stopChan:
			return
		case <-cem.stopChan:
			return
		}
	}
}

// executeTask 執行具體任務
func (cem *ConcurrentExecutionManager) executeTask(worker *ConcurrentWorker, task *ExecutionTask) *ConcurrentExecutionResult {
	startTime := time.Now()
	
	result := &ConcurrentExecutionResult{
		TaskID:   task.ID,
		WorkerID: worker.ID,
		Duration: 0,
	}
	
	// 為工作者創建專用的客戶端
	if worker.client == nil {
		worker.client = NewRalphLoopClientWithConfig(task.Config)
	}
	
	// 執行任務
	loopResults, err := worker.client.ExecuteUntilCompletion(task.Context, task.Prompt, task.MaxLoops)
	
	result.Results = loopResults
	result.Error = err
	result.Duration = time.Since(startTime)
	
	// 調用回調
	if task.Callback != nil {
		task.Callback(result)
	}
	
	return result
}

// ConcurrentStats 併發執行統計信息
type ConcurrentStats struct {
	MaxWorkers       int  // 最大工作者數量
	ActiveWorkers    int  // 活躍工作者數量
	QueuedTasks      int  // 排隊任務數量
	MaxQueueSize     int  // 最大佇列大小
	AvailableWorkers int  // 可用工作者數量
	IsRunning        bool // 是否正在運行
}

// MemoryStats 記憶體使用統計
type MemoryStats struct {
	AllocatedBytes   uint64  // 已分配字節數
	TotalAllocBytes  uint64  // 總分配字節數
	SystemBytes      uint64  // 系統字節數
	GCCycles         uint32  // GC 循環次數
	GoroutineCount   int     // Goroutine 數量
	HeapInUse        uint64  // 堆使用中字節數
	HeapIdle         uint64  // 堆閒置字節數
}

// GetMemoryStats 獲取當前記憶體使用統計
func GetMemoryStats() MemoryStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	return MemoryStats{
		AllocatedBytes:  m.Alloc,
		TotalAllocBytes: m.TotalAlloc,
		SystemBytes:     m.Sys,
		GCCycles:        m.NumGC,
		GoroutineCount:  runtime.NumGoroutine(),
		HeapInUse:       m.HeapInuse,
		HeapIdle:        m.HeapIdle,
	}
}

// ForceGC 強制執行垃圾回收
func ForceGC() {
	runtime.GC()
}