package providers

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
	"github.com/openai/openai-go/v2/packages/param"
	"github.com/openai/openai-go/v2/shared"
	"github.com/liliang-cn/rago/v2/pkg/core"
)

// OpenAIProvider implements the Provider interface for OpenAI API
type OpenAIProvider struct {
	config core.ProviderConfig
	name   string
	client *openai.Client
}

// NewOpenAIProvider creates a new OpenAI provider for the LLM pillar
func NewOpenAIProvider(name string, config core.ProviderConfig) (*OpenAIProvider, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}
	
	// Build client options
	opts := []option.RequestOption{
		option.WithAPIKey(config.APIKey),
	}
	
	// Add custom base URL if provided
	if config.BaseURL != "" && config.BaseURL != "https://api.openai.com/v1" {
		opts = append(opts, option.WithBaseURL(config.BaseURL))
	}
	
	// Add timeout if specified
	if config.Timeout > 0 {
		opts = append(opts, option.WithRequestTimeout(config.Timeout))
	}
	
	// Create OpenAI client - NewClient returns a value, not a pointer
	client := openai.NewClient(opts...)
	
	return &OpenAIProvider{
		config: config,
		name:   name,
		client: &client,
	}, nil
}

// Name returns the provider name
func (p *OpenAIProvider) Name() string {
	return p.name
}

// Type returns the provider type
func (p *OpenAIProvider) Type() string {
	return "openai"
}

// Model returns the configured model
func (p *OpenAIProvider) Model() string {
	return p.config.Model
}

// Config returns the provider configuration
func (p *OpenAIProvider) Config() core.ProviderConfig {
	return p.config
}

// Health checks the provider health
func (p *OpenAIProvider) Health(ctx context.Context) error {
	// Check OpenAI API by listing models
	_, err := p.client.Models.List(ctx)
	if err != nil {
		return fmt.Errorf("OpenAI API not reachable: %w", err)
	}
	return nil
}

// Generate generates text using the OpenAI provider
func (p *OpenAIProvider) Generate(ctx context.Context, req *GenerationRequest) (*GenerationResponse, error) {
	// Build messages
	messages := p.buildMessages(req)
	
	// Create OpenAI request - direct assignment, no F() wrapper
	params := openai.ChatCompletionNewParams{
		Model:    p.config.Model,
		Messages: messages,
	}
	
	if req.Temperature > 0 {
		params.Temperature = param.NewOpt(float64(req.Temperature))
	}
	
	if req.MaxTokens > 0 {
		params.MaxTokens = param.NewOpt(int64(req.MaxTokens))
	}
	
	startTime := time.Now()
	
	// Generate response
	completion, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("OpenAI generation failed: %w", err)
	}
	
	if len(completion.Choices) == 0 {
		return nil, fmt.Errorf("no choices in OpenAI response")
	}
	
	// Create response
	return &GenerationResponse{
		Content:  completion.Choices[0].Message.Content,
		Model:    completion.Model,
		Provider: p.name,
		Usage: TokenUsage{
			PromptTokens:     int(completion.Usage.PromptTokens),
			CompletionTokens: int(completion.Usage.CompletionTokens),
			TotalTokens:      int(completion.Usage.TotalTokens),
		},
		Duration: time.Since(startTime),
		Metadata: map[string]interface{}{
			"id":      completion.ID,
			"created": completion.Created,
		},
	}, nil
}

