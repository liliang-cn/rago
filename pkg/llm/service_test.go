package llm

import (
	"context"
	"errors"
	"testing"

	"github.com/liliang-cn/rago/v2/pkg/domain"
)

// Mock LLM provider for testing
type MockLLMProvider struct {
	generateFunc           func(ctx context.Context, prompt string, opts *domain.GenerationOptions) (string, error)
	streamFunc             func(ctx context.Context, prompt string, opts *domain.GenerationOptions, callback func(string)) error
	generateWithToolsFunc  func(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions) (*domain.GenerationResult, error)
	streamWithToolsFunc    func(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions, callback domain.ToolCallCallback) error
	generateStructuredFunc func(ctx context.Context, prompt string, schema interface{}, opts *domain.GenerationOptions) (*domain.StructuredResult, error)
	extractMetadataFunc    func(ctx context.Context, content string, model string) (*domain.ExtractedMetadata, error)
	healthFunc             func(ctx context.Context) error
	providerType           domain.ProviderType
}

func (m *MockLLMProvider) Generate(ctx context.Context, prompt string, opts *domain.GenerationOptions) (string, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, prompt, opts)
	}
	return "Generated response", nil
}

func (m *MockLLMProvider) Stream(ctx context.Context, prompt string, opts *domain.GenerationOptions, callback func(string)) error {
	if m.streamFunc != nil {
		return m.streamFunc(ctx, prompt, opts, callback)
	}
	callback("Streamed response")
	return nil
}

func (m *MockLLMProvider) GenerateWithTools(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions) (*domain.GenerationResult, error) {
	if m.generateWithToolsFunc != nil {
		return m.generateWithToolsFunc(ctx, messages, tools, opts)
	}
	return &domain.GenerationResult{
		Content:  "Response with tools",
		Finished: true,
	}, nil
}

func (m *MockLLMProvider) StreamWithTools(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions, callback domain.ToolCallCallback) error {
	if m.streamWithToolsFunc != nil {
		return m.streamWithToolsFunc(ctx, messages, tools, opts, callback)
	}
	return callback("Streamed with tools", nil)
}

func (m *MockLLMProvider) GenerateStructured(ctx context.Context, prompt string, schema interface{}, opts *domain.GenerationOptions) (*domain.StructuredResult, error) {
	if m.generateStructuredFunc != nil {
		return m.generateStructuredFunc(ctx, prompt, schema, opts)
	}
	return &domain.StructuredResult{
		Data:  map[string]string{"result": "structured"},
		Raw:   `{"result": "structured"}`,
		Valid: true,
	}, nil
}

func (m *MockLLMProvider) ExtractMetadata(ctx context.Context, content string, model string) (*domain.ExtractedMetadata, error) {
	if m.extractMetadataFunc != nil {
		return m.extractMetadataFunc(ctx, content, model)
	}
	return &domain.ExtractedMetadata{
		Summary:      "Extracted summary",
		Keywords:     []string{"test", "metadata"},
		DocumentType: "text",
	}, nil
}

func (m *MockLLMProvider) Health(ctx context.Context) error {
	if m.healthFunc != nil {
		return m.healthFunc(ctx)
	}
	return nil
}

func (m *MockLLMProvider) ProviderType() domain.ProviderType {
	return m.providerType
}

func TestNewService(t *testing.T) {
	provider := &MockLLMProvider{
		providerType: domain.ProviderOpenAI,
	}

	service := NewService(provider)
	if service == nil {
		t.Fatal("NewService returned nil")
	}

	if service.provider != provider {
		t.Error("Service provider not set correctly")
	}
}

func TestService_Generate(t *testing.T) {
	testCases := []struct {
		name     string
		provider *MockLLMProvider
		prompt   string
		expected string
		wantErr  bool
	}{
		{
			name: "successful generation",
			provider: &MockLLMProvider{
				generateFunc: func(ctx context.Context, prompt string, opts *domain.GenerationOptions) (string, error) {
					if prompt == "test prompt" {
						return "test response", nil
					}
					return "default response", nil
				},
			},
			prompt:   "test prompt",
			expected: "test response",
			wantErr:  false,
		},
		{
			name: "generation error",
			provider: &MockLLMProvider{
				generateFunc: func(ctx context.Context, prompt string, opts *domain.GenerationOptions) (string, error) {
					return "", errors.New("generation failed")
				},
			},
			prompt:   "fail prompt",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := NewService(tc.provider)
			ctx := context.Background()

			response, err := service.Generate(ctx, tc.prompt, nil)

			if tc.wantErr && err == nil {
				t.Error("Expected error but got none")
			}

			if !tc.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if response != tc.expected {
				t.Errorf("Expected response %q, got %q", tc.expected, response)
			}
		})
	}
}

