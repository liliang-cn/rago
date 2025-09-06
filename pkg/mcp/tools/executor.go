package tools

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/core"
)

// Executor handles tool execution with concurrency control, retries, and isolation
type Executor struct {
	registry         *Registry
	cache            *Cache
	serverManager    ServerManager
	maxConcurrent    int
	defaultTimeout   time.Duration
	retryPolicy      RetryPolicy
	isolationMode    IsolationMode
	executionPool    *ExecutionPool
	metrics          *ExecutionMetrics
	middlewares      []ExecutionMiddleware
	mu               sync.RWMutex
}

// ServerManager is defined in interfaces.go

// RetryPolicy defines retry behavior for failed tool calls
type RetryPolicy struct {
	MaxRetries     int
	InitialDelay   time.Duration
	MaxDelay       time.Duration
	BackoffFactor  float64
	RetryableErrors []string
}

// IsolationMode defines how tool execution is isolated
type IsolationMode int

const (
	IsolationNone IsolationMode = iota
	IsolationProcess
	IsolationContainer
)

// ExecutionPool manages concurrent tool execution
type ExecutionPool struct {
	workers       int
	taskQueue     chan *ExecutionTask
	resultQueue   chan *ExecutionResult
	workerGroup   sync.WaitGroup
	shutdownCh    chan struct{}
	activeWorkers int32
	mu            sync.Mutex
}

// ExecutionTask represents a tool execution task
type ExecutionTask struct {
	ID          string
	Tool        *RegisteredTool
	Request     *core.ToolCallRequest
	Context     context.Context
	ResultChan  chan *ExecutionResult
	RetryCount  int
	StartTime   time.Time
	Timeout     time.Duration
}

// ExecutionResult represents the result of tool execution
type ExecutionResult struct {
	ID         string
	Success    bool
	Data       interface{}
	Error      error
	Duration   time.Duration
	RetryCount int
	CacheHit   bool
	Metadata   map[string]interface{}
}

// ExecutionMiddleware allows modification of execution behavior
type ExecutionMiddleware func(next ExecutionHandler) ExecutionHandler

// ExecutionHandler handles tool execution
type ExecutionHandler func(ctx context.Context, task *ExecutionTask) *ExecutionResult

// ExecutionMetrics tracks execution metrics
type ExecutionMetrics struct {
	mu               sync.RWMutex
	totalExecutions  int64
	successfulExecs  int64
	failedExecs      int64
	cacheHits        int64
	cacheMisses      int64
	totalRetries     int64
	avgExecutionTime time.Duration
	executionTimes   []time.Duration
	maxQueueSize     int
	currentQueueSize int
}

// NewExecutor creates a new tool executor
func NewExecutor(registry *Registry, cache *Cache, serverManager ServerManager) *Executor {
	e := &Executor{
		registry:       registry,
		cache:          cache,
		serverManager:  serverManager,
		maxConcurrent:  10,
		defaultTimeout: 30 * time.Second,
		retryPolicy: RetryPolicy{
			MaxRetries:    3,
			InitialDelay:  1 * time.Second,
			MaxDelay:      30 * time.Second,
			BackoffFactor: 2.0,
		},
		isolationMode: IsolationNone,
		metrics:       &ExecutionMetrics{},
	}
	
	// Initialize execution pool
	e.executionPool = NewExecutionPool(e.maxConcurrent)
	
	// Add default middlewares
	e.AddMiddleware(e.loggingMiddleware())
	e.AddMiddleware(e.metricsMiddleware())
	e.AddMiddleware(e.rateLimitMiddleware())
	
	return e
}

// NewExecutionPool creates a new execution pool
func NewExecutionPool(workers int) *ExecutionPool {
	return &ExecutionPool{
		workers:     workers,
		taskQueue:   make(chan *ExecutionTask, workers*2),
		resultQueue: make(chan *ExecutionResult, workers*2),
		shutdownCh:  make(chan struct{}),
	}
}

// Start starts the executor
func (e *Executor) Start(ctx context.Context) error {
	// Start execution pool
	e.executionPool.Start(e.executeTask)
	return nil
}

