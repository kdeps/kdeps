package errors

import (
	"fmt"
	"time"
)

// ErrorCode represents specific error types in the kdeps system
type ErrorCode string

const (
	// Resource handling errors
	ErrResourceNotFound   ErrorCode = "RESOURCE_NOT_FOUND"
	ErrResourceReload     ErrorCode = "RESOURCE_RELOAD_FAILED"
	ErrResourceProcessing ErrorCode = "RESOURCE_PROCESSING_FAILED"
	ErrResourceStorage    ErrorCode = "RESOURCE_STORAGE_FAILED"

	// PKL errors
	ErrPKLTemplateEvaluation ErrorCode = "PKL_TEMPLATE_EVALUATION_FAILED"
	ErrPKLResourceLoading    ErrorCode = "PKL_RESOURCE_LOADING_FAILED"

	// Network errors
	ErrHTTPRequest ErrorCode = "HTTP_REQUEST_FAILED"
	ErrHTTPTimeout ErrorCode = "HTTP_TIMEOUT"

	// Execution errors
	ErrCommandExecution ErrorCode = "COMMAND_EXECUTION_FAILED"
	ErrPythonExecution  ErrorCode = "PYTHON_EXECUTION_FAILED"

	// LLM errors
	ErrLLMGeneration ErrorCode = "LLM_GENERATION_FAILED"
	ErrLLMTimeout    ErrorCode = "LLM_TIMEOUT"

	// Dependency errors
	ErrDependencyResolution ErrorCode = "DEPENDENCY_RESOLUTION_FAILED"
	ErrCircularDependency   ErrorCode = "CIRCULAR_DEPENDENCY_DETECTED"

	// Storage errors
	ErrPklresAccess   ErrorCode = "PKLRES_ACCESS_FAILED"
	ErrFileOperations ErrorCode = "FILE_OPERATIONS_FAILED"
)

// ResourceError represents a structured error in the kdeps system
type ResourceError struct {
	Code         ErrorCode              `json:"code"`
	Message      string                 `json:"message"`
	ActionID     string                 `json:"action_id,omitempty"`
	ResourceType string                 `json:"resource_type,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
	Cause        error                  `json:"-"`
	Context      map[string]interface{} `json:"context,omitempty"`
}

// Error implements the error interface
func (re *ResourceError) Error() string {
	if re.ActionID != "" && re.ResourceType != "" {
		return fmt.Sprintf("[%s] %s (actionID: %s, resourceType: %s): %s",
			re.Code, re.ResourceType, re.ActionID, re.ResourceType, re.Message)
	}
	return fmt.Sprintf("[%s]: %s", re.Code, re.Message)
}

// Unwrap returns the underlying cause error
func (re *ResourceError) Unwrap() error {
	return re.Cause
}

// NewResourceError creates a new structured resource error
func NewResourceError(code ErrorCode, message string) *ResourceError {
	return &ResourceError{
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
		Context:   make(map[string]interface{}),
	}
}

// WithActionID adds action ID context to the error
func (re *ResourceError) WithActionID(actionID string) *ResourceError {
	re.ActionID = actionID
	return re
}

// WithResourceType adds resource type context to the error
func (re *ResourceError) WithResourceType(resourceType string) *ResourceError {
	re.ResourceType = resourceType
	return re
}

// WithCause adds the underlying cause error
func (re *ResourceError) WithCause(err error) *ResourceError {
	re.Cause = err
	return re
}

// WithContext adds arbitrary context to the error
func (re *ResourceError) WithContext(key string, value interface{}) *ResourceError {
	re.Context[key] = value
	return re
}

// IsResourceError checks if an error is a ResourceError
func IsResourceError(err error) (*ResourceError, bool) {
	if re, ok := err.(*ResourceError); ok {
		return re, true
	}
	return nil, false
}

// HasErrorCode checks if an error has a specific error code
func HasErrorCode(err error, code ErrorCode) bool {
	if re, ok := IsResourceError(err); ok {
		return re.Code == code
	}
	return false
}

// WrapError wraps a regular error as a ResourceError
func WrapError(err error, code ErrorCode, message string) *ResourceError {
	return NewResourceError(code, message).WithCause(err)
}

// Common error constructors for frequently used errors

func NewResourceNotFoundError(actionID, resourceType string) *ResourceError {
	return NewResourceError(ErrResourceNotFound, "resource not found").
		WithActionID(actionID).
		WithResourceType(resourceType)
}

func NewResourceReloadError(actionID, resourceType string, cause error) *ResourceError {
	return NewResourceError(ErrResourceReload, "failed to reload resource").
		WithActionID(actionID).
		WithResourceType(resourceType).
		WithCause(cause)
}

func NewResourceProcessingError(actionID, resourceType string, cause error) *ResourceError {
	return NewResourceError(ErrResourceProcessing, "failed to process resource").
		WithActionID(actionID).
		WithResourceType(resourceType).
		WithCause(cause)
}

func NewHTTPRequestError(actionID, url string, cause error) *ResourceError {
	return NewResourceError(ErrHTTPRequest, "HTTP request failed").
		WithActionID(actionID).
		WithResourceType("HTTP").
		WithContext("url", url).
		WithCause(cause)
}

func NewLLMGenerationError(actionID, model string, cause error) *ResourceError {
	return NewResourceError(ErrLLMGeneration, "LLM generation failed").
		WithActionID(actionID).
		WithResourceType("LLM").
		WithContext("model", model).
		WithCause(cause)
}

func NewCommandExecutionError(actionID, command string, exitCode int, cause error) *ResourceError {
	return NewResourceError(ErrCommandExecution, "command execution failed").
		WithActionID(actionID).
		WithResourceType("Exec").
		WithContext("command", command).
		WithContext("exit_code", exitCode).
		WithCause(cause)
}

func NewPythonExecutionError(actionID string, exitCode int, cause error) *ResourceError {
	return NewResourceError(ErrPythonExecution, "Python execution failed").
		WithActionID(actionID).
		WithResourceType("Python").
		WithContext("exit_code", exitCode).
		WithCause(cause)
}
