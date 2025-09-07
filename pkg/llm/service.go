// Package llm implements the LLM (Large Language Model) pillar.
// This pillar focuses on provider management, load balancing, and generation operations.
package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"
	
	"github.com/liliang-cn/rago/v2/pkg/core"
	"github.com/liliang-cn/rago/v2/pkg/llm/providers"
)

// MCPService interface for MCP integration
type MCPService interface {
	GetTools() []core.ToolInfo
	GetToolsForLLM() []core.ToolInfo
}

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
	
	// Tool integration
	mcpService      MCPService
	
	// Lifecycle management
	started bool
	mu      sync.RWMutex
}

// NewService creates a new LLM service instance.
func NewService(config core.LLMConfig) (*Service, error) {
	// Validate configuration
	allProviders := config.Providers.GetProviders()
	if len(allProviders) == 0 {
		return nil, fmt.Errorf("no providers configured: at least one LLM provider must be specified")
	}
	
	enabledProviders := config.Providers.GetEnabledProviders()
	if len(enabledProviders) == 0 {
		return nil, fmt.Errorf("no enabled providers: at least one LLM provider must be enabled")
	}
	
	// Validate default provider exists
	if config.DefaultProvider != "" {
		if _, exists := config.Providers.GetProvider(config.DefaultProvider); !exists {
			return nil, fmt.Errorf("default provider '%s' not found in configured providers", config.DefaultProvider)
		}
	}
	
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
	
	// Don't run health check here - let it happen asynchronously
	// This allows status command to show "checking..." first
	return s.pool.GetProviderHealth()
}

// TriggerHealthCheck triggers an immediate health check on all providers
func (s *Service) TriggerHealthCheck() {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.healthChecker != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		// CheckAllProviders waits for all checks to complete
		s.healthChecker.CheckAllProviders(ctx)
	}
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
	
	// Check if provider supports tool calling
	capabilities := providerEntry.Provider.Capabilities()
	if !capabilities.SupportsToolCalls {
		// Fallback to regular generation for providers without tool support
		genReq := req.GenerationRequest
		response, err := s.Generate(ctx, genReq)
		if err != nil {
			return nil, err
		}
		return &core.ToolGenerationResponse{
			GenerationResponse: *response,
			ToolCalls:         []core.ToolCall{},
		}, nil
	}
	
	// Check circuit breaker
	if !s.circuitBreaker.ShouldAllowRequest(providerEntry.Name) {
		return nil, core.ErrProviderUnhealthy
	}
	
	// If no tools provided but MCP service is available, fetch available tools
	if len(req.Tools) == 0 && s.mcpService != nil {
		mcpTools := s.mcpService.GetToolsForLLM()
		req.Tools = s.convertMCPToolsToLLMTools(mcpTools)
	}
	
	// Convert to provider-specific tool request
	providerReq := s.convertToProviderToolRequest(req)
	
	// Record metrics for active request
	s.loadBalancer.updateProviderSelection(providerEntry.Name)
	defer s.loadBalancer.FinishRequest(providerEntry.Name)
	
	startTime := time.Now()
	
	// Perform generation with tools
	providerResp, err := providerEntry.Provider.GenerateWithTools(ctx, providerReq)
	
	latency := time.Since(startTime)
	success := err == nil
	
	// Record circuit breaker result
	s.circuitBreaker.RecordResult(providerEntry.Name, success)
	
	// Record load balancer metrics
	s.loadBalancer.RecordRequest(providerEntry.Name, latency, success)
	
	if err != nil {
		// Record failure metrics
		errorType := "tool_generation_failed"
		if core.IsTimeoutError(err) {
			errorType = "timeout"
		} else if core.IsNetworkError(err) {
			errorType = "network"
		}
		s.metricsCollector.RecordRequest(providerEntry.Name, false, latency, nil, errorType)
		return nil, fmt.Errorf("tool generation failed: %w", err)
	}
	
	// Convert provider response to core response
	response := &core.ToolGenerationResponse{
		GenerationResponse: core.GenerationResponse{
			Content:  providerResp.Content,
			Model:    providerResp.Model,
			Provider: providerResp.Provider,
			Usage: core.TokenUsage{
				PromptTokens:     providerResp.Usage.PromptTokens,
				CompletionTokens: providerResp.Usage.CompletionTokens,
				TotalTokens:      providerResp.Usage.TotalTokens,
			},
			Metadata: providerResp.Metadata,
			Duration: latency,
		},
		ToolCalls: s.convertProviderToolCalls(providerResp.ToolCalls),
	}
	
	// Record success metrics
	s.metricsCollector.RecordRequest(providerEntry.Name, true, latency, &response.Usage, "")
	
	return response, nil
}

