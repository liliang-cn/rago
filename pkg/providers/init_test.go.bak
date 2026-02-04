package providers

import (
	"context"
	"errors"
	"testing"

	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
)

// Mock services for testing
type MockEmbedder struct {
	embedFunc  func(ctx context.Context, text string) ([]float64, error)
	healthFunc func(ctx context.Context) error
}

func (m *MockEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	if m.embedFunc != nil {
		return m.embedFunc(ctx, text)
	}
	return []float64{0.1, 0.2, 0.3}, nil
}

func (m *MockEmbedder) Health(ctx context.Context) error {
	if m.healthFunc != nil {
		return m.healthFunc(ctx)
	}
	return nil
}

type MockGenerator struct {
	generateFunc func(ctx context.Context, prompt string, opts *domain.GenerationOptions) (string, error)
	healthFunc   func(ctx context.Context) error
}

func (m *MockGenerator) Generate(ctx context.Context, prompt string, opts *domain.GenerationOptions) (string, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, prompt, opts)
	}
	return "Generated response", nil
}

func (m *MockGenerator) Stream(ctx context.Context, prompt string, opts *domain.GenerationOptions, callback func(string)) error {
	response, err := m.Generate(ctx, prompt, opts)
	if err != nil {
		return err
	}
	callback(response)
	return nil
}

func (m *MockGenerator) GenerateWithTools(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions) (*domain.GenerationResult, error) {
	return &domain.GenerationResult{
		Content:  "Tool response",
		Finished: true,
	}, nil
}

func (m *MockGenerator) StreamWithTools(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions, callback domain.ToolCallCallback) error {
	return callback("Stream with tools", nil)
}

func (m *MockGenerator) GenerateStructured(ctx context.Context, prompt string, schema interface{}, opts *domain.GenerationOptions) (*domain.StructuredResult, error) {
	return &domain.StructuredResult{
		Data:  map[string]string{"result": "structured"},
		Raw:   `{"result": "structured"}`,
		Valid: true,
	}, nil
}

func (m *MockGenerator) RecognizeIntent(ctx context.Context, request string) (*domain.IntentResult, error) {
	return &domain.IntentResult{
		Intent:     domain.IntentAction,
		Confidence: 0.9,
		NeedsTools: true,
	}, nil
}

func (m *MockGenerator) Health(ctx context.Context) error {
	if m.healthFunc != nil {
		return m.healthFunc(ctx)
	}
	return nil
}

func TestInitializeEmbedder_NoProviderConfigured(t *testing.T) {
	cfg := &config.Config{
		Providers: config.ProvidersConfig{},
	}

	ctx := context.Background()
	var factory *Factory // Not needed for this test

	_, err := InitializeEmbedder(ctx, cfg, factory)
	if err == nil {
		t.Error("Expected error when no embedder provider configured")
	}

	if err.Error() != "no OpenAI provider configuration found" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestInitializeLLM_NoProviderConfigured(t *testing.T) {
	cfg := &config.Config{
		Providers: config.ProvidersConfig{},
	}

	ctx := context.Background()
	var factory *Factory // Not needed for this test

	_, err := InitializeLLM(ctx, cfg, factory)
	if err == nil {
		t.Error("Expected error when no LLM provider configured")
	}

	if err.Error() != "no OpenAI provider configuration found" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestCheckProviderHealth(t *testing.T) {
	testCases := []struct {
		name      string
		embedder  domain.Embedder
		generator domain.Generator
		wantErr   bool
		errMsg    string
	}{
		{
			name: "both services healthy",
			embedder: &MockEmbedder{
				healthFunc: func(ctx context.Context) error {
					return nil
				},
			},
			generator: &MockGenerator{
				healthFunc: func(ctx context.Context) error {
					return nil
				},
			},
			wantErr: false,
		},
		{
			name: "embedder unhealthy",
			embedder: &MockEmbedder{
				healthFunc: func(ctx context.Context) error {
					return errors.New("embedder failed")
				},
			},
			generator: &MockGenerator{
				healthFunc: func(ctx context.Context) error {
					return nil
				},
			},
			wantErr: true,
			errMsg:  "embedder health check failed",
		},
		{
			name: "generator unhealthy",
			embedder: &MockEmbedder{
				healthFunc: func(ctx context.Context) error {
					return nil
				},
			},
			generator: &MockGenerator{
				healthFunc: func(ctx context.Context) error {
					return errors.New("generator failed")
				},
			},
			wantErr: true,
			errMsg:  "LLM health check failed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			err := CheckProviderHealth(ctx, tc.embedder, tc.generator)

			if tc.wantErr && err == nil {
				t.Error("Expected error but got none")
			}

			if !tc.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tc.wantErr && tc.errMsg != "" {
				if err == nil || !contains(err.Error(), tc.errMsg) {
					t.Errorf("Expected error containing %q, got %v", tc.errMsg, err)
				}
			}
		})
	}
}

func TestComposePrompt(t *testing.T) {
	testCases := []struct {
		name   string
		chunks []domain.Chunk
		query  string
		want   string
	}{
		{
			name:   "no chunks",
			chunks: []domain.Chunk{},
			query:  "What is AI?",
			want:   "Please answer the following question:\n\nWhat is AI?",
		},
		{
			name: "single chunk",
			chunks: []domain.Chunk{
				{Content: "AI stands for Artificial Intelligence."},
			},
			query: "What is AI?",
			want: `Based on the following document content, please answer the user's question. If the documents do not contain relevant information, please indicate that you cannot find an answer from the provided documents.

Document Content:
[Document Fragment 1]
AI stands for Artificial Intelligence.

User Question: What is AI?

Please provide a detailed and accurate answer based on the document content:`,
		},
		{
			name: "multiple chunks",
			chunks: []domain.Chunk{
				{Content: "AI stands for Artificial Intelligence."},
				{Content: "Machine learning is a subset of AI."},
			},
			query: "What is AI?",
			want: `Based on the following document content, please answer the user's question. If the documents do not contain relevant information, please indicate that you cannot find an answer from the provided documents.

Document Content:
[Document Fragment 1]
AI stands for Artificial Intelligence.

[Document Fragment 2]
Machine learning is a subset of AI.

User Question: What is AI?

Please provide a detailed and accurate answer based on the document content:`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ComposePrompt(tc.chunks, tc.query)
			if result != tc.want {
				t.Errorf("ComposePrompt() =\n%q\nwant =\n%q", result, tc.want)
			}
		})
	}
}

func TestComposePrompt_EmptyQuery(t *testing.T) {
	chunks := []domain.Chunk{
		{Content: "Test content"},
	}

	result := ComposePrompt(chunks, "")
	if !contains(result, "User Question: ") {
		t.Error("Expected prompt to contain 'User Question:' even with empty query")
	}
}

func TestComposePrompt_EmptyChunkContent(t *testing.T) {
	chunks := []domain.Chunk{
		{Content: ""},
		{Content: "Non-empty content"},
	}

	result := ComposePrompt(chunks, "Test query")
	if !contains(result, "[Document Fragment 1]") {
		t.Error("Expected prompt to contain document fragment markers")
	}

	if !contains(result, "[Document Fragment 2]") {
		t.Error("Expected prompt to contain second document fragment marker")
	}

	if !contains(result, "Non-empty content") {
		t.Error("Expected prompt to contain non-empty content")
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsHelper(s, substr))))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
