package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/liliang-cn/ollama-go"
	"github.com/liliang-cn/rago/v2/pkg/domain"
)


// OllamaLLMProvider wraps the existing Ollama LLM service as a provider
type OllamaLLMProvider struct {
	client *ollama.Client
	config *domain.OllamaProviderConfig
}

var thinkTagRegex = regexp.MustCompile(`(?s)<think>.*?(?:</think>|$)`)

// removeThinkTags removes content between <think></think> tags if configured
func (p *OllamaLLMProvider) removeThinkTags(content string) string {
	if p.config.HideBuiltinThinkTag {
		return thinkTagRegex.ReplaceAllString(content, "")
	}
	return content
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

// toOllamaMessages converts domain messages to the Ollama API format
func toOllamaMessages(messages []domain.Message) []ollama.Message {
	ollamaMessages := make([]ollama.Message, 0, len(messages))
	for _, msg := range messages {
		ollamaMsg := ollama.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
		if len(msg.ToolCalls) > 0 {
			ollamaMsg.ToolCalls = make([]ollama.ToolCall, len(msg.ToolCalls))
			for i, tc := range msg.ToolCalls {
				// The Function field is an anonymous struct, so we cannot name the type.
				// This is a limitation of the current ollama-go library.
				// We have to rely on the structure and JSON marshaling.
				// As we are building the struct, not marshaling, this is tricky.
				// For now, we will assume the library handles this internally if we provide the right data.
				// A better solution would be for the library to export the Function type.
				// Let's try to build it by creating a map and letting the JSON marshaler handle it.
				// This is a workaround.
				toolCallMap := map[string]interface{}{
					"function": map[string]interface{}{
						"name":      tc.Function.Name,
						"arguments": tc.Function.Arguments,
					},
				}
				// This is not ideal, as we are converting back and forth. But it's a safe way to handle the anonymous struct.
				var tempToolCall ollama.ToolCall
				bytes, _ := json.Marshal(toolCallMap)
				if err := json.Unmarshal(bytes, &tempToolCall); err != nil {
					fmt.Printf("Warning: failed to unmarshal tool call: %v\n", err)
				}
				ollamaMsg.ToolCalls[i] = tempToolCall
			}
		}
		ollamaMessages = append(ollamaMessages, ollamaMsg)
	}
	return ollamaMessages
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
	}

	resp, err := p.client.Generate(ctx, req)
	if err != nil {
		return "", fmt.Errorf("%w: %v", domain.ErrGenerationFailed, err)
	}

	return p.removeThinkTags(resp.Response), nil
}