// Stop stops the executor
func (e *Executor) Stop() error {
	// Stop execution pool
	e.executionPool.Stop()
	return nil
}

// Execute executes a tool call synchronously
func (e *Executor) Execute(ctx context.Context, request *core.ToolCallRequest) (*core.ToolCallResponse, error) {
	// Get tool from registry
	tool, err := e.registry.Get(request.ToolName)
	if err != nil {
		return nil, err
	}
	
	// Check if tool is available
	if !tool.Available {
		return nil, fmt.Errorf("tool %s is not available", request.ToolName)
	}
	
	// Check rate limit
	if tool.RateLimit != nil && !tool.RateLimit.CheckRateLimit() {
		return nil, fmt.Errorf("rate limit exceeded for tool %s", request.ToolName)
	}
	
	// Check cache if tool is cacheable
	if tool.Cacheable {
		if cachedResult := e.cache.Get(request); cachedResult != nil {
			e.metrics.RecordCacheHit()
			return cachedResult, nil
		}
		e.metrics.RecordCacheMiss()
	}
	
	// Create execution task
	task := &ExecutionTask{
		ID:         generateTaskID(),
		Tool:       tool,
		Request:    request,
		Context:    ctx,
		ResultChan: make(chan *ExecutionResult, 1),
		StartTime:  time.Now(),
		Timeout:    tool.Timeout,
	}
	
	// Submit task to execution pool
	e.executionPool.Submit(task)
	
	// Wait for result
	select {
	case result := <-task.ResultChan:
		response := &core.ToolCallResponse{
			ToolName:  request.ToolName,
			Success:   result.Success,
			Result:    result.Data,
			Error:     result.Error,
			Duration:  result.Duration,
			Metadata:  result.Metadata,
		}
		
		// Cache successful results if tool is cacheable
		if result.Success && tool.Cacheable {
			e.cache.Set(request, response, tool.CacheDuration)
		}
		
		// Update tool usage stats
		e.registry.UpdateUsage(tool.ID, result.Success)
		
		return response, nil
		
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// ExecuteAsync executes a tool call asynchronously
func (e *Executor) ExecuteAsync(ctx context.Context, request *core.ToolCallRequest) (<-chan *core.ToolCallResponse, error) {
	// Get tool from registry
	tool, err := e.registry.Get(request.ToolName)
	if err != nil {
		return nil, err
	}
	
	// Check if tool is available
	if !tool.Available {
		return nil, fmt.Errorf("tool %s is not available", request.ToolName)
	}
	
	// Create response channel
	responseChan := make(chan *core.ToolCallResponse, 1)
	
	// Execute in goroutine
	go func() {
		response, err := e.Execute(ctx, request)
		if err != nil {
			responseChan <- &core.ToolCallResponse{
				ToolName: request.ToolName,
				Success:  false,
				Error:    err,
			}
		} else {
			responseChan <- response
		}
	}()
	
	return responseChan, nil
}

// ExecuteBatch executes multiple tool calls in parallel
func (e *Executor) ExecuteBatch(ctx context.Context, requests []core.ToolCallRequest) ([]core.ToolCallResponse, error) {
	responses := make([]core.ToolCallResponse, len(requests))
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error
	
	for i, request := range requests {
		wg.Add(1)
		go func(idx int, req core.ToolCallRequest) {
			defer wg.Done()
			
			response, err := e.Execute(ctx, &req)
			mu.Lock()
			defer mu.Unlock()
			
			if err != nil && firstErr == nil {
				firstErr = err
			}
			
			if response != nil {
				responses[idx] = *response
			} else {
				responses[idx] = core.ToolCallResponse{
					ToolName: req.ToolName,
					Success:  false,
					Error:    err,
				}
			}
		}(i, request)
	}
	
	wg.Wait()
	return responses, firstErr
}

// executeTask executes a single task
func (e *Executor) executeTask(task *ExecutionTask) {
	// Build execution handler chain
	handler := e.buildHandlerChain()
	
	// Execute with timeout
	ctx, cancel := context.WithTimeout(task.Context, task.Timeout)
	defer cancel()
	
	task.Context = ctx
	result := handler(ctx, task)
	
	// Send result
	select {
	case task.ResultChan <- result:
	default:
		// Result channel full or closed
	}
}

// buildHandlerChain builds the middleware chain
func (e *Executor) buildHandlerChain() ExecutionHandler {
	// Start with the core execution handler
	handler := e.coreExecutionHandler()
	
	// Apply middlewares in reverse order
	for i := len(e.middlewares) - 1; i >= 0; i-- {
		handler = e.middlewares[i](handler)
	}
	
	return handler
}

// coreExecutionHandler is the core tool execution logic
func (e *Executor) coreExecutionHandler() ExecutionHandler {
	return func(ctx context.Context, task *ExecutionTask) *ExecutionResult {
		startTime := time.Now()
		
		// Get server for the tool
		server, err := e.serverManager.GetServer(task.Tool.ServerName)
		if err != nil {
			return &ExecutionResult{
				ID:       task.ID,
				Success:  false,
				Error:    fmt.Errorf("failed to get server: %w", err),
				Duration: time.Since(startTime),
			}
		}
		
		// Execute tool call with retries
		var result *ToolResult
		var lastErr error
		
		for attempt := 0; attempt <= task.Tool.MaxRetries; attempt++ {
			if attempt > 0 {
				// Calculate retry delay
				delay := e.calculateRetryDelay(attempt)
				select {
				case <-time.After(delay):
				case <-ctx.Done():
					return &ExecutionResult{
						ID:       task.ID,
						Success:  false,
						Error:    ctx.Err(),
						Duration: time.Since(startTime),
					}
				}
			}
			
			// Call the tool
			result, err = server.CallTool(ctx, task.Tool.Name, task.Request.Arguments)
			if err == nil && result.Success {
				break
			}
			
			lastErr = err
			task.RetryCount++
			
			// Check if error is retryable
			if !e.isRetryableError(err) {
				break
			}
		}
		
		if result != nil && result.Success {
			return &ExecutionResult{
				ID:         task.ID,
				Success:    true,
				Data:       result.Data,
				Duration:   time.Since(startTime),
				RetryCount: task.RetryCount,
			}
		}
		
		return &ExecutionResult{
			ID:         task.ID,
			Success:    false,
			Error:      lastErr,
			Duration:   time.Since(startTime),
			RetryCount: task.RetryCount,
		}
	}
}

// calculateRetryDelay calculates the delay for a retry attempt
func (e *Executor) calculateRetryDelay(attempt int) time.Duration {
	delay := e.retryPolicy.InitialDelay
	for i := 1; i < attempt; i++ {
		delay = time.Duration(float64(delay) * e.retryPolicy.BackoffFactor)
		if delay > e.retryPolicy.MaxDelay {
			delay = e.retryPolicy.MaxDelay
			break
		}
	}
	return delay
}

// isRetryableError checks if an error is retryable
func (e *Executor) isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	
	// Check against retryable error patterns
	errStr := err.Error()
	for _, pattern := range e.retryPolicy.RetryableErrors {
		if contains(errStr, pattern) {
			return true
		}
	}
	
	// Default retryable errors
	retryablePatterns := []string{
		"timeout",
		"connection refused",
		"temporary failure",
		"rate limit",
	}
	
	for _, pattern := range retryablePatterns {
		if contains(errStr, pattern) {
			return true
		}
	}
	
	return false
}

// AddMiddleware adds an execution middleware
func (e *Executor) AddMiddleware(middleware ExecutionMiddleware) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.middlewares = append(e.middlewares, middleware)
}