func TestService_Stream(t *testing.T) {
	provider := &MockLLMProvider{
		streamFunc: func(ctx context.Context, prompt string, opts *domain.GenerationOptions, callback func(string)) error {
			callback("chunk 1")
			callback("chunk 2")
			return nil
		},
	}

	service := NewService(provider)
	ctx := context.Background()

	var chunks []string
	callback := func(chunk string) {
		chunks = append(chunks, chunk)
	}

	err := service.Stream(ctx, "test prompt", nil, callback)
	if err != nil {
		t.Errorf("Stream failed: %v", err)
	}

	if len(chunks) != 2 {
		t.Errorf("Expected 2 chunks, got %d", len(chunks))
	}

	if chunks[0] != "chunk 1" || chunks[1] != "chunk 2" {
		t.Errorf("Unexpected chunks: %v", chunks)
	}
}

func TestService_GenerateWithTools(t *testing.T) {
	provider := &MockLLMProvider{
		generateWithToolsFunc: func(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions) (*domain.GenerationResult, error) {
			return &domain.GenerationResult{
				Content:  "Tool-enhanced response",
				Finished: true,
			}, nil
		},
	}

	service := NewService(provider)
	ctx := context.Background()

	messages := []domain.Message{{Role: "user", Content: "test message"}}
	tools := []domain.ToolDefinition{{Type: "function"}}

	result, err := service.GenerateWithTools(ctx, messages, tools, nil)
	if err != nil {
		t.Errorf("GenerateWithTools failed: %v", err)
	}

	if result == nil {
		t.Fatal("GenerateWithTools returned nil result")
	}

	if result.Content != "Tool-enhanced response" {
		t.Errorf("Expected 'Tool-enhanced response', got %q", result.Content)
	}

	if !result.Finished {
		t.Error("Expected result to be finished")
	}
}

func TestService_GenerateStructured(t *testing.T) {
	provider := &MockLLMProvider{
		generateStructuredFunc: func(ctx context.Context, prompt string, schema interface{}, opts *domain.GenerationOptions) (*domain.StructuredResult, error) {
			return &domain.StructuredResult{
				Data:  map[string]interface{}{"key": "value"},
				Raw:   `{"key": "value"}`,
				Valid: true,
			}, nil
		},
	}

	service := NewService(provider)
	ctx := context.Background()

	schema := map[string]interface{}{"type": "object"}

	result, err := service.GenerateStructured(ctx, "test prompt", schema, nil)
	if err != nil {
		t.Errorf("GenerateStructured failed: %v", err)
	}

	if result == nil {
		t.Fatal("GenerateStructured returned nil result")
	}

	if !result.Valid {
		t.Error("Expected valid structured result")
	}

	if result.Raw != `{"key": "value"}` {
		t.Errorf("Expected raw JSON, got %q", result.Raw)
	}
}

func TestService_ExtractMetadata(t *testing.T) {
	provider := &MockLLMProvider{
		extractMetadataFunc: func(ctx context.Context, content string, model string) (*domain.ExtractedMetadata, error) {
			return &domain.ExtractedMetadata{
				Summary:      "Test summary",
				Keywords:     []string{"keyword1", "keyword2"},
				DocumentType: "article",
			}, nil
		},
	}

	service := NewService(provider)
	ctx := context.Background()

	metadata, err := service.ExtractMetadata(ctx, "test content", "test-model")
	if err != nil {
		t.Errorf("ExtractMetadata failed: %v", err)
	}

	if metadata == nil {
		t.Fatal("ExtractMetadata returned nil")
	}

	if metadata.Summary != "Test summary" {
		t.Errorf("Expected 'Test summary', got %q", metadata.Summary)
	}

	if len(metadata.Keywords) != 2 {
		t.Errorf("Expected 2 keywords, got %d", len(metadata.Keywords))
	}

	if metadata.DocumentType != "article" {
		t.Errorf("Expected 'article', got %q", metadata.DocumentType)
	}
}

func TestService_Health(t *testing.T) {
	testCases := []struct {
		name     string
		provider *MockLLMProvider
		wantErr  bool
	}{
		{
			name: "healthy provider",
			provider: &MockLLMProvider{
				healthFunc: func(ctx context.Context) error {
					return nil
				},
			},
			wantErr: false,
		},
		{
			name: "unhealthy provider",
			provider: &MockLLMProvider{
				healthFunc: func(ctx context.Context) error {
					return errors.New("provider unhealthy")
				},
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := NewService(tc.provider)
			ctx := context.Background()

			err := service.Health(ctx)

			if tc.wantErr && err == nil {
				t.Error("Expected error but got none")
			}

			if !tc.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestService_ProviderType(t *testing.T) {
	testCases := []struct {
		name         string
		providerType domain.ProviderType
	}{
		{
			name:         "ollama provider",
			providerType: domain.ProviderOllama,
		},
		{
			name:         "openai provider",
			providerType: domain.ProviderOpenAI,
		},
		{
			name:         "lmstudio provider",
			providerType: domain.ProviderLMStudio,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			provider := &MockLLMProvider{
				providerType: tc.providerType,
			}

			service := NewService(provider)
			providerType := service.ProviderType()

			if providerType != tc.providerType {
				t.Errorf("Expected provider type %s, got %s", tc.providerType, providerType)
			}
		})
	}
}
