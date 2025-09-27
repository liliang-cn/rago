package server

import (
	"context"
	"encoding/json"
	"time"

	pb "github.com/liliang-cn/rago/v2/proto/rago"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/metadata"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

// LLMServiceServer implements the LLM gRPC service
type LLMServiceServer struct {
	pb.UnimplementedLLMServiceServer
	llmProvider       domain.LLMProvider
	metadataExtractor *metadata.MetadataExtractor
}

// NewLLMServiceServer creates a new LLM service server
func NewLLMServiceServer(llmProvider domain.LLMProvider) *LLMServiceServer {
	return &LLMServiceServer{
		llmProvider:       llmProvider,
		metadataExtractor: metadata.NewMetadataExtractor(llmProvider),
	}
}

// Generate generates text using the LLM
func (s *LLMServiceServer) Generate(ctx context.Context, req *pb.GenerateRequest) (*pb.GenerateResponse, error) {
	if req.Prompt == "" {
		return nil, status.Error(codes.InvalidArgument, "prompt cannot be empty")
	}

	startTime := time.Now()

	// Convert generation options
	opts := convertGenerationOptions(req.Options)

	// Generate text
	response, err := s.llmProvider.Generate(ctx, req.Prompt, opts)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generation failed: %v", err)
	}

	return &pb.GenerateResponse{
		Text:             response,
		Model:            req.Model,
		GenerationTimeMs: time.Since(startTime).Milliseconds(),
	}, nil
}

// StreamGenerate performs streaming text generation
func (s *LLMServiceServer) StreamGenerate(req *pb.GenerateRequest, stream pb.LLMService_StreamGenerateServer) error {
	if req.Prompt == "" {
		return status.Error(codes.InvalidArgument, "prompt cannot be empty")
	}

	ctx := stream.Context()
	startTime := time.Now()
	totalTokens := int32(0)

	// Convert generation options
	opts := convertGenerationOptions(req.Options)

	// Create streaming callback
	callback := func(chunk string) {
		totalTokens++ // Simple token counting (actual implementation would be more sophisticated)
		stream.Send(&pb.StreamGenerateResponse{
			Content: &pb.StreamGenerateResponse_TextChunk{
				TextChunk: chunk,
			},
		})
	}

	// Perform streaming generation
	err := s.llmProvider.Stream(ctx, req.Prompt, opts, callback)
	if err != nil {
		stream.Send(&pb.StreamGenerateResponse{
			Content: &pb.StreamGenerateResponse_Error{
				Error: err.Error(),
			},
		})
		return nil
	}

	// Send metadata
	stream.Send(&pb.StreamGenerateResponse{
		Content: &pb.StreamGenerateResponse_Metadata{
			Metadata: &pb.GenerationMetadata{
				TokensUsed:       totalTokens,
				GenerationTimeMs: time.Since(startTime).Milliseconds(),
				FinishReason:     "stop",
			},
		},
	})

	return nil
}

// GenerateWithTools generates text with tool calling capability
func (s *LLMServiceServer) GenerateWithTools(ctx context.Context, req *pb.GenerateWithToolsRequest) (*pb.GenerateWithToolsResponse, error) {
	if len(req.Messages) == 0 {
		return nil, status.Error(codes.InvalidArgument, "messages cannot be empty")
	}

	// Convert messages
	messages := make([]domain.Message, 0, len(req.Messages))
	for _, msg := range req.Messages {
		domainMsg := domain.Message{
			Role:       msg.Role,
			Content:    msg.Content,
			ToolCallID: msg.ToolCallId,
		}

		// Convert tool calls
		if len(msg.ToolCalls) > 0 {
			domainMsg.ToolCalls = make([]domain.ToolCall, 0, len(msg.ToolCalls))
			for _, tc := range msg.ToolCalls {
				// Parse arguments from JSON string to map
				var args map[string]interface{}
				if tc.Function != nil && tc.Function.Arguments != "" {
					json.Unmarshal([]byte(tc.Function.Arguments), &args)
				}
				
				domainMsg.ToolCalls = append(domainMsg.ToolCalls, domain.ToolCall{
					ID:   tc.Id,
					Type: tc.Type,
					Function: domain.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: args,
					},
				})
			}
		}

		messages = append(messages, domainMsg)
	}

	// Convert tools
	tools := make([]domain.ToolDefinition, 0, len(req.Tools))
	for _, tool := range req.Tools {
		// Convert parameters from protobuf Struct to map
		var params map[string]interface{}
		if tool.Function != nil && tool.Function.Parameters != nil {
			paramsBytes, _ := tool.Function.Parameters.MarshalJSON()
			json.Unmarshal(paramsBytes, &params)
		}

		tools = append(tools, domain.ToolDefinition{
			Type: tool.Type,
			Function: domain.ToolFunction{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Parameters:  params,
			},
		})
	}

	// Convert options
	opts := convertGenerationOptions(req.Options)

	// Generate with tools
	result, err := s.llmProvider.GenerateWithTools(ctx, messages, tools, opts)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generation with tools failed: %v", err)
	}

	// Convert tool calls in response
	toolCalls := make([]*pb.ToolCall, 0, len(result.ToolCalls))
	for _, tc := range result.ToolCalls {
		// Convert arguments from map to JSON string
		argsJSON := ""
		if tc.Function.Arguments != nil {
			if bytes, err := json.Marshal(tc.Function.Arguments); err == nil {
				argsJSON = string(bytes)
			}
		}
		
		toolCalls = append(toolCalls, &pb.ToolCall{
			Id:   tc.ID,
			Type: tc.Type,
			Function: &pb.FunctionCall{
				Name:      tc.Function.Name,
				Arguments: argsJSON,
			},
		})
	}

	return &pb.GenerateWithToolsResponse{
		Content:      result.Content,
		ToolCalls:    toolCalls,
		FinishReason: "stop", // Default finish reason
		TokensUsed:   0,      // Token counting not implemented yet
		Model:        req.Model,
	}, nil
}

