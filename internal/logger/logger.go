package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LogLevel 定義日誌級別
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

// String 返回日誌級別的字串表示
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// LogEntry 表示一個日誌條目
type LogEntry struct {
	Timestamp   time.Time              `json:"timestamp"`
	Level       string                 `json:"level"`
	Message     string                 `json:"message"`
	Component   string                 `json:"component,omitempty"`
	RequestID   string                 `json:"request_id,omitempty"`
	LoopID      string                 `json:"loop_id,omitempty"`
	Fields      map[string]interface{} `json:"fields,omitempty"`
	Caller      string                 `json:"caller,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Duration    string                 `json:"duration,omitempty"`
}

// Logger 提供結構化日誌功能
type Logger struct {
	mu           sync.RWMutex
	level        LogLevel
	outputs      []io.Writer
	jsonFormat   bool
	enableCaller bool
	component    string
	fields       map[string]interface{}
}

// Config 日誌配置
type Config struct {
	Level        LogLevel
	JSONFormat   bool
	EnableCaller bool
	OutputFile   string
	Component    string
}

// DefaultConfig 返回預設配置
func DefaultConfig() *Config {
	return &Config{
		Level:        INFO,
		JSONFormat:   true,
		EnableCaller: true,
		Component:    "ralph-loop",
	}
}

// New 創建新的日誌器
func New(config *Config) (*Logger, error) {
	logger := &Logger{
		level:        config.Level,
		jsonFormat:   config.JSONFormat,
		enableCaller: config.EnableCaller,
		component:    config.Component,
		fields:       make(map[string]interface{}),
		outputs:      []io.Writer{os.Stdout},
	}

	// 如果指定了輸出文件，添加文件輸出
	if config.OutputFile != "" {
		if err := os.MkdirAll(filepath.Dir(config.OutputFile), 0755); err != nil {
			return nil, fmt.Errorf("建立日誌目錄失敗: %w", err)
		}
		
		file, err := os.OpenFile(config.OutputFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("開啟日誌文件失敗: %w", err)
		}
		
		logger.outputs = append(logger.outputs, file)
	}

	return logger, nil
}

// WithField 添加字段
func (l *Logger) WithField(key string, value interface{}) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// 複製欄位而不拷貝 Mutex
	newFields := make(map[string]interface{}, len(l.fields)+1)
	for k, v := range l.fields {
		newFields[k] = v
	}
	newFields[key] = value
	
	return &Logger{
		level:        l.level,
		outputs:      l.outputs,
		jsonFormat:   l.jsonFormat,
		enableCaller: l.enableCaller,
		component:    l.component,
		fields:       newFields,
	}
}

// WithFields 添加多個字段
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// 複製欄位而不拷貝 Mutex
	newFields := make(map[string]interface{}, len(l.fields)+len(fields))
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}
	
	return &Logger{
		level:        l.level,
		outputs:      l.outputs,
		jsonFormat:   l.jsonFormat,
		enableCaller: l.enableCaller,
		component:    l.component,
		fields:       newFields,
	}
}

// WithRequestID設置請求 ID
func (l *Logger) WithRequestID(requestID string) *Logger {
	return l.WithField("request_id", requestID)
}

// WithLoopID 設置迴圈 ID
func (l *Logger) WithLoopID(loopID string) *Logger {
	return l.WithField("loop_id", loopID)
}

// WithComponent 設置組件名稱
func (l *Logger) WithComponent(component string) *Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	// 複製欄位而不拷貝 Mutex
	newFields := make(map[string]interface{}, len(l.fields))
	for k, v := range l.fields {
		newFields[k] = v
	}
	
	return &Logger{
		level:        l.level,
		outputs:      l.outputs,
		jsonFormat:   l.jsonFormat,
		enableCaller: l.enableCaller,
		component:    component,
		fields:       newFields,
	}
}

// WithDuration 記錄執行時間
func (l *Logger) WithDuration(duration time.Duration) *Logger {
	return l.WithField("duration", duration.String())
}

// WithError 記錄錯誤
func (l *Logger) WithError(err error) *Logger {
	if err != nil {
		return l.WithField("error", err.Error())
	}
	return l
}

// Debug 輸出除錯級別日誌
func (l *Logger) Debug(message string) {
	l.log(DEBUG, message, nil)
}

// Debugf 輸出格式化除錯級別日誌
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.log(DEBUG, fmt.Sprintf(format, args...), nil)
}

// Info 輸出資訊級別日誌
func (l *Logger) Info(message string) {
	l.log(INFO, message, nil)
}

// Infof 輸出格式化資訊級別日誌
func (l *Logger) Infof(format string, args ...interface{}) {
	l.log(INFO, fmt.Sprintf(format, args...), nil)
}

// Warn 輸出警告級別日誌
func (l *Logger) Warn(message string) {
	l.log(WARN, message, nil)
}

// Warnf 輸出格式化警告級別日誌
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.log(WARN, fmt.Sprintf(format, args...), nil)
}

// Error 輸出錯誤級別日誌
func (l *Logger) Error(message string) {
	l.log(ERROR, message, nil)
}

// Errorf 輸出格式化錯誤級別日誌
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log(ERROR, fmt.Sprintf(format, args...), nil)
}

// Fatal 輸出致命級別日誌並退出程式
func (l *Logger) Fatal(message string) {
	l.log(FATAL, message, nil)
	os.Exit(1)
}

// Fatalf 輸出格式化致命級別日誌並退出程式
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.log(FATAL, fmt.Sprintf(format, args...), nil)
	os.Exit(1)
}

// log 內部日誌輸出方法
func (l *Logger) log(level LogLevel, message string, extraFields map[string]interface{}) {
	if level < l.level {
		return
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level.String(),
		Message:   message,
		Component: l.component,
		Fields:    make(map[string]interface{}),
	}

	// 複製現有字段
	for k, v := range l.fields {
		entry.Fields[k] = v
	}

	// 添加額外字段
	for k, v := range extraFields {
		entry.Fields[k] = v
	}

	// 如果沒有字段，設置為 nil 以避免 JSON 中出現空物件
	if len(entry.Fields) == 0 {
		entry.Fields = nil
	}

	// 提取常用字段到頂層
	if requestID, ok := entry.Fields["request_id"]; ok {
		entry.RequestID = fmt.Sprintf("%v", requestID)
		delete(entry.Fields, "request_id")
	}
	if loopID, ok := entry.Fields["loop_id"]; ok {
		entry.LoopID = fmt.Sprintf("%v", loopID)
		delete(entry.Fields, "loop_id")
	}
	if duration, ok := entry.Fields["duration"]; ok {
		entry.Duration = fmt.Sprintf("%v", duration)
		delete(entry.Fields, "duration")
	}
	if err, ok := entry.Fields["error"]; ok {
		entry.Error = fmt.Sprintf("%v", err)
		delete(entry.Fields, "error")
	}

	// 輸出日誌
	output := l.formatEntry(&entry)
	for _, writer := range l.outputs {
		fmt.Fprintln(writer, output)
	}
}

// formatEntry 格式化日誌條目
func (l *Logger) formatEntry(entry *LogEntry) string {
	if l.jsonFormat {
		data, _ := json.Marshal(entry)
		return string(data)
	}

	// 文字格式輸出
	timestamp := entry.Timestamp.Format("15:04:05.000")
	output := fmt.Sprintf("[%s %s]", entry.Level, timestamp)
	
	if entry.Component != "" {
		output += fmt.Sprintf(" [%s]", entry.Component)
	}
	
	if entry.RequestID != "" {
		reqID := entry.RequestID
		if len(reqID) > 8 {
			reqID = reqID[:8]
		}
		output += fmt.Sprintf(" [req:%s]", reqID)
	}
	
	if entry.LoopID != "" {
		output += fmt.Sprintf(" [loop:%s]", entry.LoopID)
	}
	
	output += fmt.Sprintf(" %s", entry.Message)
	
	if entry.Duration != "" {
		output += fmt.Sprintf(" (耗時:%s)", entry.Duration)
	}
	
	if entry.Error != "" {
		output += fmt.Sprintf(" error: %s", entry.Error)
	}

	return output
}

// SetLevel 設置日誌級別
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// GetLevel 獲取當前日誌級別
func (l *Logger) GetLevel() LogLevel {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.level
}

// Close 關閉日誌器
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	for _, output := range l.outputs {
		if closer, ok := output.(io.Closer); ok && output != os.Stdout && output != os.Stderr {
			if err := closer.Close(); err != nil {
				return err
			}
		}
	}
	
	return nil
}

// 全域日誌器實例
var defaultLogger *Logger

// init 初始化預設日誌器
func init() {
	config := DefaultConfig()
	
	// 從環境變數讀取配置
	if debugEnv := os.Getenv("RALPH_DEBUG"); debugEnv == "1" {
		config.Level = DEBUG
	}
	
	if logFile := os.Getenv("RALPH_LOG_FILE"); logFile != "" {
		config.OutputFile = logFile
	}

	var err error
	defaultLogger, err = New(config)
	if err != nil {
		// 如果創建失敗，使用基本日誌器
		defaultLogger = &Logger{
			level:      INFO,
			jsonFormat: false,
			outputs:    []io.Writer{os.Stdout},
			fields:     make(map[string]interface{}),
		}
	}
}

// 全域日誌函數

// Debug 全域除錯日誌
func Debug(message string) {
	defaultLogger.Debug(message)
}

// Debugf 全域格式化除錯日誌
func Debugf(format string, args ...interface{}) {
	defaultLogger.Debugf(format, args...)
}

// Info 全域資訊日誌
func Info(message string) {
	defaultLogger.Info(message)
}

// Infof 全域格式化資訊日誌
func Infof(format string, args ...interface{}) {
	defaultLogger.Infof(format, args...)
}

// Warn 全域警告日誌
func Warn(message string) {
	defaultLogger.Warn(message)
}

// Warnf 全域格式化警告日誌
func Warnf(format string, args ...interface{}) {
	defaultLogger.Warnf(format, args...)
}

// Error 全域錯誤日誌
func Error(message string) {
	defaultLogger.Error(message)
}

// Errorf 全域格式化錯誤日誌
func Errorf(format string, args ...interface{}) {
	defaultLogger.Errorf(format, args...)
}

// WithField 全域字段日誌
func WithField(key string, value interface{}) *Logger {
	return defaultLogger.WithField(key, value)
}

// WithFields 全域多字段日誌
func WithFields(fields map[string]interface{}) *Logger {
	return defaultLogger.WithFields(fields)
}

// WithRequestID 全域請求 ID 日誌
func WithRequestID(requestID string) *Logger {
	return defaultLogger.WithRequestID(requestID)
}

// WithLoopID 全域迴圈 ID 日誌
func WithLoopID(loopID string) *Logger {
	return defaultLogger.WithLoopID(loopID)
}

// SetGlobalLevel 設置全域日誌級別
func SetGlobalLevel(level LogLevel) {
	defaultLogger.SetLevel(level)
}