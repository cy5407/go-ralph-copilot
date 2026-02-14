package ghcopilot

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestOutputFormatter(t *testing.T) {
	// 創建測試數據
	results := []*LoopResult{
		{
			LoopID:          "test-1",
			LoopIndex:       1,
			ShouldContinue:  true,
			CompletionScore: 50,
			ExitReason:      "繼續執行",
			Timestamp:       time.Now(),
		},
		{
			LoopID:          "test-2",
			LoopIndex:       2,
			ShouldContinue:  false,
			CompletionScore: 100,
			ExitReason:      "任務完成",
			Timestamp:       time.Now(),
		},
	}
	totalTime := 5 * time.Minute
	
	t.Run("FormatJSON", func(t *testing.T) {
		var buf bytes.Buffer
		formatter := NewOutputFormatter(FormatJSON)
		formatter.SetWriter(&buf)
		
		err := formatter.FormatResults(results, totalTime, nil)
		if err != nil {
			t.Fatalf("FormatResults failed: %v", err)
		}
		
		// 驗證 JSON 格式
		var output map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
			t.Fatalf("Invalid JSON output: %v", err)
		}
		
		// 檢查必要欄位
		if output["total_loops"].(float64) != 2 {
			t.Error("Expected total_loops = 2")
		}
		if output["success"].(bool) != true {
			t.Error("Expected success = true")
		}
	})
	
	t.Run("FormatTable", func(t *testing.T) {
		var buf bytes.Buffer
		formatter := NewOutputFormatter(FormatTable)
		formatter.SetWriter(&buf)
		
		err := formatter.FormatResults(results, totalTime, nil)
		if err != nil {
			t.Fatalf("FormatResults failed: %v", err)
		}
		
		output := buf.String()
		
		// 驗證表格包含必要元素
		if !strings.Contains(output, "┌") || !strings.Contains(output, "└") {
			t.Error("Expected table borders")
		}
		if !strings.Contains(output, "迴圈") {
			t.Error("Expected table header")
		}
		if !strings.Contains(output, "總迴圈數: 2") {
			t.Error("Expected summary")
		}
	})
	
	t.Run("FormatText", func(t *testing.T) {
		var buf bytes.Buffer
		formatter := NewOutputFormatter(FormatText)
		formatter.SetWriter(&buf)
		
		err := formatter.FormatResults(results, totalTime, nil)
		if err != nil {
			t.Fatalf("FormatResults failed: %v", err)
		}
		
		output := buf.String()
		
		// 驗證文字輸出包含必要資訊
		if !strings.Contains(output, "執行結果摘要") {
			t.Error("Expected summary title")
		}
		if !strings.Contains(output, "總迴圈數: 2") {
			t.Error("Expected loop count")
		}
		if !strings.Contains(output, "迴圈歷史:") {
			t.Error("Expected history section")
		}
	})
	
	t.Run("FormatWithError", func(t *testing.T) {
		var buf bytes.Buffer
		formatter := NewOutputFormatter(FormatJSON)
		formatter.SetWriter(&buf)
		
		testErr := "測試錯誤"
		err := formatter.FormatResults(results, totalTime, &testError{testErr})
		if err != nil {
			t.Fatalf("FormatResults failed: %v", err)
		}
		
		var output map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
			t.Fatalf("Invalid JSON output: %v", err)
		}
		
		// 驗證錯誤訊息
		if output["success"].(bool) != false {
			t.Error("Expected success = false")
		}
		if output["error"].(string) != testErr {
			t.Error("Expected error message in output")
		}
	})
}

func TestFormatStatus(t *testing.T) {
	status := &ClientStatus{
		Initialized:         true,
		Closed:              false,
		CircuitBreakerOpen:  false,
		CircuitBreakerState: StateClosed,
		LoopsExecuted:       5,
		Summary: map[string]interface{}{
			"total_time": "5m",
			"success":    true,
		},
	}
	
	t.Run("FormatStatusJSON", func(t *testing.T) {
		var buf bytes.Buffer
		formatter := NewOutputFormatter(FormatJSON)
		formatter.SetWriter(&buf)
		
		err := formatter.FormatStatus(status)
		if err != nil {
			t.Fatalf("FormatStatus failed: %v", err)
		}
		
		// 驗證 JSON 格式
		var output map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
			t.Fatalf("Invalid JSON output: %v", err)
		}
		
		if output["Initialized"].(bool) != true {
			t.Error("Expected Initialized = true")
		}
		if output["LoopsExecuted"].(float64) != 5 {
			t.Error("Expected LoopsExecuted = 5")
		}
	})
	
	t.Run("FormatStatusTable", func(t *testing.T) {
		var buf bytes.Buffer
		formatter := NewOutputFormatter(FormatTable)
		formatter.SetWriter(&buf)
		
		err := formatter.FormatStatus(status)
		if err != nil {
			t.Fatalf("FormatStatus failed: %v", err)
		}
		
		output := buf.String()
		
		// 驗證表格包含狀態資訊
		if !strings.Contains(output, "初始化") {
			t.Error("Expected status field")
		}
		if !strings.Contains(output, "摘要:") {
			t.Error("Expected summary section")
		}
	})
	
	t.Run("FormatStatusText", func(t *testing.T) {
		var buf bytes.Buffer
		formatter := NewOutputFormatter(FormatText)
		formatter.SetWriter(&buf)
		
		err := formatter.FormatStatus(status)
		if err != nil {
			t.Fatalf("FormatStatus failed: %v", err)
		}
		
		output := buf.String()
		
		// 驗證文字輸出
		if !strings.Contains(output, "Ralph Loop 狀態") {
			t.Error("Expected status title")
		}
		if !strings.Contains(output, "初始化: true") {
			t.Error("Expected initialized status")
		}
	})
}

// 測試用錯誤類型
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
