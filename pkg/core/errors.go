package core

import (
	"errors"
	"fmt"
)

// ===== COMMON ERRORS =====

// ErrNotFound is returned when a requested resource is not found
var ErrNotFound = errors.New("resource not found")

// ErrInvalidConfiguration is returned for configuration errors
var ErrInvalidConfiguration = errors.New("invalid configuration")

// ErrServiceUnavailable is returned when a service is unavailable
var ErrServiceUnavailable = errors.New("service unavailable")

// ErrTimeout is returned when an operation times out
var ErrTimeout = errors.New("operation timeout")

// ErrInvalidRequest is returned for malformed requests
var ErrInvalidRequest = errors.New("invalid request")

// ErrUnauthorized is returned for authorization failures
var ErrUnauthorized = errors.New("unauthorized")

// ErrQuotaExceeded is returned when limits are exceeded
var ErrQuotaExceeded = errors.New("quota exceeded")

// ErrInternal is returned for internal system errors
var ErrInternal = errors.New("internal error")

// ===== LLM PILLAR ERRORS =====

// ErrProviderNotFound is returned when a provider is not found
var ErrProviderNotFound = errors.New("provider not found")

// ErrProviderUnhealthy is returned when a provider is unhealthy
var ErrProviderUnhealthy = errors.New("provider unhealthy")

// ErrModelNotSupported is returned when a model is not supported
var ErrModelNotSupported = errors.New("model not supported")

// ErrGenerationFailed is returned when generation fails
var ErrGenerationFailed = errors.New("generation failed")

// ErrStreamingNotSupported is returned when streaming is not supported
var ErrStreamingNotSupported = errors.New("streaming not supported")

// ErrToolCallFailed is returned when tool calling fails
var ErrToolCallFailed = errors.New("tool call failed")

// ErrNoProvidersAvailable is returned when no providers are available
var ErrNoProvidersAvailable = errors.New("no providers available")

// ===== RAG PILLAR ERRORS =====

// ErrDocumentNotFound is returned when a document is not found
var ErrDocumentNotFound = errors.New("document not found")

// ErrInvalidDocument is returned for invalid documents
var ErrInvalidDocument = errors.New("invalid document")

// ErrIngestFailed is returned when document ingestion fails
var ErrIngestFailed = errors.New("document ingest failed")

// ErrSearchFailed is returned when search operations fail
var ErrSearchFailed = errors.New("search failed")

// ErrChunkingFailed is returned when document chunking fails
var ErrChunkingFailed = errors.New("document chunking failed")

// ErrEmbeddingFailed is returned when embedding generation fails
var ErrEmbeddingFailed = errors.New("embedding generation failed")

// ErrStorageFailed is returned when storage operations fail
var ErrStorageFailed = errors.New("storage operation failed")

// ErrIndexCorrupted is returned when index is corrupted
var ErrIndexCorrupted = errors.New("index corrupted")

// ===== MCP PILLAR ERRORS =====

// ErrServerNotFound is returned when an MCP server is not found
var ErrServerNotFound = errors.New("server not found")

// ErrServerUnhealthy is returned when an MCP server is unhealthy
var ErrServerUnhealthy = errors.New("server unhealthy")

// ErrToolNotFound is returned when a tool is not found
var ErrToolNotFound = errors.New("tool not found")

// ErrToolExecutionFailed is returned when tool execution fails
var ErrToolExecutionFailed = errors.New("tool execution failed")

// ErrToolTimeout is returned when tool execution times out
var ErrToolTimeout = errors.New("tool execution timeout")

// ErrServerRegistrationFailed is returned when server registration fails
var ErrServerRegistrationFailed = errors.New("server registration failed")

// ErrProtocolViolation is returned for MCP protocol violations
var ErrProtocolViolation = errors.New("MCP protocol violation")

// ===== AGENT PILLAR ERRORS =====

// ErrWorkflowNotFound is returned when a workflow is not found
var ErrWorkflowNotFound = errors.New("workflow not found")

// ErrAgentNotFound is returned when an agent is not found
var ErrAgentNotFound = errors.New("agent not found")

// ErrWorkflowFailed is returned when workflow execution fails
var ErrWorkflowFailed = errors.New("workflow execution failed")

// ErrAgentFailed is returned when agent execution fails
var ErrAgentFailed = errors.New("agent execution failed")

// ErrInvalidWorkflow is returned for invalid workflow definitions
var ErrInvalidWorkflow = errors.New("invalid workflow definition")

// ErrInvalidAgent is returned for invalid agent definitions
var ErrInvalidAgent = errors.New("invalid agent definition")

// ErrWorkflowTimeout is returned when workflow execution times out
var ErrWorkflowTimeout = errors.New("workflow execution timeout")

// ErrStepFailed is returned when a workflow step fails
var ErrStepFailed = errors.New("workflow step failed")

// ErrSchedulingFailed is returned when scheduling fails
var ErrSchedulingFailed = errors.New("scheduling failed")

// ErrReasoningFailed is returned when reasoning chains fail
var ErrReasoningFailed = errors.New("reasoning chain failed")

// ===== ERROR TYPES =====

