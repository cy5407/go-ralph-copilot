package ghcopilot

import (
	"errors"
	"fmt"
)

// ErrorType 定義錯誤類型
type ErrorType string

const (
	// ErrorTypeTimeout 超時錯誤
	ErrorTypeTimeout ErrorType = "TIMEOUT"

	// ErrorTypeCircuitOpen 熔斷器開啟錯誤
	ErrorTypeCircuitOpen ErrorType = "CIRCUIT_OPEN"

	// ErrorTypeConfigError 配置錯誤
	ErrorTypeConfigError ErrorType = "CONFIG_ERROR"

	// ErrorTypeExecutionError 執行錯誤
	ErrorTypeExecutionError ErrorType = "EXECUTION_ERROR"

	// ErrorTypeParsingError 解析錯誤
	ErrorTypeParsingError ErrorType = "PARSING_ERROR"

	// ErrorTypeAuthError 認證錯誤
	ErrorTypeAuthError ErrorType = "AUTH_ERROR"

	// ErrorTypeNetworkError 網路錯誤
	ErrorTypeNetworkError ErrorType = "NETWORK_ERROR"

	// ErrorTypeQuotaError API 配額錯誤
	ErrorTypeQuotaError ErrorType = "QUOTA_ERROR"

	// ErrorTypeRetryExhausted 重試次數耗盡
	ErrorTypeRetryExhausted ErrorType = "RETRY_EXHAUSTED"

	// ErrorTypeInvalidInput 無效輸入
	ErrorTypeInvalidInput ErrorType = "INVALID_INPUT"

	// ErrorTypePersistenceError 持久化錯誤
	ErrorTypePersistenceError ErrorType = "PERSISTENCE_ERROR"
)

// RalphLoopError 統一的錯誤結構
type RalphLoopError struct {
	Type    ErrorType // 錯誤類型
	Message string    // 錯誤訊息
	Cause   error     // 原始錯誤
	Context map[string]interface{} // 額外上下文資訊
}

// Error 實作 error 介面
func (e *RalphLoopError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

// Unwrap 實作 errors.Unwrap 支援
func (e *RalphLoopError) Unwrap() error {
	return e.Cause
}

// Is 實作 errors.Is 支援
func (e *RalphLoopError) Is(target error) bool {
	t, ok := target.(*RalphLoopError)
	if !ok {
		return false
	}
	return e.Type == t.Type
}

// WithContext 添加上下文資訊
func (e *RalphLoopError) WithContext(key string, value interface{}) *RalphLoopError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// NewError 創建新的錯誤
func NewError(errType ErrorType, message string) *RalphLoopError {
	return &RalphLoopError{
		Type:    errType,
		Message: message,
	}
}

// WrapError 包裝現有錯誤
func WrapError(errType ErrorType, message string, cause error) *RalphLoopError {
	return &RalphLoopError{
		Type:    errType,
		Message: message,
		Cause:   cause,
	}
}

// 預定義的常見錯誤
var (
	// ErrTimeout 超時錯誤
	ErrTimeout = NewError(ErrorTypeTimeout, "操作超時")

	// ErrCircuitOpen 熔斷器開啟
	ErrCircuitOpen = NewError(ErrorTypeCircuitOpen, "熔斷器已開啟，停止執行")

	// ErrInvalidConfig 無效配置
	ErrInvalidConfig = NewError(ErrorTypeConfigError, "配置無效")

	// ErrAuthFailed 認證失敗
	ErrAuthFailed = NewError(ErrorTypeAuthError, "認證失敗")

	// ErrQuotaExceeded API 配額超限
	ErrQuotaExceeded = NewError(ErrorTypeQuotaError, "API 配額已超限")

	// ErrRetryExhausted 重試次數耗盡
	ErrRetryExhausted = NewError(ErrorTypeRetryExhausted, "重試次數已耗盡")

	// ErrInvalidInput 無效輸入
	ErrInvalidInput = NewError(ErrorTypeInvalidInput, "輸入無效")
)

// IsTimeout 檢查是否為超時錯誤
func IsTimeout(err error) bool {
	var ralphErr *RalphLoopError
	if errors.As(err, &ralphErr) {
		return ralphErr.Type == ErrorTypeTimeout
	}
	return false
}

// IsCircuitOpen 檢查是否為熔斷器錯誤
func IsCircuitOpen(err error) bool {
	var ralphErr *RalphLoopError
	if errors.As(err, &ralphErr) {
		return ralphErr.Type == ErrorTypeCircuitOpen
	}
	return false
}

// IsRetryable 檢查錯誤是否可重試
func IsRetryable(err error) bool {
	var ralphErr *RalphLoopError
	if errors.As(err, &ralphErr) {
		switch ralphErr.Type {
		case ErrorTypeTimeout, ErrorTypeNetworkError:
			return true
		case ErrorTypeCircuitOpen, ErrorTypeQuotaError, ErrorTypeAuthError, ErrorTypeConfigError:
			return false
		default:
			return true
		}
	}
	return true
}

// IsFatal 檢查錯誤是否為致命錯誤（不應重試）
func IsFatal(err error) bool {
	return !IsRetryable(err)
}

// GetErrorType 取得錯誤類型
func GetErrorType(err error) ErrorType {
	var ralphErr *RalphLoopError
	if errors.As(err, &ralphErr) {
		return ralphErr.Type
	}
	return ErrorTypeExecutionError
}

// FormatUserFriendlyError 格式化使用者友善的錯誤訊息
func FormatUserFriendlyError(err error) string {
	if err == nil {
		return ""
	}

	var ralphErr *RalphLoopError
	if !errors.As(err, &ralphErr) {
		return fmt.Sprintf("❌ 執行失敗: %v", err)
	}

	var suggestion string
	switch ralphErr.Type {
	case ErrorTypeTimeout:
		suggestion = "\n💡 建議: 請增加超時設定 (--timeout) 或檢查網路連線"
	case ErrorTypeCircuitOpen:
		suggestion = "\n💡 建議: 請執行 'ralph-loop reset' 重置熔斷器"
	case ErrorTypeAuthError:
		suggestion = "\n💡 建議: 請執行 'copilot auth' 重新認證"
	case ErrorTypeQuotaError:
		suggestion = "\n💡 建議: 請等待 API 配額重置或檢查訂閱狀態"
	case ErrorTypeConfigError:
		suggestion = "\n💡 建議: 請檢查配置檔案格式與參數設定"
	case ErrorTypeNetworkError:
		suggestion = "\n💡 建議: 請檢查網路連線與防火牆設定"
	}

	return fmt.Sprintf("❌ [%s] %s%s", ralphErr.Type, ralphErr.Message, suggestion)
}
