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

// GeminiProvider implements the LLMProvider interface for Google Gemini
type GeminiProvider struct {
	config     *domain.GeminiProviderConfig
	httpClient *http.Client
}

// NewGeminiProvider creates a new Gemini provider
func NewGeminiProvider(config *domain.GeminiProviderConfig) (*GeminiProvider, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("gemini api key is required")
	}
	
	if config.LLMModel == "" {
		config.LLMModel = "gemini-pro" // Default model
	}
	
	if config.BaseURL == "" {
		config.BaseURL = "https://generativelanguage.googleapis.com"
	}
	
	timeout := 30 * time.Second
	if config.Timeout > 0 {
		timeout = config.Timeout
	}
	
	return &GeminiProvider{
		config: config,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// ProviderType returns the provider type
func (p *GeminiProvider) ProviderType() domain.ProviderType {
	return domain.ProviderGemini
}

// Health checks if the provider is healthy
func (p *GeminiProvider) Health(ctx context.Context) error {
	// Simple test call to verify API key and connectivity
	_, err := p.Generate(ctx, "Hello", &domain.GenerationOptions{MaxTokens: 10})
	if err != nil {
		return fmt.Errorf("gemini health check failed: %w", err)
	}
	return nil
}

// Generate generates text using Gemini
func (p *GeminiProvider) Generate(ctx context.Context, prompt string, opts *domain.GenerationOptions) (string, error) {
	geminiReq := &GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{Text: prompt},
				},
			},
		},
		GenerationConfig: &GeminiGenerationConfig{},
	}
	
	if opts != nil {
		if opts.MaxTokens > 0 {
			geminiReq.GenerationConfig.MaxOutputTokens = &opts.MaxTokens
		}
		if opts.Temperature > 0 {
			geminiReq.GenerationConfig.Temperature = &opts.Temperature
		}
	}
	
	response, err := p.callGemini(ctx, geminiReq)
	if err != nil {
		return "", err
	}
	
	if len(response.Candidates) == 0 {
		return "", fmt.Errorf("no candidates in gemini response")
	}
	
	candidate := response.Candidates[0]
	if len(candidate.Content.Parts) == 0 {
		return "", fmt.Errorf("no parts in gemini candidate")
	}
	
	return candidate.Content.Parts[0].Text, nil
}

// Stream generates streaming text using Gemini
func (p *GeminiProvider) Stream(ctx context.Context, prompt string, opts *domain.GenerationOptions, callback func(string)) error {
	// For now, implement as non-streaming
	response, err := p.Generate(ctx, prompt, opts)
	if err != nil {
		return err
	}
	
	callback(response)
	return nil
}

// GenerateWithTools generates text with tool calling support
func (p *GeminiProvider) GenerateWithTools(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions) (*domain.GenerationResult, error) {
	geminiReq := &GeminiRequest{
		Contents:         make([]GeminiContent, 0, len(messages)),
		GenerationConfig: &GeminiGenerationConfig{},
	}
	
	if opts != nil {
		if opts.MaxTokens > 0 {
			geminiReq.GenerationConfig.MaxOutputTokens = &opts.MaxTokens
		}
		if opts.Temperature > 0 {
			geminiReq.GenerationConfig.Temperature = &opts.Temperature
		}
	}
	
	// Convert tools to Gemini format
	if len(tools) > 0 {
		geminiReq.Tools = make([]GeminiTool, 0, len(tools))
		for _, tool := range tools {
			geminiReq.Tools = append(geminiReq.Tools, GeminiTool{
				FunctionDeclarations: []GeminiFunctionDeclaration{
					{
						Name:        tool.Function.Name,
						Description: tool.Function.Description,
						Parameters:  tool.Function.Parameters,
					},
				},
			})
		}
	}
	
	// Convert messages
	for _, msg := range messages {
		role := msg.Role
		if role == "assistant" {
			role = "model"
		}
		
		geminiReq.Contents = append(geminiReq.Contents, GeminiContent{
			Role: role,
			Parts: []GeminiPart{
				{Text: msg.Content},
			},
		})
	}
	
	response, err := p.callGemini(ctx, geminiReq)
	if err != nil {
		return nil, err
	}
	
	if len(response.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates in gemini response")
	}
	
	candidate := response.Candidates[0]
	if len(candidate.Content.Parts) == 0 {
		return nil, fmt.Errorf("no parts in gemini candidate")
	}
	
	// Handle function calls
	var toolCalls []domain.ToolCall
	content := ""
	
	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			content = part.Text
		}
		if part.FunctionCall != nil {
			toolCalls = append(toolCalls, domain.ToolCall{
				ID:   fmt.Sprintf("call_%s", part.FunctionCall.Name),
				Type: "function",
				Function: domain.FunctionCall{
					Name:      part.FunctionCall.Name,
					Arguments: part.FunctionCall.Args,
				},
			})
		}
	}
	
	return &domain.GenerationResult{
		Content:   content,
		ToolCalls: toolCalls,
		Finished:  candidate.FinishReason == "STOP",
	}, nil
}

