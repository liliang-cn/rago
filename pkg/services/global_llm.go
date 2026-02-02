package services

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/providers"
	"github.com/liliang-cn/rago/v2/pkg/usage"
)

// GlobalLLMService manages a single LLM instance for the entire application
type GlobalLLMService struct {
	config     *config.Config
	factory    *providers.Factory
	llmService domain.Generator

	// Thread safety
	mu        sync.RWMutex
	initialized bool

	// Usage statistics
	requestCount int64
	activeRequests int64
}

var (
	globalLLMService *GlobalLLMService
	globalLLMMu      sync.RWMutex
)

// GetGlobalLLMService returns the singleton instance of GlobalLLMService
func GetGlobalLLMService() *GlobalLLMService {
	globalLLMMu.RLock()
	if globalLLMService != nil {
		globalLLMMu.RUnlock()
		return globalLLMService
	}
	globalLLMMu.RUnlock()

	globalLLMMu.Lock()
	defer globalLLMMu.Unlock()

	// Double-check after acquiring write lock
	if globalLLMService != nil {
		return globalLLMService
	}

	globalLLMService = &GlobalLLMService{}
	return globalLLMService
}

// Initialize initializes the global LLM service with configuration
func (g *GlobalLLMService) Initialize(ctx context.Context, cfg *config.Config) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.initialized {
		return nil
	}

	g.config = cfg
	g.factory = providers.NewFactory()

	// Create LLM service using existing providers.InitializeLLM function
	llmService, err := providers.InitializeLLM(ctx, cfg, g.factory)
	if err != nil {
		return fmt.Errorf("failed to initialize global LLM service: %w", err)
	}

	// Create usage service for token tracking
	usageService, err := usage.NewService(cfg)
	if err != nil {
		return fmt.Errorf("failed to create usage service: %w", err)
	}

	// Wrap LLM provider with usage tracking
	trackedProvider := providers.NewTrackedLLMProvider(llmService.(domain.LLMProvider), usageService)

	g.llmService = trackedProvider
	g.initialized = true

	return nil
}

// GetLLMService returns the LLM service instance
func (g *GlobalLLMService) GetLLMService() (domain.Generator, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if !g.initialized {
		return nil, fmt.Errorf("global LLM service not initialized")
	}

	return g.llmService, nil
}

// GetLLMServiceWithStats returns the LLM service instance with usage tracking
func (g *GlobalLLMService) GetLLMServiceWithStats() (domain.Generator, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if !g.initialized {
		return nil, fmt.Errorf("global LLM service not initialized")
	}

	// Increment counters
	atomic.AddInt64(&g.requestCount, 1)
	atomic.AddInt64(&g.activeRequests, 1)

	// Return a wrapper that decrements active requests when done
	return &LLMServiceWrapper{
		Generator:     g.llmService,
		globalService: g,
	}, nil
}

// GetStats returns usage statistics
func (g *GlobalLLMService) GetStats() *LLMStats {
	return &LLMStats{
		TotalRequests:   atomic.LoadInt64(&g.requestCount),
		ActiveRequests:  atomic.LoadInt64(&g.activeRequests),
		Initialized:     g.initialized,
	}
}

// LLMStats represents LLM service usage statistics
type LLMStats struct {
	TotalRequests  int64 `json:"total_requests"`
	ActiveRequests int64 `json:"active_requests"`
	Initialized    bool  `json:"initialized"`
}

// LLMServiceWrapper wraps the LLM service to track usage
type LLMServiceWrapper struct {
	domain.Generator
	globalService *GlobalLLMService
}

// Generate wraps the original Generate method with usage tracking
func (w *LLMServiceWrapper) Generate(ctx context.Context, prompt string, opts *domain.GenerationOptions) (string, error) {
	defer atomic.AddInt64(&w.globalService.activeRequests, -1)
	return w.Generator.Generate(ctx, prompt, opts)
}

