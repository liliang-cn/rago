package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/liliang-cn/lmstudio-go"
	"github.com/liliang-cn/rago/pkg/domain"
)

// LMStudioLLMProvider implements the LLMProvider interface using LM Studio
type LMStudioLLMProvider struct {
	client       *lmstudio.Client
	config       *domain.LMStudioProviderConfig
	llmModel     string
	providerType domain.ProviderType
}

// NewLMStudioLLMProvider creates a new LM Studio LLM provider
func NewLMStudioLLMProvider(config *domain.LMStudioProviderConfig) (*LMStudioLLMProvider, error) {
	if config == nil {
		return nil, fmt.Errorf("LM Studio provider config is nil")
	}

	if config.BaseURL == "" {
		config.BaseURL = "http://localhost:1234" // Default LM Studio port
	}

	if config.LLMModel == "" {
		return nil, fmt.Errorf("LLM model is required for LM Studio provider")
	}

	// Create LM Studio client with custom config
	clientConfig := &lmstudio.Config{
		BaseURL: config.BaseURL,
		Timeout: config.Timeout,
	}
	client := lmstudio.NewClientWithConfig(clientConfig)

	return &LMStudioLLMProvider{
		client:       client,
		config:       config,
		llmModel:     config.LLMModel,
		providerType: domain.ProviderLMStudio,
	}, nil
}

// Generate implements the Generator interface for single-turn generation
func (p *LMStudioLLMProvider) Generate(ctx context.Context, prompt string, opts *domain.GenerationOptions) (string, error) {
	if opts == nil {
		opts = &domain.GenerationOptions{
			Temperature: 0.7,
			MaxTokens:   4000,
		}
	}

	response, err := p.client.Chat.SimpleChat(ctx, p.llmModel, prompt)
	if err != nil {
		return "", fmt.Errorf("LM Studio completion failed: %w", err)
	}

	return response, nil
}

// GenerateWithTools implements the Generator interface for tool-enabled generation
func (p *LMStudioLLMProvider) GenerateWithTools(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions) (*domain.GenerationResult, error) {
	if opts == nil {
		opts = &domain.GenerationOptions{
			Temperature: 0.7,
			MaxTokens:   4000,
		}
	}

	// Convert messages to LM Studio format
	chatMessages := make([]lmstudio.Message, len(messages))
	for i, msg := range messages {
		chatMessages[i] = lmstudio.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	req := &lmstudio.ChatRequest{
		Model:       p.llmModel,
		Messages:    chatMessages,
		MaxTokens:   &opts.MaxTokens,
		Temperature: &opts.Temperature,
	}

	// Convert tools to LM Studio format if provided
	if len(tools) > 0 {
		lmTools := make([]lmstudio.Tool, len(tools))
		for i, tool := range tools {
			lmTools[i] = lmstudio.Tool{
				Type: "function",
				Function: lmstudio.ToolFunction{
					Name:        tool.Function.Name,
					Description: tool.Function.Description,
					Parameters:  tool.Function.Parameters,
				},
			}
		}

		toolsReq := &lmstudio.ChatRequestWithTools{
			ChatRequest: *req,
			Tools:       lmTools,
		}

		resp, err := p.client.Chat.CompleteWithTools(ctx, toolsReq)
		if err != nil {
			return nil, fmt.Errorf("LM Studio chat completion with tools failed: %w", err)
		}

		result := &domain.GenerationResult{
			Content: resp.Choices[0].Message.Content,
		}

		// Handle tool calls if present
		if len(resp.Choices[0].Message.ToolCalls) > 0 {
			toolCalls := make([]domain.ToolCall, len(resp.Choices[0].Message.ToolCalls))
			for i, tc := range resp.Choices[0].Message.ToolCalls {
				// Parse the JSON arguments string into a map
				var args map[string]interface{}
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
					return nil, fmt.Errorf("failed to parse tool call arguments: %w", err)
				}

				toolCalls[i] = domain.ToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: domain.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: args,
					},
				}
			}
			result.ToolCalls = toolCalls
		}

		return result, nil
	}

	// Regular chat without tools
	resp, err := p.client.Chat.Complete(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LM Studio chat completion failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no completion choices returned from LM Studio")
	}

	result := &domain.GenerationResult{
		Content: resp.Choices[0].Message.Content,
	}

	return result, nil
}

// Stream implements streaming generation
func (p *LMStudioLLMProvider) Stream(ctx context.Context, prompt string, opts *domain.GenerationOptions, callback func(string)) error {
	if opts == nil {
		opts = &domain.GenerationOptions{
			Temperature: 0.7,
			MaxTokens:   4000,
		}
	}

	req := &lmstudio.ChatRequest{
		Model:       p.llmModel,
		Messages:    []lmstudio.Message{{Role: "user", Content: prompt}},
		MaxTokens:   &opts.MaxTokens,
		Temperature: &opts.Temperature,
	}

	streamResp, err := p.client.Chat.Stream(ctx, req)
	if err != nil {
		return fmt.Errorf("LM Studio stream failed: %w", err)
	}

	// Read from the response channel
	for {
		select {
		case chunk, ok := <-streamResp.Response:
			if !ok {
				return nil // Stream completed normally
			}
			if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != nil && *chunk.Choices[0].Delta.Content != "" {
				callback(*chunk.Choices[0].Delta.Content)
			}
		case err := <-streamResp.Errors:
			if err != nil {
				return fmt.Errorf("stream error: %w", err)
			}
		case <-ctx.Done():
			streamResp.Close()
			return ctx.Err()
		}
	}
}

