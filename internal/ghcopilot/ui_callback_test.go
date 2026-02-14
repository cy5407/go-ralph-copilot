package ghcopilot

import (
	"bytes"
	"testing"
	"time"
)

func TestUICallback(t *testing.T) {
	t.Run("DefaultUICallback basic operations", func(t *testing.T) {
		callback := NewDefaultUICallback(false, false)
		
		// 測試迴圈開始
		callback.OnLoopStart(1, 10)
		
		// 測試迴圈完成
		result := &LoopResult{
			LoopID:          "test-1",
			LoopIndex:       1,
			ShouldContinue:  true,
			CompletionScore: 50,
			ExitReason:      "continuing",
			Timestamp:       time.Now(),
		}
		callback.OnLoopComplete(1, result)
		
		// 測試進度
		callback.OnProgress("測試進度訊息")
		
		// 測試警告
		callback.OnWarning("測試警告訊息")
		
		// 測試完成
		callback.OnComplete(10, nil)
	})
	
	t.Run("DefaultUICallback with verbose", func(t *testing.T) {
		var buf bytes.Buffer
		callback := NewDefaultUICallback(true, false)
		callback.writer = &buf
		
		callback.OnVerbose("詳細訊息")
		
		// 驗證有輸出
		if buf.Len() == 0 {
			t.Error("Expected verbose output but got none")
		}
	})
	
	t.Run("DefaultUICallback with quiet", func(t *testing.T) {
		var buf bytes.Buffer
		callback := NewDefaultUICallback(false, true)
		callback.writer = &buf
		
		callback.OnLoopStart(1, 10)
		callback.OnProgress("測試")
		
		// 靜默模式下應該沒有輸出
		if buf.Len() > 0 {
			t.Error("Expected no output in quiet mode but got some")
		}
	})
	
	t.Run("makeErrorActionable", func(t *testing.T) {
		tests := []struct {
			errMsg   string
			wantHint bool
		}{
			{"command not found", true},
			{"timeout occurred", true},
			{"402 quota exceeded", true},
			{"401 unauthorized", true},
			{"circuit breaker opened", true},
			{"no progress detected", true},
			{"connection refused", true},
			{"random error", false},
		}
		
		for _, tt := range tests {
			hint := makeErrorActionable(tt.errMsg)
			hasHint := hint != ""
			if hasHint != tt.wantHint {
				t.Errorf("makeErrorActionable(%q) hasHint = %v, want %v", tt.errMsg, hasHint, tt.wantHint)
			}
		}
	})
}

func TestColorize(t *testing.T) {
	t.Run("colorize with enabled", func(t *testing.T) {
		EnableColor()
		result := colorize("test", colorRed)
		if result == "test" {
			t.Error("Expected colored output but got plain text")
		}
	})
	
	t.Run("colorize with disabled", func(t *testing.T) {
		DisableColor()
		result := colorize("test", colorRed)
		if result != "test" {
			t.Error("Expected plain text but got colored output")
		}
	})
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		want     string
	}{
		{30 * time.Second, "30s"},
		{90 * time.Second, "1m30s"},
		{3665 * time.Second, "1h1m5s"},
		{0, "0s"},
	}
	
	for _, tt := range tests {
		got := formatDuration(tt.duration)
		if got != tt.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.duration, got, tt.want)
		}
	}
}