// Stream wraps the original Stream method with usage tracking
func (w *LLMServiceWrapper) Stream(ctx context.Context, prompt string, opts *domain.GenerationOptions, callback func(string)) error {
	defer atomic.AddInt64(&w.globalService.activeRequests, -1)
	return w.Generator.Stream(ctx, prompt, opts, callback)
}

// GenerateWithTools wraps the original GenerateWithTools method with usage tracking
func (w *LLMServiceWrapper) GenerateWithTools(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions) (*domain.GenerationResult, error) {
	defer atomic.AddInt64(&w.globalService.activeRequests, -1)
	return w.Generator.GenerateWithTools(ctx, messages, tools, opts)
}

// StreamWithTools wraps the original StreamWithTools method with usage tracking
func (w *LLMServiceWrapper) StreamWithTools(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions, callback domain.ToolCallCallback) error {
	defer atomic.AddInt64(&w.globalService.activeRequests, -1)
	return w.Generator.StreamWithTools(ctx, messages, tools, opts, callback)
}

// GenerateStructured wraps the original GenerateStructured method with usage tracking
func (w *LLMServiceWrapper) GenerateStructured(ctx context.Context, prompt string, schema interface{}, opts *domain.GenerationOptions) (*domain.StructuredResult, error) {
	defer atomic.AddInt64(&w.globalService.activeRequests, -1)
	return w.Generator.GenerateStructured(ctx, prompt, schema, opts)
}

// RecognizeIntent wraps the original RecognizeIntent method with usage tracking
func (w *LLMServiceWrapper) RecognizeIntent(ctx context.Context, request string) (*domain.IntentResult, error) {
	defer atomic.AddInt64(&w.globalService.activeRequests, -1)
	return w.Generator.RecognizeIntent(ctx, request)
}

// GetEmbeddingService returns a new embedding service instance
// Note: Embedding services are created per request since they may have different configurations
func (g *GlobalLLMService) GetEmbeddingService(ctx context.Context) (domain.Embedder, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if !g.initialized {
		return nil, fmt.Errorf("global LLM service not initialized")
	}

	// Create embedder service using factory (with custom providers support)
	embedderConfig, err := providers.GetEmbedderProviderConfigWithCustom(&g.config.Providers.ProviderConfigs, g.config.Providers.DefaultEmbedder, g.config.Providers.Providers)
	if err != nil {
		return nil, fmt.Errorf("failed to get embedder config: %w", err)
	}

	embedder, err := g.factory.CreateEmbedderProvider(ctx, embedderConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	return embedder, nil
}

// GetFactory returns the provider factory for advanced usage
func (g *GlobalLLMService) GetFactory() (*providers.Factory, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if !g.initialized {
		return nil, fmt.Errorf("global LLM service not initialized")
	}

	return g.factory, nil
}

// Close closes the global LLM service
func (g *GlobalLLMService) Close() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if !g.initialized {
		return nil
	}

	// Close LLM service if it implements a Close method
	if closer, ok := g.llmService.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			return fmt.Errorf("failed to close LLM service: %w", err)
		}
	}

	g.llmService = nil
	g.factory = nil
	g.config = nil
	g.initialized = false

	return nil
}

// Shutdown shuts down the global LLM service and cleans up the singleton
func (g *GlobalLLMService) Shutdown() error {
	globalLLMMu.Lock()
	defer globalLLMMu.Unlock()

	if err := g.Close(); err != nil {
		return err
	}

	globalLLMService = nil
	return nil
}

// IsInitialized returns whether the global LLM service has been initialized
func (g *GlobalLLMService) IsInitialized() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.initialized
}

// Reinitialize reinitializes the global LLM service with new configuration
func (g *GlobalLLMService) Reinitialize(ctx context.Context, cfg *config.Config) error {
	// Close existing service
	if err := g.Close(); err != nil {
		return fmt.Errorf("failed to close existing service: %w", err)
	}

	// Initialize with new configuration
	return g.Initialize(ctx, cfg)
}