package providers

import (
	"context"
	"testing"

	"github.com/liliang-cn/rago/v2/pkg/domain"
)

func TestNewClaudeProvider(t *testing.T) {
	tests := []struct {
		name    string
		config  *domain.ClaudeProviderConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &domain.ClaudeProviderConfig{
				APIKey:   "test-key",
				LLMModel: "claude-3-sonnet-20240229",
			},
			wantErr: false,
		},
		{
			name: "missing api key",
			config: &domain.ClaudeProviderConfig{
				LLMModel: "claude-3-sonnet-20240229",
			},
			wantErr: true,
		},
		{
			name: "default model",
			config: &domain.ClaudeProviderConfig{
				APIKey: "test-key",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewClaudeProvider(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClaudeProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && provider == nil {
				t.Error("NewClaudeProvider() returned nil provider")
			}
			if !tt.wantErr && provider.ProviderType() != domain.ProviderClaude {
				t.Errorf("ProviderType() = %v, want %v", provider.ProviderType(), domain.ProviderClaude)
			}
		})
	}
}

func TestClaudeProvider_Generate(t *testing.T) {
	config := &domain.ClaudeProviderConfig{
		APIKey:   "test-key",
		LLMModel: "claude-3-sonnet-20240229",
	}
	
	provider, err := NewClaudeProvider(config)
	if err != nil {
		t.Fatalf("NewClaudeProvider() error = %v", err)
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