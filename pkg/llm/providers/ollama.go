package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	ollama "github.com/liliang-cn/ollama-go"
	"github.com/liliang-cn/rago/v2/pkg/core"
)

// OllamaProvider implements the Provider interface for Ollama
type OllamaProvider struct {
	config  core.ProviderConfig
	name    string
	client  *ollama.Client
}

// NewOllamaProvider creates a new Ollama provider for the LLM pillar
func NewOllamaProvider(name string, config core.ProviderConfig) (*OllamaProvider, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	
	// Create Ollama client with host URL
	client, err := ollama.NewClient(ollama.WithHost(baseURL))
	if err != nil {
		return nil, fmt.Errorf("failed to create Ollama client: %w", err)
	}
	
	return &OllamaProvider{
		config: config,
		name:   name,
		client: client,
	}, nil
}

// Name returns the provider name
func (p *OllamaProvider) Name() string {
	return p.name
}

// Type returns the provider type
func (p *OllamaProvider) Type() string {
	return "ollama"
}

// Model returns the configured model
func (p *OllamaProvider) Model() string {
	return p.config.Model
}

// Config returns the provider configuration
func (p *OllamaProvider) Config() core.ProviderConfig {
	return p.config
}

// Health checks the provider health
func (p *OllamaProvider) Health(ctx context.Context) error {
	// Check if Ollama server is running by listing models
	_, err := p.client.List(ctx)
	if err != nil {
		return fmt.Errorf("Ollama server not reachable: %w", err)
	}
	return nil
}

// Generate generates text using the Ollama provider
func (p *OllamaProvider) Generate(ctx context.Context, req *GenerationRequest) (*GenerationResponse, error) {
	// Build the prompt from context messages
	prompt := p.buildPrompt(req)
	
	// Create Ollama request with proper Options
	temperature := float64(req.Temperature)
	numPredict := req.MaxTokens
	ollamaReq := &ollama.GenerateRequest{
		Model:  p.config.Model,
		Prompt: prompt,
		Options: &ollama.Options{
			Temperature: &temperature,
			NumPredict:  &numPredict,
		},
	}
	
	startTime := time.Now()
	
	// Generate response
	resp, err := p.client.Generate(ctx, ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("Ollama generation failed: %w", err)
	}
	
	// Create response with cleaned content
	cleanedContent := p.stripThinkTags(resp.Response)
	return &GenerationResponse{
		Content:  cleanedContent,
		Model:    resp.Model,
		Provider: p.name,
		Usage: TokenUsage{
			PromptTokens:     resp.PromptEvalCount,
			CompletionTokens: resp.EvalCount,
			TotalTokens:      resp.PromptEvalCount + resp.EvalCount,
		},
		Duration: time.Since(startTime),
		Metadata: map[string]interface{}{
			"total_duration": resp.TotalDuration,
			"load_duration":  resp.LoadDuration,
			"eval_duration":  resp.EvalDuration,
		},
	}, nil
}

// stripThinkTags removes <think>...</think> tags from the content
// This is needed because some models like Qwen3 include built-in thinking tags
func (p *OllamaProvider) stripThinkTags(content string) string {
	// Regular expression to match <think>...</think> tags including nested ones
	thinkRegex := regexp.MustCompile(`<think>[\s\S]*?</think>`)
	cleaned := thinkRegex.ReplaceAllString(content, "")
	
	// Also remove any standalone <think> or </think> tags
	cleaned = regexp.MustCompile(`</?think>`).ReplaceAllString(cleaned, "")
	
	return cleaned
}

