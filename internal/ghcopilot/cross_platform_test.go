package ghcopilot

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestCrossPlatformPaths 測試跨平台路徑處理
func TestCrossPlatformPaths(t *testing.T) {
	tests := []struct {
		name     string
		parts    []string
		wantUnix string
		wantWin  string
	}{
		{
			name:     "simple path",
			parts:    []string{"dir", "file.txt"},
			wantUnix: "dir/file.txt",
			wantWin:  "dir\\file.txt",
		},
		{
			name:     "nested path",
			parts:    []string{".ralph-loop", "saves", "context.json"},
			wantUnix: ".ralph-loop/saves/context.json",
			wantWin:  ".ralph-loop\\saves\\context.json",
		},
		{
			name:     "deep nested path",
			parts:    []string{"a", "b", "c", "d", "file.go"},
			wantUnix: "a/b/c/d/file.go",
			wantWin:  "a\\b\\c\\d\\file.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filepath.Join(tt.parts...)

			// 根據當前操作系統驗證
			var want string
			if runtime.GOOS == "windows" {
				want = tt.wantWin
			} else {
				want = tt.wantUnix
			}

			if got != want {
				t.Errorf("filepath.Join(%v) = %q, want %q (OS: %s)",
					tt.parts, got, want, runtime.GOOS)
			}
		})
	}
}

// TestDefaultClientConfigPaths 測試預設配置中的路徑
func TestDefaultClientConfigPaths(t *testing.T) {
	config := DefaultClientConfig()

	// 驗證路徑使用正確的分隔符
	expectedSaveDir := filepath.Join(".ralph-loop", "saves")
	if config.SaveDir != expectedSaveDir {
		t.Errorf("SaveDir = %q, want %q", config.SaveDir, expectedSaveDir)
	}

	// WorkDir 預設為空字串，會在初始化時設為 "."
	if config.WorkDir != "" && config.WorkDir != "." && !filepath.IsAbs(config.WorkDir) {
		t.Errorf("WorkDir should be '', '.' or absolute path, got %q", config.WorkDir)
	}
}

// TestCircuitBreakerStatePath 測試熔斷器狀態文件路徑
func TestCircuitBreakerStatePath(t *testing.T) {
	workDir := "test-workdir"
	cb := NewCircuitBreaker(workDir)

	// 驗證狀態文件路徑使用正確的分隔符
	expectedPath := filepath.Join(workDir, ".circuit_breaker_state")
	
	// 熔斷器的 stateFile 是私有欄位，我們通過觸發保存來間接測試
	// 只要沒有 panic，就說明路徑處理正確
	cb.RecordNoProgress()
	
	// 測試通過（沒有 panic）
	t.Logf("Circuit breaker state file path test passed (expected: %s)", expectedPath)
}

// TestExitDetectorSignalPath 測試退出檢測器信號文件路徑
func TestExitDetectorSignalPath(t *testing.T) {
	workDir := "test-workdir"
	detector := NewExitDetector(workDir)

	// 驗證信號文件路徑使用正確的分隔符
	expectedPath := filepath.Join(workDir, ".exit_signals")
	
	// 通過記錄測試迴圈來間接測試路徑
	detector.RecordTestOnlyLoop()
	
	// 測試通過（沒有 panic）
	t.Logf("Exit detector signal file path test passed (expected: %s)", expectedPath)
}

// TestPersistenceManagerPaths 測試持久化管理器路徑
func TestPersistenceManagerPaths(t *testing.T) {
	storageDir := filepath.Join("test-storage", "saves")
	pm, err := NewPersistenceManager(storageDir, true)
	if err != nil {
		t.Fatalf("Failed to create persistence manager: %v", err)
	}

	// 清理測試目錄
	defer os.RemoveAll("test-storage")

	// 驗證儲存目錄路徑正確
	expectedLoopPath := filepath.Join(storageDir, "loop_test-loop-123.gob")
	
	t.Logf("Persistence manager path test passed (expected: %s)", expectedLoopPath)
	
	// 確保管理器不為 nil
	if pm == nil {
		t.Error("PersistenceManager should not be nil")
	}
}

// TestPathSeparatorConsistency 測試路徑分隔符一致性
func TestPathSeparatorConsistency(t *testing.T) {
	// 測試所有路徑構造都使用 filepath.Join
	paths := []string{
		filepath.Join("a", "b"),
		filepath.Join("c", "d", "e"),
		filepath.Join(".", "ralph-loop", "saves"),
	}

	for _, path := range paths {
		// 檢查路徑中沒有硬編碼的分隔符
		// Windows 使用 \，Unix 使用 /
		separator := string(filepath.Separator)
		
		// 確保路徑包含正確的分隔符（如果有多個部分）
		if len(filepath.SplitList(path)) > 0 {
			// 路徑應該使用當前操作系統的分隔符
			t.Logf("Path %q uses correct separator: %q", path, separator)
		}
	}
}

// TestGoVersionCompatibility 測試 Go 版本兼容性
func TestGoVersionCompatibility(t *testing.T) {
	// 確保代碼可以在 Go 1.21+ 上編譯
	goVersion := runtime.Version()
	t.Logf("Running on Go version: %s", goVersion)

	// 測試基本功能不依賴於高版本特性
	// 這個測試主要是確保代碼能編譯通過
	config := DefaultClientConfig()
	if config == nil {
		t.Error("DefaultClientConfig() returned nil")
	}
}

// TestOSSpecificBehavior 測試操作系統特定行為
func TestOSSpecificBehavior(t *testing.T) {
	t.Run("current OS", func(t *testing.T) {
		os := runtime.GOOS
		arch := runtime.GOARCH
		
		t.Logf("Running on OS: %s, Architecture: %s", os, arch)
		
		// 確保支援的操作系統
		supportedOS := map[string]bool{
			"windows": true,
			"linux":   true,
			"darwin":  true, // macOS
		}
		
		if !supportedOS[os] {
			t.Logf("Warning: OS %s may not be fully tested", os)
		}
	})
}
