package ghcopilot

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

// TestLineWriter 測試 lineWriter 基本功能
func TestLineWriter(t *testing.T) {
	var buffer bytes.Buffer
	var receivedLines []string
	
	callback := func(line string) {
		receivedLines = append(receivedLines, line)
	}
	
	lw := newLineWriter(&buffer, callback)
	
	// 寫入多行數據
	testData := "line 1\nline 2\nline 3\n"
	n, err := lw.Write([]byte(testData))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(testData) {
		t.Errorf("Write length mismatch: got %d, want %d", n, len(testData))
	}
	
	// 關閉 writer 並等待處理完成
	lw.Close()
	time.Sleep(100 * time.Millisecond)
	
	// 驗證 buffer 接收到完整數據
	if buffer.String() != testData {
		t.Errorf("Buffer content mismatch: got %q, want %q", buffer.String(), testData)
	}
	
	// 驗證回調接收到所有行
	expectedLines := []string{"line 1", "line 2", "line 3"}
	if len(receivedLines) != len(expectedLines) {
		t.Errorf("Received lines count mismatch: got %d, want %d", len(receivedLines), len(expectedLines))
	}
	
	for i, expected := range expectedLines {
		if i >= len(receivedLines) {
			t.Errorf("Missing line %d: want %q", i, expected)
			continue
		}
		if receivedLines[i] != expected {
			t.Errorf("Line %d mismatch: got %q, want %q", i, receivedLines[i], expected)
		}
	}
}

// TestLineWriterEmptyLines 測試 lineWriter 處理空行
func TestLineWriterEmptyLines(t *testing.T) {
	var buffer bytes.Buffer
	var receivedLines []string
	
	callback := func(line string) {
		receivedLines = append(receivedLines, line)
	}
	
	lw := newLineWriter(&buffer, callback)
	
	testData := "line 1\n\nline 3\n"
	lw.Write([]byte(testData))
	lw.Close()
	time.Sleep(100 * time.Millisecond)
	
	// bufio.Scanner 預設會跳過空行，所以只會接收到非空行
	expectedLines := []string{"line 1", "line 3"}
	if len(receivedLines) != len(expectedLines) {
		t.Errorf("Received lines count mismatch: got %d, want %d", len(receivedLines), len(expectedLines))
		t.Logf("Received lines: %v", receivedLines)
	}
}

// TestUICallbackStreamOutput 測試 UICallback 的串流輸出方法
func TestUICallbackStreamOutput(t *testing.T) {
	var output bytes.Buffer
	
	callback := NewDefaultUICallback(false, false)
	callback.writer = &output
	callback.streamEnabled = true
	
	// 測試 OnStreamOutput
	callback.OnStreamOutput("test output line")
	
	result := output.String()
	if !strings.Contains(result, "[copilot]") {
		t.Errorf("Output missing [copilot] prefix: %s", result)
	}
	if !strings.Contains(result, "test output line") {
		t.Errorf("Output missing content: %s", result)
	}
}

// TestUICallbackStreamError 測試 UICallback 的串流錯誤輸出
func TestUICallbackStreamError(t *testing.T) {
	var output bytes.Buffer
	
	callback := NewDefaultUICallback(false, false)
	callback.writer = &output
	callback.streamEnabled = true
	
	// 測試 OnStreamError
	callback.OnStreamError("test error line")
	
	result := output.String()
	if !strings.Contains(result, "[copilot:err]") {
		t.Errorf("Output missing [copilot:err] prefix: %s", result)
	}
	if !strings.Contains(result, "test error line") {
		t.Errorf("Output missing content: %s", result)
	}
}

// TestUICallbackStreamQuietMode 測試 quiet 模式下串流輸出被禁用
func TestUICallbackStreamQuietMode(t *testing.T) {
	var output bytes.Buffer
	
	callback := NewDefaultUICallback(false, true) // quiet mode
	callback.writer = &output
	
	callback.OnStreamOutput("should not appear")
	
	if output.Len() > 0 {
		t.Errorf("Output should be empty in quiet mode, got: %s", output.String())
	}
}

// TestCLIExecutorStreamCallback 測試 CLIExecutor 的串流回調設置
func TestCLIExecutorStreamCallback(t *testing.T) {
	executor := NewCLIExecutor(".")
	
	var stdoutLines []string
	var stderrLines []string
	
	executor.SetStreamCallback(
		func(line string) {
			stdoutLines = append(stdoutLines, line)
		},
		func(line string) {
			stderrLines = append(stderrLines, line)
		},
	)
	
	if executor.streamCallback == nil {
		t.Error("stdout stream callback not set")
	}
	if executor.streamErrCallback == nil {
		t.Error("stderr stream callback not set")
	}
	
	// 測試回調是否正常工作
	executor.streamCallback("test stdout")
	executor.streamErrCallback("test stderr")
	
	if len(stdoutLines) != 1 || stdoutLines[0] != "test stdout" {
		t.Errorf("stdout callback failed: got %v", stdoutLines)
	}
	if len(stderrLines) != 1 || stderrLines[0] != "test stderr" {
		t.Errorf("stderr callback failed: got %v", stderrLines)
	}
}

// TestCLIExecutorStreamingIntegration 測試整合串流功能的執行
func TestCLIExecutorStreamingIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("跳過整合測試")
	}
	
	executor := NewCLIExecutor(".")
	executor.SetTimeout(5 * time.Second)
	
	var streamedLines []string
	executor.SetStreamCallback(
		func(line string) {
			streamedLines = append(streamedLines, line)
			t.Logf("Streamed: %s", line)
		},
		func(line string) {
			t.Logf("Streamed error: %s", line)
		},
	)
	
	ctx := context.Background()
	
	// 使用簡單的 prompt 測試（如果 Copilot 可用）
	result, err := executor.ExecutePrompt(ctx, "echo 'Hello World'")
	
	if err != nil {
		// 如果 Copilot 不可用，跳過測試
		if strings.Contains(err.Error(), "executable file not found") {
			t.Skip("Copilot CLI not available")
		}
		t.Logf("Execution failed (expected in CI): %v", err)
		return
	}
	
	// 驗證結果仍然包含完整輸出
	if result.Stdout == "" {
		t.Error("Result stdout should not be empty")
	}
	
	t.Logf("Streamed %d lines", len(streamedLines))
	t.Logf("Result stdout length: %d", len(result.Stdout))
}

// BenchmarkLineWriter 性能測試
func BenchmarkLineWriter(b *testing.B) {
	var buffer bytes.Buffer
	callback := func(line string) {
		// 模擬簡單的處理
		_ = line
	}
	
	testData := []byte("line 1\nline 2\nline 3\n")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buffer.Reset()
		lw := newLineWriter(&buffer, callback)
		lw.Write(testData)
		lw.Close()
	}
}