// stripThinkTagsFromDelta removes think tag content from delta but preserves spacing
func (p *OllamaProvider) stripThinkTagsFromDelta(delta string, insideThinkTag *bool) string {
	result := ""
	i := 0
	
	for i < len(delta) {
		// Check for opening think tag
		if strings.HasPrefix(delta[i:], "<think>") {
			*insideThinkTag = true
			i += 7 // Skip "<think>"
			continue
		}
		
		// Check for closing think tag
		if strings.HasPrefix(delta[i:], "</think>") {
			*insideThinkTag = false
			i += 8 // Skip "</think>"
			continue
		}
		
		// If we're inside a think tag, skip this character
		if *insideThinkTag {
			i++
			continue
		}
		
		// Otherwise, include this character in the result
		result += string(delta[i])
		i++
	}
	
	return result
}

// Stream generates streaming text using the Ollama provider
func (p *OllamaProvider) Stream(ctx context.Context, req *GenerationRequest, callback StreamCallback) error {
	// Build the prompt from context messages
	prompt := p.buildPrompt(req)
	
	// Create Ollama request with proper Options
	temperature := float64(req.Temperature)
	numPredict := req.MaxTokens
	ollamaReq := &ollama.GenerateRequest{
		Model:  p.config.Model,
		Prompt: prompt,
		Options: &ollama.Options{
			Temperature: &temperature,
			NumPredict:  &numPredict,
		},
	}
	
	startTime := time.Now()
	var totalContent strings.Builder
	var insideThinkTag bool = false // Track if we're inside a think tag
	
	// Stream response - returns two channels
	respChan, errChan := p.client.GenerateStream(ctx, ollamaReq)
	
	for {
		select {
		case resp, ok := <-respChan:
			if !ok {
				// Channel closed, we're done
				return nil
			}
			
			totalContent.WriteString(resp.Response)
			
			// Clean content and delta from think tags using stateful processing
			cleanedContent := p.stripThinkTags(totalContent.String())
			cleanedDelta := p.stripThinkTagsFromDelta(resp.Response, &insideThinkTag)
			
			streamChunk := &StreamChunk{
				Content:  cleanedContent,
				Delta:    cleanedDelta,
				Finished: resp.Done,
				Duration: time.Since(startTime),
			}
			
			if resp.Done {
				streamChunk.Usage = TokenUsage{
					PromptTokens:     resp.PromptEvalCount,
					CompletionTokens: resp.EvalCount,
					TotalTokens:      resp.PromptEvalCount + resp.EvalCount,
				}
			}
			
			callback(streamChunk)
			
			if resp.Done {
				return nil
			}
			
		case err := <-errChan:
			if err != nil {
				return fmt.Errorf("Ollama stream failed: %w", err)
			}
		}
	}
}

// GenerateWithTools generates text with native tool calling support
func (p *OllamaProvider) GenerateWithTools(ctx context.Context, req *ToolGenerationRequest) (*ToolGenerationResponse, error) {
	// Build messages from context
	var messages []ollama.Message
	for _, msg := range req.Context {
		messages = append(messages, ollama.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	
	// Add the main prompt as user message if provided
	if req.Prompt != "" {
		messages = append(messages, ollama.Message{
			Role:    "user",
			Content: req.Prompt,
		})
	}
	
	// Convert tools to Ollama format with proper JSON schema
	var tools []ollama.Tool
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
		
		ollamaTool := ollama.Tool{
			Type: "function",
			Function: &ollama.ToolFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  parameters,
			},
		}
		tools = append(tools, ollamaTool)
	}
	
	// Get model name
	model := req.Model
	if model == "" {
		model = p.config.Model
	}
	
	// Create stream pointer
	stream := false
	
	// Create chat request
	chatReq := &ollama.ChatRequest{
		Model:    model,
		Messages: messages,
		Tools:    tools,
		Stream:   &stream,
	}
	
	// Set options if needed
	if req.Temperature > 0 || req.MaxTokens > 0 {
		temp := float64(req.Temperature)
		numPred := req.MaxTokens
		
		options := &ollama.Options{}
		if req.Temperature > 0 {
			options.Temperature = &temp
		}
		if req.MaxTokens > 0 {
			options.NumPredict = &numPred
		}
		chatReq.Options = options
	}
	
	// Make the API call
	response, err := p.client.Chat(ctx, chatReq)
	if err != nil {
		return nil, fmt.Errorf("ollama tool generation failed: %w", err)
	}
	
	// Convert tool calls from response
	var toolCalls []ToolCall
	if response.Message.ToolCalls != nil {
		for i, tc := range response.Message.ToolCalls {
			toolCall := ToolCall{
				ID:         fmt.Sprintf("call_%d", i+1),
				Name:       tc.Function.Name,
				Parameters: tc.Function.Arguments,
			}
			toolCalls = append(toolCalls, toolCall)
		}
	}
	
	// Build response
	return &ToolGenerationResponse{
		GenerationResponse: GenerationResponse{
			Content:  response.Message.Content,
			Model:    response.Model,
			Provider: p.name,
			Usage: TokenUsage{
				PromptTokens:     response.PromptEvalCount,
				CompletionTokens: response.EvalCount,
				TotalTokens:      response.PromptEvalCount + response.EvalCount,
			},
			Metadata: map[string]interface{}{
				"done":           response.Done,
				"has_tools":      len(toolCalls) > 0,
				"total_duration": response.TotalDuration,
			},
		},
		ToolCalls: toolCalls,
	}, nil
}

