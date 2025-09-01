package client

import (
	"context"
	"errors"
	"testing"

	"github.com/liliang-cn/rago/v2/pkg/domain"
)

// mockGenerator is a mock implementation of the domain.Generator interface for testing.
type mockGenerator struct {
	GenerateFunc           func(ctx context.Context, prompt string, opts *domain.GenerationOptions) (string, error)
	StreamFunc             func(ctx context.Context, prompt string, opts *domain.GenerationOptions, callback func(string)) error
	GenerateWithToolsFunc  func(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions) (*domain.GenerationResult, error)
	StreamWithToolsFunc    func(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions, callback domain.ToolCallCallback) error
	GenerateStructuredFunc func(ctx context.Context, prompt string, schema interface{}, opts *domain.GenerationOptions) (*domain.StructuredResult, error)
}

func (m *mockGenerator) Generate(ctx context.Context, prompt string, opts *domain.GenerationOptions) (string, error) {
	if m.GenerateFunc != nil {
		return m.GenerateFunc(ctx, prompt, opts)
	}
	return "", errors.New("Generate not implemented")
}

func (m *mockGenerator) Stream(ctx context.Context, prompt string, opts *domain.GenerationOptions, callback func(string)) error {
	if m.StreamFunc != nil {
		return m.StreamFunc(ctx, prompt, opts, callback)
	}
	return errors.New("Stream not implemented")
}

func (m *mockGenerator) GenerateWithTools(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions) (*domain.GenerationResult, error) {
	if m.GenerateWithToolsFunc != nil {
		return m.GenerateWithToolsFunc(ctx, messages, tools, opts)
	}
	return nil, errors.New("GenerateWithTools not implemented")
}

func (m *mockGenerator) StreamWithTools(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions, callback domain.ToolCallCallback) error {
	if m.StreamWithToolsFunc != nil {
		return m.StreamWithToolsFunc(ctx, messages, tools, opts, callback)
	}
	return errors.New("StreamWithTools not implemented")
}

func (m *mockGenerator) GenerateStructured(ctx context.Context, prompt string, schema interface{}, opts *domain.GenerationOptions) (*domain.StructuredResult, error) {
	if m.GenerateStructuredFunc != nil {
		return m.GenerateStructuredFunc(ctx, prompt, schema, opts)
	}
	return nil, errors.New("GenerateStructured not implemented")
}

func TestLLMGenerate(t *testing.T) {
	client := &Client{
		llm: &mockGenerator{
			GenerateFunc: func(ctx context.Context, prompt string, opts *domain.GenerationOptions) (string, error) {
				if prompt == "hello" {
					return "world", nil
				}
				return "", errors.New("unexpected prompt")
			},
		},
	}

	req := LLMGenerateRequest{Prompt: "hello"}
	resp, err := client.LLMGenerate(context.Background(), req)
	if err != nil {
		t.Fatalf("LLMGenerate failed: %v", err)
	}

	if resp.Content != "world" {
		t.Errorf("Expected content 'world', got '%s'", resp.Content)
	}
}

func TestLLMGenerateStream(t *testing.T) {
	client := &Client{
		llm: &mockGenerator{
			StreamFunc: func(ctx context.Context, prompt string, opts *domain.GenerationOptions, callback func(string)) error {
				if prompt == "hello stream" {
					callback("world")
					callback(" stream")
					return nil
				}
				return errors.New("unexpected prompt")
			},
		},
	}

	req := LLMGenerateRequest{Prompt: "hello stream"}
	var result string
	err := client.LLMGenerateStream(context.Background(), req, func(s string) {
		result += s
	})

	if err != nil {
		t.Fatalf("LLMGenerateStream failed: %v", err)
	}

	if result != "world stream" {
		t.Errorf("Expected content 'world stream', got '%s'", result)
	}
}

func TestLLMChat(t *testing.T) {
	client := &Client{
		llm: &mockGenerator{
			GenerateWithToolsFunc: func(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions) (*domain.GenerationResult, error) {
				if len(messages) == 1 && messages[0].Content == "chat hello" {
					return &domain.GenerationResult{Content: "chat world"}, nil
				}
				return nil, errors.New("unexpected messages")
			},
		},
	}

	req := LLMChatRequest{
		Messages: []ChatMessage{
			{Role: "user", Content: "chat hello"},
		},
	}
	resp, err := client.LLMChat(context.Background(), req)
	if err != nil {
		t.Fatalf("LLMChat failed: %v", err)
	}

	if resp.Content != "chat world" {
		t.Errorf("Expected content 'chat world', got '%s'", resp.Content)
	}
}