// loggingMiddleware provides execution logging
func (e *Executor) loggingMiddleware() ExecutionMiddleware {
	return func(next ExecutionHandler) ExecutionHandler {
		return func(ctx context.Context, task *ExecutionTask) *ExecutionResult {
			// Log execution start
			fmt.Printf("Executing tool %s (task %s)\n", task.Tool.Name, task.ID)
			
			result := next(ctx, task)
			
			// Log execution result
			if result.Success {
				fmt.Printf("Tool %s executed successfully in %v\n", task.Tool.Name, result.Duration)
			} else {
				fmt.Printf("Tool %s failed: %v\n", task.Tool.Name, result.Error)
			}
			
			return result
		}
	}
}

// metricsMiddleware collects execution metrics
func (e *Executor) metricsMiddleware() ExecutionMiddleware {
	return func(next ExecutionHandler) ExecutionHandler {
		return func(ctx context.Context, task *ExecutionTask) *ExecutionResult {
			result := next(ctx, task)
			
			// Update metrics
			e.metrics.RecordExecution(result)
			
			return result
		}
	}
}

// rateLimitMiddleware enforces rate limits
func (e *Executor) rateLimitMiddleware() ExecutionMiddleware {
	return func(next ExecutionHandler) ExecutionHandler {
		return func(ctx context.Context, task *ExecutionTask) *ExecutionResult {
			// Rate limit check is done before submission
			// This middleware could implement additional rate limiting logic
			return next(ctx, task)
		}
	}
}