// StreamWithTools generates streaming text with tool calling capability
func (p *OllamaProvider) StreamWithTools(ctx context.Context, req *ToolGenerationRequest, callback ToolStreamCallback) error {
	// Build prompt with tool definitions
	promptWithTools := p.buildPromptWithTools(req)
	
	// Create generation request with tool-aware prompt
	genReq := &GenerationRequest{
		Prompt:      promptWithTools,
		Model:       req.Model,
		Parameters:  req.Parameters,
		Context:     req.Context,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	}
	
	var accumulatedContent strings.Builder
	var pendingToolCalls []ToolCall
	
	return p.Stream(ctx, genReq, func(chunk *StreamChunk) {
		accumulatedContent.WriteString(chunk.Delta)
		
		// Check if we've accumulated a complete tool call
		if chunk.Finished {
			// Parse tool calls from the complete content
			toolCalls, cleanedContent := p.parseToolCalls(accumulatedContent.String())
			pendingToolCalls = toolCalls
			
			// Create final chunk with tool calls
			toolChunk := &ToolStreamChunk{
				StreamChunk: StreamChunk{
					Content:  cleanedContent,
					Delta:    "", // Final chunk has no delta
					Finished: true,
					Usage:    chunk.Usage,
					Duration: chunk.Duration,
				},
				ToolCalls: pendingToolCalls,
			}
			callback(toolChunk)
		} else {
			// Regular streaming chunk without tool calls yet
			toolChunk := &ToolStreamChunk{
				StreamChunk: *chunk,
				ToolCalls:   []ToolCall{},
			}
			callback(toolChunk)
		}
	})
}

// Capabilities returns the provider capabilities
func (p *OllamaProvider) Capabilities() ProviderCapabilities {
	return ProviderCapabilities{
		SupportsStreaming:  true,
		SupportsToolCalls: true, // Now supports tool calling via prompt engineering
		SupportsBatch:     false,
		MaxTokens:         4096, // Default for most Ollama models
		MaxContextLength:  4096,
	}
}

// Metadata returns additional provider metadata
func (p *OllamaProvider) Metadata() map[string]interface{} {
	return map[string]interface{}{
		"provider_type": "ollama",
		"model":         p.config.Model,
		"timeout":       p.config.Timeout,
		"local":         true,
	}
}

// GenerateEmbedding generates embeddings for the provided text using Ollama's embedding API
func (p *OllamaProvider) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	// Create embedding request for Ollama
	req := &ollama.EmbeddingsRequest{
		Model:  p.config.Model, // Use configured model (should be an embedding model)
		Prompt: text,
	}
	
	// Generate embedding
	resp, err := p.client.Embeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("Ollama embedding generation failed: %w", err)
	}
	
	// Return the embedding directly (already []float64)
	return resp.Embedding, nil
}