// StreamWithTools implements streaming generation with tools
func (p *LMStudioLLMProvider) StreamWithTools(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions, callback domain.ToolCallCallback) error {
	if opts == nil {
		opts = &domain.GenerationOptions{
			Temperature: 0.7,
			MaxTokens:   4000,
		}
	}

	// Convert messages to LM Studio format
	chatMessages := make([]lmstudio.Message, len(messages))
	for i, msg := range messages {
		chatMessages[i] = lmstudio.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	req := &lmstudio.ChatRequest{
		Model:       p.llmModel,
		Messages:    chatMessages,
		MaxTokens:   &opts.MaxTokens,
		Temperature: &opts.Temperature,
	}

	if len(tools) > 0 {
		lmTools := make([]lmstudio.Tool, len(tools))
		for i, tool := range tools {
			lmTools[i] = lmstudio.Tool{
				Type: "function",
				Function: lmstudio.ToolFunction{
					Name:        tool.Function.Name,
					Description: tool.Function.Description,
					Parameters:  tool.Function.Parameters,
				},
			}
		}

		// For tools, we'll use regular Complete and then callback once
		// since LM Studio's streaming with tools may not be available in the same way
		toolsReq := &lmstudio.ChatRequestWithTools{
			ChatRequest: *req,
			Tools:       lmTools,
		}

		resp, err := p.client.Chat.CompleteWithTools(ctx, toolsReq)
		if err != nil {
			return fmt.Errorf("LM Studio chat completion with tools failed: %w", err)
		}

		if len(resp.Choices) > 0 {
			choice := resp.Choices[0]
			content := choice.Message.Content

			// First, stream the content in chunks if there is any
			if content != "" {
				// Simulate streaming by breaking content into words
				words := strings.Fields(content)
				for _, word := range words {
					if err := callback(word+" ", nil); err != nil {
						return err
					}
				}
			}

			// Handle tool calls if present - send them at the end
			var toolCalls []domain.ToolCall
			if len(choice.Message.ToolCalls) > 0 {
				toolCalls = make([]domain.ToolCall, len(choice.Message.ToolCalls))
				for i, tc := range choice.Message.ToolCalls {
					// Parse the JSON arguments string
					var args map[string]interface{}
					if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
						return fmt.Errorf("failed to parse tool call arguments: %w", err)
					}

					toolCalls[i] = domain.ToolCall{
						ID:   tc.ID,
						Type: tc.Type,
						Function: domain.FunctionCall{
							Name:      tc.Function.Name,
							Arguments: args,
						},
					}
				}

				// Send tool calls with empty content
				if err := callback("", toolCalls); err != nil {
					return err
				}
			}
		}
	} else {
		// Stream without tools
		streamResp, err := p.client.Chat.Stream(ctx, req)
		if err != nil {
			return fmt.Errorf("LM Studio stream failed: %w", err)
		}

		// Read from the response channel
		for {
			select {
			case chunk, ok := <-streamResp.Response:
				if !ok {
					return nil // Stream completed normally
				}
				if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != nil && *chunk.Choices[0].Delta.Content != "" {
					if err := callback(*chunk.Choices[0].Delta.Content, nil); err != nil {
						streamResp.Close()
						return err
					}
				}
			case err := <-streamResp.Errors:
				if err != nil {
					return fmt.Errorf("stream error: %w", err)
				}
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return nil
}

// ProviderType returns the provider type
func (p *LMStudioLLMProvider) ProviderType() domain.ProviderType {
	return p.providerType
}

// GenerateStructured implements structured JSON output generation using LMStudio's native structured output
func (p *LMStudioLLMProvider) GenerateStructured(ctx context.Context, prompt string, schema interface{}, opts *domain.GenerationOptions) (*domain.StructuredResult, error) {
	if err := ValidateStructuredRequest(prompt, schema); err != nil {
		return nil, err
	}

	if opts == nil {
		opts = DefaultStructuredOptions()
	}
	if err := ValidateGenerationOptions(opts); err != nil {
		return nil, err
	}

	// Create messages for structured generation
	messages := []lmstudio.Message{
		{Role: "user", Content: prompt},
	}

	// Use LMStudio's native CompleteWithSchema for structured output
	structuredResp, err := p.client.Chat.CompleteWithSchema(
		ctx,
		p.llmModel,
		messages,
		schema,
	)
	if err != nil {
		return nil, WrapStructuredOutputError(p.providerType, err)
	}

	return &domain.StructuredResult{
		Data:  structuredResp.Parsed, // Use the parsed structured data
		Raw:   "", // LMStudio doesn't provide raw JSON in this format
		Valid: true, // CompleteWithSchema ensures validity
	}, nil
}

// Health checks the health of the LM Studio provider
func (p *LMStudioLLMProvider) Health(ctx context.Context) error {
	// Test the actual configured model with a strict test
	response, err := p.client.Chat.SimpleChat(ctx, p.llmModel, "You must respond with exactly 'This is a test' and nothing else. Do not add any additional words, explanations, or punctuation.")
	if err != nil {
		return fmt.Errorf("LLM model health check failed: %w", err)
	}
	
	// Check if we got exactly the expected response
	if response == "" {
		return fmt.Errorf("LLM model health check failed: empty response from model %s", p.llmModel)
	}
	
	// Trim whitespace and check for exact match
	trimmedResponse := strings.TrimSpace(response)
	expectedResponse := "This is a test"
	if trimmedResponse != expectedResponse {
		return fmt.Errorf("LLM model health check failed: model %s did not respond correctly. Expected: %q, Got: %q", p.llmModel, expectedResponse, trimmedResponse)
	}
	
	return nil
}

// ExtractMetadata extracts metadata from content using the LLM
func (p *LMStudioLLMProvider) ExtractMetadata(ctx context.Context, content string, model string) (*domain.ExtractedMetadata, error) {
	// Use the configured model if no specific model is provided
	if model == "" {
		model = p.llmModel
	}

	prompt := fmt.Sprintf(`Please extract metadata from the following content and return it in JSON format with the following fields:
- title: The main title or subject
- summary: A brief summary (max 200 characters)
- topics: An array of main topics/themes
- entities: An array of important named entities (people, places, organizations)
- language: The detected language code (e.g., "en", "zh", "ja")
- content_type: The type of content (e.g., "article", "documentation", "code", "email")

Content:
%s

Return only the JSON object:`, content)

	response, err := p.client.Chat.SimpleChat(ctx, model, prompt)
	if err != nil {
		return nil, fmt.Errorf("metadata extraction failed: %w", err)
	}

	responseText := strings.TrimSpace(response)

	// Parse the JSON response (you may want to add proper JSON parsing here)
	metadata := &domain.ExtractedMetadata{
		Summary: responseText[:min(200, len(responseText))],
	}

	return metadata, nil
}

// LMStudioEmbedderProvider implements the EmbedderProvider interface using LM Studio
type LMStudioEmbedderProvider struct {
	client         *lmstudio.Client
	config         *domain.LMStudioProviderConfig
	embeddingModel string
	providerType   domain.ProviderType
}

// NewLMStudioEmbedderProvider creates a new LM Studio embedder provider
func NewLMStudioEmbedderProvider(config *domain.LMStudioProviderConfig) (*LMStudioEmbedderProvider, error) {
	if config == nil {
		return nil, fmt.Errorf("LM Studio provider config is nil")
	}

	if config.BaseURL == "" {
		config.BaseURL = "http://localhost:1234" // Default LM Studio port
	}

	if config.EmbeddingModel == "" {
		return nil, fmt.Errorf("embedding model is required for LM Studio provider")
	}

	// Create LM Studio client with custom config
	clientConfig := &lmstudio.Config{
		BaseURL: config.BaseURL,
		Timeout: config.Timeout,
	}
	client := lmstudio.NewClientWithConfig(clientConfig)

	return &LMStudioEmbedderProvider{
		client:         client,
		config:         config,
		embeddingModel: config.EmbeddingModel,
		providerType:   domain.ProviderLMStudio,
	}, nil
}

// Embed generates embeddings for the given text
func (p *LMStudioEmbedderProvider) Embed(ctx context.Context, text string) ([]float64, error) {
	req := &lmstudio.EmbeddingRequest{
		Model: p.embeddingModel,
		Input: []string{text},
	}

	resp, err := p.client.Embeddings.Create(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LM Studio embedding failed: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data returned from LM Studio")
	}

	return resp.Data[0].Embedding, nil
}

// ProviderType returns the provider type
func (p *LMStudioEmbedderProvider) ProviderType() domain.ProviderType {
	return p.providerType
}

// Health checks the health of the LM Studio embedder provider
func (p *LMStudioEmbedderProvider) Health(ctx context.Context) error {
	// Test the actual configured embedding model with a simple test
	req := &lmstudio.EmbeddingRequest{
		Model: p.embeddingModel,
		Input: []string{"test"},
	}

	resp, err := p.client.Embeddings.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("embedding model health check failed: %w", err)
	}

	// Check if we got a reasonable embedding response
	if resp == nil || len(resp.Data) == 0 || len(resp.Data[0].Embedding) == 0 {
		return fmt.Errorf("embedding model health check failed: empty embedding from model %s", p.embeddingModel)
	}

	return nil
}

// ExtractMetadata implements the MetadataExtractor interface (placeholder)
func (p *LMStudioEmbedderProvider) ExtractMetadata(ctx context.Context, content string, model string) (*domain.ExtractedMetadata, error) {
	// Embedder providers typically don't extract metadata, return a basic metadata
	return &domain.ExtractedMetadata{
		Summary: content[:min(200, len(content))],
	}, nil
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
