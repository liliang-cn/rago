package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/liliang-cn/ollama-go"
	"github.com/liliang-cn/rago/internal/domain"
)

// OllamaLLMProvider wraps the existing Ollama LLM service as a provider
type OllamaLLMProvider struct {
	client  *ollama.Client
	config  *domain.OllamaProviderConfig
}

// NewOllamaLLMProvider creates a new Ollama LLM provider
func NewOllamaLLMProvider(config *domain.OllamaProviderConfig) (domain.LLMProvider, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	client, err := ollama.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create ollama client: %w", err)
	}

	return &OllamaLLMProvider{
		client: client,
		config: config,
	}, nil
}

// ProviderType returns the provider type
func (p *OllamaLLMProvider) ProviderType() domain.ProviderType {
	return domain.ProviderOllama
}

// Generate generates text using Ollama
func (p *OllamaLLMProvider) Generate(ctx context.Context, prompt string, opts *domain.GenerationOptions) (string, error) {
	if prompt == "" {
		return "", fmt.Errorf("%w: empty prompt", domain.ErrInvalidInput)
	}

	stream := false
	req := &ollama.GenerateRequest{
		Model:  p.config.LLMModel,
		Prompt: prompt,
		Stream: &stream,
	}

	if opts != nil {
		options := &ollama.Options{}
		if opts.Temperature >= 0 {
			options.Temperature = &opts.Temperature
		}
		if opts.MaxTokens > 0 {
			numPredict := opts.MaxTokens
			options.NumPredict = &numPredict
		}
		req.Options = options

		if opts.Think != nil {
			req.Think = opts.Think
		}
	}

	resp, err := p.client.Generate(ctx, req)
	if err != nil {
		return "", fmt.Errorf("%w: %v", domain.ErrGenerationFailed, err)
	}

	return resp.Response, nil
}

