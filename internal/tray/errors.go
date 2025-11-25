package tray

import (
	"fmt"
)

// TrayError 托盘相关错误的包装器
type TrayError struct {
	Code    TrayErrorCode
	Message string
	Cause   error
}

// TrayErrorCode 托盘错误代码类型
type TrayErrorCode int

const (
	// ErrCodeUnknown 未知错误
	ErrCodeUnknown TrayErrorCode = iota
	// ErrCodeInitialization 初始化失败
	ErrCodeInitialization
	// ErrCodeIconLoad 图标加载失败
	ErrCodeIconLoad
	// ErrCodeMenuCreation 菜单创建失败
	ErrCodeMenuCreation
	// ErrCodeConfiguration 配置错误
	ErrCodeConfiguration
	// ErrCodePermission 权限错误
	ErrCodePermission
	// ErrCodeResource 资源不足
	ErrCodeResource
	// ErrCodeRuntime 运行时错误
	ErrCodeRuntime
)

// Error 实现error接口
func (te *TrayError) Error() string {
	if te.Cause != nil {
		return fmt.Sprintf("[%d] %s: %v", te.Code, te.Message, te.Cause)
	}
	return fmt.Sprintf("[%d] %s", te.Code, te.Message)
}

// Unwrap 支持错误链
func (te *TrayError) Unwrap() error {
	return te.Cause
}

// NewTrayError 创建新的托盘错误
func NewTrayError(code TrayErrorCode, message string, cause error) *TrayError {
	return &TrayError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// IsTrayError 检查是否为托盘错误
func IsTrayError(err error) (*TrayError, bool) {
	if trayErr, ok := err.(*TrayError); ok {
		return trayErr, true
	}
	return nil, false
}

// Predefined error messages
var (
	ErrNotInitialized    = NewTrayError(ErrCodeInitialization, "tray manager not initialized", nil)
	ErrAlreadyRunning    = NewTrayError(ErrCodeRuntime, "tray manager already running", nil)
	ErrIconNotFound      = NewTrayError(ErrCodeIconLoad, "tray icon file not found", nil)
	ErrInvalidIconFormat = NewTrayError(ErrCodeIconLoad, "invalid icon format", nil)
	ErrMenuEmpty         = NewTrayError(ErrCodeMenuCreation, "tray menu is empty", nil)
	ErrConfigMissing     = NewTrayError(ErrCodeConfiguration, "required configuration missing", nil)
)