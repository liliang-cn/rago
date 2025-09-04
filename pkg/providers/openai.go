package providers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/liliang-cn/rago/v2/pkg/domain"
	openai "github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
	"github.com/openai/openai-go/v2/shared"
)

// OpenAILLMProvider implements LLMProvider for OpenAI-compatible services
type OpenAILLMProvider struct {
	client openai.Client
	config *domain.OpenAIProviderConfig
}

// NewOpenAILLMProvider creates a new OpenAI LLM provider
func NewOpenAILLMProvider(config *domain.OpenAIProviderConfig) (domain.LLMProvider, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	opts := []option.RequestOption{
		option.WithAPIKey(config.APIKey),
	}

	if config.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(config.BaseURL))
	}

	return &OpenAILLMProvider{
		client: openai.NewClient(opts...),
		config: config,
	}, nil
}

// ProviderType returns the provider type
func (p *OpenAILLMProvider) ProviderType() domain.ProviderType {
	return domain.ProviderOpenAI
}

// toOpenAIMessages converts domain messages to the OpenAI API format
func toOpenAIMessages(messages []domain.Message) ([]openai.ChatCompletionMessageParamUnion, error) {
	openAIMessages := make([]openai.ChatCompletionMessageParamUnion, len(messages))
	for i, msg := range messages {
		switch msg.Role {
		case "user":
			openAIMessages[i] = openai.UserMessage(msg.Content)
		case "system":
			openAIMessages[i] = openai.SystemMessage(msg.Content)
		case "tool":
			openAIMessages[i] = openai.ToolMessage(msg.Content, msg.ToolCallID)
		case "assistant":
			assistantMsg := openai.ChatCompletionMessage{
				Role:    "assistant",
				Content: msg.Content,
			}
			if len(msg.ToolCalls) > 0 {
				assistantMsg.ToolCalls = make([]openai.ChatCompletionMessageToolCallUnion, len(msg.ToolCalls))
				for j, tc := range msg.ToolCalls {
					args, err := json.Marshal(tc.Function.Arguments)
					if err != nil {
						return nil, fmt.Errorf("failed to marshal tool call arguments: %w", err)
					}
					assistantMsg.ToolCalls[j] = openai.ChatCompletionMessageToolCallUnion{
						ID:   tc.ID,
						Type: "function",
						Function: openai.ChatCompletionMessageFunctionToolCallFunction{
							Name:      tc.Function.Name,
							Arguments: string(args),
						},
					}
				}
			}
			openAIMessages[i] = assistantMsg.ToParam()
		default:
			return nil, fmt.Errorf("unknown message role: %s", msg.Role)
		}
	}
	return openAIMessages, nil
}

// Generate generates text using OpenAI API
func (p *OpenAILLMProvider) Generate(ctx context.Context, prompt string, opts *domain.GenerationOptions) (string, error) {
	if prompt == "" {
		return "", fmt.Errorf("%w: empty prompt", domain.ErrInvalidInput)
	}

	messages := []openai.ChatCompletionMessageParamUnion{
		openai.UserMessage(prompt),
	}

	params := openai.ChatCompletionNewParams{
		Model:    shared.ChatModel(p.config.LLMModel),
		Messages: messages,
	}

	if opts != nil {
		if opts.Temperature >= 0 {
			params.Temperature = openai.Float(opts.Temperature)
		}
		if opts.MaxTokens > 0 {
			params.MaxCompletionTokens = openai.Int(int64(opts.MaxTokens))
		}
	}

	completion, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return "", fmt.Errorf("%w: %v", domain.ErrGenerationFailed, err)
	}

	if len(completion.Choices) == 0 {
		return "", fmt.Errorf("%w: no choices returned", domain.ErrGenerationFailed)
	}

	return completion.Choices[0].Message.Content, nil
}