// ExtractMetadata extracts metadata from content
func (s *LLMServiceServer) ExtractMetadata(ctx context.Context, req *pb.ExtractMetadataRequest) (*pb.ExtractMetadataResponse, error) {
	if req.Content == "" {
		return nil, status.Error(codes.InvalidArgument, "content cannot be empty")
	}

	// Extract metadata
	metadata, err := s.metadataExtractor.ExtractEnhancedMetadata(ctx, req.Content, req.DocumentType)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "metadata extraction failed: %v", err)
	}

	// Convert entities to protobuf format
	entities := make(map[string]*structpb.ListValue)
	for k, v := range metadata.Entities {
		values := make([]*structpb.Value, 0, len(v))
		for _, item := range v {
			val, _ := structpb.NewValue(item)
			values = append(values, val)
		}
		entities[k] = &structpb.ListValue{Values: values}
	}

	// Convert custom metadata
	customMeta := make(map[string]*structpb.Value)
	for k, v := range metadata.CustomMeta {
		val, _ := structpb.NewValue(v)
		customMeta[k] = val
	}

	// Get topics from custom metadata
	var topics []string
	if topicsVal, ok := metadata.CustomMeta["topics"].([]string); ok {
		topics = topicsVal
	}

	// Get sentiment and language from custom metadata
	sentiment, _ := metadata.CustomMeta["sentiment"].(string)
	language, _ := metadata.CustomMeta["language"].(string)

	return &pb.ExtractMetadataResponse{
		Summary:        metadata.Summary,
		Keywords:       metadata.Keywords,
		DocumentType:   metadata.DocumentType,
		Entities:       entities,
		Topics:         topics,
		CustomMetadata: customMeta,
		Sentiment:      sentiment,
		Language:       language,
	}, nil
}

// ListModels lists available models
func (s *LLMServiceServer) ListModels(ctx context.Context, req *pb.ListModelsRequest) (*pb.ListModelsResponse, error) {
	// Get provider type
	providerType := s.llmProvider.ProviderType()

	// For now, return a static list based on provider
	// In a real implementation, this would query the provider for available models
	models := []*pb.ModelInfo{
		{
			Id:       "default",
			Name:     "Default Model",
			Provider: string(providerType),
			Type:     "llm",
			Capabilities: map[string]string{
				"max_tokens":      "4096",
				"supports_tools":  "true",
				"supports_stream": "true",
			},
			Available: true,
		},
	}

	// Add provider-specific models
	switch providerType {
	case domain.ProviderOpenAI:
		models = append(models, &pb.ModelInfo{
			Id:       "gpt-4",
			Name:     "GPT-4",
			Provider: string(providerType),
			Type:     "llm",
			Available: true,
		})
		models = append(models, &pb.ModelInfo{
			Id:       "gpt-3.5-turbo",
			Name:     "GPT-3.5 Turbo",
			Provider: string(providerType),
			Type:     "llm",
			Available: true,
		})
	case domain.ProviderClaude:
		models = append(models, &pb.ModelInfo{
			Id:       "claude-3-opus",
			Name:     "Claude 3 Opus",
			Provider: string(providerType),
			Type:     "llm",
			Available: true,
		})
		models = append(models, &pb.ModelInfo{
			Id:       "claude-3-sonnet",
			Name:     "Claude 3 Sonnet",
			Provider: string(providerType),
			Type:     "llm",
			Available: true,
		})
	case domain.ProviderGemini:
		models = append(models, &pb.ModelInfo{
			Id:       "gemini-pro",
			Name:     "Gemini Pro",
			Provider: string(providerType),
			Type:     "llm",
			Available: true,
		})
		models = append(models, &pb.ModelInfo{
			Id:       "gemini-1.5-pro",
			Name:     "Gemini 1.5 Pro",
			Provider: string(providerType),
			Type:     "llm",
			Available: true,
		})
	}

	return &pb.ListModelsResponse{
		Models: models,
	}, nil
}

// convertGenerationOptions converts protobuf generation options to domain options
func convertGenerationOptions(pbOpts *pb.GenerationOptions) *domain.GenerationOptions {
	if pbOpts == nil {
		return &domain.GenerationOptions{
			MaxTokens:   1000,
			Temperature: 0.7,
		}
	}

	opts := &domain.GenerationOptions{
		MaxTokens:   int(pbOpts.MaxTokens),
		Temperature: pbOpts.Temperature,
	}

	// Store additional options in ToolChoice field as JSON for now
	// In production, you'd extend the domain.GenerationOptions struct
	return opts
}