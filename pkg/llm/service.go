// Package llm implements the LLM (Large Language Model) pillar.
// This pillar focuses on provider management, load balancing, and generation operations.
package llm

import (
	"context"
	"fmt"
	"sync"
	"time"
	
	"github.com/liliang-cn/rago/v2/pkg/core"
	"github.com/liliang-cn/rago/v2/pkg/llm/providers"
)

// Service implements the LLM pillar service interface.
// This is the main entry point for all LLM operations including provider
// management and text generation.
type Service struct {
	config          core.LLMConfig
	pool            *ProviderPool
	loadBalancer    *LoadBalancer
	healthChecker   *HealthChecker
	metricsCollector *MetricsCollector
	circuitBreaker  *CircuitBreakerManager
	
	// Lifecycle management
	started bool
	mu      sync.RWMutex
}

// NewService creates a new LLM service instance.
func NewService(config core.LLMConfig) (*Service, error) {
	// Create provider pool
	pool := NewProviderPool(config.LoadBalancing, config.HealthCheck)
	
	// Create load balancer
	loadBalancer := NewLoadBalancer(config.LoadBalancing.Strategy, pool)
	
	// Create health checker
	healthChecker := NewHealthChecker(config.HealthCheck, pool)
	
	// Create metrics collector
	metricsCollector := NewMetricsCollector()
	
	// Create circuit breaker manager
	circuitBreaker := NewCircuitBreakerManager(3, 30*time.Second, pool)
	
	service := &Service{
		config:          config,
		pool:            pool,
		loadBalancer:    loadBalancer,
		healthChecker:   healthChecker,
		metricsCollector: metricsCollector,
		circuitBreaker:  circuitBreaker,
	}
	
	// Initialize configured providers
	if err := service.initializeProviders(); err != nil {
		return nil, fmt.Errorf("failed to initialize providers: %w", err)
	}
	
	// Start health checking
	if err := healthChecker.Start(); err != nil {
		return nil, fmt.Errorf("failed to start health checker: %w", err)
	}
	
	service.started = true
	return service, nil
}

// ===== PROVIDER MANAGEMENT =====

// AddProvider adds a new provider to the service.
func (s *Service) AddProvider(name string, config core.ProviderConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	return s.pool.AddProvider(name, config)
}

// RemoveProvider removes a provider from the service.
func (s *Service) RemoveProvider(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	return s.pool.RemoveProvider(name)
}

// ListProviders lists all registered providers.
func (s *Service) ListProviders() []core.ProviderInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return s.pool.ListProviders()
}

// GetProviderHealth gets the health status of all providers.
func (s *Service) GetProviderHealth() map[string]core.HealthStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return s.pool.GetProviderHealth()
}

// ===== GENERATION OPERATIONS =====

// Generate generates text using the configured providers.
func (s *Service) Generate(ctx context.Context, req core.GenerationRequest) (*core.GenerationResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if !s.started {
		return nil, core.ErrServiceUnavailable
	}
	
	// Select provider using load balancer
	providerEntry, err := s.loadBalancer.SelectProvider()
	if err != nil {
		return nil, err
	}
	
	// Check circuit breaker
	if !s.circuitBreaker.ShouldAllowRequest(providerEntry.Name) {
		return nil, core.ErrProviderUnhealthy
	}
	
	// Convert request to LLM pillar format
	providerReq := s.convertToProviderRequest(req)
	
	// Record metrics for active request
	s.loadBalancer.updateProviderSelection(providerEntry.Name)
	defer s.loadBalancer.FinishRequest(providerEntry.Name)
	
	startTime := time.Now()
	
	// Perform generation using LLM pillar provider interface
	providerResp, err := providerEntry.Provider.Generate(ctx, providerReq)
	
	latency := time.Since(startTime)
	success := err == nil
	
	// Record circuit breaker result
	s.circuitBreaker.RecordResult(providerEntry.Name, success)
	
	// Record load balancer metrics
	s.loadBalancer.RecordRequest(providerEntry.Name, latency, success)
	
	if err != nil {
		// Record failure metrics
		errorType := "generation_failed"
		if core.IsTimeoutError(err) {
			errorType = "timeout"
		} else if core.IsNetworkError(err) {
			errorType = "network"
		}
		s.metricsCollector.RecordRequest(providerEntry.Name, false, latency, nil, errorType)
		return nil, fmt.Errorf("generation failed: %w", err)
	}
	
	// Convert provider response to core response
	response := &core.GenerationResponse{
		Content:  providerResp.Content,
		Model:    providerResp.Model,
		Provider: providerResp.Provider,
		Usage:    core.TokenUsage{
			PromptTokens:     providerResp.Usage.PromptTokens,
			CompletionTokens: providerResp.Usage.CompletionTokens,
			TotalTokens:      providerResp.Usage.TotalTokens,
		},
		Metadata: providerResp.Metadata,
		Duration: latency,
	}
	
	// Record success metrics
	s.metricsCollector.RecordRequest(providerEntry.Name, true, latency, &response.Usage, "")
	
	return response, nil
}