// Stream generates text with streaming using Ollama
func (p *OllamaLLMProvider) Stream(ctx context.Context, prompt string, opts *domain.GenerationOptions, callback func(string)) error {
	if prompt == "" {
		return fmt.Errorf("%w: empty prompt", domain.ErrInvalidInput)
	}

	if callback == nil {
		return fmt.Errorf("%w: nil callback", domain.ErrInvalidInput)
	}

	// Use the functional API like in your working example
	options := []func(*ollama.GenerateRequest){}

	if opts != nil {
		if opts.Temperature >= 0 {
			options = append(options, func(req *ollama.GenerateRequest) {
				if req.Options == nil {
					req.Options = &ollama.Options{}
				}
				req.Options.Temperature = &opts.Temperature
			})
		}
		if opts.MaxTokens > 0 {
			options = append(options, func(req *ollama.GenerateRequest) {
				if req.Options == nil {
					req.Options = &ollama.Options{}
				}
				req.Options.NumPredict = &opts.MaxTokens
			})
		}
	}

	// Use the functional API that works correctly
	respCh, errCh := ollama.GenerateStream(ctx, p.config.LLMModel, prompt, options...)

	for {
		select {
		case resp, ok := <-respCh:
			if !ok {
				// Channel closed, streaming is done
				return nil
			}
			if resp != nil && resp.Response != "" {
				callback(resp.Response)
			}
		case err := <-errCh:
			if err != nil {
				return fmt.Errorf("%w: %v", domain.ErrGenerationFailed, err)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Health checks the health of the Ollama service
func (p *OllamaLLMProvider) Health(ctx context.Context) error {
	_, err := p.client.Version(ctx)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrServiceUnavailable, err)
	}
	return nil
}

const metadataExtractionPromptTemplate = `You are an expert data extractor. Analyze the following document content and return ONLY a single valid JSON object with the following fields:
- "summary": A concise, one-sentence summary of the document.
- "keywords": An array of 3 to 5 most relevant keywords.
- "document_type": The type of the document (e.g., "Article", "Meeting Notes", "Technical Manual", "Code Snippet", "Essay").
- "creation_date": The creation date found in the document text in "YYYY-MM-DD" format. If no date is found, use null.

Document Content:
"""
%s
"""

JSON Output:`

// ExtractMetadata extracts metadata from content using Ollama
func (p *OllamaLLMProvider) ExtractMetadata(ctx context.Context, content string, model string) (*domain.ExtractedMetadata, error) {
	if content == "" {
		return nil, fmt.Errorf("%w: content cannot be empty", domain.ErrInvalidInput)
	}

	prompt := fmt.Sprintf(metadataExtractionPromptTemplate, content)

	// Use the specified model if provided, otherwise use the default LLM model
	llmModel := p.config.LLMModel
	if model != "" {
		llmModel = model
	}

	stream := false
	format := "json"
	req := &ollama.GenerateRequest{
		Model:  llmModel,
		Prompt: prompt,
		Stream: &stream,
		Format: &format,
	}

	resp, err := p.client.Generate(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%w: metadata extraction failed: %v", domain.ErrGenerationFailed, err)
	}

	var metadata domain.ExtractedMetadata
	if err := json.Unmarshal([]byte(resp.Response), &metadata); err != nil {
		return nil, fmt.Errorf("%w: failed to unmarshal metadata response: %v. Raw response: %s", domain.ErrInvalidInput, err, resp.Response)
	}

	return &metadata, nil
}

// isAlmostSamePromptTemplate is the prompt template used to determine if input and output are essentially the same
const isAlmostSamePromptTemplate = `You are an expert judge evaluating whether two pieces of text represent the same information. 
Please compare the original input and the generated output and determine if they convey the same core meaning.

Respond with ONLY "true" if they are essentially the same, or "false" if they are different.

Original input: "%s"
Generated output: "%s"

Are these essentially the same? Respond with only "true" or "false":`

// IsAlmostSame determines if the input and output are essentially the same using LLM
func (p *OllamaLLMProvider) IsAlmostSame(ctx context.Context, input, output string) (bool, error) {
	if input == "" || output == "" {
		return false, nil
	}

	prompt := fmt.Sprintf(isAlmostSamePromptTemplate, input, output)

	stream := false
	req := &ollama.GenerateRequest{
		Model:  p.config.LLMModel,
		Prompt: prompt,
		Stream: &stream,
	}

	resp, err := p.client.Generate(ctx, req)
	if err != nil {
		return false, fmt.Errorf("failed to generate similarity judgment: %w", err)
	}

	// Parse the response as a boolean
	result := strings.TrimSpace(strings.ToLower(resp.Response))

	// Handle cases where the model might return "true" or "false" with extra text
	if strings.Contains(result, "true") {
		return true, nil
	}

	if strings.Contains(result, "false") {
		return false, nil
	}

	// Default to false if we can't determine
	return false, nil
}

// GenerateWithTools generates text with tool calling support using Ollama
func (p *OllamaLLMProvider) GenerateWithTools(ctx context.Context, prompt string, tools []domain.ToolDefinition, opts *domain.GenerationOptions) (*domain.GenerationResult, error) {
	if prompt == "" {
		return nil, fmt.Errorf("%w: empty prompt", domain.ErrInvalidInput)
	}

	// Convert domain.ToolDefinition to ollama.Tool
	ollamaTools := make([]ollama.Tool, len(tools))
	for i, tool := range tools {
		ollamaTools[i] = ollama.Tool{
			Type: tool.Type,
			Function: &ollama.ToolFunction{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Parameters:  tool.Function.Parameters,
			},
		}
	}

	// Prepare messages
	messages := []ollama.Message{
		{
			Role:    "user",
			Content: prompt,
		},
	}

	// Create chat request
	req := &ollama.ChatRequest{
		Model:    p.config.LLMModel,
		Messages: messages,
		Tools:    ollamaTools,
		Stream:   new(bool), // false for non-streaming
	}

	// Apply generation options
	if opts != nil {
		options := &ollama.Options{}
		if opts.Temperature >= 0 {
			options.Temperature = &opts.Temperature
		}
		if opts.MaxTokens > 0 {
			numPredict := opts.MaxTokens
			options.NumPredict = &numPredict
		}
		req.Options = options
	}

	// Make the request
	resp, err := p.client.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrGenerationFailed, err)
	}

	// Convert response
	result := &domain.GenerationResult{
		Content:  resp.Message.Content,
		Finished: true,
	}

	// Convert tool calls if any
	if len(resp.Message.ToolCalls) > 0 {
		result.ToolCalls = make([]domain.ToolCall, len(resp.Message.ToolCalls))
		for i, tc := range resp.Message.ToolCalls {
			result.ToolCalls[i] = domain.ToolCall{
				ID:   tc.Function.Name, // Use function name as ID if no ID provided
				Type: "function",
				Function: domain.FunctionCall{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			}
		}
	}

	return result, nil
}

// StreamWithTools generates text with tool calling support in streaming mode using Ollama
func (p *OllamaLLMProvider) StreamWithTools(ctx context.Context, prompt string, tools []domain.ToolDefinition, opts *domain.GenerationOptions, callback domain.ToolCallCallback) error {
	if prompt == "" {
		return fmt.Errorf("%w: empty prompt", domain.ErrInvalidInput)
	}

	if callback == nil {
		return fmt.Errorf("%w: nil callback", domain.ErrInvalidInput)
	}

	// Convert domain.ToolDefinition to ollama.Tool
	ollamaTools := make([]ollama.Tool, len(tools))
	for i, tool := range tools {
		ollamaTools[i] = ollama.Tool{
			Type: tool.Type,
			Function: &ollama.ToolFunction{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Parameters:  tool.Function.Parameters,
			},
		}
	}

	// Prepare messages
	messages := []ollama.Message{
		{
			Role:    "user",
			Content: prompt,
		},
	}

	// Create chat request
	req := &ollama.ChatRequest{
		Model:    p.config.LLMModel,
		Messages: messages,
		Tools:    ollamaTools,
		Stream:   func() *bool { b := true; return &b }(), // true for streaming
	}

	// Apply generation options
	if opts != nil {
		options := &ollama.Options{}
		if opts.Temperature >= 0 {
			options.Temperature = &opts.Temperature
		}
		if opts.MaxTokens > 0 {
			numPredict := opts.MaxTokens
			options.NumPredict = &numPredict
		}
		req.Options = options
	}

	// Make streaming request
	respCh, errCh := ollama.ChatStream(ctx, p.config.LLMModel, messages, func(r *ollama.ChatRequest) {
		r.Tools = ollamaTools
		if req.Options != nil {
			r.Options = req.Options
		}
	})

	for {
		select {
		case resp, ok := <-respCh:
			if !ok {
				// Channel closed, streaming is done
				return nil
			}
			if resp != nil {
				// Convert tool calls if any
				var toolCalls []domain.ToolCall
				if len(resp.Message.ToolCalls) > 0 {
					toolCalls = make([]domain.ToolCall, len(resp.Message.ToolCalls))
					for i, tc := range resp.Message.ToolCalls {
						toolCalls[i] = domain.ToolCall{
							ID:   tc.Function.Name, // Use function name as ID if no ID provided
							Type: "function",
							Function: domain.FunctionCall{
								Name:      tc.Function.Name,
								Arguments: tc.Function.Arguments,
							},
						}
					}
				}

				// Call the callback with content and tool calls
				if err := callback(resp.Message.Content, toolCalls); err != nil {
					return fmt.Errorf("callback error: %w", err)
				}
			}
		case err := <-errCh:
			if err != nil {
				return fmt.Errorf("%w: %v", domain.ErrGenerationFailed, err)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// OllamaEmbedderProvider wraps the existing Ollama embedder service as a provider
type OllamaEmbedderProvider struct {
	client *ollama.Client
	config *domain.OllamaProviderConfig
}

// NewOllamaEmbedderProvider creates a new Ollama embedder provider
func NewOllamaEmbedderProvider(config *domain.OllamaProviderConfig) (domain.EmbedderProvider, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	client, err := ollama.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create ollama client: %w", err)
	}

	return &OllamaEmbedderProvider{
		client: client,
		config: config,
	}, nil
}

// ProviderType returns the provider type
func (p *OllamaEmbedderProvider) ProviderType() domain.ProviderType {
	return domain.ProviderOllama
}

// Embed generates embeddings using Ollama
func (p *OllamaEmbedderProvider) Embed(ctx context.Context, text string) ([]float64, error) {
	if text == "" {
		return nil, fmt.Errorf("%w: empty text", domain.ErrInvalidInput)
	}

	req := &ollama.EmbedRequest{
		Model: p.config.EmbeddingModel,
		Input: text,
	}

	resp, err := p.client.Embed(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrEmbeddingFailed, err)
	}

	if len(resp.Embeddings) == 0 || len(resp.Embeddings[0]) == 0 {
		return nil, fmt.Errorf("%w: empty embedding response", domain.ErrEmbeddingFailed)
	}

	return resp.Embeddings[0], nil
}

// Health checks the health of the Ollama embeddings service
func (p *OllamaEmbedderProvider) Health(ctx context.Context) error {
	_, err := p.client.Version(ctx)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrServiceUnavailable, err)
	}
	return nil
}