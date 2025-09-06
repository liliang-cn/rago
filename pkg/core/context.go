package core

import (
	"context"
	"time"
)

// ===== CONTEXT KEYS =====

type contextKey string

const (
	// RequestIDKey is the context key for request IDs
	RequestIDKey contextKey = "request_id"
	
	// UserIDKey is the context key for user IDs
	UserIDKey contextKey = "user_id"
	
	// SessionIDKey is the context key for session IDs
	SessionIDKey contextKey = "session_id"
	
	// TraceIDKey is the context key for trace IDs
	TraceIDKey contextKey = "trace_id"
	
	// TenantIDKey is the context key for tenant/organization IDs
	TenantIDKey contextKey = "tenant_id"
	
	// PillarKey is the context key for the current pillar
	PillarKey contextKey = "pillar"
	
	// OperationKey is the context key for the current operation
	OperationKey contextKey = "operation"
	
	// MetadataKey is the context key for additional metadata
	MetadataKey contextKey = "metadata"
)

// ===== CONTEXT UTILITIES =====

// WithRequestID adds a request ID to the context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// GetRequestID retrieves the request ID from the context
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

// WithUserID adds a user ID to the context
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// GetUserID retrieves the user ID from the context
func GetUserID(ctx context.Context) string {
	if id, ok := ctx.Value(UserIDKey).(string); ok {
		return id
	}
	return ""
}

// WithSessionID adds a session ID to the context
func WithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, SessionIDKey, sessionID)
}

// GetSessionID retrieves the session ID from the context
func GetSessionID(ctx context.Context) string {
	if id, ok := ctx.Value(SessionIDKey).(string); ok {
		return id
	}
	return ""
}

// WithTraceID adds a trace ID to the context
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// GetTraceID retrieves the trace ID from the context
func GetTraceID(ctx context.Context) string {
	if id, ok := ctx.Value(TraceIDKey).(string); ok {
		return id
	}
	return ""
}

// WithTenantID adds a tenant ID to the context
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, TenantIDKey, tenantID)
}

// GetTenantID retrieves the tenant ID from the context
func GetTenantID(ctx context.Context) string {
	if id, ok := ctx.Value(TenantIDKey).(string); ok {
		return id
	}
	return ""
}

// WithPillar adds the current pillar to the context
func WithPillar(ctx context.Context, pillar string) context.Context {
	return context.WithValue(ctx, PillarKey, pillar)
}

// GetPillar retrieves the current pillar from the context
func GetPillar(ctx context.Context) string {
	if pillar, ok := ctx.Value(PillarKey).(string); ok {
		return pillar
	}
	return ""
}

// WithOperation adds the current operation to the context
func WithOperation(ctx context.Context, operation string) context.Context {
	return context.WithValue(ctx, OperationKey, operation)
}

// GetOperation retrieves the current operation from the context
func GetOperation(ctx context.Context) string {
	if op, ok := ctx.Value(OperationKey).(string); ok {
		return op
	}
	return ""
}

// WithMetadata adds metadata to the context
func WithMetadata(ctx context.Context, metadata map[string]interface{}) context.Context {
	return context.WithValue(ctx, MetadataKey, metadata)
}

// GetMetadata retrieves metadata from the context
func GetMetadata(ctx context.Context) map[string]interface{} {
	if metadata, ok := ctx.Value(MetadataKey).(map[string]interface{}); ok {
		return metadata
	}
	return nil
}

// AddToMetadata adds a key-value pair to the context metadata
func AddToMetadata(ctx context.Context, key string, value interface{}) context.Context {
	metadata := GetMetadata(ctx)
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	
	// Create a copy to avoid modifying the original
	newMetadata := make(map[string]interface{})
	for k, v := range metadata {
		newMetadata[k] = v
	}
	newMetadata[key] = value
	
	return WithMetadata(ctx, newMetadata)
}