// Stream generates text with streaming using Ollama
func (p *OllamaLLMProvider) Stream(ctx context.Context, prompt string, opts *domain.GenerationOptions, callback func(string)) error {
	if prompt == "" {
		return fmt.Errorf("%w: empty prompt", domain.ErrInvalidInput)
	}
	if callback == nil {
		return fmt.Errorf("%w: nil callback", domain.ErrInvalidInput)
	}

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
				numPredict := opts.MaxTokens
				req.Options.NumPredict = &numPredict
			})
		}
	}

	respCh, errCh := ollama.GenerateStream(ctx, p.config.LLMModel, prompt, options...)

	// Use buffering for think tag removal if enabled
	var streamBuffer *StreamBuffer
	actualCallback := callback
	if p.config.HideBuiltinThinkTag {
		streamBuffer = NewStreamBuffer(callback)
		actualCallback = func(chunk string) {
			streamBuffer.Process(chunk)
		}
	}

	for {
		select {
		case resp, ok := <-respCh:
			if !ok {
				// Flush any remaining buffer when stream ends
				if streamBuffer != nil {
					streamBuffer.Flush()
				}
				return nil
			}
			if resp != nil && resp.Response != "" {
				actualCallback(resp.Response)
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

// GenerateWithTools generates text with tool calling support using Ollama
func (p *OllamaLLMProvider) GenerateWithTools(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions) (*domain.GenerationResult, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("%w: empty messages", domain.ErrInvalidInput)
	}

	ollamaMessages := toOllamaMessages(messages)

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

	req := &ollama.ChatRequest{
		Model:    p.config.LLMModel,
		Messages: ollamaMessages,
		Tools:    ollamaTools,
		Stream:   new(bool), // false
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
	}

	resp, err := p.client.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrGenerationFailed, err)
	}

	result := &domain.GenerationResult{
		Content:  p.removeThinkTags(resp.Message.Content),
		Finished: true,
	}

	if len(resp.Message.ToolCalls) > 0 {
		result.ToolCalls = make([]domain.ToolCall, len(resp.Message.ToolCalls))
		for i, tc := range resp.Message.ToolCalls {
			result.ToolCalls[i] = domain.ToolCall{
				ID:   tc.Function.Name, // Ollama doesn't provide an ID, so we use the name
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
func (p *OllamaLLMProvider) StreamWithTools(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions, callback domain.ToolCallCallback) error {
	if len(messages) == 0 {
		return fmt.Errorf("%w: empty messages", domain.ErrInvalidInput)
	}
	if callback == nil {
		return fmt.Errorf("%w: nil callback", domain.ErrInvalidInput)
	}

	ollamaMessages := toOllamaMessages(messages)

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

	chatOptions := func(req *ollama.ChatRequest) {
		req.Tools = ollamaTools
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
	}

	respCh, errCh := ollama.ChatStream(ctx, p.config.LLMModel, ollamaMessages, chatOptions)

	// Use buffering for think tag removal if enabled
	var streamBuffer *StreamBuffer
	actualCallback := callback
	if p.config.HideBuiltinThinkTag {
		streamBuffer = NewStreamBuffer(func(filtered string) {
			callback(filtered, nil)
		})
		actualCallback = func(content string, toolCalls []domain.ToolCall) error {
			if len(toolCalls) > 0 {
				// If we have tool calls, emit them immediately
				return callback("", toolCalls)
			}
			// Otherwise buffer the content for think tag filtering
			streamBuffer.Process(content)
			return nil
		}
	}

	for {
		select {
		case resp, ok := <-respCh:
			if !ok {
				// Flush any remaining buffer when stream ends
				if streamBuffer != nil {
					streamBuffer.Flush()
				}
				return nil
			}
			if resp != nil {
				var toolCalls []domain.ToolCall
				if len(resp.Message.ToolCalls) > 0 {
					toolCalls = make([]domain.ToolCall, len(resp.Message.ToolCalls))
					for i, tc := range resp.Message.ToolCalls {
						toolCalls[i] = domain.ToolCall{
							ID:   tc.Function.Name,
							Type: "function",
							Function: domain.FunctionCall{
								Name:      tc.Function.Name,
								Arguments: tc.Function.Arguments,
							},
						}
					}
				}
				if err := actualCallback(resp.Message.Content, toolCalls); err != nil {
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

// Health checks the health of the Ollama service
func (p *OllamaLLMProvider) Health(ctx context.Context) error {
	// First check if the service is available
	_, err := p.client.Version(ctx)
	if err != nil {
		return fmt.Errorf("%w: service unavailable: %v", domain.ErrServiceUnavailable, err)
	}

	// Now test the actual configured model - just verify it can respond
	stream := false
	req := &ollama.GenerateRequest{
		Model:  p.config.LLMModel,
		Prompt: "Hello",
		Stream: &stream,
		Options: &ollama.Options{
			Temperature: &[]float64{0.7}[0],
			NumPredict:  &[]int{10}[0], // Keep it short
		},
	}

	resp, err := p.client.Generate(ctx, req)
	if err != nil {
		return fmt.Errorf("LLM model health check failed: %w", err)
	}

	// Check if we got any response (don't validate content since models like qwen3 have built-in <think> tags)
	if resp == nil || resp.Response == "" {
		return fmt.Errorf("LLM model health check failed: empty response from model %s", p.config.LLMModel)
	}

	return nil
}

// GenerateStructured implements structured JSON output generation for Ollama using native JSON format
func (p *OllamaLLMProvider) GenerateStructured(ctx context.Context, prompt string, schema interface{}, opts *domain.GenerationOptions) (*domain.StructuredResult, error) {
	if err := ValidateStructuredRequest(prompt, schema); err != nil {
		return nil, err
	}

	if opts == nil {
		opts = DefaultStructuredOptions()
	}
	if err := ValidateGenerationOptions(opts); err != nil {
		return nil, err
	}

	messages := []ollama.Message{
		{Role: "user", Content: prompt},
	}

	// Use Ollama's native structured output with Format field
	response, err := ollama.Chat(ctx, p.config.LLMModel, messages, func(req *ollama.ChatRequest) {
		req.Format = schema
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
	})

	if err != nil {
		return nil, WrapStructuredOutputError(domain.ProviderOllama, err)
	}

	rawJSON := p.removeThinkTags(response.Message.Content)

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

const enhancedMetadataExtractionPromptTemplate = `You are an expert data extractor. Today's date is %s.

Analyze this document:
"""
%s
"""

Extract ALL information into this JSON structure:
{
  "summary": "one-sentence summary of the document",
  "keywords": ["keyword1", "keyword2", "keyword3"],
  "document_type": "Medical Record|Article|Meeting Notes|Technical Manual|Code|Essay",
  "collection": "Choose the most appropriate collection name based on content (use snake_case, e.g., medical_records, meeting_notes, technical_docs, research_papers, personal_notes, project_docs, legal_documents, financial_reports, customer_feedback, code_snippets)",
  "creation_date": "YYYY-MM-DD or null if not found",
  "temporal_refs": {
    "today": "%s",
    "tomorrow": "YYYY-MM-DD",
    "yesterday": "YYYY-MM-DD",
    "next week": "YYYY-MM-DD"
  },
  "entities": {
    "person": ["names of people"],
    "location": ["West China Hospital", "places mentioned"],
    "organization": ["companies or institutions"],
    "medical": ["vitrectomy", "medical procedures or terms"]
  },
  "events": ["pre-surgery examination", "scheduled surgery", "actions mentioned"],
  "custom_meta": {}
}

IMPORTANT: 
- Include ALL temporal references, entities, and events found in the text. If a category is empty, use empty array [].
- Choose collection name intelligently based on document content and purpose
- Use consistent collection names for similar content types`

// ExtractMetadata extracts metadata from content using Ollama
func (p *OllamaLLMProvider) ExtractMetadata(ctx context.Context, content string, model string) (*domain.ExtractedMetadata, error) {
	if content == "" {
		return nil, fmt.Errorf("%w: content cannot be empty", domain.ErrInvalidInput)
	}

	// Always use structured generation for metadata extraction
	currentDate := time.Now().Format("2006-01-02")
	prompt := fmt.Sprintf(enhancedMetadataExtractionPromptTemplate, currentDate, content, currentDate)
		
		// Define the schema for structured output as a map for Ollama's format
		schema := map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"summary": map[string]string{"type": "string"},
				"keywords": map[string]interface{}{
					"type": "array",
					"items": map[string]string{"type": "string"},
				},
				"document_type": map[string]string{"type": "string"},
				"collection": map[string]string{"type": "string"},
				"creation_date": map[string]string{"type": "string"},
				"temporal_refs": map[string]interface{}{
					"type": "object",
					"additionalProperties": map[string]string{"type": "string"},
				},
				"entities": map[string]interface{}{
					"type": "object",
					"additionalProperties": map[string]interface{}{
						"type": "array",
						"items": map[string]string{"type": "string"},
					},
				},
				"events": map[string]interface{}{
					"type": "array",
					"items": map[string]string{"type": "string"},
				},
				"custom_meta": map[string]interface{}{
					"type": "object",
					"additionalProperties": true,
				},
			},
			"required": []string{"summary", "keywords", "document_type", "collection"},
		}
		
		// Temporarily override model if specified
		originalModel := p.config.LLMModel
		if model != "" {
			p.config.LLMModel = model
			defer func() { p.config.LLMModel = originalModel }()
		}
		
		// Use GenerateStructured for reliable JSON parsing
		result, err := p.GenerateStructured(ctx, prompt, schema, nil)
		if err != nil {
			return nil, fmt.Errorf("%w: enhanced metadata extraction failed: %v", domain.ErrGenerationFailed, err)
		}
		
		// Parse the structured result directly from the raw JSON response
		var metadata domain.ExtractedMetadata
		if err := json.Unmarshal([]byte(result.Raw), &metadata); err != nil {
			return nil, fmt.Errorf("%w: failed to unmarshal metadata: %v", domain.ErrInvalidInput, err)
		}
		
		// Initialize empty maps/slices if nil to ensure consistent output
		if metadata.TemporalRefs == nil {
			metadata.TemporalRefs = make(map[string]string)
		}
		if metadata.Entities == nil {
			metadata.Entities = make(map[string][]string)
		}
		if metadata.Events == nil {
			metadata.Events = []string{}
		}
		if metadata.CustomMeta == nil {
			metadata.CustomMeta = make(map[string]interface{})
		}
		
	return &metadata, nil
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
	// First check if the service is available
	_, err := p.client.Version(ctx)
	if err != nil {
		return fmt.Errorf("%w: service unavailable: %v", domain.ErrServiceUnavailable, err)
	}

	// Now test the actual configured embedding model with a simple test
	req := &ollama.EmbedRequest{
		Model: p.config.EmbeddingModel,
		Input: "test",
	}

	resp, err := p.client.Embed(ctx, req)
	if err != nil {
		return fmt.Errorf("embedding model health check failed: %w", err)
	}

	// Check if we got a reasonable embedding response
	if resp == nil || len(resp.Embeddings) == 0 || len(resp.Embeddings[0]) == 0 {
		return fmt.Errorf("embedding model health check failed: empty embedding from model %s", p.config.EmbeddingModel)
	}

	return nil
}

// RecognizeIntent analyzes user request to determine intent type
func (p *OllamaLLMProvider) RecognizeIntent(ctx context.Context, request string) (*domain.IntentResult, error) {
	if request == "" {
		return nil, fmt.Errorf("%w: empty request", domain.ErrInvalidInput)
	}

	prompt := fmt.Sprintf(`Analyze this request and determine the user's intent. Categorize it into ONE of these types:

REQUEST: %s

INTENT CATEGORIES:
- "question": User is asking a question that needs an answer (what, why, how, explain, etc.)
- "action": User wants to perform an action (create, delete, modify, run, execute, etc.)
- "analysis": User wants to analyze or process data (analyze, compare, evaluate, review, etc.)
- "search": User wants to find or list information (find, search, list, show, get, etc.)
- "calculation": User wants a mathematical or logical computation (calculate, compute, count, etc.)
- "status": User wants to check status or state (check, status, verify, test, etc.)

Also determine if this request requires external tools to complete.

Respond with JSON:
{
  "intent": "one of the categories above",
  "confidence": 0.0-1.0,
  "key_verbs": ["verb1", "verb2"],
  "entities": ["entity1", "entity2"],
  "needs_tools": true/false,
  "reasoning": "brief explanation of why tools are needed or not"
}`, request)

	// Define the schema for intent recognition
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"intent": map[string]interface{}{
				"type": "string",
				"enum": []string{"question", "action", "analysis", "search", "calculation", "status", "unknown"},
			},
			"confidence": map[string]interface{}{
				"type": "number",
				"minimum": 0.0,
				"maximum": 1.0,
			},
			"key_verbs": map[string]interface{}{
				"type": "array",
				"items": map[string]string{"type": "string"},
			},
			"entities": map[string]interface{}{
				"type": "array",
				"items": map[string]string{"type": "string"},
			},
			"needs_tools": map[string]interface{}{
				"type": "boolean",
			},
			"reasoning": map[string]interface{}{
				"type": "string",
			},
		},
		"required": []string{"intent", "confidence", "needs_tools"},
	}

	// Use structured generation for reliable parsing
	opts := &domain.GenerationOptions{
		Temperature: 0.1,
		MaxTokens:   300,
	}
	
	result, err := p.GenerateStructured(ctx, prompt, schema, opts)
	if err != nil {
		// Fallback to unknown intent
		return &domain.IntentResult{
			Intent:     domain.IntentUnknown,
			Confidence: 0.0,
			NeedsTools: true, // Conservative default
			Reasoning:  "Failed to determine intent",
		}, nil
	}

	// Parse the structured result
	var intentData struct {
		Intent     string   `json:"intent"`
		Confidence float64  `json:"confidence"`
		KeyVerbs   []string `json:"key_verbs"`
		Entities   []string `json:"entities"`
		NeedsTools bool     `json:"needs_tools"`
		Reasoning  string   `json:"reasoning"`
	}

	if err := json.Unmarshal([]byte(result.Raw), &intentData); err != nil {
		return &domain.IntentResult{
			Intent:     domain.IntentUnknown,
			Confidence: 0.0,
			NeedsTools: true,
			Reasoning:  "Failed to parse intent",
		}, nil
	}

	// Map string to IntentType
	var intentType domain.IntentType
	switch intentData.Intent {
	case "question":
		intentType = domain.IntentQuestion
	case "action":
		intentType = domain.IntentAction
	case "analysis":
		intentType = domain.IntentAnalysis
	case "search":
		intentType = domain.IntentSearch
	case "calculation":
		intentType = domain.IntentCalculation
	case "status":
		intentType = domain.IntentStatus
	default:
		intentType = domain.IntentUnknown
	}

	return &domain.IntentResult{
		Intent:     intentType,
		Confidence: intentData.Confidence,
		KeyVerbs:   intentData.KeyVerbs,
		Entities:   intentData.Entities,
		NeedsTools: intentData.NeedsTools,
		Reasoning:  intentData.Reasoning,
	}, nil
}
