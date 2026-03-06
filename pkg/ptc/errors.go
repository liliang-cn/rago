package ptc

import (
	"errors"
	"fmt"
)

// Core errors
var (
	// ErrExecutionTimeout indicates the execution timed out
	ErrExecutionTimeout = errors.New("execution timeout")
	// ErrExecutionCancelled indicates the execution was cancelled
	ErrExecutionCancelled = errors.New("execution cancelled")
	// ErrMemoryLimitExceeded indicates memory limit was exceeded
	ErrMemoryLimitExceeded = errors.New("memory limit exceeded")
	// ErrCodeSizeExceeded indicates code size limit was exceeded
	ErrCodeSizeExceeded = errors.New("code size exceeded")
	// ErrOutputSizeExceeded indicates output size limit was exceeded
	ErrOutputSizeExceeded = errors.New("output size exceeded")
	// ErrMaxToolCallsExceeded indicates max tool calls was exceeded
	ErrMaxToolCallsExceeded = errors.New("maximum tool calls exceeded")

	// Sandbox errors
	// ErrSandboxNotReady indicates the sandbox is not ready
	ErrSandboxNotReady = errors.New("sandbox not ready")
	// ErrSandboxClosed indicates the sandbox is closed
	ErrSandboxClosed = errors.New("sandbox closed")
	// ErrCompilationFailed indicates code compilation failed
	ErrCompilationFailed = errors.New("compilation failed")
	// ErrRuntimeError indicates a runtime error occurred
	ErrRuntimeError = errors.New("runtime error")

	// Tool errors
	// ErrToolNotFound indicates the requested tool was not found
	ErrToolNotFound = errors.New("tool not found")
	// ErrToolNotAllowed indicates the tool is not allowed by security policy
	ErrToolNotAllowed = errors.New("tool not allowed")
	// ErrToolExecutionFailed indicates tool execution failed
	ErrToolExecutionFailed = errors.New("tool execution failed")

	// Security errors
	// ErrForbiddenPattern indicates forbidden code pattern detected
	ErrForbiddenPattern = errors.New("forbidden code pattern detected")
	// ErrSecurityViolation indicates a security violation
	ErrSecurityViolation = errors.New("security violation")

	// Configuration errors
	// ErrInvalidConfig indicates invalid configuration
	ErrInvalidConfig = errors.New("invalid configuration")
	// ErrRuntimeNotSupported indicates the runtime type is not supported
	ErrRuntimeNotSupported = errors.New("runtime not supported")

	// gRPC errors
	// ErrGRPCNotEnabled indicates gRPC is not enabled
	ErrGRPCNotEnabled = errors.New("grpc not enabled")
	// ErrGRPCConnection indicates gRPC connection error
	ErrGRPCConnection = errors.New("grpc connection error")
)

// ExecutionError wraps an error with execution context
type ExecutionError struct {
	Err      error
	ID       string
	Phase    string // "compile", "execute", "tool_call"
	ToolName string
	Line     int
	Column   int
	Source   string
}

// Error implements the error interface
func (e *ExecutionError) Error() string {
	if e.ToolName != "" {
		return fmt.Sprintf("execution error [%s] in tool '%s': %v", e.Phase, e.ToolName, e.Err)
	}
	if e.Line > 0 {
		return fmt.Sprintf("execution error [%s] at line %d: %v", e.Phase, e.Line, e.Err)
	}
	if e.Phase != "" {
		return fmt.Sprintf("execution error [%s]: %v", e.Phase, e.Err)
	}
	return e.Err.Error()
}

// Unwrap returns the underlying error
func (e *ExecutionError) Unwrap() error {
	return e.Err
}

// NewExecutionError creates a new execution error
func NewExecutionError(err error, phase string) *ExecutionError {
	return &ExecutionError{
		Err:   err,
		Phase: phase,
	}
}

// WithID adds execution ID to the error
func (e *ExecutionError) WithID(id string) *ExecutionError {
	e.ID = id
	return e
}

// WithTool adds tool name to the error
func (e *ExecutionError) WithTool(name string) *ExecutionError {
	e.ToolName = name
	return e
}

// WithLocation adds source location to the error
func (e *ExecutionError) WithLocation(line, column int) *ExecutionError {
	e.Line = line
	e.Column = column
	return e
}

// WithSource adds source code to the error
func (e *ExecutionError) WithSource(source string) *ExecutionError {
	e.Source = source
	return e
}

// IsTimeout checks if the error is a timeout error
func IsTimeout(err error) bool {
	return errors.Is(err, ErrExecutionTimeout)
}

// IsCancelled checks if the error is a cancellation error
func IsCancelled(err error) bool {
	return errors.Is(err, ErrExecutionCancelled)
}

// IsToolError checks if the error is related to tool execution
func IsToolError(err error) bool {
	return errors.Is(err, ErrToolNotFound) ||
		errors.Is(err, ErrToolNotAllowed) ||
		errors.Is(err, ErrToolExecutionFailed)
}

// IsSecurityError checks if the error is a security violation
func IsSecurityError(err error) bool {
	return errors.Is(err, ErrForbiddenPattern) ||
		errors.Is(err, ErrSecurityViolation)
}

// IsSandboxError checks if the error is related to sandbox
func IsSandboxError(err error) bool {
	return errors.Is(err, ErrSandboxNotReady) ||
		errors.Is(err, ErrSandboxClosed) ||
		errors.Is(err, ErrCompilationFailed) ||
		errors.Is(err, ErrRuntimeError)
}
