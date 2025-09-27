package server

import (
	"context"
	"testing"

	pb "github.com/liliang-cn/rago/v2/proto/rago"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

// MockLLMProvider implements the LLM provider interface for testing
type MockLLMProvider struct {
	mock.Mock
}

func (m *MockLLMProvider) Generate(ctx context.Context, prompt string, opts *domain.GenerationOptions) (string, error) {
	args := m.Called(ctx, prompt, opts)
	return args.String(0), args.Error(1)
}

func (m *MockLLMProvider) Stream(ctx context.Context, prompt string, opts *domain.GenerationOptions, callback func(string)) error {
	args := m.Called(ctx, prompt, opts, callback)
	if callback != nil {
		callback("chunk1")
		callback("chunk2")
	}
	return args.Error(0)
}

func (m *MockLLMProvider) GenerateWithTools(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions) (*domain.GenerationResult, error) {
	args := m.Called(ctx, messages, tools, opts)
	return args.Get(0).(*domain.GenerationResult), args.Error(1)
}

func (m *MockLLMProvider) StreamWithTools(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions, callback domain.ToolCallCallback) error {
	args := m.Called(ctx, messages, tools, opts, callback)
	return args.Error(0)
}

func (m *MockLLMProvider) GenerateStructured(ctx context.Context, prompt string, schema interface{}, opts *domain.GenerationOptions) (*domain.StructuredResult, error) {
	args := m.Called(ctx, prompt, schema, opts)
	return args.Get(0).(*domain.StructuredResult), args.Error(1)
}

func (m *MockLLMProvider) RecognizeIntent(ctx context.Context, request string) (*domain.IntentResult, error) {
	args := m.Called(ctx, request)
	return args.Get(0).(*domain.IntentResult), args.Error(1)
}

func (m *MockLLMProvider) Health(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockLLMProvider) ExtractMetadata(ctx context.Context, content string, model string) (*domain.ExtractedMetadata, error) {
	args := m.Called(ctx, content, model)
	return args.Get(0).(*domain.ExtractedMetadata), args.Error(1)
}

func (m *MockLLMProvider) ProviderType() domain.ProviderType {
	args := m.Called()
	return args.Get(0).(domain.ProviderType)
}

// MockStreamGenerateServer implements the streaming server interface for testing
type MockStreamGenerateServer struct {
	mock.Mock
	ctx context.Context
}

func (m *MockStreamGenerateServer) Send(resp *pb.StreamGenerateResponse) error {
	args := m.Called(resp)
	return args.Error(0)
}

func (m *MockStreamGenerateServer) Context() context.Context {
	return m.ctx
}

func (m *MockStreamGenerateServer) SendMsg(msg interface{}) error {
	args := m.Called(msg)
	return args.Error(0)
}

func (m *MockStreamGenerateServer) RecvMsg(msg interface{}) error {
	args := m.Called(msg)
	return args.Error(0)
}

func (m *MockStreamGenerateServer) SetHeader(md metadata.MD) error {
	args := m.Called(md)
	return args.Error(0)
}

func (m *MockStreamGenerateServer) SendHeader(md metadata.MD) error {
	args := m.Called(md)
	return args.Error(0)
}

func (m *MockStreamGenerateServer) SetTrailer(md metadata.MD) {
	m.Called(md)
}

func TestNewLLMServiceServer(t *testing.T) {
	mockProvider := &MockLLMProvider{}
	server := NewLLMServiceServer(mockProvider)
	
	assert.NotNil(t, server)
	assert.Equal(t, mockProvider, server.llmProvider)
	assert.NotNil(t, server.metadataExtractor)
}

func TestLLMServiceServer_Generate(t *testing.T) {
	mockProvider := &MockLLMProvider{}
	server := NewLLMServiceServer(mockProvider)
	ctx := context.Background()

	t.Run("successful generation", func(t *testing.T) {
		req := &pb.GenerateRequest{
			Prompt: "test prompt",
			Model:  "test-model",
			Options: &pb.GenerationOptions{
				MaxTokens:   1000,
				Temperature: 0.7,
			},
		}

		expectedOpts := &domain.GenerationOptions{
			MaxTokens:   1000,
			Temperature: 0.7,
		}

		mockProvider.On("Generate", ctx, "test prompt", expectedOpts).Return("generated text", nil)

		resp, err := server.Generate(ctx, req)

		assert.NoError(t, err)
		assert.Equal(t, "generated text", resp.Text)
		assert.Equal(t, "test-model", resp.Model)
		assert.Greater(t, resp.GenerationTimeMs, int64(0))
		mockProvider.AssertExpectations(t)
	})

	t.Run("empty prompt", func(t *testing.T) {
		req := &pb.GenerateRequest{
			Prompt: "",
		}

		resp, err := server.Generate(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("generation failure", func(t *testing.T) {
		req := &pb.GenerateRequest{
			Prompt: "test prompt",
		}

		expectedOpts := &domain.GenerationOptions{
			MaxTokens:   1000,
			Temperature: 0.7,
		}

		mockProvider.On("Generate", ctx, "test prompt", expectedOpts).Return("", assert.AnError)

		resp, err := server.Generate(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
		mockProvider.AssertExpectations(t)
	})

	t.Run("generation with nil options", func(t *testing.T) {
		req := &pb.GenerateRequest{
			Prompt:  "test prompt",
			Options: nil,
		}

		expectedOpts := &domain.GenerationOptions{
			MaxTokens:   1000,
			Temperature: 0.7,
		}

		mockProvider.On("Generate", ctx, "test prompt", expectedOpts).Return("generated text", nil)

		resp, err := server.Generate(ctx, req)

		assert.NoError(t, err)
		assert.Equal(t, "generated text", resp.Text)
		mockProvider.AssertExpectations(t)
	})
}

func TestLLMServiceServer_StreamGenerate(t *testing.T) {
	mockProvider := &MockLLMProvider{}
	server := NewLLMServiceServer(mockProvider)
	ctx := context.Background()

	t.Run("successful stream generation", func(t *testing.T) {
		req := &pb.GenerateRequest{
			Prompt: "test prompt",
			Options: &pb.GenerationOptions{
				MaxTokens:   1000,
				Temperature: 0.7,
			},
		}

		mockStream := &MockStreamGenerateServer{ctx: ctx}
		
		// Expect text chunks
		mockStream.On("Send", mock.MatchedBy(func(resp *pb.StreamGenerateResponse) bool {
			return resp.GetTextChunk() == "chunk1"
		})).Return(nil)
		
		mockStream.On("Send", mock.MatchedBy(func(resp *pb.StreamGenerateResponse) bool {
			return resp.GetTextChunk() == "chunk2"
		})).Return(nil)
		
		// Expect metadata
		mockStream.On("Send", mock.MatchedBy(func(resp *pb.StreamGenerateResponse) bool {
			return resp.GetMetadata() != nil
		})).Return(nil)

		expectedOpts := &domain.GenerationOptions{
			MaxTokens:   1000,
			Temperature: 0.7,
		}

		mockProvider.On("Stream", ctx, "test prompt", expectedOpts, mock.AnythingOfType("func(string)")).Return(nil)

		err := server.StreamGenerate(req, mockStream)

		assert.NoError(t, err)
		mockProvider.AssertExpectations(t)
		mockStream.AssertExpectations(t)
	})

	t.Run("empty prompt", func(t *testing.T) {
		req := &pb.GenerateRequest{
			Prompt: "",
		}

		mockStream := &MockStreamGenerateServer{ctx: ctx}

		err := server.StreamGenerate(req, mockStream)

		assert.Error(t, err)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("stream generation failure", func(t *testing.T) {
		req := &pb.GenerateRequest{
			Prompt: "test prompt",
		}

		mockStream := &MockStreamGenerateServer{ctx: ctx}
		
		// Expect error message
		mockStream.On("Send", mock.MatchedBy(func(resp *pb.StreamGenerateResponse) bool {
			return resp.GetError() != ""
		})).Return(nil)

		expectedOpts := &domain.GenerationOptions{
			MaxTokens:   1000,
			Temperature: 0.7,
		}

		mockProvider.On("Stream", ctx, "test prompt", expectedOpts, mock.AnythingOfType("func(string)")).Return(assert.AnError)

		err := server.StreamGenerate(req, mockStream)

		assert.NoError(t, err) // Function doesn't return error, sends it via stream
		mockProvider.AssertExpectations(t)
		mockStream.AssertExpectations(t)
	})
}

func TestLLMServiceServer_GenerateWithTools(t *testing.T) {
	mockProvider := &MockLLMProvider{}
	server := NewLLMServiceServer(mockProvider)
	ctx := context.Background()

	t.Run("successful generation with tools", func(t *testing.T) {
		// Create test schema
		schema, _ := structpb.NewStruct(map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query",
				},
			},
			"required": []interface{}{"query"},
		})

		req := &pb.GenerateWithToolsRequest{
			Messages: []*pb.Message{
				{
					Role:    "user",
					Content: "Search for information about AI",
				},
			},
			Tools: []*pb.ToolDefinition{
				{
					Type: "function",
					Function: &pb.FunctionDefinition{
						Name:        "search",
						Description: "Search for information",
						Parameters:  schema,
					},
				},
			},
			Options: &pb.GenerationOptions{
				MaxTokens:   1000,
				Temperature: 0.7,
			},
		}

		expectedMessages := []domain.Message{
			{
				Role:    "user",
				Content: "Search for information about AI",
			},
		}

		expectedTools := []domain.ToolDefinition{
			{
				Type: "function",
				Function: domain.ToolFunction{
					Name:        "search",
					Description: "Search for information",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"query": map[string]interface{}{
								"type":        "string",
								"description": "Search query",
							},
						},
						"required": []interface{}{"query"},
					},
				},
			},
		}

		expectedOpts := &domain.GenerationOptions{
			MaxTokens:   1000,
			Temperature: 0.7,
		}

		expectedResult := &domain.GenerationResult{
			Content: "I'll search for AI information",
			ToolCalls: []domain.ToolCall{
				{
					ID:   "call-123",
					Type: "function",
					Function: domain.FunctionCall{
						Name: "search",
						Arguments: map[string]interface{}{
							"query": "artificial intelligence",
						},
					},
				},
			},
		}

		mockProvider.On("GenerateWithTools", ctx, expectedMessages, expectedTools, expectedOpts).Return(expectedResult, nil)

		resp, err := server.GenerateWithTools(ctx, req)

		assert.NoError(t, err)
		assert.Equal(t, "I'll search for AI information", resp.Content)
		assert.Len(t, resp.ToolCalls, 1)
		assert.Equal(t, "call-123", resp.ToolCalls[0].Id)
		assert.Equal(t, "function", resp.ToolCalls[0].Type)
		assert.Equal(t, "search", resp.ToolCalls[0].Function.Name)
		mockProvider.AssertExpectations(t)
	})

	t.Run("empty messages", func(t *testing.T) {
		req := &pb.GenerateWithToolsRequest{
			Messages: []*pb.Message{},
		}

		resp, err := server.GenerateWithTools(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("generation with tools failure", func(t *testing.T) {
		req := &pb.GenerateWithToolsRequest{
			Messages: []*pb.Message{
				{
					Role:    "user",
					Content: "test message",
				},
			},
		}

		expectedMessages := []domain.Message{
			{
				Role:    "user",
				Content: "test message",
			},
		}

		expectedOpts := &domain.GenerationOptions{
			MaxTokens:   1000,
			Temperature: 0.7,
		}

		mockProvider.On("GenerateWithTools", ctx, expectedMessages, []domain.ToolDefinition{}, expectedOpts).Return((*domain.GenerationResult)(nil), assert.AnError)

		resp, err := server.GenerateWithTools(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
		mockProvider.AssertExpectations(t)
	})
}

func TestLLMServiceServer_ExtractMetadata(t *testing.T) {
	mockProvider := &MockLLMProvider{}
	server := NewLLMServiceServer(mockProvider)
	ctx := context.Background()

	t.Run("successful metadata extraction", func(t *testing.T) {
		req := &pb.ExtractMetadataRequest{
			Content:      "This is a test document about artificial intelligence",
			DocumentType: "text",
		}

		expectedMetadata := &domain.ExtractedMetadata{
			Summary:      "A document about AI",
			Keywords:     []string{"artificial", "intelligence", "AI"},
			DocumentType: "text",
			Entities: map[string][]string{
				"technology": {"artificial intelligence", "AI"},
			},
			CustomMeta: map[string]interface{}{
				"topics":    []string{"technology", "AI"},
				"sentiment": "neutral",
				"language":  "english",
			},
		}

		mockProvider.On("ExtractMetadata", ctx, "This is a test document about artificial intelligence", "").Return(expectedMetadata, nil)

		resp, err := server.ExtractMetadata(ctx, req)

		assert.NoError(t, err)
		assert.Equal(t, "A document about AI", resp.Summary)
		assert.Equal(t, []string{"artificial", "intelligence", "AI"}, resp.Keywords)
		assert.Equal(t, "text", resp.DocumentType)
		assert.Contains(t, resp.Entities, "technology")
		assert.Equal(t, []string{"technology", "AI"}, resp.Topics)
		assert.Equal(t, "neutral", resp.Sentiment)
		assert.Equal(t, "english", resp.Language)
		mockProvider.AssertExpectations(t)
	})

	t.Run("empty content", func(t *testing.T) {
		req := &pb.ExtractMetadataRequest{
			Content: "",
		}

		resp, err := server.ExtractMetadata(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("metadata extraction failure", func(t *testing.T) {
		req := &pb.ExtractMetadataRequest{
			Content:      "test content",
			DocumentType: "text",
		}

		mockProvider.On("ExtractMetadata", ctx, "test content", "").Return((*domain.ExtractedMetadata)(nil), assert.AnError)

		resp, err := server.ExtractMetadata(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
		mockProvider.AssertExpectations(t)
	})
}

func TestLLMServiceServer_ListModels(t *testing.T) {
	mockProvider := &MockLLMProvider{}
	server := NewLLMServiceServer(mockProvider)
	ctx := context.Background()

	t.Run("list models for OpenAI provider", func(t *testing.T) {
		req := &pb.ListModelsRequest{}

		mockProvider.On("ProviderType").Return(domain.ProviderOpenAI)

		resp, err := server.ListModels(ctx, req)

		assert.NoError(t, err)
		assert.Greater(t, len(resp.Models), 1) // Should have default + OpenAI models
		
		// Check for default model
		defaultFound := false
		gpt4Found := false
		for _, model := range resp.Models {
			if model.Id == "default" {
				defaultFound = true
				assert.Equal(t, "Default Model", model.Name)
				assert.Equal(t, string(domain.ProviderOpenAI), model.Provider)
			}
			if model.Id == "gpt-4" {
				gpt4Found = true
				assert.Equal(t, "GPT-4", model.Name)
				assert.Equal(t, string(domain.ProviderOpenAI), model.Provider)
			}
		}
		assert.True(t, defaultFound)
		assert.True(t, gpt4Found)
		mockProvider.AssertExpectations(t)
	})

	t.Run("list models for Claude provider", func(t *testing.T) {
		req := &pb.ListModelsRequest{}

		mockProvider.On("ProviderType").Return(domain.ProviderClaude)

		resp, err := server.ListModels(ctx, req)

		assert.NoError(t, err)
		assert.Greater(t, len(resp.Models), 1)
		
		// Check for Claude models
		claudeOpusFound := false
		for _, model := range resp.Models {
			if model.Id == "claude-3-opus" {
				claudeOpusFound = true
				assert.Equal(t, "Claude 3 Opus", model.Name)
				assert.Equal(t, string(domain.ProviderClaude), model.Provider)
			}
		}
		assert.True(t, claudeOpusFound)
		mockProvider.AssertExpectations(t)
	})

	t.Run("list models for Gemini provider", func(t *testing.T) {
		req := &pb.ListModelsRequest{}

		mockProvider.On("ProviderType").Return(domain.ProviderGemini)

		resp, err := server.ListModels(ctx, req)

		assert.NoError(t, err)
		assert.Greater(t, len(resp.Models), 1)
		
		// Check for Gemini models
		geminiProFound := false
		for _, model := range resp.Models {
			if model.Id == "gemini-pro" {
				geminiProFound = true
				assert.Equal(t, "Gemini Pro", model.Name)
				assert.Equal(t, string(domain.ProviderGemini), model.Provider)
			}
		}
		assert.True(t, geminiProFound)
		mockProvider.AssertExpectations(t)
	})

	t.Run("list models for unknown provider", func(t *testing.T) {
		req := &pb.ListModelsRequest{}

		mockProvider.On("ProviderType").Return(domain.ProviderType("unknown"))

		resp, err := server.ListModels(ctx, req)

		assert.NoError(t, err)
		assert.Len(t, resp.Models, 1) // Only default model
		assert.Equal(t, "default", resp.Models[0].Id)
		mockProvider.AssertExpectations(t)
	})
}

func TestConvertGenerationOptions(t *testing.T) {
	t.Run("convert valid options", func(t *testing.T) {
		pbOpts := &pb.GenerationOptions{
			MaxTokens:   2000,
			Temperature: 0.8,
		}

		opts := convertGenerationOptions(pbOpts)

		assert.Equal(t, 2000, opts.MaxTokens)
		assert.Equal(t, 0.8, opts.Temperature)
	})

	t.Run("convert nil options", func(t *testing.T) {
		opts := convertGenerationOptions(nil)

		assert.Equal(t, 1000, opts.MaxTokens)
		assert.Equal(t, 0.7, opts.Temperature)
	})
}