// StreamWithTools generates text with tool calling in streaming mode.
func (s *Service) StreamWithTools(ctx context.Context, req core.ToolGenerationRequest, callback core.ToolStreamCallback) error {
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
	
	// Check if provider supports tool calling
	capabilities := providerEntry.Provider.Capabilities()
	if !capabilities.SupportsToolCalls {
		// Fallback to regular streaming for providers without tool support
		genReq := req.GenerationRequest
		return s.Stream(ctx, genReq, func(chunk core.StreamChunk) error {
			toolChunk := core.ToolStreamChunk{
				StreamChunk: chunk,
				ToolCalls:   []core.ToolCall{},
			}
			return callback(toolChunk)
		})
	}
	
	// Check circuit breaker
	if !s.circuitBreaker.ShouldAllowRequest(providerEntry.Name) {
		return core.ErrProviderUnhealthy
	}
	
	// If no tools provided but MCP service is available, fetch available tools
	if len(req.Tools) == 0 && s.mcpService != nil {
		mcpTools := s.mcpService.GetToolsForLLM()
		req.Tools = s.convertMCPToolsToLLMTools(mcpTools)
	}
	
	// Convert to provider-specific tool request
	providerReq := s.convertToProviderToolRequest(req)
	
	// Record metrics for active request
	s.loadBalancer.updateProviderSelection(providerEntry.Name)
	defer s.loadBalancer.FinishRequest(providerEntry.Name)
	
	startTime := time.Now()
	chunksCount := 0
	
	// Create provider callback
	providerCallback := func(chunk *providers.ToolStreamChunk) {
		chunksCount++
		
		// Convert to core tool stream chunk
		coreChunk := core.ToolStreamChunk{
			StreamChunk: core.StreamChunk{
				Content:  chunk.Content,
				Delta:    chunk.Delta,
				Finished: chunk.Finished,
				Usage: core.TokenUsage{
					PromptTokens:     chunk.Usage.PromptTokens,
					CompletionTokens: chunk.Usage.CompletionTokens,
					TotalTokens:      chunk.Usage.TotalTokens,
				},
				Duration: time.Since(startTime),
			},
			ToolCalls: s.convertProviderToolCalls(chunk.ToolCalls),
		}
		
		callback(coreChunk)
	}
	
	// Perform streaming generation with tools
	err = providerEntry.Provider.StreamWithTools(ctx, providerReq, providerCallback)
	
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
			errorType = "tool_streaming_failed"
		}
	}
	
	s.metricsCollector.RecordStreamingRequest(providerEntry.Name, success, duration, chunksCount, 0, errorType)
	
	if err != nil {
		return fmt.Errorf("tool streaming failed: %w", err)
	}
	
	return nil
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

// ===== STRUCTURED JSON GENERATION =====

// GenerateStructured generates JSON-structured output according to a schema
func (s *Service) GenerateStructured(ctx context.Context, req core.StructuredGenerationRequest) (*core.StructuredResult, error) {
	s.mu.RLock()
	if !s.started {
		s.mu.RUnlock()
		return nil, fmt.Errorf("LLM service not started")
	}
	s.mu.RUnlock()
	
	// Build the prompt that includes JSON instructions
	jsonPrompt := s.buildJSONPrompt(req)
	
	// Create a generation request with JSON formatting
	genReq := core.GenerationRequest{
		Prompt:      jsonPrompt,
		Model:       req.Model,
		Parameters:  req.Parameters,
		Context:     req.Context,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	}
	
	// If provider supports force JSON mode, add it to parameters
	if req.ForceJSON {
		if genReq.Parameters == nil {
			genReq.Parameters = make(map[string]interface{})
		}
		genReq.Parameters["response_format"] = map[string]string{"type": "json_object"}
	}
	
	// Generate the response
	resp, err := s.Generate(ctx, genReq)
	if err != nil {
		return nil, fmt.Errorf("failed to generate structured output: %w", err)
	}
	
	// Parse and validate the JSON response
	return s.parseAndValidateJSON(resp.Content, req.Schema)
}

