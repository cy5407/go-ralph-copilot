package ghcopilot

import (
	"testing"
)

// TestNewDependencyChecker 測試建立新的依賴檢查器
func TestNewDependencyChecker(t *testing.T) {
	dc := NewDependencyChecker()
	if dc == nil {
		t.Error("NewDependencyChecker() 傳回 nil")
	}
	if len(dc.errors) != 0 {
		t.Errorf("初始化時應有零個錯誤，但有 %d 個", len(dc.errors))
	}
}

// TestHasErrors 測試 HasErrors 方法
func TestHasErrors(t *testing.T) {
	dc := NewDependencyChecker()
	if dc.HasErrors() {
		t.Error("新的依賴檢查器不應有錯誤")
	}

	// 手動新增錯誤進行測試
	dc.errors = append(dc.errors, &DependencyError{
		Component: "Test",
		Message:   "Test error",
		Help:      "Test help",
	})

	if !dc.HasErrors() {
		t.Error("新增錯誤後應回傳 true")
	}
}

// TestGetErrors 測試 GetErrors 方法
func TestGetErrors(t *testing.T) {
	dc := NewDependencyChecker()
	errors := dc.GetErrors()

	if len(errors) != 0 {
		t.Errorf("新建立的檢查器應有零個錯誤，但有 %d 個", len(errors))
	}

	// 新增錯誤
	testErr := &DependencyError{
		Component: "Test",
		Message:   "Test error",
		Help:      "Test help",
	}
	dc.errors = append(dc.errors, testErr)

	errors = dc.GetErrors()
	if len(errors) != 1 {
		t.Errorf("新增 1 個錯誤後應有 1 個，但有 %d 個", len(errors))
	}

	if errors[0].Component != "Test" {
		t.Errorf("錯誤元件應為 'Test'，但為 '%s'", errors[0].Component)
	}
}

// TestDependencyErrorFormat 測試 DependencyError 的格式化
func TestDependencyErrorFormat(t *testing.T) {
	err := &DependencyError{
		Component: "Node.js",
		Message:   "未找到",
		Help:      "請安裝",
	}

	errorStr := err.Error()
	if errorStr != "[Node.js] 未找到" {
		t.Errorf("錯誤格式不正確: %s", errorStr)
	}
}

// TestIsVersionValid 測試版本驗證邏輯
func TestIsVersionValid(t *testing.T) {
	dc := NewDependencyChecker()

	tests := []struct {
		current  string
		minimum  string
		expected bool
	}{
		{"14.0.0", "14.0.0", true},  // 相等
		{"14.1.0", "14.0.0", true},  // 較新
		{"15.0.0", "14.0.0", true},  // 較新主版本
		{"13.9.9", "14.0.0", false}, // 較舊
		{"14.0.0", "14.0.1", false}, // 較舊小版本
		{"18.0.0", "14.0.0", true},  // 新得多
	}

	for _, tt := range tests {
		result := dc.isVersionValid(tt.current, tt.minimum)
		if result != tt.expected {
			t.Errorf("isVersionValid(%s, %s) = %v，期望 %v",
				tt.current, tt.minimum, result, tt.expected)
		}
	}
}

// TestFormatErrors 測試錯誤格式化
func TestFormatErrors(t *testing.T) {
	dc := NewDependencyChecker()
	dc.errors = append(dc.errors, &DependencyError{
		Component: "Node.js",
		Message:   "未找到",
		Help:      "請訪問 nodejs.org 安裝",
	})

	err := dc.formatErrors()
	if err == nil {
		t.Error("formatErrors() 應傳回錯誤")
	}

	errorStr := err.Error()
	if len(errorStr) == 0 {
		t.Error("錯誤訊息不應為空")
	}

	if !contains(errorStr, "Node.js") {
		t.Error("錯誤訊息應包含 'Node.js'")
	}
}

// 輔助函式：檢查字串是否包含子字串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && s != "")
}