// Stream generates streaming text using the OpenAI provider
func (p *OpenAIProvider) Stream(ctx context.Context, req *GenerationRequest, callback StreamCallback) error {
	// Build messages
	messages := p.buildMessages(req)
	
	// Create OpenAI request for streaming - no Stream field needed
	params := openai.ChatCompletionNewParams{
		Model:    p.config.Model,
		Messages: messages,
	}
	
	if req.Temperature > 0 {
		params.Temperature = param.NewOpt(float64(req.Temperature))
	}
	
	if req.MaxTokens > 0 {
		params.MaxTokens = param.NewOpt(int64(req.MaxTokens))
	}
	
	startTime := time.Now()
	var totalContent strings.Builder
	
	// Stream response
	stream := p.client.Chat.Completions.NewStreaming(ctx, params)
	defer stream.Close()
	
	for stream.Next() {
		chunk := stream.Current()
		
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			delta := chunk.Choices[0].Delta.Content
			totalContent.WriteString(delta)
			
			streamChunk := &StreamChunk{
				Content:  totalContent.String(),
				Delta:    delta,
				Finished: false,
				Duration: time.Since(startTime),
			}
			callback(streamChunk)
		}
		
		// Check if this is the final chunk
		if len(chunk.Choices) > 0 && chunk.Choices[0].FinishReason != "" {
			streamChunk := &StreamChunk{
				Content:  totalContent.String(),
				Delta:    "",
				Finished: true,
				Duration: time.Since(startTime),
			}
			
			// Note: Streaming responses don't include usage in openai-go v2
			// Usage is only available in non-streaming responses
			
			callback(streamChunk)
			break
		}
	}
	
	if err := stream.Err(); err != nil {
		if err != io.EOF {
			return fmt.Errorf("OpenAI stream failed: %w", err)
		}
	}
	
	return nil
}

// GenerateWithTools generates text with tool calling
func (p *OpenAIProvider) GenerateWithTools(ctx context.Context, req *ToolGenerationRequest) (*ToolGenerationResponse, error) {
	// Build messages and tools
	messages := p.buildMessages(&req.GenerationRequest)
	tools := p.buildTools(req.Tools)
	
	// Create OpenAI request - direct assignment, no F() wrapper
	params := openai.ChatCompletionNewParams{
		Model:    p.config.Model,
		Messages: messages,
		Tools:    tools,
	}
	
	if req.Temperature > 0 {
		params.Temperature = param.NewOpt(float64(req.Temperature))
	}
	
	if req.MaxTokens > 0 {
		params.MaxTokens = param.NewOpt(int64(req.MaxTokens))
	}
	
	if req.ToolChoice != "" {
		// Tool choice handling - auto, none, or required
		switch req.ToolChoice {
		case "auto":
			params.ToolChoice = openai.ChatCompletionToolChoiceOptionUnionParam{
				OfAuto: param.NewOpt("auto"),
			}
		case "none":
			params.ToolChoice = openai.ChatCompletionToolChoiceOptionUnionParam{
				OfAuto: param.NewOpt("none"),
			}
		default:
			// For "required" or specific function names
			params.ToolChoice = openai.ChatCompletionToolChoiceOptionUnionParam{
				OfAuto: param.NewOpt("required"),
			}
		}
	}
	
	startTime := time.Now()
	
	// Generate response
	completion, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("OpenAI tool generation failed: %w", err)
	}
	
	if len(completion.Choices) == 0 {
		return nil, fmt.Errorf("no choices in OpenAI response")
	}
	
	// Extract tool calls
	choice := completion.Choices[0]
	toolCalls := make([]ToolCall, 0)
	
	for _, tc := range choice.Message.ToolCalls {
		toolCalls = append(toolCalls, ToolCall{
			ID:   tc.ID,
			Name: tc.Function.Name,
			Parameters: map[string]interface{}{
				"arguments": tc.Function.Arguments,
			},
		})
	}
	
	// Create response
	return &ToolGenerationResponse{
		GenerationResponse: GenerationResponse{
			Content:  choice.Message.Content,
			Model:    completion.Model,
			Provider: p.name,
			Usage: TokenUsage{
				PromptTokens:     int(completion.Usage.PromptTokens),
				CompletionTokens: int(completion.Usage.CompletionTokens),
				TotalTokens:      int(completion.Usage.TotalTokens),
			},
			Duration: time.Since(startTime),
			Metadata: map[string]interface{}{
				"id":      completion.ID,
				"created": completion.Created,
			},
		},
		ToolCalls: toolCalls,
	}, nil
}

