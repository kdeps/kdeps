package logging

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"
)

// ContextKey represents keys for storing values in context
type ContextKey string

const (
	RequestIDKey  ContextKey = "request_id"
	ActionIDKey   ContextKey = "action_id"
	ResourceKey   ContextKey = "resource_type"
	WorkflowKey   ContextKey = "workflow_id"
	AgentKey      ContextKey = "agent_id"
	TraceIDKey    ContextKey = "trace_id"
)

// EnhancedLogger provides structured logging with context and performance tracking
type EnhancedLogger struct {
	base   *Logger
	ctx    context.Context
	fields map[string]interface{}
}

// NewEnhancedLogger creates a new enhanced logger with context
func NewEnhancedLogger(base *Logger, ctx context.Context) *EnhancedLogger {
	return &EnhancedLogger{
		base:   base,
		ctx:    ctx,
		fields: make(map[string]interface{}),
	}
}

// WithContext creates a new logger with additional context
func (el *EnhancedLogger) WithContext(ctx context.Context) *EnhancedLogger {
	return &EnhancedLogger{
		base:   el.base,
		ctx:    ctx,
		fields: el.copyFields(),
	}
}

// WithField adds a field to the logger context
func (el *EnhancedLogger) WithField(key string, value interface{}) *EnhancedLogger {
	newLogger := &EnhancedLogger{
		base:   el.base,
		ctx:    el.ctx,
		fields: el.copyFields(),
	}
	newLogger.fields[key] = value
	return newLogger
}

// WithFields adds multiple fields to the logger context
func (el *EnhancedLogger) WithFields(fields map[string]interface{}) *EnhancedLogger {
	newLogger := &EnhancedLogger{
		base:   el.base,
		ctx:    el.ctx,
		fields: el.copyFields(),
	}
	for k, v := range fields {
		newLogger.fields[k] = v
	}
	return newLogger
}

// WithRequestID adds request ID to the logger context
func (el *EnhancedLogger) WithRequestID(requestID string) *EnhancedLogger {
	return el.WithField("request_id", requestID)
}

// WithActionID adds action ID to the logger context
func (el *EnhancedLogger) WithActionID(actionID string) *EnhancedLogger {
	return el.WithField("action_id", actionID)
}

// WithResourceType adds resource type to the logger context
func (el *EnhancedLogger) WithResourceType(resourceType string) *EnhancedLogger {
	return el.WithField("resource_type", resourceType)
}

// TimeOperation logs the duration of an operation
func (el *EnhancedLogger) TimeOperation(operation string, fn func() error) error {
	start := time.Now()
	el.Debug("starting operation", "operation", operation)
	
	err := fn()
	duration := time.Since(start)
	
	if err != nil {
		el.Error("operation failed", 
			"operation", operation,
			"duration", duration,
			"error", err)
	} else {
		el.Info("operation completed",
			"operation", operation,
			"duration", duration)
	}
	
	return err
}

// TimeOperationWithResult logs the duration of an operation that returns a result
func (el *EnhancedLogger) TimeOperationWithResult(operation string, fn func() (interface{}, error)) (interface{}, error) {
	start := time.Now()
	el.Debug("starting operation", "operation", operation)
	
	result, err := fn()
	duration := time.Since(start)
	
	if err != nil {
		el.Error("operation failed", 
			"operation", operation,
			"duration", duration,
			"error", err)
	} else {
		el.Info("operation completed",
			"operation", operation,
			"duration", duration)
	}
	
	return result, err
}

// Trace logs at trace level with context
func (el *EnhancedLogger) Trace(msg string, args ...interface{}) {
	el.logWithContext("TRACE", msg, args...)
}

// Debug logs at debug level with context
func (el *EnhancedLogger) Debug(msg string, args ...interface{}) {
	el.logWithContext("DEBUG", msg, args...)
}

// Info logs at info level with context
func (el *EnhancedLogger) Info(msg string, args ...interface{}) {
	el.logWithContext("INFO", msg, args...)
}

// Warn logs at warn level with context
func (el *EnhancedLogger) Warn(msg string, args ...interface{}) {
	el.logWithContext("WARN", msg, args...)
}

// Error logs at error level with context and stack trace
func (el *EnhancedLogger) Error(msg string, args ...interface{}) {
	// Add stack trace for errors
	if _, file, line, ok := runtime.Caller(1); ok {
		args = append(args, "caller", fmt.Sprintf("%s:%d", file, line))
	}
	el.logWithContext("ERROR", msg, args...)
}

// Fatal logs at fatal level with context and exits
func (el *EnhancedLogger) Fatal(msg string, args ...interface{}) {
	// Add stack trace for fatal errors
	if _, file, line, ok := runtime.Caller(1); ok {
		args = append(args, "caller", fmt.Sprintf("%s:%d", file, line))
	}
	el.logWithContext("FATAL", msg, args...)
	// Note: In a real implementation, you might call os.Exit(1) here
}

