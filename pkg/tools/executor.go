package tools

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/time/rate"
)

// Executor manages the execution of tools with rate limiting, concurrency control, and logging
type Executor struct {
	registry   *Registry
	limiter    *rate.Limiter
	semaphore  chan struct{}
	logger     Logger
	mu         sync.RWMutex
	executions map[string]*ExecutionInfo
	config     *ExecutorConfig
}

// ExecutorConfig contains configuration for the executor
type ExecutorConfig struct {
	MaxConcurrency int           `json:"max_concurrency"`
	DefaultTimeout time.Duration `json:"default_timeout"`
	EnableLogging  bool          `json:"enable_logging"`
	LogExecution   bool          `json:"log_execution"`
	RetryAttempts  int           `json:"retry_attempts"`
	RetryDelay     time.Duration `json:"retry_delay"`
}

// ExecutionInfo tracks information about a tool execution
type ExecutionInfo struct {
	ID        string                 `json:"id"`
	ToolName  string                 `json:"tool_name"`
	StartTime time.Time              `json:"start_time"`
	EndTime   *time.Time             `json:"end_time,omitempty"`
	Status    ExecutionStatus        `json:"status"`
	Context   *ExecutionContext      `json:"context"`
	Arguments map[string]interface{} `json:"arguments"`
	Result    *ToolResult            `json:"result,omitempty"`
	Error     error                  `json:"error,omitempty"`
	Attempts  int                    `json:"attempts"`
}

// ExecutionStatus represents the status of a tool execution
type ExecutionStatus string

const (
	StatusPending   ExecutionStatus = "pending"
	StatusRunning   ExecutionStatus = "running"
	StatusCompleted ExecutionStatus = "completed"
	StatusFailed    ExecutionStatus = "failed"
	StatusTimeout   ExecutionStatus = "timeout"
	StatusCanceled  ExecutionStatus = "canceled"
)

// NewExecutor creates a new tool executor
func NewExecutor(registry *Registry, config *ExecutorConfig) *Executor {
	if config == nil {
		config = &ExecutorConfig{
			MaxConcurrency: 3,
			DefaultTimeout: 30 * time.Second,
			EnableLogging:  true,
			LogExecution:   true,
			RetryAttempts:  2,
			RetryDelay:     time.Second,
		}
	}

	// Create semaphore for concurrency control
	var semaphore chan struct{}
	if config.MaxConcurrency > 0 {
		semaphore = make(chan struct{}, config.MaxConcurrency)
	}

	// Create rate limiter from registry config if available
	var limiter *rate.Limiter
	if registry != nil && registry.config != nil && registry.config.RateLimit.CallsPerMinute > 0 {
		limiter = rate.NewLimiter(
			rate.Every(time.Minute/time.Duration(registry.config.RateLimit.CallsPerMinute)),
			registry.config.RateLimit.BurstSize,
		)
	}

	executor := &Executor{
		registry:   registry,
		limiter:    limiter,
		semaphore:  semaphore,
		logger:     &DefaultLogger{},
		executions: make(map[string]*ExecutionInfo),
		config:     config,
	}

	return executor
}

// SetLogger sets a custom logger for the executor
func (e *Executor) SetLogger(logger Logger) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.logger = logger
}

// Execute runs a tool with the given name and arguments
func (e *Executor) Execute(ctx context.Context, execCtx *ExecutionContext, toolName string, args map[string]interface{}) (*ToolResult, error) {
	if execCtx == nil {
		execCtx = &ExecutionContext{
			RequestID: uuid.New().String(),
		}
	}

	// Create execution info
	execInfo := &ExecutionInfo{
		ID:        uuid.New().String(),
		ToolName:  toolName,
		StartTime: time.Now(),
		Status:    StatusPending,
		Context:   execCtx,
		Arguments: args,
		Attempts:  0,
	}

	// Store execution info
	e.mu.Lock()
	e.executions[execInfo.ID] = execInfo
	e.mu.Unlock()

	defer func() {
		// Clean up execution info after some time
		go func() {
			time.Sleep(5 * time.Minute)
			e.mu.Lock()
			delete(e.executions, execInfo.ID)
			e.mu.Unlock()
		}()
	}()

	// Log execution start
	if e.config.LogExecution {
		e.logger.Info("Starting tool execution: %s (ID: %s, Request: %s)",
			toolName, execInfo.ID, execCtx.RequestID)
	}

	// Execute with retries
	var result *ToolResult
	var err error

	for attempt := 1; attempt <= e.config.RetryAttempts+1; attempt++ {
		execInfo.Attempts = attempt
		result, err = e.executeOnce(ctx, execInfo)

		if err == nil {
			break
		}

		// Don't retry certain types of errors
		if isNonRetryableError(err) {
			break
		}

		if attempt < e.config.RetryAttempts+1 {
			e.logger.Warn("Tool execution failed (attempt %d/%d), retrying: %v",
				attempt, e.config.RetryAttempts+1, err)

			select {
			case <-ctx.Done():
				err = ctx.Err()
				goto done
			case <-time.After(e.config.RetryDelay * time.Duration(attempt)):
				// Continue to next attempt
			}
		}
	}

done:
	// Update final status
	now := time.Now()
	execInfo.EndTime = &now
	execInfo.Result = result
	execInfo.Error = err

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			execInfo.Status = StatusTimeout
		} else if ctx.Err() == context.Canceled {
			execInfo.Status = StatusCanceled
		} else {
			execInfo.Status = StatusFailed
		}
	} else {
		execInfo.Status = StatusCompleted
	}

	// Log execution end
	if e.config.LogExecution {
		elapsed := now.Sub(execInfo.StartTime)
		if err != nil {
			e.logger.Error("Tool execution failed: %s (ID: %s, Elapsed: %v, Error: %v)",
				toolName, execInfo.ID, elapsed, err)
		} else {
			e.logger.Info("Tool execution completed: %s (ID: %s, Elapsed: %v)",
				toolName, execInfo.ID, elapsed)
		}
	}

	return result, err
}