// Stream generates text with streaming using the configured providers.
func (s *Service) Stream(ctx context.Context, req core.GenerationRequest, callback core.StreamCallback) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if !s.started {
		return core.ErrServiceUnavailable
	}
	
	// Select provider using load balancer
	providerEntry, err := s.loadBalancer.SelectProvider()
	if err != nil {
		return err
	}
	
	// Check circuit breaker
	if !s.circuitBreaker.ShouldAllowRequest(providerEntry.Name) {
		return core.ErrProviderUnhealthy
	}
	
	// Convert request to LLM pillar format
	providerReq := s.convertToProviderRequest(req)
	
	// Record metrics for active request
	s.loadBalancer.updateProviderSelection(providerEntry.Name)
	defer s.loadBalancer.FinishRequest(providerEntry.Name)
	
	startTime := time.Now()
	chunksCount := 0
	
	// Create LLM pillar stream callback
	providerCallback := func(chunk *providers.StreamChunk) {
		chunksCount++
		
		// Convert to core stream chunk
		coreChunk := core.StreamChunk{
			Content:  chunk.Content,
			Delta:    chunk.Delta,
			Finished: chunk.Finished,
			Usage: core.TokenUsage{
				PromptTokens:     chunk.Usage.PromptTokens,
				CompletionTokens: chunk.Usage.CompletionTokens,
				TotalTokens:      chunk.Usage.TotalTokens,
			},
			Duration: time.Since(startTime),
		}
		
		callback(coreChunk)
	}
	
	// Perform streaming generation
	err = providerEntry.Provider.Stream(ctx, providerReq, providerCallback)
	
	// Final chunk is handled by the provider callback
	
	duration := time.Since(startTime)
	success := err == nil
	
	// Record circuit breaker result
	s.circuitBreaker.RecordResult(providerEntry.Name, success)
	
	// Record load balancer metrics
	s.loadBalancer.RecordRequest(providerEntry.Name, duration, success)
	
	// Record streaming metrics
	errorType := ""
	if err != nil {
		if core.IsTimeoutError(err) {
			errorType = "timeout"
		} else if core.IsNetworkError(err) {
			errorType = "network"
		} else {
			errorType = "streaming_failed"
		}
	}
	
	s.metricsCollector.RecordStreamingRequest(providerEntry.Name, success, duration, chunksCount, 0, errorType)
	
	if err != nil {
		return fmt.Errorf("streaming failed: %w", err)
	}
	
	return nil
}

// ===== TOOL OPERATIONS =====

// GenerateWithTools generates text with tool calling capability.
func (s *Service) GenerateWithTools(ctx context.Context, req core.ToolGenerationRequest) (*core.ToolGenerationResponse, error) {
	// For now, delegate to regular generation and add empty tool calls
	// This can be enhanced later with actual tool integration
	genReq := req.GenerationRequest
	response, err := s.Generate(ctx, genReq)
	if err != nil {
		return nil, err
	}
	
	return &core.ToolGenerationResponse{
		GenerationResponse: *response,
		ToolCalls:         []core.ToolCall{}, // No tool calls for now
	}, nil
}

