package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"
)

func TestLogger_Basic(t *testing.T) {
	var buf bytes.Buffer
	
	logger := &Logger{
		level:      DEBUG,
		jsonFormat: true,
		outputs:    []io.Writer{&buf},
		fields:     make(map[string]interface{}),
		component:  "test",
	}

	logger.Info("test message")
	
	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("期望包含 'test message'，但得到: %s", output)
	}
	
	// 驗證 JSON 格式
	var entry LogEntry
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Errorf("無法解析 JSON 輸出: %v", err)
	}
	
	if entry.Message != "test message" {
		t.Errorf("期望訊息為 'test message'，但得到: %s", entry.Message)
	}
	
	if entry.Level != "INFO" {
		t.Errorf("期望級別為 'INFO'，但得到: %s", entry.Level)
	}
	
	if entry.Component != "test" {
		t.Errorf("期望組件為 'test'，但得到: %s", entry.Component)
	}
}

func TestLogger_WithFields(t *testing.T) {
	var buf bytes.Buffer
	
	logger := &Logger{
		level:      DEBUG,
		jsonFormat: true,
		outputs:    []io.Writer{&buf},
		fields:     make(map[string]interface{}),
	}

	logger.WithFields(map[string]interface{}{
		"request_id": "req-123",
		"loop_id":    "loop-456",
		"custom":     "value",
	}).Info("test message with fields")
	
	output := buf.String()
	
	var entry LogEntry
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Errorf("無法解析 JSON 輸出: %v", err)
	}
	
	// 檢查提升到頂層的字段
	if entry.RequestID != "req-123" {
		t.Errorf("期望 RequestID 為 'req-123'，但得到: %s", entry.RequestID)
	}
	
	if entry.LoopID != "loop-456" {
		t.Errorf("期望 LoopID 為 'loop-456'，但得到: %s", entry.LoopID)
	}
	
	// 檢查自定義字段
	if entry.Fields["custom"] != "value" {
		t.Errorf("期望自定義字段 'custom' 為 'value'，但得到: %v", entry.Fields["custom"])
	}
}

func TestLogger_Levels(t *testing.T) {
	var buf bytes.Buffer
	
	logger := &Logger{
		level:      WARN,
		jsonFormat: false,
		outputs:    []io.Writer{&buf},
		fields:     make(map[string]interface{}),
	}

	// 這些應該被過濾掉
	logger.Debug("debug message")
	logger.Info("info message")
	
	// 這些應該顯示
	logger.Warn("warn message")
	logger.Error("error message")
	
	output := buf.String()
	
	if strings.Contains(output, "debug message") {
		t.Error("DEBUG 訊息不應該顯示")
	}
	
	if strings.Contains(output, "info message") {
		t.Error("INFO 訊息不應該顯示")
	}
	
	if !strings.Contains(output, "warn message") {
		t.Error("WARN 訊息應該顯示")
	}
	
	if !strings.Contains(output, "error message") {
		t.Error("ERROR 訊息應該顯示")
	}
}

func TestLogger_TextFormat(t *testing.T) {
	var buf bytes.Buffer
	
	logger := &Logger{
		level:      INFO,
		jsonFormat: false,
		outputs:    []io.Writer{&buf},
		fields:     make(map[string]interface{}),
		component:  "test",
	}

	logger.WithRequestID("req-123456789").
		WithDuration(100*time.Millisecond).
		WithError(fmt.Errorf("test error")).
		Info("test message")
	
	output := buf.String()
	
	// 檢查文字格式的各個部分
	if !strings.Contains(output, "[INFO") {
		t.Error("輸出應該包含 '[INFO'")
	}
	
	if !strings.Contains(output, "[test]") {
		t.Error("輸出應該包含 '[test]'")
	}
	
	if !strings.Contains(output, "[req:req-1234") { // 截斷到 8 個字符
		t.Error("輸出應該包含截斷的請求 ID")
	}
	
	if !strings.Contains(output, "test message") {
		t.Error("輸出應該包含訊息")
	}
	
	if !strings.Contains(output, "(耗時:100ms)") {
		t.Error("輸出應該包含耗時資訊")
	}
	
	if !strings.Contains(output, "error: test error") {
		t.Error("輸出應該包含錯誤資訊")
	}
}

func TestLogLevel_String(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{DEBUG, "DEBUG"},
		{INFO, "INFO"},
		{WARN, "WARN"},
		{ERROR, "ERROR"},
		{FATAL, "FATAL"},
		{LogLevel(99), "UNKNOWN"},
	}

	for _, test := range tests {
		if test.level.String() != test.expected {
			t.Errorf("期望 %v.String() = %s，但得到: %s", test.level, test.expected, test.level.String())
		}
	}
}

func TestGlobalFunctions(t *testing.T) {
	// 測試全域函數是否正常工作
	// 注意：這些會輸出到 stdout，在實際測試中可能需要重定向
	
	Debug("global debug")
	Info("global info")
	Warn("global warn")
	Error("global error")
	
	// 測試有字段的全域函數
	WithField("test", "value").Info("with field")
	WithFields(map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}).Info("with fields")
	
	WithRequestID("req-789").Info("with request id")
	WithLoopID("loop-123").Info("with loop id")
}

func TestLogger_WithError(t *testing.T) {
	var buf bytes.Buffer
	
	logger := &Logger{
		level:      INFO,
		jsonFormat: true,
		outputs:    []io.Writer{&buf},
		fields:     make(map[string]interface{}),
	}

	testErr := fmt.Errorf("test error message")
	logger.WithError(testErr).Error("operation failed")
	
	output := buf.String()
	
	var entry LogEntry
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Errorf("無法解析 JSON 輸出: %v", err)
	}
	
	if entry.Error != "test error message" {
		t.Errorf("期望錯誤訊息為 'test error message'，但得到: %s", entry.Error)
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	
	if config.Level != INFO {
		t.Errorf("期望預設級別為 INFO，但得到: %v", config.Level)
	}
	
	if !config.JSONFormat {
		t.Error("期望預設 JSON 格式為 true")
	}
	
	if !config.EnableCaller {
		t.Error("期望預設啟用調用者為 true")
	}
	
	if config.Component != "ralph-loop" {
		t.Errorf("期望預設組件為 'ralph-loop'，但得到: %s", config.Component)
	}
}