// Stream generates text with streaming using OpenAI API
func (p *OpenAILLMProvider) Stream(ctx context.Context, prompt string, opts *domain.GenerationOptions, callback func(string)) error {
	if prompt == "" {
		return fmt.Errorf("%w: empty prompt", domain.ErrInvalidInput)
	}
	if callback == nil {
		return fmt.Errorf("%w: nil callback", domain.ErrInvalidInput)
	}

	messages := []openai.ChatCompletionMessageParamUnion{
		openai.UserMessage(prompt),
	}

	params := openai.ChatCompletionNewParams{
		Model:    shared.ChatModel(p.config.LLMModel),
		Messages: messages,
	}

	if opts != nil {
		if opts.Temperature >= 0 {
			params.Temperature = openai.Float(opts.Temperature)
		}
		if opts.MaxTokens > 0 {
			params.MaxCompletionTokens = openai.Int(int64(opts.MaxTokens))
		}
	}

	stream := p.client.Chat.Completions.NewStreaming(ctx, params)

	for stream.Next() {
		chunk := stream.Current()
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			callback(chunk.Choices[0].Delta.Content)
		}
	}

	if err := stream.Err(); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrGenerationFailed, err)
	}

	return nil
}

// GenerateWithTools generates text with tool calling support
func (p *OpenAILLMProvider) GenerateWithTools(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions) (*domain.GenerationResult, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("%w: empty messages", domain.ErrInvalidInput)
	}

	openAIMessages, err := toOpenAIMessages(messages)
	if err != nil {
		return nil, fmt.Errorf("failed to convert messages: %w", err)
	}

	params := openai.ChatCompletionNewParams{
		Model:    shared.ChatModel(p.config.LLMModel),
		Messages: openAIMessages,
	}

	if len(tools) > 0 {
		openaiTools := make([]openai.ChatCompletionToolUnionParam, len(tools))
		for i, tool := range tools {
			openaiTools[i] = openai.ChatCompletionFunctionTool(shared.FunctionDefinitionParam{
				Name:        tool.Function.Name,
				Description: openai.String(tool.Function.Description),
				Parameters:  tool.Function.Parameters,
			})
		}
		params.Tools = openaiTools

		// Set tool choice if specified
		if opts != nil && opts.ToolChoice != "" {
			switch opts.ToolChoice {
			case "required":
				// Force the model to call one of the tools
				params.ToolChoice = openai.ToolChoiceOptionFunctionToolChoice(openai.ChatCompletionNamedToolChoiceFunctionParam{
					Name: tools[0].Function.Name, // Use first tool as default
				})
			case "auto", "none":
				// Let OpenAI handle auto/none behavior naturally
				// "auto" is default when tools are present
			default:
				// Assume it's a specific function name
				params.ToolChoice = openai.ToolChoiceOptionFunctionToolChoice(openai.ChatCompletionNamedToolChoiceFunctionParam{
					Name: opts.ToolChoice,
				})
			}
		}
	}

	if opts != nil {
		if opts.Temperature >= 0 {
			params.Temperature = openai.Float(opts.Temperature)
		}
		if opts.MaxTokens > 0 {
			params.MaxCompletionTokens = openai.Int(int64(opts.MaxTokens))
		}
	}

	completion, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrGenerationFailed, err)
	}

	if len(completion.Choices) == 0 {
		return nil, fmt.Errorf("%w: no choices returned", domain.ErrGenerationFailed)
	}

	choice := completion.Choices[0]
	result := &domain.GenerationResult{
		Content:  choice.Message.Content,
		Finished: true,
	}

	if len(choice.Message.ToolCalls) > 0 {
		result.ToolCalls = make([]domain.ToolCall, len(choice.Message.ToolCalls))
		for i, tc := range choice.Message.ToolCalls {
			var args map[string]interface{}
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
				return nil, fmt.Errorf("failed to parse tool call arguments: %w", err)
			}
			result.ToolCalls[i] = domain.ToolCall{
				ID:   tc.ID,
				Type: tc.Type,
				Function: domain.FunctionCall{
					Name:      tc.Function.Name,
					Arguments: args,
				},
			}
		}
	}

	return result, nil
}

