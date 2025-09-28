package providers

import (
	"context"
	"fmt"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/usage"
)

// TrackedLLMProvider wraps an LLMProvider with usage tracking
type TrackedLLMProvider struct {
	domain.LLMProvider
	usageService *usage.Service
	providerName string
}

// NewTrackedLLMProvider creates a new tracked LLM provider
func NewTrackedLLMProvider(provider domain.LLMProvider, usageService *usage.Service) domain.LLMProvider {
	if usageService == nil {
		// Return the original provider if no usage service is provided
		return provider
	}
	
	return &TrackedLLMProvider{
		LLMProvider:  provider,
		usageService: usageService,
		providerName: string(provider.ProviderType()),
	}
}

// Generate generates text with usage tracking
func (t *TrackedLLMProvider) Generate(ctx context.Context, prompt string, opts *domain.GenerationOptions) (string, error) {
	startTime := time.Now()
	
	// Get current conversation and add user message
	if t.usageService != nil {
		_, _ = t.usageService.AddMessage(ctx, "user", prompt)
	}
	
	// Call the underlying provider
	result, err := t.LLMProvider.Generate(ctx, prompt, opts)
	
	// Track the usage
	if t.usageService != nil {
		model := ""
		// Use provider name as default model name since Model field doesn't exist
		if false { // opts.Model doesn't exist
			model = ""
		}
		
		if err != nil {
			// Track error
			_, _ = t.usageService.TrackError(ctx, usage.CallTypeLLM, t.providerName, model, err.Error(), startTime)
		} else {
			// Track successful call
			_, _ = t.usageService.TrackLLMCall(ctx, t.providerName, model, prompt, result, startTime)
			// Add assistant message
			_, _ = t.usageService.AddMessage(ctx, "assistant", result)
		}
	}
	
	return result, err
}

// Stream generates text with streaming and usage tracking
func (t *TrackedLLMProvider) Stream(ctx context.Context, prompt string, opts *domain.GenerationOptions, callback func(string)) error {
	startTime := time.Now()
	
	// Get current conversation and add user message
	if t.usageService != nil {
		_, _ = t.usageService.AddMessage(ctx, "user", prompt)
	}
	
	// Collect the streamed response
	var fullResponse string
	wrappedCallback := func(chunk string) {
		fullResponse += chunk
		callback(chunk)
	}
	
	// Call the underlying provider
	err := t.LLMProvider.Stream(ctx, prompt, opts, wrappedCallback)
	
	// Track the usage
	if t.usageService != nil {
		model := ""
		// Use provider name as default model name since Model field doesn't exist
		if false { // opts.Model doesn't exist
			model = ""
		}
		
		if err != nil {
			// Track error
			_, _ = t.usageService.TrackError(ctx, usage.CallTypeLLM, t.providerName, model, err.Error(), startTime)
		} else {
			// Track successful call
			_, _ = t.usageService.TrackLLMCall(ctx, t.providerName, model, prompt, fullResponse, startTime)
			// Add assistant message
			_, _ = t.usageService.AddMessage(ctx, "assistant", fullResponse)
		}
	}
	
	return err
}

// GenerateWithTools generates with tools and usage tracking
func (t *TrackedLLMProvider) GenerateWithTools(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions) (*domain.GenerationResult, error) {
	startTime := time.Now()
	
	// Track input messages
	if t.usageService != nil {
		for _, msg := range messages {
			_, _ = t.usageService.AddMessage(ctx, msg.Role, msg.Content)
		}
	}
	
	// Call the underlying provider
	result, err := t.LLMProvider.GenerateWithTools(ctx, messages, tools, opts)
	
	// Track the usage
	if t.usageService != nil {
		model := ""
		// Use provider name as default model name since Model field doesn't exist
		if false { // opts.Model doesn't exist
			model = ""
		}
		
		if err != nil {
			// Track error
			_, _ = t.usageService.TrackError(ctx, usage.CallTypeLLM, t.providerName, model, err.Error(), startTime)
		} else {
			// Build input string from messages
			inputStr := ""
			for _, msg := range messages {
				inputStr += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
			}
			
			// Track successful call
			_, _ = t.usageService.TrackLLMCall(ctx, t.providerName, model, inputStr, result.Content, startTime)
			// Add assistant message
			_, _ = t.usageService.AddMessage(ctx, "assistant", result.Content)
			
			// Track tool calls if any
			for _, toolCall := range result.ToolCalls {
				_, _ = t.usageService.TrackMCPCall(ctx, toolCall.Function.Name, toolCall.Function.Arguments, startTime)
			}
		}
	}
	
	return result, err
}