// ExtractMetadata extracts structured metadata from content
func (s *Service) ExtractMetadata(ctx context.Context, req core.MetadataExtractionRequest) (*core.ExtractedMetadata, error) {
	s.mu.RLock()
	if !s.started {
		s.mu.RUnlock()
		return nil, fmt.Errorf("LLM service not started")
	}
	s.mu.RUnlock()
	
	// Build extraction prompt
	extractionPrompt := s.buildExtractionPrompt(req)
	
	// Create structured generation request for metadata
	structuredReq := core.StructuredGenerationRequest{
		GenerationRequest: core.GenerationRequest{
			Prompt:    extractionPrompt,
			Model:     req.Model,
			Parameters: req.Parameters,
			MaxTokens: 1000, // Usually enough for metadata
		},
		Schema:    &core.ExtractedMetadata{}, // Use ExtractedMetadata as schema
		ForceJSON: true,
	}
	
	// Generate structured output
	result, err := s.GenerateStructured(ctx, structuredReq)
	if err != nil {
		return nil, fmt.Errorf("failed to extract metadata: %w", err)
	}
	
	// Convert result to ExtractedMetadata
	metadata, ok := result.Data.(*core.ExtractedMetadata)
	if !ok {
		return nil, fmt.Errorf("failed to parse extracted metadata")
	}
	
	return metadata, nil
}

// buildJSONPrompt builds a prompt that instructs the model to output JSON
func (s *Service) buildJSONPrompt(req core.StructuredGenerationRequest) string {
	prompt := req.Prompt
	
	// Add JSON formatting instructions
	prompt += "\n\nPlease respond with valid JSON that conforms to the following structure:"
	
	// Add schema description if available
	if req.Schema != nil {
		schemaJSON, _ := json.Marshal(req.Schema)
		prompt += fmt.Sprintf("\n```json\n%s\n```", string(schemaJSON))
	}
	
	// Add example if provided
	if req.ExampleJSON != "" {
		prompt += fmt.Sprintf("\n\nExample output:\n```json\n%s\n```", req.ExampleJSON)
	}
	
	prompt += "\n\nProvide only the JSON response without any additional text or markdown formatting."
	
	return prompt
}

// buildExtractionPrompt builds a prompt for metadata extraction
func (s *Service) buildExtractionPrompt(req core.MetadataExtractionRequest) string {
	prompt := fmt.Sprintf("Extract structured metadata from the following content:\n\n%s\n\n", req.Content)
	
	prompt += "Extract the following information as JSON:\n"
	prompt += "- summary: A brief summary of the content\n"
	prompt += "- keywords: Key terms and concepts\n"
	prompt += "- document_type: The type of document (e.g., article, report, email)\n"
	prompt += "- language: The language of the content\n"
	prompt += "- sentiment: Overall sentiment (positive, negative, neutral)\n"
	
	// Add specific fields if requested
	if len(req.Fields) > 0 {
		prompt += fmt.Sprintf("\nAlso extract these specific fields: %v\n", req.Fields)
	}
	
	return prompt
}

// parseAndValidateJSON parses and validates JSON against a schema
func (s *Service) parseAndValidateJSON(content string, schema interface{}) (*core.StructuredResult, error) {
	// Clean the content (remove markdown code blocks if present)
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)
	
	// Parse the JSON
	var data interface{}
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return &core.StructuredResult{
			Raw:   content,
			Valid: false,
			Error: fmt.Sprintf("invalid JSON: %v", err),
		}, nil
	}
	
	// If schema is provided as a struct, unmarshal into it
	if schema != nil {
		schemaType := reflect.TypeOf(schema)
		if schemaType.Kind() == reflect.Ptr {
			// Create a new instance of the schema type
			newSchema := reflect.New(schemaType.Elem()).Interface()
			if err := json.Unmarshal([]byte(content), newSchema); err != nil {
				return &core.StructuredResult{
					Raw:   content,
					Data:  data,
					Valid: false,
					Error: fmt.Sprintf("schema validation failed: %v", err),
				}, nil
			}
			data = newSchema
		}
	}
	
	return &core.StructuredResult{
		Data:  data,
		Raw:   content,
		Valid: true,
	}, nil
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

// SetMCPService sets the MCP service for tool integration
func (s *Service) SetMCPService(mcpService MCPService) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.mcpService = mcpService
}

// GenerateWithToolExecution generates text with tool calling and automatic execution
func (s *Service) GenerateWithToolExecution(ctx context.Context, req core.ToolGenerationRequest) (*core.ToolGenerationResponse, error) {
	// Initial generation with tools
	response, err := s.GenerateWithTools(ctx, req)
	if err != nil {
		return nil, err
	}
	
	// If no tool calls were made, return the response as-is
	if len(response.ToolCalls) == 0 {
		return response, nil
	}
	
	// Return the response with tool calls for the LLM to handle
	return response, nil
}

