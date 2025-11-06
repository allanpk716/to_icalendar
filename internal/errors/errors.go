package errors

import (
	"fmt"
	"net/http"
	"time"
)

// ErrorType represents different types of errors in the application
type ErrorType string

const (
	ErrorTypeValidation     ErrorType = "validation"
	ErrorTypeAPI           ErrorType = "api"
	ErrorTypeClipboard     ErrorType = "clipboard"
	ErrorTypeFileOperation ErrorType = "file_operation"
	ErrorTypeProcessing    ErrorType = "processing"
	ErrorTypeConfiguration ErrorType = "configuration"
	ErrorTypeTimeout       ErrorType = "timeout"
	ErrorTypeNetwork       ErrorType = "network"
	ErrorTypeMemory        ErrorType = "memory"
)

// AppError represents a structured application error
type AppError struct {
	Type        ErrorType `json:"type"`
	Code        string    `json:"code"`
	Message     string    `json:"message"`
	Details     string    `json:"details,omitempty"`
	HTTPStatus  int       `json:"http_status,omitempty"`
	Retryable   bool      `json:"retryable"`
	Cause       error     `json:"-"`
	Timestamp   int64     `json:"timestamp"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("[%s:%s] %s: %s", e.Type, e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%s:%s] %s", e.Type, e.Code, e.Message)
}

// Unwrap returns the underlying cause
func (e *AppError) Unwrap() error {
	return e.Cause
}

// NewValidationError creates a new validation error
func NewValidationError(code, message string, details ...string) *AppError {
	err := &AppError{
		Type:       ErrorTypeValidation,
		Code:       code,
		Message:    message,
		HTTPStatus: http.StatusBadRequest,
		Retryable:  false,
		Timestamp:  timestamp(),
	}
	if len(details) > 0 {
		err.Details = details[0]
	}
	return err
}

// NewAPIError creates a new API error
func NewAPIError(code, message string, httpStatus int, retryable bool, cause error) *AppError {
	return &AppError{
		Type:       ErrorTypeAPI,
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		Retryable:  retryable,
		Cause:      cause,
		Timestamp:  timestamp(),
	}
}

// NewClipboardError creates a new clipboard error
func NewClipboardError(code, message string, cause error) *AppError {
	return &AppError{
		Type:       ErrorTypeClipboard,
		Code:       code,
		Message:    message,
		HTTPStatus: http.StatusInternalServerError,
		Retryable:  false,
		Cause:      cause,
		Timestamp:  timestamp(),
	}
}

// NewFileOperationError creates a new file operation error
func NewFileOperationError(code, message, filePath string, cause error) *AppError {
	details := fmt.Sprintf("file: %s", filePath)
	return &AppError{
		Type:       ErrorTypeFileOperation,
		Code:       code,
		Message:    message,
		Details:    details,
		HTTPStatus: http.StatusInternalServerError,
		Retryable:  false,
		Cause:      cause,
		Timestamp:  timestamp(),
	}
}

// NewProcessingError creates a new processing error
func NewProcessingError(code, message string, retryable bool, cause error) *AppError {
	return &AppError{
		Type:       ErrorTypeProcessing,
		Code:       code,
		Message:    message,
		HTTPStatus: http.StatusInternalServerError,
		Retryable:  retryable,
		Cause:      cause,
		Timestamp:  timestamp(),
	}
}

// NewConfigurationError creates a new configuration error
func NewConfigurationError(code, message string, details ...string) *AppError {
	err := &AppError{
		Type:       ErrorTypeConfiguration,
		Code:       code,
		Message:    message,
		HTTPStatus: http.StatusInternalServerError,
		Retryable:  false,
		Timestamp:  timestamp(),
	}
	if len(details) > 0 {
		err.Details = details[0]
	}
	return err
}

// NewTimeoutError creates a new timeout error
func NewTimeoutError(operation string, timeout int) *AppError {
	message := fmt.Sprintf("Operation '%s' timed out after %d seconds", operation, timeout)
	return &AppError{
		Type:       ErrorTypeTimeout,
		Code:       "timeout",
		Message:    message,
		HTTPStatus: http.StatusRequestTimeout,
		Retryable:  true,
		Timestamp:  timestamp(),
	}
}

// NewNetworkError creates a new network error
func NewNetworkError(code, message string, retryable bool, cause error) *AppError {
	return &AppError{
		Type:       ErrorTypeNetwork,
		Code:       code,
		Message:    message,
		HTTPStatus: http.StatusBadGateway,
		Retryable:  retryable,
		Cause:      cause,
		Timestamp:  timestamp(),
	}
}

// NewMemoryError creates a new memory error
func NewMemoryError(code, message string, cause error) *AppError {
	return &AppError{
		Type:       ErrorTypeMemory,
		Code:       code,
		Message:    message,
		HTTPStatus: http.StatusInternalServerError,
		Retryable:  false,
		Cause:      cause,
		Timestamp:  timestamp(),
	}
}

// WrapError wraps an existing error with additional context
func WrapError(err error, errorType ErrorType, code, message string) *AppError {
	if err == nil {
		return nil
	}

	// If it's already an AppError, add context
	if appErr, ok := err.(*AppError); ok {
		details := message
		if appErr.Details != "" {
			details = fmt.Sprintf("%s | %s", message, appErr.Details)
		}
		return &AppError{
			Type:       errorType,
			Code:       code,
			Message:    appErr.Message,
			Details:    details,
			HTTPStatus: appErr.HTTPStatus,
			Retryable:  appErr.Retryable,
			Cause:      appErr,
			Timestamp:  timestamp(),
		}
	}

	// Wrap regular error
	return &AppError{
		Type:       errorType,
		Code:       code,
		Message:    message,
		Details:    err.Error(),
		HTTPStatus: http.StatusInternalServerError,
		Retryable:  false,
		Cause:      err,
		Timestamp:  timestamp(),
	}
}

// IsRetryable checks if the error is retryable
func IsRetryable(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Retryable
	}
	return false
}

// GetErrorType returns the error type
func GetErrorType(err error) ErrorType {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Type
	}
	return ErrorTypeProcessing // Default type
}

// GetErrorCode returns the error code
func GetErrorCode(err error) string {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code
	}
	return "unknown"
}

// timestamp returns current Unix timestamp
func timestamp() int64 {
	return time.Now().Unix()
}

// Common error codes
const (
	// Validation errors
	ErrCodeEmptyContent      = "empty_content"
	ErrCodeInvalidFormat     = "invalid_format"
	ErrCodeContentTooLarge   = "content_too_large"
	ErrCodeInvalidCharacters = "invalid_characters"

	// Clipboard errors
	ErrCodeClipboardEmpty    = "clipboard_empty"
	ErrCodeClipboardAccess   = "clipboard_access_denied"
	ErrCodeUnsupportedFormat = "unsupported_format"

	// API errors
	ErrCodeAPIConnection     = "api_connection_failed"
	ErrCodeAPITimeout        = "api_timeout"
	ErrCodeAPIResponse       = "api_response_error"
	ErrCodeAPIRateLimit      = "api_rate_limit"

	// File operation errors
	ErrCodeFileNotFound      = "file_not_found"
	ErrCodeFilePermission    = "file_permission_denied"
	ErrCodeFileCorrupted     = "file_corrupted"

	// Processing errors
	ErrCodeProcessingFailed  = "processing_failed"
	ErrCodeParsingFailed     = "parsing_failed"
	ErrCodeQualityLow        = "quality_too_low"

	// Configuration errors
	ErrCodeConfigMissing     = "config_missing"
	ErrCodeConfigInvalid     = "config_invalid"
	ErrCodeCredentialMissing = "credential_missing"
)