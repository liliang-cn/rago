package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/liliang-cn/lmstudio-go"
	"github.com/liliang-cn/rago/v2/pkg/core"
)

// LMStudioProvider implements the Provider interface for LMStudio
type LMStudioProvider struct {
	config  core.ProviderConfig
	name    string
	client  *lmstudio.Client
	baseURL string
}

// NewLMStudioProvider creates a new LMStudio provider for the LLM pillar
func NewLMStudioProvider(name string, config core.ProviderConfig) (*LMStudioProvider, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:1234"
	}

	// Create LMStudio client with config
	lmConfig := &lmstudio.Config{
		BaseURL: baseURL,
		Timeout: time.Duration(config.Timeout) * time.Second,
	}
	client := lmstudio.NewClientWithConfig(lmConfig)

	return &LMStudioProvider{
		config:  config,
		name:    name,
		client:  client,
		baseURL: baseURL,
	}, nil
}

// Name returns the provider name
func (p *LMStudioProvider) Name() string {
	return p.name
}

// Type returns the provider type
func (p *LMStudioProvider) Type() string {
	return "lmstudio"
}

// Model returns the configured model
func (p *LMStudioProvider) Model() string {
	return p.config.Model
}

// Config returns the provider configuration
func (p *LMStudioProvider) Config() core.ProviderConfig {
	return p.config
}

// Health checks the provider health
func (p *LMStudioProvider) Health(ctx context.Context) error {
	// Check if LMStudio server is running by listing models
	_, err := p.client.Models.List(ctx)
	if err != nil {
		return fmt.Errorf("LMStudio server not reachable: %w", err)
	}
	return nil
}

// Generate generates text using the LMStudio provider
func (p *LMStudioProvider) Generate(ctx context.Context, req *GenerationRequest) (*GenerationResponse, error) {
	// Build messages
	messages := p.buildMessages(req)

	// Create temperature pointer
	temperature := float64(req.Temperature)
	maxTokens := req.MaxTokens

	// Create LMStudio request
	lmReq := &lmstudio.ChatRequest{
		Model:       p.config.Model,
		Messages:    messages,
		Temperature: &temperature,
		MaxTokens:   &maxTokens,
		Stream:      false,
	}

	startTime := time.Now()

	// Generate response
	resp, err := p.client.Chat.Complete(ctx, lmReq)
	if err != nil {
		return nil, fmt.Errorf("LMStudio generation failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in LMStudio response")
	}

	// Create response
	return &GenerationResponse{
		Content:  resp.Choices[0].Message.Content,
		Model:    resp.Model,
		Provider: p.name,
		Usage: TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
		Duration: time.Since(startTime),
		Metadata: map[string]interface{}{
			"id":      resp.ID,
			"created": resp.Created,
		},
	}, nil
}