// ConfigurationError represents configuration-related errors
type ConfigurationError struct {
	Component string
	Field     string
	Message   string
	Cause     error
}

func (e *ConfigurationError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("configuration error in %s.%s: %s (caused by: %v)", 
			e.Component, e.Field, e.Message, e.Cause)
	}
	return fmt.Sprintf("configuration error in %s.%s: %s", 
		e.Component, e.Field, e.Message)
}

func (e *ConfigurationError) Unwrap() error {
	return e.Cause
}

// ValidationError represents validation errors
type ValidationError struct {
	Field   string
	Value   interface{}
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for field '%s': %s (value: %v)", 
		e.Field, e.Message, e.Value)
}

// ServiceError represents service-level errors with context
type ServiceError struct {
	Service   string
	Operation string
	Message   string
	Cause     error
	Context   map[string]interface{}
}

func (e *ServiceError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s service error in %s: %s (caused by: %v)", 
			e.Service, e.Operation, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s service error in %s: %s", 
		e.Service, e.Operation, e.Message)
}

func (e *ServiceError) Unwrap() error {
	return e.Cause
}

// NetworkError represents network-related errors
type NetworkError struct {
	Host      string
	Operation string
	Message   string
	Cause     error
}

func (e *NetworkError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("network error connecting to %s during %s: %s (caused by: %v)", 
			e.Host, e.Operation, e.Message, e.Cause)
	}
	return fmt.Sprintf("network error connecting to %s during %s: %s", 
		e.Host, e.Operation, e.Message)
}

func (e *NetworkError) Unwrap() error {
	return e.Cause
}

// TimeoutError represents timeout errors with context
type TimeoutError struct {
	Operation string
	Duration  string
	Message   string
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("timeout error in %s after %s: %s", 
		e.Operation, e.Duration, e.Message)
}

// ===== ERROR CONSTRUCTORS =====

// NewConfigurationError creates a new configuration error
func NewConfigurationError(component, field, message string, cause error) *ConfigurationError {
	return &ConfigurationError{
		Component: component,
		Field:     field,
		Message:   message,
		Cause:     cause,
	}
}

// NewValidationError creates a new validation error
func NewValidationError(field string, value interface{}, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}

// NewServiceError creates a new service error
func NewServiceError(service, operation, message string, cause error) *ServiceError {
	return &ServiceError{
		Service:   service,
		Operation: operation,
		Message:   message,
		Cause:     cause,
		Context:   make(map[string]interface{}),
	}
}

// WithContext adds context to a service error
func (e *ServiceError) WithContext(key string, value interface{}) *ServiceError {
	e.Context[key] = value
	return e
}

// NewNetworkError creates a new network error
func NewNetworkError(host, operation, message string, cause error) *NetworkError {
	return &NetworkError{
		Host:      host,
		Operation: operation,
		Message:   message,
		Cause:     cause,
	}
}

// NewTimeoutError creates a new timeout error
func NewTimeoutError(operation, duration, message string) *TimeoutError {
	return &TimeoutError{
		Operation: operation,
		Duration:  duration,
		Message:   message,
	}
}

// ===== ERROR CHECKING UTILITIES =====

// IsNotFoundError checks if an error is a "not found" error
func IsNotFoundError(err error) bool {
	return errors.Is(err, ErrNotFound) ||
		errors.Is(err, ErrProviderNotFound) ||
		errors.Is(err, ErrDocumentNotFound) ||
		errors.Is(err, ErrServerNotFound) ||
		errors.Is(err, ErrToolNotFound) ||
		errors.Is(err, ErrWorkflowNotFound) ||
		errors.Is(err, ErrAgentNotFound)
}

// IsTimeoutError checks if an error is a timeout error
func IsTimeoutError(err error) bool {
	return errors.Is(err, ErrTimeout) ||
		errors.Is(err, ErrToolTimeout) ||
		errors.Is(err, ErrWorkflowTimeout)
}

// IsServiceUnavailableError checks if an error indicates service unavailability
func IsServiceUnavailableError(err error) bool {
	return errors.Is(err, ErrServiceUnavailable) ||
		errors.Is(err, ErrProviderUnhealthy) ||
		errors.Is(err, ErrServerUnhealthy) ||
		errors.Is(err, ErrNoProvidersAvailable)
}

// IsValidationError checks if an error is a validation error
func IsValidationError(err error) bool {
	var validationErr *ValidationError
	return errors.As(err, &validationErr)
}

// IsConfigurationError checks if an error is a configuration error
func IsConfigurationError(err error) bool {
	var configErr *ConfigurationError
	return errors.As(err, &configErr) || errors.Is(err, ErrInvalidConfiguration)
}

// IsNetworkError checks if an error is a network error
func IsNetworkError(err error) bool {
	var networkErr *NetworkError
	return errors.As(err, &networkErr)
}

// ===== ERROR WRAPPING UTILITIES =====

// WrapError wraps an error with additional context
func WrapError(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// WrapErrorWithContext wraps an error with a service context
func WrapErrorWithContext(err error, service, operation, message string) error {
	if err == nil {
		return nil
	}
	return &ServiceError{
		Service:   service,
		Operation: operation,
		Message:   message,
		Cause:     err,
		Context:   make(map[string]interface{}),
	}
}