package client

import (
	"context"
	"time"
)

// StatusResult represents the result of a status check
type StatusResult struct {
	ProvidersAvailable bool   `json:"providers_available"`
	LLMProvider        string `json:"llm_provider"`
	EmbedderProvider   string `json:"embedder_provider"`
	Error              error  `json:"error,omitempty"`
}

// CheckStatus checks the health and status of the rago client (delegated to status checker)
func (c *BaseClient) CheckStatus() StatusResult {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := StatusResult{
		LLMProvider:      string(c.config.Providers.DefaultLLM),
		EmbedderProvider: string(c.config.Providers.DefaultEmbedder),
	}

	// Use the status checker service
	if c.statusChecker != nil {
		status, err := c.statusChecker.CheckAll(ctx)
		if err != nil {
			result.ProvidersAvailable = false
			result.Error = err
		} else {
			result.ProvidersAvailable = status.Healthy
		}
	} else {
		// Fallback if status checker not available
		result.ProvidersAvailable = c.llm != nil && c.embedder != nil
	}

	return result
}