// GetMetrics returns execution metrics
func (e *Executor) GetMetrics() map[string]interface{} {
	return e.metrics.GetMetrics()
}

// ExecutionPool methods

// Start starts the execution pool workers
func (p *ExecutionPool) Start(handler func(*ExecutionTask)) {
	for i := 0; i < p.workers; i++ {
		p.workerGroup.Add(1)
		go p.worker(handler)
	}
}

// Stop stops the execution pool
func (p *ExecutionPool) Stop() {
	close(p.shutdownCh)
	p.workerGroup.Wait()
}

// Submit submits a task to the execution pool
func (p *ExecutionPool) Submit(task *ExecutionTask) {
	select {
	case p.taskQueue <- task:
	default:
		// Queue full, execute synchronously
		// In production, this should be handled better
		go func() {
			p.taskQueue <- task
		}()
	}
}

// worker is the execution pool worker
func (p *ExecutionPool) worker(handler func(*ExecutionTask)) {
	defer p.workerGroup.Done()
	
	for {
		select {
		case <-p.shutdownCh:
			return
		case task := <-p.taskQueue:
			if task != nil {
				handler(task)
			}
		}
	}
}

// ExecutionMetrics methods

// RecordExecution records an execution result
func (m *ExecutionMetrics) RecordExecution(result *ExecutionResult) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.totalExecutions++
	if result.Success {
		m.successfulExecs++
	} else {
		m.failedExecs++
	}
	
	if result.CacheHit {
		m.cacheHits++
	} else {
		m.cacheMisses++
	}
	
	m.totalRetries += int64(result.RetryCount)
	
	// Update average execution time
	m.executionTimes = append(m.executionTimes, result.Duration)
	if len(m.executionTimes) > 1000 {
		m.executionTimes = m.executionTimes[500:] // Keep last 500
	}
	
	total := time.Duration(0)
	for _, d := range m.executionTimes {
		total += d
	}
	m.avgExecutionTime = total / time.Duration(len(m.executionTimes))
}

// RecordCacheHit records a cache hit
func (m *ExecutionMetrics) RecordCacheHit() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cacheHits++
}

// RecordCacheMiss records a cache miss
func (m *ExecutionMetrics) RecordCacheMiss() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cacheMisses++
}

// GetMetrics returns current metrics
func (m *ExecutionMetrics) GetMetrics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	successRate := float64(0)
	if m.totalExecutions > 0 {
		successRate = float64(m.successfulExecs) / float64(m.totalExecutions)
	}
	
	cacheHitRate := float64(0)
	if (m.cacheHits + m.cacheMisses) > 0 {
		cacheHitRate = float64(m.cacheHits) / float64(m.cacheHits+m.cacheMisses)
	}
	
	return map[string]interface{}{
		"total_executions":   m.totalExecutions,
		"successful_execs":   m.successfulExecs,
		"failed_execs":       m.failedExecs,
		"success_rate":       successRate,
		"cache_hits":         m.cacheHits,
		"cache_misses":       m.cacheMisses,
		"cache_hit_rate":     cacheHitRate,
		"total_retries":      m.totalRetries,
		"avg_execution_time": m.avgExecutionTime,
	}
}

// generateTaskID generates a unique task ID
func generateTaskID() string {
	return fmt.Sprintf("task_%d", time.Now().UnixNano())
}