// StreamWithTools generates text with tool calling support in streaming mode
func (p *OpenAILLMProvider) StreamWithTools(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions, callback domain.ToolCallCallback) error {
	if len(messages) == 0 {
		return fmt.Errorf("%w: empty messages", domain.ErrInvalidInput)
	}
	if callback == nil {
		return fmt.Errorf("%w: nil callback", domain.ErrInvalidInput)
	}

	openAIMessages, err := toOpenAIMessages(messages)
	if err != nil {
		return fmt.Errorf("failed to convert messages: %w", err)
	}

	params := openai.ChatCompletionNewParams{
		Model:    shared.ChatModel(p.config.LLMModel),
		Messages: openAIMessages,
	}

	if len(tools) > 0 {
		openaiTools := make([]openai.ChatCompletionToolUnionParam, len(tools))
		for i, tool := range tools {
			openaiTools[i] = openai.ChatCompletionFunctionTool(shared.FunctionDefinitionParam{
				Name:        tool.Function.Name,
				Description: openai.String(tool.Function.Description),
				Parameters:  tool.Function.Parameters,
			})
		}
		params.Tools = openaiTools

		// Set tool choice if specified
		if opts != nil && opts.ToolChoice != "" {
			switch opts.ToolChoice {
			case "required":
				// Force the model to call one of the tools
				params.ToolChoice = openai.ToolChoiceOptionFunctionToolChoice(openai.ChatCompletionNamedToolChoiceFunctionParam{
					Name: tools[0].Function.Name, // Use first tool as default
				})
			case "auto", "none":
				// Let OpenAI handle auto/none behavior naturally
				// "auto" is default when tools are present
			default:
				// Assume it's a specific function name
				params.ToolChoice = openai.ToolChoiceOptionFunctionToolChoice(openai.ChatCompletionNamedToolChoiceFunctionParam{
					Name: opts.ToolChoice,
				})
			}
		}
	}

	if opts != nil {
		if opts.Temperature >= 0 {
			params.Temperature = openai.Float(opts.Temperature)
		}
		if opts.MaxTokens > 0 {
			params.MaxCompletionTokens = openai.Int(int64(opts.MaxTokens))
		}
	}

	stream := p.client.Chat.Completions.NewStreaming(ctx, params)

	for stream.Next() {
		chunk := stream.Current()
		if len(chunk.Choices) == 0 {
			continue
		}

		choice := chunk.Choices[0]
		var toolCalls []domain.ToolCall

		if len(choice.Delta.ToolCalls) > 0 {
			toolCalls = make([]domain.ToolCall, len(choice.Delta.ToolCalls))
			for i, tc := range choice.Delta.ToolCalls {
				var args map[string]interface{}
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
					args = make(map[string]interface{})
				}
				toolCalls[i] = domain.ToolCall{
					ID:   tc.ID,
					Type: "function",
					Function: domain.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: args,
					},
				}
			}
		}

		if err := callback(choice.Delta.Content, toolCalls); err != nil {
			return fmt.Errorf("callback error: %w", err)
		}
	}

	if err := stream.Err(); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrGenerationFailed, err)
	}

	return nil
}

// GenerateStructured implements structured JSON output generation for OpenAI using native structured output
func (p *OpenAILLMProvider) GenerateStructured(ctx context.Context, prompt string, schema interface{}, opts *domain.GenerationOptions) (*domain.StructuredResult, error) {
	if err := ValidateStructuredRequest(prompt, schema); err != nil {
		return nil, err
	}

	if opts == nil {
		opts = DefaultStructuredOptions()
	}
	if err := ValidateGenerationOptions(opts); err != nil {
		return nil, err
	}

	messages := []openai.ChatCompletionMessageParamUnion{
		openai.UserMessage(prompt),
	}

	// Use OpenAI's native structured output with ResponseFormat
	schemaParam := openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        "structured_response",
		Description: openai.String("Structured response based on the provided schema"),
		Schema:      schema,
		Strict:      openai.Bool(true),
	}

	params := openai.ChatCompletionNewParams{
		Model:    shared.ChatModel(p.config.LLMModel),
		Messages: messages,
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{JSONSchema: schemaParam},
		},
	}

	if opts.Temperature >= 0 {
		params.Temperature = openai.Float(opts.Temperature)
	}
	if opts.MaxTokens > 0 {
		params.MaxCompletionTokens = openai.Int(int64(opts.MaxTokens))
	}

	resp, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, WrapStructuredOutputError(domain.ProviderOpenAI, err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response choices returned")
	}

	rawJSON := resp.Choices[0].Message.Content

	// Try to parse the JSON into the provided schema
	var isValid bool
	if err := json.Unmarshal([]byte(rawJSON), schema); err == nil {
		isValid = true
	}

	return &domain.StructuredResult{
		Data:  schema,
		Raw:   rawJSON,
		Valid: isValid,
	}, nil
}