// ===== TIMEOUT UTILITIES =====

// WithTimeout creates a context with a timeout
func WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, timeout)
}

// WithDeadline creates a context with a deadline
func WithDeadline(ctx context.Context, deadline time.Time) (context.Context, context.CancelFunc) {
	return context.WithDeadline(ctx, deadline)
}

// WithCancel creates a context with cancellation
func WithCancel(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithCancel(ctx)
}

// ===== CONTEXT BUILDERS =====

// NewRequestContext creates a new context for a request with common fields
func NewRequestContext(parent context.Context, requestID, userID, sessionID string) context.Context {
	ctx := parent
	if ctx == nil {
		ctx = context.Background()
	}
	
	if requestID != "" {
		ctx = WithRequestID(ctx, requestID)
	}
	if userID != "" {
		ctx = WithUserID(ctx, userID)
	}
	if sessionID != "" {
		ctx = WithSessionID(ctx, sessionID)
	}
	
	return ctx
}

// NewOperationContext creates a context for a pillar operation
func NewOperationContext(parent context.Context, pillar, operation string) context.Context {
	ctx := parent
	if ctx == nil {
		ctx = context.Background()
	}
	
	ctx = WithPillar(ctx, pillar)
	ctx = WithOperation(ctx, operation)
	
	return ctx
}

// NewOperationContextWithTimeout creates a context for a pillar operation with timeout
func NewOperationContextWithTimeout(parent context.Context, pillar, operation string, timeout time.Duration) (context.Context, context.CancelFunc) {
	ctx := NewOperationContext(parent, pillar, operation)
	return WithTimeout(ctx, timeout)
}

// ===== CONTEXT VALIDATION =====

// HasRequiredContext checks if the context has required fields for operations
func HasRequiredContext(ctx context.Context, requireRequestID, requireUserID bool) error {
	if requireRequestID && GetRequestID(ctx) == "" {
		return NewValidationError("request_id", "", "request ID is required")
	}
	
	if requireUserID && GetUserID(ctx) == "" {
		return NewValidationError("user_id", "", "user ID is required")
	}
	
	return nil
}

// IsContextExpired checks if the context is expired or cancelled
func IsContextExpired(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

// GetContextError returns the context error if any
func GetContextError(ctx context.Context) error {
	return ctx.Err()
}

// ===== LOGGING HELPERS =====

// ExtractLoggingFields extracts common fields from context for structured logging
func ExtractLoggingFields(ctx context.Context) map[string]interface{} {
	fields := make(map[string]interface{})
	
	if requestID := GetRequestID(ctx); requestID != "" {
		fields["request_id"] = requestID
	}
	
	if userID := GetUserID(ctx); userID != "" {
		fields["user_id"] = userID
	}
	
	if sessionID := GetSessionID(ctx); sessionID != "" {
		fields["session_id"] = sessionID
	}
	
	if traceID := GetTraceID(ctx); traceID != "" {
		fields["trace_id"] = traceID
	}
	
	if tenantID := GetTenantID(ctx); tenantID != "" {
		fields["tenant_id"] = tenantID
	}
	
	if pillar := GetPillar(ctx); pillar != "" {
		fields["pillar"] = pillar
	}
	
	if operation := GetOperation(ctx); operation != "" {
		fields["operation"] = operation
	}
	
	if metadata := GetMetadata(ctx); metadata != nil {
		fields["metadata"] = metadata
	}
	
	return fields
}

// ===== PILLAR CONSTANTS =====

const (
	// PillarLLM identifies the LLM pillar
	PillarLLM = "llm"
	
	// PillarRAG identifies the RAG pillar
	PillarRAG = "rag"
	
	// PillarMCP identifies the MCP pillar
	PillarMCP = "mcp"
	
	// PillarAgent identifies the Agent pillar
	PillarAgent = "agent"
	
	// PillarClient identifies unified client operations
	PillarClient = "client"
)