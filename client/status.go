package client

import (
	"context"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/utils"
)

// StatusResult represents the result of a status check
type StatusResult struct {
	ProvidersAvailable bool   `json:"providers_available"`
	LLMProvider        string `json:"llm_provider"`
	EmbedderProvider   string `json:"embedder_provider"`
	Error              error  `json:"error,omitempty"`
}

// CheckStatus checks the health and status of the rago client
func (c *Client) CheckStatus() StatusResult {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := StatusResult{
		LLMProvider:      c.config.Providers.DefaultLLM,
		EmbedderProvider: c.config.Providers.DefaultEmbedder,
	}

	// If using legacy config, show that instead
	if c.config.Providers.DefaultLLM == "" {
		result.LLMProvider = "ollama (legacy)"
		result.EmbedderProvider = "ollama (legacy)"
	}

	// Check provider health using the utils function
	if err := utils.CheckProviderHealth(ctx, c.embedder, c.llm); err != nil {
		result.ProvidersAvailable = false
		result.Error = err
	} else {
		result.ProvidersAvailable = true
	}

	return result
}