// logWithContext logs a message with full context information
func (el *EnhancedLogger) logWithContext(level string, msg string, args ...interface{}) {
	// Combine context fields, logger fields, and provided args
	allArgs := make([]interface{}, 0, len(el.fields)*2+len(args))
	
	// Add context-derived fields
	if el.ctx != nil {
		if requestID := el.ctx.Value(RequestIDKey); requestID != nil {
			allArgs = append(allArgs, "request_id", requestID)
		}
		if actionID := el.ctx.Value(ActionIDKey); actionID != nil {
			allArgs = append(allArgs, "action_id", actionID)
		}
		if resourceType := el.ctx.Value(ResourceKey); resourceType != nil {
			allArgs = append(allArgs, "resource_type", resourceType)
		}
		if workflowID := el.ctx.Value(WorkflowKey); workflowID != nil {
			allArgs = append(allArgs, "workflow_id", workflowID)
		}
		if agentID := el.ctx.Value(AgentKey); agentID != nil {
			allArgs = append(allArgs, "agent_id", agentID)
		}
		if traceID := el.ctx.Value(TraceIDKey); traceID != nil {
			allArgs = append(allArgs, "trace_id", traceID)
		}
	}
	
	// Add logger-specific fields
	for k, v := range el.fields {
		allArgs = append(allArgs, k, v)
	}
	
	// Add provided args
	allArgs = append(allArgs, args...)
	
	// Log using the base logger
	switch level {
	case "TRACE":
		if el.base != nil {
			el.base.Debug(msg, allArgs...)
		}
	case "DEBUG":
		if el.base != nil {
			el.base.Debug(msg, allArgs...)
		}
	case "INFO":
		if el.base != nil {
			el.base.Info(msg, allArgs...)
		}
	case "WARN":
		if el.base != nil {
			el.base.Warn(msg, allArgs...)
		}
	case "ERROR":
		if el.base != nil {
			el.base.Error(msg, allArgs...)
		}
	case "FATAL":
		if el.base != nil {
			el.base.Error(msg, allArgs...)
		}
	}
}

// copyFields creates a copy of the current fields map
func (el *EnhancedLogger) copyFields() map[string]interface{} {
	fields := make(map[string]interface{}, len(el.fields))
	for k, v := range el.fields {
		fields[k] = v
	}
	return fields
}

// LogResourceProcessing provides structured logging for resource processing
func (el *EnhancedLogger) LogResourceProcessing(actionID, resourceType string, fn func(*EnhancedLogger) error) error {
	resourceLogger := el.WithActionID(actionID).WithResourceType(resourceType)
	
	return resourceLogger.TimeOperation(
		fmt.Sprintf("process_%s_resource", strings.ToLower(resourceType)),
		func() error {
			return fn(resourceLogger)
		},
	)
}

// LogDependencyResolution provides structured logging for dependency resolution
func (el *EnhancedLogger) LogDependencyResolution(workflowID string, dependencies []string, fn func(*EnhancedLogger) error) error {
	depLogger := el.WithField("workflow_id", workflowID).
		WithField("dependency_count", len(dependencies)).
		WithField("dependencies", dependencies)
	
	return depLogger.TimeOperation("resolve_dependencies", func() error {
		return fn(depLogger)
	})
}

// LogHTTPRequest provides structured logging for HTTP requests
func (el *EnhancedLogger) LogHTTPRequest(method, url string, statusCode int, duration time.Duration, err error) {
	fields := map[string]interface{}{
		"http_method":     method,
		"http_url":        url,
		"http_status":     statusCode,
		"http_duration":   duration,
	}
	
	logger := el.WithFields(fields)
	
	if err != nil {
		logger.Error("HTTP request failed", "error", err)
	} else if statusCode >= 400 {
		logger.Warn("HTTP request completed with error status")
	} else {
		logger.Info("HTTP request completed successfully")
	}
}

// LogLLMGeneration provides structured logging for LLM generation
func (el *EnhancedLogger) LogLLMGeneration(model, prompt string, responseLength int, duration time.Duration, err error) {
	fields := map[string]interface{}{
		"llm_model":           model,
		"llm_prompt_length":   len(prompt),
		"llm_response_length": responseLength,
		"llm_duration":        duration,
	}
	
	logger := el.WithFields(fields)
	
	if err != nil {
		logger.Error("LLM generation failed", "error", err)
	} else {
		logger.Info("LLM generation completed successfully")
	}
}

// LogCommandExecution provides structured logging for command execution
func (el *EnhancedLogger) LogCommandExecution(command string, exitCode int, duration time.Duration, err error) {
	fields := map[string]interface{}{
		"command":      command,
		"exit_code":    exitCode,
		"exec_duration": duration,
	}
	
	logger := el.WithFields(fields)
	
	if err != nil {
		logger.Error("command execution failed", "error", err)
	} else if exitCode != 0 {
		logger.Warn("command execution completed with non-zero exit code")
	} else {
		logger.Info("command execution completed successfully")
	}
}

// MemoryUsage logs current memory usage statistics
func (el *EnhancedLogger) MemoryUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	el.Debug("memory usage statistics",
		"alloc_mb", m.Alloc/1024/1024,
		"total_alloc_mb", m.TotalAlloc/1024/1024,
		"sys_mb", m.Sys/1024/1024,
		"num_gc", m.NumGC,
	)
}

// GoroutineCount logs the current number of goroutines
func (el *EnhancedLogger) GoroutineCount() {
	el.Debug("goroutine count", "count", runtime.NumGoroutine())
}