// StreamWithTools generates streaming text with tool calling support
func (p *GeminiProvider) StreamWithTools(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions, callback domain.ToolCallCallback) error {
	// For now, implement as non-streaming
	result, err := p.GenerateWithTools(ctx, messages, tools, opts)
	if err != nil {
		return err
	}
	
	return callback(result.Content, result.ToolCalls)
}

// GenerateStructured generates structured output
func (p *GeminiProvider) GenerateStructured(ctx context.Context, prompt string, schema interface{}, opts *domain.GenerationOptions) (*domain.StructuredResult, error) {
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
func (p *GeminiProvider) RecognizeIntent(ctx context.Context, request string) (*domain.IntentResult, error) {
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
func (p *GeminiProvider) ExtractMetadata(ctx context.Context, content string, model string) (*domain.ExtractedMetadata, error) {
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
			Summary:      content[:minIntGemini(200, len(content))],
			Keywords:     []string{},
			DocumentType: "unknown",
		}, nil
	}
	
	return &metadata, nil
}

// callGemini makes HTTP request to Gemini API
func (p *GeminiProvider) callGemini(ctx context.Context, req *GeminiRequest) (*GeminiResponse, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", 
		p.config.BaseURL, p.config.LLMModel, p.config.APIKey)
	
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	httpReq.Header.Set("Content-Type", "application/json")
	
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
		return nil, fmt.Errorf("gemini api error %d: %s", resp.StatusCode, string(body))
	}
	
	var geminiResp GeminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	
	return &geminiResp, nil
}

// Gemini API types
type GeminiRequest struct {
	Contents           []GeminiContent          `json:"contents"`
	Tools              []GeminiTool             `json:"tools,omitempty"`
	GenerationConfig   *GeminiGenerationConfig  `json:"generationConfig,omitempty"`
	SystemInstruction  *GeminiContent           `json:"systemInstruction,omitempty"`
}

type GeminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text         string                `json:"text,omitempty"`
	FunctionCall *GeminiFunctionCall   `json:"functionCall,omitempty"`
}

type GeminiFunctionCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

type GeminiTool struct {
	FunctionDeclarations []GeminiFunctionDeclaration `json:"functionDeclarations"`
}

type GeminiFunctionDeclaration struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

type GeminiGenerationConfig struct {
	Temperature      *float64 `json:"temperature,omitempty"`
	MaxOutputTokens  *int     `json:"maxOutputTokens,omitempty"`
	TopP             *float64 `json:"topP,omitempty"`
	TopK             *int     `json:"topK,omitempty"`
}

type GeminiResponse struct {
	Candidates    []GeminiCandidate `json:"candidates"`
	UsageMetadata GeminiUsage       `json:"usageMetadata"`
}

type GeminiCandidate struct {
	Content      GeminiContent `json:"content"`
	FinishReason string        `json:"finishReason"`
	Index        int           `json:"index"`
}

type GeminiUsage struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

// Helper function
func minIntGemini(a, b int) int {
	if a < b {
		return a
	}
	return b
}