// Close closes the provider and cleans up resources
func (p *OllamaProvider) Close() error {
	// Nothing to clean up for Ollama client
	return nil
}

// buildPrompt builds a prompt string from the request
func (p *OllamaProvider) buildPrompt(req *GenerationRequest) string {
	var prompt strings.Builder
	
	// Add context messages
	for _, msg := range req.Context {
		switch msg.Role {
		case "system":
			prompt.WriteString(fmt.Sprintf("System: %s\n\n", msg.Content))
		case "user":
			prompt.WriteString(fmt.Sprintf("User: %s\n\n", msg.Content))
		case "assistant":
			prompt.WriteString(fmt.Sprintf("Assistant: %s\n\n", msg.Content))
		case "tool":
			// Handle tool result messages
			prompt.WriteString(fmt.Sprintf("Tool Result (ID: %s): %s\n\n", msg.ToolCallID, msg.Content))
		default:
			prompt.WriteString(fmt.Sprintf("%s: %s\n\n", msg.Role, msg.Content))
		}
	}
	
	// Add the main prompt
	if req.Prompt != "" {
		prompt.WriteString(fmt.Sprintf("User: %s\n\nAssistant: ", req.Prompt))
	}
	
	return prompt.String()
}

// buildPromptWithTools builds a prompt that includes tool definitions
func (p *OllamaProvider) buildPromptWithTools(req *ToolGenerationRequest) string {
	var prompt strings.Builder
	
	// Add system message with tool instructions
	prompt.WriteString("System: You have access to the following tools that you can use to help answer questions and perform tasks:\n\n")
	
	// Add tool definitions
	if len(req.Tools) > 0 {
		prompt.WriteString("Available Tools:\n")
		for _, tool := range req.Tools {
			prompt.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name, tool.Description))
			if tool.Parameters != nil {
				paramsJSON, _ := json.Marshal(tool.Parameters)
				prompt.WriteString(fmt.Sprintf("  Parameters: %s\n", string(paramsJSON)))
			}
		}
		prompt.WriteString("\n")
		
		// Add tool calling instructions
		prompt.WriteString("To use a tool, respond with a JSON object in the following format:\n")
		prompt.WriteString("```json\n")
		prompt.WriteString("{\n")
		prompt.WriteString("  \"tool_calls\": [\n")
		prompt.WriteString("    {\n")
		prompt.WriteString("      \"id\": \"unique_call_id\",\n")
		prompt.WriteString("      \"name\": \"tool_name\",\n")
		prompt.WriteString("      \"parameters\": {\"param1\": \"value1\", \"param2\": \"value2\"}\n")
		prompt.WriteString("    }\n")
		prompt.WriteString("  ],\n")
		prompt.WriteString("  \"reasoning\": \"Brief explanation of why you're using this tool\"\n")
		prompt.WriteString("}\n")
		prompt.WriteString("```\n\n")
		prompt.WriteString("After calling a tool, wait for the result before continuing. ")
		prompt.WriteString("If you don't need to use any tools, just respond normally without JSON.\n\n")
	}
	
	// Add context messages
	for _, msg := range req.Context {
		switch msg.Role {
		case "system":
			// Skip if we already added our tool system message
			if !strings.Contains(msg.Content, "Available Tools:") {
				prompt.WriteString(fmt.Sprintf("System: %s\n\n", msg.Content))
			}
		case "user":
			prompt.WriteString(fmt.Sprintf("User: %s\n\n", msg.Content))
		case "assistant":
			prompt.WriteString(fmt.Sprintf("Assistant: %s\n\n", msg.Content))
		case "tool":
			// Handle tool result messages
			prompt.WriteString(fmt.Sprintf("Tool Result (ID: %s): %s\n\n", msg.ToolCallID, msg.Content))
		default:
			prompt.WriteString(fmt.Sprintf("%s: %s\n\n", msg.Role, msg.Content))
		}
	}
	
	// Add the main prompt
	if req.Prompt != "" {
		prompt.WriteString(fmt.Sprintf("User: %s\n\n", req.Prompt))
		
		// Add tool choice guidance if specified
		if req.ToolChoice == "required" {
			prompt.WriteString("You MUST use one or more of the available tools to answer this question.\n\n")
		} else if req.ToolChoice == "none" {
			prompt.WriteString("Please answer directly without using any tools.\n\n")
		}
		
		prompt.WriteString("Assistant: ")
	}
	
	return prompt.String()
}

