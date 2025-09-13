package providers

import (
	"context"
	"fmt"

	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
)

// StatusChecker provides provider health and status checking
type StatusChecker struct {
	config   *config.Config
	embedder domain.Embedder
	llm      domain.Generator
}

// NewStatusChecker creates a new status checker
func NewStatusChecker(cfg *config.Config, embedder domain.Embedder, llm domain.Generator) *StatusChecker {
	return &StatusChecker{
		config:   cfg,
		embedder: embedder,
		llm:      llm,
	}
}

// ProviderHealthStatus represents the health status of a provider
type ProviderHealthStatus struct {
	Type        string                 `json:"type"`
	Provider    string                 `json:"provider"`
	Healthy     bool                   `json:"healthy"`
	Error       string                 `json:"error,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// CheckAll checks the status of all configured providers
func (s *StatusChecker) CheckAll(ctx context.Context) (*Status, error) {
	status := &Status{
		Providers: []ProviderHealthStatus{},
	}

	// Check LLM provider
	if s.llm != nil {
		llmStatus := s.checkLLMProvider(ctx)
		status.Providers = append(status.Providers, llmStatus)
		if llmStatus.Healthy {
			status.LLMHealthy = true
		}
	}

	// Check Embedder provider
	if s.embedder != nil {
		embedderStatus := s.checkEmbedderProvider(ctx)
		status.Providers = append(status.Providers, embedderStatus)
		if embedderStatus.Healthy {
			status.EmbedderHealthy = true
		}
	}

	// Overall health
	status.Healthy = status.LLMHealthy && status.EmbedderHealthy

	return status, nil
}

// Status represents the overall system status
type Status struct {
	Healthy         bool             `json:"healthy"`
	LLMHealthy      bool             `json:"llm_healthy"`
	EmbedderHealthy bool             `json:"embedder_healthy"`
	Providers       []ProviderHealthStatus `json:"providers"`
}

// checkLLMProvider checks the LLM provider status
func (s *StatusChecker) checkLLMProvider(ctx context.Context) ProviderHealthStatus {
	status := ProviderHealthStatus{
		Type:     "llm",
		Provider: string(s.config.Providers.DefaultLLM),
		Details:  make(map[string]interface{}),
	}

	// Get provider details
	if s.config.Providers.ProviderConfigs.Ollama != nil && s.config.Providers.DefaultLLM == "ollama" {
		cfg := s.config.Providers.ProviderConfigs.Ollama
		status.Details["base_url"] = cfg.BaseURL
		status.Details["model"] = cfg.LLMModel
		status.Details["timeout"] = cfg.Timeout.String()
	} else if s.config.Providers.ProviderConfigs.OpenAI != nil && s.config.Providers.DefaultLLM == "openai" {
		cfg := s.config.Providers.ProviderConfigs.OpenAI
		status.Details["base_url"] = cfg.BaseURL
		status.Details["model"] = cfg.LLMModel
		status.Details["timeout"] = cfg.Timeout.String()
	} else if s.config.Providers.ProviderConfigs.LMStudio != nil && s.config.Providers.DefaultLLM == "lmstudio" {
		cfg := s.config.Providers.ProviderConfigs.LMStudio
		status.Details["base_url"] = cfg.BaseURL
		status.Details["model"] = cfg.LLMModel
		status.Details["timeout"] = cfg.Timeout.String()
	}

	// Check health
	if healthChecker, ok := s.llm.(interface{ Health(context.Context) error }); ok {
		if err := healthChecker.Health(ctx); err != nil {
			status.Healthy = false
			status.Error = fmt.Sprintf("health check failed: %v", err)
		} else {
			status.Healthy = true
		}
	} else {
		// No health check available, assume healthy
		status.Healthy = true
	}

	return status
}

// checkEmbedderProvider checks the embedder provider status
func (s *StatusChecker) checkEmbedderProvider(ctx context.Context) ProviderHealthStatus {
	status := ProviderHealthStatus{
		Type:     "embedder",
		Provider: string(s.config.Providers.DefaultEmbedder),
		Details:  make(map[string]interface{}),
	}

	// Get provider details
	if s.config.Providers.ProviderConfigs.Ollama != nil && s.config.Providers.DefaultEmbedder == "ollama" {
		cfg := s.config.Providers.ProviderConfigs.Ollama
		status.Details["base_url"] = cfg.BaseURL
		status.Details["model"] = cfg.EmbeddingModel
		status.Details["timeout"] = cfg.Timeout.String()
	} else if s.config.Providers.ProviderConfigs.OpenAI != nil && s.config.Providers.DefaultEmbedder == "openai" {
		cfg := s.config.Providers.ProviderConfigs.OpenAI
		status.Details["base_url"] = cfg.BaseURL
		status.Details["model"] = cfg.EmbeddingModel
		status.Details["timeout"] = cfg.Timeout.String()
	} else if s.config.Providers.ProviderConfigs.LMStudio != nil && s.config.Providers.DefaultEmbedder == "lmstudio" {
		cfg := s.config.Providers.ProviderConfigs.LMStudio
		status.Details["base_url"] = cfg.BaseURL
		status.Details["model"] = cfg.EmbeddingModel
		status.Details["timeout"] = cfg.Timeout.String()
	}

	// Check health
	if healthChecker, ok := s.embedder.(interface{ Health(context.Context) error }); ok {
		if err := healthChecker.Health(ctx); err != nil {
			status.Healthy = false
			status.Error = fmt.Sprintf("health check failed: %v", err)
		} else {
			status.Healthy = true
		}
	} else {
		// No health check available, assume healthy
		status.Healthy = true
	}

	return status
}

// CheckLLM checks only the LLM provider
func (s *StatusChecker) CheckLLM(ctx context.Context) (bool, error) {
	if s.llm == nil {
		return false, fmt.Errorf("no LLM provider configured")
	}

	if healthChecker, ok := s.llm.(interface{ Health(context.Context) error }); ok {
		if err := healthChecker.Health(ctx); err != nil {
			return false, err
		}
	}

	return true, nil
}

// CheckEmbedder checks only the embedder provider
func (s *StatusChecker) CheckEmbedder(ctx context.Context) (bool, error) {
	if s.embedder == nil {
		return false, fmt.Errorf("no embedder provider configured")
	}

	if healthChecker, ok := s.embedder.(interface{ Health(context.Context) error }); ok {
		if err := healthChecker.Health(ctx); err != nil {
			return false, err
		}
	}

	return true, nil
}