func TestLLMChatStream(t *testing.T) {
	client := &Client{
		llm: &mockGenerator{
			StreamWithToolsFunc: func(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions, callback domain.ToolCallCallback) error {
				if len(messages) == 1 && messages[0].Content == "chat stream hello" {
					if err := callback("chat world", nil); err != nil {
						return err
					}
					if err := callback(" stream", nil); err != nil {
						return err
					}
					return nil
				}
				return errors.New("unexpected messages")
			},
		},
	}

	req := LLMChatRequest{
		Messages: []ChatMessage{
			{Role: "user", Content: "chat stream hello"},
		},
	}
	var result string
		err := client.LLMChatStream(context.Background(), req, func(s string) {
		result += s
	})

	if err != nil {
		t.Fatalf("LLMChatStream failed: %v", err)
	}

	if result != "chat world stream" {
		t.Errorf("Expected content 'chat world stream', got '%s'", result)
	}
}

func TestLLMGenerate_Error(t *testing.T) {
	client := &Client{
		llm: &mockGenerator{
			GenerateFunc: func(ctx context.Context, prompt string, opts *domain.GenerationOptions) (string, error) {
				return "", errors.New("generate error")
			},
		},
	}

	req := LLMGenerateRequest{Prompt: "hello"}
	_, err := client.LLMGenerate(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error but got none")
	}
	if err.Error() != "LLM generation failed: generate error" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestLLMGenerateStream_Error(t *testing.T) {
	client := &Client{
		llm: &mockGenerator{
			StreamFunc: func(ctx context.Context, prompt string, opts *domain.GenerationOptions, callback func(string)) error {
				return errors.New("stream error")
			},
		},
	}

	req := LLMGenerateRequest{Prompt: "hello"}
	err := client.LLMGenerateStream(context.Background(), req, func(s string) {})
	if err == nil {
		t.Fatal("Expected error but got none")
	}
	if err.Error() != "stream error" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestLLMChat_Error(t *testing.T) {
	client := &Client{
		llm: &mockGenerator{
			GenerateWithToolsFunc: func(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions) (*domain.GenerationResult, error) {
				return nil, errors.New("chat error")
			},
		},
	}

	req := LLMChatRequest{}
	_, err := client.LLMChat(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error but got none")
	}
	if err.Error() != "LLM chat failed: chat error" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestLLMChatStream_Error(t *testing.T) {
	client := &Client{
		llm: &mockGenerator{
			StreamWithToolsFunc: func(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions, callback domain.ToolCallCallback) error {
				return errors.New("chat stream error")
			},
		},
	}

	req := LLMChatRequest{}
	err := client.LLMChatStream(context.Background(), req, func(s string) {})
	if err == nil {
		t.Fatal("Expected error but got none")
	}
	if err.Error() != "chat stream error" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestLLMGenerate_NoLLM(t *testing.T) {
	client := &Client{}
	req := LLMGenerateRequest{Prompt: "hello"}
	_, err := client.LLMGenerate(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error but got none")
	}
	if err.Error() != "LLM service not initialized" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestLLMGenerateStream_NoLLM(t *testing.T) {
	client := &Client{}
	req := LLMGenerateRequest{Prompt: "hello"}
	err := client.LLMGenerateStream(context.Background(), req, func(s string) {})
	if err == nil {
		t.Fatal("Expected error but got none")
	}
	if err.Error() != "LLM service not initialized" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestLLMChat_NoLLM(t *testing.T) {
	client := &Client{}
	req := LLMChatRequest{}
	_, err := client.LLMChat(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error but got none")
	}
	if err.Error() != "LLM service not initialized" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestLLMChatStream_NoLLM(t *testing.T) {
	client := &Client{}
	req := LLMChatRequest{}
	err := client.LLMChatStream(context.Background(), req, func(s string) {})
	if err == nil {
		t.Fatal("Expected error but got none")
	}
	if err.Error() != "LLM service not initialized" {
		t.Errorf("Unexpected error message: %v", err)
	}
}