// executeOnce performs a single execution attempt
func (e *Executor) executeOnce(ctx context.Context, execInfo *ExecutionInfo) (*ToolResult, error) {
	// Check rate limits
	if err := e.checkRateLimit(ctx); err != nil {
		return nil, fmt.Errorf("rate limit check failed: %w", err)
	}

	// Get tool from registry
	tool, exists := e.registry.Get(execInfo.ToolName)
	if !exists {
		return nil, fmt.Errorf("tool '%s' not found", execInfo.ToolName)
	}

	// Check if tool is enabled
	if !e.registry.IsEnabled(execInfo.ToolName) {
		return nil, fmt.Errorf("tool '%s' is disabled", execInfo.ToolName)
	}

	// Validate arguments
	if err := tool.Validate(execInfo.Arguments); err != nil {
		return nil, fmt.Errorf("argument validation failed: %w", err)
	}

	// Acquire semaphore for concurrency control
	if e.semaphore != nil {
		select {
		case e.semaphore <- struct{}{}:
			defer func() { <-e.semaphore }()
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// Update status to running
	execInfo.Status = StatusRunning

	// Create timeout context
	execCtx := ctx
	if e.config.DefaultTimeout > 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, e.config.DefaultTimeout)
		defer cancel()
	}

	// Execute the tool
	result, err := tool.Execute(execCtx, execInfo.Arguments)
	if err != nil {
		return nil, fmt.Errorf("tool execution failed: %w", err)
	}

	if result == nil {
		return nil, fmt.Errorf("tool returned nil result")
	}

	return result, nil
}

// ExecuteAsync runs a tool asynchronously and returns the execution ID
func (e *Executor) ExecuteAsync(ctx context.Context, execCtx *ExecutionContext, toolName string, args map[string]interface{}) (string, error) {
	if execCtx == nil {
		execCtx = &ExecutionContext{
			RequestID: uuid.New().String(),
		}
	}

	execID := uuid.New().String()

	go func() {
		result, err := e.Execute(ctx, execCtx, toolName, args)

		// Store result for later retrieval
		e.mu.Lock()
		if execInfo, exists := e.executions[execID]; exists {
			execInfo.Result = result
			execInfo.Error = err
		}
		e.mu.Unlock()
	}()

	return execID, nil
}

// GetExecutionInfo returns information about a specific execution
func (e *Executor) GetExecutionInfo(executionID string) (*ExecutionInfo, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	info, exists := e.executions[executionID]
	return info, exists
}

// ListExecutions returns information about all current executions
func (e *Executor) ListExecutions() []*ExecutionInfo {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var executions []*ExecutionInfo
	for _, info := range e.executions {
		executions = append(executions, info)
	}

	return executions
}

// CancelExecution cancels a running execution
func (e *Executor) CancelExecution(executionID string) error {
	e.mu.RLock()
	info, exists := e.executions[executionID]
	e.mu.RUnlock()

	if !exists {
		return fmt.Errorf("execution '%s' not found", executionID)
	}

	if info.Status != StatusRunning {
		return fmt.Errorf("execution '%s' is not running", executionID)
	}

	// Note: In a real implementation, you would need to store and cancel the context
	// This is a simplified version
	info.Status = StatusCanceled
	now := time.Now()
	info.EndTime = &now

	return nil
}

// GetStats returns statistics about tool executions
func (e *Executor) GetStats() map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()

	stats := map[string]interface{}{
		"total_executions":     len(e.executions),
		"status_counts":        make(map[ExecutionStatus]int),
		"tool_usage_counts":    make(map[string]int),
		"avg_execution_time":   0.0,
		"total_execution_time": time.Duration(0),
	}

	statusCounts := make(map[ExecutionStatus]int)
	toolCounts := make(map[string]int)
	var totalTime time.Duration
	var completedCount int

	for _, info := range e.executions {
		statusCounts[info.Status]++
		toolCounts[info.ToolName]++

		if info.EndTime != nil {
			execTime := info.EndTime.Sub(info.StartTime)
			totalTime += execTime
			completedCount++
		}
	}

	stats["status_counts"] = statusCounts
	stats["tool_usage_counts"] = toolCounts
	stats["total_execution_time"] = totalTime

	if completedCount > 0 {
		stats["avg_execution_time"] = float64(totalTime.Nanoseconds()) / float64(completedCount) / 1e6 // in milliseconds
	}

	return stats
}

// checkRateLimit checks if the current request is within rate limits
func (e *Executor) checkRateLimit(ctx context.Context) error {
	if e.limiter == nil {
		return nil
	}

	if !e.limiter.Allow() {
		return fmt.Errorf("rate limit exceeded")
	}

	return nil
}

// isNonRetryableError determines if an error should not be retried
func isNonRetryableError(err error) bool {
	// Don't retry validation errors, not found errors, etc.
	errorString := err.Error()
	nonRetryablePatterns := []string{
		"not found",
		"validation failed",
		"disabled",
		"invalid",
		"unauthorized",
		"forbidden",
	}

	for _, pattern := range nonRetryablePatterns {
		if len(errorString) > 0 && len(pattern) > 0 {
			// Simple string contains check
			for i := 0; i <= len(errorString)-len(pattern); i++ {
				if errorString[i:i+len(pattern)] == pattern {
					return true
				}
			}
		}
	}

	return false
}
