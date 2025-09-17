package errors

import (
	"errors"
	"fmt"
)

var (
	ErrNoAPIKey           = errors.New("no API key provided")
	ErrInvalidModel       = errors.New("invalid model specified")
	ErrMaxTurnsExceeded   = errors.New("maximum conversation turns exceeded")
	ErrTimeout            = errors.New("operation timed out")
	ErrPermissionDenied   = errors.New("permission denied")
	ErrToolNotFound       = errors.New("tool not found")
	ErrToolExecutionFailed = errors.New("tool execution failed")
	ErrHookExecutionFailed = errors.New("hook execution failed")
	ErrStreamingFailed    = errors.New("streaming failed")
	ErrSessionNotFound    = errors.New("session not found")
	ErrInvalidMessage     = errors.New("invalid message format")
	ErrConnectionFailed   = errors.New("connection to API failed")
	ErrRateLimited        = errors.New("API rate limit exceeded")
	ErrInvalidArguments   = errors.New("invalid arguments provided")
	ErrContextCanceled    = errors.New("context was canceled")
)

type ClaudeSDKError struct {
	Code    string
	Message string
	Details map[string]any
	Cause   error
}

func (e *ClaudeSDKError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *ClaudeSDKError) Unwrap() error {
	return e.Cause
}

func NewSDKError(code, message string, cause error) *ClaudeSDKError {
	return &ClaudeSDKError{
		Code:    code,
		Message: message,
		Cause:   cause,
		Details: make(map[string]any),
	}
}

func (e *ClaudeSDKError) WithDetail(key string, value any) *ClaudeSDKError {
	e.Details[key] = value
	return e
}

type ToolError struct {
	ToolName string
	Message  string
	Cause    error
}

func (e *ToolError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("tool '%s' failed: %s (caused by: %v)", e.ToolName, e.Message, e.Cause)
	}
	return fmt.Sprintf("tool '%s' failed: %s", e.ToolName, e.Message)
}

func (e *ToolError) Unwrap() error {
	return e.Cause
}

func NewToolError(toolName, message string, cause error) *ToolError {
	return &ToolError{
		ToolName: toolName,
		Message:  message,
		Cause:    cause,
	}
}

type PermissionError struct {
	Tool      string
	Operation string
	Path      string
	Reason    string
}

func (e *PermissionError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("permission denied: %s operation on '%s' for tool '%s': %s",
			e.Operation, e.Path, e.Tool, e.Reason)
	}
	return fmt.Sprintf("permission denied: %s operation for tool '%s': %s",
		e.Operation, e.Tool, e.Reason)
}

func NewPermissionError(tool, operation, path, reason string) *PermissionError {
	return &PermissionError{
		Tool:      tool,
		Operation: operation,
		Path:      path,
		Reason:    reason,
	}
}

type ValidationError struct {
	Field   string
	Value   any
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field '%s': %s (value: %v)", e.Field, e.Message, e.Value)
}

func NewValidationError(field string, value any, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}

func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	var sdkErr *ClaudeSDKError
	if errors.As(err, &sdkErr) {
		switch sdkErr.Code {
		case "rate_limited", "timeout", "connection_failed":
			return true
		}
	}

	return errors.Is(err, ErrRateLimited) ||
		   errors.Is(err, ErrTimeout) ||
		   errors.Is(err, ErrConnectionFailed)
}