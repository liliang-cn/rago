package domain

import (
	"context"
	"time"
)

// ProviderType represents different LLM provider types
type ProviderType string

const (
	ProviderOpenAI ProviderType = "openai"
)

// BaseProviderConfig contains common configuration for all providers
type BaseProviderConfig struct {
	Type    ProviderType  `mapstructure:"type"`
	Timeout time.Duration `mapstructure:"timeout"`
}

// OpenAIProviderConfig contains OpenAI-compatible provider configuration
// This is used for OpenAI API compatible LLMs including Ollama, LMStudio, etc.
type OpenAIProviderConfig struct {
	BaseProviderConfig `mapstructure:",squash"`
	BaseURL            string `mapstructure:"base_url"`
	APIKey             string `mapstructure:"api_key"`
	EmbeddingModel     string `mapstructure:"embedding_model"`
	LLMModel           string `mapstructure:"llm_model"`
	Organization       string `mapstructure:"organization,omitempty"`
	Project            string `mapstructure:"project,omitempty"`
}

// ProviderConfig is a union type for provider configurations
type ProviderConfig struct {
	OpenAI *OpenAIProviderConfig `mapstructure:"openai,omitempty"`
}

// LLMProvider wraps the Generator interface with provider-specific information
type LLMProvider interface {
	Generator
	ProviderType() ProviderType
	Health(ctx context.Context) error
	ExtractMetadata(ctx context.Context, content string, model string) (*ExtractedMetadata, error)
}

// EmbedderProvider wraps the Embedder interface with provider-specific information
type EmbedderProvider interface {
	Embedder
	ProviderType() ProviderType
	Health(ctx context.Context) error
}

// ProviderFactory interface for creating providers
type ProviderFactory interface {
	CreateLLMProvider(ctx context.Context, config interface{}) (LLMProvider, error)
	CreateEmbedderProvider(ctx context.Context, config interface{}) (EmbedderProvider, error)
}