// StreamWithTools generates streaming text with tool calling
func (p *OpenAIProvider) StreamWithTools(ctx context.Context, req *ToolGenerationRequest, callback ToolStreamCallback) error {
	// Build messages and tools
	messages := p.buildMessages(&req.GenerationRequest)
	tools := p.buildTools(req.Tools)
	
	// Create OpenAI request for streaming with tools - no Stream field needed
	params := openai.ChatCompletionNewParams{
		Model:    p.config.Model,
		Messages: messages,
		Tools:    tools,
	}
	
	if req.Temperature > 0 {
		params.Temperature = param.NewOpt(float64(req.Temperature))
	}
	
	if req.MaxTokens > 0 {
		params.MaxTokens = param.NewOpt(int64(req.MaxTokens))
	}
	
	if req.ToolChoice != "" {
		// Tool choice handling - auto, none, or required
		switch req.ToolChoice {
		case "auto":
			params.ToolChoice = openai.ChatCompletionToolChoiceOptionUnionParam{
				OfAuto: param.NewOpt("auto"),
			}
		case "none":
			params.ToolChoice = openai.ChatCompletionToolChoiceOptionUnionParam{
				OfAuto: param.NewOpt("none"),
			}
		default:
			// For "required" or specific function names
			params.ToolChoice = openai.ChatCompletionToolChoiceOptionUnionParam{
				OfAuto: param.NewOpt("required"),
			}
		}
	}
	
	startTime := time.Now()
	var totalContent strings.Builder
	var accumulatedToolCalls []ToolCall
	
	// Stream response
	stream := p.client.Chat.Completions.NewStreaming(ctx, params)
	defer stream.Close()
	
	for stream.Next() {
		chunk := stream.Current()
		
		if len(chunk.Choices) > 0 {
			choice := chunk.Choices[0]
			
			// Handle content delta
			if choice.Delta.Content != "" {
				delta := choice.Delta.Content
				totalContent.WriteString(delta)
				
				toolChunk := &ToolStreamChunk{
					StreamChunk: StreamChunk{
						Content:  totalContent.String(),
						Delta:    delta,
						Finished: false,
						Duration: time.Since(startTime),
					},
					ToolCalls: accumulatedToolCalls,
				}
				callback(toolChunk)
			}
			
			// Handle tool calls
			for _, tc := range choice.Delta.ToolCalls {
				// Accumulate tool calls
				found := false
				for i, existing := range accumulatedToolCalls {
					if existing.ID == tc.ID {
						// Update existing tool call
						if tc.Function.Arguments != "" {
							if args, ok := existing.Parameters["arguments"].(string); ok {
								accumulatedToolCalls[i].Parameters["arguments"] = args + tc.Function.Arguments
							} else {
								accumulatedToolCalls[i].Parameters["arguments"] = tc.Function.Arguments
							}
						}
						found = true
						break
					}
				}
				
				if !found && tc.ID != "" {
					// New tool call
					accumulatedToolCalls = append(accumulatedToolCalls, ToolCall{
						ID:   tc.ID,
						Name: tc.Function.Name,
						Parameters: map[string]interface{}{
							"arguments": tc.Function.Arguments,
						},
					})
				}
			}
			
			// Check if this is the final chunk
			if choice.FinishReason != "" {
				toolChunk := &ToolStreamChunk{
					StreamChunk: StreamChunk{
						Content:  totalContent.String(),
						Delta:    "",
						Finished: true,
						Duration: time.Since(startTime),
					},
					ToolCalls: accumulatedToolCalls,
				}
				callback(toolChunk)
				break
			}
		}
	}
	
	if err := stream.Err(); err != nil {
		if err != io.EOF {
			return fmt.Errorf("OpenAI stream with tools failed: %w", err)
		}
	}
	
	return nil
}