// parseToolCalls extracts tool calls from the response content
func (p *OllamaProvider) parseToolCalls(content string) ([]ToolCall, string) {
	var toolCalls []ToolCall
	cleanedContent := content
	
	// Look for JSON blocks that contain tool calls
	// Match ```json...``` blocks or direct JSON objects with tool_calls
	jsonBlockRegex := regexp.MustCompile(`(?s)` +
		`(?:` +
		`\x60\x60\x60json\s*\n?(.+?)\n?\x60\x60\x60` + // ```json ... ```
		`|` +
		`\x60\x60\x60\s*\n?(.+?)\n?\x60\x60\x60` + // ``` ... ```
		`|` +
		`(\{[^{}]*"tool_calls"[^{}]*\})` + // Direct JSON object with tool_calls
		`)`)
	
	matches := jsonBlockRegex.FindAllStringSubmatch(content, -1)
	
	for _, match := range matches {
		// Get the JSON content from whichever group matched
		var jsonStr string
		for i := 1; i < len(match); i++ {
			if match[i] != "" {
				jsonStr = match[i]
				break
			}
		}
		
		if jsonStr == "" {
			continue
		}
		
		// Try to parse the JSON
		var toolCallData struct {
			ToolCalls []struct {
				ID         string                 `json:"id"`
				Name       string                 `json:"name"`
				Parameters map[string]interface{} `json:"parameters"`
			} `json:"tool_calls"`
			Reasoning string `json:"reasoning,omitempty"`
		}
		
		if err := json.Unmarshal([]byte(jsonStr), &toolCallData); err == nil && len(toolCallData.ToolCalls) > 0 {
			// Convert to ToolCall type
			for _, tc := range toolCallData.ToolCalls {
				// Generate ID if not provided
				id := tc.ID
				if id == "" {
					id = fmt.Sprintf("call_%d_%d", time.Now().Unix(), len(toolCalls))
				}
				
				toolCall := ToolCall{
					ID:         id,
					Name:       tc.Name,
					Parameters: tc.Parameters,
				}
				
				// Ensure Parameters is not nil
				if toolCall.Parameters == nil {
					toolCall.Parameters = make(map[string]interface{})
				}
				
				toolCalls = append(toolCalls, toolCall)
			}
			
			// Remove the JSON block from the content
			cleanedContent = strings.Replace(cleanedContent, match[0], "", 1)
			
			// Add reasoning if present (as explanation text)
			if toolCallData.Reasoning != "" {
				cleanedContent = strings.TrimSpace(cleanedContent)
				if cleanedContent != "" && !strings.HasSuffix(cleanedContent, ".") {
					cleanedContent += ". "
				}
				cleanedContent += toolCallData.Reasoning
			}
		}
	}
	
	// Clean up the content
	cleanedContent = strings.TrimSpace(cleanedContent)
	
	// If we found tool calls but no other content, provide a default message
	if len(toolCalls) > 0 && cleanedContent == "" {
		cleanedContent = "I'll help you with that using the available tools."
	}
	
	return toolCalls, cleanedContent
}