// ===== PRIVATE METHODS =====

// initializeProviders initializes all configured providers
func (s *Service) initializeProviders() error {
	enabledProviders := s.config.Providers.GetEnabledProviders()
	for _, config := range enabledProviders {
		// Validate provider configuration
		if config.Type == "" {
			return fmt.Errorf("provider %s has empty type", config.Name)
		}
		
		if config.Name == "" {
			return fmt.Errorf("provider has empty name")
		}
		
		if err := s.pool.AddProvider(config.Name, config); err != nil {
			return fmt.Errorf("failed to add provider %s: %w", config.Name, err)
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

// convertToProviderToolRequest converts core.ToolGenerationRequest to provider request
func (s *Service) convertToProviderToolRequest(req core.ToolGenerationRequest) *providers.ToolGenerationRequest {
	// Convert messages
	providerMessages := make([]providers.Message, len(req.Context))
	for i, msg := range req.Context {
		providerMessages[i] = providers.Message{
			Role:       msg.Role,
			Content:    msg.Content,
			ToolCallID: msg.ToolCallID,
		}
	}
	
	// Convert tools
	providerTools := make([]providers.ToolInfo, len(req.Tools))
	for i, tool := range req.Tools {
		providerTools[i] = providers.ToolInfo{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  tool.InputSchema,
		}
	}
	
	return &providers.ToolGenerationRequest{
		GenerationRequest: providers.GenerationRequest{
			Prompt:      req.Prompt,
			Model:       req.Model,
			Parameters:  req.Parameters,
			Context:     providerMessages,
			MaxTokens:   req.MaxTokens,
			Temperature: req.Temperature,
		},
		Tools:        providerTools,
		ToolChoice:   req.ToolChoice,
		MaxToolCalls: req.MaxToolCalls,
	}
}

// convertProviderToolCalls converts provider tool calls to core tool calls
func (s *Service) convertProviderToolCalls(providerCalls []providers.ToolCall) []core.ToolCall {
	calls := make([]core.ToolCall, len(providerCalls))
	for i, call := range providerCalls {
		calls[i] = core.ToolCall{
			ID:         call.ID,
			Name:       call.Name,
			Parameters: call.Parameters,
		}
	}
	return calls
}

// convertMCPToolsToLLMTools converts MCP tools to LLM tool format
func (s *Service) convertMCPToolsToLLMTools(mcpTools []core.ToolInfo) []core.ToolInfo {
	// The core.ToolInfo is already in the right format
	// Just ensure the tools are properly formatted
	tools := make([]core.ToolInfo, 0, len(mcpTools))
	for _, tool := range mcpTools {
		// Prefix MCP tools with "mcp_" to distinguish them
		if !strings.HasPrefix(tool.Name, "mcp_") {
			tool.Name = "mcp_" + tool.Name
		}
		tools = append(tools, tool)
	}
	return tools
}

// GenerateEmbedding generates embeddings for the provided text using available embedding models.
// This method attempts to find and use an appropriate embedding model from the provider pool.
func (s *Service) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	s.mu.RLock()
	if !s.started {
		s.mu.RUnlock()
		return nil, fmt.Errorf("service not started")
	}
	s.mu.RUnlock()
	
	// Look for a provider that supports embeddings
	providers := s.pool.GetActiveProviders()
	
	for _, providerName := range providers {
		provider := s.pool.GetProviderByName(providerName)
		if provider == nil {
			continue
		}
		
		// Check if provider name indicates embedding capability
		lowerName := strings.ToLower(providerName)
		if strings.Contains(lowerName, "embed") || 
		   strings.Contains(lowerName, "nomic") ||
		   strings.Contains(lowerName, "bge") ||
		   strings.Contains(lowerName, "sentence") {
			
			// Try to generate embedding with this provider
			if embedder, ok := provider.(interface {
				GenerateEmbedding(ctx context.Context, text string) ([]float64, error)
			}); ok {
				embedding, err := embedder.GenerateEmbedding(ctx, text)
				if err == nil {
					return embedding, nil
				}
				// Continue to next provider if this one fails
				continue
			}
		}
	}
	
	// If no embedding provider is found, return an error
	return nil, fmt.Errorf("no embedding provider available: please configure an embedding model (e.g., nomic-embed-text)")
}