package providers

import (
	"context"
	"testing"

	"github.com/liliang-cn/rago/v2/pkg/domain"
)

func TestNewGeminiProvider(t *testing.T) {
	tests := []struct {
		name    string
		config  *domain.GeminiProviderConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &domain.GeminiProviderConfig{
				APIKey:   "test-key",
				LLMModel: "gemini-pro",
			},
			wantErr: false,
		},
		{
			name: "missing api key",
			config: &domain.GeminiProviderConfig{
				LLMModel: "gemini-pro",
			},
			wantErr: true,
		},
		{
			name: "default model",
			config: &domain.GeminiProviderConfig{
				APIKey: "test-key",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewGeminiProvider(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewGeminiProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && provider == nil {
				t.Error("NewGeminiProvider() returned nil provider")
			}
			if !tt.wantErr && provider.ProviderType() != domain.ProviderGemini {
				t.Errorf("ProviderType() = %v, want %v", provider.ProviderType(), domain.ProviderGemini)
			}
		})
	}
}

func TestGeminiProvider_Generate(t *testing.T) {
	config := &domain.GeminiProviderConfig{
		APIKey:   "test-key",
		LLMModel: "gemini-pro",
	}
	
	provider, err := NewGeminiProvider(config)
	if err != nil {
		t.Fatalf("NewGeminiProvider() error = %v", err)
	}
	
	// This test just verifies the interface is implemented correctly
	// Real API calls would require valid credentials
	ctx := context.Background()
	_, err = provider.Generate(ctx, "test prompt", &domain.GenerationOptions{MaxTokens: 10})
	
	// We expect an error since we don't have real credentials
	if err == nil {
		t.Log("Note: Generate call succeeded - this might be using real credentials")
	}
}