// Capabilities returns the provider capabilities
func (p *OpenAIProvider) Capabilities() ProviderCapabilities {
	return ProviderCapabilities{
		SupportsStreaming:  true,
		SupportsToolCalls:  true,
		SupportsBatch:      false, // Could be implemented with batch API
		MaxTokens:          getMaxTokensForModel(p.config.Model),
		MaxContextLength:   getContextLengthForModel(p.config.Model),
	}
}

// Metadata returns additional provider metadata
func (p *OpenAIProvider) Metadata() map[string]interface{} {
	return map[string]interface{}{
		"provider_type":   "openai",
		"model":           p.config.Model,
		"timeout":         p.config.Timeout,
		"supports_tools":  true,
		"supports_stream": true,
	}
}

// Close closes the provider and cleans up resources
func (p *OpenAIProvider) Close() error {
	// Nothing to clean up for OpenAI client
	return nil
}

// buildMessages builds OpenAI chat messages from the request
func (p *OpenAIProvider) buildMessages(req *GenerationRequest) []openai.ChatCompletionMessageParamUnion {
	messages := make([]openai.ChatCompletionMessageParamUnion, 0)
	
	// Add context messages
	for _, msg := range req.Context {
		switch msg.Role {
		case "system":
			messages = append(messages, openai.SystemMessage(msg.Content))
		case "user":
			messages = append(messages, openai.UserMessage(msg.Content))
		case "assistant":
			messages = append(messages, openai.AssistantMessage(msg.Content))
		case "tool":
			// Tool messages need special handling
			messages = append(messages, openai.ToolMessage(msg.ToolCallID, msg.Content))
		default:
			// Default to user message for unknown roles
			messages = append(messages, openai.UserMessage(msg.Content))
		}
	}
	
	// Add the main prompt as user message
	if req.Prompt != "" {
		messages = append(messages, openai.UserMessage(req.Prompt))
	}
	
	return messages
}

// buildTools builds OpenAI tool definitions
func (p *OpenAIProvider) buildTools(tools []ToolInfo) []openai.ChatCompletionToolUnionParam {
	openaiTools := make([]openai.ChatCompletionToolUnionParam, len(tools))
	
	for i, tool := range tools {
		// Use ChatCompletionToolUnionParam for tools
		openaiTools[i] = openai.ChatCompletionToolUnionParam{
			OfFunction: &openai.ChatCompletionFunctionToolParam{
				Type: "function",
				Function: shared.FunctionDefinitionParam{
					Name:        tool.Name,
					Description: param.NewOpt(tool.Description),
					Parameters:  shared.FunctionParameters(tool.Parameters),
				},
			},
		}
	}
	
	return openaiTools
}

// getMaxTokensForModel returns the maximum tokens for a given OpenAI model
func getMaxTokensForModel(model string) int {
	switch model {
	case "gpt-4-turbo", "gpt-4-turbo-preview", "gpt-4o", "gpt-4o-mini":
		return 128000
	case "gpt-4", "gpt-4-0613", "gpt-4-32k":
		return 8192
	case "gpt-3.5-turbo", "gpt-3.5-turbo-0613":
		return 4096
	case "gpt-3.5-turbo-16k":
		return 16384
	default:
		return 4096 // Default fallback
	}
}

// getContextLengthForModel returns the context length for a given OpenAI model
func getContextLengthForModel(model string) int {
	switch model {
	case "gpt-4-turbo", "gpt-4-turbo-preview", "gpt-4o", "gpt-4o-mini":
		return 128000
	case "gpt-4-32k":
		return 32768
	case "gpt-4", "gpt-4-0613":
		return 8192
	case "gpt-3.5-turbo-16k":
		return 16384
	case "gpt-3.5-turbo", "gpt-3.5-turbo-0613":
		return 4096
	default:
		return 4096 // Default fallback
	}
}