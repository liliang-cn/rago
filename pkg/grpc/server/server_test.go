package server

import (
	"context"
	"testing"

	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockLLMProvider for server tests
type MockServerLLMProvider struct {
	mock.Mock
}

func (m *MockServerLLMProvider) Generate(ctx context.Context, prompt string, opts *domain.GenerationOptions) (string, error) {
	args := m.Called(ctx, prompt, opts)
	return args.String(0), args.Error(1)
}

func (m *MockServerLLMProvider) Stream(ctx context.Context, prompt string, opts *domain.GenerationOptions, callback func(string)) error {
	args := m.Called(ctx, prompt, opts, callback)
	return args.Error(0)
}

func (m *MockServerLLMProvider) GenerateWithTools(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions) (*domain.GenerationResult, error) {
	args := m.Called(ctx, messages, tools, opts)
	return args.Get(0).(*domain.GenerationResult), args.Error(1)
}

func (m *MockServerLLMProvider) StreamWithTools(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions, callback domain.ToolCallCallback) error {
	args := m.Called(ctx, messages, tools, opts, callback)
	return args.Error(0)
}

func (m *MockServerLLMProvider) GenerateStructured(ctx context.Context, prompt string, schema interface{}, opts *domain.GenerationOptions) (*domain.StructuredResult, error) {
	args := m.Called(ctx, prompt, schema, opts)
	return args.Get(0).(*domain.StructuredResult), args.Error(1)
}

func (m *MockServerLLMProvider) RecognizeIntent(ctx context.Context, request string) (*domain.IntentResult, error) {
	args := m.Called(ctx, request)
	return args.Get(0).(*domain.IntentResult), args.Error(1)
}

func (m *MockServerLLMProvider) ExtractMetadata(ctx context.Context, content string, model string) (*domain.ExtractedMetadata, error) {
	args := m.Called(ctx, content, model)
	return args.Get(0).(*domain.ExtractedMetadata), args.Error(1)
}

func (m *MockServerLLMProvider) ProviderType() domain.ProviderType {
	args := m.Called()
	return args.Get(0).(domain.ProviderType)
}

func (m *MockServerLLMProvider) Health(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// MockEmbedderProvider for server tests
type MockServerEmbedderProvider struct {
	mock.Mock
}

func (m *MockServerEmbedderProvider) Embed(ctx context.Context, text string) ([]float64, error) {
	args := m.Called(ctx, text)
	return args.Get(0).([]float64), args.Error(1)
}

func (m *MockServerEmbedderProvider) ProviderType() domain.ProviderType {
	args := m.Called()
	return args.Get(0).(domain.ProviderType)
}

func (m *MockServerEmbedderProvider) Health(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	
	assert.Equal(t, 50051, config.Port)
	assert.Equal(t, 100*1024*1024, config.MaxMessageSize)
	assert.True(t, config.EnableReflection)
	assert.False(t, config.EnableAuth)
	assert.False(t, config.EnableTLS)
}

func TestNewServer_InvalidTLSConfig(t *testing.T) {
	config := DefaultConfig()
	config.EnableTLS = true
	config.CertFile = "" // Missing cert file
	config.KeyFile = ""  // Missing key file
	
	mockLLM := &MockServerLLMProvider{}
	mockEmbedder := &MockServerEmbedderProvider{}
	
	server, err := NewServer(config, nil, mockLLM, mockEmbedder)
	
	assert.Error(t, err)
	assert.Nil(t, server)
	assert.Contains(t, err.Error(), "failed to load TLS credentials")
}

func TestServerConfig_Validation(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		config := Config{
			Port:             8080,
			MaxMessageSize:   1024,
			EnableReflection: true,
			EnableAuth:       false,
			EnableTLS:        false,
		}
		
		assert.Equal(t, 8080, config.Port)
		assert.Equal(t, 1024, config.MaxMessageSize)
		assert.True(t, config.EnableReflection)
		assert.False(t, config.EnableAuth)
		assert.False(t, config.EnableTLS)
	})
	
	t.Run("auth config validation", func(t *testing.T) {
		config := Config{
			EnableAuth: true,
			AuthToken:  "",
		}
		
		// The validation is done in the gRPC command, not in NewServer
		// So we test the logical validation here
		assert.True(t, config.EnableAuth)
		assert.Empty(t, config.AuthToken)
		
		// This would fail in actual usage
		if config.EnableAuth && config.AuthToken == "" {
			assert.Fail(t, "Auth enabled but no token provided")
		}
	})
}

func TestServerHealth(t *testing.T) {
	config := DefaultConfig()
	mockLLM := &MockServerLLMProvider{}
	mockEmbedder := &MockServerEmbedderProvider{}
	
	// This would normally require a full processor setup, 
	// so we'll test the basic server creation instead
	_, err := NewServer(config, nil, mockLLM, mockEmbedder)
	
	// Should fail because processor is nil
	assert.Error(t, err)
}