// StreamWithTools generates text with tool calling in streaming mode.
func (s *Service) StreamWithTools(ctx context.Context, req core.ToolGenerationRequest, callback core.ToolStreamCallback) error {
	// For now, delegate to regular streaming and add empty tool calls
	// This can be enhanced later with actual tool integration
	genReq := req.GenerationRequest
	return s.Stream(ctx, genReq, func(chunk core.StreamChunk) error {
		toolChunk := core.ToolStreamChunk{
			StreamChunk: chunk,
			ToolCalls:   []core.ToolCall{}, // No tool calls for now
		}
		return callback(toolChunk)
	})
}

// ===== BATCH OPERATIONS =====

// GenerateBatch generates text for multiple requests in batch.
func (s *Service) GenerateBatch(ctx context.Context, requests []core.GenerationRequest) ([]core.GenerationResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if !s.started {
		return nil, core.ErrServiceUnavailable
	}
	
	if len(requests) == 0 {
		return []core.GenerationResponse{}, nil
	}
	
	responses := make([]core.GenerationResponse, len(requests))
	errors := make([]error, len(requests))
	
	// Process requests concurrently with limited concurrency
	maxConcurrency := 5 // Could be configurable
	semaphore := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	
	for i, req := range requests {
		wg.Add(1)
		go func(index int, request core.GenerationRequest) {
			defer wg.Done()
			
			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			// Generate response
			response, err := s.Generate(ctx, request)
			if err != nil {
				errors[index] = err
				return
			}
			
			responses[index] = *response
		}(i, req)
	}
	
	wg.Wait()
	
	// Check for errors and build final response
	finalResponses := make([]core.GenerationResponse, 0, len(responses))
	for i, response := range responses {
		if errors[i] != nil {
			// Log error but continue with other responses
			continue
		}
		finalResponses = append(finalResponses, response)
	}
	
	// If no successful responses, return error
	if len(finalResponses) == 0 {
		return nil, core.ErrGenerationFailed
	}
	
	return finalResponses, nil
}

// Close closes the LLM service and cleans up resources.
func (s *Service) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if !s.started {
		return nil
	}
	
	// Stop health checker
	if s.healthChecker != nil {
		s.healthChecker.Stop()
	}
	
	// Close provider pool
	if s.pool != nil {
		s.pool.Close()
	}
	
	s.started = false
	return nil
}

// ===== SERVICE STATUS AND METRICS =====

// GetMetrics returns comprehensive service metrics
func (s *Service) GetMetrics() *ServiceMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return s.metricsCollector.GetServiceMetrics(s.healthChecker)
}

// GetProviderMetrics returns detailed metrics for a specific provider
func (s *Service) GetProviderMetrics(providerName string) *ProviderPerformanceMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return s.metricsCollector.GetProviderMetrics(providerName)
}

// IsStarted returns true if the service is started and ready
func (s *Service) IsStarted() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return s.started
}

// ===== PRIVATE METHODS =====

// initializeProviders initializes all configured providers
func (s *Service) initializeProviders() error {
	for name, config := range s.config.Providers {
		if err := s.pool.AddProvider(name, config); err != nil {
			return fmt.Errorf("failed to add provider %s: %w", name, err)
		}
	}
	return nil
}

// convertToProviderRequest converts core.GenerationRequest to LLM pillar provider request
func (s *Service) convertToProviderRequest(req core.GenerationRequest) *providers.GenerationRequest {
	// Convert messages
	providerMessages := make([]providers.Message, len(req.Context))
	for i, msg := range req.Context {
		providerMessages[i] = providers.Message{
			Role:       msg.Role,
			Content:    msg.Content,
			ToolCallID: msg.ToolCallID,
		}
	}
	
	return &providers.GenerationRequest{
		Prompt:      req.Prompt,
		Model:       req.Model,
		Parameters:  req.Parameters,
		Context:     providerMessages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	}
}