// StreamWithTools streams with tools and usage tracking
func (t *TrackedLLMProvider) StreamWithTools(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions, callback domain.ToolCallCallback) error {
	startTime := time.Now()
	
	// Track input messages
	if t.usageService != nil {
		for _, msg := range messages {
			_, _ = t.usageService.AddMessage(ctx, msg.Role, msg.Content)
		}
	}
	
	// Collect the streamed response
	var fullContent string
	var toolCalls []domain.ToolCall
	wrappedCallback := func(content string, tc []domain.ToolCall) error {
		if content != "" {
			fullContent += content
		}
		if len(tc) > 0 {
			toolCalls = tc
		}
		return callback(content, tc)
	}
	
	// Call the underlying provider
	err := t.LLMProvider.StreamWithTools(ctx, messages, tools, opts, wrappedCallback)
	
	// Track the usage
	if t.usageService != nil {
		model := ""
		// Use provider name as default model name since Model field doesn't exist
		if false { // opts.Model doesn't exist
			model = ""
		}
		
		if err != nil {
			// Track error
			_, _ = t.usageService.TrackError(ctx, usage.CallTypeLLM, t.providerName, model, err.Error(), startTime)
		} else {
			// Build input string from messages
			inputStr := ""
			for _, msg := range messages {
				inputStr += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
			}
			
			// Track successful call
			_, _ = t.usageService.TrackLLMCall(ctx, t.providerName, model, inputStr, fullContent, startTime)
			// Add assistant message
			if fullContent != "" {
				_, _ = t.usageService.AddMessage(ctx, "assistant", fullContent)
			}
			
			// Track tool calls if any
			for _, toolCall := range toolCalls {
				_, _ = t.usageService.TrackMCPCall(ctx, toolCall.Function.Name, toolCall.Function.Arguments, startTime)
			}
		}
	}
	
	return err
}

// GenerateStructured generates structured output with usage tracking
func (t *TrackedLLMProvider) GenerateStructured(ctx context.Context, prompt string, schema interface{}, opts *domain.GenerationOptions) (*domain.StructuredResult, error) {
	startTime := time.Now()
	
	// Get current conversation and add user message
	if t.usageService != nil {
		_, _ = t.usageService.AddMessage(ctx, "user", prompt)
	}
	
	// Call the underlying provider
	result, err := t.LLMProvider.GenerateStructured(ctx, prompt, schema, opts)
	
	// Track the usage
	if t.usageService != nil {
		model := ""
		// Use provider name as default model name since Model field doesn't exist
		if false { // opts.Model doesn't exist
			model = ""
		}
		
		if err != nil {
			// Track error
			_, _ = t.usageService.TrackError(ctx, usage.CallTypeLLM, t.providerName, model, err.Error(), startTime)
		} else {
			// Track successful call
			outputStr := fmt.Sprintf("%v", result.Data)
			_, _ = t.usageService.TrackLLMCall(ctx, t.providerName, model, prompt, outputStr, startTime)
			// Add assistant message
			_, _ = t.usageService.AddMessage(ctx, "assistant", outputStr)
		}
	}
	
	return result, err
}

// RecognizeIntent recognizes intent with usage tracking
func (t *TrackedLLMProvider) RecognizeIntent(ctx context.Context, request string) (*domain.IntentResult, error) {
	startTime := time.Now()
	
	// Call the underlying provider
	result, err := t.LLMProvider.RecognizeIntent(ctx, request)
	
	// Track the usage
	if t.usageService != nil {
		if err != nil {
			// Track error
			_, _ = t.usageService.TrackError(ctx, usage.CallTypeLLM, t.providerName, "intent", err.Error(), startTime)
		} else {
			// Track successful call
			outputStr := fmt.Sprintf("Intent: %s, Confidence: %.2f", result.Intent, result.Confidence)
			_, _ = t.usageService.TrackLLMCall(ctx, t.providerName, "intent", request, outputStr, startTime)
		}
	}
	
	return result, err
}

// TrackedEmbedderProvider wraps an EmbedderProvider with usage tracking
type TrackedEmbedderProvider struct {
	domain.EmbedderProvider
	usageService *usage.Service
	providerName string
}

// NewTrackedEmbedderProvider creates a new tracked embedder provider
func NewTrackedEmbedderProvider(provider domain.EmbedderProvider, usageService *usage.Service) domain.EmbedderProvider {
	if usageService == nil {
		// Return the original provider if no usage service is provided
		return provider
	}
	
	return &TrackedEmbedderProvider{
		EmbedderProvider: provider,
		usageService:     usageService,
		providerName:     string(provider.ProviderType()),
	}
}

// Embed generates embeddings with usage tracking
func (t *TrackedEmbedderProvider) Embed(ctx context.Context, text string) ([]float64, error) {
	startTime := time.Now()
	
	// Call the underlying provider
	result, err := t.EmbedderProvider.Embed(ctx, text)
	
	// Track the usage
	if t.usageService != nil {
		model := t.providerName // Use provider name as model
		if err != nil {
			// Track error
			_, _ = t.usageService.TrackError(ctx, usage.CallTypeLLM, t.providerName, model, err.Error(), startTime)
		} else {
			// Track successful call (embedding calls typically have lower token usage)
			_, _ = t.usageService.TrackLLMCall(ctx, t.providerName, model, text, "", startTime)
		}
	}
	
	return result, err
}

// Note: EmbedBatch is not part of the EmbedderProvider interface.
// If batch embedding is needed, users should call Embed multiple times.