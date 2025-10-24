package services

import (
	"context"
	"fmt"

	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/providers"
)

// GetGlobalLLM returns the global LLM service instance
// Convenience function for easy access throughout the application
func GetGlobalLLM() (domain.Generator, error) {
	service := GetGlobalLLMService()
	return service.GetLLMService()
}

// GetGlobalEmbeddingService returns a new embedding service instance
// Convenience function for easy access throughout the application
func GetGlobalEmbeddingService(ctx context.Context) (domain.Embedder, error) {
	service := GetGlobalLLMService()
	return service.GetEmbeddingService(ctx)
}

// GetGlobalFactory returns the provider factory for advanced usage
// Convenience function for easy access throughout the application
func GetGlobalFactory() (*providers.Factory, error) {
	service := GetGlobalLLMService()
	return service.GetFactory()
}

// WithGlobalLLM executes a function with the global LLM service
// Useful for ensuring the LLM service is available and handling errors consistently
func WithGlobalLLM(ctx context.Context, fn func(domain.Generator) error) error {
	llm, err := GetGlobalLLM()
	if err != nil {
		return fmt.Errorf("failed to get global LLM service: %w", err)
	}
	return fn(llm)
}

// WithGlobalLLMWithStats executes a function with the global LLM service and tracks usage
// This version provides usage statistics for monitoring concurrent access
func WithGlobalLLMWithStats(ctx context.Context, fn func(domain.Generator) error) error {
	service := GetGlobalLLMService()
	llm, err := service.GetLLMServiceWithStats()
	if err != nil {
		return fmt.Errorf("failed to get global LLM service with stats: %w", err)
	}
	return fn(llm)
}

// GetGlobalLLMStats returns usage statistics for the global LLM service
func GetGlobalLLMStats() *LLMStats {
	service := GetGlobalLLMService()
	return service.GetStats()
}

// WithGlobalEmbedder executes a function with a new embedding service
// Useful for ensuring the embedder service is available and handling errors consistently
func WithGlobalEmbedder(ctx context.Context, fn func(domain.Embedder) error) error {
	embedder, err := GetGlobalEmbeddingService(ctx)
	if err != nil {
		return fmt.Errorf("failed to get global embedder service: %w", err)
	}
	return fn(embedder)
}

// WithGlobalProviders executes a function with both LLM and embedding services
// Convenience function when both services are needed
func WithGlobalProviders(ctx context.Context, fn func(domain.Generator, domain.Embedder) error) error {
	llm, err := GetGlobalLLM()
	if err != nil {
		return fmt.Errorf("failed to get global LLM service: %w", err)
	}

	embedder, err := GetGlobalEmbeddingService(ctx)
	if err != nil {
		return fmt.Errorf("failed to get global embedder service: %w", err)
	}

	return fn(llm, embedder)
}

// IsGlobalLLMInitialized returns whether the global LLM service has been initialized
// Useful for checking service status before operations
func IsGlobalLLMInitialized() bool {
	service := GetGlobalLLMService()
	return service.IsInitialized()
}