// Stream generates streaming text using the LMStudio provider
func (p *LMStudioProvider) Stream(ctx context.Context, req *GenerationRequest, callback StreamCallback) error {
	// Build messages
	messages := p.buildMessages(req)

	// Create temperature and max tokens pointers
	temperature := float64(req.Temperature)
	maxTokens := req.MaxTokens

	// Create LMStudio request
	lmReq := &lmstudio.ChatRequest{
		Model:       p.config.Model,
		Messages:    messages,
		Temperature: &temperature,
		MaxTokens:   &maxTokens,
		Stream:      true,
	}

	startTime := time.Now()
	var totalContent strings.Builder

	// Stream response
	stream, err := p.client.Chat.Stream(ctx, lmReq)
	if err != nil {
		return fmt.Errorf("LMStudio stream failed: %w", err)
	}
	defer stream.Close()

	// Read from the response channel
	for {
		select {
		case resp, ok := <-stream.Response:
			if !ok {
				// Channel closed, streaming finished
				return nil
			}

			if len(resp.Choices) > 0 && resp.Choices[0].Delta.Content != nil {
				delta := *resp.Choices[0].Delta.Content
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
			if len(resp.Choices) > 0 && resp.Choices[0].FinishReason != nil && *resp.Choices[0].FinishReason != "" {
				streamChunk := &StreamChunk{
					Content:  totalContent.String(),
					Delta:    "",
					Finished: true,
					Duration: time.Since(startTime),
				}

				// Note: Streaming responses typically don't include usage info
				// but we'll check just in case

				callback(streamChunk)
				return nil
			}

		case err := <-stream.Errors:
			if err != nil {
				return fmt.Errorf("stream error: %w", err)
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// GenerateWithTools generates text with native tool calling support
func (p *LMStudioProvider) GenerateWithTools(ctx context.Context, req *ToolGenerationRequest) (*ToolGenerationResponse, error) {
	// Build messages from request
	messages := p.buildMessages(&req.GenerationRequest)
	
	// Convert tools to LMStudio format with proper JSON schema
	var tools []lmstudio.Tool
	for _, tool := range req.Tools {
		// Ensure parameters have proper JSON schema structure
		parameters := tool.Parameters
		if parameters == nil {
			parameters = make(map[string]interface{})
		}
		
		// Ensure required fields for JSON schema
		if _, ok := parameters["type"]; !ok {
			parameters["type"] = "object"
		}
		if _, ok := parameters["properties"]; !ok {
			parameters["properties"] = make(map[string]interface{})
		}
		
		lmTool := lmstudio.Tool{
			Type: "function",
			Function: lmstudio.ToolFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  parameters,
			},
		}
		tools = append(tools, lmTool)
	}
	
	// Get model name
	model := req.Model
	if model == "" {
		model = p.config.Model
	}
	
	// Create temperature and max tokens pointers
	temp := float64(req.Temperature)
	maxTok := req.MaxTokens
	
	// Create chat request with tools
	chatReq := &lmstudio.ChatRequestWithTools{
		ChatRequest: lmstudio.ChatRequest{
			Model:       model,
			Messages:    messages,
			Temperature: &temp,
			MaxTokens:   &maxTok,
		},
		Tools: tools,
	}
	
	// Set tool choice if specified
	if req.ToolChoice != "" {
		if req.ToolChoice == "auto" {
			chatReq.ToolChoice = lmstudio.ToolChoiceAuto
		} else if req.ToolChoice == "none" {
			chatReq.ToolChoice = lmstudio.ToolChoiceNone
		} else if req.ToolChoice == "required" {
			chatReq.ToolChoice = lmstudio.ToolChoiceRequired
		}
	}
	
	// Make the API call
	response, err := p.client.Chat.CompleteWithTools(ctx, chatReq)
	if err != nil {
		return nil, fmt.Errorf("lmstudio tool generation failed: %w", err)
	}
	
	// Convert tool calls from response - LMStudio uses ChatResponseWithTools
	var toolCalls []ToolCall
	if len(response.Choices) > 0 {
		// Access the extended choice type with tool calls
		choice := response.Choices[0]
		if choice.Message.ToolCalls != nil {
			for _, tc := range choice.Message.ToolCalls {
				toolCall := ToolCall{
					ID:   tc.ID,
					Name: tc.Function.Name,
				}
				
				// Parse arguments - LMStudio returns them as JSON string
				var params map[string]interface{}
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &params); err == nil {
					toolCall.Parameters = params
				} else {
					// If parsing fails, try to use as-is
					toolCall.Parameters = map[string]interface{}{
						"raw": tc.Function.Arguments,
					}
				}
				
				toolCalls = append(toolCalls, toolCall)
			}
		}
	}
	
	// Build response
	content := ""
	if len(response.Choices) > 0 {
		content = response.Choices[0].Message.Content
	}
	
	// Extract finish reason
	finishReason := ""
	if len(response.Choices) > 0 {
		finishReason = response.Choices[0].FinishReason
	}
	
	return &ToolGenerationResponse{
		GenerationResponse: GenerationResponse{
			Content:  content,
			Model:    response.Model,
			Provider: p.name,
			Usage: TokenUsage{
				PromptTokens:     response.Usage.PromptTokens,
				CompletionTokens: response.Usage.CompletionTokens,
				TotalTokens:      response.Usage.TotalTokens,
			},
			Metadata: map[string]interface{}{
				"finish_reason": finishReason,
				"has_tools":     len(toolCalls) > 0,
				"tool_calls":    len(toolCalls),
			},
		},
		ToolCalls: toolCalls,
	}, nil
}


// StreamWithTools generates streaming text with tool calling (placeholder implementation)
func (p *LMStudioProvider) StreamWithTools(ctx context.Context, req *ToolGenerationRequest, callback ToolStreamCallback) error {
	// For now, delegate to regular streaming
	genReq := &GenerationRequest{
		Prompt:      req.Prompt,
		Model:       req.Model,
		Parameters:  req.Parameters,
		Context:     req.Context,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	}

	return p.Stream(ctx, genReq, func(chunk *StreamChunk) {
		toolChunk := &ToolStreamChunk{
			StreamChunk: *chunk,
			ToolCalls:   []ToolCall{}, // No tool calls for now
		}
		callback(toolChunk)
	})
}

// Capabilities returns the provider capabilities
func (p *LMStudioProvider) Capabilities() ProviderCapabilities {
	return ProviderCapabilities{
		SupportsStreaming: true,
		SupportsToolCalls: true,  // LMStudio now supports tool calling
		SupportsBatch:     false,
		MaxTokens:         4096, // Default for most local models
		MaxContextLength:  4096, // Depends on the loaded model
	}
}

// Metadata returns additional provider metadata
func (p *LMStudioProvider) Metadata() map[string]interface{} {
	return map[string]interface{}{
		"provider_type":   "lmstudio",
		"base_url":        p.baseURL,
		"model":           p.config.Model,
		"timeout":         p.config.Timeout,
		"supports_tools":  true,
		"supports_stream": true,
		"local":           true, // LMStudio is typically local
	}
}

// Close closes the provider and cleans up resources
func (p *LMStudioProvider) Close() error {
	// Nothing to clean up for LMStudio client
	return nil
}

// buildMessages builds LMStudio chat messages from the request
func (p *LMStudioProvider) buildMessages(req *GenerationRequest) []lmstudio.Message {
	messages := make([]lmstudio.Message, 0)

	// Add context messages
	for _, msg := range req.Context {
		messages = append(messages, lmstudio.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// Add the main prompt as user message
	if req.Prompt != "" {
		messages = append(messages, lmstudio.Message{
			Role:    "user",
			Content: req.Prompt,
		})
	}

	// If no messages, create a default user message
	if len(messages) == 0 && req.Prompt != "" {
		messages = append(messages, lmstudio.Message{
			Role:    "user",
			Content: req.Prompt,
		})
	}

	return messages
}


