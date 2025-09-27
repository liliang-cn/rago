package domain

import (
	"context"
	"time"
)

// ProviderType represents different LLM provider types
type ProviderType string

const (
	ProviderOllama   ProviderType = "ollama"
	ProviderOpenAI   ProviderType = "openai"
	ProviderLMStudio ProviderType = "lmstudio"
	ProviderClaude   ProviderType = "claude"
	ProviderGemini   ProviderType = "gemini"
)

// BaseProviderConfig contains common configuration for all providers
type BaseProviderConfig struct {
	Type    ProviderType  `mapstructure:"type"`
	Timeout time.Duration `mapstructure:"timeout"`
}

// OllamaProviderConfig contains Ollama-specific configuration
type OllamaProviderConfig struct {
	BaseProviderConfig `mapstructure:",squash"`
	BaseURL            string `mapstructure:"base_url"`
	EmbeddingModel     string `mapstructure:"embedding_model"`
	LLMModel           string `mapstructure:"llm_model"`
	HideBuiltinThinkTag bool   `mapstructure:"hide_builtin_think_tag"`
}

// OpenAIProviderConfig contains OpenAI-compatible provider configuration
type OpenAIProviderConfig struct {
	BaseProviderConfig `mapstructure:",squash"`
	BaseURL            string `mapstructure:"base_url"`
	APIKey             string `mapstructure:"api_key"`
	EmbeddingModel     string `mapstructure:"embedding_model"`
	LLMModel           string `mapstructure:"llm_model"`
	Organization       string `mapstructure:"organization,omitempty"`
	Project            string `mapstructure:"project,omitempty"`
}

// LMStudioProviderConfig contains LM Studio-specific configuration
type LMStudioProviderConfig struct {
	BaseProviderConfig `mapstructure:",squash"`
	BaseURL            string `mapstructure:"base_url"`
	LLMModel           string `mapstructure:"llm_model"`
	EmbeddingModel     string `mapstructure:"embedding_model"`
	APIKey             string `mapstructure:"api_key,omitempty"` // Optional API key
}

// ClaudeProviderConfig contains Claude (Anthropic) provider configuration
type ClaudeProviderConfig struct {
	BaseProviderConfig `mapstructure:",squash"`
	APIKey             string `mapstructure:"api_key"`
	BaseURL            string `mapstructure:"base_url,omitempty"` // Optional custom endpoint
	LLMModel           string `mapstructure:"llm_model"`
	MaxTokens          int    `mapstructure:"max_tokens,omitempty"`
	AnthropicVersion   string `mapstructure:"anthropic_version,omitempty"` // API version
}

// GeminiProviderConfig contains Google Gemini provider configuration
type GeminiProviderConfig struct {
	BaseProviderConfig `mapstructure:",squash"`
	APIKey             string `mapstructure:"api_key"`
	BaseURL            string `mapstructure:"base_url,omitempty"` // Optional custom endpoint
	LLMModel           string `mapstructure:"llm_model"`
	EmbeddingModel     string `mapstructure:"embedding_model,omitempty"`
	ProjectID          string `mapstructure:"project_id,omitempty"` // For Google Cloud
	Location           string `mapstructure:"location,omitempty"`   // For Google Cloud
}

// ProviderConfig is a union type for provider configurations
type ProviderConfig struct {
	Ollama   *OllamaProviderConfig   `mapstructure:"ollama,omitempty"`
	OpenAI   *OpenAIProviderConfig   `mapstructure:"openai,omitempty"`
	LMStudio *LMStudioProviderConfig `mapstructure:"lmstudio,omitempty"`
	Claude   *ClaudeProviderConfig   `mapstructure:"claude,omitempty"`
	Gemini   *GeminiProviderConfig   `mapstructure:"gemini,omitempty"`
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