// Health checks the health of the OpenAI service
func (p *OpenAILLMProvider) Health(ctx context.Context) error {
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.UserMessage("Hello"),
	}

	params := openai.ChatCompletionNewParams{
		Model:               shared.ChatModel(p.config.LLMModel),
		Messages:            messages,
		MaxCompletionTokens: openai.Int(1),
	}

	_, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrServiceUnavailable, err)
	}

	return nil
}

// ExtractMetadata extracts metadata from content using OpenAI API
func (p *OpenAILLMProvider) ExtractMetadata(ctx context.Context, content string, model string) (*domain.ExtractedMetadata, error) {
	if content == "" {
		return nil, fmt.Errorf("%w: content cannot be empty", domain.ErrInvalidInput)
	}

	prompt := fmt.Sprintf(`You are an expert data extractor. Analyze the following document content and return ONLY a single valid JSON object with the following fields:
- "summary": A concise, one-sentence summary of the document.
- "keywords": An array of 3 to 5 most relevant keywords.
- "document_type": The type of the document (e.g., "Article", "Meeting Notes", "Technical Manual", "Code Snippet", "Essay").
- "creation_date": The creation date found in the document text in "YYYY-MM-DD" format. If no date is found, use null.

Document Content:
"""
%s
"""

JSON Output:`, content)

	messages := []openai.ChatCompletionMessageParamUnion{
		openai.UserMessage(prompt),
	}

	llmModel := p.config.LLMModel
	if model != "" {
		llmModel = model
	}

	params := openai.ChatCompletionNewParams{
		Model:    shared.ChatModel(llmModel),
		Messages: messages,
	}

	completion, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("%w: metadata extraction failed: %v", domain.ErrGenerationFailed, err)
	}

	if len(completion.Choices) == 0 {
		return nil, fmt.Errorf("%w: no choices returned for metadata extraction", domain.ErrGenerationFailed)
	}

	var metadata domain.ExtractedMetadata
	if err := json.Unmarshal([]byte(completion.Choices[0].Message.Content), &metadata); err != nil {
		return nil, fmt.Errorf("%w: failed to unmarshal metadata response: %v. Raw response: %s",
			domain.ErrInvalidInput, err, completion.Choices[0].Message.Content)
	}

	return &metadata, nil
}

// OpenAIEmbedderProvider implements EmbedderProvider for OpenAI-compatible services
type OpenAIEmbedderProvider struct {
	client openai.Client
	config *domain.OpenAIProviderConfig
}

// NewOpenAIEmbedderProvider creates a new OpenAI embedder provider
func NewOpenAIEmbedderProvider(config *domain.OpenAIProviderConfig) (domain.EmbedderProvider, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	opts := []option.RequestOption{
		option.WithAPIKey(config.APIKey),
	}

	if config.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(config.BaseURL))
	}

	return &OpenAIEmbedderProvider{
		client: openai.NewClient(opts...),
		config: config,
	}, nil
}

// ProviderType returns the provider type
func (p *OpenAIEmbedderProvider) ProviderType() domain.ProviderType {
	return domain.ProviderOpenAI
}

// Embed generates embeddings using OpenAI API
func (p *OpenAIEmbedderProvider) Embed(ctx context.Context, text string) ([]float64, error) {
	if text == "" {
		return nil, fmt.Errorf("%w: empty text", domain.ErrInvalidInput)
	}

	params := openai.EmbeddingNewParams{
		Model: openai.EmbeddingModel(p.config.EmbeddingModel),
		Input: openai.EmbeddingNewParamsInputUnion{
			OfString: openai.String(text),
		},
	}

	embedding, err := p.client.Embeddings.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrEmbeddingFailed, err)
	}

	if len(embedding.Data) == 0 {
		return nil, fmt.Errorf("%w: no embedding data returned", domain.ErrEmbeddingFailed)
	}

	vec64 := make([]float64, len(embedding.Data[0].Embedding))
	for i, v := range embedding.Data[0].Embedding {
		vec64[i] = float64(v)
	}

	return vec64, nil
}

// Health checks the health of the OpenAI embeddings service
func (p *OpenAIEmbedderProvider) Health(ctx context.Context) error {
	params := openai.EmbeddingNewParams{
		Model: openai.EmbeddingModel(p.config.EmbeddingModel),
		Input: openai.EmbeddingNewParamsInputUnion{
			OfString: openai.String("test"),
		},
	}

	_, err := p.client.Embeddings.New(ctx, params)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrServiceUnavailable, err)
	}

	return nil
}
