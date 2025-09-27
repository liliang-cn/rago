package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/domain"
)

// ClaudeProvider implements the LLMProvider interface for Anthropic Claude
type ClaudeProvider struct {
	config     *domain.ClaudeProviderConfig
	httpClient *http.Client
}

// NewClaudeProvider creates a new Claude provider
func NewClaudeProvider(config *domain.ClaudeProviderConfig) (*ClaudeProvider, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("claude api key is required")
	}
	
	if config.LLMModel == "" {
		config.LLMModel = "claude-3-sonnet-20240229" // Default model
	}
	
	if config.BaseURL == "" {
		config.BaseURL = "https://api.anthropic.com"
	}
	
	if config.AnthropicVersion == "" {
		config.AnthropicVersion = "2023-06-01"
	}
	
	if config.MaxTokens == 0 {
		config.MaxTokens = 4096
	}
	
	timeout := 30 * time.Second
	if config.Timeout > 0 {
		timeout = config.Timeout
	}
	
	return &ClaudeProvider{
		config: config,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// ProviderType returns the provider type
func (p *ClaudeProvider) ProviderType() domain.ProviderType {
	return domain.ProviderClaude
}

// Health checks if the provider is healthy
func (p *ClaudeProvider) Health(ctx context.Context) error {
	// Simple test call to verify API key and connectivity
	_, err := p.Generate(ctx, "Hello", &domain.GenerationOptions{MaxTokens: 10})
	if err != nil {
		return fmt.Errorf("claude health check failed: %w", err)
	}
	return nil
}

// Generate generates text using Claude
func (p *ClaudeProvider) Generate(ctx context.Context, prompt string, opts *domain.GenerationOptions) (string, error) {
	claudeReq := &ClaudeRequest{
		Model:     p.config.LLMModel,
		MaxTokens: p.config.MaxTokens,
		Messages: []ClaudeMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}
	
	if opts != nil {
		if opts.MaxTokens > 0 {
			claudeReq.MaxTokens = opts.MaxTokens
		}
		if opts.Temperature > 0 {
			claudeReq.Temperature = &opts.Temperature
		}
	}
	
	response, err := p.callClaude(ctx, claudeReq)
	if err != nil {
		return "", err
	}
	
	if len(response.Content) == 0 {
		return "", fmt.Errorf("no content in claude response")
	}
	
	return response.Content[0].Text, nil
}

// Stream generates streaming text using Claude
func (p *ClaudeProvider) Stream(ctx context.Context, prompt string, opts *domain.GenerationOptions, callback func(string)) error {
	// For now, implement as non-streaming
	response, err := p.Generate(ctx, prompt, opts)
	if err != nil {
		return err
	}
	
	callback(response)
	return nil
}

// GenerateWithTools generates text with tool calling support
func (p *ClaudeProvider) GenerateWithTools(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions) (*domain.GenerationResult, error) {
	claudeReq := &ClaudeRequest{
		Model:     p.config.LLMModel,
		MaxTokens: p.config.MaxTokens,
		Messages:  make([]ClaudeMessage, 0, len(messages)),
	}
	
	if opts != nil {
		if opts.MaxTokens > 0 {
			claudeReq.MaxTokens = opts.MaxTokens
		}
		if opts.Temperature > 0 {
			claudeReq.Temperature = &opts.Temperature
		}
	}
	
	// Convert tools to Claude format
	if len(tools) > 0 {
		claudeReq.Tools = make([]ClaudeTool, 0, len(tools))
		for _, tool := range tools {
			claudeReq.Tools = append(claudeReq.Tools, ClaudeTool{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				InputSchema: tool.Function.Parameters,
			})
		}
	}
	
	// Convert messages
	for _, msg := range messages {
		claudeReq.Messages = append(claudeReq.Messages, ClaudeMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	
	response, err := p.callClaude(ctx, claudeReq)
	if err != nil {
		return nil, err
	}
	
	// Handle tool calls if present
	var toolCalls []domain.ToolCall
	content := ""
	
	if len(response.Content) > 0 {
		for _, contentBlock := range response.Content {
			if contentBlock.Type == "text" {
				content = contentBlock.Text
			} else if contentBlock.Type == "tool_use" {
				argsMap := make(map[string]interface{})
				if contentBlock.Input != nil {
					json.Unmarshal(contentBlock.Input, &argsMap)
				}
				
				toolCalls = append(toolCalls, domain.ToolCall{
					ID:   contentBlock.ID,
					Type: "function",
					Function: domain.FunctionCall{
						Name:      contentBlock.Name,
						Arguments: argsMap,
					},
				})
			}
		}
	}
	
	return &domain.GenerationResult{
		Content:   content,
		ToolCalls: toolCalls,
		Finished:  response.StopReason == "end_turn" || response.StopReason == "stop_sequence",
	}, nil
}

// StreamWithTools generates streaming text with tool calling support
func (p *ClaudeProvider) StreamWithTools(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions, callback domain.ToolCallCallback) error {
	// For now, implement as non-streaming
	result, err := p.GenerateWithTools(ctx, messages, tools, opts)
	if err != nil {
		return err
	}
	
	return callback(result.Content, result.ToolCalls)
}

// GenerateStructured generates structured output (not directly supported by Claude)
func (p *ClaudeProvider) GenerateStructured(ctx context.Context, prompt string, schema interface{}, opts *domain.GenerationOptions) (*domain.StructuredResult, error) {
	// Convert to regular generate request with schema instructions
	enhancedPrompt := prompt + "\n\nPlease respond in the exact JSON format specified."
	
	response, err := p.Generate(ctx, enhancedPrompt, opts)
	if err != nil {
		return nil, err
	}
	
	var data interface{}
	valid := true
	if err := json.Unmarshal([]byte(response), &data); err != nil {
		valid = false
	}
	
	return &domain.StructuredResult{
		Data:  data,
		Raw:   response,
		Valid: valid,
	}, nil
}

// RecognizeIntent recognizes intent from user input
func (p *ClaudeProvider) RecognizeIntent(ctx context.Context, request string) (*domain.IntentResult, error) {
	prompt := fmt.Sprintf(`Analyze the following user request and determine the intent. Return a JSON object with:
- intent: one of [question, action, analysis, search, calculation, status, unknown]
- confidence: a number between 0 and 1
- key_verbs: array of action verbs
- entities: array of important entities mentioned
- needs_tools: boolean indicating if tools are needed
- reasoning: brief explanation

User request: %s

Return only the JSON object:`, request)
	
	response, err := p.Generate(ctx, prompt, &domain.GenerationOptions{MaxTokens: 500})
	if err != nil {
		return nil, err
	}
	
	var result domain.IntentResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		// If parsing fails, return a basic result
		return &domain.IntentResult{
			Intent:     domain.IntentUnknown,
			Confidence: 0.5,
			NeedsTools: false,
			Reasoning:  "Failed to parse intent analysis",
		}, nil
	}
	
	return &result, nil
}

// ExtractMetadata extracts metadata from content
func (p *ClaudeProvider) ExtractMetadata(ctx context.Context, content string, model string) (*domain.ExtractedMetadata, error) {
	if model == "" {
		model = p.config.LLMModel
	}
	
	prompt := fmt.Sprintf(`Extract metadata from the following content. Return a JSON object with these fields:
- summary: A brief summary (max 200 words)
- keywords: Array of relevant keywords
- document_type: Document type/category
- creation_date: Extracted or estimated creation date
- collection: Suggested collection name

Content:
%s

Return only the JSON object:`, content)
	
	response, err := p.Generate(ctx, prompt, &domain.GenerationOptions{MaxTokens: 1000})
	if err != nil {
		return nil, err
	}
	
	var metadata domain.ExtractedMetadata
	if err := json.Unmarshal([]byte(response), &metadata); err != nil {
		// If JSON parsing fails, create basic metadata
		return &domain.ExtractedMetadata{
			Summary:      content[:minInt(200, len(content))],
			Keywords:     []string{},
			DocumentType: "unknown",
		}, nil
	}
	
	return &metadata, nil
}

// callClaude makes HTTP request to Claude API
func (p *ClaudeProvider) callClaude(ctx context.Context, req *ClaudeRequest) (*ClaudeResponse, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL+"/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.config.APIKey)
	httpReq.Header.Set("anthropic-version", p.config.AnthropicVersion)
	
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("claude api error %d: %s", resp.StatusCode, string(body))
	}
	
	var claudeResp ClaudeResponse
	if err := json.Unmarshal(body, &claudeResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	
	return &claudeResp, nil
}

// Claude API types
type ClaudeRequest struct {
	Model       string          `json:"model"`
	MaxTokens   int             `json:"max_tokens"`
	Messages    []ClaudeMessage `json:"messages"`
	System      string          `json:"system,omitempty"`
	Temperature *float64        `json:"temperature,omitempty"`
	Tools       []ClaudeTool    `json:"tools,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
}

type ClaudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ClaudeTool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"input_schema"`
}

type ClaudeResponse struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Role         string                 `json:"role"`
	Content      []ClaudeContentBlock   `json:"content"`
	Model        string                 `json:"model"`
	StopReason   string                 `json:"stop_reason"`
	StopSequence string                 `json:"stop_sequence"`
	Usage        ClaudeUsage            `json:"usage"`
}

type ClaudeContentBlock struct {
	Type string          `json:"type"`
	Text string          `json:"text,omitempty"`
	ID   string          `json:"id,omitempty"`
	Name string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

type ClaudeUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Helper function
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}